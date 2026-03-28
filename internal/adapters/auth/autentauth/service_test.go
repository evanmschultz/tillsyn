package autentauth

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	autent "github.com/evanmschultz/autent"
	autentdomain "github.com/evanmschultz/autent/domain"
	"github.com/hylla/tillsyn/internal/adapters/storage/sqlite"
	"github.com/hylla/tillsyn/internal/app"
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

// TestAuthorizeApprovedPathRejectsProjectMutationForBranchScopedApproval verifies narrower approvals do not silently widen to project scope.
func TestAuthorizeApprovedPathRejectsProjectMutationForBranchScopedApproval(t *testing.T) {
	t.Parallel()

	err := authorizeApprovedPath(
		map[string]string{"approved_path": "project/p1/branch/b1"},
		map[string]string{"project_id": "p1"},
	)
	if !errors.Is(err, domain.ErrInvalidScopeID) {
		t.Fatalf("authorizeApprovedPath() error = %v, want ErrInvalidScopeID", err)
	}
}

// TestAuthorizeApprovedPathAllowsMatchingBranchMutation verifies matching branch context remains allowed.
func TestAuthorizeApprovedPathAllowsMatchingBranchMutation(t *testing.T) {
	t.Parallel()

	err := authorizeApprovedPath(
		map[string]string{"approved_path": "project/p1/branch/b1"},
		map[string]string{
			"project_id": "p1",
			"scope_type": string(domain.ScopeLevelBranch),
			"scope_id":   "b1",
		},
	)
	if err != nil {
		t.Fatalf("authorizeApprovedPath() error = %v, want nil", err)
	}
}

// TestAuthorizeApprovedPathRejectsDifferentPhaseMutation verifies phase-scoped approvals block sibling phase work.
func TestAuthorizeApprovedPathRejectsDifferentPhaseMutation(t *testing.T) {
	t.Parallel()

	err := authorizeApprovedPath(
		map[string]string{"approved_path": "project/p1/branch/b1/phase/ph-a"},
		map[string]string{
			"project_id": "p1",
			"branch_id":  "b1",
			"scope_type": string(domain.ScopeLevelPhase),
			"scope_id":   "ph-b",
		},
	)
	if !errors.Is(err, domain.ErrInvalidScopeID) {
		t.Fatalf("authorizeApprovedPath() error = %v, want ErrInvalidScopeID", err)
	}
}

// TestAuthorizeApprovedPathAllowsMultiProjectOrchestratorScope verifies multi-project approvals authorize matching projects without pretending to be project-rooted paths.
func TestAuthorizeApprovedPathAllowsMultiProjectOrchestratorScope(t *testing.T) {
	t.Parallel()

	err := authorizeApprovedPath(
		map[string]string{"approved_path": "projects/p1,p2"},
		map[string]string{
			"project_id": "p2",
			"scope_type": string(domain.ScopeLevelProject),
			"scope_id":   "p2",
		},
	)
	if err != nil {
		t.Fatalf("authorizeApprovedPath() error = %v, want nil", err)
	}
}

// TestAuthorizeApprovedPathAllowsGlobalScope verifies global approvals authorize work across projects.
func TestAuthorizeApprovedPathAllowsGlobalScope(t *testing.T) {
	t.Parallel()

	err := authorizeApprovedPath(
		map[string]string{"approved_path": "global"},
		map[string]string{
			"project_id": "p9",
			"scope_type": string(domain.ScopeLevelProject),
			"scope_id":   "p9",
		},
	)
	if err != nil {
		t.Fatalf("authorizeApprovedPath() error = %v, want nil", err)
	}
}

