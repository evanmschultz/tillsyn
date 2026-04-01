package domain

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"slices"
	"sort"
	"strings"
	"time"
)

// TemplateLibraryScope identifies the namespace of a template library.
type TemplateLibraryScope string

// TemplateLibraryScope values identify the supported template-library namespaces.
const (
	TemplateLibraryScopeGlobal  TemplateLibraryScope = "global"
	TemplateLibraryScopeProject TemplateLibraryScope = "project"
	TemplateLibraryScopeDraft   TemplateLibraryScope = "draft"
)

// TemplateLibraryStatus identifies the lifecycle state of a template library.
type TemplateLibraryStatus string

// TemplateLibraryStatus values identify the supported template-library states.
const (
	TemplateLibraryStatusDraft    TemplateLibraryStatus = "draft"
	TemplateLibraryStatusApproved TemplateLibraryStatus = "approved"
	TemplateLibraryStatusArchived TemplateLibraryStatus = "archived"
)

// ProjectTemplateBindingDrift values describe whether one project binding is current with the latest library revision.
const (
	ProjectTemplateBindingDriftCurrent         = "current"
	ProjectTemplateBindingDriftUpdateAvailable = "update_available"
	ProjectTemplateBindingDriftLibraryMissing  = "library_missing"
)

// TemplateActorKind identifies the workflow actor kind referenced by template rules.
type TemplateActorKind string

// TemplateActorKind values identify the fixed MVP workflow actor kinds.
const (
	TemplateActorKindHuman        TemplateActorKind = "human"
	TemplateActorKindOrchestrator TemplateActorKind = "orchestrator"
	TemplateActorKindBuilder      TemplateActorKind = "builder"
	TemplateActorKindQA           TemplateActorKind = "qa"
	TemplateActorKindResearch     TemplateActorKind = "research"
)

// TemplateLibrary stores one approved or draft template-library definition.
type TemplateLibrary struct {
	ID                  string                `json:"id"`
	Scope               TemplateLibraryScope  `json:"scope"`
	ProjectID           string                `json:"project_id,omitempty"`
	Name                string                `json:"name"`
	Description         string                `json:"description,omitempty"`
	Status              TemplateLibraryStatus `json:"status"`
	SourceLibraryID     string                `json:"source_library_id,omitempty"`
	BuiltinManaged      bool                  `json:"builtin_managed,omitempty"`
	BuiltinSource       string                `json:"builtin_source,omitempty"`
	BuiltinVersion      string                `json:"builtin_version,omitempty"`
	Revision            int                   `json:"revision"`
	RevisionDigest      string                `json:"revision_digest,omitempty"`
	CreatedByActorID    string                `json:"created_by_actor_id,omitempty"`
	CreatedByActorName  string                `json:"created_by_actor_name,omitempty"`
	CreatedByActorType  ActorType             `json:"created_by_actor_type,omitempty"`
	CreatedAt           time.Time             `json:"created_at"`
	UpdatedAt           time.Time             `json:"updated_at"`
	ApprovedByActorID   string                `json:"approved_by_actor_id,omitempty"`
	ApprovedByActorName string                `json:"approved_by_actor_name,omitempty"`
	ApprovedByActorType ActorType             `json:"approved_by_actor_type,omitempty"`
	ApprovedAt          *time.Time            `json:"approved_at,omitempty"`
	NodeTemplates       []NodeTemplate        `json:"node_templates"`
}

// NodeTemplate stores one node-template rule for one scope level and node kind.
type NodeTemplate struct {
	ID                      string              `json:"id"`
	LibraryID               string              `json:"library_id"`
	ScopeLevel              KindAppliesTo       `json:"scope_level"`
	NodeKindID              KindID              `json:"node_kind_id"`
	DisplayName             string              `json:"display_name"`
	DescriptionMarkdown     string              `json:"description_markdown,omitempty"`
	ProjectMetadataDefaults *ProjectMetadata    `json:"project_metadata_defaults,omitempty"`
	TaskMetadataDefaults    *TaskMetadata       `json:"task_metadata_defaults,omitempty"`
	ChildRules              []TemplateChildRule `json:"child_rules"`
}

