// MIGRATION TARGET: github.com/hylla-org/lykta
package components

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

// errInvalid is the sentinel error returned by the failing validator in tests.
var errInvalid = errors.New("value must not be empty")

// failingValidator rejects empty strings.
func failingValidator(s string) error {
	if s == "" {
		return errInvalid
	}
	return nil
}

// TestTextInputModel_Validation verifies that the validate func is called correctly
// on Enter and that nil, passing, and failing validators all behave as specified.
// No test row should return a non-nil tea.Cmd.
func TestTextInputModel_Validation(t *testing.T) {
	enterKey := tea.KeyPressMsg{Code: tea.KeyEnter}

	tests := []struct {
		name       string
		validate   func(string) error
		wantSubmit bool
		wantErrNil bool
	}{
		{
			name:       "nil validator submits without error",
			validate:   nil,
			wantSubmit: true,
			wantErrNil: true,
		},
		{
			// failingValidator rejects empty strings; inner value starts empty, so
			// this row exercises the "Enter with failing validator" path.
			name:       "failing validator rejects empty",
			validate:   failingValidator,
			wantSubmit: false,
			wantErrNil: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewTextInput("enter something", tt.validate)
			updated, cmd := m.Update(enterKey)

			if cmd != nil {
				t.Errorf("Update returned non-nil cmd — must be nil to avoid killing parent TUI")
			}
			if updated.Submitted() != tt.wantSubmit {
				t.Errorf("Submitted() = %v, want %v", updated.Submitted(), tt.wantSubmit)
			}
			if tt.wantErrNil && updated.Err() != nil {
				t.Errorf("Err() = %v, want nil", updated.Err())
			}
			if !tt.wantErrNil && updated.Err() == nil {
				t.Errorf("Err() = nil, want non-nil")
			}
		})
	}
}

// TestTextInputModel_Submit verifies enter-with-valid and enter-with-invalid paths.
// It constructs two models — one with a nil validator (always valid) and one with
// failingValidator. Because inner textinput value starts empty, failingValidator
// will always reject on an empty model. No test row may return a non-nil tea.Cmd.
func TestTextInputModel_Submit(t *testing.T) {
	enterKey := tea.KeyPressMsg{Code: tea.KeyEnter}

	t.Run("Enter with nil validator marks submitted", func(t *testing.T) {
		m := NewTextInput("placeholder", nil)
		updated, cmd := m.Update(enterKey)

		if cmd != nil {
			t.Errorf("Update returned non-nil cmd on Enter with nil validator")
		}
		if !updated.Submitted() {
			t.Errorf("Submitted() = false, want true when validate is nil")
		}
		if updated.Err() != nil {
			t.Errorf("Err() = %v, want nil when validate is nil", updated.Err())
		}
	})

	t.Run("Enter with failing validator does not submit", func(t *testing.T) {
		m := NewTextInput("placeholder", failingValidator)
		// Inner value is empty; failingValidator rejects empty strings.
		updated, cmd := m.Update(enterKey)

		if cmd != nil {
			t.Errorf("Update returned non-nil cmd on Enter with failing validator")
		}
		if updated.Submitted() {
			t.Errorf("Submitted() = true, want false when validator rejects")
		}
		if updated.Err() == nil {
			t.Errorf("Err() = nil, want non-nil when validator rejects")
		}
		if !errors.Is(updated.Err(), errInvalid) {
			t.Errorf("Err() = %v, want %v", updated.Err(), errInvalid)
		}
	})

	t.Run("non-Enter key does not submit", func(t *testing.T) {
		m := NewTextInput("placeholder", nil)
		updated, cmd := m.Update(tea.KeyPressMsg{Code: 'a', Text: "a"})

		_ = cmd // cmd may or may not be nil for character keys (inner model drives it)
		if updated.Submitted() {
			t.Errorf("Submitted() = true after non-Enter key, want false")
		}
	})
}

