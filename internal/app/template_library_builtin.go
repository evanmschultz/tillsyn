package app

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/evanmschultz/tillsyn/internal/domain"
)

const (
	defaultGoBuiltinLibraryID       = "default-go"
	defaultFrontendBuiltinLibraryID = "default-frontend"
)

// EnsureBuiltinTemplateLibraryInput stores one explicit builtin install or refresh request.
type EnsureBuiltinTemplateLibraryInput struct {
	LibraryID string
	ActorID   string
	ActorName string
	ActorType domain.ActorType
}

// GetBuiltinTemplateLibraryStatus returns install and drift state for one supported builtin library.
func (s *Service) GetBuiltinTemplateLibraryStatus(ctx context.Context, libraryID string) (domain.BuiltinTemplateLibraryStatus, error) {
	spec, err := builtinTemplateLibrarySpec(strings.TrimSpace(libraryID), builtinTemplateActor{})
	if err != nil {
		return domain.BuiltinTemplateLibraryStatus{}, err
	}
	status := domain.BuiltinTemplateLibraryStatus{
		LibraryID:             spec.ID,
		Name:                  spec.Name,
		BuiltinSource:         spec.BuiltinSource,
		BuiltinVersion:        spec.BuiltinVersion,
		RequiredKindIDs:       builtinTemplateRequiredKinds(spec),
		State:                 domain.BuiltinTemplateLibraryStateMissing,
		BuiltinRevisionDigest: builtinTemplateRevisionDigest(spec),
	}
	status.MissingKindIDs, err = s.builtinTemplateMissingKinds(ctx, status.RequiredKindIDs)
	if err != nil {
		return domain.BuiltinTemplateLibraryStatus{}, err
	}

	library, err := s.repo.GetTemplateLibrary(ctx, spec.ID)
	switch {
	case errors.Is(err, ErrNotFound):
		return status, nil
	case err != nil:
		return domain.BuiltinTemplateLibraryStatus{}, err
	}

	status.Installed = true
	status.InstalledLibraryName = firstNonEmptyTrimmed(library.Name, library.ID)
	status.InstalledStatus = library.Status
	status.InstalledRevision = max(library.Revision, 1)
	status.InstalledDigest = strings.TrimSpace(library.RevisionDigest)
	status.InstalledBuiltin = library.BuiltinManaged
	if !library.UpdatedAt.IsZero() {
		ts := library.UpdatedAt.UTC()
		status.InstalledUpdatedAt = &ts
	}

	if status.InstalledDigest == status.BuiltinRevisionDigest {
		status.State = domain.BuiltinTemplateLibraryStateCurrent
	} else {
		status.State = domain.BuiltinTemplateLibraryStateUpdateAvailable
	}
	return status, nil
}

// EnsureBuiltinTemplateLibrary installs or refreshes one supported builtin library explicitly.
func (s *Service) EnsureBuiltinTemplateLibrary(ctx context.Context, in EnsureBuiltinTemplateLibraryInput) (domain.BuiltinTemplateLibraryEnsureResult, error) {
	actor := builtinTemplateActor{
		ID:   strings.TrimSpace(in.ActorID),
		Name: strings.TrimSpace(in.ActorName),
		Type: normalizeActorTypeInput(in.ActorType),
	}
	spec, err := builtinTemplateLibrarySpec(strings.TrimSpace(in.LibraryID), actor)
	if err != nil {
		return domain.BuiltinTemplateLibraryEnsureResult{}, err
	}
	requiredKinds := builtinTemplateRequiredKinds(spec)
	missingKinds, err := s.builtinTemplateMissingKinds(ctx, requiredKinds)
	if err != nil {
		return domain.BuiltinTemplateLibraryEnsureResult{}, err
	}
	if len(missingKinds) > 0 {
		values := make([]string, 0, len(missingKinds))
		for _, kindID := range missingKinds {
			values = append(values, string(kindID))
		}
		return domain.BuiltinTemplateLibraryEnsureResult{}, fmt.Errorf("%w: builtin template %q requires kind definitions [%s] in the active runtime DB; call get_builtin_status first, confirm you are on the intended stable or dev runtime, and bootstrap the missing kinds before ensure_builtin", domain.ErrBuiltinTemplateBootstrapRequired, spec.ID, strings.Join(values, ", "))
	}

	changed := true
	existing, err := s.repo.GetTemplateLibrary(ctx, spec.ID)
	switch {
	case err == nil:
		changed = existing.RevisionFingerprint() != builtinTemplateRevisionDigest(spec)
	case errors.Is(err, ErrNotFound):
	default:
		return domain.BuiltinTemplateLibraryEnsureResult{}, err
	}

	library, err := s.UpsertTemplateLibrary(ctx, spec)
	if err != nil {
		return domain.BuiltinTemplateLibraryEnsureResult{}, err
	}
	status, err := s.GetBuiltinTemplateLibraryStatus(ctx, spec.ID)
	if err != nil {
		return domain.BuiltinTemplateLibraryEnsureResult{}, err
	}
	return domain.BuiltinTemplateLibraryEnsureResult{
		Library: library,
		Status:  status,
		Changed: changed,
	}, nil
}

