// Package mcpapi provides stateless HTTP and stdio MCP adapters.
package mcpapi

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/charmbracelet/log"
	"github.com/hylla/tillsyn/internal/adapters/server/common"
	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

// Config captures MCP transport configuration.
type Config struct {
	ServerName                    string
	ServerVersion                 string
	EndpointPath                  string
	ExposeLegacyLeaseTools        bool
	ExposeLegacyCoordinationTools bool
	ExposeLegacyProjectTools      bool
	ExposeLegacyPlanItemTools     bool
}

// Handler wraps one stateless MCP streamable HTTP handler.
type Handler struct {
	httpHandler http.Handler
}

// authRequestCreateResult keeps create-time resume ownership proof available to the requester
// without exposing private continuation metadata in general auth-request inventory reads.
type authRequestCreateResult struct {
	common.AuthRequestRecord
	ResumeToken string `json:"resume_token,omitempty"`
}

// NewServer builds one MCP server with the full tillsyn tool surface.
func NewServer(cfg Config, captureState common.CaptureStateReader, attention common.AttentionService) (*mcpserver.MCPServer, Config, error) {
	if captureState == nil {
		return nil, Config{}, fmt.Errorf("capture_state service is required")
	}
	cfg = normalizeConfig(cfg)

	mcpSrv := mcpserver.NewMCPServer(
		cfg.ServerName,
		cfg.ServerVersion,
		mcpserver.WithToolCapabilities(false),
	)
	registerCaptureStateTool(mcpSrv, captureState)
	if attention != nil {
		registerAttentionTools(mcpSrv, attention, cfg.ExposeLegacyCoordinationTools)
	}
	registerAuthRequestTools(mcpSrv, pickAuthRequestService(captureState, attention))
	registerBootstrapTool(mcpSrv, pickBootstrapGuideReader(captureState, attention))
	registerInstructionsTool(mcpSrv)
	registerProjectTools(
		mcpSrv,
		pickProjectService(captureState, attention),
		pickKindCatalogService(captureState, attention),
		pickTemplateLibraryService(captureState, attention),
		pickChangeFeedService(captureState, attention),
		cfg.ExposeLegacyProjectTools,
	)
	registerTaskTools(
		mcpSrv,
		pickTaskService(captureState, attention),
		pickSearchService(captureState, attention),
		pickEmbeddingsService(captureState, attention),
		cfg.ExposeLegacyPlanItemTools,
	)
	registerKindTools(mcpSrv, pickKindCatalogService(captureState, attention), cfg.ExposeLegacyProjectTools)
	registerTemplateLibraryTools(mcpSrv, pickTemplateLibraryService(captureState, attention), cfg.ExposeLegacyProjectTools)
	registerCapabilityLeaseTools(mcpSrv, pickCapabilityLeaseService(captureState, attention), cfg.ExposeLegacyLeaseTools)
	registerCommentTools(mcpSrv, pickCommentService(captureState, attention))
	registerHandoffTools(mcpSrv, pickHandoffService(captureState, attention), cfg.ExposeLegacyCoordinationTools)
	return mcpSrv, cfg, nil
}

