package mcpapi

import (
	"context"
	"fmt"
	"strings"

	"github.com/hylla/tillsyn/internal/adapters/server/common"
	"github.com/hylla/tillsyn/internal/domain"
	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

// registerBootstrapTool registers the onboarding guidance tool for empty-instance flows.
func registerBootstrapTool(srv *mcpserver.MCPServer, guide common.BootstrapGuideReader) {
	if guide == nil {
		return
	}
	srv.AddTool(
		mcp.NewTool(
			"till.get_bootstrap_guide",
			mcp.WithDescription("Return bootstrap guidance when no project context exists yet."),
		),
		func(ctx context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			out, err := guide.GetBootstrapGuide(ctx)
			if err != nil {
				return toolResultFromError(err), nil
			}
			result, err := mcp.NewToolResultJSON(out)
			if err != nil {
				return nil, fmt.Errorf("encode get_bootstrap_guide result: %w", err)
			}
			return result, nil
		},
	)
}

// registerProjectTools registers list/create/update project tools.
func registerProjectTools(srv *mcpserver.MCPServer, projects common.ProjectService) {
	if projects == nil {
		return
	}

	srv.AddTool(
		mcp.NewTool(
			"till.list_projects",
			mcp.WithDescription("List projects."),
			mcp.WithBoolean("include_archived", mcp.Description("Include archived projects")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			rows, err := projects.ListProjects(ctx, req.GetBool("include_archived", false))
			if err != nil {
				return toolResultFromError(err), nil
			}
			result, err := mcp.NewToolResultJSON(map[string]any{"projects": rows})
			if err != nil {
				return nil, fmt.Errorf("encode list_projects result: %w", err)
			}
			return result, nil
		},
	)

	srv.AddTool(
		mcp.NewTool(
			"till.create_project",
			mcp.WithDescription("Create one project."),
			mcp.WithString("name", mcp.Required(), mcp.Description("Project name")),
			mcp.WithString("description", mcp.Description("Project description")),
			mcp.WithString("kind", mcp.Description("Project kind id")),
			mcp.WithObject("metadata", mcp.Description("Optional project metadata object")),
			mcp.WithString("actor_type", mcp.Description("user|agent|system"), mcp.Enum("user", "agent", "system")),
			mcp.WithString("agent_name", mcp.Description("Agent name for guarded calls")),
			mcp.WithString("agent_instance_id", mcp.Description("Agent instance id for guarded calls")),
			mcp.WithString("lease_token", mcp.Description("Lease token for guarded calls")),
			mcp.WithString("override_token", mcp.Description("Optional override token")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var args struct {
				Name            string                 `json:"name"`
				Description     string                 `json:"description"`
				Kind            string                 `json:"kind"`
				Metadata        domain.ProjectMetadata `json:"metadata"`
				ActorType       string                 `json:"actor_type"`
				AgentName       string                 `json:"agent_name"`
				AgentInstanceID string                 `json:"agent_instance_id"`
				LeaseToken      string                 `json:"lease_token"`
				OverrideToken   string                 `json:"override_token"`
			}
			if err := req.BindArguments(&args); err != nil {
				return invalidRequestToolResult(err), nil
			}
			if strings.TrimSpace(args.Name) == "" {
				return mcp.NewToolResultError(`invalid_request: required argument "name" not found`), nil
			}
			project, err := projects.CreateProject(ctx, common.CreateProjectRequest{
				Name:        args.Name,
				Description: args.Description,
				Kind:        args.Kind,
				Metadata:    args.Metadata,
				Actor: common.ActorLeaseTuple{
					ActorType:       args.ActorType,
					AgentName:       args.AgentName,
					AgentInstanceID: args.AgentInstanceID,
					LeaseToken:      args.LeaseToken,
					OverrideToken:   args.OverrideToken,
				},
			})
			if err != nil {
				return toolResultFromError(err), nil
			}
			result, err := mcp.NewToolResultJSON(project)
			if err != nil {
				return nil, fmt.Errorf("encode create_project result: %w", err)
			}
			return result, nil
		},
	)

	srv.AddTool(
		mcp.NewTool(
			"till.update_project",
			mcp.WithDescription("Update one project."),
			mcp.WithString("project_id", mcp.Required(), mcp.Description("Project identifier")),
			mcp.WithString("name", mcp.Required(), mcp.Description("Project name")),
			mcp.WithString("description", mcp.Description("Project description")),
			mcp.WithString("kind", mcp.Description("Project kind id")),
			mcp.WithObject("metadata", mcp.Description("Optional project metadata object")),
			mcp.WithString("actor_type", mcp.Description("user|agent|system"), mcp.Enum("user", "agent", "system")),
			mcp.WithString("agent_name", mcp.Description("Agent name for guarded calls")),
			mcp.WithString("agent_instance_id", mcp.Description("Agent instance id for guarded calls")),
			mcp.WithString("lease_token", mcp.Description("Lease token for guarded calls")),
			mcp.WithString("override_token", mcp.Description("Optional override token")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var args struct {
				ProjectID       string                 `json:"project_id"`
				Name            string                 `json:"name"`
				Description     string                 `json:"description"`
				Kind            string                 `json:"kind"`
				Metadata        domain.ProjectMetadata `json:"metadata"`
				ActorType       string                 `json:"actor_type"`
				AgentName       string                 `json:"agent_name"`
				AgentInstanceID string                 `json:"agent_instance_id"`
				LeaseToken      string                 `json:"lease_token"`
				OverrideToken   string                 `json:"override_token"`
			}
			if err := req.BindArguments(&args); err != nil {
				return invalidRequestToolResult(err), nil
			}
			if strings.TrimSpace(args.ProjectID) == "" {
				return mcp.NewToolResultError(`invalid_request: required argument "project_id" not found`), nil
			}
			if strings.TrimSpace(args.Name) == "" {
				return mcp.NewToolResultError(`invalid_request: required argument "name" not found`), nil
			}
			project, err := projects.UpdateProject(ctx, common.UpdateProjectRequest{
				ProjectID:   args.ProjectID,
				Name:        args.Name,
				Description: args.Description,
				Kind:        args.Kind,
				Metadata:    args.Metadata,
				Actor: common.ActorLeaseTuple{
					ActorType:       args.ActorType,
					AgentName:       args.AgentName,
					AgentInstanceID: args.AgentInstanceID,
					LeaseToken:      args.LeaseToken,
					OverrideToken:   args.OverrideToken,
				},
			})
			if err != nil {
				return toolResultFromError(err), nil
			}
			result, err := mcp.NewToolResultJSON(project)
			if err != nil {
				return nil, fmt.Errorf("encode update_project result: %w", err)
			}
			return result, nil
		},
	)
}

// registerTaskTools registers list/search/create/update/mutation task tools.
func registerTaskTools(
	srv *mcpserver.MCPServer,
	tasks common.TaskService,
	search common.SearchService,
	changes common.ChangeFeedService,
) {
	if tasks != nil {
		srv.AddTool(
			mcp.NewTool(
				"till.list_tasks",
				mcp.WithDescription("List tasks/work-items for one project."),
				mcp.WithString("project_id", mcp.Required(), mcp.Description("Project identifier")),
				mcp.WithBoolean("include_archived", mcp.Description("Include archived tasks")),
			),
			func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
				projectID, err := req.RequireString("project_id")
				if err != nil {
					return mcp.NewToolResultError(err.Error()), nil
				}
				rows, err := tasks.ListTasks(ctx, projectID, req.GetBool("include_archived", false))
				if err != nil {
					return toolResultFromError(err), nil
				}
				result, err := mcp.NewToolResultJSON(map[string]any{"tasks": rows})
				if err != nil {
					return nil, fmt.Errorf("encode list_tasks result: %w", err)
				}
				return result, nil
			},
		)

		srv.AddTool(
			mcp.NewTool(
				"till.create_task",
				mcp.WithDescription("Create one task/work-item (branch|phase|subphase|task|subtask via scope/kind)."),
				mcp.WithString("project_id", mcp.Required(), mcp.Description("Project identifier")),
				mcp.WithString("column_id", mcp.Required(), mcp.Description("Column identifier")),
				mcp.WithString("title", mcp.Required(), mcp.Description("Task title")),
				mcp.WithString("parent_id", mcp.Description("Optional parent task id")),
				mcp.WithString("kind", mcp.Description("Kind identifier")),
				mcp.WithString("scope", mcp.Description("project|branch|phase|subphase|task|subtask"), mcp.Enum(common.SupportedScopeTypes()...)),
				mcp.WithString("description", mcp.Description("Task description")),
				mcp.WithString("priority", mcp.Description("low|medium|high"), mcp.Enum("low", "medium", "high")),
				mcp.WithString("due_at", mcp.Description("Optional RFC3339 timestamp")),
				mcp.WithArray("labels", mcp.Description("Optional labels"), mcp.WithStringItems()),
				mcp.WithObject("metadata", mcp.Description("Optional task metadata object")),
				mcp.WithString("actor_id", mcp.Description("Optional actor id override")),
				mcp.WithString("actor_name", mcp.Description("Optional actor display name override")),
				mcp.WithString("actor_type", mcp.Description("user|agent|system"), mcp.Enum("user", "agent", "system")),
				mcp.WithString("agent_name", mcp.Description("Agent name for guarded calls")),
				mcp.WithString("agent_instance_id", mcp.Description("Agent instance id for guarded calls")),
				mcp.WithString("lease_token", mcp.Description("Lease token for guarded calls")),
				mcp.WithString("override_token", mcp.Description("Optional override token")),
			),
			func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
				var args struct {
					ProjectID       string              `json:"project_id"`
					ParentID        string              `json:"parent_id"`
					Kind            string              `json:"kind"`
					Scope           string              `json:"scope"`
					ColumnID        string              `json:"column_id"`
					Title           string              `json:"title"`
					Description     string              `json:"description"`
					Priority        string              `json:"priority"`
					DueAt           string              `json:"due_at"`
					Labels          []string            `json:"labels"`
					Metadata        domain.TaskMetadata `json:"metadata"`
					ActorID         string              `json:"actor_id"`
					ActorName       string              `json:"actor_name"`
					ActorType       string              `json:"actor_type"`
					AgentName       string              `json:"agent_name"`
					AgentInstanceID string              `json:"agent_instance_id"`
					LeaseToken      string              `json:"lease_token"`
					OverrideToken   string              `json:"override_token"`
				}
				if err := req.BindArguments(&args); err != nil {
					return invalidRequestToolResult(err), nil
				}
				if strings.TrimSpace(args.ProjectID) == "" {
					return mcp.NewToolResultError(`invalid_request: required argument "project_id" not found`), nil
				}
				if strings.TrimSpace(args.ColumnID) == "" {
					return mcp.NewToolResultError(`invalid_request: required argument "column_id" not found`), nil
				}
				if strings.TrimSpace(args.Title) == "" {
					return mcp.NewToolResultError(`invalid_request: required argument "title" not found`), nil
				}
				task, err := tasks.CreateTask(ctx, common.CreateTaskRequest{
					ProjectID:   args.ProjectID,
					ParentID:    args.ParentID,
					Kind:        args.Kind,
					Scope:       args.Scope,
					ColumnID:    args.ColumnID,
					Title:       args.Title,
					Description: args.Description,
					Priority:    args.Priority,
					DueAt:       args.DueAt,
					Labels:      append([]string(nil), args.Labels...),
					Metadata:    args.Metadata,
					Actor: common.ActorLeaseTuple{
						ActorID:         args.ActorID,
						ActorName:       args.ActorName,
						ActorType:       args.ActorType,
						AgentName:       args.AgentName,
						AgentInstanceID: args.AgentInstanceID,
						LeaseToken:      args.LeaseToken,
						OverrideToken:   args.OverrideToken,
					},
				})
				if err != nil {
					return toolResultFromError(err), nil
				}
				result, err := mcp.NewToolResultJSON(task)
				if err != nil {
					return nil, fmt.Errorf("encode create_task result: %w", err)
				}
				return result, nil
			},
		)

		srv.AddTool(
			mcp.NewTool(
				"till.update_task",
				mcp.WithDescription("Update one task/work-item."),
				mcp.WithString("task_id", mcp.Required(), mcp.Description("Task identifier")),
				mcp.WithString("title", mcp.Required(), mcp.Description("Task title")),
				mcp.WithString("description", mcp.Description("Task description")),
				mcp.WithString("priority", mcp.Description("low|medium|high"), mcp.Enum("low", "medium", "high")),
				mcp.WithString("due_at", mcp.Description("Optional RFC3339 timestamp")),
				mcp.WithArray("labels", mcp.Description("Optional labels"), mcp.WithStringItems()),
				mcp.WithObject("metadata", mcp.Description("Optional task metadata object")),
				mcp.WithString("actor_id", mcp.Description("Optional actor id override")),
				mcp.WithString("actor_name", mcp.Description("Optional actor display name override")),
				mcp.WithString("actor_type", mcp.Description("user|agent|system"), mcp.Enum("user", "agent", "system")),
				mcp.WithString("agent_name", mcp.Description("Agent name for guarded calls")),
				mcp.WithString("agent_instance_id", mcp.Description("Agent instance id for guarded calls")),
				mcp.WithString("lease_token", mcp.Description("Lease token for guarded calls")),
				mcp.WithString("override_token", mcp.Description("Optional override token")),
			),
			func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
				var args struct {
					TaskID          string               `json:"task_id"`
					Title           string               `json:"title"`
					Description     string               `json:"description"`
					Priority        string               `json:"priority"`
					DueAt           string               `json:"due_at"`
					Labels          []string             `json:"labels"`
					Metadata        *domain.TaskMetadata `json:"metadata"`
					ActorID         string               `json:"actor_id"`
					ActorName       string               `json:"actor_name"`
					ActorType       string               `json:"actor_type"`
					AgentName       string               `json:"agent_name"`
					AgentInstanceID string               `json:"agent_instance_id"`
					LeaseToken      string               `json:"lease_token"`
					OverrideToken   string               `json:"override_token"`
				}
				if err := req.BindArguments(&args); err != nil {
					return invalidRequestToolResult(err), nil
				}
				if strings.TrimSpace(args.TaskID) == "" {
					return mcp.NewToolResultError(`invalid_request: required argument "task_id" not found`), nil
				}
				if strings.TrimSpace(args.Title) == "" {
					return mcp.NewToolResultError(`invalid_request: required argument "title" not found`), nil
				}
				task, err := tasks.UpdateTask(ctx, common.UpdateTaskRequest{
					TaskID:      args.TaskID,
					Title:       args.Title,
					Description: args.Description,
					Priority:    args.Priority,
					DueAt:       args.DueAt,
					Labels:      append([]string(nil), args.Labels...),
					Metadata:    args.Metadata,
					Actor: common.ActorLeaseTuple{
						ActorID:         args.ActorID,
						ActorName:       args.ActorName,
						ActorType:       args.ActorType,
						AgentName:       args.AgentName,
						AgentInstanceID: args.AgentInstanceID,
						LeaseToken:      args.LeaseToken,
						OverrideToken:   args.OverrideToken,
					},
				})
				if err != nil {
					return toolResultFromError(err), nil
				}
				result, err := mcp.NewToolResultJSON(task)
				if err != nil {
					return nil, fmt.Errorf("encode update_task result: %w", err)
				}
				return result, nil
			},
		)

		srv.AddTool(
			mcp.NewTool(
				"till.move_task",
				mcp.WithDescription("Move one task/work-item to another column/position."),
				mcp.WithString("task_id", mcp.Required(), mcp.Description("Task identifier")),
				mcp.WithString("to_column_id", mcp.Required(), mcp.Description("Destination column identifier")),
				mcp.WithNumber("position", mcp.Required(), mcp.Description("Destination position")),
				mcp.WithString("actor_type", mcp.Description("user|agent|system"), mcp.Enum("user", "agent", "system")),
				mcp.WithString("agent_name", mcp.Description("Agent name for guarded calls")),
				mcp.WithString("agent_instance_id", mcp.Description("Agent instance id for guarded calls")),
				mcp.WithString("lease_token", mcp.Description("Lease token for guarded calls")),
				mcp.WithString("override_token", mcp.Description("Optional override token")),
			),
			func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
				taskID, err := req.RequireString("task_id")
				if err != nil {
					return mcp.NewToolResultError(err.Error()), nil
				}
				toColumnID, err := req.RequireString("to_column_id")
				if err != nil {
					return mcp.NewToolResultError(err.Error()), nil
				}
				position, err := req.RequireInt("position")
				if err != nil {
					return mcp.NewToolResultError(err.Error()), nil
				}
				task, err := tasks.MoveTask(ctx, common.MoveTaskRequest{
					TaskID:     taskID,
					ToColumnID: toColumnID,
					Position:   position,
					Actor: common.ActorLeaseTuple{
						ActorType:       req.GetString("actor_type", ""),
						AgentName:       req.GetString("agent_name", ""),
						AgentInstanceID: req.GetString("agent_instance_id", ""),
						LeaseToken:      req.GetString("lease_token", ""),
						OverrideToken:   req.GetString("override_token", ""),
					},
				})
				if err != nil {
					return toolResultFromError(err), nil
				}
				result, err := mcp.NewToolResultJSON(task)
				if err != nil {
					return nil, fmt.Errorf("encode move_task result: %w", err)
				}
				return result, nil
			},
		)

		srv.AddTool(
			mcp.NewTool(
				"till.delete_task",
				mcp.WithDescription("Delete one task/work-item (archive or hard)."),
				mcp.WithString("task_id", mcp.Required(), mcp.Description("Task identifier")),
				mcp.WithString("mode", mcp.Description("archive|hard"), mcp.Enum("archive", "hard")),
				mcp.WithString("actor_type", mcp.Description("user|agent|system"), mcp.Enum("user", "agent", "system")),
				mcp.WithString("agent_name", mcp.Description("Agent name for guarded calls")),
				mcp.WithString("agent_instance_id", mcp.Description("Agent instance id for guarded calls")),
				mcp.WithString("lease_token", mcp.Description("Lease token for guarded calls")),
				mcp.WithString("override_token", mcp.Description("Optional override token")),
			),
			func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
				taskID, err := req.RequireString("task_id")
				if err != nil {
					return mcp.NewToolResultError(err.Error()), nil
				}
				if err := tasks.DeleteTask(ctx, common.DeleteTaskRequest{
					TaskID: taskID,
					Mode:   req.GetString("mode", ""),
					Actor: common.ActorLeaseTuple{
						ActorType:       req.GetString("actor_type", ""),
						AgentName:       req.GetString("agent_name", ""),
						AgentInstanceID: req.GetString("agent_instance_id", ""),
						LeaseToken:      req.GetString("lease_token", ""),
						OverrideToken:   req.GetString("override_token", ""),
					},
				}); err != nil {
					return toolResultFromError(err), nil
				}
				result, err := mcp.NewToolResultJSON(map[string]any{
					"deleted": true,
					"task_id": taskID,
					"mode":    req.GetString("mode", ""),
				})
				if err != nil {
					return nil, fmt.Errorf("encode delete_task result: %w", err)
				}
				return result, nil
			},
		)

		srv.AddTool(
			mcp.NewTool(
				"till.restore_task",
				mcp.WithDescription("Restore one archived task/work-item."),
				mcp.WithString("task_id", mcp.Required(), mcp.Description("Task identifier")),
				mcp.WithString("actor_type", mcp.Description("user|agent|system"), mcp.Enum("user", "agent", "system")),
				mcp.WithString("agent_name", mcp.Description("Agent name for guarded calls")),
				mcp.WithString("agent_instance_id", mcp.Description("Agent instance id for guarded calls")),
				mcp.WithString("lease_token", mcp.Description("Lease token for guarded calls")),
				mcp.WithString("override_token", mcp.Description("Optional override token")),
			),
			func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
				taskID, err := req.RequireString("task_id")
				if err != nil {
					return mcp.NewToolResultError(err.Error()), nil
				}
				task, err := tasks.RestoreTask(ctx, common.RestoreTaskRequest{
					TaskID: taskID,
					Actor: common.ActorLeaseTuple{
						ActorType:       req.GetString("actor_type", ""),
						AgentName:       req.GetString("agent_name", ""),
						AgentInstanceID: req.GetString("agent_instance_id", ""),
						LeaseToken:      req.GetString("lease_token", ""),
						OverrideToken:   req.GetString("override_token", ""),
					},
				})
				if err != nil {
					return toolResultFromError(err), nil
				}
				result, err := mcp.NewToolResultJSON(task)
				if err != nil {
					return nil, fmt.Errorf("encode restore_task result: %w", err)
				}
				return result, nil
			},
		)

		srv.AddTool(
			mcp.NewTool(
				"till.reparent_task",
				mcp.WithDescription("Change parent relationship for one task/work-item."),
				mcp.WithString("task_id", mcp.Required(), mcp.Description("Task identifier")),
				mcp.WithString("parent_id", mcp.Description("New parent identifier (empty to unset where allowed)")),
				mcp.WithString("actor_type", mcp.Description("user|agent|system"), mcp.Enum("user", "agent", "system")),
				mcp.WithString("agent_name", mcp.Description("Agent name for guarded calls")),
				mcp.WithString("agent_instance_id", mcp.Description("Agent instance id for guarded calls")),
				mcp.WithString("lease_token", mcp.Description("Lease token for guarded calls")),
				mcp.WithString("override_token", mcp.Description("Optional override token")),
			),
			func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
				taskID, err := req.RequireString("task_id")
				if err != nil {
					return mcp.NewToolResultError(err.Error()), nil
				}
				task, err := tasks.ReparentTask(ctx, common.ReparentTaskRequest{
					TaskID:   taskID,
					ParentID: req.GetString("parent_id", ""),
					Actor: common.ActorLeaseTuple{
						ActorType:       req.GetString("actor_type", ""),
						AgentName:       req.GetString("agent_name", ""),
						AgentInstanceID: req.GetString("agent_instance_id", ""),
						LeaseToken:      req.GetString("lease_token", ""),
						OverrideToken:   req.GetString("override_token", ""),
					},
				})
				if err != nil {
					return toolResultFromError(err), nil
				}
				result, err := mcp.NewToolResultJSON(task)
				if err != nil {
					return nil, fmt.Errorf("encode reparent_task result: %w", err)
				}
				return result, nil
			},
		)

		srv.AddTool(
			mcp.NewTool(
				"till.list_child_tasks",
				mcp.WithDescription("List child tasks for a parent scope."),
				mcp.WithString("project_id", mcp.Required(), mcp.Description("Project identifier")),
				mcp.WithString("parent_id", mcp.Required(), mcp.Description("Parent task identifier")),
				mcp.WithBoolean("include_archived", mcp.Description("Include archived child rows")),
			),
			func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
				projectID, err := req.RequireString("project_id")
				if err != nil {
					return mcp.NewToolResultError(err.Error()), nil
				}
				parentID, err := req.RequireString("parent_id")
				if err != nil {
					return mcp.NewToolResultError(err.Error()), nil
				}
				rows, err := tasks.ListChildTasks(ctx, projectID, parentID, req.GetBool("include_archived", false))
				if err != nil {
					return toolResultFromError(err), nil
				}
				result, err := mcp.NewToolResultJSON(map[string]any{"tasks": rows})
				if err != nil {
					return nil, fmt.Errorf("encode list_child_tasks result: %w", err)
				}
				return result, nil
			},
		)
	}

	if search != nil {
		srv.AddTool(
			mcp.NewTool(
				"till.search_task_matches",
				mcp.WithDescription("Search task/work-item matches by query, states, and scope."),
				mcp.WithString("project_id", mcp.Description("Project identifier for non-cross-project queries")),
				mcp.WithString("query", mcp.Description("Search query")),
				mcp.WithBoolean("cross_project", mcp.Description("Search across all projects")),
				mcp.WithBoolean("include_archived", mcp.Description("Include archived projects/items")),
				mcp.WithArray("states", mcp.Description("Optional state filter"), mcp.WithStringItems()),
			),
			func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
				rows, err := search.SearchTasks(ctx, common.SearchTasksRequest{
					ProjectID:       req.GetString("project_id", ""),
					Query:           req.GetString("query", ""),
					CrossProject:    req.GetBool("cross_project", false),
					IncludeArchived: req.GetBool("include_archived", false),
					States:          req.GetStringSlice("states", nil),
				})
				if err != nil {
					return toolResultFromError(err), nil
				}
				result, err := mcp.NewToolResultJSON(map[string]any{"matches": rows})
				if err != nil {
					return nil, fmt.Errorf("encode search_task_matches result: %w", err)
				}
				return result, nil
			},
		)
	}

	if changes != nil {
		srv.AddTool(
			mcp.NewTool(
				"till.list_project_change_events",
				mcp.WithDescription("List recent project change events."),
				mcp.WithString("project_id", mcp.Required(), mcp.Description("Project identifier")),
				mcp.WithNumber("limit", mcp.Description("Maximum rows to return")),
			),
			func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
				projectID, err := req.RequireString("project_id")
				if err != nil {
					return mcp.NewToolResultError(err.Error()), nil
				}
				rows, err := changes.ListProjectChangeEvents(ctx, projectID, req.GetInt("limit", 25))
				if err != nil {
					return toolResultFromError(err), nil
				}
				result, err := mcp.NewToolResultJSON(map[string]any{"events": rows})
				if err != nil {
					return nil, fmt.Errorf("encode list_project_change_events result: %w", err)
				}
				return result, nil
			},
		)

		srv.AddTool(
			mcp.NewTool(
				"till.get_project_dependency_rollup",
				mcp.WithDescription("Return dependency/blocking rollups for one project."),
				mcp.WithString("project_id", mcp.Required(), mcp.Description("Project identifier")),
			),
			func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
				projectID, err := req.RequireString("project_id")
				if err != nil {
					return mcp.NewToolResultError(err.Error()), nil
				}
				rollup, err := changes.GetProjectDependencyRollup(ctx, projectID)
				if err != nil {
					return toolResultFromError(err), nil
				}
				result, err := mcp.NewToolResultJSON(rollup)
				if err != nil {
					return nil, fmt.Errorf("encode get_project_dependency_rollup result: %w", err)
				}
				return result, nil
			},
		)
	}
}

