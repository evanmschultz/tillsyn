package domain

import (
	"slices"
	"strings"
	"time"
)

// AttentionState identifies one lifecycle state for an attention item.
type AttentionState string

// AttentionState values.
const (
	AttentionStateOpen         AttentionState = "open"
	AttentionStateAcknowledged AttentionState = "acknowledged"
	AttentionStateResolved     AttentionState = "resolved"
)

// validAttentionStates stores supported attention-state values.
var validAttentionStates = []AttentionState{
	AttentionStateOpen,
	AttentionStateAcknowledged,
	AttentionStateResolved,
}

// AttentionKind identifies one typed attention signal.
type AttentionKind string

// AttentionKind values.
const (
	AttentionKindBlocker           AttentionKind = "blocker"
	AttentionKindConsensusRequired AttentionKind = "consensus_required"
	AttentionKindApprovalRequired  AttentionKind = "approval_required"
	AttentionKindMention           AttentionKind = "mention"
	AttentionKindHandoff           AttentionKind = "handoff"
	AttentionKindRiskNote          AttentionKind = "risk_note"
	// AttentionKindTemplateRejection flags an attention row materialized when
	// a template's AllowsNesting check rejects an auth-gated CreateActionItem
	// request. Per Drop 3 droplet 3.16 + finding 5.B.15 (N2 scope-narrow):
	// rejection-comments + attention items fire ONLY on auth-gated creates
	// (human/agent driven); dispatcher-internal auto-create rejections route
	// differently in Drop 4 (failed state on parent, no comment).
	AttentionKindTemplateRejection AttentionKind = "template_rejection"
)

// validAttentionKinds stores supported attention-kind values.
var validAttentionKinds = []AttentionKind{
	AttentionKindBlocker,
	AttentionKindConsensusRequired,
	AttentionKindApprovalRequired,
	AttentionKindMention,
	AttentionKindHandoff,
	AttentionKindRiskNote,
	AttentionKindTemplateRejection,
}

// AttentionItem stores one scoped attention record with lifecycle/audit data.
type AttentionItem struct {
	ID                  string
	ProjectID           string
	BranchID            string
	ScopeType           ScopeLevel
	ScopeID             string
	State               AttentionState
	Kind                AttentionKind
	Summary             string
	BodyMarkdown        string
	TargetRole          string
	RequiresUserAction  bool
	CreatedByActor      string
	CreatedByType       ActorType
	CreatedAt           time.Time
	AcknowledgedByActor string
	AcknowledgedByType  ActorType
	AcknowledgedAt      *time.Time
	ResolvedByActor     string
	ResolvedByType      ActorType
	ResolvedAt          *time.Time
}

// AttentionItemInput holds write-time values for creating one attention item.
type AttentionItemInput struct {
	ID                 string
	ProjectID          string
	BranchID           string
	ScopeType          ScopeLevel
	ScopeID            string
	State              AttentionState
	Kind               AttentionKind
	Summary            string
	BodyMarkdown       string
	TargetRole         string
	RequiresUserAction bool
	CreatedByActor     string
	CreatedByType      ActorType
}

// AttentionListFilter holds scoped query values for listing attention items.
type AttentionListFilter struct {
	ProjectID          string
	ScopeType          ScopeLevel
	ScopeID            string
	UnresolvedOnly     bool
	States             []AttentionState
	Kinds              []AttentionKind
	TargetRole         string
	RequiresUserAction *bool
	Limit              int
}

// NewAttentionItem validates and normalizes one attention-item create request.
func NewAttentionItem(in AttentionItemInput, now time.Time) (AttentionItem, error) {
	in.ID = strings.TrimSpace(in.ID)
	in.Summary = strings.TrimSpace(in.Summary)
	in.BodyMarkdown = strings.TrimSpace(in.BodyMarkdown)
	in.TargetRole = normalizeCoordinationRoleLabel(in.TargetRole)
	in.State = NormalizeAttentionState(in.State)
	in.Kind = NormalizeAttentionKind(in.Kind)

	if in.ID == "" {
		return AttentionItem{}, ErrInvalidID
	}
	if in.Summary == "" {
		return AttentionItem{}, ErrInvalidSummary
	}
	if in.State == "" {
		in.State = AttentionStateOpen
	}
	if !IsValidAttentionState(in.State) {
		return AttentionItem{}, ErrInvalidAttentionState
	}
	if !IsValidAttentionKind(in.Kind) {
		return AttentionItem{}, ErrInvalidAttentionKind
	}

	level, err := NewLevelTuple(LevelTupleInput{
		ProjectID: in.ProjectID,
		BranchID:  in.BranchID,
		ScopeType: in.ScopeType,
		ScopeID:   in.ScopeID,
	})
	if err != nil {
		return AttentionItem{}, err
	}

	createdByActor := strings.TrimSpace(in.CreatedByActor)
	if createdByActor == "" {
		createdByActor = "tillsyn-user"
	}
	createdByType := normalizeActorTypeValue(in.CreatedByType)
	if createdByType == "" {
		createdByType = ActorTypeUser
	}
	if !isValidActorType(createdByType) {
		return AttentionItem{}, ErrInvalidActorType
	}

	ts := now.UTC()
	item := AttentionItem{
		ID:                 in.ID,
		ProjectID:          level.ProjectID,
		BranchID:           level.BranchID,
		ScopeType:          level.ScopeType,
		ScopeID:            level.ScopeID,
		State:              in.State,
		Kind:               in.Kind,
		Summary:            in.Summary,
		BodyMarkdown:       in.BodyMarkdown,
		TargetRole:         in.TargetRole,
		RequiresUserAction: in.RequiresUserAction,
		CreatedByActor:     createdByActor,
		CreatedByType:      createdByType,
		CreatedAt:          ts,
	}
	if item.State == AttentionStateAcknowledged {
		item.AcknowledgedByActor = createdByActor
		item.AcknowledgedByType = createdByType
		item.AcknowledgedAt = &ts
	}
	if item.State == AttentionStateResolved {
		item.ResolvedByActor = createdByActor
		item.ResolvedByType = createdByType
		item.ResolvedAt = &ts
	}
	return item, nil
}

