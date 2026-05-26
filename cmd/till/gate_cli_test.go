package main

import (
	"bytes"
	"context"
	"testing"
)

// TestGateCommand verifies that `till gate` is a registered cobra subcommand
// that reads stdin, handles JSON parsing, and exits 0 (fail-open).
func TestGateCommand(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantErr   bool
		wantExit0 bool
	}{
		{
			name:      "empty stdin defers (exit 0)",
			input:     "",
			wantErr:   false,
			wantExit0: true,
		},
		{
			name:      "malformed JSON defers (exit 0)",
			input:     `{invalid json`,
			wantErr:   false,
			wantExit0: true,
		},
		{
			name:      "valid empty object parses (exit 0)",
			input:     `{}`,
			wantErr:   false,
			wantExit0: true,
		},
		{
			name:      "valid event with fields (exit 0)",
			input:     `{"tool_name":"Edit","tool_input":{"file_path":"/abs/path"}}`,
			wantErr:   false,
			wantExit0: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := newGateCommand()
			cmd.SetArgs([]string{})

			stdin := bytes.NewBufferString(tt.input)
			stdout := new(bytes.Buffer)
			stderr := new(bytes.Buffer)

			cmd.SetIn(stdin)
			cmd.SetOut(stdout)
			cmd.SetErr(stderr)

			err := cmd.ExecuteContext(context.Background())

			if tt.wantErr && err == nil {
				t.Errorf("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("expected nil, got error: %v", err)
			}
		})
	}
}

// TestGateCommandRegistration verifies that the command can be created and
// has the correct Use field for cobra registration.
func TestGateCommandRegistration(t *testing.T) {
	cmd := newGateCommand()
	if cmd.Use != "gate" {
		t.Errorf("expected Use='gate', got %q", cmd.Use)
	}
	if cmd.Short == "" {
		t.Errorf("expected non-empty Short")
	}
	if cmd.Long == "" {
		t.Errorf("expected non-empty Long")
	}
	if cmd.RunE == nil {
		t.Errorf("expected non-nil RunE")
	}
}
