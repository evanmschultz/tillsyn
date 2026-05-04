package app_test

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/evanmschultz/tillsyn/internal/adapters/auth/autentauth"
	"github.com/evanmschultz/tillsyn/internal/adapters/storage/sqlite"
	"github.com/evanmschultz/tillsyn/internal/app"
	"github.com/evanmschultz/tillsyn/internal/domain"
)

// authRequestServiceFixture stores one real service stack for auth-request lifecycle tests.
type authRequestServiceFixture struct {
	svc     *app.Service
	repo    *sqlite.Repository
	project domain.Project
	auth    *autentauth.Service
}

// newAuthRequestServiceFixture constructs one real app/auth/sqlite stack for lifecycle tests.
func newAuthRequestServiceFixture(t *testing.T) authRequestServiceFixture {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "tillsyn.db")
	repo, err := sqlite.Open(dbPath)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(func() {
		_ = repo.Close()
	})

	now := time.Date(2026, 3, 20, 12, 0, 0, 0, time.UTC)
	nextID := 0
	next := func() string {
		nextID++
		return fmt.Sprintf("id-%02d", nextID)
	}

	auth, err := autentauth.NewSharedDB(autentauth.Config{
		DB:          repo.DB(),
		IDGenerator: next,
		Clock:       time.Now,
	})
	if err != nil {
		t.Fatalf("NewSharedDB() error = %v", err)
	}
	if err := auth.EnsureDogfoodPolicy(context.Background()); err != nil {
		t.Fatalf("EnsureDogfoodPolicy() error = %v", err)
	}

	project, err := domain.NewProjectFromInput(domain.ProjectInput{ID: "p1", Name: "Project One"}, now)
	if err != nil {
		t.Fatalf("NewProjectFromInput() error = %v", err)
	}
	if err := repo.CreateProject(context.Background(), project); err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}

	svc := app.NewService(repo, next, time.Now, app.ServiceConfig{
		AuthRequests: auth,
		AuthBackend:  auth,
	})
	return authRequestServiceFixture{
		svc:     svc,
		repo:    repo,
		project: project,
		auth:    auth,
	}
}

// TestServiceAuthRequestApproveMirrorsAttention verifies create and approve flows mirror into attention and issue a usable session.
func TestServiceAuthRequestApproveMirrorsAttention(t *testing.T) {
	fixture := newAuthRequestServiceFixture(t)

	request, err := fixture.svc.CreateAuthRequest(context.Background(), app.CreateAuthRequestInput{
		Path:                "project/" + fixture.project.ID,
		PrincipalID:         "review-agent",
		PrincipalType:       "agent",
		PrincipalName:       "Review Agent",
		ClientID:            "till-mcp-stdio",
		ClientType:          "mcp-stdio",
		ClientName:          "Till MCP STDIO",
		RequestedSessionTTL: 2 * time.Hour,
		Reason:              "manual MCP review",
		RequestedBy:         "lane-user",
		RequestedType:       domain.ActorTypeUser,
		Timeout:             10 * time.Minute,
	})
	if err != nil {
		t.Fatalf("CreateAuthRequest() error = %v", err)
	}
	if got := request.State; got != domain.AuthRequestStatePending {
		t.Fatalf("CreateAuthRequest() state = %q, want pending", got)
	}

	attention, err := fixture.svc.ListAttentionItems(context.Background(), app.ListAttentionItemsInput{
		Level: domain.LevelTupleInput{
			ProjectID: fixture.project.ID,
			ScopeType: domain.ScopeLevelProject,
			ScopeID:   fixture.project.ID,
		},
		UnresolvedOnly: true,
	})
	if err != nil {
		t.Fatalf("ListAttentionItems(open) error = %v", err)
	}
	if len(attention) != 1 || attention[0].ID != request.ID {
		t.Fatalf("expected mirrored attention for request %q, got %#v", request.ID, attention)
	}
	if !attention[0].RequiresUserAction {
		t.Fatal("expected mirrored attention item to require user action")
	}

	approved, err := fixture.svc.ApproveAuthRequest(context.Background(), app.ApproveAuthRequestInput{
		RequestID:      request.ID,
		ResolvedBy:     "approver-1",
		ResolvedType:   domain.ActorTypeUser,
		ResolutionNote: "approved for dogfood",
	})
	if err != nil {
		t.Fatalf("ApproveAuthRequest() error = %v", err)
	}
	if got := approved.Request.State; got != domain.AuthRequestStateApproved {
		t.Fatalf("ApproveAuthRequest() state = %q, want approved", got)
	}
	if approved.SessionSecret == "" {
		t.Fatal("ApproveAuthRequest() returned empty session secret")
	}

	resolvedAttention, err := fixture.repo.GetAttentionItem(context.Background(), request.ID)
	if err != nil {
		t.Fatalf("GetAttentionItem() error = %v", err)
	}
	if got := resolvedAttention.State; got != domain.AttentionStateResolved {
		t.Fatalf("attention state = %q, want resolved", got)
	}

	validated, err := fixture.svc.ValidateAuthSession(context.Background(), approved.Request.IssuedSessionID, approved.SessionSecret)
	if err != nil {
		t.Fatalf("ValidateAuthSession() error = %v", err)
	}
	if got := validated.Session.PrincipalID; got != "review-agent" {
		t.Fatalf("validated principal_id = %q, want review-agent", got)
	}
	if got := validated.Session.ClientID; got != "till-mcp-stdio" {
		t.Fatalf("validated client_id = %q, want till-mcp-stdio", got)
	}
}

// TestServiceClaimAuthRequestReturnsApprovedSecret verifies continuation-based claim returns the approved secret only when the resume token matches.
func TestServiceClaimAuthRequestReturnsApprovedSecret(t *testing.T) {
	fixture := newAuthRequestServiceFixture(t)

	request, err := fixture.svc.CreateAuthRequest(context.Background(), app.CreateAuthRequestInput{
		Path:                "project/" + fixture.project.ID,
		PrincipalID:         "review-agent",
		PrincipalType:       "agent",
		ClientID:            "till-mcp-stdio",
		ClientType:          "mcp-stdio",
		RequestedSessionTTL: 2 * time.Hour,
		Reason:              "resume after approval",
		Continuation:        map[string]any{"resume_token": "resume-123", "resume_tool": "till.raise_attention_item"},
		RequestedBy:         "review-agent",
		RequesterClientID:   "till-mcp-stdio",
	})
	if err != nil {
		t.Fatalf("CreateAuthRequest() error = %v", err)
	}
	approved, err := fixture.svc.ApproveAuthRequest(context.Background(), app.ApproveAuthRequestInput{
		RequestID:      request.ID,
		ResolvedBy:     "approver-1",
		ResolvedType:   domain.ActorTypeUser,
		ResolutionNote: "approved for resume",
	})
	if err != nil {
		t.Fatalf("ApproveAuthRequest() error = %v", err)
	}

	claimed, err := fixture.svc.ClaimAuthRequest(context.Background(), app.ClaimAuthRequestInput{
		RequestID:   request.ID,
		ResumeToken: "resume-123",
		PrincipalID: "review-agent",
		ClientID:    "till-mcp-stdio",
	})
	if err != nil {
		t.Fatalf("ClaimAuthRequest() error = %v", err)
	}
	if got := claimed.Request.State; got != domain.AuthRequestStateApproved {
		t.Fatalf("ClaimAuthRequest() state = %q, want approved", got)
	}
	if got := claimed.Request.IssuedSessionID; got != approved.Request.IssuedSessionID {
		t.Fatalf("ClaimAuthRequest() issued_session_id = %q, want %q", got, approved.Request.IssuedSessionID)
	}
	if got := claimed.SessionSecret; got != approved.SessionSecret {
		t.Fatalf("ClaimAuthRequest() session_secret = %q, want approved secret", got)
	}
}

// TestServiceClaimAuthRequestRejectsWrongResumeToken verifies continuation claim fails closed when the requester token does not match.
func TestServiceClaimAuthRequestRejectsWrongResumeToken(t *testing.T) {
	fixture := newAuthRequestServiceFixture(t)

	request, err := fixture.svc.CreateAuthRequest(context.Background(), app.CreateAuthRequestInput{
		Path:              "project/" + fixture.project.ID,
		PrincipalID:       "review-agent",
		ClientID:          "till-mcp-stdio",
		ClientType:        "mcp-stdio",
		Reason:            "resume after approval",
		Continuation:      map[string]any{"resume_token": "resume-123"},
		RequestedBy:       "review-agent",
		RequesterClientID: "till-mcp-stdio",
	})
	if err != nil {
		t.Fatalf("CreateAuthRequest() error = %v", err)
	}
	if _, err := fixture.svc.ClaimAuthRequest(context.Background(), app.ClaimAuthRequestInput{
		RequestID:   request.ID,
		ResumeToken: "wrong-token",
		PrincipalID: "review-agent",
		ClientID:    "till-mcp-stdio",
	}); err == nil || err != domain.ErrInvalidAuthContinuation {
		t.Fatalf("ClaimAuthRequest() error = %v, want ErrInvalidAuthContinuation", err)
	}
}

