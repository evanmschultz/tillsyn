package dispatcher

import (
	"slices"
	"strconv"
	"sync"
	"testing"
)

// TestFileLockAcquireSinglePathSucceeds asserts the happy-path baseline: a
// fresh manager grants a single-path Acquire to its first caller with no
// conflicts and the returned slice/map shapes match the spec.
func TestFileLockAcquireSinglePathSucceeds(t *testing.T) {
	t.Parallel()

	mgr := newFileLockManager()

	acquired, conflicts, err := mgr.Acquire("item-1", []string{"a"})
	if err != nil {
		t.Fatalf("Acquire returned error: %v", err)
	}
	if len(conflicts) != 0 {
		t.Fatalf("expected zero conflicts, got %d: %v", len(conflicts), conflicts)
	}
	if len(acquired) != 1 || acquired[0] != "a" {
		t.Fatalf("expected acquired=[a], got %v", acquired)
	}
}

// TestFileLockAcquireSamePathTwiceByDifferentItemsConflicts asserts that a
// second action item asking for an already-held path is rejected via the
// conflicts map and receives an empty acquired slice.
func TestFileLockAcquireSamePathTwiceByDifferentItemsConflicts(t *testing.T) {
	t.Parallel()

	mgr := newFileLockManager()

	if _, _, err := mgr.Acquire("item-1", []string{"a"}); err != nil {
		t.Fatalf("Acquire item-1: %v", err)
	}

	acquired, conflicts, err := mgr.Acquire("item-2", []string{"a"})
	if err != nil {
		t.Fatalf("Acquire item-2: %v", err)
	}
	if len(acquired) != 0 {
		t.Fatalf("expected acquired=[], got %v", acquired)
	}
	if got, want := conflicts["a"], "item-1"; got != want {
		t.Fatalf("expected conflicts[a]=%q, got %q", want, got)
	}
	if len(conflicts) != 1 {
		t.Fatalf("expected exactly one conflict, got %d: %v", len(conflicts), conflicts)
	}
}

// TestFileLockReleaseFreesAllPathsHeldByItem asserts that Release frees every
// path the action item holds (not just one) and that a subsequent Acquire by
// a different item succeeds for the full set.
func TestFileLockReleaseFreesAllPathsHeldByItem(t *testing.T) {
	t.Parallel()

	mgr := newFileLockManager()

	if _, _, err := mgr.Acquire("item-1", []string{"a", "b", "c"}); err != nil {
		t.Fatalf("Acquire item-1: %v", err)
	}

	mgr.Release("item-1")

	acquired, conflicts, err := mgr.Acquire("item-2", []string{"a", "b", "c"})
	if err != nil {
		t.Fatalf("Acquire item-2: %v", err)
	}
	if len(conflicts) != 0 {
		t.Fatalf("expected zero conflicts after Release, got %v", conflicts)
	}
	if !slices.Equal(acquired, []string{"a", "b", "c"}) {
		t.Fatalf("expected acquired=[a b c], got %v", acquired)
	}
}

// TestFileLockAcquirePartialConflictReturnsConflicts asserts the partial-acquire
// contract: when one of N requested paths is already held, the free paths are
// returned in acquired and the held one(s) are returned in conflicts.
func TestFileLockAcquirePartialConflictReturnsConflicts(t *testing.T) {
	t.Parallel()

	mgr := newFileLockManager()

	if _, _, err := mgr.Acquire("item-1", []string{"b"}); err != nil {
		t.Fatalf("Acquire item-1: %v", err)
	}

	acquired, conflicts, err := mgr.Acquire("item-2", []string{"a", "b"})
	if err != nil {
		t.Fatalf("Acquire item-2: %v", err)
	}
	if !slices.Equal(acquired, []string{"a"}) {
		t.Fatalf("expected acquired=[a], got %v", acquired)
	}
	if len(conflicts) != 1 || conflicts["b"] != "item-1" {
		t.Fatalf("expected conflicts={b:item-1}, got %v", conflicts)
	}

	// Sanity: the partially-acquired path is now held by item-2 even though
	// the call also returned a conflict. A retry by item-2 against [a, c]
	// should treat [a] as same-holder idempotent and [c] as fresh.
	acquired2, conflicts2, err := mgr.Acquire("item-2", []string{"a", "c"})
	if err != nil {
		t.Fatalf("Acquire item-2 retry: %v", err)
	}
	if !slices.Equal(acquired2, []string{"a", "c"}) {
		t.Fatalf("expected acquired=[a c] on idempotent retry, got %v", acquired2)
	}
	if len(conflicts2) != 0 {
		t.Fatalf("expected zero conflicts on idempotent retry, got %v", conflicts2)
	}
}

