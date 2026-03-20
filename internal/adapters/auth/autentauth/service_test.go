package autentauth

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	autentdomain "github.com/evanmschultz/autent/domain"
	"github.com/hylla/tillsyn/internal/adapters/storage/sqlite"
	"github.com/hylla/tillsyn/internal/domain"
)

// TestServiceSharedDBAuthorizeAllow verifies shared-DB setup, default policy seeding, and one allowed authorization flow.
func TestServiceSharedDBAuthorizeAllow(t *testing.T) {
	t.Parallel()

	repo, err := sqlite.Open(filepath.Join(t.TempDir(), "tillsyn.db"))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(func() {
		_ = repo.Close()
	})

	now := time.Date(2026, 3, 20, 12, 0, 0, 0, time.UTC)
	nextID := 0
	auth, err := NewSharedDB(Config{
		DB: repo.DB(),
		IDGenerator: func() string {
			nextID++
			return fmt.Sprintf("auth-id-%03d", nextID)
		},
		Clock: func() time.Time {
			return now
		},
	})
	if err != nil {
		t.Fatalf("NewSharedDB() error = %v", err)
	}
	if err := auth.EnsureDogfoodPolicy(context.Background()); err != nil {
		t.Fatalf("EnsureDogfoodPolicy() error = %v", err)
	}

	var version int
	row := repo.DB().QueryRowContext(context.Background(), "SELECT version FROM autent_schema_migrations LIMIT 1")
	if err := row.Scan(&version); err != nil {
		t.Fatalf("scan autent schema version error = %v", err)
	}
	if version != 1 {
		t.Fatalf("autent schema version = %d, want 1", version)
	}

	issued, err := auth.IssueSession(context.Background(), IssueSessionInput{
		PrincipalID:   "agent-1",
		PrincipalType: "agent",
		PrincipalName: "Agent One",
		ClientID:      "till-mcp-stdio",
		ClientType:    "mcp-stdio",
		ClientName:    "Till MCP STDIO",
	})
	if err != nil {
		t.Fatalf("IssueSession() error = %v", err)
	}
	if issued.Session.ID == "" || issued.Secret == "" {
		t.Fatalf("IssueSession() returned empty session bundle: %#v", issued)
	}

	result, err := auth.Authorize(context.Background(), AuthorizationRequest{
		SessionID:     issued.Session.ID,
		SessionSecret: issued.Secret,
		Action:        "create_task",
		Namespace:     "project:p1",
		ResourceType:  "task",
		ResourceID:    "new",
	})
	if err != nil {
		t.Fatalf("Authorize() error = %v", err)
	}
	if result.DecisionCode != "allow" {
		t.Fatalf("Authorize() decision = %q, want allow", result.DecisionCode)
	}
	if result.Caller.PrincipalID != "agent-1" {
		t.Fatalf("Authorize() caller principal_id = %q, want agent-1", result.Caller.PrincipalID)
	}
	if result.Caller.PrincipalName != "Agent One" {
		t.Fatalf("Authorize() caller principal_name = %q, want Agent One", result.Caller.PrincipalName)
	}
}

