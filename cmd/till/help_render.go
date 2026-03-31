package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	lipgloss "charm.land/lipgloss/v2"
	"github.com/charmbracelet/fang"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// helpTokenKind identifies the semantic role of one help-output token.
type helpTokenKind int

const (
	// helpTokenWhitespace renders separating spaces between tokens.
	helpTokenWhitespace helpTokenKind = iota
	// helpTokenProgram renders the root program name.
	helpTokenProgram
	// helpTokenCommand renders subcommand path segments.
	helpTokenCommand
	// helpTokenFlag renders CLI flags such as --project-id.
	helpTokenFlag
	// helpTokenArgument renders ordinary arguments and placeholders.
	helpTokenArgument
	// helpTokenDimmed renders optional usage markers and shell operators.
	helpTokenDimmed
	// helpTokenQuoted renders quoted string literals.
	helpTokenQuoted
	// helpTokenComment renders help-example comment lines.
	helpTokenComment
)

// helpToken stores one styled fragment in a rendered help example.
type helpToken struct {
	text string
	kind helpTokenKind
}

// helpStyles collects lipgloss styles for the local help renderer.
type helpStyles struct {
	text        lipgloss.Style
	title       lipgloss.Style
	block       lipgloss.Style
	comment     lipgloss.Style
	program     lipgloss.Style
	command     lipgloss.Style
	flag        lipgloss.Style
	argument    lipgloss.Style
	dimmed      lipgloss.Style
	quoted      lipgloss.Style
	description lipgloss.Style
	defaults    lipgloss.Style
}

// helpRow stores one key/description row in a help section.
type helpRow struct {
	key         string
	description string
	keyKind     helpTokenKind
}

// isHelpInvocation reports whether the normalized argument list is an explicit help request.
func isHelpInvocation(args []string) bool {
	for _, arg := range args {
		switch strings.TrimSpace(arg) {
		case "--help", "-h":
			return true
		}
	}
	return false
}

// executeHelpCommand runs one custom-styled help flow without routing through Fang's example tokenizer.
func executeHelpCommand(ctx context.Context, root *cobra.Command, args []string, stdout io.Writer) error {
	if root == nil {
		return nil
	}
	root.CompletionOptions.DisableDefaultCmd = true
	root.SetArgs(args)
	root.SetHelpFunc(func(cmd *cobra.Command, _ []string) {
		writeCommandHelp(stdout, cmd)
	})
	return root.ExecuteContext(ctx)
}

// writeCommandHelp renders one command help page with consistent example-token styling.
func writeCommandHelp(w io.Writer, cmd *cobra.Command) {
	if w == nil || cmd == nil {
		return
	}

	styles := newHelpStyles(w)
	writeHelpLongShort(w, styles, firstNonEmptyTrimmed(cmd.Long, strings.TrimSpace(cmd.Short)))

	usage := renderHelpUsage(cmd, styles)
	if usage != "" {
		writeHelpTitle(w, styles, "usage")
		writeHelpBlock(w, styles, usage)
	}

	examples := renderHelpExamples(cmd, styles)
	if len(examples) > 0 {
		writeHelpTitle(w, styles, "examples")
		writeHelpBlock(w, styles, strings.Join(examples, "\n"))
	}

	commands := collectHelpCommandRows(cmd)
	if len(commands) > 0 {
		writeHelpRows(w, styles, "commands", commands)
	}

	flags := collectHelpFlagRows(cmd)
	if len(flags) > 0 {
		writeHelpRows(w, styles, "flags", flags)
	}

	_, _ = fmt.Fprintln(w)
}

