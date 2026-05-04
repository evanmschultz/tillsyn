package autentauth

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/evanmschultz/tillsyn/internal/adapters/storage/sqlite"
	"github.com/evanmschultz/tillsyn/internal/app"
	"github.com/evanmschultz/tillsyn/internal/domain"
)

// openSharedDBService constructs one deterministic shared-DB auth service plus its shared repository for app-facing session tests.
func openSharedDBService(t *testing.T, now time.Time) (*Service, *sqlite.Repository) {
	t.Helper()
	repo, err := sqlite.Open(filepath.Join(t.TempDir(), "tillsyn.db"))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(func() {
		_ = repo.Close()
	})

	nextID := 0
	auth, err := NewSharedDB(Config{
		DB: repo.DB(),
		IDGenerator: func() string {
			nextID++
			return fmt.Sprintf("auth-id-%03d", nextID)
		},
		Clock: func() time.Time { return now },
	})
	if err != nil {
		t.Fatalf("NewSharedDB() error = %v", err)
	}
	if err := auth.EnsureDogfoodPolicy(context.Background()); err != nil {
		t.Fatalf("EnsureDogfoodPolicy() error = %v", err)
	}
	return auth, repo
}

// TestServiceAppFacingSessionLifecycle verifies app-facing session wrappers preserve identity decoration and fail closed on invalid state filters.
func TestServiceAppFacingSessionLifecycle(t *testing.T) {
	t.Parallel()

	auth, _ := openSharedDBService(t, time.Date(2026, 3, 20, 12, 0, 0, 0, time.UTC))

	issued, err := auth.IssueAuthSession(context.Background(), app.AuthSessionIssueInput{
		PrincipalID:   "agent-1",
		PrincipalType: "agent",
		PrincipalName: "Agent One",
		ClientID:      "till-mcp-stdio",
		ClientType:    "mcp-stdio",
		ClientName:    "Till MCP STDIO",
		TTL:           2 * time.Hour,
	})
	if err != nil {
		t.Fatalf("IssueAuthSession() error = %v", err)
	}
	if issued.Session.SessionID == "" || issued.Secret == "" {
		t.Fatalf("IssueAuthSession() = %#v, want issued session and secret", issued)
	}

	listed, err := auth.ListAuthSessions(context.Background(), app.AuthSessionFilter{
		ProjectID:   "",
		PrincipalID: "agent-1",
		State:       "active",
		Limit:       10,
	})
	if err != nil {
		t.Fatalf("ListAuthSessions(active) error = %v", err)
	}
	if len(listed) != 1 || listed[0].SessionID != issued.Session.SessionID {
		t.Fatalf("ListAuthSessions(active) = %#v, want issued session", listed)
	}
	if got := listed[0].ProjectID; got != "" {
		t.Fatalf("ListAuthSessions(active) project_id = %q, want empty for direct session", got)
	}

	validated, err := auth.ValidateAuthSession(context.Background(), issued.Session.SessionID, issued.Secret)
	if err != nil {
		t.Fatalf("ValidateAuthSession() error = %v", err)
	}
	if validated.Session.PrincipalID != "agent-1" || validated.Session.PrincipalType != "agent" {
		t.Fatalf("ValidateAuthSession() = %#v, want preserved agent identity", validated)
	}

	revoked, err := auth.RevokeAuthSession(context.Background(), issued.Session.SessionID, "operator_revoke")
	if err != nil {
		t.Fatalf("RevokeAuthSession() error = %v", err)
	}
	if revoked.RevokedAt == nil || revoked.RevocationReason != "operator_revoke" {
		t.Fatalf("RevokeAuthSession() = %#v, want revoked session details", revoked)
	}

	if _, err := auth.ListAuthSessions(context.Background(), app.AuthSessionFilter{
		PrincipalID: "agent-1",
		State:       "bad-state",
		Limit:       10,
	}); err == nil {
		t.Fatal("ListAuthSessions(bad-state) error = nil, want validation failure")
	}
}

