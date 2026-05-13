// MIGRATION TARGET: github.com/hylla-org/lykta
package keybindings

import tea "charm.land/bubbletea/v2"

// Dispatcher routes tea.KeyMsg events to registered handlers per mode.
// It is stateless with respect to mode — the caller is responsible for tracking the
// current mode and passing it to Dispatch on each event.
type Dispatcher struct {
	bindings map[Mode]map[string]HandlerFunc // mode → key_string → handler
	commands map[string]HandlerFunc          // command-id → handler
}

// NewDispatcher builds a Dispatcher from the provided Bindings.
//
// Commands with a single-element Keys array are registered in bindings[ModeNav].
// Commands with multi-key Keys arrays (len > 1) are intentionally skipped in nav-mode
// registration; they remain accessible via DispatchCommand by ID for command-mode routing.
//
// TODO(KEYBIND-R4): implement leader-key state machine for multi-key nav bindings
// (e.g., ["Space","n"] for new-drop, ["Space","c"] for complete-drop).
//
// Commands with a non-empty CommandName are registered in the commands map by ID so they
// can be retrieved via DispatchCommand.
func NewDispatcher(b Bindings) *Dispatcher {
	d := &Dispatcher{
		bindings: make(map[Mode]map[string]HandlerFunc),
		commands: make(map[string]HandlerFunc),
	}

	for _, cmd := range b.Commands {
		if len(cmd.Keys) == 1 {
			if d.bindings[ModeNav] == nil {
				d.bindings[ModeNav] = make(map[string]HandlerFunc)
			}
			// Capture cmd for the closure.
			c := cmd
			d.bindings[ModeNav][c.Keys[0]] = func() tea.Cmd { return nil }
			// Register by ID in commands as well for command-mode lookup.
			d.commands[c.ID] = func() tea.Cmd { return nil }
		} else if len(cmd.Keys) > 1 {
			// Multi-key nav binding — skip ModeNav registration until KEYBIND-R4.
			// The command is still registered by ID for DispatchCommand.
			c := cmd
			d.commands[c.ID] = func() tea.Cmd { return nil }
		}

		if cmd.CommandName != "" {
			c := cmd
			d.commands[c.ID] = func() tea.Cmd { return nil }
		}
	}

	return d
}

// Register adds or replaces the handler for a given mode and key string. This allows the
// caller to attach real handler logic after construction, or to override defaults.
func (d *Dispatcher) Register(mode Mode, key string, h HandlerFunc) {
	if d.bindings[mode] == nil {
		d.bindings[mode] = make(map[string]HandlerFunc)
	}
	d.bindings[mode][key] = h
}

// RegisterCommand adds or replaces the handler for the given command ID. This allows the
// caller to wire actual business logic to a command after construction.
func (d *Dispatcher) RegisterCommand(id string, h HandlerFunc) {
	d.commands[id] = h
}

// Dispatch looks up the handler registered for the key represented by msg in the given mode.
// It performs an internal type-switch to extract a string key from the concrete
// tea.KeyPressMsg type; release events and other non-press messages return NoOp.
// Multi-key commands are not registered in nav-mode bindings and therefore return NoOp from
// Dispatch (KEYBIND-R4 pending).
func (d *Dispatcher) Dispatch(msg tea.KeyMsg, mode Mode) HandlerFunc {
	var keyStr string
	switch m := msg.(type) {
	case tea.KeyPressMsg:
		keyStr = m.String()
	default:
		return NoOp
	}

	if modeMap, ok := d.bindings[mode]; ok {
		if h, ok := modeMap[keyStr]; ok {
			return h
		}
	}
	return NoOp
}

// DispatchCommand looks up the handler registered for the given command ID. This supports
// command-mode routing and multi-key commands that cannot be resolved by single-call Dispatch.
// Returns NoOp if the command ID is not registered.
func (d *Dispatcher) DispatchCommand(id string) HandlerFunc {
	if h, ok := d.commands[id]; ok {
		return h
	}
	return NoOp
}
