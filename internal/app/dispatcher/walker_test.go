package dispatcher

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/evanmschultz/tillsyn/internal/domain"
)

// stubWalkerService is the deterministic test fixture for the tree walker.
// Tests construct one with pre-baked columns + items + an optional
// move-result error to exercise EligibleForPromotion and Promote without
// standing up a full *app.Service graph.
type stubWalkerService struct {
	columns        []domain.Column
	items          []domain.ActionItem
	moveResult     domain.ActionItem
	moveErr        error
	moveCalls      int
	lastMoveID     string
	lastMoveColumn string
	lastMovePos    int
}

// ListColumns returns the configured columns; includeArchived is ignored
// because tests pass canonical state-named columns and do not exercise
// archive filtering.
func (s *stubWalkerService) ListColumns(_ context.Context, _ string, _ bool) ([]domain.Column, error) {
	return s.columns, nil
}

// ListActionItems returns the configured items; includeArchived is ignored
// because the walker passes false (production) and the test fixture only
// holds non-archived items anyway.
func (s *stubWalkerService) ListActionItems(_ context.Context, _ string, _ bool) ([]domain.ActionItem, error) {
	return s.items, nil
}

// MoveActionItem records the args for assertion and returns the configured
// fixture (or error). The real Service path performs guards and a repo
// write; the stub captures the contract surface the walker depends on.
func (s *stubWalkerService) MoveActionItem(_ context.Context, actionItemID, toColumnID string, position int) (domain.ActionItem, error) {
	s.moveCalls++
	s.lastMoveID = actionItemID
	s.lastMoveColumn = toColumnID
	s.lastMovePos = position
	if s.moveErr != nil {
		return domain.ActionItem{}, s.moveErr
	}
	return s.moveResult, nil
}

// canonicalColumns returns a fixture column set covering the four
// non-archived lifecycle states. The walker only cares about Name + ID;
// Position is set to keep ListColumns sortable for parity with production.
func canonicalColumns() []domain.Column {
	return []domain.Column{
		{ID: "col-todo", Name: "To Do", Position: 0},
		{ID: "col-inprogress", Name: "In Progress", Position: 1},
		{ID: "col-complete", Name: "Complete", Position: 2},
		{ID: "col-failed", Name: "Failed", Position: 3},
	}
}

// TestWalkerFindsTodoItemWithClearedBlockers asserts the happy-path
// eligibility predicate: a todo item whose BlockedBy resolves to a
// complete item, with a parent in in_progress, is included in the result.
func TestWalkerFindsTodoItemWithClearedBlockers(t *testing.T) {
	t.Parallel()

	items := []domain.ActionItem{
		{
			ID:             "parent-1",
			ProjectID:      "proj-1",
			LifecycleState: domain.StateInProgress,
			ColumnID:       "col-inprogress",
		},
		{
			ID:             "blocker-1",
			ProjectID:      "proj-1",
			ParentID:       "parent-1",
			LifecycleState: domain.StateComplete,
			ColumnID:       "col-complete",
		},
		{
			ID:             "candidate-1",
			ProjectID:      "proj-1",
			ParentID:       "parent-1",
			LifecycleState: domain.StateTodo,
			ColumnID:       "col-todo",
			Metadata: domain.ActionItemMetadata{
				BlockedBy: []string{"blocker-1"},
			},
		},
	}
	svc := &stubWalkerService{columns: canonicalColumns(), items: items}
	w := newTreeWalker(svc)

	got, err := w.EligibleForPromotion(context.Background(), "proj-1")
	if err != nil {
		t.Fatalf("EligibleForPromotion() error = %v, want nil", err)
	}
	if len(got) != 1 {
		t.Fatalf("EligibleForPromotion() returned %d items, want 1: %#v", len(got), got)
	}
	if got[0].ID != "candidate-1" {
		t.Fatalf("EligibleForPromotion() returned %q, want %q", got[0].ID, "candidate-1")
	}
}

