package app

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/evanmschultz/tillsyn/internal/domain"
)

// seedEmbeddingRuntimeScope stores one minimal project/column/actionItem scope for embeddings runtime tests.
func seedEmbeddingRuntimeScope(t *testing.T, repo *fakeRepo, now time.Time) (domain.Project, domain.Column, domain.ActionItem) {
	t.Helper()

	project, err := domain.NewProject("p-runtime", "Runtime", "", now)
	if err != nil {
		t.Fatalf("NewProject() error = %v", err)
	}
	column, err := domain.NewColumn("c-runtime", project.ID, "To Do", 0, 0, now)
	if err != nil {
		t.Fatalf("NewColumn() error = %v", err)
	}
	actionItem, err := domain.NewActionItem(domain.ActionItemInput{
		ID:          "t-runtime",
		ProjectID:   project.ID,
		ColumnID:    column.ID,
		Position:    0,
		Title:       "Ship embeddings",
		Description: "Make lifecycle state operational",
		Priority:    domain.PriorityMedium,
		Labels:      []string{"embeddings"},
	}, now)
	if err != nil {
		t.Fatalf("NewActionItem() error = %v", err)
	}

	repo.projects[project.ID] = project
	repo.columns[column.ID] = column
	repo.tasks[actionItem.ID] = actionItem
	return project, column, actionItem
}

// TestServiceCreateActionItemDoesNotEnqueueEmbeddingsWhenDisabled verifies disabled embeddings do not accumulate pending lifecycle rows.
func TestServiceCreateActionItemDoesNotEnqueueEmbeddingsWhenDisabled(t *testing.T) {
	repo := newFakeRepo()
	now := time.Date(2026, 3, 29, 19, 0, 0, 0, time.UTC)
	project, column, _ := seedEmbeddingRuntimeScope(t, repo, now)
	lifecycle := newFakeEmbeddingLifecycleStore()

	svc := NewService(repo, func() string { return "t-disabled" }, func() time.Time { return now }, ServiceConfig{
		EmbeddingLifecycle: lifecycle,
		EmbeddingRuntime: EmbeddingRuntimeConfig{
			Enabled:        false,
			Provider:       "fantasy",
			Model:          "mini",
			Dimensions:     3,
			ModelSignature: BuildEmbeddingModelSignature("fantasy", "mini", "", 3),
		},
	})
	if _, err := svc.CreateActionItem(context.Background(), CreateActionItemInput{
		ProjectID: project.ID,
		ColumnID:  column.ID,
		Title:     "Disabled queue should stay empty",
	}); err != nil {
		t.Fatalf("CreateActionItem() error = %v", err)
	}
	if len(lifecycle.inputs) != 0 {
		t.Fatalf("enqueue inputs = %#v, want none when embeddings are disabled", lifecycle.inputs)
	}
}

// TestServiceSearchActionItemsLeavesMissingLifecycleRowsUntracked verifies search does not invent pending status for subjects with no lifecycle row.
func TestServiceSearchActionItemsLeavesMissingLifecycleRowsUntracked(t *testing.T) {
	repo := newFakeRepo()
	now := time.Date(2026, 3, 29, 19, 5, 0, 0, time.UTC)
	project, _, actionItem := seedEmbeddingRuntimeScope(t, repo, now)

	svc := NewService(repo, func() string { return "t-search" }, func() time.Time { return now }, ServiceConfig{
		EmbeddingLifecycle: newFakeEmbeddingLifecycleStore(),
		EmbeddingRuntime: EmbeddingRuntimeConfig{
			Enabled:        true,
			Provider:       "fantasy",
			Model:          "mini",
			Dimensions:     3,
			ModelSignature: BuildEmbeddingModelSignature("fantasy", "mini", "", 3),
		},
	})

	result, err := svc.SearchActionItems(context.Background(), SearchActionItemsFilter{
		ProjectID: project.ID,
		Query:     "ship embeddings",
		Mode:      SearchModeKeyword,
	})
	if err != nil {
		t.Fatalf("SearchActionItems() error = %v", err)
	}
	if len(result.Matches) != 1 {
		t.Fatalf("match count = %d, want 1", len(result.Matches))
	}
	if result.Matches[0].ActionItem.ID != actionItem.ID {
		t.Fatalf("match actionItem id = %q, want %q", result.Matches[0].ActionItem.ID, actionItem.ID)
	}
	if result.Matches[0].EmbeddingStatus != "" {
		t.Fatalf("embedding status = %q, want untracked empty status", result.Matches[0].EmbeddingStatus)
	}
	if result.EmbeddingSummary.PendingCount != 0 {
		t.Fatalf("embedding summary = %#v, want no pending rows for missing lifecycle state", result.EmbeddingSummary)
	}
}

