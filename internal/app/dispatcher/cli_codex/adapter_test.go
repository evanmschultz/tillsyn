package cli_codex

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/evanmschultz/tillsyn/internal/app/dispatcher"
)

// ptrString returns a pointer to s for use with pointer-typed BindingResolved fields.
func ptrString(s string) *string { return &s }

// ptrFloat returns a pointer to f for use with pointer-typed BindingResolved fields.
func ptrFloat(f float64) *float64 { return &f }

// ptrInt returns a pointer to i for use with pointer-typed BindingResolved fields.
func ptrInt(i int) *int { return &i }

// minimalBinding returns a BindingResolved suitable for tests that only care
// about basic BuildCommand behavior (no env requirements, no optional flags).
func minimalBinding() dispatcher.BindingResolved {
	return dispatcher.BindingResolved{
		AgentName: "go-planner-agent",
		CLIKind:   dispatcher.CLIKindCodex,
	}
}

// minimalPaths returns a BundlePaths pointing at a caller-supplied tmpdir.
// The caller must write a file at SystemPromptPath before calling BuildCommand,
// because BuildCommand reads (not opens) the prompt file.
func minimalPaths(t *testing.T) dispatcher.BundlePaths {
	t.Helper()
	dir := t.TempDir()
	promptPath := filepath.Join(dir, "system-prompt.md")
	if err := os.WriteFile(promptPath, []byte("test prompt"), 0o600); err != nil {
		t.Fatalf("write system-prompt fixture: %v", err)
	}
	return dispatcher.BundlePaths{
		Root:             dir,
		SystemPromptPath: promptPath,
	}
}

// envSliceToMap converts a cmd.Env slice (NAME=value lines) to a map for
// assertion convenience.
func envSliceToMap(env []string) map[string]string {
	m := make(map[string]string, len(env))
	for _, line := range env {
		idx := strings.Index(line, "=")
		if idx < 0 {
			continue
		}
		m[line[:idx]] = line[idx+1:]
	}
	return m
}

// hasArg reports whether args contains an exact-match element.
func hasArg(args []string, flag string) bool {
	for _, a := range args {
		if a == flag {
			return true
		}
	}
	return false
}

// --- Test 1: constructor ---------------------------------------------------

// TestNew_ReturnsNonNilAdapter asserts that New() returns a non-nil
// dispatcher.CLIAdapter. The compile-time interface assertion in adapter.go
// (var _ dispatcher.CLIAdapter = (*codexAdapter)(nil)) covers structural
// conformance; this test reinforces it at runtime.
func TestNew_ReturnsNonNilAdapter(t *testing.T) {
	t.Parallel()

	a := New()
	if a == nil {
		t.Fatalf("New() = nil; want non-nil dispatcher.CLIAdapter")
	}
}

// --- Test 2: binary name --------------------------------------------------

// TestBuildCommand_HardcodedBinaryName asserts BuildCommand always produces a
// command whose base name is "codex" regardless of other inputs. Per F.7.17
// REV-1 there is no Command override path.
func TestBuildCommand_HardcodedBinaryName(t *testing.T) {
	t.Parallel()

	a := New()
	cmd, err := a.BuildCommand(context.Background(), minimalBinding(), minimalPaths(t))
	if err != nil {
		t.Fatalf("BuildCommand: %v", err)
	}
	if got := filepath.Base(cmd.Path); got != "codex" {
		t.Fatalf("filepath.Base(cmd.Path) = %q; want %q (REV-1: binary must be hardcoded)", got, "codex")
	}
}

// --- Test 3: minimal argv shape -------------------------------------------