// registerKindTools registers kind catalog and project allowlist tools.
func registerKindTools(srv *mcpserver.MCPServer, kinds common.KindCatalogService) {
	if kinds == nil {
		return
	}

	srv.AddTool(
		mcp.NewTool(
			"till.list_kind_definitions",
			mcp.WithDescription("List kind catalog definitions."),
			mcp.WithBoolean("include_archived", mcp.Description("Include archived kind definitions")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			rows, err := kinds.ListKindDefinitions(ctx, req.GetBool("include_archived", false))
			if err != nil {
				return toolResultFromError(err), nil
			}
			result, err := mcp.NewToolResultJSON(map[string]any{"kinds": rows})
			if err != nil {
				return nil, fmt.Errorf("encode list_kind_definitions result: %w", err)
			}
			return result, nil
		},
	)

	srv.AddTool(
		mcp.NewTool(
			"till.upsert_kind_definition",
			mcp.WithDescription("Create or update one kind definition."),
			mcp.WithString("id", mcp.Required(), mcp.Description("Kind identifier")),
			mcp.WithString("display_name", mcp.Description("Kind display name")),
			mcp.WithString("description_markdown", mcp.Description("Kind description markdown")),
			mcp.WithArray("applies_to", mcp.Required(), mcp.Description("Allowed applies_to scope list"), mcp.WithStringItems()),
			mcp.WithArray("allowed_parent_scopes", mcp.Description("Allowed parent scope list"), mcp.WithStringItems()),
			mcp.WithString("payload_schema_json", mcp.Description("Optional payload schema JSON")),
			mcp.WithObject("template", mcp.Description("Optional template object")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var args struct {
				ID                  string              `json:"id"`
				DisplayName         string              `json:"display_name"`
				DescriptionMarkdown string              `json:"description_markdown"`
				AppliesTo           []string            `json:"applies_to"`
				AllowedParentScopes []string            `json:"allowed_parent_scopes"`
				PayloadSchemaJSON   string              `json:"payload_schema_json"`
				Template            domain.KindTemplate `json:"template"`
			}
			if err := req.BindArguments(&args); err != nil {
				return invalidRequestToolResult(err), nil
			}
			if strings.TrimSpace(args.ID) == "" {
				return mcp.NewToolResultError(`invalid_request: required argument "id" not found`), nil
			}
			if len(args.AppliesTo) == 0 {
				return mcp.NewToolResultError(`invalid_request: required argument "applies_to" not found`), nil
			}
			kind, err := kinds.UpsertKindDefinition(ctx, common.UpsertKindDefinitionRequest{
				ID:                  args.ID,
				DisplayName:         args.DisplayName,
				DescriptionMarkdown: args.DescriptionMarkdown,
				AppliesTo:           append([]string(nil), args.AppliesTo...),
				AllowedParentScopes: append([]string(nil), args.AllowedParentScopes...),
				PayloadSchemaJSON:   args.PayloadSchemaJSON,
				Template:            args.Template,
			})
			if err != nil {
				return toolResultFromError(err), nil
			}
			result, err := mcp.NewToolResultJSON(kind)
			if err != nil {
				return nil, fmt.Errorf("encode upsert_kind_definition result: %w", err)
			}
			return result, nil
		},
	)

	srv.AddTool(
		mcp.NewTool(
			"till.set_project_allowed_kinds",
			mcp.WithDescription("Set explicit project allowed kind identifiers."),
			mcp.WithString("project_id", mcp.Required(), mcp.Description("Project identifier")),
			mcp.WithArray("kind_ids", mcp.Required(), mcp.Description("Allowed kind id list"), mcp.WithStringItems()),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			projectID, err := req.RequireString("project_id")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			kindIDs, err := req.RequireStringSlice("kind_ids")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			if err := kinds.SetProjectAllowedKinds(ctx, common.SetProjectAllowedKindsRequest{
				ProjectID: projectID,
				KindIDs:   append([]string(nil), kindIDs...),
			}); err != nil {
				return toolResultFromError(err), nil
			}
			result, err := mcp.NewToolResultJSON(map[string]any{
				"updated":    true,
				"project_id": projectID,
				"kind_ids":   kindIDs,
			})
			if err != nil {
				return nil, fmt.Errorf("encode set_project_allowed_kinds result: %w", err)
			}
			return result, nil
		},
	)

	srv.AddTool(
		mcp.NewTool(
			"till.list_project_allowed_kinds",
			mcp.WithDescription("List explicit project allowed kind identifiers."),
			mcp.WithString("project_id", mcp.Required(), mcp.Description("Project identifier")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			projectID, err := req.RequireString("project_id")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			kindIDs, err := kinds.ListProjectAllowedKinds(ctx, projectID)
			if err != nil {
				return toolResultFromError(err), nil
			}
			result, err := mcp.NewToolResultJSON(map[string]any{"kind_ids": kindIDs})
			if err != nil {
				return nil, fmt.Errorf("encode list_project_allowed_kinds result: %w", err)
			}
			return result, nil
		},
	)
}

// registerCapabilityLeaseTools registers lease issue/heartbeat/renew/revoke tools.
func registerCapabilityLeaseTools(srv *mcpserver.MCPServer, leases common.CapabilityLeaseService) {
	if leases == nil {
		return
	}

	srv.AddTool(
		mcp.NewTool(
			"till.issue_capability_lease",
			mcp.WithDescription("Issue one capability lease."),
			mcp.WithString("project_id", mcp.Required(), mcp.Description("Project identifier")),
			mcp.WithString("scope_type", mcp.Required(), mcp.Description("project|branch|phase|subphase|task|subtask"), mcp.Enum(common.SupportedScopeTypes()...)),
			mcp.WithString("scope_id", mcp.Description("Scope identifier")),
			mcp.WithString("role", mcp.Required(), mcp.Description("orchestrator|worker|system"), mcp.Enum("orchestrator", "worker", "system")),
			mcp.WithString("agent_name", mcp.Required(), mcp.Description("Agent display/name identifier")),
			mcp.WithString("agent_instance_id", mcp.Description("Optional stable agent instance id")),
			mcp.WithString("parent_instance_id", mcp.Description("Optional parent lease instance id")),
			mcp.WithBoolean("allow_equal_scope_delegation", mcp.Description("Allow equal-scope delegation")),
			mcp.WithNumber("requested_ttl_seconds", mcp.Description("Optional TTL in seconds")),
			mcp.WithString("override_token", mcp.Description("Optional orchestrator overlap override token")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			projectID, err := req.RequireString("project_id")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			scopeType, err := req.RequireString("scope_type")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			role, err := req.RequireString("role")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			agentName, err := req.RequireString("agent_name")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			lease, err := leases.IssueCapabilityLease(ctx, common.IssueCapabilityLeaseRequest{
				ProjectID:                 projectID,
				ScopeType:                 scopeType,
				ScopeID:                   req.GetString("scope_id", ""),
				Role:                      role,
				AgentName:                 agentName,
				AgentInstanceID:           req.GetString("agent_instance_id", ""),
				ParentInstanceID:          req.GetString("parent_instance_id", ""),
				AllowEqualScopeDelegation: req.GetBool("allow_equal_scope_delegation", false),
				RequestedTTLSeconds:       req.GetInt("requested_ttl_seconds", 0),
				OverrideToken:             req.GetString("override_token", ""),
			})
			if err != nil {
				return toolResultFromError(err), nil
			}
			result, err := mcp.NewToolResultJSON(lease)
			if err != nil {
				return nil, fmt.Errorf("encode issue_capability_lease result: %w", err)
			}
			return result, nil
		},
	)

	srv.AddTool(
		mcp.NewTool(
			"till.heartbeat_capability_lease",
			mcp.WithDescription("Heartbeat one active capability lease."),
			mcp.WithString("agent_instance_id", mcp.Required(), mcp.Description("Agent instance identifier")),
			mcp.WithString("lease_token", mcp.Required(), mcp.Description("Lease token")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			instanceID, err := req.RequireString("agent_instance_id")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			leaseToken, err := req.RequireString("lease_token")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			lease, err := leases.HeartbeatCapabilityLease(ctx, common.HeartbeatCapabilityLeaseRequest{
				AgentInstanceID: instanceID,
				LeaseToken:      leaseToken,
			})
			if err != nil {
				return toolResultFromError(err), nil
			}
			result, err := mcp.NewToolResultJSON(lease)
			if err != nil {
				return nil, fmt.Errorf("encode heartbeat_capability_lease result: %w", err)
			}
			return result, nil
		},
	)

	srv.AddTool(
		mcp.NewTool(
			"till.renew_capability_lease",
			mcp.WithDescription("Renew one capability lease expiry."),
			mcp.WithString("agent_instance_id", mcp.Required(), mcp.Description("Agent instance identifier")),
			mcp.WithString("lease_token", mcp.Required(), mcp.Description("Lease token")),
			mcp.WithNumber("ttl_seconds", mcp.Description("Optional renewal TTL in seconds")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			instanceID, err := req.RequireString("agent_instance_id")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			leaseToken, err := req.RequireString("lease_token")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			lease, err := leases.RenewCapabilityLease(ctx, common.RenewCapabilityLeaseRequest{
				AgentInstanceID: instanceID,
				LeaseToken:      leaseToken,
				TTLSeconds:      req.GetInt("ttl_seconds", 0),
			})
			if err != nil {
				return toolResultFromError(err), nil
			}
			result, err := mcp.NewToolResultJSON(lease)
			if err != nil {
				return nil, fmt.Errorf("encode renew_capability_lease result: %w", err)
			}
			return result, nil
		},
	)

	srv.AddTool(
		mcp.NewTool(
			"till.revoke_capability_lease",
			mcp.WithDescription("Revoke one capability lease by instance id."),
			mcp.WithString("agent_instance_id", mcp.Required(), mcp.Description("Agent instance identifier")),
			mcp.WithString("reason", mcp.Description("Optional revocation reason")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			instanceID, err := req.RequireString("agent_instance_id")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			lease, err := leases.RevokeCapabilityLease(ctx, common.RevokeCapabilityLeaseRequest{
				AgentInstanceID: instanceID,
				Reason:          req.GetString("reason", ""),
			})
			if err != nil {
				return toolResultFromError(err), nil
			}
			result, err := mcp.NewToolResultJSON(lease)
			if err != nil {
				return nil, fmt.Errorf("encode revoke_capability_lease result: %w", err)
			}
			return result, nil
		},
	)

	srv.AddTool(
		mcp.NewTool(
			"till.revoke_all_capability_leases",
			mcp.WithDescription("Revoke all capability leases for one project scope."),
			mcp.WithString("project_id", mcp.Required(), mcp.Description("Project identifier")),
			mcp.WithString("scope_type", mcp.Required(), mcp.Description("project|branch|phase|subphase|task|subtask"), mcp.Enum(common.SupportedScopeTypes()...)),
			mcp.WithString("scope_id", mcp.Description("Scope identifier")),
			mcp.WithString("reason", mcp.Description("Optional revocation reason")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			projectID, err := req.RequireString("project_id")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			scopeType, err := req.RequireString("scope_type")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			if err := leases.RevokeAllCapabilityLeases(ctx, common.RevokeAllCapabilityLeasesRequest{
				ProjectID: projectID,
				ScopeType: scopeType,
				ScopeID:   req.GetString("scope_id", ""),
				Reason:    req.GetString("reason", ""),
			}); err != nil {
				return toolResultFromError(err), nil
			}
			result, err := mcp.NewToolResultJSON(map[string]any{
				"updated":    true,
				"project_id": projectID,
				"scope_type": scopeType,
				"scope_id":   req.GetString("scope_id", ""),
			})
			if err != nil {
				return nil, fmt.Errorf("encode revoke_all_capability_leases result: %w", err)
			}
			return result, nil
		},
	)
}

// registerCommentTools registers comment create/list tools.
func registerCommentTools(srv *mcpserver.MCPServer, comments common.CommentService) {
	if comments == nil {
		return
	}

	srv.AddTool(
		mcp.NewTool(
			"till.create_comment",
			mcp.WithDescription("Create one markdown comment for a project/branch/phase/subphase/task/subtask/decision/note target."),
			mcp.WithString("project_id", mcp.Required(), mcp.Description("Project identifier")),
			mcp.WithString("target_type", mcp.Required(), mcp.Description("project|branch|phase|subphase|task|subtask|decision|note"), mcp.Enum("project", "branch", "phase", "subphase", "task", "subtask", "decision", "note")),
			mcp.WithString("target_id", mcp.Required(), mcp.Description("Target identifier")),
			mcp.WithString("body_markdown", mcp.Required(), mcp.Description("Markdown body")),
			mcp.WithString("actor_id", mcp.Description("Optional actor id override")),
			mcp.WithString("actor_name", mcp.Description("Optional actor display name override")),
			mcp.WithString("actor_type", mcp.Description("user|agent|system"), mcp.Enum("user", "agent", "system")),
			mcp.WithString("agent_name", mcp.Description("Agent name for guarded calls")),
			mcp.WithString("agent_instance_id", mcp.Description("Agent instance id for guarded calls")),
			mcp.WithString("lease_token", mcp.Description("Lease token for guarded calls")),
			mcp.WithString("override_token", mcp.Description("Optional override token")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			projectID, err := req.RequireString("project_id")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			targetType, err := req.RequireString("target_type")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			targetID, err := req.RequireString("target_id")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			bodyMarkdown, err := req.RequireString("body_markdown")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			comment, err := comments.CreateComment(ctx, common.CreateCommentRequest{
				ProjectID:    projectID,
				TargetType:   targetType,
				TargetID:     targetID,
				BodyMarkdown: bodyMarkdown,
				Actor: common.ActorLeaseTuple{
					ActorID:         req.GetString("actor_id", ""),
					ActorName:       req.GetString("actor_name", ""),
					ActorType:       req.GetString("actor_type", ""),
					AgentName:       req.GetString("agent_name", ""),
					AgentInstanceID: req.GetString("agent_instance_id", ""),
					LeaseToken:      req.GetString("lease_token", ""),
					OverrideToken:   req.GetString("override_token", ""),
				},
			})
			if err != nil {
				return toolResultFromError(err), nil
			}
			result, err := mcp.NewToolResultJSON(comment)
			if err != nil {
				return nil, fmt.Errorf("encode create_comment result: %w", err)
			}
			return result, nil
		},
	)

	srv.AddTool(
		mcp.NewTool(
			"till.list_comments_by_target",
			mcp.WithDescription("List comments for one target."),
			mcp.WithString("project_id", mcp.Required(), mcp.Description("Project identifier")),
			mcp.WithString("target_type", mcp.Required(), mcp.Description("project|branch|phase|subphase|task|subtask|decision|note"), mcp.Enum("project", "branch", "phase", "subphase", "task", "subtask", "decision", "note")),
			mcp.WithString("target_id", mcp.Required(), mcp.Description("Target identifier")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			projectID, err := req.RequireString("project_id")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			targetType, err := req.RequireString("target_type")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			targetID, err := req.RequireString("target_id")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			rows, err := comments.ListCommentsByTarget(ctx, common.ListCommentsByTargetRequest{
				ProjectID:  projectID,
				TargetType: targetType,
				TargetID:   targetID,
			})
			if err != nil {
				return toolResultFromError(err), nil
			}
			result, err := mcp.NewToolResultJSON(map[string]any{"comments": rows})
			if err != nil {
				return nil, fmt.Errorf("encode list_comments_by_target result: %w", err)
			}
			return result, nil
		},
	)
}

// invalidRequestToolResult wraps argument-binding failures as deterministic tool errors.
func invalidRequestToolResult(err error) *mcp.CallToolResult {
	if err == nil {
		return mcp.NewToolResultError("invalid_request: malformed arguments")
	}
	return mcp.NewToolResultError("invalid_request: " + err.Error())
}
