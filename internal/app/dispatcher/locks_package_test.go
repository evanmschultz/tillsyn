package dispatcher

import (
	"strconv"
	"sync"
	"testing"
)

// TestPackageLockAcquireSinglePackageSucceeds asserts the happy-path baseline:
// a fresh manager grants a single-package Acquire to its first caller with no
// conflicts and the returned slice/map shapes match the spec.
func TestPackageLockAcquireSinglePackageSucceeds(t *testing.T) {
	t.Parallel()

	mgr := newPackageLockManager()

	acquired, conflicts, err := mgr.Acquire("item-1", []string{"internal/app"})
	if err != nil {
		t.Fatalf("Acquire returned error: %v", err)
	}
	if len(conflicts) != 0 {
		t.Fatalf("expected zero conflicts, got %d: %v", len(conflicts), conflicts)
	}
	if len(acquired) != 1 || acquired[0] != "internal/app" {
		t.Fatalf("expected acquired=[internal/app], got %v", acquired)
	}
}

// TestPackageLockAcquireSamePackageTwiceConflicts asserts that a second action
// item asking for an already-held package is rejected via the conflicts map
// and receives an empty acquired slice.
func TestPackageLockAcquireSamePackageTwiceConflicts(t *testing.T) {
	t.Parallel()

	mgr := newPackageLockManager()

	if _, _, err := mgr.Acquire("item-1", []string{"internal/app"}); err != nil {
		t.Fatalf("Acquire item-1: %v", err)
	}

	acquired, conflicts, err := mgr.Acquire("item-2", []string{"internal/app"})
	if err != nil {
		t.Fatalf("Acquire item-2: %v", err)
	}
	if len(acquired) != 0 {
		t.Fatalf("expected acquired=[], got %v", acquired)
	}
	if got, want := conflicts["internal/app"], "item-1"; got != want {
		t.Fatalf("expected conflicts[internal/app]=%q, got %q", want, got)
	}
	if len(conflicts) != 1 {
		t.Fatalf("expected exactly one conflict, got %d: %v", len(conflicts), conflicts)
	}
}

// TestPackageLockReleaseFreesAllPackagesHeldByItem asserts that Release frees
// every package the action item holds (not just one) and that a subsequent
// Acquire by a different item succeeds for the full set.
func TestPackageLockReleaseFreesAllPackagesHeldByItem(t *testing.T) {
	t.Parallel()

	mgr := newPackageLockManager()

	if _, _, err := mgr.Acquire("item-1", []string{"internal/app", "internal/domain", "internal/tui"}); err != nil {
		t.Fatalf("Acquire item-1: %v", err)
	}

	mgr.Release("item-1")

	acquired, conflicts, err := mgr.Acquire("item-2", []string{"internal/app", "internal/domain", "internal/tui"})
	if err != nil {
		t.Fatalf("Acquire item-2: %v", err)
	}
	if len(conflicts) != 0 {
		t.Fatalf("expected zero conflicts after Release, got %v", conflicts)
	}
	if !equalStringSlices(acquired, []string{"internal/app", "internal/domain", "internal/tui"}) {
		t.Fatalf("expected acquired=[internal/app internal/domain internal/tui], got %v", acquired)
	}
}