// TestBuildCommand_ArgvMinimal asserts the always-on flags are present and no
// conditional flags appear on a minimal binding (Model nil, Effort nil).
func TestBuildCommand_ArgvMinimal(t *testing.T) {
	t.Parallel()

	paths := minimalPaths(t)
	a := New()
	cmd, err := a.BuildCommand(context.Background(), minimalBinding(), paths)
	if err != nil {
		t.Fatalf("BuildCommand: %v", err)
	}

	// Always-on flags must be present.
	for _, want := range []string{"exec", "--json", "--ephemeral", "--skip-git-repo-check", "-C"} {
		if !hasArg(cmd.Args, want) {
			t.Errorf("argv missing always-on flag %q; got %v", want, cmd.Args)
		}
	}
	// -C must be followed by the bundle root.
	for i, a := range cmd.Args {
		if a == "-C" {
			if i+1 >= len(cmd.Args) || cmd.Args[i+1] != paths.Root {
				t.Errorf("-C value = %q; want %q", func() string {
					if i+1 < len(cmd.Args) {
						return cmd.Args[i+1]
					}
					return "<missing>"
				}(), paths.Root)
			}
			break
		}
	}

	// Conditional flags must NOT appear on a minimal binding.
	for _, absent := range []string{"-m", "model_reasoning_effort"} {
		if hasArg(cmd.Args, absent) {
			t.Errorf("argv unexpectedly contains %q on minimal binding: %v", absent, cmd.Args)
		}
	}
}

// --- Test 4: -m flag when Model is set ------------------------------------

// TestBuildCommand_ArgvWithModel asserts the -m flag and model value appear
// when binding.Model is non-nil.
func TestBuildCommand_ArgvWithModel(t *testing.T) {
	t.Parallel()

	binding := minimalBinding()
	binding.Model = ptrString("gpt-5")

	a := New()
	cmd, err := a.BuildCommand(context.Background(), binding, minimalPaths(t))
	if err != nil {
		t.Fatalf("BuildCommand: %v", err)
	}

	for i, arg := range cmd.Args {
		if arg == "-m" {
			if i+1 >= len(cmd.Args) {
				t.Fatalf("-m present but no following value in argv %v", cmd.Args)
			}
			if got, want := cmd.Args[i+1], "gpt-5"; got != want {
				t.Errorf("-m value = %q; want %q", got, want)
			}
			return
		}
	}
	t.Errorf("-m flag missing from argv when Model = \"gpt-5\"; got %v", cmd.Args)
}

// --- Test 5: -c flag when Effort is set -----------------------------------

// TestBuildCommand_ArgvWithEffort asserts the -c config override appears
// when binding.Effort is non-nil.
func TestBuildCommand_ArgvWithEffort(t *testing.T) {
	t.Parallel()

	binding := minimalBinding()
	binding.Effort = ptrString("high")

	a := New()
	cmd, err := a.BuildCommand(context.Background(), binding, minimalPaths(t))
	if err != nil {
		t.Fatalf("BuildCommand: %v", err)
	}

	wantValue := "model_reasoning_effort=high"
	for i, arg := range cmd.Args {
		if arg == "-c" {
			if i+1 >= len(cmd.Args) {
				t.Fatalf("-c present but no following value in argv %v", cmd.Args)
			}
			if got := cmd.Args[i+1]; got != wantValue {
				t.Errorf("-c value = %q; want %q", got, wantValue)
			}
			return
		}
	}
	t.Errorf("-c flag missing from argv when Effort = \"high\"; got %v", cmd.Args)
}

// --- Test 6: env isolation (sentinel must not leak) -----------------------