// TemplateChildRule stores one auto-generated child rule under one node template.
type TemplateChildRule struct {
	ID                        string              `json:"id"`
	NodeTemplateID            string              `json:"node_template_id"`
	Position                  int                 `json:"position"`
	ChildScopeLevel           KindAppliesTo       `json:"child_scope_level"`
	ChildKindID               KindID              `json:"child_kind_id"`
	TitleTemplate             string              `json:"title_template"`
	DescriptionTemplate       string              `json:"description_template,omitempty"`
	ResponsibleActorKind      TemplateActorKind   `json:"responsible_actor_kind"`
	EditableByActorKinds      []TemplateActorKind `json:"editable_by_actor_kinds"`
	CompletableByActorKinds   []TemplateActorKind `json:"completable_by_actor_kinds"`
	OrchestratorMayComplete   bool                `json:"orchestrator_may_complete,omitempty"`
	RequiredForParentDone     bool                `json:"required_for_parent_done,omitempty"`
	RequiredForContainingDone bool                `json:"required_for_containing_done,omitempty"`
}

// ProjectTemplateBinding stores the active template-library binding for one project.
type ProjectTemplateBinding struct {
	ProjectID              string           `json:"project_id"`
	LibraryID              string           `json:"library_id"`
	LibraryName            string           `json:"library_name,omitempty"`
	BoundRevision          int              `json:"bound_revision"`
	BoundRevisionDigest    string           `json:"bound_revision_digest,omitempty"`
	BoundLibraryUpdatedAt  time.Time        `json:"bound_library_updated_at"`
	DriftStatus            string           `json:"drift_status,omitempty"`
	LatestRevision         int              `json:"latest_revision,omitempty"`
	LatestRevisionDigest   string           `json:"latest_revision_digest,omitempty"`
	LatestLibraryUpdatedAt *time.Time       `json:"latest_library_updated_at,omitempty"`
	BoundByActorID         string           `json:"bound_by_actor_id,omitempty"`
	BoundByActorName       string           `json:"bound_by_actor_name,omitempty"`
	BoundByActorType       ActorType        `json:"bound_by_actor_type,omitempty"`
	BoundAt                time.Time        `json:"bound_at"`
	BoundLibrarySnapshot   *TemplateLibrary `json:"bound_library_snapshot,omitempty"`
}

// TemplateLibraryFilter stores listing criteria for template-library queries.
type TemplateLibraryFilter struct {
	Scope     TemplateLibraryScope
	ProjectID string
	Status    TemplateLibraryStatus
}

// NodeContractSnapshot stores one persisted generated-node contract snapshot.
type NodeContractSnapshot struct {
	NodeID                    string              `json:"node_id"`
	ProjectID                 string              `json:"project_id"`
	SourceLibraryID           string              `json:"source_library_id"`
	SourceNodeTemplateID      string              `json:"source_node_template_id"`
	SourceChildRuleID         string              `json:"source_child_rule_id"`
	CreatedByActorID          string              `json:"created_by_actor_id,omitempty"`
	CreatedByActorType        ActorType           `json:"created_by_actor_type,omitempty"`
	ResponsibleActorKind      TemplateActorKind   `json:"responsible_actor_kind"`
	EditableByActorKinds      []TemplateActorKind `json:"editable_by_actor_kinds"`
	CompletableByActorKinds   []TemplateActorKind `json:"completable_by_actor_kinds"`
	OrchestratorMayComplete   bool                `json:"orchestrator_may_complete,omitempty"`
	RequiredForParentDone     bool                `json:"required_for_parent_done,omitempty"`
	RequiredForContainingDone bool                `json:"required_for_containing_done,omitempty"`
	CreatedAt                 time.Time           `json:"created_at"`
}

// TemplateLibraryInput stores write-time values for constructing one template library.
type TemplateLibraryInput struct {
	ID                  string
	Scope               TemplateLibraryScope
	ProjectID           string
	Name                string
	Description         string
	Status              TemplateLibraryStatus
	SourceLibraryID     string
	BuiltinManaged      bool
	BuiltinSource       string
	BuiltinVersion      string
	Revision            int
	RevisionDigest      string
	CreatedByActorID    string
	CreatedByActorName  string
	CreatedByActorType  ActorType
	ApprovedByActorID   string
	ApprovedByActorName string
	ApprovedByActorType ActorType
	ApprovedAt          *time.Time
	NodeTemplates       []NodeTemplateInput
}

