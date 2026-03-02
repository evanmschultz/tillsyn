package tui

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"charm.land/bubbles/v2/key"
)

// keyMap represents key map data used by this package.
type keyMap struct {
	quit             key.Binding
	reload           key.Binding
	toggleHelp       key.Binding
	moveLeft         key.Binding
	moveRight        key.Binding
	moveUp           key.Binding
	moveDown         key.Binding
	addTask          key.Binding
	addSubtask       key.Binding
	taskInfo         key.Binding
	editTask         key.Binding
	newProject       key.Binding
	editProject      key.Binding
	commandPalette   key.Binding
	quickActions     key.Binding
	deleteTask       key.Binding
	archiveTask      key.Binding
	moveTaskLeft     key.Binding
	moveTaskRight    key.Binding
	hardDeleteTask   key.Binding
	restoreTask      key.Binding
	search           key.Binding
	projects         key.Binding
	toggleArchived   key.Binding
	toggleSelectMode key.Binding
	focusSubtree     key.Binding
	clearFocus       key.Binding
	multiSelect      key.Binding
	activityLog      key.Binding
	undo             key.Binding
	redo             key.Binding
}

// newKeyMap constructs key map.
func newKeyMap() keyMap {
	return keyMap{
		quit:             key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
		reload:           key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "reload")),
		toggleHelp:       key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "toggle help")),
		moveLeft:         key.NewBinding(key.WithKeys("h", "left"), key.WithHelp("h/←", "column left")),
		moveRight:        key.NewBinding(key.WithKeys("l", "right"), key.WithHelp("l/→", "column right")),
		moveUp:           key.NewBinding(key.WithKeys("k", "up"), key.WithHelp("k/↑", "move up")),
		moveDown:         key.NewBinding(key.WithKeys("j", "down"), key.WithHelp("j/↓", "move down")),
		addTask:          key.NewBinding(key.WithKeys("n"), key.WithHelp("n", "new task")),
		addSubtask:       key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "new subtask")),
		taskInfo:         key.NewBinding(key.WithKeys("i", "enter"), key.WithHelp("i/enter", "task info")),
		editTask:         key.NewBinding(key.WithKeys("e"), key.WithHelp("e", "edit task")),
		newProject:       key.NewBinding(key.WithKeys("N"), key.WithHelp("N", "new project")),
		editProject:      key.NewBinding(key.WithKeys("M"), key.WithHelp("M", "edit project")),
		commandPalette:   key.NewBinding(key.WithKeys(":"), key.WithHelp(":", "command palette")),
		quickActions:     key.NewBinding(key.WithKeys("."), key.WithHelp(".", "quick actions")),
		deleteTask:       key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "delete (default)")),
		archiveTask:      key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "archive task")),
		moveTaskLeft:     key.NewBinding(key.WithKeys("["), key.WithHelp("[", "move task left")),
		moveTaskRight:    key.NewBinding(key.WithKeys("]"), key.WithHelp("]", "move task right")),
		hardDeleteTask:   key.NewBinding(key.WithKeys("D", "shift+d"), key.WithHelp("D", "hard delete")),
		restoreTask:      key.NewBinding(key.WithKeys("u"), key.WithHelp("u", "restore task")),
		search:           key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "search")),
		projects:         key.NewBinding(key.WithKeys("p", "P"), key.WithHelp("p/P", "project picker")),
		toggleArchived:   key.NewBinding(key.WithKeys("t"), key.WithHelp("t", "toggle archived")),
		toggleSelectMode: key.NewBinding(key.WithKeys("ctrl+y"), key.WithHelp("ctrl+y", "text select mode")),
		focusSubtree:     key.NewBinding(key.WithKeys("f"), key.WithHelp("f", "focus subtree")),
		clearFocus:       key.NewBinding(key.WithKeys("F", "shift+f"), key.WithHelp("F", "full board")),
		multiSelect:      key.NewBinding(key.WithKeys(" ", "space"), key.WithHelp("space", "toggle select")),
		activityLog:      key.NewBinding(key.WithKeys("g"), key.WithHelp("g", "activity log")),
		undo:             key.NewBinding(key.WithKeys("z"), key.WithHelp("z", "undo")),
		redo:             key.NewBinding(key.WithKeys("Z", "shift+z"), key.WithHelp("Z", "redo")),
	}
}

// applyConfig applies user keybinding overrides.
func (k *keyMap) applyConfig(cfg KeyConfig) {
	configureBinding(&k.commandPalette, cfg.CommandPalette, ":", "command palette")
	configureBinding(&k.quickActions, cfg.QuickActions, ".", "quick actions")
	configureBinding(&k.multiSelect, cfg.MultiSelect, " ", "toggle select")
	configureBinding(&k.activityLog, cfg.ActivityLog, "g", "activity log")
	configureBinding(&k.undo, cfg.Undo, "z", "undo")
	configureBinding(&k.redo, cfg.Redo, "Z", "redo")
}

// ShortHelp handles short help.
func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{
		k.addTask, k.addSubtask, k.taskInfo, k.editTask, k.focusSubtree, k.toggleSelectMode, k.commandPalette, k.undo, k.activityLog, k.quit,
	}
}

// FullHelp handles full help.
func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.addTask, k.addSubtask, k.taskInfo, k.editTask, k.newProject, k.editProject, k.commandPalette, k.quickActions, k.search, k.projects, k.toggleArchived, k.toggleSelectMode, k.focusSubtree, k.clearFocus, k.toggleHelp, k.reload, k.quit},
		{k.moveLeft, k.moveRight, k.moveUp, k.moveDown, k.moveTaskLeft, k.moveTaskRight},
		{k.deleteTask, k.hardDeleteTask, k.restoreTask, k.multiSelect, k.undo, k.redo, k.activityLog},
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