// TestFileLockConcurrentAcquireRaceFree asserts that N goroutines racing
// Acquire on the same path produce exactly one winner, and that all losers
// see the winner's ID in their conflicts map. -race in mage testPkg is the
// teeth that catches a missing mutex; this assertion is the spec teeth that
// catches a serialized-but-non-deterministic lock acquisition.
func TestFileLockConcurrentAcquireRaceFree(t *testing.T) {
	t.Parallel()

	mgr := newFileLockManager()

	const goroutines = 32
	const path = "shared"

	var (
		wg          sync.WaitGroup
		startGate   = make(chan struct{})
		mu          sync.Mutex
		winners     []string
		conflictMap = make(map[string]string) // loser ID → winner ID
	)

	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		id := "item-" + strconv.Itoa(i)
		go func(id string) {
			defer wg.Done()
			<-startGate
			acquired, conflicts, err := mgr.Acquire(id, []string{path})
			if err != nil {
				t.Errorf("goroutine %s: Acquire returned error: %v", id, err)
				return
			}
			mu.Lock()
			defer mu.Unlock()
			if len(acquired) == 1 {
				winners = append(winners, id)
				return
			}
			if holder, ok := conflicts[path]; ok {
				conflictMap[id] = holder
				return
			}
			t.Errorf("goroutine %s: neither acquired nor conflicted: acquired=%v conflicts=%v",
				id, acquired, conflicts)
		}(id)
	}

	close(startGate)
	wg.Wait()

	if len(winners) != 1 {
		t.Fatalf("expected exactly one winner, got %d: %v", len(winners), winners)
	}
	winner := winners[0]
	if got, want := len(conflictMap), goroutines-1; got != want {
		t.Fatalf("expected %d losers, got %d", want, got)
	}
	for loser, holder := range conflictMap {
		if holder != winner {
			t.Errorf("loser %s saw holder=%s, want %s", loser, holder, winner)
		}
	}

	// Release the winner and re-race; another goroutine should win the
	// freed path. This pins the post-Release acquire path under contention.
	mgr.Release(winner)

	acquired, conflicts, err := mgr.Acquire("recovery", []string{path})
	if err != nil {
		t.Fatalf("recovery Acquire: %v", err)
	}
	if !slices.Equal(acquired, []string{path}) || len(conflicts) != 0 {
		t.Fatalf("expected recovery acquire to succeed cleanly, got acquired=%v conflicts=%v",
			acquired, conflicts)
	}
}

// TestFileLockPathsAreOpaque asserts the documented opacity contract: the
// manager does NOT normalize paths. `./a` and `a` are distinct keys. This is
// the runtime guarantee the conflict detector (Wave 2.7) relies on so it can
// own normalization itself without the lock manager second-guessing it.
func TestFileLockPathsAreOpaque(t *testing.T) {
	t.Parallel()

	mgr := newFileLockManager()

	if _, _, err := mgr.Acquire("item-1", []string{"./a"}); err != nil {
		t.Fatalf("Acquire item-1: %v", err)
	}

	acquired, conflicts, err := mgr.Acquire("item-2", []string{"a"})
	if err != nil {
		t.Fatalf("Acquire item-2: %v", err)
	}
	if !slices.Equal(acquired, []string{"a"}) {
		t.Fatalf("expected acquired=[a] (distinct key from ./a), got %v", acquired)
	}
	if len(conflicts) != 0 {
		t.Fatalf("expected zero conflicts, got %v", conflicts)
	}
}

