// MIGRATION TARGET: github.com/hylla-org/lykta
package components

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

// TestConfirmModel_Update verifies the full key-handling matrix for ConfirmModel.
// Each row sends one KeyPressMsg directly to model.Update and asserts the resulting
// accessor state. No test row should return a non-nil tea.Cmd.
func TestConfirmModel_Update(t *testing.T) {
	tests := []struct {
		name          string
		defaultYes    bool
		key           tea.KeyPressMsg
		wantConfirmed bool
		wantCancelled bool
		wantDone      bool
	}{
		{
			name:          "y confirms",
			defaultYes:    false,
			key:           tea.KeyPressMsg{Code: 'y', Text: "y"},
			wantConfirmed: true,
			wantCancelled: false,
			wantDone:      true,
		},
		{
			name:          "Y confirms",
			defaultYes:    false,
			key:           tea.KeyPressMsg{Code: 'Y', Text: "Y"},
			wantConfirmed: true,
			wantCancelled: false,
			wantDone:      true,
		},
		{
			name:          "n cancels",
			defaultYes:    true,
			key:           tea.KeyPressMsg{Code: 'n', Text: "n"},
			wantConfirmed: false,
			wantCancelled: true,
			wantDone:      true,
		},
		{
			name:          "N cancels",
			defaultYes:    true,
			key:           tea.KeyPressMsg{Code: 'N', Text: "N"},
			wantConfirmed: false,
			wantCancelled: true,
			wantDone:      true,
		},
		{
			name:          "Enter with defaultYes=true confirms",
			defaultYes:    true,
			key:           tea.KeyPressMsg{Code: tea.KeyEnter},
			wantConfirmed: true,
			wantCancelled: false,
			wantDone:      true,
		},
		{
			name:          "Enter with defaultYes=false cancels",
			defaultYes:    false,
			key:           tea.KeyPressMsg{Code: tea.KeyEnter},
			wantConfirmed: false,
			wantCancelled: true,
			wantDone:      true,
		},
		{
			name:          "Escape cancels",
			defaultYes:    true,
			key:           tea.KeyPressMsg{Code: tea.KeyEsc},
			wantConfirmed: false,
			wantCancelled: true,
			wantDone:      true,
		},
		{
			name:          "unhandled key is ignored",
			defaultYes:    false,
			key:           tea.KeyPressMsg{Code: 'x', Text: "x"},
			wantConfirmed: false,
			wantCancelled: false,
			wantDone:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewConfirm("delete this item?", tt.defaultYes)
			updated, cmd := m.Update(tt.key)

			if cmd != nil {
				t.Errorf("Update returned non-nil cmd — must be nil to avoid killing parent TUI")
			}
			if updated.Confirmed() != tt.wantConfirmed {
				t.Errorf("Confirmed() = %v, want %v", updated.Confirmed(), tt.wantConfirmed)
			}
			if updated.Cancelled() != tt.wantCancelled {
				t.Errorf("Cancelled() = %v, want %v", updated.Cancelled(), tt.wantCancelled)
			}
			if updated.Done() != tt.wantDone {
				t.Errorf("Done() = %v, want %v", updated.Done(), tt.wantDone)
			}
		})
	}
}

// TestConfirmModel_Update_NonKeyMsg verifies that non-key messages are passed through
// without state change and with a nil cmd.
func TestConfirmModel_Update_NonKeyMsg(t *testing.T) {
	m := NewConfirm("confirm?", false)
	updated, cmd := m.Update("not-a-key-msg")
	if cmd != nil {
		t.Errorf("Update returned non-nil cmd for non-key message")
	}
	if updated.Done() {
		t.Errorf("Done() = true after non-key message, want false")
	}
}

// TestConfirmModel_View verifies that View renders the prompt and a y/N or Y/n indicator.
func TestConfirmModel_View(t *testing.T) {
	t.Run("defaultYes=false shows y/N", func(t *testing.T) {
		m := NewConfirm("delete?", false)
		v := m.View()
		if !strings.Contains(v, "delete?") {
			t.Errorf("View() missing prompt: %q", v)
		}
		if !strings.Contains(v, "y/N") {
			t.Errorf("View() missing [y/N] indicator: %q", v)
		}
	})

	t.Run("defaultYes=true shows Y/n", func(t *testing.T) {
		m := NewConfirm("continue?", true)
		v := m.View()
		if !strings.Contains(v, "Y/n") {
			t.Errorf("View() missing [Y/n] indicator: %q", v)
		}
	})
}

// TestProgress_View verifies that Progress.View() renders the message and
// that WithMessage returns a copy with the updated message.
func TestProgress_View(t *testing.T) {
	t.Run("renders message", func(t *testing.T) {
		p := NewProgress("loading...")
		v := p.View()
		if !strings.Contains(v, "loading...") {
			t.Errorf("Progress.View() = %q, want to contain %q", v, "loading...")
		}
	})

	t.Run("WithMessage updates message", func(t *testing.T) {
		p := NewProgress("initial")
		p2 := p.WithMessage("updated")
		v := p2.View()
		if !strings.Contains(v, "updated") {
			t.Errorf("Progress.View() after WithMessage = %q, want to contain %q", v, "updated")
		}
		// Original unchanged.
		if !strings.Contains(p.View(), "initial") {
			t.Errorf("original Progress.View() = %q, want to contain %q", p.View(), "initial")
		}
	})

	t.Run("empty message renders without panic", func(t *testing.T) {
		p := NewProgress("")
		_ = p.View() // must not panic
	})
}
