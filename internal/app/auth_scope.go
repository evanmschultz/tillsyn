package app

import (
	"context"
	"fmt"
	"strings"

	"github.com/hylla/tillsyn/internal/domain"
)

// AuthScopeContext stores the narrowest project-rooted auth path one resource proves.
type AuthScopeContext struct {
	ProjectID string
	BranchID  string
	ScopeType domain.ScopeLevel
	ScopeID   string
	PhaseIDs  []string
}

// GetTask returns one work-item row by id.
func (s *Service) GetTask(ctx context.Context, taskID string) (domain.Task, error) {
	if s == nil || s.repo == nil {
		return domain.Task{}, fmt.Errorf("service is not configured")
	}
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return domain.Task{}, domain.ErrInvalidID
	}
	return s.repo.GetTask(ctx, taskID)
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

	task, err := s.repo.GetTask(ctx, level.ScopeID)
	if err != nil {
		return AuthScopeContext{}, err
	}
	if strings.TrimSpace(task.ProjectID) != level.ProjectID {
		return AuthScopeContext{}, ErrNotFound
	}

	actualScopeType := domain.ScopeLevelFromKindAppliesTo(task.Scope)
	if actualScopeType == "" {
		return AuthScopeContext{}, domain.ErrInvalidScopeType
	}
	if level.ScopeType != actualScopeType {
		return AuthScopeContext{}, domain.ErrInvalidScopeType
	}

	lineage, err := s.taskLineage(ctx, task)
	if err != nil {
		return AuthScopeContext{}, err
	}
	contextScope, err := authScopeContextFromTaskLineage(level.ProjectID, actualScopeType, task.ID, lineage)
	if err != nil {
		return AuthScopeContext{}, err
	}
	return contextScope, nil
}

// taskLineage returns the root-to-leaf lineage for one task-scoped hierarchy node.
func (s *Service) taskLineage(ctx context.Context, task domain.Task) ([]domain.Task, error) {
	projectID := strings.TrimSpace(task.ProjectID)
	if projectID == "" {
		return nil, domain.ErrInvalidID
	}

	reversed := make([]domain.Task, 0, 8)
	current := task
	for {
		reversed = append(reversed, current)

		parentID := strings.TrimSpace(current.ParentID)
		if parentID == "" {
			break
		}
		parent, err := s.repo.GetTask(ctx, parentID)
		if err != nil {
			return nil, err
		}
		if strings.TrimSpace(parent.ProjectID) != projectID {
			return nil, ErrNotFound
		}
		current = parent
	}

	lineage := make([]domain.Task, 0, len(reversed))
	for idx := len(reversed) - 1; idx >= 0; idx-- {
		lineage = append(lineage, reversed[idx])
	}
	return lineage, nil
}

// authScopeContextFromTaskLineage converts one validated task lineage into auth-path context.
func authScopeContextFromTaskLineage(projectID string, scopeType domain.ScopeLevel, scopeID string, lineage []domain.Task) (AuthScopeContext, error) {
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
	for _, node := range lineage {
		switch domain.ScopeLevelFromKindAppliesTo(node.Scope) {
		case domain.ScopeLevelBranch:
			out.BranchID = node.ID
		case domain.ScopeLevelPhase:
			out.PhaseIDs = append(out.PhaseIDs, node.ID)
		}
	}

	// The auth path model only narrows below project once a branch root exists.
	if out.BranchID == "" && (scopeType == domain.ScopeLevelTask || scopeType == domain.ScopeLevelSubtask) {
		out.ScopeType = domain.ScopeLevelProject
		out.ScopeID = projectID
	}
	if out.ScopeType == domain.ScopeLevelPhase && out.BranchID == "" {
		return AuthScopeContext{}, domain.ErrInvalidScopeID
	}
	return out, nil
}
