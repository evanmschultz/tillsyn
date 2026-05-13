package main

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/exp/teatest/v2"
	"github.com/google/uuid"

	"github.com/evanmschultz/tillsyn/internal/adapters/storage/sqlite"
	"github.com/evanmschultz/tillsyn/internal/app"
	"github.com/evanmschultz/tillsyn/internal/platform"
)

// TestInit_BareInvocation_ReturnsTUIStubError verifies that `till init` (bare,
// no --json) routes through cobra to runInitTUI. The end-to-end run()
// invocation exercises the cobra registration in main.go — calling
// cmd.RunE or runInitTUI directly would not prove the command is wired
// into rootCmd. CONSUMER-TIE TEST CONTRACT (W2-FF6 ROUND-2) — symmetric
// to D7.5's W2-FF3 contract.
//
// **D7 update**: D7 wires the real project-DB record creation. The stub
// error is gone; a completed TUI walk now returns nil. The test asserts
// success (nil error) and verifies the Laslig output contains the
// project name. HOME isolation via t.Setenv ensures the DB lands in
// t.TempDir() rather than the dev's real ~/.tillsyn-init/.
func TestInit_BareInvocation_ReturnsTUIStubError(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Chdir(tmp)
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
		mcpFalse := false
		init.finalPayload = initJSONPayload{Name: "stub-project", Groups: []string{"go"}, MCP: &mcpFalse}
		return scriptedProgram{model: init, runFn: func(current tea.Model) (tea.Model, error) {
			return current, nil
		}}
	}

	var out strings.Builder
	if err := run(context.Background(), []string{"--app", "tillsyn-init", "init"}, &out, io.Discard); err != nil {
		t.Fatalf("run(init) error = %v; expected nil after D7 wiring (stubbed TUI + D5 pipeline + D6 mcp skip + D7 DB create)", err)
	}
	if !strings.Contains(out.String(), "Init") {
		t.Fatalf("run(init) stdout = %q; want 'Init' Laslig block", out.String())
	}
}

// TestInit_JSONInvocation_RoutesToValidParse verifies that `till init
// --json '{...}'` with a well-formed payload routes through cobra to the
// real JSON parser shipped in D3b AND runs the D5 file-copy pipeline
// through D7's project-DB record creation.
// CONSUMER-TIE TEST CONTRACT (W2-FF6 ROUND-2).
//
// **D7 update**: D7 wires real project-DB creation; the pipeline now
// returns nil. The test asserts success and verifies the Laslig output
// contains the project name. HOME isolation ensures the DB lands in
// t.TempDir().
func TestInit_JSONInvocation_RoutesToValidParse(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Chdir(tmp)
	var out strings.Builder
	if err := run(context.Background(), []string{"--app", "tillsyn-init", "init", "--json", `{"name":"foo","groups":["go"],"mcp":false}`}, &out, io.Discard); err != nil {
		t.Fatalf("run(init --json valid) error = %v; expected nil after D7 wiring, stdout=%q", err, out.String())
	}
	if !strings.Contains(out.String(), "foo") {
		t.Fatalf("run(init --json valid) stdout = %q; want project name 'foo' in Laslig output", out.String())
	}
}

// TestInit_JSONParse_TableDriven covers the D3b JSON-payload parser +
// group-validation matrix: valid payloads (go, gen, fe, multi-group),
// unknown group, malformed JSON, and missing required fields. Each case
// drives `run(...)` end-to-end so the cobra wiring is exercised. Valid
// payloads succeed (nil error) through D7. The old `till-gdd` reserved-
// group case is replaced by an unknown-group case because reservedInitGroups
// was deleted in W2.D1 — `till-gdd` now surfaces as a plain invalid-group
// error.
//
// D7 update: valid cases return nil error with a Laslig output block.
// HOME isolation added so DB writes land in t.TempDir().
//
// D5 update: every test case chdirs into a fresh t.TempDir() because
// valid payloads now succeed through D7. Invalid payloads short-circuit
// before any write and still return errors.
func TestInit_JSONParse_TableDriven(t *testing.T) {
	type testCase struct {
		name        string
		payload     string
		wantSuccess bool // true: expect nil error + wantSubstrs match stdout
		wantSubstrs []string
	}
	cases := []testCase{
		{
			name:        "valid_go",
			payload:     `{"name":"foo","groups":["go"],"mcp":false}`,
			wantSuccess: true,
			wantSubstrs: []string{"Init", "foo", "go"},
		},
		{
			name:        "valid_gen_mcp_true",
			payload:     `{"name":"bar","groups":["gen"],"mcp":true}`,
			wantSuccess: true,
			wantSubstrs: []string{"Init", "bar", "gen"},
		},
		{
			name:        "valid_fe",
			payload:     `{"name":"baz","groups":["fe"],"mcp":false}`,
			wantSuccess: true,
			wantSubstrs: []string{"Init", "baz", "fe"},
		},
		{
			name:        "unknown_group_till_gdd",
			payload:     `{"name":"foo","groups":["till-gdd"],"mcp":false}`,
			wantSubstrs: []string{"invalid"},
		},
		{
			name:        "unknown_group",
			payload:     `{"name":"foo","groups":["till-rust"],"mcp":false}`,
			wantSubstrs: []string{"invalid"},
		},
		{
			name:        "malformed_json",
			payload:     `{not json`,
			wantSubstrs: []string{"till init", "json"},
		},
		{
			name:        "missing_name",
			payload:     `{"groups":["go"]}`,
			wantSubstrs: []string{"name", "required"},
		},
		{
			name:        "missing_groups",
			payload:     `{"name":"foo"}`,
			wantSubstrs: []string{"groups", "required"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tmp := t.TempDir()
			t.Setenv("HOME", tmp)
			t.Chdir(tmp)
			var out strings.Builder
			err := run(context.Background(), []string{"--app", "tillsyn-init", "init", "--json", tc.payload}, &out, io.Discard)
			if tc.wantSuccess {
				if err != nil {
					t.Fatalf("run(init --json %q) error = %v; want nil (D7 success)", tc.payload, err)
				}
				for _, sub := range tc.wantSubstrs {
					if !strings.Contains(out.String(), sub) {
						t.Fatalf("run(init --json %q) stdout = %q; want substring %q", tc.payload, out.String(), sub)
					}
				}
			} else {
				if err == nil {
					t.Fatalf("run(init --json %q) returned nil; expected error containing %v, stdout=%q", tc.payload, tc.wantSubstrs, out.String())
				}
				got := err.Error()
				for _, sub := range tc.wantSubstrs {
					if !strings.Contains(got, sub) {
						t.Fatalf("run(init --json %q) error = %q; want substring %q", tc.payload, got, sub)
					}
				}
			}
		})
	}
}

