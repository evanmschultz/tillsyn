package cli_claude

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/evanmschultz/tillsyn/internal/app/dispatcher"
)

// TestNewReturnsCLIAdapter pins the constructor's return type to the
// dispatcher.CLIAdapter interface so callers cannot accidentally start
// depending on adapter-private fields.
func TestNewReturnsCLIAdapter(t *testing.T) {
	t.Parallel()

	a := New()
	if a == nil {
		t.Fatalf("New() = nil; want non-nil dispatcher.CLIAdapter")
	}
	// Interface satisfaction is asserted at compile time via the
	// `var _ dispatcher.CLIAdapter = (*claudeAdapter)(nil)` line in
	// adapter.go — duplicating the assertion at runtime would be redundant.
}

// TestBuildCommandHardcodedBinary asserts that BuildCommand always uses
// the hardcoded `claude` binary name regardless of any other input. Per
// F.7.17 REV-1 there is no Command override path; this guards against
// regression if someone reintroduces a Command field on BindingResolved.
func TestBuildCommandHardcodedBinary(t *testing.T) {
	t.Parallel()

	a := New()
	cmd, err := a.BuildCommand(context.Background(), minimalBinding(t), minimalPaths(t))
	if err != nil {
		t.Fatalf("BuildCommand returned error: %v", err)
	}
	if got, want := filepath.Base(cmd.Path), "claude"; got != want {
		t.Fatalf("cmd.Path basename = %q; want %q (binary must be hardcoded per REV-1)", got, want)
	}
	if len(cmd.Args) == 0 || cmd.Args[0] != "claude" {
		t.Fatalf("cmd.Args[0] = %q; want %q", firstArg(cmd.Args), "claude")
	}
}

// TestBuildCommandArgvShapeMinimal asserts the always-on argv flags appear
// in the documented order with default placeholder paths plumbed through.
// Conditional pointer-typed fields (Model, Effort, Tools, …) are nil so
// their flags MUST NOT appear.
func TestBuildCommandArgvShapeMinimal(t *testing.T) {
	t.Parallel()

	binding := dispatcher.BindingResolved{
		AgentName: "go-builder-agent",
		CLIKind:   dispatcher.CLIKindClaude,
		// Env intentionally empty so no required-env failure.
	}
	paths := dispatcher.BundlePaths{
		Root:             "/tmp/spawn-xyz",
		SystemPromptPath: "/tmp/spawn-xyz/system-prompt.md",
		// SystemAppendPath empty → --append-system-prompt-file MUST NOT emit.
	}

	a := New()
	cmd, err := a.BuildCommand(context.Background(), binding, paths)
	if err != nil {
		t.Fatalf("BuildCommand: %v", err)
	}

	// Always-present flags + their argument values.
	expectFlagPresent(t, cmd.Args, "--bare", "")
	expectFlagPresent(t, cmd.Args, "--plugin-dir", "/tmp/spawn-xyz/plugin")
	expectFlagPresent(t, cmd.Args, "--agent", "go-builder-agent")
	expectFlagPresent(t, cmd.Args, "--system-prompt-file", "/tmp/spawn-xyz/system-prompt.md")
	expectFlagPresent(t, cmd.Args, "--settings", "/tmp/spawn-xyz/plugin/settings.json")
	expectFlagPresent(t, cmd.Args, "--setting-sources", "")
	expectFlagPresent(t, cmd.Args, "--strict-mcp-config", "")
	expectFlagPresent(t, cmd.Args, "--permission-mode", "acceptEdits")
	expectFlagPresent(t, cmd.Args, "--output-format", "stream-json")
	expectFlagPresent(t, cmd.Args, "--verbose", "")
	expectFlagPresent(t, cmd.Args, "--no-session-persistence", "")
	expectFlagPresent(t, cmd.Args, "--exclude-dynamic-system-prompt-sections", "")
	expectFlagPresent(t, cmd.Args, "--mcp-config", "/tmp/spawn-xyz/plugin/.mcp.json")
	expectFlagPresent(t, cmd.Args, "-p", "")

	// Conditional flags MUST be absent on a minimal binding.
	for _, flag := range []string{"--max-budget-usd", "--max-turns", "--effort", "--model", "--tools", "--append-system-prompt-file"} {
		if hasArg(cmd.Args, flag) {
			t.Fatalf("argv unexpectedly contains %q on minimal binding: %v", flag, cmd.Args)
		}
	}
}

