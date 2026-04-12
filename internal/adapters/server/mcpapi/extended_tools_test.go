package mcpapi

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/hylla/tillsyn/internal/adapters/server/common"
	"github.com/hylla/tillsyn/internal/domain"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

// stubExpandedService provides deterministic responses for expanded MCP tool coverage tests.
type stubExpandedService struct {
	stubCaptureStateReader
	stubMutationAuthorizer
	lastCreateProjectReq             common.CreateProjectRequest
	lastListProjectsArchived         bool
	lastGetTaskID                    string
	lastListTasksProjectID           string
	lastListTasksArchived            bool
	lastListChildProjectID           string
	lastListChildParentID            string
	lastListChildArchived            bool
	lastCreateTaskReq                common.CreateTaskRequest
	lastUpdateTaskReq                common.UpdateTaskRequest
	lastMoveTaskStateReq             common.MoveTaskStateRequest
	lastRestoreTaskReq               common.RestoreTaskRequest
	lastIssueLeaseReq                common.IssueCapabilityLeaseRequest
	lastListLeaseReq                 common.ListCapabilityLeasesRequest
	lastCreateCommentReq             common.CreateCommentRequest
	lastListCommentReq               common.ListCommentsByTargetRequest
	lastCreateHandoffReq             common.CreateHandoffRequest
	lastUpdateHandoffReq             common.UpdateHandoffRequest
	lastListHandoffsReq              common.ListHandoffsRequest
	lastListTemplateReq              common.ListTemplateLibrariesRequest
	lastListKindsArchived            bool
	lastUpsertKindReq                common.UpsertKindDefinitionRequest
	lastSetAllowedKindsReq           common.SetProjectAllowedKindsRequest
	lastEnsureBuiltinReq             common.EnsureBuiltinTemplateLibraryRequest
	lastUpsertTemplateReq            common.UpsertTemplateLibraryRequest
	lastBindTemplateReq              common.BindProjectTemplateLibraryRequest
	lastApproveTemplateMigrationsReq common.ApproveProjectTemplateMigrationsRequest
	lastGetTemplateID                string
	lastGetBuiltinTemplateID         string
	lastGetTemplateBindingID         string
	lastGetTemplatePreviewID         string
	lastGetNodeContractID            string
	lastSearchTasksReq               common.SearchTasksRequest
	lastEmbeddingsStatusReq          common.EmbeddingsStatusRequest
	lastEmbeddingsReindexReq         common.ReindexEmbeddingsRequest
	lastCreateAuthRequestReq         common.CreateAuthRequestRequest
	lastListAuthRequestsReq          common.ListAuthRequestsRequest
	lastGetAuthRequestID             string
	lastGetHandoffID                 string
	lastClaimAuthRequestReq          common.ClaimAuthRequestRequest
	lastCancelAuthRequestReq         common.CancelAuthRequestRequest
	lastListAuthSessionsReq          common.ListAuthSessionsRequest
	lastValidateAuthSessionReq       common.ValidateAuthSessionRequest
	lastCheckAuthSessionReq          common.CheckAuthSessionGovernanceRequest
	lastRevokeAuthSessionReq         common.RevokeAuthSessionRequest
}

// GetBootstrapGuide returns one deterministic bootstrap payload.
func (s *stubExpandedService) GetBootstrapGuide(_ context.Context) (common.BootstrapGuide, error) {
	return common.BootstrapGuide{
		Mode:          "bootstrap_required",
		Summary:       "No project context exists yet. Create an auth request, wait for approval, claim it, then create the project.",
		WhatTillsynIs: "Tillsyn is a scoped planner with comments, handoffs, auth requests, capture-state recovery, and template workflow contracts.",
		NextSteps: []string{
			"If it is not approved yet, create an auth request with till.auth_request(operation=create).",
			"After approval, claim the request with till.auth_request(operation=claim), then create the project with till.project(operation=create).",
			"Use till.capture_state after restart instead of rerunning bootstrap on an existing instance.",
		},
		Recommended: []string{
			"till.get_instructions",
			"till.auth_request",
			"till.project",
			"till.capture_state",
		},
	}, nil
}

// ListProjects returns one deterministic project row.
func (s *stubExpandedService) ListProjects(_ context.Context, includeArchived bool) ([]domain.Project, error) {
	s.lastListProjectsArchived = includeArchived
	now := time.Date(2026, 2, 24, 12, 0, 0, 0, time.UTC)
	return []domain.Project{
		{
			ID:        "p1",
			Slug:      "proj-1",
			Name:      "Project One",
			Kind:      domain.KindID("go-project"),
			Metadata:  domain.ProjectMetadata{StandardsMarkdown: "Use MCP tools first.\nRun TDD-style changes and finish with mage ci."},
			CreatedAt: now,
			UpdatedAt: now,
		},
	}, nil
}

// CreateProject returns one deterministic project row.
func (s *stubExpandedService) CreateProject(_ context.Context, in common.CreateProjectRequest) (domain.Project, error) {
	s.lastCreateProjectReq = in
	now := time.Date(2026, 2, 24, 12, 0, 0, 0, time.UTC)
	return domain.Project{ID: "p1", Slug: "proj-1", Name: "Project One", CreatedAt: now, UpdatedAt: now}, nil
}

// UpdateProject returns one deterministic updated project row.
func (s *stubExpandedService) UpdateProject(_ context.Context, _ common.UpdateProjectRequest) (domain.Project, error) {
	now := time.Date(2026, 2, 24, 12, 0, 0, 0, time.UTC)
	return domain.Project{ID: "p1", Slug: "proj-1", Name: "Project One Updated", CreatedAt: now, UpdatedAt: now}, nil
}

// CreateAuthRequest returns one deterministic auth-request row.
func (s *stubExpandedService) CreateAuthRequest(_ context.Context, in common.CreateAuthRequestRequest) (common.AuthRequestRecord, error) {
	s.lastCreateAuthRequestReq = in
	now := time.Date(2026, 3, 20, 12, 0, 0, 0, time.UTC)
	return common.AuthRequestRecord{
		ID:                  "req-1",
		State:               "pending",
		Path:                strings.TrimSpace(in.Path),
		PrincipalRole:       strings.TrimSpace(in.PrincipalRole),
		ProjectID:           "p1",
		ScopeType:           common.ScopeTypeProject,
		ScopeID:             "p1",
		PrincipalID:         strings.TrimSpace(in.PrincipalID),
		PrincipalType:       strings.TrimSpace(in.PrincipalType),
		ClientID:            strings.TrimSpace(in.ClientID),
		ClientType:          strings.TrimSpace(in.ClientType),
		RequestedSessionTTL: "2h0m0s",
		Reason:              strings.TrimSpace(in.Reason),
		CreatedAt:           now,
		ExpiresAt:           now.Add(30 * time.Minute),
	}, nil
}

// ListAuthRequests returns one deterministic auth-request inventory row.
func (s *stubExpandedService) ListAuthRequests(_ context.Context, in common.ListAuthRequestsRequest) ([]common.AuthRequestRecord, error) {
	s.lastListAuthRequestsReq = in
	now := time.Date(2026, 3, 20, 12, 0, 0, 0, time.UTC)
	return []common.AuthRequestRecord{{
		ID:                  "req-1",
		State:               "pending",
		Path:                "project/p1",
		ProjectID:           "p1",
		ScopeType:           common.ScopeTypeProject,
		ScopeID:             "p1",
		PrincipalID:         "review-agent",
		PrincipalType:       "agent",
		ClientID:            "till-mcp-stdio",
		ClientType:          "mcp-stdio",
		RequestedSessionTTL: "2h0m0s",
		Reason:              "manual MCP review",
		CreatedAt:           now,
		ExpiresAt:           now.Add(30 * time.Minute),
	}}, nil
}

// GetAuthRequest returns one deterministic auth-request record.
func (s *stubExpandedService) GetAuthRequest(_ context.Context, requestID string) (common.AuthRequestRecord, error) {
	s.lastGetAuthRequestID = requestID
	now := time.Date(2026, 3, 20, 12, 0, 0, 0, time.UTC)
	return common.AuthRequestRecord{
		ID:                  strings.TrimSpace(requestID),
		State:               "pending",
		Path:                "project/p1",
		ProjectID:           "p1",
		ScopeType:           common.ScopeTypeProject,
		ScopeID:             "p1",
		PrincipalID:         "review-agent",
		PrincipalType:       "agent",
		ClientID:            "till-mcp-stdio",
		ClientType:          "mcp-stdio",
		RequestedSessionTTL: "2h0m0s",
		Reason:              "manual MCP review",
		CreatedAt:           now,
		ExpiresAt:           now.Add(30 * time.Minute),
	}, nil
}

// ClaimAuthRequest returns one deterministic approved continuation result.
func (s *stubExpandedService) ClaimAuthRequest(_ context.Context, in common.ClaimAuthRequestRequest) (common.AuthRequestClaimResult, error) {
	s.lastClaimAuthRequestReq = in
	expiresAt := time.Date(2026, 3, 20, 14, 0, 0, 0, time.UTC)
	now := time.Date(2026, 3, 20, 12, 0, 0, 0, time.UTC)
	return common.AuthRequestClaimResult{
		Request: common.AuthRequestRecord{
			ID:                     strings.TrimSpace(in.RequestID),
			State:                  "approved",
			Path:                   "project/p1",
			ApprovedPath:           "project/p1/branch/review",
			ProjectID:              "p1",
			ScopeType:              common.ScopeTypeProject,
			ScopeID:                "p1",
			PrincipalID:            "review-agent",
			PrincipalType:          "agent",
			PrincipalRole:          "builder",
			ClientID:               "till-mcp-stdio",
			ClientType:             "mcp-stdio",
			RequestedSessionTTL:    "2h0m0s",
			ApprovedSessionTTL:     "2h0m0s",
			Reason:                 "manual MCP review",
			CreatedAt:              now,
			ExpiresAt:              now.Add(30 * time.Minute),
			IssuedSessionID:        "sess-1",
			IssuedSessionExpiresAt: &expiresAt,
		},
		SessionSecret: "secret-1",
	}, nil
}

// CancelAuthRequest returns one deterministic canceled auth-request result.
func (s *stubExpandedService) CancelAuthRequest(_ context.Context, in common.CancelAuthRequestRequest) (common.AuthRequestRecord, error) {
	s.lastCancelAuthRequestReq = in
	now := time.Date(2026, 3, 20, 12, 0, 0, 0, time.UTC)
	return common.AuthRequestRecord{
		ID:               strings.TrimSpace(in.RequestID),
		State:            "canceled",
		Path:             "project/p1",
		ProjectID:        "p1",
		ScopeType:        common.ScopeTypeProject,
		ScopeID:          "p1",
		PrincipalID:      "review-agent",
		PrincipalType:    "agent",
		ClientID:         "till-mcp-stdio",
		ClientType:       "mcp-stdio",
		RequestedByActor: strings.TrimSpace(in.PrincipalID),
		RequestedByType:  "agent",
		ResolutionNote:   strings.TrimSpace(in.ResolutionNote),
		CreatedAt:        now,
		ExpiresAt:        now.Add(30 * time.Minute),
	}, nil
}

// ListAuthSessions returns one deterministic auth-session inventory row.
func (s *stubExpandedService) ListAuthSessions(_ context.Context, in common.ListAuthSessionsRequest) ([]common.AuthSessionRecord, error) {
	s.lastListAuthSessionsReq = in
	now := time.Date(2026, 3, 20, 12, 0, 0, 0, time.UTC)
	return []common.AuthSessionRecord{{
		SessionID:     "sess-1",
		State:         "active",
		ProjectID:     "p1",
		AuthRequestID: "req-1",
		ApprovedPath:  "project/p1",
		PrincipalID:   "review-agent",
		PrincipalType: "agent",
		PrincipalRole: "builder",
		PrincipalName: "Review Agent",
		ClientID:      "till-mcp-stdio",
		ClientType:    "mcp-stdio",
		ClientName:    "Till MCP STDIO",
		IssuedAt:      now,
		ExpiresAt:     now.Add(2 * time.Hour),
	}}, nil
}

// ValidateAuthSession returns one deterministic auth-session row.
func (s *stubExpandedService) ValidateAuthSession(_ context.Context, in common.ValidateAuthSessionRequest) (common.AuthSessionRecord, error) {
	s.lastValidateAuthSessionReq = in
	now := time.Date(2026, 3, 20, 12, 0, 0, 0, time.UTC)
	lastValidatedAt := now.Add(5 * time.Minute)
	return common.AuthSessionRecord{
		SessionID:       strings.TrimSpace(in.SessionID),
		State:           "active",
		ProjectID:       "p1",
		AuthRequestID:   "req-1",
		ApprovedPath:    "project/p1",
		PrincipalID:     "review-agent",
		PrincipalType:   "agent",
		PrincipalRole:   "builder",
		PrincipalName:   "Review Agent",
		ClientID:        "till-mcp-stdio",
		ClientType:      "mcp-stdio",
		ClientName:      "Till MCP STDIO",
		IssuedAt:        now,
		ExpiresAt:       now.Add(2 * time.Hour),
		LastValidatedAt: &lastValidatedAt,
	}, nil
}

// CheckAuthSessionGovernance returns one deterministic governance-check result.
func (s *stubExpandedService) CheckAuthSessionGovernance(_ context.Context, in common.CheckAuthSessionGovernanceRequest) (common.AuthSessionGovernanceCheckResult, error) {
	s.lastCheckAuthSessionReq = in
	now := time.Date(2026, 3, 20, 12, 0, 0, 0, time.UTC)
	return common.AuthSessionGovernanceCheckResult{
		Authorized:          false,
		DecisionReason:      "out_of_scope",
		ActingSessionID:     strings.TrimSpace(in.ActingSessionID),
		ActingPrincipalID:   "review-agent",
		ActingPrincipalRole: "research",
		ActingApprovedPath:  "project/p1",
		TargetSession: common.AuthSessionRecord{
			SessionID:     strings.TrimSpace(in.SessionID),
			State:         "active",
			ApprovedPath:  "global",
			PrincipalID:   "global-agent",
			PrincipalType: "agent",
			PrincipalRole: "orchestrator",
			ClientID:      "global-client",
			ClientType:    "mcp-stdio",
			IssuedAt:      now,
			ExpiresAt:     now.Add(2 * time.Hour),
		},
	}, nil
}

// RevokeAuthSession returns one deterministic revoked auth-session row.
func (s *stubExpandedService) RevokeAuthSession(_ context.Context, in common.RevokeAuthSessionRequest) (common.AuthSessionRecord, error) {
	s.lastRevokeAuthSessionReq = in
	now := time.Date(2026, 3, 20, 12, 0, 0, 0, time.UTC)
	revokedAt := now.Add(10 * time.Minute)
	return common.AuthSessionRecord{
		SessionID:        strings.TrimSpace(in.SessionID),
		State:            "revoked",
		ProjectID:        "p1",
		AuthRequestID:    "req-1",
		ApprovedPath:     "project/p1",
		PrincipalID:      "review-agent",
		PrincipalType:    "agent",
		PrincipalRole:    "builder",
		PrincipalName:    "Review Agent",
		ClientID:         "till-mcp-stdio",
		ClientType:       "mcp-stdio",
		ClientName:       "Till MCP STDIO",
		IssuedAt:         now,
		ExpiresAt:        now.Add(2 * time.Hour),
		RevokedAt:        &revokedAt,
		RevocationReason: strings.TrimSpace(in.Reason),
	}, nil
}

// ListTasks returns one deterministic task row.
func (s *stubExpandedService) ListTasks(_ context.Context, projectID string, includeArchived bool) ([]domain.Task, error) {
	s.lastListTasksProjectID = projectID
	s.lastListTasksArchived = includeArchived
	now := time.Date(2026, 2, 24, 12, 0, 0, 0, time.UTC)
	return []domain.Task{
		{
			ID:             "t1",
			ProjectID:      "p1",
			ColumnID:       "c1",
			Position:       0,
			Title:          "Task One",
			Kind:           domain.WorkKindTask,
			Scope:          domain.KindAppliesToTask,
			LifecycleState: domain.StateTodo,
			Priority:       domain.PriorityMedium,
			CreatedAt:      now,
			UpdatedAt:      now,
		},
	}, nil
}