// registerAuthRequestTools registers optional pre-session auth-request tools for MCP callers.
func registerAuthRequestTools(srv *mcpserver.MCPServer, authRequests common.AuthRequestService) {
	if authRequests == nil {
		return
	}
	srv.AddTool(
		mcp.NewTool(
			"till.auth_request",
			mcp.WithDescription("Create, inspect, or resume one auth-request lifecycle operation. Use operation=create|list|get|claim|cancel."),
			mcp.WithString("operation", mcp.Required(), mcp.Description("Auth-request operation"), mcp.Enum("create", "list", "get", "claim", "cancel")),
			mcp.WithString("project_id", mcp.Description("Optional project identifier filter")),
			mcp.WithString("state", mcp.Description("Optional request state filter"), mcp.Enum("pending", "approved", "denied", "canceled", "expired")),
			mcp.WithNumber("limit", mcp.Description("Optional maximum rows to return")),
			mcp.WithString("path", mcp.Description("Required for operation=create. Auth scope path: project/<project-id>[/branch/<branch-id>[/phase/<phase-id>...]] | projects/<project-id>,<project-id>... | global")),
			mcp.WithString("principal_id", mcp.Description("Required for operation=create|claim|cancel. Requested or requester principal identifier depending on operation")),
			mcp.WithString("principal_type", mcp.Description("Requested principal type for operation=create"), mcp.Enum("user", "agent", "service")),
			mcp.WithString("principal_role", mcp.Description("Optional requested agent role for operation=create"), mcp.Enum("orchestrator", "builder", "qa")),
			mcp.WithString("principal_name", mcp.Description("Optional principal display name for operation=create")),
			mcp.WithString("requested_by_actor", mcp.Description("Optional requester actor identifier for operation=create")),
			mcp.WithString("requested_by_type", mcp.Description("Optional requester actor type for operation=create"), mcp.Enum("user", "agent", "system")),
			mcp.WithString("requester_client_id", mcp.Description("Optional requester client identifier for operation=create")),
			mcp.WithString("client_id", mcp.Description("Required for operation=create|claim|cancel. Requesting or requester client identifier depending on operation")),
			mcp.WithString("client_type", mcp.Description("Requesting client type for operation=create")),
			mcp.WithString("client_name", mcp.Description("Optional client display name for operation=create")),
			mcp.WithString("requested_ttl", mcp.Description("Optional approved-session lifetime override for operation=create, for example 2h")),
			mcp.WithString("timeout", mcp.Description("Optional pending-request timeout for operation=create, for example 30m")),
			mcp.WithString("reason", mcp.Description("Required for operation=create. Human-readable reason shown to the approving user")),
			mcp.WithString("continuation_json", mcp.Description("Optional JSON continuation payload for operation=create. If omitted, till.auth_request auto-generates a requester-owned resume_token and returns it in the create result. If provided for MCP claim/cancel flows, continuation_json.resume_token must be a non-empty string.")),
			mcp.WithString("request_id", mcp.Description("Auth request identifier. Required for operation=get|claim|cancel")),
			mcp.WithString("resume_token", mcp.Description("Requester-owned resume token. Required for operation=claim|cancel. Use the token returned by operation=create when continuation_json was omitted.")),
			mcp.WithString("wait_timeout", mcp.Description("Optional how long to wait for human approval before returning the current request state, for example 30m")),
			mcp.WithString("resolution_note", mcp.Description("Optional requester-visible note explaining why the pending request was withdrawn")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var args struct {
				Operation         string `json:"operation"`
				ProjectID         string `json:"project_id"`
				State             string `json:"state"`
				Limit             int    `json:"limit"`
				Path              string `json:"path"`
				PrincipalID       string `json:"principal_id"`
				PrincipalType     string `json:"principal_type"`
				PrincipalRole     string `json:"principal_role"`
				PrincipalName     string `json:"principal_name"`
				RequestedByActor  string `json:"requested_by_actor"`
				RequestedByType   string `json:"requested_by_type"`
				RequesterClientID string `json:"requester_client_id"`
				ClientID          string `json:"client_id"`
				ClientType        string `json:"client_type"`
				ClientName        string `json:"client_name"`
				RequestedTTL      string `json:"requested_ttl"`
				Timeout           string `json:"timeout"`
				Reason            string `json:"reason"`
				ContinuationJSON  string `json:"continuation_json"`
				RequestID         string `json:"request_id"`
				ResumeToken       string `json:"resume_token"`
				WaitTimeout       string `json:"wait_timeout"`
				ResolutionNote    string `json:"resolution_note"`
			}
			if err := req.BindArguments(&args); err != nil {
				return invalidRequestToolResult(err), nil
			}
			switch strings.TrimSpace(args.Operation) {
			case "create":
				if strings.TrimSpace(args.Path) == "" {
					return mcp.NewToolResultError(`invalid_request: required argument "path" not found`), nil
				}
				if strings.TrimSpace(args.PrincipalID) == "" {
					return mcp.NewToolResultError(`invalid_request: required argument "principal_id" not found`), nil
				}
				if strings.TrimSpace(args.ClientID) == "" {
					return mcp.NewToolResultError(`invalid_request: required argument "client_id" not found`), nil
				}
				if strings.TrimSpace(args.Reason) == "" {
					return mcp.NewToolResultError(`invalid_request: required argument "reason" not found`), nil
				}
				record, err := authRequests.CreateAuthRequest(ctx, common.CreateAuthRequestRequest{
					Path:              args.Path,
					PrincipalID:       args.PrincipalID,
					PrincipalType:     args.PrincipalType,
					PrincipalRole:     args.PrincipalRole,
					PrincipalName:     args.PrincipalName,
					RequestedByActor:  args.RequestedByActor,
					RequestedByType:   args.RequestedByType,
					RequesterClientID: args.RequesterClientID,
					ClientID:          args.ClientID,
					ClientType:        args.ClientType,
					ClientName:        args.ClientName,
					RequestedTTL:      args.RequestedTTL,
					Timeout:           args.Timeout,
					Reason:            args.Reason,
					ContinuationJSON:  args.ContinuationJSON,
				})
				if err != nil {
					return toolResultFromError(err), nil
				}
				resumeToken, _ := record.Continuation["resume_token"].(string)
				result, err := mcp.NewToolResultJSON(authRequestCreateResult{
					AuthRequestRecord: record,
					ResumeToken:       strings.TrimSpace(resumeToken),
				})
				if err != nil {
					return nil, fmt.Errorf("encode auth_request create result: %w", err)
				}
				return result, nil
			case "list":
				requests, err := authRequests.ListAuthRequests(ctx, common.ListAuthRequestsRequest{
					ProjectID: args.ProjectID,
					State:     args.State,
					Limit:     args.Limit,
				})
				if err != nil {
					return toolResultFromError(err), nil
				}
				result, err := mcp.NewToolResultJSON(map[string]any{"requests": requests})
				if err != nil {
					return nil, fmt.Errorf("encode auth_request list result: %w", err)
				}
				return result, nil
			case "get":
				requestID := strings.TrimSpace(args.RequestID)
				if requestID == "" {
					return mcp.NewToolResultError(`invalid_request: required argument "request_id" not found`), nil
				}
				record, err := authRequests.GetAuthRequest(ctx, requestID)
				if err != nil {
					return toolResultFromError(err), nil
				}
				result, err := mcp.NewToolResultJSON(record)
				if err != nil {
					return nil, fmt.Errorf("encode auth_request get result: %w", err)
				}
				return result, nil
			case "claim":
				requestID := strings.TrimSpace(args.RequestID)
				if requestID == "" {
					return mcp.NewToolResultError(`invalid_request: required argument "request_id" not found`), nil
				}
				resumeToken := strings.TrimSpace(args.ResumeToken)
				if resumeToken == "" {
					return mcp.NewToolResultError(`invalid_request: required argument "resume_token" not found`), nil
				}
				principalID := strings.TrimSpace(args.PrincipalID)
				if principalID == "" {
					return mcp.NewToolResultError(`invalid_request: required argument "principal_id" not found`), nil
				}
				clientID := strings.TrimSpace(args.ClientID)
				if clientID == "" {
					return mcp.NewToolResultError(`invalid_request: required argument "client_id" not found`), nil
				}
				if trimmed := strings.TrimSpace(args.WaitTimeout); trimmed != "" {
					parsed, parseErr := time.ParseDuration(trimmed)
					if parseErr != nil || parsed < 0 {
						return invalidRequestToolResult(fmt.Errorf("wait_timeout %q is invalid", trimmed)), nil
					}
				}
				record, err := authRequests.ClaimAuthRequest(ctx, common.ClaimAuthRequestRequest{
					RequestID:   requestID,
					ResumeToken: resumeToken,
					PrincipalID: principalID,
					ClientID:    clientID,
					WaitTimeout: args.WaitTimeout,
				})
				if err != nil {
					return toolResultFromError(err), nil
				}
				result, err := mcp.NewToolResultJSON(record)
				if err != nil {
					return nil, fmt.Errorf("encode auth_request claim result: %w", err)
				}
				return result, nil
			case "cancel":
				requestID := strings.TrimSpace(args.RequestID)
				if requestID == "" {
					return mcp.NewToolResultError(`invalid_request: required argument "request_id" not found`), nil
				}
				resumeToken := strings.TrimSpace(args.ResumeToken)
				if resumeToken == "" {
					return mcp.NewToolResultError(`invalid_request: required argument "resume_token" not found`), nil
				}
				principalID := strings.TrimSpace(args.PrincipalID)
				if principalID == "" {
					return mcp.NewToolResultError(`invalid_request: required argument "principal_id" not found`), nil
				}
				clientID := strings.TrimSpace(args.ClientID)
				if clientID == "" {
					return mcp.NewToolResultError(`invalid_request: required argument "client_id" not found`), nil
				}
				record, err := authRequests.CancelAuthRequest(ctx, common.CancelAuthRequestRequest{
					RequestID:      requestID,
					ResumeToken:    resumeToken,
					PrincipalID:    principalID,
					ClientID:       clientID,
					ResolutionNote: args.ResolutionNote,
				})
				if err != nil {
					return toolResultFromError(err), nil
				}
				result, err := mcp.NewToolResultJSON(record)
				if err != nil {
					return nil, fmt.Errorf("encode auth_request cancel result: %w", err)
				}
				return result, nil
			default:
				return mcp.NewToolResultError(`invalid_request: required argument "operation" not found`), nil
			}
		},
	)
}

// NewHandler builds one stateless MCP streamable HTTP adapter with capture_state, attention, and optional app-backed tools.
func NewHandler(cfg Config, captureState common.CaptureStateReader, attention common.AttentionService) (*Handler, error) {
	mcpSrv, cfg, err := NewServer(cfg, captureState, attention)
	if err != nil {
		return nil, err
	}
	streamable := mcpserver.NewStreamableHTTPServer(
		mcpSrv,
		mcpserver.WithEndpointPath(cfg.EndpointPath),
		mcpserver.WithStateLess(true),
	)
	return &Handler{httpHandler: streamable}, nil
}

// ServeStdio starts one MCP server over stdio for local tool integrations.
func ServeStdio(cfg Config, captureState common.CaptureStateReader, attention common.AttentionService) error {
	mcpSrv, _, err := NewServer(cfg, captureState, attention)
	if err != nil {
		return err
	}
	return mcpserver.ServeStdio(mcpSrv)
}

// ServeHTTP handles one MCP streamable HTTP request.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.httpHandler == nil {
		http.Error(w, "mcp handler unavailable", http.StatusServiceUnavailable)
		return
	}
	h.httpHandler.ServeHTTP(w, r)
}

