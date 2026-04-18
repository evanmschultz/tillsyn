package app

import (
	"context"
	"strings"

	"github.com/evanmschultz/tillsyn/internal/domain"
)

// mutationScopeCandidate stores one normalized capability-scope tuple.
type mutationScopeCandidate struct {
	ScopeType domain.CapabilityScopeType
	ScopeID   string
}

// newProjectMutationScopeCandidate returns one normalized project-scope tuple.
func newProjectMutationScopeCandidate(projectID string) mutationScopeCandidate {
	return mutationScopeCandidate{
		ScopeType: domain.CapabilityScopeProject,
		ScopeID:   strings.TrimSpace(projectID),
	}
}

// capabilityScopesForActionItemLineage resolves guardrail scope candidates for one actionItem lineage.
func (s *Service) capabilityScopesForActionItemLineage(ctx context.Context, actionItem domain.ActionItem) ([]mutationScopeCandidate, error) {
	projectID := strings.TrimSpace(actionItem.ProjectID)
	if projectID == "" {
		return nil, domain.ErrInvalidID
	}

	scopes := make([]mutationScopeCandidate, 0, 6)
	scopes = appendMutationScopeCandidate(scopes, newProjectMutationScopeCandidate(projectID))
	lineage, err := s.actionItemLineage(ctx, actionItem)
	if err != nil {
		return nil, err
	}
	for _, current := range lineage {
		scope := mutationScopeCandidate{
			ScopeType: capabilityScopeTypeForActionItem(current),
			ScopeID:   strings.TrimSpace(current.ID),
		}
		if scope.ScopeType == domain.CapabilityScopeProject {
			scope.ScopeID = projectID
		}
		scopes = appendMutationScopeCandidate(scopes, scope)
	}
	return scopes, nil
}

// capabilityScopesForLease resolves the scope candidates a lease request should inherit or match.
func (s *Service) capabilityScopesForLease(ctx context.Context, projectID string, scopeType domain.CapabilityScopeType, scopeID string) ([]mutationScopeCandidate, error) {
	projectID = strings.TrimSpace(projectID)
	scopeType = domain.NormalizeCapabilityScopeType(scopeType)
	scopeID = strings.TrimSpace(scopeID)
	if projectID == "" {
		return nil, domain.ErrInvalidID
	}
	if !domain.IsValidCapabilityScopeType(scopeType) {
		return nil, domain.ErrInvalidCapabilityScope
	}
	if scopeType == domain.CapabilityScopeProject {
		return []mutationScopeCandidate{newProjectMutationScopeCandidate(projectID)}, nil
	}
	if scopeID == "" {
		return nil, domain.ErrInvalidCapabilityScope
	}
	actionItem, err := s.repo.GetActionItem(ctx, scopeID)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(actionItem.ProjectID) != projectID {
		return nil, ErrNotFound
	}
	if capabilityScopeTypeForActionItem(actionItem) != scopeType {
		return nil, domain.ErrInvalidCapabilityScope
	}
	return s.capabilityScopesForActionItemLineage(ctx, actionItem)
}

// appendMutationScopeCandidate adds one scope candidate only when valid and unique.
func appendMutationScopeCandidate(scopes []mutationScopeCandidate, candidate mutationScopeCandidate) []mutationScopeCandidate {
	candidate.ScopeType = domain.NormalizeCapabilityScopeType(candidate.ScopeType)
	candidate.ScopeID = strings.TrimSpace(candidate.ScopeID)
	if !domain.IsValidCapabilityScopeType(candidate.ScopeType) {
		return scopes
	}
	if candidate.ScopeType != domain.CapabilityScopeProject && candidate.ScopeID == "" {
		return scopes
	}
	if candidate.ScopeType == domain.CapabilityScopeProject && candidate.ScopeID == "" {
		return scopes
	}
	for _, existing := range scopes {
		if existing.ScopeType == candidate.ScopeType && existing.ScopeID == candidate.ScopeID {
			return scopes
		}
	}
	return append(scopes, candidate)
}
