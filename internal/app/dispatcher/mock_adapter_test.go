package dispatcher

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"
)

// mock_adapter_test.go ships the F.7.17.4 MockAdapter contract test fixture.
// MockAdapter is TEST-ONLY: it lives in a _test.go file so it is invisible to
// production code, satisfies the CLIAdapter interface declared in
// cli_adapter.go, and exercises the contract WITHOUT touching real claude /
// codex binaries. Per F.7.17 master PLAN L19 + F.7.17 REV-6 acceptance, the
// fixture is the load-bearing proof that the CLIAdapter seam is multi-adapter
// ready BEFORE the real claudeAdapter (droplet 4c.F.7.17.3) and codexAdapter
// (Drop 4d) plug in.
//
// Recorded fixture: testdata/mock_stream_minimal.jsonl is a 3-line JSONL
// trace covering two non-terminal mock_chunk events plus one mock_terminal
// event with cost / denials / reason / errors populated.
//
// Forward-collision note: the F.7.17 plan body originally named this file
// cli_adapter_test.go, but droplet 4c.F.7.17.2 took that name first. The
// orchestrator routed this droplet to mock_adapter_test.go, which is also
// more descriptive.

// mockTerminalPayload is the adapter-private wire shape MockAdapter decodes
// out of the recorded fixture's terminal event. Adapters keep their CLI's
// payload struct private so the cross-CLI canonical TerminalReport stays
// narrow; this mirrors what claudeAdapter (4c.F.7.17.3) will do for claude's
// "result" event.
type mockTerminalPayload struct {
	Type    string              `json:"type"`
	Cost    *float64            `json:"cost,omitempty"`
	Reason  string              `json:"reason,omitempty"`
	Denials []mockDenialPayload `json:"denials,omitempty"`
	Errors  []string            `json:"errors,omitempty"`
}

// mockDenialPayload is the adapter-private wire shape for one tool denial.
type mockDenialPayload struct {
	ToolName  string          `json:"tool_name"`
	ToolInput json.RawMessage `json:"tool_input"`
}

// mockChunkPayload is the adapter-private wire shape for non-terminal events.
type mockChunkPayload struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// mockBuildCommandCall records the args MockAdapter.BuildCommand was invoked
// with. Tests inspect calls via the MockAdapter.Calls() accessor.
type mockBuildCommandCall struct {
	Binding BindingResolved
	Paths   BundlePaths
}

// MockAdapter is a CLIAdapter that returns deterministic stub commands and
// parses the mock_chunk / mock_terminal JSONL taxonomy. It records every
// BuildCommand invocation so tests can assert on the args verbatim.
//
// MockAdapter is intentionally exported (capital M) inside this _test.go
// file so the package's own contract tests can pass it through []CLIAdapter
// table rows; because the file is a _test.go, the symbol is invisible to
// production code and to other packages' production builds.
type MockAdapter struct {
	mu    sync.Mutex
	calls []mockBuildCommandCall
}

// newMockAdapter constructs a fresh MockAdapter with an empty call log.
func newMockAdapter() *MockAdapter {
	return &MockAdapter{}
}