// TestBuildCommand_EnvNotInheritedFromOSEnviron asserts that a sentinel env
// var set in the orchestrator process does NOT appear in cmd.Env when it is
// not in the closed baseline AND not in binding.Env. Proves the F.7.17 L8
// isolation guarantee.
func TestBuildCommand_EnvNotInheritedFromOSEnviron(t *testing.T) {
	// NOT t.Parallel() — we mutate process env via t.Setenv.

	const sentinelName = "TILLSYN_TEST_CODEX_SENTINEL_xyzzy"
	const sentinelValue = "should-not-leak"
	t.Setenv(sentinelName, sentinelValue)

	a := New()
	cmd, err := a.BuildCommand(context.Background(), minimalBinding(), minimalPaths(t))
	if err != nil {
		t.Fatalf("BuildCommand: %v", err)
	}

	for _, line := range cmd.Env {
		if strings.HasPrefix(line, sentinelName+"=") {
			t.Fatalf("sentinel env var %q LEAKED into cmd.Env: %q (L8 isolation broken)", sentinelName, line)
		}
		if strings.Contains(line, sentinelValue) {
			t.Fatalf("sentinel value leaked via different name: %q", line)
		}
	}
}

// --- Test 7: env baseline includes PATH and HOME --------------------------

// TestBuildCommand_EnvBaselineIncludesPathHome asserts that PATH and HOME
// (the most fundamental POSIX process vars) appear in cmd.Env when set in
// the orchestrator process.
func TestBuildCommand_EnvBaselineIncludesPathHome(t *testing.T) {
	// NOT t.Parallel() — we mutate process env via t.Setenv.

	t.Setenv("PATH", "/usr/bin:/bin")
	t.Setenv("HOME", "/home/testuser")

	a := New()
	cmd, err := a.BuildCommand(context.Background(), minimalBinding(), minimalPaths(t))
	if err != nil {
		t.Fatalf("BuildCommand: %v", err)
	}

	envMap := envSliceToMap(cmd.Env)
	if got, want := envMap["PATH"], "/usr/bin:/bin"; got != want {
		t.Errorf("PATH in cmd.Env = %q; want %q", got, want)
	}
	if got, want := envMap["HOME"], "/home/testuser"; got != want {
		t.Errorf("HOME in cmd.Env = %q; want %q", got, want)
	}
}

// --- Test 8: binding allow-list resolved into env -------------------------

// TestBuildCommand_EnvBindingAllowlistResolved asserts that a name in
// binding.Env resolves to its orchestrator-process value in cmd.Env.
func TestBuildCommand_EnvBindingAllowlistResolved(t *testing.T) {
	// NOT t.Parallel() — we mutate process env via t.Setenv.

	t.Setenv("TILLSYN_TEST_BINDING_VAR", "allowed-value")

	binding := minimalBinding()
	binding.Env = []string{"TILLSYN_TEST_BINDING_VAR"}

	a := New()
	cmd, err := a.BuildCommand(context.Background(), binding, minimalPaths(t))
	if err != nil {
		t.Fatalf("BuildCommand: %v", err)
	}

	envMap := envSliceToMap(cmd.Env)
	if got, want := envMap["TILLSYN_TEST_BINDING_VAR"], "allowed-value"; got != want {
		t.Errorf("TILLSYN_TEST_BINDING_VAR in cmd.Env = %q; want %q", got, want)
	}
}

// --- Test 9: missing required env fails loud ------------------------------

// TestBuildCommand_MissingRequiredEnvFailsLoud asserts that a binding.Env
// name with no value in the orchestrator process returns an error wrapping
// ErrMissingRequiredEnv. Per F.7.17 P5 the dispatcher routes this to
// pre-lock so the spawn never starts with an incomplete environment.
func TestBuildCommand_MissingRequiredEnvFailsLoud(t *testing.T) {
	// NOT t.Parallel() — we mutate process env via Unsetenv.

	const missingName = "TILLSYN_TEST_CODEX_UNSET_VAR"
	if err := os.Unsetenv(missingName); err != nil {
		t.Fatalf("os.Unsetenv: %v", err)
	}

	binding := minimalBinding()
	binding.Env = []string{missingName}

	a := New()
	cmd, err := a.BuildCommand(context.Background(), binding, minimalPaths(t))
	if err == nil {
		t.Fatalf("BuildCommand returned nil error for missing required env var; want ErrMissingRequiredEnv")
	}
	if !errors.Is(err, ErrMissingRequiredEnv) {
		t.Errorf("error chain does not wrap ErrMissingRequiredEnv: %v", err)
	}
	if !strings.Contains(err.Error(), missingName) {
		t.Errorf("error message %q does not name the missing var %q", err.Error(), missingName)
	}
	if cmd != nil {
		t.Errorf("BuildCommand returned non-nil cmd alongside error; want nil")
	}
}

