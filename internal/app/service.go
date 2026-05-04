package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/log"
	"github.com/evanmschultz/tillsyn/internal/domain"
	"github.com/evanmschultz/tillsyn/internal/templates"
)

// DeleteMode represents a selectable mode.
type DeleteMode string

// DeleteModeArchive and related constants define package defaults.
const (
	DeleteModeArchive DeleteMode = "archive"
	DeleteModeHard    DeleteMode = "hard"
)

// SearchMode represents a selectable search strategy.
type SearchMode string

// Search mode constants define supported search behavior contracts.
const (
	SearchModeKeyword  SearchMode = "keyword"
	SearchModeSemantic SearchMode = "semantic"
	SearchModeHybrid   SearchMode = "hybrid"
)

// SearchSort defines supported result ordering options.
type SearchSort string

// Search sort and pagination constants define supported contracts and defaults.
const (
	SearchSortRankDesc      SearchSort = "rank_desc"
	SearchSortTitleAsc      SearchSort = "title_asc"
	SearchSortCreatedAtDesc SearchSort = "created_at_desc"
	SearchSortUpdatedAtDesc SearchSort = "updated_at_desc"

	defaultSearchLimit              = 50
	maxSearchLimit                  = 200
	defaultSearchLexicalWeight      = 0.55
	defaultSearchSemanticWeight     = 0.45
	defaultSearchSemanticCandidates = 200
)

// supportedSearchLevelFilters lists accepted level values for search filters.
// Scope mirrors kind per the 12-value Kind enum, so the accepted level set is
// the lowercase form of every KindAppliesTo value.
var supportedSearchLevelFilters = map[string]struct{}{
	string(domain.KindAppliesToPlan):                 {},
	string(domain.KindAppliesToResearch):             {},
	string(domain.KindAppliesToBuild):                {},
	string(domain.KindAppliesToPlanQAProof):          {},
	string(domain.KindAppliesToPlanQAFalsification):  {},
	string(domain.KindAppliesToBuildQAProof):         {},
	string(domain.KindAppliesToBuildQAFalsification): {},
	string(domain.KindAppliesToCloseout):             {},
	string(domain.KindAppliesToCommit):               {},
	string(domain.KindAppliesToRefinement):           {},
	string(domain.KindAppliesToDiscussion):           {},
	string(domain.KindAppliesToHumanVerify):          {},
}

// ServiceConfig holds configuration for service.
type ServiceConfig struct {
	DefaultDeleteMode        DeleteMode
	StateTemplates           []StateTemplate
	AutoCreateProjectColumns bool
	// AutoSeedStewardAnchors controls whether project + numbered-drop creation
	// auto-materialize the cascade template's STEWARD persistent anchors and
	// per-drop level_2 findings + refinements-gate (droplet 3.20). Defaults
	// false so existing tests that pre-allocate a fixed ID slice are not
	// retro-broken; production callers (cmd/till/main.go) explicitly opt in.
	AutoSeedStewardAnchors   bool
	CapabilityLeaseTTL       time.Duration
	RequireAgentLease        *bool
	AuthRequests             AuthRequestGateway
	EmbeddingGenerator       EmbeddingGenerator
	SearchIndex              EmbeddingSearchIndex
	EmbeddingLifecycle       EmbeddingLifecycleStore
	EmbeddingRuntime         EmbeddingRuntimeConfig
	SearchLexicalWeight      float64
	SearchSemanticWeight     float64
	SearchSemanticCandidates int
	AuthBackend              AuthBackend
	LiveWaitBroker           LiveWaitBroker
}

// StateTemplate represents state template data used by this package.
type StateTemplate struct {
	ID       string
	Name     string
	WIPLimit int
	Position int
	Hidden   bool
}

// IDGenerator returns unique identifiers for new entities.
type IDGenerator func() string

// Clock returns the current time.
type Clock func() time.Time

// Service represents service data used by this package.
type Service struct {
	repo               Repository
	idGen              IDGenerator
	clock              Clock
	defaultDeleteMode  DeleteMode
	stateTemplates     []StateTemplate
	autoProjectCols    bool
	autoSeedSteward    bool
	defaultLeaseTTL    time.Duration
	requireAgentLease  bool
	authRequests       AuthRequestGateway
	handoffRepo        HandoffRepository
	schemaCache        map[string]schemaCacheEntry
	schemaCacheMu      sync.RWMutex
	embeddingGenerator EmbeddingGenerator
	searchIndex        EmbeddingSearchIndex
	embeddingLifecycle EmbeddingLifecycleStore
	embeddingRuntime   EmbeddingRuntimeConfig
	searchLexicalW     float64
	searchSemanticW    float64
	searchSemanticK    int
	authBackend        AuthBackend
	liveWait           LiveWaitBroker
}

// NewService constructs a new value for this package.
func NewService(repo Repository, idGen IDGenerator, clock Clock, cfg ServiceConfig) *Service {
	if idGen == nil {
		idGen = func() string { return "" }
	}
	if clock == nil {
		clock = time.Now
	}
	if cfg.DefaultDeleteMode == "" {
		cfg.DefaultDeleteMode = DeleteModeArchive
	}
	if cfg.CapabilityLeaseTTL <= 0 {
		cfg.CapabilityLeaseTTL = defaultCapabilityLeaseTTL
	}
	requireAgentLease := true
	if cfg.RequireAgentLease != nil {
		requireAgentLease = *cfg.RequireAgentLease
	}
	templates := sanitizeStateTemplates(cfg.StateTemplates)
	if len(templates) == 0 {
		templates = defaultStateTemplates()
	}
	searchIndex := cfg.SearchIndex
	if searchIndex == nil {
		if idx, ok := repo.(EmbeddingSearchIndex); ok {
			searchIndex = idx
		}
	}
	embeddingLifecycle := cfg.EmbeddingLifecycle
	if embeddingLifecycle == nil {
		if lifecycle, ok := repo.(EmbeddingLifecycleStore); ok {
			embeddingLifecycle = lifecycle
		}
	}
	handoffRepo, _ := repo.(HandoffRepository)
	lexicalWeight, semanticWeight := normalizeSearchWeights(cfg.SearchLexicalWeight, cfg.SearchSemanticWeight)
	semanticCandidates := cfg.SearchSemanticCandidates
	if semanticCandidates <= 0 {
		semanticCandidates = defaultSearchSemanticCandidates
	}
	liveWait := cfg.LiveWaitBroker
	if liveWait == nil {
		liveWait = NewInProcessLiveWaitBroker()
	}

	return &Service{
		repo:               repo,
		idGen:              idGen,
		clock:              clock,
		defaultDeleteMode:  cfg.DefaultDeleteMode,
		stateTemplates:     templates,
		autoProjectCols:    cfg.AutoCreateProjectColumns,
		autoSeedSteward:    cfg.AutoSeedStewardAnchors,
		defaultLeaseTTL:    cfg.CapabilityLeaseTTL,
		requireAgentLease:  requireAgentLease,
		authRequests:       cfg.AuthRequests,
		handoffRepo:        handoffRepo,
		schemaCache:        map[string]schemaCacheEntry{},
		embeddingGenerator: cfg.EmbeddingGenerator,
		searchIndex:        searchIndex,
		embeddingLifecycle: embeddingLifecycle,
		embeddingRuntime:   cfg.EmbeddingRuntime.Normalize(),
		searchLexicalW:     lexicalWeight,
		searchSemanticW:    semanticWeight,
		searchSemanticK:    semanticCandidates,
		authBackend:        cfg.AuthBackend,
		liveWait:           liveWait,
	}
}

// Clock returns the service's current-time source so callers that share this
// service can stay aligned with its time perspective (including test clocks).
func (s *Service) Clock() Clock {
	if s == nil || s.clock == nil {
		return time.Now
	}
	return s.clock
}

// EnsureDefaultProject ensures default project.
func (s *Service) EnsureDefaultProject(ctx context.Context) (domain.Project, error) {
	projects, err := s.repo.ListProjects(ctx, false)
	if err != nil {
		return domain.Project{}, err
	}
	if len(projects) > 0 {
		return projects[0], nil
	}

	now := s.clock()
	project, err := domain.NewProject(s.idGen(), "Inbox", "Default project", now)
	if err != nil {
		return domain.Project{}, err
	}
	if err := bakeProjectKindCatalog(&project); err != nil {
		return domain.Project{}, err
	}
	if err := s.repo.CreateProject(ctx, project); err != nil {
		return domain.Project{}, err
	}
	if err := s.initializeProjectAllowedKinds(ctx, project); err != nil {
		return domain.Project{}, err
	}

	if err := s.createDefaultColumns(ctx, project.ID, now); err != nil {
		return domain.Project{}, err
	}

	if s.autoSeedSteward {
		if err := s.seedStewardAnchors(ctx, project); err != nil {
			return domain.Project{}, err
		}
	}

	return project, nil
}

// CreateProjectInput holds input values for create project operations.
type CreateProjectInput struct {
	Name          string
	Description   string
	Kind          domain.KindID
	Metadata      domain.ProjectMetadata
	UpdatedBy     string
	UpdatedByName string
	UpdatedType   domain.ActorType
}

// CreateProject creates project.
func (s *Service) CreateProject(ctx context.Context, name, description string) (domain.Project, error) {
	return s.CreateProjectWithMetadata(ctx, CreateProjectInput{
		Name:        name,
		Description: description,
	})
}

// CreateProjectWithMetadata creates project with metadata.
func (s *Service) CreateProjectWithMetadata(ctx context.Context, in CreateProjectInput) (domain.Project, error) {
	ctx, resolvedActor, hasResolvedActor := withResolvedMutationActor(ctx, in.UpdatedBy, in.UpdatedByName, in.UpdatedType)
	now := s.clock()
	project, err := domain.NewProject(s.idGen(), in.Name, in.Description, now)
	if err != nil {
		return domain.Project{}, err
	}
	mergedMetadata, err := domain.MergeProjectMetadata(in.Metadata, nil)
	if err != nil {
		return domain.Project{}, err
	}
	if hasResolvedActor && resolvedActor.ActorType == domain.ActorTypeUser && strings.TrimSpace(mergedMetadata.Owner) == "" {
		mergedMetadata.Owner = strings.TrimSpace(resolvedActor.ActorName)
	}
	if err := project.UpdateDetails(project.Name, project.Description, mergedMetadata, now); err != nil {
		return domain.Project{}, err
	}
	if err := bakeProjectKindCatalog(&project); err != nil {
		return domain.Project{}, err
	}
	if err := s.repo.CreateProject(ctx, project); err != nil {
		return domain.Project{}, err
	}
	if err := s.initializeProjectAllowedKinds(ctx, project); err != nil {
		return domain.Project{}, err
	}
	if s.autoProjectCols {
		if err := s.createDefaultColumns(ctx, project.ID, now); err != nil {
			return domain.Project{}, err
		}
		if s.autoSeedSteward {
			if err := s.seedStewardAnchors(ctx, project); err != nil {
				return domain.Project{}, err
			}
		}
	}
	if _, err := s.enqueueProjectDocumentEmbedding(ctx, project, false, "project_created"); err != nil {
		return domain.Project{}, err
	}
	return project, nil
}

