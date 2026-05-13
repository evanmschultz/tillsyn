// MIGRATION TARGET: github.com/hylla-org/lykta
package components

import (
	"testing"

	tea "charm.land/bubbletea/v2"
)

// TestPickerSingleModel_Navigation verifies that j/k move the cursor and that
// wrapping occurs correctly at both list boundaries.
// No test step may return a non-nil tea.Cmd.
func TestPickerSingleModel_Navigation(t *testing.T) {
	items := []string{"alpha", "beta", "gamma"}
	m := NewPickerSingle(items)

	// Initial cursor is at 0.
	t.Run("initial_cursor_is_zero", func(t *testing.T) {
		if m.cursor != 0 {
			t.Fatalf("initial cursor = %d, want 0", m.cursor)
		}
	})

	// j moves cursor down.
	t.Run("j moves cursor down", func(t *testing.T) {
		updated, cmd := m.Update(tea.KeyPressMsg{Code: 'j'})
		if cmd != nil {
			t.Errorf("Update returned non-nil cmd on j, want nil")
		}
		if updated.cursor != 1 {
			t.Errorf("cursor after j = %d, want 1", updated.cursor)
		}
	})

	// k moves cursor up.
	t.Run("k moves cursor up", func(t *testing.T) {
		atOne := m
		atOne.cursor = 1
		updated, cmd := atOne.Update(tea.KeyPressMsg{Code: 'k'})
		if cmd != nil {
			t.Errorf("Update returned non-nil cmd on k, want nil")
		}
		if updated.cursor != 0 {
			t.Errorf("cursor after k = %d, want 0", updated.cursor)
		}
	})

	// j wraps at the bottom boundary.
	t.Run("j wraps at bottom", func(t *testing.T) {
		atEnd := m
		atEnd.cursor = len(items) - 1
		updated, cmd := atEnd.Update(tea.KeyPressMsg{Code: 'j'})
		if cmd != nil {
			t.Errorf("Update returned non-nil cmd on j at bottom, want nil")
		}
		if updated.cursor != 0 {
			t.Errorf("cursor after j at bottom = %d, want 0 (wrap)", updated.cursor)
		}
	})

	// k wraps at the top boundary.
	t.Run("k wraps at top", func(t *testing.T) {
		atTop := m
		atTop.cursor = 0
		updated, cmd := atTop.Update(tea.KeyPressMsg{Code: 'k'})
		if cmd != nil {
			t.Errorf("Update returned non-nil cmd on k at top, want nil")
		}
		if updated.cursor != len(items)-1 {
			t.Errorf("cursor after k at top = %d, want %d (wrap)", updated.cursor, len(items)-1)
		}
	})
}

// TestPickerSingleModel_Select verifies Enter confirms the selection and Escape
// cancels without a selection. No test step may return a non-nil tea.Cmd.
func TestPickerSingleModel_Select(t *testing.T) {
	items := []string{"apple", "banana", "cherry"}

	// Enter confirms the cursor item.
	t.Run("Enter confirms selection at cursor", func(t *testing.T) {
		m := NewPickerSingle(items)
		m.cursor = 1 // "banana"

		updated, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
		if cmd != nil {
			t.Errorf("Update returned non-nil cmd on Enter, want nil")
		}
		if !updated.Done() {
			t.Errorf("Done() = false after Enter, want true")
		}
		if updated.Selected() != "banana" {
			t.Errorf("Selected() = %q, want %q", updated.Selected(), "banana")
		}
	})

	// Escape cancels without a selection.
	t.Run("Escape cancels without selection", func(t *testing.T) {
		m := NewPickerSingle(items)
		m.cursor = 2 // "cherry"

		updated, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
		if cmd != nil {
			t.Errorf("Update returned non-nil cmd on Escape, want nil")
		}
		if !updated.Done() {
			t.Errorf("Done() = false after Escape, want true")
		}
		if updated.Selected() != "" {
			t.Errorf("Selected() = %q after Escape, want empty string", updated.Selected())
		}
	})

	// Updates after done are no-ops (model is frozen).
	t.Run("Updates after done are no-ops", func(t *testing.T) {
		m := NewPickerSingle(items)
		m.cursor = 0
		confirmed, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
		// Further j press should not change cursor on a done model.
		afterJ, cmd := confirmed.Update(tea.KeyPressMsg{Code: 'j'})
		if cmd != nil {
			t.Errorf("Update returned non-nil cmd after done, want nil")
		}
		if afterJ.cursor != confirmed.cursor {
			t.Errorf("cursor changed after done: got %d, want %d", afterJ.cursor, confirmed.cursor)
		}
	})
}
