//go:build integration

package templates

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"testing"
)

// hookScriptPath writes the embedded validate-action-item-paths.sh.tmpl to a
// temporary file and returns the path. The __HASH__ placeholder in the header
// comment is left as-is; the script does not inspect its own hash, so no
// substitution is needed for behavioral testing (option b per droplet spec).
func hookScriptPath(t *testing.T) string {
	t.Helper()
	data, err := DefaultTemplateFS.ReadFile("builtin/hooks/validate-action-item-paths.sh.tmpl")
	if err != nil {
		t.Fatalf("hookScriptPath: read embedded template: %v", err)
	}
	f, err := os.CreateTemp(t.TempDir(), "validate-action-item-paths-*.sh")
	if err != nil {
		t.Fatalf("hookScriptPath: create temp file: %v", err)
	}
	if _, err := f.Write(data); err != nil {
		_ = f.Close()
		t.Fatalf("hookScriptPath: write script: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("hookScriptPath: close temp file: %v", err)
	}
	if err := os.Chmod(f.Name(), 0o755); err != nil {
		t.Fatalf("hookScriptPath: chmod script: %v", err)
	}
	return f.Name()
}

// tillsyn stub script content.  Emits {"Paths": ["only/this/dir/"]} for any
// invocation, matching the Go json.Marshal field-name convention (.Paths not
// .paths).  The stub ignores all arguments.
const stubBinScript = `#!/bin/sh
printf '{"Paths": ["only/this/dir/"]}'
`

// tillBinStub writes the stub binary to a temp dir and returns its path.
func tillBinStub(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	path := dir + "/till-stub"
	if err := os.WriteFile(path, []byte(stubBinScript), 0o755); err != nil {
		t.Fatalf("tillBinStub: write stub: %v", err)
	}
	return path
}

// hookEnv returns a baseline environment map with all required variables set
// for normal (non-degraded) test execution.
func hookEnv(tillBin, actionItemID string) []string {
	return []string{
		"HOME=/tmp",
		"TILLSYN_BIN=" + tillBin,
		"TILLSYN_ACTION_ITEM_ID=" + actionItemID,
		"PATH=" + os.Getenv("PATH"), // inherit PATH so jq is reachable
	}
}

// runHook executes the hook script with the given stdin payload and environment
// overrides.  It returns the exit code and combined stderr output.
func runHook(t *testing.T, scriptPath, payload string, env []string) (exitCode int, stderr string) {
	t.Helper()
	cmd := exec.Command("bash", scriptPath)
	cmd.Stdin = strings.NewReader(payload)
	cmd.Env = env

	var stderrBuf strings.Builder
	cmd.Stderr = &stderrBuf

	// Stdout is discarded; the hook only writes diagnostic messages to stderr.
	cmd.Stdout = io.Discard

	err := cmd.Run()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return exitErr.ExitCode(), stderrBuf.String()
		}
		// Unexpected error (e.g. script not found).
		t.Fatalf("runHook: exec.Command failed unexpectedly: %v", err)
	}
	return 0, stderrBuf.String()
}

// toolPayload builds a minimal PreToolUse JSON payload for Edit / Write tools.
func toolPayload(toolName, filePath string) string {
	return fmt.Sprintf(
		`{"tool_name":%q,"tool_input":{"file_path":%q}}`,
		toolName, filePath,
	)
}

// bashPayload builds a minimal PreToolUse JSON payload for Bash tools.
func bashPayload(command string) string {
	return fmt.Sprintf(
		`{"tool_name":"Bash","tool_input":{"command":%q}}`,
		command,
	)
}

// TestHookIntegration_EnforcementCases verifies that the hook allows in-scope
// paths (exit 0) and blocks out-of-scope paths (exit 2) for Edit, Write, and
// Bash tool calls.  Each sub-test follows the red-green protocol: the assertion
// is the specification; the implementation is the hook script.
func TestHookIntegration_EnforcementCases(t *testing.T) {
	t.Parallel()

	// Skip the entire suite if jq is not available.  Without jq the hook
	// soft-skips to exit 0, making out-of-path assertions vacuously wrong.
	if _, err := exec.LookPath("jq"); err != nil {
		t.Skip("jq not on PATH; enforcement cases require jq for path validation")
	}

	script := hookScriptPath(t)
	stub := tillBinStub(t)
	const actionItemID = "00000000-0000-0000-0000-000000000001"
	env := hookEnv(stub, actionItemID)

	cases := []struct {
		name     string
		payload  string
		wantExit int
	}{
		// Case 1: Edit within declared path prefix — allow.
		{
			name:     "in-path Edit",
			payload:  toolPayload("Edit", "only/this/dir/foo.go"),
			wantExit: 0,
		},
		// Case 2: Edit outside declared path — block.
		{
			name:     "out-of-path Edit",
			payload:  toolPayload("Edit", "another/dir/bar.go"),
			wantExit: 2,
		},
		// Case 3: Write outside declared path — block.
		{
			name:     "out-of-path Write",
			payload:  toolPayload("Write", "yet/another/dir/baz.go"),
			wantExit: 2,
		},
		// Case 4: Bash git checkout outside declared path — block.
		{
			name:     "Bash git checkout out-of-path",
			payload:  bashPayload("git checkout -- another/dir/bar.go"),
			wantExit: 2,
		},
		// Case 5: Bash git restore outside declared path — block.
		{
			name:     "Bash git restore out-of-path",
			payload:  bashPayload("git restore another/dir/bar.go"),
			wantExit: 2,
		},
		// Case 6: Bash rm outside declared path — block.
		{
			name:     "Bash rm out-of-path",
			payload:  bashPayload("rm another/dir/bar.go"),
			wantExit: 2,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, _ := runHook(t, script, tc.payload, env)
			if got != tc.wantExit {
				t.Errorf("exit code = %d; want %d (payload: %s)", got, tc.wantExit, tc.payload)
			}
		})
	}
}

