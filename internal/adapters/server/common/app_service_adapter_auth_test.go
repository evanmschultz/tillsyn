//go:build commonhash

package common

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	autentdomain "github.com/evanmschultz/autent/domain"
	"github.com/hylla/tillsyn/internal/adapters/auth/autentauth"
	"github.com/hylla/tillsyn/internal/adapters/storage/sqlite"
	"github.com/hylla/tillsyn/internal/app"
	"github.com/hylla/tillsyn/internal/domain"
)

// newAuthOnlyAdapterForTest constructs one adapter with a shared-DB autent service and no app service.
func newAuthOnlyAdapterForTest(t *testing.T) (*AppServiceAdapter, *autentauth.Service) {
	t.Helper()

	repo, err := sqlite.Open(filepath.Join(t.TempDir(), "tillsyn.db"))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(func() {
		_ = repo.Close()
	})

	auth, err := autentauth.NewSharedDB(autentauth.Config{DB: repo.DB()})
	if err != nil {
		t.Fatalf("NewSharedDB() error = %v", err)
	}
	return NewAppServiceAdapter(nil, auth), auth
}

// mustIssueUserSessionForTest issues one deterministic user session for adapter auth tests.
func mustIssueUserSessionForTest(t *testing.T, auth *autentauth.Service) (string, string) {
	t.Helper()

	issued, err := auth.IssueSession(context.Background(), autentauth.IssueSessionInput{
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
	return issued.Session.ID, issued.Secret
}

// mustReplaceAuthRulesForTest replaces the persisted auth rules with one validated rule set.
func mustReplaceAuthRulesForTest(t *testing.T, auth *autentauth.Service, rules ...autentdomain.Rule) {
	t.Helper()

	if err := auth.ReplaceRules(context.Background(), rules); err != nil {
		t.Fatalf("ReplaceRules() error = %v", err)
	}
}

// mustNormalizeRuleForTest validates one auth rule for stable adapter tests.
func mustNormalizeRuleForTest(t *testing.T, rule autentdomain.Rule) autentdomain.Rule {
	t.Helper()

	normalized, err := autentdomain.ValidateAndNormalizeRule(rule)
	if err != nil {
		t.Fatalf("ValidateAndNormalizeRule() error = %v", err)
	}
	return normalized
}

// TestAppServiceAdapterAuthorizeMutationRevokedSession verifies revoked sessions map to invalid authentication.
func TestAppServiceAdapterAuthorizeMutationRevokedSession(t *testing.T) {
	t.Parallel()

	adapter, auth := newAuthOnlyAdapterForTest(t)
	if err := auth.EnsureDogfoodPolicy(context.Background()); err != nil {
		t.Fatalf("EnsureDogfoodPolicy() error = %v", err)
	}
	sessionID, sessionSecret := mustIssueUserSessionForTest(t, auth)
	if _, err := auth.RevokeSession(context.Background(), sessionID, "operator_revoke"); err != nil {
		t.Fatalf("RevokeSession() error = %v", err)
	}

	_, err := adapter.AuthorizeMutation(context.Background(), MutationAuthorizationRequest{
		SessionID:     sessionID,
		SessionSecret: sessionSecret,
		Action:        "create_task",
		Namespace:     "project:p1",
		ResourceType:  "task",
		ResourceID:    "new",
	})
	if !errors.Is(err, ErrInvalidAuthentication) {
		t.Fatalf("AuthorizeMutation() error = %v, want ErrInvalidAuthentication", err)
	}
}

// TestAppServiceAdapterAuthorizeMutationDeniedByRule verifies real deny decisions map to authorization denied.
func TestAppServiceAdapterAuthorizeMutationDeniedByRule(t *testing.T) {
	t.Parallel()

	adapter, auth := newAuthOnlyAdapterForTest(t)
	mustReplaceAuthRulesForTest(t, auth, mustNormalizeRuleForTest(t, autentdomain.Rule{
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
	}))
	sessionID, sessionSecret := mustIssueUserSessionForTest(t, auth)

	_, err := adapter.AuthorizeMutation(context.Background(), MutationAuthorizationRequest{
		SessionID:     sessionID,
		SessionSecret: sessionSecret,
		Action:        "create_task",
		Namespace:     "project:p1",
		ResourceType:  "task",
		ResourceID:    "new",
	})
	if !errors.Is(err, ErrAuthorizationDenied) {
		t.Fatalf("AuthorizeMutation() error = %v, want ErrAuthorizationDenied", err)
	}
}

// TestAppServiceAdapterAuthorizeMutationGrantRequired verifies real grant-required decisions map to the grant sentinel.
func TestAppServiceAdapterAuthorizeMutationGrantRequired(t *testing.T) {
	t.Parallel()

	adapter, auth := newAuthOnlyAdapterForTest(t)
	mustReplaceAuthRulesForTest(t, auth, mustNormalizeRuleForTest(t, autentdomain.Rule{
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
	}))
	sessionID, sessionSecret := mustIssueUserSessionForTest(t, auth)

	_, err := adapter.AuthorizeMutation(context.Background(), MutationAuthorizationRequest{
		SessionID:     sessionID,
		SessionSecret: sessionSecret,
		Action:        "create_task",
		Namespace:     "project:p1",
		ResourceType:  "task",
		ResourceID:    "new",
	})
	if !errors.Is(err, ErrGrantRequired) {
		t.Fatalf("AuthorizeMutation() error = %v, want ErrGrantRequired", err)
	}
}

// TestWithMutationGuardContextRejectsGuardTupleWithoutExplicitAgent verifies the old implicit-agent fallback is gone.
func TestWithMutationGuardContextRejectsGuardTupleWithoutExplicitAgent(t *testing.T) {
	t.Parallel()

	_, _, err := withMutationGuardContext(context.Background(), ActorLeaseTuple{
		AgentName:       "agent-1",
		AgentInstanceID: "agent-1-instance",
		LeaseToken:      "lease-1",
	})
	if !errors.Is(err, ErrInvalidCaptureStateRequest) {
		t.Fatalf("withMutationGuardContext() error = %v, want ErrInvalidCaptureStateRequest", err)
	}
}

// TestWithMutationGuardContextAcceptsExplicitAgentGuard verifies explicit agent tuples still install the local guard.
func TestWithMutationGuardContextAcceptsExplicitAgentGuard(t *testing.T) {
	t.Parallel()

	ctx, actorType, err := withMutationGuardContext(context.Background(), ActorLeaseTuple{
		ActorType:       string(domain.ActorTypeAgent),
		AgentName:       "agent-1",
		AgentInstanceID: "agent-1-instance",
		LeaseToken:      "lease-1",
		OverrideToken:   "override-1",
	})
	if err != nil {
		t.Fatalf("withMutationGuardContext() error = %v", err)
	}
	if actorType != domain.ActorTypeAgent {
		t.Fatalf("actorType = %q, want %q", actorType, domain.ActorTypeAgent)
	}
	guard, ok := app.MutationGuardFromContext(ctx)
	if !ok {
		t.Fatal("MutationGuardFromContext() guard missing, want present")
	}
	if guard.AgentInstanceID != "agent-1-instance" {
		t.Fatalf("guard.AgentInstanceID = %q, want agent-1-instance", guard.AgentInstanceID)
	}
	if guard.LeaseToken != "lease-1" {
		t.Fatalf("guard.LeaseToken = %q, want lease-1", guard.LeaseToken)
	}
	if guard.OverrideToken != "override-1" {
		t.Fatalf("guard.OverrideToken = %q, want override-1", guard.OverrideToken)
	}
}
