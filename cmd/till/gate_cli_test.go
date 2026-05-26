package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/evanmschultz/tillsyn/internal/app/dispatcher/pretoolgate"
)

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

// TestGateStdinMalformedJSON verifies that malformed JSON defers (exit 0, no output).
func TestGateStdinMalformedJSON(t *testing.T) {
	cmd := newGateCommand()
	cmd.SetArgs([]string{})

	stdin := bytes.NewBufferString(`{invalid json`)
	stdout := new(bytes.Buffer)

	cmd.SetIn(stdin)
	cmd.SetOut(stdout)

	err := cmd.ExecuteContext(context.Background())
	if err != nil {
		t.Errorf("expected nil error, got: %v", err)
	}
	if stdout.Len() > 0 {
		t.Errorf("expected empty stdout (defer), got: %s", stdout.String())
	}
}

// TestGateStdinEmptyDefers verifies that empty stdin defers (exit 0, no output).
func TestGateStdinEmptyDefers(t *testing.T) {
	cmd := newGateCommand()
	cmd.SetArgs([]string{})

	stdin := bytes.NewBufferString("")
	stdout := new(bytes.Buffer)

	cmd.SetIn(stdin)
	cmd.SetOut(stdout)

	err := cmd.ExecuteContext(context.Background())
	if err != nil {
		t.Errorf("expected nil error, got: %v", err)
	}
	if stdout.Len() > 0 {
		t.Errorf("expected empty stdout (defer), got: %s", stdout.String())
	}
}

// TestGateUngatedDeferEmptyOutput verifies that ungated events (no agent_id, no allowlist)
// defer with empty stdout.
func TestGateUngatedDeferEmptyOutput(t *testing.T) {
	cmd := newGateCommand()
	cmd.SetArgs([]string{})

	event := pretoolgate.Event{
		ToolName: "Edit",
		Cwd:      "/repo",
		// No agent_id, no allowlist env → should defer
	}
	data, _ := json.Marshal(event)

	stdin := bytes.NewBufferString(string(data))
	stdout := new(bytes.Buffer)

	cmd.SetIn(stdin)
	cmd.SetOut(stdout)

	// Clear env to ensure no allowlist
	oldEnv := os.Getenv("TILL_GATE_ALLOWLIST")
	os.Unsetenv("TILL_GATE_ALLOWLIST")
	defer func() {
		if oldEnv != "" {
			os.Setenv("TILL_GATE_ALLOWLIST", oldEnv)
		}
	}()

	err := cmd.ExecuteContext(context.Background())
	if err != nil {
		t.Errorf("expected nil error, got: %v", err)
	}
	if stdout.Len() > 0 {
		t.Errorf("expected empty stdout (defer for ungated), got: %s", stdout.String())
	}
}

// TestGateEditDenyOffAllowlist verifies that an Edit tool targeting a file NOT in the
// allowlist produces a deny decision with JSON output.
func TestGateEditDenyOffAllowlist(t *testing.T) {
	cmd := newGateCommand()
	cmd.SetArgs([]string{})

	// Set allowlist with a restricted edit scope
	allowlist := map[string]interface{}{
		"edit": []string{"/repo/allowed.go"},
	}
	allowlistJSON, _ := json.Marshal(allowlist)
	os.Setenv("TILL_GATE_ALLOWLIST", string(allowlistJSON))
	defer os.Unsetenv("TILL_GATE_ALLOWLIST")

	event := pretoolgate.Event{
		ToolName: "Edit",
		ToolInput: map[string]any{
			"file_path": "/repo/forbidden.go",
		},
		Cwd:       "/repo",
		AgentID:   "test-agent-1",
		AgentType: "ta-test",
	}
	data, _ := json.Marshal(event)

	stdin := bytes.NewBufferString(string(data))
	stdout := new(bytes.Buffer)

	cmd.SetIn(stdin)
	cmd.SetOut(stdout)

	err := cmd.ExecuteContext(context.Background())
	if err != nil {
		t.Errorf("expected nil error, got: %v", err)
	}

	out := stdout.String()
	if !strings.Contains(out, "deny") {
		t.Errorf("expected 'deny' in stdout, got: %s", out)
	}
	if !strings.Contains(out, "permissionDecision") {
		t.Errorf("expected 'permissionDecision' in JSON output, got: %s", out)
	}

	// Verify valid JSON
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Errorf("expected valid JSON output, got error: %v", err)
	}
}

