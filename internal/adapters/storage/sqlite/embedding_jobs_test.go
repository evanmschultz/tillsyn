package sqlite

import (
	"context"
	"testing"
	"time"

	"github.com/evanmschultz/tillsyn/internal/app"
	"github.com/evanmschultz/tillsyn/internal/domain"
)

func TestRepositoryEmbeddingJobLifecycleTransitions(t *testing.T) {
	ctx := context.Background()
	repo, err := OpenInMemory()
	if err != nil {
		t.Fatalf("OpenInMemory() error = %v", err)
	}
	t.Cleanup(func() {
		_ = repo.Close()
	})

	now := time.Date(2026, 3, 29, 12, 0, 0, 0, time.UTC)
	project, column := mustSeedEmbeddingScope(t, repo, now, "p-embeddings-1")
	actionItemReady := mustSeedEmbeddingActionItem(t, repo, project.ID, column.ID, "actionItem-ready", 0, now, domain.ActionItemMetadata{
		Objective:          "Ship a safe embeddings lifecycle",
		AcceptanceCriteria: "Storage records lifecycle transitions",
		ValidationPlan:     "Run sqlite tests",
		RiskNotes:          "Keep vector rows separate from lifecycle state",
	}, []string{"lifecycle", "embeddings"})
	actionItemStale := mustSeedEmbeddingActionItem(t, repo, project.ID, column.ID, "actionItem-stale", 1, now, domain.ActionItemMetadata{
		Objective:      "Reindex stale work",
		BlockedReason:  "Needs operator review",
		ValidationPlan: "Confirm stale transition",
	}, []string{"stale"})

	readyHash := hashEmbeddingContent(buildSQLiteActionItemEmbeddingContent(actionItemReady))
	readyRecord, changed, err := repo.UpsertEmbeddingJob(ctx, EmbeddingJobUpsertInput{
		SubjectType:   "actionItem",
		SubjectID:     actionItemReady.ID,
		ProjectID:     project.ID,
		DesiredHash:   readyHash,
		ModelProvider: "fantasy",
		ModelName:     "mini",
		ModelSig:      "fantasy/mini/3",
		ModelDims:     3,
		MaxAttempts:   3,
	})
	if err != nil {
		t.Fatalf("UpsertEmbeddingJob() error = %v", err)
	}
	if !changed {
		t.Fatalf("expected first upsert to change state")
	}
	if readyRecord.Status != EmbeddingJobStatusPending {
		t.Fatalf("status = %s, want %s", readyRecord.Status, EmbeddingJobStatusPending)
	}
	if readyRecord.AttemptCount != 0 {
		t.Fatalf("attempt_count = %d, want 0", readyRecord.AttemptCount)
	}

	idempotentRecord, changed, err := repo.UpsertEmbeddingJob(ctx, EmbeddingJobUpsertInput{
		SubjectType:   "actionItem",
		SubjectID:     actionItemReady.ID,
		ProjectID:     project.ID,
		DesiredHash:   readyHash,
		ModelProvider: "fantasy",
		ModelName:     "mini",
		ModelSig:      "fantasy/mini/3",
		ModelDims:     3,
		MaxAttempts:   3,
	})
	if err != nil {
		t.Fatalf("second UpsertEmbeddingJob() error = %v", err)
	}
	if changed {
		t.Fatalf("expected identical upsert to be idempotent")
	}
	if idempotentRecord.SubjectID != actionItemReady.ID {
		t.Fatalf("unexpected subject id %q", idempotentRecord.SubjectID)
	}

	claimNow := time.Now().UTC().Add(1 * time.Second)
	claimed, found, err := repo.ClaimNextEmbeddingJob(ctx, EmbeddingJobClaimNextInput{
		ProjectID:   project.ID,
		SubjectType: "actionItem",
		WorkerID:    "worker-a",
		ClaimTTL:    15 * time.Minute,
	}, claimNow)
	if err != nil {
		t.Fatalf("ClaimNextEmbeddingJob() error = %v", err)
	}
	if !found {
		t.Fatalf("expected one claimable row")
	}
	if claimed.Status != EmbeddingJobStatusRunning {
		t.Fatalf("status = %s, want running", claimed.Status)
	}
	if claimed.AttemptCount != 1 {
		t.Fatalf("attempt_count = %d, want 1", claimed.AttemptCount)
	}
	if claimed.ClaimedBy != "worker-a" {
		t.Fatalf("claimed_by = %q, want worker-a", claimed.ClaimedBy)
	}

	heartbeat, changed, err := repo.HeartbeatEmbeddingJob(ctx, EmbeddingJobHeartbeatInput{
		SubjectType: "actionItem",
		SubjectID:   actionItemReady.ID,
		WorkerID:    "worker-a",
		ClaimTTL:    20 * time.Minute,
	}, claimNow.Add(2*time.Minute))
	if err != nil {
		t.Fatalf("HeartbeatEmbeddingJob() error = %v", err)
	}
	if !changed {
		t.Fatalf("expected heartbeat to update the row")
	}
	if heartbeat.LastHeartbeatAt == nil {
		t.Fatalf("expected last_heartbeat_at to be set")
	}
	if heartbeat.ClaimExpiresAt == nil {
		t.Fatalf("expected claim_expires_at to be extended")
	}

	completed, changed, err := repo.CompleteEmbeddingJob(ctx, EmbeddingJobCompleteInput{
		SubjectType:          "actionItem",
		SubjectID:            actionItemReady.ID,
		WorkerID:             "worker-a",
		ProcessedContentHash: readyHash,
		ProcessedModelSig:    "fantasy/mini/3",
	})
	if err != nil {
		t.Fatalf("CompleteEmbeddingJob() error = %v", err)
	}
	if !changed {
		t.Fatalf("expected completion to update the row")
	}
	if completed.Status != EmbeddingJobStatusReady {
		t.Fatalf("status = %s, want ready", completed.Status)
	}
	if completed.IndexedContentHash != readyHash {
		t.Fatalf("indexed hash = %q, want %q", completed.IndexedContentHash, readyHash)
	}
	if completed.AttemptCount != 1 {
		t.Fatalf("attempt_count = %d, want 1", completed.AttemptCount)
	}

	staleHash := hashEmbeddingContent(buildSQLiteActionItemEmbeddingContent(actionItemStale))
	staleUpsert, changed, err := repo.UpsertEmbeddingJob(ctx, EmbeddingJobUpsertInput{
		SubjectType:   "actionItem",
		SubjectID:     actionItemStale.ID,
		ProjectID:     project.ID,
		DesiredHash:   staleHash,
		ModelProvider: "fantasy",
		ModelName:     "mini",
		ModelSig:      "fantasy/mini/3",
		ModelDims:     3,
		MaxAttempts:   3,
	})
	if err != nil {
		t.Fatalf("UpsertEmbeddingJob(stale) error = %v", err)
	}
	if !changed {
		t.Fatalf("expected stale actionItem upsert to create a row")
	}
	if staleUpsert.Status != EmbeddingJobStatusPending {
		t.Fatalf("status = %s, want pending", staleUpsert.Status)
	}

	staleClaim, found, err := repo.ClaimNextEmbeddingJob(ctx, EmbeddingJobClaimNextInput{
		ProjectID:   project.ID,
		SubjectType: "actionItem",
		WorkerID:    "worker-b",
		ClaimTTL:    15 * time.Minute,
	}, time.Now().UTC().Add(1*time.Second))
	if err != nil {
		t.Fatalf("ClaimNextEmbeddingJob(stale) error = %v", err)
	}
	if !found {
		t.Fatalf("expected stale row to be claimable")
	}
	staled, changed, err := repo.CompleteEmbeddingJob(ctx, EmbeddingJobCompleteInput{
		SubjectType:          "actionItem",
		SubjectID:            staleClaim.SubjectID,
		WorkerID:             "worker-b",
		ProcessedContentHash: "mismatched-hash",
		ProcessedModelSig:    "fantasy/mini/4",
	})
	if err != nil {
		t.Fatalf("CompleteEmbeddingJob(stale) error = %v", err)
	}
	if !changed {
		t.Fatalf("expected mismatched completion to mark stale")
	}
	if staled.Status != EmbeddingJobStatusStale {
		t.Fatalf("status = %s, want stale", staled.Status)
	}
	if staled.StaleReason == "" {
		t.Fatalf("expected stale reason to be populated")
	}

	summary, err := repo.SummarizeEmbeddingJobs(ctx, EmbeddingJobListFilter{ProjectID: project.ID})
	if err != nil {
		t.Fatalf("SummarizeEmbeddingJobs() error = %v", err)
	}
	if summary.Total != 2 || summary.Ready != 1 || summary.Stale != 1 {
		t.Fatalf("summary = %#v, want total 2 ready 1 stale 1", summary)
	}

	list, err := repo.ListEmbeddingJobs(ctx, EmbeddingJobListFilter{
		ProjectID: project.ID,
		Statuses:  []EmbeddingJobStatus{EmbeddingJobStatusReady, EmbeddingJobStatusStale},
	})
	if err != nil {
		t.Fatalf("ListEmbeddingJobs() error = %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 lifecycle rows, got %d", len(list))
	}
}

