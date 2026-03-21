package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/hylla/tillsyn/internal/app"
	"github.com/hylla/tillsyn/internal/domain"
)

// TestRepository_ProjectColumnTaskLifecycle verifies behavior for the covered scenario.
func TestRepository_ProjectColumnTaskLifecycle(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "tillsyn.db")
	repo, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(func() {
		_ = repo.Close()
	})

	now := time.Date(2026, 2, 21, 12, 0, 0, 0, time.UTC)
	project, err := domain.NewProject("p1", "Example", "desc", now)
	if err != nil {
		t.Fatalf("NewProject() error = %v", err)
	}
	if err := repo.CreateProject(ctx, project); err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}

	loadedProject, err := repo.GetProject(ctx, project.ID)
	if err != nil {
		t.Fatalf("GetProject() error = %v", err)
	}
	if loadedProject.Name != "Example" {
		t.Fatalf("unexpected project name %q", loadedProject.Name)
	}

	column, err := domain.NewColumn("c1", project.ID, "To Do", 0, 0, now)
	if err != nil {
		t.Fatalf("NewColumn() error = %v", err)
	}
	if err := repo.CreateColumn(ctx, column); err != nil {
		t.Fatalf("CreateColumn() error = %v", err)
	}

	due := now.Add(24 * time.Hour)
	task, err := domain.NewTask(domain.TaskInput{
		ID:          "t1",
		ProjectID:   project.ID,
		ColumnID:    column.ID,
		Position:    0,
		Title:       "Task title",
		Description: "Task details",
		Priority:    domain.PriorityHigh,
		DueAt:       &due,
		Labels:      []string{"a", "b"},
	}, now)
	if err != nil {
		t.Fatalf("NewTask() error = %v", err)
	}
	if err := repo.CreateTask(ctx, task); err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}

	tasks, err := repo.ListTasks(ctx, project.ID, false)
	if err != nil {
		t.Fatalf("ListTasks() error = %v", err)
	}
	if len(tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(tasks))
	}
	if len(tasks[0].Labels) != 2 {
		t.Fatalf("unexpected labels %#v", tasks[0].Labels)
	}

	task.Archive(now.Add(1 * time.Hour))
	if err := repo.UpdateTask(ctx, task); err != nil {
		t.Fatalf("UpdateTask() error = %v", err)
	}
	activeTasks, err := repo.ListTasks(ctx, project.ID, false)
	if err != nil {
		t.Fatalf("ListTasks(active) error = %v", err)
	}
	if len(activeTasks) != 0 {
		t.Fatalf("expected 0 active tasks, got %d", len(activeTasks))
	}

	allTasks, err := repo.ListTasks(ctx, project.ID, true)
	if err != nil {
		t.Fatalf("ListTasks(all) error = %v", err)
	}
	if len(allTasks) != 1 || allTasks[0].ArchivedAt == nil {
		t.Fatalf("expected archived task in full list, got %#v", allTasks)
	}

	if err := repo.DeleteTask(ctx, task.ID); err != nil {
		t.Fatalf("DeleteTask() error = %v", err)
	}
	if _, err := repo.GetTask(ctx, task.ID); err != app.ErrNotFound {
		t.Fatalf("expected app.ErrNotFound, got %v", err)
	}
}

// TestRepository_TaskEmbeddingsRoundTrip verifies embedding upsert/search/delete behavior.
func TestRepository_TaskEmbeddingsRoundTrip(t *testing.T) {
	ctx := context.Background()
	repo, err := OpenInMemory()
	if err != nil {
		t.Fatalf("OpenInMemory() error = %v", err)
	}
	t.Cleanup(func() {
		_ = repo.Close()
	})
	if !repo.vecAvailable {
		t.Skip("sqlite-vec capability unavailable in runtime")
	}

	now := time.Date(2026, 3, 3, 14, 0, 0, 0, time.UTC)
	project, err := domain.NewProject("p1", "Example", "", now)
	if err != nil {
		t.Fatalf("NewProject() error = %v", err)
	}
	if err := repo.CreateProject(ctx, project); err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}
	column, err := domain.NewColumn("c1", project.ID, "To Do", 0, 0, now)
	if err != nil {
		t.Fatalf("NewColumn() error = %v", err)
	}
	if err := repo.CreateColumn(ctx, column); err != nil {
		t.Fatalf("CreateColumn() error = %v", err)
	}
	task, err := domain.NewTask(domain.TaskInput{
		ID:        "t1",
		ProjectID: project.ID,
		ColumnID:  column.ID,
		Position:  0,
		Title:     "Task with embedding",
		Priority:  domain.PriorityMedium,
	}, now)
	if err != nil {
		t.Fatalf("NewTask() error = %v", err)
	}
	if err := repo.CreateTask(ctx, task); err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}

	if err := repo.UpsertTaskEmbedding(ctx, app.TaskEmbeddingDocument{
		TaskID:      task.ID,
		ProjectID:   project.ID,
		Content:     "task embedding content",
		ContentHash: "hash123",
		Vector:      []float32{0.1, 0.2, 0.3},
		UpdatedAt:   now,
	}); err != nil {
		t.Fatalf("UpsertTaskEmbedding() error = %v", err)
	}

	rows, err := repo.SearchTaskEmbeddings(ctx, app.TaskEmbeddingSearchInput{
		ProjectIDs: []string{project.ID},
		Vector:     []float32{0.1, 0.2, 0.3},
		Limit:      10,
	})
	if err != nil {
		t.Fatalf("SearchTaskEmbeddings() error = %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 embedding match, got %d", len(rows))
	}
	if rows[0].TaskID != task.ID {
		t.Fatalf("expected task id %q, got %q", task.ID, rows[0].TaskID)
	}

	if err := repo.DeleteTaskEmbedding(ctx, task.ID); err != nil {
		t.Fatalf("DeleteTaskEmbedding() error = %v", err)
	}
	rows, err = repo.SearchTaskEmbeddings(ctx, app.TaskEmbeddingSearchInput{
		ProjectIDs: []string{project.ID},
		Vector:     []float32{0.1, 0.2, 0.3},
		Limit:      10,
	})
	if err != nil {
		t.Fatalf("SearchTaskEmbeddings(after delete) error = %v", err)
	}
	if len(rows) != 0 {
		t.Fatalf("expected 0 embedding matches after delete, got %d", len(rows))
	}
}

// TestRepository_TaskEmbeddingMethodsReturnVecUnavailable verifies vector methods return a stable error when sqlite-vec is unavailable.
func TestRepository_TaskEmbeddingMethodsReturnVecUnavailable(t *testing.T) {
	ctx := context.Background()
	repo, err := OpenInMemory()
	if err != nil {
		t.Fatalf("OpenInMemory() error = %v", err)
	}
	t.Cleanup(func() {
		_ = repo.Close()
	})

	// Force the guard path so the test remains deterministic regardless of host runtime capabilities.
	repo.vecAvailable = false

	err = repo.UpsertTaskEmbedding(ctx, app.TaskEmbeddingDocument{
		TaskID:      "t1",
		ProjectID:   "p1",
		Content:     "task embedding content",
		ContentHash: "hash123",
		Vector:      []float32{0.1, 0.2, 0.3},
		UpdatedAt:   time.Date(2026, 3, 3, 14, 0, 0, 0, time.UTC),
	})
	if !errors.Is(err, errSQLiteVecUnavailable) {
		t.Fatalf("expected errSQLiteVecUnavailable, got %v", err)
	}

	_, err = repo.SearchTaskEmbeddings(ctx, app.TaskEmbeddingSearchInput{
		ProjectIDs: []string{"p1"},
		Vector:     []float32{0.1, 0.2, 0.3},
		Limit:      10,
	})
	if !errors.Is(err, errSQLiteVecUnavailable) {
		t.Fatalf("expected errSQLiteVecUnavailable, got %v", err)
	}
}

