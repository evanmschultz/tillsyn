package sqlite

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/hylla/tillsyn/internal/domain"
)

// EmbeddingJobStatus identifies one persisted embeddings lifecycle state.
type EmbeddingJobStatus string

// Embedding job lifecycle states.
const (
	EmbeddingJobStatusPending EmbeddingJobStatus = "pending"
	EmbeddingJobStatusRunning EmbeddingJobStatus = "running"
	EmbeddingJobStatusReady   EmbeddingJobStatus = "ready"
	EmbeddingJobStatusFailed  EmbeddingJobStatus = "failed"
	EmbeddingJobStatusStale   EmbeddingJobStatus = "stale"
)

// defaultEmbeddingJobMaxAttempts defines the default terminal retry budget.
const defaultEmbeddingJobMaxAttempts = 5

// rollbackEmbeddingLifecycleTx rolls back one lifecycle transaction unless it has already committed.
func rollbackEmbeddingLifecycleTx(tx *sql.Tx, committed *bool) {
	if tx == nil || committed == nil || *committed {
		return
	}
	_ = tx.Rollback()
}

// EmbeddingJobRecord stores one persisted embeddings lifecycle row.
type EmbeddingJobRecord struct {
	SubjectType        string
	SubjectID          string
	ProjectID          string
	DesiredContentHash string
	IndexedContentHash string
	ModelProvider      string
	ModelName          string
	ModelDimensions    int64
	ModelSignature     string
	Status             EmbeddingJobStatus
	AttemptCount       int
	RetryCount         int
	MaxAttempts        int
	LastEnqueuedAt     time.Time
	LastStartedAt      *time.Time
	LastHeartbeatAt    *time.Time
	LastSucceededAt    *time.Time
	LastFailedAt       *time.Time
	NextAttemptAt      *time.Time
	ClaimedBy          string
	ClaimExpiresAt     *time.Time
	LastErrorCode      string
	LastErrorMessage   string
	LastErrorSummary   string
	StaleReason        string
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

// EmbeddingJobUpsertInput holds inputs for normal enqueue/upsert lifecycle transitions.
type EmbeddingJobUpsertInput struct {
	SubjectType   string
	SubjectID     string
	ProjectID     string
	DesiredHash   string
	ModelProvider string
	ModelName     string
	ModelSig      string
	ModelDims     int64
	MaxAttempts   int
	Force         bool
	ResetAttempts bool
	PurgeVector   bool
}

// EmbeddingJobClaimNextInput holds inputs for worker claim requests.
type EmbeddingJobClaimNextInput struct {
	ProjectID   string
	SubjectType string
	WorkerID    string
	ClaimTTL    time.Duration
}

// EmbeddingJobHeartbeatInput holds inputs for one claim heartbeat update.
type EmbeddingJobHeartbeatInput struct {
	SubjectType string
	SubjectID   string
	WorkerID    string
	ClaimTTL    time.Duration
}

// EmbeddingJobCompleteInput holds inputs for successful embedding completion.
type EmbeddingJobCompleteInput struct {
	SubjectType          string
	SubjectID            string
	WorkerID             string
	ProcessedContentHash string
	ProcessedModelSig    string
}

// EmbeddingJobFailInput holds inputs for embedding failure handling.
type EmbeddingJobFailInput struct {
	SubjectType  string
	SubjectID    string
	WorkerID     string
	ErrorCode    string
	ErrorMessage string
	ErrorSummary string
	Retryable    bool
	RetryAfter   time.Duration
	MaxAttempts  int
	PurgeVector  bool
}

// EmbeddingJobStaleInput holds inputs for stale-state marking.
type EmbeddingJobStaleInput struct {
	SubjectType string
	SubjectID   string
	Reason      string
	PurgeVector bool
}

// EmbeddingJobListFilter holds query fields for lifecycle inventory scans.
type EmbeddingJobListFilter struct {
	ProjectID   string
	SubjectType string
	Statuses    []EmbeddingJobStatus
	Limit       int
	Offset      int
}

// EmbeddingJobSummary stores aggregate lifecycle counts.
type EmbeddingJobSummary struct {
	Total   int
	Pending int
	Running int
	Ready   int
	Failed  int
	Stale   int
}

// EmbeddingTaskBackfillInput holds inputs for scanning tasks and seeding lifecycle rows.
type EmbeddingTaskBackfillInput struct {
	ProjectID       string
	IncludeArchived bool
	ModelProvider   string
	ModelName       string
	ModelSig        string
	ModelDims       int64
	MaxAttempts     int
	Force           bool
}

// UpsertEmbeddingJob enqueues or refreshes one lifecycle row idempotently.
func (r *Repository) UpsertEmbeddingJob(ctx context.Context, in EmbeddingJobUpsertInput) (EmbeddingJobRecord, bool, error) {
	return r.upsertEmbeddingJob(ctx, in, false)
}

// EnqueueEmbeddingJob forces one lifecycle row back into the pending queue.
func (r *Repository) EnqueueEmbeddingJob(ctx context.Context, in EmbeddingJobUpsertInput) (EmbeddingJobRecord, bool, error) {
	in.Force = true
	in.ResetAttempts = true
	in.PurgeVector = true
	return r.upsertEmbeddingJob(ctx, in, true)
}

// ClaimNextEmbeddingJob claims the next eligible lifecycle row for one worker.
func (r *Repository) ClaimNextEmbeddingJob(ctx context.Context, in EmbeddingJobClaimNextInput, now time.Time) (EmbeddingJobRecord, bool, error) {
	subjectType := normalizeEmbeddingSubjectType(in.SubjectType)
	workerID := strings.TrimSpace(in.WorkerID)
	if subjectType == "" {
		return EmbeddingJobRecord{}, false, domain.ErrInvalidID
	}
	if workerID == "" {
		return EmbeddingJobRecord{}, false, domain.ErrInvalidID
	}
	claimTTL := in.ClaimTTL
	if claimTTL <= 0 {
		claimTTL = 5 * time.Minute
	}
	now = now.UTC()
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return EmbeddingJobRecord{}, false, err
	}
	committed := false
	defer rollbackEmbeddingLifecycleTx(tx, &committed)

	row, found, err := r.selectNextEligibleEmbeddingJob(ctx, tx, EmbeddingJobListFilter{
		ProjectID:   strings.TrimSpace(in.ProjectID),
		SubjectType: subjectType,
		Statuses:    []EmbeddingJobStatus{EmbeddingJobStatusPending, EmbeddingJobStatusStale},
		Limit:       1,
	}, now)
	if err != nil {
		return EmbeddingJobRecord{}, false, err
	}
	if !found {
		return EmbeddingJobRecord{}, false, nil
	}
	claimExpiresAt := now.Add(claimTTL)
	res, err := tx.ExecContext(ctx, `
		UPDATE embedding_jobs
		SET status = ?, attempt_count = attempt_count + 1, claimed_by = ?, claim_expires_at = ?, last_started_at = ?, last_heartbeat_at = ?, updated_at = ?
		WHERE subject_type = ? AND subject_id = ? AND status = ? AND COALESCE(next_attempt_at, ?) <= ?
	`,
		string(EmbeddingJobStatusRunning),
		workerID,
		ts(claimExpiresAt),
		ts(now),
		ts(now),
		ts(now),
		row.SubjectType,
		row.SubjectID,
		string(row.Status),
		ts(now),
		ts(now),
	)
	if err != nil {
		return EmbeddingJobRecord{}, false, err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return EmbeddingJobRecord{}, false, err
	}
	if affected == 0 {
		return EmbeddingJobRecord{}, false, nil
	}
	claimed, err := r.getEmbeddingJobByKey(ctx, tx, row.SubjectType, row.SubjectID)
	if err != nil {
		return EmbeddingJobRecord{}, false, err
	}
	if err = tx.Commit(); err != nil {
		return EmbeddingJobRecord{}, false, err
	}
	committed = true
	return claimed, true, nil
}

