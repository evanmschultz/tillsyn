package mcpapi

import (
	"context"
	"fmt"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	autent "github.com/evanmschultz/autent"
	"github.com/evanmschultz/tillsyn/internal/adapters/auth/autentauth"
	servercommon "github.com/evanmschultz/tillsyn/internal/adapters/server/common"
	"github.com/evanmschultz/tillsyn/internal/adapters/storage/sqlite"
	"github.com/evanmschultz/tillsyn/internal/app"
	"github.com/evanmschultz/tillsyn/internal/domain"
)

// newRealMCPAttentionHandlerForTest constructs one real auth-backed MCP handler plus its backing store.
func newRealMCPAttentionHandlerForTest(t *testing.T) (*Handler, *sqlite.Repository, *autentauth.Service, string) {
	t.Helper()

	repo, err := sqlite.Open(filepath.Join(t.TempDir(), "tillsyn.db"))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(func() {
		_ = repo.Close()
	})

	auth, err := autentauth.NewSharedDB(autentauth.Config{DB: repo.DB()})
	if err != nil {
		t.Fatalf("NewSharedDB() error = %v", err)
	}
	if err := auth.EnsureDogfoodPolicy(context.Background()); err != nil {
		t.Fatalf("EnsureDogfoodPolicy() error = %v", err)
	}

	nextID := 0
	svc := app.NewService(repo, func() string {
		nextID++
		return fmt.Sprintf("id-%03d", nextID)
	}, nil, app.ServiceConfig{
		AutoCreateProjectColumns: true,
	})
	project, err := svc.CreateProject(context.Background(), "Demo", "")
	if err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}

	adapter := servercommon.NewAppServiceAdapter(svc, auth)
	handler, err := NewHandler(Config{}, adapter, adapter)
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}
	return handler, repo, auth, project.ID
}

// issueUserMCPTestSession issues one deterministic user session for real MCP auth-backed integration tests.
func issueUserMCPTestSession(t *testing.T, auth *autentauth.Service) autent.IssuedSession {
	t.Helper()

	issued, err := auth.IssueSession(context.Background(), autentauth.IssueSessionInput{
		PrincipalID:   "user-1",
		PrincipalType: "user",
		PrincipalName: "User One",
		ClientID:      "till-mcp-stdio",
		ClientType:    "mcp-stdio",
		ClientName:    "Till MCP STDIO",
	})
	if err != nil {
		t.Fatalf("IssueSession() error = %v", err)
	}
	return issued
}

// approvedPathHandoffFixture stores one real MCP handler plus approved-path handoff fixtures.
type approvedPathHandoffFixture struct {
	handler             *Handler
	auth                *autentauth.Service
	repo                *sqlite.Repository
	projectID           string
	branchID            string
	phaseID             string
	handoffID           string
	outOfScopeHandoffID string
}

