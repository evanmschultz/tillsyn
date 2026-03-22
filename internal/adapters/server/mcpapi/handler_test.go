package mcpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"

	charmLog "github.com/charmbracelet/log"
	"github.com/hylla/tillsyn/internal/adapters/server/common"
	"github.com/hylla/tillsyn/internal/domain"
	"github.com/mark3labs/mcp-go/mcp"
)

// stubCaptureStateReader provides deterministic capture-state responses for MCP tool tests.
type stubCaptureStateReader struct {
	captureState common.CaptureState
	err          error
	lastRequest  common.CaptureStateRequest
}

// CaptureState records the latest request and returns one fixture result.
func (s *stubCaptureStateReader) CaptureState(_ context.Context, req common.CaptureStateRequest) (common.CaptureState, error) {
	s.lastRequest = req
	if s.err != nil {
		return common.CaptureState{}, s.err
	}
	return s.captureState, nil
}

// stubAttentionService provides deterministic attention responses for MCP tool tests.
type stubAttentionService struct {
	stubMutationAuthorizer
	items          []common.AttentionItem
	raised         common.AttentionItem
	resolved       common.AttentionItem
	authRequests   []common.AuthRequestRecord
	authRequest    common.AuthRequestRecord
	listErr        error
	raiseErr       error
	resolveErr     error
	authRequestErr error
	lastList       common.ListAttentionItemsRequest
	lastRaise      common.RaiseAttentionItemRequest
	lastResolve    common.ResolveAttentionItemRequest
	lastCreateAuth common.CreateAuthRequestRequest
	lastListAuth   common.ListAuthRequestsRequest
	lastGetAuthID  string
}

// stubProjectService provides deterministic project responses for expanded MCP tool registration tests.
type stubProjectService struct {
	stubCaptureStateReader
	stubMutationAuthorizer
	projects            []domain.Project
	createResult        domain.Project
	updateResult        domain.Project
	listErr             error
	createErr           error
	updateErr           error
	lastIncludeArchived bool
	lastCreate          common.CreateProjectRequest
	lastUpdate          common.UpdateProjectRequest
}

// stubAuthRequestService provides deterministic auth-request responses for MCP tool tests.
type stubAuthRequestService struct {
	stubCaptureStateReader
	created     common.AuthRequestRecord
	requests    []common.AuthRequestRecord
	getResult   common.AuthRequestRecord
	claimResult common.AuthRequestClaimResult
	createErr   error
	listErr     error
	getErr      error
	claimErr    error
	lastCreate  common.CreateAuthRequestRequest
	lastList    common.ListAuthRequestsRequest
	lastGetID   string
	lastClaim   common.ClaimAuthRequestRequest
}

// stubMutationAuthorizer provides deterministic session-auth results for mutating MCP tool tests.
type stubMutationAuthorizer struct {
	authErr         error
	authCaller      domain.AuthenticatedCaller
	lastAuthRequest common.MutationAuthorizationRequest
}

// AuthorizeMutation records one auth request and returns one deterministic caller/error.
func (s *stubMutationAuthorizer) AuthorizeMutation(_ context.Context, req common.MutationAuthorizationRequest) (domain.AuthenticatedCaller, error) {
	s.lastAuthRequest = req
	if s.authErr != nil {
		return domain.AuthenticatedCaller{}, s.authErr
	}
	if strings.TrimSpace(req.SessionID) == "" || strings.TrimSpace(req.SessionSecret) == "" {
		return domain.AuthenticatedCaller{}, errors.Join(common.ErrSessionRequired, errors.New("missing session credentials"))
	}
	caller := domain.NormalizeAuthenticatedCaller(s.authCaller)
	if caller.IsZero() {
		caller = domain.AuthenticatedCaller{
			PrincipalID:   "agent-1",
			PrincipalName: "Agent One",
			PrincipalType: domain.ActorTypeAgent,
			SessionID:     strings.TrimSpace(req.SessionID),
		}
	}
	return caller, nil
}

// ListProjects returns deterministic project list rows.
func (s *stubProjectService) ListProjects(_ context.Context, includeArchived bool) ([]domain.Project, error) {
	s.lastIncludeArchived = includeArchived
	if s.listErr != nil {
		return nil, s.listErr
	}
	return append([]domain.Project(nil), s.projects...), nil
}

// CreateProject records and returns deterministic project creation results.
func (s *stubProjectService) CreateProject(_ context.Context, req common.CreateProjectRequest) (domain.Project, error) {
	s.lastCreate = req
	if s.createErr != nil {
		return domain.Project{}, s.createErr
	}
	return s.createResult, nil
}

// UpdateProject records and returns deterministic project update results.
func (s *stubProjectService) UpdateProject(_ context.Context, req common.UpdateProjectRequest) (domain.Project, error) {
	s.lastUpdate = req
	if s.updateErr != nil {
		return domain.Project{}, s.updateErr
	}
	return s.updateResult, nil
}

// CreateAuthRequest records and returns one deterministic auth-request row.
func (s *stubAuthRequestService) CreateAuthRequest(_ context.Context, req common.CreateAuthRequestRequest) (common.AuthRequestRecord, error) {
	s.lastCreate = req
	if s.createErr != nil {
		return common.AuthRequestRecord{}, s.createErr
	}
	return s.created, nil
}

// ListAuthRequests records and returns deterministic auth-request rows.
func (s *stubAuthRequestService) ListAuthRequests(_ context.Context, req common.ListAuthRequestsRequest) ([]common.AuthRequestRecord, error) {
	s.lastList = req
	if s.listErr != nil {
		return nil, s.listErr
	}
	return append([]common.AuthRequestRecord(nil), s.requests...), nil
}

// GetAuthRequest records and returns one deterministic auth-request row.
func (s *stubAuthRequestService) GetAuthRequest(_ context.Context, requestID string) (common.AuthRequestRecord, error) {
	s.lastGetID = requestID
	if s.getErr != nil {
		return common.AuthRequestRecord{}, s.getErr
	}
	return s.getResult, nil
}

// ClaimAuthRequest records one continuation claim and returns one deterministic claim result.
func (s *stubAuthRequestService) ClaimAuthRequest(_ context.Context, req common.ClaimAuthRequestRequest) (common.AuthRequestClaimResult, error) {
	s.lastClaim = req
	if s.claimErr != nil {
		return common.AuthRequestClaimResult{}, s.claimErr
	}
	return s.claimResult, nil
}

// ListAttentionItems returns deterministic list data.
func (s *stubAttentionService) ListAttentionItems(_ context.Context, req common.ListAttentionItemsRequest) ([]common.AttentionItem, error) {
	s.lastList = req
	if s.listErr != nil {
		return nil, s.listErr
	}
	return append([]common.AttentionItem(nil), s.items...), nil
}

// RaiseAttentionItem records and returns one fixture item.
func (s *stubAttentionService) RaiseAttentionItem(_ context.Context, req common.RaiseAttentionItemRequest) (common.AttentionItem, error) {
	s.lastRaise = req
	if s.raiseErr != nil {
		return common.AttentionItem{}, s.raiseErr
	}
	return s.raised, nil
}

