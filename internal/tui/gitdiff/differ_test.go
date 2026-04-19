package gitdiff

import (
	"context"
	"errors"
	"testing"
)

// TestDivergenceStatus_String asserts the stable, short labels the package
// publishes for each enum value. The labels feed log messages and the
// diff-pane banner text, so regressions here are user-visible.
func TestDivergenceStatus_String(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		status DivergenceStatus
		want   string
	}{
		{name: "ancestor", status: DivergenceAncestor, want: "ancestor"},
		{name: "diverged", status: DivergenceDiverged, want: "diverged"},
		{name: "unknown", status: DivergenceUnknown, want: "unknown"},
		{name: "out of range", status: DivergenceStatus(99), want: "unknown"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := tc.status.String()
			if got != tc.want {
				t.Fatalf("DivergenceStatus(%d).String() = %q, want %q", tc.status, got, tc.want)
			}
		})
	}
}

// TestNewExecDiffer_ReturnsInterface pins the constructor contract: it must
// hand back a value typed as the Differ interface so production callers
// cannot accidentally bind to the concrete struct and break the abstraction.
func TestNewExecDiffer_ReturnsInterface(t *testing.T) {
	t.Parallel()

	var differ Differ = NewExecDiffer()
	if differ == nil {
		t.Fatal("NewExecDiffer() returned nil")
	}
}

// TestDiffer_EmptyRevisionRejected verifies that empty revision strings are
// rejected with ErrEmptyRevision before any exec happens. The test uses a
// nil-directory differ — it never reaches git, so there is nothing to stage.
func TestDiffer_EmptyRevisionRejected(t *testing.T) {
	t.Parallel()

	d := NewExecDiffer()
	_, err := d.Diff(context.Background(), "", "HEAD", nil)
	if !errors.Is(err, ErrEmptyRevision) {
		t.Fatalf("empty start: got err=%v, want ErrEmptyRevision", err)
	}

	_, err = d.Diff(context.Background(), "HEAD", "   ", nil)
	if !errors.Is(err, ErrEmptyRevision) {
		t.Fatalf("blank end: got err=%v, want ErrEmptyRevision", err)
	}
}
