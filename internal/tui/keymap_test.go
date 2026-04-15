package tui

import (
	"testing"

	"charm.land/bubbles/v2/key"
)

// TestParseBindingKeys verifies key parsing behavior for configured overrides.
func TestParseBindingKeys(t *testing.T) {
	t.Run("space aliases", func(t *testing.T) {
		keys, help := parseBindingKeys("space", ".")
		if len(keys) != 2 || keys[0] != " " || keys[1] != "space" {
			t.Fatalf("unexpected parsed space keys %#v", keys)
		}
		if help != "space" {
			t.Fatalf("unexpected space help text %q", help)
		}
	})

	t.Run("uppercase rune includes shift alias", func(t *testing.T) {
		keys, help := parseBindingKeys("Z", "z")
		if len(keys) != 2 || keys[0] != "Z" || keys[1] != "shift+z" {
			t.Fatalf("unexpected uppercase parsed keys %#v", keys)
		}
		if help != "Z" {
			t.Fatalf("unexpected uppercase help text %q", help)
		}
	})

	t.Run("multi rune lowercases key matcher", func(t *testing.T) {
		keys, help := parseBindingKeys("Ctrl+R", "r")
		if len(keys) != 1 || keys[0] != "ctrl+r" {
			t.Fatalf("unexpected multi-rune parsed keys %#v", keys)
		}
		if help != "Ctrl+R" {
			t.Fatalf("unexpected multi-rune help text %q", help)
		}
	})

	t.Run("blank uses fallback", func(t *testing.T) {
		keys, help := parseBindingKeys("", "x")
		if len(keys) != 1 || keys[0] != "x" {
			t.Fatalf("unexpected fallback parsed keys %#v", keys)
		}
		if help != "x" {
			t.Fatalf("unexpected fallback help text %q", help)
		}
	})
}

// TestConfigureBinding verifies binding override application behavior.
func TestConfigureBinding(t *testing.T) {
	b := key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "old"))
	configureBinding(&b, "v", "a", "activity log")
	keys := b.Keys()
	if len(keys) != 1 || keys[0] != "v" {
		t.Fatalf("unexpected configured keys %#v", keys)
	}
	if b.Help().Key != "v" || b.Help().Desc != "activity log" {
		t.Fatalf("unexpected configured help %#v", b.Help())
	}
}

// TestKeyMapApplyConfig verifies dynamic key map override behavior.
func TestKeyMapApplyConfig(t *testing.T) {
	k := newKeyMap()
	k.applyConfig(KeyConfig{
		CommandPalette: ";",
		QuickActions:   ",",
		MultiSelect:    "x",
		ActivityLog:    "v",
		Undo:           "u",
		Redo:           "R",
	})

	assertKeys := func(name string, binding key.Binding, expected ...string) {
		t.Helper()
		got := binding.Keys()
		if len(got) != len(expected) {
			t.Fatalf("%s key count mismatch got=%#v expected=%#v", name, got, expected)
		}
		for i := range expected {
			if got[i] != expected[i] {
				t.Fatalf("%s key mismatch got=%#v expected=%#v", name, got, expected)
			}
		}
	}

	assertKeys("command palette", k.commandPalette, ";")
	assertKeys("quick actions", k.quickActions, ",")
	assertKeys("multi select", k.multiSelect, "x")
	assertKeys("activity log", k.activityLog, "v")
	assertKeys("undo", k.undo, "u")
	assertKeys("redo", k.redo, "R", "shift+r")
}

// TestKeyMapDefaultsIncludeProjectionKeys verifies subtree projection key defaults.
func TestKeyMapDefaultsIncludeProjectionKeys(t *testing.T) {
	k := newKeyMap()
	if got := k.focusSubtree.Keys(); len(got) != 2 || got[0] != "f" || got[1] != "enter" {
		t.Fatalf("unexpected focus subtree keys %#v", got)
	}
	gotClear := k.clearFocus.Keys()
	if len(gotClear) != 2 || gotClear[0] != "F" || gotClear[1] != "shift+f" {
		t.Fatalf("unexpected clear focus keys %#v", gotClear)
	}
}