// TestRepository_CreateAndListCommentsByTarget verifies comment ordering and ownership persistence.
func TestRepository_CreateAndListCommentsByTarget(t *testing.T) {
	ctx := context.Background()
	repo, err := OpenInMemory()
	if err != nil {
		t.Fatalf("OpenInMemory() error = %v", err)
	}
	t.Cleanup(func() {
		_ = repo.Close()
	})

	now := time.Date(2026, 2, 23, 9, 0, 0, 0, time.UTC)
	project, err := domain.NewProject("p1", "Example", "", now)
	if err != nil {
		t.Fatalf("NewProject() error = %v", err)
	}
	if err := repo.CreateProject(ctx, project); err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}

	comment2, err := domain.NewComment(domain.CommentInput{
		ID:           "c2",
		ProjectID:    project.ID,
		TargetType:   domain.CommentTargetTypeTask,
		TargetID:     "t1",
		BodyMarkdown: "second",
		ActorID:      "agent-1",
		ActorName:    "Agent One",
		ActorType:    domain.ActorType("AGENT"),
	}, now)
	if err != nil {
		t.Fatalf("NewComment(c2) error = %v", err)
	}
	comment1, err := domain.NewComment(domain.CommentInput{
		ID:           "c1",
		ProjectID:    project.ID,
		TargetType:   domain.CommentTargetTypeTask,
		TargetID:     "t1",
		BodyMarkdown: "\n\nfirst line\nadditional details",
		ActorID:      "user-1",
		ActorName:    "User One",
		ActorType:    domain.ActorTypeUser,
	}, now)
	if err != nil {
		t.Fatalf("NewComment(c1) error = %v", err)
	}
	projectComment, err := domain.NewComment(domain.CommentInput{
		ID:           "c3",
		ProjectID:    project.ID,
		TargetType:   domain.CommentTargetTypeProject,
		TargetID:     project.ID,
		BodyMarkdown: "project note",
		ActorID:      "tillsyn",
		ActorName:    "Tillsyn",
		ActorType:    domain.ActorTypeSystem,
	}, now.Add(time.Minute))
	if err != nil {
		t.Fatalf("NewComment(c3) error = %v", err)
	}

	if err := repo.CreateComment(ctx, comment2); err != nil {
		t.Fatalf("CreateComment(c2) error = %v", err)
	}
	if err := repo.CreateComment(ctx, comment1); err != nil {
		t.Fatalf("CreateComment(c1) error = %v", err)
	}
	if err := repo.CreateComment(ctx, projectComment); err != nil {
		t.Fatalf("CreateComment(c3) error = %v", err)
	}
	var commentSummary string
	if err := repo.db.QueryRowContext(ctx, `SELECT summary FROM comments WHERE id = 'c1'`).Scan(&commentSummary); err != nil {
		t.Fatalf("query comment summary error = %v", err)
	}
	if commentSummary != "first line" {
		t.Fatalf("expected persisted comment summary %q, got %q", "first line", commentSummary)
	}

	taskComments, err := repo.ListCommentsByTarget(ctx, domain.CommentTarget{
		ProjectID:  project.ID,
		TargetType: domain.CommentTargetTypeTask,
		TargetID:   "t1",
	})
	if err != nil {
		t.Fatalf("ListCommentsByTarget(task) error = %v", err)
	}
	if len(taskComments) != 2 {
		t.Fatalf("expected 2 task comments, got %d", len(taskComments))
	}
	if taskComments[0].ID != "c1" || taskComments[1].ID != "c2" {
		t.Fatalf("expected deterministic created_at/id ordering, got %#v", taskComments)
	}
	if taskComments[1].ActorType != domain.ActorTypeAgent {
		t.Fatalf("expected normalized actor type agent, got %q", taskComments[1].ActorType)
	}
	if taskComments[1].ActorID != "agent-1" || taskComments[1].ActorName != "Agent One" {
		t.Fatalf("expected actor tuple to persist, got %#v", taskComments[1])
	}

	comments, err := repo.ListCommentsByTarget(ctx, domain.CommentTarget{
		ProjectID:  project.ID,
		TargetType: domain.CommentTargetTypeProject,
		TargetID:   project.ID,
	})
	if err != nil {
		t.Fatalf("ListCommentsByTarget(project) error = %v", err)
	}
	if len(comments) != 1 || comments[0].ID != "c3" {
		t.Fatalf("unexpected project comments %#v", comments)
	}
}

// TestRepository_ServiceCreateCommentPersistsContextActorName verifies comment persistence keeps the context display name.
func TestRepository_ServiceCreateCommentPersistsContextActorName(t *testing.T) {
	baseCtx := context.Background()
	repo, err := OpenInMemory()
	if err != nil {
		t.Fatalf("OpenInMemory() error = %v", err)
	}
	t.Cleanup(func() {
		_ = repo.Close()
	})

	now := time.Date(2026, 2, 26, 11, 30, 0, 0, time.UTC)
	ids := []string{"p1", "c1", "t1", "comment-1"}
	idIdx := 0
	svc := app.NewService(repo, func() string {
		id := ids[idIdx]
		idIdx++
		return id
	}, func() time.Time {
		return now
	}, app.ServiceConfig{})

	project, err := svc.CreateProjectWithMetadata(baseCtx, app.CreateProjectInput{
		Name: "Inbox",
	})
	if err != nil {
		t.Fatalf("CreateProjectWithMetadata() error = %v", err)
	}
	column, err := svc.CreateColumn(baseCtx, project.ID, "To Do", 0, 0)
	if err != nil {
		t.Fatalf("CreateColumn() error = %v", err)
	}
	task, err := svc.CreateTask(baseCtx, app.CreateTaskInput{
		ProjectID: project.ID,
		ColumnID:  column.ID,
		Title:     "Task",
		Priority:  domain.PriorityMedium,
	})
	if err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}

	ctx := app.WithMutationActor(baseCtx, app.MutationActor{
		ActorID:   "user-1",
		ActorName: "Evan Schultz",
		ActorType: domain.ActorTypeUser,
	})
	comment, err := svc.CreateComment(ctx, app.CreateCommentInput{
		ProjectID:    project.ID,
		TargetType:   domain.CommentTargetTypeTask,
		TargetID:     task.ID,
		BodyMarkdown: "hello",
		ActorID:      "user-1",
	})
	if err != nil {
		t.Fatalf("CreateComment() error = %v", err)
	}
	if comment.ActorName != "Evan Schultz" {
		t.Fatalf("comment actor name = %q, want Evan Schultz", comment.ActorName)
	}
	comments, err := repo.ListCommentsByTarget(baseCtx, domain.CommentTarget{
		ProjectID:  project.ID,
		TargetType: domain.CommentTargetTypeTask,
		TargetID:   task.ID,
	})
	if err != nil {
		t.Fatalf("ListCommentsByTarget() error = %v", err)
	}
	if len(comments) != 1 {
		t.Fatalf("expected 1 comment, got %d", len(comments))
	}
	if comments[0].ActorID != "user-1" || comments[0].ActorName != "Evan Schultz" {
		t.Fatalf("expected persisted comment attribution user-1/Evan Schultz, got %q/%q", comments[0].ActorID, comments[0].ActorName)
	}
}

// TestRepository_NotFoundCases verifies behavior for the covered scenario.
func TestRepository_NotFoundCases(t *testing.T) {
	repo, err := OpenInMemory()
	if err != nil {
		t.Fatalf("OpenInMemory() error = %v", err)
	}
	t.Cleanup(func() {
		_ = repo.Close()
	})

	ctx := context.Background()
	if _, err := repo.GetProject(ctx, "missing"); err != app.ErrNotFound {
		t.Fatalf("expected app.ErrNotFound for project, got %v", err)
	}
	if _, err := repo.GetTask(ctx, "missing"); err != app.ErrNotFound {
		t.Fatalf("expected app.ErrNotFound for task, got %v", err)
	}
	if err := repo.DeleteTask(ctx, "missing"); err != app.ErrNotFound {
		t.Fatalf("expected app.ErrNotFound for delete, got %v", err)
	}
	if err := repo.DeleteProject(ctx, "missing"); err != app.ErrNotFound {
		t.Fatalf("expected app.ErrNotFound for project delete, got %v", err)
	}
}

// TestRepository_ProjectAndColumnUpdates verifies behavior for the covered scenario.
func TestRepository_ProjectAndColumnUpdates(t *testing.T) {
	ctx := context.Background()
	repo, err := Open(filepath.Join(t.TempDir(), "tillsyn.db"))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(func() {
		_ = repo.Close()
	})

	now := time.Date(2026, 2, 21, 12, 0, 0, 0, time.UTC)
	project, _ := domain.NewProject("p1", "Alpha", "desc", now)
	project.Metadata = domain.ProjectMetadata{
		Owner: "owner-1",
		Tags:  []string{"tillsyn"},
	}
	if err := repo.CreateProject(ctx, project); err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}

	if err := project.Rename("Beta", now.Add(time.Minute)); err != nil {
		t.Fatalf("Rename() error = %v", err)
	}
	if err := repo.UpdateProject(ctx, project); err != nil {
		t.Fatalf("UpdateProject() error = %v", err)
	}

	activeProjects, err := repo.ListProjects(ctx, false)
	if err != nil {
		t.Fatalf("ListProjects(active) error = %v", err)
	}
	if len(activeProjects) != 1 || activeProjects[0].Name != "Beta" {
		t.Fatalf("unexpected active projects %#v", activeProjects)
	}
	if activeProjects[0].Metadata.Owner != "owner-1" || len(activeProjects[0].Metadata.Tags) != 1 {
		t.Fatalf("expected metadata persisted, got %#v", activeProjects[0].Metadata)
	}

	project.Archive(now.Add(2 * time.Minute))
	if err := repo.UpdateProject(ctx, project); err != nil {
		t.Fatalf("UpdateProject(archive) error = %v", err)
	}

	activeProjects, err = repo.ListProjects(ctx, false)
	if err != nil {
		t.Fatalf("ListProjects(active after archive) error = %v", err)
	}
	if len(activeProjects) != 0 {
		t.Fatalf("expected no active projects, got %#v", activeProjects)
	}

	allProjects, err := repo.ListProjects(ctx, true)
	if err != nil {
		t.Fatalf("ListProjects(all) error = %v", err)
	}
	if len(allProjects) != 1 || allProjects[0].ArchivedAt == nil {
		t.Fatalf("expected archived project in all list, got %#v", allProjects)
	}

	column, _ := domain.NewColumn("c1", project.ID, "To Do", 0, 1, now)
	if err := repo.CreateColumn(ctx, column); err != nil {
		t.Fatalf("CreateColumn() error = %v", err)
	}
	if err := column.Rename("Doing", now.Add(3*time.Minute)); err != nil {
		t.Fatalf("Rename() error = %v", err)
	}
	if err := column.SetPosition(2, now.Add(4*time.Minute)); err != nil {
		t.Fatalf("SetPosition() error = %v", err)
	}
	if err := repo.UpdateColumn(ctx, column); err != nil {
		t.Fatalf("UpdateColumn() error = %v", err)
	}

	columns, err := repo.ListColumns(ctx, project.ID, false)
	if err != nil {
		t.Fatalf("ListColumns() error = %v", err)
	}
	if len(columns) != 1 || columns[0].Name != "Doing" {
		t.Fatalf("unexpected columns %#v", columns)
	}

	column.Archive(now.Add(5 * time.Minute))
	if err := repo.UpdateColumn(ctx, column); err != nil {
		t.Fatalf("UpdateColumn(archive) error = %v", err)
	}
	activeCols, err := repo.ListColumns(ctx, project.ID, false)
	if err != nil {
		t.Fatalf("ListColumns(active) error = %v", err)
	}
	if len(activeCols) != 0 {
		t.Fatalf("expected no active columns, got %#v", activeCols)
	}
	allCols, err := repo.ListColumns(ctx, project.ID, true)
	if err != nil {
		t.Fatalf("ListColumns(all) error = %v", err)
	}
	if len(allCols) != 1 || allCols[0].ArchivedAt == nil {
		t.Fatalf("expected archived column in all list, got %#v", allCols)
	}
}

