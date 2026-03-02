package app

import (
	"context"
	"strings"

	"github.com/hylla/tillsyn/internal/domain"
)

// MutationGuard carries the capability-lease tuple required for agent mutations.
type MutationGuard struct {
	AgentName       string
	AgentInstanceID string
	LeaseToken      string
	OverrideToken   string
}

// MutationActor carries normalized caller identity metadata for mutation attribution.
type MutationActor struct {
	ActorID   string
	ActorName string
	ActorType domain.ActorType
}

// WithMutationGuard attaches a normalized mutation guard to context.
func WithMutationGuard(ctx context.Context, guard MutationGuard) context.Context {
	guard = normalizeMutationGuard(guard)
	return context.WithValue(ctx, mutationGuardContextKey{}, guard)
}

// MutationGuardFromContext returns a normalized mutation guard when present.
func MutationGuardFromContext(ctx context.Context) (MutationGuard, bool) {
	raw := ctx.Value(mutationGuardContextKey{})
	guard, ok := raw.(MutationGuard)
	if !ok {
		return MutationGuard{}, false
	}
	guard = normalizeMutationGuard(guard)
	if guard.AgentName == "" && guard.AgentInstanceID == "" && guard.LeaseToken == "" && guard.OverrideToken == "" {
		return MutationGuard{}, false
	}
	return guard, true
}

// mutationGuardContextKey stores context keys for mutation guard values.
type mutationGuardContextKey struct{}

// WithMutationActor attaches normalized mutation-actor identity metadata to context.
func WithMutationActor(ctx context.Context, actor MutationActor) context.Context {
	actor = normalizeMutationActor(actor)
	return context.WithValue(ctx, mutationActorContextKey{}, actor)
}

// MutationActorFromContext returns normalized mutation-actor metadata when present.
func MutationActorFromContext(ctx context.Context) (MutationActor, bool) {
	raw := ctx.Value(mutationActorContextKey{})
	actor, ok := raw.(MutationActor)
	if !ok {
		return MutationActor{}, false
	}
	actor = normalizeMutationActor(actor)
	if actor.ActorID == "" {
		return MutationActor{}, false
	}
	return actor, true
}

// mutationActorContextKey stores context keys for mutation actor metadata.
type mutationActorContextKey struct{}

// WithMutationGuardRequired marks a context as requiring guard validation for non-user actors.
func WithMutationGuardRequired(ctx context.Context) context.Context {
	return context.WithValue(ctx, mutationGuardRequiredContextKey{}, true)
}

// MutationGuardRequired reports whether guard enforcement was explicitly requested.
func MutationGuardRequired(ctx context.Context) bool {
	raw := ctx.Value(mutationGuardRequiredContextKey{})
	required, ok := raw.(bool)
	return ok && required
}

// mutationGuardRequiredContextKey stores context keys for strict-guard enforcement flags.
type mutationGuardRequiredContextKey struct{}

// normalizeMutationGuard trims and canonicalizes guard fields.
func normalizeMutationGuard(guard MutationGuard) MutationGuard {
	guard.AgentName = strings.TrimSpace(guard.AgentName)
	guard.AgentInstanceID = strings.TrimSpace(guard.AgentInstanceID)
	guard.LeaseToken = strings.TrimSpace(guard.LeaseToken)
	guard.OverrideToken = strings.TrimSpace(guard.OverrideToken)
	return guard
}

// normalizeMutationActor trims and canonicalizes mutation actor metadata.
func normalizeMutationActor(actor MutationActor) MutationActor {
	actor.ActorID = strings.TrimSpace(actor.ActorID)
	actor.ActorName = strings.TrimSpace(actor.ActorName)
	actor.ActorType = domain.ActorType(strings.TrimSpace(strings.ToLower(string(actor.ActorType))))
	switch actor.ActorType {
	case domain.ActorTypeUser, domain.ActorTypeAgent, domain.ActorTypeSystem:
	default:
		actor.ActorType = domain.ActorTypeUser
	}
	return actor
}