// NodeTemplateInput stores write-time values for constructing one node template.
type NodeTemplateInput struct {
	ID                      string
	ScopeLevel              KindAppliesTo
	NodeKindID              KindID
	DisplayName             string
	DescriptionMarkdown     string
	ProjectMetadataDefaults *ProjectMetadata
	TaskMetadataDefaults    *TaskMetadata
	ChildRules              []TemplateChildRuleInput
}

// TemplateChildRuleInput stores write-time values for constructing one child rule.
type TemplateChildRuleInput struct {
	ID                        string
	Position                  int
	ChildScopeLevel           KindAppliesTo
	ChildKindID               KindID
	TitleTemplate             string
	DescriptionTemplate       string
	ResponsibleActorKind      TemplateActorKind
	EditableByActorKinds      []TemplateActorKind
	CompletableByActorKinds   []TemplateActorKind
	OrchestratorMayComplete   bool
	RequiredForParentDone     bool
	RequiredForContainingDone bool
}

// ProjectTemplateBindingInput stores write-time values for constructing one project binding.
type ProjectTemplateBindingInput struct {
	ProjectID             string
	LibraryID             string
	LibraryName           string
	BoundRevision         int
	BoundRevisionDigest   string
	BoundLibraryUpdatedAt *time.Time
	BoundByActorID        string
	BoundByActorName      string
	BoundByActorType      ActorType
	BoundLibrarySnapshot  *TemplateLibrary
}

// NodeContractSnapshotInput stores write-time values for constructing one node-contract snapshot.
type NodeContractSnapshotInput struct {
	NodeID                    string
	ProjectID                 string
	SourceLibraryID           string
	SourceNodeTemplateID      string
	SourceChildRuleID         string
	CreatedByActorID          string
	CreatedByActorType        ActorType
	ResponsibleActorKind      TemplateActorKind
	EditableByActorKinds      []TemplateActorKind
	CompletableByActorKinds   []TemplateActorKind
	OrchestratorMayComplete   bool
	RequiredForParentDone     bool
	RequiredForContainingDone bool
}

var validTemplateLibraryScopes = []TemplateLibraryScope{
	TemplateLibraryScopeGlobal,
	TemplateLibraryScopeProject,
	TemplateLibraryScopeDraft,
}

var validTemplateLibraryStatuses = []TemplateLibraryStatus{
	TemplateLibraryStatusDraft,
	TemplateLibraryStatusApproved,
	TemplateLibraryStatusArchived,
}

var validTemplateActorKinds = []TemplateActorKind{
	TemplateActorKindHuman,
	TemplateActorKindOrchestrator,
	TemplateActorKindBuilder,
	TemplateActorKindQA,
	TemplateActorKindResearch,
}

// NormalizeTemplateLibraryID canonicalizes template-library and nested template identifiers.
func NormalizeTemplateLibraryID(id string) string {
	return strings.TrimSpace(strings.ToLower(id))
}

// NormalizeTemplateLibraryScope canonicalizes template-library scope values.
func NormalizeTemplateLibraryScope(scope TemplateLibraryScope) TemplateLibraryScope {
	return TemplateLibraryScope(strings.TrimSpace(strings.ToLower(string(scope))))
}

// NormalizeTemplateLibraryStatus canonicalizes template-library status values.
func NormalizeTemplateLibraryStatus(status TemplateLibraryStatus) TemplateLibraryStatus {
	return TemplateLibraryStatus(strings.TrimSpace(strings.ToLower(string(status))))
}

// NormalizeTemplateActorKind canonicalizes workflow actor-kind values used by templates.
func NormalizeTemplateActorKind(kind TemplateActorKind) TemplateActorKind {
	return TemplateActorKind(strings.TrimSpace(strings.ToLower(string(kind))))
}

// IsValidTemplateLibraryScope reports whether a template-library scope is supported.
func IsValidTemplateLibraryScope(scope TemplateLibraryScope) bool {
	scope = NormalizeTemplateLibraryScope(scope)
	return slices.Contains(validTemplateLibraryScopes, scope)
}