func TestRepositoryEmbeddingJobBackfillRetryRecoveryAndStaleMarking(t *testing.T) {
	ctx := context.Background()
	repo, err := OpenInMemory()
	if err != nil {
		t.Fatalf("OpenInMemory() error = %v", err)
	}
	t.Cleanup(func() {
		_ = repo.Close()
	})

	now := time.Date(2026, 3, 29, 13, 0, 0, 0, time.UTC)
	project, column := mustSeedEmbeddingScope(t, repo, now, "p-embeddings-2")
	actionItemBackfillA := mustSeedEmbeddingActionItem(t, repo, project.ID, column.ID, "actionItem-backfill-a", 0, now, domain.ActionItemMetadata{
		Objective:          "Backfill the queue",
		AcceptanceCriteria: "Lifecycle rows exist for each actionItem",
		ValidationPlan:     "Run backfill and inspect summary",
	}, []string{"queue", "backfill"})
	_ = mustSeedEmbeddingActionItem(t, repo, project.ID, column.ID, "actionItem-backfill-b", 1, now, domain.ActionItemMetadata{
		Objective:      "Retry failed embeddings",
		RiskNotes:      "Retry budgets must be bounded",
		BlockedReason:  "Awaiting provider response",
		ValidationPlan: "Claim, fail, and recover",
	}, []string{"retry"})

	count, err := repo.BackfillActionItemEmbeddingJobs(ctx, EmbeddingActionItemBackfillInput{
		ProjectID:       project.ID,
		IncludeArchived: false,
		ModelProvider:   "fantasy",
		ModelName:       "mini",
		ModelSig:        "fantasy/mini/3",
		ModelDims:       3,
		MaxAttempts:     4,
	})
	if err != nil {
		t.Fatalf("BackfillActionItemEmbeddingJobs() error = %v", err)
	}
	if count != 2 {
		t.Fatalf("backfill count = %d, want 2", count)
	}

	backfillHash := hashEmbeddingContent(buildSQLiteActionItemEmbeddingContent(actionItemBackfillA))
	backfillRow, err := repo.GetEmbeddingJob(ctx, "actionItem", actionItemBackfillA.ID)
	if err != nil {
		t.Fatalf("GetEmbeddingJob() error = %v", err)
	}
	if backfillRow.DesiredContentHash != backfillHash {
		t.Fatalf("desired hash = %q, want %q", backfillRow.DesiredContentHash, backfillHash)
	}

	summary, err := repo.SummarizeEmbeddingJobs(ctx, EmbeddingJobListFilter{ProjectID: project.ID, SubjectType: "actionItem"})
	if err != nil {
		t.Fatalf("SummarizeEmbeddingJobs() error = %v", err)
	}
	if summary.Pending != 2 || summary.Total != 2 {
		t.Fatalf("summary = %#v, want total 2 pending 2", summary)
	}
	secondBackfillCount, err := repo.BackfillActionItemEmbeddingJobs(ctx, EmbeddingActionItemBackfillInput{
		ProjectID:       project.ID,
		IncludeArchived: false,
		ModelProvider:   "fantasy",
		ModelName:       "mini",
		ModelSig:        "fantasy/mini/3",
		ModelDims:       3,
		MaxAttempts:     4,
	})
	if err != nil {
		t.Fatalf("BackfillActionItemEmbeddingJobs(idempotent) error = %v", err)
	}
	if secondBackfillCount != 0 {
		t.Fatalf("idempotent backfill count = %d, want 0", secondBackfillCount)
	}

	list, err := repo.ListEmbeddingJobs(ctx, EmbeddingJobListFilter{
		ProjectID:   project.ID,
		SubjectType: "actionItem",
		Statuses:    []EmbeddingJobStatus{EmbeddingJobStatusPending},
		Limit:       10,
	})
	if err != nil {
		t.Fatalf("ListEmbeddingJobs() error = %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 pending rows, got %d", len(list))
	}

	claimed, found, err := repo.ClaimNextEmbeddingJob(ctx, EmbeddingJobClaimNextInput{
		ProjectID:   project.ID,
		SubjectType: "actionItem",
		WorkerID:    "worker-retry",
		ClaimTTL:    30 * time.Second,
	}, time.Now().UTC().Add(1*time.Second))
	if err != nil {
		t.Fatalf("ClaimNextEmbeddingJob() error = %v", err)
	}
	if !found {
		t.Fatalf("expected a claimable row")
	}
	failStart := time.Now()
	failed, changed, err := repo.FailEmbeddingJob(ctx, EmbeddingJobFailInput{
		SubjectType:  "actionItem",
		SubjectID:    claimed.SubjectID,
		WorkerID:     "worker-retry",
		ErrorCode:    "provider_unavailable",
		ErrorMessage: "temporary outage",
		ErrorSummary: "provider unavailable",
		Retryable:    true,
		MaxAttempts:  4,
	})
	if err != nil {
		t.Fatalf("FailEmbeddingJob() error = %v", err)
	}
	if !changed {
		t.Fatalf("expected failure handling to update the row")
	}
	if failed.Status != EmbeddingJobStatusPending {
		t.Fatalf("status = %s, want pending retry", failed.Status)
	}
	if failed.RetryCount != 1 {
		t.Fatalf("retry_count = %d, want 1", failed.RetryCount)
	}
	if failed.AttemptCount != 1 {
		t.Fatalf("attempt_count = %d, want 1", failed.AttemptCount)
	}
	if failed.NextAttemptAt == nil || failed.NextAttemptAt.Before(failStart.Add(9*time.Second)) {
		t.Fatalf("next_attempt_at = %v, want at least ~10s in the future", failed.NextAttemptAt)
	}

	secondClaim, found, err := repo.ClaimNextEmbeddingJob(ctx, EmbeddingJobClaimNextInput{
		ProjectID:   project.ID,
		SubjectType: "actionItem",
		WorkerID:    "worker-retry",
		ClaimTTL:    30 * time.Second,
	}, time.Now().UTC().Add(1*time.Second))
	if err != nil {
		t.Fatalf("ClaimNextEmbeddingJob(second actionItem) error = %v", err)
	}
	if !found {
		t.Fatalf("expected second pending row to be claimable")
	}
	if secondClaim.SubjectID == claimed.SubjectID {
		t.Fatalf("expected second claim to target a different row")
	}
	secondTerminal, changed, err := repo.FailEmbeddingJob(ctx, EmbeddingJobFailInput{
		SubjectType:  "actionItem",
		SubjectID:    secondClaim.SubjectID,
		WorkerID:     "worker-retry",
		ErrorCode:    "provider_unavailable",
		ErrorMessage: "temporary outage",
		ErrorSummary: "provider unavailable",
		Retryable:    false,
		MaxAttempts:  4,
	})
	if err != nil {
		t.Fatalf("FailEmbeddingJob(second actionItem) error = %v", err)
	}
	if !changed {
		t.Fatalf("expected second actionItem terminal failure to update the row")
	}
	if secondTerminal.Status != EmbeddingJobStatusFailed {
		t.Fatalf("status = %s, want failed", secondTerminal.Status)
	}

	if _, found, err = repo.ClaimNextEmbeddingJob(ctx, EmbeddingJobClaimNextInput{
		ProjectID:   project.ID,
		SubjectType: "actionItem",
		WorkerID:    "worker-retry",
		ClaimTTL:    30 * time.Second,
	}, time.Now().UTC().Add(5*time.Second)); err != nil {
		t.Fatalf("ClaimNextEmbeddingJob(retry too early) error = %v", err)
	} else if found {
		t.Fatalf("expected retry to remain hidden until next_attempt_at")
	}

	retryClaim, found, err := repo.ClaimNextEmbeddingJob(ctx, EmbeddingJobClaimNextInput{
		ProjectID:   project.ID,
		SubjectType: "actionItem",
		WorkerID:    "worker-retry",
		ClaimTTL:    30 * time.Second,
	}, time.Now().UTC().Add(11*time.Second))
	if err != nil {
		t.Fatalf("ClaimNextEmbeddingJob(retry due) error = %v", err)
	}
	if !found {
		t.Fatalf("expected retry to become claimable after backoff")
	}
	terminal, changed, err := repo.FailEmbeddingJob(ctx, EmbeddingJobFailInput{
		SubjectType:  "actionItem",
		SubjectID:    retryClaim.SubjectID,
		WorkerID:     "worker-retry",
		ErrorCode:    "provider_unavailable",
		ErrorMessage: "temporary outage",
		ErrorSummary: "provider unavailable",
		Retryable:    false,
		MaxAttempts:  4,
	})
	if err != nil {
		t.Fatalf("FailEmbeddingJob(terminal) error = %v", err)
	}
	if !changed {
		t.Fatalf("expected terminal failure to update the row")
	}
	if terminal.Status != EmbeddingJobStatusFailed {
		t.Fatalf("status = %s, want failed", terminal.Status)
	}
}

