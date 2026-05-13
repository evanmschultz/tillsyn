// MIGRATION TARGET: github.com/hylla-org/lykta
// Package keybindings provides a vim-style keybinding dispatcher for Tillsyn's TUI.
package keybindings

import tea "charm.land/bubbletea/v2"

// Mode represents a vim-style input mode.
type Mode int

const (
	// ModeNav is the default mode; letter and arrow keys navigate.
	ModeNav Mode = iota
	// ModeInsert is active while typing in any text field; single-key bindings disabled.
	ModeInsert
	// ModeVisual is selection mode for items in a list.
	ModeVisual
	// ModeVisualBlock is selection mode for blocks in a rendered document.
	ModeVisualBlock
	// ModeCommand is active after `:` — command palette active.
	ModeCommand
	// ModeHint is active after `f`/`F` — overlay codes on clickable elements.
	ModeHint
)

// String returns the lowercase name of the mode.
func (m Mode) String() string {
	switch m {
	case ModeNav:
		return "nav"
	case ModeInsert:
		return "insert"
	case ModeVisual:
		return "visual"
	case ModeVisualBlock:
		return "visual-block"
	case ModeCommand:
		return "command"
	case ModeHint:
		return "hint"
	default:
		return "unknown"
	}
}

// HandlerFunc is a function that handles a keybinding action and returns an optional tea.Cmd.
type HandlerFunc func() tea.Cmd

// NoOp is the no-op HandlerFunc returned when no binding is registered for a key in the
// current mode. Callers may compare via pointer identity is not reliable; use the return
// value of Dispatch and check whether the returned cmd is nil after invocation.
var NoOp HandlerFunc = func() tea.Cmd { return nil }
