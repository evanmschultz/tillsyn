package dispatcher

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// agent_definition.go parses `.claude/agents/ta-*.md` files into the
// in-memory AgentDefinition representation that downstream Lane C droplets
// (D2.5 dispatch tool conversion, D4 spawn wiring, D7 routing) consume. The
// canonical conversion contract for downstream codex / ollama backends lives
// at ~/.claude/codex-mcp-dispatch-tool-conversion.md.
//
// The parser is intentionally small: YAML frontmatter via gopkg.in/yaml.v3,
// frontmatter delimiter scan via the existing package-private
// extractFrontmatter helper from hook_preflight.go (reused for DRY), and
// filename-derived role/axis/language classification via a single regex.
//
// Drop 4d Wave 0 directive: the `model` field encodes dev's routing thesis
// per project_multi_backend_dogfood_direction.md memory:
//
//   - builders (kind=build, role=builder)          -> haiku  (cheap implementation)
//   - build-QA (kind=build-qa-*, role=qa-*)        -> sonnet (specialist verification)
//   - plan-QA  (kind=plan-qa-*, role=qa-*)         -> opus   (decomposition critique)
//   - planners (kind=plan, role=planner)           -> sonnet (decomposition synthesis)
//   - closeout (kind=closeout, role=closeout)      -> haiku  (wrap-up only)
//
// Empty `model` in the parsed AgentDefinition means "field absent from
// frontmatter"; the routing layer (D4 / D7) treats absence as
// "inherit-from-template-default", not as a hard fail. The parser only
// surfaces the value verbatim.

// AgentDefinition is the in-memory representation of one `.claude/agents/ta-*.md`
// file. Fields Name, Description, and Tools come straight from the YAML
// frontmatter. SystemPrompt is the markdown body after the closing `---`.
// Role, Axis, and Language are derived from the filename pattern by
// classifyAgentName.
type AgentDefinition struct {
	// Name is the agent identifier from the `name:` frontmatter key. Must
	// match the basename (sans .md) of the source file.
	Name string

	// Description is the one-line summary from the `description:` frontmatter
	// key. Used as the spawn-time hint surfaced to humans.
	Description string

	// Model is the routing-target model from the `model:` frontmatter key.
	// Empty string means absent — the routing layer falls back to template
	// defaults. Known values today: "haiku", "sonnet", "opus".
	Model string

	// Tools is the parsed comma-separated `tools:` frontmatter line, split
	// on commas and trimmed of surrounding whitespace. Empty list when the
	// field is absent or whitespace-only.
	Tools []string

	// SystemPrompt is the markdown body following the closing `---`
	// delimiter. Leading newlines are preserved verbatim so downstream
	// renderers can choose to trim or not.
	SystemPrompt string

	// Role is derived from the filename: "planner" | "builder" |
	// "qa-proof" | "qa-falsification" | "closeout".
	Role string

	// Axis is derived from the filename: "plan" | "build" | "none". QA
	// personas carry "plan" or "build" per their plan-axis vs build-axis
	// split. Planning + builder carry their own axis. Closeout = "none".
	Axis string

	// Language is derived from the filename: "go" | "fe" | "none".
	// ta-closeout has Language="none" because closeout runs on both lanes.
	Language string
}

// ErrInvalidAgentName is returned by ParseAgentDefinition when the supplied
// filename does not match the canonical
// `ta-{go|fe}-(planning|builder|plan-qa-(proof|falsification)|build-qa-(proof|falsification))`
// or `ta-closeout` shape. Callers detect this via errors.Is.
var ErrInvalidAgentName = errors.New("dispatcher: invalid agent name")

// ErrMalformedFrontmatter is returned by ParseAgentDefinition when the file
// is missing its leading `---` block, the closing `---` is absent, or the
// frontmatter content fails to parse as YAML. Callers detect this via
// errors.Is.
var ErrMalformedFrontmatter = errors.New("dispatcher: malformed agent frontmatter")

// agentFilenameRe captures the canonical filename pattern for the
// 13 ta-* agent personas. The single capture group is the persona suffix
// (everything after `ta-`). The closeout persona is matched separately
// because it has no language axis.
var agentFilenameRe = regexp.MustCompile(`^ta-(go|fe)-(planning|builder|plan-qa-proof|plan-qa-falsification|build-qa-proof|build-qa-falsification)$`)

// agentFrontmatterFields is the typed subset of the YAML frontmatter that
// ParseAgentDefinition decodes. Unknown keys (e.g. the `hooks:` block
// consumed by hook_preflight.go) are silently ignored by yaml.v3.
type agentFrontmatterFields struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	Model       string `yaml:"model"`
	Tools       string `yaml:"tools"`
}