// builtinTemplateActor stores audit identity used for explicit builtin library ensure operations.
type builtinTemplateActor struct {
	ID   string
	Name string
	Type domain.ActorType
}

// builtinTemplateLibrarySpec returns the supported builtin library spec as one ordinary upsert input.
func builtinTemplateLibrarySpec(libraryID string, actor builtinTemplateActor) (UpsertTemplateLibraryInput, error) {
	switch domain.NormalizeTemplateLibraryID(libraryID) {
	case "", defaultGoBuiltinLibraryID:
		return defaultGoBuiltinTemplateLibrarySpec(actor)
	case defaultFrontendBuiltinLibraryID:
		return defaultFrontendBuiltinTemplateLibrarySpec(actor)
	default:
		return UpsertTemplateLibraryInput{}, fmt.Errorf("%w: unsupported builtin template library %q", domain.ErrInvalidTemplateLibrary, strings.TrimSpace(libraryID))
	}
}

// builtinTemplateRequiredKinds returns every kind referenced by one builtin template spec.
func builtinTemplateRequiredKinds(spec UpsertTemplateLibraryInput) []domain.KindID {
	seen := map[domain.KindID]struct{}{}
	kinds := make([]domain.KindID, 0)
	for _, nodeTemplate := range spec.NodeTemplates {
		if _, ok := seen[nodeTemplate.NodeKindID]; !ok && nodeTemplate.NodeKindID != "" {
			seen[nodeTemplate.NodeKindID] = struct{}{}
			kinds = append(kinds, nodeTemplate.NodeKindID)
		}
		for _, childRule := range nodeTemplate.ChildRules {
			if _, ok := seen[childRule.ChildKindID]; !ok && childRule.ChildKindID != "" {
				seen[childRule.ChildKindID] = struct{}{}
				kinds = append(kinds, childRule.ChildKindID)
			}
		}
	}
	slices.Sort(kinds)
	return kinds
}

// builtinTemplateRevisionDigest normalizes one builtin spec into the same logical revision digest used for stored libraries.
func builtinTemplateRevisionDigest(spec UpsertTemplateLibraryInput) string {
	library, err := domain.NewTemplateLibrary(domain.TemplateLibraryInput{
		ID:                  spec.ID,
		Scope:               spec.Scope,
		ProjectID:           spec.ProjectID,
		Name:                spec.Name,
		Description:         spec.Description,
		Status:              spec.Status,
		SourceLibraryID:     spec.SourceLibraryID,
		BuiltinManaged:      spec.BuiltinManaged,
		BuiltinSource:       spec.BuiltinSource,
		BuiltinVersion:      spec.BuiltinVersion,
		CreatedByActorID:    firstNonEmptyTrimmed(spec.CreatedByActorID, templateSystemActorID),
		CreatedByActorName:  firstNonEmptyTrimmed(spec.CreatedByActorName, templateSystemActorName),
		CreatedByActorType:  firstActorType(spec.CreatedByActorType, domain.ActorTypeSystem),
		ApprovedByActorID:   firstNonEmptyTrimmed(spec.ApprovedByActorID, templateSystemActorID),
		ApprovedByActorName: firstNonEmptyTrimmed(spec.ApprovedByActorName, templateSystemActorName),
		ApprovedByActorType: firstActorType(spec.ApprovedByActorType, domain.ActorTypeSystem),
		NodeTemplates:       builtinTemplateNodeInputs(spec.NodeTemplates),
	}, time.Now().UTC())
	if err != nil {
		return ""
	}
	return library.RevisionFingerprint()
}