// bakeProjectKindCatalog populates project.KindCatalogJSON with the JSON
// envelope of a templates.KindCatalog baked from the project's bound
// Template at create time.
//
// Per Drop 3 droplet 3.12 + fix L5: the catalog is loaded from
//
//   - <project_root>/.tillsyn/template.toml when an explicit per-project
//     template exists (file-system source resolution lands in droplet 3.14
//     alongside the embedded default), OR
//   - the embedded internal/templates/builtin/default.toml fallback that
//     droplet 3.14 introduces.
//
// Until 3.14 lands, neither source is available from this droplet and the
// helper is a no-op: KindCatalogJSON stays empty (length 0). Per droplet
// 3.12 acceptance criterion an empty envelope routes through the legacy
// repo fallback in resolveActionItemKindDefinition, preserving Drop 2.8
// universal-nesting boot compatibility.
//
// The helper accepts a *domain.Project so 3.14 can substitute a real
// implementation without touching the call site in CreateProjectWithMetadata.
// It returns error so future template-load failures can propagate without
// another signature break.
//
// Per Drop 3 finding 5.B.14: edits to <project_root>/.tillsyn/template.toml
// AFTER project creation are ignored — the catalog is the create-time
// snapshot. Re-baking on every project lookup is explicitly out of scope.
func bakeProjectKindCatalog(project *domain.Project) error {
	if project == nil {
		return nil
	}
	tpl, ok, err := loadProjectTemplate()
	if err != nil {
		return err
	}
	if !ok {
		return nil
	}
	catalog := templates.Bake(tpl)
	encoded, err := json.Marshal(catalog)
	if err != nil {
		return fmt.Errorf("encode kind catalog: %w", err)
	}
	project.KindCatalogJSON = encoded
	return nil
}

// loadProjectTemplate is the future hook for resolving the Template that
// CreateProjectWithMetadata bakes into a project's KindCatalog. Droplet
// 3.14 fills this in with file-system + embedded TOML resolution. Until
// then it returns (zero, false, nil), which routes the create path through
// the empty-catalog branch.
//
// Note: droplet 3.20's STEWARD-seed auto-generator does NOT depend on this
// helper — it loads the embedded default template independently via
// templates.LoadDefaultTemplate so seed materialization is decoupled from
// the KindCatalog-bake fallback semantics. See seedStewardAnchors below.
func loadProjectTemplate() (templates.Template, bool, error) {
	return templates.Template{}, false, nil
}

// UpdateProjectInput holds input values for update project operations.
type UpdateProjectInput struct {
	ProjectID     string
	Name          string
	Description   string
	Kind          domain.KindID
	Metadata      domain.ProjectMetadata
	UpdatedBy     string
	UpdatedByName string
	UpdatedType   domain.ActorType
}

// UpdateProject updates state for the requested operation.
func (s *Service) UpdateProject(ctx context.Context, in UpdateProjectInput) (domain.Project, error) {
	project, err := s.repo.GetProject(ctx, in.ProjectID)
	if err != nil {
		return domain.Project{}, err
	}
	ctx, _, _ = withResolvedMutationActor(ctx, in.UpdatedBy, in.UpdatedByName, in.UpdatedType)
	actorType := in.UpdatedType
	if actorType == "" {
		actorType = domain.ActorTypeUser
	}
	if err := s.enforceMutationGuard(ctx, project.ID, actorType, domain.CapabilityScopeProject, project.ID, domain.CapabilityActionEditNode); err != nil {
		return domain.Project{}, err
	}
	if err := project.UpdateDetails(in.Name, in.Description, in.Metadata, s.clock()); err != nil {
		return domain.Project{}, err
	}
	if err := s.repo.UpdateProject(ctx, project); err != nil {
		return domain.Project{}, err
	}
	if _, err := s.enqueueProjectDocumentEmbedding(ctx, project, false, "project_updated"); err != nil {
		return domain.Project{}, err
	}
	if _, err := s.enqueueThreadContextEmbedding(ctx, domain.CommentTarget{
		ProjectID:  project.ID,
		TargetType: domain.CommentTargetTypeProject,
		TargetID:   project.ID,
	}, false, "project_updated"); err != nil && !errors.Is(err, ErrNotFound) {
		return domain.Project{}, err
	}
	return project, nil
}

// ArchiveProject archives one project.
func (s *Service) ArchiveProject(ctx context.Context, projectID string) (domain.Project, error) {
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		return domain.Project{}, domain.ErrInvalidID
	}
	project, err := s.repo.GetProject(ctx, projectID)
	if err != nil {
		return domain.Project{}, err
	}
	if err := s.enforceMutationGuard(ctx, project.ID, domain.ActorTypeUser, domain.CapabilityScopeProject, project.ID, domain.CapabilityActionArchiveOrCleanup); err != nil {
		return domain.Project{}, err
	}
	project.Archive(s.clock())
	if err := s.repo.UpdateProject(ctx, project); err != nil {
		return domain.Project{}, err
	}
	return project, nil
}

// RestoreProject restores one archived project.
func (s *Service) RestoreProject(ctx context.Context, projectID string) (domain.Project, error) {
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		return domain.Project{}, domain.ErrInvalidID
	}
	project, err := s.repo.GetProject(ctx, projectID)
	if err != nil {
		return domain.Project{}, err
	}
	if err := s.enforceMutationGuard(ctx, project.ID, domain.ActorTypeUser, domain.CapabilityScopeProject, project.ID, domain.CapabilityActionArchiveOrCleanup); err != nil {
		return domain.Project{}, err
	}
	project.Restore(s.clock())
	if err := s.repo.UpdateProject(ctx, project); err != nil {
		return domain.Project{}, err
	}
	return project, nil
}

// DeleteProject deletes one project and all associated rows.
func (s *Service) DeleteProject(ctx context.Context, projectID string) error {
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		return domain.ErrInvalidID
	}
	project, err := s.repo.GetProject(ctx, projectID)
	if err != nil {
		return err
	}
	if err := s.enforceMutationGuard(ctx, project.ID, domain.ActorTypeUser, domain.CapabilityScopeProject, project.ID, domain.CapabilityActionArchiveOrCleanup); err != nil {
		return err
	}
	return s.repo.DeleteProject(ctx, project.ID)
}

// CreateColumn creates column.
func (s *Service) CreateColumn(ctx context.Context, projectID, name string, position, wipLimit int) (domain.Column, error) {
	column, err := domain.NewColumn(s.idGen(), projectID, name, position, wipLimit, s.clock())
	if err != nil {
		return domain.Column{}, err
	}
	if err := s.repo.CreateColumn(ctx, column); err != nil {
		return domain.Column{}, err
	}
	return column, nil
}

// CreateActionItemInput holds input values for create actionItem operations.
type CreateActionItemInput struct {
	ProjectID string
	ParentID  string
	Kind      domain.Kind
	Scope     domain.KindAppliesTo
	// Role optionally tags the action item with a closed-enum role (e.g.
	// builder, qa-proof, planner). Empty string is permitted; non-empty
	// values must match the closed Role enum or domain.NewActionItem returns
	// ErrInvalidRole.
	Role domain.Role
	// StructuralType places the action item on the cascade tree's shape
	// axis (drop|segment|confluence|droplet). Empty is REJECTED on create —
	// domain.NewActionItem returns ErrInvalidStructuralType. This diverges
	// from Role's permissive empty: the cascade-methodology shape axis is
	// mandatory at creation time.
	StructuralType domain.StructuralType
	// Owner optionally tags the new action item with a principal-name
	// string (e.g. "STEWARD"). Empty string is permitted; whitespace-only
	// collapses to empty. Domain primitive (per L13) — not STEWARD-specific.
	// Threaded into domain.NewActionItem; semantics per
	// `ta-docs/cascade-methodology.md` §11.2.
	Owner string
	// DropNumber stores the cascade drop index. Zero is permitted (treated
	// as "not a numbered drop"); positive values round-trip; negative values
	// reject with ErrInvalidDropNumber via domain.NewActionItem. Domain
	// primitive — not STEWARD-specific.
	DropNumber int
	// Persistent marks long-lived umbrella / anchor / perpetual-tracking
	// nodes. Default false. Domain primitive — not STEWARD-specific.
	Persistent bool
	// DevGated marks nodes whose terminal transition requires dev sign-off
	// (refinement rollups, human-verify hold points). Default false. Domain
	// primitive — not STEWARD-specific.
	DevGated bool
	// Paths optionally enumerates the action item's write-scope file paths
	// (forward-slash, repo-root-relative). Empty slice IS the meaningful
	// zero value (no path scope declared) — no pointer-sentinel needed.
	// domain.NewActionItem trims + dedupes; whitespace-only / backslash-
	// bearing entries reject with ErrInvalidPaths. Domain primitive per
	// Drop 4a L3.
	Paths []string
	// Packages optionally enumerates the Go-package import paths covering
	// Paths. Empty slice IS the meaningful zero value (no package scope) —
	// no pointer-sentinel needed at the create boundary. domain.NewActionItem
	// trims + dedupes; whitespace-only / empty entries reject with
	// ErrInvalidPackages. Domain coverage invariant: non-empty Paths
	// requires non-empty Packages (else ErrInvalidPackages). Domain
	// primitive per Drop 4a L3 / WAVE_1_PLAN.md §1.2.
	Packages       []string
	ColumnID       string
	Title          string
	Description    string
	Priority       domain.Priority
	DueAt          *time.Time
	Labels         []string
	Metadata       domain.ActionItemMetadata
	CreatedByActor string
	CreatedByName  string
	UpdatedByActor string
	UpdatedByName  string
	UpdatedByType  domain.ActorType
}

// UpdateActionItemInput holds input values for update actionItem operations.
type UpdateActionItemInput struct {
	ActionItemID string
	Title        string
	Description  string
	Priority     domain.Priority
	DueAt        *time.Time
	Labels       []string
	// Role optionally updates the action item's closed-enum role. Empty
	// string preserves the existing value (no-op). A non-empty value must
	// match the closed Role enum or the service returns ErrInvalidRole.
	Role domain.Role
	// StructuralType optionally updates the action item's closed-enum
	// structural type. Empty preserves the existing value (no-op) — mirrors
	// Role's required-on-create / optional-on-update split. A non-empty
	// value must match the closed StructuralType enum or the service
	// returns ErrInvalidStructuralType.
	StructuralType domain.StructuralType
	// Owner optionally updates the action-item Owner field. nil preserves
	// the existing value (no-op); non-nil sets to the dereferenced string
	// (whitespace-trimmed). Pointer-sentinel mirrors
	// UpdateActionItemRequest.Owner so the L1 STEWARD field-level guard at
	// the adapter boundary can distinguish "absent" from "explicit empty".
	// Domain primitive (per L13) — not STEWARD-specific.
	Owner *string
	// DropNumber optionally updates the action-item DropNumber. nil preserves
	// the existing value (no-op); non-nil sets the dereferenced int.
	// Negative values reject with ErrInvalidDropNumber. Pointer-sentinel
	// rationale matches Owner above.
	DropNumber *int
	// Persistent optionally updates the action-item Persistent flag. nil
	// preserves the existing value (no-op); non-nil sets the dereferenced
	// bool. Pointer-sentinel keeps "absent" distinguishable from
	// "explicit false" so a description-only update doesn't silently clobber
	// a STEWARD-seeded Persistent=true anchor node.
	Persistent *bool
	// DevGated optionally updates the action-item DevGated flag. Same
	// pointer-sentinel rationale as Persistent above.
	DevGated *bool
	// Paths optionally updates the action-item Paths slice. nil preserves
	// the existing value (no-op); non-nil replaces it. Pointer-sentinel
	// distinguishes "absent / preserve" from "explicit empty / clear all
	// declared paths" — a description-only update by an agent must NOT
	// silently clobber a planner-set Paths declaration. Domain primitive
	// per Drop 4a L3; service trims/dedupes via domain.NewActionItem-style
	// normalization at apply time.
	Paths *[]string
	// Packages optionally updates the action-item Packages slice. nil
	// preserves the existing value (no-op); non-nil replaces it. Same
	// pointer-sentinel rationale as Paths above — a description-only update
	// must NOT silently clobber a planner-set Packages declaration. Service
	// applies via domain.NormalizeActionItemPackages so the create-time
	// trim/dedupe rules apply equally on update; the coverage invariant
	// (non-empty Paths requires non-empty Packages) is re-checked against
	// the post-apply pair so paired Paths/Packages updates land atomically.
	// Domain primitive per Drop 4a L3 / WAVE_1_PLAN.md §1.2.
	Packages      *[]string
	Metadata      *domain.ActionItemMetadata
	UpdatedBy     string
	UpdatedByName string
	UpdatedType   domain.ActorType
}

