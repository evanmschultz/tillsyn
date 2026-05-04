package dispatcher

import (
	"context"
	"errors"
	"reflect"
	"sort"
	"testing"

	"github.com/evanmschultz/tillsyn/internal/app"
	"github.com/evanmschultz/tillsyn/internal/domain"
)

// stubConflictService is the deterministic test fixture for conflictDetector.
// Tests construct one and assert against the captured update + attention
// inputs to verify InsertRuntimeBlockedBy's contract without standing up a
// full *app.Service graph.
type stubConflictService struct {
	updateCalls  int
	lastUpdate   app.UpdateActionItemInput
	updateErr    error
	updateResult domain.ActionItem

	attentionCalls  int
	lastAttention   app.RaiseAttentionItemInput
	attentionErr    error
	attentionResult domain.AttentionItem
}

// UpdateActionItem records the input + returns the configured fixture (or
// error). The real Service path performs guards + a repo write; the stub
// captures only the contract surface the conflict detector depends on.
func (s *stubConflictService) UpdateActionItem(_ context.Context, in app.UpdateActionItemInput) (domain.ActionItem, error) {
	s.updateCalls++
	s.lastUpdate = in
	if s.updateErr != nil {
		return domain.ActionItem{}, s.updateErr
	}
	return s.updateResult, nil
}

// RaiseAttentionItem records the input + returns the configured fixture (or
// error). Mirrors UpdateActionItem above for the attention raise path.
func (s *stubConflictService) RaiseAttentionItem(_ context.Context, in app.RaiseAttentionItemInput) (domain.AttentionItem, error) {
	s.attentionCalls++
	s.lastAttention = in
	if s.attentionErr != nil {
		return domain.AttentionItem{}, s.attentionErr
	}
	return s.attentionResult, nil
}

// TestDetectorFindsFileOverlapBetweenSiblings asserts the happy-path file
// overlap detection: two siblings under the same parent declaring the same
// path produce one SiblingOverlap entry of kind=file with the shared path.
func TestDetectorFindsFileOverlapBetweenSiblings(t *testing.T) {
	t.Parallel()

	item := domain.ActionItem{
		ID:       "candidate",
		ParentID: "parent-1",
		Paths:    []string{"internal/app/dispatcher/walker.go"},
		Packages: []string{"internal/app/dispatcher"},
	}
	siblings := []domain.ActionItem{
		{
			ID:       "sibling",
			ParentID: "parent-1",
			Paths:    []string{"internal/app/dispatcher/walker.go"},
			Packages: []string{"internal/app/dispatcher"},
		},
	}
	d := newConflictDetector(&stubConflictService{})

	overlaps, err := d.DetectSiblingOverlap(context.Background(), item, siblings)
	if err != nil {
		t.Fatalf("DetectSiblingOverlap() error = %v, want nil", err)
	}
	// Both file and package overlap should appear; assert file is present
	// and check its shape.
	var got *SiblingOverlap
	for i := range overlaps {
		if overlaps[i].OverlapKind == SiblingOverlapFile {
			got = &overlaps[i]
			break
		}
	}
	if got == nil {
		t.Fatalf("DetectSiblingOverlap() returned no file overlap: %+v", overlaps)
	}
	want := SiblingOverlap{
		SiblingID:            "sibling",
		OverlapKind:          SiblingOverlapFile,
		OverlapValue:         "internal/app/dispatcher/walker.go",
		HasExplicitBlockedBy: false,
	}
	if *got != want {
		t.Fatalf("file overlap mismatch: got %+v, want %+v", *got, want)
	}
}

