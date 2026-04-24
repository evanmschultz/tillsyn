package domain

import (
	"fmt"
	"slices"
	"strings"
	"time"
)

// Priority represents priority data used by this package.
type Priority string

// PriorityLow and related constants define package defaults.
const (
	PriorityLow    Priority = "low"
	PriorityMedium Priority = "medium"
	PriorityHigh   Priority = "high"
)

// validPriorities stores a package-level helper value.
var validPriorities = []Priority{PriorityLow, PriorityMedium, PriorityHigh}

// ActionItem represents actionItem data used by this package.
type ActionItem struct {
	ID             string
	ProjectID      string
	ParentID       string
	Kind           Kind
	Scope          KindAppliesTo
	LifecycleState LifecycleState
	ColumnID       string
	Position       int
	Title          string
	Description    string
	Priority       Priority
	DueAt          *time.Time
	Labels         []string
	Metadata       ActionItemMetadata
	CreatedByActor string
	CreatedByName  string
	UpdatedByActor string
	UpdatedByName  string
	UpdatedByType  ActorType
	CreatedAt      time.Time
	UpdatedAt      time.Time
	StartedAt      *time.Time
	CompletedAt    *time.Time
	ArchivedAt     *time.Time
	CanceledAt     *time.Time
}

// ActionItemInput holds input values for actionItem operations.
type ActionItemInput struct {
	ID             string
	ProjectID      string
	ParentID       string
	Kind           Kind
	Scope          KindAppliesTo
	LifecycleState LifecycleState
	ColumnID       string
	Position       int
	Title          string
	Description    string
	Priority       Priority
	DueAt          *time.Time
	Labels         []string
	Metadata       ActionItemMetadata
	CreatedByActor string
	CreatedByName  string
	UpdatedByActor string
	UpdatedByName  string
	UpdatedByType  ActorType
}

// DefaultActionItemScope returns the canonical default scope for one work-item
// kind. Scope mirrors kind per the 12-value Kind enum, so the scope is the
// KindAppliesTo value whose stored form equals the supplied kind. The helper
// returns the empty KindAppliesTo when the kind is not a member of the enum so
// the caller can reject with ErrInvalidKind.
func DefaultActionItemScope(kind Kind) KindAppliesTo {
	if !IsValidKind(kind) {
		return ""
	}
	return KindAppliesTo(Kind(strings.TrimSpace(strings.ToLower(string(kind)))))
}