// CreateCommentInput holds input values for create comment operations.
type CreateCommentInput struct {
	ProjectID    string
	TargetType   domain.CommentTargetType
	TargetID     string
	Summary      string
	BodyMarkdown string
	ActorID      string
	ActorName    string
	ActorType    domain.ActorType
}

// ListCommentsByTargetInput holds input values for list comment operations.
type ListCommentsByTargetInput struct {
	ProjectID   string
	TargetType  domain.CommentTargetType
	TargetID    string
	WaitTimeout time.Duration
}

// SearchActionItemsFilter defines filtering criteria for queries.
type SearchActionItemsFilter struct {
	ProjectID       string
	Query           string
	CrossProject    bool
	IncludeArchived bool
	States          []string
	Levels          []string
	Kinds           []string
	LabelsAny       []string
	LabelsAll       []string
	Mode            SearchMode
	Sort            SearchSort
	Limit           int
	Offset          int
}

// ActionItemMatch describes a matched result.
type ActionItemMatch struct {
	Project                   domain.Project
	ActionItem                domain.ActionItem
	StateID                   string
	EmbeddingSubjectType      EmbeddingSubjectType
	EmbeddingSubjectID        string
	EmbeddingStatus           EmbeddingLifecycleStatus
	EmbeddingUpdatedAt        *time.Time
	EmbeddingStaleReason      string
	EmbeddingLastErrorSummary string
	SemanticScore             float64
	UsedSemantic              bool
}

// SearchActionItemMatchesResult stores search rows plus execution metadata.
type SearchActionItemMatchesResult struct {
	Matches                []ActionItemMatch
	RequestedMode          SearchMode
	EffectiveMode          SearchMode
	FallbackReason         string
	SemanticAvailable      bool
	SemanticCandidateCount int
	EmbeddingSummary       EmbeddingSummary
}

// CreateActionItem creates actionItem.
func (s *Service) CreateActionItem(ctx context.Context, in CreateActionItemInput) (domain.ActionItem, error) {
	actorType := in.UpdatedByType
	if actorType == "" {
		actorType = domain.ActorTypeUser
	}
	ctx, resolvedActor, _ := withResolvedMutationActor(
		ctx,
		firstNonEmptyTrimmed(in.UpdatedByActor, in.CreatedByActor),
		firstNonEmptyTrimmed(in.UpdatedByName, in.CreatedByName),
		actorType,
	)
	var parent *domain.ActionItem
	guardScopes := []mutationScopeCandidate{
		newProjectMutationScopeCandidate(in.ProjectID),
	}
	if strings.TrimSpace(in.ParentID) != "" {
		parentActionItem, err := s.repo.GetActionItem(ctx, in.ParentID)
		if err != nil {
			return domain.ActionItem{}, err
		}
		if parentActionItem.ProjectID != in.ProjectID {
			return domain.ActionItem{}, domain.ErrInvalidParentID
		}
		parent = &parentActionItem
		guardScopes, err = s.capabilityScopesForActionItemLineage(ctx, parentActionItem)
		if err != nil {
			return domain.ActionItem{}, err
		}
	}
	if err := s.enforceMutationGuardAcrossScopes(ctx, in.ProjectID, actorType, guardScopes, domain.CapabilityActionCreateChild); err != nil {
		return domain.ActionItem{}, err
	}

	// Per Drop 3 droplet 3.16 + finding 5.B.15: every Template.AllowsNesting
	// rejection at the auth-gated CreateActionItem boundary writes a
	// till.comment on the parent + an attention_item with kind =
	// template_rejection. The check runs BEFORE
	// resolveActionItemKindDefinition so we can surface the catalog's
	// reason verbatim (the resolver wraps the same decision but flattens
	// the reason into a generic ErrKindNotAllowed). Update + Reparent
	// paths do NOT emit audit rows — finding 5.B.15 narrows the audit
	// trail to the create boundary; their resolver-level rejection still
	// fires with the same wrapped error.
	if parent != nil {
		parentKind := domain.Kind(parent.Kind)
		childKind := domain.Kind(domain.NormalizeKindID(domain.KindID(in.Kind)))
		allowed, reason, decisionErr := s.templateNestingDecision(ctx, in.ProjectID, parentKind, childKind)
		if decisionErr != nil {
			return domain.ActionItem{}, decisionErr
		}
		if !allowed {
			auditActor := MutationActor{
				ActorID:   firstNonEmptyTrimmed(resolvedActor.ActorID, in.UpdatedByActor, in.CreatedByActor),
				ActorName: firstNonEmptyTrimmed(resolvedActor.ActorName, in.UpdatedByName, in.CreatedByName),
				ActorType: actorType,
			}
			if auditErr := s.recordTemplateRejectionAudit(ctx, *parent, childKind, reason, auditActor); auditErr != nil {
				return domain.ActionItem{}, auditErr
			}
			return domain.ActionItem{}, fmt.Errorf("%w: %s", domain.ErrKindNotAllowed, reason)
		}
	}

	scope := normalizeActionItemScopeForKind(domain.KindID(in.Kind), in.Scope, parent)
	kindDef, err := s.resolveActionItemKindDefinition(ctx, in.ProjectID, domain.KindID(in.Kind), scope, parent)
	if err != nil {
		return domain.ActionItem{}, err
	}
	mergedMetadata, err := mergeActionItemMetadataWithKindTemplate(in.Metadata, kindDef)
	if err != nil {
		return domain.ActionItem{}, err
	}
	if err := s.validateKindPayload(kindDef, mergedMetadata.KindPayload); err != nil {
		return domain.ActionItem{}, err
	}
	tasks, err := s.repo.ListActionItems(ctx, in.ProjectID, false)
	if err != nil {
		return domain.ActionItem{}, err
	}
	columns, err := s.repo.ListColumns(ctx, in.ProjectID, true)
	if err != nil {
		return domain.ActionItem{}, err
	}
	lifecycleState := lifecycleStateForColumnID(columns, in.ColumnID)
	if lifecycleState == "" {
		lifecycleState = domain.StateTodo
	}
	position := 0
	for _, t := range tasks {
		if t.ColumnID == in.ColumnID && t.Position >= position {
			position = t.Position + 1
		}
	}

	actionItem, err := domain.NewActionItem(domain.ActionItemInput{
		ID:             s.idGen(),
		ProjectID:      in.ProjectID,
		ParentID:       in.ParentID,
		Kind:           domain.Kind(kindDef.ID),
		Scope:          scope,
		Role:           in.Role,
		StructuralType: in.StructuralType,
		Owner:          in.Owner,
		DropNumber:     in.DropNumber,
		Persistent:     in.Persistent,
		DevGated:       in.DevGated,
		Paths:          in.Paths,
		Packages:       in.Packages,
		LifecycleState: lifecycleState,
		ColumnID:       in.ColumnID,
		Position:       position,
		Title:          in.Title,
		Description:    in.Description,
		Priority:       in.Priority,
		DueAt:          in.DueAt,
		Labels:         in.Labels,
		Metadata:       mergedMetadata,
		CreatedByActor: firstNonEmptyTrimmed(in.CreatedByActor, resolvedActor.ActorID),
		CreatedByName:  firstNonEmptyTrimmed(in.CreatedByName, resolvedActor.ActorName, in.CreatedByActor, resolvedActor.ActorID),
		UpdatedByActor: firstNonEmptyTrimmed(in.UpdatedByActor, resolvedActor.ActorID, in.CreatedByActor),
		UpdatedByName:  firstNonEmptyTrimmed(in.UpdatedByName, resolvedActor.ActorName, in.UpdatedByActor, resolvedActor.ActorID, in.CreatedByName, in.CreatedByActor),
		UpdatedByType:  actorType,
	}, s.clock())
	if err != nil {
		return domain.ActionItem{}, err
	}

	if err := s.repo.CreateActionItem(ctx, actionItem); err != nil {
		return domain.ActionItem{}, err
	}
	if _, err := s.enqueueActionItemEmbedding(ctx, actionItem, false, "task_created"); err != nil {
		return domain.ActionItem{}, err
	}
	// Per droplet 3.20: when a level_1 numbered drop lands (parent is the
	// project root, drop_number > 0), auto-generate the 5 STEWARD-owned
	// drop-end findings + the refinements-gate confluence. The seeder is a
	// no-op for non-numbered drops (drop_number == 0) and for non-level_1
	// items (parent_id != "") so it is safe to call unconditionally for
	// numbered level_1 drops only. The check excludes the seeder's own
	// findings (which carry drop_number=N but live under STEWARD anchors,
	// so parent_id != "") to prevent infinite recursion.
	if s.autoSeedSteward && strings.TrimSpace(actionItem.ParentID) == "" && actionItem.DropNumber > 0 {
		if err := s.seedDropFindingsAndGate(ctx, actionItem); err != nil {
			return domain.ActionItem{}, err
		}
	}
	return actionItem, nil
}

// MoveActionItem moves actionItem.
func (s *Service) MoveActionItem(ctx context.Context, actionItemID, toColumnID string, position int) (domain.ActionItem, error) {
	actionItem, err := s.repo.GetActionItem(ctx, actionItemID)
	if err != nil {
		return domain.ActionItem{}, err
	}
	guardScopes, err := s.capabilityScopesForActionItemLineage(ctx, actionItem)
	if err != nil {
		return domain.ActionItem{}, err
	}
	columns, err := s.repo.ListColumns(ctx, actionItem.ProjectID, true)
	if err != nil {
		return domain.ActionItem{}, err
	}
	fromState := lifecycleStateForColumnID(columns, actionItem.ColumnID)
	if fromState == "" {
		fromState = actionItem.LifecycleState
	}
	toState := lifecycleStateForColumnID(columns, toColumnID)
	if toState == "" {
		toState = fromState
	}
	moveAction := domain.CapabilityActionEditNode
	switch {
	case toState == domain.StateComplete:
		moveAction = domain.CapabilityActionMarkComplete
	case toState == domain.StateFailed:
		moveAction = domain.CapabilityActionMarkFailed
	case fromState == domain.StateTodo && toState == domain.StateInProgress:
		moveAction = domain.CapabilityActionMarkInProgress
	}
	if err := s.enforceMutationGuardAcrossScopes(ctx, actionItem.ProjectID, currentMutationActorType(ctx, ""), guardScopes, moveAction); err != nil {
		return domain.ActionItem{}, err
	}
	// Terminal-state guard: transitions FROM done or failed are blocked until
	// override auth (D3) is implemented. Once D3 lands, this guard will check
	// for an override token instead of blocking unconditionally.
	if domain.IsTerminalState(fromState) && fromState != toState {
		return domain.ActionItem{}, fmt.Errorf("%w: cannot transition from terminal state %q without override auth", domain.ErrTransitionBlocked, fromState)
	}
	if fromState == domain.StateTodo && toState == domain.StateInProgress {
		if unmet := actionItem.StartCriteriaUnmet(); len(unmet) > 0 {
			return domain.ActionItem{}, fmt.Errorf("%w: start criteria unmet (%s)", domain.ErrTransitionBlocked, strings.Join(unmet, ", "))
		}
	}
	if toState == domain.StateComplete {
		projectActionItems, listErr := s.ListActionItems(ctx, actionItem.ProjectID, true)
		if listErr != nil {
			return domain.ActionItem{}, listErr
		}
		if blockErr := s.ensureActionItemCompletionBlockersClear(ctx, actionItem, projectActionItems); blockErr != nil {
			return domain.ActionItem{}, blockErr
		}
		if blockErr := s.ensureActionItemCompletionAttentionClear(ctx, actionItem); blockErr != nil {
			return domain.ActionItem{}, blockErr
		}
	}
	if err := actionItem.Move(toColumnID, position, s.clock()); err != nil {
		return domain.ActionItem{}, err
	}
	if err := actionItem.SetLifecycleState(toState, s.clock()); err != nil {
		return domain.ActionItem{}, err
	}
	applyMutationActorToActionItem(ctx, &actionItem)
	if err := s.repo.UpdateActionItem(ctx, actionItem); err != nil {
		return domain.ActionItem{}, err
	}
	if _, err := s.enqueueActionItemEmbedding(ctx, actionItem, false, "task_moved"); err != nil {
		return domain.ActionItem{}, err
	}
	// Per droplet 3.22 + finding 5.C.11: when a refinements-gate transitions
	// to complete, run the safety-net check that warns the dev if any
	// drop_number=N items remain non-terminal at gate-close time. The check
	// is a regression net for the "drop-orch forgot to update blocked_by"
	// failure mode — it documents the cause without papering over the
	// underlying parent-blocks-on-incomplete-child invariant on the level_1
	// drop. Non-gate moves and non-complete transitions are no-ops inside
	// raiseRefinementsGateForgottenAttention.
	if toState == domain.StateComplete && isRefinementsGate(actionItem) {
		if err := s.raiseRefinementsGateForgottenAttention(ctx, actionItem); err != nil {
			return domain.ActionItem{}, err
		}
	}
	return actionItem, nil
}

