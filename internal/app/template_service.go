// Drop 4c.5 droplet F.3.1: read-only template inspection service.
//
// `till.template` MCP tool's `get` and `list_builtin` operations route
// through the methods on this file. Drop 4c.5 droplet F.3.2 added the
// purely-lexical `validate` operation alongside the read-only inspectors.
// Drop 4c.5 droplet F.3.3 added `SetProjectTemplate` (auth-gated atomic
// install + re-bake) — see the doc-comment on that method for the
// transactional ordering and rollback contract.
//
// The service-level methods sit on *app.Service so the existing
// AppServiceAdapter dispatch (capture_state-or-attention picks the same
// concrete object) wires through with no plumbing changes.

package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/evanmschultz/tillsyn/internal/domain"
	"github.com/evanmschultz/tillsyn/internal/templates"
)

// templateBakeSourceBareRoot and templateBakeSourcePrimaryWorktree are the
// bake-source provenance tokens the F.3.1 wire envelope reports for
// project-tier template resolutions.
const (
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
// names that LoadBuiltinTemplate can resolve. Read-only,
// project-context-free, deterministic across processes.
//
// Returns `["till-fe", "till-gen", "till-go"]` in stable lexical order
// (post-Drop-4c.6.1 W4.D2, which added till-fe alongside the till-gen +
// till-go pair rebadged in Drop 4c.6 W5).
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
	// No project-tier candidate found. Per REFINEMENTS.md 2026-05-14
	// "Remove project.Language; templates are project-tier opt-in only",
	// templates are project-tier opt-in only — no embedded language-default
	// fallback. Mirroring the bake-time loadProjectTemplate / loadProjectTemplatesForGroups
	// contract, return a zero Template + empty bake-source token to signal
	// "no template bound." The MCP `till.template get` wire shape carries
	// the empty result; callers interpret an empty BakeSource as "this
	// project has not authored a template."
	return templates.Template{}, "", nil
}

// ValidateCandidateTemplateInput carries the candidate TOML bytes the
// caller wants validated. The MCP boundary applies the 1MB input-size cap
// before invoking the service so adapter-side memory pressure stays
// bounded; pre-cap inputs reach this method in full.
//
// Drop 4c.5 droplet F.3.2.
type ValidateCandidateTemplateInput struct {
	TemplateTOML string
}

// ValidateCandidateTemplateOutput is the in-band validation report produced
// by ValidateCandidateTemplate. Validation failures are returned via
// Valid == false rather than as Go errors so the MCP wire envelope sees a
// uniform success/failure shape (`{"valid": ..., ...}`) regardless of which
// validator inside templates.LoadWithOptions tripped.
//
// SentinelName carries the canonical templates-package sentinel that
// errors.Is matched (e.g. "ErrUnknownTemplateKey", "ErrIncoherentStructuralType").
// When no sentinel matches but Load returned an error, SentinelName falls
// back to "validation_error" so callers can route on a stable non-empty
// string. Error carries the wrapped-error string verbatim, including
// position-aware detail when pelletier/go-toml/v2 supplied it.
//
// Warnings are F.5.1's `validateAgentBindingFiles` warn lines captured via
// LoadOptions.WarnLogger. They are independent of Valid: a successfully-
// loading template (Valid == true) MAY carry warnings, and a failing
// template (Valid == false) MAY also have collected warnings before the
// fatal-error validator fired.
//
// Drop 4c.5 droplet F.3.2.
type ValidateCandidateTemplateOutput struct {
	Valid        bool
	SentinelName string
	Error        string
	Warnings     []string
}

