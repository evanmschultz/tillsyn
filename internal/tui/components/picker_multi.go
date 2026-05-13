// MIGRATION TARGET: github.com/hylla-org/lykta

// Package components provides Bubble Tea sub-components for the Tillsyn TUI.
// These are composable building blocks, NOT standalone tea.Model implementations.
package components

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
)

// PickerMultiModel is a multi-selection list Bubble Tea sub-component.
// It does NOT implement tea.Model (Update returns a concrete type tuple, not
// tea.Model). The parent TUI polls Done(), Cancelled(), and Selected() after
// each Update call to advance its own state machine. Update never returns tea.Quit.
type PickerMultiModel struct {
	items     []string
	cursor    int
	selected  map[int]bool
	done      bool
	cancelled bool
}

// NewPickerMulti constructs a PickerMultiModel with the given items.
// items should be non-nil; an empty slice is accepted but produces a no-op picker.
func NewPickerMulti(items []string) PickerMultiModel {
	return PickerMultiModel{
		items:    items,
		selected: make(map[int]bool),
	}
}

// Init returns nil. PickerMultiModel requires no startup command.
func (m PickerMultiModel) Init() tea.Cmd {
	return nil
}

// Update handles incoming messages and returns the updated model and a command.
// On tea.KeyPressMsg:
//   - 'j': move cursor down (wraps to 0 at the bottom).
//   - 'k': move cursor up (wraps to len(items)-1 at the top).
//   - Space: toggle the selection state of the item at the cursor.
//   - Enter: confirm; sets done=true, cancelled stays false.
//   - Escape: cancel; sets done=true, cancelled=true.
//
// Update NEVER returns tea.Quit; terminal state is communicated via Done(),
// Cancelled(), and Selected(). Returns m unchanged for all other messages.
func (m PickerMultiModel) Update(msg tea.Msg) (PickerMultiModel, tea.Cmd) {
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
	case tea.KeySpace:
		// Clone the map before mutating to preserve value-semantics.
		next := make(map[int]bool, len(m.selected))
		for k, v := range m.selected {
			next[k] = v
		}
		next[m.cursor] = !next[m.cursor]
		m.selected = next
	case tea.KeyEnter:
		m.done = true
	case tea.KeyEscape:
		m.cancelled = true
		m.done = true
	}
	return m, nil
}

// View renders the item list with checkbox indicators and a '>' cursor.
// Each row shows '[ ]' for unselected items and '[x]' for selected items,
// with the cursor row prefixed by '>'.
func (m PickerMultiModel) View() string {
	if len(m.items) == 0 {
		return "(no items)"
	}
	var sb strings.Builder
	for i, item := range m.items {
		check := "[ ]"
		if m.selected[i] {
			check = "[x]"
		}
		cursor := "  "
		if i == m.cursor {
			cursor = "> "
		}
		sb.WriteString(fmt.Sprintf("%s%s %s\n", cursor, check, item))
	}
	return sb.String()
}

// Selected returns the selected items in their original list order.
// Returns nil (empty) when the picker was cancelled.
func (m PickerMultiModel) Selected() []string {
	if m.cancelled {
		return nil
	}
	var result []string
	for i, item := range m.items {
		if m.selected[i] {
			result = append(result, item)
		}
	}
	return result
}

// Done reports whether the picker has finished (either confirmed or cancelled).
func (m PickerMultiModel) Done() bool { return m.done }

// Cancelled reports whether the picker was cancelled via Escape.
func (m PickerMultiModel) Cancelled() bool { return m.cancelled }
