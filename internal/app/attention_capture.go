package app

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/hylla/tillsyn/internal/domain"
)

// CaptureStateView identifies the sizing profile used by CaptureState.
type CaptureStateView string

// CaptureStateView values.
const (
	CaptureStateViewSummary CaptureStateView = "summary"
	CaptureStateViewFull    CaptureStateView = "full"
)

// captureStateAttentionLimit defines the default unresolved attention window size.
const captureStateAttentionLimit = 10

// RaiseAttentionItemInput holds write-time fields for creating one attention item.
type RaiseAttentionItemInput struct {
	Level              domain.LevelTupleInput
	Kind               domain.AttentionKind
	Summary            string
	BodyMarkdown       string
	RequiresUserAction bool
	CreatedBy          string
	CreatedType        domain.ActorType
}

// ListAttentionItemsInput holds scoped query fields for listing attention items.
type ListAttentionItemsInput struct {
	Level              domain.LevelTupleInput
	UnresolvedOnly     bool
	States             []domain.AttentionState
	Kinds              []domain.AttentionKind
	RequiresUserAction *bool
	Limit              int
}

// ResolveAttentionItemInput holds write-time fields for resolving one attention item.
type ResolveAttentionItemInput struct {
	AttentionID  string
	ResolvedBy   string
	ResolvedType domain.ActorType
}

// CaptureStateInput holds level-scoped options for summary-first recovery context.
type CaptureStateInput struct {
	Level   domain.LevelTupleInput
	View    CaptureStateView
	Include []string
}

// CaptureStateSummary stores summary-first reorientation fields for one scope.
type CaptureStateSummary struct {
	CapturedAt        time.Time                     `json:"captured_at"`
	Level             domain.LevelTuple             `json:"level"`
	GoalOverview      string                        `json:"goal_overview"`
	AttentionOverview CaptureStateAttentionOverview `json:"attention_overview"`
	WorkOverview      CaptureStateWorkOverview      `json:"work_overview"`
	FollowUpPointers  CaptureStateFollowUpPointers  `json:"follow_up_pointers"`
}

// CaptureStateAttentionOverview stores unresolved attention aggregates and highlights.
type CaptureStateAttentionOverview struct {
	UnresolvedCount         int                         `json:"unresolved_count"`
	BlockingCount           int                         `json:"blocking_count"`
	RequiresUserActionCount int                         `json:"requires_user_action_count"`
	Items                   []CaptureStateAttentionItem `json:"items"`
}

// CaptureStateAttentionItem stores one compact unresolved attention row.
type CaptureStateAttentionItem struct {
	ID                 string                `json:"id"`
	Kind               domain.AttentionKind  `json:"kind"`
	State              domain.AttentionState `json:"state"`
	Summary            string                `json:"summary"`
	RequiresUserAction bool                  `json:"requires_user_action"`
	CreatedAt          time.Time             `json:"created_at"`
}

// CaptureStateWorkOverview stores compact work-state aggregates for one project scope.
type CaptureStateWorkOverview struct {
	TotalItems      int    `json:"total_items"`
	ActiveItems     int    `json:"active_items"`
	InProgressItems int    `json:"in_progress_items"`
	DoneItems       int    `json:"done_items"`
	BlockedItems    int    `json:"blocked_items"`
	FocusItemID     string `json:"focus_item_id,omitempty"`
	OpenChildItems  int    `json:"open_child_items"`
}

// CaptureStateFollowUpPointers stores deterministic follow-up hints for deeper calls.
type CaptureStateFollowUpPointers struct {
	ListAttentionItems      string `json:"list_attention_items"`
	ListProjectChangeEvents string `json:"list_project_change_events"`
	ListChildTasks          string `json:"list_child_tasks,omitempty"`
}