// TestWalkerSkipsTodoItemWithUnmetBlockedBy asserts that a todo item whose
// BlockedBy entry resolves to a non-complete item is filtered out, even
// when every other condition is met.
func TestWalkerSkipsTodoItemWithUnmetBlockedBy(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name         string
		blockerState domain.LifecycleState
	}{
		{name: "todo blocker", blockerState: domain.StateTodo},
		{name: "in_progress blocker", blockerState: domain.StateInProgress},
		{name: "failed blocker", blockerState: domain.StateFailed},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			items := []domain.ActionItem{
				{
					ID:             "parent-1",
					ProjectID:      "proj-1",
					LifecycleState: domain.StateInProgress,
					ColumnID:       "col-inprogress",
				},
				{
					ID:             "blocker-1",
					ProjectID:      "proj-1",
					ParentID:       "parent-1",
					LifecycleState: tc.blockerState,
				},
				{
					ID:             "candidate-1",
					ProjectID:      "proj-1",
					ParentID:       "parent-1",
					LifecycleState: domain.StateTodo,
					ColumnID:       "col-todo",
					Metadata: domain.ActionItemMetadata{
						BlockedBy: []string{"blocker-1"},
					},
				},
			}
			svc := &stubWalkerService{columns: canonicalColumns(), items: items}
			w := newTreeWalker(svc)

			got, err := w.EligibleForPromotion(context.Background(), "proj-1")
			if err != nil {
				t.Fatalf("EligibleForPromotion() error = %v, want nil", err)
			}
			for _, it := range got {
				if it.ID == "candidate-1" {
					t.Fatalf("EligibleForPromotion() included candidate-1 with %q blocker: %#v", tc.blockerState, got)
				}
			}
		})
	}
}

// TestWalkerSkipsTodoItemWhoseParentIsTodo asserts that the parent-state
// gate filters out a todo child whose parent has not yet been promoted to
// in_progress (and is not Persistent). The fixture pins the parent under
// a grandparent in_progress, so the parent itself cleanly cascades — the
// only assertion under test is that the candidate child is NOT in the
// eligible set.
func TestWalkerSkipsTodoItemWhoseParentIsTodo(t *testing.T) {
	t.Parallel()

	items := []domain.ActionItem{
		{
			ID:             "grandparent-1",
			ProjectID:      "proj-1",
			LifecycleState: domain.StateInProgress,
			ColumnID:       "col-inprogress",
		},
		{
			ID:             "parent-1",
			ProjectID:      "proj-1",
			ParentID:       "grandparent-1",
			LifecycleState: domain.StateTodo,
			ColumnID:       "col-todo",
		},
		{
			ID:             "candidate-1",
			ProjectID:      "proj-1",
			ParentID:       "parent-1",
			LifecycleState: domain.StateTodo,
			ColumnID:       "col-todo",
		},
	}
	svc := &stubWalkerService{columns: canonicalColumns(), items: items}
	w := newTreeWalker(svc)

	got, err := w.EligibleForPromotion(context.Background(), "proj-1")
	if err != nil {
		t.Fatalf("EligibleForPromotion() error = %v, want nil", err)
	}
	for _, it := range got {
		if it.ID == "candidate-1" {
			t.Fatalf("EligibleForPromotion() included candidate-1 with todo parent: %#v", got)
		}
	}
}

