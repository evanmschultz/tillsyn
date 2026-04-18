package domain

import (
	"bytes"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"
)

// KindID identifies one reusable kind definition in the global kind catalog.
type KindID string

// DefaultProjectKind defines the default project kind identifier.
const DefaultProjectKind KindID = "project"

// KindAppliesTo identifies the node types a kind can be used for.
type KindAppliesTo string

// KindAppliesTo values.
const (
	KindAppliesToProject    KindAppliesTo = "project"
	KindAppliesToBranch     KindAppliesTo = "branch"
	KindAppliesToPhase      KindAppliesTo = "phase"
	KindAppliesToActionItem KindAppliesTo = "actionItem"
	KindAppliesToSubtask    KindAppliesTo = "subtask"
)

// validKindAppliesTo stores all supported applies_to values.
var validKindAppliesTo = []KindAppliesTo{
	KindAppliesToProject,
	KindAppliesToBranch,
	KindAppliesToPhase,
	KindAppliesToActionItem,
	KindAppliesToSubtask,
}

// validWorkItemAppliesTo stores applies_to values valid for work-items.
var validWorkItemAppliesTo = []KindAppliesTo{
	KindAppliesToBranch,
	KindAppliesToPhase,
	KindAppliesToActionItem,
	KindAppliesToSubtask,
}

// KindTemplateChildSpec defines one child item auto-created by a kind template.
type KindTemplateChildSpec struct {
	Title           string          `json:"title"`
	Description     string          `json:"description"`
	Kind            KindID          `json:"kind"`
	AppliesTo       KindAppliesTo   `json:"applies_to"`
	Labels          []string        `json:"labels"`
	MetadataPayload json.RawMessage `json:"metadata_payload,omitempty"`
}

// KindTemplate stores template-driven system actions and default metadata for a kind definition.
type KindTemplate struct {
	AutoCreateChildren         []KindTemplateChildSpec `json:"auto_create_children"`
	CompletionChecklist        []ChecklistItem         `json:"completion_checklist"`
	ProjectMetadataDefaults    *ProjectMetadata        `json:"project_metadata_defaults,omitempty"`
	ActionItemMetadataDefaults *ActionItemMetadata     `json:"task_metadata_defaults,omitempty"`
}

// KindDefinition stores one reusable kind definition.
type KindDefinition struct {
	ID                  KindID
	DisplayName         string
	DescriptionMarkdown string
	AppliesTo           []KindAppliesTo
	AllowedParentScopes []KindAppliesTo
	PayloadSchemaJSON   string
	Template            KindTemplate
	CreatedAt           time.Time
	UpdatedAt           time.Time
	ArchivedAt          *time.Time
}

// KindDefinitionInput holds write-time values for creating/updating a kind definition.
type KindDefinitionInput struct {
	ID                  KindID
	DisplayName         string
	DescriptionMarkdown string
	AppliesTo           []KindAppliesTo
	AllowedParentScopes []KindAppliesTo
	PayloadSchemaJSON   string
	Template            KindTemplate
}

// NewKindDefinition validates and normalizes one kind definition.
func NewKindDefinition(in KindDefinitionInput, now time.Time) (KindDefinition, error) {
	in.ID = NormalizeKindID(in.ID)
	if in.ID == "" {
		return KindDefinition{}, ErrInvalidKindID
	}

	displayName := strings.TrimSpace(in.DisplayName)
	if displayName == "" {
		displayName = string(in.ID)
	}

	appliesTo, err := normalizeKindAppliesToList(in.AppliesTo)
	if err != nil {
		return KindDefinition{}, err
	}
	if len(appliesTo) == 0 {
		return KindDefinition{}, ErrInvalidKindAppliesTo
	}

	allowedParentScopes, err := normalizeKindParentScopes(in.AllowedParentScopes)
	if err != nil {
		return KindDefinition{}, err
	}

	schemaJSON := strings.TrimSpace(in.PayloadSchemaJSON)
	if schemaJSON != "" {
		if !json.Valid([]byte(schemaJSON)) {
			return KindDefinition{}, ErrInvalidKindPayloadSchema
		}
	}

	template, err := normalizeKindTemplate(in.Template)
	if err != nil {
		return KindDefinition{}, err
	}

	ts := now.UTC()
	return KindDefinition{
		ID:                  in.ID,
		DisplayName:         displayName,
		DescriptionMarkdown: strings.TrimSpace(in.DescriptionMarkdown),
		AppliesTo:           appliesTo,
		AllowedParentScopes: allowedParentScopes,
		PayloadSchemaJSON:   schemaJSON,
		Template:            template,
		CreatedAt:           ts,
		UpdatedAt:           ts,
	}, nil
}

