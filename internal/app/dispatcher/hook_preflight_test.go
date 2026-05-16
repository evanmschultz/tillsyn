package dispatcher

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/evanmschultz/tillsyn/internal/templates"
)

// agentsDir constructs the .tillsyn/agents/<group>/ path under a temp worktree
// and creates the directory, returning the group dir path.
func agentsDir(t *testing.T, worktree, group string) string {
	t.Helper()
	dir := filepath.Join(worktree, ".tillsyn", "agents", group)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir agents dir: %v", err)
	}
	return dir
}

// writeAgentFile writes body to <groupDir>/<agentName>.md.
func writeAgentFile(t *testing.T, groupDir, agentName, body string) {
	t.Helper()
	p := filepath.Join(groupDir, agentName+".md")
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatalf("write agent file: %v", err)
	}
}

// hooksDir constructs the .claude/hooks/ path under a temp worktree and
// creates the directory, returning the hooks dir path.
func hooksDir(t *testing.T, worktree string) string {
	t.Helper()
	dir := filepath.Join(worktree, ".claude", "hooks")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir hooks dir: %v", err)
	}
	return dir
}

// writeHookScript writes content to <hooksDir>/validate-action-item-paths.sh
// and marks it executable.
func writeHookScript(t *testing.T, hDir, content string) {
	t.Helper()
	p := filepath.Join(hDir, hookScriptName)
	if err := os.WriteFile(p, []byte(content), 0o755); err != nil {
		t.Fatalf("write hook script: %v", err)
	}
}

// syntheticFrontmatterWithHook is an agent frontmatter block that declares
// a validate-action-item-paths PreToolUse hook.
const syntheticFrontmatterWithHook = `---
name: builder-agent
description: test agent
hooks:
  PreToolUse:
    - matcher: "Edit|Write|Bash"
      hooks:
        - type: command
          command: "./.claude/hooks/validate-action-item-paths.sh"
---

# test body content that is longer than 200 characters to satisfy the validator.
This is placeholder content used only in hook_preflight tests and must be
at least two hundred characters long to pass the bundle shape check. Adding
more padding here to ensure we exceed the minimum length requirement easily.
`

// syntheticFrontmatterNoHook is an agent frontmatter block with no hooks block.
const syntheticFrontmatterNoHook = `---
name: planning-agent
description: test agent without hooks
---

# test body
`

// TestCheckHookArtifactsCase1AgentFileMissing verifies Case 1: when no agent
// file exists in the project tier the function returns nil and does NOT error.
// The dispatcher cannot block a spawn on behalf of a non-existent opt-in signal.
func TestCheckHookArtifactsCase1AgentFileMissing(t *testing.T) {
	worktree := t.TempDir()
	// No agent file written.
	err := CheckHookArtifacts(worktree, "builder-agent")
	if err != nil {
		t.Fatalf("CheckHookArtifacts: expected nil when agent file missing, got: %v", err)
	}
}

// TestCheckHookArtifactsCase2NoHooksBlock verifies Case 2: when the agent file
// exists but declares no hooks: block the function returns nil (agent has not
// opted into the isolation hook).
func TestCheckHookArtifactsCase2NoHooksBlock(t *testing.T) {
	worktree := t.TempDir()
	dir := agentsDir(t, worktree, "go")
	writeAgentFile(t, dir, "planning-agent", syntheticFrontmatterNoHook)

	err := CheckHookArtifacts(worktree, "planning-agent")
	if err != nil {
		t.Fatalf("CheckHookArtifacts: expected nil when no hooks block, got: %v", err)
	}
}

// TestCheckHookArtifactsCase3ScriptMissing verifies Case 3: when the agent
// declares the hook but the script is absent from .claude/hooks/ the function
// returns wrapped ErrHookArtifactStale.
func TestCheckHookArtifactsCase3ScriptMissing(t *testing.T) {
	worktree := t.TempDir()
	dir := agentsDir(t, worktree, "go")
	writeAgentFile(t, dir, "builder-agent", syntheticFrontmatterWithHook)
	// No hook script written.

	err := CheckHookArtifacts(worktree, "builder-agent")
	if err == nil {
		t.Fatal("CheckHookArtifacts: expected ErrHookArtifactStale when script missing, got nil")
	}
	if !errors.Is(err, ErrHookArtifactStale) {
		t.Fatalf("CheckHookArtifacts: expected errors.Is(err, ErrHookArtifactStale); got: %v", err)
	}
}