func TestRepositoryEmbeddingJobManualStaleMarking(t *testing.T) {
	ctx := context.Background()
	repo, err := OpenInMemory()
	if err != nil {
		t.Fatalf("OpenInMemory() error = %v", err)
	}
	t.Cleanup(func() {
		_ = repo.Close()
	})

	now := time.Date(2026, 3, 29, 14, 0, 0, 0, time.UTC)
	project, column := mustSeedEmbeddingScope(t, repo, now, "p-embeddings-3")
	actionItem := mustSeedEmbeddingActionItem(t, repo, project.ID, column.ID, "actionItem-manual-stale", 0, now, domain.ActionItemMetadata{
		Objective:          "Let operators mark stale rows",
		AcceptanceCriteria: "Manual stale marks clear the vector row",
		ValidationPlan:     "Mark stale and inspect status",
	}, []string{"manual"})

	if _, changed, err := repo.UpsertEmbeddingJob(ctx, EmbeddingJobUpsertInput{
		SubjectType:   "actionItem",
		SubjectID:     actionItem.ID,
		ProjectID:     project.ID,
		DesiredHash:   hashEmbeddingContent(buildSQLiteActionItemEmbeddingContent(actionItem)),
		ModelProvider: "fantasy",
		ModelName:     "mini",
		ModelSig:      "fantasy/mini/3",
		ModelDims:     3,
		MaxAttempts:   4,
	}); err != nil {
		t.Fatalf("UpsertEmbeddingJob() error = %v", err)
	} else if !changed {
		t.Fatalf("expected seed upsert to create a row")
	}

	marked, changed, err := repo.MarkEmbeddingJobStale(ctx, EmbeddingJobStaleInput{
		SubjectType: "actionItem",
		SubjectID:   actionItem.ID,
		Reason:      "operator requested reindex",
		PurgeVector: true,
	})
	if err != nil {
		t.Fatalf("MarkEmbeddingJobStale() error = %v", err)
	}
	if !changed {
		t.Fatalf("expected manual stale mark to update the row")
	}
	if marked.Status != EmbeddingJobStatusStale {
		t.Fatalf("status = %s, want stale", marked.Status)
	}
	if marked.StaleReason != "operator requested reindex" {
		t.Fatalf("stale_reason = %q, want operator requested reindex", marked.StaleReason)
	}
}