// RestoreActionItem restores actionItem.
func (s *Service) RestoreActionItem(ctx context.Context, actionItemID string) (domain.ActionItem, error) {
	actionItem, err := s.repo.GetActionItem(ctx, actionItemID)
	if err != nil {
		return domain.ActionItem{}, err
	}
	guardScopes, err := s.capabilityScopesForActionItemLineage(ctx, actionItem)
	if err != nil {
		return domain.ActionItem{}, err
	}
	// Guard enforcement must follow the caller's request actor, not historical actionItem attribution.
	guardActorType := currentMutationActorType(ctx, "")
	if err := s.enforceMutationGuardAcrossScopes(ctx, actionItem.ProjectID, guardActorType, guardScopes, domain.CapabilityActionArchiveOrCleanup); err != nil {
		return domain.ActionItem{}, err
	}
	actionItem.Restore(s.clock())
	columns, err := s.repo.ListColumns(ctx, actionItem.ProjectID, true)
	if err != nil {
		return domain.ActionItem{}, err
	}
	restoredState := lifecycleStateForColumnID(columns, actionItem.ColumnID)
	if restoredState == "" {
		restoredState = domain.StateTodo
	}
	if err := actionItem.SetLifecycleState(restoredState, s.clock()); err != nil {
		return domain.ActionItem{}, err
	}
	applyMutationActorToActionItem(ctx, &actionItem)
	if err := s.repo.UpdateActionItem(ctx, actionItem); err != nil {
		return domain.ActionItem{}, err
	}
	if _, err := s.enqueueActionItemEmbedding(ctx, actionItem, false, "task_restored"); err != nil {
		return domain.ActionItem{}, err
	}
	return actionItem, nil
}

// RenameActionItem renames actionItem.
func (s *Service) RenameActionItem(ctx context.Context, actionItemID, title string) (domain.ActionItem, error) {
	actionItem, err := s.repo.GetActionItem(ctx, actionItemID)
	if err != nil {
		return domain.ActionItem{}, err
	}
	guardScopes, err := s.capabilityScopesForActionItemLineage(ctx, actionItem)
	if err != nil {
		return domain.ActionItem{}, err
	}
	if err := s.enforceMutationGuardAcrossScopes(ctx, actionItem.ProjectID, currentMutationActorType(ctx, ""), guardScopes, domain.CapabilityActionEditNode); err != nil {
		return domain.ActionItem{}, err
	}
	if err := actionItem.UpdateDetails(title, actionItem.Description, actionItem.Priority, actionItem.DueAt, actionItem.Labels, s.clock()); err != nil {
		return domain.ActionItem{}, err
	}
	applyMutationActorToActionItem(ctx, &actionItem)
	if err := s.repo.UpdateActionItem(ctx, actionItem); err != nil {
		return domain.ActionItem{}, err
	}
	if _, err := s.enqueueActionItemEmbedding(ctx, actionItem, false, "task_renamed"); err != nil {
		return domain.ActionItem{}, err
	}
	if _, err := s.enqueueThreadContextEmbedding(ctx, domain.CommentTarget{
		ProjectID:  actionItem.ProjectID,
		TargetType: snapshotCommentTargetTypeForActionItem(actionItem),
		TargetID:   actionItem.ID,
	}, false, "task_renamed"); err != nil && !errors.Is(err, ErrNotFound) {
		return domain.ActionItem{}, err
	}
	return actionItem, nil
}

// UpdateActionItem updates state for the requested operation.
func (s *Service) UpdateActionItem(ctx context.Context, in UpdateActionItemInput) (domain.ActionItem, error) {
	actionItem, err := s.repo.GetActionItem(ctx, in.ActionItemID)
	if err != nil {
		return domain.ActionItem{}, err
	}
	ctx, resolvedActor, hasResolvedActor := withResolvedMutationActor(ctx, in.UpdatedBy, in.UpdatedByName, in.UpdatedType)
	actorType := currentMutationActorType(ctx, in.UpdatedType)
	guardScopes, err := s.capabilityScopesForActionItemLineage(ctx, actionItem)
	if err != nil {
		return domain.ActionItem{}, err
	}
	if err := s.enforceMutationGuardAcrossScopes(ctx, actionItem.ProjectID, actorType, guardScopes, domain.CapabilityActionEditNode); err != nil {
		return domain.ActionItem{}, err
	}
	if hasResolvedActor && strings.TrimSpace(resolvedActor.ActorID) != "" {
		actionItem.UpdatedByActor = resolvedActor.ActorID
		actionItem.UpdatedByName = firstNonEmptyTrimmed(resolvedActor.ActorName, resolvedActor.ActorID)
		actionItem.UpdatedByType = actorType
	} else if updatedBy := strings.TrimSpace(in.UpdatedBy); updatedBy != "" {
		actionItem.UpdatedByActor = updatedBy
		actionItem.UpdatedByName = firstNonEmptyTrimmed(in.UpdatedByName, updatedBy)
		actionItem.UpdatedByType = actorType
	}
	applyMutationActorToActionItem(ctx, &actionItem)
	priority := in.Priority
	if strings.TrimSpace(string(priority)) == "" {
		priority = actionItem.Priority
	}
	if err := actionItem.UpdateDetails(in.Title, in.Description, priority, in.DueAt, in.Labels, s.clock()); err != nil {
		return domain.ActionItem{}, err
	}
	// Role update: empty input preserves the existing role (no-op). A
	// non-empty input is normalized + validated against the closed Role
	// enum; mismatches surface as ErrInvalidRole. Mirrors the validation
	// performed by domain.NewActionItem on create.
	if normalized := domain.NormalizeRole(in.Role); normalized != "" {
		if !domain.IsValidRole(normalized) {
			return domain.ActionItem{}, domain.ErrInvalidRole
		}
		actionItem.Role = normalized
		actionItem.UpdatedAt = s.clock().UTC()
	}
	// StructuralType update: empty input preserves the existing value
	// (no-op) — mirrors Role's required-on-create / optional-on-update
	// split. A non-empty input is normalized + validated against the
	// closed StructuralType enum; mismatches surface as
	// ErrInvalidStructuralType.
	if normalized := domain.NormalizeStructuralType(in.StructuralType); normalized != "" {
		if !domain.IsValidStructuralType(normalized) {
			return domain.ActionItem{}, domain.ErrInvalidStructuralType
		}
		actionItem.StructuralType = normalized
		actionItem.UpdatedAt = s.clock().UTC()
	}
	// Owner / DropNumber / Persistent / DevGated updates use pointer
	// sentinels: nil preserves the existing value (no-op); non-nil applies
	// the dereferenced value. Pre-droplet-3.21 callers that did NOT supply
	// these fields are preserved cleanly; STEWARD-owned anchor nodes seeded
	// with Persistent=true do not get clobbered by description-only edits.
	// Per L13, all four are domain primitives — not STEWARD-specific.
	if in.Owner != nil {
		actionItem.Owner = strings.TrimSpace(*in.Owner)
		actionItem.UpdatedAt = s.clock().UTC()
	}
	if in.DropNumber != nil {
		if *in.DropNumber < 0 {
			return domain.ActionItem{}, domain.ErrInvalidDropNumber
		}
		actionItem.DropNumber = *in.DropNumber
		actionItem.UpdatedAt = s.clock().UTC()
	}
	if in.Persistent != nil {
		actionItem.Persistent = *in.Persistent
		actionItem.UpdatedAt = s.clock().UTC()
	}
	if in.DevGated != nil {
		actionItem.DevGated = *in.DevGated
		actionItem.UpdatedAt = s.clock().UTC()
	}
	// Paths update uses pointer-sentinel: nil preserves the existing slice
	// (no-op); non-nil applies the dereferenced slice through the canonical
	// domain.NormalizeActionItemPaths gate so the same trim/dedupe/forward-
	// slash rules NewActionItem enforces apply equally on update. Empty
	// dereferenced slice clears all declared paths (explicit caller intent).
	if in.Paths != nil {
		normalized, err := domain.NormalizeActionItemPaths(*in.Paths)
		if err != nil {
			return domain.ActionItem{}, err
		}
		actionItem.Paths = normalized
		actionItem.UpdatedAt = s.clock().UTC()
	}
	// Packages update uses pointer-sentinel: nil preserves the existing
	// slice; non-nil applies the dereferenced slice through
	// domain.NormalizeActionItemPackages so the create-time trim/dedupe
	// rules apply equally on update. Empty dereferenced slice clears all
	// declared packages (explicit caller intent).
	if in.Packages != nil {
		normalized, err := domain.NormalizeActionItemPackages(*in.Packages)
		if err != nil {
			return domain.ActionItem{}, err
		}
		actionItem.Packages = normalized
		actionItem.UpdatedAt = s.clock().UTC()
	}
	// Re-check the coverage invariant against the post-apply pair so that
	// paired Paths / Packages updates land atomically: a caller may, for
	// example, set Paths to a non-empty slice and Packages to nil within
	// the same call (preserving prior Packages), or supply both together.
	// The invariant is the same as in domain.NewActionItem: non-empty
	// Paths requires non-empty Packages. Re-checking here (vs only inside
	// the per-field branches above) catches the edge case where Paths is
	// populated on update while existing Packages is empty, or where
	// Packages is explicitly cleared while existing Paths remains
	// populated. WAVE_1_PLAN.md §1.2.
	if (in.Paths != nil || in.Packages != nil) && len(actionItem.Paths) > 0 && len(actionItem.Packages) == 0 {
		return domain.ActionItem{}, domain.ErrInvalidPackages
	}
	if in.Metadata != nil {
		var parent *domain.ActionItem
		if strings.TrimSpace(actionItem.ParentID) != "" {
			parentActionItem, parentErr := s.repo.GetActionItem(ctx, actionItem.ParentID)
			if parentErr != nil {
				return domain.ActionItem{}, parentErr
			}
			parent = &parentActionItem
		}
		if _, validateErr := s.validateActionItemKind(ctx, actionItem.ProjectID, domain.KindID(actionItem.Kind), actionItem.Scope, parent, in.Metadata.KindPayload); validateErr != nil {
			return domain.ActionItem{}, validateErr
		}
		if err := actionItem.UpdatePlanningMetadata(*in.Metadata, actionItem.UpdatedByActor, actionItem.UpdatedByType, s.clock()); err != nil {
			return domain.ActionItem{}, err
		}
	}
	if err := s.repo.UpdateActionItem(ctx, actionItem); err != nil {
		return domain.ActionItem{}, err
	}
	if _, err := s.enqueueActionItemEmbedding(ctx, actionItem, false, "task_updated"); err != nil {
		return domain.ActionItem{}, err
	}
	if _, err := s.enqueueThreadContextEmbedding(ctx, domain.CommentTarget{
		ProjectID:  actionItem.ProjectID,
		TargetType: snapshotCommentTargetTypeForActionItem(actionItem),
		TargetID:   actionItem.ID,
	}, false, "task_updated"); err != nil && !errors.Is(err, ErrNotFound) {
		return domain.ActionItem{}, err
	}
	return actionItem, nil
}