// RaiseAttentionItem creates one scoped attention item with capability-guard enforcement.
func (s *Service) RaiseAttentionItem(ctx context.Context, in RaiseAttentionItemInput) (domain.AttentionItem, error) {
	level, err := domain.NewLevelTuple(in.Level)
	if err != nil {
		return domain.AttentionItem{}, err
	}
	scopeID, err := s.validateCapabilityScopeTuple(ctx, level.ProjectID, level.ScopeType.ToCapabilityScopeType(), level.ScopeID)
	if err != nil {
		return domain.AttentionItem{}, err
	}
	level.ScopeID = scopeID

	createdType := normalizeActorTypeInput(in.CreatedType)
	if err := s.enforceMutationGuard(ctx, level.ProjectID, createdType, level.ScopeType.ToCapabilityScopeType(), level.ScopeID, domain.CapabilityActionComment); err != nil {
		return domain.AttentionItem{}, err
	}

	item, err := domain.NewAttentionItem(domain.AttentionItemInput{
		ID:                 s.idGen(),
		ProjectID:          level.ProjectID,
		BranchID:           level.BranchID,
		ScopeType:          level.ScopeType,
		ScopeID:            level.ScopeID,
		Kind:               in.Kind,
		Summary:            in.Summary,
		BodyMarkdown:       in.BodyMarkdown,
		RequiresUserAction: in.RequiresUserAction,
		CreatedByActor:     in.CreatedBy,
		CreatedByType:      createdType,
	}, s.clock())
	if err != nil {
		return domain.AttentionItem{}, err
	}

	if err := s.repo.CreateAttentionItem(ctx, item); err != nil {
		return domain.AttentionItem{}, err
	}
	return item, nil
}

// ListAttentionItems lists scoped attention items in deterministic order.
func (s *Service) ListAttentionItems(ctx context.Context, in ListAttentionItemsInput) ([]domain.AttentionItem, error) {
	level, err := domain.NewLevelTuple(in.Level)
	if err != nil {
		return nil, err
	}
	filter, err := domain.NormalizeAttentionListFilter(domain.AttentionListFilter{
		ProjectID:          level.ProjectID,
		ScopeType:          level.ScopeType,
		ScopeID:            level.ScopeID,
		UnresolvedOnly:     in.UnresolvedOnly,
		States:             in.States,
		Kinds:              in.Kinds,
		RequiresUserAction: in.RequiresUserAction,
		Limit:              in.Limit,
	})
	if err != nil {
		return nil, err
	}
	return s.repo.ListAttentionItems(ctx, filter)
}

// ResolveAttentionItem marks one attention item as resolved and returns the updated row.
func (s *Service) ResolveAttentionItem(ctx context.Context, in ResolveAttentionItemInput) (domain.AttentionItem, error) {
	attentionID := strings.TrimSpace(in.AttentionID)
	if attentionID == "" {
		return domain.AttentionItem{}, domain.ErrInvalidID
	}
	existing, err := s.repo.GetAttentionItem(ctx, attentionID)
	if err != nil {
		return domain.AttentionItem{}, err
	}
	resolvedType := normalizeActorTypeInput(in.ResolvedType)
	if err := s.enforceMutationGuard(ctx, existing.ProjectID, resolvedType, existing.ScopeType.ToCapabilityScopeType(), existing.ScopeID, domain.CapabilityActionResolveAttention); err != nil {
		return domain.AttentionItem{}, err
	}
	return s.repo.ResolveAttentionItem(ctx, attentionID, strings.TrimSpace(in.ResolvedBy), resolvedType, s.clock().UTC())
}

// CaptureState returns summary-first level-scoped context for deterministic recovery.
func (s *Service) CaptureState(ctx context.Context, in CaptureStateInput) (CaptureStateSummary, error) {
	level, err := domain.NewLevelTuple(in.Level)
	if err != nil {
		return CaptureStateSummary{}, err
	}
	view := normalizeCaptureStateView(in.View)
	if _, err := s.repo.GetProject(ctx, level.ProjectID); err != nil {
		return CaptureStateSummary{}, err
	}

	attention, err := s.repo.ListAttentionItems(ctx, domain.AttentionListFilter{
		ProjectID:      level.ProjectID,
		ScopeType:      level.ScopeType,
		ScopeID:        level.ScopeID,
		UnresolvedOnly: true,
		Limit:          captureStateAttentionLimit,
	})
	if err != nil {
		return CaptureStateSummary{}, err
	}

	tasks, err := s.repo.ListTasks(ctx, level.ProjectID, true)
	if err != nil {
		return CaptureStateSummary{}, err
	}

	goalOverview := fmt.Sprintf("scope=%s:%s project=%s view=%s", level.ScopeType, level.ScopeID, level.ProjectID, view)
	return CaptureStateSummary{
		CapturedAt:        s.clock().UTC(),
		Level:             level,
		GoalOverview:      goalOverview,
		AttentionOverview: buildCaptureStateAttentionOverview(attention),
		WorkOverview:      buildCaptureStateWorkOverview(level, tasks),
		FollowUpPointers: CaptureStateFollowUpPointers{
			ListAttentionItems:      fmt.Sprintf("till.attention_item(operation=list,project_id=%q,scope_type=%q,scope_id=%q,state=%q)", level.ProjectID, level.ScopeType, level.ScopeID, "open"),
			ListProjectChangeEvents: fmt.Sprintf("till.project(operation=list_change_events,project_id=%q,limit=25)", level.ProjectID),
			ListChildTasks:          fmt.Sprintf("till.plan_item(operation=list,project_id=%q,parent_id=%q,include_archived=false)", level.ProjectID, level.ScopeID),
		},
	}, nil
}

