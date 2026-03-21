package common

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/hylla/tillsyn/internal/adapters/auth/autentauth"
	"github.com/hylla/tillsyn/internal/adapters/storage/sqlite"
	"github.com/hylla/tillsyn/internal/app"
	"github.com/hylla/tillsyn/internal/domain"
)

var authRequestTestNow = time.Date(2026, 3, 20, 12, 0, 0, 0, time.UTC)

// newAuthRequestAdapterForTest constructs one real adapter backed by sqlite and autent auth-request state.
func newAuthRequestAdapterForTest(t *testing.T) (*AppServiceAdapter, *sqlite.Repository) {
	t.Helper()

	repo, err := sqlite.Open(filepath.Join(t.TempDir(), "tillsyn.db"))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(func() {
		_ = repo.Close()
	})
	auth, err := autentauth.NewSharedDB(autentauth.Config{
		DB:    repo.DB(),
		Clock: func() time.Time { return authRequestTestNow },
	})
	if err != nil {
		t.Fatalf("NewSharedDB() error = %v", err)
	}
	if err := auth.EnsureDogfoodPolicy(context.Background()); err != nil {
		t.Fatalf("EnsureDogfoodPolicy() error = %v", err)
	}
	project, err := domain.NewProject("p1", "Project One", "", authRequestTestNow)
	if err != nil {
		t.Fatalf("NewProject() error = %v", err)
	}
	if err := repo.CreateProject(context.Background(), project); err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}
	nextID := 0
	idGen := func() string {
		nextID++
		return fmt.Sprintf("req-%d", nextID)
	}
	svc := app.NewService(repo, idGen, func() time.Time { return authRequestTestNow }, app.ServiceConfig{
		AuthRequests: auth,
		AuthBackend:  auth,
	})
	return NewAppServiceAdapter(svc, auth), repo
}

// TestAppServiceAdapterAuthRequestLifecycle verifies create/list/get auth-request transport mapping.
func TestAppServiceAdapterAuthRequestLifecycle(t *testing.T) {
	adapter, _ := newAuthRequestAdapterForTest(t)

	created, err := adapter.CreateAuthRequest(context.Background(), CreateAuthRequestRequest{
		Path:             "project/p1",
		PrincipalID:      "review-agent",
		PrincipalType:    "agent",
		PrincipalName:    "Review Agent",
		ClientID:         "till-mcp-stdio",
		ClientType:       "mcp-stdio",
		ClientName:       "Till MCP STDIO",
		RequestedTTL:     "2h",
		Timeout:          "30m",
		Reason:           "manual MCP review",
		ContinuationJSON: `{"resume_tool":"till.create_task","resume":{"path":"project/p1","attempt":1}}`,
	})
	if err != nil {
		t.Fatalf("CreateAuthRequest() error = %v", err)
	}
	if got := created.State; got != "pending" {
		t.Fatalf("CreateAuthRequest() state = %q, want pending", got)
	}
	if got := created.RequestedByActor; got != "review-agent" {
		t.Fatalf("CreateAuthRequest() requested_by_actor = %q, want review-agent", got)
	}
	if got, _ := created.Continuation["resume_tool"].(string); got != "till.create_task" {
		t.Fatalf("CreateAuthRequest() continuation resume_tool = %q, want till.create_task", got)
	}
	resume, ok := created.Continuation["resume"].(map[string]any)
	if !ok {
		t.Fatalf("CreateAuthRequest() continuation resume = %#v, want object", created.Continuation["resume"])
	}
	if got, _ := resume["path"].(string); got != "project/p1" {
		t.Fatalf("CreateAuthRequest() continuation resume.path = %q, want project/p1", got)
	}

	listed, err := adapter.ListAuthRequests(context.Background(), ListAuthRequestsRequest{
		ProjectID: "p1",
		State:     "pending",
		Limit:     10,
	})
	if err != nil {
		t.Fatalf("ListAuthRequests() error = %v", err)
	}
	if len(listed) != 1 || listed[0].ID != created.ID {
		t.Fatalf("ListAuthRequests() = %#v, want created request %q", listed, created.ID)
	}

	got, err := adapter.GetAuthRequest(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("GetAuthRequest() error = %v", err)
	}
	if got.ID != created.ID {
		t.Fatalf("GetAuthRequest() id = %q, want %q", got.ID, created.ID)
	}
	if got.Path != "project/p1" {
		t.Fatalf("GetAuthRequest() path = %q, want project/p1", got.Path)
	}

	approved, err := adapter.service.ApproveAuthRequest(context.Background(), app.ApproveAuthRequestInput{
		RequestID:      created.ID,
		ResolvedBy:     "operator-1",
		ResolvedType:   domain.ActorTypeUser,
		ResolutionNote: "approved for continuation",
	})
	if err != nil {
		t.Fatalf("ApproveAuthRequest() error = %v", err)
	}
	if approved.SessionSecret == "" {
		t.Fatal("ApproveAuthRequest() returned empty session secret")
	}

	claimed, err := adapter.ClaimAuthRequest(context.Background(), ClaimAuthRequestRequest{
		RequestID:   created.ID,
		ResumeToken: "",
	})
	if err == nil {
		t.Fatal("ClaimAuthRequest() error = nil, want invalid continuation")
	}

	adapterCreated, err := adapter.CreateAuthRequest(context.Background(), CreateAuthRequestRequest{
		Path:             "project/p1",
		PrincipalID:      "resume-agent",
		PrincipalType:    "agent",
		ClientID:         "till-mcp-stdio",
		ClientType:       "mcp-stdio",
		Reason:           "continuation claim",
		ContinuationJSON: `{"resume_token":"resume-123","resume_tool":"till.create_task"}`,
	})
	if err != nil {
		t.Fatalf("CreateAuthRequest(continuation) error = %v", err)
	}
	approved, err = adapter.service.ApproveAuthRequest(context.Background(), app.ApproveAuthRequestInput{
		RequestID:      adapterCreated.ID,
		ResolvedBy:     "operator-1",
		ResolvedType:   domain.ActorTypeUser,
		ResolutionNote: "approved for continuation",
	})
	if err != nil {
		t.Fatalf("ApproveAuthRequest(continuation) error = %v", err)
	}
	claimed, err = adapter.ClaimAuthRequest(context.Background(), ClaimAuthRequestRequest{
		RequestID:   adapterCreated.ID,
		ResumeToken: "resume-123",
	})
	if err != nil {
		t.Fatalf("ClaimAuthRequest() error = %v", err)
	}
	if got := claimed.Request.State; got != "approved" {
		t.Fatalf("ClaimAuthRequest() state = %q, want approved", got)
	}
	if got := claimed.SessionSecret; got != approved.SessionSecret {
		t.Fatalf("ClaimAuthRequest() session_secret = %q, want approved secret", got)
	}
}

// TestAppServiceAdapterCreateAuthRequestRejectsBadContinuationJSON verifies invalid continuation input fails closed.
func TestAppServiceAdapterCreateAuthRequestRejectsBadContinuationJSON(t *testing.T) {
	adapter, _ := newAuthRequestAdapterForTest(t)

	_, err := adapter.CreateAuthRequest(context.Background(), CreateAuthRequestRequest{
		Path:             "project/p1",
		PrincipalID:      "review-agent",
		ClientID:         "till-mcp-stdio",
		Reason:           "manual MCP review",
		ContinuationJSON: `{"resume_tool":`,
	})
	if err == nil || !strings.Contains(err.Error(), "continuation_json") {
		t.Fatalf("CreateAuthRequest() error = %v, want continuation_json validation", err)
	}
}