// HeartbeatEmbeddingJob refreshes one active worker claim.
func (r *Repository) HeartbeatEmbeddingJob(ctx context.Context, in EmbeddingJobHeartbeatInput, now time.Time) (EmbeddingJobRecord, bool, error) {
	subjectType := normalizeEmbeddingSubjectType(in.SubjectType)
	subjectID := strings.TrimSpace(in.SubjectID)
	workerID := strings.TrimSpace(in.WorkerID)
	if subjectType == "" || subjectID == "" || workerID == "" {
		return EmbeddingJobRecord{}, false, domain.ErrInvalidID
	}
	claimTTL := in.ClaimTTL
	if claimTTL <= 0 {
		claimTTL = 5 * time.Minute
	}
	now = now.UTC()
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return EmbeddingJobRecord{}, false, err
	}
	committed := false
	defer rollbackEmbeddingLifecycleTx(tx, &committed)

	current, err := r.getEmbeddingJobByKey(ctx, tx, subjectType, subjectID)
	if err != nil {
		return EmbeddingJobRecord{}, false, err
	}
	if current.Status != EmbeddingJobStatusRunning || strings.TrimSpace(current.ClaimedBy) != workerID {
		return current, false, nil
	}
	claimExpiresAt := now.Add(claimTTL)
	res, err := tx.ExecContext(ctx, `
		UPDATE embedding_jobs
		SET last_heartbeat_at = ?, claim_expires_at = ?, updated_at = ?
		WHERE subject_type = ? AND subject_id = ? AND claimed_by = ? AND status = ?
	`,
		ts(now),
		ts(claimExpiresAt),
		ts(now),
		subjectType,
		subjectID,
		workerID,
		string(EmbeddingJobStatusRunning),
	)
	if err != nil {
		return EmbeddingJobRecord{}, false, err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return EmbeddingJobRecord{}, false, err
	}
	if affected == 0 {
		return current, false, nil
	}
	updated, err := r.getEmbeddingJobByKey(ctx, tx, subjectType, subjectID)
	if err != nil {
		return EmbeddingJobRecord{}, false, err
	}
	if err = tx.Commit(); err != nil {
		return EmbeddingJobRecord{}, false, err
	}
	committed = true
	return updated, true, nil
}