// TestAuthContextPathBuildsTaskScopePath verifies task/subtask contexts must carry lineage to satisfy narrowed approvals.
func TestAuthContextPathBuildsTaskScopePath(t *testing.T) {
	t.Parallel()

	path, err := authContextPath(map[string]string{
		"project_id": "p1",
		"scope_type": string(domain.ScopeLevelTask),
		"scope_id":   "task-1",
		"branch_id":  "b1",
		"phase_path": "ph-a/ph-b",
	})
	if err != nil {
		t.Fatalf("authContextPath() error = %v", err)
	}
	if got := path.String(); got != "project/p1/branch/b1/phase/ph-a/phase/ph-b" {
		t.Fatalf("authContextPath() path = %q, want project/p1/branch/b1/phase/ph-a/phase/ph-b", got)
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

// TestServiceAuthRequestLifecycleWithScopedApproval verifies request lifecycle state transitions and approved-path enforcement.
func TestServiceAuthRequestLifecycleWithScopedApproval(t *testing.T) {
	ctx := context.Background()
	repo, err := sqlite.Open(filepath.Join(t.TempDir(), "tillsyn.db"))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(func() {
		_ = repo.Close()
	})

	now := time.Date(2026, 3, 20, 12, 15, 0, 0, time.UTC)
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
	if err := auth.EnsureDogfoodPolicy(ctx); err != nil {
		t.Fatalf("EnsureDogfoodPolicy() error = %v", err)
	}
	project, err := domain.NewProject("p1", "Project One", "", now)
	if err != nil {
		t.Fatalf("NewProject() error = %v", err)
	}
	if err := repo.CreateProject(ctx, project); err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}

	request, err := auth.CreateAuthRequest(ctx, domain.AuthRequest{
		ID:                  "req-1",
		Path:                "project/p1/branch/b1/phase/ph1",
		ProjectID:           "p1",
		BranchID:            "b1",
		PhaseIDs:            []string{"ph1"},
		ScopeType:           domain.ScopeLevelPhase,
		ScopeID:             "ph1",
		PrincipalID:         "agent-1",
		PrincipalType:       "agent",
		PrincipalName:       "Agent One",
		ClientID:            "till-mcp-stdio",
		ClientType:          "mcp-stdio",
		ClientName:          "Till MCP STDIO",
		RequestedSessionTTL: 2 * time.Hour,
		Reason:              "needs review",
		Continuation:        map[string]any{"resume_tool": "till.raise_attention_item", "resume_token": "resume-123", "resume": map[string]any{"path": "project/p1"}},
		State:               domain.AuthRequestStatePending,
		RequestedByActor:    "lane-user",
		RequestedByType:     domain.ActorTypeUser,
		CreatedAt:           now,
		ExpiresAt:           now.Add(30 * time.Minute),
	})
	if err != nil {
		t.Fatalf("CreateAuthRequest() error = %v", err)
	}
	if request.RequestedByActor != "lane-user" || request.State != domain.AuthRequestStatePending {
		t.Fatalf("CreateAuthRequest() = %#v, want pending request", request)
	}

	listed, err := auth.ListAuthRequests(ctx, domain.AuthRequestListFilter{ProjectID: "p1", State: domain.AuthRequestStatePending, Limit: 10})
	if err != nil {
		t.Fatalf("ListAuthRequests() error = %v", err)
	}
	if len(listed) != 1 || listed[0].ID != request.ID {
		t.Fatalf("ListAuthRequests() = %#v, want pending request %q", listed, request.ID)
	}

	approved, err := auth.ApproveAuthRequest(ctx, app.ApproveAuthRequestGatewayInput{
		RequestID:      request.ID,
		ResolvedBy:     "approver-1",
		ResolvedType:   domain.ActorTypeUser,
		ResolutionNote: "approved for branch review",
		PathOverride:   &domain.AuthRequestPath{ProjectID: "p1", BranchID: "b1", PhaseIDs: []string{"ph1", "ph2"}},
		TTLOverride:    time.Hour,
	})
	if err != nil {
		t.Fatalf("ApproveAuthRequest() error = %v", err)
	}
	if approved.Request.State != domain.AuthRequestStateApproved {
		t.Fatalf("ApproveAuthRequest() state = %q, want approved", approved.Request.State)
	}
	if approved.Request.Path != "project/p1/branch/b1/phase/ph1" {
		t.Fatalf("ApproveAuthRequest() requested path = %q, want project/p1/branch/b1/phase/ph1", approved.Request.Path)
	}
	if approved.Request.ApprovedPath != "project/p1/branch/b1/phase/ph1/phase/ph2" {
		t.Fatalf("ApproveAuthRequest() approved_path = %q, want project/p1/branch/b1/phase/ph1/phase/ph2", approved.Request.ApprovedPath)
	}
	if approved.Request.RequestedSessionTTL != 2*time.Hour {
		t.Fatalf("ApproveAuthRequest() requested ttl = %s, want 2h", approved.Request.RequestedSessionTTL)
	}
	if approved.Request.ApprovedSessionTTL != time.Hour {
		t.Fatalf("ApproveAuthRequest() approved ttl = %s, want 1h", approved.Request.ApprovedSessionTTL)
	}
	if approved.SessionSecret == "" {
		t.Fatal("ApproveAuthRequest() returned empty session secret")
	}
	claimed, err := auth.ClaimAuthRequest(ctx, app.ClaimAuthRequestInput{
		RequestID:   request.ID,
		ResumeToken: "resume-123",
		PrincipalID: "agent-1",
		ClientID:    "till-mcp-stdio",
	})
	if err != nil {
		t.Fatalf("ClaimAuthRequest() error = %v", err)
	}
	if claimed.Request.State != domain.AuthRequestStateApproved {
		t.Fatalf("ClaimAuthRequest() state = %q, want approved", claimed.Request.State)
	}
	if claimed.SessionSecret != approved.SessionSecret {
		t.Fatalf("ClaimAuthRequest() session_secret = %q, want approved secret", claimed.SessionSecret)
	}

	allowed, err := auth.Authorize(ctx, AuthorizationRequest{
		SessionID:     approved.Request.IssuedSessionID,
		SessionSecret: approved.SessionSecret,
		Action:        "create_task",
		Namespace:     "project:p1",
		ResourceType:  "task",
		ResourceID:    "new",
		Context: map[string]string{
			"project_id": "p1",
			"branch_id":  "b1",
			"scope_type": string(domain.ScopeLevelPhase),
			"scope_id":   "ph2",
			"phase_path": "ph1/ph2",
		},
	})
	if err != nil {
		t.Fatalf("Authorize(allowed) error = %v", err)
	}
	if allowed.DecisionCode != "allow" {
		t.Fatalf("Authorize(allowed) decision = %q, want allow", allowed.DecisionCode)
	}
	if allowed.Caller.PrincipalID != "agent-1" {
		t.Fatalf("Authorize(allowed) caller principal_id = %q, want agent-1", allowed.Caller.PrincipalID)
	}

	denied, err := auth.Authorize(ctx, AuthorizationRequest{
		SessionID:     approved.Request.IssuedSessionID,
		SessionSecret: approved.SessionSecret,
		Action:        "create_task",
		Namespace:     "project:p1",
		ResourceType:  "task",
		ResourceID:    "new",
		Context: map[string]string{
			"project_id": "p1",
			"branch_id":  "b1",
			"scope_type": string(domain.ScopeLevelPhase),
			"scope_id":   "ph3",
			"phase_path": "ph1/ph3",
		},
	})
	if err != nil {
		t.Fatalf("Authorize(denied) error = %v", err)
	}
	if denied.DecisionCode != "deny" {
		t.Fatalf("Authorize(denied) decision = %q, want deny", denied.DecisionCode)
	}
}

// TestServiceDelegatedAuthRequestClaimSupportsChildOnly verifies delegated
// continuation claims are owned by the approved child principal, while the
// original requester and unrelated callers still fail closed.
func TestServiceDelegatedAuthRequestClaimSupportsChildOnly(t *testing.T) {
	ctx := context.Background()
	repo, err := sqlite.Open(filepath.Join(t.TempDir(), "tillsyn.db"))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(func() {
		_ = repo.Close()
	})

	now := time.Date(2026, 3, 28, 15, 15, 0, 0, time.UTC)
	auth, err := NewSharedDB(Config{
		DB:    repo.DB(),
		Clock: func() time.Time { return now },
	})
	if err != nil {
		t.Fatalf("NewSharedDB() error = %v", err)
	}
	if err := auth.EnsureDogfoodPolicy(ctx); err != nil {
		t.Fatalf("EnsureDogfoodPolicy() error = %v", err)
	}
	project, err := domain.NewProject("p1", "Project One", "", now)
	if err != nil {
		t.Fatalf("NewProject() error = %v", err)
	}
	if err := repo.CreateProject(ctx, project); err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}

	request, err := auth.CreateAuthRequest(ctx, domain.AuthRequest{
		ID:                  "req-delegated",
		Path:                "project/p1",
		ProjectID:           "p1",
		ScopeType:           domain.ScopeLevelProject,
		ScopeID:             "p1",
		PrincipalID:         "builder-1",
		PrincipalType:       "agent",
		PrincipalRole:       "builder",
		ClientID:            "builder-client",
		ClientType:          "mcp-stdio",
		RequestedSessionTTL: 2 * time.Hour,
		Reason:              "orchestrator requests builder scope",
		Continuation: map[string]any{
			"resume_token": "resume-456",
			app.AuthRequestContinuationRequesterClientIDKey: "orchestrator-client",
		},
		State:            domain.AuthRequestStatePending,
		RequestedByActor: "orchestrator-1",
		RequestedByType:  domain.ActorTypeAgent,
		CreatedAt:        now,
		ExpiresAt:        now.Add(30 * time.Minute),
	})
	if err != nil {
		t.Fatalf("CreateAuthRequest() error = %v", err)
	}
	approved, err := auth.ApproveAuthRequest(ctx, app.ApproveAuthRequestGatewayInput{
		RequestID:      request.ID,
		ResolvedBy:     "approver-1",
		ResolvedType:   domain.ActorTypeUser,
		ResolutionNote: "approved for delegated claim",
	})
	if err != nil {
		t.Fatalf("ApproveAuthRequest() error = %v", err)
	}

	claimedByChild, err := auth.ClaimAuthRequest(ctx, app.ClaimAuthRequestInput{
		RequestID:   request.ID,
		ResumeToken: "resume-456",
		PrincipalID: "builder-1",
		ClientID:    "builder-client",
	})
	if err != nil {
		t.Fatalf("ClaimAuthRequest(child) error = %v", err)
	}
	if claimedByChild.SessionSecret != approved.SessionSecret {
		t.Fatalf("ClaimAuthRequest(child) session_secret = %q, want approved secret", claimedByChild.SessionSecret)
	}

	if _, err := auth.ClaimAuthRequest(ctx, app.ClaimAuthRequestInput{
		RequestID:   request.ID,
		ResumeToken: "resume-456",
		PrincipalID: "orchestrator-1",
		ClientID:    "orchestrator-client",
	}); !errors.Is(err, domain.ErrAuthRequestClaimMismatch) {
		t.Fatalf("ClaimAuthRequest(requester) error = %v, want ErrAuthRequestClaimMismatch", err)
	}

	if _, err := auth.ClaimAuthRequest(ctx, app.ClaimAuthRequestInput{
		RequestID:   request.ID,
		ResumeToken: "resume-456",
		PrincipalID: "other-agent",
		ClientID:    "other-client",
	}); !errors.Is(err, domain.ErrAuthRequestClaimMismatch) {
		t.Fatalf("ClaimAuthRequest(other) error = %v, want ErrAuthRequestClaimMismatch", err)
	}
}

