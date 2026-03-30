package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/hylla/tillsyn/internal/app"
	"github.com/hylla/tillsyn/internal/domain"
)

// Ensure the repository satisfies the app-layer lifecycle contract.
var _ app.EmbeddingLifecycleStore = (*Repository)(nil)

// EnqueueEmbedding persists one lifecycle enqueue request for a work-item subject.
func (r *Repository) EnqueueEmbedding(ctx context.Context, in app.EmbeddingEnqueueInput) (app.EmbeddingRecord, error) {
	subjectType := storageEmbeddingSubjectType(in.SubjectType)
	if subjectType == "" || strings.TrimSpace(in.SubjectID) == "" || strings.TrimSpace(in.ProjectID) == "" {
		return app.EmbeddingRecord{}, domain.ErrInvalidID
	}

	upsertInput := EmbeddingJobUpsertInput{
		SubjectType:   subjectType,
		SubjectID:     strings.TrimSpace(in.SubjectID),
		ProjectID:     strings.TrimSpace(in.ProjectID),
		DesiredHash:   strings.TrimSpace(in.ContentHash),
		ModelProvider: strings.TrimSpace(in.ModelProvider),
		ModelName:     strings.TrimSpace(in.ModelName),
		ModelSig:      strings.TrimSpace(in.ModelSignature),
		ModelDims:     in.ModelDimensions,
		MaxAttempts:   in.MaxAttempts,
	}

	var (
		record EmbeddingJobRecord
		err    error
	)
	if in.Force {
		record, _, err = r.EnqueueEmbeddingJob(ctx, upsertInput)
	} else {
		record, _, err = r.UpsertEmbeddingJob(ctx, upsertInput)
	}
	if err != nil {
		return app.EmbeddingRecord{}, err
	}
	return mapEmbeddingJobRecord(record), nil
}

// DeleteEmbeddingSubject removes one lifecycle row and its vector state when present.
func (r *Repository) DeleteEmbeddingSubject(ctx context.Context, subjectType app.EmbeddingSubjectType, subjectID string) error {
	storageType := storageEmbeddingSubjectType(subjectType)
	subjectID = strings.TrimSpace(subjectID)
	if storageType == "" || subjectID == "" {
		return domain.ErrInvalidID
	}

	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return err
	}
	committed := false
	defer rollbackEmbeddingLifecycleTx(tx, &committed)

	if err := r.deleteEmbeddingVectorForSubject(ctx, tx, storageType, subjectID); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `DELETE FROM embedding_jobs WHERE subject_type = ? AND subject_id = ?`, storageType, subjectID); err != nil {
		return fmt.Errorf("delete embedding lifecycle row: %w", err)
	}
	if err = tx.Commit(); err != nil {
		return err
	}
	committed = true
	return nil
}

