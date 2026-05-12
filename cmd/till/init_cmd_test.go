package main

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/exp/teatest/v2"
)

// TestInit_BareInvocation_ReturnsTUIStubError verifies that `till init` (bare,
// no --json) routes through cobra to runInitTUI. The end-to-end run()
// invocation exercises the cobra registration in main.go — calling
// cmd.RunE or runInitTUI directly would not prove the command is wired
// into rootCmd. CONSUMER-TIE TEST CONTRACT (W2-FF6 ROUND-2) — symmetric
// to D7.5's W2-FF3 contract.
//
// **D4 update**: with runInitTUI's bubbletea walk wired, the smoke test
// stubs `programFactory` so the test never tries to open a real terminal.
// The stubbed program returns a `initTUIModel` already advanced to the
// Done step with a synthetic payload — that exercises the success branch
// of runInitTUI, which then surfaces the D5 file-copy stub error. The
// test name is preserved for git-blame continuity even though the
// surfaced error is no longer the D3a TUI-stub literal.
func TestInit_BareInvocation_ReturnsTUIStubError(t *testing.T) {
	origFactory := programFactory
	t.Cleanup(func() { programFactory = origFactory })
	programFactory = func(m tea.Model) program {
		init, ok := m.(initTUIModel)
		if !ok {
			t.Fatalf("programFactory received model type %T; want initTUIModel", m)
		}
		// Simulate a completed walk: advance the model to Done with a
		// synthetic payload so runInitTUI exercises its happy-path branch
		// without needing a real terminal.
		init.step = initTUIStepDone
		init.finalPayload = initJSONPayload{Name: "stub", Group: "till-go", MCP: false}
		return scriptedProgram{model: init, runFn: func(current tea.Model) (tea.Model, error) {
			return current, nil
		}}
	}

	var out strings.Builder
	err := run(context.Background(), []string{"--app", "tillsyn-init", "init"}, &out, io.Discard)
	if err == nil {
		t.Fatalf("run(init) returned nil; expected D5 file-copy stub error after stubbed TUI walk, got stdout=%q", out.String())
	}
	want := "till init: file copy not yet wired (W2.D5)"
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("run(init) error = %q; want substring %q", err.Error(), want)
	}
}

// TestInit_JSONInvocation_RoutesToValidParse verifies that `till init
// --json '{...}'` with a well-formed payload routes through cobra to the
// real JSON parser shipped in D3b. A valid payload parses + validates and
// then surfaces the D5-stub error from the file-copy pipeline (which D5
// will wire). CONSUMER-TIE TEST CONTRACT (W2-FF6 ROUND-2).
func TestInit_JSONInvocation_RoutesToValidParse(t *testing.T) {
	var out strings.Builder
	err := run(context.Background(), []string{"--app", "tillsyn-init", "init", "--json", `{"name":"foo","group":"till-go","mcp":false}`}, &out, io.Discard)
	if err == nil {
		t.Fatalf("run(init --json valid) returned nil; expected D5 file-copy stub error, got stdout=%q", out.String())
	}
	want := "till init: file copy not yet wired (W2.D5)"
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("run(init --json valid) error = %q; want substring %q", err.Error(), want)
	}
}