// CompleteEmbeddingJob records one successful embedding write.
func (r *Repository) CompleteEmbeddingJob(ctx context.Context, in EmbeddingJobCompleteInput) (EmbeddingJobRecord, bool, error) {
	subjectType := normalizeEmbeddingSubjectType(in.SubjectType)
	subjectID := strings.TrimSpace(in.SubjectID)
	workerID := strings.TrimSpace(in.WorkerID)
	if subjectType == "" || subjectID == "" || workerID == "" {
		return EmbeddingJobRecord{}, false, domain.ErrInvalidID
	}
	processedHash := strings.TrimSpace(in.ProcessedContentHash)
	processedSignature := strings.TrimSpace(in.ProcessedModelSig)
	if processedHash == "" || processedSignature == "" {
		return EmbeddingJobRecord{}, false, domain.ErrInvalidID
	}
	now := time.Now().UTC()
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return EmbeddingJobRecord{}, false, err
	}
	committed := false
	defer rollbackEmbeddingLifecycleTx(tx, &committed)

	current, err := r.getEmbeddingJobByKey(ctx, tx, subjectType, subjectID)
	if err != nil {
		return EmbeddingJobRecord{}, false, err
	}
	if current.Status != EmbeddingJobStatusRunning || strings.TrimSpace(current.ClaimedBy) != workerID {
		return current, false, nil
	}
	if current.DesiredContentHash != processedHash || current.ModelSignature != processedSignature {
		if err := r.deleteEmbeddingVectorForSubject(ctx, tx, subjectType, subjectID); err != nil {
			return EmbeddingJobRecord{}, false, err
		}
		staleReason := staleEmbeddingReason(current.DesiredContentHash, current.ModelSignature, processedHash, processedSignature)
		if err := r.updateEmbeddingJobStatus(ctx, tx, subjectType, subjectID, EmbeddingJobStatusStale, now, func(row *EmbeddingJobRecord) {
			row.IndexedContentHash = ""
			row.ClaimedBy = ""
			row.ClaimExpiresAt = nil
			row.StaleReason = staleReason
			row.LastErrorCode = ""
			row.LastErrorMessage = ""
			row.LastErrorSummary = ""
			row.NextAttemptAt = ptrTime(now)
			row.LastSucceededAt = nil
			row.LastFailedAt = nil
			row.LastHeartbeatAt = ptrTime(now)
		}); err != nil {
			return EmbeddingJobRecord{}, false, err
		}
		updated, err := r.getEmbeddingJobByKey(ctx, tx, subjectType, subjectID)
		if err != nil {
			return EmbeddingJobRecord{}, false, err
		}
		if err = tx.Commit(); err != nil {
			return EmbeddingJobRecord{}, false, err
		}
		committed = true
		return updated, true, nil
	}
	res, err := tx.ExecContext(ctx, `
		UPDATE embedding_jobs
		SET status = ?, indexed_content_hash = ?, claimed_by = ?, claim_expires_at = NULL, last_succeeded_at = ?, last_failed_at = NULL,
			last_error_code = '', last_error_message = '', last_error_summary = '', stale_reason = '', next_attempt_at = NULL, updated_at = ?
		WHERE subject_type = ? AND subject_id = ? AND claimed_by = ? AND status = ?
	`,
		string(EmbeddingJobStatusReady),
		processedHash,
		"",
		ts(now),
		ts(now),
		subjectType,
		subjectID,
		workerID,
		string(EmbeddingJobStatusRunning),
	)
	if err != nil {
		return EmbeddingJobRecord{}, false, err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return EmbeddingJobRecord{}, false, err
	}
	if affected == 0 {
		return current, false, nil
	}
	updated, err := r.getEmbeddingJobByKey(ctx, tx, subjectType, subjectID)
	if err != nil {
		return EmbeddingJobRecord{}, false, err
	}
	if err = tx.Commit(); err != nil {
		return EmbeddingJobRecord{}, false, err
	}
	committed = true
	return updated, true, nil
}

// FailEmbeddingJob records one failed embedding attempt and schedules a retry when allowed.
func (r *Repository) FailEmbeddingJob(ctx context.Context, in EmbeddingJobFailInput) (EmbeddingJobRecord, bool, error) {
	subjectType := normalizeEmbeddingSubjectType(in.SubjectType)
	subjectID := strings.TrimSpace(in.SubjectID)
	workerID := strings.TrimSpace(in.WorkerID)
	if subjectType == "" || subjectID == "" || workerID == "" {
		return EmbeddingJobRecord{}, false, domain.ErrInvalidID
	}
	now := time.Now().UTC()
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return EmbeddingJobRecord{}, false, err
	}
	committed := false
	defer rollbackEmbeddingLifecycleTx(tx, &committed)

	current, err := r.getEmbeddingJobByKey(ctx, tx, subjectType, subjectID)
	if err != nil {
		return EmbeddingJobRecord{}, false, err
	}
	if current.Status != EmbeddingJobStatusRunning || strings.TrimSpace(current.ClaimedBy) != workerID {
		return current, false, nil
	}
	if err := r.deleteEmbeddingVectorForSubject(ctx, tx, subjectType, subjectID); err != nil {
		return EmbeddingJobRecord{}, false, err
	}
	maxAttempts := in.MaxAttempts
	if maxAttempts <= 0 {
		maxAttempts = current.MaxAttempts
	}
	if maxAttempts <= 0 {
		maxAttempts = defaultEmbeddingJobMaxAttempts
	}
	retryable := in.Retryable && current.AttemptCount < maxAttempts
	nextAttemptAt := (*time.Time)(nil)
	status := EmbeddingJobStatusFailed
	retryCount := current.RetryCount
	if retryable {
		delay := in.RetryAfter
		if delay <= 0 {
			delay = embeddingRetryBackoff(current.AttemptCount)
		}
		next := now.Add(delay)
		nextAttemptAt = &next
		status = EmbeddingJobStatusPending
		retryCount++
	}
	res, err := tx.ExecContext(ctx, `
		UPDATE embedding_jobs
		SET status = ?, retry_count = ?, max_attempts = ?, claimed_by = '', claim_expires_at = NULL, last_failed_at = ?, next_attempt_at = ?,
			last_error_code = ?, last_error_message = ?, last_error_summary = ?, stale_reason = '', updated_at = ?
		WHERE subject_type = ? AND subject_id = ? AND claimed_by = ? AND status = ?
	`,
		string(status),
		retryCount,
		maxAttempts,
		ts(now),
		nullableTS(nextAttemptAt),
		strings.TrimSpace(in.ErrorCode),
		strings.TrimSpace(in.ErrorMessage),
		strings.TrimSpace(in.ErrorSummary),
		ts(now),
		subjectType,
		subjectID,
		workerID,
		string(EmbeddingJobStatusRunning),
	)
	if err != nil {
		return EmbeddingJobRecord{}, false, err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return EmbeddingJobRecord{}, false, err
	}
	if affected == 0 {
		return current, false, nil
	}
	updated, err := r.getEmbeddingJobByKey(ctx, tx, subjectType, subjectID)
	if err != nil {
		return EmbeddingJobRecord{}, false, err
	}
	if err = tx.Commit(); err != nil {
		return EmbeddingJobRecord{}, false, err
	}
	committed = true
	return updated, true, nil
}