// TestFileLockEmptyInputsAreNoOps pins the documented edge cases: empty paths
// is a no-op, Release on an unknown ID is a no-op, and same-holder reacquire
// is idempotent. These are the four corners called out in locks_file.go's
// doc comment; the test exists to prevent silent regression.
func TestFileLockEmptyInputsAreNoOps(t *testing.T) {
	t.Parallel()

	mgr := newFileLockManager()

	acquired, conflicts, err := mgr.Acquire("item-1", nil)
	if err != nil {
		t.Fatalf("Acquire nil: %v", err)
	}
	if len(acquired) != 0 {
		t.Fatalf("expected empty acquired for nil paths, got %v", acquired)
	}
	if len(conflicts) != 0 {
		t.Fatalf("expected empty conflicts for nil paths, got %v", conflicts)
	}

	acquired, conflicts, err = mgr.Acquire("item-1", []string{})
	if err != nil {
		t.Fatalf("Acquire empty: %v", err)
	}
	if len(acquired) != 0 || len(conflicts) != 0 {
		t.Fatalf("expected empty results for empty paths, got acquired=%v conflicts=%v",
			acquired, conflicts)
	}

	// Release on an unknown ID is a no-op.
	mgr.Release("never-acquired")

	// Same-holder idempotent reacquire.
	if _, _, err := mgr.Acquire("item-1", []string{"a"}); err != nil {
		t.Fatalf("Acquire item-1 [a]: %v", err)
	}
	acquired, conflicts, err = mgr.Acquire("item-1", []string{"a"})
	if err != nil {
		t.Fatalf("Acquire item-1 [a] (re): %v", err)
	}
	if !slices.Equal(acquired, []string{"a"}) {
		t.Fatalf("expected idempotent reacquire to return acquired=[a], got %v", acquired)
	}
	if len(conflicts) != 0 {
		t.Fatalf("expected zero conflicts on idempotent reacquire, got %v", conflicts)
	}
}

// TestFileLockZeroValueIsUsable pins the doc-comment claim that
// fileLockManager{} (zero value) is valid alongside newFileLockManager().
// Wave 2 callers go through the constructor, but the lazy-init paths in
// Acquire mirror broker_sub.go's defensive style and need explicit coverage.
func TestFileLockZeroValueIsUsable(t *testing.T) {
	t.Parallel()

	var mgr fileLockManager

	acquired, conflicts, err := mgr.Acquire("item-1", []string{"a"})
	if err != nil {
		t.Fatalf("zero-value Acquire: %v", err)
	}
	if !slices.Equal(acquired, []string{"a"}) {
		t.Fatalf("expected acquired=[a], got %v", acquired)
	}
	if len(conflicts) != 0 {
		t.Fatalf("expected zero conflicts, got %v", conflicts)
	}

	mgr.Release("item-1")

	// After release the path is free for another holder.
	acquired2, _, err := mgr.Acquire("item-2", []string{"a"})
	if err != nil {
		t.Fatalf("zero-value Acquire item-2: %v", err)
	}
	if !slices.Equal(acquired2, []string{"a"}) {
		t.Fatalf("expected acquired=[a] for item-2, got %v", acquired2)
	}
}

