package pretoolgate

import (
	"encoding/json"
	"os"
	"testing"
)

// setGateEnv sets TILL_GATE_ALLOWLIST to the JSON-encoded map and returns a
// cleanup function that restores the previous state.
func setGateEnv(t *testing.T, spec map[string]any) func() {
	t.Helper()
	b, err := json.Marshal(spec)
	if err != nil {
		t.Fatalf("setGateEnv marshal: %v", err)
	}
	prev, prevOk := os.LookupEnv("TILL_GATE_ALLOWLIST")
	if err := os.Setenv("TILL_GATE_ALLOWLIST", string(b)); err != nil {
		t.Fatalf("setGateEnv setenv: %v", err)
	}
	return func() {
		if prevOk {
			_ = os.Setenv("TILL_GATE_ALLOWLIST", prev)
		} else {
			_ = os.Unsetenv("TILL_GATE_ALLOWLIST")
		}
	}
}

// clearGateEnv ensures TILL_GATE_ALLOWLIST is unset and returns cleanup.
func clearGateEnv(t *testing.T) func() {
	t.Helper()
	prev, prevOk := os.LookupEnv("TILL_GATE_ALLOWLIST")
	_ = os.Unsetenv("TILL_GATE_ALLOWLIST")
	return func() {
		if prevOk {
			_ = os.Setenv("TILL_GATE_ALLOWLIST", prev)
		}
	}
}

