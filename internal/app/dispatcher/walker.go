package dispatcher

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/evanmschultz/tillsyn/internal/domain"
)

// Walker contract overview.
//
// The treeWalker is the cascade dispatcher's auto-promotion engine. It reads
// the current project tree through a narrow service interface, evaluates each
// todo action item against the eligibility predicate documented below, and
// returns the subset that is safe to promote to in_progress. Promotion itself
// is split into a separate Promote method so the Wave 2.7 conflict detector
// can intercede between "eligible" and "promoted" — for example, to insert a
// runtime blocker when two siblings hit the same file lock.
//
// Read-only on the tree (acceptance §5):
//
// EligibleForPromotion never mutates state. It returns a slice of
// domain.ActionItem snapshots; the caller decides whether to call Promote on
// each one. This separation is intentional: the conflict detector (4a.20)
// inspects the eligible set, optionally writes a runtime blocker via the
// service, and only then calls Promote on the items it cleared. Walker has
// no opinion about ordering or batching across the eligible set.
//
// Broker ordering contract (acceptance §6):
//
// The walker is invoked by the broker subscriber loop in broker_sub.go AFTER
// the repo write that triggered the LiveWaitEventActionItemChanged event. The
// in-process LiveWaitBroker stamps every Publish into its latest map BEFORE
// waking waiters, and Service.MoveActionItem / UpdateActionItem call Publish
// only after the repo write returns. By the time the walker reads the tree,
// the latest write is durable. Coalesced bursts collapse into a single wake;
// the walker re-reads the project tree on each wake so coalescing is
// invisible to the eligibility decision. Wave 2.10's manual-trigger CLI
// invokes the walker imperatively rather than off the broker — the same
// read-after-write semantics apply because CLI invocations come from outside
// any in-flight repo transaction.
//
// Eligibility predicate (acceptance §2):
//
//  1. The action item's LifecycleState is StateTodo. Items already
//     in_progress / complete / failed / archived are filtered out.
//  2. Every entry in actionItem.Metadata.BlockedBy resolves to an action
//     item in StateComplete. Missing references (deleted siblings, typos)
//     are treated as not-clear and skip the item — this is conservative on
//     purpose: the planner sets BlockedBy and a missing target is a
//     planner-side bug, not a walker-side override.
//  3. Children-complete is NOT required for promotion to in_progress. The
//     children-complete invariant is enforced by Service.MoveActionItem
//     when the destination state is complete (see
//     ensureActionItemCompletionBlockersClear in internal/app/service.go);
//     auto-promotion to in_progress on a parent only requires the parent's
//     own gating, not its descendants. The asymmetry exists because
//     in_progress is a working state — children may be todo / in_progress
//     concurrently, and the "parent before children" cascade ordering is
//     captured by blocked_by edges, not by children-complete.
//  4. Parent (if any) is in StateInProgress or has Persistent=true. A
//     project-root item (ParentID == "") is treated as if its parent is
//     in_progress — there is nothing above the root to gate on, and Drop
//     1.5's drop-orch flow already pins the root in_progress at drop start.
//     Persistent parents (refinement umbrellas, anchor nodes) are
//     long-lived and may sit in any state — children under them are
//     allowed to promote regardless.
//
// Self-reference and cycle handling: BlockedBy entries that point back at
// the item itself are treated as not-clear (the item's own state is
// StateTodo, not StateComplete) so the walker skips it. The walker does NOT
// detect or break cycles in BlockedBy — that is a planner / domain
// validation concern.
//
// Concurrency: treeWalker holds no mutable state. Concurrent calls are safe
// to the extent the underlying service permits; Service.MoveActionItem is
// already serialized by SQLite's write lock.

// ErrPromotionBlocked is the typed error returned by Promote when the
// service rejects the in_progress transition. It wraps the inner
// domain.ErrTransitionBlocked (and the underlying service error) so callers
// can distinguish "planner-side blocker still pending" (errors.Is on
// ErrPromotionBlocked) from "infrastructure error" (any other non-nil
// error). Conflict-detector consumers (Wave 2.7) treat ErrPromotionBlocked
// as a recoverable signal to either insert a runtime blocker or back off
// for the next walker tick.
var ErrPromotionBlocked = errors.New("dispatcher: promotion to in_progress blocked")