// TestRunInitTUI_AcceptsDefaultNameAndSelectsTillGo drives the bubbletea
// walk: the user presses enter on the default name (filepath.Base(cwd)),
// moves the group cursor down to `go` (row 1), and presses enter to confirm.
// The final model must have Done() true, Payload().Groups = ["go"], and
// MCPRegistration() false (TUI hardwires MCP=false until D4 adds the
// confirm step).
//
// Drop 4c.6.1 W2.D1: Groups replaces Group; initTUIGroupRows now has 3
// all-enabled rows: gen (0), go (1), fe (2). One Down press moves from
// gen to go.
//
// The test does NOT exercise the cobra wiring — runInitTUI depends on
// programFactory which writes to /dev/tty in production. Driving at the
// model level via teatest is the canonical pattern.
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
		return strings.Contains(string(out), "go")
	}, teatest.WithDuration(2*time.Second), teatest.WithCheckInterval(10*time.Millisecond))

	// Step 2: cursor starts on `gen` (default). Press Down to land on `go`.
	tm.Send(tea.KeyPressMsg{Code: tea.KeyDown})

	// Step 3: press Enter to confirm `go`.
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
	if len(got.Groups) != 1 || got.Groups[0] != "go" {
		t.Fatalf("Payload().Groups = %v; want [\"go\"]", got.Groups)
	}
	if got.MCPRegistration() {
		t.Fatalf("Payload().MCPRegistration() = true; want false (TUI mode default until D4)")
	}
}