// BuildCommand returns a stub *exec.Cmd that runs /bin/true (a POSIX no-op)
// with the spawn's BundlePaths.Root and binding's AgentName threaded through
// Args verbatim so tests can inspect the passthrough. Env is set explicitly
// per F.7.17 L8 (os.Environ NOT inherited) — MockAdapter copies a tiny closed
// baseline plus the binding's resolved Env names so contract tests can
// confirm the seam respects L8 without depending on the production
// claudeAdapter's full L6 baseline.
//
// The command is NEVER executed by the contract tests — they only inspect
// *exec.Cmd.Path / .Args / .Env. MockAdapter therefore does not need to
// guarantee /bin/true is present on the host (though it is on macOS + Linux,
// and Windows is out of scope per F.7.17 L9).
func (m *MockAdapter) BuildCommand(ctx context.Context, binding BindingResolved, paths BundlePaths) (*exec.Cmd, error) {
	if ctx == nil {
		return nil, errors.New("MockAdapter.BuildCommand: nil context")
	}
	if paths.Root == "" {
		return nil, errors.New("MockAdapter.BuildCommand: BundlePaths.Root must be set")
	}

	args := []string{
		"--mock-flag", "fixture-value",
		"--bundle-root", paths.Root,
		"--agent-name", binding.AgentName,
	}
	cmd := exec.CommandContext(ctx, "/bin/true", args...)

	// Closed minimal Env baseline — analogous to F.7.17 L6 but trimmed for
	// fixture purposes. The contract test asserts the resolved binding's
	// Env names get materialized via os.Getenv.
	env := []string{"PATH=" + os.Getenv("PATH")}
	for _, name := range binding.Env {
		env = append(env, name+"="+os.Getenv(name))
	}
	cmd.Env = env

	m.mu.Lock()
	m.calls = append(m.calls, mockBuildCommandCall{Binding: binding, Paths: paths})
	m.mu.Unlock()

	return cmd, nil
}

// ParseStreamEvent decodes one JSONL line into the canonical StreamEvent
// shape. The MockAdapter taxonomy has exactly two event types:
//
//   - "mock_chunk":    non-terminal; carries Text payload.
//   - "mock_terminal": terminal;     carries cost / denials / reason / errors.
//
// Any other type is accepted but produces a non-terminal StreamEvent with
// IsTerminal=false — adapters are tolerant of unknown event subtypes per the
// F.7.17 design note that monitor stays CLI-agnostic via Type+IsTerminal
// routing only.
func (m *MockAdapter) ParseStreamEvent(line []byte) (StreamEvent, error) {
	// Strip a single trailing newline if present so the caller can hand us
	// raw scanner output OR a bytes-delimited slice without thinking.
	trimmed := bytes.TrimRight(line, "\n")
	if len(trimmed) == 0 {
		return StreamEvent{}, errors.New("MockAdapter.ParseStreamEvent: empty line")
	}

	// First decode the type discriminator only. We keep Raw as the original
	// (un-trimmed-newline-aware) bytes so ExtractTerminalReport can re-decode
	// the adapter-private fields.
	var probe struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(trimmed, &probe); err != nil {
		return StreamEvent{}, fmt.Errorf("MockAdapter.ParseStreamEvent: decode type discriminator: %w", err)
	}

	// json.RawMessage requires a copy because trimmed aliases the caller's
	// buffer — the caller may reuse it for the next line.
	rawCopy := make(json.RawMessage, len(trimmed))
	copy(rawCopy, trimmed)

	switch probe.Type {
	case "mock_terminal":
		return StreamEvent{
			Type:       "mock_terminal",
			IsTerminal: true,
			Raw:        rawCopy,
		}, nil
	case "mock_chunk":
		var chunk mockChunkPayload
		if err := json.Unmarshal(trimmed, &chunk); err != nil {
			return StreamEvent{}, fmt.Errorf("MockAdapter.ParseStreamEvent: decode mock_chunk: %w", err)
		}
		return StreamEvent{
			Type:       "mock_chunk",
			IsTerminal: false,
			Text:       chunk.Text,
			Raw:        rawCopy,
		}, nil
	default:
		// Unknown event type: pass through as non-terminal with the type
		// preserved so the monitor can still log it.
		return StreamEvent{
			Type:       probe.Type,
			IsTerminal: false,
			Raw:        rawCopy,
		}, nil
	}
}

