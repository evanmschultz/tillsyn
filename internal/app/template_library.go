package app

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/evanmschultz/tillsyn/internal/domain"
)

const (
	templateSystemActorID   = "tillsyn-system-template"
	templateSystemActorName = "Tillsyn System Template"
)

// UpsertTemplateLibraryInput stores write-time values for one template-library upsert.
type UpsertTemplateLibraryInput struct {
	ID                  string
	Scope               domain.TemplateLibraryScope
	ProjectID           string
	Name                string
	Description         string
	Status              domain.TemplateLibraryStatus
	SourceLibraryID     string
	BuiltinManaged      bool
	BuiltinSource       string
	BuiltinVersion      string
	CreatedByActorID    string
	CreatedByActorName  string
	CreatedByActorType  domain.ActorType
	ApprovedByActorID   string
	ApprovedByActorName string
	ApprovedByActorType domain.ActorType
	NodeTemplates       []UpsertNodeTemplateInput
}

// UpsertNodeTemplateInput stores write-time values for one node template nested under a library.
type UpsertNodeTemplateInput struct {
	ID                         string
	ScopeLevel                 domain.KindAppliesTo
	NodeKindID                 domain.KindID
	DisplayName                string
	DescriptionMarkdown        string
	ProjectMetadataDefaults    *domain.ProjectMetadata
	ActionItemMetadataDefaults *domain.ActionItemMetadata
	ChildRules                 []UpsertTemplateChildRuleInput
}

// UpsertTemplateChildRuleInput stores write-time values for one nested child rule.
type UpsertTemplateChildRuleInput struct {
	ID                        string
	Position                  int
	ChildScopeLevel           domain.KindAppliesTo
	ChildKindID               domain.KindID
	TitleTemplate             string
	DescriptionTemplate       string
	ResponsibleActorKind      domain.TemplateActorKind
	EditableByActorKinds      []domain.TemplateActorKind
	CompletableByActorKinds   []domain.TemplateActorKind
	OrchestratorMayComplete   bool
	RequiredForParentDone     bool
	RequiredForContainingDone bool
}

// ListTemplateLibrariesInput stores list-time filters for template-library queries.
type ListTemplateLibrariesInput struct {
	Scope     domain.TemplateLibraryScope
	ProjectID string
	Status    domain.TemplateLibraryStatus
}

// BindProjectTemplateLibraryInput stores one project-to-library binding request.
type BindProjectTemplateLibraryInput struct {
	ProjectID        string
	LibraryID        string
	BoundByActorID   string
	BoundByActorName string
	BoundByActorType domain.ActorType
}

// UnbindProjectTemplateLibraryInput stores one project-to-library unbind request.
type UnbindProjectTemplateLibraryInput struct {
	ProjectID string
}

// ListTemplateLibraries lists template libraries with deterministic ordering.
func (s *Service) ListTemplateLibraries(ctx context.Context, in ListTemplateLibrariesInput) ([]domain.TemplateLibrary, error) {
	libraries, err := s.repo.ListTemplateLibraries(ctx, domain.TemplateLibraryFilter{
		Scope:     in.Scope,
		ProjectID: strings.TrimSpace(in.ProjectID),
		Status:    in.Status,
	})
	if err != nil {
		return nil, err
	}
	sort.SliceStable(libraries, func(i, j int) bool {
		if libraries[i].Scope == libraries[j].Scope {
			if libraries[i].ProjectID == libraries[j].ProjectID {
				if libraries[i].Name == libraries[j].Name {
					return libraries[i].ID < libraries[j].ID
				}
				return libraries[i].Name < libraries[j].Name
			}
			return libraries[i].ProjectID < libraries[j].ProjectID
		}
		return libraries[i].Scope < libraries[j].Scope
	})
	return libraries, nil
}

