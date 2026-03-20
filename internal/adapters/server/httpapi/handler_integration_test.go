package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	autent "github.com/evanmschultz/autent"
	"github.com/hylla/tillsyn/internal/adapters/auth/autentauth"
	servercommon "github.com/hylla/tillsyn/internal/adapters/server/common"
	"github.com/hylla/tillsyn/internal/adapters/storage/sqlite"
	"github.com/hylla/tillsyn/internal/app"
)

// newRealAttentionHandlerForTest constructs one real auth-backed HTTP attention handler plus its backing store.
func newRealAttentionHandlerForTest(t *testing.T) (*Handler, *sqlite.Repository, *autentauth.Service, string) {
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
	return NewHandler(adapter, adapter), repo, auth, project.ID
}

// issueUserSessionForTest issues one deterministic user session for real HTTP auth-backed integration tests.
func issueUserSessionForTest(t *testing.T, auth *autentauth.Service) autent.IssuedSession {
	t.Helper()

	issued, err := auth.IssueSession(context.Background(), autentauth.IssueSessionInput{
		PrincipalID:   "user-1",
		PrincipalType: "user",
		PrincipalName: "User One",
		ClientID:      "till-http-api",
		ClientType:    "http-api",
		ClientName:    "Till HTTP API",
	})
	if err != nil {
		t.Fatalf("IssueSession() error = %v", err)
	}
	return issued
}

// TestHandlerAttentionMutationPersistsAuthenticatedAttribution verifies a real authenticated HTTP mutation persists caller attribution.
func TestHandlerAttentionMutationPersistsAuthenticatedAttribution(t *testing.T) {
	handler, repo, auth, projectID := newRealAttentionHandlerForTest(t)
	issued := issueUserSessionForTest(t, auth)

	raiseReq := httptest.NewRequest(
		http.MethodPost,
		"/attention/items",
		strings.NewReader(fmt.Sprintf(`{"project_id":"%s","scope_type":"project","scope_id":"%s","kind":"risk_note","summary":"Raised by auth","session_id":"%s","session_secret":"%s"}`, projectID, projectID, issued.Session.ID, issued.Secret)),
	)
	raiseReq.Header.Set("Content-Type", "application/json")
	raiseRec := httptest.NewRecorder()
	handler.ServeHTTP(raiseRec, raiseReq)
	if raiseRec.Code != http.StatusCreated {
		t.Fatalf("raise status = %d, want %d", raiseRec.Code, http.StatusCreated)
	}

	var raised struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(raiseRec.Body).Decode(&raised); err != nil {
		t.Fatalf("Decode(raise) error = %v", err)
	}
	if raised.ID == "" {
		t.Fatal("raise attention id = empty, want persisted row id")
	}

	var createdByActor string
	var createdByType string
	if err := repo.DB().QueryRowContext(
		context.Background(),
		`SELECT created_by_actor, created_by_type FROM attention_items WHERE id = ?`,
		raised.ID,
	).Scan(&createdByActor, &createdByType); err != nil {
		t.Fatalf("QueryRow(created attribution) error = %v", err)
	}
	if createdByActor != "user-1" {
		t.Fatalf("created_by_actor = %q, want user-1", createdByActor)
	}
	if createdByType != "user" {
		t.Fatalf("created_by_type = %q, want user", createdByType)
	}

	resolveReq := httptest.NewRequest(
		http.MethodPost,
		"/attention/items/"+raised.ID+"/resolve",
		strings.NewReader(fmt.Sprintf(`{"reason":"approved","session_id":"%s","session_secret":"%s"}`, issued.Session.ID, issued.Secret)),
	)
	resolveReq.Header.Set("Content-Type", "application/json")
	resolveRec := httptest.NewRecorder()
	handler.ServeHTTP(resolveRec, resolveReq)
	if resolveRec.Code != http.StatusOK {
		t.Fatalf("resolve status = %d, want %d", resolveRec.Code, http.StatusOK)
	}

	var resolvedByActor string
	var resolvedByType string
	if err := repo.DB().QueryRowContext(
		context.Background(),
		`SELECT resolved_by_actor, resolved_by_type FROM attention_items WHERE id = ?`,
		raised.ID,
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
