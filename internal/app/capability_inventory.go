package app

import (
	"context"
	"strings"

	"github.com/evanmschultz/tillsyn/internal/domain"
)

// ListCapabilityLeasesInput captures scope query values for listing capability leases.
type ListCapabilityLeasesInput struct {
	ProjectID      string
	ScopeType      domain.CapabilityScopeType
	ScopeID        string
	IncludeRevoked bool
}

// ListCapabilityLeases lists scoped capability leases in deterministic repository order.
func (s *Service) ListCapabilityLeases(ctx context.Context, in ListCapabilityLeasesInput) ([]domain.CapabilityLease, error) {
	projectID := strings.TrimSpace(in.ProjectID)
	if projectID == "" {
		return nil, domain.ErrInvalidID
	}
	if _, err := s.repo.GetProject(ctx, projectID); err != nil {
		return nil, err
	}

	scopeType := domain.NormalizeCapabilityScopeType(in.ScopeType)
	if scopeType == "" {
		scopeType = domain.CapabilityScopeProject
	}
	if !domain.IsValidCapabilityScopeType(scopeType) {
		return nil, domain.ErrInvalidCapabilityScope
	}

	scopeID := strings.TrimSpace(in.ScopeID)
	if scopeID != "" {
		validatedScopeID, err := s.validateCapabilityScopeTuple(ctx, projectID, scopeType, scopeID)
		if err != nil {
			return nil, err
		}
		scopeID = validatedScopeID
	}

	leases, err := s.repo.ListCapabilityLeasesByScope(ctx, projectID, scopeType, scopeID)
	if err != nil {
		return nil, err
	}
	if in.IncludeRevoked {
		return leases, nil
	}

	active := make([]domain.CapabilityLease, 0, len(leases))
	for _, lease := range leases {
		if lease.RevokedAt != nil {
			continue
		}
		active = append(active, lease)
	}
	return active, nil
}
