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

	project, err := domain.NewProject("p1", "Project One", "", now)
	if err != nil {
		t.Fatalf("NewProject() error = %v", err)
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
