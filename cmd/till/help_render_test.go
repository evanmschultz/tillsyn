package main

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// TestParseHelpExampleSegmentsKeepsFlagsAfterPlaceholders verifies angle-bracket placeholders do not downgrade later flags.
func TestParseHelpExampleSegmentsKeepsFlagsAfterPlaceholders(t *testing.T) {
	root := &cobra.Command{Use: "till"}
	lease := &cobra.Command{Use: "lease"}
	issue := &cobra.Command{Use: "issue"}
	root.AddCommand(lease)
	lease.AddCommand(issue)

	segments := parseHelpExampleSegments(issue, "till lease issue --project-id <project-id> --agent-name <agent-name> --role builder")
	wantKinds := map[string]helpTokenKind{
		"till":          helpTokenProgram,
		"lease":         helpTokenCommand,
		"issue":         helpTokenCommand,
		"--project-id":  helpTokenFlag,
		"<project-id>":  helpTokenArgument,
		"--agent-name":  helpTokenFlag,
		"<agent-name>":  helpTokenArgument,
		"--role":        helpTokenFlag,
		"builder":       helpTokenArgument,
	}

	for text, want := range wantKinds {
		if got, ok := lookupHelpTokenKind(segments, text); !ok {
			t.Fatalf("expected token %q in %v", text, segments)
		} else if got != want {
			t.Fatalf("token %q kind = %v, want %v", text, got, want)
		}
	}
}

// TestRunHelpExamplesKeepPlaceholderFlagsVisible verifies help rendering keeps full placeholder examples intact.
func TestRunHelpExamplesKeepPlaceholderFlagsVisible(t *testing.T) {
	var out strings.Builder
	if err := run(context.Background(), []string{"lease", "issue", "--help"}, &out, io.Discard); err != nil {
		t.Fatalf("run(help) error = %v", err)
	}
	output := out.String()
	for _, want := range []string{
		"till lease issue --project-id <project-id> --agent-name <agent-name> --role builder",
		"till lease issue --project-id <project-id> --scope-type task --scope-id <task-id> --agent-name <agent-name> --role qa --requested-ttl 30m",
		"till lease issue --project-id <project-id> --agent-name <agent-name> --role orchestrator --allow-equal-scope-delegation",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected %q in help output, got %q", want, output)
		}
	}
}

// lookupHelpTokenKind returns the kind of the first segment whose text matches.
func lookupHelpTokenKind(tokens []helpToken, text string) (helpTokenKind, bool) {
	for _, token := range tokens {
		if token.text == text {
			return token.kind, true
		}
	}
	return helpTokenWhitespace, false
}