// ExtractTerminalReport pulls the TerminalReport out of a parsed StreamEvent.
// Returns (TerminalReport{}, false) when the event is not a mock_terminal —
// per F.7.17 L11 the bool return signals "this is the terminal event,"
// distinct from the Cost-pointer's "this CLI emitted a cost number" signal.
//
// When the underlying terminal payload omits the cost field, Cost stays nil
// (NOT a pointer to 0.0). Tests assert this contract directly.
func (m *MockAdapter) ExtractTerminalReport(ev StreamEvent) (TerminalReport, bool) {
	if !ev.IsTerminal {
		return TerminalReport{}, false
	}
	if ev.Type != "mock_terminal" {
		return TerminalReport{}, false
	}

	var payload mockTerminalPayload
	if err := json.Unmarshal(ev.Raw, &payload); err != nil {
		// A malformed terminal payload still counts AS the terminal event
		// (so the monitor stops reading the stream), but the report is
		// empty and Errors carries the decode failure so post-mortem can
		// see what happened.
		return TerminalReport{
			Errors: []string{fmt.Sprintf("MockAdapter.ExtractTerminalReport: decode mock_terminal: %v", err)},
		}, true
	}

	denials := make([]ToolDenial, 0, len(payload.Denials))
	for _, d := range payload.Denials {
		denials = append(denials, ToolDenial{
			ToolName:  d.ToolName,
			ToolInput: d.ToolInput,
		})
	}
	if len(denials) == 0 {
		denials = nil
	}

	return TerminalReport{
		Cost:    payload.Cost,
		Denials: denials,
		Reason:  payload.Reason,
		Errors:  payload.Errors,
	}, true
}

// Calls returns a snapshot of every BuildCommand invocation MockAdapter has
// recorded. The slice is a copy — callers may mutate it freely.
func (m *MockAdapter) Calls() []mockBuildCommandCall {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]mockBuildCommandCall, len(m.calls))
	copy(out, m.calls)
	return out
}

// _ is the compile-time assertion that MockAdapter satisfies the CLIAdapter
// interface. If the interface drifts, this line fails to compile and every
// downstream test fails loud rather than silently mis-asserting.
var _ CLIAdapter = (*MockAdapter)(nil)

