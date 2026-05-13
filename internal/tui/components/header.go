// MIGRATION TARGET: github.com/hylla-org/lykta

// Package components provides Bubble Tea sub-components for the Tillsyn TUI.
// These are composable building blocks, NOT standalone tea.Model implementations.
package components

import (
	"strings"

	"charm.land/lipgloss/v2"
)

// Header renders a styled top chrome bar.
// It is a passive render struct — it has no Init(), no Update(), and no internal
// state machine. Callers construct it via NewHeader, adjust width via WithWidth,
// and call View() to obtain a rendered string for the parent tea.Model.
type Header struct {
	title    string
	subtitle string
	width    int
}

// NewHeader constructs a Header with the given title, subtitle, and display width.
// A zero or negative width is accepted; View() renders without right-alignment padding.
func NewHeader(title, subtitle string, width int) Header {
	return Header{
		title:    title,
		subtitle: subtitle,
		width:    width,
	}
}

// WithWidth returns a copy of the Header with the display width set to w.
// The original Header is not modified.
func (h Header) WithWidth(w int) Header {
	h.width = w
	return h
}

// View renders the header as a full-width bar: title left-aligned, subtitle
// right-aligned, with a gap filled by spaces to reach h.width columns.
// Inline lipgloss styles are used: title is bold primary text, subtitle is muted.
// If the combined rendered width of title and subtitle exceeds h.width, the gap
// is zero (no padding) and both values are still rendered without truncation.
func (h Header) View() string {
	titleStyle := lipgloss.NewStyle().Bold(true)
	subtitleStyle := lipgloss.NewStyle().Faint(true).Foreground(lipgloss.Color("#6e7280"))

	left := titleStyle.Render(h.title)
	right := subtitleStyle.Render(h.subtitle)

	leftW := lipgloss.Width(left)
	rightW := lipgloss.Width(right)

	gap := h.width - leftW - rightW
	if gap < 0 {
		gap = 0
	}

	return left + strings.Repeat(" ", gap) + right
}
