package dispatcher

import "sync"

// fileLockManager is the in-process file-level lock manager that serializes
// Wave-2 dispatcher spawns whose action items declare overlapping `paths`.
// It tracks one holder per path: the action-item ID that currently owns the
// lock. The manager is the runtime backstop for the planner's static
// `blocked_by` wiring — even when two siblings slip past plan-QA without an
// explicit blocker on a shared file, the conflict detector (Wave 2.7) reads
// this manager to decide whether to insert a runtime blocker rather than
// double-spawn.
//
// Wave 2.3 ships only the in-process implementation. Drop 4b decides whether
// to mirror the path → holder map into SQLite for crash recovery; the API on
// this file is intentionally kept small so the persistence decision can land
// without breaking call sites.
//
// Concurrency: the manager is safe for concurrent use by multiple goroutines.
// All reads and writes happen under a single sync.Mutex; the critical section
// is small (map lookup + map write per path) so contention is bounded by the
// number of paths in a single Acquire call.
//
// Path opacity: paths are treated as opaque strings. The manager does NOT
// normalize or canonicalize them — `./a` and `a` are distinct keys, as are
// `a` and `a/`. Callers (the walker in Wave 2.5, the conflict detector in
// Wave 2.7) own normalization. This guarantee is what lets the manager stay
// allocation-free in the hot path and lets Drop 4b's SQLite mirror use a
// straight TEXT primary key without collation surprises.
//
// Deadlock-free contract: the API is flat and synchronous. There are no
// callbacks, no goroutines spawned by the manager, and no nested locking.
// Acquire and Release run to completion before returning; a caller cannot
// re-enter the manager from inside one of its calls.
type fileLockManager struct {
	mu sync.Mutex
	// holders maps path → holding action-item ID. A path is locked iff it
	// has an entry in this map. The map is the single source of truth;
	// itemPaths below is a derived index for O(holder-set) Release.
	holders map[string]string
	// itemPaths maps action-item ID → set of paths that item currently
	// holds. Maintained alongside holders so Release does not have to walk
	// the entire holders map. The inner map's values are always struct{};
	// presence is the signal.
	itemPaths map[string]map[string]struct{}
}

// newFileLockManager returns an empty fileLockManager ready for use. The
// constructor exists for symmetry with packageLockManager (Wave 2.4) and to
// pre-allocate the inner maps so the first Acquire does not pay the
// map-allocation cost. Zero-value fileLockManager{} is also valid; the lazy
// nil-check in Acquire mirrors the broker-subscriber's defensive shape.
func newFileLockManager() *fileLockManager {
	return &fileLockManager{
		holders:   make(map[string]string),
		itemPaths: make(map[string]map[string]struct{}),
	}
}

// Acquire attempts to lock every path in paths for actionItemID. Acquisition
// is partial: paths already held by a different action item are reported in
// the conflicts map (path → holding action-item ID), and every other path is
// added to acquired in input order.
//
// Same-holder semantics: a path already held by actionItemID is treated as a
// successful (idempotent) acquire — it appears in acquired, not in conflicts.
// This makes Acquire safe to retry without the caller having to track which
// paths it already owns.
//
// Empty inputs: an empty paths slice returns an empty acquired slice and an
// empty conflicts map; an empty actionItemID is treated as opaque (no
// validation in this droplet). Both forms are deterministic no-ops or
// near-no-ops; neither mutates state when paths is empty.
//
// The returned conflicts map is freshly allocated per call. Callers may
// mutate it freely; the manager retains no reference. The acquired slice
// preserves input order for caller-side determinism.
//
// Acquire never returns a non-nil error today; the err return exists in the
// signature so Drop 4b can surface SQLite-mirror failures without a breaking
// API change.
func (m *fileLockManager) Acquire(actionItemID string, paths []string) (acquired []string, conflicts map[string]string, err error) {
	conflicts = make(map[string]string)
	if len(paths) == 0 {
		return nil, conflicts, nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if m.holders == nil {
		m.holders = make(map[string]string)
	}
	if m.itemPaths == nil {
		m.itemPaths = make(map[string]map[string]struct{})
	}

	acquired = make([]string, 0, len(paths))
	for _, path := range paths {
		holder, taken := m.holders[path]
		if taken && holder != actionItemID {
			conflicts[path] = holder
			continue
		}
		// Either path is free, or already held by actionItemID
		// (idempotent re-acquire). Record either way.
		m.holders[path] = actionItemID
		owned, ok := m.itemPaths[actionItemID]
		if !ok {
			owned = make(map[string]struct{})
			m.itemPaths[actionItemID] = owned
		}
		owned[path] = struct{}{}
		acquired = append(acquired, path)
	}
	return acquired, conflicts, nil
}

// WouldConflict reports — without acquiring or mutating any state — which
// supplied paths would conflict with the manager's current holders if
// actionItemID called Acquire(actionItemID, paths) right now. The returned
// map mirrors the conflicts shape Acquire produces: path → holding
// action-item ID. Same-holder paths (already held by actionItemID) are NOT
// reported because Acquire treats those as idempotent successes.
//
// WouldConflict is the read-only seam consumed by PreviewSpawn (the
// dispatcher's --dry-run entry point) so a dev can inspect spawn
// reachability without contending with the live lock state. It takes the
// same mutex Acquire takes — concurrent Acquire/Release on the same manager
// will serialize WouldConflict's snapshot against them, but the returned
// map is a snapshot only; the caller MUST NOT treat it as a reservation.
func (m *fileLockManager) WouldConflict(actionItemID string, paths []string) map[string]string {
	conflicts := make(map[string]string)
	if len(paths) == 0 {
		return conflicts
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.holders == nil {
		return conflicts
	}
	for _, path := range paths {
		holder, taken := m.holders[path]
		if taken && holder != actionItemID {
			conflicts[path] = holder
		}
	}
	return conflicts
}

// Release frees every path currently held by actionItemID. Calling Release
// for an action item that holds no paths (including an action item that has
// never called Acquire, or an empty actionItemID) is a no-op.
//
// Release is the only path that removes entries from the holders map. After
// Release returns, the same actionItemID may call Acquire again and reclaim
// those paths (or any subset of them); the manager retains no historical
// state about prior holders.
func (m *fileLockManager) Release(actionItemID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	owned, ok := m.itemPaths[actionItemID]
	if !ok {
		return
	}
	for path := range owned {
		delete(m.holders, path)
	}
	delete(m.itemPaths, actionItemID)
}