// TestMockAdapterBuildCommand asserts BuildCommand returns a *exec.Cmd whose
// Path / Args / Env reflect the bundle paths + binding the dispatcher hands
// in. Per F.7.17 L8 the cmd's Env is set explicitly (not inherited from
// os.Environ) — the test confirms the L8 isolation property by checking a
// secret-bearing variable from the parent environment is NOT carried through
// unless the binding's Env list opts it in.
func TestMockAdapterBuildCommand(t *testing.T) {
	// NOTE: NOT t.Parallel — t.Setenv below is incompatible with t.Parallel
	// per testing.checkParallel. The other adapter tests stay parallel.

	// Set a secret-bearing parent env var; assert it does NOT leak into
	// cmd.Env unless the binding declares it.
	t.Setenv("TILLSYN_MOCK_SECRET_NEVER_FORWARDED", "leaked")
	t.Setenv("TILLSYN_MOCK_DECLARED", "declared-value")

	adapter := newMockAdapter()
	binding := BindingResolved{
		AgentName: "go-builder-agent",
		CLIKind:   CLIKindClaude, // CLIKind is closed; mock reuses claude as label since no CLIKindMock exists
		Env:       []string{"TILLSYN_MOCK_DECLARED"},
	}
	paths := BundlePaths{
		Root:             "/tmp/tillsyn-mock-spawn-1",
		SystemPromptPath: "/tmp/tillsyn-mock-spawn-1/system-prompt.md",
		StreamLogPath:    "/tmp/tillsyn-mock-spawn-1/stream.jsonl",
		ManifestPath:     "/tmp/tillsyn-mock-spawn-1/manifest.json",
	}

	cmd, err := adapter.BuildCommand(context.Background(), binding, paths)
	if err != nil {
		t.Fatalf("BuildCommand returned err = %v; want nil", err)
	}
	if cmd == nil {
		t.Fatalf("BuildCommand returned nil *exec.Cmd; want non-nil")
	}

	if cmd.Path != "/bin/true" {
		t.Errorf("cmd.Path = %q; want %q", cmd.Path, "/bin/true")
	}

	wantArgsTail := []string{
		"--mock-flag", "fixture-value",
		"--bundle-root", "/tmp/tillsyn-mock-spawn-1",
		"--agent-name", "go-builder-agent",
	}
	if len(cmd.Args) < 1+len(wantArgsTail) {
		t.Fatalf("cmd.Args has %d elements; want at least %d (program + %v)", len(cmd.Args), 1+len(wantArgsTail), wantArgsTail)
	}
	for i, want := range wantArgsTail {
		got := cmd.Args[1+i]
		if got != want {
			t.Errorf("cmd.Args[%d] = %q; want %q", 1+i, got, want)
		}
	}

	// Env isolation: declared name forwarded; undeclared name suppressed.
	envMap := make(map[string]string, len(cmd.Env))
	for _, kv := range cmd.Env {
		// Split on first '='.
		for i := 0; i < len(kv); i++ {
			if kv[i] == '=' {
				envMap[kv[:i]] = kv[i+1:]
				break
			}
		}
	}
	if got := envMap["TILLSYN_MOCK_DECLARED"]; got != "declared-value" {
		t.Errorf("cmd.Env[TILLSYN_MOCK_DECLARED] = %q; want %q (declared-name should be forwarded)", got, "declared-value")
	}
	if _, present := envMap["TILLSYN_MOCK_SECRET_NEVER_FORWARDED"]; present {
		t.Errorf("cmd.Env contains forbidden TILLSYN_MOCK_SECRET_NEVER_FORWARDED=%q (L8 isolation violated)", envMap["TILLSYN_MOCK_SECRET_NEVER_FORWARDED"])
	}
	if _, present := envMap["PATH"]; !present {
		t.Errorf("cmd.Env missing PATH (closed baseline forwarding broken)")
	}

	// Call recording: BuildCommand was invoked once with the supplied args.
	calls := adapter.Calls()
	if len(calls) != 1 {
		t.Fatalf("adapter.Calls() returned %d entries; want 1", len(calls))
	}
	if calls[0].Binding.AgentName != "go-builder-agent" {
		t.Errorf("recorded binding AgentName = %q; want %q", calls[0].Binding.AgentName, "go-builder-agent")
	}
	if calls[0].Paths.Root != "/tmp/tillsyn-mock-spawn-1" {
		t.Errorf("recorded paths.Root = %q; want %q", calls[0].Paths.Root, "/tmp/tillsyn-mock-spawn-1")
	}
}

// TestMockAdapterBuildCommandRejectsBadInput pins the defensive-validation
// contract: nil context and zero-value BundlePaths.Root return wrapped errors
// rather than producing a half-baked *exec.Cmd. The dispatcher relies on
// BuildCommand failing loud for these cases per the F.7.17.3 acceptance
// criteria pattern (claudeAdapter applies the same rule).
func TestMockAdapterBuildCommandRejectsBadInput(t *testing.T) {
	t.Parallel()

	adapter := newMockAdapter()

	if _, err := adapter.BuildCommand(nil, BindingResolved{}, BundlePaths{Root: "/tmp/x"}); err == nil {
		t.Errorf("BuildCommand(nil ctx, ...) returned nil err; want non-nil")
	}
	if _, err := adapter.BuildCommand(context.Background(), BindingResolved{}, BundlePaths{}); err == nil {
		t.Errorf("BuildCommand(_, _, BundlePaths{}) returned nil err; want non-nil (Root required)")
	}
}

