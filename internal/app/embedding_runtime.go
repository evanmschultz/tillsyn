package app

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/log"
	"github.com/evanmschultz/tillsyn/internal/domain"
)

// EmbeddingSubjectType identifies one indexed subject family.
type EmbeddingSubjectType string

// Supported embedding subject families.
const (
	EmbeddingSubjectTypeWorkItem        EmbeddingSubjectType = "work_item"
	EmbeddingSubjectTypeThreadContext   EmbeddingSubjectType = "thread_context"
	EmbeddingSubjectTypeProjectDocument EmbeddingSubjectType = "project_document"
)

// EmbeddingLifecycleStatus identifies one persistent indexing state.
type EmbeddingLifecycleStatus string

// Supported embedding lifecycle states.
const (
	EmbeddingLifecyclePending EmbeddingLifecycleStatus = "pending"
	EmbeddingLifecycleRunning EmbeddingLifecycleStatus = "running"
	EmbeddingLifecycleReady   EmbeddingLifecycleStatus = "ready"
	EmbeddingLifecycleFailed  EmbeddingLifecycleStatus = "failed"
	EmbeddingLifecycleStale   EmbeddingLifecycleStatus = "stale"
)

const (
	defaultEmbeddingClaimBatchSize      = 16
	defaultEmbeddingMaxAttempts         = 5
	defaultEmbeddingPollInterval        = 2 * time.Second
	defaultEmbeddingClaimTTL            = 2 * time.Minute
	defaultEmbeddingInitialRetryBackoff = 15 * time.Second
	defaultEmbeddingMaxRetryBackoff     = 15 * time.Minute
)

