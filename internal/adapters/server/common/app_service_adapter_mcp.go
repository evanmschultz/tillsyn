package common

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/hylla/tillsyn/internal/app"
	"github.com/hylla/tillsyn/internal/domain"
)

// GetBootstrapGuide returns summary-first onboarding guidance for empty-instance flows.
func (a *AppServiceAdapter) GetBootstrapGuide(_ context.Context) (BootstrapGuide, error) {
	if a == nil || a.service == nil {
		return BootstrapGuide{}, fmt.Errorf("app service adapter is not configured: %w", ErrInvalidCaptureStateRequest)
	}
	return BootstrapGuide{
		Mode:          "bootstrap_required",
		Summary:       "No project context exists yet. Start by creating your first project and then capture state.",
		WhatTillsynIs: "Tillsyn is a strict task/state planner with level-scoped work (project|branch|phase|task|subtask), guardrailed mutations, and summary-first recovery context.",
		Capabilities: []string{
			"Level-scoped capture_state for summary-first recovery",
			"Task graph operations across branch/phase/task/subtask scopes",
			"Attention/blocker signaling with user-action visibility",
			"Kind catalog and template-driven child/checklist auto-actions",
			"Capability lease issuance and guardrailed non-user mutations",
		},
		NextSteps: []string{
			"Create a project with till.create_project",
			"Create level-scoped work items with till.create_task",
			"Call till.capture_state to reorient and continue safely",
		},
		Recommended: []string{
			"till.list_projects",
			"till.create_project",
			"till.create_task",
			"till.capture_state",
		},
		RoadmapNotice: "Import/export transport-closure and advanced conflict tooling remain roadmap-only for this wave.",
	}, nil
}

// ListProjects returns project rows from app-level APIs.
func (a *AppServiceAdapter) ListProjects(ctx context.Context, includeArchived bool) ([]domain.Project, error) {
	if a == nil || a.service == nil {
		return nil, fmt.Errorf("app service adapter is not configured: %w", ErrInvalidCaptureStateRequest)
	}
	projects, err := a.service.ListProjects(ctx, includeArchived)
	if err != nil {
		return nil, mapAppError("list projects", err)
	}
	return projects, nil
}

// CreateAuthRequest creates one persisted pre-session auth request through app-level APIs.
func (a *AppServiceAdapter) CreateAuthRequest(ctx context.Context, in CreateAuthRequestRequest) (AuthRequestRecord, error) {
	if a == nil || a.service == nil {
		return AuthRequestRecord{}, fmt.Errorf("app service adapter is not configured: %w", ErrInvalidCaptureStateRequest)
	}
	requestedTTL, err := parseOptionalDurationString(in.RequestedTTL, "requested_ttl")
	if err != nil {
		return AuthRequestRecord{}, err
	}
	timeout, err := parseOptionalDurationString(in.Timeout, "timeout")
	if err != nil {
		return AuthRequestRecord{}, err
	}
	continuation, err := parseContinuationJSON(in.ContinuationJSON)
	if err != nil {
		return AuthRequestRecord{}, err
	}
	request, err := a.service.CreateAuthRequest(ctx, app.CreateAuthRequestInput{
		Path:                strings.TrimSpace(in.Path),
		PrincipalID:         strings.TrimSpace(in.PrincipalID),
		PrincipalType:       strings.TrimSpace(in.PrincipalType),
		PrincipalRole:       strings.TrimSpace(in.PrincipalRole),
		PrincipalName:       strings.TrimSpace(in.PrincipalName),
		ClientID:            strings.TrimSpace(in.ClientID),
		ClientType:          strings.TrimSpace(in.ClientType),
		ClientName:          strings.TrimSpace(in.ClientName),
		RequesterClientID:   requesterClientID(in),
		RequestedSessionTTL: requestedTTL,
		Reason:              strings.TrimSpace(in.Reason),
		Continuation:        continuation,
		RequestedBy:         firstNonEmptyRequestedBy(in.RequestedByActor, in.PrincipalID),
		RequestedType:       requestedActorType(in.RequestedByType, in.PrincipalType),
		Timeout:             timeout,
	})
	if err != nil {
		return AuthRequestRecord{}, mapAppError("create auth request", err)
	}
	return mapAuthRequestRecord(request), nil
}

// ListAuthRequests returns auth-request inventory rows from app-level APIs.
func (a *AppServiceAdapter) ListAuthRequests(ctx context.Context, in ListAuthRequestsRequest) ([]AuthRequestRecord, error) {
	if a == nil || a.service == nil {
		return nil, fmt.Errorf("app service adapter is not configured: %w", ErrInvalidCaptureStateRequest)
	}
	requests, err := a.service.ListAuthRequests(ctx, domain.AuthRequestListFilter{
		ProjectID: strings.TrimSpace(in.ProjectID),
		State:     domain.AuthRequestState(strings.TrimSpace(in.State)),
		Limit:     in.Limit,
	})
	if err != nil {
		return nil, mapAppError("list auth requests", err)
	}
	out := make([]AuthRequestRecord, 0, len(requests))
	for _, request := range requests {
		out = append(out, mapAuthRequestRecord(request))
	}
	return out, nil
}

