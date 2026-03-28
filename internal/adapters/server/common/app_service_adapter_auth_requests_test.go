package common

import (
	"context"
	"errors"
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
		PrincipalRole:    "orchestrator",
		PrincipalName:    "Review Agent",
		RequestedByActor: "orchestrator-1",
		RequestedByType:  "agent",
		ClientID:         "till-mcp-stdio",
		ClientType:       "mcp-stdio",
		ClientName:       "Till MCP STDIO",
		RequestedTTL:     "2h",
		Timeout:          "30m",
		Reason:           "manual MCP review",
		ContinuationJSON: `{"resume_token":"resume-123","resume_tool":"till.create_task","resume":{"path":"project/p1","attempt":1}}`,
	})
	if err != nil {
		t.Fatalf("CreateAuthRequest() error = %v", err)
	}
	if got := created.State; got != "pending" {
		t.Fatalf("CreateAuthRequest() state = %q, want pending", got)
	}
	if got := created.RequestedByActor; got != "orchestrator-1" {
		t.Fatalf("CreateAuthRequest() requested_by_actor = %q, want orchestrator-1", got)
	}
	if got := created.HasContinuation; !got {
		t.Fatal("CreateAuthRequest() has_continuation = false, want true")
	}
	if got := created.PrincipalRole; got != "orchestrator" {
		t.Fatalf("CreateAuthRequest() principal_role = %q, want orchestrator", got)
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
	if got.HasContinuation != true {
		t.Fatal("GetAuthRequest() has_continuation = false, want true")
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
		PrincipalID: "review-agent",
		ClientID:    "till-mcp-stdio",
	})
	if err == nil {
		t.Fatal("ClaimAuthRequest() error = nil, want invalid continuation")
	}

	adapterCreated, err := adapter.CreateAuthRequest(context.Background(), CreateAuthRequestRequest{
		Path:              "project/p1",
		PrincipalID:       "resume-agent",
		PrincipalType:     "agent",
		PrincipalRole:     "builder",
		RequestedByActor:  "orchestrator-1",
		RequestedByType:   "agent",
		RequesterClientID: "orchestrator-client",
		ClientID:          "builder-client",
		ClientType:        "mcp-stdio",
		RequestedTTL:      "4h",
		Reason:            "continuation claim",
		ContinuationJSON:  `{"resume_token":"resume-123","resume_tool":"till.create_task"}`,
	})
	if err != nil {
		t.Fatalf("CreateAuthRequest(continuation) error = %v", err)
	}
	approved, err = adapter.service.ApproveAuthRequest(context.Background(), app.ApproveAuthRequestInput{
		RequestID:      adapterCreated.ID,
		ResolvedBy:     "operator-1",
		ResolvedType:   domain.ActorTypeUser,
		Path:           "project/p1/branch/branch-1",
		SessionTTL:     2 * time.Hour,
		ResolutionNote: "approved for continuation",
	})
	if err != nil {
		t.Fatalf("ApproveAuthRequest(continuation) error = %v", err)
	}
	claimed, err = adapter.ClaimAuthRequest(context.Background(), ClaimAuthRequestRequest{
		RequestID:   adapterCreated.ID,
		ResumeToken: "resume-123",
		PrincipalID: "orchestrator-1",
		ClientID:    "orchestrator-client",
	})
	if err != nil {
		t.Fatalf("ClaimAuthRequest() error = %v", err)
	}
	if got := claimed.Request.State; got != "approved" {
		t.Fatalf("ClaimAuthRequest() state = %q, want approved", got)
	}
	if got := claimed.Request.Path; got != "project/p1" {
		t.Fatalf("ClaimAuthRequest() requested path = %q, want project/p1", got)
	}
	if got := claimed.Request.ApprovedPath; got != "project/p1/branch/branch-1" {
		t.Fatalf("ClaimAuthRequest() approved_path = %q, want project/p1/branch/branch-1", got)
	}
	if got := claimed.Request.RequestedSessionTTL; got != "4h0m0s" {
		t.Fatalf("ClaimAuthRequest() requested_session_ttl = %q, want 4h0m0s", got)
	}
	if got := claimed.Request.ApprovedSessionTTL; got != "2h0m0s" {
		t.Fatalf("ClaimAuthRequest() approved_session_ttl = %q, want 2h0m0s", got)
	}
	if !claimed.Request.HasContinuation {
		t.Fatal("ClaimAuthRequest() has_continuation = false, want true")
	}
	if got := claimed.Request.PrincipalRole; got != "builder" {
		t.Fatalf("ClaimAuthRequest() principal_role = %q, want builder", got)
	}
	if got := claimed.Request.RequestedByActor; got != "orchestrator-1" {
		t.Fatalf("ClaimAuthRequest() requested_by_actor = %q, want orchestrator-1", got)
	}
	if got := claimed.SessionSecret; got != approved.SessionSecret {
		t.Fatalf("ClaimAuthRequest() session_secret = %q, want approved secret", got)
	}
}

