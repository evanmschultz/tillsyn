package dispatcher

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/evanmschultz/tillsyn/internal/app"
	"github.com/evanmschultz/tillsyn/internal/domain"
)

// Conflict detector overview.
//
// The conflictDetector intercedes between treeWalker.EligibleForPromotion
// (Wave 2.5) and treeWalker.Promote: when two siblings are eligible at the
// same time and their declared `paths` or `packages` overlap WITHOUT an
// explicit `blocked_by` edge between them, the dispatcher would otherwise
// race them onto the same file or package compile. The detector reports
// every such overlap so the caller can either insert a runtime
// `blocked_by` (via InsertRuntimeBlockedBy below) or, when the planner
// already wired an explicit blocker, no-op cleanly.
//
// Sibling-only scope (acceptance §7):
//
// Two action items are siblings iff they share the same ParentID. The
// detector does NOT walk cross-subtree cousins — items under different
// parents that happen to declare the same path are out of scope for Wave
// 2.7. Cross-subtree conflict resolution is a Drop 4b concern: the
// runtime fileLockManager + packageLockManager already serialize spawn
// across the entire tree at acquire time, so cross-subtree overlap does
// not race agents; only sibling overlap can race within a single
// promotion sweep, which is what this detector guards against.
//
// Tie-break for equally-eligible siblings (acceptance §8):
//
// When two siblings share the same parent, overlap on a path or package,
// and have no explicit blocked_by between them, the detector picks a
// canonical "earlier" sibling by (Position ascending, then ID lexically
// ascending). The caller (Wave 2.10's manual-trigger CLI; Drop 4b's
// continuous-mode dispatcher) inserts the runtime blocker on the LATER
// sibling pointing at the EARLIER one, preserving the planner's
// position-driven intent. Tie-break is exposed via TieBreakSibling so the
// caller does not have to recompute the ordering.
//
// Permanence of inserted runtime blocker (acceptance §9):
//
// Once InsertRuntimeBlockedBy succeeds, the new entry in BlockedBy is
// permanent — there is no cleanup hook that removes it after the holder
// completes. The runtime blocker IS the dependency edge. This matches
// the planner's static blocked_by semantics (BlockedBy is append-only at
// the planner layer) and avoids the race where a cleanup hook removes
// the edge after the holder completes but before the eligibility walk
// re-reads the tree, briefly re-opening the conflict window. The cost is
// a stale row in BlockedBy after the conflict is resolved by completion;
// that row is harmless because eligibility checks treat completed
// blockers as cleared.
//
// Concurrency: conflictDetector holds no mutable state. DetectSiblingOverlap
// is read-only on the input slices and on the underlying service.
// InsertRuntimeBlockedBy serializes through Service.UpdateActionItem +
// Service.RaiseAttentionItem, both of which are guarded by the service's
// own mutation locks.

// SiblingOverlapKind is a closed enum classifying one overlap dimension.
type SiblingOverlapKind string

// SiblingOverlapKind values.
const (
	// SiblingOverlapFile reports that two siblings declare a shared entry
	// in their respective Paths slices.
	SiblingOverlapFile SiblingOverlapKind = "file"
	// SiblingOverlapPackage reports that two siblings declare a shared
	// entry in their respective Packages slices.
	SiblingOverlapPackage SiblingOverlapKind = "package"
)

// SiblingOverlap reports one (item, sibling, kind, value) tuple where two
// sibling action items share a declared path or package. HasExplicitBlockedBy
// is true iff `item.Metadata.BlockedBy` already contains `siblingID` — in
// that case the planner has wired the dependency explicitly and the
// caller skips InsertRuntimeBlockedBy.
type SiblingOverlap struct {
	// SiblingID is the ID of the sibling action item that shares the
	// overlap value. Always non-empty in detector output.
	SiblingID string
	// OverlapKind classifies the overlap dimension (file or package).
	OverlapKind SiblingOverlapKind
	// OverlapValue carries the conflicting path (kind=file) or package
	// import path (kind=package). Treated as opaque by the detector — no
	// canonicalization beyond the trim/dedupe domain.NewActionItem
	// already applies on create.
	OverlapValue string
	// HasExplicitBlockedBy is true when the planner has already wired an
	// explicit blocked_by edge from item to siblingID. The caller treats
	// this as "already covered, no runtime insert needed".
	HasExplicitBlockedBy bool
}

// conflictDetectorService is the narrow consumer-side view the conflict
// detector uses to mutate action items and raise attention rows. *app.Service
// satisfies this interface; the test suite injects a deterministic stub so
// scenarios run without the full service + repository graph.
//
// Method names mirror Service exactly so the production binding is a
// trivial assignment in NewDispatcher (wired in 4a.23 along with the rest
// of the dispatcher graph).
type conflictDetectorService interface {
	UpdateActionItem(ctx context.Context, in app.UpdateActionItemInput) (domain.ActionItem, error)
	RaiseAttentionItem(ctx context.Context, in app.RaiseAttentionItemInput) (domain.AttentionItem, error)
}