// TestServiceAuthRequestTerminalTransitions verifies deny/cancel/expire transitions and filtered listings.
func TestServiceAuthRequestTerminalTransitions(t *testing.T) {
	ctx := context.Background()
	repo, err := sqlite.Open(filepath.Join(t.TempDir(), "tillsyn.db"))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(func() {
		_ = repo.Close()
	})

	now := time.Date(2026, 3, 20, 12, 20, 0, 0, time.UTC)
	current := now
	auth, err := NewSharedDB(Config{
		DB:    repo.DB(),
		Clock: func() time.Time { return current },
	})
	if err != nil {
		t.Fatalf("NewSharedDB() error = %v", err)
	}
	if err := auth.EnsureDogfoodPolicy(ctx); err != nil {
		t.Fatalf("EnsureDogfoodPolicy() error = %v", err)
	}
	project, err := domain.NewProject("p1", "Project One", "", now)
	if err != nil {
		t.Fatalf("NewProject() error = %v", err)
	}
	if err := repo.CreateProject(ctx, project); err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}

	create := func(id, principal string, timeout time.Duration) domain.AuthRequest {
		t.Helper()
		request, err := auth.CreateAuthRequest(ctx, domain.AuthRequest{
			ID:                  id,
			Path:                "project/p1",
			ProjectID:           "p1",
			ScopeType:           domain.ScopeLevelProject,
			ScopeID:             "p1",
			PrincipalID:         principal,
			ClientID:            "till-tui",
			ClientType:          "tui",
			RequestedSessionTTL: 2 * time.Hour,
			Reason:              "needs review",
			State:               domain.AuthRequestStatePending,
			RequestedByType:     domain.ActorTypeUser,
			CreatedAt:           now,
			ExpiresAt:           now.Add(timeout),
		})
		if err != nil {
			t.Fatalf("CreateAuthRequest(%s) error = %v", id, err)
		}
		return request
	}

	deniedReq := create("req-deny", "user-deny", 30*time.Minute)
	denied, err := auth.DenyAuthRequest(ctx, deniedReq.ID, "approver-1", domain.ActorTypeUser, "outside scope")
	if err != nil {
		t.Fatalf("DenyAuthRequest() error = %v", err)
	}
	if denied.State != domain.AuthRequestStateDenied || denied.IssuedSessionID != "" {
		t.Fatalf("DenyAuthRequest() = %#v, want denied without session", denied)
	}

	canceledReq := create("req-cancel", "user-cancel", 30*time.Minute)
	canceled, err := auth.CancelAuthRequest(ctx, canceledReq.ID, "approver-2", domain.ActorTypeUser, "superseded")
	if err != nil {
		t.Fatalf("CancelAuthRequest() error = %v", err)
	}
	if canceled.State != domain.AuthRequestStateCanceled {
		t.Fatalf("CancelAuthRequest() = %#v, want canceled", canceled)
	}

	expiringReq := create("req-expire", "user-expire", time.Millisecond)
	current = now.Add(10 * time.Millisecond)
	expired, err := auth.GetAuthRequest(ctx, expiringReq.ID)
	if err != nil {
		t.Fatalf("GetAuthRequest(expired) error = %v", err)
	}
	if expired.State != domain.AuthRequestStateExpired || expired.ResolutionNote != "timed_out" {
		t.Fatalf("GetAuthRequest(expired) = %#v, want expired timed_out", expired)
	}

	pendingList, err := auth.ListAuthRequests(ctx, domain.AuthRequestListFilter{ProjectID: "p1", State: domain.AuthRequestStatePending, Limit: 10})
	if err != nil {
		t.Fatalf("ListAuthRequests(pending) error = %v", err)
	}
	if len(pendingList) != 0 {
		t.Fatalf("expected no pending requests, got %#v", pendingList)
	}
}

