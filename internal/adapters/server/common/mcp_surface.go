package common

import (
	"context"
	"errors"
	"time"

	"github.com/hylla/tillsyn/internal/domain"
)

// ErrBootstrapRequired reports that no project data exists and guided bootstrap is required.
var ErrBootstrapRequired = errors.New("bootstrap is required")

// ErrGuardrailViolation reports fail-closed lease/scope/completion guardrail failures.
var ErrGuardrailViolation = errors.New("guardrail violation")

// BootstrapGuide stores summary-first onboarding guidance for empty-instance flows.
type BootstrapGuide struct {
	Mode          string   `json:"mode"`
	Summary       string   `json:"summary"`
	WhatTillsynIs string   `json:"what_tillsyn_is"`
	Capabilities  []string `json:"capabilities"`
	NextSteps     []string `json:"next_steps"`
	Recommended   []string `json:"recommended_tools"`
	RoadmapNotice string   `json:"roadmap_notice,omitempty"`
}

// ActorLeaseTuple captures optional actor/lease fields used by guarded mutations.
type ActorLeaseTuple struct {
	ActorID         string
	ActorName       string
	ActorType       string
	AgentName       string
	AgentInstanceID string
	LeaseToken      string
	OverrideToken   string
}

// CreateProjectRequest stores transport input for project creation.
type CreateProjectRequest struct {
	Name        string
	Description string
	Kind        string
	Metadata    domain.ProjectMetadata
	Actor       ActorLeaseTuple
}

// UpdateProjectRequest stores transport input for project updates.
type UpdateProjectRequest struct {
	ProjectID   string
	Name        string
	Description string
	Kind        string
	Metadata    domain.ProjectMetadata
	Actor       ActorLeaseTuple
}

// CreateTaskRequest stores transport input for task creation.
type CreateTaskRequest struct {
	ProjectID   string
	ParentID    string
	Kind        string
	Scope       string
	ColumnID    string
	Title       string
	Description string
	Priority    string
	DueAt       string
	Labels      []string
	Metadata    domain.TaskMetadata
	Actor       ActorLeaseTuple
}

// UpdateTaskRequest stores transport input for task updates.
type UpdateTaskRequest struct {
	TaskID      string
	Title       string
	Description string
	Priority    string
	DueAt       string
	Labels      []string
	Metadata    *domain.TaskMetadata
	Actor       ActorLeaseTuple
}

// MoveTaskRequest stores transport input for task move operations.
type MoveTaskRequest struct {
	TaskID     string
	ToColumnID string
	Position   int
	Actor      ActorLeaseTuple
}

// DeleteTaskRequest stores transport input for task delete operations.
type DeleteTaskRequest struct {
	TaskID string
	Mode   string
	Actor  ActorLeaseTuple
}

// RestoreTaskRequest stores transport input for restore operations.
type RestoreTaskRequest struct {
	TaskID string
	Actor  ActorLeaseTuple
}

// ReparentTaskRequest stores transport input for parent-link updates.
type ReparentTaskRequest struct {
	TaskID   string
	ParentID string
	Actor    ActorLeaseTuple
}

// SearchTasksRequest stores transport input for search queries.
type SearchTasksRequest struct {
	ProjectID       string
	Query           string
	CrossProject    bool
	IncludeArchived bool
	States          []string
	Levels          []string
	Kinds           []string
	LabelsAny       []string
	LabelsAll       []string
	Mode            string
	Sort            string
	Limit           int
	Offset          int
}

// SearchTaskMatch stores one transport-facing search match row.
type SearchTaskMatch struct {
	Project domain.Project `json:"project"`
	Task    domain.Task    `json:"task"`
	StateID string         `json:"state_id"`
}

// UpsertKindDefinitionRequest stores transport input for kind upserts.
type UpsertKindDefinitionRequest struct {
	ID                  string
	DisplayName         string
	DescriptionMarkdown string
	AppliesTo           []string
	AllowedParentScopes []string
	PayloadSchemaJSON   string
	Template            domain.KindTemplate
}

// SetProjectAllowedKindsRequest stores transport input for allowlist updates.
type SetProjectAllowedKindsRequest struct {
	ProjectID string
	KindIDs   []string
}

