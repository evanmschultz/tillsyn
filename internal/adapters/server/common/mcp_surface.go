package common

import (
	"context"
	"errors"
	"time"

	"github.com/evanmschultz/tillsyn/internal/domain"
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
	// AuthRequestPrincipalType carries the auth-request principal-class
	// axis (user|agent|service|steward) sourced from the authenticated
	// session. Drop 3 droplet 3.19 added it so the STEWARD owner-state-lock
	// can key on the "steward" value without collapsing it into "agent" at
	// the actor-class boundary. Empty string is the dominant case (legacy
	// callers + non-MCP test fixtures); the gate treats absent as
	// non-steward and rejects STEWARD-owned mutations accordingly.
	AuthRequestPrincipalType string
}

// CreateProjectRequest stores transport input for project creation.
type CreateProjectRequest struct {
	Name        string
	Description string
	Metadata    domain.ProjectMetadata
	Actor       ActorLeaseTuple
}

// UpdateProjectRequest stores transport input for project updates.
type UpdateProjectRequest struct {
	ProjectID   string
	Name        string
	Description string
	Metadata    domain.ProjectMetadata
	Actor       ActorLeaseTuple
}

// CreateActionItemRequest stores transport input for actionItem creation.
type CreateActionItemRequest struct {
	ProjectID string
	ParentID  string
	Kind      string
	Scope     string
	// Role optionally tags the new actionItem with a closed-enum role
	// value. Empty string is permitted; non-empty values must match the
	// closed Role enum (see domain.IsValidRole) or domain.NewActionItem
	// returns ErrInvalidRole.
	Role string
	// StructuralType MUST carry a member of the closed StructuralType
	// enum (drop|segment|confluence|droplet — see
	// domain.IsValidStructuralType). Empty string is REJECTED on create —
	// domain.NewActionItem returns ErrInvalidStructuralType. Diverges from
	// Role's permissive empty.
	StructuralType string
	// Owner optionally tags the new action item with a principal-name
	// string (e.g. "STEWARD"). Empty string is permitted; whitespace-only
	// collapses to empty. Domain primitive (per L13) — not STEWARD-specific.
	// Threaded through app.CreateActionItemInput → domain.NewActionItem in
	// droplet 3.21.
	Owner string
	// DropNumber stores the cascade drop index. Zero is permitted (treated
	// as "not a numbered drop"); positive values round-trip; negative values
	// reject with ErrInvalidDropNumber. Domain primitive — not STEWARD-
	// specific.
	DropNumber int
	// Persistent marks long-lived umbrella / anchor / perpetual-tracking
	// nodes. Default false. Domain primitive — not STEWARD-specific.
	Persistent bool
	// DevGated marks nodes whose terminal transition requires dev sign-off
	// (refinement rollups, human-verify hold points). Default false.
	// Domain primitive — not STEWARD-specific.
	DevGated bool
	// Paths optionally enumerates the new action item's write-scope file
	// paths (forward-slash, repo-root-relative). Empty slice IS the
	// meaningful zero value (no path scope) — no pointer-sentinel needed at
	// the create boundary. Threaded through app.CreateActionItemInput →
	// domain.NewActionItem in droplet 4a.5. Domain primitive per Drop 4a L3.
	Paths       []string
	ColumnID    string
	Title       string
	Description string
	Priority    string
	DueAt       string
	Labels      []string
	Metadata    domain.ActionItemMetadata
	Actor       ActorLeaseTuple
}

