package app

import (
	"context"
	"fmt"
	"strings"

	"github.com/hylla/tillsyn/internal/domain"
)

// CreateHandoffInput captures fields for creating one durable handoff.
type CreateHandoffInput struct {
	Level           domain.LevelTupleInput
	SourceRole      string
	TargetBranchID  string
	TargetScopeType domain.ScopeLevel
	TargetScopeID   string
	TargetRole      string
	Status          domain.HandoffStatus
	Summary         string
	NextAction      string
	MissingEvidence []string
	RelatedRefs     []string
	CreatedBy       string
	CreatedType     domain.ActorType
}

// UpdateHandoffInput captures fields for updating one durable handoff.
type UpdateHandoffInput struct {
	HandoffID       string
	Status          domain.HandoffStatus
	SourceRole      string
	TargetBranchID  string
	TargetScopeType domain.ScopeLevel
	TargetScopeID   string
	TargetRole      string
	Summary         string
	NextAction      string
	MissingEvidence []string
	RelatedRefs     []string
	UpdatedBy       string
	UpdatedType     domain.ActorType
	ResolvedBy      string
	ResolvedType    domain.ActorType
	ResolutionNote  string
}

// ListHandoffsInput captures fields for listing handoffs within one scope.
type ListHandoffsInput struct {
	Level    domain.LevelTupleInput
	Statuses []domain.HandoffStatus
	Limit    int
}

// CreateHandoff creates one durable handoff record.
func (s *Service) CreateHandoff(ctx context.Context, in CreateHandoffInput) (domain.Handoff, error) {
	if s.handoffRepo == nil {
		return domain.Handoff{}, fmt.Errorf("handoffs are not configured")
	}

	ctxActor, hasCtxActor := MutationActorFromContext(ctx)
	ctx, resolvedActor, hasResolvedActor := withResolvedMutationActor(ctx, in.CreatedBy, "", in.CreatedType)
	level, err := domain.NewLevelTuple(in.Level)
	if err != nil {
		return domain.Handoff{}, err
	}
	scopeID, err := s.validateCapabilityScopeTuple(ctx, level.ProjectID, level.ScopeType.ToCapabilityScopeType(), level.ScopeID)
	if err != nil {
		return domain.Handoff{}, err
	}
	level.ScopeID = scopeID

	createdType := normalizeActorTypeInput(in.CreatedType)
	createdBy := strings.TrimSpace(in.CreatedBy)
	if hasCtxActor {
		createdBy = ctxActor.ActorID
		createdType = normalizeActorTypeInput(ctxActor.ActorType)
	} else if hasResolvedActor {
		createdBy = resolvedActor.ActorID
		createdType = resolvedActor.ActorType
	}
	if err := s.enforceMutationGuard(ctx, level.ProjectID, createdType, level.ScopeType.ToCapabilityScopeType(), level.ScopeID, domain.CapabilityActionComment); err != nil {
		return domain.Handoff{}, err
	}

	handoff, err := domain.NewHandoff(domain.HandoffInput{
		ID:              s.idGen(),
		ProjectID:       level.ProjectID,
		BranchID:        level.BranchID,
		ScopeType:       level.ScopeType,
		ScopeID:         level.ScopeID,
		SourceRole:      in.SourceRole,
		TargetBranchID:  in.TargetBranchID,
		TargetScopeType: in.TargetScopeType,
		TargetScopeID:   in.TargetScopeID,
		TargetRole:      in.TargetRole,
		Status:          in.Status,
		Summary:         in.Summary,
		NextAction:      in.NextAction,
		MissingEvidence: in.MissingEvidence,
		RelatedRefs:     in.RelatedRefs,
		CreatedByActor:  createdBy,
		CreatedByType:   createdType,
		UpdatedByActor:  createdBy,
		UpdatedByType:   createdType,
	}, s.clock())
	if err != nil {
		return domain.Handoff{}, err
	}
	if err := s.handoffRepo.CreateHandoff(ctx, handoff); err != nil {
		return domain.Handoff{}, err
	}
	if err := s.syncHandoffInboxAttention(ctx, handoff); err != nil {
		return domain.Handoff{}, err
	}
	return handoff, nil
}

// GetHandoff returns one durable handoff by id.
func (s *Service) GetHandoff(ctx context.Context, handoffID string) (domain.Handoff, error) {
	if s.handoffRepo == nil {
		return domain.Handoff{}, fmt.Errorf("handoffs are not configured")
	}
	handoffID = strings.TrimSpace(handoffID)
	if handoffID == "" {
		return domain.Handoff{}, domain.ErrInvalidID
	}
	return s.handoffRepo.GetHandoff(ctx, handoffID)
}

// ListHandoffs lists durable handoffs for one source scope.
func (s *Service) ListHandoffs(ctx context.Context, in ListHandoffsInput) ([]domain.Handoff, error) {
	if s.handoffRepo == nil {
		return nil, fmt.Errorf("handoffs are not configured")
	}
	level, err := domain.NewLevelTuple(in.Level)
	if err != nil {
		return nil, err
	}
	scopeID, err := s.validateCapabilityScopeTuple(ctx, level.ProjectID, level.ScopeType.ToCapabilityScopeType(), level.ScopeID)
	if err != nil {
		return nil, err
	}
	level.ScopeID = scopeID
	filter, err := domain.NormalizeHandoffListFilter(domain.HandoffListFilter{
		ProjectID: level.ProjectID,
		BranchID:  level.BranchID,
		ScopeType: level.ScopeType,
		ScopeID:   level.ScopeID,
		Statuses:  in.Statuses,
		Limit:     in.Limit,
	})
	if err != nil {
		return nil, err
	}
	return s.handoffRepo.ListHandoffs(ctx, filter)
}

