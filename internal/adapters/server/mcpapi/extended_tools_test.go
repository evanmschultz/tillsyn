package mcpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/hylla/tillsyn/internal/adapters/server/common"
	"github.com/hylla/tillsyn/internal/domain"
)

// stubExpandedService provides deterministic responses for expanded MCP tool coverage tests.
type stubExpandedService struct {
	stubCaptureStateReader
	lastCreateTaskReq    common.CreateTaskRequest
	lastUpdateTaskReq    common.UpdateTaskRequest
	lastRestoreTaskReq   common.RestoreTaskRequest
	lastCreateCommentReq common.CreateCommentRequest
	lastListCommentReq   common.ListCommentsByTargetRequest
	lastSearchTasksReq   common.SearchTasksRequest
}

// GetBootstrapGuide returns one deterministic bootstrap payload.
func (s *stubExpandedService) GetBootstrapGuide(_ context.Context) (common.BootstrapGuide, error) {
	return common.BootstrapGuide{
		Mode:    "bootstrap_required",
		Summary: "create project",
	}, nil
}

// ListProjects returns one deterministic project row.
func (s *stubExpandedService) ListProjects(_ context.Context, _ bool) ([]domain.Project, error) {
	now := time.Date(2026, 2, 24, 12, 0, 0, 0, time.UTC)
	return []domain.Project{
		{
			ID:        "p1",
			Slug:      "proj-1",
			Name:      "Project One",
			CreatedAt: now,
			UpdatedAt: now,
		},
	}, nil
}

// CreateProject returns one deterministic project row.
func (s *stubExpandedService) CreateProject(_ context.Context, _ common.CreateProjectRequest) (domain.Project, error) {
	now := time.Date(2026, 2, 24, 12, 0, 0, 0, time.UTC)
	return domain.Project{ID: "p1", Slug: "proj-1", Name: "Project One", CreatedAt: now, UpdatedAt: now}, nil
}

// UpdateProject returns one deterministic updated project row.
func (s *stubExpandedService) UpdateProject(_ context.Context, _ common.UpdateProjectRequest) (domain.Project, error) {
	now := time.Date(2026, 2, 24, 12, 0, 0, 0, time.UTC)
	return domain.Project{ID: "p1", Slug: "proj-1", Name: "Project One Updated", CreatedAt: now, UpdatedAt: now}, nil
}

