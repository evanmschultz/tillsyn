// Package buildinfo exposes compile-time build stamps (commit SHA + dirty flag)
// injected via `-ldflags "-X"` at `mage install` time.
//
// The `mage install` target builds the promoted binary inside a throw-away
// `git worktree add --detach <sha>` checkout of the current HEAD, so the
// stamped commit always identifies a real committed revision regardless of
// whether the caller's working tree is dirty. The HEAD SHA is injected into
// `Commit` via ldflags; `Dirty` exists so future tooling can surface working-
// tree state alongside the SHA without changing the build-stamp schema.
// Surfaced through Summary() and wired to the CLI via `till --version`.
//
// Defaults keep local `mage build` output ergonomic — an unstamped dev binary
// reports a synthetic "unknown" commit instead of an empty string, and the
// dirty flag stays off unless explicitly set.
package buildinfo

import "strings"

// unknownCommit is the sentinel used when no commit SHA is injected at build time.
const unknownCommit = "unknown"

// Commit is the git SHA the installed binary was built from.
// The `mage install` target resolves HEAD inside a detached temp worktree and
// injects the SHA via `-ldflags "-X ...Commit=<sha>"`. Local `mage build`
// (dev) leaves it empty; callers must go through Summary() or ResolvedCommit()
// to get a rendered value.
var Commit = ""

// Dirty is "true" when install-time tooling chose to flag the promoted binary
// as built from a dirty state. `mage install` does not currently set this —
// it always builds from a clean detached worktree at HEAD's SHA — so on
// installed binaries it stays empty. It exists so future tooling can surface
// working-tree state alongside the SHA without changing the build-stamp
// schema.
var Dirty = ""

// ResolvedCommit returns the injected commit SHA or the unknown-commit sentinel
// when the binary was built without an `-ldflags "-X Commit=..."` stamp.
func ResolvedCommit() string {
	trimmed := strings.TrimSpace(Commit)
	if trimmed == "" {
		return unknownCommit
	}
	return trimmed
}

// IsDirty reports whether the install-time working tree was dirty.
// Any non-empty Dirty value other than the literal "false" (case-insensitive)
// is treated as dirty so ldflags injection tolerates common boolean spellings.
func IsDirty() bool {
	trimmed := strings.TrimSpace(Dirty)
	if trimmed == "" {
		return false
	}
	return !strings.EqualFold(trimmed, "false")
}

// Summary returns a single-line commit descriptor suitable for CLI rendering.
// The rendered `commit` is the SHA the installed binary was built from — HEAD
// at `mage install` time, resolved inside the detached temp worktree and
// injected via ldflags. An unstamped binary reports "unknown"; a stamped-and-
// dirty binary gets a "-dirty" suffix so operators can see when install-time
// tooling flagged the promoted binary.
func Summary() string {
	commit := ResolvedCommit()
	if IsDirty() {
		return commit + "-dirty"
	}
	return commit
}