// MarkEmbeddingJobStale marks one lifecycle row stale and clears its vector row when present.
func (r *Repository) MarkEmbeddingJobStale(ctx context.Context, in EmbeddingJobStaleInput) (EmbeddingJobRecord, bool, error) {
	subjectType := normalizeEmbeddingSubjectType(in.SubjectType)
	subjectID := strings.TrimSpace(in.SubjectID)
	if subjectType == "" || subjectID == "" {
		return EmbeddingJobRecord{}, false, domain.ErrInvalidID
	}
	now := time.Now().UTC()
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return EmbeddingJobRecord{}, false, err
	}
	committed := false
	defer rollbackEmbeddingLifecycleTx(tx, &committed)

	current, err := r.getEmbeddingJobByKey(ctx, tx, subjectType, subjectID)
	if err != nil {
		return EmbeddingJobRecord{}, false, err
	}
	if err := r.deleteEmbeddingVectorForSubject(ctx, tx, subjectType, subjectID); err != nil {
		return EmbeddingJobRecord{}, false, err
	}
	res, err := tx.ExecContext(ctx, `
		UPDATE embedding_jobs
		SET status = ?, claimed_by = '', claim_expires_at = NULL, indexed_content_hash = '', stale_reason = ?, next_attempt_at = ?, updated_at = ?
		WHERE subject_type = ? AND subject_id = ?
	`,
		string(EmbeddingJobStatusStale),
		strings.TrimSpace(in.Reason),
		ts(now),
		ts(now),
		subjectType,
		subjectID,
	)
	if err != nil {
		return EmbeddingJobRecord{}, false, err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return EmbeddingJobRecord{}, false, err
	}
	if affected == 0 {
		return current, false, nil
	}
	updated, err := r.getEmbeddingJobByKey(ctx, tx, subjectType, subjectID)
	if err != nil {
		return EmbeddingJobRecord{}, false, err
	}
	if err = tx.Commit(); err != nil {
		return EmbeddingJobRecord{}, false, err
	}
	committed = true
	return updated, true, nil
}

// GetEmbeddingJob returns one lifecycle row by subject key.
func (r *Repository) GetEmbeddingJob(ctx context.Context, subjectType, subjectID string) (EmbeddingJobRecord, error) {
	return r.getEmbeddingJobByKey(ctx, r.db, normalizeEmbeddingSubjectType(subjectType), strings.TrimSpace(subjectID))
}

