package mcpapi

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/hylla/tillsyn/internal/adapters/server/common"
	"github.com/hylla/tillsyn/internal/app"
	"github.com/hylla/tillsyn/internal/domain"
	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

const (
	mcpMutationSessionDescription       = "Authenticated MCP session identifier"
	mcpMutationSessionSecretDescription = "Authenticated MCP session secret"
)

// mcpSessionAuthArgs stores the session-secret pair required for mutating MCP calls.
type mcpSessionAuthArgs struct {
	SessionID     string `json:"session_id"`
	SessionSecret string `json:"session_secret"`
}

// mcpMutationGuardArgs stores the secondary local lease tuple used after session auth succeeds.
type mcpMutationGuardArgs struct {
	AgentInstanceID string `json:"agent_instance_id"`
	LeaseToken      string `json:"lease_token"`
	OverrideToken   string `json:"override_token"`
}

// authorizeMCPMutation validates the caller session for one mutating tool.
func authorizeMCPMutation(
	ctx context.Context,
	authorizer common.MutationAuthorizer,
	auth mcpSessionAuthArgs,
	action string,
	namespace string,
	resourceType string,
	resourceID string,
	authContext map[string]string,
) (domain.AuthenticatedCaller, error) {
	if authorizer == nil {
		return domain.AuthenticatedCaller{}, fmt.Errorf("mutation authorizer is unavailable")
	}
	return authorizer.AuthorizeMutation(ctx, common.MutationAuthorizationRequest{
		SessionID:     strings.TrimSpace(auth.SessionID),
		SessionSecret: strings.TrimSpace(auth.SessionSecret),
		Action:        strings.TrimSpace(action),
		Namespace:     strings.TrimSpace(namespace),
		ResourceType:  strings.TrimSpace(resourceType),
		ResourceID:    strings.TrimSpace(resourceID),
		Context:       authContext,
	})
}

// buildAuthenticatedMutationActor converts one authenticated caller plus optional guard tuple into the app adapter actor contract.
func buildAuthenticatedMutationActor(caller domain.AuthenticatedCaller, guard mcpMutationGuardArgs) (common.ActorLeaseTuple, error) {
	caller = domain.NormalizeAuthenticatedCaller(caller)
	if caller.IsZero() {
		return common.ActorLeaseTuple{}, fmt.Errorf("invalid_request: authenticated caller is required for mutating MCP tools")
	}
	actor := common.ActorLeaseTuple{
		ActorID:   caller.PrincipalID,
		ActorName: caller.PrincipalName,
		ActorType: string(caller.PrincipalType),
	}

	guard.AgentInstanceID = strings.TrimSpace(guard.AgentInstanceID)
	guard.LeaseToken = strings.TrimSpace(guard.LeaseToken)
	guard.OverrideToken = strings.TrimSpace(guard.OverrideToken)
	hasGuardTuple := guard.AgentInstanceID != "" || guard.LeaseToken != "" || guard.OverrideToken != ""

	if caller.PrincipalType != domain.ActorTypeAgent {
		if hasGuardTuple {
			return common.ActorLeaseTuple{}, fmt.Errorf("invalid_request: guarded mutation tuple requires an authenticated agent session")
		}
		return actor, nil
	}
	if guard.AgentInstanceID == "" || guard.LeaseToken == "" {
		return common.ActorLeaseTuple{}, fmt.Errorf("invalid_request: agent_name, agent_instance_id, and lease_token are required for authenticated agent mutations")
	}

	// Lease identity must stay tied to the stable principal id; display name remains
	// available separately through ActorName for audit-friendly attribution.
	actor.AgentName = firstNonEmptyString(caller.PrincipalID, caller.PrincipalName)
	actor.AgentInstanceID = guard.AgentInstanceID
	actor.LeaseToken = guard.LeaseToken
	actor.OverrideToken = guard.OverrideToken
	return actor, nil
}

// firstNonEmptyString returns the first non-empty trimmed string in order.
func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

// buildProjectRootedMutationAuthScope normalizes project/global admin mutations onto one rooted project scope.
func buildProjectRootedMutationAuthScope(projectID string, authContext map[string]string) (string, map[string]string) {
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		projectID = domain.AuthRequestGlobalProjectID
	}
	normalized := make(map[string]string, len(authContext)+3)
	for key, value := range authContext {
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if key == "" || value == "" {
			continue
		}
		normalized[key] = value
	}
	normalized["project_id"] = projectID
	normalized["scope_type"] = string(domain.ScopeLevelProject)
	normalized["scope_id"] = projectID
	return "project:" + projectID, normalized
}

// capabilityLeaseMutationArgs stores the shared mutation payload for lease lifecycle operations.
type capabilityLeaseMutationArgs struct {
	Operation                 string `json:"operation"`
	ProjectID                 string `json:"project_id"`
	ScopeType                 string `json:"scope_type"`
	ScopeID                   string `json:"scope_id"`
	Role                      string `json:"role"`
	AgentName                 string `json:"agent_name"`
	AgentInstanceID           string `json:"agent_instance_id"`
	ParentInstanceID          string `json:"parent_instance_id"`
	AllowEqualScopeDelegation bool   `json:"allow_equal_scope_delegation"`
	RequestedTTLSeconds       int    `json:"requested_ttl_seconds"`
	OverrideToken             string `json:"override_token"`
	LeaseToken                string `json:"lease_token"`
	TTLSeconds                int    `json:"ttl_seconds"`
	Reason                    string `json:"reason"`
	SessionID                 string `json:"session_id"`
	SessionSecret             string `json:"session_secret"`
}

// handleCapabilityLeaseMutation routes one lease lifecycle operation through the shared tool surface.
func handleCapabilityLeaseMutation(
	ctx context.Context,
	leases common.CapabilityLeaseService,
	args capabilityLeaseMutationArgs,
) (*mcp.CallToolResult, error) {
	projectID := strings.TrimSpace(args.ProjectID)
	scopeType := strings.TrimSpace(args.ScopeType)
	scopeID := strings.TrimSpace(args.ScopeID)
	role := strings.TrimSpace(args.Role)
	agentName := strings.TrimSpace(args.AgentName)
	instanceID := strings.TrimSpace(args.AgentInstanceID)
	leaseToken := strings.TrimSpace(args.LeaseToken)
	reason := strings.TrimSpace(args.Reason)
	operation := strings.TrimSpace(args.Operation)

	switch operation {
	case "issue":
		if projectID == "" {
			return mcp.NewToolResultError(`invalid_request: required argument "project_id" not found`), nil
		}
		if scopeType == "" {
			return mcp.NewToolResultError(`invalid_request: required argument "scope_type" not found`), nil
		}
		if role == "" {
			return mcp.NewToolResultError(`invalid_request: required argument "role" not found`), nil
		}
		if agentName == "" {
			return mcp.NewToolResultError(`invalid_request: required argument "agent_name" not found`), nil
		}
		caller, err := authorizeMCPMutation(
			ctx,
			pickMutationAuthorizer(leases),
			mcpSessionAuthArgs{SessionID: args.SessionID, SessionSecret: args.SessionSecret},
			"issue_capability_lease",
			"project:"+projectID,
			"capability_lease",
			firstNonEmptyString(scopeID, projectID),
			map[string]string{
				"project_id": projectID,
				"scope_type": scopeType,
				"scope_id":   scopeID,
				"role":       role,
			},
		)
		if err != nil {
			return toolResultFromError(err), nil
		}
		if caller.PrincipalType == domain.ActorTypeAgent {
			agentName = firstNonEmptyString(caller.PrincipalID, caller.PrincipalName)
		}
		lease, err := leases.IssueCapabilityLease(ctx, common.IssueCapabilityLeaseRequest{
			ProjectID:                 projectID,
			ScopeType:                 scopeType,
			ScopeID:                   args.ScopeID,
			Role:                      role,
			AgentName:                 agentName,
			AgentInstanceID:           args.AgentInstanceID,
			ParentInstanceID:          args.ParentInstanceID,
			AllowEqualScopeDelegation: args.AllowEqualScopeDelegation,
			RequestedTTLSeconds:       args.RequestedTTLSeconds,
			OverrideToken:             args.OverrideToken,
		})
		if err != nil {
			return toolResultFromError(err), nil
		}
		result, err := mcp.NewToolResultJSON(lease)
		if err != nil {
			return nil, fmt.Errorf("encode capability_lease issue result: %w", err)
		}
		return result, nil
	case "heartbeat":
		if instanceID == "" {
			return mcp.NewToolResultError(`invalid_request: required argument "agent_instance_id" not found`), nil
		}
		if leaseToken == "" {
			return mcp.NewToolResultError(`invalid_request: required argument "lease_token" not found`), nil
		}
		if _, err := authorizeMCPMutation(
			ctx,
			pickMutationAuthorizer(leases),
			mcpSessionAuthArgs{SessionID: args.SessionID, SessionSecret: args.SessionSecret},
			"heartbeat_capability_lease",
			"tillsyn",
			"capability_lease",
			instanceID,
			map[string]string{"agent_instance_id": instanceID},
		); err != nil {
			return toolResultFromError(err), nil
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
			return nil, fmt.Errorf("encode capability_lease heartbeat result: %w", err)
		}
		return result, nil
	case "renew":
		if instanceID == "" {
			return mcp.NewToolResultError(`invalid_request: required argument "agent_instance_id" not found`), nil
		}
		if leaseToken == "" {
			return mcp.NewToolResultError(`invalid_request: required argument "lease_token" not found`), nil
		}
		if _, err := authorizeMCPMutation(
			ctx,
			pickMutationAuthorizer(leases),
			mcpSessionAuthArgs{SessionID: args.SessionID, SessionSecret: args.SessionSecret},
			"renew_capability_lease",
			"tillsyn",
			"capability_lease",
			instanceID,
			map[string]string{"agent_instance_id": instanceID},
		); err != nil {
			return toolResultFromError(err), nil
		}
		lease, err := leases.RenewCapabilityLease(ctx, common.RenewCapabilityLeaseRequest{
			AgentInstanceID: instanceID,
			LeaseToken:      leaseToken,
			TTLSeconds:      args.TTLSeconds,
		})
		if err != nil {
			return toolResultFromError(err), nil
		}
		result, err := mcp.NewToolResultJSON(lease)
		if err != nil {
			return nil, fmt.Errorf("encode capability_lease renew result: %w", err)
		}
		return result, nil
	case "revoke":
		if instanceID == "" {
			return mcp.NewToolResultError(`invalid_request: required argument "agent_instance_id" not found`), nil
		}
		if _, err := authorizeMCPMutation(
			ctx,
			pickMutationAuthorizer(leases),
			mcpSessionAuthArgs{SessionID: args.SessionID, SessionSecret: args.SessionSecret},
			"revoke_capability_lease",
			"tillsyn",
			"capability_lease",
			instanceID,
			map[string]string{"agent_instance_id": instanceID},
		); err != nil {
			return toolResultFromError(err), nil
		}
		lease, err := leases.RevokeCapabilityLease(ctx, common.RevokeCapabilityLeaseRequest{
			AgentInstanceID: instanceID,
			Reason:          reason,
		})
		if err != nil {
			return toolResultFromError(err), nil
		}
		result, err := mcp.NewToolResultJSON(lease)
		if err != nil {
			return nil, fmt.Errorf("encode capability_lease revoke result: %w", err)
		}
		return result, nil
	case "revoke_all":
		if projectID == "" {
			return mcp.NewToolResultError(`invalid_request: required argument "project_id" not found`), nil
		}
		if scopeType == "" {
			return mcp.NewToolResultError(`invalid_request: required argument "scope_type" not found`), nil
		}
		if _, err := authorizeMCPMutation(
			ctx,
			pickMutationAuthorizer(leases),
			mcpSessionAuthArgs{SessionID: args.SessionID, SessionSecret: args.SessionSecret},
			"revoke_all_capability_leases",
			"project:"+projectID,
			"capability_lease",
			firstNonEmptyString(scopeID, projectID),
			map[string]string{"project_id": projectID, "scope_type": scopeType},
		); err != nil {
			return toolResultFromError(err), nil
		}
		if err := leases.RevokeAllCapabilityLeases(ctx, common.RevokeAllCapabilityLeasesRequest{
			ProjectID: projectID,
			ScopeType: scopeType,
			ScopeID:   args.ScopeID,
			Reason:    args.Reason,
		}); err != nil {
			return toolResultFromError(err), nil
		}
		result, err := mcp.NewToolResultJSON(map[string]any{
			"updated":    true,
			"project_id": projectID,
			"scope_type": scopeType,
			"scope_id":   args.ScopeID,
		})
		if err != nil {
			return nil, fmt.Errorf("encode capability_lease revoke_all result: %w", err)
		}
		return result, nil
	default:
		return mcp.NewToolResultError(`invalid_request: required argument "operation" not found`), nil
	}
}

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