// newHelpStyles returns the local help palette, reusing Fang's default scheme when styled output is available.
func newHelpStyles(w io.Writer) helpStyles {
	styled := supportsStyledOutputFunc(w) && strings.TrimSpace(os.Getenv("NO_COLOR")) == ""
	if !styled {
		return helpStyles{
			text:        lipgloss.NewStyle(),
			title:       lipgloss.NewStyle(),
			block:       lipgloss.NewStyle(),
			comment:     lipgloss.NewStyle(),
			program:     lipgloss.NewStyle(),
			command:     lipgloss.NewStyle(),
			flag:        lipgloss.NewStyle(),
			argument:    lipgloss.NewStyle(),
			dimmed:      lipgloss.NewStyle(),
			quoted:      lipgloss.NewStyle(),
			description: lipgloss.NewStyle(),
			defaults:    lipgloss.NewStyle(),
		}
	}

	isDark := lipgloss.HasDarkBackground(os.Stdin, os.Stdout)
	scheme := fang.DefaultColorScheme(lipgloss.LightDark(isDark))

	return helpStyles{
		text: lipgloss.NewStyle().
			Foreground(scheme.Base),
		title: lipgloss.NewStyle().
			Bold(true).
			Foreground(scheme.Title).
			Transform(strings.ToUpper).
			Margin(1, 0, 0, 0),
		block: lipgloss.NewStyle().
			Background(scheme.Codeblock).
			Foreground(scheme.Base).
			MarginLeft(2).
			Padding(1, 2),
		comment: lipgloss.NewStyle().
			Foreground(scheme.Comment).
			Background(scheme.Codeblock),
		program: lipgloss.NewStyle().
			Foreground(scheme.Program).
			Background(scheme.Codeblock),
		command: lipgloss.NewStyle().
			Foreground(scheme.Command).
			Background(scheme.Codeblock),
		flag: lipgloss.NewStyle().
			Foreground(scheme.Flag).
			Background(scheme.Codeblock),
		argument: lipgloss.NewStyle().
			Foreground(scheme.Base).
			Background(scheme.Codeblock),
		dimmed: lipgloss.NewStyle().
			Foreground(scheme.DimmedArgument).
			Background(scheme.Codeblock),
		quoted: lipgloss.NewStyle().
			Foreground(scheme.QuotedString).
			Background(scheme.Codeblock),
		description: lipgloss.NewStyle().
			Foreground(scheme.Description),
		defaults: lipgloss.NewStyle().
			Foreground(scheme.FlagDefault),
	}
}

// writeHelpLongShort prints the descriptive lead-in text for one help page.
func writeHelpLongShort(w io.Writer, styles helpStyles, text string) {
	text = strings.TrimSpace(text)
	if text == "" {
		return
	}
	lines := strings.Split(text, "\n")
	_, _ = fmt.Fprintln(w)
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			_, _ = fmt.Fprintln(w)
			continue
		}
		_, _ = fmt.Fprintln(w, "  "+styles.text.Render(line))
	}
}

// writeHelpTitle prints one uppercased section title.
func writeHelpTitle(w io.Writer, styles helpStyles, title string) {
	_, _ = fmt.Fprintln(w, styles.title.Render(strings.TrimSpace(title)))
}

// writeHelpBlock prints one multi-line usage/examples block.
func writeHelpBlock(w io.Writer, styles helpStyles, body string) {
	body = strings.TrimSpace(body)
	if body == "" {
		return
	}
	if styles.block.GetBackground() == nil {
		for _, line := range strings.Split(body, "\n") {
			_, _ = fmt.Fprintln(w, "  "+line)
		}
		return
	}
	_, _ = fmt.Fprintln(w, styles.block.Render(body))
}

// writeHelpRows prints one commands-or-flags table.
func writeHelpRows(w io.Writer, styles helpStyles, title string, rows []helpRow) {
	if len(rows) == 0 {
		return
	}
	writeHelpTitle(w, styles, title)

	maxKeyWidth := 0
	for _, row := range rows {
		if width := lipgloss.Width(row.key); width > maxKeyWidth {
			maxKeyWidth = width
		}
	}
	for _, row := range rows {
		renderedKey := renderHelpToken(helpToken{text: row.key, kind: row.keyKind}, styles)
		padding := strings.Repeat(" ", max(2, maxKeyWidth-lipgloss.Width(row.key)+2))
		description := styles.description.Render(strings.TrimSpace(row.description))
		_, _ = fmt.Fprintln(w, "  "+renderedKey+padding+description)
	}
}