// GetTask returns one deterministic task row by id.
func (s *stubExpandedService) GetTask(_ context.Context, taskID string) (domain.Task, error) {
	s.lastGetTaskID = taskID
	now := time.Date(2026, 2, 24, 12, 0, 0, 0, time.UTC)
	return domain.Task{
		ID:             strings.TrimSpace(taskID),
		ProjectID:      "p1",
		ColumnID:       "c1",
		Position:       0,
		Title:          "Task One",
		Kind:           domain.WorkKindTask,
		Scope:          domain.KindAppliesToTask,
		LifecycleState: domain.StateTodo,
		Priority:       domain.PriorityMedium,
		Description:    "Implement the scoped instructions explainer.",
		Metadata: domain.TaskMetadata{
			Objective:                "Explain the node's real workflow rules.",
			ImplementationNotesAgent: "Use MCP surfaces and keep Go changes idiomatic.",
			AcceptanceCriteria:       "The explainer shows project rules and node-local rules.",
			DefinitionOfDone:         "Focused tests pass and docs are aligned.",
			ValidationPlan:           "Run mage test-pkg ./internal/adapters/server/mcpapi and mage ci.",
			DependsOn:                []string{"phase-plan"},
			BlockedBy:                []string{"task-design-review"},
			BlockedReason:            "Waiting for the design review task to finish before implementation starts.",
		},
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

// CreateTask returns one deterministic created task row.
func (s *stubExpandedService) CreateTask(_ context.Context, in common.CreateTaskRequest) (domain.Task, error) {
	s.lastCreateTaskReq = in
	now := time.Date(2026, 2, 24, 12, 0, 0, 0, time.UTC)
	return domain.Task{
		ID:             "t1",
		ProjectID:      "p1",
		ColumnID:       "c1",
		Position:       0,
		Title:          "Task One",
		Kind:           domain.WorkKindTask,
		Scope:          domain.KindAppliesToTask,
		LifecycleState: domain.StateTodo,
		Priority:       domain.PriorityMedium,
		CreatedAt:      now,
		UpdatedAt:      now,
	}, nil
}

// UpdateTask returns one deterministic updated task row.
func (s *stubExpandedService) UpdateTask(_ context.Context, in common.UpdateTaskRequest) (domain.Task, error) {
	s.lastUpdateTaskReq = in
	now := time.Date(2026, 2, 24, 12, 0, 0, 0, time.UTC)
	return domain.Task{
		ID:             "t1",
		ProjectID:      "p1",
		ColumnID:       "c1",
		Position:       0,
		Title:          "Task One Updated",
		Kind:           domain.WorkKindTask,
		Scope:          domain.KindAppliesToTask,
		LifecycleState: domain.StateTodo,
		Priority:       domain.PriorityMedium,
		CreatedAt:      now,
		UpdatedAt:      now,
	}, nil
}

// MoveTask returns one deterministic moved task row.
func (s *stubExpandedService) MoveTask(_ context.Context, _ common.MoveTaskRequest) (domain.Task, error) {
	now := time.Date(2026, 2, 24, 12, 0, 0, 0, time.UTC)
	return domain.Task{
		ID:             "t1",
		ProjectID:      "p1",
		ColumnID:       "c2",
		Position:       1,
		Title:          "Task One",
		Kind:           domain.WorkKindTask,
		Scope:          domain.KindAppliesToTask,
		LifecycleState: domain.StateProgress,
		Priority:       domain.PriorityMedium,
		CreatedAt:      now,
		UpdatedAt:      now,
	}, nil
}

// MoveTaskState returns one deterministic moved-by-state task row.
func (s *stubExpandedService) MoveTaskState(_ context.Context, in common.MoveTaskStateRequest) (domain.Task, error) {
	s.lastMoveTaskStateReq = in
	now := time.Date(2026, 2, 24, 12, 0, 0, 0, time.UTC)
	return domain.Task{
		ID:             strings.TrimSpace(in.TaskID),
		ProjectID:      "p1",
		ColumnID:       "c2",
		Position:       0,
		Title:          "Task One",
		Kind:           domain.WorkKindTask,
		Scope:          domain.KindAppliesToTask,
		LifecycleState: domain.StateDone,
		Priority:       domain.PriorityMedium,
		CreatedAt:      now,
		UpdatedAt:      now,
	}, nil
}

// DeleteTask reports deterministic success.
func (s *stubExpandedService) DeleteTask(_ context.Context, _ common.DeleteTaskRequest) error {
	return nil
}

// RestoreTask returns one deterministic restored row.
func (s *stubExpandedService) RestoreTask(_ context.Context, in common.RestoreTaskRequest) (domain.Task, error) {
	s.lastRestoreTaskReq = in
	now := time.Date(2026, 2, 24, 12, 0, 0, 0, time.UTC)
	return domain.Task{
		ID:             "t1",
		ProjectID:      "p1",
		ColumnID:       "c1",
		Position:       0,
		Title:          "Task One",
		Kind:           domain.WorkKindTask,
		Scope:          domain.KindAppliesToTask,
		LifecycleState: domain.StateTodo,
		Priority:       domain.PriorityMedium,
		CreatedAt:      now,
		UpdatedAt:      now,
	}, nil
}

// ReparentTask returns one deterministic reparented row.
func (s *stubExpandedService) ReparentTask(_ context.Context, _ common.ReparentTaskRequest) (domain.Task, error) {
	now := time.Date(2026, 2, 24, 12, 0, 0, 0, time.UTC)
	return domain.Task{
		ID:             "t1",
		ProjectID:      "p1",
		ParentID:       "parent-1",
		ColumnID:       "c1",
		Position:       0,
		Title:          "Task One",
		Kind:           domain.WorkKindTask,
		Scope:          domain.KindAppliesToTask,
		LifecycleState: domain.StateTodo,
		Priority:       domain.PriorityMedium,
		CreatedAt:      now,
		UpdatedAt:      now,
	}, nil
}

// ListChildTasks returns one deterministic child row.
func (s *stubExpandedService) ListChildTasks(_ context.Context, projectID, parentID string, includeArchived bool) ([]domain.Task, error) {
	s.lastListChildProjectID = projectID
	s.lastListChildParentID = parentID
	s.lastListChildArchived = includeArchived
	now := time.Date(2026, 2, 24, 12, 0, 0, 0, time.UTC)
	return []domain.Task{
		{
			ID:             "child-1",
			ProjectID:      "p1",
			ParentID:       "parent-1",
			ColumnID:       "c1",
			Position:       0,
			Title:          "Child",
			Kind:           domain.WorkKindSubtask,
			Scope:          domain.KindAppliesToSubtask,
			LifecycleState: domain.StateTodo,
			Priority:       domain.PriorityMedium,
			CreatedAt:      now,
			UpdatedAt:      now,
		},
	}, nil
}

// SearchTasks returns one deterministic match row plus search metadata.
func (s *stubExpandedService) SearchTasks(_ context.Context, in common.SearchTasksRequest) (common.SearchTasksResult, error) {
	s.lastSearchTasksReq = in
	now := time.Date(2026, 2, 24, 12, 0, 0, 0, time.UTC)
	return common.SearchTasksResult{
		Matches: []common.SearchTaskMatch{{
			Project: domain.Project{ID: "p1", Slug: "proj-1", Name: "Project One", CreatedAt: now, UpdatedAt: now},
			Task: domain.Task{
				ID:             "t1",
				ProjectID:      "p1",
				ColumnID:       "c1",
				Position:       0,
				Title:          "Task One",
				Kind:           domain.WorkKindTask,
				Scope:          domain.KindAppliesToTask,
				LifecycleState: domain.StateTodo,
				Priority:       domain.PriorityMedium,
				CreatedAt:      now,
				UpdatedAt:      now,
			},
			StateID:              "todo",
			EmbeddingSubjectType: "thread_context",
			EmbeddingSubjectID:   "comment-1",
			EmbeddingStatus:      "ready",
			UsedSemantic:         true,
			SemanticScore:        0.91,
		}},
		RequestedMode:          strings.TrimSpace(in.Mode),
		EffectiveMode:          "semantic",
		SemanticAvailable:      true,
		SemanticCandidateCount: 1,
		EmbeddingSummary: common.EmbeddingSummary{
			SubjectType: "work_item",
			ProjectIDs:  []string{"p1"},
			ReadyCount:  1,
		},
	}, nil
}

// GetEmbeddingsStatus returns one deterministic embeddings status response.
func (s *stubExpandedService) GetEmbeddingsStatus(_ context.Context, in common.EmbeddingsStatusRequest) (common.EmbeddingsStatusResult, error) {
	s.lastEmbeddingsStatusReq = in
	now := time.Date(2026, 2, 24, 12, 0, 0, 0, time.UTC)
	return common.EmbeddingsStatusResult{
		ProjectIDs: []string{"p1"},
		Summary: common.EmbeddingSummary{
			SubjectType:  "mixed",
			ProjectIDs:   []string{"p1"},
			PendingCount: 1,
			ReadyCount:   2,
			FailedCount:  1,
			StaleCount:   1,
		},
		Rows: []common.EmbeddingStatusRow{{
			SubjectType:      "work_item",
			SubjectID:        "t1",
			ProjectID:        "p1",
			Status:           "failed",
			ModelSignature:   "fantasy|mini||3",
			LastErrorSummary: "provider unavailable",
			UpdatedAt:        now,
		}, {
			SubjectType:    "thread_context",
			SubjectID:      "c1",
			ProjectID:      "p1",
			Status:         "pending",
			ModelSignature: "fantasy|mini||3",
			UpdatedAt:      now,
		}, {
			SubjectType:    "project_document",
			SubjectID:      "p1",
			ProjectID:      "p1",
			Status:         "ready",
			ModelSignature: "fantasy|mini||3",
			UpdatedAt:      now,
		}},
	}, nil
}

// ReindexEmbeddings returns one deterministic reindex response.
func (s *stubExpandedService) ReindexEmbeddings(_ context.Context, in common.ReindexEmbeddingsRequest) (common.ReindexEmbeddingsResult, error) {
	s.lastEmbeddingsReindexReq = in
	return common.ReindexEmbeddingsResult{
		TargetProjects: []string{"p1"},
		ScannedCount:   3,
		QueuedCount:    2,
		ReadyCount:     1,
		PendingCount:   2,
	}, nil
}

// ListProjectChangeEvents returns one deterministic change row.
func (s *stubExpandedService) ListProjectChangeEvents(_ context.Context, _ string, _ int) ([]domain.ChangeEvent, error) {
	return []domain.ChangeEvent{
		{
			ID:         1,
			ProjectID:  "p1",
			WorkItemID: "t1",
			Operation:  domain.ChangeOperationUpdate,
			ActorID:    "tester",
			ActorName:  "tester",
			ActorType:  domain.ActorTypeUser,
			Metadata:   map[string]string{"field": "title"},
			OccurredAt: time.Date(2026, 2, 24, 12, 0, 0, 0, time.UTC),
		},
	}, nil
}

// GetProjectDependencyRollup returns one deterministic dependency rollup.
func (s *stubExpandedService) GetProjectDependencyRollup(_ context.Context, _ string) (domain.DependencyRollup, error) {
	return domain.DependencyRollup{
		ProjectID:                 "p1",
		TotalItems:                2,
		ItemsWithDependencies:     1,
		DependencyEdges:           1,
		BlockedItems:              1,
		BlockedByEdges:            1,
		UnresolvedDependencyEdges: 1,
	}, nil
}

// ListKindDefinitions returns one deterministic kind row.
func (s *stubExpandedService) ListKindDefinitions(_ context.Context, includeArchived bool) ([]domain.KindDefinition, error) {
	s.lastListKindsArchived = includeArchived
	now := time.Date(2026, 2, 24, 12, 0, 0, 0, time.UTC)
	return []domain.KindDefinition{
		{
			ID:                  domain.KindID("task"),
			DisplayName:         "Task",
			DescriptionMarkdown: "Normal implementation work item. Prefer comments for progress and handoffs for explicit routing.",
			AppliesTo:           []domain.KindAppliesTo{domain.KindAppliesToTask},
			AllowedParentScopes: []domain.KindAppliesTo{domain.KindAppliesToPhase, domain.KindAppliesToTask},
			CreatedAt:           now,
			UpdatedAt:           now,
		},
	}, nil
}

// UpsertKindDefinition returns one deterministic kind row.
func (s *stubExpandedService) UpsertKindDefinition(_ context.Context, in common.UpsertKindDefinitionRequest) (domain.KindDefinition, error) {
	s.lastUpsertKindReq = in
	now := time.Date(2026, 2, 24, 12, 0, 0, 0, time.UTC)
	return domain.KindDefinition{
		ID:          domain.KindID("phase"),
		DisplayName: "Phase",
		AppliesTo:   []domain.KindAppliesTo{domain.KindAppliesToPhase},
		CreatedAt:   now,
		UpdatedAt:   now,
	}, nil
}

// SetProjectAllowedKinds reports deterministic success.
func (s *stubExpandedService) SetProjectAllowedKinds(_ context.Context, in common.SetProjectAllowedKindsRequest) error {
	s.lastSetAllowedKindsReq = in
	return nil
}

// ListProjectAllowedKinds returns deterministic allowlist rows.
func (s *stubExpandedService) ListProjectAllowedKinds(_ context.Context, _ string) ([]string, error) {
	return []string{"build-task", "go-project", "qa-check", "task"}, nil
}

// ListTemplateLibraries returns one deterministic template-library row.
func (s *stubExpandedService) ListTemplateLibraries(_ context.Context, in common.ListTemplateLibrariesRequest) ([]domain.TemplateLibrary, error) {
	s.lastListTemplateReq = in
	now := time.Date(2026, 3, 29, 12, 0, 0, 0, time.UTC)
	return []domain.TemplateLibrary{
		{
			ID:          "go-defaults",
			Scope:       domain.TemplateLibraryScopeGlobal,
			Name:        "Go Defaults",
			Description: "Default Go workflow contract. Use MCP first, test with mage, and route QA explicitly.",
			Status:      domain.TemplateLibraryStatusApproved,
			Revision:    3,
			CreatedAt:   now,
			UpdatedAt:   now,
			NodeTemplates: []domain.NodeTemplate{
				{
					ID:                  "tmpl-task-build",
					LibraryID:           "go-defaults",
					ScopeLevel:          domain.KindAppliesToTask,
					NodeKindID:          domain.KindID("build-task"),
					DisplayName:         "Build Task",
					DescriptionMarkdown: "Build work should stay TDD-first and hand off to QA once implementation evidence is attached.",
					ChildRules: []domain.TemplateChildRule{{
						ID:                      "rule-qa-pass",
						NodeTemplateID:          "tmpl-task-build",
						ChildScopeLevel:         domain.KindAppliesToSubtask,
						ChildKindID:             domain.KindID("qa-check"),
						TitleTemplate:           "QA PROOF REVIEW",
						ResponsibleActorKind:    domain.TemplateActorKindQA,
						EditableByActorKinds:    []domain.TemplateActorKind{domain.TemplateActorKindQA},
						CompletableByActorKinds: []domain.TemplateActorKind{domain.TemplateActorKindQA, domain.TemplateActorKindHuman},
						RequiredForParentDone:   true,
					}},
				},
			},
		},
	}, nil
}

// GetTemplateLibrary returns one deterministic template-library row.
func (s *stubExpandedService) GetTemplateLibrary(_ context.Context, libraryID string) (domain.TemplateLibrary, error) {
	s.lastGetTemplateID = strings.TrimSpace(libraryID)
	rows, _ := s.ListTemplateLibraries(context.Background(), common.ListTemplateLibrariesRequest{})
	return rows[0], nil
}

// GetBuiltinTemplateLibraryStatus returns one deterministic builtin template lifecycle view.
func (s *stubExpandedService) GetBuiltinTemplateLibraryStatus(_ context.Context, libraryID string) (domain.BuiltinTemplateLibraryStatus, error) {
	s.lastGetBuiltinTemplateID = strings.TrimSpace(libraryID)
	return domain.BuiltinTemplateLibraryStatus{
		LibraryID:             firstNonEmptyString(strings.TrimSpace(libraryID), "default-go"),
		Name:                  "Default Go",
		BuiltinSource:         "builtin://tillsyn/default-go",
		BuiltinVersion:        "2026-04-10.2",
		BuiltinRevisionDigest: "builtin-digest",
		RequiredKindIDs:       []domain.KindID{"branch", "build-phase", "build-task", "branch-cleanup-phase", "closeout-phase", "commit-and-reingest", "go-project", "plan-phase", "project-setup-phase", "qa-check", "task"},
		State:                 domain.BuiltinTemplateLibraryStateCurrent,
		Installed:             true,
		InstalledLibraryName:  "Default Go",
		InstalledStatus:       domain.TemplateLibraryStatusApproved,
		InstalledRevision:     3,
		InstalledDigest:       "builtin-digest",
		InstalledBuiltin:      true,
	}, nil
}

// EnsureBuiltinTemplateLibrary returns one deterministic builtin ensure result.
func (s *stubExpandedService) EnsureBuiltinTemplateLibrary(_ context.Context, in common.EnsureBuiltinTemplateLibraryRequest) (domain.BuiltinTemplateLibraryEnsureResult, error) {
	s.lastEnsureBuiltinReq = in
	status, _ := s.GetBuiltinTemplateLibraryStatus(context.Background(), in.LibraryID)
	return domain.BuiltinTemplateLibraryEnsureResult{
		Library: domain.TemplateLibrary{
			ID:             status.LibraryID,
			Name:           status.Name,
			Scope:          domain.TemplateLibraryScopeGlobal,
			Status:         domain.TemplateLibraryStatusApproved,
			BuiltinManaged: true,
			BuiltinSource:  status.BuiltinSource,
			BuiltinVersion: status.BuiltinVersion,
			Revision:       status.InstalledRevision,
			RevisionDigest: status.BuiltinRevisionDigest,
		},
		Status:  status,
		Changed: true,
	}, nil
}

// UpsertTemplateLibrary returns one deterministic updated template-library row.
func (s *stubExpandedService) UpsertTemplateLibrary(_ context.Context, in common.UpsertTemplateLibraryRequest) (domain.TemplateLibrary, error) {
	s.lastUpsertTemplateReq = in
	now := time.Date(2026, 3, 29, 12, 0, 0, 0, time.UTC)
	return domain.TemplateLibrary{
		ID:        strings.TrimSpace(in.ID),
		Scope:     in.Scope,
		ProjectID: strings.TrimSpace(in.ProjectID),
		Name:      strings.TrimSpace(in.Name),
		Status:    in.Status,
		CreatedAt: now,
		UpdatedAt: now,
		NodeTemplates: []domain.NodeTemplate{
			{
				ID:          "tmpl-task-build",
				LibraryID:   strings.TrimSpace(in.ID),
				ScopeLevel:  domain.KindAppliesToTask,
				NodeKindID:  domain.KindID("build-task"),
				DisplayName: "Build Task",
			},
		},
	}, nil
}

// BindProjectTemplateLibrary returns one deterministic project binding.
func (s *stubExpandedService) BindProjectTemplateLibrary(_ context.Context, in common.BindProjectTemplateLibraryRequest) (domain.ProjectTemplateBinding, error) {
	s.lastBindTemplateReq = in
	return domain.ProjectTemplateBinding{
		ProjectID: in.ProjectID,
		LibraryID: in.LibraryID,
		BoundAt:   time.Date(2026, 3, 29, 12, 5, 0, 0, time.UTC),
	}, nil
}

// GetProjectTemplateBinding returns one deterministic project binding.
func (s *stubExpandedService) GetProjectTemplateBinding(_ context.Context, projectID string) (domain.ProjectTemplateBinding, error) {
	s.lastGetTemplateBindingID = strings.TrimSpace(projectID)
	return domain.ProjectTemplateBinding{
		ProjectID:     projectID,
		LibraryID:     "go-defaults",
		LibraryName:   "Go Defaults",
		BoundRevision: 3,
		DriftStatus:   domain.ProjectTemplateBindingDriftCurrent,
		BoundAt:       time.Date(2026, 3, 29, 12, 5, 0, 0, time.UTC),
	}, nil
}

// GetProjectTemplateReapplyPreview returns one deterministic project reapply preview.
func (s *stubExpandedService) GetProjectTemplateReapplyPreview(_ context.Context, projectID string) (domain.ProjectTemplateReapplyPreview, error) {
	s.lastGetTemplatePreviewID = strings.TrimSpace(projectID)
	return domain.ProjectTemplateReapplyPreview{
		ProjectID:              strings.TrimSpace(projectID),
		LibraryID:              "go-defaults",
		LibraryName:            "Go Defaults",
		DriftStatus:            domain.ProjectTemplateBindingDriftUpdateAvailable,
		BoundRevision:          2,
		LatestRevision:         3,
		EligibleMigrationCount: 1,
		ReviewRequired:         true,
		ChildRuleChanges: []domain.ProjectTemplateChildRuleChange{{
			NodeTemplateID:        "build-task-template",
			NodeTemplateName:      "Build Task",
			ChildRuleID:           "qa-proof-review",
			ChangeKinds:           []string{"title", "editable_by"},
			PreviousTitleTemplate: "QA PROOF REVIEW",
			CurrentTitleTemplate:  "QA PROOF REVIEW UPDATE",
		}},
		MigrationCandidates: []domain.ProjectTemplateMigrationCandidate{{
			TaskID:      "task-qa-1",
			Title:       "QA PROOF REVIEW",
			Status:      domain.ProjectTemplateReapplyCandidateEligible,
			ChangeKinds: []string{"title", "editable_by"},
		}},
	}, nil
}

// ApproveProjectTemplateMigrations returns one deterministic migration-approval result.
func (s *stubExpandedService) ApproveProjectTemplateMigrations(_ context.Context, in common.ApproveProjectTemplateMigrationsRequest) (domain.ProjectTemplateMigrationApprovalResult, error) {
	s.lastApproveTemplateMigrationsReq = in
	return domain.ProjectTemplateMigrationApprovalResult{
		ProjectID:              strings.TrimSpace(in.ProjectID),
		LibraryID:              "go-defaults",
		LibraryName:            "Go Defaults",
		DriftStatus:            domain.ProjectTemplateBindingDriftUpdateAvailable,
		ApprovedAll:            in.ApproveAll,
		AppliedCount:           max(1, len(in.TaskIDs)),
		RemainingEligibleCount: 0,
		Approvals: []domain.ProjectTemplateMigrationApproval{{
			TaskID:      "task-qa-1",
			Title:       "QA PROOF REVIEW",
			ChangeKinds: []string{"title", "editable_by"},
			NewTitle:    "QA PROOF REVIEW UPDATE",
		}},
	}, nil
}

// GetNodeContractSnapshot returns one deterministic node-contract snapshot.
func (s *stubExpandedService) GetNodeContractSnapshot(_ context.Context, nodeID string) (domain.NodeContractSnapshot, error) {
	s.lastGetNodeContractID = strings.TrimSpace(nodeID)
	return domain.NodeContractSnapshot{
		NodeID:                  nodeID,
		ProjectID:               "p1",
		SourceLibraryID:         "go-defaults",
		SourceNodeTemplateID:    "tmpl-task-build",
		SourceChildRuleID:       "rule-qa-pass",
		ResponsibleActorKind:    domain.TemplateActorKindQA,
		EditableByActorKinds:    []domain.TemplateActorKind{domain.TemplateActorKindQA},
		CompletableByActorKinds: []domain.TemplateActorKind{domain.TemplateActorKindQA},
		RequiredForParentDone:   true,
		CreatedAt:               time.Date(2026, 3, 29, 12, 6, 0, 0, time.UTC),
	}, nil
}

// newStubCapabilityLease returns one deterministic lease row without mutating request capture state.
func newStubCapabilityLease(in common.IssueCapabilityLeaseRequest) domain.CapabilityLease {
	now := time.Date(2026, 2, 24, 12, 0, 0, 0, time.UTC)
	expiresAt := now.Add(time.Hour)
	return domain.CapabilityLease{
		InstanceID:  "inst-1",
		LeaseToken:  "tok-1",
		AgentName:   strings.TrimSpace(in.AgentName),
		ProjectID:   "p1",
		ScopeType:   domain.CapabilityScopeProject,
		ScopeID:     "p1",
		Role:        domain.CapabilityRoleWorker,
		IssuedAt:    now,
		ExpiresAt:   expiresAt,
		HeartbeatAt: now,
	}
}

// IssueCapabilityLease returns one deterministic lease row.
func (s *stubExpandedService) IssueCapabilityLease(_ context.Context, in common.IssueCapabilityLeaseRequest) (domain.CapabilityLease, error) {
	s.lastIssueLeaseReq = in
	return newStubCapabilityLease(in), nil
}

// ListCapabilityLeases returns one deterministic lease inventory row.
func (s *stubExpandedService) ListCapabilityLeases(_ context.Context, in common.ListCapabilityLeasesRequest) ([]domain.CapabilityLease, error) {
	s.lastListLeaseReq = in
	lease := newStubCapabilityLease(common.IssueCapabilityLeaseRequest{})
	if in.IncludeRevoked {
		revokedAt := time.Date(2026, 2, 24, 13, 0, 0, 0, time.UTC)
		revoked := lease
		revoked.InstanceID = "inst-2"
		revoked.LeaseToken = "tok-2"
		revoked.RevokedAt = &revokedAt
		revoked.RevokedReason = "manual cleanup"
		return []domain.CapabilityLease{lease, revoked}, nil
	}
	return []domain.CapabilityLease{lease}, nil
}

// HeartbeatCapabilityLease returns one deterministic lease row.
func (s *stubExpandedService) HeartbeatCapabilityLease(_ context.Context, _ common.HeartbeatCapabilityLeaseRequest) (domain.CapabilityLease, error) {
	return newStubCapabilityLease(common.IssueCapabilityLeaseRequest{}), nil
}

// RenewCapabilityLease returns one deterministic lease row.
func (s *stubExpandedService) RenewCapabilityLease(_ context.Context, _ common.RenewCapabilityLeaseRequest) (domain.CapabilityLease, error) {
	return newStubCapabilityLease(common.IssueCapabilityLeaseRequest{}), nil
}

// RevokeCapabilityLease returns one deterministic lease row.
func (s *stubExpandedService) RevokeCapabilityLease(_ context.Context, _ common.RevokeCapabilityLeaseRequest) (domain.CapabilityLease, error) {
	lease := newStubCapabilityLease(common.IssueCapabilityLeaseRequest{})
	now := time.Date(2026, 2, 24, 13, 0, 0, 0, time.UTC)
	lease.RevokedAt = &now
	lease.RevokedReason = "test revoke"
	return lease, nil
}

// RevokeAllCapabilityLeases reports deterministic success.
func (s *stubExpandedService) RevokeAllCapabilityLeases(_ context.Context, _ common.RevokeAllCapabilityLeasesRequest) error {
	return nil
}

// CreateComment returns one deterministic comment row.
func (s *stubExpandedService) CreateComment(_ context.Context, in common.CreateCommentRequest) (common.CommentRecord, error) {
	s.lastCreateCommentReq = in
	now := time.Date(2026, 2, 24, 12, 0, 0, 0, time.UTC)
	targetType := domain.NormalizeCommentTargetType(domain.CommentTargetType(in.TargetType))
	if targetType == "" {
		targetType = domain.CommentTargetTypeTask
	}
	return common.CommentRecord{
		ID:           "c1",
		ProjectID:    in.ProjectID,
		TargetType:   string(targetType),
		TargetID:     in.TargetID,
		Summary:      in.Summary,
		BodyMarkdown: in.BodyMarkdown,
		ActorID:      "tester",
		ActorName:    "tester",
		ActorType:    string(domain.ActorTypeUser),
		CreatedAt:    now,
		UpdatedAt:    now,
	}, nil
}

// ListCommentsByTarget returns one deterministic comment row.
func (s *stubExpandedService) ListCommentsByTarget(_ context.Context, in common.ListCommentsByTargetRequest) ([]common.CommentRecord, error) {
	s.lastListCommentReq = in
	comment, _ := s.CreateComment(context.Background(), common.CreateCommentRequest{
		ProjectID:    in.ProjectID,
		TargetType:   in.TargetType,
		TargetID:     in.TargetID,
		Summary:      "Thread summary",
		BodyMarkdown: "Thread summary\n\nDetails",
	})
	return []common.CommentRecord{comment}, nil
}

// CreateHandoff returns one deterministic handoff row.
func (s *stubExpandedService) CreateHandoff(_ context.Context, in common.CreateHandoffRequest) (domain.Handoff, error) {
	s.lastCreateHandoffReq = in
	now := time.Date(2026, 3, 21, 12, 0, 0, 0, time.UTC)
	return domain.Handoff{
		ID:              "handoff-1",
		ProjectID:       strings.TrimSpace(in.ProjectID),
		BranchID:        strings.TrimSpace(in.BranchID),
		ScopeType:       domain.ScopeLevelTask,
		ScopeID:         firstNonEmptyString(strings.TrimSpace(in.ScopeID), "task-1"),
		SourceRole:      strings.TrimSpace(in.SourceRole),
		TargetBranchID:  strings.TrimSpace(in.TargetBranchID),
		TargetScopeType: domain.ScopeLevelTask,
		TargetScopeID:   firstNonEmptyString(strings.TrimSpace(in.TargetScopeID), "task-qa-1"),
		TargetRole:      strings.TrimSpace(in.TargetRole),
		Status:          domain.HandoffStatusWaiting,
		Summary:         strings.TrimSpace(in.Summary),
		NextAction:      strings.TrimSpace(in.NextAction),
		MissingEvidence: append([]string(nil), in.MissingEvidence...),
		RelatedRefs:     append([]string(nil), in.RelatedRefs...),
		CreatedByActor:  "agent-session-1",
		CreatedByType:   domain.ActorTypeAgent,
		CreatedAt:       now,
		UpdatedByActor:  "agent-session-1",
		UpdatedByType:   domain.ActorTypeAgent,
		UpdatedAt:       now,
	}, nil
}

// GetHandoff returns one deterministic handoff row by id.
func (s *stubExpandedService) GetHandoff(_ context.Context, handoffID string) (domain.Handoff, error) {
	s.lastGetHandoffID = handoffID
	return s.CreateHandoff(context.Background(), common.CreateHandoffRequest{
		ProjectID: "p1",
		BranchID:  "branch-1",
		ScopeType: "task",
		ScopeID:   "task-1",
		Summary:   strings.TrimSpace(handoffID),
	})
}

// ListHandoffs returns one deterministic handoff inventory row.
func (s *stubExpandedService) ListHandoffs(_ context.Context, in common.ListHandoffsRequest) ([]domain.Handoff, error) {
	s.lastListHandoffsReq = in
	handoff, _ := s.CreateHandoff(context.Background(), common.CreateHandoffRequest{
		ProjectID:       strings.TrimSpace(in.ProjectID),
		BranchID:        strings.TrimSpace(in.BranchID),
		ScopeType:       strings.TrimSpace(in.ScopeType),
		ScopeID:         strings.TrimSpace(in.ScopeID),
		SourceRole:      "builder",
		TargetBranchID:  "branch-1",
		TargetScopeType: "task",
		TargetScopeID:   "task-qa-1",
		TargetRole:      "qa",
		Summary:         "handoff summary",
		NextAction:      "run qa",
	})
	if len(in.Statuses) > 0 {
		handoff.Status = domain.HandoffStatus(strings.TrimSpace(in.Statuses[0]))
	}
	return []domain.Handoff{handoff}, nil
}

// UpdateHandoff returns one deterministic updated handoff row.
func (s *stubExpandedService) UpdateHandoff(_ context.Context, in common.UpdateHandoffRequest) (domain.Handoff, error) {
	s.lastUpdateHandoffReq = in
	handoff, _ := s.CreateHandoff(context.Background(), common.CreateHandoffRequest{
		ProjectID:       "p1",
		BranchID:        "branch-1",
		ScopeType:       "task",
		ScopeID:         "task-1",
		SourceRole:      in.SourceRole,
		TargetBranchID:  in.TargetBranchID,
		TargetScopeType: in.TargetScopeType,
		TargetScopeID:   in.TargetScopeID,
		TargetRole:      in.TargetRole,
		Summary:         in.Summary,
		NextAction:      in.NextAction,
		MissingEvidence: in.MissingEvidence,
		RelatedRefs:     in.RelatedRefs,
	})
	handoff.ID = strings.TrimSpace(in.HandoffID)
	if trimmed := strings.TrimSpace(in.Status); trimmed != "" {
		handoff.Status = domain.HandoffStatus(trimmed)
	}
	handoff.ResolutionNote = strings.TrimSpace(in.ResolutionNote)
	return handoff, nil
}

// findToolSchemaByName returns one tool schema map from tools/list payload rows.
func findToolSchemaByName(t *testing.T, tools []any, toolName string) map[string]any {
	t.Helper()
	for _, toolRaw := range tools {
		toolMap, ok := toolRaw.(map[string]any)
		if !ok {
			continue
		}
		name, _ := toolMap["name"].(string)
		if name != toolName {
			continue
		}
		schema, ok := toolMap["inputSchema"].(map[string]any)
		if !ok {
			t.Fatalf("tool %q inputSchema missing: %#v", toolName, toolMap)
		}
		return schema
	}
	t.Fatalf("tool %q missing from tool list", toolName)
	return nil
}

// schemaStringPropertyDescription returns one schema property description for assertions.
func schemaStringPropertyDescription(t *testing.T, schema map[string]any, property string) string {
	t.Helper()
	properties, ok := schema["properties"].(map[string]any)
	if !ok {
		t.Fatalf("schema properties missing: %#v", schema)
	}
	propRaw, ok := properties[property].(map[string]any)
	if !ok {
		t.Fatalf("property %q missing from schema: %#v", property, properties)
	}
	description, _ := propRaw["description"].(string)
	return description
}

// schemaPropertyEnumStrings returns schema enum values for a property as strings.
func schemaPropertyEnumStrings(t *testing.T, schema map[string]any, property string) []string {
	t.Helper()
	properties, ok := schema["properties"].(map[string]any)
	if !ok {
		t.Fatalf("schema properties missing: %#v", schema)
	}
	propRaw, ok := properties[property].(map[string]any)
	if !ok {
		t.Fatalf("property %q missing from schema: %#v", property, properties)
	}
	enumRaw, _ := propRaw["enum"].([]any)
	enum := make([]string, 0, len(enumRaw))
	for _, item := range enumRaw {
		value, ok := item.(string)
		if !ok {
			continue
		}
		enum = append(enum, value)
	}
	return enum
}

// schemaPropertyNumberField returns one numeric schema field value for assertions.
func schemaPropertyNumberField(t *testing.T, schema map[string]any, property, field string) float64 {
	t.Helper()
	properties, ok := schema["properties"].(map[string]any)
	if !ok {
		t.Fatalf("schema properties missing: %#v", schema)
	}
	propRaw, ok := properties[property].(map[string]any)
	if !ok {
		t.Fatalf("property %q missing from schema: %#v", property, properties)
	}
	raw, ok := propRaw[field]
	if !ok {
		t.Fatalf("property %q missing numeric field %q: %#v", property, field, propRaw)
	}
	switch value := raw.(type) {
	case float64:
		return value
	case int:
		return float64(value)
	case int32:
		return float64(value)
	case int64:
		return float64(value)
	default:
		t.Fatalf("property %q field %q has non-numeric type %T (%#v)", property, field, raw, raw)
	}
	return 0
}

// joinAnyStrings joins JSON-decoded string arrays into one assertion-friendly string.
func joinAnyStrings(values []any) string {
	parts := make([]string, 0, len(values))
	for _, value := range values {
		text, _ := value.(string)
		text = strings.TrimSpace(text)
		if text == "" {
			continue
		}
		parts = append(parts, text)
	}
	return strings.Join(parts, " | ")
}

// TestHandlerExpandedToolSurfaceSuccessPaths exercises success paths for the expanded MCP tool set.
func TestHandlerExpandedToolSurfaceSuccessPaths(t *testing.T) {
	service := &stubExpandedService{
		stubCaptureStateReader: stubCaptureStateReader{
			captureState: common.CaptureState{StateHash: "abc123"},
		},
	}
	handler, err := NewHandler(Config{}, service, nil)
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}

	server := httptest.NewServer(handler)
	defer server.Close()
	_, _ = postJSONRPC(t, server.Client(), server.URL, initializeRequest())
	_, toolsResp := postJSONRPC(t, server.Client(), server.URL, map[string]any{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "tools/list",
	})
	toolsRaw, ok := toolsResp.Result["tools"].([]any)
	if !ok {
		t.Fatalf("tools list payload missing tools: %#v", toolsResp.Result)
	}
	toolNames := make([]string, 0, len(toolsRaw))
	for _, toolRaw := range toolsRaw {
		toolMap, ok := toolRaw.(map[string]any)
		if !ok {
			continue
		}
		name, _ := toolMap["name"].(string)
		toolNames = append(toolNames, name)
	}
	requiredTools := []string{
		"till.get_bootstrap_guide",
		"till.get_instructions",
		"till.auth_request",
		"till.plan_item",
		"till.project",
		"till.embeddings",
		"till.kind",
		"till.template",
		"till.capability_lease",
		"till.comment",
		"till.handoff",
	}
	for _, toolName := range requiredTools {
		found := false
		for _, candidate := range toolNames {
			if candidate == toolName {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("tool %q missing from expanded surface: %#v", toolName, toolNames)
		}
	}

	calls := []struct {
		name string
		args map[string]any
	}{
		{name: "till.get_bootstrap_guide", args: map[string]any{}},
		{name: "till.get_instructions", args: map[string]any{"include_markdown": false}},
		{name: "till.project", args: map[string]any{"operation": "list", "include_archived": true}},
		{name: "till.project", args: mergeArgs(validSessionArgs(), map[string]any{
			"operation":         "create",
			"name":              "Project One",
			"agent_instance_id": "inst-1",
			"lease_token":       "tok-1",
		})},
		{name: "till.project", args: mergeArgs(validSessionArgs(), map[string]any{
			"operation":         "update",
			"project_id":        "p1",
			"name":              "Project One Updated",
			"agent_instance_id": "inst-1",
			"lease_token":       "tok-1",
		})},
		{name: "till.auth_request", args: map[string]any{
			"operation":      "create",
			"path":           "project/p1",
			"principal_id":   "review-agent",
			"principal_type": "agent",
			"principal_role": "research",
			"client_id":      "till-mcp-stdio",
			"client_type":    "mcp-stdio",
			"requested_ttl":  "2h",
			"timeout":        "30m",
			"reason":         "manual MCP review",
		}},
		{name: "till.auth_request", args: map[string]any{"operation": "list", "project_id": "p1", "state": "pending", "limit": 10}},
		{name: "till.auth_request", args: map[string]any{"operation": "get", "request_id": "req-1"}},
		{name: "till.auth_request", args: map[string]any{"operation": "claim", "request_id": "req-1", "resume_token": "resume-123", "principal_id": "review-agent", "client_id": "till-mcp-stdio"}},
		{name: "till.auth_request", args: map[string]any{"operation": "cancel", "request_id": "req-1", "resume_token": "resume-123", "principal_id": "review-agent", "client_id": "till-mcp-stdio", "resolution_note": "superseded"}},
		{name: "till.auth_request", args: map[string]any{
			"operation":             "check_session_governance",
			"session_id":            "sess-global",
			"acting_session_id":     "sess-1",
			"acting_session_secret": "secret-1",
		}},
		{name: "till.plan_item", args: map[string]any{"operation": "list", "project_id": "p1"}},
		{name: "till.plan_item", args: map[string]any{"operation": "get", "task_id": "t1"}},
		{name: "till.plan_item", args: mergeArgs(validSessionArgs(), map[string]any{
			"operation":         "create",
			"project_id":        "p1",
			"column_id":         "c1",
			"title":             "Task One",
			"agent_instance_id": "inst-1",
			"lease_token":       "tok-1",
		})},
		{name: "till.plan_item", args: mergeArgs(validSessionArgs(), map[string]any{
			"operation":         "update",
			"task_id":           "t1",
			"title":             "Task One Updated",
			"agent_instance_id": "inst-1",
			"lease_token":       "tok-1",
		})},
		{name: "till.plan_item", args: mergeArgs(validSessionArgs(), map[string]any{
			"operation":         "move",
			"task_id":           "t1",
			"to_column_id":      "c2",
			"position":          1,
			"agent_instance_id": "inst-1",
			"lease_token":       "tok-1",
		})},
		{name: "till.plan_item", args: mergeArgs(validSessionArgs(), map[string]any{
			"operation":         "move_state",
			"task_id":           "t1",
			"state":             "done",
			"agent_instance_id": "inst-1",
			"lease_token":       "tok-1",
		})},
		{name: "till.plan_item", args: mergeArgs(validSessionArgs(), map[string]any{
			"operation":         "delete",
			"task_id":           "t1",
			"agent_instance_id": "inst-1",
			"lease_token":       "tok-1",
		})},
		{name: "till.plan_item", args: mergeArgs(validSessionArgs(), map[string]any{
			"operation":         "restore",
			"task_id":           "t1",
			"agent_instance_id": "inst-1",
			"lease_token":       "tok-1",
		})},
		{name: "till.plan_item", args: mergeArgs(validSessionArgs(), map[string]any{
			"operation":         "reparent",
			"task_id":           "t1",
			"parent_id":         "parent-1",
			"agent_instance_id": "inst-1",
			"lease_token":       "tok-1",
		})},
		{name: "till.plan_item", args: map[string]any{"operation": "list", "project_id": "p1", "parent_id": "parent-1"}},
		{name: "till.plan_item", args: map[string]any{"operation": "search", "project_id": "p1", "query": "task"}},
		{name: "till.project", args: map[string]any{"operation": "list_change_events", "project_id": "p1", "limit": 25}},
		{name: "till.project", args: map[string]any{"operation": "get_dependency_rollup", "project_id": "p1"}},
		{name: "till.kind", args: map[string]any{"operation": "list"}},
		{name: "till.kind", args: mergeArgs(validSessionArgs(), map[string]any{"operation": "upsert", "id": "phase", "applies_to": []any{"phase"}})},
		{name: "till.project", args: mergeArgs(validSessionArgs(), map[string]any{"operation": "set_allowed_kinds", "project_id": "p1", "kind_ids": []any{"phase", "task"}})},
		{name: "till.project", args: map[string]any{"operation": "list_allowed_kinds", "project_id": "p1"}},
		{name: "till.template", args: map[string]any{"operation": "list", "scope": "global", "status": "approved"}},
		{name: "till.template", args: map[string]any{"operation": "get", "library_id": "go-defaults"}},
		{name: "till.template", args: map[string]any{"operation": "get_builtin_status", "library_id": "default-go"}},
		{name: "till.template", args: mergeArgs(validSessionArgs(), map[string]any{"operation": "ensure_builtin", "library_id": "default-go"})},
		{name: "till.template", args: mergeArgs(validSessionArgs(), map[string]any{
			"operation": "upsert",
			"library": map[string]any{
				"id":     "go-defaults",
				"scope":  "global",
				"name":   "Go Defaults",
				"status": "approved",
				"node_templates": []any{
					map[string]any{
						"id":           "tmpl-task-build",
						"scope_level":  "task",
						"node_kind_id": "phase",
						"display_name": "Build Task",
					},
				},
			},
		})},
		{name: "till.project", args: mergeArgs(validSessionArgs(), map[string]any{"operation": "bind_template", "project_id": "p1", "template_library_id": "go-defaults"})},
		{name: "till.project", args: map[string]any{"operation": "get_template_binding", "project_id": "p1"}},
		{name: "till.project", args: map[string]any{"operation": "preview_template_reapply", "project_id": "p1"}},
		{name: "till.project", args: mergeArgs(validSessionArgs(), map[string]any{
			"operation":         "approve_template_migrations",
			"project_id":        "p1",
			"task_ids":          []any{"task-qa-1"},
			"agent_instance_id": "inst-1",
			"lease_token":       "tok-1",
		})},
		{name: "till.template", args: map[string]any{"operation": "get_node_contract", "node_id": "task-qa-1"}},
		{name: "till.embeddings", args: map[string]any{"operation": "status", "project_id": "p1", "limit": 10}},
		{name: "till.embeddings", args: map[string]any{"operation": "reindex", "project_id": "p1", "wait": true}},
		{name: "till.capability_lease", args: map[string]any{"operation": "list", "project_id": "p1", "scope_type": "project", "include_revoked": true}},
		{name: "till.capability_lease", args: mergeArgs(validSessionArgs(), map[string]any{"operation": "issue", "project_id": "p1", "scope_type": "project", "role": "research", "agent_name": "agent-1"})},
		{name: "till.capability_lease", args: mergeArgs(validSessionArgs(), map[string]any{"operation": "heartbeat", "agent_instance_id": "inst-1", "lease_token": "tok-1"})},
		{name: "till.capability_lease", args: mergeArgs(validSessionArgs(), map[string]any{"operation": "renew", "agent_instance_id": "inst-1", "lease_token": "tok-1", "ttl_seconds": 60})},
		{name: "till.capability_lease", args: mergeArgs(validSessionArgs(), map[string]any{"operation": "revoke", "agent_instance_id": "inst-1"})},
		{name: "till.capability_lease", args: mergeArgs(validSessionArgs(), map[string]any{"operation": "revoke_all", "project_id": "p1", "scope_type": "project"})},
		{name: "till.comment", args: mergeArgs(validSessionArgs(), map[string]any{
			"operation":         "create",
			"project_id":        "p1",
			"target_type":       "task",
			"target_id":         "t1",
			"summary":           "Thread summary",
			"body_markdown":     "hello",
			"agent_instance_id": "inst-1",
			"lease_token":       "tok-1",
		})},
		{name: "till.comment", args: map[string]any{"operation": "list", "project_id": "p1", "target_type": "task", "target_id": "t1"}},
		{name: "till.handoff", args: mergeArgs(validSessionArgs(), map[string]any{
			"operation":         "create",
			"project_id":        "p1",
			"branch_id":         "branch-1",
			"scope_type":        "task",
			"scope_id":          "task-1",
			"source_role":       "builder",
			"target_branch_id":  "branch-1",
			"target_scope_type": "task",
			"target_scope_id":   "task-qa-1",
			"target_role":       "qa",
			"status":            "waiting",
			"summary":           "handoff summary",
			"next_action":       "run qa",
			"missing_evidence":  []any{"qa note"},
			"related_refs":      []any{"comment:c1"},
			"agent_instance_id": "inst-1",
			"lease_token":       "tok-1",
		})},
		{name: "till.handoff", args: map[string]any{"operation": "get", "handoff_id": "handoff-1"}},
		{name: "till.handoff", args: map[string]any{"operation": "list", "project_id": "p1", "branch_id": "branch-1", "scope_type": "task", "scope_id": "task-1", "statuses": []any{"waiting"}, "limit": 10}},
		{name: "till.handoff", args: mergeArgs(validSessionArgs(), map[string]any{
			"operation":         "update",
			"handoff_id":        "handoff-1",
			"status":            "resolved",
			"source_role":       "builder",
			"target_branch_id":  "branch-1",
			"target_scope_type": "task",
			"target_scope_id":   "task-qa-1",
			"target_role":       "qa",
			"summary":           "handoff summary",
			"next_action":       "none",
			"missing_evidence":  []any{},
			"related_refs":      []any{"comment:c1"},
			"resolution_note":   "qa passed",
			"agent_instance_id": "inst-1",
			"lease_token":       "tok-1",
		})},
	}
	for idx, tc := range calls {
		resp, callResp := postJSONRPC(t, server.Client(), server.URL, callToolRequest(100+idx, tc.name, tc.args))
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("tool %q status = %d, want %d", tc.name, resp.StatusCode, http.StatusOK)
		}
		if isError, _ := callResp.Result["isError"].(bool); isError {
			t.Fatalf("tool %q returned isError=true: %#v", tc.name, callResp.Result)
		}
	}
	if got := service.lastApproveTemplateMigrationsReq.ProjectID; got != "p1" {
		t.Fatalf("approve_template_migrations project_id = %q, want p1", got)
	}
	if got := service.lastApproveTemplateMigrationsReq.TaskIDs; !slices.Equal(got, []string{"task-qa-1"}) {
		t.Fatalf("approve_template_migrations task_ids = %#v, want [task-qa-1]", got)
	}
	if got := service.lastCreateAuthRequestReq.PrincipalRole; got != "research" {
		t.Fatalf("create auth request principal_role = %q, want research", got)
	}
	if got := service.lastCheckAuthSessionReq.SessionID; got != "sess-global" {
		t.Fatalf("check_session_governance session_id = %q, want sess-global", got)
	}
	if got := service.lastCheckAuthSessionReq.ActingSessionID; got != "sess-1" {
		t.Fatalf("check_session_governance acting_session_id = %q, want sess-1", got)
	}
	if got := service.lastIssueLeaseReq.Role; got != "research" {
		t.Fatalf("issue capability lease role = %q, want research", got)
	}
}

// TestHandlerExpandedLeaseToolVisibility verifies the reduced lease surface is default and legacy aliases are opt-in.
func TestHandlerExpandedLeaseToolVisibility(t *testing.T) {
	t.Parallel()

	collectToolNames := func(t *testing.T, cfg Config) []string {
		t.Helper()
		service := &stubExpandedService{
			stubCaptureStateReader: stubCaptureStateReader{
				captureState: common.CaptureState{StateHash: "abc123"},
			},
		}
		handler, err := NewHandler(cfg, service, nil)
		if err != nil {
			t.Fatalf("NewHandler() error = %v", err)
		}
		server := httptest.NewServer(handler)
		defer server.Close()
		_, _ = postJSONRPC(t, server.Client(), server.URL, initializeRequest())
		_, toolsResp := postJSONRPC(t, server.Client(), server.URL, map[string]any{
			"jsonrpc": "2.0",
			"id":      2,
			"method":  "tools/list",
		})
		toolsRaw, ok := toolsResp.Result["tools"].([]any)
		if !ok {
			t.Fatalf("tools list payload missing tools: %#v", toolsResp.Result)
		}
		toolNames := make([]string, 0, len(toolsRaw))
		for _, toolRaw := range toolsRaw {
			toolMap, ok := toolRaw.(map[string]any)
			if !ok {
				continue
			}
			name, _ := toolMap["name"].(string)
			toolNames = append(toolNames, name)
		}
		return toolNames
	}

	defaultTools := collectToolNames(t, Config{})
	if !slices.Contains(defaultTools, "till.capability_lease") {
		t.Fatalf("default surface missing till.capability_lease: %#v", defaultTools)
	}
	if slices.Contains(defaultTools, "till.list_capability_leases") {
		t.Fatalf("unexpected legacy lease read tool in default surface: %#v", defaultTools)
	}
	for _, legacy := range []string{
		"till.issue_capability_lease",
		"till.heartbeat_capability_lease",
		"till.renew_capability_lease",
		"till.revoke_capability_lease",
		"till.revoke_all_capability_leases",
	} {
		if slices.Contains(defaultTools, legacy) {
			t.Fatalf("unexpected legacy lease tool %q in default surface: %#v", legacy, defaultTools)
		}
	}

	legacyTools := collectToolNames(t, Config{ExposeLegacyLeaseTools: true})
	for _, required := range []string{
		"till.capability_lease",
		"till.list_capability_leases",
		"till.issue_capability_lease",
		"till.heartbeat_capability_lease",
		"till.renew_capability_lease",
		"till.revoke_capability_lease",
		"till.revoke_all_capability_leases",
	} {
		if !slices.Contains(legacyTools, required) {
			t.Fatalf("legacy lease mode missing %q: %#v", required, legacyTools)
		}
	}
}

// TestHandlerExpandedCoordinationToolVisibility verifies reduced coordination mutations are default and legacy aliases are opt-in.
func TestHandlerExpandedCoordinationToolVisibility(t *testing.T) {
	t.Parallel()

	collectToolNames := func(t *testing.T, cfg Config) []string {
		t.Helper()
		service := &stubExpandedService{
			stubCaptureStateReader: stubCaptureStateReader{
				captureState: common.CaptureState{StateHash: "abc123"},
			},
		}
		handler, err := NewHandler(cfg, service, &stubAttentionService{})
		if err != nil {
			t.Fatalf("NewHandler() error = %v", err)
		}
		server := httptest.NewServer(handler)
		defer server.Close()
		_, _ = postJSONRPC(t, server.Client(), server.URL, initializeRequest())
		_, toolsResp := postJSONRPC(t, server.Client(), server.URL, map[string]any{
			"jsonrpc": "2.0",
			"id":      2,
			"method":  "tools/list",
		})
		toolsRaw, ok := toolsResp.Result["tools"].([]any)
		if !ok {
			t.Fatalf("tools list payload missing tools: %#v", toolsResp.Result)
		}
		toolNames := make([]string, 0, len(toolsRaw))
		for _, toolRaw := range toolsRaw {
			toolMap, ok := toolRaw.(map[string]any)
			if !ok {
				continue
			}
			name, _ := toolMap["name"].(string)
			toolNames = append(toolNames, name)
		}
		return toolNames
	}

	defaultTools := collectToolNames(t, Config{})
	for _, required := range []string{"till.attention_item", "till.handoff"} {
		if !slices.Contains(defaultTools, required) {
			t.Fatalf("default coordination surface missing %q: %#v", required, defaultTools)
		}
	}
	for _, legacyRead := range []string{"till.list_attention_items", "till.get_handoff", "till.list_handoffs"} {
		if slices.Contains(defaultTools, legacyRead) {
			t.Fatalf("unexpected legacy coordination read tool %q in default surface: %#v", legacyRead, defaultTools)
		}
	}
	for _, legacy := range []string{
		"till.raise_attention_item",
		"till.resolve_attention_item",
		"till.create_handoff",
		"till.update_handoff",
	} {
		if slices.Contains(defaultTools, legacy) {
			t.Fatalf("unexpected legacy coordination tool %q in default surface: %#v", legacy, defaultTools)
		}
	}

	legacyTools := collectToolNames(t, Config{ExposeLegacyCoordinationTools: true})
	for _, required := range []string{
		"till.attention_item",
		"till.handoff",
		"till.list_attention_items",
		"till.get_handoff",
		"till.list_handoffs",
		"till.raise_attention_item",
		"till.resolve_attention_item",
		"till.create_handoff",
		"till.update_handoff",
	} {
		if !slices.Contains(legacyTools, required) {
			t.Fatalf("legacy coordination mode missing %q: %#v", required, legacyTools)
		}
	}
}

// TestHandlerExpandedProjectToolVisibility verifies reduced project mutations are default and legacy aliases are opt-in.
func TestHandlerExpandedProjectToolVisibility(t *testing.T) {
	t.Parallel()

	collectToolNames := func(t *testing.T, cfg Config) []string {
		t.Helper()
		service := &stubExpandedService{
			stubCaptureStateReader: stubCaptureStateReader{
				captureState: common.CaptureState{StateHash: "abc123"},
			},
		}
		handler, err := NewHandler(cfg, service, nil)
		if err != nil {
			t.Fatalf("NewHandler() error = %v", err)
		}
		server := httptest.NewServer(handler)
		defer server.Close()
		_, _ = postJSONRPC(t, server.Client(), server.URL, initializeRequest())
		_, toolsResp := postJSONRPC(t, server.Client(), server.URL, map[string]any{
			"jsonrpc": "2.0",
			"id":      2,
			"method":  "tools/list",
		})
		toolsRaw, ok := toolsResp.Result["tools"].([]any)
		if !ok {
			t.Fatalf("tools list payload missing tools: %#v", toolsResp.Result)
		}
		toolNames := make([]string, 0, len(toolsRaw))
		for _, toolRaw := range toolsRaw {
			toolMap, ok := toolRaw.(map[string]any)
			if !ok {
				continue
			}
			name, _ := toolMap["name"].(string)
			toolNames = append(toolNames, name)
		}
		return toolNames
	}

	defaultTools := collectToolNames(t, Config{})
	if !slices.Contains(defaultTools, "till.project") {
		t.Fatalf("default project surface missing till.project: %#v", defaultTools)
	}
	for _, legacy := range []string{
		"till.list_projects",
		"till.create_project",
		"till.update_project",
		"till.list_kind_definitions",
		"till.upsert_kind_definition",
		"till.set_project_allowed_kinds",
		"till.bind_project_template_library",
		"till.list_template_libraries",
		"till.get_template_library",
		"till.upsert_template_library",
	} {
		if slices.Contains(defaultTools, legacy) {
			t.Fatalf("unexpected legacy project tool %q in default surface: %#v", legacy, defaultTools)
		}
	}

	legacyTools := collectToolNames(t, Config{ExposeLegacyProjectTools: true})
	for _, required := range []string{
		"till.project",
		"till.list_projects",
		"till.create_project",
		"till.update_project",
		"till.list_kind_definitions",
		"till.upsert_kind_definition",
		"till.set_project_allowed_kinds",
		"till.bind_project_template_library",
		"till.list_template_libraries",
		"till.get_template_library",
		"till.upsert_template_library",
	} {
		if !slices.Contains(legacyTools, required) {
			t.Fatalf("legacy project mode missing %q: %#v", required, legacyTools)
		}
	}
}

// TestHandlerExpandedPlanItemToolVisibility verifies reduced plan-item mutations are default and legacy task aliases are opt-in.
func TestHandlerExpandedPlanItemToolVisibility(t *testing.T) {
	t.Parallel()

	collectToolNames := func(t *testing.T, cfg Config) []string {
		t.Helper()
		service := &stubExpandedService{
			stubCaptureStateReader: stubCaptureStateReader{
				captureState: common.CaptureState{StateHash: "abc123"},
			},
		}
		handler, err := NewHandler(cfg, service, nil)
		if err != nil {
			t.Fatalf("NewHandler() error = %v", err)
		}
		server := httptest.NewServer(handler)
		defer server.Close()
		_, _ = postJSONRPC(t, server.Client(), server.URL, initializeRequest())
		_, toolsResp := postJSONRPC(t, server.Client(), server.URL, map[string]any{
			"jsonrpc": "2.0",
			"id":      2,
			"method":  "tools/list",
		})
		toolsRaw, ok := toolsResp.Result["tools"].([]any)
		if !ok {
			t.Fatalf("tools list payload missing tools: %#v", toolsResp.Result)
		}
		toolNames := make([]string, 0, len(toolsRaw))
		for _, toolRaw := range toolsRaw {
			toolMap, ok := toolRaw.(map[string]any)
			if !ok {
				continue
			}
			name, _ := toolMap["name"].(string)
			toolNames = append(toolNames, name)
		}
		return toolNames
	}

	defaultTools := collectToolNames(t, Config{})
	if !slices.Contains(defaultTools, "till.plan_item") {
		t.Fatalf("default plan-item surface missing till.plan_item: %#v", defaultTools)
	}
	for _, legacy := range []string{
		"till.list_tasks",
		"till.create_task",
		"till.update_task",
		"till.move_task",
		"till.delete_task",
		"till.restore_task",
		"till.reparent_task",
		"till.list_child_tasks",
		"till.search_task_matches",
	} {
		if slices.Contains(defaultTools, legacy) {
			t.Fatalf("unexpected legacy task tool %q in default surface: %#v", legacy, defaultTools)
		}
	}

	legacyTools := collectToolNames(t, Config{ExposeLegacyPlanItemTools: true})
	for _, required := range []string{
		"till.plan_item",
		"till.list_tasks",
		"till.create_task",
		"till.update_task",
		"till.move_task",
		"till.delete_task",
		"till.restore_task",
		"till.reparent_task",
		"till.list_child_tasks",
		"till.search_task_matches",
	} {
		if !slices.Contains(legacyTools, required) {
			t.Fatalf("legacy plan-item mode missing %q: %#v", required, legacyTools)
		}
	}
}

// TestHandlerExpandedLegacyPlanItemMutationAliases verifies the legacy task mutation aliases still execute when enabled.
func TestHandlerExpandedLegacyPlanItemMutationAliases(t *testing.T) {
	t.Parallel()

	service := &stubExpandedService{
		stubCaptureStateReader: stubCaptureStateReader{
			captureState: common.CaptureState{StateHash: "abc123"},
		},
		stubMutationAuthorizer: stubMutationAuthorizer{
			authCaller: domain.AuthenticatedCaller{
				PrincipalID:   "agent-session-1",
				PrincipalName: "Agent Session One",
				PrincipalType: domain.ActorTypeAgent,
				SessionID:     "sess-1",
			},
		},
	}
	handler, err := NewHandler(Config{ExposeLegacyPlanItemTools: true}, service, nil)
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}

	server := httptest.NewServer(handler)
	defer server.Close()
	_, _ = postJSONRPC(t, server.Client(), server.URL, initializeRequest())

	callCases := []struct {
		name string
		tool string
		args map[string]any
	}{
		{
			name: "create_task",
			tool: "till.create_task",
			args: mergeArgs(validSessionArgs(), map[string]any{
				"project_id":        "p1",
				"column_id":         "c1",
				"title":             "Task One",
				"agent_instance_id": "inst-1",
				"lease_token":       "tok-1",
			}),
		},
		{
			name: "update_task",
			tool: "till.update_task",
			args: mergeArgs(validSessionArgs(), map[string]any{
				"task_id":           "t1",
				"title":             "Task One Updated",
				"agent_instance_id": "inst-1",
				"lease_token":       "tok-1",
			}),
		},
		{
			name: "move_task",
			tool: "till.move_task",
			args: mergeArgs(validSessionArgs(), map[string]any{
				"task_id":           "t1",
				"to_column_id":      "c2",
				"position":          1,
				"agent_instance_id": "inst-1",
				"lease_token":       "tok-1",
			}),
		},
		{
			name: "delete_task",
			tool: "till.delete_task",
			args: mergeArgs(validSessionArgs(), map[string]any{
				"task_id":           "t1",
				"mode":              "archive",
				"agent_instance_id": "inst-1",
				"lease_token":       "tok-1",
			}),
		},
		{
			name: "restore_task",
			tool: "till.restore_task",
			args: mergeArgs(validSessionArgs(), map[string]any{
				"task_id":           "t1",
				"agent_instance_id": "inst-1",
				"lease_token":       "tok-1",
			}),
		},
		{
			name: "reparent_task",
			tool: "till.reparent_task",
			args: mergeArgs(validSessionArgs(), map[string]any{
				"task_id":           "t1",
				"parent_id":         "parent-1",
				"agent_instance_id": "inst-1",
				"lease_token":       "tok-1",
			}),
		},
	}

	for idx, tc := range callCases {
		_, callResp := postJSONRPC(t, server.Client(), server.URL, callToolRequest(3300+idx, tc.tool, tc.args))
		if isError, _ := callResp.Result["isError"].(bool); isError {
			t.Fatalf("%s returned isError=true: %#v", tc.name, callResp.Result)
		}
	}
}

