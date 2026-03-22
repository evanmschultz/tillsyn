package domain

import (
	"slices"
	"strings"
	"time"
)

// HandoffStatus identifies one durable coordination state for a handoff.
type HandoffStatus string

// Handoff status values.
const (
	HandoffStatusReady      HandoffStatus = "ready"
	HandoffStatusWaiting    HandoffStatus = "waiting"
	HandoffStatusBlocked    HandoffStatus = "blocked"
	HandoffStatusFailed     HandoffStatus = "failed"
	HandoffStatusReturned   HandoffStatus = "returned"
	HandoffStatusSuperseded HandoffStatus = "superseded"
	HandoffStatusResolved   HandoffStatus = "resolved"
)

// validHandoffStatuses stores supported handoff-state values.
var validHandoffStatuses = []HandoffStatus{
	HandoffStatusReady,
	HandoffStatusWaiting,
	HandoffStatusBlocked,
	HandoffStatusFailed,
	HandoffStatusReturned,
	HandoffStatusSuperseded,
	HandoffStatusResolved,
}

// Handoff stores one durable, structured coordination record between agents and humans.
type Handoff struct {
	ID              string
	ProjectID       string
	BranchID        string
	ScopeType       ScopeLevel
	ScopeID         string
	SourceRole      string
	TargetBranchID  string
	TargetScopeType ScopeLevel
	TargetScopeID   string
	TargetRole      string
	Status          HandoffStatus
	Summary         string
	NextAction      string
	MissingEvidence []string
	RelatedRefs     []string
	CreatedByActor  string
	CreatedByType   ActorType
	CreatedAt       time.Time
	UpdatedByActor  string
	UpdatedByType   ActorType
	UpdatedAt       time.Time
	ResolvedByActor string
	ResolvedByType  ActorType
	ResolvedAt      *time.Time
	ResolutionNote  string
}

// HandoffInput holds write-time values for creating one handoff.
type HandoffInput struct {
	ID              string
	ProjectID       string
	BranchID        string
	ScopeType       ScopeLevel
	ScopeID         string
	SourceRole      string
	TargetBranchID  string
	TargetScopeType ScopeLevel
	TargetScopeID   string
	TargetRole      string
	Status          HandoffStatus
	Summary         string
	NextAction      string
	MissingEvidence []string
	RelatedRefs     []string
	CreatedByActor  string
	CreatedByType   ActorType
	UpdatedByActor  string
	UpdatedByType   ActorType
}

// HandoffUpdateInput holds write-time values for updating one handoff.
type HandoffUpdateInput struct {
	Status          HandoffStatus
	SourceRole      string
	TargetBranchID  string
	TargetScopeType ScopeLevel
	TargetScopeID   string
	TargetRole      string
	Summary         string
	NextAction      string
	MissingEvidence []string
	RelatedRefs     []string
	UpdatedByActor  string
	UpdatedByType   ActorType
	ResolvedByActor string
	ResolvedByType  ActorType
	ResolutionNote  string
}

// HandoffListFilter holds scoped query values for listing handoffs.
type HandoffListFilter struct {
	ProjectID string
	BranchID  string
	ScopeType ScopeLevel
	ScopeID   string
	Statuses  []HandoffStatus
	Limit     int
}