// registerProjectTools registers list/read project tools plus the reduced project mutation family.
func registerProjectTools(
	srv *mcpserver.MCPServer,
	projects common.ProjectService,
	kinds common.KindCatalogService,
	templates common.TemplateLibraryService,
	exposeLegacyProjectTools bool,
) {
	if projects == nil && kinds == nil && templates == nil {
		return
	}

	if projects != nil {
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
	}

	srv.AddTool(
		mcp.NewTool(
			"till.project",
			mcp.WithDescription("Mutate one project-root operation. Use operation=create|update|bind_template|set_allowed_kinds."),
			mcp.WithString("operation", mcp.Required(), mcp.Description("Project mutation operation"), mcp.Enum("create", "update", "bind_template", "set_allowed_kinds")),
			mcp.WithString("project_id", mcp.Description("Project identifier. Required for operation=update|bind_template|set_allowed_kinds")),
			mcp.WithString("name", mcp.Description("Project name. Required for operation=create|update")),
			mcp.WithString("description", mcp.Description("Project details in markdown-rich text")),
			mcp.WithString("kind", mcp.Description("Project kind id")),
			mcp.WithString("template_library_id", mcp.Description("Template library identifier. Used by operation=create or bind_template")),
			mcp.WithArray("kind_ids", mcp.Description("Allowed kind id list for operation=set_allowed_kinds"), mcp.WithStringItems()),
			mcp.WithObject("metadata", mcp.Description("Optional project metadata object")),
			mcp.WithString("session_id", mcp.Required(), mcp.Description(mcpMutationSessionDescription)),
			mcp.WithString("session_secret", mcp.Required(), mcp.Description(mcpMutationSessionSecretDescription)),
			mcp.WithString("agent_instance_id", mcp.Description("Optional agent lease instance id for secondary local guard checks")),
			mcp.WithString("lease_token", mcp.Description("Optional agent lease token for secondary local guard checks")),
			mcp.WithString("override_token", mcp.Description("Optional override token")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var args struct {
				Operation         string                 `json:"operation"`
				ProjectID         string                 `json:"project_id"`
				Name              string                 `json:"name"`
				Description       string                 `json:"description"`
				Kind              string                 `json:"kind"`
				TemplateLibraryID string                 `json:"template_library_id"`
				KindIDs           []string               `json:"kind_ids"`
				Metadata          domain.ProjectMetadata `json:"metadata"`
				SessionID         string                 `json:"session_id"`
				SessionSecret     string                 `json:"session_secret"`
				AgentInstanceID   string                 `json:"agent_instance_id"`
				LeaseToken        string                 `json:"lease_token"`
				OverrideToken     string                 `json:"override_token"`
			}
			if err := req.BindArguments(&args); err != nil {
				return invalidRequestToolResult(err), nil
			}
			switch strings.TrimSpace(args.Operation) {
			case "create":
				if projects == nil {
					return mcp.NewToolResultError("invalid_request: project service is unavailable"), nil
				}
				if strings.TrimSpace(args.Name) == "" {
					return mcp.NewToolResultError(`invalid_request: required argument "name" not found`), nil
				}
				namespace, authContext := buildProjectRootedMutationAuthScope("", map[string]string{
					"name": strings.TrimSpace(args.Name),
				})
				caller, err := authorizeMCPMutation(
					ctx,
					pickMutationAuthorizer(projects),
					mcpSessionAuthArgs{
						SessionID:     args.SessionID,
						SessionSecret: args.SessionSecret,
					},
					"create_project",
					namespace,
					"project",
					"new",
					authContext,
				)
				if err != nil {
					return toolResultFromError(err), nil
				}
				actor, err := buildAuthenticatedMutationActor(caller, mcpMutationGuardArgs{
					AgentInstanceID: args.AgentInstanceID,
					LeaseToken:      args.LeaseToken,
					OverrideToken:   args.OverrideToken,
				})
				if err != nil {
					return mcp.NewToolResultError(err.Error()), nil
				}
				project, err := projects.CreateProject(ctx, common.CreateProjectRequest{
					Name:              args.Name,
					Description:       args.Description,
					Kind:              args.Kind,
					TemplateLibraryID: args.TemplateLibraryID,
					Metadata:          args.Metadata,
					Actor:             actor,
				})
				if err != nil {
					return toolResultFromError(err), nil
				}
				result, err := mcp.NewToolResultJSON(project)
				if err != nil {
					return nil, fmt.Errorf("encode project create result: %w", err)
				}
				return result, nil
			case "update":
				if projects == nil {
					return mcp.NewToolResultError("invalid_request: project service is unavailable"), nil
				}
				if strings.TrimSpace(args.ProjectID) == "" {
					return mcp.NewToolResultError(`invalid_request: required argument "project_id" not found`), nil
				}
				if strings.TrimSpace(args.Name) == "" {
					return mcp.NewToolResultError(`invalid_request: required argument "name" not found`), nil
				}
				caller, err := authorizeMCPMutation(
					ctx,
					pickMutationAuthorizer(projects),
					mcpSessionAuthArgs{
						SessionID:     args.SessionID,
						SessionSecret: args.SessionSecret,
					},
					"update_project",
					"tillsyn",
					"project",
					args.ProjectID,
					map[string]string{"project_id": strings.TrimSpace(args.ProjectID)},
				)
				if err != nil {
					return toolResultFromError(err), nil
				}
				actor, err := buildAuthenticatedMutationActor(caller, mcpMutationGuardArgs{
					AgentInstanceID: args.AgentInstanceID,
					LeaseToken:      args.LeaseToken,
					OverrideToken:   args.OverrideToken,
				})
				if err != nil {
					return mcp.NewToolResultError(err.Error()), nil
				}
				project, err := projects.UpdateProject(ctx, common.UpdateProjectRequest{
					ProjectID:   args.ProjectID,
					Name:        args.Name,
					Description: args.Description,
					Kind:        args.Kind,
					Metadata:    args.Metadata,
					Actor:       actor,
				})
				if err != nil {
					return toolResultFromError(err), nil
				}
				result, err := mcp.NewToolResultJSON(project)
				if err != nil {
					return nil, fmt.Errorf("encode project update result: %w", err)
				}
				return result, nil
			case "bind_template":
				if templates == nil {
					return mcp.NewToolResultError("invalid_request: template library service is unavailable"), nil
				}
				projectID := strings.TrimSpace(args.ProjectID)
				if projectID == "" {
					return mcp.NewToolResultError(`invalid_request: required argument "project_id" not found`), nil
				}
				libraryID := strings.TrimSpace(args.TemplateLibraryID)
				if libraryID == "" {
					return mcp.NewToolResultError(`invalid_request: required argument "template_library_id" not found`), nil
				}
				caller, err := authorizeMCPMutation(
					ctx,
					pickMutationAuthorizer(templates),
					mcpSessionAuthArgs{
						SessionID:     args.SessionID,
						SessionSecret: args.SessionSecret,
					},
					"bind_project_template_library",
					"tillsyn",
					"project",
					projectID,
					map[string]string{
						"project_id": projectID,
						"library_id": libraryID,
					},
				)
				if err != nil {
					return toolResultFromError(err), nil
				}
				ctx = app.WithAuthenticatedCaller(ctx, caller)
				binding, err := templates.BindProjectTemplateLibrary(ctx, common.BindProjectTemplateLibraryRequest{
					ProjectID: projectID,
					LibraryID: libraryID,
				})
				if err != nil {
					return toolResultFromError(err), nil
				}
				result, err := mcp.NewToolResultJSON(binding)
				if err != nil {
					return nil, fmt.Errorf("encode project bind_template result: %w", err)
				}
				return result, nil
			case "set_allowed_kinds":
				if kinds == nil {
					return mcp.NewToolResultError("invalid_request: kind catalog service is unavailable"), nil
				}
				projectID := strings.TrimSpace(args.ProjectID)
				if projectID == "" {
					return mcp.NewToolResultError(`invalid_request: required argument "project_id" not found`), nil
				}
				if len(args.KindIDs) == 0 {
					return mcp.NewToolResultError(`invalid_request: required argument "kind_ids" not found`), nil
				}
				if _, err := authorizeMCPMutation(
					ctx,
					pickMutationAuthorizer(kinds),
					mcpSessionAuthArgs{
						SessionID:     args.SessionID,
						SessionSecret: args.SessionSecret,
					},
					"set_project_allowed_kinds",
					"tillsyn",
					"project",
					projectID,
					map[string]string{"project_id": projectID},
				); err != nil {
					return toolResultFromError(err), nil
				}
				if err := kinds.SetProjectAllowedKinds(ctx, common.SetProjectAllowedKindsRequest{
					ProjectID: projectID,
					KindIDs:   append([]string(nil), args.KindIDs...),
				}); err != nil {
					return toolResultFromError(err), nil
				}
				result, err := mcp.NewToolResultJSON(map[string]any{
					"updated":    true,
					"project_id": projectID,
					"kind_ids":   args.KindIDs,
				})
				if err != nil {
					return nil, fmt.Errorf("encode project set_allowed_kinds result: %w", err)
				}
				return result, nil
			default:
				return mcp.NewToolResultError(`invalid_request: required argument "operation" not found`), nil
			}
		},
	)

	if !exposeLegacyProjectTools {
		return
	}

	if projects != nil {
		srv.AddTool(
			mcp.NewTool(
				"till.create_project",
				mcp.WithDescription("Create one project."),
				mcp.WithString("name", mcp.Required(), mcp.Description("Project name")),
				mcp.WithString("description", mcp.Description("Project details in markdown-rich text")),
				mcp.WithString("kind", mcp.Description("Project kind id")),
				mcp.WithString("template_library_id", mcp.Description("Optional approved global template library id to bind during project creation")),
				mcp.WithObject("metadata", mcp.Description("Optional project metadata object")),
				mcp.WithString("session_id", mcp.Required(), mcp.Description(mcpMutationSessionDescription)),
				mcp.WithString("session_secret", mcp.Required(), mcp.Description(mcpMutationSessionSecretDescription)),
				mcp.WithString("agent_instance_id", mcp.Description("Optional agent lease instance id for secondary local guard checks")),
				mcp.WithString("lease_token", mcp.Description("Optional agent lease token for secondary local guard checks")),
				mcp.WithString("override_token", mcp.Description("Optional override token")),
			),
			func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
				var args struct {
					Name              string                 `json:"name"`
					Description       string                 `json:"description"`
					Kind              string                 `json:"kind"`
					TemplateLibraryID string                 `json:"template_library_id"`
					Metadata          domain.ProjectMetadata `json:"metadata"`
					SessionID         string                 `json:"session_id"`
					SessionSecret     string                 `json:"session_secret"`
					AgentInstanceID   string                 `json:"agent_instance_id"`
					LeaseToken        string                 `json:"lease_token"`
					OverrideToken     string                 `json:"override_token"`
				}
				if err := req.BindArguments(&args); err != nil {
					return invalidRequestToolResult(err), nil
				}
				if strings.TrimSpace(args.Name) == "" {
					return mcp.NewToolResultError(`invalid_request: required argument "name" not found`), nil
				}
				namespace, authContext := buildProjectRootedMutationAuthScope("", map[string]string{
					"name": strings.TrimSpace(args.Name),
				})
				caller, err := authorizeMCPMutation(
					ctx,
					pickMutationAuthorizer(projects),
					mcpSessionAuthArgs{
						SessionID:     args.SessionID,
						SessionSecret: args.SessionSecret,
					},
					"create_project",
					namespace,
					"project",
					"new",
					authContext,
				)
				if err != nil {
					return toolResultFromError(err), nil
				}
				actor, err := buildAuthenticatedMutationActor(caller, mcpMutationGuardArgs{
					AgentInstanceID: args.AgentInstanceID,
					LeaseToken:      args.LeaseToken,
					OverrideToken:   args.OverrideToken,
				})
				if err != nil {
					return mcp.NewToolResultError(err.Error()), nil
				}
				project, err := projects.CreateProject(ctx, common.CreateProjectRequest{
					Name:              args.Name,
					Description:       args.Description,
					Kind:              args.Kind,
					TemplateLibraryID: args.TemplateLibraryID,
					Metadata:          args.Metadata,
					Actor:             actor,
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
				mcp.WithString("description", mcp.Description("Project details in markdown-rich text")),
				mcp.WithString("kind", mcp.Description("Project kind id")),
				mcp.WithObject("metadata", mcp.Description("Optional project metadata object")),
				mcp.WithString("session_id", mcp.Required(), mcp.Description(mcpMutationSessionDescription)),
				mcp.WithString("session_secret", mcp.Required(), mcp.Description(mcpMutationSessionSecretDescription)),
				mcp.WithString("agent_instance_id", mcp.Description("Optional agent lease instance id for secondary local guard checks")),
				mcp.WithString("lease_token", mcp.Description("Optional agent lease token for secondary local guard checks")),
				mcp.WithString("override_token", mcp.Description("Optional override token")),
			),
			func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
				var args struct {
					ProjectID       string                 `json:"project_id"`
					Name            string                 `json:"name"`
					Description     string                 `json:"description"`
					Kind            string                 `json:"kind"`
					Metadata        domain.ProjectMetadata `json:"metadata"`
					SessionID       string                 `json:"session_id"`
					SessionSecret   string                 `json:"session_secret"`
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
				caller, err := authorizeMCPMutation(
					ctx,
					pickMutationAuthorizer(projects),
					mcpSessionAuthArgs{
						SessionID:     args.SessionID,
						SessionSecret: args.SessionSecret,
					},
					"update_project",
					"tillsyn",
					"project",
					args.ProjectID,
					map[string]string{"project_id": strings.TrimSpace(args.ProjectID)},
				)
				if err != nil {
					return toolResultFromError(err), nil
				}
				actor, err := buildAuthenticatedMutationActor(caller, mcpMutationGuardArgs{
					AgentInstanceID: args.AgentInstanceID,
					LeaseToken:      args.LeaseToken,
					OverrideToken:   args.OverrideToken,
				})
				if err != nil {
					return mcp.NewToolResultError(err.Error()), nil
				}
				project, err := projects.UpdateProject(ctx, common.UpdateProjectRequest{
					ProjectID:   args.ProjectID,
					Name:        args.Name,
					Description: args.Description,
					Kind:        args.Kind,
					Metadata:    args.Metadata,
					Actor:       actor,
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
}

// registerTaskTools registers list/search/create/update/mutation task tools.
func registerTaskTools(
	srv *mcpserver.MCPServer,
	tasks common.TaskService,
	search common.SearchService,
	embeddings common.EmbeddingsService,
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
				mcp.WithDescription("Create one task/work-item (branch|phase|task|subtask via scope/kind)."),
				mcp.WithString("project_id", mcp.Required(), mcp.Description("Project identifier")),
				mcp.WithString("column_id", mcp.Required(), mcp.Description("Column identifier")),
				mcp.WithString("title", mcp.Required(), mcp.Description("Task title")),
				mcp.WithString("parent_id", mcp.Description("Optional parent task id")),
				mcp.WithString("kind", mcp.Description("Kind identifier")),
				mcp.WithString("scope", mcp.Description("project|branch|phase|task|subtask"), mcp.Enum(common.SupportedScopeTypes()...)),
				mcp.WithString("description", mcp.Description("Task details in markdown-rich text")),
				mcp.WithString("priority", mcp.Description("low|medium|high"), mcp.Enum("low", "medium", "high")),
				mcp.WithString("due_at", mcp.Description("Optional RFC3339 timestamp")),
				mcp.WithArray("labels", mcp.Description("Optional labels"), mcp.WithStringItems()),
				mcp.WithObject("metadata", mcp.Description("Optional task metadata object")),
				mcp.WithString("session_id", mcp.Required(), mcp.Description(mcpMutationSessionDescription)),
				mcp.WithString("session_secret", mcp.Required(), mcp.Description(mcpMutationSessionSecretDescription)),
				mcp.WithString("agent_instance_id", mcp.Description("Optional agent lease instance id for secondary local guard checks")),
				mcp.WithString("lease_token", mcp.Description("Optional agent lease token for secondary local guard checks")),
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
					SessionID       string              `json:"session_id"`
					SessionSecret   string              `json:"session_secret"`
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
				caller, err := authorizeMCPMutation(
					ctx,
					pickMutationAuthorizer(tasks),
					mcpSessionAuthArgs{
						SessionID:     args.SessionID,
						SessionSecret: args.SessionSecret,
					},
					"create_task",
					"project:"+strings.TrimSpace(args.ProjectID),
					"task",
					"new",
					map[string]string{
						"project_id": strings.TrimSpace(args.ProjectID),
						"parent_id":  strings.TrimSpace(args.ParentID),
						"column_id":  strings.TrimSpace(args.ColumnID),
						"scope":      strings.TrimSpace(args.Scope),
					},
				)
				if err != nil {
					return toolResultFromError(err), nil
				}
				actor, err := buildAuthenticatedMutationActor(caller, mcpMutationGuardArgs{
					AgentInstanceID: args.AgentInstanceID,
					LeaseToken:      args.LeaseToken,
					OverrideToken:   args.OverrideToken,
				})
				if err != nil {
					return mcp.NewToolResultError(err.Error()), nil
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
					Actor:       actor,
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
				mcp.WithString("description", mcp.Description("Task details in markdown-rich text")),
				mcp.WithString("priority", mcp.Description("low|medium|high"), mcp.Enum("low", "medium", "high")),
				mcp.WithString("due_at", mcp.Description("Optional RFC3339 timestamp")),
				mcp.WithArray("labels", mcp.Description("Optional labels"), mcp.WithStringItems()),
				mcp.WithObject("metadata", mcp.Description("Optional task metadata object")),
				mcp.WithString("session_id", mcp.Required(), mcp.Description(mcpMutationSessionDescription)),
				mcp.WithString("session_secret", mcp.Required(), mcp.Description(mcpMutationSessionSecretDescription)),
				mcp.WithString("agent_instance_id", mcp.Description("Optional agent lease instance id for secondary local guard checks")),
				mcp.WithString("lease_token", mcp.Description("Optional agent lease token for secondary local guard checks")),
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
					SessionID       string               `json:"session_id"`
					SessionSecret   string               `json:"session_secret"`
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
				caller, err := authorizeMCPMutation(
					ctx,
					pickMutationAuthorizer(tasks),
					mcpSessionAuthArgs{
						SessionID:     args.SessionID,
						SessionSecret: args.SessionSecret,
					},
					"update_task",
					"tillsyn",
					"task",
					strings.TrimSpace(args.TaskID),
					map[string]string{"task_id": strings.TrimSpace(args.TaskID)},
				)
				if err != nil {
					return toolResultFromError(err), nil
				}
				actor, err := buildAuthenticatedMutationActor(caller, mcpMutationGuardArgs{
					AgentInstanceID: args.AgentInstanceID,
					LeaseToken:      args.LeaseToken,
					OverrideToken:   args.OverrideToken,
				})
				if err != nil {
					return mcp.NewToolResultError(err.Error()), nil
				}
				task, err := tasks.UpdateTask(ctx, common.UpdateTaskRequest{
					TaskID:      args.TaskID,
					Title:       args.Title,
					Description: args.Description,
					Priority:    args.Priority,
					DueAt:       args.DueAt,
					Labels:      append([]string(nil), args.Labels...),
					Metadata:    args.Metadata,
					Actor:       actor,
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
				mcp.WithString("session_id", mcp.Required(), mcp.Description(mcpMutationSessionDescription)),
				mcp.WithString("session_secret", mcp.Required(), mcp.Description(mcpMutationSessionSecretDescription)),
				mcp.WithString("agent_instance_id", mcp.Description("Optional agent lease instance id for secondary local guard checks")),
				mcp.WithString("lease_token", mcp.Description("Optional agent lease token for secondary local guard checks")),
				mcp.WithString("override_token", mcp.Description("Optional override token")),
			),
			func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
				var args struct {
					TaskID          string `json:"task_id"`
					ToColumnID      string `json:"to_column_id"`
					Position        int    `json:"position"`
					SessionID       string `json:"session_id"`
					SessionSecret   string `json:"session_secret"`
					AgentInstanceID string `json:"agent_instance_id"`
					LeaseToken      string `json:"lease_token"`
					OverrideToken   string `json:"override_token"`
				}
				if err := req.BindArguments(&args); err != nil {
					return invalidRequestToolResult(err), nil
				}
				taskID := strings.TrimSpace(args.TaskID)
				if taskID == "" {
					return mcp.NewToolResultError(`invalid_request: required argument "task_id" not found`), nil
				}
				toColumnID := strings.TrimSpace(args.ToColumnID)
				if toColumnID == "" {
					return mcp.NewToolResultError(`invalid_request: required argument "to_column_id" not found`), nil
				}
				caller, err := authorizeMCPMutation(
					ctx,
					pickMutationAuthorizer(tasks),
					mcpSessionAuthArgs{
						SessionID:     args.SessionID,
						SessionSecret: args.SessionSecret,
					},
					"move_task",
					"tillsyn",
					"task",
					taskID,
					map[string]string{"task_id": taskID, "to_column_id": toColumnID},
				)
				if err != nil {
					return toolResultFromError(err), nil
				}
				actor, err := buildAuthenticatedMutationActor(caller, mcpMutationGuardArgs{
					AgentInstanceID: args.AgentInstanceID,
					LeaseToken:      args.LeaseToken,
					OverrideToken:   args.OverrideToken,
				})
				if err != nil {
					return mcp.NewToolResultError(err.Error()), nil
				}
				task, err := tasks.MoveTask(ctx, common.MoveTaskRequest{
					TaskID:     taskID,
					ToColumnID: toColumnID,
					Position:   args.Position,
					Actor:      actor,
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
				mcp.WithString("session_id", mcp.Required(), mcp.Description(mcpMutationSessionDescription)),
				mcp.WithString("session_secret", mcp.Required(), mcp.Description(mcpMutationSessionSecretDescription)),
				mcp.WithString("agent_instance_id", mcp.Description("Optional agent lease instance id for secondary local guard checks")),
				mcp.WithString("lease_token", mcp.Description("Optional agent lease token for secondary local guard checks")),
				mcp.WithString("override_token", mcp.Description("Optional override token")),
			),
			func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
				var args struct {
					TaskID          string `json:"task_id"`
					Mode            string `json:"mode"`
					SessionID       string `json:"session_id"`
					SessionSecret   string `json:"session_secret"`
					AgentInstanceID string `json:"agent_instance_id"`
					LeaseToken      string `json:"lease_token"`
					OverrideToken   string `json:"override_token"`
				}
				if err := req.BindArguments(&args); err != nil {
					return invalidRequestToolResult(err), nil
				}
				taskID := strings.TrimSpace(args.TaskID)
				if taskID == "" {
					return mcp.NewToolResultError(`invalid_request: required argument "task_id" not found`), nil
				}
				caller, err := authorizeMCPMutation(
					ctx,
					pickMutationAuthorizer(tasks),
					mcpSessionAuthArgs{
						SessionID:     args.SessionID,
						SessionSecret: args.SessionSecret,
					},
					"delete_task",
					"tillsyn",
					"task",
					taskID,
					map[string]string{"task_id": taskID, "mode": strings.TrimSpace(args.Mode)},
				)
				if err != nil {
					return toolResultFromError(err), nil
				}
				actor, err := buildAuthenticatedMutationActor(caller, mcpMutationGuardArgs{
					AgentInstanceID: args.AgentInstanceID,
					LeaseToken:      args.LeaseToken,
					OverrideToken:   args.OverrideToken,
				})
				if err != nil {
					return mcp.NewToolResultError(err.Error()), nil
				}
				if err := tasks.DeleteTask(ctx, common.DeleteTaskRequest{
					TaskID: taskID,
					Mode:   args.Mode,
					Actor:  actor,
				}); err != nil {
					return toolResultFromError(err), nil
				}
				result, err := mcp.NewToolResultJSON(map[string]any{
					"deleted": true,
					"task_id": taskID,
					"mode":    args.Mode,
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
				mcp.WithString("session_id", mcp.Required(), mcp.Description(mcpMutationSessionDescription)),
				mcp.WithString("session_secret", mcp.Required(), mcp.Description(mcpMutationSessionSecretDescription)),
				mcp.WithString("agent_instance_id", mcp.Description("Optional agent lease instance id for secondary local guard checks")),
				mcp.WithString("lease_token", mcp.Description("Optional agent lease token for secondary local guard checks")),
				mcp.WithString("override_token", mcp.Description("Optional override token")),
			),
			func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
				var args struct {
					TaskID          string `json:"task_id"`
					SessionID       string `json:"session_id"`
					SessionSecret   string `json:"session_secret"`
					AgentInstanceID string `json:"agent_instance_id"`
					LeaseToken      string `json:"lease_token"`
					OverrideToken   string `json:"override_token"`
				}
				if err := req.BindArguments(&args); err != nil {
					return invalidRequestToolResult(err), nil
				}
				taskID := strings.TrimSpace(args.TaskID)
				if taskID == "" {
					return mcp.NewToolResultError(`invalid_request: required argument "task_id" not found`), nil
				}
				caller, err := authorizeMCPMutation(
					ctx,
					pickMutationAuthorizer(tasks),
					mcpSessionAuthArgs{
						SessionID:     args.SessionID,
						SessionSecret: args.SessionSecret,
					},
					"restore_task",
					"tillsyn",
					"task",
					taskID,
					map[string]string{"task_id": taskID},
				)
				if err != nil {
					return toolResultFromError(err), nil
				}
				actor, err := buildAuthenticatedMutationActor(caller, mcpMutationGuardArgs{
					AgentInstanceID: args.AgentInstanceID,
					LeaseToken:      args.LeaseToken,
					OverrideToken:   args.OverrideToken,
				})
				if err != nil {
					return mcp.NewToolResultError(err.Error()), nil
				}
				task, err := tasks.RestoreTask(ctx, common.RestoreTaskRequest{
					TaskID: taskID,
					Actor:  actor,
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
				mcp.WithString("session_id", mcp.Required(), mcp.Description(mcpMutationSessionDescription)),
				mcp.WithString("session_secret", mcp.Required(), mcp.Description(mcpMutationSessionSecretDescription)),
				mcp.WithString("agent_instance_id", mcp.Description("Optional agent lease instance id for secondary local guard checks")),
				mcp.WithString("lease_token", mcp.Description("Optional agent lease token for secondary local guard checks")),
				mcp.WithString("override_token", mcp.Description("Optional override token")),
			),
			func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
				var args struct {
					TaskID          string `json:"task_id"`
					ParentID        string `json:"parent_id"`
					SessionID       string `json:"session_id"`
					SessionSecret   string `json:"session_secret"`
					AgentInstanceID string `json:"agent_instance_id"`
					LeaseToken      string `json:"lease_token"`
					OverrideToken   string `json:"override_token"`
				}
				if err := req.BindArguments(&args); err != nil {
					return invalidRequestToolResult(err), nil
				}
				taskID := strings.TrimSpace(args.TaskID)
				if taskID == "" {
					return mcp.NewToolResultError(`invalid_request: required argument "task_id" not found`), nil
				}
				caller, err := authorizeMCPMutation(
					ctx,
					pickMutationAuthorizer(tasks),
					mcpSessionAuthArgs{
						SessionID:     args.SessionID,
						SessionSecret: args.SessionSecret,
					},
					"reparent_task",
					"tillsyn",
					"task",
					taskID,
					map[string]string{"task_id": taskID, "parent_id": strings.TrimSpace(args.ParentID)},
				)
				if err != nil {
					return toolResultFromError(err), nil
				}
				actor, err := buildAuthenticatedMutationActor(caller, mcpMutationGuardArgs{
					AgentInstanceID: args.AgentInstanceID,
					LeaseToken:      args.LeaseToken,
					OverrideToken:   args.OverrideToken,
				})
				if err != nil {
					return mcp.NewToolResultError(err.Error()), nil
				}
				task, err := tasks.ReparentTask(ctx, common.ReparentTaskRequest{
					TaskID:   taskID,
					ParentID: args.ParentID,
					Actor:    actor,
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
				mcp.WithDescription("Search task/work-item matches by query, mode, sort, filters, and scope."),
				mcp.WithString("project_id", mcp.Description("Project identifier for non-cross-project queries")),
				mcp.WithString("query", mcp.Description("Search query")),
				mcp.WithBoolean("cross_project", mcp.Description("Search across all projects")),
				mcp.WithBoolean("include_archived", mcp.Description("Include archived projects/items")),
				mcp.WithArray("states", mcp.Description("Optional state filter"), mcp.WithStringItems()),
				mcp.WithArray("levels", mcp.Description("Optional level/scope filter"), mcp.WithStringItems()),
				mcp.WithArray("kinds", mcp.Description("Optional kind filter"), mcp.WithStringItems()),
				mcp.WithArray("labels_any", mcp.Description("Optional labels-any filter (matches when any listed label is present)"), mcp.WithStringItems()),
				mcp.WithArray("labels_all", mcp.Description("Optional labels-all filter (matches only when all listed labels are present)"), mcp.WithStringItems()),
				mcp.WithString("mode", mcp.Description("keyword|semantic|hybrid (default hybrid; semantic/hybrid fall back to keyword when embeddings/vector search is unavailable)"), mcp.Enum("keyword", "semantic", "hybrid")),
				mcp.WithString("sort", mcp.Description("rank_desc|title_asc|created_at_desc|updated_at_desc (default rank_desc)"), mcp.Enum("rank_desc", "title_asc", "created_at_desc", "updated_at_desc")),
				mcp.WithNumber(
					"limit",
					mcp.Description("Optional maximum rows (default 50, max 200)"),
					mcp.DefaultNumber(50),
					mcp.Min(0),
					mcp.Max(200),
				),
				mcp.WithNumber(
					"offset",
					mcp.Description("Optional row offset (default 0, must be >= 0)"),
					mcp.DefaultNumber(0),
					mcp.Min(0),
				),
			),
			func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
				resultPayload, err := search.SearchTasks(ctx, common.SearchTasksRequest{
					ProjectID:       req.GetString("project_id", ""),
					Query:           req.GetString("query", ""),
					CrossProject:    req.GetBool("cross_project", false),
					IncludeArchived: req.GetBool("include_archived", false),
					States:          req.GetStringSlice("states", nil),
					Levels:          req.GetStringSlice("levels", nil),
					Kinds:           req.GetStringSlice("kinds", nil),
					LabelsAny:       req.GetStringSlice("labels_any", nil),
					LabelsAll:       req.GetStringSlice("labels_all", nil),
					Mode:            req.GetString("mode", ""),
					Sort:            req.GetString("sort", ""),
					Limit:           req.GetInt("limit", 0),
					Offset:          req.GetInt("offset", 0),
				})
				if err != nil {
					return toolResultFromError(err), nil
				}
				result, err := mcp.NewToolResultJSON(resultPayload)
				if err != nil {
					return nil, fmt.Errorf("encode search_task_matches result: %w", err)
				}
				return result, nil
			},
		)
	}

	if embeddings != nil {
		srv.AddTool(
			mcp.NewTool(
				"till.get_embeddings_status",
				mcp.WithDescription("Show embeddings lifecycle summary counts and per-subject status rows."),
				mcp.WithString("project_id", mcp.Description("Project identifier for non-cross-project inventory")),
				mcp.WithBoolean("cross_project", mcp.Description("Inspect embeddings lifecycle across all projects")),
				mcp.WithBoolean("include_archived", mcp.Description("Include archived projects when resolving cross-project scope")),
				mcp.WithArray("statuses", mcp.Description("Optional lifecycle status filter"), mcp.WithStringItems()),
				mcp.WithNumber(
					"limit",
					mcp.Description("Optional maximum status rows (default 100, max 500)"),
					mcp.DefaultNumber(100),
					mcp.Min(0),
					mcp.Max(500),
				),
			),
			func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
				resultPayload, err := embeddings.GetEmbeddingsStatus(ctx, common.EmbeddingsStatusRequest{
					ProjectID:       req.GetString("project_id", ""),
					CrossProject:    req.GetBool("cross_project", false),
					IncludeArchived: req.GetBool("include_archived", false),
					Statuses:        req.GetStringSlice("statuses", nil),
					Limit:           req.GetInt("limit", 0),
				})
				if err != nil {
					return toolResultFromError(err), nil
				}
				result, err := mcp.NewToolResultJSON(resultPayload)
				if err != nil {
					return nil, fmt.Errorf("encode get_embeddings_status result: %w", err)
				}
				return result, nil
			},
		)

		srv.AddTool(
			mcp.NewTool(
				"till.reindex_embeddings",
				mcp.WithDescription("Enqueue or force explicit embeddings backfill/reindex work and optionally wait for steady state."),
				mcp.WithString("project_id", mcp.Description("Project identifier for non-cross-project reindex requests")),
				mcp.WithBoolean("cross_project", mcp.Description("Reindex embeddings across all projects")),
				mcp.WithBoolean("include_archived", mcp.Description("Include archived projects and work items in the reindex scope")),
				mcp.WithBoolean("force", mcp.Description("Force ready rows back into the queue even when hashes already match")),
				mcp.WithBoolean("wait", mcp.Description("Wait for the requested scope to reach a steady lifecycle state")),
				mcp.WithString("wait_timeout", mcp.Description("Optional wait timeout duration (for example 30s or 2m)")),
				mcp.WithString("wait_poll_interval", mcp.Description("Optional wait polling interval duration")),
			),
			func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
				waitTimeout, err := optionalDurationArg(req.GetString("wait_timeout", ""))
				if err != nil {
					return mcp.NewToolResultError(err.Error()), nil
				}
				waitPollInterval, err := optionalDurationArg(req.GetString("wait_poll_interval", ""))
				if err != nil {
					return mcp.NewToolResultError(err.Error()), nil
				}
				resultPayload, err := embeddings.ReindexEmbeddings(ctx, common.ReindexEmbeddingsRequest{
					ProjectID:        req.GetString("project_id", ""),
					CrossProject:     req.GetBool("cross_project", false),
					IncludeArchived:  req.GetBool("include_archived", false),
					Force:            req.GetBool("force", false),
					Wait:             req.GetBool("wait", false),
					WaitTimeout:      waitTimeout,
					WaitPollInterval: waitPollInterval,
				})
				if err != nil {
					return toolResultFromError(err), nil
				}
				result, err := mcp.NewToolResultJSON(resultPayload)
				if err != nil {
					return nil, fmt.Errorf("encode reindex_embeddings result: %w", err)
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
func registerKindTools(srv *mcpserver.MCPServer, kinds common.KindCatalogService, exposeLegacyProjectTools bool) {
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
			mcp.WithString("session_id", mcp.Required(), mcp.Description(mcpMutationSessionDescription)),
			mcp.WithString("session_secret", mcp.Required(), mcp.Description(mcpMutationSessionSecretDescription)),
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
				SessionID           string              `json:"session_id"`
				SessionSecret       string              `json:"session_secret"`
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
			namespace, authContext := buildProjectRootedMutationAuthScope("", map[string]string{
				"kind_id": strings.TrimSpace(args.ID),
			})
			if _, err := authorizeMCPMutation(
				ctx,
				pickMutationAuthorizer(kinds),
				mcpSessionAuthArgs{
					SessionID:     args.SessionID,
					SessionSecret: args.SessionSecret,
				},
				"upsert_kind_definition",
				namespace,
				"kind_definition",
				strings.TrimSpace(args.ID),
				authContext,
			); err != nil {
				return toolResultFromError(err), nil
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

	if exposeLegacyProjectTools {
		srv.AddTool(
			mcp.NewTool(
				"till.set_project_allowed_kinds",
				mcp.WithDescription("Set explicit project allowed kind identifiers."),
				mcp.WithString("project_id", mcp.Required(), mcp.Description("Project identifier")),
				mcp.WithArray("kind_ids", mcp.Required(), mcp.Description("Allowed kind id list"), mcp.WithStringItems()),
				mcp.WithString("session_id", mcp.Required(), mcp.Description(mcpMutationSessionDescription)),
				mcp.WithString("session_secret", mcp.Required(), mcp.Description(mcpMutationSessionSecretDescription)),
			),
			func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
				var args struct {
					ProjectID     string   `json:"project_id"`
					KindIDs       []string `json:"kind_ids"`
					SessionID     string   `json:"session_id"`
					SessionSecret string   `json:"session_secret"`
				}
				if err := req.BindArguments(&args); err != nil {
					return invalidRequestToolResult(err), nil
				}
				projectID := strings.TrimSpace(args.ProjectID)
				if projectID == "" {
					return mcp.NewToolResultError(`invalid_request: required argument "project_id" not found`), nil
				}
				if len(args.KindIDs) == 0 {
					return mcp.NewToolResultError(`invalid_request: required argument "kind_ids" not found`), nil
				}
				if _, err := authorizeMCPMutation(
					ctx,
					pickMutationAuthorizer(kinds),
					mcpSessionAuthArgs{
						SessionID:     args.SessionID,
						SessionSecret: args.SessionSecret,
					},
					"set_project_allowed_kinds",
					"tillsyn",
					"project",
					projectID,
					map[string]string{"project_id": projectID},
				); err != nil {
					return toolResultFromError(err), nil
				}
				if err := kinds.SetProjectAllowedKinds(ctx, common.SetProjectAllowedKindsRequest{
					ProjectID: projectID,
					KindIDs:   append([]string(nil), args.KindIDs...),
				}); err != nil {
					return toolResultFromError(err), nil
				}
				result, err := mcp.NewToolResultJSON(map[string]any{
					"updated":    true,
					"project_id": projectID,
					"kind_ids":   args.KindIDs,
				})
				if err != nil {
					return nil, fmt.Errorf("encode set_project_allowed_kinds result: %w", err)
				}
				return result, nil
			},
		)
	}

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

// registerTemplateLibraryTools registers template-library and node-contract inspection/binding tools.
func registerTemplateLibraryTools(srv *mcpserver.MCPServer, templates common.TemplateLibraryService, exposeLegacyProjectTools bool) {
	if templates == nil {
		return
	}

	srv.AddTool(
		mcp.NewTool(
			"till.list_template_libraries",
			mcp.WithDescription("List template libraries. SQLite remains the source of truth, and JSON is the stable MCP transport format for these library records."),
			mcp.WithString("scope", mcp.Description("Optional template-library scope filter"), mcp.Enum("global", "project", "draft")),
			mcp.WithString("project_id", mcp.Description("Optional project identifier filter")),
			mcp.WithString("status", mcp.Description("Optional template-library status filter"), mcp.Enum("draft", "approved", "archived")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			rows, err := templates.ListTemplateLibraries(ctx, common.ListTemplateLibrariesRequest{
				Scope:     domain.TemplateLibraryScope(req.GetString("scope", "")),
				ProjectID: req.GetString("project_id", ""),
				Status:    domain.TemplateLibraryStatus(req.GetString("status", "")),
			})
			if err != nil {
				return toolResultFromError(err), nil
			}
			result, err := mcp.NewToolResultJSON(map[string]any{"libraries": rows})
			if err != nil {
				return nil, fmt.Errorf("encode list_template_libraries result: %w", err)
			}
			return result, nil
		},
	)

	srv.AddTool(
		mcp.NewTool(
			"till.get_template_library",
			mcp.WithDescription("Show one template library by id."),
			mcp.WithString("library_id", mcp.Required(), mcp.Description("Template library identifier")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			libraryID, err := req.RequireString("library_id")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			library, err := templates.GetTemplateLibrary(ctx, libraryID)
			if err != nil {
				return toolResultFromError(err), nil
			}
			result, err := mcp.NewToolResultJSON(library)
			if err != nil {
				return nil, fmt.Errorf("encode get_template_library result: %w", err)
			}
			return result, nil
		},
	)

	srv.AddTool(
		mcp.NewTool(
			"till.upsert_template_library",
			mcp.WithDescription("Create or update one template library via JSON transport. SQLite remains canonical, while the TUI is the primary human review and approval surface."),
			mcp.WithObject("library", mcp.Required(), mcp.Description("Template library object")),
			mcp.WithString("session_id", mcp.Required(), mcp.Description(mcpMutationSessionDescription)),
			mcp.WithString("session_secret", mcp.Required(), mcp.Description(mcpMutationSessionSecretDescription)),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var args struct {
				Library       common.UpsertTemplateLibraryRequest `json:"library"`
				SessionID     string                              `json:"session_id"`
				SessionSecret string                              `json:"session_secret"`
			}
			if err := req.BindArguments(&args); err != nil {
				return invalidRequestToolResult(err), nil
			}
			resourceID := strings.TrimSpace(args.Library.ID)
			if resourceID == "" {
				resourceID = "new"
			}
			namespace, authContext := buildProjectRootedMutationAuthScope(strings.TrimSpace(args.Library.ProjectID), map[string]string{
				"library_id": resourceID,
			})
			caller, err := authorizeMCPMutation(
				ctx,
				pickMutationAuthorizer(templates),
				mcpSessionAuthArgs{
					SessionID:     args.SessionID,
					SessionSecret: args.SessionSecret,
				},
				"upsert_template_library",
				namespace,
				"template_library",
				resourceID,
				authContext,
			)
			if err != nil {
				return toolResultFromError(err), nil
			}
			ctx = app.WithAuthenticatedCaller(ctx, caller)
			library, err := templates.UpsertTemplateLibrary(ctx, args.Library)
			if err != nil {
				return toolResultFromError(err), nil
			}
			result, err := mcp.NewToolResultJSON(library)
			if err != nil {
				return nil, fmt.Errorf("encode upsert_template_library result: %w", err)
			}
			return result, nil
		},
	)

	if exposeLegacyProjectTools {
		srv.AddTool(
			mcp.NewTool(
				"till.bind_project_template_library",
				mcp.WithDescription("Bind one project to one approved template library."),
				mcp.WithString("project_id", mcp.Required(), mcp.Description("Project identifier")),
				mcp.WithString("library_id", mcp.Required(), mcp.Description("Template library identifier")),
				mcp.WithString("session_id", mcp.Required(), mcp.Description(mcpMutationSessionDescription)),
				mcp.WithString("session_secret", mcp.Required(), mcp.Description(mcpMutationSessionSecretDescription)),
			),
			func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
				var args struct {
					ProjectID     string `json:"project_id"`
					LibraryID     string `json:"library_id"`
					SessionID     string `json:"session_id"`
					SessionSecret string `json:"session_secret"`
				}
				if err := req.BindArguments(&args); err != nil {
					return invalidRequestToolResult(err), nil
				}
				projectID := strings.TrimSpace(args.ProjectID)
				if projectID == "" {
					return mcp.NewToolResultError(`invalid_request: required argument "project_id" not found`), nil
				}
				libraryID := strings.TrimSpace(args.LibraryID)
				if libraryID == "" {
					return mcp.NewToolResultError(`invalid_request: required argument "library_id" not found`), nil
				}
				caller, err := authorizeMCPMutation(
					ctx,
					pickMutationAuthorizer(templates),
					mcpSessionAuthArgs{
						SessionID:     args.SessionID,
						SessionSecret: args.SessionSecret,
					},
					"bind_project_template_library",
					"tillsyn",
					"project",
					projectID,
					map[string]string{
						"project_id": projectID,
						"library_id": libraryID,
					},
				)
				if err != nil {
					return toolResultFromError(err), nil
				}
				ctx = app.WithAuthenticatedCaller(ctx, caller)
				binding, err := templates.BindProjectTemplateLibrary(ctx, common.BindProjectTemplateLibraryRequest{
					ProjectID: projectID,
					LibraryID: libraryID,
				})
				if err != nil {
					return toolResultFromError(err), nil
				}
				result, err := mcp.NewToolResultJSON(binding)
				if err != nil {
					return nil, fmt.Errorf("encode bind_project_template_library result: %w", err)
				}
				return result, nil
			},
		)
	}

	srv.AddTool(
		mcp.NewTool(
			"till.get_project_template_binding",
			mcp.WithDescription("Show the active template-library binding for one project."),
			mcp.WithString("project_id", mcp.Required(), mcp.Description("Project identifier")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			projectID, err := req.RequireString("project_id")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			binding, err := templates.GetProjectTemplateBinding(ctx, projectID)
			if err != nil {
				return toolResultFromError(err), nil
			}
			result, err := mcp.NewToolResultJSON(binding)
			if err != nil {
				return nil, fmt.Errorf("encode get_project_template_binding result: %w", err)
			}
			return result, nil
		},
	)

	srv.AddTool(
		mcp.NewTool(
			"till.get_node_contract_snapshot",
			mcp.WithDescription("Show one generated-node contract snapshot used for truthful completion and actor-kind enforcement."),
			mcp.WithString("node_id", mcp.Required(), mcp.Description("Generated node identifier")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			nodeID, err := req.RequireString("node_id")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			snapshot, err := templates.GetNodeContractSnapshot(ctx, nodeID)
			if err != nil {
				return toolResultFromError(err), nil
			}
			result, err := mcp.NewToolResultJSON(snapshot)
			if err != nil {
				return nil, fmt.Errorf("encode get_node_contract_snapshot result: %w", err)
			}
			return result, nil
		},
	)
}

// registerCapabilityLeaseTools registers lease visibility and lifecycle tools.
func registerCapabilityLeaseTools(srv *mcpserver.MCPServer, leases common.CapabilityLeaseService, exposeLegacyLeaseTools bool) {
	if leases == nil {
		return
	}

	srv.AddTool(
		mcp.NewTool(
			"till.list_capability_leases",
			mcp.WithDescription("List active or historical capability leases for one project scope."),
			mcp.WithString("project_id", mcp.Required(), mcp.Description("Project identifier")),
			mcp.WithString("scope_type", mcp.Description("Optional scope level filter"), mcp.Enum(common.SupportedScopeTypes()...)),
			mcp.WithString("scope_id", mcp.Description("Optional scope identifier; defaults to the project id for project scope")),
			mcp.WithBoolean("include_revoked", mcp.Description("Include revoked leases in addition to active leases")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			projectID, err := req.RequireString("project_id")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			leasesRows, err := leases.ListCapabilityLeases(ctx, common.ListCapabilityLeasesRequest{
				ProjectID:      projectID,
				ScopeType:      req.GetString("scope_type", ""),
				ScopeID:        req.GetString("scope_id", ""),
				IncludeRevoked: req.GetBool("include_revoked", false),
			})
			if err != nil {
				return toolResultFromError(err), nil
			}
			result, err := mcp.NewToolResultJSON(map[string]any{"leases": leasesRows})
			if err != nil {
				return nil, fmt.Errorf("encode list_capability_leases result: %w", err)
			}
			return result, nil
		},
	)

	srv.AddTool(
		mcp.NewTool(
			"till.capability_lease",
			mcp.WithDescription("Mutate one capability lease lifecycle. Use operation=issue|heartbeat|renew|revoke|revoke_all."),
			mcp.WithString("operation", mcp.Required(), mcp.Description("Lease mutation operation"), mcp.Enum("issue", "heartbeat", "renew", "revoke", "revoke_all")),
			mcp.WithString("project_id", mcp.Description("Project identifier. Required for operation=issue|revoke_all")),
			mcp.WithString("scope_type", mcp.Description("project|branch|phase|task|subtask. Required for operation=issue|revoke_all"), mcp.Enum(common.SupportedScopeTypes()...)),
			mcp.WithString("scope_id", mcp.Description("Scope identifier. Optional for project scope; otherwise used by operation=issue|revoke_all")),
			mcp.WithString("role", mcp.Description("orchestrator|builder|qa. Required for operation=issue"), mcp.Enum("orchestrator", "builder", "qa")),
			mcp.WithString("agent_name", mcp.Description("Agent display/name identifier. Required for operation=issue")),
			mcp.WithString("agent_instance_id", mcp.Description("Agent instance identifier. Required for operation=heartbeat|renew|revoke and optional for operation=issue")),
			mcp.WithString("parent_instance_id", mcp.Description("Optional parent lease instance id for operation=issue")),
			mcp.WithBoolean("allow_equal_scope_delegation", mcp.Description("Allow equal-scope delegation for operation=issue")),
			mcp.WithNumber("requested_ttl_seconds", mcp.Description("Optional TTL in seconds for operation=issue")),
			mcp.WithString("override_token", mcp.Description("Optional orchestrator overlap override token for operation=issue")),
			mcp.WithString("lease_token", mcp.Description("Lease token. Required for operation=heartbeat|renew")),
			mcp.WithNumber("ttl_seconds", mcp.Description("Optional renewal TTL in seconds for operation=renew")),
			mcp.WithString("reason", mcp.Description("Optional revocation reason for operation=revoke|revoke_all")),
			mcp.WithString("session_id", mcp.Required(), mcp.Description(mcpMutationSessionDescription)),
			mcp.WithString("session_secret", mcp.Required(), mcp.Description(mcpMutationSessionSecretDescription)),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var args capabilityLeaseMutationArgs
			if err := req.BindArguments(&args); err != nil {
				return invalidRequestToolResult(err), nil
			}
			return handleCapabilityLeaseMutation(ctx, leases, args)
		},
	)

	if !exposeLegacyLeaseTools {
		return
	}

	srv.AddTool(
		mcp.NewTool(
			"till.issue_capability_lease",
			mcp.WithDescription("Issue one capability lease."),
			mcp.WithString("project_id", mcp.Required(), mcp.Description("Project identifier")),
			mcp.WithString("scope_type", mcp.Required(), mcp.Description("project|branch|phase|task|subtask"), mcp.Enum(common.SupportedScopeTypes()...)),
			mcp.WithString("scope_id", mcp.Description("Scope identifier")),
			mcp.WithString("role", mcp.Required(), mcp.Description("orchestrator|builder|qa"), mcp.Enum("orchestrator", "builder", "qa")),
			mcp.WithString("agent_name", mcp.Required(), mcp.Description("Agent display/name identifier")),
			mcp.WithString("agent_instance_id", mcp.Description("Optional stable agent instance id")),
			mcp.WithString("parent_instance_id", mcp.Description("Optional parent lease instance id")),
			mcp.WithBoolean("allow_equal_scope_delegation", mcp.Description("Allow equal-scope delegation")),
			mcp.WithNumber("requested_ttl_seconds", mcp.Description("Optional TTL in seconds")),
			mcp.WithString("override_token", mcp.Description("Optional orchestrator overlap override token")),
			mcp.WithString("session_id", mcp.Required(), mcp.Description(mcpMutationSessionDescription)),
			mcp.WithString("session_secret", mcp.Required(), mcp.Description(mcpMutationSessionSecretDescription)),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var args struct {
				ProjectID                 string `json:"project_id"`
				ScopeType                 string `json:"scope_type"`
				ScopeID                   string `json:"scope_id"`
				Role                      string `json:"role"`
				AgentName                 string `json:"agent_name"`
				AgentInstanceID           string `json:"agent_instance_id"`
				ParentInstanceID          string `json:"parent_instance_id"`
				AllowEqualScopeDelegation bool   `json:"allow_equal_scope_delegation"`
				RequestedTTLSeconds       int    `json:"requested_ttl_seconds"`
				OverrideToken             string `json:"override_token"`
				SessionID                 string `json:"session_id"`
				SessionSecret             string `json:"session_secret"`
			}
			if err := req.BindArguments(&args); err != nil {
				return invalidRequestToolResult(err), nil
			}
			return handleCapabilityLeaseMutation(ctx, leases, capabilityLeaseMutationArgs{
				Operation:                 "issue",
				ProjectID:                 args.ProjectID,
				ScopeType:                 args.ScopeType,
				ScopeID:                   args.ScopeID,
				Role:                      args.Role,
				AgentName:                 args.AgentName,
				AgentInstanceID:           args.AgentInstanceID,
				ParentInstanceID:          args.ParentInstanceID,
				AllowEqualScopeDelegation: args.AllowEqualScopeDelegation,
				RequestedTTLSeconds:       args.RequestedTTLSeconds,
				OverrideToken:             args.OverrideToken,
				SessionID:                 args.SessionID,
				SessionSecret:             args.SessionSecret,
			})
		},
	)

	srv.AddTool(
		mcp.NewTool(
			"till.heartbeat_capability_lease",
			mcp.WithDescription("Heartbeat one active capability lease."),
			mcp.WithString("agent_instance_id", mcp.Required(), mcp.Description("Agent instance identifier")),
			mcp.WithString("lease_token", mcp.Required(), mcp.Description("Lease token")),
			mcp.WithString("session_id", mcp.Required(), mcp.Description(mcpMutationSessionDescription)),
			mcp.WithString("session_secret", mcp.Required(), mcp.Description(mcpMutationSessionSecretDescription)),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var args struct {
				AgentInstanceID string `json:"agent_instance_id"`
				LeaseToken      string `json:"lease_token"`
				SessionID       string `json:"session_id"`
				SessionSecret   string `json:"session_secret"`
			}
			if err := req.BindArguments(&args); err != nil {
				return invalidRequestToolResult(err), nil
			}
			return handleCapabilityLeaseMutation(ctx, leases, capabilityLeaseMutationArgs{
				Operation:       "heartbeat",
				AgentInstanceID: args.AgentInstanceID,
				LeaseToken:      args.LeaseToken,
				SessionID:       args.SessionID,
				SessionSecret:   args.SessionSecret,
			})
		},
	)

	srv.AddTool(
		mcp.NewTool(
			"till.renew_capability_lease",
			mcp.WithDescription("Renew one capability lease expiry."),
			mcp.WithString("agent_instance_id", mcp.Required(), mcp.Description("Agent instance identifier")),
			mcp.WithString("lease_token", mcp.Required(), mcp.Description("Lease token")),
			mcp.WithNumber("ttl_seconds", mcp.Description("Optional renewal TTL in seconds")),
			mcp.WithString("session_id", mcp.Required(), mcp.Description(mcpMutationSessionDescription)),
			mcp.WithString("session_secret", mcp.Required(), mcp.Description(mcpMutationSessionSecretDescription)),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var args struct {
				AgentInstanceID string `json:"agent_instance_id"`
				LeaseToken      string `json:"lease_token"`
				TTLSeconds      int    `json:"ttl_seconds"`
				SessionID       string `json:"session_id"`
				SessionSecret   string `json:"session_secret"`
			}
			if err := req.BindArguments(&args); err != nil {
				return invalidRequestToolResult(err), nil
			}
			return handleCapabilityLeaseMutation(ctx, leases, capabilityLeaseMutationArgs{
				Operation:       "renew",
				AgentInstanceID: args.AgentInstanceID,
				LeaseToken:      args.LeaseToken,
				TTLSeconds:      args.TTLSeconds,
				SessionID:       args.SessionID,
				SessionSecret:   args.SessionSecret,
			})
		},
	)

	srv.AddTool(
		mcp.NewTool(
			"till.revoke_capability_lease",
			mcp.WithDescription("Revoke one capability lease by instance id."),
			mcp.WithString("agent_instance_id", mcp.Required(), mcp.Description("Agent instance identifier")),
			mcp.WithString("reason", mcp.Description("Optional revocation reason")),
			mcp.WithString("session_id", mcp.Required(), mcp.Description(mcpMutationSessionDescription)),
			mcp.WithString("session_secret", mcp.Required(), mcp.Description(mcpMutationSessionSecretDescription)),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var args struct {
				AgentInstanceID string `json:"agent_instance_id"`
				Reason          string `json:"reason"`
				SessionID       string `json:"session_id"`
				SessionSecret   string `json:"session_secret"`
			}
			if err := req.BindArguments(&args); err != nil {
				return invalidRequestToolResult(err), nil
			}
			return handleCapabilityLeaseMutation(ctx, leases, capabilityLeaseMutationArgs{
				Operation:       "revoke",
				AgentInstanceID: args.AgentInstanceID,
				Reason:          args.Reason,
				SessionID:       args.SessionID,
				SessionSecret:   args.SessionSecret,
			})
		},
	)

	srv.AddTool(
		mcp.NewTool(
			"till.revoke_all_capability_leases",
			mcp.WithDescription("Revoke all capability leases for one project scope."),
			mcp.WithString("project_id", mcp.Required(), mcp.Description("Project identifier")),
			mcp.WithString("scope_type", mcp.Required(), mcp.Description("project|branch|phase|task|subtask"), mcp.Enum(common.SupportedScopeTypes()...)),
			mcp.WithString("scope_id", mcp.Description("Scope identifier")),
			mcp.WithString("reason", mcp.Description("Optional revocation reason")),
			mcp.WithString("session_id", mcp.Required(), mcp.Description(mcpMutationSessionDescription)),
			mcp.WithString("session_secret", mcp.Required(), mcp.Description(mcpMutationSessionSecretDescription)),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var args struct {
				ProjectID     string `json:"project_id"`
				ScopeType     string `json:"scope_type"`
				ScopeID       string `json:"scope_id"`
				Reason        string `json:"reason"`
				SessionID     string `json:"session_id"`
				SessionSecret string `json:"session_secret"`
			}
			if err := req.BindArguments(&args); err != nil {
				return invalidRequestToolResult(err), nil
			}
			return handleCapabilityLeaseMutation(ctx, leases, capabilityLeaseMutationArgs{
				Operation:     "revoke_all",
				ProjectID:     args.ProjectID,
				ScopeType:     args.ScopeType,
				ScopeID:       args.ScopeID,
				Reason:        args.Reason,
				SessionID:     args.SessionID,
				SessionSecret: args.SessionSecret,
			})
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
			mcp.WithDescription("Create one thread comment with markdown-rich summary/details for a project/branch/phase/task/subtask/decision/note target."),
			mcp.WithString("project_id", mcp.Required(), mcp.Description("Project identifier")),
			mcp.WithString("target_type", mcp.Required(), mcp.Description("project|branch|phase|task|subtask|decision|note"), mcp.Enum("project", "branch", "phase", "task", "subtask", "decision", "note")),
			mcp.WithString("target_id", mcp.Required(), mcp.Description("Target identifier")),
			mcp.WithString("summary", mcp.Required(), mcp.Description("Markdown-rich summary for thread previews")),
			mcp.WithString("body_markdown", mcp.Description("Optional markdown-rich details/body for the comment")),
			mcp.WithString("session_id", mcp.Required(), mcp.Description(mcpMutationSessionDescription)),
			mcp.WithString("session_secret", mcp.Required(), mcp.Description(mcpMutationSessionSecretDescription)),
			mcp.WithString("agent_instance_id", mcp.Description("Optional agent lease instance id for secondary local guard checks")),
			mcp.WithString("lease_token", mcp.Description("Optional agent lease token for secondary local guard checks")),
			mcp.WithString("override_token", mcp.Description("Optional override token")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var args struct {
				ProjectID       string `json:"project_id"`
				TargetType      string `json:"target_type"`
				TargetID        string `json:"target_id"`
				Summary         string `json:"summary"`
				BodyMarkdown    string `json:"body_markdown"`
				SessionID       string `json:"session_id"`
				SessionSecret   string `json:"session_secret"`
				AgentInstanceID string `json:"agent_instance_id"`
				LeaseToken      string `json:"lease_token"`
				OverrideToken   string `json:"override_token"`
			}
			if err := req.BindArguments(&args); err != nil {
				return invalidRequestToolResult(err), nil
			}
			projectID := strings.TrimSpace(args.ProjectID)
			if projectID == "" {
				return mcp.NewToolResultError(`invalid_request: required argument "project_id" not found`), nil
			}
			targetType := strings.TrimSpace(args.TargetType)
			if targetType == "" {
				return mcp.NewToolResultError(`invalid_request: required argument "target_type" not found`), nil
			}
			targetID := strings.TrimSpace(args.TargetID)
			if targetID == "" {
				return mcp.NewToolResultError(`invalid_request: required argument "target_id" not found`), nil
			}
			summary := strings.TrimSpace(args.Summary)
			if summary == "" {
				return mcp.NewToolResultError(`invalid_request: required argument "summary" not found`), nil
			}
			caller, err := authorizeMCPMutation(
				ctx,
				pickMutationAuthorizer(comments),
				mcpSessionAuthArgs{
					SessionID:     args.SessionID,
					SessionSecret: args.SessionSecret,
				},
				"create_comment",
				"project:"+projectID,
				"comment",
				targetID,
				map[string]string{
					"project_id":  projectID,
					"target_type": targetType,
				},
			)
			if err != nil {
				return toolResultFromError(err), nil
			}
			actor, err := buildAuthenticatedMutationActor(caller, mcpMutationGuardArgs{
				AgentInstanceID: args.AgentInstanceID,
				LeaseToken:      args.LeaseToken,
				OverrideToken:   args.OverrideToken,
			})
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			comment, err := comments.CreateComment(ctx, common.CreateCommentRequest{
				ProjectID:    projectID,
				TargetType:   targetType,
				TargetID:     targetID,
				Summary:      summary,
				BodyMarkdown: strings.TrimSpace(args.BodyMarkdown),
				Actor:        actor,
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
			mcp.WithDescription("List comments for one target, including markdown-rich summary/details fields."),
			mcp.WithString("project_id", mcp.Required(), mcp.Description("Project identifier")),
			mcp.WithString("target_type", mcp.Required(), mcp.Description("project|branch|phase|task|subtask|decision|note"), mcp.Enum("project", "branch", "phase", "task", "subtask", "decision", "note")),
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

// optionalDurationArg parses one optional duration argument and keeps the zero value when omitted.
func optionalDurationArg(raw string) (time.Duration, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, nil
	}
	duration, err := time.ParseDuration(raw)
	if err != nil {
		return 0, fmt.Errorf("invalid duration %q: %w", raw, err)
	}
	return duration, nil
}

// invalidRequestToolResult wraps argument-binding failures as deterministic tool errors.
func invalidRequestToolResult(err error) *mcp.CallToolResult {
	if err == nil {
		return mcp.NewToolResultError("invalid_request: malformed arguments")
	}
	return mcp.NewToolResultError("invalid_request: " + err.Error())
}