// ListEmbeddingJobs lists lifecycle rows in deterministic order.
func (r *Repository) ListEmbeddingJobs(ctx context.Context, filter EmbeddingJobListFilter) ([]EmbeddingJobRecord, error) {
	rows, err := r.queryEmbeddingJobRows(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]EmbeddingJobRecord, 0)
	for rows.Next() {
		row, scanErr := scanEmbeddingJobRecord(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

// SummarizeEmbeddingJobs returns aggregate lifecycle counts for the requested scope.
func (r *Repository) SummarizeEmbeddingJobs(ctx context.Context, filter EmbeddingJobListFilter) (EmbeddingJobSummary, error) {
	rows, err := r.queryEmbeddingJobRows(ctx, filter)
	if err != nil {
		return EmbeddingJobSummary{}, err
	}
	defer rows.Close()

	summary := EmbeddingJobSummary{}
	for rows.Next() {
		var row EmbeddingJobRecord
		if row, err = scanEmbeddingJobRecord(rows); err != nil {
			return EmbeddingJobSummary{}, err
		}
		summary.Total++
		switch row.Status {
		case EmbeddingJobStatusPending:
			summary.Pending++
		case EmbeddingJobStatusRunning:
			summary.Running++
		case EmbeddingJobStatusReady:
			summary.Ready++
		case EmbeddingJobStatusFailed:
			summary.Failed++
		case EmbeddingJobStatusStale:
			summary.Stale++
		}
	}
	if err := rows.Err(); err != nil {
		return EmbeddingJobSummary{}, err
	}
	return summary, nil
}

// RecoverExpiredEmbeddingJobs returns running rows whose claims have expired and moves them back to pending.
func (r *Repository) RecoverExpiredEmbeddingJobs(ctx context.Context, before time.Time) ([]EmbeddingJobRecord, error) {
	before = before.UTC()
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return nil, err
	}
	committed := false
	defer rollbackEmbeddingLifecycleTx(tx, &committed)

	rows, err := tx.QueryContext(ctx, `
		SELECT
			subject_type, subject_id, project_id, desired_content_hash, indexed_content_hash, model_provider, model_name, model_dimensions, model_signature,
			status, attempt_count, retry_count, max_attempts, last_enqueued_at, last_started_at, last_heartbeat_at, last_succeeded_at, last_failed_at,
			next_attempt_at, claimed_by, claim_expires_at, last_error_code, last_error_message, last_error_summary, stale_reason, created_at, updated_at
		FROM embedding_jobs
		WHERE status = ? AND claim_expires_at IS NOT NULL AND claim_expires_at <= ?
		ORDER BY claim_expires_at ASC, updated_at ASC, subject_type ASC, subject_id ASC
	`, string(EmbeddingJobStatusRunning), ts(before))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	candidates := make([]EmbeddingJobRecord, 0)
	for rows.Next() {
		row, scanErr := scanEmbeddingJobRecord(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		candidates = append(candidates, row)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	recovered := make([]EmbeddingJobRecord, 0, len(candidates))
	for _, row := range candidates {
		res, err := tx.ExecContext(ctx, `
			UPDATE embedding_jobs
			SET status = ?, claimed_by = '', claim_expires_at = NULL, next_attempt_at = ?, stale_reason = ?, updated_at = ?
			WHERE subject_type = ? AND subject_id = ? AND status = ? AND claim_expires_at IS NOT NULL AND claim_expires_at <= ?
		`,
			string(EmbeddingJobStatusPending),
			ts(before),
			row.StaleReason,
			ts(before),
			row.SubjectType,
			row.SubjectID,
			string(EmbeddingJobStatusRunning),
			ts(before),
		)
		if err != nil {
			return nil, err
		}
		affected, err := res.RowsAffected()
		if err != nil {
			return nil, err
		}
		if affected == 0 {
			continue
		}
		updated, err := r.getEmbeddingJobByKey(ctx, tx, row.SubjectType, row.SubjectID)
		if err != nil {
			return nil, err
		}
		recovered = append(recovered, updated)
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	committed = true
	return recovered, nil
}

// BackfillTaskEmbeddingJobs scans tasks and seeds lifecycle rows for task subjects.
func (r *Repository) BackfillTaskEmbeddingJobs(ctx context.Context, in EmbeddingTaskBackfillInput) (int, error) {
	projectID := strings.TrimSpace(in.ProjectID)
	if projectID == "" {
		return 0, domain.ErrInvalidID
	}
	tasks, err := r.ListTasks(ctx, projectID, in.IncludeArchived)
	if err != nil {
		return 0, err
	}
	count := 0
	for _, task := range tasks {
		content := buildSQLiteTaskEmbeddingContent(task)
		hash := hashEmbeddingContent(content)
		upsertInput := EmbeddingJobUpsertInput{
			SubjectType:   "task",
			SubjectID:     task.ID,
			ProjectID:     task.ProjectID,
			DesiredHash:   hash,
			ModelProvider: strings.TrimSpace(in.ModelProvider),
			ModelName:     strings.TrimSpace(in.ModelName),
			ModelSig:      strings.TrimSpace(in.ModelSig),
			ModelDims:     in.ModelDims,
			MaxAttempts:   in.MaxAttempts,
		}
		var (
			changed   bool
			upsertErr error
		)
		if in.Force {
			_, changed, upsertErr = r.EnqueueEmbeddingJob(ctx, upsertInput)
		} else {
			_, changed, upsertErr = r.UpsertEmbeddingJob(ctx, upsertInput)
		}
		if upsertErr != nil {
			return count, upsertErr
		}
		if changed {
			count++
		}
	}
	return count, nil
}

func (r *Repository) upsertEmbeddingJob(ctx context.Context, in EmbeddingJobUpsertInput, force bool) (EmbeddingJobRecord, bool, error) {
	subjectType := normalizeEmbeddingSubjectType(in.SubjectType)
	subjectID := strings.TrimSpace(in.SubjectID)
	projectID := strings.TrimSpace(in.ProjectID)
	desiredHash := strings.TrimSpace(in.DesiredHash)
	modelProvider := strings.TrimSpace(in.ModelProvider)
	modelName := strings.TrimSpace(in.ModelName)
	modelSignature := strings.TrimSpace(in.ModelSig)
	if subjectType == "" || subjectID == "" || projectID == "" || desiredHash == "" || modelProvider == "" || modelName == "" || modelSignature == "" {
		return EmbeddingJobRecord{}, false, domain.ErrInvalidID
	}
	now := time.Now().UTC()
	maxAttempts := in.MaxAttempts
	if maxAttempts <= 0 {
		maxAttempts = defaultEmbeddingJobMaxAttempts
	}
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return EmbeddingJobRecord{}, false, err
	}
	committed := false
	defer rollbackEmbeddingLifecycleTx(tx, &committed)

	current, found, err := r.getEmbeddingJobByKeyMaybe(ctx, tx, subjectType, subjectID)
	if err != nil {
		return EmbeddingJobRecord{}, false, err
	}
	if !found {
		_, err = tx.ExecContext(ctx, `
			INSERT INTO embedding_jobs(
				subject_type, subject_id, project_id, desired_content_hash, indexed_content_hash, model_provider, model_name, model_dimensions, model_signature,
				status, attempt_count, retry_count, max_attempts, last_enqueued_at, last_started_at, last_heartbeat_at, last_succeeded_at, last_failed_at,
				next_attempt_at, claimed_by, claim_expires_at, last_error_code, last_error_message, last_error_summary, stale_reason, created_at, updated_at
			)
			VALUES (?, ?, ?, ?, '', ?, ?, ?, ?, ?, 0, 0, ?, ?, NULL, NULL, NULL, NULL, ?, '', NULL, '', '', '', '', ?, ?)
		`,
			subjectType,
			subjectID,
			projectID,
			desiredHash,
			modelProvider,
			modelName,
			in.ModelDims,
			modelSignature,
			string(EmbeddingJobStatusPending),
			maxAttempts,
			ts(now),
			ts(now),
			ts(now),
			ts(now),
		)
		if err != nil {
			return EmbeddingJobRecord{}, false, fmt.Errorf("insert embedding job: %w", err)
		}
		inserted, err := r.getEmbeddingJobByKey(ctx, tx, subjectType, subjectID)
		if err != nil {
			return EmbeddingJobRecord{}, false, err
		}
		if err = tx.Commit(); err != nil {
			return EmbeddingJobRecord{}, false, err
		}
		committed = true
		return inserted, true, nil
	}

	sameModel := current.ModelSignature == modelSignature && current.ModelProvider == modelProvider && current.ModelName == modelName && current.ModelDimensions == in.ModelDims
	sameDesired := current.DesiredContentHash == desiredHash && sameModel
	if !force && sameDesired && current.Status == EmbeddingJobStatusReady && current.IndexedContentHash == desiredHash {
		return current, false, nil
	}

	updated := current
	updated.ProjectID = projectID
	updated.DesiredContentHash = desiredHash
	updated.ModelProvider = modelProvider
	updated.ModelName = modelName
	updated.ModelDimensions = in.ModelDims
	updated.ModelSignature = modelSignature
	updated.MaxAttempts = maxAttempts
	updated.LastEnqueuedAt = now
	updated.UpdatedAt = now
	updated.ClaimedBy = ""
	updated.ClaimExpiresAt = nil
	updated.NextAttemptAt = ptrTime(now)
	updated.LastErrorCode = ""
	updated.LastErrorMessage = ""
	updated.LastErrorSummary = ""
	updated.StaleReason = ""
	if in.ResetAttempts || force {
		updated.AttemptCount = 0
		updated.RetryCount = 0
	}

	if force {
		if in.PurgeVector {
			if err := r.deleteEmbeddingVectorForSubject(ctx, tx, subjectType, subjectID); err != nil {
				return EmbeddingJobRecord{}, false, err
			}
		}
		updated.Status = EmbeddingJobStatusPending
		updated.IndexedContentHash = ""
		if err := r.updateEmbeddingJobRecord(ctx, tx, updated); err != nil {
			return EmbeddingJobRecord{}, false, err
		}
		if err = tx.Commit(); err != nil {
			return EmbeddingJobRecord{}, false, err
		}
		committed = true
		return updated, true, nil
	}

	if sameDesired {
		// The record already tracks the requested model/content combination, so a normal
		// mutation path can remain a no-op unless it was previously absent.
		return current, false, nil
	}

	if current.IndexedContentHash != "" && current.IndexedContentHash != desiredHash {
		if err := r.deleteEmbeddingVectorForSubject(ctx, tx, subjectType, subjectID); err != nil {
			return EmbeddingJobRecord{}, false, err
		}
		updated.Status = EmbeddingJobStatusStale
		updated.StaleReason = staleEmbeddingReason(current.DesiredContentHash, current.ModelSignature, desiredHash, modelSignature)
	} else {
		updated.Status = EmbeddingJobStatusPending
	}
	if err := r.updateEmbeddingJobRecord(ctx, tx, updated); err != nil {
		return EmbeddingJobRecord{}, false, err
	}
	if err = tx.Commit(); err != nil {
		return EmbeddingJobRecord{}, false, err
	}
	committed = true
	return updated, true, nil
}

func (r *Repository) updateEmbeddingJobRecord(ctx context.Context, tx *sql.Tx, row EmbeddingJobRecord) error {
	_, err := tx.ExecContext(ctx, `
		UPDATE embedding_jobs
		SET project_id = ?, desired_content_hash = ?, indexed_content_hash = ?, model_provider = ?, model_name = ?, model_dimensions = ?, model_signature = ?,
			status = ?, attempt_count = ?, retry_count = ?, max_attempts = ?, last_enqueued_at = ?, last_started_at = ?, last_heartbeat_at = ?, last_succeeded_at = ?,
			last_failed_at = ?, next_attempt_at = ?, claimed_by = ?, claim_expires_at = ?, last_error_code = ?, last_error_message = ?, last_error_summary = ?,
			stale_reason = ?, updated_at = ?
		WHERE subject_type = ? AND subject_id = ?
	`,
		row.ProjectID,
		row.DesiredContentHash,
		row.IndexedContentHash,
		row.ModelProvider,
		row.ModelName,
		row.ModelDimensions,
		row.ModelSignature,
		string(row.Status),
		row.AttemptCount,
		row.RetryCount,
		row.MaxAttempts,
		ts(row.LastEnqueuedAt),
		nullableTS(row.LastStartedAt),
		nullableTS(row.LastHeartbeatAt),
		nullableTS(row.LastSucceededAt),
		nullableTS(row.LastFailedAt),
		nullableTS(row.NextAttemptAt),
		row.ClaimedBy,
		nullableTS(row.ClaimExpiresAt),
		row.LastErrorCode,
		row.LastErrorMessage,
		row.LastErrorSummary,
		row.StaleReason,
		ts(row.UpdatedAt),
		row.SubjectType,
		row.SubjectID,
	)
	if err != nil {
		return fmt.Errorf("update embedding job: %w", err)
	}
	return nil
}

func (r *Repository) updateEmbeddingJobStatus(ctx context.Context, tx *sql.Tx, subjectType, subjectID string, status EmbeddingJobStatus, now time.Time, mutate func(*EmbeddingJobRecord)) error {
	current, err := r.getEmbeddingJobByKey(ctx, tx, subjectType, subjectID)
	if err != nil {
		return err
	}
	current.Status = status
	current.UpdatedAt = now.UTC()
	if mutate != nil {
		mutate(&current)
	}
	return r.updateEmbeddingJobRecord(ctx, tx, current)
}

func (r *Repository) getEmbeddingJobByKey(ctx context.Context, q queryer, subjectType, subjectID string) (EmbeddingJobRecord, error) {
	row := q.QueryRowContext(ctx, `
		SELECT
			subject_type, subject_id, project_id, desired_content_hash, indexed_content_hash, model_provider, model_name, model_dimensions, model_signature,
			status, attempt_count, retry_count, max_attempts, last_enqueued_at, last_started_at, last_heartbeat_at, last_succeeded_at, last_failed_at,
			next_attempt_at, claimed_by, claim_expires_at, last_error_code, last_error_message, last_error_summary, stale_reason, created_at, updated_at
		FROM embedding_jobs
		WHERE subject_type = ? AND subject_id = ?
	`, subjectType, subjectID)
	return scanEmbeddingJobRecord(row)
}

func (r *Repository) getEmbeddingJobByKeyMaybe(ctx context.Context, q queryer, subjectType, subjectID string) (EmbeddingJobRecord, bool, error) {
	row := q.QueryRowContext(ctx, `
		SELECT
			subject_type, subject_id, project_id, desired_content_hash, indexed_content_hash, model_provider, model_name, model_dimensions, model_signature,
			status, attempt_count, retry_count, max_attempts, last_enqueued_at, last_started_at, last_heartbeat_at, last_succeeded_at, last_failed_at,
			next_attempt_at, claimed_by, claim_expires_at, last_error_code, last_error_message, last_error_summary, stale_reason, created_at, updated_at
		FROM embedding_jobs
		WHERE subject_type = ? AND subject_id = ?
	`, subjectType, subjectID)
	record, err := scanEmbeddingJobRecord(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return EmbeddingJobRecord{}, false, nil
		}
		return EmbeddingJobRecord{}, false, err
	}
	return record, true, nil
}

func (r *Repository) queryEmbeddingJobRows(ctx context.Context, filter EmbeddingJobListFilter) (*sql.Rows, error) {
	query := `
		SELECT
			subject_type, subject_id, project_id, desired_content_hash, indexed_content_hash, model_provider, model_name, model_dimensions, model_signature,
			status, attempt_count, retry_count, max_attempts, last_enqueued_at, last_started_at, last_heartbeat_at, last_succeeded_at, last_failed_at,
			next_attempt_at, claimed_by, claim_expires_at, last_error_code, last_error_message, last_error_summary, stale_reason, created_at, updated_at
		FROM embedding_jobs
		WHERE 1 = 1
	`
	args := make([]any, 0, 4)
	if projectID := strings.TrimSpace(filter.ProjectID); projectID != "" {
		query += ` AND project_id = ?`
		args = append(args, projectID)
	}
	if subjectType := normalizeEmbeddingSubjectType(filter.SubjectType); subjectType != "" {
		query += ` AND subject_type = ?`
		args = append(args, subjectType)
	}
	if len(filter.Statuses) > 0 {
		query += ` AND status IN (` + queryPlaceholders(len(filter.Statuses)) + `)`
		for _, status := range filter.Statuses {
			args = append(args, string(normalizeEmbeddingJobStatus(status)))
		}
	}
	query += ` ORDER BY updated_at DESC, subject_type ASC, subject_id ASC`
	if filter.Limit > 0 {
		query += ` LIMIT ?`
		args = append(args, filter.Limit)
	}
	if filter.Offset > 0 {
		if filter.Limit <= 0 {
			query += ` LIMIT -1`
		}
		query += ` OFFSET ?`
		args = append(args, filter.Offset)
	}
	return r.db.QueryContext(ctx, query, args...)
}

func (r *Repository) selectNextEligibleEmbeddingJob(ctx context.Context, tx *sql.Tx, filter EmbeddingJobListFilter, now time.Time) (EmbeddingJobRecord, bool, error) {
	query := `
		SELECT
			subject_type, subject_id, project_id, desired_content_hash, indexed_content_hash, model_provider, model_name, model_dimensions, model_signature,
			status, attempt_count, retry_count, max_attempts, last_enqueued_at, last_started_at, last_heartbeat_at, last_succeeded_at, last_failed_at,
			next_attempt_at, claimed_by, claim_expires_at, last_error_code, last_error_message, last_error_summary, stale_reason, created_at, updated_at
		FROM embedding_jobs
		WHERE 1 = 1
	`
	args := make([]any, 0, 5)
	if projectID := strings.TrimSpace(filter.ProjectID); projectID != "" {
		query += ` AND project_id = ?`
		args = append(args, projectID)
	}
	if subjectType := normalizeEmbeddingSubjectType(filter.SubjectType); subjectType != "" {
		query += ` AND subject_type = ?`
		args = append(args, subjectType)
	}
	statuses := filter.Statuses
	if len(statuses) == 0 {
		statuses = []EmbeddingJobStatus{EmbeddingJobStatusPending, EmbeddingJobStatusStale}
	}
	query += ` AND status IN (` + queryPlaceholders(len(statuses)) + `)`
	for _, status := range statuses {
		args = append(args, string(normalizeEmbeddingJobStatus(status)))
	}
	query += ` AND COALESCE(next_attempt_at, last_enqueued_at) <= ?`
	args = append(args, ts(now))
	query += ` ORDER BY COALESCE(next_attempt_at, last_enqueued_at) ASC, updated_at ASC, subject_type ASC, subject_id ASC LIMIT 1`
	row := tx.QueryRowContext(ctx, query, args...)
	record, err := scanEmbeddingJobRecord(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return EmbeddingJobRecord{}, false, nil
		}
		return EmbeddingJobRecord{}, false, err
	}
	return record, true, nil
}

func (r *Repository) deleteEmbeddingVectorForSubject(ctx context.Context, q execer, subjectType, subjectID string) error {
	return deleteEmbeddingDocument(ctx, q, normalizeEmbeddingSubjectType(subjectType), subjectID)
}

// buildSQLiteTaskEmbeddingContent produces canonical searchable text for one task.
func buildSQLiteTaskEmbeddingContent(task domain.Task) string {
	parts := make([]string, 0, 10)
	appendIfPresent := func(value string) {
		value = strings.TrimSpace(value)
		if value == "" {
			return
		}
		parts = append(parts, value)
	}
	appendIfPresent(task.Title)
	appendIfPresent(task.Description)
	if len(task.Labels) > 0 {
		appendIfPresent(strings.Join(task.Labels, ", "))
	}
	appendIfPresent(task.Metadata.Objective)
	appendIfPresent(task.Metadata.AcceptanceCriteria)
	appendIfPresent(task.Metadata.ValidationPlan)
	appendIfPresent(task.Metadata.BlockedReason)
	appendIfPresent(task.Metadata.RiskNotes)
	return strings.Join(parts, "\n")
}

func hashEmbeddingContent(content string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(content)))
	return hex.EncodeToString(sum[:])
}