// TestBuildCommandArgvShapeFullyPopulated asserts every conditional flag
// emits with its resolved value when its pointer is non-nil. Crucial for
// the F.7.17 L9 contract: pointer-typed fields distinguish absent from
// explicit-zero, and the adapter MUST honor that distinction.
func TestBuildCommandArgvShapeFullyPopulated(t *testing.T) {
	t.Parallel()

	model := "opus"
	effort := "high"
	maxBudget := 5.5
	maxTurns := 12
	binding := dispatcher.BindingResolved{
		AgentName:    "go-builder-agent",
		CLIKind:      dispatcher.CLIKindClaude,
		Model:        &model,
		Effort:       &effort,
		MaxBudgetUSD: &maxBudget,
		MaxTurns:     &maxTurns,
		Tools:        []string{"Read", "Grep", "Glob"},
	}
	paths := dispatcher.BundlePaths{
		Root:             "/tmp/spawn-xyz",
		SystemPromptPath: "/tmp/spawn-xyz/system-prompt.md",
		SystemAppendPath: "/tmp/spawn-xyz/system-append.md",
	}

	a := New()
	cmd, err := a.BuildCommand(context.Background(), binding, paths)
	if err != nil {
		t.Fatalf("BuildCommand: %v", err)
	}

	expectFlagPresent(t, cmd.Args, "--max-budget-usd", "5.5")
	expectFlagPresent(t, cmd.Args, "--max-turns", "12")
	expectFlagPresent(t, cmd.Args, "--effort", "high")
	expectFlagPresent(t, cmd.Args, "--model", "opus")
	expectFlagPresent(t, cmd.Args, "--tools", "Read,Grep,Glob")
	expectFlagPresent(t, cmd.Args, "--append-system-prompt-file", "/tmp/spawn-xyz/system-append.md")
}

// TestBuildCommandMaxBudgetWholeNumberFormatting asserts whole-dollar
// budget values render without trailing decimals, matching the 4a.19 stub
// formatter behavior so argv parity holds during the F.7.17.5 wiring
// rewrite.
func TestBuildCommandMaxBudgetWholeNumberFormatting(t *testing.T) {
	t.Parallel()

	budget := 5.0
	binding := dispatcher.BindingResolved{
		AgentName:    "go-builder-agent",
		CLIKind:      dispatcher.CLIKindClaude,
		MaxBudgetUSD: &budget,
	}
	paths := dispatcher.BundlePaths{Root: "/tmp/r", SystemPromptPath: "/tmp/r/sp.md"}

	a := New()
	cmd, err := a.BuildCommand(context.Background(), binding, paths)
	if err != nil {
		t.Fatalf("BuildCommand: %v", err)
	}
	expectFlagPresent(t, cmd.Args, "--max-budget-usd", "5")
}

// TestEnvBaselineNamesAllInherited asserts every name in the closed
// baseline list whose os.Getenv returns non-empty appears in the emitted
// cmd.Env. The test sets a known value for each baseline name first so
// the assertion is deterministic.
func TestEnvBaselineNamesAllInherited(t *testing.T) {
	// NOT t.Parallel() — we mutate process env via t.Setenv.

	// Each baseline name gets a known sentinel value.
	for _, name := range closedBaselineEnvNames {
		t.Setenv(name, "TEST_"+name)
	}

	binding := dispatcher.BindingResolved{
		AgentName: "go-builder-agent",
		CLIKind:   dispatcher.CLIKindClaude,
	}
	env, err := assembleEnv(binding)
	if err != nil {
		t.Fatalf("assembleEnv: %v", err)
	}

	envMap := envSliceToMap(env)
	for _, name := range closedBaselineEnvNames {
		got, ok := envMap[name]
		if !ok {
			t.Errorf("baseline name %q missing from emitted env (have %d entries)", name, len(envMap))
			continue
		}
		if want := "TEST_" + name; got != want {
			t.Errorf("baseline name %q: got %q; want %q", name, got, want)
		}
	}
}