// newApprovedPathHandoffFixture constructs one real MCP fixture for approved-path handoff updates.
func newApprovedPathHandoffFixture(t *testing.T) approvedPathHandoffFixture {
	t.Helper()

	repo, err := sqlite.Open(filepath.Join(t.TempDir(), "tillsyn.db"))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(func() {
		_ = repo.Close()
	})

	auth, err := autentauth.NewSharedDB(autentauth.Config{DB: repo.DB()})
	if err != nil {
		t.Fatalf("NewSharedDB() error = %v", err)
	}
	if err := auth.EnsureDogfoodPolicy(context.Background()); err != nil {
		t.Fatalf("EnsureDogfoodPolicy() error = %v", err)
	}

	nextID := 0
	service := app.NewService(repo, func() string {
		nextID++
		return fmt.Sprintf("id-%03d", nextID)
	}, nil, app.ServiceConfig{
		AutoCreateProjectColumns: true,
	})
	project, err := service.CreateProject(context.Background(), "Demo", "")
	if err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}
	columnID := firstMCPProjectColumnIDForTest(t, repo, project.ID)
	branch, phase, actionItem := createMCPScopedActionItemChainForTest(t, service, project.ID, columnID)
	otherBranch, _, otherActionItem := createMCPScopedActionItemChainForTest(t, service, project.ID, columnID)
	handoff, err := service.CreateHandoff(context.Background(), app.CreateHandoffInput{
		Level: domain.LevelTupleInput{
			ProjectID: project.ID,
			BranchID:  branch.ID,
			ScopeType: domain.ScopeLevelActionItem,
			ScopeID:   actionItem.ID,
		},
		SourceRole:  "builder",
		TargetRole:  "qa",
		Summary:     "handoff summary",
		NextAction:  "review",
		CreatedBy:   "user-1",
		CreatedType: domain.ActorTypeUser,
	})
	if err != nil {
		t.Fatalf("CreateHandoff() error = %v", err)
	}
	outOfScopeHandoff, err := service.CreateHandoff(context.Background(), app.CreateHandoffInput{
		Level: domain.LevelTupleInput{
			ProjectID: project.ID,
			BranchID:  otherBranch.ID,
			ScopeType: domain.ScopeLevelActionItem,
			ScopeID:   otherActionItem.ID,
		},
		SourceRole:  "builder",
		TargetRole:  "qa",
		Summary:     "out-of-scope handoff",
		NextAction:  "review",
		CreatedBy:   "user-1",
		CreatedType: domain.ActorTypeUser,
	})
	if err != nil {
		t.Fatalf("CreateHandoff(out of scope) error = %v", err)
	}

	adapter := servercommon.NewAppServiceAdapter(service, auth)
	handler, err := NewHandler(Config{}, adapter, adapter)
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}
	return approvedPathHandoffFixture{
		handler:             handler,
		auth:                auth,
		repo:                repo,
		projectID:           project.ID,
		branchID:            branch.ID,
		phaseID:             phase.ID,
		handoffID:           handoff.ID,
		outOfScopeHandoffID: outOfScopeHandoff.ID,
	}
}

// firstMCPProjectColumnIDForTest returns one auto-created project column for MCP integration fixtures.
func firstMCPProjectColumnIDForTest(t *testing.T, repo *sqlite.Repository, projectID string) string {
	t.Helper()

	columns, err := repo.ListColumns(context.Background(), projectID, true)
	if err != nil {
		t.Fatalf("ListColumns() error = %v", err)
	}
	if len(columns) == 0 {
		t.Fatal("ListColumns() returned no columns, want defaults")
	}
	return columns[0].ID
}

// createMCPScopedActionItemChainForTest creates one branch -> phase -> actionItem chain for MCP auth-context tests.
func createMCPScopedActionItemChainForTest(t *testing.T, service *app.Service, projectID, columnID string) (domain.ActionItem, domain.ActionItem, domain.ActionItem) {
	t.Helper()

	branch, err := service.CreateActionItem(context.Background(), app.CreateActionItemInput{
		ProjectID:      projectID,
		Kind:           domain.Kind("branch"),
		Scope:          domain.KindAppliesToBranch,
		ColumnID:       columnID,
		Title:          "Branch",
		CreatedByActor: "user-1",
		CreatedByName:  "User One",
		UpdatedByActor: "user-1",
		UpdatedByName:  "User One",
		UpdatedByType:  domain.ActorTypeUser,
	})
	if err != nil {
		t.Fatalf("CreateActionItem(branch) error = %v", err)
	}
	phase, err := service.CreateActionItem(context.Background(), app.CreateActionItemInput{
		ProjectID:      projectID,
		ParentID:       branch.ID,
		Kind:           domain.KindPhase,
		Scope:          domain.KindAppliesToPhase,
		ColumnID:       columnID,
		Title:          "Phase",
		CreatedByActor: "user-1",
		CreatedByName:  "User One",
		UpdatedByActor: "user-1",
		UpdatedByName:  "User One",
		UpdatedByType:  domain.ActorTypeUser,
	})
	if err != nil {
		t.Fatalf("CreateActionItem(phase) error = %v", err)
	}
	actionItem, err := service.CreateActionItem(context.Background(), app.CreateActionItemInput{
		ProjectID:      projectID,
		ParentID:       phase.ID,
		Kind:           domain.KindActionItem,
		Scope:          domain.KindAppliesToActionItem,
		ColumnID:       columnID,
		Title:          "ActionItem",
		CreatedByActor: "user-1",
		CreatedByName:  "User One",
		UpdatedByActor: "user-1",
		UpdatedByName:  "User One",
		UpdatedByType:  domain.ActorTypeUser,
	})
	if err != nil {
		t.Fatalf("CreateActionItem(actionItem) error = %v", err)
	}
	return branch, phase, actionItem
}

