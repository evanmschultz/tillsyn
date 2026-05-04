package mcpapi

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/evanmschultz/tillsyn/internal/adapters/server/common"
	"github.com/evanmschultz/tillsyn/internal/app"
	"github.com/evanmschultz/tillsyn/internal/domain"
	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

const (
	mcpMutationSessionDescription       = "Authenticated MCP session identifier"
	mcpMutationSessionSecretDescription = "Authenticated MCP session secret"
	mcpMutationAuthContextDescription   = "Bound MCP auth context handle returned by till.auth_request claim/validate_session on stdio runtimes"
	mcpGuardedMutationTupleDescription  = "Only for authenticated agent sessions; supplying this with a user session is invalid. Claim or validate a project-scoped approved agent session before guarded in-project mutations."
	mcpAgentInstanceDescription         = "Optional agent lease instance id for secondary local guard checks. " + mcpGuardedMutationTupleDescription
	mcpLeaseTokenDescription            = "Optional agent lease token for secondary local guard checks. " + mcpGuardedMutationTupleDescription
	mcpOverrideTokenDescription         = "Optional override token for secondary local guard checks. " + mcpGuardedMutationTupleDescription
	mcpGuardedMutationToolSuffix        = " Guarded lease tuple fields are only for authenticated agent sessions; a user session plus agent_instance_id/lease_token is invalid. Claim or validate a project-scoped approved agent session before guarded in-project mutations."
	mcpCapabilityLeaseToolSuffix        = " Issuing or renewing a lease does not upgrade a user session into an agent session; guarded mutation tuples on other tools still require authenticated agent sessions."
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
	resolvedAuth, err := resolveMCPMutationAuth(ctx, auth)
	if err != nil {
		return domain.AuthenticatedCaller{}, err
	}
	return authorizer.AuthorizeMutation(ctx, common.MutationAuthorizationRequest{
		SessionID:     strings.TrimSpace(resolvedAuth.SessionID),
		SessionSecret: strings.TrimSpace(resolvedAuth.SessionSecret),
		Action:        strings.TrimSpace(action),
		Namespace:     strings.TrimSpace(namespace),
		ResourceType:  strings.TrimSpace(resourceType),
		ResourceID:    strings.TrimSpace(resourceID),
		Context:       authContext,
	})
}

