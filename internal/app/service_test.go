package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"testing"
	"time"

	fantasyembed "github.com/evanmschultz/tillsyn/internal/adapters/embeddings/fantasy"
	"github.com/evanmschultz/tillsyn/internal/domain"
)

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
	project, err := domain.NewProject("p-second-wave", "Second Wave", "Project description", now)
	if err != nil {
		t.Fatalf("NewProject() error = %v", err)
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
		Title:        "new title",
		Description:  "details",
		Priority:     domain.PriorityHigh,
		DueAt:        &due,
		Labels:       []string{"frontend", "backend"},
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
		Title:        "new title",
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
	project, _ := domain.NewProject("p1", "Inbox", "", now)
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
		Title:         "new title",
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
		Title:        "new title",
	})
	if err != nil {
		t.Fatalf("UpdateActionItem(title-only) error = %v", err)
	}
	if updated.Priority != domain.PriorityMedium {
		t.Fatalf("priority = %q, want %q", updated.Priority, domain.PriorityMedium)
	}
}

// TestListAndSortHelpers verifies behavior for the covered scenario.
func TestListAndSortHelpers(t *testing.T) {
	repo := newFakeRepo()
	now := time.Date(2026, 2, 21, 12, 0, 0, 0, time.UTC)
	p, _ := domain.NewProject("p1", "Project", "", now)
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
	p1, _ := domain.NewProject("p1", "Inbox", "", now)
	p2, _ := domain.NewProject("p2", "Client", "", now)
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
	p1, _ := domain.NewProject("p1", "Inbox", "", now)
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
	project, _ := domain.NewProject("p1", "Inbox", "", now)
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
	project, _ := domain.NewProject("p1", "Inbox", "", now)
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
	project, _ := domain.NewProject("p1", "Inbox", "", now)
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
	project, _ := domain.NewProject("p1", "Inbox", "", now)
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
	project, _ := domain.NewProject("p1", "Inbox", "", now)
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
		Title:        "Ship hybrid search",
		Description:  created.Description,
		Priority:     created.Priority,
		Labels:       created.Labels,
		DueAt:        created.DueAt,
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
	project, _ := domain.NewProject("p1", "Inbox", "", now)
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
	project, _ := domain.NewProject("p1", "Inbox", "", now)
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
	project, _ := domain.NewProject("p1", "Inbox", "", now)
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
	project, _ := domain.NewProject("p1", "Inbox", "", now)
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
	p, _ := domain.NewProject("p1", "Existing", "", now)
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
	project, _ := domain.NewProject("p1", "Inbox", "old desc", now)
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
	project, _ := domain.NewProject("p1", "Inbox", "desc", now)
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
	project, _ := domain.NewProject("p1", "Inbox", "", now)
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
	project, _ := domain.NewProject("p1", "Inbox", "", now)
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
				Policy: domain.CompletionPolicy{RequireChildrenComplete: true},
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

// TestMoveActionItemBlocksDoneWhenCompletionContractRequiresChildren verifies legacy require-children behavior remains intact.
func TestMoveActionItemBlocksDoneWhenCompletionContractRequiresChildren(t *testing.T) {
	repo := newFakeRepo()
	now := time.Date(2026, 2, 21, 12, 0, 0, 0, time.UTC)
	project, _ := domain.NewProject("p1", "Inbox", "", now)
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
				Policy: domain.CompletionPolicy{RequireChildrenComplete: true},
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
	project, _ := domain.NewProject("p1", "Inbox", "", now)
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
	project, _ := domain.NewProject("p1", "Inbox", "", now)
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
	project, _ := domain.NewProject("p1", "Inbox", "", now)
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
	project, _ := domain.NewProject("p1", "Inbox", "", now)
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
	project, _ := domain.NewProject("p1", "Inbox", "", now)
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
	project, _ := domain.NewProject("p1", "Inbox", "", now)
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
	project, _ := domain.NewProject("p1", "Inbox", "", now)
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
	project, _ := domain.NewProject("p1", "Inbox", "", now)
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
		Title:        "Branch A",
		Description:  "branch-updated",
		Priority:     domain.PriorityMedium,
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
		Title:        "ActionItem B1",
		Description:  "out of scope",
		Priority:     domain.PriorityMedium,
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

	project, err := svc.CreateProject(context.Background(), "Kinds", "")
	if err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}
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
	project, _ := domain.NewProject("p1", "Inbox", "", now)
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
	project, _ := domain.NewProject("p1", "Inbox", "", now)
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
			CompletionContract: domain.CompletionContract{
				CompletionCriteria: []domain.ChecklistItem{{ID: "c1", Text: "tests green", Complete: false}},
				Policy:             domain.CompletionPolicy{RequireChildrenComplete: true},
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
	project, _ := domain.NewProject("p1", "Inbox", "", now)
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
	project, _ := domain.NewProject("p1", "Inbox", "", now)
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
	project, _ := domain.NewProject("p1", "Inbox", "", now)
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