// TestServiceAuthorizeInvalidSecretReturnsDecision verifies invalid secrets fail with stable invalid semantics.
func TestServiceAuthorizeInvalidSecretReturnsDecision(t *testing.T) {
	t.Parallel()

	repo, err := sqlite.Open(filepath.Join(t.TempDir(), "tillsyn.db"))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(func() {
		_ = repo.Close()
	})

	auth, err := NewSharedDB(Config{DB: repo.DB()})
	if err != nil {
		t.Fatalf("NewSharedDB() error = %v", err)
	}
	if err := auth.EnsureDogfoodPolicy(context.Background()); err != nil {
		t.Fatalf("EnsureDogfoodPolicy() error = %v", err)
	}
	issued, err := auth.IssueSession(context.Background(), IssueSessionInput{
		PrincipalID:   "user-1",
		PrincipalType: "user",
		PrincipalName: "User One",
		ClientID:      "till-mcp-stdio",
		ClientType:    "mcp-stdio",
		ClientName:    "Till MCP STDIO",
	})
	if err != nil {
		t.Fatalf("IssueSession() error = %v", err)
	}

	result, err := auth.Authorize(context.Background(), AuthorizationRequest{
		SessionID:     issued.Session.ID,
		SessionSecret: "wrong-secret",
		Action:        "update_task",
		Namespace:     "project:p1",
		ResourceType:  "task",
		ResourceID:    "t1",
	})
	if err != nil {
		t.Fatalf("Authorize() error = %v", err)
	}
	if result.DecisionCode != "invalid" {
		t.Fatalf("Authorize() decision = %q, want invalid", result.DecisionCode)
	}
}

// TestNewSharedDBRequiresDB verifies shared-DB setup fails closed without one caller-owned database handle.
func TestNewSharedDBRequiresDB(t *testing.T) {
	t.Parallel()

	if _, err := NewSharedDB(Config{}); !errors.Is(err, autentdomain.ErrInvalidConfig) {
		t.Fatalf("NewSharedDB() error = %v, want invalid config", err)
	}
}

// TestServiceIssueSessionReusesExistingPrincipalAndClient verifies repeated issuance reuses already-registered auth records.
func TestServiceIssueSessionReusesExistingPrincipalAndClient(t *testing.T) {
	t.Parallel()

	repo, err := sqlite.Open(filepath.Join(t.TempDir(), "tillsyn.db"))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(func() {
		_ = repo.Close()
	})

	auth, err := NewSharedDB(Config{DB: repo.DB()})
	if err != nil {
		t.Fatalf("NewSharedDB() error = %v", err)
	}
	if err := auth.EnsureDogfoodPolicy(context.Background()); err != nil {
		t.Fatalf("EnsureDogfoodPolicy() error = %v", err)
	}

	first, err := auth.IssueSession(context.Background(), IssueSessionInput{
		PrincipalID:   "agent-1",
		PrincipalType: "agent",
		ClientID:      "till-mcp-stdio",
		ClientType:    "mcp-stdio",
	})
	if err != nil {
		t.Fatalf("first IssueSession() error = %v", err)
	}
	second, err := auth.IssueSession(context.Background(), IssueSessionInput{
		PrincipalID:   "agent-1",
		PrincipalType: "agent",
		ClientID:      "till-mcp-stdio",
		ClientType:    "mcp-stdio",
	})
	if err != nil {
		t.Fatalf("second IssueSession() error = %v", err)
	}
	if first.Session.ID == second.Session.ID {
		t.Fatalf("IssueSession() reused session id %q, want distinct sessions", first.Session.ID)
	}

	var principals int
	if err := repo.DB().QueryRowContext(context.Background(), "SELECT COUNT(*) FROM autent_principals").Scan(&principals); err != nil {
		t.Fatalf("count autent_principals error = %v", err)
	}
	if principals != 1 {
		t.Fatalf("autent_principals count = %d, want 1", principals)
	}

	var clients int
	if err := repo.DB().QueryRowContext(context.Background(), "SELECT COUNT(*) FROM autent_clients").Scan(&clients); err != nil {
		t.Fatalf("count autent_clients error = %v", err)
	}
	if clients != 1 {
		t.Fatalf("autent_clients count = %d, want 1", clients)
	}
}