// GetTemplateLibrary loads one template library by id.
func (s *Service) GetTemplateLibrary(ctx context.Context, libraryID string) (domain.TemplateLibrary, error) {
	libraryID = domain.NormalizeTemplateLibraryID(libraryID)
	if libraryID == "" {
		return domain.TemplateLibrary{}, domain.ErrInvalidID
	}
	return s.repo.GetTemplateLibrary(ctx, libraryID)
}

// UpsertTemplateLibrary creates or updates one template library and all nested rules.
func (s *Service) UpsertTemplateLibrary(ctx context.Context, in UpsertTemplateLibraryInput) (domain.TemplateLibrary, error) {
	if err := s.ensureKindCatalogBootstrapped(ctx); err != nil {
		return domain.TemplateLibrary{}, err
	}
	now := s.clock()
	ctx, resolvedActor, hasResolvedActor := withResolvedMutationActor(ctx, in.CreatedByActorID, in.CreatedByActorName, in.CreatedByActorType)
	if in.Scope != domain.TemplateLibraryScopeGlobal {
		if strings.TrimSpace(in.ProjectID) == "" {
			return domain.TemplateLibrary{}, domain.ErrInvalidID
		}
		if _, err := s.repo.GetProject(ctx, strings.TrimSpace(in.ProjectID)); err != nil {
			return domain.TemplateLibrary{}, err
		}
	}
	nodeTemplates := make([]domain.NodeTemplateInput, 0, len(in.NodeTemplates))
	for _, nodeTemplateIn := range in.NodeTemplates {
		nodeTemplateID := domain.NormalizeTemplateLibraryID(nodeTemplateIn.ID)
		if nodeTemplateID == "" {
			nodeTemplateID = domain.NormalizeTemplateLibraryID(s.idGen())
		}
		if _, err := s.repo.GetKindDefinition(ctx, nodeTemplateIn.NodeKindID); err != nil {
			if errors.Is(err, ErrNotFound) {
				return domain.TemplateLibrary{}, fmt.Errorf("%w: %q", domain.ErrKindNotFound, nodeTemplateIn.NodeKindID)
			}
			return domain.TemplateLibrary{}, err
		}
		childRules := make([]domain.TemplateChildRuleInput, 0, len(nodeTemplateIn.ChildRules))
		for _, childRuleIn := range nodeTemplateIn.ChildRules {
			childRuleID := domain.NormalizeTemplateLibraryID(childRuleIn.ID)
			if childRuleID == "" {
				childRuleID = domain.NormalizeTemplateLibraryID(s.idGen())
			}
			if _, err := s.repo.GetKindDefinition(ctx, childRuleIn.ChildKindID); err != nil {
				if errors.Is(err, ErrNotFound) {
					return domain.TemplateLibrary{}, fmt.Errorf("%w: %q", domain.ErrKindNotFound, childRuleIn.ChildKindID)
				}
				return domain.TemplateLibrary{}, err
			}
			childRules = append(childRules, domain.TemplateChildRuleInput{
				ID:                        childRuleID,
				Position:                  childRuleIn.Position,
				ChildScopeLevel:           childRuleIn.ChildScopeLevel,
				ChildKindID:               childRuleIn.ChildKindID,
				TitleTemplate:             childRuleIn.TitleTemplate,
				DescriptionTemplate:       childRuleIn.DescriptionTemplate,
				ResponsibleActorKind:      childRuleIn.ResponsibleActorKind,
				EditableByActorKinds:      append([]domain.TemplateActorKind(nil), childRuleIn.EditableByActorKinds...),
				CompletableByActorKinds:   append([]domain.TemplateActorKind(nil), childRuleIn.CompletableByActorKinds...),
				OrchestratorMayComplete:   childRuleIn.OrchestratorMayComplete,
				RequiredForParentDone:     childRuleIn.RequiredForParentDone,
				RequiredForContainingDone: childRuleIn.RequiredForContainingDone,
			})
		}
		nodeTemplates = append(nodeTemplates, domain.NodeTemplateInput{
			ID:                         nodeTemplateID,
			ScopeLevel:                 nodeTemplateIn.ScopeLevel,
			NodeKindID:                 nodeTemplateIn.NodeKindID,
			DisplayName:                nodeTemplateIn.DisplayName,
			DescriptionMarkdown:        nodeTemplateIn.DescriptionMarkdown,
			ProjectMetadataDefaults:    nodeTemplateIn.ProjectMetadataDefaults,
			ActionItemMetadataDefaults: nodeTemplateIn.ActionItemMetadataDefaults,
			ChildRules:                 childRules,
		})
	}

	approvedAt := (*time.Time)(nil)
	if in.Status == domain.TemplateLibraryStatusApproved {
		ts := now.UTC()
		approvedAt = &ts
	}
	library, err := domain.NewTemplateLibrary(domain.TemplateLibraryInput{
		ID:                  firstNonEmptyTrimmed(in.ID, s.idGen()),
		Scope:               in.Scope,
		ProjectID:           strings.TrimSpace(in.ProjectID),
		Name:                in.Name,
		Description:         in.Description,
		Status:              in.Status,
		SourceLibraryID:     in.SourceLibraryID,
		BuiltinManaged:      in.BuiltinManaged,
		BuiltinSource:       in.BuiltinSource,
		BuiltinVersion:      in.BuiltinVersion,
		CreatedByActorID:    firstNonEmptyTrimmed(in.CreatedByActorID, resolvedActor.ActorID),
		CreatedByActorName:  firstNonEmptyTrimmed(in.CreatedByActorName, resolvedActor.ActorName, in.CreatedByActorID),
		CreatedByActorType:  normalizeActorTypeInput(firstActorType(in.CreatedByActorType, resolvedActor.ActorType)),
		ApprovedByActorID:   firstNonEmptyTrimmed(in.ApprovedByActorID, resolvedActor.ActorID),
		ApprovedByActorName: firstNonEmptyTrimmed(in.ApprovedByActorName, resolvedActor.ActorName, in.ApprovedByActorID),
		ApprovedByActorType: normalizeActorTypeInput(firstActorType(in.ApprovedByActorType, resolvedActor.ActorType)),
		ApprovedAt:          approvedAt,
		NodeTemplates:       nodeTemplates,
	}, now)
	if err != nil {
		return domain.TemplateLibrary{}, err
	}
	library.RevisionDigest = library.RevisionFingerprint()
	if library.RevisionDigest == "" {
		return domain.TemplateLibrary{}, domain.ErrInvalidTemplateLibrary
	}

	existing, getErr := s.repo.GetTemplateLibrary(ctx, library.ID)
	switch {
	case getErr == nil:
		library.CreatedAt = existing.CreatedAt
		library.CreatedByActorID = existing.CreatedByActorID
		library.CreatedByActorName = existing.CreatedByActorName
		library.CreatedByActorType = existing.CreatedByActorType
		library.Revision = max(existing.Revision, 1)
		if strings.TrimSpace(existing.RevisionDigest) != "" && existing.RevisionDigest != library.RevisionDigest {
			library.Revision++
		}
		library.UpdatedAt = now.UTC()
	case errors.Is(getErr, ErrNotFound):
		library.Revision = max(library.Revision, 1)
		if hasResolvedActor && library.CreatedByActorID == "" {
			library.CreatedByActorID = resolvedActor.ActorID
			library.CreatedByActorName = resolvedActor.ActorName
			library.CreatedByActorType = resolvedActor.ActorType
		}
	default:
		return domain.TemplateLibrary{}, getErr
	}
	if err := s.repo.UpsertTemplateLibrary(ctx, library); err != nil {
		return domain.TemplateLibrary{}, err
	}
	return library, nil
}

