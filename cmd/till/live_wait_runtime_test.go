package main

import (
	"context"
	"database/sql"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/hylla/tillsyn/internal/adapters/auth/autentauth"
	"github.com/hylla/tillsyn/internal/adapters/livewait/localipc"
	"github.com/hylla/tillsyn/internal/adapters/storage/sqlite"
	"github.com/hylla/tillsyn/internal/app"
	"github.com/hylla/tillsyn/internal/domain"
)

// TestRunConstructsLiveWaitBroker verifies the runtime bootstrap path constructs the shared live-wait broker.
func TestRunConstructsLiveWaitBroker(t *testing.T) {
	origProgramFactory := programFactory
	origBrokerFactory := newRuntimeLiveWaitBrokerFunc
	t.Cleanup(func() {
		programFactory = origProgramFactory
		newRuntimeLiveWaitBrokerFunc = origBrokerFactory
	})

	programFactory = func(_ tea.Model) program {
		return fakeProgram{}
	}

	called := false
	newRuntimeLiveWaitBrokerFunc = func(db *sql.DB, rootDir string) (*localipc.Broker, error) {
		called = true
		broker, err := newRuntimeLiveWaitBroker(db, rootDir)
		if err != nil {
			return nil, err
		}
		secretPath := runtimeLiveWaitSecretPath(rootDir)
		if _, err := os.Stat(secretPath); err != nil {
			t.Fatalf("runtime live wait secret path %q error = %v", secretPath, err)
		}
		return broker, nil
	}

	root := t.TempDir()
	dbPath := filepath.Join(root, "tillsyn.db")
	cfgPath := filepath.Join(root, "config.toml")
	writeBootstrapReadyConfig(t, cfgPath, root)

	if err := run(context.Background(), []string{"--db", dbPath, "--config", cfgPath}, io.Discard, io.Discard); err != nil {
		t.Fatalf("run() error = %v", err)
	}
	if !called {
		t.Fatal("expected runtime live-wait broker constructor to be called")
	}
}