// TestServiceAppFacingSessionWrappers verifies app-facing session lifecycle wrappers preserve decorated session details.
func TestServiceAppFacingSessionWrappers(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
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
	if err := auth.EnsureDogfoodPolicy(ctx); err != nil {
		t.Fatalf("EnsureDogfoodPolicy() error = %v", err)
	}

	issued, err := auth.IssueAuthSession(ctx, app.AuthSessionIssueInput{
		PrincipalID:   "agent-1",
		PrincipalType: "agent",
		PrincipalName: "Agent One",
		ClientID:      "till-mcp-stdio",
		ClientType:    "mcp-stdio",
		ClientName:    "Till MCP STDIO",
		TTL:           time.Hour,
	})
	if err != nil {
		t.Fatalf("IssueAuthSession() error = %v", err)
	}
	if issued.Session.SessionID == "" || issued.Secret == "" {
		t.Fatalf("IssueAuthSession() = %#v, want session + secret", issued)
	}

	listed, err := auth.ListAuthSessions(ctx, app.AuthSessionFilter{
		PrincipalID: "agent-1",
		State:       "active",
		Limit:       10,
	})
	if err != nil {
		t.Fatalf("ListAuthSessions() error = %v", err)
	}
	if len(listed) != 1 || listed[0].PrincipalName != "Agent One" || listed[0].ClientName != "Till MCP STDIO" {
		t.Fatalf("ListAuthSessions() = %#v, want decorated session row", listed)
	}

	validated, err := auth.ValidateAuthSession(ctx, issued.Session.SessionID, issued.Secret)
	if err != nil {
		t.Fatalf("ValidateAuthSession() error = %v", err)
	}
	if validated.Session.PrincipalType != "agent" || validated.Session.ClientType != "mcp-stdio" {
		t.Fatalf("ValidateAuthSession() = %#v, want agent/mcp-stdio", validated)
	}

	revoked, err := auth.RevokeAuthSession(ctx, issued.Session.SessionID, "operator_revoke")
	if err != nil {
		t.Fatalf("RevokeAuthSession() error = %v", err)
	}
	if revoked.SessionID != issued.Session.SessionID || revoked.RevocationReason != "operator_revoke" || revoked.RevokedAt == nil {
		t.Fatalf("RevokeAuthSession() = %#v, want revoked session details", revoked)
	}
}