// --- Test 10: ParseStreamEvent on malformed input -------------------------

// TestParseStreamEvent_PopulatesRawOnMalformedInput asserts that a non-JSON
// line returns an error wrapping ErrMalformedStreamLine and that ev.Raw
// contains the original bytes.
func TestParseStreamEvent_PopulatesRawOnMalformedInput(t *testing.T) {
	t.Parallel()

	line := []byte("{not valid json")
	ev, err := parseStreamEvent(line)
	if err == nil {
		t.Fatalf("parseStreamEvent on malformed JSON returned nil error")
	}
	if !errors.Is(err, ErrMalformedStreamLine) {
		t.Errorf("error chain does not wrap ErrMalformedStreamLine: %v", err)
	}
	if !bytes.Equal(ev.Raw, line) {
		t.Errorf("ev.Raw = %q; want original line %q", string(ev.Raw), string(line))
	}
}

// --- Test 11: ParseStreamEvent on empty JSON object -----------------------

// TestParseStreamEvent_PopulatesRawOnEmptyJsonObject asserts that a valid
// but empty JSON object is parsed permissively — no error, ev.Raw is
// populated, ev.Type is empty (no type field present).
func TestParseStreamEvent_PopulatesRawOnEmptyJsonObject(t *testing.T) {
	t.Parallel()

	line := []byte("{}")
	ev, err := parseStreamEvent(line)
	if err != nil {
		t.Fatalf("parseStreamEvent on empty JSON object returned error: %v", err)
	}
	if !bytes.Equal(ev.Raw, line) {
		t.Errorf("ev.Raw = %q; want original line %q", string(ev.Raw), string(line))
	}
	if ev.Type != "" {
		t.Errorf("ev.Type = %q; want empty string for object with no type field", ev.Type)
	}
}

// --- Test 12: ExtractTerminalReport on non-terminal event -----------------

// TestExtractTerminalReport_NonTerminalReturnsFalse asserts the (zero, false)
// contract for non-terminal events. Callers gate extraction on this bool so
// misuse on non-terminal events MUST return an unmistakable zero report.
func TestExtractTerminalReport_NonTerminalReturnsFalse(t *testing.T) {
	t.Parallel()

	ev := dispatcher.StreamEvent{IsTerminal: false}
	report, ok := extractTerminalReport(ev)
	if ok {
		t.Fatalf("extractTerminalReport ok=true on non-terminal event; want false")
	}
	zero := dispatcher.TerminalReport{}
	if !reflect.DeepEqual(report, zero) {
		t.Errorf("report = %+v; want zero TerminalReport on non-terminal event", report)
	}
}

// --- Test for Fix 1: stdin contents via bytes.Reader ---------------------

// TestBuildCommand_StdinContentsMatchPromptFile asserts that cmd.Stdin is
// non-nil and readable, and its contents match the system-prompt file.
// This implicitly validates the bytes.Reader path (Fix 1): no *os.File fd,
// no fd lifecycle concern.
func TestBuildCommand_StdinContentsMatchPromptFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	promptPath := filepath.Join(dir, "system-prompt.md")
	wantContent := []byte("# System Prompt\nYou are a test agent.")
	if err := os.WriteFile(promptPath, wantContent, 0o600); err != nil {
		t.Fatalf("write system-prompt: %v", err)
	}

	paths := dispatcher.BundlePaths{
		Root:             dir,
		SystemPromptPath: promptPath,
	}

	a := New()
	cmd, err := a.BuildCommand(context.Background(), minimalBinding(), paths)
	if err != nil {
		t.Fatalf("BuildCommand: %v", err)
	}
	if cmd.Stdin == nil {
		t.Fatalf("cmd.Stdin = nil; want non-nil (prompt must be fed via stdin)")
	}

	// Read all content from the stdin reader to verify it contains the
	// expected prompt text.
	got, err := io.ReadAll(cmd.Stdin.(io.Reader))
	if err != nil {
		t.Fatalf("read cmd.Stdin: %v", err)
	}
	if !bytes.Equal(got, wantContent) {
		t.Errorf("cmd.Stdin content = %q; want %q", string(got), string(wantContent))
	}
}

