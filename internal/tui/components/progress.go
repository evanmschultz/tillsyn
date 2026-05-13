// MIGRATION TARGET: github.com/hylla-org/lykta
package components

import "charm.land/lipgloss/v2"

// Progress renders a single-line status message. It is a passive render-only struct,
// not a Bubble Tea sub-component (no Init/Update). It uses an inline lipgloss style
// and does NOT import internal/tui/style to avoid introducing a blocked_by D1 dependency.
type Progress struct {
	message string
}

// NewProgress constructs a Progress with the given initial message.
func NewProgress(message string) Progress {
	return Progress{message: message}
}

// View renders the progress message with a muted foreground style.
func (p Progress) View() string {
	style := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	return style.Render(p.message)
}

// WithMessage returns a copy of Progress with the message replaced.
func (p Progress) WithMessage(message string) Progress {
	p.message = message
	return p
}
