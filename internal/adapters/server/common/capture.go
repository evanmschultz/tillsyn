package common

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/evanmschultz/tillsyn/internal/domain"
)

// CaptureStateService builds summary-first capture_state responses from app read models.
type CaptureStateService struct {
	read      CaptureStateReadModel
	attention AttentionService
	now       func() time.Time
}

// captureStateCommentReadModel defines optional comment reads for capture comment counters.
type captureStateCommentReadModel interface {
	ListCommentsByTarget(context.Context, domain.CommentTarget) ([]domain.Comment, error)
}

// NewCaptureStateService constructs one capture-state adapter over app-level read methods.
func NewCaptureStateService(read CaptureStateReadModel, attention AttentionService, now func() time.Time) *CaptureStateService {
	if now == nil {
		now = time.Now
	}
	return &CaptureStateService{
		read:      read,
		attention: attention,
		now:       now,
	}
}

// CaptureState resolves one deterministic capture_state summary for a supported scope.
func (s *CaptureStateService) CaptureState(ctx context.Context, in CaptureStateRequest) (CaptureState, error) {
	if s == nil || s.read == nil {
		return CaptureState{}, fmt.Errorf("capture service is not configured: %w", ErrInvalidCaptureStateRequest)
	}
	req, err := normalizeCaptureStateRequest(in)
	if err != nil {
		return CaptureState{}, err
	}

	projects, err := s.read.ListProjects(ctx, true)
	if err != nil {
		return CaptureState{}, fmt.Errorf("list projects: %w", err)
	}
	project, ok := findProjectByID(projects, req.ProjectID)
	if !ok {
		return CaptureState{}, fmt.Errorf("project %q: %w", req.ProjectID, ErrNotFound)
	}

	columns, err := s.read.ListColumns(ctx, project.ID, true)
	if err != nil {
		return CaptureState{}, fmt.Errorf("list columns: %w", err)
	}
	tasks, err := s.read.ListActionItems(ctx, project.ID, true)
	if err != nil {
		return CaptureState{}, fmt.Errorf("list tasks: %w", err)
	}
	sortColumns(columns)
	sortActionItems(tasks)

	attentionOverview, err := s.buildAttentionOverview(ctx, req)
	if err != nil {
		return CaptureState{}, err
	}
	commentOverview, err := s.buildCommentOverview(ctx, req)
	if err != nil {
		return CaptureState{}, err
	}
	workOverview := buildWorkOverview(tasks)
	warningsOverview := buildWarningsOverview(workOverview, attentionOverview)

	capturedAt := s.now().UTC().Truncate(time.Second)
	stateHash, err := computeStateHash(project, columns, tasks, attentionOverview)
	if err != nil {
		return CaptureState{}, fmt.Errorf("compute state hash: %w", err)
	}

	scopePath := []ScopeNode{
		{
			ScopeType: ScopeTypeProject,
			ScopeID:   project.ID,
			Name:      project.Name,
		},
	}
	if req.ScopeType != ScopeTypeProject {
		scopePath = append(scopePath, ScopeNode{
			ScopeType: req.ScopeType,
			ScopeID:   req.ScopeID,
			Name:      req.ScopeID,
		})
	}

	return CaptureState{
		CapturedAt: capturedAt,
		ScopePath:  scopePath,
		StateHash:  stateHash,
		GoalOverview: GoalOverview{
			ProjectID:          project.ID,
			ProjectName:        project.Name,
			ProjectDescription: project.Description,
		},
		AttentionOverview: attentionOverview,
		WorkOverview:      workOverview,
		CommentOverview:   commentOverview,
		WarningsOverview:  warningsOverview,
		ResumeHints: []ResumeHint{
			{
				Rel:  "till.capture_state",
				Note: "request view=full for expanded, summary-safe context",
			},
			{
				Rel:  "till.attention_item",
				Note: "list unresolved attention records with till.attention_item(operation=list, state=\"open\") when available",
			},
		},
		RequestedView:      req.View,
		RequestedScopeType: req.ScopeType,
	}, nil
}

