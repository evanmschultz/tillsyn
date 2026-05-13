package common

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/evanmschultz/tillsyn/internal/adapters/auth/autentauth"
	"github.com/evanmschultz/tillsyn/internal/adapters/storage/sqlite"
	"github.com/evanmschultz/tillsyn/internal/app"
	"github.com/evanmschultz/tillsyn/internal/domain"
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
	project, err := domain.NewProjectFromInput(domain.ProjectInput{ID: "p1", Name: "Project One"}, authRequestTestNow)
	if err != nil {
		t.Fatalf("NewProjectFromInput() error = %v", err)
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

// mustCreateProjectForAuthRequestTest creates one extra project row for auth-session scope tests.
func mustCreateProjectForAuthRequestTest(t *testing.T, repo *sqlite.Repository, id, name string) {
	t.Helper()

	project, err := domain.NewProjectFromInput(domain.ProjectInput{ID: id, Name: name}, authRequestTestNow)
	if err != nil {
		t.Fatalf("NewProjectFromInput(%q) error = %v", id, err)
	}
	if err := repo.CreateProject(context.Background(), project); err != nil {
		t.Fatalf("CreateProject(%q) error = %v", id, err)
	}
}

// mustCreateApprovedAuthSessionForTest creates and approves one auth request and returns the approved session bundle.
func mustCreateApprovedAuthSessionForTest(
	t *testing.T,
	adapter *AppServiceAdapter,
	req CreateAuthRequestRequest,
	approvePath string,
) app.ApprovedAuthRequestResult {
	t.Helper()

	created, err := adapter.CreateAuthRequest(context.Background(), req)
	if err != nil {
		t.Fatalf("CreateAuthRequest(%q) error = %v", req.Path, err)
	}
	approve := app.ApproveAuthRequestInput{
		RequestID:      created.ID,
		ResolvedBy:     "operator-1",
		ResolvedType:   domain.ActorTypeUser,
		ResolutionNote: "approved for auth session governance",
	}
	if strings.TrimSpace(approvePath) != "" {
		approve.Path = approvePath
	}
	approved, err := adapter.service.ApproveAuthRequest(context.Background(), approve)
	if err != nil {
		t.Fatalf("ApproveAuthRequest(%q) error = %v", req.Path, err)
	}
	return approved
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
		ContinuationJSON: `{"resume_token":"resume-123","resume_tool":"till.action_item","resume":{"path":"project/p1","attempt":1}}`,
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
	if got := got.Continuation[app.AuthRequestContinuationRequesterClientIDKey]; got != "till-mcp-stdio" {
		t.Fatalf("GetAuthRequest() continuation requester client = %#v, want till-mcp-stdio", got)
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
		ContinuationJSON:  `{"resume_token":"resume-123","resume_tool":"till.action_item"}`,
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
		PrincipalID: "resume-agent",
		ClientID:    "builder-client",
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
	if got := claimed.Request.PrincipalID; got != "resume-agent" {
		t.Fatalf("ClaimAuthRequest() principal_id = %q, want resume-agent", got)
	}
	if got := claimed.Request.Continuation[app.AuthRequestContinuationRequesterClientIDKey]; got != "orchestrator-client" {
		t.Fatalf("ClaimAuthRequest() continuation requester client = %#v, want orchestrator-client", got)
	}
	if got := claimed.SessionSecret; got != approved.SessionSecret {
		t.Fatalf("ClaimAuthRequest() session_secret = %q, want approved secret", got)
	}
}

// TestAppServiceAdapterCreateAuthRequestDelegatedResearchUsesActingSession verifies delegated
// research requests derive requester ownership from the acting orchestrator session.
func TestAppServiceAdapterCreateAuthRequestDelegatedResearchUsesActingSession(t *testing.T) {
	adapter, _ := newAuthRequestAdapterForTest(t)

	acting := mustCreateApprovedAuthSessionForTest(t, adapter, CreateAuthRequestRequest{
		Path:             "project/p1",
		PrincipalID:      "orchestrator-1",
		PrincipalType:    "agent",
		PrincipalRole:    "orchestrator",
		ClientID:         "orch-client",
		ClientType:       "mcp-stdio",
		Reason:           "orchestrator project scope",
		ContinuationJSON: `{"resume_token":"resume-orch"}`,
	}, "")

	created, err := adapter.CreateAuthRequest(context.Background(), CreateAuthRequestRequest{
		Path:                "project/p1/branch/research-1",
		PrincipalID:         "research-1",
		PrincipalType:       "agent",
		PrincipalRole:       "research",
		ClientID:            "research-client",
		ClientType:          "mcp-stdio",
		Reason:              "delegated research scope",
		ActingSessionID:     acting.Request.IssuedSessionID,
		ActingSessionSecret: acting.SessionSecret,
	})
	if err != nil {
		t.Fatalf("CreateAuthRequest(delegated research) error = %v", err)
	}
	if got := created.RequestedByActor; got != "orchestrator-1" {
		t.Fatalf("CreateAuthRequest(delegated research) requested_by_actor = %q, want orchestrator-1", got)
	}
	if got := created.RequestedByType; got != "agent" {
		t.Fatalf("CreateAuthRequest(delegated research) requested_by_type = %q, want agent", got)
	}
	if got := created.PrincipalRole; got != "research" {
		t.Fatalf("CreateAuthRequest(delegated research) principal_role = %q, want research", got)
	}
	if got := created.Continuation[app.AuthRequestContinuationRequesterClientIDKey]; got != "orch-client" {
		t.Fatalf("CreateAuthRequest(delegated research) continuation requester client = %#v, want orch-client", got)
	}
	if got := created.Path; got != "project/p1/branch/research-1" {
		t.Fatalf("CreateAuthRequest(delegated research) path = %q, want project/p1/branch/research-1", got)
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

// TestAppServiceAdapterAuthSessionLifecycle verifies session list/validate/revoke transport mapping.
func TestAppServiceAdapterAuthSessionLifecycle(t *testing.T) {
	adapter, _ := newAuthRequestAdapterForTest(t)

	created, err := adapter.CreateAuthRequest(context.Background(), CreateAuthRequestRequest{
		Path:              "project/p1",
		PrincipalID:       "builder-1",
		PrincipalType:     "agent",
		PrincipalRole:     "builder",
		RequestedByActor:  "orchestrator-1",
		RequestedByType:   "agent",
		RequesterClientID: "orchestrator-client",
		ClientID:          "builder-client",
		ClientType:        "mcp-stdio",
		ClientName:        "Builder MCP",
		RequestedTTL:      "720h",
		Reason:            "session lifecycle proof",
		ContinuationJSON:  `{"resume_token":"resume-session-1"}`,
	})
	if err != nil {
		t.Fatalf("CreateAuthRequest() error = %v", err)
	}
	approved, err := adapter.service.ApproveAuthRequest(context.Background(), app.ApproveAuthRequestInput{
		RequestID:      created.ID,
		ResolvedBy:     "operator-1",
		ResolvedType:   domain.ActorTypeUser,
		ResolutionNote: "approved for session lifecycle proof",
	})
	if err != nil {
		t.Fatalf("ApproveAuthRequest() error = %v", err)
	}
	sessionID := approved.Request.IssuedSessionID

	sessions, err := adapter.ListAuthSessions(context.Background(), ListAuthSessionsRequest{
		ProjectID:           "p1",
		SessionID:           sessionID,
		State:               "active",
		Limit:               10,
		ActingSessionID:     sessionID,
		ActingSessionSecret: approved.SessionSecret,
	})
	if err != nil {
		t.Fatalf("ListAuthSessions() error = %v", err)
	}
	if len(sessions) != 1 {
		t.Fatalf("ListAuthSessions() len = %d, want 1", len(sessions))
	}
	if got := sessions[0].SessionID; got != sessionID {
		t.Fatalf("ListAuthSessions() session_id = %q, want %q", got, sessionID)
	}
	if got := sessions[0].State; got != "active" {
		t.Fatalf("ListAuthSessions() state = %q, want active", got)
	}
	if got := sessions[0].AuthRequestID; got != created.ID {
		t.Fatalf("ListAuthSessions() auth_request_id = %q, want %q", got, created.ID)
	}

	validated, err := adapter.ValidateAuthSession(context.Background(), ValidateAuthSessionRequest{
		SessionID:     sessionID,
		SessionSecret: approved.SessionSecret,
	})
	if err != nil {
		t.Fatalf("ValidateAuthSession() error = %v", err)
	}
	if got := validated.State; got != "active" {
		t.Fatalf("ValidateAuthSession() state = %q, want active", got)
	}
	if validated.LastValidatedAt == nil {
		t.Fatal("ValidateAuthSession() last_validated_at = nil, want timestamp")
	}

	revoked, err := adapter.RevokeAuthSession(context.Background(), RevokeAuthSessionRequest{
		SessionID:           sessionID,
		Reason:              "operator cleanup",
		ActingSessionID:     sessionID,
		ActingSessionSecret: approved.SessionSecret,
	})
	if err != nil {
		t.Fatalf("RevokeAuthSession() error = %v", err)
	}
	if got := revoked.State; got != "revoked" {
		t.Fatalf("RevokeAuthSession() state = %q, want revoked", got)
	}
	if got := revoked.RevocationReason; got != "operator cleanup" {
		t.Fatalf("RevokeAuthSession() revocation_reason = %q, want operator cleanup", got)
	}
	if revoked.RevokedAt == nil {
		t.Fatal("RevokeAuthSession() revoked_at = nil, want timestamp")
	}
}

// TestAppServiceAdapterListAuthSessionsFiltersByActingApprovedPath verifies project-scoped governance
// does not widen to broader global or multi-project sessions just because they overlap one project filter.
func TestAppServiceAdapterListAuthSessionsFiltersByActingApprovedPath(t *testing.T) {
	adapter, repo := newAuthRequestAdapterForTest(t)
	mustCreateProjectForAuthRequestTest(t, repo, "p2", "Project Two")

	acting := mustCreateApprovedAuthSessionForTest(t, adapter, CreateAuthRequestRequest{
		Path:             "project/p1",
		PrincipalID:      "orch-p1",
		PrincipalType:    "agent",
		PrincipalRole:    "orchestrator",
		ClientID:         "orch-client-p1",
		ClientType:       "mcp-stdio",
		Reason:           "project governance",
		ContinuationJSON: `{"resume_token":"resume-acting-p1"}`,
	}, "")
	projectChild := mustCreateApprovedAuthSessionForTest(t, adapter, CreateAuthRequestRequest{
		Path:             "project/p1/branch/b1",
		PrincipalID:      "builder-p1",
		PrincipalType:    "agent",
		PrincipalRole:    "builder",
		ClientID:         "builder-client-p1",
		ClientType:       "mcp-stdio",
		Reason:           "project child session",
		ContinuationJSON: `{"resume_token":"resume-child-p1"}`,
	}, "")
	multi := mustCreateApprovedAuthSessionForTest(t, adapter, CreateAuthRequestRequest{
		Path:             "projects/p1,p2",
		PrincipalID:      "orch-multi",
		PrincipalType:    "agent",
		PrincipalRole:    "orchestrator",
		ClientID:         "orch-client-multi",
		ClientType:       "mcp-stdio",
		Reason:           "multi project session",
		ContinuationJSON: `{"resume_token":"resume-multi"}`,
	}, "")
	global := mustCreateApprovedAuthSessionForTest(t, adapter, CreateAuthRequestRequest{
		Path:             "global",
		PrincipalID:      "orch-global",
		PrincipalType:    "agent",
		PrincipalRole:    "orchestrator",
		ClientID:         "orch-client-global",
		ClientType:       "mcp-stdio",
		Reason:           "global session",
		ContinuationJSON: `{"resume_token":"resume-global"}`,
	}, "")

	sessions, err := adapter.ListAuthSessions(context.Background(), ListAuthSessionsRequest{
		ProjectID:           "p1",
		State:               "active",
		ActingSessionID:     acting.Request.IssuedSessionID,
		ActingSessionSecret: acting.SessionSecret,
		Limit:               10,
	})
	if err != nil {
		t.Fatalf("ListAuthSessions(project scoped) error = %v", err)
	}
	if len(sessions) != 2 {
		t.Fatalf("ListAuthSessions(project scoped) len = %d, want 2", len(sessions))
	}
	gotIDs := []string{sessions[0].SessionID, sessions[1].SessionID}
	if !slices.Contains(gotIDs, acting.Request.IssuedSessionID) {
		t.Fatalf("ListAuthSessions(project scoped) missing acting session: %#v", gotIDs)
	}
	if !slices.Contains(gotIDs, projectChild.Request.IssuedSessionID) {
		t.Fatalf("ListAuthSessions(project scoped) missing child project session: %#v", gotIDs)
	}
	if slices.Contains(gotIDs, multi.Request.IssuedSessionID) {
		t.Fatalf("ListAuthSessions(project scoped) unexpectedly included multi-project session: %#v", gotIDs)
	}
	if slices.Contains(gotIDs, global.Request.IssuedSessionID) {
		t.Fatalf("ListAuthSessions(project scoped) unexpectedly included global session: %#v", gotIDs)
	}
}

// TestAppServiceAdapterMultiProjectAuthSessionGovernance verifies multi-project acting approvals
// can govern matching child sessions without gaining global session power.
func TestAppServiceAdapterMultiProjectAuthSessionGovernance(t *testing.T) {
	adapter, repo := newAuthRequestAdapterForTest(t)
	mustCreateProjectForAuthRequestTest(t, repo, "p2", "Project Two")

	acting := mustCreateApprovedAuthSessionForTest(t, adapter, CreateAuthRequestRequest{
		Path:             "projects/p1,p2",
		PrincipalID:      "orch-multi-acting",
		PrincipalType:    "agent",
		PrincipalRole:    "orchestrator",
		ClientID:         "orch-client-multi-acting",
		ClientType:       "mcp-stdio",
		Reason:           "multi project governance",
		ContinuationJSON: `{"resume_token":"resume-multi-acting"}`,
	}, "")
	projectChild := mustCreateApprovedAuthSessionForTest(t, adapter, CreateAuthRequestRequest{
		Path:             "project/p2",
		PrincipalID:      "builder-p2",
		PrincipalType:    "agent",
		PrincipalRole:    "builder",
		ClientID:         "builder-client-p2",
		ClientType:       "mcp-stdio",
		Reason:           "project p2 session",
		ContinuationJSON: `{"resume_token":"resume-project-p2"}`,
	}, "")
	multiChild := mustCreateApprovedAuthSessionForTest(t, adapter, CreateAuthRequestRequest{
		Path:             "projects/p1,p2",
		PrincipalID:      "orch-multi-child",
		PrincipalType:    "agent",
		PrincipalRole:    "orchestrator",
		ClientID:         "orch-client-multi-child",
		ClientType:       "mcp-stdio",
		Reason:           "matching multi project session",
		ContinuationJSON: `{"resume_token":"resume-orch-multi-child"}`,
	}, "")
	global := mustCreateApprovedAuthSessionForTest(t, adapter, CreateAuthRequestRequest{
		Path:             "global",
		PrincipalID:      "orch-global",
		PrincipalType:    "agent",
		PrincipalRole:    "orchestrator",
		ClientID:         "orch-client-global",
		ClientType:       "mcp-stdio",
		Reason:           "global session",
		ContinuationJSON: `{"resume_token":"resume-global"}`,
	}, "")

	sessions, err := adapter.ListAuthSessions(context.Background(), ListAuthSessionsRequest{
		State:               "active",
		ActingSessionID:     acting.Request.IssuedSessionID,
		ActingSessionSecret: acting.SessionSecret,
		Limit:               10,
	})
	if err != nil {
		t.Fatalf("ListAuthSessions(multi project) error = %v", err)
	}
	var gotIDs []string
	for _, session := range sessions {
		gotIDs = append(gotIDs, session.SessionID)
	}
	for _, want := range []string{
		acting.Request.IssuedSessionID,
		projectChild.Request.IssuedSessionID,
		multiChild.Request.IssuedSessionID,
	} {
		if !slices.Contains(gotIDs, want) {
			t.Fatalf("ListAuthSessions(multi project) missing governed session %q in %#v", want, gotIDs)
		}
	}
	if slices.Contains(gotIDs, global.Request.IssuedSessionID) {
		t.Fatalf("ListAuthSessions(multi project) unexpectedly included global session: %#v", gotIDs)
	}

	revoked, err := adapter.RevokeAuthSession(context.Background(), RevokeAuthSessionRequest{
		SessionID:           multiChild.Request.IssuedSessionID,
		Reason:              "multi project cleanup",
		ActingSessionID:     acting.Request.IssuedSessionID,
		ActingSessionSecret: acting.SessionSecret,
	})
	if err != nil {
		t.Fatalf("RevokeAuthSession(multi project) error = %v", err)
	}
	if got := revoked.State; got != "revoked" {
		t.Fatalf("RevokeAuthSession(multi project) state = %q, want revoked", got)
	}

	if _, err := adapter.RevokeAuthSession(context.Background(), RevokeAuthSessionRequest{
		SessionID:           global.Request.IssuedSessionID,
		Reason:              "should fail",
		ActingSessionID:     acting.Request.IssuedSessionID,
		ActingSessionSecret: acting.SessionSecret,
	}); !errors.Is(err, ErrAuthorizationDenied) {
		t.Fatalf("RevokeAuthSession(global via multi project) error = %v, want ErrAuthorizationDenied", err)
	}
}

// TestAppServiceAdapterCheckAuthSessionGovernanceReportsOutOfScope verifies non-destructive checks
// return a structured denial for broader sessions that fall outside the acting session scope.
func TestAppServiceAdapterCheckAuthSessionGovernanceReportsOutOfScope(t *testing.T) {
	adapter, repo := newAuthRequestAdapterForTest(t)
	mustCreateProjectForAuthRequestTest(t, repo, "p2", "Project Two")

	acting := mustCreateApprovedAuthSessionForTest(t, adapter, CreateAuthRequestRequest{
		Path:             "projects/p1,p2",
		PrincipalID:      "orch-multi-acting",
		PrincipalType:    "agent",
		PrincipalRole:    "orchestrator",
		ClientID:         "orch-client-multi-acting",
		ClientType:       "mcp-stdio",
		Reason:           "multi project governance",
		ContinuationJSON: `{"resume_token":"resume-multi-acting"}`,
	}, "")
	global := mustCreateApprovedAuthSessionForTest(t, adapter, CreateAuthRequestRequest{
		Path:             "global",
		PrincipalID:      "orch-global",
		PrincipalType:    "agent",
		PrincipalRole:    "orchestrator",
		ClientID:         "orch-client-global",
		ClientType:       "mcp-stdio",
		Reason:           "global session",
		ContinuationJSON: `{"resume_token":"resume-global"}`,
	}, "")

	decision, err := adapter.CheckAuthSessionGovernance(context.Background(), CheckAuthSessionGovernanceRequest{
		SessionID:           global.Request.IssuedSessionID,
		ActingSessionID:     acting.Request.IssuedSessionID,
		ActingSessionSecret: acting.SessionSecret,
	})
	if err != nil {
		t.Fatalf("CheckAuthSessionGovernance(global via multi project) error = %v", err)
	}
	if decision.Authorized {
		t.Fatalf("CheckAuthSessionGovernance(global via multi project) authorized = true, want false")
	}
	if got := decision.DecisionReason; got != "out_of_scope" {
		t.Fatalf("CheckAuthSessionGovernance(global via multi project) decision_reason = %q, want out_of_scope", got)
	}
	if got := decision.ActingPrincipalRole; got != "orchestrator" {
		t.Fatalf("CheckAuthSessionGovernance(global via multi project) acting_principal_role = %q, want orchestrator", got)
	}
	if got := decision.TargetSession.SessionID; got != global.Request.IssuedSessionID {
		t.Fatalf("CheckAuthSessionGovernance(global via multi project) target session = %q, want %q", got, global.Request.IssuedSessionID)
	}
}

// TestAppServiceAdapterRevokeAuthSessionAllowsSelfCleanup verifies non-orchestrator sessions
// may revoke themselves without gaining broader session-governance power.
func TestAppServiceAdapterRevokeAuthSessionAllowsSelfCleanup(t *testing.T) {
	adapter, _ := newAuthRequestAdapterForTest(t)

	builder := mustCreateApprovedAuthSessionForTest(t, adapter, CreateAuthRequestRequest{
		Path:             "project/p1/branch/b1",
		PrincipalID:      "builder-p1",
		PrincipalType:    "agent",
		PrincipalRole:    "builder",
		ClientID:         "builder-client-p1",
		ClientType:       "mcp-stdio",
		Reason:           "builder self cleanup",
		ContinuationJSON: `{"resume_token":"resume-builder-p1"}`,
	}, "")

	decision, err := adapter.CheckAuthSessionGovernance(context.Background(), CheckAuthSessionGovernanceRequest{
		SessionID:           builder.Request.IssuedSessionID,
		ActingSessionID:     builder.Request.IssuedSessionID,
		ActingSessionSecret: builder.SessionSecret,
	})
	if err != nil {
		t.Fatalf("CheckAuthSessionGovernance(self) error = %v", err)
	}
	if !decision.Authorized {
		t.Fatalf("CheckAuthSessionGovernance(self) authorized = false, want true")
	}
	if got := decision.DecisionReason; got != "self" {
		t.Fatalf("CheckAuthSessionGovernance(self) decision_reason = %q, want self", got)
	}

	revoked, err := adapter.RevokeAuthSession(context.Background(), RevokeAuthSessionRequest{
		SessionID:           builder.Request.IssuedSessionID,
		Reason:              "builder self cleanup",
		ActingSessionID:     builder.Request.IssuedSessionID,
		ActingSessionSecret: builder.SessionSecret,
	})
	if err != nil {
		t.Fatalf("RevokeAuthSession(self) error = %v", err)
	}
	if got := revoked.State; got != "revoked" {
		t.Fatalf("RevokeAuthSession(self) state = %q, want revoked", got)
	}
}

// TestAppServiceAdapterNonOrchestratorGovernanceFailsClosed verifies non-orchestrator sessions
// cannot inspect or inventory sibling sessions even when those sessions share project scope.
func TestAppServiceAdapterNonOrchestratorGovernanceFailsClosed(t *testing.T) {
	adapter, _ := newAuthRequestAdapterForTest(t)

	builder := mustCreateApprovedAuthSessionForTest(t, adapter, CreateAuthRequestRequest{
		Path:             "project/p1/branch/b1",
		PrincipalID:      "builder-p1",
		PrincipalType:    "agent",
		PrincipalRole:    "builder",
		ClientID:         "builder-client-p1",
		ClientType:       "mcp-stdio",
		Reason:           "builder session",
		ContinuationJSON: `{"resume_token":"resume-builder-p1"}`,
	}, "")
	qa := mustCreateApprovedAuthSessionForTest(t, adapter, CreateAuthRequestRequest{
		Path:             "project/p1/branch/b1",
		PrincipalID:      "qa-p1",
		PrincipalType:    "agent",
		PrincipalRole:    "qa",
		ClientID:         "qa-client-p1",
		ClientType:       "mcp-stdio",
		Reason:           "qa sibling session",
		ContinuationJSON: `{"resume_token":"resume-qa-p1"}`,
	}, "")

	decision, err := adapter.CheckAuthSessionGovernance(context.Background(), CheckAuthSessionGovernanceRequest{
		SessionID:           qa.Request.IssuedSessionID,
		ActingSessionID:     builder.Request.IssuedSessionID,
		ActingSessionSecret: builder.SessionSecret,
	})
	if err != nil {
		t.Fatalf("CheckAuthSessionGovernance(sibling via builder) error = %v", err)
	}
	if decision.Authorized {
		t.Fatalf("CheckAuthSessionGovernance(sibling via builder) authorized = true, want false")
	}
	if got := decision.DecisionReason; got != "role_denied" {
		t.Fatalf("CheckAuthSessionGovernance(sibling via builder) decision_reason = %q, want role_denied", got)
	}

	if _, err := adapter.ListAuthSessions(context.Background(), ListAuthSessionsRequest{
		ProjectID:           "p1",
		SessionID:           qa.Request.IssuedSessionID,
		ActingSessionID:     builder.Request.IssuedSessionID,
		ActingSessionSecret: builder.SessionSecret,
	}); !errors.Is(err, ErrAuthorizationDenied) {
		t.Fatalf("ListAuthSessions(sibling via builder) error = %v, want ErrAuthorizationDenied", err)
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

	_, err = adapter.CreateAuthRequest(context.Background(), CreateAuthRequestRequest{
		Path:             "project/p1",
		PrincipalID:      "review-agent",
		ClientID:         "till-mcp-stdio",
		Reason:           "manual MCP review",
		ContinuationJSON: `{"resume_tool":"till.action_item"}`,
	})
	if err == nil || !strings.Contains(err.Error(), "continuation_json.resume_token") {
		t.Fatalf("CreateAuthRequest() error = %v, want continuation_json.resume_token validation", err)
	}
}

// TestAppServiceAdapterCreateAuthRequestAutoGeneratesResumeToken verifies MCP create flows become claimable by default when continuation_json is omitted.
func TestAppServiceAdapterCreateAuthRequestAutoGeneratesResumeToken(t *testing.T) {
	adapter, _ := newAuthRequestAdapterForTest(t)

	created, err := adapter.CreateAuthRequest(context.Background(), CreateAuthRequestRequest{
		Path:             "project/p1",
		PrincipalID:      "review-agent",
		PrincipalType:    "agent",
		ClientID:         "till-mcp-stdio",
		ClientType:       "mcp-stdio",
		Reason:           "manual MCP review",
		RequestedByActor: "orchestrator-1",
		RequestedByType:  "agent",
	})
	if err != nil {
		t.Fatalf("CreateAuthRequest() error = %v", err)
	}
	if !created.HasContinuation {
		t.Fatal("CreateAuthRequest() has_continuation = false, want true")
	}
	resumeToken := strings.TrimSpace(authRequestResumeToken(created.Continuation))
	if resumeToken == "" {
		t.Fatal("CreateAuthRequest() continuation missing auto-generated resume_token")
	}
	if got := created.Continuation[app.AuthRequestContinuationRequesterClientIDKey]; got != "till-mcp-stdio" {
		t.Fatalf("CreateAuthRequest() continuation requester client = %#v, want till-mcp-stdio", got)
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

// TestAppServiceAdapterCreateAuthRequestDelegationRejectsNonOrchestrator verifies non-orchestrator
// acting sessions cannot mint sibling child auth requests for other principals.
func TestAppServiceAdapterCreateAuthRequestDelegationRejectsNonOrchestrator(t *testing.T) {
	adapter, _ := newAuthRequestAdapterForTest(t)

	builder := mustCreateApprovedAuthSessionForTest(t, adapter, CreateAuthRequestRequest{
		Path:             "project/p1/branch/build-1",
		PrincipalID:      "builder-1",
		PrincipalType:    "agent",
		PrincipalRole:    "builder",
		ClientID:         "builder-client",
		ClientType:       "mcp-stdio",
		Reason:           "builder branch scope",
		ContinuationJSON: `{"resume_token":"resume-builder"}`,
	}, "")

	_, err := adapter.CreateAuthRequest(context.Background(), CreateAuthRequestRequest{
		Path:                "project/p1/branch/build-1/phase/qa-pass",
		PrincipalID:         "qa-1",
		PrincipalType:       "agent",
		PrincipalRole:       "qa",
		ClientID:            "qa-client",
		ClientType:          "mcp-stdio",
		Reason:              "builder tries sibling delegation",
		ActingSessionID:     builder.Request.IssuedSessionID,
		ActingSessionSecret: builder.SessionSecret,
	})
	if !errors.Is(err, ErrAuthorizationDenied) {
		t.Fatalf("CreateAuthRequest(non-orchestrator delegation) error = %v, want ErrAuthorizationDenied", err)
	}
}

// TestAppServiceAdapterCreateAuthRequestDelegationRejectsBroaderScope verifies delegated
// child auth requests remain bounded by the acting approved path.
func TestAppServiceAdapterCreateAuthRequestDelegationRejectsBroaderScope(t *testing.T) {
	adapter, _ := newAuthRequestAdapterForTest(t)

	acting := mustCreateApprovedAuthSessionForTest(t, adapter, CreateAuthRequestRequest{
		Path:             "project/p1",
		PrincipalID:      "orchestrator-1",
		PrincipalType:    "agent",
		PrincipalRole:    "orchestrator",
		ClientID:         "orch-client",
		ClientType:       "mcp-stdio",
		Reason:           "orchestrator project scope",
		ContinuationJSON: `{"resume_token":"resume-orch"}`,
	}, "")

	_, err := adapter.CreateAuthRequest(context.Background(), CreateAuthRequestRequest{
		Path:                "projects/p1,p2",
		PrincipalID:         "orchestrator-2",
		PrincipalType:       "agent",
		PrincipalRole:       "orchestrator",
		ClientID:            "orch-child-client",
		ClientType:          "mcp-stdio",
		Reason:              "broader delegated scope",
		ActingSessionID:     acting.Request.IssuedSessionID,
		ActingSessionSecret: acting.SessionSecret,
	})
	if !errors.Is(err, ErrAuthorizationDenied) {
		t.Fatalf("CreateAuthRequest(broader delegation) error = %v, want ErrAuthorizationDenied", err)
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

// TestAppServiceAdapterCancelAuthRequestRejectsClaimantMismatch verifies cancel stays requester-bound even when a child principal can claim later.
func TestAppServiceAdapterCancelAuthRequestRejectsClaimantMismatch(t *testing.T) {
	adapter, _ := newAuthRequestAdapterForTest(t)

	request, err := adapter.CreateAuthRequest(context.Background(), CreateAuthRequestRequest{
		Path:              "project/p1",
		PrincipalID:       "child-agent",
		PrincipalType:     "agent",
		RequestedByActor:  "orchestrator-1",
		RequestedByType:   "agent",
		RequesterClientID: "orchestrator-client",
		ClientID:          "child-client",
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
		PrincipalID: "child-agent",
		ClientID:    "child-client",
	})
	if err == nil || !errors.Is(err, ErrInvalidCaptureStateRequest) {
		t.Fatalf("CancelAuthRequest(child claimant mismatch) error = %v, want ErrInvalidCaptureStateRequest", err)
	}
}

// TestAppServiceAdapterClaimAuthRequestRejectsClaimantMismatch verifies claim transport rejects attempts to adopt another caller's approved auth request.
func TestAppServiceAdapterClaimAuthRequestRejectsClaimantMismatch(t *testing.T) {
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
		t.Fatalf("ClaimAuthRequest(claimant mismatch) error = %v, want ErrInvalidCaptureStateRequest", err)
	}
}

// TestAppServiceAdapterApproveAuthRequestOrchSelfApproval verifies the
// adapter-level happy path for the orch-self-approval cascade landed in
// Drop 4a Wave 3 W3.1: an orchestrator with a project-scoped approved
// session approves a pending builder request and receives both the
// updated record and the issued subagent session secret.
func TestAppServiceAdapterApproveAuthRequestOrchSelfApproval(t *testing.T) {
	adapter, _ := newAuthRequestAdapterForTest(t)

	// Step 1: bootstrap an orchestrator session via the legacy dev-TUI
	// approval path (gate bypassed because all four approver-identity
	// fields are empty). This gives us the acting_session_id +
	// acting_session_secret pair the new approve operation needs.
	orchApproved := mustCreateApprovedAuthSessionForTest(t, adapter, CreateAuthRequestRequest{
		Path:             "project/p1",
		PrincipalID:      "ORCH_INSTANCE_42",
		PrincipalType:    "agent",
		PrincipalRole:    "orchestrator",
		PrincipalName:    "Drop Orchestrator",
		ClientID:         "till-mcp-stdio-orch",
		ClientType:       "mcp-stdio",
		ClientName:       "Drop Orch MCP",
		RequestedByActor: "lane-user",
		RequestedByType:  "user",
		Reason:           "drop-orch session for cascade approvals",
	}, "")
	if orchApproved.SessionSecret == "" {
		t.Fatal("orchestrator session secret missing from dev-TUI approve")
	}

	// Step 2: orchestrator delegates a builder auth request (created via
	// the same dev-TUI path so the approve gate has a pending request to
	// transition).
	pendingBuilder, err := adapter.CreateAuthRequest(context.Background(), CreateAuthRequestRequest{
		Path:             "project/p1",
		PrincipalID:      "BUILDER_AGENT_99",
		PrincipalType:    "agent",
		PrincipalRole:    "builder",
		PrincipalName:    "Builder Subagent",
		ClientID:         "till-mcp-stdio-builder",
		ClientType:       "mcp-stdio",
		ClientName:       "Builder MCP",
		RequestedByActor: "ORCH_INSTANCE_42",
		RequestedByType:  "agent",
		Reason:           "delegated builder for drop work",
	})
	if err != nil {
		t.Fatalf("CreateAuthRequest(builder) error = %v", err)
	}

	// Step 3: orch approves the builder request via the new approve
	// operation. Adapter pulls approver identity from the validated
	// session; caller supplies agent_instance_id + lease_token.
	result, err := adapter.ApproveAuthRequest(context.Background(), ApproveAuthRequestRequest{
		RequestID:           pendingBuilder.ID,
		ActingSessionID:     orchApproved.Request.IssuedSessionID,
		ActingSessionSecret: orchApproved.SessionSecret,
		AgentInstanceID:     "AGENT_INSTANCE_42",
		LeaseToken:          "LEASE_TOKEN_42",
		ResolutionNote:      "orch self-approval cascade",
	})
	if err != nil {
		t.Fatalf("ApproveAuthRequest() error = %v", err)
	}
	if got := result.Request.State; got != "approved" {
		t.Fatalf("ApproveAuthRequest() state = %q, want approved", got)
	}
	if result.SessionSecret == "" {
		t.Fatal("ApproveAuthRequest() returned empty SessionSecret")
	}
	if result.Request.IssuedSessionID == "" {
		t.Fatal("ApproveAuthRequest() returned empty IssuedSessionID")
	}
}

// TestAppServiceAdapterApproveAuthRequestRejectsMissingFields verifies the
// adapter rejects approve calls with any of the five required fields
// missing (request_id, acting_session_id, acting_session_secret,
// agent_instance_id, lease_token).
func TestAppServiceAdapterApproveAuthRequestRejectsMissingFields(t *testing.T) {
	adapter, _ := newAuthRequestAdapterForTest(t)
	full := ApproveAuthRequestRequest{
		RequestID:           "req-1",
		ActingSessionID:     "sess-1",
		ActingSessionSecret: "secret-1",
		AgentInstanceID:     "AGENT_INSTANCE_42",
		LeaseToken:          "LEASE_TOKEN_42",
	}
	cases := []struct {
		name string
		mut  func(r *ApproveAuthRequestRequest)
	}{
		{"request_id_empty", func(r *ApproveAuthRequestRequest) { r.RequestID = "" }},
		{"acting_session_id_empty", func(r *ApproveAuthRequestRequest) { r.ActingSessionID = "" }},
		{"acting_session_secret_empty", func(r *ApproveAuthRequestRequest) { r.ActingSessionSecret = "" }},
		{"agent_instance_id_empty", func(r *ApproveAuthRequestRequest) { r.AgentInstanceID = "" }},
		{"lease_token_empty", func(r *ApproveAuthRequestRequest) { r.LeaseToken = "" }},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			req := full
			tc.mut(&req)
			_, err := adapter.ApproveAuthRequest(context.Background(), req)
			if err == nil || !errors.Is(err, ErrInvalidCaptureStateRequest) {
				t.Fatalf("ApproveAuthRequest(%s) error = %v, want ErrInvalidCaptureStateRequest", tc.name, err)
			}
		})
	}
}

// TestAppServiceAdapterApproveAuthRequestHonorsProjectToggleDisabled
// verifies the Drop 4a Wave 3 W3.2 project-metadata opt-out toggle
// (Metadata.OrchSelfApprovalEnabled = *false) backstops the adapter-level
// orch-self-approval cascade. The toggle rejection surfaces with
// ErrOrchSelfApprovalDisabled BEFORE the role / path / cross-orch gate
// runs — even when the approver session and request would otherwise
// satisfy the W3.1 happy path.
func TestAppServiceAdapterApproveAuthRequestHonorsProjectToggleDisabled(t *testing.T) {
	adapter, repo := newAuthRequestAdapterForTest(t)

	// Step 1: bootstrap orch session via legacy dev-TUI approve path —
	// gate is bypassed because all four approver-identity fields are
	// empty. Mirrors TestAppServiceAdapterApproveAuthRequestOrchSelfApproval.
	orchApproved := mustCreateApprovedAuthSessionForTest(t, adapter, CreateAuthRequestRequest{
		Path:             "project/p1",
		PrincipalID:      "ORCH_INSTANCE_42",
		PrincipalType:    "agent",
		PrincipalRole:    "orchestrator",
		PrincipalName:    "Drop Orchestrator",
		ClientID:         "till-mcp-stdio-orch",
		ClientType:       "mcp-stdio",
		ClientName:       "Drop Orch MCP",
		RequestedByActor: "lane-user",
		RequestedByType:  "user",
		Reason:           "drop-orch session for cascade approvals",
	}, "")
	if orchApproved.SessionSecret == "" {
		t.Fatal("orchestrator session secret missing from dev-TUI approve")
	}

	// Step 2: flip the toggle on project p1 to *false via the repository.
	// Production callers use till.project(operation=update) — Metadata
	// flows through CreateProjectRequest / UpdateProjectRequest unchanged
	// because the request struct embeds domain.ProjectMetadata directly.
	// We bypass the mutation-guard context here so the test stays focused
	// on the gate's read of the toggle, not on the mutation pipeline.
	ctx := context.Background()
	project, err := repo.GetProject(ctx, "p1")
	if err != nil {
		t.Fatalf("GetProject(p1) error = %v", err)
	}
	disabled := false
	project.Metadata.OrchSelfApprovalEnabled = &disabled
	if err := repo.UpdateProject(ctx, project); err != nil {
		t.Fatalf("UpdateProject(toggle=false) error = %v", err)
	}

	// Sanity check: re-read the project and confirm the toggle persisted.
	reloaded, err := repo.GetProject(ctx, "p1")
	if err != nil {
		t.Fatalf("GetProject(p1, reload) error = %v", err)
	}
	if reloaded.Metadata.OrchSelfApprovalIsEnabled() {
		t.Fatalf("toggle did not round-trip; OrchSelfApprovalIsEnabled() = true after UpdateProject(*false)")
	}

	// Step 3: orch creates a delegated builder request and attempts
	// approval through the adapter. The toggle must reject before the
	// role / path / cross-orch gate runs.
	pendingBuilder, err := adapter.CreateAuthRequest(ctx, CreateAuthRequestRequest{
		Path:             "project/p1",
		PrincipalID:      "BUILDER_AGENT_99",
		PrincipalType:    "agent",
		PrincipalRole:    "builder",
		PrincipalName:    "Builder Subagent",
		ClientID:         "till-mcp-stdio-builder",
		ClientType:       "mcp-stdio",
		ClientName:       "Builder MCP",
		RequestedByActor: "ORCH_INSTANCE_42",
		RequestedByType:  "agent",
		Reason:           "delegated builder for drop work",
	})
	if err != nil {
		t.Fatalf("CreateAuthRequest(builder) error = %v", err)
	}

	_, err = adapter.ApproveAuthRequest(ctx, ApproveAuthRequestRequest{
		RequestID:           pendingBuilder.ID,
		ActingSessionID:     orchApproved.Request.IssuedSessionID,
		ActingSessionSecret: orchApproved.SessionSecret,
		AgentInstanceID:     "AGENT_INSTANCE_42",
		LeaseToken:          "LEASE_TOKEN_42",
		ResolutionNote:      "toggle-disabled cascade attempt",
	})
	if !errors.Is(err, domain.ErrOrchSelfApprovalDisabled) {
		t.Fatalf("ApproveAuthRequest(toggle-disabled) error = %v, want ErrOrchSelfApprovalDisabled", err)
	}

	// Step 4: flip the toggle back to nil (or *true). The same approve
	// call now succeeds — proving the toggle, not some unrelated gate
	// drift, was the cause of the rejection.
	reloaded.Metadata.OrchSelfApprovalEnabled = nil
	if err := repo.UpdateProject(ctx, reloaded); err != nil {
		t.Fatalf("UpdateProject(toggle=nil) error = %v", err)
	}

	// Re-create the request because the previous one is still pending but
	// the gate fetches it again; using a fresh request keeps the assertion
	// independent of any partial-state side effects from the rejected call.
	pendingBuilder2, err := adapter.CreateAuthRequest(ctx, CreateAuthRequestRequest{
		Path:             "project/p1",
		PrincipalID:      "BUILDER_AGENT_99B",
		PrincipalType:    "agent",
		PrincipalRole:    "builder",
		PrincipalName:    "Builder Subagent B",
		ClientID:         "till-mcp-stdio-builder-b",
		ClientType:       "mcp-stdio",
		ClientName:       "Builder MCP B",
		RequestedByActor: "ORCH_INSTANCE_42",
		RequestedByType:  "agent",
		Reason:           "post-toggle-reset cascade",
	})
	if err != nil {
		t.Fatalf("CreateAuthRequest(builder-2) error = %v", err)
	}
	result, err := adapter.ApproveAuthRequest(ctx, ApproveAuthRequestRequest{
		RequestID:           pendingBuilder2.ID,
		ActingSessionID:     orchApproved.Request.IssuedSessionID,
		ActingSessionSecret: orchApproved.SessionSecret,
		AgentInstanceID:     "AGENT_INSTANCE_42",
		LeaseToken:          "LEASE_TOKEN_42",
		ResolutionNote:      "post-toggle-reset cascade",
	})
	if err != nil {
		t.Fatalf("ApproveAuthRequest(post-toggle-reset) error = %v, want success", err)
	}
	if result.Request.State != "approved" {
		t.Fatalf("ApproveAuthRequest(post-toggle-reset) state = %q, want approved", result.Request.State)
	}
}

// TestAuthRequestRecordSurfacesApprovingOrchIdentity verifies the Drop 4a
// Wave 3 W3.3 audit-trail surface: GetAuthRequest after an orch-self-approval
// returns a record whose JSON encodes all three approving_* fields, and
// after a dev-TUI approval the record encodes the omitempty-elided shape
// (no approving_* keys present).
func TestAuthRequestRecordSurfacesApprovingOrchIdentity(t *testing.T) {
	adapter, _ := newAuthRequestAdapterForTest(t)
	ctx := context.Background()

	// Bootstrap an orchestrator session via the dev-TUI path.
	orchApproved := mustCreateApprovedAuthSessionForTest(t, adapter, CreateAuthRequestRequest{
		Path:             "project/p1",
		PrincipalID:      "ORCH_INSTANCE_W33",
		PrincipalType:    "agent",
		PrincipalRole:    "orchestrator",
		PrincipalName:    "Drop W3.3 Orchestrator",
		ClientID:         "till-mcp-stdio-orch",
		ClientType:       "mcp-stdio",
		ClientName:       "Orch MCP",
		RequestedByActor: "lane-user",
		RequestedByType:  "user",
		Reason:           "W3.3 audit-trail orch session",
	}, "")
	if orchApproved.SessionSecret == "" {
		t.Fatal("orchestrator session secret missing from dev-TUI approve")
	}

	// Cascade orch-approves a pending builder request — populates audit
	// fields end-to-end.
	pendingBuilder, err := adapter.CreateAuthRequest(ctx, CreateAuthRequestRequest{
		Path:             "project/p1",
		PrincipalID:      "BUILDER_AGENT_W33",
		PrincipalType:    "agent",
		PrincipalRole:    "builder",
		PrincipalName:    "Builder Subagent",
		ClientID:         "till-mcp-stdio-builder",
		ClientType:       "mcp-stdio",
		ClientName:       "Builder MCP",
		RequestedByActor: "ORCH_INSTANCE_W33",
		RequestedByType:  "agent",
		Reason:           "delegated builder for W3.3 audit-trail",
	})
	if err != nil {
		t.Fatalf("CreateAuthRequest(builder) error = %v", err)
	}
	approveResult, err := adapter.ApproveAuthRequest(ctx, ApproveAuthRequestRequest{
		RequestID:           pendingBuilder.ID,
		ActingSessionID:     orchApproved.Request.IssuedSessionID,
		ActingSessionSecret: orchApproved.SessionSecret,
		AgentInstanceID:     "AGENT_INSTANCE_W33",
		LeaseToken:          "LEASE_TOKEN_W33",
		ResolutionNote:      "W3.3 audit-trail cascade",
	})
	if err != nil {
		t.Fatalf("ApproveAuthRequest() error = %v", err)
	}
	if approveResult.Request.ApprovingPrincipalID != "ORCH_INSTANCE_W33" ||
		approveResult.Request.ApprovingAgentInstanceID != "AGENT_INSTANCE_W33" ||
		approveResult.Request.ApprovingLeaseToken != "LEASE_TOKEN_W33" {
		t.Fatalf("ApproveAuthRequest() approving fields = %q/%q/%q, want orch identity round-trip",
			approveResult.Request.ApprovingPrincipalID,
			approveResult.Request.ApprovingAgentInstanceID,
			approveResult.Request.ApprovingLeaseToken,
		)
	}

	// Adapter GetAuthRequest round-trip: the persisted columns surface back
	// onto the transport-facing record.
	gotOrch, err := adapter.GetAuthRequest(ctx, pendingBuilder.ID)
	if err != nil {
		t.Fatalf("GetAuthRequest(orch-approved) error = %v", err)
	}
	if gotOrch.ApprovingPrincipalID != "ORCH_INSTANCE_W33" ||
		gotOrch.ApprovingAgentInstanceID != "AGENT_INSTANCE_W33" ||
		gotOrch.ApprovingLeaseToken != "LEASE_TOKEN_W33" {
		t.Fatalf("GetAuthRequest(orch-approved) approving fields = %q/%q/%q, want round-tripped",
			gotOrch.ApprovingPrincipalID,
			gotOrch.ApprovingAgentInstanceID,
			gotOrch.ApprovingLeaseToken,
		)
	}

	encodedOrch, err := json.Marshal(gotOrch)
	if err != nil {
		t.Fatalf("json.Marshal(orch-approved) error = %v", err)
	}
	for _, key := range []string{
		`"approving_principal_id":"ORCH_INSTANCE_W33"`,
		`"approving_agent_instance_id":"AGENT_INSTANCE_W33"`,
		`"approving_lease_token":"LEASE_TOKEN_W33"`,
	} {
		if !strings.Contains(string(encodedOrch), key) {
			t.Fatalf("json(orch-approved) = %s, want to contain %s", encodedOrch, key)
		}
	}

	// Counter-case: dev-TUI approval emits omitempty-elided JSON shape.
	pendingTUI, err := adapter.CreateAuthRequest(ctx, CreateAuthRequestRequest{
		Path:             "project/p1",
		PrincipalID:      "BUILDER_AGENT_TUI",
		PrincipalType:    "agent",
		PrincipalRole:    "builder",
		ClientID:         "till-mcp-stdio-tui",
		ClientType:       "mcp-stdio",
		RequestedByActor: "lane-user",
		RequestedByType:  "user",
		Reason:           "dev-TUI counter-case",
	})
	if err != nil {
		t.Fatalf("CreateAuthRequest(tui) error = %v", err)
	}
	if _, err := adapter.service.ApproveAuthRequest(ctx, app.ApproveAuthRequestInput{
		RequestID:      pendingTUI.ID,
		ResolvedBy:     "operator-1",
		ResolvedType:   domain.ActorTypeUser,
		ResolutionNote: "dev TUI",
	}); err != nil {
		t.Fatalf("ApproveAuthRequest(tui) error = %v", err)
	}
	gotTUI, err := adapter.GetAuthRequest(ctx, pendingTUI.ID)
	if err != nil {
		t.Fatalf("GetAuthRequest(tui) error = %v", err)
	}
	if gotTUI.ApprovingPrincipalID != "" || gotTUI.ApprovingAgentInstanceID != "" || gotTUI.ApprovingLeaseToken != "" {
		t.Fatalf("GetAuthRequest(tui) approving fields = %q/%q/%q, want all empty",
			gotTUI.ApprovingPrincipalID,
			gotTUI.ApprovingAgentInstanceID,
			gotTUI.ApprovingLeaseToken,
		)
	}
	encodedTUI, err := json.Marshal(gotTUI)
	if err != nil {
		t.Fatalf("json.Marshal(tui) error = %v", err)
	}
	for _, key := range []string{
		`approving_principal_id`,
		`approving_agent_instance_id`,
		`approving_lease_token`,
	} {
		if strings.Contains(string(encodedTUI), key) {
			t.Fatalf("json(tui) = %s, want omitempty to elide %s", encodedTUI, key)
		}
	}
}