// TestHandlerExpandedLegacyProjectMutationAliases verifies the legacy project-root mutation aliases still execute when enabled.
func TestHandlerExpandedLegacyProjectMutationAliases(t *testing.T) {
	t.Parallel()

	service := &stubExpandedService{
		stubCaptureStateReader: stubCaptureStateReader{
			captureState: common.CaptureState{StateHash: "abc123"},
		},
		stubMutationAuthorizer: stubMutationAuthorizer{
			authCaller: domain.AuthenticatedCaller{
				PrincipalID:   "agent-session-1",
				PrincipalName: "Agent Session One",
				PrincipalType: domain.ActorTypeAgent,
				SessionID:     "sess-1",
			},
		},
	}
	handler, err := NewHandler(Config{ExposeLegacyProjectTools: true}, service, nil)
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}

	server := httptest.NewServer(handler)
	defer server.Close()
	_, _ = postJSONRPC(t, server.Client(), server.URL, initializeRequest())

	callCases := []struct {
		name string
		tool string
		args map[string]any
	}{
		{
			name: "create_project",
			tool: "till.create_project",
			args: mergeArgs(validSessionArgs(), map[string]any{
				"name":                "Project One",
				"kind":                "go-service",
				"template_library_id": "go-defaults",
				"agent_instance_id":   "inst-1",
				"lease_token":         "tok-1",
			}),
		},
		{
			name: "update_project",
			tool: "till.update_project",
			args: mergeArgs(validSessionArgs(), map[string]any{
				"project_id":        "p1",
				"name":              "Project One Updated",
				"agent_instance_id": "inst-1",
				"lease_token":       "tok-1",
			}),
		},
		{
			name: "set_project_allowed_kinds",
			tool: "till.set_project_allowed_kinds",
			args: mergeArgs(validSessionArgs(), map[string]any{
				"project_id": "p1",
				"kind_ids":   []any{"phase", "task"},
			}),
		},
		{
			name: "bind_project_template_library",
			tool: "till.bind_project_template_library",
			args: mergeArgs(validSessionArgs(), map[string]any{
				"project_id": "p1",
				"library_id": "go-defaults",
			}),
		},
	}

	for idx, tc := range callCases {
		_, callResp := postJSONRPC(t, server.Client(), server.URL, callToolRequest(1400+idx, tc.tool, tc.args))
		if isError, _ := callResp.Result["isError"].(bool); isError {
			t.Fatalf("%s returned isError=true: %#v", tc.name, callResp.Result)
		}
	}

	if got := service.lastCreateProjectReq.TemplateLibraryID; got != "go-defaults" {
		t.Fatalf("legacy create_project template_library_id = %q, want go-defaults", got)
	}
	if got := service.lastBindTemplateReq.LibraryID; got != "go-defaults" {
		t.Fatalf("legacy bind_project_template_library library_id = %q, want go-defaults", got)
	}
}