// TestServiceRevokeSessionMarksSessionRevoked verifies revocation returns caller-safe session state.
func TestServiceRevokeSessionMarksSessionRevoked(t *testing.T) {
	t.Parallel()

	repo, err := sqlite.Open(filepath.Join(t.TempDir(), "tillsyn.db"))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(func() {
		_ = repo.Close()
	})

	auth, err := NewSharedDB(Config{DB: repo.DB()})
	if err != nil {
		t.Fatalf("NewSharedDB() error = %v", err)
	}
	if err := auth.EnsureDogfoodPolicy(context.Background()); err != nil {
		t.Fatalf("EnsureDogfoodPolicy() error = %v", err)
	}
	issued, err := auth.IssueSession(context.Background(), IssueSessionInput{
		PrincipalID: "user-1",
		ClientID:    "till-mcp-stdio",
		ClientType:  "mcp-stdio",
	})
	if err != nil {
		t.Fatalf("IssueSession() error = %v", err)
	}

	revoked, err := auth.RevokeSession(context.Background(), issued.Session.ID, "operator_revoke")
	if err != nil {
		t.Fatalf("RevokeSession() error = %v", err)
	}
	if revoked.ID != issued.Session.ID {
		t.Fatalf("RevokeSession() id = %q, want %q", revoked.ID, issued.Session.ID)
	}
	if revoked.RevokedAt == nil {
		t.Fatal("RevokeSession() revoked_at = nil, want timestamp")
	}
	if revoked.RevocationReason != "operator_revoke" {
		t.Fatalf("RevokeSession() reason = %q, want operator_revoke", revoked.RevocationReason)
	}
}

// TestServiceAuthorizeRevokedSessionReturnsDecision verifies revoked sessions remain distinguishable from valid sessions.
func TestServiceAuthorizeRevokedSessionReturnsDecision(t *testing.T) {
	t.Parallel()

	repo, err := sqlite.Open(filepath.Join(t.TempDir(), "tillsyn.db"))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(func() {
		_ = repo.Close()
	})

	auth, err := NewSharedDB(Config{DB: repo.DB()})
	if err != nil {
		t.Fatalf("NewSharedDB() error = %v", err)
	}
	if err := auth.EnsureDogfoodPolicy(context.Background()); err != nil {
		t.Fatalf("EnsureDogfoodPolicy() error = %v", err)
	}
	issued, err := auth.IssueSession(context.Background(), IssueSessionInput{
		PrincipalID: "user-1",
		ClientID:    "till-mcp-stdio",
		ClientType:  "mcp-stdio",
	})
	if err != nil {
		t.Fatalf("IssueSession() error = %v", err)
	}
	if _, err := auth.RevokeSession(context.Background(), issued.Session.ID, "operator_revoke"); err != nil {
		t.Fatalf("RevokeSession() error = %v", err)
	}

	result, err := auth.Authorize(context.Background(), AuthorizationRequest{
		SessionID:     issued.Session.ID,
		SessionSecret: issued.Secret,
		Action:        "update_task",
		Namespace:     "project:p1",
		ResourceType:  "task",
		ResourceID:    "t1",
	})
	if err != nil {
		t.Fatalf("Authorize() error = %v", err)
	}
	if result.DecisionCode != "invalid" {
		t.Fatalf("Authorize() decision = %q, want invalid", result.DecisionCode)
	}
}

// TestServiceAuthorizeSessionRequired verifies missing credentials fail with stable session-required semantics.
func TestServiceAuthorizeSessionRequired(t *testing.T) {
	t.Parallel()

	repo, err := sqlite.Open(filepath.Join(t.TempDir(), "tillsyn.db"))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(func() {
		_ = repo.Close()
	})

	auth, err := NewSharedDB(Config{DB: repo.DB()})
	if err != nil {
		t.Fatalf("NewSharedDB() error = %v", err)
	}
	if err := auth.EnsureDogfoodPolicy(context.Background()); err != nil {
		t.Fatalf("EnsureDogfoodPolicy() error = %v", err)
	}

	result, err := auth.Authorize(context.Background(), AuthorizationRequest{
		Action:       "create_task",
		Namespace:    "project:p1",
		ResourceType: "task",
		ResourceID:   "new",
	})
	if err != nil {
		t.Fatalf("Authorize() error = %v", err)
	}
	if result.DecisionCode != "session_required" {
		t.Fatalf("Authorize() decision = %q, want session_required", result.DecisionCode)
	}
}