// TestInit_JSONParse_TableDriven covers the D3b JSON-payload parser +
// group-validation matrix: valid payload, reserved `till-gdd` group,
// unknown group, malformed JSON, and missing required fields. Each case
// drives `run(...)` end-to-end so the cobra wiring is exercised; failure
// surfaces are matched by substring against the wrapped error returned
// from `runInitJSON`.
func TestInit_JSONParse_TableDriven(t *testing.T) {
	cases := []struct {
		name        string
		payload     string
		wantSubstrs []string
	}{
		{
			name:        "valid_till_go",
			payload:     `{"name":"foo","group":"till-go","mcp":false}`,
			wantSubstrs: []string{"file copy not yet wired (W2.D5)"},
		},
		{
			name:        "valid_till_gen_mcp_true",
			payload:     `{"name":"bar","group":"till-gen","mcp":true}`,
			wantSubstrs: []string{"file copy not yet wired (W2.D5)"},
		},
		{
			name:        "reserved_group_till_gdd",
			payload:     `{"name":"foo","group":"till-gdd","mcp":false}`,
			wantSubstrs: []string{"till-gdd", "reserved"},
		},
		{
			name:        "unknown_group",
			payload:     `{"name":"foo","group":"till-rust","mcp":false}`,
			wantSubstrs: []string{"group must be one of"},
		},
		{
			name:        "malformed_json",
			payload:     `{not json`,
			wantSubstrs: []string{"till init", "json"},
		},
		{
			name:        "missing_name",
			payload:     `{"group":"till-go"}`,
			wantSubstrs: []string{"name", "required"},
		},
		{
			name:        "missing_group",
			payload:     `{"name":"foo"}`,
			wantSubstrs: []string{"group", "required"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var out strings.Builder
			err := run(context.Background(), []string{"--app", "tillsyn-init", "init", "--json", tc.payload}, &out, io.Discard)
			if err == nil {
				t.Fatalf("run(init --json %q) returned nil; expected error containing %v, stdout=%q", tc.payload, tc.wantSubstrs, out.String())
			}
			got := err.Error()
			for _, sub := range tc.wantSubstrs {
				if !strings.Contains(got, sub) {
					t.Fatalf("run(init --json %q) error = %q; want substring %q", tc.payload, got, sub)
				}
			}
		})
	}
}

// TestRunInitTUI_AcceptsDefaultNameAndSelectsTillGo drives the bubbletea
// walk shipped in D4: the user presses enter on the default name (which is
// `filepath.Base(cwd)`), moves the group cursor down to `till-go`, and
// presses enter to confirm. The final model exposes a Payload() that must
// equal `{Name: <cwd-base>, Group: "till-go", MCP: false}` and Done() must
// be true.
//
// The test does NOT exercise the cobra wiring — `runInitTUI` itself depends
// on `programFactory`, which writes to /dev/tty in production. Driving the
// walk at the model level via teatest is the canonical pattern used in
// `internal/tui/model_teatest_test.go` and keeps the test deterministic.
func TestRunInitTUI_AcceptsDefaultNameAndSelectsTillGo(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd: %v", err)
	}
	wantName := filepath.Base(cwd)

	m := newInitTUIModel(cwd)
	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(80, 24))
	t.Cleanup(func() {
		_ = tm.Quit()
	})

	// Wait for the initial frame to render so we know the program is
	// processing input.
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return strings.Contains(string(out), "Project name")
	}, teatest.WithDuration(2*time.Second), teatest.WithCheckInterval(10*time.Millisecond))

	// Step 1: press Enter on the default name field to advance to the
	// group picker.
	tm.Send(tea.KeyPressMsg{Code: tea.KeyEnter})

	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return strings.Contains(string(out), "till-go")
	}, teatest.WithDuration(2*time.Second), teatest.WithCheckInterval(10*time.Millisecond))

	// Step 2: cursor starts on `till-gen` (default). Press Down to land on
	// `till-go`.
	tm.Send(tea.KeyPressMsg{Code: tea.KeyDown})

	// Step 3: press Enter to confirm `till-go`.
	tm.Send(tea.KeyPressMsg{Code: tea.KeyEnter})

	tm.WaitFinished(t, teatest.WithFinalTimeout(2*time.Second))

	final, ok := tm.FinalModel(t).(initTUIModel)
	if !ok {
		t.Fatalf("FinalModel type = %T; want initTUIModel", tm.FinalModel(t))
	}
	if !final.Done() {
		t.Fatalf("final.Done() = false; want true after enter on group")
	}
	if final.Cancelled() {
		t.Fatalf("final.Cancelled() = true; want false after a complete walk")
	}
	got := final.Payload()
	if got.Name != wantName {
		t.Fatalf("Payload().Name = %q; want %q", got.Name, wantName)
	}
	if got.Group != "till-go" {
		t.Fatalf("Payload().Group = %q; want %q", got.Group, "till-go")
	}
	if got.MCP {
		t.Fatalf("Payload().MCP = true; want false (TUI mode default)")
	}
}