// NewHandoff validates and constructs one durable handoff record.
func NewHandoff(in HandoffInput, now time.Time) (Handoff, error) {
	in.ID = strings.TrimSpace(in.ID)
	in.Summary = strings.TrimSpace(in.Summary)
	in.NextAction = strings.TrimSpace(in.NextAction)
	in.SourceRole = normalizeHandoffRole(in.SourceRole)
	in.TargetRole = normalizeHandoffRole(in.TargetRole)
	in.Status = NormalizeHandoffStatus(in.Status)
	in.CreatedByActor = strings.TrimSpace(in.CreatedByActor)
	in.UpdatedByActor = strings.TrimSpace(in.UpdatedByActor)

	if in.ID == "" {
		return Handoff{}, ErrInvalidID
	}
	if in.Summary == "" {
		return Handoff{}, ErrInvalidSummary
	}

	source, err := NewLevelTuple(LevelTupleInput{
		ProjectID: in.ProjectID,
		BranchID:  in.BranchID,
		ScopeType: in.ScopeType,
		ScopeID:   in.ScopeID,
	})
	if err != nil {
		return Handoff{}, err
	}

	target, hasTarget, err := normalizeHandoffTarget(source.ProjectID, HandoffTargetInput{
		BranchID:  in.TargetBranchID,
		ScopeType: in.TargetScopeType,
		ScopeID:   in.TargetScopeID,
	})
	if err != nil {
		return Handoff{}, err
	}

	if in.Status == "" {
		in.Status = HandoffStatusWaiting
	}
	if !IsValidHandoffStatus(in.Status) {
		return Handoff{}, ErrInvalidHandoffStatus
	}
	if IsTerminalHandoffStatus(in.Status) {
		return Handoff{}, ErrInvalidHandoffTransition
	}

	createdByType := normalizeActorTypeValue(in.CreatedByType)
	if createdByType == "" {
		createdByType = ActorTypeUser
	}
	if !isValidActorType(createdByType) {
		return Handoff{}, ErrInvalidActorType
	}
	updatedByType := normalizeActorTypeValue(in.UpdatedByType)
	if updatedByType == "" {
		updatedByType = createdByType
	}
	if !isValidActorType(updatedByType) {
		return Handoff{}, ErrInvalidActorType
	}

	createdByActor := normalizeHandoffActor(in.CreatedByActor)
	if createdByActor == "" {
		createdByActor = "tillsyn-user"
	}
	updatedByActor := normalizeHandoffActor(in.UpdatedByActor)
	if updatedByActor == "" {
		updatedByActor = createdByActor
	}

	ts := now.UTC()
	handoff := Handoff{
		ID:              in.ID,
		ProjectID:       source.ProjectID,
		BranchID:        source.BranchID,
		ScopeType:       source.ScopeType,
		ScopeID:         source.ScopeID,
		SourceRole:      in.SourceRole,
		Status:          in.Status,
		Summary:         in.Summary,
		NextAction:      in.NextAction,
		MissingEvidence: normalizeHandoffList(in.MissingEvidence),
		RelatedRefs:     normalizeHandoffList(in.RelatedRefs),
		CreatedByActor:  createdByActor,
		CreatedByType:   createdByType,
		CreatedAt:       ts,
		UpdatedByActor:  updatedByActor,
		UpdatedByType:   updatedByType,
		UpdatedAt:       ts,
	}
	if hasTarget {
		handoff.TargetBranchID = target.BranchID
		handoff.TargetScopeType = target.ScopeType
		handoff.TargetScopeID = target.ScopeID
		handoff.TargetRole = in.TargetRole
	}
	return handoff, nil
}

// IsTerminal reports whether one handoff is in a terminal status.
func (h Handoff) IsTerminal() bool {
	return IsTerminalHandoffStatus(h.Status)
}

// Update validates and applies one handoff update in place.
func (h *Handoff) Update(in HandoffUpdateInput, now time.Time) error {
	if h == nil {
		return ErrInvalidID
	}
	if h.IsTerminal() {
		return ErrInvalidHandoffTransition
	}

	in.SourceRole = normalizeHandoffRole(in.SourceRole)
	in.TargetRole = normalizeHandoffRole(in.TargetRole)
	in.Status = NormalizeHandoffStatus(in.Status)
	in.UpdatedByActor = normalizeHandoffActor(in.UpdatedByActor)
	in.ResolvedByActor = normalizeHandoffActor(in.ResolvedByActor)

	if strings.TrimSpace(in.Summary) == "" {
		return ErrInvalidSummary
	}
	if in.Status == "" {
		in.Status = h.Status
	}
	if !IsValidHandoffStatus(in.Status) {
		return ErrInvalidHandoffStatus
	}

	target, hasTarget, err := normalizeHandoffTarget(h.ProjectID, HandoffTargetInput{
		BranchID:  in.TargetBranchID,
		ScopeType: in.TargetScopeType,
		ScopeID:   in.TargetScopeID,
	})
	if err != nil {
		return err
	}

	updatedByType := normalizeActorTypeValue(in.UpdatedByType)
	if updatedByType == "" {
		updatedByType = h.UpdatedByType
	}
	if !isValidActorType(updatedByType) {
		return ErrInvalidActorType
	}
	updatedByActor := normalizeHandoffActor(in.UpdatedByActor)
	if updatedByActor == "" {
		updatedByActor = normalizeHandoffActor(h.UpdatedByActor)
	}
	if updatedByActor == "" {
		updatedByActor = normalizeHandoffActor(h.CreatedByActor)
	}
	if updatedByActor == "" {
		updatedByActor = "tillsyn-user"
	}
	resolvedByType := normalizeActorTypeValue(in.ResolvedByType)
	if resolvedByType == "" {
		resolvedByType = updatedByType
	}
	if !isValidActorType(resolvedByType) {
		return ErrInvalidActorType
	}

	h.SourceRole = in.SourceRole
	h.Summary = strings.TrimSpace(in.Summary)
	h.NextAction = strings.TrimSpace(in.NextAction)
	h.MissingEvidence = normalizeHandoffList(in.MissingEvidence)
	h.RelatedRefs = normalizeHandoffList(in.RelatedRefs)
	h.UpdatedByActor = updatedByActor
	h.UpdatedByType = updatedByType
	h.UpdatedAt = now.UTC()
	h.Status = in.Status

	if hasTarget {
		h.TargetBranchID = target.BranchID
		h.TargetScopeType = target.ScopeType
		h.TargetScopeID = target.ScopeID
		h.TargetRole = in.TargetRole
	} else {
		h.TargetBranchID = ""
		h.TargetScopeType = ""
		h.TargetScopeID = ""
		h.TargetRole = ""
	}

	if IsTerminalHandoffStatus(h.Status) {
		resolvedByActor := in.ResolvedByActor
		if resolvedByActor == "" {
			resolvedByActor = h.UpdatedByActor
		}
		h.ResolvedByActor = resolvedByActor
		h.ResolvedByType = resolvedByType
		ts := now.UTC()
		h.ResolvedAt = &ts
		h.ResolutionNote = strings.TrimSpace(in.ResolutionNote)
		return nil
	}

	h.ResolvedByActor = ""
	h.ResolvedByType = ""
	h.ResolvedAt = nil
	h.ResolutionNote = ""
	return nil
}