// TestServiceAuthorizeExpiredSessionReturnsDecision verifies expired sessions remain distinguishable from invalid credentials.
func TestServiceAuthorizeExpiredSessionReturnsDecision(t *testing.T) {
	t.Parallel()

	repo, err := sqlite.Open(filepath.Join(t.TempDir(), "tillsyn.db"))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(func() {
		_ = repo.Close()
	})

	now := time.Date(2026, 3, 20, 12, 0, 0, 0, time.UTC)
	auth, err := NewSharedDB(Config{
		DB: repo.DB(),
		Clock: func() time.Time {
			return now
		},
	})
	if err != nil {
		t.Fatalf("NewSharedDB() error = %v", err)
	}
	if err := auth.EnsureDogfoodPolicy(context.Background()); err != nil {
		t.Fatalf("EnsureDogfoodPolicy() error = %v", err)
	}
	issued, err := auth.IssueSession(context.Background(), IssueSessionInput{
		PrincipalID: "user-1",
		ClientID:    "till-mcp-stdio",
		ClientType:  "mcp-stdio",
		TTL:         time.Minute,
	})
	if err != nil {
		t.Fatalf("IssueSession() error = %v", err)
	}

	now = now.Add(2 * time.Minute)
	result, err := auth.Authorize(context.Background(), AuthorizationRequest{
		SessionID:     issued.Session.ID,
		SessionSecret: issued.Secret,
		Action:        "update_task",
		Namespace:     "project:p1",
		ResourceType:  "task",
		ResourceID:    "t1",
	})
	if err != nil {
		t.Fatalf("Authorize() error = %v", err)
	}
	if result.DecisionCode != "session_expired" {
		t.Fatalf("Authorize() decision = %q, want session_expired", result.DecisionCode)
	}
}

// TestServiceAuthorizeDenyRuleReturnsDecision verifies explicit deny policy returns stable deny semantics.
func TestServiceAuthorizeDenyRuleReturnsDecision(t *testing.T) {
	t.Parallel()

	repo, err := sqlite.Open(filepath.Join(t.TempDir(), "tillsyn.db"))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(func() {
		_ = repo.Close()
	})

	auth, err := NewSharedDB(Config{DB: repo.DB()})
	if err != nil {
		t.Fatalf("NewSharedDB() error = %v", err)
	}
	denyRule, err := autentdomain.ValidateAndNormalizeRule(autentdomain.Rule{
		ID:     "deny-create-task",
		Effect: autentdomain.EffectDeny,
		Actions: []autentdomain.StringPattern{
			{Operator: autentdomain.MatchExact, Value: "create_task"},
		},
		Resources: []autentdomain.ResourcePattern{
			{
				Namespace: autentdomain.StringPattern{Operator: autentdomain.MatchExact, Value: "project:p1"},
				Type:      autentdomain.StringPattern{Operator: autentdomain.MatchExact, Value: "task"},
				ID:        autentdomain.StringPattern{Operator: autentdomain.MatchExact, Value: "new"},
			},
		},
		Priority: 10,
	})
	if err != nil {
		t.Fatalf("ValidateAndNormalizeRule() error = %v", err)
	}
	if err := auth.ReplaceRules(context.Background(), []autentdomain.Rule{denyRule}); err != nil {
		t.Fatalf("ReplaceRules() error = %v", err)
	}
	issued, err := auth.IssueSession(context.Background(), IssueSessionInput{
		PrincipalID: "user-1",
		ClientID:    "till-mcp-stdio",
		ClientType:  "mcp-stdio",
	})
	if err != nil {
		t.Fatalf("IssueSession() error = %v", err)
	}

	result, err := auth.Authorize(context.Background(), AuthorizationRequest{
		SessionID:     issued.Session.ID,
		SessionSecret: issued.Secret,
		Action:        "create_task",
		Namespace:     "project:p1",
		ResourceType:  "task",
		ResourceID:    "new",
	})
	if err != nil {
		t.Fatalf("Authorize() error = %v", err)
	}
	if result.DecisionCode != "deny" {
		t.Fatalf("Authorize() decision = %q, want deny", result.DecisionCode)
	}
}