func embeddingRetryBackoff(attemptCount int) time.Duration {
	if attemptCount < 1 {
		attemptCount = 1
	}
	delay := 10 * time.Second
	for i := 1; i < attemptCount; i++ {
		if delay >= 10*time.Minute {
			return 10 * time.Minute
		}
		delay *= 2
	}
	if delay > 10*time.Minute {
		return 10 * time.Minute
	}
	return delay
}

func staleEmbeddingReason(desiredHash, desiredSignature, processedHash, processedSignature string) string {
	desiredHash = strings.TrimSpace(desiredHash)
	desiredSignature = strings.TrimSpace(desiredSignature)
	processedHash = strings.TrimSpace(processedHash)
	processedSignature = strings.TrimSpace(processedSignature)
	if desiredHash != processedHash && desiredSignature != processedSignature {
		return "content hash and model signature changed during indexing"
	}
	if desiredHash != processedHash {
		return "content hash changed during indexing"
	}
	if desiredSignature != processedSignature {
		return "model signature changed during indexing"
	}
	return "embedding content became stale"
}

func normalizeEmbeddingSubjectType(subjectType string) string {
	switch strings.TrimSpace(strings.ToLower(subjectType)) {
	case "task":
		return "work_item"
	default:
		return strings.TrimSpace(strings.ToLower(subjectType))
	}
}

