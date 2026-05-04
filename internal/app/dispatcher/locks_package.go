package dispatcher

import "sync"

// packageLockManager is the in-process package-level lock manager that
// serializes Wave-2 dispatcher spawns whose action items declare overlapping
// `packages`. It tracks one holder per package: the action-item ID that
// currently owns the lock. The manager is the runtime backstop for the
// planner's static `blocked_by` wiring — even when two siblings slip past
// plan-QA without an explicit blocker on a shared Go-package compile unit,
// the conflict detector (Wave 2.7 / droplet 4a.20) reads this manager to
// decide whether to insert a runtime blocker rather than double-spawn.
//
// Independence from fileLockManager: this manager and fileLockManager
// (locks_file.go) are TWO INDEPENDENT MAPS. A path lock on
// `internal/app/foo.go` and a package lock on `internal/app` do NOT collide
// inside either manager. Cross-locking semantics — i.e. "if any file in
// package P is path-locked, treat P as effectively locked too" — live in the
// walker (droplet 4a.18) and the conflict detector (droplet 4a.20), where the
// rule is explicit and reviewable. Keeping the maps independent here means
// each manager stays a simple key→holder map, and the cross-axis policy is
// owned by exactly one component instead of being smeared across both locks.
//
// Wave 2.4 ships only the in-process implementation. Drop 4b decides whether
// to mirror the package → holder map into SQLite for crash recovery; the API
// on this file is intentionally kept small so the persistence decision can
// land without breaking call sites. Package-lock and file-lock are NOT
// collapsed into a single generic `lockManager[K]` because their planned
// Drop 4b evolutions diverge: package-lock is expected to grow per-Go-package
// resolution via `go list -json`, while file-lock will stay opaque-string.
// Premature generic lands as YAGNI.
//
// Concurrency: the manager is safe for concurrent use by multiple goroutines.
// All reads and writes happen under a single sync.Mutex; the critical section
// is small (map lookup + map write per package) so contention is bounded by
// the number of packages in a single Acquire call.
//
// Package opacity: package identifiers are treated as opaque strings. The
// manager does NOT normalize or canonicalize them — `internal/app` and
// `internal/app/` are distinct keys, as are `./internal/app` and
// `internal/app`. Callers (the walker in droplet 4a.18, the conflict detector
// in droplet 4a.20) own normalization. This guarantee is what lets the
// manager stay allocation-free in the hot path and lets Drop 4b's SQLite
// mirror use a straight TEXT primary key without collation surprises.
//
// Deadlock-free contract: the API is flat and synchronous. There are no
// callbacks, no goroutines spawned by the manager, and no nested locking.
// Acquire and Release run to completion before returning; a caller cannot
// re-enter the manager from inside one of its calls.
type packageLockManager struct {
	mu sync.Mutex
	// holders maps package → holding action-item ID. A package is locked iff
	// it has an entry in this map. The map is the single source of truth;
	// itemPackages below is a derived index for O(holder-set) Release.
	holders map[string]string
	// itemPackages maps action-item ID → set of packages that item currently
	// holds. Maintained alongside holders so Release does not have to walk
	// the entire holders map. The inner map's values are always struct{};
	// presence is the signal.
	itemPackages map[string]map[string]struct{}
}

// newPackageLockManager returns an empty packageLockManager ready for use.
// The constructor mirrors newFileLockManager (locks_file.go) and pre-allocates
// the inner maps so the first Acquire does not pay the map-allocation cost.
// Zero-value packageLockManager{} is also valid; the lazy nil-check in
// Acquire mirrors the broker-subscriber's defensive shape.
func newPackageLockManager() *packageLockManager {
	return &packageLockManager{
		holders:      make(map[string]string),
		itemPackages: make(map[string]map[string]struct{}),
	}
}

// Acquire attempts to lock every package in packages for actionItemID.
// Acquisition is partial: packages already held by a different action item
// are reported in the conflicts map (package → holding action-item ID), and
// every other package is added to acquired in input order.
//
// Same-holder semantics: a package already held by actionItemID is treated
// as a successful (idempotent) acquire — it appears in acquired, not in
// conflicts. This makes Acquire safe to retry without the caller having to
// track which packages it already owns.
//
// Empty inputs: an empty packages slice returns an empty acquired slice and
// an empty conflicts map; an empty actionItemID is treated as opaque (no
// validation in this droplet). Both forms are deterministic no-ops or
// near-no-ops; neither mutates state when packages is empty.
//
// The returned conflicts map is freshly allocated per call. Callers may
// mutate it freely; the manager retains no reference. The acquired slice
// preserves input order for caller-side determinism.
//
// Acquire never returns a non-nil error today; the err return exists in the
// signature so Drop 4b can surface SQLite-mirror failures without a breaking
// API change.
func (m *packageLockManager) Acquire(actionItemID string, packages []string) (acquired []string, conflicts map[string]string, err error) {
	conflicts = make(map[string]string)
	if len(packages) == 0 {
		return nil, conflicts, nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if m.holders == nil {
		m.holders = make(map[string]string)
	}
	if m.itemPackages == nil {
		m.itemPackages = make(map[string]map[string]struct{})
	}

	acquired = make([]string, 0, len(packages))
	for _, pkg := range packages {
		holder, taken := m.holders[pkg]
		if taken && holder != actionItemID {
			conflicts[pkg] = holder
			continue
		}
		// Either pkg is free, or already held by actionItemID
		// (idempotent re-acquire). Record either way.
		m.holders[pkg] = actionItemID
		owned, ok := m.itemPackages[actionItemID]
		if !ok {
			owned = make(map[string]struct{})
			m.itemPackages[actionItemID] = owned
		}
		owned[pkg] = struct{}{}
		acquired = append(acquired, pkg)
	}
	return acquired, conflicts, nil
}

// Release frees every package currently held by actionItemID. Calling Release
// for an action item that holds no packages (including an action item that
// has never called Acquire, or an empty actionItemID) is a no-op.
//
// Release is the only path that removes entries from the holders map. After
// Release returns, the same actionItemID may call Acquire again and reclaim
// those packages (or any subset of them); the manager retains no historical
// state about prior holders.
func (m *packageLockManager) Release(actionItemID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	owned, ok := m.itemPackages[actionItemID]
	if !ok {
		return
	}
	for pkg := range owned {
		delete(m.holders, pkg)
	}
	delete(m.itemPackages, actionItemID)
}
