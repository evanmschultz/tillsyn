package pretoolgate

import (
	"testing"
)

func TestRoleGateShape(t *testing.T) {
	tests := []struct {
		name string
		rg   RoleGate
		want RoleGate
	}{
		{
			name: "zero-value RoleGate is legal",
			rg:   RoleGate{},
			want: RoleGate{},
		},
		{
			name: "populated RoleGate round-trips fields",
			rg: RoleGate{
				Role:      "builder",
				CLIKind:   "claude",
				Channel:   ChannelBuiltin,
				OAuth:     true,
				MCPGrants: []string{"context7", "gopls"},
				Spec: GateSpec{
					Edit:         []string{"//abs/file.go"},
					WritableDirs: []string{},
					BashDeny:     []string{"git commit", "git push"},
					Network:      false,
				},
			},
			want: RoleGate{
				Role:      "builder",
				CLIKind:   "claude",
				Channel:   ChannelBuiltin,
				OAuth:     true,
				MCPGrants: []string{"context7", "gopls"},
				Spec: GateSpec{
					Edit:         []string{"//abs/file.go"},
					WritableDirs: []string{},
					BashDeny:     []string{"git commit", "git push"},
					Network:      false,
				},
			},
		},
		{
			name: "read-only role with edit:[]",
			rg: RoleGate{
				Role:      "plan-qa-proof",
				CLIKind:   "claude",
				Channel:   ChannelBuiltin,
				OAuth:     true,
				MCPGrants: []string{"context7"},
				Spec: GateSpec{
					Edit:     []string{},
					BashDeny: []string{"git commit"},
					Network:  false,
				},
			},
			want: RoleGate{
				Role:      "plan-qa-proof",
				CLIKind:   "claude",
				Channel:   ChannelBuiltin,
				OAuth:     true,
				MCPGrants: []string{"context7"},
				Spec: GateSpec{
					Edit:     []string{},
					BashDeny: []string{"git commit"},
					Network:  false,
				},
			},
		},
		{
			name: "codex role with writable_dirs",
			rg: RoleGate{
				Role:      "planning",
				CLIKind:   "codex",
				Channel:   ChannelSubprocess,
				OAuth:     false,
				MCPGrants: []string{"context7"},
				Spec: GateSpec{
					WritableDirs: []string{"/abs/droplet-dir"},
					BashDeny:     []string{"git commit"},
					Network:      false,
				},
			},
			want: RoleGate{
				Role:      "planning",
				CLIKind:   "codex",
				Channel:   ChannelSubprocess,
				OAuth:     false,
				MCPGrants: []string{"context7"},
				Spec: GateSpec{
					WritableDirs: []string{"/abs/droplet-dir"},
					BashDeny:     []string{"git commit"},
					Network:      false,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Assert that the RoleGate round-trips its fields.
			if tt.rg.Role != tt.want.Role {
				t.Errorf("Role: got %q, want %q", tt.rg.Role, tt.want.Role)
			}
			if tt.rg.CLIKind != tt.want.CLIKind {
				t.Errorf("CLIKind: got %q, want %q", tt.rg.CLIKind, tt.want.CLIKind)
			}
			if tt.rg.Channel != tt.want.Channel {
				t.Errorf("Channel: got %q, want %q", tt.rg.Channel, tt.want.Channel)
			}
			if tt.rg.OAuth != tt.want.OAuth {
				t.Errorf("OAuth: got %v, want %v", tt.rg.OAuth, tt.want.OAuth)
			}
			if len(tt.rg.MCPGrants) != len(tt.want.MCPGrants) {
				t.Errorf("MCPGrants len: got %d, want %d", len(tt.rg.MCPGrants), len(tt.want.MCPGrants))
			}
			for i, grant := range tt.rg.MCPGrants {
				if i < len(tt.want.MCPGrants) && grant != tt.want.MCPGrants[i] {
					t.Errorf("MCPGrants[%d]: got %q, want %q", i, grant, tt.want.MCPGrants[i])
				}
			}
			if len(tt.rg.Spec.Edit) != len(tt.want.Spec.Edit) {
				t.Errorf("Spec.Edit len: got %d, want %d", len(tt.rg.Spec.Edit), len(tt.want.Spec.Edit))
			}
			if len(tt.rg.Spec.WritableDirs) != len(tt.want.Spec.WritableDirs) {
				t.Errorf("Spec.WritableDirs len: got %d, want %d", len(tt.rg.Spec.WritableDirs), len(tt.want.Spec.WritableDirs))
			}
		})
	}
}

func TestChannelEnumLiterals(t *testing.T) {
	tests := []struct {
		name string
		ch   Channel
		want string
	}{
		{
			name: "ChannelBuiltin equals its string literal",
			ch:   ChannelBuiltin,
			want: "builtin",
		},
		{
			name: "ChannelSubprocess equals its string literal",
			ch:   ChannelSubprocess,
			want: "subprocess",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.ch) != tt.want {
				t.Errorf("Channel literal: got %q, want %q", string(tt.ch), tt.want)
			}
		})
	}
}