// DeleteActionItem deletes actionItem.
func (s *Service) DeleteActionItem(ctx context.Context, actionItemID string, mode DeleteMode) error {
	if mode == "" {
		mode = s.defaultDeleteMode
	}

	switch mode {
	case DeleteModeArchive:
		actionItem, err := s.repo.GetActionItem(ctx, actionItemID)
		if err != nil {
			return err
		}
		guardScopes, guardErr := s.capabilityScopesForActionItemLineage(ctx, actionItem)
		if guardErr != nil {
			return guardErr
		}
		if err := s.enforceMutationGuardAcrossScopes(ctx, actionItem.ProjectID, currentMutationActorType(ctx, ""), guardScopes, domain.CapabilityActionArchiveOrCleanup); err != nil {
			return err
		}
		actionItem.Archive(s.clock())
		applyMutationActorToActionItem(ctx, &actionItem)
		return s.repo.UpdateActionItem(ctx, actionItem)
	case DeleteModeHard:
		actionItem, err := s.repo.GetActionItem(ctx, actionItemID)
		if err != nil {
			return err
		}
		guardScopes, guardErr := s.capabilityScopesForActionItemLineage(ctx, actionItem)
		if guardErr != nil {
			return guardErr
		}
		if err := s.enforceMutationGuardAcrossScopes(ctx, actionItem.ProjectID, currentMutationActorType(ctx, ""), guardScopes, domain.CapabilityActionArchiveOrCleanup); err != nil {
			return err
		}
		if err := s.repo.DeleteActionItem(ctx, actionItemID); err != nil {
			return err
		}
		if s.searchIndex != nil {
			if err := s.searchIndex.DeleteEmbeddingDocument(ctx, EmbeddingSubjectTypeWorkItem, actionItemID); err != nil {
				return err
			}
			threadSubjectID := BuildThreadContextSubjectID(domain.CommentTarget{
				ProjectID:  actionItem.ProjectID,
				TargetType: snapshotCommentTargetTypeForActionItem(actionItem),
				TargetID:   actionItem.ID,
			})
			if threadSubjectID != "" {
				if err := s.searchIndex.DeleteEmbeddingDocument(ctx, EmbeddingSubjectTypeThreadContext, threadSubjectID); err != nil {
					return err
				}
			}
		}
		if s.embeddingLifecycle != nil {
			if err := s.embeddingLifecycle.DeleteEmbeddingSubject(ctx, EmbeddingSubjectTypeWorkItem, actionItemID); err != nil {
				return err
			}
			threadSubjectID := BuildThreadContextSubjectID(domain.CommentTarget{
				ProjectID:  actionItem.ProjectID,
				TargetType: snapshotCommentTargetTypeForActionItem(actionItem),
				TargetID:   actionItem.ID,
			})
			if threadSubjectID != "" {
				if err := s.embeddingLifecycle.DeleteEmbeddingSubject(ctx, EmbeddingSubjectTypeThreadContext, threadSubjectID); err != nil {
					return err
				}
			}
		}
		return nil
	default:
		return ErrInvalidDeleteMode
	}
}

// ListProjects lists projects.
func (s *Service) ListProjects(ctx context.Context, includeArchived bool) ([]domain.Project, error) {
	return s.repo.ListProjects(ctx, includeArchived)
}

// ListColumns lists columns.
func (s *Service) ListColumns(ctx context.Context, projectID string, includeArchived bool) ([]domain.Column, error) {
	columns, err := s.repo.ListColumns(ctx, projectID, includeArchived)
	if err != nil {
		return nil, err
	}
	slices.SortFunc(columns, func(a, b domain.Column) int {
		return a.Position - b.Position
	})
	return columns, nil
}

// ListActionItems lists tasks.
func (s *Service) ListActionItems(ctx context.Context, projectID string, includeArchived bool) ([]domain.ActionItem, error) {
	tasks, err := s.repo.ListActionItems(ctx, projectID, includeArchived)
	if err != nil {
		return nil, err
	}
	slices.SortFunc(tasks, func(a, b domain.ActionItem) int {
		if a.ColumnID == b.ColumnID {
			return a.Position - b.Position
		}
		return strings.Compare(a.ColumnID, b.ColumnID)
	})
	return tasks, nil
}

// CreateComment creates a comment for a concrete project target.
func (s *Service) CreateComment(ctx context.Context, in CreateCommentInput) (domain.Comment, error) {
	target, err := normalizeCommentTargetInput(in.ProjectID, in.TargetType, in.TargetID)
	if err != nil {
		return domain.Comment{}, err
	}
	ctx, resolvedActor, _ := withResolvedMutationActor(ctx, in.ActorID, in.ActorName, in.ActorType)
	actorType := normalizeActorTypeInput(in.ActorType)
	body := strings.TrimSpace(in.BodyMarkdown)
	if body == "" {
		return domain.Comment{}, domain.ErrInvalidBodyMarkdown
	}

	guardScopes := []mutationScopeCandidate{
		newProjectMutationScopeCandidate(target.ProjectID),
	}
	if target.TargetType != domain.CommentTargetTypeProject {
		actionItem, actionItemErr := s.repo.GetActionItem(ctx, target.TargetID)
		if actionItemErr != nil {
			return domain.Comment{}, actionItemErr
		}
		if actionItem.ProjectID != target.ProjectID {
			return domain.Comment{}, ErrNotFound
		}
		guardScopes, err = s.capabilityScopesForActionItemLineage(ctx, actionItem)
		if err != nil {
			return domain.Comment{}, err
		}
	}
	if err := s.enforceMutationGuardAcrossScopes(ctx, target.ProjectID, actorType, guardScopes, domain.CapabilityActionComment); err != nil {
		return domain.Comment{}, err
	}
	if err := s.ensureCommentTargetExists(ctx, target); err != nil {
		return domain.Comment{}, err
	}

	comment, err := domain.NewComment(domain.CommentInput{
		ID:           s.idGen(),
		ProjectID:    target.ProjectID,
		TargetType:   target.TargetType,
		TargetID:     target.TargetID,
		Summary:      in.Summary,
		BodyMarkdown: body,
		ActorID:      firstNonEmptyTrimmed(resolvedActor.ActorID, in.ActorID),
		ActorName:    firstNonEmptyTrimmed(resolvedActor.ActorName, in.ActorName),
		ActorType:    actorType,
	}, s.clock())
	if err != nil {
		return domain.Comment{}, err
	}
	if err := s.repo.CreateComment(ctx, comment); err != nil {
		return domain.Comment{}, err
	}
	if err := s.syncCommentInboxAttention(ctx, comment); err != nil {
		return domain.Comment{}, err
	}
	if _, err := s.enqueueThreadContextEmbedding(ctx, domain.CommentTarget{
		ProjectID:  comment.ProjectID,
		TargetType: comment.TargetType,
		TargetID:   comment.TargetID,
	}, false, "comment_created"); err != nil {
		return domain.Comment{}, err
	}
	s.publishCommentChanged(target)
	s.publishAttentionChanged(target.ProjectID)
	return comment, nil
}

// ensureCommentTargetExists validates one comment target reference before mutation.
func (s *Service) ensureCommentTargetExists(ctx context.Context, target domain.CommentTarget) error {
	if _, err := s.repo.GetProject(ctx, target.ProjectID); err != nil {
		return err
	}
	if target.TargetType == domain.CommentTargetTypeProject {
		if target.TargetID != target.ProjectID {
			return ErrNotFound
		}
		return nil
	}
	actionItem, err := s.repo.GetActionItem(ctx, target.TargetID)
	if err != nil {
		return err
	}
	if actionItem.ProjectID != target.ProjectID {
		return ErrNotFound
	}
	return nil
}

// ListCommentsByTarget lists comments for a specific target in deterministic order.
func (s *Service) ListCommentsByTarget(ctx context.Context, in ListCommentsByTargetInput) ([]domain.Comment, error) {
	target, err := normalizeCommentTargetInput(in.ProjectID, in.TargetType, in.TargetID)
	if err != nil {
		return nil, err
	}
	waitKey := commentLiveWaitKey(target)
	baselineSequence, err := s.liveWaitBaselineSequence(ctx, LiveWaitEventCommentChanged, waitKey)
	if err != nil {
		return nil, err
	}
	comments, err := s.repo.ListCommentsByTarget(ctx, target)
	if err != nil {
		return nil, err
	}
	if in.WaitTimeout > 0 {
		woke, err := s.waitForLiveEvent(ctx, LiveWaitEventCommentChanged, waitKey, baselineSequence, in.WaitTimeout)
		if err != nil {
			return nil, err
		}
		if woke {
			comments, err = s.repo.ListCommentsByTarget(ctx, target)
			if err != nil {
				return nil, err
			}
		}
	}
	slices.SortFunc(comments, func(a, b domain.Comment) int {
		switch {
		case a.CreatedAt.Before(b.CreatedAt):
			return -1
		case a.CreatedAt.After(b.CreatedAt):
			return 1
		default:
			return strings.Compare(a.ID, b.ID)
		}
	})
	return comments, nil
}

// ListProjectChangeEvents lists recent change events for a project.
func (s *Service) ListProjectChangeEvents(ctx context.Context, projectID string, limit int) ([]domain.ChangeEvent, error) {
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		return nil, domain.ErrInvalidID
	}
	return s.repo.ListProjectChangeEvents(ctx, projectID, limit)
}

// GetProjectDependencyRollup summarizes dependency and blocked-state counts.
func (s *Service) GetProjectDependencyRollup(ctx context.Context, projectID string) (domain.DependencyRollup, error) {
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		return domain.DependencyRollup{}, domain.ErrInvalidID
	}
	if _, err := s.repo.GetProject(ctx, projectID); err != nil {
		return domain.DependencyRollup{}, err
	}
	tasks, err := s.repo.ListActionItems(ctx, projectID, false)
	if err != nil {
		return domain.DependencyRollup{}, err
	}
	return buildDependencyRollup(projectID, tasks), nil
}

// ListChildActionItems lists child tasks for a parent within the same project.
func (s *Service) ListChildActionItems(ctx context.Context, projectID, parentID string, includeArchived bool) ([]domain.ActionItem, error) {
	parentID = strings.TrimSpace(parentID)
	if parentID == "" {
		return nil, domain.ErrInvalidParentID
	}
	tasks, err := s.ListActionItems(ctx, projectID, includeArchived)
	if err != nil {
		return nil, err
	}
	out := make([]domain.ActionItem, 0)
	for _, actionItem := range tasks {
		if actionItem.ParentID == parentID {
			out = append(out, actionItem)
		}
	}
	return out, nil
}

