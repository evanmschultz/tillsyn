package main

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/evanmschultz/tillsyn/internal/adapters/storage/sqlite"
	"github.com/evanmschultz/tillsyn/internal/app"
	"github.com/evanmschultz/tillsyn/internal/config"
	"github.com/evanmschultz/tillsyn/internal/domain"
)

// TestBuildEmbeddingRuntimeConfigParsesDurations verifies CLI runtime config parsing stays aligned with TOML fields.
func TestBuildEmbeddingRuntimeConfigParsesDurations(t *testing.T) {
	cfg := config.Default(":memory:")
	cfg.Embeddings.Enabled = true
	cfg.Embeddings.Provider = "deterministic"
	cfg.Embeddings.Model = "hash-bow-v1"
	cfg.Embeddings.BaseURL = ""
	cfg.Embeddings.Dimensions = 3
	cfg.Embeddings.WorkerPollInterval = "5s"
	cfg.Embeddings.ClaimTTL = "3m"
	cfg.Embeddings.MaxAttempts = 7
	cfg.Embeddings.InitialRetryBackoff = "20s"
	cfg.Embeddings.MaxRetryBackoff = "10m"

	runtimeCfg, err := buildEmbeddingRuntimeConfig(cfg, "tillsyn-test", "serve")
	if err != nil {
		t.Fatalf("buildEmbeddingRuntimeConfig() error = %v", err)
	}
	if runtimeCfg.PollInterval != 5*time.Second {
		t.Fatalf("poll interval = %s, want 5s", runtimeCfg.PollInterval)
	}
	if runtimeCfg.ClaimTTL != 3*time.Minute {
		t.Fatalf("claim ttl = %s, want 3m", runtimeCfg.ClaimTTL)
	}
	if runtimeCfg.MaxAttempts != 7 {
		t.Fatalf("max attempts = %d, want 7", runtimeCfg.MaxAttempts)
	}
	if runtimeCfg.InitialRetryBackoff != 20*time.Second {
		t.Fatalf("initial retry backoff = %s, want 20s", runtimeCfg.InitialRetryBackoff)
	}
	if runtimeCfg.MaxRetryBackoff != 10*time.Minute {
		t.Fatalf("max retry backoff = %s, want 10m", runtimeCfg.MaxRetryBackoff)
	}
	if runtimeCfg.ModelSignature != app.BuildEmbeddingModelSignature("deterministic", "hash-bow-v1", "", 3) {
		t.Fatalf("model signature = %q", runtimeCfg.ModelSignature)
	}
	if !strings.Contains(runtimeCfg.WorkerID, "tillsyn-test:serve:") {
		t.Fatalf("worker id = %q, want app and command segments", runtimeCfg.WorkerID)
	}
}

// TestRunEmbeddingsStatusRendersSummaryAndRows verifies the status command prints lifecycle counts and rows.
func TestRunEmbeddingsStatusRendersSummaryAndRows(t *testing.T) {
	svc, projectID, _, createdActionItemID := newEmbeddingsCLIServiceForTest(t)
	if _, err := svc.UpdateProject(context.Background(), app.UpdateProjectInput{
		ProjectID:     projectID,
		Name:          "Inbox",
		Description:   "Refresh the project document for embeddings inventory",
		UpdatedBy:     "tillsyn-user",
		UpdatedByName: "tillsyn-user",
		UpdatedType:   domain.ActorTypeUser,
	}); err != nil {
		t.Fatalf("UpdateProject() error = %v", err)
	}
	if _, err := svc.CreateComment(context.Background(), app.CreateCommentInput{
		ProjectID:    projectID,
		TargetType:   domain.CommentTargetTypeActionItem,
		TargetID:     createdActionItemID,
		Summary:      "Thread context note",
		BodyMarkdown: "Thread context note",
		ActorID:      "tillsyn-user",
		ActorName:    "tillsyn-user",
		ActorType:    domain.ActorTypeUser,
	}); err != nil {
		t.Fatalf("CreateComment() error = %v", err)
	}

	var out strings.Builder
	if err := runEmbeddingsStatus(context.Background(), svc, embeddingsStatusCommandOptions{
		projectID: projectID,
		limit:     10,
	}, &out); err != nil {
		t.Fatalf("runEmbeddingsStatus() error = %v", err)
	}
	rendered := normalizeCLIOutput(out.String())
	for _, want := range []string{
		"Embeddings Status",
		"runtime unavailable",
		"PROJECT",
		"TYPE",
		"SUBJECT",
		"STATUS",
		"project_document",
		"thread_context",
		"work_item",
		"pending 3",
		projectID,
		createdActionItemID,
		"Healthy means ready > 0",
	} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("expected %q in embeddings status output, got %q", want, rendered)
		}
	}
}

