// MIGRATION TARGET: github.com/hylla-org/lykta

// Package components provides Bubble Tea sub-components for the Tillsyn TUI.
// These are composable building blocks, NOT standalone tea.Model implementations.
// They expose Update(tea.Msg) (X, tea.Cmd), View() string, and typed accessor methods;
// they are composed into the outer tea.Model at internal/tui/model.go.
package components

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// ConfirmModel is a y/n prompt Bubble Tea sub-component.
// It does NOT implement tea.Model (View() returns string, not tea.View).
// The parent TUI polls Confirmed(), Cancelled(), and Done() after each Update call
// to advance its own state machine. Update never returns tea.Quit.
type ConfirmModel struct {
	prompt     string
	defaultYes bool
	confirmed  bool
	cancelled  bool
	done       bool
}

// NewConfirm constructs a ConfirmModel with the given prompt text.
// When defaultYes is true, pressing Enter confirms; when false, pressing Enter cancels.
func NewConfirm(prompt string, defaultYes bool) ConfirmModel {
	return ConfirmModel{
		prompt:     prompt,
		defaultYes: defaultYes,
	}
}

// Init returns nil — ConfirmModel requires no setup command.
func (m ConfirmModel) Init() tea.Cmd {
	return nil
}

// Update handles incoming messages and returns the updated model and a nil command.
// Handled keys: y/Y (confirm), n/N (cancel), Enter (default action), Escape (cancel).
// All other messages are ignored. Update NEVER returns tea.Quit; done state is
// communicated via the Confirmed(), Cancelled(), and Done() accessors.
func (m ConfirmModel) Update(msg tea.Msg) (ConfirmModel, tea.Cmd) {
	kp, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return m, nil
	}
	switch kp.Code {
	case 'y', 'Y':
		m.confirmed = true
		m.done = true
	case 'n', 'N':
		m.cancelled = true
		m.done = true
	case tea.KeyEnter:
		if m.defaultYes {
			m.confirmed = true
		} else {
			m.cancelled = true
		}
		m.done = true
	case tea.KeyEsc:
		m.cancelled = true
		m.done = true
	}
	return m, nil
}

// View renders the prompt with a [Y/n] or [y/N] indicator reflecting the default.
func (m ConfirmModel) View() string {
	indicator := "[y/N]"
	if m.defaultYes {
		indicator = "[Y/n]"
	}
	promptStyle := lipgloss.NewStyle().Bold(true)
	hintStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	return promptStyle.Render(m.prompt) + " " + hintStyle.Render(indicator) + " "
}

// Confirmed reports whether the user pressed y/Y or Enter (when defaultYes is true).
func (m ConfirmModel) Confirmed() bool { return m.confirmed }

// Cancelled reports whether the user pressed n/N, Escape, or Enter (when defaultYes is false).
func (m ConfirmModel) Cancelled() bool { return m.cancelled }

// Done reports whether the user has made a choice (confirmed or cancelled).
func (m ConfirmModel) Done() bool { return m.done }