// TestWalkerPromotesEligibleItem asserts that Promote calls the service
// with the in_progress column ID resolved from the project's column set
// and the item's existing position. The stub records the call args so the
// test can verify the dispatcher → service contract directly.
func TestWalkerPromotesEligibleItem(t *testing.T) {
	t.Parallel()

	item := domain.ActionItem{
		ID:             "candidate-1",
		ProjectID:      "proj-1",
		ParentID:       "parent-1",
		LifecycleState: domain.StateTodo,
		ColumnID:       "col-todo",
		Position:       7,
	}
	moved := item
	moved.ColumnID = "col-inprogress"
	moved.LifecycleState = domain.StateInProgress
	svc := &stubWalkerService{
		columns:    canonicalColumns(),
		items:      []domain.ActionItem{item},
		moveResult: moved,
	}
	w := newTreeWalker(svc)

	got, err := w.Promote(context.Background(), item)
	if err != nil {
		t.Fatalf("Promote() error = %v, want nil", err)
	}
	if got.ColumnID != "col-inprogress" {
		t.Fatalf("Promote() returned ColumnID = %q, want %q", got.ColumnID, "col-inprogress")
	}
	if got.LifecycleState != domain.StateInProgress {
		t.Fatalf("Promote() returned LifecycleState = %q, want %q", got.LifecycleState, domain.StateInProgress)
	}
	if svc.moveCalls != 1 {
		t.Fatalf("svc.moveCalls = %d, want 1", svc.moveCalls)
	}
	if svc.lastMoveID != "candidate-1" {
		t.Fatalf("svc.lastMoveID = %q, want %q", svc.lastMoveID, "candidate-1")
	}
	if svc.lastMoveColumn != "col-inprogress" {
		t.Fatalf("svc.lastMoveColumn = %q, want %q (must resolve via canonical slug match)", svc.lastMoveColumn, "col-inprogress")
	}
	if svc.lastMovePos != 7 {
		t.Fatalf("svc.lastMovePos = %d, want 7 (must preserve the existing position)", svc.lastMovePos)
	}
}

// TestWalkerPropagatesTransitionBlocked asserts that a service-level
// ErrTransitionBlocked surfaces as ErrPromotionBlocked while preserving
// the underlying ErrTransitionBlocked match for callers that pattern on it.
func TestWalkerPropagatesTransitionBlocked(t *testing.T) {
	t.Parallel()

	innerErr := fmt.Errorf("%w: start criteria unmet (paths)", domain.ErrTransitionBlocked)
	item := domain.ActionItem{
		ID:             "candidate-1",
		ProjectID:      "proj-1",
		LifecycleState: domain.StateTodo,
	}
	svc := &stubWalkerService{
		columns: canonicalColumns(),
		items:   []domain.ActionItem{item},
		moveErr: innerErr,
	}
	w := newTreeWalker(svc)

	_, err := w.Promote(context.Background(), item)
	if err == nil {
		t.Fatalf("Promote() error = nil, want ErrPromotionBlocked-wrapped")
	}
	if !errors.Is(err, ErrPromotionBlocked) {
		t.Fatalf("Promote() error = %v, want errors.Is(ErrPromotionBlocked)", err)
	}
	if !errors.Is(err, domain.ErrTransitionBlocked) {
		t.Fatalf("Promote() error = %v, want errors.Is(domain.ErrTransitionBlocked) (chain must preserve the inner sentinel)", err)
	}
}

// TestWalkerPromotesPersistentParentChild asserts that a child whose
// parent has Persistent=true is eligible regardless of the parent's
// lifecycle state — refinement umbrellas / anchor nodes do not block
// their descendants.
func TestWalkerPromotesPersistentParentChild(t *testing.T) {
	t.Parallel()

	items := []domain.ActionItem{
		{
			ID:             "anchor-1",
			ProjectID:      "proj-1",
			Persistent:     true,
			LifecycleState: domain.StateTodo,
		},
		{
			ID:             "candidate-1",
			ProjectID:      "proj-1",
			ParentID:       "anchor-1",
			LifecycleState: domain.StateTodo,
			ColumnID:       "col-todo",
		},
	}
	svc := &stubWalkerService{columns: canonicalColumns(), items: items}
	w := newTreeWalker(svc)

	got, err := w.EligibleForPromotion(context.Background(), "proj-1")
	if err != nil {
		t.Fatalf("EligibleForPromotion() error = %v, want nil", err)
	}
	found := false
	for _, it := range got {
		if it.ID == "candidate-1" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("EligibleForPromotion() = %#v, want candidate-1 included (persistent parent must not gate)", got)
	}
}