// IssueCapabilityLeaseRequest stores transport input for lease issuance.
type IssueCapabilityLeaseRequest struct {
	ProjectID                 string
	ScopeType                 string
	ScopeID                   string
	Role                      string
	AgentName                 string
	AgentInstanceID           string
	ParentInstanceID          string
	AllowEqualScopeDelegation bool
	RequestedTTLSeconds       int
	OverrideToken             string
}

// HeartbeatCapabilityLeaseRequest stores transport input for lease heartbeat.
type HeartbeatCapabilityLeaseRequest struct {
	AgentInstanceID string
	LeaseToken      string
}

// RenewCapabilityLeaseRequest stores transport input for lease renewal.
type RenewCapabilityLeaseRequest struct {
	AgentInstanceID string
	LeaseToken      string
	TTLSeconds      int
}

// RevokeCapabilityLeaseRequest stores transport input for single lease revoke.
type RevokeCapabilityLeaseRequest struct {
	AgentInstanceID string
	Reason          string
}

// RevokeAllCapabilityLeasesRequest stores transport input for scoped revoke-all.
type RevokeAllCapabilityLeasesRequest struct {
	ProjectID string
	ScopeType string
	ScopeID   string
	Reason    string
}

// CreateCommentRequest stores transport input for comment creation.
type CreateCommentRequest struct {
	ProjectID    string
	TargetType   string
	TargetID     string
	Summary      string
	BodyMarkdown string
	Actor        ActorLeaseTuple
}

// CreateAuthRequestRequest stores transport input for pre-session auth request creation.
type CreateAuthRequestRequest struct {
	Path             string
	PrincipalID      string
	PrincipalType    string
	PrincipalName    string
	ClientID         string
	ClientType       string
	ClientName       string
	RequestedTTL     string
	Timeout          string
	Reason           string
	ContinuationJSON string
}

// ListAuthRequestsRequest stores transport input for auth request inventory.
type ListAuthRequestsRequest struct {
	ProjectID string
	State     string
	Limit     int
}

// AuthRequestRecord stores one transport-facing auth request row.
type AuthRequestRecord struct {
	ID                     string         `json:"id"`
	State                  string         `json:"state"`
	Path                   string         `json:"path"`
	ProjectID              string         `json:"project_id"`
	BranchID               string         `json:"branch_id,omitempty"`
	PhaseIDs               []string       `json:"phase_ids,omitempty"`
	ScopeType              string         `json:"scope_type"`
	ScopeID                string         `json:"scope_id"`
	PrincipalID            string         `json:"principal_id"`
	PrincipalType          string         `json:"principal_type"`
	PrincipalName          string         `json:"principal_name,omitempty"`
	ClientID               string         `json:"client_id"`
	ClientType             string         `json:"client_type"`
	ClientName             string         `json:"client_name,omitempty"`
	RequestedSessionTTL    string         `json:"requested_session_ttl"`
	Reason                 string         `json:"reason,omitempty"`
	Continuation           map[string]any `json:"continuation,omitempty"`
	RequestedByActor       string         `json:"requested_by_actor"`
	RequestedByType        string         `json:"requested_by_type"`
	CreatedAt              time.Time      `json:"created_at"`
	ExpiresAt              time.Time      `json:"expires_at"`
	ResolvedByActor        string         `json:"resolved_by_actor,omitempty"`
	ResolvedByType         string         `json:"resolved_by_type,omitempty"`
	ResolvedAt             *time.Time     `json:"resolved_at,omitempty"`
	ResolutionNote         string         `json:"resolution_note,omitempty"`
	IssuedSessionID        string         `json:"issued_session_id,omitempty"`
	IssuedSessionExpiresAt *time.Time     `json:"issued_session_expires_at,omitempty"`
}

// ListCommentsByTargetRequest stores transport input for comment list queries.
type ListCommentsByTargetRequest struct {
	ProjectID  string
	TargetType string
	TargetID   string
}