// ResolveAttentionItem records and returns one fixture item.
func (s *stubAttentionService) ResolveAttentionItem(_ context.Context, req common.ResolveAttentionItemRequest) (common.AttentionItem, error) {
	s.lastResolve = req
	if s.resolveErr != nil {
		return common.AttentionItem{}, s.resolveErr
	}
	return s.resolved, nil
}

// CreateAuthRequest records and returns one deterministic auth request row.
func (s *stubAttentionService) CreateAuthRequest(_ context.Context, req common.CreateAuthRequestRequest) (common.AuthRequestRecord, error) {
	s.lastCreateAuth = req
	if s.authRequestErr != nil {
		return common.AuthRequestRecord{}, s.authRequestErr
	}
	if s.authRequest.ID != "" {
		return s.authRequest, nil
	}
	return common.AuthRequestRecord{}, nil
}

// ListAuthRequests records list filters and returns deterministic auth request rows.
func (s *stubAttentionService) ListAuthRequests(_ context.Context, req common.ListAuthRequestsRequest) ([]common.AuthRequestRecord, error) {
	s.lastListAuth = req
	if s.authRequestErr != nil {
		return nil, s.authRequestErr
	}
	return append([]common.AuthRequestRecord(nil), s.authRequests...), nil
}

// GetAuthRequest records the requested id and returns one deterministic auth request row.
func (s *stubAttentionService) GetAuthRequest(_ context.Context, requestID string) (common.AuthRequestRecord, error) {
	s.lastGetAuthID = requestID
	if s.authRequestErr != nil {
		return common.AuthRequestRecord{}, s.authRequestErr
	}
	return s.authRequest, nil
}

// jsonRPCResponse models minimal JSON-RPC response fields used in MCP adapter tests.
type jsonRPCResponse struct {
	ID     float64        `json:"id"`
	Result map[string]any `json:"result"`
	Error  map[string]any `json:"error"`
}

// callToolRequest constructs one deterministic tools/call JSON-RPC request payload.
func callToolRequest(id int, toolName string, arguments map[string]any) map[string]any {
	return map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"method":  "tools/call",
		"params": map[string]any{
			"name":      toolName,
			"arguments": arguments,
		},
	}
}

// validSessionArgs returns one deterministic auth session argument set for mutating tool calls.
func validSessionArgs() map[string]any {
	return map[string]any{
		"session_id":     "sess-1",
		"session_secret": "secret-1",
	}
}

// mergeArgs returns one shallow-merged copy of multiple argument maps.
func mergeArgs(maps ...map[string]any) map[string]any {
	out := map[string]any{}
	for _, input := range maps {
		for key, value := range input {
			out[key] = value
		}
	}
	return out
}

// ptrTime returns one heap-stable copy of the input time.
func ptrTime(ts time.Time) *time.Time {
	return &ts
}

// toolResultText decodes the first text entry from one tool-call result payload.
func toolResultText(t *testing.T, result map[string]any) string {
	t.Helper()

	contentRaw, ok := result["content"].([]any)
	if !ok || len(contentRaw) == 0 {
		t.Fatalf("content missing in tool result: %#v", result)
	}
	first, ok := contentRaw[0].(map[string]any)
	if !ok {
		t.Fatalf("first content entry has unexpected type: %#v", contentRaw[0])
	}
	text, ok := first["text"].(string)
	if !ok {
		t.Fatalf("content text missing in tool result: %#v", first)
	}
	return text
}

// toolResultStructured decodes structuredContent as one map for stable assertions.
func toolResultStructured(t *testing.T, result map[string]any) map[string]any {
	t.Helper()
	structured, ok := result["structuredContent"].(map[string]any)
	if !ok {
		t.Fatalf("structuredContent missing in tool result: %#v", result)
	}
	return structured
}

// postJSONRPC sends one JSON-RPC payload and decodes the response body.
func postJSONRPC(t *testing.T, client *http.Client, url string, payload any) (*http.Response, jsonRPCResponse) {
	t.Helper()
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(body))
	if err != nil {
		t.Fatalf("NewRequest() error = %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}
	var decoded jsonRPCResponse
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if err := resp.Body.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	return resp, decoded
}

// initializeRequest builds a deterministic MCP initialize request payload.
func initializeRequest() map[string]any {
	return map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params": map[string]any{
			"protocolVersion": mcp.LATEST_PROTOCOL_VERSION,
			"clientInfo": map[string]any{
				"name":    "tillsyn-test",
				"version": "1.0.0",
			},
		},
	}
}

// callToolResultText decodes the first textual content block from a CallToolResult.
func callToolResultText(t *testing.T, result *mcp.CallToolResult) string {
	t.Helper()
	if result == nil {
		t.Fatalf("result = nil, want non-nil")
	}
	if len(result.Content) == 0 {
		t.Fatalf("result content is empty")
	}
	text, ok := result.Content[0].(mcp.TextContent)
	if !ok {
		t.Fatalf("content[0] has unexpected type %T", result.Content[0])
	}
	return text.Text
}

// captureDefaultLoggerOutput redirects package-level logging to one buffer for assertions.
func captureDefaultLoggerOutput(t *testing.T) (*bytes.Buffer, func()) {
	t.Helper()

	var output bytes.Buffer
	previous := charmLog.Default()
	charmLog.SetDefault(charmLog.NewWithOptions(&output, charmLog.Options{
		Level:           charmLog.DebugLevel,
		Formatter:       charmLog.LogfmtFormatter,
		ReportTimestamp: false,
	}))
	return &output, func() {
		charmLog.SetDefault(previous)
	}
}

// TestHandlerUsesStatelessTransport verifies MCP transport does not issue session ids.
func TestHandlerUsesStatelessTransport(t *testing.T) {
	capture := &stubCaptureStateReader{
		captureState: common.CaptureState{
			StateHash: "abc123",
		},
	}
	handler, err := NewHandler(Config{}, capture, nil)
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}

	server := httptest.NewServer(handler)
	defer server.Close()

	resp, decoded := postJSONRPC(t, server.Client(), server.URL, initializeRequest())
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	if decoded.ID != 1 {
		t.Fatalf("id = %v, want 1", decoded.ID)
	}
	if got := resp.Header.Get("Mcp-Session-Id"); got != "" {
		t.Fatalf("Mcp-Session-Id header = %q, want empty (stateless transport)", got)
	}
}

// TestHandlerRegistersCaptureStateTool verifies MCP tool discovery includes till.capture_state.
func TestHandlerRegistersCaptureStateTool(t *testing.T) {
	capture := &stubCaptureStateReader{
		captureState: common.CaptureState{
			StateHash: "abc123",
		},
	}
	handler, err := NewHandler(Config{}, capture, nil)
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
	if !slices.Contains(toolNames, "till.capture_state") {
		t.Fatalf("tool list missing till.capture_state: %#v", toolNames)
	}
	if slices.Contains(toolNames, "till.list_attention_items") {
		t.Fatalf("unexpected attention tool without attention service: %#v", toolNames)
	}
}

