package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
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
	// GitStatusChecker overrides the default `git status --porcelain` pre-
	// check used by Service.CreateActionItem (droplet 4b.6). Production
	// callers leave this nil so NewService wires defaultGitStatusChecker;
	// tests inject a stub directly via the struct field for deterministic,
	// process-isolated pre-check semantics.
	GitStatusChecker GitStatusChecker
	// BootstrapProjectHooks, when non-nil, is invoked after CreateProjectWithMetadata
	// successfully creates a project whose RepoPrimaryWorktree is non-empty + non-whitespace.
	// The function receives the worktree path; the production implementation (wired in
	// cmd/till and MCP boundaries) writes .claude/hooks/ + .claude/settings.json per the
	// Tillsyn bootstrap pipeline. Non-nil errors are LOGGED at warn level but do NOT
	// fail project creation -- Option C green-path semantics per agent-isolation cascade.
	BootstrapProjectHooks func(worktreePath string) error
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
	// gitStatusChecker is the pre-check seam called by CreateActionItem
	// when input.Paths is non-empty. Defaults to defaultGitStatusChecker;
	// tests overwrite this field directly (same package) to inject deterministic
	// fakes that never spawn `git`.
	gitStatusChecker GitStatusChecker
	// bootstrapProjectHooks mirrors ServiceConfig.BootstrapProjectHooks.
	// Nil means the bootstrap step is disabled (Option C: no-op when not wired).
	bootstrapProjectHooks func(worktreePath string) error
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
	gitChecker := cfg.GitStatusChecker
	if gitChecker == nil {
		gitChecker = defaultGitStatusChecker
	}

	return &Service{
		repo:                  repo,
		idGen:                 idGen,
		clock:                 clock,
		defaultDeleteMode:     cfg.DefaultDeleteMode,
		stateTemplates:        templates,
		autoProjectCols:       cfg.AutoCreateProjectColumns,
		autoSeedSteward:       cfg.AutoSeedStewardAnchors,
		defaultLeaseTTL:       cfg.CapabilityLeaseTTL,
		requireAgentLease:     requireAgentLease,
		authRequests:          cfg.AuthRequests,
		handoffRepo:           handoffRepo,
		schemaCache:           map[string]schemaCacheEntry{},
		embeddingGenerator:    cfg.EmbeddingGenerator,
		searchIndex:           searchIndex,
		embeddingLifecycle:    embeddingLifecycle,
		embeddingRuntime:      cfg.EmbeddingRuntime.Normalize(),
		searchLexicalW:        lexicalWeight,
		searchSemanticW:       semanticWeight,
		searchSemanticK:       semanticCandidates,
		authBackend:           cfg.AuthBackend,
		liveWait:              liveWait,
		gitStatusChecker:      gitChecker,
		bootstrapProjectHooks: cfg.BootstrapProjectHooks,
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
	project, err := domain.NewProjectFromInput(domain.ProjectInput{
		ID:          s.idGen(),
		Name:        "Inbox",
		Description: "Default project",
	}, now)
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
//
// HyllaArtifactRef / RepoBareRoot / RepoPrimaryWorktree / BuildTool /
// DevMcpServerName are the five Drop 4a L4 first-class project-node
// fields. They round-trip through Service.CreateProject →
// domain.NewProjectFromInput → repo.CreateProject. Empty strings are the
// meaningful zero value (project not yet bootstrapped) and round-trip
// untouched.
type CreateProjectInput struct {
	Name                string
	Description         string
	Kind                domain.KindID
	Metadata            domain.ProjectMetadata
	HyllaArtifactRef    string
	RepoBareRoot        string
	RepoPrimaryWorktree string
	BuildTool           string
	DevMcpServerName    string
	UpdatedBy           string
	UpdatedByName       string
	UpdatedType         domain.ActorType
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
	project, err := domain.NewProjectFromInput(domain.ProjectInput{
		ID:                  s.idGen(),
		Name:                in.Name,
		Description:         in.Description,
		HyllaArtifactRef:    in.HyllaArtifactRef,
		RepoBareRoot:        in.RepoBareRoot,
		RepoPrimaryWorktree: in.RepoPrimaryWorktree,
		BuildTool:           in.BuildTool,
		DevMcpServerName:    in.DevMcpServerName,
	}, now)
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
	if err := project.UpdateDetails(
		project.Name,
		project.Description,
		project.HyllaArtifactRef,
		project.RepoBareRoot,
		project.RepoPrimaryWorktree,
		project.BuildTool,
		project.DevMcpServerName,
		mergedMetadata,
		now,
	); err != nil {
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
	if s.bootstrapProjectHooks != nil && strings.TrimSpace(project.RepoPrimaryWorktree) != "" {
		if bootstrapErr := s.bootstrapProjectHooks(project.RepoPrimaryWorktree); bootstrapErr != nil {
			// Option C green-path: log + continue. Bootstrap failures must not fail project creation.
			log.Warn("project bootstrap hooks failed",
				"project_id", project.ID,
				"worktree", project.RepoPrimaryWorktree,
				"err", bootstrapErr)
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
//   - the embedded internal/templates/builtin/till-go.toml or
//     till-gen.toml fallback (Go builtin rebadged from `default-go.toml`
//     to `till-go.toml` in Drop 4c.6 W5.D1; language-agnostic builtin
//     rebadged from `default-generic.toml` to `till-gen.toml` in Drop
//     4c.6 W5.D2 — alongside the F.2.1 + F.2.2 dual-history; selected
//     per project Language axis).
//
// RELEASE NOTE — Drop 4c.5 droplets F.1.1 + F.1.2 BEHAVIOR CHANGE: prior
// to F.1.1 this helper was a no-op for every project (loadProjectTemplate
// returned ok=false unconditionally per the Drop 3.14 deferral), so
// KindCatalogJSON stayed empty and resolveActionItemKindDefinition routed
// through the legacy repo fallback. Post-F.1.1 + F.1.2, EVERY project
// receives a non-empty catalog at create time: F.1.2's filesystem walk
// inspects (1) <RepoBareRoot>/.tillsyn/template.toml then (2)
// <RepoPrimaryWorktree>/.tillsyn/template.toml, and falls through to the
// embedded language-default (F.1.1 + F.1.3) when neither candidate
// matches. This is safe because downstream callers
// (initializeProjectAllowedKinds in CreateProjectWithMetadata, the
// kind-catalog-aware resolveActionItemKindDefinition path, the
// dispatcher's spawn-command builder) all already handle non-empty
// catalogs since Drop 3.14. Pre-MVP no project relied on the
// empty-catalog branch; the only escape hatch today is authoring a
// minimal `<repo_root>/.tillsyn/template.toml` to override the embedded
// default at create time.
//
// The helper accepts a *domain.Project so 3.14 can substitute a real
// implementation without touching the call site in CreateProjectWithMetadata.
// It returns error so future template-load failures can propagate without
// another signature break.
//
// Per Drop 3 finding 5.B.14: edits to <project_root>/.tillsyn/template.toml
// AFTER project creation are ignored — the catalog is the create-time
// snapshot. Re-baking on every project lookup is explicitly out of scope.
//
// Drop 4c.6.1 W1.D2: when project.Metadata.Groups is non-empty,
// bakeProjectKindCatalog routes through bakeProjectKindCatalogWithHome to
// loadProjectTemplatesForGroups, which walks the HOME tier per group and
// aggregates results via mergeTemplates. Single-group projects (Groups nil or
// empty) continue through loadProjectTemplate unchanged.
func bakeProjectKindCatalog(project *domain.Project) error {
	if project == nil {
		return nil
	}
	homeDir := ""
	if h, err := os.UserHomeDir(); err == nil && strings.TrimSpace(h) != "" {
		homeDir = h
	}
	return bakeProjectKindCatalogWithHome(project, homeDir)
}

// bakeProjectKindCatalogWithHome is the testability seam for
// bakeProjectKindCatalog. bakeProjectKindCatalog calls it with the real
// os.UserHomeDir() result; tests call it with a t.TempDir() fake homeDir so
// the real $HOME is never consulted. This mirrors the loadProjectTemplateWithHome
// seam introduced by Drop 4c.6.1 W1.D1.
//
// When project.Metadata.Groups is non-empty, the multi-group coordinator
// loadProjectTemplatesForGroups walks the HOME tier for each group and merges
// the resulting templates. When Groups is nil or empty, the existing
// single-group path via loadProjectTemplate is used unchanged.
func bakeProjectKindCatalogWithHome(project *domain.Project, homeDir string) error {
	if project == nil {
		return nil
	}
	var (
		tpl templates.Template
		ok  bool
		err error
	)
	if len(project.Metadata.Groups) > 0 {
		tpl, ok, err = loadProjectTemplatesForGroups(project, homeDir)
	} else {
		tpl, ok, err = loadProjectTemplate(project)
	}
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

// loadProjectTemplatesForGroups is the multi-group coordinator for
// bakeProjectKindCatalogWithHome. It iterates project.Metadata.Groups, calls
// loadProjectTemplateWithHome per non-empty group, and merges all resulting
// templates via mergeTemplates (last-group-wins on key collisions for map
// fields, append for slice fields).
//
// Empty-group guard: entries where strings.TrimSpace(group) == "" are skipped
// silently. A bare empty string would produce a malformed HOME tier path
// (~/.tillsyn/templates/.toml). Whitespace-only entries are also skipped.
//
// The homeDir parameter is the testability injection point. The caller
// (bakeProjectKindCatalogWithHome) passes the real os.UserHomeDir() result;
// tests pass t.TempDir() to avoid touching the real $HOME.
//
// Returns (zero, false, err) if any non-empty group's template load fails.
// Returns (zero, false, nil) when all groups are empty/whitespace-only or
// when all non-empty groups have no on-disk template at any tier — no embedded
// fallback fires. Templates are project-tier opt-in only per REFINEMENTS.md
// 2026-05-14. Returns (merged, true, nil) when at least one group resolved
// an on-disk template.
func loadProjectTemplatesForGroups(project *domain.Project, homeDir string) (templates.Template, bool, error) {
	merged := templates.Template{}
	hasMerged := false
	for _, group := range project.Metadata.Groups {
		if strings.TrimSpace(group) == "" {
			continue
		}
		tpl, ok, err := loadProjectTemplateWithHome(project, homeDir, group)
		if err != nil {
			return templates.Template{}, false, err
		}
		if !ok {
			// No on-disk template for this group; skip — do not merge zero value.
			continue
		}
		if !hasMerged {
			merged = tpl
			hasMerged = true
		} else {
			merged = mergeTemplates(merged, tpl)
		}
	}
	if !hasMerged {
		// All groups were empty/whitespace-only, or all non-empty groups had no
		// on-disk template. Return (zero, false, nil) — no embedded fallback.
		return templates.Template{}, false, nil
	}
	return merged, true, nil
}

// mergeTemplates merges overlay on top of base and returns the combined
// Template. The merge strategy is per-field as documented below (Drop
// 4c.6.1 W1.D2 acceptance criterion #4):
//
//   - SchemaVersion: last-group-wins (overlay overwrites base when non-empty).
//   - Kinds: per-key last-group-wins (overlay key replaces base key on collision).
//   - ChildRules: append base + overlay; dedup on (WhenParentKind,
//     CreateChildKind) tuple, overlay entry wins on collision.
//   - AgentBindings: per-key last-group-wins (primary multi-group use case).
//   - Agents: per-key last-group-wins.
//   - Gates: per-key last-group-wins (overlay slice replaces base slice for
//     same kind key; NOT concat). Gate ordering within a kind is the overlay
//     author's responsibility.
//   - GateRulesRaw: per-key shallow merge, last-group-wins on collision.
//   - Tillsyn: whole-struct last-group-wins; overlay Tillsyn replaces base if
//     overlay is non-zero (MaxContextBundleChars != 0 ||
//     MaxAggregatorDuration != 0 || SpawnTempRoot != ""). RequiresPlugins
//     does NOT contribute to the non-zero check per the spec-enumerated
//     condition; the three named fields are the semantically meaningful
//     non-zero signals.
//   - StewardSeeds: append base + overlay (no dedup; seeds are
//     project-unique and append order is significant).
//
// Refinement MERGE-FIELD-AXIS-R1: revisit per-field semantics for Tillsyn,
// StewardSeeds, Gates, GateRulesRaw, ChildRules, Kinds, Agents when
// multi-group projects start exercising these fields in dogfood. Pre-MVP,
// last-group-wins / append is the pragmatic default.
func mergeTemplates(base, overlay templates.Template) templates.Template {
	out := base

	// SchemaVersion: last-group-wins.
	if overlay.SchemaVersion != "" {
		out.SchemaVersion = overlay.SchemaVersion
	}

	// Kinds: per-key last-group-wins.
	if len(overlay.Kinds) > 0 {
		if out.Kinds == nil {
			out.Kinds = make(map[domain.Kind]templates.KindRule, len(overlay.Kinds))
		}
		for k, v := range overlay.Kinds {
			out.Kinds[k] = v
		}
	}

	// ChildRules: append base + overlay with dedup on
	// (WhenParentKind, CreateChildKind) tuple; overlay wins on collision.
	if len(overlay.ChildRules) > 0 {
		type childRuleKey struct {
			WhenParentKind  domain.Kind
			CreateChildKind domain.Kind
		}
		idx := make(map[childRuleKey]int, len(out.ChildRules))
		for i, r := range out.ChildRules {
			idx[childRuleKey{r.WhenParentKind, r.CreateChildKind}] = i
		}
		for _, r := range overlay.ChildRules {
			key := childRuleKey{r.WhenParentKind, r.CreateChildKind}
			if i, exists := idx[key]; exists {
				out.ChildRules[i] = r
			} else {
				out.ChildRules = append(out.ChildRules, r)
				idx[key] = len(out.ChildRules) - 1
			}
		}
	}

	// AgentBindings: per-key last-group-wins.
	if len(overlay.AgentBindings) > 0 {
		if out.AgentBindings == nil {
			out.AgentBindings = make(map[domain.Kind]templates.AgentBinding, len(overlay.AgentBindings))
		}
		for k, v := range overlay.AgentBindings {
			out.AgentBindings[k] = v
		}
	}

	// Agents: per-key last-group-wins.
	if len(overlay.Agents) > 0 {
		if out.Agents == nil {
			out.Agents = make(map[domain.Kind]templates.AgentRuntime, len(overlay.Agents))
		}
		for k, v := range overlay.Agents {
			out.Agents[k] = v
		}
	}

	// Gates: per-key last-group-wins (slice replaces, NOT concat).
	if len(overlay.Gates) > 0 {
		if out.Gates == nil {
			out.Gates = make(map[domain.Kind][]templates.GateKind, len(overlay.Gates))
		}
		for k, v := range overlay.Gates {
			out.Gates[k] = v
		}
	}

	// GateRulesRaw: per-key shallow merge, last-group-wins on collision.
	if len(overlay.GateRulesRaw) > 0 {
		if out.GateRulesRaw == nil {
			out.GateRulesRaw = make(map[string]any, len(overlay.GateRulesRaw))
		}
		for k, v := range overlay.GateRulesRaw {
			out.GateRulesRaw[k] = v
		}
	}

	// Tillsyn: whole-struct last-group-wins if overlay is non-zero.
	if overlay.Tillsyn.MaxContextBundleChars != 0 ||
		overlay.Tillsyn.MaxAggregatorDuration != 0 ||
		overlay.Tillsyn.SpawnTempRoot != "" {
		out.Tillsyn = overlay.Tillsyn
	}

	// StewardSeeds: append base + overlay (no dedup).
	if len(overlay.StewardSeeds) > 0 {
		out.StewardSeeds = append(out.StewardSeeds, overlay.StewardSeeds...)
	}

	return out
}

// projectTemplateFilename is the canonical filename loadProjectTemplate
// looks for under the project's bare-root and primary-worktree
// .tillsyn/ directories. Hard-coded here rather than exported because
// adopters do not configure this name today; renaming requires a deliberate
// drop touching every consumer (F.3.3's `set` op writes to the same name).
const projectTemplateFilename = "template.toml"

// projectTemplateDir is the canonical sub-directory under each candidate
// repo root that loadProjectTemplate walks. Same hard-coded rationale as
// projectTemplateFilename.
const projectTemplateDir = ".tillsyn"

// loadProjectTemplate resolves the Template that CreateProjectWithMetadata
// bakes into a project's KindCatalog.
//
// Drop 4c.5 droplet F.1.2 extends F.1.1's empty-path embedded fallback
// with the on-disk filesystem walk Drop 3.14 had deferred. Drop 4c.6.1
// droplet W1.D1 inserts the user HOME tier between the project-worktree
// tier and the embedded fallback. Resolution order, in priority sequence:
//
//  1. <project.RepoBareRoot>/.tillsyn/template.toml — when RepoBareRoot
//     (after whitespace trim) is non-empty AND the file exists. The bare
//     root is the orchestration root in a bare-repo + worktree layout
//     (e.g. /Users/.../hylla/tillsyn/ for the tillsyn dogfood project)
//     and is checked first because it survives worktree recreation.
//  2. <project.RepoPrimaryWorktree>/.tillsyn/template.toml — when
//     RepoPrimaryWorktree (after whitespace trim) is non-empty AND the
//     file exists. Adopters that do not use a bare-root layout author
//     their template here.
//  3. <$HOME>/.tillsyn/templates/<group>.toml — where group is drawn from
//     project.Metadata.Groups (see loadProjectTemplatesForGroups).
//     Skipped when os.UserHomeDir() fails or when the group list is empty.
//     Allows users to override templates for all projects that share a
//     group without per-project template files (W1.D1).
//
// First-candidate-wins on success: as soon as a candidate file exists
// AND templates.Load returns nil error, the function returns. Subsequent
// candidates are NOT consulted. This is deliberate — once the dev has
// authored a template at any candidate path, that authored content is
// authoritative.
//
// Error propagation on candidate-load failure (F.1.2 spec falsification
// mitigation #2): if a candidate file EXISTS but templates.Load returns
// an error (typo, malformed TOML, schema-version mismatch, validator
// rejection), the error PROPAGATES wrapped with the offending path —
// the function does NOT fall through to the next candidate. Silent
// fall-through would hide typos in dev-authored templates and let
// apparently-correct project creates run against unintended embedded
// defaults. The wrapping format is `template at <abs-path>: <wrapped>`
// so callers retain `errors.Is(err, ErrUnknownTemplateKey)` /
// `errors.Is(err, templates.ErrUnknownTemplateKey)` etc. against the
// templates package sentinels AND see the offending path.
//
// File-not-exist vs other open errors: only fs.ErrNotExist on os.Open
// triggers fallthrough to the next candidate. Permission-denied or any
// other I/O failure propagates as a wrapped error so the dev sees the
// actual root cause rather than silently inheriting the embedded
// default. (TOCTOU note: by routing through os.Open + ErrNotExist
// rather than os.Stat-then-Open, we avoid the race window where a
// candidate file is removed between Stat and Open.)
//
// Relative-path safety (F.1.2 spec falsification mitigation #1): empty
// RepoBareRoot and empty RepoPrimaryWorktree skip their respective
// candidate lookups outright. Without the early-return, filepath.Join("",
// ".tillsyn", "template.toml") would produce the relative path
// `.tillsyn/template.toml`, which os.Open would then resolve against
// the process's current working directory — leaking CWD-dependent
// behavior into project create. The early-empty-skip is the canonical
// guard.
//
// Symlink policy (F.1.2 spec falsification mitigation #3): os.Open
// follows symlinks transparently. A `.tillsyn/template.toml` that is a
// symlink to an authored TOML elsewhere is honored. Aggressive symlink
// hardening (e.g. rejecting links that escape the repo root) is
// deferred to a future refinement; pre-MVP, the dev controls both ends
// of the chain.
//
// Project nil-guard: bakeProjectKindCatalog already nil-checks before
// calling, but this function nil-guards too so an accidental direct
// caller doesn't deref. Returns (zero, false, nil) on nil project to
// match the prior "skip" behavior.
//
// Note: droplet 3.20's STEWARD-seed auto-generator does NOT depend on
// this helper — it loads the embedded default template independently
// via the package-level `loadStewardSeedTemplate` seam (see
// auto_generate_steward.go) so seed materialization is decoupled from
// the KindCatalog-bake fallback semantics. Post-Phase-4.2 the seam is
// called with `""` (generic template) until Phase 4.4 wires per-project
// STEWARD seed migration.
//
// Per Drop 3 finding 5.B.14: edits to the on-disk template AFTER project
// creation are ignored — the catalog is the create-time snapshot.
// Re-baking on every lookup is explicitly out of scope.
func loadProjectTemplate(project *domain.Project) (templates.Template, bool, error) {
	if project == nil {
		return templates.Template{}, false, nil
	}
	homeDir := ""
	if h, err := os.UserHomeDir(); err == nil && strings.TrimSpace(h) != "" {
		homeDir = h
	}
	return loadProjectTemplateWithHome(project, homeDir, "")
}

// loadProjectTemplateWithHome is the testability seam for the 3-tier template
// resolution walk. loadProjectTemplate calls it with the real os.UserHomeDir()
// result and a group string (empty string skips the HOME tier). D2's
// multi-group coordinator calls it per element in project.Metadata.Groups
// with the same homeDir.
//
// Resolution order:
//
//  1. <project.RepoBareRoot>/.tillsyn/template.toml
//  2. <project.RepoPrimaryWorktree>/.tillsyn/template.toml
//  3. <homeDir>/.tillsyn/templates/<group>.toml (HOME tier — new in D1)
//
// When all on-disk candidates are absent (or no candidates exist because repo
// paths are empty and the HOME tier was skipped), returns (zero, false, nil).
// The caller (bakeProjectKindCatalogWithHome) handles ok=false by leaving
// KindCatalogJSON empty — templates are project-tier opt-in only per
// REFINEMENTS.md 2026-05-14.
//
// HOME tier is skipped when homeDir is empty (os.UserHomeDir() failed or
// returned whitespace) or when group is empty (no meaningful group name →
// malformed path avoided). This mirrors the readUserTierAgent pattern in
// render.go.
//
// First-candidate-wins and error-propagation semantics are identical to the
// pre-D1 walk: a candidate file that EXISTS but fails templates.Load
// propagates an error without falling through to subsequent candidates.
func loadProjectTemplateWithHome(project *domain.Project, homeDir, group string) (templates.Template, bool, error) {
	if project == nil {
		return templates.Template{}, false, nil
	}
	bareRoot := strings.TrimSpace(project.RepoBareRoot)
	primaryWorktree := strings.TrimSpace(project.RepoPrimaryWorktree)
	// Build the candidate list in priority order. Empty paths are
	// dropped here so the walk loop never feeds filepath.Join an empty
	// root (which would silently produce a CWD-relative path). See the
	// "Relative-path safety" doc-comment paragraph on loadProjectTemplate.
	candidates := make([]string, 0, 3)
	if bareRoot != "" {
		candidates = append(candidates, filepath.Join(bareRoot, projectTemplateDir, projectTemplateFilename))
	}
	if primaryWorktree != "" {
		candidates = append(candidates, filepath.Join(primaryWorktree, projectTemplateDir, projectTemplateFilename))
	}
	if homeDir != "" && group != "" {
		candidates = append(candidates, filepath.Join(homeDir, ".tillsyn", "templates", group+".toml"))
	}
	for _, candidatePath := range candidates {
		tpl, ok, err := loadProjectTemplateCandidate(candidatePath)
		if err != nil {
			return templates.Template{}, false, err
		}
		if ok {
			return tpl, true, nil
		}
	}
	// All on-disk candidates absent (or no candidates at all because both
	// repo-path fields are empty and HOME tier was skipped). Return
	// (zero, false, nil) — no embedded fallback. Templates are project-tier
	// opt-in only per REFINEMENTS.md 2026-05-14.
	return templates.Template{}, false, nil
}

// loadProjectTierTemplateOnly resolves a project's template via the
// project-tier candidates ONLY — no HOME tier, no embedded language-default
// fallback. Returns (zero, false, nil) when neither
// <RepoBareRoot>/.tillsyn/template.toml nor
// <RepoPrimaryWorktree>/.tillsyn/template.toml exists.
//
// This is the project-tier-opt-in contract documented in REFINEMENTS.md
// 2026-05-14 "Remove project.Language; templates are project-tier opt-in
// only": child_rules auto-spawn at action_item create time fires only when a
// project has authored its own template, never via the embedded fallback. A
// project with no project-tier template gets no auto-create — pure tracking.
//
// Errors from a present-but-malformed candidate file (typo, schema mismatch,
// validator rejection) propagate wrapped with the offending path; the
// function does NOT silently fall through to a different candidate on
// non-not-exist open failures.
//
// Project nil-guard mirrors loadProjectTemplate.
func loadProjectTierTemplateOnly(project *domain.Project) (templates.Template, bool, error) {
	if project == nil {
		return templates.Template{}, false, nil
	}
	bareRoot := strings.TrimSpace(project.RepoBareRoot)
	primaryWorktree := strings.TrimSpace(project.RepoPrimaryWorktree)
	candidates := make([]string, 0, 2)
	if bareRoot != "" {
		candidates = append(candidates, filepath.Join(bareRoot, projectTemplateDir, projectTemplateFilename))
	}
	if primaryWorktree != "" {
		candidates = append(candidates, filepath.Join(primaryWorktree, projectTemplateDir, projectTemplateFilename))
	}
	for _, candidatePath := range candidates {
		tpl, ok, err := loadProjectTemplateCandidate(candidatePath)
		if err != nil {
			return templates.Template{}, false, err
		}
		if ok {
			return tpl, true, nil
		}
	}
	return templates.Template{}, false, nil
}

// loadProjectTemplateCandidate opens and parses a single on-disk template
// candidate at the given absolute path. Return contract:
//
//   - (tpl, true, nil): file exists and templates.Load succeeded.
//   - (zero, false, nil): file does not exist (fs.ErrNotExist on Open).
//     Caller continues the candidate walk.
//   - (zero, false, err): file exists but Load failed, OR Open failed
//     for a reason other than not-exist (permission denied, I/O error).
//     Error is wrapped with the offending path so callers retain
//     errors.Is routing against templates package sentinels while seeing
//     the source location. Caller MUST propagate without falling through
//     to subsequent candidates.
//
// Helper extraction keeps loadProjectTemplate's walk loop a clean
// sequence of "(skip / win / fail)" without an inline open+defer
// dance per iteration.
func loadProjectTemplateCandidate(candidatePath string) (templates.Template, bool, error) {
	file, err := os.Open(candidatePath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			// File-not-exist is the canonical "skip and try next
			// candidate" signal. Other open errors (permission,
			// I/O) propagate so the dev sees the actual cause.
			return templates.Template{}, false, nil
		}
		return templates.Template{}, false, fmt.Errorf("template at %s: %w", candidatePath, err)
	}
	defer file.Close()
	tpl, err := templates.Load(file)
	if err != nil {
		return templates.Template{}, false, fmt.Errorf("template at %s: %w", candidatePath, err)
	}
	return tpl, true, nil
}

// UpdateProjectInput holds input values for update project operations.
//
// Five Drop 4a L4 first-class fields ride alongside Name / Description.
// Per WAVE_1_PLAN.md §1.8 the Project surface is admin-driven, so the
// fields are value-typed (no pointer-sentinels). Callers that want to
// preserve existing values must read the project first and pass them
// through unchanged.
type UpdateProjectInput struct {
	ProjectID           string
	Name                string
	Description         string
	Kind                domain.KindID
	Metadata            domain.ProjectMetadata
	HyllaArtifactRef    string
	RepoBareRoot        string
	RepoPrimaryWorktree string
	BuildTool           string
	DevMcpServerName    string
	UpdatedBy           string
	UpdatedByName       string
	UpdatedType         domain.ActorType
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
	if err := project.UpdateDetails(
		in.Name,
		in.Description,
		in.HyllaArtifactRef,
		in.RepoBareRoot,
		in.RepoPrimaryWorktree,
		in.BuildTool,
		in.DevMcpServerName,
		in.Metadata,
		s.clock(),
	); err != nil {
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
	Packages []string
	// Files optionally enumerates the action item's reference-material file
	// paths (forward-slash, repo-root-relative). Empty slice IS the
	// meaningful zero value (no reference files attached) — no pointer-
	// sentinel needed at the create boundary. domain.NewActionItem trims +
	// dedupes; whitespace-only / backslash-bearing entries reject with
	// ErrInvalidFiles. Disjoint-axis with Paths — Files (read attention)
	// and Paths (write intent / lock scope) may legitimately overlap, so
	// no cross-axis check is performed. Domain primitive per Drop 4a L3 /
	// WAVE_1_PLAN.md §1.3.
	Files []string
	// StartCommit optionally seeds the action-item start-commit hash at
	// creation time (free-form trimmed string; empty IS the meaningful
	// zero value "not yet captured"). No pointer-sentinel needed at the
	// create boundary — an absent StartCommit at creation is the dominant
	// case (caller hasn't run `git rev-parse HEAD` yet). Domain trims
	// surrounding whitespace; no format check applies. Threaded through
	// app.CreateActionItemInput → domain.NewActionItem in droplet 4a.8.
	// Opaque-domain field — domain layer never calls git itself. Domain
	// primitive per Drop 4a L3 / WAVE_1_PLAN.md §1.4.
	StartCommit string
	// EndCommit optionally seeds the action-item end-commit hash at
	// creation time (free-form trimmed string; empty IS the meaningful
	// zero value "not yet captured"). Mirrors StartCommit's value-type
	// shape at the create boundary — no pointer-sentinel needed because
	// absent-at-creation is the dominant case (terminal capture happens
	// later via UpdateActionItem before MoveActionItemState). Domain
	// trims surrounding whitespace; no format check applies. Threaded
	// through app.CreateActionItemInput → domain.NewActionItem in droplet
	// 4a.9. Opaque-domain field — domain layer never calls git itself.
	// Empty is valid until terminal state; domain does NOT enforce non-
	// empty-on-terminal. Domain primitive per Drop 4a L3 /
	// WAVE_1_PLAN.md §1.5.
	EndCommit      string
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
//
// PATCH semantics (Drop 4c.5 droplet A.1): Title / Description / Priority /
// DueAt / Labels each use pointer-sentinels — nil preserves the existing
// stored value (no-op), non-nil applies the dereferenced value. An empty
// dereferenced string / slice clears the stored field (caller's explicit
// intent), with the sole exception of Title where empty still surfaces
// ErrInvalidTitle (title is required by domain.UpdateDetails). Pre-A.1
// behavior unconditionally wrote every field, so an agent issuing a
// "description-only" partial update would silently clobber Title /
// Priority / DueAt / Labels with their value-typed zero values; the
// pointer-sentinel pattern (already used by Owner / DropNumber /
// Persistent / DevGated / Paths / Packages / Files / StartCommit /
// EndCommit) extends the same shape to the original five fields.
type UpdateActionItemInput struct {
	ActionItemID string
	// Title optionally updates the action-item Title. nil preserves the
	// existing value (no-op); non-nil applies the dereferenced string after
	// trim. An empty dereferenced string still triggers ErrInvalidTitle via
	// domain.UpdateDetails — title is required.
	Title *string
	// Description optionally updates the action-item Description. nil
	// preserves the existing value (no-op); non-nil applies the
	// dereferenced string after trim. An empty dereferenced string clears
	// the stored description (caller's explicit intent).
	Description *string
	// Priority optionally updates the action-item Priority. nil preserves
	// the existing value (no-op); non-nil applies the dereferenced enum
	// after lowercase normalization. Pointer-sentinel replaces the
	// pre-A.1 empty-string defaulting block.
	Priority *domain.Priority
	// DueAt optionally updates the action-item DueAt. nil (outer pointer)
	// preserves the existing value (no-op); non-nil applies the
	// dereferenced *time.Time. A non-nil outer pointer holding a nil inner
	// pointer (or zero time) clears DueAt (caller's explicit intent).
	// Double-pointer is required because *time.Time itself doubles as a
	// presence sentinel inside the domain entity, so the partial-update
	// shape needs a second level of indirection to distinguish "missing
	// from input" from "explicitly cleared".
	DueAt **time.Time
	// Labels optionally updates the action-item Labels. nil preserves the
	// existing slice (no-op); non-nil applies the dereferenced slice. An
	// empty dereferenced slice clears all labels (caller's explicit intent,
	// mirroring Paths / Packages).
	Labels *[]string
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
	Packages *[]string
	// Files optionally updates the action-item Files slice. nil preserves
	// the existing value (no-op); non-nil replaces it. Same pointer-
	// sentinel rationale as Paths/Packages above — a description-only
	// update must NOT silently clobber a planner-set Files declaration.
	// Service applies via domain.NormalizeActionItemFiles so the create-
	// time trim/dedupe / forward-slash-check rules apply equally on
	// update. Disjoint-axis: no coverage / overlap check against Paths —
	// Files and Paths are independent (read attention vs write intent).
	// Domain primitive per Drop 4a L3 / WAVE_1_PLAN.md §1.3.
	Files *[]string
	// StartCommit optionally updates the action-item StartCommit string.
	// Pointer-sentinel: nil preserves the existing value (no-op); non-nil
	// applies the dereferenced string (empty dereferenced string clears
	// the prior commit hash — explicit caller intent). Pointer shape
	// matters because a description-only update by an agent must NOT
	// silently clobber a dispatcher-set start commit. Service applies
	// inline `strings.TrimSpace` so the create-time trim rule applies
	// equally on update (no domain helper exposed — single-line trim is
	// too thin to warrant a wrapper, matching Owner's inline-trim
	// precedent at the create site). Domain primitive per Drop 4a L3 /
	// WAVE_1_PLAN.md §1.4. Pointer-sentinel locked per WAVE_1_PLAN
	// post-4a.5 amendment.
	StartCommit *string
	// EndCommit optionally updates the action-item EndCommit string.
	// Pointer-sentinel mirrors StartCommit: nil preserves the existing
	// value (no-op); non-nil applies the dereferenced string (empty
	// dereferenced string clears the prior commit hash — explicit caller
	// intent, e.g. dispatcher rolling back a retry). Pointer shape
	// matters because a description-only update by an agent must NOT
	// silently clobber a dispatcher-set end commit. Wave 2 dispatcher
	// populates this via UpdateActionItem BEFORE MoveActionItemState so
	// the terminal capture lands cleanly. Service applies inline
	// `strings.TrimSpace` so the create-time trim rule applies equally on
	// update. No domain helper exposed (single-line trim too thin to
	// warrant a wrapper). Domain primitive per Drop 4a L3 /
	// WAVE_1_PLAN.md §1.5. Pointer-sentinel locked per WAVE_1_PLAN
	// post-4a.5 amendment.
	EndCommit     *string
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
	// Auto-treat parent_id == project_id as top-level: a project UUID is not
	// an action-item UUID. Callers who think "project IS the parent for top-level
	// items" pass project_id here; clearing it produces the correct outcome
	// (top-level item) without a confusing not_found error.
	if strings.TrimSpace(in.ParentID) == strings.TrimSpace(in.ProjectID) {
		in.ParentID = ""
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
	// Per Drop 4c.5 droplet F.6.1: the legacy
	// mergeActionItemMetadataWithKindTemplate stub (a no-op pass-through left
	// over from the Drop 3 droplet 3.15 KindTemplate-surface deletion) was
	// inlined here. Future template-driven action-item metadata defaults will
	// be reintroduced through a different mechanism if the need arises.
	mergedMetadata := in.Metadata
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
		Files:          in.Files,
		StartCommit:    in.StartCommit,
		EndCommit:      in.EndCommit,
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

	// Droplet 4b.6 — git-status pre-check. When the constructed action item
	// declares write-scope paths, verify each one is clean in the project's
	// primary worktree BEFORE the repo write so a dirty-tree reject never
	// mutates state. Runs on the post-domain-validation Paths slice so
	// malformed inputs (empty / whitespace / backslash entries) reject with
	// domain.ErrInvalidPaths above before any git subprocess fires. The
	// check is skipped on empty Paths (degenerate input) and on projects
	// with empty RepoPrimaryWorktree (legacy / unbootstrapped projects per
	// droplet 4b.6 acceptance criterion 2). Always-on per REVISION_BRIEF
	// L4: bypass requires the post-MVP supersede CLI.
	if len(actionItem.Paths) > 0 {
		if err := s.runGitStatusPreCheck(ctx, actionItem.ProjectID, actionItem.Paths); err != nil {
			return domain.ActionItem{}, err
		}
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
	if err := s.applyChildRulesForCreate(ctx, actionItem); err != nil {
		return domain.ActionItem{}, fmt.Errorf("apply template child_rules: %w", err)
	}
	s.publishActionItemChanged(actionItem.ProjectID)
	return actionItem, nil
}

// applyChildRulesForCreate auto-creates child action items per the project's
// template ChildRules whenever a parent action item's (Kind, StructuralType)
// matches a [[child_rules]] entry. The canonical case is:
//
//   - build  parent  -> build-qa-proof + build-qa-falsification children
//   - plan   parent  -> plan-qa-proof  + plan-qa-falsification  children
//
// Recursion terminates naturally because the QA kinds typically generated as
// children have no child_rules of their own. The recursive CreateActionItem
// call inherits the caller's auth ctx, so no fresh auth provisioning is
// required for the auto-children: the caller that was authorized to create
// the parent is implicitly authorized to materialize the cascade's mandated
// siblings/twins of that parent.
//
// Errors from individual child creates bubble up — partial cascade trees are
// the worst class of failure to leave behind silently. If a child fails the
// parent still exists in the repo (already committed by the time we get
// here); callers can re-run via supersede after diagnosing.
//
// The Owner field on the auto-created child is taken from the resolved kind
// rule (e.g. STEWARD-owned kinds carry the owner forward); BlockedBy is set
// when the rule declares blocked_by_parent. CreatedBy/UpdatedBy fields
// inherit from the parent so the audit trail traces back to the original
// human or agent that triggered the cascade.
func (s *Service) applyChildRulesForCreate(ctx context.Context, parent domain.ActionItem) error {
	project, err := s.repo.GetProject(ctx, parent.ProjectID)
	if err != nil {
		return fmt.Errorf("load project: %w", err)
	}
	// Project-tier-only template lookup. Per the design captured in
	// REFINEMENTS.md 2026-05-14 ("Remove project.Language; templates are
	// project-tier opt-in only"), child_rules auto-spawn fires only when the
	// project has explicitly authored a template at
	// <project.RepoBareRoot>/.tillsyn/template.toml or
	// <project.RepoPrimaryWorktree>/.tillsyn/template.toml. NO embedded
	// language-default fallback at this boundary — falling through to the
	// embedded till-<lang>.toml would impose arbitrary cascade behavior on
	// projects that did not opt in. The bake-time KindCatalog path
	// (bakeProjectKindCatalog) keeps the embedded fallback for backwards
	// compatibility of kind / agent-binding resolution until the architecture
	// refinement drop removes it; the create-time auto-spawn boundary is the
	// first place to enforce the project-tier-only invariant.
	tpl, ok, err := loadProjectTierTemplateOnly(&project)
	if err != nil {
		return fmt.Errorf("load project-tier template: %w", err)
	}
	if !ok {
		return nil
	}
	rules := tpl.ChildRulesFor(parent.Kind, parent.StructuralType)
	if len(rules) == 0 {
		return nil
	}
	columns, err := s.repo.ListColumns(ctx, parent.ProjectID, false)
	if err != nil {
		return fmt.Errorf("list columns: %w", err)
	}
	if len(columns) == 0 {
		return fmt.Errorf("project %q has no columns for auto-children", parent.ProjectID)
	}
	for _, rule := range rules {
		meta := domain.ActionItemMetadata{}
		if rule.BlockedByParent {
			meta.BlockedBy = []string{parent.ID}
		}
		childDescription := fmt.Sprintf("Auto-created by template child_rule on parent action item %s (kind=%s, structural_type=%s).",
			parent.ID, parent.Kind, parent.StructuralType)
		childOwner := strings.TrimSpace(string(rule.Owner))
		if _, createErr := s.CreateActionItem(ctx, CreateActionItemInput{
			ProjectID:      parent.ProjectID,
			ParentID:       parent.ID,
			Kind:           rule.Kind,
			StructuralType: rule.StructuralType,
			Owner:          childOwner,
			ColumnID:       columns[0].ID,
			Title:          rule.Title,
			Description:    childDescription,
			Priority:       domain.PriorityMedium,
			Metadata:       meta,
			CreatedByActor: parent.CreatedByActor,
			CreatedByName:  parent.CreatedByName,
			UpdatedByActor: parent.UpdatedByActor,
			UpdatedByName:  parent.UpdatedByName,
			UpdatedByType:  parent.UpdatedByType,
		}); createErr != nil {
			return fmt.Errorf("auto-create child kind=%s title=%q: %w", rule.Kind, rule.Title, createErr)
		}
	}
	return nil
}

// runGitStatusPreCheck enforces the droplet 4b.6 invariant that no action
// item with declared write-scope paths is created on top of a dirty tree.
// Caller has already verified len(paths) > 0.
//
// Skip cases (return nil silently):
//   - Project lookup fails — defer to the upstream `s.repo.ListColumns`
//     call inside CreateActionItem which already surfaces a typed error
//     for non-existent projects; pre-check is best-effort here.
//   - Project.RepoPrimaryWorktree is empty — pre-MVP escape valve per
//     droplet 4b.6 acceptance criterion 2. A project that hasn't been
//     bootstrapped to a checkout layout cannot enforce a dirty-tree gate.
//
// Reject cases:
//   - The configured GitStatusChecker reports one or more dirty paths;
//     wrap ErrPathsDirty with a comma-joined list of dirty paths.
//   - The checker returns a non-nil error (git missing on PATH, ctx
//     cancellation, pathspec out-of-worktree); propagate verbatim.
func (s *Service) runGitStatusPreCheck(ctx context.Context, projectID string, paths []string) error {
	if s.gitStatusChecker == nil {
		// Defensive: tests may inject a nil seam to bypass the check (e.g.
		// when a Service is constructed via direct struct-literal rather than
		// NewService — see internal/app tests that pre-date the seam wiring
		// and the dispatcher integration tests that opt out of the
		// dirty-tree gate). Production wiring (NewService at line ~196)
		// always populates defaultGitStatusChecker, so a nil seam in
		// production would itself be a bug — but we treat nil as "skip"
		// deliberately, NOT as "should never happen", because panicking
		// here would break the test-injection contract.
		return nil
	}
	project, err := s.repo.GetProject(ctx, projectID)
	if err != nil {
		// Project lookup races / non-existent projects surface via the
		// downstream repo write — keep the pre-check as a best-effort
		// guard rather than a second source of "not found" errors.
		return nil
	}
	worktree := strings.TrimSpace(project.RepoPrimaryWorktree)
	if worktree == "" {
		return nil
	}
	dirty, err := s.gitStatusChecker(ctx, worktree, paths)
	if err != nil {
		return fmt.Errorf("git status pre-check: %w", err)
	}
	if len(dirty) == 0 {
		return nil
	}
	return fmt.Errorf("%w: %s", ErrPathsDirty, strings.Join(dirty, ", "))
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
	// Drop 4c.5 droplet A.4: require a non-empty `metadata.outcome` from the
	// closed set {"failure", "blocked", "superseded"} on transitions into
	// `failed`. The check is positioned after the terminal-state guard so it
	// cannot race with partial state mutations, and it explicitly carves out
	// idempotent failed→failed self-moves so pre-A.4 data rows (action items
	// already at `failed` with empty outcome) are not retroactively rejected.
	// Asymmetric — the transition into `complete` does NOT require an
	// outcome; agents claiming success leave outcome unset by convention.
	// The expected agent pattern is `UpdateActionItem` to set
	// `metadata.outcome` BEFORE `MoveActionItem` flips the column (see
	// CLAUDE.md § "Action-Item Lifecycle"); this guard is a regression net
	// for buggy agents that skip the update step. `"success"` is rejected on
	// `→failed` because it is semantically nonsense (a success outcome on a
	// failed transition).
	if toState == domain.StateFailed && fromState != domain.StateFailed {
		outcome := strings.TrimSpace(strings.ToLower(actionItem.Metadata.Outcome))
		switch outcome {
		case "failure", "blocked", "superseded":
			// accepted
		default:
			return domain.ActionItem{}, fmt.Errorf("%w: metadata.outcome must be one of {failure, blocked, superseded} on transition to failed (got %q)", domain.ErrInvalidMetadataOutcome, actionItem.Metadata.Outcome)
		}
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
	s.publishActionItemChanged(actionItem.ProjectID)
	return actionItem, nil
}

// SupersedeActionItem is the dev-only escape hatch (Drop 4c.5 droplet B.1)
// that transitions one action item from `failed` to `complete` with
// `metadata.outcome = "superseded"` and the supplied dev-intent reason
// persisted on `metadata.transition_notes`. The supersede path bypasses
// `MoveActionItem`'s terminal-state guard (`service.go` ~line 1116) — that
// guard rejects every `failed → complete` move unconditionally pre-D3
// override-auth; this method is the typed escape hatch the project is
// willing to accept pre-MVP.
//
// Semantics (per THEME_BD_PLAN §3.1):
//
//   - Operates on EXACTLY the named node. NO cascade. Descendants in
//     non-terminal state keep their own state — supersede is "clear THIS
//     failure," not "abandon this whole subtree." The parent-blocks-on-
//     incomplete-child invariant still gates any subsequent attempt to move
//     a higher ancestor through complete via the existing
//     `ensureActionItemCompletionBlockersClear` chain.
//   - Only `failed` items are eligible. `todo` / `in_progress` /
//     `complete` / `archived` items reject with `domain.ErrTransitionBlocked`
//     wrapped with a "supersede only applies to failed items" hint. The
//     reject is NOT a silent no-op — calling supersede on a non-failed item
//     is operator confusion and must surface.
//   - Empty / whitespace-only reason rejects with a clear error. The reason
//     is the audit-trail substance the escape hatch exists to capture; an
//     empty reason defeats the point. The CLI layer also pre-rejects empty
//     reasons before invoking this method.
//   - The reason text persists on `metadata.transition_notes` (existing
//     free-form field on `ActionItemMetadata`). No new
//     `Metadata.SupersedeReason` field is added (YAGNI per THEME_BD_PLAN
//     §3.4 + cross-theme decisions).
//   - The capability guard runs with `CapabilityActionMarkComplete` for
//     symmetry with `MoveActionItem`'s `→complete` branch. No new
//     `CapabilityActionSupersede` action is introduced (YAGNI).
//   - STEWARD owner-state-lock: the adapter-layer caller is responsible for
//     the `assertOwnerStateGate` check before invoking this method, mirror-
//     ing the `MoveActionItem` adapter's pattern. See
//     `internal/adapters/mcp_common/app_service_adapter_mcp.go` for the
//     adapter passthrough.
//
// Returns the post-supersede `domain.ActionItem` with `LifecycleState =
// StateComplete`, `Metadata.Outcome = "superseded"`, and
// `Metadata.TransitionNotes = trimmed reason`. Errors propagate the
// underlying repo `ErrNotFound`, the in-method `ErrTransitionBlocked`, and
// any guard rejection verbatim.
func (s *Service) SupersedeActionItem(ctx context.Context, actionItemID, reason string) (domain.ActionItem, error) {
	actionItemID = strings.TrimSpace(actionItemID)
	if actionItemID == "" {
		return domain.ActionItem{}, fmt.Errorf("supersede: action_item_id is required")
	}
	trimmedReason := strings.TrimSpace(reason)
	if trimmedReason == "" {
		return domain.ActionItem{}, fmt.Errorf("supersede: reason is required (whitespace-only rejected)")
	}
	actionItem, err := s.repo.GetActionItem(ctx, actionItemID)
	if err != nil {
		return domain.ActionItem{}, err
	}
	guardScopes, err := s.capabilityScopesForActionItemLineage(ctx, actionItem)
	if err != nil {
		return domain.ActionItem{}, err
	}
	if err := s.enforceMutationGuardAcrossScopes(ctx, actionItem.ProjectID, currentMutationActorType(ctx, ""), guardScopes, domain.CapabilityActionMarkComplete); err != nil {
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
	if fromState != domain.StateFailed {
		return domain.ActionItem{}, fmt.Errorf("%w: supersede only applies to failed items (got state %q)", domain.ErrTransitionBlocked, fromState)
	}
	// Resolve the destination `complete` column for this project. The
	// auto-created column set always seeds a complete column; a missing
	// mapping signals project-wiring corruption rather than a normal flow
	// and surfaces explicitly so the dev sees the cause.
	completeColumnID := ""
	for _, column := range columns {
		if lifecycleStateForColumnID(columns, column.ID) == domain.StateComplete {
			completeColumnID = column.ID
			break
		}
	}
	if completeColumnID == "" {
		return domain.ActionItem{}, fmt.Errorf("supersede: project %q has no column mapped to lifecycle state %q", actionItem.ProjectID, domain.StateComplete)
	}
	// Stamp the audit-trail metadata BEFORE the column move. `outcome` and
	// `transition_notes` are existing free-form fields; the canonical
	// "superseded" outcome value is already accepted by
	// `validateMetadataOutcome` at the MCP adapter boundary
	// (app_service_adapter_mcp.go:1216). Direct field assignment is
	// intentional — we are NOT routing through the public
	// `UpdatePlanningMetadata` path because that path re-runs the full
	// metadata normalizer and would clobber any already-canonical fields the
	// caller did not touch. Trimming here mirrors the normalizer's
	// trim-on-input rule for `Outcome` and `TransitionNotes`.
	actionItem.Metadata.Outcome = "superseded"
	actionItem.Metadata.TransitionNotes = trimmedReason
	if err := actionItem.Move(completeColumnID, actionItem.Position, s.clock()); err != nil {
		return domain.ActionItem{}, err
	}
	if err := actionItem.SetLifecycleState(domain.StateComplete, s.clock()); err != nil {
		return domain.ActionItem{}, err
	}
	applyMutationActorToActionItem(ctx, &actionItem)
	if err := s.repo.UpdateActionItem(ctx, actionItem); err != nil {
		return domain.ActionItem{}, err
	}
	if _, err := s.enqueueActionItemEmbedding(ctx, actionItem, false, "task_superseded"); err != nil {
		return domain.ActionItem{}, err
	}
	s.publishActionItemChanged(actionItem.ProjectID)
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
	s.publishActionItemChanged(actionItem.ProjectID)
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
	s.publishActionItemChanged(actionItem.ProjectID)
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
	// Pointer-sentinel PATCH (Drop 4c.5 droplet A.1): each of Title /
	// Description / Priority / DueAt / Labels resolves to "preserve the
	// existing value" when its input pointer is nil, otherwise the
	// dereferenced value flows into the canonical domain.UpdateDetails
	// validator. Title's empty-string rejection still applies via
	// UpdateDetails — pointer-sentinel preserves field-presence semantics
	// and does not relax the title-required invariant.
	title := actionItem.Title
	if in.Title != nil {
		title = *in.Title
	}
	description := actionItem.Description
	if in.Description != nil {
		description = *in.Description
	}
	priority := actionItem.Priority
	if in.Priority != nil {
		priority = *in.Priority
	}
	dueAt := actionItem.DueAt
	if in.DueAt != nil {
		dueAt = *in.DueAt
	}
	labels := actionItem.Labels
	if in.Labels != nil {
		labels = *in.Labels
	}
	if err := actionItem.UpdateDetails(title, description, priority, dueAt, labels, s.clock()); err != nil {
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
	//
	// After the assignment, the post-merge (structural_type, parent_id)
	// pair is re-checked against the cascade-positional invariant via
	// domain.ValidatePositionalInvariant so the Update path cannot bypass
	// the rule NewActionItem enforces at construction (Lane A D5 — closes
	// the C2 finding from the D1 build-QA-falsification comment). ParentID
	// is not mutable through UpdateActionItem, so the existing persisted
	// value flows in directly; the symmetric ReparentActionItem gate covers
	// the parent_id-mutation half of the bypass.
	if normalized := domain.NormalizeStructuralType(in.StructuralType); normalized != "" {
		if !domain.IsValidStructuralType(normalized) {
			return domain.ActionItem{}, domain.ErrInvalidStructuralType
		}
		if err := domain.ValidatePositionalInvariant(normalized, actionItem.ParentID); err != nil {
			return domain.ActionItem{}, err
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
	// Files update uses pointer-sentinel: nil preserves the existing slice;
	// non-nil applies the dereferenced slice through
	// domain.NormalizeActionItemFiles so the create-time trim/dedupe /
	// forward-slash-check rules apply equally on update. Empty dereferenced
	// slice clears all declared files (explicit caller intent). Disjoint-
	// axis with Paths — no coverage / overlap re-check applies.
	if in.Files != nil {
		normalized, err := domain.NormalizeActionItemFiles(*in.Files)
		if err != nil {
			return domain.ActionItem{}, err
		}
		actionItem.Files = normalized
		actionItem.UpdatedAt = s.clock().UTC()
	}
	// StartCommit update uses pointer-sentinel: nil preserves the existing
	// value; non-nil applies the dereferenced string, trimmed to match the
	// create-time rule. Empty dereferenced string clears the prior commit
	// hash (explicit caller intent — e.g. dispatcher rolling back a
	// retry). Inline trim mirrors NewActionItem's inline-trim approach;
	// no domain helper exposed (single-line trim too thin to warrant the
	// wrapper). Domain primitive per Drop 4a L3 / WAVE_1_PLAN.md §1.4.
	if in.StartCommit != nil {
		actionItem.StartCommit = strings.TrimSpace(*in.StartCommit)
		actionItem.UpdatedAt = s.clock().UTC()
	}
	// EndCommit update mirrors StartCommit's pointer-sentinel handling:
	// nil preserves the existing value; non-nil applies the dereferenced
	// string, trimmed to match the create-time rule. Empty dereferenced
	// string clears the prior commit hash (explicit caller intent — e.g.
	// dispatcher rolling back a retry). Wave 2 dispatcher populates this
	// before MoveActionItemState so the terminal capture lands cleanly.
	// Inline trim mirrors NewActionItem; no domain helper exposed. Domain
	// primitive per Drop 4a L3 / WAVE_1_PLAN.md §1.5.
	if in.EndCommit != nil {
		actionItem.EndCommit = strings.TrimSpace(*in.EndCommit)
		actionItem.UpdatedAt = s.clock().UTC()
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
	s.publishActionItemChanged(actionItem.ProjectID)
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
		if err := s.repo.UpdateActionItem(ctx, actionItem); err != nil {
			return err
		}
		s.publishActionItemChanged(actionItem.ProjectID)
		return nil
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
		s.publishActionItemChanged(actionItem.ProjectID)
		return nil
	default:
		return ErrInvalidDeleteMode
	}
}

// ListProjects lists projects.
func (s *Service) ListProjects(ctx context.Context, includeArchived bool) ([]domain.Project, error) {
	return s.repo.ListProjects(ctx, includeArchived)
}

// GetProject returns one project by ID. Wave 2.10 (droplet 4a.23) needs
// repo-level project lookup so the manual-trigger CLI can resolve the
// dispatcher's spawn-time fields (RepoPrimaryWorktree, KindCatalogJSON,
// HyllaArtifactRef) from an action item's ProjectID. Mirrors GetActionItem's
// shape: trim, validate non-empty, delegate to repo, surface ErrInvalidID on
// empty input.
func (s *Service) GetProject(ctx context.Context, projectID string) (domain.Project, error) {
	if s == nil || s.repo == nil {
		return domain.Project{}, fmt.Errorf("service is not configured")
	}
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		return domain.Project{}, domain.ErrInvalidID
	}
	return s.repo.GetProject(ctx, projectID)
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

// ListActionItemsByState returns the project's action items whose
// LifecycleState equals state, sorted by UpdatedAt descending. Drop 4c.5
// droplet B.2's failure-listing CLI is the canonical caller — the dev needs a
// pre-TUI view of `failed` items so they can clear stuck nodes via the
// supersede CLI. The contract is intentionally narrow: filter by ONE
// lifecycle state, no kind / role / priority cross-filters, no pagination
// (pre-MVP scale is hundreds of items per project).
//
// Filter semantics:
//
//  1. The filter is applied IN MEMORY after `Service.ListActionItems` returns
//     the project's full action-item set. At pre-MVP scale (<1k items per
//     project) this is fine; an indexed-query refactor is deferred until
//     measurement justifies it. The decision is documented inline so a
//     future planner sees the intent rather than guessing the scale ceiling.
//  2. `includeArchived` extends the underlying `ListActionItems` to also
//     surface archived rows. When `state == StateArchived` the flag is
//     forced to true (asking for archived items implies including them);
//     for every other state the caller's flag is honored as-is. This
//     resolves the "archived ≠ failed" axis-orthogonality cleanly: a row
//     that is BOTH `state=failed` AND `archived_at != nil` shows up exactly
//     once when the caller passes `state=failed, includeArchived=true`,
//     because `ListActionItems` returns each row once regardless of the
//     two flags.
//  3. Sort order is `UpdatedAt` DESC so the most-recently-failed items
//     surface first — the pre-TUI use case is "what is stuck right now,"
//     not historical archaeology.
//
// Validation:
//
//   - Empty `projectID` returns `ErrInvalidID`.
//   - Empty `state` returns a clear error naming the valid set; the CLI
//     defaults the flag to `"failed"` so this branch only fires on bad
//     direct callers (the CLI itself never passes an empty state).
//   - Invalid `state` returns a clear error naming the valid set.
//
// The MCP adapter does NOT yet expose this method; it is CLI-only today.
func (s *Service) ListActionItemsByState(ctx context.Context, projectID string, state domain.LifecycleState, includeArchived bool) ([]domain.ActionItem, error) {
	if s == nil || s.repo == nil {
		return nil, fmt.Errorf("service is not configured")
	}
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		return nil, domain.ErrInvalidID
	}
	normalized := domain.LifecycleState(strings.TrimSpace(strings.ToLower(string(state))))
	if normalized == "" {
		return nil, fmt.Errorf("list action items by state: state is required (valid: todo, in_progress, complete, failed, archived)")
	}
	switch normalized {
	case domain.StateTodo, domain.StateInProgress, domain.StateComplete, domain.StateFailed, domain.StateArchived:
		// known lifecycle state; fall through to filter
	default:
		return nil, fmt.Errorf("list action items by state: unknown state %q (valid: todo, in_progress, complete, failed, archived)", string(state))
	}
	// Asking for archived items implies including them. For every other
	// state the caller's flag is honored as-is so failed+archived rows
	// surface only when explicitly requested via `includeArchived=true`.
	effectiveIncludeArchived := includeArchived
	if normalized == domain.StateArchived {
		effectiveIncludeArchived = true
	}
	all, err := s.ListActionItems(ctx, projectID, effectiveIncludeArchived)
	if err != nil {
		return nil, err
	}
	filtered := make([]domain.ActionItem, 0, len(all))
	for _, item := range all {
		if item.LifecycleState == normalized {
			filtered = append(filtered, item)
		}
	}
	slices.SortFunc(filtered, func(a, b domain.ActionItem) int {
		// Most recent first. Tie-break on ID for total ordering so test
		// assertions are stable across runs.
		if a.UpdatedAt.Equal(b.UpdatedAt) {
			return strings.Compare(a.ID, b.ID)
		}
		if a.UpdatedAt.After(b.UpdatedAt) {
			return -1
		}
		return 1
	})
	return filtered, nil
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
	// Cascade-positional invariant gate (Lane A D5 — closes the C2 finding
	// from the D1 build-QA-falsification comment). Every reparent changes
	// parent_id, so the new (structural_type, parent_id) pair must satisfy
	// domain.ValidatePositionalInvariant against the current
	// actionItem.StructuralType. Checked BEFORE the parent existence /
	// project-match / cycle gates so the invariant short-circuits without
	// extra repo round-trips when the tuple itself is invalid (e.g.
	// reparenting a cascade row to any non-empty parent is unconditionally
	// wrong regardless of whether that parent exists). UpdateActionItem
	// holds the symmetric structural_type-patch half of the bypass closure.
	if err := domain.ValidatePositionalInvariant(actionItem.StructuralType, parentID); err != nil {
		return domain.ActionItem{}, err
	}
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
	s.publishActionItemChanged(actionItem.ProjectID)
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