// NormalizeAttentionState canonicalizes one attention-state value.
func NormalizeAttentionState(state AttentionState) AttentionState {
	return AttentionState(strings.TrimSpace(strings.ToLower(string(state))))
}

// IsValidAttentionState reports whether an attention state is supported.
func IsValidAttentionState(state AttentionState) bool {
	state = NormalizeAttentionState(state)
	return slices.Contains(validAttentionStates, state)
}

// NormalizeAttentionKind canonicalizes one attention-kind value.
func NormalizeAttentionKind(kind AttentionKind) AttentionKind {
	return AttentionKind(strings.TrimSpace(strings.ToLower(string(kind))))
}

// IsValidAttentionKind reports whether an attention kind is supported.
func IsValidAttentionKind(kind AttentionKind) bool {
	kind = NormalizeAttentionKind(kind)
	return slices.Contains(validAttentionKinds, kind)
}

// NormalizeAttentionListFilter validates and normalizes one list filter.
func NormalizeAttentionListFilter(filter AttentionListFilter) (AttentionListFilter, error) {
	filter.ProjectID = strings.TrimSpace(filter.ProjectID)
	filter.ScopeType = NormalizeScopeLevel(filter.ScopeType)
	filter.ScopeID = strings.TrimSpace(filter.ScopeID)
	filter.TargetRole = normalizeCoordinationRoleLabel(filter.TargetRole)
	if filter.ProjectID == "" {
		return AttentionListFilter{}, ErrInvalidID
	}
	if filter.ScopeType != "" && !IsValidScopeLevel(filter.ScopeType) {
		return AttentionListFilter{}, ErrInvalidScopeType
	}
	if filter.ScopeType == "" && filter.ScopeID != "" {
		return AttentionListFilter{}, ErrInvalidScopeType
	}

	seenStates := map[AttentionState]struct{}{}
	normalizedStates := make([]AttentionState, 0, len(filter.States))
	for _, raw := range filter.States {
		state := NormalizeAttentionState(raw)
		if state == "" {
			continue
		}
		if !IsValidAttentionState(state) {
			return AttentionListFilter{}, ErrInvalidAttentionState
		}
		if _, ok := seenStates[state]; ok {
			continue
		}
		seenStates[state] = struct{}{}
		normalizedStates = append(normalizedStates, state)
	}
	filter.States = normalizedStates

	seenKinds := map[AttentionKind]struct{}{}
	normalizedKinds := make([]AttentionKind, 0, len(filter.Kinds))
	for _, raw := range filter.Kinds {
		kind := NormalizeAttentionKind(raw)
		if kind == "" {
			continue
		}
		if !IsValidAttentionKind(kind) {
			return AttentionListFilter{}, ErrInvalidAttentionKind
		}
		if _, ok := seenKinds[kind]; ok {
			continue
		}
		seenKinds[kind] = struct{}{}
		normalizedKinds = append(normalizedKinds, kind)
	}
	filter.Kinds = normalizedKinds

	if filter.Limit < 0 {
		filter.Limit = 0
	}

	return filter, nil
}

// IsUnresolved reports whether the attention item remains unresolved.
func (item AttentionItem) IsUnresolved() bool {
	return NormalizeAttentionState(item.State) != AttentionStateResolved
}

// BlocksCompletion reports whether unresolved attention should block completion.
func (item AttentionItem) BlocksCompletion() bool {
	if !item.IsUnresolved() {
		return false
	}
	if item.RequiresUserAction {
		return true
	}
	switch NormalizeAttentionKind(item.Kind) {
	case AttentionKindBlocker, AttentionKindConsensusRequired, AttentionKindApprovalRequired:
		return true
	default:
		return false
	}
}

// Resolve marks one attention item as resolved with actor attribution.
func (item *AttentionItem) Resolve(resolvedBy string, resolvedByType ActorType, now time.Time) error {
	if item == nil {
		return ErrInvalidID
	}
	resolvedBy = strings.TrimSpace(resolvedBy)
	if resolvedBy == "" {
		resolvedBy = "tillsyn-user"
	}
	resolvedByType = normalizeActorTypeValue(resolvedByType)
	if resolvedByType == "" {
		resolvedByType = ActorTypeUser
	}
	if !isValidActorType(resolvedByType) {
		return ErrInvalidActorType
	}
	ts := now.UTC()
	item.State = AttentionStateResolved
	item.ResolvedByActor = resolvedBy
	item.ResolvedByType = resolvedByType
	item.ResolvedAt = &ts
	return nil
}