// TestCheckHookArtifactsCase4NoHashHeader verifies Case 4: when the agent
// declares the hook, the script exists, but the script has no
// tillsyn-hook-hash header line the function returns wrapped ErrHookArtifactStale.
func TestCheckHookArtifactsCase4NoHashHeader(t *testing.T) {
	worktree := t.TempDir()
	dir := agentsDir(t, worktree, "go")
	writeAgentFile(t, dir, "builder-agent", syntheticFrontmatterWithHook)
	hDir := hooksDir(t, worktree)
	writeHookScript(t, hDir, "#!/usr/bin/env bash\n# no hash header here\necho ok\n")

	err := CheckHookArtifacts(worktree, "builder-agent")
	if err == nil {
		t.Fatal("CheckHookArtifacts: expected ErrHookArtifactStale when hash header missing, got nil")
	}
	if !errors.Is(err, ErrHookArtifactStale) {
		t.Fatalf("CheckHookArtifacts: expected errors.Is(err, ErrHookArtifactStale); got: %v", err)
	}
}

// TestCheckHookArtifactsCase5MalformedHash verifies Case 5: when the agent
// declares the hook, the script exists, but the hash header value is not 64
// lowercase hex characters the function returns wrapped ErrHookArtifactStale.
func TestCheckHookArtifactsCase5MalformedHash(t *testing.T) {
	worktree := t.TempDir()
	dir := agentsDir(t, worktree, "go")
	writeAgentFile(t, dir, "builder-agent", syntheticFrontmatterWithHook)
	hDir := hooksDir(t, worktree)
	// Hash is only 16 chars — not 64.
	writeHookScript(t, hDir, "#!/usr/bin/env bash\n# tillsyn-hook-hash: deadbeefdeadbeef\necho ok\n")

	err := CheckHookArtifacts(worktree, "builder-agent")
	if err == nil {
		t.Fatal("CheckHookArtifacts: expected ErrHookArtifactStale when hash malformed, got nil")
	}
	if !errors.Is(err, ErrHookArtifactStale) {
		t.Fatalf("CheckHookArtifacts: expected errors.Is(err, ErrHookArtifactStale); got: %v", err)
	}
}

// TestCheckHookArtifactsCase6HashMismatch verifies Case 6: when the agent
// declares the hook, the script exists with a properly-formatted hash header,
// but the hash does not match templates.ComputeHookHash() the function returns
// wrapped ErrHookArtifactStale.
func TestCheckHookArtifactsCase6HashMismatch(t *testing.T) {
	worktree := t.TempDir()
	dir := agentsDir(t, worktree, "go")
	writeAgentFile(t, dir, "builder-agent", syntheticFrontmatterWithHook)
	hDir := hooksDir(t, worktree)
	// A valid-format hash that is deliberately wrong (all zeros).
	staleHash := "0000000000000000000000000000000000000000000000000000000000000000"
	writeHookScript(t, hDir, fmt.Sprintf(
		"#!/usr/bin/env bash\n# tillsyn-hook-hash: %s\necho ok\n", staleHash,
	))

	err := CheckHookArtifacts(worktree, "builder-agent")
	if err == nil {
		t.Fatal("CheckHookArtifacts: expected ErrHookArtifactStale when hash mismatches, got nil")
	}
	if !errors.Is(err, ErrHookArtifactStale) {
		t.Fatalf("CheckHookArtifacts: expected errors.Is(err, ErrHookArtifactStale); got: %v", err)
	}
}

