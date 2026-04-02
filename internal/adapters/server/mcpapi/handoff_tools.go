package mcpapi

import (
	"context"
	"fmt"
	"strings"

	"github.com/hylla/tillsyn/internal/adapters/server/common"
	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

// registerHandoffTools registers durable handoff create/read/update/list tools.
func registerHandoffTools(srv *mcpserver.MCPServer, handoffs common.HandoffService, authContexts *mcpAuthContextStore, exposeLegacyCoordinationTools bool) {
	if handoffs == nil {
		return
	}

	srv.AddTool(
		mcp.NewTool(
			"till.handoff",
			mcp.WithDescription("Create, update, get, or list durable handoffs for structured agent-agent or human-agent coordination."),
			mcp.WithString("operation",
				mcp.Required(),
				mcp.Enum("create", "update", "get", "list"),
				mcp.Description("Handoff operation"),
			),
			mcp.WithString("project_id", mcp.Description("Project identifier. Required for operation=create|list")),
			mcp.WithString("handoff_id", mcp.Description("Handoff identifier. Required for operation=update|get")),
			mcp.WithString("branch_id", mcp.Description("Optional source branch identifier when operation=create|list")),
			mcp.WithString("scope_type", mcp.Description("Optional source scope level when operation=create|list"), mcp.Enum(common.SupportedScopeTypes()...)),
			mcp.WithString("scope_id", mcp.Description("Optional source scope identifier; defaults to the project id for project scope when operation=create|list")),
			mcp.WithString("source_role", mcp.Description("Optional source role label, for example orchestrator, builder, or qa")),
			mcp.WithString("target_branch_id", mcp.Description("Optional target branch identifier")),
			mcp.WithString("target_scope_type", mcp.Description("Optional target scope level"), mcp.Enum(common.SupportedScopeTypes()...)),
			mcp.WithString("target_scope_id", mcp.Description("Optional target scope identifier")),
			mcp.WithString("target_role", mcp.Description("Optional target role label, for example orchestrator, builder, or qa")),
			mcp.WithString("status", mcp.Description("Optional handoff status"), mcp.Enum("ready", "waiting", "blocked", "failed", "returned", "superseded", "resolved")),
			mcp.WithArray("statuses", mcp.Description("Optional handoff status filter when operation=list"), mcp.WithStringItems()),
			mcp.WithNumber("limit", mcp.Description("Optional maximum rows to return when operation=list")),
			mcp.WithString("summary", mcp.Description("Short handoff summary. Required for operation=create and operation=update")),
			mcp.WithString("next_action", mcp.Description("Optional explicit next action for the receiver")),
			mcp.WithArray("missing_evidence", mcp.Description("Optional missing evidence checklist"), mcp.WithStringItems()),
			mcp.WithArray("related_refs", mcp.Description("Optional related ids or references"), mcp.WithStringItems()),
			mcp.WithString("resolution_note", mcp.Description("Optional resolution note when closing or superseding the handoff during operation=update")),
			mcp.WithString("session_id", mcp.Description("Required for operation=create|update. "+mcpMutationSessionDescription)),
			mcp.WithString("session_secret", mcp.Description("Required for operation=create|update. "+mcpMutationSessionSecretDescription)),
			mcp.WithString("auth_context_id", mcp.Description("Required for operation=create|update when using a bound stdio auth handle. "+mcpMutationAuthContextDescription)),
			mcp.WithString("agent_instance_id", mcp.Description("Optional agent lease instance id for secondary local guard checks")),
			mcp.WithString("lease_token", mcp.Description("Optional agent lease token for secondary local guard checks")),
			mcp.WithString("override_token", mcp.Description("Optional override token")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			ctx = withMCPToolAuthRuntime(ctx, authContexts, req)
			var args handoffMutationArgs
			if err := req.BindArguments(&args); err != nil {
				return invalidRequestToolResult(err), nil
			}
			return handleHandoffMutation(ctx, handoffs, args)
		},
	)

	if exposeLegacyCoordinationTools {
		registerLegacyHandoffReadTools(srv, handoffs)
		registerLegacyHandoffMutationTools(srv, handoffs)
	}
}

type handoffMutationArgs struct {
	Operation       string   `json:"operation"`
	ProjectID       string   `json:"project_id"`
	HandoffID       string   `json:"handoff_id"`
	BranchID        string   `json:"branch_id"`
	ScopeType       string   `json:"scope_type"`
	ScopeID         string   `json:"scope_id"`
	SourceRole      string   `json:"source_role"`
	TargetBranchID  string   `json:"target_branch_id"`
	TargetScopeType string   `json:"target_scope_type"`
	TargetScopeID   string   `json:"target_scope_id"`
	TargetRole      string   `json:"target_role"`
	Status          string   `json:"status"`
	Summary         string   `json:"summary"`
	NextAction      string   `json:"next_action"`
	MissingEvidence []string `json:"missing_evidence"`
	RelatedRefs     []string `json:"related_refs"`
	ResolutionNote  string   `json:"resolution_note"`
	Statuses        []string `json:"statuses"`
	Limit           int      `json:"limit"`
	SessionID       string   `json:"session_id"`
	SessionSecret   string   `json:"session_secret"`
	AgentInstanceID string   `json:"agent_instance_id"`
	LeaseToken      string   `json:"lease_token"`
	OverrideToken   string   `json:"override_token"`
}

func registerLegacyHandoffReadTools(srv *mcpserver.MCPServer, handoffs common.HandoffService) {
	srv.AddTool(
		mcp.NewTool(
			"till.get_handoff",
			mcp.WithDescription("Return one durable handoff by id."),
			mcp.WithString("handoff_id", mcp.Required(), mcp.Description("Handoff identifier")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var args handoffMutationArgs
			if err := req.BindArguments(&args); err != nil {
				return invalidRequestToolResult(err), nil
			}
			args.Operation = "get"
			return handleHandoffMutation(ctx, handoffs, args)
		},
	)

	srv.AddTool(
		mcp.NewTool(
			"till.list_handoffs",
			mcp.WithDescription("List durable handoffs for one scope tuple."),
			mcp.WithString("project_id", mcp.Required(), mcp.Description("Project identifier")),
			mcp.WithString("branch_id", mcp.Description("Optional source branch identifier")),
			mcp.WithString("scope_type", mcp.Description("Optional source scope level"), mcp.Enum(common.SupportedScopeTypes()...)),
			mcp.WithString("scope_id", mcp.Description("Optional source scope identifier; defaults to the project id for project scope")),
			mcp.WithArray("statuses", mcp.Description("Optional handoff status filter"), mcp.WithStringItems()),
			mcp.WithNumber("limit", mcp.Description("Optional maximum rows to return")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var args handoffMutationArgs
			if err := req.BindArguments(&args); err != nil {
				return invalidRequestToolResult(err), nil
			}
			args.Operation = "list"
			return handleHandoffMutation(ctx, handoffs, args)
		},
	)
}

func registerLegacyHandoffMutationTools(srv *mcpserver.MCPServer, handoffs common.HandoffService) {
	srv.AddTool(
		mcp.NewTool(
			"till.create_handoff",
			mcp.WithDescription("Create one durable handoff for structured agent-agent or human-agent coordination."),
			mcp.WithString("project_id", mcp.Required(), mcp.Description("Project identifier")),
			mcp.WithString("branch_id", mcp.Description("Optional source branch identifier")),
			mcp.WithString("scope_type", mcp.Description("Optional source scope level"), mcp.Enum(common.SupportedScopeTypes()...)),
			mcp.WithString("scope_id", mcp.Description("Optional source scope identifier; defaults to the project id for project scope")),
			mcp.WithString("source_role", mcp.Description("Optional source role label, for example orchestrator, builder, or qa")),
			mcp.WithString("target_branch_id", mcp.Description("Optional target branch identifier")),
			mcp.WithString("target_scope_type", mcp.Description("Optional target scope level"), mcp.Enum(common.SupportedScopeTypes()...)),
			mcp.WithString("target_scope_id", mcp.Description("Optional target scope identifier")),
			mcp.WithString("target_role", mcp.Description("Optional target role label, for example orchestrator, builder, or qa")),
			mcp.WithString("status", mcp.Description("Optional handoff status"), mcp.Enum("ready", "waiting", "blocked", "failed", "returned", "superseded", "resolved")),
			mcp.WithString("summary", mcp.Required(), mcp.Description("Short handoff summary")),
			mcp.WithString("next_action", mcp.Description("Optional explicit next action for the receiver")),
			mcp.WithArray("missing_evidence", mcp.Description("Optional missing evidence checklist"), mcp.WithStringItems()),
			mcp.WithArray("related_refs", mcp.Description("Optional related ids or references"), mcp.WithStringItems()),
			mcp.WithString("session_id", mcp.Required(), mcp.Description(mcpMutationSessionDescription)),
			mcp.WithString("session_secret", mcp.Required(), mcp.Description(mcpMutationSessionSecretDescription)),
			mcp.WithString("agent_instance_id", mcp.Description("Optional agent lease instance id for secondary local guard checks")),
			mcp.WithString("lease_token", mcp.Description("Optional agent lease token for secondary local guard checks")),
			mcp.WithString("override_token", mcp.Description("Optional override token")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var args handoffMutationArgs
			if err := req.BindArguments(&args); err != nil {
				return invalidRequestToolResult(err), nil
			}
			args.Operation = "create"
			return handleHandoffMutation(ctx, handoffs, args)
		},
	)

	srv.AddTool(
		mcp.NewTool(
			"till.update_handoff",
			mcp.WithDescription("Update one durable handoff state or routing."),
			mcp.WithString("handoff_id", mcp.Required(), mcp.Description("Handoff identifier")),
			mcp.WithString("status", mcp.Description("Optional handoff status"), mcp.Enum("ready", "waiting", "blocked", "failed", "returned", "superseded", "resolved")),
			mcp.WithString("source_role", mcp.Description("Optional source role label, for example orchestrator, builder, or qa")),
			mcp.WithString("target_branch_id", mcp.Description("Optional target branch identifier")),
			mcp.WithString("target_scope_type", mcp.Description("Optional target scope level"), mcp.Enum(common.SupportedScopeTypes()...)),
			mcp.WithString("target_scope_id", mcp.Description("Optional target scope identifier")),
			mcp.WithString("target_role", mcp.Description("Optional target role label, for example orchestrator, builder, or qa")),
			mcp.WithString("summary", mcp.Required(), mcp.Description("Short handoff summary")),
			mcp.WithString("next_action", mcp.Description("Optional explicit next action for the receiver")),
			mcp.WithArray("missing_evidence", mcp.Description("Optional missing evidence checklist"), mcp.WithStringItems()),
			mcp.WithArray("related_refs", mcp.Description("Optional related ids or references"), mcp.WithStringItems()),
			mcp.WithString("resolution_note", mcp.Description("Optional resolution note when closing or superseding the handoff")),
			mcp.WithString("session_id", mcp.Required(), mcp.Description(mcpMutationSessionDescription)),
			mcp.WithString("session_secret", mcp.Required(), mcp.Description(mcpMutationSessionSecretDescription)),
			mcp.WithString("agent_instance_id", mcp.Description("Optional agent lease instance id for secondary local guard checks")),
			mcp.WithString("lease_token", mcp.Description("Optional agent lease token for secondary local guard checks")),
			mcp.WithString("override_token", mcp.Description("Optional override token")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var args handoffMutationArgs
			if err := req.BindArguments(&args); err != nil {
				return invalidRequestToolResult(err), nil
			}
			args.Operation = "update"
			return handleHandoffMutation(ctx, handoffs, args)
		},
	)
}

func handleHandoffMutation(ctx context.Context, handoffs common.HandoffService, args handoffMutationArgs) (*mcp.CallToolResult, error) {
	operation := strings.TrimSpace(args.Operation)
	switch operation {
	case "get":
		handoffID := strings.TrimSpace(args.HandoffID)
		if handoffID == "" {
			return mcp.NewToolResultError(`invalid_request: required argument "handoff_id" not found`), nil
		}
		handoff, err := handoffs.GetHandoff(ctx, handoffID)
		if err != nil {
			return toolResultFromError(err), nil
		}
		result, err := mcp.NewToolResultJSON(handoff)
		if err != nil {
			return nil, fmt.Errorf("encode handoff get result: %w", err)
		}
		return result, nil
	case "list":
		projectID := strings.TrimSpace(args.ProjectID)
		if projectID == "" {
			return mcp.NewToolResultError(`invalid_request: required argument "project_id" not found`), nil
		}
		handoffRows, err := handoffs.ListHandoffs(ctx, common.ListHandoffsRequest{
			ProjectID: projectID,
			BranchID:  strings.TrimSpace(args.BranchID),
			ScopeType: strings.TrimSpace(args.ScopeType),
			ScopeID:   strings.TrimSpace(args.ScopeID),
			Statuses:  append([]string(nil), args.Statuses...),
			Limit:     args.Limit,
		})
		if err != nil {
			return toolResultFromError(err), nil
		}
		result, err := mcp.NewToolResultJSON(map[string]any{"handoffs": handoffRows})
		if err != nil {
			return nil, fmt.Errorf("encode handoff list result: %w", err)
		}
		return result, nil
	case "create":
		projectID := strings.TrimSpace(args.ProjectID)
		if projectID == "" {
			return mcp.NewToolResultError(`invalid_request: required argument "project_id" not found`), nil
		}
		summary := strings.TrimSpace(args.Summary)
		if summary == "" {
			return mcp.NewToolResultError(`invalid_request: required argument "summary" not found`), nil
		}
		scopeID := firstNonEmptyString(strings.TrimSpace(args.ScopeID), projectID)
		caller, err := authorizeMCPMutation(
			ctx,
			pickMutationAuthorizer(handoffs),
			mcpSessionAuthArgs{
				SessionID:     args.SessionID,
				SessionSecret: args.SessionSecret,
			},
			"create_handoff",
			"project:"+projectID,
			"handoff",
			scopeID,
			map[string]string{
				"project_id": projectID,
				"scope_type": strings.TrimSpace(args.ScopeType),
				"scope_id":   scopeID,
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
		handoff, err := handoffs.CreateHandoff(ctx, common.CreateHandoffRequest{
			ProjectID:       projectID,
			BranchID:        strings.TrimSpace(args.BranchID),
			ScopeType:       strings.TrimSpace(args.ScopeType),
			ScopeID:         strings.TrimSpace(args.ScopeID),
			SourceRole:      strings.TrimSpace(args.SourceRole),
			TargetBranchID:  strings.TrimSpace(args.TargetBranchID),
			TargetScopeType: strings.TrimSpace(args.TargetScopeType),
			TargetScopeID:   strings.TrimSpace(args.TargetScopeID),
			TargetRole:      strings.TrimSpace(args.TargetRole),
			Status:          strings.TrimSpace(args.Status),
			Summary:         summary,
			NextAction:      strings.TrimSpace(args.NextAction),
			MissingEvidence: append([]string(nil), args.MissingEvidence...),
			RelatedRefs:     append([]string(nil), args.RelatedRefs...),
			Actor:           actor,
		})
		if err != nil {
			return toolResultFromError(err), nil
		}
		result, err := mcp.NewToolResultJSON(handoff)
		if err != nil {
			return nil, fmt.Errorf("encode handoff create result: %w", err)
		}
		return result, nil
	case "update":
		handoffID := strings.TrimSpace(args.HandoffID)
		if handoffID == "" {
			return mcp.NewToolResultError(`invalid_request: required argument "handoff_id" not found`), nil
		}
		summary := strings.TrimSpace(args.Summary)
		if summary == "" {
			return mcp.NewToolResultError(`invalid_request: required argument "summary" not found`), nil
		}
		caller, err := authorizeMCPMutation(
			ctx,
			pickMutationAuthorizer(handoffs),
			mcpSessionAuthArgs{
				SessionID:     args.SessionID,
				SessionSecret: args.SessionSecret,
			},
			"update_handoff",
			"tillsyn",
			"handoff",
			handoffID,
			map[string]string{"handoff_id": handoffID},
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
		handoff, err := handoffs.UpdateHandoff(ctx, common.UpdateHandoffRequest{
			HandoffID:       handoffID,
			Status:          strings.TrimSpace(args.Status),
			SourceRole:      strings.TrimSpace(args.SourceRole),
			TargetBranchID:  strings.TrimSpace(args.TargetBranchID),
			TargetScopeType: strings.TrimSpace(args.TargetScopeType),
			TargetScopeID:   strings.TrimSpace(args.TargetScopeID),
			TargetRole:      strings.TrimSpace(args.TargetRole),
			Summary:         summary,
			NextAction:      strings.TrimSpace(args.NextAction),
			MissingEvidence: append([]string(nil), args.MissingEvidence...),
			RelatedRefs:     append([]string(nil), args.RelatedRefs...),
			ResolutionNote:  strings.TrimSpace(args.ResolutionNote),
			Actor:           actor,
		})
		if err != nil {
			return toolResultFromError(err), nil
		}
		result, err := mcp.NewToolResultJSON(handoff)
		if err != nil {
			return nil, fmt.Errorf("encode handoff update result: %w", err)
		}
		return result, nil
	default:
		return mcp.NewToolResultError(`invalid_request: required argument "operation" not found`), nil
	}
}