// EmbeddingRecord stores one durable lifecycle row.
type EmbeddingRecord struct {
	SubjectType        EmbeddingSubjectType
	SubjectID          string
	ProjectID          string
	ContentHashDesired string
	ContentHashIndexed string
	ModelProvider      string
	ModelName          string
	ModelDimensions    int64
	ModelSignature     string
	Status             EmbeddingLifecycleStatus
	AttemptCount       int
	RetryCount         int
	MaxAttempts        int
	NextAttemptAt      *time.Time
	LastEnqueuedAt     *time.Time
	LastStartedAt      *time.Time
	LastHeartbeatAt    *time.Time
	LastSucceededAt    *time.Time
	LastFailedAt       *time.Time
	LastErrorCode      string
	LastErrorMessage   string
	LastErrorSummary   string
	StaleReason        string
	ClaimedBy          string
	ClaimExpiresAt     *time.Time
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

// EmbeddingListFilter scopes lifecycle inventory queries.
type EmbeddingListFilter struct {
	ProjectIDs  []string
	SubjectType EmbeddingSubjectType
	SubjectIDs  []string
	Statuses    []EmbeddingLifecycleStatus
	Limit       int
}

// EmbeddingSummary stores aggregated lifecycle counts.
type EmbeddingSummary struct {
	SubjectType  EmbeddingSubjectType
	ProjectIDs   []string
	PendingCount int
	RunningCount int
	ReadyCount   int
	FailedCount  int
	StaleCount   int
}

// EmbeddingEnqueueInput stores lifecycle enqueue input.
type EmbeddingEnqueueInput struct {
	SubjectType     EmbeddingSubjectType
	SubjectID       string
	ProjectID       string
	ContentHash     string
	ModelProvider   string
	ModelName       string
	ModelDimensions int64
	ModelSignature  string
	MaxAttempts     int
	Force           bool
	Reason          string
	EnqueuedAt      time.Time
}

// EmbeddingClaimInput stores worker-claim options.
type EmbeddingClaimInput struct {
	SubjectType EmbeddingSubjectType
	WorkerID    string
	Now         time.Time
	Limit       int
	ClaimTTL    time.Duration
}

// EmbeddingHeartbeatInput stores worker heartbeat options.
type EmbeddingHeartbeatInput struct {
	SubjectType EmbeddingSubjectType
	SubjectID   string
	WorkerID    string
	Now         time.Time
	ClaimTTL    time.Duration
}

// EmbeddingSuccessInput stores success-transition data.
type EmbeddingSuccessInput struct {
	SubjectType     EmbeddingSubjectType
	SubjectID       string
	ProjectID       string
	ContentHash     string
	ModelProvider   string
	ModelName       string
	ModelDimensions int64
	ModelSignature  string
	WorkerID        string
	CompletedAt     time.Time
}

// EmbeddingFailureInput stores failure-transition data.
type EmbeddingFailureInput struct {
	SubjectType         EmbeddingSubjectType
	SubjectID           string
	ProjectID           string
	ModelSignature      string
	WorkerID            string
	Retryable           bool
	ErrorCode           string
	ErrorMessage        string
	ErrorSummary        string
	FailedAt            time.Time
	InitialRetryBackoff time.Duration
	MaxRetryBackoff     time.Duration
}

// EmbeddingStaleByModelInput stores stale-by-model sweep options.
type EmbeddingStaleByModelInput struct {
	ModelProvider   string
	ModelName       string
	ModelDimensions int64
	SubjectType     EmbeddingSubjectType
	ModelSignature  string
	Reason          string
	StaledAt        time.Time
}

// EmbeddingLifecycleStore describes durable lifecycle storage for indexed subjects.
type EmbeddingLifecycleStore interface {
	EnqueueEmbedding(context.Context, EmbeddingEnqueueInput) (EmbeddingRecord, error)
	DeleteEmbeddingSubject(context.Context, EmbeddingSubjectType, string) error
	ListEmbeddings(context.Context, EmbeddingListFilter) ([]EmbeddingRecord, error)
	SummarizeEmbeddings(context.Context, EmbeddingListFilter) (EmbeddingSummary, error)
	ClaimEmbeddings(context.Context, EmbeddingClaimInput) ([]EmbeddingRecord, error)
	HeartbeatEmbedding(context.Context, EmbeddingHeartbeatInput) error
	MarkEmbeddingSuccess(context.Context, EmbeddingSuccessInput) (EmbeddingRecord, error)
	MarkEmbeddingFailure(context.Context, EmbeddingFailureInput) (EmbeddingRecord, error)
	RecoverExpiredEmbeddingClaims(context.Context, time.Time) ([]EmbeddingRecord, error)
	MarkEmbeddingsStaleByModel(context.Context, EmbeddingStaleByModelInput) ([]EmbeddingRecord, error)
}

// EmbeddingRuntimeConfig stores background lifecycle settings.
type EmbeddingRuntimeConfig struct {
	Enabled             bool
	Provider            string
	Model               string
	BaseURL             string
	Dimensions          int64
	ModelSignature      string
	MaxAttempts         int
	PollInterval        time.Duration
	ClaimTTL            time.Duration
	InitialRetryBackoff time.Duration
	MaxRetryBackoff     time.Duration
	ClaimBatchSize      int
	WorkerID            string
}

// DefaultEmbeddingRuntimeConfig returns stable runtime defaults.
func DefaultEmbeddingRuntimeConfig() EmbeddingRuntimeConfig {
	return EmbeddingRuntimeConfig{
		MaxAttempts:         defaultEmbeddingMaxAttempts,
		PollInterval:        defaultEmbeddingPollInterval,
		ClaimTTL:            defaultEmbeddingClaimTTL,
		InitialRetryBackoff: defaultEmbeddingInitialRetryBackoff,
		MaxRetryBackoff:     defaultEmbeddingMaxRetryBackoff,
		ClaimBatchSize:      defaultEmbeddingClaimBatchSize,
	}
}

// Normalize fills empty runtime config values with defaults.
func (cfg EmbeddingRuntimeConfig) Normalize() EmbeddingRuntimeConfig {
	defaults := DefaultEmbeddingRuntimeConfig()
	if cfg.MaxAttempts <= 0 {
		cfg.MaxAttempts = defaults.MaxAttempts
	}
	if cfg.PollInterval <= 0 {
		cfg.PollInterval = defaults.PollInterval
	}
	if cfg.ClaimTTL <= 0 {
		cfg.ClaimTTL = defaults.ClaimTTL
	}
	if cfg.InitialRetryBackoff <= 0 {
		cfg.InitialRetryBackoff = defaults.InitialRetryBackoff
	}
	if cfg.MaxRetryBackoff <= 0 {
		cfg.MaxRetryBackoff = defaults.MaxRetryBackoff
	}
	if cfg.ClaimBatchSize <= 0 {
		cfg.ClaimBatchSize = defaults.ClaimBatchSize
	}
	if cfg.MaxRetryBackoff < cfg.InitialRetryBackoff {
		cfg.MaxRetryBackoff = cfg.InitialRetryBackoff
	}
	cfg.Provider = strings.TrimSpace(cfg.Provider)
	cfg.Model = strings.TrimSpace(cfg.Model)
	cfg.BaseURL = strings.TrimSpace(cfg.BaseURL)
	cfg.WorkerID = strings.TrimSpace(cfg.WorkerID)
	if strings.TrimSpace(cfg.ModelSignature) == "" && cfg.Enabled {
		cfg.ModelSignature = BuildEmbeddingModelSignature(cfg.Provider, cfg.Model, cfg.BaseURL, cfg.Dimensions)
	}
	return cfg
}

// BuildEmbeddingModelSignature computes one deterministic model/version fingerprint.
func BuildEmbeddingModelSignature(provider, model, baseURL string, dimensions int64) string {
	provider = strings.TrimSpace(strings.ToLower(provider))
	model = strings.TrimSpace(model)
	baseURL = strings.TrimSpace(baseURL)
	return fmt.Sprintf("%s|%s|%s|%d", provider, model, baseURL, dimensions)
}

// ReindexEmbeddingsInput stores explicit reindex request options.
type ReindexEmbeddingsInput struct {
	ProjectID        string
	CrossProject     bool
	IncludeArchived  bool
	Force            bool
	Wait             bool
	WaitTimeout      time.Duration
	WaitPollInterval time.Duration
}

// ReindexEmbeddingsResult stores explicit reindex outcomes.
type ReindexEmbeddingsResult struct {
	TargetProjects []string
	ScannedCount   int
	QueuedCount    int
	ReadyCount     int
	FailedCount    int
	StaleCount     int
	RunningCount   int
	PendingCount   int
	Completed      bool
	TimedOut       bool
}

// PrepareEmbeddingsLifecycle performs startup reconciliation that does not require provider calls.
func PrepareEmbeddingsLifecycle(ctx context.Context, store EmbeddingLifecycleStore, cfg EmbeddingRuntimeConfig) error {
	if store == nil || !cfg.Enabled {
		return nil
	}
	cfg = cfg.Normalize()
	if err := runEmbeddingRecoverySweep(ctx, store, time.Now().UTC(), "startup"); err != nil {
		return err
	}
	if strings.TrimSpace(cfg.ModelSignature) == "" {
		return nil
	}
	for _, subjectType := range allEmbeddingSubjectTypes() {
		staled, err := store.MarkEmbeddingsStaleByModel(ctx, EmbeddingStaleByModelInput{
			ModelProvider:   cfg.Provider,
			ModelName:       cfg.Model,
			ModelDimensions: cfg.Dimensions,
			SubjectType:     subjectType,
			ModelSignature:  cfg.ModelSignature,
			Reason:          "model_signature_changed",
			StaledAt:        time.Now().UTC(),
		})
		if err != nil {
			return fmt.Errorf("mark embeddings stale by model for %s: %w", subjectType, err)
		}
		for _, record := range staled {
			log.Info(
				"embedding lifecycle transition",
				"event", "stale",
				"subject_type", record.SubjectType,
				"subject_id", record.SubjectID,
				"project_id", record.ProjectID,
				"status", record.Status,
				"reason", record.StaleReason,
				"content_hash", record.ContentHashDesired,
				"model_signature", record.ModelSignature,
				"worker_id", "startup",
			)
		}
	}
	return nil
}

func runEmbeddingRecoverySweep(ctx context.Context, store EmbeddingLifecycleStore, now time.Time, trigger string) error {
	if store == nil {
		return nil
	}
	recovered, err := store.RecoverExpiredEmbeddingClaims(ctx, now.UTC())
	if err != nil {
		return fmt.Errorf("recover expired embedding claims: %w", err)
	}
	for _, record := range recovered {
		log.Info(
			"embedding lifecycle transition",
			"event", "retry",
			"subject_type", record.SubjectType,
			"subject_id", record.SubjectID,
			"project_id", record.ProjectID,
			"status", record.Status,
			"reason", "claim_expired",
			"attempt", record.AttemptCount,
			"retry_count", record.RetryCount,
			"next_attempt_at", record.NextAttemptAt,
			"worker_id", strings.TrimSpace(trigger),
		)
	}
	return nil
}

// EmbeddingWorker runs queued embedding jobs for one subject family.
type EmbeddingWorker struct {
	repo        Repository
	lifecycle   EmbeddingLifecycleStore
	generator   EmbeddingGenerator
	searchIndex EmbeddingSearchIndex
	clock       Clock
	cfg         EmbeddingRuntimeConfig
}

// NewEmbeddingWorker constructs one background embedding worker.
func NewEmbeddingWorker(
	repo Repository,
	lifecycle EmbeddingLifecycleStore,
	generator EmbeddingGenerator,
	searchIndex EmbeddingSearchIndex,
	clock Clock,
	cfg EmbeddingRuntimeConfig,
) *EmbeddingWorker {
	if clock == nil {
		clock = time.Now
	}
	return &EmbeddingWorker{
		repo:        repo,
		lifecycle:   lifecycle,
		generator:   generator,
		searchIndex: searchIndex,
		clock:       clock,
		cfg:         cfg.Normalize(),
	}
}

// allEmbeddingSubjectTypes returns the supported subject families in worker-processing order.
func allEmbeddingSubjectTypes() []EmbeddingSubjectType {
	return []EmbeddingSubjectType{
		EmbeddingSubjectTypeWorkItem,
		EmbeddingSubjectTypeThreadContext,
		EmbeddingSubjectTypeProjectDocument,
	}
}

// Run executes the worker loop until the context is canceled.
func (w *EmbeddingWorker) Run(ctx context.Context) error {
	if w == nil || w.repo == nil || w.lifecycle == nil || w.searchIndex == nil || w.generator == nil || !w.cfg.Enabled {
		return nil
	}
	ticker := time.NewTicker(w.cfg.PollInterval)
	defer ticker.Stop()
	for {
		if err := w.processOnce(ctx); err != nil && !errors.Is(err, context.Canceled) {
			log.Warn("embedding worker iteration failed", "err", err)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func (w *EmbeddingWorker) processOnce(ctx context.Context) error {
	if err := runEmbeddingRecoverySweep(ctx, w.lifecycle, w.clock().UTC(), "worker_poll"); err != nil {
		return err
	}
	for _, subjectType := range allEmbeddingSubjectTypes() {
		claims, err := w.lifecycle.ClaimEmbeddings(ctx, EmbeddingClaimInput{
			SubjectType: subjectType,
			WorkerID:    w.cfg.WorkerID,
			Now:         w.clock().UTC(),
			Limit:       w.cfg.ClaimBatchSize,
			ClaimTTL:    w.cfg.ClaimTTL,
		})
		if err != nil {
			return fmt.Errorf("claim %s embeddings: %w", subjectType, err)
		}
		for _, claim := range claims {
			if err := w.processClaim(ctx, claim); err != nil {
				if errors.Is(err, context.Canceled) {
					return err
				}
				log.Warn("embedding worker claim failed", "subject_type", claim.SubjectType, "subject_id", claim.SubjectID, "project_id", claim.ProjectID, "err", err)
			}
		}
	}
	return nil
}

func (w *EmbeddingWorker) processClaim(ctx context.Context, claim EmbeddingRecord) error {
	startedAt := w.clock().UTC()
	log.Info(
		"embedding lifecycle transition",
		"event", "start",
		"subject_type", claim.SubjectType,
		"subject_id", claim.SubjectID,
		"project_id", claim.ProjectID,
		"status", EmbeddingLifecycleRunning,
		"attempt", claim.AttemptCount,
		"retry_count", claim.RetryCount,
		"model_signature", w.cfg.ModelSignature,
		"worker_id", w.cfg.WorkerID,
	)

	stopHeartbeat := w.startHeartbeat(ctx, claim)
	defer stopHeartbeat()

	switch claim.SubjectType {
	case EmbeddingSubjectTypeWorkItem:
		return w.processWorkItemClaim(ctx, claim, startedAt)
	case EmbeddingSubjectTypeThreadContext:
		return w.processThreadContextClaim(ctx, claim, startedAt)
	case EmbeddingSubjectTypeProjectDocument:
		return w.processProjectDocumentClaim(ctx, claim, startedAt)
	default:
		return fmt.Errorf("unsupported embedding subject type %q", claim.SubjectType)
	}
}

func (w *EmbeddingWorker) startHeartbeat(ctx context.Context, claim EmbeddingRecord) func() {
	heartbeatCtx, cancel := context.WithCancel(ctx)
	ticker := time.NewTicker(max(time.Second, w.cfg.ClaimTTL/3))
	done := make(chan struct{})
	go func() {
		defer close(done)
		defer ticker.Stop()
		for {
			select {
			case <-heartbeatCtx.Done():
				return
			case <-ticker.C:
				err := w.lifecycle.HeartbeatEmbedding(heartbeatCtx, EmbeddingHeartbeatInput{
					SubjectType: claim.SubjectType,
					SubjectID:   claim.SubjectID,
					WorkerID:    w.cfg.WorkerID,
					Now:         w.clock().UTC(),
					ClaimTTL:    w.cfg.ClaimTTL,
				})
				if err != nil && !errors.Is(err, context.Canceled) {
					if errors.Is(err, ErrEmbeddingClaimLost) {
						log.Warn("embedding heartbeat lost claim", "subject_type", claim.SubjectType, "subject_id", claim.SubjectID, "project_id", claim.ProjectID, "worker_id", w.cfg.WorkerID)
						return
					}
					log.Warn("embedding heartbeat failed", "subject_type", claim.SubjectType, "subject_id", claim.SubjectID, "project_id", claim.ProjectID, "err", err)
				}
			}
		}
	}()
	return func() {
		cancel()
		<-done
	}
}

func (w *EmbeddingWorker) processWorkItemClaim(ctx context.Context, claim EmbeddingRecord, startedAt time.Time) error {
	actionItem, err := w.repo.GetActionItem(ctx, claim.SubjectID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return w.dropEmbeddingSubject(ctx, claim, "subject_missing")
		}
		return w.failClaim(ctx, claim, startedAt, "load_subject_failed", err, true)
	}

	content := buildActionItemEmbeddingContent(actionItem)
	if strings.TrimSpace(content) == "" {
		return w.dropEmbeddingSubject(ctx, claim, "empty_content")
	}
	return w.processResolvedDocumentClaim(ctx, claim, startedAt, EmbeddingDocument{
		SubjectType:      EmbeddingSubjectTypeWorkItem,
		SubjectID:        actionItem.ID,
		ProjectID:        actionItem.ProjectID,
		SearchTargetType: EmbeddingSearchTargetTypeWorkItem,
		SearchTargetID:   actionItem.ID,
		Content:          content,
	})
}

func (w *EmbeddingWorker) processThreadContextClaim(ctx context.Context, claim EmbeddingRecord, startedAt time.Time) error {
	target, err := parseThreadContextSubjectID(claim.SubjectID)
	if err != nil {
		return w.failClaim(ctx, claim, startedAt, "invalid_thread_context_subject_id", err, false)
	}

	var targetTitle string
	var targetBody string
	switch target.TargetType {
	case domain.CommentTargetTypeProject:
		project, getErr := w.repo.GetProject(ctx, target.ProjectID)
		if getErr != nil {
			if errors.Is(getErr, ErrNotFound) {
				return w.dropEmbeddingSubject(ctx, claim, "subject_missing")
			}
			return w.failClaim(ctx, claim, startedAt, "load_subject_failed", getErr, true)
		}
		if project.ID != target.TargetID {
			return w.dropEmbeddingSubject(ctx, claim, "subject_missing")
		}
		targetTitle = project.Name
		targetBody = project.Description
	default:
		actionItem, getErr := w.repo.GetActionItem(ctx, target.TargetID)
		if getErr != nil {
			if errors.Is(getErr, ErrNotFound) {
				return w.dropEmbeddingSubject(ctx, claim, "subject_missing")
			}
			return w.failClaim(ctx, claim, startedAt, "load_subject_failed", getErr, true)
		}
		if actionItem.ProjectID != target.ProjectID {
			return w.dropEmbeddingSubject(ctx, claim, "subject_missing")
		}
		targetTitle = actionItem.Title
		targetBody = actionItem.Description
	}

	comments, err := w.repo.ListCommentsByTarget(ctx, target)
	if err != nil {
		return w.failClaim(ctx, claim, startedAt, "load_comments_failed", err, true)
	}
	if len(comments) == 0 {
		return w.dropEmbeddingSubject(ctx, claim, "no_comments")
	}

	searchTargetType, searchTargetID, err := commentTargetEmbeddingSearchTarget(target)
	if err != nil {
		return w.failClaim(ctx, claim, startedAt, "resolve_search_target_failed", err, false)
	}
	content := buildThreadContextEmbeddingContent(target, targetTitle, targetBody, comments)
	if strings.TrimSpace(content) == "" {
		return w.dropEmbeddingSubject(ctx, claim, "empty_content")
	}
	return w.processResolvedDocumentClaim(ctx, claim, startedAt, EmbeddingDocument{
		SubjectType:      EmbeddingSubjectTypeThreadContext,
		SubjectID:        claim.SubjectID,
		ProjectID:        target.ProjectID,
		SearchTargetType: searchTargetType,
		SearchTargetID:   searchTargetID,
		Content:          content,
	})
}

func (w *EmbeddingWorker) processProjectDocumentClaim(ctx context.Context, claim EmbeddingRecord, startedAt time.Time) error {
	project, err := w.repo.GetProject(ctx, claim.SubjectID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return w.dropEmbeddingSubject(ctx, claim, "subject_missing")
		}
		return w.failClaim(ctx, claim, startedAt, "load_subject_failed", err, true)
	}
	content := buildProjectDocumentEmbeddingContent(project)
	if strings.TrimSpace(content) == "" {
		return w.dropEmbeddingSubject(ctx, claim, "empty_content")
	}
	return w.processResolvedDocumentClaim(ctx, claim, startedAt, EmbeddingDocument{
		SubjectType:      EmbeddingSubjectTypeProjectDocument,
		SubjectID:        project.ID,
		ProjectID:        project.ID,
		SearchTargetType: EmbeddingSearchTargetTypeProject,
		SearchTargetID:   project.ID,
		Content:          content,
	})
}

func (w *EmbeddingWorker) processResolvedDocumentClaim(ctx context.Context, claim EmbeddingRecord, startedAt time.Time, doc EmbeddingDocument) error {
	contentHash := hashEmbeddingContent(doc.Content)
	vectorRows, err := w.generator.Embed(ctx, []string{doc.Content})
	if err != nil {
		return w.failClaim(ctx, claim, startedAt, "embed_failed", err, true)
	}
	if len(vectorRows) == 0 || len(vectorRows[0]) == 0 {
		return w.failClaim(ctx, claim, startedAt, "empty_embedding_vector", errors.New("embedding provider returned no vector"), true)
	}
	if err := w.lifecycle.HeartbeatEmbedding(ctx, EmbeddingHeartbeatInput{
		SubjectType: claim.SubjectType,
		SubjectID:   claim.SubjectID,
		WorkerID:    w.cfg.WorkerID,
		Now:         w.clock().UTC(),
		ClaimTTL:    w.cfg.ClaimTTL,
	}); err != nil {
		return fmt.Errorf("verify embedding claim before vector upsert: %w", err)
	}
	doc.ContentHash = contentHash
	doc.Vector = append([]float32(nil), vectorRows[0]...)
	doc.UpdatedAt = w.clock().UTC()
	if err := w.searchIndex.UpsertEmbeddingDocument(ctx, doc); err != nil {
		return w.failClaim(ctx, claim, startedAt, "upsert_embedding_failed", err, true)
	}
	record, err := w.lifecycle.MarkEmbeddingSuccess(ctx, EmbeddingSuccessInput{
		SubjectType:     claim.SubjectType,
		SubjectID:       claim.SubjectID,
		ProjectID:       doc.ProjectID,
		ContentHash:     contentHash,
		ModelProvider:   w.cfg.Provider,
		ModelName:       w.cfg.Model,
		ModelDimensions: w.cfg.Dimensions,
		ModelSignature:  w.cfg.ModelSignature,
		WorkerID:        w.cfg.WorkerID,
		CompletedAt:     w.clock().UTC(),
	})
	if err != nil {
		return fmt.Errorf("mark embedding success: %w", err)
	}
	log.Info(
		"embedding lifecycle transition",
		"event", "success",
		"subject_type", record.SubjectType,
		"subject_id", record.SubjectID,
		"project_id", record.ProjectID,
		"status", record.Status,
		"content_hash", contentHash,
		"model_signature", w.cfg.ModelSignature,
		"attempt", record.AttemptCount,
		"retry_count", record.RetryCount,
		"worker_id", w.cfg.WorkerID,
		"duration_ms", time.Since(startedAt).Milliseconds(),
	)
	return nil
}

func (w *EmbeddingWorker) dropEmbeddingSubject(ctx context.Context, claim EmbeddingRecord, reason string) error {
	if err := w.searchIndex.DeleteEmbeddingDocument(ctx, claim.SubjectType, claim.SubjectID); err != nil {
		return fmt.Errorf("delete embedding document for %s/%s: %w", claim.SubjectType, claim.SubjectID, err)
	}
	if err := w.lifecycle.DeleteEmbeddingSubject(ctx, claim.SubjectType, claim.SubjectID); err != nil {
		return fmt.Errorf("delete embedding lifecycle row for %s/%s: %w", claim.SubjectType, claim.SubjectID, err)
	}
	log.Info(
		"embedding lifecycle transition",
		"event", "skip",
		"subject_type", claim.SubjectType,
		"subject_id", claim.SubjectID,
		"project_id", claim.ProjectID,
		"status", "deleted",
		"reason", strings.TrimSpace(reason),
		"worker_id", w.cfg.WorkerID,
	)
	return nil
}

func (w *EmbeddingWorker) failClaim(
	ctx context.Context,
	claim EmbeddingRecord,
	startedAt time.Time,
	errorCode string,
	err error,
	retryable bool,
) error {
	record, markErr := w.lifecycle.MarkEmbeddingFailure(ctx, EmbeddingFailureInput{
		SubjectType:         claim.SubjectType,
		SubjectID:           claim.SubjectID,
		ProjectID:           claim.ProjectID,
		ModelSignature:      w.cfg.ModelSignature,
		WorkerID:            w.cfg.WorkerID,
		Retryable:           retryable,
		ErrorCode:           strings.TrimSpace(errorCode),
		ErrorMessage:        err.Error(),
		ErrorSummary:        err.Error(),
		FailedAt:            w.clock().UTC(),
		InitialRetryBackoff: w.cfg.InitialRetryBackoff,
		MaxRetryBackoff:     w.cfg.MaxRetryBackoff,
	})
	if markErr != nil {
		return fmt.Errorf("mark embedding failure after %q: %w", errorCode, markErr)
	}
	log.Warn(
		"embedding lifecycle transition",
		"event", "fail",
		"subject_type", record.SubjectType,
		"subject_id", record.SubjectID,
		"project_id", record.ProjectID,
		"status", record.Status,
		"reason", errorCode,
		"model_signature", w.cfg.ModelSignature,
		"attempt", record.AttemptCount,
		"retry_count", record.RetryCount,
		"next_attempt_at", record.NextAttemptAt,
		"worker_id", w.cfg.WorkerID,
		"duration_ms", time.Since(startedAt).Milliseconds(),
		"err", err,
	)
	if retryable && record.NextAttemptAt != nil {
		log.Info(
			"embedding lifecycle transition",
			"event", "retry",
			"subject_type", record.SubjectType,
			"subject_id", record.SubjectID,
			"project_id", record.ProjectID,
			"status", record.Status,
			"reason", errorCode,
			"attempt", record.AttemptCount,
			"retry_count", record.RetryCount,
			"next_attempt_at", record.NextAttemptAt,
			"worker_id", w.cfg.WorkerID,
		)
	}
	return nil
}

func (s *Service) embeddingLifecycleEnabled() bool {
	return s != nil && s.embeddingRuntime.Enabled && s.embeddingLifecycle != nil && strings.TrimSpace(s.embeddingRuntime.ModelSignature) != ""
}

// EmbeddingsOperational reports whether the runtime has enough wiring to process embeddings work.
func (s *Service) EmbeddingsOperational() bool {
	return s != nil &&
		s.embeddingRuntime.Enabled &&
		s.embeddingLifecycle != nil &&
		s.searchIndex != nil &&
		s.embeddingGenerator != nil &&
		strings.TrimSpace(s.embeddingRuntime.ModelSignature) != ""
}

func (s *Service) enqueueEmbeddingSubject(
	ctx context.Context,
	subjectType EmbeddingSubjectType,
	subjectID, projectID, content string,
	force bool,
	reason string,
) (EmbeddingRecord, error) {
	if s == nil || !s.embeddingLifecycleEnabled() {
		return EmbeddingRecord{}, nil
	}
	record, err := s.embeddingLifecycle.EnqueueEmbedding(ctx, EmbeddingEnqueueInput{
		SubjectType:     subjectType,
		SubjectID:       strings.TrimSpace(subjectID),
		ProjectID:       strings.TrimSpace(projectID),
		ContentHash:     hashEmbeddingContent(content),
		ModelProvider:   s.embeddingRuntime.Provider,
		ModelName:       s.embeddingRuntime.Model,
		ModelDimensions: s.embeddingRuntime.Dimensions,
		ModelSignature:  s.embeddingRuntime.ModelSignature,
		MaxAttempts:     s.embeddingRuntime.MaxAttempts,
		Force:           force,
		Reason:          strings.TrimSpace(reason),
		EnqueuedAt:      s.clock().UTC(),
	})
	if err != nil {
		return EmbeddingRecord{}, err
	}
	log.Info(
		"embedding lifecycle transition",
		"event", "enqueue",
		"subject_type", record.SubjectType,
		"subject_id", record.SubjectID,
		"project_id", record.ProjectID,
		"status", record.Status,
		"reason", strings.TrimSpace(reason),
		"content_hash", record.ContentHashDesired,
		"model_signature", record.ModelSignature,
		"max_attempts", record.MaxAttempts,
	)
	if record.Status == EmbeddingLifecycleStale {
		staleReason := strings.TrimSpace(record.StaleReason)
		if staleReason == "" {
			staleReason = strings.TrimSpace(reason)
		}
		log.Info(
			"embedding lifecycle transition",
			"event", "stale",
			"subject_type", record.SubjectType,
			"subject_id", record.SubjectID,
			"project_id", record.ProjectID,
			"status", record.Status,
			"reason", staleReason,
			"content_hash", record.ContentHashDesired,
			"model_signature", record.ModelSignature,
		)
	}
	return record, nil
}

func (s *Service) enqueueActionItemEmbedding(ctx context.Context, actionItem domain.ActionItem, force bool, reason string) (EmbeddingRecord, error) {
	content := buildActionItemEmbeddingContent(actionItem)
	return s.enqueueEmbeddingSubject(ctx, EmbeddingSubjectTypeWorkItem, actionItem.ID, actionItem.ProjectID, content, force, reason)
}

func (s *Service) enqueueProjectDocumentEmbedding(ctx context.Context, project domain.Project, force bool, reason string) (EmbeddingRecord, error) {
	content := buildProjectDocumentEmbeddingContent(project)
	return s.enqueueEmbeddingSubject(ctx, EmbeddingSubjectTypeProjectDocument, project.ID, project.ID, content, force, reason)
}

func (s *Service) enqueueThreadContextEmbedding(ctx context.Context, target domain.CommentTarget, force bool, reason string) (EmbeddingRecord, error) {
	target, err := domain.NormalizeCommentTarget(target)
	if err != nil {
		return EmbeddingRecord{}, err
	}

	var targetTitle string
	var targetBody string
	switch target.TargetType {
	case domain.CommentTargetTypeProject:
		project, getErr := s.repo.GetProject(ctx, target.ProjectID)
		if getErr != nil {
			return EmbeddingRecord{}, getErr
		}
		if project.ID != target.TargetID {
			return EmbeddingRecord{}, ErrNotFound
		}
		targetTitle = project.Name
		targetBody = project.Description
	default:
		actionItem, getErr := s.repo.GetActionItem(ctx, target.TargetID)
		if getErr != nil {
			return EmbeddingRecord{}, getErr
		}
		if actionItem.ProjectID != target.ProjectID {
			return EmbeddingRecord{}, ErrNotFound
		}
		targetTitle = actionItem.Title
		targetBody = actionItem.Description
	}

	comments, err := s.repo.ListCommentsByTarget(ctx, target)
	if err != nil {
		return EmbeddingRecord{}, err
	}
	if len(comments) == 0 {
		return EmbeddingRecord{}, nil
	}
	subjectID, err := buildThreadContextSubjectID(target)
	if err != nil {
		return EmbeddingRecord{}, err
	}
	content := buildThreadContextEmbeddingContent(target, targetTitle, targetBody, comments)
	return s.enqueueEmbeddingSubject(ctx, EmbeddingSubjectTypeThreadContext, subjectID, target.ProjectID, content, force, reason)
}

// ListEmbeddingStates returns durable lifecycle rows for operator surfaces.
func (s *Service) ListEmbeddingStates(ctx context.Context, filter EmbeddingListFilter) ([]EmbeddingRecord, error) {
	if s == nil || s.embeddingLifecycle == nil {
		return []EmbeddingRecord{}, nil
	}
	if filter.ProjectIDs != nil && len(filter.ProjectIDs) == 0 {
		return []EmbeddingRecord{}, nil
	}
	return s.embeddingLifecycle.ListEmbeddings(ctx, filter)
}

// SummarizeEmbeddingStates returns aggregate lifecycle counts for operator surfaces.
func (s *Service) SummarizeEmbeddingStates(ctx context.Context, filter EmbeddingListFilter) (EmbeddingSummary, error) {
	if s == nil || s.embeddingLifecycle == nil {
		return EmbeddingSummary{SubjectType: filter.SubjectType, ProjectIDs: append([]string(nil), filter.ProjectIDs...)}, nil
	}
	if filter.ProjectIDs != nil && len(filter.ProjectIDs) == 0 {
		return EmbeddingSummary{SubjectType: filter.SubjectType, ProjectIDs: append([]string(nil), filter.ProjectIDs...)}, nil
	}
	return s.embeddingLifecycle.SummarizeEmbeddings(ctx, filter)
}

// ReindexEmbeddings enqueues lifecycle work for existing indexed subjects and optionally waits for steady state.
func (s *Service) ReindexEmbeddings(ctx context.Context, in ReindexEmbeddingsInput) (ReindexEmbeddingsResult, error) {
	if s == nil {
		return ReindexEmbeddingsResult{}, fmt.Errorf("app service is not configured")
	}
	if !s.embeddingRuntime.Enabled || s.embeddingLifecycle == nil {
		return ReindexEmbeddingsResult{}, ErrEmbeddingsDisabled
	}
	targetProjects, err := s.reindexTargetProjects(ctx, in)
	if err != nil {
		return ReindexEmbeddingsResult{}, err
	}
	result := ReindexEmbeddingsResult{
		TargetProjects: targetProjects,
	}
	for _, projectID := range targetProjects {
		project, projectErr := s.repo.GetProject(ctx, projectID)
		if projectErr != nil {
			return ReindexEmbeddingsResult{}, projectErr
		}
		result.ScannedCount++
		record, enqueueErr := s.enqueueProjectDocumentEmbedding(ctx, project, in.Force, "manual_reindex")
		if enqueueErr != nil {
			return ReindexEmbeddingsResult{}, enqueueErr
		}
		accumulateReindexRecord(&result, record)

		tasks, listErr := s.repo.ListActionItems(ctx, projectID, true)
		if listErr != nil {
			return ReindexEmbeddingsResult{}, listErr
		}
		actionItemByID := make(map[string]domain.ActionItem, len(tasks))
		for _, actionItem := range tasks {
			actionItemByID[actionItem.ID] = actionItem
			if actionItem.ArchivedAt != nil && !in.IncludeArchived {
				continue
			}
			result.ScannedCount++
			record, enqueueErr := s.enqueueActionItemEmbedding(ctx, actionItem, in.Force, "manual_reindex")
			if enqueueErr != nil {
				return ReindexEmbeddingsResult{}, enqueueErr
			}
			accumulateReindexRecord(&result, record)
		}

		targets, targetsErr := s.repo.ListCommentTargets(ctx, projectID)
		if targetsErr != nil {
			return ReindexEmbeddingsResult{}, targetsErr
		}
		for _, target := range targets {
			if target.TargetType != domain.CommentTargetTypeProject && !in.IncludeArchived {
				actionItem, ok := actionItemByID[target.TargetID]
				if ok && actionItem.ArchivedAt != nil {
					continue
				}
			}
			result.ScannedCount++
			record, enqueueErr := s.enqueueThreadContextEmbedding(ctx, target, in.Force, "manual_reindex")
			if enqueueErr != nil {
				if errors.Is(enqueueErr, ErrNotFound) {
					continue
				}
				return ReindexEmbeddingsResult{}, enqueueErr
			}
			accumulateReindexRecord(&result, record)
		}
	}
	if !in.Wait {
		return result, nil
	}
	waitSummary, timedOut, err := s.waitForEmbeddingSteadyState(ctx, targetProjects, in)
	if err != nil {
		return ReindexEmbeddingsResult{}, err
	}
	result.PendingCount = waitSummary.PendingCount
	result.RunningCount = waitSummary.RunningCount
	result.ReadyCount = waitSummary.ReadyCount
	result.FailedCount = waitSummary.FailedCount
	result.StaleCount = waitSummary.StaleCount
	result.Completed = waitSummary.PendingCount == 0 && waitSummary.RunningCount == 0 && waitSummary.StaleCount == 0 && waitSummary.FailedCount == 0
	result.TimedOut = timedOut
	return result, nil
}

func (s *Service) reindexTargetProjects(ctx context.Context, in ReindexEmbeddingsInput) ([]string, error) {
	if in.CrossProject {
		projects, err := s.repo.ListProjects(ctx, in.IncludeArchived)
		if err != nil {
			return nil, err
		}
		out := make([]string, 0, len(projects))
		for _, project := range projects {
			out = append(out, project.ID)
		}
		return out, nil
	}
	projectID := strings.TrimSpace(in.ProjectID)
	if projectID == "" {
		return nil, domain.ErrInvalidID
	}
	if _, err := s.repo.GetProject(ctx, projectID); err != nil {
		return nil, err
	}
	return []string{projectID}, nil
}

func (s *Service) waitForEmbeddingSteadyState(ctx context.Context, projectIDs []string, in ReindexEmbeddingsInput) (EmbeddingSummary, bool, error) {
	pollInterval := in.WaitPollInterval
	if pollInterval <= 0 {
		pollInterval = 2 * time.Second
	}
	timeout := in.WaitTimeout
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	deadline := s.clock().UTC().Add(timeout)
	for {
		summary, err := s.embeddingLifecycle.SummarizeEmbeddings(ctx, EmbeddingListFilter{
			ProjectIDs: append([]string(nil), projectIDs...),
		})
		if err != nil {
			return EmbeddingSummary{}, false, err
		}
		if summary.PendingCount == 0 && summary.RunningCount == 0 && summary.StaleCount == 0 {
			return summary, false, nil
		}
		if s.clock().UTC().After(deadline) {
			return summary, true, nil
		}
		select {
		case <-ctx.Done():
			return EmbeddingSummary{}, false, ctx.Err()
		case <-time.After(pollInterval):
		}
	}
}

func accumulateReindexRecord(result *ReindexEmbeddingsResult, record EmbeddingRecord) {
	if result == nil || record.SubjectType == "" {
		return
	}
	switch record.Status {
	case EmbeddingLifecycleReady:
		result.ReadyCount++
	case EmbeddingLifecycleFailed:
		result.FailedCount++
	case EmbeddingLifecycleStale:
		result.StaleCount++
		result.QueuedCount++
	case EmbeddingLifecycleRunning:
		result.RunningCount++
		result.QueuedCount++
	case EmbeddingLifecyclePending:
		result.PendingCount++
		result.QueuedCount++
	}
}