// GetAuthRequest returns one auth request by id through app-level APIs.
func (a *AppServiceAdapter) GetAuthRequest(ctx context.Context, requestID string) (AuthRequestRecord, error) {
	if a == nil || a.service == nil {
		return AuthRequestRecord{}, fmt.Errorf("app service adapter is not configured: %w", ErrInvalidCaptureStateRequest)
	}
	requestID = strings.TrimSpace(requestID)
	if requestID == "" {
		return AuthRequestRecord{}, fmt.Errorf("request_id is required: %w", ErrInvalidCaptureStateRequest)
	}
	request, err := a.service.GetAuthRequest(ctx, requestID)
	if err != nil {
		return AuthRequestRecord{}, mapAppError("get auth request", err)
	}
	return mapAuthRequestRecord(request), nil
}

// ClaimAuthRequest returns one requester-visible auth request state and approved session secret through continuation proof.
func (a *AppServiceAdapter) ClaimAuthRequest(ctx context.Context, in ClaimAuthRequestRequest) (AuthRequestClaimResult, error) {
	if a == nil || a.service == nil {
		return AuthRequestClaimResult{}, fmt.Errorf("app service adapter is not configured: %w", ErrInvalidCaptureStateRequest)
	}
	requestID := strings.TrimSpace(in.RequestID)
	if requestID == "" {
		return AuthRequestClaimResult{}, fmt.Errorf("request_id is required: %w", ErrInvalidCaptureStateRequest)
	}
	waitTimeout, err := parseOptionalDurationString(in.WaitTimeout, "wait_timeout")
	if err != nil {
		return AuthRequestClaimResult{}, err
	}
	result, err := a.service.ClaimAuthRequest(ctx, app.ClaimAuthRequestInput{
		RequestID:   requestID,
		ResumeToken: strings.TrimSpace(in.ResumeToken),
		PrincipalID: strings.TrimSpace(in.PrincipalID),
		ClientID:    strings.TrimSpace(in.ClientID),
		WaitTimeout: waitTimeout,
	})
	if err != nil {
		return AuthRequestClaimResult{}, mapAppError("claim auth request", err)
	}
	return AuthRequestClaimResult{
		Request:       mapAuthRequestRecord(result.Request),
		SessionSecret: result.SessionSecret,
		Waiting:       result.Waiting,
	}, nil
}

// CreateProject creates one project with optional kind and metadata.
func (a *AppServiceAdapter) CreateProject(ctx context.Context, in CreateProjectRequest) (domain.Project, error) {
	if a == nil || a.service == nil {
		return domain.Project{}, fmt.Errorf("app service adapter is not configured: %w", ErrInvalidCaptureStateRequest)
	}
	ctx, actorType, err := withMutationGuardContext(ctx, in.Actor)
	if err != nil {
		return domain.Project{}, err
	}
	actorID, actorName := deriveMutationActorIdentity(in.Actor)
	project, err := a.service.CreateProjectWithMetadata(ctx, app.CreateProjectInput{
		Name:          strings.TrimSpace(in.Name),
		Description:   strings.TrimSpace(in.Description),
		Kind:          domain.KindID(strings.TrimSpace(in.Kind)),
		Metadata:      in.Metadata,
		UpdatedBy:     actorID,
		UpdatedByName: actorName,
		UpdatedType:   actorType,
	})
	if err != nil {
		return domain.Project{}, mapAppError("create project", err)
	}
	return project, nil
}

// UpdateProject updates one project.
func (a *AppServiceAdapter) UpdateProject(ctx context.Context, in UpdateProjectRequest) (domain.Project, error) {
	if a == nil || a.service == nil {
		return domain.Project{}, fmt.Errorf("app service adapter is not configured: %w", ErrInvalidCaptureStateRequest)
	}
	ctx, actorType, err := withMutationGuardContext(ctx, in.Actor)
	if err != nil {
		return domain.Project{}, err
	}
	actorID, actorName := deriveMutationActorIdentity(in.Actor)
	project, err := a.service.UpdateProject(ctx, app.UpdateProjectInput{
		ProjectID:     strings.TrimSpace(in.ProjectID),
		Name:          strings.TrimSpace(in.Name),
		Description:   strings.TrimSpace(in.Description),
		Kind:          domain.KindID(strings.TrimSpace(in.Kind)),
		Metadata:      in.Metadata,
		UpdatedBy:     actorID,
		UpdatedByName: actorName,
		UpdatedType:   actorType,
	})
	if err != nil {
		return domain.Project{}, mapAppError("update project", err)
	}
	return project, nil
}

// ListTasks returns tasks for one project with deterministic ordering from app-level APIs.
func (a *AppServiceAdapter) ListTasks(ctx context.Context, projectID string, includeArchived bool) ([]domain.Task, error) {
	if a == nil || a.service == nil {
		return nil, fmt.Errorf("app service adapter is not configured: %w", ErrInvalidCaptureStateRequest)
	}
	tasks, err := a.service.ListTasks(ctx, strings.TrimSpace(projectID), includeArchived)
	if err != nil {
		return nil, mapAppError("list tasks", err)
	}
	return tasks, nil
}