// ReparentActionItem changes parent actionItem relationship.
func (s *Service) ReparentActionItem(ctx context.Context, actionItemID, parentID string) (domain.ActionItem, error) {
	actionItem, err := s.repo.GetActionItem(ctx, actionItemID)
	if err != nil {
		return domain.ActionItem{}, err
	}
	actionItemScopes, err := s.capabilityScopesForActionItemLineage(ctx, actionItem)
	if err != nil {
		return domain.ActionItem{}, err
	}
	if err := s.enforceMutationGuardAcrossScopes(ctx, actionItem.ProjectID, currentMutationActorType(ctx, ""), actionItemScopes, domain.CapabilityActionEditNode); err != nil {
		return domain.ActionItem{}, err
	}
	parentID = strings.TrimSpace(parentID)
	var parent *domain.ActionItem
	if parentID != "" {
		parentActionItem, parentErr := s.repo.GetActionItem(ctx, parentID)
		if parentErr != nil {
			return domain.ActionItem{}, parentErr
		}
		if parentActionItem.ProjectID != actionItem.ProjectID {
			return domain.ActionItem{}, domain.ErrInvalidParentID
		}
		parent = &parentActionItem
		parentScopes, scopeErr := s.capabilityScopesForActionItemLineage(ctx, parentActionItem)
		if scopeErr != nil {
			return domain.ActionItem{}, scopeErr
		}
		if err := s.enforceMutationGuardAcrossScopes(ctx, actionItem.ProjectID, currentMutationActorType(ctx, ""), parentScopes, domain.CapabilityActionEditNode); err != nil {
			return domain.ActionItem{}, err
		}
		tasks, listErr := s.repo.ListActionItems(ctx, actionItem.ProjectID, true)
		if listErr != nil {
			return domain.ActionItem{}, listErr
		}
		if wouldCreateParentCycle(actionItem.ID, parentActionItem.ID, tasks) {
			return domain.ActionItem{}, domain.ErrInvalidParentID
		}
	}
	if _, err := s.validateActionItemKind(ctx, actionItem.ProjectID, domain.KindID(actionItem.Kind), actionItem.Scope, parent, actionItem.Metadata.KindPayload); err != nil {
		return domain.ActionItem{}, err
	}
	if err := actionItem.Reparent(parentID, s.clock()); err != nil {
		return domain.ActionItem{}, err
	}
	applyMutationActorToActionItem(ctx, &actionItem)
	if err := s.repo.UpdateActionItem(ctx, actionItem); err != nil {
		return domain.ActionItem{}, err
	}
	return actionItem, nil
}

// SearchActionItemMatches finds actionItem matches using project, state, and archive filters.
func (s *Service) SearchActionItemMatches(ctx context.Context, in SearchActionItemsFilter) ([]ActionItemMatch, error) {
	result, err := s.SearchActionItems(ctx, in)
	if err != nil {
		return nil, err
	}
	return result.Matches, nil
}

// SearchActionItems finds actionItem matches and includes execution metadata for operator-visible surfaces.
func (s *Service) SearchActionItems(ctx context.Context, in SearchActionItemsFilter) (SearchActionItemMatchesResult, error) {
	mode, err := normalizeSearchMode(in.Mode)
	if err != nil {
		return SearchActionItemMatchesResult{}, err
	}
	sortOrder, err := normalizeSearchSort(in.Sort)
	if err != nil {
		return SearchActionItemMatchesResult{}, err
	}
	limit, offset, err := normalizeSearchPagination(in.Limit, in.Offset)
	if err != nil {
		return SearchActionItemMatchesResult{}, err
	}

	stateFilter := map[string]struct{}{}
	for _, raw := range in.States {
		state := strings.TrimSpace(strings.ToLower(raw))
		if state == "" {
			continue
		}
		stateFilter[state] = struct{}{}
	}
	levelFilter := normalizeLowerFilterSet(in.Levels)
	kindFilter := normalizeLowerFilterSet(in.Kinds)
	labelsAnyFilter := normalizeLowerFilterSet(in.LabelsAny)
	labelsAllFilter := normalizeLowerFilterSet(in.LabelsAll)
	if invalid := unsupportedSearchLevels(levelFilter); len(invalid) > 0 {
		log.Warn("search request includes unsupported levels filter values", "levels", strings.Join(invalid, ","))
	}
	allowAllStates := len(stateFilter) == 0
	wantsArchivedState := allowAllStates
	if !allowAllStates {
		_, wantsArchivedState = stateFilter["archived"]
	}

	targetProjects := []domain.Project{}
	if in.CrossProject {
		projects, err := s.repo.ListProjects(ctx, in.IncludeArchived)
		if err != nil {
			return SearchActionItemMatchesResult{}, err
		}
		targetProjects = append(targetProjects, projects...)
	} else {
		projectID := strings.TrimSpace(in.ProjectID)
		if projectID == "" {
			return SearchActionItemMatchesResult{}, domain.ErrInvalidID
		}
		project, err := s.repo.GetProject(ctx, projectID)
		if err != nil {
			return SearchActionItemMatchesResult{}, err
		}
		if !in.IncludeArchived && project.ArchivedAt != nil {
			return SearchActionItemMatchesResult{
				Matches:          []ActionItemMatch{},
				RequestedMode:    mode,
				EffectiveMode:    mode,
				EmbeddingSummary: EmbeddingSummary{},
			}, nil
		}
		targetProjects = append(targetProjects, project)
	}

	query := strings.TrimSpace(strings.ToLower(in.Query))
	out := make([]ActionItemMatch, 0)
	lexicalScores := map[string]float64{}
	projectIDs := make([]string, 0, len(targetProjects))
	for _, project := range targetProjects {
		projectIDs = append(projectIDs, project.ID)
		columns, err := s.repo.ListColumns(ctx, project.ID, true)
		if err != nil {
			return SearchActionItemMatchesResult{}, err
		}
		stateByColumn := make(map[string]string, len(columns))
		for _, column := range columns {
			stateByColumn[column.ID] = normalizeStateID(column.Name)
		}

		tasks, err := s.repo.ListActionItems(ctx, project.ID, true)
		if err != nil {
			return SearchActionItemMatchesResult{}, err
		}
		for _, actionItem := range tasks {
			stateID := stateByColumn[actionItem.ColumnID]
			if stateID == "" {
				stateID = string(actionItem.LifecycleState)
			}
			if stateID == "" {
				stateID = "todo"
			}
			if actionItem.ArchivedAt != nil {
				if !in.IncludeArchived || !wantsArchivedState {
					continue
				}
				stateID = "archived"
			} else if !allowAllStates {
				if _, ok := stateFilter[stateID]; !ok {
					continue
				}
			}
			if !actionItemMatchesExtendedSearchFilters(actionItem, levelFilter, kindFilter, labelsAnyFilter, labelsAllFilter) {
				continue
			}
			lexicalScores[actionItem.ID] = actionItemLexicalMatchScore(actionItem, query)

			out = append(out, ActionItemMatch{
				Project:    project,
				ActionItem: actionItem,
				StateID:    stateID,
			})
		}
	}

	semanticScores := map[string]float64{}
	semanticSubjects := map[string]EmbeddingSearchMatch{}
	semanticReady := false
	effectiveMode := mode
	fallbackReason := ""
	if len(projectIDs) > 0 && query != "" && (mode == SearchModeSemantic || mode == SearchModeHybrid) &&
		s.embeddingGenerator != nil && s.searchIndex != nil && s.embeddingLifecycle != nil {
		queryVectors, embedErr := s.embeddingGenerator.Embed(ctx, []string{query})
		if embedErr == nil && len(queryVectors) > 0 && len(queryVectors[0]) > 0 {
			semanticLimit := max(limit*4, s.searchSemanticK)
			rows, searchErr := s.searchIndex.SearchEmbeddingDocuments(ctx, EmbeddingSearchInput{
				ProjectIDs:        projectIDs,
				SearchTargetTypes: []EmbeddingSearchTargetType{EmbeddingSearchTargetTypeWorkItem},
				Vector:            queryVectors[0],
				Limit:             semanticLimit,
			})
			if searchErr == nil {
				readyRows, filterErr := s.filterReadySemanticMatches(ctx, projectIDs, rows)
				if filterErr != nil {
					fallbackReason = "embedding_status_unavailable"
				} else {
					for _, row := range readyRows {
						actionItemID := strings.TrimSpace(row.SearchTargetID)
						if actionItemID == "" || row.SearchTargetType != EmbeddingSearchTargetTypeWorkItem {
							continue
						}
						score := clamp01(row.Similarity)
						if previous, ok := semanticScores[actionItemID]; !ok || score > previous {
							semanticScores[actionItemID] = score
							semanticSubjects[actionItemID] = row
						}
					}
				}
				if len(semanticScores) > 0 {
					semanticReady = true
				} else {
					fallbackReason = "semantic_index_not_ready"
				}
			} else {
				fallbackReason = "vector_search_failed"
			}
		} else {
			fallbackReason = "query_embedding_failed"
		}
	}
	if (mode == SearchModeSemantic || mode == SearchModeHybrid) && !semanticReady {
		effectiveMode = SearchModeKeyword
		if fallbackReason == "" {
			switch {
			case s.embeddingGenerator == nil:
				fallbackReason = "embedding_runtime_unavailable"
			case s.searchIndex == nil:
				fallbackReason = "search_index_unavailable"
			case s.embeddingLifecycle == nil:
				fallbackReason = "embedding_lifecycle_unavailable"
			default:
				fallbackReason = "semantic_unavailable"
			}
		}
	}

	if query != "" {
		filtered := make([]ActionItemMatch, 0, len(out))
		for _, match := range out {
			actionItemID := match.ActionItem.ID
			lexicalScore := lexicalScores[actionItemID]
			_, hasSemantic := semanticScores[actionItemID]
			switch effectiveMode {
			case SearchModeKeyword:
				if lexicalScore <= 0 {
					continue
				}
			case SearchModeSemantic:
				if !hasSemantic {
					continue
				}
			case SearchModeHybrid:
				if lexicalScore <= 0 && !hasSemantic {
					continue
				}
			}
			filtered = append(filtered, match)
		}
		out = filtered
	}

	rankScores := map[string]float64{}
	if query != "" {
		for idx := range out {
			actionItemID := out[idx].ActionItem.ID
			lexicalScore := clamp01(lexicalScores[actionItemID])
			semanticScore := clamp01(semanticScores[actionItemID])
			out[idx].SemanticScore = semanticScore
			out[idx].UsedSemantic = semanticScore > 0 && effectiveMode != SearchModeKeyword
			switch effectiveMode {
			case SearchModeSemantic:
				rankScores[actionItemID] = semanticScore
			case SearchModeHybrid:
				rankScores[actionItemID] = (s.searchLexicalW * lexicalScore) + (s.searchSemanticW * semanticScore)
			default:
				rankScores[actionItemID] = lexicalScore
			}
		}
	}

	slices.SortFunc(out, func(a, b ActionItemMatch) int {
		switch sortOrder {
		case SearchSortTitleAsc:
			left := strings.ToLower(strings.TrimSpace(a.ActionItem.Title))
			right := strings.ToLower(strings.TrimSpace(b.ActionItem.Title))
			if cmp := strings.Compare(left, right); cmp != 0 {
				return cmp
			}
			if cmp := strings.Compare(a.ActionItem.Title, b.ActionItem.Title); cmp != 0 {
				return cmp
			}
		case SearchSortCreatedAtDesc:
			if cmp := compareTimeDesc(a.ActionItem.CreatedAt, b.ActionItem.CreatedAt); cmp != 0 {
				return cmp
			}
		case SearchSortUpdatedAtDesc:
			if cmp := compareTimeDesc(a.ActionItem.UpdatedAt, b.ActionItem.UpdatedAt); cmp != 0 {
				return cmp
			}
		case SearchSortRankDesc:
			if query != "" {
				if cmp := compareFloat64Desc(rankScores[a.ActionItem.ID], rankScores[b.ActionItem.ID]); cmp != 0 {
					return cmp
				}
			}
		}
		return compareActionItemMatchRankDesc(a, b)
	})

	if offset < len(out) {
		end := min(offset+limit, len(out))
		out = append([]ActionItemMatch(nil), out[offset:end]...)
	} else {
		out = []ActionItemMatch{}
	}

	s.annotateActionItemMatchesWithEmbeddingState(ctx, projectIDs, out, semanticSubjects)
	return SearchActionItemMatchesResult{
		Matches:                out,
		RequestedMode:          mode,
		EffectiveMode:          effectiveMode,
		FallbackReason:         fallbackReason,
		SemanticAvailable:      semanticReady,
		SemanticCandidateCount: len(semanticScores),
		EmbeddingSummary:       s.embeddingSummaryForProjects(ctx, projectIDs),
	}, nil
}