// TestServiceListSessionsRejectsUnsupportedState verifies raw session inventory fails closed on unknown state filters.
func TestServiceListSessionsRejectsUnsupportedState(t *testing.T) {
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
	if _, err := auth.ListSessions(context.Background(), SessionListFilter{State: "weird"}); !errors.Is(err, domain.ErrInvalidAuthRequestState) {
		t.Fatalf("ListSessions() error = %v, want ErrInvalidAuthRequestState", err)
	}
}

// TestServiceListSessionsFiltersByProject verifies approved sessions can be narrowed by project id.
func TestServiceListSessionsFiltersByProject(t *testing.T) {
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
	issuedA, err := auth.IssueSession(context.Background(), IssueSessionInput{
		PrincipalID:   "agent-a",
		PrincipalType: "agent",
		ClientID:      "till-mcp-stdio",
		ClientType:    "mcp-stdio",
		Metadata: map[string]string{
			"project_id":      "p1",
			"approved_path":   "project/p1",
			"auth_request_id": "req-a",
		},
	})
	if err != nil {
		t.Fatalf("IssueSession(project p1) error = %v", err)
	}
	issuedB, err := auth.IssueSession(context.Background(), IssueSessionInput{
		PrincipalID:   "agent-b",
		PrincipalType: "agent",
		ClientID:      "till-mcp-stdio",
		ClientType:    "mcp-stdio",
		Metadata: map[string]string{
			"project_id":      "p2",
			"approved_path":   "project/p2",
			"auth_request_id": "req-b",
		},
	})
	if err != nil {
		t.Fatalf("IssueSession(project p2) error = %v", err)
	}

	filtered, err := auth.ListSessions(context.Background(), SessionListFilter{
		ProjectID: "p1",
		State:     string(autent.SessionStateActive),
	})
	if err != nil {
		t.Fatalf("ListSessions(project) error = %v", err)
	}
	if len(filtered) != 1 || filtered[0].ID != issuedA.Session.ID {
		t.Fatalf("ListSessions(project) = %#v, want only project p1 session %q", filtered, issuedA.Session.ID)
	}
	if got := filtered[0].Metadata["approved_path"]; got != "project/p1" {
		t.Fatalf("ListSessions(project) approved_path = %q, want project/p1", got)
	}

	unfiltered, err := auth.ListSessions(context.Background(), SessionListFilter{
		State: string(autent.SessionStateActive),
	})
	if err != nil {
		t.Fatalf("ListSessions(global) error = %v", err)
	}
	if len(unfiltered) < 2 {
		t.Fatalf("ListSessions(global) = %#v, want at least two active sessions", unfiltered)
	}
	if issuedB.Session.ID == "" {
		t.Fatal("expected second session to be issued")
	}
}