// ListTasks returns one deterministic task row.
func (s *stubExpandedService) ListTasks(_ context.Context, _ string, _ bool) ([]domain.Task, error) {
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
func (s *stubExpandedService) ListChildTasks(_ context.Context, _, _ string, _ bool) ([]domain.Task, error) {
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

// SearchTasks returns one deterministic match row.
func (s *stubExpandedService) SearchTasks(_ context.Context, in common.SearchTasksRequest) ([]common.SearchTaskMatch, error) {
	s.lastSearchTasksReq = in
	now := time.Date(2026, 2, 24, 12, 0, 0, 0, time.UTC)
	return []common.SearchTaskMatch{
		{
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
			StateID: "todo",
		},
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
func (s *stubExpandedService) ListKindDefinitions(_ context.Context, _ bool) ([]domain.KindDefinition, error) {
	now := time.Date(2026, 2, 24, 12, 0, 0, 0, time.UTC)
	return []domain.KindDefinition{
		{
			ID:          domain.KindID("phase"),
			DisplayName: "Phase",
			AppliesTo:   []domain.KindAppliesTo{domain.KindAppliesToPhase},
			CreatedAt:   now,
			UpdatedAt:   now,
		},
	}, nil
}

// UpsertKindDefinition returns one deterministic kind row.
func (s *stubExpandedService) UpsertKindDefinition(_ context.Context, _ common.UpsertKindDefinitionRequest) (domain.KindDefinition, error) {
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
func (s *stubExpandedService) SetProjectAllowedKinds(_ context.Context, _ common.SetProjectAllowedKindsRequest) error {
	return nil
}

// ListProjectAllowedKinds returns deterministic allowlist rows.
func (s *stubExpandedService) ListProjectAllowedKinds(_ context.Context, _ string) ([]string, error) {
	return []string{"phase", "task"}, nil
}

// IssueCapabilityLease returns one deterministic lease row.
func (s *stubExpandedService) IssueCapabilityLease(_ context.Context, _ common.IssueCapabilityLeaseRequest) (domain.CapabilityLease, error) {
	now := time.Date(2026, 2, 24, 12, 0, 0, 0, time.UTC)
	expiresAt := now.Add(time.Hour)
	return domain.CapabilityLease{
		InstanceID:  "inst-1",
		LeaseToken:  "tok-1",
		AgentName:   "agent-1",
		ProjectID:   "p1",
		ScopeType:   domain.CapabilityScopeProject,
		ScopeID:     "p1",
		Role:        domain.CapabilityRoleWorker,
		IssuedAt:    now,
		ExpiresAt:   expiresAt,
		HeartbeatAt: now,
	}, nil
}

// HeartbeatCapabilityLease returns one deterministic lease row.
func (s *stubExpandedService) HeartbeatCapabilityLease(_ context.Context, _ common.HeartbeatCapabilityLeaseRequest) (domain.CapabilityLease, error) {
	return s.IssueCapabilityLease(context.Background(), common.IssueCapabilityLeaseRequest{})
}

// RenewCapabilityLease returns one deterministic lease row.
func (s *stubExpandedService) RenewCapabilityLease(_ context.Context, _ common.RenewCapabilityLeaseRequest) (domain.CapabilityLease, error) {
	return s.IssueCapabilityLease(context.Background(), common.IssueCapabilityLeaseRequest{})
}

// RevokeCapabilityLease returns one deterministic lease row.
func (s *stubExpandedService) RevokeCapabilityLease(_ context.Context, _ common.RevokeCapabilityLeaseRequest) (domain.CapabilityLease, error) {
	lease, _ := s.IssueCapabilityLease(context.Background(), common.IssueCapabilityLeaseRequest{})
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
		"till.list_projects",
		"till.create_project",
		"till.update_project",
		"till.list_tasks",
		"till.create_task",
		"till.update_task",
		"till.move_task",
		"till.delete_task",
		"till.restore_task",
		"till.reparent_task",
		"till.list_child_tasks",
		"till.search_task_matches",
		"till.list_project_change_events",
		"till.get_project_dependency_rollup",
		"till.list_kind_definitions",
		"till.upsert_kind_definition",
		"till.set_project_allowed_kinds",
		"till.list_project_allowed_kinds",
		"till.issue_capability_lease",
		"till.heartbeat_capability_lease",
		"till.renew_capability_lease",
		"till.revoke_capability_lease",
		"till.revoke_all_capability_leases",
		"till.create_comment",
		"till.list_comments_by_target",
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
		{name: "till.list_projects", args: map[string]any{"include_archived": true}},
		{name: "till.create_project", args: map[string]any{
			"name":              "Project One",
			"actor_type":        "agent_orchestrator",
			"agent_name":        "agent-1",
			"agent_instance_id": "inst-1",
			"lease_token":       "tok-1",
		}},
		{name: "till.update_project", args: map[string]any{
			"project_id":        "p1",
			"name":              "Project One Updated",
			"actor_type":        "agent_orchestrator",
			"agent_name":        "agent-1",
			"agent_instance_id": "inst-1",
			"lease_token":       "tok-1",
		}},
		{name: "till.list_tasks", args: map[string]any{"project_id": "p1"}},
		{name: "till.create_task", args: map[string]any{
			"project_id":        "p1",
			"column_id":         "c1",
			"title":             "Task One",
			"actor_type":        "agent_orchestrator",
			"agent_name":        "agent-1",
			"agent_instance_id": "inst-1",
			"lease_token":       "tok-1",
		}},
		{name: "till.update_task", args: map[string]any{
			"task_id":           "t1",
			"title":             "Task One Updated",
			"actor_type":        "agent_subagent",
			"agent_name":        "agent-1",
			"agent_instance_id": "inst-1",
			"lease_token":       "tok-1",
		}},
		{name: "till.move_task", args: map[string]any{
			"task_id":           "t1",
			"to_column_id":      "c2",
			"position":          1,
			"actor_type":        "agent_orchestrator",
			"agent_name":        "agent-1",
			"agent_instance_id": "inst-1",
			"lease_token":       "tok-1",
		}},
		{name: "till.delete_task", args: map[string]any{
			"task_id":           "t1",
			"actor_type":        "agent_orchestrator",
			"agent_name":        "agent-1",
			"agent_instance_id": "inst-1",
			"lease_token":       "tok-1",
		}},
		{name: "till.restore_task", args: map[string]any{
			"task_id":           "t1",
			"actor_type":        "agent_subagent",
			"agent_name":        "agent-1",
			"agent_instance_id": "inst-1",
			"lease_token":       "tok-1",
		}},
		{name: "till.reparent_task", args: map[string]any{
			"task_id":           "t1",
			"parent_id":         "parent-1",
			"actor_type":        "agent_orchestrator",
			"agent_name":        "agent-1",
			"agent_instance_id": "inst-1",
			"lease_token":       "tok-1",
		}},
		{name: "till.list_child_tasks", args: map[string]any{"project_id": "p1", "parent_id": "parent-1"}},
		{name: "till.search_task_matches", args: map[string]any{"project_id": "p1", "query": "task"}},
		{name: "till.list_project_change_events", args: map[string]any{"project_id": "p1", "limit": 25}},
		{name: "till.get_project_dependency_rollup", args: map[string]any{"project_id": "p1"}},
		{name: "till.list_kind_definitions", args: map[string]any{}},
		{name: "till.upsert_kind_definition", args: map[string]any{"id": "phase", "applies_to": []any{"phase"}}},
		{name: "till.set_project_allowed_kinds", args: map[string]any{"project_id": "p1", "kind_ids": []any{"phase", "task"}}},
		{name: "till.list_project_allowed_kinds", args: map[string]any{"project_id": "p1"}},
		{name: "till.issue_capability_lease", args: map[string]any{"project_id": "p1", "scope_type": "project", "role": "worker", "agent_name": "agent-1"}},
		{name: "till.heartbeat_capability_lease", args: map[string]any{"agent_instance_id": "inst-1", "lease_token": "tok-1"}},
		{name: "till.renew_capability_lease", args: map[string]any{"agent_instance_id": "inst-1", "lease_token": "tok-1", "ttl_seconds": 60}},
		{name: "till.revoke_capability_lease", args: map[string]any{"agent_instance_id": "inst-1"}},
		{name: "till.revoke_all_capability_leases", args: map[string]any{"project_id": "p1", "scope_type": "project"}},
		{name: "till.create_comment", args: map[string]any{
			"project_id":        "p1",
			"target_type":       "task",
			"target_id":         "t1",
			"summary":           "Thread summary",
			"body_markdown":     "hello",
			"actor_type":        "agent_orchestrator",
			"agent_name":        "agent-1",
			"agent_instance_id": "inst-1",
			"lease_token":       "tok-1",
		}},
		{name: "till.list_comments_by_target", args: map[string]any{"project_id": "p1", "target_type": "task", "target_id": "t1"}},
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
	createSchema := findToolSchemaByName(t, toolsRaw, "till.create_comment")
	requiredRaw, _ := createSchema["required"].([]any)
	required := make([]string, 0, len(requiredRaw))
	for _, item := range requiredRaw {
		if value, ok := item.(string); ok {
			required = append(required, value)
		}
	}
	if !slices.Contains(required, "summary") {
		t.Fatalf("create_comment required args missing summary: %#v", required)
	}
	summaryDesc := schemaStringPropertyDescription(t, createSchema, "summary")
	if !strings.Contains(strings.ToLower(summaryDesc), "markdown-rich") {
		t.Fatalf("summary description = %q, want markdown-rich guidance", summaryDesc)
	}
	bodyDesc := schemaStringPropertyDescription(t, createSchema, "body_markdown")
	if !strings.Contains(strings.ToLower(bodyDesc), "markdown-rich") {
		t.Fatalf("body_markdown description = %q, want markdown-rich guidance", bodyDesc)
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

	searchSchema := findToolSchemaByName(t, toolsRaw, "till.search_task_matches")
	modeDesc := schemaStringPropertyDescription(t, searchSchema, "mode")
	if !strings.Contains(modeDesc, "default hybrid") {
		t.Fatalf("mode description = %q, want default hybrid guidance", modeDesc)
	}
	if !strings.Contains(modeDesc, "fall back to keyword") {
		t.Fatalf("mode description = %q, want keyword fallback guidance", modeDesc)
	}
	modeEnum := schemaPropertyEnumStrings(t, searchSchema, "mode")
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

	_, callResp := postJSONRPC(t, server.Client(), server.URL, callToolRequest(610, "till.search_task_matches", map[string]any{
		"project_id":       "p1",
		"query":            "task",
		"cross_project":    true,
		"include_archived": true,
		"states":           []any{"todo"},
		"levels":           []any{"phase"},
		"kinds":            []any{"phase"},
		"labels_any":       []any{"backend", "ops"},
		"labels_all":       []any{"urgent"},
		"mode":             "hybrid",
		"sort":             "title_asc",
		"limit":            75,
		"offset":           10,
	}))
	if isError, _ := callResp.Result["isError"].(bool); isError {
		t.Fatalf("search_task_matches returned isError=true: %#v", callResp.Result)
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

	_, defaultResp := postJSONRPC(t, server.Client(), server.URL, callToolRequest(611, "till.search_task_matches", map[string]any{
		"project_id": "p1",
	}))
	if isError, _ := defaultResp.Result["isError"].(bool); isError {
		t.Fatalf("default search_task_matches returned isError=true: %#v", defaultResp.Result)
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

// TestHandlerExpandedToolForwardsActorTupleFields verifies actor tuple fields flow through task/comment tool requests.
func TestHandlerExpandedToolForwardsActorTupleFields(t *testing.T) {
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

	_, createResp := postJSONRPC(t, server.Client(), server.URL, callToolRequest(300, "till.create_task", map[string]any{
		"project_id":        "p1",
		"column_id":         "c1",
		"title":             "Task One",
		"actor_type":        "agent_orchestrator",
		"actor_id":          "actor-1",
		"actor_name":        "Actor One",
		"agent_name":        "agent-name",
		"agent_instance_id": "inst-1",
		"lease_token":       "lease-1",
	}))
	if isError, _ := createResp.Result["isError"].(bool); isError {
		t.Fatalf("create_task returned isError=true: %#v", createResp.Result)
	}
	if got := service.lastCreateTaskReq.Actor.ActorType; got != "agent" {
		t.Fatalf("create_task actor_type = %q, want agent", got)
	}
	if got := service.lastCreateTaskReq.Actor.ActorID; got != "actor-1" {
		t.Fatalf("create_task actor_id = %q, want actor-1", got)
	}
	if got := service.lastCreateTaskReq.Actor.ActorName; got != "Actor One" {
		t.Fatalf("create_task actor_name = %q, want Actor One", got)
	}

	_, updateResp := postJSONRPC(t, server.Client(), server.URL, callToolRequest(301, "till.update_task", map[string]any{
		"task_id":           "t1",
		"title":             "Task One Updated",
		"actor_type":        "agent_subagent",
		"actor_id":          "upd-1",
		"actor_name":        "Updater One",
		"agent_name":        "EVAN",
		"agent_instance_id": "inst-1",
		"lease_token":       "lease-1",
	}))
	if isError, _ := updateResp.Result["isError"].(bool); isError {
		t.Fatalf("update_task returned isError=true: %#v", updateResp.Result)
	}
	if got := service.lastUpdateTaskReq.Actor.ActorType; got != "agent" {
		t.Fatalf("update_task actor_type = %q, want agent", got)
	}
	if got := service.lastUpdateTaskReq.Actor.AgentName; got != "EVAN" {
		t.Fatalf("update_task agent_name = %q, want EVAN", got)
	}
	if got := service.lastUpdateTaskReq.Actor.ActorID; got != "upd-1" {
		t.Fatalf("update_task actor_id = %q, want upd-1", got)
	}
	if got := service.lastUpdateTaskReq.Actor.ActorName; got != "Updater One" {
		t.Fatalf("update_task actor_name = %q, want Updater One", got)
	}

	_, commentResp := postJSONRPC(t, server.Client(), server.URL, callToolRequest(3011, "till.create_comment", map[string]any{
		"project_id":        "p1",
		"target_type":       "task",
		"target_id":         "t1",
		"summary":           "Thread summary",
		"body_markdown":     "hello",
		"actor_id":          "commenter-1",
		"actor_name":        "Commenter One",
		"actor_type":        "agent_orchestrator",
		"agent_name":        "agent-comment",
		"agent_instance_id": "inst-comment",
		"lease_token":       "lease-comment",
	}))
	if isError, _ := commentResp.Result["isError"].(bool); isError {
		t.Fatalf("create_comment returned isError=true: %#v", commentResp.Result)
	}
	if got := service.lastCreateCommentReq.Actor.ActorType; got != "agent" {
		t.Fatalf("create_comment actor_type = %q, want agent", got)
	}
	if got := service.lastCreateCommentReq.Actor.ActorID; got != "commenter-1" {
		t.Fatalf("create_comment actor_id = %q, want commenter-1", got)
	}
	if got := service.lastCreateCommentReq.Actor.ActorName; got != "Commenter One" {
		t.Fatalf("create_comment actor_name = %q, want Commenter One", got)
	}
	if got := service.lastCreateCommentReq.Summary; got != "Thread summary" {
		t.Fatalf("create_comment summary = %q, want Thread summary", got)
	}

	_, restoreResp := postJSONRPC(t, server.Client(), server.URL, callToolRequest(302, "till.restore_task", map[string]any{
		"task_id":           "t1",
		"actor_type":        "agent_subagent",
		"agent_name":        "agent-1",
		"agent_instance_id": "agent-1",
		"lease_token":       "lease-1",
		"override_token":    "override-1",
	}))
	if isError, _ := restoreResp.Result["isError"].(bool); isError {
		t.Fatalf("restore_task returned isError=true: %#v", restoreResp.Result)
	}
	if got := service.lastRestoreTaskReq.Actor.ActorType; got != "agent" {
		t.Fatalf("restore_task actor_type = %q, want agent", got)
	}
	if got := service.lastRestoreTaskReq.Actor.AgentName; got != "agent-1" {
		t.Fatalf("restore_task agent_name = %q, want agent-1", got)
	}
	if got := service.lastRestoreTaskReq.Actor.AgentInstanceID; got != "agent-1" {
		t.Fatalf("restore_task agent_instance_id = %q, want agent-1", got)
	}
	if got := service.lastRestoreTaskReq.Actor.LeaseToken; got != "lease-1" {
		t.Fatalf("restore_task lease_token = %q, want lease-1", got)
	}
	if got := service.lastRestoreTaskReq.Actor.OverrideToken; got != "override-1" {
		t.Fatalf("restore_task override_token = %q, want override-1", got)
	}
}

// TestHandlerExpandedToolRejectsMCPUserSystemActorsAndMissingLease verifies MCP mutation actor policy enforcement.
func TestHandlerExpandedToolRejectsMCPUserSystemActorsAndMissingLease(t *testing.T) {
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

	userHTTPResp, userResp := postJSONRPC(t, server.Client(), server.URL, callToolRequest(4010, "till.create_task", map[string]any{
		"project_id":        "p1",
		"column_id":         "c1",
		"title":             "user mutation",
		"actor_type":        "user",
		"agent_name":        "agent-1",
		"agent_instance_id": "inst-1",
		"lease_token":       "lease-1",
	}))
	if userHTTPResp.StatusCode == http.StatusOK {
		if isError, _ := userResp.Result["isError"].(bool); !isError && len(userResp.Error) == 0 {
			t.Fatalf("user actor_type call isError = %v, want true", userResp.Result["isError"])
		}
	}
	if isError, _ := userResp.Result["isError"].(bool); isError {
		if got := toolResultText(t, userResp.Result); !strings.Contains(got, "actor_type must be") {
			t.Fatalf("user actor_type error = %q, want actor_type guidance", got)
		}
	}
	if userHTTPResp.StatusCode != http.StatusOK && userHTTPResp.StatusCode != http.StatusBadRequest {
		t.Fatalf("user actor_type call isError = %v, want true", userResp.Result["isError"])
	}

	systemHTTPResp, systemResp := postJSONRPC(t, server.Client(), server.URL, callToolRequest(4011, "till.create_task", map[string]any{
		"project_id":        "p1",
		"column_id":         "c1",
		"title":             "system mutation",
		"actor_type":        "system",
		"agent_name":        "agent-1",
		"agent_instance_id": "inst-1",
		"lease_token":       "lease-1",
	}))
	if systemHTTPResp.StatusCode == http.StatusOK {
		if isError, _ := systemResp.Result["isError"].(bool); !isError && len(systemResp.Error) == 0 {
			t.Fatalf("system actor_type call isError = %v, want true", systemResp.Result["isError"])
		}
	}
	if isError, _ := systemResp.Result["isError"].(bool); isError {
		if got := toolResultText(t, systemResp.Result); !strings.Contains(got, "actor_type must be") {
			t.Fatalf("system actor_type error = %q, want actor_type guidance", got)
		}
	}
	if systemHTTPResp.StatusCode != http.StatusOK && systemHTTPResp.StatusCode != http.StatusBadRequest {
		t.Fatalf("system actor_type call isError = %v, want true", systemResp.Result["isError"])
	}

	missingLeaseHTTPResp, missingLeaseResp := postJSONRPC(t, server.Client(), server.URL, callToolRequest(4012, "till.create_task", map[string]any{
		"project_id":  "p1",
		"column_id":   "c1",
		"title":       "missing lease mutation",
		"actor_type":  "agent_orchestrator",
		"agent_name":  "agent-1",
		"lease_token": "lease-1",
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

	_, createResp := postJSONRPC(t, server.Client(), server.URL, callToolRequest(401, "till.create_comment", map[string]any{
		"project_id":        "p1",
		"target_type":       "branch",
		"target_id":         "branch-1",
		"summary":           "Branch note",
		"body_markdown":     "hello",
		"actor_type":        "agent_orchestrator",
		"agent_name":        "agent-1",
		"agent_instance_id": "inst-1",
		"lease_token":       "lease-1",
	}))
	if isError, _ := createResp.Result["isError"].(bool); isError {
		t.Fatalf("create_comment returned isError=true: %#v", createResp.Result)
	}
	if got := service.lastCreateCommentReq.TargetType; got != "branch" {
		t.Fatalf("create_comment target_type = %q, want branch", got)
	}
	if got := service.lastCreateCommentReq.TargetID; got != "branch-1" {
		t.Fatalf("create_comment target_id = %q, want branch-1", got)
	}

	_, listResp := postJSONRPC(t, server.Client(), server.URL, callToolRequest(402, "till.list_comments_by_target", map[string]any{
		"project_id":  "p1",
		"target_type": "subphase",
		"target_id":   "subphase-1",
	}))
	if isError, _ := listResp.Result["isError"].(bool); isError {
		t.Fatalf("list_comments_by_target returned isError=true: %#v", listResp.Result)
	}
	if got := service.lastListCommentReq.TargetType; got != "subphase" {
		t.Fatalf("list_comments_by_target target_type = %q, want subphase", got)
	}
	if got := service.lastListCommentReq.TargetID; got != "subphase-1" {
		t.Fatalf("list_comments_by_target target_id = %q, want subphase-1", got)
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

	_, callResp := postJSONRPC(t, server.Client(), server.URL, callToolRequest(201, "till.create_project", map[string]any{
		"name": 123,
	}))
	if isError, _ := callResp.Result["isError"].(bool); !isError {
		t.Fatalf("isError = %v, want true", callResp.Result["isError"])
	}
	if got := toolResultText(t, callResp.Result); !strings.HasPrefix(got, "invalid_request:") {
		t.Fatalf("error text = %q, want prefix invalid_request:", got)
	}
}
