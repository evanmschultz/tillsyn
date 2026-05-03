package app

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/evanmschultz/tillsyn/internal/domain"
	"github.com/evanmschultz/tillsyn/internal/templates"
)

// stewardOwner is the canonical Owner string the auto-generator stamps on
// every STEWARD-owned anchor seed and every drop-end finding it materializes.
// Mirrors the constant in
// internal/adapters/server/common/app_service_adapter_mcp.go (kept duplicated
// rather than imported to avoid a cross-package coupling between the app
// service and the MCP adapter — both packages are stable consumers of the
// same string literal). Per fix L13 this string is a domain primitive value,
// not a STEWARD-specific concept; the auth gate consumes it via
// app_service_adapter_mcp.go's owner-state-lock check (droplet 3.19).
const stewardOwner = "STEWARD"

// errStewardParentNotSeeded is returned by seedDropFindingsAndGate when a
// caller asks for level_2 finding materialization on a project whose STEWARD
// persistent anchor parents have not been seeded. The auto-generator never
// fabricates the anchor inline — that would mask a project-creation bug
// where seedStewardAnchors silently failed. Callers see ErrNotFound from the
// repository wrapped with this sentinel for diagnostic clarity.
var errStewardParentNotSeeded = errors.New("steward persistent parent not seeded for project")

// loadStewardSeedTemplate is a package-level seam returning the cascade
// Template the auto-generator iterates. It defaults to
// templates.LoadDefaultTemplate so the embedded internal/templates/builtin/
// default.toml is the canonical source. Tests substitute a hand-built
// Template (or an error) by replacing this seam in a t.Cleanup; the
// production code does not mutate it.
//
// Pre-MVP rule: per-project template override resolution
// (<project_root>/.tillsyn/template.toml) is deferred to a future drop —
// until then every project shares the embedded default.
var loadStewardSeedTemplate = func() (templates.Template, error) {
	return templates.LoadDefaultTemplate()
}

// seedStewardAnchors materializes the long-lived STEWARD-owned anchor
// nodes recorded in the cascade template's StewardSeeds slice. Called once
// per project creation by Service.CreateProjectWithMetadata (and
// Service.EnsureDefaultProject) AFTER repo.CreateProject + before column
// auto-creation so the seeded children land beneath a fully-persisted
// project row.
//
// Each seed materializes as a level_1 ActionItem with:
//
//   - Kind           = domain.KindDiscussion (cross-cutting anchor; no
//     auto-QA twins, matching CLAUDE.md's
//     "Required Children" rule for discussion kinds).
//   - StructuralType = domain.StructuralTypeDroplet (terminal anchor —
//     domain.NewActionItem rejects an empty
//     StructuralType, so this is the closest fit
//     to "no cascade shape" the methodology calls
//     for in PLAN.md § 19.3 line 1637).
//   - Owner          = "STEWARD"  (per fix L7 — the auth gate keys on this).
//   - Persistent     = true       (per fix L9 — survives across drops).
//   - DevGated       = false      (anchors do not gate dev sign-off
//     themselves; their CHILDREN may).
//   - Title          = seed.Title (FULL UPPERCASE per memory rule
//     project_tillsyn_titles).
//   - Description    = seed.Description (verbatim).
//   - DropNumber     = 0          (anchors are not numbered drops).
//
// Idempotency: the helper checks for an existing
// (project_id, owner = "STEWARD", title) row via
// Repository.FindActionItemByOwnerAndTitle BEFORE creating each seed.
// Existing rows are left untouched (no description re-write, no
// re-stamping); missing rows are created. This preserves the dev's
// post-creation edits to anchor descriptions and makes re-running the hook
// safe for partially-seeded projects. The repository call uses the
// idx_action_items_owner_title (project_id, owner, title) composite landed
// in droplet 3.18, so the lookup is index-covered.
//
// Returns the first error encountered. The helper does NOT roll back
// already-created seeds on later failures; partial seeding is preferred to
// a project with zero anchors. Callers that detect the failure can
// re-invoke seedStewardAnchors after fixing the root cause and idempotency
// will pick up where the previous attempt left off.
func (s *Service) seedStewardAnchors(ctx context.Context, project domain.Project) error {
	tpl, err := loadStewardSeedTemplate()
	if err != nil {
		return fmt.Errorf("seed steward anchors: load template: %w", err)
	}
	if len(tpl.StewardSeeds) == 0 {
		return nil
	}
	column, err := s.firstColumnForProject(ctx, project.ID)
	if err != nil {
		return fmt.Errorf("seed steward anchors: %w", err)
	}
	for _, seed := range tpl.StewardSeeds {
		title := strings.TrimSpace(seed.Title)
		if title == "" {
			return fmt.Errorf("seed steward anchors: empty title in template")
		}
		existing, lookupErr := s.repo.FindActionItemByOwnerAndTitle(ctx, project.ID, stewardOwner, title)
		if lookupErr == nil {
			_ = existing
			continue
		}
		if !errors.Is(lookupErr, ErrNotFound) {
			return fmt.Errorf("seed steward anchors: lookup %q: %w", title, lookupErr)
		}
		if _, createErr := s.CreateActionItem(ctx, CreateActionItemInput{
			ProjectID:      project.ID,
			Kind:           domain.KindDiscussion,
			StructuralType: domain.StructuralTypeDroplet,
			Owner:          stewardOwner,
			Persistent:     true,
			DevGated:       false,
			ColumnID:       column.ID,
			Title:          title,
			Description:    strings.TrimSpace(seed.Description),
			Priority:       domain.PriorityMedium,
			CreatedByActor: stewardOwner,
			CreatedByName:  stewardOwner,
			UpdatedByActor: stewardOwner,
			UpdatedByName:  stewardOwner,
			UpdatedByType:  domain.ActorTypeUser,
		}); createErr != nil {
			return fmt.Errorf("seed steward anchors: create %q: %w", title, createErr)
		}
	}
	return nil
}

