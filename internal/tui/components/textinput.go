// MIGRATION TARGET: github.com/hylla-org/lykta

// Package components provides Bubble Tea sub-components for the Tillsyn TUI.
// These are composable building blocks, NOT standalone tea.Model implementations.
// They expose Update(tea.Msg) (X, tea.Cmd), View() string, and typed accessor methods;
// they are composed into the outer tea.Model at internal/tui/model.go.
package components

import (
	"fmt"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
)

// TextInputModel is a single-line text-entry Bubble Tea sub-component wrapping
// charm.land/bubbles/v2/textinput. It does NOT implement tea.Model (Update returns
// a concrete type tuple, not tea.Model). The parent TUI polls Submitted() and Err()
// after each Update call to advance its own state machine. Update never injects
// tea.Quit into the command stream; pass-through of inner-model commands on non-Enter
// messages is unchanged and depends on the upstream bubbles library.
type TextInputModel struct {
	inner     textinput.Model
	validate  func(string) error
	err       error
	submitted bool
}

// NewTextInput constructs a TextInputModel with the given placeholder text and
// optional validation function. When validate is nil, any value is accepted on Enter.
// validate is invoked on every Update call including cursor-blink messages; keep it
// cheap and side-effect-free.
func NewTextInput(placeholder string, validate func(string) error) TextInputModel {
	ti := textinput.New()
	ti.Placeholder = placeholder
	_ = ti.Focus() // Focus has pointer-receiver side-effects (sets focused state); the
	// returned blink cmd is intentionally discarded here — Init() returns textinput.Blink
	// to start the blink loop via the Bubble Tea runtime instead.
	return TextInputModel{
		inner:    ti,
		validate: validate,
	}
}

// Init returns the cursor-blink command so the parent program can start the blink
// animation. Delegates to the package-level textinput.Blink variable per the
// bubbles v2 pattern (textinput.Model has no Init method).
func (m TextInputModel) Init() tea.Cmd {
	return textinput.Blink
}

// Update handles incoming messages and returns the updated model and a command.
// On a tea.KeyPressMsg with the Enter key, validate is called if non-nil:
//   - validation passes: submitted is set to true, err is cleared to nil.
//   - validation fails: submitted stays false, err is set to the validation error.
//
// On all other messages, the message is delegated to the inner textinput model and
// validate is re-run to keep err current (so errors clear as the user types valid input
// and appear as soon as the value becomes invalid). The inner cmd is propagated.
//
// Update never injects tea.Quit; terminal state is communicated via Submitted() and Err().
func (m TextInputModel) Update(msg tea.Msg) (TextInputModel, tea.Cmd) {
	kp, ok := msg.(tea.KeyPressMsg)
	if ok && kp.Code == tea.KeyEnter {
		// Delegate to inner first so any internal Enter handling occurs.
		var cmd tea.Cmd
		m.inner, cmd = m.inner.Update(msg)
		// Discard inner cmd on Enter: Enter is a terminal event for this wrapper and
		// we always return nil, suppressing any side-effect command the inner model might
		// produce. This is a deliberate coupling point to bubbles internals — if a future
		// version of bubbles/textinput gains an Enter-side-effect command, it will be
		// silently dropped here by design.
		_ = cmd

		if m.validate != nil {
			if err := m.validate(m.inner.Value()); err != nil {
				m.err = err
				return m, nil
			}
		}
		m.submitted = true
		m.err = nil
		return m, nil
	}

	// For non-Enter messages, delegate to inner and propagate its command, then
	// re-run validate so err reflects the current value after every keystroke.
	var cmd tea.Cmd
	m.inner, cmd = m.inner.Update(msg)
	if m.validate != nil {
		m.err = m.validate(m.inner.Value())
	}
	return m, cmd
}

// View renders the inner textinput component and appends the validation error
// on a new line when Err() is non-nil.
func (m TextInputModel) View() string {
	v := m.inner.View()
	if m.err != nil {
		v += fmt.Sprintf("\n%s", m.err.Error())
	}
	return v
}

// Value returns the current text entered in the input field.
func (m TextInputModel) Value() string { return m.inner.Value() }

// Err returns the most recent validation error, or nil if the last validation passed
// or no validation has been attempted.
func (m TextInputModel) Err() error { return m.err }

// Submitted reports whether the user pressed Enter and the value passed validation
// (or no validator was set).
func (m TextInputModel) Submitted() bool { return m.submitted }