// builtinTemplateNodeInputs converts builtin upsert node templates into domain constructor inputs for digest calculation.
func builtinTemplateNodeInputs(nodeTemplates []UpsertNodeTemplateInput) []domain.NodeTemplateInput {
	out := make([]domain.NodeTemplateInput, 0, len(nodeTemplates))
	for _, nodeTemplate := range nodeTemplates {
		childRules := make([]domain.TemplateChildRuleInput, 0, len(nodeTemplate.ChildRules))
		for _, childRule := range nodeTemplate.ChildRules {
			childRules = append(childRules, domain.TemplateChildRuleInput{
				ID:                        childRule.ID,
				Position:                  childRule.Position,
				ChildScopeLevel:           childRule.ChildScopeLevel,
				ChildKindID:               childRule.ChildKindID,
				TitleTemplate:             childRule.TitleTemplate,
				DescriptionTemplate:       childRule.DescriptionTemplate,
				ResponsibleActorKind:      childRule.ResponsibleActorKind,
				EditableByActorKinds:      append([]domain.TemplateActorKind(nil), childRule.EditableByActorKinds...),
				CompletableByActorKinds:   append([]domain.TemplateActorKind(nil), childRule.CompletableByActorKinds...),
				OrchestratorMayComplete:   childRule.OrchestratorMayComplete,
				RequiredForParentDone:     childRule.RequiredForParentDone,
				RequiredForContainingDone: childRule.RequiredForContainingDone,
			})
		}
		out = append(out, domain.NodeTemplateInput{
			ID:                         nodeTemplate.ID,
			ScopeLevel:                 nodeTemplate.ScopeLevel,
			NodeKindID:                 nodeTemplate.NodeKindID,
			DisplayName:                nodeTemplate.DisplayName,
			DescriptionMarkdown:        nodeTemplate.DescriptionMarkdown,
			ProjectMetadataDefaults:    nodeTemplate.ProjectMetadataDefaults,
			ActionItemMetadataDefaults: nodeTemplate.ActionItemMetadataDefaults,
			ChildRules:                 childRules,
		})
	}
	return out
}

// builtinTemplateMissingKinds reports which required kinds are not currently defined.
func (s *Service) builtinTemplateMissingKinds(ctx context.Context, requiredKinds []domain.KindID) ([]domain.KindID, error) {
	missing := make([]domain.KindID, 0)
	for _, kindID := range requiredKinds {
		if _, err := s.repo.GetKindDefinition(ctx, kindID); err != nil {
			if errors.Is(err, ErrNotFound) {
				missing = append(missing, kindID)
				continue
			}
			return nil, err
		}
	}
	return missing, nil
}

// defaultGoBuiltinTemplateLibrarySpec loads the builtin default-go library contract from the repo-visible embedded source.
func defaultGoBuiltinTemplateLibrarySpec(actor builtinTemplateActor) (UpsertTemplateLibraryInput, error) {
	doc, err := loadBuiltinTemplateLibraryDocument(defaultGoBuiltinLibraryID)
	if err != nil {
		return UpsertTemplateLibraryInput{}, err
	}
	return builtinTemplateLibraryInputFromDocument(doc, actor), nil
}

// defaultFrontendBuiltinTemplateLibrarySpec loads the builtin default-frontend library contract from the repo-visible embedded source.
func defaultFrontendBuiltinTemplateLibrarySpec(actor builtinTemplateActor) (UpsertTemplateLibraryInput, error) {
	doc, err := loadBuiltinTemplateLibraryDocument(defaultFrontendBuiltinLibraryID)
	if err != nil {
		return UpsertTemplateLibraryInput{}, err
	}
	return builtinTemplateLibraryInputFromDocument(doc, actor), nil
}
