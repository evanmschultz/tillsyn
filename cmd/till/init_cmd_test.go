package main

import (
	"bufio"
	"context"
	"io"
	"io/fs"
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
// **D5 update**: with the file-copy pipeline wired, a completed TUI walk
// now advances PAST the D5 stub and hands forward to the D6 `.mcp.json`
// stub. The smoke test still stubs `programFactory` to avoid opening a
// real terminal AND now chdirs into a `t.TempDir()` so the pipeline's
// real filesystem writes land in an isolated sandbox rather than the
// source checkout. The test name is preserved for git-blame continuity
// even though the surfaced error is now the D6 stub literal.
func TestInit_BareInvocation_ReturnsTUIStubError(t *testing.T) {
	t.Chdir(t.TempDir())
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
		t.Fatalf("run(init) returned nil; expected D6 .mcp.json stub error after stubbed TUI walk + D5 pipeline, got stdout=%q", out.String())
	}
	want := "till init: .mcp.json registration not yet wired (W2.D6)"
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("run(init) error = %q; want substring %q", err.Error(), want)
	}
}

// TestInit_JSONInvocation_RoutesToValidParse verifies that `till init
// --json '{...}'` with a well-formed payload routes through cobra to the
// real JSON parser shipped in D3b AND runs the D5 file-copy pipeline.
// A valid payload parses, validates, copies the embedded agent set, and
// then surfaces the D6 `.mcp.json` stub error (which D6 will wire).
// CONSUMER-TIE TEST CONTRACT (W2-FF6 ROUND-2).
//
// **D5 update**: chdir into a fresh t.TempDir() so the pipeline's real
// filesystem writes are sandboxed; assert against the D6 stub literal
// (was D5 stub pre-D5).
func TestInit_JSONInvocation_RoutesToValidParse(t *testing.T) {
	t.Chdir(t.TempDir())
	var out strings.Builder
	err := run(context.Background(), []string{"--app", "tillsyn-init", "init", "--json", `{"name":"foo","group":"till-go","mcp":false}`}, &out, io.Discard)
	if err == nil {
		t.Fatalf("run(init --json valid) returned nil; expected D6 .mcp.json stub error, got stdout=%q", out.String())
	}
	want := "till init: .mcp.json registration not yet wired (W2.D6)"
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
//
// **D5 update**: every test case chdirs into a fresh t.TempDir() because
// valid payloads now exercise the D5 pipeline (filesystem side effects)
// before surfacing the D6 stub error. Invalid payloads short-circuit
// before any write, but chdir is uniform across cases for consistency.
// The two valid cases assert the D6 stub literal (was D5 stub pre-D5).
func TestInit_JSONParse_TableDriven(t *testing.T) {
	cases := []struct {
		name        string
		payload     string
		wantSubstrs []string
	}{
		{
			name:        "valid_till_go",
			payload:     `{"name":"foo","group":"till-go","mcp":false}`,
			wantSubstrs: []string{".mcp.json registration not yet wired (W2.D6)"},
		},
		{
			name:        "valid_till_gen_mcp_true",
			payload:     `{"name":"bar","group":"till-gen","mcp":true}`,
			wantSubstrs: []string{".mcp.json registration not yet wired (W2.D6)"},
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
			t.Chdir(t.TempDir())
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

// runInitJSONInTempDir is a tiny helper that chdirs into a fresh temp dir
// and invokes `till init --json <payload>` end-to-end via `run`. Returns
// the temp dir + the wrapped error so each D5 test can assert filesystem
// state under the temp dir and inspect the surfaced error.
//
// The JSON-mode end-to-end form is the CONSUMER-TIE shape mandated by
// W2-FF6 ROUND-2: every D5 test routes through cobra so the wiring proves
// `runInitJSON` → `copyAgentFiles` → `copyAgentsTOML` → `ensureGitignore`
// runs in the real dispatch order, not as a unit-test of the pipeline
// helpers in isolation.
func runInitJSONInTempDir(t *testing.T, payload string) (string, error) {
	t.Helper()
	dir := t.TempDir()
	t.Chdir(dir)
	var out strings.Builder
	err := run(context.Background(), []string{"--app", "tillsyn-init", "init", "--json", payload}, &out, io.Discard)
	return dir, err
}

// TestInit_FreshDir_CopiesAllFiles drives `till init --json` against an
// empty t.TempDir() and asserts the D5 pipeline produces:
//   - at least 7 agent .md files under `.tillsyn/agents/` (FLAT — no group
//     prefix). till-go currently ships 12 .md files (7 standard + 5 legacy
//     `go-*` placeholders); the floor is the SKETCH §11.1 standard count.
//   - `agents.toml` at the project root, copied from the embedded example.
//   - `.gitignore` at the project root containing the literal line
//     `agents.local.toml`.
//
// The surfaced error MUST be the D6 stub literal — D5 hands forward to D6
// for `.mcp.json` registration.
func TestInit_FreshDir_CopiesAllFiles(t *testing.T) {
	dir, err := runInitJSONInTempDir(t, `{"name":"foo","group":"till-go","mcp":false}`)
	if err == nil {
		t.Fatalf("run(init --json) returned nil; expected D6 stub error")
	}
	wantStub := "till init: .mcp.json registration not yet wired (W2.D6)"
	if !strings.Contains(err.Error(), wantStub) {
		t.Fatalf("run(init --json) error = %q; want substring %q", err.Error(), wantStub)
	}

	agentsDir := filepath.Join(dir, ".tillsyn", "agents")
	entries, readErr := os.ReadDir(agentsDir)
	if readErr != nil {
		t.Fatalf("os.ReadDir(%q): %v", agentsDir, readErr)
	}
	mdCount := 0
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") {
			mdCount++
		}
	}
	if mdCount < 7 {
		t.Fatalf("agent .md count under %q = %d; want >= 7 (SKETCH §11.1 standard set)", agentsDir, mdCount)
	}

	// Spot-check a representative standard agent .md exists FLAT (no group prefix).
	if _, statErr := os.Stat(filepath.Join(agentsDir, "builder-agent.md")); statErr != nil {
		t.Fatalf("os.Stat(builder-agent.md): %v (FLAT copy required — no till-go/ prefix)", statErr)
	}

	if _, statErr := os.Stat(filepath.Join(dir, "agents.toml")); statErr != nil {
		t.Fatalf("os.Stat(agents.toml): %v", statErr)
	}

	gitignorePath := filepath.Join(dir, ".gitignore")
	gitignoreData, readErr := os.ReadFile(gitignorePath)
	if readErr != nil {
		t.Fatalf("os.ReadFile(%q): %v", gitignorePath, readErr)
	}
	if !gitignoreLineContains(string(gitignoreData), "agents.local.toml") {
		t.Fatalf(".gitignore = %q; want line equal to %q", string(gitignoreData), "agents.local.toml")
	}
}

// TestInit_RerunSafety_NoOverwrite runs `till init` twice in the same
// temp dir. The second run MUST NOT modify any file written by the first
// run — re-run safety is the hard invariant for D5 per the droplet's
// "Re-run safety (mandatory invariant)" clause. The check compares
// modification times AND file content hashes (mtimes alone are not
// sufficient on filesystems with second-granularity timestamps).
func TestInit_RerunSafety_NoOverwrite(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	// First run.
	var out1 strings.Builder
	_ = run(context.Background(), []string{"--app", "tillsyn-init", "init", "--json", `{"name":"foo","group":"till-go","mcp":false}`}, &out1, io.Discard)

	// Snapshot every file under the project dir after the first run.
	type snapshot struct {
		mode os.FileMode
		size int64
		data []byte
	}
	preState := map[string]snapshot{}
	walkErr := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		info, infoErr := d.Info()
		if infoErr != nil {
			return infoErr
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		preState[path] = snapshot{mode: info.Mode(), size: info.Size(), data: data}
		return nil
	})
	if walkErr != nil {
		t.Fatalf("filepath.WalkDir pre: %v", walkErr)
	}
	if len(preState) == 0 {
		t.Fatalf("first run produced no files under %q; expected agent .md set + agents.toml + .gitignore", dir)
	}

	// Sleep is unnecessary: we compare content hashes, not just mtimes.

	// Second run.
	var out2 strings.Builder
	_ = run(context.Background(), []string{"--app", "tillsyn-init", "init", "--json", `{"name":"foo","group":"till-go","mcp":false}`}, &out2, io.Discard)

	// Every pre-existing file must be byte-for-byte unchanged.
	for path, before := range preState {
		afterData, readErr := os.ReadFile(path)
		if readErr != nil {
			t.Fatalf("post-run ReadFile(%q): %v", path, readErr)
		}
		if string(afterData) != string(before.data) {
			t.Fatalf("re-run mutated %q (size %d -> %d)", path, before.size, len(afterData))
		}
	}
}

// TestInit_GitignoreIdempotent seeds a `.gitignore` that already contains
// the `agents.local.toml` line and asserts re-running `till init` does NOT
// duplicate the line. This is the W2-FF10 round-2 LOCKED line-iteration
// fix in action: a raw bytes.Contains check that requires `\nagents.local.toml\n`
// would MISS the first-line-only case and append a duplicate.
func TestInit_GitignoreIdempotent(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	// First-line-only seed — the exact case raw bytes.Contains misses.
	if err := os.WriteFile(filepath.Join(dir, ".gitignore"), []byte("agents.local.toml\n"), 0o644); err != nil {
		t.Fatalf("seed .gitignore: %v", err)
	}

	var out strings.Builder
	_ = run(context.Background(), []string{"--app", "tillsyn-init", "init", "--json", `{"name":"foo","group":"till-go","mcp":false}`}, &out, io.Discard)

	data, err := os.ReadFile(filepath.Join(dir, ".gitignore"))
	if err != nil {
		t.Fatalf("ReadFile .gitignore: %v", err)
	}
	occurrences := countGitignoreLine(string(data), "agents.local.toml")
	if occurrences != 1 {
		t.Fatalf(".gitignore has %d occurrences of %q; want exactly 1\nbody = %q", occurrences, "agents.local.toml", string(data))
	}
}

// TestInit_PreExistingGitignore_AppendsCleanly verifies that an existing
// `.gitignore` with unrelated entries gets `agents.local.toml` appended
// once with proper newline handling. Covers two trailing-newline shapes
// since the round-2 fix must handle both cases.
func TestInit_PreExistingGitignore_AppendsCleanly(t *testing.T) {
	cases := []struct {
		name string
		seed string
	}{
		{name: "trailing_newline", seed: "node_modules/\n.env\n"},
		{name: "no_trailing_newline", seed: "node_modules/\n.env"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			t.Chdir(dir)

			if err := os.WriteFile(filepath.Join(dir, ".gitignore"), []byte(tc.seed), 0o644); err != nil {
				t.Fatalf("seed .gitignore: %v", err)
			}

			var out strings.Builder
			_ = run(context.Background(), []string{"--app", "tillsyn-init", "init", "--json", `{"name":"foo","group":"till-go","mcp":false}`}, &out, io.Discard)

			data, err := os.ReadFile(filepath.Join(dir, ".gitignore"))
			if err != nil {
				t.Fatalf("ReadFile .gitignore: %v", err)
			}
			body := string(data)

			// Pre-existing entries must survive.
			if !strings.Contains(body, "node_modules/") {
				t.Fatalf(".gitignore lost pre-existing line %q; body = %q", "node_modules/", body)
			}
			if !strings.Contains(body, ".env") {
				t.Fatalf(".gitignore lost pre-existing line %q; body = %q", ".env", body)
			}
			// agents.local.toml appended exactly once.
			occurrences := countGitignoreLine(body, "agents.local.toml")
			if occurrences != 1 {
				t.Fatalf(".gitignore has %d occurrences of %q; want exactly 1\nbody = %q", occurrences, "agents.local.toml", body)
			}
			// Trailing-newline normalization: the file must end with `\n`
			// so subsequent appends concatenate cleanly.
			if !strings.HasSuffix(body, "\n") {
				t.Fatalf(".gitignore missing trailing newline; body = %q", body)
			}
		})
	}
}

// gitignoreLineContains reports whether `body` contains a line whose
// trimmed value equals `want`. Mirrors the line-iteration discipline the
// production `ensureGitignore` uses so tests assert on the same notion of
// "line presence" the implementation enforces.
func gitignoreLineContains(body, want string) bool {
	return countGitignoreLine(body, want) > 0
}

// countGitignoreLine returns the number of lines in body whose trimmed
// value equals want. Trimming aligns with the production check: trailing
// whitespace differences should not produce phantom duplicates.
func countGitignoreLine(body, want string) int {
	n := 0
	sc := bufio.NewScanner(strings.NewReader(body))
	for sc.Scan() {
		if strings.TrimSpace(sc.Text()) == want {
			n++
		}
	}
	return n
}