func TestRepositoryEmbeddingJobStartupRecovery(t *testing.T) {
	ctx := context.Background()
	repo, err := OpenInMemory()
	if err != nil {
		t.Fatalf("OpenInMemory() error = %v", err)
	}
	t.Cleanup(func() {
		_ = repo.Close()
	})

	now := time.Date(2026, 3, 29, 15, 0, 0, 0, time.UTC)
	project, column := mustSeedEmbeddingScope(t, repo, now, "p-embeddings-4")
	actionItem := mustSeedEmbeddingActionItem(t, repo, project.ID, column.ID, "actionItem-recovery", 0, now, domain.ActionItemMetadata{
		Objective:      "Recover stuck claims",
		ValidationPlan: "Expire the claim and recover it",
	}, []string{"recovery"})

	if _, changed, err := repo.UpsertEmbeddingJob(ctx, EmbeddingJobUpsertInput{
		SubjectType:   "actionItem",
		SubjectID:     actionItem.ID,
		ProjectID:     project.ID,
		DesiredHash:   hashEmbeddingContent(buildSQLiteActionItemEmbeddingContent(actionItem)),
		ModelProvider: "fantasy",
		ModelName:     "mini",
		ModelSig:      "fantasy/mini/3",
		ModelDims:     3,
		MaxAttempts:   4,
	}); err != nil {
		t.Fatalf("UpsertEmbeddingJob() error = %v", err)
	} else if !changed {
		t.Fatalf("expected seed upsert to create a row")
	}

	claimed, found, err := repo.ClaimNextEmbeddingJob(ctx, EmbeddingJobClaimNextInput{
		ProjectID:   project.ID,
		SubjectType: "actionItem",
		WorkerID:    "worker-recovery",
		ClaimTTL:    1 * time.Millisecond,
	}, time.Now().UTC().Add(1*time.Second))
	if err != nil {
		t.Fatalf("ClaimNextEmbeddingJob() error = %v", err)
	}
	if !found {
		t.Fatalf("expected recovery target to be claimable")
	}
	if claimed.Status != EmbeddingJobStatusRunning {
		t.Fatalf("status = %s, want running", claimed.Status)
	}

	recovered, err := repo.RecoverExpiredEmbeddingJobs(ctx, time.Now().UTC().Add(2*time.Second))
	if err != nil {
		t.Fatalf("RecoverExpiredEmbeddingJobs() error = %v", err)
	}
	if len(recovered) != 1 {
		t.Fatalf("recovered = %d, want 1", len(recovered))
	}
	recoveredRow, err := repo.GetEmbeddingJob(ctx, "actionItem", actionItem.ID)
	if err != nil {
		t.Fatalf("GetEmbeddingJob() error = %v", err)
	}
	if recoveredRow.Status != EmbeddingJobStatusPending {
		t.Fatalf("status = %s, want pending after recovery", recoveredRow.Status)
	}
	if recoveredRow.ClaimedBy != "" {
		t.Fatalf("claimed_by = %q, want empty after recovery", recoveredRow.ClaimedBy)
	}
}

