package app

import (
	"context"
	"fmt"
	"strings"

	"github.com/evanmschultz/tillsyn/internal/domain"
)

// currentMutationActorType returns the actor type that should drive guard enforcement for this request.
func currentMutationActorType(ctx context.Context, explicit domain.ActorType) domain.ActorType {
	if actor, ok := MutationActorFromContext(ctx); ok {
		return normalizeActorTypeInput(actor.ActorType)
	}
	if strings.TrimSpace(string(explicit)) != "" {
		return normalizeActorTypeInput(explicit)
	}
	return domain.ActorTypeUser
}

// ensureActionItemCompletionBlockersClear enforces parent completion criteria against active (non-archived) children.
func (s *Service) ensureActionItemCompletionBlockersClear(ctx context.Context, actionItem domain.ActionItem, projectActionItems []domain.ActionItem) error {
	_ = ctx
	activeChildren := make([]domain.ActionItem, 0)
	for _, child := range projectActionItems {
		if strings.TrimSpace(child.ParentID) != strings.TrimSpace(actionItem.ID) {
			continue
		}
		if child.ArchivedAt != nil {
			continue
		}
		activeChildren = append(activeChildren, child)
	}
	if unmet := actionItem.CompletionCriteriaUnmet(activeChildren); len(unmet) > 0 {
		return fmt.Errorf("%w: completion criteria unmet (%s)", domain.ErrTransitionBlocked, strings.Join(unmet, ", "))
	}
	return nil
}

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

// WithAuthenticatedCaller attaches a normalized authenticated caller to context.
func WithAuthenticatedCaller(ctx context.Context, caller domain.AuthenticatedCaller) context.Context {
	caller = domain.NormalizeAuthenticatedCaller(caller)
	return context.WithValue(ctx, authenticatedCallerContextKey{}, caller)
}

// AuthenticatedCallerFromContext returns normalized authenticated caller metadata when present.
func AuthenticatedCallerFromContext(ctx context.Context) (domain.AuthenticatedCaller, bool) {
	raw := ctx.Value(authenticatedCallerContextKey{})
	caller, ok := raw.(domain.AuthenticatedCaller)
	if !ok {
		return domain.AuthenticatedCaller{}, false
	}
	caller = domain.NormalizeAuthenticatedCaller(caller)
	if caller.IsZero() {
		return domain.AuthenticatedCaller{}, false
	}
	return caller, true
}

// authenticatedCallerContextKey stores context keys for authenticated caller metadata.
type authenticatedCallerContextKey struct{}

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
	ctx = context.WithValue(ctx, mutationActorContextKey{}, actor)
	if caller := mutationActorToAuthenticatedCaller(actor); !caller.IsZero() {
		ctx = WithAuthenticatedCaller(ctx, caller)
	}
	return ctx
}

// MutationActorFromContext returns normalized mutation-actor metadata when present.
func MutationActorFromContext(ctx context.Context) (MutationActor, bool) {
	raw := ctx.Value(mutationActorContextKey{})
	actor, ok := raw.(MutationActor)
	if ok {
		actor = normalizeMutationActor(actor)
		if actor.ActorID != "" {
			return actor, true
		}
	}
	caller, ok := AuthenticatedCallerFromContext(ctx)
	if !ok {
		return MutationActor{}, false
	}
	actor = authenticatedCallerToMutationActor(caller)
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

// mutationActorToAuthenticatedCaller converts persisted mutation attribution into caller identity form.
func mutationActorToAuthenticatedCaller(actor MutationActor) domain.AuthenticatedCaller {
	actor = normalizeMutationActor(actor)
	return domain.NormalizeAuthenticatedCaller(domain.AuthenticatedCaller{
		PrincipalID:   actor.ActorID,
		PrincipalName: actor.ActorName,
		PrincipalType: actor.ActorType,
	})
}

// authenticatedCallerToMutationActor converts caller identity into mutation attribution form.
func authenticatedCallerToMutationActor(caller domain.AuthenticatedCaller) MutationActor {
	caller = domain.NormalizeAuthenticatedCaller(caller)
	return normalizeMutationActor(MutationActor{
		ActorID:   caller.PrincipalID,
		ActorName: caller.PrincipalName,
		ActorType: caller.PrincipalType,
	})
}