// TestRunEmbeddingsReindexRendersQueuedWork verifies the reindex command prints queued work counts.
func TestRunEmbeddingsReindexRendersQueuedWork(t *testing.T) {
	svc, projectID, repo, _ := newEmbeddingsCLIServiceForTest(t)
	now := time.Date(2026, 3, 29, 18, 30, 0, 0, time.UTC)
	actionItem, err := domain.NewActionItem(domain.ActionItemInput{
		ID:             "actionItem-reindex",
		ProjectID:      projectID,
		ColumnID:       "c1",
		Position:       1,
		Title:          "Backfill semantic rows",
		UpdatedByType:  domain.ActorTypeUser,
		CreatedByActor: "tillsyn-user",
		CreatedByName:  "tillsyn-user",
		UpdatedByActor: "tillsyn-user",
		UpdatedByName:  "tillsyn-user",
	}, now)
	if err != nil {
		t.Fatalf("NewActionItem() error = %v", err)
	}
	if err := repo.CreateActionItem(context.Background(), actionItem); err != nil {
		t.Fatalf("CreateActionItem() error = %v", err)
	}

	var out strings.Builder
	cfg := config.Default(":memory:")
	cfg.Embeddings.Enabled = true
	if err := runEmbeddingsReindex(context.Background(), svc, cfg, embeddingsReindexCommandOptions{
		projectID: projectID,
	}, &out); err != nil {
		t.Fatalf("runEmbeddingsReindex() error = %v", err)
	}
	rendered := normalizeCLIOutput(out.String())
	for _, want := range []string{
		"Embeddings Reindex",
		"status queued",
		projectID,
		"queued 3",
		"pending 3",
	} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("expected %q in embeddings reindex output, got %q", want, rendered)
		}
	}
}

// TestRunEmbeddingsReindexRejectsDisabledEmbeddings verifies explicit reindex refuses to enqueue when embeddings are disabled.
func TestRunEmbeddingsReindexRejectsDisabledEmbeddings(t *testing.T) {
	svc, projectID, _, _ := newEmbeddingsCLIServiceForTest(t)

	var out strings.Builder
	err := runEmbeddingsReindex(context.Background(), svc, config.Default(":memory:"), embeddingsReindexCommandOptions{
		projectID: projectID,
	}, &out)
	if err == nil || !strings.Contains(err.Error(), "embeddings are disabled") {
		t.Fatalf("runEmbeddingsReindex() error = %v, want disabled embeddings guidance", err)
	}
}

// TestWriteEmbeddingReindexResultRendersFailedStatus verifies terminal failures do not render as completed work.
func TestWriteEmbeddingReindexResultRendersFailedStatus(t *testing.T) {
	var out strings.Builder
	err := writeEmbeddingReindexResult(&out, app.ReindexEmbeddingsResult{
		TargetProjects: []string{"p-embeddings-cli"},
		ScannedCount:   3,
		QueuedCount:    3,
		FailedCount:    1,
	})
	if err != nil {
		t.Fatalf("writeEmbeddingReindexResult() error = %v", err)
	}
	rendered := normalizeCLIOutput(out.String())
	for _, want := range []string{
		"status failed",
		"failed 1",
		"terminal failed state",
	} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("expected %q in failed reindex output, got %q", want, rendered)
		}
	}
}

// normalizeCLIOutput collapses laslig-rendered spacing so substring assertions stay stable.
func normalizeCLIOutput(raw string) string {
	return strings.Join(strings.Fields(raw), " ")
}

// newEmbeddingsCLIServiceForTest seeds one app service with lifecycle-enabled work-item state.
func newEmbeddingsCLIServiceForTest(t *testing.T) (*app.Service, string, *sqlite.Repository, string) {
	t.Helper()

	repo, err := sqlite.OpenInMemory()
	if err != nil {
		t.Fatalf("OpenInMemory() error = %v", err)
	}
	t.Cleanup(func() {
		_ = repo.Close()
	})

	now := time.Date(2026, 3, 29, 18, 0, 0, 0, time.UTC)
	project, err := domain.NewProject("p-embeddings-cli", "Inbox", "", now)
	if err != nil {
		t.Fatalf("NewProject() error = %v", err)
	}
	if err := repo.CreateProject(context.Background(), project); err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}
	column, err := domain.NewColumn("c1", project.ID, "To Do", 0, 0, now)
	if err != nil {
		t.Fatalf("NewColumn() error = %v", err)
	}
	if err := repo.CreateColumn(context.Background(), column); err != nil {
		t.Fatalf("CreateColumn() error = %v", err)
	}

	svc := app.NewService(repo, func() string { return "actionItem-created" }, func() time.Time { return now }, app.ServiceConfig{
		EmbeddingRuntime: app.EmbeddingRuntimeConfig{
			Enabled:        true,
			Provider:       "deterministic",
			Model:          "hash-bow-v1",
			Dimensions:     3,
			ModelSignature: app.BuildEmbeddingModelSignature("deterministic", "hash-bow-v1", "", 3),
			MaxAttempts:    5,
		},
	})
	created, err := svc.CreateActionItem(context.Background(), app.CreateActionItemInput{
		ProjectID: project.ID,
		ColumnID:  column.ID,
		Title:     "Ship operational embeddings",
	})
	if err != nil {
		t.Fatalf("CreateActionItem() error = %v", err)
	}
	return svc, project.ID, repo, created.ID
}
