package tui

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"charm.land/bubbles/v2/key"
)

// keyMap represents key map data used by this package.
type keyMap struct {
	quit                 key.Binding
	reload               key.Binding
	toggleHelp           key.Binding
	moveLeft             key.Binding
	moveRight            key.Binding
	moveUp               key.Binding
	moveDown             key.Binding
	addActionItem        key.Binding
	actionItemInfo       key.Binding
	editActionItem       key.Binding
	newProject           key.Binding
	editProject          key.Binding
	commandPalette       key.Binding
	quickActions         key.Binding
	deleteActionItem     key.Binding
	archiveActionItem    key.Binding
	moveActionItemLeft   key.Binding
	moveActionItemRight  key.Binding
	hardDeleteActionItem key.Binding
	restoreActionItem    key.Binding
	search               key.Binding
	projects             key.Binding
	toggleArchived       key.Binding
	toggleSelectMode     key.Binding
	focusSubtree         key.Binding
	clearFocus           key.Binding
	multiSelect          key.Binding
	activityLog          key.Binding
	undo                 key.Binding
	redo                 key.Binding
	diffModeToggle       key.Binding
}

// newKeyMap constructs key map.
func newKeyMap() keyMap {
	return keyMap{
		quit:                 key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
		reload:               key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "reload")),
		toggleHelp:           key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "toggle help")),
		moveLeft:             key.NewBinding(key.WithKeys("h", "left"), key.WithHelp("h/←", "column left")),
		moveRight:            key.NewBinding(key.WithKeys("l", "right"), key.WithHelp("l/→", "column right")),
		moveUp:               key.NewBinding(key.WithKeys("k", "up"), key.WithHelp("k/↑", "move up")),
		moveDown:             key.NewBinding(key.WithKeys("j", "down"), key.WithHelp("j/↓", "move down")),
		addActionItem:        key.NewBinding(key.WithKeys("n"), key.WithHelp("n", "new actionItem")),
		actionItemInfo:       key.NewBinding(key.WithKeys("i"), key.WithHelp("i", "actionItem info")),
		editActionItem:       key.NewBinding(key.WithKeys("e"), key.WithHelp("e", "edit actionItem")),
		newProject:           key.NewBinding(key.WithKeys("N"), key.WithHelp("N", "new project")),
		editProject:          key.NewBinding(key.WithKeys("M"), key.WithHelp("M", "edit project")),
		commandPalette:       key.NewBinding(key.WithKeys(":"), key.WithHelp(":", "command palette")),
		quickActions:         key.NewBinding(key.WithKeys("."), key.WithHelp(".", "quick actions")),
		deleteActionItem:     key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "delete (default)")),
		archiveActionItem:    key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "archive actionItem")),
		moveActionItemLeft:   key.NewBinding(key.WithKeys("["), key.WithHelp("[", "move actionItem left")),
		moveActionItemRight:  key.NewBinding(key.WithKeys("]"), key.WithHelp("]", "move actionItem right")),
		hardDeleteActionItem: key.NewBinding(key.WithKeys("D", "shift+d"), key.WithHelp("D", "hard delete")),
		restoreActionItem:    key.NewBinding(key.WithKeys("u"), key.WithHelp("u", "restore actionItem")),
		search:               key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "search")),
		projects:             key.NewBinding(key.WithKeys("p", "P"), key.WithHelp("p/P", "project picker")),
		toggleArchived:       key.NewBinding(key.WithKeys("t"), key.WithHelp("t", "toggle archived")),
		toggleSelectMode:     key.NewBinding(key.WithKeys("ctrl+y"), key.WithHelp("ctrl+y", "text select mode")),
		focusSubtree:         key.NewBinding(key.WithKeys("f", "enter"), key.WithHelp("f/enter", "focus subtree")),
		clearFocus:           key.NewBinding(key.WithKeys("F", "shift+f"), key.WithHelp("F", "full board")),
		multiSelect:          key.NewBinding(key.WithKeys(" ", "space"), key.WithHelp("space", "toggle select")),
		activityLog:          key.NewBinding(key.WithKeys("g"), key.WithHelp("g", "activity log")),
		undo:                 key.NewBinding(key.WithKeys("ctrl+z"), key.WithHelp("ctrl+z", "undo")),
		redo:                 key.NewBinding(key.WithKeys("ctrl+shift+z"), key.WithHelp("ctrl+shift+z", "redo")),
		diffModeToggle:       key.NewBinding(key.WithKeys("ctrl+d"), key.WithHelp("ctrl+d", "diff mode")),
	}
}

// applyConfig applies user keybinding overrides.
func (k *keyMap) applyConfig(cfg KeyConfig) {
	configureBinding(&k.commandPalette, cfg.CommandPalette, ":", "command palette")
	configureBinding(&k.quickActions, cfg.QuickActions, ".", "quick actions")
	configureBinding(&k.multiSelect, cfg.MultiSelect, " ", "toggle select")
	configureBinding(&k.activityLog, cfg.ActivityLog, "g", "activity log")
	configureBinding(&k.undo, cfg.Undo, "ctrl+z", "undo")
	configureBinding(&k.redo, cfg.Redo, "ctrl+shift+z", "redo")
}

// ShortHelp handles short help.
func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{
		k.addActionItem, k.actionItemInfo, k.editActionItem, k.focusSubtree, k.toggleSelectMode, k.commandPalette, k.undo, k.activityLog, k.quit,
	}
}

// FullHelp handles full help.
func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.addActionItem, k.actionItemInfo, k.editActionItem, k.newProject, k.editProject, k.commandPalette, k.quickActions, k.search, k.projects, k.toggleArchived, k.toggleSelectMode, k.focusSubtree, k.clearFocus, k.toggleHelp, k.reload, k.quit},
		{k.moveLeft, k.moveRight, k.moveUp, k.moveDown, k.moveActionItemLeft, k.moveActionItemRight},
		{k.deleteActionItem, k.hardDeleteActionItem, k.restoreActionItem, k.multiSelect, k.undo, k.redo, k.activityLog},
	}
}

// configureBinding applies one keybinding override with fallback handling.
func configureBinding(binding *key.Binding, raw, fallback, desc string) {
	keys, helpKey := parseBindingKeys(raw, fallback)
	binding.SetKeys(keys...)
	binding.SetHelp(helpKey, desc)
}

// parseBindingKeys normalizes configured key text into key-matcher inputs and help text.
func parseBindingKeys(raw, fallback string) ([]string, string) {
	value := strings.TrimSpace(raw)
	if value == "" {
		value = fallback
	}
	if strings.EqualFold(value, "space") || value == " " {
		return []string{" ", "space"}, "space"
	}

	rs := []rune(value)
	if len(rs) == 1 {
		// Bubble Tea key strings use shifted rune values (e.g. "Z"), so include
		// explicit shift aliases for uppercase rune bindings.
		if unicode.IsUpper(rs[0]) {
			lower := strings.ToLower(value)
			return []string{value, "shift+" + lower}, value
		}
		return []string{value}, value
	}

	if utf8.RuneCountInString(value) > 1 {
		return []string{strings.ToLower(value)}, value
	}
	return []string{value}, value
}