// ParseAgentDefinition parses one `.claude/agents/ta-*.md` file body into
// an AgentDefinition. The filename argument is the basename without .md
// (e.g. "ta-go-builder"); body is the full file content including
// frontmatter.
//
// Returns ErrInvalidAgentName if filename does not match the canonical
// pattern, ErrMalformedFrontmatter if the YAML block is missing /
// unterminated / unparseable.
func ParseAgentDefinition(filename string, body []byte) (AgentDefinition, error) {
	role, axis, language, err := classifyAgentName(filename)
	if err != nil {
		return AgentDefinition{}, err
	}

	fmContent, found := extractFrontmatter(string(body))
	if !found {
		return AgentDefinition{}, fmt.Errorf("%w: %s missing frontmatter delimiters",
			ErrMalformedFrontmatter, filename)
	}

	var parsed agentFrontmatterFields
	if err := yaml.Unmarshal([]byte(fmContent), &parsed); err != nil {
		return AgentDefinition{}, fmt.Errorf("%w: %s: %v",
			ErrMalformedFrontmatter, filename, err)
	}

	systemPrompt := bodyAfterFrontmatter(string(body))

	return AgentDefinition{
		Name:         strings.TrimSpace(parsed.Name),
		Description:  strings.TrimSpace(parsed.Description),
		Model:        strings.TrimSpace(parsed.Model),
		Tools:        splitToolsField(parsed.Tools),
		SystemPrompt: systemPrompt,
		Role:         role,
		Axis:         axis,
		Language:     language,
	}, nil
}

// LoadAgentDefinition reads the file at path and parses it via
// ParseAgentDefinition. The filename component (basename sans .md) drives
// role/axis/language derivation. Returns the underlying os.ReadFile error
// verbatim on I/O failure; otherwise the same error contract as
// ParseAgentDefinition.
func LoadAgentDefinition(path string) (AgentDefinition, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return AgentDefinition{}, fmt.Errorf("dispatcher: read agent file %s: %w", path, err)
	}

	base := filepath.Base(path)
	name := strings.TrimSuffix(base, filepath.Ext(base))
	return ParseAgentDefinition(name, data)
}

// classifyAgentName derives the (role, axis, language) triple from the
// agent filename basename (e.g. "ta-go-builder" -> ("builder", "build",
// "go")). Returns ErrInvalidAgentName for any filename that does not match
// the canonical pattern.
//
// Classification table:
//
//	ta-closeout                       -> (closeout,         none,  none)
//	ta-{go|fe}-planning               -> (planner,          plan,  go|fe)
//	ta-{go|fe}-builder                -> (builder,          build, go|fe)
//	ta-{go|fe}-plan-qa-proof          -> (qa-proof,         plan,  go|fe)
//	ta-{go|fe}-plan-qa-falsification  -> (qa-falsification, plan,  go|fe)
//	ta-{go|fe}-build-qa-proof         -> (qa-proof,         build, go|fe)
//	ta-{go|fe}-build-qa-falsification -> (qa-falsification, build, go|fe)
func classifyAgentName(filename string) (role, axis, language string, err error) {
	if filename == "ta-closeout" {
		return "closeout", "none", "none", nil
	}

	matches := agentFilenameRe.FindStringSubmatch(filename)
	if len(matches) != 3 {
		return "", "", "", fmt.Errorf("%w: %q does not match ta-{go|fe}-(planning|builder|plan-qa-(proof|falsification)|build-qa-(proof|falsification)) or ta-closeout",
			ErrInvalidAgentName, filename)
	}

	language = matches[1]
	suffix := matches[2]

	switch suffix {
	case "planning":
		return "planner", "plan", language, nil
	case "builder":
		return "builder", "build", language, nil
	case "plan-qa-proof":
		return "qa-proof", "plan", language, nil
	case "plan-qa-falsification":
		return "qa-falsification", "plan", language, nil
	case "build-qa-proof":
		return "qa-proof", "build", language, nil
	case "build-qa-falsification":
		return "qa-falsification", "build", language, nil
	default:
		// Unreachable given the regex above; defense-in-depth.
		return "", "", "", fmt.Errorf("%w: unrecognized suffix %q", ErrInvalidAgentName, suffix)
	}
}

// splitToolsField parses the `tools:` frontmatter line, which is a
// comma-separated string (NOT a YAML list) per the existing
// .claude/agents/ta-*.md convention. Returns nil when raw is empty or
// whitespace-only.
func splitToolsField(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		t := strings.TrimSpace(p)
		if t == "" {
			continue
		}
		out = append(out, t)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// bodyAfterFrontmatter returns everything after the closing `---`
// delimiter, preserving the inter-delimiter newline. When no closing
// delimiter is found the empty string is returned; the malformed-
// frontmatter case is handled upstream by ParseAgentDefinition.
func bodyAfterFrontmatter(body string) string {
	// First `---\n` opens the block. We need the close.
	// Reuse the scanner-driven approach by tracking the delimiter line
	// positions in raw bytes, since extractFrontmatter returns only the
	// inner content.
	lines := strings.SplitAfter(body, "\n")
	openIdx := -1
	closeIdx := -1
	for i, line := range lines {
		trimmed := strings.TrimRight(line, " \t\r\n")
		if trimmed != "---" {
			continue
		}
		if openIdx == -1 {
			openIdx = i
			continue
		}
		closeIdx = i
		break
	}
	if openIdx == -1 || closeIdx == -1 || closeIdx+1 >= len(lines) {
		return ""
	}
	return strings.Join(lines[closeIdx+1:], "")
}
