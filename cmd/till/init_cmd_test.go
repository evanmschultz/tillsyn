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
// walk: the user presses Enter on the default name then immediately presses
// Enter again to confirm the default group selection, then Enter again to
// confirm the MCP default YES. After D3, the group picker pre-selects "gen"
// (row 0), so one Enter on the group step advances to the MCP confirm step.
// After D4, a second Enter on the MCP step accepts the default YES and
// completes the walk with MCPRegistration() = true.
//
// Drop 4c.6.1 W2.D4: MCP confirm step added between group and done.
// The walk is now name -> group -> MCP -> done (three Enter presses).
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
		return strings.Contains(string(out), "gen")
	}, teatest.WithDuration(2*time.Second), teatest.WithCheckInterval(10*time.Millisecond))

	// Step 2: press Enter to confirm the default group selection (["gen"]
	// pre-selected). Advances to the MCP confirm step (W2.D4).
	tm.Send(tea.KeyPressMsg{Code: tea.KeyEnter})

	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return strings.Contains(string(out), "mcp.json") || strings.Contains(string(out), "Y/n")
	}, teatest.WithDuration(2*time.Second), teatest.WithCheckInterval(10*time.Millisecond))

	// Step 3: press Enter to accept the default YES on the MCP confirm step
	// (D4: defaultYes=true). Advances to initTUIStepDone.
	tm.Send(tea.KeyPressMsg{Code: tea.KeyEnter})

	tm.WaitFinished(t, teatest.WithFinalTimeout(2*time.Second))

	final, ok := tm.FinalModel(t).(initTUIModel)
	if !ok {
		t.Fatalf("FinalModel type = %T; want initTUIModel", tm.FinalModel(t))
	}
	if !final.Done() {
		t.Fatalf("final.Done() = false; want true after enter on MCP step")
	}
	if final.Cancelled() {
		t.Fatalf("final.Cancelled() = true; want false after a complete walk")
	}
	got := final.Payload()
	if got.Name != wantName {
		t.Fatalf("Payload().Name = %q; want %q", got.Name, wantName)
	}
	if len(got.Groups) != 1 || got.Groups[0] != "gen" {
		t.Fatalf("Payload().Groups = %v; want [\"gen\"] (default pre-selection)", got.Groups)
	}
	if !got.MCPRegistration() {
		t.Fatalf("Payload().MCPRegistration() = false; want true (D4: default YES on Enter)")
	}
}