// TestHandlerRegistersAttentionToolsWhenAvailable verifies optional attention tools are exposed.
func TestHandlerRegistersAttentionToolsWhenAvailable(t *testing.T) {
	capture := &stubCaptureStateReader{
		captureState: common.CaptureState{
			StateHash: "abc123",
		},
	}
	attention := &stubAttentionService{}
	handler, err := NewHandler(Config{}, capture, attention)
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
	for _, required := range []string{
		"till.capture_state",
		"till.list_attention_items",
		"till.raise_attention_item",
		"till.resolve_attention_item",
	} {
		if !slices.Contains(toolNames, required) {
			t.Fatalf("tool list missing %q: %#v", required, toolNames)
		}
	}
}

// TestHandlerRegistersAuthRequestToolsWhenAvailable verifies optional auth-request tools register when the service exposes that surface.
func TestHandlerRegistersAuthRequestToolsWhenAvailable(t *testing.T) {
	capture := &stubAuthRequestService{
		stubCaptureStateReader: stubCaptureStateReader{
			captureState: common.CaptureState{StateHash: "abc123"},
		},
	}
	handler, err := NewHandler(Config{}, capture, nil)
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
	for _, required := range []string{
		"till.create_auth_request",
		"till.list_auth_requests",
		"till.get_auth_request",
	} {
		if !slices.Contains(toolNames, required) {
			t.Fatalf("tool list missing %q: %#v", required, toolNames)
		}
	}
}

// TestHandlerRaiseAttentionToolSchemaGuidance verifies markdown-rich summary/details guidance on raise_attention_item args.
func TestHandlerRaiseAttentionToolSchemaGuidance(t *testing.T) {
	capture := &stubCaptureStateReader{
		captureState: common.CaptureState{
			StateHash: "abc123",
		},
	}
	attention := &stubAttentionService{}
	handler, err := NewHandler(Config{}, capture, attention)
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
	schema := findToolSchemaByName(t, toolsRaw, "till.raise_attention_item")
	summaryDesc := schemaStringPropertyDescription(t, schema, "summary")
	if !strings.Contains(strings.ToLower(summaryDesc), "markdown-rich") {
		t.Fatalf("summary description = %q, want markdown-rich guidance", summaryDesc)
	}
	bodyDesc := schemaStringPropertyDescription(t, schema, "body_markdown")
	if !strings.Contains(strings.ToLower(bodyDesc), "markdown-rich") {
		t.Fatalf("body_markdown description = %q, want markdown-rich guidance", bodyDesc)
	}
}

// TestHandlerRegistersProjectToolsWhenAvailable verifies expanded project tools register when the capture adapter exposes project APIs.
func TestHandlerRegistersProjectToolsWhenAvailable(t *testing.T) {
	capture := &stubProjectService{
		stubCaptureStateReader: stubCaptureStateReader{
			captureState: common.CaptureState{StateHash: "abc123"},
		},
	}
	handler, err := NewHandler(Config{}, capture, nil)
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
	for _, required := range []string{
		"till.capture_state",
		"till.list_projects",
		"till.create_project",
		"till.update_project",
	} {
		if !slices.Contains(toolNames, required) {
			t.Fatalf("tool list missing %q: %#v", required, toolNames)
		}
	}
}

// TestHandlerProjectToolCall verifies expanded project tool wiring returns structured project rows.
func TestHandlerProjectToolCall(t *testing.T) {
	now := time.Date(2026, 2, 24, 12, 0, 0, 0, time.UTC)
	capture := &stubProjectService{
		stubCaptureStateReader: stubCaptureStateReader{
			captureState: common.CaptureState{StateHash: "abc123"},
		},
		projects: []domain.Project{
			{
				ID:        "p1",
				Slug:      "roadmap",
				Name:      "Roadmap",
				CreatedAt: now,
				UpdatedAt: now,
			},
		},
	}
	handler, err := NewHandler(Config{}, capture, nil)
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}

	server := httptest.NewServer(handler)
	defer server.Close()
	_, _ = postJSONRPC(t, server.Client(), server.URL, initializeRequest())

	_, callResp := postJSONRPC(t, server.Client(), server.URL, callToolRequest(3, "till.list_projects", map[string]any{
		"include_archived": true,
	}))
	structured := toolResultStructured(t, callResp.Result)
	projectsRaw, ok := structured["projects"].([]any)
	if !ok || len(projectsRaw) != 1 {
		t.Fatalf("projects = %#v, want one row", structured["projects"])
	}
	if !capture.lastIncludeArchived {
		t.Fatalf("include_archived = false, want true")
	}
}

// TestHandlerCaptureStateToolCall verifies tool-call wiring returns structured capture data.
func TestHandlerCaptureStateToolCall(t *testing.T) {
	now := time.Date(2026, 2, 24, 12, 0, 0, 0, time.UTC)
	capture := &stubCaptureStateReader{
		captureState: common.CaptureState{
			CapturedAt: now,
			StateHash:  "abc123",
			GoalOverview: common.GoalOverview{
				ProjectID:   "p1",
				ProjectName: "Roadmap",
			},
		},
	}
	handler, err := NewHandler(Config{}, capture, nil)
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}

	server := httptest.NewServer(handler)
	defer server.Close()
	_, _ = postJSONRPC(t, server.Client(), server.URL, initializeRequest())

	_, callResp := postJSONRPC(t, server.Client(), server.URL, map[string]any{
		"jsonrpc": "2.0",
		"id":      3,
		"method":  "tools/call",
		"params": map[string]any{
			"name": "till.capture_state",
			"arguments": map[string]any{
				"project_id": "p1",
				"view":       "full",
			},
		},
	})
	result, ok := callResp.Result["structuredContent"].(map[string]any)
	if !ok {
		t.Fatalf("structuredContent missing in response: %#v", callResp.Result)
	}
	if got, _ := result["state_hash"].(string); got != "abc123" {
		t.Fatalf("state_hash = %q, want abc123", got)
	}
	if capture.lastRequest.ProjectID != "p1" {
		t.Fatalf("project_id = %q, want p1", capture.lastRequest.ProjectID)
	}
	if capture.lastRequest.View != "full" {
		t.Fatalf("view = %q, want full", capture.lastRequest.View)
	}
}

// TestNewHandlerRequiresCaptureState verifies capture_state dependency enforcement.
func TestNewHandlerRequiresCaptureState(t *testing.T) {
	handler, err := NewHandler(Config{}, nil, nil)
	if err == nil {
		t.Fatalf("NewHandler() error = nil, want non-nil")
	}
	if handler != nil {
		t.Fatalf("handler = %#v, want nil", handler)
	}
}

