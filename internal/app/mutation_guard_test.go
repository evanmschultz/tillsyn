package app

import (
	"context"
	"testing"

	"github.com/hylla/tillsyn/internal/domain"
)

// TestMutationGuardContextRoundTrip verifies normalization and retrieval from context.
func TestMutationGuardContextRoundTrip(t *testing.T) {
	ctx := WithMutationGuard(context.Background(), MutationGuard{
		AgentName:       " orchestrator ",
		AgentInstanceID: " inst-1 ",
		LeaseToken:      " lease-1 ",
		OverrideToken:   " override ",
	})
	guard, ok := MutationGuardFromContext(ctx)
	if !ok {
		t.Fatal("MutationGuardFromContext() expected guard")
	}
	if guard.AgentName != "orchestrator" {
		t.Fatalf("AgentName = %q, want orchestrator", guard.AgentName)
	}
	if guard.AgentInstanceID != "inst-1" {
		t.Fatalf("AgentInstanceID = %q, want inst-1", guard.AgentInstanceID)
	}
	if guard.LeaseToken != "lease-1" {
		t.Fatalf("LeaseToken = %q, want lease-1", guard.LeaseToken)
	}
	if guard.OverrideToken != "override" {
		t.Fatalf("OverrideToken = %q, want override", guard.OverrideToken)
	}
}

// TestMutationGuardContextEmptyAndRequired verifies absence and required-flag semantics.
func TestMutationGuardContextEmptyAndRequired(t *testing.T) {
	if _, ok := MutationGuardFromContext(context.Background()); ok {
		t.Fatal("MutationGuardFromContext() expected no guard for empty context")
	}
	empty := WithMutationGuard(context.Background(), MutationGuard{})
	if _, ok := MutationGuardFromContext(empty); ok {
		t.Fatal("MutationGuardFromContext() expected no guard for empty value")
	}
	if MutationGuardRequired(context.Background()) {
		t.Fatal("MutationGuardRequired() expected false by default")
	}
	required := WithMutationGuardRequired(context.Background())
	if !MutationGuardRequired(required) {
		t.Fatal("MutationGuardRequired() expected true after marker")
	}
}

// TestAuthenticatedCallerContextRoundTrip verifies normalized caller identity round-trips through context.
func TestAuthenticatedCallerContextRoundTrip(t *testing.T) {
	ctx := WithAuthenticatedCaller(context.Background(), domain.AuthenticatedCaller{
		PrincipalID:   " user-1 ",
		PrincipalName: " Evan Schultz ",
		PrincipalType: domain.ActorTypeUser,
		SessionID:     " session-1 ",
	})
	caller, ok := AuthenticatedCallerFromContext(ctx)
	if !ok {
		t.Fatal("AuthenticatedCallerFromContext() expected caller")
	}
	if caller.PrincipalID != "user-1" {
		t.Fatalf("PrincipalID = %q, want user-1", caller.PrincipalID)
	}
	if caller.PrincipalName != "Evan Schultz" {
		t.Fatalf("PrincipalName = %q, want Evan Schultz", caller.PrincipalName)
	}
	if caller.PrincipalType != domain.ActorTypeUser {
		t.Fatalf("PrincipalType = %q, want %q", caller.PrincipalType, domain.ActorTypeUser)
	}
	if caller.SessionID != "session-1" {
		t.Fatalf("SessionID = %q, want session-1", caller.SessionID)
	}
}

// TestMutationActorFromContextFallsBackToAuthenticatedCaller verifies caller identity can drive attribution.
func TestMutationActorFromContextFallsBackToAuthenticatedCaller(t *testing.T) {
	ctx := WithAuthenticatedCaller(context.Background(), domain.AuthenticatedCaller{
		PrincipalID:   "agent-1",
		PrincipalName: "Planner Bot",
		PrincipalType: domain.ActorTypeAgent,
		SessionID:     "session-1",
	})
	actor, ok := MutationActorFromContext(ctx)
	if !ok {
		t.Fatal("MutationActorFromContext() expected derived actor")
	}
	if actor.ActorID != "agent-1" {
		t.Fatalf("ActorID = %q, want agent-1", actor.ActorID)
	}
	if actor.ActorName != "Planner Bot" {
		t.Fatalf("ActorName = %q, want Planner Bot", actor.ActorName)
	}
	if actor.ActorType != domain.ActorTypeAgent {
		t.Fatalf("ActorType = %q, want %q", actor.ActorType, domain.ActorTypeAgent)
	}
}