// TestRunInitTUI_SelectsFeRow verifies that the "fe" group (row 2) is
// selectable as the sole group. The key sequence deselects the default "gen"
// and selects "fe" only:
//
//	Enter (name) → j → j (cursor to fe) → Space (select fe) →
//	k → k (cursor to gen) → Space (deselect gen) → j → j (cursor to fe) →
//	Enter (confirm group) → Enter (accept MCP default YES)
//
// Final selection: Groups = ["fe"].
//
// Drop 4c.6.1 W2.D3: picker_multi.go uses j/k (not Up/Down) for navigation
// and Space for toggle. gen is pre-selected by default; user must deselect it
// to get an exclusive fe selection.
//
// Drop 4c.6.1 W2.D4: MCP confirm step added after group step. An extra Enter
// accepts the default YES.
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

	// Navigate to fe (row 2) and select it.
	tm.Send(tea.KeyPressMsg{Code: 'j'})          // cursor gen(0) -> go(1)
	tm.Send(tea.KeyPressMsg{Code: 'j'})          // cursor go(1) -> fe(2)
	tm.Send(tea.KeyPressMsg{Code: tea.KeySpace}) // select fe; selected={0:true,2:true}

	// Navigate back to gen and deselect it.
	tm.Send(tea.KeyPressMsg{Code: 'k'})          // cursor fe(2) -> go(1)
	tm.Send(tea.KeyPressMsg{Code: 'k'})          // cursor go(1) -> gen(0)
	tm.Send(tea.KeyPressMsg{Code: tea.KeySpace}) // deselect gen; selected={0:false,2:true}

	// Navigate back to fe for clarity, then confirm group selection.
	// Advances to MCP confirm step (W2.D4).
	tm.Send(tea.KeyPressMsg{Code: 'j'})          // cursor gen(0) -> go(1)
	tm.Send(tea.KeyPressMsg{Code: 'j'})          // cursor go(1) -> fe(2)
	tm.Send(tea.KeyPressMsg{Code: tea.KeyEnter}) // confirm group; -> initTUIStepMCP

	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return strings.Contains(string(out), "mcp.json") || strings.Contains(string(out), "Y/n")
	}, teatest.WithDuration(2*time.Second), teatest.WithCheckInterval(10*time.Millisecond))

	// Accept default YES on MCP confirm step.
	tm.Send(tea.KeyPressMsg{Code: tea.KeyEnter}) // -> initTUIStepDone

	tm.WaitFinished(t, teatest.WithFinalTimeout(2*time.Second))

	final, ok := tm.FinalModel(t).(initTUIModel)
	if !ok {
		t.Fatalf("FinalModel type = %T; want initTUIModel", tm.FinalModel(t))
	}
	if final.Cancelled() {
		t.Fatalf("final.Cancelled() = true; want false (walk completed with fe selection)")
	}
	if !final.Done() {
		t.Fatalf("final.Done() = false; want true after enter on MCP step")
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
//   - at least 7 agent .md files under `.tillsyn/agents/go/` (subdir-per-group
//     layout; D5 refactor). `go` currently ships 10 .md files; the floor is 7.
//   - `agents.toml` at the project root, copied from the embedded example.
//   - `.gitignore` at the project root containing the literal line
//     `agents.local.toml`.
//
// **D5 update**: agent files land in `.tillsyn/agents/<group>/` (subdir-per-group),
// NOT flat at `.tillsyn/agents/`. The spot-check uses the group subdir path.
// **D7 update**: the pipeline now returns nil (project-DB record created).
func TestInit_FreshDir_CopiesAllFiles(t *testing.T) {
	dir, err := runInitJSONInTempDir(t, `{"name":"foo","groups":["go"],"mcp":false}`)
	if err != nil {
		t.Fatalf("run(init --json) error = %v; want nil after D7 wiring", err)
	}

	// D5: files land in the per-group subdir, not the flat agents root.
	goDir := filepath.Join(dir, ".tillsyn", "agents", "go")
	entries, readErr := os.ReadDir(goDir)
	if readErr != nil {
		t.Fatalf("os.ReadDir(%q): %v — per-group subdir must be created by D5", goDir, readErr)
	}
	mdCount := 0
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") {
			mdCount++
		}
	}
	if mdCount < 7 {
		t.Fatalf("agent .md count under %q = %d; want >= 7 (SKETCH §11.1 standard set)", goDir, mdCount)
	}

	// Spot-check a representative standard agent .md exists in the go/ subdir.
	if _, statErr := os.Stat(filepath.Join(goDir, "builder-agent.md")); statErr != nil {
		t.Fatalf("os.Stat(go/builder-agent.md): %v (D5 subdir-per-group copy required)", statErr)
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
	// D5 update: "group" key renamed to "groups" (comma-joined list) in Laslig summary.
	for _, want := range []string{"project name", "groups", "agents copied", "added", "skipped"} {
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
		// D5: copyAgentFiles now processes all groups; both go and fe agent
		// files are embedded and will be copied to their respective subdirs.
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

// TestRunInitPipeline_FLATDetection exercises the three CONSUMER-TIE cases
// mandated by W2.D2 acceptance criteria for FLAT agent layout detection:
//
//	(a) FLAT layout present (.tillsyn/agents/ contains a .md file at root)
//	    -> run() returns non-zero error containing "FLAT agent layout".
//	(b) Old-schema agents.toml present (first line starts with "[agents.")
//	    -> run() returns non-zero error containing "agents.toml uses the old".
//	(c) Clean state (no FLAT agents dir, no old-schema agents.toml)
//	    -> both checks pass, run() returns nil.
func TestRunInitPipeline_FLATDetection(t *testing.T) {
	t.Run("flat_layout_present", func(t *testing.T) {
		tmp := t.TempDir()
		t.Setenv("HOME", tmp)
		t.Chdir(tmp)

		// Seed a FLAT-layout agents dir with a .md file directly at root.
		agentsDir := filepath.Join(tmp, ".tillsyn", "agents")
		if err := os.MkdirAll(agentsDir, 0o755); err != nil {
			t.Fatalf("MkdirAll %q: %v", agentsDir, err)
		}
		if err := os.WriteFile(filepath.Join(agentsDir, "builder-agent.md"), []byte("# builder\n"), 0o644); err != nil {
			t.Fatalf("WriteFile builder-agent.md: %v", err)
		}

		var out strings.Builder
		err := run(context.Background(), []string{
			"--app", "tillsyn-init", "init", "--json",
			`{"name":"flattest","groups":["go"],"mcp":false}`,
		}, &out, io.Discard)
		if err == nil {
			t.Fatalf("run() = nil; want error containing 'FLAT agent layout'")
		}
		if !strings.Contains(err.Error(), "FLAT agent layout") {
			t.Fatalf("error = %q; want substring 'FLAT agent layout'", err.Error())
		}
	})

	t.Run("clean_state_no_flat_layout", func(t *testing.T) {
		// Clean temp dir — no .tillsyn/agents/, no agents.toml. Both checks
		// must pass and the pipeline must succeed.
		tmp := t.TempDir()
		t.Setenv("HOME", tmp)
		t.Chdir(tmp)

		var out strings.Builder
		err := run(context.Background(), []string{
			"--app", "tillsyn-init", "init", "--json",
			`{"name":"cleantest","groups":["go"],"mcp":false}`,
		}, &out, io.Discard)
		if err != nil {
			t.Fatalf("run() on clean state error = %v; want nil (both detection checks should pass)", err)
		}
		if !strings.Contains(out.String(), "Init") {
			t.Fatalf("stdout = %q; want Laslig Init block", out.String())
		}
	})
}

// TestRunInitPipeline_OldSchemaDetection exercises the old-schema agents.toml
// detection mandated by W2.D2 acceptance criteria:
//
//	(a) agents.toml with "[agents." prefix on a line -> non-zero error.
//	(b) agents.toml absent -> no-op (clean-state covered by FLATDetection above).
//	(c) agents.toml with "[agents]" only (no dot) -> no match, pipeline succeeds.
func TestRunInitPipeline_OldSchemaDetection(t *testing.T) {
	t.Run("old_schema_first_line", func(t *testing.T) {
		tmp := t.TempDir()
		t.Setenv("HOME", tmp)
		t.Chdir(tmp)

		// Seed agents.toml with old-schema header.
		if err := os.WriteFile(filepath.Join(tmp, "agents.toml"), []byte("[agents.build]\nfoo = \"bar\"\n"), 0o644); err != nil {
			t.Fatalf("WriteFile agents.toml: %v", err)
		}

		var out strings.Builder
		err := run(context.Background(), []string{
			"--app", "tillsyn-init", "init", "--json",
			`{"name":"oldschema","groups":["go"],"mcp":false}`,
		}, &out, io.Discard)
		if err == nil {
			t.Fatalf("run() = nil; want error containing 'agents.toml uses the old'")
		}
		if !strings.Contains(err.Error(), "agents.toml uses the old") {
			t.Fatalf("error = %q; want substring 'agents.toml uses the old'", err.Error())
		}
	})

	t.Run("no_dot_agents_section_not_old_schema", func(t *testing.T) {
		// [agents] without a trailing dot must NOT trigger the check. The
		// detection is prefix "[agents." (with dot) only.
		tmp := t.TempDir()
		t.Setenv("HOME", tmp)
		t.Chdir(tmp)

		// Write agents.toml with [agents] (no dot) — must not trigger detection.
		// Note: copyAgentsTOML skips if agents.toml already exists, so the
		// pipeline proceeds normally.
		if err := os.WriteFile(filepath.Join(tmp, "agents.toml"), []byte("[agents]\nfoo = \"bar\"\n"), 0o644); err != nil {
			t.Fatalf("WriteFile agents.toml: %v", err)
		}

		var out strings.Builder
		err := run(context.Background(), []string{
			"--app", "tillsyn-init", "init", "--json",
			`{"name":"nodotschema","groups":["go"],"mcp":false}`,
		}, &out, io.Discard)
		if err != nil {
			t.Fatalf("run() error = %v; want nil ([agents] without dot must not trigger old-schema detection)", err)
		}
	})

	t.Run("old_schema_within_first_20_lines", func(t *testing.T) {
		// Seed agents.toml with a comment block (15 lines) before the
		// [agents.plan] section — must still be detected within the 20-line
		// heuristic window.
		tmp := t.TempDir()
		t.Setenv("HOME", tmp)
		t.Chdir(tmp)

		var body strings.Builder
		for i := range 15 {
			body.WriteString("# comment line ")
			body.WriteString(strings.Repeat("x", i))
			body.WriteString("\n")
		}
		body.WriteString("[agents.plan]\nfoo = \"bar\"\n")
		if err := os.WriteFile(filepath.Join(tmp, "agents.toml"), []byte(body.String()), 0o644); err != nil {
			t.Fatalf("WriteFile agents.toml: %v", err)
		}

		var out strings.Builder
		err := run(context.Background(), []string{
			"--app", "tillsyn-init", "init", "--json",
			`{"name":"deepschema","groups":["go"],"mcp":false}`,
		}, &out, io.Discard)
		if err == nil {
			t.Fatalf("run() = nil; want error (old-schema within 20 lines must be detected)")
		}
		if !strings.Contains(err.Error(), "agents.toml uses the old") {
			t.Fatalf("error = %q; want substring 'agents.toml uses the old'", err.Error())
		}
	})

	t.Run("old_schema_beyond_20_lines_not_detected", func(t *testing.T) {
		// The 20-line heuristic: [agents.X] appearing AFTER line 20 is NOT
		// detected. This is the documented pragmatic bound (W2.D2 RiskNotes).
		tmp := t.TempDir()
		t.Setenv("HOME", tmp)
		t.Chdir(tmp)

		var body strings.Builder
		for range 25 {
			body.WriteString("# comment\n")
		}
		body.WriteString("[agents.plan]\nfoo = \"bar\"\n")
		if err := os.WriteFile(filepath.Join(tmp, "agents.toml"), []byte(body.String()), 0o644); err != nil {
			t.Fatalf("WriteFile agents.toml: %v", err)
		}

		var out strings.Builder
		err := run(context.Background(), []string{
			"--app", "tillsyn-init", "init", "--json",
			`{"name":"farschema","groups":["go"],"mcp":false}`,
		}, &out, io.Discard)
		// No error expected — the old-schema line is beyond the 20-line window.
		if err != nil {
			t.Fatalf("run() error = %v; want nil (old-schema beyond 20-line window should not be detected)", err)
		}
	})
}

// TestInitTUIModel_GroupMultiSelect exercises the picker_multi.go-backed
// group selection step:
//
//   - default_gen_preselected: pressing Enter immediately after advancing to
//     the group step confirms Groups = ["gen"] — the default pre-selection.
//   - multi_select_go_and_fe: deselects gen, selects go and fe via j/Space,
//     confirms Groups = ["go", "fe"] in order.
//   - min1_empty_selection_refuses_advance: after deselecting gen, pressing
//     Enter does NOT advance the walk; step remains initTUIStepGroup and
//     emptyHint is set.
//
// These tests drive the model directly (no teatest program) so they can
// inspect mid-walk state (emptyHint, step) without relying on WaitFinished.
func TestInitTUIModel_GroupMultiSelect(t *testing.T) {
	update := func(m initTUIModel, msg tea.Msg) initTUIModel {
		t.Helper()
		next, _ := m.Update(msg)
		cast, ok := next.(initTUIModel)
		if !ok {
			t.Fatalf("Update returned unexpected type %T; want initTUIModel", next)
		}
		return cast
	}
	advanceToGroupStep := func(t *testing.T, cwd string) initTUIModel {
		t.Helper()
		m := newInitTUIModel(cwd)
		// Accept default name (Enter advances name -> group step).
		m = update(m, tea.KeyPressMsg{Code: tea.KeyEnter})
		if m.step != initTUIStepGroup {
			t.Fatalf("step after name Enter = %v; want initTUIStepGroup", m.step)
		}
		return m
	}

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd: %v", err)
	}

	t.Run("default_gen_preselected", func(t *testing.T) {
		m := advanceToGroupStep(t, cwd)
		// gen is pre-selected — pressing Enter immediately confirms ["gen"] and
		// advances to the MCP confirm step (W2.D4).
		m = update(m, tea.KeyPressMsg{Code: tea.KeyEnter})
		if m.step != initTUIStepMCP {
			t.Fatalf("step after group Enter = %v; want initTUIStepMCP (D4: group -> MCP)", m.step)
		}
		// Accept the default YES on the MCP confirm step.
		m = update(m, tea.KeyPressMsg{Code: tea.KeyEnter})
		if m.step != initTUIStepDone {
			t.Fatalf("step after MCP Enter = %v; want initTUIStepDone", m.step)
		}
		if got := m.finalPayload.Groups; len(got) != 1 || got[0] != "gen" {
			t.Fatalf("Groups = %v; want [\"gen\"] (default pre-selection)", got)
		}
	})

	t.Run("multi_select_go_and_fe", func(t *testing.T) {
		m := advanceToGroupStep(t, cwd)
		// Deselect gen (cursor is at 0, gen is pre-selected).
		m = update(m, tea.KeyPressMsg{Code: tea.KeySpace}) // deselect gen
		// Move to go (1) and select it.
		m = update(m, tea.KeyPressMsg{Code: 'j'})          // cursor -> 1
		m = update(m, tea.KeyPressMsg{Code: tea.KeySpace}) // select go
		// Move to fe (2) and select it.
		m = update(m, tea.KeyPressMsg{Code: 'j'})          // cursor -> 2
		m = update(m, tea.KeyPressMsg{Code: tea.KeySpace}) // select fe
		// Confirm group selection -> MCP confirm step (W2.D4).
		m = update(m, tea.KeyPressMsg{Code: tea.KeyEnter})
		if m.step != initTUIStepMCP {
			t.Fatalf("step after group Enter = %v; want initTUIStepMCP (D4: group -> MCP)", m.step)
		}
		// Accept the default YES on the MCP confirm step.
		m = update(m, tea.KeyPressMsg{Code: tea.KeyEnter})
		if m.step != initTUIStepDone {
			t.Fatalf("step after MCP Enter = %v; want initTUIStepDone", m.step)
		}
		got := m.finalPayload.Groups
		if len(got) != 2 || got[0] != "go" || got[1] != "fe" {
			t.Fatalf("Groups = %v; want [\"go\",\"fe\"]", got)
		}
	})

	t.Run("min1_empty_selection_refuses_advance", func(t *testing.T) {
		m := advanceToGroupStep(t, cwd)
		// Deselect gen (the only pre-selected item) — now nothing is selected.
		m = update(m, tea.KeyPressMsg{Code: tea.KeySpace}) // deselect gen; selected={}
		// Press Enter — must NOT advance (empty selection is rejected).
		m = update(m, tea.KeyPressMsg{Code: tea.KeyEnter})
		if m.step != initTUIStepGroup {
			t.Fatalf("step after Enter with empty selection = %v; want initTUIStepGroup (min-1 enforcement)", m.step)
		}
		if m.emptyHint == "" {
			t.Fatalf("emptyHint = \"\"; want a non-empty hint message after Enter on empty selection")
		}
		// Now select gen again and confirm -> MCP confirm step (W2.D4).
		m = update(m, tea.KeyPressMsg{Code: tea.KeySpace}) // re-select gen
		m = update(m, tea.KeyPressMsg{Code: tea.KeyEnter})
		if m.step != initTUIStepMCP {
			t.Fatalf("step after re-selecting gen and Enter = %v; want initTUIStepMCP (D4: group -> MCP)", m.step)
		}
		if m.emptyHint != "" {
			t.Fatalf("emptyHint = %q; want \"\" after successful group confirmation", m.emptyHint)
		}
		// Accept MCP confirm.
		m = update(m, tea.KeyPressMsg{Code: tea.KeyEnter})
		if m.step != initTUIStepDone {
			t.Fatalf("step after MCP Enter = %v; want initTUIStepDone", m.step)
		}
		if got := m.finalPayload.Groups; len(got) != 1 || got[0] != "gen" {
			t.Fatalf("Groups = %v; want [\"gen\"]", got)
		}
	})
}

// TestRunInit_JSONMode_MultiGroup is the CONSUMER-TIE supplement mandated by
// W2.D3 acceptance criteria. It exercises the JSON multi-group path directly
// via run() end-to-end, verifying that groups:["go","fe"] parses correctly,
// passes validateInitPayload, and runs through the full init pipeline without
// error. The pipeline uses Groups[0] as the copyAgentFiles stub (D5 upgrades
// to full multi-group); D3's concern is parsing and TUI capture, not copy.
func TestRunInit_JSONMode_MultiGroup(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Chdir(tmp)

	var out strings.Builder
	err := run(context.Background(),
		[]string{"--app", "tillsyn-init", "init", "--json", `{"name":"x","groups":["go","fe"],"mcp":false}`},
		&out, io.Discard)
	if err != nil {
		t.Fatalf("run(--json groups:[go,fe]) error = %v; want nil (multi-group JSON path must succeed)", err)
	}
	stdout := out.String()
	if !strings.Contains(stdout, "Init") {
		t.Fatalf("stdout = %q; want Laslig Init block", stdout)
	}
}

// TestInitTUIModel_MCPStep verifies the MCP confirm step added by W2.D4:
//
//   - enter_yes: advance name->group->MCP via Enter x2, then press Enter at
//     the MCP prompt to accept the default YES. Payload().MCPRegistration()
//     must return true and Done() must be true.
//   - n_no: advance to MCP, press 'n' to decline. MCPRegistration() returns
//     false and Done() returns true.
//   - esc_cancel: advance to MCP, press Esc. Cancelled() returns true,
//     Done() returns false.
//
// Tests drive the model directly (no teatest program) to inspect mid-walk
// state without relying on WaitFinished. Pattern mirrors
// TestInitTUIModel_GroupMultiSelect above.
func TestInitTUIModel_MCPStep(t *testing.T) {
	update := func(m initTUIModel, msg tea.Msg) initTUIModel {
		t.Helper()
		next, _ := m.Update(msg)
		cast, ok := next.(initTUIModel)
		if !ok {
			t.Fatalf("Update returned unexpected type %T; want initTUIModel", next)
		}
		return cast
	}

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd: %v", err)
	}

	// advanceToMCPStep drives the model through the name step (Enter) and the
	// group step (Enter on default pre-selection) to arrive at initTUIStepMCP.
	advanceToMCPStep := func(t *testing.T) initTUIModel {
		t.Helper()
		m := newInitTUIModel(cwd)
		// Accept default name.
		m = update(m, tea.KeyPressMsg{Code: tea.KeyEnter})
		if m.step != initTUIStepGroup {
			t.Fatalf("step after name Enter = %v; want initTUIStepGroup", m.step)
		}
		// Confirm default group selection (gen pre-selected).
		m = update(m, tea.KeyPressMsg{Code: tea.KeyEnter})
		if m.step != initTUIStepMCP {
			t.Fatalf("step after group Enter = %v; want initTUIStepMCP", m.step)
		}
		return m
	}

	t.Run("enter_yes", func(t *testing.T) {
		m := advanceToMCPStep(t)
		// Enter accepts the default YES (defaultYes=true in NewConfirm).
		m = update(m, tea.KeyPressMsg{Code: tea.KeyEnter})
		if !m.Done() {
			t.Fatalf("Done() = false after Enter on MCP step; want true")
		}
		if m.Cancelled() {
			t.Fatalf("Cancelled() = true after Enter YES on MCP step; want false")
		}
		if got := m.Payload().MCPRegistration(); !got {
			t.Fatalf("MCPRegistration() = false after Enter YES; want true")
		}
	})

	t.Run("n_no", func(t *testing.T) {
		m := advanceToMCPStep(t)
		// 'n' declines.
		m = update(m, tea.KeyPressMsg{Code: 'n'})
		if !m.Done() {
			t.Fatalf("Done() = false after 'n' on MCP step; want true")
		}
		if m.Cancelled() {
			t.Fatalf("Cancelled() = true after 'n' on MCP step; want false (NO is a valid answer, not a cancel)")
		}
		if got := m.Payload().MCPRegistration(); got {
			t.Fatalf("MCPRegistration() = true after 'n'; want false")
		}
	})

	t.Run("esc_cancel", func(t *testing.T) {
		m := advanceToMCPStep(t)
		// Esc cancels the walk.
		m = update(m, tea.KeyPressMsg{Code: tea.KeyEsc})
		if !m.Cancelled() {
			t.Fatalf("Cancelled() = false after Esc on MCP step; want true")
		}
		if m.Done() {
			t.Fatalf("Done() = true after Esc cancel on MCP step; want false")
		}
	})
}