// CreateTask creates one level-scoped task/work item.
func (a *AppServiceAdapter) CreateTask(ctx context.Context, in CreateTaskRequest) (domain.Task, error) {
	if a == nil || a.service == nil {
		return domain.Task{}, fmt.Errorf("app service adapter is not configured: %w", ErrInvalidCaptureStateRequest)
	}
	dueAt, err := parseOptionalRFC3339(in.DueAt)
	if err != nil {
		return domain.Task{}, err
	}
	ctx, actorType, err := withMutationGuardContext(ctx, in.Actor)
	if err != nil {
		return domain.Task{}, err
	}
	actorID, actorName := deriveMutationActorIdentity(in.Actor)
	task, err := a.service.CreateTask(ctx, app.CreateTaskInput{
		ProjectID:      strings.TrimSpace(in.ProjectID),
		ParentID:       strings.TrimSpace(in.ParentID),
		Kind:           domain.WorkKind(strings.TrimSpace(in.Kind)),
		Scope:          domain.KindAppliesTo(strings.TrimSpace(in.Scope)),
		ColumnID:       strings.TrimSpace(in.ColumnID),
		Title:          strings.TrimSpace(in.Title),
		Description:    strings.TrimSpace(in.Description),
		Priority:       domain.Priority(strings.TrimSpace(strings.ToLower(in.Priority))),
		DueAt:          dueAt,
		Labels:         append([]string(nil), in.Labels...),
		Metadata:       in.Metadata,
		CreatedByActor: actorID,
		CreatedByName:  actorName,
		UpdatedByActor: actorID,
		UpdatedByName:  actorName,
		UpdatedByType:  actorType,
	})
	if err != nil {
		return domain.Task{}, mapAppError("create task", err)
	}
	return task, nil
}

// UpdateTask updates one task/work-item row.
func (a *AppServiceAdapter) UpdateTask(ctx context.Context, in UpdateTaskRequest) (domain.Task, error) {
	if a == nil || a.service == nil {
		return domain.Task{}, fmt.Errorf("app service adapter is not configured: %w", ErrInvalidCaptureStateRequest)
	}
	dueAt, err := parseOptionalRFC3339(in.DueAt)
	if err != nil {
		return domain.Task{}, err
	}
	ctx, actorType, err := withMutationGuardContext(ctx, in.Actor)
	if err != nil {
		return domain.Task{}, err
	}
	actorID, actorName := deriveMutationActorIdentity(in.Actor)
	task, err := a.service.UpdateTask(ctx, app.UpdateTaskInput{
		TaskID:        strings.TrimSpace(in.TaskID),
		Title:         strings.TrimSpace(in.Title),
		Description:   strings.TrimSpace(in.Description),
		Priority:      domain.Priority(strings.TrimSpace(strings.ToLower(in.Priority))),
		DueAt:         dueAt,
		Labels:        append([]string(nil), in.Labels...),
		Metadata:      in.Metadata,
		UpdatedBy:     actorID,
		UpdatedByName: actorName,
		UpdatedType:   actorType,
	})
	if err != nil {
		return domain.Task{}, mapAppError("update task", err)
	}
	return task, nil
}

// MoveTask moves one task to a target column/position.
func (a *AppServiceAdapter) MoveTask(ctx context.Context, in MoveTaskRequest) (domain.Task, error) {
	if a == nil || a.service == nil {
		return domain.Task{}, fmt.Errorf("app service adapter is not configured: %w", ErrInvalidCaptureStateRequest)
	}
	ctx, _, err := withMutationGuardContext(ctx, in.Actor)
	if err != nil {
		return domain.Task{}, err
	}
	task, err := a.service.MoveTask(ctx, strings.TrimSpace(in.TaskID), strings.TrimSpace(in.ToColumnID), in.Position)
	if err != nil {
		return domain.Task{}, mapAppError("move task", err)
	}
	return task, nil
}

// DeleteTask applies archive/hard delete behavior for one task.
func (a *AppServiceAdapter) DeleteTask(ctx context.Context, in DeleteTaskRequest) error {
	if a == nil || a.service == nil {
		return fmt.Errorf("app service adapter is not configured: %w", ErrInvalidCaptureStateRequest)
	}
	ctx, _, err := withMutationGuardContext(ctx, in.Actor)
	if err != nil {
		return err
	}
	if err := a.service.DeleteTask(ctx, strings.TrimSpace(in.TaskID), app.DeleteMode(strings.TrimSpace(in.Mode))); err != nil {
		return mapAppError("delete task", err)
	}
	return nil
}

// RestoreTask restores one archived task.
func (a *AppServiceAdapter) RestoreTask(ctx context.Context, in RestoreTaskRequest) (domain.Task, error) {
	if a == nil || a.service == nil {
		return domain.Task{}, fmt.Errorf("app service adapter is not configured: %w", ErrInvalidCaptureStateRequest)
	}
	ctx, _, err := withMutationGuardContext(ctx, in.Actor)
	if err != nil {
		return domain.Task{}, err
	}
	task, err := a.service.RestoreTask(ctx, strings.TrimSpace(in.TaskID))
	if err != nil {
		return domain.Task{}, mapAppError("restore task", err)
	}
	return task, nil
}

