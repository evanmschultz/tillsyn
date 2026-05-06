// Package gitenv provides a shared helper for building isolation-safe
// environments for child `git` invocations.
//
// The single export Filtered() returns os.Environ() with every "GIT_*=..."
// entry removed, so callers can append their own explicit GIT_* values (e.g.
// GIT_DIR / GIT_CEILING_DIRECTORIES / GIT_CONFIG_GLOBAL) without inheriting
// a parent process's GIT_* state. This matters most when the caller runs
// inside a `git push` pre-push hook: git itself sets GIT_DIR, GIT_INDEX_FILE,
// GIT_WORK_TREE, GIT_PREFIX, etc. on the hook's environment, all pointing at
// the bare repo running the push. GIT_DIR in particular overrides repository
// discovery entirely — GIT_CEILING_DIRECTORIES does NOT undo it. Stripping
// the inherited GIT_* keys before appending isolation overrides is the only
// way to guarantee the child invocation operates inside the caller's
// declared repo root.
//
// Callers today:
//   - internal/app/git_status.go (production: action-item Paths pre-check
//     during Service.CreateActionItem).
//   - internal/tui/gitdiff/exec_differ_test.go (test fixture: gitFixture's
//     per-call git env composition).
//
// Both callers append their own explicit GIT_* keys (GIT_AUTHOR_DATE,
// GIT_COMMITTER_DATE, GIT_CEILING_DIRECTORIES, etc.) after Filtered(); the
// re-added keys take effect because exec.Cmd resolves duplicate env entries
// as last-wins.
package gitenv

import (
	"os"
	"strings"
)

// Filtered returns os.Environ() minus every entry whose key has the "GIT_"
// prefix. The returned slice is a fresh allocation, safe for callers to
// append additional env entries to without aliasing os.Environ()'s backing
// store.
//
// Callers that need to set specific GIT_* keys (e.g. GIT_CEILING_DIRECTORIES,
// GIT_CONFIG_NOSYSTEM) must append them to the returned slice — Filtered
// itself does NOT inject any GIT_* values.
func Filtered() []string {
	src := os.Environ()
	out := make([]string, 0, len(src))
	for _, e := range src {
		if strings.HasPrefix(e, "GIT_") {
			continue
		}
		out = append(out, e)
	}
	return out
}