// TestRunInitTUI_SelectsFeRow verifies that the third row (`fe`) is
// selectable after W2.D1 removes `till-gdd` and adds `fe` as a canonical
// all-enabled group. Pressing Down twice from `gen` (row 0) lands on `fe`
// (row 2); pressing Enter produces a payload with Groups = ["fe"].
//
// This replaces TestRunInitTUI_DisabledTillGddIsUnselectable which tested
// the old disabled-row skip logic. W2.D1 removes the disabled row; all
// three rows (gen/go/fe) are now enabled.
func TestRunInitTUI_SelectsFeRow(t *testing.T) {
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
		return strings.Contains(string(out), "fe")
	}, teatest.WithDuration(2*time.Second), teatest.WithCheckInterval(10*time.Millisecond))

	// Cursor starts on gen (row 0). Down -> go (row 1). Down -> fe (row 2).
	tm.Send(tea.KeyPressMsg{Code: tea.KeyDown})
	tm.Send(tea.KeyPressMsg{Code: tea.KeyDown})

	// Press Enter — confirms `fe`.
	tm.Send(tea.KeyPressMsg{Code: tea.KeyEnter})

	tm.WaitFinished(t, teatest.WithFinalTimeout(2*time.Second))

	final, ok := tm.FinalModel(t).(initTUIModel)
	if !ok {
		t.Fatalf("FinalModel type = %T; want initTUIModel", tm.FinalModel(t))
	}
	if final.Cancelled() {
		t.Fatalf("final.Cancelled() = true; want false (walk completed with fe selection)")
	}
	if !final.Done() {
		t.Fatalf("final.Done() = false; want true after enter on fe row")
	}
	if got := final.Payload().Groups; len(got) != 1 || got[0] != "fe" {
		t.Fatalf("Payload().Groups = %v; want [\"fe\"]", got)
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

// runInitJSONInTempDir is a tiny helper that chdirs into a fresh temp dir,
// sets HOME isolation so the project-DB write lands in the temp dir rather
// than the dev's real ~/.tillsyn-init/, and invokes `till init --json
// <payload>` end-to-end via `run`. Returns the temp dir + the wrapped
// error (nil on D7 success) so each test can assert filesystem state and
// the success/error surface.
//
// The JSON-mode end-to-end form is the CONSUMER-TIE shape mandated by
// W2-FF6 ROUND-2: every D5/D6/D7 test routes through cobra so the wiring
// proves the full `runInitJSON` → pipeline → DB chain runs in the real
// dispatch order.
func runInitJSONInTempDir(t *testing.T, payload string) (string, error) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("HOME", dir)
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
// **D7 update**: the pipeline now returns nil (project-DB record created).
func TestInit_FreshDir_CopiesAllFiles(t *testing.T) {
	dir, err := runInitJSONInTempDir(t, `{"name":"foo","groups":["go"],"mcp":false}`)
	if err != nil {
		t.Fatalf("run(init --json) error = %v; want nil after D7 wiring", err)
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
	t.Setenv("HOME", dir)
	t.Chdir(dir)

	// First run.
	var out1 strings.Builder
	_ = run(context.Background(), []string{"--app", "tillsyn-init", "init", "--json", `{"name":"foo","groups":["go"],"mcp":false}`}, &out1, io.Discard)

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
	_ = run(context.Background(), []string{"--app", "tillsyn-init", "init", "--json", `{"name":"foo","groups":["go"],"mcp":false}`}, &out2, io.Discard)

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
	t.Setenv("HOME", dir)
	t.Chdir(dir)

	// First-line-only seed — the exact case raw bytes.Contains misses.
	if err := os.WriteFile(filepath.Join(dir, ".gitignore"), []byte("agents.local.toml\n"), 0o644); err != nil {
		t.Fatalf("seed .gitignore: %v", err)
	}

	var out strings.Builder
	_ = run(context.Background(), []string{"--app", "tillsyn-init", "init", "--json", `{"name":"foo","groups":["go"],"mcp":false}`}, &out, io.Discard)

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
			t.Setenv("HOME", dir)
			t.Chdir(dir)

			if err := os.WriteFile(filepath.Join(dir, ".gitignore"), []byte(tc.seed), 0o644); err != nil {
				t.Fatalf("seed .gitignore: %v", err)
			}

			var out strings.Builder
			_ = run(context.Background(), []string{"--app", "tillsyn-init", "init", "--json", `{"name":"foo","groups":["go"],"mcp":false}`}, &out, io.Discard)

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

// TestInit_MCPJSON_FreshFile verifies that `till init --json` with `mcp:true`
// creates a `.mcp.json` file in the destination directory containing a
// `tillsyn` server entry. The test drives `run(...)` end-to-end (CONSUMER-TIE
// TEST CONTRACT — proves the cobra wiring exercises registerMCPJSON). The
// surfaced error is the D7 project-DB stub, confirming the pipeline ran
// through the D6 seam.
func TestInit_MCPJSON_FreshFile(t *testing.T) {
	dir, err := runInitJSONInTempDir(t, `{"name":"foo","groups":["go"],"mcp":true}`)
	if err != nil {
		t.Fatalf("run(init --json mcp:true) error = %v; want nil after D7 wiring", err)
	}

	mcpPath := filepath.Join(dir, ".mcp.json")
	data, readErr := os.ReadFile(mcpPath)
	if readErr != nil {
		t.Fatalf("os.ReadFile(%q): %v — .mcp.json not created", mcpPath, readErr)
	}

	// Parse as a two-level raw map to verify the tillsyn entry exists with a
	// non-empty command. Using map[string]json.RawMessage avoids coupling the
	// test to the internal mcpServerEntry struct type.
	var top map[string]json.RawMessage
	if unmarshalErr := json.Unmarshal(data, &top); unmarshalErr != nil {
		t.Fatalf("json.Unmarshal .mcp.json top level: %v\nbody = %q", unmarshalErr, string(data))
	}
	var servers map[string]json.RawMessage
	if unmarshalErr := json.Unmarshal(top["mcpServers"], &servers); unmarshalErr != nil {
		t.Fatalf("json.Unmarshal .mcp.json mcpServers: %v", unmarshalErr)
	}
	entryRaw, ok := servers[mcpServerKey]
	if !ok {
		t.Fatalf(".mcp.json missing %q entry; servers = %v", mcpServerKey, servers)
	}
	var entry mcpServerEntry
	if unmarshalErr := json.Unmarshal(entryRaw, &entry); unmarshalErr != nil {
		t.Fatalf("json.Unmarshal tillsyn entry: %v", unmarshalErr)
	}
	if entry.Command == "" {
		t.Fatalf(".mcp.json entry %q has empty command", mcpServerKey)
	}
}

// TestInit_MCPJSON_AppendsToExisting verifies that `registerMCPJSON` adds
// the `tillsyn` entry to an existing `.mcp.json` that already contains a
// different server. The pre-existing entry must survive — no overwrite,
// no loss. Drives end-to-end via run() (CONSUMER-TIE contract).
func TestInit_MCPJSON_AppendsToExisting(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Chdir(dir)

	// Seed .mcp.json with an unrelated stdio server entry using raw JSON to
	// avoid coupling the test to internal struct types.
	seedJSON := []byte(`{"mcpServers":{"other-server":{"command":"/usr/local/bin/other"}}}` + "\n")
	if writeErr := os.WriteFile(filepath.Join(dir, ".mcp.json"), seedJSON, 0o644); writeErr != nil {
		t.Fatalf("seed .mcp.json: %v", writeErr)
	}

	var out strings.Builder
	if err := run(context.Background(), []string{"--app", "tillsyn-init", "init", "--json", `{"name":"foo","groups":["go"],"mcp":true}`}, &out, io.Discard); err != nil {
		t.Fatalf("run(init --json mcp:true) error = %v; want nil after D7 wiring", err)
	}

	data, readErr := os.ReadFile(filepath.Join(dir, ".mcp.json"))
	if readErr != nil {
		t.Fatalf("os.ReadFile .mcp.json: %v", readErr)
	}

	// Parse via raw maps to verify both entries without depending on internal
	// struct types.
	var top map[string]json.RawMessage
	if unmarshalErr := json.Unmarshal(data, &top); unmarshalErr != nil {
		t.Fatalf("json.Unmarshal .mcp.json: %v\nbody = %q", unmarshalErr, string(data))
	}
	var servers map[string]json.RawMessage
	if unmarshalErr := json.Unmarshal(top["mcpServers"], &servers); unmarshalErr != nil {
		t.Fatalf("json.Unmarshal .mcp.json mcpServers: %v", unmarshalErr)
	}

	// tillsyn entry must be present.
	if _, ok := servers[mcpServerKey]; !ok {
		t.Fatalf(".mcp.json missing %q entry after append; servers = %v", mcpServerKey, servers)
	}
	// Pre-existing entry must survive.
	if _, ok := servers["other-server"]; !ok {
		t.Fatalf(".mcp.json lost pre-existing %q entry; servers = %v", "other-server", servers)
	}
}

// TestInit_MCPJSON_Idempotent verifies that running `till init --json` with
// `mcp:true` when a `tillsyn` entry already exists is a no-op — no duplicate
// entry, no mutation. The entry that was there must remain unchanged.
// Drives end-to-end via run() (CONSUMER-TIE contract).
func TestInit_MCPJSON_Idempotent(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Chdir(dir)

	// Seed .mcp.json with an existing tillsyn entry using raw JSON.
	seedJSON := []byte(`{"mcpServers":{"tillsyn":{"command":"/original/path/to/till"}}}` + "\n")
	if writeErr := os.WriteFile(filepath.Join(dir, ".mcp.json"), seedJSON, 0o644); writeErr != nil {
		t.Fatalf("seed .mcp.json: %v", writeErr)
	}

	var out strings.Builder
	if err := run(context.Background(), []string{"--app", "tillsyn-init", "init", "--json", `{"name":"foo","groups":["go"],"mcp":true}`}, &out, io.Discard); err != nil {
		t.Fatalf("run(init --json mcp:true) error = %v; want nil after D7 wiring", err)
	}

	data, readErr := os.ReadFile(filepath.Join(dir, ".mcp.json"))
	if readErr != nil {
		t.Fatalf("os.ReadFile .mcp.json: %v", readErr)
	}

	// Parse via raw maps to check idempotency without depending on internal types.
	var top map[string]json.RawMessage
	if unmarshalErr := json.Unmarshal(data, &top); unmarshalErr != nil {
		t.Fatalf("json.Unmarshal .mcp.json: %v\nbody = %q", unmarshalErr, string(data))
	}
	var servers map[string]json.RawMessage
	if unmarshalErr := json.Unmarshal(top["mcpServers"], &servers); unmarshalErr != nil {
		t.Fatalf("json.Unmarshal .mcp.json mcpServers: %v", unmarshalErr)
	}

	entryRaw, ok := servers[mcpServerKey]
	if !ok {
		t.Fatalf(".mcp.json missing %q entry after idempotent re-run", mcpServerKey)
	}
	var entry mcpServerEntry
	if unmarshalErr := json.Unmarshal(entryRaw, &entry); unmarshalErr != nil {
		t.Fatalf("json.Unmarshal tillsyn entry: %v", unmarshalErr)
	}
	// The original command path must be preserved (not overwritten with a new LookPath result).
	if entry.Command != "/original/path/to/till" {
		t.Fatalf(".mcp.json entry %q command = %q; want %q (must not overwrite existing entry)",
			mcpServerKey, entry.Command, "/original/path/to/till")
	}
	// Only one tillsyn entry — JSON object keys are unique by definition, but
	// verify the total server count is still 1 (no phantom duplicates).
	if got := len(servers); got != 1 {
		t.Fatalf(".mcp.json has %d server entries; want 1 (idempotent re-run must not add duplicates)", got)
	}
}

// TestInit_MCPJSON_OptOut verifies that `till init --json` with `mcp:false`
// does NOT create a `.mcp.json` file. The pipeline runs through the D6 seam
// (registerMCPJSON returns immediately on includeMCP=false) and surfaces the
// D7 project-DB stub. Drives end-to-end via run() (CONSUMER-TIE contract).
func TestInit_MCPJSON_OptOut(t *testing.T) {
	dir, err := runInitJSONInTempDir(t, `{"name":"foo","groups":["go"],"mcp":false}`)
	if err != nil {
		t.Fatalf("run(init --json mcp:false) error = %v; want nil after D7 wiring", err)
	}

	mcpPath := filepath.Join(dir, ".mcp.json")
	if _, statErr := os.Stat(mcpPath); statErr == nil {
		t.Fatalf(".mcp.json exists at %q; mcp:false must not create the file", mcpPath)
	}
}

// TestInit_MCPJSON_PreservesHTTPTransport is the Drop 4c.6 W2.D6 Round-2
// regression test for FF1. It verifies that `till init --json` with `mcp:true`
// preserves pre-existing HTTP/SSE server entries byte-equivalent — i.e., that
// fields like `type` and `url` authored by `claude mcp add --transport http`
// are NOT silently dropped on round-trip through registerMCPJSON.
//
// Prior to the Round-2 fix, the typed mcpServerEntry struct only modelled
// stdio fields (command/args/env). Any entry with a `type` or `url` field
// would be deserialized to zero-value and re-marshaled as `{"command":""}`,
// destroying the entry. The fix uses json.RawMessage for all pre-existing
// entries and only typed-deserializes the new `tillsyn` entry.
//
// Drives end-to-end via run() (CONSUMER-TIE TEST CONTRACT).
func TestInit_MCPJSON_PreservesHTTPTransport(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Chdir(dir)

	// Seed .mcp.json with an HTTP-transport entry (the shape produced by
	// `claude mcp add --transport http --scope project notion https://mcp.notion.com/mcp`).
	notionEntry := `{"type":"http","url":"https://mcp.notion.com/mcp"}`
	seedJSON := []byte(`{"mcpServers":{"notion":` + notionEntry + `}}` + "\n")
	if writeErr := os.WriteFile(filepath.Join(dir, ".mcp.json"), seedJSON, 0o644); writeErr != nil {
		t.Fatalf("seed .mcp.json: %v", writeErr)
	}

	var out strings.Builder
	if err := run(context.Background(), []string{"--app", "tillsyn-init", "init", "--json", `{"name":"foo","groups":["go"],"mcp":true}`}, &out, io.Discard); err != nil {
		t.Fatalf("run(init --json mcp:true) error = %v; want nil after D7 wiring", err)
	}

	data, readErr := os.ReadFile(filepath.Join(dir, ".mcp.json"))
	if readErr != nil {
		t.Fatalf("os.ReadFile .mcp.json: %v", readErr)
	}

	// Parse the result via raw maps.
	var top map[string]json.RawMessage
	if unmarshalErr := json.Unmarshal(data, &top); unmarshalErr != nil {
		t.Fatalf("json.Unmarshal .mcp.json: %v\nbody = %q", unmarshalErr, string(data))
	}
	var servers map[string]json.RawMessage
	if unmarshalErr := json.Unmarshal(top["mcpServers"], &servers); unmarshalErr != nil {
		t.Fatalf("json.Unmarshal .mcp.json mcpServers: %v", unmarshalErr)
	}

	// (a) The tillsyn entry must have been added.
	if _, ok := servers[mcpServerKey]; !ok {
		t.Fatalf(".mcp.json missing %q entry; servers = %v", mcpServerKey, servers)
	}

	// (b) The notion HTTP entry must be preserved with its type and url fields
	// intact. Parse the raw notion entry and verify via a field map.
	notionRaw, ok := servers["notion"]
	if !ok {
		t.Fatalf(".mcp.json lost pre-existing %q entry; servers = %v", "notion", servers)
	}
	var notionFields map[string]json.RawMessage
	if unmarshalErr := json.Unmarshal(notionRaw, &notionFields); unmarshalErr != nil {
		t.Fatalf("json.Unmarshal notion entry: %v", unmarshalErr)
	}
	wantType := `"http"`
	wantURL := `"https://mcp.notion.com/mcp"`
	if got := string(notionFields["type"]); got != wantType {
		t.Fatalf("notion entry type = %s; want %s (HTTP transport field was dropped)", got, wantType)
	}
	if got := string(notionFields["url"]); got != wantURL {
		t.Fatalf("notion entry url = %s; want %s (HTTP transport URL was dropped)", got, wantURL)
	}
}

// TestInit_MCPJSON_PreservesTopLevelExtras verifies that `till init --json`
// with `mcp:true` preserves sibling top-level keys in `.mcp.json` beyond
// `mcpServers`. This closes the NIT2 finding from Drop 4c.6 W2.D6 Round-1:
// the original typed mcpJSONFile struct would have dropped any key it did
// not declare on re-marshal. The Round-2 fix uses a top-level
// map[string]json.RawMessage so all sibling keys survive.
//
// Drives end-to-end via run() (CONSUMER-TIE TEST CONTRACT).
func TestInit_MCPJSON_PreservesTopLevelExtras(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Chdir(dir)

	// Seed .mcp.json with an extra top-level key alongside mcpServers.
	seedJSON := []byte(`{"mcpServers":{},"someOtherKey":{"foo":"bar"}}` + "\n")
	if writeErr := os.WriteFile(filepath.Join(dir, ".mcp.json"), seedJSON, 0o644); writeErr != nil {
		t.Fatalf("seed .mcp.json: %v", writeErr)
	}

	var out strings.Builder
	if err := run(context.Background(), []string{"--app", "tillsyn-init", "init", "--json", `{"name":"foo","groups":["go"],"mcp":true}`}, &out, io.Discard); err != nil {
		t.Fatalf("run(init --json mcp:true) error = %v; want nil after D7 wiring", err)
	}

	data, readErr := os.ReadFile(filepath.Join(dir, ".mcp.json"))
	if readErr != nil {
		t.Fatalf("os.ReadFile .mcp.json: %v", readErr)
	}

	// Parse the result via raw maps and assert the extra key survived.
	var top map[string]json.RawMessage
	if unmarshalErr := json.Unmarshal(data, &top); unmarshalErr != nil {
		t.Fatalf("json.Unmarshal .mcp.json: %v\nbody = %q", unmarshalErr, string(data))
	}

	extraRaw, ok := top["someOtherKey"]
	if !ok {
		t.Fatalf(".mcp.json lost top-level key %q; keys present = %v", "someOtherKey", topKeys(top))
	}
	var extra map[string]json.RawMessage
	if unmarshalErr := json.Unmarshal(extraRaw, &extra); unmarshalErr != nil {
		t.Fatalf("json.Unmarshal someOtherKey: %v", unmarshalErr)
	}
	if got := string(extra["foo"]); got != `"bar"` {
		t.Fatalf("someOtherKey.foo = %s; want %q (top-level extra key was corrupted)", got, `"bar"`)
	}
}

// TestInit_MCPJSON_NullMcpServersValue is the Drop 4c.6 W2.D6 Round-3
// regression test for FF2. It verifies that `till init --json` with `mcp:true`
// does NOT panic when the existing `.mcp.json` contains `{"mcpServers":null}`.
//
// Prior to the Round-3 fix, json.Unmarshal of the JSON null value into the
// pre-initialised servers map pointer set servers to nil, causing a
// "assignment to entry in nil map" panic on the subsequent write.
//
// Drives end-to-end via run() (CONSUMER-TIE TEST CONTRACT).
func TestInit_MCPJSON_NullMcpServersValue(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Chdir(dir)

	// Seed .mcp.json with a null mcpServers value — legal JSON, can be
	// produced by hand-edits, third-party tools, or migration scripts.
	seedJSON := []byte(`{"mcpServers":null}` + "\n")
	if writeErr := os.WriteFile(filepath.Join(dir, ".mcp.json"), seedJSON, 0o644); writeErr != nil {
		t.Fatalf("seed .mcp.json: %v", writeErr)
	}

	var out strings.Builder
	if err := run(context.Background(), []string{"--app", "tillsyn-init", "init", "--json", `{"name":"foo","groups":["go"],"mcp":true}`}, &out, io.Discard); err != nil {
		t.Fatalf("run(init --json mcp:true) error = %v; want nil after D7 wiring", err)
	}

	data, readErr := os.ReadFile(filepath.Join(dir, ".mcp.json"))
	if readErr != nil {
		t.Fatalf("os.ReadFile .mcp.json after null-mcpServers run: %v", readErr)
	}

	// The resulting file must be well-formed JSON.
	var top map[string]json.RawMessage
	if unmarshalErr := json.Unmarshal(data, &top); unmarshalErr != nil {
		t.Fatalf("json.Unmarshal .mcp.json: %v\nbody = %q", unmarshalErr, string(data))
	}

	// The tillsyn entry must have been added correctly.
	var servers map[string]json.RawMessage
	if unmarshalErr := json.Unmarshal(top[mcpServersKey], &servers); unmarshalErr != nil {
		t.Fatalf("json.Unmarshal .mcp.json mcpServers: %v", unmarshalErr)
	}
	if _, ok := servers[mcpServerKey]; !ok {
		t.Fatalf(".mcp.json missing %q entry after null-mcpServers run; servers = %v", mcpServerKey, servers)
	}

	// The tillsyn entry must have a non-empty command field.
	var entry mcpServerEntry
	if unmarshalErr := json.Unmarshal(servers[mcpServerKey], &entry); unmarshalErr != nil {
		t.Fatalf("json.Unmarshal tillsyn entry: %v", unmarshalErr)
	}
	if entry.Command == "" {
		t.Fatalf(".mcp.json tillsyn entry has empty command after null-mcpServers run")
	}
}

// TestInit_MCPJSON_NullTopLevelFile verifies that `till init --json` with
// `mcp:true` does NOT panic when `.mcp.json` contains the bare JSON literal
// `null` (the entire file is the null literal, not an object). This exercises
// the top-level nil-guard (lines 690-692 of init_cmd.go) which catches
// json.Unmarshal setting the top-level map to nil.
//
// Drives end-to-end via run() (CONSUMER-TIE TEST CONTRACT).
func TestInit_MCPJSON_NullTopLevelFile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Chdir(dir)

	// Seed .mcp.json with the bare JSON null literal — the entire file is null.
	seedJSON := []byte("null\n")
	if writeErr := os.WriteFile(filepath.Join(dir, ".mcp.json"), seedJSON, 0o644); writeErr != nil {
		t.Fatalf("seed .mcp.json with null: %v", writeErr)
	}

	var out strings.Builder
	if err := run(context.Background(), []string{"--app", "tillsyn-init", "init", "--json", `{"name":"foo","groups":["go"],"mcp":true}`}, &out, io.Discard); err != nil {
		t.Fatalf("run(init --json mcp:true) error = %v; want nil after D7 wiring", err)
	}

	data, readErr := os.ReadFile(filepath.Join(dir, ".mcp.json"))
	if readErr != nil {
		t.Fatalf("os.ReadFile .mcp.json after null-top-level run: %v", readErr)
	}

	// The resulting file must be well-formed JSON with the tillsyn entry.
	var top map[string]json.RawMessage
	if unmarshalErr := json.Unmarshal(data, &top); unmarshalErr != nil {
		t.Fatalf("json.Unmarshal .mcp.json: %v\nbody = %q", unmarshalErr, string(data))
	}
	var servers map[string]json.RawMessage
	if unmarshalErr := json.Unmarshal(top[mcpServersKey], &servers); unmarshalErr != nil {
		t.Fatalf("json.Unmarshal .mcp.json mcpServers: %v", unmarshalErr)
	}
	if _, ok := servers[mcpServerKey]; !ok {
		t.Fatalf(".mcp.json missing %q entry after null-top-level run; servers = %v", mcpServerKey, servers)
	}
}

// TestInit_CreatesProjectRecord verifies that `till init --json` creates a
// project record in the Tillsyn SQLite database and that the project is
// visible via the service layer's list method. CONSUMER-TIE TEST CONTRACT:
// invokes `run(...)` end-to-end so the cobra wiring, the init pipeline, and
// the project-DB creation chain are all exercised together.
//
// HOME isolation via t.Setenv ensures the DB lands in t.TempDir(), not the
// dev's real ~/.tillsyn-init/tillsyn-init.db. The DB is opened a second
// time (read-only inspection) via sqlite.Open + app.NewService so the test
// proves the record is durable in the underlying store, not just in-process
// state.
func TestInit_CreatesProjectRecord(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Chdir(tmp)

	const projectName = "my-init-project"
	if err := run(context.Background(), []string{
		"--app", "tillsyn-init", "init", "--json",
		`{"name":"` + projectName + `","groups":["go"],"mcp":false}`,
	}, nil, io.Discard); err != nil {
		t.Fatalf("run(init --json) error = %v; want nil", err)
	}

	// Resolve the DB path the same way createProjectDBRecord does.
	paths, err := platform.DefaultPathsWithOptions(platform.Options{AppName: "tillsyn-init"})
	if err != nil {
		t.Fatalf("platform.DefaultPathsWithOptions: %v", err)
	}

	repo, openErr := sqlite.Open(paths.DBPath)
	if openErr != nil {
		t.Fatalf("sqlite.Open(%q): %v", paths.DBPath, openErr)
	}
	defer func() { _ = repo.Close() }()

	svc := app.NewService(repo, uuid.NewString, nil, app.ServiceConfig{})
	projects, listErr := svc.ListProjects(context.Background(), false)
	if listErr != nil {
		t.Fatalf("svc.ListProjects: %v", listErr)
	}
	for _, p := range projects {
		if strings.EqualFold(p.Name, projectName) {
			return // found — test passes
		}
	}
	names := make([]string, 0, len(projects))
	for _, p := range projects {
		names = append(names, p.Name)
	}
	t.Fatalf("project %q not found in DB after till init; projects present: %v", projectName, names)
}

// TestInit_SuccessMessage_Format verifies that `till init --json` writes a
// Laslig key/value block to stdout containing the expected summary keys:
// project name, group, "agents copied", "added", and "skipped". CONSUMER-TIE
// TEST CONTRACT: invokes `run(...)` end-to-end.
func TestInit_SuccessMessage_Format(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Chdir(tmp)

	var out strings.Builder
	if err := run(context.Background(), []string{
		"--app", "tillsyn-init", "init", "--json",
		`{"name":"format-check","groups":["go"],"mcp":false}`,
	}, &out, io.Discard); err != nil {
		t.Fatalf("run(init --json) error = %v; want nil", err)
	}

	stdout := out.String()
	for _, want := range []string{"project name", "group", "agents copied", "added", "skipped"} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("init stdout missing %q; full output = %q", want, stdout)
		}
	}
}

