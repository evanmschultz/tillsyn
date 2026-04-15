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

// TestRepositoryEmbeddingLifecycleAdapterClaimLossReturnsConflict verifies stale workers surface lost claims instead of silently succeeding.
func TestRepositoryEmbeddingLifecycleAdapterClaimLossReturnsConflict(t *testing.T) {
	ctx := context.Background()
	repo, err := OpenInMemory()
	if err != nil {
		t.Fatalf("OpenInMemory() error = %v", err)
	}
	t.Cleanup(func() {
		_ = repo.Close()
	})

	now := time.Date(2026, 3, 29, 20, 0, 0, 0, time.UTC)
	project, column := mustSeedEmbeddingScope(t, repo, now, "p-embeddings-adapter-1")
	task := mustSeedEmbeddingTask(t, repo, project.ID, column.ID, "task-claim-loss", 0, now, mustEmbeddingMetadata("claim loss"), []string{"claim"})
	contentHash := hashEmbeddingContent(buildSQLiteTaskEmbeddingContent(task))

	if _, err := repo.EnqueueEmbedding(ctx, app.EmbeddingEnqueueInput{
		SubjectType:     app.EmbeddingSubjectTypeWorkItem,
		SubjectID:       task.ID,
		ProjectID:       project.ID,
		ContentHash:     contentHash,
		ModelProvider:   "fantasy",
		ModelName:       "mini",
		ModelDimensions: 3,
		ModelSignature:  "fantasy|mini||3",
		MaxAttempts:     5,
	}); err != nil {
		t.Fatalf("EnqueueEmbedding() error = %v", err)
	}
	claimNow := time.Now().UTC().Add(time.Second)
	claims, err := repo.ClaimEmbeddings(ctx, app.EmbeddingClaimInput{
		SubjectType: app.EmbeddingSubjectTypeWorkItem,
		WorkerID:    "worker-a",
		Now:         claimNow,
		Limit:       1,
		ClaimTTL:    time.Millisecond,
	})
	if err != nil {
		t.Fatalf("ClaimEmbeddings() error = %v", err)
	}
	if len(claims) != 1 {
		t.Fatalf("claim count = %d, want 1", len(claims))
	}
	if recovered, err := repo.RecoverExpiredEmbeddingClaims(ctx, claimNow.Add(2*time.Second)); err != nil {
		t.Fatalf("RecoverExpiredEmbeddingClaims() error = %v", err)
	} else if len(recovered) != 1 {
		t.Fatalf("recovered = %d, want 1", len(recovered))
	}

	if err := repo.HeartbeatEmbedding(ctx, app.EmbeddingHeartbeatInput{
		SubjectType: app.EmbeddingSubjectTypeWorkItem,
		SubjectID:   task.ID,
		WorkerID:    "worker-a",
		Now:         claimNow.Add(3 * time.Second),
		ClaimTTL:    time.Minute,
	}); !errors.Is(err, app.ErrEmbeddingClaimLost) {
		t.Fatalf("HeartbeatEmbedding() error = %v, want ErrEmbeddingClaimLost", err)
	}
	if _, err := repo.MarkEmbeddingSuccess(ctx, app.EmbeddingSuccessInput{
		SubjectType:     app.EmbeddingSubjectTypeWorkItem,
		SubjectID:       task.ID,
		ProjectID:       project.ID,
		ContentHash:     contentHash,
		ModelProvider:   "fantasy",
		ModelName:       "mini",
		ModelDimensions: 3,
		ModelSignature:  "fantasy|mini||3",
		WorkerID:        "worker-a",
		CompletedAt:     claimNow.Add(3 * time.Second),
	}); !errors.Is(err, app.ErrEmbeddingClaimLost) {
		t.Fatalf("MarkEmbeddingSuccess() error = %v, want ErrEmbeddingClaimLost", err)
	}
	if _, err := repo.MarkEmbeddingFailure(ctx, app.EmbeddingFailureInput{
		SubjectType:         app.EmbeddingSubjectTypeWorkItem,
		SubjectID:           task.ID,
		ProjectID:           project.ID,
		ModelSignature:      "fantasy|mini||3",
		WorkerID:            "worker-a",
		Retryable:           true,
		ErrorCode:           "provider_unavailable",
		ErrorMessage:        "temporary outage",
		ErrorSummary:        "temporary outage",
		FailedAt:            claimNow.Add(3 * time.Second),
		InitialRetryBackoff: 15 * time.Second,
		MaxRetryBackoff:     time.Minute,
	}); !errors.Is(err, app.ErrEmbeddingClaimLost) {
		t.Fatalf("MarkEmbeddingFailure() error = %v, want ErrEmbeddingClaimLost", err)
	}

	row, err := repo.GetEmbeddingJob(ctx, "task", task.ID)
	if err != nil {
		t.Fatalf("GetEmbeddingJob() error = %v", err)
	}
	if row.Status != EmbeddingJobStatusPending {
		t.Fatalf("status = %s, want pending after recovery", row.Status)
	}
}

