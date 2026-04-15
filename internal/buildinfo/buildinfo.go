// Package buildinfo exposes a slot for future commit-stamp injection at
// build time. Today no build target injects a value, so on every binary
// (dev or installed) Summary() reports "unknown". The package stays wired
// to `till --version` via Summary() so future tooling can populate Commit /
// Dirty via ldflags without changing the call site.
package buildinfo

import "strings"

// unknownCommit is the sentinel returned when no commit SHA has been injected.
const unknownCommit = "unknown"

// Commit is the git SHA of the build. Currently always empty — no build target
// injects a value. Reserved as a stamping slot for future tooling.
var Commit = ""

// Dirty is "true" when build-time tooling flags the binary as built from a
// dirty tree. Currently always empty — no build target sets it.
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
// Today every binary reports "unknown" because no build target injects a
// commit. A stamped binary would render the SHA, with a "-dirty" suffix when
// install-time tooling flagged it.
func Summary() string {
	commit := ResolvedCommit()
	if IsDirty() {
		return commit + "-dirty"
	}
	return commit
}