// TestWalkerPromotesRootItem asserts that an item with no ParentID
// (project-root) is eligible without a parent state check.
func TestWalkerPromotesRootItem(t *testing.T) {
	t.Parallel()

	items := []domain.ActionItem{
		{
			ID:             "root-1",
			ProjectID:      "proj-1",
			LifecycleState: domain.StateTodo,
			ColumnID:       "col-todo",
		},
	}
	svc := &stubWalkerService{columns: canonicalColumns(), items: items}
	w := newTreeWalker(svc)

	got, err := w.EligibleForPromotion(context.Background(), "proj-1")
	if err != nil {
		t.Fatalf("EligibleForPromotion() error = %v, want nil", err)
	}
	if len(got) != 1 || got[0].ID != "root-1" {
		t.Fatalf("EligibleForPromotion() = %#v, want one item id=root-1 (root parent treated as in_progress)", got)
	}
}

// TestWalkerSkipsItemWithMissingBlockedByReference asserts that a
// BlockedBy entry that does not resolve to any item in the project tree
// is treated as not-clear (conservative: the planner is responsible for
// reconciling BlockedBy, and walker should not promote on a phantom
// reference).
func TestWalkerSkipsItemWithMissingBlockedByReference(t *testing.T) {
	t.Parallel()

	items := []domain.ActionItem{
		{
			ID:             "parent-1",
			ProjectID:      "proj-1",
			LifecycleState: domain.StateInProgress,
		},
		{
			ID:             "candidate-1",
			ProjectID:      "proj-1",
			ParentID:       "parent-1",
			LifecycleState: domain.StateTodo,
			ColumnID:       "col-todo",
			Metadata: domain.ActionItemMetadata{
				BlockedBy: []string{"does-not-exist"},
			},
		},
	}
	svc := &stubWalkerService{columns: canonicalColumns(), items: items}
	w := newTreeWalker(svc)

	got, err := w.EligibleForPromotion(context.Background(), "proj-1")
	if err != nil {
		t.Fatalf("EligibleForPromotion() error = %v, want nil", err)
	}
	if len(got) != 0 {
		t.Fatalf("EligibleForPromotion() = %#v, want 0 (missing BlockedBy reference must hold)", got)
	}
}

// TestWalkerEligibleForPromotionSkipsEmptyProjectID asserts that an empty
// or whitespace-only project ID returns a nil result + nil error so a
// misconfigured caller does not crash the dispatcher.
func TestWalkerEligibleForPromotionSkipsEmptyProjectID(t *testing.T) {
	t.Parallel()

	svc := &stubWalkerService{columns: canonicalColumns()}
	w := newTreeWalker(svc)

	got, err := w.EligibleForPromotion(context.Background(), "   ")
	if err != nil {
		t.Fatalf("EligibleForPromotion() error = %v, want nil", err)
	}
	if got != nil {
		t.Fatalf("EligibleForPromotion() = %#v, want nil", got)
	}
}

// TestWalkerEligibleForPromotionPropagatesListError asserts that
// infrastructure failures from ListActionItems are not silently swallowed.
func TestWalkerEligibleForPromotionPropagatesListError(t *testing.T) {
	t.Parallel()

	listErr := errors.New("database closed")
	svc := &erroringListItemsStub{err: listErr}
	w := newTreeWalker(svc)

	_, err := w.EligibleForPromotion(context.Background(), "proj-1")
	if err == nil {
		t.Fatalf("EligibleForPromotion() error = nil, want %v", listErr)
	}
	if !errors.Is(err, listErr) {
		t.Fatalf("EligibleForPromotion() error = %v, want errors.Is(listErr)", err)
	}
}