// ReparentTask changes the parent relationship for one task.
func (a *AppServiceAdapter) ReparentTask(ctx context.Context, in ReparentTaskRequest) (domain.Task, error) {
	if a == nil || a.service == nil {
		return domain.Task{}, fmt.Errorf("app service adapter is not configured: %w", ErrInvalidCaptureStateRequest)
	}
	ctx, _, err := withMutationGuardContext(ctx, in.Actor)
	if err != nil {
		return domain.Task{}, err
	}
	task, err := a.service.ReparentTask(ctx, strings.TrimSpace(in.TaskID), strings.TrimSpace(in.ParentID))
	if err != nil {
		return domain.Task{}, mapAppError("reparent task", err)
	}
	return task, nil
}

// ListChildTasks lists children for one parent task.
func (a *AppServiceAdapter) ListChildTasks(ctx context.Context, projectID, parentID string, includeArchived bool) ([]domain.Task, error) {
	if a == nil || a.service == nil {
		return nil, fmt.Errorf("app service adapter is not configured: %w", ErrInvalidCaptureStateRequest)
	}
	tasks, err := a.service.ListChildTasks(ctx, strings.TrimSpace(projectID), strings.TrimSpace(parentID), includeArchived)
	if err != nil {
		return nil, mapAppError("list child tasks", err)
	}
	return tasks, nil
}

// SearchTasks runs a scoped or cross-project search query.
func (a *AppServiceAdapter) SearchTasks(ctx context.Context, in SearchTasksRequest) ([]SearchTaskMatch, error) {
	if a == nil || a.service == nil {
		return nil, fmt.Errorf("app service adapter is not configured: %w", ErrInvalidCaptureStateRequest)
	}
	matches, err := a.service.SearchTaskMatches(ctx, app.SearchTasksFilter{
		ProjectID:       strings.TrimSpace(in.ProjectID),
		Query:           strings.TrimSpace(in.Query),
		CrossProject:    in.CrossProject,
		IncludeArchived: in.IncludeArchived,
		States:          append([]string(nil), in.States...),
		Levels:          append([]string(nil), in.Levels...),
		Kinds:           append([]string(nil), in.Kinds...),
		LabelsAny:       append([]string(nil), in.LabelsAny...),
		LabelsAll:       append([]string(nil), in.LabelsAll...),
		Mode:            app.SearchMode(strings.TrimSpace(in.Mode)),
		Sort:            app.SearchSort(strings.TrimSpace(in.Sort)),
		Limit:           in.Limit,
		Offset:          in.Offset,
	})
	if err != nil {
		return nil, mapAppError("search task matches", err)
	}
	out := make([]SearchTaskMatch, 0, len(matches))
	for _, match := range matches {
		out = append(out, SearchTaskMatch{
			Project: match.Project,
			Task:    match.Task,
			StateID: match.StateID,
		})
	}
	return out, nil
}

// ListProjectChangeEvents returns recent change events for one project.
func (a *AppServiceAdapter) ListProjectChangeEvents(ctx context.Context, projectID string, limit int) ([]domain.ChangeEvent, error) {
	if a == nil || a.service == nil {
		return nil, fmt.Errorf("app service adapter is not configured: %w", ErrInvalidCaptureStateRequest)
	}
	events, err := a.service.ListProjectChangeEvents(ctx, strings.TrimSpace(projectID), limit)
	if err != nil {
		return nil, mapAppError("list project change events", err)
	}
	return events, nil
}

// GetProjectDependencyRollup returns dependency counts for one project.
func (a *AppServiceAdapter) GetProjectDependencyRollup(ctx context.Context, projectID string) (domain.DependencyRollup, error) {
	if a == nil || a.service == nil {
		return domain.DependencyRollup{}, fmt.Errorf("app service adapter is not configured: %w", ErrInvalidCaptureStateRequest)
	}
	rollup, err := a.service.GetProjectDependencyRollup(ctx, strings.TrimSpace(projectID))
	if err != nil {
		return domain.DependencyRollup{}, mapAppError("get project dependency rollup", err)
	}
	return rollup, nil
}

// ListKindDefinitions lists kind catalog entries.
func (a *AppServiceAdapter) ListKindDefinitions(ctx context.Context, includeArchived bool) ([]domain.KindDefinition, error) {
	if a == nil || a.service == nil {
		return nil, fmt.Errorf("app service adapter is not configured: %w", ErrInvalidCaptureStateRequest)
	}
	kinds, err := a.service.ListKindDefinitions(ctx, includeArchived)
	if err != nil {
		return nil, mapAppError("list kind definitions", err)
	}
	return kinds, nil
}