// UpdateActionItemRequest stores transport input for actionItem updates.
type UpdateActionItemRequest struct {
	ActionItemID string
	Title        string
	Description  string
	Priority     string
	DueAt        string
	Labels       []string
	// Role optionally updates the closed-enum role value. Empty string
	// preserves the existing value (no-op); a non-empty value must match
	// the closed Role enum or the service returns ErrInvalidRole.
	Role string
	// StructuralType optionally updates the closed-enum structural type.
	// Empty string preserves the existing value (no-op) — mirrors Role's
	// optional-on-update semantics. A non-empty value must match the
	// closed StructuralType enum or the service returns
	// ErrInvalidStructuralType.
	StructuralType string
	// Owner optionally updates the action-item owner (a free-form
	// principal-name string). nil pointer = "preserve existing" (the
	// dominant case); non-nil pointer = "set to this value". The pointer
	// shape is required by Drop 3 droplet 3.19's STEWARD owner-state-lock
	// field-level guard: without a sentinel, an absent field is
	// indistinguishable from "" and a description-only update by a
	// non-steward agent on a STEWARD-owned item would falsely trigger the
	// "Owner differs" rejection. Service-side wiring of the value lands in
	// 3.21 (alongside the rest of the Owner/DropNumber/Persistent/DevGated
	// MCP plumbing); 3.19 reads the pointer in the gate only.
	Owner *string
	// DropNumber optionally updates the action-item drop-number. nil
	// pointer = "preserve existing"; non-nil pointer = "set to this value".
	// Negative values reject with domain.ErrInvalidDropNumber at the
	// service-validation boundary. Same pointer-sentinel reasoning as
	// Owner above.
	DropNumber *int
	// Persistent optionally updates the Persistent flag. nil =
	// "preserve existing"; non-nil = "set to this value". The pointer
	// sentinel matters because Persistent=true is a load-bearing marker on
	// STEWARD anchor nodes seeded by the default template; a description-
	// only update with a value-typed bool would silently clobber it to
	// false. Domain primitive (per L13) — not STEWARD-specific. Service-
	// side wiring lands in 3.21 alongside the rest of the
	// Owner/DropNumber/Persistent/DevGated MCP plumbing.
	Persistent *bool
	// DevGated optionally updates the DevGated flag. Same pointer-sentinel
	// rationale as Persistent above.
	DevGated *bool
	// Paths optionally updates the action-item Paths slice. nil pointer =
	// "preserve existing"; non-nil pointer = "replace with this slice"
	// (empty dereferenced slice clears all declared paths). The pointer
	// sentinel matters because a description-only update by an agent must
	// NOT silently clobber a planner-set Paths declaration. Threaded
	// through app.UpdateActionItemInput; service applies via
	// domain.NormalizeActionItemPaths so the create-time trim/dedupe/
	// forward-slash rules apply equally on update. Domain primitive per
	// Drop 4a L3.
	Paths    *[]string
	Metadata *domain.ActionItemMetadata
	Actor    ActorLeaseTuple
}

// MoveActionItemRequest stores transport input for actionItem move operations.
type MoveActionItemRequest struct {
	ActionItemID string
	ToColumnID   string
	Position     int
	Actor        ActorLeaseTuple
}

// MoveActionItemStateRequest stores transport input for workflow-state transitions.
type MoveActionItemStateRequest struct {
	ActionItemID string
	State        string
	Actor        ActorLeaseTuple
}

// DeleteActionItemRequest stores transport input for actionItem delete operations.
type DeleteActionItemRequest struct {
	ActionItemID string
	Mode         string
	Actor        ActorLeaseTuple
}

// RestoreActionItemRequest stores transport input for restore operations.
type RestoreActionItemRequest struct {
	ActionItemID string
	Actor        ActorLeaseTuple
}

// ReparentActionItemRequest stores transport input for parent-link updates.
type ReparentActionItemRequest struct {
	ActionItemID string
	ParentID     string
	Actor        ActorLeaseTuple
}

