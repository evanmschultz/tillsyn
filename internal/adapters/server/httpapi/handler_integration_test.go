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
	"github.com/hylla/tillsyn/internal/domain"
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

// approvedPathAttentionFixture stores one real HTTP handler plus approved-path attention fixtures.
type approvedPathAttentionFixture struct {
	handler               *Handler
	repo                  *sqlite.Repository
	auth                  *autentauth.Service
	projectID             string
	branchID              string
	phaseID               string
	attentionID           string
	outOfScopeAttentionID string
}

// newApprovedPathAttentionFixture constructs one real HTTP fixture for approved-path resolve tests.
func newApprovedPathAttentionFixture(t *testing.T) approvedPathAttentionFixture {
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
	columnID := firstHTTPProjectColumnIDForTest(t, repo, project.ID)
	branch, phase, task := createHTTPScopedTaskChainForTest(t, service, project.ID, columnID)
	otherBranch, _, otherTask := createHTTPScopedTaskChainForTest(t, service, project.ID, columnID)
	attention, err := service.RaiseAttentionItem(context.Background(), app.RaiseAttentionItemInput{
		Level: domain.LevelTupleInput{
			ProjectID: project.ID,
			BranchID:  branch.ID,
			ScopeType: domain.ScopeLevelTask,
			ScopeID:   task.ID,
		},
		Kind:        domain.AttentionKindRiskNote,
		Summary:     "Needs review",
		CreatedBy:   "user-1",
		CreatedType: domain.ActorTypeUser,
	})
	if err != nil {
		t.Fatalf("RaiseAttentionItem() error = %v", err)
	}
	outOfScopeAttention, err := service.RaiseAttentionItem(context.Background(), app.RaiseAttentionItemInput{
		Level: domain.LevelTupleInput{
			ProjectID: project.ID,
			BranchID:  otherBranch.ID,
			ScopeType: domain.ScopeLevelTask,
			ScopeID:   otherTask.ID,
		},
		Kind:        domain.AttentionKindRiskNote,
		Summary:     "Out of scope",
		CreatedBy:   "user-1",
		CreatedType: domain.ActorTypeUser,
	})
	if err != nil {
		t.Fatalf("RaiseAttentionItem(out of scope) error = %v", err)
	}

	adapter := servercommon.NewAppServiceAdapter(service, auth)
	return approvedPathAttentionFixture{
		handler:               NewHandler(adapter, adapter),
		repo:                  repo,
		auth:                  auth,
		projectID:             project.ID,
		branchID:              branch.ID,
		phaseID:               phase.ID,
		attentionID:           attention.ID,
		outOfScopeAttentionID: outOfScopeAttention.ID,
	}
}

