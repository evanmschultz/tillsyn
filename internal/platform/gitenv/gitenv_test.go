package gitenv

import (
	"slices"
	"strings"
	"testing"
)

// TestFilteredDropsGitKeysAndRetainsOthers pins the core contract: every
// "GIT_*=..." entry is stripped while non-GIT_ entries (HOME and friends)
// survive untouched. We use t.Setenv so the env mutation is automatically
// reverted on test exit and the helper is goroutine-safe under -race.
func TestFilteredDropsGitKeysAndRetainsOthers(t *testing.T) {
	// t.Setenv prohibits t.Parallel — leave this test serial so the env
	// mutations are deterministic.
	t.Setenv("GIT_DIR", "/foo")
	t.Setenv("HOME", "/bar")

	got := Filtered()

	for _, e := range got {
		if strings.HasPrefix(e, "GIT_") {
			t.Errorf("Filtered() leaked GIT_* entry %q", e)
		}
	}

	if !slices.Contains(got, "HOME=/bar") {
		t.Errorf("Filtered() dropped HOME=/bar; got %v", got)
	}
	if slices.Contains(got, "GIT_DIR=/foo") {
		t.Errorf("Filtered() retained GIT_DIR=/foo; got %v", got)
	}
}

// TestFilteredStripsAllGitPrefixVariants verifies every GIT_-prefixed key in
// the parent env is filtered, not just GIT_DIR. This pins the prefix-match
// contract called out in the package doc-comment.
func TestFilteredStripsAllGitPrefixVariants(t *testing.T) {
	t.Setenv("GIT_DIR", "/foo")
	t.Setenv("GIT_INDEX_FILE", "/foo/index")
	t.Setenv("GIT_WORK_TREE", "/foo/work")
	t.Setenv("GIT_PREFIX", "")
	t.Setenv("HOME", "/bar")

	got := Filtered()

	for _, key := range []string{"GIT_DIR=", "GIT_INDEX_FILE=", "GIT_WORK_TREE=", "GIT_PREFIX="} {
		for _, e := range got {
			if strings.HasPrefix(e, key) {
				t.Errorf("Filtered() leaked %s entry %q", strings.TrimSuffix(key, "="), e)
			}
		}
	}
	if !slices.Contains(got, "HOME=/bar") {
		t.Errorf("Filtered() dropped HOME=/bar; got %v", got)
	}
}

// TestFilteredReturnsFreshSliceSafeForAppend verifies the returned slice has
// independent backing storage so callers can append without mutating the
// shared os.Environ() snapshot.
func TestFilteredReturnsFreshSliceSafeForAppend(t *testing.T) {
	t.Setenv("HOME", "/bar")

	first := Filtered()
	firstLen := len(first)

	// Append a sentinel that should NOT appear in the next Filtered() call.
	first = append(first, "SENTINEL_NOT_IN_ENV=1")

	second := Filtered()
	if len(second) != firstLen {
		t.Errorf("second Filtered() len=%d want %d (append leaked back into os.Environ snapshot)", len(second), firstLen)
	}
	if slices.Contains(second, "SENTINEL_NOT_IN_ENV=1") {
		t.Errorf("second Filtered() contains sentinel from prior append; backing store is shared")
	}
}
