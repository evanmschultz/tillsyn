package main

import "github.com/spf13/cobra"

// installHelpAliases keeps the help hook stable without mutating the Cobra tree.
func installHelpAliases(_ *cobra.Command) {}

// normalizeHelpInvocationArgs rewrites trailing `help` and `h` into `--help`.
func normalizeHelpInvocationArgs(args []string) []string {
	if len(args) == 0 {
		return nil
	}
	normalized := append([]string(nil), args...)
	switch normalized[len(normalized)-1] {
	case "help", "h":
		normalized[len(normalized)-1] = "--help"
	}
	return normalized
}