// dropFindingPlan describes one DROP_N_<KIND>_<SUFFIX> level_2 finding the
// auto-generator materializes when a level_1 numbered drop is created.
// Each finding lives under a specific STEWARD persistent anchor, matched by
// title. The slice below pins the canonical 5 findings from the dev rule
// "every numbered drop emits these five drop-end findings" — the order is
// load-bearing only for deterministic test assertions.
type dropFindingPlan struct {
	parentTitle string
	suffix      string
}

// canonicalDropFindings is the ordered list of (anchor parent, drop-end
// finding suffix) pairs the auto-generator materializes per numbered drop.
// The anchor titles match the seeds in templates.StewardSeed defaults; the
// suffixes follow the DROP_<N>_<SUFFIX> convention used by drop-orchs in
// per-drop workflow MDs.
var canonicalDropFindings = []dropFindingPlan{
	{parentTitle: "HYLLA_FINDINGS", suffix: "HYLLA_FINDINGS"},
	{parentTitle: "LEDGER", suffix: "LEDGER_ENTRY"},
	{parentTitle: "WIKI_CHANGELOG", suffix: "WIKI_CHANGELOG_ENTRY"},
	{parentTitle: "REFINEMENTS", suffix: "REFINEMENTS_RAISED"},
	{parentTitle: "HYLLA_REFINEMENTS", suffix: "HYLLA_REFINEMENTS_RAISED"},
}