// conflictDetector is the sibling-overlap analyzer + runtime-blocker writer
// described above.
type conflictDetector struct {
	svc conflictDetectorService
}

// newConflictDetector constructs a conflictDetector bound to svc. svc MUST
// be non-nil; callers wire the production *app.Service via the dispatcher
// constructor (deferred to 4a.23). The test suite passes a stub through
// this constructor.
func newConflictDetector(svc conflictDetectorService) *conflictDetector {
	return &conflictDetector{svc: svc}
}

// DetectSiblingOverlap scans the siblings slice for items that share the
// same ParentID as item AND overlap with item on at least one entry in
// Paths or Packages. The result is one SiblingOverlap entry per
// (siblingID, kind, value) combination, deduped — if two siblings overlap
// on multiple paths, each path emits its own entry; duplicates within
// item.Paths or sibling.Paths collapse to a single entry per kind/value
// pair under the same siblingID.
//
// Sibling filtering rules (defensive against bad inputs):
//   - siblings entries with a different ParentID than item are skipped
//     silently (caller is supposed to pre-filter, but the detector enforces
//     same-parent at the boundary so a misuse cannot leak cross-subtree
//     reports).
//   - siblings entries with the same ID as item itself are skipped (an
//     item cannot conflict with itself).
//   - siblings with empty Paths AND empty Packages contribute no overlaps.
//
// Determinism: results are sorted by (SiblingID, OverlapKind, OverlapValue)
// so callers and tests can compare slices without re-sorting.
//
// Read-only on the tree: DetectSiblingOverlap never calls the service. The
// ctx argument is reserved for future cross-subtree expansion (Drop 4b)
// where a service lookup is required to resolve sibling sets beyond the
// caller-provided slice.
//
// Empty inputs: an empty siblings slice returns a nil slice and a nil
// error. An item with empty Paths AND empty Packages can never produce
// overlaps regardless of siblings, so the detector short-circuits to nil.
func (d *conflictDetector) DetectSiblingOverlap(_ context.Context, item domain.ActionItem, siblings []domain.ActionItem) ([]SiblingOverlap, error) {
	if d == nil {
		return nil, fmt.Errorf("%w: conflict detector is nil", ErrInvalidDispatcherConfig)
	}
	if len(siblings) == 0 || (len(item.Paths) == 0 && len(item.Packages) == 0) {
		return nil, nil
	}

	// Build set views of item's declared paths/packages. Using maps keeps
	// the per-sibling intersection O(N+M) in path counts rather than O(N*M).
	itemPaths := make(map[string]struct{}, len(item.Paths))
	for _, p := range item.Paths {
		itemPaths[p] = struct{}{}
	}
	itemPackages := make(map[string]struct{}, len(item.Packages))
	for _, p := range item.Packages {
		itemPackages[p] = struct{}{}
	}

	// blockedBy lookup: an item already explicitly blocked by sibling
	// reports HasExplicitBlockedBy=true and the caller skips the runtime
	// insert.
	explicit := make(map[string]struct{}, len(item.Metadata.BlockedBy))
	for _, id := range item.Metadata.BlockedBy {
		explicit[strings.TrimSpace(id)] = struct{}{}
	}

	// dedupe key = siblingID + "\x00" + kind + "\x00" + value.
	seen := make(map[string]struct{})
	var overlaps []SiblingOverlap

	for _, sibling := range siblings {
		if sibling.ID == item.ID {
			continue
		}
		if sibling.ParentID != item.ParentID {
			continue
		}
		_, hasBlock := explicit[sibling.ID]

		for _, sp := range sibling.Paths {
			if _, ok := itemPaths[sp]; !ok {
				continue
			}
			key := sibling.ID + "\x00" + string(SiblingOverlapFile) + "\x00" + sp
			if _, dup := seen[key]; dup {
				continue
			}
			seen[key] = struct{}{}
			overlaps = append(overlaps, SiblingOverlap{
				SiblingID:            sibling.ID,
				OverlapKind:          SiblingOverlapFile,
				OverlapValue:         sp,
				HasExplicitBlockedBy: hasBlock,
			})
		}
		for _, sp := range sibling.Packages {
			if _, ok := itemPackages[sp]; !ok {
				continue
			}
			key := sibling.ID + "\x00" + string(SiblingOverlapPackage) + "\x00" + sp
			if _, dup := seen[key]; dup {
				continue
			}
			seen[key] = struct{}{}
			overlaps = append(overlaps, SiblingOverlap{
				SiblingID:            sibling.ID,
				OverlapKind:          SiblingOverlapPackage,
				OverlapValue:         sp,
				HasExplicitBlockedBy: hasBlock,
			})
		}
	}

	sort.Slice(overlaps, func(i, j int) bool {
		if overlaps[i].SiblingID != overlaps[j].SiblingID {
			return overlaps[i].SiblingID < overlaps[j].SiblingID
		}
		if overlaps[i].OverlapKind != overlaps[j].OverlapKind {
			return overlaps[i].OverlapKind < overlaps[j].OverlapKind
		}
		return overlaps[i].OverlapValue < overlaps[j].OverlapValue
	})
	return overlaps, nil
}