// CommentRecord stores transport-facing comment payloads with summary and markdown details.
type CommentRecord struct {
	ID           string    `json:"id"`
	ProjectID    string    `json:"project_id"`
	TargetType   string    `json:"target_type"`
	TargetID     string    `json:"target_id"`
	Summary      string    `json:"summary"`
	BodyMarkdown string    `json:"body_markdown"`
	ActorID      string    `json:"actor_id"`
	ActorName    string    `json:"actor_name"`
	ActorType    string    `json:"actor_type"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// BootstrapGuideReader resolves onboarding guidance for empty-instance flows.
type BootstrapGuideReader interface {
	GetBootstrapGuide(context.Context) (BootstrapGuide, error)
}

// ProjectService exposes project list/create/update operations.
type ProjectService interface {
	ListProjects(context.Context, bool) ([]domain.Project, error)
	CreateProject(context.Context, CreateProjectRequest) (domain.Project, error)
	UpdateProject(context.Context, UpdateProjectRequest) (domain.Project, error)
}

// TaskService exposes task list/mutation operations.
type TaskService interface {
	ListTasks(context.Context, string, bool) ([]domain.Task, error)
	CreateTask(context.Context, CreateTaskRequest) (domain.Task, error)
	UpdateTask(context.Context, UpdateTaskRequest) (domain.Task, error)
	MoveTask(context.Context, MoveTaskRequest) (domain.Task, error)
	DeleteTask(context.Context, DeleteTaskRequest) error
	RestoreTask(context.Context, RestoreTaskRequest) (domain.Task, error)
	ReparentTask(context.Context, ReparentTaskRequest) (domain.Task, error)
	ListChildTasks(context.Context, string, string, bool) ([]domain.Task, error)
}

// SearchService exposes cross-project and project-scoped task search.
type SearchService interface {
	SearchTasks(context.Context, SearchTasksRequest) ([]SearchTaskMatch, error)
}

// ChangeFeedService exposes project-level activity and dependency summaries.
type ChangeFeedService interface {
	ListProjectChangeEvents(context.Context, string, int) ([]domain.ChangeEvent, error)
	GetProjectDependencyRollup(context.Context, string) (domain.DependencyRollup, error)
}

// KindCatalogService exposes kind catalog and project allowlist operations.
type KindCatalogService interface {
	ListKindDefinitions(context.Context, bool) ([]domain.KindDefinition, error)
	UpsertKindDefinition(context.Context, UpsertKindDefinitionRequest) (domain.KindDefinition, error)
	SetProjectAllowedKinds(context.Context, SetProjectAllowedKindsRequest) error
	ListProjectAllowedKinds(context.Context, string) ([]string, error)
}

// CapabilityLeaseService exposes lease issuance and lifecycle operations.
type CapabilityLeaseService interface {
	IssueCapabilityLease(context.Context, IssueCapabilityLeaseRequest) (domain.CapabilityLease, error)
	HeartbeatCapabilityLease(context.Context, HeartbeatCapabilityLeaseRequest) (domain.CapabilityLease, error)
	RenewCapabilityLease(context.Context, RenewCapabilityLeaseRequest) (domain.CapabilityLease, error)
	RevokeCapabilityLease(context.Context, RevokeCapabilityLeaseRequest) (domain.CapabilityLease, error)
	RevokeAllCapabilityLeases(context.Context, RevokeAllCapabilityLeasesRequest) error
}

// CommentService exposes comment create/list operations for target scopes.
type CommentService interface {
	CreateComment(context.Context, CreateCommentRequest) (CommentRecord, error)
	ListCommentsByTarget(context.Context, ListCommentsByTargetRequest) ([]CommentRecord, error)
}

// AuthRequestService exposes pre-session auth request creation and inventory operations.
type AuthRequestService interface {
	CreateAuthRequest(context.Context, CreateAuthRequestRequest) (AuthRequestRecord, error)
	ListAuthRequests(context.Context, ListAuthRequestsRequest) ([]AuthRequestRecord, error)
	GetAuthRequest(context.Context, string) (AuthRequestRecord, error)
}

// durationFromSeconds converts positive integer seconds to a transport duration.
func durationFromSeconds(seconds int) time.Duration {
	if seconds <= 0 {
		return 0
	}
	return time.Duration(seconds) * time.Second
}