func TestDecide(t *testing.T) {
	const cwd = "/repo"
	const allowedFile = "/repo/a.go"

	tests := []struct {
		name       string
		setupEnv   func(t *testing.T) func() // returns cleanup
		ev         Event
		wantDefer  bool
		wantPerm   string // "allow" or "deny"; ignored when wantDefer=true
		wantSubstr string // non-empty: reason must contain this substring
	}{
		{
			// Case 1: no TILL_GATE_ALLOWLIST, no agent_id → ungated → defer.
			name: "case1_orch_defer_no_allowlist_no_agent_id",
			setupEnv: func(t *testing.T) func() {
				return clearGateEnv(t)
			},
			ev:        Event{ToolName: "Bash", ToolInput: map[string]any{"command": "mage ci"}, Cwd: cwd},
			wantDefer: true,
		},
		{
			// Case 2: gated builder, Bash `mage ci` (not in deny list) → allow.
			name: "case2_gated_bash_allowed",
			setupEnv: func(t *testing.T) func() {
				return setGateEnv(t, map[string]any{
					"edit":      []string{allowedFile},
					"bash_deny": []string{"git commit", "git push", "mage install"},
				})
			},
			ev:       Event{ToolName: "Bash", ToolInput: map[string]any{"command": "mage ci"}, Cwd: cwd, AgentID: "agent-1"},
			wantPerm: "allow",
		},
		{
			// Case 3: gated builder, Edit to off-scope file → deny.
			name: "case3_edit_off_scope_deny",
			setupEnv: func(t *testing.T) func() {
				return setGateEnv(t, map[string]any{
					"edit": []string{allowedFile},
				})
			},
			ev:         Event{ToolName: "Edit", ToolInput: map[string]any{"file_path": "/repo/other.go"}, Cwd: cwd, AgentID: "agent-1"},
			wantPerm:   "deny",
			wantSubstr: "prompt-vs-allowlist contradiction",
		},
		{
			// Case 4: gated qa role with edit:[] (present-empty) → deny all edits.
			name: "case4_edit_empty_list_deny_all",
			setupEnv: func(t *testing.T) func() {
				return setGateEnv(t, map[string]any{
					"edit": []string{}, // present-empty = deny-all
				})
			},
			ev:         Event{ToolName: "Write", ToolInput: map[string]any{"file_path": allowedFile}, Cwd: cwd, AgentID: "agent-qa"},
			wantPerm:   "deny",
			wantSubstr: "prompt-vs-allowlist contradiction",
		},
		{
			// Case 5: gated agent, Bash `git commit -m x` → deny via bash_deny git verb.
			name: "case5_bash_git_commit_deny_via_bash_deny",
			setupEnv: func(t *testing.T) func() {
				return setGateEnv(t, map[string]any{
					"edit":      []string{allowedFile},
					"bash_deny": []string{"git commit", "git push"},
				})
			},
			ev:         Event{ToolName: "Bash", ToolInput: map[string]any{"command": "git commit -m 'x'"}, Cwd: cwd, AgentID: "agent-1"},
			wantPerm:   "deny",
			wantSubstr: "forbidden pattern",
		},
		{
			// Fix #5 baseline proof (C1): spec.bash_deny does NOT list git verbs,
			// but gitMutation floor fires UNCONDITIONALLY and denies `git commit`.
			// This is the load-bearing ordering invariant.
			name: "fix5_baseline_git_commit_denied_even_without_git_in_bash_deny",
			setupEnv: func(t *testing.T) func() {
				return setGateEnv(t, map[string]any{
					"edit":      []string{allowedFile},
					"bash_deny": []string{"mage install"}, // git NOT in bash_deny
				})
			},
			ev:         Event{ToolName: "Bash", ToolInput: map[string]any{"command": "git commit -m x"}, Cwd: cwd, AgentID: "agent-1"},
			wantPerm:   "deny",
			wantSubstr: "forbidden pattern",
		},
		{
			// Fix #5 baseline: same spec (no git in bash_deny), `git diff --stat` → allow.
			// Read-only git is NOT in the gitMutation floor.
			name: "fix5_baseline_git_diff_allowed",
			setupEnv: func(t *testing.T) func() {
				return setGateEnv(t, map[string]any{
					"edit":      []string{allowedFile},
					"bash_deny": []string{"mage install"},
				})
			},
			ev:       Event{ToolName: "Bash", ToolInput: map[string]any{"command": "git diff --stat"}, Cwd: cwd, AgentID: "agent-1"},
			wantPerm: "allow",
		},
		{
			// Fix #5 baseline: same spec, `mage ci` (no deny hit) → allow.
			name: "fix5_baseline_mage_ci_allowed",
			setupEnv: func(t *testing.T) func() {
				return setGateEnv(t, map[string]any{
					"edit":      []string{allowedFile},
					"bash_deny": []string{"mage install"},
				})
			},
			ev:       Event{ToolName: "Bash", ToolInput: map[string]any{"command": "mage ci"}, Cwd: cwd, AgentID: "agent-1"},
			wantPerm: "allow",
		},
		{
			// Case 12: gated agent, Bash `git diff HEAD` → allow (read-only git).
			name: "case12_bash_git_diff_allow",
			setupEnv: func(t *testing.T) func() {
				return setGateEnv(t, map[string]any{
					"edit":      []string{allowedFile},
					"bash_deny": []string{"git commit", "git push"},
				})
			},
			ev:       Event{ToolName: "Bash", ToolInput: map[string]any{"command": "git diff HEAD"}, Cwd: cwd, AgentID: "agent-1"},
			wantPerm: "allow",
		},
		{
			// Case 14: gated agent, Read tool → allow (falls through).
			name: "case14_read_tool_allow",
			setupEnv: func(t *testing.T) func() {
				return setGateEnv(t, map[string]any{
					"edit":      []string{allowedFile},
					"bash_deny": []string{"git commit"},
				})
			},
			ev:       Event{ToolName: "Read", ToolInput: map[string]any{"file_path": "/repo/other.go"}, Cwd: cwd, AgentID: "agent-1"},
			wantPerm: "allow",
		},
		{
			// bash_deny absent (nil), edit-scoped agent, shell write vector → deny.
			// spec.Edit != nil triggers the write-vector check.
			name: "bash_write_vector_denied_for_edit_scoped_agent",
			setupEnv: func(t *testing.T) func() {
				return setGateEnv(t, map[string]any{
					"edit": []string{allowedFile},
					// bash_deny absent
				})
			},
			ev:         Event{ToolName: "Bash", ToolInput: map[string]any{"command": "cat /etc/hosts > /repo/a.go"}, Cwd: cwd, AgentID: "agent-1"},
			wantPerm:   "deny",
			wantSubstr: "shell file-write/mutation vector",
		},
		{
			// bash_deny absent, edit key absent (nil), write vector → allow.
			// spec.Edit == nil means the "edit" key was absent → bashWriteVector check SKIPPED.
			name: "bash_write_vector_allowed_when_edit_key_absent",
			setupEnv: func(t *testing.T) func() {
				// No "edit" key — spec.Edit will be nil after mapToGateSpec.
				return setGateEnv(t, map[string]any{
					"bash_deny": []string{"mage install"},
				})
			},
			ev:       Event{ToolName: "Bash", ToolInput: map[string]any{"command": "cat /etc/hosts > /tmp/out.txt"}, Cwd: cwd, AgentID: "agent-1"},
			wantPerm: "allow",
		},
		{
			// Edit to in-scope file → allow.
			name: "edit_in_scope_allow",
			setupEnv: func(t *testing.T) func() {
				return setGateEnv(t, map[string]any{
					"edit": []string{allowedFile},
				})
			},
			ev:       Event{ToolName: "Edit", ToolInput: map[string]any{"file_path": allowedFile}, Cwd: cwd, AgentID: "agent-1"},
			wantPerm: "allow",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cleanup := tc.setupEnv(t)
			defer cleanup()

			got := Decide(tc.ev)

			if tc.wantDefer {
				if !got.Defer {
					t.Errorf("Decide() Defer = false, want true (Decision=%+v)", got)
				}
				return
			}

			if got.Defer {
				t.Errorf("Decide() Defer = true unexpectedly (Decision=%+v)", got)
				return
			}
			if got.Permission != tc.wantPerm {
				t.Errorf("Decide() Permission = %q, want %q (Reason=%q)", got.Permission, tc.wantPerm, got.Reason)
			}
			if tc.wantSubstr != "" {
				if got.Reason == "" {
					t.Errorf("Decide() Reason is empty, want substring %q", tc.wantSubstr)
				} else {
					found := false
					for i := 0; i <= len(got.Reason)-len(tc.wantSubstr); i++ {
						if got.Reason[i:i+len(tc.wantSubstr)] == tc.wantSubstr {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("Decide() Reason = %q, want substring %q", got.Reason, tc.wantSubstr)
					}
				}
			}
		})
	}
}