// TestServiceAuthorizeGrantRequiredReturnsDecision verifies escalation-capable rules return grant-required before approval.
func TestServiceAuthorizeGrantRequiredReturnsDecision(t *testing.T) {
	t.Parallel()

	repo, err := sqlite.Open(filepath.Join(t.TempDir(), "tillsyn.db"))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(func() {
		_ = repo.Close()
	})

	auth, err := NewSharedDB(Config{DB: repo.DB()})
	if err != nil {
		t.Fatalf("NewSharedDB() error = %v", err)
	}
	grantRule, err := autentdomain.ValidateAndNormalizeRule(autentdomain.Rule{
		ID:     "grant-create-task",
		Effect: autentdomain.EffectAllow,
		Actions: []autentdomain.StringPattern{
			{Operator: autentdomain.MatchExact, Value: "create_task"},
		},
		Resources: []autentdomain.ResourcePattern{
			{
				Namespace: autentdomain.StringPattern{Operator: autentdomain.MatchExact, Value: "project:p1"},
				Type:      autentdomain.StringPattern{Operator: autentdomain.MatchExact, Value: "task"},
				ID:        autentdomain.StringPattern{Operator: autentdomain.MatchExact, Value: "new"},
			},
		},
		Escalation: &autentdomain.EscalationRequirement{Allowed: true},
		Priority:   10,
	})
	if err != nil {
		t.Fatalf("ValidateAndNormalizeRule() error = %v", err)
	}
	if err := auth.ReplaceRules(context.Background(), []autentdomain.Rule{grantRule}); err != nil {
		t.Fatalf("ReplaceRules() error = %v", err)
	}
	issued, err := auth.IssueSession(context.Background(), IssueSessionInput{
		PrincipalID: "user-1",
		ClientID:    "till-mcp-stdio",
		ClientType:  "mcp-stdio",
	})
	if err != nil {
		t.Fatalf("IssueSession() error = %v", err)
	}

	result, err := auth.Authorize(context.Background(), AuthorizationRequest{
		SessionID:     issued.Session.ID,
		SessionSecret: issued.Secret,
		Action:        "create_task",
		Namespace:     "project:p1",
		ResourceType:  "task",
		ResourceID:    "new",
	})
	if err != nil {
		t.Fatalf("Authorize() error = %v", err)
	}
	if result.DecisionCode != "grant_required" {
		t.Fatalf("Authorize() decision = %q, want grant_required", result.DecisionCode)
	}
}

// TestPrincipalTypeHelpers verifies principal-type normalization and caller mapping stay stable for dogfood auth.
func TestPrincipalTypeHelpers(t *testing.T) {
	t.Parallel()

	if got := normalizePrincipalType("AGENT"); got != autentdomain.PrincipalTypeAgent {
		t.Fatalf("normalizePrincipalType() = %q, want %q", got, autentdomain.PrincipalTypeAgent)
	}
	if got := principalTypeToActorType(autentdomain.PrincipalTypeService); got != domain.ActorTypeAgent {
		t.Fatalf("principalTypeToActorType(service) = %q, want agent", got)
	}
	if got := principalTypeToActorType(autentdomain.PrincipalTypeUser); got != domain.ActorTypeUser {
		t.Fatalf("principalTypeToActorType(user) = %q, want user", got)
	}
	if clone := cloneContext(nil); clone != nil {
		t.Fatalf("cloneContext(nil) = %#v, want nil", clone)
	}
}