// TestRepositoryEmbeddingLifecycleAdapterModelInvalidationUpdatesMetadata verifies stale-by-model sweeps rewrite the desired model metadata.
func TestRepositoryEmbeddingLifecycleAdapterModelInvalidationUpdatesMetadata(t *testing.T) {
	ctx := context.Background()
	repo, err := OpenInMemory()
	if err != nil {
		t.Fatalf("OpenInMemory() error = %v", err)
	}
	t.Cleanup(func() {
		_ = repo.Close()
	})

	now := time.Date(2026, 3, 29, 20, 5, 0, 0, time.UTC)
	project, column := mustSeedEmbeddingScope(t, repo, now, "p-embeddings-adapter-2")
	task := mustSeedEmbeddingTask(t, repo, project.ID, column.ID, "task-model-sweep", 0, now, mustEmbeddingMetadata("model sweep"), []string{"model"})
	contentHash := hashEmbeddingContent(buildSQLiteTaskEmbeddingContent(task))

	if _, changed, err := repo.UpsertEmbeddingJob(ctx, EmbeddingJobUpsertInput{
		SubjectType:   "task",
		SubjectID:     task.ID,
		ProjectID:     project.ID,
		DesiredHash:   contentHash,
		ModelProvider: "fantasy",
		ModelName:     "mini",
		ModelDims:     3,
		ModelSig:      "fantasy|mini||3",
		MaxAttempts:   5,
	}); err != nil {
		t.Fatalf("UpsertEmbeddingJob() error = %v", err)
	} else if !changed {
		t.Fatal("expected seed upsert to create a lifecycle row")
	}
	claimNow := time.Now().UTC().Add(time.Second)
	claimed, found, err := repo.ClaimNextEmbeddingJob(ctx, EmbeddingJobClaimNextInput{
		SubjectType: "task",
		ProjectID:   project.ID,
		WorkerID:    "worker-model",
		ClaimTTL:    time.Minute,
	}, claimNow)
	if err != nil {
		t.Fatalf("ClaimNextEmbeddingJob() error = %v", err)
	}
	if !found {
		t.Fatal("expected lifecycle row to be claimable")
	}
	if _, changed, err := repo.CompleteEmbeddingJob(ctx, EmbeddingJobCompleteInput{
		SubjectType:          claimed.SubjectType,
		SubjectID:            claimed.SubjectID,
		WorkerID:             "worker-model",
		ProcessedContentHash: contentHash,
		ProcessedModelSig:    "fantasy|mini||3",
	}); err != nil {
		t.Fatalf("CompleteEmbeddingJob() error = %v", err)
	} else if !changed {
		t.Fatal("expected completion to update the lifecycle row")
	}

	updatedRows, err := repo.MarkEmbeddingsStaleByModel(ctx, app.EmbeddingStaleByModelInput{
		SubjectType:     app.EmbeddingSubjectTypeWorkItem,
		ModelProvider:   "openai",
		ModelName:       "text-embedding-3-small",
		ModelDimensions: 1536,
		ModelSignature:  "openai|text-embedding-3-small||1536",
		Reason:          "model_signature_changed",
		StaledAt:        now.Add(2 * time.Second),
	})
	if err != nil {
		t.Fatalf("MarkEmbeddingsStaleByModel() error = %v", err)
	}
	if len(updatedRows) != 1 {
		t.Fatalf("updated count = %d, want 1", len(updatedRows))
	}

	rows, err := repo.ListEmbeddings(ctx, app.EmbeddingListFilter{
		ProjectIDs:  []string{project.ID},
		SubjectType: app.EmbeddingSubjectTypeWorkItem,
	})
	if err != nil {
		t.Fatalf("ListEmbeddings() error = %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("row count = %d, want 1", len(rows))
	}
	row := rows[0]
	if row.Status != app.EmbeddingLifecycleStale {
		t.Fatalf("status = %q, want stale", row.Status)
	}
	if row.ModelProvider != "openai" || row.ModelName != "text-embedding-3-small" || row.ModelDimensions != 1536 {
		t.Fatalf("model metadata = %#v, want updated provider/name/dimensions", row)
	}
	if row.ModelSignature != "openai|text-embedding-3-small||1536" {
		t.Fatalf("model signature = %q, want updated signature", row.ModelSignature)
	}
}

// TestRepositoryEmbeddingLifecycleAdapterExplicitEmptyScopeReturnsEmpty verifies explicit empty project scope does not broaden into global inventory.
func TestRepositoryEmbeddingLifecycleAdapterExplicitEmptyScopeReturnsEmpty(t *testing.T) {
	ctx := context.Background()
	repo, err := OpenInMemory()
	if err != nil {
		t.Fatalf("OpenInMemory() error = %v", err)
	}
	t.Cleanup(func() {
		_ = repo.Close()
	})

	now := time.Date(2026, 3, 29, 20, 10, 0, 0, time.UTC)
	project, column := mustSeedEmbeddingScope(t, repo, now, "p-embeddings-adapter-3")
	task := mustSeedEmbeddingTask(t, repo, project.ID, column.ID, "task-empty-scope", 0, now, mustEmbeddingMetadata("empty scope"), []string{"scope"})
	if _, err := repo.EnqueueEmbedding(ctx, app.EmbeddingEnqueueInput{
		SubjectType:     app.EmbeddingSubjectTypeWorkItem,
		SubjectID:       task.ID,
		ProjectID:       project.ID,
		ContentHash:     hashEmbeddingContent(buildSQLiteTaskEmbeddingContent(task)),
		ModelProvider:   "fantasy",
		ModelName:       "mini",
		ModelDimensions: 3,
		ModelSignature:  "fantasy|mini||3",
		MaxAttempts:     5,
	}); err != nil {
		t.Fatalf("EnqueueEmbedding() error = %v", err)
	}

	rows, err := repo.ListEmbeddings(ctx, app.EmbeddingListFilter{
		ProjectIDs:  []string{},
		SubjectType: app.EmbeddingSubjectTypeWorkItem,
	})
	if err != nil {
		t.Fatalf("ListEmbeddings() error = %v", err)
	}
	if len(rows) != 0 {
		t.Fatalf("row count = %d, want 0 for explicit empty scope", len(rows))
	}
	summary, err := repo.SummarizeEmbeddings(ctx, app.EmbeddingListFilter{
		ProjectIDs:  []string{},
		SubjectType: app.EmbeddingSubjectTypeWorkItem,
	})
	if err != nil {
		t.Fatalf("SummarizeEmbeddings() error = %v", err)
	}
	if summary.PendingCount != 0 || summary.ReadyCount != 0 || summary.FailedCount != 0 || summary.StaleCount != 0 || summary.RunningCount != 0 {
		t.Fatalf("summary = %#v, want zero counts for explicit empty scope", summary)
	}
}

// TestRepositoryEmbeddingLifecycleSchemaMigratesLegacyDatabase verifies existing sqlite files gain the embeddings lifecycle table and indexes on open.
func TestRepositoryEmbeddingLifecycleSchemaMigratesLegacyDatabase(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "legacy-embeddings.db")
	db, err := sql.Open(driverName, dbPath)
	if err != nil {
		t.Fatalf("sql.Open() error = %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	now := time.Date(2026, 3, 29, 20, 15, 0, 0, time.UTC).Format(time.RFC3339Nano)
	for _, stmt := range []string{
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
			position INTEGER NOT NULL,
			wip_limit INTEGER NOT NULL DEFAULT 0,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			archived_at TEXT
		)`,
		`CREATE TABLE work_items (
			id TEXT PRIMARY KEY,
			project_id TEXT NOT NULL,
			column_id TEXT NOT NULL,
			parent_id TEXT,
			kind TEXT NOT NULL DEFAULT 'task',
			lifecycle_state TEXT NOT NULL DEFAULT 'todo',
			position INTEGER NOT NULL,
			title TEXT NOT NULL,
			description TEXT NOT NULL DEFAULT '',
			priority TEXT NOT NULL DEFAULT 'medium',
			due_at TEXT,
			labels_json TEXT NOT NULL DEFAULT '[]',
			metadata_json TEXT NOT NULL DEFAULT '{}',
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			archived_at TEXT,
			canceled_at TEXT
		)`,
		`CREATE TABLE task_embeddings (
			task_id TEXT PRIMARY KEY,
			project_id TEXT NOT NULL,
			content_hash TEXT NOT NULL,
			content TEXT NOT NULL DEFAULT '',
			embedding BLOB NOT NULL,
			updated_at TEXT NOT NULL
		)`,
	} {
		if _, err := db.ExecContext(ctx, stmt); err != nil {
			t.Fatalf("create legacy schema error = %v", err)
		}
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO projects(id, slug, name, description, created_at, updated_at, archived_at) VALUES (?, ?, ?, ?, ?, ?, NULL)`, "p1", "legacy", "Legacy", "", now, now); err != nil {
		t.Fatalf("seed legacy project error = %v", err)
	}

	repo, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(func() {
		_ = repo.Close()
	})

	var tableCount int
	if err := repo.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='embedding_jobs'`).Scan(&tableCount); err != nil {
		t.Fatalf("count embedding_jobs table error = %v", err)
	}
	if tableCount != 1 {
		t.Fatalf("embedding_jobs table count = %d, want 1", tableCount)
	}
	for _, indexName := range []string{
		"idx_embedding_jobs_project_status_updated_at",
		"idx_embedding_jobs_project_next_attempt",
		"idx_embedding_jobs_claim_expires",
	} {
		var indexCount int
		if err := repo.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM sqlite_master WHERE type='index' AND name = ?`, indexName).Scan(&indexCount); err != nil {
			t.Fatalf("count %s error = %v", indexName, err)
		}
		if indexCount != 1 {
			t.Fatalf("%s count = %d, want 1", indexName, indexCount)
		}
	}
}

// mustEmbeddingMetadata builds one minimal metadata payload for embeddings adapter tests.
func mustEmbeddingMetadata(objective string) domain.TaskMetadata {
	return domain.TaskMetadata{
		Objective:          objective,
		AcceptanceCriteria: "Keep lifecycle durable",
		ValidationPlan:     "Run storage tests",
	}
}
