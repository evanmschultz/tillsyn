// MIGRATION TARGET: github.com/hylla-org/lykta
package keybindings

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	tea "charm.land/bubbletea/v2"
)

// ---------------------------------------------------------------------------
// Loader tests
// ---------------------------------------------------------------------------

func TestLoadBindings_BaselineOnly(t *testing.T) {
	b, err := LoadBindings(DefaultBaselineJSON(), "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got, want := len(b.Commands), 4; got != want {
		t.Errorf("command count: got %d, want %d", got, want)
	}
}

func TestLoadBindings_WithLocal(t *testing.T) {
	// Canonical local file shape per REVISION_BRIEF §2.19 — 5 command-mode-only entries.
	localJSON := `{
		"schema_version": 1,
		"name": "tillsyn-bindings",
		"description": "Local Tillsyn overrides.",
		"extends": "stil-baseline",
		"product_extensions": {
			"tillsyn": {
				"commands": [
					{"id":"dispatch",  "command":"dispatch",  "description":"Open dispatcher."},
					{"id":"plan",      "command":"plan",      "description":"Open plan view."},
					{"id":"archive",   "command":"archive",   "description":"Archive focused item."},
					{"id":"settings",  "command":"settings",  "description":"Open settings."},
					{"id":"help",      "command":"help",      "description":"Show help."}
				]
			}
		}
	}`

	dir := t.TempDir()
	localPath := filepath.Join(dir, "bindings.json")
	if err := os.WriteFile(localPath, []byte(localJSON), 0o600); err != nil {
		t.Fatalf("write local fixture: %v", err)
	}

	b, err := LoadBindings(DefaultBaselineJSON(), localPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got, want := len(b.Commands), 9; got != want {
		t.Errorf("command count after merge: got %d, want %d", got, want)
	}
}

func TestLoadBindings_LocalWins(t *testing.T) {
	// Override the "handoff" baseline command with a different description.
	localJSON := `{
		"product_extensions": {
			"tillsyn": {
				"commands": [
					{"id":"handoff", "command":"handoff", "description":"OVERRIDDEN handoff description."}
				]
			}
		}
	}`

	dir := t.TempDir()
	localPath := filepath.Join(dir, "bindings.json")
	if err := os.WriteFile(localPath, []byte(localJSON), 0o600); err != nil {
		t.Fatalf("write local fixture: %v", err)
	}

	b, err := LoadBindings(DefaultBaselineJSON(), localPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Still 4 commands (no new IDs added).
	if got, want := len(b.Commands), 4; got != want {
		t.Errorf("command count: got %d, want %d", got, want)
	}
	// Local description must win for "handoff".
	var found bool
	for _, c := range b.Commands {
		if c.ID == "handoff" {
			found = true
			if c.Description != "OVERRIDDEN handoff description." {
				t.Errorf("handoff description: got %q, want OVERRIDDEN", c.Description)
			}
		}
	}
	if !found {
		t.Error("handoff command not found in merged results")
	}
}

func TestLoadBindings_MissingLocalFile(t *testing.T) {
	nonExistent := filepath.Join(t.TempDir(), "does-not-exist.json")
	b, err := LoadBindings(DefaultBaselineJSON(), nonExistent)
	if err != nil {
		t.Fatalf("missing local file must not return error, got: %v", err)
	}
	if got, want := len(b.Commands), 4; got != want {
		t.Errorf("command count (baseline-only fallback): got %d, want %d", got, want)
	}
}

func TestLoadBindings_MalformedLocalFile(t *testing.T) {
	dir := t.TempDir()
	localPath := filepath.Join(dir, "bindings.json")
	if err := os.WriteFile(localPath, []byte(`{not valid json`), 0o600); err != nil {
		t.Fatalf("write malformed fixture: %v", err)
	}
	_, err := LoadBindings(DefaultBaselineJSON(), localPath)
	if err == nil {
		t.Error("malformed local file must return an error")
	}
}

// ---------------------------------------------------------------------------
// Dispatcher tests
// ---------------------------------------------------------------------------

func TestDispatcher_Dispatch(t *testing.T) {
	b, err := LoadBindings(DefaultBaselineJSON(), "")
	if err != nil {
		t.Fatalf("load bindings: %v", err)
	}
	d := NewDispatcher(b)

	// handoff and comment are command-mode only (no Keys field); Dispatch for any
	// single key in ModeNav returns NoOp before a real handler is registered.
	t.Run("unregistered_nav_key_returns_NoOp", func(t *testing.T) {
		msg := tea.KeyPressMsg{Code: 'z', Text: "z"}
		got := d.Dispatch(msg, ModeNav)
		if got == nil {
			t.Fatal("Dispatch must not return nil HandlerFunc — want NoOp")
		}
		cmd := got()
		if cmd != nil {
			t.Errorf("NoOp() must return nil tea.Cmd, got %T", cmd)
		}
	})

	// Register a real handler then Dispatch it.
	t.Run("registered_single_key_nav_binding_returned", func(t *testing.T) {
		called := false
		d.Register(ModeNav, "j", func() tea.Cmd {
			called = true
			return nil
		})
		msg := tea.KeyPressMsg{Code: 'j', Text: "j"}
		h := d.Dispatch(msg, ModeNav)
		if h == nil {
			t.Fatal("Dispatch returned nil HandlerFunc")
		}
		_ = h()
		if !called {
			t.Error("registered handler was not invoked")
		}
	})
}

// TestDispatcher_MultiKey_Returns_NoOp documents that multi-key sequence commands
// (new-drop ["Space","n"], complete-drop ["Space","c"]) are intentionally not registered
// in nav-mode bindings for single-call Dispatch. KEYBIND-R4 tracks the leader-key state
// machine that will enable these.
func TestDispatcher_MultiKey_Returns_NoOp(t *testing.T) {
	b, err := LoadBindings(DefaultBaselineJSON(), "")
	if err != nil {
		t.Fatalf("load bindings: %v", err)
	}
	d := NewDispatcher(b)

	cases := []struct {
		name string
		code rune
		text string
	}{
		{name: "Space_alone_returns_NoOp", code: ' ', text: " "},
		{name: "n_alone_returns_NoOp_for_new_drop", code: 'n', text: "n"},
		{name: "c_alone_returns_NoOp_for_complete_drop", code: 'c', text: "c"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			msg := tea.KeyPressMsg{Code: tc.code, Text: tc.text}
			h := d.Dispatch(msg, ModeNav)
			if h == nil {
				t.Fatal("Dispatch must not return nil HandlerFunc — want NoOp")
			}
			cmd := h()
			if cmd != nil {
				t.Errorf("expected NoOp() to return nil tea.Cmd, got %T", cmd)
			}
		})
	}
}

func TestDispatcher_Register(t *testing.T) {
	d := NewDispatcher(Bindings{})

	called := false
	d.Register(ModeCommand, ":", func() tea.Cmd {
		called = true
		return nil
	})

	msg := tea.KeyPressMsg{Code: ':', Text: ":"}
	h := d.Dispatch(msg, ModeCommand)
	if h == nil {
		t.Fatal("Dispatch returned nil HandlerFunc after Register")
	}
	_ = h()
	if !called {
		t.Error("registered handler was not called")
	}
}

func TestDispatcher_DispatchCommand(t *testing.T) {
	b, err := LoadBindings(DefaultBaselineJSON(), "")
	if err != nil {
		t.Fatalf("load bindings: %v", err)
	}
	d := NewDispatcher(b)

	// "handoff" has CommandName="handoff" and is auto-registered by NewDispatcher.
	// Override with a handler that records the call.
	called := false
	d.RegisterCommand("handoff", func() tea.Cmd {
		called = true
		return nil
	})

	h := d.DispatchCommand("handoff")
	if h == nil {
		t.Fatal("DispatchCommand returned nil HandlerFunc")
	}
	_ = h()
	if !called {
		t.Error("registered command handler was not called")
	}

	// Unknown command ID returns NoOp (not nil).
	h2 := d.DispatchCommand("nonexistent-id")
	if h2 == nil {
		t.Fatal("DispatchCommand must return NoOp for unknown ID, not nil")
	}
	if cmd := h2(); cmd != nil {
		t.Errorf("NoOp() for unknown ID must return nil cmd, got %T", cmd)
	}
}

// ---------------------------------------------------------------------------
// Mode tests — coverage for modes.go
// ---------------------------------------------------------------------------

func TestMode_String(t *testing.T) {
	cases := []struct {
		mode Mode
		want string
	}{
		{ModeNav, "nav"},
		{ModeInsert, "insert"},
		{ModeVisual, "visual"},
		{ModeVisualBlock, "visual-block"},
		{ModeCommand, "command"},
		{ModeHint, "hint"},
		{Mode(99), "unknown"},
	}
	for _, tc := range cases {
		if got := tc.mode.String(); got != tc.want {
			t.Errorf("Mode(%d).String() = %q, want %q", int(tc.mode), got, tc.want)
		}
	}
}

func TestNoOp_ReturnsNilCmd(t *testing.T) {
	if NoOp == nil {
		t.Fatal("NoOp must not be nil")
	}
	if cmd := NoOp(); cmd != nil {
		t.Errorf("NoOp() must return nil tea.Cmd, got %T", cmd)
	}
}

// ---------------------------------------------------------------------------
// Loader structural tests — ensure embedded JSON parses correctly
// ---------------------------------------------------------------------------

func TestDefaultBaselineJSON_ValidJSON(t *testing.T) {
	data := DefaultBaselineJSON()
	if len(data) == 0 {
		t.Fatal("DefaultBaselineJSON returned empty bytes")
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("DefaultBaselineJSON is not valid JSON: %v", err)
	}
	if _, ok := raw["product_extensions"]; !ok {
		t.Error("DefaultBaselineJSON missing product_extensions key")
	}
}