// IsValidTemplateLibraryStatus reports whether a template-library status is supported.
func IsValidTemplateLibraryStatus(status TemplateLibraryStatus) bool {
	status = NormalizeTemplateLibraryStatus(status)
	return slices.Contains(validTemplateLibraryStatuses, status)
}

// IsValidTemplateActorKind reports whether a workflow actor kind is supported in MVP.
func IsValidTemplateActorKind(kind TemplateActorKind) bool {
	kind = NormalizeTemplateActorKind(kind)
	return slices.Contains(validTemplateActorKinds, kind)
}

// NewTemplateLibrary validates and normalizes one template library and all nested rules.
func NewTemplateLibrary(in TemplateLibraryInput, now time.Time) (TemplateLibrary, error) {
	in.ID = NormalizeTemplateLibraryID(in.ID)
	if in.ID == "" {
		return TemplateLibrary{}, ErrInvalidID
	}
	in.Scope = NormalizeTemplateLibraryScope(in.Scope)
	if !IsValidTemplateLibraryScope(in.Scope) {
		return TemplateLibrary{}, ErrInvalidTemplateLibraryScope
	}
	in.Status = NormalizeTemplateLibraryStatus(in.Status)
	if !IsValidTemplateLibraryStatus(in.Status) {
		return TemplateLibrary{}, ErrInvalidTemplateStatus
	}
	projectID := strings.TrimSpace(in.ProjectID)
	if in.Scope != TemplateLibraryScopeGlobal && projectID == "" {
		return TemplateLibrary{}, fmt.Errorf("%w: project scope requires project_id", ErrInvalidTemplateLibrary)
	}
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return TemplateLibrary{}, ErrInvalidName
	}
	createdType := normalizeTemplateActorType(in.CreatedByActorType)
	if createdType == "" {
		createdType = ActorTypeUser
	}
	approvedType := normalizeTemplateActorType(in.ApprovedByActorType)
	if approvedType == "" && strings.TrimSpace(in.ApprovedByActorID) != "" {
		approvedType = ActorTypeUser
	}
	ts := now.UTC()
	out := TemplateLibrary{
		ID:                  in.ID,
		Scope:               in.Scope,
		ProjectID:           projectID,
		Name:                name,
		Description:         strings.TrimSpace(in.Description),
		Status:              in.Status,
		SourceLibraryID:     NormalizeTemplateLibraryID(in.SourceLibraryID),
		BuiltinManaged:      in.BuiltinManaged,
		BuiltinSource:       strings.TrimSpace(in.BuiltinSource),
		BuiltinVersion:      strings.TrimSpace(in.BuiltinVersion),
		Revision:            max(in.Revision, 1),
		RevisionDigest:      strings.TrimSpace(in.RevisionDigest),
		CreatedByActorID:    strings.TrimSpace(in.CreatedByActorID),
		CreatedByActorName:  strings.TrimSpace(in.CreatedByActorName),
		CreatedByActorType:  createdType,
		CreatedAt:           ts,
		UpdatedAt:           ts,
		ApprovedByActorID:   strings.TrimSpace(in.ApprovedByActorID),
		ApprovedByActorName: strings.TrimSpace(in.ApprovedByActorName),
		ApprovedByActorType: approvedType,
		ApprovedAt:          normalizeTemplateNullableTS(in.ApprovedAt),
		NodeTemplates:       make([]NodeTemplate, 0, len(in.NodeTemplates)),
	}
	seenTemplates := map[string]struct{}{}
	for _, nodeTemplateIn := range in.NodeTemplates {
		nodeTemplate, err := newNodeTemplate(in.ID, nodeTemplateIn)
		if err != nil {
			return TemplateLibrary{}, err
		}
		key := string(nodeTemplate.ScopeLevel) + ":" + string(nodeTemplate.NodeKindID)
		if _, exists := seenTemplates[key]; exists {
			return TemplateLibrary{}, fmt.Errorf("%w: duplicate node template %s", ErrInvalidTemplateLibrary, key)
		}
		seenTemplates[key] = struct{}{}
		out.NodeTemplates = append(out.NodeTemplates, nodeTemplate)
	}
	sort.Slice(out.NodeTemplates, func(i, j int) bool {
		if out.NodeTemplates[i].ScopeLevel == out.NodeTemplates[j].ScopeLevel {
			if out.NodeTemplates[i].NodeKindID == out.NodeTemplates[j].NodeKindID {
				return out.NodeTemplates[i].ID < out.NodeTemplates[j].ID
			}
			return out.NodeTemplates[i].NodeKindID < out.NodeTemplates[j].NodeKindID
		}
		return out.NodeTemplates[i].ScopeLevel < out.NodeTemplates[j].ScopeLevel
	})
	return out, nil
}