// filterReadySemanticMatches removes semantic candidates whose durable lifecycle state is not ready.
func (s *Service) filterReadySemanticMatches(ctx context.Context, projectIDs []string, rows []EmbeddingSearchMatch) ([]EmbeddingSearchMatch, error) {
	if len(projectIDs) == 0 || len(rows) == 0 {
		return []EmbeddingSearchMatch{}, nil
	}
	if s == nil {
		return rows, nil
	}
	if s.embeddingLifecycle == nil {
		return []EmbeddingSearchMatch{}, nil
	}
	subjectIDsByType := make(map[EmbeddingSubjectType][]string)
	for _, row := range rows {
		subjectType := row.SubjectType
		subjectID := strings.TrimSpace(row.SubjectID)
		if subjectType == "" || subjectID == "" {
			continue
		}
		subjectIDsByType[subjectType] = append(subjectIDsByType[subjectType], subjectID)
	}
	readyKeys := map[string]struct{}{}
	for subjectType, subjectIDs := range subjectIDsByType {
		lifecycleRows, err := s.embeddingLifecycle.ListEmbeddings(ctx, EmbeddingListFilter{
			ProjectIDs:  append([]string(nil), projectIDs...),
			SubjectType: subjectType,
			SubjectIDs:  subjectIDs,
			Statuses:    []EmbeddingLifecycleStatus{EmbeddingLifecycleReady},
			Limit:       len(subjectIDs),
		})
		if err != nil {
			return nil, err
		}
		for _, row := range lifecycleRows {
			readyKeys[embeddingRecordKey(row.SubjectType, row.SubjectID)] = struct{}{}
		}
	}
	readyRows := make([]EmbeddingSearchMatch, 0, len(rows))
	for _, row := range rows {
		if _, ok := readyKeys[embeddingRecordKey(row.SubjectType, row.SubjectID)]; !ok {
			continue
		}
		readyRows = append(readyRows, row)
	}
	return readyRows, nil
}

func (s *Service) annotateActionItemMatchesWithEmbeddingState(ctx context.Context, projectIDs []string, matches []ActionItemMatch, semanticSubjects map[string]EmbeddingSearchMatch) {
	if len(matches) == 0 {
		return
	}
	for idx := range matches {
		matches[idx].EmbeddingSubjectType = ""
		matches[idx].EmbeddingSubjectID = ""
		matches[idx].EmbeddingStatus = ""
	}
	if s.embeddingLifecycle == nil {
		return
	}
	subjectIDsByType := make(map[EmbeddingSubjectType][]string)
	for idx := range matches {
		selectedType := EmbeddingSubjectTypeWorkItem
		selectedID := matches[idx].ActionItem.ID
		if selected, ok := semanticSubjects[matches[idx].ActionItem.ID]; ok {
			selectedType = selected.SubjectType
			selectedID = selected.SubjectID
		}
		if strings.TrimSpace(selectedID) == "" || selectedType == "" {
			continue
		}
		matches[idx].EmbeddingSubjectType = selectedType
		matches[idx].EmbeddingSubjectID = selectedID
		subjectIDsByType[selectedType] = append(subjectIDsByType[selectedType], selectedID)
	}
	byID := map[string]EmbeddingRecord{}
	for subjectType, subjectIDs := range subjectIDsByType {
		rows, err := s.embeddingLifecycle.ListEmbeddings(ctx, EmbeddingListFilter{
			ProjectIDs:  append([]string(nil), projectIDs...),
			SubjectType: subjectType,
			SubjectIDs:  subjectIDs,
			Limit:       len(subjectIDs),
		})
		if err != nil {
			log.Warn("list embedding lifecycle rows for search annotation failed", "subject_type", subjectType, "err", err)
			return
		}
		for _, row := range rows {
			byID[embeddingRecordKey(row.SubjectType, row.SubjectID)] = row
		}
	}
	for idx := range matches {
		row, ok := byID[embeddingRecordKey(matches[idx].EmbeddingSubjectType, matches[idx].EmbeddingSubjectID)]
		if !ok {
			continue
		}
		matches[idx].EmbeddingStatus = row.Status
		matches[idx].EmbeddingStaleReason = row.StaleReason
		matches[idx].EmbeddingLastErrorSummary = row.LastErrorSummary
		if row.LastSucceededAt != nil {
			ts := *row.LastSucceededAt
			matches[idx].EmbeddingUpdatedAt = &ts
		}
	}
}

func (s *Service) embeddingSummaryForProjects(ctx context.Context, projectIDs []string) EmbeddingSummary {
	if len(projectIDs) == 0 {
		return EmbeddingSummary{
			ProjectIDs: append([]string(nil), projectIDs...),
		}
	}
	if s == nil || s.embeddingLifecycle == nil {
		return EmbeddingSummary{
			ProjectIDs: append([]string(nil), projectIDs...),
		}
	}
	summary, err := s.embeddingLifecycle.SummarizeEmbeddings(ctx, EmbeddingListFilter{
		ProjectIDs: append([]string(nil), projectIDs...),
	})
	if err != nil {
		log.Warn("summarize embedding lifecycle rows failed", "err", err)
		return EmbeddingSummary{
			ProjectIDs: append([]string(nil), projectIDs...),
		}
	}
	return summary
}

func embeddingRecordKey(subjectType EmbeddingSubjectType, subjectID string) string {
	return string(subjectType) + "\x00" + strings.TrimSpace(subjectID)
}

// normalizeSearchMode returns the supported mode or a default when omitted.
func normalizeSearchMode(raw SearchMode) (SearchMode, error) {
	mode := SearchMode(strings.TrimSpace(strings.ToLower(string(raw))))
	if mode == "" {
		return SearchModeHybrid, nil
	}
	switch mode {
	case SearchModeKeyword, SearchModeSemantic, SearchModeHybrid:
		return mode, nil
	default:
		return "", fmt.Errorf("invalid search mode %q: %w", raw, domain.ErrInvalidID)
	}
}

// normalizeSearchSort returns the supported sort order or a default when omitted.
func normalizeSearchSort(raw SearchSort) (SearchSort, error) {
	sortOrder := SearchSort(strings.TrimSpace(strings.ToLower(string(raw))))
	if sortOrder == "" {
		return SearchSortRankDesc, nil
	}
	switch sortOrder {
	case SearchSortRankDesc, SearchSortTitleAsc, SearchSortCreatedAtDesc, SearchSortUpdatedAtDesc:
		return sortOrder, nil
	default:
		return "", fmt.Errorf("invalid search sort %q: %w", raw, domain.ErrInvalidID)
	}
}

// normalizeSearchPagination returns validated pagination with defaults and upper bounds.
func normalizeSearchPagination(limit, offset int) (int, int, error) {
	if limit < 0 {
		return 0, 0, fmt.Errorf("search limit must be >= 0: %w", domain.ErrInvalidID)
	}
	if offset < 0 {
		return 0, 0, fmt.Errorf("search offset must be >= 0: %w", domain.ErrInvalidID)
	}
	if limit == 0 {
		limit = defaultSearchLimit
	}
	if limit > maxSearchLimit {
		limit = maxSearchLimit
	}
	return limit, offset, nil
}

// normalizeLowerFilterSet canonicalizes optional filter values into a lower-cased membership set.
func normalizeLowerFilterSet(values []string) map[string]struct{} {
	out := make(map[string]struct{}, len(values))
	for _, raw := range values {
		value := strings.TrimSpace(strings.ToLower(raw))
		if value == "" {
			continue
		}
		out[value] = struct{}{}
	}
	return out
}

// unsupportedSearchLevels returns sorted unsupported level values from a normalized level filter set.
func unsupportedSearchLevels(levelFilter map[string]struct{}) []string {
	out := make([]string, 0)
	for level := range levelFilter {
		if _, ok := supportedSearchLevelFilters[level]; ok {
			continue
		}
		out = append(out, level)
	}
	slices.Sort(out)
	return out
}

// actionItemMatchesExtendedSearchFilters applies optional level/kind/label filter constraints to one actionItem.
func actionItemMatchesExtendedSearchFilters(actionItem domain.ActionItem, levelFilter, kindFilter, labelsAnyFilter, labelsAllFilter map[string]struct{}) bool {
	if len(levelFilter) > 0 {
		if _, ok := levelFilter[strings.ToLower(strings.TrimSpace(string(actionItem.Scope)))]; !ok {
			return false
		}
	}
	if len(kindFilter) > 0 {
		if _, ok := kindFilter[strings.ToLower(strings.TrimSpace(string(actionItem.Kind)))]; !ok {
			return false
		}
	}
	if len(labelsAnyFilter) == 0 && len(labelsAllFilter) == 0 {
		return true
	}

	actionItemLabelSet := make(map[string]struct{}, len(actionItem.Labels))
	for _, raw := range actionItem.Labels {
		label := strings.TrimSpace(strings.ToLower(raw))
		if label == "" {
			continue
		}
		actionItemLabelSet[label] = struct{}{}
	}
	if len(labelsAnyFilter) > 0 {
		matchedAny := false
		for label := range labelsAnyFilter {
			if _, ok := actionItemLabelSet[label]; ok {
				matchedAny = true
				break
			}
		}
		if !matchedAny {
			return false
		}
	}
	for label := range labelsAllFilter {
		if _, ok := actionItemLabelSet[label]; !ok {
			return false
		}
	}
	return true
}

// compareActionItemMatchRankDesc keeps the legacy deterministic rank ordering for matches.
func compareActionItemMatchRankDesc(a, b ActionItemMatch) int {
	if a.Project.ID == b.Project.ID {
		if a.StateID == b.StateID {
			if a.ActionItem.ColumnID == b.ActionItem.ColumnID {
				if a.ActionItem.Position == b.ActionItem.Position {
					return strings.Compare(a.ActionItem.ID, b.ActionItem.ID)
				}
				return a.ActionItem.Position - b.ActionItem.Position
			}
			return strings.Compare(a.ActionItem.ColumnID, b.ActionItem.ColumnID)
		}
		return strings.Compare(a.StateID, b.StateID)
	}
	return strings.Compare(a.Project.ID, b.Project.ID)
}

// compareTimeDesc compares timestamps in descending order.
func compareTimeDesc(left, right time.Time) int {
	if left.Equal(right) {
		return 0
	}
	if left.After(right) {
		return -1
	}
	return 1
}

// compareFloat64Desc compares numeric values in descending order.
func compareFloat64Desc(left, right float64) int {
	if left == right {
		return 0
	}
	if left > right {
		return -1
	}
	return 1
}

// clamp01 constrains score values to the [0,1] range.
func clamp01(value float64) float64 {
	if value < 0 {
		return 0
	}
	if value > 1 {
		return 1
	}
	return value
}