// TestFileLockManagerAcquirePreservesInputOrder pins the input-order semantics
// documented on Acquire: given paths in arbitrary input order against an
// empty manager, acquired mirrors the caller's argument exactly,
// element-by-element. The assertion uses slices.Equal (not sort-then-compare)
// so a future implementation that sorts internally for deadlock-avoidance
// would surface here as a behavior change requiring its own droplet.
func TestFileLockManagerAcquirePreservesInputOrder(t *testing.T) {
	t.Parallel()

	mgr := newFileLockManager()

	input := []string{"c", "a", "b"}
	acquired, conflicts, err := mgr.Acquire("item-1", input)
	if err != nil {
		t.Fatalf("Acquire item-1: %v", err)
	}
	if len(conflicts) != 0 {
		t.Fatalf("expected zero conflicts on empty manager, got %v", conflicts)
	}
	if !slices.Equal(acquired, []string{"c", "a", "b"}) {
		t.Fatalf("expected acquired=[c a b] preserving input order, got %v", acquired)
	}

	// Mixed conflict + free input: item-2 asks for [b, x, a, y] where item-1
	// already holds a + b. Free entries (x, y) must appear in acquired in
	// their input positions; held entries (a, b) must appear in conflicts;
	// neither slice is sorted by the manager.
	acquired2, conflicts2, err := mgr.Acquire("item-2", []string{"b", "x", "a", "y"})
	if err != nil {
		t.Fatalf("Acquire item-2: %v", err)
	}
	if !slices.Equal(acquired2, []string{"x", "y"}) {
		t.Fatalf("expected acquired=[x y] preserving input order, got %v", acquired2)
	}
	if got, want := conflicts2["a"], "item-1"; got != want {
		t.Fatalf("expected conflicts[a]=%q, got %q", want, got)
	}
	if got, want := conflicts2["b"], "item-1"; got != want {
		t.Fatalf("expected conflicts[b]=%q, got %q", want, got)
	}
	if len(conflicts2) != 2 {
		t.Fatalf("expected exactly two conflicts, got %d: %v", len(conflicts2), conflicts2)
	}
}

// TestFileLockManagerAcquireDuplicateInputIdempotent pins the duplicate-input
// semantics documented on Acquire: a duplicate within a single call is a
// same-holder idempotent success per occurrence. Each duplicate appears in
// acquired in its original input position, while the manager's internal
// holders / itemPaths maps end identical to the de-duplicated case (one
// entry each). This pins the chosen behavior; a future change to dedupe-on-
// input would be a behavior shift requiring its own droplet.
func TestFileLockManagerAcquireDuplicateInputIdempotent(t *testing.T) {
	t.Parallel()

	mgr := newFileLockManager()

	input := []string{"a", "a", "b"}
	acquired, conflicts, err := mgr.Acquire("item-1", input)
	if err != nil {
		t.Fatalf("Acquire item-1: %v", err)
	}
	if len(conflicts) != 0 {
		t.Fatalf("expected zero conflicts on empty manager, got %v", conflicts)
	}
	// Per the documented semantics: each occurrence is recorded
	// independently, so acquired carries the duplicate.
	if !slices.Equal(acquired, []string{"a", "a", "b"}) {
		t.Fatalf("expected acquired=[a a b] (each occurrence preserved), got %v", acquired)
	}

	// Internal state is collapsed: one holder per distinct path. We probe
	// this externally by asking item-2 to acquire [a, b]; both must
	// register as conflicts held by item-1. If duplicates had created two
	// "holders" of "a" inside the manager, the second would overwrite the
	// first — but holders[path] is a single string, so the invariant is
	// observable as: item-2 sees one conflict per distinct path, not two.
	_, conflicts2, err := mgr.Acquire("item-2", []string{"a", "b"})
	if err != nil {
		t.Fatalf("Acquire item-2: %v", err)
	}
	if len(conflicts2) != 2 {
		t.Fatalf("expected two conflicts (one per distinct path), got %d: %v",
			len(conflicts2), conflicts2)
	}
	if got, want := conflicts2["a"], "item-1"; got != want {
		t.Fatalf("expected conflicts[a]=%q, got %q", want, got)
	}
	if got, want := conflicts2["b"], "item-1"; got != want {
		t.Fatalf("expected conflicts[b]=%q, got %q", want, got)
	}

	// Release item-1 and verify both distinct paths free up — confirming the
	// duplicate input did not leave a stray holder entry behind.
	mgr.Release("item-1")
	acquired3, conflicts3, err := mgr.Acquire("item-3", []string{"a", "b"})
	if err != nil {
		t.Fatalf("Acquire item-3 after Release: %v", err)
	}
	if !slices.Equal(acquired3, []string{"a", "b"}) {
		t.Fatalf("expected acquired=[a b] after item-1 Release, got %v", acquired3)
	}
	if len(conflicts3) != 0 {
		t.Fatalf("expected zero conflicts after item-1 Release, got %v", conflicts3)
	}
}