// AppliesToScope reports whether the kind allows the given target scope.
func (k KindDefinition) AppliesToScope(scope KindAppliesTo) bool {
	scope = NormalizeKindAppliesTo(scope)
	for _, candidate := range k.AppliesTo {
		if NormalizeKindAppliesTo(candidate) == scope {
			return true
		}
	}
	return false
}

// AllowsParentScope reports whether the kind allows a parent in the given scope.
func (k KindDefinition) AllowsParentScope(scope KindAppliesTo) bool {
	scope = NormalizeKindAppliesTo(scope)
	if len(k.AllowedParentScopes) == 0 {
		return true
	}
	for _, candidate := range k.AllowedParentScopes {
		if NormalizeKindAppliesTo(candidate) == scope {
			return true
		}
	}
	return false
}

// NormalizeKindID canonicalizes kind identifiers for storage/lookup.
// The input is trimmed and lowercased, then any "actionitem" token
// (whole-word matched against `-` or `_` boundaries) is rewritten to
// the canonical "actionItem" camelCase spelling so kind ids like
// "actionItem" or "build-actionItem" survive the round-trip intact.
func NormalizeKindID(id KindID) KindID {
	trimmed := strings.TrimSpace(string(id))
	if trimmed == "" {
		return ""
	}
	return KindID(canonicalizeActionItemToken(strings.ToLower(trimmed)))
}

// NormalizeKindAppliesTo canonicalizes applies_to values. Inputs are matched
// case-insensitively against the supported set and returned in their
// canonical camelCase form (e.g. "actionItem"); unknown values are returned
// lowercased so callers can still detect invalid inputs.
func NormalizeKindAppliesTo(scope KindAppliesTo) KindAppliesTo {
	lowered := strings.TrimSpace(strings.ToLower(string(scope)))
	if lowered == "" {
		return ""
	}
	for _, candidate := range validKindAppliesTo {
		if strings.ToLower(string(candidate)) == lowered {
			return candidate
		}
	}
	return KindAppliesTo(lowered)
}

// IsValidKindAppliesTo reports whether a value is supported for catalog definitions.
func IsValidKindAppliesTo(scope KindAppliesTo) bool {
	scope = NormalizeKindAppliesTo(scope)
	return slices.Contains(validKindAppliesTo, scope)
}

// IsValidWorkItemAppliesTo reports whether a value is supported for work-item rows.
func IsValidWorkItemAppliesTo(scope KindAppliesTo) bool {
	scope = NormalizeKindAppliesTo(scope)
	return slices.Contains(validWorkItemAppliesTo, scope)
}

// normalizeKindAppliesToList trims, validates, and de-duplicates applies_to values.
func normalizeKindAppliesToList(in []KindAppliesTo) ([]KindAppliesTo, error) {
	out := make([]KindAppliesTo, 0, len(in))
	seen := map[KindAppliesTo]struct{}{}
	for _, raw := range in {
		scope := NormalizeKindAppliesTo(raw)
		if scope == "" {
			continue
		}
		if !IsValidKindAppliesTo(scope) {
			return nil, fmt.Errorf("%w: %q", ErrInvalidKindAppliesTo, scope)
		}
		if _, ok := seen[scope]; ok {
			continue
		}
		seen[scope] = struct{}{}
		out = append(out, scope)
	}
	return out, nil
}