// TestDetectorFindsPackageOverlapBetweenSiblings asserts package-only
// overlap (disjoint Paths, shared Package) produces a kind=package entry.
func TestDetectorFindsPackageOverlapBetweenSiblings(t *testing.T) {
	t.Parallel()

	item := domain.ActionItem{
		ID:       "candidate",
		ParentID: "parent-1",
		Paths:    []string{"internal/app/dispatcher/walker.go"},
		Packages: []string{"internal/app/dispatcher"},
	}
	siblings := []domain.ActionItem{
		{
			ID:       "sibling",
			ParentID: "parent-1",
			// Different file → no file overlap; same package → package overlap.
			Paths:    []string{"internal/app/dispatcher/spawn.go"},
			Packages: []string{"internal/app/dispatcher"},
		},
	}
	d := newConflictDetector(&stubConflictService{})

	overlaps, err := d.DetectSiblingOverlap(context.Background(), item, siblings)
	if err != nil {
		t.Fatalf("DetectSiblingOverlap() error = %v, want nil", err)
	}
	if len(overlaps) != 1 {
		t.Fatalf("DetectSiblingOverlap() len = %d, want 1: %+v", len(overlaps), overlaps)
	}
	want := SiblingOverlap{
		SiblingID:            "sibling",
		OverlapKind:          SiblingOverlapPackage,
		OverlapValue:         "internal/app/dispatcher",
		HasExplicitBlockedBy: false,
	}
	if overlaps[0] != want {
		t.Fatalf("package overlap mismatch: got %+v, want %+v", overlaps[0], want)
	}
}

// TestDetectorIgnoresNonSiblings asserts that items with a different
// ParentID are filtered out even when they declare overlapping paths /
// packages. Cross-subtree overlap is a Drop 4b concern.
func TestDetectorIgnoresNonSiblings(t *testing.T) {
	t.Parallel()

	item := domain.ActionItem{
		ID:       "candidate",
		ParentID: "parent-1",
		Paths:    []string{"shared/path.go"},
		Packages: []string{"shared/pkg"},
	}
	siblings := []domain.ActionItem{
		{
			ID:       "cousin",
			ParentID: "parent-2", // different parent → filtered.
			Paths:    []string{"shared/path.go"},
			Packages: []string{"shared/pkg"},
		},
	}
	d := newConflictDetector(&stubConflictService{})

	overlaps, err := d.DetectSiblingOverlap(context.Background(), item, siblings)
	if err != nil {
		t.Fatalf("DetectSiblingOverlap() error = %v, want nil", err)
	}
	if len(overlaps) != 0 {
		t.Fatalf("expected zero overlaps for non-sibling, got %d: %+v", len(overlaps), overlaps)
	}
}

// TestDetectorReportsExplicitBlockedByCovered asserts that an overlap is
// still reported (the planner+QA pipeline wants visibility) but that
// HasExplicitBlockedBy is true so the caller knows the runtime insert is
// unnecessary.
func TestDetectorReportsExplicitBlockedByCovered(t *testing.T) {
	t.Parallel()

	item := domain.ActionItem{
		ID:       "candidate",
		ParentID: "parent-1",
		Paths:    []string{"internal/app/dispatcher/walker.go"},
		Packages: []string{"internal/app/dispatcher"},
		Metadata: domain.ActionItemMetadata{
			BlockedBy: []string{"sibling"},
		},
	}
	siblings := []domain.ActionItem{
		{
			ID:       "sibling",
			ParentID: "parent-1",
			Paths:    []string{"internal/app/dispatcher/walker.go"},
			Packages: []string{"internal/app/dispatcher"},
		},
	}
	d := newConflictDetector(&stubConflictService{})

	overlaps, err := d.DetectSiblingOverlap(context.Background(), item, siblings)
	if err != nil {
		t.Fatalf("DetectSiblingOverlap() error = %v, want nil", err)
	}
	if len(overlaps) == 0 {
		t.Fatalf("expected overlaps to be reported with HasExplicitBlockedBy=true, got none")
	}
	for _, o := range overlaps {
		if !o.HasExplicitBlockedBy {
			t.Fatalf("expected HasExplicitBlockedBy=true for sibling=%q kind=%q value=%q, got false",
				o.SiblingID, o.OverlapKind, o.OverlapValue)
		}
	}
}