// actionItemLexicalMatchScore calculates a normalized lexical score for one actionItem/query pair.
func actionItemLexicalMatchScore(actionItem domain.ActionItem, query string) float64 {
	query = strings.TrimSpace(strings.ToLower(query))
	if query == "" {
		return 0
	}
	score := 0.0
	score = max(score, fieldLexicalScore(actionItem.Title, query))
	score = max(score, fieldLexicalScore(actionItem.Description, query)*0.9)
	for _, label := range actionItem.Labels {
		score = max(score, fieldLexicalScore(label, query)*0.8)
	}
	score = max(score, fieldLexicalScore(actionItem.Metadata.Objective, query)*0.82)
	score = max(score, fieldLexicalScore(actionItem.Metadata.AcceptanceCriteria, query)*0.8)
	score = max(score, fieldLexicalScore(actionItem.Metadata.ValidationPlan, query)*0.78)
	score = max(score, fieldLexicalScore(actionItem.Metadata.BlockedReason, query)*0.76)
	score = max(score, fieldLexicalScore(actionItem.Metadata.RiskNotes, query)*0.76)
	return clamp01(score)
}

// fieldLexicalScore returns one lexical score using exact/prefix/contains/fuzzy matching tiers.
func fieldLexicalScore(candidate, query string) float64 {
	query = strings.TrimSpace(strings.ToLower(query))
	candidate = strings.TrimSpace(strings.ToLower(candidate))
	if query == "" || candidate == "" {
		return 0
	}
	switch {
	case candidate == query:
		return 1
	case strings.HasPrefix(candidate, query):
		return 0.95
	case strings.Contains(candidate, query):
		return 0.85
	case fuzzyContainsQuery(candidate, query):
		return 0.6
	default:
		return 0
	}
}

// fuzzyContainsQuery reports whether candidate matches query by exact/prefix/contains
// checks first, then by deterministic rune-order subsequence matching.
func fuzzyContainsQuery(candidate, query string) bool {
	query = strings.TrimSpace(strings.ToLower(query))
	candidate = strings.TrimSpace(strings.ToLower(candidate))
	if query == "" {
		return true
	}
	if candidate == "" {
		return false
	}
	if strings.Contains(candidate, query) {
		return true
	}

	qRunes := []rune(query)
	qi := 0
	// Fallback to subsequence matching so fuzzy queries work across gaps.
	for _, r := range []rune(candidate) {
		if r != qRunes[qi] {
			continue
		}
		qi++
		if qi == len(qRunes) {
			return true
		}
	}
	return false
}

// buildDependencyRollup computes aggregate dependency and blocked-state counts.
func buildDependencyRollup(projectID string, tasks []domain.ActionItem) domain.DependencyRollup {
	rollup := domain.DependencyRollup{
		ProjectID:  projectID,
		TotalItems: len(tasks),
	}
	stateByID := make(map[string]domain.LifecycleState, len(tasks))
	for _, actionItem := range tasks {
		stateByID[actionItem.ID] = actionItem.LifecycleState
	}
	for _, actionItem := range tasks {
		dependsOn := uniqueNonEmptyIDs(actionItem.Metadata.DependsOn)
		blockedBy := uniqueNonEmptyIDs(actionItem.Metadata.BlockedBy)

		if len(dependsOn) > 0 {
			rollup.ItemsWithDependencies++
			rollup.DependencyEdges += len(dependsOn)
		}
		if len(blockedBy) > 0 || strings.TrimSpace(actionItem.Metadata.BlockedReason) != "" {
			rollup.BlockedItems++
		}
		rollup.BlockedByEdges += len(blockedBy)

		// Dependencies are unresolved when the target is missing or not complete.
		for _, depID := range dependsOn {
			state, ok := stateByID[depID]
			if !ok || state != domain.StateComplete {
				rollup.UnresolvedDependencyEdges++
			}
		}
	}
	return rollup
}

// uniqueNonEmptyIDs trims and de-duplicates IDs while preserving order.
func uniqueNonEmptyIDs(in []string) []string {
	out := make([]string, 0, len(in))
	seen := map[string]struct{}{}
	for _, raw := range in {
		id := strings.TrimSpace(raw)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

// wouldCreateParentCycle reports whether assigning candidateParentID would create a cycle.
func wouldCreateParentCycle(actionItemID, candidateParentID string, tasks []domain.ActionItem) bool {
	actionItemID = strings.TrimSpace(actionItemID)
	candidateParentID = strings.TrimSpace(candidateParentID)
	if actionItemID == "" || candidateParentID == "" {
		return false
	}
	parentByID := make(map[string]string, len(tasks))
	for _, actionItem := range tasks {
		parentByID[actionItem.ID] = strings.TrimSpace(actionItem.ParentID)
	}
	current := candidateParentID
	visited := map[string]struct{}{}
	for current != "" {
		if current == actionItemID {
			return true
		}
		if _, ok := visited[current]; ok {
			return true
		}
		visited[current] = struct{}{}
		next, ok := parentByID[current]
		if !ok {
			return false
		}
		current = next
	}
	return false
}

// defaultStateTemplates returns default state templates.
func defaultStateTemplates() []StateTemplate {
	return []StateTemplate{
		{ID: "todo", Name: "To Do", WIPLimit: 0, Position: 0},
		{ID: "in_progress", Name: "In Progress", WIPLimit: 0, Position: 1},
		{ID: "complete", Name: "Complete", WIPLimit: 0, Position: 2},
		{ID: "failed", Name: "Failed", WIPLimit: 0, Position: 3, Hidden: true},
	}
}

// sanitizeStateTemplates handles sanitize state templates.
func sanitizeStateTemplates(in []StateTemplate) []StateTemplate {
	if len(in) == 0 {
		return nil
	}
	out := make([]StateTemplate, 0, len(in))
	seen := map[string]struct{}{}
	for idx, state := range in {
		state.Name = strings.TrimSpace(state.Name)
		state.ID = strings.TrimSpace(strings.ToLower(state.ID))
		if state.Name == "" {
			continue
		}
		if state.ID == "" {
			state.ID = normalizeStateID(state.Name)
		}
		dedupeID := strings.ReplaceAll(strings.ReplaceAll(state.ID, "-", ""), "_", "")
		if _, ok := seen[dedupeID]; ok {
			continue
		}
		seen[dedupeID] = struct{}{}
		if state.Position < 0 {
			state.Position = idx
		}
		if state.WIPLimit < 0 {
			state.WIPLimit = 0
		}
		out = append(out, state)
	}
	slices.SortFunc(out, func(a, b StateTemplate) int {
		if a.Position == b.Position {
			return strings.Compare(a.ID, b.ID)
		}
		return a.Position - b.Position
	})
	return out
}

// normalizeStateID normalizes a column display name into its canonical state-id slug.
// Strict-canonical: returns canonical state IDs (todo, in_progress, complete, failed,
// archived) when the input slug matches; otherwise returns the slugified form for
// non-state columns. Legacy aliases (done, completed, progress, doing, in-progress)
// are REJECTED with an empty-string return — callers test the empty passthrough
// as the unknown-state error path. Note: "to-do" remains a kebab-spelled canonical
// (matches "to_do" after slug → maps to "todo") and is NOT a legacy alias.
func normalizeStateID(name string) string {
	name = strings.TrimSpace(strings.ToLower(name))
	if name == "" {
		return ""
	}
	switch name {
	case "done", "completed", "progress", "doing", "in-progress":
		return ""
	}
	var b strings.Builder
	lastUnderscore := false
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
			lastUnderscore = false
		case r >= '0' && r <= '9':
			b.WriteRune(r)
			lastUnderscore = false
		default:
			if !lastUnderscore {
				b.WriteByte('_')
				lastUnderscore = true
			}
		}
	}
	normalized := strings.Trim(b.String(), "_")
	switch normalized {
	case "to_do", "todo":
		return "todo"
	case "in_progress":
		return "in_progress"
	case "complete":
		return "complete"
	case "failed":
		return "failed"
	case "archived":
		return "archived"
	default:
		return normalized
	}
}

// lifecycleStateForColumnID resolves canonical lifecycle state for a column.
func lifecycleStateForColumnID(columns []domain.Column, columnID string) domain.LifecycleState {
	for _, column := range columns {
		if column.ID != columnID {
			continue
		}
		switch normalizeStateID(column.Name) {
		case "todo":
			return domain.StateTodo
		case "in_progress":
			return domain.StateInProgress
		case "complete":
			return domain.StateComplete
		case "failed":
			return domain.StateFailed
		case "archived":
			return domain.StateArchived
		default:
			return domain.StateTodo
		}
	}
	return ""
}

// normalizeCommentTargetInput canonicalizes and validates comment target fields.
func normalizeCommentTargetInput(projectID string, targetType domain.CommentTargetType, targetID string) (domain.CommentTarget, error) {
	return domain.NormalizeCommentTarget(domain.CommentTarget{
		ProjectID:  projectID,
		TargetType: targetType,
		TargetID:   targetID,
	})
}

// normalizeActorTypeInput canonicalizes actor-type input and applies a default.
func normalizeActorTypeInput(actorType domain.ActorType) domain.ActorType {
	actorType = domain.ActorType(strings.TrimSpace(strings.ToLower(string(actorType))))
	if actorType == "" {
		return domain.ActorTypeUser
	}
	return actorType
}

// applyMutationActorToActionItem applies context-provided mutation actor metadata to a actionItem.
func applyMutationActorToActionItem(ctx context.Context, actionItem *domain.ActionItem) {
	if actionItem == nil {
		return
	}
	actor, ok := MutationActorFromContext(ctx)
	if !ok {
		return
	}
	if actorID := strings.TrimSpace(actor.ActorID); actorID != "" {
		actionItem.UpdatedByActor = actorID
	}
	if actorName := strings.TrimSpace(actor.ActorName); actorName != "" {
		actionItem.UpdatedByName = actorName
	} else if strings.TrimSpace(actionItem.UpdatedByName) == "" && strings.TrimSpace(actionItem.UpdatedByActor) != "" {
		actionItem.UpdatedByName = actionItem.UpdatedByActor
	}
	actionItem.UpdatedByType = normalizeActorTypeInput(actor.ActorType)
}

// withResolvedMutationActor merges explicit mutation-attribution input with context identity metadata.
func withResolvedMutationActor(ctx context.Context, actorID, actorName string, actorType domain.ActorType) (context.Context, MutationActor, bool) {
	resolved := MutationActor{
		ActorID:   strings.TrimSpace(actorID),
		ActorName: strings.TrimSpace(actorName),
		ActorType: domain.ActorType(strings.TrimSpace(strings.ToLower(string(actorType)))),
	}
	if ctxActor, ok := MutationActorFromContext(ctx); ok {
		if resolved.ActorID == "" {
			resolved.ActorID = ctxActor.ActorID
		}
		// Only borrow the context display name when the explicit identity is absent or matches.
		if resolved.ActorName == "" && (resolved.ActorID == "" || resolved.ActorID == ctxActor.ActorID) {
			resolved.ActorName = ctxActor.ActorName
		}
		if resolved.ActorType == "" {
			resolved.ActorType = normalizeActorTypeInput(ctxActor.ActorType)
		}
	}
	resolved = normalizeMutationActor(resolved)
	if resolved.ActorID == "" {
		return ctx, MutationActor{}, false
	}
	if resolved.ActorName == "" {
		resolved.ActorName = resolved.ActorID
	}
	return WithMutationActor(ctx, resolved), resolved, true
}

// firstNonEmptyTrimmed returns the first non-empty trimmed string.
func firstNonEmptyTrimmed(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

// createDefaultColumns creates default columns.
func (s *Service) createDefaultColumns(ctx context.Context, projectID string, now time.Time) error {
	for idx, state := range s.stateTemplates {
		position := state.Position
		if position < 0 {
			position = idx
		}
		column, err := domain.NewColumn(s.idGen(), projectID, state.Name, position, state.WIPLimit, now)
		if err != nil {
			return fmt.Errorf("create default column %q: %w", state.Name, err)
		}
		if state.Hidden {
			column.ArchivedAt = &now
		}
		if err := s.repo.CreateColumn(ctx, column); err != nil {
			return fmt.Errorf("persist default column %q: %w", state.Name, err)
		}
	}
	return nil
}
