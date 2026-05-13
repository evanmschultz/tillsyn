// MIGRATION TARGET: github.com/hylla-org/lykta
package style

import "testing"

// TestAllColors_NonEmpty verifies that AllColors returns a non-empty slice
// and that every color is a non-nil, non-zero value. This test is mandatory
// for the internal/tui/style package: pure-var/const packages without test
// functions cause the magefile coverage runner to error with "no coverage
// rows parsed."
func TestAllColors_NonEmpty(t *testing.T) {
	colors := AllColors()
	if len(colors) == 0 {
		t.Fatal("AllColors() returned empty slice; expected at least one color")
	}
	for i, c := range colors {
		if c == nil {
			t.Errorf("AllColors()[%d] is nil; expected a non-nil color", i)
			continue
		}
		// Verify the color encodes to non-zero RGBA. A zero result
		// would indicate a missing or unset color value.
		r, g, b, a := c.RGBA()
		if r == 0 && g == 0 && b == 0 && a == 0 {
			t.Errorf("AllColors()[%d] RGBA is all-zero; expected a real color value", i)
		}
	}
}