// BindProjectTemplateLibrary binds one project to one approved template library.
func (s *Service) BindProjectTemplateLibrary(ctx context.Context, in BindProjectTemplateLibraryInput) (domain.ProjectTemplateBinding, error) {
	projectID := strings.TrimSpace(in.ProjectID)
	if projectID == "" {
		return domain.ProjectTemplateBinding{}, domain.ErrInvalidID
	}
	project, err := s.repo.GetProject(ctx, projectID)
	if err != nil {
		return domain.ProjectTemplateBinding{}, err
	}
	library, err := s.repo.GetTemplateLibrary(ctx, in.LibraryID)
	if err != nil {
		return domain.ProjectTemplateBinding{}, err
	}
	if library.Status != domain.TemplateLibraryStatusApproved {
		return domain.ProjectTemplateBinding{}, fmt.Errorf("%w: library must be approved before binding", domain.ErrInvalidTemplateBinding)
	}
	if library.Scope == domain.TemplateLibraryScopeProject && strings.TrimSpace(library.ProjectID) != "" && strings.TrimSpace(library.ProjectID) != projectID {
		return domain.ProjectTemplateBinding{}, fmt.Errorf("%w: project library %q belongs to project %q", domain.ErrInvalidTemplateBinding, library.ID, library.ProjectID)
	}
	now := s.clock()
	binding, err := domain.NewProjectTemplateBinding(domain.ProjectTemplateBindingInput{
		ProjectID:             projectID,
		LibraryID:             library.ID,
		LibraryName:           library.Name,
		BoundRevision:         max(library.Revision, 1),
		BoundRevisionDigest:   library.RevisionDigest,
		BoundLibraryUpdatedAt: &library.UpdatedAt,
		BoundByActorID:        in.BoundByActorID,
		BoundByActorName:      in.BoundByActorName,
		BoundByActorType:      in.BoundByActorType,
		BoundLibrarySnapshot:  &library,
	}, now)
	if err != nil {
		return domain.ProjectTemplateBinding{}, err
	}
	previousBinding, previousBindingFound, err := loadExistingProjectBindingForAllowlist(ctx, s.repo, projectID)
	if err != nil {
		return domain.ProjectTemplateBinding{}, err
	}
	if err := s.repo.UpsertProjectTemplateBinding(ctx, binding); err != nil {
		return domain.ProjectTemplateBinding{}, err
	}
	refreshAllowlist, err := s.shouldRefreshProjectAllowlistForBinding(ctx, project, previousBinding, previousBindingFound)
	if err != nil {
		return domain.ProjectTemplateBinding{}, err
	}
	if refreshAllowlist {
		if err := s.initializeProjectAllowedKinds(ctx, project, &library); err != nil {
			return domain.ProjectTemplateBinding{}, err
		}
	}
	return s.enrichProjectTemplateBinding(ctx, binding)
}