// seedDropFindingsAndGate materializes the 5 STEWARD-owned level_2 drop-end
// findings (one each under HYLLA_FINDINGS / LEDGER / WIKI_CHANGELOG /
// REFINEMENTS / HYLLA_REFINEMENTS) plus the refinements-gate confluence
// inside the new drop's tree. Called by Service.CreateActionItem when a
// level_1 numbered-drop action item lands.
//
// The five findings each carry:
//
//   - Kind           = domain.KindDiscussion (placeholder cross-cutting kind;
//     future drops may refine).
//   - StructuralType = domain.StructuralTypeDroplet (single non-decomposable
//     finding entry).
//   - Owner          = "STEWARD".
//   - Persistent     = false (these are drop-bound, not perpetual).
//   - DevGated       = false.
//   - DropNumber     = N (the numbered-drop index from the parent).
//   - Title          = "DROP_<N>_<SUFFIX>" (e.g. "DROP_3_HYLLA_FINDINGS").
//   - ParentID       = the matching STEWARD anchor's ID.
//
// The refinements-gate confluence carries:
//
//   - Kind           = domain.KindPlan.
//   - StructuralType = domain.StructuralTypeConfluence (multiple inputs join
//     before the next drop can start).
//   - Owner          = "STEWARD".
//   - Persistent     = false.
//   - DevGated       = true (per fix L10 — refinements-gate requires dev
//     sign-off before DROP_N+1 can begin).
//   - DropNumber     = N.
//   - Title          = "DROP_<N>_REFINEMENTS_GATE_BEFORE_DROP_<N+1>".
//   - ParentID       = the new drop's own ID (lives inside the drop tree).
//
// blocked_by wiring: the gate's blocked_by enumerates every action item
// whose drop_number = N (excluding the gate itself) PLUS the 5 newly-
// created findings — so closing the gate is contingent on every drop
// participant reaching its terminal state. Today this is recorded as
// metadata.blocked_by; the dispatcher (Drop 4) will consume it as a real
// edge. See ListActionItemsByDropNumber for the index-covered query.
//
// Idempotency: if the helper finds an already-existing (project_id,
// owner=STEWARD, title) row for any of the 5 findings or the gate, that
// row is skipped (consistent with seedStewardAnchors). Re-creating a
// numbered drop after a transient failure is therefore safe.
//
// The helper assumes seedStewardAnchors has already run for the project;
// missing anchors return errStewardParentNotSeeded so the caller can
// surface a clear diagnostic rather than silently dropping the finding.
func (s *Service) seedDropFindingsAndGate(ctx context.Context, drop domain.ActionItem) error {
	if drop.DropNumber <= 0 {
		return nil
	}
	column, err := s.firstColumnForProject(ctx, drop.ProjectID)
	if err != nil {
		return fmt.Errorf("seed drop findings: %w", err)
	}
	dropPrefix := fmt.Sprintf("DROP_%d_", drop.DropNumber)
	createdFindingIDs := make([]string, 0, len(canonicalDropFindings))
	for _, plan := range canonicalDropFindings {
		anchor, anchorErr := s.repo.FindActionItemByOwnerAndTitle(ctx, drop.ProjectID, stewardOwner, plan.parentTitle)
		if anchorErr != nil {
			if errors.Is(anchorErr, ErrNotFound) {
				return fmt.Errorf("%w: %q", errStewardParentNotSeeded, plan.parentTitle)
			}
			return fmt.Errorf("seed drop findings: lookup anchor %q: %w", plan.parentTitle, anchorErr)
		}
		findingTitle := dropPrefix + plan.suffix
		if existing, lookupErr := s.repo.FindActionItemByOwnerAndTitle(ctx, drop.ProjectID, stewardOwner, findingTitle); lookupErr == nil {
			createdFindingIDs = append(createdFindingIDs, existing.ID)
			continue
		} else if !errors.Is(lookupErr, ErrNotFound) {
			return fmt.Errorf("seed drop findings: lookup %q: %w", findingTitle, lookupErr)
		}
		finding, createErr := s.CreateActionItem(ctx, CreateActionItemInput{
			ProjectID:      drop.ProjectID,
			ParentID:       anchor.ID,
			Kind:           domain.KindDiscussion,
			StructuralType: domain.StructuralTypeDroplet,
			Owner:          stewardOwner,
			DropNumber:     drop.DropNumber,
			Persistent:     false,
			DevGated:       false,
			ColumnID:       column.ID,
			Title:          findingTitle,
			Description:    fmt.Sprintf("STEWARD-owned %s entry for drop %d. Description body authored by drop-orch at drop end.", plan.suffix, drop.DropNumber),
			Priority:       domain.PriorityMedium,
			CreatedByActor: stewardOwner,
			CreatedByName:  stewardOwner,
			UpdatedByActor: stewardOwner,
			UpdatedByName:  stewardOwner,
			UpdatedByType:  domain.ActorTypeUser,
		})
		if createErr != nil {
			return fmt.Errorf("seed drop findings: create %q: %w", findingTitle, createErr)
		}
		createdFindingIDs = append(createdFindingIDs, finding.ID)
	}
	gateTitle := fmt.Sprintf("DROP_%d_REFINEMENTS_GATE_BEFORE_DROP_%d", drop.DropNumber, drop.DropNumber+1)
	if _, lookupErr := s.repo.FindActionItemByOwnerAndTitle(ctx, drop.ProjectID, stewardOwner, gateTitle); lookupErr == nil {
		return nil
	} else if !errors.Is(lookupErr, ErrNotFound) {
		return fmt.Errorf("seed drop findings: lookup gate %q: %w", gateTitle, lookupErr)
	}
	blockedBy, blockedByErr := s.assembleRefinementsGateBlockedBy(ctx, drop, createdFindingIDs)
	if blockedByErr != nil {
		return fmt.Errorf("seed drop findings: assemble blocked_by: %w", blockedByErr)
	}
	gateMetadata := domain.ActionItemMetadata{
		BlockedBy: blockedBy,
	}
	gateDescription := fmt.Sprintf("STEWARD-owned refinements-gate confluence for drop %d. blocked_by enumerates every drop_number=%d action item plus the 5 STEWARD-owned drop-end findings; the gate's terminal transition requires dev sign-off (DevGated=true).",
		drop.DropNumber, drop.DropNumber)
	if _, createErr := s.CreateActionItem(ctx, CreateActionItemInput{
		ProjectID:      drop.ProjectID,
		ParentID:       drop.ID,
		Kind:           domain.KindPlan,
		StructuralType: domain.StructuralTypeConfluence,
		Owner:          stewardOwner,
		DropNumber:     drop.DropNumber,
		Persistent:     false,
		DevGated:       true,
		ColumnID:       column.ID,
		Title:          gateTitle,
		Description:    gateDescription,
		Priority:       domain.PriorityHigh,
		Metadata:       gateMetadata,
		CreatedByActor: stewardOwner,
		CreatedByName:  stewardOwner,
		UpdatedByActor: stewardOwner,
		UpdatedByName:  stewardOwner,
		UpdatedByType:  domain.ActorTypeUser,
	}); createErr != nil {
		return fmt.Errorf("seed drop findings: create gate %q: %w", gateTitle, createErr)
	}
	return nil
}