// TestInsertRuntimeBlockedByIsIdempotent asserts that calling
// InsertRuntimeBlockedBy with a siblingID already present in
// item.Metadata.BlockedBy is a true no-op: zero UpdateActionItem calls,
// zero RaiseAttentionItem calls, nil error.
func TestInsertRuntimeBlockedByIsIdempotent(t *testing.T) {
	t.Parallel()

	stub := &stubConflictService{}
	d := newConflictDetector(stub)
	item := domain.ActionItem{
		ID:        "candidate",
		ProjectID: "proj-1",
		ParentID:  "parent-1",
		Metadata: domain.ActionItemMetadata{
			BlockedBy: []string{"sibling"},
		},
	}

	if err := d.InsertRuntimeBlockedBy(context.Background(), item, "sibling", "test reason"); err != nil {
		t.Fatalf("InsertRuntimeBlockedBy() error = %v, want nil (idempotent no-op)", err)
	}
	if stub.updateCalls != 0 {
		t.Fatalf("UpdateActionItem call count = %d, want 0 on idempotent path", stub.updateCalls)
	}
	if stub.attentionCalls != 0 {
		t.Fatalf("RaiseAttentionItem call count = %d, want 0 on idempotent path", stub.attentionCalls)
	}
}

// TestInsertRuntimeBlockedByPostsAttentionItem asserts the happy-path:
// a fresh insertion calls UpdateActionItem with the augmented BlockedBy
// slice AND raises an AttentionKindBlocker scoped to the action item.
func TestInsertRuntimeBlockedByPostsAttentionItem(t *testing.T) {
	t.Parallel()

	stub := &stubConflictService{}
	d := newConflictDetector(stub)
	item := domain.ActionItem{
		ID:        "candidate",
		ProjectID: "proj-1",
		ParentID:  "parent-1",
		Metadata: domain.ActionItemMetadata{
			BlockedBy: []string{"existing-blocker"},
		},
	}

	if err := d.InsertRuntimeBlockedBy(context.Background(), item, "sibling", "shared file walker.go"); err != nil {
		t.Fatalf("InsertRuntimeBlockedBy() error = %v, want nil", err)
	}
	if stub.updateCalls != 1 {
		t.Fatalf("UpdateActionItem call count = %d, want 1", stub.updateCalls)
	}
	if stub.attentionCalls != 1 {
		t.Fatalf("RaiseAttentionItem call count = %d, want 1", stub.attentionCalls)
	}

	// UpdateActionItem must carry the augmented BlockedBy slice with the
	// pre-existing blocker preserved AND the new sibling appended.
	if stub.lastUpdate.ActionItemID != "candidate" {
		t.Fatalf("UpdateActionItem ActionItemID = %q, want %q", stub.lastUpdate.ActionItemID, "candidate")
	}
	if stub.lastUpdate.Metadata == nil {
		t.Fatalf("UpdateActionItem Metadata is nil; want non-nil with augmented BlockedBy")
	}
	got := append([]string(nil), stub.lastUpdate.Metadata.BlockedBy...)
	sort.Strings(got)
	want := []string{"existing-blocker", "sibling"}
	sort.Strings(want)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("UpdateActionItem BlockedBy = %v, want %v", got, want)
	}

	// Attention raise must target the action item scope and use Blocker kind.
	if stub.lastAttention.Kind != domain.AttentionKindBlocker {
		t.Fatalf("attention kind = %q, want %q", stub.lastAttention.Kind, domain.AttentionKindBlocker)
	}
	if stub.lastAttention.Level.ProjectID != "proj-1" {
		t.Fatalf("attention level.ProjectID = %q, want %q", stub.lastAttention.Level.ProjectID, "proj-1")
	}
	if stub.lastAttention.Level.ScopeType != domain.ScopeLevelActionItem {
		t.Fatalf("attention level.ScopeType = %q, want %q", stub.lastAttention.Level.ScopeType, domain.ScopeLevelActionItem)
	}
	if stub.lastAttention.Level.ScopeID != "candidate" {
		t.Fatalf("attention level.ScopeID = %q, want %q", stub.lastAttention.Level.ScopeID, "candidate")
	}
	if stub.lastAttention.Summary == "" {
		t.Fatalf("attention summary is empty; want non-empty")
	}
}