// NewProjectTemplateBinding validates and normalizes one project binding.
func NewProjectTemplateBinding(in ProjectTemplateBindingInput, now time.Time) (ProjectTemplateBinding, error) {
	projectID := strings.TrimSpace(in.ProjectID)
	if projectID == "" {
		return ProjectTemplateBinding{}, ErrInvalidID
	}
	libraryID := NormalizeTemplateLibraryID(in.LibraryID)
	if libraryID == "" {
		return ProjectTemplateBinding{}, ErrInvalidTemplateBinding
	}
	actorType := normalizeTemplateActorType(in.BoundByActorType)
	if actorType == "" {
		actorType = ActorTypeUser
	}
	boundLibraryUpdatedAt := now.UTC()
	if in.BoundLibraryUpdatedAt != nil && !in.BoundLibraryUpdatedAt.IsZero() {
		boundLibraryUpdatedAt = in.BoundLibraryUpdatedAt.UTC()
	}
	return ProjectTemplateBinding{
		ProjectID:             projectID,
		LibraryID:             libraryID,
		LibraryName:           strings.TrimSpace(in.LibraryName),
		BoundRevision:         max(in.BoundRevision, 1),
		BoundRevisionDigest:   strings.TrimSpace(in.BoundRevisionDigest),
		BoundLibraryUpdatedAt: boundLibraryUpdatedAt,
		BoundByActorID:        strings.TrimSpace(in.BoundByActorID),
		BoundByActorName:      strings.TrimSpace(in.BoundByActorName),
		BoundByActorType:      actorType,
		BoundAt:               now.UTC(),
		BoundLibrarySnapshot:  cloneOptionalTemplateLibrary(in.BoundLibrarySnapshot),
	}, nil
}

// NewNodeContractSnapshot validates and normalizes one generated-node contract snapshot.
func NewNodeContractSnapshot(in NodeContractSnapshotInput, now time.Time) (NodeContractSnapshot, error) {
	nodeID := strings.TrimSpace(in.NodeID)
	if nodeID == "" {
		return NodeContractSnapshot{}, ErrInvalidID
	}
	projectID := strings.TrimSpace(in.ProjectID)
	if projectID == "" {
		return NodeContractSnapshot{}, ErrInvalidID
	}
	libraryID := NormalizeTemplateLibraryID(in.SourceLibraryID)
	if libraryID == "" {
		return NodeContractSnapshot{}, ErrInvalidTemplateLibrary
	}
	responsibleActorKind := NormalizeTemplateActorKind(in.ResponsibleActorKind)
	if !IsValidTemplateActorKind(responsibleActorKind) {
		return NodeContractSnapshot{}, ErrInvalidTemplateActorKind
	}
	actorType := normalizeTemplateActorType(in.CreatedByActorType)
	if actorType == "" {
		actorType = ActorTypeSystem
	}
	return NodeContractSnapshot{
		NodeID:                    nodeID,
		ProjectID:                 projectID,
		SourceLibraryID:           libraryID,
		SourceNodeTemplateID:      NormalizeTemplateLibraryID(in.SourceNodeTemplateID),
		SourceChildRuleID:         NormalizeTemplateLibraryID(in.SourceChildRuleID),
		CreatedByActorID:          strings.TrimSpace(in.CreatedByActorID),
		CreatedByActorType:        actorType,
		ResponsibleActorKind:      responsibleActorKind,
		EditableByActorKinds:      normalizeTemplateActorKinds(in.EditableByActorKinds, responsibleActorKind),
		CompletableByActorKinds:   normalizeTemplateActorKinds(in.CompletableByActorKinds, responsibleActorKind),
		OrchestratorMayComplete:   in.OrchestratorMayComplete,
		RequiredForParentDone:     in.RequiredForParentDone,
		RequiredForContainingDone: in.RequiredForContainingDone,
		CreatedAt:                 now.UTC(),
	}, nil
}