// NewActionItem constructs a new value for this package.
func NewActionItem(in ActionItemInput, now time.Time) (ActionItem, error) {
	in.ID = strings.TrimSpace(in.ID)
	in.ProjectID = strings.TrimSpace(in.ProjectID)
	in.ParentID = strings.TrimSpace(in.ParentID)
	in.ColumnID = strings.TrimSpace(in.ColumnID)
	in.Title = strings.TrimSpace(in.Title)
	in.Description = strings.TrimSpace(in.Description)

	if in.ID == "" {
		return ActionItem{}, ErrInvalidID
	}
	if in.ProjectID == "" {
		return ActionItem{}, ErrInvalidID
	}
	if in.ParentID != "" && in.ParentID == in.ID {
		return ActionItem{}, ErrInvalidParentID
	}
	if in.ColumnID == "" {
		return ActionItem{}, ErrInvalidColumnID
	}
	if in.Title == "" {
		return ActionItem{}, ErrInvalidTitle
	}
	if in.Position < 0 {
		return ActionItem{}, ErrInvalidPosition
	}

	if in.Priority == "" {
		in.Priority = PriorityMedium
	}
	if !slices.Contains(validPriorities, in.Priority) {
		return ActionItem{}, ErrInvalidPriority
	}
	in.Kind = Kind(strings.TrimSpace(strings.ToLower(string(in.Kind))))
	if in.Kind == "" {
		return ActionItem{}, ErrInvalidKind
	}
	if !IsValidKind(in.Kind) {
		return ActionItem{}, ErrInvalidKind
	}
	in.Scope = NormalizeKindAppliesTo(in.Scope)
	if in.Scope == "" {
		in.Scope = DefaultActionItemScope(in.Kind)
	}
	if !IsValidWorkItemAppliesTo(in.Scope) {
		return ActionItem{}, ErrInvalidKindAppliesTo
	}
	// Scope mirrors kind per the 12-value Kind enum. Reject any caller that
	// supplies a scope that disagrees with the kind; downstream persistence
	// relies on the mirror invariant.
	if in.Scope != KindAppliesTo(in.Kind) {
		return ActionItem{}, ErrInvalidKindAppliesTo
	}
	if in.LifecycleState == "" {
		in.LifecycleState = StateTodo
	}
	if !isValidLifecycleState(in.LifecycleState) {
		return ActionItem{}, ErrInvalidLifecycleState
	}
	if in.UpdatedByType == "" {
		in.UpdatedByType = ActorTypeUser
	}
	if !isValidActorType(in.UpdatedByType) {
		return ActionItem{}, ErrInvalidActorType
	}
	if strings.TrimSpace(in.CreatedByActor) == "" {
		in.CreatedByActor = "tillsyn-user"
	}
	if strings.TrimSpace(in.CreatedByName) == "" {
		in.CreatedByName = strings.TrimSpace(in.CreatedByActor)
	}
	if strings.TrimSpace(in.UpdatedByActor) == "" {
		in.UpdatedByActor = in.CreatedByActor
	}
	if strings.TrimSpace(in.UpdatedByName) == "" {
		if strings.TrimSpace(in.UpdatedByActor) == strings.TrimSpace(in.CreatedByActor) {
			in.UpdatedByName = strings.TrimSpace(in.CreatedByName)
		}
		if strings.TrimSpace(in.UpdatedByName) == "" {
			in.UpdatedByName = strings.TrimSpace(in.UpdatedByActor)
		}
	}

	labels := normalizeLabels(in.Labels)
	metadata, err := normalizeActionItemMetadata(in.Metadata)
	if err != nil {
		return ActionItem{}, err
	}

	return ActionItem{
		ID:             in.ID,
		ProjectID:      in.ProjectID,
		ParentID:       in.ParentID,
		Kind:           in.Kind,
		Scope:          in.Scope,
		LifecycleState: in.LifecycleState,
		ColumnID:       in.ColumnID,
		Position:       in.Position,
		Title:          in.Title,
		Description:    in.Description,
		Priority:       in.Priority,
		DueAt:          normalizeDueAt(in.DueAt),
		Labels:         labels,
		Metadata:       metadata,
		CreatedByActor: strings.TrimSpace(in.CreatedByActor),
		CreatedByName:  strings.TrimSpace(in.CreatedByName),
		UpdatedByActor: strings.TrimSpace(in.UpdatedByActor),
		UpdatedByName:  strings.TrimSpace(in.UpdatedByName),
		UpdatedByType:  in.UpdatedByType,
		CreatedAt:      now.UTC(),
		UpdatedAt:      now.UTC(),
	}, nil
}

// Move moves the requested operation.
func (t *ActionItem) Move(columnID string, position int, now time.Time) error {
	columnID = strings.TrimSpace(columnID)
	if columnID == "" {
		return ErrInvalidColumnID
	}
	if position < 0 {
		return ErrInvalidPosition
	}
	t.ColumnID = columnID
	t.Position = position
	t.UpdatedAt = now.UTC()
	return nil
}

// UpdateDetails updates state for the requested operation.
func (t *ActionItem) UpdateDetails(title, description string, priority Priority, dueAt *time.Time, labels []string, now time.Time) error {
	title = strings.TrimSpace(title)
	description = strings.TrimSpace(description)
	if title == "" {
		return ErrInvalidTitle
	}
	if !slices.Contains(validPriorities, priority) {
		return ErrInvalidPriority
	}
	t.Title = title
	t.Description = description
	t.Priority = priority
	t.DueAt = normalizeDueAt(dueAt)
	t.Labels = normalizeLabels(labels)
	t.UpdatedAt = now.UTC()
	return nil
}

// UpdatePlanningMetadata updates planning-specific metadata for the actionItem.
func (t *ActionItem) UpdatePlanningMetadata(metadata ActionItemMetadata, actorID string, actorType ActorType, now time.Time) error {
	if !isValidActorType(actorType) {
		return ErrInvalidActorType
	}
	normalized, err := normalizeActionItemMetadata(metadata)
	if err != nil {
		return err
	}
	t.Metadata = normalized
	actorID = strings.TrimSpace(actorID)
	if actorID == "" {
		actorID = t.UpdatedByActor
		if actorID == "" {
			actorID = "tillsyn-user"
		}
	}
	t.UpdatedByActor = actorID
	t.UpdatedByType = actorType
	t.UpdatedAt = now.UTC()
	return nil
}