// GetProjectTemplateBinding loads the active binding for one project.
func (s *Service) GetProjectTemplateBinding(ctx context.Context, projectID string) (domain.ProjectTemplateBinding, error) {
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		return domain.ProjectTemplateBinding{}, domain.ErrInvalidID
	}
	binding, err := s.repo.GetProjectTemplateBinding(ctx, projectID)
	if err != nil {
		return domain.ProjectTemplateBinding{}, err
	}
	return s.enrichProjectTemplateBinding(ctx, binding)
}

// UnbindProjectTemplateLibrary removes the active template-library binding for one project.
func (s *Service) UnbindProjectTemplateLibrary(ctx context.Context, in UnbindProjectTemplateLibraryInput) error {
	projectID := strings.TrimSpace(in.ProjectID)
	if projectID == "" {
		return domain.ErrInvalidID
	}
	if _, err := s.repo.GetProject(ctx, projectID); err != nil {
		return err
	}
	if err := s.repo.DeleteProjectTemplateBinding(ctx, projectID); err != nil {
		return err
	}
	return nil
}

// GetNodeContractSnapshot loads one generated-node contract snapshot.
func (s *Service) GetNodeContractSnapshot(ctx context.Context, nodeID string) (domain.NodeContractSnapshot, error) {
	nodeID = strings.TrimSpace(nodeID)
	if nodeID == "" {
		return domain.NodeContractSnapshot{}, domain.ErrInvalidID
	}
	return s.repo.GetNodeContractSnapshot(ctx, nodeID)
}