// TestMockAdapterParseStreamEventChunkAndTerminal round-trips the recorded
// testdata/mock_stream_minimal.jsonl fixture through ParseStreamEvent. The
// fixture is 3 lines: 2 mock_chunk + 1 mock_terminal. The test asserts that
// IsTerminal flips correctly between non-terminal and terminal events, that
// the first two events carry their Text payload, and that Raw is preserved
// verbatim (so ExtractTerminalReport can re-decode adapter-private fields).
func TestMockAdapterParseStreamEventChunkAndTerminal(t *testing.T) {
	t.Parallel()

	adapter := newMockAdapter()
	fixturePath := filepath.Join("testdata", "mock_stream_minimal.jsonl")
	f, err := os.Open(fixturePath)
	if err != nil {
		t.Fatalf("open fixture %q: %v", fixturePath, err)
	}
	t.Cleanup(func() { _ = f.Close() })

	scanner := bufio.NewScanner(f)
	var events []StreamEvent
	for scanner.Scan() {
		line := scanner.Bytes()
		ev, perr := adapter.ParseStreamEvent(line)
		if perr != nil {
			t.Fatalf("ParseStreamEvent line %q: %v", string(line), perr)
		}
		events = append(events, ev)
	}
	if scanErr := scanner.Err(); scanErr != nil {
		t.Fatalf("scanner.Err: %v", scanErr)
	}
	if len(events) != 3 {
		t.Fatalf("parsed %d events; want 3", len(events))
	}

	// Lines 1+2: non-terminal mock_chunk events.
	for i, ev := range events[:2] {
		if ev.Type != "mock_chunk" {
			t.Errorf("events[%d].Type = %q; want %q", i, ev.Type, "mock_chunk")
		}
		if ev.IsTerminal {
			t.Errorf("events[%d].IsTerminal = true; want false", i)
		}
		if ev.Text == "" {
			t.Errorf("events[%d].Text empty; want populated mock_chunk text", i)
		}
		if len(ev.Raw) == 0 {
			t.Errorf("events[%d].Raw empty; want preserved JSONL line", i)
		}
	}

	// Line 3: terminal mock_terminal event.
	terminal := events[2]
	if terminal.Type != "mock_terminal" {
		t.Errorf("events[2].Type = %q; want %q", terminal.Type, "mock_terminal")
	}
	if !terminal.IsTerminal {
		t.Errorf("events[2].IsTerminal = false; want true")
	}
	if len(terminal.Raw) == 0 {
		t.Errorf("events[2].Raw empty; want preserved JSONL line")
	}
}

// TestMockAdapterParseStreamEventMalformedJSON asserts a malformed JSONL line
// produces a wrapped error rather than a half-decoded StreamEvent. Tolerance
// for malformed lines is the monitor's responsibility (existing F.7.4 logic
// logs and continues); the adapter contract is to fail loud at parse time.
func TestMockAdapterParseStreamEventMalformedJSON(t *testing.T) {
	t.Parallel()

	adapter := newMockAdapter()
	cases := [][]byte{
		[]byte(""),
		[]byte("\n"),
		[]byte("{not json"),
		[]byte(`{"type": 12345}`), // wrong type for the discriminator
	}
	for _, line := range cases {
		ev, err := adapter.ParseStreamEvent(line)
		if err == nil {
			t.Errorf("ParseStreamEvent(%q) returned nil err; want non-nil. ev=%+v", string(line), ev)
		}
	}
}

// TestMockAdapterExtractTerminalReportPopulatedTerminal asserts a terminal
// event with cost / denials / reason / errors populates a non-zero
// TerminalReport. The fixture's terminal line carries:
//
//   - cost = 0.5 (Cost should be a non-nil pointer to 0.5)
//   - reason = "ok"
//   - one denial (Bash + tool_input)
//   - errors = [] (empty)
func TestMockAdapterExtractTerminalReportPopulatedTerminal(t *testing.T) {
	t.Parallel()

	adapter := newMockAdapter()
	terminalLine := []byte(`{"type":"mock_terminal","cost":0.5,"reason":"ok","denials":[{"tool_name":"Bash","tool_input":{"cmd":"rm -rf /"}}],"errors":[]}`)

	ev, err := adapter.ParseStreamEvent(terminalLine)
	if err != nil {
		t.Fatalf("ParseStreamEvent terminal: %v", err)
	}
	report, ok := adapter.ExtractTerminalReport(ev)
	if !ok {
		t.Fatalf("ExtractTerminalReport returned ok=false on terminal event")
	}

	if report.Cost == nil {
		t.Fatalf("report.Cost = nil; want non-nil pointer to 0.5")
	}
	if got, want := *report.Cost, 0.5; got != want {
		t.Errorf("*report.Cost = %v; want %v", got, want)
	}
	if report.Reason != "ok" {
		t.Errorf("report.Reason = %q; want %q", report.Reason, "ok")
	}
	if len(report.Denials) != 1 {
		t.Fatalf("len(report.Denials) = %d; want 1", len(report.Denials))
	}
	if report.Denials[0].ToolName != "Bash" {
		t.Errorf("report.Denials[0].ToolName = %q; want %q", report.Denials[0].ToolName, "Bash")
	}
	if len(report.Denials[0].ToolInput) == 0 {
		t.Errorf("report.Denials[0].ToolInput empty; want preserved raw JSON")
	}
}