// buildAuthenticatedMutationActor converts one authenticated caller plus optional guard tuple into the app adapter actor contract.
func buildAuthenticatedMutationActor(caller domain.AuthenticatedCaller, guard mcpMutationGuardArgs, allowUnguardedAgent bool) (common.ActorLeaseTuple, error) {
	caller = domain.NormalizeAuthenticatedCaller(caller)
	if caller.IsZero() {
		return common.ActorLeaseTuple{}, fmt.Errorf("invalid_request: authenticated caller is required for mutating MCP tools")
	}
	// Drop 3 droplet 3.19: thread AuthRequestPrincipalType through the
	// MCP-layer actor tuple so the STEWARD owner-state-lock survives the
	// trip into withMutationGuardContext. Mirrors the HTTP transport.
	actor := common.ActorLeaseTuple{
		ActorID:                  caller.PrincipalID,
		ActorName:                caller.PrincipalName,
		ActorType:                string(caller.PrincipalType),
		AuthRequestPrincipalType: caller.AuthRequestPrincipalType,
	}

	guard.AgentInstanceID = strings.TrimSpace(guard.AgentInstanceID)
	guard.LeaseToken = strings.TrimSpace(guard.LeaseToken)
	guard.OverrideToken = strings.TrimSpace(guard.OverrideToken)
	hasGuardTuple := guard.AgentInstanceID != "" || guard.LeaseToken != "" || guard.OverrideToken != ""

	if caller.PrincipalType != domain.ActorTypeAgent {
		if hasGuardTuple {
			return common.ActorLeaseTuple{}, fmt.Errorf(
				"invalid_request: guarded mutation tuple requires an authenticated agent session; current session principal_type=%s. Remove agent_instance_id/lease_token to act as a human, or claim/validate a project-scoped approved agent session before retrying",
				caller.PrincipalType,
			)
		}
		return actor, nil
	}
	if allowUnguardedAgent && !hasGuardTuple {
		return actor, nil
	}
	if guard.AgentInstanceID == "" || guard.LeaseToken == "" {
		return common.ActorLeaseTuple{}, fmt.Errorf("invalid_request: agent_instance_id and lease_token are required for authenticated agent mutations; agent identity comes from the authenticated session")
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
	IncludeRevoked            bool   `json:"include_revoked"`
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
	case "list":
		if projectID == "" {
			return mcp.NewToolResultError(`invalid_request: required argument "project_id" not found`), nil
		}
		leasesRows, err := leases.ListCapabilityLeases(ctx, common.ListCapabilityLeasesRequest{
			ProjectID:      projectID,
			ScopeType:      scopeType,
			ScopeID:        scopeID,
			IncludeRevoked: args.IncludeRevoked,
		})
		if err != nil {
			return toolResultFromError(err), nil
		}
		result, err := mcp.NewToolResultJSON(map[string]any{"leases": leasesRows})
		if err != nil {
			return nil, fmt.Errorf("encode capability_lease list result: %w", err)
		}
		return result, nil
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
		} else if agentName == "" {
			return mcp.NewToolResultError(`invalid_request: required argument "agent_name" not found`), nil
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
			mcp.WithDescription("Return lightweight runtime bootstrap guidance for empty-instance flows. This remains the compatibility wrapper on the frozen surface; prefer till.get_instructions(mode=explain, focus=topic, topic=bootstrap) for the richer canonical bootstrap explanation. For restart/recovery on an existing instance, use till.capture_state first instead of bootstrap."),
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
	changes common.ChangeFeedService,
	authContexts *mcpAuthContextStore,
	exposeLegacyProjectTools bool,
) {
	if projects == nil && kinds == nil && changes == nil {
		return
	}

	srv.AddTool(
		mcp.NewTool(
			"till.project",
			mcp.WithDescription("Read or mutate one project-root operation. Use operation=list|create|update|set_allowed_kinds|list_allowed_kinds|list_change_events|get_dependency_rollup."+mcpGuardedMutationToolSuffix),
			mcp.WithString("operation", mcp.Required(), mcp.Description("Project operation"), mcp.Enum("list", "create", "update", "set_allowed_kinds", "list_allowed_kinds", "list_change_events", "get_dependency_rollup")),
			mcp.WithString("project_id", mcp.Description("Project identifier. Required for operation=update|set_allowed_kinds|list_allowed_kinds|list_change_events|get_dependency_rollup")),
			mcp.WithBoolean("include_archived", mcp.Description("Include archived projects for operation=list")),
			mcp.WithNumber("limit", mcp.Description("Maximum rows to return for operation=list_change_events")),
			mcp.WithString("name", mcp.Description("Project name. Required for operation=create|update")),
			mcp.WithString("description", mcp.Description("Project details in markdown-rich text")),
			mcp.WithArray("kind_ids", mcp.Description("Allowed kind id list for operation=set_allowed_kinds."), mcp.WithStringItems()),
			mcp.WithObject("metadata", mcp.Description("Optional project metadata object")),
			mcp.WithString("session_id", mcp.Description("Required for mutating operations. "+mcpMutationSessionDescription)),
			mcp.WithString("session_secret", mcp.Description("Required for mutating operations. "+mcpMutationSessionSecretDescription)),
			mcp.WithString("auth_context_id", mcp.Description("Required for mutating operations when using a bound stdio auth handle. "+mcpMutationAuthContextDescription)),
			mcp.WithString("agent_instance_id", mcp.Description(mcpAgentInstanceDescription)),
			mcp.WithString("lease_token", mcp.Description(mcpLeaseTokenDescription)),
			mcp.WithString("override_token", mcp.Description(mcpOverrideTokenDescription)),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			ctx = withMCPToolAuthRuntime(ctx, authContexts, req)
			var args struct {
				Operation       string                 `json:"operation"`
				ProjectID       string                 `json:"project_id"`
				IncludeArchived bool                   `json:"include_archived"`
				Limit           int                    `json:"limit"`
				Name            string                 `json:"name"`
				Description     string                 `json:"description"`
				KindIDs         []string               `json:"kind_ids"`
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
			switch strings.TrimSpace(args.Operation) {
			case "list":
				if projects == nil {
					return mcp.NewToolResultError("invalid_request: project service is unavailable"), nil
				}
				rows, err := projects.ListProjects(ctx, args.IncludeArchived)
				if err != nil {
					return toolResultFromError(err), nil
				}
				result, err := mcp.NewToolResultJSON(map[string]any{"projects": rows})
				if err != nil {
					return nil, fmt.Errorf("encode project list result: %w", err)
				}
				return result, nil
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
				}, true)
				if err != nil {
					return mcp.NewToolResultError(err.Error()), nil
				}
				project, err := projects.CreateProject(ctx, common.CreateProjectRequest{
					Name:        args.Name,
					Description: args.Description,
					Metadata:    args.Metadata,
					Actor:       actor,
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
				}, false)
				if err != nil {
					return mcp.NewToolResultError(err.Error()), nil
				}
				project, err := projects.UpdateProject(ctx, common.UpdateProjectRequest{
					ProjectID:   args.ProjectID,
					Name:        args.Name,
					Description: args.Description,
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
			case "list_allowed_kinds":
				if kinds == nil {
					return mcp.NewToolResultError("invalid_request: kind catalog service is unavailable"), nil
				}
				projectID := strings.TrimSpace(args.ProjectID)
				if projectID == "" {
					return mcp.NewToolResultError(`invalid_request: required argument "project_id" not found`), nil
				}
				kindIDs, err := kinds.ListProjectAllowedKinds(ctx, projectID)
				if err != nil {
					return toolResultFromError(err), nil
				}
				result, err := mcp.NewToolResultJSON(map[string]any{"kind_ids": kindIDs})
				if err != nil {
					return nil, fmt.Errorf("encode project list_allowed_kinds result: %w", err)
				}
				return result, nil
			case "list_change_events":
				if changes == nil {
					return mcp.NewToolResultError("invalid_request: change feed service is unavailable"), nil
				}
				projectID := strings.TrimSpace(args.ProjectID)
				if projectID == "" {
					return mcp.NewToolResultError(`invalid_request: required argument "project_id" not found`), nil
				}
				rows, err := changes.ListProjectChangeEvents(ctx, projectID, args.Limit)
				if err != nil {
					return toolResultFromError(err), nil
				}
				result, err := mcp.NewToolResultJSON(map[string]any{"events": rows})
				if err != nil {
					return nil, fmt.Errorf("encode project list_change_events result: %w", err)
				}
				return result, nil
			case "get_dependency_rollup":
				if changes == nil {
					return mcp.NewToolResultError("invalid_request: change feed service is unavailable"), nil
				}
				projectID := strings.TrimSpace(args.ProjectID)
				if projectID == "" {
					return mcp.NewToolResultError(`invalid_request: required argument "project_id" not found`), nil
				}
				rollup, err := changes.GetProjectDependencyRollup(ctx, projectID)
				if err != nil {
					return toolResultFromError(err), nil
				}
				result, err := mcp.NewToolResultJSON(rollup)
				if err != nil {
					return nil, fmt.Errorf("encode project get_dependency_rollup result: %w", err)
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
				"till.list_projects",
				mcp.WithDescription("List projects (legacy alias for till.project operation=list)."),
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
				mcp.WithString("description", mcp.Description("Project details in markdown-rich text")),
				mcp.WithObject("metadata", mcp.Description("Optional project metadata object")),
				mcp.WithString("session_id", mcp.Required(), mcp.Description(mcpMutationSessionDescription)),
				mcp.WithString("session_secret", mcp.Required(), mcp.Description(mcpMutationSessionSecretDescription)),
				mcp.WithString("agent_instance_id", mcp.Description(mcpAgentInstanceDescription)),
				mcp.WithString("lease_token", mcp.Description(mcpLeaseTokenDescription)),
				mcp.WithString("override_token", mcp.Description(mcpOverrideTokenDescription)),
			),
			func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
				var args struct {
					Name            string                 `json:"name"`
					Description     string                 `json:"description"`
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
				}, false)
				if err != nil {
					return mcp.NewToolResultError(err.Error()), nil
				}
				project, err := projects.CreateProject(ctx, common.CreateProjectRequest{
					Name:        args.Name,
					Description: args.Description,
					Metadata:    args.Metadata,
					Actor:       actor,
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
				mcp.WithObject("metadata", mcp.Description("Optional project metadata object")),
				mcp.WithString("session_id", mcp.Required(), mcp.Description(mcpMutationSessionDescription)),
				mcp.WithString("session_secret", mcp.Required(), mcp.Description(mcpMutationSessionSecretDescription)),
				mcp.WithString("agent_instance_id", mcp.Description(mcpAgentInstanceDescription)),
				mcp.WithString("lease_token", mcp.Description(mcpLeaseTokenDescription)),
				mcp.WithString("override_token", mcp.Description(mcpOverrideTokenDescription)),
			),
			func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
				var args struct {
					ProjectID       string                 `json:"project_id"`
					Name            string                 `json:"name"`
					Description     string                 `json:"description"`
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
				}, false)
				if err != nil {
					return mcp.NewToolResultError(err.Error()), nil
				}
				project, err := projects.UpdateProject(ctx, common.UpdateProjectRequest{
					ProjectID:   args.ProjectID,
					Name:        args.Name,
					Description: args.Description,
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

// registerActionItemTools registers actionItem reads plus the reduced action-item mutation family.
func registerActionItemTools(
	srv *mcpserver.MCPServer,
	tasks common.ActionItemService,
	search common.SearchService,
	embeddings common.EmbeddingsService,
	authContexts *mcpAuthContextStore,
	exposeLegacyActionItemTools bool,
) {
	if tasks != nil {
		handleActionItemOperation := func(ctx context.Context, req mcp.CallToolRequest, toolLabel string, fixedOperation string) (*mcp.CallToolResult, error) {
			ctx = withMCPToolAuthRuntime(ctx, authContexts, req)
			var args struct {
				Operation       string                     `json:"operation"`
				ProjectID       string                     `json:"project_id"`
				ParentID        string                     `json:"parent_id"`
				Kind            string                     `json:"kind"`
				Scope           string                     `json:"scope"`
				Role            string                     `json:"role"`
				StructuralType  string                     `json:"structural_type"`
				Owner           *string                    `json:"owner"`
				DropNumber      *int                       `json:"drop_number"`
				Persistent      *bool                      `json:"persistent"`
				DevGated        *bool                      `json:"dev_gated"`
				Paths           *[]string                  `json:"paths"`
				Packages        *[]string                  `json:"packages"`
				Files           *[]string                  `json:"files"`
				StartCommit     *string                    `json:"start_commit"`
				EndCommit       *string                    `json:"end_commit"`
				ColumnID        string                     `json:"column_id"`
				Title           string                     `json:"title"`
				Description     string                     `json:"description"`
				Priority        string                     `json:"priority"`
				DueAt           string                     `json:"due_at"`
				Labels          []string                   `json:"labels"`
				Metadata        *domain.ActionItemMetadata `json:"metadata"`
				ActionItemID    string                     `json:"action_item_id"`
				ToColumnID      string                     `json:"to_column_id"`
				Position        *int                       `json:"position"`
				State           string                     `json:"state"`
				IncludeArchived bool                       `json:"include_archived"`
				Query           string                     `json:"query"`
				CrossProject    bool                       `json:"cross_project"`
				States          []string                   `json:"states"`
				Levels          []string                   `json:"levels"`
				Kinds           []string                   `json:"kinds"`
				LabelsAny       []string                   `json:"labels_any"`
				LabelsAll       []string                   `json:"labels_all"`
				SearchMode      string                     `json:"search_mode"`
				Sort            string                     `json:"sort"`
				Limit           *int                       `json:"limit"`
				Offset          *int                       `json:"offset"`
				Mode            string                     `json:"mode"`
				SessionID       string                     `json:"session_id"`
				SessionSecret   string                     `json:"session_secret"`
				AgentInstanceID string                     `json:"agent_instance_id"`
				LeaseToken      string                     `json:"lease_token"`
				OverrideToken   string                     `json:"override_token"`
			}
			if err := req.BindArguments(&args); err != nil {
				return invalidRequestToolResult(err), nil
			}
			operation := strings.TrimSpace(fixedOperation)
			if operation == "" {
				operation = strings.TrimSpace(args.Operation)
			}

			switch operation {
			case "get":
				actionItemID := strings.TrimSpace(args.ActionItemID)
				if actionItemID == "" {
					return mcp.NewToolResultError(`invalid_request: required argument "action_item_id" not found`), nil
				}
				resolvedID, err := resolveActionItemIDForRead(ctx, tasks, strings.TrimSpace(args.ProjectID), actionItemID)
				if err != nil {
					return toolResultFromError(err), nil
				}
				actionItem, err := tasks.GetActionItem(ctx, resolvedID)
				if err != nil {
					return toolResultFromError(err), nil
				}
				result, err := mcp.NewToolResultJSON(actionItem)
				if err != nil {
					return nil, fmt.Errorf("encode %s get result: %w", toolLabel, err)
				}
				return result, nil
			case "list":
				projectID := strings.TrimSpace(args.ProjectID)
				if projectID == "" {
					return mcp.NewToolResultError(`invalid_request: required argument "project_id" not found`), nil
				}
				includeArchived := args.IncludeArchived
				parentID := strings.TrimSpace(args.ParentID)
				if parentID != "" {
					rows, err := tasks.ListChildActionItems(ctx, projectID, parentID, includeArchived)
					if err != nil {
						return toolResultFromError(err), nil
					}
					result, err := mcp.NewToolResultJSON(map[string]any{"tasks": rows})
					if err != nil {
						return nil, fmt.Errorf("encode %s list child result: %w", toolLabel, err)
					}
					return result, nil
				}
				rows, err := tasks.ListActionItems(ctx, projectID, includeArchived)
				if err != nil {
					return toolResultFromError(err), nil
				}
				result, err := mcp.NewToolResultJSON(map[string]any{"tasks": rows})
				if err != nil {
					return nil, fmt.Errorf("encode %s list result: %w", toolLabel, err)
				}
				return result, nil
			case "search":
				if search == nil {
					return mcp.NewToolResultError("invalid_request: search service is unavailable"), nil
				}
				searchMode := strings.TrimSpace(args.SearchMode)
				if searchMode == "" {
					searchMode = strings.TrimSpace(req.GetString("mode", ""))
				}
				searchReq := common.SearchActionItemsRequest{
					ProjectID:       strings.TrimSpace(args.ProjectID),
					Query:           strings.TrimSpace(args.Query),
					CrossProject:    args.CrossProject,
					IncludeArchived: args.IncludeArchived,
					States:          append([]string(nil), args.States...),
					Levels:          append([]string(nil), args.Levels...),
					Kinds:           append([]string(nil), args.Kinds...),
					LabelsAny:       append([]string(nil), args.LabelsAny...),
					LabelsAll:       append([]string(nil), args.LabelsAll...),
					Mode:            searchMode,
					Sort:            strings.TrimSpace(args.Sort),
				}
				if args.Limit != nil {
					searchReq.Limit = *args.Limit
				}
				if args.Offset != nil {
					searchReq.Offset = *args.Offset
				}
				resultPayload, err := search.SearchActionItems(ctx, searchReq)
				if err != nil {
					return toolResultFromError(err), nil
				}
				result, err := mcp.NewToolResultJSON(resultPayload)
				if err != nil {
					return nil, fmt.Errorf("encode %s search result: %w", toolLabel, err)
				}
				return result, nil
			case "create":
				projectID := strings.TrimSpace(args.ProjectID)
				if projectID == "" {
					return mcp.NewToolResultError(`invalid_request: required argument "project_id" not found`), nil
				}
				columnID := strings.TrimSpace(args.ColumnID)
				if columnID == "" {
					return mcp.NewToolResultError(`invalid_request: required argument "column_id" not found`), nil
				}
				title := strings.TrimSpace(args.Title)
				if title == "" {
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
					"project:"+projectID,
					"actionItem",
					"new",
					map[string]string{
						"project_id": projectID,
						"parent_id":  strings.TrimSpace(args.ParentID),
						"column_id":  columnID,
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
				}, false)
				if err != nil {
					return mcp.NewToolResultError(err.Error()), nil
				}
				var metadata domain.ActionItemMetadata
				if args.Metadata != nil {
					metadata = *args.Metadata
				}
				createReq := common.CreateActionItemRequest{
					ProjectID:      args.ProjectID,
					ParentID:       args.ParentID,
					Kind:           args.Kind,
					Scope:          args.Scope,
					Role:           args.Role,
					StructuralType: args.StructuralType,
					ColumnID:       args.ColumnID,
					Title:          args.Title,
					Description:    args.Description,
					Priority:       args.Priority,
					DueAt:          args.DueAt,
					Labels:         append([]string(nil), args.Labels...),
					Metadata:       metadata,
					Actor:          actor,
				}
				// Owner / DropNumber / Persistent / DevGated are domain
				// primitives (per L13). Pointer-sentinel inputs from the args
				// struct collapse to value-type fields on the create request:
				// nil = "not supplied" = leave the request's zero value
				// (empty string / 0 / false), which domain.NewActionItem
				// accepts as the default.
				if args.Owner != nil {
					createReq.Owner = *args.Owner
				}
				if args.DropNumber != nil {
					createReq.DropNumber = *args.DropNumber
				}
				if args.Persistent != nil {
					createReq.Persistent = *args.Persistent
				}
				if args.DevGated != nil {
					createReq.DevGated = *args.DevGated
				}
				// Paths pointer-sentinel: nil = "not supplied" → leave the
				// request's empty-slice zero value (domain accepts as no
				// path scope); non-nil → copy the supplied slice through.
				if args.Paths != nil {
					createReq.Paths = append([]string(nil), (*args.Paths)...)
				}
				// Packages pointer-sentinel matches Paths above: nil = "not
				// supplied" → leave empty-slice zero value; non-nil → copy
				// supplied slice through. Domain enforces the coverage
				// invariant (non-empty Paths requires non-empty Packages)
				// at NewActionItem time.
				if args.Packages != nil {
					createReq.Packages = append([]string(nil), (*args.Packages)...)
				}
				// Files pointer-sentinel matches Paths/Packages above: nil
				// = "not supplied" → leave empty-slice zero value; non-nil
				// → copy supplied slice through. Domain trims + dedupes;
				// no cross-axis coverage check vs Paths (Files is a
				// disjoint axis — read attention vs write intent).
				if args.Files != nil {
					createReq.Files = append([]string(nil), (*args.Files)...)
				}
				// StartCommit pointer-sentinel: nil = "not supplied" →
				// leave the request's empty-string zero value (domain
				// accepts as "not yet captured"); non-nil → set to the
				// dereferenced string (domain trims surrounding
				// whitespace at NewActionItem time). Free-form opaque-
				// domain field — no format check applies.
				if args.StartCommit != nil {
					createReq.StartCommit = *args.StartCommit
				}
				// EndCommit pointer-sentinel mirrors StartCommit above:
				// nil = "not supplied" → leave the request's empty-string
				// zero value (domain accepts as "not yet captured");
				// non-nil → set to the dereferenced string (domain trims
				// surrounding whitespace at NewActionItem time). Free-form
				// opaque-domain field — no format check applies.
				if args.EndCommit != nil {
					createReq.EndCommit = *args.EndCommit
				}
				actionItem, err := tasks.CreateActionItem(ctx, createReq)
				if err != nil {
					return toolResultFromError(err), nil
				}
				result, err := mcp.NewToolResultJSON(actionItem)
				if err != nil {
					return nil, fmt.Errorf("encode %s create result: %w", toolLabel, err)
				}
				return result, nil
			case "update":
				actionItemID := strings.TrimSpace(args.ActionItemID)
				if actionItemID == "" {
					return mcp.NewToolResultError(`invalid_request: required argument "action_item_id" not found`), nil
				}
				if err := rejectMutationDottedActionItemID(actionItemID); err != nil {
					return toolResultFromError(err), nil
				}
				title := strings.TrimSpace(args.Title)
				if title == "" {
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
					"actionItem",
					actionItemID,
					map[string]string{"action_item_id": actionItemID},
				)
				if err != nil {
					return toolResultFromError(err), nil
				}
				actor, err := buildAuthenticatedMutationActor(caller, mcpMutationGuardArgs{
					AgentInstanceID: args.AgentInstanceID,
					LeaseToken:      args.LeaseToken,
					OverrideToken:   args.OverrideToken,
				}, false)
				if err != nil {
					return mcp.NewToolResultError(err.Error()), nil
				}
				actionItem, err := tasks.UpdateActionItem(ctx, common.UpdateActionItemRequest{
					ActionItemID:   args.ActionItemID,
					Title:          args.Title,
					Description:    args.Description,
					Priority:       args.Priority,
					DueAt:          args.DueAt,
					Labels:         append([]string(nil), args.Labels...),
					Role:           args.Role,
					StructuralType: args.StructuralType,
					// Pointer-sentinels pass through verbatim — nil preserves
					// the existing value at the service boundary; non-nil
					// applies the dereferenced value. Lets the L1 STEWARD
					// field-level guard distinguish "absent" from "explicit
					// empty/zero/false" without collapsing the wire shape.
					Owner:      args.Owner,
					DropNumber: args.DropNumber,
					Persistent: args.Persistent,
					DevGated:   args.DevGated,
					// Paths pointer-sentinel passes through verbatim: nil
					// preserves the existing slice at the service boundary;
					// non-nil applies the dereferenced slice (empty
					// dereferenced slice clears all declared paths).
					Paths: args.Paths,
					// Packages pointer-sentinel mirrors Paths: nil preserves
					// the existing slice; non-nil applies the dereferenced
					// slice (empty dereferenced slice clears all declared
					// packages). Coverage invariant ("non-empty Paths
					// requires non-empty Packages") is re-checked against
					// the post-apply pair at the service boundary.
					Packages: args.Packages,
					// Files pointer-sentinel mirrors Paths/Packages: nil
					// preserves the existing slice; non-nil applies the
					// dereferenced slice (empty dereferenced slice clears
					// all declared files). No cross-axis coverage check
					// against Paths — Files is disjoint-axis (read
					// attention) and may legitimately overlap with Paths.
					Files: args.Files,
					// StartCommit pointer-sentinel passes through verbatim:
					// nil preserves the existing value at the service
					// boundary; non-nil applies the dereferenced string
					// (empty dereferenced string clears the prior commit
					// hash). Service applies inline strings.TrimSpace so
					// the create-time trim rule applies equally on update.
					StartCommit: args.StartCommit,
					// EndCommit pointer-sentinel mirrors StartCommit: nil
					// preserves the existing value; non-nil applies the
					// dereferenced string (empty dereferenced string clears
					// the prior commit hash). Wave 2 dispatcher populates
					// this BEFORE MoveActionItemState. Service applies
					// inline strings.TrimSpace so the create-time trim
					// rule applies equally on update.
					EndCommit: args.EndCommit,
					Metadata:  args.Metadata,
					Actor:     actor,
				})
				if err != nil {
					return toolResultFromError(err), nil
				}
				result, err := mcp.NewToolResultJSON(actionItem)
				if err != nil {
					return nil, fmt.Errorf("encode %s update result: %w", toolLabel, err)
				}
				return result, nil
			case "move":
				actionItemID := strings.TrimSpace(args.ActionItemID)
				if actionItemID == "" {
					return mcp.NewToolResultError(`invalid_request: required argument "action_item_id" not found`), nil
				}
				if err := rejectMutationDottedActionItemID(actionItemID); err != nil {
					return toolResultFromError(err), nil
				}
				toColumnID := strings.TrimSpace(args.ToColumnID)
				if toColumnID == "" {
					return mcp.NewToolResultError(`invalid_request: required argument "to_column_id" not found`), nil
				}
				if args.Position == nil {
					return mcp.NewToolResultError(`invalid_request: required argument "position" not found`), nil
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
					"actionItem",
					actionItemID,
					map[string]string{"action_item_id": actionItemID, "to_column_id": toColumnID},
				)
				if err != nil {
					return toolResultFromError(err), nil
				}
				actor, err := buildAuthenticatedMutationActor(caller, mcpMutationGuardArgs{
					AgentInstanceID: args.AgentInstanceID,
					LeaseToken:      args.LeaseToken,
					OverrideToken:   args.OverrideToken,
				}, false)
				if err != nil {
					return mcp.NewToolResultError(err.Error()), nil
				}
				actionItem, err := tasks.MoveActionItem(ctx, common.MoveActionItemRequest{
					ActionItemID: actionItemID,
					ToColumnID:   toColumnID,
					Position:     *args.Position,
					Actor:        actor,
				})
				if err != nil {
					return toolResultFromError(err), nil
				}
				result, err := mcp.NewToolResultJSON(actionItem)
				if err != nil {
					return nil, fmt.Errorf("encode %s move result: %w", toolLabel, err)
				}
				return result, nil
			case "move_state":
				actionItemID := strings.TrimSpace(args.ActionItemID)
				if actionItemID == "" {
					return mcp.NewToolResultError(`invalid_request: required argument "action_item_id" not found`), nil
				}
				if err := rejectMutationDottedActionItemID(actionItemID); err != nil {
					return toolResultFromError(err), nil
				}
				state := strings.TrimSpace(args.State)
				if state == "" {
					return mcp.NewToolResultError(`invalid_request: required argument "state" not found`), nil
				}
				caller, err := authorizeMCPMutation(
					ctx,
					pickMutationAuthorizer(tasks),
					mcpSessionAuthArgs{
						SessionID:     args.SessionID,
						SessionSecret: args.SessionSecret,
					},
					"move_task_state",
					"tillsyn",
					"actionItem",
					actionItemID,
					map[string]string{"action_item_id": actionItemID, "state": state},
				)
				if err != nil {
					return toolResultFromError(err), nil
				}
				actor, err := buildAuthenticatedMutationActor(caller, mcpMutationGuardArgs{
					AgentInstanceID: args.AgentInstanceID,
					LeaseToken:      args.LeaseToken,
					OverrideToken:   args.OverrideToken,
				}, false)
				if err != nil {
					return mcp.NewToolResultError(err.Error()), nil
				}
				actionItem, err := tasks.MoveActionItemState(ctx, common.MoveActionItemStateRequest{
					ActionItemID: actionItemID,
					State:        state,
					Actor:        actor,
				})
				if err != nil {
					return toolResultFromError(err), nil
				}
				result, err := mcp.NewToolResultJSON(actionItem)
				if err != nil {
					return nil, fmt.Errorf("encode %s move_state result: %w", toolLabel, err)
				}
				return result, nil
			case "delete":
				actionItemID := strings.TrimSpace(args.ActionItemID)
				if actionItemID == "" {
					return mcp.NewToolResultError(`invalid_request: required argument "action_item_id" not found`), nil
				}
				if err := rejectMutationDottedActionItemID(actionItemID); err != nil {
					return toolResultFromError(err), nil
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
					"actionItem",
					actionItemID,
					map[string]string{"action_item_id": actionItemID, "mode": strings.TrimSpace(args.Mode)},
				)
				if err != nil {
					return toolResultFromError(err), nil
				}
				actor, err := buildAuthenticatedMutationActor(caller, mcpMutationGuardArgs{
					AgentInstanceID: args.AgentInstanceID,
					LeaseToken:      args.LeaseToken,
					OverrideToken:   args.OverrideToken,
				}, false)
				if err != nil {
					return mcp.NewToolResultError(err.Error()), nil
				}
				if err := tasks.DeleteActionItem(ctx, common.DeleteActionItemRequest{
					ActionItemID: actionItemID,
					Mode:         args.Mode,
					Actor:        actor,
				}); err != nil {
					return toolResultFromError(err), nil
				}
				result, err := mcp.NewToolResultJSON(map[string]any{
					"deleted":        true,
					"action_item_id": actionItemID,
					"mode":           args.Mode,
				})
				if err != nil {
					return nil, fmt.Errorf("encode %s delete result: %w", toolLabel, err)
				}
				return result, nil
			case "restore":
				actionItemID := strings.TrimSpace(args.ActionItemID)
				if actionItemID == "" {
					return mcp.NewToolResultError(`invalid_request: required argument "action_item_id" not found`), nil
				}
				if err := rejectMutationDottedActionItemID(actionItemID); err != nil {
					return toolResultFromError(err), nil
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
					"actionItem",
					actionItemID,
					map[string]string{"action_item_id": actionItemID},
				)
				if err != nil {
					return toolResultFromError(err), nil
				}
				actor, err := buildAuthenticatedMutationActor(caller, mcpMutationGuardArgs{
					AgentInstanceID: args.AgentInstanceID,
					LeaseToken:      args.LeaseToken,
					OverrideToken:   args.OverrideToken,
				}, false)
				if err != nil {
					return mcp.NewToolResultError(err.Error()), nil
				}
				actionItem, err := tasks.RestoreActionItem(ctx, common.RestoreActionItemRequest{
					ActionItemID: actionItemID,
					Actor:        actor,
				})
				if err != nil {
					return toolResultFromError(err), nil
				}
				result, err := mcp.NewToolResultJSON(actionItem)
				if err != nil {
					return nil, fmt.Errorf("encode %s restore result: %w", toolLabel, err)
				}
				return result, nil
			case "reparent":
				actionItemID := strings.TrimSpace(args.ActionItemID)
				if actionItemID == "" {
					return mcp.NewToolResultError(`invalid_request: required argument "action_item_id" not found`), nil
				}
				if err := rejectMutationDottedActionItemID(actionItemID); err != nil {
					return toolResultFromError(err), nil
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
					"actionItem",
					actionItemID,
					map[string]string{"action_item_id": actionItemID, "parent_id": strings.TrimSpace(args.ParentID)},
				)
				if err != nil {
					return toolResultFromError(err), nil
				}
				actor, err := buildAuthenticatedMutationActor(caller, mcpMutationGuardArgs{
					AgentInstanceID: args.AgentInstanceID,
					LeaseToken:      args.LeaseToken,
					OverrideToken:   args.OverrideToken,
				}, false)
				if err != nil {
					return mcp.NewToolResultError(err.Error()), nil
				}
				actionItem, err := tasks.ReparentActionItem(ctx, common.ReparentActionItemRequest{
					ActionItemID: actionItemID,
					ParentID:     args.ParentID,
					Actor:        actor,
				})
				if err != nil {
					return toolResultFromError(err), nil
				}
				result, err := mcp.NewToolResultJSON(actionItem)
				if err != nil {
					return nil, fmt.Errorf("encode %s reparent result: %w", toolLabel, err)
				}
				return result, nil
			default:
				return mcp.NewToolResultError(`invalid_request: required argument "operation" not found`), nil
			}
		}

		srv.AddTool(
			mcp.NewTool(
				"till.action_item",
				mcp.WithDescription("Read or mutate one action-item operation for branch|phase|actionItem|subtask hierarchy nodes under a project. Use operation=get|list|search|create|update|move|move_state|delete|restore|reparent. operation=get accepts action_item_id as either a UUID or a dotted address (e.g. \"1.5.2\" or \"<project_slug>:1.5.2\"); dotted form requires project_id (or the slug-prefix form, which carries the slug). Mutation operations (update|move|move_state|delete|restore|reparent) require a UUID action_item_id and reject dotted addresses with an invalid_request error — dotted addresses are positional and shift under sibling reordering, so allowing them on mutations would let a caller silently mutate the wrong item."+mcpGuardedMutationToolSuffix),
				mcp.WithString("operation", mcp.Required(), mcp.Description("Action-item operation"), mcp.Enum("get", "list", "search", "create", "update", "move", "move_state", "delete", "restore", "reparent")),
				mcp.WithString("project_id", mcp.Description("Project identifier. Required for operation=list|create, optional for operation=search, and required for operation=get when action_item_id is a bare dotted address (omit when action_item_id is a UUID or carries a slug-prefix shorthand)")),
				mcp.WithString("action_item_id", mcp.Description("Action-item identifier. Required for operation=get|update|move|move_state|delete|restore|reparent. operation=get accepts a UUID OR a dotted address (\"1.5.2\" or \"<slug>:1.5.2\"); mutations reject dotted form and require the UUID")),
				mcp.WithString("column_id", mcp.Description("Column identifier. Required for operation=create")),
				mcp.WithString("to_column_id", mcp.Description("Destination column identifier. Required for operation=move")),
				mcp.WithNumber("position", mcp.Description("Destination position. Required for operation=move")),
				mcp.WithString("state", mcp.Description("Lifecycle state target for operation=move_state (for example: todo|in_progress|complete)")),
				mcp.WithString("title", mcp.Description("Title. Required for operation=create|update")),
				mcp.WithString("parent_id", mcp.Description("Optional parent action-item id for operation=create, new parent id for operation=reparent, or child root for operation=list")),
				mcp.WithString("kind", mcp.Description("Kind identifier for operation=create")),
				mcp.WithString("scope", mcp.Description("project|branch|phase|actionItem|subtask"), mcp.Enum(common.SupportedScopeTypes()...)),
				mcp.WithString("role", mcp.Description("Optional role tag for operation=create|update — see allowed values (closed enum: builder|qa-proof|qa-falsification|qa-a11y|qa-visual|design|commit|planner|research). Empty string preserves the existing value on update.")),
				mcp.WithString("structural_type", mcp.Description("Required for operation=create — closed enum: drop|segment|confluence|droplet (waterfall metaphor — see WIKI.md §Cascade Vocabulary). Empty rejects on create. Empty preserves prior value on update."), mcp.Enum("drop", "segment", "confluence", "droplet")),
				mcp.WithString("owner", mcp.Description("Optional Owner principal-name string for operation=create|update (e.g. \"STEWARD\"). Free-form, whitespace-trimmed; domain primitive — not STEWARD-specific. On update, omit to preserve the existing value; supplying any value (including empty string) triggers the L1 STEWARD field-level guard at the adapter boundary.")),
				mcp.WithNumber("drop_number", mcp.Description("Optional cascade drop index for operation=create|update. Zero = \"not a numbered drop\"; positive values round-trip; negative values reject with invalid_request. On update, omit to preserve the existing value.")),
				mcp.WithBoolean("persistent", mcp.Description("Optional Persistent flag for operation=create|update — long-lived umbrella / anchor / perpetual-tracking nodes. Default false. Domain primitive — not STEWARD-specific. On update, omit to preserve the existing value (a value-typed bool would silently clobber Persistent=true on STEWARD anchors).")),
				mcp.WithBoolean("dev_gated", mcp.Description("Optional DevGated flag for operation=create|update — nodes whose terminal transition requires dev sign-off. Default false. Domain primitive — not STEWARD-specific. On update, omit to preserve the existing value.")),
				mcp.WithArray("paths", mcp.Description("Optional Paths string-array for operation=create|update — declares the action item's write-scope file paths (forward-slash, repo-root-relative). Empty array on create = no path scope declared. On update, omit to preserve the existing slice; supplying any array (including empty) replaces the declared paths. Domain trims + dedupes; whitespace-only / backslash-bearing entries reject with invalid_request. Domain primitive — Drop 4a L3 lock-domain field consumed by the Wave 2 dispatcher's file-level lock manager."), mcp.WithStringItems()),
				mcp.WithArray("packages", mcp.Description("Optional Packages string-array for operation=create|update — declares the Go-package import paths covering Paths. Empty array on create = no package scope declared. On update, omit to preserve the existing slice; supplying any array (including empty) replaces the declared packages. Domain trims + dedupes; whitespace-only / empty entries reject with invalid_request. Coverage invariant: non-empty Paths requires non-empty Packages — paired Paths/Packages updates are validated atomically against the post-apply pair. No Go-import-path format enforcement; planner-set values are what matter. Domain primitive — Drop 4a L3 / WAVE_1_PLAN.md §1.2 lock-domain field consumed by the Wave 2 dispatcher's package-level lock manager."), mcp.WithStringItems()),
				mcp.WithArray("files", mcp.Description("Optional Files string-array for operation=create|update — declares reference-material file paths the agent should read (forward-slash, repo-root-relative). Distinct from Paths, which declares write-scope / lock domain. Empty array on create = no reference files attached. On update, omit to preserve the existing slice; supplying any array (including empty) replaces the declared files. Domain trims + dedupes; whitespace-only / backslash-bearing entries reject with invalid_request. Disjoint-axis with Paths — Files (read attention) and Paths (write intent) may legitimately overlap (e.g. read-then-edit workflows), so no cross-axis coverage check applies. Path-exists is NOT enforced at the domain layer — the canonical consumer is the Drop 4.5 TUI file-viewer pane, which validates existence at view time. Domain primitive — Drop 4a L3 / WAVE_1_PLAN.md §1.3."), mcp.WithStringItems()),
				mcp.WithString("start_commit", mcp.Description("Optional StartCommit free-form string for operation=create|update — records the git commit hash captured at the moment work begins on this action item, typically the current `git rev-parse HEAD` of the bare-root or active worktree at in_progress transition time. Empty string is the meaningful zero value (\"not yet captured\"). On update, omit to preserve the existing value; supplying any string (including empty) replaces it. Domain trims surrounding whitespace; no format check applies — short-SHAs (7-char), full-SHAs (40-char), and any caller-supplied identifier all round-trip. Opaque-domain field: the domain layer holds the value opaquely and never calls git itself — the caller (orchestrator pre-cascade; Wave 2 dispatcher; Drop 4b commit-agent) supplies it. Domain primitive — Drop 4a L3 / WAVE_1_PLAN.md §1.4. Drop 4b commit-agent consumes this for diff context (`git diff <start_commit>..<end_commit>` baseline).")),
				mcp.WithString("end_commit", mcp.Description("Optional EndCommit free-form string for operation=create|update — records the git commit hash captured at the moment work completes on this action item, typically the current `git rev-parse HEAD` of the bare-root or active worktree captured just before the terminal state transition (caller populates via update BEFORE move_state). Empty string is the meaningful zero value (\"not yet captured\") and is valid until terminal state — domain does NOT enforce non-empty-on-terminal (Drop 4b dispatcher concern). On update, omit to preserve the existing value; supplying any string (including empty) replaces it. Domain trims surrounding whitespace; no format check applies — short-SHAs (7-char), full-SHAs (40-char), and any caller-supplied identifier all round-trip. Opaque-domain field: the domain layer holds the value opaquely and never calls git itself — the caller (orchestrator pre-cascade; Wave 2 dispatcher; Drop 4b commit-agent) supplies it. No chronology check against StartCommit. Domain primitive — Drop 4a L3 / WAVE_1_PLAN.md §1.5. Drop 4b commit-agent consumes this for diff context (`git diff <start_commit>..<end_commit>` baseline).")),
				mcp.WithString("description", mcp.Description("Action-item details in markdown-rich text")),
				mcp.WithString("priority", mcp.Description("low|medium|high"), mcp.Enum("low", "medium", "high")),
				mcp.WithString("due_at", mcp.Description("Optional RFC3339 timestamp")),
				mcp.WithArray("labels", mcp.Description("Optional labels"), mcp.WithStringItems()),
				mcp.WithObject("metadata", mcp.Description("Optional action-item metadata object")),
				mcp.WithBoolean("include_archived", mcp.Description("Include archived action-items for operation=list|search")),
				mcp.WithString("query", mcp.Description("Search query for operation=search")),
				mcp.WithBoolean("cross_project", mcp.Description("Search across all projects for operation=search")),
				mcp.WithArray("states", mcp.Description("Optional state filter for operation=search"), mcp.WithStringItems()),
				mcp.WithArray("levels", mcp.Description("Optional level/scope filter for operation=search"), mcp.WithStringItems()),
				mcp.WithArray("kinds", mcp.Description("Optional kind filter for operation=search"), mcp.WithStringItems()),
				mcp.WithArray("labels_any", mcp.Description("Optional labels-any filter for operation=search"), mcp.WithStringItems()),
				mcp.WithArray("labels_all", mcp.Description("Optional labels-all filter for operation=search"), mcp.WithStringItems()),
				mcp.WithString("search_mode", mcp.Description("keyword|semantic|hybrid (default hybrid; semantic/hybrid fall back to keyword when embeddings/vector search is unavailable)"), mcp.Enum("keyword", "semantic", "hybrid")),
				mcp.WithString("sort", mcp.Description("rank_desc|title_asc|created_at_desc|updated_at_desc (default rank_desc)"), mcp.Enum("rank_desc", "title_asc", "created_at_desc", "updated_at_desc")),
				mcp.WithNumber("limit", mcp.Description("Optional maximum rows for operation=search (default 50, max 200)"), mcp.DefaultNumber(50), mcp.Min(0), mcp.Max(200)),
				mcp.WithNumber("offset", mcp.Description("Optional row offset for operation=search (default 0, must be >= 0)"), mcp.DefaultNumber(0), mcp.Min(0)),
				mcp.WithString("mode", mcp.Description("archive|hard for operation=delete"), mcp.Enum("archive", "hard")),
				mcp.WithString("session_id", mcp.Description(mcpMutationSessionDescription)),
				mcp.WithString("session_secret", mcp.Description(mcpMutationSessionSecretDescription)),
				mcp.WithString("auth_context_id", mcp.Description(mcpMutationAuthContextDescription)),
				mcp.WithString("agent_instance_id", mcp.Description(mcpAgentInstanceDescription)),
				mcp.WithString("lease_token", mcp.Description(mcpLeaseTokenDescription)),
				mcp.WithString("override_token", mcp.Description(mcpOverrideTokenDescription)),
			),
			func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
				return handleActionItemOperation(ctx, req, "action_item", "")
			},
		)

		if exposeLegacyActionItemTools {
			srv.AddTool(
				mcp.NewTool(
					"till.list_tasks",
					mcp.WithDescription("List tasks/work-items for one project (legacy alias for till.action_item operation=list)."),
					mcp.WithString("project_id", mcp.Required(), mcp.Description("Project identifier")),
					mcp.WithBoolean("include_archived", mcp.Description("Include archived tasks")),
				),
				func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
					return handleActionItemOperation(ctx, req, "list_tasks", "list")
				},
			)

			srv.AddTool(
				mcp.NewTool(
					"till.create_task",
					mcp.WithDescription("Create one actionItem/work-item (legacy alias for till.action_item operation=create)."),
					mcp.WithString("project_id", mcp.Required(), mcp.Description("Project identifier")),
					mcp.WithString("column_id", mcp.Required(), mcp.Description("Column identifier")),
					mcp.WithString("title", mcp.Required(), mcp.Description("ActionItem title")),
					mcp.WithString("parent_id", mcp.Description("Optional parent actionItem id")),
					mcp.WithString("kind", mcp.Description("Kind identifier")),
					mcp.WithString("scope", mcp.Description("project|branch|phase|actionItem|subtask"), mcp.Enum(common.SupportedScopeTypes()...)),
					mcp.WithString("role", mcp.Description("Optional role tag — closed enum: builder|qa-proof|qa-falsification|qa-a11y|qa-visual|design|commit|planner|research")),
					mcp.WithString("structural_type", mcp.Description("Required for operation=create — closed enum: drop|segment|confluence|droplet (waterfall metaphor — see WIKI.md §Cascade Vocabulary). Empty rejects on create. Empty preserves prior value on update."), mcp.Enum("drop", "segment", "confluence", "droplet")),
					mcp.WithString("description", mcp.Description("ActionItem details in markdown-rich text")),
					mcp.WithString("priority", mcp.Description("low|medium|high"), mcp.Enum("low", "medium", "high")),
					mcp.WithString("due_at", mcp.Description("Optional RFC3339 timestamp")),
					mcp.WithArray("labels", mcp.Description("Optional labels"), mcp.WithStringItems()),
					mcp.WithObject("metadata", mcp.Description("Optional actionItem metadata object")),
					mcp.WithString("session_id", mcp.Required(), mcp.Description(mcpMutationSessionDescription)),
					mcp.WithString("session_secret", mcp.Required(), mcp.Description(mcpMutationSessionSecretDescription)),
					mcp.WithString("agent_instance_id", mcp.Description(mcpAgentInstanceDescription)),
					mcp.WithString("lease_token", mcp.Description(mcpLeaseTokenDescription)),
					mcp.WithString("override_token", mcp.Description(mcpOverrideTokenDescription)),
				),
				func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
					return handleActionItemOperation(ctx, req, "create_task", "create")
				},
			)

			srv.AddTool(
				mcp.NewTool(
					"till.update_task",
					mcp.WithDescription("Update one actionItem/work-item (legacy alias for till.action_item operation=update)."),
					mcp.WithString("action_item_id", mcp.Required(), mcp.Description("ActionItem identifier")),
					mcp.WithString("title", mcp.Required(), mcp.Description("ActionItem title")),
					mcp.WithString("description", mcp.Description("ActionItem details in markdown-rich text")),
					mcp.WithString("priority", mcp.Description("low|medium|high"), mcp.Enum("low", "medium", "high")),
					mcp.WithString("due_at", mcp.Description("Optional RFC3339 timestamp")),
					mcp.WithArray("labels", mcp.Description("Optional labels"), mcp.WithStringItems()),
					mcp.WithString("role", mcp.Description("Optional role tag — closed enum: builder|qa-proof|qa-falsification|qa-a11y|qa-visual|design|commit|planner|research. Empty preserves prior value.")),
					mcp.WithString("structural_type", mcp.Description("Optional structural-type update — closed enum: drop|segment|confluence|droplet (waterfall metaphor — see WIKI.md §Cascade Vocabulary). Empty preserves prior value."), mcp.Enum("drop", "segment", "confluence", "droplet")),
					mcp.WithObject("metadata", mcp.Description("Optional actionItem metadata object")),
					mcp.WithString("session_id", mcp.Required(), mcp.Description(mcpMutationSessionDescription)),
					mcp.WithString("session_secret", mcp.Required(), mcp.Description(mcpMutationSessionSecretDescription)),
					mcp.WithString("agent_instance_id", mcp.Description(mcpAgentInstanceDescription)),
					mcp.WithString("lease_token", mcp.Description(mcpLeaseTokenDescription)),
					mcp.WithString("override_token", mcp.Description(mcpOverrideTokenDescription)),
				),
				func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
					return handleActionItemOperation(ctx, req, "update_task", "update")
				},
			)

			srv.AddTool(
				mcp.NewTool(
					"till.move_task",
					mcp.WithDescription("Move one actionItem/work-item to another column/position (legacy alias for till.action_item operation=move)."),
					mcp.WithString("action_item_id", mcp.Required(), mcp.Description("ActionItem identifier")),
					mcp.WithString("to_column_id", mcp.Required(), mcp.Description("Destination column identifier")),
					mcp.WithNumber("position", mcp.Required(), mcp.Description("Destination position")),
					mcp.WithString("session_id", mcp.Required(), mcp.Description(mcpMutationSessionDescription)),
					mcp.WithString("session_secret", mcp.Required(), mcp.Description(mcpMutationSessionSecretDescription)),
					mcp.WithString("agent_instance_id", mcp.Description(mcpAgentInstanceDescription)),
					mcp.WithString("lease_token", mcp.Description(mcpLeaseTokenDescription)),
					mcp.WithString("override_token", mcp.Description(mcpOverrideTokenDescription)),
				),
				func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
					return handleActionItemOperation(ctx, req, "move_task", "move")
				},
			)

			srv.AddTool(
				mcp.NewTool(
					"till.delete_task",
					mcp.WithDescription("Delete one actionItem/work-item (archive or hard; legacy alias for till.action_item operation=delete)."),
					mcp.WithString("action_item_id", mcp.Required(), mcp.Description("ActionItem identifier")),
					mcp.WithString("mode", mcp.Description("archive|hard"), mcp.Enum("archive", "hard")),
					mcp.WithString("session_id", mcp.Required(), mcp.Description(mcpMutationSessionDescription)),
					mcp.WithString("session_secret", mcp.Required(), mcp.Description(mcpMutationSessionSecretDescription)),
					mcp.WithString("agent_instance_id", mcp.Description(mcpAgentInstanceDescription)),
					mcp.WithString("lease_token", mcp.Description(mcpLeaseTokenDescription)),
					mcp.WithString("override_token", mcp.Description(mcpOverrideTokenDescription)),
				),
				func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
					return handleActionItemOperation(ctx, req, "delete_task", "delete")
				},
			)

			srv.AddTool(
				mcp.NewTool(
					"till.restore_task",
					mcp.WithDescription("Restore one archived actionItem/work-item (legacy alias for till.action_item operation=restore)."),
					mcp.WithString("action_item_id", mcp.Required(), mcp.Description("ActionItem identifier")),
					mcp.WithString("session_id", mcp.Required(), mcp.Description(mcpMutationSessionDescription)),
					mcp.WithString("session_secret", mcp.Required(), mcp.Description(mcpMutationSessionSecretDescription)),
					mcp.WithString("agent_instance_id", mcp.Description(mcpAgentInstanceDescription)),
					mcp.WithString("lease_token", mcp.Description(mcpLeaseTokenDescription)),
					mcp.WithString("override_token", mcp.Description(mcpOverrideTokenDescription)),
				),
				func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
					return handleActionItemOperation(ctx, req, "restore_task", "restore")
				},
			)

			srv.AddTool(
				mcp.NewTool(
					"till.reparent_task",
					mcp.WithDescription("Change parent relationship for one actionItem/work-item (legacy alias for till.action_item operation=reparent)."),
					mcp.WithString("action_item_id", mcp.Required(), mcp.Description("ActionItem identifier")),
					mcp.WithString("parent_id", mcp.Description("New parent identifier (empty to unset where allowed)")),
					mcp.WithString("session_id", mcp.Required(), mcp.Description(mcpMutationSessionDescription)),
					mcp.WithString("session_secret", mcp.Required(), mcp.Description(mcpMutationSessionSecretDescription)),
					mcp.WithString("agent_instance_id", mcp.Description(mcpAgentInstanceDescription)),
					mcp.WithString("lease_token", mcp.Description(mcpLeaseTokenDescription)),
					mcp.WithString("override_token", mcp.Description(mcpOverrideTokenDescription)),
				),
				func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
					return handleActionItemOperation(ctx, req, "reparent_task", "reparent")
				},
			)
			srv.AddTool(
				mcp.NewTool(
					"till.list_child_tasks",
					mcp.WithDescription("List child tasks for a parent scope (legacy alias for till.action_item operation=list with parent_id)."),
					mcp.WithString("project_id", mcp.Required(), mcp.Description("Project identifier")),
					mcp.WithString("parent_id", mcp.Required(), mcp.Description("Parent actionItem identifier")),
					mcp.WithBoolean("include_archived", mcp.Description("Include archived child rows")),
				),
				func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
					return handleActionItemOperation(ctx, req, "list_child_tasks", "list")
				},
			)
			srv.AddTool(
				mcp.NewTool(
					"till.search_task_matches",
					mcp.WithDescription("Search actionItem/work-item matches by query, mode, sort, filters, and scope (legacy alias for till.action_item operation=search)."),
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
					mcp.WithNumber("limit", mcp.Description("Optional maximum rows (default 50, max 200)"), mcp.DefaultNumber(50), mcp.Min(0), mcp.Max(200)),
					mcp.WithNumber("offset", mcp.Description("Optional row offset (default 0, must be >= 0)"), mcp.DefaultNumber(0), mcp.Min(0)),
				),
				func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
					return handleActionItemOperation(ctx, req, "search_task_matches", "search")
				},
			)
		}
	}

	if embeddings != nil {
		srv.AddTool(
			mcp.NewTool(
				"till.embeddings",
				mcp.WithDescription("Inspect or reindex embeddings lifecycle state. Use operation=status|reindex."),
				mcp.WithString("operation", mcp.Required(), mcp.Description("Embeddings operation"), mcp.Enum("status", "reindex")),
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
				switch strings.TrimSpace(req.GetString("operation", "")) {
				case "status":
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
						return nil, fmt.Errorf("encode embeddings status result: %w", err)
					}
					return result, nil
				case "reindex":
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
						return nil, fmt.Errorf("encode embeddings reindex result: %w", err)
					}
					return result, nil
				default:
					return mcp.NewToolResultError(`invalid_request: required argument "operation" not found`), nil
				}
			},
		)
	}
}