// TestHookIntegration_KnownBypassCases documents known bypass patterns that
// the hook intentionally does NOT block.  These are accepted out-of-scope
// defense-in-depth gaps; future hardening is welcome but not required by this
// droplet.
func TestHookIntegration_KnownBypassCases(t *testing.T) {
	t.Parallel()

	if _, err := exec.LookPath("jq"); err != nil {
		t.Skip("jq not on PATH; skipping known-bypass cases")
	}

	script := hookScriptPath(t)
	stub := tillBinStub(t)
	const actionItemID = "00000000-0000-0000-0000-000000000001"
	env := hookEnv(stub, actionItemID)

	// known-bypass cases accepted as out-of-scope defense-in-depth gaps; future hardening welcome
	//
	// The hook tokenises the command string on whitespace and only watches for
	// git / rm / mv as top-level tokens.  Any wrapping via an unwatched shell
	// primitive (eval, bash -c, sh -c, etc.) hides the dangerous sub-command
	// from the tokeniser entirely, producing exit 0.  Detecting these patterns
	// would require full shell-grammar parsing, which is out of scope for this
	// defence-in-depth hook.
	cases := []struct {
		name    string
		payload string
	}{
		// Case 7: eval wrapping — tokeniser sees "eval" which is not a watched command.
		{
			name:    "eval bypass",
			payload: bashPayload("eval 'git checkout -- another/dir/bar.go'"),
		},
		// Case 8: bash -c wrapping — "bash" is not a watched token; the inner
		// command string is opaque to the hook's whitespace tokeniser.
		{
			name:    "bash -c wrapper bypass",
			payload: bashPayload("bash -c 'git checkout -- another/dir/bar.go'"),
		},
		// Case 9: sh -c wrapping — analogous to case 8; "sh" is also unwatched.
		{
			name:    "sh -c wrapper bypass",
			payload: bashPayload("sh -c 'rm another/dir/bar.go'"),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, _ := runHook(t, script, tc.payload, env)
			if got != 0 {
				t.Errorf("exit code = %d; want 0 (known bypass — hook should not block): payload: %s", got, tc.payload)
			}
		})
	}
}

// TestHookIntegration_DegradedMode verifies the hook's soft-skip behaviour when
// required dependencies or configuration are absent.  Each case expects exit 0
// and a warning on stderr.
func TestHookIntegration_DegradedMode(t *testing.T) {
	t.Parallel()

	script := hookScriptPath(t)
	stub := tillBinStub(t)
	const actionItemID = "00000000-0000-0000-0000-000000000001"

	// Use a payload that would trigger blocking if the hook ran normally,
	// so a spurious exit-2 would surface clearly as a failure.
	outOfPathPayload := toolPayload("Edit", "another/dir/bar.go")

	t.Run("jq missing", func(t *testing.T) {
		t.Parallel()

		// Override PATH to a directory that exists but contains no executables.
		// bash is resolved by exec.Command using the PARENT process PATH before
		// cmd.Env is applied, so bash itself runs fine.  Inside the hook script,
		// "command -v jq" consults the restricted PATH and finds nothing.
		//
		// Limitation: if jq is installed as a shell built-in (rare) this PATH
		// override would not hide it.  On standard Linux/macOS installations jq
		// is always an external binary, so the override is effective in practice.
		emptyBinDir := t.TempDir()
		env := []string{
			"HOME=/tmp",
			"TILLSYN_BIN=" + stub,
			"TILLSYN_ACTION_ITEM_ID=" + actionItemID,
			"PATH=" + emptyBinDir,
		}
		exitCode, stderr := runHook(t, script, outOfPathPayload, env)
		if exitCode != 0 {
			t.Errorf("exit code = %d; want 0 (soft-skip when jq missing)", exitCode)
		}
		if !strings.Contains(stderr, "jq not found") {
			t.Errorf("stderr = %q; want it to contain %q", stderr, "jq not found")
		}
	})

	t.Run("TILLSYN_BIN empty", func(t *testing.T) {
		t.Parallel()

		env := []string{
			"HOME=/tmp",
			"TILLSYN_BIN=",
			"TILLSYN_ACTION_ITEM_ID=" + actionItemID,
			"PATH=" + os.Getenv("PATH"),
		}
		exitCode, stderr := runHook(t, script, outOfPathPayload, env)
		if exitCode != 0 {
			t.Errorf("exit code = %d; want 0 (soft-skip when TILLSYN_BIN empty)", exitCode)
		}
		if !strings.Contains(stderr, "TILLSYN_BIN not set") {
			t.Errorf("stderr = %q; want it to contain %q", stderr, "TILLSYN_BIN not set")
		}
	})

	t.Run("TILLSYN_ACTION_ITEM_ID empty", func(t *testing.T) {
		t.Parallel()

		env := []string{
			"HOME=/tmp",
			"TILLSYN_BIN=" + stub,
			"TILLSYN_ACTION_ITEM_ID=",
			"PATH=" + os.Getenv("PATH"),
		}
		exitCode, stderr := runHook(t, script, outOfPathPayload, env)
		if exitCode != 0 {
			t.Errorf("exit code = %d; want 0 (soft-skip when TILLSYN_ACTION_ITEM_ID empty)", exitCode)
		}
		if !strings.Contains(stderr, "TILLSYN_ACTION_ITEM_ID not set") {
			t.Errorf("stderr = %q; want it to contain %q", stderr, "TILLSYN_ACTION_ITEM_ID not set")
		}
	})
}
