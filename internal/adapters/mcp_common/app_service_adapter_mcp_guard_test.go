package common

import (
	"context"
	"errors"
	"testing"

	"github.com/evanmschultz/tillsyn/internal/app"
	"github.com/evanmschultz/tillsyn/internal/domain"
)

// TestWithMutationGuardContext validates actor-type and lease tuple normalization/guarding.
func TestWithMutationGuardContext(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		actor           ActorLeaseTuple
		wantActorType   domain.ActorType
		wantGuard       bool
		wantInvalidErr  bool
		wantAgentName   string
		wantInstanceID  string
		wantLeaseToken  string
		wantOverrideTok string
	}{
		{
			name: "plain user mutation keeps user actor without guard",
			actor: ActorLeaseTuple{
				ActorType: string(domain.ActorTypeUser),
			},
			wantActorType: domain.ActorTypeUser,
			wantGuard:     false,
		},
		{
			name: "user actor may provide agent_name attribution without guard tuple",
			actor: ActorLeaseTuple{
				ActorType: string(domain.ActorTypeUser),
				AgentName: "evan",
			},
			wantActorType: domain.ActorTypeUser,
			wantGuard:     false,
		},
		{
			name: "explicit user actor with guard tuple is rejected",
			actor: ActorLeaseTuple{
				ActorType:       string(domain.ActorTypeUser),
				AgentName:       "agent-a",
				AgentInstanceID: "agent-a-1",
				LeaseToken:      "lease-a",
			},
			wantInvalidErr: true,
		},
		{
			name: "omitted actor type with guard tuple is rejected",
			actor: ActorLeaseTuple{
				AgentName:       "agent-b",
				AgentInstanceID: "agent-b-1",
				LeaseToken:      "lease-b",
				OverrideToken:   "override-b",
			},
			wantInvalidErr: true,
		},
		{
			name: "explicit agent actor with guard tuple is accepted",
			actor: ActorLeaseTuple{
				ActorType:       string(domain.ActorTypeAgent),
				AgentName:       "agent-c",
				AgentInstanceID: "agent-c-1",
				LeaseToken:      "lease-c",
			},
			wantActorType:  domain.ActorTypeAgent,
			wantGuard:      true,
			wantAgentName:  "agent-c",
			wantInstanceID: "agent-c-1",
			wantLeaseToken: "lease-c",
		},
		{
			name: "non-user actor without lease tuple is rejected",
			actor: ActorLeaseTuple{
				ActorType: string(domain.ActorTypeAgent),
			},
			wantInvalidErr: true,
		},
		{
			name: "system actor type is rejected",
			actor: ActorLeaseTuple{
				ActorType: string(domain.ActorTypeSystem),
			},
			wantInvalidErr: true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			gotCtx, gotActorType, err := withMutationGuardContext(context.Background(), tc.actor)
			if tc.wantInvalidErr {
				if err == nil {
					t.Fatal("withMutationGuardContext() expected error")
				}
				if !errors.Is(err, ErrInvalidCaptureStateRequest) {
					t.Fatalf("withMutationGuardContext() expected ErrInvalidCaptureStateRequest, got %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("withMutationGuardContext() unexpected error: %v", err)
			}
			if gotActorType != tc.wantActorType {
				t.Fatalf("withMutationGuardContext() actor type = %q, want %q", gotActorType, tc.wantActorType)
			}
			guard, hasGuard := app.MutationGuardFromContext(gotCtx)
			if hasGuard != tc.wantGuard {
				t.Fatalf("withMutationGuardContext() guard presence = %t, want %t", hasGuard, tc.wantGuard)
			}
			if !tc.wantGuard {
				return
			}
			if guard.AgentName != tc.wantAgentName {
				t.Fatalf("MutationGuard.AgentName = %q, want %q", guard.AgentName, tc.wantAgentName)
			}
			if guard.AgentInstanceID != tc.wantInstanceID {
				t.Fatalf("MutationGuard.AgentInstanceID = %q, want %q", guard.AgentInstanceID, tc.wantInstanceID)
			}
			if guard.LeaseToken != tc.wantLeaseToken {
				t.Fatalf("MutationGuard.LeaseToken = %q, want %q", guard.LeaseToken, tc.wantLeaseToken)
			}
			if guard.OverrideToken != tc.wantOverrideTok {
				t.Fatalf("MutationGuard.OverrideToken = %q, want %q", guard.OverrideToken, tc.wantOverrideTok)
			}
		})
	}
}

// TestWithMutationGuardContextAllowUnguardedAgent validates the narrow project-create
// exception for approved agent sessions without a lease tuple.
func TestWithMutationGuardContextAllowUnguardedAgent(t *testing.T) {
	t.Parallel()

	gotCtx, gotActorType, err := withMutationGuardContextAllowUnguardedAgent(context.Background(), ActorLeaseTuple{
		ActorID:   "agent-1",
		ActorName: "Agent One",
		ActorType: string(domain.ActorTypeAgent),
	}, true)
	if err != nil {
		t.Fatalf("withMutationGuardContextAllowUnguardedAgent() unexpected error: %v", err)
	}
	if gotActorType != domain.ActorTypeAgent {
		t.Fatalf("withMutationGuardContextAllowUnguardedAgent() actor type = %q, want %q", gotActorType, domain.ActorTypeAgent)
	}
	if _, hasGuard := app.MutationGuardFromContext(gotCtx); hasGuard {
		t.Fatal("withMutationGuardContextAllowUnguardedAgent() unexpectedly attached a mutation guard")
	}
	caller, ok := app.AuthenticatedCallerFromContext(gotCtx)
	if !ok {
		t.Fatal("withMutationGuardContextAllowUnguardedAgent() missing authenticated caller")
	}
	if caller.PrincipalID != "agent-1" {
		t.Fatalf("authenticated caller principal_id = %q, want agent-1", caller.PrincipalID)
	}
}