// renderHelpUsage styles one command's use line without Fang's placeholder misclassification.
func renderHelpUsage(cmd *cobra.Command, styles helpStyles) string {
	if cmd == nil {
		return ""
	}
	pathTokens := strings.Fields(cmd.CommandPath())
	if len(pathTokens) == 0 {
		return ""
	}

	useParts := strings.Fields(strings.TrimSpace(cmd.Use))
	argTokens := []string{}
	if len(useParts) > 1 {
		argTokens = append(argTokens, useParts[1:]...)
	}

	parts := []string{renderHelpToken(helpToken{text: pathTokens[0], kind: helpTokenProgram}, styles)}
	for _, command := range pathTokens[1:] {
		parts = append(parts, renderHelpToken(helpToken{text: " ", kind: helpTokenWhitespace}, styles))
		parts = append(parts, renderHelpToken(helpToken{text: command, kind: helpTokenCommand}, styles))
	}
	for _, token := range argTokens {
		parts = append(parts, renderHelpToken(helpToken{text: " ", kind: helpTokenWhitespace}, styles))
		parts = append(parts, renderHelpToken(helpUsageToken(token), styles))
	}
	if cmd.HasAvailableSubCommands() {
		parts = append(parts, renderHelpToken(helpToken{text: " ", kind: helpTokenWhitespace}, styles))
		parts = append(parts, renderHelpToken(helpToken{text: "[command]", kind: helpTokenDimmed}, styles))
	}
	if commandHasAnyFlags(cmd) {
		parts = append(parts, renderHelpToken(helpToken{text: " ", kind: helpTokenWhitespace}, styles))
		parts = append(parts, renderHelpToken(helpToken{text: "[--flags]", kind: helpTokenDimmed}, styles))
	}
	return strings.Join(parts, "")
}

// helpUsageToken classifies one usage-line argument marker.
func helpUsageToken(token string) helpToken {
	token = strings.TrimSpace(token)
	switch {
	case token == "":
		return helpToken{text: token, kind: helpTokenWhitespace}
	case strings.HasPrefix(token, "[") || strings.HasPrefix(token, "<"):
		return helpToken{text: token, kind: helpTokenDimmed}
	default:
		return helpToken{text: token, kind: helpTokenArgument}
	}
}

// commandHasAnyFlags reports whether the command exposes any local or inherited visible flags.
func commandHasAnyFlags(cmd *cobra.Command) bool {
	if cmd == nil {
		return false
	}
	return cmd.HasFlags() || cmd.HasPersistentFlags() || cmd.HasAvailableFlags()
}

// renderHelpExamples styles each example line with shell-like token classes.
func renderHelpExamples(cmd *cobra.Command, styles helpStyles) []string {
	if cmd == nil || strings.TrimSpace(cmd.Example) == "" {
		return nil
	}
	lines := strings.Split(cmd.Example, "\n")
	rendered := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		rendered = append(rendered, renderHelpExampleLine(cmd, line, styles))
	}
	return rendered
}

// renderHelpExampleLine converts one example line into consistently colored segments.
func renderHelpExampleLine(cmd *cobra.Command, line string, styles helpStyles) string {
	segments := parseHelpExampleSegments(cmd, line)
	parts := make([]string, 0, len(segments))
	for _, segment := range segments {
		parts = append(parts, renderHelpToken(segment, styles))
	}
	return strings.Join(parts, "")
}

// parseHelpExampleSegments tokenizes one example line without treating `<placeholder>` values as redirects.
func parseHelpExampleSegments(cmd *cobra.Command, line string) []helpToken {
	line = strings.TrimSpace(line)
	if line == "" {
		return nil
	}
	if strings.HasPrefix(line, "# ") {
		return []helpToken{{text: line, kind: helpTokenComment}}
	}

	rootName := cmd.Root().Name()
	commandPath := strings.Fields(cmd.CommandPath())
	remainingCommands := append([]string(nil), commandPath...)
	if len(remainingCommands) > 0 {
		remainingCommands = remainingCommands[1:]
	}

	rawTokens := strings.Fields(line)
	segments := make([]helpToken, 0, len(rawTokens)*2)
	inQuoted := false
	for i, token := range rawTokens {
		if i > 0 {
			segments = append(segments, helpToken{text: " ", kind: helpTokenWhitespace})
		}

		if inQuoted {
			segments = append(segments, helpToken{text: token, kind: helpTokenQuoted})
			inQuoted = !tokenEndsQuote(token)
			continue
		}

		if tokenStartsQuote(token) {
			segments = append(segments, helpToken{text: token, kind: helpTokenQuoted})
			inQuoted = !tokenEndsQuote(token)
			continue
		}

		if i == 0 && token == rootName {
			segments = append(segments, helpToken{text: token, kind: helpTokenProgram})
			continue
		}
		if len(remainingCommands) > 0 && token == remainingCommands[0] {
			segments = append(segments, helpToken{text: token, kind: helpTokenCommand})
			remainingCommands = remainingCommands[1:]
			continue
		}
		if isShellOperatorToken(token) {
			segments = append(segments, helpToken{text: token, kind: helpTokenDimmed})
			continue
		}
		if strings.HasPrefix(token, "-") {
			name, value, hasValue := strings.Cut(token, "=")
			segments = append(segments, helpToken{text: name, kind: helpTokenFlag})
			if hasValue {
				segments = append(segments, helpToken{text: "=", kind: helpTokenDimmed})
				segments = append(segments, helpToken{text: value, kind: helpTokenArgument})
			}
			continue
		}
		segments = append(segments, helpToken{text: token, kind: helpTokenArgument})
	}
	return segments
}