func TestRepositoryEmbeddingJobMixedSubjectFamilies(t *testing.T) {
	ctx := context.Background()
	repo, err := OpenInMemory()
	if err != nil {
		t.Fatalf("OpenInMemory() error = %v", err)
	}
	t.Cleanup(func() {
		_ = repo.Close()
	})

	now := time.Date(2026, 3, 29, 16, 0, 0, 0, time.UTC)
	project, column := mustSeedEmbeddingScope(t, repo, now, "p-embeddings-mixed")
	actionItem := mustSeedEmbeddingActionItem(t, repo, project.ID, column.ID, "actionItem-mixed", 0, now, domain.ActionItemMetadata{
		Objective:      "Keep lifecycle state mixed but stable",
		ValidationPlan: "Exercise all subject families",
	}, []string{"mixed"})

	threadSubjectID := app.BuildThreadContextSubjectID(domain.CommentTarget{
		ProjectID:  project.ID,
		TargetType: domain.CommentTargetTypeActionItem,
		TargetID:   actionItem.ID,
	})

	seedRows := []EmbeddingJobUpsertInput{
		{
			SubjectType:   "actionItem",
			SubjectID:     actionItem.ID,
			ProjectID:     project.ID,
			DesiredHash:   hashEmbeddingContent(buildSQLiteActionItemEmbeddingContent(actionItem)),
			ModelProvider: "fantasy",
			ModelName:     "mini",
			ModelSig:      "fantasy/mini/3",
			ModelDims:     3,
			MaxAttempts:   4,
		},
		{
			SubjectType:   "thread_context",
			SubjectID:     threadSubjectID,
			ProjectID:     project.ID,
			DesiredHash:   "hash-thread-context",
			ModelProvider: "fantasy",
			ModelName:     "mini",
			ModelSig:      "fantasy/mini/3",
			ModelDims:     3,
			MaxAttempts:   4,
		},
		{
			SubjectType:   "project_document",
			SubjectID:     project.ID,
			ProjectID:     project.ID,
			DesiredHash:   "hash-project-document",
			ModelProvider: "fantasy",
			ModelName:     "mini",
			ModelSig:      "fantasy/mini/3",
			ModelDims:     3,
			MaxAttempts:   4,
		},
	}
	for _, row := range seedRows {
		if _, changed, err := repo.UpsertEmbeddingJob(ctx, row); err != nil {
			t.Fatalf("UpsertEmbeddingJob(%s) error = %v", row.SubjectType, err)
		} else if !changed {
			t.Fatalf("expected seed upsert for %s to create a row", row.SubjectType)
		}
	}

	listed, err := repo.ListEmbeddings(ctx, app.EmbeddingListFilter{ProjectIDs: []string{project.ID}})
	if err != nil {
		t.Fatalf("ListEmbeddings() error = %v", err)
	}
	if len(listed) != 3 {
		t.Fatalf("expected 3 lifecycle rows, got %d", len(listed))
	}
	seen := map[app.EmbeddingSubjectType]bool{}
	for _, row := range listed {
		seen[row.SubjectType] = true
	}
	for _, subjectType := range []app.EmbeddingSubjectType{
		app.EmbeddingSubjectTypeWorkItem,
		app.EmbeddingSubjectTypeThreadContext,
		app.EmbeddingSubjectTypeProjectDocument,
	} {
		if !seen[subjectType] {
			t.Fatalf("missing subject type %s in list results %#v", subjectType, listed)
		}
	}

	staled, err := repo.MarkEmbeddingsStaleByModel(ctx, app.EmbeddingStaleByModelInput{
		ModelProvider:   "fantasy",
		ModelName:       "mini",
		ModelDimensions: 3,
		SubjectType:     app.EmbeddingSubjectTypeProjectDocument,
		ModelSignature:  "fantasy/mini/4",
		Reason:          "model_signature_changed",
		StaledAt:        now.Add(2 * time.Second),
	})
	if err != nil {
		t.Fatalf("MarkEmbeddingsStaleByModel() error = %v", err)
	}
	if len(staled) != 1 {
		t.Fatalf("expected 1 project document to stale, got %d", len(staled))
	}
	if staled[0].Status != app.EmbeddingLifecycleStale {
		t.Fatalf("project document status = %s, want stale", staled[0].Status)
	}

	summary, err := repo.SummarizeEmbeddings(ctx, app.EmbeddingListFilter{ProjectIDs: []string{project.ID}})
	if err != nil {
		t.Fatalf("SummarizeEmbeddings() error = %v", err)
	}
	if summary.PendingCount != 2 || summary.ReadyCount != 0 || summary.FailedCount != 0 || summary.StaleCount != 1 {
		t.Fatalf("summary = %#v, want pending 2 ready 0 failed 0 stale 1", summary)
	}
}