// TestMockAdapterExtractTerminalReportNonTerminalReturnsFalse asserts the
// (TerminalReport{}, false) contract for non-terminal events. Per L11 the
// bool is the "this IS the terminal event" signal — the dispatcher monitor
// keeps reading the stream until it flips.
func TestMockAdapterExtractTerminalReportNonTerminalReturnsFalse(t *testing.T) {
	t.Parallel()

	adapter := newMockAdapter()
	chunkLine := []byte(`{"type":"mock_chunk","text":"mid-stream"}`)

	ev, err := adapter.ParseStreamEvent(chunkLine)
	if err != nil {
		t.Fatalf("ParseStreamEvent chunk: %v", err)
	}
	report, ok := adapter.ExtractTerminalReport(ev)
	if ok {
		t.Errorf("ExtractTerminalReport on non-terminal returned ok=true; want false")
	}
	// TerminalReport contains slice fields, so '!=' isn't valid; assert each
	// field's zero-ness explicitly.
	if report.Cost != nil {
		t.Errorf("non-terminal report.Cost = %v; want nil", report.Cost)
	}
	if report.Reason != "" {
		t.Errorf("non-terminal report.Reason = %q; want empty", report.Reason)
	}
	if len(report.Denials) != 0 {
		t.Errorf("non-terminal report.Denials = %v; want empty", report.Denials)
	}
	if len(report.Errors) != 0 {
		t.Errorf("non-terminal report.Errors = %v; want empty", report.Errors)
	}
}

// TestMockAdapterExtractTerminalReportCostNilWhenAbsent pins the Cost-pointer
// semantic from F.7.17 L11: a terminal event WITHOUT a cost field produces
// Cost == nil (NOT a pointer to 0.0). This is the load-bearing distinction
// for adapters whose CLI does not emit cost telemetry — the dispatcher must
// be able to tell "no cost reported" apart from "cost was zero."
func TestMockAdapterExtractTerminalReportCostNilWhenAbsent(t *testing.T) {
	t.Parallel()

	adapter := newMockAdapter()
	costlessTerminal := []byte(`{"type":"mock_terminal","reason":"ok","denials":[],"errors":[]}`)

	ev, err := adapter.ParseStreamEvent(costlessTerminal)
	if err != nil {
		t.Fatalf("ParseStreamEvent: %v", err)
	}
	report, ok := adapter.ExtractTerminalReport(ev)
	if !ok {
		t.Fatalf("ExtractTerminalReport returned ok=false; want true (event IS terminal)")
	}
	if report.Cost != nil {
		t.Errorf("report.Cost non-nil (points to %v); want nil (cost field absent from terminal payload)", *report.Cost)
	}
	if report.Reason != "ok" {
		t.Errorf("report.Reason = %q; want %q", report.Reason, "ok")
	}
}