// TestServiceSearchActionItemsFallsBackWhenSemanticCandidatesAreNotReady verifies semantic ranking ignores pending lifecycle rows.
func TestServiceSearchActionItemsFallsBackWhenSemanticCandidatesAreNotReady(t *testing.T) {
	repo := newFakeRepo()
	now := time.Date(2026, 3, 29, 19, 10, 0, 0, time.UTC)
	project, _, actionItem := seedEmbeddingRuntimeScope(t, repo, now)
	lifecycle := newFakeEmbeddingLifecycleStore()
	lifecycle.enqueues[lifecycle.embeddingKey(EmbeddingSubjectTypeWorkItem, actionItem.ID)] = EmbeddingRecord{
		SubjectType: EmbeddingSubjectTypeWorkItem,
		SubjectID:   actionItem.ID,
		ProjectID:   project.ID,
		Status:      EmbeddingLifecyclePending,
	}
	generator := &fakeEmbeddingGenerator{vectors: [][]float32{{0.9, 0.1, 0.2}}}
	searchIndex := &fakeActionItemSearchIndex{
		searchRows: []EmbeddingSearchMatch{{SubjectType: EmbeddingSubjectTypeWorkItem, SubjectID: actionItem.ID, SearchTargetType: EmbeddingSearchTargetTypeWorkItem, SearchTargetID: actionItem.ID, Similarity: 0.98}},
	}

	svc := NewService(repo, func() string { return "t-semantic" }, func() time.Time { return now }, ServiceConfig{
		EmbeddingLifecycle: lifecycle,
		EmbeddingGenerator: generator,
		SearchIndex:        searchIndex,
		EmbeddingRuntime: EmbeddingRuntimeConfig{
			Enabled:        true,
			Provider:       "fantasy",
			Model:          "mini",
			Dimensions:     3,
			ModelSignature: BuildEmbeddingModelSignature("fantasy", "mini", "", 3),
		},
	})

	result, err := svc.SearchActionItems(context.Background(), SearchActionItemsFilter{
		ProjectID: project.ID,
		Query:     "ship embeddings",
		Mode:      SearchModeSemantic,
	})
	if err != nil {
		t.Fatalf("SearchActionItems() error = %v", err)
	}
	if result.EffectiveMode != SearchModeKeyword {
		t.Fatalf("effective mode = %q, want keyword fallback", result.EffectiveMode)
	}
	if result.FallbackReason != "semantic_index_not_ready" {
		t.Fatalf("fallback reason = %q, want semantic_index_not_ready", result.FallbackReason)
	}
	if len(result.Matches) != 1 {
		t.Fatalf("match count = %d, want lexical fallback row", len(result.Matches))
	}
	if result.Matches[0].UsedSemantic {
		t.Fatalf("UsedSemantic = true, want false when lifecycle row is pending")
	}
	if result.Matches[0].EmbeddingStatus != EmbeddingLifecyclePending {
		t.Fatalf("embedding status = %q, want pending", result.Matches[0].EmbeddingStatus)
	}
}

// TestServiceSearchActionItemsFallsBackWithoutLifecycleStore verifies semantic ranking is disabled when lifecycle truth is unavailable.
func TestServiceSearchActionItemsFallsBackWithoutLifecycleStore(t *testing.T) {
	repo := newFakeRepo()
	now := time.Date(2026, 3, 29, 19, 12, 0, 0, time.UTC)
	project, _, _ := seedEmbeddingRuntimeScope(t, repo, now)
	generator := &fakeEmbeddingGenerator{vectors: [][]float32{{0.9, 0.1, 0.2}}}
	searchIndex := &fakeActionItemSearchIndex{
		searchRows: []EmbeddingSearchMatch{{SubjectType: EmbeddingSubjectTypeWorkItem, SubjectID: "t-runtime", SearchTargetType: EmbeddingSearchTargetTypeWorkItem, SearchTargetID: "t-runtime", Similarity: 0.98}},
	}

	svc := NewService(repo, func() string { return "t-semantic-no-lifecycle" }, func() time.Time { return now }, ServiceConfig{
		EmbeddingGenerator: generator,
		SearchIndex:        searchIndex,
		EmbeddingRuntime: EmbeddingRuntimeConfig{
			Enabled:        true,
			Provider:       "fantasy",
			Model:          "mini",
			Dimensions:     3,
			ModelSignature: BuildEmbeddingModelSignature("fantasy", "mini", "", 3),
		},
	})

	result, err := svc.SearchActionItems(context.Background(), SearchActionItemsFilter{
		ProjectID: project.ID,
		Query:     "ship embeddings",
		Mode:      SearchModeSemantic,
	})
	if err != nil {
		t.Fatalf("SearchActionItems() error = %v", err)
	}
	if result.EffectiveMode != SearchModeKeyword {
		t.Fatalf("effective mode = %q, want keyword fallback", result.EffectiveMode)
	}
	if result.FallbackReason != "embedding_lifecycle_unavailable" {
		t.Fatalf("fallback reason = %q, want embedding_lifecycle_unavailable", result.FallbackReason)
	}
	if len(result.Matches) != 1 {
		t.Fatalf("match count = %d, want lexical fallback row", len(result.Matches))
	}
	if result.Matches[0].UsedSemantic {
		t.Fatalf("UsedSemantic = true, want false without lifecycle store")
	}
}

