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

// BootstrapGuide stores summary-first onboarding guidance for empty-instance and pre-approval flows.
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
	Name              string
	Description       string
	Kind              string
	TemplateLibraryID string
	Metadata          domain.ProjectMetadata
	Actor             ActorLeaseTuple
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

// MoveTaskStateRequest stores transport input for workflow-state transitions.
type MoveTaskStateRequest struct {
	TaskID string
	State  string
	Actor  ActorLeaseTuple
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
	Project                   domain.Project `json:"project"`
	Task                      domain.Task    `json:"task"`
	StateID                   string         `json:"state_id"`
	EmbeddingSubjectType      string         `json:"embedding_subject_type,omitempty"`
	EmbeddingSubjectID        string         `json:"embedding_subject_id,omitempty"`
	EmbeddingStatus           string         `json:"embedding_status,omitempty"`
	EmbeddingUpdatedAt        *time.Time     `json:"embedding_updated_at,omitempty"`
	EmbeddingStaleReason      string         `json:"embedding_stale_reason,omitempty"`
	EmbeddingLastErrorSummary string         `json:"embedding_last_error_summary,omitempty"`
	SemanticScore             float64        `json:"semantic_score,omitempty"`
	UsedSemantic              bool           `json:"used_semantic,omitempty"`
}

// SearchTasksResult stores search rows plus execution metadata.
type SearchTasksResult struct {
	Matches                []SearchTaskMatch `json:"matches"`
	RequestedMode          string            `json:"requested_mode,omitempty"`
	EffectiveMode          string            `json:"effective_mode,omitempty"`
	FallbackReason         string            `json:"fallback_reason,omitempty"`
	SemanticAvailable      bool              `json:"semantic_available"`
	SemanticCandidateCount int               `json:"semantic_candidate_count"`
	EmbeddingSummary       EmbeddingSummary  `json:"embedding_summary"`
}

// EmbeddingSummary stores aggregate lifecycle counts for transport surfaces.
type EmbeddingSummary struct {
	SubjectType  string   `json:"subject_type"`
	ProjectIDs   []string `json:"project_ids,omitempty"`
	PendingCount int      `json:"pending_count"`
	RunningCount int      `json:"running_count"`
	ReadyCount   int      `json:"ready_count"`
	FailedCount  int      `json:"failed_count"`
	StaleCount   int      `json:"stale_count"`
}