// TestRepository_DeleteProjectCascades verifies project hard-delete cascades to child rows.
func TestRepository_DeleteProjectCascades(t *testing.T) {
	ctx := context.Background()
	repo, err := Open(filepath.Join(t.TempDir(), "cascade.db"))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(func() {
		_ = repo.Close()
	})

	now := time.Date(2026, 2, 21, 12, 0, 0, 0, time.UTC)
	project, _ := domain.NewProject("p1", "Alpha", "", now)
	if err := repo.CreateProject(ctx, project); err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}
	column, _ := domain.NewColumn("c1", project.ID, "To Do", 0, 0, now)
	if err := repo.CreateColumn(ctx, column); err != nil {
		t.Fatalf("CreateColumn() error = %v", err)
	}
	task, _ := domain.NewTask(domain.TaskInput{
		ID:        "t1",
		ProjectID: project.ID,
		ColumnID:  column.ID,
		Position:  0,
		Title:     "Task",
		Priority:  domain.PriorityMedium,
	}, now)
	if err := repo.CreateTask(ctx, task); err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}

	if err := repo.DeleteProject(ctx, project.ID); err != nil {
		t.Fatalf("DeleteProject() error = %v", err)
	}
	if _, err := repo.GetProject(ctx, project.ID); err != app.ErrNotFound {
		t.Fatalf("expected app.ErrNotFound for project, got %v", err)
	}
	if _, err := repo.GetTask(ctx, task.ID); err != app.ErrNotFound {
		t.Fatalf("expected app.ErrNotFound for task cascade, got %v", err)
	}
}

// TestRepository_MigratesLegacyProjectsTable verifies behavior for the covered scenario.
func TestRepository_MigratesLegacyProjectsTable(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "legacy.db")
	db, err := sql.Open(driverName, dbPath)
	if err != nil {
		t.Fatalf("sql.Open() error = %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})
	_, err = db.ExecContext(ctx, `
		CREATE TABLE projects (
			id TEXT PRIMARY KEY,
			slug TEXT NOT NULL,
			name TEXT NOT NULL,
			description TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			archived_at TEXT
		)
	`)
	if err != nil {
		t.Fatalf("create legacy table error = %v", err)
	}

	repo, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open() on legacy db error = %v", err)
	}
	t.Cleanup(func() {
		_ = repo.Close()
	})

	project, _ := domain.NewProject("p1", "Legacy", "", time.Date(2026, 2, 21, 12, 0, 0, 0, time.UTC))
	project.Metadata = domain.ProjectMetadata{Owner: "evan"}
	if err := repo.CreateProject(ctx, project); err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}
	loaded, err := repo.GetProject(ctx, project.ID)
	if err != nil {
		t.Fatalf("GetProject() error = %v", err)
	}
	if loaded.Metadata.Owner != "evan" {
		t.Fatalf("expected metadata owner to persist after migration, got %#v", loaded.Metadata)
	}
}

// TestRepository_MigratesLegacyTasksTable verifies behavior for the covered scenario.
func TestRepository_MigratesLegacyTasksTable(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "legacy-tasks.db")
	db, err := sql.Open(driverName, dbPath)
	if err != nil {
		t.Fatalf("sql.Open() error = %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	legacySchema := []string{
		`CREATE TABLE projects (
			id TEXT PRIMARY KEY,
			slug TEXT NOT NULL,
			name TEXT NOT NULL,
			description TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			archived_at TEXT
		)`,
		`CREATE TABLE columns_v1 (
			id TEXT PRIMARY KEY,
			project_id TEXT NOT NULL,
			name TEXT NOT NULL,
			wip_limit INTEGER NOT NULL DEFAULT 0,
			position INTEGER NOT NULL,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			archived_at TEXT
		)`,
		`CREATE TABLE tasks (
			id TEXT PRIMARY KEY,
			project_id TEXT NOT NULL,
			column_id TEXT NOT NULL,
			position INTEGER NOT NULL,
			title TEXT NOT NULL,
			description TEXT NOT NULL DEFAULT '',
			priority TEXT NOT NULL,
			due_at TEXT,
			labels_json TEXT NOT NULL DEFAULT '[]',
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			archived_at TEXT
		)`,
	}
	for _, stmt := range legacySchema {
		if _, err := db.ExecContext(ctx, stmt); err != nil {
			t.Fatalf("create legacy schema error = %v", err)
		}
	}
	now := time.Date(2026, 2, 21, 12, 0, 0, 0, time.UTC)
	for _, stmt := range []string{
		`INSERT INTO projects(id, slug, name, description, created_at, updated_at, archived_at)
		 VALUES ('p1', 'legacy', 'Legacy', '', '` + now.Format(time.RFC3339Nano) + `', '` + now.Format(time.RFC3339Nano) + `', NULL)`,
		`INSERT INTO columns_v1(id, project_id, name, wip_limit, position, created_at, updated_at, archived_at)
		 VALUES ('c1', 'p1', 'To Do', 0, 0, '` + now.Format(time.RFC3339Nano) + `', '` + now.Format(time.RFC3339Nano) + `', NULL)`,
		`INSERT INTO tasks(id, project_id, column_id, position, title, description, priority, due_at, labels_json, created_at, updated_at, archived_at)
		 VALUES ('t1', 'p1', 'c1', 0, 'Legacy task', 'desc', 'medium', NULL, '["legacy"]', '` + now.Format(time.RFC3339Nano) + `', '` + now.Format(time.RFC3339Nano) + `', NULL)`,
	} {
		if _, err := db.ExecContext(ctx, stmt); err != nil {
			t.Fatalf("seed legacy rows error = %v", err)
		}
	}

	repo, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open() on legacy task db error = %v", err)
	}
	t.Cleanup(func() {
		_ = repo.Close()
	})

	rows, err := repo.db.QueryContext(ctx, `PRAGMA table_info(tasks)`)
	if err != nil {
		t.Fatalf("PRAGMA table_info(tasks) error = %v", err)
	}
	t.Cleanup(func() {
		_ = rows.Close()
	})

	seenParentID := false
	for rows.Next() {
		var (
			cid        int
			name       string
			colType    string
			notNull    int
			defaultVal sql.NullString
			primaryKey int
		)
		if err := rows.Scan(&cid, &name, &colType, &notNull, &defaultVal, &primaryKey); err != nil {
			t.Fatalf("rows.Scan() error = %v", err)
		}
		if name == "parent_id" {
			seenParentID = true
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows.Err() = %v", err)
	}
	if !seenParentID {
		t.Fatal("expected parent_id column to be added during migration")
	}

	var workItemCount int
	if err := repo.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM work_items WHERE id = 't1'`).Scan(&workItemCount); err != nil {
		t.Fatalf("count work_items error = %v", err)
	}
	if workItemCount != 1 {
		t.Fatalf("expected migrated work_items row count 1, got %d", workItemCount)
	}
	loaded, err := repo.GetTask(ctx, "t1")
	if err != nil {
		t.Fatalf("GetTask() migrated row error = %v", err)
	}
	if loaded.Title != "Legacy task" || loaded.ProjectID != "p1" {
		t.Fatalf("unexpected migrated task %#v", loaded)
	}
	if loaded.Kind != domain.WorkKindTask || loaded.LifecycleState != domain.StateTodo {
		t.Fatalf("expected default kind/state migration values, got kind=%q state=%q", loaded.Kind, loaded.LifecycleState)
	}

	var tableCount int
	if err := repo.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='change_events'`).Scan(&tableCount); err != nil {
		t.Fatalf("count change_events table error = %v", err)
	}
	if tableCount != 1 {
		t.Fatalf("expected change_events table to exist after migration, got %d", tableCount)
	}
	if err := repo.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='comments'`).Scan(&tableCount); err != nil {
		t.Fatalf("count comments table error = %v", err)
	}
	if tableCount != 1 {
		t.Fatalf("expected comments table to exist after migration, got %d", tableCount)
	}
	commentColumns := map[string]struct{}{}
	commentRows, err := repo.db.QueryContext(ctx, `PRAGMA table_info(comments)`)
	if err != nil {
		t.Fatalf("PRAGMA table_info(comments) error = %v", err)
	}
	for commentRows.Next() {
		var (
			cid        int
			name       string
			colType    string
			notNull    int
			defaultVal sql.NullString
			primaryKey int
		)
		if err := commentRows.Scan(&cid, &name, &colType, &notNull, &defaultVal, &primaryKey); err != nil {
			_ = commentRows.Close()
			t.Fatalf("scan comments table_info error = %v", err)
		}
		commentColumns[name] = struct{}{}
	}
	if err := commentRows.Close(); err != nil {
		t.Fatalf("close comments table_info rows error = %v", err)
	}
	if _, ok := commentColumns["actor_id"]; !ok {
		t.Fatalf("expected comments.actor_id in migrated schema, got %#v", commentColumns)
	}
	if _, ok := commentColumns["actor_name"]; !ok {
		t.Fatalf("expected comments.actor_name in migrated schema, got %#v", commentColumns)
	}
	if _, ok := commentColumns["summary"]; !ok {
		t.Fatalf("expected comments.summary in migrated schema, got %#v", commentColumns)
	}
	if _, ok := commentColumns["author_name"]; ok {
		t.Fatalf("expected comments.author_name to be removed from canonical schema, got %#v", commentColumns)
	}
	changeEventColumns := map[string]struct{}{}
	changeRows, err := repo.db.QueryContext(ctx, `PRAGMA table_info(change_events)`)
	if err != nil {
		t.Fatalf("PRAGMA table_info(change_events) error = %v", err)
	}
	for changeRows.Next() {
		var (
			cid        int
			name       string
			colType    string
			notNull    int
			defaultVal sql.NullString
			primaryKey int
		)
		if err := changeRows.Scan(&cid, &name, &colType, &notNull, &defaultVal, &primaryKey); err != nil {
			_ = changeRows.Close()
			t.Fatalf("scan change_events table_info error = %v", err)
		}
		changeEventColumns[name] = struct{}{}
	}
	if err := changeRows.Close(); err != nil {
		t.Fatalf("close change_events table_info rows error = %v", err)
	}
	if _, ok := changeEventColumns["actor_name"]; !ok {
		t.Fatalf("expected change_events.actor_name in migrated schema, got %#v", changeEventColumns)
	}
	if err := repo.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='attention_items'`).Scan(&tableCount); err != nil {
		t.Fatalf("count attention_items table error = %v", err)
	}
	if tableCount != 1 {
		t.Fatalf("expected attention_items table to exist after migration, got %d", tableCount)
	}

	var indexCount int
	if err := repo.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM sqlite_master WHERE type='index' AND name='idx_comments_project_target_created_at'`).Scan(&indexCount); err != nil {
		t.Fatalf("count comments index error = %v", err)
	}
	if indexCount != 1 {
		t.Fatalf("expected comments target index to exist after migration, got %d", indexCount)
	}
	if err := repo.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM sqlite_master WHERE type='index' AND name='idx_attention_scope_state_created_at'`).Scan(&indexCount); err != nil {
		t.Fatalf("count attention scope index error = %v", err)
	}
	if indexCount != 1 {
		t.Fatalf("expected attention scope index to exist after migration, got %d", indexCount)
	}
}