// TestNormalizeConfig verifies deterministic config defaults and path normalization.
func TestNormalizeConfig(t *testing.T) {
	cases := []struct {
		name string
		in   Config
		want Config
	}{
		{
			name: "defaults",
			in:   Config{},
			want: Config{
				ServerName:    "tillsyn",
				ServerVersion: "dev",
				EndpointPath:  "/mcp",
			},
		},
		{
			name: "trimmed values and slash prefix",
			in: Config{
				ServerName:    " tillsyn-server ",
				ServerVersion: " v1.2.3 ",
				EndpointPath:  "custom/path",
			},
			want: Config{
				ServerName:    "tillsyn-server",
				ServerVersion: "v1.2.3",
				EndpointPath:  "/custom/path",
			},
		},
		{
			name: "endpoint trim of repeated slashes",
			in: Config{
				ServerName:    "tillsyn",
				ServerVersion: "dev",
				EndpointPath:  "///mcp///",
			},
			want: Config{
				ServerName:    "tillsyn",
				ServerVersion: "dev",
				EndpointPath:  "/mcp",
			},
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeConfig(tt.in)
			if got.ServerName != tt.want.ServerName {
				t.Fatalf("ServerName = %q, want %q", got.ServerName, tt.want.ServerName)
			}
			if got.ServerVersion != tt.want.ServerVersion {
				t.Fatalf("ServerVersion = %q, want %q", got.ServerVersion, tt.want.ServerVersion)
			}
			if got.EndpointPath != tt.want.EndpointPath {
				t.Fatalf("EndpointPath = %q, want %q", got.EndpointPath, tt.want.EndpointPath)
			}
		})
	}
}

// TestHandlerServeHTTPUnavailable verifies nil handler paths fail closed with 503.
func TestHandlerServeHTTPUnavailable(t *testing.T) {
	cases := []struct {
		name    string
		handler *Handler
	}{
		{
			name:    "nil receiver",
			handler: nil,
		},
		{
			name:    "missing inner http handler",
			handler: &Handler{},
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewBufferString(`{}`))
			rec := httptest.NewRecorder()

			tt.handler.ServeHTTP(rec, req)

			if rec.Code != http.StatusServiceUnavailable {
				t.Fatalf("status = %d, want %d", rec.Code, http.StatusServiceUnavailable)
			}
			if !strings.Contains(rec.Body.String(), "mcp handler unavailable") {
				t.Fatalf("body = %q, want mcp handler unavailable", rec.Body.String())
			}
		})
	}
}

// TestToolResultFromErrorMapping verifies deterministic error-to-tool-result mapping.
func TestToolResultFromErrorMapping(t *testing.T) {
	cases := []struct {
		name         string
		err          error
		wantPrefix   string
		wantLogCode  string
		wantLogClass string
	}{
		{
			name:         "nil error",
			err:          nil,
			wantPrefix:   "unknown error",
			wantLogCode:  "internal_error",
			wantLogClass: "internal",
		},
		{
			name:         "bootstrap required",
			err:          errors.Join(common.ErrBootstrapRequired, errors.New("no projects")),
			wantPrefix:   "bootstrap_required:",
			wantLogCode:  "bootstrap_required",
			wantLogClass: "bootstrap",
		},
		{
			name:         "guardrail violation",
			err:          errors.Join(common.ErrGuardrailViolation, errors.New("lease mismatch")),
			wantPrefix:   "guardrail_failed:",
			wantLogCode:  "guardrail_failed",
			wantLogClass: "guardrail",
		},
		{
			name:         "session required",
			err:          errors.Join(common.ErrSessionRequired, errors.New("missing session")),
			wantPrefix:   "session_required:",
			wantLogCode:  "session_required",
			wantLogClass: "auth",
		},
		{
			name:         "invalid auth",
			err:          errors.Join(common.ErrInvalidAuthentication, errors.New("bad secret")),
			wantPrefix:   "invalid_auth:",
			wantLogCode:  "invalid_auth",
			wantLogClass: "auth",
		},
		{
			name:         "session expired",
			err:          errors.Join(common.ErrSessionExpired, errors.New("expired")),
			wantPrefix:   "session_expired:",
			wantLogCode:  "session_expired",
			wantLogClass: "auth",
		},
		{
			name:         "auth denied",
			err:          errors.Join(common.ErrAuthorizationDenied, errors.New("policy deny")),
			wantPrefix:   "auth_denied:",
			wantLogCode:  "auth_denied",
			wantLogClass: "auth",
		},
		{
			name:         "grant required",
			err:          errors.Join(common.ErrGrantRequired, errors.New("approval needed")),
			wantPrefix:   "grant_required:",
			wantLogCode:  "grant_required",
			wantLogClass: "auth",
		},
		{
			name:         "invalid capture request",
			err:          errors.Join(common.ErrInvalidCaptureStateRequest, errors.New("bad request")),
			wantPrefix:   "invalid_request:",
			wantLogCode:  "invalid_request",
			wantLogClass: "invalid",
		},
		{
			name:         "unsupported scope",
			err:          errors.Join(common.ErrUnsupportedScope, errors.New("scope mismatch")),
			wantPrefix:   "invalid_request:",
			wantLogCode:  "invalid_request",
			wantLogClass: "invalid",
		},
		{
			name:         "not found",
			err:          errors.Join(common.ErrNotFound, errors.New("missing")),
			wantPrefix:   "not_found:",
			wantLogCode:  "not_found",
			wantLogClass: "not_found",
		},
		{
			name:         "attention unavailable",
			err:          errors.Join(common.ErrAttentionUnavailable, errors.New("disabled")),
			wantPrefix:   "not_implemented:",
			wantLogCode:  "not_implemented",
			wantLogClass: "not_implemented",
		},
		{
			name:         "internal",
			err:          errors.New("boom"),
			wantPrefix:   "internal_error:",
			wantLogCode:  "internal_error",
			wantLogClass: "internal",
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			logOutput, restoreLogger := captureDefaultLoggerOutput(t)
			defer restoreLogger()

			result := toolResultFromError(tt.err)
			if !result.IsError {
				t.Fatalf("IsError = false, want true")
			}
			if got := callToolResultText(t, result); !strings.HasPrefix(got, tt.wantPrefix) {
				t.Fatalf("text = %q, want prefix %q", got, tt.wantPrefix)
			}
			if got := logOutput.String(); !strings.Contains(got, "mcp tool error mapped") {
				t.Fatalf("log output = %q, want message marker", got)
			}
			if got := logOutput.String(); !strings.Contains(got, "transport=mcp") {
				t.Fatalf("log output = %q, want transport=mcp", got)
			}
			if got := logOutput.String(); !strings.Contains(got, "error_code="+tt.wantLogCode) {
				t.Fatalf("log output = %q, want error_code=%q", got, tt.wantLogCode)
			}
			if got := logOutput.String(); !strings.Contains(got, "error_class="+tt.wantLogClass) {
				t.Fatalf("log output = %q, want error_class=%q", got, tt.wantLogClass)
			}
		})
	}
}