// EmbeddingStatusRow stores one transport-facing lifecycle inventory row.
type EmbeddingStatusRow struct {
	SubjectType      string     `json:"subject_type"`
	SubjectID        string     `json:"subject_id"`
	ProjectID        string     `json:"project_id"`
	Status           string     `json:"status"`
	ModelSignature   string     `json:"model_signature"`
	NextAttemptAt    *time.Time `json:"next_attempt_at,omitempty"`
	LastStartedAt    *time.Time `json:"last_started_at,omitempty"`
	LastSucceededAt  *time.Time `json:"last_succeeded_at,omitempty"`
	LastFailedAt     *time.Time `json:"last_failed_at,omitempty"`
	LastErrorSummary string     `json:"last_error_summary,omitempty"`
	StaleReason      string     `json:"stale_reason,omitempty"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

// EmbeddingsStatusRequest stores transport input for embeddings inventory queries.
type EmbeddingsStatusRequest struct {
	ProjectID       string
	CrossProject    bool
	IncludeArchived bool
	Statuses        []string
	Limit           int
}

// EmbeddingsStatusResult stores transport-facing lifecycle inventory results.
type EmbeddingsStatusResult struct {
	ProjectIDs         []string             `json:"project_ids,omitempty"`
	RuntimeOperational bool                 `json:"runtime_operational"`
	Summary            EmbeddingSummary     `json:"summary"`
	Rows               []EmbeddingStatusRow `json:"rows"`
}

// ReindexEmbeddingsRequest stores transport input for explicit reindex requests.
type ReindexEmbeddingsRequest struct {
	ProjectID        string
	CrossProject     bool
	IncludeArchived  bool
	Force            bool
	Wait             bool
	WaitTimeout      time.Duration
	WaitPollInterval time.Duration
}

// ReindexEmbeddingsResult stores transport-facing reindex outcomes.
type ReindexEmbeddingsResult struct {
	TargetProjects []string `json:"target_projects,omitempty"`
	ScannedCount   int      `json:"scanned_count"`
	QueuedCount    int      `json:"queued_count"`
	ReadyCount     int      `json:"ready_count"`
	FailedCount    int      `json:"failed_count"`
	StaleCount     int      `json:"stale_count"`
	RunningCount   int      `json:"running_count"`
	PendingCount   int      `json:"pending_count"`
	Completed      bool     `json:"completed"`
	TimedOut       bool     `json:"timed_out"`
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

// ListTemplateLibrariesRequest stores transport input for template-library listing.
type ListTemplateLibrariesRequest struct {
	Scope     domain.TemplateLibraryScope  `json:"scope,omitempty"`
	ProjectID string                       `json:"project_id,omitempty"`
	Status    domain.TemplateLibraryStatus `json:"status,omitempty"`
}

// UpsertTemplateChildRuleRequest stores transport input for one nested template child rule.
type UpsertTemplateChildRuleRequest struct {
	ID                        string                     `json:"id,omitempty"`
	Position                  int                        `json:"position"`
	ChildScopeLevel           domain.KindAppliesTo       `json:"child_scope_level"`
	ChildKindID               domain.KindID              `json:"child_kind_id"`
	TitleTemplate             string                     `json:"title_template"`
	DescriptionTemplate       string                     `json:"description_template,omitempty"`
	ResponsibleActorKind      domain.TemplateActorKind   `json:"responsible_actor_kind"`
	EditableByActorKinds      []domain.TemplateActorKind `json:"editable_by_actor_kinds,omitempty"`
	CompletableByActorKinds   []domain.TemplateActorKind `json:"completable_by_actor_kinds,omitempty"`
	OrchestratorMayComplete   bool                       `json:"orchestrator_may_complete,omitempty"`
	RequiredForParentDone     bool                       `json:"required_for_parent_done,omitempty"`
	RequiredForContainingDone bool                       `json:"required_for_containing_done,omitempty"`
}

// UpsertNodeTemplateRequest stores transport input for one nested node template.
type UpsertNodeTemplateRequest struct {
	ID                      string                           `json:"id,omitempty"`
	ScopeLevel              domain.KindAppliesTo             `json:"scope_level"`
	NodeKindID              domain.KindID                    `json:"node_kind_id"`
	DisplayName             string                           `json:"display_name"`
	DescriptionMarkdown     string                           `json:"description_markdown,omitempty"`
	ProjectMetadataDefaults *domain.ProjectMetadata          `json:"project_metadata_defaults,omitempty"`
	TaskMetadataDefaults    *domain.TaskMetadata             `json:"task_metadata_defaults,omitempty"`
	ChildRules              []UpsertTemplateChildRuleRequest `json:"child_rules,omitempty"`
}

// UpsertTemplateLibraryRequest stores transport input for one template-library upsert.
type UpsertTemplateLibraryRequest struct {
	ID              string                       `json:"id,omitempty"`
	Scope           domain.TemplateLibraryScope  `json:"scope"`
	ProjectID       string                       `json:"project_id,omitempty"`
	Name            string                       `json:"name"`
	Description     string                       `json:"description,omitempty"`
	Status          domain.TemplateLibraryStatus `json:"status"`
	SourceLibraryID string                       `json:"source_library_id,omitempty"`
	BuiltinManaged  bool                         `json:"builtin_managed,omitempty"`
	BuiltinSource   string                       `json:"builtin_source,omitempty"`
	BuiltinVersion  string                       `json:"builtin_version,omitempty"`
	NodeTemplates   []UpsertNodeTemplateRequest  `json:"node_templates,omitempty"`
}

// EnsureBuiltinTemplateLibraryRequest stores transport input for one explicit builtin install or refresh.
type EnsureBuiltinTemplateLibraryRequest struct {
	LibraryID string `json:"library_id,omitempty"`
}

// BindProjectTemplateLibraryRequest stores transport input for project-to-library binding.
type BindProjectTemplateLibraryRequest struct {
	ProjectID string `json:"project_id"`
	LibraryID string `json:"library_id"`
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

// ListCapabilityLeasesRequest stores transport input for scoped lease inventory.
type ListCapabilityLeasesRequest struct {
	ProjectID      string
	ScopeType      string
	ScopeID        string
	IncludeRevoked bool
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

// CreateHandoffRequest stores transport input for durable handoff creation.
type CreateHandoffRequest struct {
	ProjectID       string
	BranchID        string
	ScopeType       string
	ScopeID         string
	SourceRole      string
	TargetBranchID  string
	TargetScopeType string
	TargetScopeID   string
	TargetRole      string
	Status          string
	Summary         string
	NextAction      string
	MissingEvidence []string
	RelatedRefs     []string
	Actor           ActorLeaseTuple
}

// UpdateHandoffRequest stores transport input for durable handoff updates.
type UpdateHandoffRequest struct {
	HandoffID       string
	Status          string
	SourceRole      string
	TargetBranchID  string
	TargetScopeType string
	TargetScopeID   string
	TargetRole      string
	Summary         string
	NextAction      string
	MissingEvidence []string
	RelatedRefs     []string
	ResolutionNote  string
	Actor           ActorLeaseTuple
}

// ListHandoffsRequest stores transport input for scoped handoff inventory.
type ListHandoffsRequest struct {
	ProjectID string
	BranchID  string
	ScopeType string
	ScopeID   string
	Statuses  []string
	Limit     int
}

// CreateAuthRequestRequest stores transport input for pre-session auth request creation.
type CreateAuthRequestRequest struct {
	Path              string
	PrincipalID       string
	PrincipalType     string
	PrincipalRole     string
	PrincipalName     string
	RequestedByActor  string
	RequestedByType   string
	RequesterClientID string
	ClientID          string
	ClientType        string
	ClientName        string
	RequestedTTL      string
	Timeout           string
	Reason            string
	ContinuationJSON  string
}

// ListAuthRequestsRequest stores transport input for auth request inventory.
type ListAuthRequestsRequest struct {
	ProjectID string
	State     string
	Limit     int
}

// ClaimAuthRequestRequest stores transport input for continuation-based auth-request resume by an approved claimant.
type ClaimAuthRequestRequest struct {
	RequestID   string
	ResumeToken string
	PrincipalID string
	ClientID    string
	WaitTimeout string
}

// CancelAuthRequestRequest stores transport input for requester-bound auth-request cleanup.
type CancelAuthRequestRequest struct {
	RequestID      string
	ResumeToken    string
	PrincipalID    string
	ClientID       string
	ResolutionNote string
}

// AuthRequestRecord stores one transport-facing auth request row.
type AuthRequestRecord struct {
	ID                  string   `json:"id"`
	State               string   `json:"state"`
	Path                string   `json:"path"`
	ApprovedPath        string   `json:"approved_path,omitempty"`
	ProjectID           string   `json:"project_id"`
	BranchID            string   `json:"branch_id,omitempty"`
	PhaseIDs            []string `json:"phase_ids,omitempty"`
	ScopeType           string   `json:"scope_type"`
	ScopeID             string   `json:"scope_id"`
	PrincipalID         string   `json:"principal_id"`
	PrincipalType       string   `json:"principal_type"`
	PrincipalRole       string   `json:"principal_role,omitempty"`
	PrincipalName       string   `json:"principal_name,omitempty"`
	ClientID            string   `json:"client_id"`
	ClientType          string   `json:"client_type"`
	ClientName          string   `json:"client_name,omitempty"`
	RequestedSessionTTL string   `json:"requested_session_ttl"`
	ApprovedSessionTTL  string   `json:"approved_session_ttl,omitempty"`
	Reason              string   `json:"reason,omitempty"`
	HasContinuation     bool     `json:"has_continuation,omitempty"`
	// Continuation keeps private ownership proof for claim/cancel validation while staying out of tool JSON.
	Continuation           map[string]any `json:"-"`
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

// AuthRequestClaimResult stores one requester-visible auth request state plus approved session secret material.
type AuthRequestClaimResult struct {
	Request       AuthRequestRecord `json:"request"`
	SessionSecret string            `json:"session_secret,omitempty"`
	Waiting       bool              `json:"waiting,omitempty"`
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

// BootstrapGuideReader resolves onboarding guidance for empty-instance and pre-approval flows.
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
	GetTask(context.Context, string) (domain.Task, error)
	ListTasks(context.Context, string, bool) ([]domain.Task, error)
	CreateTask(context.Context, CreateTaskRequest) (domain.Task, error)
	UpdateTask(context.Context, UpdateTaskRequest) (domain.Task, error)
	MoveTask(context.Context, MoveTaskRequest) (domain.Task, error)
	MoveTaskState(context.Context, MoveTaskStateRequest) (domain.Task, error)
	DeleteTask(context.Context, DeleteTaskRequest) error
	RestoreTask(context.Context, RestoreTaskRequest) (domain.Task, error)
	ReparentTask(context.Context, ReparentTaskRequest) (domain.Task, error)
	ListChildTasks(context.Context, string, string, bool) ([]domain.Task, error)
}

// SearchService exposes cross-project and project-scoped task search.
type SearchService interface {
	SearchTasks(context.Context, SearchTasksRequest) (SearchTasksResult, error)
}

// EmbeddingsService exposes operator-facing lifecycle inventory and reindex actions.
type EmbeddingsService interface {
	GetEmbeddingsStatus(context.Context, EmbeddingsStatusRequest) (EmbeddingsStatusResult, error)
	ReindexEmbeddings(context.Context, ReindexEmbeddingsRequest) (ReindexEmbeddingsResult, error)
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

// TemplateLibraryService exposes template-library and node-contract inspection plus project binding operations.
type TemplateLibraryService interface {
	ListTemplateLibraries(context.Context, ListTemplateLibrariesRequest) ([]domain.TemplateLibrary, error)
	GetTemplateLibrary(context.Context, string) (domain.TemplateLibrary, error)
	GetBuiltinTemplateLibraryStatus(context.Context, string) (domain.BuiltinTemplateLibraryStatus, error)
	EnsureBuiltinTemplateLibrary(context.Context, EnsureBuiltinTemplateLibraryRequest) (domain.BuiltinTemplateLibraryEnsureResult, error)
	UpsertTemplateLibrary(context.Context, UpsertTemplateLibraryRequest) (domain.TemplateLibrary, error)
	BindProjectTemplateLibrary(context.Context, BindProjectTemplateLibraryRequest) (domain.ProjectTemplateBinding, error)
	GetProjectTemplateBinding(context.Context, string) (domain.ProjectTemplateBinding, error)
	GetNodeContractSnapshot(context.Context, string) (domain.NodeContractSnapshot, error)
}

// CapabilityLeaseService exposes lease issuance and lifecycle operations.
type CapabilityLeaseService interface {
	ListCapabilityLeases(context.Context, ListCapabilityLeasesRequest) ([]domain.CapabilityLease, error)
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

// HandoffService exposes durable handoff create/read/update/list operations.
type HandoffService interface {
	CreateHandoff(context.Context, CreateHandoffRequest) (domain.Handoff, error)
	GetHandoff(context.Context, string) (domain.Handoff, error)
	ListHandoffs(context.Context, ListHandoffsRequest) ([]domain.Handoff, error)
	UpdateHandoff(context.Context, UpdateHandoffRequest) (domain.Handoff, error)
}

// AuthRequestService exposes pre-session auth request creation and inventory operations.
type AuthRequestService interface {
	CreateAuthRequest(context.Context, CreateAuthRequestRequest) (AuthRequestRecord, error)
	ListAuthRequests(context.Context, ListAuthRequestsRequest) ([]AuthRequestRecord, error)
	GetAuthRequest(context.Context, string) (AuthRequestRecord, error)
	ClaimAuthRequest(context.Context, ClaimAuthRequestRequest) (AuthRequestClaimResult, error)
	CancelAuthRequest(context.Context, CancelAuthRequestRequest) (AuthRequestRecord, error)
}

// durationFromSeconds converts positive integer seconds to a transport duration.
func durationFromSeconds(seconds int) time.Duration {
	if seconds <= 0 {
		return 0
	}
	return time.Duration(seconds) * time.Second
}