// TestHandlerExpandedLegacyProjectReadAdminAliases verifies the remaining legacy read/admin aliases still execute when enabled.
func TestHandlerExpandedLegacyProjectReadAdminAliases(t *testing.T) {
	t.Parallel()

	service := &stubExpandedService{
		stubCaptureStateReader: stubCaptureStateReader{
			captureState: common.CaptureState{StateHash: "abc123"},
		},
		stubMutationAuthorizer: stubMutationAuthorizer{},
	}
	handler, err := NewHandler(Config{ExposeLegacyProjectTools: true}, service, nil)
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}

	server := httptest.NewServer(handler)
	defer server.Close()
	_, _ = postJSONRPC(t, server.Client(), server.URL, initializeRequest())

	callCases := []struct {
		name string
		tool string
		args map[string]any
	}{
		{
			name: "list_projects",
			tool: "till.list_projects",
			args: map[string]any{"include_archived": true},
		},
		{
			name: "list_kind_definitions",
			tool: "till.list_kind_definitions",
			args: map[string]any{"include_archived": true},
		},
		{
			name: "upsert_kind_definition",
			tool: "till.upsert_kind_definition",
			args: mergeArgs(validSessionArgs(), map[string]any{
				"id":         "go-project",
				"applies_to": []any{"project"},
			}),
		},
		{
			name: "get_template_library",
			tool: "till.get_template_library",
			args: map[string]any{"library_id": "go-defaults"},
		},
		{
			name: "list_template_libraries",
			tool: "till.list_template_libraries",
			args: map[string]any{"scope": "global", "status": "approved"},
		},
		{
			name: "upsert_template_library",
			tool: "till.upsert_template_library",
			args: mergeArgs(validSessionArgs(), map[string]any{
				"library": map[string]any{
					"id":     "go-defaults",
					"scope":  "global",
					"name":   "Go Defaults",
					"status": "approved",
				},
			}),
		},
	}

	for idx, tc := range callCases {
		_, callResp := postJSONRPC(t, server.Client(), server.URL, callToolRequest(1500+idx, tc.tool, tc.args))
		if isError, _ := callResp.Result["isError"].(bool); isError {
			t.Fatalf("%s returned isError=true: %#v", tc.name, callResp.Result)
		}
	}

	if !service.lastListProjectsArchived {
		t.Fatal("legacy list_projects include_archived = false, want true")
	}
	if !service.lastListKindsArchived {
		t.Fatal("legacy list_kind_definitions include_archived = false, want true")
	}
	if got := service.lastUpsertKindReq.ID; got != "go-project" {
		t.Fatalf("legacy upsert_kind_definition id = %q, want go-project", got)
	}
	if got := service.lastSetAllowedKindsReq.ProjectID; got != "" {
		t.Fatalf("legacy set_project_allowed_kinds unexpectedly ran, got project_id %q", got)
	}
	if got := service.lastListTemplateReq.Scope; got != domain.TemplateLibraryScopeGlobal {
		t.Fatalf("legacy list_template_libraries scope = %q, want global", got)
	}
	if got := service.lastListTemplateReq.Status; got != domain.TemplateLibraryStatusApproved {
		t.Fatalf("legacy list_template_libraries status = %q, want approved", got)
	}
	if got := service.lastGetTemplateID; got != "go-defaults" {
		t.Fatalf("legacy get_template_library library_id = %q, want go-defaults", got)
	}
	if got := service.lastUpsertTemplateReq.ID; got != "go-defaults" {
		t.Fatalf("legacy upsert_template_library id = %q, want go-defaults", got)
	}
}

