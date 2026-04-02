// Package common provides transport-agnostic server contracts used by HTTP and MCP adapters.
package common

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/hylla/tillsyn/internal/domain"
)

// ScopeTypeProject identifies project-level scope for capture and attention operations.
const ScopeTypeProject = "project"

// ScopeTypeBranch identifies branch-level scope for capture and attention operations.
const ScopeTypeBranch = "branch"

// ScopeTypePhase identifies phase-level scope for capture and attention operations.
const ScopeTypePhase = "phase"

// ScopeTypeTask identifies task-level scope for capture and attention operations.
const ScopeTypeTask = "task"

// ScopeTypeSubtask identifies subtask-level scope for capture and attention operations.
const ScopeTypeSubtask = "subtask"

// supportedScopeTypes stores all transport-accepted scope values in canonical order.
var supportedScopeTypes = []string{
	ScopeTypeProject,
	ScopeTypeBranch,
	ScopeTypePhase,
	ScopeTypeTask,
	ScopeTypeSubtask,
}

// SupportedScopeTypes returns all canonical scope_type values accepted by transport adapters.
func SupportedScopeTypes() []string {
	return append([]string(nil), supportedScopeTypes...)
}

// commentTargetTypeFromScope maps transport scope_type values to comment target types.
func commentTargetTypeFromScope(scopeType string) (domain.CommentTargetType, bool) {
	switch strings.ToLower(strings.TrimSpace(scopeType)) {
	case ScopeTypeProject:
		return domain.CommentTargetTypeProject, true
	case ScopeTypeBranch:
		return domain.CommentTargetTypeBranch, true
	case ScopeTypePhase:
		return domain.CommentTargetTypePhase, true
	case ScopeTypeTask:
		return domain.CommentTargetTypeTask, true
	case ScopeTypeSubtask:
		return domain.CommentTargetTypeSubtask, true
	default:
		return "", false
	}
}

// AttentionStateOpen identifies unresolved attention records.
const AttentionStateOpen = "open"

// AttentionStateAcknowledged identifies acknowledged-but-unresolved attention records.
const AttentionStateAcknowledged = "acknowledged"

// AttentionStateResolved identifies closed attention records.
const AttentionStateResolved = "resolved"

// ErrInvalidCaptureStateRequest reports malformed capture-state input.
var ErrInvalidCaptureStateRequest = errors.New("invalid capture_state request")

// ErrUnsupportedScope reports unsupported scope tuples.
var ErrUnsupportedScope = errors.New("unsupported scope")

// ErrAttentionUnavailable reports missing attention backing support.
var ErrAttentionUnavailable = errors.New("attention surface unavailable")

// ErrNotFound reports missing transport-visible resources.
var ErrNotFound = errors.New("not found")

// CaptureStateRequest captures one summary request for a scoped board/project state snapshot.
type CaptureStateRequest struct {
	ProjectID string
	ScopeType string
	ScopeID   string
	View      string
}

// ScopeNode describes one node in the resolved scope path for capture_state responses.
type ScopeNode struct {
	ScopeType string `json:"scope_type"`
	ScopeID   string `json:"scope_id"`
	Name      string `json:"name"`
}

// GoalOverview summarizes the active goal context.
type GoalOverview struct {
	ProjectID          string `json:"project_id"`
	ProjectName        string `json:"project_name"`
	ProjectDescription string `json:"project_description,omitempty"`
}

// AttentionItem represents one attention record surfaced by transport adapters.
type AttentionItem struct {
	ID                 string     `json:"id"`
	ProjectID          string     `json:"project_id"`
	ScopeType          string     `json:"scope_type"`
	ScopeID            string     `json:"scope_id"`
	State              string     `json:"state"`
	Kind               string     `json:"kind"`
	Summary            string     `json:"summary"`
	BodyMarkdown       string     `json:"body_markdown,omitempty"`
	TargetRole         string     `json:"target_role,omitempty"`
	RequiresUserAction bool       `json:"requires_user_action"`
	CreatedAt          time.Time  `json:"created_at"`
	ResolvedAt         *time.Time `json:"resolved_at,omitempty"`
}