// UpsertKindDefinition creates or updates one kind catalog entry.
func (a *AppServiceAdapter) UpsertKindDefinition(ctx context.Context, in UpsertKindDefinitionRequest) (domain.KindDefinition, error) {
	if a == nil || a.service == nil {
		return domain.KindDefinition{}, fmt.Errorf("app service adapter is not configured: %w", ErrInvalidCaptureStateRequest)
	}
	kind, err := a.service.UpsertKindDefinition(ctx, app.CreateKindDefinitionInput{
		ID:                  domain.KindID(strings.TrimSpace(in.ID)),
		DisplayName:         strings.TrimSpace(in.DisplayName),
		DescriptionMarkdown: strings.TrimSpace(in.DescriptionMarkdown),
		AppliesTo:           toKindAppliesToList(in.AppliesTo),
		AllowedParentScopes: toKindAppliesToList(in.AllowedParentScopes),
		PayloadSchemaJSON:   strings.TrimSpace(in.PayloadSchemaJSON),
		Template:            in.Template,
	})
	if err != nil {
		return domain.KindDefinition{}, mapAppError("upsert kind definition", err)
	}
	return kind, nil
}

// SetProjectAllowedKinds updates a project's kind allowlist.
func (a *AppServiceAdapter) SetProjectAllowedKinds(ctx context.Context, in SetProjectAllowedKindsRequest) error {
	if a == nil || a.service == nil {
		return fmt.Errorf("app service adapter is not configured: %w", ErrInvalidCaptureStateRequest)
	}
	if err := a.service.SetProjectAllowedKinds(ctx, app.SetProjectAllowedKindsInput{
		ProjectID: strings.TrimSpace(in.ProjectID),
		KindIDs:   toKindIDList(in.KindIDs),
	}); err != nil {
		return mapAppError("set project allowed kinds", err)
	}
	return nil
}

// parseOptionalDurationString parses one optional Go duration string used by transport auth request inputs.
func parseOptionalDurationString(raw, field string) (time.Duration, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, nil
	}
	value, err := time.ParseDuration(raw)
	if err != nil {
		return 0, fmt.Errorf("%s %q is invalid: %w", field, raw, ErrInvalidCaptureStateRequest)
	}
	if value < 0 {
		return 0, fmt.Errorf("%s %q is invalid: %w", field, raw, ErrInvalidCaptureStateRequest)
	}
	return value, nil
}

// parseContinuationJSON decodes one optional continuation metadata object encoded as JSON.
func parseContinuationJSON(raw string) (map[string]any, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	var continuation map[string]any
	if err := json.Unmarshal([]byte(raw), &continuation); err != nil {
		return nil, fmt.Errorf("continuation_json is invalid: %w", ErrInvalidCaptureStateRequest)
	}
	return cloneJSONObject(continuation), nil
}

// requestedActorType resolves explicit requester attribution and falls back to requested principal type.
func requestedActorType(requestedByType, principalType string) domain.ActorType {
	switch strings.TrimSpace(strings.ToLower(requestedByType)) {
	case string(domain.ActorTypeAgent):
		return domain.ActorTypeAgent
	case string(domain.ActorTypeSystem):
		return domain.ActorTypeSystem
	case string(domain.ActorTypeUser):
		return domain.ActorTypeUser
	}
	switch strings.TrimSpace(strings.ToLower(principalType)) {
	case "agent", "service", "system":
		return domain.ActorTypeAgent
	default:
		return domain.ActorTypeUser
	}
}