// TestHandlerExpandedPlanItemReadOperations verifies default plan-item reads route through get/list shapes.
func TestHandlerExpandedPlanItemReadOperations(t *testing.T) {
	t.Parallel()

	service := &stubExpandedService{
		stubCaptureStateReader: stubCaptureStateReader{
			captureState: common.CaptureState{StateHash: "abc123"},
		},
	}
	handler, err := NewHandler(Config{}, service, nil)
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}

	server := httptest.NewServer(handler)
	defer server.Close()
	_, _ = postJSONRPC(t, server.Client(), server.URL, initializeRequest())

	_, getResp := postJSONRPC(t, server.Client(), server.URL, callToolRequest(4800, "till.plan_item", map[string]any{
		"operation": "get",
		"task_id":   "t1",
	}))
	if isError, _ := getResp.Result["isError"].(bool); isError {
		t.Fatalf("plan_item get returned isError=true: %#v", getResp.Result)
	}
	if got := service.lastGetTaskID; got != "t1" {
		t.Fatalf("plan_item get task_id = %q, want t1", got)
	}

	_, listResp := postJSONRPC(t, server.Client(), server.URL, callToolRequest(4801, "till.plan_item", map[string]any{
		"operation":        "list",
		"project_id":       "p1",
		"include_archived": true,
	}))
	if isError, _ := listResp.Result["isError"].(bool); isError {
		t.Fatalf("plan_item list returned isError=true: %#v", listResp.Result)
	}
	if got := service.lastListTasksProjectID; got != "p1" {
		t.Fatalf("plan_item list project_id = %q, want p1", got)
	}
	if !service.lastListTasksArchived {
		t.Fatal("plan_item list include_archived = false, want true")
	}

	_, childListResp := postJSONRPC(t, server.Client(), server.URL, callToolRequest(4802, "till.plan_item", map[string]any{
		"operation":  "list",
		"project_id": "p1",
		"parent_id":  "parent-1",
	}))
	if isError, _ := childListResp.Result["isError"].(bool); isError {
		t.Fatalf("plan_item child list returned isError=true: %#v", childListResp.Result)
	}
	if got := service.lastListChildProjectID; got != "p1" {
		t.Fatalf("plan_item child list project_id = %q, want p1", got)
	}
	if got := service.lastListChildParentID; got != "parent-1" {
		t.Fatalf("plan_item child list parent_id = %q, want parent-1", got)
	}
}

// TestHandlerInstructionsToolReturnsEmbeddedDocs verifies till.get_instructions returns embedded markdown inventory and guidance.
func TestHandlerInstructionsToolReturnsEmbeddedDocs(t *testing.T) {
	t.Parallel()

	service := &stubExpandedService{
		stubCaptureStateReader: stubCaptureStateReader{
			captureState: common.CaptureState{StateHash: "abc123"},
		},
	}
	handler, err := NewHandler(Config{}, service, nil)
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}

	server := httptest.NewServer(handler)
	defer server.Close()

	resp, callResp := postJSONRPC(t, server.Client(), server.URL, callToolRequest(
		500,
		"till.get_instructions",
		map[string]any{
			"doc_names":               []any{"README.md", "AGENTS.md"},
			"include_markdown":        false,
			"include_recommendations": true,
		},
	))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	if isError, _ := callResp.Result["isError"].(bool); isError {
		t.Fatalf("tool returned isError=true: %#v", callResp.Result)
	}
	structured := toolResultStructured(t, callResp.Result)
	availableAny, ok := structured["available_docs"].([]any)
	if !ok || len(availableAny) == 0 {
		t.Fatalf("available_docs missing/empty: %#v", structured)
	}
	available := make([]string, 0, len(availableAny))
	for _, raw := range availableAny {
		value, _ := raw.(string)
		if strings.TrimSpace(value) == "" {
			continue
		}
		available = append(available, value)
	}
	if !slices.Contains(available, "README.md") {
		t.Fatalf("available docs missing README.md: %#v", available)
	}
	if !slices.Contains(available, "AGENTS.md") {
		t.Fatalf("available docs missing AGENTS.md: %#v", available)
	}
	mdGuidance, ok := structured["md_file_guidance"].(map[string]any)
	if !ok {
		t.Fatalf("md_file_guidance missing: %#v", structured)
	}
	if _, ok := mdGuidance["AGENTS.md"]; !ok {
		t.Fatalf("md_file_guidance missing AGENTS.md guidance: %#v", mdGuidance)
	}
}