// TestServiceClaimAuthRequestRejectsMismatchedRequesterIdentity verifies continuation claims fail closed when the requester identity does not match the request.
func TestServiceClaimAuthRequestRejectsMismatchedRequesterIdentity(t *testing.T) {
	fixture := newAuthRequestServiceFixture(t)

	request, err := fixture.svc.CreateAuthRequest(context.Background(), app.CreateAuthRequestInput{
		Path:              "project/" + fixture.project.ID,
		PrincipalID:       "review-agent",
		PrincipalType:     "agent",
		ClientID:          "till-mcp-stdio",
		ClientType:        "mcp-stdio",
		Reason:            "resume after approval",
		Continuation:      map[string]any{"resume_token": "resume-123"},
		RequestedBy:       "review-agent",
		RequesterClientID: "till-mcp-stdio",
	})
	if err != nil {
		t.Fatalf("CreateAuthRequest() error = %v", err)
	}
	if _, err := fixture.svc.ClaimAuthRequest(context.Background(), app.ClaimAuthRequestInput{
		RequestID:   request.ID,
		ResumeToken: "resume-123",
		PrincipalID: "other-agent",
		ClientID:    "other-client",
	}); err == nil || err != domain.ErrAuthRequestClaimMismatch {
		t.Fatalf("ClaimAuthRequest() error = %v, want ErrAuthRequestClaimMismatch", err)
	}
}

// TestServiceClaimAuthRequestRejectsRequesterOverride verifies delegated continuation
// claims fail closed when the original requester tries to adopt the approved child claim.
func TestServiceClaimAuthRequestRejectsRequesterOverride(t *testing.T) {
	fixture := newAuthRequestServiceFixture(t)

	request, err := fixture.svc.CreateAuthRequest(context.Background(), app.CreateAuthRequestInput{
		Path:              "project/" + fixture.project.ID,
		PrincipalID:       "builder-1",
		PrincipalType:     "agent",
		PrincipalRole:     "builder",
		ClientID:          "builder-client",
		ClientType:        "mcp-stdio",
		Reason:            "orchestrator requests builder scope",
		Continuation:      map[string]any{"resume_token": "resume-456"},
		RequestedBy:       "orchestrator-1",
		RequestedType:     domain.ActorTypeAgent,
		RequesterClientID: "orchestrator-client",
	})
	if err != nil {
		t.Fatalf("CreateAuthRequest() error = %v", err)
	}
	if _, err := fixture.svc.ApproveAuthRequest(context.Background(), app.ApproveAuthRequestInput{
		RequestID:      request.ID,
		ResolvedBy:     "approver-1",
		ResolvedType:   domain.ActorTypeUser,
		ResolutionNote: "approved for builder handoff",
	}); err != nil {
		t.Fatalf("ApproveAuthRequest() error = %v", err)
	}

	if _, err := fixture.svc.ClaimAuthRequest(context.Background(), app.ClaimAuthRequestInput{
		RequestID:   request.ID,
		ResumeToken: "resume-456",
		PrincipalID: "orchestrator-1",
		ClientID:    "orchestrator-client",
	}); !errors.Is(err, domain.ErrAuthRequestClaimMismatch) {
		t.Fatalf("ClaimAuthRequest() error = %v, want ErrAuthRequestClaimMismatch", err)
	}
}

// TestServiceClaimAuthRequestAllowsChildSelfClaim verifies an on-behalf-of child can claim its own approved request directly.
func TestServiceClaimAuthRequestAllowsChildSelfClaim(t *testing.T) {
	fixture := newAuthRequestServiceFixture(t)

	request, err := fixture.svc.CreateAuthRequest(context.Background(), app.CreateAuthRequestInput{
		Path:              "project/" + fixture.project.ID,
		PrincipalID:       "builder-1",
		PrincipalType:     "agent",
		PrincipalRole:     "builder",
		ClientID:          "builder-client",
		ClientType:        "mcp-stdio",
		Reason:            "child accepts delegated approval",
		Continuation:      map[string]any{"resume_token": "resume-789"},
		RequestedBy:       "orchestrator-1",
		RequestedType:     domain.ActorTypeAgent,
		RequesterClientID: "orchestrator-client",
	})
	if err != nil {
		t.Fatalf("CreateAuthRequest() error = %v", err)
	}
	approved, err := fixture.svc.ApproveAuthRequest(context.Background(), app.ApproveAuthRequestInput{
		RequestID:      request.ID,
		ResolvedBy:     "approver-1",
		ResolvedType:   domain.ActorTypeUser,
		ResolutionNote: "approved for child claim",
	})
	if err != nil {
		t.Fatalf("ApproveAuthRequest() error = %v", err)
	}

	claimed, err := fixture.svc.ClaimAuthRequest(context.Background(), app.ClaimAuthRequestInput{
		RequestID:   request.ID,
		ResumeToken: "resume-789",
		PrincipalID: "builder-1",
		ClientID:    "builder-client",
	})
	if err != nil {
		t.Fatalf("ClaimAuthRequest() error = %v", err)
	}
	if got := claimed.Request.RequestedByActor; got != "orchestrator-1" {
		t.Fatalf("ClaimAuthRequest() requested_by_actor = %q, want orchestrator-1", got)
	}
	if got := claimed.Request.PrincipalID; got != "builder-1" {
		t.Fatalf("ClaimAuthRequest() principal_id = %q, want builder-1", got)
	}
	if got := claimed.SessionSecret; got != approved.SessionSecret {
		t.Fatalf("ClaimAuthRequest() session_secret = %q, want approved secret", got)
	}
}

