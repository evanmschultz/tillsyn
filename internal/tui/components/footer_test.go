// MIGRATION TARGET: github.com/hylla-org/lykta
package components

import (
	"strings"
	"testing"
)

// TestNewFooter verifies that NewFooter stores hints and width.
func TestNewFooter(t *testing.T) {
	hints := []string{"q: quit", "?: help"}
	f := NewFooter(hints, 80)
	if len(f.hints) != 2 {
		t.Errorf("hints len: got %d, want %d", len(f.hints), 2)
	}
	if f.width != 80 {
		t.Errorf("width: got %d, want %d", f.width, 80)
	}
}

// TestFooterWithWidth verifies that WithWidth returns a copy with the new width and
// leaves the original unchanged.
func TestFooterWithWidth(t *testing.T) {
	orig := NewFooter([]string{"q: quit"}, 40)
	updated := orig.WithWidth(100)
	if updated.width != 100 {
		t.Errorf("updated width: got %d, want %d", updated.width, 100)
	}
	if orig.width != 40 {
		t.Errorf("original width mutated: got %d, want %d", orig.width, 40)
	}
}

// TestFooterWithWidth_Zero verifies that WithWidth(0) returns a copy with width=0
// and leaves the original unchanged. Documents the future-composition intent: callers
// that intend to suppress width-driven layout can pass 0 and get a consistent round-trip.
func TestFooterWithWidth_Zero(t *testing.T) {
	orig := NewFooter([]string{"a"}, 10)
	updated := orig.WithWidth(0)
	if updated.width != 0 {
		t.Errorf("updated width: got %d, want 0", updated.width)
	}
	if orig.width != 10 {
		t.Errorf("original width mutated: got %d, want 10", orig.width)
	}
}

// TestFooterView_ContainsHints verifies that View renders each hint string.
func TestFooterView_ContainsHints(t *testing.T) {
	f := NewFooter([]string{"q: quit", "?: help"}, 80)
	out := f.View()
	if !strings.Contains(out, "q: quit") {
		t.Errorf("View() missing first hint: %q", out)
	}
	if !strings.Contains(out, "?: help") {
		t.Errorf("View() missing second hint: %q", out)
	}
}

// TestFooterView_EmptyHints verifies that View returns an empty string for a nil
// hints slice (no panic, no output).
func TestFooterView_EmptyHints(t *testing.T) {
	f := NewFooter(nil, 80)
	out := f.View()
	if out != "" {
		t.Errorf("View() with nil hints: got %q, want empty string", out)
	}
}

// TestFooterView_EmptySlice verifies that View returns an empty string for an
// explicitly empty (non-nil) hints slice.
func TestFooterView_EmptySlice(t *testing.T) {
	f := NewFooter([]string{}, 80)
	out := f.View()
	if out != "" {
		t.Errorf("View() with empty hints: got %q, want empty string", out)
	}
}

// TestFooterView_SingleHint verifies that View renders a single hint without a separator.
func TestFooterView_SingleHint(t *testing.T) {
	f := NewFooter([]string{"q: quit"}, 80)
	out := f.View()
	if !strings.Contains(out, "q: quit") {
		t.Errorf("View() missing single hint: %q", out)
	}
}
