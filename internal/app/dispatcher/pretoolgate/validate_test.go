package pretoolgate

import (
	"errors"
	"testing"
)

func TestValidateRoleGate(t *testing.T) {
	tests := []struct {
		name       string
		rg         RoleGate
		wantErr    bool
		wantSentinel error
	}{
		// Valid cases
		{
			name: "valid go-builder (claude/builtin/oauth, edit non-empty)",
			rg: RoleGate{
				Role:      "builder",
				CLIKind:   "claude",
				Channel:   ChannelBuiltin,
				OAuth:     true,
				MCPGrants: []string{"context7", "gopls"},
				Spec: GateSpec{
					Edit:        []string{"//absolute/path/file.go"},
					WritableDirs: nil,
				},
			},
			wantErr: false,
		},
		{
			name: "valid codex planner (codex/subprocess, edit empty, writable_dirs empty)",
			rg: RoleGate{
				Role:      "planning",
				CLIKind:   "codex",
				Channel:   ChannelSubprocess,
				OAuth:     false,
				MCPGrants: []string{"context7"},
				Spec: GateSpec{
					Edit:         []string{},
					WritableDirs: []string{},
				},
			},
			wantErr: false,
		},
		{
			name: "valid plan-qa-proof (read-only, edit present-empty)",
			rg: RoleGate{
				Role:      "plan-qa-proof",
				CLIKind:   "claude",
				Channel:   ChannelBuiltin,
				OAuth:     true,
				MCPGrants: []string{"hylla"},
				Spec: GateSpec{
					Edit: []string{},
				},
			},
			wantErr: false,
		},
		{
			name: "valid build-qa-falsification without hylla (edit present-empty)",
			rg: RoleGate{
				Role:      "build-qa-falsification",
				CLIKind:   "claude",
				Channel:   ChannelBuiltin,
				OAuth:     true,
				MCPGrants: []string{"context7"},
				Spec: GateSpec{
					Edit: []string{},
				},
			},
			wantErr: false,
		},

		// Case 1: codex + non-empty Edit
		{
			name: "invalid: codex + edit non-empty (should be read-only)",
			rg: RoleGate{
				Role:      "builder",
				CLIKind:   "codex",
				Channel:   ChannelSubprocess,
				OAuth:     false,
				MCPGrants: nil,
				Spec: GateSpec{
					Edit: []string{"//file.go"},
				},
			},
			wantErr:      true,
			wantSentinel: ErrCodexReadOnly,
		},

		// Case 2: OAuth + subprocess
		{
			name: "invalid: OAuth role + subprocess channel",
			rg: RoleGate{
				Role:      "builder",
				CLIKind:   "claude",
				Channel:   ChannelSubprocess,
				OAuth:     true,
				MCPGrants: nil,
				Spec: GateSpec{
					Edit: []string{"//file.go"},
				},
			},
			wantErr:      true,
			wantSentinel: ErrOAuthMustUseBuiltin,
		},

		// Case 3: build-qa-proof + hylla
		{
			name: "invalid: build-qa-proof + hylla MCP",
			rg: RoleGate{
				Role:      "build-qa-proof",
				CLIKind:   "claude",
				Channel:   ChannelBuiltin,
				OAuth:     true,
				MCPGrants: []string{"hylla"},
				Spec: GateSpec{
					Edit: []string{},
				},
			},
			wantErr:      true,
			wantSentinel: ErrForbiddenMCP,
		},

		// Case 3: build-qa-falsification + hylla
		{
			name: "invalid: build-qa-falsification + hylla MCP",
			rg: RoleGate{
				Role:      "build-qa-falsification",
				CLIKind:   "claude",
				Channel:   ChannelBuiltin,
				OAuth:     true,
				MCPGrants: []string{"context7", "hylla"},
				Spec: GateSpec{
					Edit: []string{},
				},
			},
			wantErr:      true,
			wantSentinel: ErrForbiddenMCP,
		},

		// Case 3 negative control: builder + hylla is OK
		{
			name: "valid: builder (non-qa-role) + hylla MCP (no restriction)",
			rg: RoleGate{
				Role:      "builder",
				CLIKind:   "claude",
				Channel:   ChannelBuiltin,
				OAuth:     true,
				MCPGrants: []string{"hylla", "context7"},
				Spec: GateSpec{
					Edit: []string{"//file.go"},
				},
			},
			wantErr: false,
		},

		// Case 4a: non-editing role with nil Edit
		{
			name: "invalid: plan-qa-proof with nil Edit (inapplicable)",
			rg: RoleGate{
				Role:      "plan-qa-proof",
				CLIKind:   "claude",
				Channel:   ChannelBuiltin,
				OAuth:     true,
				MCPGrants: nil,
				Spec: GateSpec{
					Edit: nil,
				},
			},
			wantErr:      true,
			wantSentinel: ErrNonEditingRoleNilEdit,
		},

		// Case 4b: non-editing role with non-empty Edit
		{
			name: "invalid: planning role with non-empty Edit",
			rg: RoleGate{
				Role:      "planning",
				CLIKind:   "claude",
				Channel:   ChannelBuiltin,
				OAuth:     true,
				MCPGrants: nil,
				Spec: GateSpec{
					Edit: []string{"//file.go"},
				},
			},
			wantErr:      true,
			wantSentinel: ErrNonEditingRoleHasEdit,
		},

		// Case 4 negative control: planning role with present-empty Edit
		{
			name: "valid: planning role with Edit:[] (present-empty, read-only)",
			rg: RoleGate{
				Role:      "planning",
				CLIKind:   "claude",
				Channel:   ChannelBuiltin,
				OAuth:     true,
				MCPGrants: nil,
				Spec: GateSpec{
					Edit: []string{},
				},
			},
			wantErr: false,
		},

		// Additional case: closeout (non-editing) with nil Edit
		{
			name: "invalid: closeout role with nil Edit",
			rg: RoleGate{
				Role:      "closeout",
				CLIKind:   "claude",
				Channel:   ChannelBuiltin,
				OAuth:     true,
				MCPGrants: nil,
				Spec: GateSpec{
					Edit: nil,
				},
			},
			wantErr:      true,
			wantSentinel: ErrNonEditingRoleNilEdit,
		},

		// Case 3: plan-qa-* roles must NOT restrict hylla (builder restriction only)
		{
			name: "valid: plan-qa-falsification + hylla (not a build-qa role)",
			rg: RoleGate{
				Role:      "plan-qa-falsification",
				CLIKind:   "claude",
				Channel:   ChannelBuiltin,
				OAuth:     true,
				MCPGrants: []string{"hylla"},
				Spec: GateSpec{
					Edit: []string{},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRoleGate(tt.rg)
			if !tt.wantErr && err != nil {
				t.Errorf("expected nil, got %v", err)
			}
			if tt.wantErr && err == nil {
				t.Errorf("expected error, got nil")
			}
			if tt.wantErr && !errors.Is(err, tt.wantSentinel) {
				t.Errorf("expected sentinel %v, got %v", tt.wantSentinel, err)
			}
		})
	}
}

func TestContainsGrant(t *testing.T) {
	tests := []struct {
		grants []string
		grant  string
		want   bool
	}{
		{
			grants: []string{"hylla", "context7"},
			grant:  "hylla",
			want:   true,
		},
		{
			grants: []string{"context7", "gopls"},
			grant:  "hylla",
			want:   false,
		},
		{
			grants: []string{},
			grant:  "hylla",
			want:   false,
		},
		{
			grants: nil,
			grant:  "hylla",
			want:   false,
		},
		{
			grants: []string{"HYLLA"},
			grant:  "hylla",
			want:   false, // case-sensitive
		},
	}

	for _, tt := range tests {
		got := containsGrant(tt.grants, tt.grant)
		if got != tt.want {
			t.Errorf("containsGrant(%v, %q) = %v, want %v", tt.grants, tt.grant, got, tt.want)
		}
	}
}

func TestIsNonEditingRole(t *testing.T) {
	tests := []struct {
		role string
		want bool
	}{
		{"planning", true},
		{"plan-qa-proof", true},
		{"plan-qa-falsification", true},
		{"build-qa-proof", true},
		{"build-qa-falsification", true},
		{"closeout", true},
		{"builder", false},
		{"unknown-role", false},
		{"", false},
	}

	for _, tt := range tests {
		got := isNonEditingRole(tt.role)
		if got != tt.want {
			t.Errorf("isNonEditingRole(%q) = %v, want %v", tt.role, got, tt.want)
		}
	}
}