// resolveProjectCreateTemplateLibrary loads one approved global library for project creation.
func (s *Service) resolveProjectCreateTemplateLibrary(ctx context.Context, libraryID string, projectKindID domain.KindID) (domain.TemplateLibrary, domain.NodeTemplate, bool, error) {
	libraryID = domain.NormalizeTemplateLibraryID(libraryID)
	if libraryID == "" {
		return domain.TemplateLibrary{}, domain.NodeTemplate{}, false, nil
	}
	library, err := s.repo.GetTemplateLibrary(ctx, libraryID)
	if err != nil {
		return domain.TemplateLibrary{}, domain.NodeTemplate{}, false, err
	}
	if library.Status != domain.TemplateLibraryStatusApproved {
		return domain.TemplateLibrary{}, domain.NodeTemplate{}, false, fmt.Errorf("%w: library must be approved before project creation", domain.ErrInvalidTemplateBinding)
	}
	if library.Scope != domain.TemplateLibraryScopeGlobal {
		return domain.TemplateLibrary{}, domain.NodeTemplate{}, false, fmt.Errorf("%w: project creation currently accepts approved global libraries only", domain.ErrInvalidTemplateBinding)
	}
	nodeTemplate, found := library.FindNodeTemplate(domain.KindAppliesToProject, projectKindID)
	return library, nodeTemplate, found, nil
}

// resolveBoundNodeTemplate resolves the bound node template for one project scope/kind tuple.
func (s *Service) resolveBoundNodeTemplate(ctx context.Context, projectID string, scope domain.KindAppliesTo, kindID domain.KindID) (domain.TemplateLibrary, domain.NodeTemplate, bool, error) {
	binding, err := s.repo.GetProjectTemplateBinding(ctx, projectID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return domain.TemplateLibrary{}, domain.NodeTemplate{}, false, nil
		}
		return domain.TemplateLibrary{}, domain.NodeTemplate{}, false, err
	}
	library := domain.TemplateLibrary{}
	if binding.BoundLibrarySnapshot != nil {
		library = *binding.BoundLibrarySnapshot
	}
	if library.ID == "" {
		library, err = s.repo.GetTemplateLibrary(ctx, binding.LibraryID)
		if err != nil {
			return domain.TemplateLibrary{}, domain.NodeTemplate{}, false, err
		}
	}
	nodeTemplate, ok := library.FindNodeTemplate(scope, kindID)
	if !ok {
		return library, domain.NodeTemplate{}, false, nil
	}
	return library, nodeTemplate, true, nil
}

func (s *Service) enrichProjectTemplateBinding(ctx context.Context, binding domain.ProjectTemplateBinding) (domain.ProjectTemplateBinding, error) {
	if binding.LibraryName == "" && binding.BoundLibrarySnapshot != nil {
		binding.LibraryName = binding.BoundLibrarySnapshot.Name
	}
	latest, err := s.repo.GetTemplateLibrary(ctx, binding.LibraryID)
	switch {
	case err == nil:
		binding.LatestRevision = max(latest.Revision, 1)
		binding.LatestRevisionDigest = latest.RevisionDigest
		ts := latest.UpdatedAt.UTC()
		binding.LatestLibraryUpdatedAt = &ts
		if binding.LibraryName == "" {
			binding.LibraryName = latest.Name
		}
		if binding.BoundRevision == binding.LatestRevision && strings.TrimSpace(binding.BoundRevisionDigest) == strings.TrimSpace(binding.LatestRevisionDigest) {
			binding.DriftStatus = domain.ProjectTemplateBindingDriftCurrent
		} else {
			binding.DriftStatus = domain.ProjectTemplateBindingDriftUpdateAvailable
		}
	case errors.Is(err, ErrNotFound):
		binding.DriftStatus = domain.ProjectTemplateBindingDriftLibraryMissing
	default:
		return domain.ProjectTemplateBinding{}, err
	}
	if binding.BoundRevision == 0 {
		binding.BoundRevision = 1
	}
	if binding.BoundLibraryUpdatedAt.IsZero() {
		if binding.BoundLibrarySnapshot != nil && !binding.BoundLibrarySnapshot.UpdatedAt.IsZero() {
			binding.BoundLibraryUpdatedAt = binding.BoundLibrarySnapshot.UpdatedAt.UTC()
		} else if binding.LatestLibraryUpdatedAt != nil {
			binding.BoundLibraryUpdatedAt = binding.LatestLibraryUpdatedAt.UTC()
		}
	}
	return binding, nil
}