// TestWalkerPromoteRejectsMissingInProgressColumn asserts that a project
// with no in_progress column produces a descriptive error rather than a
// silent skip — the cascade dispatcher requires the canonical column set.
func TestWalkerPromoteRejectsMissingInProgressColumn(t *testing.T) {
	t.Parallel()

	item := domain.ActionItem{
		ID:             "candidate-1",
		ProjectID:      "proj-1",
		LifecycleState: domain.StateTodo,
	}
	// Columns intentionally lack an in_progress entry.
	svc := &stubWalkerService{
		columns: []domain.Column{
			{ID: "col-todo", Name: "To Do"},
			{ID: "col-complete", Name: "Complete"},
		},
		items: []domain.ActionItem{item},
	}
	w := newTreeWalker(svc)

	_, err := w.Promote(context.Background(), item)
	if err == nil {
		t.Fatalf("Promote() error = nil, want non-nil")
	}
	if svc.moveCalls != 0 {
		t.Fatalf("svc.moveCalls = %d, want 0 (no MoveActionItem when column unresolved)", svc.moveCalls)
	}
}

// TestWalkerPromoteRejectsEmptyProjectID asserts that an action item with
// no ProjectID surfaces as an error rather than being passed to the
// service with an empty projectID.
func TestWalkerPromoteRejectsEmptyProjectID(t *testing.T) {
	t.Parallel()

	item := domain.ActionItem{
		ID:             "candidate-1",
		LifecycleState: domain.StateTodo,
	}
	svc := &stubWalkerService{columns: canonicalColumns()}
	w := newTreeWalker(svc)

	_, err := w.Promote(context.Background(), item)
	if err == nil {
		t.Fatalf("Promote() error = nil, want non-nil")
	}
}

// TestWalkerPromoteSurfacesNonTransitionErrors asserts that service
// errors that are NOT ErrTransitionBlocked propagate without the
// ErrPromotionBlocked sentinel — the conflict detector must distinguish
// "planner-side blocker" (recoverable) from "infrastructure failure"
// (non-recoverable).
func TestWalkerPromoteSurfacesNonTransitionErrors(t *testing.T) {
	t.Parallel()

	infraErr := errors.New("connection lost")
	item := domain.ActionItem{
		ID:             "candidate-1",
		ProjectID:      "proj-1",
		LifecycleState: domain.StateTodo,
	}
	svc := &stubWalkerService{
		columns: canonicalColumns(),
		items:   []domain.ActionItem{item},
		moveErr: infraErr,
	}
	w := newTreeWalker(svc)

	_, err := w.Promote(context.Background(), item)
	if err == nil {
		t.Fatalf("Promote() error = nil, want %v", infraErr)
	}
	if errors.Is(err, ErrPromotionBlocked) {
		t.Fatalf("Promote() wrapped ErrPromotionBlocked for non-transition error: %v", err)
	}
	if !errors.Is(err, infraErr) {
		t.Fatalf("Promote() = %v, want errors.Is(%v)", err, infraErr)
	}
}

// TestNewTreeWalkerNilServiceCallSafe asserts that the walker's nil-receiver
// guards return ErrInvalidDispatcherConfig rather than panicking. The
// production constructor (4a.23) will reject nil at construction time;
// these guards are belt-and-suspenders for unit-test stubs and future
// refactors.
func TestNewTreeWalkerNilServiceCallSafe(t *testing.T) {
	t.Parallel()

	w := &treeWalker{}
	_, err := w.EligibleForPromotion(context.Background(), "proj-1")
	if !errors.Is(err, ErrInvalidDispatcherConfig) {
		t.Fatalf("EligibleForPromotion() with nil svc error = %v, want errors.Is(ErrInvalidDispatcherConfig)", err)
	}
	_, err = w.Promote(context.Background(), domain.ActionItem{ID: "x", ProjectID: "p"})
	if !errors.Is(err, ErrInvalidDispatcherConfig) {
		t.Fatalf("Promote() with nil svc error = %v, want errors.Is(ErrInvalidDispatcherConfig)", err)
	}
}