// TestRepository_MigratesLegacyCommentAndEventOwnership verifies ownership tuple backfill from legacy schemas.
func TestRepository_MigratesLegacyCommentAndEventOwnership(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "legacy-ownership.db")
	db, err := sql.Open(driverName, dbPath)
	if err != nil {
		t.Fatalf("sql.Open() error = %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	now := time.Date(2026, 2, 21, 12, 0, 0, 0, time.UTC).Format(time.RFC3339Nano)
	legacySchema := []string{
		`CREATE TABLE projects (
			id TEXT PRIMARY KEY,
			slug TEXT NOT NULL,
			name TEXT NOT NULL,
			description TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			archived_at TEXT
		)`,
		`CREATE TABLE comments (
			id TEXT PRIMARY KEY,
			project_id TEXT NOT NULL,
			target_type TEXT NOT NULL,
			target_id TEXT NOT NULL,
			body_markdown TEXT NOT NULL,
			actor_type TEXT NOT NULL DEFAULT 'user',
			author_name TEXT NOT NULL DEFAULT 'tillsyn-user',
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)`,
		`CREATE TABLE change_events (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			project_id TEXT NOT NULL,
			work_item_id TEXT NOT NULL,
			operation TEXT NOT NULL,
			actor_id TEXT NOT NULL,
			actor_type TEXT NOT NULL,
			metadata_json TEXT NOT NULL DEFAULT '{}',
			created_at TEXT NOT NULL
		)`,
	}
	for _, stmt := range legacySchema {
		if _, err := db.ExecContext(ctx, stmt); err != nil {
			t.Fatalf("create legacy schema error = %v", err)
		}
	}
	for _, stmt := range []string{
		`INSERT INTO projects(id, slug, name, description, created_at, updated_at, archived_at) VALUES ('p1', 'legacy', 'Legacy', '', '` + now + `', '` + now + `', NULL)`,
		`INSERT INTO comments(id, project_id, target_type, target_id, body_markdown, actor_type, author_name, created_at, updated_at) VALUES ('c1', 'p1', 'project', 'p1', 'legacy comment', 'user', 'legacy-author', '` + now + `', '` + now + `')`,
		`INSERT INTO comments(id, project_id, target_type, target_id, body_markdown, actor_type, author_name, created_at, updated_at) VALUES ('c2', 'p1', 'project', 'p1', 'fallback comment', 'user', '', '` + now + `', '` + now + `')`,
		`INSERT INTO change_events(project_id, work_item_id, operation, actor_id, actor_type, metadata_json, created_at) VALUES ('p1', 't1', 'update', 'legacy-actor', 'user', '{}', '` + now + `')`,
	} {
		if _, err := db.ExecContext(ctx, stmt); err != nil {
			t.Fatalf("seed legacy rows error = %v", err)
		}
	}
	if _, err := db.ExecContext(
		ctx,
		`INSERT INTO comments(id, project_id, target_type, target_id, body_markdown, actor_type, author_name, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"c3",
		"p1",
		"project",
		"p1",
		"\n\nlegacy headline\nlegacy body",
		"user",
		"legacy-author-3",
		now,
		now,
	); err != nil {
		t.Fatalf("seed multiline legacy comment error = %v", err)
	}

	repo, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open() on legacy ownership db error = %v", err)
	}
	t.Cleanup(func() {
		_ = repo.Close()
	})

	var actorID, actorName, summary string
	if err := repo.db.QueryRowContext(ctx, `SELECT actor_id, actor_name, summary FROM comments WHERE id = 'c1'`).Scan(&actorID, &actorName, &summary); err != nil {
		t.Fatalf("query migrated c1 actor tuple error = %v", err)
	}
	if actorID != "legacy-author" || actorName != "legacy-author" {
		t.Fatalf("expected migrated actor tuple legacy-author/legacy-author, got %q/%q", actorID, actorName)
	}
	if summary != "legacy comment" {
		t.Fatalf("expected migrated summary for c1 %q, got %q", "legacy comment", summary)
	}
	if err := repo.db.QueryRowContext(ctx, `SELECT actor_id, actor_name, summary FROM comments WHERE id = 'c2'`).Scan(&actorID, &actorName, &summary); err != nil {
		t.Fatalf("query migrated c2 actor tuple error = %v", err)
	}
	if actorID != "tillsyn-user" || actorName != "tillsyn-user" {
		t.Fatalf("expected fallback actor tuple tillsyn-user/tillsyn-user, got %q/%q", actorID, actorName)
	}
	if summary != "fallback comment" {
		t.Fatalf("expected migrated summary for c2 %q, got %q", "fallback comment", summary)
	}
	if err := repo.db.QueryRowContext(ctx, `SELECT actor_id, actor_name, summary FROM comments WHERE id = 'c3'`).Scan(&actorID, &actorName, &summary); err != nil {
		t.Fatalf("query migrated c3 actor tuple error = %v", err)
	}
	if actorID != "legacy-author-3" || actorName != "legacy-author-3" {
		t.Fatalf("expected migrated actor tuple legacy-author-3/legacy-author-3, got %q/%q", actorID, actorName)
	}
	if summary != "legacy headline" {
		t.Fatalf("expected migrated summary for c3 %q, got %q", "legacy headline", summary)
	}

	var eventActorName string
	if err := repo.db.QueryRowContext(ctx, `SELECT actor_name FROM change_events WHERE work_item_id = 't1'`).Scan(&eventActorName); err != nil {
		t.Fatalf("query migrated change_event actor_name error = %v", err)
	}
	if eventActorName != "legacy-actor" {
		t.Fatalf("expected migrated change_event actor_name legacy-actor, got %q", eventActorName)
	}
}

// TestRepository_MigratesLegacyCommentSummaryIdempotent verifies summary migration for canonical legacy comments.
func TestRepository_MigratesLegacyCommentSummaryIdempotent(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "legacy-summary.db")
	db, err := sql.Open(driverName, dbPath)
	if err != nil {
		t.Fatalf("sql.Open() error = %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	now := time.Date(2026, 2, 24, 8, 30, 0, 0, time.UTC).Format(time.RFC3339Nano)
	legacySchema := []string{
		`CREATE TABLE projects (
			id TEXT PRIMARY KEY,
			slug TEXT NOT NULL,
			name TEXT NOT NULL,
			description TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			archived_at TEXT
		)`,
		`CREATE TABLE comments (
			id TEXT PRIMARY KEY,
			project_id TEXT NOT NULL,
			target_type TEXT NOT NULL,
			target_id TEXT NOT NULL,
			body_markdown TEXT NOT NULL,
			actor_id TEXT NOT NULL DEFAULT 'tillsyn-user',
			actor_name TEXT NOT NULL DEFAULT 'tillsyn-user',
			actor_type TEXT NOT NULL DEFAULT 'user',
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)`,
	}
	for _, stmt := range legacySchema {
		if _, err := db.ExecContext(ctx, stmt); err != nil {
			t.Fatalf("create legacy schema error = %v", err)
		}
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO projects(id, slug, name, description, created_at, updated_at, archived_at) VALUES (?, ?, ?, ?, ?, ?, NULL)`, "p1", "legacy", "Legacy", "", now, now); err != nil {
		t.Fatalf("seed legacy project error = %v", err)
	}
	if _, err := db.ExecContext(
		ctx,
		`INSERT INTO comments(id, project_id, target_type, target_id, body_markdown, actor_id, actor_name, actor_type, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"c1",
		"p1",
		"project",
		"p1",
		"\n\nsummary headline\ndetail",
		"legacy-user",
		"Legacy User",
		"user",
		now,
		now,
	); err != nil {
		t.Fatalf("seed multiline legacy comment error = %v", err)
	}
	if _, err := db.ExecContext(
		ctx,
		`INSERT INTO comments(id, project_id, target_type, target_id, body_markdown, actor_id, actor_name, actor_type, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"c2",
		"p1",
		"project",
		"p1",
		"single line summary",
		"legacy-user-2",
		"Legacy User Two",
		"user",
		now,
		now,
	); err != nil {
		t.Fatalf("seed single-line legacy comment error = %v", err)
	}

	repo, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open() on legacy summary db error = %v", err)
	}
	var summary string
	if err := repo.db.QueryRowContext(ctx, `SELECT summary FROM comments WHERE id = 'c1'`).Scan(&summary); err != nil {
		_ = repo.Close()
		t.Fatalf("query migrated summary c1 error = %v", err)
	}
	if summary != "summary headline" {
		_ = repo.Close()
		t.Fatalf("expected c1 summary %q, got %q", "summary headline", summary)
	}
	if err := repo.db.QueryRowContext(ctx, `SELECT summary FROM comments WHERE id = 'c2'`).Scan(&summary); err != nil {
		_ = repo.Close()
		t.Fatalf("query migrated summary c2 error = %v", err)
	}
	if summary != "single line summary" {
		_ = repo.Close()
		t.Fatalf("expected c2 summary %q, got %q", "single line summary", summary)
	}
	if err := repo.Close(); err != nil {
		t.Fatalf("first close after migration error = %v", err)
	}

	repo, err = Open(dbPath)
	if err != nil {
		t.Fatalf("re-open() on migrated summary db error = %v", err)
	}
	t.Cleanup(func() {
		_ = repo.Close()
	})
	if err := repo.db.QueryRowContext(ctx, `SELECT summary FROM comments WHERE id = 'c1'`).Scan(&summary); err != nil {
		t.Fatalf("query re-opened summary c1 error = %v", err)
	}
	if summary != "summary headline" {
		t.Fatalf("expected stable c1 summary %q after re-open, got %q", "summary headline", summary)
	}
}

// TestRepositoryOpenValidation verifies behavior for the covered scenario.
func TestRepositoryOpenValidation(t *testing.T) {
	if _, err := Open("   "); err == nil {
		t.Fatal("expected error for empty sqlite path")
	}
}

// TestRepositoryUpdateNotFound verifies behavior for the covered scenario.
func TestRepositoryUpdateNotFound(t *testing.T) {
	repo, err := OpenInMemory()
	if err != nil {
		t.Fatalf("OpenInMemory() error = %v", err)
	}
	t.Cleanup(func() {
		_ = repo.Close()
	})

	now := time.Now().UTC()
	p, _ := domain.NewProject("missing", "nope", "", now)
	if err := repo.UpdateProject(context.Background(), p); err != app.ErrNotFound {
		t.Fatalf("expected app.ErrNotFound for UpdateProject, got %v", err)
	}

	c, _ := domain.NewColumn("missing-col", "missing", "todo", 0, 0, now)
	if err := repo.UpdateColumn(context.Background(), c); err != app.ErrNotFound {
		t.Fatalf("expected app.ErrNotFound for UpdateColumn, got %v", err)
	}

	tk, _ := domain.NewTask(domain.TaskInput{
		ID:        "missing-task",
		ProjectID: "missing",
		ColumnID:  "missing-col",
		Position:  0,
		Title:     "x",
		Priority:  domain.PriorityLow,
	}, now)
	if err := repo.UpdateTask(context.Background(), tk); err != app.ErrNotFound {
		t.Fatalf("expected app.ErrNotFound for UpdateTask, got %v", err)
	}
}

// TestRepository_ListProjectChangeEventsLifecycle verifies behavior for the covered scenario.
func TestRepository_ListProjectChangeEventsLifecycle(t *testing.T) {
	ctx := context.Background()
	repo, err := Open(filepath.Join(t.TempDir(), "events.db"))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(func() {
		_ = repo.Close()
	})

	now := time.Date(2026, 2, 21, 12, 0, 0, 0, time.UTC)
	project, _ := domain.NewProject("p1", "Events", "", now)
	if err := repo.CreateProject(ctx, project); err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}
	todo, _ := domain.NewColumn("c1", project.ID, "To Do", 0, 0, now)
	done, _ := domain.NewColumn("c2", project.ID, "Done", 1, 0, now)
	if err := repo.CreateColumn(ctx, todo); err != nil {
		t.Fatalf("CreateColumn(todo) error = %v", err)
	}
	if err := repo.CreateColumn(ctx, done); err != nil {
		t.Fatalf("CreateColumn(done) error = %v", err)
	}

	task, _ := domain.NewTask(domain.TaskInput{
		ID:             "t1",
		ProjectID:      project.ID,
		ColumnID:       todo.ID,
		Position:       0,
		Title:          "Track me",
		Priority:       domain.PriorityMedium,
		CreatedByActor: "user-1",
		CreatedByName:  "Evan Schultz",
		UpdatedByActor: "user-1",
		UpdatedByName:  "Evan Schultz",
		UpdatedByType:  domain.ActorTypeUser,
	}, now)
	if err := repo.CreateTask(ctx, task); err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}

	if err := task.UpdateDetails("Track me v2", task.Description, task.Priority, task.DueAt, task.Labels, now.Add(time.Minute)); err != nil {
		t.Fatalf("UpdateDetails() error = %v", err)
	}
	task.UpdatedByActor = "agent-1"
	task.UpdatedByName = "Planner Bot"
	task.UpdatedByType = domain.ActorTypeAgent
	if err := repo.UpdateTask(ctx, task); err != nil {
		t.Fatalf("UpdateTask(update) error = %v", err)
	}

	if err := task.Move(done.ID, 1, now.Add(2*time.Minute)); err != nil {
		t.Fatalf("Move() error = %v", err)
	}
	task.UpdatedByActor = "user-2"
	task.UpdatedByName = "Evan Schultz"
	task.UpdatedByType = domain.ActorTypeUser
	if err := repo.UpdateTask(ctx, task); err != nil {
		t.Fatalf("UpdateTask(move) error = %v", err)
	}

	task.Archive(now.Add(3 * time.Minute))
	task.UpdatedByActor = "user-3"
	task.UpdatedByName = "Evan Schultz"
	task.UpdatedByType = domain.ActorTypeUser
	if err := repo.UpdateTask(ctx, task); err != nil {
		t.Fatalf("UpdateTask(archive) error = %v", err)
	}

	task.Restore(now.Add(4 * time.Minute))
	task.UpdatedByActor = "user-4"
	task.UpdatedByName = "Evan Schultz"
	task.UpdatedByType = domain.ActorTypeUser
	if err := repo.UpdateTask(ctx, task); err != nil {
		t.Fatalf("UpdateTask(restore) error = %v", err)
	}

	if err := repo.DeleteTask(ctx, task.ID); err != nil {
		t.Fatalf("DeleteTask() error = %v", err)
	}

	events, err := repo.ListProjectChangeEvents(ctx, project.ID, 10)
	if err != nil {
		t.Fatalf("ListProjectChangeEvents() error = %v", err)
	}
	if len(events) != 6 {
		t.Fatalf("expected 6 events, got %d (%#v)", len(events), events)
	}

	wantOps := []domain.ChangeOperation{
		domain.ChangeOperationDelete,
		domain.ChangeOperationRestore,
		domain.ChangeOperationArchive,
		domain.ChangeOperationMove,
		domain.ChangeOperationUpdate,
		domain.ChangeOperationCreate,
	}
	for i, want := range wantOps {
		if events[i].Operation != want {
			t.Fatalf("unexpected event operation at index %d: got %q want %q", i, events[i].Operation, want)
		}
	}

	if events[3].Metadata["from_column_id"] != todo.ID || events[3].Metadata["to_column_id"] != done.ID {
		t.Fatalf("expected move metadata to include column transition, got %#v", events[3].Metadata)
	}
	if events[5].ActorID != "user-1" {
		t.Fatalf("expected create actor user-1, got %q", events[5].ActorID)
	}
	if events[5].ActorName != "Evan Schultz" {
		t.Fatalf("expected create actor_name Evan Schultz, got %q", events[5].ActorName)
	}
	if events[4].ActorName != "Planner Bot" {
		t.Fatalf("expected update actor_name Planner Bot, got %q", events[4].ActorName)
	}
}