// --- Test for Fix 2: MaxTurns warning path --------------------------------

// TestBuildCommand_MaxTurnsWarningDoesNotError asserts that a binding with
// MaxTurns non-nil still returns a valid *exec.Cmd without error — the WARN
// is logged but the spawn is not aborted. This is the Fix 2 Option A path:
// loud-but-non-fatal.
func TestBuildCommand_MaxTurnsWarningDoesNotError(t *testing.T) {
	t.Parallel()

	binding := minimalBinding()
	binding.MaxTurns = ptrInt(10)

	a := New()
	cmd, err := a.BuildCommand(context.Background(), binding, minimalPaths(t))
	if err != nil {
		t.Fatalf("BuildCommand with MaxTurns set returned error: %v", err)
	}
	if cmd == nil {
		t.Fatalf("BuildCommand with MaxTurns set returned nil cmd")
	}
	// The codex adapter must NOT silently pass MaxTurns as a CLI flag —
	// there is no codex exec equivalent.
	if hasArg(cmd.Args, "--max-turns") {
		t.Errorf("cmd.Args contains --max-turns; codex adapter must not emit unsupported flags")
	}
}

// TestBuildCommand_MaxBudgetUSDWarningDoesNotError asserts that a binding
// with MaxBudgetUSD non-nil still returns a valid *exec.Cmd without error.
func TestBuildCommand_MaxBudgetUSDWarningDoesNotError(t *testing.T) {
	t.Parallel()

	binding := minimalBinding()
	binding.MaxBudgetUSD = ptrFloat(10.0)

	a := New()
	cmd, err := a.BuildCommand(context.Background(), binding, minimalPaths(t))
	if err != nil {
		t.Fatalf("BuildCommand with MaxBudgetUSD set returned error: %v", err)
	}
	if cmd == nil {
		t.Fatalf("BuildCommand with MaxBudgetUSD set returned nil cmd")
	}
	// The codex adapter must NOT emit a --max-budget-usd flag — codex has no
	// equivalent. The WARN is emitted to the logger; the spawn proceeds.
	if hasArg(cmd.Args, "--max-budget-usd") {
		t.Errorf("cmd.Args contains --max-budget-usd; codex adapter must not emit unsupported flags")
	}
}

// --- Additional stream parsing tests for coverage -------------------------

// TestParseStreamEvent_DoneTerminal asserts that a JSON object with done=true
// is treated as a terminal event with Type="result".
func TestParseStreamEvent_DoneTerminal(t *testing.T) {
	t.Parallel()

	line := []byte(`{"done":true}`)
	ev, err := parseStreamEvent(line)
	if err != nil {
		t.Fatalf("parseStreamEvent: %v", err)
	}
	if !ev.IsTerminal {
		t.Errorf("IsTerminal = false; want true for done=true event")
	}
	if ev.Type != "result" {
		t.Errorf("Type = %q; want %q", ev.Type, "result")
	}
}

