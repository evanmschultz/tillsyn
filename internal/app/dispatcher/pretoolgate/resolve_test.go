package pretoolgate

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// TestResolveFromEnv verifies that TILL_GATE_ALLOWLIST env var is parsed correctly.
func TestResolveFromEnv(t *testing.T) {
	tests := []struct {
		name       string
		envValue   string
		expectSpec bool
		expectEdit *[]string
		expectDeny *[]string
		expectNil  bool
	}{
		{
			name:       "env unset",
			envValue:   "",
			expectSpec: false,
			expectNil:  true,
		},
		{
			name:       "env empty string",
			envValue:   "   ",
			expectSpec: false,
			expectNil:  true,
		},
		{
			name:       "env invalid JSON",
			envValue:   `{invalid`,
			expectSpec: false,
			expectNil:  true,
		},
		{
			name:       "env valid JSON with edit array",
			envValue:   `{"edit": ["/abs/fileA", "/abs/fileB"], "bash_deny": ["git commit"]}`,
			expectSpec: true,
			expectEdit: ptrSlice([]string{"/abs/fileA", "/abs/fileB"}),
			expectDeny: ptrSlice([]string{"git commit"}),
		},
		{
			name:       "env valid JSON with empty edit array (present-empty, not nil)",
			envValue:   `{"edit": [], "bash_deny": []}`,
			expectSpec: true,
			expectEdit: ptrSlice([]string{}), // non-nil empty slice
			expectDeny: ptrSlice([]string{}),
		},
		{
			name:       "env valid JSON missing edit key (absent, should be nil)",
			envValue:   `{"bash_deny": ["git commit"]}`,
			expectSpec: true,
			expectEdit: nil, // absent key → nil
			expectDeny: ptrSlice([]string{"git commit"}),
		},
		{
			name:       "env valid JSON missing bash_deny key (absent, should be nil)",
			envValue:   `{"edit": ["/abs/fileA"]}`,
			expectSpec: true,
			expectEdit: ptrSlice([]string{"/abs/fileA"}),
			expectDeny: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear env first.
			os.Unsetenv("TILL_GATE_ALLOWLIST")

			// Set env if non-empty.
			if tt.envValue != "" {
				os.Setenv("TILL_GATE_ALLOWLIST", tt.envValue)
				defer os.Unsetenv("TILL_GATE_ALLOWLIST")
			}

			spec := resolveFromEnv()

			if tt.expectNil && spec != nil {
				t.Errorf("expected nil spec, got %v", spec)
			}
			if !tt.expectNil && spec == nil {
				t.Errorf("expected non-nil spec, got nil")
			}

			if tt.expectSpec && spec != nil {
				// Check Edit field (nil vs empty).
				if !slicesEqual(spec.Edit, tt.expectEdit) {
					t.Errorf("Edit mismatch: got %v, want %v", spec.Edit, tt.expectEdit)
				}
				// Check BashDeny field (nil vs empty).
				if !slicesEqual(spec.BashDeny, tt.expectDeny) {
					t.Errorf("BashDeny mismatch: got %v, want %v", spec.BashDeny, tt.expectDeny)
				}
			}
		})
	}
}

// TestResolveFromTranscript verifies transcript scanning with last-match-wins.
func TestResolveFromTranscript(t *testing.T) {
	// Create a temporary transcript file with two dispatches of the same agent_type.
	tmpFile := filepath.Join(t.TempDir(), "transcript.jsonl")

	// Event 1: First dispatch of "ta-go-builder" with allowlist.
	event1 := map[string]interface{}{
		"message": map[string]interface{}{
			"content": []interface{}{
				map[string]interface{}{
					"type": "tool_use",
					"name": "Agent",
					"input": map[string]interface{}{
						"subagent_type": "ta-go-builder",
						"prompt":        "<TA_ALLOWLIST>\n{\"edit\": [\"/file1\"], \"bash_deny\": []}\n</TA_ALLOWLIST>",
					},
				},
			},
		},
	}

	// Event 2: Second dispatch of "ta-go-builder" with different allowlist (should win).
	event2 := map[string]interface{}{
		"message": map[string]interface{}{
			"content": []interface{}{
				map[string]interface{}{
					"type": "tool_use",
					"name": "Agent",
					"input": map[string]interface{}{
						"subagent_type": "ta-go-builder",
						"prompt":        "<TA_ALLOWLIST>\n{\"edit\": [\"/file2\", \"/file3\"], \"bash_deny\": [\"git commit\"]}\n</TA_ALLOWLIST>",
					},
				},
			},
		},
	}

	// Write both events to the transcript file.
	data1, _ := json.Marshal(event1)
	data2, _ := json.Marshal(event2)

	content := string(data1) + "\n" + string(data2) + "\n"
	if err := os.WriteFile(tmpFile, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to create transcript: %v", err)
	}

	// Resolve the allowlist for "ta-go-builder".
	spec := resolveFromTranscript(tmpFile, "ta-go-builder")

	// Should get the SECOND (last) dispatch's allowlist.
	if spec == nil {
		// Debug: read file and print first 500 chars
		data, _ := os.ReadFile(tmpFile)
		t.Logf("transcript content (first 500 chars):\n%s\n", string(data)[:minInt(len(data), 500)])
		t.Fatalf("expected non-nil spec, got nil")
	}

	if !stringsEqual(spec.Edit, []string{"/file2", "/file3"}) {
		t.Errorf("Edit mismatch: got %v, want [/file2 /file3]", spec.Edit)
	}
	if !stringsEqual(spec.BashDeny, []string{"git commit"}) {
		t.Errorf("BashDeny mismatch: got %v, want [git commit]", spec.BashDeny)
	}
}