// TestRepository_TaskLifecyclePreservesMutationActorName verifies task change events keep request actor_name attribution.
func TestRepository_TaskLifecyclePreservesMutationActorName(t *testing.T) {
	cases := []struct {
		name  string
		actor app.MutationActor
	}{
		{
			name: "user",
			actor: app.MutationActor{
				ActorID:   "user-1",
				ActorName: "Evan Schultz",
				ActorType: domain.ActorTypeUser,
			},
		},
		{
			name: "agent",
			actor: app.MutationActor{
				ActorID:   "agent-1",
				ActorName: "Planner Bot",
				ActorType: domain.ActorTypeAgent,
			},
		},
		{
			name: "system",
			actor: app.MutationActor{
				ActorID:   "system-1",
				ActorName: "Background Sync",
				ActorType: domain.ActorTypeSystem,
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			baseCtx := context.Background()
			ctx := app.WithMutationActor(baseCtx, tc.actor)
			repo, err := OpenInMemory()
			if err != nil {
				t.Fatalf("OpenInMemory() error = %v", err)
			}
			t.Cleanup(func() {
				_ = repo.Close()
			})

			now := time.Date(2026, 2, 25, 10, 0, 0, 0, time.UTC)
			project, _ := domain.NewProject("p1", "Inbox", "", now)
			if err := repo.CreateProject(baseCtx, project); err != nil {
				t.Fatalf("CreateProject() error = %v", err)
			}
			todo, _ := domain.NewColumn("c1", project.ID, "To Do", 0, 0, now)
			if err := repo.CreateColumn(baseCtx, todo); err != nil {
				t.Fatalf("CreateColumn() error = %v", err)
			}
			task, _ := domain.NewTask(domain.TaskInput{
				ID:        "t1",
				ProjectID: project.ID,
				ColumnID:  todo.ID,
				Position:  0,
				Title:     "Ownership",
				Priority:  domain.PriorityLow,
			}, now)
			if err := repo.CreateTask(ctx, task); err != nil {
				t.Fatalf("CreateTask() error = %v", err)
			}

			if err := task.UpdateDetails("Ownership v2", task.Description, task.Priority, task.DueAt, task.Labels, now.Add(time.Minute)); err != nil {
				t.Fatalf("UpdateDetails() error = %v", err)
			}
			if err := repo.UpdateTask(ctx, task); err != nil {
				t.Fatalf("UpdateTask() error = %v", err)
			}
			if err := repo.DeleteTask(ctx, task.ID); err != nil {
				t.Fatalf("DeleteTask() error = %v", err)
			}

			events, err := repo.ListProjectChangeEvents(baseCtx, project.ID, 3)
			if err != nil {
				t.Fatalf("ListProjectChangeEvents() error = %v", err)
			}
			if len(events) != 3 {
				t.Fatalf("expected 3 events, got %d (%#v)", len(events), events)
			}
			for _, event := range events {
				if event.ActorID != tc.actor.ActorID {
					t.Fatalf("expected actor_id %q, got %q", tc.actor.ActorID, event.ActorID)
				}
				if event.ActorName != tc.actor.ActorName {
					t.Fatalf("expected actor_name %q, got %q", tc.actor.ActorName, event.ActorName)
				}
				if event.ActorType != tc.actor.ActorType {
					t.Fatalf("expected actor_type %q, got %q", tc.actor.ActorType, event.ActorType)
				}
			}
		})
	}
}