// TestParseStreamEvent_FinishReasonTerminal asserts a finish_reason string
// triggers the terminal path with Subtype set to the finish_reason value.
func TestParseStreamEvent_FinishReasonTerminal(t *testing.T) {
	t.Parallel()

	line := []byte(`{"finish_reason":"stop","done":false}`)
	ev, err := parseStreamEvent(line)
	if err != nil {
		t.Fatalf("parseStreamEvent: %v", err)
	}
	if !ev.IsTerminal {
		t.Errorf("IsTerminal = false; want true for finish_reason event")
	}
	if ev.Subtype != "stop" {
		t.Errorf("Subtype = %q; want %q", ev.Subtype, "stop")
	}
}

// TestParseStreamEvent_MessageRoleAssistant asserts a message event with
// role=assistant maps to Type="assistant" with a text content field.
func TestParseStreamEvent_MessageRoleAssistant(t *testing.T) {
	t.Parallel()

	line := []byte(`{"type":"message","role":"assistant","content":"hello from codex"}`)
	ev, err := parseStreamEvent(line)
	if err != nil {
		t.Fatalf("parseStreamEvent: %v", err)
	}
	if ev.Type != "assistant" {
		t.Errorf("Type = %q; want %q", ev.Type, "assistant")
	}
	if ev.Text != "hello from codex" {
		t.Errorf("Text = %q; want %q", ev.Text, "hello from codex")
	}
}

// TestParseStreamEvent_MessageRoleUser asserts a message event with
// role=user maps to Type="user".
func TestParseStreamEvent_MessageRoleUser(t *testing.T) {
	t.Parallel()

	line := []byte(`{"type":"message","role":"user"}`)
	ev, err := parseStreamEvent(line)
	if err != nil {
		t.Fatalf("parseStreamEvent: %v", err)
	}
	if ev.Type != "user" {
		t.Errorf("Type = %q; want %q", ev.Type, "user")
	}
}

// TestParseStreamEvent_MessageUnknownRole asserts a message event with an
// unknown role maps to Type="message" (pass-through).
func TestParseStreamEvent_MessageUnknownRole(t *testing.T) {
	t.Parallel()

	line := []byte(`{"type":"message","role":"system"}`)
	ev, err := parseStreamEvent(line)
	if err != nil {
		t.Fatalf("parseStreamEvent: %v", err)
	}
	if ev.Type != "message" {
		t.Errorf("Type = %q; want %q on unknown role", ev.Type, "message")
	}
}

// TestParseStreamEvent_FunctionCallEvent asserts a function_call event maps
// to Type="assistant" with ToolName populated.
func TestParseStreamEvent_FunctionCallEvent(t *testing.T) {
	t.Parallel()

	line := []byte(`{"type":"function_call","name":"Bash","arguments":{"command":"ls"}}`)
	ev, err := parseStreamEvent(line)
	if err != nil {
		t.Fatalf("parseStreamEvent: %v", err)
	}
	if ev.Type != "assistant" {
		t.Errorf("Type = %q; want %q for function_call", ev.Type, "assistant")
	}
	if ev.ToolName != "Bash" {
		t.Errorf("ToolName = %q; want %q", ev.ToolName, "Bash")
	}
	if ev.ToolInput == nil {
		t.Errorf("ToolInput = nil; want non-nil for function_call with arguments")
	}
}

// TestParseStreamEvent_ErrorEvent asserts an error-type event maps to
// Type="error" with the message field in Text.
func TestParseStreamEvent_ErrorEvent(t *testing.T) {
	t.Parallel()

	line := []byte(`{"type":"error","message":"something failed"}`)
	ev, err := parseStreamEvent(line)
	if err != nil {
		t.Fatalf("parseStreamEvent: %v", err)
	}
	if ev.Type != "error" {
		t.Errorf("Type = %q; want %q", ev.Type, "error")
	}
	if ev.Text != "something failed" {
		t.Errorf("Text = %q; want %q", ev.Text, "something failed")
	}
}