// TestResolveFromTranscriptEmptyVsNil verifies nil-vs-empty distinction in transcript.
func TestResolveFromTranscriptEmptyVsNil(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "transcript.jsonl")

	// Event with edit present-empty ([]) and bash_deny absent (nil).
	event := map[string]interface{}{
		"message": map[string]interface{}{
			"content": []interface{}{
				map[string]interface{}{
					"type": "tool_use",
					"name": "Agent",
					"input": map[string]interface{}{
						"subagent_type": "ta-qa-proof",
						"prompt":        "<TA_ALLOWLIST>\n{\"edit\": []}\n</TA_ALLOWLIST>",
					},
				},
			},
		},
	}

	data, _ := json.Marshal(event)
	if err := os.WriteFile(tmpFile, []byte(string(data)+"\n"), 0o644); err != nil {
		t.Fatalf("failed to create transcript: %v", err)
	}

	spec := resolveFromTranscript(tmpFile, "ta-qa-proof")

	if spec == nil {
		t.Fatalf("expected non-nil spec, got nil")
	}

	// Edit should be non-nil empty slice (present in JSON as []).
	if spec.Edit == nil {
		t.Errorf("Edit should be non-nil empty slice, got nil")
	}
	if len(spec.Edit) != 0 {
		t.Errorf("Edit should be empty, got len=%d", len(spec.Edit))
	}

	// BashDeny should be nil (absent from JSON).
	if spec.BashDeny != nil {
		t.Errorf("BashDeny should be nil (absent from JSON), got %v", spec.BashDeny)
	}
}

// TestResolveAllowlistNoAgentIDDefers tests that agentID="" → defer (no transcript scan).
func TestResolveAllowlistNoAgentIDDefers(t *testing.T) {
	os.Unsetenv("TILL_GATE_ALLOWLIST")

	tmpFile := filepath.Join(t.TempDir(), "transcript.jsonl")
	event := map[string]interface{}{
		"message": map[string]interface{}{
			"content": []interface{}{
				map[string]interface{}{
					"type": "tool_use",
					"name": "Agent",
					"input": map[string]interface{}{
						"subagent_type": "ta-go-builder",
						"prompt":        "<TA_ALLOWLIST>\n{\"edit\": [\"/file1\"]}\n</TA_ALLOWLIST>",
					},
				},
			},
		},
	}

	data, _ := json.Marshal(event)
	if err := os.WriteFile(tmpFile, []byte(string(data)+"\n"), 0o644); err != nil {
		t.Fatalf("failed to create transcript: %v", err)
	}

	// Call ResolveAllowlist with empty agentID and empty env.
	// Should return nil (defer) without scanning the transcript.
	spec := ResolveAllowlist(context.Background(), "", "ta-go-builder", tmpFile)

	if spec != nil {
		t.Errorf("expected nil (defer) when agentID is empty, got %v", spec)
	}
}

// TestResolveAllowlistEnvTakesPrecedence tests that env var wins over transcript.
func TestResolveAllowlistEnvTakesPrecedence(t *testing.T) {
	// Set env var with a specific allowlist.
	os.Setenv("TILL_GATE_ALLOWLIST", `{"edit": ["/env/file"]}`)
	defer os.Unsetenv("TILL_GATE_ALLOWLIST")

	// Create a transcript with a different allowlist.
	tmpFile := filepath.Join(t.TempDir(), "transcript.jsonl")
	event := map[string]interface{}{
		"message": map[string]interface{}{
			"content": []interface{}{
				map[string]interface{}{
					"type": "tool_use",
					"name": "Agent",
					"input": map[string]interface{}{
						"subagent_type": "ta-go-builder",
						"prompt":        "<TA_ALLOWLIST>\n{\"edit\": [\"/transcript/file\"]}\n</TA_ALLOWLIST>",
					},
				},
			},
		},
	}

	data, _ := json.Marshal(event)
	if err := os.WriteFile(tmpFile, []byte(string(data)+"\n"), 0o644); err != nil {
		t.Fatalf("failed to create transcript: %v", err)
	}

	// Resolve with a non-empty agentID (so transcript scanning is enabled).
	spec := ResolveAllowlist(context.Background(), "agent-123", "ta-go-builder", tmpFile)

	if spec == nil {
		t.Fatalf("expected non-nil spec, got nil")
	}

	// Should get the env var's allowlist, not the transcript's.
	if !stringsEqual(spec.Edit, []string{"/env/file"}) {
		t.Errorf("Edit should be from env var: got %v, want [/env/file]", spec.Edit)
	}
}

// Helper: convert slices to pointers for test expectations.
func ptrSlice(s []string) *[]string {
	return &s
}

// Helper: return minimum of two ints.
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Helper: compare slice pointer with actual slice (nil vs empty).
func slicesEqual(actual []string, expected *[]string) bool {
	if expected == nil {
		return actual == nil
	}
	if actual == nil {
		return false
	}
	if len(actual) != len(*expected) {
		return false
	}
	for i := range actual {
		if actual[i] != (*expected)[i] {
			return false
		}
	}
	return true
}

// Helper: compare two string slices for equality.
func stringsEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