// TestRunInit_JSONMode_MCPPaths is the CONSUMER-TIE supplement mandated by
// W2.D4 acceptance criteria. It exercises three JSON-mode MCP paths via
// run() end-to-end to verify that D1's MCPRegistration() accessor and the
// D4 pipeline consume the field correctly:
//
//   - mcp_true: {"groups":["go"],"mcp":true} -> MCPRegistration() = true
//     (pipeline runs registerMCPJSON with includeMCP=true).
//   - mcp_false: {"groups":["go"],"mcp":false} -> MCPRegistration() = false
//     (pipeline skips .mcp.json write).
//   - no_mcp_key: {"groups":["go"]} (omitted) -> MCPRegistration() = true
//     (nil *bool defaults to true per D1 opt-out model).
//
// Each case invokes run(...) end-to-end so the cobra wiring, JSON parsing,
// validateInitPayload, runInitJSON -> runInitPipeline -> registerMCPJSON
// chain are all exercised. Success is nil error + Laslig Init block in stdout.
func TestRunInit_JSONMode_MCPPaths(t *testing.T) {
	t.Run("mcp_true", func(t *testing.T) {
		tmp := t.TempDir()
		t.Setenv("HOME", tmp)
		t.Chdir(tmp)
		var out strings.Builder
		err := run(context.Background(),
			[]string{"--app", "tillsyn-init", "init", "--json", `{"name":"x","groups":["go"],"mcp":true}`},
			&out, io.Discard)
		if err != nil {
			t.Fatalf("run(mcp:true) error = %v; want nil", err)
		}
		if !strings.Contains(out.String(), "Init") {
			t.Fatalf("stdout = %q; want Laslig Init block", out.String())
		}
	})

	t.Run("mcp_false", func(t *testing.T) {
		tmp := t.TempDir()
		t.Setenv("HOME", tmp)
		t.Chdir(tmp)
		var out strings.Builder
		err := run(context.Background(),
			[]string{"--app", "tillsyn-init", "init", "--json", `{"name":"x","groups":["go"],"mcp":false}`},
			&out, io.Discard)
		if err != nil {
			t.Fatalf("run(mcp:false) error = %v; want nil", err)
		}
		if !strings.Contains(out.String(), "Init") {
			t.Fatalf("stdout = %q; want Laslig Init block", out.String())
		}
		// mcp:false must NOT create .mcp.json.
		mcpPath := filepath.Join(tmp, ".mcp.json")
		if _, statErr := os.Stat(mcpPath); statErr == nil {
			t.Fatalf(".mcp.json exists at %q; mcp:false must not create the file", mcpPath)
		}
	})

	t.Run("no_mcp_key", func(t *testing.T) {
		tmp := t.TempDir()
		t.Setenv("HOME", tmp)
		t.Chdir(tmp)
		var out strings.Builder
		err := run(context.Background(),
			[]string{"--app", "tillsyn-init", "init", "--json", `{"name":"x","groups":["go"]}`},
			&out, io.Discard)
		if err != nil {
			t.Fatalf("run(no mcp key) error = %v; want nil (nil->true default)", err)
		}
		if !strings.Contains(out.String(), "Init") {
			t.Fatalf("stdout = %q; want Laslig Init block", out.String())
		}
		// Omitting mcp key defaults to true -> .mcp.json should be created
		// (either added or already exists).
		mcpPath := filepath.Join(tmp, ".mcp.json")
		if _, statErr := os.Stat(mcpPath); statErr != nil {
			t.Fatalf(".mcp.json not found at %q; nil mcp key defaults to true -> file must be created: %v", mcpPath, statErr)
		}
	})
}