// normalizeKindParentScopes trims, validates, and de-duplicates allowed parent scopes.
func normalizeKindParentScopes(in []KindAppliesTo) ([]KindAppliesTo, error) {
	out := make([]KindAppliesTo, 0, len(in))
	seen := map[KindAppliesTo]struct{}{}
	for _, raw := range in {
		scope := NormalizeKindAppliesTo(raw)
		if scope == "" {
			continue
		}
		if !IsValidWorkItemAppliesTo(scope) {
			return nil, fmt.Errorf("%w: parent scope %q", ErrInvalidKindAppliesTo, scope)
		}
		if _, ok := seen[scope]; ok {
			continue
		}
		seen[scope] = struct{}{}
		out = append(out, scope)
	}
	return out, nil
}

// normalizeKindTemplate validates template-driven behavior fields.
func normalizeKindTemplate(in KindTemplate) (KindTemplate, error) {
	children := make([]KindTemplateChildSpec, 0, len(in.AutoCreateChildren))
	for idx, child := range in.AutoCreateChildren {
		child.Title = strings.TrimSpace(child.Title)
		child.Description = strings.TrimSpace(child.Description)
		child.Kind = NormalizeKindID(child.Kind)
		child.AppliesTo = NormalizeKindAppliesTo(child.AppliesTo)
		child.Labels = normalizeLabels(child.Labels)
		child.MetadataPayload = bytes.TrimSpace(child.MetadataPayload)

		if child.Title == "" {
			return KindTemplate{}, fmt.Errorf("%w: template child %d title is required", ErrInvalidKindTemplate, idx)
		}
		if child.Kind == "" {
			return KindTemplate{}, fmt.Errorf("%w: template child %d kind is required", ErrInvalidKindTemplate, idx)
		}
		if child.AppliesTo == "" {
			child.AppliesTo = KindAppliesToSubtask
		}
		if !IsValidWorkItemAppliesTo(child.AppliesTo) {
			return KindTemplate{}, fmt.Errorf("%w: template child %d applies_to %q", ErrInvalidKindTemplate, idx, child.AppliesTo)
		}
		if len(child.MetadataPayload) > 0 && !json.Valid(child.MetadataPayload) {
			return KindTemplate{}, fmt.Errorf("%w: template child %d metadata payload", ErrInvalidKindTemplate, idx)
		}
		children = append(children, child)
	}

	checklist, err := normalizeChecklist(in.CompletionChecklist)
	if err != nil {
		return KindTemplate{}, fmt.Errorf("%w: %v", ErrInvalidKindTemplate, err)
	}

	var projectDefaults *ProjectMetadata
	if in.ProjectMetadataDefaults != nil {
		normalized, err := normalizeProjectMetadata(*in.ProjectMetadataDefaults)
		if err != nil {
			return KindTemplate{}, fmt.Errorf("%w: project metadata defaults: %v", ErrInvalidKindTemplate, err)
		}
		projectDefaults = &normalized
	}
	var actionItemDefaults *ActionItemMetadata
	if in.ActionItemMetadataDefaults != nil {
		normalized, err := normalizeActionItemMetadata(*in.ActionItemMetadataDefaults)
		if err != nil {
			return KindTemplate{}, fmt.Errorf("%w: actionItem metadata defaults: %v", ErrInvalidKindTemplate, err)
		}
		actionItemDefaults = &normalized
	}

	return KindTemplate{
		AutoCreateChildren:         children,
		CompletionChecklist:        checklist,
		ProjectMetadataDefaults:    projectDefaults,
		ActionItemMetadataDefaults: actionItemDefaults,
	}, nil
}
