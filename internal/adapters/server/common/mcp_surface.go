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
	WhatKanIs     string   `json:"what_kan_is"`
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
	BodyMarkdown string
	Actor        ActorLeaseTuple
}

// ListCommentsByTargetRequest stores transport input for comment list queries.
type ListCommentsByTargetRequest struct {
	ProjectID  string
	TargetType string
	TargetID   string
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
	CreateComment(context.Context, CreateCommentRequest) (domain.Comment, error)
	ListCommentsByTarget(context.Context, ListCommentsByTargetRequest) ([]domain.Comment, error)
}

// durationFromSeconds converts positive integer seconds to a transport duration.
func durationFromSeconds(seconds int) time.Duration {
	if seconds <= 0 {
		return 0
	}
	return time.Duration(seconds) * time.Second
}
