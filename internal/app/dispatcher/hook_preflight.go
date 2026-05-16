package dispatcher

import (
	"bufio"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/charmbracelet/log"
	"gopkg.in/yaml.v3"

	"github.com/evanmschultz/tillsyn/internal/templates"
)

// ErrHookArtifactStale is returned by CheckHookArtifacts when the bound
// agent declares a validate-action-item-paths PreToolUse hook in its YAML
// frontmatter but the rendered on-disk script is missing, lacks the expected
// hash header, or its hash does not match the embedded template hash from
// templates.ComputeHookHash.
//
// Callers detect this via errors.Is and route to a "run 'till init'" user
// message rather than treating the failure as a transient spawn error.
var ErrHookArtifactStale = errors.New("tillsyn hook artifacts stale; run 'till init' to refresh")

// hookScriptName is the basename of the PreToolUse hook script that
// CheckHookArtifacts validates. The script lives at
// <worktreePath>/.claude/hooks/<hookScriptName>.
const hookScriptName = "validate-action-item-paths.sh"

// hookScriptRelPath is the path of the hook script relative to the worktree
// root. Mirrors the command value declared in agent frontmatter:
// "./.claude/hooks/validate-action-item-paths.sh".
const hookScriptRelPath = ".claude/hooks/" + hookScriptName

// hookHashHeaderRe matches the hash header line in the rendered hook script.
// Format (per the .sh.tmpl template): `# tillsyn-hook-hash: <64-hex-chars>`.
// The regex anchors the full line so partial matches or additional content
// after the hash produce a nil match rather than a false positive.
var hookHashHeaderRe = regexp.MustCompile(`(?m)^#\s+tillsyn-hook-hash:\s+([0-9a-f]{64})\s*$`)

// agentFrontmatter is the typed subset of an agent .md file's YAML
// frontmatter that CheckHookArtifacts inspects. Only the `hooks.PreToolUse`
// block is needed; unknown keys are silently ignored by the yaml.v3 decoder.
type agentFrontmatter struct {
	Hooks agentHooks `yaml:"hooks"`
}

// agentHooks holds the per-phase hook declarations in the agent frontmatter.
type agentHooks struct {
	PreToolUse []agentPreToolUseEntry `yaml:"PreToolUse"`
}

// agentPreToolUseEntry is one entry in the hooks.PreToolUse list. Each entry
// may declare multiple hooks; CheckHookArtifacts searches the nested hooks
// slice for any command containing validate-action-item-paths.sh.
type agentPreToolUseEntry struct {
	Matcher string          `yaml:"matcher"`
	Hooks   []agentHookCmd  `yaml:"hooks"`
}

// agentHookCmd is one hook declaration inside a PreToolUse entry.
type agentHookCmd struct {
	Type    string `yaml:"type"`
	Command string `yaml:"command"`
}

// CheckHookArtifacts verifies that if the bound agent declares a PreToolUse
// validate-action-item-paths hook in its YAML frontmatter, the rendered
// script exists at <worktreePath>/.claude/hooks/ and its hash matches the
// embedded template via templates.ComputeHookHash.
//
// Behavior:
//   - Agent file missing OR no hooks.PreToolUse declaring validate-action-item-paths.sh
//     in frontmatter -> return nil + log warning/info (agent does not opt in).
//   - hooks declared + script missing -> return wrapped ErrHookArtifactStale.
//   - hooks declared + hash header missing/malformed -> wrap ErrHookArtifactStale.
//   - hooks declared + hash mismatches templates.ComputeHookHash() -> wrap ErrHookArtifactStale.
//   - hooks declared + hash matches -> return nil.
//   - Malformed YAML frontmatter -> return nil + warning (graceful skip;
//     parse errors are not spawn-blockers).
//   - No frontmatter block at all -> return nil + info.
//
// Agent file resolution: CheckHookArtifacts walks <worktreePath>/.tillsyn/agents/
// looking for any <group>/<agentName>.md file. The first match is used.
// This mirrors the project-tier of the 3-tier resolver in the render package
// without requiring a group parameter at the callsite.
//
// Closes the silent-disable counterexample: a user who deletes the hook
// script while the agent's frontmatter still declares it sees a clear
// ErrHookArtifactStale failure from the dispatcher rather than silent
// isolation loss. Mirrors the package-level hook-seam pattern from
// plugin_preflight.go.
func CheckHookArtifacts(worktreePath string, agentName string) error {
	if strings.TrimSpace(worktreePath) == "" || strings.TrimSpace(agentName) == "" {
		return nil
	}

	agentBody, found, err := findProjectTierAgent(worktreePath, agentName)
	if err != nil {
		log.Warn("hook preflight: could not read agent file; skipping hook validation",
			"agent", agentName, "err", err)
		return nil
	}
	if !found {
		log.Info("hook preflight: agent file not found in project tier; skipping hook validation",
			"agent", agentName, "worktree", worktreePath)
		return nil
	}

	declares, err := agentDeclaresHook(agentBody)
	if err != nil {
		// Malformed YAML frontmatter: graceful skip per acceptance criterion 3.
		log.Warn("hook preflight: malformed YAML frontmatter in agent file; skipping hook validation",
			"agent", agentName, "err", err)
		return nil
	}
	if !declares {
		log.Info("hook preflight: agent does not declare validate-action-item-paths hook; skipping",
			"agent", agentName)
		return nil
	}

	// Agent has opted in. Now verify the on-disk script.
	scriptPath := filepath.Join(worktreePath, hookScriptRelPath)
	scriptData, err := os.ReadFile(scriptPath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("%w: %s not found at %s",
				ErrHookArtifactStale, hookScriptName, scriptPath)
		}
		return fmt.Errorf("%w: reading hook script %s: %v",
			ErrHookArtifactStale, scriptPath, err)
	}

	onDiskHash, ok := extractHookHash(string(scriptData))
	if !ok {
		return fmt.Errorf("%w: %s missing or malformed tillsyn-hook-hash header",
			ErrHookArtifactStale, hookScriptName)
	}

	embeddedHash, err := templates.ComputeHookHash()
	if err != nil {
		return fmt.Errorf("hook preflight: compute embedded hook hash: %w", err)
	}

	if onDiskHash != embeddedHash {
		return fmt.Errorf("%w: %s hash mismatch (on-disk %s... != embedded %s...)",
			ErrHookArtifactStale, hookScriptName, onDiskHash[:8], embeddedHash[:8])
	}

	return nil
}