func normalizeEmbeddingJobStatus(status EmbeddingJobStatus) EmbeddingJobStatus {
	switch strings.TrimSpace(strings.ToLower(string(status))) {
	case string(EmbeddingJobStatusPending):
		return EmbeddingJobStatusPending
	case string(EmbeddingJobStatusRunning):
		return EmbeddingJobStatusRunning
	case string(EmbeddingJobStatusReady):
		return EmbeddingJobStatusReady
	case string(EmbeddingJobStatusFailed):
		return EmbeddingJobStatusFailed
	case string(EmbeddingJobStatusStale):
		return EmbeddingJobStatusStale
	default:
		return EmbeddingJobStatusPending
	}
}

// queryer describes query-capable storage handles used by this package.
type queryer interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

// execer describes exec-capable storage handles used by this package.
type execer interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}

func scanEmbeddingJobRecord(s scanner) (EmbeddingJobRecord, error) {
	var (
		record           EmbeddingJobRecord
		lastEnqueuedRaw  string
		lastStartedRaw   sql.NullString
		lastHeartbeatRaw sql.NullString
		lastSucceededRaw sql.NullString
		lastFailedRaw    sql.NullString
		nextAttemptRaw   sql.NullString
		claimExpiresRaw  sql.NullString
		createdRaw       string
		updatedRaw       string
	)
	if err := s.Scan(
		&record.SubjectType,
		&record.SubjectID,
		&record.ProjectID,
		&record.DesiredContentHash,
		&record.IndexedContentHash,
		&record.ModelProvider,
		&record.ModelName,
		&record.ModelDimensions,
		&record.ModelSignature,
		&record.Status,
		&record.AttemptCount,
		&record.RetryCount,
		&record.MaxAttempts,
		&lastEnqueuedRaw,
		&lastStartedRaw,
		&lastHeartbeatRaw,
		&lastSucceededRaw,
		&lastFailedRaw,
		&nextAttemptRaw,
		&record.ClaimedBy,
		&claimExpiresRaw,
		&record.LastErrorCode,
		&record.LastErrorMessage,
		&record.LastErrorSummary,
		&record.StaleReason,
		&createdRaw,
		&updatedRaw,
	); err != nil {
		return EmbeddingJobRecord{}, err
	}
	record.Status = normalizeEmbeddingJobStatus(record.Status)
	record.LastEnqueuedAt = parseTS(lastEnqueuedRaw)
	record.LastStartedAt = parseNullTS(lastStartedRaw)
	record.LastHeartbeatAt = parseNullTS(lastHeartbeatRaw)
	record.LastSucceededAt = parseNullTS(lastSucceededRaw)
	record.LastFailedAt = parseNullTS(lastFailedRaw)
	record.NextAttemptAt = parseNullTS(nextAttemptRaw)
	record.ClaimExpiresAt = parseNullTS(claimExpiresRaw)
	record.CreatedAt = parseTS(createdRaw)
	record.UpdatedAt = parseTS(updatedRaw)
	return record, nil
}

func ptrTime(t time.Time) *time.Time {
	tt := t.UTC()
	return &tt
}