// ListEmbeddings lists durable lifecycle rows for the requested scope.
func (r *Repository) ListEmbeddings(ctx context.Context, filter app.EmbeddingListFilter) ([]app.EmbeddingRecord, error) {
	if filter.ProjectIDs != nil && len(normalizedStringSet(filter.ProjectIDs)) == 0 {
		return []app.EmbeddingRecord{}, nil
	}
	rows, err := r.queryAppEmbeddingRows(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]app.EmbeddingRecord, 0)
	for rows.Next() {
		record, scanErr := scanEmbeddingJobRecord(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		out = append(out, mapEmbeddingJobRecord(record))
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

// SummarizeEmbeddings returns aggregate lifecycle counts for the requested scope.
func (r *Repository) SummarizeEmbeddings(ctx context.Context, filter app.EmbeddingListFilter) (app.EmbeddingSummary, error) {
	rows, err := r.ListEmbeddings(ctx, filter)
	if err != nil {
		return app.EmbeddingSummary{}, err
	}

	summary := app.EmbeddingSummary{
		SubjectType: filter.SubjectType,
		ProjectIDs:  append([]string(nil), filter.ProjectIDs...),
	}
	for _, row := range rows {
		switch row.Status {
		case app.EmbeddingLifecyclePending:
			summary.PendingCount++
		case app.EmbeddingLifecycleRunning:
			summary.RunningCount++
		case app.EmbeddingLifecycleReady:
			summary.ReadyCount++
		case app.EmbeddingLifecycleFailed:
			summary.FailedCount++
		case app.EmbeddingLifecycleStale:
			summary.StaleCount++
		}
	}
	return summary, nil
}

// ClaimEmbeddings claims up to the requested number of ready-to-run lifecycle rows.
func (r *Repository) ClaimEmbeddings(ctx context.Context, in app.EmbeddingClaimInput) ([]app.EmbeddingRecord, error) {
	storageType := storageEmbeddingSubjectType(in.SubjectType)
	workerID := strings.TrimSpace(in.WorkerID)
	if storageType == "" || workerID == "" {
		return nil, domain.ErrInvalidID
	}

	now := in.Now.UTC()
	if now.IsZero() {
		now = time.Now().UTC()
	}
	limit := in.Limit
	if limit <= 0 {
		limit = 1
	}

	out := make([]app.EmbeddingRecord, 0, limit)
	for len(out) < limit {
		row, found, err := r.ClaimNextEmbeddingJob(ctx, EmbeddingJobClaimNextInput{
			SubjectType: storageType,
			WorkerID:    workerID,
			ClaimTTL:    in.ClaimTTL,
		}, now)
		if err != nil {
			return nil, err
		}
		if !found {
			break
		}
		out = append(out, mapEmbeddingJobRecord(row))
	}
	return out, nil
}

// HeartbeatEmbedding extends one active claim.
func (r *Repository) HeartbeatEmbedding(ctx context.Context, in app.EmbeddingHeartbeatInput) error {
	storageType := storageEmbeddingSubjectType(in.SubjectType)
	if storageType == "" || strings.TrimSpace(in.SubjectID) == "" || strings.TrimSpace(in.WorkerID) == "" {
		return domain.ErrInvalidID
	}

	now := in.Now.UTC()
	if now.IsZero() {
		now = time.Now().UTC()
	}
	_, changed, err := r.HeartbeatEmbeddingJob(ctx, EmbeddingJobHeartbeatInput{
		SubjectType: storageType,
		SubjectID:   strings.TrimSpace(in.SubjectID),
		WorkerID:    strings.TrimSpace(in.WorkerID),
		ClaimTTL:    in.ClaimTTL,
	}, now)
	if err != nil {
		return err
	}
	if !changed {
		return fmt.Errorf("heartbeat embedding claim: %w", app.ErrEmbeddingClaimLost)
	}
	return nil
}

// MarkEmbeddingSuccess marks one claimed lifecycle row complete.
func (r *Repository) MarkEmbeddingSuccess(ctx context.Context, in app.EmbeddingSuccessInput) (app.EmbeddingRecord, error) {
	storageType := storageEmbeddingSubjectType(in.SubjectType)
	if storageType == "" || strings.TrimSpace(in.SubjectID) == "" || strings.TrimSpace(in.WorkerID) == "" {
		return app.EmbeddingRecord{}, domain.ErrInvalidID
	}

	record, changed, err := r.CompleteEmbeddingJob(ctx, EmbeddingJobCompleteInput{
		SubjectType:          storageType,
		SubjectID:            strings.TrimSpace(in.SubjectID),
		WorkerID:             strings.TrimSpace(in.WorkerID),
		ProcessedContentHash: strings.TrimSpace(in.ContentHash),
		ProcessedModelSig:    strings.TrimSpace(in.ModelSignature),
	})
	if err != nil {
		return app.EmbeddingRecord{}, err
	}
	if !changed {
		return app.EmbeddingRecord{}, fmt.Errorf("mark embedding success: %w", app.ErrEmbeddingClaimLost)
	}
	return mapEmbeddingJobRecord(record), nil
}

// MarkEmbeddingFailure records one failed claim and schedules retry when applicable.
func (r *Repository) MarkEmbeddingFailure(ctx context.Context, in app.EmbeddingFailureInput) (app.EmbeddingRecord, error) {
	storageType := storageEmbeddingSubjectType(in.SubjectType)
	if storageType == "" || strings.TrimSpace(in.SubjectID) == "" || strings.TrimSpace(in.WorkerID) == "" {
		return app.EmbeddingRecord{}, domain.ErrInvalidID
	}

	retryAfter := time.Duration(0)
	if in.Retryable {
		current, err := r.GetEmbeddingJob(ctx, storageType, strings.TrimSpace(in.SubjectID))
		if err != nil {
			return app.EmbeddingRecord{}, err
		}
		retryAfter = embeddingRetryBackoffWithBounds(current.AttemptCount, in.InitialRetryBackoff, in.MaxRetryBackoff)
	}

	record, changed, err := r.FailEmbeddingJob(ctx, EmbeddingJobFailInput{
		SubjectType:  storageType,
		SubjectID:    strings.TrimSpace(in.SubjectID),
		WorkerID:     strings.TrimSpace(in.WorkerID),
		ErrorCode:    strings.TrimSpace(in.ErrorCode),
		ErrorMessage: strings.TrimSpace(in.ErrorMessage),
		ErrorSummary: strings.TrimSpace(in.ErrorSummary),
		Retryable:    in.Retryable,
		RetryAfter:   retryAfter,
	})
	if err != nil {
		return app.EmbeddingRecord{}, err
	}
	if !changed {
		return app.EmbeddingRecord{}, fmt.Errorf("mark embedding failure: %w", app.ErrEmbeddingClaimLost)
	}
	return mapEmbeddingJobRecord(record), nil
}

// RecoverExpiredEmbeddingClaims moves expired running claims back to pending.
func (r *Repository) RecoverExpiredEmbeddingClaims(ctx context.Context, before time.Time) ([]app.EmbeddingRecord, error) {
	rows, err := r.RecoverExpiredEmbeddingJobs(ctx, before)
	if err != nil {
		return nil, err
	}
	out := make([]app.EmbeddingRecord, 0, len(rows))
	for _, row := range rows {
		out = append(out, mapEmbeddingJobRecord(row))
	}
	return out, nil
}

// MarkEmbeddingsStaleByModel invalidates rows whose stored model signature no longer matches runtime.
func (r *Repository) MarkEmbeddingsStaleByModel(ctx context.Context, in app.EmbeddingStaleByModelInput) ([]app.EmbeddingRecord, error) {
	storageType := storageEmbeddingSubjectType(in.SubjectType)
	modelSignature := strings.TrimSpace(in.ModelSignature)
	if storageType == "" || modelSignature == "" {
		return nil, domain.ErrInvalidID
	}

	now := in.StaledAt.UTC()
	if now.IsZero() {
		now = time.Now().UTC()
	}
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
		WHERE subject_type = ? AND model_signature <> ?
	`, storageType, modelSignature)
	if err != nil {
		return nil, err
	}
	candidates := make([]EmbeddingJobRecord, 0)
	for rows.Next() {
		record, scanErr := scanEmbeddingJobRecord(rows)
		if scanErr != nil {
			_ = rows.Close()
			return nil, scanErr
		}
		candidates = append(candidates, record)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(candidates) == 0 {
		return nil, nil
	}

	for _, row := range candidates {
		if err := r.deleteEmbeddingVectorForSubject(ctx, tx, storageType, row.SubjectID); err != nil {
			return nil, err
		}
	}
	res, err := tx.ExecContext(ctx, `
		UPDATE embedding_jobs
		SET status = ?, indexed_content_hash = '', model_provider = ?, model_name = ?, model_dimensions = ?, model_signature = ?, claimed_by = '', claim_expires_at = NULL,
			last_error_code = '', last_error_message = '', last_error_summary = '', stale_reason = ?, next_attempt_at = ?, updated_at = ?
		WHERE subject_type = ? AND model_signature <> ?
	`,
		string(EmbeddingJobStatusStale),
		strings.TrimSpace(in.ModelProvider),
		strings.TrimSpace(in.ModelName),
		in.ModelDimensions,
		modelSignature,
		firstNonEmptyTrimmed(in.Reason, "model_signature_changed"),
		ts(now),
		ts(now),
		storageType,
		modelSignature,
	)
	if err != nil {
		return nil, err
	}
	if _, err := res.RowsAffected(); err != nil {
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	committed = true
	out := make([]app.EmbeddingRecord, 0, len(candidates))
	for _, row := range candidates {
		row.IndexedContentHash = ""
		row.ModelProvider = strings.TrimSpace(in.ModelProvider)
		row.ModelName = strings.TrimSpace(in.ModelName)
		row.ModelDimensions = in.ModelDimensions
		row.ModelSignature = modelSignature
		row.Status = EmbeddingJobStatusStale
		row.ClaimedBy = ""
		row.ClaimExpiresAt = nil
		row.LastErrorCode = ""
		row.LastErrorMessage = ""
		row.LastErrorSummary = ""
		row.StaleReason = firstNonEmptyTrimmed(in.Reason, "model_signature_changed")
		row.NextAttemptAt = ptrTime(now)
		row.UpdatedAt = now
		out = append(out, mapEmbeddingJobRecord(row))
	}
	return out, nil
}

// queryAppEmbeddingRows applies the app-facing lifecycle filters without exposing storage-only subject naming.
func (r *Repository) queryAppEmbeddingRows(ctx context.Context, filter app.EmbeddingListFilter) (*sql.Rows, error) {
	query := `
		SELECT
			subject_type, subject_id, project_id, desired_content_hash, indexed_content_hash, model_provider, model_name, model_dimensions, model_signature,
			status, attempt_count, retry_count, max_attempts, last_enqueued_at, last_started_at, last_heartbeat_at, last_succeeded_at, last_failed_at,
			next_attempt_at, claimed_by, claim_expires_at, last_error_code, last_error_message, last_error_summary, stale_reason, created_at, updated_at
		FROM embedding_jobs
		WHERE 1 = 1
	`
	args := make([]any, 0, 8)

	if projectIDs := normalizedStringSet(filter.ProjectIDs); len(projectIDs) > 0 {
		query += ` AND project_id IN (` + queryPlaceholders(len(projectIDs)) + `)`
		for _, projectID := range projectIDs {
			args = append(args, projectID)
		}
	}
	if subjectType := storageEmbeddingSubjectType(filter.SubjectType); subjectType != "" {
		query += ` AND subject_type = ?`
		args = append(args, subjectType)
	}
	if subjectIDs := normalizedStringSet(filter.SubjectIDs); len(subjectIDs) > 0 {
		query += ` AND subject_id IN (` + queryPlaceholders(len(subjectIDs)) + `)`
		for _, subjectID := range subjectIDs {
			args = append(args, subjectID)
		}
	}
	if statuses := storageEmbeddingStatuses(filter.Statuses); len(statuses) > 0 {
		query += ` AND status IN (` + queryPlaceholders(len(statuses)) + `)`
		for _, status := range statuses {
			args = append(args, string(status))
		}
	}
	query += ` ORDER BY updated_at DESC, subject_type ASC, subject_id ASC`
	if filter.Limit > 0 {
		query += ` LIMIT ?`
		args = append(args, filter.Limit)
	}
	return r.db.QueryContext(ctx, query, args...)
}

// mapEmbeddingJobRecord converts one storage lifecycle row into the app-facing record.
func mapEmbeddingJobRecord(row EmbeddingJobRecord) app.EmbeddingRecord {
	lastEnqueuedAt := row.LastEnqueuedAt
	return app.EmbeddingRecord{
		SubjectType:        appEmbeddingSubjectType(row.SubjectType),
		SubjectID:          row.SubjectID,
		ProjectID:          row.ProjectID,
		ContentHashDesired: row.DesiredContentHash,
		ContentHashIndexed: row.IndexedContentHash,
		ModelProvider:      row.ModelProvider,
		ModelName:          row.ModelName,
		ModelDimensions:    row.ModelDimensions,
		ModelSignature:     row.ModelSignature,
		Status:             app.EmbeddingLifecycleStatus(row.Status),
		AttemptCount:       row.AttemptCount,
		RetryCount:         row.RetryCount,
		MaxAttempts:        row.MaxAttempts,
		NextAttemptAt:      cloneTimePtr(row.NextAttemptAt),
		LastEnqueuedAt:     &lastEnqueuedAt,
		LastStartedAt:      cloneTimePtr(row.LastStartedAt),
		LastHeartbeatAt:    cloneTimePtr(row.LastHeartbeatAt),
		LastSucceededAt:    cloneTimePtr(row.LastSucceededAt),
		LastFailedAt:       cloneTimePtr(row.LastFailedAt),
		LastErrorCode:      row.LastErrorCode,
		LastErrorMessage:   row.LastErrorMessage,
		LastErrorSummary:   row.LastErrorSummary,
		StaleReason:        row.StaleReason,
		ClaimedBy:          row.ClaimedBy,
		ClaimExpiresAt:     cloneTimePtr(row.ClaimExpiresAt),
		CreatedAt:          row.CreatedAt,
		UpdatedAt:          row.UpdatedAt,
	}
}

// storageEmbeddingSubjectType maps one app-facing subject family onto the sqlite lifecycle subject family.
func storageEmbeddingSubjectType(subjectType app.EmbeddingSubjectType) string {
	switch strings.TrimSpace(strings.ToLower(string(subjectType))) {
	case "work_item":
		return "work_item"
	case "thread_context":
		return "thread_context"
	case "project_document":
		return "project_document"
	default:
		return ""
	}
}

// appEmbeddingSubjectType maps one sqlite lifecycle subject family onto the app-facing type.
func appEmbeddingSubjectType(subjectType string) app.EmbeddingSubjectType {
	switch strings.TrimSpace(strings.ToLower(subjectType)) {
	case "task", "work_item":
		return app.EmbeddingSubjectTypeWorkItem
	case "thread_context":
		return app.EmbeddingSubjectTypeThreadContext
	case "project_document":
		return app.EmbeddingSubjectTypeProjectDocument
	default:
		return ""
	}
}

// storageEmbeddingStatuses maps app-facing lifecycle statuses onto sqlite lifecycle statuses.
func storageEmbeddingStatuses(statuses []app.EmbeddingLifecycleStatus) []EmbeddingJobStatus {
	out := make([]EmbeddingJobStatus, 0, len(statuses))
	for _, status := range statuses {
		switch strings.TrimSpace(strings.ToLower(string(status))) {
		case string(app.EmbeddingLifecyclePending):
			out = append(out, EmbeddingJobStatusPending)
		case string(app.EmbeddingLifecycleRunning):
			out = append(out, EmbeddingJobStatusRunning)
		case string(app.EmbeddingLifecycleReady):
			out = append(out, EmbeddingJobStatusReady)
		case string(app.EmbeddingLifecycleFailed):
			out = append(out, EmbeddingJobStatusFailed)
		case string(app.EmbeddingLifecycleStale):
			out = append(out, EmbeddingJobStatusStale)
		}
	}
	return out
}

// normalizedStringSet trims, deduplicates, and sorts one string filter list.
func normalizedStringSet(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || slices.Contains(out, value) {
			continue
		}
		out = append(out, value)
	}
	slices.Sort(out)
	return out
}

// cloneTimePtr copies one optional timestamp so app callers cannot mutate storage-owned pointers.
func cloneTimePtr(ts *time.Time) *time.Time {
	if ts == nil {
		return nil
	}
	value := ts.UTC()
	return &value
}

// embeddingRetryBackoffWithBounds applies the runtime-configured retry envelope to one failure attempt count.
func embeddingRetryBackoffWithBounds(attemptCount int, initial, maximum time.Duration) time.Duration {
	if initial <= 0 {
		initial = 15 * time.Second
	}
	if maximum <= 0 || maximum < initial {
		maximum = initial
	}
	if attemptCount < 1 {
		attemptCount = 1
	}
	delay := initial
	for i := 1; i < attemptCount; i++ {
		if delay >= maximum {
			return maximum
		}
		delay *= 2
	}
	if delay > maximum {
		return maximum
	}
	return delay
}

// firstNonEmptyTrimmed returns the first non-empty trimmed string in order.
func firstNonEmptyTrimmed(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}