// FindNodeTemplate finds one node template by scope and node kind within the library.
func (l TemplateLibrary) FindNodeTemplate(scope KindAppliesTo, kindID KindID) (NodeTemplate, bool) {
	scope = NormalizeKindAppliesTo(scope)
	kindID = NormalizeKindID(kindID)
	for _, nodeTemplate := range l.NodeTemplates {
		if nodeTemplate.ScopeLevel == scope && nodeTemplate.NodeKindID == kindID {
			return nodeTemplate, true
		}
	}
	return NodeTemplate{}, false
}

// RevisionFingerprint returns one stable digest input for logical template-library revisions.
func (l TemplateLibrary) RevisionFingerprint() string {
	canonical := struct {
		ID              string                `json:"id"`
		Scope           TemplateLibraryScope  `json:"scope"`
		ProjectID       string                `json:"project_id,omitempty"`
		Name            string                `json:"name"`
		Description     string                `json:"description,omitempty"`
		Status          TemplateLibraryStatus `json:"status"`
		SourceLibraryID string                `json:"source_library_id,omitempty"`
		BuiltinManaged  bool                  `json:"builtin_managed,omitempty"`
		BuiltinSource   string                `json:"builtin_source,omitempty"`
		BuiltinVersion  string                `json:"builtin_version,omitempty"`
		NodeTemplates   []NodeTemplate        `json:"node_templates"`
	}{
		ID:              NormalizeTemplateLibraryID(l.ID),
		Scope:           NormalizeTemplateLibraryScope(l.Scope),
		ProjectID:       strings.TrimSpace(l.ProjectID),
		Name:            strings.TrimSpace(l.Name),
		Description:     strings.TrimSpace(l.Description),
		Status:          NormalizeTemplateLibraryStatus(l.Status),
		SourceLibraryID: NormalizeTemplateLibraryID(l.SourceLibraryID),
		BuiltinManaged:  l.BuiltinManaged,
		BuiltinSource:   strings.TrimSpace(l.BuiltinSource),
		BuiltinVersion:  strings.TrimSpace(l.BuiltinVersion),
		NodeTemplates:   cloneNodeTemplates(l.NodeTemplates),
	}
	raw, err := json.Marshal(canonical)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

// newNodeTemplate validates and normalizes one nested node template.
func newNodeTemplate(libraryID string, in NodeTemplateInput) (NodeTemplate, error) {
	in.ID = NormalizeTemplateLibraryID(in.ID)
	if in.ID == "" {
		return NodeTemplate{}, ErrInvalidID
	}
	in.ScopeLevel = NormalizeKindAppliesTo(in.ScopeLevel)
	if !IsValidKindAppliesTo(in.ScopeLevel) {
		return NodeTemplate{}, ErrInvalidKindAppliesTo
	}
	in.NodeKindID = NormalizeKindID(in.NodeKindID)
	if in.NodeKindID == "" {
		return NodeTemplate{}, ErrInvalidKindID
	}
	displayName := strings.TrimSpace(in.DisplayName)
	if displayName == "" {
		displayName = string(in.NodeKindID)
	}
	out := NodeTemplate{
		ID:                  in.ID,
		LibraryID:           NormalizeTemplateLibraryID(libraryID),
		ScopeLevel:          in.ScopeLevel,
		NodeKindID:          in.NodeKindID,
		DisplayName:         displayName,
		DescriptionMarkdown: strings.TrimSpace(in.DescriptionMarkdown),
		ChildRules:          make([]TemplateChildRule, 0, len(in.ChildRules)),
	}
	if in.ProjectMetadataDefaults != nil {
		normalized, err := normalizeProjectMetadata(*in.ProjectMetadataDefaults)
		if err != nil {
			return NodeTemplate{}, err
		}
		out.ProjectMetadataDefaults = &normalized
	}
	if in.TaskMetadataDefaults != nil {
		normalized, err := normalizeTaskMetadata(*in.TaskMetadataDefaults)
		if err != nil {
			return NodeTemplate{}, err
		}
		out.TaskMetadataDefaults = &normalized
	}
	seenRules := map[string]struct{}{}
	for _, childRuleIn := range in.ChildRules {
		childRule, err := newTemplateChildRule(in.ID, childRuleIn)
		if err != nil {
			return NodeTemplate{}, err
		}
		if _, exists := seenRules[childRule.ID]; exists {
			return NodeTemplate{}, fmt.Errorf("%w: duplicate child rule %q", ErrInvalidTemplateLibrary, childRule.ID)
		}
		seenRules[childRule.ID] = struct{}{}
		out.ChildRules = append(out.ChildRules, childRule)
	}
	sort.Slice(out.ChildRules, func(i, j int) bool {
		if out.ChildRules[i].Position == out.ChildRules[j].Position {
			return out.ChildRules[i].ID < out.ChildRules[j].ID
		}
		return out.ChildRules[i].Position < out.ChildRules[j].Position
	})
	return out, nil
}

// newTemplateChildRule validates and normalizes one nested child rule.
func newTemplateChildRule(nodeTemplateID string, in TemplateChildRuleInput) (TemplateChildRule, error) {
	in.ID = NormalizeTemplateLibraryID(in.ID)
	if in.ID == "" {
		return TemplateChildRule{}, ErrInvalidID
	}
	in.ChildScopeLevel = NormalizeKindAppliesTo(in.ChildScopeLevel)
	if !IsValidWorkItemAppliesTo(in.ChildScopeLevel) {
		return TemplateChildRule{}, ErrInvalidKindAppliesTo
	}
	in.ChildKindID = NormalizeKindID(in.ChildKindID)
	if in.ChildKindID == "" {
		return TemplateChildRule{}, ErrInvalidKindID
	}
	titleTemplate := strings.TrimSpace(in.TitleTemplate)
	if titleTemplate == "" {
		return TemplateChildRule{}, ErrInvalidTitle
	}
	responsibleActorKind := NormalizeTemplateActorKind(in.ResponsibleActorKind)
	if !IsValidTemplateActorKind(responsibleActorKind) {
		return TemplateChildRule{}, ErrInvalidTemplateActorKind
	}
	return TemplateChildRule{
		ID:                        in.ID,
		NodeTemplateID:            NormalizeTemplateLibraryID(nodeTemplateID),
		Position:                  max(in.Position, 0),
		ChildScopeLevel:           in.ChildScopeLevel,
		ChildKindID:               in.ChildKindID,
		TitleTemplate:             titleTemplate,
		DescriptionTemplate:       strings.TrimSpace(in.DescriptionTemplate),
		ResponsibleActorKind:      responsibleActorKind,
		EditableByActorKinds:      normalizeTemplateActorKinds(in.EditableByActorKinds, responsibleActorKind),
		CompletableByActorKinds:   normalizeTemplateActorKinds(in.CompletableByActorKinds, responsibleActorKind),
		OrchestratorMayComplete:   in.OrchestratorMayComplete,
		RequiredForParentDone:     in.RequiredForParentDone,
		RequiredForContainingDone: in.RequiredForContainingDone,
	}, nil
}

// normalizeTemplateActorKinds trims, validates, de-duplicates, and sorts actor-kind lists.
func normalizeTemplateActorKinds(in []TemplateActorKind, fallback TemplateActorKind) []TemplateActorKind {
	out := make([]TemplateActorKind, 0, len(in)+1)
	seen := map[TemplateActorKind]struct{}{}
	for _, raw := range in {
		kind := NormalizeTemplateActorKind(raw)
		if !IsValidTemplateActorKind(kind) {
			continue
		}
		if _, exists := seen[kind]; exists {
			continue
		}
		seen[kind] = struct{}{}
		out = append(out, kind)
	}
	fallback = NormalizeTemplateActorKind(fallback)
	if len(out) == 0 && IsValidTemplateActorKind(fallback) {
		out = append(out, fallback)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i] < out[j]
	})
	return out
}

