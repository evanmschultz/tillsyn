package app

import (
	"context"
	"fmt"
	"strings"

	"github.com/evanmschultz/tillsyn/internal/domain"
)

// AuthScopeContext stores the narrowest project-rooted auth path one resource proves.
type AuthScopeContext struct {
	ProjectID string
	BranchID  string
	ScopeType domain.ScopeLevel
	ScopeID   string
	PhaseIDs  []string
}

// GetActionItem returns one work-item row by id.
func (s *Service) GetActionItem(ctx context.Context, actionItemID string) (domain.ActionItem, error) {
	if s == nil || s.repo == nil {
		return domain.ActionItem{}, fmt.Errorf("service is not configured")
	}
	actionItemID = strings.TrimSpace(actionItemID)
	if actionItemID == "" {
		return domain.ActionItem{}, domain.ErrInvalidID
	}
	return s.repo.GetActionItem(ctx, actionItemID)
}

// GetAttentionItem returns one attention item by id.
func (s *Service) GetAttentionItem(ctx context.Context, attentionID string) (domain.AttentionItem, error) {
	if s == nil || s.repo == nil {
		return domain.AttentionItem{}, fmt.Errorf("service is not configured")
	}
	attentionID = strings.TrimSpace(attentionID)
	if attentionID == "" {
		return domain.AttentionItem{}, domain.ErrInvalidID
	}
	return s.repo.GetAttentionItem(ctx, attentionID)
}

// GetCapabilityLease returns one capability lease by instance id.
func (s *Service) GetCapabilityLease(ctx context.Context, instanceID string) (domain.CapabilityLease, error) {
	if s == nil || s.repo == nil {
		return domain.CapabilityLease{}, fmt.Errorf("service is not configured")
	}
	instanceID = strings.TrimSpace(instanceID)
	if instanceID == "" {
		return domain.CapabilityLease{}, domain.ErrInvalidID
	}
	return s.repo.GetCapabilityLease(ctx, instanceID)
}

// ResolveAuthScopeContext derives the narrowest project-rooted auth path a level tuple proves.
func (s *Service) ResolveAuthScopeContext(ctx context.Context, in domain.LevelTupleInput) (AuthScopeContext, error) {
	if s == nil || s.repo == nil {
		return AuthScopeContext{}, fmt.Errorf("service is not configured")
	}

	level, err := domain.NewLevelTuple(in)
	if err != nil {
		return AuthScopeContext{}, err
	}
	if level.ScopeType == domain.ScopeLevelProject {
		return AuthScopeContext{
			ProjectID: level.ProjectID,
			ScopeType: domain.ScopeLevelProject,
			ScopeID:   level.ProjectID,
		}, nil
	}

	actionItem, err := s.repo.GetActionItem(ctx, level.ScopeID)
	if err != nil {
		return AuthScopeContext{}, err
	}
	if strings.TrimSpace(actionItem.ProjectID) != level.ProjectID {
		return AuthScopeContext{}, ErrNotFound
	}

	// Scope mirrors kind in the 12-value enum, so every action-item row is
	// ScopeLevelActionItem for auth-path purposes regardless of kind.
	actualScopeType := scopeLevelForActionItem(actionItem)
	if level.ScopeType != actualScopeType {
		return AuthScopeContext{}, domain.ErrInvalidScopeType
	}

	lineage, err := s.actionItemLineage(ctx, actionItem)
	if err != nil {
		return AuthScopeContext{}, err
	}
	contextScope, err := authScopeContextFromActionItemLineage(level.ProjectID, actualScopeType, actionItem.ID, lineage)
	if err != nil {
		return AuthScopeContext{}, err
	}
	return contextScope, nil
}

// actionItemLineage returns the root-to-leaf lineage for one actionItem-scoped hierarchy node.
func (s *Service) actionItemLineage(ctx context.Context, actionItem domain.ActionItem) ([]domain.ActionItem, error) {
	projectID := strings.TrimSpace(actionItem.ProjectID)
	if projectID == "" {
		return nil, domain.ErrInvalidID
	}

	reversed := make([]domain.ActionItem, 0, 8)
	current := actionItem
	for {
		reversed = append(reversed, current)

		parentID := strings.TrimSpace(current.ParentID)
		if parentID == "" {
			break
		}
		parent, err := s.repo.GetActionItem(ctx, parentID)
		if err != nil {
			return nil, err
		}
		if strings.TrimSpace(parent.ProjectID) != projectID {
			return nil, ErrNotFound
		}
		current = parent
	}

	lineage := make([]domain.ActionItem, 0, len(reversed))
	for idx := len(reversed) - 1; idx >= 0; idx-- {
		lineage = append(lineage, reversed[idx])
	}
	return lineage, nil
}

// authScopeContextFromActionItemLineage converts one validated actionItem lineage into auth-path context.
func authScopeContextFromActionItemLineage(projectID string, scopeType domain.ScopeLevel, scopeID string, lineage []domain.ActionItem) (AuthScopeContext, error) {
	projectID = strings.TrimSpace(projectID)
	scopeType = domain.NormalizeScopeLevel(scopeType)
	scopeID = strings.TrimSpace(scopeID)
	if projectID == "" {
		return AuthScopeContext{}, domain.ErrInvalidID
	}
	if scopeType == "" {
		return AuthScopeContext{}, domain.ErrInvalidScopeType
	}
	if scopeID == "" {
		return AuthScopeContext{}, domain.ErrInvalidScopeID
	}

	out := AuthScopeContext{
		ProjectID: projectID,
		ScopeType: scopeType,
		ScopeID:   scopeID,
	}
	// Lineage previously populated BranchID / PhaseIDs when an ancestor row
	// carried KindAppliesToBranch or KindAppliesToPhase. The 12-value Kind
	// enum no longer includes those scopes, so no ancestor contributes a
	// branch or phase anchor and the fallback below forces project scope.
	_ = lineage

	// The auth path model only narrows below project once a branch root exists.
	if out.BranchID == "" && (scopeType == domain.ScopeLevelPhase || scopeType == domain.ScopeLevelActionItem || scopeType == domain.ScopeLevelSubtask) {
		out.ScopeType = domain.ScopeLevelProject
		out.ScopeID = projectID
		out.PhaseIDs = nil
	}
	return out, nil
}
