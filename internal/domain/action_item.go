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
	ID        string
	ProjectID string
	ParentID  string
	Kind      Kind
	Scope     KindAppliesTo
	// Role optionally tags an action item with a closed-enum role (e.g.
	// builder, qa-proof, planner). Empty string is the zero value and is
	// permitted — callers that require a role should validate downstream.
	Role Role
	// StructuralType places the action item on the cascade tree's shape
	// axis — drop / segment / confluence / droplet — independent of Kind
	// (which describes the work-type axis). MUST be a member of the closed
	// StructuralType enum; empty is rejected by NewActionItem with
	// ErrInvalidStructuralType. Semantics per `ta-docs/cascade-methodology.md`
	// §11.3.
	StructuralType StructuralType
	// Irreducible marks droplets that cannot decompose further — single
	// function-signature changes, single SQL migrations, single template
	// edits. Default false. Semantics per `ta-docs/cascade-methodology.md`
	// §2.3 and §11.3.
	Irreducible bool
	// Owner optionally tags an action item with the principal-name of an
	// orchestrator that owns it (e.g. "STEWARD"). Free-form string —
	// trim-only, no closed-enum membership check; any non-empty trimmed
	// value is permitted so future template-defined owned kinds can pick
	// their own owner names without a domain-package change. STEWARD is the
	// dominant present-day consumer (auth-gate keys on it), but Owner is a
	// domain primitive — not STEWARD-specific. Empty is the zero value and
	// means "no owner / orchestrator-routable by default." Semantics per
	// `ta-docs/cascade-methodology.md` §11.2.
	Owner string
	// DropNumber stores the cascade drop index a node is associated with
	// (e.g. 3 for "drop 3"). Zero is the zero value and is treated as "not
	// a numbered drop" rather than "drop 0"; consumers MUST NOT distinguish
	// between unset and drop-0. Negative values are rejected by
	// NewActionItem with ErrInvalidDropNumber.
	//
	// Rollback-cost note (per Drop 3 plan-QA Round 2 finding 5.C.16): if
	// this is later moved back into metadata-JSON, every consumer except
	// the auth gate (3.18 schema column, 3.20 read-side filters, 3.21 MCP
	// plumbing, plus their tests) must change shape — non-trivial backout.
	// Domain primitive, not STEWARD-specific. Semantics per
	// `ta-docs/cascade-methodology.md` §11.2.
	DropNumber int
	// Persistent marks long-lived umbrella nodes (refinement rollups,
	// anchor nodes, perpetual tracking trees) that survive across drops
	// rather than being created and closed within a single drop. Default
	// false. The 6 anchor nodes are the dominant present-day consumer, but
	// Persistent is a domain primitive — not STEWARD-specific. Semantics
	// per `ta-docs/cascade-methodology.md` §11.2.
	Persistent bool
	// DevGated marks nodes whose terminal transition requires explicit dev
	// sign-off — typically refinement rollups and human-verify hold points.
	// Default false. The refinements gate is the dominant present-day
	// consumer, but DevGated is a domain primitive — not STEWARD-specific.
	// Semantics per `ta-docs/cascade-methodology.md` §11.2.
	DevGated bool
	// Paths optionally enumerates the relative-from-repo-root file paths the
	// action item declares as its write scope (lock domain). Forward slashes
	// only (matches `git ls-files` output convention). Empty slice is the
	// meaningful zero value — no path scope declared. NewActionItem trims
	// each entry, dedupes (matches Labels normalization), and rejects
	// whitespace-only / backslash-bearing entries with ErrInvalidPaths.
	// Path-exists is NOT enforced at the domain layer — paths often refer to
	// files the build droplet will create. Validation is consumer-side
	// (Drop 4a Wave 2 lock manager). Domain primitive per Drop 4a L3.
	Paths []string
	// Packages optionally enumerates the Go-package import paths that cover
	// the entries in Paths. Used as the package-level lock domain by the
	// Wave 2 dispatcher's lock manager — sibling action items sharing a
	// package contend even when their Paths sets are disjoint within that
	// package. Free-form non-empty trimmed strings (no Go-import-path format
	// enforcement); planner-set values are what matter, not syntactic
	// checks. NewActionItem trims, dedupes, and rejects whitespace-only /
	// empty entries with ErrInvalidPackages. Coverage invariant: when Paths
	// is non-empty, Packages MUST also be non-empty (rejected with
	// ErrInvalidPackages, message "packages must cover paths"). Strict
	// path→package resolution is deferred to the Wave 2 lock manager.
	// Domain primitive per Drop 4a L3 / WAVE_1_PLAN.md §1.2.
	Packages []string
	// Files optionally enumerates relative-from-repo-root file paths the
	// action item attaches as reference material — files the agent should
	// read or look at while doing the work, distinct from Paths (which
	// declares write-scope / lock domain). Forward slashes only (matches
	// `git ls-files` convention). Empty slice is the meaningful zero value
	// — no reference files attached. NewActionItem trims each entry,
	// dedupes (matches Labels / Paths normalization), and rejects
	// whitespace-only / backslash-bearing entries with ErrInvalidFiles.
	// Path-exists is NOT enforced at the domain layer — the canonical
	// consumer is the Drop 4.5 TUI file-viewer pane, which validates path
	// existence at view time. Disjoint-axis with Paths: Files and Paths
	// are NOT cross-checked for overlap or disjointness — Paths declares
	// write intent (lock scope) while Files declares read attention
	// (reference attachments), and legitimate overlap is permitted (e.g.
	// an agent edits a file referenced as a viewer in a read-then-edit
	// workflow). Domain primitive per Drop 4a L3 / WAVE_1_PLAN.md §1.3.
	Files          []string
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
	ID        string
	ProjectID string
	ParentID  string
	Kind      Kind
	Scope     KindAppliesTo
	// Role optionally tags the action item with a closed-enum role. Empty
	// string is permitted and round-trips as the zero-value Role; non-empty
	// values must match the closed Role enum or NewActionItem returns
	// ErrInvalidRole.
	Role Role
	// StructuralType MUST be a member of the closed StructuralType enum.
	// Empty is rejected — NewActionItem returns ErrInvalidStructuralType.
	// This diverges from Role's permissive empty because the cascade tree's
	// shape axis is mandatory: every node must place itself as drop /
	// segment / confluence / droplet.
	StructuralType StructuralType
	// Irreducible marks single-function-signature changes / single SQL
	// migrations / single template edits — droplets that cannot decompose
	// further. Default false. Semantics per
	// `ta-docs/cascade-methodology.md` §2.3 and §11.3.
	Irreducible bool
	// Owner optionally tags the action item with a principal-name string
	// (e.g. "STEWARD"). Free-form — NewActionItem trims surrounding
	// whitespace but does NOT enforce a closed enum, so any non-empty
	// trimmed value round-trips. Whitespace-only collapses to empty.
	// Semantics per `ta-docs/cascade-methodology.md` §11.2.
	Owner string
	// DropNumber stores the cascade drop index. Zero is the zero value and
	// means "not a numbered drop"; positive values round-trip; negative
	// values reject with ErrInvalidDropNumber. Semantics per
	// `ta-docs/cascade-methodology.md` §11.2.
	DropNumber int
	// Persistent marks long-lived umbrella / anchor / perpetual-tracking
	// nodes. Default false. Domain primitive — not STEWARD-specific.
	// Semantics per `ta-docs/cascade-methodology.md` §11.2.
	Persistent bool
	// DevGated marks nodes whose terminal transition requires dev sign-off
	// (refinement rollups, human-verify hold points). Default false.
	// Domain primitive — not STEWARD-specific. Semantics per
	// `ta-docs/cascade-methodology.md` §11.2.
	DevGated bool
	// Paths optionally enumerates the action item's write-scope relative
	// paths (forward-slash, repo-root-relative). Empty slice is the
	// meaningful zero value (no path scope declared). NewActionItem trims +
	// dedupes; whitespace-only / backslash-bearing entries reject with
	// ErrInvalidPaths. Path-exists is NOT enforced at this layer — paths may
	// refer to files the build droplet will create. Domain primitive per
	// Drop 4a L3.
	Paths []string
	// Packages optionally enumerates the Go-package import paths that cover
	// Paths. Empty slice is the meaningful zero value (no package scope).
	// NewActionItem trims + dedupes; whitespace-only / empty entries reject
	// with ErrInvalidPackages. Coverage invariant: non-empty Paths requires
	// non-empty Packages (else ErrInvalidPackages "packages must cover
	// paths"). Strict path→package resolution is deferred to the Wave 2
	// lock manager. Domain primitive per Drop 4a L3 / WAVE_1_PLAN.md §1.2.
	Packages []string
	// Files optionally enumerates the action item's reference-material
	// relative paths (forward-slash, repo-root-relative). Empty slice is
	// the meaningful zero value (no reference files attached).
	// NewActionItem trims + dedupes; whitespace-only / backslash-bearing
	// entries reject with ErrInvalidFiles. Disjoint-axis with Paths —
	// Files (read attention) and Paths (write intent / lock scope) may
	// legitimately overlap, so no cross-axis check is performed. Domain
	// primitive per Drop 4a L3 / WAVE_1_PLAN.md §1.3.
	Files          []string
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
	// Role is optional. NormalizeRole collapses whitespace-only input to the
	// empty string; an empty normalized role is permitted and round-trips as
	// the zero-value Role. A non-empty normalized value must be a member of
	// the closed Role enum — short-circuit on emptiness because IsValidRole
	// rejects the empty string.
	in.Role = NormalizeRole(in.Role)
	if in.Role != "" && !IsValidRole(in.Role) {
		return ActionItem{}, ErrInvalidRole
	}
	// StructuralType is mandatory per the cascade-methodology shape axis.
	// Normalize first so whitespace-only collapses to empty and the empty
	// check fires uniformly. Empty AND non-enum values both reject with
	// ErrInvalidStructuralType. This diverges from Role's permissive empty
	// — the cascade tree's shape axis must be supplied at creation.
	in.StructuralType = NormalizeStructuralType(in.StructuralType)
	if in.StructuralType == "" {
		return ActionItem{}, ErrInvalidStructuralType
	}
	if !IsValidStructuralType(in.StructuralType) {
		return ActionItem{}, ErrInvalidStructuralType
	}
	// Owner is a free-form principal-name string. Trim only — no closed-enum
	// membership check, since future template-defined owned kinds can pick
	// their own owner names. Whitespace-only collapses to empty. Persistent
	// and DevGated are bools with no validation; their zero-value (false) is
	// the dominant case.
	in.Owner = strings.TrimSpace(in.Owner)
	// DropNumber: zero is the zero value and is permitted (treated as "not a
	// numbered drop"); positive values round-trip; negative values reject.
	if in.DropNumber < 0 {
		return ActionItem{}, ErrInvalidDropNumber
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

	paths, err := normalizeActionItemPaths(in.Paths)
	if err != nil {
		return ActionItem{}, err
	}

	packages, err := normalizeActionItemPackages(in.Packages)
	if err != nil {
		return ActionItem{}, err
	}
	// Coverage invariant: non-empty Paths requires non-empty Packages so the
	// Wave 2 lock manager always has a package-level lock domain to attach
	// path-level locks to. Strict path→package resolution is deferred to
	// the lock manager — this is the simpler "packages must cover paths"
	// gate per WAVE_1_PLAN.md §1.2.
	if len(paths) > 0 && len(packages) == 0 {
		return ActionItem{}, ErrInvalidPackages
	}

	// Files is the reference-attachment slice (read attention). Normalized
	// independently of Paths/Packages — disjoint-axis rule per
	// WAVE_1_PLAN.md §1.3 means legitimate overlap with Paths is permitted
	// and no cross-axis coverage check applies.
	files, err := normalizeActionItemFiles(in.Files)
	if err != nil {
		return ActionItem{}, err
	}

	return ActionItem{
		ID:             in.ID,
		ProjectID:      in.ProjectID,
		ParentID:       in.ParentID,
		Kind:           in.Kind,
		Scope:          in.Scope,
		Role:           in.Role,
		StructuralType: in.StructuralType,
		Irreducible:    in.Irreducible,
		Owner:          in.Owner,
		DropNumber:     in.DropNumber,
		Persistent:     in.Persistent,
		DevGated:       in.DevGated,
		Paths:          paths,
		Packages:       packages,
		Files:          files,
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
	if prev != StateInProgress && state == StateInProgress && t.StartedAt == nil {
		t.StartedAt = &ts
	}
	// CompletedAt is reused for both complete and failed (D1). The metadata.outcome
	// field (D6) distinguishes success from failure. Both branches must be
	// updated atomically — setting one without the other causes CompletedAt to
	// be set and immediately nilled in the same call.
	if (prev != StateComplete && state == StateComplete) || (prev != StateFailed && state == StateFailed) {
		t.CompletedAt = &ts
	}
	if state != StateComplete && state != StateFailed {
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
	if t.Metadata.CompletionContract.Policy.RequireChildrenComplete {
		for _, child := range children {
			if child.ArchivedAt != nil {
				continue
			}
			if normalizeLifecycleState(child.LifecycleState) != StateComplete {
				out = append(out, fmt.Sprintf("child item %q is not complete", child.Title))
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
		if item.Complete {
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

// NormalizeActionItemPaths normalizes the Paths slice using the same rules
// NewActionItem applies at construction time. Exposed so callers that
// mutate ActionItem.Paths after construction (e.g. Service.UpdateActionItem
// applying a pointer-sentinel update) reuse the canonical
// trim/dedupe/forward-slash-check logic rather than reimplementing it.
//
// Behaviour: each entry is trimmed; whitespace-only / empty entries reject
// with ErrInvalidPaths (rather than silently dropping — empty entries
// almost always indicate a planner bug, not benign noise). Backslash-
// bearing entries also reject with ErrInvalidPaths to enforce the forward-
// slash / `git ls-files` convention. Duplicates after trim are silently
// deduped to match the Labels precedent (path duplicates almost always
// come from copy-paste in agent prompts; rejecting forces agent retries on
// benign noise). Insertion order is preserved (the dispatcher's lock
// manager reads the slice as ordered, so deterministic ordering matters).
// Empty input (nil or len == 0) returns nil. Path-exists is intentionally
// NOT enforced — paths often refer to files the build droplet will create.
// Drop 4a Wave 2 lock manager performs runtime validation when locks are
// acquired.
func NormalizeActionItemPaths(paths []string) ([]string, error) {
	return normalizeActionItemPaths(paths)
}

// normalizeActionItemPaths is the internal worker for NormalizeActionItemPaths;
// see that function's doc for behaviour. NewActionItem calls this directly
// to avoid the extra wrapper hop.
func normalizeActionItemPaths(paths []string) ([]string, error) {
	if len(paths) == 0 {
		return nil, nil
	}
	out := make([]string, 0, len(paths))
	seen := map[string]struct{}{}
	for _, raw := range paths {
		path := strings.TrimSpace(raw)
		if path == "" {
			return nil, ErrInvalidPaths
		}
		if strings.ContainsRune(path, '\\') {
			return nil, ErrInvalidPaths
		}
		if _, ok := seen[path]; ok {
			continue
		}
		seen[path] = struct{}{}
		out = append(out, path)
	}
	return out, nil
}

// NormalizeActionItemPackages normalizes the Packages slice using the same
// rules NewActionItem applies at construction time. Exposed so callers
// that mutate ActionItem.Packages after construction (e.g.
// Service.UpdateActionItem applying a pointer-sentinel update) reuse the
// canonical trim/dedupe logic rather than reimplementing it.
//
// Behaviour: each entry is trimmed; whitespace-only / empty entries reject
// with ErrInvalidPackages (rather than silently dropping — empty entries
// almost always indicate a planner bug, not benign noise). No Go-import-
// path format enforcement is applied — any non-empty trimmed string
// round-trips, since planner-set values are what matter and a syntactic
// validator would reject legitimate forms (`internal/domain`,
// `github.com/foo/bar`, etc.). Duplicates after trim are silently deduped
// to match the Labels / Paths precedent. Insertion order is preserved
// (the dispatcher's lock manager reads the slice as ordered, so
// deterministic ordering matters). Empty input (nil or len == 0) returns
// nil.
//
// The coverage invariant — non-empty Paths requires non-empty Packages —
// lives in NewActionItem (and Service.UpdateActionItem after applying the
// update), not in this normalizer, so callers can normalize Packages in
// isolation without forcing them to also know about Paths.
func NormalizeActionItemPackages(packages []string) ([]string, error) {
	return normalizeActionItemPackages(packages)
}

// normalizeActionItemPackages is the internal worker for
// NormalizeActionItemPackages; see that function's doc for behaviour.
// NewActionItem calls this directly to avoid the extra wrapper hop.
func normalizeActionItemPackages(packages []string) ([]string, error) {
	if len(packages) == 0 {
		return nil, nil
	}
	out := make([]string, 0, len(packages))
	seen := map[string]struct{}{}
	for _, raw := range packages {
		pkg := strings.TrimSpace(raw)
		if pkg == "" {
			return nil, ErrInvalidPackages
		}
		if _, ok := seen[pkg]; ok {
			continue
		}
		seen[pkg] = struct{}{}
		out = append(out, pkg)
	}
	return out, nil
}

// NormalizeActionItemFiles normalizes the Files slice using the same rules
// NewActionItem applies at construction time. Exposed so callers that
// mutate ActionItem.Files after construction (e.g. Service.UpdateActionItem
// applying a pointer-sentinel update) reuse the canonical
// trim/dedupe/forward-slash-check logic rather than reimplementing it.
//
// Behaviour: each entry is trimmed; whitespace-only / empty entries reject
// with ErrInvalidFiles (rather than silently dropping — empty entries
// almost always indicate a planner bug, not benign noise). Backslash-
// bearing entries also reject with ErrInvalidFiles to enforce the
// forward-slash / `git ls-files` convention. Duplicates after trim are
// silently deduped to match the Labels / Paths precedent. Insertion order
// is preserved (the canonical consumer is the Drop 4.5 TUI file-viewer
// pane, which reads the slice as ordered). Empty input (nil or len == 0)
// returns nil. Path-exists is intentionally NOT enforced — paths often
// refer to files the build droplet will create, and consumer-side
// validation (Drop 4.5 file-viewer) handles existence at view time.
//
// Disjoint-axis with Paths: Files is NOT cross-checked against Paths for
// overlap or disjointness, since legitimate overlap is permitted —
// e.g. an agent may edit a file referenced as a viewer in a read-then-
// edit workflow.
func NormalizeActionItemFiles(files []string) ([]string, error) {
	return normalizeActionItemFiles(files)
}

// normalizeActionItemFiles is the internal worker for
// NormalizeActionItemFiles; see that function's doc for behaviour.
// NewActionItem calls this directly to avoid the extra wrapper hop.
func normalizeActionItemFiles(files []string) ([]string, error) {
	if len(files) == 0 {
		return nil, nil
	}
	out := make([]string, 0, len(files))
	seen := map[string]struct{}{}
	for _, raw := range files {
		file := strings.TrimSpace(raw)
		if file == "" {
			return nil, ErrInvalidFiles
		}
		if strings.ContainsRune(file, '\\') {
			return nil, ErrInvalidFiles
		}
		if _, ok := seen[file]; ok {
			continue
		}
		seen[file] = struct{}{}
		out = append(out, file)
	}
	return out, nil
}