// loadExistingProjectBindingForAllowlist loads one binding when it exists so bind-time allowlist sync can compare policy shapes.
func loadExistingProjectBindingForAllowlist(ctx context.Context, repo Repository, projectID string) (domain.ProjectTemplateBinding, bool, error) {
	binding, err := repo.GetProjectTemplateBinding(ctx, projectID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return domain.ProjectTemplateBinding{}, false, nil
		}
		return domain.ProjectTemplateBinding{}, false, err
	}
	return binding, true, nil
}

// shouldRefreshProjectAllowlistForBinding reports whether an explicit bind should replace the current allowlist with template-derived kinds.
func (s *Service) shouldRefreshProjectAllowlistForBinding(ctx context.Context, project domain.Project, previousBinding domain.ProjectTemplateBinding, previousBindingFound bool) (bool, error) {
	currentKinds, err := s.ListProjectAllowedKinds(ctx, project.ID)
	if err != nil {
		return false, err
	}
	currentKinds = normalizeKindIDList(currentKinds)
	defaultKinds, err := s.defaultProjectAllowedKindIDs(ctx, project.Kind)
	if err != nil {
		return false, err
	}
	if slices.Equal(currentKinds, defaultKinds) {
		return true, nil
	}
	if !previousBindingFound {
		return false, nil
	}
	previousLibrary, err := librarySnapshotForBinding(ctx, s.repo, previousBinding)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return false, nil
		}
		return false, err
	}
	previousTemplateKinds := templateDerivedProjectAllowedKindIDs(project.Kind, previousLibrary)
	return slices.Equal(currentKinds, previousTemplateKinds), nil
}

// librarySnapshotForBinding resolves the most specific library snapshot available for one binding.
func librarySnapshotForBinding(ctx context.Context, repo Repository, binding domain.ProjectTemplateBinding) (*domain.TemplateLibrary, error) {
	if binding.BoundLibrarySnapshot != nil {
		snapshot := *binding.BoundLibrarySnapshot
		return &snapshot, nil
	}
	libraryID := strings.TrimSpace(binding.LibraryID)
	if libraryID == "" {
		return nil, ErrNotFound
	}
	library, err := repo.GetTemplateLibrary(ctx, libraryID)
	if err != nil {
		return nil, err
	}
	return &library, nil
}

// mergeActionItemMetadataWithNodeTemplate applies actionItem defaults from one node template at create time.
func mergeActionItemMetadataWithNodeTemplate(base domain.ActionItemMetadata, nodeTemplate domain.NodeTemplate) (domain.ActionItemMetadata, error) {
	if nodeTemplate.ActionItemMetadataDefaults == nil {
		return domain.MergeActionItemMetadata(base, nil)
	}
	return domain.MergeActionItemMetadata(base, nodeTemplate.ActionItemMetadataDefaults)
}

// mergeProjectMetadataWithNodeTemplate applies project defaults from one node template at create time.
func mergeProjectMetadataWithNodeTemplate(base domain.ProjectMetadata, nodeTemplate domain.NodeTemplate) (domain.ProjectMetadata, error) {
	if nodeTemplate.ProjectMetadataDefaults == nil {
		return domain.MergeProjectMetadata(base, nil)
	}
	return domain.MergeProjectMetadata(base, nodeTemplate.ProjectMetadataDefaults)
}