// buildAttentionOverview resolves unresolved attention metadata when the attention surface exists.
func (s *CaptureStateService) buildAttentionOverview(ctx context.Context, req CaptureStateRequest) (AttentionOverview, error) {
	overview := AttentionOverview{
		Available: s.attention != nil,
	}
	if s.attention == nil {
		return overview, nil
	}

	items, err := s.attention.ListAttentionItems(ctx, ListAttentionItemsRequest{
		ProjectID: req.ProjectID,
		ScopeType: req.ScopeType,
		ScopeID:   req.ScopeID,
		State:     AttentionStateOpen,
	})
	if err != nil {
		return AttentionOverview{}, fmt.Errorf("list attention items: %w", err)
	}
	slices.SortFunc(items, compareAttentionItems)
	overview.Items = items
	overview.OpenCount = len(items)
	for _, item := range items {
		if item.RequiresUserAction {
			overview.RequiresUserAction++
		}
	}
	return overview, nil
}

// normalizeCaptureStateRequest validates and normalizes a capture_state request.
func normalizeCaptureStateRequest(in CaptureStateRequest) (CaptureStateRequest, error) {
	projectID := strings.TrimSpace(in.ProjectID)
	if projectID == "" {
		return CaptureStateRequest{}, fmt.Errorf("project_id is required: %w", ErrInvalidCaptureStateRequest)
	}

	scopeType := canonicalScopeType(in.ScopeType)
	if scopeType == "" {
		scopeType = ScopeTypeProject
	}
	if !slices.Contains(supportedScopeTypes, scopeType) {
		return CaptureStateRequest{}, fmt.Errorf("scope_type %q is unsupported: %w", scopeType, ErrUnsupportedScope)
	}

	scopeID := strings.TrimSpace(in.ScopeID)
	switch scopeType {
	case ScopeTypeProject:
		if scopeID == "" {
			scopeID = projectID
		}
		if scopeID != projectID {
			return CaptureStateRequest{}, fmt.Errorf("scope_id %q must equal project_id %q for project scope: %w", scopeID, projectID, ErrUnsupportedScope)
		}
	default:
		if scopeID == "" {
			return CaptureStateRequest{}, fmt.Errorf("scope_id is required for scope_type %q: %w", scopeType, ErrUnsupportedScope)
		}
	}
	view := strings.ToLower(strings.TrimSpace(in.View))
	if view == "" {
		view = "summary"
	}
	if view != "summary" && view != "full" {
		return CaptureStateRequest{}, fmt.Errorf("view %q is unsupported: %w", view, ErrInvalidCaptureStateRequest)
	}

	return CaptureStateRequest{
		ProjectID: projectID,
		ScopeType: scopeType,
		ScopeID:   scopeID,
		View:      view,
	}, nil
}

// findProjectByID resolves one project by id.
func findProjectByID(projects []domain.Project, projectID string) (domain.Project, bool) {
	for _, project := range projects {
		if strings.TrimSpace(project.ID) == projectID {
			return project, true
		}
	}
	return domain.Project{}, false
}

// sortColumns keeps column ordering deterministic for capture_state responses and hashing.
func sortColumns(columns []domain.Column) {
	slices.SortFunc(columns, func(a, b domain.Column) int {
		if a.Position != b.Position {
			if a.Position < b.Position {
				return -1
			}
			return 1
		}
		return strings.Compare(a.ID, b.ID)
	})
}

// sortActionItems keeps actionItem ordering deterministic for capture_state responses and hashing.
func sortActionItems(tasks []domain.ActionItem) {
	slices.SortFunc(tasks, func(a, b domain.ActionItem) int {
		if a.Position != b.Position {
			if a.Position < b.Position {
				return -1
			}
			return 1
		}
		return strings.Compare(a.ID, b.ID)
	})
}