// firstHTTPProjectColumnIDForTest returns one auto-created project column for HTTP integration fixtures.
func firstHTTPProjectColumnIDForTest(t *testing.T, repo *sqlite.Repository, projectID string) string {
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

// createHTTPScopedTaskChainForTest creates one branch -> phase -> task chain for HTTP approved-path tests.
func createHTTPScopedTaskChainForTest(t *testing.T, service *app.Service, projectID, columnID string) (domain.Task, domain.Task, domain.Task) {
	t.Helper()

	branch, err := service.CreateTask(context.Background(), app.CreateTaskInput{
		ProjectID:      projectID,
		Kind:           domain.WorkKind("branch"),
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
		t.Fatalf("CreateTask(branch) error = %v", err)
	}
	phase, err := service.CreateTask(context.Background(), app.CreateTaskInput{
		ProjectID:      projectID,
		ParentID:       branch.ID,
		Kind:           domain.WorkKindPhase,
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
		t.Fatalf("CreateTask(phase) error = %v", err)
	}
	task, err := service.CreateTask(context.Background(), app.CreateTaskInput{
		ProjectID:      projectID,
		ParentID:       phase.ID,
		Kind:           domain.WorkKindTask,
		Scope:          domain.KindAppliesToTask,
		ColumnID:       columnID,
		Title:          "Task",
		CreatedByActor: "user-1",
		CreatedByName:  "User One",
		UpdatedByActor: "user-1",
		UpdatedByName:  "User One",
		UpdatedByType:  domain.ActorTypeUser,
	})
	if err != nil {
		t.Fatalf("CreateTask(task) error = %v", err)
	}
	return branch, phase, task
}

// issueApprovedPathHTTPTestSession issues one HTTP user session constrained to the requested branch/phase path.
func issueApprovedPathHTTPTestSession(t *testing.T, auth *autentauth.Service, projectID, branchID string, phaseIDs ...string) autent.IssuedSession {
	t.Helper()

	issued, err := auth.IssueSession(context.Background(), autentauth.IssueSessionInput{
		PrincipalID:   "user-1",
		PrincipalType: "user",
		PrincipalName: "User One",
		ClientID:      "till-http-api",
		ClientType:    "http-api",
		ClientName:    "Till HTTP API",
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

// TestHandlerResolveAttentionItemApprovedPath verifies the HTTP mutation path resolves by-id scope before auth.
func TestHandlerResolveAttentionItemApprovedPath(t *testing.T) {
	fixture := newApprovedPathAttentionFixture(t)
	issued := issueApprovedPathHTTPTestSession(t, fixture.auth, fixture.projectID, fixture.branchID, fixture.phaseID)

	resolveReq := httptest.NewRequest(
		http.MethodPost,
		"/attention/items/"+fixture.attentionID+"/resolve",
		strings.NewReader(fmt.Sprintf(`{"reason":"approved","session_id":"%s","session_secret":"%s"}`, issued.Session.ID, issued.Secret)),
	)
	resolveReq.Header.Set("Content-Type", "application/json")
	resolveRec := httptest.NewRecorder()
	fixture.handler.ServeHTTP(resolveRec, resolveReq)
	if resolveRec.Code != http.StatusOK {
		t.Fatalf("resolve status = %d, want %d", resolveRec.Code, http.StatusOK)
	}

	var resolvedByActor string
	if err := fixture.repo.DB().QueryRowContext(
		context.Background(),
		`SELECT resolved_by_actor FROM attention_items WHERE id = ?`,
		fixture.attentionID,
	).Scan(&resolvedByActor); err != nil {
		t.Fatalf("QueryRow(resolved attribution) error = %v", err)
	}
	if resolvedByActor != "user-1" {
		t.Fatalf("resolved_by_actor = %q, want user-1", resolvedByActor)
	}
}

// TestHandlerResolveAttentionItemOutOfScopeApprovedPathDenied verifies the HTTP transport still fails closed for out-of-scope approved-path sessions.
func TestHandlerResolveAttentionItemOutOfScopeApprovedPathDenied(t *testing.T) {
	fixture := newApprovedPathAttentionFixture(t)
	issued := issueApprovedPathHTTPTestSession(t, fixture.auth, fixture.projectID, fixture.branchID, fixture.phaseID)

	resolveReq := httptest.NewRequest(
		http.MethodPost,
		"/attention/items/"+fixture.outOfScopeAttentionID+"/resolve",
		strings.NewReader(fmt.Sprintf(`{"reason":"approved","session_id":"%s","session_secret":"%s"}`, issued.Session.ID, issued.Secret)),
	)
	resolveReq.Header.Set("Content-Type", "application/json")
	resolveRec := httptest.NewRecorder()
	fixture.handler.ServeHTTP(resolveRec, resolveReq)
	if resolveRec.Code != http.StatusForbidden {
		t.Fatalf("resolve status = %d, want %d", resolveRec.Code, http.StatusForbidden)
	}

	var envelope ErrorEnvelope
	if err := json.NewDecoder(resolveRec.Body).Decode(&envelope); err != nil {
		t.Fatalf("Decode(error envelope) error = %v", err)
	}
	if envelope.Error.Code != "auth_denied" {
		t.Fatalf("error.code = %q, want auth_denied", envelope.Error.Code)
	}

	var resolvedByActor string
	if err := fixture.repo.DB().QueryRowContext(
		context.Background(),
		`SELECT COALESCE(resolved_by_actor, '') FROM attention_items WHERE id = ?`,
		fixture.outOfScopeAttentionID,
	).Scan(&resolvedByActor); err != nil {
		t.Fatalf("QueryRow(out-of-scope attention) error = %v", err)
	}
	if resolvedByActor != "" {
		t.Fatalf("resolved_by_actor = %q, want empty for denied resolve", resolvedByActor)
	}
}