// TestCrossProcessAuthWaitWakesOnApproval verifies one service instance wakes when another service instance approves the request.
func TestCrossProcessAuthWaitWakesOnApproval(t *testing.T) {
	fixture := newCrossProcessAuthFixture(t)

	request, err := fixture.waitService.CreateAuthRequest(context.Background(), app.CreateAuthRequestInput{
		Path:                "project/" + fixture.project.ID,
		PrincipalID:         "review-agent",
		PrincipalType:       "agent",
		ClientID:            "till-mcp-stdio",
		ClientType:          "mcp-stdio",
		RequestedSessionTTL: time.Hour,
		Reason:              "cross-process approval wake",
		Continuation:        map[string]any{"resume_token": "resume-token"},
		RequestedBy:         "review-agent",
		RequesterClientID:   "till-mcp-stdio",
	})
	if err != nil {
		t.Fatalf("CreateAuthRequest() error = %v", err)
	}

	claimedCh := make(chan app.ClaimedAuthRequestResult, 1)
	errCh := make(chan error, 1)
	go func() {
		claimed, claimErr := fixture.waitService.ClaimAuthRequest(context.Background(), app.ClaimAuthRequestInput{
			RequestID:   request.ID,
			ResumeToken: "resume-token",
			PrincipalID: "review-agent",
			ClientID:    "till-mcp-stdio",
			WaitTimeout: 5 * time.Second,
		})
		if claimErr != nil {
			errCh <- claimErr
			return
		}
		claimedCh <- claimed
	}()

	waitForLiveWaitSubscription(t, fixture.waitRepo.DB(), string(app.LiveWaitEventAuthRequestResolved), request.ID)
	approved, err := fixture.approveService.ApproveAuthRequest(context.Background(), app.ApproveAuthRequestInput{
		RequestID:      request.ID,
		ResolvedBy:     "approver-1",
		ResolvedType:   domain.ActorTypeUser,
		ResolutionNote: "approved for cross-process wake",
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
	case <-time.After(2 * time.Second):
		t.Fatal("ClaimAuthRequest() did not wake after approval")
	}
}

// TestCrossProcessAuthWaitWakesOnDeny verifies one service instance wakes when another service instance denies the request.
func TestCrossProcessAuthWaitWakesOnDeny(t *testing.T) {
	fixture := newCrossProcessAuthFixture(t)

	request, err := fixture.waitService.CreateAuthRequest(context.Background(), app.CreateAuthRequestInput{
		Path:              "project/" + fixture.project.ID,
		PrincipalID:       "review-agent",
		PrincipalType:     "agent",
		ClientID:          "till-mcp-stdio",
		ClientType:        "mcp-stdio",
		Reason:            "cross-process deny wake",
		Continuation:      map[string]any{"resume_token": "resume-token"},
		RequestedBy:       "review-agent",
		RequesterClientID: "till-mcp-stdio",
	})
	if err != nil {
		t.Fatalf("CreateAuthRequest() error = %v", err)
	}

	claimedCh := make(chan app.ClaimedAuthRequestResult, 1)
	errCh := make(chan error, 1)
	go func() {
		claimed, claimErr := fixture.waitService.ClaimAuthRequest(context.Background(), app.ClaimAuthRequestInput{
			RequestID:   request.ID,
			ResumeToken: "resume-token",
			PrincipalID: "review-agent",
			ClientID:    "till-mcp-stdio",
			WaitTimeout: 5 * time.Second,
		})
		if claimErr != nil {
			errCh <- claimErr
			return
		}
		claimedCh <- claimed
	}()

	waitForLiveWaitSubscription(t, fixture.waitRepo.DB(), string(app.LiveWaitEventAuthRequestResolved), request.ID)
	denied, err := fixture.approveService.DenyAuthRequest(context.Background(), app.DenyAuthRequestInput{
		RequestID:      request.ID,
		ResolvedBy:     "approver-1",
		ResolvedType:   domain.ActorTypeUser,
		ResolutionNote: "denied for cross-process wake",
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
	case <-time.After(2 * time.Second):
		t.Fatal("ClaimAuthRequest() did not wake after denial")
	}
}

// TestCrossProcessAuthWaitWakesOnCancel verifies one service instance wakes when another service instance cancels the request.
func TestCrossProcessAuthWaitWakesOnCancel(t *testing.T) {
	fixture := newCrossProcessAuthFixture(t)

	request, err := fixture.waitService.CreateAuthRequest(context.Background(), app.CreateAuthRequestInput{
		Path:              "project/" + fixture.project.ID,
		PrincipalID:       "review-agent",
		PrincipalType:     "agent",
		ClientID:          "till-mcp-stdio",
		ClientType:        "mcp-stdio",
		Reason:            "cross-process cancel wake",
		Continuation:      map[string]any{"resume_token": "resume-token"},
		RequestedBy:       "review-agent",
		RequesterClientID: "till-mcp-stdio",
	})
	if err != nil {
		t.Fatalf("CreateAuthRequest() error = %v", err)
	}

	claimedCh := make(chan app.ClaimedAuthRequestResult, 1)
	errCh := make(chan error, 1)
	go func() {
		claimed, claimErr := fixture.waitService.ClaimAuthRequest(context.Background(), app.ClaimAuthRequestInput{
			RequestID:   request.ID,
			ResumeToken: "resume-token",
			PrincipalID: "review-agent",
			ClientID:    "till-mcp-stdio",
			WaitTimeout: 5 * time.Second,
		})
		if claimErr != nil {
			errCh <- claimErr
			return
		}
		claimedCh <- claimed
	}()

	waitForLiveWaitSubscription(t, fixture.waitRepo.DB(), string(app.LiveWaitEventAuthRequestResolved), request.ID)
	canceled, err := fixture.approveService.CancelAuthRequest(context.Background(), app.CancelAuthRequestInput{
		RequestID:      request.ID,
		ResolvedBy:     "approver-1",
		ResolvedType:   domain.ActorTypeUser,
		ResolutionNote: "canceled for cross-process wake",
	})
	if err != nil {
		t.Fatalf("CancelAuthRequest() error = %v", err)
	}

	select {
	case err := <-errCh:
		t.Fatalf("ClaimAuthRequest() error = %v", err)
	case claimed := <-claimedCh:
		if got := claimed.Request.State; got != domain.AuthRequestStateCanceled {
			t.Fatalf("ClaimAuthRequest() state = %q, want canceled", got)
		}
		if got := claimed.Request.ResolutionNote; got != canceled.ResolutionNote {
			t.Fatalf("ClaimAuthRequest() resolution_note = %q, want %q", got, canceled.ResolutionNote)
		}
		if got := claimed.SessionSecret; got != "" {
			t.Fatalf("ClaimAuthRequest() session_secret = %q, want empty after cancel", got)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("ClaimAuthRequest() did not wake after cancel")
	}
}

// TestLoadOrCreateRuntimeLiveWaitSecretSurvivesConcurrentBootstrap verifies first-run secret bootstrap converges on one shared secret.
func TestLoadOrCreateRuntimeLiveWaitSecretSurvivesConcurrentBootstrap(t *testing.T) {
	root := t.TempDir()

	const workers = 8
	results := make(chan string, workers)
	errs := make(chan error, workers)
	var start sync.WaitGroup
	start.Add(1)

	for i := 0; i < workers; i++ {
		go func() {
			start.Wait()
			secret, err := loadOrCreateRuntimeLiveWaitSecret(root)
			if err != nil {
				errs <- err
				return
			}
			results <- secret
		}()
	}
	start.Done()

	var want string
	for i := 0; i < workers; i++ {
		select {
		case err := <-errs:
			t.Fatalf("loadOrCreateRuntimeLiveWaitSecret() error = %v", err)
		case got := <-results:
			if want == "" {
				want = got
				continue
			}
			if got != want {
				t.Fatalf("secret = %q, want %q", got, want)
			}
		case <-time.After(2 * time.Second):
			t.Fatal("concurrent secret bootstrap timed out")
		}
	}
	if want == "" {
		t.Fatal("expected one converged secret")
	}
}

// TestLoadOrCreateRuntimeLiveWaitSecretRepairsInvalidFile verifies invalid persisted secrets are regenerated.
func TestLoadOrCreateRuntimeLiveWaitSecretRepairsInvalidFile(t *testing.T) {
	root := t.TempDir()
	secretPath := runtimeLiveWaitSecretPath(root)
	if err := os.MkdirAll(filepath.Dir(secretPath), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(secretPath, []byte("\n"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	secret, err := loadOrCreateRuntimeLiveWaitSecret(root)
	if err != nil {
		t.Fatalf("loadOrCreateRuntimeLiveWaitSecret() error = %v", err)
	}
	if _, err := hex.DecodeString(secret); err != nil {
		t.Fatalf("secret decode error = %v", err)
	}
}

// crossProcessAuthFixture stores one shared runtime for service-level auth wake tests.
type crossProcessAuthFixture struct {
	waitRepo       *sqlite.Repository
	approveRepo    *sqlite.Repository
	waitService    *app.Service
	approveService *app.Service
	project        domain.Project
}

// newCrossProcessAuthFixture constructs two service instances backed by the same runtime database.
func newCrossProcessAuthFixture(t *testing.T) crossProcessAuthFixture {
	t.Helper()

	root := t.TempDir()
	dbPath := filepath.Join(root, "tillsyn.db")

	waitRepo, err := sqlite.Open(dbPath)
	if err != nil {
		t.Fatalf("sqlite.Open(wait) error = %v", err)
	}
	t.Cleanup(func() {
		_ = waitRepo.Close()
	})

	approveRepo, err := sqlite.Open(dbPath)
	if err != nil {
		t.Fatalf("sqlite.Open(approve) error = %v", err)
	}
	t.Cleanup(func() {
		_ = approveRepo.Close()
	})

	waitAuth, err := autentauth.NewSharedDB(autentauth.Config{DB: waitRepo.DB()})
	if err != nil {
		t.Fatalf("NewSharedDB(wait) error = %v", err)
	}
	if err := waitAuth.EnsureDogfoodPolicy(context.Background()); err != nil {
		t.Fatalf("EnsureDogfoodPolicy(wait) error = %v", err)
	}

	approveAuth, err := autentauth.NewSharedDB(autentauth.Config{DB: approveRepo.DB()})
	if err != nil {
		t.Fatalf("NewSharedDB(approve) error = %v", err)
	}
	if err := approveAuth.EnsureDogfoodPolicy(context.Background()); err != nil {
		t.Fatalf("EnsureDogfoodPolicy(approve) error = %v", err)
	}

	waitBroker, err := newRuntimeLiveWaitBroker(waitRepo.DB(), root)
	if err != nil {
		t.Fatalf("NewBroker(wait) error = %v", err)
	}
	t.Cleanup(func() {
		_ = waitBroker.Close()
	})

	approveBroker, err := newRuntimeLiveWaitBroker(approveRepo.DB(), root)
	if err != nil {
		t.Fatalf("NewBroker(approve) error = %v", err)
	}
	t.Cleanup(func() {
		_ = approveBroker.Close()
	})
	if _, err := os.Stat(runtimeLiveWaitSecretPath(root)); err != nil {
		t.Fatalf("runtimeLiveWaitSecretPath(%q) error = %v", root, err)
	}

	project, err := domain.NewProject("p1", "Project One", "", time.Date(2026, 3, 20, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("NewProject() error = %v", err)
	}
	if err := waitRepo.CreateProject(context.Background(), project); err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}

	return crossProcessAuthFixture{
		waitRepo:    waitRepo,
		approveRepo: approveRepo,
		waitService: app.NewService(waitRepo, func() string { return "wait-id" }, time.Now, app.ServiceConfig{
			AuthRequests:   waitAuth,
			AuthBackend:    waitAuth,
			LiveWaitBroker: waitBroker,
		}),
		approveService: app.NewService(approveRepo, func() string { return "approve-id" }, time.Now, app.ServiceConfig{
			AuthRequests:   approveAuth,
			AuthBackend:    approveAuth,
			LiveWaitBroker: approveBroker,
		}),
		project: project,
	}
}

// waitForLiveWaitSubscription waits until one claimant has registered a live-wait subscription for the given event key.
func waitForLiveWaitSubscription(t *testing.T, db *sql.DB, eventType, key string) {
	t.Helper()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		var count int
		if err := db.QueryRowContext(context.Background(), `
			SELECT count(*)
			FROM live_wait_subscriptions
			WHERE event_type = ? AND key = ?
		`, eventType, key).Scan(&count); err != nil {
			t.Fatalf("query live wait subscription count error = %v", err)
		}
		if count > 0 {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("live wait subscription for event_type=%q key=%q never appeared", eventType, key)
}