// walkerService is the narrow consumer-side view the tree walker uses to
// read project columns + items and to execute promotions. *app.Service
// satisfies this interface; the test suite injects deterministic stubs so
// walker scenarios run without the full service + repository graph.
//
// Method names mirror Service exactly so the production binding is a
// trivial assignment in NewDispatcher (wired in 4a.23 along with the rest
// of the dispatcher graph).
type walkerService interface {
	ListColumns(ctx context.Context, projectID string, includeArchived bool) ([]domain.Column, error)
	ListActionItems(ctx context.Context, projectID string, includeArchived bool) ([]domain.ActionItem, error)
	MoveActionItem(ctx context.Context, actionItemID, toColumnID string, position int) (domain.ActionItem, error)
}

// treeWalker is the auto-promotion engine described above.
type treeWalker struct {
	svc walkerService
}

// newTreeWalker constructs a treeWalker bound to svc. svc MUST be non-nil;
// callers wire the production *app.Service via the dispatcher constructor
// (deferred to 4a.23). The test suite passes a stub walkerService directly
// through this constructor.
func newTreeWalker(svc walkerService) *treeWalker {
	return &treeWalker{svc: svc}
}

// EligibleForPromotion returns the snapshot of action items that are
// currently eligible for promotion to in_progress under the predicate
// documented in the package overview. The result is read-only on the tree:
// no state changes occur as a side effect of this call.
//
// Empty / whitespace projectID returns a nil slice and a nil error. The
// caller filters projects upstream; an empty project ID is a defensive
// no-op rather than a fatal misconfiguration.
//
// On a service error from ListColumns or ListActionItems the walker
// surfaces the wrapped error to the caller — eligibility evaluation is
// best-effort but it does not silently swallow infrastructure failures.
func (w *treeWalker) EligibleForPromotion(ctx context.Context, projectID string) ([]domain.ActionItem, error) {
	if w == nil || w.svc == nil {
		return nil, fmt.Errorf("%w: tree walker service is nil", ErrInvalidDispatcherConfig)
	}
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		return nil, nil
	}
	items, err := w.svc.ListActionItems(ctx, projectID, false)
	if err != nil {
		return nil, fmt.Errorf("walker: list action items %q: %w", projectID, err)
	}
	if len(items) == 0 {
		return nil, nil
	}

	// Index every item by ID once so the per-item eligibility check (which
	// dereferences BlockedBy and ParentID) stays O(1) per edge.
	byID := make(map[string]domain.ActionItem, len(items))
	for _, it := range items {
		byID[it.ID] = it
	}

	eligible := make([]domain.ActionItem, 0)
	for _, item := range items {
		if !w.isEligible(item, byID) {
			continue
		}
		eligible = append(eligible, item)
	}
	return eligible, nil
}

// isEligible evaluates the predicate documented in the package overview
// against one item, given the by-ID index of every action item in the
// project. Returns true iff item is in todo, every BlockedBy resolves to a
// complete item, and the parent (if any) is in_progress or Persistent.
func (w *treeWalker) isEligible(item domain.ActionItem, byID map[string]domain.ActionItem) bool {
	if item.LifecycleState != domain.StateTodo {
		return false
	}
	for _, blockerID := range item.Metadata.BlockedBy {
		blocker, ok := byID[strings.TrimSpace(blockerID)]
		if !ok {
			// Missing reference: treat as not-clear. A deleted-or-typo
			// blocker is a planner-side bug; the conservative path is to
			// hold this item until the planner reconciles BlockedBy.
			return false
		}
		if blocker.LifecycleState != domain.StateComplete {
			return false
		}
	}
	parentID := strings.TrimSpace(item.ParentID)
	if parentID == "" {
		// Project-root: nothing to gate on, and the drop-orch flow pins
		// the root to in_progress at drop start.
		return true
	}
	parent, ok := byID[parentID]
	if !ok {
		// Detached child (parent missing from the listing): treat as
		// not-clear. This is a project-shape bug elsewhere — walker is
		// not the place to repair it.
		return false
	}
	if parent.Persistent {
		return true
	}
	return parent.LifecycleState == domain.StateInProgress
}