// TestHandlerCaptureStateToolCallErrorPaths verifies required-arg and mapped-service errors.
func TestHandlerCaptureStateToolCallErrorPaths(t *testing.T) {
	capture := &stubCaptureStateReader{
		err: errors.Join(common.ErrUnsupportedScope, errors.New("scope mismatch")),
	}
	handler, err := NewHandler(Config{}, capture, nil)
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}

	server := httptest.NewServer(handler)
	defer server.Close()
	_, _ = postJSONRPC(t, server.Client(), server.URL, initializeRequest())

	_, missingArgResp := postJSONRPC(t, server.Client(), server.URL, callToolRequest(2, "till.capture_state", map[string]any{}))
	if isError, _ := missingArgResp.Result["isError"].(bool); !isError {
		t.Fatalf("isError = %v, want true", missingArgResp.Result["isError"])
	}
	if got := toolResultText(t, missingArgResp.Result); !strings.Contains(got, `required argument "project_id" not found`) {
		t.Fatalf("error text = %q, want required project_id message", got)
	}

	_, mappedErrResp := postJSONRPC(t, server.Client(), server.URL, callToolRequest(3, "till.capture_state", map[string]any{
		"project_id": "p1",
	}))
	if isError, _ := mappedErrResp.Result["isError"].(bool); !isError {
		t.Fatalf("isError = %v, want true", mappedErrResp.Result["isError"])
	}
	if got := toolResultText(t, mappedErrResp.Result); !strings.HasPrefix(got, "invalid_request:") {
		t.Fatalf("error text = %q, want prefix invalid_request:", got)
	}
}

// TestHandlerAttentionToolCalls verifies optional attention tools execute and map request arguments.
func TestHandlerAttentionToolCalls(t *testing.T) {
	now := time.Date(2026, 2, 24, 12, 0, 0, 0, time.UTC)
	capture := &stubCaptureStateReader{
		captureState: common.CaptureState{StateHash: "abc123"},
	}
	attention := &stubAttentionService{
		items: []common.AttentionItem{
			{
				ID:        "a1",
				ProjectID: "p1",
				ScopeType: common.ScopeTypeProject,
				ScopeID:   "p1",
				State:     common.AttentionStateOpen,
				Kind:      "risk_note",
				Summary:   "Need user action",
				CreatedAt: now,
			},
		},
		raised: common.AttentionItem{
			ID:                 "a2",
			ProjectID:          "p1",
			ScopeType:          common.ScopeTypeProject,
			ScopeID:            "p1",
			State:              common.AttentionStateOpen,
			Kind:               "blocker",
			Summary:            "Raised by tool",
			BodyMarkdown:       "Details",
			RequiresUserAction: true,
			CreatedAt:          now,
		},
		resolved: common.AttentionItem{
			ID:        "a1",
			ProjectID: "p1",
			ScopeType: common.ScopeTypeProject,
			ScopeID:   "p1",
			State:     common.AttentionStateResolved,
			Kind:      "risk_note",
			Summary:   "Need user action",
			CreatedAt: now,
		},
	}

	handler, err := NewHandler(Config{}, capture, attention)
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}

	server := httptest.NewServer(handler)
	defer server.Close()
	_, _ = postJSONRPC(t, server.Client(), server.URL, initializeRequest())

	_, listResp := postJSONRPC(t, server.Client(), server.URL, callToolRequest(2, "till.list_attention_items", map[string]any{
		"project_id": "p1",
		"scope_type": "project",
		"scope_id":   "p1",
		"state":      "open",
	}))
	listStructured := toolResultStructured(t, listResp.Result)
	itemsRaw, ok := listStructured["items"].([]any)
	if !ok || len(itemsRaw) != 1 {
		t.Fatalf("list structured items = %#v, want one item", listStructured["items"])
	}
	if attention.lastList.ProjectID != "p1" {
		t.Fatalf("list project_id = %q, want p1", attention.lastList.ProjectID)
	}
	if attention.lastList.ScopeType != "project" {
		t.Fatalf("list scope_type = %q, want project", attention.lastList.ScopeType)
	}
	if attention.lastList.ScopeID != "p1" {
		t.Fatalf("list scope_id = %q, want p1", attention.lastList.ScopeID)
	}
	if attention.lastList.State != "open" {
		t.Fatalf("list state = %q, want open", attention.lastList.State)
	}

	_, raiseResp := postJSONRPC(t, server.Client(), server.URL, callToolRequest(3, "till.raise_attention_item", mergeArgs(validSessionArgs(), map[string]any{
		"project_id":           "p1",
		"scope_type":           "project",
		"scope_id":             "p1",
		"kind":                 "blocker",
		"summary":              "Raised by tool",
		"body_markdown":        "Details",
		"requires_user_action": true,
		"agent_instance_id":    "inst-1",
		"lease_token":          "lease-1",
	})))
	raiseStructured := toolResultStructured(t, raiseResp.Result)
	if got, _ := raiseStructured["id"].(string); got != "a2" {
		t.Fatalf("raised id = %q, want a2", got)
	}
	if attention.lastRaise.ProjectID != "p1" {
		t.Fatalf("raise project_id = %q, want p1", attention.lastRaise.ProjectID)
	}
	if attention.lastRaise.ScopeType != "project" {
		t.Fatalf("raise scope_type = %q, want project", attention.lastRaise.ScopeType)
	}
	if attention.lastRaise.ScopeID != "p1" {
		t.Fatalf("raise scope_id = %q, want p1", attention.lastRaise.ScopeID)
	}
	if attention.lastRaise.Kind != "blocker" {
		t.Fatalf("raise kind = %q, want blocker", attention.lastRaise.Kind)
	}
	if attention.lastRaise.Summary != "Raised by tool" {
		t.Fatalf("raise summary = %q, want Raised by tool", attention.lastRaise.Summary)
	}
	if attention.lastRaise.BodyMarkdown != "Details" {
		t.Fatalf("raise body_markdown = %q, want Details", attention.lastRaise.BodyMarkdown)
	}
	if !attention.lastRaise.RequiresUserAction {
		t.Fatalf("raise requires_user_action = false, want true")
	}
	if got := attention.lastRaise.Actor.ActorID; got != "agent-1" {
		t.Fatalf("raise actor_id = %q, want agent-1", got)
	}

	_, resolveResp := postJSONRPC(t, server.Client(), server.URL, callToolRequest(4, "till.resolve_attention_item", mergeArgs(validSessionArgs(), map[string]any{
		"id":                "a1",
		"reason":            "approved",
		"agent_instance_id": "inst-1",
		"lease_token":       "lease-1",
	})))
	resolveStructured := toolResultStructured(t, resolveResp.Result)
	if got, _ := resolveStructured["state"].(string); got != common.AttentionStateResolved {
		t.Fatalf("resolved state = %q, want %q", got, common.AttentionStateResolved)
	}
	if attention.lastResolve.ID != "a1" {
		t.Fatalf("resolve id = %q, want a1", attention.lastResolve.ID)
	}
	if attention.lastResolve.Reason != "approved" {
		t.Fatalf("resolve reason = %q, want approved", attention.lastResolve.Reason)
	}
	if got := attention.lastResolve.Actor.ActorID; got != "agent-1" {
		t.Fatalf("resolve actor_id = %q, want agent-1", got)
	}
}