// firstNonEmptyRequestedBy returns the first non-empty trimmed requester identifier.
func firstNonEmptyRequestedBy(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

// requesterClientID resolves one requester-bound claim client identifier with fallback to the requested client.
func requesterClientID(in CreateAuthRequestRequest) string {
	if trimmed := strings.TrimSpace(in.RequesterClientID); trimmed != "" {
		return trimmed
	}
	return strings.TrimSpace(in.ClientID)
}

// mapAuthRequestRecord converts one domain auth request into one transport-facing record.
func mapAuthRequestRecord(request domain.AuthRequest) AuthRequestRecord {
	approvedSessionTTL := ""
	if request.ApprovedSessionTTL > 0 {
		approvedSessionTTL = request.ApprovedSessionTTL.String()
	}
	return AuthRequestRecord{
		ID:                     request.ID,
		State:                  string(request.State),
		Path:                   request.Path,
		ApprovedPath:           request.ApprovedPath,
		ProjectID:              request.ProjectID,
		BranchID:               request.BranchID,
		PhaseIDs:               append([]string(nil), request.PhaseIDs...),
		ScopeType:              string(request.ScopeType),
		ScopeID:                request.ScopeID,
		PrincipalID:            request.PrincipalID,
		PrincipalType:          request.PrincipalType,
		PrincipalRole:          request.PrincipalRole,
		PrincipalName:          request.PrincipalName,
		ClientID:               request.ClientID,
		ClientType:             request.ClientType,
		ClientName:             request.ClientName,
		RequestedSessionTTL:    request.RequestedSessionTTL.String(),
		ApprovedSessionTTL:     approvedSessionTTL,
		Reason:                 request.Reason,
		HasContinuation:        len(request.Continuation) > 0,
		RequestedByActor:       request.RequestedByActor,
		RequestedByType:        string(request.RequestedByType),
		CreatedAt:              request.CreatedAt.UTC(),
		ExpiresAt:              request.ExpiresAt.UTC(),
		ResolvedByActor:        request.ResolvedByActor,
		ResolvedByType:         string(request.ResolvedByType),
		ResolvedAt:             request.ResolvedAt,
		ResolutionNote:         request.ResolutionNote,
		IssuedSessionID:        request.IssuedSessionID,
		IssuedSessionExpiresAt: request.IssuedSessionExpiresAt,
	}
}

// ListProjectAllowedKinds lists canonical kind ids in one project's allowlist.
func (a *AppServiceAdapter) ListProjectAllowedKinds(ctx context.Context, projectID string) ([]string, error) {
	if a == nil || a.service == nil {
		return nil, fmt.Errorf("app service adapter is not configured: %w", ErrInvalidCaptureStateRequest)
	}
	kindIDs, err := a.service.ListProjectAllowedKinds(ctx, strings.TrimSpace(projectID))
	if err != nil {
		return nil, mapAppError("list project allowed kinds", err)
	}
	out := make([]string, 0, len(kindIDs))
	for _, kindID := range kindIDs {
		out = append(out, string(kindID))
	}
	return out, nil
}

// IssueCapabilityLease issues one scope-bound capability lease.
func (a *AppServiceAdapter) IssueCapabilityLease(ctx context.Context, in IssueCapabilityLeaseRequest) (domain.CapabilityLease, error) {
	if a == nil || a.service == nil {
		return domain.CapabilityLease{}, fmt.Errorf("app service adapter is not configured: %w", ErrInvalidCaptureStateRequest)
	}
	lease, err := a.service.IssueCapabilityLease(ctx, app.IssueCapabilityLeaseInput{
		ProjectID:                 strings.TrimSpace(in.ProjectID),
		ScopeType:                 domain.CapabilityScopeType(strings.TrimSpace(in.ScopeType)),
		ScopeID:                   strings.TrimSpace(in.ScopeID),
		Role:                      domain.CapabilityRole(strings.TrimSpace(in.Role)),
		AgentName:                 strings.TrimSpace(in.AgentName),
		AgentInstanceID:           strings.TrimSpace(in.AgentInstanceID),
		ParentInstanceID:          strings.TrimSpace(in.ParentInstanceID),
		AllowEqualScopeDelegation: in.AllowEqualScopeDelegation,
		RequestedTTL:              durationFromSeconds(in.RequestedTTLSeconds),
		OverrideToken:             strings.TrimSpace(in.OverrideToken),
	})
	if err != nil {
		return domain.CapabilityLease{}, mapAppError("issue capability lease", err)
	}
	return lease, nil
}

// HeartbeatCapabilityLease records one heartbeat against an active lease.
func (a *AppServiceAdapter) HeartbeatCapabilityLease(ctx context.Context, in HeartbeatCapabilityLeaseRequest) (domain.CapabilityLease, error) {
	if a == nil || a.service == nil {
		return domain.CapabilityLease{}, fmt.Errorf("app service adapter is not configured: %w", ErrInvalidCaptureStateRequest)
	}
	lease, err := a.service.HeartbeatCapabilityLease(ctx, app.HeartbeatCapabilityLeaseInput{
		AgentInstanceID: strings.TrimSpace(in.AgentInstanceID),
		LeaseToken:      strings.TrimSpace(in.LeaseToken),
	})
	if err != nil {
		return domain.CapabilityLease{}, mapAppError("heartbeat capability lease", err)
	}
	return lease, nil
}

// RenewCapabilityLease extends one lease expiry.
func (a *AppServiceAdapter) RenewCapabilityLease(ctx context.Context, in RenewCapabilityLeaseRequest) (domain.CapabilityLease, error) {
	if a == nil || a.service == nil {
		return domain.CapabilityLease{}, fmt.Errorf("app service adapter is not configured: %w", ErrInvalidCaptureStateRequest)
	}
	lease, err := a.service.RenewCapabilityLease(ctx, app.RenewCapabilityLeaseInput{
		AgentInstanceID: strings.TrimSpace(in.AgentInstanceID),
		LeaseToken:      strings.TrimSpace(in.LeaseToken),
		TTL:             durationFromSeconds(in.TTLSeconds),
	})
	if err != nil {
		return domain.CapabilityLease{}, mapAppError("renew capability lease", err)
	}
	return lease, nil
}

// RevokeCapabilityLease revokes one lease by instance id.
func (a *AppServiceAdapter) RevokeCapabilityLease(ctx context.Context, in RevokeCapabilityLeaseRequest) (domain.CapabilityLease, error) {
	if a == nil || a.service == nil {
		return domain.CapabilityLease{}, fmt.Errorf("app service adapter is not configured: %w", ErrInvalidCaptureStateRequest)
	}
	lease, err := a.service.RevokeCapabilityLease(ctx, app.RevokeCapabilityLeaseInput{
		AgentInstanceID: strings.TrimSpace(in.AgentInstanceID),
		Reason:          strings.TrimSpace(in.Reason),
	})
	if err != nil {
		return domain.CapabilityLease{}, mapAppError("revoke capability lease", err)
	}
	return lease, nil
}

// RevokeAllCapabilityLeases revokes all matching leases for one scope tuple.
func (a *AppServiceAdapter) RevokeAllCapabilityLeases(ctx context.Context, in RevokeAllCapabilityLeasesRequest) error {
	if a == nil || a.service == nil {
		return fmt.Errorf("app service adapter is not configured: %w", ErrInvalidCaptureStateRequest)
	}
	if err := a.service.RevokeAllCapabilityLeases(ctx, app.RevokeAllCapabilityLeasesInput{
		ProjectID: strings.TrimSpace(in.ProjectID),
		ScopeType: domain.CapabilityScopeType(strings.TrimSpace(in.ScopeType)),
		ScopeID:   strings.TrimSpace(in.ScopeID),
		Reason:    strings.TrimSpace(in.Reason),
	}); err != nil {
		return mapAppError("revoke all capability leases", err)
	}
	return nil
}

// CreateComment creates one markdown-rich comment for a concrete target.
func (a *AppServiceAdapter) CreateComment(ctx context.Context, in CreateCommentRequest) (CommentRecord, error) {
	if a == nil || a.service == nil {
		return CommentRecord{}, fmt.Errorf("app service adapter is not configured: %w", ErrInvalidCaptureStateRequest)
	}
	summary := strings.TrimSpace(in.Summary)
	if summary == "" {
		return CommentRecord{}, fmt.Errorf("summary is required: %w", ErrInvalidCaptureStateRequest)
	}
	ctx, actorType, err := withMutationGuardContext(ctx, in.Actor)
	if err != nil {
		return CommentRecord{}, err
	}
	actorID, actorName := deriveMutationActorIdentity(in.Actor)
	comment, err := a.service.CreateComment(ctx, app.CreateCommentInput{
		ProjectID:    strings.TrimSpace(in.ProjectID),
		TargetType:   domain.CommentTargetType(strings.TrimSpace(in.TargetType)),
		TargetID:     strings.TrimSpace(in.TargetID),
		BodyMarkdown: buildCommentBodyMarkdown(summary, in.BodyMarkdown),
		ActorID:      actorID,
		ActorName:    actorName,
		ActorType:    actorType,
	})
	if err != nil {
		return CommentRecord{}, mapAppError("create comment", err)
	}
	record := mapDomainCommentRecord(comment)
	record.Summary = summary
	return record, nil
}

// ListCommentsByTarget lists comments for one concrete target.
func (a *AppServiceAdapter) ListCommentsByTarget(ctx context.Context, in ListCommentsByTargetRequest) ([]CommentRecord, error) {
	if a == nil || a.service == nil {
		return nil, fmt.Errorf("app service adapter is not configured: %w", ErrInvalidCaptureStateRequest)
	}
	comments, err := a.service.ListCommentsByTarget(ctx, app.ListCommentsByTargetInput{
		ProjectID:  strings.TrimSpace(in.ProjectID),
		TargetType: domain.CommentTargetType(strings.TrimSpace(in.TargetType)),
		TargetID:   strings.TrimSpace(in.TargetID),
	})
	if err != nil {
		return nil, mapAppError("list comments by target", err)
	}
	out := make([]CommentRecord, 0, len(comments))
	for _, comment := range comments {
		out = append(out, mapDomainCommentRecord(comment))
	}
	return out, nil
}

// mapDomainCommentRecord maps one domain comment into the transport comment contract.
func mapDomainCommentRecord(comment domain.Comment) CommentRecord {
	return CommentRecord{
		ID:           comment.ID,
		ProjectID:    comment.ProjectID,
		TargetType:   string(comment.TargetType),
		TargetID:     comment.TargetID,
		Summary:      commentSummaryFromMarkdown(comment.BodyMarkdown),
		BodyMarkdown: comment.BodyMarkdown,
		ActorID:      comment.ActorID,
		ActorName:    comment.ActorName,
		ActorType:    string(comment.ActorType),
		CreatedAt:    comment.CreatedAt.UTC(),
		UpdatedAt:    comment.UpdatedAt.UTC(),
	}
}

// commentSummaryFromMarkdown extracts one deterministic summary line from markdown text.
func commentSummaryFromMarkdown(markdown string) string {
	lines := strings.Split(strings.TrimSpace(markdown), "\n")
	for _, line := range lines {
		candidate := strings.TrimSpace(line)
		candidate = strings.TrimLeft(candidate, "#>*-` ")
		candidate = strings.TrimSpace(candidate)
		if candidate != "" {
			return candidate
		}
	}
	return ""
}

// buildCommentBodyMarkdown combines summary and optional markdown details into one comment body.
func buildCommentBodyMarkdown(summary, bodyMarkdown string) string {
	summary = strings.TrimSpace(summary)
	bodyMarkdown = strings.TrimSpace(bodyMarkdown)
	switch {
	case summary == "":
		return bodyMarkdown
	case bodyMarkdown == "":
		return summary
	case bodyMarkdown == summary:
		return summary
	default:
		return summary + "\n\n" + bodyMarkdown
	}
}

// parseOptionalRFC3339 parses one optional RFC3339 timestamp string.
func parseOptionalRFC3339(raw string) (*time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	ts, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return nil, fmt.Errorf("due_at must be RFC3339: %w", ErrInvalidCaptureStateRequest)
	}
	utc := ts.UTC()
	return &utc, nil
}

