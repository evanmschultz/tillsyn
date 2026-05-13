// MIGRATION TARGET: github.com/hylla-org/lykta

// Package components provides Bubble Tea sub-components for the Tillsyn TUI.
// These are composable building blocks, NOT standalone tea.Model implementations.
package components

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
)

// PickerSingleModel is a single-selection list Bubble Tea sub-component.
// It does NOT implement tea.Model (Update returns a concrete type tuple, not
// tea.Model). The parent TUI polls Done() and Selected() after each Update
// call to advance its own state machine. Update never returns tea.Quit.
type PickerSingleModel struct {
	items    []string
	cursor   int
	selected string
	done     bool
}

// NewPickerSingle constructs a PickerSingleModel with the given items.
// items should be non-nil; an empty slice is accepted but produces a no-op picker.
func NewPickerSingle(items []string) PickerSingleModel {
	return PickerSingleModel{
		items: items,
	}
}

// Init returns nil. PickerSingleModel requires no startup command.
func (m PickerSingleModel) Init() tea.Cmd {
	return nil
}

// Update handles incoming messages and returns the updated model and a command.
// On tea.KeyPressMsg:
//   - 'j': move cursor down (wraps to 0 at the bottom).
//   - 'k': move cursor up (wraps to len(items)-1 at the top).
//   - Enter: confirm selection; sets selected=items[cursor] and done=true.
//   - Escape: cancel without selection; sets done=true, selected stays empty.
//
// Update NEVER returns tea.Quit; terminal state is communicated via Done() and
// Selected(). Returns m unchanged for all other messages.
func (m PickerSingleModel) Update(msg tea.Msg) (PickerSingleModel, tea.Cmd) {
	if m.done || len(m.items) == 0 {
		return m, nil
	}
	kp, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return m, nil
	}
	switch kp.Code {
	case 'j':
		m.cursor = (m.cursor + 1) % len(m.items)
	case 'k':
		m.cursor = (m.cursor - 1 + len(m.items)) % len(m.items)
	case tea.KeyEnter:
		m.selected = m.items[m.cursor]
		m.done = true
	case tea.KeyEscape:
		m.done = true
	}
	return m, nil
}

// View renders the item list with a '>' cursor indicator on the active row.
// The selected item (once confirmed) is highlighted with brackets.
func (m PickerSingleModel) View() string {
	if len(m.items) == 0 {
		return "(no items)"
	}
	var sb strings.Builder
	for i, item := range m.items {
		if i == m.cursor {
			if m.done && m.selected == item {
				sb.WriteString(fmt.Sprintf("> [%s]\n", item))
			} else {
				sb.WriteString(fmt.Sprintf("> %s\n", item))
			}
		} else {
			sb.WriteString(fmt.Sprintf("  %s\n", item))
		}
	}
	return sb.String()
}

// Selected returns the confirmed selection string, or the empty string if the
// picker was cancelled or has not yet been confirmed.
func (m PickerSingleModel) Selected() string { return m.selected }

// Done reports whether the picker has finished (either confirmed or cancelled).
func (m PickerSingleModel) Done() bool { return m.done }