// TestHandlerAuthRequestToolCalls verifies auth-request create/list/show tools map request arguments and JSON results.
func TestHandlerAuthRequestToolCalls(t *testing.T) {
	now := time.Date(2026, 3, 20, 12, 0, 0, 0, time.UTC)
	capture := &stubAuthRequestService{
		stubCaptureStateReader: stubCaptureStateReader{
			captureState: common.CaptureState{StateHash: "abc123"},
		},
		created: common.AuthRequestRecord{
			ID:                  "req-1",
			State:               "pending",
			Path:                "project/p1",
			ProjectID:           "p1",
			ScopeType:           common.ScopeTypeProject,
			ScopeID:             "p1",
			PrincipalID:         "review-agent",
			PrincipalType:       "agent",
			PrincipalRole:       "orchestrator",
			ClientID:            "till-mcp-stdio",
			ClientType:          "mcp-stdio",
			RequestedSessionTTL: "2h0m0s",
			HasContinuation:     true,
			Reason:              "manual MCP review",
			RequestedByActor:    "orchestrator-1",
			RequestedByType:     "agent",
			CreatedAt:           now,
			ExpiresAt:           now.Add(30 * time.Minute),
		},
		requests: []common.AuthRequestRecord{
			{
				ID:                  "req-1",
				State:               "pending",
				Path:                "project/p1",
				ProjectID:           "p1",
				ScopeType:           common.ScopeTypeProject,
				ScopeID:             "p1",
				PrincipalID:         "review-agent",
				PrincipalType:       "agent",
				PrincipalRole:       "orchestrator",
				ClientID:            "till-mcp-stdio",
				ClientType:          "mcp-stdio",
				RequestedSessionTTL: "2h0m0s",
				HasContinuation:     true,
				Reason:              "manual MCP review",
				RequestedByActor:    "orchestrator-1",
				RequestedByType:     "agent",
				CreatedAt:           now,
				ExpiresAt:           now.Add(30 * time.Minute),
			},
		},
		getResult: common.AuthRequestRecord{
			ID:                  "req-1",
			State:               "pending",
			Path:                "project/p1",
			ProjectID:           "p1",
			ScopeType:           common.ScopeTypeProject,
			ScopeID:             "p1",
			PrincipalID:         "review-agent",
			PrincipalType:       "agent",
			PrincipalRole:       "orchestrator",
			ClientID:            "till-mcp-stdio",
			ClientType:          "mcp-stdio",
			RequestedSessionTTL: "2h0m0s",
			HasContinuation:     true,
			Reason:              "manual MCP review",
			RequestedByActor:    "orchestrator-1",
			RequestedByType:     "agent",
			CreatedAt:           now,
			ExpiresAt:           now.Add(30 * time.Minute),
		},
		claimResult: common.AuthRequestClaimResult{
			Request: common.AuthRequestRecord{
				ID:                     "req-1",
				State:                  "approved",
				Path:                   "project/p1",
				ApprovedPath:           "project/p1/branch/review",
				ProjectID:              "p1",
				ScopeType:              common.ScopeTypeProject,
				ScopeID:                "p1",
				PrincipalID:            "review-agent",
				PrincipalType:          "agent",
				PrincipalRole:          "subagent",
				ClientID:               "till-mcp-stdio",
				ClientType:             "mcp-stdio",
				RequestedSessionTTL:    "2h0m0s",
				HasContinuation:        true,
				ApprovedSessionTTL:     "2h0m0s",
				Reason:                 "manual MCP review",
				RequestedByActor:       "orchestrator-1",
				RequestedByType:        "agent",
				CreatedAt:              now,
				ExpiresAt:              now.Add(30 * time.Minute),
				IssuedSessionID:        "sess-1",
				IssuedSessionExpiresAt: ptrTime(now.Add(2 * time.Hour)),
			},
			SessionSecret: "secret-1",
		},
	}

	handler, err := NewHandler(Config{}, capture, nil)
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}

	server := httptest.NewServer(handler)
	defer server.Close()
	_, _ = postJSONRPC(t, server.Client(), server.URL, initializeRequest())

	_, createResp := postJSONRPC(t, server.Client(), server.URL, callToolRequest(2, "till.create_auth_request", map[string]any{
		"path":                "project/p1",
		"principal_id":        "review-agent",
		"principal_type":      "agent",
		"principal_role":      "orchestrator",
		"requested_by_actor":  "orchestrator-1",
		"requested_by_type":   "agent",
		"requester_client_id": "orchestrator-client",
		"client_id":           "till-mcp-stdio",
		"client_type":         "mcp-stdio",
		"requested_ttl":       "2h",
		"timeout":             "30m",
		"reason":              "manual MCP review",
		"continuation_json":   `{"resume_tool":"till.create_task"}`,
	}))
	createStructured := toolResultStructured(t, createResp.Result)
	if got := createStructured["id"].(string); got != "req-1" {
		t.Fatalf("create auth request id = %q, want req-1", got)
	}
	if got := createStructured["principal_role"].(string); got != "orchestrator" {
		t.Fatalf("create auth request principal_role = %q, want orchestrator", got)
	}
	if got := createStructured["requested_by_actor"].(string); got != "orchestrator-1" {
		t.Fatalf("create auth request requested_by_actor = %q, want orchestrator-1", got)
	}
	if got := createStructured["has_continuation"].(bool); !got {
		t.Fatal("create auth request has_continuation = false, want true")
	}
	if _, ok := createStructured["continuation"]; ok {
		t.Fatalf("create auth request leaked continuation = %#v, want omitted", createStructured["continuation"])
	}
	if got := capture.lastCreate.Path; got != "project/p1" {
		t.Fatalf("CreateAuthRequest() path = %q, want project/p1", got)
	}
	if got := capture.lastCreate.PrincipalRole; got != "orchestrator" {
		t.Fatalf("CreateAuthRequest() principal_role = %q, want orchestrator", got)
	}
	if got := capture.lastCreate.RequestedByActor; got != "orchestrator-1" {
		t.Fatalf("CreateAuthRequest() requested_by_actor = %q, want orchestrator-1", got)
	}
	if got := capture.lastCreate.RequesterClientID; got != "orchestrator-client" {
		t.Fatalf("CreateAuthRequest() requester_client_id = %q, want orchestrator-client", got)
	}
	if got := capture.lastCreate.ContinuationJSON; !strings.Contains(got, "resume_tool") {
		t.Fatalf("CreateAuthRequest() continuation_json = %q, want resume payload", got)
	}

	_, listResp := postJSONRPC(t, server.Client(), server.URL, callToolRequest(3, "till.list_auth_requests", map[string]any{
		"project_id": "p1",
		"state":      "pending",
		"limit":      10,
	}))
	listStructured := toolResultStructured(t, listResp.Result)
	requestsRaw, ok := listStructured["requests"].([]any)
	if !ok || len(requestsRaw) != 1 {
		t.Fatalf("list auth requests payload = %#v, want one request", listStructured)
	}
	if got := capture.lastList.ProjectID; got != "p1" {
		t.Fatalf("ListAuthRequests() project_id = %q, want p1", got)
	}
	if got := capture.lastList.State; got != "pending" {
		t.Fatalf("ListAuthRequests() state = %q, want pending", got)
	}

	_, getResp := postJSONRPC(t, server.Client(), server.URL, callToolRequest(4, "till.get_auth_request", map[string]any{
		"request_id": "req-1",
	}))
	getStructured := toolResultStructured(t, getResp.Result)
	if got := getStructured["id"].(string); got != "req-1" {
		t.Fatalf("get auth request id = %q, want req-1", got)
	}
	if got := capture.lastGetID; got != "req-1" {
		t.Fatalf("GetAuthRequest() request_id = %q, want req-1", got)
	}
	if got := getStructured["has_continuation"].(bool); !got {
		t.Fatal("get auth request has_continuation = false, want true")
	}
	if _, ok := getStructured["continuation"]; ok {
		t.Fatalf("get auth request leaked continuation = %#v, want omitted", getStructured["continuation"])
	}

	_, claimResp := postJSONRPC(t, server.Client(), server.URL, callToolRequest(5, "till.claim_auth_request", map[string]any{
		"request_id":   "req-1",
		"resume_token": "resume-1",
		"principal_id": "review-agent",
		"client_id":    "till-mcp-stdio",
		"wait_timeout": "30s",
	}))
	claimStructured := toolResultStructured(t, claimResp.Result)
	requestRecord, ok := claimStructured["request"].(map[string]any)
	if !ok {
		t.Fatalf("claim auth request payload = %#v, want nested request record", claimStructured)
	}
	if got := requestRecord["state"].(string); got != "approved" {
		t.Fatalf("claim auth request state = %q, want approved", got)
	}
	if got := requestRecord["approved_path"].(string); got != "project/p1/branch/review" {
		t.Fatalf("claim auth request approved_path = %q, want project/p1/branch/review", got)
	}
	if got := requestRecord["approved_session_ttl"].(string); got != "2h0m0s" {
		t.Fatalf("claim auth request approved_session_ttl = %q, want 2h0m0s", got)
	}
	if got := requestRecord["has_continuation"].(bool); !got {
		t.Fatal("claim auth request has_continuation = false, want true")
	}
	if _, ok := requestRecord["continuation"]; ok {
		t.Fatalf("claim auth request leaked continuation = %#v, want omitted", requestRecord["continuation"])
	}
	if got := claimStructured["session_secret"].(string); got != "secret-1" {
		t.Fatalf("claim auth request session_secret = %q, want secret-1", got)
	}
	if got := capture.lastClaim.RequestID; got != "req-1" {
		t.Fatalf("ClaimAuthRequest() request_id = %q, want req-1", got)
	}
	if got := capture.lastClaim.ResumeToken; got != "resume-1" {
		t.Fatalf("ClaimAuthRequest() resume_token = %q, want resume-1", got)
	}
	if got := capture.lastClaim.PrincipalID; got != "review-agent" {
		t.Fatalf("ClaimAuthRequest() principal_id = %q, want review-agent", got)
	}
	if got := capture.lastClaim.ClientID; got != "till-mcp-stdio" {
		t.Fatalf("ClaimAuthRequest() client_id = %q, want till-mcp-stdio", got)
	}
	if got := capture.lastClaim.WaitTimeout; got != "30s" {
		t.Fatalf("ClaimAuthRequest() wait_timeout = %q, want 30s", got)
	}
}