// withMutationGuardContext validates actor tuple semantics and optionally attaches lease guard context.
func withMutationGuardContext(ctx context.Context, actor ActorLeaseTuple) (context.Context, domain.ActorType, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	actorType := normalizeActorType(actor.ActorType)
	if !isValidActorType(actorType) {
		return nil, "", fmt.Errorf("actor_type %q is unsupported: %w", actor.ActorType, ErrInvalidCaptureStateRequest)
	}

	agentName := strings.TrimSpace(actor.AgentName)
	agentInstanceID := strings.TrimSpace(actor.AgentInstanceID)
	leaseToken := strings.TrimSpace(actor.LeaseToken)
	overrideToken := strings.TrimSpace(actor.OverrideToken)
	hasGuardTuple := agentInstanceID != "" || leaseToken != "" || overrideToken != ""
	if hasGuardTuple && actorType == domain.ActorTypeUser {
		return nil, "", fmt.Errorf("actor_type=user cannot be used with guarded mutation tuple: %w", ErrInvalidCaptureStateRequest)
	}
	if actorType != domain.ActorTypeUser || hasGuardTuple {
		if agentName == "" || agentInstanceID == "" || leaseToken == "" {
			return nil, "", fmt.Errorf("agent_name, agent_instance_id, and lease_token are required for non-user or guarded mutations: %w", ErrInvalidCaptureStateRequest)
		}
		ctx = app.WithMutationGuard(ctx, app.MutationGuard{
			AgentName:       agentName,
			AgentInstanceID: agentInstanceID,
			LeaseToken:      leaseToken,
			OverrideToken:   overrideToken,
		})
	}
	hasIdentityInput := strings.TrimSpace(actor.ActorID) != "" ||
		strings.TrimSpace(actor.ActorName) != "" ||
		agentName != "" ||
		agentInstanceID != ""
	if hasIdentityInput {
		actorID, actorName := deriveMutationActorIdentity(actor)
		ctx = app.WithAuthenticatedCaller(ctx, domain.AuthenticatedCaller{
			PrincipalID:   actorID,
			PrincipalName: actorName,
			PrincipalType: actorType,
		})
	}
	return ctx, actorType, nil
}

