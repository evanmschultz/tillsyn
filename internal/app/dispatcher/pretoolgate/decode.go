package pretoolgate

import (
	"errors"
	"fmt"
	"io"

	toml "github.com/pelletier/go-toml/v2"
)

// DecodeGateConfig decodes a TOML-encoded gate configuration from an io.Reader
// into a slice of validated RoleGate structs. It enforces strict TOML decoding
// (DisallowUnknownFields) and validates each decoded gate via ValidateRoleGate.
//
// On success, returns the decoded and validated []RoleGate slice (may be empty).
// On parse error (malformed TOML, unknown fields, type mismatch), returns
// ErrGateConfigDecode wrapped with position-aware error context.
// On validation error (per ValidateRoleGate), returns the first validation
// failure's sentinel error (ErrCodexReadOnly, ErrOAuthMustUseBuiltin,
// ErrForbiddenMCP, ErrNonEditingRoleNilEdit, ErrNonEditingRoleHasEdit) wrapped
// with position context so errors.Is can route on the sentinel.
func DecodeGateConfig(r io.Reader) ([]RoleGate, error) {
	if r == nil {
		return nil, errors.New("pretoolgate: nil reader")
	}

	// Decode struct mirroring the TOML shape: [[gate.roles]].
	var cfg struct {
		Gate struct {
			Roles []struct {
				Role         string   `toml:"role"`
				CLIKind      string   `toml:"cli_kind"`
				Channel      string   `toml:"channel"`
				OAuth        bool     `toml:"oauth"`
				MCPGrants    []string `toml:"mcp_grants"`
				Edit         []string `toml:"edit"`
				WritableDirs []string `toml:"writable_dirs"`
				BashDeny     []string `toml:"bash_deny"`
				Network      bool     `toml:"network"`
			} `toml:"roles"`
		} `toml:"gate"`
	}

	// Strict decode: DisallowUnknownFields rejects unknown keys, matching
	// internal/templates/load.go Step 3 pattern.
	decoder := toml.NewDecoder(r)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&cfg); err != nil {
		// Wrap parse errors (including StrictMissingError for unknown fields)
		// with ErrGateConfigDecode so callers can route on the sentinel.
		return nil, fmt.Errorf("%w: %v", ErrGateConfigDecode, err)
	}

	// Map decoded rows to RoleGate and validate each.
	result := make([]RoleGate, 0, len(cfg.Gate.Roles))
	for _, row := range cfg.Gate.Roles {
		rg := RoleGate{
			Role:      row.Role,
			CLIKind:   row.CLIKind,
			Channel:   Channel(row.Channel),
			OAuth:     row.OAuth,
			MCPGrants: row.MCPGrants,
			Spec: GateSpec{
				Edit:         row.Edit,
				WritableDirs: row.WritableDirs,
				BashDeny:     row.BashDeny,
				Network:      row.Network,
			},
		}

		// Validate via D2B's ValidateRoleGate.
		// The first validation error propagates with %w so errors.Is works.
		if err := ValidateRoleGate(rg); err != nil {
			return nil, err
		}

		result = append(result, rg)
	}

	return result, nil
}

// ErrGateConfigDecode is the sentinel error returned when TOML parsing fails
// (malformed syntax, unknown fields, type mismatch). Validation failures
// (via ValidateRoleGate) propagate their own sentinels instead.
var ErrGateConfigDecode = errors.New("gate config parse error")