// TestRepository_ServiceCreateTaskPersistsHumanActorName verifies service-provided display names reach persisted change events.
func TestRepository_ServiceCreateTaskPersistsHumanActorName(t *testing.T) {
	ctx := context.Background()
	repo, err := OpenInMemory()
	if err != nil {
		t.Fatalf("OpenInMemory() error = %v", err)
	}
	t.Cleanup(func() {
		_ = repo.Close()
	})

	now := time.Date(2026, 2, 26, 11, 0, 0, 0, time.UTC)
	ids := []string{"p1", "c1", "t1"}
	idIdx := 0
	svc := app.NewService(repo, func() string {
		id := ids[idIdx]
		idIdx++
		return id
	}, func() time.Time {
		return now
	}, app.ServiceConfig{})

	project, err := svc.CreateProjectWithMetadata(ctx, app.CreateProjectInput{
		Name:          "Inbox",
		Description:   "",
		UpdatedBy:     "user-1",
		UpdatedByName: "Evan Schultz",
		UpdatedType:   domain.ActorTypeUser,
	})
	if err != nil {
		t.Fatalf("CreateProjectWithMetadata() error = %v", err)
	}
	column, err := svc.CreateColumn(ctx, project.ID, "To Do", 0, 0)
	if err != nil {
		t.Fatalf("CreateColumn() error = %v", err)
	}
	created, err := svc.CreateTask(ctx, app.CreateTaskInput{
		ProjectID:      project.ID,
		ColumnID:       column.ID,
		Title:          "Ownership",
		Priority:       domain.PriorityMedium,
		CreatedByActor: "user-1",
		CreatedByName:  "Evan Schultz",
		UpdatedByType:  domain.ActorTypeUser,
	})
	if err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}
	events, err := repo.ListProjectChangeEvents(ctx, project.ID, 1)
	if err != nil {
		t.Fatalf("ListProjectChangeEvents() error = %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d (%#v)", len(events), events)
	}
	if events[0].WorkItemID != created.ID {
		t.Fatalf("expected event work item id %q, got %q", created.ID, events[0].WorkItemID)
	}
	if events[0].ActorID != "user-1" || events[0].ActorName != "Evan Schultz" {
		t.Fatalf("expected human attribution user-1/Evan Schultz, got %q/%q", events[0].ActorID, events[0].ActorName)
	}
	loaded, err := repo.GetTask(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetTask() error = %v", err)
	}
	if loaded.CreatedByName != "Evan Schultz" || loaded.UpdatedByName != "Evan Schultz" {
		t.Fatalf("expected persisted task names Evan Schultz/Evan Schultz, got %q/%q", loaded.CreatedByName, loaded.UpdatedByName)
	}
}

// TestRepository_KindCatalogAndAllowlistRoundTrip verifies kind catalog persistence and project allowlist wiring.
func TestRepository_KindCatalogAndAllowlistRoundTrip(t *testing.T) {
	ctx := context.Background()
	repo, err := OpenInMemory()
	if err != nil {
		t.Fatalf("OpenInMemory() error = %v", err)
	}
	t.Cleanup(func() {
		_ = repo.Close()
	})

	now := time.Date(2026, 2, 24, 10, 0, 0, 0, time.UTC)
	project, _ := domain.NewProject("p-kind", "Kinds", "", now)
	if err := repo.CreateProject(ctx, project); err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}

	kind, err := domain.NewKindDefinition(domain.KindDefinitionInput{
		ID:                "refactor",
		DisplayName:       "Refactor",
		AppliesTo:         []domain.KindAppliesTo{domain.KindAppliesToTask},
		PayloadSchemaJSON: `{"type":"object","required":["package"],"properties":{"package":{"type":"string"}}}`,
	}, now)
	if err != nil {
		t.Fatalf("NewKindDefinition() error = %v", err)
	}
	if err := repo.CreateKindDefinition(ctx, kind); err != nil {
		t.Fatalf("CreateKindDefinition() error = %v", err)
	}
	loadedKind, err := repo.GetKindDefinition(ctx, kind.ID)
	if err != nil {
		t.Fatalf("GetKindDefinition() error = %v", err)
	}
	if loadedKind.DisplayName != "Refactor" {
		t.Fatalf("unexpected kind display name %q", loadedKind.DisplayName)
	}

	if err := repo.SetProjectAllowedKinds(ctx, project.ID, []domain.KindID{kind.ID, domain.DefaultProjectKind}); err != nil {
		t.Fatalf("SetProjectAllowedKinds() error = %v", err)
	}
	allowed, err := repo.ListProjectAllowedKinds(ctx, project.ID)
	if err != nil {
		t.Fatalf("ListProjectAllowedKinds() error = %v", err)
	}
	if len(allowed) != 2 {
		t.Fatalf("expected 2 allowed kinds, got %#v", allowed)
	}
}

// TestRepository_CapabilityLeaseRoundTrip verifies lease persistence and scope revoke behavior.
func TestRepository_CapabilityLeaseRoundTrip(t *testing.T) {
	ctx := context.Background()
	repo, err := OpenInMemory()
	if err != nil {
		t.Fatalf("OpenInMemory() error = %v", err)
	}
	t.Cleanup(func() {
		_ = repo.Close()
	})

	now := time.Date(2026, 2, 24, 10, 0, 0, 0, time.UTC)
	project, _ := domain.NewProject("p-lease", "Leases", "", now)
	if err := repo.CreateProject(ctx, project); err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}

	lease, err := domain.NewCapabilityLease(domain.CapabilityLeaseInput{
		InstanceID: "inst-1",
		LeaseToken: "tok-1",
		AgentName:  "orch",
		ProjectID:  project.ID,
		ScopeType:  domain.CapabilityScopeProject,
		Role:       domain.CapabilityRoleOrchestrator,
		ExpiresAt:  now.Add(time.Hour),
	}, now)
	if err != nil {
		t.Fatalf("NewCapabilityLease() error = %v", err)
	}
	if err := repo.CreateCapabilityLease(ctx, lease); err != nil {
		t.Fatalf("CreateCapabilityLease() error = %v", err)
	}
	loaded, err := repo.GetCapabilityLease(ctx, lease.InstanceID)
	if err != nil {
		t.Fatalf("GetCapabilityLease() error = %v", err)
	}
	if loaded.AgentName != "orch" {
		t.Fatalf("unexpected lease agent name %q", loaded.AgentName)
	}

	listed, err := repo.ListCapabilityLeasesByScope(ctx, project.ID, domain.CapabilityScopeProject, "")
	if err != nil {
		t.Fatalf("ListCapabilityLeasesByScope() error = %v", err)
	}
	if len(listed) != 1 {
		t.Fatalf("expected one listed lease, got %#v", listed)
	}

	if err := repo.RevokeCapabilityLeasesByScope(ctx, project.ID, domain.CapabilityScopeProject, "", now.Add(2*time.Minute), "manual"); err != nil {
		t.Fatalf("RevokeCapabilityLeasesByScope() error = %v", err)
	}
	revoked, err := repo.GetCapabilityLease(ctx, lease.InstanceID)
	if err != nil {
		t.Fatalf("GetCapabilityLease(revoked) error = %v", err)
	}
	if revoked.RevokedAt == nil {
		t.Fatal("expected revoked_at to be set")
	}
}

