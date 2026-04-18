package tui

import "charm.land/bubbles/v2/key"

// filePickerKeymap bundles the key bindings used by the file-picker core surface.
//
// Bindings are mode-scoped — they are activated only when a file-picker variant
// is the active input mode. They are intentionally independent of the global
// keyMap so future file-picker variants (path-picker, file-ref-picker) can
// reuse the same core without leaking key choices into global help.
//
// Semantic contract for each binding:
//   - moveUp/moveDown: navigate the entry list.
//   - openDir: descend into the highlighted directory (right/l).
//   - parent: ascend into the parent directory (left/h).
//   - toggle: toggle multi-select on the highlighted entry (tab/space).
//   - accept: finalize the current selection (enter).
//   - cancel: leave the picker without mutating state (esc/ctrl+c).
//   - clear: clear the filter input (ctrl+u).
type filePickerKeymap struct {
	moveUp   key.Binding
	moveDown key.Binding
	openDir  key.Binding
	parent   key.Binding
	toggle   key.Binding
	accept   key.Binding
	cancel   key.Binding
	clear    key.Binding
}

// newFilePickerKeymap constructs the default file-picker keymap.
func newFilePickerKeymap() filePickerKeymap {
	return filePickerKeymap{
		moveUp:   key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "move up")),
		moveDown: key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "move down")),
		openDir:  key.NewBinding(key.WithKeys("right", "l"), key.WithHelp("→/l", "open dir")),
		parent:   key.NewBinding(key.WithKeys("left", "h"), key.WithHelp("←/h", "parent")),
		toggle:   key.NewBinding(key.WithKeys("tab", " ", "space"), key.WithHelp("tab/space", "toggle select")),
		accept:   key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "accept")),
		cancel:   key.NewBinding(key.WithKeys("esc", "ctrl+c"), key.WithHelp("esc", "cancel")),
		clear:    key.NewBinding(key.WithKeys("ctrl+u"), key.WithHelp("ctrl+u", "clear filter")),
	}
}

// shortHelp returns one short-help row describing the core bindings.
func (k filePickerKeymap) shortHelp() []key.Binding {
	return []key.Binding{k.moveUp, k.moveDown, k.openDir, k.parent, k.toggle, k.accept, k.cancel}
}