// TestHandlerInstructionsToolExplainsProjectScope verifies till.get_instructions can explain project-scoped policy.
func TestHandlerInstructionsToolExplainsProjectScope(t *testing.T) {
	t.Parallel()

	service := &stubExpandedService{
		stubCaptureStateReader: stubCaptureStateReader{
			captureState: common.CaptureState{StateHash: "abc123"},
		},
	}
	handler, err := NewHandler(Config{}, service, nil)
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}

	server := httptest.NewServer(handler)
	defer server.Close()

	resp, callResp := postJSONRPC(t, server.Client(), server.URL, callToolRequest(
		501,
		"till.get_instructions",
		map[string]any{
			"focus":            "project",
			"project_id":       "p1",
			"include_evidence": true,
		},
	))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	if isError, _ := callResp.Result["isError"].(bool); isError {
		t.Fatalf("tool returned isError=true: %#v", callResp.Result)
	}
	structured := toolResultStructured(t, callResp.Result)
	if got, _ := structured["focus"].(string); got != "project" {
		t.Fatalf("focus = %q, want project", got)
	}
	explanation, ok := structured["explanation"].(map[string]any)
	if !ok {
		t.Fatalf("explanation missing: %#v", structured)
	}
	scopedRules, ok := explanation["scoped_rules"].([]any)
	if !ok || len(scopedRules) == 0 {
		t.Fatalf("scoped_rules missing: %#v", explanation)
	}
	rulesText := strings.ToLower(joinAnyStrings(scopedRules))
	if !strings.Contains(rulesText, "standards_markdown") {
		t.Fatalf("scoped_rules = %q, want standards_markdown guidance", rulesText)
	}
	if !strings.Contains(rulesText, "template library") {
		t.Fatalf("scoped_rules = %q, want template binding guidance", rulesText)
	}
}

// TestHandlerInstructionsToolExplainsNodeScope verifies till.get_instructions can explain node-local workflow rules.
func TestHandlerInstructionsToolExplainsNodeScope(t *testing.T) {
	t.Parallel()

	service := &stubExpandedService{
		stubCaptureStateReader: stubCaptureStateReader{
			captureState: common.CaptureState{StateHash: "abc123"},
		},
	}
	handler, err := NewHandler(Config{}, service, nil)
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}

	server := httptest.NewServer(handler)
	defer server.Close()

	resp, callResp := postJSONRPC(t, server.Client(), server.URL, callToolRequest(
		502,
		"till.get_instructions",
		map[string]any{
			"focus":            "node",
			"node_id":          "task-1",
			"include_evidence": true,
		},
	))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	if isError, _ := callResp.Result["isError"].(bool); isError {
		t.Fatalf("tool returned isError=true: %#v", callResp.Result)
	}
	structured := toolResultStructured(t, callResp.Result)
	resolved, ok := structured["resolved_scope"].(map[string]any)
	if !ok {
		t.Fatalf("resolved_scope missing: %#v", structured)
	}
	if got, _ := resolved["node_id"].(string); got != "task-1" {
		t.Fatalf("node_id = %q, want task-1", got)
	}
	explanation, ok := structured["explanation"].(map[string]any)
	if !ok {
		t.Fatalf("explanation missing: %#v", structured)
	}
	workflow, ok := explanation["workflow_contract"].([]any)
	if !ok || len(workflow) == 0 {
		t.Fatalf("workflow_contract missing: %#v", explanation)
	}
	workflowText := strings.ToLower(joinAnyStrings(workflow))
	if !strings.Contains(workflowText, "responsible actor kind") {
		t.Fatalf("workflow_contract = %q, want responsible actor guidance", workflowText)
	}
	scopedRules, ok := explanation["scoped_rules"].([]any)
	if !ok || len(scopedRules) == 0 {
		t.Fatalf("scoped_rules missing: %#v", explanation)
	}
	rulesText := strings.ToLower(joinAnyStrings(scopedRules))
	if !strings.Contains(rulesText, "validation plan") {
		t.Fatalf("scoped_rules = %q, want validation plan guidance", rulesText)
	}
	if !strings.Contains(rulesText, "depends on") {
		t.Fatalf("scoped_rules = %q, want depends_on sequencing guidance", rulesText)
	}
	if !strings.Contains(rulesText, "blocked by") {
		t.Fatalf("scoped_rules = %q, want blocked_by sequencing guidance", rulesText)
	}
	if !strings.Contains(workflowText, "task-level sequencing") {
		t.Fatalf("workflow_contract = %q, want task sequencing guidance", workflowText)
	}
}

// TestHandlerInstructionsToolExplainsBootstrapTopic verifies bootstrap guidance now lives in till.get_instructions.
func TestHandlerInstructionsToolExplainsBootstrapTopic(t *testing.T) {
	t.Parallel()

	service := &stubExpandedService{
		stubCaptureStateReader: stubCaptureStateReader{
			captureState: common.CaptureState{StateHash: "abc123"},
		},
	}
	handler, err := NewHandler(Config{}, service, nil)
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}

	server := httptest.NewServer(handler)
	defer server.Close()

	resp, callResp := postJSONRPC(t, server.Client(), server.URL, callToolRequest(
		503,
		"till.get_instructions",
		map[string]any{
			"mode":  "explain",
			"focus": "topic",
			"topic": "bootstrap",
		},
	))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	if isError, _ := callResp.Result["isError"].(bool); isError {
		t.Fatalf("tool returned isError=true: %#v", callResp.Result)
	}
	structured := toolResultStructured(t, callResp.Result)
	if got, _ := structured["summary"].(string); !strings.Contains(strings.ToLower(got), "auth request") {
		t.Fatalf("summary = %q, want bootstrap auth guidance", got)
	}
	explanation, ok := structured["explanation"].(map[string]any)
	if !ok {
		t.Fatalf("explanation missing: %#v", structured)
	}
	workflow, ok := explanation["workflow_contract"].([]any)
	if !ok || len(workflow) == 0 {
		t.Fatalf("workflow_contract missing: %#v", explanation)
	}
	workflowText := strings.ToLower(joinAnyStrings(workflow))
	if !strings.Contains(workflowText, "till.project(operation=create)") {
		t.Fatalf("workflow_contract = %q, want bootstrap project-create guidance", workflowText)
	}
}

// TestHandlerExpandedCommentToolSchema verifies summary/details markdown guidance in comment tool args.
func TestHandlerExpandedCommentToolSchema(t *testing.T) {
	service := &stubExpandedService{
		stubCaptureStateReader: stubCaptureStateReader{
			captureState: common.CaptureState{StateHash: "abc123"},
		},
	}
	handler, err := NewHandler(Config{}, service, nil)
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}
	server := httptest.NewServer(handler)
	defer server.Close()
	_, _ = postJSONRPC(t, server.Client(), server.URL, initializeRequest())
	_, toolsResp := postJSONRPC(t, server.Client(), server.URL, map[string]any{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "tools/list",
	})
	toolsRaw, ok := toolsResp.Result["tools"].([]any)
	if !ok {
		t.Fatalf("tools list payload missing tools: %#v", toolsResp.Result)
	}
	createSchema := findToolSchemaByName(t, toolsRaw, "till.comment")
	requiredRaw, _ := createSchema["required"].([]any)
	required := make([]string, 0, len(requiredRaw))
	for _, item := range requiredRaw {
		if value, ok := item.(string); ok {
			required = append(required, value)
		}
	}
	if !slices.Contains(required, "operation") {
		t.Fatalf("comment required args missing operation: %#v", required)
	}
	summaryDesc := schemaStringPropertyDescription(t, createSchema, "summary")
	if !strings.Contains(strings.ToLower(summaryDesc), "markdown-rich") {
		t.Fatalf("summary description = %q, want markdown-rich guidance", summaryDesc)
	}
	bodyDesc := schemaStringPropertyDescription(t, createSchema, "body_markdown")
	if !strings.Contains(strings.ToLower(bodyDesc), "markdown-rich") {
		t.Fatalf("body_markdown description = %q, want markdown-rich guidance", bodyDesc)
	}
	authContextDesc := schemaStringPropertyDescription(t, createSchema, "auth_context_id")
	if !strings.Contains(strings.ToLower(authContextDesc), "auth context") {
		t.Fatalf("auth_context_id description = %q, want auth-context guidance", authContextDesc)
	}
}

// TestHandlerExpandedSearchToolSchemaOptions verifies search mode/sort/pagination tool schema guidance.
func TestHandlerExpandedSearchToolSchemaOptions(t *testing.T) {
	service := &stubExpandedService{
		stubCaptureStateReader: stubCaptureStateReader{
			captureState: common.CaptureState{StateHash: "abc123"},
		},
	}
	handler, err := NewHandler(Config{}, service, nil)
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}

	server := httptest.NewServer(handler)
	defer server.Close()
	_, _ = postJSONRPC(t, server.Client(), server.URL, initializeRequest())
	_, toolsResp := postJSONRPC(t, server.Client(), server.URL, map[string]any{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "tools/list",
	})
	toolsRaw, ok := toolsResp.Result["tools"].([]any)
	if !ok {
		t.Fatalf("tools list payload missing tools: %#v", toolsResp.Result)
	}

	searchSchema := findToolSchemaByName(t, toolsRaw, "till.plan_item")
	modeDesc := schemaStringPropertyDescription(t, searchSchema, "search_mode")
	if !strings.Contains(modeDesc, "default hybrid") {
		t.Fatalf("mode description = %q, want default hybrid guidance", modeDesc)
	}
	if !strings.Contains(modeDesc, "fall back to keyword") {
		t.Fatalf("mode description = %q, want keyword fallback guidance", modeDesc)
	}
	modeEnum := schemaPropertyEnumStrings(t, searchSchema, "search_mode")
	for _, want := range []string{"keyword", "semantic", "hybrid"} {
		if !slices.Contains(modeEnum, want) {
			t.Fatalf("mode enum missing %q: %#v", want, modeEnum)
		}
	}
	levelsDesc := schemaStringPropertyDescription(t, searchSchema, "levels")
	if !strings.Contains(strings.ToLower(levelsDesc), "level") {
		t.Fatalf("levels description = %q, want level filter guidance", levelsDesc)
	}
	kindsDesc := schemaStringPropertyDescription(t, searchSchema, "kinds")
	if !strings.Contains(strings.ToLower(kindsDesc), "kind") {
		t.Fatalf("kinds description = %q, want kind filter guidance", kindsDesc)
	}
	labelsAnyDesc := schemaStringPropertyDescription(t, searchSchema, "labels_any")
	if !strings.Contains(strings.ToLower(labelsAnyDesc), "any") {
		t.Fatalf("labels_any description = %q, want labels-any guidance", labelsAnyDesc)
	}
	labelsAllDesc := schemaStringPropertyDescription(t, searchSchema, "labels_all")
	if !strings.Contains(strings.ToLower(labelsAllDesc), "all") {
		t.Fatalf("labels_all description = %q, want labels-all guidance", labelsAllDesc)
	}

	sortDesc := schemaStringPropertyDescription(t, searchSchema, "sort")
	if !strings.Contains(sortDesc, "rank_desc") || !strings.Contains(sortDesc, "default rank_desc") {
		t.Fatalf("sort description = %q, want rank_desc default guidance", sortDesc)
	}
	sortEnum := schemaPropertyEnumStrings(t, searchSchema, "sort")
	for _, want := range []string{"rank_desc", "title_asc", "created_at_desc", "updated_at_desc"} {
		if !slices.Contains(sortEnum, want) {
			t.Fatalf("sort enum missing %q: %#v", want, sortEnum)
		}
	}

	limitDesc := schemaStringPropertyDescription(t, searchSchema, "limit")
	if !strings.Contains(limitDesc, "default 50") || !strings.Contains(limitDesc, "max 200") {
		t.Fatalf("limit description = %q, want default/max guidance", limitDesc)
	}
	if got := schemaPropertyNumberField(t, searchSchema, "limit", "minimum"); got != 0 {
		t.Fatalf("limit minimum = %v, want 0", got)
	}
	if got := schemaPropertyNumberField(t, searchSchema, "limit", "maximum"); got != 200 {
		t.Fatalf("limit maximum = %v, want 200", got)
	}
	if got := schemaPropertyNumberField(t, searchSchema, "limit", "default"); got != 50 {
		t.Fatalf("limit default = %v, want 50", got)
	}
	offsetDesc := schemaStringPropertyDescription(t, searchSchema, "offset")
	if !strings.Contains(offsetDesc, "default 0") {
		t.Fatalf("offset description = %q, want default guidance", offsetDesc)
	}
	if got := schemaPropertyNumberField(t, searchSchema, "offset", "minimum"); got != 0 {
		t.Fatalf("offset minimum = %v, want 0", got)
	}
	if got := schemaPropertyNumberField(t, searchSchema, "offset", "default"); got != 0 {
		t.Fatalf("offset default = %v, want 0", got)
	}
}

// TestHandlerExpandedSearchToolForwardsExtendedFilters verifies mode/sort/pagination fields are forwarded.
func TestHandlerExpandedSearchToolForwardsExtendedFilters(t *testing.T) {
	service := &stubExpandedService{
		stubCaptureStateReader: stubCaptureStateReader{
			captureState: common.CaptureState{StateHash: "abc123"},
		},
	}
	handler, err := NewHandler(Config{}, service, nil)
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}

	server := httptest.NewServer(handler)
	defer server.Close()
	_, _ = postJSONRPC(t, server.Client(), server.URL, initializeRequest())

	_, callResp := postJSONRPC(t, server.Client(), server.URL, callToolRequest(610, "till.plan_item", map[string]any{
		"operation":        "search",
		"project_id":       "p1",
		"query":            "task",
		"cross_project":    true,
		"include_archived": true,
		"states":           []any{"todo"},
		"levels":           []any{"phase"},
		"kinds":            []any{"phase"},
		"labels_any":       []any{"backend", "ops"},
		"labels_all":       []any{"urgent"},
		"search_mode":      "hybrid",
		"sort":             "title_asc",
		"limit":            75,
		"offset":           10,
	}))
	if isError, _ := callResp.Result["isError"].(bool); isError {
		t.Fatalf("plan_item search returned isError=true: %#v", callResp.Result)
	}

	if got := service.lastSearchTasksReq.ProjectID; got != "p1" {
		t.Fatalf("project_id = %q, want p1", got)
	}
	if got := service.lastSearchTasksReq.Query; got != "task" {
		t.Fatalf("query = %q, want task", got)
	}
	if !service.lastSearchTasksReq.CrossProject {
		t.Fatalf("cross_project = false, want true")
	}
	if !service.lastSearchTasksReq.IncludeArchived {
		t.Fatalf("include_archived = false, want true")
	}
	if got := service.lastSearchTasksReq.Mode; got != "hybrid" {
		t.Fatalf("mode = %q, want hybrid", got)
	}
	if got := service.lastSearchTasksReq.Sort; got != "title_asc" {
		t.Fatalf("sort = %q, want title_asc", got)
	}
	if got := service.lastSearchTasksReq.Limit; got != 75 {
		t.Fatalf("limit = %d, want 75", got)
	}
	if got := service.lastSearchTasksReq.Offset; got != 10 {
		t.Fatalf("offset = %d, want 10", got)
	}
	if len(service.lastSearchTasksReq.States) != 1 || service.lastSearchTasksReq.States[0] != "todo" {
		t.Fatalf("states = %#v, want [todo]", service.lastSearchTasksReq.States)
	}
	if got := service.lastSearchTasksReq.Levels; !slices.Equal(got, []string{"phase"}) {
		t.Fatalf("levels = %#v, want [phase]", got)
	}
	if got := service.lastSearchTasksReq.Kinds; !slices.Equal(got, []string{"phase"}) {
		t.Fatalf("kinds = %#v, want [phase]", got)
	}
	if got := service.lastSearchTasksReq.LabelsAny; !slices.Equal(got, []string{"backend", "ops"}) {
		t.Fatalf("labels_any = %#v, want [backend ops]", got)
	}
	if got := service.lastSearchTasksReq.LabelsAll; !slices.Equal(got, []string{"urgent"}) {
		t.Fatalf("labels_all = %#v, want [urgent]", got)
	}

	_, defaultResp := postJSONRPC(t, server.Client(), server.URL, callToolRequest(611, "till.plan_item", map[string]any{
		"operation":  "search",
		"project_id": "p1",
	}))
	if isError, _ := defaultResp.Result["isError"].(bool); isError {
		t.Fatalf("default plan_item search returned isError=true: %#v", defaultResp.Result)
	}
	if got := service.lastSearchTasksReq.Mode; got != "" {
		t.Fatalf("default mode = %q, want empty for app-defaulting", got)
	}
	if got := service.lastSearchTasksReq.Sort; got != "" {
		t.Fatalf("default sort = %q, want empty for app-defaulting", got)
	}
	if got := service.lastSearchTasksReq.Limit; got != 0 {
		t.Fatalf("default limit = %d, want 0 for app-defaulting", got)
	}
	if got := service.lastSearchTasksReq.Offset; got != 0 {
		t.Fatalf("default offset = %d, want 0", got)
	}
	if len(service.lastSearchTasksReq.Levels) != 0 {
		t.Fatalf("default levels = %#v, want empty", service.lastSearchTasksReq.Levels)
	}
	if len(service.lastSearchTasksReq.Kinds) != 0 {
		t.Fatalf("default kinds = %#v, want empty", service.lastSearchTasksReq.Kinds)
	}
	if len(service.lastSearchTasksReq.LabelsAny) != 0 {
		t.Fatalf("default labels_any = %#v, want empty", service.lastSearchTasksReq.LabelsAny)
	}
	if len(service.lastSearchTasksReq.LabelsAll) != 0 {
		t.Fatalf("default labels_all = %#v, want empty", service.lastSearchTasksReq.LabelsAll)
	}
}