// TestPackageLockAcquirePartialConflictReturnsConflicts asserts the
// partial-acquire contract: when one of N requested packages is already held,
// the free packages are returned in acquired and the held one(s) are returned
// in conflicts.
func TestPackageLockAcquirePartialConflictReturnsConflicts(t *testing.T) {
	t.Parallel()

	mgr := newPackageLockManager()

	if _, _, err := mgr.Acquire("item-1", []string{"internal/domain"}); err != nil {
		t.Fatalf("Acquire item-1: %v", err)
	}

	acquired, conflicts, err := mgr.Acquire("item-2", []string{"internal/app", "internal/domain"})
	if err != nil {
		t.Fatalf("Acquire item-2: %v", err)
	}
	if !equalStringSlices(acquired, []string{"internal/app"}) {
		t.Fatalf("expected acquired=[internal/app], got %v", acquired)
	}
	if len(conflicts) != 1 || conflicts["internal/domain"] != "item-1" {
		t.Fatalf("expected conflicts={internal/domain:item-1}, got %v", conflicts)
	}

	// Sanity: the partially-acquired package is now held by item-2 even
	// though the call also returned a conflict. A retry by item-2 against
	// [internal/app, internal/tui] should treat [internal/app] as
	// same-holder idempotent and [internal/tui] as fresh.
	acquired2, conflicts2, err := mgr.Acquire("item-2", []string{"internal/app", "internal/tui"})
	if err != nil {
		t.Fatalf("Acquire item-2 retry: %v", err)
	}
	if !equalStringSlices(acquired2, []string{"internal/app", "internal/tui"}) {
		t.Fatalf("expected acquired=[internal/app internal/tui] on idempotent retry, got %v", acquired2)
	}
	if len(conflicts2) != 0 {
		t.Fatalf("expected zero conflicts on idempotent retry, got %v", conflicts2)
	}
}