// TestServiceAuthRequestHelpers verifies approval bounds, context derivation, cloning, and session-state normalization helpers.
func TestServiceAuthRequestHelpers(t *testing.T) {
	t.Parallel()

	request := domain.AuthRequest{
		ID:                  "req-helper",
		Path:                "project/p1/branch/b1/phase/ph1",
		ProjectID:           "p1",
		BranchID:            "b1",
		PhaseIDs:            []string{"ph1"},
		ScopeType:           domain.ScopeLevelPhase,
		ScopeID:             "ph1",
		RequestedSessionTTL: 2 * time.Hour,
		Continuation:        map[string]any{"resume_tool": "tool", "resume": map[string]any{"path": "project/p1"}},
	}
	path, ttl, err := approvedAuthRequestValues(request, &domain.AuthRequestPath{ProjectID: "p1", BranchID: "b1", PhaseIDs: []string{"ph1"}}, time.Hour)
	if err != nil {
		t.Fatalf("approvedAuthRequestValues() error = %v", err)
	}
	if path.String() != "project/p1/branch/b1/phase/ph1" || ttl != time.Hour {
		t.Fatalf("approvedAuthRequestValues() = %s / %s, want project/p1/branch/b1/phase/ph1 / 1h", path.String(), ttl)
	}
	if _, _, err := approvedAuthRequestValues(request, &domain.AuthRequestPath{ProjectID: "p2"}, time.Hour); !errors.Is(err, domain.ErrInvalidAuthRequestPath) {
		t.Fatalf("approvedAuthRequestValues() error = %v, want ErrInvalidAuthRequestPath", err)
	}
	if _, _, err := approvedAuthRequestValues(request, nil, 3*time.Hour); !errors.Is(err, domain.ErrInvalidAuthRequestTTL) {
		t.Fatalf("approvedAuthRequestValues() error = %v, want ErrInvalidAuthRequestTTL", err)
	}
	if !authRequestPathWithin(domain.AuthRequestPath{ProjectID: "p1", BranchID: "b1", PhaseIDs: []string{"ph1"}}, domain.AuthRequestPath{ProjectID: "p1", BranchID: "b1", PhaseIDs: []string{"ph1", "ph2"}}) {
		t.Fatal("authRequestPathWithin() = false, want true for narrower candidate")
	}
	if authRequestPathWithin(domain.AuthRequestPath{ProjectID: "p1", BranchID: "b1"}, domain.AuthRequestPath{ProjectID: "p1", BranchID: "b2"}) {
		t.Fatal("authRequestPathWithin() = true, want false for different branch")
	}

	ctxPath, err := authContextPath(map[string]string{
		"project_id": "p1",
		"scope_type": string(domain.ScopeLevelTask),
		"scope_id":   "task-1",
		"branch_id":  "b1",
		"phase_path": "ph-a/ph-b",
	})
	if err != nil {
		t.Fatalf("authContextPath() error = %v", err)
	}
	if got := ctxPath.String(); got != "project/p1/branch/b1/phase/ph-a/phase/ph-b" {
		t.Fatalf("authContextPath() = %q, want project/p1/branch/b1/phase/ph-a/phase/ph-b", got)
	}
	if phases := authContextPhaseIDs(map[string]string{"phase_id": "ph-single"}); len(phases) != 1 || phases[0] != "ph-single" {
		t.Fatalf("authContextPhaseIDs() = %#v, want single phase", phases)
	}
	if state, err := normalizeSessionStateFilter(" revoked "); err != nil || state != autent.SessionStateRevoked {
		t.Fatalf("normalizeSessionStateFilter() = %q, %v, want revoked nil", state, err)
	}
	if _, err := normalizeSessionStateFilter("unknown"); err == nil {
		t.Fatal("normalizeSessionStateFilter() error = nil, want error")
	}

	clone := cloneAuthRequest(request)
	cloneResume, ok := clone.Continuation["resume"].(map[string]any)
	if !ok {
		t.Fatalf("cloneAuthRequest() resume = %#v, want nested map", clone.Continuation["resume"])
	}
	cloneResume["path"] = "edited"
	originalResume, ok := request.Continuation["resume"].(map[string]any)
	if !ok {
		t.Fatalf("request continuation resume = %#v, want nested map", request.Continuation["resume"])
	}
	if originalResume["path"] == "edited" {
		t.Fatal("cloneAuthRequest() did not deep-copy continuation map")
	}
}

