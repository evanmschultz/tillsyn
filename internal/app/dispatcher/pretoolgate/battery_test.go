package pretoolgate

import (
	"encoding/json"
	"os"
	"testing"
)

// TestBatteryGateTestShParity verifies 20 concrete test cases from the proven
// PoC reference (_sandbox_poc_reference/gate_test.sh) + the fix-#5 floor rows
// (close-C1 git-mutation-independent checks) + redirection-negative rows
// (close-F1 RE2 lookahead-port precision).
//
// All 20 rows must PASS for full parity with the Python oracle.
// This is the acceptance battery A.6 — the non-negotiable proof that the Go
// Decide port faithfully reproduces the proven-gate logic including edge cases.
func TestBatteryGateTestShParity(t *testing.T) {
	const cwd = "/repo"

	// Fixtures: two allowlists from gate_test.sh lines 40-41.
	allowOne := map[string]any{
		"edit":      []string{"/repo/a.go"},
		"bash_deny": []string{"git commit", "git push", "git add", "mage install", "go get"},
	}

	qaEmpty := map[string]any{
		"edit":      []string{},
		"bash_deny": []string{"git commit", "git push"},
	}

	// floorProbe: the orch-forgetful floor case (no git in bash_deny).
	floorProbe := map[string]any{
		"edit":      []string{"/repo/a.go"},
		"bash_deny": []string{"mage install"}, // intentionally NO git verbs
	}

	tests := []struct {
		name       string
		allowlist  map[string]any // nil => no env
		event      Event
		wantPerm   string // "defer", "allow", "deny"
		wantSubstr string // optional: reason substring to verify
	}{
		// === The 14 base cases from gate_test.sh ===

		{
			// Case 1: orch defer (no agent_id, no env) → defer
			name:      "1_orch_defer_no_agent_id_no_env",
			allowlist: nil,
			event: Event{
				ToolName:  "Edit",
				ToolInput: map[string]any{"file_path": "/repo/anything.go"},
				Cwd:       cwd,
				// no agent_id
			},
			wantPerm: "defer",
		},

		{
			// Case 2: edit in-scope → allow
			name:      "2_edit_in_scope",
			allowlist: allowOne,
			event: Event{
				ToolName:  "Edit",
				ToolInput: map[string]any{"file_path": "/repo/a.go"},
				Cwd:       cwd,
				AgentID:   "x",
				AgentType: "ta-go-builder",
			},
			wantPerm: "allow",
		},

		{
			// Case 3: edit off-scope → deny
			name:      "3_edit_off_scope",
			allowlist: allowOne,
			event: Event{
				ToolName:  "Write",
				ToolInput: map[string]any{"file_path": "/repo/b.go"},
				Cwd:       cwd,
				AgentID:   "x",
				AgentType: "ta-go-builder",
			},
			wantPerm: "deny",
		},

		{
			// Case 4: QA edit:[] (present-empty) → deny all edits
			name:      "4_qa_edit_empty_deny",
			allowlist: qaEmpty,
			event: Event{
				ToolName:  "Edit",
				ToolInput: map[string]any{"file_path": "/repo/review.go"},
				Cwd:       cwd,
				AgentID:   "x",
				AgentType: "ta-go-build-qa-proof",
			},
			wantPerm: "deny",
		},

		{
			// Case 5: bash git commit → deny
			name:      "5_bash_git_commit",
			allowlist: allowOne,
			event: Event{
				ToolName:  "Bash",
				ToolInput: map[string]any{"command": "git commit -m wip"},
				Cwd:       cwd,
				AgentID:   "x",
				AgentType: "ta-go-builder",
			},
			wantPerm: "deny",
		},

		{
			// Case 6: bash git -C dir commit (evasion) → deny
			name:      "6_bash_git_C_dir_commit_evasion",
			allowlist: allowOne,
			event: Event{
				ToolName:  "Bash",
				ToolInput: map[string]any{"command": "git -C /repo commit -m wip"},
				Cwd:       cwd,
				AgentID:   "x",
				AgentType: "ta-go-builder",
			},
			wantPerm: "deny",
		},

		{
			// Case 7: bash FOO=1 /usr/bin/git push (evasion) → deny
			name:      "7_bash_env_prefixed_git_push_evasion",
			allowlist: allowOne,
			event: Event{
				ToolName:  "Bash",
				ToolInput: map[string]any{"command": "FOO=1 /usr/bin/git push origin main"},
				Cwd:       cwd,
				AgentID:   "x",
				AgentType: "ta-go-builder",
			},
			wantPerm: "deny",
		},

		{
			// Case 8: bash echo > off-scope (write bypass) → deny
			name:      "8_bash_echo_redirect_write_bypass",
			allowlist: allowOne,
			event: Event{
				ToolName:  "Bash",
				ToolInput: map[string]any{"command": "echo pwned > /repo/b.go"},
				Cwd:       cwd,
				AgentID:   "x",
				AgentType: "ta-go-builder",
			},
			wantPerm: "deny",
		},

		{
			// Case 9: bash sed -i (write bypass) → deny
			name:      "9_bash_sed_i_write_bypass",
			allowlist: allowOne,
			event: Event{
				ToolName:  "Bash",
				ToolInput: map[string]any{"command": "sed -i s/a/b/ /repo/a.go"},
				Cwd:       cwd,
				AgentID:   "x",
				AgentType: "ta-go-builder",
			},
			wantPerm: "deny",
		},

		{
			// Case 10: bash python3 -c (write bypass) → deny
			name:      "10_bash_python3_c_write_bypass",
			allowlist: allowOne,
			event: Event{
				ToolName:  "Bash",
				ToolInput: map[string]any{"command": "python3 -c \"open(0,1)\""},
				Cwd:       cwd,
				AgentID:   "x",
				AgentType: "ta-go-builder",
			},
			wantPerm: "deny",
		},

		{
			// Case 11: bash mage install (bash_deny pattern) → deny
			name:      "11_bash_mage_install_deny_pattern",
			allowlist: allowOne,
			event: Event{
				ToolName:  "Bash",
				ToolInput: map[string]any{"command": "mage install"},
				Cwd:       cwd,
				AgentID:   "x",
				AgentType: "ta-go-builder",
			},
			wantPerm: "deny",
		},

		{
			// Case 12: bash git diff (read-only git, allowed) → allow
			name:      "12_bash_git_diff_read_allowed",
			allowlist: allowOne,
			event: Event{
				ToolName:  "Bash",
				ToolInput: map[string]any{"command": "git diff --stat"},
				Cwd:       cwd,
				AgentID:   "x",
				AgentType: "ta-go-builder",
			},
			wantPerm: "allow",
		},

		{
			// Case 13: bash mage ci (build command, allowed) → allow
			name:      "13_bash_mage_ci_allowed",
			allowlist: allowOne,
			event: Event{
				ToolName:  "Bash",
				ToolInput: map[string]any{"command": "mage ci"},
				Cwd:       cwd,
				AgentID:   "x",
				AgentType: "ta-go-builder",
			},
			wantPerm: "allow",
		},

		{
			// Case 14: Read tool (allowed) → allow
			name:      "14_read_tool_allowed",
			allowlist: allowOne,
			event: Event{
				ToolName:  "Read",
				ToolInput: map[string]any{"file_path": "/repo/whatever.go"},
				Cwd:       cwd,
				AgentID:   "x",
				AgentType: "ta-go-builder",
			},
			wantPerm: "allow",
		},

		// === Fix-#5 Git-Mutation Floor (closes C1) ===
		// These rows are the FIRST-CLASS battery rows that prove the baseline fires
		// even when the orch forgets to list git in bash_deny.

		{
			// Case 15: floor baseline — orch omits git from bash_deny,
			// but gitMutation floor fires UNCONDITIONALLY.
			// This is the exact counterexample from falsification verdict a686b305:
			// a port shipping bashForbidden alone would ALLOW here.
			// THE single most load-bearing row in the battery.
			name:      "15_floor_orch_omits_git_commit_denied",
			allowlist: floorProbe,
			event: Event{
				ToolName:  "Bash",
				ToolInput: map[string]any{"command": "git commit -m x"},
				Cwd:       cwd,
				AgentID:   "x",
				AgentType: "ta-go-builder",
			},
			wantPerm:   "deny",
			wantSubstr: "forbidden pattern",
		},

		{
			// Case 16: floor evasion — git -C /repo push with floor-probe spec
			name:      "16_floor_git_C_push_evasion_denied",
			allowlist: floorProbe,
			event: Event{
				ToolName:  "Bash",
				ToolInput: map[string]any{"command": "git -C /repo push origin main"},
				Cwd:       cwd,
				AgentID:   "x",
				AgentType: "ta-go-builder",
			},
			wantPerm: "deny",
		},

		{
			// Case 17: floor — fetch is a mutation verb in the hardcoded 28-verb set.
			// Proves exactness: not a curated subset.
			name:      "17_floor_git_fetch_mutation_denied",
			allowlist: floorProbe,
			event: Event{
				ToolName:  "Bash",
				ToolInput: map[string]any{"command": "FOO=1 /usr/bin/git fetch"},
				Cwd:       cwd,
				AgentID:   "x",
				AgentType: "ta-go-builder",
			},
			wantPerm: "deny",
		},

		{
			// Case 18: floor NEGATIVE — read-only git still allowed under floor-probe.
			// Guards against the floor over-blocking.
			// This proves that diff (∉ frozenset AND ∉ bash_deny) → allow.
			name:      "18_floor_git_diff_read_allowed_negative",
			allowlist: floorProbe,
			event: Event{
				ToolName:  "Bash",
				ToolInput: map[string]any{"command": "git diff --stat"},
				Cwd:       cwd,
				AgentID:   "x",
				AgentType: "ta-go-builder",
			},
			wantPerm: "allow",
		},

		// === Redirection-Negatives (closes F1) ===
		// RE2 lookahead-port documented failure mode:
		// The pattern `(?!&|/dev/null\b)` must correctly exclude /dev/null redirects.

		{
			// Case 19: cmd > /dev/null → ALLOW
			// The /dev/null redirection-target exclusion; NOT a write-vector.
			name:      "19_bash_redirection_to_dev_null_allowed",
			allowlist: allowOne,
			event: Event{
				ToolName:  "Bash",
				ToolInput: map[string]any{"command": "mage ci > /dev/null"},
				Cwd:       cwd,
				AgentID:   "x",
				AgentType: "ta-go-builder",
			},
			wantPerm: "allow",
		},

		{
			// Case 20: cmd >& /dev/null → ALLOW
			// The fd-dup (>&) exclusion; stdout+stderr redirect to null is allowed.
			name:      "20_bash_fd_dup_redirect_to_dev_null_allowed",
			allowlist: allowOne,
			event: Event{
				ToolName:  "Bash",
				ToolInput: map[string]any{"command": "mage test-pkg ./... >& /dev/null"},
				Cwd:       cwd,
				AgentID:   "x",
				AgentType: "ta-go-builder",
			},
			wantPerm: "allow",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Set up env if allowlist is provided.
			if tc.allowlist != nil {
				b, err := json.Marshal(tc.allowlist)
				if err != nil {
					t.Fatalf("marshal allowlist: %v", err)
				}
				prev, prevOk := os.LookupEnv("TILL_GATE_ALLOWLIST")
				if err := os.Setenv("TILL_GATE_ALLOWLIST", string(b)); err != nil {
					t.Fatalf("setenv: %v", err)
				}
				defer func() {
					if prevOk {
						_ = os.Setenv("TILL_GATE_ALLOWLIST", prev)
					} else {
						_ = os.Unsetenv("TILL_GATE_ALLOWLIST")
					}
				}()
			} else {
				// Clear env for this test.
				prev, prevOk := os.LookupEnv("TILL_GATE_ALLOWLIST")
				_ = os.Unsetenv("TILL_GATE_ALLOWLIST")
				defer func() {
					if prevOk {
						_ = os.Setenv("TILL_GATE_ALLOWLIST", prev)
					}
				}()
			}

			// Call Decide.
			got := Decide(tc.event)

			// Check the verdict.
			switch tc.wantPerm {
			case "defer":
				if !got.Defer {
					t.Errorf("Decide() Defer = false, want true; Decision = %+v", got)
				}

			case "allow":
				if got.Defer {
					t.Errorf("Decide() Defer = true, want false (allow); Decision = %+v", got)
					return
				}
				if got.Permission != "allow" {
					t.Errorf("Decide() Permission = %q, want allow; Reason = %q", got.Permission, got.Reason)
				}

			case "deny":
				if got.Defer {
					t.Errorf("Decide() Defer = true, want false (deny); Decision = %+v", got)
					return
				}
				if got.Permission != "deny" {
					t.Errorf("Decide() Permission = %q, want deny; Reason = %q", got.Permission, got.Reason)
				}
				// Check reason substring if provided.
				if tc.wantSubstr != "" {
					if !contains(got.Reason, tc.wantSubstr) {
						t.Errorf("Decide() Reason = %q, want substring %q", got.Reason, tc.wantSubstr)
					}
				}

			default:
				t.Fatalf("invalid wantPerm: %q", tc.wantPerm)
			}
		})
	}
}