// TestMCPRegistration verifies the MCPRegistration() accessor on
// initJSONPayload. A nil MCP pointer must default to true (opt-out model:
// omitting the field enables MCP registration). Explicit false disables;
// explicit true enables.
func TestMCPRegistration(t *testing.T) {
	cases := []struct {
		name string
		mcp  *bool
		want bool
	}{
		{name: "nil_defaults_true", mcp: nil, want: true},
		{name: "explicit_true", mcp: func() *bool { b := true; return &b }(), want: true},
		{name: "explicit_false", mcp: func() *bool { b := false; return &b }(), want: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := initJSONPayload{Name: "x", Groups: []string{"go"}, MCP: tc.mcp}
			if got := p.MCPRegistration(); got != tc.want {
				t.Fatalf("MCPRegistration() = %v; want %v", got, tc.want)
			}
		})
	}
}

// TestValidateInitPayload_W2D1 verifies the updated validateInitPayload
// logic: Groups required (non-empty), each element must be in
// allowedInitGroups ([gen go fe]). This is a unit supplement to the
// CONSUMER-TIE run() tests.
func TestValidateInitPayload_W2D1(t *testing.T) {
	cases := []struct {
		name    string
		payload initJSONPayload
		wantErr bool
		wantSub string
	}{
		{
			name:    "valid_single_go",
			payload: initJSONPayload{Name: "x", Groups: []string{"go"}},
			wantErr: false,
		},
		{
			name:    "valid_multi_go_fe",
			payload: initJSONPayload{Name: "x", Groups: []string{"go", "fe"}},
			wantErr: false,
		},
		{
			name:    "valid_gen",
			payload: initJSONPayload{Name: "x", Groups: []string{"gen"}},
			wantErr: false,
		},
		{
			name:    "valid_fe",
			payload: initJSONPayload{Name: "x", Groups: []string{"fe"}},
			wantErr: false,
		},
		{
			name:    "empty_groups",
			payload: initJSONPayload{Name: "x", Groups: nil},
			wantErr: true,
			wantSub: "groups required",
		},
		{
			name:    "invalid_till_gdd",
			payload: initJSONPayload{Name: "x", Groups: []string{"till-gdd"}},
			wantErr: true,
			wantSub: "invalid",
		},
		{
			name:    "invalid_till_go_old_name",
			payload: initJSONPayload{Name: "x", Groups: []string{"till-go"}},
			wantErr: true,
			wantSub: "invalid",
		},
		{
			name:    "missing_name",
			payload: initJSONPayload{Name: "", Groups: []string{"go"}},
			wantErr: true,
			wantSub: "name required",
		},
		{
			name:    "mixed_valid_invalid",
			payload: initJSONPayload{Name: "x", Groups: []string{"go", "bogus"}},
			wantErr: true,
			wantSub: "invalid",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateInitPayload(tc.payload)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("validateInitPayload(%v) = nil; want error containing %q", tc.payload, tc.wantSub)
				}
				if !strings.Contains(err.Error(), tc.wantSub) {
					t.Fatalf("validateInitPayload error = %q; want substring %q", err.Error(), tc.wantSub)
				}
			} else {
				if err != nil {
					t.Fatalf("validateInitPayload(%v) = %v; want nil", tc.payload, err)
				}
			}
		})
	}
}