// TestCLIAdapterContractTableDriven exercises the BuildCommand →
// ParseStreamEvent → ExtractTerminalReport sequence end-to-end against every
// adapter that satisfies CLIAdapter. The droplet ships with MockAdapter as
// the only row; F.7.17.5 (claudeAdapter) extends this table when it lands.
//
// This is the load-bearing multi-adapter readiness proof per F.7.17 master
// PLAN L19 + REV-6 — if claudeAdapter snuck a claude-specific assumption
// into the contract that MockAdapter cannot satisfy, the row would not even
// type-check.
func TestCLIAdapterContractTableDriven(t *testing.T) {
	t.Parallel()

	type contractCase struct {
		name           string
		adapter        CLIAdapter
		nonTerminal    []byte
		terminal       []byte
		wantTermType   string
		wantCostPtrSet bool
		wantCostValue  float64
	}

	cases := []contractCase{
		{
			name:           "MockAdapter",
			adapter:        newMockAdapter(),
			nonTerminal:    []byte(`{"type":"mock_chunk","text":"hello"}`),
			terminal:       []byte(`{"type":"mock_terminal","cost":0.5,"reason":"ok","denials":[],"errors":[]}`),
			wantTermType:   "mock_terminal",
			wantCostPtrSet: true,
			wantCostValue:  0.5,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Step 1: BuildCommand returns a usable *exec.Cmd.
			binding := BindingResolved{
				AgentName: "contract-test-agent",
				CLIKind:   CLIKindClaude,
			}
			paths := BundlePaths{Root: "/tmp/contract-" + tc.name}
			cmd, err := tc.adapter.BuildCommand(context.Background(), binding, paths)
			if err != nil {
				t.Fatalf("BuildCommand: %v", err)
			}
			if cmd == nil {
				t.Fatalf("BuildCommand returned nil *exec.Cmd")
			}
			if cmd.Path == "" {
				t.Errorf("cmd.Path empty; adapter %q must set executable path", tc.name)
			}

			// Step 2: ParseStreamEvent on a non-terminal line returns
			// IsTerminal=false.
			nonTermEv, err := tc.adapter.ParseStreamEvent(tc.nonTerminal)
			if err != nil {
				t.Fatalf("ParseStreamEvent non-terminal: %v", err)
			}
			if nonTermEv.IsTerminal {
				t.Errorf("non-terminal event got IsTerminal=true; adapter %q misclassified", tc.name)
			}
			if nonTermEv.Type == "" {
				t.Errorf("non-terminal event has empty Type; adapter %q failed to populate", tc.name)
			}

			// Step 3a: ExtractTerminalReport on the non-terminal event
			// returns (zero, false).
			if _, ok := tc.adapter.ExtractTerminalReport(nonTermEv); ok {
				t.Errorf("ExtractTerminalReport on non-terminal returned ok=true; adapter %q violates L11 bool-return semantic", tc.name)
			}

			// Step 4: ParseStreamEvent on the terminal line returns
			// IsTerminal=true.
			termEv, err := tc.adapter.ParseStreamEvent(tc.terminal)
			if err != nil {
				t.Fatalf("ParseStreamEvent terminal: %v", err)
			}
			if !termEv.IsTerminal {
				t.Errorf("terminal event got IsTerminal=false; adapter %q misclassified", tc.name)
			}
			if termEv.Type != tc.wantTermType {
				t.Errorf("terminal event Type = %q; want %q", termEv.Type, tc.wantTermType)
			}

			// Step 5: ExtractTerminalReport on the terminal event returns
			// (populated, true). Cost-pointer semantics per F.7.17 L11.
			report, ok := tc.adapter.ExtractTerminalReport(termEv)
			if !ok {
				t.Fatalf("ExtractTerminalReport returned ok=false on terminal event; adapter %q broken", tc.name)
			}
			if tc.wantCostPtrSet {
				if report.Cost == nil {
					t.Errorf("report.Cost = nil; adapter %q should have set cost pointer", tc.name)
				} else if *report.Cost != tc.wantCostValue {
					t.Errorf("*report.Cost = %v; want %v", *report.Cost, tc.wantCostValue)
				}
			}
		})
	}
}