// TestParseStreamEvent_KindFieldFallback asserts that when no "type" field
// is present but a "kind" field is, the kind value is used as Type.
func TestParseStreamEvent_KindFieldFallback(t *testing.T) {
	t.Parallel()

	line := []byte(`{"kind":"function_call","name":"Read","arguments":{}}`)
	ev, err := parseStreamEvent(line)
	if err != nil {
		t.Fatalf("parseStreamEvent: %v", err)
	}
	// function_call via kind fallback maps to assistant.
	if ev.Type != "assistant" {
		t.Errorf("Type = %q; want %q via kind= fallback", ev.Type, "assistant")
	}
}

// TestExtractTerminalReport_WithCostAndErrors asserts the terminal-report
// decoder surfaces cost and errors from a codex terminal event.
func TestExtractTerminalReport_WithCostAndErrors(t *testing.T) {
	t.Parallel()

	// Build a synthetic terminal event as if parseStreamEvent produced it.
	terminalLine := []byte(`{"done":true,"finish_reason":"stop","total_cost_usd":0.05,"errors":["rate limit hit"]}`)
	ev, err := parseStreamEvent(terminalLine)
	if err != nil {
		t.Fatalf("parseStreamEvent: %v", err)
	}
	if !ev.IsTerminal {
		t.Fatalf("parseStreamEvent did not mark event as terminal")
	}

	report, ok := extractTerminalReport(ev)
	if !ok {
		t.Fatalf("extractTerminalReport returned ok=false on terminal event")
	}
	if report.Cost == nil {
		t.Fatalf("report.Cost = nil; want pointer to 0.05")
	}
	if got, want := *report.Cost, 0.05; got != want {
		t.Errorf("*report.Cost = %v; want %v", got, want)
	}
	if len(report.Errors) != 1 || report.Errors[0] != "rate limit hit" {
		t.Errorf("report.Errors = %v; want [\"rate limit hit\"]", report.Errors)
	}
}

// TestExtractTerminalReport_SingleErrorField asserts the single `error`
// string field (not `errors` array) is surfaced in report.Errors.
func TestExtractTerminalReport_SingleErrorField(t *testing.T) {
	t.Parallel()

	terminalLine := []byte(`{"done":true,"error":"spawn failed"}`)
	ev, err := parseStreamEvent(terminalLine)
	if err != nil {
		t.Fatalf("parseStreamEvent: %v", err)
	}

	report, ok := extractTerminalReport(ev)
	if !ok {
		t.Fatalf("extractTerminalReport returned ok=false on terminal event")
	}
	if len(report.Errors) != 1 || report.Errors[0] != "spawn failed" {
		t.Errorf("report.Errors = %v; want [\"spawn failed\"]", report.Errors)
	}
}

// TestExtractTerminalReport_EmptyRawReturnsTrue asserts that a terminal event
// with no Raw bytes still returns (zero, true) — the dispatcher needs to
// know the spawn ended even when there is no payload to decode.
func TestExtractTerminalReport_EmptyRawReturnsTrue(t *testing.T) {
	t.Parallel()

	ev := dispatcher.StreamEvent{IsTerminal: true, Raw: nil}
	_, ok := extractTerminalReport(ev)
	if !ok {
		t.Fatalf("extractTerminalReport ok=false on terminal event with empty Raw; want true")
	}
}

// TestParseStreamEvent_ContentArrayBlocks asserts that a content field
// structured as an array of typed blocks extracts the first "text" block.
func TestParseStreamEvent_ContentArrayBlocks(t *testing.T) {
	t.Parallel()

	line := []byte(`{"type":"message","role":"assistant","content":[{"type":"text","text":"block text"},{"type":"other","text":"ignored"}]}`)
	ev, err := parseStreamEvent(line)
	if err != nil {
		t.Fatalf("parseStreamEvent: %v", err)
	}
	if ev.Type != "assistant" {
		t.Errorf("Type = %q; want %q", ev.Type, "assistant")
	}
	if ev.Text != "block text" {
		t.Errorf("Text = %q; want %q (first text block)", ev.Text, "block text")
	}
}