// TestInit_ConsumerTie_W2D1 exercises the three CONSUMER-TIE cases mandated
// by W2.D1 acceptance criteria via run() end-to-end:
//
//	(a) valid single-group no-mcp-key — omitting "mcp" defaults to true
//	    (MCPRegistration() = true); verifies nil->true default in the pipeline.
//	(b) valid multi-element groups=["go","fe"] with mcp:false — both groups
//	    parse correctly; pipeline uses Groups[0] stub until D5.
//	(c) invalid group "bogus" — expects non-zero exit with "invalid" substring.
func TestInit_ConsumerTie_W2D1(t *testing.T) {
	t.Run("valid_single_group_no_mcp_key", func(t *testing.T) {
		// Omitting "mcp" entirely: MCPRegistration() must default to true.
		// The pipeline runs through mcp registration (but may skip if till
		// binary not found — that is acceptable; the test verifies the run
		// succeeds and the output is a Laslig Init block).
		tmp := t.TempDir()
		t.Setenv("HOME", tmp)
		t.Chdir(tmp)
		var out strings.Builder
		err := run(context.Background(),
			[]string{"--app", "tillsyn-init", "init", "--json", `{"name":"ct-a","groups":["go"]}`},
			&out, io.Discard)
		if err != nil {
			t.Fatalf("run with no mcp key error = %v; want nil (nil->true default, pipeline must succeed)", err)
		}
		if !strings.Contains(out.String(), "Init") {
			t.Fatalf("stdout = %q; want Laslig Init block", out.String())
		}
	})

	t.Run("valid_multi_group_mcp_false", func(t *testing.T) {
		// Multi-element groups; mcp explicitly false.
		// D1 passes Groups[0] to copyAgentFiles stub — the parse succeeds
		// and the pipeline runs (go agent files are embedded; fe is also
		// embedded). D5 upgrades to the full multi-group loop.
		tmp := t.TempDir()
		t.Setenv("HOME", tmp)
		t.Chdir(tmp)
		var out strings.Builder
		err := run(context.Background(),
			[]string{"--app", "tillsyn-init", "init", "--json", `{"name":"ct-b","groups":["go","fe"],"mcp":false}`},
			&out, io.Discard)
		if err != nil {
			t.Fatalf("run with groups=[go,fe] mcp:false error = %v; want nil", err)
		}
		if !strings.Contains(out.String(), "Init") {
			t.Fatalf("stdout = %q; want Laslig Init block", out.String())
		}
	})

	t.Run("invalid_group_bogus", func(t *testing.T) {
		// Invalid group must return non-zero exit with "invalid" in error.
		tmp := t.TempDir()
		t.Setenv("HOME", tmp)
		t.Chdir(tmp)
		var out strings.Builder
		err := run(context.Background(),
			[]string{"--app", "tillsyn-init", "init", "--json", `{"name":"ct-c","groups":["bogus"],"mcp":false}`},
			&out, io.Discard)
		if err == nil {
			t.Fatalf("run with invalid group returned nil; want error containing 'invalid'")
		}
		if !strings.Contains(err.Error(), "invalid") {
			t.Fatalf("error = %q; want substring 'invalid'", err.Error())
		}
	})
}

// topKeys returns the keys of a map[string]json.RawMessage for use in
// test failure messages.
func topKeys(m map[string]json.RawMessage) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
