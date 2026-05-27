package pretoolgate

import (
	"errors"
	"strings"
	"testing"
)

func TestDecodeGateConfig(t *testing.T) {
	tests := []struct {
		name         string
		toml         string
		wantErr      bool
		wantSentinel error
		wantLen      int
	}{
		{
			name: "valid single builder role",
			toml: `
[[gate.roles]]
role          = "builder"
cli_kind      = "claude"
channel       = "builtin"
oauth         = true
mcp_grants    = ["context7","gopls"]
edit          = ["//abs/file.go"]
writable_dirs = []
bash_deny     = []
network       = false
`,
			wantErr: false,
			wantLen: 1,
		},
		{
			name: "valid multiple roles (builder + planner)",
			toml: `
[[gate.roles]]
role          = "builder"
cli_kind      = "claude"
channel       = "builtin"
oauth         = true
mcp_grants    = ["context7"]
edit          = ["//file.go"]
writable_dirs = []
bash_deny     = []
network       = false

[[gate.roles]]
role          = "planning"
cli_kind      = "codex"
channel       = "subprocess"
oauth         = false
mcp_grants    = ["context7","hylla"]
edit          = []
writable_dirs = []
bash_deny     = ["git commit"]
network       = false
`,
			wantErr: false,
			wantLen: 2,
		},
		{
			name:    "empty roles array",
			toml:    ``,
			wantErr: false,
			wantLen: 0,
		},
		{
			name: "malformed TOML (syntax error)",
			toml: `
[[gate.roles]]
role = "builder"
cli_kind = [unclosed,
`,
			wantErr:      true,
			wantSentinel: ErrGateConfigDecode,
		},
		{
			name: "unknown field in strict mode",
			toml: `
[[gate.roles]]
role          = "builder"
cli_kind      = "claude"
channel       = "builtin"
oauth         = true
mcp_grants    = []
edit          = []
writable_dirs = []
bash_deny     = []
network       = false
unknown_field = "should fail"
`,
			wantErr:      true,
			wantSentinel: ErrGateConfigDecode,
		},
		{
			name: "validation failure: codex + edit non-empty",
			toml: `
[[gate.roles]]
role          = "builder"
cli_kind      = "codex"
channel       = "subprocess"
oauth         = false
mcp_grants    = []
edit          = ["//file.go"]
writable_dirs = []
bash_deny     = []
network       = false
`,
			wantErr:      true,
			wantSentinel: ErrCodexReadOnly,
		},
		{
			name: "validation failure: OAuth + subprocess",
			toml: `
[[gate.roles]]
role          = "plan-qa-proof"
cli_kind      = "claude"
channel       = "subprocess"
oauth         = true
mcp_grants    = []
edit          = []
writable_dirs = []
bash_deny     = []
network       = false
`,
			wantErr:      true,
			wantSentinel: ErrOAuthMustUseBuiltin,
		},
		{
			name: "validation failure: build-qa-proof + hylla",
			toml: `
[[gate.roles]]
role          = "build-qa-proof"
cli_kind      = "claude"
channel       = "builtin"
oauth         = true
mcp_grants    = ["hylla"]
edit          = []
writable_dirs = []
bash_deny     = []
network       = false
`,
			wantErr:      true,
			wantSentinel: ErrForbiddenMCP,
		},
		{
			name: "validation failure: non-editing role with nil Edit",
			toml: `
[[gate.roles]]
role          = "planning"
cli_kind      = "claude"
channel       = "builtin"
oauth         = true
mcp_grants    = []
writable_dirs = []
bash_deny     = []
network       = false
`,
			wantErr:      true,
			wantSentinel: ErrNonEditingRoleNilEdit,
		},
		{
			name: "validation failure: non-editing role with non-empty Edit",
			toml: `
[[gate.roles]]
role          = "plan-qa-falsification"
cli_kind      = "claude"
channel       = "builtin"
oauth         = true
mcp_grants    = []
edit          = ["//file.go"]
writable_dirs = []
bash_deny     = []
network       = false
`,
			wantErr:      true,
			wantSentinel: ErrNonEditingRoleHasEdit,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := DecodeGateConfig(strings.NewReader(tt.toml))

			if tt.wantErr {
				if err == nil {
					t.Errorf("DecodeGateConfig() error = nil, want error")
					return
				}
				if tt.wantSentinel != nil && !errors.Is(err, tt.wantSentinel) {
					t.Errorf("DecodeGateConfig() error = %v, want errors.Is(err, %v)", err, tt.wantSentinel)
				}
				return
			}

			if err != nil {
				t.Errorf("DecodeGateConfig() error = %v, want nil", err)
				return
			}

			if len(got) != tt.wantLen {
				t.Errorf("DecodeGateConfig() len = %d, want %d", len(got), tt.wantLen)
			}
		})
	}
}
