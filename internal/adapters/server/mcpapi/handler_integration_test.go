package mcpapi

import (
	"context"
	"fmt"
	"net/http/httptest"
	"path/filepath"
	"testing"

	autent "github.com/evanmschultz/autent"
	"github.com/hylla/tillsyn/internal/adapters/auth/autentauth"
	servercommon "github.com/hylla/tillsyn/internal/adapters/server/common"
	"github.com/hylla/tillsyn/internal/adapters/storage/sqlite"
	"github.com/hylla/tillsyn/internal/app"
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

// TestHandlerAttentionMutationPersistsAuthenticatedAttribution verifies the real MCP mutation path persists authenticated attribution.
func TestHandlerAttentionMutationPersistsAuthenticatedAttribution(t *testing.T) {
	handler, repo, auth, projectID := newRealMCPAttentionHandlerForTest(t)
	issued := issueUserMCPTestSession(t, auth)

	server := httptest.NewServer(handler)
	defer server.Close()
	_, _ = postJSONRPC(t, server.Client(), server.URL, initializeRequest())

	_, raiseResp := postJSONRPC(t, server.Client(), server.URL, callToolRequest(2, "till.raise_attention_item", map[string]any{
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

	_, resolveResp := postJSONRPC(t, server.Client(), server.URL, callToolRequest(3, "till.resolve_attention_item", map[string]any{
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