// TestSessionViewAndPrincipalHelpers verifies view mapping and principal-type helpers stay deterministic.
func TestSessionViewAndPrincipalHelpers(t *testing.T) {
	t.Parallel()

	lastSeen := time.Date(2026, 3, 20, 15, 0, 0, 0, time.UTC)
	revokedAt := time.Date(2026, 3, 20, 16, 0, 0, 0, time.UTC)
	session := mapSessionView(autent.SessionView{
		ID:               "sess-1",
		PrincipalID:      "agent-1",
		ClientID:         "till-mcp-stdio",
		IssuedAt:         time.Date(2026, 3, 20, 12, 0, 0, 0, time.UTC),
		ExpiresAt:        time.Date(2026, 3, 20, 20, 0, 0, 0, time.UTC),
		LastSeenAt:       lastSeen,
		RevokedAt:        &revokedAt,
		RevocationReason: "operator_revoke",
	}, "agent", "Agent One", "mcp-stdio", "Till MCP STDIO")
	if session.SessionID != "sess-1" || session.PrincipalType != "agent" || session.ClientType != "mcp-stdio" {
		t.Fatalf("mapSessionView() = %#v, want preserved principal/client identity", session)
	}
	if session.LastValidatedAt == nil || !session.LastValidatedAt.Equal(lastSeen.UTC()) {
		t.Fatalf("mapSessionView() last_validated_at = %#v, want %s", session.LastValidatedAt, lastSeen.UTC())
	}
	if session.RevokedAt == nil || !session.RevokedAt.Equal(revokedAt.UTC()) {
		t.Fatalf("mapSessionView() revoked_at = %#v, want %s", session.RevokedAt, revokedAt.UTC())
	}

	if got := normalizePrincipalType("SeRvIcE"); got != autentdomain.PrincipalTypeService {
		t.Fatalf("normalizePrincipalType(service) = %q, want service", got)
	}
	if got := principalTypeToActorType(autentdomain.PrincipalTypeAgent); got != domain.ActorTypeAgent {
		t.Fatalf("principalTypeToActorType(agent) = %q, want agent", got)
	}
	if got := firstNonEmpty("", "  ", "agent-1", "fallback"); got != "agent-1" {
		t.Fatalf("firstNonEmpty() = %q, want agent-1", got)
	}
}