// Promote moves item to its project's in_progress column via the service.
// The walker resolves the column ID dynamically by listing the project's
// columns and matching the canonical state slug — this mirrors the
// adapter's resolveActionItemColumnIDForState helper without taking on the
// adapter's auth-gate dependency, which is irrelevant inside the
// dispatcher (the dispatcher already holds project-scoped auth via the
// constructor).
//
// On Service.MoveActionItem returning a domain.ErrTransitionBlocked-wrapped
// error, Promote rewraps with ErrPromotionBlocked so callers can detect
// the planner-side-blocker condition with errors.Is. errors.Is also still
// matches the original ErrTransitionBlocked because the join preserves
// both targets in the wrapped error chain.
//
// Other service errors (database failures, transport errors, missing
// project) bubble up wrapped with context but without the
// ErrPromotionBlocked sentinel — the conflict detector treats those as
// real failures rather than recoverable promotion blocks.
func (w *treeWalker) Promote(ctx context.Context, item domain.ActionItem) (domain.ActionItem, error) {
	if w == nil || w.svc == nil {
		return domain.ActionItem{}, fmt.Errorf("%w: tree walker service is nil", ErrInvalidDispatcherConfig)
	}
	projectID := strings.TrimSpace(item.ProjectID)
	if projectID == "" {
		return domain.ActionItem{}, fmt.Errorf("walker: action item %q has empty project_id", item.ID)
	}
	columns, err := w.svc.ListColumns(ctx, projectID, true)
	if err != nil {
		return domain.ActionItem{}, fmt.Errorf("walker: list columns for project %q: %w", projectID, err)
	}
	columnID := columnIDForLifecycleState(columns, domain.StateInProgress)
	if columnID == "" {
		return domain.ActionItem{}, fmt.Errorf("walker: project %q has no in_progress column", projectID)
	}
	moved, err := w.svc.MoveActionItem(ctx, item.ID, columnID, item.Position)
	if err != nil {
		if errors.Is(err, domain.ErrTransitionBlocked) {
			return domain.ActionItem{}, fmt.Errorf("walker: promote action item %q: %w", item.ID, errors.Join(ErrPromotionBlocked, err))
		}
		return domain.ActionItem{}, fmt.Errorf("walker: promote action item %q: %w", item.ID, err)
	}
	return moved, nil
}

// columnIDForLifecycleState returns the column ID whose Name slugifies to
// the canonical state slug for state. Empty string when no column matches —
// callers surface this as an error.
//
// The slug rules mirror the canonical normalizer in
// internal/adapters/server/common (normalizeStateLikeID): trim, lowercase,
// collapse non-alphanumerics to underscores, then map "to_do" → "todo" and
// the canonical state slugs through. Legacy aliases (done, completed,
// progress, doing, in-progress) are intentionally rejected by returning
// the empty string from the slug step so the column lookup fails closed.
func columnIDForLifecycleState(columns []domain.Column, state domain.LifecycleState) string {
	want := canonicalStateSlug(string(state))
	if want == "" {
		return ""
	}
	for _, column := range columns {
		if canonicalStateSlug(column.Name) != want {
			continue
		}
		return strings.TrimSpace(column.ID)
	}
	return ""
}

// canonicalStateSlug normalizes a column name (or a domain.LifecycleState
// string) into the canonical state slug used for state↔column matching.
// Returns empty string for legacy aliases and unknown names, which causes
// the caller's lookup to fail closed.
func canonicalStateSlug(name string) string {
	name = strings.TrimSpace(strings.ToLower(name))
	if name == "" {
		return ""
	}
	switch name {
	case "done", "completed", "progress", "doing", "in-progress":
		return ""
	}
	var b strings.Builder
	lastUnderscore := false
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
			lastUnderscore = false
		case r >= '0' && r <= '9':
			b.WriteRune(r)
			lastUnderscore = false
		default:
			if !lastUnderscore {
				b.WriteByte('_')
				lastUnderscore = true
			}
		}
	}
	normalized := strings.Trim(b.String(), "_")
	switch normalized {
	case "to_do", "todo":
		return "todo"
	case "in_progress":
		return "in_progress"
	case "complete":
		return "complete"
	case "failed":
		return "failed"
	case "archived":
		return "archived"
	default:
		return ""
	}
}
