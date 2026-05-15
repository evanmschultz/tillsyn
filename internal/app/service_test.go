package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	fantasyembed "github.com/evanmschultz/tillsyn/internal/adapters/embeddings/fantasy"
	"github.com/evanmschultz/tillsyn/internal/domain"
	"github.com/evanmschultz/tillsyn/internal/templates"
)

// ptrTo returns a pointer to v. Test-only convenience for the
// pointer-sentinel UpdateActionItemInput shape introduced by Drop 4c.5
// droplet A.1; co-located here so test bodies stay readable instead of
// each case allocating a named local just to take its address.
func ptrTo[T any](v T) *T {
	return &v
}

// fakeRepo represents fake repo data used by this package.
type fakeRepo struct {
	projects              map[string]domain.Project
	columns               map[string]domain.Column
	tasks                 map[string]domain.ActionItem
	comments              map[string][]domain.Comment
	attentionItems        map[string]domain.AttentionItem
	authRequests          map[string]domain.AuthRequest
	handoffs              map[string]domain.Handoff
	changeEvents          map[string][]domain.ChangeEvent
	kindDefs              map[domain.KindID]domain.KindDefinition
	projectAllowedKinds   map[string][]domain.KindID
	capabilityLeases      map[string]domain.CapabilityLease
	createProjectActor    MutationActor
	updateProjectActor    MutationActor
	createActionItemActor MutationActor
	updateActionItemActor MutationActor
	createCommentActor    MutationActor
}

// newFakeRepo constructs fake repo.
//
// Post-Drop-1.75 the app-layer kind-catalog bootstrap helper has been deleted;
// the SQLite schema seeds the 12-value Kind enum via CREATE-time INSERT OR
// IGNORE, but this in-memory fake never runs SQL migrations. Seed every
// member of the new enum so tests have all kind IDs available for fixture
// construction.
func newFakeRepo() *fakeRepo {
	now := time.Now().UTC()
	kinds := []struct {
		kind      domain.Kind
		display   string
		appliesTo domain.KindAppliesTo
	}{
		{kind: domain.KindPlan, display: "Plan", appliesTo: domain.KindAppliesToPlan},
		{kind: domain.KindResearch, display: "Research", appliesTo: domain.KindAppliesToResearch},
		{kind: domain.KindBuild, display: "Build", appliesTo: domain.KindAppliesToBuild},
		{kind: domain.KindPlanQAProof, display: "Plan QA Proof", appliesTo: domain.KindAppliesToPlanQAProof},
		{kind: domain.KindPlanQAFalsification, display: "Plan QA Falsification", appliesTo: domain.KindAppliesToPlanQAFalsification},
		{kind: domain.KindBuildQAProof, display: "Build QA Proof", appliesTo: domain.KindAppliesToBuildQAProof},
		{kind: domain.KindBuildQAFalsification, display: "Build QA Falsification", appliesTo: domain.KindAppliesToBuildQAFalsification},
		{kind: domain.KindCloseout, display: "Closeout", appliesTo: domain.KindAppliesToCloseout},
		{kind: domain.KindCommit, display: "Commit", appliesTo: domain.KindAppliesToCommit},
		{kind: domain.KindRefinement, display: "Refinement", appliesTo: domain.KindAppliesToRefinement},
		{kind: domain.KindDiscussion, display: "Discussion", appliesTo: domain.KindAppliesToDiscussion},
		{kind: domain.KindHumanVerify, display: "Human Verify", appliesTo: domain.KindAppliesToHumanVerify},
	}
	kindDefs := map[domain.KindID]domain.KindDefinition{}
	for _, entry := range kinds {
		id := domain.KindID(entry.kind)
		kindDefs[id] = domain.KindDefinition{
			ID:          id,
			DisplayName: entry.display,
			AppliesTo:   []domain.KindAppliesTo{entry.appliesTo},
			CreatedAt:   now,
			UpdatedAt:   now,
		}
	}
	return &fakeRepo{
		projects:            map[string]domain.Project{},
		columns:             map[string]domain.Column{},
		tasks:               map[string]domain.ActionItem{},
		comments:            map[string][]domain.Comment{},
		attentionItems:      map[string]domain.AttentionItem{},
		authRequests:        map[string]domain.AuthRequest{},
		handoffs:            map[string]domain.Handoff{},
		changeEvents:        map[string][]domain.ChangeEvent{},
		kindDefs:            kindDefs,
		projectAllowedKinds: map[string][]domain.KindID{},
		capabilityLeases:    map[string]domain.CapabilityLease{},
	}
}

// fakeEmbeddingGenerator captures embedding requests for service tests.
type fakeEmbeddingGenerator struct {
	vectors [][]float32
	err     error
	inputs  [][]string
}

// Embed returns one configured set of vectors and records inputs.
func (f *fakeEmbeddingGenerator) Embed(_ context.Context, inputs []string) ([][]float32, error) {
	copyInputs := append([]string(nil), inputs...)
	f.inputs = append(f.inputs, copyInputs)
	if f.err != nil {
		return nil, f.err
	}
	out := make([][]float32, 0, len(f.vectors))
	for _, row := range f.vectors {
		out = append(out, append([]float32(nil), row...))
	}
	return out, nil
}

// fakeActionItemSearchIndex captures semantic document writes/search requests in tests.
type fakeActionItemSearchIndex struct {
	upserts            []EmbeddingDocument
	deletedSubjectIDs  []string
	deletedSubjectType []EmbeddingSubjectType
	searchIn           EmbeddingSearchInput
	searchRows         []EmbeddingSearchMatch
	searchErr          error
}

// UpsertEmbeddingDocument stores one in-memory upsert call.
func (f *fakeActionItemSearchIndex) UpsertEmbeddingDocument(_ context.Context, in EmbeddingDocument) error {
	doc := in
	doc.Vector = append([]float32(nil), in.Vector...)
	f.upserts = append(f.upserts, doc)
	return nil
}

// DeleteEmbeddingDocument stores one in-memory delete call.
func (f *fakeActionItemSearchIndex) DeleteEmbeddingDocument(_ context.Context, subjectType EmbeddingSubjectType, subjectID string) error {
	f.deletedSubjectType = append(f.deletedSubjectType, subjectType)
	f.deletedSubjectIDs = append(f.deletedSubjectIDs, subjectID)
	return nil
}

// SearchEmbeddingDocuments returns configured semantic match rows.
func (f *fakeActionItemSearchIndex) SearchEmbeddingDocuments(_ context.Context, in EmbeddingSearchInput) ([]EmbeddingSearchMatch, error) {
	f.searchIn = in
	if f.searchErr != nil {
		return nil, f.searchErr
	}
	out := make([]EmbeddingSearchMatch, 0, len(f.searchRows))
	for _, row := range f.searchRows {
		out = append(out, row)
	}
	return out, nil
}

// fakeEmbeddingLifecycleStore captures lifecycle operations for service tests.
type fakeEmbeddingLifecycleStore struct {
	enqueues         map[string]EmbeddingRecord
	inputs           []EmbeddingEnqueueInput
	summarySequence  []EmbeddingSummary
	summaryCallCount int
	recoverCalls     int
	recoveredCount   int
	recoverErr       error
}

// newFakeEmbeddingLifecycleStore constructs one in-memory lifecycle stub.
func newFakeEmbeddingLifecycleStore() *fakeEmbeddingLifecycleStore {
	return &fakeEmbeddingLifecycleStore{
		enqueues: map[string]EmbeddingRecord{},
	}
}

// EnqueueEmbedding records one durable enqueue request.
func (f *fakeEmbeddingLifecycleStore) EnqueueEmbedding(_ context.Context, in EmbeddingEnqueueInput) (EmbeddingRecord, error) {
	record := EmbeddingRecord{
		SubjectType:        in.SubjectType,
		SubjectID:          strings.TrimSpace(in.SubjectID),
		ProjectID:          strings.TrimSpace(in.ProjectID),
		ContentHashDesired: strings.TrimSpace(in.ContentHash),
		ModelProvider:      strings.TrimSpace(in.ModelProvider),
		ModelName:          strings.TrimSpace(in.ModelName),
		ModelDimensions:    in.ModelDimensions,
		ModelSignature:     strings.TrimSpace(in.ModelSignature),
		Status:             EmbeddingLifecyclePending,
		MaxAttempts:        in.MaxAttempts,
	}
	f.inputs = append(f.inputs, in)
	f.enqueues[f.embeddingKey(in.SubjectType, in.SubjectID)] = record
	return record, nil
}

// DeleteEmbeddingSubject removes one lifecycle row from the in-memory store.
func (f *fakeEmbeddingLifecycleStore) DeleteEmbeddingSubject(_ context.Context, subjectType EmbeddingSubjectType, subjectID string) error {
	delete(f.enqueues, f.embeddingKey(subjectType, subjectID))
	return nil
}

// ListEmbeddings lists lifecycle rows that satisfy the requested filters.
func (f *fakeEmbeddingLifecycleStore) ListEmbeddings(_ context.Context, filter EmbeddingListFilter) ([]EmbeddingRecord, error) {
	out := make([]EmbeddingRecord, 0, len(f.enqueues))
	for _, record := range f.enqueues {
		if filter.SubjectType != "" && record.SubjectType != filter.SubjectType {
			continue
		}
		if len(filter.ProjectIDs) > 0 && !containsString(filter.ProjectIDs, record.ProjectID) {
			continue
		}
		if len(filter.SubjectIDs) > 0 && !containsString(filter.SubjectIDs, record.SubjectID) {
			continue
		}
		if len(filter.Statuses) > 0 && !containsEmbeddingStatus(filter.Statuses, record.Status) {
			continue
		}
		out = append(out, record)
	}
	if filter.Limit > 0 && len(out) > filter.Limit {
		out = out[:filter.Limit]
	}
	return out, nil
}

// SummarizeEmbeddings aggregates lifecycle counts for the requested filter.
func (f *fakeEmbeddingLifecycleStore) SummarizeEmbeddings(ctx context.Context, filter EmbeddingListFilter) (EmbeddingSummary, error) {
	if len(f.summarySequence) > 0 {
		idx := min(f.summaryCallCount, len(f.summarySequence)-1)
		f.summaryCallCount++
		summary := f.summarySequence[idx]
		if summary.SubjectType == "" {
			summary.SubjectType = filter.SubjectType
		}
		if summary.ProjectIDs == nil {
			summary.ProjectIDs = append([]string(nil), filter.ProjectIDs...)
		}
		return summary, nil
	}
	rows, err := f.ListEmbeddings(ctx, filter)
	if err != nil {
		return EmbeddingSummary{}, err
	}
	summary := EmbeddingSummary{
		SubjectType: filter.SubjectType,
		ProjectIDs:  append([]string(nil), filter.ProjectIDs...),
	}
	for _, row := range rows {
		switch row.Status {
		case EmbeddingLifecyclePending:
			summary.PendingCount++
		case EmbeddingLifecycleRunning:
			summary.RunningCount++
		case EmbeddingLifecycleReady:
			summary.ReadyCount++
		case EmbeddingLifecycleFailed:
			summary.FailedCount++
		case EmbeddingLifecycleStale:
			summary.StaleCount++
		}
	}
	return summary, nil
}

// ClaimEmbeddings returns no claimed work for service-layer tests.
func (f *fakeEmbeddingLifecycleStore) ClaimEmbeddings(_ context.Context, _ EmbeddingClaimInput) ([]EmbeddingRecord, error) {
	return nil, nil
}

// HeartbeatEmbedding is a no-op for service-layer tests.
func (f *fakeEmbeddingLifecycleStore) HeartbeatEmbedding(_ context.Context, _ EmbeddingHeartbeatInput) error {
	return nil
}

// MarkEmbeddingSuccess updates one in-memory row to ready.
func (f *fakeEmbeddingLifecycleStore) MarkEmbeddingSuccess(_ context.Context, in EmbeddingSuccessInput) (EmbeddingRecord, error) {
	record := f.enqueues[f.embeddingKey(in.SubjectType, in.SubjectID)]
	record.Status = EmbeddingLifecycleReady
	record.ContentHashIndexed = in.ContentHash
	f.enqueues[f.embeddingKey(in.SubjectType, in.SubjectID)] = record
	return record, nil
}

// MarkEmbeddingFailure updates one in-memory row to failed.
func (f *fakeEmbeddingLifecycleStore) MarkEmbeddingFailure(_ context.Context, in EmbeddingFailureInput) (EmbeddingRecord, error) {
	record := f.enqueues[f.embeddingKey(in.SubjectType, in.SubjectID)]
	record.Status = EmbeddingLifecycleFailed
	record.LastErrorSummary = in.ErrorSummary
	f.enqueues[f.embeddingKey(in.SubjectType, in.SubjectID)] = record
	return record, nil
}

// RecoverExpiredEmbeddingClaims reports recovered rows for service-layer tests.
func (f *fakeEmbeddingLifecycleStore) RecoverExpiredEmbeddingClaims(_ context.Context, _ time.Time) ([]EmbeddingRecord, error) {
	f.recoverCalls++
	if f.recoverErr != nil {
		return nil, f.recoverErr
	}
	out := make([]EmbeddingRecord, 0, f.recoveredCount)
	for idx := 0; idx < f.recoveredCount; idx++ {
		out = append(out, EmbeddingRecord{
			SubjectType: EmbeddingSubjectTypeWorkItem,
			SubjectID:   fmt.Sprintf("recovered-%d", idx+1),
			Status:      EmbeddingLifecyclePending,
		})
	}
	return out, nil
}

// MarkEmbeddingsStaleByModel marks rows stale when their stored model signature differs from runtime.
func (f *fakeEmbeddingLifecycleStore) MarkEmbeddingsStaleByModel(_ context.Context, in EmbeddingStaleByModelInput) ([]EmbeddingRecord, error) {
	out := make([]EmbeddingRecord, 0)
	for key, record := range f.enqueues {
		if record.SubjectType != in.SubjectType {
			continue
		}
		if strings.TrimSpace(record.ModelSignature) == strings.TrimSpace(in.ModelSignature) {
			continue
		}
		record.Status = EmbeddingLifecycleStale
		record.StaleReason = in.Reason
		record.ModelSignature = in.ModelSignature
		f.enqueues[key] = record
		out = append(out, record)
	}
	return out, nil
}

// embeddingKey builds one stable in-memory lifecycle row key.
func (f *fakeEmbeddingLifecycleStore) embeddingKey(subjectType EmbeddingSubjectType, subjectID string) string {
	return string(subjectType) + "::" + strings.TrimSpace(subjectID)
}

// seedReadyActionItemEmbedding stores ready lifecycle rows for the provided tasks.
func seedReadyActionItemEmbeddings(lifecycle *fakeEmbeddingLifecycleStore, projectID string, tasks ...domain.ActionItem) {
	if lifecycle == nil {
		return
	}
	for _, actionItem := range tasks {
		lifecycle.enqueues[lifecycle.embeddingKey(EmbeddingSubjectTypeWorkItem, actionItem.ID)] = EmbeddingRecord{
			SubjectType:        EmbeddingSubjectTypeWorkItem,
			SubjectID:          actionItem.ID,
			ProjectID:          projectID,
			Status:             EmbeddingLifecycleReady,
			ContentHashDesired: hashEmbeddingContent(buildActionItemEmbeddingContent(actionItem)),
			ContentHashIndexed: hashEmbeddingContent(buildActionItemEmbeddingContent(actionItem)),
		}
	}
}

// seedReadyThreadContextEmbeddings stores ready lifecycle rows for the provided comment targets.
func seedReadyThreadContextEmbeddings(lifecycle *fakeEmbeddingLifecycleStore, projectID string, targets ...domain.CommentTarget) {
	if lifecycle == nil {
		return
	}
	for _, target := range targets {
		subjectID := BuildThreadContextSubjectID(target)
		lifecycle.enqueues[lifecycle.embeddingKey(EmbeddingSubjectTypeThreadContext, subjectID)] = EmbeddingRecord{
			SubjectType:        EmbeddingSubjectTypeThreadContext,
			SubjectID:          subjectID,
			ProjectID:          projectID,
			Status:             EmbeddingLifecycleReady,
			ContentHashDesired: hashEmbeddingContent(subjectID),
			ContentHashIndexed: hashEmbeddingContent(subjectID),
		}
	}
}

// newSecondWaveEmbeddingService seeds one fake-repo service used by second-wave app-layer embeddings tests.
func newSecondWaveEmbeddingService(t *testing.T, now time.Time, idGen func() string) (*Service, *fakeRepo, *fakeEmbeddingLifecycleStore, domain.Project, domain.Column, domain.ActionItem) {
	t.Helper()

	repo := newFakeRepo()
	project, err := domain.NewProjectFromInput(domain.ProjectInput{ID: "p-second-wave", Name: "Second Wave", Description: "Project description"}, now)
	if err != nil {
		t.Fatalf("NewProjectFromInput() error = %v", err)
	}
	project.Metadata = domain.ProjectMetadata{
		Tags:              []string{"embeddings", "docs"},
		StandardsMarkdown: "Follow the documented review path.",
	}
	if err := repo.CreateProject(context.Background(), project); err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}
	column, err := domain.NewColumn("c-second-wave", project.ID, "To Do", 0, 0, now)
	if err != nil {
		t.Fatalf("NewColumn() error = %v", err)
	}
	if err := repo.CreateColumn(context.Background(), column); err != nil {
		t.Fatalf("CreateColumn() error = %v", err)
	}
	actionItem, err := domain.NewActionItemForTest(domain.ActionItemInput{
		Kind:        domain.KindPlan,
		ID:          "t-second-wave",
		ProjectID:   project.ID,
		ColumnID:    column.ID,
		Position:    0,
		Title:       "Ship thread-context search",
		Description: "Tune semantic retrieval",
		Priority:    domain.PriorityMedium,
	}, now)
	if err != nil {
		t.Fatalf("NewActionItem() error = %v", err)
	}
	if err := repo.CreateActionItem(context.Background(), actionItem); err != nil {
		t.Fatalf("CreateActionItem() error = %v", err)
	}
	lifecycle := newFakeEmbeddingLifecycleStore()

	svc := NewService(repo, idGen, func() time.Time { return now }, ServiceConfig{
		EmbeddingLifecycle: lifecycle,
		EmbeddingRuntime: EmbeddingRuntimeConfig{
			Enabled:        true,
			Provider:       "deterministic",
			Model:          "hash-bow-v1",
			Dimensions:     32,
			ModelSignature: BuildEmbeddingModelSignature("deterministic", "hash-bow-v1", "", 32),
			MaxAttempts:    5,
		},
	})
	return svc, repo, lifecycle, project, column, actionItem
}

// mustDeterministicEmbeddingGenerator returns one deterministic embedding generator for second-wave search tests.
func mustDeterministicEmbeddingGenerator(t *testing.T, dims int64) EmbeddingGenerator {
	t.Helper()

	gen, err := fantasyembed.New(context.Background(), fantasyembed.Config{
		Provider:   "deterministic",
		Model:      "hash-bow-v1",
		Dimensions: dims,
	})
	if err != nil {
		t.Fatalf("fantasyembed.New() error = %v", err)
	}
	return gen
}

// containsString reports whether one trimmed string list contains the requested value.
func containsString(values []string, target string) bool {
	target = strings.TrimSpace(target)
	for _, value := range values {
		if strings.TrimSpace(value) == target {
			return true
		}
	}
	return false
}

// containsEmbeddingStatus reports whether one lifecycle status slice contains the requested status.
func containsEmbeddingStatus(values []EmbeddingLifecycleStatus, target EmbeddingLifecycleStatus) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

// CreateProject creates project.
func (f *fakeRepo) CreateProject(ctx context.Context, p domain.Project) error {
	f.createProjectActor, _ = MutationActorFromContext(ctx)
	f.projects[p.ID] = p
	return nil
}

// UpdateProject updates state for the requested operation.
func (f *fakeRepo) UpdateProject(ctx context.Context, p domain.Project) error {
	f.updateProjectActor, _ = MutationActorFromContext(ctx)
	f.projects[p.ID] = p
	return nil
}

// DeleteProject deletes one project.
func (f *fakeRepo) DeleteProject(_ context.Context, id string) error {
	if _, ok := f.projects[id]; !ok {
		return ErrNotFound
	}
	delete(f.projects, id)
	for columnID, column := range f.columns {
		if column.ProjectID != id {
			continue
		}
		delete(f.columns, columnID)
	}
	for actionItemID, actionItem := range f.tasks {
		if actionItem.ProjectID != id {
			continue
		}
		delete(f.tasks, actionItemID)
	}
	return nil
}

// GetProject returns project.
func (f *fakeRepo) GetProject(_ context.Context, id string) (domain.Project, error) {
	p, ok := f.projects[id]
	if !ok {
		return domain.Project{}, ErrNotFound
	}
	return p, nil
}

// GetProjectBySlug returns the project whose slug equals the supplied value.
func (f *fakeRepo) GetProjectBySlug(_ context.Context, slug string) (domain.Project, error) {
	for _, p := range f.projects {
		if p.Slug == slug {
			return p, nil
		}
	}
	return domain.Project{}, ErrNotFound
}

// ListProjects lists projects.
func (f *fakeRepo) ListProjects(_ context.Context, includeArchived bool) ([]domain.Project, error) {
	out := make([]domain.Project, 0, len(f.projects))
	for _, p := range f.projects {
		if !includeArchived && p.ArchivedAt != nil {
			continue
		}
		out = append(out, p)
	}
	return out, nil
}

// SetProjectAllowedKinds updates one project's kind allowlist.
func (f *fakeRepo) SetProjectAllowedKinds(_ context.Context, projectID string, kindIDs []domain.KindID) error {
	f.projectAllowedKinds[projectID] = append([]domain.KindID(nil), kindIDs...)
	return nil
}

// ListProjectAllowedKinds lists one project's kind allowlist.
func (f *fakeRepo) ListProjectAllowedKinds(_ context.Context, projectID string) ([]domain.KindID, error) {
	return append([]domain.KindID(nil), f.projectAllowedKinds[projectID]...), nil
}

// CreateKindDefinition creates one kind definition.
func (f *fakeRepo) CreateKindDefinition(_ context.Context, kind domain.KindDefinition) error {
	f.kindDefs[kind.ID] = kind
	return nil
}

// UpdateKindDefinition updates one kind definition.
func (f *fakeRepo) UpdateKindDefinition(_ context.Context, kind domain.KindDefinition) error {
	if _, ok := f.kindDefs[kind.ID]; !ok {
		return ErrNotFound
	}
	f.kindDefs[kind.ID] = kind
	return nil
}

// GetKindDefinition returns one kind definition by ID.
func (f *fakeRepo) GetKindDefinition(_ context.Context, kindID domain.KindID) (domain.KindDefinition, error) {
	kind, ok := f.kindDefs[kindID]
	if !ok {
		return domain.KindDefinition{}, ErrNotFound
	}
	return kind, nil
}

// ListKindDefinitions lists kind definitions.
func (f *fakeRepo) ListKindDefinitions(_ context.Context, includeArchived bool) ([]domain.KindDefinition, error) {
	out := make([]domain.KindDefinition, 0, len(f.kindDefs))
	for _, kind := range f.kindDefs {
		if !includeArchived && kind.ArchivedAt != nil {
			continue
		}
		out = append(out, kind)
	}
	return out, nil
}

// CreateColumn creates column.
func (f *fakeRepo) CreateColumn(_ context.Context, c domain.Column) error {
	f.columns[c.ID] = c
	return nil
}

// UpdateColumn updates state for the requested operation.
func (f *fakeRepo) UpdateColumn(_ context.Context, c domain.Column) error {
	f.columns[c.ID] = c
	return nil
}

// ListColumns lists columns.
func (f *fakeRepo) ListColumns(_ context.Context, projectID string, includeArchived bool) ([]domain.Column, error) {
	out := make([]domain.Column, 0, len(f.columns))
	for _, c := range f.columns {
		if c.ProjectID != projectID {
			continue
		}
		if !includeArchived && c.ArchivedAt != nil {
			continue
		}
		out = append(out, c)
	}
	return out, nil
}

// CreateActionItem creates actionItem.
func (f *fakeRepo) CreateActionItem(ctx context.Context, t domain.ActionItem) error {
	f.createActionItemActor, _ = MutationActorFromContext(ctx)
	f.tasks[t.ID] = t
	return nil
}

// UpdateActionItem updates state for the requested operation.
func (f *fakeRepo) UpdateActionItem(ctx context.Context, t domain.ActionItem) error {
	f.updateActionItemActor, _ = MutationActorFromContext(ctx)
	if _, ok := f.tasks[t.ID]; !ok {
		return ErrNotFound
	}
	f.tasks[t.ID] = t
	return nil
}

// GetActionItem returns actionItem.
func (f *fakeRepo) GetActionItem(_ context.Context, id string) (domain.ActionItem, error) {
	t, ok := f.tasks[id]
	if !ok {
		return domain.ActionItem{}, ErrNotFound
	}
	return t, nil
}

// ListActionItems lists tasks.
func (f *fakeRepo) ListActionItems(_ context.Context, projectID string, includeArchived bool) ([]domain.ActionItem, error) {
	out := make([]domain.ActionItem, 0, len(f.tasks))
	for _, t := range f.tasks {
		if t.ProjectID != projectID {
			continue
		}
		if !includeArchived && t.ArchivedAt != nil {
			continue
		}
		out = append(out, t)
	}
	return out, nil
}

// ListActionItemsByParent lists action items whose ParentID matches parentID
// within the supplied project, ordered deterministically by CreatedAt ASC,
// ID ASC. The empty parentID returns level-1 children (no parent). Mirrors
// the SQLite-side adapter contract used by the dotted-address resolver.
func (f *fakeRepo) ListActionItemsByParent(_ context.Context, projectID, parentID string) ([]domain.ActionItem, error) {
	out := make([]domain.ActionItem, 0, len(f.tasks))
	for _, t := range f.tasks {
		if t.ProjectID != projectID {
			continue
		}
		if t.ParentID != parentID {
			continue
		}
		out = append(out, t)
	}
	sort.Slice(out, func(i, j int) bool {
		if !out[i].CreatedAt.Equal(out[j].CreatedAt) {
			return out[i].CreatedAt.Before(out[j].CreatedAt)
		}
		return out[i].ID < out[j].ID
	})
	return out, nil
}

// FindActionItemByOwnerAndTitle returns the first action item in the project
// matching the supplied owner + title pair. Mirrors the SQLite-side adapter
// contract used by the auto-generator (droplet 3.20). Returns ErrNotFound
// when no row matches. Iteration is deterministic by id ASC so callers can
// assert exact behavior.
func (f *fakeRepo) FindActionItemByOwnerAndTitle(_ context.Context, projectID, owner, title string) (domain.ActionItem, error) {
	matches := make([]domain.ActionItem, 0)
	for _, t := range f.tasks {
		if t.ProjectID != projectID {
			continue
		}
		if t.Owner != owner {
			continue
		}
		if t.Title != title {
			continue
		}
		matches = append(matches, t)
	}
	if len(matches) == 0 {
		return domain.ActionItem{}, ErrNotFound
	}
	sort.Slice(matches, func(i, j int) bool {
		return matches[i].ID < matches[j].ID
	})
	return matches[0], nil
}

// ListActionItemsByDropNumber returns every action item in the supplied
// project whose DropNumber matches dropNumber, ordered deterministically by
// CreatedAt ASC, ID ASC. Mirrors the SQLite-side adapter contract used by
// the auto-generator (droplet 3.20). Includes archived rows; callers filter
// at the call site.
func (f *fakeRepo) ListActionItemsByDropNumber(_ context.Context, projectID string, dropNumber int) ([]domain.ActionItem, error) {
	out := make([]domain.ActionItem, 0, len(f.tasks))
	for _, t := range f.tasks {
		if t.ProjectID != projectID {
			continue
		}
		if t.DropNumber != dropNumber {
			continue
		}
		out = append(out, t)
	}
	sort.Slice(out, func(i, j int) bool {
		if !out[i].CreatedAt.Equal(out[j].CreatedAt) {
			return out[i].CreatedAt.Before(out[j].CreatedAt)
		}
		return out[i].ID < out[j].ID
	})
	return out, nil
}

// DeleteActionItem deletes actionItem.
func (f *fakeRepo) DeleteActionItem(_ context.Context, id string) error {
	actionItem, ok := f.tasks[id]
	if !ok {
		return ErrNotFound
	}
	delete(f.tasks, id)
	targetKey := actionItem.ProjectID + "|" + string(snapshotCommentTargetTypeForActionItem(actionItem)) + "|" + actionItem.ID
	delete(f.comments, targetKey)
	return nil
}

// CreateComment creates comment.
func (f *fakeRepo) CreateComment(ctx context.Context, comment domain.Comment) error {
	f.createCommentActor, _ = MutationActorFromContext(ctx)
	key := comment.ProjectID + "|" + string(comment.TargetType) + "|" + comment.TargetID
	f.comments[key] = append(f.comments[key], comment)
	return nil
}

// ListCommentsByTarget lists comments for a target.
func (f *fakeRepo) ListCommentsByTarget(_ context.Context, target domain.CommentTarget) ([]domain.Comment, error) {
	key := target.ProjectID + "|" + string(target.TargetType) + "|" + target.TargetID
	return append([]domain.Comment(nil), f.comments[key]...), nil
}

// ListCommentTargets lists distinct comment targets for one project.
func (f *fakeRepo) ListCommentTargets(_ context.Context, projectID string) ([]domain.CommentTarget, error) {
	projectID = strings.TrimSpace(projectID)
	out := make([]domain.CommentTarget, 0, len(f.comments))
	for key, comments := range f.comments {
		if len(comments) == 0 {
			continue
		}
		parts := strings.Split(key, "|")
		if len(parts) != 3 || parts[0] != projectID {
			continue
		}
		target, err := domain.NormalizeCommentTarget(domain.CommentTarget{
			ProjectID:  parts[0],
			TargetType: domain.CommentTargetType(parts[1]),
			TargetID:   parts[2],
		})
		if err != nil {
			return nil, err
		}
		out = append(out, target)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].TargetType != out[j].TargetType {
			return out[i].TargetType < out[j].TargetType
		}
		return out[i].TargetID < out[j].TargetID
	})
	return out, nil
}

// CreateAttentionItem creates one attention item row.
func (f *fakeRepo) CreateAttentionItem(_ context.Context, item domain.AttentionItem) error {
	f.attentionItems[item.ID] = item
	return nil
}

// UpsertAttentionItem creates or replaces one attention item row.
func (f *fakeRepo) UpsertAttentionItem(_ context.Context, item domain.AttentionItem) error {
	f.attentionItems[item.ID] = item
	return nil
}

// GetAttentionItem returns one attention item row by id.
func (f *fakeRepo) GetAttentionItem(_ context.Context, attentionID string) (domain.AttentionItem, error) {
	item, ok := f.attentionItems[attentionID]
	if !ok {
		return domain.AttentionItem{}, ErrNotFound
	}
	return item, nil
}

// ListAttentionItems lists scoped attention items in deterministic order.
func (f *fakeRepo) ListAttentionItems(_ context.Context, filter domain.AttentionListFilter) ([]domain.AttentionItem, error) {
	filter, err := domain.NormalizeAttentionListFilter(filter)
	if err != nil {
		return nil, err
	}

	matchesState := func(item domain.AttentionItem) bool {
		if len(filter.States) == 0 {
			return true
		}
		for _, state := range filter.States {
			if item.State == state {
				return true
			}
		}
		return false
	}
	matchesKind := func(item domain.AttentionItem) bool {
		if len(filter.Kinds) == 0 {
			return true
		}
		for _, kind := range filter.Kinds {
			if item.Kind == kind {
				return true
			}
		}
		return false
	}

	out := make([]domain.AttentionItem, 0)
	for _, item := range f.attentionItems {
		if item.ProjectID != filter.ProjectID {
			continue
		}
		if filter.ScopeType != "" && item.ScopeType != filter.ScopeType {
			continue
		}
		if filter.ScopeType != "" && item.ScopeID != filter.ScopeID {
			continue
		}
		if filter.TargetRole != "" && item.TargetRole != filter.TargetRole {
			continue
		}
		if filter.UnresolvedOnly && !item.IsUnresolved() {
			continue
		}
		if filter.RequiresUserAction != nil && item.RequiresUserAction != *filter.RequiresUserAction {
			continue
		}
		if !matchesState(item) || !matchesKind(item) {
			continue
		}
		out = append(out, item)
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].CreatedAt.Equal(out[j].CreatedAt) {
			return out[i].ID > out[j].ID
		}
		return out[i].CreatedAt.After(out[j].CreatedAt)
	})
	if filter.Limit > 0 && len(out) > filter.Limit {
		out = out[:filter.Limit]
	}
	return out, nil
}

// ResolveAttentionItem resolves one attention item row and returns the updated value.
func (f *fakeRepo) ResolveAttentionItem(_ context.Context, attentionID string, resolvedBy string, resolvedByType domain.ActorType, resolvedAt time.Time) (domain.AttentionItem, error) {
	item, ok := f.attentionItems[attentionID]
	if !ok {
		return domain.AttentionItem{}, ErrNotFound
	}
	if err := item.Resolve(resolvedBy, resolvedByType, resolvedAt); err != nil {
		return domain.AttentionItem{}, err
	}
	f.attentionItems[attentionID] = item
	return item, nil
}

// CreateAuthRequest stores one auth request row.
func (f *fakeRepo) CreateAuthRequest(_ context.Context, request domain.AuthRequest) error {
	f.authRequests[request.ID] = request
	return nil
}

// GetAuthRequest returns one auth request by id.
func (f *fakeRepo) GetAuthRequest(_ context.Context, requestID string) (domain.AuthRequest, error) {
	request, ok := f.authRequests[requestID]
	if !ok {
		return domain.AuthRequest{}, ErrNotFound
	}
	return request, nil
}

// ListAuthRequests lists auth requests with deterministic filtering.
func (f *fakeRepo) ListAuthRequests(_ context.Context, filter domain.AuthRequestListFilter) ([]domain.AuthRequest, error) {
	filter, err := domain.NormalizeAuthRequestListFilter(filter)
	if err != nil {
		return nil, err
	}
	out := make([]domain.AuthRequest, 0, len(f.authRequests))
	for _, request := range f.authRequests {
		if filter.ProjectID != "" && request.ProjectID != filter.ProjectID {
			continue
		}
		if filter.State != "" && domain.NormalizeAuthRequestState(request.State) != filter.State {
			continue
		}
		out = append(out, request)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].CreatedAt.Equal(out[j].CreatedAt) {
			return out[i].ID > out[j].ID
		}
		return out[i].CreatedAt.After(out[j].CreatedAt)
	})
	if filter.Limit > 0 && len(out) > filter.Limit {
		out = out[:filter.Limit]
	}
	return out, nil
}

// UpdateAuthRequest stores one auth request update.
func (f *fakeRepo) UpdateAuthRequest(_ context.Context, request domain.AuthRequest) error {
	if _, ok := f.authRequests[request.ID]; !ok {
		return ErrNotFound
	}
	f.authRequests[request.ID] = request
	return nil
}

// CreateHandoff stores one durable handoff row.
func (f *fakeRepo) CreateHandoff(_ context.Context, handoff domain.Handoff) error {
	f.handoffs[handoff.ID] = handoff
	return nil
}

// GetHandoff returns one durable handoff row by id.
func (f *fakeRepo) GetHandoff(_ context.Context, handoffID string) (domain.Handoff, error) {
	handoff, ok := f.handoffs[handoffID]
	if !ok {
		return domain.Handoff{}, ErrNotFound
	}
	return handoff, nil
}

// ListHandoffs lists durable handoffs with deterministic filtering and ordering.
func (f *fakeRepo) ListHandoffs(_ context.Context, filter domain.HandoffListFilter) ([]domain.Handoff, error) {
	filter, err := domain.NormalizeHandoffListFilter(filter)
	if err != nil {
		return nil, err
	}
	matchesStatus := func(handoff domain.Handoff) bool {
		if len(filter.Statuses) == 0 {
			return true
		}
		for _, status := range filter.Statuses {
			if handoff.Status == status {
				return true
			}
		}
		return false
	}

	out := make([]domain.Handoff, 0, len(f.handoffs))
	for _, handoff := range f.handoffs {
		if handoff.ProjectID != filter.ProjectID {
			continue
		}
		if filter.BranchID != "" && handoff.BranchID != filter.BranchID {
			continue
		}
		if filter.ScopeType != "" && handoff.ScopeType != filter.ScopeType {
			continue
		}
		if filter.ScopeID != "" && handoff.ScopeID != filter.ScopeID {
			continue
		}
		if !matchesStatus(handoff) {
			continue
		}
		out = append(out, handoff)
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].UpdatedAt.Equal(out[j].UpdatedAt) {
			return out[i].ID > out[j].ID
		}
		return out[i].UpdatedAt.After(out[j].UpdatedAt)
	})
	if filter.Limit > 0 && len(out) > filter.Limit {
		out = out[:filter.Limit]
	}
	return out, nil
}

// UpdateHandoff stores one durable handoff update.
func (f *fakeRepo) UpdateHandoff(_ context.Context, handoff domain.Handoff) error {
	if _, ok := f.handoffs[handoff.ID]; !ok {
		return ErrNotFound
	}
	f.handoffs[handoff.ID] = handoff
	return nil
}

// ListProjectChangeEvents lists change events.
func (f *fakeRepo) ListProjectChangeEvents(_ context.Context, projectID string, limit int) ([]domain.ChangeEvent, error) {
	events := append([]domain.ChangeEvent(nil), f.changeEvents[projectID]...)
	if limit <= 0 || limit >= len(events) {
		return events, nil
	}
	return events[:limit], nil
}

// CreateCapabilityLease creates one capability lease row.
func (f *fakeRepo) CreateCapabilityLease(_ context.Context, lease domain.CapabilityLease) error {
	f.capabilityLeases[lease.InstanceID] = lease
	return nil
}

// UpdateCapabilityLease updates one capability lease row.
func (f *fakeRepo) UpdateCapabilityLease(_ context.Context, lease domain.CapabilityLease) error {
	if _, ok := f.capabilityLeases[lease.InstanceID]; !ok {
		return ErrNotFound
	}
	f.capabilityLeases[lease.InstanceID] = lease
	return nil
}

// GetCapabilityLease returns one capability lease row.
func (f *fakeRepo) GetCapabilityLease(_ context.Context, instanceID string) (domain.CapabilityLease, error) {
	lease, ok := f.capabilityLeases[instanceID]
	if !ok {
		return domain.CapabilityLease{}, ErrNotFound
	}
	return lease, nil
}

// ListCapabilityLeasesByScope lists scope-matching capability leases.
func (f *fakeRepo) ListCapabilityLeasesByScope(_ context.Context, projectID string, scopeType domain.CapabilityScopeType, scopeID string) ([]domain.CapabilityLease, error) {
	out := make([]domain.CapabilityLease, 0)
	for _, lease := range f.capabilityLeases {
		if lease.ProjectID != projectID {
			continue
		}
		if lease.ScopeType != scopeType {
			continue
		}
		if strings.TrimSpace(scopeID) != "" && lease.ScopeID != strings.TrimSpace(scopeID) {
			continue
		}
		out = append(out, lease)
	}
	return out, nil
}

// RevokeCapabilityLeasesByScope revokes all scope-matching leases.
func (f *fakeRepo) RevokeCapabilityLeasesByScope(_ context.Context, projectID string, scopeType domain.CapabilityScopeType, scopeID string, revokedAt time.Time, reason string) error {
	for instanceID, lease := range f.capabilityLeases {
		if lease.ProjectID != projectID {
			continue
		}
		if lease.ScopeType != scopeType {
			continue
		}
		if strings.TrimSpace(scopeID) != "" && lease.ScopeID != strings.TrimSpace(scopeID) {
			continue
		}
		lease.Revoke(reason, revokedAt)
		f.capabilityLeases[instanceID] = lease
	}
	return nil
}

// TestEnsureDefaultProject verifies behavior for the covered scenario.
func TestEnsureDefaultProject(t *testing.T) {
	repo := newFakeRepo()
	idCounter := 0
	svc := NewService(repo, func() string {
		idCounter++
		return "id-" + string(rune('0'+idCounter))
	}, func() time.Time {
		return time.Date(2026, 2, 21, 12, 0, 0, 0, time.UTC)
	}, ServiceConfig{DefaultDeleteMode: DeleteModeArchive})

	project, err := svc.EnsureDefaultProject(context.Background())
	if err != nil {
		t.Fatalf("EnsureDefaultProject() error = %v", err)
	}
	if project.Name != "Inbox" {
		t.Fatalf("unexpected project name %q", project.Name)
	}
	columns, err := svc.ListColumns(context.Background(), project.ID, false)
	if err != nil {
		t.Fatalf("ListColumns() error = %v", err)
	}
	if len(columns) != 3 {
		t.Fatalf("expected 3 default columns, got %d", len(columns))
	}
}

// TestCreateActionItemMoveSearchAndDeleteModes verifies behavior for the covered scenario.
func TestCreateActionItemMoveSearchAndDeleteModes(t *testing.T) {
	repo := newFakeRepo()
	now := time.Date(2026, 2, 21, 12, 0, 0, 0, time.UTC)
	ids := []string{"p1", "c1", "c2", "t1"}
	idx := 0
	svc := NewService(repo, func() string {
		id := ids[idx]
		idx++
		return id
	}, func() time.Time {
		return now
	}, ServiceConfig{DefaultDeleteMode: DeleteModeArchive})

	project, err := svc.CreateProject(context.Background(), "Project", "")
	if err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}
	col1, err := svc.CreateColumn(context.Background(), project.ID, "To Do", 0, 0)
	if err != nil {
		t.Fatalf("CreateColumn() error = %v", err)
	}
	col2, err := svc.CreateColumn(context.Background(), project.ID, "Done", 1, 0)
	if err != nil {
		t.Fatalf("CreateColumn() error = %v", err)
	}

	actionItem, err := svc.CreateActionItem(context.Background(), CreateActionItemInput{
		ProjectID:      project.ID,
		ColumnID:       col1.ID,
		Title:          "Fix parser",
		Description:    "Add tests for parser",
		Priority:       domain.PriorityHigh,
		Labels:         []string{"parser"},
		StructuralType: domain.StructuralTypeDroplet,
	})
	if err != nil {
		t.Fatalf("CreateActionItem() error = %v", err)
	}
	if actionItem.Position != 0 {
		t.Fatalf("unexpected actionItem position %d", actionItem.Position)
	}

	actionItem, err = svc.MoveActionItem(context.Background(), actionItem.ID, col2.ID, 1)
	if err != nil {
		t.Fatalf("MoveActionItem() error = %v", err)
	}
	if actionItem.ColumnID != col2.ID || actionItem.Position != 1 {
		t.Fatalf("unexpected moved actionItem %#v", actionItem)
	}

	search, err := svc.SearchActionItemMatches(context.Background(), SearchActionItemsFilter{
		ProjectID: project.ID,
		Query:     "parser",
	})
	if err != nil {
		t.Fatalf("SearchActionItemMatches() error = %v", err)
	}
	if len(search) != 1 {
		t.Fatalf("expected 1 search result, got %d", len(search))
	}

	if err := svc.DeleteActionItem(context.Background(), actionItem.ID, ""); err != nil {
		t.Fatalf("DeleteActionItem(archive default) error = %v", err)
	}
	tAfterArchive, err := repo.GetActionItem(context.Background(), actionItem.ID)
	if err != nil {
		t.Fatalf("GetActionItem() error = %v", err)
	}
	if tAfterArchive.ArchivedAt == nil {
		t.Fatal("expected actionItem to be archived")
	}

	restored, err := svc.RestoreActionItem(context.Background(), actionItem.ID)
	if err != nil {
		t.Fatalf("RestoreActionItem() error = %v", err)
	}
	if restored.ArchivedAt != nil {
		t.Fatal("expected actionItem to be restored")
	}

	if err := svc.DeleteActionItem(context.Background(), actionItem.ID, DeleteModeHard); err != nil {
		t.Fatalf("DeleteActionItem(hard) error = %v", err)
	}
	if _, err := repo.GetActionItem(context.Background(), actionItem.ID); err != ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

// TestRestoreActionItemUsesRequestActorContext verifies restore guard actor type comes from request actor context.
func TestRestoreActionItemUsesRequestActorContext(t *testing.T) {
	repo := newFakeRepo()
	ids := []string{"p1", "c1", "t1"}
	idx := 0
	now := time.Date(2026, 2, 24, 10, 0, 0, 0, time.UTC)
	svc := NewService(repo, func() string {
		id := ids[idx]
		idx++
		return id
	}, func() time.Time {
		return now
	}, ServiceConfig{DefaultDeleteMode: DeleteModeArchive})

	project, err := svc.CreateProject(context.Background(), "Restore Guard", "")
	if err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}
	column, err := svc.CreateColumn(context.Background(), project.ID, "To Do", 0, 0)
	if err != nil {
		t.Fatalf("CreateColumn() error = %v", err)
	}
	actionItem, err := svc.CreateActionItem(context.Background(), CreateActionItemInput{
		ProjectID:      project.ID,
		ColumnID:       column.ID,
		Title:          "archived actionItem",
		Priority:       domain.PriorityMedium,
		UpdatedByType:  domain.ActorTypeUser,
		StructuralType: domain.StructuralTypeDroplet,
	})
	if err != nil {
		t.Fatalf("CreateActionItem() error = %v", err)
	}
	if err := svc.DeleteActionItem(context.Background(), actionItem.ID, DeleteModeArchive); err != nil {
		t.Fatalf("DeleteActionItem(archive) error = %v", err)
	}

	archivedActionItem, err := repo.GetActionItem(context.Background(), actionItem.ID)
	if err != nil {
		t.Fatalf("GetActionItem(archived) error = %v", err)
	}
	// Simulate prior archival attribution from an agent mutation.
	archivedActionItem.UpdatedByActor = "agent-1"
	archivedActionItem.UpdatedByType = domain.ActorTypeAgent
	repo.tasks[actionItem.ID] = archivedActionItem

	ctx := WithMutationActor(context.Background(), MutationActor{
		ActorID:   "user-1",
		ActorType: domain.ActorTypeUser,
	})
	restored, err := svc.RestoreActionItem(ctx, actionItem.ID)
	if err != nil {
		t.Fatalf("RestoreActionItem() error = %v", err)
	}
	if restored.ArchivedAt != nil {
		t.Fatal("expected restore to clear archived_at")
	}
	if restored.UpdatedByActor != "user-1" {
		t.Fatalf("restored updated_by_actor = %q, want user-1", restored.UpdatedByActor)
	}
	if restored.UpdatedByType != domain.ActorTypeUser {
		t.Fatalf("restored updated_by_type = %q, want %q", restored.UpdatedByType, domain.ActorTypeUser)
	}
}

// TestRestoreActionItemRequiresLeaseForNonUserCaller verifies non-user restore calls fail closed without a lease tuple.
func TestRestoreActionItemRequiresLeaseForNonUserCaller(t *testing.T) {
	repo := newFakeRepo()
	ids := []string{"p1", "c1", "t1"}
	idx := 0
	now := time.Date(2026, 2, 24, 10, 0, 0, 0, time.UTC)
	svc := NewService(repo, func() string {
		id := ids[idx]
		idx++
		return id
	}, func() time.Time {
		return now
	}, ServiceConfig{DefaultDeleteMode: DeleteModeArchive})

	project, err := svc.CreateProject(context.Background(), "Restore Guard", "")
	if err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}
	column, err := svc.CreateColumn(context.Background(), project.ID, "To Do", 0, 0)
	if err != nil {
		t.Fatalf("CreateColumn() error = %v", err)
	}
	actionItem, err := svc.CreateActionItem(context.Background(), CreateActionItemInput{
		ProjectID:      project.ID,
		ColumnID:       column.ID,
		Title:          "archived actionItem",
		Priority:       domain.PriorityMedium,
		StructuralType: domain.StructuralTypeDroplet,
	})
	if err != nil {
		t.Fatalf("CreateActionItem() error = %v", err)
	}
	if err := svc.DeleteActionItem(context.Background(), actionItem.ID, DeleteModeArchive); err != nil {
		t.Fatalf("DeleteActionItem(archive) error = %v", err)
	}

	ctx := WithMutationActor(context.Background(), MutationActor{
		ActorID:   "agent-1",
		ActorType: domain.ActorTypeAgent,
	})
	_, err = svc.RestoreActionItem(ctx, actionItem.ID)
	if !errors.Is(err, domain.ErrMutationLeaseRequired) {
		t.Fatalf("RestoreActionItem() error = %v, want ErrMutationLeaseRequired", err)
	}
}

// TestDeleteActionItemModeValidation verifies behavior for the covered scenario.
func TestDeleteActionItemModeValidation(t *testing.T) {
	repo := newFakeRepo()
	svc := NewService(repo, func() string { return "x" }, time.Now, ServiceConfig{})
	err := svc.DeleteActionItem(context.Background(), "actionItem-1", DeleteMode("invalid"))
	if err != ErrInvalidDeleteMode {
		t.Fatalf("expected ErrInvalidDeleteMode, got %v", err)
	}
}

// TestRenameActionItem verifies behavior for the covered scenario.
func TestRenameActionItem(t *testing.T) {
	repo := newFakeRepo()
	now := time.Date(2026, 2, 21, 12, 0, 0, 0, time.UTC)
	actionItem, _ := domain.NewActionItemForTest(domain.ActionItemInput{
		Kind:      domain.KindPlan,
		ID:        "t1",
		ProjectID: "p1",
		ColumnID:  "c1",
		Position:  0,
		Title:     "old",
		Priority:  domain.PriorityLow,
	}, now)
	repo.tasks[actionItem.ID] = actionItem

	svc := NewService(repo, nil, func() time.Time { return now.Add(time.Minute) }, ServiceConfig{})
	updated, err := svc.RenameActionItem(context.Background(), actionItem.ID, "new title")
	if err != nil {
		t.Fatalf("RenameActionItem() error = %v", err)
	}
	if updated.Title != "new title" {
		t.Fatalf("unexpected title %q", updated.Title)
	}
}

// TestUpdateActionItem verifies behavior for the covered scenario.
func TestUpdateActionItem(t *testing.T) {
	repo := newFakeRepo()
	now := time.Date(2026, 2, 21, 12, 0, 0, 0, time.UTC)
	actionItem, _ := domain.NewActionItemForTest(domain.ActionItemInput{
		Kind:      domain.KindPlan,
		ID:        "t1",
		ProjectID: "p1",
		ColumnID:  "c1",
		Position:  0,
		Title:     "old",
		Priority:  domain.PriorityLow,
	}, now)
	repo.tasks[actionItem.ID] = actionItem

	svc := NewService(repo, nil, func() time.Time { return now.Add(time.Minute) }, ServiceConfig{})
	due := now.Add(24 * time.Hour)
	updated, err := svc.UpdateActionItem(context.Background(), UpdateActionItemInput{
		ActionItemID: actionItem.ID,
		Title:        ptrTo("new title"),
		Description:  ptrTo("details"),
		Priority:     ptrTo(domain.PriorityHigh),
		DueAt:        ptrTo(&due),
		Labels:       ptrTo([]string{"frontend", "backend"}),
	})
	if err != nil {
		t.Fatalf("UpdateActionItem() error = %v", err)
	}
	if updated.Title != "new title" || updated.Description != "details" || updated.Priority != domain.PriorityHigh {
		t.Fatalf("unexpected updated actionItem %#v", updated)
	}
	if updated.DueAt == nil || len(updated.Labels) != 2 {
		t.Fatalf("expected due date and labels, got %#v", updated)
	}
}

// TestUpdateActionItemAppliesMutationActorContext verifies context-supplied actor attribution is persisted on updates.
func TestUpdateActionItemAppliesMutationActorContext(t *testing.T) {
	repo := newFakeRepo()
	now := time.Date(2026, 2, 21, 12, 0, 0, 0, time.UTC)
	actionItem, _ := domain.NewActionItemForTest(domain.ActionItemInput{
		Kind:           domain.KindPlan,
		ID:             "t1",
		ProjectID:      "p1",
		ColumnID:       "c1",
		Position:       0,
		Title:          "old",
		Priority:       domain.PriorityLow,
		CreatedByActor: "EVAN",
		UpdatedByActor: "EVAN",
		UpdatedByType:  domain.ActorTypeUser,
	}, now)
	repo.tasks[actionItem.ID] = actionItem

	svc := NewService(repo, nil, func() time.Time { return now.Add(time.Minute) }, ServiceConfig{})
	ctx := WithMutationActor(context.Background(), MutationActor{
		ActorID:   "user-context-1",
		ActorType: domain.ActorTypeUser,
	})
	updated, err := svc.UpdateActionItem(ctx, UpdateActionItemInput{
		ActionItemID: actionItem.ID,
		Title:        ptrTo("new title"),
	})
	if err != nil {
		t.Fatalf("UpdateActionItem() error = %v", err)
	}
	if updated.UpdatedByActor != "user-context-1" {
		t.Fatalf("updated actor id = %q, want user-context-1", updated.UpdatedByActor)
	}
	if updated.UpdatedByType != domain.ActorTypeUser {
		t.Fatalf("updated actor type = %q, want %q", updated.UpdatedByType, domain.ActorTypeUser)
	}
}

// TestCreateActionItemCarriesHumanActorName verifies actionItem mutations pass display attribution to the repo boundary.
func TestCreateActionItemCarriesHumanActorName(t *testing.T) {
	repo := newFakeRepo()
	now := time.Date(2026, 2, 26, 10, 0, 0, 0, time.UTC)
	project, _ := domain.NewProjectFromInput(domain.ProjectInput{ID: "p1", Name: "Inbox"}, now)
	repo.projects[project.ID] = project
	column, _ := domain.NewColumn("c1", project.ID, "To Do", 0, 0, now)
	repo.columns[column.ID] = column
	kind, err := domain.NewKindDefinition(domain.KindDefinitionInput{
		ID:        domain.KindID(domain.KindPlan),
		AppliesTo: []domain.KindAppliesTo{domain.KindAppliesToPlan},
	}, now)
	if err != nil {
		t.Fatalf("NewKindDefinition() error = %v", err)
	}
	repo.kindDefs[kind.ID] = kind

	svc := NewService(repo, func() string { return "t1" }, func() time.Time { return now }, ServiceConfig{})
	created, err := svc.CreateActionItem(context.Background(), CreateActionItemInput{
		ProjectID:      project.ID,
		ColumnID:       column.ID,
		Title:          "Ship attribution",
		Priority:       domain.PriorityMedium,
		CreatedByActor: "user-1",
		CreatedByName:  "Evan Schultz",
		UpdatedByType:  domain.ActorTypeUser,
		StructuralType: domain.StructuralTypeDroplet,
	})
	if err != nil {
		t.Fatalf("CreateActionItem() error = %v", err)
	}
	if created.CreatedByActor != "user-1" || created.UpdatedByActor != "user-1" {
		t.Fatalf("expected actionItem attribution to use actor id user-1, got %#v", created)
	}
	if repo.createActionItemActor.ActorID != "user-1" {
		t.Fatalf("create actionItem actor id = %q, want user-1", repo.createActionItemActor.ActorID)
	}
	if repo.createActionItemActor.ActorName != "Evan Schultz" {
		t.Fatalf("create actionItem actor name = %q, want Evan Schultz", repo.createActionItemActor.ActorName)
	}
	if repo.createActionItemActor.ActorType != domain.ActorTypeUser {
		t.Fatalf("create actionItem actor type = %q, want %q", repo.createActionItemActor.ActorType, domain.ActorTypeUser)
	}
}

// TestUpdateActionItemCarriesExplicitActorName verifies explicit update attribution is propagated without a pre-seeded context.
func TestUpdateActionItemCarriesExplicitActorName(t *testing.T) {
	repo := newFakeRepo()
	now := time.Date(2026, 2, 26, 10, 30, 0, 0, time.UTC)
	actionItem, _ := domain.NewActionItemForTest(domain.ActionItemInput{
		Kind:      domain.KindPlan,
		ID:        "t1",
		ProjectID: "p1",
		ColumnID:  "c1",
		Position:  0,
		Title:     "old",
		Priority:  domain.PriorityLow,
	}, now)
	repo.tasks[actionItem.ID] = actionItem

	svc := NewService(repo, nil, func() time.Time { return now.Add(time.Minute) }, ServiceConfig{})
	updated, err := svc.UpdateActionItem(context.Background(), UpdateActionItemInput{
		ActionItemID:  actionItem.ID,
		Title:         ptrTo("new title"),
		UpdatedBy:     "user-2",
		UpdatedByName: "Evan Schultz",
		UpdatedType:   domain.ActorTypeUser,
	})
	if err != nil {
		t.Fatalf("UpdateActionItem() error = %v", err)
	}
	if updated.UpdatedByActor != "user-2" {
		t.Fatalf("updated actor id = %q, want user-2", updated.UpdatedByActor)
	}
	if updated.UpdatedByType != domain.ActorTypeUser {
		t.Fatalf("updated actor type = %q, want %q", updated.UpdatedByType, domain.ActorTypeUser)
	}
	if repo.updateActionItemActor.ActorID != "user-2" {
		t.Fatalf("update actionItem actor id = %q, want user-2", repo.updateActionItemActor.ActorID)
	}
	if repo.updateActionItemActor.ActorName != "Evan Schultz" {
		t.Fatalf("update actionItem actor name = %q, want Evan Schultz", repo.updateActionItemActor.ActorName)
	}
	if repo.updateActionItemActor.ActorType != domain.ActorTypeUser {
		t.Fatalf("update actionItem actor type = %q, want %q", repo.updateActionItemActor.ActorType, domain.ActorTypeUser)
	}
}

// TestUpdateActionItemPreservesPriorityWhenOmitted verifies update behavior when priority is omitted.
func TestUpdateActionItemPreservesPriorityWhenOmitted(t *testing.T) {
	repo := newFakeRepo()
	now := time.Date(2026, 2, 21, 12, 0, 0, 0, time.UTC)
	actionItem, _ := domain.NewActionItemForTest(domain.ActionItemInput{
		Kind:      domain.KindPlan,
		ID:        "t1",
		ProjectID: "p1",
		ColumnID:  "c1",
		Position:  0,
		Title:     "old",
		Priority:  domain.PriorityMedium,
	}, now)
	repo.tasks[actionItem.ID] = actionItem

	svc := NewService(repo, nil, func() time.Time { return now.Add(time.Minute) }, ServiceConfig{})
	updated, err := svc.UpdateActionItem(context.Background(), UpdateActionItemInput{
		ActionItemID: actionItem.ID,
		Title:        ptrTo("new title"),
	})
	if err != nil {
		t.Fatalf("UpdateActionItem(title-only) error = %v", err)
	}
	if updated.Priority != domain.PriorityMedium {
		t.Fatalf("priority = %q, want %q", updated.Priority, domain.PriorityMedium)
	}
}

// TestUpdateActionItemPartialPATCHSemantics covers Drop 4c.5 droplet A.1's
// pointer-sentinel PATCH contract on Service.UpdateActionItem. Each row
// seeds a stored action item, then issues a partial-update with the named
// pointer fields and asserts the post-update state matches the
// preserve-vs-apply-vs-clear contract:
//
//   - nil input pointer → preserve the stored value;
//   - non-nil pointer → apply the dereferenced value (empty deref clears,
//     except Title where empty surfaces ErrInvalidTitle);
//   - DueAt uses **time.Time so the outer-pointer-nil / outer-non-nil-
//     inner-nil / outer-non-nil-inner-non-nil triplet covers
//     preserve / clear / set.
//
// The 9-row table mirrors THEME_A_PLAN.md § A.1's acceptance scenarios.
func TestUpdateActionItemPartialPATCHSemantics(t *testing.T) {
	now := time.Date(2026, 5, 5, 12, 0, 0, 0, time.UTC)
	preStoredDue := now.Add(48 * time.Hour)

	// seed builds a fresh repo + service + stored action item with the
	// canonical pre-update state (title="old title", description="old desc",
	// priority=high, due_at=preStoredDue, labels=[a b]). Each subtest
	// receives its own seeded fixture so writes don't leak across cases.
	seed := func(t *testing.T) (*Service, domain.ActionItem) {
		t.Helper()
		repo := newFakeRepo()
		actionItem, err := domain.NewActionItemForTest(domain.ActionItemInput{
			Kind:        domain.KindPlan,
			ID:          "t1",
			ProjectID:   "p1",
			ColumnID:    "c1",
			Position:    0,
			Title:       "old title",
			Description: "old desc",
			Priority:    domain.PriorityHigh,
			DueAt:       &preStoredDue,
			Labels:      []string{"a", "b"},
		}, now)
		if err != nil {
			t.Fatalf("seed action item: %v", err)
		}
		repo.tasks[actionItem.ID] = actionItem
		svc := NewService(repo, nil, func() time.Time { return now.Add(time.Minute) }, ServiceConfig{})
		return svc, actionItem
	}

	cases := []struct {
		name          string
		buildInput    func(actionItemID string) UpdateActionItemInput
		expectErr     error
		expectTitle   string
		expectDesc    string
		expectPrio    domain.Priority
		expectDueNil  bool
		expectDueTime time.Time
		expectLabels  []string
	}{
		{
			name: "description nil preserves",
			buildInput: func(id string) UpdateActionItemInput {
				return UpdateActionItemInput{
					ActionItemID: id,
					Title:        ptrTo("new title"),
				}
			},
			expectTitle:   "new title",
			expectDesc:    "old desc",
			expectPrio:    domain.PriorityHigh,
			expectDueTime: preStoredDue,
			expectLabels:  []string{"a", "b"},
		},
		{
			name: "description empty pointer clears",
			buildInput: func(id string) UpdateActionItemInput {
				return UpdateActionItemInput{
					ActionItemID: id,
					Title:        ptrTo("new title"),
					Description:  ptrTo(""),
				}
			},
			expectTitle:   "new title",
			expectDesc:    "",
			expectPrio:    domain.PriorityHigh,
			expectDueTime: preStoredDue,
			expectLabels:  []string{"a", "b"},
		},
		{
			name: "description non-empty replaces",
			buildInput: func(id string) UpdateActionItemInput {
				return UpdateActionItemInput{
					ActionItemID: id,
					Title:        ptrTo("new title"),
					Description:  ptrTo("fresh"),
				}
			},
			expectTitle:   "new title",
			expectDesc:    "fresh",
			expectPrio:    domain.PriorityHigh,
			expectDueTime: preStoredDue,
			expectLabels:  []string{"a", "b"},
		},
		{
			name: "title nil preserves",
			buildInput: func(id string) UpdateActionItemInput {
				return UpdateActionItemInput{
					ActionItemID: id,
					Description:  ptrTo("new desc"),
				}
			},
			expectTitle:   "old title",
			expectDesc:    "new desc",
			expectPrio:    domain.PriorityHigh,
			expectDueTime: preStoredDue,
			expectLabels:  []string{"a", "b"},
		},
		{
			name: "title empty pointer rejected",
			buildInput: func(id string) UpdateActionItemInput {
				return UpdateActionItemInput{
					ActionItemID: id,
					Title:        ptrTo(""),
					Description:  ptrTo("new desc"),
				}
			},
			expectErr: domain.ErrInvalidTitle,
		},
		{
			name: "labels nil preserves",
			buildInput: func(id string) UpdateActionItemInput {
				return UpdateActionItemInput{
					ActionItemID: id,
					Title:        ptrTo("new title"),
				}
			},
			expectTitle:   "new title",
			expectDesc:    "old desc",
			expectPrio:    domain.PriorityHigh,
			expectDueTime: preStoredDue,
			expectLabels:  []string{"a", "b"},
		},
		{
			name: "labels empty pointer clears",
			buildInput: func(id string) UpdateActionItemInput {
				return UpdateActionItemInput{
					ActionItemID: id,
					Title:        ptrTo("new title"),
					Labels:       ptrTo([]string{}),
				}
			},
			expectTitle:   "new title",
			expectDesc:    "old desc",
			expectPrio:    domain.PriorityHigh,
			expectDueTime: preStoredDue,
			expectLabels:  []string{},
		},
		{
			name: "priority nil preserves",
			buildInput: func(id string) UpdateActionItemInput {
				return UpdateActionItemInput{
					ActionItemID: id,
					Title:        ptrTo("new title"),
				}
			},
			expectTitle:   "new title",
			expectDesc:    "old desc",
			expectPrio:    domain.PriorityHigh,
			expectDueTime: preStoredDue,
			expectLabels:  []string{"a", "b"},
		},
		{
			name: "due_at nil preserves",
			buildInput: func(id string) UpdateActionItemInput {
				return UpdateActionItemInput{
					ActionItemID: id,
					Title:        ptrTo("new title"),
				}
			},
			expectTitle:   "new title",
			expectDesc:    "old desc",
			expectPrio:    domain.PriorityHigh,
			expectDueTime: preStoredDue,
			expectLabels:  []string{"a", "b"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc, actionItem := seed(t)
			updated, err := svc.UpdateActionItem(context.Background(), tc.buildInput(actionItem.ID))
			if tc.expectErr != nil {
				if !errors.Is(err, tc.expectErr) {
					t.Fatalf("UpdateActionItem() error = %v, want %v", err, tc.expectErr)
				}
				// On error, the stored item must remain unchanged.
				stored, getErr := svc.GetActionItem(context.Background(), actionItem.ID)
				if getErr != nil {
					t.Fatalf("GetActionItem(after-error) = %v", getErr)
				}
				if stored.Title != "old title" || stored.Description != "old desc" {
					t.Fatalf("rejected update mutated stored item: title=%q desc=%q", stored.Title, stored.Description)
				}
				return
			}
			if err != nil {
				t.Fatalf("UpdateActionItem() unexpected error = %v", err)
			}
			if updated.Title != tc.expectTitle {
				t.Fatalf("Title = %q, want %q", updated.Title, tc.expectTitle)
			}
			if updated.Description != tc.expectDesc {
				t.Fatalf("Description = %q, want %q", updated.Description, tc.expectDesc)
			}
			if updated.Priority != tc.expectPrio {
				t.Fatalf("Priority = %q, want %q", updated.Priority, tc.expectPrio)
			}
			if tc.expectDueNil {
				if updated.DueAt != nil {
					t.Fatalf("DueAt = %v, want nil", updated.DueAt)
				}
			} else {
				if updated.DueAt == nil {
					t.Fatalf("DueAt = nil, want %v", tc.expectDueTime)
				}
				if !updated.DueAt.Equal(tc.expectDueTime) {
					t.Fatalf("DueAt = %v, want %v", *updated.DueAt, tc.expectDueTime)
				}
			}
			gotLabels := updated.Labels
			if gotLabels == nil {
				gotLabels = []string{}
			}
			wantLabels := tc.expectLabels
			if wantLabels == nil {
				wantLabels = []string{}
			}
			if len(gotLabels) != len(wantLabels) {
				t.Fatalf("Labels length = %d, want %d (got=%v want=%v)", len(gotLabels), len(wantLabels), gotLabels, wantLabels)
			}
			for i := range gotLabels {
				if gotLabels[i] != wantLabels[i] {
					t.Fatalf("Labels[%d] = %q, want %q", i, gotLabels[i], wantLabels[i])
				}
			}
		})
	}
}

// TestListAndSortHelpers verifies behavior for the covered scenario.
func TestListAndSortHelpers(t *testing.T) {
	repo := newFakeRepo()
	now := time.Date(2026, 2, 21, 12, 0, 0, 0, time.UTC)
	p, _ := domain.NewProjectFromInput(domain.ProjectInput{ID: "p1", Name: "Project"}, now)
	repo.projects[p.ID] = p
	c1, _ := domain.NewColumn("c1", p.ID, "First", 5, 0, now)
	c2, _ := domain.NewColumn("c2", p.ID, "Second", 1, 0, now)
	repo.columns[c1.ID] = c1
	repo.columns[c2.ID] = c2

	t1, _ := domain.NewActionItemForTest(domain.ActionItemInput{
		Kind:      domain.KindPlan,
		ID:        "t1",
		ProjectID: p.ID,
		ColumnID:  c1.ID,
		Position:  2,
		Title:     "later",
		Priority:  domain.PriorityLow,
	}, now)
	t2, _ := domain.NewActionItemForTest(domain.ActionItemInput{
		Kind:      domain.KindPlan,
		ID:        "t2",
		ProjectID: p.ID,
		ColumnID:  c1.ID,
		Position:  1,
		Title:     "earlier",
		Priority:  domain.PriorityLow,
	}, now)
	t3, _ := domain.NewActionItemForTest(domain.ActionItemInput{
		Kind:      domain.KindPlan,
		ID:        "t3",
		ProjectID: p.ID,
		ColumnID:  c2.ID,
		Position:  0,
		Title:     "other column",
		Priority:  domain.PriorityLow,
	}, now)
	repo.tasks[t1.ID] = t1
	repo.tasks[t2.ID] = t2
	repo.tasks[t3.ID] = t3

	svc := NewService(repo, nil, func() time.Time { return now }, ServiceConfig{})
	projects, err := svc.ListProjects(context.Background(), false)
	if err != nil {
		t.Fatalf("ListProjects() error = %v", err)
	}
	if len(projects) != 1 {
		t.Fatalf("expected 1 project, got %d", len(projects))
	}

	columns, err := svc.ListColumns(context.Background(), p.ID, false)
	if err != nil {
		t.Fatalf("ListColumns() error = %v", err)
	}
	if columns[0].ID != c2.ID {
		t.Fatalf("expected column c2 first after sort, got %q", columns[0].ID)
	}

	tasks, err := svc.ListActionItems(context.Background(), p.ID, false)
	if err != nil {
		t.Fatalf("ListActionItems() error = %v", err)
	}
	if len(tasks) != 3 {
		t.Fatalf("expected 3 tasks, got %d", len(tasks))
	}
	if tasks[0].ID != t2.ID || tasks[1].ID != t1.ID || tasks[2].ID != t3.ID {
		t.Fatalf("unexpected actionItem order: %#v", tasks)
	}

	allWithEmptyQuery, err := svc.SearchActionItemMatches(context.Background(), SearchActionItemsFilter{
		ProjectID: p.ID,
		Query:     " ",
	})
	if err != nil {
		t.Fatalf("SearchActionItemMatches(empty) error = %v", err)
	}
	if len(allWithEmptyQuery) != 3 {
		t.Fatalf("expected 3 results for empty query, got %d", len(allWithEmptyQuery))
	}
}

// TestSearchActionItemMatchesAcrossProjectsAndStates verifies behavior for the covered scenario.
func TestSearchActionItemMatchesAcrossProjectsAndStates(t *testing.T) {
	repo := newFakeRepo()
	now := time.Date(2026, 2, 21, 12, 0, 0, 0, time.UTC)
	p1, _ := domain.NewProjectFromInput(domain.ProjectInput{ID: "p1", Name: "Inbox"}, now)
	p2, _ := domain.NewProjectFromInput(domain.ProjectInput{ID: "p2", Name: "Client"}, now)
	repo.projects[p1.ID] = p1
	repo.projects[p2.ID] = p2

	c1, _ := domain.NewColumn("c1", p1.ID, "To Do", 0, 0, now)
	c2, _ := domain.NewColumn("c2", p1.ID, "In Progress", 1, 0, now)
	c3, _ := domain.NewColumn("c3", p2.ID, "In Progress", 0, 0, now)
	repo.columns[c1.ID] = c1
	repo.columns[c2.ID] = c2
	repo.columns[c3.ID] = c3

	t1, _ := domain.NewActionItemForTest(domain.ActionItemInput{
		Kind:        domain.KindPlan,
		ID:          "t1",
		ProjectID:   p1.ID,
		ColumnID:    c1.ID,
		Position:    0,
		Title:       "Roadmap draft",
		Description: "planning",
		Priority:    domain.PriorityMedium,
	}, now)
	t2, _ := domain.NewActionItemForTest(domain.ActionItemInput{
		Kind:        domain.KindPlan,
		ID:          "t2",
		ProjectID:   p1.ID,
		ColumnID:    c2.ID,
		Position:    0,
		Title:       "Implement parser",
		Description: "roadmap parser",
		Priority:    domain.PriorityHigh,
	}, now)
	t3, _ := domain.NewActionItemForTest(domain.ActionItemInput{
		Kind:        domain.KindPlan,
		ID:          "t3",
		ProjectID:   p2.ID,
		ColumnID:    c3.ID,
		Position:    0,
		Title:       "Client sync",
		Description: "roadmap review",
		Priority:    domain.PriorityLow,
	}, now)
	t3.Archive(now.Add(time.Minute))
	repo.tasks[t1.ID] = t1
	repo.tasks[t2.ID] = t2
	repo.tasks[t3.ID] = t3

	svc := NewService(repo, nil, func() time.Time { return now }, ServiceConfig{})

	matches, err := svc.SearchActionItemMatches(context.Background(), SearchActionItemsFilter{
		CrossProject:    true,
		IncludeArchived: false,
		States:          []string{"in_progress"},
		Query:           "parser",
	})
	if err != nil {
		t.Fatalf("SearchActionItemMatches() error = %v", err)
	}
	if len(matches) != 1 || matches[0].ActionItem.ID != "t2" || matches[0].StateID != "in_progress" {
		t.Fatalf("unexpected active matches %#v", matches)
	}

	matches, err = svc.SearchActionItemMatches(context.Background(), SearchActionItemsFilter{
		CrossProject:    true,
		IncludeArchived: true,
		States:          []string{"archived"},
		Query:           "roadmap",
	})
	if err != nil {
		t.Fatalf("SearchActionItemMatches(archived) error = %v", err)
	}
	if len(matches) != 1 || matches[0].ActionItem.ID != "t3" || matches[0].StateID != "archived" {
		t.Fatalf("unexpected archived matches %#v", matches)
	}
}

// TestSearchActionItemMatchesFuzzyQuery verifies behavior for the covered scenario.
func TestSearchActionItemMatchesFuzzyQuery(t *testing.T) {
	repo := newFakeRepo()
	now := time.Date(2026, 2, 21, 12, 0, 0, 0, time.UTC)
	p1, _ := domain.NewProjectFromInput(domain.ProjectInput{ID: "p1", Name: "Inbox"}, now)
	repo.projects[p1.ID] = p1

	c1, _ := domain.NewColumn("c1", p1.ID, "To Do", 0, 0, now)
	repo.columns[c1.ID] = c1

	t1, _ := domain.NewActionItemForTest(domain.ActionItemInput{
		Kind:        domain.KindPlan,
		ID:          "t1",
		ProjectID:   p1.ID,
		ColumnID:    c1.ID,
		Position:    0,
		Title:       "Implement parser",
		Description: "tokenization pipeline",
		Priority:    domain.PriorityMedium,
		Labels:      []string{"frontend", "parsing"},
	}, now)
	t2, _ := domain.NewActionItemForTest(domain.ActionItemInput{
		Kind:        domain.KindPlan,
		ID:          "t2",
		ProjectID:   p1.ID,
		ColumnID:    c1.ID,
		Position:    1,
		Title:       "Write docs",
		Description: "onboarding guide",
		Priority:    domain.PriorityLow,
		Labels:      []string{"docs"},
	}, now)
	repo.tasks[t1.ID] = t1
	repo.tasks[t2.ID] = t2

	svc := NewService(repo, nil, func() time.Time { return now }, ServiceConfig{})

	tests := []struct {
		name    string
		query   string
		wantIDs []string
	}{
		{
			name:    "title subsequence",
			query:   "imppsr",
			wantIDs: []string{"t1"},
		},
		{
			name:    "description subsequence",
			query:   "tkpln",
			wantIDs: []string{"t1"},
		},
		{
			name:    "label subsequence",
			query:   "frnd",
			wantIDs: []string{"t1"},
		},
		{
			name:    "preserves rune order",
			query:   "psrmpi",
			wantIDs: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches, err := svc.SearchActionItemMatches(context.Background(), SearchActionItemsFilter{
				ProjectID: p1.ID,
				Query:     tt.query,
			})
			if err != nil {
				t.Fatalf("SearchActionItemMatches() error = %v", err)
			}
			if len(matches) != len(tt.wantIDs) {
				t.Fatalf("expected %d results, got %d for query %q", len(tt.wantIDs), len(matches), tt.query)
			}
			for i := range tt.wantIDs {
				if matches[i].ActionItem.ID != tt.wantIDs[i] {
					t.Fatalf("unexpected result order for query %q: got %q want %q", tt.query, matches[i].ActionItem.ID, tt.wantIDs[i])
				}
			}
		})
	}
}

// TestSearchActionItemMatchesExtendedFilters verifies optional level/kind/label filters and pagination.
func TestSearchActionItemMatchesExtendedFilters(t *testing.T) {
	repo := newFakeRepo()
	now := time.Date(2026, 3, 3, 11, 0, 0, 0, time.UTC)
	project, _ := domain.NewProjectFromInput(domain.ProjectInput{ID: "p1", Name: "Inbox"}, now)
	repo.projects[project.ID] = project

	todo, _ := domain.NewColumn("c1", project.ID, "To Do", 0, 0, now)
	progress, _ := domain.NewColumn("c2", project.ID, "In Progress", 1, 0, now)
	repo.columns[todo.ID] = todo
	repo.columns[progress.ID] = progress

	t1, _ := domain.NewActionItemForTest(domain.ActionItemInput{
		ID:        "t1",
		ProjectID: project.ID,
		ColumnID:  todo.ID,
		Position:  0,
		Title:     "Phase planning",
		Kind:      domain.KindDiscussion,
		Scope:     domain.KindAppliesToDiscussion,
		Priority:  domain.PriorityMedium,
		Labels:    []string{"backend", "urgent"},
	}, now)
	t2, _ := domain.NewActionItemForTest(domain.ActionItemInput{
		ID:        "t2",
		ProjectID: project.ID,
		ColumnID:  todo.ID,
		Position:  1,
		Title:     "Phase QA",
		Kind:      domain.KindDiscussion,
		Scope:     domain.KindAppliesToDiscussion,
		Priority:  domain.PriorityMedium,
		Labels:    []string{"backend"},
	}, now)
	t3, _ := domain.NewActionItemForTest(domain.ActionItemInput{
		ID:        "t3",
		ProjectID: project.ID,
		ColumnID:  progress.ID,
		Position:  0,
		Title:     "ActionItem implementation",
		Kind:      domain.KindPlan,
		Scope:     domain.KindAppliesToPlan,
		Priority:  domain.PriorityMedium,
		Labels:    []string{"backend", "urgent"},
	}, now)
	t4, _ := domain.NewActionItemForTest(domain.ActionItemInput{
		ID:        "t4",
		ProjectID: project.ID,
		ColumnID:  progress.ID,
		Position:  1,
		Title:     "Archived phase note",
		Kind:      domain.KindDiscussion,
		Scope:     domain.KindAppliesToDiscussion,
		Priority:  domain.PriorityLow,
		Labels:    []string{"backend", "urgent"},
	}, now)
	t4.Archive(now.Add(1 * time.Minute))
	repo.tasks[t1.ID] = t1
	repo.tasks[t2.ID] = t2
	repo.tasks[t3.ID] = t3
	repo.tasks[t4.ID] = t4

	svc := NewService(repo, nil, func() time.Time { return now }, ServiceConfig{})

	strictMatches, err := svc.SearchActionItemMatches(context.Background(), SearchActionItemsFilter{
		ProjectID: project.ID,
		Levels:    []string{"discussion"},
		Kinds:     []string{"discussion"},
		LabelsAny: []string{"backend"},
		LabelsAll: []string{"urgent"},
	})
	if err != nil {
		t.Fatalf("SearchActionItemMatches(strict filters) error = %v", err)
	}
	if len(strictMatches) != 1 || strictMatches[0].ActionItem.ID != "t1" {
		t.Fatalf("strict filter rows = %#v, want only t1", strictMatches)
	}

	pagedMatches, err := svc.SearchActionItemMatches(context.Background(), SearchActionItemsFilter{
		ProjectID: project.ID,
		Levels:    []string{"discussion"},
		Kinds:     []string{"discussion"},
		LabelsAny: []string{"backend"},
		Limit:     1,
		Offset:    1,
	})
	if err != nil {
		t.Fatalf("SearchActionItemMatches(paged filters) error = %v", err)
	}
	if len(pagedMatches) != 1 || pagedMatches[0].ActionItem.ID != "t2" {
		t.Fatalf("paged filter rows = %#v, want only t2", pagedMatches)
	}

	archivedMatches, err := svc.SearchActionItemMatches(context.Background(), SearchActionItemsFilter{
		ProjectID:       project.ID,
		IncludeArchived: true,
		States:          []string{"archived"},
		Levels:          []string{"discussion"},
		Kinds:           []string{"discussion"},
		LabelsAny:       []string{"backend"},
		LabelsAll:       []string{"urgent"},
	})
	if err != nil {
		t.Fatalf("SearchActionItemMatches(archived filters) error = %v", err)
	}
	if len(archivedMatches) != 1 || archivedMatches[0].ActionItem.ID != "t4" || archivedMatches[0].StateID != "archived" {
		t.Fatalf("archived filter rows = %#v, want only archived t4", archivedMatches)
	}
}

// TestSearchActionItemMatchesLexicalMetadataFields verifies lexical scoring covers embedding metadata fields.
func TestSearchActionItemMatchesLexicalMetadataFields(t *testing.T) {
	repo := newFakeRepo()
	now := time.Date(2026, 3, 3, 11, 30, 0, 0, time.UTC)
	project, _ := domain.NewProjectFromInput(domain.ProjectInput{ID: "p1", Name: "Inbox"}, now)
	repo.projects[project.ID] = project
	column, _ := domain.NewColumn("c1", project.ID, "To Do", 0, 0, now)
	repo.columns[column.ID] = column

	actionItem, _ := domain.NewActionItemForTest(domain.ActionItemInput{
		Kind:        domain.KindPlan,
		ID:          "t1",
		ProjectID:   project.ID,
		ColumnID:    column.ID,
		Position:    0,
		Title:       "General follow-up",
		Description: "No metadata keywords in primary fields",
		Priority:    domain.PriorityLow,
		Labels:      []string{"ops"},
		Metadata: domain.ActionItemMetadata{
			Objective:          "objective-alpha-signal",
			AcceptanceCriteria: "acceptance-beta-signal",
			ValidationPlan:     "validation-gamma-signal",
			BlockedReason:      "blocked-delta-signal",
			RiskNotes:          "risk-epsilon-signal",
		},
	}, now)
	repo.tasks[actionItem.ID] = actionItem

	svc := NewService(repo, nil, func() time.Time { return now }, ServiceConfig{})
	tests := []struct {
		name  string
		query string
	}{
		{name: "objective", query: "objective-alpha-signal"},
		{name: "acceptance_criteria", query: "acceptance-beta-signal"},
		{name: "validation_plan", query: "validation-gamma-signal"},
		{name: "blocked_reason", query: "blocked-delta-signal"},
		{name: "risk_notes", query: "risk-epsilon-signal"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches, err := svc.SearchActionItemMatches(context.Background(), SearchActionItemsFilter{
				ProjectID: project.ID,
				Query:     tt.query,
			})
			if err != nil {
				t.Fatalf("SearchActionItemMatches(%s) error = %v", tt.name, err)
			}
			if len(matches) != 1 || matches[0].ActionItem.ID != actionItem.ID {
				t.Fatalf("query %q rows = %#v, want only %q", tt.query, matches, actionItem.ID)
			}
		})
	}
}

// TestSearchActionItemMatchesSortAndPagination verifies optioned sorting and pagination behavior.
func TestSearchActionItemMatchesSortAndPagination(t *testing.T) {
	repo := newFakeRepo()
	now := time.Date(2026, 2, 21, 12, 0, 0, 0, time.UTC)
	project, _ := domain.NewProjectFromInput(domain.ProjectInput{ID: "p1", Name: "Inbox"}, now)
	repo.projects[project.ID] = project

	column, _ := domain.NewColumn("c1", project.ID, "To Do", 0, 0, now)
	repo.columns[column.ID] = column

	t1, _ := domain.NewActionItemForTest(domain.ActionItemInput{
		Kind:      domain.KindPlan,
		ID:        "t1",
		ProjectID: project.ID,
		ColumnID:  column.ID,
		Position:  2,
		Title:     "Charlie",
		Priority:  domain.PriorityLow,
	}, now)
	t2, _ := domain.NewActionItemForTest(domain.ActionItemInput{
		Kind:      domain.KindPlan,
		ID:        "t2",
		ProjectID: project.ID,
		ColumnID:  column.ID,
		Position:  0,
		Title:     "Alpha",
		Priority:  domain.PriorityLow,
	}, now)
	t3, _ := domain.NewActionItemForTest(domain.ActionItemInput{
		Kind:      domain.KindPlan,
		ID:        "t3",
		ProjectID: project.ID,
		ColumnID:  column.ID,
		Position:  1,
		Title:     "Bravo",
		Priority:  domain.PriorityLow,
	}, now)
	t4, _ := domain.NewActionItemForTest(domain.ActionItemInput{
		Kind:      domain.KindPlan,
		ID:        "t4",
		ProjectID: project.ID,
		ColumnID:  column.ID,
		Position:  3,
		Title:     "Alpha",
		Priority:  domain.PriorityLow,
	}, now)

	t1.CreatedAt = now.Add(1 * time.Minute)
	t2.CreatedAt = now.Add(3 * time.Minute)
	t3.CreatedAt = now.Add(2 * time.Minute)
	t4.CreatedAt = now.Add(4 * time.Minute)
	t1.UpdatedAt = now.Add(10 * time.Minute)
	t2.UpdatedAt = now.Add(3 * time.Minute)
	t3.UpdatedAt = now.Add(20 * time.Minute)
	t4.UpdatedAt = now.Add(15 * time.Minute)

	repo.tasks[t1.ID] = t1
	repo.tasks[t2.ID] = t2
	repo.tasks[t3.ID] = t3
	repo.tasks[t4.ID] = t4

	svc := NewService(repo, nil, func() time.Time { return now }, ServiceConfig{})

	tests := []struct {
		name    string
		filter  SearchActionItemsFilter
		wantIDs []string
	}{
		{
			name: "default rank_desc order",
			filter: SearchActionItemsFilter{
				ProjectID: project.ID,
			},
			wantIDs: []string{"t2", "t3", "t1", "t4"},
		},
		{
			name: "title_asc sort with deterministic tie-breaker",
			filter: SearchActionItemsFilter{
				ProjectID: project.ID,
				Sort:      SearchSortTitleAsc,
			},
			wantIDs: []string{"t2", "t4", "t3", "t1"},
		},
		{
			name: "created_at_desc sort",
			filter: SearchActionItemsFilter{
				ProjectID: project.ID,
				Sort:      SearchSortCreatedAtDesc,
			},
			wantIDs: []string{"t4", "t2", "t3", "t1"},
		},
		{
			name: "updated_at_desc sort",
			filter: SearchActionItemsFilter{
				ProjectID: project.ID,
				Sort:      SearchSortUpdatedAtDesc,
			},
			wantIDs: []string{"t3", "t4", "t1", "t2"},
		},
		{
			name: "pagination limit and offset",
			filter: SearchActionItemsFilter{
				ProjectID: project.ID,
				Limit:     2,
				Offset:    1,
			},
			wantIDs: []string{"t3", "t1"},
		},
		{
			name: "hybrid mode default remains backward-compatible in this lane",
			filter: SearchActionItemsFilter{
				ProjectID: project.ID,
				Query:     "alpha",
			},
			wantIDs: []string{"t2", "t4"},
		},
		{
			name: "semantic mode remains backward-compatible in this lane",
			filter: SearchActionItemsFilter{
				ProjectID: project.ID,
				Query:     "alpha",
				Mode:      SearchModeSemantic,
			},
			wantIDs: []string{"t2", "t4"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches, err := svc.SearchActionItemMatches(context.Background(), tt.filter)
			if err != nil {
				t.Fatalf("SearchActionItemMatches() error = %v", err)
			}
			if len(matches) != len(tt.wantIDs) {
				t.Fatalf("expected %d rows, got %d", len(tt.wantIDs), len(matches))
			}
			for idx := range tt.wantIDs {
				if matches[idx].ActionItem.ID != tt.wantIDs[idx] {
					t.Fatalf("unexpected id at %d: got %q want %q", idx, matches[idx].ActionItem.ID, tt.wantIDs[idx])
				}
			}
		})
	}
}

// TestSearchActionItemMatchesLimitDefaultsAndCaps verifies default and capped limits.
func TestSearchActionItemMatchesLimitDefaultsAndCaps(t *testing.T) {
	repo := newFakeRepo()
	now := time.Date(2026, 2, 21, 12, 0, 0, 0, time.UTC)
	project, _ := domain.NewProjectFromInput(domain.ProjectInput{ID: "p1", Name: "Inbox"}, now)
	repo.projects[project.ID] = project

	column, _ := domain.NewColumn("c1", project.ID, "To Do", 0, 0, now)
	repo.columns[column.ID] = column

	for i := 0; i < 205; i++ {
		actionItem, _ := domain.NewActionItemForTest(domain.ActionItemInput{
			Kind:      domain.KindPlan,
			ID:        fmt.Sprintf("t%03d", i),
			ProjectID: project.ID,
			ColumnID:  column.ID,
			Position:  i,
			Title:     fmt.Sprintf("ActionItem %03d", i),
			Priority:  domain.PriorityLow,
		}, now)
		repo.tasks[actionItem.ID] = actionItem
	}

	svc := NewService(repo, nil, func() time.Time { return now }, ServiceConfig{})

	defaultRows, err := svc.SearchActionItemMatches(context.Background(), SearchActionItemsFilter{ProjectID: project.ID})
	if err != nil {
		t.Fatalf("SearchActionItemMatches(default limit) error = %v", err)
	}
	if len(defaultRows) != 50 {
		t.Fatalf("default limit rows = %d, want 50", len(defaultRows))
	}
	if defaultRows[0].ActionItem.ID != "t000" || defaultRows[len(defaultRows)-1].ActionItem.ID != "t049" {
		t.Fatalf("unexpected default row ids: first=%q last=%q", defaultRows[0].ActionItem.ID, defaultRows[len(defaultRows)-1].ActionItem.ID)
	}

	clampedRows, err := svc.SearchActionItemMatches(context.Background(), SearchActionItemsFilter{
		ProjectID: project.ID,
		Limit:     500,
	})
	if err != nil {
		t.Fatalf("SearchActionItemMatches(clamped limit) error = %v", err)
	}
	if len(clampedRows) != 200 {
		t.Fatalf("clamped limit rows = %d, want 200", len(clampedRows))
	}

	tailRows, err := svc.SearchActionItemMatches(context.Background(), SearchActionItemsFilter{
		ProjectID: project.ID,
		Limit:     20,
		Offset:    198,
	})
	if err != nil {
		t.Fatalf("SearchActionItemMatches(tail page) error = %v", err)
	}
	if len(tailRows) != 7 {
		t.Fatalf("tail page rows = %d, want 7", len(tailRows))
	}
	if tailRows[0].ActionItem.ID != "t198" || tailRows[len(tailRows)-1].ActionItem.ID != "t204" {
		t.Fatalf("unexpected tail row ids: first=%q last=%q", tailRows[0].ActionItem.ID, tailRows[len(tailRows)-1].ActionItem.ID)
	}
}

// TestSearchActionItemMatchesRejectsInvalidOptions verifies mode/sort/pagination validation.
func TestSearchActionItemMatchesRejectsInvalidOptions(t *testing.T) {
	svc := NewService(newFakeRepo(), nil, time.Now, ServiceConfig{})
	tests := []struct {
		name   string
		filter SearchActionItemsFilter
	}{
		{
			name: "invalid mode",
			filter: SearchActionItemsFilter{
				Mode: "unsupported",
			},
		},
		{
			name: "invalid sort",
			filter: SearchActionItemsFilter{
				Sort: "unsupported",
			},
		},
		{
			name: "negative limit",
			filter: SearchActionItemsFilter{
				Limit: -1,
			},
		},
		{
			name: "negative offset",
			filter: SearchActionItemsFilter{
				Offset: -1,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.SearchActionItemMatches(context.Background(), tt.filter)
			if !errors.Is(err, domain.ErrInvalidID) {
				t.Fatalf("expected ErrInvalidID, got %v", err)
			}
		})
	}
}

// TestServiceCreateAndUpdateActionItemEnqueueEmbeddingLifecycle verifies actionItem writes enqueue durable lifecycle work.
func TestServiceCreateAndUpdateActionItemEnqueueEmbeddingLifecycle(t *testing.T) {
	repo := newFakeRepo()
	now := time.Date(2026, 3, 3, 10, 0, 0, 0, time.UTC)
	project, _ := domain.NewProjectFromInput(domain.ProjectInput{ID: "p1", Name: "Inbox"}, now)
	repo.projects[project.ID] = project
	column, _ := domain.NewColumn("c1", project.ID, "To Do", 0, 0, now)
	repo.columns[column.ID] = column

	lifecycle := newFakeEmbeddingLifecycleStore()
	svc := NewService(repo, func() string { return "t1" }, func() time.Time { return now }, ServiceConfig{
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

	created, err := svc.CreateActionItem(context.Background(), CreateActionItemInput{
		ProjectID:      project.ID,
		ColumnID:       column.ID,
		Title:          "Ship search",
		Description:    "Finalize ranking",
		Priority:       domain.PriorityMedium,
		Labels:         []string{"search", "vector"},
		UpdatedByType:  domain.ActorTypeUser,
		StructuralType: domain.StructuralTypeDroplet,
		Metadata: domain.ActionItemMetadata{
			Objective:          "Stabilize search quality",
			AcceptanceCriteria: "Rank semantic matches first",
			ValidationPlan:     "Run focused package tests",
			BlockedReason:      "waiting for API key",
			RiskNotes:          "weight tuning may regress lexical ranking",
		},
	})
	if err != nil {
		t.Fatalf("CreateActionItem() error = %v", err)
	}
	if len(lifecycle.inputs) != 1 {
		t.Fatalf("expected 1 enqueue after create, got %d", len(lifecycle.inputs))
	}
	content := buildActionItemEmbeddingContent(created)
	for _, want := range []string{
		created.Title,
		"Finalize ranking",
		"Stabilize search quality",
		"Rank semantic matches first",
		"Run focused package tests",
		"waiting for API key",
		"weight tuning may regress lexical ranking",
	} {
		if !strings.Contains(content, want) {
			t.Fatalf("embedding content missing %q: %q", want, content)
		}
	}
	if lifecycle.inputs[0].ContentHash != hashEmbeddingContent(content) {
		t.Fatalf("create content hash = %q, want %q", lifecycle.inputs[0].ContentHash, hashEmbeddingContent(content))
	}

	updated, err := svc.UpdateActionItem(context.Background(), UpdateActionItemInput{
		ActionItemID: created.ID,
		Title:        ptrTo("Ship hybrid search"),
		Description:  ptrTo(created.Description),
		Priority:     ptrTo(created.Priority),
		Labels:       ptrTo(created.Labels),
		DueAt:        ptrTo(created.DueAt),
		Metadata:     &created.Metadata,
	})
	if err != nil {
		t.Fatalf("UpdateActionItem() error = %v", err)
	}
	if len(lifecycle.inputs) != 2 {
		t.Fatalf("expected second enqueue after update, got %d", len(lifecycle.inputs))
	}
	if lifecycle.inputs[0].ContentHash == lifecycle.inputs[1].ContentHash {
		t.Fatalf("expected content hash change after title update, both were %q", lifecycle.inputs[0].ContentHash)
	}
	if len(lifecycle.enqueues) != 1 {
		t.Fatalf("expected one tracked lifecycle row, got %d", len(lifecycle.enqueues))
	}
	row := lifecycle.enqueues[lifecycle.embeddingKey(EmbeddingSubjectTypeWorkItem, created.ID)]
	if row.Status != EmbeddingLifecyclePending {
		t.Fatalf("status = %s, want pending", row.Status)
	}
	if row.ContentHashDesired != hashEmbeddingContent(buildActionItemEmbeddingContent(updated)) {
		t.Fatalf("desired content hash = %q, want %q", row.ContentHashDesired, hashEmbeddingContent(buildActionItemEmbeddingContent(updated)))
	}
}

// TestSearchActionItemMatchesSemanticModeUsesIndex verifies semantic mode ranking can return non-lexical rows.
func TestSearchActionItemMatchesSemanticModeUsesIndex(t *testing.T) {
	repo := newFakeRepo()
	now := time.Date(2026, 3, 3, 12, 0, 0, 0, time.UTC)
	project, _ := domain.NewProjectFromInput(domain.ProjectInput{ID: "p1", Name: "Inbox"}, now)
	repo.projects[project.ID] = project
	column, _ := domain.NewColumn("c1", project.ID, "To Do", 0, 0, now)
	repo.columns[column.ID] = column

	t1, _ := domain.NewActionItemForTest(domain.ActionItemInput{
		Kind:      domain.KindPlan,
		ID:        "t1",
		ProjectID: project.ID,
		ColumnID:  column.ID,
		Position:  0,
		Title:     "Update docs",
		Priority:  domain.PriorityLow,
	}, now)
	t2, _ := domain.NewActionItemForTest(domain.ActionItemInput{
		Kind:      domain.KindPlan,
		ID:        "t2",
		ProjectID: project.ID,
		ColumnID:  column.ID,
		Position:  1,
		Title:     "Improve observability",
		Priority:  domain.PriorityLow,
	}, now)
	repo.tasks[t1.ID] = t1
	repo.tasks[t2.ID] = t2

	searchIndex := &fakeActionItemSearchIndex{
		searchRows: []EmbeddingSearchMatch{
			{SubjectType: EmbeddingSubjectTypeWorkItem, SubjectID: "t2", SearchTargetType: EmbeddingSearchTargetTypeWorkItem, SearchTargetID: "t2", Similarity: 0.93},
			{SubjectType: EmbeddingSubjectTypeWorkItem, SubjectID: "t1", SearchTargetType: EmbeddingSearchTargetTypeWorkItem, SearchTargetID: "t1", Similarity: 0.22},
		},
	}
	embedder := &fakeEmbeddingGenerator{vectors: [][]float32{{0.7, 0.1, 0.3}}}
	lifecycle := newFakeEmbeddingLifecycleStore()
	seedReadyActionItemEmbeddings(lifecycle, project.ID, t1, t2)
	svc := NewService(repo, nil, func() time.Time { return now }, ServiceConfig{
		EmbeddingGenerator: embedder,
		SearchIndex:        searchIndex,
		EmbeddingLifecycle: lifecycle,
	})

	result, err := svc.SearchActionItems(context.Background(), SearchActionItemsFilter{
		ProjectID: project.ID,
		Query:     "semantic query",
		Mode:      SearchModeSemantic,
		Limit:     10,
	})
	if err != nil {
		t.Fatalf("SearchActionItems() error = %v", err)
	}
	matches := result.Matches
	if len(matches) != 2 {
		t.Fatalf("semantic mode rows = %d, want 2", len(matches))
	}
	if matches[0].ActionItem.ID != "t2" || matches[1].ActionItem.ID != "t1" {
		t.Fatalf("unexpected semantic ordering: %#v", []string{matches[0].ActionItem.ID, matches[1].ActionItem.ID})
	}
	if len(searchIndex.searchIn.ProjectIDs) != 1 || searchIndex.searchIn.ProjectIDs[0] != project.ID {
		t.Fatalf("semantic project filter = %#v, want [%s]", searchIndex.searchIn.ProjectIDs, project.ID)
	}
	if result.RequestedMode != SearchModeSemantic || result.EffectiveMode != SearchModeSemantic {
		t.Fatalf("search modes = requested %q effective %q, want semantic/semantic", result.RequestedMode, result.EffectiveMode)
	}
	if !result.SemanticAvailable {
		t.Fatal("expected semantic search to be available")
	}
	if !matches[0].UsedSemantic || matches[0].SemanticScore <= 0 {
		t.Fatalf("expected top semantic match metadata, got %#v", matches[0])
	}
}

// TestSearchActionItemMatchesSemanticFallsBackToKeyword verifies semantic mode falls back when embeddings are unavailable.
func TestSearchActionItemMatchesSemanticFallsBackToKeyword(t *testing.T) {
	repo := newFakeRepo()
	now := time.Date(2026, 3, 3, 12, 15, 0, 0, time.UTC)
	project, _ := domain.NewProjectFromInput(domain.ProjectInput{ID: "p1", Name: "Inbox"}, now)
	repo.projects[project.ID] = project
	column, _ := domain.NewColumn("c1", project.ID, "To Do", 0, 0, now)
	repo.columns[column.ID] = column

	keywordActionItem, _ := domain.NewActionItemForTest(domain.ActionItemInput{
		Kind:      domain.KindPlan,
		ID:        "t1",
		ProjectID: project.ID,
		ColumnID:  column.ID,
		Position:  0,
		Title:     "Server rollout checklist",
		Priority:  domain.PriorityLow,
	}, now)
	otherActionItem, _ := domain.NewActionItemForTest(domain.ActionItemInput{
		Kind:      domain.KindPlan,
		ID:        "t2",
		ProjectID: project.ID,
		ColumnID:  column.ID,
		Position:  1,
		Title:     "Roadmap grooming",
		Priority:  domain.PriorityLow,
	}, now)
	repo.tasks[keywordActionItem.ID] = keywordActionItem
	repo.tasks[otherActionItem.ID] = otherActionItem

	embedder := &fakeEmbeddingGenerator{err: errors.New("embedding unavailable")}
	searchIndex := &fakeActionItemSearchIndex{
		searchRows: []EmbeddingSearchMatch{{SubjectType: EmbeddingSubjectTypeWorkItem, SubjectID: "t2", SearchTargetType: EmbeddingSearchTargetTypeWorkItem, SearchTargetID: "t2", Similarity: 0.99}},
	}
	lifecycle := newFakeEmbeddingLifecycleStore()
	seedReadyActionItemEmbeddings(lifecycle, project.ID, keywordActionItem, otherActionItem)
	svc := NewService(repo, nil, func() time.Time { return now }, ServiceConfig{
		EmbeddingGenerator: embedder,
		SearchIndex:        searchIndex,
		EmbeddingLifecycle: lifecycle,
	})

	result, err := svc.SearchActionItems(context.Background(), SearchActionItemsFilter{
		ProjectID: project.ID,
		Query:     "server",
		Mode:      SearchModeSemantic,
	})
	if err != nil {
		t.Fatalf("SearchActionItems() error = %v", err)
	}
	matches := result.Matches
	if len(matches) != 1 || matches[0].ActionItem.ID != keywordActionItem.ID {
		t.Fatalf("semantic fallback rows = %#v, want only %q", matches, keywordActionItem.ID)
	}
	if result.RequestedMode != SearchModeSemantic || result.EffectiveMode != SearchModeKeyword {
		t.Fatalf("search modes = requested %q effective %q, want semantic/keyword", result.RequestedMode, result.EffectiveMode)
	}
	if result.FallbackReason != "query_embedding_failed" {
		t.Fatalf("fallback reason = %q, want query_embedding_failed", result.FallbackReason)
	}
	if result.SemanticAvailable {
		t.Fatal("expected semantic search availability to be false after embed failure")
	}
}

// TestSearchActionItemMatchesSemanticModeDuplicateRowsKeepMaxSimilarity verifies duplicate semantic rows keep the highest similarity per actionItem.
func TestSearchActionItemMatchesSemanticModeDuplicateRowsKeepMaxSimilarity(t *testing.T) {
	repo := newFakeRepo()
	now := time.Date(2026, 3, 3, 12, 20, 0, 0, time.UTC)
	project, _ := domain.NewProjectFromInput(domain.ProjectInput{ID: "p1", Name: "Inbox"}, now)
	repo.projects[project.ID] = project
	column, _ := domain.NewColumn("c1", project.ID, "To Do", 0, 0, now)
	repo.columns[column.ID] = column

	t1, _ := domain.NewActionItemForTest(domain.ActionItemInput{
		Kind:      domain.KindPlan,
		ID:        "t1",
		ProjectID: project.ID,
		ColumnID:  column.ID,
		Position:  0,
		Title:     "Alpha",
		Priority:  domain.PriorityLow,
	}, now)
	t2, _ := domain.NewActionItemForTest(domain.ActionItemInput{
		Kind:      domain.KindPlan,
		ID:        "t2",
		ProjectID: project.ID,
		ColumnID:  column.ID,
		Position:  1,
		Title:     "Bravo",
		Priority:  domain.PriorityLow,
	}, now)
	repo.tasks[t1.ID] = t1
	repo.tasks[t2.ID] = t2

	searchIndex := &fakeActionItemSearchIndex{
		searchRows: []EmbeddingSearchMatch{
			{SubjectType: EmbeddingSubjectTypeWorkItem, SubjectID: "t1", SearchTargetType: EmbeddingSearchTargetTypeWorkItem, SearchTargetID: "t1", Similarity: 0.93},
			{SubjectType: EmbeddingSubjectTypeWorkItem, SubjectID: "t2", SearchTargetType: EmbeddingSearchTargetTypeWorkItem, SearchTargetID: "t2", Similarity: 0.85},
			{SubjectType: EmbeddingSubjectTypeWorkItem, SubjectID: "t1", SearchTargetType: EmbeddingSearchTargetTypeWorkItem, SearchTargetID: "t1", Similarity: 0.10},
		},
	}
	embedder := &fakeEmbeddingGenerator{vectors: [][]float32{{0.4, 0.2, 0.9}}}
	lifecycle := newFakeEmbeddingLifecycleStore()
	seedReadyActionItemEmbeddings(lifecycle, project.ID, t1, t2)
	svc := NewService(repo, nil, func() time.Time { return now }, ServiceConfig{
		EmbeddingGenerator: embedder,
		SearchIndex:        searchIndex,
		EmbeddingLifecycle: lifecycle,
	})

	matches, err := svc.SearchActionItemMatches(context.Background(), SearchActionItemsFilter{
		ProjectID: project.ID,
		Query:     "semantic query",
		Mode:      SearchModeSemantic,
		Limit:     10,
	})
	if err != nil {
		t.Fatalf("SearchActionItemMatches() error = %v", err)
	}
	if len(matches) != 2 {
		t.Fatalf("semantic mode rows = %d, want 2", len(matches))
	}
	if matches[0].ActionItem.ID != "t1" || matches[1].ActionItem.ID != "t2" {
		t.Fatalf("unexpected semantic ordering with duplicate rows: %#v", []string{matches[0].ActionItem.ID, matches[1].ActionItem.ID})
	}
}

// TestSearchActionItemMatchesHybridFallsBackToKeyword verifies hybrid mode falls back when semantic lookup fails.
func TestSearchActionItemMatchesHybridFallsBackToKeyword(t *testing.T) {
	repo := newFakeRepo()
	now := time.Date(2026, 3, 3, 12, 30, 0, 0, time.UTC)
	project, _ := domain.NewProjectFromInput(domain.ProjectInput{ID: "p1", Name: "Inbox"}, now)
	repo.projects[project.ID] = project
	column, _ := domain.NewColumn("c1", project.ID, "To Do", 0, 0, now)
	repo.columns[column.ID] = column

	keywordActionItem, _ := domain.NewActionItemForTest(domain.ActionItemInput{
		Kind:      domain.KindPlan,
		ID:        "t1",
		ProjectID: project.ID,
		ColumnID:  column.ID,
		Position:  0,
		Title:     "Server rollout checklist",
		Priority:  domain.PriorityLow,
	}, now)
	otherActionItem, _ := domain.NewActionItemForTest(domain.ActionItemInput{
		Kind:      domain.KindPlan,
		ID:        "t2",
		ProjectID: project.ID,
		ColumnID:  column.ID,
		Position:  1,
		Title:     "Roadmap grooming",
		Priority:  domain.PriorityLow,
	}, now)
	repo.tasks[keywordActionItem.ID] = keywordActionItem
	repo.tasks[otherActionItem.ID] = otherActionItem

	embedder := &fakeEmbeddingGenerator{err: errors.New("embedding unavailable")}
	searchIndex := &fakeActionItemSearchIndex{
		searchRows: []EmbeddingSearchMatch{{SubjectType: EmbeddingSubjectTypeWorkItem, SubjectID: "t2", SearchTargetType: EmbeddingSearchTargetTypeWorkItem, SearchTargetID: "t2", Similarity: 0.99}},
	}
	svc := NewService(repo, nil, func() time.Time { return now }, ServiceConfig{
		EmbeddingGenerator: embedder,
		SearchIndex:        searchIndex,
	})

	matches, err := svc.SearchActionItemMatches(context.Background(), SearchActionItemsFilter{
		ProjectID: project.ID,
		Query:     "server",
		Mode:      SearchModeHybrid,
	})
	if err != nil {
		t.Fatalf("SearchActionItemMatches() error = %v", err)
	}
	if len(matches) != 1 || matches[0].ActionItem.ID != keywordActionItem.ID {
		t.Fatalf("hybrid fallback rows = %#v, want only %q", matches, keywordActionItem.ID)
	}
}

// TestEnsureDefaultProjectAlreadyExists verifies behavior for the covered scenario.
func TestEnsureDefaultProjectAlreadyExists(t *testing.T) {
	repo := newFakeRepo()
	now := time.Now()
	p, _ := domain.NewProjectFromInput(domain.ProjectInput{ID: "p1", Name: "Existing"}, now)
	repo.projects[p.ID] = p

	svc := NewService(repo, func() string { return "new-id" }, func() time.Time { return now }, ServiceConfig{})
	got, err := svc.EnsureDefaultProject(context.Background())
	if err != nil {
		t.Fatalf("EnsureDefaultProject() error = %v", err)
	}
	if got.ID != p.ID {
		t.Fatalf("expected existing project id %q, got %q", p.ID, got.ID)
	}
	if len(repo.columns) != 0 {
		t.Fatalf("expected no default columns to be inserted, got %d", len(repo.columns))
	}
}

// TestCreateProjectWithMetadataAndAutoColumns verifies behavior for the covered scenario.
func TestCreateProjectWithMetadataAndAutoColumns(t *testing.T) {
	repo := newFakeRepo()
	now := time.Date(2026, 2, 21, 12, 0, 0, 0, time.UTC)
	ids := []string{"p1", "c1", "c2"}
	idx := 0
	svc := NewService(repo, func() string {
		id := ids[idx]
		idx++
		return id
	}, func() time.Time { return now }, ServiceConfig{
		AutoCreateProjectColumns: true,
		StateTemplates: []StateTemplate{
			{ID: "todo", Name: "To Do", Position: 0},
			{ID: "doing", Name: "Doing", Position: 1},
		},
	})

	project, err := svc.CreateProjectWithMetadata(context.Background(), CreateProjectInput{
		Name:        "Roadmap",
		Description: "Q2 plan",
		Metadata: domain.ProjectMetadata{
			Owner: "Evan",
			Tags:  []string{"Roadmap", "roadmap"},
		},
	})
	if err != nil {
		t.Fatalf("CreateProjectWithMetadata() error = %v", err)
	}
	if project.Metadata.Owner != "Evan" || len(project.Metadata.Tags) != 1 {
		t.Fatalf("unexpected project metadata %#v", project.Metadata)
	}
	columns, err := svc.ListColumns(context.Background(), project.ID, false)
	if err != nil {
		t.Fatalf("ListColumns() error = %v", err)
	}
	if len(columns) != 2 {
		t.Fatalf("expected 2 auto-created columns, got %d", len(columns))
	}
	if columns[0].Name != "To Do" || columns[1].Name != "Doing" {
		t.Fatalf("unexpected column names %#v", columns)
	}
}

// TestCreateProjectWithMetadataDefaultsOwnerFromResolvedUser verifies empty owner metadata falls back to the acting local user.
func TestCreateProjectWithMetadataDefaultsOwnerFromResolvedUser(t *testing.T) {
	repo := newFakeRepo()
	now := time.Date(2026, 3, 29, 12, 0, 0, 0, time.UTC)
	svc := NewService(repo, func() string { return "p1" }, func() time.Time { return now }, ServiceConfig{
		AutoCreateProjectColumns: false,
	})
	ctx := WithMutationActor(context.Background(), MutationActor{
		ActorID:   "user-1",
		ActorName: "Evan",
		ActorType: domain.ActorTypeUser,
	})

	project, err := svc.CreateProjectWithMetadata(ctx, CreateProjectInput{
		Name:        "Inbox",
		Description: "Owner fallback",
	})
	if err != nil {
		t.Fatalf("CreateProjectWithMetadata() error = %v", err)
	}
	if got := project.Metadata.Owner; got != "Evan" {
		t.Fatalf("project owner = %q, want %q", got, "Evan")
	}
	if got := repo.projects[project.ID].Metadata.Owner; got != "Evan" {
		t.Fatalf("persisted project owner = %q, want %q", got, "Evan")
	}
}

// TestUpdateProject verifies behavior for the covered scenario.
func TestUpdateProject(t *testing.T) {
	repo := newFakeRepo()
	now := time.Date(2026, 2, 21, 12, 0, 0, 0, time.UTC)
	project, _ := domain.NewProjectFromInput(domain.ProjectInput{ID: "p1", Name: "Inbox", Description: "old desc"}, now)
	repo.projects[project.ID] = project

	svc := NewService(repo, nil, func() time.Time { return now.Add(time.Minute) }, ServiceConfig{})
	updated, err := svc.UpdateProject(context.Background(), UpdateProjectInput{
		ProjectID:   project.ID,
		Name:        "Platform",
		Description: "new desc",
		Metadata: domain.ProjectMetadata{
			Owner: "team-tillsyn",
			Tags:  []string{"go", "Go"},
		},
	})
	if err != nil {
		t.Fatalf("UpdateProject() error = %v", err)
	}
	if updated.Name != "Platform" || updated.Description != "new desc" {
		t.Fatalf("unexpected updated project %#v", updated)
	}
	if updated.Metadata.Owner != "team-tillsyn" || len(updated.Metadata.Tags) != 1 || updated.Metadata.Tags[0] != "go" {
		t.Fatalf("unexpected metadata %#v", updated.Metadata)
	}
}

// TestUpdateProjectAndCreateCommentEnqueueSecondWaveEmbeddings verifies project documents and thread-context subjects are queued on the real sqlite store.
func TestUpdateProjectAndCreateCommentEnqueueSecondWaveEmbeddings(t *testing.T) {
	now := time.Date(2026, 3, 29, 12, 30, 0, 0, time.UTC)
	svc, repo, lifecycle, project, _, actionItem := newSecondWaveEmbeddingService(t, now, func() string {
		return "comment-second-wave"
	})

	updated, err := svc.UpdateProject(context.Background(), UpdateProjectInput{
		ProjectID:   project.ID,
		Name:        "Second Wave",
		Description: "Project description refreshed for project-document indexing.",
		Metadata: domain.ProjectMetadata{
			Tags:              []string{"embeddings", "docs", "threads"},
			StandardsMarkdown: "Keep project docs and thread context searchable.",
		},
	})
	if err != nil {
		t.Fatalf("UpdateProject() error = %v", err)
	}
	comment, err := svc.CreateComment(context.Background(), CreateCommentInput{
		ProjectID:    project.ID,
		TargetType:   domain.CommentTargetTypeActionItem,
		TargetID:     actionItem.ID,
		Summary:      "Thread context note",
		BodyMarkdown: "Latency budget belongs in thread context.",
	})
	if err != nil {
		t.Fatalf("CreateComment() error = %v", err)
	}

	if len(lifecycle.inputs) != 2 {
		t.Fatalf("expected 2 lifecycle enqueue inputs, got %d", len(lifecycle.inputs))
	}
	if lifecycle.inputs[0].SubjectType != EmbeddingSubjectTypeProjectDocument {
		t.Fatalf("first enqueue subject type = %s, want project_document", lifecycle.inputs[0].SubjectType)
	}
	if lifecycle.inputs[1].SubjectType != EmbeddingSubjectTypeThreadContext {
		t.Fatalf("second enqueue subject type = %s, want thread_context", lifecycle.inputs[1].SubjectType)
	}
	projectRow := lifecycle.enqueues[lifecycle.embeddingKey(EmbeddingSubjectTypeProjectDocument, project.ID)]
	if projectRow.SubjectID != project.ID {
		t.Fatalf("project_document subject_id = %q, want %q", projectRow.SubjectID, project.ID)
	}
	if projectRow.Status != EmbeddingLifecyclePending {
		t.Fatalf("project_document status = %s, want pending", projectRow.Status)
	}
	if projectRow.ContentHashDesired != hashEmbeddingContent(buildProjectDocumentEmbeddingContent(updated)) {
		t.Fatalf("project_document content hash = %q, want %q", projectRow.ContentHashDesired, hashEmbeddingContent(buildProjectDocumentEmbeddingContent(updated)))
	}
	threadRow := lifecycle.enqueues[lifecycle.embeddingKey(EmbeddingSubjectTypeThreadContext, BuildThreadContextSubjectID(domain.CommentTarget{
		ProjectID:  project.ID,
		TargetType: domain.CommentTargetTypeActionItem,
		TargetID:   actionItem.ID,
	}))]
	wantThreadID := BuildThreadContextSubjectID(domain.CommentTarget{
		ProjectID:  project.ID,
		TargetType: domain.CommentTargetTypeActionItem,
		TargetID:   actionItem.ID,
	})
	if threadRow.SubjectID != wantThreadID {
		t.Fatalf("thread_context subject_id = %q, want %q", threadRow.SubjectID, wantThreadID)
	}
	if threadRow.Status != EmbeddingLifecyclePending {
		t.Fatalf("thread_context status = %s, want pending", threadRow.Status)
	}
	comments, err := repo.ListCommentsByTarget(context.Background(), domain.CommentTarget{
		ProjectID:  project.ID,
		TargetType: domain.CommentTargetTypeActionItem,
		TargetID:   actionItem.ID,
	})
	if err != nil {
		t.Fatalf("ListCommentsByTarget() error = %v", err)
	}
	wantThreadContent := buildThreadContextEmbeddingContent(domain.CommentTarget{
		ProjectID:  project.ID,
		TargetType: domain.CommentTargetTypeActionItem,
		TargetID:   actionItem.ID,
	}, actionItem.Title, actionItem.Description, comments)
	if threadRow.ContentHashDesired != hashEmbeddingContent(wantThreadContent) {
		t.Fatalf("thread_context content hash = %q, want %q", threadRow.ContentHashDesired, hashEmbeddingContent(wantThreadContent))
	}
	if comment.BodyMarkdown == "" {
		t.Fatal("expected comment body to be persisted")
	}
	if got := len(lifecycle.enqueues); got != 2 {
		t.Fatalf("expected 2 lifecycle rows, got %d", got)
	}
}

// TestSearchActionItemMatchesSemanticUsesThreadContextDocuments verifies comment language can rank work items through thread-context documents.
func TestSearchActionItemMatchesSemanticUsesThreadContextDocuments(t *testing.T) {
	now := time.Date(2026, 3, 29, 13, 0, 0, 0, time.UTC)
	ids := []string{"comment-a", "comment-b"}
	nextID := 0
	svc, repo, lifecycle, project, column, actionItemA := newSecondWaveEmbeddingService(t, now, func() string {
		id := ids[nextID]
		nextID++
		return id
	})
	actionItemB, err := domain.NewActionItemForTest(domain.ActionItemInput{
		Kind:        domain.KindPlan,
		ID:          "t-thread-b",
		ProjectID:   project.ID,
		ColumnID:    column.ID,
		Position:    1,
		Title:       "Release checklist",
		Description: "Keep updates terse.",
		Priority:    domain.PriorityLow,
	}, now)
	if err != nil {
		t.Fatalf("NewActionItem() error = %v", err)
	}
	if err := repo.CreateActionItem(context.Background(), actionItemB); err != nil {
		t.Fatalf("CreateActionItem() error = %v", err)
	}
	svc.embeddingGenerator = mustDeterministicEmbeddingGenerator(t, 32)
	searchIndex := &fakeActionItemSearchIndex{
		searchRows: []EmbeddingSearchMatch{
			{
				SubjectType:      EmbeddingSubjectTypeThreadContext,
				SubjectID:        BuildThreadContextSubjectID(domain.CommentTarget{ProjectID: project.ID, TargetType: domain.CommentTargetTypeActionItem, TargetID: actionItemA.ID}),
				SearchTargetType: EmbeddingSearchTargetTypeWorkItem,
				SearchTargetID:   actionItemA.ID,
				Similarity:       0.93,
			},
			{
				SubjectType:      EmbeddingSubjectTypeThreadContext,
				SubjectID:        BuildThreadContextSubjectID(domain.CommentTarget{ProjectID: project.ID, TargetType: domain.CommentTargetTypeActionItem, TargetID: actionItemB.ID}),
				SearchTargetType: EmbeddingSearchTargetTypeWorkItem,
				SearchTargetID:   actionItemB.ID,
				Similarity:       0.24,
			},
		},
	}
	svc.searchIndex = searchIndex

	if _, err := svc.CreateComment(context.Background(), CreateCommentInput{
		ProjectID:    project.ID,
		TargetType:   domain.CommentTargetTypeActionItem,
		TargetID:     actionItemA.ID,
		Summary:      "Latency budget",
		BodyMarkdown: "Latency budget needs attention before launch.",
	}); err != nil {
		t.Fatalf("CreateComment(actionItem A) error = %v", err)
	}
	if _, err := svc.CreateComment(context.Background(), CreateCommentInput{
		ProjectID:    project.ID,
		TargetType:   domain.CommentTargetTypeActionItem,
		TargetID:     actionItemB.ID,
		Summary:      "Release checklist",
		BodyMarkdown: "Release checklist stays small and routine.",
	}); err != nil {
		t.Fatalf("CreateComment(actionItem B) error = %v", err)
	}
	seedReadyThreadContextEmbeddings(lifecycle, project.ID,
		domain.CommentTarget{ProjectID: project.ID, TargetType: domain.CommentTargetTypeActionItem, TargetID: actionItemA.ID},
		domain.CommentTarget{ProjectID: project.ID, TargetType: domain.CommentTargetTypeActionItem, TargetID: actionItemB.ID},
	)

	result, err := svc.SearchActionItems(context.Background(), SearchActionItemsFilter{
		ProjectID: project.ID,
		Query:     "latency budget",
		Mode:      SearchModeSemantic,
		Limit:     10,
	})
	if err != nil {
		t.Fatalf("SearchActionItems() error = %v", err)
	}
	if result.RequestedMode != SearchModeSemantic || result.EffectiveMode != SearchModeSemantic {
		t.Fatalf("search modes = requested %q effective %q, want semantic/semantic", result.RequestedMode, result.EffectiveMode)
	}
	if !result.SemanticAvailable {
		t.Fatal("expected semantic search to be available")
	}
	if len(result.Matches) != 2 {
		t.Fatalf("expected 2 semantic matches, got %d", len(result.Matches))
	}
	if result.Matches[0].ActionItem.ID != actionItemA.ID || result.Matches[1].ActionItem.ID != actionItemB.ID {
		t.Fatalf("unexpected semantic ordering %#v", []string{result.Matches[0].ActionItem.ID, result.Matches[1].ActionItem.ID})
	}
	if result.Matches[0].EmbeddingSubjectType != EmbeddingSubjectTypeThreadContext {
		t.Fatalf("top match subject type = %q, want thread_context", result.Matches[0].EmbeddingSubjectType)
	}
	if result.Matches[0].EmbeddingSubjectID != BuildThreadContextSubjectID(domain.CommentTarget{
		ProjectID:  project.ID,
		TargetType: domain.CommentTargetTypeActionItem,
		TargetID:   actionItemA.ID,
	}) {
		t.Fatalf("top match subject id = %q, want thread-context subject id for actionItem A", result.Matches[0].EmbeddingSubjectID)
	}
	if !result.Matches[0].UsedSemantic || result.Matches[0].SemanticScore <= 0 {
		t.Fatalf("expected top semantic metadata, got %#v", result.Matches[0])
	}
}

// TestReindexEmbeddingsSeedsProjectDocumentsAndCommentTargets verifies real-db reindex/backfill covers project docs and comment targets.
func TestReindexEmbeddingsSeedsProjectDocumentsAndCommentTargets(t *testing.T) {
	now := time.Date(2026, 3, 29, 13, 30, 0, 0, time.UTC)
	ids := []string{"comment-project", "comment-actionItem"}
	nextID := 0
	svc, _, lifecycle, project, _, actionItem := newSecondWaveEmbeddingService(t, now, func() string {
		id := ids[nextID]
		nextID++
		return id
	})

	if _, err := svc.CreateComment(context.Background(), CreateCommentInput{
		ProjectID:    project.ID,
		TargetType:   domain.CommentTargetTypeProject,
		TargetID:     project.ID,
		Summary:      "Project note",
		BodyMarkdown: "Project thread for docs and standards.",
	}); err != nil {
		t.Fatalf("CreateComment(project) error = %v", err)
	}
	if _, err := svc.CreateComment(context.Background(), CreateCommentInput{
		ProjectID:    project.ID,
		TargetType:   domain.CommentTargetTypeActionItem,
		TargetID:     actionItem.ID,
		Summary:      "ActionItem note",
		BodyMarkdown: "ActionItem thread for retrieval coverage.",
	}); err != nil {
		t.Fatalf("CreateComment(actionItem) error = %v", err)
	}
	lifecycle.inputs = nil

	result, err := svc.ReindexEmbeddings(context.Background(), ReindexEmbeddingsInput{
		ProjectID: project.ID,
		Wait:      false,
	})
	if err != nil {
		t.Fatalf("ReindexEmbeddings() error = %v", err)
	}
	if len(result.TargetProjects) != 1 || result.TargetProjects[0] != project.ID {
		t.Fatalf("target projects = %#v, want [%s]", result.TargetProjects, project.ID)
	}
	if result.ScannedCount != 4 || result.QueuedCount != 4 {
		t.Fatalf("reindex result = %#v, want 4 scanned and 4 queued", result)
	}

	if len(lifecycle.inputs) != 4 {
		t.Fatalf("expected 4 lifecycle enqueue inputs, got %d", len(lifecycle.inputs))
	}
	types := map[EmbeddingSubjectType]int{}
	for _, input := range lifecycle.inputs {
		types[input.SubjectType]++
	}
	if types[EmbeddingSubjectTypeProjectDocument] != 1 {
		t.Fatalf("project_document enqueues = %d, want 1", types[EmbeddingSubjectTypeProjectDocument])
	}
	if types[EmbeddingSubjectTypeWorkItem] != 1 {
		t.Fatalf("work_item enqueues = %d, want 1", types[EmbeddingSubjectTypeWorkItem])
	}
	if types[EmbeddingSubjectTypeThreadContext] != 2 {
		t.Fatalf("thread_context enqueues = %d, want 2", types[EmbeddingSubjectTypeThreadContext])
	}
	wantProjectThread := BuildThreadContextSubjectID(domain.CommentTarget{
		ProjectID:  project.ID,
		TargetType: domain.CommentTargetTypeProject,
		TargetID:   project.ID,
	})
	wantActionItemThread := BuildThreadContextSubjectID(domain.CommentTarget{
		ProjectID:  project.ID,
		TargetType: domain.CommentTargetTypeActionItem,
		TargetID:   actionItem.ID,
	})
	seenProjectThread := false
	seenActionItemThread := false
	seenProjectDocument := false
	seenWorkItem := false
	for key, record := range lifecycle.enqueues {
		if record.ProjectID != project.ID {
			t.Fatalf("unexpected project id in lifecycle row %q: %#v", key, record)
		}
		switch record.SubjectType {
		case EmbeddingSubjectTypeProjectDocument:
			seenProjectDocument = record.SubjectID == project.ID
		case EmbeddingSubjectTypeWorkItem:
			seenWorkItem = record.SubjectID == actionItem.ID
		case EmbeddingSubjectTypeThreadContext:
			switch record.SubjectID {
			case wantProjectThread:
				seenProjectThread = true
			case wantActionItemThread:
				seenActionItemThread = true
			}
		}
	}
	if !seenProjectDocument {
		t.Fatal("expected project_document lifecycle row to be present")
	}
	if !seenWorkItem {
		t.Fatal("expected work_item lifecycle row to be present")
	}
	if !seenProjectThread || !seenActionItemThread {
		t.Fatalf("expected both thread-context rows to be present (project=%v actionItem=%v)", seenProjectThread, seenActionItemThread)
	}
}

// TestCreateProjectWithMetadataCarriesActorName verifies project mutations carry display attribution to the repo boundary.
func TestCreateProjectWithMetadataCarriesActorName(t *testing.T) {
	repo := newFakeRepo()
	now := time.Date(2026, 2, 26, 9, 0, 0, 0, time.UTC)
	svc := NewService(repo, func() string { return "p1" }, func() time.Time { return now }, ServiceConfig{})

	project, err := svc.CreateProjectWithMetadata(context.Background(), CreateProjectInput{
		Name:          "Roadmap",
		Description:   "Q3 plan",
		UpdatedBy:     "user-1",
		UpdatedByName: "Evan Schultz",
		UpdatedType:   domain.ActorTypeUser,
	})
	if err != nil {
		t.Fatalf("CreateProjectWithMetadata() error = %v", err)
	}
	if project.ID != "p1" {
		t.Fatalf("unexpected project id %q", project.ID)
	}
	if repo.createProjectActor.ActorID != "user-1" {
		t.Fatalf("create project actor id = %q, want user-1", repo.createProjectActor.ActorID)
	}
	if repo.createProjectActor.ActorName != "Evan Schultz" {
		t.Fatalf("create project actor name = %q, want Evan Schultz", repo.createProjectActor.ActorName)
	}
	if repo.createProjectActor.ActorType != domain.ActorTypeUser {
		t.Fatalf("create project actor type = %q, want %q", repo.createProjectActor.ActorType, domain.ActorTypeUser)
	}
}

// TestArchiveRestoreAndDeleteProject verifies project archive, restore, and hard-delete behavior.
func TestArchiveRestoreAndDeleteProject(t *testing.T) {
	repo := newFakeRepo()
	now := time.Date(2026, 2, 24, 8, 0, 0, 0, time.UTC)
	project, _ := domain.NewProjectFromInput(domain.ProjectInput{ID: "p1", Name: "Inbox", Description: "desc"}, now)
	repo.projects[project.ID] = project

	column, _ := domain.NewColumn("c1", project.ID, "To Do", 0, 0, now)
	repo.columns[column.ID] = column
	actionItem, _ := domain.NewActionItemForTest(domain.ActionItemInput{
		Kind:      domain.KindPlan,
		ID:        "t1",
		ProjectID: project.ID,
		ColumnID:  column.ID,
		Position:  0,
		Title:     "actionItem",
		Priority:  domain.PriorityMedium,
	}, now)
	repo.tasks[actionItem.ID] = actionItem

	svc := NewService(repo, nil, func() time.Time { return now.Add(time.Minute) }, ServiceConfig{})

	archived, err := svc.ArchiveProject(context.Background(), project.ID)
	if err != nil {
		t.Fatalf("ArchiveProject() error = %v", err)
	}
	if archived.ArchivedAt == nil {
		t.Fatal("expected project archived_at to be set")
	}

	active, err := svc.ListProjects(context.Background(), false)
	if err != nil {
		t.Fatalf("ListProjects(active) error = %v", err)
	}
	if len(active) != 0 {
		t.Fatalf("expected no active projects after archive, got %d", len(active))
	}

	restored, err := svc.RestoreProject(context.Background(), project.ID)
	if err != nil {
		t.Fatalf("RestoreProject() error = %v", err)
	}
	if restored.ArchivedAt != nil {
		t.Fatal("expected project archived_at cleared after restore")
	}

	if err := svc.DeleteProject(context.Background(), project.ID); err != nil {
		t.Fatalf("DeleteProject() error = %v", err)
	}
	if _, err := repo.GetProject(context.Background(), project.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected deleted project not found, got %v", err)
	}
	if _, ok := repo.columns[column.ID]; ok {
		t.Fatal("expected project columns deleted with project")
	}
	if _, ok := repo.tasks[actionItem.ID]; ok {
		t.Fatal("expected project tasks deleted with project")
	}
}

// TestStateTemplateSanitization verifies behavior for the covered scenario.
func TestStateTemplateSanitization(t *testing.T) {
	got := sanitizeStateTemplates([]StateTemplate{
		{ID: "", Name: " To Do ", Position: 3},
		{ID: "todo", Name: "Duplicate", Position: 1},
		{ID: "", Name: "In Progress", Position: 2, WIPLimit: -1},
		{ID: "", Name: " ", Position: 4},
	})
	if len(got) != 2 {
		t.Fatalf("expected 2 sanitized states, got %#v", got)
	}
	if got[0].ID != "in_progress" || got[1].ID != "todo" {
		t.Fatalf("unexpected sanitized IDs %#v", got)
	}
	if got[0].WIPLimit != 0 {
		t.Fatalf("expected clamped wip limit, got %d", got[0].WIPLimit)
	}
}

// TestNormalizeStateIDStrictCanonicalRejectsLegacyLiterals verifies the
// strict-canonical contract: legacy state literals are rejected via empty-string
// return rather than slug-passthrough. Round 1 of Droplet 2.7 left them
// passing through to themselves (e.g. "done" → "done") which gave callers a
// false-positive "valid slug" signal; PLAN.md acceptance line 222 mandates
// the unknown-state error path.
func TestNormalizeStateIDStrictCanonicalRejectsLegacyLiterals(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{name: "canonical todo", in: "todo", want: "todo"},
		{name: "canonical in_progress", in: "in_progress", want: "in_progress"},
		{name: "canonical complete", in: "complete", want: "complete"},
		{name: "canonical failed", in: "failed", want: "failed"},
		{name: "canonical archived", in: "archived", want: "archived"},
		{name: "kebab to-do is canonical (not legacy)", in: "to-do", want: "todo"},
		{name: "display In Progress slugs canonical", in: "In Progress", want: "in_progress"},
		{name: "display Complete slugs canonical", in: "Complete", want: "complete"},
		{name: "legacy done rejected", in: "done", want: ""},
		{name: "legacy completed rejected", in: "completed", want: ""},
		{name: "legacy progress rejected", in: "progress", want: ""},
		{name: "legacy doing rejected", in: "doing", want: ""},
		{name: "legacy in-progress rejected", in: "in-progress", want: ""},
		{name: "legacy uppercase Done rejected", in: "Done", want: ""},
		{name: "legacy with surrounding whitespace rejected", in: "  progress  ", want: ""},
		{name: "custom column name preserved", in: "My Custom Column", want: "my_custom_column"},
		{name: "empty stays empty", in: "", want: ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := normalizeStateID(tc.in); got != tc.want {
				t.Fatalf("normalizeStateID(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

// failingRepo represents failing repo data used by this package.
type failingRepo struct {
	*fakeRepo
	err error
}

// ListProjects lists projects.
func (f failingRepo) ListProjects(context.Context, bool) ([]domain.Project, error) {
	return nil, f.err
}

// TestEnsureDefaultProjectErrorPropagation verifies behavior for the covered scenario.
func TestEnsureDefaultProjectErrorPropagation(t *testing.T) {
	expected := errors.New("boom")
	svc := NewService(failingRepo{fakeRepo: newFakeRepo(), err: expected}, nil, time.Now, ServiceConfig{})
	_, err := svc.EnsureDefaultProject(context.Background())
	if !errors.Is(err, expected) {
		t.Fatalf("expected wrapped error %v, got %v", expected, err)
	}
}

// TestMoveActionItemBlocksWhenStartCriteriaUnmet verifies behavior for the covered scenario.
func TestMoveActionItemBlocksWhenStartCriteriaUnmet(t *testing.T) {
	repo := newFakeRepo()
	now := time.Date(2026, 2, 21, 12, 0, 0, 0, time.UTC)
	project, _ := domain.NewProjectFromInput(domain.ProjectInput{ID: "p1", Name: "Inbox"}, now)
	repo.projects[project.ID] = project
	todo, _ := domain.NewColumn("c1", project.ID, "To Do", 0, 0, now)
	progress, _ := domain.NewColumn("c2", project.ID, "In Progress", 1, 0, now)
	repo.columns[todo.ID] = todo
	repo.columns[progress.ID] = progress

	actionItem, _ := domain.NewActionItemForTest(domain.ActionItemInput{
		Kind:      domain.KindPlan,
		ID:        "t1",
		ProjectID: project.ID,
		ColumnID:  todo.ID,
		Position:  0,
		Title:     "blocked",
		Priority:  domain.PriorityMedium,
		Metadata: domain.ActionItemMetadata{
			CompletionContract: domain.CompletionContract{
				StartCriteria: []domain.ChecklistItem{{ID: "s1", Text: "design reviewed", Complete: false}},
			},
		},
	}, now)
	repo.tasks[actionItem.ID] = actionItem

	svc := NewService(repo, nil, func() time.Time { return now }, ServiceConfig{})
	_, err := svc.MoveActionItem(context.Background(), actionItem.ID, progress.ID, 0)
	if err == nil || !errors.Is(err, domain.ErrTransitionBlocked) {
		t.Fatalf("expected ErrTransitionBlocked, got %v", err)
	}
}

// TestMoveActionItemAllowsDoneWhenContractsSatisfied verifies behavior for the covered scenario.
func TestMoveActionItemAllowsDoneWhenContractsSatisfied(t *testing.T) {
	repo := newFakeRepo()
	now := time.Date(2026, 2, 21, 12, 0, 0, 0, time.UTC)
	project, _ := domain.NewProjectFromInput(domain.ProjectInput{ID: "p1", Name: "Inbox"}, now)
	repo.projects[project.ID] = project
	progress, _ := domain.NewColumn("c2", project.ID, "In Progress", 1, 0, now)
	done, _ := domain.NewColumn("c3", project.ID, "Complete", 2, 0, now)
	repo.columns[progress.ID] = progress
	repo.columns[done.ID] = done

	parent, _ := domain.NewActionItemForTest(domain.ActionItemInput{
		Kind:           domain.KindPlan,
		ID:             "t-parent",
		ProjectID:      project.ID,
		ColumnID:       progress.ID,
		Position:       0,
		Title:          "parent",
		Priority:       domain.PriorityHigh,
		LifecycleState: domain.StateInProgress,
		Metadata: domain.ActionItemMetadata{
			CompletionContract: domain.CompletionContract{
				CompletionCriteria: []domain.ChecklistItem{{ID: "c1", Text: "tests green", Complete: true}},
				CompletionChecklist: []domain.ChecklistItem{
					{ID: "k1", Text: "docs updated", Complete: true},
				},
			},
		},
	}, now)
	child, _ := domain.NewActionItemForTest(domain.ActionItemInput{
		Kind:           domain.KindPlan,
		ID:             "t-child",
		ProjectID:      project.ID,
		ParentID:       parent.ID,
		ColumnID:       done.ID,
		Position:       0,
		Title:          "child",
		Priority:       domain.PriorityLow,
		LifecycleState: domain.StateComplete,
	}, now)
	repo.tasks[parent.ID] = parent
	repo.tasks[child.ID] = child

	svc := NewService(repo, nil, func() time.Time { return now.Add(time.Minute) }, ServiceConfig{})
	moved, err := svc.MoveActionItem(context.Background(), parent.ID, done.ID, 0)
	if err != nil {
		t.Fatalf("MoveActionItem() error = %v", err)
	}
	if moved.LifecycleState != domain.StateComplete {
		t.Fatalf("expected complete lifecycle state, got %q", moved.LifecycleState)
	}
	if moved.CompletedAt == nil {
		t.Fatal("expected completed_at to be set")
	}
}

// TestMoveActionItemBlocksDoneWhenChildIncomplete verifies the always-on
// parent-blocks-on-incomplete-child invariant (Drop 4a Wave 1.7): a parent
// with a non-archived non-Complete child cannot move to StateComplete.
func TestMoveActionItemBlocksDoneWhenChildIncomplete(t *testing.T) {
	repo := newFakeRepo()
	now := time.Date(2026, 2, 21, 12, 0, 0, 0, time.UTC)
	project, _ := domain.NewProjectFromInput(domain.ProjectInput{ID: "p1", Name: "Inbox"}, now)
	repo.projects[project.ID] = project
	progress, _ := domain.NewColumn("c2", project.ID, "In Progress", 1, 0, now)
	done, _ := domain.NewColumn("c3", project.ID, "Complete", 2, 0, now)
	repo.columns[progress.ID] = progress
	repo.columns[done.ID] = done

	parent, _ := domain.NewActionItemForTest(domain.ActionItemInput{
		Kind:           domain.KindPlan,
		ID:             "t-parent",
		ProjectID:      project.ID,
		ColumnID:       progress.ID,
		Position:       0,
		Title:          "parent",
		Priority:       domain.PriorityHigh,
		LifecycleState: domain.StateInProgress,
	}, now)
	child, _ := domain.NewActionItemForTest(domain.ActionItemInput{
		Kind:           domain.KindPlan,
		ID:             "t-child",
		ProjectID:      project.ID,
		ParentID:       parent.ID,
		ColumnID:       progress.ID,
		Position:       1,
		Title:          "child",
		Priority:       domain.PriorityLow,
		LifecycleState: domain.StateInProgress,
	}, now)
	repo.tasks[parent.ID] = parent
	repo.tasks[child.ID] = child

	svc := NewService(repo, nil, func() time.Time { return now.Add(time.Minute) }, ServiceConfig{})
	_, err := svc.MoveActionItem(context.Background(), parent.ID, done.ID, 0)
	if err == nil || !errors.Is(err, domain.ErrTransitionBlocked) {
		t.Fatalf("expected ErrTransitionBlocked, got %v", err)
	}
	if !strings.Contains(err.Error(), "child item") {
		t.Fatalf("expected incomplete subtask reason, got %v", err)
	}
}

// TestReparentActionItemAndListChildActionItems verifies behavior for the covered scenario.
func TestReparentActionItemAndListChildActionItems(t *testing.T) {
	repo := newFakeRepo()
	now := time.Date(2026, 2, 21, 12, 0, 0, 0, time.UTC)
	project, _ := domain.NewProjectFromInput(domain.ProjectInput{ID: "p1", Name: "Inbox"}, now)
	repo.projects[project.ID] = project
	column, _ := domain.NewColumn("c1", project.ID, "To Do", 0, 0, now)
	repo.columns[column.ID] = column
	parent, _ := domain.NewActionItemForTest(domain.ActionItemInput{
		Kind:      domain.KindPlan,
		ID:        "parent",
		ProjectID: project.ID,
		ColumnID:  column.ID,
		Position:  0,
		Title:     "parent",
		Priority:  domain.PriorityMedium,
	}, now)
	child, _ := domain.NewActionItemForTest(domain.ActionItemInput{
		Kind:      domain.KindPlan,
		ID:        "child",
		ProjectID: project.ID,
		ColumnID:  column.ID,
		Position:  1,
		Title:     "child",
		Priority:  domain.PriorityLow,
	}, now)
	repo.tasks[parent.ID] = parent
	repo.tasks[child.ID] = child

	svc := NewService(repo, nil, func() time.Time { return now.Add(2 * time.Minute) }, ServiceConfig{})
	updated, err := svc.ReparentActionItem(context.Background(), child.ID, parent.ID)
	if err != nil {
		t.Fatalf("ReparentActionItem() error = %v", err)
	}
	if updated.ParentID != parent.ID {
		t.Fatalf("expected parent id %q, got %q", parent.ID, updated.ParentID)
	}
	children, err := svc.ListChildActionItems(context.Background(), project.ID, parent.ID, false)
	if err != nil {
		t.Fatalf("ListChildActionItems() error = %v", err)
	}
	if len(children) != 1 || children[0].ID != child.ID {
		t.Fatalf("unexpected child list %#v", children)
	}
}

// TestGetProjectDependencyRollup verifies behavior for the covered scenario.
func TestGetProjectDependencyRollup(t *testing.T) {
	repo := newFakeRepo()
	now := time.Date(2026, 2, 21, 12, 0, 0, 0, time.UTC)
	project, _ := domain.NewProjectFromInput(domain.ProjectInput{ID: "p1", Name: "Inbox"}, now)
	repo.projects[project.ID] = project
	column, _ := domain.NewColumn("c1", project.ID, "To Do", 0, 0, now)
	repo.columns[column.ID] = column

	readyDep, _ := domain.NewActionItemForTest(domain.ActionItemInput{
		Kind:           domain.KindPlan,
		ID:             "dep-ready",
		ProjectID:      project.ID,
		ColumnID:       column.ID,
		Position:       0,
		Title:          "ready dep",
		Priority:       domain.PriorityLow,
		LifecycleState: domain.StateComplete,
	}, now)
	openDep, _ := domain.NewActionItemForTest(domain.ActionItemInput{
		Kind:           domain.KindPlan,
		ID:             "dep-open",
		ProjectID:      project.ID,
		ColumnID:       column.ID,
		Position:       1,
		Title:          "open dep",
		Priority:       domain.PriorityLow,
		LifecycleState: domain.StateInProgress,
	}, now)
	blocked, _ := domain.NewActionItemForTest(domain.ActionItemInput{
		Kind:      domain.KindPlan,
		ID:        "blocked",
		ProjectID: project.ID,
		ColumnID:  column.ID,
		Position:  2,
		Title:     "blocked",
		Priority:  domain.PriorityMedium,
		Metadata: domain.ActionItemMetadata{
			DependsOn:     []string{"dep-ready", "dep-open", "dep-missing"},
			BlockedBy:     []string{"dep-open"},
			BlockedReason: "waiting on review",
		},
	}, now)

	repo.tasks[readyDep.ID] = readyDep
	repo.tasks[openDep.ID] = openDep
	repo.tasks[blocked.ID] = blocked

	svc := NewService(repo, nil, func() time.Time { return now }, ServiceConfig{})
	rollup, err := svc.GetProjectDependencyRollup(context.Background(), project.ID)
	if err != nil {
		t.Fatalf("GetProjectDependencyRollup() error = %v", err)
	}
	if rollup.TotalItems != 3 {
		t.Fatalf("expected 3 total items, got %d", rollup.TotalItems)
	}
	if rollup.ItemsWithDependencies != 1 || rollup.DependencyEdges != 3 {
		t.Fatalf("unexpected dependency counts %#v", rollup)
	}
	if rollup.BlockedItems != 1 || rollup.BlockedByEdges != 1 {
		t.Fatalf("unexpected blocked counts %#v", rollup)
	}
	if rollup.UnresolvedDependencyEdges != 2 {
		t.Fatalf("expected 2 unresolved dependencies, got %d", rollup.UnresolvedDependencyEdges)
	}
}

// TestListProjectChangeEvents verifies behavior for the covered scenario.
func TestListProjectChangeEvents(t *testing.T) {
	repo := newFakeRepo()
	now := time.Date(2026, 2, 21, 12, 0, 0, 0, time.UTC)
	project, _ := domain.NewProjectFromInput(domain.ProjectInput{ID: "p1", Name: "Inbox"}, now)
	repo.projects[project.ID] = project
	repo.changeEvents[project.ID] = []domain.ChangeEvent{
		{ID: 3, ProjectID: project.ID, ActionItemID: "t1", Operation: domain.ChangeOperationUpdate},
		{ID: 2, ProjectID: project.ID, ActionItemID: "t1", Operation: domain.ChangeOperationCreate},
	}

	svc := NewService(repo, nil, func() time.Time { return now }, ServiceConfig{})
	events, err := svc.ListProjectChangeEvents(context.Background(), project.ID, 1)
	if err != nil {
		t.Fatalf("ListProjectChangeEvents() error = %v", err)
	}
	if len(events) != 1 || events[0].Operation != domain.ChangeOperationUpdate {
		t.Fatalf("unexpected events %#v", events)
	}
}

// TestCreateAndListCommentsByTarget verifies behavior for the covered scenario.
func TestCreateAndListCommentsByTarget(t *testing.T) {
	repo := newFakeRepo()
	now := time.Date(2026, 2, 23, 9, 0, 0, 0, time.UTC)
	ids := []string{"comment-2", "comment-1"}
	nextID := 0
	project, _ := domain.NewProjectFromInput(domain.ProjectInput{ID: "p1", Name: "Inbox"}, now)
	repo.projects[project.ID] = project
	actionItem, _ := domain.NewActionItemForTest(domain.ActionItemInput{
		Kind:      domain.KindPlan,
		ID:        "t1",
		ProjectID: project.ID,
		ColumnID:  "c1",
		Position:  0,
		Title:     "ActionItem",
		Priority:  domain.PriorityLow,
	}, now)
	repo.tasks[actionItem.ID] = actionItem

	svc := NewService(repo, func() string {
		id := ids[nextID]
		nextID++
		return id
	}, func() time.Time {
		// Fixed clock intentionally forces tie timestamps so ID ordering is tested.
		return now
	}, ServiceConfig{})

	first, err := svc.CreateComment(context.Background(), CreateCommentInput{
		ProjectID:    project.ID,
		TargetType:   domain.CommentTargetTypeActionItem,
		TargetID:     actionItem.ID,
		Summary:      "  explicit summary  ",
		BodyMarkdown: "first",
		ActorType:    domain.ActorType("USER"),
		ActorID:      "user-1",
		ActorName:    "user-1",
	})
	if err != nil {
		t.Fatalf("CreateComment(first) error = %v", err)
	}
	second, err := svc.CreateComment(context.Background(), CreateCommentInput{
		ProjectID:    project.ID,
		TargetType:   domain.CommentTargetTypeActionItem,
		TargetID:     actionItem.ID,
		BodyMarkdown: "\n\nsecond",
	})
	if err != nil {
		t.Fatalf("CreateComment(second) error = %v", err)
	}
	if first.Summary != "explicit summary" {
		t.Fatalf("expected explicit summary normalization, got %q", first.Summary)
	}
	if second.Summary != "second" {
		t.Fatalf("expected summary fallback from body markdown, got %q", second.Summary)
	}
	if second.ActorType != domain.ActorTypeUser {
		t.Fatalf("expected default actor type user, got %q", second.ActorType)
	}
	if second.ActorID != "tillsyn-user" {
		t.Fatalf("expected default actor id tillsyn-user, got %q", second.ActorID)
	}
	if second.ActorName != "tillsyn-user" {
		t.Fatalf("expected default actor name tillsyn-user, got %q", second.ActorName)
	}

	comments, err := svc.ListCommentsByTarget(context.Background(), ListCommentsByTargetInput{
		ProjectID:  project.ID,
		TargetType: domain.CommentTargetTypeActionItem,
		TargetID:   actionItem.ID,
	})
	if err != nil {
		t.Fatalf("ListCommentsByTarget() error = %v", err)
	}
	if len(comments) != 2 {
		t.Fatalf("expected 2 comments, got %d", len(comments))
	}
	if comments[0].ID != "comment-1" || comments[1].ID != "comment-2" {
		t.Fatalf("expected deterministic id ordering on equal timestamps, got %#v", comments)
	}
}

// TestListCommentsByTargetWaitsForLiveChange verifies comment list wait_timeout resumes on the next thread update.
func TestListCommentsByTargetWaitsForLiveChange(t *testing.T) {
	repo := newFakeRepo()
	now := time.Date(2026, 4, 2, 10, 30, 0, 0, time.UTC)
	project, _ := domain.NewProjectFromInput(domain.ProjectInput{ID: "p1", Name: "Inbox"}, now)
	repo.projects[project.ID] = project

	svc := NewService(repo, func() string { return "comment-live" }, func() time.Time { return now }, ServiceConfig{})
	resultCh := make(chan []domain.Comment, 1)
	errCh := make(chan error, 1)
	go func() {
		items, listErr := svc.ListCommentsByTarget(context.Background(), ListCommentsByTargetInput{
			ProjectID:   project.ID,
			TargetType:  domain.CommentTargetTypeProject,
			TargetID:    project.ID,
			WaitTimeout: time.Second,
		})
		if listErr != nil {
			errCh <- listErr
			return
		}
		resultCh <- items
	}()

	select {
	case got := <-resultCh:
		t.Fatalf("ListCommentsByTarget() returned early with %#v before a live change", got)
	case err := <-errCh:
		t.Fatalf("ListCommentsByTarget() early error = %v", err)
	case <-time.After(25 * time.Millisecond):
	}

	if _, err := svc.CreateComment(context.Background(), CreateCommentInput{
		ProjectID:    project.ID,
		TargetType:   domain.CommentTargetTypeProject,
		TargetID:     project.ID,
		Summary:      "Project kickoff",
		BodyMarkdown: "First live thread update",
		ActorType:    domain.ActorTypeUser,
		ActorID:      "user-1",
		ActorName:    "user-1",
	}); err != nil {
		t.Fatalf("CreateComment() error = %v", err)
	}

	select {
	case err := <-errCh:
		t.Fatalf("ListCommentsByTarget() error = %v", err)
	case items := <-resultCh:
		if len(items) != 1 || items[0].ID != "comment-live" {
			t.Fatalf("ListCommentsByTarget() = %#v, want comment-live after wake", items)
		}
	case <-time.After(time.Second):
		t.Fatal("ListCommentsByTarget() did not wake after a live comment change")
	}
}

// TestCreateCommentUsesContextActorNameFallback verifies comment mutations reuse the context display name for matching actors.
func TestCreateCommentUsesContextActorNameFallback(t *testing.T) {
	repo := newFakeRepo()
	now := time.Date(2026, 2, 26, 11, 0, 0, 0, time.UTC)
	project, _ := domain.NewProjectFromInput(domain.ProjectInput{ID: "p1", Name: "Inbox"}, now)
	repo.projects[project.ID] = project
	actionItem, _ := domain.NewActionItemForTest(domain.ActionItemInput{
		Kind:      domain.KindPlan,
		ID:        "t1",
		ProjectID: project.ID,
		ColumnID:  "c1",
		Position:  0,
		Title:     "ActionItem",
		Priority:  domain.PriorityLow,
	}, now)
	repo.tasks[actionItem.ID] = actionItem

	svc := NewService(repo, func() string { return "comment-1" }, func() time.Time { return now }, ServiceConfig{})
	ctx := WithMutationActor(context.Background(), MutationActor{
		ActorID:   "user-1",
		ActorName: "Evan Schultz",
		ActorType: domain.ActorTypeUser,
	})
	comment, err := svc.CreateComment(ctx, CreateCommentInput{
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
	if repo.createCommentActor.ActorName != "Evan Schultz" {
		t.Fatalf("repo comment actor name = %q, want Evan Schultz", repo.createCommentActor.ActorName)
	}
}

// TestCreateCommentCreatesMentionInboxAttention verifies routed @mentions materialize scoped inbox attention rows.
func TestCreateCommentCreatesMentionInboxAttention(t *testing.T) {
	repo := newFakeRepo()
	now := time.Date(2026, 2, 26, 11, 30, 0, 0, time.UTC)
	project, _ := domain.NewProjectFromInput(domain.ProjectInput{ID: "p1", Name: "Inbox"}, now)
	repo.projects[project.ID] = project
	actionItem, _ := domain.NewActionItemForTest(domain.ActionItemInput{
		Kind:      domain.KindPlan,
		ID:        "t1",
		ProjectID: project.ID,
		ColumnID:  "c1",
		Position:  0,
		Title:     "ActionItem",
		Priority:  domain.PriorityLow,
	}, now)
	repo.tasks[actionItem.ID] = actionItem

	svc := NewService(repo, func() string { return "comment-1" }, func() time.Time { return now }, ServiceConfig{})
	comment, err := svc.CreateComment(context.Background(), CreateCommentInput{
		ProjectID:    project.ID,
		TargetType:   domain.CommentTargetTypeActionItem,
		TargetID:     actionItem.ID,
		Summary:      "Need review from @dev and @qa",
		BodyMarkdown: "Please check this branch, @qa. @dev already has context.",
		ActorID:      "user-1",
		ActorName:    "Evan Schultz",
		ActorType:    domain.ActorTypeUser,
	})
	if err != nil {
		t.Fatalf("CreateComment() error = %v", err)
	}

	builderMention, ok := repo.attentionItems[comment.ID+"::mention::builder"]
	if !ok {
		t.Fatalf("expected builder mention attention item, got %#v", repo.attentionItems)
	}
	if builderMention.Kind != domain.AttentionKindMention || builderMention.TargetRole != "builder" {
		t.Fatalf("unexpected builder mention %#v", builderMention)
	}
	if builderMention.ScopeType != domain.ScopeLevelActionItem || builderMention.ScopeID != actionItem.ID {
		t.Fatalf("expected actionItem-scoped builder mention, got %#v", builderMention)
	}

	qaMention, ok := repo.attentionItems[comment.ID+"::mention::qa"]
	if !ok {
		t.Fatalf("expected qa mention attention item, got %#v", repo.attentionItems)
	}
	if qaMention.Kind != domain.AttentionKindMention || qaMention.TargetRole != "qa" {
		t.Fatalf("unexpected qa mention %#v", qaMention)
	}
	if len(repo.attentionItems) != 2 {
		t.Fatalf("expected exactly two routed mention attention rows, got %#v", repo.attentionItems)
	}
}

// TestCreateCommentValidation verifies behavior for the covered scenario.
func TestCreateCommentValidation(t *testing.T) {
	repo := newFakeRepo()
	now := time.Date(2026, 2, 23, 9, 0, 0, 0, time.UTC)
	project, _ := domain.NewProjectFromInput(domain.ProjectInput{ID: "p1", Name: "Inbox"}, now)
	repo.projects[project.ID] = project
	actionItem, _ := domain.NewActionItemForTest(domain.ActionItemInput{
		Kind:      domain.KindPlan,
		ID:        "t1",
		ProjectID: project.ID,
		ColumnID:  "c1",
		Position:  0,
		Title:     "ActionItem",
		Priority:  domain.PriorityLow,
	}, now)
	repo.tasks[actionItem.ID] = actionItem
	svc := NewService(repo, func() string { return "comment-1" }, time.Now, ServiceConfig{})

	_, err := svc.CreateComment(context.Background(), CreateCommentInput{
		ProjectID:    "",
		TargetType:   domain.CommentTargetTypeProject,
		TargetID:     "p1",
		BodyMarkdown: "body",
	})
	if err != domain.ErrInvalidID {
		t.Fatalf("expected ErrInvalidID for missing project id, got %v", err)
	}

	_, err = svc.CreateComment(context.Background(), CreateCommentInput{
		ProjectID:    project.ID,
		TargetType:   domain.CommentTargetTypeActionItem,
		TargetID:     actionItem.ID,
		BodyMarkdown: " ",
	})
	if err != domain.ErrInvalidBodyMarkdown {
		t.Fatalf("expected ErrInvalidBodyMarkdown, got %v", err)
	}
	_, err = svc.CreateComment(context.Background(), CreateCommentInput{
		ProjectID:    project.ID,
		TargetType:   domain.CommentTargetTypeActionItem,
		TargetID:     "missing-actionItem",
		BodyMarkdown: "body",
	})
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound for unknown target, got %v", err)
	}

	_, err = svc.ListCommentsByTarget(context.Background(), ListCommentsByTargetInput{
		ProjectID:  project.ID,
		TargetType: domain.CommentTargetType("invalid"),
		TargetID:   actionItem.ID,
	})
	if err != domain.ErrInvalidTargetType {
		t.Fatalf("expected ErrInvalidTargetType, got %v", err)
	}
}

// TestSnapshotCommentTargetTypeForActionItemAlwaysActionItem verifies every
// kind in the 12-value enum maps to CommentTargetTypeActionItem. Branch /
// phase / subtask comment target types were removed alongside the
// scope-mirrors-kind collapse.
func TestSnapshotCommentTargetTypeForActionItemAlwaysActionItem(t *testing.T) {
	tests := []struct {
		name       string
		actionItem domain.ActionItem
	}{
		{name: "plan", actionItem: domain.ActionItem{Kind: domain.KindPlan, Scope: domain.KindAppliesToPlan}},
		{name: "build", actionItem: domain.ActionItem{Kind: domain.KindBuild, Scope: domain.KindAppliesToBuild}},
		{name: "research", actionItem: domain.ActionItem{Kind: domain.KindResearch, Scope: domain.KindAppliesToResearch}},
		{name: "build-qa-proof", actionItem: domain.ActionItem{Kind: domain.KindBuildQAProof, Scope: domain.KindAppliesToBuildQAProof}},
	}

	for _, tc := range tests {
		got := snapshotCommentTargetTypeForActionItem(tc.actionItem)
		if got != domain.CommentTargetTypeActionItem {
			t.Fatalf("%s: snapshotCommentTargetTypeForActionItem() = %q, want %q", tc.name, got, domain.CommentTargetTypeActionItem)
		}
	}
}

// TestIssueCapabilityLeaseOverlapPolicy verifies same-identity orchestrator overlap
// behavior and override token handling. All four sub-cases use the same AgentName so
// the overlap gate at kind_capability.go:ensureOrchestratorOverlapPolicy is exercised
// on the same-identity lane (distinct AgentInstanceIDs keep the short-circuit at the
// top of the loop from suppressing the check).
func TestIssueCapabilityLeaseOverlapPolicy(t *testing.T) {
	repo := newFakeRepo()
	ids := []string{"p1", "lease-a", "lease-token-a", "lease-b", "lease-token-b", "lease-c", "lease-token-c", "lease-d", "lease-token-d"}
	idx := 0
	svc := NewService(repo, func() string {
		id := ids[idx]
		idx++
		return id
	}, func() time.Time {
		return time.Date(2026, 2, 24, 10, 0, 0, 0, time.UTC)
	}, ServiceConfig{DefaultDeleteMode: DeleteModeArchive})

	project, err := svc.CreateProjectWithMetadata(context.Background(), CreateProjectInput{
		Name:        "Lease Policy",
		Description: "",
		Metadata: domain.ProjectMetadata{
			CapabilityPolicy: domain.ProjectCapabilityPolicy{
				AllowOrchestratorOverride: true,
				OrchestratorOverrideToken: "override-123",
			},
		},
	})
	if err != nil {
		t.Fatalf("CreateProjectWithMetadata() error = %v", err)
	}

	if _, err := svc.IssueCapabilityLease(context.Background(), IssueCapabilityLeaseInput{
		ProjectID:       project.ID,
		ScopeType:       domain.CapabilityScopeProject,
		Role:            domain.CapabilityRoleOrchestrator,
		AgentName:       "orch-alpha",
		AgentInstanceID: "orch-alpha-1",
	}); err != nil {
		t.Fatalf("IssueCapabilityLease(first) error = %v", err)
	}

	if _, err := svc.IssueCapabilityLease(context.Background(), IssueCapabilityLeaseInput{
		ProjectID:       project.ID,
		ScopeType:       domain.CapabilityScopeProject,
		Role:            domain.CapabilityRoleOrchestrator,
		AgentName:       "orch-alpha",
		AgentInstanceID: "orch-alpha-2",
	}); err != domain.ErrOverrideTokenRequired {
		t.Fatalf("expected ErrOverrideTokenRequired, got %v", err)
	}

	if _, err := svc.IssueCapabilityLease(context.Background(), IssueCapabilityLeaseInput{
		ProjectID:       project.ID,
		ScopeType:       domain.CapabilityScopeProject,
		Role:            domain.CapabilityRoleOrchestrator,
		AgentName:       "orch-alpha",
		AgentInstanceID: "orch-alpha-3",
		OverrideToken:   "wrong",
	}); err != domain.ErrOverrideTokenInvalid {
		t.Fatalf("expected ErrOverrideTokenInvalid, got %v", err)
	}

	if _, err := svc.IssueCapabilityLease(context.Background(), IssueCapabilityLeaseInput{
		ProjectID:       project.ID,
		ScopeType:       domain.CapabilityScopeProject,
		Role:            domain.CapabilityRoleOrchestrator,
		AgentName:       "orch-alpha",
		AgentInstanceID: "orch-alpha-4",
		OverrideToken:   "override-123",
	}); err != nil {
		t.Fatalf("IssueCapabilityLease(override) error = %v", err)
	}
}

// TestIssueCapabilityLeaseAllowsDistinctOrchestratorIdentities verifies two orchestrator
// leases with different AgentName values coexist at the same scope without an override
// token. Cements acceptance 2.1 of the multi-orch unblock.
func TestIssueCapabilityLeaseAllowsDistinctOrchestratorIdentities(t *testing.T) {
	repo := newFakeRepo()
	ids := []string{"p1", "lease-token-a", "lease-token-b"}
	idx := 0
	svc := NewService(repo, func() string {
		id := ids[idx]
		idx++
		return id
	}, func() time.Time {
		return time.Date(2026, 2, 24, 10, 0, 0, 0, time.UTC)
	}, ServiceConfig{DefaultDeleteMode: DeleteModeArchive})

	project, err := svc.CreateProject(context.Background(), "Distinct Identities", "")
	if err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}

	if _, err := svc.IssueCapabilityLease(context.Background(), IssueCapabilityLeaseInput{
		ProjectID:       project.ID,
		ScopeType:       domain.CapabilityScopeProject,
		Role:            domain.CapabilityRoleOrchestrator,
		AgentName:       "orch-a",
		AgentInstanceID: "orch-a-inst",
	}); err != nil {
		t.Fatalf("IssueCapabilityLease(orch-a) error = %v", err)
	}

	if _, err := svc.IssueCapabilityLease(context.Background(), IssueCapabilityLeaseInput{
		ProjectID:       project.ID,
		ScopeType:       domain.CapabilityScopeProject,
		Role:            domain.CapabilityRoleOrchestrator,
		AgentName:       "orch-b",
		AgentInstanceID: "orch-b-inst",
	}); err != nil {
		t.Fatalf("IssueCapabilityLease(orch-b) error = %v", err)
	}

	leases, err := svc.ListCapabilityLeases(context.Background(), ListCapabilityLeasesInput{
		ProjectID: project.ID,
		ScopeType: domain.CapabilityScopeProject,
	})
	if err != nil {
		t.Fatalf("ListCapabilityLeases() error = %v", err)
	}
	if len(leases) != 2 {
		t.Fatalf("ListCapabilityLeases() len = %d, want 2", len(leases))
	}
	names := map[string]bool{}
	for _, lease := range leases {
		names[lease.AgentName] = true
	}
	if !names["orch-a"] || !names["orch-b"] {
		t.Fatalf("ListCapabilityLeases() AgentNames = %v, want both orch-a and orch-b", names)
	}
}

// TestIssueCapabilityLeaseRejectsSameIdentityReclaim verifies a second orchestrator lease
// issued by the same AgentName at the same scope is rejected with an override-token error
// when the project allows override-with-token, exercising the same-identity lane of
// ensureOrchestratorOverlapPolicy. Cements acceptance 2.2.
func TestIssueCapabilityLeaseRejectsSameIdentityReclaim(t *testing.T) {
	repo := newFakeRepo()
	ids := []string{"p1", "lease-token-a", "lease-token-b"}
	idx := 0
	svc := NewService(repo, func() string {
		id := ids[idx]
		idx++
		return id
	}, func() time.Time {
		return time.Date(2026, 2, 24, 10, 0, 0, 0, time.UTC)
	}, ServiceConfig{DefaultDeleteMode: DeleteModeArchive})

	project, err := svc.CreateProjectWithMetadata(context.Background(), CreateProjectInput{
		Name: "Same Identity Reclaim",
		Metadata: domain.ProjectMetadata{
			CapabilityPolicy: domain.ProjectCapabilityPolicy{
				AllowOrchestratorOverride: true,
				OrchestratorOverrideToken: "override-xyz",
			},
		},
	})
	if err != nil {
		t.Fatalf("CreateProjectWithMetadata() error = %v", err)
	}

	if _, err := svc.IssueCapabilityLease(context.Background(), IssueCapabilityLeaseInput{
		ProjectID:       project.ID,
		ScopeType:       domain.CapabilityScopeProject,
		Role:            domain.CapabilityRoleOrchestrator,
		AgentName:       "orch-a",
		AgentInstanceID: "orch-a-inst-1",
	}); err != nil {
		t.Fatalf("IssueCapabilityLease(first) error = %v", err)
	}

	if _, err := svc.IssueCapabilityLease(context.Background(), IssueCapabilityLeaseInput{
		ProjectID:       project.ID,
		ScopeType:       domain.CapabilityScopeProject,
		Role:            domain.CapabilityRoleOrchestrator,
		AgentName:       "orch-a",
		AgentInstanceID: "orch-a-inst-2",
	}); err != domain.ErrOverrideTokenRequired {
		t.Fatalf("IssueCapabilityLease(same-identity reclaim) error = %v, want ErrOverrideTokenRequired", err)
	}
}

// TestIssueCapabilityLeaseRevokeOneIdentityLeavesOthers verifies revoking one orchestrator
// identity's lease does not revoke or invalidate a peer orchestrator identity's lease at
// the same scope. Cements acceptance 2.3.
func TestIssueCapabilityLeaseRevokeOneIdentityLeavesOthers(t *testing.T) {
	repo := newFakeRepo()
	ids := []string{"p1", "lease-token-a", "lease-token-b"}
	idx := 0
	svc := NewService(repo, func() string {
		id := ids[idx]
		idx++
		return id
	}, func() time.Time {
		return time.Date(2026, 2, 24, 10, 0, 0, 0, time.UTC)
	}, ServiceConfig{DefaultDeleteMode: DeleteModeArchive})

	project, err := svc.CreateProject(context.Background(), "Revoke One", "")
	if err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}

	leaseA, err := svc.IssueCapabilityLease(context.Background(), IssueCapabilityLeaseInput{
		ProjectID:       project.ID,
		ScopeType:       domain.CapabilityScopeProject,
		Role:            domain.CapabilityRoleOrchestrator,
		AgentName:       "orch-a",
		AgentInstanceID: "orch-a-inst",
	})
	if err != nil {
		t.Fatalf("IssueCapabilityLease(orch-a) error = %v", err)
	}
	if _, err := svc.IssueCapabilityLease(context.Background(), IssueCapabilityLeaseInput{
		ProjectID:       project.ID,
		ScopeType:       domain.CapabilityScopeProject,
		Role:            domain.CapabilityRoleOrchestrator,
		AgentName:       "orch-b",
		AgentInstanceID: "orch-b-inst",
	}); err != nil {
		t.Fatalf("IssueCapabilityLease(orch-b) error = %v", err)
	}

	if _, err := svc.RevokeCapabilityLease(context.Background(), RevokeCapabilityLeaseInput{
		AgentInstanceID: leaseA.InstanceID,
		Reason:          "done",
	}); err != nil {
		t.Fatalf("RevokeCapabilityLease(orch-a) error = %v", err)
	}

	active, err := svc.ListCapabilityLeases(context.Background(), ListCapabilityLeasesInput{
		ProjectID: project.ID,
		ScopeType: domain.CapabilityScopeProject,
	})
	if err != nil {
		t.Fatalf("ListCapabilityLeases(active) error = %v", err)
	}
	if len(active) != 1 || active[0].AgentName != "orch-b" {
		t.Fatalf("ListCapabilityLeases(active) = %#v, want only orch-b active", active)
	}
	if active[0].RevokedAt != nil {
		t.Fatalf("surviving lease was revoked: %#v", active[0])
	}
}

// TestIssueCapabilityLeaseOverlapDifferentIdentitiesNoTokenRequired verifies the override
// policy is never consulted when the two orchestrator leases carry different AgentName
// values, even on a project that does not opt into override. Belt-and-suspenders cement of
// acceptance 2.1 against the policy-less default.
func TestIssueCapabilityLeaseOverlapDifferentIdentitiesNoTokenRequired(t *testing.T) {
	repo := newFakeRepo()
	ids := []string{"p1", "lease-token-a", "lease-token-b"}
	idx := 0
	svc := NewService(repo, func() string {
		id := ids[idx]
		idx++
		return id
	}, func() time.Time {
		return time.Date(2026, 2, 24, 10, 0, 0, 0, time.UTC)
	}, ServiceConfig{DefaultDeleteMode: DeleteModeArchive})

	// No CapabilityPolicy overrides set — AllowOrchestratorOverride defaults false.
	project, err := svc.CreateProject(context.Background(), "No Override Policy", "")
	if err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}

	if _, err := svc.IssueCapabilityLease(context.Background(), IssueCapabilityLeaseInput{
		ProjectID:       project.ID,
		ScopeType:       domain.CapabilityScopeProject,
		Role:            domain.CapabilityRoleOrchestrator,
		AgentName:       "orch-a",
		AgentInstanceID: "orch-a-inst",
	}); err != nil {
		t.Fatalf("IssueCapabilityLease(orch-a) error = %v", err)
	}
	if _, err := svc.IssueCapabilityLease(context.Background(), IssueCapabilityLeaseInput{
		ProjectID:       project.ID,
		ScopeType:       domain.CapabilityScopeProject,
		Role:            domain.CapabilityRoleOrchestrator,
		AgentName:       "orch-b",
		AgentInstanceID: "orch-b-inst",
	}); err != nil {
		t.Fatalf("IssueCapabilityLease(orch-b, no override policy) error = %v, want nil", err)
	}
}

// TestIssueCapabilityLeaseSameInstanceIDRetry verifies the AgentInstanceID short-circuit at
// the top of ensureOrchestratorOverlapPolicy's loop: a retry that reuses the same
// AgentInstanceID does not hit the identity check. Under the fake repo's idempotent
// CreateCapabilityLease, the second issue overwrites the first without error.
func TestIssueCapabilityLeaseSameInstanceIDRetry(t *testing.T) {
	repo := newFakeRepo()
	ids := []string{"p1", "lease-token-a", "lease-token-b"}
	idx := 0
	svc := NewService(repo, func() string {
		id := ids[idx]
		idx++
		return id
	}, func() time.Time {
		return time.Date(2026, 2, 24, 10, 0, 0, 0, time.UTC)
	}, ServiceConfig{DefaultDeleteMode: DeleteModeArchive})

	project, err := svc.CreateProject(context.Background(), "Same Instance Retry", "")
	if err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}

	first, err := svc.IssueCapabilityLease(context.Background(), IssueCapabilityLeaseInput{
		ProjectID:       project.ID,
		ScopeType:       domain.CapabilityScopeProject,
		Role:            domain.CapabilityRoleOrchestrator,
		AgentName:       "orch-a",
		AgentInstanceID: "orch-a-inst",
	})
	if err != nil {
		t.Fatalf("IssueCapabilityLease(first) error = %v", err)
	}

	second, err := svc.IssueCapabilityLease(context.Background(), IssueCapabilityLeaseInput{
		ProjectID:       project.ID,
		ScopeType:       domain.CapabilityScopeProject,
		Role:            domain.CapabilityRoleOrchestrator,
		AgentName:       "orch-a",
		AgentInstanceID: "orch-a-inst",
	})
	if err != nil {
		t.Fatalf("IssueCapabilityLease(same-instance retry) error = %v, want nil via InstanceID short-circuit", err)
	}
	if second.InstanceID != first.InstanceID {
		t.Fatalf("second.InstanceID = %q, want %q", second.InstanceID, first.InstanceID)
	}
	if second.LeaseToken == first.LeaseToken {
		t.Fatalf("second.LeaseToken should rotate on retry; both were %q", second.LeaseToken)
	}

	leases, err := svc.ListCapabilityLeases(context.Background(), ListCapabilityLeasesInput{
		ProjectID:      project.ID,
		ScopeType:      domain.CapabilityScopeProject,
		IncludeRevoked: true,
	})
	if err != nil {
		t.Fatalf("ListCapabilityLeases() error = %v", err)
	}
	if len(leases) != 1 {
		t.Fatalf("ListCapabilityLeases() len = %d, want 1 after same-instance retry on fake repo", len(leases))
	}
}

// TestIssueCapabilityLeaseSameIdentityAfterExpiry verifies that once an existing same-identity
// orchestrator lease has passed its ExpiresAt deadline, the !existing.IsActive(now)
// short-circuit at kind_capability.go:439 lets the same identity reissue without override token.
func TestIssueCapabilityLeaseSameIdentityAfterExpiry(t *testing.T) {
	repo := newFakeRepo()
	ids := []string{"p1", "lease-token-a", "lease-token-b"}
	idx := 0
	clockNow := time.Date(2026, 2, 24, 10, 0, 0, 0, time.UTC)
	svc := NewService(repo, func() string {
		id := ids[idx]
		idx++
		return id
	}, func() time.Time {
		return clockNow
	}, ServiceConfig{DefaultDeleteMode: DeleteModeArchive})

	project, err := svc.CreateProject(context.Background(), "After Expiry", "")
	if err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}

	if _, err := svc.IssueCapabilityLease(context.Background(), IssueCapabilityLeaseInput{
		ProjectID:       project.ID,
		ScopeType:       domain.CapabilityScopeProject,
		Role:            domain.CapabilityRoleOrchestrator,
		AgentName:       "orch-a",
		AgentInstanceID: "orch-a-inst-1",
		RequestedTTL:    5 * time.Minute,
	}); err != nil {
		t.Fatalf("IssueCapabilityLease(first) error = %v", err)
	}

	// Advance the clock past the first lease's ExpiresAt.
	clockNow = clockNow.Add(10 * time.Minute)

	if _, err := svc.IssueCapabilityLease(context.Background(), IssueCapabilityLeaseInput{
		ProjectID:       project.ID,
		ScopeType:       domain.CapabilityScopeProject,
		Role:            domain.CapabilityRoleOrchestrator,
		AgentName:       "orch-a",
		AgentInstanceID: "orch-a-inst-2",
	}); err != nil {
		t.Fatalf("IssueCapabilityLease(after expiry) error = %v, want nil via !IsActive short-circuit", err)
	}
}

// TestIssueCapabilityLeaseSameIdentityAfterRevoke verifies that once an existing same-identity
// orchestrator lease is explicitly revoked, the same identity can reissue without override
// token (revoked leases are not IsActive).
func TestIssueCapabilityLeaseSameIdentityAfterRevoke(t *testing.T) {
	repo := newFakeRepo()
	ids := []string{"p1", "lease-token-a", "lease-token-b"}
	idx := 0
	svc := NewService(repo, func() string {
		id := ids[idx]
		idx++
		return id
	}, func() time.Time {
		return time.Date(2026, 2, 24, 10, 0, 0, 0, time.UTC)
	}, ServiceConfig{DefaultDeleteMode: DeleteModeArchive})

	project, err := svc.CreateProject(context.Background(), "After Revoke", "")
	if err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}

	first, err := svc.IssueCapabilityLease(context.Background(), IssueCapabilityLeaseInput{
		ProjectID:       project.ID,
		ScopeType:       domain.CapabilityScopeProject,
		Role:            domain.CapabilityRoleOrchestrator,
		AgentName:       "orch-a",
		AgentInstanceID: "orch-a-inst-1",
	})
	if err != nil {
		t.Fatalf("IssueCapabilityLease(first) error = %v", err)
	}

	if _, err := svc.RevokeCapabilityLease(context.Background(), RevokeCapabilityLeaseInput{
		AgentInstanceID: first.InstanceID,
		Reason:          "rotate",
	}); err != nil {
		t.Fatalf("RevokeCapabilityLease() error = %v", err)
	}

	if _, err := svc.IssueCapabilityLease(context.Background(), IssueCapabilityLeaseInput{
		ProjectID:       project.ID,
		ScopeType:       domain.CapabilityScopeProject,
		Role:            domain.CapabilityRoleOrchestrator,
		AgentName:       "orch-a",
		AgentInstanceID: "orch-a-inst-2",
	}); err != nil {
		t.Fatalf("IssueCapabilityLease(after revoke) error = %v, want nil via !IsActive short-circuit", err)
	}
}

// TestIssueCapabilityLeaseDistinctIdentitiesBranchScope proves the multi-identity allowance
// is scope-type-agnostic: two distinct orchestrator identities holding branch-scope leases
// on the same branch row coexist without override token.
func TestIssueCapabilityLeaseDistinctIdentitiesBranchScope(t *testing.T) {
	repo := newFakeRepo()
	ids := []string{
		"p1", "c1", "branch-1",
		"lease-token-a", "lease-token-b",
	}
	idx := 0
	svc := NewService(repo, func() string {
		id := ids[idx]
		idx++
		return id
	}, func() time.Time {
		return time.Date(2026, 2, 24, 10, 0, 0, 0, time.UTC)
	}, ServiceConfig{DefaultDeleteMode: DeleteModeArchive})

	project, err := svc.CreateProject(context.Background(), "Branch Scope", "")
	if err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}
	column, err := svc.CreateColumn(context.Background(), project.ID, "To Do", 0, 0)
	if err != nil {
		t.Fatalf("CreateColumn() error = %v", err)
	}
	branch, err := svc.CreateActionItem(context.Background(), CreateActionItemInput{
		ProjectID:      project.ID,
		ColumnID:       column.ID,
		Kind:           domain.KindPlan,
		Scope:          domain.KindAppliesToPlan,
		Title:          "Branch A",
		Priority:       domain.PriorityMedium,
		StructuralType: domain.StructuralTypeDroplet,
	})
	if err != nil {
		t.Fatalf("CreateActionItem(branch) error = %v", err)
	}

	// Post-Drop-1.75 every action-item row resolves to
	// CapabilityScopeActionItem; the legacy branch scope no longer attaches
	// to action items.
	if _, err := svc.IssueCapabilityLease(context.Background(), IssueCapabilityLeaseInput{
		ProjectID:       project.ID,
		ScopeType:       domain.CapabilityScopeActionItem,
		ScopeID:         branch.ID,
		Role:            domain.CapabilityRoleOrchestrator,
		AgentName:       "orch-a",
		AgentInstanceID: "orch-a-inst",
	}); err != nil {
		t.Fatalf("IssueCapabilityLease(orch-a, actionItem) error = %v", err)
	}

	if _, err := svc.IssueCapabilityLease(context.Background(), IssueCapabilityLeaseInput{
		ProjectID:       project.ID,
		ScopeType:       domain.CapabilityScopeActionItem,
		ScopeID:         branch.ID,
		Role:            domain.CapabilityRoleOrchestrator,
		AgentName:       "orch-b",
		AgentInstanceID: "orch-b-inst",
	}); err != nil {
		t.Fatalf("IssueCapabilityLease(orch-b, actionItem) error = %v", err)
	}

	leases, err := svc.ListCapabilityLeases(context.Background(), ListCapabilityLeasesInput{
		ProjectID: project.ID,
		ScopeType: domain.CapabilityScopeActionItem,
		ScopeID:   branch.ID,
	})
	if err != nil {
		t.Fatalf("ListCapabilityLeases(actionItem) error = %v", err)
	}
	if len(leases) != 2 {
		t.Fatalf("ListCapabilityLeases(actionItem) len = %d, want 2", len(leases))
	}
}

// TestCreateActionItemMutationGuardRequiredForAgent verifies strict guard enforcement for non-user actor writes.
func TestCreateActionItemMutationGuardRequiredForAgent(t *testing.T) {
	repo := newFakeRepo()
	ids := []string{"p1", "c1", "t1", "lease-1", "lease-token-1", "t2"}
	idx := 0
	svc := NewService(repo, func() string {
		id := ids[idx]
		idx++
		return id
	}, func() time.Time {
		return time.Date(2026, 2, 24, 10, 0, 0, 0, time.UTC)
	}, ServiceConfig{DefaultDeleteMode: DeleteModeArchive})

	project, err := svc.CreateProject(context.Background(), "Guard Project", "")
	if err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}
	column, err := svc.CreateColumn(context.Background(), project.ID, "To Do", 0, 0)
	if err != nil {
		t.Fatalf("CreateColumn() error = %v", err)
	}

	_, err = svc.CreateActionItem(context.Background(), CreateActionItemInput{
		ProjectID:      project.ID,
		ColumnID:       column.ID,
		Title:          "agent actionItem",
		Priority:       domain.PriorityMedium,
		CreatedByActor: "agent-1",
		UpdatedByActor: "agent-1",
		UpdatedByType:  domain.ActorTypeAgent,
		StructuralType: domain.StructuralTypeDroplet,
	})
	if err != domain.ErrMutationLeaseRequired {
		t.Fatalf("expected ErrMutationLeaseRequired, got %v", err)
	}

	lease, err := svc.IssueCapabilityLease(context.Background(), IssueCapabilityLeaseInput{
		ProjectID:       project.ID,
		ScopeType:       domain.CapabilityScopeProject,
		Role:            domain.CapabilityRoleBuilder,
		AgentName:       "agent-1",
		AgentInstanceID: "agent-1",
	})
	if err != nil {
		t.Fatalf("IssueCapabilityLease() error = %v", err)
	}

	guardedCtx := WithMutationGuard(context.Background(), MutationGuard{
		AgentName:       "agent-1",
		AgentInstanceID: lease.InstanceID,
		LeaseToken:      lease.LeaseToken,
	})
	created, err := svc.CreateActionItem(guardedCtx, CreateActionItemInput{
		ProjectID:      project.ID,
		ColumnID:       column.ID,
		Title:          "guarded agent actionItem",
		Priority:       domain.PriorityMedium,
		CreatedByActor: "agent-1",
		UpdatedByActor: "agent-1",
		UpdatedByType:  domain.ActorTypeAgent,
		StructuralType: domain.StructuralTypeDroplet,
	})
	if err != nil {
		t.Fatalf("CreateActionItem(guarded) error = %v", err)
	}
	if strings.TrimSpace(created.ID) == "" {
		t.Fatal("expected created actionItem id to be populated")
	}
	if created.UpdatedByType != domain.ActorTypeAgent {
		t.Fatalf("expected agent attribution on guarded actionItem, got %q", created.UpdatedByType)
	}
}

// TestScopedLeaseAllowsLineageMutations verifies branch/phase/actionItem scoped lease behavior in-subtree.
func TestScopedLeaseAllowsLineageMutations(t *testing.T) {
	repo := newFakeRepo()
	ids := []string{
		"p1", "c1",
		"branch-1", "phase-1", "actionItem-1",
		"lease-branch", "lease-token-branch",
		"lease-phase", "lease-token-phase",
		"actionItem-2",
		"lease-actionItem", "lease-token-actionItem",
		"comment-1",
	}
	idx := 0
	svc := NewService(repo, func() string {
		id := ids[idx]
		idx++
		return id
	}, func() time.Time {
		return time.Date(2026, 2, 24, 10, 0, 0, 0, time.UTC)
	}, ServiceConfig{DefaultDeleteMode: DeleteModeArchive})

	project, err := svc.CreateProject(context.Background(), "Scoped", "")
	if err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}
	column, err := svc.CreateColumn(context.Background(), project.ID, "To Do", 0, 0)
	if err != nil {
		t.Fatalf("CreateColumn() error = %v", err)
	}
	branch, err := svc.CreateActionItem(context.Background(), CreateActionItemInput{
		ProjectID:      project.ID,
		ColumnID:       column.ID,
		Kind:           domain.KindPlan,
		Scope:          domain.KindAppliesToPlan,
		Title:          "Branch A",
		Priority:       domain.PriorityMedium,
		StructuralType: domain.StructuralTypeDroplet,
	})
	if err != nil {
		t.Fatalf("CreateActionItem(branch) error = %v", err)
	}
	phase, err := svc.CreateActionItem(context.Background(), CreateActionItemInput{
		ProjectID:      project.ID,
		ParentID:       branch.ID,
		ColumnID:       column.ID,
		Kind:           domain.KindDiscussion,
		Scope:          domain.KindAppliesToDiscussion,
		Title:          "Phase A",
		Priority:       domain.PriorityMedium,
		StructuralType: domain.StructuralTypeDroplet,
	})
	if err != nil {
		t.Fatalf("CreateActionItem(phase) error = %v", err)
	}
	actionItem, err := svc.CreateActionItem(context.Background(), CreateActionItemInput{
		ProjectID:      project.ID,
		ParentID:       phase.ID,
		ColumnID:       column.ID,
		Kind:           domain.KindPlan,
		Scope:          domain.KindAppliesToPlan,
		Title:          "ActionItem A1",
		Priority:       domain.PriorityMedium,
		StructuralType: domain.StructuralTypeDroplet,
	})
	if err != nil {
		t.Fatalf("CreateActionItem(actionItem) error = %v", err)
	}

	// Post-Drop-1.75 every action-item row lives at CapabilityScopeActionItem;
	// the legacy branch/phase capability scopes no longer attach to action
	// items, so the lineage test exercises the action-item scope throughout.
	branchLease, err := svc.IssueCapabilityLease(context.Background(), IssueCapabilityLeaseInput{
		ProjectID:       project.ID,
		ScopeType:       domain.CapabilityScopeActionItem,
		ScopeID:         branch.ID,
		Role:            domain.CapabilityRoleBuilder,
		AgentName:       "branch-agent",
		AgentInstanceID: "branch-agent",
	})
	if err != nil {
		t.Fatalf("IssueCapabilityLease(branch-as-actionItem) error = %v", err)
	}
	branchCtx := WithMutationGuard(context.Background(), MutationGuard{
		AgentName:       branchLease.AgentName,
		AgentInstanceID: branchLease.InstanceID,
		LeaseToken:      branchLease.LeaseToken,
	})
	if _, err := svc.UpdateActionItem(branchCtx, UpdateActionItemInput{
		ActionItemID: branch.ID,
		Title:        ptrTo("Branch A"),
		Description:  ptrTo("branch-updated"),
		Priority:     ptrTo(domain.PriorityMedium),
		UpdatedBy:    "branch-agent",
		UpdatedType:  domain.ActorTypeAgent,
	}); err != nil {
		t.Fatalf("UpdateActionItem(branch scoped) error = %v", err)
	}

	phaseLease, err := svc.IssueCapabilityLease(context.Background(), IssueCapabilityLeaseInput{
		ProjectID:       project.ID,
		ScopeType:       domain.CapabilityScopeActionItem,
		ScopeID:         phase.ID,
		Role:            domain.CapabilityRoleBuilder,
		AgentName:       "phase-agent",
		AgentInstanceID: "phase-agent",
	})
	if err != nil {
		t.Fatalf("IssueCapabilityLease(phase-as-actionItem) error = %v", err)
	}
	phaseCtx := WithMutationGuard(context.Background(), MutationGuard{
		AgentName:       phaseLease.AgentName,
		AgentInstanceID: phaseLease.InstanceID,
		LeaseToken:      phaseLease.LeaseToken,
	})
	if _, err := svc.CreateActionItem(phaseCtx, CreateActionItemInput{
		ProjectID:      project.ID,
		ParentID:       phase.ID,
		ColumnID:       column.ID,
		Kind:           domain.KindPlan,
		Scope:          domain.KindAppliesToPlan,
		Title:          "ActionItem A2",
		Priority:       domain.PriorityMedium,
		CreatedByActor: "phase-agent",
		UpdatedByActor: "phase-agent",
		UpdatedByType:  domain.ActorTypeAgent,
		StructuralType: domain.StructuralTypeDroplet,
	}); err != nil {
		t.Fatalf("CreateActionItem(phase scoped) error = %v", err)
	}

	actionItemLease, err := svc.IssueCapabilityLease(context.Background(), IssueCapabilityLeaseInput{
		ProjectID:       project.ID,
		ScopeType:       domain.CapabilityScopeActionItem,
		ScopeID:         actionItem.ID,
		Role:            domain.CapabilityRoleBuilder,
		AgentName:       "actionItem-agent",
		AgentInstanceID: "actionItem-agent",
	})
	if err != nil {
		t.Fatalf("IssueCapabilityLease(actionItem) error = %v", err)
	}
	actionItemCtx := WithMutationGuard(context.Background(), MutationGuard{
		AgentName:       actionItemLease.AgentName,
		AgentInstanceID: actionItemLease.InstanceID,
		LeaseToken:      actionItemLease.LeaseToken,
	})
	if _, err := svc.CreateComment(actionItemCtx, CreateCommentInput{
		ProjectID:    project.ID,
		TargetType:   domain.CommentTargetTypeActionItem,
		TargetID:     actionItem.ID,
		BodyMarkdown: "actionItem scoped comment",
		ActorType:    domain.ActorTypeAgent,
		ActorID:      "actionItem-agent",
		ActorName:    "actionItem-agent",
	}); err != nil {
		t.Fatalf("CreateComment(actionItem scoped) error = %v", err)
	}
}

// TestScopedLeaseRejectsSiblingMutations verifies out-of-scope sibling writes fail closed.
func TestScopedLeaseRejectsSiblingMutations(t *testing.T) {
	repo := newFakeRepo()
	ids := []string{
		"p1", "c1",
		"branch-1", "phase-a", "phase-b",
		"actionItem-a", "actionItem-b",
		"lease-phase-a", "lease-token-phase-a",
	}
	idx := 0
	svc := NewService(repo, func() string {
		id := ids[idx]
		idx++
		return id
	}, func() time.Time {
		return time.Date(2026, 2, 24, 10, 0, 0, 0, time.UTC)
	}, ServiceConfig{DefaultDeleteMode: DeleteModeArchive})

	project, err := svc.CreateProject(context.Background(), "Scoped Deny", "")
	if err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}
	column, err := svc.CreateColumn(context.Background(), project.ID, "To Do", 0, 0)
	if err != nil {
		t.Fatalf("CreateColumn() error = %v", err)
	}
	branch, err := svc.CreateActionItem(context.Background(), CreateActionItemInput{
		ProjectID:      project.ID,
		ColumnID:       column.ID,
		Kind:           domain.KindPlan,
		Scope:          domain.KindAppliesToPlan,
		Title:          "Branch",
		Priority:       domain.PriorityMedium,
		StructuralType: domain.StructuralTypeDroplet,
	})
	if err != nil {
		t.Fatalf("CreateActionItem(branch) error = %v", err)
	}
	phaseA, err := svc.CreateActionItem(context.Background(), CreateActionItemInput{
		ProjectID:      project.ID,
		ParentID:       branch.ID,
		ColumnID:       column.ID,
		Kind:           domain.KindDiscussion,
		Scope:          domain.KindAppliesToDiscussion,
		Title:          "Phase A",
		Priority:       domain.PriorityMedium,
		StructuralType: domain.StructuralTypeDroplet,
	})
	if err != nil {
		t.Fatalf("CreateActionItem(phaseA) error = %v", err)
	}
	phaseB, err := svc.CreateActionItem(context.Background(), CreateActionItemInput{
		ProjectID:      project.ID,
		ParentID:       branch.ID,
		ColumnID:       column.ID,
		Kind:           domain.KindDiscussion,
		Scope:          domain.KindAppliesToDiscussion,
		Title:          "Phase B",
		Priority:       domain.PriorityMedium,
		StructuralType: domain.StructuralTypeDroplet,
	})
	if err != nil {
		t.Fatalf("CreateActionItem(phaseB) error = %v", err)
	}
	actionItemB, err := svc.CreateActionItem(context.Background(), CreateActionItemInput{
		ProjectID:      project.ID,
		ParentID:       phaseB.ID,
		ColumnID:       column.ID,
		Kind:           domain.KindPlan,
		Scope:          domain.KindAppliesToPlan,
		Title:          "ActionItem B1",
		Priority:       domain.PriorityMedium,
		StructuralType: domain.StructuralTypeDroplet,
	})
	if err != nil {
		t.Fatalf("CreateActionItem(actionItemB) error = %v", err)
	}

	// All action-item rows live at CapabilityScopeActionItem post-Drop-1.75;
	// sibling mutation denial is exercised by leasing one phase-as-actionItem
	// and attempting a write on the other phase's child.
	phaseALease, err := svc.IssueCapabilityLease(context.Background(), IssueCapabilityLeaseInput{
		ProjectID:       project.ID,
		ScopeType:       domain.CapabilityScopeActionItem,
		ScopeID:         phaseA.ID,
		Role:            domain.CapabilityRoleBuilder,
		AgentName:       "phase-a-agent",
		AgentInstanceID: "phase-a-agent",
	})
	if err != nil {
		t.Fatalf("IssueCapabilityLease(phaseA-as-actionItem) error = %v", err)
	}
	phaseACtx := WithMutationGuard(context.Background(), MutationGuard{
		AgentName:       phaseALease.AgentName,
		AgentInstanceID: phaseALease.InstanceID,
		LeaseToken:      phaseALease.LeaseToken,
	})

	if _, err := svc.UpdateActionItem(phaseACtx, UpdateActionItemInput{
		ActionItemID: actionItemB.ID,
		Title:        ptrTo("ActionItem B1"),
		Description:  ptrTo("out of scope"),
		Priority:     ptrTo(domain.PriorityMedium),
		UpdatedBy:    "phase-a-agent",
		UpdatedType:  domain.ActorTypeAgent,
	}); !errors.Is(err, domain.ErrMutationLeaseInvalid) {
		t.Fatalf("UpdateActionItem(out of scope) error = %v, want ErrMutationLeaseInvalid", err)
	}

	if _, err := svc.CreateActionItem(phaseACtx, CreateActionItemInput{
		ProjectID:      project.ID,
		ParentID:       phaseB.ID,
		ColumnID:       column.ID,
		Kind:           domain.KindPlan,
		Scope:          domain.KindAppliesToPlan,
		Title:          "ActionItem B2",
		Priority:       domain.PriorityMedium,
		CreatedByActor: "phase-a-agent",
		UpdatedByActor: "phase-a-agent",
		UpdatedByType:  domain.ActorTypeAgent,
		StructuralType: domain.StructuralTypeDroplet,
	}); !errors.Is(err, domain.ErrMutationLeaseInvalid) {
		t.Fatalf("CreateActionItem(out of scope) error = %v, want ErrMutationLeaseInvalid", err)
	}
}

// TestCreateActionItemKindPayloadValidation verifies schema-based runtime validation for dynamic kinds.
//
// Drop 4c.5 droplet F.1.2 NOTE: post-F.1.2 every freshly-created
// project bakes a non-empty KindCatalogJSON from the embedded default
// (when no on-disk .tillsyn/template.toml is found). The catalog-first
// resolveActionItemKindDefinition path uses synthesizeKindDefinitionFromCatalog,
// which deliberately leaves PayloadSchemaJSON = "" (per
// kind_capability.go § synthesizeKindDefinitionFromCatalog doc-comment:
// "templates v1 does not encode schemas; legacy repo path remains the
// only schema source until a future drop"). To exercise the legacy
// repo path that UpsertKindDefinition writes to, this test forcibly
// clears project.KindCatalogJSON after CreateProjectWithMetadata —
// this is the explicit "force legacy path" pattern that survives the
// F.1.2 walk landing (which would otherwise auto-populate the
// catalog). Future drops that fold PayloadSchemaJSON into the catalog
// can drop the clear and assert directly against the catalog-baked
// schema.
func TestCreateActionItemKindPayloadValidation(t *testing.T) {
	repo := newFakeRepo()
	ids := []string{"p1", "c1", "t1", "t2"}
	idx := 0
	svc := NewService(repo, func() string {
		id := ids[idx]
		idx++
		return id
	}, func() time.Time {
		return time.Date(2026, 2, 24, 10, 0, 0, 0, time.UTC)
	}, ServiceConfig{DefaultDeleteMode: DeleteModeArchive})

	project, err := svc.CreateProjectWithMetadata(context.Background(), CreateProjectInput{
		Name:                "Kinds",
		RepoPrimaryWorktree: "/abs/path/to/worktree",
	})
	if err != nil {
		t.Fatalf("CreateProjectWithMetadata() error = %v", err)
	}
	// Force the legacy repo-path resolution by clearing the F.1.2-baked
	// catalog. UpsertKindDefinition below writes PayloadSchemaJSON to
	// the repo's KindDefinition store; the legacy path reads from there
	// when the catalog is empty.
	stored := repo.projects[project.ID]
	stored.KindCatalogJSON = nil
	repo.projects[project.ID] = stored
	column, err := svc.CreateColumn(context.Background(), project.ID, "To Do", 0, 0)
	if err != nil {
		t.Fatalf("CreateColumn() error = %v", err)
	}

	// Attach a payload schema to a member of the 12-value Kind enum; custom
	// kinds can no longer be used for action-item rows post-Drop-1.75.
	_, err = svc.UpsertKindDefinition(context.Background(), CreateKindDefinitionInput{
		ID:                domain.KindID(domain.KindRefinement),
		DisplayName:       "Refinement",
		AppliesTo:         []domain.KindAppliesTo{domain.KindAppliesToRefinement},
		PayloadSchemaJSON: `{"type":"object","required":["package"],"properties":{"package":{"type":"string"}},"additionalProperties":false}`,
	})
	if err != nil {
		t.Fatalf("UpsertKindDefinition() error = %v", err)
	}
	if err := svc.SetProjectAllowedKinds(context.Background(), SetProjectAllowedKindsInput{
		ProjectID: project.ID,
		KindIDs:   []domain.KindID{domain.KindID(domain.KindPlan), domain.KindID(domain.KindBuild), domain.KindID(domain.KindRefinement)},
	}); err != nil {
		t.Fatalf("SetProjectAllowedKinds() error = %v", err)
	}

	_, err = svc.CreateActionItem(context.Background(), CreateActionItemInput{
		ProjectID:      project.ID,
		ColumnID:       column.ID,
		Kind:           domain.KindRefinement,
		Title:          "invalid payload",
		Priority:       domain.PriorityMedium,
		Metadata:       domain.ActionItemMetadata{KindPayload: json.RawMessage(`{"missing":"value"}`)},
		StructuralType: domain.StructuralTypeDroplet,
	})
	if !errors.Is(err, domain.ErrInvalidKindPayload) {
		t.Fatalf("expected ErrInvalidKindPayload, got %v", err)
	}

	created, err := svc.CreateActionItem(context.Background(), CreateActionItemInput{
		ProjectID:      project.ID,
		ColumnID:       column.ID,
		Kind:           domain.KindRefinement,
		Title:          "valid payload",
		Priority:       domain.PriorityMedium,
		Metadata:       domain.ActionItemMetadata{KindPayload: json.RawMessage(`{"package":"internal/app"}`)},
		StructuralType: domain.StructuralTypeDroplet,
	})
	if err != nil {
		t.Fatalf("CreateActionItem(valid payload) error = %v", err)
	}
	if created.Kind != domain.KindRefinement {
		t.Fatalf("expected refinement kind, got %q", created.Kind)
	}
}

// TestReparentActionItemRejectsCycle verifies cycle prevention during reparenting.
func TestReparentActionItemRejectsCycle(t *testing.T) {
	repo := newFakeRepo()
	ids := []string{"p1", "c1", "t-parent", "t-child"}
	idx := 0
	svc := NewService(repo, func() string {
		id := ids[idx]
		idx++
		return id
	}, func() time.Time {
		return time.Date(2026, 2, 24, 10, 0, 0, 0, time.UTC)
	}, ServiceConfig{DefaultDeleteMode: DeleteModeArchive})

	project, err := svc.CreateProject(context.Background(), "Hierarchy", "")
	if err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}
	column, err := svc.CreateColumn(context.Background(), project.ID, "To Do", 0, 0)
	if err != nil {
		t.Fatalf("CreateColumn() error = %v", err)
	}
	parent, err := svc.CreateActionItem(context.Background(), CreateActionItemInput{
		ProjectID:      project.ID,
		ColumnID:       column.ID,
		Title:          "parent",
		Priority:       domain.PriorityMedium,
		StructuralType: domain.StructuralTypeDroplet,
	})
	if err != nil {
		t.Fatalf("CreateActionItem(parent) error = %v", err)
	}
	child, err := svc.CreateActionItem(context.Background(), CreateActionItemInput{
		ProjectID:      project.ID,
		ParentID:       parent.ID,
		Kind:           domain.KindBuild,
		ColumnID:       column.ID,
		Title:          "child",
		Priority:       domain.PriorityMedium,
		StructuralType: domain.StructuralTypeDroplet,
	})
	if err != nil {
		t.Fatalf("CreateActionItem(child) error = %v", err)
	}

	if _, err := svc.ReparentActionItem(context.Background(), parent.ID, child.ID); err != domain.ErrInvalidParentID {
		t.Fatalf("expected ErrInvalidParentID, got %v", err)
	}
}

// TestMoveActionItemToFailedUsesMarkFailedCapability verifies that moving to the failed column uses CapabilityActionMarkFailed.
func TestMoveActionItemToFailedUsesMarkFailedCapability(t *testing.T) {
	repo := newFakeRepo()
	now := time.Date(2026, 2, 21, 12, 0, 0, 0, time.UTC)
	project, _ := domain.NewProjectFromInput(domain.ProjectInput{ID: "p1", Name: "Inbox"}, now)
	repo.projects[project.ID] = project
	progress, _ := domain.NewColumn("c2", project.ID, "In Progress", 1, 0, now)
	failed, _ := domain.NewColumn("c4", project.ID, "Failed", 3, 0, now)
	repo.columns[progress.ID] = progress
	repo.columns[failed.ID] = failed

	actionItem, _ := domain.NewActionItemForTest(domain.ActionItemInput{
		Kind:           domain.KindPlan,
		ID:             "t1",
		ProjectID:      project.ID,
		ColumnID:       progress.ID,
		Position:       0,
		Title:          "failing actionItem",
		Priority:       domain.PriorityMedium,
		LifecycleState: domain.StateInProgress,
		// Drop 4c.5 droplet A.4: transitions into StateFailed require
		// metadata.outcome ∈ {"failure", "blocked", "superseded"}. The
		// pre-A.4 version of this test left outcome empty and relied on the
		// service to accept the move; the new invariant rejects that path,
		// so we pre-populate the outcome here. Production agents follow the
		// same documented order — UpdateActionItem to set metadata BEFORE
		// MoveActionItem flips the column (CLAUDE.md § "Action-Item
		// Lifecycle").
		Metadata: domain.ActionItemMetadata{Outcome: "failure"},
	}, now)
	repo.tasks[actionItem.ID] = actionItem

	svc := NewService(repo, nil, func() time.Time { return now.Add(time.Minute) }, ServiceConfig{})
	moved, err := svc.MoveActionItem(context.Background(), actionItem.ID, failed.ID, 0)
	if err != nil {
		t.Fatalf("MoveActionItem() error = %v", err)
	}
	if moved.LifecycleState != domain.StateFailed {
		t.Fatalf("expected failed lifecycle state, got %q", moved.LifecycleState)
	}
	if moved.CompletedAt == nil {
		t.Fatal("expected completed_at to be set for failed state")
	}
}

// TestMoveActionItemToFailedSkipsCompletionCriteria verifies that moving to failed does not check completion criteria.
func TestMoveActionItemToFailedSkipsCompletionCriteria(t *testing.T) {
	repo := newFakeRepo()
	now := time.Date(2026, 2, 21, 12, 0, 0, 0, time.UTC)
	project, _ := domain.NewProjectFromInput(domain.ProjectInput{ID: "p1", Name: "Inbox"}, now)
	repo.projects[project.ID] = project
	progress, _ := domain.NewColumn("c2", project.ID, "In Progress", 1, 0, now)
	failed, _ := domain.NewColumn("c4", project.ID, "Failed", 3, 0, now)
	repo.columns[progress.ID] = progress
	repo.columns[failed.ID] = failed

	parent, _ := domain.NewActionItemForTest(domain.ActionItemInput{
		Kind:           domain.KindPlan,
		ID:             "t-parent",
		ProjectID:      project.ID,
		ColumnID:       progress.ID,
		Position:       0,
		Title:          "parent with incomplete children",
		Priority:       domain.PriorityHigh,
		LifecycleState: domain.StateInProgress,
		Metadata: domain.ActionItemMetadata{
			// Drop 4c.5 droplet A.4: transitions into StateFailed require a
			// non-empty metadata.outcome. The completion-criteria-bypass
			// behavior under test is orthogonal to outcome; we set
			// "failure" here because the parent is genuinely failing.
			Outcome: "failure",
			CompletionContract: domain.CompletionContract{
				CompletionCriteria: []domain.ChecklistItem{{ID: "c1", Text: "tests green", Complete: false}},
			},
		},
	}, now)
	child, _ := domain.NewActionItemForTest(domain.ActionItemInput{
		Kind:           domain.KindPlan,
		ID:             "t-child",
		ProjectID:      project.ID,
		ParentID:       parent.ID,
		ColumnID:       progress.ID,
		Position:       1,
		Title:          "incomplete child",
		Priority:       domain.PriorityLow,
		LifecycleState: domain.StateInProgress,
	}, now)
	repo.tasks[parent.ID] = parent
	repo.tasks[child.ID] = child

	svc := NewService(repo, nil, func() time.Time { return now.Add(time.Minute) }, ServiceConfig{})
	moved, err := svc.MoveActionItem(context.Background(), parent.ID, failed.ID, 0)
	if err != nil {
		t.Fatalf("MoveActionItem() to failed should succeed with incomplete children, got error = %v", err)
	}
	if moved.LifecycleState != domain.StateFailed {
		t.Fatalf("expected failed lifecycle state, got %q", moved.LifecycleState)
	}
}

// TestMoveActionItemFromFailedToTodoBlocked verifies that transitions FROM the failed terminal state are blocked.
func TestMoveActionItemFromFailedToTodoBlocked(t *testing.T) {
	repo := newFakeRepo()
	now := time.Date(2026, 2, 21, 12, 0, 0, 0, time.UTC)
	project, _ := domain.NewProjectFromInput(domain.ProjectInput{ID: "p1", Name: "Inbox"}, now)
	repo.projects[project.ID] = project
	todo, _ := domain.NewColumn("c1", project.ID, "To Do", 0, 0, now)
	failed, _ := domain.NewColumn("c4", project.ID, "Failed", 3, 0, now)
	repo.columns[todo.ID] = todo
	repo.columns[failed.ID] = failed

	actionItem, _ := domain.NewActionItemForTest(domain.ActionItemInput{
		Kind:           domain.KindPlan,
		ID:             "t1",
		ProjectID:      project.ID,
		ColumnID:       failed.ID,
		Position:       0,
		Title:          "failed actionItem",
		Priority:       domain.PriorityMedium,
		LifecycleState: domain.StateFailed,
	}, now)
	repo.tasks[actionItem.ID] = actionItem

	svc := NewService(repo, nil, func() time.Time { return now.Add(time.Minute) }, ServiceConfig{})
	_, err := svc.MoveActionItem(context.Background(), actionItem.ID, todo.ID, 0)
	if err == nil {
		t.Fatal("MoveActionItem() from failed to todo should return an error")
	}
	if !errors.Is(err, domain.ErrTransitionBlocked) {
		t.Fatalf("expected ErrTransitionBlocked, got %v", err)
	}
}

// TestMoveActionItemFromDoneToTodoBlocked verifies that transitions FROM the complete terminal state are blocked.
func TestMoveActionItemFromDoneToTodoBlocked(t *testing.T) {
	repo := newFakeRepo()
	now := time.Date(2026, 2, 21, 12, 0, 0, 0, time.UTC)
	project, _ := domain.NewProjectFromInput(domain.ProjectInput{ID: "p1", Name: "Inbox"}, now)
	repo.projects[project.ID] = project
	todo, _ := domain.NewColumn("c1", project.ID, "To Do", 0, 0, now)
	done, _ := domain.NewColumn("c3", project.ID, "Complete", 2, 0, now)
	repo.columns[todo.ID] = todo
	repo.columns[done.ID] = done

	actionItem, _ := domain.NewActionItemForTest(domain.ActionItemInput{
		Kind:           domain.KindPlan,
		ID:             "t1",
		ProjectID:      project.ID,
		ColumnID:       done.ID,
		Position:       0,
		Title:          "complete actionItem",
		Priority:       domain.PriorityMedium,
		LifecycleState: domain.StateComplete,
	}, now)
	repo.tasks[actionItem.ID] = actionItem

	svc := NewService(repo, nil, func() time.Time { return now.Add(time.Minute) }, ServiceConfig{})
	_, err := svc.MoveActionItem(context.Background(), actionItem.ID, todo.ID, 0)
	if err == nil {
		t.Fatal("MoveActionItem() from complete to todo should return an error")
	}
	if !errors.Is(err, domain.ErrTransitionBlocked) {
		t.Fatalf("expected ErrTransitionBlocked, got %v", err)
	}
}

// TestMoveActionItemFromFailedIdempotentAllowed verifies that idempotent moves (same column, same state) are permitted for terminal states.
func TestMoveActionItemFromFailedIdempotentAllowed(t *testing.T) {
	repo := newFakeRepo()
	now := time.Date(2026, 2, 21, 12, 0, 0, 0, time.UTC)
	project, _ := domain.NewProjectFromInput(domain.ProjectInput{ID: "p1", Name: "Inbox"}, now)
	repo.projects[project.ID] = project
	failed, _ := domain.NewColumn("c4", project.ID, "Failed", 3, 0, now)
	repo.columns[failed.ID] = failed

	actionItem, _ := domain.NewActionItemForTest(domain.ActionItemInput{
		Kind:           domain.KindPlan,
		ID:             "t1",
		ProjectID:      project.ID,
		ColumnID:       failed.ID,
		Position:       0,
		Title:          "failed actionItem",
		Priority:       domain.PriorityMedium,
		LifecycleState: domain.StateFailed,
	}, now)
	repo.tasks[actionItem.ID] = actionItem

	svc := NewService(repo, nil, func() time.Time { return now.Add(time.Minute) }, ServiceConfig{})
	moved, err := svc.MoveActionItem(context.Background(), actionItem.ID, failed.ID, 0)
	if err != nil {
		t.Fatalf("MoveActionItem() idempotent move on failed actionItem should succeed, got error = %v", err)
	}
	if moved.LifecycleState != domain.StateFailed {
		t.Fatalf("expected failed lifecycle state, got %q", moved.LifecycleState)
	}
}

// TestMoveActionItemFailedTransitionRequiresOutcome pins the Drop 4c.5
// droplet A.4 invariant: transitions INTO StateFailed require a non-empty
// metadata.outcome from the closed set {"failure", "blocked", "superseded"}.
// The check is asymmetric — moves into StateComplete or StateInProgress do
// NOT enforce outcome shape. Idempotent failed→failed self-moves carve out
// (existing test TestMoveActionItemFromFailedIdempotentAllowed pins that
// path; this test focuses on the actual transition into failed).
//
// Each row sets up an in_progress action item with a specific
// metadata.outcome value, attempts a Move into the named destination state,
// and asserts the wrapped error class plus post-move lifecycle state.
func TestMoveActionItemFailedTransitionRequiresOutcome(t *testing.T) {
	type row struct {
		name           string
		outcome        string
		toStateColumn  string // "failed" | "complete" | "in_progress"
		wantErrIs      error  // nil = move must succeed
		wantFinalState domain.LifecycleState
	}

	rows := []row{
		{
			name:          "failed-no-outcome rejected",
			outcome:       "",
			toStateColumn: "failed",
			wantErrIs:     domain.ErrInvalidMetadataOutcome,
			// State unchanged on rejection — guard fires before column move.
			wantFinalState: domain.StateInProgress,
		},
		{
			name:           "failed-whitespace-outcome rejected",
			outcome:        "   ",
			toStateColumn:  "failed",
			wantErrIs:      domain.ErrInvalidMetadataOutcome,
			wantFinalState: domain.StateInProgress,
		},
		{
			name:           "failed-with-success-outcome rejected",
			outcome:        "success",
			toStateColumn:  "failed",
			wantErrIs:      domain.ErrInvalidMetadataOutcome,
			wantFinalState: domain.StateInProgress,
		},
		{
			name:           "failed-with-garbage-outcome rejected",
			outcome:        "garbage-not-in-enum",
			toStateColumn:  "failed",
			wantErrIs:      domain.ErrInvalidMetadataOutcome,
			wantFinalState: domain.StateInProgress,
		},
		{
			name:           "failed-with-failure-outcome accepted",
			outcome:        "failure",
			toStateColumn:  "failed",
			wantErrIs:      nil,
			wantFinalState: domain.StateFailed,
		},
		{
			name:           "failed-with-blocked-outcome accepted",
			outcome:        "blocked",
			toStateColumn:  "failed",
			wantErrIs:      nil,
			wantFinalState: domain.StateFailed,
		},
		{
			name:           "failed-with-superseded-outcome accepted",
			outcome:        "superseded",
			toStateColumn:  "failed",
			wantErrIs:      nil,
			wantFinalState: domain.StateFailed,
		},
		{
			name: "failed-with-mixed-case-outcome accepted (case-insensitive)",
			// The validator lowercases on compare so callers that send
			// "Failure" / "BLOCKED" / "Superseded" match the enum. This row
			// pins the case-insensitivity contract.
			outcome:        "Failure",
			toStateColumn:  "failed",
			wantErrIs:      nil,
			wantFinalState: domain.StateFailed,
		},
		{
			// Asymmetry assertion: the same empty-outcome shape that fails
			// the failed transition succeeds on transition into complete.
			name:           "complete-no-outcome accepted (asymmetry)",
			outcome:        "",
			toStateColumn:  "complete",
			wantErrIs:      nil,
			wantFinalState: domain.StateComplete,
		},
		{
			// Sanity row: the in_progress→in_progress (different position
			// only) move is a no-op for the outcome guard regardless of
			// outcome shape.
			name:           "in_progress-no-outcome accepted",
			outcome:        "",
			toStateColumn:  "in_progress",
			wantErrIs:      nil,
			wantFinalState: domain.StateInProgress,
		},
	}

	for _, r := range rows {
		t.Run(r.name, func(t *testing.T) {
			repo := newFakeRepo()
			now := time.Date(2026, 5, 5, 12, 0, 0, 0, time.UTC)
			project, _ := domain.NewProjectFromInput(domain.ProjectInput{ID: "p1", Name: "Inbox"}, now)
			repo.projects[project.ID] = project
			progress, _ := domain.NewColumn("c2", project.ID, "In Progress", 1, 0, now)
			done, _ := domain.NewColumn("c3", project.ID, "Complete", 2, 0, now)
			failed, _ := domain.NewColumn("c4", project.ID, "Failed", 3, 0, now)
			repo.columns[progress.ID] = progress
			repo.columns[done.ID] = done
			repo.columns[failed.ID] = failed

			actionItem, _ := domain.NewActionItemForTest(domain.ActionItemInput{
				Kind:           domain.KindPlan,
				ID:             "t-a4",
				ProjectID:      project.ID,
				ColumnID:       progress.ID,
				Position:       0,
				Title:          "A.4 outcome guard test",
				Priority:       domain.PriorityMedium,
				LifecycleState: domain.StateInProgress,
				Metadata:       domain.ActionItemMetadata{Outcome: r.outcome},
			}, now)
			repo.tasks[actionItem.ID] = actionItem

			var targetCol domain.Column
			switch r.toStateColumn {
			case "failed":
				targetCol = failed
			case "complete":
				targetCol = done
			case "in_progress":
				targetCol = progress
			default:
				t.Fatalf("unknown toStateColumn fixture %q", r.toStateColumn)
			}

			svc := NewService(repo, nil, func() time.Time { return now.Add(time.Minute) }, ServiceConfig{})
			moved, err := svc.MoveActionItem(context.Background(), actionItem.ID, targetCol.ID, 0)

			if r.wantErrIs == nil {
				if err != nil {
					t.Fatalf("MoveActionItem() error = %v, want nil", err)
				}
				if moved.LifecycleState != r.wantFinalState {
					t.Fatalf("MoveActionItem() final state = %q, want %q", moved.LifecycleState, r.wantFinalState)
				}
				return
			}

			if err == nil {
				t.Fatalf("MoveActionItem() error = nil, want errors.Is(err, %v)", r.wantErrIs)
			}
			if !errors.Is(err, r.wantErrIs) {
				t.Fatalf("MoveActionItem() error = %v, want errors.Is(err, %v)", err, r.wantErrIs)
			}
			// Item must remain at its starting state — guard fires before
			// the column move and the metadata write.
			refetched, getErr := svc.GetActionItem(context.Background(), actionItem.ID)
			if getErr != nil {
				t.Fatalf("GetActionItem() after rejected move error = %v", getErr)
			}
			if refetched.LifecycleState != r.wantFinalState {
				t.Fatalf("post-rejection lifecycle state = %q, want %q (guard must fire before column move)", refetched.LifecycleState, r.wantFinalState)
			}
		})
	}
}

// supersedeFixture wires a fakeRepo with one project + the four canonical
// lifecycle columns (todo / in_progress / complete / failed) so the Drop
// 4c.5 droplet B.1 supersede tests can drop one action item per case at the
// desired starting column. The fixture intentionally uses NewActionItemForTest
// to skirt the create-time path (no embedding queue, no auto-seeding) — the
// supersede method's contract is the unit under test, not the create path.
type supersedeFixture struct {
	repo    *fakeRepo
	svc     *Service
	project domain.Project
	cols    map[domain.LifecycleState]domain.Column
	now     time.Time
}

// newSupersedeFixture seeds the fixture used by every supersede table-driven
// row. Each call gets its own fakeRepo so the rows do not share storage and
// the tests can run in parallel safely.
func newSupersedeFixture(t *testing.T) supersedeFixture {
	t.Helper()
	repo := newFakeRepo()
	now := time.Date(2026, 5, 5, 12, 0, 0, 0, time.UTC)
	project, err := domain.NewProjectFromInput(domain.ProjectInput{ID: "p-supersede", Name: "Supersede Fixture"}, now)
	if err != nil {
		t.Fatalf("NewProjectFromInput() error = %v", err)
	}
	repo.projects[project.ID] = project
	cols := map[domain.LifecycleState]domain.Column{}
	colSpecs := []struct {
		id    string
		name  string
		pos   int
		state domain.LifecycleState
	}{
		{id: "c-todo", name: "To Do", pos: 0, state: domain.StateTodo},
		{id: "c-progress", name: "In Progress", pos: 1, state: domain.StateInProgress},
		{id: "c-complete", name: "Complete", pos: 2, state: domain.StateComplete},
		{id: "c-failed", name: "Failed", pos: 3, state: domain.StateFailed},
	}
	for _, spec := range colSpecs {
		col, err := domain.NewColumn(spec.id, project.ID, spec.name, spec.pos, 0, now)
		if err != nil {
			t.Fatalf("NewColumn(%q) error = %v", spec.name, err)
		}
		repo.columns[col.ID] = col
		cols[spec.state] = col
	}
	svc := NewService(repo, nil, func() time.Time { return now.Add(time.Minute) }, ServiceConfig{})
	return supersedeFixture{repo: repo, svc: svc, project: project, cols: cols, now: now}
}

// seedSupersedeItem drops one action item into the requested starting column
// + lifecycle state. The metadata is empty by default; callers can pre-stamp
// outcome / transition_notes via the in-line helper if a row needs it.
func (f *supersedeFixture) seedSupersedeItem(t *testing.T, id string, state domain.LifecycleState) domain.ActionItem {
	t.Helper()
	col, ok := f.cols[state]
	if !ok {
		t.Fatalf("supersedeFixture: no column for state %q", state)
	}
	actionItem, err := domain.NewActionItemForTest(domain.ActionItemInput{
		Kind:           domain.KindBuild,
		ID:             id,
		ProjectID:      f.project.ID,
		ColumnID:       col.ID,
		Position:       0,
		Title:          "B.1 supersede test",
		Priority:       domain.PriorityMedium,
		LifecycleState: state,
	}, f.now)
	if err != nil {
		t.Fatalf("NewActionItemForTest() error = %v", err)
	}
	f.repo.tasks[actionItem.ID] = actionItem
	return actionItem
}

// TestService_SupersedeActionItem pins the Drop 4c.5 droplet B.1 contract:
// supersede transitions a `failed` action item to `complete` with
// `metadata.outcome = "superseded"` and the supplied reason persisted on
// `metadata.transition_notes`. Non-failed items reject with
// `domain.ErrTransitionBlocked`. Empty / whitespace-only reasons reject
// before any state mutation. Missing items propagate the repo's
// not-found error verbatim.
func TestService_SupersedeActionItem(t *testing.T) {
	t.Parallel()

	t.Run("failed item supersedes to complete with audit trail", func(t *testing.T) {
		t.Parallel()
		f := newSupersedeFixture(t)
		seeded := f.seedSupersedeItem(t, "t-failed-1", domain.StateFailed)
		got, err := f.svc.SupersedeActionItem(context.Background(), seeded.ID, "rejected by dev — re-planning required")
		if err != nil {
			t.Fatalf("SupersedeActionItem() error = %v, want nil", err)
		}
		if got.LifecycleState != domain.StateComplete {
			t.Fatalf("post-supersede state = %q, want %q", got.LifecycleState, domain.StateComplete)
		}
		if got.Metadata.Outcome != "superseded" {
			t.Fatalf("post-supersede outcome = %q, want %q", got.Metadata.Outcome, "superseded")
		}
		if got.Metadata.TransitionNotes != "rejected by dev — re-planning required" {
			t.Fatalf("post-supersede transition_notes = %q, want reason text", got.Metadata.TransitionNotes)
		}
		// ColumnID must match the project's complete column.
		if got.ColumnID != f.cols[domain.StateComplete].ID {
			t.Fatalf("post-supersede column = %q, want %q", got.ColumnID, f.cols[domain.StateComplete].ID)
		}
	})

	t.Run("supersede trims whitespace from the reason", func(t *testing.T) {
		t.Parallel()
		f := newSupersedeFixture(t)
		seeded := f.seedSupersedeItem(t, "t-failed-trim", domain.StateFailed)
		got, err := f.svc.SupersedeActionItem(context.Background(), seeded.ID, "   trimmed reason   ")
		if err != nil {
			t.Fatalf("SupersedeActionItem() error = %v, want nil", err)
		}
		if got.Metadata.TransitionNotes != "trimmed reason" {
			t.Fatalf("transition_notes = %q, want trimmed", got.Metadata.TransitionNotes)
		}
	})

	t.Run("non-failed states reject with ErrTransitionBlocked", func(t *testing.T) {
		t.Parallel()
		rows := []struct {
			name  string
			state domain.LifecycleState
		}{
			{name: "todo", state: domain.StateTodo},
			{name: "in_progress", state: domain.StateInProgress},
			{name: "complete", state: domain.StateComplete},
		}
		for _, r := range rows {
			t.Run(r.name, func(t *testing.T) {
				t.Parallel()
				f := newSupersedeFixture(t)
				seeded := f.seedSupersedeItem(t, "t-"+r.name, r.state)
				_, err := f.svc.SupersedeActionItem(context.Background(), seeded.ID, "valid reason")
				if err == nil {
					t.Fatalf("SupersedeActionItem(%s) error = nil, want ErrTransitionBlocked", r.name)
				}
				if !errors.Is(err, domain.ErrTransitionBlocked) {
					t.Fatalf("SupersedeActionItem(%s) error = %v, want ErrTransitionBlocked", r.name, err)
				}
				if !strings.Contains(err.Error(), "supersede only applies to failed items") {
					t.Fatalf("error message %q missing 'supersede only applies to failed items' hint", err.Error())
				}
				// State unchanged on rejection.
				refetched, getErr := f.svc.GetActionItem(context.Background(), seeded.ID)
				if getErr != nil {
					t.Fatalf("GetActionItem() after rejection error = %v", getErr)
				}
				if refetched.LifecycleState != r.state {
					t.Fatalf("post-rejection state = %q, want %q (guard must fire before mutation)", refetched.LifecycleState, r.state)
				}
				if refetched.Metadata.Outcome == "superseded" {
					t.Fatalf("post-rejection outcome was stamped to %q despite rejection", refetched.Metadata.Outcome)
				}
			})
		}
	})

	t.Run("archived item rejects (lifecycle column lookup miss)", func(t *testing.T) {
		t.Parallel()
		f := newSupersedeFixture(t)
		// Archived items live in their own column; no archived column was
		// seeded so the column resolver maps to the empty state and the
		// fromState fallback uses the action_item's stored LifecycleState
		// (StateArchived). Archived ≠ failed → ErrTransitionBlocked.
		now := f.now
		archived, err := domain.NewActionItemForTest(domain.ActionItemInput{
			Kind:           domain.KindBuild,
			ID:             "t-archived",
			ProjectID:      f.project.ID,
			ColumnID:       f.cols[domain.StateComplete].ID, // any seeded column; state field is the gate
			Position:       0,
			Title:          "archived test",
			Priority:       domain.PriorityMedium,
			LifecycleState: domain.StateArchived,
		}, now)
		if err != nil {
			t.Fatalf("NewActionItemForTest(archived) error = %v", err)
		}
		f.repo.tasks[archived.ID] = archived
		_, supErr := f.svc.SupersedeActionItem(context.Background(), archived.ID, "valid reason")
		if supErr == nil {
			t.Fatal("SupersedeActionItem(archived) error = nil, want non-nil")
		}
		// The lifecycleStateForColumnID resolves to StateComplete via the
		// column lookup, so the failure surfaces as ErrTransitionBlocked
		// with the canonical hint regardless of the item's own
		// LifecycleState. The point is that supersede on an archived item
		// is REJECTED (not silently no-oped).
		if !errors.Is(supErr, domain.ErrTransitionBlocked) {
			t.Fatalf("SupersedeActionItem(archived) error = %v, want ErrTransitionBlocked", supErr)
		}
	})

	t.Run("empty reason rejects before any state mutation", func(t *testing.T) {
		t.Parallel()
		f := newSupersedeFixture(t)
		seeded := f.seedSupersedeItem(t, "t-empty-reason", domain.StateFailed)
		_, err := f.svc.SupersedeActionItem(context.Background(), seeded.ID, "")
		if err == nil {
			t.Fatal("SupersedeActionItem(empty reason) error = nil, want non-nil")
		}
		if !strings.Contains(err.Error(), "reason is required") {
			t.Fatalf("error message %q missing 'reason is required'", err.Error())
		}
		// State must be unchanged.
		refetched, getErr := f.svc.GetActionItem(context.Background(), seeded.ID)
		if getErr != nil {
			t.Fatalf("GetActionItem() after rejection error = %v", getErr)
		}
		if refetched.LifecycleState != domain.StateFailed {
			t.Fatalf("post-rejection state = %q, want %q", refetched.LifecycleState, domain.StateFailed)
		}
	})

	t.Run("whitespace-only reason rejects", func(t *testing.T) {
		t.Parallel()
		f := newSupersedeFixture(t)
		seeded := f.seedSupersedeItem(t, "t-ws-reason", domain.StateFailed)
		_, err := f.svc.SupersedeActionItem(context.Background(), seeded.ID, "   ")
		if err == nil {
			t.Fatal("SupersedeActionItem(whitespace reason) error = nil, want non-nil")
		}
		if !strings.Contains(err.Error(), "reason is required") {
			t.Fatalf("error message %q missing 'reason is required'", err.Error())
		}
	})

	t.Run("empty action_item_id rejects before service path", func(t *testing.T) {
		t.Parallel()
		f := newSupersedeFixture(t)
		_, err := f.svc.SupersedeActionItem(context.Background(), "", "valid reason")
		if err == nil {
			t.Fatal("SupersedeActionItem(empty id) error = nil, want non-nil")
		}
		if !strings.Contains(err.Error(), "action_item_id is required") {
			t.Fatalf("error message %q missing 'action_item_id is required'", err.Error())
		}
	})

	t.Run("missing action_item propagates ErrNotFound from repo", func(t *testing.T) {
		t.Parallel()
		f := newSupersedeFixture(t)
		_, err := f.svc.SupersedeActionItem(context.Background(), "no-such-id", "valid reason")
		if err == nil {
			t.Fatal("SupersedeActionItem(missing) error = nil, want non-nil")
		}
		if !errors.Is(err, ErrNotFound) {
			t.Fatalf("SupersedeActionItem(missing) error = %v, want errors.Is(err, ErrNotFound)", err)
		}
	})

	t.Run("descendants in non-terminal state are NOT cascaded (B.1 §3.1 invariant)", func(t *testing.T) {
		t.Parallel()
		f := newSupersedeFixture(t)
		parent := f.seedSupersedeItem(t, "t-parent-failed", domain.StateFailed)
		// Child item under the failed parent, still in_progress. Supersede
		// on the parent must NOT touch the child.
		child, err := domain.NewActionItemForTest(domain.ActionItemInput{
			Kind:           domain.KindBuild,
			ID:             "t-child-progress",
			ProjectID:      f.project.ID,
			ColumnID:       f.cols[domain.StateInProgress].ID,
			Position:       0,
			ParentID:       parent.ID,
			Title:          "child still running",
			Priority:       domain.PriorityMedium,
			LifecycleState: domain.StateInProgress,
		}, f.now)
		if err != nil {
			t.Fatalf("NewActionItemForTest(child) error = %v", err)
		}
		f.repo.tasks[child.ID] = child
		if _, err := f.svc.SupersedeActionItem(context.Background(), parent.ID, "clearing parent only"); err != nil {
			t.Fatalf("SupersedeActionItem(parent) error = %v, want nil", err)
		}
		// Re-fetch child — must be unchanged.
		refetched, err := f.svc.GetActionItem(context.Background(), child.ID)
		if err != nil {
			t.Fatalf("GetActionItem(child) error = %v", err)
		}
		if refetched.LifecycleState != domain.StateInProgress {
			t.Fatalf("child state after parent supersede = %q, want %q (no cascade)", refetched.LifecycleState, domain.StateInProgress)
		}
		if refetched.Metadata.Outcome == "superseded" {
			t.Fatalf("child outcome stamped %q after parent supersede (no cascade)", refetched.Metadata.Outcome)
		}
	})
}

// recordingGitChecker is a stub GitStatusChecker used by the droplet 4b.6
// CreateActionItem pre-check tests. Callers configure dirtyPaths + err to
// shape the response; calls capture the (worktree, paths) tuple so tests
// can assert the pre-check was invoked exactly when expected.
type recordingGitChecker struct {
	dirtyPaths []string
	err        error
	calls      []recordingGitCheckerCall
}

// recordingGitCheckerCall captures one invocation of the stub checker so
// tests can assert call count + the worktree/paths arguments.
type recordingGitCheckerCall struct {
	worktree string
	paths    []string
}

// check is the GitStatusChecker function adapter so the stub can be
// installed via direct field assignment on Service.gitStatusChecker.
func (r *recordingGitChecker) check(_ context.Context, worktree string, paths []string) ([]string, error) {
	r.calls = append(r.calls, recordingGitCheckerCall{worktree: worktree, paths: append([]string(nil), paths...)})
	if r.err != nil {
		return nil, r.err
	}
	return append([]string(nil), r.dirtyPaths...), nil
}

// newDirtyPathTestService wires a Service backed by a fakeRepo with one
// project (RepoPrimaryWorktree configurable per-test) + one column, ready
// for CreateActionItem. The recordingGitChecker is installed via direct
// field assignment so the production checker never fires.
func newDirtyPathTestService(t *testing.T, worktree string, checker *recordingGitChecker) (*Service, *fakeRepo, domain.Project, domain.Column) {
	t.Helper()
	repo := newFakeRepo()
	now := time.Date(2026, 5, 4, 12, 0, 0, 0, time.UTC)
	ids := []string{"p1", "c1", "t1", "t2"}
	idx := 0
	svc := NewService(repo, func() string {
		id := ids[idx]
		idx++
		return id
	}, func() time.Time { return now }, ServiceConfig{DefaultDeleteMode: DeleteModeArchive})
	if checker != nil {
		svc.gitStatusChecker = checker.check
	}
	project, err := svc.CreateProjectWithMetadata(context.Background(), CreateProjectInput{
		Name:                "Pre-Check Project",
		RepoPrimaryWorktree: worktree,
	})
	if err != nil {
		t.Fatalf("CreateProjectWithMetadata() error = %v", err)
	}
	column, err := svc.CreateColumn(context.Background(), project.ID, "To Do", 0, 0)
	if err != nil {
		t.Fatalf("CreateColumn() error = %v", err)
	}
	return svc, repo, project, column
}

// TestCreateActionItemRejectsDirtyPath asserts the droplet 4b.6 happy
// reject path: one declared path is dirty, the checker reports it, and
// CreateActionItem returns a wrapped ErrPathsDirty whose message names the
// dirty path. No row is written.
func TestCreateActionItemRejectsDirtyPath(t *testing.T) {
	checker := &recordingGitChecker{dirtyPaths: []string{"internal/foo/bar.go"}}
	svc, repo, project, column := newDirtyPathTestService(t, "/tmp/worktree", checker)

	_, err := svc.CreateActionItem(context.Background(), CreateActionItemInput{
		ProjectID:      project.ID,
		ColumnID:       column.ID,
		Title:          "Dirty path build",
		Priority:       domain.PriorityMedium,
		Paths:          []string{"internal/foo/bar.go"},
		Packages:       []string{"internal/foo"},
		StructuralType: domain.StructuralTypeDroplet,
	})
	if !errors.Is(err, ErrPathsDirty) {
		t.Fatalf("CreateActionItem err = %v, want wrapped ErrPathsDirty", err)
	}
	if !strings.Contains(err.Error(), "internal/foo/bar.go") {
		t.Fatalf("err = %v, must name the dirty path 'internal/foo/bar.go'", err)
	}
	if len(checker.calls) != 1 {
		t.Fatalf("checker calls = %d, want 1", len(checker.calls))
	}
	if checker.calls[0].worktree != "/tmp/worktree" {
		t.Fatalf("checker worktree = %q, want /tmp/worktree", checker.calls[0].worktree)
	}
	if len(repo.tasks) != 0 {
		t.Fatalf("repo.tasks = %d, want 0 (creation must reject before persist)", len(repo.tasks))
	}
}

// TestCreateActionItemAcceptsCleanPaths asserts the happy success path:
// the checker reports no dirty paths, CreateActionItem succeeds, and the
// resulting row carries the declared Paths verbatim.
func TestCreateActionItemAcceptsCleanPaths(t *testing.T) {
	checker := &recordingGitChecker{}
	svc, _, project, column := newDirtyPathTestService(t, "/tmp/worktree", checker)

	created, err := svc.CreateActionItem(context.Background(), CreateActionItemInput{
		ProjectID:      project.ID,
		ColumnID:       column.ID,
		Title:          "Clean path build",
		Priority:       domain.PriorityMedium,
		Paths:          []string{"internal/foo/bar.go", "internal/foo/baz.go"},
		Packages:       []string{"internal/foo"},
		StructuralType: domain.StructuralTypeDroplet,
	})
	if err != nil {
		t.Fatalf("CreateActionItem err = %v, want nil", err)
	}
	if len(created.Paths) != 2 {
		t.Fatalf("created.Paths = %v, want 2 entries", created.Paths)
	}
	if len(checker.calls) != 1 {
		t.Fatalf("checker calls = %d, want 1", len(checker.calls))
	}
	gotPaths := checker.calls[0].paths
	if len(gotPaths) != 2 || gotPaths[0] != "internal/foo/bar.go" || gotPaths[1] != "internal/foo/baz.go" {
		t.Fatalf("checker paths = %v, want [bar.go, baz.go]", gotPaths)
	}
}

// TestCreateActionItemRejectsMultipleDirtyPaths asserts that when several
// paths are dirty the wrapped ErrPathsDirty message lists every dirty
// path so the dev can fix them all in one pass. Paths are joined in input
// order so the message is deterministic.
func TestCreateActionItemRejectsMultipleDirtyPaths(t *testing.T) {
	checker := &recordingGitChecker{dirtyPaths: []string{"alpha.go", "gamma.go"}}
	svc, _, project, column := newDirtyPathTestService(t, "/tmp/worktree", checker)

	_, err := svc.CreateActionItem(context.Background(), CreateActionItemInput{
		ProjectID:      project.ID,
		ColumnID:       column.ID,
		Title:          "Multi-dirty build",
		Priority:       domain.PriorityMedium,
		Paths:          []string{"alpha.go", "beta.go", "gamma.go"},
		Packages:       []string{"top"},
		StructuralType: domain.StructuralTypeDroplet,
	})
	if !errors.Is(err, ErrPathsDirty) {
		t.Fatalf("CreateActionItem err = %v, want wrapped ErrPathsDirty", err)
	}
	if !strings.Contains(err.Error(), "alpha.go") {
		t.Fatalf("err = %v, must name alpha.go", err)
	}
	if !strings.Contains(err.Error(), "gamma.go") {
		t.Fatalf("err = %v, must name gamma.go", err)
	}
	if strings.Contains(err.Error(), "beta.go") {
		t.Fatalf("err = %v, must NOT name clean beta.go", err)
	}
}

// TestCreateActionItemHandlesEmptyPaths asserts the degenerate-input fast
// path: when input.Paths is nil/empty, the pre-check is skipped entirely
// (zero checker invocations) and the existing creation path runs unchanged.
// This pins backwards-compatible behavior for every caller that doesn't
// declare write-scope paths.
func TestCreateActionItemHandlesEmptyPaths(t *testing.T) {
	checker := &recordingGitChecker{}
	svc, _, project, column := newDirtyPathTestService(t, "/tmp/worktree", checker)

	_, err := svc.CreateActionItem(context.Background(), CreateActionItemInput{
		ProjectID:      project.ID,
		ColumnID:       column.ID,
		Title:          "No paths build",
		Priority:       domain.PriorityMedium,
		StructuralType: domain.StructuralTypeDroplet,
	})
	if err != nil {
		t.Fatalf("CreateActionItem err = %v, want nil", err)
	}
	if len(checker.calls) != 0 {
		t.Fatalf("checker calls = %d, want 0 (empty Paths must skip pre-check)", len(checker.calls))
	}
}

// TestCreateActionItemSkipsCheckOnEmptyWorktree asserts the pre-MVP escape
// valve from droplet 4b.6 acceptance criterion 2: a project with empty
// RepoPrimaryWorktree (legacy / unbootstrapped) silently skips the check
// rather than blocking creation. This keeps existing test fixtures and
// pre-Drop-4a projects working through the cascade migration.
func TestCreateActionItemSkipsCheckOnEmptyWorktree(t *testing.T) {
	checker := &recordingGitChecker{dirtyPaths: []string{"would-be-dirty.go"}}
	svc, _, project, column := newDirtyPathTestService(t, "", checker)

	_, err := svc.CreateActionItem(context.Background(), CreateActionItemInput{
		ProjectID:      project.ID,
		ColumnID:       column.ID,
		Title:          "Worktree-less build",
		Priority:       domain.PriorityMedium,
		Paths:          []string{"would-be-dirty.go"},
		Packages:       []string{"top"},
		StructuralType: domain.StructuralTypeDroplet,
	})
	if err != nil {
		t.Fatalf("CreateActionItem err = %v, want nil (empty worktree skips check)", err)
	}
	if len(checker.calls) != 0 {
		t.Fatalf("checker calls = %d, want 0 (empty worktree must short-circuit pre-check)", len(checker.calls))
	}
}

// TestCreateActionItemEnvIsolatesFromBareRoot asserts that the production
// pre-check does not honor an inherited GIT_DIR pointed at a bogus path —
// the env-isolation contract from gitdiff round-3 must hold for the
// CreateActionItem hot path. Uses the production defaultGitStatusChecker
// (NOT the stub) against a real fixture-built worktree, with t.Setenv
// pre-poisoning GIT_DIR / GIT_INDEX_FILE the way a `git push` pre-push
// hook would.
func TestCreateActionItemEnvIsolatesFromBareRoot(t *testing.T) {
	// No t.Parallel — t.Setenv requires non-parallel test.
	if _, err := exec.LookPath("git"); err != nil {
		t.Skipf("git not on PATH: %v", err)
	}
	fx := newGitStatusFixture(t)
	fx.commit("internal/foo/bar.go", "package foo\n", "add bar")
	fx.dirty("internal/foo/bar.go", "package foo\n\n// modified\n")

	t.Setenv("GIT_DIR", filepath.Join(t.TempDir(), "bogus.git"))
	t.Setenv("GIT_INDEX_FILE", filepath.Join(t.TempDir(), "bogus.index"))

	// Wire the production checker (no stub) — the env-isolation behavior
	// is exactly what we want to exercise.
	svc, _, project, column := newDirtyPathTestService(t, fx.root, nil)

	_, err := svc.CreateActionItem(context.Background(), CreateActionItemInput{
		ProjectID:      project.ID,
		ColumnID:       column.ID,
		Title:          "Env-isolated build",
		Priority:       domain.PriorityMedium,
		Paths:          []string{"internal/foo/bar.go"},
		Packages:       []string{"internal/foo"},
		StructuralType: domain.StructuralTypeDroplet,
	})
	if !errors.Is(err, ErrPathsDirty) {
		t.Fatalf("CreateActionItem err = %v, want wrapped ErrPathsDirty (env isolation must keep checker on fx.root, not GIT_DIR)", err)
	}
	if !strings.Contains(err.Error(), "internal/foo/bar.go") {
		t.Fatalf("err = %v, must name internal/foo/bar.go", err)
	}
}

// TestCreateActionItemRejectsMalformedPathsBeforePreCheck pins the
// ordering invariant from droplet 4b.6 round-2 QA-Falsification finding
// C1: domain.NewActionItem (which calls normalizeActionItemPaths) MUST
// run BEFORE runGitStatusPreCheck so a malformed input like Paths=[""]
// rejects with domain.ErrInvalidPaths rather than reaching `git status
// --porcelain -- ""`. The recordingGitChecker proves the pre-check
// never fired by asserting len(checker.calls) == 0.
//
// The fixture supplies one valid path AND one empty entry — the empty
// entry is the malformed input. normalizeActionItemPaths short-circuits
// at the first empty/whitespace/backslash entry per action_item.go:713.
func TestCreateActionItemRejectsMalformedPathsBeforePreCheck(t *testing.T) {
	checker := &recordingGitChecker{}
	svc, repo, project, column := newDirtyPathTestService(t, "/tmp/worktree", checker)

	_, err := svc.CreateActionItem(context.Background(), CreateActionItemInput{
		ProjectID:      project.ID,
		ColumnID:       column.ID,
		Title:          "Malformed paths build",
		Priority:       domain.PriorityMedium,
		Paths:          []string{"internal/foo/bar.go", ""},
		Packages:       []string{"internal/foo"},
		StructuralType: domain.StructuralTypeDroplet,
	})
	if !errors.Is(err, domain.ErrInvalidPaths) {
		t.Fatalf("CreateActionItem err = %v, want domain.ErrInvalidPaths (domain validation must run before git pre-check)", err)
	}
	if len(checker.calls) != 0 {
		t.Fatalf("checker calls = %d, want 0 (pre-check must not fire on domain-invalid Paths)", len(checker.calls))
	}
	if len(repo.tasks) != 0 {
		t.Fatalf("repo.tasks = %d, want 0 (malformed input must not persist)", len(repo.tasks))
	}
}

// waitForActionItemChangedSinceCursor blocks up to 100 ms for one
// LiveWaitEventActionItemChanged event published AFTER the supplied cursor
// for the supplied projectID. Helper for droplet 4b.8 publisher-addition
// tests; centralizes the broker-cursor + Wait-with-timeout pattern so the
// per-method tests stay focused on the lifecycle method under test.
//
// The cursor is captured BEFORE the lifecycle call so the Wait afterSequence
// value reliably ignores any pre-existing Latest stamp the broker may carry
// from prior fixture wiring (today this is always zero since each test
// constructs a fresh broker; the helper still threads the cursor through to
// keep the contract robust against future fixture sharing).
func waitForActionItemChangedSinceCursor(t *testing.T, broker LiveWaitBroker, projectID string, cursor int64) LiveWaitEvent {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	event, err := broker.Wait(ctx, LiveWaitEventActionItemChanged, projectID, cursor)
	if err != nil {
		t.Fatalf("broker.Wait(LiveWaitEventActionItemChanged, %q) error = %v (publish missing?)", projectID, err)
	}
	if event.Type != LiveWaitEventActionItemChanged {
		t.Fatalf("event Type = %q, want %q", event.Type, LiveWaitEventActionItemChanged)
	}
	if event.Key != projectID {
		t.Fatalf("event Key = %q, want %q", event.Key, projectID)
	}
	if event.Sequence <= cursor {
		t.Fatalf("event Sequence = %d, want > cursor %d", event.Sequence, cursor)
	}
	return event
}

// captureActionItemChangedCursor returns the broker's Latest sequence for the
// LiveWaitEventActionItemChanged event keyed on projectID. Returns 0 when no
// prior event is stamped — the natural starting cursor for a fresh broker.
func captureActionItemChangedCursor(t *testing.T, broker LiveWaitBroker, projectID string) int64 {
	t.Helper()
	event, ok, err := broker.Latest(context.Background(), LiveWaitEventActionItemChanged, projectID)
	if err != nil {
		t.Fatalf("broker.Latest error = %v", err)
	}
	if !ok {
		return 0
	}
	return event.Sequence
}

// TestRestoreActionItemPublishesActionItemChanged asserts that
// Service.RestoreActionItem emits one LiveWaitEventActionItemChanged event
// keyed on the action item's project_id after the successful repo write.
// Droplet 4b.8 / Drop 4a refinement R1 closure: the cascade dispatcher's
// broker subscriber depends on this event firing for every action-item
// write surface so the auto-promotion walker re-walks the tree on restore.
func TestRestoreActionItemPublishesActionItemChanged(t *testing.T) {
	t.Parallel()

	repo := newFakeRepo()
	now := time.Date(2026, 5, 4, 10, 0, 0, 0, time.UTC)
	project, _ := domain.NewProjectFromInput(domain.ProjectInput{ID: "p1", Name: "Inbox"}, now)
	repo.projects[project.ID] = project
	column, _ := domain.NewColumn("c1", project.ID, "To Do", 0, 0, now)
	repo.columns[column.ID] = column
	archived, _ := domain.NewActionItemForTest(domain.ActionItemInput{
		Kind:      domain.KindPlan,
		ID:        "t1",
		ProjectID: project.ID,
		ColumnID:  column.ID,
		Position:  0,
		Title:     "archived item",
		Priority:  domain.PriorityLow,
	}, now)
	archivedAt := now.Add(time.Minute)
	archived.ArchivedAt = &archivedAt
	repo.tasks[archived.ID] = archived

	broker := NewInProcessLiveWaitBroker()
	svc := NewService(repo, nil, func() time.Time { return now.Add(2 * time.Minute) }, ServiceConfig{LiveWaitBroker: broker})

	cursor := captureActionItemChangedCursor(t, broker, project.ID)
	if _, err := svc.RestoreActionItem(context.Background(), archived.ID); err != nil {
		t.Fatalf("RestoreActionItem() error = %v", err)
	}
	waitForActionItemChangedSinceCursor(t, broker, project.ID, cursor)
}

// TestRenameActionItemPublishesActionItemChanged asserts RenameActionItem
// emits one LiveWaitEventActionItemChanged event keyed on the action item's
// project_id after the successful repo write. Droplet 4b.8 / Drop 4a R1.
func TestRenameActionItemPublishesActionItemChanged(t *testing.T) {
	t.Parallel()

	repo := newFakeRepo()
	now := time.Date(2026, 5, 4, 10, 0, 0, 0, time.UTC)
	actionItem, _ := domain.NewActionItemForTest(domain.ActionItemInput{
		Kind:      domain.KindPlan,
		ID:        "t1",
		ProjectID: "p1",
		ColumnID:  "c1",
		Position:  0,
		Title:     "old",
		Priority:  domain.PriorityLow,
	}, now)
	repo.tasks[actionItem.ID] = actionItem

	broker := NewInProcessLiveWaitBroker()
	svc := NewService(repo, nil, func() time.Time { return now.Add(time.Minute) }, ServiceConfig{LiveWaitBroker: broker})

	cursor := captureActionItemChangedCursor(t, broker, actionItem.ProjectID)
	if _, err := svc.RenameActionItem(context.Background(), actionItem.ID, "new title"); err != nil {
		t.Fatalf("RenameActionItem() error = %v", err)
	}
	waitForActionItemChangedSinceCursor(t, broker, actionItem.ProjectID, cursor)
}

// TestDeleteActionItemArchivePublishesActionItemChanged asserts the
// DeleteActionItem(mode=DeleteModeArchive) branch emits one
// LiveWaitEventActionItemChanged event keyed on the action item's
// project_id after the successful repo write. There is no separate
// ArchiveActionItem method post-Drop-4a; the archive lifecycle lives
// inside DeleteActionItem's archive branch — so droplet 4b.8 publishes
// from BOTH the archive branch and the hard-delete branch (a hard delete
// is also an action-item-changed surface the dispatcher subscriber cares
// about). Droplet 4b.8 / Drop 4a R1.
func TestDeleteActionItemArchivePublishesActionItemChanged(t *testing.T) {
	t.Parallel()

	repo := newFakeRepo()
	now := time.Date(2026, 5, 4, 10, 0, 0, 0, time.UTC)
	actionItem, _ := domain.NewActionItemForTest(domain.ActionItemInput{
		Kind:      domain.KindPlan,
		ID:        "t1",
		ProjectID: "p1",
		ColumnID:  "c1",
		Position:  0,
		Title:     "to archive",
		Priority:  domain.PriorityLow,
	}, now)
	repo.tasks[actionItem.ID] = actionItem

	broker := NewInProcessLiveWaitBroker()
	svc := NewService(repo, nil, func() time.Time { return now.Add(time.Minute) }, ServiceConfig{LiveWaitBroker: broker})

	cursor := captureActionItemChangedCursor(t, broker, actionItem.ProjectID)
	if err := svc.DeleteActionItem(context.Background(), actionItem.ID, DeleteModeArchive); err != nil {
		t.Fatalf("DeleteActionItem(archive) error = %v", err)
	}
	waitForActionItemChangedSinceCursor(t, broker, actionItem.ProjectID, cursor)
}

// TestDeleteActionItemHardPublishesActionItemChanged asserts the
// DeleteActionItem(mode=DeleteModeHard) branch emits one
// LiveWaitEventActionItemChanged event after the successful repo delete.
// Hard deletes are state changes the dispatcher subscriber must observe
// (the action item disappears from the tree walk; subscribers re-walk on
// every wakeup so the right action is taken). Droplet 4b.8 / Drop 4a R1.
func TestDeleteActionItemHardPublishesActionItemChanged(t *testing.T) {
	t.Parallel()

	repo := newFakeRepo()
	now := time.Date(2026, 5, 4, 10, 0, 0, 0, time.UTC)
	actionItem, _ := domain.NewActionItemForTest(domain.ActionItemInput{
		Kind:      domain.KindPlan,
		ID:        "t1",
		ProjectID: "p1",
		ColumnID:  "c1",
		Position:  0,
		Title:     "to hard-delete",
		Priority:  domain.PriorityLow,
	}, now)
	repo.tasks[actionItem.ID] = actionItem

	broker := NewInProcessLiveWaitBroker()
	svc := NewService(repo, nil, func() time.Time { return now.Add(time.Minute) }, ServiceConfig{LiveWaitBroker: broker})

	cursor := captureActionItemChangedCursor(t, broker, actionItem.ProjectID)
	if err := svc.DeleteActionItem(context.Background(), actionItem.ID, DeleteModeHard); err != nil {
		t.Fatalf("DeleteActionItem(hard) error = %v", err)
	}
	waitForActionItemChangedSinceCursor(t, broker, actionItem.ProjectID, cursor)
}

// TestReparentActionItemPublishesActionItemChanged asserts ReparentActionItem
// emits one LiveWaitEventActionItemChanged event keyed on the action item's
// project_id after the successful repo write. Droplet 4b.8 / Drop 4a R1.
func TestReparentActionItemPublishesActionItemChanged(t *testing.T) {
	t.Parallel()

	repo := newFakeRepo()
	now := time.Date(2026, 5, 4, 10, 0, 0, 0, time.UTC)
	project, _ := domain.NewProjectFromInput(domain.ProjectInput{ID: "p1", Name: "Inbox"}, now)
	repo.projects[project.ID] = project
	column, _ := domain.NewColumn("c1", project.ID, "To Do", 0, 0, now)
	repo.columns[column.ID] = column
	parent, _ := domain.NewActionItemForTest(domain.ActionItemInput{
		Kind:      domain.KindPlan,
		ID:        "parent",
		ProjectID: project.ID,
		ColumnID:  column.ID,
		Position:  0,
		Title:     "parent",
		Priority:  domain.PriorityMedium,
	}, now)
	child, _ := domain.NewActionItemForTest(domain.ActionItemInput{
		Kind:      domain.KindPlan,
		ID:        "child",
		ProjectID: project.ID,
		ColumnID:  column.ID,
		Position:  1,
		Title:     "child",
		Priority:  domain.PriorityLow,
	}, now)
	repo.tasks[parent.ID] = parent
	repo.tasks[child.ID] = child

	broker := NewInProcessLiveWaitBroker()
	svc := NewService(repo, nil, func() time.Time { return now.Add(2 * time.Minute) }, ServiceConfig{LiveWaitBroker: broker})

	cursor := captureActionItemChangedCursor(t, broker, project.ID)
	if _, err := svc.ReparentActionItem(context.Background(), child.ID, parent.ID); err != nil {
		t.Fatalf("ReparentActionItem() error = %v", err)
	}
	waitForActionItemChangedSinceCursor(t, broker, project.ID, cursor)
}

// listByStateFixture seeds a project with one column per lifecycle state and
// a small set of items per state so the Drop 4c.5 droplet B.2 listing tests
// can drop items at known starting columns / lifecycle values.
type listByStateFixture struct {
	repo    *fakeRepo
	svc     *Service
	project domain.Project
	cols    map[domain.LifecycleState]domain.Column
	now     time.Time
}

func newListByStateFixture(t *testing.T) listByStateFixture {
	t.Helper()
	repo := newFakeRepo()
	now := time.Date(2026, 5, 5, 12, 0, 0, 0, time.UTC)
	project, err := domain.NewProjectFromInput(domain.ProjectInput{ID: "p-listbystate", Name: "List By State Fixture"}, now)
	if err != nil {
		t.Fatalf("NewProjectFromInput() error = %v", err)
	}
	repo.projects[project.ID] = project
	cols := map[domain.LifecycleState]domain.Column{}
	colSpecs := []struct {
		id    string
		name  string
		pos   int
		state domain.LifecycleState
	}{
		{id: "lbs-todo", name: "To Do", pos: 0, state: domain.StateTodo},
		{id: "lbs-progress", name: "In Progress", pos: 1, state: domain.StateInProgress},
		{id: "lbs-complete", name: "Complete", pos: 2, state: domain.StateComplete},
		{id: "lbs-failed", name: "Failed", pos: 3, state: domain.StateFailed},
		{id: "lbs-archived", name: "Archived", pos: 4, state: domain.StateArchived},
	}
	for _, spec := range colSpecs {
		col, err := domain.NewColumn(spec.id, project.ID, spec.name, spec.pos, 0, now)
		if err != nil {
			t.Fatalf("NewColumn(%q) error = %v", spec.name, err)
		}
		repo.columns[col.ID] = col
		cols[spec.state] = col
	}
	svc := NewService(repo, nil, func() time.Time { return now.Add(time.Minute) }, ServiceConfig{})
	return listByStateFixture{repo: repo, svc: svc, project: project, cols: cols, now: now}
}

// seedListByStateItem drops one action item into the requested starting
// state. `archivedAt` non-nil flips the row's ArchivedAt pointer so the
// fixture can exercise the failed+archived cross-axis case.
func (f *listByStateFixture) seedListByStateItem(t *testing.T, id string, state domain.LifecycleState, updatedAt time.Time, archivedAt *time.Time) domain.ActionItem {
	t.Helper()
	col, ok := f.cols[state]
	if !ok {
		t.Fatalf("listByStateFixture: no column for state %q", state)
	}
	item, err := domain.NewActionItemForTest(domain.ActionItemInput{
		Kind:           domain.KindBuild,
		ID:             id,
		ProjectID:      f.project.ID,
		ColumnID:       col.ID,
		Position:       0,
		Title:          "B.2 list by state " + id,
		Priority:       domain.PriorityMedium,
		LifecycleState: state,
		Role:           domain.RoleBuilder,
	}, f.now)
	if err != nil {
		t.Fatalf("NewActionItemForTest(%q) error = %v", id, err)
	}
	item.UpdatedAt = updatedAt
	if archivedAt != nil {
		stamped := *archivedAt
		item.ArchivedAt = &stamped
	}
	f.repo.tasks[item.ID] = item
	return item
}

// TestService_ListActionItemsByState pins the Drop 4c.5 droplet B.2 contract:
// filter the project's action items by lifecycle state, with an
// includeArchived flag that is forced true when state==archived. Sort order
// is UpdatedAt DESC, tie-broken on ID. Empty / unknown state rejects with a
// clear error naming the valid set. Empty projectID rejects with
// ErrInvalidID.
func TestService_ListActionItemsByState(t *testing.T) {
	t.Parallel()

	t.Run("filter by failed returns only failed items, sorted by updated_at desc", func(t *testing.T) {
		t.Parallel()
		f := newListByStateFixture(t)
		// Two failed items, three non-failed items. Failed items get
		// distinct UpdatedAt values so ordering is observable.
		later := f.now.Add(2 * time.Hour)
		earlier := f.now.Add(time.Hour)
		f.seedListByStateItem(t, "f-1-later", domain.StateFailed, later, nil)
		f.seedListByStateItem(t, "f-2-earlier", domain.StateFailed, earlier, nil)
		f.seedListByStateItem(t, "todo-1", domain.StateTodo, later, nil)
		f.seedListByStateItem(t, "progress-1", domain.StateInProgress, later, nil)
		f.seedListByStateItem(t, "complete-1", domain.StateComplete, later, nil)
		got, err := f.svc.ListActionItemsByState(context.Background(), f.project.ID, domain.StateFailed, false)
		if err != nil {
			t.Fatalf("ListActionItemsByState() error = %v", err)
		}
		if len(got) != 2 {
			t.Fatalf("len(got) = %d, want 2; got items = %+v", len(got), got)
		}
		if got[0].ID != "f-1-later" {
			t.Fatalf("got[0].ID = %q, want f-1-later (more recent updated_at sorts first)", got[0].ID)
		}
		if got[1].ID != "f-2-earlier" {
			t.Fatalf("got[1].ID = %q, want f-2-earlier", got[1].ID)
		}
	})

	t.Run("zero failed items yields empty slice (not nil)", func(t *testing.T) {
		t.Parallel()
		f := newListByStateFixture(t)
		f.seedListByStateItem(t, "todo-only", domain.StateTodo, f.now, nil)
		got, err := f.svc.ListActionItemsByState(context.Background(), f.project.ID, domain.StateFailed, false)
		if err != nil {
			t.Fatalf("ListActionItemsByState() error = %v", err)
		}
		if got == nil {
			t.Fatal("expected non-nil empty slice, got nil")
		}
		if len(got) != 0 {
			t.Fatalf("len(got) = %d, want 0", len(got))
		}
	})

	t.Run("unknown state rejects naming the valid set", func(t *testing.T) {
		t.Parallel()
		f := newListByStateFixture(t)
		_, err := f.svc.ListActionItemsByState(context.Background(), f.project.ID, domain.LifecycleState("weird"), false)
		if err == nil {
			t.Fatal("expected error for unknown state, got nil")
		}
		if !strings.Contains(err.Error(), "unknown state") {
			t.Fatalf("error %q missing 'unknown state' phrase", err)
		}
		if !strings.Contains(err.Error(), "todo") || !strings.Contains(err.Error(), "failed") {
			t.Fatalf("error %q does not name the valid set", err)
		}
	})

	t.Run("empty state rejects with required-state hint", func(t *testing.T) {
		t.Parallel()
		f := newListByStateFixture(t)
		_, err := f.svc.ListActionItemsByState(context.Background(), f.project.ID, domain.LifecycleState(""), false)
		if err == nil {
			t.Fatal("expected error for empty state, got nil")
		}
		if !strings.Contains(err.Error(), "state is required") {
			t.Fatalf("error %q missing 'state is required'", err)
		}
	})

	t.Run("empty projectID returns ErrInvalidID", func(t *testing.T) {
		t.Parallel()
		f := newListByStateFixture(t)
		_, err := f.svc.ListActionItemsByState(context.Background(), "", domain.StateFailed, false)
		if err == nil {
			t.Fatal("expected error for empty projectID, got nil")
		}
		if !errors.Is(err, domain.ErrInvalidID) {
			t.Fatalf("expected ErrInvalidID, got %v", err)
		}
	})

	t.Run("state=archived forces includeArchived true", func(t *testing.T) {
		t.Parallel()
		f := newListByStateFixture(t)
		archivedAt := f.now.Add(time.Hour)
		f.seedListByStateItem(t, "arch-1", domain.StateArchived, f.now.Add(time.Hour), &archivedAt)
		// Caller passes includeArchived=false; helper must override.
		got, err := f.svc.ListActionItemsByState(context.Background(), f.project.ID, domain.StateArchived, false)
		if err != nil {
			t.Fatalf("ListActionItemsByState() error = %v", err)
		}
		if len(got) != 1 {
			t.Fatalf("len(got) = %d, want 1 (state=archived must force includeArchived=true)", len(got))
		}
		if got[0].ID != "arch-1" {
			t.Fatalf("got[0].ID = %q, want arch-1", got[0].ID)
		}
	})

	t.Run("failed AND archived item appears once when includeArchived=true (B.2 §F.1)", func(t *testing.T) {
		t.Parallel()
		f := newListByStateFixture(t)
		archivedAt := f.now.Add(time.Hour)
		// One row: state=failed AND archived_at != nil. Spec falsification
		// mitigation #1: filter must not double-count across the two
		// orthogonal axes (state vs archived flag).
		f.seedListByStateItem(t, "failed-and-archived", domain.StateFailed, f.now.Add(2*time.Hour), &archivedAt)
		got, err := f.svc.ListActionItemsByState(context.Background(), f.project.ID, domain.StateFailed, true)
		if err != nil {
			t.Fatalf("ListActionItemsByState() error = %v", err)
		}
		if len(got) != 1 {
			t.Fatalf("len(got) = %d, want exactly 1 (no double-count of failed+archived)", len(got))
		}
		if got[0].ID != "failed-and-archived" {
			t.Fatalf("got[0].ID = %q, want failed-and-archived", got[0].ID)
		}
	})

	t.Run("failed AND archived item omitted when includeArchived=false", func(t *testing.T) {
		t.Parallel()
		f := newListByStateFixture(t)
		archivedAt := f.now.Add(time.Hour)
		f.seedListByStateItem(t, "failed-and-archived", domain.StateFailed, f.now.Add(2*time.Hour), &archivedAt)
		f.seedListByStateItem(t, "failed-only", domain.StateFailed, f.now.Add(time.Hour), nil)
		got, err := f.svc.ListActionItemsByState(context.Background(), f.project.ID, domain.StateFailed, false)
		if err != nil {
			t.Fatalf("ListActionItemsByState() error = %v", err)
		}
		// fakeRepo.ListActionItems honors includeArchived; failed-and-archived
		// is filtered at the repo layer.
		ids := make([]string, 0, len(got))
		for _, item := range got {
			ids = append(ids, item.ID)
		}
		if len(got) != 1 || got[0].ID != "failed-only" {
			t.Fatalf("got ids = %v, want [failed-only] (failed-and-archived must be excluded when includeArchived=false)", ids)
		}
	})

	t.Run("filter by todo returns only todo items", func(t *testing.T) {
		t.Parallel()
		f := newListByStateFixture(t)
		f.seedListByStateItem(t, "todo-1", domain.StateTodo, f.now, nil)
		f.seedListByStateItem(t, "failed-1", domain.StateFailed, f.now, nil)
		got, err := f.svc.ListActionItemsByState(context.Background(), f.project.ID, domain.StateTodo, false)
		if err != nil {
			t.Fatalf("ListActionItemsByState() error = %v", err)
		}
		if len(got) != 1 || got[0].ID != "todo-1" {
			t.Fatalf("got = %+v, want [todo-1]", got)
		}
	})

	t.Run("filter by in_progress returns only in_progress items", func(t *testing.T) {
		t.Parallel()
		f := newListByStateFixture(t)
		f.seedListByStateItem(t, "progress-1", domain.StateInProgress, f.now, nil)
		f.seedListByStateItem(t, "failed-1", domain.StateFailed, f.now, nil)
		got, err := f.svc.ListActionItemsByState(context.Background(), f.project.ID, domain.StateInProgress, false)
		if err != nil {
			t.Fatalf("ListActionItemsByState() error = %v", err)
		}
		if len(got) != 1 || got[0].ID != "progress-1" {
			t.Fatalf("got = %+v, want [progress-1]", got)
		}
	})

	t.Run("state value is case-folded (FAILED → failed)", func(t *testing.T) {
		t.Parallel()
		f := newListByStateFixture(t)
		f.seedListByStateItem(t, "f-1", domain.StateFailed, f.now, nil)
		got, err := f.svc.ListActionItemsByState(context.Background(), f.project.ID, domain.LifecycleState("FAILED"), false)
		if err != nil {
			t.Fatalf("ListActionItemsByState(FAILED) error = %v", err)
		}
		if len(got) != 1 || got[0].ID != "f-1" {
			t.Fatalf("got = %+v, want [f-1]", got)
		}
	})

	t.Run("equal updated_at ties broken on ID for stable order", func(t *testing.T) {
		t.Parallel()
		f := newListByStateFixture(t)
		// Same updated_at so the sort comparator falls through to the
		// ID tie-breaker; b sorts before c by ID compare.
		updated := f.now.Add(time.Hour)
		f.seedListByStateItem(t, "f-c", domain.StateFailed, updated, nil)
		f.seedListByStateItem(t, "f-b", domain.StateFailed, updated, nil)
		got, err := f.svc.ListActionItemsByState(context.Background(), f.project.ID, domain.StateFailed, false)
		if err != nil {
			t.Fatalf("ListActionItemsByState() error = %v", err)
		}
		if len(got) != 2 || got[0].ID != "f-b" || got[1].ID != "f-c" {
			ids := []string{}
			for _, i := range got {
				ids = append(ids, i.ID)
			}
			t.Fatalf("got order = %v, want [f-b f-c] (id-asc tie-breaker)", ids)
		}
	})
}

// TestLoadProjectTemplate_NoOnDiskTemplate covers the post-REFINEMENTS-2026-05-14
// contract: when a project has no on-disk template at any tier (project-tier,
// HOME tier), loadProjectTemplateWithHome returns (zero, false, nil).
// Templates are project-tier opt-in only — no embedded fallback fires.
//
// Each row exercises a path variant that previously fell through to the embedded
// fallback; after the removal, all return ok=false without error.
func TestLoadProjectTemplate_NoOnDiskTemplate(t *testing.T) {
	tests := []struct {
		name    string
		project domain.Project
	}{
		{
			name:    "zero-value project → no template",
			project: domain.Project{},
		},
		{
			name:    "empty paths → no template",
			project: domain.Project{},
		},
		{
			name: "whitespace-only RepoBareRoot trims to empty → no template",
			project: domain.Project{
				RepoBareRoot:        "   ",
				RepoPrimaryWorktree: "\t  ",
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			project := tc.project
			// Route through the testability seam with empty homeDir so the
			// HOME tier is always skipped.
			tpl, ok, err := loadProjectTemplateWithHome(&project, "", "")
			if err != nil {
				t.Fatalf("loadProjectTemplateWithHome(): unexpected error = %v", err)
			}
			if ok {
				t.Fatalf("loadProjectTemplateWithHome(): ok = true; want false (no on-disk template at any tier)")
			}
			if tpl.SchemaVersion != "" {
				t.Fatalf("loadProjectTemplateWithHome(): tpl.SchemaVersion = %q; want \"\" (zero-value Template when ok=false)", tpl.SchemaVersion)
			}
		})
	}
}

// TestLoadProjectTemplate_NilProjectReturnsSkip covers the nil-guard
// branch added by Drop 4c.5 droplet F.1.1's defensive contract. Direct
// callers (bakeProjectKindCatalog already nil-checks) get the same
// "skip template binding" behavior as the pre-F.1.1 stub when invoked
// with a nil project, so an accidental wiring mistake does not panic.
func TestLoadProjectTemplate_NilProjectReturnsSkip(t *testing.T) {
	tpl, ok, err := loadProjectTemplate(nil)
	if err != nil {
		t.Fatalf("loadProjectTemplate(nil): unexpected error = %v", err)
	}
	if ok {
		t.Fatalf("loadProjectTemplate(nil): ok = true; want false (nil-guard skip)")
	}
	if tpl.SchemaVersion != "" {
		t.Fatalf("loadProjectTemplate(nil): tpl.SchemaVersion = %q; want \"\" (zero-value Template on skip)", tpl.SchemaVersion)
	}
}

// mustReadDefaultGoTOML reads the on-disk byte content of the embedded
// Go-flavored builtin (post-Drop-4c.6 W5.D1: `till-go.toml`) so F.1.2 walk
// fixtures can author valid v1 templates without inlining ~hundred lines of
// TOML per test. The path is relative to the test working directory (Go's
// `testing` package runs each test with the package directory as cwd), so
// `../templates/...` resolves to internal/templates/builtin/till-go.toml
// regardless of where the test binary was built. Failure to read is a hard
// test failure — the file MUST exist post-W5.D1 rename. The helper name is
// retained (rather than renamed to `mustReadTillGoTOML`) to keep the
// caller-audit footprint of W5.D1 minimal; renaming the helper would touch
// every test that uses it.
func mustReadDefaultGoTOML(t *testing.T) []byte {
	t.Helper()
	path := filepath.Join("..", "templates", "builtin", "till-go.toml")
	bytes, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read till-go.toml at %s: %v", path, err)
	}
	return bytes
}

// withTillsynMarker returns a copy of base TOML with a `[tillsyn]` table
// appended carrying max_context_bundle_chars set to marker. Used by
// F.1.2 walk fixtures to author candidate templates whose loaded
// Tillsyn.MaxContextBundleChars uniquely identifies which on-disk
// candidate Load consumed. Validity is preserved — Tillsyn fields are
// optional per templates.schema.go § Tillsyn doc-comment, and a
// non-zero positive int passes validateTillsyn.
//
// Pre-condition: base must NOT already contain a `[tillsyn]` table
// (the embedded till-go.toml does not, as of F.2.1; rebadged from
// default-go.toml in Drop 4c.6 W5.D1). If a future drop adds one to
// till-go.toml, this helper must be reworked to in-place mutate rather
// than append.
func withTillsynMarker(base []byte, marker int) []byte {
	suffix := fmt.Sprintf("\n[tillsyn]\nmax_context_bundle_chars = %d\n", marker)
	out := make([]byte, 0, len(base)+len(suffix))
	out = append(out, base...)
	out = append(out, []byte(suffix)...)
	return out
}

// writeProjectTemplateFixture creates <root>/.tillsyn/template.toml with
// the supplied content and fails the test on I/O error. Returns the
// directory root (echoed for caller convenience) so the test can pass
// it directly to domain.Project.RepoBareRoot or RepoPrimaryWorktree.
func writeProjectTemplateFixture(t *testing.T, root string, content []byte) string {
	t.Helper()
	dir := filepath.Join(root, ".tillsyn")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", dir, err)
	}
	if err := os.WriteFile(filepath.Join(dir, "template.toml"), content, 0o644); err != nil {
		t.Fatalf("write template.toml under %s: %v", dir, err)
	}
	return root
}

// writeHomeTemplateFixture creates <homeDir>/.tillsyn/templates/<group>.toml
// with the supplied content and fails the test on I/O error. Returns homeDir
// (echoed for caller convenience). Used by TestLoadProjectTemplate_HomeTier
// cases to seed the HOME tier candidate without touching the test-process's
// real HOME directory.
func writeHomeTemplateFixture(t *testing.T, homeDir, group string, content []byte) string {
	t.Helper()
	dir := filepath.Join(homeDir, ".tillsyn", "templates")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", dir, err)
	}
	if err := os.WriteFile(filepath.Join(dir, group+".toml"), content, 0o644); err != nil {
		t.Fatalf("write %s.toml under %s: %v", group, dir, err)
	}
	return homeDir
}

// TestLoadProjectTemplate_HomeTier covers D1 (Drop 4c.6.1.W1.D1) acceptance
// criteria: the HOME tier is the third candidate in the 4-tier walk
// (bare-root → primary-worktree → HOME → embedded). Tests call
// loadProjectTemplateWithHome directly with a t.TempDir() fake homeDir so
// the real $HOME is never consulted.
func TestLoadProjectTemplate_HomeTier(t *testing.T) {
	const homeMarker = 5555
	base := mustReadDefaultGoTOML(t)

	tests := []struct {
		name string
		fn   func(t *testing.T)
	}{
		{
			// AC #1 + #3: when HOME file exists it is used in preference to
			// the embedded default. No bare-root or primary-worktree
			// candidate is present so the walk reaches the HOME tier.
			name: "HOME file exists is used before embedded fallback",
			fn: func(t *testing.T) {
				fakeHome := writeHomeTemplateFixture(t, t.TempDir(), "go", withTillsynMarker(base, homeMarker))
				project := domain.Project{}
				tpl, ok, err := loadProjectTemplateWithHome(&project, fakeHome, "go")
				if err != nil {
					t.Fatalf("loadProjectTemplateWithHome(): unexpected error = %v", err)
				}
				if !ok {
					t.Fatal("loadProjectTemplateWithHome(): ok = false; want true (HOME candidate exists)")
				}
				if tpl.Tillsyn.MaxContextBundleChars != homeMarker {
					t.Fatalf("Tillsyn.MaxContextBundleChars = %d; want %d (HOME tier must win over embedded)", tpl.Tillsyn.MaxContextBundleChars, homeMarker)
				}
			},
		},
		{
			// Post-REFINEMENTS-2026-05-14: when the HOME file is absent and no
			// repo-path candidates exist, returns (zero, false, nil) — no
			// embedded fallback. homeDir is a real tempdir but contains no
			// .tillsyn/templates/go.toml.
			name: "HOME file absent returns no template",
			fn: func(t *testing.T) {
				fakeHome := t.TempDir() // real dir; no .tillsyn/templates/ inside
				project := domain.Project{}
				tpl, ok, err := loadProjectTemplateWithHome(&project, fakeHome, "go")
				if err != nil {
					t.Fatalf("loadProjectTemplateWithHome(): unexpected error = %v", err)
				}
				if ok {
					t.Fatal("loadProjectTemplateWithHome(): ok = true; want false (no on-disk template at any tier)")
				}
				if tpl.SchemaVersion != "" {
					t.Fatalf("loadProjectTemplateWithHome(): tpl.SchemaVersion = %q; want \"\" (zero-value when ok=false)", tpl.SchemaVersion)
				}
			},
		},
		{
			// AC #4: if the HOME file EXISTS but templates.Load rejects it
			// (malformed TOML), the error propagates; the walk does NOT fall
			// through to the embedded default.
			name: "HOME file malformed error propagates",
			fn: func(t *testing.T) {
				fakeHome := writeHomeTemplateFixture(t, t.TempDir(), "go", []byte("schema_version = \"v1\"\nunknown_key = \"boom\"\n"))
				project := domain.Project{}
				tpl, ok, err := loadProjectTemplateWithHome(&project, fakeHome, "go")
				if err == nil {
					t.Fatal("loadProjectTemplateWithHome(): err = nil; want wrapped ErrUnknownTemplateKey from HOME candidate")
				}
				if !errors.Is(err, templates.ErrUnknownTemplateKey) {
					t.Fatalf("loadProjectTemplateWithHome(): err = %v; want errors.Is(templates.ErrUnknownTemplateKey)", err)
				}
				if ok {
					t.Fatal("loadProjectTemplateWithHome(): ok = true on error; want false")
				}
				if tpl.SchemaVersion != "" {
					t.Fatalf("loadProjectTemplateWithHome(): tpl non-zero on error: %+v", tpl)
				}
			},
		},
		{
			// Post-REFINEMENTS-2026-05-14: both RepoBareRoot and
			// RepoPrimaryWorktree are empty, HOME file absent. Walk has no
			// on-disk candidates for any tier; returns (zero, false, nil) —
			// no embedded fallback.
			name: "empty worktree paths and no HOME file returns no template",
			fn: func(t *testing.T) {
				fakeHome := t.TempDir() // no .tillsyn/templates/ inside
				project := domain.Project{}
				tpl, ok, err := loadProjectTemplateWithHome(&project, fakeHome, "go")
				if err != nil {
					t.Fatalf("loadProjectTemplateWithHome(): unexpected error = %v", err)
				}
				if ok {
					t.Fatal("loadProjectTemplateWithHome(): ok = true; want false (no on-disk template at any tier)")
				}
				if tpl.SchemaVersion != "" {
					t.Fatalf("loadProjectTemplateWithHome(): tpl.SchemaVersion = %q; want \"\" (zero-value when ok=false)", tpl.SchemaVersion)
				}
			},
		},
		{
			// Post-REFINEMENTS-2026-05-14: empty group ("") causes the HOME-tier
			// candidate construction to be skipped. No repo-path candidates
			// exist either. Returns (zero, false, nil) — no embedded fallback.
			name: "empty-group skip — HOME tier not constructed, no template",
			fn: func(t *testing.T) {
				fakeHome := t.TempDir() // real dir; guard fires on empty group, not on homeDir
				project := domain.Project{}
				tpl, ok, err := loadProjectTemplateWithHome(&project, fakeHome, "")
				if err != nil {
					t.Fatalf("loadProjectTemplateWithHome(): unexpected error = %v", err)
				}
				if ok {
					t.Fatal("loadProjectTemplateWithHome(): ok = true; want false (no on-disk template, group empty)")
				}
				if tpl.SchemaVersion != "" {
					t.Fatalf("loadProjectTemplateWithHome(): tpl.SchemaVersion = %q; want \"\" (zero-value when ok=false)", tpl.SchemaVersion)
				}
			},
		},
		{
			// Post-REFINEMENTS-2026-05-14: empty homeDir causes the HOME-tier
			// candidate construction to be skipped. No repo-path candidates
			// exist. Returns (zero, false, nil) — no embedded fallback.
			name: "empty-homeDir skip — HOME tier not constructed, no template",
			fn: func(t *testing.T) {
				project := domain.Project{}
				tpl, ok, err := loadProjectTemplateWithHome(&project, "", "go")
				if err != nil {
					t.Fatalf("loadProjectTemplateWithHome(): unexpected error = %v", err)
				}
				if ok {
					t.Fatal("loadProjectTemplateWithHome(): ok = true; want false (no on-disk template, homeDir empty)")
				}
				if tpl.SchemaVersion != "" {
					t.Fatalf("loadProjectTemplateWithHome(): tpl.SchemaVersion = %q; want \"\" (zero-value when ok=false)", tpl.SchemaVersion)
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, tt.fn)
	}
}

// TestLoadProjectTemplate_BareRootWins covers Drop 4c.5 droplet F.1.2's
// core acceptance criterion #1.1 + #2: when both bare-root and
// primary-worktree carry an on-disk template, the bare-root candidate
// wins (priority order: bare → primary → embedded). Distinct
// max_context_bundle_chars markers in the two fixtures let the
// assertion verify which file Load actually consumed.
func TestLoadProjectTemplate_BareRootWins(t *testing.T) {
	const bareMarker = 7777
	const primaryMarker = 8888
	base := mustReadDefaultGoTOML(t)
	bareRoot := writeProjectTemplateFixture(t, t.TempDir(), withTillsynMarker(base, bareMarker))
	primaryWorktree := writeProjectTemplateFixture(t, t.TempDir(), withTillsynMarker(base, primaryMarker))
	project := domain.Project{
		RepoBareRoot:        bareRoot,
		RepoPrimaryWorktree: primaryWorktree,
	}
	tpl, ok, err := loadProjectTemplate(&project)
	if err != nil {
		t.Fatalf("loadProjectTemplate(): unexpected error = %v", err)
	}
	if !ok {
		t.Fatalf("loadProjectTemplate(): ok = false; want true (bare-root candidate exists)")
	}
	if tpl.Tillsyn.MaxContextBundleChars != bareMarker {
		t.Fatalf("Tillsyn.MaxContextBundleChars = %d; want %d (bare-root must win over primary-worktree)", tpl.Tillsyn.MaxContextBundleChars, bareMarker)
	}
}

// TestLoadProjectTemplate_PrimaryWorktreeFallback covers Drop 4c.5
// droplet F.1.2 acceptance criterion #1.2: when bare-root is non-empty
// but the file is absent, the walk falls through to primary-worktree.
// The bare-root directory exists and is non-empty (RepoBareRoot points
// to a real path) — only the .tillsyn/template.toml file is missing.
func TestLoadProjectTemplate_PrimaryWorktreeFallback(t *testing.T) {
	const primaryMarker = 8888
	base := mustReadDefaultGoTOML(t)
	bareRoot := t.TempDir() // Real directory; no .tillsyn/template.toml inside.
	primaryWorktree := writeProjectTemplateFixture(t, t.TempDir(), withTillsynMarker(base, primaryMarker))
	project := domain.Project{
		RepoBareRoot:        bareRoot,
		RepoPrimaryWorktree: primaryWorktree,
	}
	tpl, ok, err := loadProjectTemplate(&project)
	if err != nil {
		t.Fatalf("loadProjectTemplate(): unexpected error = %v", err)
	}
	if !ok {
		t.Fatalf("loadProjectTemplate(): ok = false; want true (primary-worktree candidate exists)")
	}
	if tpl.Tillsyn.MaxContextBundleChars != primaryMarker {
		t.Fatalf("Tillsyn.MaxContextBundleChars = %d; want %d (primary-worktree fallback must load when bare-root file absent)", tpl.Tillsyn.MaxContextBundleChars, primaryMarker)
	}
}

// TestLoadProjectTemplate_BareRootSyntaxErrorPropagates covers Drop 4c.5
// droplet F.1.2 acceptance criterion #3 + falsification mitigation #2:
// when a candidate file EXISTS but templates.Load rejects it (here, a
// strict-decode unknown-key rejection), the error PROPAGATES wrapped
// with the offending path; the walk does NOT fall through to the
// primary-worktree candidate. Silent fall-through would hide typos in
// dev-authored templates, which is the explicit non-goal F.1.2's spec
// names.
//
// The path-wrap format `template at <abs-path>: <wrapped>` lets
// downstream callers continue routing on `errors.Is` against templates
// package sentinels (asserted via templates.ErrUnknownTemplateKey
// below) AND see the offending path in the error string (asserted via
// strings.Contains).
func TestLoadProjectTemplate_BareRootSyntaxErrorPropagates(t *testing.T) {
	// Malformed template — schema_version is correct but a top-level
	// unknown key trips strict decode → ErrUnknownTemplateKey. The
	// primary-worktree fixture is valid; if fall-through happened, the
	// test would falsely succeed.
	bareRoot := writeProjectTemplateFixture(t, t.TempDir(), []byte("schema_version = \"v1\"\nunknown_top_key = \"oops\"\n"))
	const primaryMarker = 8888
	primaryWorktree := writeProjectTemplateFixture(t, t.TempDir(), withTillsynMarker(mustReadDefaultGoTOML(t), primaryMarker))
	project := domain.Project{
		RepoBareRoot:        bareRoot,
		RepoPrimaryWorktree: primaryWorktree,
	}
	tpl, ok, err := loadProjectTemplate(&project)
	if err == nil {
		t.Fatalf("loadProjectTemplate(): err = nil; want wrapped ErrUnknownTemplateKey from bare-root candidate")
	}
	if !errors.Is(err, templates.ErrUnknownTemplateKey) {
		t.Fatalf("loadProjectTemplate(): err = %v; want errors.Is(templates.ErrUnknownTemplateKey)", err)
	}
	expectedPath := filepath.Join(bareRoot, ".tillsyn", "template.toml")
	if !strings.Contains(err.Error(), expectedPath) {
		t.Fatalf("loadProjectTemplate(): err = %q; want substring %q (path-wrap surfaces offending file)", err.Error(), expectedPath)
	}
	if ok {
		t.Fatalf("loadProjectTemplate(): ok = true on error; want false")
	}
	if tpl.SchemaVersion != "" {
		t.Fatalf("loadProjectTemplate(): tpl = %+v; want zero-value Template on error", tpl)
	}
	// Falsification check: if the walk fell through to primary, the
	// returned tpl would have the primary marker. We already asserted
	// SchemaVersion=="" and ok=false above, so the marker would be
	// zero anyway — but assert explicitly for clarity.
	if tpl.Tillsyn.MaxContextBundleChars == primaryMarker {
		t.Fatalf("loadProjectTemplate(): primary-worktree fallback was consulted on bare-root error; want propagation without fall-through")
	}
}

// TestLoadProjectTemplate_BothAbsentNoTemplate covers the
// post-REFINEMENTS-2026-05-14 contract: when both repo-path fields are
// non-empty but neither carries an on-disk .tillsyn/template.toml, and the
// HOME tier is skipped, loadProjectTemplateWithHome returns (zero, false, nil)
// — no embedded fallback. Templates are project-tier opt-in only.
func TestLoadProjectTemplate_BothAbsentNoTemplate(t *testing.T) {
	bareRoot := t.TempDir()
	primaryWorktree := t.TempDir()
	project := domain.Project{
		RepoBareRoot:        bareRoot,
		RepoPrimaryWorktree: primaryWorktree,
	}
	// Route through the testability seam with empty homeDir so the HOME tier
	// is always skipped.
	tpl, ok, err := loadProjectTemplateWithHome(&project, "", "go")
	if err != nil {
		t.Fatalf("loadProjectTemplateWithHome(): unexpected error = %v", err)
	}
	if ok {
		t.Fatalf("loadProjectTemplateWithHome(): ok = true; want false (no on-disk template at any tier)")
	}
	if tpl.SchemaVersion != "" {
		t.Fatalf("loadProjectTemplateWithHome(): tpl.SchemaVersion = %q; want \"\" (zero-value when ok=false)", tpl.SchemaVersion)
	}
}

// TestLoadProjectTemplate_RelativePathSafety covers Drop 4c.5 droplet
// F.1.2 spec falsification mitigation #1: empty RepoBareRoot must NOT
// cause loadProjectTemplate to open `.tillsyn/template.toml` relative
// to the process CWD. Without the early-empty-skip guard,
// filepath.Join("", ".tillsyn", "template.toml") produces the relative
// path, which os.Open then resolves against CWD — leaking
// CWD-dependent behavior into project create.
//
// Test technique: t.Chdir into a tempdir that contains a
// `.tillsyn/template.toml` with a unique marker. If the implementation
// accidentally opened the relative path, the test would return ok=true
// (the file exists in CWD). Post-REFINEMENTS-2026-05-14, empty-path
// projects return (zero, false, nil) — no embedded fallback.
func TestLoadProjectTemplate_RelativePathSafety(t *testing.T) {
	const cwdMarker = 9999
	cwdTrap := t.TempDir()
	writeProjectTemplateFixture(t, cwdTrap, withTillsynMarker(mustReadDefaultGoTOML(t), cwdMarker))
	t.Chdir(cwdTrap)
	// Empty RepoBareRoot AND empty RepoPrimaryWorktree — must skip both
	// candidate lookups (no relative-path os.Open) and return (zero, false, nil).
	project := domain.Project{}
	tpl, ok, err := loadProjectTemplateWithHome(&project, "", "go")
	if err != nil {
		t.Fatalf("loadProjectTemplateWithHome(): unexpected error = %v", err)
	}
	if ok {
		// If ok=true, the CWD-relative file was (incorrectly) opened.
		t.Fatalf("loadProjectTemplateWithHome(): ok = true; want false (empty paths must NOT open CWD-relative .tillsyn/template.toml; cwd trap marker=%d, got MaxContextBundleChars=%d)",
			cwdMarker, tpl.Tillsyn.MaxContextBundleChars)
	}
	if tpl.SchemaVersion != "" {
		t.Fatalf("loadProjectTemplateWithHome(): tpl.SchemaVersion = %q; want \"\" (zero-value when ok=false)", tpl.SchemaVersion)
	}
}

// TestLoadProjectTemplate_NoOnDiskTemplateAnyLanguage covers the
// post-REFINEMENTS-2026-05-14 contract: language value is irrelevant when
// no on-disk template exists — loadProjectTemplateWithHome returns
// (zero, false, nil) regardless of Language. The embedded language resolver
// is no longer consulted; unsupported language values like "rust" are
// silently accepted without error.
func TestLoadProjectTemplate_NoOnDiskTemplateAnyLanguage(t *testing.T) {
	// Route through the testability seam with empty homeDir so the HOME tier
	// is always skipped.
	project := domain.Project{}
	tpl, ok, err := loadProjectTemplateWithHome(&project, "", "rust")
	if err != nil {
		t.Fatalf("loadProjectTemplateWithHome(): unexpected error = %v; want nil (no embedded resolver called)", err)
	}
	if ok {
		t.Fatalf("loadProjectTemplateWithHome(): ok = true; want false (no on-disk template at any tier)")
	}
	if tpl.SchemaVersion != "" {
		t.Fatalf("loadProjectTemplateWithHome(): tpl = %+v; want zero-value Template when ok=false", tpl)
	}
}

// writeHomeGroupTemplateFixture writes <homeDir>/.tillsyn/templates/<group>.toml
// with a minimal valid v1 template containing one AgentBinding entry. Used
// exclusively by TestBakeProjectKindCatalog_MultiGroup to seed per-group HOME
// tier candidates without touching the test-process's real HOME directory.
//
// The fixture is a self-contained minimal TOML (NOT derived from till-go.toml)
// so it does not collide with any existing [agent_bindings.<kind>] tables that
// till-go.toml already declares. The [tillsyn] max_context_bundle_chars marker
// uniquely identifies which group's template was loaded; the single
// [agent_bindings.<kindKey>] entry drives per-key last-group-wins assertions.
func writeHomeGroupTemplateFixture(t *testing.T, homeDir, group string, markerChars int, kindKey domain.Kind, agentName string) string {
	t.Helper()
	content := []byte(fmt.Sprintf(
		"schema_version = \"v1\"\n\n[tillsyn]\nmax_context_bundle_chars = %d\n\n[agent_bindings.%s]\nagent_name = %q\nmodel = \"sonnet\"\nmax_tries = 3\nmax_budget_usd = 0.0\nmax_turns = 50\n",
		markerChars, kindKey, agentName,
	))
	return writeHomeTemplateFixture(t, homeDir, group, content)
}

// TestBakeProjectKindCatalog_MultiGroup covers Drop 4c.6.1 W1.D2 acceptance
// criteria for the multi-group branch in bakeProjectKindCatalog:
//
//   - AC5: when Groups non-empty, bakeProjectKindCatalog routes to
//     loadProjectTemplatesForGroups instead of loadProjectTemplate.
//   - AC3: per-group loadProjectTemplateWithHome + mergeTemplates aggregation.
//   - AC4 AgentBindings: per-key last-group-wins collision resolution.
//   - AC3 empty-group guard: empty-string entries in Groups are skipped.
//
// Tests call bakeProjectKindCatalogWithHome (the testability seam) with a
// fake homeDir so the real $HOME is never consulted.
func TestBakeProjectKindCatalog_MultiGroup(t *testing.T) {
	cases := []struct {
		name string
		fn   func(t *testing.T)
	}{
		{
			// AC5 + AC3 (a): 2 groups, both HOME files present.
			// AgentBindings for each group carry a different kind key so
			// both are present in the merged catalog (no collision).
			name: "both_groups_present_aggregated",
			fn: func(t *testing.T) {
				fakeHome := t.TempDir()
				writeHomeGroupTemplateFixture(t, fakeHome, "go", 1001, domain.KindBuild, "builder-agent")
				writeHomeGroupTemplateFixture(t, fakeHome, "fe", 1002, domain.KindResearch, "research-agent")
				project := domain.Project{
					Metadata: domain.ProjectMetadata{Groups: []string{"go", "fe"}},
				}
				if err := bakeProjectKindCatalogWithHome(&project, fakeHome); err != nil {
					t.Fatalf("bakeProjectKindCatalogWithHome(): unexpected error = %v", err)
				}
				if len(project.KindCatalogJSON) == 0 {
					t.Fatal("KindCatalogJSON is empty after multi-group bake")
				}
				var catalog templates.KindCatalog
				if err := json.Unmarshal(project.KindCatalogJSON, &catalog); err != nil {
					t.Fatalf("json.Unmarshal(KindCatalogJSON): error = %v", err)
				}
				// Both groups contributed an AgentBinding entry under different kinds.
				if _, ok := catalog.AgentBindings[domain.KindBuild]; !ok {
					t.Fatal("catalog.AgentBindings missing KindBuild (go group contribution)")
				}
				if _, ok := catalog.AgentBindings[domain.KindResearch]; !ok {
					t.Fatal("catalog.AgentBindings missing KindResearch (fe group contribution)")
				}
				if catalog.AgentBindings[domain.KindBuild].AgentName != "builder-agent" {
					t.Fatalf("AgentBindings[build].AgentName = %q; want %q", catalog.AgentBindings[domain.KindBuild].AgentName, "builder-agent")
				}
				if catalog.AgentBindings[domain.KindResearch].AgentName != "research-agent" {
					t.Fatalf("AgentBindings[research].AgentName = %q; want %q", catalog.AgentBindings[domain.KindResearch].AgentName, "research-agent")
				}
			},
		},
		{
			// Post-REFINEMENTS-2026-05-14: 2 groups, one HOME file absent.
			// Absent group is skipped (no embedded fallback). Present group
			// contributes its template; the catalog is populated from the
			// present group only.
			name: "one_group_absent_skipped_present_group_contributes",
			fn: func(t *testing.T) {
				fakeHome := t.TempDir()
				// Only "go" has a HOME file; "fe" is absent.
				writeHomeGroupTemplateFixture(t, fakeHome, "go", 2001, domain.KindBuild, "builder-agent")
				// No file for "fe" group.
				project := domain.Project{
					Metadata: domain.ProjectMetadata{Groups: []string{"go", "fe"}},
				}
				if err := bakeProjectKindCatalogWithHome(&project, fakeHome); err != nil {
					t.Fatalf("bakeProjectKindCatalogWithHome(): unexpected error = %v (absent HOME file must not error)", err)
				}
				if len(project.KindCatalogJSON) == 0 {
					t.Fatal("KindCatalogJSON is empty; present go group must contribute a catalog")
				}
				var catalog templates.KindCatalog
				if err := json.Unmarshal(project.KindCatalogJSON, &catalog); err != nil {
					t.Fatalf("json.Unmarshal(KindCatalogJSON): error = %v", err)
				}
				// go group contributed its AgentBinding entry.
				if _, ok := catalog.AgentBindings[domain.KindBuild]; !ok {
					t.Fatal("catalog.AgentBindings missing KindBuild (go group contribution)")
				}
			},
		},
		{
			// AC4 AgentBindings (c): collision on same kind key → last group wins.
			// Both "go" and "fe" groups write an AgentBinding under KindBuild but
			// with different model strings so the test can verify which entry won.
			// "fe" is the second (later) group, so its entry's model must win.
			// Using different models (not different agent names) because agent name
			// must be an embedded-floor-resolvable value and both groups use the
			// same valid "builder-agent". The model string is unconstrained and
			// uniquely identifies each group's contribution.
			name: "collision_same_kind_key_last_group_wins",
			fn: func(t *testing.T) {
				fakeHome := t.TempDir()
				// Write "go" group with model "sonnet" (first group).
				goContent := []byte(
					"schema_version = \"v1\"\n\n" +
						"[agent_bindings.build]\nagent_name = \"builder-agent\"\nmodel = \"sonnet\"\nmax_tries = 3\nmax_budget_usd = 0.0\nmax_turns = 50\n",
				)
				writeHomeTemplateFixture(t, fakeHome, "go", goContent)
				// Write "fe" group with model "opus" (second group — must win).
				feContent := []byte(
					"schema_version = \"v1\"\n\n" +
						"[agent_bindings.build]\nagent_name = \"builder-agent\"\nmodel = \"opus\"\nmax_tries = 3\nmax_budget_usd = 0.0\nmax_turns = 50\n",
				)
				writeHomeTemplateFixture(t, fakeHome, "fe", feContent)
				project := domain.Project{
					Metadata: domain.ProjectMetadata{Groups: []string{"go", "fe"}},
				}
				if err := bakeProjectKindCatalogWithHome(&project, fakeHome); err != nil {
					t.Fatalf("bakeProjectKindCatalogWithHome(): unexpected error = %v", err)
				}
				var catalog templates.KindCatalog
				if err := json.Unmarshal(project.KindCatalogJSON, &catalog); err != nil {
					t.Fatalf("json.Unmarshal(KindCatalogJSON): error = %v", err)
				}
				// "fe" is last in Groups; its AgentBinding model must win.
				if catalog.AgentBindings[domain.KindBuild].Model != "opus" {
					t.Fatalf("AgentBindings[build].Model = %q; want %q (last group wins)", catalog.AgentBindings[domain.KindBuild].Model, "opus")
				}
			},
		},
		{
			// AC3 empty-group guard (d): empty-string in Groups is skipped.
			// The bake must not error and must produce a valid catalog from
			// the non-empty group entries (or embedded fallback if none match).
			name: "empty_string_in_groups_skipped_without_error",
			fn: func(t *testing.T) {
				fakeHome := t.TempDir()
				writeHomeGroupTemplateFixture(t, fakeHome, "go", 4001, domain.KindBuild, "builder-agent")
				// Groups contains an empty string that must be skipped.
				project := domain.Project{
					Metadata: domain.ProjectMetadata{Groups: []string{"go", "", "  "}},
				}
				if err := bakeProjectKindCatalogWithHome(&project, fakeHome); err != nil {
					t.Fatalf("bakeProjectKindCatalogWithHome(): unexpected error = %v (empty-string groups must be silently skipped)", err)
				}
				if len(project.KindCatalogJSON) == 0 {
					t.Fatal("KindCatalogJSON is empty after skipping empty groups")
				}
				var catalog templates.KindCatalog
				if err := json.Unmarshal(project.KindCatalogJSON, &catalog); err != nil {
					t.Fatalf("json.Unmarshal(KindCatalogJSON): error = %v", err)
				}
				// "go" was the only non-empty group; its entry must be present.
				if _, ok := catalog.AgentBindings[domain.KindBuild]; !ok {
					t.Fatal("catalog.AgentBindings missing KindBuild (go group contribution after skipping empty entries)")
				}
			},
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			tc.fn(t)
		})
	}
}

// TestBakeProjectKindCatalog_NoTemplateEmptyCatalog covers the
// post-REFINEMENTS-2026-05-14 contract at the bake-helper boundary:
// a project with empty repo paths and no HOME-tier template receives
// empty KindCatalogJSON (ok=false from loadProjectTemplate → bake skipped).
// Templates are project-tier opt-in only.
func TestBakeProjectKindCatalog_NoTemplateEmptyCatalog(t *testing.T) {
	project := domain.Project{}
	if err := bakeProjectKindCatalogWithHome(&project, ""); err != nil {
		t.Fatalf("bakeProjectKindCatalogWithHome(): unexpected error = %v", err)
	}
	if len(project.KindCatalogJSON) != 0 {
		t.Fatalf("project.KindCatalogJSON is non-empty (%d bytes); want empty (no on-disk template → catalog not baked)", len(project.KindCatalogJSON))
	}
}

// TestBakeProjectKindCatalog_NonEmptyPathNoTemplateEmptyCatalog covers the
// post-REFINEMENTS-2026-05-14 contract: when repo paths are non-empty but
// neither carries an on-disk .tillsyn/template.toml, bakeProjectKindCatalog
// leaves KindCatalogJSON empty — no embedded fallback. Templates are
// project-tier opt-in only.
func TestBakeProjectKindCatalog_NonEmptyPathNoTemplateEmptyCatalog(t *testing.T) {
	project := domain.Project{
		RepoBareRoot:        t.TempDir(),
		RepoPrimaryWorktree: t.TempDir(),
	}
	if err := bakeProjectKindCatalogWithHome(&project, ""); err != nil {
		t.Fatalf("bakeProjectKindCatalogWithHome(): unexpected error = %v", err)
	}
	if len(project.KindCatalogJSON) != 0 {
		t.Fatalf("project.KindCatalogJSON is non-empty (%d bytes); want empty (no on-disk template → catalog not baked)", len(project.KindCatalogJSON))
	}
}

// TestSeedStewardAnchors_FallsBackToGenericWhenNoProjectTemplate verifies
// that seedStewardAnchors invokes the loadStewardSeedTemplate seam exactly
// once per project create and that the seam falls back to the generic
// embedded template when the project has no project-tier template (empty
// RepoBareRoot + empty RepoPrimaryWorktree).
//
// Phase 4.4 D2 migrated the seam from func(lang string) to
// func(project domain.Project). The production impl tries
// loadProjectTierTemplateOnly first; with no project-tier paths configured
// (ok=false), it falls through to templates.LoadBuiltinTemplate("till-gen").
// This test installs a fixture that counts invocations and returns a
// single-seed template, then asserts:
//  1. The seam is invoked exactly once per project create.
//  2. The seam receives the project value (ID is non-empty).
//  3. The materialized STEWARD anchors come from the fixture the seam
//     returned (proves the fixture was actually used to drive seed
//     materialization).
func TestSeedStewardAnchors_FallsBackToGenericWhenNoProjectTemplate(t *testing.T) {
	const anchorTitle = "GENERIC_FALLBACK_ANCHOR"
	var seamProjectsObserved []domain.Project
	withSeedTemplateFixture(t, func(proj domain.Project) (templates.Template, error) {
		seamProjectsObserved = append(seamProjectsObserved, proj)
		return templates.Template{
			SchemaVersion: templates.SchemaVersionV1,
			StewardSeeds: []templates.StewardSeed{
				{Title: anchorTitle, Description: "generic-fallback seeded anchor"},
			},
		}, nil
	})

	svc, repo := newSeederService(t)
	project, err := svc.CreateProjectWithMetadata(context.Background(), CreateProjectInput{
		Name: "Generic Fallback Demo",
		// RepoBareRoot and RepoPrimaryWorktree intentionally empty —
		// no project-tier template; production impl falls back to till-gen.
	})
	if err != nil {
		t.Fatalf("CreateProjectWithMetadata() error = %v", err)
	}

	// Acceptance #1: exactly one invocation per project create.
	if got := len(seamProjectsObserved); got != 1 {
		t.Fatalf("seam invocations = %d; want 1 (seedStewardAnchors fires once at project create)", got)
	}
	// Acceptance #2: seam receives the project (non-empty ID).
	if seamProjectsObserved[0].ID == "" {
		t.Fatalf("seam received project with empty ID; want the created project's ID")
	}

	// Acceptance #3: materialized anchor title matches fixture.
	var stewardTitles []string
	for _, item := range repo.tasks {
		if item.ProjectID != project.ID {
			continue
		}
		if item.Owner != stewardOwner {
			continue
		}
		stewardTitles = append(stewardTitles, item.Title)
	}
	sort.Strings(stewardTitles)
	wantTitles := []string{anchorTitle}
	if !reflect.DeepEqual(stewardTitles, wantTitles) {
		t.Fatalf("STEWARD anchor titles = %v; want %v (generic fixture must drive materialization)",
			stewardTitles, wantTitles)
	}
}

// TestSeedStewardAnchors_UsesProjectTierTemplateWhenPresent verifies that
// the production loadStewardSeedTemplate seam uses a project-tier template
// (from <RepoPrimaryWorktree>/.tillsyn/template.toml) when one is present
// and its StewardSeeds slice is non-empty, instead of falling back to the
// embedded generic (till-gen.toml).
//
// The test writes a minimal template.toml with a project-tier-only seed
// title ("PROJECT_TIER_ANCHOR") into a t.TempDir() directory and sets the
// project's RepoPrimaryWorktree to that dir. The production seam impl
// (loadProjectTierTemplateOnly) will find and parse the file; the test then
// asserts that the materialized STEWARD anchor carries the project-tier
// seed title, NOT the generic fallback seeds.
func TestSeedStewardAnchors_UsesProjectTierTemplateWhenPresent(t *testing.T) {
	// Write a minimal valid template.toml into a temp dir.
	tmpDir := t.TempDir()
	tillsynDir := filepath.Join(tmpDir, ".tillsyn")
	if err := os.MkdirAll(tillsynDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(.tillsyn) error = %v", err)
	}
	templatePath := filepath.Join(tillsynDir, "template.toml")
	// Minimal template: schema_version + one [[steward_seeds]] entry.
	// The kind catalog and child_rules are omitted — till-gen.toml defaults
	// fill them during Load; seedStewardAnchors only consumes StewardSeeds.
	templateContent := `schema_version = "v1"

[[steward_seeds]]
title = "PROJECT_TIER_ANCHOR"
description = "Seeded from project-tier template."
`
	if err := os.WriteFile(templatePath, []byte(templateContent), 0o644); err != nil {
		t.Fatalf("WriteFile(template.toml) error = %v", err)
	}

	// Do NOT install a fixture: use the REAL production loadStewardSeedTemplate
	// so this test exercises the actual project-tier-first logic.

	svc, repo := newSeederService(t)
	project, err := svc.CreateProjectWithMetadata(context.Background(), CreateProjectInput{
		Name:                "Project Tier Demo",
		RepoPrimaryWorktree: tmpDir,
		// RepoBareRoot intentionally empty — production prefers bare root
		// when present; primary worktree is the canonical non-bare layout.
	})
	if err != nil {
		t.Fatalf("CreateProjectWithMetadata() error = %v", err)
	}

	// The materialized STEWARD anchor must carry the project-tier seed title.
	var stewardTitles []string
	for _, item := range repo.tasks {
		if item.ProjectID != project.ID {
			continue
		}
		if item.Owner != stewardOwner {
			continue
		}
		stewardTitles = append(stewardTitles, item.Title)
	}
	sort.Strings(stewardTitles)
	wantTitles := []string{"PROJECT_TIER_ANCHOR"}
	if !reflect.DeepEqual(stewardTitles, wantTitles) {
		t.Fatalf("STEWARD anchor titles = %v; want %v (project-tier template must take priority over generic fallback)",
			stewardTitles, wantTitles)
	}
}

// TestCreateActionItem_AppliesTemplateChildRules is the canonical regression for
// the cascade template's auto-create contract: when CreateActionItem persists a
// parent whose (Kind, StructuralType) matches a [[child_rules]] entry in the
// project's template, the matching child action items are auto-created with
// blocked_by_parent edges wired. The canonical case under till-go.toml is:
//
//   - parent kind=build, structural_type=droplet -> build-qa-proof +
//     build-qa-falsification, both with metadata.BlockedBy = [parent.ID].
//
// Failure modes guarded against:
//   - the parent landing in the repo without spawning the cascade-mandated
//     QA twins (template-bound but rules silently ignored at create time);
//   - the auto-children missing the blocked_by edge (cascade dispatcher then
//     fires the QA agents before the build has actually completed);
//   - recursive cascade explosion (the auto-children themselves match no
//     [[child_rules]] in till-go.toml, so the recursion terminates after one
//     level — this test asserts exactly 2 auto-children, not 4 or more).
func TestCreateActionItem_AppliesTemplateChildRules(t *testing.T) {
	repo := newFakeRepo()
	now := time.Date(2026, 5, 14, 12, 0, 0, 0, time.UTC)
	idCounter := 0
	svc := NewService(repo, func() string {
		idCounter++
		return fmt.Sprintf("id-%04d", idCounter)
	}, func() time.Time {
		return now
	}, ServiceConfig{})

	// Project-tier template authoring is the contract: write the embedded
	// till-go.toml content to <RepoPrimaryWorktree>/.tillsyn/template.toml so
	// loadProjectTierTemplateOnly resolves a real on-disk file. Per REFINEMENTS
	// 2026-05-14 ("project-tier opt-in only"), the create-time auto-spawn
	// does NOT consult embedded fallbacks.
	worktree := t.TempDir()
	tplDir := filepath.Join(worktree, ".tillsyn")
	if err := os.MkdirAll(tplDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(%q): %v", tplDir, err)
	}
	tplBytes, err := templates.DefaultTemplateFS.ReadFile("builtin/till-go.toml")
	if err != nil {
		t.Fatalf("ReadFile embedded till-go.toml: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tplDir, "template.toml"), tplBytes, 0o644); err != nil {
		t.Fatalf("WriteFile template.toml: %v", err)
	}

	project, err := svc.CreateProjectWithMetadata(context.Background(), CreateProjectInput{
		Name:                "Cascade Template Test",
		RepoPrimaryWorktree: worktree,
		Metadata:            domain.ProjectMetadata{Groups: []string{"go"}},
		UpdatedBy:           "user-1",
		UpdatedType:         domain.ActorTypeUser,
	})
	if err != nil {
		t.Fatalf("CreateProjectWithMetadata() error = %v", err)
	}
	column, err := svc.CreateColumn(context.Background(), project.ID, "To Do", 0, 0)
	if err != nil {
		t.Fatalf("CreateColumn() error = %v", err)
	}

	parent, err := svc.CreateActionItem(context.Background(), CreateActionItemInput{
		ProjectID:      project.ID,
		ColumnID:       column.ID,
		Kind:           domain.KindBuild,
		StructuralType: domain.StructuralTypeDroplet,
		Title:          "PARENT BUILD",
		UpdatedByType:  domain.ActorTypeUser,
		UpdatedByActor: "user-1",
		CreatedByActor: "user-1",
	})
	if err != nil {
		t.Fatalf("CreateActionItem(parent build) error = %v", err)
	}

	items, err := svc.ListActionItems(context.Background(), project.ID, false)
	if err != nil {
		t.Fatalf("ListActionItems() error = %v", err)
	}
	var children []domain.ActionItem
	for _, item := range items {
		if item.ParentID == parent.ID {
			children = append(children, item)
		}
	}
	if len(children) != 2 {
		t.Fatalf("expected 2 auto-created children of parent build; got %d (items: %d)", len(children), len(items))
	}
	kinds := map[domain.Kind]int{}
	for _, c := range children {
		kinds[c.Kind]++
		if len(c.Metadata.BlockedBy) != 1 || c.Metadata.BlockedBy[0] != parent.ID {
			t.Errorf("child kind=%s BlockedBy = %v; want [%s] (cascade gate requires blocked_by parent)",
				c.Kind, c.Metadata.BlockedBy, parent.ID)
		}
		if c.ProjectID != project.ID {
			t.Errorf("child ProjectID = %q; want %q", c.ProjectID, project.ID)
		}
	}
	if kinds[domain.KindBuildQAProof] != 1 {
		t.Errorf("expected exactly 1 build-qa-proof child; got %d", kinds[domain.KindBuildQAProof])
	}
	if kinds[domain.KindBuildQAFalsification] != 1 {
		t.Errorf("expected exactly 1 build-qa-falsification child; got %d", kinds[domain.KindBuildQAFalsification])
	}
}

// TestCreateActionItemParentIDEqualsProjectID covers three parent-ID paths:
// Case 1 — ParentID == ProjectID is auto-cleared, producing a top-level item.
// Case 2 — a legitimate action-item ParentID succeeds and wires the child correctly.
// Case 3 — a non-existent ParentID (neither project nor any action item) returns ErrNotFound.
func TestCreateActionItemParentIDEqualsProjectID(t *testing.T) {
	now := time.Date(2026, 5, 15, 10, 0, 0, 0, time.UTC)

	// seedFixture seeds one project + column directly into the fakeRepo and
	// returns them. Bypasses svc.CreateProject to avoid template/file I/O
	// and child_rules side-effects, matching the pattern used by
	// TestCreateActionItemCarriesHumanActorName.
	seedFixture := func(t *testing.T, repo *fakeRepo, projectID, colID string) (domain.Project, domain.Column) {
		t.Helper()
		project, err := domain.NewProjectFromInput(domain.ProjectInput{
			ID:   projectID,
			Name: "Test Project",
		}, now)
		if err != nil {
			t.Fatalf("NewProjectFromInput() error = %v", err)
		}
		if repoErr := repo.CreateProject(context.Background(), project); repoErr != nil {
			t.Fatalf("repo.CreateProject() error = %v", repoErr)
		}
		col, colErr := domain.NewColumn(colID, project.ID, "To Do", 0, 0, now)
		if colErr != nil {
			t.Fatalf("NewColumn() error = %v", colErr)
		}
		if repoErr := repo.CreateColumn(context.Background(), col); repoErr != nil {
			t.Fatalf("repo.CreateColumn() error = %v", repoErr)
		}
		return project, col
	}

	// ---- Case 1: ParentID == ProjectID → auto-cleared → top-level item. ----
	t.Run("parent_id_equals_project_id_produces_top_level", func(t *testing.T) {
		repo := newFakeRepo()
		svc := NewService(repo, func() string { return "item-1a" }, func() time.Time { return now }, ServiceConfig{})
		project, col := seedFixture(t, repo, "proj-1", "col-1")

		item, err := svc.CreateActionItem(context.Background(), CreateActionItemInput{
			Kind:           domain.KindPlan,
			ProjectID:      project.ID,
			ParentID:       project.ID, // caller passes project UUID as parent
			ColumnID:       col.ID,
			Title:          "Top-level item",
			Priority:       domain.PriorityMedium,
			StructuralType: domain.StructuralTypeDroplet,
		})
		if err != nil {
			t.Fatalf("CreateActionItem(parent==project) error = %v; want nil", err)
		}
		if item.ParentID != "" {
			t.Errorf("item.ParentID = %q; want empty (top-level)", item.ParentID)
		}
		// Verify the stored row also has an empty ParentID.
		stored, getErr := repo.GetActionItem(context.Background(), item.ID)
		if getErr != nil {
			t.Fatalf("GetActionItem() error = %v", getErr)
		}
		if stored.ParentID != "" {
			t.Errorf("stored.ParentID = %q; want empty (top-level)", stored.ParentID)
		}
	})

	// ---- Case 2: legitimate action-item ParentID succeeds. ----
	t.Run("legitimate_parent_id_succeeds", func(t *testing.T) {
		repo := newFakeRepo()
		ids := []string{"child-2a"}
		idIdx := 0
		svc := NewService(repo, func() string {
			id := ids[idIdx]
			idIdx++
			return id
		}, func() time.Time { return now }, ServiceConfig{})
		project, col := seedFixture(t, repo, "proj-2", "col-2")

		// Seed a parent action item directly to avoid ID-generator side-effects.
		parentItem, newErr := domain.NewActionItemForTest(domain.ActionItemInput{
			Kind:      domain.KindPlan,
			ID:        "parent-2",
			ProjectID: project.ID,
			ColumnID:  col.ID,
			Position:  0,
			Title:     "Parent item",
			Priority:  domain.PriorityMedium,
		}, now)
		if newErr != nil {
			t.Fatalf("NewActionItemForTest() error = %v", newErr)
		}
		if repoErr := repo.CreateActionItem(context.Background(), parentItem); repoErr != nil {
			t.Fatalf("repo.CreateActionItem(parent) error = %v", repoErr)
		}

		child, err := svc.CreateActionItem(context.Background(), CreateActionItemInput{
			Kind:           domain.KindBuild,
			ProjectID:      project.ID,
			ParentID:       parentItem.ID, // a real action-item UUID
			ColumnID:       col.ID,
			Title:          "Child item",
			Priority:       domain.PriorityLow,
			StructuralType: domain.StructuralTypeDroplet,
		})
		if err != nil {
			t.Fatalf("CreateActionItem(legitimate parent) error = %v; want nil", err)
		}
		if child.ParentID != parentItem.ID {
			t.Errorf("child.ParentID = %q; want %q", child.ParentID, parentItem.ID)
		}
	})

	// ---- Case 3: non-existent ParentID returns ErrNotFound. ----
	t.Run("nonexistent_parent_id_returns_not_found", func(t *testing.T) {
		repo := newFakeRepo()
		svc := NewService(repo, func() string { return "item-3a" }, func() time.Time { return now }, ServiceConfig{})
		project, col := seedFixture(t, repo, "proj-3", "col-3")

		_, err := svc.CreateActionItem(context.Background(), CreateActionItemInput{
			Kind:           domain.KindPlan,
			ProjectID:      project.ID,
			ParentID:       "nonexistent-action-item-id", // in neither table
			ColumnID:       col.ID,
			Title:          "Orphan item",
			Priority:       domain.PriorityLow,
			StructuralType: domain.StructuralTypeDroplet,
		})
		if !errors.Is(err, ErrNotFound) {
			t.Errorf("CreateActionItem(bad parent) error = %v; want ErrNotFound", err)
		}
	})
}