// registerKindTools registers kind catalog and project allowlist tools.
func registerKindTools(srv *mcpserver.MCPServer, kinds common.KindCatalogService, authContexts *mcpAuthContextStore, exposeLegacyProjectTools bool) {
	if kinds == nil {
		return
	}

	// Per Drop 3 droplet 3.15 (finding 5.B.13 / CE8) the till.kind tool is
	// read-only: operation=upsert was deleted from the wire surface, the
	// till.upsert_kind_definition legacy alias was deleted, and the till
	// kind upsert CLI subcommand was deleted. Programmatic
	// Service.UpsertKindDefinition callers (snapshot import + tests) keep
	// the path alive — wire-level mutation re-lands when the new template
	// system needs an explicit edit channel.
	srv.AddTool(
		mcp.NewTool(
			"till.kind",
			mcp.WithDescription("Inspect kind catalog definitions. Use operation=list."),
			mcp.WithString("operation", mcp.Required(), mcp.Description("Kind operation"), mcp.Enum("list")),
			mcp.WithBoolean("include_archived", mcp.Description("Include archived kind definitions")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			ctx = withMCPToolAuthRuntime(ctx, authContexts, req)
			switch strings.TrimSpace(req.GetString("operation", "")) {
			case "list":
				rows, err := kinds.ListKindDefinitions(ctx, req.GetBool("include_archived", false))
				if err != nil {
					return toolResultFromError(err), nil
				}
				result, err := mcp.NewToolResultJSON(map[string]any{"kinds": rows})
				if err != nil {
					return nil, fmt.Errorf("encode kind list result: %w", err)
				}
				return result, nil
			default:
				return mcp.NewToolResultError(`invalid_request: required argument "operation" not found`), nil
			}
		},
	)

	if exposeLegacyProjectTools {
		srv.AddTool(
			mcp.NewTool(
				"till.list_kind_definitions",
				mcp.WithDescription("List kind catalog definitions (legacy alias for till.kind operation=list)."),
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

		// Per Drop 3 droplet 3.15 (finding 5.B.13 / CE8) the
		// till.upsert_kind_definition legacy alias was deleted along with
		// the till.kind operation=upsert wire surface above. Read-only
		// till.list_kind_definitions stays for legacy clients; mutating
		// upsert paths are no longer exposed over the wire.

		srv.AddTool(
			mcp.NewTool(
				"till.set_project_allowed_kinds",
				mcp.WithDescription("Set explicit project allowed kind identifiers. Use this to keep a project limited to its template kinds or to intentionally opt specific generic kinds in after discussing that policy with the dev."),
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
}

// registerCapabilityLeaseTools registers lease visibility and lifecycle tools.
func registerCapabilityLeaseTools(srv *mcpserver.MCPServer, leases common.CapabilityLeaseService, authContexts *mcpAuthContextStore, exposeLegacyLeaseTools bool) {
	if leases == nil {
		return
	}

	srv.AddTool(
		mcp.NewTool(
			"till.capability_lease",
			mcp.WithDescription("List or mutate capability lease lifecycle state. Use operation=list|issue|heartbeat|renew|revoke|revoke_all."+mcpCapabilityLeaseToolSuffix),
			mcp.WithString("operation", mcp.Required(), mcp.Description("Capability lease operation"), mcp.Enum("list", "issue", "heartbeat", "renew", "revoke", "revoke_all")),
			mcp.WithString("project_id", mcp.Description("Project identifier. Required for operation=list|issue|revoke_all")),
			mcp.WithString("scope_type", mcp.Description("project|branch|phase|actionItem|subtask. Optional for operation=list; required for operation=issue|revoke_all"), mcp.Enum(common.SupportedScopeTypes()...)),
			mcp.WithString("scope_id", mcp.Description("Scope identifier. Optional for operation=list and for project scope; otherwise used by operation=issue|revoke_all")),
			mcp.WithBoolean("include_revoked", mcp.Description("Include revoked leases in addition to active leases when operation=list")),
			mcp.WithString("role", mcp.Description("orchestrator|builder|qa|research. Required for operation=issue"), mcp.Enum("orchestrator", "builder", "qa", "research")),
			mcp.WithString("agent_name", mcp.Description("Agent display/name identifier. Optional when issuing under an authenticated agent session because the live lease identity is derived from that session; otherwise required for operation=issue. Issuing a lease under a user session does not convert that user session into an authenticated agent session for later guarded mutations.")),
			mcp.WithString("agent_instance_id", mcp.Description("Agent instance identifier. Required for operation=heartbeat|renew|revoke and optional for operation=issue")),
			mcp.WithString("parent_instance_id", mcp.Description("Optional parent lease instance id for operation=issue")),
			mcp.WithBoolean("allow_equal_scope_delegation", mcp.Description("Allow equal-scope delegation for operation=issue")),
			mcp.WithNumber("requested_ttl_seconds", mcp.Description("Optional TTL in seconds for operation=issue")),
			mcp.WithString("override_token", mcp.Description("Optional orchestrator overlap override token for operation=issue")),
			mcp.WithString("lease_token", mcp.Description("Lease token. Required for operation=heartbeat|renew")),
			mcp.WithNumber("ttl_seconds", mcp.Description("Optional renewal TTL in seconds for operation=renew")),
			mcp.WithString("reason", mcp.Description("Optional revocation reason for operation=revoke|revoke_all")),
			mcp.WithString("session_id", mcp.Description("Required for operation=issue|heartbeat|renew|revoke|revoke_all. "+mcpMutationSessionDescription)),
			mcp.WithString("session_secret", mcp.Description("Required for operation=issue|heartbeat|renew|revoke|revoke_all. "+mcpMutationSessionSecretDescription)),
			mcp.WithString("auth_context_id", mcp.Description("Required for operation=issue|heartbeat|renew|revoke|revoke_all when using a bound stdio auth handle. "+mcpMutationAuthContextDescription)),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			ctx = withMCPToolAuthRuntime(ctx, authContexts, req)
			var args capabilityLeaseMutationArgs
			if err := req.BindArguments(&args); err != nil {
				return invalidRequestToolResult(err), nil
			}
			return handleCapabilityLeaseMutation(ctx, leases, args)
		},
	)

	if exposeLegacyLeaseTools {
		registerLegacyCapabilityLeaseReadTool(srv, leases)
		registerLegacyCapabilityLeaseMutationTools(srv, leases)
	}
}

func registerLegacyCapabilityLeaseReadTool(srv *mcpserver.MCPServer, leases common.CapabilityLeaseService) {
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
			var args capabilityLeaseMutationArgs
			if err := req.BindArguments(&args); err != nil {
				return invalidRequestToolResult(err), nil
			}
			args.Operation = "list"
			return handleCapabilityLeaseMutation(ctx, leases, args)
		},
	)
}

func registerLegacyCapabilityLeaseMutationTools(srv *mcpserver.MCPServer, leases common.CapabilityLeaseService) {
	srv.AddTool(
		mcp.NewTool(
			"till.issue_capability_lease",
			mcp.WithDescription("Issue one capability lease."),
			mcp.WithString("project_id", mcp.Required(), mcp.Description("Project identifier")),
			mcp.WithString("scope_type", mcp.Required(), mcp.Description("project|branch|phase|actionItem|subtask"), mcp.Enum(common.SupportedScopeTypes()...)),
			mcp.WithString("scope_id", mcp.Description("Scope identifier")),
			mcp.WithString("role", mcp.Required(), mcp.Description("orchestrator|builder|qa|research"), mcp.Enum("orchestrator", "builder", "qa", "research")),
			mcp.WithString("agent_name", mcp.Description("Agent display/name identifier. Optional when issuing under an authenticated agent session because the live lease identity is derived from that session. Issuing a lease under a user session does not convert that user session into an authenticated agent session for later guarded mutations.")),
			mcp.WithString("agent_instance_id", mcp.Description("Optional stable agent instance id")),
			mcp.WithString("parent_instance_id", mcp.Description("Optional parent lease instance id")),
			mcp.WithBoolean("allow_equal_scope_delegation", mcp.Description("Allow equal-scope delegation")),
			mcp.WithNumber("requested_ttl_seconds", mcp.Description("Optional TTL in seconds")),
			mcp.WithString("override_token", mcp.Description("Optional orchestrator overlap override token")),
			mcp.WithString("session_id", mcp.Required(), mcp.Description(mcpMutationSessionDescription)),
			mcp.WithString("session_secret", mcp.Required(), mcp.Description(mcpMutationSessionSecretDescription)),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var args capabilityLeaseMutationArgs
			if err := req.BindArguments(&args); err != nil {
				return invalidRequestToolResult(err), nil
			}
			args.Operation = "issue"
			return handleCapabilityLeaseMutation(ctx, leases, args)
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
			var args capabilityLeaseMutationArgs
			if err := req.BindArguments(&args); err != nil {
				return invalidRequestToolResult(err), nil
			}
			args.Operation = "heartbeat"
			return handleCapabilityLeaseMutation(ctx, leases, args)
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
			var args capabilityLeaseMutationArgs
			if err := req.BindArguments(&args); err != nil {
				return invalidRequestToolResult(err), nil
			}
			args.Operation = "renew"
			return handleCapabilityLeaseMutation(ctx, leases, args)
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
			var args capabilityLeaseMutationArgs
			if err := req.BindArguments(&args); err != nil {
				return invalidRequestToolResult(err), nil
			}
			args.Operation = "revoke"
			return handleCapabilityLeaseMutation(ctx, leases, args)
		},
	)

	srv.AddTool(
		mcp.NewTool(
			"till.revoke_all_capability_leases",
			mcp.WithDescription("Revoke all capability leases for one project scope."),
			mcp.WithString("project_id", mcp.Required(), mcp.Description("Project identifier")),
			mcp.WithString("scope_type", mcp.Required(), mcp.Description("project|branch|phase|actionItem|subtask"), mcp.Enum(common.SupportedScopeTypes()...)),
			mcp.WithString("scope_id", mcp.Description("Scope identifier")),
			mcp.WithString("reason", mcp.Description("Optional revocation reason")),
			mcp.WithString("session_id", mcp.Required(), mcp.Description(mcpMutationSessionDescription)),
			mcp.WithString("session_secret", mcp.Required(), mcp.Description(mcpMutationSessionSecretDescription)),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var args capabilityLeaseMutationArgs
			if err := req.BindArguments(&args); err != nil {
				return invalidRequestToolResult(err), nil
			}
			args.Operation = "revoke_all"
			return handleCapabilityLeaseMutation(ctx, leases, args)
		},
	)
}

// registerCommentTools registers comment create/list tools.
func registerCommentTools(srv *mcpserver.MCPServer, comments common.CommentService, authContexts *mcpAuthContextStore) {
	if comments == nil {
		return
	}

	srv.AddTool(
		mcp.NewTool(
			"till.comment",
			mcp.WithDescription("Create or list append-only shared thread comments. Use comments for discussion/status updates; @mentions route comment inbox rows and are not the same as Action Required handoffs. During active runs, operation=list can wait for the next thread update, and after client shutdown/restart you should rerun capture_state plus comment/attention reads to recover thread state."+mcpGuardedMutationToolSuffix),
			mcp.WithString("operation", mcp.Required(), mcp.Description("Comment operation"), mcp.Enum("create", "list")),
			mcp.WithString("project_id", mcp.Description("Project identifier")),
			mcp.WithString("target_type", mcp.Description("project|branch|phase|actionItem|subtask|decision|note"), mcp.Enum("project", "branch", "phase", "actionItem", "subtask", "decision", "note")),
			mcp.WithString("target_id", mcp.Description("Target identifier")),
			mcp.WithString("wait_timeout", mcp.Description("Optional how long operation=list should wait for the next thread change after capturing the current thread state, for example 30s. Use this while actively watching a thread; without a new change before timeout it returns the current comments, and after restart you should rerun operation=list to recover current thread state.")),
			mcp.WithString("summary", mcp.Description("Required for operation=create. Markdown-rich thread summary; use @human, @dev, @builder, @qa, @orchestrator, or @research when routing comment inbox mentions")),
			mcp.WithString("body_markdown", mcp.Description("Optional markdown-rich details/body for the comment; rendered as shared thread content, not as a private role mailbox")),
			mcp.WithString("session_id", mcp.Description("Required for operation=create. "+mcpMutationSessionDescription)),
			mcp.WithString("session_secret", mcp.Description("Required for operation=create. "+mcpMutationSessionSecretDescription)),
			mcp.WithString("auth_context_id", mcp.Description("Required for operation=create when using a bound stdio auth handle. "+mcpMutationAuthContextDescription)),
			mcp.WithString("agent_instance_id", mcp.Description(mcpAgentInstanceDescription)),
			mcp.WithString("lease_token", mcp.Description(mcpLeaseTokenDescription)),
			mcp.WithString("override_token", mcp.Description(mcpOverrideTokenDescription)),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			ctx = withMCPToolAuthRuntime(ctx, authContexts, req)
			var args struct {
				ProjectID       string `json:"project_id"`
				TargetType      string `json:"target_type"`
				TargetID        string `json:"target_id"`
				WaitTimeout     string `json:"wait_timeout"`
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
			switch strings.TrimSpace(req.GetString("operation", "")) {
			case "create":
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
				}, false)
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
					return nil, fmt.Errorf("encode comment create result: %w", err)
				}
				return result, nil
			case "list":
				rows, err := comments.ListCommentsByTarget(ctx, common.ListCommentsByTargetRequest{
					ProjectID:   projectID,
					TargetType:  targetType,
					TargetID:    targetID,
					WaitTimeout: strings.TrimSpace(args.WaitTimeout),
				})
				if err != nil {
					return toolResultFromError(err), nil
				}
				result, err := mcp.NewToolResultJSON(map[string]any{"comments": rows})
				if err != nil {
					return nil, fmt.Errorf("encode comment list result: %w", err)
				}
				return result, nil
			default:
				return mcp.NewToolResultError(`invalid_request: required argument "operation" not found`), nil
			}
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

// resolveActionItemIDForRead accepts the raw `action_item_id` string from a
// read-only MCP operation and returns the canonical UUID. UUID-shaped input
// is returned unchanged. Dotted input (with or without `<slug>:` prefix) is
// resolved against a projectID drawn from one of two sources, in order:
//
//  1. The dotted string carries a `<slug>:<body>` prefix — the slug is mapped
//     to a projectID via tasks.GetProjectBySlug; the dotted form (slug intact)
//     is then forwarded into the resolver, which re-validates the slug against
//     the resolved projectID's slug.
//  2. The caller supplied an explicit `project_id` argument.
//
// If neither source yields a projectID, an invalid_request error is returned
// naming the missing input. Resolver errors (not-found / invalid-syntax) flow
// through tasks.ResolveActionItemID's transport-mapped error sentinels.
func resolveActionItemIDForRead(ctx context.Context, tasks common.ActionItemService, projectID, idOrDotted string) (string, error) {
	idOrDotted = strings.TrimSpace(idOrDotted)
	if idOrDotted == "" {
		return "", fmt.Errorf(`invalid_request: required argument "action_item_id" not found`)
	}
	if !app.IsLikelyDottedAddress(idOrDotted) {
		// Not dotted — let ResolveActionItemID validate UUID shape and pass through.
		return tasks.ResolveActionItemID(ctx, projectID, idOrDotted)
	}
	if slug := app.SplitDottedSlugPrefix(idOrDotted); slug != "" {
		project, err := tasks.GetProjectBySlug(ctx, slug)
		if err != nil {
			return "", err
		}
		projectID = project.ID
	}
	if strings.TrimSpace(projectID) == "" {
		return "", fmt.Errorf(`invalid_request: project_id is required when action_item_id is a dotted address without a slug prefix`)
	}
	return tasks.ResolveActionItemID(ctx, projectID, idOrDotted)
}

// rejectMutationDottedActionItemID enforces that mutation operations receive a
// UUID action_item_id, not a dotted address. Returning a non-nil error from
// this helper at the start of a mutation case sends an invalid_request error
// back to the caller before any auth or service work is attempted. Empty input
// returns nil so the existing required-argument check still owns that error
// surface. The returned error is wrapped under common.ErrInvalidCaptureStateRequest
// so toolResultFromError maps it to the `invalid_request:` 400-class.
func rejectMutationDottedActionItemID(actionItemID string) error {
	actionItemID = strings.TrimSpace(actionItemID)
	if actionItemID == "" {
		return nil
	}
	if err := app.ValidateActionItemIDForMutation(actionItemID); err != nil {
		return fmt.Errorf("%w: %w", common.ErrInvalidCaptureStateRequest, err)
	}
	return nil
}