// TestHandlerExpandedEmbeddingsToolsExposeMixedSubjectMetadata verifies the embeddings tools surface mixed subject families and per-match subject metadata.
func TestHandlerExpandedEmbeddingsToolsExposeMixedSubjectMetadata(t *testing.T) {
	t.Parallel()

	service := &stubExpandedService{
		stubCaptureStateReader: stubCaptureStateReader{
			captureState: common.CaptureState{StateHash: "abc123"},
		},
	}
	handler, err := NewHandler(Config{}, service, nil)
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}

	server := httptest.NewServer(handler)
	defer server.Close()
	_, _ = postJSONRPC(t, server.Client(), server.URL, initializeRequest())

	_, searchResp := postJSONRPC(t, server.Client(), server.URL, callToolRequest(612, "till.plan_item", map[string]any{
		"operation":   "search",
		"project_id":  "p1",
		"query":       "task",
		"search_mode": "semantic",
	}))
	if isError, _ := searchResp.Result["isError"].(bool); isError {
		t.Fatalf("plan_item search returned isError=true: %#v", searchResp.Result)
	}
	searchStructured := toolResultStructured(t, searchResp.Result)
	matchesAny, ok := searchStructured["matches"].([]any)
	if !ok || len(matchesAny) == 0 {
		t.Fatalf("search matches missing/empty: %#v", searchStructured)
	}
	firstMatch, ok := matchesAny[0].(map[string]any)
	if !ok {
		t.Fatalf("search match has unexpected type: %#v", matchesAny[0])
	}
	if got := firstMatch["embedding_subject_type"]; got != "thread_context" {
		t.Fatalf("embedding_subject_type = %#v, want thread_context", got)
	}
	if got := firstMatch["embedding_subject_id"]; got != "comment-1" {
		t.Fatalf("embedding_subject_id = %#v, want comment-1", got)
	}
	if got := firstMatch["embedding_status"]; got != "ready" {
		t.Fatalf("embedding_status = %#v, want ready", got)
	}

	_, statusResp := postJSONRPC(t, server.Client(), server.URL, callToolRequest(613, "till.embeddings", map[string]any{
		"operation":  "status",
		"project_id": "p1",
		"limit":      10,
	}))
	if isError, _ := statusResp.Result["isError"].(bool); isError {
		t.Fatalf("get_embeddings_status returned isError=true: %#v", statusResp.Result)
	}
	statusStructured := toolResultStructured(t, statusResp.Result)
	rowsAny, ok := statusStructured["rows"].([]any)
	if !ok || len(rowsAny) < 3 {
		t.Fatalf("status rows missing/too short: %#v", statusStructured)
	}
	types := map[string]bool{}
	for _, raw := range rowsAny {
		row, ok := raw.(map[string]any)
		if !ok {
			t.Fatalf("status row has unexpected type: %#v", raw)
		}
		if subjectType, _ := row["subject_type"].(string); subjectType != "" {
			types[subjectType] = true
		}
	}
	for _, want := range []string{"work_item", "thread_context", "project_document"} {
		if !types[want] {
			t.Fatalf("status rows missing subject type %q: %#v", want, rowsAny)
		}
	}
}

// TestHandlerExpandedRecoveryToolsForwardScopeFilters verifies lease/handoff discovery tools forward scope and status filters.
func TestHandlerExpandedRecoveryToolsForwardScopeFilters(t *testing.T) {
	service := &stubExpandedService{
		stubCaptureStateReader: stubCaptureStateReader{
			captureState: common.CaptureState{StateHash: "abc123"},
		},
	}
	handler, err := NewHandler(Config{}, service, nil)
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}

	server := httptest.NewServer(handler)
	defer server.Close()
	_, _ = postJSONRPC(t, server.Client(), server.URL, initializeRequest())

	_, leaseResp := postJSONRPC(t, server.Client(), server.URL, callToolRequest(620, "till.capability_lease", map[string]any{
		"operation":       "list",
		"project_id":      "p1",
		"scope_type":      "task",
		"scope_id":        "task-1",
		"include_revoked": true,
	}))
	if isError, _ := leaseResp.Result["isError"].(bool); isError {
		t.Fatalf("list_capability_leases returned isError=true: %#v", leaseResp.Result)
	}
	if got := service.lastListLeaseReq.ProjectID; got != "p1" {
		t.Fatalf("list_capability_leases project_id = %q, want p1", got)
	}
	if got := service.lastListLeaseReq.ScopeType; got != "task" {
		t.Fatalf("list_capability_leases scope_type = %q, want task", got)
	}
	if got := service.lastListLeaseReq.ScopeID; got != "task-1" {
		t.Fatalf("list_capability_leases scope_id = %q, want task-1", got)
	}
	if !service.lastListLeaseReq.IncludeRevoked {
		t.Fatal("list_capability_leases include_revoked = false, want true")
	}

	_, getResp := postJSONRPC(t, server.Client(), server.URL, callToolRequest(621, "till.handoff", map[string]any{
		"operation":  "get",
		"handoff_id": "handoff-1",
	}))
	if isError, _ := getResp.Result["isError"].(bool); isError {
		t.Fatalf("get_handoff returned isError=true: %#v", getResp.Result)
	}
	if got := service.lastGetHandoffID; got != "handoff-1" {
		t.Fatalf("get_handoff handoff_id = %q, want handoff-1", got)
	}

	_, listResp := postJSONRPC(t, server.Client(), server.URL, callToolRequest(622, "till.handoff", map[string]any{
		"operation":    "list",
		"project_id":   "p1",
		"branch_id":    "branch-1",
		"scope_type":   "task",
		"scope_id":     "task-1",
		"statuses":     []any{"waiting", "blocked"},
		"limit":        25,
		"wait_timeout": "30s",
	}))
	if isError, _ := listResp.Result["isError"].(bool); isError {
		t.Fatalf("list_handoffs returned isError=true: %#v", listResp.Result)
	}
	if got := service.lastListHandoffsReq.ProjectID; got != "p1" {
		t.Fatalf("list_handoffs project_id = %q, want p1", got)
	}
	if got := service.lastListHandoffsReq.BranchID; got != "branch-1" {
		t.Fatalf("list_handoffs branch_id = %q, want branch-1", got)
	}
	if got := service.lastListHandoffsReq.ScopeType; got != "task" {
		t.Fatalf("list_handoffs scope_type = %q, want task", got)
	}
	if got := service.lastListHandoffsReq.ScopeID; got != "task-1" {
		t.Fatalf("list_handoffs scope_id = %q, want task-1", got)
	}
	if got := service.lastListHandoffsReq.Statuses; !slices.Equal(got, []string{"waiting", "blocked"}) {
		t.Fatalf("list_handoffs statuses = %#v, want [waiting blocked]", got)
	}
	if got := service.lastListHandoffsReq.Limit; got != 25 {
		t.Fatalf("list_handoffs limit = %d, want 25", got)
	}
	if got := service.lastListHandoffsReq.WaitTimeout; got != "30s" {
		t.Fatalf("list_handoffs wait_timeout = %q, want 30s", got)
	}
}

// TestHandlerExpandedToolBuildsActorTupleFromAuthenticatedSession verifies mutation identity comes from auth, not caller-supplied actor fields.
func TestHandlerExpandedToolBuildsActorTupleFromAuthenticatedSession(t *testing.T) {
	service := &stubExpandedService{
		stubCaptureStateReader: stubCaptureStateReader{
			captureState: common.CaptureState{StateHash: "abc123"},
		},
		stubMutationAuthorizer: stubMutationAuthorizer{
			authCaller: domain.AuthenticatedCaller{
				PrincipalID:   "agent-session-1",
				PrincipalName: "Agent Session One",
				PrincipalType: domain.ActorTypeAgent,
				SessionID:     "sess-1",
			},
		},
	}
	handler, err := NewHandler(Config{}, service, nil)
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}

	server := httptest.NewServer(handler)
	defer server.Close()
	_, _ = postJSONRPC(t, server.Client(), server.URL, initializeRequest())

	_, createResp := postJSONRPC(t, server.Client(), server.URL, callToolRequest(300, "till.plan_item", mergeArgs(validSessionArgs(), map[string]any{
		"operation":         "create",
		"project_id":        "p1",
		"column_id":         "c1",
		"title":             "Task One",
		"agent_instance_id": "inst-1",
		"lease_token":       "lease-1",
	})))
	if isError, _ := createResp.Result["isError"].(bool); isError {
		t.Fatalf("plan_item create returned isError=true: %#v", createResp.Result)
	}
	if got := service.lastCreateTaskReq.Actor.ActorType; got != "agent" {
		t.Fatalf("plan_item create actor_type = %q, want agent", got)
	}
	if got := service.lastCreateTaskReq.Actor.ActorID; got != "agent-session-1" {
		t.Fatalf("plan_item create actor_id = %q, want agent-session-1", got)
	}
	if got := service.lastCreateTaskReq.Actor.ActorName; got != "Agent Session One" {
		t.Fatalf("plan_item create actor_name = %q, want Agent Session One", got)
	}

	_, updateResp := postJSONRPC(t, server.Client(), server.URL, callToolRequest(301, "till.plan_item", mergeArgs(validSessionArgs(), map[string]any{
		"operation":         "update",
		"task_id":           "t1",
		"title":             "Task One Updated",
		"agent_instance_id": "inst-1",
		"lease_token":       "lease-1",
	})))
	if isError, _ := updateResp.Result["isError"].(bool); isError {
		t.Fatalf("plan_item update returned isError=true: %#v", updateResp.Result)
	}
	if got := service.lastUpdateTaskReq.Actor.ActorType; got != "agent" {
		t.Fatalf("plan_item update actor_type = %q, want agent", got)
	}
	if got := service.lastUpdateTaskReq.Actor.AgentName; got != "agent-session-1" {
		t.Fatalf("plan_item update agent_name = %q, want agent-session-1", got)
	}
	if got := service.lastUpdateTaskReq.Actor.ActorID; got != "agent-session-1" {
		t.Fatalf("plan_item update actor_id = %q, want agent-session-1", got)
	}
	if got := service.lastUpdateTaskReq.Actor.ActorName; got != "Agent Session One" {
		t.Fatalf("plan_item update actor_name = %q, want Agent Session One", got)
	}

	_, moveStateResp := postJSONRPC(t, server.Client(), server.URL, callToolRequest(3010, "till.plan_item", mergeArgs(validSessionArgs(), map[string]any{
		"operation":         "move_state",
		"task_id":           "t1",
		"state":             "done",
		"agent_instance_id": "inst-1",
		"lease_token":       "lease-1",
	})))
	if isError, _ := moveStateResp.Result["isError"].(bool); isError {
		t.Fatalf("plan_item move_state returned isError=true: %#v", moveStateResp.Result)
	}
	if got := service.lastMoveTaskStateReq.Actor.ActorType; got != "agent" {
		t.Fatalf("plan_item move_state actor_type = %q, want agent", got)
	}
	if got := service.lastMoveTaskStateReq.Actor.ActorID; got != "agent-session-1" {
		t.Fatalf("plan_item move_state actor_id = %q, want agent-session-1", got)
	}
	if got := service.lastMoveTaskStateReq.State; got != "done" {
		t.Fatalf("plan_item move_state state = %q, want done", got)
	}

	_, commentResp := postJSONRPC(t, server.Client(), server.URL, callToolRequest(3011, "till.comment", mergeArgs(validSessionArgs(), map[string]any{
		"operation":         "create",
		"project_id":        "p1",
		"target_type":       "task",
		"target_id":         "t1",
		"summary":           "Thread summary",
		"body_markdown":     "hello",
		"agent_instance_id": "inst-comment",
		"lease_token":       "lease-comment",
	})))
	if isError, _ := commentResp.Result["isError"].(bool); isError {
		t.Fatalf("comment create returned isError=true: %#v", commentResp.Result)
	}
	if got := service.lastCreateCommentReq.Actor.ActorType; got != "agent" {
		t.Fatalf("comment create actor_type = %q, want agent", got)
	}
	if got := service.lastCreateCommentReq.Actor.ActorID; got != "agent-session-1" {
		t.Fatalf("comment create actor_id = %q, want agent-session-1", got)
	}
	if got := service.lastCreateCommentReq.Actor.ActorName; got != "Agent Session One" {
		t.Fatalf("comment create actor_name = %q, want Agent Session One", got)
	}
	if got := service.lastCreateCommentReq.Summary; got != "Thread summary" {
		t.Fatalf("comment create summary = %q, want Thread summary", got)
	}

	_, createHandoffResp := postJSONRPC(t, server.Client(), server.URL, callToolRequest(3012, "till.handoff", mergeArgs(validSessionArgs(), map[string]any{
		"operation":         "create",
		"project_id":        "p1",
		"branch_id":         "branch-1",
		"scope_type":        "task",
		"scope_id":          "task-1",
		"source_role":       "builder",
		"target_branch_id":  "branch-1",
		"target_scope_type": "task",
		"target_scope_id":   "task-qa-1",
		"target_role":       "qa",
		"summary":           "handoff summary",
		"next_action":       "run qa",
		"agent_instance_id": "inst-handoff",
		"lease_token":       "lease-handoff",
	})))
	if isError, _ := createHandoffResp.Result["isError"].(bool); isError {
		t.Fatalf("handoff create returned isError=true: %#v", createHandoffResp.Result)
	}
	if got := service.lastCreateHandoffReq.Actor.ActorType; got != "agent" {
		t.Fatalf("handoff create actor_type = %q, want agent", got)
	}
	if got := service.lastCreateHandoffReq.Actor.ActorID; got != "agent-session-1" {
		t.Fatalf("handoff create actor_id = %q, want agent-session-1", got)
	}
	if got := service.lastCreateHandoffReq.Actor.ActorName; got != "Agent Session One" {
		t.Fatalf("handoff create actor_name = %q, want Agent Session One", got)
	}

	_, updateHandoffResp := postJSONRPC(t, server.Client(), server.URL, callToolRequest(3013, "till.handoff", mergeArgs(validSessionArgs(), map[string]any{
		"operation":         "update",
		"handoff_id":        "handoff-1",
		"status":            "resolved",
		"source_role":       "builder",
		"target_branch_id":  "branch-1",
		"target_scope_type": "task",
		"target_scope_id":   "task-qa-1",
		"target_role":       "qa",
		"summary":           "handoff summary",
		"next_action":       "none",
		"resolution_note":   "qa passed",
		"agent_instance_id": "inst-handoff",
		"lease_token":       "lease-handoff",
	})))
	if isError, _ := updateHandoffResp.Result["isError"].(bool); isError {
		t.Fatalf("handoff update returned isError=true: %#v", updateHandoffResp.Result)
	}
	if got := service.lastUpdateHandoffReq.Actor.ActorType; got != "agent" {
		t.Fatalf("handoff update actor_type = %q, want agent", got)
	}
	if got := service.lastUpdateHandoffReq.Actor.ActorID; got != "agent-session-1" {
		t.Fatalf("handoff update actor_id = %q, want agent-session-1", got)
	}
	if got := service.lastUpdateHandoffReq.Actor.ActorName; got != "Agent Session One" {
		t.Fatalf("handoff update actor_name = %q, want Agent Session One", got)
	}

	_, restoreResp := postJSONRPC(t, server.Client(), server.URL, callToolRequest(302, "till.plan_item", mergeArgs(validSessionArgs(), map[string]any{
		"operation":         "restore",
		"task_id":           "t1",
		"agent_instance_id": "agent-1",
		"lease_token":       "lease-1",
		"override_token":    "override-1",
	})))
	if isError, _ := restoreResp.Result["isError"].(bool); isError {
		t.Fatalf("plan_item restore returned isError=true: %#v", restoreResp.Result)
	}
	if got := service.lastRestoreTaskReq.Actor.ActorType; got != "agent" {
		t.Fatalf("plan_item restore actor_type = %q, want agent", got)
	}
	if got := service.lastRestoreTaskReq.Actor.AgentName; got != "agent-session-1" {
		t.Fatalf("plan_item restore agent_name = %q, want agent-session-1", got)
	}
	if got := service.lastRestoreTaskReq.Actor.AgentInstanceID; got != "agent-1" {
		t.Fatalf("plan_item restore agent_instance_id = %q, want agent-1", got)
	}
	if got := service.lastRestoreTaskReq.Actor.LeaseToken; got != "lease-1" {
		t.Fatalf("plan_item restore lease_token = %q, want lease-1", got)
	}
	if got := service.lastRestoreTaskReq.Actor.OverrideToken; got != "override-1" {
		t.Fatalf("plan_item restore override_token = %q, want override-1", got)
	}

	_, issueLeaseResp := postJSONRPC(t, server.Client(), server.URL, callToolRequest(3014, "till.capability_lease", mergeArgs(validSessionArgs(), map[string]any{
		"operation":   "issue",
		"project_id":  "p1",
		"scope_type":  "project",
		"role":        "orchestrator",
		"agent_name":  "caller-supplied-display-name",
		"scope_id":    "p1",
		"lease_token": "",
	})))
	if isError, _ := issueLeaseResp.Result["isError"].(bool); isError {
		t.Fatalf("capability_lease issue returned isError=true: %#v", issueLeaseResp.Result)
	}
	if got := service.lastIssueLeaseReq.AgentName; got != "agent-session-1" {
		t.Fatalf("capability_lease issue agent_name = %q, want agent-session-1", got)
	}

	_, issueLeaseNoNameResp := postJSONRPC(t, server.Client(), server.URL, callToolRequest(3015, "till.capability_lease", mergeArgs(validSessionArgs(), map[string]any{
		"operation":  "issue",
		"project_id": "p1",
		"scope_type": "project",
		"role":       "builder",
		"scope_id":   "p1",
	})))
	if isError, _ := issueLeaseNoNameResp.Result["isError"].(bool); isError {
		t.Fatalf("capability_lease issue without agent_name returned isError=true: %#v", issueLeaseNoNameResp.Result)
	}
	if got := service.lastIssueLeaseReq.AgentName; got != "agent-session-1" {
		t.Fatalf("capability_lease issue without agent_name agent_name = %q, want agent-session-1", got)
	}
}