// applyTemplateChildRules creates generated child nodes and stores their node-contract snapshots.
func (s *Service) applyTemplateChildRules(ctx context.Context, parent domain.ActionItem, library domain.TemplateLibrary, nodeTemplate domain.NodeTemplate, depth int) error {
	if len(nodeTemplate.ChildRules) == 0 {
		return nil
	}
	for _, childRule := range nodeTemplate.ChildRules {
		child, err := s.createActionItemWithTemplates(withInternalTemplateMutation(ctx), CreateActionItemInput{
			ProjectID:      parent.ProjectID,
			ParentID:       parent.ID,
			Kind:           domain.WorkKind(childRule.ChildKindID),
			Scope:          childRule.ChildScopeLevel,
			ColumnID:       parent.ColumnID,
			Title:          childRule.TitleTemplate,
			Description:    childRule.DescriptionTemplate,
			Priority:       domain.PriorityMedium,
			CreatedByActor: templateSystemActorID,
			CreatedByName:  templateSystemActorName,
			UpdatedByActor: templateSystemActorID,
			UpdatedByName:  templateSystemActorName,
			UpdatedByType:  domain.ActorTypeSystem,
		}, depth)
		if err != nil {
			return err
		}
		snapshot, err := domain.NewNodeContractSnapshot(domain.NodeContractSnapshotInput{
			NodeID:                    child.ID,
			ProjectID:                 child.ProjectID,
			SourceLibraryID:           library.ID,
			SourceNodeTemplateID:      nodeTemplate.ID,
			SourceChildRuleID:         childRule.ID,
			CreatedByActorID:          templateSystemActorID,
			CreatedByActorType:        domain.ActorTypeSystem,
			ResponsibleActorKind:      childRule.ResponsibleActorKind,
			EditableByActorKinds:      append([]domain.TemplateActorKind(nil), childRule.EditableByActorKinds...),
			CompletableByActorKinds:   append([]domain.TemplateActorKind(nil), childRule.CompletableByActorKinds...),
			OrchestratorMayComplete:   childRule.OrchestratorMayComplete,
			RequiredForParentDone:     childRule.RequiredForParentDone,
			RequiredForContainingDone: childRule.RequiredForContainingDone,
		}, s.clock())
		if err != nil {
			return err
		}
		if err := s.repo.CreateNodeContractSnapshot(ctx, snapshot); err != nil {
			return err
		}
	}
	return nil
}

// applyProjectTemplateChildRules creates project-root generated nodes and stores their node-contract snapshots.
func (s *Service) applyProjectTemplateChildRules(ctx context.Context, project domain.Project, library domain.TemplateLibrary, nodeTemplate domain.NodeTemplate, depth int) error {
	if len(nodeTemplate.ChildRules) == 0 {
		return nil
	}
	columnID, err := s.ensureTemplateRootColumn(ctx, project.ID, s.clock())
	if err != nil {
		return err
	}
	for _, childRule := range nodeTemplate.ChildRules {
		child, err := s.createActionItemWithTemplates(withInternalTemplateMutation(ctx), CreateActionItemInput{
			ProjectID:      project.ID,
			Kind:           domain.WorkKind(childRule.ChildKindID),
			Scope:          childRule.ChildScopeLevel,
			ColumnID:       columnID,
			Title:          childRule.TitleTemplate,
			Description:    childRule.DescriptionTemplate,
			Priority:       domain.PriorityMedium,
			CreatedByActor: templateSystemActorID,
			CreatedByName:  templateSystemActorName,
			UpdatedByActor: templateSystemActorID,
			UpdatedByName:  templateSystemActorName,
			UpdatedByType:  domain.ActorTypeSystem,
		}, depth)
		if err != nil {
			return err
		}
		snapshot, err := domain.NewNodeContractSnapshot(domain.NodeContractSnapshotInput{
			NodeID:                    child.ID,
			ProjectID:                 child.ProjectID,
			SourceLibraryID:           library.ID,
			SourceNodeTemplateID:      nodeTemplate.ID,
			SourceChildRuleID:         childRule.ID,
			CreatedByActorID:          templateSystemActorID,
			CreatedByActorType:        domain.ActorTypeSystem,
			ResponsibleActorKind:      childRule.ResponsibleActorKind,
			EditableByActorKinds:      append([]domain.TemplateActorKind(nil), childRule.EditableByActorKinds...),
			CompletableByActorKinds:   append([]domain.TemplateActorKind(nil), childRule.CompletableByActorKinds...),
			OrchestratorMayComplete:   childRule.OrchestratorMayComplete,
			RequiredForParentDone:     childRule.RequiredForParentDone,
			RequiredForContainingDone: childRule.RequiredForContainingDone,
		}, s.clock())
		if err != nil {
			return err
		}
		if err := s.repo.CreateNodeContractSnapshot(ctx, snapshot); err != nil {
			return err
		}
	}
	return nil
}

