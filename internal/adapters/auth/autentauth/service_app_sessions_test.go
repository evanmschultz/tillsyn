package autentauth

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/hylla/tillsyn/internal/adapters/storage/sqlite"
	"github.com/hylla/tillsyn/internal/app"
)

// openSharedDBService constructs one deterministic shared-DB auth service for app-facing session tests.
func openSharedDBService(t *testing.T, now time.Time) *Service {
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
	return auth
}

// TestServiceAppFacingSessionLifecycle verifies app-facing session wrappers preserve identity decoration and fail closed on invalid state filters.
func TestServiceAppFacingSessionLifecycle(t *testing.T) {
	t.Parallel()

	auth := openSharedDBService(t, time.Date(2026, 3, 20, 12, 0, 0, 0, time.UTC))

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
