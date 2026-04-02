package app

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	repoassets "github.com/hylla/tillsyn/templates"

	"github.com/hylla/tillsyn/internal/domain"
)

// builtinTemplateLibraryDocument stores one repo-authored builtin template library payload.
type builtinTemplateLibraryDocument struct {
	ID             string                        `json:"id"`
	Name           string                        `json:"name"`
	Description    string                        `json:"description"`
	BuiltinSource  string                        `json:"builtin_source"`
	BuiltinVersion string                        `json:"builtin_version"`
	NodeTemplates  []builtinNodeTemplateDocument `json:"node_templates"`
}

// builtinNodeTemplateDocument stores one repo-authored node template payload.
type builtinNodeTemplateDocument struct {
	ID                      string                     `json:"id"`
	ScopeLevel              domain.KindAppliesTo       `json:"scope_level"`
	NodeKindID              domain.KindID              `json:"node_kind_id"`
	DisplayName             string                     `json:"display_name"`
	DescriptionMarkdown     string                     `json:"description_markdown"`
	ProjectMetadataDefaults *domain.ProjectMetadata    `json:"project_metadata_defaults"`
	TaskMetadataDefaults    *domain.TaskMetadata       `json:"task_metadata_defaults"`
	ChildRules              []builtinChildRuleDocument `json:"child_rules"`
}

// builtinChildRuleDocument stores one repo-authored child rule payload.
type builtinChildRuleDocument struct {
	ID                        string                     `json:"id"`
	Position                  int                        `json:"position"`
	ChildScopeLevel           domain.KindAppliesTo       `json:"child_scope_level"`
	ChildKindID               domain.KindID              `json:"child_kind_id"`
	TitleTemplate             string                     `json:"title_template"`
	DescriptionTemplate       string                     `json:"description_template"`
	ResponsibleActorKind      domain.TemplateActorKind   `json:"responsible_actor_kind"`
	EditableByActorKinds      []domain.TemplateActorKind `json:"editable_by_actor_kinds"`
	CompletableByActorKinds   []domain.TemplateActorKind `json:"completable_by_actor_kinds"`
	OrchestratorMayComplete   bool                       `json:"orchestrator_may_complete"`
	RequiredForParentDone     bool                       `json:"required_for_parent_done"`
	RequiredForContainingDone bool                       `json:"required_for_containing_done"`
}

// loadBuiltinTemplateLibraryDocument reads one repo-visible embedded builtin template document.
func loadBuiltinTemplateLibraryDocument(libraryID string) (builtinTemplateLibraryDocument, error) {
	path := fmt.Sprintf("builtin/%s.json", domain.NormalizeTemplateLibraryID(libraryID))
	data, err := repoassets.ReadFile(path)
	if err != nil {
		return builtinTemplateLibraryDocument{}, fmt.Errorf("read builtin template source %q: %w", path, err)
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var doc builtinTemplateLibraryDocument
	if err := decoder.Decode(&doc); err != nil {
		return builtinTemplateLibraryDocument{}, fmt.Errorf("decode builtin template source %q: %w", path, err)
	}
	if strings.TrimSpace(doc.ID) == "" {
		return builtinTemplateLibraryDocument{}, fmt.Errorf("%w: builtin template source %q is missing id", domain.ErrInvalidTemplateLibrary, path)
	}
	if got := domain.NormalizeTemplateLibraryID(doc.ID); got != domain.NormalizeTemplateLibraryID(libraryID) {
		return builtinTemplateLibraryDocument{}, fmt.Errorf("%w: builtin template source %q has id %q, want %q", domain.ErrInvalidTemplateLibrary, path, doc.ID, libraryID)
	}
	return doc, nil
}

// builtinTemplateLibraryInputFromDocument converts one repo-authored builtin document into an ordinary upsert input.
func builtinTemplateLibraryInputFromDocument(doc builtinTemplateLibraryDocument, actor builtinTemplateActor) UpsertTemplateLibraryInput {
	actorID := firstNonEmptyTrimmed(actor.ID, templateSystemActorID)
	actorName := firstNonEmptyTrimmed(actor.Name, templateSystemActorName)
	actorType := firstActorType(actor.Type, domain.ActorTypeSystem)

	nodeTemplates := make([]UpsertNodeTemplateInput, 0, len(doc.NodeTemplates))
	for _, nodeTemplate := range doc.NodeTemplates {
		childRules := make([]UpsertTemplateChildRuleInput, 0, len(nodeTemplate.ChildRules))
		for _, childRule := range nodeTemplate.ChildRules {
			childRules = append(childRules, UpsertTemplateChildRuleInput{
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
		nodeTemplates = append(nodeTemplates, UpsertNodeTemplateInput{
			ID:                      nodeTemplate.ID,
			ScopeLevel:              nodeTemplate.ScopeLevel,
			NodeKindID:              nodeTemplate.NodeKindID,
			DisplayName:             nodeTemplate.DisplayName,
			DescriptionMarkdown:     nodeTemplate.DescriptionMarkdown,
			ProjectMetadataDefaults: cloneProjectMetadata(nodeTemplate.ProjectMetadataDefaults),
			TaskMetadataDefaults:    cloneTaskMetadata(nodeTemplate.TaskMetadataDefaults),
			ChildRules:              childRules,
		})
	}

	return UpsertTemplateLibraryInput{
		ID:                  doc.ID,
		Scope:               domain.TemplateLibraryScopeGlobal,
		Name:                doc.Name,
		Description:         doc.Description,
		Status:              domain.TemplateLibraryStatusApproved,
		BuiltinManaged:      true,
		BuiltinSource:       doc.BuiltinSource,
		BuiltinVersion:      doc.BuiltinVersion,
		CreatedByActorID:    actorID,
		CreatedByActorName:  actorName,
		CreatedByActorType:  actorType,
		ApprovedByActorID:   actorID,
		ApprovedByActorName: actorName,
		ApprovedByActorType: actorType,
		NodeTemplates:       nodeTemplates,
	}
}

// cloneProjectMetadata deep-copies optional project metadata defaults loaded from builtin template documents.
func cloneProjectMetadata(meta *domain.ProjectMetadata) *domain.ProjectMetadata {
	if meta == nil {
		return nil
	}
	cloned := *meta
	cloned.Tags = append([]string(nil), meta.Tags...)
	if len(meta.KindPayload) > 0 {
		cloned.KindPayload = append([]byte(nil), meta.KindPayload...)
	}
	return &cloned
}

// cloneTaskMetadata deep-copies optional task metadata defaults loaded from builtin template documents.
func cloneTaskMetadata(meta *domain.TaskMetadata) *domain.TaskMetadata {
	if meta == nil {
		return nil
	}
	cloned := *meta
	cloned.CommandSnippets = append([]string(nil), meta.CommandSnippets...)
	cloned.ExpectedOutputs = append([]string(nil), meta.ExpectedOutputs...)
	cloned.DecisionLog = append([]string(nil), meta.DecisionLog...)
	cloned.RelatedItems = append([]string(nil), meta.RelatedItems...)
	cloned.DependsOn = append([]string(nil), meta.DependsOn...)
	cloned.BlockedBy = append([]string(nil), meta.BlockedBy...)
	cloned.ContextBlocks = append([]domain.ContextBlock(nil), meta.ContextBlocks...)
	cloned.ResourceRefs = append([]domain.ResourceRef(nil), meta.ResourceRefs...)
	if len(meta.KindPayload) > 0 {
		cloned.KindPayload = append([]byte(nil), meta.KindPayload...)
	}
	return &cloned
}
