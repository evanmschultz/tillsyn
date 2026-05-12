package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"
)

// initJSONPayload is the schema for `till init --json '{...}'` headless
// invocations. `Name` and `Group` are required; `MCP` defaults to false
// (zero value). Group must be one of the W2-supported values
// (`till-gen`, `till-go`); `till-gdd` is greyed-out per SKETCH §9.3 and
// rejected as reserved.
type initJSONPayload struct {
	Name  string `json:"name"`
	Group string `json:"group"`
	MCP   bool   `json:"mcp"`
}

// allowedInitGroups lists the active agent groups `till init` accepts in
// W2. `till-gdd` is deliberately omitted — it is reserved per SKETCH §9.3
// and will be re-enabled once GDD methodology lands post-dogfood. Order
// is preserved for the validation error message.
var allowedInitGroups = []string{"till-gen", "till-go"}

// reservedInitGroups lists groups recognized in the schema but rejected
// at validation time. Each entry returns a tailored "reserved" error so
// callers can distinguish typos (unknown group) from intentional-but-not-
// yet-shipped groups.
var reservedInitGroups = map[string]string{
	"till-gdd": "till-gdd",
}

// newInitCommand returns the `till init` cobra command. D3a ships the
// skeleton: --json flag wired (default ""), RunE dispatches to a TUI stub
// or a JSON stub. Subsequent droplets fill the bodies — D3b wires the JSON
// payload parser, D4 wires runInitTUI's bubbletea walk, D5 wires the
// file-copy pipeline both branches share.
func newInitCommand(stdout io.Writer, rootOpts rootCommandOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Seed a Tillsyn project (agents directory, agents.toml, .gitignore, optional .mcp.json)",
		Long: strings.TrimSpace(`
Initialize the current working directory as a Tillsyn project: copy the
agent .md files for the chosen group into <project>/.tillsyn/agents/, write
agents.toml from the shipped example, ensure agents.local.toml is gitignored,
optionally register the project with Claude Code via .mcp.json, and create
the project record in the local Tillsyn database.

Run interactively (TUI walk for project name + group picker) or in headless
mode by passing a JSON payload via --json. Re-running till init in an
already-initialized project is safe — every write is idempotent and existing
files are skipped, never overwritten.
`),
		Example: strings.Join([]string{
			"  till init",
			"  till init --json '{\"name\":\"my-project\",\"group\":\"till-go\",\"mcp\":true}'",
		}, "\n"),
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			payload, err := cmd.Flags().GetString("json")
			if err != nil {
				return err
			}
			if strings.TrimSpace(payload) != "" {
				return runInitJSON(stdout, rootOpts, payload)
			}
			return runInitTUI(stdout, rootOpts)
		},
	}
	cmd.Flags().String("json", "", "Run init in headless mode with a JSON payload (e.g. --json '{\"name\":\"foo\",\"group\":\"till-go\",\"mcp\":false}')")
	return cmd
}

// runInitTUI drives the interactive bubbletea walk (project name + group
// picker) for `till init`. D3a ships a stub error; D4 wires the real walk.
func runInitTUI(stdout io.Writer, opts rootCommandOptions) error {
	_ = stdout
	_ = opts
	return errors.New("till init: TUI walk not yet wired (W2.D4)")
}

// runInitJSON parses the headless `--json` payload, validates required
// fields and the group selection, then dispatches to the shared file-copy
// pipeline. D3b ships parse + validation; the file-copy step is a stub
// that D5 fills in. The stub error is the contract D5 consumes, so the
// wording is preserved verbatim across D3b → D5.
func runInitJSON(stdout io.Writer, opts rootCommandOptions, payload string) error {
	_ = stdout
	_ = opts

	var parsed initJSONPayload
	if err := json.Unmarshal([]byte(payload), &parsed); err != nil {
		return fmt.Errorf("till init: invalid json payload: %w", err)
	}

	if err := validateInitPayload(parsed); err != nil {
		return err
	}

	// D5 wires the actual file-copy pipeline. Until then a successful
	// parse + validate surfaces this stub so callers (and tests) can
	// confirm the parser ran without short-circuiting on a malformed
	// payload.
	return errors.New("till init: file copy not yet wired (W2.D5)")
}

// validateInitPayload checks required fields and the group selection on
// a parsed `initJSONPayload`. Returns a wrapped error pointing at the
// first failed invariant; `Name` and `Group` are required, and `Group`
// must be one of `allowedInitGroups` (reserved groups like `till-gdd`
// surface a tailored "reserved" error).
func validateInitPayload(p initJSONPayload) error {
	if strings.TrimSpace(p.Name) == "" {
		return errors.New("till init: name required")
	}
	if strings.TrimSpace(p.Group) == "" {
		return errors.New("till init: group required")
	}
	if reserved, ok := reservedInitGroups[p.Group]; ok {
		return fmt.Errorf("till init: group must be one of %v; %q is reserved", allowedInitGroups, reserved)
	}
	for _, allowed := range allowedInitGroups {
		if p.Group == allowed {
			return nil
		}
	}
	return fmt.Errorf("till init: group must be one of %v; got %q", allowedInitGroups, p.Group)
}