// TestPackageLockConcurrentAcquireRaceFree asserts that N goroutines racing
// Acquire on the same package produce exactly one winner, and that all losers
// see the winner's ID in their conflicts map. -race in mage testPkg is the
// teeth that catches a missing mutex; this assertion is the spec teeth that
// catches a serialized-but-non-deterministic lock acquisition.
func TestPackageLockConcurrentAcquireRaceFree(t *testing.T) {
	t.Parallel()

	mgr := newPackageLockManager()

	const goroutines = 32
	const pkg = "internal/app/shared"

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
			acquired, conflicts, err := mgr.Acquire(id, []string{pkg})
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
			if holder, ok := conflicts[pkg]; ok {
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
	// freed package. This pins the post-Release acquire path under
	// contention.
	mgr.Release(winner)

	acquired, conflicts, err := mgr.Acquire("recovery", []string{pkg})
	if err != nil {
		t.Fatalf("recovery Acquire: %v", err)
	}
	if !equalStringSlices(acquired, []string{pkg}) || len(conflicts) != 0 {
		t.Fatalf("expected recovery acquire to succeed cleanly, got acquired=%v conflicts=%v",
			acquired, conflicts)
	}
}

// TestPackageLockPackagesAreOpaque asserts the documented opacity contract:
// the manager does NOT normalize package identifiers. `./internal/app` and
// `internal/app` are distinct keys, as are `internal/app` and `internal/app/`.
// This is the runtime guarantee the conflict detector (droplet 4a.20) relies
// on so it can own normalization itself without the lock manager
// second-guessing it.
func TestPackageLockPackagesAreOpaque(t *testing.T) {
	t.Parallel()

	mgr := newPackageLockManager()

	if _, _, err := mgr.Acquire("item-1", []string{"./internal/app"}); err != nil {
		t.Fatalf("Acquire item-1: %v", err)
	}

	acquired, conflicts, err := mgr.Acquire("item-2", []string{"internal/app"})
	if err != nil {
		t.Fatalf("Acquire item-2: %v", err)
	}
	if !equalStringSlices(acquired, []string{"internal/app"}) {
		t.Fatalf("expected acquired=[internal/app] (distinct key from ./internal/app), got %v", acquired)
	}
	if len(conflicts) != 0 {
		t.Fatalf("expected zero conflicts, got %v", conflicts)
	}
}

// TestPackageLockEmptyInputsAreNoOps pins the documented edge cases: empty
// packages is a no-op, Release on an unknown ID is a no-op, and same-holder
// reacquire is idempotent. These are the four corners called out in
// locks_package.go's doc comment; the test exists to prevent silent
// regression and to cover the early-return branch in Acquire.
func TestPackageLockEmptyInputsAreNoOps(t *testing.T) {
	t.Parallel()

	mgr := newPackageLockManager()

	acquired, conflicts, err := mgr.Acquire("item-1", nil)
	if err != nil {
		t.Fatalf("Acquire nil: %v", err)
	}
	if len(acquired) != 0 {
		t.Fatalf("expected empty acquired for nil packages, got %v", acquired)
	}
	if len(conflicts) != 0 {
		t.Fatalf("expected empty conflicts for nil packages, got %v", conflicts)
	}

	acquired, conflicts, err = mgr.Acquire("item-1", []string{})
	if err != nil {
		t.Fatalf("Acquire empty: %v", err)
	}
	if len(acquired) != 0 || len(conflicts) != 0 {
		t.Fatalf("expected empty results for empty packages, got acquired=%v conflicts=%v",
			acquired, conflicts)
	}

	// Release on an unknown ID is a no-op.
	mgr.Release("never-acquired")

	// Same-holder idempotent reacquire.
	if _, _, err := mgr.Acquire("item-1", []string{"internal/app"}); err != nil {
		t.Fatalf("Acquire item-1 [internal/app]: %v", err)
	}
	acquired, conflicts, err = mgr.Acquire("item-1", []string{"internal/app"})
	if err != nil {
		t.Fatalf("Acquire item-1 [internal/app] (re): %v", err)
	}
	if !equalStringSlices(acquired, []string{"internal/app"}) {
		t.Fatalf("expected idempotent reacquire to return acquired=[internal/app], got %v", acquired)
	}
	if len(conflicts) != 0 {
		t.Fatalf("expected zero conflicts on idempotent reacquire, got %v", conflicts)
	}
}

// TestPackageLockZeroValueIsUsable pins the doc-comment claim that
// packageLockManager{} (zero value) is valid alongside newPackageLockManager().
// Wave 2 callers go through the constructor, but the lazy-init paths in
// Acquire mirror broker_sub.go's defensive style and need explicit coverage
// to hit the nil-map branches.
func TestPackageLockZeroValueIsUsable(t *testing.T) {
	t.Parallel()

	var mgr packageLockManager

	acquired, conflicts, err := mgr.Acquire("item-1", []string{"internal/app"})
	if err != nil {
		t.Fatalf("zero-value Acquire: %v", err)
	}
	if !equalStringSlices(acquired, []string{"internal/app"}) {
		t.Fatalf("expected acquired=[internal/app], got %v", acquired)
	}
	if len(conflicts) != 0 {
		t.Fatalf("expected zero conflicts, got %v", conflicts)
	}

	mgr.Release("item-1")

	// After release the package is free for another holder.
	acquired2, _, err := mgr.Acquire("item-2", []string{"internal/app"})
	if err != nil {
		t.Fatalf("zero-value Acquire item-2: %v", err)
	}
	if !equalStringSlices(acquired2, []string{"internal/app"}) {
		t.Fatalf("expected acquired=[internal/app] for item-2, got %v", acquired2)
	}
}

// TestPackageLockIndependentFromFileLock pins the doc-comment guarantee that
// packageLockManager and fileLockManager are TWO INDEPENDENT MAPS. Locking
// a path string in fileLockManager does NOT pre-lock a same-string package
// in packageLockManager, and vice versa. Cross-locking semantics belong to
// the walker (droplet 4a.18) and the conflict detector (droplet 4a.20), not
// to either lock manager.
func TestPackageLockIndependentFromFileLock(t *testing.T) {
	t.Parallel()

	pkgMgr := newPackageLockManager()
	fileMgr := newFileLockManager()

	// item-1 takes the file lock on the literal string "internal/app".
	if _, _, err := fileMgr.Acquire("item-1", []string{"internal/app"}); err != nil {
		t.Fatalf("file Acquire item-1: %v", err)
	}

	// item-2 should still be able to take the package lock on the same
	// literal string — the maps are independent.
	acquired, conflicts, err := pkgMgr.Acquire("item-2", []string{"internal/app"})
	if err != nil {
		t.Fatalf("package Acquire item-2: %v", err)
	}
	if !equalStringSlices(acquired, []string{"internal/app"}) {
		t.Fatalf("expected acquired=[internal/app] (independent of file lock), got %v", acquired)
	}
	if len(conflicts) != 0 {
		t.Fatalf("expected zero conflicts (independent of file lock), got %v", conflicts)
	}
}