// ensureTaskCompletionAttentionClear blocks completion when unresolved blocking attention exists.
func (s *Service) ensureTaskCompletionAttentionClear(ctx context.Context, task domain.Task) error {
	scopeType := domain.ScopeLevelFromKindAppliesTo(task.Scope)
	if scopeType == "" {
		scopeType = domain.ScopeLevelTask
	}
	attention, err := s.repo.ListAttentionItems(ctx, domain.AttentionListFilter{
		ProjectID:      task.ProjectID,
		ScopeType:      scopeType,
		ScopeID:        task.ID,
		UnresolvedOnly: true,
	})
	if err != nil {
		return err
	}

	blocking := make([]string, 0)
	for _, item := range attention {
		if !item.BlocksCompletion() {
			continue
		}
		reason := strings.TrimSpace(item.Summary)
		if reason == "" {
			reason = string(item.Kind)
		}
		blocking = append(blocking, fmt.Sprintf("%s:%s", item.ID, reason))
	}
	if len(blocking) > 0 {
		return fmt.Errorf("%w: unresolved attention (%s)", domain.ErrTransitionBlocked, strings.Join(blocking, ", "))
	}
	return nil
}

// normalizeCaptureStateView canonicalizes capture-state view hints.
func normalizeCaptureStateView(view CaptureStateView) CaptureStateView {
	switch CaptureStateView(strings.TrimSpace(strings.ToLower(string(view)))) {
	case CaptureStateViewFull:
		return CaptureStateViewFull
	default:
		return CaptureStateViewSummary
	}
}

// buildCaptureStateAttentionOverview computes unresolved attention summary fields.
func buildCaptureStateAttentionOverview(items []domain.AttentionItem) CaptureStateAttentionOverview {
	out := CaptureStateAttentionOverview{
		UnresolvedCount: len(items),
		Items:           make([]CaptureStateAttentionItem, 0, len(items)),
	}
	for _, item := range items {
		if item.BlocksCompletion() {
			out.BlockingCount++
		}
		if item.RequiresUserAction {
			out.RequiresUserActionCount++
		}
		out.Items = append(out.Items, CaptureStateAttentionItem{
			ID:                 item.ID,
			Kind:               item.Kind,
			State:              item.State,
			Summary:            item.Summary,
			RequiresUserAction: item.RequiresUserAction,
			CreatedAt:          item.CreatedAt,
		})
	}
	return out
}

// buildCaptureStateWorkOverview computes compact work aggregates for a project scope.
func buildCaptureStateWorkOverview(level domain.LevelTuple, tasks []domain.Task) CaptureStateWorkOverview {
	out := CaptureStateWorkOverview{
		TotalItems:  len(tasks),
		FocusItemID: level.ScopeID,
	}
	for _, task := range tasks {
		if task.ArchivedAt == nil {
			out.ActiveItems++
		}
		if task.LifecycleState == domain.StateProgress {
			out.InProgressItems++
		}
		if task.LifecycleState == domain.StateDone {
			out.DoneItems++
		}
		if len(task.Metadata.BlockedBy) > 0 || strings.TrimSpace(task.Metadata.BlockedReason) != "" {
			out.BlockedItems++
		}
		if level.ScopeType == domain.ScopeLevelProject {
			continue
		}
		if task.ParentID != level.ScopeID {
			continue
		}
		if task.ArchivedAt != nil {
			continue
		}
		if task.LifecycleState != domain.StateDone {
			out.OpenChildItems++
		}
	}
	if level.ScopeType == domain.ScopeLevelProject {
		out.FocusItemID = ""
	}
	return out
}