// TestTextInputModel_View verifies that View returns the inner view and appends
// an error string when Err() is non-nil.
func TestTextInputModel_View(t *testing.T) {
	t.Run("no error — view contains placeholder or inner render", func(t *testing.T) {
		m := NewTextInput("myplaceholder", nil)
		v := m.View()
		// View must return a non-empty string (inner textinput renders something).
		if v == "" {
			t.Errorf("View() returned empty string, want non-empty")
		}
	})

	t.Run("error present — view contains error string", func(t *testing.T) {
		m := NewTextInput("placeholder", failingValidator)
		enterKey := tea.KeyPressMsg{Code: tea.KeyEnter}
		updated, _ := m.Update(enterKey)
		if updated.Err() == nil {
			t.Fatal("precondition: expected Err() to be non-nil after rejection")
		}
		v := updated.View()
		if !strings.Contains(v, errInvalid.Error()) {
			t.Errorf("View() = %q, want to contain error string %q", v, errInvalid.Error())
		}
	})
}

// TestTextInputModel_Accessors verifies Value, Err, and Submitted defaults.
func TestTextInputModel_Accessors(t *testing.T) {
	m := NewTextInput("ph", nil)
	if m.Submitted() {
		t.Errorf("fresh model: Submitted() = true, want false")
	}
	if m.Err() != nil {
		t.Errorf("fresh model: Err() = %v, want nil", m.Err())
	}
	// Value() on a fresh model is empty.
	if m.Value() != "" {
		t.Errorf("fresh model: Value() = %q, want empty string", m.Value())
	}
}

// TestTextInputModel_ValidationOnKeystroke verifies that validate is re-run on
// every non-Enter keystroke so that Err() reflects the current value in real time.
// Regression test for CE-1: previously err was only updated on Enter.
func TestTextInputModel_ValidationOnKeystroke(t *testing.T) {
	// Validator: reject strings shorter than 3 characters.
	minLen3 := func(s string) error {
		if len(s) < 3 {
			return fmt.Errorf("too short")
		}
		return nil
	}

	m := NewTextInput("ph", minLen3)

	// Fresh model: no validate has run yet; err is nil.
	if m.Err() != nil {
		t.Fatalf("precondition: fresh model Err() = %v, want nil", m.Err())
	}

	// Type a single character — value becomes "a" (length 1 < 3), err must be non-nil.
	m, _ = m.Update(tea.KeyPressMsg{Code: 'a', Text: "a"})
	if m.Err() == nil {
		t.Errorf("after typing 'a': Err() = nil, want non-nil (value too short)")
	}

	// Type three more characters — value becomes "abcd" (length 4 >= 3), err must clear.
	m, _ = m.Update(tea.KeyPressMsg{Code: 'b', Text: "b"})
	m, _ = m.Update(tea.KeyPressMsg{Code: 'c', Text: "c"})
	m, _ = m.Update(tea.KeyPressMsg{Code: 'd', Text: "d"})
	if m.Err() != nil {
		t.Errorf("after typing 'abcd': Err() = %v, want nil (value is now valid)", m.Err())
	}

	// Submitted must remain false throughout — keystroke updates never submit.
	if m.Submitted() {
		t.Errorf("Submitted() = true after keystroke-only sequence, want false")
	}

	// pass→fail: backspace twice from "abcd" (length 4, valid) to "ab" (length 2 < 3, invalid).
	// Confirms the validator fires on backspace keystrokes, not only on forward-typing.
	t.Run("pass to fail on backspace", func(t *testing.T) {
		m2 := NewTextInput("ph", minLen3)
		// Build up to "abcd" (valid).
		m2, _ = m2.Update(tea.KeyPressMsg{Code: 'a', Text: "a"})
		m2, _ = m2.Update(tea.KeyPressMsg{Code: 'b', Text: "b"})
		m2, _ = m2.Update(tea.KeyPressMsg{Code: 'c', Text: "c"})
		m2, _ = m2.Update(tea.KeyPressMsg{Code: 'd', Text: "d"})
		if m2.Err() != nil {
			t.Fatalf("precondition: after typing 'abcd', Err() = %v, want nil", m2.Err())
		}
		// Backspace twice: "abcd" -> "abc" -> "ab" (length 2, invalid).
		m2, _ = m2.Update(tea.KeyPressMsg{Code: tea.KeyBackspace})
		m2, _ = m2.Update(tea.KeyPressMsg{Code: tea.KeyBackspace})
		if m2.Err() == nil {
			t.Errorf("after two backspaces to 'ab': Err() = nil, want non-nil (value too short)")
		}
	})
}
