package buildinfo

import "testing"

// TestResolvedCommitDefaultsToUnknown confirms an unstamped binary reports the sentinel.
func TestResolvedCommitDefaultsToUnknown(t *testing.T) {
	t.Cleanup(restoreCommit(Commit))
	Commit = ""
	if got := ResolvedCommit(); got != "unknown" {
		t.Fatalf("ResolvedCommit() = %q, want %q", got, "unknown")
	}
}

// TestResolvedCommitTrimsWhitespace keeps ldflags injection resilient to stray whitespace.
func TestResolvedCommitTrimsWhitespace(t *testing.T) {
	t.Cleanup(restoreCommit(Commit))
	Commit = "  abc1234  "
	if got := ResolvedCommit(); got != "abc1234" {
		t.Fatalf("ResolvedCommit() = %q, want %q", got, "abc1234")
	}
}

// TestResolvedCommitReturnsInjectedValue covers the happy path for a stamped binary.
func TestResolvedCommitReturnsInjectedValue(t *testing.T) {
	t.Cleanup(restoreCommit(Commit))
	Commit = "7188ab5"
	if got := ResolvedCommit(); got != "7188ab5" {
		t.Fatalf("ResolvedCommit() = %q, want %q", got, "7188ab5")
	}
}

// TestIsDirtyCases covers the small decision table exhaustively so contract drift fails a test.
func TestIsDirtyCases(t *testing.T) {
	cases := []struct {
		name  string
		value string
		want  bool
	}{
		{name: "empty is clean", value: "", want: false},
		{name: "false literal is clean", value: "false", want: false},
		{name: "upper FALSE is clean", value: "FALSE", want: false},
		{name: "mixed-case False is clean", value: "False", want: false},
		{name: "whitespace false is clean", value: "  false  ", want: false},
		{name: "true literal is dirty", value: "true", want: true},
		{name: "1 is dirty", value: "1", want: true},
		{name: "yes is dirty", value: "yes", want: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Cleanup(restoreDirty(Dirty))
			Dirty = tc.value
			if got := IsDirty(); got != tc.want {
				t.Fatalf("IsDirty() with Dirty=%q = %v, want %v", tc.value, got, tc.want)
			}
		})
	}
}

// TestSummaryComposesCommitAndDirty verifies the CLI-rendered descriptor for all corner cases.
func TestSummaryComposesCommitAndDirty(t *testing.T) {
	cases := []struct {
		name   string
		commit string
		dirty  string
		want   string
	}{
		{name: "unstamped clean", commit: "", dirty: "", want: "unknown"},
		{name: "unstamped dirty", commit: "", dirty: "true", want: "unknown-dirty"},
		{name: "stamped clean", commit: "7188ab5", dirty: "", want: "7188ab5"},
		{name: "stamped dirty", commit: "7188ab5", dirty: "true", want: "7188ab5-dirty"},
		{name: "stamped explicit false is clean", commit: "7188ab5", dirty: "false", want: "7188ab5"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Cleanup(restoreCommit(Commit))
			t.Cleanup(restoreDirty(Dirty))
			Commit = tc.commit
			Dirty = tc.dirty
			if got := Summary(); got != tc.want {
				t.Fatalf("Summary() with Commit=%q Dirty=%q = %q, want %q", tc.commit, tc.dirty, got, tc.want)
			}
		})
	}
}

// restoreCommit returns a cleanup func that restores Commit to its previous value.
func restoreCommit(previous string) func() {
	return func() { Commit = previous }
}

// restoreDirty returns a cleanup func that restores Dirty to its previous value.
func restoreDirty(previous string) func() {
	return func() { Dirty = previous }
}