// TestGateEditAllowInAllowlist verifies that an Edit tool targeting a file IN the
// allowlist produces an allow decision.
func TestGateEditAllowInAllowlist(t *testing.T) {
	cmd := newGateCommand()
	cmd.SetArgs([]string{})

	// Set allowlist with a restricted edit scope
	allowlist := map[string]interface{}{
		"edit": []string{"/repo/allowed.go"},
	}
	allowlistJSON, _ := json.Marshal(allowlist)
	os.Setenv("TILL_GATE_ALLOWLIST", string(allowlistJSON))
	defer os.Unsetenv("TILL_GATE_ALLOWLIST")

	event := pretoolgate.Event{
		ToolName: "Edit",
		ToolInput: map[string]any{
			"file_path": "/repo/allowed.go",
		},
		Cwd:       "/repo",
		AgentID:   "test-agent-2",
		AgentType: "ta-test",
	}
	data, _ := json.Marshal(event)

	stdin := bytes.NewBufferString(string(data))
	stdout := new(bytes.Buffer)

	cmd.SetIn(stdin)
	cmd.SetOut(stdout)

	err := cmd.ExecuteContext(context.Background())
	if err != nil {
		t.Errorf("expected nil error, got: %v", err)
	}

	out := stdout.String()
	if !strings.Contains(out, "allow") {
		t.Errorf("expected 'allow' in stdout, got: %s", out)
	}
	if !strings.Contains(out, "permissionDecision") {
		t.Errorf("expected 'permissionDecision' in JSON output, got: %s", out)
	}
}

// TestGateBashGitFloorDeny verifies that a Bash git mutation (git commit) is denied
// by the hardcoded baseline floor, regardless of bash_deny allowlist.
func TestGateBashGitFloorDeny(t *testing.T) {
	cmd := newGateCommand()
	cmd.SetArgs([]string{})

	// Set allowlist with edit but no bash_deny (missing git from the list)
	allowlist := map[string]interface{}{
		"edit": []string{"/repo/allowed.go"},
		// bash_deny intentionally absent or empty
	}
	allowlistJSON, _ := json.Marshal(allowlist)
	os.Setenv("TILL_GATE_ALLOWLIST", string(allowlistJSON))
	defer os.Unsetenv("TILL_GATE_ALLOWLIST")

	event := pretoolgate.Event{
		ToolName: "Bash",
		ToolInput: map[string]any{
			"command": "git commit -m 'test'",
		},
		Cwd:       "/repo",
		AgentID:   "test-agent-3",
		AgentType: "ta-test",
	}
	data, _ := json.Marshal(event)

	stdin := bytes.NewBufferString(string(data))
	stdout := new(bytes.Buffer)

	cmd.SetIn(stdin)
	cmd.SetOut(stdout)

	err := cmd.ExecuteContext(context.Background())
	if err != nil {
		t.Errorf("expected nil error, got: %v", err)
	}

	out := stdout.String()
	if !strings.Contains(out, "deny") {
		t.Errorf("expected 'deny' for git commit (floor rule), got: %s", out)
	}
}

// TestGateBashAllowedCommand verifies that a Bash read-only command is allowed.
func TestGateBashAllowedCommand(t *testing.T) {
	cmd := newGateCommand()
	cmd.SetArgs([]string{})

	// Set allowlist
	allowlist := map[string]interface{}{
		"edit": []string{"/repo/allowed.go"},
	}
	allowlistJSON, _ := json.Marshal(allowlist)
	os.Setenv("TILL_GATE_ALLOWLIST", string(allowlistJSON))
	defer os.Unsetenv("TILL_GATE_ALLOWLIST")

	event := pretoolgate.Event{
		ToolName: "Bash",
		ToolInput: map[string]any{
			"command": "ls -la /repo",
		},
		Cwd:       "/repo",
		AgentID:   "test-agent-4",
		AgentType: "ta-test",
	}
	data, _ := json.Marshal(event)

	stdin := bytes.NewBufferString(string(data))
	stdout := new(bytes.Buffer)

	cmd.SetIn(stdin)
	cmd.SetOut(stdout)

	err := cmd.ExecuteContext(context.Background())
	if err != nil {
		t.Errorf("expected nil error, got: %v", err)
	}

	out := stdout.String()
	if !strings.Contains(out, "allow") {
		t.Errorf("expected 'allow' for read-only bash, got: %s", out)
	}
}

// TestGateReadToolAlwaysAllowed verifies that Read tools are always allowed.
func TestGateReadToolAlwaysAllowed(t *testing.T) {
	cmd := newGateCommand()
	cmd.SetArgs([]string{})

	// Set allowlist with no read tools in allowlist
	allowlist := map[string]interface{}{
		"edit": []string{"/repo/allowed.go"},
	}
	allowlistJSON, _ := json.Marshal(allowlist)
	os.Setenv("TILL_GATE_ALLOWLIST", string(allowlistJSON))
	defer os.Unsetenv("TILL_GATE_ALLOWLIST")

	event := pretoolgate.Event{
		ToolName: "Read",
		ToolInput: map[string]any{
			"file_path": "/repo/any-file.go",
		},
		Cwd:       "/repo",
		AgentID:   "test-agent-5",
		AgentType: "ta-test",
	}
	data, _ := json.Marshal(event)

	stdin := bytes.NewBufferString(string(data))
	stdout := new(bytes.Buffer)

	cmd.SetIn(stdin)
	cmd.SetOut(stdout)

	err := cmd.ExecuteContext(context.Background())
	if err != nil {
		t.Errorf("expected nil error, got: %v", err)
	}

	out := stdout.String()
	if !strings.Contains(out, "allow") {
		t.Errorf("expected 'allow' for Read tool, got: %s", out)
	}
}