// TestCopyAgentFiles_SubdirPerGroup verifies the D5 subdir-per-group
// refactor of copyAgentFiles. For each group the function must:
//   - create <destDir>/.tillsyn/agents/<group>/ subdir
//   - copy embedded builtin/agents/<group>/*.md into that subdir (not flat)
//   - skip existing files (idempotent)
//
// Two sub-tests: single-group (go) and multi-group (go + fe).
func TestCopyAgentFiles_SubdirPerGroup(t *testing.T) {
	t.Run("single_group_go", func(t *testing.T) {
		destDir := t.TempDir()
		added, skipped, err := copyAgentFiles(destDir, []string{"go"})
		if err != nil {
			t.Fatalf("copyAgentFiles([go]) error = %v; want nil", err)
		}
		if added < 1 {
			t.Fatalf("copyAgentFiles([go]) added = %d; want >= 1", added)
		}
		if skipped != 0 {
			t.Fatalf("copyAgentFiles([go]) skipped = %d; want 0 on fresh dir", skipped)
		}

		// Subdir must exist.
		goDir := filepath.Join(destDir, ".tillsyn", "agents", "go")
		entries, readErr := os.ReadDir(goDir)
		if readErr != nil {
			t.Fatalf("os.ReadDir(%q): %v — subdir not created", goDir, readErr)
		}
		mdCount := 0
		for _, e := range entries {
			if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") {
				mdCount++
			}
		}
		if mdCount == 0 {
			t.Fatalf("no .md files under %q; want agent files in per-group subdir", goDir)
		}
		if mdCount != added {
			t.Fatalf("os.ReadDir md count = %d; want == added (%d)", mdCount, added)
		}

		// Spot-check representative file in subdir (not flat root).
		goBuilderPath := filepath.Join(goDir, "builder-agent.md")
		if _, statErr := os.Stat(goBuilderPath); statErr != nil {
			t.Fatalf("os.Stat(%q): %v — expected subdir copy not flat", goBuilderPath, statErr)
		}

		// Flat root must NOT contain any .md files.
		flatRoot := filepath.Join(destDir, ".tillsyn", "agents")
		flatEntries, _ := os.ReadDir(flatRoot)
		for _, e := range flatEntries {
			if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") {
				t.Fatalf("flat root %q contains .md file %q; D5 must write to subdir only", flatRoot, e.Name())
			}
		}
	})

	t.Run("multi_group_go_and_fe", func(t *testing.T) {
		destDir := t.TempDir()
		added, skipped, err := copyAgentFiles(destDir, []string{"go", "fe"})
		if err != nil {
			t.Fatalf("copyAgentFiles([go,fe]) error = %v; want nil", err)
		}
		if added < 2 {
			t.Fatalf("copyAgentFiles([go,fe]) added = %d; want >= 2 (at least 1 per group)", added)
		}
		if skipped != 0 {
			t.Fatalf("copyAgentFiles([go,fe]) skipped = %d; want 0 on fresh dir", skipped)
		}

		// Both subdirs must exist with .md files.
		for _, group := range []string{"go", "fe"} {
			groupDir := filepath.Join(destDir, ".tillsyn", "agents", group)
			entries, readErr := os.ReadDir(groupDir)
			if readErr != nil {
				t.Fatalf("os.ReadDir(%q): %v — subdir for group %q not created", groupDir, readErr, group)
			}
			mdCount := 0
			for _, e := range entries {
				if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") {
					mdCount++
				}
			}
			if mdCount == 0 {
				t.Fatalf("no .md files under %q for group %q", groupDir, group)
			}
		}
	})

	t.Run("idempotent_skip", func(t *testing.T) {
		destDir := t.TempDir()
		// First run.
		added1, _, err := copyAgentFiles(destDir, []string{"go"})
		if err != nil {
			t.Fatalf("copyAgentFiles first run error = %v", err)
		}
		// Second run — all files exist, must all be skipped.
		added2, skipped2, err := copyAgentFiles(destDir, []string{"go"})
		if err != nil {
			t.Fatalf("copyAgentFiles second run error = %v", err)
		}
		if added2 != 0 {
			t.Fatalf("copyAgentFiles second run added = %d; want 0 (idempotent)", added2)
		}
		if skipped2 != added1 {
			t.Fatalf("copyAgentFiles second run skipped = %d; want %d (= files from first run)", skipped2, added1)
		}
	})
}