// TestEnvBaselineUnsetNamesOmitted asserts that baseline names with no
// orchestrator value are silently omitted from cmd.Env (we do NOT emit
// "NAME=" for absent baseline vars). Distinguishes from the binding.Env
// path which fails loud on absence.
func TestEnvBaselineUnsetNamesOmitted(t *testing.T) {
	// NOT t.Parallel() — we mutate process env via Unsetenv.

	// Snapshot + clear all baseline names so we control absence
	// deterministically. t.Setenv with empty string still EMITS NAME=,
	// so we must use os.Unsetenv. t.Cleanup restores the prior values.
	for _, name := range closedBaselineEnvNames {
		prev, hadPrev := os.LookupEnv(name)
		if err := os.Unsetenv(name); err != nil {
			t.Fatalf("os.Unsetenv(%q): %v", name, err)
		}
		t.Cleanup(func() {
			if hadPrev {
				_ = os.Setenv(name, prev)
			} else {
				_ = os.Unsetenv(name)
			}
		})
	}

	binding := dispatcher.BindingResolved{
		AgentName: "go-builder-agent",
		CLIKind:   dispatcher.CLIKindClaude,
	}
	env, err := assembleEnv(binding)
	if err != nil {
		t.Fatalf("assembleEnv: %v", err)
	}
	for _, line := range env {
		for _, name := range closedBaselineEnvNames {
			if strings.HasPrefix(line, name+"=") {
				t.Errorf("baseline name %q unexpectedly emitted as %q while unset", name, line)
			}
		}
	}
}

// TestEnvBindingNamesAppended asserts that every name in BindingResolved.Env
// resolves to its os.Getenv value and appears in the emitted cmd.Env in
// addition to the closed baseline. Names NOT in the baseline are appended
// in sorted order for snapshot stability.
func TestEnvBindingNamesAppended(t *testing.T) {
	// NOT t.Parallel() — we mutate process env.

	t.Setenv("TILLSYN_TEST_BINDING_VAR", "binding-value")
	t.Setenv("TILLSYN_TEST_OTHER_VAR", "other-value")
	t.Setenv("PATH", "/usr/bin:/bin")

	binding := dispatcher.BindingResolved{
		AgentName: "go-builder-agent",
		CLIKind:   dispatcher.CLIKindClaude,
		Env:       []string{"TILLSYN_TEST_BINDING_VAR", "TILLSYN_TEST_OTHER_VAR"},
	}
	env, err := assembleEnv(binding)
	if err != nil {
		t.Fatalf("assembleEnv: %v", err)
	}

	envMap := envSliceToMap(env)
	if got, want := envMap["TILLSYN_TEST_BINDING_VAR"], "binding-value"; got != want {
		t.Errorf("TILLSYN_TEST_BINDING_VAR: got %q; want %q", got, want)
	}
	if got, want := envMap["TILLSYN_TEST_OTHER_VAR"], "other-value"; got != want {
		t.Errorf("TILLSYN_TEST_OTHER_VAR: got %q; want %q", got, want)
	}
	if got, want := envMap["PATH"], "/usr/bin:/bin"; got != want {
		t.Errorf("PATH: got %q; want %q", got, want)
	}
}

