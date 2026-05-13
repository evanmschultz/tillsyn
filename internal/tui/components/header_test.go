// MIGRATION TARGET: github.com/hylla-org/lykta
package components

import (
	"strings"
	"testing"
)

// TestNewHeader verifies that NewHeader stores title, subtitle, and width.
func TestNewHeader(t *testing.T) {
	h := NewHeader("My Title", "sub", 80)
	if h.title != "My Title" {
		t.Errorf("title: got %q, want %q", h.title, "My Title")
	}
	if h.subtitle != "sub" {
		t.Errorf("subtitle: got %q, want %q", h.subtitle, "sub")
	}
	if h.width != 80 {
		t.Errorf("width: got %d, want %d", h.width, 80)
	}
}

// TestHeaderWithWidth verifies that WithWidth returns a copy with the new width and
// leaves the original unchanged.
func TestHeaderWithWidth(t *testing.T) {
	orig := NewHeader("T", "S", 40)
	updated := orig.WithWidth(120)
	if updated.width != 120 {
		t.Errorf("updated width: got %d, want %d", updated.width, 120)
	}
	if orig.width != 40 {
		t.Errorf("original width mutated: got %d, want %d", orig.width, 40)
	}
}

// TestHeaderView_ContainsTitleAndSubtitle verifies that View renders both the title
// and subtitle strings.
func TestHeaderView_ContainsTitleAndSubtitle(t *testing.T) {
	h := NewHeader("MyTitle", "MySub", 60)
	out := h.View()
	if !strings.Contains(out, "MyTitle") {
		t.Errorf("View() missing title: %q", out)
	}
	if !strings.Contains(out, "MySub") {
		t.Errorf("View() missing subtitle: %q", out)
	}
}

// TestHeaderView_ZeroWidth verifies that View does not panic when width is zero.
func TestHeaderView_ZeroWidth(t *testing.T) {
	h := NewHeader("T", "S", 0)
	out := h.View()
	if !strings.Contains(out, "T") {
		t.Errorf("View() missing title: %q", out)
	}
}

// TestHeaderView_WidthSmallerThanContent verifies that View does not panic when the
// combined rendered width exceeds h.width (gap clamped to zero).
func TestHeaderView_WidthSmallerThanContent(t *testing.T) {
	h := NewHeader("LongTitle", "LongSubtitle", 5)
	out := h.View()
	if !strings.Contains(out, "LongTitle") {
		t.Errorf("View() missing title on narrow width: %q", out)
	}
}
