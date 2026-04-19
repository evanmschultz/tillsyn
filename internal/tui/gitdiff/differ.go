// Package gitdiff provides a Differ interface and exec-backed implementation
// for computing git diffs in the TUI diff pane.
//
// The package owns `git diff` execution so the diff pane surface can depend on
// a narrow, stable abstraction (Differ) rather than on a concrete git client.
// The default implementation shells out to the `git` binary via os/exec — it
// is intentionally consumer-driven, returns raw unified patch text, and emits
// a branch-divergence status computed from `git merge-base --is-ancestor`.
package gitdiff

import (
	"context"
	"errors"
)

// DivergenceStatus describes the relationship between the diff's start commit
// and the repository HEAD at the moment Diff was invoked.
//
// The status is informational: the patch is still computed when the start
// commit is not an ancestor of HEAD. Callers typically render a banner when
// Divergence is Diverged so the viewer is not misled about scope.
type DivergenceStatus int

// DivergenceStatus values.
//
// Ancestor means `git rev-parse --is-ancestor <start> HEAD` exited 0. Diverged
// means it exited 1 — the history forked. Unknown means the check could not
// reach a definitive verdict, typically because one of the refs failed to
// resolve; callers should treat Unknown as "do not claim ancestry either way"
// rather than as a hard error.
const (
	DivergenceUnknown DivergenceStatus = iota
	DivergenceAncestor
	DivergenceDiverged
)

// String renders the DivergenceStatus as a short, stable label suitable for
// log messages and banner text.
func (d DivergenceStatus) String() string {
	switch d {
	case DivergenceAncestor:
		return "ancestor"
	case DivergenceDiverged:
		return "diverged"
	default:
		return "unknown"
	}
}

// DiffResult carries the output of one Differ.Diff invocation.
//
// Patch holds the raw unified-diff text produced by `git diff`; it may be the
// empty string when start and end resolve to the same tree. Divergence records
// the ancestry verdict between start and HEAD. StartSHA and EndSHA echo the
// resolved commit identifiers so downstream consumers (chroma highlighter,
// banner renderer) can label the patch without re-resolving refs.
type DiffResult struct {
	Patch      string
	Divergence DivergenceStatus
	StartSHA   string
	EndSHA     string
}

// Differ computes a unified diff between two git revisions, optionally
// restricted to a subset of paths, and reports ancestry between the start
// commit and HEAD.
//
// Implementations must honor ctx cancellation and return wrapped errors that
// preserve the underlying cause (use fmt.Errorf with %w). The interface is
// consumer-side — the diff pane in internal/tui binds to Differ, not to any
// concrete type.
type Differ interface {
	Diff(ctx context.Context, start, end string, paths []string) (DiffResult, error)
}

// ErrEmptyRevision is returned when Diff is called with an empty start or end
// revision string. Callers that build revisions from user input can use
// errors.Is to detect this and present a targeted message.
var ErrEmptyRevision = errors.New("gitdiff: empty revision")

// ErrUnknownCommit is returned when `git diff` or `git rev-parse` reports that
// the supplied revision cannot be resolved in the current repository. The
// underlying exec error is wrapped into the returned value, so callers can
// errors.Is(err, ErrUnknownCommit) to trigger a "bad ref" branch while still
// surfacing the raw git stderr via errors.Unwrap.
var ErrUnknownCommit = errors.New("gitdiff: unknown commit")
