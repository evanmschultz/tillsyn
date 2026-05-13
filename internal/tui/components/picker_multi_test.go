// MIGRATION TARGET: github.com/hylla-org/lykta
package components

import (
	"testing"

	tea "charm.land/bubbletea/v2"
)

// TestPickerMultiModel_Toggle verifies that Space toggles selection on and off
// for the item at the cursor position. No test step may return a non-nil tea.Cmd.
func TestPickerMultiModel_Toggle(t *testing.T) {
	items := []string{"red", "green", "blue"}
	m := NewPickerMulti(items)
	m.cursor = 1 // point at "green"

	// First Space: select "green".
	toggled, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeySpace})
	if cmd != nil {
		t.Errorf("Update returned non-nil cmd on Space, want nil")
	}
	if !toggled.selected[1] {
		t.Errorf("selected[1] = false after first Space, want true")
	}

	// Second Space: deselect "green".
	deselected, cmd := toggled.Update(tea.KeyPressMsg{Code: tea.KeySpace})
	if cmd != nil {
		t.Errorf("Update returned non-nil cmd on second Space, want nil")
	}
	if deselected.selected[1] {
		t.Errorf("selected[1] = true after second Space, want false")
	}
}

// TestPickerMultiModel_Navigation verifies j/k cursor movement and wrap behaviour.
// No test step may return a non-nil tea.Cmd.
func TestPickerMultiModel_Navigation(t *testing.T) {
	items := []string{"one", "two", "three"}
	m := NewPickerMulti(items)

	if m.cursor != 0 {
		t.Fatalf("initial cursor = %d, want 0", m.cursor)
	}

	t.Run("j moves cursor down", func(t *testing.T) {
		updated, cmd := m.Update(tea.KeyPressMsg{Code: 'j'})
		if cmd != nil {
			t.Errorf("Update returned non-nil cmd on j, want nil")
		}
		if updated.cursor != 1 {
			t.Errorf("cursor after j = %d, want 1", updated.cursor)
		}
	})

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

// TestPickerMultiModel_Confirm verifies that Enter sets Done()=true,
// Cancelled()=false, and Selected() returns the toggled items in original order.
// No test step may return a non-nil tea.Cmd.
func TestPickerMultiModel_Confirm(t *testing.T) {
	items := []string{"alpha", "beta", "gamma", "delta"}
	m := NewPickerMulti(items)

	// Toggle "alpha" (index 0) and "gamma" (index 2).
	m.cursor = 0
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeySpace})
	m.cursor = 2
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeySpace})

	// Confirm with Enter.
	confirmed, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd != nil {
		t.Errorf("Update returned non-nil cmd on Enter, want nil")
	}
	if !confirmed.Done() {
		t.Errorf("Done() = false after Enter, want true")
	}
	if confirmed.Cancelled() {
		t.Errorf("Cancelled() = true after Enter, want false")
	}

	got := confirmed.Selected()
	want := []string{"alpha", "gamma"}
	if len(got) != len(want) {
		t.Fatalf("Selected() = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("Selected()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

// TestPickerMultiModel_UpdatesAfterDone verifies that Update is a no-op once the
// model is in the done state. Parity with TestPickerSingleModel_Select
// "Updates after done are no-ops" subtest.
func TestPickerMultiModel_UpdatesAfterDone(t *testing.T) {
	items := []string{"red", "green", "blue"}
	m := NewPickerMulti(items)

	// Toggle "red" (index 0) and then confirm with Enter.
	m.cursor = 0
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeySpace})
	confirmed, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if !confirmed.Done() {
		t.Fatalf("precondition: Done() = false after Enter, want true")
	}

	// Capture state before the post-done update.
	cursorBefore := confirmed.cursor
	selectedBefore := confirmed.selected[0]

	// Send a Space key — should be a no-op on a done model.
	afterSpace, cmd := confirmed.Update(tea.KeyPressMsg{Code: tea.KeySpace})
	if cmd != nil {
		t.Errorf("Update returned non-nil cmd after done, want nil")
	}
	if afterSpace.cursor != cursorBefore {
		t.Errorf("cursor changed after done: got %d, want %d", afterSpace.cursor, cursorBefore)
	}
	if afterSpace.selected[0] != selectedBefore {
		t.Errorf("selected[0] changed after done: got %v, want %v", afterSpace.selected[0], selectedBefore)
	}
}

// TestPickerMultiModel_Cancel verifies that Escape sets Done()=true,
// Cancelled()=true, and Selected() returns empty even when items were toggled.
// No test step may return a non-nil tea.Cmd.
func TestPickerMultiModel_Cancel(t *testing.T) {
	items := []string{"x", "y", "z"}
	m := NewPickerMulti(items)

	// Toggle "x" before cancelling.
	m.cursor = 0
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeySpace})

	// Cancel with Escape.
	cancelled, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if cmd != nil {
		t.Errorf("Update returned non-nil cmd on Escape, want nil")
	}
	if !cancelled.Done() {
		t.Errorf("Done() = false after Escape, want true")
	}
	if !cancelled.Cancelled() {
		t.Errorf("Cancelled() = false after Escape, want true")
	}

	got := cancelled.Selected()
	if len(got) != 0 {
		t.Errorf("Selected() = %v after Escape, want empty", got)
	}
}