// buildWorkOverview summarizes actionItem-state counters and completion blockers.
func buildWorkOverview(tasks []domain.ActionItem) WorkOverview {
	overview := WorkOverview{
		TotalActionItems: len(tasks),
	}
	childrenByParent := make(map[string][]domain.ActionItem, len(tasks))
	for _, actionItem := range tasks {
		parentID := strings.TrimSpace(actionItem.ParentID)
		if parentID == "" {
			continue
		}
		childrenByParent[parentID] = append(childrenByParent[parentID], actionItem)
	}

	for _, actionItem := range tasks {
		switch canonicalLifecycleState(actionItem.LifecycleState) {
		case domain.StateTodo:
			overview.TodoActionItems++
		case domain.StateProgress:
			overview.InProgressActionItems++
		case domain.StateDone:
			overview.DoneActionItems++
		case domain.StateFailed:
			overview.FailedActionItems++
		case domain.StateArchived:
			overview.ArchivedActionItems++
		default:
			overview.TodoActionItems++
		}

		if actionItem.ArchivedAt != nil {
			continue
		}
		if strings.TrimSpace(actionItem.Metadata.BlockedReason) != "" || len(actionItem.Metadata.BlockedBy) > 0 {
			overview.ActionItemsWithOpenBlockers++
		}
		if len(actionItem.CompletionCriteriaUnmet(childrenByParent[actionItem.ID])) > 0 {
			overview.IncompleteCompletionCriteria++
		}
	}

	return overview
}

// buildWarningsOverview synthesizes warning text from work and attention rollups.
func buildWarningsOverview(work WorkOverview, attention AttentionOverview) WarningsOverview {
	warnings := make([]string, 0, 2)
	if work.ActionItemsWithOpenBlockers > 0 {
		warnings = append(warnings, fmt.Sprintf("%d work items report open blockers", work.ActionItemsWithOpenBlockers))
	}
	if attention.RequiresUserAction > 0 {
		warnings = append(warnings, fmt.Sprintf("%d attention items require user action", attention.RequiresUserAction))
	}
	return WarningsOverview{Warnings: warnings}
}

// canonicalLifecycleState normalizes lifecycle aliases into canonical values.
func canonicalLifecycleState(state domain.LifecycleState) domain.LifecycleState {
	switch strings.ToLower(strings.TrimSpace(string(state))) {
	case "todo", "to-do":
		return domain.StateTodo
	case "progress", "in-progress", "doing":
		return domain.StateProgress
	case "done", "complete", "completed":
		return domain.StateDone
	case "failed", "fail":
		return domain.StateFailed
	case "archived", "archive":
		return domain.StateArchived
	default:
		return domain.StateTodo
	}
}

// buildCommentOverview resolves scoped comment counters when the read model supports comment queries.
func (s *CaptureStateService) buildCommentOverview(ctx context.Context, req CaptureStateRequest) (CommentOverview, error) {
	commentRead, ok := s.read.(captureStateCommentReadModel)
	if !ok {
		return CommentOverview{}, nil
	}
	targetType, ok := commentTargetTypeFromScope(req.ScopeType)
	if !ok {
		return CommentOverview{}, nil
	}
	comments, err := commentRead.ListCommentsByTarget(ctx, domain.CommentTarget{
		ProjectID:  req.ProjectID,
		TargetType: targetType,
		TargetID:   req.ScopeID,
	})
	if err != nil {
		return CommentOverview{}, fmt.Errorf("list comments by target: %w", err)
	}
	return summarizeCommentOverview(comments), nil
}

// compareAttentionItems deterministically sorts attention records for stable outputs and hashes.
func compareAttentionItems(a, b AttentionItem) int {
	if !a.CreatedAt.Equal(b.CreatedAt) {
		if a.CreatedAt.Before(b.CreatedAt) {
			return -1
		}
		return 1
	}
	return strings.Compare(a.ID, b.ID)
}

// computeStateHash returns a deterministic summary hash for capture_state responses.
func computeStateHash(project domain.Project, columns []domain.Column, tasks []domain.ActionItem, attention AttentionOverview) (string, error) {
	payload := struct {
		Project            domain.Project      `json:"project"`
		Columns            []domain.Column     `json:"columns"`
		ActionItems        []domain.ActionItem `json:"tasks"`
		AttentionOpenCount int                 `json:"attention_open_count"`
		AttentionRequires  int                 `json:"attention_requires_user_action"`
	}{
		Project:            project,
		Columns:            columns,
		ActionItems:        tasks,
		AttentionOpenCount: attention.OpenCount,
		AttentionRequires:  attention.RequiresUserAction,
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal capture payload: %w", err)
	}
	sum := sha256.Sum256(encoded)
	return hex.EncodeToString(sum[:]), nil
}