// TestRepository_AttentionItemRoundTrip verifies scoped attention persistence, ordering, and resolution.
func TestRepository_AttentionItemRoundTrip(t *testing.T) {
	ctx := context.Background()
	repo, err := OpenInMemory()
	if err != nil {
		t.Fatalf("OpenInMemory() error = %v", err)
	}
	t.Cleanup(func() {
		_ = repo.Close()
	})

	now := time.Date(2026, 2, 24, 10, 0, 0, 0, time.UTC)
	project, _ := domain.NewProject("p-attn", "Attention", "", now)
	if err := repo.CreateProject(ctx, project); err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}

	risk, err := domain.NewAttentionItem(domain.AttentionItemInput{
		ID:                 "attn-risk",
		ProjectID:          project.ID,
		ScopeType:          domain.ScopeLevelTask,
		ScopeID:            "t1",
		Kind:               domain.AttentionKindRiskNote,
		Summary:            "Track rollout risk",
		RequiresUserAction: false,
		CreatedByActor:     "user-1",
		CreatedByType:      domain.ActorTypeUser,
	}, now)
	if err != nil {
		t.Fatalf("NewAttentionItem(risk) error = %v", err)
	}
	blocker, err := domain.NewAttentionItem(domain.AttentionItemInput{
		ID:                 "attn-blocker",
		ProjectID:          project.ID,
		ScopeType:          domain.ScopeLevelTask,
		ScopeID:            "t1",
		Kind:               domain.AttentionKindBlocker,
		Summary:            "Need approval to proceed",
		RequiresUserAction: true,
		CreatedByActor:     "agent-1",
		CreatedByType:      domain.ActorTypeAgent,
	}, now.Add(time.Minute))
	if err != nil {
		t.Fatalf("NewAttentionItem(blocker) error = %v", err)
	}

	if err := repo.CreateAttentionItem(ctx, risk); err != nil {
		t.Fatalf("CreateAttentionItem(risk) error = %v", err)
	}
	if err := repo.CreateAttentionItem(ctx, blocker); err != nil {
		t.Fatalf("CreateAttentionItem(blocker) error = %v", err)
	}

	unresolved, err := repo.ListAttentionItems(ctx, domain.AttentionListFilter{
		ProjectID:      project.ID,
		ScopeType:      domain.ScopeLevelTask,
		ScopeID:        "t1",
		UnresolvedOnly: true,
	})
	if err != nil {
		t.Fatalf("ListAttentionItems(unresolved) error = %v", err)
	}
	if len(unresolved) != 2 {
		t.Fatalf("expected 2 unresolved items, got %#v", unresolved)
	}
	if unresolved[0].ID != blocker.ID || unresolved[1].ID != risk.ID {
		t.Fatalf("expected newest-first deterministic order, got %#v", unresolved)
	}

	requiresUserAction := true
	userActionOnly, err := repo.ListAttentionItems(ctx, domain.AttentionListFilter{
		ProjectID:          project.ID,
		ScopeType:          domain.ScopeLevelTask,
		ScopeID:            "t1",
		UnresolvedOnly:     true,
		RequiresUserAction: &requiresUserAction,
	})
	if err != nil {
		t.Fatalf("ListAttentionItems(requires_user_action) error = %v", err)
	}
	if len(userActionOnly) != 1 || userActionOnly[0].ID != blocker.ID {
		t.Fatalf("expected one requires_user_action item, got %#v", userActionOnly)
	}

	resolved, err := repo.ResolveAttentionItem(ctx, blocker.ID, "user-2", domain.ActorTypeUser, now.Add(2*time.Minute))
	if err != nil {
		t.Fatalf("ResolveAttentionItem() error = %v", err)
	}
	if resolved.State != domain.AttentionStateResolved || resolved.ResolvedAt == nil {
		t.Fatalf("expected resolved attention row, got %#v", resolved)
	}

	unresolved, err = repo.ListAttentionItems(ctx, domain.AttentionListFilter{
		ProjectID:      project.ID,
		ScopeType:      domain.ScopeLevelTask,
		ScopeID:        "t1",
		UnresolvedOnly: true,
	})
	if err != nil {
		t.Fatalf("ListAttentionItems(unresolved after resolve) error = %v", err)
	}
	if len(unresolved) != 1 || unresolved[0].ID != risk.ID {
		t.Fatalf("expected only unresolved risk item, got %#v", unresolved)
	}
}

// TestRepository_AttentionItemValidationErrors verifies fail-closed validation paths for attention writes/queries.
func TestRepository_AttentionItemValidationErrors(t *testing.T) {
	ctx := context.Background()
	repo, err := OpenInMemory()
	if err != nil {
		t.Fatalf("OpenInMemory() error = %v", err)
	}
	t.Cleanup(func() {
		_ = repo.Close()
	})

	now := time.Date(2026, 2, 24, 10, 0, 0, 0, time.UTC)
	project, _ := domain.NewProject("p-attn-validate", "Attention Validate", "", now)
	if err := repo.CreateProject(ctx, project); err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}

	base := domain.AttentionItem{
		ID:                 "attn-valid",
		ProjectID:          project.ID,
		ScopeType:          domain.ScopeLevelTask,
		ScopeID:            "task-1",
		State:              domain.AttentionStateOpen,
		Kind:               domain.AttentionKindRiskNote,
		Summary:            "summary",
		RequiresUserAction: false,
		CreatedByActor:     "user-1",
		CreatedByType:      domain.ActorTypeUser,
		CreatedAt:          now,
	}

	t.Run("create invalid id", func(t *testing.T) {
		item := base
		item.ID = "   "
		if err := repo.CreateAttentionItem(ctx, item); !errors.Is(err, domain.ErrInvalidID) {
			t.Fatalf("CreateAttentionItem() error = %v, want %v", err, domain.ErrInvalidID)
		}
	})

	t.Run("create invalid scope", func(t *testing.T) {
		item := base
		item.ID = "attn-invalid-scope"
		item.ScopeType = domain.ScopeLevel("bad-scope")
		if err := repo.CreateAttentionItem(ctx, item); !errors.Is(err, domain.ErrInvalidScopeType) {
			t.Fatalf("CreateAttentionItem() error = %v, want %v", err, domain.ErrInvalidScopeType)
		}
	})

	t.Run("create invalid kind", func(t *testing.T) {
		item := base
		item.ID = "attn-invalid-kind"
		item.Kind = domain.AttentionKind("bad-kind")
		if err := repo.CreateAttentionItem(ctx, item); !errors.Is(err, domain.ErrInvalidAttentionKind) {
			t.Fatalf("CreateAttentionItem() error = %v, want %v", err, domain.ErrInvalidAttentionKind)
		}
	})

	t.Run("create empty summary", func(t *testing.T) {
		item := base
		item.ID = "attn-empty-summary"
		item.Summary = "   "
		if err := repo.CreateAttentionItem(ctx, item); !errors.Is(err, domain.ErrInvalidSummary) {
			t.Fatalf("CreateAttentionItem() error = %v, want %v", err, domain.ErrInvalidSummary)
		}
	})

	t.Run("list filter validation errors", func(t *testing.T) {
		cases := []struct {
			name   string
			filter domain.AttentionListFilter
			want   error
		}{
			{
				name: "missing project id",
				filter: domain.AttentionListFilter{
					ScopeType: domain.ScopeLevelTask,
					ScopeID:   "task-1",
				},
				want: domain.ErrInvalidID,
			},
			{
				name: "scope id without scope type",
				filter: domain.AttentionListFilter{
					ProjectID: project.ID,
					ScopeID:   "task-1",
				},
				want: domain.ErrInvalidScopeType,
			},
			{
				name: "invalid state",
				filter: domain.AttentionListFilter{
					ProjectID: project.ID,
					ScopeType: domain.ScopeLevelTask,
					ScopeID:   "task-1",
					States:    []domain.AttentionState{"bad-state"},
				},
				want: domain.ErrInvalidAttentionState,
			},
			{
				name: "invalid kind",
				filter: domain.AttentionListFilter{
					ProjectID: project.ID,
					ScopeType: domain.ScopeLevelTask,
					ScopeID:   "task-1",
					Kinds:     []domain.AttentionKind{"bad-kind"},
				},
				want: domain.ErrInvalidAttentionKind,
			},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				_, err := repo.ListAttentionItems(ctx, tc.filter)
				if !errors.Is(err, tc.want) {
					t.Fatalf("ListAttentionItems() error = %v, want %v", err, tc.want)
				}
			})
		}
	})

	t.Run("resolve invalid id", func(t *testing.T) {
		_, err := repo.ResolveAttentionItem(ctx, "   ", "user-2", domain.ActorTypeUser, now.Add(time.Minute))
		if !errors.Is(err, domain.ErrInvalidID) {
			t.Fatalf("ResolveAttentionItem() error = %v, want %v", err, domain.ErrInvalidID)
		}
	})

	t.Run("resolve missing item", func(t *testing.T) {
		_, err := repo.ResolveAttentionItem(ctx, "missing", "user-2", domain.ActorTypeUser, now.Add(time.Minute))
		if !errors.Is(err, app.ErrNotFound) {
			t.Fatalf("ResolveAttentionItem() error = %v, want %v", err, app.ErrNotFound)
		}
	})
}

// TestRepository_SeedDefaultKindsIncludeNestedPhaseSupport verifies seeded defaults include nested phase support.
func TestRepository_SeedDefaultKindsIncludeNestedPhaseSupport(t *testing.T) {
	ctx := context.Background()
	repo, err := OpenInMemory()
	if err != nil {
		t.Fatalf("OpenInMemory() error = %v", err)
	}
	t.Cleanup(func() {
		_ = repo.Close()
	})

	phase, err := repo.GetKindDefinition(ctx, domain.KindID(domain.WorkKindPhase))
	if err != nil {
		t.Fatalf("GetKindDefinition(phase) error = %v", err)
	}
	if !phase.AppliesToScope(domain.KindAppliesToPhase) {
		t.Fatalf("expected phase kind to apply to phase, got %#v", phase.AppliesTo)
	}
	if !phase.AllowsParentScope(domain.KindAppliesToPhase) {
		t.Fatalf("expected phase kind parent scopes to include phase, got %#v", phase.AllowedParentScopes)
	}
}