// TestEnvMissingBindingNameFailsLoud asserts that a binding.Env name with
// no orchestrator value returns ErrMissingRequiredEnv. Per F.7.17 P5 the
// dispatcher routes this to pre-lock failure, so we MUST surface it
// from BuildCommand without producing a Cmd.
func TestEnvMissingBindingNameFailsLoud(t *testing.T) {
	// NOT t.Parallel() — we mutate process env via Unsetenv.

	const missingName = "TILLSYN_TEST_DEFINITELY_UNSET"
	if err := os.Unsetenv(missingName); err != nil {
		t.Fatalf("os.Unsetenv: %v", err)
	}

	binding := dispatcher.BindingResolved{
		AgentName: "go-builder-agent",
		CLIKind:   dispatcher.CLIKindClaude,
		Env:       []string{missingName},
	}

	a := New()
	cmd, err := a.BuildCommand(context.Background(), binding, minimalPaths(t))
	if err == nil {
		t.Fatalf("BuildCommand returned nil error for missing binding env var; want ErrMissingRequiredEnv")
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

// TestEnvOSEnvironNotInherited asserts that orchestrator env vars NOT in
// the closed baseline AND NOT in binding.Env do not leak through to the
// emitted cmd.Env. This is the load-bearing F.7.17 L8 isolation guarantee
// — direnv-style secret-bearing vars (AWS_ACCESS_KEY_ID, etc.) MUST NOT
// reach the spawned claude process.
func TestEnvOSEnvironNotInherited(t *testing.T) {
	// NOT t.Parallel() — we mutate process env.

	const sentinelName = "TILLSYN_TEST_LEAK_SENTINEL_xyzzy"
	const sentinelValue = "must-not-leak"
	t.Setenv(sentinelName, sentinelValue)

	binding := dispatcher.BindingResolved{
		AgentName: "go-builder-agent",
		CLIKind:   dispatcher.CLIKindClaude,
		// Crucially: sentinel name NOT in binding.Env.
	}

	a := New()
	cmd, err := a.BuildCommand(context.Background(), binding, minimalPaths(t))
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

// TestEnvNotInheritedFromOSEnviron is a stricter sibling of
// TestEnvOSEnvironNotInherited: it asserts cmd.Env LENGTH is bounded by
// (closed-baseline-set + binding.Env) — no extras possible. Catches the
// regression where someone wires `cmd.Env = append(os.Environ(), ...)`.
func TestEnvNotInheritedFromOSEnviron(t *testing.T) {
	// NOT t.Parallel() — we mutate process env.

	// Set a known value for every baseline name + one binding-only name.
	for _, name := range closedBaselineEnvNames {
		t.Setenv(name, "BASE_"+name)
	}
	t.Setenv("TILLSYN_TEST_BINDING_ONLY", "binding-only-val")
	// Set a third sentinel that is in NEITHER list.
	t.Setenv("TILLSYN_TEST_OUTSIDER", "outsider-val")

	binding := dispatcher.BindingResolved{
		AgentName: "go-builder-agent",
		CLIKind:   dispatcher.CLIKindClaude,
		Env:       []string{"TILLSYN_TEST_BINDING_ONLY"},
	}

	a := New()
	cmd, err := a.BuildCommand(context.Background(), binding, minimalPaths(t))
	if err != nil {
		t.Fatalf("BuildCommand: %v", err)
	}

	// Expected size: every baseline name (each set above so all emit) + 1
	// binding-only name.
	wantSize := len(closedBaselineEnvNames) + 1
	if got := len(cmd.Env); got != wantSize {
		t.Fatalf("cmd.Env size = %d; want %d (baseline %d + binding 1). Env=%v",
			got, wantSize, len(closedBaselineEnvNames), cmd.Env)
	}

	// And specifically the outsider must not be present.
	for _, line := range cmd.Env {
		if strings.HasPrefix(line, "TILLSYN_TEST_OUTSIDER=") {
			t.Fatalf("outsider env unexpectedly in cmd.Env: %q", line)
		}
	}
}

// TestParseStreamEventSystemInit asserts the system/init line decodes to
// canonical Type "system_init" with Subtype "init" and IsTerminal=false.
func TestParseStreamEventSystemInit(t *testing.T) {
	t.Parallel()

	line := []byte(`{"type":"system","subtype":"init","cwd":"/tmp","model":"opus"}`)
	ev, err := parseStreamEvent(line)
	if err != nil {
		t.Fatalf("parseStreamEvent: %v", err)
	}
	if ev.Type != "system_init" {
		t.Errorf("Type = %q; want %q", ev.Type, "system_init")
	}
	if ev.Subtype != "init" {
		t.Errorf("Subtype = %q; want %q", ev.Subtype, "init")
	}
	if ev.IsTerminal {
		t.Errorf("IsTerminal = true; want false on system_init")
	}
	if len(ev.Raw) == 0 {
		t.Errorf("Raw is empty; want retained line bytes")
	}
}

// TestParseStreamEventAssistantWithTextAndToolUse asserts the assistant
// event surfaces the first text block as ev.Text and the first tool_use
// block as ev.ToolName + ev.ToolInput.
func TestParseStreamEventAssistantWithTextAndToolUse(t *testing.T) {
	t.Parallel()

	line := []byte(`{"type":"assistant","message":{"content":[` +
		`{"type":"thinking","thinking":"plan"},` +
		`{"type":"text","text":"hello"},` +
		`{"type":"tool_use","name":"Read","input":{"file_path":"/x"}}` +
		`]}}`)

	ev, err := parseStreamEvent(line)
	if err != nil {
		t.Fatalf("parseStreamEvent: %v", err)
	}
	if ev.Type != "assistant" {
		t.Fatalf("Type = %q; want %q", ev.Type, "assistant")
	}
	if ev.Text != "hello" {
		t.Errorf("Text = %q; want %q", ev.Text, "hello")
	}
	if ev.ToolName != "Read" {
		t.Errorf("ToolName = %q; want %q", ev.ToolName, "Read")
	}
	var toolInput map[string]string
	if err := json.Unmarshal(ev.ToolInput, &toolInput); err != nil {
		t.Fatalf("ev.ToolInput unmarshal: %v (raw=%s)", err, string(ev.ToolInput))
	}
	if got, want := toolInput["file_path"], "/x"; got != want {
		t.Errorf("ToolInput.file_path = %q; want %q", got, want)
	}
	if ev.IsTerminal {
		t.Errorf("IsTerminal = true; want false on assistant")
	}
}

// TestParseStreamEventUserToolResult asserts the user/tool_result line
// maps to Type="user" and IsTerminal=false. Tool-level detail stays in
// Raw for downstream consumers.
func TestParseStreamEventUserToolResult(t *testing.T) {
	t.Parallel()

	line := []byte(`{"type":"user","message":{"content":[{"type":"tool_result","tool_use_id":"tu","content":"ok","is_error":false}]}}`)
	ev, err := parseStreamEvent(line)
	if err != nil {
		t.Fatalf("parseStreamEvent: %v", err)
	}
	if ev.Type != "user" {
		t.Errorf("Type = %q; want %q", ev.Type, "user")
	}
	if ev.IsTerminal {
		t.Errorf("IsTerminal = true; want false on user event")
	}
}

// TestParseStreamEventResultTerminal asserts the result event sets
// IsTerminal=true so callers know to invoke ExtractTerminalReport.
func TestParseStreamEventResultTerminal(t *testing.T) {
	t.Parallel()

	line := []byte(`{"type":"result","subtype":"success","is_error":false,"total_cost_usd":0.01}`)
	ev, err := parseStreamEvent(line)
	if err != nil {
		t.Fatalf("parseStreamEvent: %v", err)
	}
	if ev.Type != "result" {
		t.Errorf("Type = %q; want %q", ev.Type, "result")
	}
	if !ev.IsTerminal {
		t.Errorf("IsTerminal = false; want true on result event")
	}
	if ev.Subtype != "success" {
		t.Errorf("Subtype = %q; want %q", ev.Subtype, "success")
	}
}

// TestParseStreamEventMalformedJSON asserts that a non-JSON line returns
// a wrapped ErrMalformedStreamLine. The returned StreamEvent retains Raw
// so callers can log the offending line.
func TestParseStreamEventMalformedJSON(t *testing.T) {
	t.Parallel()

	line := []byte(`not-json {{{`)
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

// TestParseStreamEventMissingType asserts the discriminator-validation
// path: a JSON object without a "type" field is malformed.
func TestParseStreamEventMissingType(t *testing.T) {
	t.Parallel()

	line := []byte(`{"subtype":"init"}`)
	_, err := parseStreamEvent(line)
	if err == nil {
		t.Fatalf("parseStreamEvent on missing-type returned nil error")
	}
	if !errors.Is(err, ErrMalformedStreamLine) {
		t.Errorf("error chain does not wrap ErrMalformedStreamLine: %v", err)
	}
}

// TestParseStreamEventUnknownType pins the forward-compat behavior: an
// unknown type passes through with Type set to the raw discriminator
// string. Future event families must not crash existing readers.
func TestParseStreamEventUnknownType(t *testing.T) {
	t.Parallel()

	line := []byte(`{"type":"future_kind","payload":42}`)
	ev, err := parseStreamEvent(line)
	if err != nil {
		t.Fatalf("parseStreamEvent: %v", err)
	}
	if ev.Type != "future_kind" {
		t.Errorf("Type = %q; want %q", ev.Type, "future_kind")
	}
	if ev.IsTerminal {
		t.Errorf("IsTerminal = true; want false on unknown type")
	}
}

// TestExtractTerminalReportPopulated asserts the terminal-report decoder
// correctly surfaces cost, denials, reason, and errors from a recorded
// result event.
func TestExtractTerminalReportPopulated(t *testing.T) {
	t.Parallel()

	resultLine := []byte(`{"type":"result","subtype":"success","total_cost_usd":0.0123,"terminal_reason":"completed","permission_denials":[{"tool_name":"Bash","tool_input":{"command":"curl evil.com"}}],"errors":["something happened"]}`)

	ev, err := parseStreamEvent(resultLine)
	if err != nil {
		t.Fatalf("parseStreamEvent: %v", err)
	}
	report, ok := extractTerminalReport(ev)
	if !ok {
		t.Fatalf("extractTerminalReport returned ok=false on terminal event")
	}
	if report.Cost == nil {
		t.Fatalf("report.Cost = nil; want non-nil pointer to 0.0123")
	}
	if got, want := *report.Cost, 0.0123; got != want {
		t.Errorf("*report.Cost = %v; want %v", got, want)
	}
	if report.Reason != "completed" {
		t.Errorf("report.Reason = %q; want %q", report.Reason, "completed")
	}
	if len(report.Denials) != 1 {
		t.Fatalf("len(report.Denials) = %d; want 1", len(report.Denials))
	}
	if got, want := report.Denials[0].ToolName, "Bash"; got != want {
		t.Errorf("Denials[0].ToolName = %q; want %q", got, want)
	}
	var denialInput map[string]string
	if err := json.Unmarshal(report.Denials[0].ToolInput, &denialInput); err != nil {
		t.Fatalf("Denials[0].ToolInput unmarshal: %v", err)
	}
	if got, want := denialInput["command"], "curl evil.com"; got != want {
		t.Errorf("Denials[0].ToolInput.command = %q; want %q", got, want)
	}
	if !reflect.DeepEqual(report.Errors, []string{"something happened"}) {
		t.Errorf("report.Errors = %v; want [\"something happened\"]", report.Errors)
	}
}

// TestExtractTerminalReportNoCost asserts that a result event without
// total_cost_usd produces TerminalReport.Cost == nil. This is the F.7.17
// L11 nil-cost-clean-degradation case: callers MUST NOT mistake nil for
// zero-cost.
func TestExtractTerminalReportNoCost(t *testing.T) {
	t.Parallel()

	resultLine := []byte(`{"type":"result","subtype":"success","terminal_reason":"completed"}`)
	ev, err := parseStreamEvent(resultLine)
	if err != nil {
		t.Fatalf("parseStreamEvent: %v", err)
	}
	report, ok := extractTerminalReport(ev)
	if !ok {
		t.Fatalf("extractTerminalReport ok=false on terminal event")
	}
	if report.Cost != nil {
		t.Errorf("report.Cost = %v; want nil (no total_cost_usd in event)", *report.Cost)
	}
}

// TestExtractTerminalReportNonTerminalReturnsZeroFalse asserts the
// (zero, false) contract for non-terminal events. Callers gate
// extraction on this bool so misuse on assistant / user events MUST
// produce an unmistakable zero return.
func TestExtractTerminalReportNonTerminalReturnsZeroFalse(t *testing.T) {
	t.Parallel()

	assistantLine := []byte(`{"type":"assistant","message":{"content":[{"type":"text","text":"hi"}]}}`)
	ev, err := parseStreamEvent(assistantLine)
	if err != nil {
		t.Fatalf("parseStreamEvent: %v", err)
	}
	report, ok := extractTerminalReport(ev)
	if ok {
		t.Fatalf("extractTerminalReport ok=true on non-terminal event")
	}
	zero := dispatcher.TerminalReport{}
	if !reflect.DeepEqual(report, zero) {
		t.Errorf("report = %+v; want zero TerminalReport", report)
	}
}

// TestRecordedFixtureRoundTrip asserts that the recorded fixture
// testdata/claude_stream_minimal.jsonl round-trips through the parser:
// every line decodes without error, the terminal event surfaces a
// populated TerminalReport with cost + a denial.
func TestRecordedFixtureRoundTrip(t *testing.T) {
	t.Parallel()

	path := filepath.Join("testdata", "claude_stream_minimal.jsonl")
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open fixture: %v", err)
	}
	t.Cleanup(func() { _ = f.Close() })

	var (
		terminalEv      dispatcher.StreamEvent
		sawTerminal     bool
		assistantEvText string
	)
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}
		ev, err := parseStreamEvent(line)
		if err != nil {
			t.Fatalf("parseStreamEvent on fixture line %q: %v", string(line), err)
		}
		switch ev.Type {
		case "assistant":
			if assistantEvText == "" && ev.Text != "" {
				assistantEvText = ev.Text
			}
		case "result":
			terminalEv = ev
			sawTerminal = true
		}
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("scanner: %v", err)
	}
	if !sawTerminal {
		t.Fatal("fixture has no result event")
	}
	if assistantEvText != "hello world" {
		t.Errorf("assistant text = %q; want %q", assistantEvText, "hello world")
	}

	report, ok := extractTerminalReport(terminalEv)
	if !ok {
		t.Fatal("extractTerminalReport ok=false on fixture terminal event")
	}
	if report.Cost == nil {
		t.Fatal("report.Cost = nil on fixture")
	}
	if got, want := *report.Cost, 0.0123; got != want {
		t.Errorf("*report.Cost = %v; want %v", got, want)
	}
	if report.Reason != "completed" {
		t.Errorf("report.Reason = %q; want %q", report.Reason, "completed")
	}
	if len(report.Denials) != 1 {
		t.Fatalf("len(report.Denials) = %d; want 1", len(report.Denials))
	}
	if got := report.Denials[0].ToolName; got != "Bash" {
		t.Errorf("Denials[0].ToolName = %q; want %q", got, "Bash")
	}
}

// --- helpers -------------------------------------------------------------

// minimalBinding returns a binding suitable for argv tests that don't
// care about per-spawn knobs. The CLIKind is set so the resolver
// invariant is honored even though this test isn't going through it.
func minimalBinding(_ *testing.T) dispatcher.BindingResolved {
	return dispatcher.BindingResolved{
		AgentName: "go-builder-agent",
		CLIKind:   dispatcher.CLIKindClaude,
	}
}

// minimalPaths returns a BundlePaths handle pointing at conventional
// locations under a fake bundle root.
func minimalPaths(_ *testing.T) dispatcher.BundlePaths {
	return dispatcher.BundlePaths{
		Root:             "/tmp/spawn-xyz",
		SystemPromptPath: "/tmp/spawn-xyz/system-prompt.md",
	}
}

// firstArg returns args[0] or "<empty>" so error messages don't panic on
// zero-length argv slices.
func firstArg(args []string) string {
	if len(args) == 0 {
		return "<empty>"
	}
	return args[0]
}

// hasArg reports whether args contains exact-match flag.
func hasArg(args []string, flag string) bool {
	for _, a := range args {
		if a == flag {
			return true
		}
	}
	return false
}

// expectFlagPresent asserts flag appears in args, and (when wantValue is
// non-empty) the immediately-following arg equals wantValue. wantValue ==
// "" means "flag is a bare switch with no following argument" — but
// callers MUST be careful: -p "" is also a flag with empty argument, so
// the test for -p uses the no-following-arg path documented at the top of
// the test cases below.
func expectFlagPresent(t *testing.T, args []string, flag, wantValue string) {
	t.Helper()
	for i, a := range args {
		if a != flag {
			continue
		}
		if wantValue == "" {
			// Two valid shapes: (a) bare switch — no arg expected; (b) flag
			// with explicit empty arg (-p "" or --setting-sources ""). For
			// (b) we still assert the next slot is empty string.
			switch flag {
			case "-p", "--setting-sources":
				if i+1 >= len(args) || args[i+1] != "" {
					t.Errorf("flag %q: expected empty next arg; argv=%v", flag, args)
				}
			default:
				// Bare switches like --bare, --strict-mcp-config: just
				// presence is enough.
			}
			return
		}
		if i+1 >= len(args) {
			t.Errorf("flag %q present but no following argument; argv=%v", flag, args)
			return
		}
		if got := args[i+1]; got != wantValue {
			t.Errorf("flag %q value = %q; want %q", flag, got, wantValue)
		}
		return
	}
	t.Errorf("flag %q missing from argv; got %v", flag, args)
}

// envSliceToMap parses a cmd.Env slice into NAME → value map for
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

// sortedKeys returns keys of m sorted lexicographically (helper for
// deterministic logging in failing assertions).
func sortedKeys(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// silenceUnused references rarely-used helpers so the compiler doesn't
// flag them when an assertion path never fires. Keeps the helper surface
// stable as new test cases land.
var _ = sortedKeys