// issueApprovedPathMCPTestSession issues one MCP user session constrained to the requested branch/phase path.
func issueApprovedPathMCPTestSession(t *testing.T, auth *autentauth.Service, projectID, branchID string, phaseIDs ...string) autent.IssuedSession {
	t.Helper()

	issued, err := auth.IssueSession(context.Background(), autentauth.IssueSessionInput{
		PrincipalID:   "user-1",
		PrincipalType: "user",
		PrincipalName: "User One",
		ClientID:      "till-mcp-stdio",
		ClientType:    "mcp-stdio",
		ClientName:    "Till MCP STDIO",
		Metadata: map[string]string{
			"approved_path": domain.AuthRequestPath{
				ProjectID: projectID,
				BranchID:  branchID,
				PhaseIDs:  append([]string(nil), phaseIDs...),
			}.String(),
			"project_id": projectID,
		},
	})
	if err != nil {
		t.Fatalf("IssueSession() error = %v", err)
	}
	return issued
}

// TestHandlerAttentionMutationPersistsAuthenticatedAttribution verifies the real MCP mutation path persists authenticated attribution.
func TestHandlerAttentionMutationPersistsAuthenticatedAttribution(t *testing.T) {
	handler, repo, auth, projectID := newRealMCPAttentionHandlerForTest(t)
	issued := issueUserMCPTestSession(t, auth)

	server := httptest.NewServer(handler)
	defer server.Close()
	_, _ = postJSONRPC(t, server.Client(), server.URL, initializeRequest())

	_, raiseResp := postJSONRPC(t, server.Client(), server.URL, callToolRequest(2, "till.attention_item", map[string]any{
		"operation":            "raise",
		"project_id":           projectID,
		"scope_type":           "project",
		"scope_id":             projectID,
		"kind":                 "risk_note",
		"summary":              "Raised by MCP auth",
		"session_id":           issued.Session.ID,
		"session_secret":       issued.Secret,
		"requires_user_action": false,
	}))
	raised := toolResultStructured(t, raiseResp.Result)
	raiseID, _ := raised["id"].(string)
	if raiseID == "" {
		t.Fatalf("raise attention id = empty, want persisted row id from %#v", raised)
	}

	var createdByActor string
	var createdByType string
	if err := repo.DB().QueryRowContext(
		context.Background(),
		`SELECT created_by_actor, created_by_type FROM attention_items WHERE id = ?`,
		raiseID,
	).Scan(&createdByActor, &createdByType); err != nil {
		t.Fatalf("QueryRow(created attribution) error = %v", err)
	}
	if createdByActor != "user-1" {
		t.Fatalf("created_by_actor = %q, want user-1", createdByActor)
	}
	if createdByType != "user" {
		t.Fatalf("created_by_type = %q, want user", createdByType)
	}

	_, resolveResp := postJSONRPC(t, server.Client(), server.URL, callToolRequest(3, "till.attention_item", map[string]any{
		"operation":      "resolve",
		"id":             raiseID,
		"reason":         "approved",
		"session_id":     issued.Session.ID,
		"session_secret": issued.Secret,
	}))
	if resolveResp.Error != nil {
		t.Fatalf("resolve response error = %#v, want nil", resolveResp.Error)
	}

	var resolvedByActor string
	var resolvedByType string
	if err := repo.DB().QueryRowContext(
		context.Background(),
		`SELECT resolved_by_actor, resolved_by_type FROM attention_items WHERE id = ?`,
		raiseID,
	).Scan(&resolvedByActor, &resolvedByType); err != nil {
		t.Fatalf("QueryRow(resolved attribution) error = %v", err)
	}
	if resolvedByActor != "user-1" {
		t.Fatalf("resolved_by_actor = %q, want user-1", resolvedByActor)
	}
	if resolvedByType != "user" {
		t.Fatalf("resolved_by_type = %q, want user", resolvedByType)
	}
}