// findProjectTierAgent walks <worktreePath>/.tillsyn/agents/<group>/<agentName>.md
// for each group subdirectory under the project agents dir, returning the
// file body of the first match. Returns ("", false, nil) when the directory
// does not exist or no match is found. Returns ("", false, err) on I/O
// errors other than fs.ErrNotExist.
func findProjectTierAgent(worktreePath, agentName string) (string, bool, error) {
	agentsRoot := filepath.Join(worktreePath, ".tillsyn", "agents")

	entries, err := os.ReadDir(agentsRoot)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return "", false, nil
		}
		return "", false, err
	}

	basename := agentName + ".md"
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		candidate := filepath.Join(agentsRoot, e.Name(), basename)
		data, readErr := os.ReadFile(candidate)
		if readErr != nil {
			if errors.Is(readErr, fs.ErrNotExist) {
				continue
			}
			return "", false, readErr
		}
		return string(data), true, nil
	}
	return "", false, nil
}

// agentDeclaresHook reports whether the agent file body (raw markdown with
// YAML frontmatter) declares a hooks.PreToolUse entry whose command contains
// validate-action-item-paths.sh as a substring. Returns (false, nil) when no
// frontmatter block is present or when the hooks block is absent.
// Returns (false, err) when the frontmatter YAML is malformed.
func agentDeclaresHook(body string) (bool, error) {
	fm, found := extractFrontmatter(body)
	if !found {
		// No frontmatter block: agent does not opt in.
		return false, nil
	}

	var parsed agentFrontmatter
	if err := yaml.Unmarshal([]byte(fm), &parsed); err != nil {
		return false, err
	}

	for _, entry := range parsed.Hooks.PreToolUse {
		for _, h := range entry.Hooks {
			if strings.Contains(h.Command, hookScriptName) {
				return true, nil
			}
		}
	}
	return false, nil
}

// extractFrontmatter extracts the YAML content between the first two `---`
// lines in body. Returns (content, true) on success, ("", false) when the
// body does not contain a valid frontmatter block (no leading `---` line or
// no closing `---` line).
//
// The delimiters themselves are not included in the returned content. The
// function treats bare `---` at line start as the delimiter, ignoring
// trailing whitespace on the line to handle CRLF files and editors that
// append a trailing space.
func extractFrontmatter(body string) (string, bool) {
	sc := bufio.NewScanner(strings.NewReader(body))

	// Consume lines until we find the opening `---` delimiter.
	var openFound bool
	for sc.Scan() {
		if strings.TrimRight(sc.Text(), " \t\r") == "---" {
			openFound = true
			break
		}
	}
	if !openFound {
		return "", false
	}

	// Collect lines until the closing `---` delimiter.
	var buf strings.Builder
	for sc.Scan() {
		line := sc.Text()
		if strings.TrimRight(line, " \t\r") == "---" {
			return buf.String(), true
		}
		buf.WriteString(line)
		buf.WriteByte('\n')
	}
	// Closing delimiter not found.
	return "", false
}

// extractHookHash reads the tillsyn-hook-hash header from the script content
// and returns (hash, true) on success or ("", false) when the header is
// absent or does not match the expected format (64 lowercase hex chars).
// The search scans the first 10 lines of the script for efficiency.
func extractHookHash(content string) (string, bool) {
	// Scan only the header region of the script (first 10 lines).
	header := firstNLines(content, 10)
	matches := hookHashHeaderRe.FindStringSubmatch(header)
	if len(matches) < 2 {
		return "", false
	}
	return matches[1], true
}

// firstNLines returns the first n lines of s joined with newlines. If s has
// fewer than n lines the entire content is returned. Used to bound the regex
// scan to the script header region.
func firstNLines(s string, n int) string {
	var buf strings.Builder
	sc := bufio.NewScanner(strings.NewReader(s))
	i := 0
	for sc.Scan() && i < n {
		buf.WriteString(sc.Text())
		buf.WriteByte('\n')
		i++
	}
	return buf.String()
}