// deriveMutationActorIdentity resolves deterministic actor tuple values for mutating requests.
// Transport adapters should populate this from authenticated session identity whenever available.
func deriveMutationActorIdentity(actor ActorLeaseTuple) (string, string) {
	actorID := strings.TrimSpace(actor.ActorID)
	if actorID == "" {
		actorID = strings.TrimSpace(actor.AgentInstanceID)
	}
	if actorID == "" {
		actorID = strings.TrimSpace(actor.AgentName)
	}
	if actorID == "" {
		actorID = "tillsyn-user"
	}
	actorName := strings.TrimSpace(actor.ActorName)
	if actorName == "" {
		actorName = strings.TrimSpace(actor.AgentName)
	}
	if actorName == "" {
		actorName = actorID
	}
	return actorID, actorName
}

// normalizeActorType canonicalizes actor type values and defaults to user.
func normalizeActorType(actorType string) domain.ActorType {
	normalized := domain.ActorType(strings.TrimSpace(strings.ToLower(actorType)))
	if normalized == "" {
		return domain.ActorTypeUser
	}
	return normalized
}

// isValidActorType reports whether actor type values are supported by app/domain rules.
func isValidActorType(actorType domain.ActorType) bool {
	switch actorType {
	case domain.ActorTypeUser, domain.ActorTypeAgent:
		return true
	default:
		return false
	}
}

// toKindAppliesToList maps string scope values into domain kind applies_to values.
func toKindAppliesToList(scopes []string) []domain.KindAppliesTo {
	out := make([]domain.KindAppliesTo, 0, len(scopes))
	for _, scope := range scopes {
		out = append(out, domain.KindAppliesTo(strings.TrimSpace(scope)))
	}
	return out
}

// toKindIDList maps string kind ids into domain kind ids.
func toKindIDList(kindIDs []string) []domain.KindID {
	out := make([]domain.KindID, 0, len(kindIDs))
	for _, kindID := range kindIDs {
		out = append(out, domain.KindID(strings.TrimSpace(kindID)))
	}
	return out
}

// cloneJSONObject deep-copies one JSON-compatible object map.
func cloneJSONObject(in map[string]any) map[string]any {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]any, len(in))
	for key, value := range in {
		out[key] = cloneJSONValue(value)
	}
	return out
}

// cloneJSONValue deep-copies one JSON-compatible nested value.
func cloneJSONValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		return cloneJSONObject(typed)
	case []any:
		out := make([]any, 0, len(typed))
		for _, item := range typed {
			out = append(out, cloneJSONValue(item))
		}
		return out
	default:
		return typed
	}
}