// TestServiceReindexEmbeddingsDisabledReturnsErrEmbeddingsDisabled verifies explicit reindex refuses disabled runtime configuration.
func TestServiceReindexEmbeddingsDisabledReturnsErrEmbeddingsDisabled(t *testing.T) {
	repo := newFakeRepo()
	now := time.Date(2026, 3, 29, 19, 15, 0, 0, time.UTC)
	project, _, _ := seedEmbeddingRuntimeScope(t, repo, now)

	svc := NewService(repo, func() string { return "t-reindex-disabled" }, func() time.Time { return now }, ServiceConfig{
		EmbeddingLifecycle: newFakeEmbeddingLifecycleStore(),
		EmbeddingRuntime: EmbeddingRuntimeConfig{
			Enabled:        false,
			ModelSignature: BuildEmbeddingModelSignature("fantasy", "mini", "", 3),
		},
	})
	_, err := svc.ReindexEmbeddings(context.Background(), ReindexEmbeddingsInput{
		ProjectID: project.ID,
	})
	if !errors.Is(err, ErrEmbeddingsDisabled) {
		t.Fatalf("ReindexEmbeddings() error = %v, want ErrEmbeddingsDisabled", err)
	}
}

// TestServiceReindexEmbeddingsWaitDoesNotCompleteWhenFailuresRemain verifies wait mode does not report success with terminal failures.
func TestServiceReindexEmbeddingsWaitDoesNotCompleteWhenFailuresRemain(t *testing.T) {
	repo := newFakeRepo()
	now := time.Date(2026, 3, 29, 19, 20, 0, 0, time.UTC)
	project, _, actionItem := seedEmbeddingRuntimeScope(t, repo, now)
	lifecycle := newFakeEmbeddingLifecycleStore()
	lifecycle.summarySequence = []EmbeddingSummary{
		{SubjectType: EmbeddingSubjectTypeWorkItem, ProjectIDs: []string{project.ID}, PendingCount: 1},
		{SubjectType: EmbeddingSubjectTypeWorkItem, ProjectIDs: []string{project.ID}, FailedCount: 1},
	}

	svc := NewService(repo, func() string { return actionItem.ID }, func() time.Time { return now }, ServiceConfig{
		EmbeddingLifecycle: lifecycle,
		EmbeddingRuntime: EmbeddingRuntimeConfig{
			Enabled:        true,
			Provider:       "fantasy",
			Model:          "mini",
			Dimensions:     3,
			ModelSignature: BuildEmbeddingModelSignature("fantasy", "mini", "", 3),
			MaxAttempts:    5,
		},
	})

	result, err := svc.ReindexEmbeddings(context.Background(), ReindexEmbeddingsInput{
		ProjectID:        project.ID,
		Wait:             true,
		WaitPollInterval: time.Millisecond,
		WaitTimeout:      25 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("ReindexEmbeddings() error = %v", err)
	}
	if result.Completed {
		t.Fatalf("Completed = true, want false when failed rows remain")
	}
	if result.FailedCount != 1 {
		t.Fatalf("failed count = %d, want 1", result.FailedCount)
	}
	if result.TimedOut {
		t.Fatalf("TimedOut = true, want false when steady state is failed")
	}
}

// TestPrepareEmbeddingsLifecycleRecoversExpiredClaims verifies startup reconciliation revives abandoned running work.
func TestPrepareEmbeddingsLifecycleRecoversExpiredClaims(t *testing.T) {
	lifecycle := newFakeEmbeddingLifecycleStore()
	lifecycle.recoveredCount = 2

	err := PrepareEmbeddingsLifecycle(context.Background(), lifecycle, EmbeddingRuntimeConfig{
		Enabled: true,
	})
	if err != nil {
		t.Fatalf("PrepareEmbeddingsLifecycle() error = %v", err)
	}
	if lifecycle.recoverCalls != 1 {
		t.Fatalf("recover calls = %d, want 1", lifecycle.recoverCalls)
	}
}

// TestEmbeddingWorkerProcessOnceRecoversExpiredClaims verifies steady-state polling also revives abandoned running work.
func TestEmbeddingWorkerProcessOnceRecoversExpiredClaims(t *testing.T) {
	repo := newFakeRepo()
	now := time.Date(2026, 3, 29, 20, 0, 0, 0, time.UTC)
	lifecycle := newFakeEmbeddingLifecycleStore()
	lifecycle.recoveredCount = 1

	worker := NewEmbeddingWorker(
		repo,
		lifecycle,
		&fakeEmbeddingGenerator{vectors: [][]float32{{0.1, 0.2, 0.3}}},
		&fakeActionItemSearchIndex{},
		func() time.Time { return now },
		EmbeddingRuntimeConfig{
			Enabled:      true,
			WorkerID:     "worker-1",
			ClaimTTL:     30 * time.Second,
			PollInterval: time.Second,
		},
	)
	if err := worker.processOnce(context.Background()); err != nil {
		t.Fatalf("processOnce() error = %v", err)
	}
	if lifecycle.recoverCalls != 1 {
		t.Fatalf("recover calls = %d, want 1", lifecycle.recoverCalls)
	}
}