func cloneTemplateLibrary(in TemplateLibrary) TemplateLibrary {
	out := in
	out.NodeTemplates = cloneNodeTemplates(in.NodeTemplates)
	out.ApprovedAt = normalizeTemplateNullableTS(in.ApprovedAt)
	if in.ApprovedAt != nil {
		ts := in.ApprovedAt.UTC()
		out.ApprovedAt = &ts
	}
	return out
}

func cloneOptionalTemplateLibrary(in *TemplateLibrary) *TemplateLibrary {
	if in == nil {
		return nil
	}
	cloned := cloneTemplateLibrary(*in)
	return &cloned
}

func cloneNodeTemplates(in []NodeTemplate) []NodeTemplate {
	if len(in) == 0 {
		return nil
	}
	out := make([]NodeTemplate, 0, len(in))
	for _, nodeTemplate := range in {
		copied := nodeTemplate
		if nodeTemplate.ProjectMetadataDefaults != nil {
			projectDefaults := *nodeTemplate.ProjectMetadataDefaults
			copied.ProjectMetadataDefaults = &projectDefaults
		}
		if nodeTemplate.TaskMetadataDefaults != nil {
			taskDefaults := *nodeTemplate.TaskMetadataDefaults
			copied.TaskMetadataDefaults = &taskDefaults
		}
		if len(nodeTemplate.ChildRules) > 0 {
			copied.ChildRules = make([]TemplateChildRule, 0, len(nodeTemplate.ChildRules))
			for _, childRule := range nodeTemplate.ChildRules {
				childCopy := childRule
				childCopy.EditableByActorKinds = append([]TemplateActorKind(nil), childRule.EditableByActorKinds...)
				childCopy.CompletableByActorKinds = append([]TemplateActorKind(nil), childRule.CompletableByActorKinds...)
				copied.ChildRules = append(copied.ChildRules, childCopy)
			}
		}
		out = append(out, copied)
	}
	return out
}