// TestTillGateAllowlistEnvVar asserts that the port reads TILL_GATE_ALLOWLIST
// (not the Python oracle's TA_GATE_ALLOWLIST).
// This is intentional per A's description: oracle TA_GATE_ALLOWLIST → till TILL_GATE_ALLOWLIST.
func TestTillGateAllowlistEnvVar(t *testing.T) {
	// Verify the env var name is correct.
	const expectedEnvVar = "TILL_GATE_ALLOWLIST"
	const pythonOracleEnvVar = "TA_GATE_ALLOWLIST" // the oracle's name, NOT ours

	spec := map[string]any{
		"edit":      []string{"/repo/a.go"},
		"bash_deny": []string{"git commit"},
	}

	// Set only TILL_GATE_ALLOWLIST; the oracle TA_GATE_ALLOWLIST is NOT set.
	b, _ := json.Marshal(spec)
	prev, prevOk := os.LookupEnv(expectedEnvVar)
	prevOracle, prevOracleOk := os.LookupEnv(pythonOracleEnvVar)

	_ = os.Setenv(expectedEnvVar, string(b))
	_ = os.Unsetenv(pythonOracleEnvVar)

	defer func() {
		if prevOk {
			_ = os.Setenv(expectedEnvVar, prev)
		} else {
			_ = os.Unsetenv(expectedEnvVar)
		}
		if prevOracleOk {
			_ = os.Setenv(pythonOracleEnvVar, prevOracle)
		} else {
			_ = os.Unsetenv(pythonOracleEnvVar)
		}
	}()

	// Call Decide; should resolve the spec from the TILL_GATE_ALLOWLIST env.
	ev := Event{
		ToolName:  "Edit",
		ToolInput: map[string]any{"file_path": "/repo/a.go"},
		Cwd:       "/repo",
		AgentID:   "agent",
		AgentType: "ta-go-builder",
	}

	got := Decide(ev)

	// Should NOT defer (the env is set).
	if got.Defer {
		t.Errorf("Decide() Defer = true, want false; env var was set correctly")
	}

	// Should allow the in-scope edit.
	if got.Permission != "allow" {
		t.Errorf("Decide() Permission = %q, want allow; the spec should have been loaded from TILL_GATE_ALLOWLIST", got.Permission)
	}
}

// contains is a helper to check if substr appears in s.
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