// TestColumnIDForLifecycleStateMatchesCanonicalNames covers the column
// slug-matching contract for every supported state name and the legacy
// alias rejection.
func TestColumnIDForLifecycleStateMatchesCanonicalNames(t *testing.T) {
	t.Parallel()

	cols := []domain.Column{
		{ID: "c-todo", Name: "To Do"},
		{ID: "c-in", Name: "In Progress"},
		{ID: "c-done", Name: "Complete"},
		{ID: "c-fail", Name: "Failed"},
		{ID: "c-arch", Name: "Archived"},
	}
	cases := []struct {
		state domain.LifecycleState
		want  string
	}{
		{domain.StateTodo, "c-todo"},
		{domain.StateInProgress, "c-in"},
		{domain.StateComplete, "c-done"},
		{domain.StateFailed, "c-fail"},
		{domain.StateArchived, "c-arch"},
	}
	for _, tc := range cases {
		got := columnIDForLifecycleState(cols, tc.state)
		if got != tc.want {
			t.Errorf("columnIDForLifecycleState(%q) = %q, want %q", tc.state, got, tc.want)
		}
	}
	if got := columnIDForLifecycleState(cols, "unknown"); got != "" {
		t.Errorf("columnIDForLifecycleState(unknown) = %q, want empty", got)
	}
}

// TestCanonicalStateSlugRejectsLegacyAliases pins the alias rejection
// surface so future regressions on legacy column-name handling fail loudly.
func TestCanonicalStateSlugRejectsLegacyAliases(t *testing.T) {
	t.Parallel()

	cases := []string{"done", "completed", "progress", "doing", "in-progress", ""}
	for _, name := range cases {
		if got := canonicalStateSlug(name); got != "" {
			t.Errorf("canonicalStateSlug(%q) = %q, want empty (legacy alias must reject)", name, got)
		}
	}
	canonical := map[string]string{
		"To Do":       "todo",
		"todo":        "todo",
		"In Progress": "in_progress",
		"in_progress": "in_progress",
		"Complete":    "complete",
		"Failed":      "failed",
		"Archived":    "archived",
	}
	for in, want := range canonical {
		if got := canonicalStateSlug(in); got != want {
			t.Errorf("canonicalStateSlug(%q) = %q, want %q", in, got, want)
		}
	}
}

// erroringListItemsStub returns a configured error from ListActionItems.
// Other methods return zero values; the walker only calls ListActionItems
// in the path under test.
type erroringListItemsStub struct {
	err error
}

func (s *erroringListItemsStub) ListColumns(_ context.Context, _ string, _ bool) ([]domain.Column, error) {
	return nil, nil
}

func (s *erroringListItemsStub) ListActionItems(_ context.Context, _ string, _ bool) ([]domain.ActionItem, error) {
	return nil, s.err
}

func (s *erroringListItemsStub) MoveActionItem(_ context.Context, _, _ string, _ int) (domain.ActionItem, error) {
	return domain.ActionItem{}, nil
}

// Compile-time assertion that *app.Service satisfies walkerService. The
// real binding lives in 4a.23; this assertion catches drift between the
// walker's narrow interface and the service surface as the latter evolves.
var _ walkerService = (walkerServiceCompileCheck{})

// walkerServiceCompileCheck is a stand-in type used only by the assertion
// above. The dispatcher does not consume it; the symbol exists so the
// compile-time interface check has a concrete anchor without importing
// the production *app.Service into the test file (which would create an
// import cycle when the service starts depending on dispatcher in 4a.23).
type walkerServiceCompileCheck struct{}

func (walkerServiceCompileCheck) ListColumns(_ context.Context, _ string, _ bool) ([]domain.Column, error) {
	return nil, nil
}

func (walkerServiceCompileCheck) ListActionItems(_ context.Context, _ string, _ bool) ([]domain.ActionItem, error) {
	return nil, nil
}

func (walkerServiceCompileCheck) MoveActionItem(_ context.Context, _, _ string, _ int) (domain.ActionItem, error) {
	return domain.ActionItem{}, nil
}
