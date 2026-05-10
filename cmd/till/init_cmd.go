package main

import (
	"errors"
	"io"
	"strings"

	"github.com/spf13/cobra"
)

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
				return errors.New("till init: JSON parse not yet wired (W2.D3b)")
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