// renderHelpToken applies the correct lipgloss style to one semantic token.
func renderHelpToken(token helpToken, styles helpStyles) string {
	switch token.kind {
	case helpTokenProgram:
		return styles.program.Render(token.text)
	case helpTokenCommand:
		return styles.command.Render(token.text)
	case helpTokenFlag:
		return styles.flag.Render(token.text)
	case helpTokenArgument:
		return styles.argument.Render(token.text)
	case helpTokenDimmed, helpTokenWhitespace:
		return styles.dimmed.Render(token.text)
	case helpTokenQuoted:
		return styles.quoted.Render(token.text)
	case helpTokenComment:
		return styles.comment.Render(token.text)
	default:
		return token.text
	}
}

// collectHelpCommandRows gathers visible subcommands for one help page.
func collectHelpCommandRows(cmd *cobra.Command) []helpRow {
	if cmd == nil {
		return nil
	}
	rows := make([]helpRow, 0, len(cmd.Commands()))
	for _, sub := range cmd.Commands() {
		if sub == nil || !sub.IsAvailableCommand() || sub.Hidden {
			continue
		}
		rows = append(rows, helpRow{
			key:         strings.TrimSpace(sub.Use),
			description: firstNonEmptyTrimmed(sub.Short, strings.TrimSpace(sub.Long)),
			keyKind:     helpTokenCommand,
		})
	}
	return rows
}

// collectHelpFlagRows gathers visible flags for one help page.
func collectHelpFlagRows(cmd *cobra.Command) []helpRow {
	if cmd == nil {
		return nil
	}
	flags := cmd.Flags()
	rows := make([]helpRow, 0)
	flags.VisitAll(func(flag *pflag.Flag) {
		if flag == nil || flag.Hidden {
			return
		}
		description := strings.TrimSpace(flag.Usage)
		if defaultText := renderHelpFlagDefault(flag); defaultText != "" {
			description += defaultText
		}
		rows = append(rows, helpRow{
			key:         formatHelpFlagLabel(flag),
			description: description,
			keyKind:     helpTokenFlag,
		})
	})
	return rows
}

// formatHelpFlagLabel renders one long/short flag label in the same order as Cobra help.
func formatHelpFlagLabel(flag *pflag.Flag) string {
	if flag == nil {
		return ""
	}
	if shorthand := strings.TrimSpace(flag.Shorthand); shorthand != "" {
		return "-" + shorthand + " --" + flag.Name
	}
	return "--" + flag.Name
}

// renderHelpFlagDefault appends human-readable flag defaults when they add real operator value.
func renderHelpFlagDefault(flag *pflag.Flag) string {
	if flag == nil {
		return ""
	}
	if strings.TrimSpace(flag.DefValue) == "" {
		return ""
	}
	if flag.Value != nil && flag.Value.Type() == "bool" && flag.DefValue == "false" {
		return ""
	}
	return " (" + flag.DefValue + ")"
}

// tokenStartsQuote reports whether a token opens a quoted string literal.
func tokenStartsQuote(token string) bool {
	return strings.HasPrefix(token, "\"") || strings.HasPrefix(token, "'")
}

// tokenEndsQuote reports whether a token closes a quoted string literal.
func tokenEndsQuote(token string) bool {
	return strings.HasSuffix(token, "\"") || strings.HasSuffix(token, "'")
}

// isShellOperatorToken reports whether a token is a shell operator that should stay dimmed.
func isShellOperatorToken(token string) bool {
	switch token {
	case "|", "||", "&", "&&", "\\", ">", "<", ">>", "1>", "2>", "2>>", "&>":
		return true
	default:
		return false
	}
}