// TieBreakSibling returns the canonical "earlier" sibling between a and b
// per acceptance §8: lower Position wins; on equal Position, lexically
// smaller ID wins. The caller uses the result to decide which sibling
// receives the runtime blocked_by entry pointing at the other (the LATER
// sibling is updated; the EARLIER one runs first).
//
// Both inputs must share the same ParentID for the comparison to be
// meaningful — TieBreakSibling does NOT enforce same-parent because the
// detector already filters by ParentID before this function is reachable.
// Callers that build their own pair (e.g. a future cross-subtree sweep)
// own the precondition.
func TieBreakSibling(a, b domain.ActionItem) domain.ActionItem {
	if a.Position != b.Position {
		if a.Position < b.Position {
			return a
		}
		return b
	}
	if a.ID < b.ID {
		return a
	}
	return b
}

// InsertRuntimeBlockedBy adds siblingID to item.Metadata.BlockedBy and
// raises an AttentionKindBlocker attention row scoped to the action item.
// The reason argument carries the human-readable summary that lands in
// the attention item's Summary field.
//
// Idempotency (acceptance §4): if siblingID is already present in
// item.Metadata.BlockedBy, the call is a no-op — neither UpdateActionItem
// nor RaiseAttentionItem is invoked, and the returned error is nil. This
// matches the same-holder semantics of fileLockManager.Acquire so a
// caller can retry the conflict-detection sweep without accumulating
// duplicate attention rows.
//
// Empty siblingID, empty item.ID, or nil receiver returns
// ErrInvalidDispatcherConfig wrapped with the offending field name.
//
// Update + attention coupling: the BlockedBy update and the attention raise
// are NOT atomic. UpdateActionItem runs first; if it succeeds, the attention
// raise follows. If RaiseAttentionItem fails the runtime blocker is still
// in place (the dependency edge is the load-bearing part) and the error
// is returned wrapped — the caller can decide whether to retry attention
// raise specifically. This trades atomicity for a simpler call shape: the
// orchestrator already polls attention items, so a missed raise surfaces
// on the next sweep.
//
// Trimming: siblingID is whitespace-trimmed before equality check + insert.
// Reason is passed through as-is to RaiseAttentionItem (which performs its
// own normalization at the domain layer).
func (d *conflictDetector) InsertRuntimeBlockedBy(ctx context.Context, item domain.ActionItem, siblingID, reason string) error {
	if d == nil || d.svc == nil {
		return fmt.Errorf("%w: conflict detector service is nil", ErrInvalidDispatcherConfig)
	}
	itemID := strings.TrimSpace(item.ID)
	if itemID == "" {
		return fmt.Errorf("%w: action item id is empty", ErrInvalidDispatcherConfig)
	}
	siblingID = strings.TrimSpace(siblingID)
	if siblingID == "" {
		return fmt.Errorf("%w: sibling id is empty", ErrInvalidDispatcherConfig)
	}

	for _, existing := range item.Metadata.BlockedBy {
		if strings.TrimSpace(existing) == siblingID {
			// Already present — true no-op (no update, no attention raise).
			return nil
		}
	}

	updatedBlockedBy := make([]string, 0, len(item.Metadata.BlockedBy)+1)
	updatedBlockedBy = append(updatedBlockedBy, item.Metadata.BlockedBy...)
	updatedBlockedBy = append(updatedBlockedBy, siblingID)

	updatedMeta := item.Metadata
	updatedMeta.BlockedBy = updatedBlockedBy

	if _, err := d.svc.UpdateActionItem(ctx, app.UpdateActionItemInput{
		ActionItemID: itemID,
		Metadata:     &updatedMeta,
		UpdatedType:  domain.ActorTypeSystem,
	}); err != nil {
		return fmt.Errorf("conflict detector: update blocked_by for %q: %w", itemID, err)
	}

	if _, err := d.svc.RaiseAttentionItem(ctx, app.RaiseAttentionItemInput{
		Level: domain.LevelTupleInput{
			ProjectID: item.ProjectID,
			ScopeType: domain.ScopeLevelActionItem,
			ScopeID:   itemID,
		},
		Kind:    domain.AttentionKindBlocker,
		Summary: fmt.Sprintf("Runtime blocked_by inserted: %s blocks %s", siblingID, itemID),
		BodyMarkdown: fmt.Sprintf(
			"Sibling overlap detected by the dispatcher's conflict detector.\n\n"+
				"- Blocker: %s\n- Blocked: %s\n- Reason: %s\n",
			siblingID, itemID, reason,
		),
		CreatedType: domain.ActorTypeSystem,
	}); err != nil {
		return fmt.Errorf("conflict detector: raise attention for %q blocked by %q: %w", itemID, siblingID, err)
	}
	return nil
}