// TestAppServiceAdapterCancelAuthRequest verifies cancel maps through the real service lifecycle and resolves mirrored attention.
func TestAppServiceAdapterCancelAuthRequest(t *testing.T) {
	adapter, _ := newAuthRequestAdapterForTest(t)

	created, err := adapter.CreateAuthRequest(context.Background(), CreateAuthRequestRequest{
		Path:              "project/p1",
		PrincipalID:       "cancel-agent",
		PrincipalType:     "agent",
		PrincipalRole:     "orchestrator",
		RequestedByActor:  "orchestrator-1",
		RequestedByType:   "agent",
		RequesterClientID: "orchestrator-client",
		ClientID:          "builder-client",
		ClientType:        "mcp-stdio",
		Reason:            "cancel review",
		ContinuationJSON:  `{"resume_token":"cancel-123"}`,
	})
	if err != nil {
		t.Fatalf("CreateAuthRequest() error = %v", err)
	}

	canceled, err := adapter.CancelAuthRequest(context.Background(), CancelAuthRequestRequest{
		RequestID:      created.ID,
		ResumeToken:    "cancel-123",
		PrincipalID:    "orchestrator-1",
		ClientID:       "orchestrator-client",
		ResolutionNote: "superseded by newer request",
	})
	if err != nil {
		t.Fatalf("CancelAuthRequest() error = %v", err)
	}
	if got := canceled.State; got != "canceled" {
		t.Fatalf("CancelAuthRequest() state = %q, want canceled", got)
	}
	if got := canceled.ResolutionNote; got != "superseded by newer request" {
		t.Fatalf("CancelAuthRequest() resolution_note = %q, want superseded by newer request", got)
	}
	if got := canceled.ResolvedByActor; got != "orchestrator-1" {
		t.Fatalf("CancelAuthRequest() resolved_by_actor = %q, want orchestrator-1", got)
	}
	if got := canceled.ResolvedByType; got != "agent" {
		t.Fatalf("CancelAuthRequest() resolved_by_type = %q, want agent", got)
	}

	got, err := adapter.GetAuthRequest(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("GetAuthRequest() error = %v", err)
	}
	if got.State != "canceled" {
		t.Fatalf("GetAuthRequest() state = %q, want canceled", got.State)
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

// TestAppServiceAdapterCreateAuthRequestRejectsInvalidRole verifies unsupported auth-request roles fail as invalid transport input.
func TestAppServiceAdapterCreateAuthRequestRejectsInvalidRole(t *testing.T) {
	adapter, _ := newAuthRequestAdapterForTest(t)

	_, err := adapter.CreateAuthRequest(context.Background(), CreateAuthRequestRequest{
		Path:          "project/p1",
		PrincipalID:   "review-agent",
		PrincipalType: "agent",
		PrincipalRole: "global",
		ClientID:      "till-mcp-stdio",
		ClientType:    "mcp-stdio",
		Reason:        "manual MCP review",
	})
	if err == nil || !errors.Is(err, ErrInvalidCaptureStateRequest) {
		t.Fatalf("CreateAuthRequest() error = %v, want ErrInvalidCaptureStateRequest", err)
	}
}

// TestAppServiceAdapterClaimAuthRequestWaitingAndValidation verifies waiting claims and negative wait timeouts map cleanly through transport helpers.
func TestAppServiceAdapterClaimAuthRequestWaitingAndValidation(t *testing.T) {
	adapter, _ := newAuthRequestAdapterForTest(t)

	created, err := adapter.CreateAuthRequest(context.Background(), CreateAuthRequestRequest{
		Path:             "project/p1",
		PrincipalID:      "pending-agent",
		PrincipalType:    "agent",
		ClientID:         "till-mcp-stdio",
		ClientType:       "mcp-stdio",
		Reason:           "waiting review",
		ContinuationJSON: `{"resume_token":"resume-pending"}`,
	})
	if err != nil {
		t.Fatalf("CreateAuthRequest() error = %v", err)
	}

	claimed, err := adapter.ClaimAuthRequest(context.Background(), ClaimAuthRequestRequest{
		RequestID:   created.ID,
		ResumeToken: "resume-pending",
		PrincipalID: "pending-agent",
		ClientID:    "till-mcp-stdio",
		WaitTimeout: "1ms",
	})
	if err != nil {
		t.Fatalf("ClaimAuthRequest(waiting) error = %v", err)
	}
	if got := claimed.Request.State; got != "pending" {
		t.Fatalf("ClaimAuthRequest(waiting) state = %q, want pending", got)
	}
	if !claimed.Waiting {
		t.Fatal("ClaimAuthRequest(waiting) waiting = false, want true")
	}
	if claimed.SessionSecret != "" {
		t.Fatalf("ClaimAuthRequest(waiting) session_secret = %q, want empty", claimed.SessionSecret)
	}

	if _, err := adapter.ClaimAuthRequest(context.Background(), ClaimAuthRequestRequest{
		RequestID:   created.ID,
		ResumeToken: "resume-pending",
		PrincipalID: "pending-agent",
		ClientID:    "till-mcp-stdio",
		WaitTimeout: "-1s",
	}); err == nil || !errors.Is(err, ErrInvalidCaptureStateRequest) {
		t.Fatalf("ClaimAuthRequest(negative wait timeout) error = %v, want ErrInvalidCaptureStateRequest", err)
	}
}

// TestAppServiceAdapterCancelAuthRequestRejectsRequesterMismatch verifies cancel reuses continuation proof and fails closed for the wrong requester.
func TestAppServiceAdapterCancelAuthRequestRejectsRequesterMismatch(t *testing.T) {
	adapter, _ := newAuthRequestAdapterForTest(t)

	request, err := adapter.CreateAuthRequest(context.Background(), CreateAuthRequestRequest{
		Path:              "project/p1",
		PrincipalID:       "cancel-agent",
		PrincipalType:     "agent",
		RequestedByActor:  "orchestrator-1",
		RequestedByType:   "agent",
		RequesterClientID: "orchestrator-client",
		ClientID:          "builder-client",
		ClientType:        "mcp-stdio",
		Reason:            "cancel review",
		ContinuationJSON:  `{"resume_token":"cancel-123"}`,
	})
	if err != nil {
		t.Fatalf("CreateAuthRequest() error = %v", err)
	}

	_, err = adapter.CancelAuthRequest(context.Background(), CancelAuthRequestRequest{
		RequestID:   request.ID,
		ResumeToken: "cancel-123",
		PrincipalID: "other-agent",
		ClientID:    "other-client",
	})
	if err == nil || !errors.Is(err, ErrInvalidCaptureStateRequest) {
		t.Fatalf("CancelAuthRequest() error = %v, want ErrInvalidCaptureStateRequest", err)
	}
}

// TestAppServiceAdapterClaimAuthRequestRejectsRequesterMismatch verifies claim transport rejects attempts to adopt another caller's approved auth request.
func TestAppServiceAdapterClaimAuthRequestRejectsRequesterMismatch(t *testing.T) {
	adapter, _ := newAuthRequestAdapterForTest(t)

	request, err := adapter.CreateAuthRequest(context.Background(), CreateAuthRequestRequest{
		Path:             "project/p1",
		PrincipalID:      "review-agent",
		PrincipalType:    "agent",
		ClientID:         "till-mcp-stdio",
		ClientType:       "mcp-stdio",
		Reason:           "continuation claim",
		ContinuationJSON: `{"resume_token":"resume-123"}`,
	})
	if err != nil {
		t.Fatalf("CreateAuthRequest() error = %v", err)
	}
	if _, err := adapter.service.ApproveAuthRequest(context.Background(), app.ApproveAuthRequestInput{
		RequestID:      request.ID,
		ResolvedBy:     "operator-1",
		ResolvedType:   domain.ActorTypeUser,
		ResolutionNote: "approved",
	}); err != nil {
		t.Fatalf("ApproveAuthRequest() error = %v", err)
	}

	if _, err := adapter.ClaimAuthRequest(context.Background(), ClaimAuthRequestRequest{
		RequestID:   request.ID,
		ResumeToken: "resume-123",
		PrincipalID: "other-agent",
		ClientID:    "other-client",
	}); err == nil || !errors.Is(err, ErrInvalidCaptureStateRequest) {
		t.Fatalf("ClaimAuthRequest(requester mismatch) error = %v, want ErrInvalidCaptureStateRequest", err)
	}
}