// assembleRefinementsGateBlockedBy collects the deterministic, sorted list
// of action-item IDs the refinements-gate must wait on before its terminal
// transition: every drop_number=N action item EXCEPT the gate itself, plus
// the 5 STEWARD-owned drop-end findings just created. The findings are
// included even though they ALSO carry drop_number=N — duplicates are
// collapsed via a set so callers cannot accidentally double-block.
//
// The gate's own ID is excluded by virtue of not yet existing — the helper
// runs before the gate's CreateActionItem call. The auto-generator then
// passes the assembled list as the gate's metadata.blocked_by.
func (s *Service) assembleRefinementsGateBlockedBy(ctx context.Context, drop domain.ActionItem, findingIDs []string) ([]string, error) {
	dropItems, err := s.repo.ListActionItemsByDropNumber(ctx, drop.ProjectID, drop.DropNumber)
	if err != nil {
		return nil, err
	}
	seen := make(map[string]struct{}, len(dropItems)+len(findingIDs))
	for _, item := range dropItems {
		seen[item.ID] = struct{}{}
	}
	for _, id := range findingIDs {
		seen[id] = struct{}{}
	}
	out := make([]string, 0, len(seen))
	for id := range seen {
		out = append(out, id)
	}
	sort.Strings(out)
	return out, nil
}

// firstColumnForProject returns the first non-archived column on the
// supplied project so seedStewardAnchors / seedDropFindingsAndGate have a
// landing column for the children they create. Auto-create columns
// (when ServiceConfig.AutoCreateProjectColumns is true) populate at least
// one column at project creation; otherwise the caller is expected to have
// created at least one column before invoking the seed paths.
func (s *Service) firstColumnForProject(ctx context.Context, projectID string) (domain.Column, error) {
	columns, err := s.repo.ListColumns(ctx, projectID, false)
	if err != nil {
		return domain.Column{}, fmt.Errorf("list columns: %w", err)
	}
	if len(columns) == 0 {
		return domain.Column{}, fmt.Errorf("project %q has no columns", projectID)
	}
	first := columns[0]
	for _, c := range columns[1:] {
		if c.Position < first.Position {
			first = c
		}
	}
	return first, nil
}
