// MIGRATION TARGET: github.com/hylla-org/lykta

// Package components provides Bubble Tea sub-components for the Tillsyn TUI.
// These are composable building blocks, NOT standalone tea.Model implementations.
package components

import (
	"strings"

	"charm.land/lipgloss/v2"
)

// Footer renders a styled bottom chrome bar with key hints.
// It is a passive render struct — it has no Init(), no Update(), and no internal
// state machine. Callers construct it via NewFooter, adjust width via WithWidth,
// and call View() to obtain a rendered string for the parent tea.Model.
type Footer struct {
	hints []string
	width int
}

// NewFooter constructs a Footer with the given key hints and display width.
// hints may be nil or empty; View() returns an empty string in that case.
// A zero or negative width is accepted and has no effect on the rendered output.
func NewFooter(hints []string, width int) Footer {
	return Footer{
		hints: hints,
		width: width,
	}
}

// WithWidth returns a copy of the Footer with the display width set to w.
// The original Footer is not modified.
func (f Footer) WithWidth(w int) Footer {
	f.width = w
	return f
}

// View renders the footer as a horizontal list of hints separated by " · ".
// Each hint is rendered in a muted (faint, low-emphasis) style.
// An empty or nil hints slice produces an empty string.
func (f Footer) View() string {
	if len(f.hints) == 0 {
		return ""
	}
	hintStyle := lipgloss.NewStyle().Faint(true).Foreground(lipgloss.Color("#6e7280"))

	rendered := make([]string, len(f.hints))
	for i, h := range f.hints {
		rendered[i] = hintStyle.Render(h)
	}
	return strings.Join(rendered, " · ")
}
