package domain

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"strings"
	"time"
)

// Project represents project data used by this package.
type Project struct {
	ID          string
	Slug        string
	Name        string
	Description string
	// HyllaArtifactRef is the project-scoped Hylla ingest reference
	// (e.g. "github.com/evanmschultz/tillsyn@main"). Free-form trimmed
	// string — Hylla resolves the ref at ingest time, so URL-shape parsing
	// is intentionally NOT performed here. Empty string is the meaningful
	// zero value (project not yet wired to a Hylla artifact). First-class
	// per Drop 4a L4 / WAVE_1_PLAN.md §1.8. Wave 2 dispatcher reads this
	// when constructing the agent-spawn invocation so subagents know which
	// Hylla artifact to query.
	HyllaArtifactRef string
	// RepoBareRoot is the absolute filesystem path to the project's bare
	// git repository (orchestration root, e.g.
	// "/Users/.../hylla/tillsyn/"). Validated as absolute via
	// filepath.IsAbs — relative paths reject with ErrInvalidRepoPath.
	// Empty string is the meaningful zero value (project not yet
	// bootstrapped to a checkout layout). First-class per Drop 4a L4 /
	// WAVE_1_PLAN.md §1.8. Wave 2 dispatcher reads this for
	// orchestration-root operations (multi-worktree coordination).
	RepoBareRoot string
	// RepoPrimaryWorktree is the absolute filesystem path to the project's
	// primary worktree (e.g. "/Users/.../hylla/tillsyn/main/"). Validated
	// as absolute via filepath.IsAbs — relative paths reject with
	// ErrInvalidRepoPath. Empty string is the meaningful zero value.
	// First-class per Drop 4a L4 / WAVE_1_PLAN.md §1.8. Wave 2 dispatcher
	// reads this as `cd` target when spawning subagents (Dir on
	// *exec.Cmd).
	RepoPrimaryWorktree string
	// Language carries the project's primary language axis. Closed enum:
	// "" | "go" | "fe". Empty is permitted for un-typed projects
	// pre-bootstrap. Non-empty values outside the enum reject with
	// ErrInvalidLanguage. First-class per Drop 4a L4 / WAVE_1_PLAN.md
	// §1.8. Wave 2 dispatcher reads this to pick the language-specific
	// agent variant (go-builder-agent vs fe-builder-agent).
	Language string
	// BuildTool carries the project's build-driver name (e.g. "mage" |
	// "npm" | "yarn" | "pnpm"). Free-form trimmed string — no closed enum
	// (build tools proliferate). Empty string is the meaningful zero
	// value. First-class per Drop 4a L4 / WAVE_1_PLAN.md §1.8. Wave 2
	// dispatcher reads this to select the verification target ("mage ci"
	// vs "npm test").
	BuildTool string
	// DevMcpServerName is the per-project `claude mcp add` registration
	// name for the dev MCP server (per CONTRIBUTING.md §"Dev MCP Server
	// Setup"). Each worktree gets a unique MCP entry pointing at its own
	// built binary; this field carries the worktree-specific name. Free-
	// form trimmed string. First-class per Drop 4a L4 / WAVE_1_PLAN.md
	// §1.8. Wave 2 dispatcher reads this when constructing the
	// `--mcp-config` plumbing for spawned subagents.
	DevMcpServerName string
	Metadata         ProjectMetadata
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

// ProjectInput holds input values for project constructor operations.
//
// Mirrors the ActionItemInput precedent (action_item.go:177+). Every Drop 4a
// L4 first-class project field travels through this struct on create.
// Validation runs in NewProjectFromInput before the Project value is
// assembled. Fields are documented inline; see Project struct for round-
// trip semantics.
type ProjectInput struct {
	ID                  string
	Name                string
	Description         string
	HyllaArtifactRef    string
	RepoBareRoot        string
	RepoPrimaryWorktree string
	Language            string
	BuildTool           string
	DevMcpServerName    string
}

// NewProjectFromInput constructs a new project from a populated
// ProjectInput, applying all Drop 4a L4 validations. Fields are
// trimmed; Language is checked against the closed enum
// ("" | "go" | "fe") with ErrInvalidLanguage; RepoBareRoot and
// RepoPrimaryWorktree must be absolute paths (or empty) with
// ErrInvalidRepoPath; HyllaArtifactRef, BuildTool, and DevMcpServerName
// are free-form trimmed strings.
func NewProjectFromInput(in ProjectInput, now time.Time) (Project, error) {
	id := strings.TrimSpace(in.ID)
	name := strings.TrimSpace(in.Name)
	if id == "" {
		return Project{}, ErrInvalidID
	}
	if name == "" {
		return Project{}, ErrInvalidName
	}

	hyllaArtifactRef := strings.TrimSpace(in.HyllaArtifactRef)
	repoBareRoot := strings.TrimSpace(in.RepoBareRoot)
	repoPrimaryWorktree := strings.TrimSpace(in.RepoPrimaryWorktree)
	language := strings.TrimSpace(in.Language)
	buildTool := strings.TrimSpace(in.BuildTool)
	devMcpServerName := strings.TrimSpace(in.DevMcpServerName)

	if !isValidProjectLanguage(language) {
		return Project{}, ErrInvalidLanguage
	}
	if repoBareRoot != "" && !filepath.IsAbs(repoBareRoot) {
		return Project{}, ErrInvalidRepoPath
	}
	if repoPrimaryWorktree != "" && !filepath.IsAbs(repoPrimaryWorktree) {
		return Project{}, ErrInvalidRepoPath
	}

	slug := normalizeSlug(name)

	return Project{
		ID:                  id,
		Slug:                slug,
		Name:                name,
		Description:         strings.TrimSpace(in.Description),
		HyllaArtifactRef:    hyllaArtifactRef,
		RepoBareRoot:        repoBareRoot,
		RepoPrimaryWorktree: repoPrimaryWorktree,
		Language:            language,
		BuildTool:           buildTool,
		DevMcpServerName:    devMcpServerName,
		Metadata:            ProjectMetadata{},
		CreatedAt:           now.UTC(),
		UpdatedAt:           now.UTC(),
	}, nil
}

// isValidProjectLanguage reports whether the supplied language string is a
// member of the closed Drop 4a L4 enum: "" | "go" | "fe".
func isValidProjectLanguage(lang string) bool {
	switch lang {
	case "", "go", "fe":
		return true
	default:
		return false
	}
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
//
// The six Drop 4a L4 first-class fields (HyllaArtifactRef, RepoBareRoot,
// RepoPrimaryWorktree, Language, BuildTool, DevMcpServerName) are
// value-typed — not pointer-sentineled. Per WAVE_1_PLAN.md §1.8, the
// Project surface is admin-driven (not agent-driven), so explicit-empty
// intent is rare and the simpler value-shape is preferred. Validation
// matches NewProjectFromInput: Language against the closed enum,
// RepoBareRoot + RepoPrimaryWorktree as absolute paths (when non-empty).
func (p *Project) UpdateDetails(
	name, description string,
	hyllaArtifactRef, repoBareRoot, repoPrimaryWorktree, language, buildTool, devMcpServerName string,
	metadata ProjectMetadata,
	now time.Time,
) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return ErrInvalidName
	}

	hyllaArtifactRef = strings.TrimSpace(hyllaArtifactRef)
	repoBareRoot = strings.TrimSpace(repoBareRoot)
	repoPrimaryWorktree = strings.TrimSpace(repoPrimaryWorktree)
	language = strings.TrimSpace(language)
	buildTool = strings.TrimSpace(buildTool)
	devMcpServerName = strings.TrimSpace(devMcpServerName)

	if !isValidProjectLanguage(language) {
		return ErrInvalidLanguage
	}
	if repoBareRoot != "" && !filepath.IsAbs(repoBareRoot) {
		return ErrInvalidRepoPath
	}
	if repoPrimaryWorktree != "" && !filepath.IsAbs(repoPrimaryWorktree) {
		return ErrInvalidRepoPath
	}

	p.Name = name
	p.Slug = normalizeSlug(name)
	p.Description = strings.TrimSpace(description)
	p.HyllaArtifactRef = hyllaArtifactRef
	p.RepoBareRoot = repoBareRoot
	p.RepoPrimaryWorktree = repoPrimaryWorktree
	p.Language = language
	p.BuildTool = buildTool
	p.DevMcpServerName = devMcpServerName
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