// NormalizeHandoffStatus canonicalizes one handoff status value.
func NormalizeHandoffStatus(status HandoffStatus) HandoffStatus {
	return HandoffStatus(strings.TrimSpace(strings.ToLower(string(status))))
}

// IsValidHandoffStatus reports whether one handoff status is supported.
func IsValidHandoffStatus(status HandoffStatus) bool {
	return slices.Contains(validHandoffStatuses, NormalizeHandoffStatus(status))
}

// IsTerminalHandoffStatus reports whether one handoff status is final.
func IsTerminalHandoffStatus(status HandoffStatus) bool {
	switch NormalizeHandoffStatus(status) {
	case HandoffStatusResolved, HandoffStatusSuperseded:
		return true
	default:
		return false
	}
}

// NormalizeHandoffListFilter validates and normalizes one handoff list filter.
func NormalizeHandoffListFilter(filter HandoffListFilter) (HandoffListFilter, error) {
	filter.ProjectID = strings.TrimSpace(filter.ProjectID)
	filter.BranchID = strings.TrimSpace(filter.BranchID)
	filter.ScopeType = NormalizeScopeLevel(filter.ScopeType)
	filter.ScopeID = strings.TrimSpace(filter.ScopeID)
	if filter.ProjectID == "" {
		return HandoffListFilter{}, ErrInvalidID
	}
	if filter.ScopeType != "" && !IsValidScopeLevel(filter.ScopeType) {
		return HandoffListFilter{}, ErrInvalidScopeType
	}
	if filter.ScopeType == "" && filter.ScopeID != "" {
		return HandoffListFilter{}, ErrInvalidScopeType
	}

	seenStatuses := map[HandoffStatus]struct{}{}
	normalizedStatuses := make([]HandoffStatus, 0, len(filter.Statuses))
	for _, raw := range filter.Statuses {
		status := NormalizeHandoffStatus(raw)
		if status == "" {
			continue
		}
		if !IsValidHandoffStatus(status) {
			return HandoffListFilter{}, ErrInvalidHandoffStatus
		}
		if _, ok := seenStatuses[status]; ok {
			continue
		}
		seenStatuses[status] = struct{}{}
		normalizedStatuses = append(normalizedStatuses, status)
	}
	filter.Statuses = normalizedStatuses

	if filter.Limit < 0 {
		filter.Limit = 0
	}
	return filter, nil
}

// HandoffTargetInput holds write-time values for target-scope normalization.
type HandoffTargetInput struct {
	BranchID  string
	ScopeType ScopeLevel
	ScopeID   string
}

// normalizeHandoffTarget validates an optional target scope tuple.
func normalizeHandoffTarget(projectID string, in HandoffTargetInput) (LevelTuple, bool, error) {
	if strings.TrimSpace(in.BranchID) == "" && strings.TrimSpace(string(in.ScopeType)) == "" && strings.TrimSpace(in.ScopeID) == "" {
		return LevelTuple{}, false, nil
	}
	target, err := NewLevelTuple(LevelTupleInput{
		ProjectID: projectID,
		BranchID:  in.BranchID,
		ScopeType: in.ScopeType,
		ScopeID:   in.ScopeID,
	})
	if err != nil {
		return LevelTuple{}, false, err
	}
	return target, true, nil
}

// normalizeHandoffList trims, deduplicates, and preserves order for freeform handoff metadata.
func normalizeHandoffList(values []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, raw := range values {
		value := strings.TrimSpace(raw)
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

// normalizeHandoffRole canonicalizes one freeform handoff role string.
func normalizeHandoffRole(role string) string {
	return strings.TrimSpace(strings.ToLower(role))
}

// normalizeHandoffActor canonicalizes one freeform actor string.
func normalizeHandoffActor(actor string) string {
	return strings.TrimSpace(actor)
}
