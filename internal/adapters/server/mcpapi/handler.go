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
	ServerName    string
	ServerVersion string
	EndpointPath  string
}

// Handler wraps one stateless MCP streamable HTTP handler.
type Handler struct {
	httpHandler http.Handler
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
		registerAttentionTools(mcpSrv, attention)
	}
	registerAuthRequestTools(mcpSrv, pickAuthRequestService(captureState, attention))
	registerBootstrapTool(mcpSrv, pickBootstrapGuideReader(captureState, attention))
	registerInstructionsTool(mcpSrv)
	registerProjectTools(mcpSrv, pickProjectService(captureState, attention))
	registerTaskTools(
		mcpSrv,
		pickTaskService(captureState, attention),
		pickSearchService(captureState, attention),
		pickEmbeddingsService(captureState, attention),
		pickChangeFeedService(captureState, attention),
	)
	registerKindTools(mcpSrv, pickKindCatalogService(captureState, attention))
	registerCapabilityLeaseTools(mcpSrv, pickCapabilityLeaseService(captureState, attention))
	registerCommentTools(mcpSrv, pickCommentService(captureState, attention))
	registerHandoffTools(mcpSrv, pickHandoffService(captureState, attention))
	return mcpSrv, cfg, nil
}

// registerAuthRequestTools registers optional pre-session auth-request tools for MCP callers.
func registerAuthRequestTools(srv *mcpserver.MCPServer, authRequests common.AuthRequestService) {
	if authRequests == nil {
		return
	}
	srv.AddTool(
		mcp.NewTool(
			"till.create_auth_request",
			mcp.WithDescription("Create one persisted pre-session auth request for MCP or local dogfooding. Include continuation_json with a requester-owned resume_token when the requester plans to resume through till.claim_auth_request after human approval. Set requested_by_actor/requested_by_type/requester_client_id when the requester differs from the requested principal/client, such as orchestrator-on-behalf-of-builder or qa flows."),
			mcp.WithString("path", mcp.Required(), mcp.Description("Required auth scope path: project/<project-id>[/branch/<branch-id>[/phase/<phase-id>...]] | projects/<project-id>,<project-id>... | global")),
			mcp.WithString("principal_id", mcp.Required(), mcp.Description("Requested principal identifier")),
			mcp.WithString("principal_type", mcp.Description("Requested principal type"), mcp.Enum("user", "agent", "service")),
			mcp.WithString("principal_role", mcp.Description("Optional requested agent role"), mcp.Enum("orchestrator", "builder", "qa")),
			mcp.WithString("principal_name", mcp.Description("Optional principal display name")),
			mcp.WithString("requested_by_actor", mcp.Description("Optional requester actor identifier when one orchestrator requests auth on behalf of another principal")),
			mcp.WithString("requested_by_type", mcp.Description("Optional requester actor type"), mcp.Enum("user", "agent", "system")),
			mcp.WithString("requester_client_id", mcp.Description("Optional requester client identifier when the claimant differs from the requested client")),
			mcp.WithString("client_id", mcp.Required(), mcp.Description("Requesting client identifier")),
			mcp.WithString("client_type", mcp.Description("Requesting client type")),
			mcp.WithString("client_name", mcp.Description("Optional client display name")),
			mcp.WithString("requested_ttl", mcp.Description("Optional approved-session lifetime override, for example 2h")),
			mcp.WithString("timeout", mcp.Description("Optional pending-request timeout, for example 30m")),
			mcp.WithString("reason", mcp.Required(), mcp.Description("Human-readable reason shown to the approving user")),
			mcp.WithString("continuation_json", mcp.Description("Optional JSON object string with client continuation metadata for post-approval resume")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var args struct {
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
			}
			if err := req.BindArguments(&args); err != nil {
				return invalidRequestToolResult(err), nil
			}
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
			result, err := mcp.NewToolResultJSON(record)
			if err != nil {
				return nil, fmt.Errorf("encode create_auth_request result: %w", err)
			}
			return result, nil
		},
	)

	srv.AddTool(
		mcp.NewTool(
			"till.list_auth_requests",
			mcp.WithDescription("List persisted pre-session auth requests. Inventory is global by default; use project_id to narrow one listing to a single project."),
			mcp.WithString("project_id", mcp.Description("Optional project identifier filter")),
			mcp.WithString("state", mcp.Description("Optional request state filter"), mcp.Enum("pending", "approved", "denied", "canceled", "expired")),
			mcp.WithNumber("limit", mcp.Description("Optional maximum rows to return")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			limitRaw := req.GetFloat("limit", 0)
			requests, err := authRequests.ListAuthRequests(ctx, common.ListAuthRequestsRequest{
				ProjectID: req.GetString("project_id", ""),
				State:     req.GetString("state", ""),
				Limit:     int(limitRaw),
			})
			if err != nil {
				return toolResultFromError(err), nil
			}
			result, err := mcp.NewToolResultJSON(map[string]any{"requests": requests})
			if err != nil {
				return nil, fmt.Errorf("encode list_auth_requests result: %w", err)
			}
			return result, nil
		},
	)

	srv.AddTool(
		mcp.NewTool(
			"till.get_auth_request",
			mcp.WithDescription("Show one persisted pre-session auth request by id."),
			mcp.WithString("request_id", mcp.Required(), mcp.Description("Auth request identifier")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			requestID, err := req.RequireString("request_id")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			record, err := authRequests.GetAuthRequest(ctx, requestID)
			if err != nil {
				return toolResultFromError(err), nil
			}
			result, err := mcp.NewToolResultJSON(record)
			if err != nil {
				return nil, fmt.Errorf("encode get_auth_request result: %w", err)
			}
			return result, nil
		},
	)

	srv.AddTool(
		mcp.NewTool(
			"till.claim_auth_request",
			mcp.WithDescription("Claim one auth request continuation result by request id, requester identity, and requester-owned resume token. wait_timeout can hold the caller in a human-review waiting state before returning the current request state."),
			mcp.WithString("request_id", mcp.Required(), mcp.Description("Auth request identifier")),
			mcp.WithString("resume_token", mcp.Required(), mcp.Description("Opaque requester-owned token stored in continuation_json when the request was created")),
			mcp.WithString("principal_id", mcp.Required(), mcp.Description("Requester principal identifier")),
			mcp.WithString("client_id", mcp.Required(), mcp.Description("Requester client identifier")),
			mcp.WithString("wait_timeout", mcp.Description("Optional how long to wait for human approval before returning the current request state, for example 30m")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			requestID, err := req.RequireString("request_id")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			resumeToken, err := req.RequireString("resume_token")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			principalID, err := req.RequireString("principal_id")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			clientID, err := req.RequireString("client_id")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			waitTimeout := req.GetString("wait_timeout", "")
			if trimmed := strings.TrimSpace(waitTimeout); trimmed != "" {
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
				WaitTimeout: waitTimeout,
			})
			if err != nil {
				return toolResultFromError(err), nil
			}
			result, err := mcp.NewToolResultJSON(record)
			if err != nil {
				return nil, fmt.Errorf("encode claim_auth_request result: %w", err)
			}
			return result, nil
		},
	)

	srv.AddTool(
		mcp.NewTool(
			"till.cancel_auth_request",
			mcp.WithDescription("Cancel one pending auth request by request id, requester identity, and requester-owned resume token. Use this to withdraw or clean up a stale request; this is distinct from reviewer denial."),
			mcp.WithString("request_id", mcp.Required(), mcp.Description("Auth request identifier")),
			mcp.WithString("resume_token", mcp.Required(), mcp.Description("Opaque requester-owned token stored in continuation_json when the request was created")),
			mcp.WithString("principal_id", mcp.Required(), mcp.Description("Requester principal identifier")),
			mcp.WithString("client_id", mcp.Required(), mcp.Description("Requester client identifier")),
			mcp.WithString("resolution_note", mcp.Description("Optional requester-visible note explaining why the pending request was withdrawn")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			requestID, err := req.RequireString("request_id")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			resumeToken, err := req.RequireString("resume_token")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			principalID, err := req.RequireString("principal_id")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			clientID, err := req.RequireString("client_id")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			record, err := authRequests.CancelAuthRequest(ctx, common.CancelAuthRequestRequest{
				RequestID:      requestID,
				ResumeToken:    resumeToken,
				PrincipalID:    principalID,
				ClientID:       clientID,
				ResolutionNote: req.GetString("resolution_note", ""),
			})
			if err != nil {
				return toolResultFromError(err), nil
			}
			result, err := mcp.NewToolResultJSON(record)
			if err != nil {
				return nil, fmt.Errorf("encode cancel_auth_request result: %w", err)
			}
			return result, nil
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

// registerAttentionTools registers optional attention list/raise/resolve tools.
func registerAttentionTools(srv *mcpserver.MCPServer, attention common.AttentionService) {
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
			projectID, err := req.RequireString("project_id")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			items, err := attention.ListAttentionItems(ctx, common.ListAttentionItemsRequest{
				ProjectID: projectID,
				ScopeType: req.GetString("scope_type", ""),
				ScopeID:   req.GetString("scope_id", ""),
				State:     req.GetString("state", ""),
			})
			if err != nil {
				return toolResultFromError(err), nil
			}
			result, err := mcp.NewToolResultJSON(map[string]any{
				"items": items,
			})
			if err != nil {
				return nil, fmt.Errorf("encode list_attention_items result: %w", err)
			}
			return result, nil
		},
	)

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
			var args struct {
				ProjectID          string `json:"project_id"`
				ScopeType          string `json:"scope_type"`
				ScopeID            string `json:"scope_id"`
				Kind               string `json:"kind"`
				Summary            string `json:"summary"`
				BodyMarkdown       string `json:"body_markdown"`
				RequiresUserAction bool   `json:"requires_user_action"`
				SessionID          string `json:"session_id"`
				SessionSecret      string `json:"session_secret"`
				AgentInstanceID    string `json:"agent_instance_id"`
				LeaseToken         string `json:"lease_token"`
				OverrideToken      string `json:"override_token"`
			}
			if err := req.BindArguments(&args); err != nil {
				return invalidRequestToolResult(err), nil
			}
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
			})
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
				return nil, fmt.Errorf("encode raise_attention_item result: %w", err)
			}
			return result, nil
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
			var args struct {
				ID              string `json:"id"`
				Reason          string `json:"reason"`
				SessionID       string `json:"session_id"`
				SessionSecret   string `json:"session_secret"`
				AgentInstanceID string `json:"agent_instance_id"`
				LeaseToken      string `json:"lease_token"`
				OverrideToken   string `json:"override_token"`
			}
			if err := req.BindArguments(&args); err != nil {
				return invalidRequestToolResult(err), nil
			}
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
			})
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
				return nil, fmt.Errorf("encode resolve_attention_item result: %w", err)
			}
			return result, nil
		},
	)
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
			Text:  "session_required: " + err.Error() + "; next step: call till.create_auth_request to request scoped access",
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
			Text:  "grant_required: " + err.Error() + "; next step: call till.create_auth_request or wait for approval on the existing request",
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

// pickMutationAuthorizer resolves one mutation authorizer from any service that supports auth-backed writes.
func pickMutationAuthorizer(service any) common.MutationAuthorizer {
	authorizer, _ := service.(common.MutationAuthorizer)
	return authorizer
}