// TestHandlerClaimAuthRequestWaitingPayload verifies waiting claims return a pending request plus waiting=true without secrets.
func TestHandlerClaimAuthRequestWaitingPayload(t *testing.T) {
	capture := &stubAuthRequestService{
		stubCaptureStateReader: stubCaptureStateReader{
			captureState: common.CaptureState{StateHash: "abc123"},
		},
		claimResult: common.AuthRequestClaimResult{
			Request: common.AuthRequestRecord{
				ID:                  "req-2",
				State:               "pending",
				Path:                "project/p1",
				ProjectID:           "p1",
				ScopeType:           common.ScopeTypeProject,
				ScopeID:             "p1",
				PrincipalID:         "review-agent",
				PrincipalType:       "agent",
				PrincipalRole:       "subagent",
				ClientID:            "till-mcp-stdio",
				ClientType:          "mcp-stdio",
				RequestedSessionTTL: "2h0m0s",
			},
			Waiting: true,
		},
	}

	handler, err := NewHandler(Config{}, capture, nil)
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}

	server := httptest.NewServer(handler)
	defer server.Close()
	_, _ = postJSONRPC(t, server.Client(), server.URL, initializeRequest())

	_, claimResp := postJSONRPC(t, server.Client(), server.URL, callToolRequest(6, "till.claim_auth_request", map[string]any{
		"request_id":   "req-2",
		"resume_token": "resume-2",
		"principal_id": "review-agent",
		"client_id":    "till-mcp-stdio",
		"wait_timeout": "5s",
	}))
	claimStructured := toolResultStructured(t, claimResp.Result)
	if got, _ := claimStructured["waiting"].(bool); !got {
		t.Fatalf("claim waiting = %v, want true", claimStructured["waiting"])
	}
	if _, ok := claimStructured["session_secret"]; ok {
		t.Fatalf("claim session_secret = %#v, want omitted while waiting", claimStructured["session_secret"])
	}
	requestRecord, ok := claimStructured["request"].(map[string]any)
	if !ok {
		t.Fatalf("claim waiting payload = %#v, want nested request record", claimStructured)
	}
	if got := requestRecord["state"].(string); got != "pending" {
		t.Fatalf("claim waiting state = %q, want pending", got)
	}
}

// TestHandlerClaimAuthRequestErrorMapping verifies invalid continuation claims fail as invalid_request tool errors.
func TestHandlerClaimAuthRequestErrorMapping(t *testing.T) {
	capture := &stubAuthRequestService{
		stubCaptureStateReader: stubCaptureStateReader{
			captureState: common.CaptureState{StateHash: "abc123"},
		},
		claimErr: errors.Join(common.ErrInvalidCaptureStateRequest, errors.New("invalid auth request continuation")),
	}

	handler, err := NewHandler(Config{}, capture, nil)
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}

	server := httptest.NewServer(handler)
	defer server.Close()
	_, _ = postJSONRPC(t, server.Client(), server.URL, initializeRequest())

	_, claimResp := postJSONRPC(t, server.Client(), server.URL, callToolRequest(6, "till.claim_auth_request", map[string]any{
		"request_id":   "req-1",
		"resume_token": "wrong-token",
		"principal_id": "review-agent",
		"client_id":    "till-mcp-stdio",
	}))
	if isError, _ := claimResp.Result["isError"].(bool); !isError {
		t.Fatalf("claim_auth_request isError = %v, want true", claimResp.Result["isError"])
	}
	if got := toolResultText(t, claimResp.Result); !strings.HasPrefix(got, "invalid_request:") {
		t.Fatalf("claim_auth_request error text = %q, want prefix invalid_request:", got)
	}
}