// validateTemplateChildRulesWithLibrary preflights nested child rules against one explicit library.
func (s *Service) validateTemplateChildRulesWithLibrary(ctx context.Context, projectID string, library domain.TemplateLibrary, childRules []domain.TemplateChildRule, parent *domain.ActionItem, depth int) error {
	if depth > maxKindTemplateApplyDepth {
		return fmt.Errorf("%w: template application depth exceeded", domain.ErrInvalidTemplateLibrary)
	}
	for _, childRule := range childRules {
		childKind, err := s.resolveActionItemKindDefinition(ctx, projectID, childRule.ChildKindID, childRule.ChildScopeLevel, parent)
		if err != nil {
			return err
		}
		childParent := &domain.ActionItem{
			ProjectID: projectID,
			Scope:     childRule.ChildScopeLevel,
		}
		if childTemplate, found := library.FindNodeTemplate(childRule.ChildScopeLevel, childKind.ID); found {
			if err := s.validateTemplateChildRulesWithLibrary(ctx, projectID, library, childTemplate.ChildRules, childParent, depth+1); err != nil {
				return err
			}
			continue
		}
		if err := s.validateKindTemplateExpansion(ctx, projectID, childKind, childParent, domain.KindAppliesToSubtask, depth+1); err != nil {
			return err
		}
	}
	return nil
}

// validateTemplateChildRules preflights nested bound child rules before persistence.
func (s *Service) validateTemplateChildRules(ctx context.Context, projectID string, childRules []domain.TemplateChildRule, parent *domain.ActionItem, depth int) error {
	if depth > maxKindTemplateApplyDepth {
		return fmt.Errorf("%w: template application depth exceeded", domain.ErrInvalidTemplateLibrary)
	}
	for _, childRule := range childRules {
		childKind, err := s.resolveActionItemKindDefinition(ctx, projectID, childRule.ChildKindID, childRule.ChildScopeLevel, parent)
		if err != nil {
			return err
		}
		childParent := &domain.ActionItem{
			ProjectID: projectID,
			Scope:     childRule.ChildScopeLevel,
		}
		_, childTemplate, found, err := s.resolveBoundNodeTemplate(ctx, projectID, childRule.ChildScopeLevel, childKind.ID)
		if err != nil {
			return err
		}
		if found {
			if err := s.validateTemplateChildRules(ctx, projectID, childTemplate.ChildRules, childParent, depth+1); err != nil {
				return err
			}
			continue
		}
		if err := s.validateKindTemplateExpansion(ctx, projectID, childKind, childParent, domain.KindAppliesToSubtask, depth+1); err != nil {
			return err
		}
	}
	return nil
}

// firstActorType returns the first non-empty actor type from the candidate list.
func firstActorType(values ...domain.ActorType) domain.ActorType {
	for _, value := range values {
		if normalized := normalizeActorTypeInput(value); normalized != "" {
			return normalized
		}
	}
	return ""
}