// SetLifecycleState changes lifecycle state and maintains lifecycle timestamps.
func (t *ActionItem) SetLifecycleState(state LifecycleState, now time.Time) error {
	if !isValidLifecycleState(state) {
		return ErrInvalidLifecycleState
	}
	state = normalizeLifecycleState(state)
	ts := now.UTC()
	prev := t.LifecycleState
	t.LifecycleState = state
	if prev != StateProgress && state == StateProgress && t.StartedAt == nil {
		t.StartedAt = &ts
	}
	// CompletedAt is reused for both done and failed (D1). The metadata.outcome
	// field (D6) distinguishes success from failure. Both branches must be
	// updated atomically — setting one without the other causes CompletedAt to
	// be set and immediately nilled in the same call.
	if (prev != StateDone && state == StateDone) || (prev != StateFailed && state == StateFailed) {
		t.CompletedAt = &ts
	}
	if state != StateDone && state != StateFailed {
		t.CompletedAt = nil
	}
	if state == StateArchived {
		t.ArchivedAt = &ts
	} else if t.ArchivedAt != nil {
		t.ArchivedAt = nil
	}
	t.UpdatedAt = ts
	return nil
}

// Reparent changes the parent relationship of a actionItem.
func (t *ActionItem) Reparent(parentID string, now time.Time) error {
	parentID = strings.TrimSpace(parentID)
	if parentID == t.ID {
		return ErrInvalidParentID
	}
	t.ParentID = parentID
	t.UpdatedAt = now.UTC()
	return nil
}

// StartCriteriaUnmet returns start-criteria items that are not yet satisfied.
func (t ActionItem) StartCriteriaUnmet() []string {
	return incompleteChecklistItems(t.Metadata.CompletionContract.StartCriteria)
}

// CompletionCriteriaUnmet returns completion requirements that are not yet satisfied.
func (t ActionItem) CompletionCriteriaUnmet(children []ActionItem) []string {
	out := incompleteChecklistItems(t.Metadata.CompletionContract.CompletionCriteria)
	out = append(out, incompleteChecklistItems(t.Metadata.CompletionContract.CompletionChecklist)...)
	if t.Metadata.CompletionContract.Policy.RequireChildrenDone {
		for _, child := range children {
			if child.ArchivedAt != nil {
				continue
			}
			if normalizeLifecycleState(child.LifecycleState) != StateDone {
				out = append(out, fmt.Sprintf("child item %q is not done", child.Title))
			}
		}
	}
	return out
}

// Archive archives the requested operation.
func (t *ActionItem) Archive(now time.Time) {
	ts := now.UTC()
	t.ArchivedAt = &ts
	t.LifecycleState = StateArchived
	t.UpdatedAt = ts
}

// Restore restores the requested operation.
func (t *ActionItem) Restore(now time.Time) {
	t.ArchivedAt = nil
	if t.LifecycleState == StateArchived {
		t.LifecycleState = StateTodo
	}
	t.UpdatedAt = now.UTC()
}

// normalizeDueAt normalizes due at.
func normalizeDueAt(dueAt *time.Time) *time.Time {
	if dueAt == nil {
		return nil
	}
	ts := dueAt.UTC().Truncate(time.Second)
	return &ts
}

// incompleteChecklistItems reports every checklist item that is not complete.
func incompleteChecklistItems(in []ChecklistItem) []string {
	out := make([]string, 0, len(in))
	for _, item := range in {
		text := strings.TrimSpace(item.Text)
		if text == "" {
			continue
		}
		if item.Done {
			continue
		}
		out = append(out, text)
	}
	return out
}

// normalizeLabels normalizes labels.
func normalizeLabels(labels []string) []string {
	out := make([]string, 0, len(labels))
	seen := map[string]struct{}{}
	for _, raw := range labels {
		label := strings.ToLower(strings.TrimSpace(raw))
		if label == "" {
			continue
		}
		if _, ok := seen[label]; ok {
			continue
		}
		seen[label] = struct{}{}
		out = append(out, label)
	}
	slices.Sort(out)
	return out
}
