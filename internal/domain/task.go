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

// Task represents task data used by this package.
type Task struct {
	ID             string
	ProjectID      string
	ParentID       string
	Kind           WorkKind
	Scope          KindAppliesTo
	LifecycleState LifecycleState
	ColumnID       string
	Position       int
	Title          string
	Description    string
	Priority       Priority
	DueAt          *time.Time
	Labels         []string
	Metadata       TaskMetadata
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

// TaskInput holds input values for task operations.
type TaskInput struct {
	ID             string
	ProjectID      string
	ParentID       string
	Kind           WorkKind
	Scope          KindAppliesTo
	LifecycleState LifecycleState
	ColumnID       string
	Position       int
	Title          string
	Description    string
	Priority       Priority
	DueAt          *time.Time
	Labels         []string
	Metadata       TaskMetadata
	CreatedByActor string
	CreatedByName  string
	UpdatedByActor string
	UpdatedByName  string
	UpdatedByType  ActorType
}

// DefaultTaskScope returns the canonical default scope for one work-item kind and parent tuple.
func DefaultTaskScope(kind WorkKind, parentID string) KindAppliesTo {
	parentID = strings.TrimSpace(parentID)
	switch strings.TrimSpace(strings.ToLower(string(kind))) {
	case "branch":
		return KindAppliesToBranch
	case "phase":
		return KindAppliesToPhase
	case "subtask":
		return KindAppliesToSubtask
	default:
		if parentID == "" {
			return KindAppliesToTask
		}
		return KindAppliesToSubtask
	}
}

// NewTask constructs a new value for this package.
func NewTask(in TaskInput, now time.Time) (Task, error) {
	in.ID = strings.TrimSpace(in.ID)
	in.ProjectID = strings.TrimSpace(in.ProjectID)
	in.ParentID = strings.TrimSpace(in.ParentID)
	in.ColumnID = strings.TrimSpace(in.ColumnID)
	in.Title = strings.TrimSpace(in.Title)
	in.Description = strings.TrimSpace(in.Description)

	if in.ID == "" {
		return Task{}, ErrInvalidID
	}
	if in.ProjectID == "" {
		return Task{}, ErrInvalidID
	}
	if in.ParentID != "" && in.ParentID == in.ID {
		return Task{}, ErrInvalidParentID
	}
	if in.ColumnID == "" {
		return Task{}, ErrInvalidColumnID
	}
	if in.Title == "" {
		return Task{}, ErrInvalidTitle
	}
	if in.Position < 0 {
		return Task{}, ErrInvalidPosition
	}

	if in.Priority == "" {
		in.Priority = PriorityMedium
	}
	if !slices.Contains(validPriorities, in.Priority) {
		return Task{}, ErrInvalidPriority
	}
	if in.Kind == "" {
		in.Kind = WorkKindTask
	}
	if !isValidWorkKind(in.Kind) {
		return Task{}, ErrInvalidKind
	}
	in.Scope = NormalizeKindAppliesTo(in.Scope)
	if in.Scope == "" {
		in.Scope = DefaultTaskScope(in.Kind, in.ParentID)
	}
	if !IsValidWorkItemAppliesTo(in.Scope) {
		return Task{}, ErrInvalidKindAppliesTo
	}
	if in.ParentID == "" && in.Scope == KindAppliesToSubtask {
		return Task{}, ErrInvalidParentID
	}
	if in.LifecycleState == "" {
		in.LifecycleState = StateTodo
	}
	if !isValidLifecycleState(in.LifecycleState) {
		return Task{}, ErrInvalidLifecycleState
	}
	if in.UpdatedByType == "" {
		in.UpdatedByType = ActorTypeUser
	}
	if !isValidActorType(in.UpdatedByType) {
		return Task{}, ErrInvalidActorType
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
	metadata, err := normalizeTaskMetadata(in.Metadata)
	if err != nil {
		return Task{}, err
	}

	return Task{
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
func (t *Task) Move(columnID string, position int, now time.Time) error {
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
func (t *Task) UpdateDetails(title, description string, priority Priority, dueAt *time.Time, labels []string, now time.Time) error {
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

// UpdatePlanningMetadata updates planning-specific metadata for the task.
func (t *Task) UpdatePlanningMetadata(metadata TaskMetadata, actorID string, actorType ActorType, now time.Time) error {
	if !isValidActorType(actorType) {
		return ErrInvalidActorType
	}
	normalized, err := normalizeTaskMetadata(metadata)
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
func (t *Task) SetLifecycleState(state LifecycleState, now time.Time) error {
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
	if prev != StateDone && state == StateDone {
		t.CompletedAt = &ts
	}
	if state != StateDone {
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

// Reparent changes the parent relationship of a task.
func (t *Task) Reparent(parentID string, now time.Time) error {
	parentID = strings.TrimSpace(parentID)
	if parentID == t.ID {
		return ErrInvalidParentID
	}
	t.ParentID = parentID
	t.UpdatedAt = now.UTC()
	return nil
}

// StartCriteriaUnmet returns start-criteria items that are not yet satisfied.
func (t Task) StartCriteriaUnmet() []string {
	return incompleteChecklistItems(t.Metadata.CompletionContract.StartCriteria)
}

// CompletionCriteriaUnmet returns completion requirements that are not yet satisfied.
func (t Task) CompletionCriteriaUnmet(children []Task) []string {
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
func (t *Task) Archive(now time.Time) {
	ts := now.UTC()
	t.ArchivedAt = &ts
	t.LifecycleState = StateArchived
	t.UpdatedAt = ts
}

// Restore restores the requested operation.
func (t *Task) Restore(now time.Time) {
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