func TestMarshalDecision(t *testing.T) {
	t.Run("defer_returns_nil", func(t *testing.T) {
		b, err := MarshalDecision(Decision{Defer: true})
		if err != nil {
			t.Fatalf("MarshalDecision(defer) error: %v", err)
		}
		if b != nil {
			t.Errorf("MarshalDecision(defer) = %s, want nil", b)
		}
	})

	t.Run("allow_shape", func(t *testing.T) {
		b, err := MarshalDecision(Decision{Permission: "allow", Reason: "within the dispatch allowlist"})
		if err != nil {
			t.Fatalf("MarshalDecision(allow) error: %v", err)
		}
		var out map[string]any
		if err := json.Unmarshal(b, &out); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		inner, ok := out["hookSpecificOutput"].(map[string]any)
		if !ok {
			t.Fatalf("hookSpecificOutput missing or wrong type")
		}
		if inner["hookEventName"] != "PreToolUse" {
			t.Errorf("hookEventName = %v, want PreToolUse", inner["hookEventName"])
		}
		if inner["permissionDecision"] != "allow" {
			t.Errorf("permissionDecision = %v, want allow", inner["permissionDecision"])
		}
	})

	t.Run("deny_shape", func(t *testing.T) {
		b, err := MarshalDecision(Decision{Permission: "deny", Reason: "blocked"})
		if err != nil {
			t.Fatalf("MarshalDecision(deny) error: %v", err)
		}
		var out map[string]any
		if err := json.Unmarshal(b, &out); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		inner, ok := out["hookSpecificOutput"].(map[string]any)
		if !ok {
			t.Fatalf("hookSpecificOutput missing or wrong type")
		}
		if inner["permissionDecision"] != "deny" {
			t.Errorf("permissionDecision = %v, want deny", inner["permissionDecision"])
		}
	})
}