// TestRepository_PersistsProjectKindAndTaskScope verifies new kind/scope columns round-trip.
func TestRepository_PersistsProjectKindAndTaskScope(t *testing.T) {
	ctx := context.Background()
	repo, err := OpenInMemory()
	if err != nil {
		t.Fatalf("OpenInMemory() error = %v", err)
	}
	t.Cleanup(func() {
		_ = repo.Close()
	})

	now := time.Date(2026, 2, 24, 10, 0, 0, 0, time.UTC)
	project, _ := domain.NewProject("p-scope", "Scope", "", now)
	if err := project.SetKind("project-template", now); err != nil {
		t.Fatalf("SetKind() error = %v", err)
	}
	if err := repo.CreateProject(ctx, project); err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}
	loadedProject, err := repo.GetProject(ctx, project.ID)
	if err != nil {
		t.Fatalf("GetProject() error = %v", err)
	}
	if loadedProject.Kind != domain.KindID("project-template") {
		t.Fatalf("expected persisted project kind, got %q", loadedProject.Kind)
	}

	column, _ := domain.NewColumn("c-scope", project.ID, "To Do", 0, 0, now)
	if err := repo.CreateColumn(ctx, column); err != nil {
		t.Fatalf("CreateColumn() error = %v", err)
	}
	task, err := domain.NewTask(domain.TaskInput{
		ID:        "t-scope",
		ProjectID: project.ID,
		ColumnID:  column.ID,
		Scope:     domain.KindAppliesToPhase,
		Kind:      domain.WorkKindPhase,
		Position:  0,
		Title:     "phase",
		Priority:  domain.PriorityMedium,
	}, now)
	if err != nil {
		t.Fatalf("NewTask() error = %v", err)
	}
	if err := repo.CreateTask(ctx, task); err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}
	loadedTask, err := repo.GetTask(ctx, task.ID)
	if err != nil {
		t.Fatalf("GetTask() error = %v", err)
	}
	if loadedTask.Scope != domain.KindAppliesToPhase {
		t.Fatalf("expected persisted task scope phase, got %q", loadedTask.Scope)
	}

	nestedPhaseTask, err := domain.NewTask(domain.TaskInput{
		ID:        "t-nested-phase",
		ProjectID: project.ID,
		ParentID:  task.ID,
		ColumnID:  column.ID,
		Scope:     domain.KindAppliesToPhase,
		Kind:      domain.WorkKindPhase,
		Position:  1,
		Title:     "nested phase",
		Priority:  domain.PriorityMedium,
	}, now)
	if err != nil {
		t.Fatalf("NewTask(nested phase) error = %v", err)
	}
	if err := repo.CreateTask(ctx, nestedPhaseTask); err != nil {
		t.Fatalf("CreateTask(nested phase) error = %v", err)
	}
	loadedNestedPhaseTask, err := repo.GetTask(ctx, nestedPhaseTask.ID)
	if err != nil {
		t.Fatalf("GetTask(nested phase) error = %v", err)
	}
	if loadedNestedPhaseTask.Scope != domain.KindAppliesToPhase {
		t.Fatalf("expected persisted task scope phase, got %q", loadedNestedPhaseTask.Scope)
	}
}

// TestRepositoryAuthRequestCRUD verifies auth-request persistence, listing, and update behavior.
func TestRepositoryAuthRequestCRUD(t *testing.T) {
	ctx := context.Background()
	repo, err := OpenInMemory()
	if err != nil {
		t.Fatalf("OpenInMemory() error = %v", err)
	}
	t.Cleanup(func() {
		_ = repo.Close()
	})

	now := time.Date(2026, 3, 20, 12, 0, 0, 0, time.UTC)
	project, err := domain.NewProject("p1", "Project One", "", now)
	if err != nil {
		t.Fatalf("NewProject() error = %v", err)
	}
	if err := repo.CreateProject(ctx, project); err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}
	var projectCount int
	if err := repo.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM projects WHERE id = ?`, project.ID).Scan(&projectCount); err != nil {
		t.Fatalf("project count query error = %v", err)
	}
	if projectCount != 1 {
		t.Fatalf("project count = %d, want 1", projectCount)
	}
	request := domain.AuthRequest{
		ID:                  "req-1",
		ProjectID:           project.ID,
		BranchID:            "b1",
		PhaseIDs:            []string{"ph1", "ph2"},
		Path:                "project/p1/branch/b1/phase/ph1/phase/ph2",
		ScopeType:           domain.ScopeLevelPhase,
		ScopeID:             "ph2",
		PrincipalID:         "agent-1",
		PrincipalType:       "agent",
		PrincipalName:       "Agent One",
		ClientID:            "till-mcp-stdio",
		ClientType:          "mcp-stdio",
		ClientName:          "Till MCP STDIO",
		RequestedSessionTTL: 2 * time.Hour,
		Reason:              "needs review",
		Continuation:        map[string]any{"resume_tool": "till.raise_attention_item", "resume": map[string]any{"path": "project/p1"}},
		State:               domain.AuthRequestStatePending,
		RequestedByActor:    "lane-user",
		RequestedByType:     domain.ActorTypeUser,
		CreatedAt:           now,
		ExpiresAt:           now.Add(30 * time.Minute),
	}
	if err := repo.CreateAuthRequest(ctx, request); err != nil {
		t.Fatalf("CreateAuthRequest() error = %v", err)
	}

	got, err := repo.GetAuthRequest(ctx, request.ID)
	if err != nil {
		t.Fatalf("GetAuthRequest() error = %v", err)
	}
	if got.Path != request.Path || got.ScopeType != request.ScopeType || got.ScopeID != request.ScopeID {
		t.Fatalf("GetAuthRequest() = %#v, want persisted request %#v", got, request)
	}
	if gotValue, _ := got.Continuation["resume_tool"].(string); gotValue != "till.raise_attention_item" {
		t.Fatalf("GetAuthRequest() continuation = %#v, want resume_tool", got.Continuation)
	}

	listed, err := repo.ListAuthRequests(ctx, domain.AuthRequestListFilter{ProjectID: project.ID, State: domain.AuthRequestStatePending, Limit: 10})
	if err != nil {
		t.Fatalf("ListAuthRequests() error = %v", err)
	}
	if len(listed) != 1 || listed[0].ID != request.ID {
		t.Fatalf("ListAuthRequests() = %#v, want request %q", listed, request.ID)
	}

	request.State = domain.AuthRequestStateApproved
	request.ResolutionNote = "approved"
	request.ResolvedByActor = "approver"
	request.ResolvedByType = domain.ActorTypeUser
	request.ResolvedAt = &now
	request.IssuedSessionID = "sess-1"
	request.IssuedSessionSecret = "secret-1"
	exp := now.Add(2 * time.Hour)
	request.IssuedSessionExpiresAt = &exp
	if err := repo.UpdateAuthRequest(ctx, request); err != nil {
		t.Fatalf("UpdateAuthRequest() error = %v", err)
	}
	approved, err := repo.GetAuthRequest(ctx, request.ID)
	if err != nil {
		t.Fatalf("GetAuthRequest(after update) error = %v", err)
	}
	if approved.State != domain.AuthRequestStateApproved || approved.IssuedSessionID != "sess-1" {
		t.Fatalf("GetAuthRequest(after update) = %#v, want approved session", approved)
	}
	approvedList, err := repo.ListAuthRequests(ctx, domain.AuthRequestListFilter{ProjectID: project.ID, State: domain.AuthRequestStateApproved, Limit: 10})
	if err != nil {
		t.Fatalf("ListAuthRequests(approved) error = %v", err)
	}
	if len(approvedList) != 1 || approvedList[0].ID != request.ID {
		t.Fatalf("ListAuthRequests(approved) = %#v, want approved request", approvedList)
	}
}

// TestRepositoryAuthRequestScanErrors verifies malformed persisted JSON surfaces scan errors.
func TestRepositoryAuthRequestScanErrors(t *testing.T) {
	ctx := context.Background()
	repo, err := OpenInMemory()
	if err != nil {
		t.Fatalf("OpenInMemory() error = %v", err)
	}
	t.Cleanup(func() {
		_ = repo.Close()
	})

	now := time.Date(2026, 3, 20, 12, 5, 0, 0, time.UTC)
	project, err := domain.NewProject("p1", "Project One", "", now)
	if err != nil {
		t.Fatalf("NewProject() error = %v", err)
	}
	if err := repo.CreateProject(ctx, project); err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}
	var projectCount int
	if err := repo.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM projects WHERE id = ?`, project.ID).Scan(&projectCount); err != nil {
		t.Fatalf("project count query error = %v", err)
	}
	if projectCount != 1 {
		t.Fatalf("project count = %d, want 1", projectCount)
	}
	request := domain.AuthRequest{
		ID:                  "req-bad-json",
		ProjectID:           project.ID,
		Path:                "project/p1",
		ScopeType:           domain.ScopeLevelProject,
		ScopeID:             "p1",
		PrincipalID:         "agent-1",
		PrincipalType:       "agent",
		ClientID:            "client-1",
		ClientType:          "mcp-stdio",
		RequestedSessionTTL: time.Hour,
		State:               domain.AuthRequestStatePending,
		RequestedByType:     domain.ActorTypeUser,
		CreatedAt:           now,
		ExpiresAt:           now.Add(time.Hour),
	}
	if err := repo.CreateAuthRequest(ctx, request); err != nil {
		t.Fatalf("CreateAuthRequest() error = %v", err)
	}
	if _, err := repo.db.ExecContext(ctx, `UPDATE auth_requests SET continuation_json = '{bad json' WHERE id = ?`, request.ID); err != nil {
		t.Fatalf("update malformed continuation_json error = %v", err)
	}
	if _, err := repo.GetAuthRequest(ctx, request.ID); err == nil {
		t.Fatal("GetAuthRequest() error = nil, want scan decode error")
	}
	if _, err := repo.db.ExecContext(ctx, `UPDATE auth_requests SET continuation_json = '{}', phase_ids_json = '[bad json' WHERE id = ?`, request.ID); err != nil {
		t.Fatalf("update malformed phase_ids_json error = %v", err)
	}
	if _, err := repo.ListAuthRequests(ctx, domain.AuthRequestListFilter{ProjectID: project.ID}); err == nil {
		t.Fatal("ListAuthRequests() error = nil, want scan decode error")
	}
}
