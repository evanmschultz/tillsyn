package cli_codex

import (
	"bytes"
	"context"
	"encoding/json"
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

// --- Additional stream parsing tests — real codex event shapes (D5) --------

// TestParseStreamEvent_ErrorEvent asserts an error-type event maps to
// Type="error" with the message field in Text, and is non-terminal.
// This shape appears in real codex fixtures as a mid-stream error signal
// preceding turn.failed.
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
	if ev.IsTerminal {
		t.Errorf("IsTerminal = true; want false for mid-stream error event")
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

// --- Fixture-based tests (D5) — verified against real codex JSONL output ---

// TestParseStreamEvent_ThreadStarted asserts fixture line 1 decodes to
// Type="thread.started", IsTerminal=false, and Raw retains the full JSON
// including the thread_id field.
func TestParseStreamEvent_ThreadStarted(t *testing.T) {
	t.Parallel()

	line := []byte(`{"type":"thread.started","thread_id":"019e480a-21d9-7312-80e1-ed6e34fb193b"}`)
	ev, err := parseStreamEvent(line)
	if err != nil {
		t.Fatalf("parseStreamEvent: %v", err)
	}
	if ev.Type != "thread.started" {
		t.Errorf("Type = %q; want %q", ev.Type, "thread.started")
	}
	if ev.IsTerminal {
		t.Errorf("IsTerminal = true; want false for thread.started")
	}
	if !bytes.Equal(ev.Raw, line) {
		t.Errorf("Raw not preserved; got %q; want %q", string(ev.Raw), string(line))
	}
	// Verify thread_id is accessible via Raw for callers that need it.
	var rawObj map[string]string
	if err := json.Unmarshal(ev.Raw, &rawObj); err != nil {
		t.Fatalf("unmarshal Raw: %v", err)
	}
	if rawObj["thread_id"] == "" {
		t.Errorf("thread_id missing from Raw; want non-empty")
	}
}

// TestParseStreamEvent_TurnStarted asserts fixture line 2 decodes to
// Type="turn.started" and IsTerminal=false.
func TestParseStreamEvent_TurnStarted(t *testing.T) {
	t.Parallel()

	line := []byte(`{"type":"turn.started"}`)
	ev, err := parseStreamEvent(line)
	if err != nil {
		t.Fatalf("parseStreamEvent: %v", err)
	}
	if ev.Type != "turn.started" {
		t.Errorf("Type = %q; want %q", ev.Type, "turn.started")
	}
	if ev.IsTerminal {
		t.Errorf("IsTerminal = true; want false for turn.started")
	}
}

// TestParseStreamEvent_ItemCompletedAgentMessage asserts fixture line 3
// decodes to the canonical vocabulary: Type="assistant", Subtype="item.completed",
// Text="ok", and IsTerminal=false. Fix 1 (D5 r1): agent_message must map to
// "assistant" so downstream consumers like commit_agent (ev.Type=="assistant"
// filter) work correctly on codex-routed spawns.
func TestParseStreamEvent_ItemCompletedAgentMessage(t *testing.T) {
	t.Parallel()

	line := []byte(`{"type":"item.completed","item":{"id":"item_0","type":"agent_message","text":"ok"}}`)
	ev, err := parseStreamEvent(line)
	if err != nil {
		t.Fatalf("parseStreamEvent: %v", err)
	}
	// agent_message MUST normalise to canonical "assistant" type (Fix 1 D5 r1).
	if ev.Type != "assistant" {
		t.Errorf("Type = %q; want %q (agent_message must map to canonical assistant type)", ev.Type, "assistant")
	}
	// Subtype carries the codex wire-format event name for forensics.
	if ev.Subtype != "item.completed" {
		t.Errorf("Subtype = %q; want %q", ev.Subtype, "item.completed")
	}
	if ev.Text != "ok" {
		t.Errorf("Text = %q; want %q", ev.Text, "ok")
	}
	if ev.IsTerminal {
		t.Errorf("IsTerminal = true; want false for item.completed")
	}
}

// TestParseStreamEvent_TurnCompletedIsTerminal asserts fixture line 4 (success
// path) decodes to Type="turn.completed" and IsTerminal=true.
func TestParseStreamEvent_TurnCompletedIsTerminal(t *testing.T) {
	t.Parallel()

	line := []byte(`{"type":"turn.completed","usage":{"input_tokens":21276,"cached_input_tokens":6528,"output_tokens":38,"reasoning_output_tokens":31}}`)
	ev, err := parseStreamEvent(line)
	if err != nil {
		t.Fatalf("parseStreamEvent: %v", err)
	}
	if ev.Type != "turn.completed" {
		t.Errorf("Type = %q; want %q", ev.Type, "turn.completed")
	}
	if !ev.IsTerminal {
		t.Errorf("IsTerminal = false; want true for turn.completed")
	}
}

// TestParseStreamEvent_TurnFailedIsTerminal asserts the rate-limit fixture
// line 4 decodes to Type="turn.failed" and IsTerminal=true.
func TestParseStreamEvent_TurnFailedIsTerminal(t *testing.T) {
	t.Parallel()

	line := []byte(`{"type":"turn.failed","error":{"message":"You've hit your usage limit. To get more access now, send a request to your admin or try again at 7:54 PM."}}`)
	ev, err := parseStreamEvent(line)
	if err != nil {
		t.Fatalf("parseStreamEvent: %v", err)
	}
	if ev.Type != "turn.failed" {
		t.Errorf("Type = %q; want %q", ev.Type, "turn.failed")
	}
	if !ev.IsTerminal {
		t.Errorf("IsTerminal = false; want true for turn.failed")
	}
}

// TestParseStreamEvent_ErrorMidStream asserts the rate-limit fixture line 3
// decodes to Type="error" and IsTerminal=false. It is a mid-stream signal
// preceding turn.failed, not a terminal event.
func TestParseStreamEvent_ErrorMidStream(t *testing.T) {
	t.Parallel()

	line := []byte(`{"type":"error","message":"You've hit your usage limit. To get more access now, send a request to your admin or try again at 7:54 PM."}`)
	ev, err := parseStreamEvent(line)
	if err != nil {
		t.Fatalf("parseStreamEvent: %v", err)
	}
	if ev.Type != "error" {
		t.Errorf("Type = %q; want %q", ev.Type, "error")
	}
	if ev.IsTerminal {
		t.Errorf("IsTerminal = true; want false for mid-stream error event")
	}
	if ev.Text == "" {
		t.Errorf("Text = empty; want non-empty error message")
	}
}

// TestExtractTerminalReport_TurnCompletedReturnsTrue asserts extractTerminalReport
// on a parsed turn.completed event returns ok=true, Cost=nil (codex has no
// dollar cost field), Reason="turn_completed", and no errors.
func TestExtractTerminalReport_TurnCompletedReturnsTrue(t *testing.T) {
	t.Parallel()

	line := []byte(`{"type":"turn.completed","usage":{"input_tokens":21276,"cached_input_tokens":6528,"output_tokens":38,"reasoning_output_tokens":31}}`)
	ev, err := parseStreamEvent(line)
	if err != nil {
		t.Fatalf("parseStreamEvent: %v", err)
	}
	if !ev.IsTerminal {
		t.Fatalf("parseStreamEvent did not mark turn.completed as terminal")
	}

	report, ok := extractTerminalReport(ev)
	if !ok {
		t.Fatalf("extractTerminalReport returned ok=false on terminal event")
	}
	// Cost MUST be nil — codex has no dollar cost field (F.7.17 L11).
	if report.Cost != nil {
		t.Errorf("report.Cost = %v; want nil (codex emits no dollar cost)", *report.Cost)
	}
	if report.Reason != "turn_completed" {
		t.Errorf("report.Reason = %q; want %q", report.Reason, "turn_completed")
	}
	if len(report.Errors) != 0 {
		t.Errorf("report.Errors = %v; want empty on successful completion", report.Errors)
	}
}

// TestExtractTerminalReport_TurnFailedReturnsTrue asserts extractTerminalReport
// on a parsed turn.failed event returns ok=true, Reason="turn_failed", and
// Errors contains the rate-limit message string.
func TestExtractTerminalReport_TurnFailedReturnsTrue(t *testing.T) {
	t.Parallel()

	line := []byte(`{"type":"turn.failed","error":{"message":"You've hit your usage limit. To get more access now, send a request to your admin or try again at 7:54 PM."}}`)
	ev, err := parseStreamEvent(line)
	if err != nil {
		t.Fatalf("parseStreamEvent: %v", err)
	}
	if !ev.IsTerminal {
		t.Fatalf("parseStreamEvent did not mark turn.failed as terminal")
	}

	report, ok := extractTerminalReport(ev)
	if !ok {
		t.Fatalf("extractTerminalReport returned ok=false on terminal event")
	}
	if report.Reason != "turn_failed" {
		t.Errorf("report.Reason = %q; want %q", report.Reason, "turn_failed")
	}
	if len(report.Errors) == 0 {
		t.Fatalf("report.Errors is empty; want at least one error message")
	}
	if !strings.Contains(report.Errors[0], "usage limit") {
		t.Errorf("report.Errors[0] = %q; want string containing %q", report.Errors[0], "usage limit")
	}
}

// TestExtractTerminalReport_NonTerminalReturnsFalse_Fixture asserts
// extractTerminalReport returns (zero, false) for a non-terminal thread.started
// event parsed from a real fixture line.
func TestExtractTerminalReport_NonTerminalReturnsFalse_Fixture(t *testing.T) {
	t.Parallel()

	line := []byte(`{"type":"thread.started","thread_id":"019e480a-21d9-7312-80e1-ed6e34fb193b"}`)
	ev, err := parseStreamEvent(line)
	if err != nil {
		t.Fatalf("parseStreamEvent: %v", err)
	}
	if ev.IsTerminal {
		t.Fatalf("thread.started unexpectedly marked terminal")
	}

	report, ok := extractTerminalReport(ev)
	if ok {
		t.Fatalf("extractTerminalReport ok=true on non-terminal event; want false")
	}
	zero := dispatcher.TerminalReport{}
	if !reflect.DeepEqual(report, zero) {
		t.Errorf("report = %+v; want zero TerminalReport on non-terminal event", report)
	}
}

// TestParseStreamEvent_FullFixtureRoundtrip reads the entire success fixture
// (codex_stream_minimal.jsonl), parses each line, and asserts exactly 4 events
// with only the last (turn.completed) marked terminal.
func TestParseStreamEvent_FullFixtureRoundtrip(t *testing.T) {
	t.Parallel()

	data, err := os.ReadFile(filepath.Join("testdata", "codex_stream_minimal.jsonl"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	lines := splitJSONLLines(data)
	if len(lines) != 4 {
		t.Fatalf("expected 4 fixture lines; got %d", len(lines))
	}

	var terminalCount int
	for i, line := range lines {
		ev, err := parseStreamEvent(line)
		if err != nil {
			t.Fatalf("line %d: parseStreamEvent: %v", i+1, err)
		}
		if ev.IsTerminal {
			terminalCount++
			if i != len(lines)-1 {
				t.Errorf("line %d is terminal; only the last line should be terminal", i+1)
			}
		}
	}
	if terminalCount != 1 {
		t.Errorf("terminal count = %d; want exactly 1 (turn.completed)", terminalCount)
	}

	// Verify the last event is specifically turn.completed.
	lastEv, _ := parseStreamEvent(lines[3])
	if lastEv.Type != "turn.completed" {
		t.Errorf("last event Type = %q; want %q", lastEv.Type, "turn.completed")
	}
}

// TestParseStreamEvent_RateLimitFixtureRoundtrip reads the entire error fixture
// (codex_stream_rate_limit_error.jsonl), parses each line, and asserts exactly
// 4 events with only the last (turn.failed) marked terminal.
func TestParseStreamEvent_RateLimitFixtureRoundtrip(t *testing.T) {
	t.Parallel()

	data, err := os.ReadFile(filepath.Join("testdata", "codex_stream_rate_limit_error.jsonl"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	lines := splitJSONLLines(data)
	if len(lines) != 4 {
		t.Fatalf("expected 4 fixture lines; got %d", len(lines))
	}

	var terminalCount int
	for i, line := range lines {
		ev, err := parseStreamEvent(line)
		if err != nil {
			t.Fatalf("line %d: parseStreamEvent: %v", i+1, err)
		}
		if ev.IsTerminal {
			terminalCount++
			if i != len(lines)-1 {
				t.Errorf("line %d is terminal; only the last line should be terminal", i+1)
			}
		}
	}
	if terminalCount != 1 {
		t.Errorf("terminal count = %d; want exactly 1 (turn.failed)", terminalCount)
	}

	// Verify the last event is specifically turn.failed.
	lastEv, _ := parseStreamEvent(lines[3])
	if lastEv.Type != "turn.failed" {
		t.Errorf("last event Type = %q; want %q", lastEv.Type, "turn.failed")
	}
}

// splitJSONLLines splits JSONL data on newlines and drops any empty trailing
// line that os.ReadFile produces for files ending with a newline character.
func splitJSONLLines(data []byte) [][]byte {
	parts := bytes.Split(data, []byte("\n"))
	// Drop trailing empty line produced by a file-final newline.
	if len(parts) > 0 && len(bytes.TrimSpace(parts[len(parts)-1])) == 0 {
		parts = parts[:len(parts)-1]
	}
	return parts
}

// --- Fix 1 (D5 r1): item.completed canonical-type mapping tests -------------

// TestParseStreamEvent_ItemCompleted_UnknownSubkindFallthrough asserts the
// permissive fallback for all item.type values that are not in the recognised
// set. Per DROP-4D-R1 (REV-1 spirit), only "agent_message" has a verified
// canonical mapping; all other subkinds — including codex's wider vocabulary
// (commandExecution, fileChange, mcpToolCall, collabToolCall, reasoning, plan,
// tool_use, function_call) — fall through to the default branch until D4-style
// fixtures exist for each. The fallback contract: ev.Type="item.completed"
// (codex wire-format event name), ev.Subtype=item.type (raw subkind for
// forensics), ev.Text=item.text if present.
func TestParseStreamEvent_ItemCompleted_UnknownSubkindFallthrough(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		itemType    string
		text        string
		wantSubtype string
	}{
		{
			name:        "commandExecution",
			itemType:    "commandExecution",
			text:        "ls -la",
			wantSubtype: "commandExecution",
		},
		{
			name:        "tool_use (speculative — no fixture)",
			itemType:    "tool_use",
			text:        "",
			wantSubtype: "tool_use",
		},
		{
			name:        "function_call (speculative — no fixture)",
			itemType:    "function_call",
			text:        "",
			wantSubtype: "function_call",
		},
		{
			name:        "reasoning (speculative — reads wrong field until fixture lands)",
			itemType:    "reasoning",
			text:        "let me think",
			wantSubtype: "reasoning",
		},
		{
			name:        "fileChange",
			itemType:    "fileChange",
			text:        "",
			wantSubtype: "fileChange",
		},
		{
			name:        "future_item_kind",
			itemType:    "future_item_kind",
			text:        "payload",
			wantSubtype: "future_item_kind",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Build a minimal item.completed JSON line for this subkind.
			var lineStr string
			if tc.text != "" {
				lineStr = `{"type":"item.completed","item":{"type":"` + tc.itemType + `","text":"` + tc.text + `"}}`
			} else {
				lineStr = `{"type":"item.completed","item":{"type":"` + tc.itemType + `"}}`
			}
			line := []byte(lineStr)

			ev, err := parseStreamEvent(line)
			if err != nil {
				t.Fatalf("parseStreamEvent: %v", err)
			}
			// All unrecognised subkinds must fall through to the permissive
			// default: ev.Type retains the outer codex wire-format event name.
			if ev.Type != "item.completed" {
				t.Errorf("Type = %q; want %q (unknown subkind must use permissive fallback)", ev.Type, "item.completed")
			}
			// Subtype carries the raw item.type for forensics.
			if ev.Subtype != tc.wantSubtype {
				t.Errorf("Subtype = %q; want %q", ev.Subtype, tc.wantSubtype)
			}
			if ev.Text != tc.text {
				t.Errorf("Text = %q; want %q", ev.Text, tc.text)
			}
			if ev.IsTerminal {
				t.Errorf("IsTerminal = true; want false for item.completed %s", tc.itemType)
			}
		})
	}
}

// --- Fix 2 (D5 r1): turn.failed sentinel test ---------------------------------

// TestExtractTerminalReport_TurnFailedEmptyErrorReturnsSentinel asserts that
// a turn.failed event with an empty or absent error.message still produces a
// non-nil Errors slice containing the sentinel string. Consumers must be able
// to distinguish "failed with no message" from "failed but report was lost."
func TestExtractTerminalReport_TurnFailedEmptyErrorReturnsSentinel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		line []byte
	}{
		{
			name: "empty error object",
			line: []byte(`{"type":"turn.failed","error":{}}`),
		},
		{
			name: "no error field",
			line: []byte(`{"type":"turn.failed"}`),
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ev, err := parseStreamEvent(tc.line)
			if err != nil {
				t.Fatalf("parseStreamEvent: %v", err)
			}
			if !ev.IsTerminal {
				t.Fatalf("IsTerminal = false; want true for turn.failed")
			}

			report, ok := extractTerminalReport(ev)
			if !ok {
				t.Fatalf("extractTerminalReport ok=false on terminal turn.failed event")
			}
			if report.Reason != "turn_failed" {
				t.Errorf("Reason = %q; want %q", report.Reason, "turn_failed")
			}
			// Errors MUST be non-nil and contain the sentinel (Fix 2 D5 r1).
			if len(report.Errors) == 0 {
				t.Fatalf("Errors is empty; want at least sentinel string")
			}
			const sentinel = "codex: turn.failed without error.message"
			if !strings.Contains(report.Errors[0], sentinel) {
				t.Errorf("Errors[0] = %q; want string containing %q", report.Errors[0], sentinel)
			}
		})
	}
}

// --- Fix 3 (D5 r1): permissive non-object JSON test ---------------------------

// TestParseStreamEvent_AcceptsValidNonObjectJSON asserts that valid JSON which
// is NOT a JSON object (array, number, string, etc.) is returned as a
// non-terminal event with Raw populated and Type empty — no error. This
// matches the doc-comment contract and the F.7.17 L14 spirit (Raw retention
// over rejection).
func TestParseStreamEvent_AcceptsValidNonObjectJSON(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input []byte
	}{
		{name: "array", input: []byte("[1,2,3]")},
		{name: "number", input: []byte("42")},
		{name: "string", input: []byte(`"hello"`)},
		{name: "null", input: []byte("null")},
		{name: "empty array", input: []byte("[]")},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ev, err := parseStreamEvent(tc.input)
			if err != nil {
				t.Errorf("parseStreamEvent(%s) = error %v; want nil (valid non-object JSON must not error)", tc.name, err)
			}
			if !bytes.Equal(ev.Raw, tc.input) {
				t.Errorf("Raw = %q; want %q (original bytes must be retained)", string(ev.Raw), string(tc.input))
			}
			if ev.Type != "" {
				t.Errorf("Type = %q; want empty string for non-object JSON", ev.Type)
			}
			if ev.IsTerminal {
				t.Errorf("IsTerminal = true; want false for non-object JSON")
			}
		})
	}
}