// TestCheckHookArtifactsCase7HashMatches verifies Case 7: when the agent
// declares the hook, the script exists, and the script hash matches
// templates.ComputeHookHash() the function returns nil (all checks pass).
func TestCheckHookArtifactsCase7HashMatches(t *testing.T) {
	worktree := t.TempDir()
	dir := agentsDir(t, worktree, "go")
	writeAgentFile(t, dir, "builder-agent", syntheticFrontmatterWithHook)
	hDir := hooksDir(t, worktree)

	embeddedHash, err := templates.ComputeHookHash()
	if err != nil {
		t.Fatalf("templates.ComputeHookHash(): %v", err)
	}
	writeHookScript(t, hDir, fmt.Sprintf(
		"#!/usr/bin/env bash\n# tillsyn-hook-hash: %s\necho ok\n", embeddedHash,
	))

	err = CheckHookArtifacts(worktree, "builder-agent")
	if err != nil {
		t.Fatalf("CheckHookArtifacts: expected nil when hash matches, got: %v", err)
	}
}

// TestCheckHookArtifactsCase8MalformedYAML verifies Case 8: when the agent
// file exists with malformed YAML frontmatter the function returns nil and
// does NOT block the spawn. Parse failures are graceful-skip events, not
// spawn-blockers.
func TestCheckHookArtifactsCase8MalformedYAML(t *testing.T) {
	worktree := t.TempDir()
	dir := agentsDir(t, worktree, "go")
	// Frontmatter with invalid YAML (unquoted colon in value without space).
	malformed := "---\nname: builder-agent\nhooks:\n  bad: [unclosed\n---\n\n# body\n"
	writeAgentFile(t, dir, "builder-agent", malformed)

	err := CheckHookArtifacts(worktree, "builder-agent")
	if err != nil {
		t.Fatalf("CheckHookArtifacts: expected nil on malformed YAML frontmatter, got: %v", err)
	}
}

// TestCheckHookArtifactsCase9NoFrontmatter verifies Case 9: when the agent
// file exists but has no frontmatter delimiters at all the function returns
// nil (agent does not opt in). A plain-body agent without frontmatter simply
// does not declare any hooks.
func TestCheckHookArtifactsCase9NoFrontmatter(t *testing.T) {
	worktree := t.TempDir()
	dir := agentsDir(t, worktree, "go")
	writeAgentFile(t, dir, "builder-agent", "# No frontmatter here\n\nJust a plain agent body.\n")

	err := CheckHookArtifacts(worktree, "builder-agent")
	if err != nil {
		t.Fatalf("CheckHookArtifacts: expected nil when no frontmatter, got: %v", err)
	}
}

// TestCheckHookArtifactsEmptyInputsReturnNil verifies that empty worktreePath
// or agentName produce a nil return without panicking. These cases arise when
// BuildSpawnCommand is invoked with incomplete domain.Project values.
func TestCheckHookArtifactsEmptyInputsReturnNil(t *testing.T) {
	if err := CheckHookArtifacts("", "builder-agent"); err != nil {
		t.Fatalf("CheckHookArtifacts(empty worktree): expected nil, got: %v", err)
	}
	if err := CheckHookArtifacts("/some/worktree", ""); err != nil {
		t.Fatalf("CheckHookArtifacts(empty agentName): expected nil, got: %v", err)
	}
}

// TestCheckHookArtifactsMultipleGroupsFindsFirst verifies that when multiple
// group subdirectories exist under .tillsyn/agents/ the search finds the
// agent file in the first matching group and applies the hook check to it.
// The test installs the hook script with the correct hash so a match in any
// group produces nil.
func TestCheckHookArtifactsMultipleGroupsFindsFirst(t *testing.T) {
	worktree := t.TempDir()
	// Create two groups; only the second one has the agent file.
	_ = agentsDir(t, worktree, "gen")
	dir := agentsDir(t, worktree, "go")
	writeAgentFile(t, dir, "builder-agent", syntheticFrontmatterWithHook)

	embeddedHash, err := templates.ComputeHookHash()
	if err != nil {
		t.Fatalf("templates.ComputeHookHash(): %v", err)
	}
	hDir := hooksDir(t, worktree)
	writeHookScript(t, hDir, fmt.Sprintf(
		"#!/usr/bin/env bash\n# tillsyn-hook-hash: %s\necho ok\n", embeddedHash,
	))

	err = CheckHookArtifacts(worktree, "builder-agent")
	if err != nil {
		t.Fatalf("CheckHookArtifacts with multiple groups: expected nil, got: %v", err)
	}
}