// TestHandlerExpandedToolRejectsMissingSessionAndGuardedUserTuples verifies session-first auth failures and tuple validation.
func TestHandlerExpandedToolRejectsMissingSessionAndGuardedUserTuples(t *testing.T) {
	service := &stubExpandedService{
		stubCaptureStateReader: stubCaptureStateReader{
			captureState: common.CaptureState{StateHash: "abc123"},
		},
	}
	handler, err := NewHandler(Config{}, service, nil)
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}

	server := httptest.NewServer(handler)
	defer server.Close()
	_, _ = postJSONRPC(t, server.Client(), server.URL, initializeRequest())

	missingSessionHTTPResp, missingSessionResp := postJSONRPC(t, server.Client(), server.URL, callToolRequest(4010, "till.plan_item", map[string]any{
		"operation":  "create",
		"project_id": "p1",
		"column_id":  "c1",
		"title":      "missing session mutation",
	}))
	if missingSessionHTTPResp.StatusCode == http.StatusOK {
		if isError, _ := missingSessionResp.Result["isError"].(bool); !isError && len(missingSessionResp.Error) == 0 {
			t.Fatalf("missing session call isError = %v, want true", missingSessionResp.Result["isError"])
		}
	}
	if isError, _ := missingSessionResp.Result["isError"].(bool); isError {
		if got := toolResultText(t, missingSessionResp.Result); !strings.Contains(got, "session_required:") {
			t.Fatalf("missing session error = %q, want session_required guidance", got)
		}
	}
	if missingSessionHTTPResp.StatusCode != http.StatusOK && missingSessionHTTPResp.StatusCode != http.StatusBadRequest {
		t.Fatalf("missing session call isError = %v, want true", missingSessionResp.Result["isError"])
	}

	service.authCaller = domain.AuthenticatedCaller{
		PrincipalID:   "user-1",
		PrincipalName: "User One",
		PrincipalType: domain.ActorTypeUser,
		SessionID:     "sess-1",
	}

	guardedUserHTTPResp, guardedUserResp := postJSONRPC(t, server.Client(), server.URL, callToolRequest(4011, "till.plan_item", mergeArgs(validSessionArgs(), map[string]any{
		"operation":         "create",
		"project_id":        "p1",
		"column_id":         "c1",
		"title":             "guarded user mutation",
		"agent_instance_id": "inst-1",
		"lease_token":       "lease-1",
	})))
	if guardedUserHTTPResp.StatusCode == http.StatusOK {
		if isError, _ := guardedUserResp.Result["isError"].(bool); !isError && len(guardedUserResp.Error) == 0 {
			t.Fatalf("guarded user call isError = %v, want true", guardedUserResp.Result["isError"])
		}
	}
	if isError, _ := guardedUserResp.Result["isError"].(bool); isError {
		if got := toolResultText(t, guardedUserResp.Result); !strings.Contains(got, "guarded mutation tuple requires an authenticated agent session") {
			t.Fatalf("guarded user error = %q, want guarded tuple guidance", got)
		}
	}
	if guardedUserHTTPResp.StatusCode != http.StatusOK && guardedUserHTTPResp.StatusCode != http.StatusBadRequest {
		t.Fatalf("guarded user call isError = %v, want true", guardedUserResp.Result["isError"])
	}

	service.authCaller = domain.AuthenticatedCaller{
		PrincipalID:   "agent-1",
		PrincipalName: "Agent One",
		PrincipalType: domain.ActorTypeAgent,
		SessionID:     "sess-1",
	}
	missingLeaseHTTPResp, missingLeaseResp := postJSONRPC(t, server.Client(), server.URL, callToolRequest(4012, "till.plan_item", map[string]any{
		"operation":      "create",
		"project_id":     "p1",
		"column_id":      "c1",
		"title":          "missing lease mutation",
		"session_id":     "sess-1",
		"session_secret": "secret-1",
		"lease_token":    "lease-1",
	}))
	if missingLeaseHTTPResp.StatusCode == http.StatusOK {
		if isError, _ := missingLeaseResp.Result["isError"].(bool); !isError && len(missingLeaseResp.Error) == 0 {
			t.Fatalf("missing lease tuple call isError = %v, want true", missingLeaseResp.Result["isError"])
		}
	}
	if isError, _ := missingLeaseResp.Result["isError"].(bool); isError {
		if got := toolResultText(t, missingLeaseResp.Result); !strings.Contains(got, "agent_name, agent_instance_id, and lease_token are required") {
			t.Fatalf("missing lease tuple error = %q, want lease tuple requirement", got)
		}
	}
	if missingLeaseHTTPResp.StatusCode != http.StatusOK && missingLeaseHTTPResp.StatusCode != http.StatusBadRequest {
		t.Fatalf("missing lease tuple call isError = %v, want true", missingLeaseResp.Result["isError"])
	}
}

// TestHandlerExpandedCommentToolsForwardHierarchyTargetTypes verifies hierarchy node target types pass through comment tools.
func TestHandlerExpandedCommentToolsForwardHierarchyTargetTypes(t *testing.T) {
	service := &stubExpandedService{
		stubCaptureStateReader: stubCaptureStateReader{
			captureState: common.CaptureState{StateHash: "abc123"},
		},
	}
	handler, err := NewHandler(Config{}, service, nil)
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}

	server := httptest.NewServer(handler)
	defer server.Close()
	_, _ = postJSONRPC(t, server.Client(), server.URL, initializeRequest())

	_, createResp := postJSONRPC(t, server.Client(), server.URL, callToolRequest(401, "till.comment", mergeArgs(validSessionArgs(), map[string]any{
		"operation":         "create",
		"project_id":        "p1",
		"target_type":       "branch",
		"target_id":         "branch-1",
		"summary":           "Branch note",
		"body_markdown":     "hello",
		"agent_instance_id": "inst-1",
		"lease_token":       "lease-1",
	})))
	if isError, _ := createResp.Result["isError"].(bool); isError {
		t.Fatalf("comment create returned isError=true: %#v", createResp.Result)
	}
	if got := service.lastCreateCommentReq.TargetType; got != "branch" {
		t.Fatalf("comment create target_type = %q, want branch", got)
	}
	if got := service.lastCreateCommentReq.TargetID; got != "branch-1" {
		t.Fatalf("comment create target_id = %q, want branch-1", got)
	}

	_, listResp := postJSONRPC(t, server.Client(), server.URL, callToolRequest(402, "till.comment", map[string]any{
		"operation":    "list",
		"project_id":   "p1",
		"target_type":  "phase",
		"target_id":    "phase-1",
		"wait_timeout": "15s",
	}))
	if isError, _ := listResp.Result["isError"].(bool); isError {
		t.Fatalf("comment list returned isError=true: %#v", listResp.Result)
	}
	if got := service.lastListCommentReq.TargetType; got != "phase" {
		t.Fatalf("comment list target_type = %q, want phase", got)
	}
	if got := service.lastListCommentReq.TargetID; got != "phase-1" {
		t.Fatalf("comment list target_id = %q, want phase-1", got)
	}
	if got := service.lastListCommentReq.WaitTimeout; got != "15s" {
		t.Fatalf("comment list wait_timeout = %q, want 15s", got)
	}
}

// TestHandlerExpandedMutationFamiliesAcceptAuthContextHandles verifies stdio-style auth handles
// can replace inline session secrets on the reduced mutation families.
func TestHandlerExpandedMutationFamiliesAcceptAuthContextHandles(t *testing.T) {
	t.Parallel()

	service := &stubExpandedService{
		stubCaptureStateReader: stubCaptureStateReader{
			captureState: common.CaptureState{StateHash: "abc123"},
		},
		stubMutationAuthorizer: stubMutationAuthorizer{},
	}
	mcpSrv, cfg, err := NewServer(Config{EnableAuthContexts: true}, service, nil)
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}
	streamable := mcpserver.NewStreamableHTTPServer(
		mcpSrv,
		mcpserver.WithEndpointPath(cfg.EndpointPath),
		mcpserver.WithStateLess(true),
	)
	server := httptest.NewServer(streamable)
	defer server.Close()
	_, _ = postJSONRPC(t, server.Client(), server.URL, initializeRequest())

	_, claimResp := postJSONRPC(t, server.Client(), server.URL, callToolRequest(500, "till.auth_request", map[string]any{
		"operation":    "claim",
		"request_id":   "req-1",
		"resume_token": "resume-123",
		"principal_id": "review-agent",
		"client_id":    "till-mcp-stdio",
	}))
	claimStructured := toolResultStructured(t, claimResp.Result)
	authContextID, _ := claimStructured["auth_context_id"].(string)
	if !strings.HasPrefix(authContextID, "authctx-") {
		t.Fatalf("claim auth_context_id = %q, want authctx-*", authContextID)
	}

	_, createProjectResp := postJSONRPC(t, server.Client(), server.URL, callToolRequest(501, "till.project", map[string]any{
		"operation":       "create",
		"name":            "Project One",
		"session_id":      "sess-1",
		"auth_context_id": authContextID,
	}))
	if isError, _ := createProjectResp.Result["isError"].(bool); isError {
		t.Fatalf("project create with auth_context_id returned isError=true: %#v", createProjectResp.Result)
	}
	if got := service.lastAuthRequest.SessionSecret; got != "secret-1" {
		t.Fatalf("AuthorizeMutation() session_secret = %q, want secret-1", got)
	}

	_, commentResp := postJSONRPC(t, server.Client(), server.URL, callToolRequest(502, "till.comment", map[string]any{
		"operation":         "create",
		"project_id":        "p1",
		"target_type":       "task",
		"target_id":         "t1",
		"summary":           "Thread summary",
		"body_markdown":     "hello",
		"session_id":        "sess-1",
		"auth_context_id":   authContextID,
		"agent_instance_id": "inst-1",
		"lease_token":       "tok-1",
	}))
	if isError, _ := commentResp.Result["isError"].(bool); isError {
		t.Fatalf("comment create with auth_context_id returned isError=true: %#v", commentResp.Result)
	}
	if got := service.lastCreateCommentReq.Summary; got != "Thread summary" {
		t.Fatalf("CreateComment() summary = %q, want Thread summary", got)
	}
	if got := service.lastAuthRequest.SessionSecret; got != "secret-1" {
		t.Fatalf("AuthorizeMutation() reused session_secret = %q, want secret-1", got)
	}
}

// TestHandlerExpandedToolInvalidBindArguments verifies bind failures map to invalid_request errors.
func TestHandlerExpandedToolInvalidBindArguments(t *testing.T) {
	service := &stubExpandedService{
		stubCaptureStateReader: stubCaptureStateReader{
			captureState: common.CaptureState{StateHash: "abc123"},
		},
	}
	handler, err := NewHandler(Config{}, service, nil)
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}

	server := httptest.NewServer(handler)
	defer server.Close()
	_, _ = postJSONRPC(t, server.Client(), server.URL, initializeRequest())

	_, callResp := postJSONRPC(t, server.Client(), server.URL, callToolRequest(201, "till.project", map[string]any{
		"operation": "create",
		"name":      123,
	}))
	if isError, _ := callResp.Result["isError"].(bool); !isError {
		t.Fatalf("isError = %v, want true", callResp.Result["isError"])
	}
	if got := toolResultText(t, callResp.Result); !strings.HasPrefix(got, "invalid_request:") {
		t.Fatalf("error text = %q, want prefix invalid_request:", got)
	}
}

// TestHandlerExpandedCreateProjectPassesTemplateLibraryID verifies the MCP project tool forwards template-library selection.
func TestHandlerExpandedCreateProjectPassesTemplateLibraryID(t *testing.T) {
	service := &stubExpandedService{
		stubCaptureStateReader: stubCaptureStateReader{
			captureState: common.CaptureState{StateHash: "abc123"},
		},
		stubMutationAuthorizer: stubMutationAuthorizer{},
	}
	handler, err := NewHandler(Config{}, service, nil)
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}

	server := httptest.NewServer(handler)
	defer server.Close()
	_, _ = postJSONRPC(t, server.Client(), server.URL, initializeRequest())

	_, callResp := postJSONRPC(t, server.Client(), server.URL, callToolRequest(202, "till.project", mergeArgs(validSessionArgs(), map[string]any{
		"operation":           "create",
		"name":                "Project One",
		"kind":                "go-service",
		"template_library_id": "go-defaults",
	})))
	if isError, _ := callResp.Result["isError"].(bool); isError {
		t.Fatalf("create_project returned isError=true: %#v", callResp.Result)
	}
	if got := service.lastCreateProjectReq.TemplateLibraryID; got != "go-defaults" {
		t.Fatalf("create_project template_library_id = %q, want go-defaults", got)
	}
	if got := service.lastCreateProjectReq.Kind; got != "go-service" {
		t.Fatalf("create_project kind = %q, want go-service", got)
	}
}

// TestHandlerExpandedGlobalAdminMutationsUseRootedProjectAuthScope verifies global/bootstrap admin tools authorize against a rooted project scope.
func TestHandlerExpandedGlobalAdminMutationsUseRootedProjectAuthScope(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name          string
		tool          string
		args          map[string]any
		wantNamespace string
		wantProjectID string
	}{
		{
			name: "create project uses global sentinel scope",
			tool: "till.project",
			args: mergeArgs(validSessionArgs(), map[string]any{
				"operation":         "create",
				"name":              "Project One",
				"agent_instance_id": "inst-1",
				"lease_token":       "tok-1",
			}),
			wantNamespace: "project:" + domain.AuthRequestGlobalProjectID,
			wantProjectID: domain.AuthRequestGlobalProjectID,
		},
		{
			name: "upsert kind definition uses global sentinel scope",
			tool: "till.kind",
			args: mergeArgs(validSessionArgs(), map[string]any{
				"operation":  "upsert",
				"id":         "go-project",
				"applies_to": []any{"project"},
			}),
			wantNamespace: "project:" + domain.AuthRequestGlobalProjectID,
			wantProjectID: domain.AuthRequestGlobalProjectID,
		},
		{
			name: "global template library uses global sentinel scope",
			tool: "till.template",
			args: mergeArgs(validSessionArgs(), map[string]any{
				"operation": "upsert",
				"library": map[string]any{
					"id":     "go-defaults",
					"scope":  "global",
					"name":   "Go Defaults",
					"status": "approved",
				},
			}),
			wantNamespace: "project:" + domain.AuthRequestGlobalProjectID,
			wantProjectID: domain.AuthRequestGlobalProjectID,
		},
		{
			name: "project template library stays rooted to its project",
			tool: "till.template",
			args: mergeArgs(validSessionArgs(), map[string]any{
				"operation": "upsert",
				"library": map[string]any{
					"id":         "go-defaults-p1",
					"scope":      "project",
					"project_id": "p1",
					"name":       "Go Defaults P1",
					"status":     "draft",
				},
			}),
			wantNamespace: "project:p1",
			wantProjectID: "p1",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			service := &stubExpandedService{
				stubCaptureStateReader: stubCaptureStateReader{
					captureState: common.CaptureState{StateHash: "abc123"},
				},
				stubMutationAuthorizer: stubMutationAuthorizer{},
			}
			handler, err := NewHandler(Config{}, service, nil)
			if err != nil {
				t.Fatalf("NewHandler() error = %v", err)
			}

			server := httptest.NewServer(handler)
			defer server.Close()
			_, _ = postJSONRPC(t, server.Client(), server.URL, initializeRequest())

			_, callResp := postJSONRPC(t, server.Client(), server.URL, callToolRequest(301, tc.tool, tc.args))
			if isError, _ := callResp.Result["isError"].(bool); isError {
				t.Fatalf("%s returned isError=true: %#v", tc.tool, callResp.Result)
			}
			if got := service.lastAuthRequest.Namespace; got != tc.wantNamespace {
				t.Fatalf("%s namespace = %q, want %q", tc.tool, got, tc.wantNamespace)
			}
			if got := service.lastAuthRequest.Context["project_id"]; got != tc.wantProjectID {
				t.Fatalf("%s context project_id = %q, want %q", tc.tool, got, tc.wantProjectID)
			}
			if got := service.lastAuthRequest.Context["scope_type"]; got != string(domain.ScopeLevelProject) {
				t.Fatalf("%s context scope_type = %q, want %q", tc.tool, got, domain.ScopeLevelProject)
			}
			if got := service.lastAuthRequest.Context["scope_id"]; got != tc.wantProjectID {
				t.Fatalf("%s context scope_id = %q, want %q", tc.tool, got, tc.wantProjectID)
			}
		})
	}
}

// TestHandlerExpandedMutationAuthErrorsMap verifies session/auth outcomes surface as deterministic tool-result errors for expanded mutation tools.
func TestHandlerExpandedMutationAuthErrorsMap(t *testing.T) {
	cases := []struct {
		name       string
		authErr    error
		wantPrefix string
	}{
		{
			name:       "invalid auth",
			authErr:    errors.Join(common.ErrInvalidAuthentication, errors.New("bad secret")),
			wantPrefix: "invalid_auth:",
		},
		{
			name:       "session expired",
			authErr:    errors.Join(common.ErrSessionExpired, errors.New("expired")),
			wantPrefix: "session_expired:",
		},
		{
			name:       "auth denied",
			authErr:    errors.Join(common.ErrAuthorizationDenied, errors.New("policy deny")),
			wantPrefix: "auth_denied:",
		},
		{
			name:       "grant required",
			authErr:    errors.Join(common.ErrGrantRequired, errors.New("approval needed")),
			wantPrefix: "grant_required:",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			service := &stubExpandedService{
				stubCaptureStateReader: stubCaptureStateReader{
					captureState: common.CaptureState{StateHash: "abc123"},
				},
				stubMutationAuthorizer: stubMutationAuthorizer{
					authErr: tc.authErr,
				},
			}
			handler, err := NewHandler(Config{}, service, nil)
			if err != nil {
				t.Fatalf("NewHandler() error = %v", err)
			}

			server := httptest.NewServer(handler)
			defer server.Close()
			_, _ = postJSONRPC(t, server.Client(), server.URL, initializeRequest())

			_, callResp := postJSONRPC(t, server.Client(), server.URL, callToolRequest(1201, "till.project", mergeArgs(validSessionArgs(), map[string]any{
				"operation":         "create",
				"name":              "Project One",
				"agent_instance_id": "inst-1",
				"lease_token":       "lease-1",
			})))
			if isError, _ := callResp.Result["isError"].(bool); !isError {
				t.Fatalf("isError = %v, want true", callResp.Result["isError"])
			}
			if got := toolResultText(t, callResp.Result); !strings.HasPrefix(got, tc.wantPrefix) {
				t.Fatalf("error text = %q, want prefix %q", got, tc.wantPrefix)
			}
		})
	}
}