// normalizeTemplateActorType canonicalizes optional actor-type values stored on template records.
func normalizeTemplateActorType(actorType ActorType) ActorType {
	switch strings.TrimSpace(strings.ToLower(string(actorType))) {
	case string(ActorTypeUser):
		return ActorTypeUser
	case string(ActorTypeAgent):
		return ActorTypeAgent
	case string(ActorTypeSystem):
		return ActorTypeSystem
	default:
		return ""
	}
}

// normalizeTemplateNullableTS canonicalizes optional timestamps used by template records.
func normalizeTemplateNullableTS(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	ts := value.UTC().Truncate(time.Second)
	return &ts
}

// cloneProjectMetadata copies one optional project metadata value.
func cloneProjectMetadata(in *ProjectMetadata) *ProjectMetadata {
	if in == nil {
		return nil
	}
	cloned := *in
	cloned.Tags = append([]string(nil), in.Tags...)
	cloned.KindPayload = bytes.TrimSpace(append([]byte(nil), in.KindPayload...))
	return &cloned
}

// cloneTaskMetadata copies one optional task metadata value.
func cloneTaskMetadata(in *TaskMetadata) *TaskMetadata {
	if in == nil {
		return nil
	}
	cloned := *in
	cloned.CommandSnippets = append([]string(nil), in.CommandSnippets...)
	cloned.ExpectedOutputs = append([]string(nil), in.ExpectedOutputs...)
	cloned.DecisionLog = append([]string(nil), in.DecisionLog...)
	cloned.RelatedItems = append([]string(nil), in.RelatedItems...)
	cloned.DependsOn = append([]string(nil), in.DependsOn...)
	cloned.BlockedBy = append([]string(nil), in.BlockedBy...)
	cloned.ContextBlocks = append([]ContextBlock(nil), in.ContextBlocks...)
	cloned.ResourceRefs = append([]ResourceRef(nil), in.ResourceRefs...)
	cloned.KindPayload = bytes.TrimSpace(append([]byte(nil), in.KindPayload...))
	cloned.CompletionContract.StartCriteria = append([]ChecklistItem(nil), in.CompletionContract.StartCriteria...)
	cloned.CompletionContract.CompletionCriteria = append([]ChecklistItem(nil), in.CompletionContract.CompletionCriteria...)
	cloned.CompletionContract.CompletionChecklist = append([]ChecklistItem(nil), in.CompletionContract.CompletionChecklist...)
	cloned.CompletionContract.CompletionEvidence = append([]string(nil), in.CompletionContract.CompletionEvidence...)
	return &cloned
}