// TestRunInitTUI_DisabledTillGddIsUnselectable verifies the SKETCH §9.3
// rule that `till-gdd` is shown but unselectable. Pressing Down from
// `till-go` must NOT land the cursor on `till-gdd` — the cursor must either
// stay on `till-go` (skip past the disabled row) or wrap, NOT advance to
// a disabled row. Pressing Enter while the cursor sits where the user last
// landed (`till-go`) confirms the group selection and finishes the walk;
// the final payload must report `till-go`, never `till-gdd`.
func TestRunInitTUI_DisabledTillGddIsUnselectable(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd: %v", err)
	}

	m := newInitTUIModel(cwd)
	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(80, 24))
	t.Cleanup(func() {
		_ = tm.Quit()
	})

	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return strings.Contains(string(out), "Project name")
	}, teatest.WithDuration(2*time.Second), teatest.WithCheckInterval(10*time.Millisecond))

	// Accept default name.
	tm.Send(tea.KeyPressMsg{Code: tea.KeyEnter})

	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return strings.Contains(string(out), "till-gdd")
	}, teatest.WithDuration(2*time.Second), teatest.WithCheckInterval(10*time.Millisecond))

	// Cursor on `till-gen`. Move down to `till-go`, then move down again —
	// `till-gdd` is disabled so the cursor must not advance onto it. After
	// two Downs the cursor must still report `till-go`.
	tm.Send(tea.KeyPressMsg{Code: tea.KeyDown})
	tm.Send(tea.KeyPressMsg{Code: tea.KeyDown})

	// Press Enter — should confirm `till-go`, NOT `till-gdd`.
	tm.Send(tea.KeyPressMsg{Code: tea.KeyEnter})

	tm.WaitFinished(t, teatest.WithFinalTimeout(2*time.Second))

	final, ok := tm.FinalModel(t).(initTUIModel)
	if !ok {
		t.Fatalf("FinalModel type = %T; want initTUIModel", tm.FinalModel(t))
	}
	if final.Cancelled() {
		t.Fatalf("final.Cancelled() = true; want false (walk completed, just on a non-disabled row)")
	}
	if got := final.Payload().Group; got != "till-go" {
		t.Fatalf("Payload().Group = %q; want %q (cursor must skip disabled till-gdd row)", got, "till-go")
	}
}

// TestRunInitTUI_EscCancelsWalk verifies that pressing Esc before completing
// the group selection sets Cancelled() and leaves Done() false. The runInitTUI
// caller treats a cancelled walk as an error so the D5-stub path does not run.
func TestRunInitTUI_EscCancelsWalk(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd: %v", err)
	}

	m := newInitTUIModel(cwd)
	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(80, 24))
	t.Cleanup(func() {
		_ = tm.Quit()
	})

	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return strings.Contains(string(out), "Project name")
	}, teatest.WithDuration(2*time.Second), teatest.WithCheckInterval(10*time.Millisecond))

	tm.Send(tea.KeyPressMsg{Code: tea.KeyEsc})

	tm.WaitFinished(t, teatest.WithFinalTimeout(2*time.Second))

	final, ok := tm.FinalModel(t).(initTUIModel)
	if !ok {
		t.Fatalf("FinalModel type = %T; want initTUIModel", tm.FinalModel(t))
	}
	if !final.Cancelled() {
		t.Fatalf("final.Cancelled() = false; want true after Esc")
	}
	if final.Done() {
		t.Fatalf("final.Done() = true; want false after Esc cancel")
	}
}
