// Package style provides semantic design tokens for the Tillsyn TUI.
// MIGRATION TARGET: github.com/hylla-org/lykta
package style

import "charm.land/lipgloss/v2"

// Typography styles map semantic text roles to lipgloss.Style values.
// Each style is constructed at package-init time via lipgloss.NewStyle()
// and references palette colors from palette.go.
//
// Consumers render text via:
//
//	output := style.Heading.Render("My Title")
//	output := style.Code.Render("fmt.Println")

// Heading renders primary-emphasis text — section titles and labels.
var Heading = lipgloss.NewStyle().
	Bold(true).
	Foreground(Primary)

// Body renders standard body text at normal weight.
var Body = lipgloss.NewStyle().
	Foreground(Primary)

// Label renders secondary-emphasis descriptive text — form labels, column headers.
var Label = lipgloss.NewStyle().
	Foreground(Secondary)

// Code renders inline code snippets in a monospace-apparent style.
// Terminal environments do not support font switching, so Code uses
// the Accent color to visually differentiate code spans from body text.
var Code = lipgloss.NewStyle().
	Foreground(Accent)

// MutedText renders low-emphasis supplementary text — hints, timestamps, footnotes.
// Named MutedText to avoid shadowing the Muted color.Color var in palette.go.
var MutedText = lipgloss.NewStyle().
	Faint(true).
	Foreground(Muted)