// AttentionOverview summarizes unresolved attention status.
type AttentionOverview struct {
	Available          bool            `json:"available"`
	OpenCount          int             `json:"open_count"`
	RequiresUserAction int             `json:"requires_user_action"`
	Items              []AttentionItem `json:"items,omitempty"`
}

// WorkOverview summarizes work-state counts for the scoped view.
type WorkOverview struct {
	TotalTasks                   int `json:"total_tasks"`
	TodoTasks                    int `json:"todo_tasks"`
	InProgressTasks              int `json:"in_progress_tasks"`
	DoneTasks                    int `json:"done_tasks"`
	ArchivedTasks                int `json:"archived_tasks"`
	TasksWithOpenBlockers        int `json:"tasks_with_open_blockers"`
	IncompleteCompletionCriteria int `json:"incomplete_completion_criteria"`
}

// CommentOverview reports compact comment counters used for resume hints.
type CommentOverview struct {
	RecentCount    int `json:"recent_count"`
	ImportantCount int `json:"important_count"`
}

// WarningsOverview carries synthesized warnings for fast triage.
type WarningsOverview struct {
	Warnings []string `json:"warnings,omitempty"`
}

// ResumeHint points clients to deterministic follow-up queries.
type ResumeHint struct {
	Rel  string `json:"rel"`
	Note string `json:"note,omitempty"`
}

// CaptureState is the summary-first state bundle returned to HTTP and MCP callers.
type CaptureState struct {
	CapturedAt         time.Time         `json:"captured_at"`
	ScopePath          []ScopeNode       `json:"scope_path"`
	StateHash          string            `json:"state_hash"`
	LastChangeEventID  string            `json:"last_change_event_id,omitempty"`
	GoalOverview       GoalOverview      `json:"goal_overview"`
	AttentionOverview  AttentionOverview `json:"attention_overview"`
	WorkOverview       WorkOverview      `json:"work_overview"`
	CommentOverview    CommentOverview   `json:"comment_overview"`
	WarningsOverview   WarningsOverview  `json:"warnings_overview"`
	ResumeHints        []ResumeHint      `json:"resume_hints,omitempty"`
	RequestedView      string            `json:"requested_view,omitempty"`
	RequestedScopeType string            `json:"requested_scope_type,omitempty"`
}

// CaptureStateReader resolves one capture_state request.
type CaptureStateReader interface {
	CaptureState(context.Context, CaptureStateRequest) (CaptureState, error)
}

// CaptureStateReadModel defines app-facing reads used to synthesize capture_state.
type CaptureStateReadModel interface {
	ListProjects(context.Context, bool) ([]domain.Project, error)
	ListColumns(context.Context, string, bool) ([]domain.Column, error)
	ListTasks(context.Context, string, bool) ([]domain.Task, error)
}

// ListAttentionItemsRequest captures list query filters for attention records.
type ListAttentionItemsRequest struct {
	ProjectID  string
	ScopeType  string
	ScopeID    string
	State      string
	AllScopes  bool
	TargetRole string
}

// RaiseAttentionItemRequest captures input for new attention records.
type RaiseAttentionItemRequest struct {
	ProjectID          string `json:"project_id"`
	ScopeType          string `json:"scope_type"`
	ScopeID            string `json:"scope_id"`
	Kind               string `json:"kind"`
	Summary            string `json:"summary"`
	BodyMarkdown       string `json:"body_markdown,omitempty"`
	TargetRole         string `json:"target_role,omitempty"`
	RequiresUserAction bool   `json:"requires_user_action"`
	Actor              ActorLeaseTuple
}

// ResolveAttentionItemRequest captures input for resolving one attention record.
type ResolveAttentionItemRequest struct {
	ID     string `json:"id"`
	Reason string `json:"reason,omitempty"`
	Actor  ActorLeaseTuple
}

// AttentionService captures optional attention operations exposed by app services.
type AttentionService interface {
	ListAttentionItems(context.Context, ListAttentionItemsRequest) ([]AttentionItem, error)
	RaiseAttentionItem(context.Context, RaiseAttentionItemRequest) (AttentionItem, error)
	ResolveAttentionItem(context.Context, ResolveAttentionItemRequest) (AttentionItem, error)
}
