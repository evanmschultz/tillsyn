package domain

import (
	"bytes"
	"encoding/json"
	"strings"
	"time"
)

// Project represents project data used by this package.
type Project struct {
	ID          string
	Slug        string
	Name        string
	Description string
	Metadata    ProjectMetadata
	// KindCatalogJSON is the lazy-decode envelope for the per-project
	// KindCatalog snapshot baked from the project's bound Template at
	// creation time. Per Drop 3 fix L5 (CE4 import-cycle resolution) this
	// field is a json.RawMessage rather than a typed catalog value: the
	// concrete KindCatalog type lives in internal/templates, and a typed
	// field here would re-introduce the forbidden
	// internal/domain → internal/templates dependency.
	//
	// Decoding lives in internal/app or internal/templates — never on
	// Project's methods. Callers consult this field via templates.KindCatalog
	// JSON-decoding helpers; an empty / nil value signals "no template was
	// bound at create time" and triggers the legacy repo fallback path per
	// droplet 3.12 acceptance criterion (preserves Drop 2.8 universal-
	// nesting boot compatibility).
	//
	// Per Drop 3 finding 5.B.14 (runtime mutability): edits to a project's
	// <project_root>/.tillsyn/template.toml AFTER project creation are
	// ignored until the dev fresh-DBs ~/.tillsyn/tillsyn.db. The catalog
	// is baked once and frozen for the project's lifetime.
	KindCatalogJSON json.RawMessage `json:"kind_catalog_json,omitempty"`
	CreatedAt       time.Time
	UpdatedAt       time.Time
	ArchivedAt      *time.Time
}

// ProjectCapabilityPolicy stores project-level capability and override policy values.
type ProjectCapabilityPolicy struct {
	AllowOrchestratorOverride bool   `json:"allow_orchestrator_override"`
	OrchestratorOverrideToken string `json:"orchestrator_override_token"`
	AllowEqualScopeDelegation bool   `json:"allow_equal_scope_delegation"`
}

// ProjectMetadata represents project metadata data used by this package.
type ProjectMetadata struct {
	Owner             string                  `json:"owner"`
	Icon              string                  `json:"icon"`
	Color             string                  `json:"color"`
	Homepage          string                  `json:"homepage"`
	Tags              []string                `json:"tags"`
	StandardsMarkdown string                  `json:"standards_markdown"`
	KindPayload       json.RawMessage         `json:"kind_payload,omitempty"`
	CapabilityPolicy  ProjectCapabilityPolicy `json:"capability_policy"`
}

// NewProject constructs a new value for this package.
func NewProject(id, name, description string, now time.Time) (Project, error) {
	id = strings.TrimSpace(id)
	name = strings.TrimSpace(name)
	if id == "" {
		return Project{}, ErrInvalidID
	}
	if name == "" {
		return Project{}, ErrInvalidName
	}

	slug := normalizeSlug(name)

	return Project{
		ID:          id,
		Slug:        slug,
		Name:        name,
		Description: strings.TrimSpace(description),
		Metadata:    ProjectMetadata{},
		CreatedAt:   now.UTC(),
		UpdatedAt:   now.UTC(),
	}, nil
}

// Rename renames the requested operation.
func (p *Project) Rename(name string, now time.Time) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return ErrInvalidName
	}
	p.Name = name
	p.Slug = normalizeSlug(name)
	p.UpdatedAt = now.UTC()
	return nil
}

// UpdateDetails updates state for the requested operation.
func (p *Project) UpdateDetails(name, description string, metadata ProjectMetadata, now time.Time) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return ErrInvalidName
	}
	p.Name = name
	p.Slug = normalizeSlug(name)
	p.Description = strings.TrimSpace(description)
	normalized, err := normalizeProjectMetadata(metadata)
	if err != nil {
		return err
	}
	p.Metadata = normalized
	p.UpdatedAt = now.UTC()
	return nil
}

// MergeProjectMetadata applies optional defaults to project metadata without weakening explicit values.
func MergeProjectMetadata(base ProjectMetadata, defaults *ProjectMetadata) (ProjectMetadata, error) {
	normalizedBase, err := normalizeProjectMetadata(base)
	if err != nil {
		return ProjectMetadata{}, err
	}
	if defaults == nil {
		return normalizedBase, nil
	}

	normalizedDefaults, err := normalizeProjectMetadata(*defaults)
	if err != nil {
		return ProjectMetadata{}, err
	}

	merged := normalizedBase
	if merged.Owner == "" {
		merged.Owner = normalizedDefaults.Owner
	}
	if merged.Icon == "" {
		merged.Icon = normalizedDefaults.Icon
	}
	if merged.Color == "" {
		merged.Color = normalizedDefaults.Color
	}
	if merged.Homepage == "" {
		merged.Homepage = normalizedDefaults.Homepage
	}
	merged.Tags = mergeStringLists(merged.Tags, normalizedDefaults.Tags)
	if len(merged.StandardsMarkdown) == 0 {
		merged.StandardsMarkdown = normalizedDefaults.StandardsMarkdown
	}
	mergedPayload, err := mergeKindPayloadDefaults(merged.KindPayload, normalizedDefaults.KindPayload)
	if err != nil {
		return ProjectMetadata{}, err
	}
	merged.KindPayload = mergedPayload
	// Capability-policy defaults are intentionally not auto-merged here.
	// The current bool-only shape cannot distinguish omitted vs explicit false,
	// so automatically widening override/delegation policy would violate the
	// "explicit user values win" rule until later tri-state policy surfaces land.

	return normalizeProjectMetadata(merged)
}

// Archive archives the requested operation.
func (p *Project) Archive(now time.Time) {
	ts := now.UTC()
	p.ArchivedAt = &ts
	p.UpdatedAt = ts
}

// Restore restores the requested operation.
func (p *Project) Restore(now time.Time) {
	p.ArchivedAt = nil
	p.UpdatedAt = now.UTC()
}

// normalizeSlug normalizes slug.
func normalizeSlug(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" {
		return ""
	}

	var b strings.Builder
	prevDash := false
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
			prevDash = false
		case r >= '0' && r <= '9':
			b.WriteRune(r)
			prevDash = false
		default:
			if !prevDash {
				b.WriteByte('-')
				prevDash = true
			}
		}
	}
	out := strings.Trim(b.String(), "-")
	return out
}

// normalizeProjectMetadata normalizes project metadata.
func normalizeProjectMetadata(meta ProjectMetadata) (ProjectMetadata, error) {
	meta.Owner = strings.TrimSpace(meta.Owner)
	meta.Icon = strings.TrimSpace(meta.Icon)
	meta.Color = strings.TrimSpace(meta.Color)
	meta.Homepage = strings.TrimSpace(meta.Homepage)
	meta.Tags = normalizeLabels(meta.Tags)
	meta.StandardsMarkdown = strings.TrimSpace(meta.StandardsMarkdown)
	meta.KindPayload = bytes.TrimSpace(meta.KindPayload)
	if len(meta.KindPayload) > 0 && !json.Valid(meta.KindPayload) {
		return ProjectMetadata{}, ErrInvalidKindPayload
	}
	meta.CapabilityPolicy.OrchestratorOverrideToken = strings.TrimSpace(meta.CapabilityPolicy.OrchestratorOverrideToken)
	return meta, nil
}