// TestServiceClaimAuthRequestWaitsForPendingResolution verifies continuation claims can hold the channel open and return a pending waiting state when unresolved.
func TestServiceClaimAuthRequestWaitsForPendingResolution(t *testing.T) {
	fixture := newAuthRequestServiceFixture(t)

	request, err := fixture.svc.CreateAuthRequest(context.Background(), app.CreateAuthRequestInput{
		Path:              "project/" + fixture.project.ID,
		PrincipalID:       "builder-1",
		PrincipalType:     "agent",
		PrincipalRole:     "builder",
		ClientID:          "builder-client",
		ClientType:        "mcp-stdio",
		Reason:            "resume after approval",
		Continuation:      map[string]any{"resume_token": "resume-123"},
		RequestedBy:       "orchestrator-1",
		RequestedType:     domain.ActorTypeAgent,
		RequesterClientID: "orchestrator-client",
	})
	if err != nil {
		t.Fatalf("CreateAuthRequest() error = %v", err)
	}
	claimed, err := fixture.svc.ClaimAuthRequest(context.Background(), app.ClaimAuthRequestInput{
		RequestID:   request.ID,
		ResumeToken: "resume-123",
		PrincipalID: "builder-1",
		ClientID:    "builder-client",
		WaitTimeout: 50 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("ClaimAuthRequest() error = %v", err)
	}
	if got := claimed.Request.State; got != domain.AuthRequestStatePending {
		t.Fatalf("ClaimAuthRequest() state = %q, want pending", got)
	}
	if !claimed.Waiting {
		t.Fatal("ClaimAuthRequest() waiting = false, want true")
	}
	if got := claimed.SessionSecret; got != "" {
		t.Fatalf("ClaimAuthRequest() session_secret = %q, want empty while pending", got)
	}
}

// TestServiceClaimAuthRequestWakesOnApproval verifies one waiting continuation claim resumes when the request is approved.
func TestServiceClaimAuthRequestWakesOnApproval(t *testing.T) {
	fixture := newAuthRequestServiceFixture(t)

	request, err := fixture.svc.CreateAuthRequest(context.Background(), app.CreateAuthRequestInput{
		Path:              "project/" + fixture.project.ID,
		PrincipalID:       "review-agent",
		PrincipalType:     "agent",
		ClientID:          "till-mcp-stdio",
		ClientType:        "mcp-stdio",
		Reason:            "resume after approval",
		Continuation:      map[string]any{"resume_token": "resume-123"},
		RequestedBy:       "review-agent",
		RequesterClientID: "till-mcp-stdio",
	})
	if err != nil {
		t.Fatalf("CreateAuthRequest() error = %v", err)
	}

	claimedCh := make(chan app.ClaimedAuthRequestResult, 1)
	errCh := make(chan error, 1)
	go func() {
		claimed, claimErr := fixture.svc.ClaimAuthRequest(context.Background(), app.ClaimAuthRequestInput{
			RequestID:   request.ID,
			ResumeToken: "resume-123",
			PrincipalID: "review-agent",
			ClientID:    "till-mcp-stdio",
			WaitTimeout: 2 * time.Second,
		})
		if claimErr != nil {
			errCh <- claimErr
			return
		}
		claimedCh <- claimed
	}()

	// The waiter should remain blocked until a terminal auth decision is published.
	select {
	case err := <-errCh:
		t.Fatalf("ClaimAuthRequest() early error = %v", err)
	case claimed := <-claimedCh:
		t.Fatalf("ClaimAuthRequest() returned early with %#v before approval", claimed)
	case <-time.After(50 * time.Millisecond):
	}

	approved, err := fixture.svc.ApproveAuthRequest(context.Background(), app.ApproveAuthRequestInput{
		RequestID:      request.ID,
		ResolvedBy:     "approver-1",
		ResolvedType:   domain.ActorTypeUser,
		ResolutionNote: "approved for live wait",
	})
	if err != nil {
		t.Fatalf("ApproveAuthRequest() error = %v", err)
	}

	select {
	case err := <-errCh:
		t.Fatalf("ClaimAuthRequest() error = %v", err)
	case claimed := <-claimedCh:
		if got := claimed.Request.State; got != domain.AuthRequestStateApproved {
			t.Fatalf("ClaimAuthRequest() state = %q, want approved", got)
		}
		if got := claimed.Request.IssuedSessionID; got != approved.Request.IssuedSessionID {
			t.Fatalf("ClaimAuthRequest() issued_session_id = %q, want %q", got, approved.Request.IssuedSessionID)
		}
		if got := claimed.SessionSecret; got != approved.SessionSecret {
			t.Fatalf("ClaimAuthRequest() session_secret = %q, want approved secret", got)
		}
		if claimed.Waiting {
			t.Fatal("ClaimAuthRequest() waiting = true, want false after approval")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("ClaimAuthRequest() did not wake after approval")
	}
}

// TestServiceClaimAuthRequestWakesOnDeny verifies one waiting continuation claim resumes with a denied result and no secret.
func TestServiceClaimAuthRequestWakesOnDeny(t *testing.T) {
	fixture := newAuthRequestServiceFixture(t)

	request, err := fixture.svc.CreateAuthRequest(context.Background(), app.CreateAuthRequestInput{
		Path:              "project/" + fixture.project.ID,
		PrincipalID:       "review-agent",
		PrincipalType:     "agent",
		ClientID:          "till-mcp-stdio",
		ClientType:        "mcp-stdio",
		Reason:            "resume after denial",
		Continuation:      map[string]any{"resume_token": "resume-123"},
		RequestedBy:       "review-agent",
		RequesterClientID: "till-mcp-stdio",
	})
	if err != nil {
		t.Fatalf("CreateAuthRequest() error = %v", err)
	}

	claimedCh := make(chan app.ClaimedAuthRequestResult, 1)
	errCh := make(chan error, 1)
	go func() {
		claimed, claimErr := fixture.svc.ClaimAuthRequest(context.Background(), app.ClaimAuthRequestInput{
			RequestID:   request.ID,
			ResumeToken: "resume-123",
			PrincipalID: "review-agent",
			ClientID:    "till-mcp-stdio",
			WaitTimeout: 2 * time.Second,
		})
		if claimErr != nil {
			errCh <- claimErr
			return
		}
		claimedCh <- claimed
	}()

	select {
	case err := <-errCh:
		t.Fatalf("ClaimAuthRequest() early error = %v", err)
	case claimed := <-claimedCh:
		t.Fatalf("ClaimAuthRequest() returned early with %#v before denial", claimed)
	case <-time.After(50 * time.Millisecond):
	}

	denied, err := fixture.svc.DenyAuthRequest(context.Background(), app.DenyAuthRequestInput{
		RequestID:      request.ID,
		ResolvedBy:     "approver-1",
		ResolvedType:   domain.ActorTypeUser,
		ResolutionNote: "denied for live wait",
	})
	if err != nil {
		t.Fatalf("DenyAuthRequest() error = %v", err)
	}

	select {
	case err := <-errCh:
		t.Fatalf("ClaimAuthRequest() error = %v", err)
	case claimed := <-claimedCh:
		if got := claimed.Request.State; got != domain.AuthRequestStateDenied {
			t.Fatalf("ClaimAuthRequest() state = %q, want denied", got)
		}
		if got := claimed.Request.ResolutionNote; got != denied.ResolutionNote {
			t.Fatalf("ClaimAuthRequest() resolution_note = %q, want %q", got, denied.ResolutionNote)
		}
		if got := claimed.SessionSecret; got != "" {
			t.Fatalf("ClaimAuthRequest() session_secret = %q, want empty after denial", got)
		}
		if claimed.Waiting {
			t.Fatal("ClaimAuthRequest() waiting = true, want false after denial")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("ClaimAuthRequest() did not wake after denial")
	}
}

// TestServiceClaimAuthRequestRejectsNegativeWaitTimeout verifies the app-facing wait contract fails closed on negative durations.
func TestServiceClaimAuthRequestRejectsNegativeWaitTimeout(t *testing.T) {
	fixture := newAuthRequestServiceFixture(t)

	request, err := fixture.svc.CreateAuthRequest(context.Background(), app.CreateAuthRequestInput{
		Path:              "project/" + fixture.project.ID,
		PrincipalID:       "review-agent",
		ClientID:          "till-mcp-stdio",
		Reason:            "resume after approval",
		Continuation:      map[string]any{"resume_token": "resume-123"},
		RequestedBy:       "review-agent",
		RequesterClientID: "till-mcp-stdio",
	})
	if err != nil {
		t.Fatalf("CreateAuthRequest() error = %v", err)
	}

	if _, err := fixture.svc.ClaimAuthRequest(context.Background(), app.ClaimAuthRequestInput{
		RequestID:   request.ID,
		ResumeToken: "resume-123",
		PrincipalID: "review-agent",
		ClientID:    "till-mcp-stdio",
		WaitTimeout: -time.Second,
	}); err == nil || err.Error() != "wait timeout must be >= 0" {
		t.Fatalf("ClaimAuthRequest() error = %v, want negative wait timeout validation", err)
	}
}

// TestServiceAuthRequestDenyAndCancelResolveAttention verifies non-approved terminal states also resolve their mirrored notifications.
func TestServiceAuthRequestDenyAndCancelResolveAttention(t *testing.T) {
	fixture := newAuthRequestServiceFixture(t)

	createRequest := func(principalID string) domain.AuthRequest {
		t.Helper()
		request, err := fixture.svc.CreateAuthRequest(context.Background(), app.CreateAuthRequestInput{
			Path:          "project/" + fixture.project.ID,
			PrincipalID:   principalID,
			PrincipalType: "user",
			ClientID:      "till-tui",
			ClientType:    "tui",
			Reason:        "review access",
		})
		if err != nil {
			t.Fatalf("CreateAuthRequest(%q) error = %v", principalID, err)
		}
		return request
	}

	deniedRequest := createRequest("user-deny")
	denied, err := fixture.svc.DenyAuthRequest(context.Background(), app.DenyAuthRequestInput{
		RequestID:      deniedRequest.ID,
		ResolvedBy:     "operator-1",
		ResolvedType:   domain.ActorTypeUser,
		ResolutionNote: "outside approved scope",
	})
	if err != nil {
		t.Fatalf("DenyAuthRequest() error = %v", err)
	}
	if got := denied.State; got != domain.AuthRequestStateDenied {
		t.Fatalf("DenyAuthRequest() state = %q, want denied", got)
	}

	canceledRequest := createRequest("user-cancel")
	canceled, err := fixture.svc.CancelAuthRequest(context.Background(), app.CancelAuthRequestInput{
		RequestID:      canceledRequest.ID,
		ResolvedBy:     "operator-2",
		ResolvedType:   domain.ActorTypeUser,
		ResolutionNote: "superseded by another request",
	})
	if err != nil {
		t.Fatalf("CancelAuthRequest() error = %v", err)
	}
	if got := canceled.State; got != domain.AuthRequestStateCanceled {
		t.Fatalf("CancelAuthRequest() state = %q, want canceled", got)
	}

	for _, requestID := range []string{deniedRequest.ID, canceledRequest.ID} {
		item, err := fixture.repo.GetAttentionItem(context.Background(), requestID)
		if err != nil {
			t.Fatalf("GetAttentionItem(%q) error = %v", requestID, err)
		}
		if got := item.State; got != domain.AttentionStateResolved {
			t.Fatalf("attention %q state = %q, want resolved", requestID, got)
		}
	}
}

// TestServiceGetAuthRequestMaterializesTimeoutAndResolvesAttention verifies timed-out requests become expired and clear mirrored notifications on read.
func TestServiceGetAuthRequestMaterializesTimeoutAndResolvesAttention(t *testing.T) {
	fixture := newAuthRequestServiceFixture(t)

	request, err := fixture.svc.CreateAuthRequest(context.Background(), app.CreateAuthRequestInput{
		Path:          "project/" + fixture.project.ID,
		PrincipalID:   "review-user",
		PrincipalType: "user",
		ClientID:      "till-tui",
		ClientType:    "tui",
		Reason:        "brief review",
		Timeout:       time.Millisecond,
	})
	if err != nil {
		t.Fatalf("CreateAuthRequest() error = %v", err)
	}

	time.Sleep(10 * time.Millisecond)

	expired, err := fixture.svc.GetAuthRequest(context.Background(), request.ID)
	if err != nil {
		t.Fatalf("GetAuthRequest() error = %v", err)
	}
	if got := expired.State; got != domain.AuthRequestStateExpired {
		t.Fatalf("GetAuthRequest() state = %q, want expired", got)
	}
	if got := expired.ResolutionNote; got != "timed_out" {
		t.Fatalf("GetAuthRequest() resolution_note = %q, want timed_out", got)
	}

	item, err := fixture.repo.GetAttentionItem(context.Background(), request.ID)
	if err != nil {
		t.Fatalf("GetAttentionItem() error = %v", err)
	}
	if got := item.State; got != domain.AttentionStateResolved {
		t.Fatalf("attention state = %q, want resolved after timeout materialization", got)
	}
}

// TestServiceCreateAuthRequestStewardOrchestratorAccepted verifies the Drop 3
// droplet 3.22 + 3.19 boundary rule: an auth request with
// principal_type="steward" + principal_role="orchestrator" lands as
// pending without a role-validation rejection, because STEWARD is itself a
// persistent orchestrator (per fix L7 / domain auth_request normalization).
func TestServiceCreateAuthRequestStewardOrchestratorAccepted(t *testing.T) {
	fixture := newAuthRequestServiceFixture(t)

	request, err := fixture.svc.CreateAuthRequest(context.Background(), app.CreateAuthRequestInput{
		Path:                "project/" + fixture.project.ID,
		PrincipalID:         "STEWARD",
		PrincipalType:       "steward",
		PrincipalRole:       "orchestrator",
		PrincipalName:       "STEWARD orch",
		ClientID:            "till-mcp-stdio",
		ClientType:          "mcp-stdio",
		ClientName:          "Till MCP STDIO",
		RequestedSessionTTL: 2 * time.Hour,
		Reason:              "STEWARD post-merge MD collation",
		RequestedBy:         "lane-user",
		RequestedType:       domain.ActorTypeUser,
		Timeout:             10 * time.Minute,
	})
	if err != nil {
		t.Fatalf("CreateAuthRequest(steward + orchestrator) error = %v, want nil (pending)", err)
	}
	if got := request.State; got != domain.AuthRequestStatePending {
		t.Fatalf("CreateAuthRequest() state = %q, want pending", got)
	}
	if got := request.PrincipalType; got != "steward" {
		t.Fatalf("CreateAuthRequest() principal_type = %q, want steward", got)
	}
	if got := request.PrincipalRole; got != string(domain.AuthRequestRoleOrchestrator) {
		t.Fatalf("CreateAuthRequest() principal_role = %q, want orchestrator", got)
	}
}

// TestServiceCreateAuthRequestStewardBuilderRejected verifies the Drop 3
// droplet 3.22 + 3.19 boundary rule: an auth request with
// principal_type="steward" + principal_role="builder" REJECTS at the
// domain validation layer with ErrInvalidAuthRequestRole, because STEWARD
// is only ever an orchestrator (no other role makes sense for a persistent
// MD-collation principal).
func TestServiceCreateAuthRequestStewardBuilderRejected(t *testing.T) {
	fixture := newAuthRequestServiceFixture(t)

	if _, err := fixture.svc.CreateAuthRequest(context.Background(), app.CreateAuthRequestInput{
		Path:                "project/" + fixture.project.ID,
		PrincipalID:         "STEWARD",
		PrincipalType:       "steward",
		PrincipalRole:       "builder",
		PrincipalName:       "STEWARD orch",
		ClientID:            "till-mcp-stdio",
		ClientType:          "mcp-stdio",
		ClientName:          "Till MCP STDIO",
		RequestedSessionTTL: 2 * time.Hour,
		Reason:              "invalid steward+builder combination",
		RequestedBy:         "lane-user",
		RequestedType:       domain.ActorTypeUser,
		Timeout:             10 * time.Minute,
	}); !errors.Is(err, domain.ErrInvalidAuthRequestRole) {
		t.Fatalf("CreateAuthRequest(steward + builder) error = %v, want ErrInvalidAuthRequestRole", err)
	}
}

// orchSelfApprovalFixture extends the auth-request fixture with one issued
// orchestrator session that the orch-self-approval gate (Drop 4a Wave 3
// W3.1) can validate against. Created here rather than at top-level because
// the existing fixture is the dev-TUI baseline and most tests do not need
// an orch session.
type orchSelfApprovalFixture struct {
	authRequestServiceFixture
	orchPrincipalID string
	orchSessionID   string
	orchSessionPath string
	orchClientID    string
	orchInstanceID  string
	orchLeaseToken  string
	subagentRequest domain.AuthRequest
	subagentResume  string
}

// newOrchSelfApprovalFixture builds the standard arrangement: one project,
// one issued orchestrator session scoped to project/<id>, one pending
// subagent (builder) auth-request. Tests exercise the gate by supplying or
// omitting the orchestrator's approver-identity fields.
func newOrchSelfApprovalFixture(t *testing.T) orchSelfApprovalFixture {
	t.Helper()
	base := newAuthRequestServiceFixture(t)

	// Issue an orchestrator-roled auth session for the gate to validate.
	// Stamp principal_role + approved_path via session Metadata so
	// mapSessionView surfaces them to the gate's ListAuthSessions query.
	orchPrincipalID := "ORCH_INSTANCE_42"
	orchClientID := "till-mcp-stdio-orch"
	approvedPath := "project/" + base.project.ID
	orchIssued, err := base.auth.IssueSession(context.Background(), autentauth.IssueSessionInput{
		PrincipalID:   orchPrincipalID,
		PrincipalType: "agent",
		PrincipalName: "Drop Orchestrator",
		ClientID:      orchClientID,
		ClientType:    "mcp-stdio",
		ClientName:    "Drop Orch MCP",
		TTL:           2 * time.Hour,
		Metadata: map[string]string{
			"principal_role": "orchestrator",
			"approved_path":  approvedPath,
			"project_id":     base.project.ID,
		},
	})
	if err != nil {
		t.Fatalf("IssueSession(orch) error = %v", err)
	}

	// Create the subagent (builder) auth-request that the orchestrator
	// will attempt to approve.
	subagentRequest, err := base.svc.CreateAuthRequest(context.Background(), app.CreateAuthRequestInput{
		Path:                approvedPath,
		PrincipalID:         "BUILDER_AGENT_99",
		PrincipalType:       "agent",
		PrincipalRole:       "builder",
		PrincipalName:       "Builder Subagent",
		ClientID:            "till-mcp-stdio-builder",
		ClientType:          "mcp-stdio",
		ClientName:          "Builder MCP",
		RequestedSessionTTL: 2 * time.Hour,
		Reason:              "drop-orch self-approval cascade",
		Continuation:        map[string]any{"resume_token": "resume-orch-approve"},
		RequestedBy:         orchPrincipalID,
		RequestedType:       domain.ActorTypeAgent,
		RequesterClientID:   orchClientID,
		Timeout:             10 * time.Minute,
	})
	if err != nil {
		t.Fatalf("CreateAuthRequest(subagent) error = %v", err)
	}

	return orchSelfApprovalFixture{
		authRequestServiceFixture: base,
		orchPrincipalID:           orchPrincipalID,
		orchSessionID:             orchIssued.Session.ID,
		orchSessionPath:           approvedPath,
		orchClientID:              orchClientID,
		orchInstanceID:            "AGENT_INSTANCE_42",
		orchLeaseToken:            "LEASE_TOKEN_42",
		subagentRequest:           subagentRequest,
		subagentResume:            "resume-orch-approve",
	}
}

// TestApproveAuthRequestRejectsMissingApproverIdentity verifies the all-or-nothing
// rule on the four approver-identity fields landed in Drop 4a Wave 3 W3.1.
func TestApproveAuthRequestRejectsMissingApproverIdentity(t *testing.T) {
	t.Parallel()
	fixture := newOrchSelfApprovalFixture(t)
	cases := []struct {
		name string
		in   app.ApproveAuthRequestInput
	}{
		{
			name: "principal_id_empty",
			in: app.ApproveAuthRequestInput{
				ApproverPrincipalID:     "",
				ApproverAgentInstanceID: fixture.orchInstanceID,
				ApproverLeaseToken:      fixture.orchLeaseToken,
				ApproverSessionID:       fixture.orchSessionID,
			},
		},
		{
			name: "agent_instance_id_empty",
			in: app.ApproveAuthRequestInput{
				ApproverPrincipalID:     fixture.orchPrincipalID,
				ApproverAgentInstanceID: "",
				ApproverLeaseToken:      fixture.orchLeaseToken,
				ApproverSessionID:       fixture.orchSessionID,
			},
		},
		{
			name: "lease_token_empty",
			in: app.ApproveAuthRequestInput{
				ApproverPrincipalID:     fixture.orchPrincipalID,
				ApproverAgentInstanceID: fixture.orchInstanceID,
				ApproverLeaseToken:      "",
				ApproverSessionID:       fixture.orchSessionID,
			},
		},
		{
			name: "session_id_empty",
			in: app.ApproveAuthRequestInput{
				ApproverPrincipalID:     fixture.orchPrincipalID,
				ApproverAgentInstanceID: fixture.orchInstanceID,
				ApproverLeaseToken:      fixture.orchLeaseToken,
				ApproverSessionID:       "",
			},
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			in := tc.in
			in.RequestID = fixture.subagentRequest.ID
			in.ResolvedBy = "approver"
			in.ResolvedType = domain.ActorTypeAgent
			_, err := fixture.svc.ApproveAuthRequest(context.Background(), in)
			if !errors.Is(err, domain.ErrInvalidID) {
				t.Fatalf("ApproveAuthRequest() error = %v, want ErrInvalidID-wrapped", err)
			}
		})
	}
}

// TestApproveAuthRequestRejectsNonOrchestratorApprover verifies the gate
// rejects approvers whose session role is not "orchestrator".
func TestApproveAuthRequestRejectsNonOrchestratorApprover(t *testing.T) {
	t.Parallel()
	fixture := newOrchSelfApprovalFixture(t)
	// Issue a separate builder-roled session for the non-orchestrator
	// approver scenario; the autent service does not expose an in-place
	// role swap, so spin up a fresh session with the wrong role.
	builderIssued, err := fixture.auth.IssueSession(context.Background(), autentauth.IssueSessionInput{
		PrincipalID:   "BUILDER_APPROVER_55",
		PrincipalType: "agent",
		PrincipalName: "Builder Mistakenly Approving",
		ClientID:      "till-mcp-stdio-bldr-approver",
		ClientType:    "mcp-stdio",
		ClientName:    "Builder Approver MCP",
		TTL:           time.Hour,
		Metadata: map[string]string{
			"principal_role": "builder",
			"approved_path":  fixture.orchSessionPath,
			"project_id":     fixture.project.ID,
		},
	})
	if err != nil {
		t.Fatalf("IssueSession(builder approver) error = %v", err)
	}
	_, err = fixture.svc.ApproveAuthRequest(context.Background(), app.ApproveAuthRequestInput{
		RequestID:               fixture.subagentRequest.ID,
		ResolvedBy:              "approver",
		ResolvedType:            domain.ActorTypeAgent,
		ApproverPrincipalID:     "BUILDER_APPROVER_55",
		ApproverAgentInstanceID: fixture.orchInstanceID,
		ApproverLeaseToken:      fixture.orchLeaseToken,
		ApproverSessionID:       builderIssued.Session.ID,
	})
	if !errors.Is(err, domain.ErrAuthorizationDenied) {
		t.Fatalf("ApproveAuthRequest(non-orch approver) error = %v, want ErrAuthorizationDenied", err)
	}
}

// TestApproveAuthRequestRejectsOrchSelfApproval verifies the gate rejects
// approval when the approver and request principal_id match (orch cannot
// approve its own request).
func TestApproveAuthRequestRejectsOrchSelfApproval(t *testing.T) {
	t.Parallel()
	fixture := newOrchSelfApprovalFixture(t)
	// Re-create the request with the same principal_id as the approver.
	selfReq, err := fixture.svc.CreateAuthRequest(context.Background(), app.CreateAuthRequestInput{
		Path:                fixture.orchSessionPath,
		PrincipalID:         fixture.orchPrincipalID, // SAME as approver
		PrincipalType:       "agent",
		PrincipalRole:       "builder",
		ClientID:            fixture.orchClientID,
		ClientType:          "mcp-stdio",
		RequestedSessionTTL: 2 * time.Hour,
		Reason:              "orch attempts self-approval",
		Continuation:        map[string]any{"resume_token": "resume-self"},
		RequestedBy:         fixture.orchPrincipalID,
		RequestedType:       domain.ActorTypeAgent,
		RequesterClientID:   fixture.orchClientID,
		Timeout:             10 * time.Minute,
	})
	if err != nil {
		t.Fatalf("CreateAuthRequest(self) error = %v", err)
	}
	_, err = fixture.svc.ApproveAuthRequest(context.Background(), app.ApproveAuthRequestInput{
		RequestID:               selfReq.ID,
		ResolvedBy:              "approver",
		ResolvedType:            domain.ActorTypeAgent,
		ApproverPrincipalID:     fixture.orchPrincipalID,
		ApproverAgentInstanceID: fixture.orchInstanceID,
		ApproverLeaseToken:      fixture.orchLeaseToken,
		ApproverSessionID:       fixture.orchSessionID,
	})
	if !errors.Is(err, domain.ErrAuthorizationDenied) {
		t.Fatalf("ApproveAuthRequest(self) error = %v, want ErrAuthorizationDenied", err)
	}
}

// TestApproveAuthRequestRejectsApproveOfOrchestratorRequest verifies the gate
// rejects when the request itself carries principal_role=orchestrator —
// orch-on-orch approvals stay dev-only.
func TestApproveAuthRequestRejectsApproveOfOrchestratorRequest(t *testing.T) {
	t.Parallel()
	fixture := newOrchSelfApprovalFixture(t)
	orchOnOrchReq, err := fixture.svc.CreateAuthRequest(context.Background(), app.CreateAuthRequestInput{
		Path:                fixture.orchSessionPath,
		PrincipalID:         "OTHER_ORCH_77",
		PrincipalType:       "agent",
		PrincipalRole:       "orchestrator",
		ClientID:            "till-mcp-stdio-other",
		ClientType:          "mcp-stdio",
		RequestedSessionTTL: 2 * time.Hour,
		Reason:              "orch-on-orch attempt",
		Continuation:        map[string]any{"resume_token": "resume-orch-on-orch"},
		RequestedBy:         "lane-user",
		RequestedType:       domain.ActorTypeUser,
		Timeout:             10 * time.Minute,
	})
	if err != nil {
		t.Fatalf("CreateAuthRequest(orch-target) error = %v", err)
	}
	_, err = fixture.svc.ApproveAuthRequest(context.Background(), app.ApproveAuthRequestInput{
		RequestID:               orchOnOrchReq.ID,
		ResolvedBy:              "approver",
		ResolvedType:            domain.ActorTypeAgent,
		ApproverPrincipalID:     fixture.orchPrincipalID,
		ApproverAgentInstanceID: fixture.orchInstanceID,
		ApproverLeaseToken:      fixture.orchLeaseToken,
		ApproverSessionID:       fixture.orchSessionID,
	})
	if !errors.Is(err, domain.ErrAuthorizationDenied) {
		t.Fatalf("ApproveAuthRequest(orch-target) error = %v, want ErrAuthorizationDenied", err)
	}
}

// TestApproveAuthRequestRejectsOutOfSubtree verifies the gate rejects when
// the approver's path does not encompass the request's path. A different
// project_id is the simplest out-of-subtree case.
func TestApproveAuthRequestRejectsOutOfSubtree(t *testing.T) {
	t.Parallel()
	fixture := newOrchSelfApprovalFixture(t)

	// Create a second project and a request scoped to it.
	otherProject, err := domain.NewProjectFromInput(domain.ProjectInput{ID: "p2", Name: "Project Two"}, time.Now())
	if err != nil {
		t.Fatalf("NewProjectFromInput(p2) error = %v", err)
	}
	if err := fixture.repo.CreateProject(context.Background(), otherProject); err != nil {
		t.Fatalf("CreateProject(p2) error = %v", err)
	}
	otherReq, err := fixture.svc.CreateAuthRequest(context.Background(), app.CreateAuthRequestInput{
		Path:                "project/p2",
		PrincipalID:         "BUILDER_FOR_P2",
		PrincipalType:       "agent",
		PrincipalRole:       "builder",
		ClientID:            "till-mcp-stdio-p2",
		ClientType:          "mcp-stdio",
		RequestedSessionTTL: 2 * time.Hour,
		Reason:              "out-of-subtree attempt",
		Continuation:        map[string]any{"resume_token": "resume-p2"},
		RequestedBy:         fixture.orchPrincipalID,
		RequestedType:       domain.ActorTypeAgent,
		RequesterClientID:   fixture.orchClientID,
		Timeout:             10 * time.Minute,
	})
	if err != nil {
		t.Fatalf("CreateAuthRequest(p2) error = %v", err)
	}

	// Approver session is scoped to p1; request is scoped to p2 → reject.
	_, err = fixture.svc.ApproveAuthRequest(context.Background(), app.ApproveAuthRequestInput{
		RequestID:               otherReq.ID,
		ResolvedBy:              "approver",
		ResolvedType:            domain.ActorTypeAgent,
		ApproverPrincipalID:     fixture.orchPrincipalID,
		ApproverAgentInstanceID: fixture.orchInstanceID,
		ApproverLeaseToken:      fixture.orchLeaseToken,
		ApproverSessionID:       fixture.orchSessionID,
	})
	if !errors.Is(err, domain.ErrAuthorizationDenied) {
		t.Fatalf("ApproveAuthRequest(out-of-subtree) error = %v, want ErrAuthorizationDenied", err)
	}
}

// TestApproveAuthRequestSucceedsForSameOrchSubagentInScope verifies the
// happy-path: approver is orch, request is its own delegated subagent
// (RequestedByActor == approver), within scope, non-orchestrator role —
// approval succeeds and a session secret issues.
func TestApproveAuthRequestSucceedsForSameOrchSubagentInScope(t *testing.T) {
	t.Parallel()
	fixture := newOrchSelfApprovalFixture(t)
	approved, err := fixture.svc.ApproveAuthRequest(context.Background(), app.ApproveAuthRequestInput{
		RequestID:               fixture.subagentRequest.ID,
		ResolvedBy:              fixture.orchPrincipalID,
		ResolvedType:            domain.ActorTypeAgent,
		ResolutionNote:          "orch self-approval cascade",
		ApproverPrincipalID:     fixture.orchPrincipalID,
		ApproverAgentInstanceID: fixture.orchInstanceID,
		ApproverLeaseToken:      fixture.orchLeaseToken,
		ApproverSessionID:       fixture.orchSessionID,
	})
	if err != nil {
		t.Fatalf("ApproveAuthRequest(happy) error = %v", err)
	}
	if approved.Request.State != domain.AuthRequestStateApproved {
		t.Fatalf("ApproveAuthRequest(happy) state = %q, want approved", approved.Request.State)
	}
	if approved.SessionSecret == "" {
		t.Fatal("ApproveAuthRequest(happy) returned empty session secret")
	}
}

// TestApproveAuthRequestRejectsCrossOrchWithoutSteward verifies the gate
// rejects when the request was created by a different orchestrator AND the
// approver is not a STEWARD session — falsification mitigation 1+5.
func TestApproveAuthRequestRejectsCrossOrchWithoutSteward(t *testing.T) {
	t.Parallel()
	fixture := newOrchSelfApprovalFixture(t)
	// Re-create the request with a DIFFERENT requested_by_actor.
	otherOrchReq, err := fixture.svc.CreateAuthRequest(context.Background(), app.CreateAuthRequestInput{
		Path:                fixture.orchSessionPath,
		PrincipalID:         "BUILDER_FOR_OTHER",
		PrincipalType:       "agent",
		PrincipalRole:       "builder",
		ClientID:            "till-mcp-stdio-builder-2",
		ClientType:          "mcp-stdio",
		RequestedSessionTTL: 2 * time.Hour,
		Reason:              "cross-orch attempt",
		Continuation:        map[string]any{"resume_token": "resume-cross"},
		RequestedBy:         "OTHER_ORCH_55", // DIFFERENT orch
		RequestedType:       domain.ActorTypeAgent,
		RequesterClientID:   "till-mcp-stdio-other",
		Timeout:             10 * time.Minute,
	})
	if err != nil {
		t.Fatalf("CreateAuthRequest(cross-orch) error = %v", err)
	}

	_, err = fixture.svc.ApproveAuthRequest(context.Background(), app.ApproveAuthRequestInput{
		RequestID:               otherOrchReq.ID,
		ResolvedBy:              fixture.orchPrincipalID,
		ResolvedType:            domain.ActorTypeAgent,
		ApproverPrincipalID:     fixture.orchPrincipalID,
		ApproverAgentInstanceID: fixture.orchInstanceID,
		ApproverLeaseToken:      fixture.orchLeaseToken,
		ApproverSessionID:       fixture.orchSessionID,
	})
	if !errors.Is(err, domain.ErrAuthorizationDenied) {
		t.Fatalf("ApproveAuthRequest(cross-orch) error = %v, want ErrAuthorizationDenied (non-steward approver)", err)
	}
}

// stewardCrossSubtreeFixture extends the orch-self-approval fixture with
// (a) a STEWARD-typed orch session (autent collapses PrincipalType
// "steward" → "agent" but preserves "steward" via session metadata key
// "auth_request_principal_type"), and (b) one persistent STEWARD-owned
// ancestor action_item under the project so the
// requireStewardPersistentAncestor walk has a target. The cross-orch
// trigger is wired by re-creating the subagent request with a different
// RequestedBy, mirroring TestApproveAuthRequestRejectsCrossOrchWithoutSteward.
//
// Drop 4a droplet 4a.24 fix verification: with the gate now reading
// AuthRequestPrincipalType (not PrincipalType), the STEWARD cross-subtree
// exception actually fires for the first time. Pre-fix every approver was
// rejected on the type check because PrincipalType was always "agent".
type stewardCrossSubtreeFixture struct {
	orchSelfApprovalFixture
	stewardSessionID    string
	stewardPrincipalID  string
	persistentAncestor  string
	subagentRequestPath string
}

// newStewardCrossSubtreeFixture issues a STEWARD-typed orch session,
// creates one column + one persistent STEWARD-owned action_item under the
// project, and re-creates a cross-orch subagent request whose path roots
// under that ancestor. Returns the fixture so the test can drive
// ApproveAuthRequest with whichever approver session it wants to verify.
func newStewardCrossSubtreeFixture(t *testing.T) stewardCrossSubtreeFixture {
	t.Helper()
	base := newOrchSelfApprovalFixture(t)
	ctx := context.Background()

	// Issue a STEWARD-typed orch session. PrincipalType "steward" is the
	// tillsyn-axis value; autent's IssueSession collapses it to "agent" at
	// the autentdomain.PrincipalType layer but preserves "steward" via the
	// "auth_request_principal_type" metadata key, which mapSessionView
	// reads into AuthSession.AuthRequestPrincipalType.
	stewardPrincipalID := "STEWARD"
	stewardClientID := "till-mcp-stdio-steward"
	stewardIssued, err := base.auth.IssueSession(ctx, autentauth.IssueSessionInput{
		PrincipalID:   stewardPrincipalID,
		PrincipalType: "steward",
		PrincipalName: "STEWARD",
		ClientID:      stewardClientID,
		ClientType:    "mcp-stdio",
		ClientName:    "STEWARD MCP",
		TTL:           2 * time.Hour,
		Metadata: map[string]string{
			"principal_role": "orchestrator",
			"approved_path":  "project/" + base.project.ID,
			"project_id":     base.project.ID,
		},
	})
	if err != nil {
		t.Fatalf("IssueSession(steward) error = %v", err)
	}

	// Create a column so the persistent ancestor action_item validates.
	now := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	column, err := domain.NewColumn("col-steward", base.project.ID, "Backlog", 0, 0, now)
	if err != nil {
		t.Fatalf("NewColumn() error = %v", err)
	}
	if err := base.repo.CreateColumn(ctx, column); err != nil {
		t.Fatalf("CreateColumn() error = %v", err)
	}

	// Persistent STEWARD-owned ancestor — the ancestry walk's success target.
	persistentID := "PERSISTENT_STEWARD_PARENT"
	ancestor, err := domain.NewActionItemForTest(domain.ActionItemInput{
		ID:         persistentID,
		ProjectID:  base.project.ID,
		Kind:       domain.KindRefinement,
		ColumnID:   column.ID,
		Title:      "STEWARD Persistent Refinement",
		Owner:      "STEWARD",
		Persistent: true,
	}, now)
	if err != nil {
		t.Fatalf("NewActionItemForTest(persistent) error = %v", err)
	}
	if err := base.repo.CreateActionItem(ctx, ancestor); err != nil {
		t.Fatalf("CreateActionItem(persistent) error = %v", err)
	}

	// Path rooted under the persistent ancestor — branch path so
	// path.ScopeID == persistentID and the walk hits Persistent+STEWARD on
	// the first iteration.
	subagentRequestPath := "project/" + base.project.ID + "/branch/" + persistentID

	return stewardCrossSubtreeFixture{
		orchSelfApprovalFixture: base,
		stewardSessionID:        stewardIssued.Session.ID,
		stewardPrincipalID:      stewardPrincipalID,
		persistentAncestor:      persistentID,
		subagentRequestPath:     subagentRequestPath,
	}
}

// TestApproveAuthRequestStewardCrossSubtreeExceptionFires verifies the
// STEWARD cross-subtree exception path approves when (a) approver is a
// STEWARD-typed orch session, (b) the request was created by a different
// orchestrator, and (c) the request path roots under a Persistent +
// Owner=="STEWARD" action_item. This positive test would have failed
// pre-4a.24-fix because the gate read session.PrincipalType (always
// "agent" after autent collapse) and rejected every steward approver —
// the exception was non-functional.
func TestApproveAuthRequestStewardCrossSubtreeExceptionFires(t *testing.T) {
	t.Parallel()
	fixture := newStewardCrossSubtreeFixture(t)
	ctx := context.Background()

	// Cross-orch subagent request rooted under the persistent ancestor.
	// Different RequestedBy than the steward principal triggers the
	// cross-orch branch in checkOrchSelfApprovalGate.
	crossOrchReq, err := fixture.svc.CreateAuthRequest(ctx, app.CreateAuthRequestInput{
		Path:                fixture.subagentRequestPath,
		PrincipalID:         "BUILDER_FROM_OTHER_ORCH",
		PrincipalType:       "agent",
		PrincipalRole:       "builder",
		ClientID:            "till-mcp-stdio-builder-other",
		ClientType:          "mcp-stdio",
		RequestedSessionTTL: 2 * time.Hour,
		Reason:              "steward cross-subtree approval test",
		Continuation:        map[string]any{"resume_token": "resume-steward-cross"},
		RequestedBy:         "OTHER_ORCH_77",
		RequestedType:       domain.ActorTypeAgent,
		RequesterClientID:   "till-mcp-stdio-other-orch",
		Timeout:             10 * time.Minute,
	})
	if err != nil {
		t.Fatalf("CreateAuthRequest(cross-orch under steward ancestor) error = %v", err)
	}

	approved, err := fixture.svc.ApproveAuthRequest(ctx, app.ApproveAuthRequestInput{
		RequestID:               crossOrchReq.ID,
		ResolvedBy:              fixture.stewardPrincipalID,
		ResolvedType:            domain.ActorTypeAgent,
		ResolutionNote:          "steward cross-subtree exception",
		ApproverPrincipalID:     fixture.stewardPrincipalID,
		ApproverAgentInstanceID: "STEWARD_INSTANCE",
		ApproverLeaseToken:      "STEWARD_LEASE",
		ApproverSessionID:       fixture.stewardSessionID,
	})
	if err != nil {
		t.Fatalf("ApproveAuthRequest(steward cross-subtree) error = %v, want success", err)
	}
	if approved.Request.State != domain.AuthRequestStateApproved {
		t.Fatalf("ApproveAuthRequest(steward cross-subtree) state = %q, want approved", approved.Request.State)
	}
	if approved.SessionSecret == "" {
		t.Fatal("ApproveAuthRequest(steward cross-subtree) returned empty session secret")
	}
}

// TestApproveAuthRequestStewardCrossSubtreeExceptionRejectsNonStewardOrch
// verifies the negative twin: the same persistent STEWARD-owned ancestor
// exists, the request roots under it, but the approver is a NON-steward
// orch (PrincipalType "agent" + AuthRequestPrincipalType "agent"). The
// gate must reject because the cross-orch branch requires the approver's
// AuthRequestPrincipalType to be "steward". This guards the
// post-4a.24-fix gate against the trivial bug-where-everyone-passes
// regression — the rejection must remain when the approver is not a
// steward, even when the persistent ancestor exists.
func TestApproveAuthRequestStewardCrossSubtreeExceptionRejectsNonStewardOrch(t *testing.T) {
	t.Parallel()
	fixture := newStewardCrossSubtreeFixture(t)
	ctx := context.Background()

	// Cross-orch subagent request rooted under the persistent ancestor.
	crossOrchReq, err := fixture.svc.CreateAuthRequest(ctx, app.CreateAuthRequestInput{
		Path:                fixture.subagentRequestPath,
		PrincipalID:         "BUILDER_FROM_OTHER_ORCH",
		PrincipalType:       "agent",
		PrincipalRole:       "builder",
		ClientID:            "till-mcp-stdio-builder-other-2",
		ClientType:          "mcp-stdio",
		RequestedSessionTTL: 2 * time.Hour,
		Reason:              "non-steward orch attempting cross-subtree approval",
		Continuation:        map[string]any{"resume_token": "resume-non-steward-cross"},
		RequestedBy:         "OTHER_ORCH_88",
		RequestedType:       domain.ActorTypeAgent,
		RequesterClientID:   "till-mcp-stdio-other-orch-2",
		Timeout:             10 * time.Minute,
	})
	if err != nil {
		t.Fatalf("CreateAuthRequest(cross-orch under steward ancestor) error = %v", err)
	}

	// Use the NON-steward orch session from the embedded base fixture.
	// Its PrincipalType is "agent" and AuthRequestPrincipalType is "agent"
	// (no steward collapse), so the gate's
	// AuthRequestPrincipalType == "steward" check must fail.
	_, err = fixture.svc.ApproveAuthRequest(ctx, app.ApproveAuthRequestInput{
		RequestID:               crossOrchReq.ID,
		ResolvedBy:              fixture.orchPrincipalID,
		ResolvedType:            domain.ActorTypeAgent,
		ApproverPrincipalID:     fixture.orchPrincipalID,
		ApproverAgentInstanceID: fixture.orchInstanceID,
		ApproverLeaseToken:      fixture.orchLeaseToken,
		ApproverSessionID:       fixture.orchSessionID,
	})
	if !errors.Is(err, domain.ErrAuthorizationDenied) {
		t.Fatalf("ApproveAuthRequest(non-steward orch cross-subtree) error = %v, want ErrAuthorizationDenied", err)
	}
}

// TestApproveAuthRequestDevTUIPathBypassesGate verifies the legacy
// dev-TUI / system approval path (all four approver-identity fields empty)
// still works post-W3.1 — the gate is bypassed.
func TestApproveAuthRequestDevTUIPathBypassesGate(t *testing.T) {
	t.Parallel()
	fixture := newAuthRequestServiceFixture(t)
	request, err := fixture.svc.CreateAuthRequest(context.Background(), app.CreateAuthRequestInput{
		Path:                "project/" + fixture.project.ID,
		PrincipalID:         "review-agent",
		PrincipalType:       "agent",
		ClientID:            "till-mcp-stdio",
		ClientType:          "mcp-stdio",
		RequestedSessionTTL: 2 * time.Hour,
		Reason:              "dev-TUI approval",
		RequestedBy:         "lane-user",
		RequestedType:       domain.ActorTypeUser,
		Timeout:             10 * time.Minute,
	})
	if err != nil {
		t.Fatalf("CreateAuthRequest() error = %v", err)
	}
	approved, err := fixture.svc.ApproveAuthRequest(context.Background(), app.ApproveAuthRequestInput{
		RequestID:      request.ID,
		ResolvedBy:     "approver-1",
		ResolvedType:   domain.ActorTypeUser,
		ResolutionNote: "approved by dev",
		// All four ApproverPrincipalID / ApproverAgentInstanceID /
		// ApproverLeaseToken / ApproverSessionID empty → dev-TUI path,
		// gate bypassed.
	})
	if err != nil {
		t.Fatalf("ApproveAuthRequest(dev-TUI) error = %v", err)
	}
	if approved.Request.State != domain.AuthRequestStateApproved {
		t.Fatalf("ApproveAuthRequest(dev-TUI) state = %q, want approved", approved.Request.State)
	}
}

// setProjectOrchSelfApprovalEnabled writes the W3.2 toggle on a project row
// directly via the repository. Test-only helper — production callers go
// through the till.project update operation.
func setProjectOrchSelfApprovalEnabled(t *testing.T, fixture authRequestServiceFixture, enabled bool) {
	t.Helper()
	ctx := context.Background()
	project, err := fixture.repo.GetProject(ctx, fixture.project.ID)
	if err != nil {
		t.Fatalf("GetProject(toggle setup) error = %v", err)
	}
	val := enabled
	project.Metadata.OrchSelfApprovalEnabled = &val
	if err := fixture.repo.UpdateProject(ctx, project); err != nil {
		t.Fatalf("UpdateProject(toggle setup) error = %v", err)
	}
}

// TestApproveAuthRequestRejectsWhenProjectToggleDisabled verifies the Drop
// 4a Wave 3 W3.2 project-metadata opt-out toggle rejects the orch-self-
// approval cascade BEFORE the role / path / cross-orch gate runs. The
// rejection is total — even the STEWARD cross-subtree path is backstopped.
//
// Two sub-cases share the same toggle-disabled state but exercise different
// approver shapes:
//
//   - non_steward_orch — same-orch self-approval (the W3.1 happy path)
//     rejects with ErrOrchSelfApprovalDisabled.
//   - steward_cross_subtree — STEWARD approver under a Persistent +
//     Owner=="STEWARD" ancestor (the W3.1 cross-subtree exception path)
//     ALSO rejects, proving the toggle is a total backstop.
func TestApproveAuthRequestRejectsWhenProjectToggleDisabled(t *testing.T) {
	t.Run("non_steward_orch", func(t *testing.T) {
		fixture := newOrchSelfApprovalFixture(t)
		setProjectOrchSelfApprovalEnabled(t, fixture.authRequestServiceFixture, false)

		_, err := fixture.svc.ApproveAuthRequest(context.Background(), app.ApproveAuthRequestInput{
			RequestID:               fixture.subagentRequest.ID,
			ResolvedBy:              fixture.orchPrincipalID,
			ResolvedType:            domain.ActorTypeAgent,
			ResolutionNote:          "toggle-disabled non-steward attempt",
			ApproverPrincipalID:     fixture.orchPrincipalID,
			ApproverAgentInstanceID: fixture.orchInstanceID,
			ApproverLeaseToken:      fixture.orchLeaseToken,
			ApproverSessionID:       fixture.orchSessionID,
		})
		if !errors.Is(err, domain.ErrOrchSelfApprovalDisabled) {
			t.Fatalf("ApproveAuthRequest(toggle-disabled, non-steward) error = %v, want ErrOrchSelfApprovalDisabled", err)
		}
		// Defense: must NOT also surface as ErrAuthorizationDenied (sentinel
		// keeps observability sharp).
		if errors.Is(err, domain.ErrAuthorizationDenied) {
			t.Fatalf("ApproveAuthRequest(toggle-disabled) wrapped ErrAuthorizationDenied; want only ErrOrchSelfApprovalDisabled")
		}
	})

	t.Run("steward_cross_subtree", func(t *testing.T) {
		fixture := newStewardCrossSubtreeFixture(t)
		setProjectOrchSelfApprovalEnabled(t, fixture.authRequestServiceFixture, false)
		ctx := context.Background()

		// Re-create a cross-orch subagent request rooted under the
		// persistent STEWARD-owned ancestor (mirrors the cross-subtree
		// happy-path test). With the toggle disabled this MUST still
		// reject — total backstop.
		crossOrchReq, err := fixture.svc.CreateAuthRequest(ctx, app.CreateAuthRequestInput{
			Path:                fixture.subagentRequestPath,
			PrincipalID:         "BUILDER_FROM_OTHER_ORCH",
			PrincipalType:       "agent",
			PrincipalRole:       "builder",
			ClientID:            "till-mcp-stdio-builder-other-toggle",
			ClientType:          "mcp-stdio",
			RequestedSessionTTL: 2 * time.Hour,
			Reason:              "toggle-disabled steward cross-subtree attempt",
			Continuation:        map[string]any{"resume_token": "resume-toggle-cross"},
			RequestedBy:         "OTHER_ORCH_77",
			RequestedType:       domain.ActorTypeAgent,
			RequesterClientID:   "till-mcp-stdio-other-orch-toggle",
			Timeout:             10 * time.Minute,
		})
		if err != nil {
			t.Fatalf("CreateAuthRequest(cross-orch under steward ancestor) error = %v", err)
		}

		_, err = fixture.svc.ApproveAuthRequest(ctx, app.ApproveAuthRequestInput{
			RequestID:               crossOrchReq.ID,
			ResolvedBy:              fixture.stewardPrincipalID,
			ResolvedType:            domain.ActorTypeAgent,
			ResolutionNote:          "toggle-disabled steward backstop",
			ApproverPrincipalID:     fixture.stewardPrincipalID,
			ApproverAgentInstanceID: "STEWARD_INSTANCE",
			ApproverLeaseToken:      "STEWARD_LEASE",
			ApproverSessionID:       fixture.stewardSessionID,
		})
		if !errors.Is(err, domain.ErrOrchSelfApprovalDisabled) {
			t.Fatalf("ApproveAuthRequest(toggle-disabled, steward) error = %v, want ErrOrchSelfApprovalDisabled", err)
		}
	})
}

// TestApproveAuthRequestAllowedWhenProjectToggleNil verifies that a project
// with no explicit toggle (nil OrchSelfApprovalEnabled) still grants the
// orch-self-approval cascade — the W3.1 behavior must be preserved as the
// default. Drop 4a Wave 3 W3.2.
func TestApproveAuthRequestAllowedWhenProjectToggleNil(t *testing.T) {
	t.Parallel()
	fixture := newOrchSelfApprovalFixture(t)

	// Sanity: fixture's freshly-created project has nil toggle by
	// construction (NewProjectFromInput leaves Metadata zero-valued).
	project, err := fixture.repo.GetProject(context.Background(), fixture.project.ID)
	if err != nil {
		t.Fatalf("GetProject() error = %v", err)
	}
	if project.Metadata.OrchSelfApprovalEnabled != nil {
		t.Fatalf("freshly-created project has non-nil toggle pointer; want nil for default-enabled")
	}
	if !project.Metadata.OrchSelfApprovalIsEnabled() {
		t.Fatalf("default ProjectMetadata.OrchSelfApprovalIsEnabled() = false; want true")
	}

	approved, err := fixture.svc.ApproveAuthRequest(context.Background(), app.ApproveAuthRequestInput{
		RequestID:               fixture.subagentRequest.ID,
		ResolvedBy:              fixture.orchPrincipalID,
		ResolvedType:            domain.ActorTypeAgent,
		ResolutionNote:          "default-enabled toggle still permits cascade",
		ApproverPrincipalID:     fixture.orchPrincipalID,
		ApproverAgentInstanceID: fixture.orchInstanceID,
		ApproverLeaseToken:      fixture.orchLeaseToken,
		ApproverSessionID:       fixture.orchSessionID,
	})
	if err != nil {
		t.Fatalf("ApproveAuthRequest(toggle-nil) error = %v, want success", err)
	}
	if approved.Request.State != domain.AuthRequestStateApproved {
		t.Fatalf("ApproveAuthRequest(toggle-nil) state = %q, want approved", approved.Request.State)
	}
	if approved.SessionSecret == "" {
		t.Fatal("ApproveAuthRequest(toggle-nil) returned empty session secret")
	}
}