// templateValidationSentinels is the closed list of templates-package
// sentinels the validate operation routes on, in evaluation order. Each
// sentinel's Go variable name doubles as the canonical wire string: the
// MCP envelope returns `SentinelName: "ErrUnknownTemplateKey"` (and so on)
// so adopters can switch on the exact identifier without reaching into
// the templates package via cgo / vendor-tree introspection.
//
// Order matters because Errs.Is unwraps left-to-right and several
// templates-package sentinels are themselves wraps of
// ErrInvalidAgentBinding (env / context / tool_gating). The more-specific
// sentinels are checked first; if they all miss, ErrInvalidAgentBinding
// is the catch-all. Drift contract: a future templates-package sentinel
// MUST be added here AND its name must match the package-scope variable
// name verbatim, or the MCP wire shape silently regresses to
// "validation_error" for the new failure mode.
var templateValidationSentinels = []struct {
	name string
	err  error
}{
	{name: "ErrUnsupportedSchemaVersion", err: templates.ErrUnsupportedSchemaVersion},
	{name: "ErrUnknownTemplateKey", err: templates.ErrUnknownTemplateKey},
	{name: "ErrTemplateCycle", err: templates.ErrTemplateCycle},
	{name: "ErrMissingRequiredChildRule", err: templates.ErrMissingRequiredChildRule},
	{name: "ErrUnreachableChildRule", err: templates.ErrUnreachableChildRule},
	{name: "ErrIncoherentStructuralType", err: templates.ErrIncoherentStructuralType},
	{name: "ErrUnknownKindReference", err: templates.ErrUnknownKindReference},
	{name: "ErrUnknownGateKind", err: templates.ErrUnknownGateKind},
	{name: "ErrInvalidAgentBindingEnv", err: templates.ErrInvalidAgentBindingEnv},
	{name: "ErrInvalidContextRules", err: templates.ErrInvalidContextRules},
	{name: "ErrInvalidAgentBindingToolGating", err: templates.ErrInvalidAgentBindingToolGating},
	{name: "ErrInvalidAgentBinding", err: templates.ErrInvalidAgentBinding},
	{name: "ErrInvalidTillsynGlobals", err: templates.ErrInvalidTillsynGlobals},
}

// validationFallbackSentinelName is returned when Load returned an error
// but none of the closed templates-package sentinels matched via errors.Is.
// The non-empty fallback guarantees adopters always see a SentinelName on
// a Valid == false result so client-side switch statements can treat
// "" as "no error" without ambiguity.
const validationFallbackSentinelName = "validation_error"

// ValidateCandidateTemplate runs the full templates.LoadWithOptions
// validation chain on the supplied TOML bytes and reports the outcome
// in-band as a ValidateCandidateTemplateOutput.
//
// Drop 4c.5 droplet F.3.2 acceptance:
//
//   - Validation chain runs the FULL Load() pipeline (no skipping).
//   - Validate is purely lexical — does NOT touch project state, does NOT
//     re-bake, does NOT persist.
//   - Warnings produced by F.5.1's validateAgentBindingFiles (warn-only)
//     are surfaced via LoadOptions.WarnLogger captured into a slice.
//   - On success the result is `{Valid: true, Warnings: [...]}`.
//   - On failure the result is `{Valid: false, SentinelName: "<name>",
//     Error: "<wrapped message>"}` with SentinelName drawn from the
//     closed templateValidationSentinels list (or
//     validationFallbackSentinelName when no sentinel matches).
//
// Concurrency: validateAgentBindingFiles is invoked sequentially inside
// LoadWithOptions so the warn-logger closure does not need a mutex.
func (s *Service) ValidateCandidateTemplate(_ context.Context, in ValidateCandidateTemplateInput) (ValidateCandidateTemplateOutput, error) {
	warnings := make([]string, 0, 4)
	opts := templates.LoadOptions{
		WarnLogger: func(line string) {
			warnings = append(warnings, line)
		},
	}
	_, err := templates.LoadWithOptions(strings.NewReader(in.TemplateTOML), opts)
	if err != nil {
		return ValidateCandidateTemplateOutput{
			Valid:        false,
			SentinelName: classifyTemplateValidationError(err),
			Error:        err.Error(),
			Warnings:     warnings,
		}, nil
	}
	return ValidateCandidateTemplateOutput{
		Valid:    true,
		Warnings: warnings,
	}, nil
}

// classifyTemplateValidationError walks the closed templateValidationSentinels
// list in order and returns the canonical sentinel name for the first
// errors.Is match. Returns validationFallbackSentinelName when no sentinel
// matches — a non-empty string the MCP envelope surfaces so client-side
// routing always sees a stable value on Valid == false results.
//
// Pre-cond: err != nil. The caller is responsible for the nil-check
// because the success path on ValidateCandidateTemplate does not invoke
// this helper.
func classifyTemplateValidationError(err error) string {
	for _, sentinel := range templateValidationSentinels {
		if errors.Is(err, sentinel.err) {
			return sentinel.name
		}
	}
	return validationFallbackSentinelName
}

// SetProjectTemplateInput carries the inputs for the auth-gated atomic
// install operation. The MCP boundary already enforced the 1MB cap on
// TemplateTOML and ran authorizeMCPMutation before reaching this method;
// the service still re-validates as the first install step (see the
// SetProjectTemplate doc-comment for the full transactional order).
//
// Drop 4c.5 droplet F.3.3.
type SetProjectTemplateInput struct {
	ProjectID     string
	TemplateTOML  []byte
	UpdatedBy     string
	UpdatedByName string
	UpdatedType   domain.ActorType
}

