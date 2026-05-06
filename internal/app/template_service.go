// Drop 4c.5 droplet F.3.1: read-only template inspection service.
//
// `till.template` MCP tool's `get` and `list_builtin` operations route
// through the methods on this file. Mutating operations (`validate`, `set`)
// land in F.3.2 + F.3.3.
//
// The service-level methods sit on *app.Service so the existing
// AppServiceAdapter dispatch (capture_state-or-attention picks the same
// concrete object) wires through with no plumbing changes.

package app

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/evanmschultz/tillsyn/internal/domain"
	"github.com/evanmschultz/tillsyn/internal/templates"
)

// TemplateBakeSourceLanguage names the embedded-default bake-source token
// for each closed Project.Language enum value. The MCP envelope uses the
// closed-vocabulary string so wire-side adopters can route on the value
// without re-querying the project's Language axis.
//
// Closed enum (mirrors templates.LoadDefaultTemplateForLanguage):
//
//   - "" (generic) → "embedded-default-generic"
//   - "go"         → "embedded-default-go"
//
// Future languages (`"fe"` and beyond) extend this map alongside the
// builtin TOML and the resolver — same drift contract as
// templates.LoadDefaultTemplateForLanguage.
const (
	templateBakeSourceEmbeddedGeneric = "embedded-default-generic"
	templateBakeSourceEmbeddedGo      = "embedded-default-go"
	templateBakeSourceBareRoot        = "<bare-root>"
	templateBakeSourcePrimaryWorktree = "<primary-worktree>"
)

// GetProjectTemplateInput carries the trimmed project identifier.
type GetProjectTemplateInput struct {
	ProjectID string
}

// GetProjectTemplateOutput carries the active per-project Template plus the
// bake-source provenance string.
type GetProjectTemplateOutput struct {
	ProjectID  string
	BakeSource string
	Template   templates.Template
}

// ListBuiltinTemplatesOutput carries the closed list of embedded builtin
// template names.
type ListBuiltinTemplatesOutput struct {
	Templates []string
}

// GetProjectTemplate resolves the active Template + bake-source provenance
// for one project. Read-only — does NOT mutate project state, does NOT
// re-bake the KindCatalog snapshot, does NOT re-walk the filesystem
// candidates.
//
// Drop 4c.5 droplet F.3.1 acceptance criterion #2: the operation requires
// a non-empty ProjectID and returns the JSON-decoded `KindCatalogJSON` from
// the project plus the bake-source provenance string. We re-resolve the
// Template from the same walk that bakeProjectKindCatalog uses at create
// time so the MCP client sees TOML body bytes (not the post-bake catalog
// snapshot), matching the F.3.1 spec wire-format choice.
//
// Snapshot-policy note: per Drop 3 finding 5.B.14 the KindCatalogJSON is
// the create-time bake; the Template returned here is the LIVE walk
// result, which CAN diverge from the catalog if the dev edited
// `<bare-root>/.tillsyn/template.toml` after project create. F.3.1's
// doc-string names this. F.3.3's `set` operation is the supported path
// for installing edits.
func (s *Service) GetProjectTemplate(ctx context.Context, in GetProjectTemplateInput) (GetProjectTemplateOutput, error) {
	if s == nil || s.repo == nil {
		return GetProjectTemplateOutput{}, fmt.Errorf("service is not configured")
	}
	projectID := strings.TrimSpace(in.ProjectID)
	if projectID == "" {
		return GetProjectTemplateOutput{}, domain.ErrInvalidID
	}
	project, err := s.repo.GetProject(ctx, projectID)
	if err != nil {
		return GetProjectTemplateOutput{}, err
	}
	tpl, source, err := resolveProjectTemplateWithSource(&project)
	if err != nil {
		return GetProjectTemplateOutput{}, err
	}
	return GetProjectTemplateOutput{
		ProjectID:  project.ID,
		BakeSource: source,
		Template:   tpl,
	}, nil
}

// ListBuiltinTemplates returns the closed list of embedded builtin template
// names that LoadDefaultTemplateForLanguage can resolve. Read-only,
// project-context-free, deterministic across processes.
//
// Drop 4c.5 droplet F.3.1 acceptance criterion #3: returns
// `["default-generic", "default-go"]` post-F.2.
func (s *Service) ListBuiltinTemplates(_ context.Context) (ListBuiltinTemplatesOutput, error) {
	return ListBuiltinTemplatesOutput{
		Templates: templates.BuiltinTemplateNames(),
	}, nil
}

// resolveProjectTemplateWithSource performs the same walk as
// loadProjectTemplate but additionally reports the bake-source provenance
// string the F.3.1 wire envelope expects.
//
// The walk order is bare-root → primary-worktree → embedded-default-by-
// language; on a successful Load at any step, the corresponding source
// token is returned. Errors propagate identically to loadProjectTemplate
// (file-not-exist on a candidate falls through; any other error is wrapped
// with the offending path and surfaced).
//
// Pre-MVP no caller other than GetProjectTemplate consumes the source
// token, so the helper lives next to the wire-shape it serves rather than
// in service.go. If a second consumer materializes (e.g. F.3.3's `set` op
// reporting the prior bake source for diagnostics), the helper can move.
func resolveProjectTemplateWithSource(project *domain.Project) (templates.Template, string, error) {
	if project == nil {
		return templates.Template{}, "", fmt.Errorf("nil project")
	}
	bareRoot := strings.TrimSpace(project.RepoBareRoot)
	primaryWorktree := strings.TrimSpace(project.RepoPrimaryWorktree)
	type candidate struct {
		path   string
		source string
	}
	candidates := make([]candidate, 0, 2)
	if bareRoot != "" {
		candidates = append(candidates, candidate{
			path:   filepath.Join(bareRoot, projectTemplateDir, projectTemplateFilename),
			source: templateBakeSourceBareRoot,
		})
	}
	if primaryWorktree != "" {
		candidates = append(candidates, candidate{
			path:   filepath.Join(primaryWorktree, projectTemplateDir, projectTemplateFilename),
			source: templateBakeSourcePrimaryWorktree,
		})
	}
	for _, cand := range candidates {
		tpl, ok, err := loadProjectTemplateCandidate(cand.path)
		if err != nil {
			return templates.Template{}, "", err
		}
		if ok {
			return tpl, cand.source, nil
		}
	}
	tpl, err := templates.LoadDefaultTemplateForLanguage(project.Language)
	if err != nil {
		return templates.Template{}, "", fmt.Errorf("load embedded default template for language %q: %w", project.Language, err)
	}
	return tpl, embeddedSourceForLanguage(project.Language), nil
}

// embeddedSourceForLanguage maps the project Language axis to the
// closed-enum bake-source token the F.3.1 wire envelope reports for an
// embedded-fallback resolution.
//
// Drift contract: kept in lockstep with
// templates.LoadDefaultTemplateForLanguage. A future drop that adds a
// language MUST extend both this map and the resolver. The default branch
// returns the empty string, which the MCP envelope surfaces verbatim — a
// loud "we resolved an embedded template but cannot name it" signal that
// triggers the closed-enum drift guard at the dev surface.
func embeddedSourceForLanguage(lang string) string {
	switch lang {
	case "":
		return templateBakeSourceEmbeddedGeneric
	case "go":
		return templateBakeSourceEmbeddedGo
	default:
		return ""
	}
}