// TestHandlerUpdateHandoffResolvesApprovedPathContext verifies the real MCP transport now resolves by-id handoff scope before auth.
func TestHandlerUpdateHandoffResolvesApprovedPathContext(t *testing.T) {
	fixture := newApprovedPathHandoffFixture(t)
	issued := issueApprovedPathMCPTestSession(t, fixture.auth, fixture.projectID, fixture.branchID, fixture.phaseID)

	server := httptest.NewServer(fixture.handler)
	defer server.Close()
	_, _ = postJSONRPC(t, server.Client(), server.URL, initializeRequest())

	_, resp := postJSONRPC(t, server.Client(), server.URL, callToolRequest(22, "till.handoff", map[string]any{
		"operation":       "update",
		"handoff_id":      fixture.handoffID,
		"status":          "resolved",
		"summary":         "resolved handoff",
		"next_action":     "none",
		"resolution_note": "done",
		"session_id":      issued.Session.ID,
		"session_secret":  issued.Secret,
	}))
	if resp.Error != nil {
		t.Fatalf("update_handoff response error = %#v, want nil", resp.Error)
	}
	if isError, _ := resp.Result["isError"].(bool); isError {
		t.Fatalf("update_handoff returned isError=true: %q", toolResultText(t, resp.Result))
	}

	var (
		status         string
		resolutionNote string
	)
	if err := fixture.repo.DB().QueryRowContext(
		context.Background(),
		`SELECT status, resolution_note FROM handoffs WHERE id = ?`,
		fixture.handoffID,
	).Scan(&status, &resolutionNote); err != nil {
		t.Fatalf("QueryRow(updated handoff) error = %v", err)
	}
	if status != "resolved" {
		t.Fatalf("handoff status = %q, want resolved", status)
	}
	if resolutionNote != "done" {
		t.Fatalf("handoff resolution_note = %q, want done", resolutionNote)
	}
}

// TestHandlerUpdateHandoffOutOfScopeApprovedPathDenied verifies the MCP transport still fails closed for out-of-scope approved-path sessions.
func TestHandlerUpdateHandoffOutOfScopeApprovedPathDenied(t *testing.T) {
	fixture := newApprovedPathHandoffFixture(t)
	issued := issueApprovedPathMCPTestSession(t, fixture.auth, fixture.projectID, fixture.branchID, fixture.phaseID)

	server := httptest.NewServer(fixture.handler)
	defer server.Close()
	_, _ = postJSONRPC(t, server.Client(), server.URL, initializeRequest())

	_, resp := postJSONRPC(t, server.Client(), server.URL, callToolRequest(23, "till.handoff", map[string]any{
		"operation":       "update",
		"handoff_id":      fixture.outOfScopeHandoffID,
		"status":          "resolved",
		"summary":         "should fail",
		"next_action":     "none",
		"resolution_note": "denied",
		"session_id":      issued.Session.ID,
		"session_secret":  issued.Secret,
	}))
	if resp.Error != nil {
		t.Fatalf("update_handoff response error = %#v, want nil", resp.Error)
	}
	if isError, _ := resp.Result["isError"].(bool); !isError {
		t.Fatalf("update_handoff isError = %v, want true", resp.Result["isError"])
	}
	if got := toolResultText(t, resp.Result); !strings.HasPrefix(got, "auth_denied:") {
		t.Fatalf("update_handoff error text = %q, want auth_denied prefix", got)
	}
}