// TestRunInitPipeline_MultiGroup is the CONSUMER-TIE test mandated by W2.D5
// acceptance criteria. It exercises the multi-group path end-to-end via
// run() and verifies that:
//   - single-group: .tillsyn/agents/go/<name>.md created (subdir layout)
//   - multi-group: both .tillsyn/agents/go/ and .tillsyn/agents/fe/ created
//   - Laslig summary contains "groups" key (not "group")
func TestRunInitPipeline_MultiGroup(t *testing.T) {
	t.Run("single_group_subdir_layout", func(t *testing.T) {
		dir, err := runInitJSONInTempDir(t, `{"name":"sub-test","groups":["go"],"mcp":false}`)
		if err != nil {
			t.Fatalf("run(init --json groups:[go]) error = %v; want nil", err)
		}
		goBuilderPath := filepath.Join(dir, ".tillsyn", "agents", "go", "builder-agent.md")
		if _, statErr := os.Stat(goBuilderPath); statErr != nil {
			t.Fatalf("os.Stat(%q): %v — expected subdir layout after D5", goBuilderPath, statErr)
		}
	})

	t.Run("multi_group_both_subdirs_created", func(t *testing.T) {
		dir, err := runInitJSONInTempDir(t, `{"name":"multi-test","groups":["go","fe"],"mcp":false}`)
		if err != nil {
			t.Fatalf("run(init --json groups:[go,fe]) error = %v; want nil", err)
		}
		for _, group := range []string{"go", "fe"} {
			groupDir := filepath.Join(dir, ".tillsyn", "agents", group)
			entries, readErr := os.ReadDir(groupDir)
			if readErr != nil {
				t.Fatalf("os.ReadDir(%q): %v — subdir for group %q not created", groupDir, readErr, group)
			}
			mdCount := 0
			for _, e := range entries {
				if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") {
					mdCount++
				}
			}
			if mdCount == 0 {
				t.Fatalf("no .md files under %q for group %q after multi-group run", groupDir, group)
			}
		}
	})

	t.Run("laslig_summary_groups_key", func(t *testing.T) {
		tmp := t.TempDir()
		t.Setenv("HOME", tmp)
		t.Chdir(tmp)
		var out strings.Builder
		if err := run(context.Background(),
			[]string{"--app", "tillsyn-init", "init", "--json", `{"name":"kv-test","groups":["go","fe"],"mcp":false}`},
			&out, io.Discard); err != nil {
			t.Fatalf("run error = %v; want nil", err)
		}
		if !strings.Contains(out.String(), "groups") {
			t.Fatalf("Laslig summary missing 'groups' key; stdout = %q", out.String())
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

// TestWriteTemplateTOML_HOMETierPresent verifies that writeTemplateTOML reads
// from the HOME tier (~/.tillsyn/templates/<group>.toml) when it exists, and
// writes the content with a [<group>] section header to template.toml.
// CONSUMER-TIE TEST CONTRACT (W2.D6) — drives through run() end-to-end so
// the full runInitPipeline → writeTemplateTOML chain is exercised.
func TestWriteTemplateTOML_HOMETierPresent(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Chdir(dir)

	// Seed the HOME-tier template for the "go" group.
	homeTemplatesDir := filepath.Join(dir, ".tillsyn", "templates")
	if err := os.MkdirAll(homeTemplatesDir, 0o755); err != nil {
		t.Fatalf("MkdirAll HOME templates dir: %v", err)
	}
	customContent := "# custom go template\nmy_custom_key = \"home-tier-value\"\n"
	if err := os.WriteFile(filepath.Join(homeTemplatesDir, "go.toml"), []byte(customContent), 0o644); err != nil {
		t.Fatalf("WriteFile HOME go.toml: %v", err)
	}

	var out strings.Builder
	if err := run(context.Background(), []string{"--app", "tillsyn-init", "init", "--json", `{"name":"home-tier","groups":["go"],"mcp":false}`}, &out, io.Discard); err != nil {
		t.Fatalf("run(init --json) error = %v; want nil", err)
	}

	// template.toml must exist and contain the HOME-tier custom content.
	tplPath := filepath.Join(dir, ".tillsyn", "template.toml")
	data, readErr := os.ReadFile(tplPath)
	if readErr != nil {
		t.Fatalf("os.ReadFile(%q): %v — template.toml not created", tplPath, readErr)
	}
	body := string(data)

	// The [go] section header must be present.
	if !strings.Contains(body, "[go]") {
		t.Fatalf("template.toml = %q; want [go] section header", body)
	}
	// The HOME-tier custom content must be present (not the embedded default).
	if !strings.Contains(body, "home-tier-value") {
		t.Fatalf("template.toml = %q; want HOME-tier custom content (my_custom_key = home-tier-value)", body)
	}

	// Laslig summary must include "template.toml" row.
	if !strings.Contains(out.String(), "template.toml") {
		t.Fatalf("Laslig stdout = %q; want 'template.toml' row", out.String())
	}
}

// TestWriteTemplateTOML_HOMETierAbsent verifies that writeTemplateTOML falls
// back to the embedded builtin/till-<group>.toml when the HOME-tier file does
// not exist. CONSUMER-TIE TEST CONTRACT (W2.D6).
func TestWriteTemplateTOML_HOMETierAbsent(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Chdir(dir)
	// No HOME-tier template created — fall back to embedded.

	var out strings.Builder
	if err := run(context.Background(), []string{"--app", "tillsyn-init", "init", "--json", `{"name":"fallback","groups":["go"],"mcp":false}`}, &out, io.Discard); err != nil {
		t.Fatalf("run(init --json) error = %v; want nil", err)
	}

	tplPath := filepath.Join(dir, ".tillsyn", "template.toml")
	data, readErr := os.ReadFile(tplPath)
	if readErr != nil {
		t.Fatalf("os.ReadFile(%q): %v — template.toml not created on HOME-absent fallback", tplPath, readErr)
	}
	body := string(data)

	// [go] section header must be present (written by writeTemplateTOML).
	if !strings.Contains(body, "[go]") {
		t.Fatalf("template.toml = %q; want [go] section header from embedded fallback", body)
	}
	// File must have non-trivial content from the embedded till-go.toml.
	if len(body) < 100 {
		t.Fatalf("template.toml length = %d; want >= 100 bytes (embedded template has substantial content)", len(body))
	}
}

// TestWriteTemplateTOML_Idempotent verifies that a second `till init` run with
// template.toml already present skips writing it (blanket skip) and reports
// "skipped" in the Laslig output. No error expected.
// CONSUMER-TIE TEST CONTRACT (W2.D6).
func TestWriteTemplateTOML_Idempotent(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Chdir(dir)

	// First run — creates template.toml.
	if err := run(context.Background(), []string{"--app", "tillsyn-init", "init", "--json", `{"name":"idem","groups":["go"],"mcp":false}`}, nil, io.Discard); err != nil {
		t.Fatalf("first run error = %v; want nil", err)
	}

	// Capture content after first run.
	tplPath := filepath.Join(dir, ".tillsyn", "template.toml")
	firstData, readErr := os.ReadFile(tplPath)
	if readErr != nil {
		t.Fatalf("ReadFile after first run: %v", readErr)
	}

	// Second run — must skip template.toml (blanket skip).
	var out strings.Builder
	if err := run(context.Background(), []string{"--app", "tillsyn-init", "init", "--json", `{"name":"idem","groups":["go"],"mcp":false}`}, &out, io.Discard); err != nil {
		t.Fatalf("second run error = %v; want nil (idempotent re-run)", err)
	}

	// File content must be unchanged.
	secondData, readErr2 := os.ReadFile(tplPath)
	if readErr2 != nil {
		t.Fatalf("ReadFile after second run: %v", readErr2)
	}
	if string(firstData) != string(secondData) {
		t.Fatalf("template.toml mutated on second run; first=%q second=%q", string(firstData), string(secondData))
	}

	// Laslig row must say "skipped" (already exists).
	if !strings.Contains(out.String(), "skipped") {
		t.Fatalf("Laslig stdout = %q; want 'skipped' in template.toml row on re-run", out.String())
	}
}

// TestWriteTemplateTOML_PartialStateWarning verifies that when template.toml
// already exists but is missing the [<group>] section for a selected group,
// a warning is printed to stdout but the run exits zero (non-fatal). The
// template.toml file must NOT be modified.
// CONSUMER-TIE TEST CONTRACT (W2.D6).
func TestWriteTemplateTOML_PartialStateWarning(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Chdir(dir)

	// Pre-create template.toml with only [gen] section — missing [go].
	tillsynDir := filepath.Join(dir, ".tillsyn")
	if err := os.MkdirAll(tillsynDir, 0o755); err != nil {
		t.Fatalf("MkdirAll .tillsyn: %v", err)
	}
	existingContent := "[gen]\n# gen content only\n"
	tplPath := filepath.Join(tillsynDir, "template.toml")
	if err := os.WriteFile(tplPath, []byte(existingContent), 0o644); err != nil {
		t.Fatalf("WriteFile template.toml: %v", err)
	}

	// Run init with groups=["go"] — template.toml exists but missing [go].
	var stdout strings.Builder
	if err := run(context.Background(), []string{"--app", "tillsyn-init", "init", "--json", `{"name":"partial","groups":["go"],"mcp":false}`}, &stdout, io.Discard); err != nil {
		t.Fatalf("run error = %v; want nil (partial-state warning is non-fatal)", err)
	}

	// The WARN message must appear in stdout.
	outStr := stdout.String()
	if !strings.Contains(outStr, "WARN") {
		t.Fatalf("stdout = %q; want WARN message about missing [go] section", outStr)
	}
	if !strings.Contains(outStr, "go") {
		t.Fatalf("stdout = %q; want WARN mentioning missing group 'go'", outStr)
	}

	// template.toml must NOT be modified — content must be unchanged.
	afterData, readErr := os.ReadFile(tplPath)
	if readErr != nil {
		t.Fatalf("ReadFile template.toml after partial-state run: %v", readErr)
	}
	if string(afterData) != existingContent {
		t.Fatalf("template.toml was modified; want unchanged\nbefore=%q\nafter=%q", existingContent, string(afterData))
	}
}