// normalizeConfig applies deterministic defaults to MCP adapter config.
func normalizeConfig(cfg Config) Config {
	cfg.ServerName = strings.TrimSpace(cfg.ServerName)
	if cfg.ServerName == "" {
		cfg.ServerName = "tillsyn"
	}
	cfg.ServerVersion = strings.TrimSpace(cfg.ServerVersion)
	if cfg.ServerVersion == "" {
		cfg.ServerVersion = "dev"
	}
	cfg.EndpointPath = strings.TrimSpace(cfg.EndpointPath)
	if cfg.EndpointPath == "" {
		cfg.EndpointPath = "/mcp"
	}
	if !strings.HasPrefix(cfg.EndpointPath, "/") {
		cfg.EndpointPath = "/" + cfg.EndpointPath
	}
	cfg.EndpointPath = "/" + strings.Trim(cfg.EndpointPath, "/")
	return cfg
}

// registerCaptureStateTool registers the `till.capture_state` tool.
func registerCaptureStateTool(srv *mcpserver.MCPServer, captureState common.CaptureStateReader) {
	srv.AddTool(
		mcp.NewTool(
			"till.capture_state",
			mcp.WithDescription("Return a summary-first state capture for one scoped level tuple."),
			mcp.WithString("project_id", mcp.Required(), mcp.Description("Project identifier")),
			mcp.WithString("scope_type", mcp.Description("Scope type"), mcp.Enum(common.SupportedScopeTypes()...)),
			mcp.WithString("scope_id", mcp.Description("Scope identifier (defaults to project_id)")),
			mcp.WithString("view", mcp.Description("summary or full"), mcp.Enum("summary", "full")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			projectID, err := req.RequireString("project_id")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			capture, err := captureState.CaptureState(ctx, common.CaptureStateRequest{
				ProjectID: projectID,
				ScopeType: req.GetString("scope_type", ""),
				ScopeID:   req.GetString("scope_id", ""),
				View:      req.GetString("view", ""),
			})
			if err != nil {
				return toolResultFromError(err), nil
			}
			result, err := mcp.NewToolResultJSON(capture)
			if err != nil {
				return nil, fmt.Errorf("encode capture_state result: %w", err)
			}
			return result, nil
		},
	)
}

type attentionItemMutationArgs struct {
	Operation          string `json:"operation"`
	ProjectID          string `json:"project_id"`
	ScopeType          string `json:"scope_type"`
	ScopeID            string `json:"scope_id"`
	State              string `json:"state"`
	Kind               string `json:"kind"`
	Summary            string `json:"summary"`
	BodyMarkdown       string `json:"body_markdown"`
	RequiresUserAction bool   `json:"requires_user_action"`
	ID                 string `json:"id"`
	Reason             string `json:"reason"`
	SessionID          string `json:"session_id"`
	SessionSecret      string `json:"session_secret"`
	AgentInstanceID    string `json:"agent_instance_id"`
	LeaseToken         string `json:"lease_token"`
	OverrideToken      string `json:"override_token"`
}

// registerAttentionTools registers optional attention list/raise/resolve tools.
func registerAttentionTools(srv *mcpserver.MCPServer, attention common.AttentionService, exposeLegacyCoordinationTools bool) {
	srv.AddTool(
		mcp.NewTool(
			"till.attention_item",
			mcp.WithDescription("Create, resolve, or list attention items."),
			mcp.WithString("operation",
				mcp.Required(),
				mcp.Enum("list", "raise", "resolve"),
				mcp.Description("Attention item operation"),
			),
			mcp.WithString("project_id", mcp.Description("Project identifier. Required for operation=list|raise")),
			mcp.WithString("scope_type", mcp.Description("Scope type. Optional for operation=list and required for operation=raise")),
			mcp.WithString("scope_id", mcp.Description("Scope identifier. Optional for operation=list and required for operation=raise")),
			mcp.WithString("state", mcp.Description("Filter by state when operation=list")),
			mcp.WithString("kind", mcp.Description("Attention kind. Required for operation=raise")),
			mcp.WithString("summary", mcp.Description("Markdown-rich summary for quick triage. Required for operation=raise")),
			mcp.WithString("body_markdown", mcp.Description("Optional markdown-rich details for deeper context when operation=raise")),
			mcp.WithBoolean("requires_user_action", mcp.Description("Whether this item blocks on user action when operation=raise")),
			mcp.WithString("id", mcp.Description("Attention item id. Required for operation=resolve")),
			mcp.WithString("reason", mcp.Description("Resolution reason when operation=resolve")),
			mcp.WithString("session_id", mcp.Description("Required for operation=raise|resolve. "+mcpMutationSessionDescription)),
			mcp.WithString("session_secret", mcp.Description("Required for operation=raise|resolve. "+mcpMutationSessionSecretDescription)),
			mcp.WithString("agent_instance_id", mcp.Description("Optional agent lease instance id for secondary local guard checks")),
			mcp.WithString("lease_token", mcp.Description("Optional agent lease token for secondary local guard checks")),
			mcp.WithString("override_token", mcp.Description("Optional override token for secondary local guard checks")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var args attentionItemMutationArgs
			if err := req.BindArguments(&args); err != nil {
				return invalidRequestToolResult(err), nil
			}
			return handleAttentionItemMutation(ctx, attention, args)
		},
	)

	if exposeLegacyCoordinationTools {
		registerLegacyAttentionListTool(srv, attention)
		registerLegacyAttentionMutationTools(srv, attention)
	}
}

func registerLegacyAttentionListTool(srv *mcpserver.MCPServer, attention common.AttentionService) {
	srv.AddTool(
		mcp.NewTool(
			"till.list_attention_items",
			mcp.WithDescription("List attention items for a project scope."),
			mcp.WithString("project_id", mcp.Required(), mcp.Description("Project identifier")),
			mcp.WithString("scope_type", mcp.Description("Scope type")),
			mcp.WithString("scope_id", mcp.Description("Scope identifier")),
			mcp.WithString("state", mcp.Description("Filter by state")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var args attentionItemMutationArgs
			if err := req.BindArguments(&args); err != nil {
				return invalidRequestToolResult(err), nil
			}
			args.Operation = "list"
			return handleAttentionItemMutation(ctx, attention, args)
		},
	)
}

func registerLegacyAttentionMutationTools(srv *mcpserver.MCPServer, attention common.AttentionService) {
	srv.AddTool(
		mcp.NewTool(
			"till.raise_attention_item",
			mcp.WithDescription("Create a new attention item with markdown-rich summary/details for a project scope."),
			mcp.WithString("project_id", mcp.Required(), mcp.Description("Project identifier")),
			mcp.WithString("scope_type", mcp.Required(), mcp.Description("Scope type")),
			mcp.WithString("scope_id", mcp.Required(), mcp.Description("Scope identifier")),
			mcp.WithString("kind", mcp.Required(), mcp.Description("Attention kind")),
			mcp.WithString("summary", mcp.Required(), mcp.Description("Markdown-rich summary for quick triage")),
			mcp.WithString("body_markdown", mcp.Description("Optional markdown-rich details for deeper context")),
			mcp.WithBoolean("requires_user_action", mcp.Description("Whether this item blocks on user action")),
			mcp.WithString("session_id", mcp.Required(), mcp.Description(mcpMutationSessionDescription)),
			mcp.WithString("session_secret", mcp.Required(), mcp.Description(mcpMutationSessionSecretDescription)),
			mcp.WithString("agent_instance_id", mcp.Description("Optional agent lease instance id for secondary local guard checks")),
			mcp.WithString("lease_token", mcp.Description("Optional agent lease token for secondary local guard checks")),
			mcp.WithString("override_token", mcp.Description("Optional override token for secondary local guard checks")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var args attentionItemMutationArgs
			if err := req.BindArguments(&args); err != nil {
				return invalidRequestToolResult(err), nil
			}
			args.Operation = "raise"
			return handleAttentionItemMutation(ctx, attention, args)
		},
	)

	srv.AddTool(
		mcp.NewTool(
			"till.resolve_attention_item",
			mcp.WithDescription("Resolve one attention item by id."),
			mcp.WithString("id", mcp.Required(), mcp.Description("Attention item id")),
			mcp.WithString("reason", mcp.Description("Resolution reason")),
			mcp.WithString("session_id", mcp.Required(), mcp.Description(mcpMutationSessionDescription)),
			mcp.WithString("session_secret", mcp.Required(), mcp.Description(mcpMutationSessionSecretDescription)),
			mcp.WithString("agent_instance_id", mcp.Description("Optional agent lease instance id for secondary local guard checks")),
			mcp.WithString("lease_token", mcp.Description("Optional agent lease token for secondary local guard checks")),
			mcp.WithString("override_token", mcp.Description("Optional override token for secondary local guard checks")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var args attentionItemMutationArgs
			if err := req.BindArguments(&args); err != nil {
				return invalidRequestToolResult(err), nil
			}
			args.Operation = "resolve"
			return handleAttentionItemMutation(ctx, attention, args)
		},
	)
}

func handleAttentionItemMutation(ctx context.Context, attention common.AttentionService, args attentionItemMutationArgs) (*mcp.CallToolResult, error) {
	operation := strings.TrimSpace(args.Operation)
	switch operation {
	case "list":
		projectID := strings.TrimSpace(args.ProjectID)
		if projectID == "" {
			return mcp.NewToolResultError(`invalid_request: required argument "project_id" not found`), nil
		}
		items, err := attention.ListAttentionItems(ctx, common.ListAttentionItemsRequest{
			ProjectID: projectID,
			ScopeType: strings.TrimSpace(args.ScopeType),
			ScopeID:   strings.TrimSpace(args.ScopeID),
			State:     strings.TrimSpace(args.State),
		})
		if err != nil {
			return toolResultFromError(err), nil
		}
		result, err := mcp.NewToolResultJSON(map[string]any{"items": items})
		if err != nil {
			return nil, fmt.Errorf("encode attention_item list result: %w", err)
		}
		return result, nil
	case "raise":
		projectID := strings.TrimSpace(args.ProjectID)
		if projectID == "" {
			return mcp.NewToolResultError(`invalid_request: required argument "project_id" not found`), nil
		}
		scopeType := strings.TrimSpace(args.ScopeType)
		if scopeType == "" {
			return mcp.NewToolResultError(`invalid_request: required argument "scope_type" not found`), nil
		}
		scopeID := strings.TrimSpace(args.ScopeID)
		if scopeID == "" {
			return mcp.NewToolResultError(`invalid_request: required argument "scope_id" not found`), nil
		}
		kind := strings.TrimSpace(args.Kind)
		if kind == "" {
			return mcp.NewToolResultError(`invalid_request: required argument "kind" not found`), nil
		}
		summary := strings.TrimSpace(args.Summary)
		if summary == "" {
			return mcp.NewToolResultError(`invalid_request: required argument "summary" not found`), nil
		}
		caller, err := authorizeMCPMutation(
			ctx,
			pickMutationAuthorizer(attention),
			mcpSessionAuthArgs{
				SessionID:     args.SessionID,
				SessionSecret: args.SessionSecret,
			},
			"raise_attention_item",
			"project:"+projectID,
			"attention_item",
			scopeID,
			map[string]string{
				"project_id": projectID,
				"scope_type": scopeType,
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
		item, err := attention.RaiseAttentionItem(ctx, common.RaiseAttentionItemRequest{
			ProjectID:          projectID,
			ScopeType:          scopeType,
			ScopeID:            scopeID,
			Kind:               kind,
			Summary:            summary,
			BodyMarkdown:       strings.TrimSpace(args.BodyMarkdown),
			RequiresUserAction: args.RequiresUserAction,
			Actor:              actor,
		})
		if err != nil {
			return toolResultFromError(err), nil
		}
		result, err := mcp.NewToolResultJSON(item)
		if err != nil {
			return nil, fmt.Errorf("encode attention_item raise result: %w", err)
		}
		return result, nil
	case "resolve":
		itemID := strings.TrimSpace(args.ID)
		if itemID == "" {
			return mcp.NewToolResultError(`invalid_request: required argument "id" not found`), nil
		}
		caller, err := authorizeMCPMutation(
			ctx,
			pickMutationAuthorizer(attention),
			mcpSessionAuthArgs{
				SessionID:     args.SessionID,
				SessionSecret: args.SessionSecret,
			},
			"resolve_attention_item",
			"tillsyn",
			"attention_item",
			itemID,
			map[string]string{"attention_id": itemID},
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
		item, err := attention.ResolveAttentionItem(ctx, common.ResolveAttentionItemRequest{
			ID:     itemID,
			Reason: strings.TrimSpace(args.Reason),
			Actor:  actor,
		})
		if err != nil {
			return toolResultFromError(err), nil
		}
		result, err := mcp.NewToolResultJSON(item)
		if err != nil {
			return nil, fmt.Errorf("encode attention_item resolve result: %w", err)
		}
		return result, nil
	default:
		return mcp.NewToolResultError(`invalid_request: required argument "operation" not found`), nil
	}
}

// toolResultFromError maps service errors into MCP-visible tool errors.
func toolResultFromError(err error) *mcp.CallToolResult {
	mapped := mapToolError(err)
	log.Error(
		"mcp tool error mapped",
		"transport",
		"mcp",
		"error_class",
		mapped.Class,
		"error_code",
		mapped.Code,
		"err",
		err,
	)
	return mcp.NewToolResultError(mapped.Text)
}

// toolErrorMapping captures one mapped MCP tool error classification and payload text.
type toolErrorMapping struct {
	Class string
	Code  string
	Text  string
}

// mapToolError converts one service error into MCP tool error metadata and response text.
func mapToolError(err error) toolErrorMapping {
	switch {
	case err == nil:
		return toolErrorMapping{
			Class: "internal",
			Code:  "internal_error",
			Text:  "unknown error",
		}
	case errors.Is(err, common.ErrBootstrapRequired):
		return toolErrorMapping{
			Class: "bootstrap",
			Code:  "bootstrap_required",
			Text:  "bootstrap_required: " + err.Error(),
		}
	case errors.Is(err, common.ErrGuardrailViolation):
		return toolErrorMapping{
			Class: "guardrail",
			Code:  "guardrail_failed",
			Text:  "guardrail_failed: " + err.Error(),
		}
	case errors.Is(err, common.ErrSessionRequired):
		return toolErrorMapping{
			Class: "auth",
			Code:  "session_required",
			Text:  "session_required: " + err.Error() + "; next step: call till.auth_request(operation=create) to request scoped access",
		}
	case errors.Is(err, common.ErrInvalidAuthentication):
		return toolErrorMapping{
			Class: "auth",
			Code:  "invalid_auth",
			Text:  "invalid_auth: " + err.Error(),
		}
	case errors.Is(err, common.ErrSessionExpired):
		return toolErrorMapping{
			Class: "auth",
			Code:  "session_expired",
			Text:  "session_expired: " + err.Error(),
		}
	case errors.Is(err, common.ErrAuthorizationDenied):
		return toolErrorMapping{
			Class: "auth",
			Code:  "auth_denied",
			Text:  "auth_denied: " + err.Error(),
		}
	case errors.Is(err, common.ErrGrantRequired):
		return toolErrorMapping{
			Class: "auth",
			Code:  "grant_required",
			Text:  "grant_required: " + err.Error() + "; next step: call till.auth_request(operation=create) or wait for approval on the existing request",
		}
	case errors.Is(err, common.ErrInvalidCaptureStateRequest), errors.Is(err, common.ErrUnsupportedScope):
		return toolErrorMapping{
			Class: "invalid",
			Code:  "invalid_request",
			Text:  "invalid_request: " + err.Error(),
		}
	case errors.Is(err, common.ErrNotFound):
		return toolErrorMapping{
			Class: "not_found",
			Code:  "not_found",
			Text:  "not_found: " + err.Error(),
		}
	case errors.Is(err, common.ErrAttentionUnavailable):
		return toolErrorMapping{
			Class: "not_implemented",
			Code:  "not_implemented",
			Text:  "not_implemented: " + err.Error(),
		}
	default:
		return toolErrorMapping{
			Class: "internal",
			Code:  "internal_error",
			Text:  "internal_error: " + err.Error(),
		}
	}
}

// pickBootstrapGuideReader resolves one bootstrap-guide provider from available services.
func pickBootstrapGuideReader(captureState common.CaptureStateReader, attention common.AttentionService) common.BootstrapGuideReader {
	if svc, ok := captureState.(common.BootstrapGuideReader); ok {
		return svc
	}
	if svc, ok := attention.(common.BootstrapGuideReader); ok {
		return svc
	}
	return nil
}

// pickProjectService resolves one project-service provider from available services.
func pickProjectService(captureState common.CaptureStateReader, attention common.AttentionService) common.ProjectService {
	if svc, ok := captureState.(common.ProjectService); ok {
		return svc
	}
	if svc, ok := attention.(common.ProjectService); ok {
		return svc
	}
	return nil
}

// pickTaskService resolves one task-service provider from available services.
func pickTaskService(captureState common.CaptureStateReader, attention common.AttentionService) common.TaskService {
	if svc, ok := captureState.(common.TaskService); ok {
		return svc
	}
	if svc, ok := attention.(common.TaskService); ok {
		return svc
	}
	return nil
}

// pickSearchService resolves one search-service provider from available services.
func pickSearchService(captureState common.CaptureStateReader, attention common.AttentionService) common.SearchService {
	if svc, ok := captureState.(common.SearchService); ok {
		return svc
	}
	if svc, ok := attention.(common.SearchService); ok {
		return svc
	}
	return nil
}

// pickChangeFeedService resolves one change-feed provider from available services.
func pickChangeFeedService(captureState common.CaptureStateReader, attention common.AttentionService) common.ChangeFeedService {
	if svc, ok := captureState.(common.ChangeFeedService); ok {
		return svc
	}
	if svc, ok := attention.(common.ChangeFeedService); ok {
		return svc
	}
	return nil
}

// pickEmbeddingsService resolves one embeddings-operator provider from available services.
func pickEmbeddingsService(captureState common.CaptureStateReader, attention common.AttentionService) common.EmbeddingsService {
	if svc, ok := captureState.(common.EmbeddingsService); ok {
		return svc
	}
	if svc, ok := attention.(common.EmbeddingsService); ok {
		return svc
	}
	return nil
}

// pickKindCatalogService resolves one kind-catalog provider from available services.
func pickKindCatalogService(captureState common.CaptureStateReader, attention common.AttentionService) common.KindCatalogService {
	if svc, ok := captureState.(common.KindCatalogService); ok {
		return svc
	}
	if svc, ok := attention.(common.KindCatalogService); ok {
		return svc
	}
	return nil
}

// pickCapabilityLeaseService resolves one lease-service provider from available services.
func pickCapabilityLeaseService(captureState common.CaptureStateReader, attention common.AttentionService) common.CapabilityLeaseService {
	if svc, ok := captureState.(common.CapabilityLeaseService); ok {
		return svc
	}
	if svc, ok := attention.(common.CapabilityLeaseService); ok {
		return svc
	}
	return nil
}

// pickCommentService resolves one comment-service provider from available services.
func pickCommentService(captureState common.CaptureStateReader, attention common.AttentionService) common.CommentService {
	if svc, ok := captureState.(common.CommentService); ok {
		return svc
	}
	if svc, ok := attention.(common.CommentService); ok {
		return svc
	}
	return nil
}

// pickHandoffService resolves one handoff-service provider from available services.
func pickHandoffService(captureState common.CaptureStateReader, attention common.AttentionService) common.HandoffService {
	if svc, ok := captureState.(common.HandoffService); ok {
		return svc
	}
	if svc, ok := attention.(common.HandoffService); ok {
		return svc
	}
	return nil
}

// pickAuthRequestService resolves one auth-request service provider from available services.
func pickAuthRequestService(captureState common.CaptureStateReader, attention common.AttentionService) common.AuthRequestService {
	if svc, ok := captureState.(common.AuthRequestService); ok {
		return svc
	}
	if svc, ok := attention.(common.AuthRequestService); ok {
		return svc
	}
	return nil
}

// pickTemplateLibraryService resolves one template-library service provider from available services.
func pickTemplateLibraryService(captureState common.CaptureStateReader, attention common.AttentionService) common.TemplateLibraryService {
	if svc, ok := captureState.(common.TemplateLibraryService); ok {
		return svc
	}
	if svc, ok := attention.(common.TemplateLibraryService); ok {
		return svc
	}
	return nil
}

// pickMutationAuthorizer resolves one mutation authorizer from any service that supports auth-backed writes.
func pickMutationAuthorizer(service any) common.MutationAuthorizer {
	authorizer, _ := service.(common.MutationAuthorizer)
	return authorizer
}