// TestServiceAppFacingSessionListMatchesProjectForBroaderScopes verifies project-filtered inventory includes broader approved scopes that still apply to the project.
func TestServiceAppFacingSessionListMatchesProjectForBroaderScopes(t *testing.T) {
	t.Parallel()

	auth, repo := openSharedDBService(t, time.Date(2026, 3, 20, 12, 0, 0, 0, time.UTC))
	ctx := context.Background()
	now := time.Date(2026, 3, 20, 12, 0, 0, 0, time.UTC)

	for _, project := range []struct {
		id   string
		name string
	}{
		{id: "p1", name: "Project One"},
		{id: "p2", name: "Project Two"},
	} {
		row, err := domain.NewProjectFromInput(domain.ProjectInput{ID: project.id, Name: project.name}, now)
		if err != nil {
			t.Fatalf("NewProjectFromInput(%q) error = %v", project.id, err)
		}
		if err := repo.CreateProject(ctx, row); err != nil {
			t.Fatalf("CreateProject(%q) error = %v", project.id, err)
		}
	}

	multi, err := auth.CreateAuthRequest(ctx, domain.AuthRequest{
		ID:                  "req-multi",
		ProjectID:           "p1",
		Path:                "projects/p1,p2",
		ScopeType:           domain.ScopeLevelProject,
		ScopeID:             "p1",
		PrincipalID:         "orch-1",
		PrincipalType:       "agent",
		PrincipalRole:       "orchestrator",
		ClientID:            "till-mcp-stdio",
		ClientType:          "mcp-stdio",
		RequestedSessionTTL: time.Hour,
		State:               domain.AuthRequestStatePending,
		RequestedByActor:    "lane-user",
		RequestedByType:     domain.ActorTypeUser,
		CreatedAt:           time.Now().UTC(),
		ExpiresAt:           time.Now().UTC().Add(30 * time.Minute),
	})
	if err != nil {
		t.Fatalf("CreateAuthRequest(multi) error = %v", err)
	}
	approvedMulti, err := auth.ApproveAuthRequest(ctx, app.ApproveAuthRequestGatewayInput{
		RequestID:    multi.ID,
		ResolvedBy:   "approver-1",
		ResolvedType: domain.ActorTypeUser,
	})
	if err != nil {
		t.Fatalf("ApproveAuthRequest(multi) error = %v", err)
	}

	global, err := auth.CreateAuthRequest(ctx, domain.AuthRequest{
		ID:                  "req-global",
		ProjectID:           domain.AuthRequestGlobalProjectID,
		Path:                "global",
		ScopeType:           domain.ScopeLevelProject,
		ScopeID:             domain.AuthRequestGlobalProjectID,
		PrincipalID:         "orch-2",
		PrincipalType:       "agent",
		PrincipalRole:       "orchestrator",
		ClientID:            "till-mcp-stdio",
		ClientType:          "mcp-stdio",
		RequestedSessionTTL: time.Hour,
		State:               domain.AuthRequestStatePending,
		RequestedByActor:    "lane-user",
		RequestedByType:     domain.ActorTypeUser,
		CreatedAt:           time.Now().UTC(),
		ExpiresAt:           time.Now().UTC().Add(30 * time.Minute),
	})
	if err != nil {
		t.Fatalf("CreateAuthRequest(global) error = %v", err)
	}
	approvedGlobal, err := auth.ApproveAuthRequest(ctx, app.ApproveAuthRequestGatewayInput{
		RequestID:    global.ID,
		ResolvedBy:   "approver-1",
		ResolvedType: domain.ActorTypeUser,
	})
	if err != nil {
		t.Fatalf("ApproveAuthRequest(global) error = %v", err)
	}

	listed, err := auth.ListAuthSessions(ctx, app.AuthSessionFilter{
		ProjectID: "p2",
		State:     "active",
	})
	if err != nil {
		t.Fatalf("ListAuthSessions(project filtered) error = %v", err)
	}
	if len(listed) != 2 {
		t.Fatalf("ListAuthSessions(project filtered) len = %d, want 2", len(listed))
	}
	if listed[0].SessionID != approvedGlobal.Request.IssuedSessionID && listed[1].SessionID != approvedGlobal.Request.IssuedSessionID {
		t.Fatalf("project filtered sessions missing global session: %#v", listed)
	}
	if listed[0].SessionID != approvedMulti.Request.IssuedSessionID && listed[1].SessionID != approvedMulti.Request.IssuedSessionID {
		t.Fatalf("project filtered sessions missing multi-project session: %#v", listed)
	}
}