// UpdateHandoff updates one durable handoff and returns the updated row.
func (s *Service) UpdateHandoff(ctx context.Context, in UpdateHandoffInput) (domain.Handoff, error) {
	if s.handoffRepo == nil {
		return domain.Handoff{}, fmt.Errorf("handoffs are not configured")
	}
	handoffID := strings.TrimSpace(in.HandoffID)
	if handoffID == "" {
		return domain.Handoff{}, domain.ErrInvalidID
	}

	existing, err := s.handoffRepo.GetHandoff(ctx, handoffID)
	if err != nil {
		return domain.Handoff{}, err
	}
	ctx, resolvedActor, hasResolvedActor := withResolvedMutationActor(ctx, in.UpdatedBy, "", in.UpdatedType)
	ctxActor, hasCtxActor := MutationActorFromContext(ctx)
	updatedType := normalizeActorTypeInput(in.UpdatedType)
	updatedBy := strings.TrimSpace(in.UpdatedBy)
	resolvedBy := strings.TrimSpace(in.ResolvedBy)
	resolvedType := normalizeActorTypeInput(in.ResolvedType)
	if hasCtxActor {
		updatedBy = ctxActor.ActorID
		updatedType = normalizeActorTypeInput(ctxActor.ActorType)
		if resolvedBy == "" {
			resolvedBy = ctxActor.ActorID
		}
		if resolvedType == "" {
			resolvedType = normalizeActorTypeInput(ctxActor.ActorType)
		}
	} else if hasResolvedActor {
		updatedBy = resolvedActor.ActorID
		updatedType = resolvedActor.ActorType
		if resolvedBy == "" {
			resolvedBy = resolvedActor.ActorID
		}
		if resolvedType == "" {
			resolvedType = resolvedActor.ActorType
		}
	}
	if err := s.enforceMutationGuard(ctx, existing.ProjectID, updatedType, existing.ScopeType.ToCapabilityScopeType(), existing.ScopeID, domain.CapabilityActionComment); err != nil {
		return domain.Handoff{}, err
	}

	update := domain.HandoffUpdateInput{
		Status:          in.Status,
		SourceRole:      in.SourceRole,
		TargetBranchID:  in.TargetBranchID,
		TargetScopeType: in.TargetScopeType,
		TargetScopeID:   in.TargetScopeID,
		TargetRole:      in.TargetRole,
		Summary:         chooseHandoffUpdateString(in.Summary, existing.Summary),
		NextAction:      in.NextAction,
		MissingEvidence: chooseHandoffUpdateStrings(in.MissingEvidence, existing.MissingEvidence),
		RelatedRefs:     chooseHandoffUpdateStrings(in.RelatedRefs, existing.RelatedRefs),
		UpdatedByActor:  updatedBy,
		UpdatedByType:   updatedType,
		ResolvedByActor: resolvedBy,
		ResolvedByType:  resolvedType,
		ResolutionNote:  in.ResolutionNote,
	}
	if err := existing.Update(update, s.clock()); err != nil {
		return domain.Handoff{}, err
	}
	if err := s.handoffRepo.UpdateHandoff(ctx, existing); err != nil {
		return domain.Handoff{}, err
	}
	if err := s.syncHandoffInboxAttention(ctx, existing); err != nil {
		return domain.Handoff{}, err
	}
	return existing, nil
}

// chooseHandoffUpdateString returns the fallback when the candidate is blank.
func chooseHandoffUpdateString(candidate, fallback string) string {
	candidate = strings.TrimSpace(candidate)
	if candidate != "" {
		return candidate
	}
	return fallback
}

// chooseHandoffUpdateScopeLevel returns the fallback when the candidate is blank.
func chooseHandoffUpdateScopeLevel(candidate, fallback domain.ScopeLevel) domain.ScopeLevel {
	candidate = domain.NormalizeScopeLevel(candidate)
	if candidate != "" {
		return candidate
	}
	return fallback
}

// chooseHandoffUpdateStrings returns the fallback when the candidate slice is nil.
func chooseHandoffUpdateStrings(candidate, fallback []string) []string {
	if candidate == nil {
		return append([]string(nil), fallback...)
	}
	return append([]string(nil), candidate...)
}

// normalizeHandoffRole canonicalizes one freeform handoff role string.
func normalizeHandoffRole(role string) string {
	return strings.TrimSpace(strings.ToLower(role))
}

// normalizeHandoffList trims, deduplicates, and preserves order for handoff metadata slices.
func normalizeHandoffList(values []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

// chooseActorID returns the first non-empty actor id or the default local actor.
func chooseActorID(candidates ...string) string {
	for _, candidate := range candidates {
		candidate = strings.TrimSpace(candidate)
		if candidate != "" {
			return candidate
		}
	}
	return "tillsyn-user"
}