func mustSeedEmbeddingScope(t *testing.T, repo *Repository, now time.Time, projectID string) (domain.Project, domain.Column) {
	t.Helper()

	project, err := domain.NewProject(projectID, "Embeddings "+projectID, "Embeddings test project", now)
	if err != nil {
		t.Fatalf("NewProject() error = %v", err)
	}
	if err := repo.CreateProject(context.Background(), project); err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}
	column, err := domain.NewColumn(projectID+"-column", project.ID, "Inbox", 0, 0, now)
	if err != nil {
		t.Fatalf("NewColumn() error = %v", err)
	}
	if err := repo.CreateColumn(context.Background(), column); err != nil {
		t.Fatalf("CreateColumn() error = %v", err)
	}
	return project, column
}

func mustSeedEmbeddingActionItem(t *testing.T, repo *Repository, projectID, columnID, actionItemID string, position int, now time.Time, meta domain.ActionItemMetadata, labels []string) domain.ActionItem {
	t.Helper()

	actionItem, err := domain.NewActionItemForTest(domain.ActionItemInput{
		Kind:        domain.KindPlan,
		ID:          actionItemID,
		ProjectID:   projectID,
		ColumnID:    columnID,
		Position:    position,
		Title:       "ActionItem " + actionItemID,
		Description: "Description for " + actionItemID,
		Priority:    domain.PriorityMedium,
		Labels:      labels,
		Metadata:    meta,
	}, now)
	if err != nil {
		t.Fatalf("NewActionItem() error = %v", err)
	}
	if err := repo.CreateActionItem(context.Background(), actionItem); err != nil {
		t.Fatalf("CreateActionItem() error = %v", err)
	}
	return actionItem
}