// TestHandlerClaimAuthRequestRejectsNegativeWaitTimeout verifies invalid wait_timeout values fail as invalid_request tool errors.
func TestHandlerClaimAuthRequestRejectsNegativeWaitTimeout(t *testing.T) {
	capture := &stubAuthRequestService{
		stubCaptureStateReader: stubCaptureStateReader{
			captureState: common.CaptureState{StateHash: "abc123"},
		},
	}

	handler, err := NewHandler(Config{}, capture, nil)
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}

	server := httptest.NewServer(handler)
	defer server.Close()
	_, _ = postJSONRPC(t, server.Client(), server.URL, initializeRequest())

	_, claimResp := postJSONRPC(t, server.Client(), server.URL, callToolRequest(7, "till.claim_auth_request", map[string]any{
		"request_id":   "req-1",
		"resume_token": "resume-1",
		"principal_id": "review-agent",
		"client_id":    "till-mcp-stdio",
		"wait_timeout": "-1s",
	}))
	if len(claimResp.Error) > 0 {
		if got := fmt.Sprint(claimResp.Error["message"]); !strings.Contains(strings.ToLower(got), "invalid") {
			t.Fatalf("claim_auth_request negative wait rpc error = %#v, want invalid params style failure", claimResp.Error)
		}
		return
	}
	if got := toolResultText(t, claimResp.Result); !strings.HasPrefix(got, "invalid_request:") {
		t.Fatalf("claim_auth_request negative wait error text = %q, want prefix invalid_request:", got)
	}
}

// TestHandlerClaimAuthRequestRejectsRequesterMismatch verifies mismatched requester claims fail as invalid_request tool errors.
func TestHandlerClaimAuthRequestRejectsRequesterMismatch(t *testing.T) {
	capture := &stubAuthRequestService{
		stubCaptureStateReader: stubCaptureStateReader{
			captureState: common.CaptureState{StateHash: "abc123"},
		},
		claimErr: errors.Join(common.ErrInvalidCaptureStateRequest, domain.ErrAuthRequestClaimMismatch),
	}

	handler, err := NewHandler(Config{}, capture, nil)
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}

	server := httptest.NewServer(handler)
	defer server.Close()
	_, _ = postJSONRPC(t, server.Client(), server.URL, initializeRequest())

	_, claimResp := postJSONRPC(t, server.Client(), server.URL, callToolRequest(8, "till.claim_auth_request", map[string]any{
		"request_id":   "req-1",
		"resume_token": "resume-1",
		"principal_id": "other-agent",
		"client_id":    "other-client",
	}))
	if isError, _ := claimResp.Result["isError"].(bool); !isError {
		t.Fatalf("claim_auth_request isError = %v, want true", claimResp.Result["isError"])
	}
	if got := toolResultText(t, claimResp.Result); !strings.HasPrefix(got, "invalid_request:") {
		t.Fatalf("claim_auth_request mismatch error text = %q, want prefix invalid_request:", got)
	}
}

// TestHandlerClaimAuthRequestRequesterMismatchMapping verifies adoption attempts fail as invalid_request tool errors.
func TestHandlerClaimAuthRequestRequesterMismatchMapping(t *testing.T) {
	capture := &stubAuthRequestService{
		stubCaptureStateReader: stubCaptureStateReader{
			captureState: common.CaptureState{StateHash: "abc123"},
		},
		claimErr: errors.Join(common.ErrInvalidCaptureStateRequest, domain.ErrAuthRequestClaimMismatch),
	}

	handler, err := NewHandler(Config{}, capture, nil)
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}

	server := httptest.NewServer(handler)
	defer server.Close()
	_, _ = postJSONRPC(t, server.Client(), server.URL, initializeRequest())

	_, claimResp := postJSONRPC(t, server.Client(), server.URL, callToolRequest(7, "till.claim_auth_request", map[string]any{
		"request_id":   "req-1",
		"resume_token": "resume-1",
		"principal_id": "other-agent",
		"client_id":    "other-client",
	}))
	if isError, _ := claimResp.Result["isError"].(bool); !isError {
		t.Fatalf("claim_auth_request isError = %v, want true", claimResp.Result["isError"])
	}
	if got := toolResultText(t, claimResp.Result); !strings.HasPrefix(got, "invalid_request:") {
		t.Fatalf("claim_auth_request mismatch error text = %q, want prefix invalid_request:", got)
	}
}

// TestHandlerClaimAuthRequestWaitingResult verifies pending auth claims can return a waiting marker without leaking a secret.
func TestHandlerClaimAuthRequestWaitingResult(t *testing.T) {
	capture := &stubAuthRequestService{
		stubCaptureStateReader: stubCaptureStateReader{
			captureState: common.CaptureState{StateHash: "abc123"},
		},
		claimResult: common.AuthRequestClaimResult{
			Request: common.AuthRequestRecord{
				ID:                  "req-1",
				State:               "pending",
				Path:                "project/p1",
				ProjectID:           "p1",
				ScopeType:           common.ScopeTypeProject,
				ScopeID:             "p1",
				PrincipalID:         "review-agent",
				PrincipalType:       "agent",
				PrincipalRole:       "subagent",
				ClientID:            "till-mcp-stdio",
				ClientType:          "mcp-stdio",
				RequestedSessionTTL: "8h0m0s",
				CreatedAt:           time.Date(2026, 3, 20, 12, 0, 0, 0, time.UTC),
				ExpiresAt:           time.Date(2026, 3, 20, 12, 30, 0, 0, time.UTC),
			},
			Waiting: true,
		},
	}

	handler, err := NewHandler(Config{}, capture, nil)
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}

	server := httptest.NewServer(handler)
	defer server.Close()
	_, _ = postJSONRPC(t, server.Client(), server.URL, initializeRequest())

	_, claimResp := postJSONRPC(t, server.Client(), server.URL, callToolRequest(6, "till.claim_auth_request", map[string]any{
		"request_id":   "req-1",
		"resume_token": "resume-1",
		"principal_id": "review-agent",
		"client_id":    "till-mcp-stdio",
		"wait_timeout": "10ms",
	}))
	claimStructured := toolResultStructured(t, claimResp.Result)
	if waiting, ok := claimStructured["waiting"].(bool); !ok || !waiting {
		t.Fatalf("claim_auth_request waiting = %#v, want true", claimStructured["waiting"])
	}
	if _, ok := claimStructured["session_secret"]; ok {
		t.Fatalf("claim_auth_request session_secret present while waiting: %#v", claimStructured)
	}
}

// TestHandlerAttentionToolCallErrorMapping verifies attention tool errors surface as tool-result errors.
func TestHandlerAttentionToolCallErrorMapping(t *testing.T) {
	capture := &stubCaptureStateReader{
		captureState: common.CaptureState{StateHash: "abc123"},
	}
	attention := &stubAttentionService{
		listErr: errors.Join(common.ErrNotFound, errors.New("attention missing")),
	}

	handler, err := NewHandler(Config{}, capture, attention)
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}

	server := httptest.NewServer(handler)
	defer server.Close()
	_, _ = postJSONRPC(t, server.Client(), server.URL, initializeRequest())

	_, callResp := postJSONRPC(t, server.Client(), server.URL, callToolRequest(2, "till.list_attention_items", map[string]any{
		"project_id": "p1",
	}))
	if isError, _ := callResp.Result["isError"].(bool); !isError {
		t.Fatalf("isError = %v, want true", callResp.Result["isError"])
	}
	if got := toolResultText(t, callResp.Result); !strings.HasPrefix(got, "not_found:") {
		t.Fatalf("error text = %q, want prefix not_found:", got)
	}
}