// SearchActionItemsRequest stores transport input for search queries.
type SearchActionItemsRequest struct {
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

// SearchActionItemMatch stores one transport-facing search match row.
type SearchActionItemMatch struct {
	Project                   domain.Project    `json:"project"`
	ActionItem                domain.ActionItem `json:"actionItem"`
	StateID                   string            `json:"state_id"`
	EmbeddingSubjectType      string            `json:"embedding_subject_type,omitempty"`
	EmbeddingSubjectID        string            `json:"embedding_subject_id,omitempty"`
	EmbeddingStatus           string            `json:"embedding_status,omitempty"`
	EmbeddingUpdatedAt        *time.Time        `json:"embedding_updated_at,omitempty"`
	EmbeddingStaleReason      string            `json:"embedding_stale_reason,omitempty"`
	EmbeddingLastErrorSummary string            `json:"embedding_last_error_summary,omitempty"`
	SemanticScore             float64           `json:"semantic_score,omitempty"`
	UsedSemantic              bool              `json:"used_semantic,omitempty"`
}

// SearchActionItemsResult stores search rows plus execution metadata.
type SearchActionItemsResult struct {
	Matches                []SearchActionItemMatch `json:"matches"`
	RequestedMode          string                  `json:"requested_mode,omitempty"`
	EffectiveMode          string                  `json:"effective_mode,omitempty"`
	FallbackReason         string                  `json:"fallback_reason,omitempty"`
	SemanticAvailable      bool                    `json:"semantic_available"`
	SemanticCandidateCount int                     `json:"semantic_candidate_count"`
	EmbeddingSummary       EmbeddingSummary        `json:"embedding_summary"`
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
//
// Per Drop 3 droplet 3.15 the legacy AllowedParentScopes + Template fields
// were removed; nesting rules now flow through the project's baked
// KindCatalog. The till.kind operation=upsert MCP wire surface and the
// till.upsert_kind_definition legacy alias were also deleted in this
// droplet — only programmatic Service.UpsertKindDefinition callers remain.
type UpsertKindDefinitionRequest struct {
	ID                  string
	DisplayName         string
	DescriptionMarkdown string
	AppliesTo           []string
	PayloadSchemaJSON   string
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
	ProjectID   string
	BranchID    string
	ScopeType   string
	ScopeID     string
	Statuses    []string
	Limit       int
	WaitTimeout string
}

// CreateAuthRequestRequest stores transport input for pre-session auth request creation.
type CreateAuthRequestRequest struct {
	Path                string
	PrincipalID         string
	PrincipalType       string
	PrincipalRole       string
	PrincipalName       string
	ActingSessionID     string
	ActingSessionSecret string
	RequestedByActor    string
	RequestedByType     string
	RequesterClientID   string
	ClientID            string
	ClientType          string
	ClientName          string
	RequestedTTL        string
	Timeout             string
	Reason              string
	ContinuationJSON    string
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
	AuthContextID string            `json:"auth_context_id,omitempty"`
	Waiting       bool              `json:"waiting,omitempty"`
}

// ListAuthSessionsRequest stores transport input for auth-session inventory queries.
type ListAuthSessionsRequest struct {
	ProjectID           string
	SessionID           string
	PrincipalID         string
	ClientID            string
	State               string
	Limit               int
	ActingSessionID     string
	ActingSessionSecret string
}

// ValidateAuthSessionRequest stores transport input for session validation.
type ValidateAuthSessionRequest struct {
	SessionID     string
	SessionSecret string
}

// CheckAuthSessionGovernanceRequest stores transport input for non-destructive session-governance checks.
type CheckAuthSessionGovernanceRequest struct {
	SessionID           string
	ActingSessionID     string
	ActingSessionSecret string
}

// RevokeAuthSessionRequest stores transport input for session revocation.
type RevokeAuthSessionRequest struct {
	SessionID           string
	Reason              string
	ActingSessionID     string
	ActingSessionSecret string
}

// AuthSessionRecord stores one transport-facing auth session row.
type AuthSessionRecord struct {
	SessionID        string     `json:"session_id"`
	State            string     `json:"state"`
	ProjectID        string     `json:"project_id,omitempty"`
	AuthRequestID    string     `json:"auth_request_id,omitempty"`
	ApprovedPath     string     `json:"approved_path,omitempty"`
	PrincipalID      string     `json:"principal_id"`
	PrincipalType    string     `json:"principal_type"`
	PrincipalRole    string     `json:"principal_role,omitempty"`
	PrincipalName    string     `json:"principal_name,omitempty"`
	ClientID         string     `json:"client_id"`
	ClientType       string     `json:"client_type"`
	ClientName       string     `json:"client_name,omitempty"`
	IssuedAt         time.Time  `json:"issued_at"`
	ExpiresAt        time.Time  `json:"expires_at"`
	LastValidatedAt  *time.Time `json:"last_validated_at,omitempty"`
	RevokedAt        *time.Time `json:"revoked_at,omitempty"`
	RevocationReason string     `json:"revocation_reason,omitempty"`
	AuthContextID    string     `json:"auth_context_id,omitempty"`
}

// AuthSessionGovernanceCheckResult stores one non-destructive session-governance decision.
type AuthSessionGovernanceCheckResult struct {
	Authorized          bool              `json:"authorized"`
	DecisionReason      string            `json:"decision_reason,omitempty"`
	ActingSessionID     string            `json:"acting_session_id,omitempty"`
	ActingPrincipalID   string            `json:"acting_principal_id,omitempty"`
	ActingPrincipalRole string            `json:"acting_principal_role,omitempty"`
	ActingApprovedPath  string            `json:"acting_approved_path,omitempty"`
	TargetSession       AuthSessionRecord `json:"target_session"`
}

// ListCommentsByTargetRequest stores transport input for comment list queries.
type ListCommentsByTargetRequest struct {
	ProjectID   string
	TargetType  string
	TargetID    string
	WaitTimeout string
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

// ActionItemService exposes actionItem list/mutation operations.
type ActionItemService interface {
	GetActionItem(context.Context, string) (domain.ActionItem, error)
	ListActionItems(context.Context, string, bool) ([]domain.ActionItem, error)
	CreateActionItem(context.Context, CreateActionItemRequest) (domain.ActionItem, error)
	UpdateActionItem(context.Context, UpdateActionItemRequest) (domain.ActionItem, error)
	MoveActionItem(context.Context, MoveActionItemRequest) (domain.ActionItem, error)
	MoveActionItemState(context.Context, MoveActionItemStateRequest) (domain.ActionItem, error)
	DeleteActionItem(context.Context, DeleteActionItemRequest) error
	RestoreActionItem(context.Context, RestoreActionItemRequest) (domain.ActionItem, error)
	ReparentActionItem(context.Context, ReparentActionItemRequest) (domain.ActionItem, error)
	ListChildActionItems(context.Context, string, string, bool) ([]domain.ActionItem, error)
	ResolveActionItemID(ctx context.Context, projectID, idOrDotted string) (string, error)
	GetProjectBySlug(ctx context.Context, slug string) (domain.Project, error)
}

// SearchService exposes cross-project and project-scoped actionItem search.
type SearchService interface {
	SearchActionItems(context.Context, SearchActionItemsRequest) (SearchActionItemsResult, error)
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
	ListAuthSessions(context.Context, ListAuthSessionsRequest) ([]AuthSessionRecord, error)
	ValidateAuthSession(context.Context, ValidateAuthSessionRequest) (AuthSessionRecord, error)
	CheckAuthSessionGovernance(context.Context, CheckAuthSessionGovernanceRequest) (AuthSessionGovernanceCheckResult, error)
	RevokeAuthSession(context.Context, RevokeAuthSessionRequest) (AuthSessionRecord, error)
}

// durationFromSeconds converts positive integer seconds to a transport duration.
func durationFromSeconds(seconds int) time.Duration {
	if seconds <= 0 {
		return 0
	}
	return time.Duration(seconds) * time.Second
}