// SetProjectTemplateOutput reports the outcome of an atomic install. The
// fields populate only on success; failure paths return an error rather
// than a zero output (the wire envelope at the adapter layer flips the
// shape into `{set: false, error: "..."}` for the MCP response). The
// closed BakeSource vocabulary mirrors GetProjectTemplate's: only
// "<bare-root>" or "<primary-worktree>" are reachable from `set`, since
// embedded fallbacks have no on-disk file to land in.
//
// Drop 4c.5 droplet F.3.3.
type SetProjectTemplateOutput struct {
	ProjectID    string
	BakeSource   string
	BytesWritten int
}

// ErrProjectHasNoCheckout is returned by SetProjectTemplate when both
// project.RepoBareRoot and project.RepoPrimaryWorktree are empty (after
// whitespace trim). The atomic install needs an on-disk write target;
// without a checkout layout there is nowhere to install. Pre-Drop-4a
// adopters create the checkout layout via till.project create or via the
// admin update path before invoking `set`.
//
// Drop 4c.5 droplet F.3.3 acceptance #3.
var ErrProjectHasNoCheckout = errors.New("project has no checkout — cannot install template; create the project's checkout layout first")

// SetProjectTemplate atomically installs a candidate template TOML into a
// project's bare-root or primary-worktree `.tillsyn/template.toml` file
// AND swaps the project's KindCatalogJSON snapshot to the freshly-baked
// catalog. The operation follows the four-step ordering mandated by Drop
// 4c.5 F.3.3 falsification mitigations F4 + F5:
//
//  1. Validate the candidate via templates.LoadWithOptions (full
//     validation chain, no skipping). Failure here aborts BEFORE any
//     filesystem or persistence side effect.
//  2. Re-bake the validated Template into a SHADOW
//     domain.Project.KindCatalogJSON in memory. Re-baking from the
//     in-memory parsed Template (NOT a fresh filesystem walk) guarantees
//     post-set bake equals post-set on-disk content — see F5 mitigation.
//  3. Atomic file install: write the candidate bytes to
//     `<dest>.tillsyn-set-<id>.tmp` then os.Rename to `<dest>`. POSIX
//     guarantees the rename is atomic for readers; concurrent `set`
//     operations serialize at the rename point (last-writer-wins).
//  4. Swap the shadow KindCatalogJSON onto the project record + persist
//     via repo.UpdateProject. If persist fails AFTER the rename has
//     already landed, the on-disk file is moved back to a sentinel
//     ".tillsyn-set-failed-<id>.toml" sibling so the dev sees the
//     orphaned artifact and can recover manually. The error wraps the
//     underlying repo error AND reports the rollback path.
//
// Snapshot policy (per Drop 3 finding 5.B.14): in-flight action items
// already carrying their own KindCatalog metadata continue to use the
// PRIOR catalog. NEW action items created after this method returns use
// the NEW catalog. The cascade does NOT auto-migrate in-flight items.
// Doc-comment names this loud so adopters know the mid-cascade boundary.
//
// Concurrency: this method takes no locks. Concurrent callers may both
// validate-and-write; the OS rename is the serialization point. The
// last-writer-wins; intermediate readers via GetProjectTemplate may
// observe either the prior or the new state but never a partial write
// (rename is atomic).
//
// Auth: this method assumes the MCP boundary already authorized the
// caller. The actor metadata threads through to the persisted project
// record so the audit trail records the correct identity. No additional
// mutation guard runs at the service layer (parity with
// SetProjectAllowedKinds). See registerTemplateTools' `set` op for the
// authorizeMCPMutation gate.
//
// Drop 4c.5 droplet F.3.3.
func (s *Service) SetProjectTemplate(ctx context.Context, in SetProjectTemplateInput) (SetProjectTemplateOutput, error) {
	if s == nil || s.repo == nil {
		return SetProjectTemplateOutput{}, fmt.Errorf("service is not configured")
	}
	projectID := strings.TrimSpace(in.ProjectID)
	if projectID == "" {
		return SetProjectTemplateOutput{}, domain.ErrInvalidID
	}
	if len(in.TemplateTOML) == 0 {
		return SetProjectTemplateOutput{}, fmt.Errorf("template_toml is empty")
	}
	project, err := s.repo.GetProject(ctx, projectID)
	if err != nil {
		return SetProjectTemplateOutput{}, err
	}
	bareRoot := strings.TrimSpace(project.RepoBareRoot)
	primaryWorktree := strings.TrimSpace(project.RepoPrimaryWorktree)
	// Acceptance #3: pick FIRST non-empty of (RepoBareRoot,
	// RepoPrimaryWorktree). Both empty → ErrProjectHasNoCheckout. The
	// closed sentinel lets adapter-layer mappers route deterministically.
	var (
		dest       string
		bakeSource string
	)
	switch {
	case bareRoot != "":
		dest = filepath.Join(bareRoot, projectTemplateDir, projectTemplateFilename)
		bakeSource = templateBakeSourceBareRoot
	case primaryWorktree != "":
		dest = filepath.Join(primaryWorktree, projectTemplateDir, projectTemplateFilename)
		bakeSource = templateBakeSourcePrimaryWorktree
	default:
		return SetProjectTemplateOutput{}, ErrProjectHasNoCheckout
	}
	// Step 1 — validate. Use LoadWithOptions so the chain matches the
	// `validate` MCP op exactly (warn-logger discarded here; the `set`
	// op surfaces only fatal errors, not warnings).
	tpl, err := templates.LoadWithOptions(strings.NewReader(string(in.TemplateTOML)), templates.LoadOptions{})
	if err != nil {
		return SetProjectTemplateOutput{}, fmt.Errorf("validate candidate template: %w", err)
	}
	// Step 2 — re-bake to shadow catalog. From the in-memory parsed
	// Template (NOT a fresh walk) so the post-set bake matches the
	// just-written file exactly.
	shadowCatalog := templates.Bake(tpl)
	shadowEncoded, err := json.Marshal(shadowCatalog)
	if err != nil {
		return SetProjectTemplateOutput{}, fmt.Errorf("encode shadow kind catalog: %w", err)
	}
	// Step 3 — atomic file install via tmp+rename.
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return SetProjectTemplateOutput{}, fmt.Errorf("ensure template directory %s: %w", filepath.Dir(dest), err)
	}
	tmpSuffix := strings.TrimSpace(s.idGen())
	if tmpSuffix == "" {
		tmpSuffix = fmt.Sprintf("set-%d", s.clock().UnixNano())
	}
	tmpPath := dest + ".tillsyn-set-" + tmpSuffix + ".tmp"
	if err := os.WriteFile(tmpPath, in.TemplateTOML, 0o644); err != nil {
		return SetProjectTemplateOutput{}, fmt.Errorf("write candidate template to %s: %w", tmpPath, err)
	}
	// Track tmp-file cleanup state. If we never reach Rename, remove the
	// tmp-file as best-effort; once Rename succeeds the tmp path no
	// longer exists, so the deferred Remove is a no-op (os.IsNotExist
	// silently ignored).
	renamed := false
	defer func() {
		if !renamed {
			_ = os.Remove(tmpPath)
		}
	}()
	if err := os.Rename(tmpPath, dest); err != nil {
		return SetProjectTemplateOutput{}, fmt.Errorf("atomic rename %s → %s: %w", tmpPath, dest, err)
	}
	renamed = true
	// Step 4 — swap shadow catalog onto the project + persist.
	project.KindCatalogJSON = shadowEncoded
	ctx, _, _ = withResolvedMutationActor(ctx, in.UpdatedBy, in.UpdatedByName, in.UpdatedType)
	if err := s.repo.UpdateProject(ctx, project); err != nil {
		// Rollback path (F4): the on-disk file landed but the catalog
		// did not persist. Move the file aside so the dev can recover
		// without the new template silently overriding via the F.1.2
		// walk on the next service restart. The sentinel suffix names
		// the failure case loud.
		failedPath := dest + ".tillsyn-set-failed-" + tmpSuffix + ".toml"
		rollbackErr := os.Rename(dest, failedPath)
		if rollbackErr != nil {
			return SetProjectTemplateOutput{}, fmt.Errorf(
				"persist project after template install: %w; ROLLBACK ALSO FAILED at %s → %s: %v; manual cleanup required",
				err, dest, failedPath, rollbackErr,
			)
		}
		return SetProjectTemplateOutput{}, fmt.Errorf(
			"persist project after template install: %w; on-disk file moved aside to %s for manual recovery",
			err, failedPath,
		)
	}
	return SetProjectTemplateOutput{
		ProjectID:    project.ID,
		BakeSource:   bakeSource,
		BytesWritten: len(in.TemplateTOML),
	}, nil
}