// TestDetectorTieBreakByPositionThenID covers acceptance §8: two equally-
// eligible siblings tie-break by Position ascending, then by ID lexically.
func TestDetectorTieBreakByPositionThenID(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		a, b domain.ActionItem
		want string
	}{
		{
			name: "lower position wins",
			a:    domain.ActionItem{ID: "second", Position: 5},
			b:    domain.ActionItem{ID: "first", Position: 1},
			want: "first",
		},
		{
			name: "equal position falls back to lex id",
			a:    domain.ActionItem{ID: "z-item", Position: 3},
			b:    domain.ActionItem{ID: "a-item", Position: 3},
			want: "a-item",
		},
		{
			name: "lower position wins when ids reverse-sorted",
			a:    domain.ActionItem{ID: "a-item", Position: 9},
			b:    domain.ActionItem{ID: "z-item", Position: 1},
			want: "z-item",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := TieBreakSibling(tc.a, tc.b)
			if got.ID != tc.want {
				t.Fatalf("TieBreakSibling() = %q, want %q (a=%+v b=%+v)", got.ID, tc.want, tc.a, tc.b)
			}
		})
	}
}

// TestDetectorEmptyInputsAreNoop asserts the defensive paths: an empty
// siblings slice, or an item with neither Paths nor Packages, returns
// (nil, nil) without invoking the service.
func TestDetectorEmptyInputsAreNoop(t *testing.T) {
	t.Parallel()

	d := newConflictDetector(&stubConflictService{})

	overlaps, err := d.DetectSiblingOverlap(context.Background(), domain.ActionItem{ID: "x", ParentID: "p"}, nil)
	if err != nil {
		t.Fatalf("nil siblings: error = %v, want nil", err)
	}
	if overlaps != nil {
		t.Fatalf("nil siblings: overlaps = %+v, want nil", overlaps)
	}

	overlaps, err = d.DetectSiblingOverlap(
		context.Background(),
		domain.ActionItem{ID: "x", ParentID: "p"},
		[]domain.ActionItem{{ID: "y", ParentID: "p", Paths: []string{"a"}, Packages: []string{"b"}}},
	)
	if err != nil {
		t.Fatalf("empty item paths/packages: error = %v, want nil", err)
	}
	if overlaps != nil {
		t.Fatalf("empty item paths/packages: overlaps = %+v, want nil", overlaps)
	}
}

// TestDetectorSelfIsNotSibling asserts the defensive filter: an entry in
// siblings sharing item.ID is skipped even when paths overlap (an item
// cannot conflict with itself).
func TestDetectorSelfIsNotSibling(t *testing.T) {
	t.Parallel()

	item := domain.ActionItem{
		ID:       "candidate",
		ParentID: "parent-1",
		Paths:    []string{"shared.go"},
		Packages: []string{"shared"},
	}
	siblings := []domain.ActionItem{
		{
			ID:       "candidate", // same ID → must be skipped.
			ParentID: "parent-1",
			Paths:    []string{"shared.go"},
			Packages: []string{"shared"},
		},
	}
	d := newConflictDetector(&stubConflictService{})

	overlaps, err := d.DetectSiblingOverlap(context.Background(), item, siblings)
	if err != nil {
		t.Fatalf("DetectSiblingOverlap() error = %v, want nil", err)
	}
	if len(overlaps) != 0 {
		t.Fatalf("expected zero overlaps for self-as-sibling, got %d: %+v", len(overlaps), overlaps)
	}
}

// TestInsertRuntimeBlockedByEmptyIDsRejected asserts the defensive guard:
// empty item.ID or empty siblingID surfaces ErrInvalidDispatcherConfig
// without invoking the service.
func TestInsertRuntimeBlockedByEmptyIDsRejected(t *testing.T) {
	t.Parallel()

	stub := &stubConflictService{}
	d := newConflictDetector(stub)

	if err := d.InsertRuntimeBlockedBy(context.Background(), domain.ActionItem{ID: ""}, "sibling", "r"); !errors.Is(err, ErrInvalidDispatcherConfig) {
		t.Fatalf("empty item.ID: error = %v, want ErrInvalidDispatcherConfig", err)
	}
	if err := d.InsertRuntimeBlockedBy(context.Background(), domain.ActionItem{ID: "x"}, "  ", "r"); !errors.Is(err, ErrInvalidDispatcherConfig) {
		t.Fatalf("empty siblingID: error = %v, want ErrInvalidDispatcherConfig", err)
	}
	if stub.updateCalls != 0 || stub.attentionCalls != 0 {
		t.Fatalf("expected zero service calls on validation rejection, got update=%d attention=%d",
			stub.updateCalls, stub.attentionCalls)
	}
}
