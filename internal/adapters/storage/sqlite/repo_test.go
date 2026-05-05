package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/evanmschultz/tillsyn/internal/app"
	"github.com/evanmschultz/tillsyn/internal/domain"
)

// TestApplySQLiteConnectionPragmas configures the live connection without relying on URI-encoded pragmas.
func TestApplySQLiteConnectionPragmas(t *testing.T) {
	db, err := sql.Open(driverName, "file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("sql.Open() error = %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	if err := applySQLiteConnectionPragmas(context.Background(), db); err != nil {
		t.Fatalf("applySQLiteConnectionPragmas() error = %v", err)
	}
	var timeout int
	if err := db.QueryRowContext(context.Background(), `PRAGMA busy_timeout`).Scan(&timeout); err != nil {
		t.Fatalf("query busy_timeout error = %v", err)
	}
	if timeout != int(defaultBusyTimeout/time.Millisecond) {
		t.Fatalf("busy_timeout = %d, want %d", timeout, defaultBusyTimeout/time.Millisecond)
	}
	var foreignKeys int
	if err := db.QueryRowContext(context.Background(), `PRAGMA foreign_keys`).Scan(&foreignKeys); err != nil {
		t.Fatalf("query foreign_keys error = %v", err)
	}
	if foreignKeys != 1 {
		t.Fatalf("foreign_keys = %d, want 1", foreignKeys)
	}
}

// TestOpenAppliesSQLiteConnectionPragmasToFileBackedDB verifies the file-backed open path initializes the expected local dogfood PRAGMAs.
func TestOpenAppliesSQLiteConnectionPragmasToFileBackedDB(t *testing.T) {
	ctx := context.Background()
	repo, err := Open(filepath.Join(t.TempDir(), "tillsyn.db"))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(func() {
		_ = repo.Close()
	})

	var timeout int
	if err := repo.DB().QueryRowContext(ctx, `PRAGMA busy_timeout`).Scan(&timeout); err != nil {
		t.Fatalf("query busy_timeout error = %v", err)
	}
	if timeout != int(defaultBusyTimeout/time.Millisecond) {
		t.Fatalf("busy_timeout = %d, want %d", timeout, defaultBusyTimeout/time.Millisecond)
	}
	var journalMode string
	if err := repo.DB().QueryRowContext(ctx, `PRAGMA journal_mode`).Scan(&journalMode); err != nil {
		t.Fatalf("query journal_mode error = %v", err)
	}
	if journalMode != "wal" {
		t.Fatalf("journal_mode = %q, want wal", journalMode)
	}
	var foreignKeys int
	if err := repo.DB().QueryRowContext(ctx, `PRAGMA foreign_keys`).Scan(&foreignKeys); err != nil {
		t.Fatalf("query foreign_keys error = %v", err)
	}
	if foreignKeys != 1 {
		t.Fatalf("foreign_keys = %d, want 1", foreignKeys)
	}
}

// TestRepository_ProjectColumnActionItemLifecycle verifies behavior for the covered scenario.
func TestRepository_ProjectColumnActionItemLifecycle(t *testing.T) {
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
	project, err := domain.NewProjectFromInput(domain.ProjectInput{ID: "p1", Name: "Example", Description: "desc"}, now)
	if err != nil {
		t.Fatalf("NewProjectFromInput() error = %v", err)
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
	actionItem, err := domain.NewActionItemForTest(domain.ActionItemInput{
		Kind:        domain.KindPlan,
		ID:          "t1",
		ProjectID:   project.ID,
		ColumnID:    column.ID,
		Position:    0,
		Title:       "ActionItem title",
		Description: "ActionItem details",
		Priority:    domain.PriorityHigh,
		DueAt:       &due,
		Labels:      []string{"a", "b"},
	}, now)
	if err != nil {
		t.Fatalf("NewActionItem() error = %v", err)
	}
	if err := repo.CreateActionItem(ctx, actionItem); err != nil {
		t.Fatalf("CreateActionItem() error = %v", err)
	}

	tasks, err := repo.ListActionItems(ctx, project.ID, false)
	if err != nil {
		t.Fatalf("ListActionItems() error = %v", err)
	}
	if len(tasks) != 1 {
		t.Fatalf("expected 1 actionItem, got %d", len(tasks))
	}
	if len(tasks[0].Labels) != 2 {
		t.Fatalf("unexpected labels %#v", tasks[0].Labels)
	}

	actionItem.Archive(now.Add(1 * time.Hour))
	if err := repo.UpdateActionItem(ctx, actionItem); err != nil {
		t.Fatalf("UpdateActionItem() error = %v", err)
	}
	activeActionItems, err := repo.ListActionItems(ctx, project.ID, false)
	if err != nil {
		t.Fatalf("ListActionItems(active) error = %v", err)
	}
	if len(activeActionItems) != 0 {
		t.Fatalf("expected 0 active tasks, got %d", len(activeActionItems))
	}

	allActionItems, err := repo.ListActionItems(ctx, project.ID, true)
	if err != nil {
		t.Fatalf("ListActionItems(all) error = %v", err)
	}
	if len(allActionItems) != 1 || allActionItems[0].ArchivedAt == nil {
		t.Fatalf("expected archived actionItem in full list, got %#v", allActionItems)
	}

	if err := repo.DeleteActionItem(ctx, actionItem.ID); err != nil {
		t.Fatalf("DeleteActionItem() error = %v", err)
	}
	if _, err := repo.GetActionItem(ctx, actionItem.ID); err != app.ErrNotFound {
		t.Fatalf("expected app.ErrNotFound, got %v", err)
	}
}

// TestRepository_ActionItemEmbeddingsRoundTrip verifies embedding upsert/search/delete behavior.
func TestRepository_ActionItemEmbeddingsRoundTrip(t *testing.T) {
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
	project, err := domain.NewProjectFromInput(domain.ProjectInput{ID: "p1", Name: "Example"}, now)
	if err != nil {
		t.Fatalf("NewProjectFromInput() error = %v", err)
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
	actionItem, err := domain.NewActionItemForTest(domain.ActionItemInput{
		Kind:      domain.KindPlan,
		ID:        "t1",
		ProjectID: project.ID,
		ColumnID:  column.ID,
		Position:  0,
		Title:     "ActionItem with embedding",
		Priority:  domain.PriorityMedium,
	}, now)
	if err != nil {
		t.Fatalf("NewActionItem() error = %v", err)
	}
	if err := repo.CreateActionItem(ctx, actionItem); err != nil {
		t.Fatalf("CreateActionItem() error = %v", err)
	}

	if err := repo.UpsertEmbeddingDocument(ctx, app.EmbeddingDocument{
		SubjectType:      app.EmbeddingSubjectTypeWorkItem,
		SubjectID:        actionItem.ID,
		ProjectID:        project.ID,
		SearchTargetType: app.EmbeddingSearchTargetTypeWorkItem,
		SearchTargetID:   actionItem.ID,
		Content:          "actionItem embedding content",
		ContentHash:      "hash123",
		Vector:           []float32{0.1, 0.2, 0.3},
		UpdatedAt:        now,
	}); err != nil {
		t.Fatalf("UpsertEmbeddingDocument() error = %v", err)
	}

	rows, err := repo.SearchEmbeddingDocuments(ctx, app.EmbeddingSearchInput{
		ProjectIDs:        []string{project.ID},
		SearchTargetTypes: []app.EmbeddingSearchTargetType{app.EmbeddingSearchTargetTypeWorkItem},
		Vector:            []float32{0.1, 0.2, 0.3},
		Limit:             10,
	})
	if err != nil {
		t.Fatalf("SearchEmbeddingDocuments() error = %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 embedding match, got %d", len(rows))
	}
	if rows[0].SearchTargetID != actionItem.ID {
		t.Fatalf("expected actionItem id %q, got %q", actionItem.ID, rows[0].SearchTargetID)
	}

	if err := repo.DeleteEmbeddingDocument(ctx, app.EmbeddingSubjectTypeWorkItem, actionItem.ID); err != nil {
		t.Fatalf("DeleteEmbeddingDocument() error = %v", err)
	}
	rows, err = repo.SearchEmbeddingDocuments(ctx, app.EmbeddingSearchInput{
		ProjectIDs:        []string{project.ID},
		SearchTargetTypes: []app.EmbeddingSearchTargetType{app.EmbeddingSearchTargetTypeWorkItem},
		Vector:            []float32{0.1, 0.2, 0.3},
		Limit:             10,
	})
	if err != nil {
		t.Fatalf("SearchEmbeddingDocuments(after delete) error = %v", err)
	}
	if len(rows) != 0 {
		t.Fatalf("expected 0 embedding matches after delete, got %d", len(rows))
	}
}

// TestRepository_EmbeddingDocumentsRoundTripMixedSubjectFamilies verifies generic embedding documents round-trip across work items, thread context, and project documents.
func TestRepository_EmbeddingDocumentsRoundTripMixedSubjectFamilies(t *testing.T) {
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

	now := time.Date(2026, 3, 29, 11, 0, 0, 0, time.UTC)
	project, err := domain.NewProjectFromInput(domain.ProjectInput{ID: "p1", Name: "Example", Description: "Project description"}, now)
	if err != nil {
		t.Fatalf("NewProjectFromInput() error = %v", err)
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
	actionItem, err := domain.NewActionItemForTest(domain.ActionItemInput{
		Kind:      domain.KindPlan,
		ID:        "t1",
		ProjectID: project.ID,
		ColumnID:  column.ID,
		Position:  0,
		Title:     "ActionItem with embedding",
		Priority:  domain.PriorityMedium,
	}, now)
	if err != nil {
		t.Fatalf("NewActionItem() error = %v", err)
	}
	if err := repo.CreateActionItem(ctx, actionItem); err != nil {
		t.Fatalf("CreateActionItem() error = %v", err)
	}

	threadSubjectID := app.BuildThreadContextSubjectID(domain.CommentTarget{
		ProjectID:  project.ID,
		TargetType: domain.CommentTargetTypeActionItem,
		TargetID:   actionItem.ID,
	})
	rows := []app.EmbeddingDocument{
		{
			SubjectType:      app.EmbeddingSubjectTypeWorkItem,
			SubjectID:        actionItem.ID,
			ProjectID:        project.ID,
			SearchTargetType: app.EmbeddingSearchTargetTypeWorkItem,
			SearchTargetID:   actionItem.ID,
			Content:          "work item content",
			ContentHash:      "hash-work-item",
			Vector:           []float32{0.1, 0.2, 0.3},
			UpdatedAt:        now,
		},
		{
			SubjectType:      app.EmbeddingSubjectTypeThreadContext,
			SubjectID:        threadSubjectID,
			ProjectID:        project.ID,
			SearchTargetType: app.EmbeddingSearchTargetTypeWorkItem,
			SearchTargetID:   actionItem.ID,
			Content:          "thread context content",
			ContentHash:      "hash-thread-context",
			Vector:           []float32{0.2, 0.1, 0.3},
			UpdatedAt:        now,
		},
		{
			SubjectType:      app.EmbeddingSubjectTypeProjectDocument,
			SubjectID:        project.ID,
			ProjectID:        project.ID,
			SearchTargetType: app.EmbeddingSearchTargetTypeProject,
			SearchTargetID:   project.ID,
			Content:          "project document content",
			ContentHash:      "hash-project-document",
			Vector:           []float32{0.3, 0.2, 0.1},
			UpdatedAt:        now,
		},
	}
	for _, row := range rows {
		if err := repo.UpsertEmbeddingDocument(ctx, row); err != nil {
			t.Fatalf("UpsertEmbeddingDocument(%s) error = %v", row.SubjectType, err)
		}
	}

	matches, err := repo.SearchEmbeddingDocuments(ctx, app.EmbeddingSearchInput{
		ProjectIDs: []string{project.ID},
		SubjectTypes: []app.EmbeddingSubjectType{
			app.EmbeddingSubjectTypeWorkItem,
			app.EmbeddingSubjectTypeThreadContext,
			app.EmbeddingSubjectTypeProjectDocument,
		},
		SearchTargetTypes: []app.EmbeddingSearchTargetType{
			app.EmbeddingSearchTargetTypeWorkItem,
			app.EmbeddingSearchTargetTypeProject,
		},
		Vector: []float32{0.1, 0.2, 0.3},
		Limit:  10,
	})
	if err != nil {
		t.Fatalf("SearchEmbeddingDocuments() error = %v", err)
	}
	if len(matches) != 3 {
		t.Fatalf("expected 3 embedding matches, got %d", len(matches))
	}
	seen := map[app.EmbeddingSubjectType]bool{}
	for _, match := range matches {
		seen[match.SubjectType] = true
	}
	for _, subjectType := range []app.EmbeddingSubjectType{
		app.EmbeddingSubjectTypeWorkItem,
		app.EmbeddingSubjectTypeThreadContext,
		app.EmbeddingSubjectTypeProjectDocument,
	} {
		if !seen[subjectType] {
			t.Fatalf("missing subject type %s in search results %#v", subjectType, matches)
		}
	}

	if err := repo.DeleteEmbeddingDocument(ctx, app.EmbeddingSubjectTypeThreadContext, threadSubjectID); err != nil {
		t.Fatalf("DeleteEmbeddingDocument(thread_context) error = %v", err)
	}
	matches, err = repo.SearchEmbeddingDocuments(ctx, app.EmbeddingSearchInput{
		ProjectIDs: []string{project.ID},
		SearchTargetTypes: []app.EmbeddingSearchTargetType{
			app.EmbeddingSearchTargetTypeWorkItem,
			app.EmbeddingSearchTargetTypeProject,
		},
		Vector: []float32{0.1, 0.2, 0.3},
		Limit:  10,
	})
	if err != nil {
		t.Fatalf("SearchEmbeddingDocuments(after delete) error = %v", err)
	}
	if len(matches) != 2 {
		t.Fatalf("expected 2 embedding matches after delete, got %d", len(matches))
	}
}

// TestRepository_ListCommentTargets verifies the repository can discover mixed comment targets for reindexing.
func TestRepository_ListCommentTargets(t *testing.T) {
	ctx := context.Background()
	repo, err := OpenInMemory()
	if err != nil {
		t.Fatalf("OpenInMemory() error = %v", err)
	}
	t.Cleanup(func() {
		_ = repo.Close()
	})

	now := time.Date(2026, 3, 29, 11, 30, 0, 0, time.UTC)
	project, err := domain.NewProjectFromInput(domain.ProjectInput{ID: "p-comment-targets", Name: "Comment Targets"}, now)
	if err != nil {
		t.Fatalf("NewProjectFromInput() error = %v", err)
	}
	if err := repo.CreateProject(ctx, project); err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}
	column, err := domain.NewColumn("c-comment-targets", project.ID, "To Do", 0, 0, now)
	if err != nil {
		t.Fatalf("NewColumn() error = %v", err)
	}
	if err := repo.CreateColumn(ctx, column); err != nil {
		t.Fatalf("CreateColumn() error = %v", err)
	}
	actionItem, err := domain.NewActionItemForTest(domain.ActionItemInput{
		Kind:      domain.KindPlan,
		ID:        "t-comment-targets",
		ProjectID: project.ID,
		ColumnID:  column.ID,
		Position:  0,
		Title:     "ActionItem target",
		Priority:  domain.PriorityLow,
	}, now)
	if err != nil {
		t.Fatalf("NewActionItem() error = %v", err)
	}
	if err := repo.CreateActionItem(ctx, actionItem); err != nil {
		t.Fatalf("CreateActionItem() error = %v", err)
	}

	actionItemComment, err := domain.NewComment(domain.CommentInput{
		ID:           "c-actionItem",
		ProjectID:    project.ID,
		TargetType:   domain.CommentTargetTypeActionItem,
		TargetID:     actionItem.ID,
		BodyMarkdown: "actionItem comment",
		ActorID:      "user-1",
		ActorName:    "User One",
		ActorType:    domain.ActorTypeUser,
	}, now)
	if err != nil {
		t.Fatalf("NewComment(actionItem) error = %v", err)
	}
	projectComment, err := domain.NewComment(domain.CommentInput{
		ID:           "c-project",
		ProjectID:    project.ID,
		TargetType:   domain.CommentTargetTypeProject,
		TargetID:     project.ID,
		BodyMarkdown: "project comment",
		ActorID:      "user-2",
		ActorName:    "User Two",
		ActorType:    domain.ActorTypeUser,
	}, now.Add(time.Minute))
	if err != nil {
		t.Fatalf("NewComment(project) error = %v", err)
	}
	if err := repo.CreateComment(ctx, actionItemComment); err != nil {
		t.Fatalf("CreateComment(actionItem) error = %v", err)
	}
	if err := repo.CreateComment(ctx, projectComment); err != nil {
		t.Fatalf("CreateComment(project) error = %v", err)
	}

	targets, err := repo.ListCommentTargets(ctx, project.ID)
	if err != nil {
		t.Fatalf("ListCommentTargets() error = %v", err)
	}
	if len(targets) != 2 {
		t.Fatalf("expected 2 comment targets, got %d", len(targets))
	}
	want := map[domain.CommentTargetType]string{
		domain.CommentTargetTypeActionItem: actionItem.ID,
		domain.CommentTargetTypeProject:    project.ID,
	}
	for _, target := range targets {
		if want[target.TargetType] != target.TargetID {
			t.Fatalf("unexpected comment target %#v", targets)
		}
	}
}

// TestRepository_ActionItemEmbeddingMethodsReturnVecUnavailable verifies vector methods return a stable error when sqlite-vec is unavailable.
func TestRepository_ActionItemEmbeddingMethodsReturnVecUnavailable(t *testing.T) {
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

	err = repo.UpsertEmbeddingDocument(ctx, app.EmbeddingDocument{
		SubjectType:      app.EmbeddingSubjectTypeWorkItem,
		SubjectID:        "t1",
		ProjectID:        "p1",
		SearchTargetType: app.EmbeddingSearchTargetTypeWorkItem,
		SearchTargetID:   "t1",
		Content:          "actionItem embedding content",
		ContentHash:      "hash123",
		Vector:           []float32{0.1, 0.2, 0.3},
		UpdatedAt:        time.Date(2026, 3, 3, 14, 0, 0, 0, time.UTC),
	})
	if !errors.Is(err, errSQLiteVecUnavailable) {
		t.Fatalf("expected errSQLiteVecUnavailable, got %v", err)
	}

	_, err = repo.SearchEmbeddingDocuments(ctx, app.EmbeddingSearchInput{
		ProjectIDs:        []string{"p1"},
		SearchTargetTypes: []app.EmbeddingSearchTargetType{app.EmbeddingSearchTargetTypeWorkItem},
		Vector:            []float32{0.1, 0.2, 0.3},
		Limit:             10,
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
	project, err := domain.NewProjectFromInput(domain.ProjectInput{ID: "p1", Name: "Example"}, now)
	if err != nil {
		t.Fatalf("NewProjectFromInput() error = %v", err)
	}
	if err := repo.CreateProject(ctx, project); err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}

	comment2, err := domain.NewComment(domain.CommentInput{
		ID:           "c2",
		ProjectID:    project.ID,
		TargetType:   domain.CommentTargetTypeActionItem,
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
		TargetType:   domain.CommentTargetTypeActionItem,
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

	actionItemComments, err := repo.ListCommentsByTarget(ctx, domain.CommentTarget{
		ProjectID:  project.ID,
		TargetType: domain.CommentTargetTypeActionItem,
		TargetID:   "t1",
	})
	if err != nil {
		t.Fatalf("ListCommentsByTarget(actionItem) error = %v", err)
	}
	if len(actionItemComments) != 2 {
		t.Fatalf("expected 2 actionItem comments, got %d", len(actionItemComments))
	}
	if actionItemComments[0].ID != "c1" || actionItemComments[1].ID != "c2" {
		t.Fatalf("expected deterministic created_at/id ordering, got %#v", actionItemComments)
	}
	if actionItemComments[1].ActorType != domain.ActorTypeAgent {
		t.Fatalf("expected normalized actor type agent, got %q", actionItemComments[1].ActorType)
	}
	if actionItemComments[1].ActorID != "agent-1" || actionItemComments[1].ActorName != "Agent One" {
		t.Fatalf("expected actor tuple to persist, got %#v", actionItemComments[1])
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
	actionItem, err := svc.CreateActionItem(baseCtx, app.CreateActionItemInput{
		ProjectID:      project.ID,
		ColumnID:       column.ID,
		Title:          "ActionItem",
		Priority:       domain.PriorityMedium,
		StructuralType: domain.StructuralTypeDroplet,
	})
	if err != nil {
		t.Fatalf("CreateActionItem() error = %v", err)
	}

	ctx := app.WithMutationActor(baseCtx, app.MutationActor{
		ActorID:   "user-1",
		ActorName: "Evan Schultz",
		ActorType: domain.ActorTypeUser,
	})
	comment, err := svc.CreateComment(ctx, app.CreateCommentInput{
		ProjectID:    project.ID,
		TargetType:   domain.CommentTargetTypeActionItem,
		TargetID:     actionItem.ID,
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
		TargetType: domain.CommentTargetTypeActionItem,
		TargetID:   actionItem.ID,
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
	if _, err := repo.GetActionItem(ctx, "missing"); err != app.ErrNotFound {
		t.Fatalf("expected app.ErrNotFound for actionItem, got %v", err)
	}
	if err := repo.DeleteActionItem(ctx, "missing"); err != app.ErrNotFound {
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
	project, _ := domain.NewProjectFromInput(domain.ProjectInput{ID: "p1", Name: "Alpha", Description: "desc"}, now)
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

// TestRepository_ProjectFirstClassFieldsRoundTrip verifies that the six
// Drop 4a L4 first-class project-node fields (HyllaArtifactRef,
// RepoBareRoot, RepoPrimaryWorktree, Language, BuildTool,
// DevMcpServerName) survive INSERT → SELECT (single + list paths) and
// UPDATE round-trips through the SQLite repository.
func TestRepository_ProjectFirstClassFieldsRoundTrip(t *testing.T) {
	ctx := context.Background()
	repo, err := Open(filepath.Join(t.TempDir(), "tillsyn.db"))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(func() {
		_ = repo.Close()
	})

	now := time.Date(2026, 5, 3, 12, 0, 0, 0, time.UTC)
	project, err := domain.NewProjectFromInput(domain.ProjectInput{
		ID:                  "p1",
		Name:                "Tillsyn",
		Description:         "tracker",
		HyllaArtifactRef:    "github.com/evanmschultz/tillsyn@main",
		RepoBareRoot:        "/Users/evan/code/tillsyn",
		RepoPrimaryWorktree: "/Users/evan/code/tillsyn/main",
		Language:            "go",
		BuildTool:           "mage",
		DevMcpServerName:    "tillsyn-dev",
	}, now)
	if err != nil {
		t.Fatalf("NewProjectFromInput() error = %v", err)
	}
	if err := repo.CreateProject(ctx, project); err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}

	loaded, err := repo.GetProject(ctx, project.ID)
	if err != nil {
		t.Fatalf("GetProject() error = %v", err)
	}
	if loaded.HyllaArtifactRef != "github.com/evanmschultz/tillsyn@main" {
		t.Fatalf("HyllaArtifactRef = %q", loaded.HyllaArtifactRef)
	}
	if loaded.RepoBareRoot != "/Users/evan/code/tillsyn" {
		t.Fatalf("RepoBareRoot = %q", loaded.RepoBareRoot)
	}
	if loaded.RepoPrimaryWorktree != "/Users/evan/code/tillsyn/main" {
		t.Fatalf("RepoPrimaryWorktree = %q", loaded.RepoPrimaryWorktree)
	}
	if loaded.Language != "go" || loaded.BuildTool != "mage" || loaded.DevMcpServerName != "tillsyn-dev" {
		t.Fatalf("scalar fields not persisted: %+v", loaded)
	}

	listed, err := repo.ListProjects(ctx, false)
	if err != nil {
		t.Fatalf("ListProjects() error = %v", err)
	}
	if len(listed) != 1 {
		t.Fatalf("ListProjects len = %d, want 1", len(listed))
	}
	if listed[0].HyllaArtifactRef != "github.com/evanmschultz/tillsyn@main" || listed[0].Language != "go" {
		t.Fatalf("ListProjects field decode missed: %+v", listed[0])
	}

	// Update path: change every first-class field, ensure persisted.
	loaded.HyllaArtifactRef = "github.com/x/y@v1"
	loaded.RepoBareRoot = "/abs/x"
	loaded.RepoPrimaryWorktree = "/abs/x/main"
	loaded.Language = "fe"
	loaded.BuildTool = "npm"
	loaded.DevMcpServerName = "x-dev"
	if err := repo.UpdateProject(ctx, loaded); err != nil {
		t.Fatalf("UpdateProject() error = %v", err)
	}
	reloaded, err := repo.GetProject(ctx, project.ID)
	if err != nil {
		t.Fatalf("GetProject(reload) error = %v", err)
	}
	if reloaded.HyllaArtifactRef != "github.com/x/y@v1" || reloaded.RepoBareRoot != "/abs/x" ||
		reloaded.RepoPrimaryWorktree != "/abs/x/main" || reloaded.Language != "fe" ||
		reloaded.BuildTool != "npm" || reloaded.DevMcpServerName != "x-dev" {
		t.Fatalf("UpdateProject field round-trip failed: %+v", reloaded)
	}
}

// TestRepository_ListProjectsExcludesGlobalAuthSentinel verifies the hidden global auth-routing project does not leak into user-facing project inventory.
func TestRepository_ListProjectsExcludesGlobalAuthSentinel(t *testing.T) {
	ctx := context.Background()
	repo, err := Open(filepath.Join(t.TempDir(), "auth-global.db"))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(func() {
		_ = repo.Close()
	})

	var sentinelCount int
	if err := repo.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM projects WHERE id = ?`, domain.AuthRequestGlobalProjectID).Scan(&sentinelCount); err != nil {
		t.Fatalf("query sentinel project count error = %v", err)
	}
	if sentinelCount != 1 {
		t.Fatalf("sentinel project count = %d, want 1", sentinelCount)
	}

	projects, err := repo.ListProjects(ctx, true)
	if err != nil {
		t.Fatalf("ListProjects(includeArchived) error = %v", err)
	}
	if len(projects) != 0 {
		t.Fatalf("ListProjects(includeArchived) = %#v, want hidden sentinel excluded", projects)
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
	project, _ := domain.NewProjectFromInput(domain.ProjectInput{ID: "p1", Name: "Alpha"}, now)
	if err := repo.CreateProject(ctx, project); err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}
	column, _ := domain.NewColumn("c1", project.ID, "To Do", 0, 0, now)
	if err := repo.CreateColumn(ctx, column); err != nil {
		t.Fatalf("CreateColumn() error = %v", err)
	}
	actionItem, _ := domain.NewActionItemForTest(domain.ActionItemInput{
		Kind:      domain.KindPlan,
		ID:        "t1",
		ProjectID: project.ID,
		ColumnID:  column.ID,
		Position:  0,
		Title:     "ActionItem",
		Priority:  domain.PriorityMedium,
	}, now)
	if err := repo.CreateActionItem(ctx, actionItem); err != nil {
		t.Fatalf("CreateActionItem() error = %v", err)
	}

	if err := repo.DeleteProject(ctx, project.ID); err != nil {
		t.Fatalf("DeleteProject() error = %v", err)
	}
	if _, err := repo.GetProject(ctx, project.ID); err != app.ErrNotFound {
		t.Fatalf("expected app.ErrNotFound for project, got %v", err)
	}
	if _, err := repo.GetActionItem(ctx, actionItem.ID); err != app.ErrNotFound {
		t.Fatalf("expected app.ErrNotFound for actionItem cascade, got %v", err)
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

	project, _ := domain.NewProjectFromInput(domain.ProjectInput{ID: "p1", Name: "Legacy"}, time.Date(2026, 2, 21, 12, 0, 0, 0, time.UTC))
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
	p, _ := domain.NewProjectFromInput(domain.ProjectInput{ID: "missing", Name: "nope"}, now)
	if err := repo.UpdateProject(context.Background(), p); err != app.ErrNotFound {
		t.Fatalf("expected app.ErrNotFound for UpdateProject, got %v", err)
	}

	c, _ := domain.NewColumn("missing-col", "missing", "todo", 0, 0, now)
	if err := repo.UpdateColumn(context.Background(), c); err != app.ErrNotFound {
		t.Fatalf("expected app.ErrNotFound for UpdateColumn, got %v", err)
	}

	tk, _ := domain.NewActionItemForTest(domain.ActionItemInput{
		Kind:      domain.KindPlan,
		ID:        "missing-actionItem",
		ProjectID: "missing",
		ColumnID:  "missing-col",
		Position:  0,
		Title:     "x",
		Priority:  domain.PriorityLow,
	}, now)
	if err := repo.UpdateActionItem(context.Background(), tk); err != app.ErrNotFound {
		t.Fatalf("expected app.ErrNotFound for UpdateActionItem, got %v", err)
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
	project, _ := domain.NewProjectFromInput(domain.ProjectInput{ID: "p1", Name: "Events"}, now)
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

	actionItem, _ := domain.NewActionItemForTest(domain.ActionItemInput{
		Kind:           domain.KindPlan,
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
	if err := repo.CreateActionItem(ctx, actionItem); err != nil {
		t.Fatalf("CreateActionItem() error = %v", err)
	}

	if err := actionItem.UpdateDetails("Track me v2", actionItem.Description, actionItem.Priority, actionItem.DueAt, actionItem.Labels, now.Add(time.Minute)); err != nil {
		t.Fatalf("UpdateDetails() error = %v", err)
	}
	actionItem.UpdatedByActor = "agent-1"
	actionItem.UpdatedByName = "Planner Bot"
	actionItem.UpdatedByType = domain.ActorTypeAgent
	if err := repo.UpdateActionItem(ctx, actionItem); err != nil {
		t.Fatalf("UpdateActionItem(update) error = %v", err)
	}

	if err := actionItem.Move(done.ID, 1, now.Add(2*time.Minute)); err != nil {
		t.Fatalf("Move() error = %v", err)
	}
	actionItem.UpdatedByActor = "user-2"
	actionItem.UpdatedByName = "Evan Schultz"
	actionItem.UpdatedByType = domain.ActorTypeUser
	if err := repo.UpdateActionItem(ctx, actionItem); err != nil {
		t.Fatalf("UpdateActionItem(move) error = %v", err)
	}

	actionItem.Archive(now.Add(3 * time.Minute))
	actionItem.UpdatedByActor = "user-3"
	actionItem.UpdatedByName = "Evan Schultz"
	actionItem.UpdatedByType = domain.ActorTypeUser
	if err := repo.UpdateActionItem(ctx, actionItem); err != nil {
		t.Fatalf("UpdateActionItem(archive) error = %v", err)
	}

	actionItem.Restore(now.Add(4 * time.Minute))
	actionItem.UpdatedByActor = "user-4"
	actionItem.UpdatedByName = "Evan Schultz"
	actionItem.UpdatedByType = domain.ActorTypeUser
	if err := repo.UpdateActionItem(ctx, actionItem); err != nil {
		t.Fatalf("UpdateActionItem(restore) error = %v", err)
	}

	if err := repo.DeleteActionItem(ctx, actionItem.ID); err != nil {
		t.Fatalf("DeleteActionItem() error = %v", err)
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

// TestRepository_ActionItemLifecyclePreservesMutationActorName verifies actionItem change events keep request actor_name attribution.
func TestRepository_ActionItemLifecyclePreservesMutationActorName(t *testing.T) {
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
			project, _ := domain.NewProjectFromInput(domain.ProjectInput{ID: "p1", Name: "Inbox"}, now)
			if err := repo.CreateProject(baseCtx, project); err != nil {
				t.Fatalf("CreateProject() error = %v", err)
			}
			todo, _ := domain.NewColumn("c1", project.ID, "To Do", 0, 0, now)
			if err := repo.CreateColumn(baseCtx, todo); err != nil {
				t.Fatalf("CreateColumn() error = %v", err)
			}
			actionItem, _ := domain.NewActionItemForTest(domain.ActionItemInput{
				Kind:      domain.KindPlan,
				ID:        "t1",
				ProjectID: project.ID,
				ColumnID:  todo.ID,
				Position:  0,
				Title:     "Ownership",
				Priority:  domain.PriorityLow,
			}, now)
			if err := repo.CreateActionItem(ctx, actionItem); err != nil {
				t.Fatalf("CreateActionItem() error = %v", err)
			}

			if err := actionItem.UpdateDetails("Ownership v2", actionItem.Description, actionItem.Priority, actionItem.DueAt, actionItem.Labels, now.Add(time.Minute)); err != nil {
				t.Fatalf("UpdateDetails() error = %v", err)
			}
			if err := repo.UpdateActionItem(ctx, actionItem); err != nil {
				t.Fatalf("UpdateActionItem() error = %v", err)
			}
			if err := repo.DeleteActionItem(ctx, actionItem.ID); err != nil {
				t.Fatalf("DeleteActionItem() error = %v", err)
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

// TestRepository_ServiceCreateActionItemPersistsHumanActorName verifies service-provided display names reach persisted change events.
func TestRepository_ServiceCreateActionItemPersistsHumanActorName(t *testing.T) {
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
	created, err := svc.CreateActionItem(ctx, app.CreateActionItemInput{
		ProjectID:      project.ID,
		ColumnID:       column.ID,
		Title:          "Ownership",
		Priority:       domain.PriorityMedium,
		CreatedByActor: "user-1",
		CreatedByName:  "Evan Schultz",
		UpdatedByType:  domain.ActorTypeUser,
		StructuralType: domain.StructuralTypeDroplet,
	})
	if err != nil {
		t.Fatalf("CreateActionItem() error = %v", err)
	}
	events, err := repo.ListProjectChangeEvents(ctx, project.ID, 1)
	if err != nil {
		t.Fatalf("ListProjectChangeEvents() error = %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d (%#v)", len(events), events)
	}
	if events[0].ActionItemID != created.ID {
		t.Fatalf("expected event work item id %q, got %q", created.ID, events[0].ActionItemID)
	}
	if events[0].ActorID != "user-1" || events[0].ActorName != "Evan Schultz" {
		t.Fatalf("expected human attribution user-1/Evan Schultz, got %q/%q", events[0].ActorID, events[0].ActorName)
	}
	loaded, err := repo.GetActionItem(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetActionItem() error = %v", err)
	}
	if loaded.CreatedByName != "Evan Schultz" || loaded.UpdatedByName != "Evan Schultz" {
		t.Fatalf("expected persisted actionItem names Evan Schultz/Evan Schultz, got %q/%q", loaded.CreatedByName, loaded.UpdatedByName)
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
	project, _ := domain.NewProjectFromInput(domain.ProjectInput{ID: "p-kind", Name: "Kinds"}, now)
	if err := repo.CreateProject(ctx, project); err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}

	// Attach a payload schema to a custom kind id not seeded by the 12-value
	// Kind enum. Custom kinds remain allowed in the catalog but cannot be
	// used for action-item rows; the test exercises catalog storage, not
	// action-item creation. The project_allowed_kinds closure can still mix
	// built-in and custom ids.
	kind, err := domain.NewKindDefinition(domain.KindDefinitionInput{
		ID:                "custom-refactor",
		DisplayName:       "Custom Refactor",
		AppliesTo:         []domain.KindAppliesTo{domain.KindAppliesToPlan},
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
	if loadedKind.DisplayName != "Custom Refactor" {
		t.Fatalf("unexpected kind display name %q", loadedKind.DisplayName)
	}

	if err := repo.SetProjectAllowedKinds(ctx, project.ID, []domain.KindID{kind.ID, domain.KindID(domain.KindPlan)}); err != nil {
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
	project, _ := domain.NewProjectFromInput(domain.ProjectInput{ID: "p-lease", Name: "Leases"}, now)
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

// TestRepository_CapabilityLeaseDistinctOrchestratorIdentitiesAtProjectScope exercises the
// distinct-identity orchestrator overlap fix end-to-end through a real SQLite-backed Service.
// It proves the app-layer sameIdentity branch in ensureOrchestratorOverlapPolicy cooperates with
// the SQLite capability_leases schema (no UNIQUE on (project_id, scope_type, scope_id, role))
// so two different orchestrator identities can hold concurrent project-scope leases and
// revoke-one-leaves-the-other behaves correctly at the persistence boundary. Complements the
// app-package fake-repo coverage for acceptance 2.1 and 2.3 from the DROP_1 multi-orch-auth
// hotfix worklog §7.2.2.
func TestRepository_CapabilityLeaseDistinctOrchestratorIdentitiesAtProjectScope(t *testing.T) {
	ctx := context.Background()
	repo, err := OpenInMemory()
	if err != nil {
		t.Fatalf("OpenInMemory() error = %v", err)
	}
	t.Cleanup(func() {
		_ = repo.Close()
	})

	now := time.Date(2026, 4, 17, 12, 0, 0, 0, time.UTC)
	ids := []string{"p-multi-orch", "lease-token-steward", "lease-token-drop-1"}
	idx := 0
	svc := app.NewService(repo, func() string {
		id := ids[idx]
		idx++
		return id
	}, func() time.Time {
		return now
	}, app.ServiceConfig{})

	project, err := svc.CreateProject(ctx, "Multi-Orch Project", "")
	if err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}

	leaseSteward, err := svc.IssueCapabilityLease(ctx, app.IssueCapabilityLeaseInput{
		ProjectID:       project.ID,
		ScopeType:       domain.CapabilityScopeProject,
		Role:            domain.CapabilityRoleOrchestrator,
		AgentName:       "orch-steward",
		AgentInstanceID: "orch-steward-inst",
	})
	if err != nil {
		t.Fatalf("IssueCapabilityLease(orch-steward) error = %v", err)
	}
	if leaseSteward.LeaseToken == "" || leaseSteward.InstanceID == "" {
		t.Fatalf("IssueCapabilityLease(orch-steward) returned empty token/instance: %#v", leaseSteward)
	}

	leaseDrop1, err := svc.IssueCapabilityLease(ctx, app.IssueCapabilityLeaseInput{
		ProjectID:       project.ID,
		ScopeType:       domain.CapabilityScopeProject,
		Role:            domain.CapabilityRoleOrchestrator,
		AgentName:       "orch-drop-1",
		AgentInstanceID: "orch-drop-1-inst",
	})
	if err != nil {
		t.Fatalf("IssueCapabilityLease(orch-drop-1) error = %v", err)
	}
	if leaseDrop1.LeaseToken == "" || leaseDrop1.InstanceID == "" {
		t.Fatalf("IssueCapabilityLease(orch-drop-1) returned empty token/instance: %#v", leaseDrop1)
	}
	if leaseDrop1.LeaseToken == leaseSteward.LeaseToken {
		t.Fatalf("distinct orchestrator leases collided on LeaseToken: %q", leaseDrop1.LeaseToken)
	}

	active, err := svc.ListCapabilityLeases(ctx, app.ListCapabilityLeasesInput{
		ProjectID: project.ID,
		ScopeType: domain.CapabilityScopeProject,
	})
	if err != nil {
		t.Fatalf("ListCapabilityLeases() error = %v", err)
	}
	if len(active) != 2 {
		t.Fatalf("expected 2 active orchestrator leases at project scope, got %d: %#v", len(active), active)
	}
	names := map[string]bool{}
	for _, lease := range active {
		if lease.RevokedAt != nil {
			t.Fatalf("active lease %q unexpectedly has RevokedAt set: %#v", lease.AgentName, lease)
		}
		names[lease.AgentName] = true
	}
	if !names["orch-steward"] || !names["orch-drop-1"] {
		t.Fatalf("expected both orch-steward and orch-drop-1 present, got %#v", names)
	}

	if _, err := svc.RevokeCapabilityLease(ctx, app.RevokeCapabilityLeaseInput{
		AgentInstanceID: leaseSteward.InstanceID,
		Reason:          "done",
	}); err != nil {
		t.Fatalf("RevokeCapabilityLease(orch-steward) error = %v", err)
	}

	remaining, err := svc.ListCapabilityLeases(ctx, app.ListCapabilityLeasesInput{
		ProjectID: project.ID,
		ScopeType: domain.CapabilityScopeProject,
	})
	if err != nil {
		t.Fatalf("ListCapabilityLeases(after revoke) error = %v", err)
	}
	if len(remaining) != 1 {
		t.Fatalf("expected exactly 1 active lease after revoke, got %d: %#v", len(remaining), remaining)
	}
	survivor := remaining[0]
	if survivor.AgentName != "orch-drop-1" {
		t.Fatalf("expected surviving active lease to be orch-drop-1, got %q", survivor.AgentName)
	}
	if survivor.RevokedAt != nil {
		t.Fatalf("surviving lease unexpectedly has RevokedAt set: %#v", survivor)
	}
	if !survivor.ExpiresAt.After(now) {
		t.Fatalf("surviving lease ExpiresAt %v is not after now %v", survivor.ExpiresAt, now)
	}

	revokedRow, err := repo.GetCapabilityLease(ctx, leaseSteward.InstanceID)
	if err != nil {
		t.Fatalf("GetCapabilityLease(orch-steward after revoke) error = %v", err)
	}
	if revokedRow.RevokedAt == nil {
		t.Fatalf("expected revoked lease to persist RevokedAt in SQLite, got %#v", revokedRow)
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
	project, _ := domain.NewProjectFromInput(domain.ProjectInput{ID: "p-attn", Name: "Attention"}, now)
	if err := repo.CreateProject(ctx, project); err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}

	risk, err := domain.NewAttentionItem(domain.AttentionItemInput{
		ID:                 "attn-risk",
		ProjectID:          project.ID,
		ScopeType:          domain.ScopeLevelActionItem,
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
		ScopeType:          domain.ScopeLevelActionItem,
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
		ScopeType:      domain.ScopeLevelActionItem,
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
		ScopeType:          domain.ScopeLevelActionItem,
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
		ScopeType:      domain.ScopeLevelActionItem,
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
	project, _ := domain.NewProjectFromInput(domain.ProjectInput{ID: "p-attn-validate", Name: "Attention Validate"}, now)
	if err := repo.CreateProject(ctx, project); err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}

	base := domain.AttentionItem{
		ID:                 "attn-valid",
		ProjectID:          project.ID,
		ScopeType:          domain.ScopeLevelActionItem,
		ScopeID:            "actionItem-1",
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
					ScopeType: domain.ScopeLevelActionItem,
					ScopeID:   "actionItem-1",
				},
				want: domain.ErrInvalidID,
			},
			{
				name: "scope id without scope type",
				filter: domain.AttentionListFilter{
					ProjectID: project.ID,
					ScopeID:   "actionItem-1",
				},
				want: domain.ErrInvalidScopeType,
			},
			{
				name: "invalid state",
				filter: domain.AttentionListFilter{
					ProjectID: project.ID,
					ScopeType: domain.ScopeLevelActionItem,
					ScopeID:   "actionItem-1",
					States:    []domain.AttentionState{"bad-state"},
				},
				want: domain.ErrInvalidAttentionState,
			},
			{
				name: "invalid kind",
				filter: domain.AttentionListFilter{
					ProjectID: project.ID,
					ScopeType: domain.ScopeLevelActionItem,
					ScopeID:   "actionItem-1",
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

// TestRepository_AttentionItemProjectWideRoleFilterAndUpsert verifies inbox-style role filtering and handoff-style upserts.
func TestRepository_AttentionItemProjectWideRoleFilterAndUpsert(t *testing.T) {
	repo, err := OpenInMemory()
	if err != nil {
		t.Fatalf("OpenInMemory() error = %v", err)
	}
	t.Cleanup(func() {
		_ = repo.Close()
	})
	ctx := context.Background()
	now := time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC)
	project, _ := domain.NewProjectFromInput(domain.ProjectInput{ID: "p-inbox", Name: "Inbox"}, now)
	if err := repo.CreateProject(ctx, project); err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}

	mention, err := domain.NewAttentionItem(domain.AttentionItemInput{
		ID:                 "attn-mention",
		ProjectID:          project.ID,
		ScopeType:          domain.ScopeLevelActionItem,
		ScopeID:            "actionItem-1",
		Kind:               domain.AttentionKindMention,
		Summary:            "mention for qa",
		BodyMarkdown:       "Please review.",
		TargetRole:         "qa",
		RequiresUserAction: false,
		CreatedByActor:     "user-1",
		CreatedByType:      domain.ActorTypeUser,
	}, now)
	if err != nil {
		t.Fatalf("NewAttentionItem(mention) error = %v", err)
	}
	if err := repo.CreateAttentionItem(ctx, mention); err != nil {
		t.Fatalf("CreateAttentionItem(mention) error = %v", err)
	}

	handoff, err := domain.NewAttentionItem(domain.AttentionItemInput{
		ID:                 "attn-handoff",
		ProjectID:          project.ID,
		ScopeType:          domain.ScopeLevelBranch,
		ScopeID:            "branch-1",
		Kind:               domain.AttentionKindHandoff,
		Summary:            "handoff for builder",
		BodyMarkdown:       "Implement the next pass.",
		TargetRole:         "dev",
		RequiresUserAction: true,
		CreatedByActor:     "orch-1",
		CreatedByType:      domain.ActorTypeAgent,
	}, now.Add(time.Minute))
	if err != nil {
		t.Fatalf("NewAttentionItem(handoff) error = %v", err)
	}
	if err := repo.UpsertAttentionItem(ctx, handoff); err != nil {
		t.Fatalf("UpsertAttentionItem(open) error = %v", err)
	}

	builderRows, err := repo.ListAttentionItems(ctx, domain.AttentionListFilter{
		ProjectID:      project.ID,
		UnresolvedOnly: true,
		TargetRole:     "builder",
	})
	if err != nil {
		t.Fatalf("ListAttentionItems(builder project-wide) error = %v", err)
	}
	if len(builderRows) != 1 || builderRows[0].ID != handoff.ID || builderRows[0].TargetRole != "builder" {
		t.Fatalf("expected one canonical builder inbox row, got %#v", builderRows)
	}

	qaRows, err := repo.ListAttentionItems(ctx, domain.AttentionListFilter{
		ProjectID:      project.ID,
		UnresolvedOnly: true,
		TargetRole:     "qa",
	})
	if err != nil {
		t.Fatalf("ListAttentionItems(qa project-wide) error = %v", err)
	}
	if len(qaRows) != 1 || qaRows[0].ID != mention.ID {
		t.Fatalf("expected one qa mention row, got %#v", qaRows)
	}

	handoff.State = domain.AttentionStateResolved
	resolvedAt := now.Add(2 * time.Minute)
	handoff.ResolvedAt = &resolvedAt
	handoff.ResolvedByActor = "qa-1"
	handoff.ResolvedByType = domain.ActorTypeUser
	if err := repo.UpsertAttentionItem(ctx, handoff); err != nil {
		t.Fatalf("UpsertAttentionItem(resolved) error = %v", err)
	}

	builderRows, err = repo.ListAttentionItems(ctx, domain.AttentionListFilter{
		ProjectID:      project.ID,
		UnresolvedOnly: true,
		TargetRole:     "builder",
	})
	if err != nil {
		t.Fatalf("ListAttentionItems(builder after resolve) error = %v", err)
	}
	if len(builderRows) != 0 {
		t.Fatalf("expected resolved upsert to disappear from unresolved inbox, got %#v", builderRows)
	}
}

// TestRepository_PersistsProjectKindAndActionItemScope verifies new kind/scope columns round-trip.
func TestRepository_PersistsProjectKindAndActionItemScope(t *testing.T) {
	ctx := context.Background()
	repo, err := OpenInMemory()
	if err != nil {
		t.Fatalf("OpenInMemory() error = %v", err)
	}
	t.Cleanup(func() {
		_ = repo.Close()
	})

	now := time.Date(2026, 2, 24, 10, 0, 0, 0, time.UTC)
	project, _ := domain.NewProjectFromInput(domain.ProjectInput{ID: "p-scope", Name: "Scope"}, now)
	if err := repo.CreateProject(ctx, project); err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}
	_, err = repo.GetProject(ctx, project.ID)
	if err != nil {
		t.Fatalf("GetProject() error = %v", err)
	}

	column, _ := domain.NewColumn("c-scope", project.ID, "To Do", 0, 0, now)
	if err := repo.CreateColumn(ctx, column); err != nil {
		t.Fatalf("CreateColumn() error = %v", err)
	}
	actionItem, err := domain.NewActionItemForTest(domain.ActionItemInput{
		ID:        "t-scope",
		ProjectID: project.ID,
		ColumnID:  column.ID,
		Scope:     domain.KindAppliesToDiscussion,
		Kind:      domain.KindDiscussion,
		Position:  0,
		Title:     "phase",
		Priority:  domain.PriorityMedium,
	}, now)
	if err != nil {
		t.Fatalf("NewActionItem() error = %v", err)
	}
	if err := repo.CreateActionItem(ctx, actionItem); err != nil {
		t.Fatalf("CreateActionItem() error = %v", err)
	}
	loadedActionItem, err := repo.GetActionItem(ctx, actionItem.ID)
	if err != nil {
		t.Fatalf("GetActionItem() error = %v", err)
	}
	if loadedActionItem.Scope != domain.KindAppliesToDiscussion {
		t.Fatalf("expected persisted actionItem scope phase, got %q", loadedActionItem.Scope)
	}

	nestedPhaseActionItem, err := domain.NewActionItemForTest(domain.ActionItemInput{
		ID:        "t-nested-phase",
		ProjectID: project.ID,
		ParentID:  actionItem.ID,
		ColumnID:  column.ID,
		Scope:     domain.KindAppliesToDiscussion,
		Kind:      domain.KindDiscussion,
		Position:  1,
		Title:     "nested phase",
		Priority:  domain.PriorityMedium,
	}, now)
	if err != nil {
		t.Fatalf("NewActionItem(nested phase) error = %v", err)
	}
	if err := repo.CreateActionItem(ctx, nestedPhaseActionItem); err != nil {
		t.Fatalf("CreateActionItem(nested phase) error = %v", err)
	}
	loadedNestedPhaseActionItem, err := repo.GetActionItem(ctx, nestedPhaseActionItem.ID)
	if err != nil {
		t.Fatalf("GetActionItem(nested phase) error = %v", err)
	}
	if loadedNestedPhaseActionItem.Scope != domain.KindAppliesToDiscussion {
		t.Fatalf("expected persisted actionItem scope phase, got %q", loadedNestedPhaseActionItem.Scope)
	}
}

// TestRepository_PersistsActionItemRole verifies the role column round-trips
// across create + get + list + update on an action item, including the empty-
// role default and a reassign to a different closed-enum value.
func TestRepository_PersistsActionItemRole(t *testing.T) {
	ctx := context.Background()
	repo, err := OpenInMemory()
	if err != nil {
		t.Fatalf("OpenInMemory() error = %v", err)
	}
	t.Cleanup(func() {
		_ = repo.Close()
	})

	now := time.Date(2026, 5, 1, 9, 0, 0, 0, time.UTC)
	project, _ := domain.NewProjectFromInput(domain.ProjectInput{ID: "p-role", Name: "Role"}, now)
	if err := repo.CreateProject(ctx, project); err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}
	column, _ := domain.NewColumn("c-role", project.ID, "To Do", 0, 0, now)
	if err := repo.CreateColumn(ctx, column); err != nil {
		t.Fatalf("CreateColumn() error = %v", err)
	}

	// Empty-role item: confirms the empty-string default round-trips and that
	// the SELECT/INSERT column ordering does not crash on the zero value.
	emptyItem, err := domain.NewActionItemForTest(domain.ActionItemInput{
		ID:        "t-role-empty",
		ProjectID: project.ID,
		ColumnID:  column.ID,
		Kind:      domain.KindBuild,
		Position:  0,
		Title:     "no role",
		Priority:  domain.PriorityMedium,
	}, now)
	if err != nil {
		t.Fatalf("NewActionItem(empty role) error = %v", err)
	}
	if err := repo.CreateActionItem(ctx, emptyItem); err != nil {
		t.Fatalf("CreateActionItem(empty role) error = %v", err)
	}
	loadedEmpty, err := repo.GetActionItem(ctx, emptyItem.ID)
	if err != nil {
		t.Fatalf("GetActionItem(empty role) error = %v", err)
	}
	if loadedEmpty.Role != "" {
		t.Fatalf("expected empty persisted role, got %q", loadedEmpty.Role)
	}

	// Builder-role item: confirms a closed-enum value round-trips through
	// create + get.
	builderItem, err := domain.NewActionItemForTest(domain.ActionItemInput{
		ID:        "t-role-builder",
		ProjectID: project.ID,
		ColumnID:  column.ID,
		Kind:      domain.KindBuild,
		Role:      domain.RoleBuilder,
		Position:  1,
		Title:     "builder item",
		Priority:  domain.PriorityMedium,
	}, now)
	if err != nil {
		t.Fatalf("NewActionItem(builder) error = %v", err)
	}
	if err := repo.CreateActionItem(ctx, builderItem); err != nil {
		t.Fatalf("CreateActionItem(builder) error = %v", err)
	}
	loadedBuilder, err := repo.GetActionItem(ctx, builderItem.ID)
	if err != nil {
		t.Fatalf("GetActionItem(builder) error = %v", err)
	}
	if loadedBuilder.Role != domain.RoleBuilder {
		t.Fatalf("expected persisted role %q, got %q", domain.RoleBuilder, loadedBuilder.Role)
	}

	// ListActionItems must also surface the role column (separate SELECT path).
	listed, err := repo.ListActionItems(ctx, project.ID, false)
	if err != nil {
		t.Fatalf("ListActionItems() error = %v", err)
	}
	var sawBuilder bool
	for _, item := range listed {
		if item.ID == builderItem.ID && item.Role == domain.RoleBuilder {
			sawBuilder = true
		}
	}
	if !sawBuilder {
		t.Fatalf("ListActionItems() did not surface builder role; got %#v", listed)
	}

	// Reassign role on update: confirms the UPDATE SET clause writes the new
	// value through. RoleQAProof is a different closed-enum value than the
	// initial RoleBuilder, so a successful round-trip proves the SET is wired.
	loadedBuilder.Role = domain.RoleQAProof
	loadedBuilder.UpdatedAt = now.Add(time.Hour)
	if err := repo.UpdateActionItem(ctx, loadedBuilder); err != nil {
		t.Fatalf("UpdateActionItem(role reassign) error = %v", err)
	}
	reloaded, err := repo.GetActionItem(ctx, builderItem.ID)
	if err != nil {
		t.Fatalf("GetActionItem(after update) error = %v", err)
	}
	if reloaded.Role != domain.RoleQAProof {
		t.Fatalf("expected reassigned role %q, got %q", domain.RoleQAProof, reloaded.Role)
	}
}

// TestRepository_PersistsActionItemStructuralTypeAndIrreducible verifies the
// structural_type and irreducible columns round-trip across create + get +
// list + update on an action item. All four StructuralType enum values are
// covered and the Irreducible flag is exercised in both true and false
// states. The test mirrors the TestRepository_PersistsActionItemRole shape so
// the SELECT/INSERT/UPDATE column-ordinal alignment for the two new columns
// is asserted on the same paths.
func TestRepository_PersistsActionItemStructuralTypeAndIrreducible(t *testing.T) {
	ctx := context.Background()
	repo, err := OpenInMemory()
	if err != nil {
		t.Fatalf("OpenInMemory() error = %v", err)
	}
	t.Cleanup(func() {
		_ = repo.Close()
	})

	now := time.Date(2026, 5, 2, 9, 0, 0, 0, time.UTC)
	project, _ := domain.NewProjectFromInput(domain.ProjectInput{ID: "p-st", Name: "StructuralType"}, now)
	if err := repo.CreateProject(ctx, project); err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}
	column, _ := domain.NewColumn("c-st", project.ID, "To Do", 0, 0, now)
	if err := repo.CreateColumn(ctx, column); err != nil {
		t.Fatalf("CreateColumn() error = %v", err)
	}

	// Round-trip every member of the closed StructuralType enum, paired with
	// distinct Irreducible flags so true and false both exercise the SQLite
	// INTEGER 0/1 conversion path.
	cases := []struct {
		id             string
		structuralType domain.StructuralType
		irreducible    bool
	}{
		{id: "t-st-drop", structuralType: domain.StructuralTypeDrop, irreducible: false},
		{id: "t-st-segment", structuralType: domain.StructuralTypeSegment, irreducible: false},
		{id: "t-st-confluence", structuralType: domain.StructuralTypeConfluence, irreducible: false},
		{id: "t-st-droplet", structuralType: domain.StructuralTypeDroplet, irreducible: true},
	}

	for i, tc := range cases {
		item, err := domain.NewActionItem(domain.ActionItemInput{
			ID:             tc.id,
			ProjectID:      project.ID,
			ColumnID:       column.ID,
			Kind:           domain.KindBuild,
			StructuralType: tc.structuralType,
			Irreducible:    tc.irreducible,
			Position:       i,
			Title:          "structural type " + string(tc.structuralType),
			Priority:       domain.PriorityMedium,
		}, now)
		if err != nil {
			t.Fatalf("NewActionItem(%s) error = %v", tc.structuralType, err)
		}
		if err := repo.CreateActionItem(ctx, item); err != nil {
			t.Fatalf("CreateActionItem(%s) error = %v", tc.structuralType, err)
		}
		loaded, err := repo.GetActionItem(ctx, item.ID)
		if err != nil {
			t.Fatalf("GetActionItem(%s) error = %v", tc.structuralType, err)
		}
		if loaded.StructuralType != tc.structuralType {
			t.Fatalf("expected persisted structural_type %q, got %q", tc.structuralType, loaded.StructuralType)
		}
		if loaded.Irreducible != tc.irreducible {
			t.Fatalf("expected persisted irreducible %v, got %v", tc.irreducible, loaded.Irreducible)
		}
	}

	// ListActionItems must surface both new columns (separate SELECT path).
	listed, err := repo.ListActionItems(ctx, project.ID, false)
	if err != nil {
		t.Fatalf("ListActionItems() error = %v", err)
	}
	if len(listed) != len(cases) {
		t.Fatalf("ListActionItems() length = %d, want %d", len(listed), len(cases))
	}
	byID := map[string]domain.ActionItem{}
	for _, item := range listed {
		byID[item.ID] = item
	}
	for _, tc := range cases {
		got, ok := byID[tc.id]
		if !ok {
			t.Fatalf("ListActionItems() missing %q", tc.id)
		}
		if got.StructuralType != tc.structuralType {
			t.Fatalf("ListActionItems()[%q].StructuralType = %q, want %q", tc.id, got.StructuralType, tc.structuralType)
		}
		if got.Irreducible != tc.irreducible {
			t.Fatalf("ListActionItems()[%q].Irreducible = %v, want %v", tc.id, got.Irreducible, tc.irreducible)
		}
	}

	// ListActionItemsByParent exercises the third SELECT path.
	parentListed, err := repo.ListActionItemsByParent(ctx, project.ID, "")
	if err != nil {
		t.Fatalf("ListActionItemsByParent() error = %v", err)
	}
	if len(parentListed) != len(cases) {
		t.Fatalf("ListActionItemsByParent() length = %d, want %d", len(parentListed), len(cases))
	}

	// Reassign on update: flip Irreducible and reassign StructuralType to a
	// different closed-enum value to confirm the UPDATE SET clause writes
	// both new columns through.
	target, err := repo.GetActionItem(ctx, "t-st-drop")
	if err != nil {
		t.Fatalf("GetActionItem(reassign source) error = %v", err)
	}
	target.StructuralType = domain.StructuralTypeConfluence
	target.Irreducible = true
	target.UpdatedAt = now.Add(time.Hour)
	if err := repo.UpdateActionItem(ctx, target); err != nil {
		t.Fatalf("UpdateActionItem(structural_type+irreducible) error = %v", err)
	}
	reloaded, err := repo.GetActionItem(ctx, target.ID)
	if err != nil {
		t.Fatalf("GetActionItem(after update) error = %v", err)
	}
	if reloaded.StructuralType != domain.StructuralTypeConfluence {
		t.Fatalf("expected reassigned structural_type %q, got %q", domain.StructuralTypeConfluence, reloaded.StructuralType)
	}
	if !reloaded.Irreducible {
		t.Fatalf("expected reassigned irreducible true, got false")
	}
}

// TestRepository_PersistsActionItemOwnerAndDropNumber verifies the owner,
// drop_number, persistent, and dev_gated columns round-trip across create +
// get + list + list-by-parent + update on an action item. Cases mix
// Owner=""/"STEWARD" with DropNumber=0/5 and exercise both true/false states
// of Persistent and DevGated so the SQLite INTEGER 0/1 conversion path is
// covered for both bool columns. The test mirrors
// TestRepository_PersistsActionItemStructuralTypeAndIrreducible so the
// SELECT/INSERT/UPDATE column-ordinal alignment for the four new columns is
// asserted on every storage path.
func TestRepository_PersistsActionItemOwnerAndDropNumber(t *testing.T) {
	ctx := context.Background()
	repo, err := OpenInMemory()
	if err != nil {
		t.Fatalf("OpenInMemory() error = %v", err)
	}
	t.Cleanup(func() {
		_ = repo.Close()
	})

	now := time.Date(2026, 5, 2, 9, 0, 0, 0, time.UTC)
	project, _ := domain.NewProjectFromInput(domain.ProjectInput{ID: "p-own", Name: "Owner"}, now)
	if err := repo.CreateProject(ctx, project); err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}
	column, _ := domain.NewColumn("c-own", project.ID, "To Do", 0, 0, now)
	if err := repo.CreateColumn(ctx, column); err != nil {
		t.Fatalf("CreateColumn() error = %v", err)
	}

	// Cover Owner=""/"STEWARD" × DropNumber=0/5 with both bools toggling so
	// every column writes/reads a non-default value at least once.
	cases := []struct {
		id         string
		owner      string
		dropNumber int
		persistent bool
		devGated   bool
	}{
		{id: "t-own-empty-zero", owner: "", dropNumber: 0, persistent: false, devGated: false},
		{id: "t-own-steward-five", owner: "STEWARD", dropNumber: 5, persistent: true, devGated: true},
		{id: "t-own-empty-five", owner: "", dropNumber: 5, persistent: true, devGated: false},
		{id: "t-own-steward-zero", owner: "STEWARD", dropNumber: 0, persistent: false, devGated: true},
	}

	for i, tc := range cases {
		item, err := domain.NewActionItem(domain.ActionItemInput{
			ID:             tc.id,
			ProjectID:      project.ID,
			ColumnID:       column.ID,
			Kind:           domain.KindBuild,
			StructuralType: domain.StructuralTypeDroplet,
			Owner:          tc.owner,
			DropNumber:     tc.dropNumber,
			Persistent:     tc.persistent,
			DevGated:       tc.devGated,
			Position:       i,
			Title:          "owner " + tc.owner + " drop " + tc.id,
			Priority:       domain.PriorityMedium,
		}, now)
		if err != nil {
			t.Fatalf("NewActionItem(%s) error = %v", tc.id, err)
		}
		if err := repo.CreateActionItem(ctx, item); err != nil {
			t.Fatalf("CreateActionItem(%s) error = %v", tc.id, err)
		}
		loaded, err := repo.GetActionItem(ctx, item.ID)
		if err != nil {
			t.Fatalf("GetActionItem(%s) error = %v", tc.id, err)
		}
		if loaded.Owner != tc.owner {
			t.Fatalf("expected persisted owner %q, got %q", tc.owner, loaded.Owner)
		}
		if loaded.DropNumber != tc.dropNumber {
			t.Fatalf("expected persisted drop_number %d, got %d", tc.dropNumber, loaded.DropNumber)
		}
		if loaded.Persistent != tc.persistent {
			t.Fatalf("expected persisted persistent %v, got %v", tc.persistent, loaded.Persistent)
		}
		if loaded.DevGated != tc.devGated {
			t.Fatalf("expected persisted dev_gated %v, got %v", tc.devGated, loaded.DevGated)
		}
	}

	// ListActionItems exercises the second SELECT path.
	listed, err := repo.ListActionItems(ctx, project.ID, false)
	if err != nil {
		t.Fatalf("ListActionItems() error = %v", err)
	}
	if len(listed) != len(cases) {
		t.Fatalf("ListActionItems() length = %d, want %d", len(listed), len(cases))
	}
	byID := map[string]domain.ActionItem{}
	for _, item := range listed {
		byID[item.ID] = item
	}
	for _, tc := range cases {
		got, ok := byID[tc.id]
		if !ok {
			t.Fatalf("ListActionItems() missing %q", tc.id)
		}
		if got.Owner != tc.owner {
			t.Fatalf("ListActionItems()[%q].Owner = %q, want %q", tc.id, got.Owner, tc.owner)
		}
		if got.DropNumber != tc.dropNumber {
			t.Fatalf("ListActionItems()[%q].DropNumber = %d, want %d", tc.id, got.DropNumber, tc.dropNumber)
		}
		if got.Persistent != tc.persistent {
			t.Fatalf("ListActionItems()[%q].Persistent = %v, want %v", tc.id, got.Persistent, tc.persistent)
		}
		if got.DevGated != tc.devGated {
			t.Fatalf("ListActionItems()[%q].DevGated = %v, want %v", tc.id, got.DevGated, tc.devGated)
		}
	}

	// ListActionItemsByParent exercises the third SELECT path.
	parentListed, err := repo.ListActionItemsByParent(ctx, project.ID, "")
	if err != nil {
		t.Fatalf("ListActionItemsByParent() error = %v", err)
	}
	if len(parentListed) != len(cases) {
		t.Fatalf("ListActionItemsByParent() length = %d, want %d", len(parentListed), len(cases))
	}
	byIDParent := map[string]domain.ActionItem{}
	for _, item := range parentListed {
		byIDParent[item.ID] = item
	}
	for _, tc := range cases {
		got, ok := byIDParent[tc.id]
		if !ok {
			t.Fatalf("ListActionItemsByParent() missing %q", tc.id)
		}
		if got.Owner != tc.owner || got.DropNumber != tc.dropNumber || got.Persistent != tc.persistent || got.DevGated != tc.devGated {
			t.Fatalf("ListActionItemsByParent()[%q] = {Owner:%q DropNumber:%d Persistent:%v DevGated:%v}, want {Owner:%q DropNumber:%d Persistent:%v DevGated:%v}",
				tc.id, got.Owner, got.DropNumber, got.Persistent, got.DevGated,
				tc.owner, tc.dropNumber, tc.persistent, tc.devGated)
		}
	}

	// Reassign on update: flip every new column to confirm the UPDATE SET
	// clause writes all four through.
	target, err := repo.GetActionItem(ctx, "t-own-empty-zero")
	if err != nil {
		t.Fatalf("GetActionItem(reassign source) error = %v", err)
	}
	target.Owner = "STEWARD"
	target.DropNumber = 7
	target.Persistent = true
	target.DevGated = true
	target.UpdatedAt = now.Add(time.Hour)
	if err := repo.UpdateActionItem(ctx, target); err != nil {
		t.Fatalf("UpdateActionItem(owner+drop_number+persistent+dev_gated) error = %v", err)
	}
	reloaded, err := repo.GetActionItem(ctx, target.ID)
	if err != nil {
		t.Fatalf("GetActionItem(after update) error = %v", err)
	}
	if reloaded.Owner != "STEWARD" {
		t.Fatalf("expected reassigned owner %q, got %q", "STEWARD", reloaded.Owner)
	}
	if reloaded.DropNumber != 7 {
		t.Fatalf("expected reassigned drop_number 7, got %d", reloaded.DropNumber)
	}
	if !reloaded.Persistent {
		t.Fatalf("expected reassigned persistent true, got false")
	}
	if !reloaded.DevGated {
		t.Fatalf("expected reassigned dev_gated true, got false")
	}
}

// TestRepository_PersistsActionItemPaths verifies the paths_json column
// added in Drop 4a droplet 4a.5 round-trips across create + get + list +
// list-by-parent + update on an action item. Cases cover empty / single /
// multi / repeated cases so the JSON encode/decode path exercises insertion-
// order preservation, the empty-slice "[]" default, and explicit-clear via
// update. The test mirrors TestRepository_PersistsActionItemOwnerAndDropNumber
// so the SELECT/INSERT/UPDATE column-ordinal alignment for the new column is
// asserted on every storage path.
func TestRepository_PersistsActionItemPaths(t *testing.T) {
	ctx := context.Background()
	repo, err := OpenInMemory()
	if err != nil {
		t.Fatalf("OpenInMemory() error = %v", err)
	}
	t.Cleanup(func() {
		_ = repo.Close()
	})

	now := time.Date(2026, 5, 3, 9, 0, 0, 0, time.UTC)
	project, _ := domain.NewProjectFromInput(domain.ProjectInput{ID: "p-paths", Name: "Paths"}, now)
	if err := repo.CreateProject(ctx, project); err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}
	column, _ := domain.NewColumn("c-paths", project.ID, "To Do", 0, 0, now)
	if err := repo.CreateColumn(ctx, column); err != nil {
		t.Fatalf("CreateColumn() error = %v", err)
	}

	cases := []struct {
		id    string
		paths []string
		want  []string
	}{
		{id: "t-paths-empty", paths: nil, want: nil},
		{id: "t-paths-single", paths: []string{"internal/domain/action_item.go"}, want: []string{"internal/domain/action_item.go"}},
		{id: "t-paths-multi", paths: []string{"a/b/c.go", "d/e/f.go"}, want: []string{"a/b/c.go", "d/e/f.go"}},
	}

	for i, tc := range cases {
		// Supply a covering Packages entry whenever Paths is populated so
		// the Drop 4a droplet 4a.6 coverage invariant doesn't reject the
		// constructor — this test exercises Paths persistence, not the
		// coverage rule.
		var pkgs []string
		if len(tc.paths) > 0 {
			pkgs = []string{"internal/domain"}
		}
		item, err := domain.NewActionItem(domain.ActionItemInput{
			ID:             tc.id,
			ProjectID:      project.ID,
			ColumnID:       column.ID,
			Kind:           domain.KindBuild,
			StructuralType: domain.StructuralTypeDroplet,
			Paths:          tc.paths,
			Packages:       pkgs,
			Position:       i,
			Title:          "paths " + tc.id,
			Priority:       domain.PriorityMedium,
		}, now)
		if err != nil {
			t.Fatalf("NewActionItem(%s) error = %v", tc.id, err)
		}
		if err := repo.CreateActionItem(ctx, item); err != nil {
			t.Fatalf("CreateActionItem(%s) error = %v", tc.id, err)
		}
		loaded, err := repo.GetActionItem(ctx, item.ID)
		if err != nil {
			t.Fatalf("GetActionItem(%s) error = %v", tc.id, err)
		}
		assertPathsEqual(t, "GetActionItem("+tc.id+")", loaded.Paths, tc.want)
	}

	// ListActionItems exercises the second SELECT path.
	listed, err := repo.ListActionItems(ctx, project.ID, false)
	if err != nil {
		t.Fatalf("ListActionItems() error = %v", err)
	}
	if len(listed) != len(cases) {
		t.Fatalf("ListActionItems() length = %d, want %d", len(listed), len(cases))
	}
	byID := map[string]domain.ActionItem{}
	for _, item := range listed {
		byID[item.ID] = item
	}
	for _, tc := range cases {
		got, ok := byID[tc.id]
		if !ok {
			t.Fatalf("ListActionItems() missing %q", tc.id)
		}
		assertPathsEqual(t, "ListActionItems()["+tc.id+"]", got.Paths, tc.want)
	}

	// ListActionItemsByParent exercises the third SELECT path.
	parentListed, err := repo.ListActionItemsByParent(ctx, project.ID, "")
	if err != nil {
		t.Fatalf("ListActionItemsByParent() error = %v", err)
	}
	byIDParent := map[string]domain.ActionItem{}
	for _, item := range parentListed {
		byIDParent[item.ID] = item
	}
	for _, tc := range cases {
		got, ok := byIDParent[tc.id]
		if !ok {
			t.Fatalf("ListActionItemsByParent() missing %q", tc.id)
		}
		assertPathsEqual(t, "ListActionItemsByParent()["+tc.id+"]", got.Paths, tc.want)
	}

	// Reassign on update: replace t-paths-empty's Paths with a populated
	// slice, then clear t-paths-multi's Paths back to nil. Verifies the
	// UPDATE SET clause writes both populated and empty payloads.
	target, err := repo.GetActionItem(ctx, "t-paths-empty")
	if err != nil {
		t.Fatalf("GetActionItem(reassign source) error = %v", err)
	}
	target.Paths = []string{"x/y.go", "z.go"}
	target.UpdatedAt = now.Add(time.Hour)
	if err := repo.UpdateActionItem(ctx, target); err != nil {
		t.Fatalf("UpdateActionItem(populate paths) error = %v", err)
	}
	reloaded, err := repo.GetActionItem(ctx, target.ID)
	if err != nil {
		t.Fatalf("GetActionItem(after populate) error = %v", err)
	}
	assertPathsEqual(t, "after populate", reloaded.Paths, []string{"x/y.go", "z.go"})

	clearTarget, err := repo.GetActionItem(ctx, "t-paths-multi")
	if err != nil {
		t.Fatalf("GetActionItem(clear source) error = %v", err)
	}
	clearTarget.Paths = nil
	clearTarget.UpdatedAt = now.Add(2 * time.Hour)
	if err := repo.UpdateActionItem(ctx, clearTarget); err != nil {
		t.Fatalf("UpdateActionItem(clear paths) error = %v", err)
	}
	clearReloaded, err := repo.GetActionItem(ctx, clearTarget.ID)
	if err != nil {
		t.Fatalf("GetActionItem(after clear) error = %v", err)
	}
	if len(clearReloaded.Paths) != 0 {
		t.Fatalf("after clear: Paths = %#v, want empty", clearReloaded.Paths)
	}
}

// assertPathsEqual fails the test when the actual and expected paths slices
// disagree on length or insertion order. nil and len-0 slices are treated as
// equal so callers may pass either form.
func assertPathsEqual(t *testing.T, label string, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%s: Paths length = %d (%#v), want %d (%#v)", label, len(got), got, len(want), want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("%s: Paths[%d] = %q, want %q (full = %#v)", label, i, got[i], want[i], got)
		}
	}
}

// TestRepository_PersistsActionItemPackages verifies the packages_json column
// added in Drop 4a droplet 4a.6 round-trips across create + get + list +
// list-by-parent + update on an action item. Cases cover empty / single /
// multi cases so the JSON encode/decode path exercises insertion-order
// preservation, the empty-slice "[]" default, and explicit-clear via update.
// Mirrors TestRepository_PersistsActionItemPaths so the SELECT/INSERT/UPDATE
// column-ordinal alignment for the new column is asserted on every storage
// path. Each populated test case supplies a covering Paths slice so the
// domain coverage invariant ("non-empty Paths requires non-empty Packages",
// applied in NewActionItem) doesn't reject the constructor — the empty
// case sets neither.
func TestRepository_PersistsActionItemPackages(t *testing.T) {
	ctx := context.Background()
	repo, err := OpenInMemory()
	if err != nil {
		t.Fatalf("OpenInMemory() error = %v", err)
	}
	t.Cleanup(func() {
		_ = repo.Close()
	})

	now := time.Date(2026, 5, 3, 9, 0, 0, 0, time.UTC)
	project, _ := domain.NewProjectFromInput(domain.ProjectInput{ID: "p-packages", Name: "Packages"}, now)
	if err := repo.CreateProject(ctx, project); err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}
	column, _ := domain.NewColumn("c-packages", project.ID, "To Do", 0, 0, now)
	if err := repo.CreateColumn(ctx, column); err != nil {
		t.Fatalf("CreateColumn() error = %v", err)
	}

	cases := []struct {
		id       string
		paths    []string
		packages []string
		want     []string
	}{
		{id: "t-pkgs-empty", paths: nil, packages: nil, want: nil},
		{id: "t-pkgs-single", paths: []string{"internal/domain/action_item.go"}, packages: []string{"internal/domain"}, want: []string{"internal/domain"}},
		{id: "t-pkgs-multi", paths: []string{"a/b.go", "c/d.go"}, packages: []string{"internal/domain", "internal/app"}, want: []string{"internal/domain", "internal/app"}},
	}

	for i, tc := range cases {
		item, err := domain.NewActionItem(domain.ActionItemInput{
			ID:             tc.id,
			ProjectID:      project.ID,
			ColumnID:       column.ID,
			Kind:           domain.KindBuild,
			StructuralType: domain.StructuralTypeDroplet,
			Paths:          tc.paths,
			Packages:       tc.packages,
			Position:       i,
			Title:          "packages " + tc.id,
			Priority:       domain.PriorityMedium,
		}, now)
		if err != nil {
			t.Fatalf("NewActionItem(%s) error = %v", tc.id, err)
		}
		if err := repo.CreateActionItem(ctx, item); err != nil {
			t.Fatalf("CreateActionItem(%s) error = %v", tc.id, err)
		}
		loaded, err := repo.GetActionItem(ctx, item.ID)
		if err != nil {
			t.Fatalf("GetActionItem(%s) error = %v", tc.id, err)
		}
		assertPackagesEqual(t, "GetActionItem("+tc.id+")", loaded.Packages, tc.want)
	}

	// ListActionItems exercises the second SELECT path.
	listed, err := repo.ListActionItems(ctx, project.ID, false)
	if err != nil {
		t.Fatalf("ListActionItems() error = %v", err)
	}
	if len(listed) != len(cases) {
		t.Fatalf("ListActionItems() length = %d, want %d", len(listed), len(cases))
	}
	byID := map[string]domain.ActionItem{}
	for _, item := range listed {
		byID[item.ID] = item
	}
	for _, tc := range cases {
		got, ok := byID[tc.id]
		if !ok {
			t.Fatalf("ListActionItems() missing %q", tc.id)
		}
		assertPackagesEqual(t, "ListActionItems()["+tc.id+"]", got.Packages, tc.want)
	}

	// ListActionItemsByParent exercises the third SELECT path.
	parentListed, err := repo.ListActionItemsByParent(ctx, project.ID, "")
	if err != nil {
		t.Fatalf("ListActionItemsByParent() error = %v", err)
	}
	byIDParent := map[string]domain.ActionItem{}
	for _, item := range parentListed {
		byIDParent[item.ID] = item
	}
	for _, tc := range cases {
		got, ok := byIDParent[tc.id]
		if !ok {
			t.Fatalf("ListActionItemsByParent() missing %q", tc.id)
		}
		assertPackagesEqual(t, "ListActionItemsByParent()["+tc.id+"]", got.Packages, tc.want)
	}

	// Reassign on update: replace t-pkgs-empty's Packages with a populated
	// slice, then clear t-pkgs-multi's Packages back to nil. Verifies the
	// UPDATE SET clause writes both populated and empty payloads. The repo
	// layer does not re-validate the domain coverage invariant on update
	// (UpdateActionItem writes raw fields), so clearing Packages while
	// Paths remains populated is permitted at the storage layer — domain
	// invariant enforcement happens in Service.UpdateActionItem.
	target, err := repo.GetActionItem(ctx, "t-pkgs-empty")
	if err != nil {
		t.Fatalf("GetActionItem(reassign source) error = %v", err)
	}
	target.Packages = []string{"github.com/foo/bar", "internal/app"}
	target.UpdatedAt = now.Add(time.Hour)
	if err := repo.UpdateActionItem(ctx, target); err != nil {
		t.Fatalf("UpdateActionItem(populate packages) error = %v", err)
	}
	reloaded, err := repo.GetActionItem(ctx, target.ID)
	if err != nil {
		t.Fatalf("GetActionItem(after populate) error = %v", err)
	}
	assertPackagesEqual(t, "after populate", reloaded.Packages, []string{"github.com/foo/bar", "internal/app"})

	clearTarget, err := repo.GetActionItem(ctx, "t-pkgs-multi")
	if err != nil {
		t.Fatalf("GetActionItem(clear source) error = %v", err)
	}
	clearTarget.Packages = nil
	clearTarget.UpdatedAt = now.Add(2 * time.Hour)
	if err := repo.UpdateActionItem(ctx, clearTarget); err != nil {
		t.Fatalf("UpdateActionItem(clear packages) error = %v", err)
	}
	clearReloaded, err := repo.GetActionItem(ctx, clearTarget.ID)
	if err != nil {
		t.Fatalf("GetActionItem(after clear) error = %v", err)
	}
	if len(clearReloaded.Packages) != 0 {
		t.Fatalf("after clear: Packages = %#v, want empty", clearReloaded.Packages)
	}
}

// assertPackagesEqual fails the test when the actual and expected packages
// slices disagree on length or insertion order. nil and len-0 slices are
// treated as equal so callers may pass either form.
func assertPackagesEqual(t *testing.T, label string, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%s: Packages length = %d (%#v), want %d (%#v)", label, len(got), got, len(want), want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("%s: Packages[%d] = %q, want %q (full = %#v)", label, i, got[i], want[i], got)
		}
	}
}

// TestRepository_PersistsActionItemFiles verifies the files_json column added
// in Drop 4a droplet 4a.7 round-trips across create + get + list +
// list-by-parent + update on an action item. Cases cover empty / single /
// multi inputs so the JSON encode/decode path exercises insertion-order
// preservation, the empty-slice "[]" default, and explicit-clear via
// update. Mirrors TestRepository_PersistsActionItemPaths /
// TestRepository_PersistsActionItemPackages so the SELECT/INSERT/UPDATE
// column-ordinal alignment for the new column is asserted on every storage
// path. Files is disjoint-axis with Paths so populated cases need NOT
// supply a covering Paths/Packages pair — Files alone is independent.
func TestRepository_PersistsActionItemFiles(t *testing.T) {
	ctx := context.Background()
	repo, err := OpenInMemory()
	if err != nil {
		t.Fatalf("OpenInMemory() error = %v", err)
	}
	t.Cleanup(func() {
		_ = repo.Close()
	})

	now := time.Date(2026, 5, 3, 12, 0, 0, 0, time.UTC)
	project, _ := domain.NewProjectFromInput(domain.ProjectInput{ID: "p-files", Name: "Files"}, now)
	if err := repo.CreateProject(ctx, project); err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}
	column, _ := domain.NewColumn("c-files", project.ID, "To Do", 0, 0, now)
	if err := repo.CreateColumn(ctx, column); err != nil {
		t.Fatalf("CreateColumn() error = %v", err)
	}

	cases := []struct {
		id    string
		files []string
		want  []string
	}{
		{id: "t-files-empty", files: nil, want: nil},
		{id: "t-files-single", files: []string{"docs/README.md"}, want: []string{"docs/README.md"}},
		{id: "t-files-multi", files: []string{"docs/A.md", "docs/B.md"}, want: []string{"docs/A.md", "docs/B.md"}},
	}

	for i, tc := range cases {
		item, err := domain.NewActionItem(domain.ActionItemInput{
			ID:             tc.id,
			ProjectID:      project.ID,
			ColumnID:       column.ID,
			Kind:           domain.KindBuild,
			StructuralType: domain.StructuralTypeDroplet,
			Files:          tc.files,
			Position:       i,
			Title:          "files " + tc.id,
			Priority:       domain.PriorityMedium,
		}, now)
		if err != nil {
			t.Fatalf("NewActionItem(%s) error = %v", tc.id, err)
		}
		if err := repo.CreateActionItem(ctx, item); err != nil {
			t.Fatalf("CreateActionItem(%s) error = %v", tc.id, err)
		}
		loaded, err := repo.GetActionItem(ctx, item.ID)
		if err != nil {
			t.Fatalf("GetActionItem(%s) error = %v", tc.id, err)
		}
		assertFilesEqual(t, "GetActionItem("+tc.id+")", loaded.Files, tc.want)
	}

	// ListActionItems exercises the second SELECT path.
	listed, err := repo.ListActionItems(ctx, project.ID, false)
	if err != nil {
		t.Fatalf("ListActionItems() error = %v", err)
	}
	if len(listed) != len(cases) {
		t.Fatalf("ListActionItems() length = %d, want %d", len(listed), len(cases))
	}
	byID := map[string]domain.ActionItem{}
	for _, item := range listed {
		byID[item.ID] = item
	}
	for _, tc := range cases {
		got, ok := byID[tc.id]
		if !ok {
			t.Fatalf("ListActionItems() missing %q", tc.id)
		}
		assertFilesEqual(t, "ListActionItems()["+tc.id+"]", got.Files, tc.want)
	}

	// ListActionItemsByParent exercises the third SELECT path.
	parentListed, err := repo.ListActionItemsByParent(ctx, project.ID, "")
	if err != nil {
		t.Fatalf("ListActionItemsByParent() error = %v", err)
	}
	byIDParent := map[string]domain.ActionItem{}
	for _, item := range parentListed {
		byIDParent[item.ID] = item
	}
	for _, tc := range cases {
		got, ok := byIDParent[tc.id]
		if !ok {
			t.Fatalf("ListActionItemsByParent() missing %q", tc.id)
		}
		assertFilesEqual(t, "ListActionItemsByParent()["+tc.id+"]", got.Files, tc.want)
	}

	// Reassign on update: replace t-files-empty's Files with a populated
	// slice, then clear t-files-multi's Files back to nil. Verifies the
	// UPDATE SET clause writes both populated and empty payloads.
	target, err := repo.GetActionItem(ctx, "t-files-empty")
	if err != nil {
		t.Fatalf("GetActionItem(reassign source) error = %v", err)
	}
	target.Files = []string{"docs/X.md", "docs/Y.md"}
	target.UpdatedAt = now.Add(time.Hour)
	if err := repo.UpdateActionItem(ctx, target); err != nil {
		t.Fatalf("UpdateActionItem(populate files) error = %v", err)
	}
	reloaded, err := repo.GetActionItem(ctx, target.ID)
	if err != nil {
		t.Fatalf("GetActionItem(after populate) error = %v", err)
	}
	assertFilesEqual(t, "after populate", reloaded.Files, []string{"docs/X.md", "docs/Y.md"})

	clearTarget, err := repo.GetActionItem(ctx, "t-files-multi")
	if err != nil {
		t.Fatalf("GetActionItem(clear source) error = %v", err)
	}
	clearTarget.Files = nil
	clearTarget.UpdatedAt = now.Add(2 * time.Hour)
	if err := repo.UpdateActionItem(ctx, clearTarget); err != nil {
		t.Fatalf("UpdateActionItem(clear files) error = %v", err)
	}
	clearReloaded, err := repo.GetActionItem(ctx, clearTarget.ID)
	if err != nil {
		t.Fatalf("GetActionItem(after clear) error = %v", err)
	}
	if len(clearReloaded.Files) != 0 {
		t.Fatalf("after clear: Files = %#v, want empty", clearReloaded.Files)
	}
}

// assertFilesEqual fails the test when the actual and expected files slices
// disagree on length or insertion order. nil and len-0 slices are treated
// as equal so callers may pass either form.
func assertFilesEqual(t *testing.T, label string, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%s: Files length = %d (%#v), want %d (%#v)", label, len(got), got, len(want), want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("%s: Files[%d] = %q, want %q (full = %#v)", label, i, got[i], want[i], got)
		}
	}
}

// TestRepository_PersistsActionItemStartCommit verifies the start_commit TEXT
// column added in Drop 4a droplet 4a.8 round-trips across create + get +
// list + list-by-parent + update on an action item. Cases cover empty /
// short-SHA / full-SHA inputs so the raw-string storage path exercises the
// empty zero-value default and explicit-clear via update. Mirrors
// TestRepository_PersistsActionItemFiles so the SELECT/INSERT/UPDATE
// column-ordinal alignment for the new column is asserted on every storage
// path. StartCommit is stored as a raw TEXT column (not JSON-encoded —
// single value, not a list) with NOT NULL DEFAULT ” so legacy rows read
// as the legitimate "not yet captured" zero value.
func TestRepository_PersistsActionItemStartCommit(t *testing.T) {
	ctx := context.Background()
	repo, err := OpenInMemory()
	if err != nil {
		t.Fatalf("OpenInMemory() error = %v", err)
	}
	t.Cleanup(func() {
		_ = repo.Close()
	})

	now := time.Date(2026, 5, 3, 12, 0, 0, 0, time.UTC)
	project, _ := domain.NewProjectFromInput(domain.ProjectInput{ID: "p-startcommit", Name: "StartCommit"}, now)
	if err := repo.CreateProject(ctx, project); err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}
	column, _ := domain.NewColumn("c-startcommit", project.ID, "To Do", 0, 0, now)
	if err := repo.CreateColumn(ctx, column); err != nil {
		t.Fatalf("CreateColumn() error = %v", err)
	}

	cases := []struct {
		id          string
		startCommit string
		want        string
	}{
		{id: "t-startcommit-empty", startCommit: "", want: ""},
		{id: "t-startcommit-short", startCommit: "0cf5194", want: "0cf5194"},
		{id: "t-startcommit-full", startCommit: "0cf5194d4cb6c8d4f9b9b1d7e1f9d3c2b4e5a6f7", want: "0cf5194d4cb6c8d4f9b9b1d7e1f9d3c2b4e5a6f7"},
	}

	for i, tc := range cases {
		item, err := domain.NewActionItem(domain.ActionItemInput{
			ID:             tc.id,
			ProjectID:      project.ID,
			ColumnID:       column.ID,
			Kind:           domain.KindBuild,
			StructuralType: domain.StructuralTypeDroplet,
			StartCommit:    tc.startCommit,
			Position:       i,
			Title:          "startcommit " + tc.id,
			Priority:       domain.PriorityMedium,
		}, now)
		if err != nil {
			t.Fatalf("NewActionItem(%s) error = %v", tc.id, err)
		}
		if err := repo.CreateActionItem(ctx, item); err != nil {
			t.Fatalf("CreateActionItem(%s) error = %v", tc.id, err)
		}
		loaded, err := repo.GetActionItem(ctx, item.ID)
		if err != nil {
			t.Fatalf("GetActionItem(%s) error = %v", tc.id, err)
		}
		if loaded.StartCommit != tc.want {
			t.Fatalf("GetActionItem(%s) StartCommit = %q, want %q", tc.id, loaded.StartCommit, tc.want)
		}
	}

	// ListActionItems exercises the second SELECT path.
	listed, err := repo.ListActionItems(ctx, project.ID, false)
	if err != nil {
		t.Fatalf("ListActionItems() error = %v", err)
	}
	if len(listed) != len(cases) {
		t.Fatalf("ListActionItems() length = %d, want %d", len(listed), len(cases))
	}
	byID := map[string]domain.ActionItem{}
	for _, item := range listed {
		byID[item.ID] = item
	}
	for _, tc := range cases {
		got, ok := byID[tc.id]
		if !ok {
			t.Fatalf("ListActionItems() missing %q", tc.id)
		}
		if got.StartCommit != tc.want {
			t.Fatalf("ListActionItems()[%s] StartCommit = %q, want %q", tc.id, got.StartCommit, tc.want)
		}
	}

	// ListActionItemsByParent exercises the third SELECT path.
	parentListed, err := repo.ListActionItemsByParent(ctx, project.ID, "")
	if err != nil {
		t.Fatalf("ListActionItemsByParent() error = %v", err)
	}
	byIDParent := map[string]domain.ActionItem{}
	for _, item := range parentListed {
		byIDParent[item.ID] = item
	}
	for _, tc := range cases {
		got, ok := byIDParent[tc.id]
		if !ok {
			t.Fatalf("ListActionItemsByParent() missing %q", tc.id)
		}
		if got.StartCommit != tc.want {
			t.Fatalf("ListActionItemsByParent()[%s] StartCommit = %q, want %q", tc.id, got.StartCommit, tc.want)
		}
	}

	// Reassign on update: replace t-startcommit-empty's StartCommit with a
	// populated value, then clear t-startcommit-full's StartCommit back to
	// empty. Verifies the UPDATE SET clause writes both populated and
	// empty payloads.
	target, err := repo.GetActionItem(ctx, "t-startcommit-empty")
	if err != nil {
		t.Fatalf("GetActionItem(reassign source) error = %v", err)
	}
	target.StartCommit = "deadbeef"
	target.UpdatedAt = now.Add(time.Hour)
	if err := repo.UpdateActionItem(ctx, target); err != nil {
		t.Fatalf("UpdateActionItem(populate startcommit) error = %v", err)
	}
	reloaded, err := repo.GetActionItem(ctx, target.ID)
	if err != nil {
		t.Fatalf("GetActionItem(after populate) error = %v", err)
	}
	if reloaded.StartCommit != "deadbeef" {
		t.Fatalf("after populate: StartCommit = %q, want %q", reloaded.StartCommit, "deadbeef")
	}

	clearTarget, err := repo.GetActionItem(ctx, "t-startcommit-full")
	if err != nil {
		t.Fatalf("GetActionItem(clear source) error = %v", err)
	}
	clearTarget.StartCommit = ""
	clearTarget.UpdatedAt = now.Add(2 * time.Hour)
	if err := repo.UpdateActionItem(ctx, clearTarget); err != nil {
		t.Fatalf("UpdateActionItem(clear startcommit) error = %v", err)
	}
	clearReloaded, err := repo.GetActionItem(ctx, clearTarget.ID)
	if err != nil {
		t.Fatalf("GetActionItem(after clear) error = %v", err)
	}
	if clearReloaded.StartCommit != "" {
		t.Fatalf("after clear: StartCommit = %q, want empty", clearReloaded.StartCommit)
	}
}

// TestRepository_PersistsActionItemEndCommit verifies the end_commit TEXT
// column added in Drop 4a droplet 4a.9 round-trips across create + get +
// list + list-by-parent + update on an action item. Cases cover empty /
// short-SHA / full-SHA inputs so the raw-string storage path exercises the
// empty zero-value default and explicit-clear via update. Mirrors
// TestRepository_PersistsActionItemStartCommit verbatim adapted to the
// EndCommit field. EndCommit is stored as a raw TEXT column (not JSON-
// encoded — single value, not a list) with NOT NULL DEFAULT ” so legacy
// rows read as the legitimate "not yet captured" zero value.
func TestRepository_PersistsActionItemEndCommit(t *testing.T) {
	ctx := context.Background()
	repo, err := OpenInMemory()
	if err != nil {
		t.Fatalf("OpenInMemory() error = %v", err)
	}
	t.Cleanup(func() {
		_ = repo.Close()
	})

	now := time.Date(2026, 5, 3, 12, 0, 0, 0, time.UTC)
	project, _ := domain.NewProjectFromInput(domain.ProjectInput{ID: "p-endcommit", Name: "EndCommit"}, now)
	if err := repo.CreateProject(ctx, project); err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}
	column, _ := domain.NewColumn("c-endcommit", project.ID, "To Do", 0, 0, now)
	if err := repo.CreateColumn(ctx, column); err != nil {
		t.Fatalf("CreateColumn() error = %v", err)
	}

	cases := []struct {
		id        string
		endCommit string
		want      string
	}{
		{id: "t-endcommit-empty", endCommit: "", want: ""},
		{id: "t-endcommit-short", endCommit: "0cf5194", want: "0cf5194"},
		{id: "t-endcommit-full", endCommit: "0cf5194d4cb6c8d4f9b9b1d7e1f9d3c2b4e5a6f7", want: "0cf5194d4cb6c8d4f9b9b1d7e1f9d3c2b4e5a6f7"},
	}

	for i, tc := range cases {
		item, err := domain.NewActionItem(domain.ActionItemInput{
			ID:             tc.id,
			ProjectID:      project.ID,
			ColumnID:       column.ID,
			Kind:           domain.KindBuild,
			StructuralType: domain.StructuralTypeDroplet,
			EndCommit:      tc.endCommit,
			Position:       i,
			Title:          "endcommit " + tc.id,
			Priority:       domain.PriorityMedium,
		}, now)
		if err != nil {
			t.Fatalf("NewActionItem(%s) error = %v", tc.id, err)
		}
		if err := repo.CreateActionItem(ctx, item); err != nil {
			t.Fatalf("CreateActionItem(%s) error = %v", tc.id, err)
		}
		loaded, err := repo.GetActionItem(ctx, item.ID)
		if err != nil {
			t.Fatalf("GetActionItem(%s) error = %v", tc.id, err)
		}
		if loaded.EndCommit != tc.want {
			t.Fatalf("GetActionItem(%s) EndCommit = %q, want %q", tc.id, loaded.EndCommit, tc.want)
		}
	}

	// ListActionItems exercises the second SELECT path.
	listed, err := repo.ListActionItems(ctx, project.ID, false)
	if err != nil {
		t.Fatalf("ListActionItems() error = %v", err)
	}
	if len(listed) != len(cases) {
		t.Fatalf("ListActionItems() length = %d, want %d", len(listed), len(cases))
	}
	byID := map[string]domain.ActionItem{}
	for _, item := range listed {
		byID[item.ID] = item
	}
	for _, tc := range cases {
		got, ok := byID[tc.id]
		if !ok {
			t.Fatalf("ListActionItems() missing %q", tc.id)
		}
		if got.EndCommit != tc.want {
			t.Fatalf("ListActionItems()[%s] EndCommit = %q, want %q", tc.id, got.EndCommit, tc.want)
		}
	}

	// ListActionItemsByParent exercises the third SELECT path.
	parentListed, err := repo.ListActionItemsByParent(ctx, project.ID, "")
	if err != nil {
		t.Fatalf("ListActionItemsByParent() error = %v", err)
	}
	byIDParent := map[string]domain.ActionItem{}
	for _, item := range parentListed {
		byIDParent[item.ID] = item
	}
	for _, tc := range cases {
		got, ok := byIDParent[tc.id]
		if !ok {
			t.Fatalf("ListActionItemsByParent() missing %q", tc.id)
		}
		if got.EndCommit != tc.want {
			t.Fatalf("ListActionItemsByParent()[%s] EndCommit = %q, want %q", tc.id, got.EndCommit, tc.want)
		}
	}

	// Reassign on update: replace t-endcommit-empty's EndCommit with a
	// populated value, then clear t-endcommit-full's EndCommit back to
	// empty. Verifies the UPDATE SET clause writes both populated and
	// empty payloads.
	target, err := repo.GetActionItem(ctx, "t-endcommit-empty")
	if err != nil {
		t.Fatalf("GetActionItem(reassign source) error = %v", err)
	}
	target.EndCommit = "deadbeef"
	target.UpdatedAt = now.Add(time.Hour)
	if err := repo.UpdateActionItem(ctx, target); err != nil {
		t.Fatalf("UpdateActionItem(populate endcommit) error = %v", err)
	}
	reloaded, err := repo.GetActionItem(ctx, target.ID)
	if err != nil {
		t.Fatalf("GetActionItem(after populate) error = %v", err)
	}
	if reloaded.EndCommit != "deadbeef" {
		t.Fatalf("after populate: EndCommit = %q, want %q", reloaded.EndCommit, "deadbeef")
	}

	clearTarget, err := repo.GetActionItem(ctx, "t-endcommit-full")
	if err != nil {
		t.Fatalf("GetActionItem(clear source) error = %v", err)
	}
	clearTarget.EndCommit = ""
	clearTarget.UpdatedAt = now.Add(2 * time.Hour)
	if err := repo.UpdateActionItem(ctx, clearTarget); err != nil {
		t.Fatalf("UpdateActionItem(clear endcommit) error = %v", err)
	}
	clearReloaded, err := repo.GetActionItem(ctx, clearTarget.ID)
	if err != nil {
		t.Fatalf("GetActionItem(after clear) error = %v", err)
	}
	if clearReloaded.EndCommit != "" {
		t.Fatalf("after clear: EndCommit = %q, want empty", clearReloaded.EndCommit)
	}
}

// TestRepository_IndexCoversDropNumberQuery sanity-checks that querying by
// (project_id, drop_number) returns the expected rows after the 3.18 schema
// change. The query shape mirrors what the 3.20 auto-generator will issue;
// the assertion is correctness rather than EXPLAIN-plan validation, since
// SQLite EXPLAIN output is brittle across versions.
func TestRepository_IndexCoversDropNumberQuery(t *testing.T) {
	ctx := context.Background()
	repo, err := OpenInMemory()
	if err != nil {
		t.Fatalf("OpenInMemory() error = %v", err)
	}
	t.Cleanup(func() {
		_ = repo.Close()
	})

	now := time.Date(2026, 5, 2, 9, 0, 0, 0, time.UTC)
	project, _ := domain.NewProjectFromInput(domain.ProjectInput{ID: "p-idx", Name: "Index"}, now)
	if err := repo.CreateProject(ctx, project); err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}
	column, _ := domain.NewColumn("c-idx", project.ID, "To Do", 0, 0, now)
	if err := repo.CreateColumn(ctx, column); err != nil {
		t.Fatalf("CreateColumn() error = %v", err)
	}

	// Three rows in drop 3, two rows in drop 4, one row in drop 0.
	rows := []struct {
		id         string
		dropNumber int
	}{
		{id: "t-d3-a", dropNumber: 3},
		{id: "t-d3-b", dropNumber: 3},
		{id: "t-d3-c", dropNumber: 3},
		{id: "t-d4-a", dropNumber: 4},
		{id: "t-d4-b", dropNumber: 4},
		{id: "t-d0-a", dropNumber: 0},
	}
	for i, row := range rows {
		item, err := domain.NewActionItem(domain.ActionItemInput{
			ID:             row.id,
			ProjectID:      project.ID,
			ColumnID:       column.ID,
			Kind:           domain.KindBuild,
			StructuralType: domain.StructuralTypeDroplet,
			DropNumber:     row.dropNumber,
			Position:       i,
			Title:          row.id,
			Priority:       domain.PriorityMedium,
		}, now)
		if err != nil {
			t.Fatalf("NewActionItem(%s) error = %v", row.id, err)
		}
		if err := repo.CreateActionItem(ctx, item); err != nil {
			t.Fatalf("CreateActionItem(%s) error = %v", row.id, err)
		}
	}

	// Query: WHERE project_id = ? AND drop_number = ? — index prefix scan
	// covered by idx_action_items_drop_number(project_id, drop_number, owner).
	queryRows, err := repo.db.QueryContext(ctx, `SELECT id FROM action_items WHERE project_id = ? AND drop_number = ? ORDER BY id ASC`, project.ID, 3)
	if err != nil {
		t.Fatalf("query drop_number error = %v", err)
	}
	defer queryRows.Close()
	var ids []string
	for queryRows.Next() {
		var id string
		if err := queryRows.Scan(&id); err != nil {
			t.Fatalf("scan id error = %v", err)
		}
		ids = append(ids, id)
	}
	if err := queryRows.Err(); err != nil {
		t.Fatalf("rows.Err() = %v", err)
	}
	want := []string{"t-d3-a", "t-d3-b", "t-d3-c"}
	if len(ids) != len(want) {
		t.Fatalf("query returned %d rows, want %d (got %v)", len(ids), len(want), ids)
	}
	for i := range want {
		if ids[i] != want[i] {
			t.Fatalf("row %d = %q, want %q", i, ids[i], want[i])
		}
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
	project, err := domain.NewProjectFromInput(domain.ProjectInput{ID: "p1", Name: "Project One"}, now)
	if err != nil {
		t.Fatalf("NewProjectFromInput() error = %v", err)
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
	project, err := domain.NewProjectFromInput(domain.ProjectInput{ID: "p1", Name: "Project One"}, now)
	if err != nil {
		t.Fatalf("NewProjectFromInput() error = %v", err)
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

// Per Drop 3 droplet 3.15 (finding 5.B.8 / CE3) the legacy kind_catalog
// boot-seed regression tests TestRepositoryFreshOpenKindCatalog and
// TestRepositoryFreshOpenKindCatalogUniversalParentAllow were retired
// because the kind_catalog table is no longer boot-seeded with the closed
// 12-value Kind enum. Equivalent universal-allow assertions live in
// internal/templates/embed_test.go (3.14 catalog), where the post-Drop-3
// nesting rules are exercised against the new templates.KindCatalog.

// TestRepositoryFreshOpenProjectsSchema verifies that a fresh DB open produces a projects table with no kind column.
func TestRepositoryFreshOpenProjectsSchema(t *testing.T) {
	ctx := context.Background()
	repo, err := OpenInMemory()
	if err != nil {
		t.Fatalf("OpenInMemory() error = %v", err)
	}
	t.Cleanup(func() {
		_ = repo.Close()
	})

	rows, err := repo.db.QueryContext(ctx, `SELECT name FROM pragma_table_info('projects')`)
	if err != nil {
		t.Fatalf("query pragma_table_info error = %v", err)
	}
	defer rows.Close()

	var columns []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatalf("scan pragma_table_info error = %v", err)
		}
		columns = append(columns, name)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate pragma_table_info error = %v", err)
	}

	for _, c := range columns {
		if c == "kind" {
			t.Fatalf("projects table still contains 'kind' column (all columns: %v)", columns)
		}
	}
	if len(columns) == 0 {
		t.Fatalf("pragma_table_info('projects') returned 0 columns — table missing?")
	}
}

// TestRepository_ListActionItemsByParent verifies the parent-scoped listing
// used by the dotted-address resolver. The test asserts (a) empty parentID
// returns level-1 children only, (b) explicit parentID returns that parent's
// direct children only (not grandchildren), (c) ordering is created_at ASC
// with id ASC tie-breaker, (d) project isolation filters out same-parent rows
// from a different project.
func TestRepository_ListActionItemsByParent(t *testing.T) {
	ctx := context.Background()
	repo, err := OpenInMemory()
	if err != nil {
		t.Fatalf("OpenInMemory() error = %v", err)
	}
	t.Cleanup(func() {
		_ = repo.Close()
	})

	base := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)

	projectA, err := domain.NewProjectFromInput(domain.ProjectInput{ID: "proj-a", Name: "Project A"}, base)
	if err != nil {
		t.Fatalf("NewProjectFromInput(A) error = %v", err)
	}
	if err := repo.CreateProject(ctx, projectA); err != nil {
		t.Fatalf("CreateProject(A) error = %v", err)
	}
	columnA, err := domain.NewColumn("col-a", projectA.ID, "Todo", 0, 0, base)
	if err != nil {
		t.Fatalf("NewColumn(A) error = %v", err)
	}
	if err := repo.CreateColumn(ctx, columnA); err != nil {
		t.Fatalf("CreateColumn(A) error = %v", err)
	}

	projectB, err := domain.NewProjectFromInput(domain.ProjectInput{ID: "proj-b", Name: "Project B"}, base)
	if err != nil {
		t.Fatalf("NewProjectFromInput(B) error = %v", err)
	}
	if err := repo.CreateProject(ctx, projectB); err != nil {
		t.Fatalf("CreateProject(B) error = %v", err)
	}
	columnB, err := domain.NewColumn("col-b", projectB.ID, "Todo", 0, 0, base)
	if err != nil {
		t.Fatalf("NewColumn(B) error = %v", err)
	}
	if err := repo.CreateColumn(ctx, columnB); err != nil {
		t.Fatalf("CreateColumn(B) error = %v", err)
	}

	// Project A tree:
	//   level-1: a-root-0 (t=+1s), a-root-1 (t=+2s).
	//   parent a-root-1: a-tie-aaa (t=+10s) and a-tie-zzz (t=+10s) — same
	//     CreatedAt; id ASC tie-break selects a-tie-aaa < a-tie-zzz first.
	//   parent a-root-0: a-leaf (t=+5s).
	type spec struct {
		id        string
		projectID string
		columnID  string
		parentID  string
		title     string
		createdAt time.Time
	}
	specs := []spec{
		{id: "a-root-0", projectID: projectA.ID, columnID: columnA.ID, parentID: "", title: "A root 0", createdAt: base.Add(1 * time.Second)},
		{id: "a-root-1", projectID: projectA.ID, columnID: columnA.ID, parentID: "", title: "A root 1", createdAt: base.Add(2 * time.Second)},
		{id: "a-leaf", projectID: projectA.ID, columnID: columnA.ID, parentID: "a-root-0", title: "A leaf", createdAt: base.Add(5 * time.Second)},
		{id: "a-tie-zzz", projectID: projectA.ID, columnID: columnA.ID, parentID: "a-root-1", title: "A child zzz", createdAt: base.Add(10 * time.Second)},
		{id: "a-tie-aaa", projectID: projectA.ID, columnID: columnA.ID, parentID: "a-root-1", title: "A child aaa", createdAt: base.Add(10 * time.Second)},
		// Project B has its OWN action items at parent_id "a-root-1" — same
		// parent_id string but different project_id; the listing must NOT
		// surface these when projectA.ID is supplied.
		{id: "b-cross", projectID: projectB.ID, columnID: columnB.ID, parentID: "a-root-1", title: "Cross-project leak guard", createdAt: base.Add(11 * time.Second)},
		{id: "b-root", projectID: projectB.ID, columnID: columnB.ID, parentID: "", title: "B root", createdAt: base.Add(3 * time.Second)},
	}

	for _, s := range specs {
		item, err := domain.NewActionItemForTest(domain.ActionItemInput{
			ID:        s.id,
			ProjectID: s.projectID,
			ParentID:  s.parentID,
			Kind:      domain.KindPlan,
			ColumnID:  s.columnID,
			Title:     s.title,
		}, s.createdAt)
		if err != nil {
			t.Fatalf("NewActionItem(%q) error = %v", s.id, err)
		}
		if err := repo.CreateActionItem(ctx, item); err != nil {
			t.Fatalf("CreateActionItem(%q) error = %v", s.id, err)
		}
	}

	// Empty parentID returns level-1 children only — and only for projectA.
	rootsA, err := repo.ListActionItemsByParent(ctx, projectA.ID, "")
	if err != nil {
		t.Fatalf("ListActionItemsByParent(A, \"\") error = %v", err)
	}
	gotIDs := make([]string, 0, len(rootsA))
	for _, item := range rootsA {
		gotIDs = append(gotIDs, item.ID)
	}
	wantRoots := []string{"a-root-0", "a-root-1"}
	if len(gotIDs) != len(wantRoots) {
		t.Fatalf("ListActionItemsByParent(A, \"\") len = %d (%v), want %d (%v)", len(gotIDs), gotIDs, len(wantRoots), wantRoots)
	}
	for i, want := range wantRoots {
		if gotIDs[i] != want {
			t.Fatalf("ListActionItemsByParent(A, \"\")[%d] = %q, want %q (full = %v)", i, gotIDs[i], want, gotIDs)
		}
	}

	// Explicit parent returns direct children only (no grandchildren), and
	// asserts the same-CreatedAt UUID tie-breaker: a-tie-aaa < a-tie-zzz
	// lexicographically, so a-tie-aaa MUST land at index 0.
	tieKids, err := repo.ListActionItemsByParent(ctx, projectA.ID, "a-root-1")
	if err != nil {
		t.Fatalf("ListActionItemsByParent(A, a-root-1) error = %v", err)
	}
	tieIDs := make([]string, 0, len(tieKids))
	for _, item := range tieKids {
		tieIDs = append(tieIDs, item.ID)
	}
	wantTie := []string{"a-tie-aaa", "a-tie-zzz"}
	if len(tieIDs) != len(wantTie) {
		t.Fatalf("ListActionItemsByParent(A, a-root-1) len = %d (%v), want %d (%v)", len(tieIDs), tieIDs, len(wantTie), wantTie)
	}
	for i, want := range wantTie {
		if tieIDs[i] != want {
			t.Fatalf("tie-break ordering[%d] = %q, want %q (full = %v)", i, tieIDs[i], want, tieIDs)
		}
	}

	// Project isolation: projectB's roots do NOT bleed into projectA's listing.
	rootsB, err := repo.ListActionItemsByParent(ctx, projectB.ID, "")
	if err != nil {
		t.Fatalf("ListActionItemsByParent(B, \"\") error = %v", err)
	}
	if len(rootsB) != 1 || rootsB[0].ID != "b-root" {
		t.Fatalf("ListActionItemsByParent(B, \"\") = %#v, want [b-root]", rootsB)
	}

	// Cross-project parent_id collision: projectA listing for parent a-root-1
	// must NOT include projectB's "b-cross" row even though their parent_id
	// strings match.
	for _, item := range tieKids {
		if item.ID == "b-cross" {
			t.Fatalf("ListActionItemsByParent(A, a-root-1) leaked projectB item %q", item.ID)
		}
	}

	// Empty result for a parent with no children.
	none, err := repo.ListActionItemsByParent(ctx, projectA.ID, "a-leaf")
	if err != nil {
		t.Fatalf("ListActionItemsByParent(A, a-leaf) error = %v", err)
	}
	if len(none) != 0 {
		t.Fatalf("ListActionItemsByParent(A, a-leaf) = %d rows, want 0 (%v)", len(none), none)
	}

	// Empty result for an unknown parent (no matching parent_id at all).
	nope, err := repo.ListActionItemsByParent(ctx, projectA.ID, "does-not-exist")
	if err != nil {
		t.Fatalf("ListActionItemsByParent(A, does-not-exist) error = %v", err)
	}
	if len(nope) != 0 {
		t.Fatalf("ListActionItemsByParent(A, does-not-exist) = %d rows, want 0 (%v)", len(nope), nope)
	}
}

// TestRepository_GetProjectBySlug verifies the slug-indexed project lookup
// added in Droplet 2.11. The lookup must (a) return the matching project for
// a known slug, (b) return ErrNoRows for an unknown slug, and (c) refuse to
// surface the hidden global-auth project even if its slug were supplied.
func TestRepository_GetProjectBySlug(t *testing.T) {
	ctx := context.Background()
	repo, err := OpenInMemory()
	if err != nil {
		t.Fatalf("OpenInMemory() error = %v", err)
	}
	t.Cleanup(func() {
		_ = repo.Close()
	})

	base := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	project, err := domain.NewProjectFromInput(domain.ProjectInput{ID: "proj-slug-test", Name: "Slug Test"}, base)
	if err != nil {
		t.Fatalf("NewProjectFromInput() error = %v", err)
	}
	project.Slug = "tillsyn-slug-fixture"
	if err := repo.CreateProject(ctx, project); err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}

	got, err := repo.GetProjectBySlug(ctx, "tillsyn-slug-fixture")
	if err != nil {
		t.Fatalf("GetProjectBySlug(known slug) error = %v", err)
	}
	if got.ID != project.ID {
		t.Fatalf("GetProjectBySlug(known slug) ID = %q, want %q", got.ID, project.ID)
	}
	if got.Slug != "tillsyn-slug-fixture" {
		t.Fatalf("GetProjectBySlug(known slug) Slug = %q, want %q", got.Slug, "tillsyn-slug-fixture")
	}

	if _, err := repo.GetProjectBySlug(ctx, "does-not-exist"); err == nil {
		t.Fatal("GetProjectBySlug(unknown slug) expected error, got nil")
	}

	// The hidden internal-auth project carries `globalAuthProjectSlug` and
	// must NOT be reachable through this surface even if a caller stumbles
	// onto the value.
	if _, err := repo.GetProjectBySlug(ctx, globalAuthProjectSlug); err == nil {
		t.Fatal("GetProjectBySlug(globalAuthProjectSlug) expected error, got nil")
	}
}

// floatPtr is a test-only convenience for creating *float64 literals used in
// the Drop 4c F.7.9 spawn-metadata round-trip tests below.
func floatPtr(v float64) *float64 { return &v }

// TestRepository_PersistsActionItemSpawnMetadata verifies the Drop 4c F.7.9
// spawn-metadata fields (SpawnBundlePath, SpawnHistory, ActualCostUSD)
// persist through SQLite via the JSON-encoded ActionItemMetadata blob and
// read back deep-equal — including the edge cases of (a) zero values
// (Drop-4a-era items with no spawn metadata round-trip without crashing
// the JSON decoder), (b) a populated bundle path with empty history (just-
// dispatched, never reached terminal), and (c) full populated metadata
// with mixed-cost history entries (cost reported on first spawn, omitted
// on second). REV-6: JSON-blob persistence — no new SQLite columns.
func TestRepository_PersistsActionItemSpawnMetadata(t *testing.T) {
	ctx := context.Background()
	repo, err := OpenInMemory()
	if err != nil {
		t.Fatalf("OpenInMemory() error = %v", err)
	}
	t.Cleanup(func() {
		_ = repo.Close()
	})

	now := time.Date(2026, 5, 4, 10, 0, 0, 0, time.UTC)
	project, _ := domain.NewProjectFromInput(domain.ProjectInput{ID: "p-spawnmeta", Name: "SpawnMetadata"}, now)
	if err := repo.CreateProject(ctx, project); err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}
	column, _ := domain.NewColumn("c-spawnmeta", project.ID, "To Do", 0, 0, now)
	if err := repo.CreateColumn(ctx, column); err != nil {
		t.Fatalf("CreateColumn() error = %v", err)
	}

	startedAt := time.Date(2026, 5, 1, 10, 0, 0, 0, time.UTC)
	terminatedAt := startedAt.Add(5 * time.Minute)

	cases := []struct {
		id   string
		meta domain.ActionItemMetadata
	}{
		{
			id:   "ai-empty-spawnmeta",
			meta: domain.ActionItemMetadata{},
		},
		{
			id: "ai-bundle-no-history",
			meta: domain.ActionItemMetadata{
				SpawnBundlePath: "/tmp/tillsyn/spawn-active/",
			},
		},
		{
			id: "ai-full-spawnmeta",
			meta: domain.ActionItemMetadata{
				SpawnBundlePath: "/tmp/tillsyn/spawn-current/",
				SpawnHistory: []domain.SpawnHistoryEntry{
					{
						SpawnID:      "spawn-1",
						BundlePath:   "/tmp/tillsyn/spawn-1/",
						StartedAt:    startedAt,
						TerminatedAt: terminatedAt,
						Outcome:      "success",
						TotalCostUSD: floatPtr(0.42),
					},
					{
						SpawnID:      "spawn-2",
						BundlePath:   "/tmp/tillsyn/spawn-2/",
						StartedAt:    startedAt.Add(time.Hour),
						TerminatedAt: terminatedAt.Add(time.Hour),
						Outcome:      "failure",
						TotalCostUSD: nil, // Edge: cost not reported.
					},
				},
				ActualCostUSD: floatPtr(0.42),
			},
		},
	}

	for i, tc := range cases {
		item, err := domain.NewActionItem(domain.ActionItemInput{
			ID:             tc.id,
			ProjectID:      project.ID,
			ColumnID:       column.ID,
			Kind:           domain.KindBuild,
			StructuralType: domain.StructuralTypeDroplet,
			Position:       i,
			Title:          "spawn metadata " + tc.id,
			Priority:       domain.PriorityMedium,
			Metadata:       tc.meta,
		}, now)
		if err != nil {
			t.Fatalf("NewActionItem(%s) error = %v", tc.id, err)
		}
		if err := repo.CreateActionItem(ctx, item); err != nil {
			t.Fatalf("CreateActionItem(%s) error = %v", tc.id, err)
		}

		loaded, err := repo.GetActionItem(ctx, tc.id)
		if err != nil {
			t.Fatalf("GetActionItem(%s) error = %v", tc.id, err)
		}
		assertSpawnMetadataEqual(t, tc.id, tc.meta, loaded.Metadata)
	}

	// Re-load via ListActionItems to exercise the second SELECT path.
	listed, err := repo.ListActionItems(ctx, project.ID, false)
	if err != nil {
		t.Fatalf("ListActionItems() error = %v", err)
	}
	byID := map[string]domain.ActionItem{}
	for _, item := range listed {
		byID[item.ID] = item
	}
	for _, tc := range cases {
		got, ok := byID[tc.id]
		if !ok {
			t.Fatalf("ListActionItems() missing %q", tc.id)
		}
		assertSpawnMetadataEqual(t, tc.id+" (list)", tc.meta, got.Metadata)
	}

	// Mutate the full-spawnmeta item: append a third history entry via
	// AppendSpawnHistory + persist via UpdateActionItem. Verifies the
	// full Drop 4c F.7.9 retry-on-failure flow round-trips through the
	// SQLite UPDATE path.
	target, err := repo.GetActionItem(ctx, "ai-full-spawnmeta")
	if err != nil {
		t.Fatalf("GetActionItem(ai-full-spawnmeta for update) error = %v", err)
	}
	tUpdate := now.Add(2 * time.Hour)
	target.AppendSpawnHistory(domain.SpawnHistoryEntry{
		SpawnID:      "spawn-3",
		BundlePath:   "/tmp/tillsyn/spawn-3/",
		StartedAt:    startedAt.Add(2 * time.Hour),
		TerminatedAt: terminatedAt.Add(2 * time.Hour),
		Outcome:      "success",
		TotalCostUSD: floatPtr(1.25),
	}, tUpdate)
	target.Metadata.ActualCostUSD = floatPtr(1.25)
	target.Metadata.SpawnBundlePath = "/tmp/tillsyn/spawn-3/"
	if err := repo.UpdateActionItem(ctx, target); err != nil {
		t.Fatalf("UpdateActionItem(append history) error = %v", err)
	}

	updated, err := repo.GetActionItem(ctx, "ai-full-spawnmeta")
	if err != nil {
		t.Fatalf("GetActionItem(after append) error = %v", err)
	}
	if len(updated.Metadata.SpawnHistory) != 3 {
		t.Fatalf("after append+update: history length = %d, want 3", len(updated.Metadata.SpawnHistory))
	}
	if updated.Metadata.SpawnHistory[2].SpawnID != "spawn-3" {
		t.Fatalf("after append+update: history[2].SpawnID = %q, want spawn-3 (preserved order)", updated.Metadata.SpawnHistory[2].SpawnID)
	}
	if updated.Metadata.SpawnBundlePath != "/tmp/tillsyn/spawn-3/" {
		t.Fatalf("after append+update: SpawnBundlePath = %q, want updated", updated.Metadata.SpawnBundlePath)
	}
	if updated.Metadata.ActualCostUSD == nil || *updated.Metadata.ActualCostUSD != 1.25 {
		t.Fatalf("after append+update: ActualCostUSD = %v, want 1.25", updated.Metadata.ActualCostUSD)
	}

	// Clear path: reset SpawnBundlePath + ActualCostUSD to zero values
	// (terminal-state cleanup) and confirm the JSON blob round-trips
	// without leaking the previous values.
	cleared, err := repo.GetActionItem(ctx, "ai-full-spawnmeta")
	if err != nil {
		t.Fatalf("GetActionItem(for clear) error = %v", err)
	}
	cleared.Metadata.SpawnBundlePath = ""
	cleared.Metadata.ActualCostUSD = nil
	cleared.UpdatedAt = tUpdate.Add(time.Hour)
	if err := repo.UpdateActionItem(ctx, cleared); err != nil {
		t.Fatalf("UpdateActionItem(clear) error = %v", err)
	}
	clearedReloaded, err := repo.GetActionItem(ctx, "ai-full-spawnmeta")
	if err != nil {
		t.Fatalf("GetActionItem(after clear) error = %v", err)
	}
	if clearedReloaded.Metadata.SpawnBundlePath != "" {
		t.Fatalf("after clear: SpawnBundlePath = %q, want empty", clearedReloaded.Metadata.SpawnBundlePath)
	}
	if clearedReloaded.Metadata.ActualCostUSD != nil {
		t.Fatalf("after clear: ActualCostUSD = %v, want nil", *clearedReloaded.Metadata.ActualCostUSD)
	}
	// Spawn history must survive the clear — it is append-only audit data.
	if len(clearedReloaded.Metadata.SpawnHistory) != 3 {
		t.Fatalf("after clear: history length = %d, want 3 (audit-only, never cleared)", len(clearedReloaded.Metadata.SpawnHistory))
	}
}

// assertSpawnMetadataEqual is a focused deep-equality helper for the Drop 4c
// F.7.9 spawn-metadata fields. Avoids reflect.DeepEqual on the full
// ActionItemMetadata struct so the failure messages name the specific
// field that diverged.
func assertSpawnMetadataEqual(t *testing.T, label string, want, got domain.ActionItemMetadata) {
	t.Helper()
	if got.SpawnBundlePath != want.SpawnBundlePath {
		t.Fatalf("%s: SpawnBundlePath = %q, want %q", label, got.SpawnBundlePath, want.SpawnBundlePath)
	}
	switch {
	case want.ActualCostUSD == nil && got.ActualCostUSD != nil:
		t.Fatalf("%s: ActualCostUSD = %v, want nil", label, *got.ActualCostUSD)
	case want.ActualCostUSD != nil && got.ActualCostUSD == nil:
		t.Fatalf("%s: ActualCostUSD = nil, want %v", label, *want.ActualCostUSD)
	case want.ActualCostUSD != nil && *got.ActualCostUSD != *want.ActualCostUSD:
		t.Fatalf("%s: ActualCostUSD = %v, want %v", label, *got.ActualCostUSD, *want.ActualCostUSD)
	}
	if len(got.SpawnHistory) != len(want.SpawnHistory) {
		t.Fatalf("%s: SpawnHistory length = %d, want %d", label, len(got.SpawnHistory), len(want.SpawnHistory))
	}
	for i := range want.SpawnHistory {
		w, g := want.SpawnHistory[i], got.SpawnHistory[i]
		if g.SpawnID != w.SpawnID {
			t.Fatalf("%s: history[%d].SpawnID = %q, want %q", label, i, g.SpawnID, w.SpawnID)
		}
		if g.BundlePath != w.BundlePath {
			t.Fatalf("%s: history[%d].BundlePath = %q, want %q", label, i, g.BundlePath, w.BundlePath)
		}
		if !g.StartedAt.Equal(w.StartedAt) {
			t.Fatalf("%s: history[%d].StartedAt = %v, want %v", label, i, g.StartedAt, w.StartedAt)
		}
		if !g.TerminatedAt.Equal(w.TerminatedAt) {
			t.Fatalf("%s: history[%d].TerminatedAt = %v, want %v", label, i, g.TerminatedAt, w.TerminatedAt)
		}
		if g.Outcome != w.Outcome {
			t.Fatalf("%s: history[%d].Outcome = %q, want %q", label, i, g.Outcome, w.Outcome)
		}
		switch {
		case w.TotalCostUSD == nil && g.TotalCostUSD != nil:
			t.Fatalf("%s: history[%d].TotalCostUSD = %v, want nil", label, i, *g.TotalCostUSD)
		case w.TotalCostUSD != nil && g.TotalCostUSD == nil:
			t.Fatalf("%s: history[%d].TotalCostUSD = nil, want %v", label, i, *w.TotalCostUSD)
		case w.TotalCostUSD != nil && *g.TotalCostUSD != *w.TotalCostUSD:
			t.Fatalf("%s: history[%d].TotalCostUSD = %v, want %v", label, i, *g.TotalCostUSD, *w.TotalCostUSD)
		}
	}
}
