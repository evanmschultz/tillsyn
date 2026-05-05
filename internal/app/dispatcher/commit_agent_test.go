package dispatcher

import (
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/evanmschultz/tillsyn/internal/domain"
	"github.com/evanmschultz/tillsyn/internal/templates"
)

// commit_agent_test.go ships the F.7-CORE F.7.12 unit-test suite for
// CommitAgent.GenerateMessage. Every test injects mocks for GitDiff,
// BuildSpawnCommand, and Monitor so the algorithm path is tested without
// depending on a real spawn pipeline, real claude binary, or real git.
//
// The 7 documented scenarios are:
//
//   1. Happy path — commits both populated, agent returns short msg.
//   2. Missing StartCommit — ErrNoCommitDiff.
//   3. Missing EndCommit — ErrNoCommitDiff.
//   4. Empty diff — diff bytes empty but commits set; spawn proceeds.
//   5. Long message — > CommitMessageMaxLen → ErrCommitMessageTooLong.
//   6. Spawn build fails — wrapped error.
//   7. Monitor fails — wrapped error.

// stubGitDiff is a deterministic GitDiffReader for tests. The diff field
// is returned for every call regardless of from/to commits; the err field
// (when set) is returned without invoking the diff branch.
type stubGitDiff struct {
	diff []byte
	err  error
}

func (s *stubGitDiff) Diff(_ context.Context, _, _ string) ([]byte, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.diff, nil
}

// nopReadCloser wraps an io.Reader as an io.ReadCloser whose Close is a
// no-op. Tests use this to feed canned stream content through the
// CommitAgent's openStream seam without touching disk.
type nopReadCloser struct {
	io.Reader
}

func (nopReadCloser) Close() error { return nil }

// fakeMonitor returns a MonitorRunner that ignores its inputs and returns
// the canned report + error. Tests inject this to assert the
// terminal-report → message extraction path verbatim.
func fakeMonitor(report TerminalReport, err error) MonitorRunner {
	return func(_ context.Context, _ CLIAdapter, _ io.Reader, _ chan<- StreamEvent) (TerminalReport, error) {
		return report, err
	}
}

// fakeMonitorWithSink returns a MonitorRunner that pushes the supplied
// StreamEvents through the sink channel before returning the canned
// report. The events are sent NON-blocking — if the sink is full
// (capacity 256 in production), excess events are dropped. Tests use
// this to exercise the assistant-text capture path.
func fakeMonitorWithSink(report TerminalReport, events []StreamEvent, err error) MonitorRunner {
	return func(_ context.Context, _ CLIAdapter, _ io.Reader, sink chan<- StreamEvent) (TerminalReport, error) {
		if sink != nil {
			for _, ev := range events {
				select {
				case sink <- ev:
				default:
					// Sink full — drop and move on.
				}
			}
		}
		return report, err
	}
}

// fakeSpawnBuilder returns a SpawnBuilder that produces a benign cmd
// (`/bin/true`) plus a SpawnDescriptor whose MCPConfigPath places the
// derived bundle root under tmpDir. Tests use this to drive the algorithm
// through real os.MkdirAll + os.WriteFile + os.Open without standing up a
// real spawn pipeline.
//
// Returning err non-nil makes BuildSpawnCommand fail at the call site.
func fakeSpawnBuilder(tmpDir string, err error) SpawnBuilder {
	return func(item domain.ActionItem, _ domain.Project, _ templates.KindCatalog, _ AuthBundle) (*exec.Cmd, SpawnDescriptor, error) {
		if err != nil {
			return nil, SpawnDescriptor{}, err
		}
		// MCPConfigPath = <bundleRoot>/plugin/.mcp.json so the algorithm
		// derives bundleRoot = filepath.Dir(filepath.Dir(...)) = tmpDir.
		mcpPath := filepath.Join(tmpDir, "plugin", ".mcp.json")
		return exec.Command("/bin/true"), SpawnDescriptor{
			AgentName:     "commit-message-agent",
			Model:         "haiku",
			MCPConfigPath: mcpPath,
			Prompt:        "synthetic prompt for " + item.ID,
			WorkingDir:    "/some/worktree",
		}, nil
	}
}

// commitCatalog builds a templates.KindCatalog with a binding for
// domain.KindCommit so the commit-agent's adapter resolution succeeds.
// The CLIKind is hardcoded to "claude" because that is the only kind the
// resolver maps via the F.7.17 L15 default. Tests that need to drive the
// adapter path register a MockAdapter under CLIKindClaude via
// withMockAdapter.
func commitCatalog() templates.KindCatalog {
	return templates.KindCatalog{
		AgentBindings: map[domain.Kind]templates.AgentBinding{
			domain.KindCommit: {
				AgentName: "commit-message-agent",
				CLIKind:   string(CLIKindClaude),
			},
		},
	}
}

// withMockAdapter wires a fresh MockAdapter into a CommitAgent's
// lookupAdapterFn seam so tests do NOT mutate the package-level
// adaptersMap. Returns the adapter so tests can inspect Calls() if
// needed.
func withMockAdapter(c *CommitAgent) *MockAdapter {
	mock := newMockAdapter()
	c.lookupAdapterFn = func(kind CLIKind) (CLIAdapter, bool) {
		if kind == CLIKindClaude {
			return mock, true
		}
		return nil, false
	}
	return mock
}

// withNopRunCmd injects a runCmd that skips actual exec — tests don't
// care whether the command runs, only that the algorithm threads through
// it. Default cmd.Run on /bin/true would also work but adds non-zero
// latency per test.
func withNopRunCmd(c *CommitAgent) {
	c.runCmd = func(*exec.Cmd) error { return nil }
}

// withStreamReader injects an openStream that returns reader for any
// path. Tests pass an empty reader because the Monitor mock ignores it.
func withStreamReader(c *CommitAgent, body string) {
	c.openStream = func(_ string) (io.ReadCloser, error) {
		return nopReadCloser{strings.NewReader(body)}, nil
	}
}

func sampleActionItem(start, end string) domain.ActionItem {
	return domain.ActionItem{
		ID:          "item-1",
		ProjectID:   "proj-1",
		Kind:        domain.KindBuild,
		StartCommit: start,
		EndCommit:   end,
		Title:       "sample build action item",
	}
}

func sampleProject() domain.Project {
	return domain.Project{
		ID:                  "proj-1",
		RepoPrimaryWorktree: "/tmp/sample-worktree",
	}
}

// TestCommitAgentGenerateMessageHappyPath asserts the canonical happy
// path: BuildSpawnCommand returns a benign cmd, Monitor returns a
// terminal report with a short conventional-commit message, the
// algorithm trims it, validates length, and returns the message.
//
// Asserts: returned message matches; no error; the diff was written to
// <bundleRoot>/context/git_diff.patch.
func TestCommitAgentGenerateMessageHappyPath(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	wantMsg := "feat(foo): add bar"

	c := &CommitAgent{
		GitDiff:           &stubGitDiff{diff: []byte("diff --git a/x b/x\n+hello\n")},
		BuildSpawnCommand: fakeSpawnBuilder(tmp, nil),
		Monitor:           fakeMonitor(TerminalReport{Reason: wantMsg}, nil),
	}
	withMockAdapter(c)
	withNopRunCmd(c)
	withStreamReader(c, "")

	got, err := c.GenerateMessage(
		context.Background(),
		sampleActionItem("a1b2c3", "f4e5d6"),
		sampleProject(),
		commitCatalog(),
		AuthBundle{},
	)
	if err != nil {
		t.Fatalf("GenerateMessage err = %v; want nil", err)
	}
	if got != wantMsg {
		t.Errorf("GenerateMessage = %q; want %q", got, wantMsg)
	}

	// The diff must land at <bundleRoot>/context/git_diff.patch where
	// bundleRoot = filepath.Dir(filepath.Dir(MCPConfigPath)) = tmp.
	patchPath := filepath.Join(tmp, "context", "git_diff.patch")
	wantPatch := "diff --git a/x b/x\n+hello\n"
	gotPatch, readErr := readFileForTest(patchPath)
	if readErr != nil {
		t.Fatalf("read git_diff.patch err = %v; want nil", readErr)
	}
	if string(gotPatch) != wantPatch {
		t.Errorf("git_diff.patch contents = %q; want %q", string(gotPatch), wantPatch)
	}
}

// TestCommitAgentGenerateMessageMissingStartCommit asserts that a missing
// StartCommit field triggers ErrNoCommitDiff before any spawn-side work
// runs. The mocks would explode if invoked (nil GitDiff/BuildSpawnCommand
// is fine since the early-return guards fire first); the test pins that
// short-circuit.
func TestCommitAgentGenerateMessageMissingStartCommit(t *testing.T) {
	t.Parallel()

	c := &CommitAgent{
		GitDiff:           &stubGitDiff{},
		BuildSpawnCommand: fakeSpawnBuilder(t.TempDir(), nil),
		Monitor:           fakeMonitor(TerminalReport{}, nil),
	}

	_, err := c.GenerateMessage(
		context.Background(),
		sampleActionItem("", "f4e5d6"),
		sampleProject(),
		commitCatalog(),
		AuthBundle{},
	)
	if !errors.Is(err, ErrNoCommitDiff) {
		t.Errorf("GenerateMessage err = %v; want errors.Is ErrNoCommitDiff", err)
	}
}

// TestCommitAgentGenerateMessageMissingEndCommit asserts that a missing
// EndCommit field triggers ErrNoCommitDiff before any spawn-side work
// runs. Mirrors the StartCommit case for symmetry.
func TestCommitAgentGenerateMessageMissingEndCommit(t *testing.T) {
	t.Parallel()

	c := &CommitAgent{
		GitDiff:           &stubGitDiff{},
		BuildSpawnCommand: fakeSpawnBuilder(t.TempDir(), nil),
		Monitor:           fakeMonitor(TerminalReport{}, nil),
	}

	_, err := c.GenerateMessage(
		context.Background(),
		sampleActionItem("a1b2c3", ""),
		sampleProject(),
		commitCatalog(),
		AuthBundle{},
	)
	if !errors.Is(err, ErrNoCommitDiff) {
		t.Errorf("GenerateMessage err = %v; want errors.Is ErrNoCommitDiff", err)
	}
}

// TestCommitAgentGenerateMessageEmptyDiff asserts that an empty diff
// (start ≠ end but no changes) does NOT trigger ErrNoCommitDiff — the
// spawn proceeds and the agent decides what to do. This pins the "let
// the agent decide" branch in the algorithm.
//
// The diff file is still written to disk, but with empty content.
func TestCommitAgentGenerateMessageEmptyDiff(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	wantMsg := "chore: empty commit"

	c := &CommitAgent{
		GitDiff:           &stubGitDiff{diff: []byte{}},
		BuildSpawnCommand: fakeSpawnBuilder(tmp, nil),
		Monitor:           fakeMonitor(TerminalReport{Reason: wantMsg}, nil),
	}
	withMockAdapter(c)
	withNopRunCmd(c)
	withStreamReader(c, "")

	got, err := c.GenerateMessage(
		context.Background(),
		sampleActionItem("a1b2c3", "f4e5d6"),
		sampleProject(),
		commitCatalog(),
		AuthBundle{},
	)
	if err != nil {
		t.Fatalf("GenerateMessage err = %v; want nil", err)
	}
	if got != wantMsg {
		t.Errorf("GenerateMessage = %q; want %q", got, wantMsg)
	}

	// Empty diff still writes the file (zero bytes).
	patchPath := filepath.Join(tmp, "context", "git_diff.patch")
	gotPatch, readErr := readFileForTest(patchPath)
	if readErr != nil {
		t.Fatalf("read git_diff.patch err = %v; want nil", readErr)
	}
	if len(gotPatch) != 0 {
		t.Errorf("git_diff.patch len = %d; want 0", len(gotPatch))
	}
}

// TestCommitAgentGenerateMessageMessageTooLong asserts that the agent
// returning a > CommitMessageMaxLen message triggers
// ErrCommitMessageTooLong with the offending text wrapped for diagnosis.
func TestCommitAgentGenerateMessageMessageTooLong(t *testing.T) {
	t.Parallel()

	// Build a message clearly over the 72-char cap.
	longMsg := strings.Repeat("x", CommitMessageMaxLen+10)

	c := &CommitAgent{
		GitDiff:           &stubGitDiff{diff: []byte("diff")},
		BuildSpawnCommand: fakeSpawnBuilder(t.TempDir(), nil),
		Monitor:           fakeMonitor(TerminalReport{Reason: longMsg}, nil),
	}
	withMockAdapter(c)
	withNopRunCmd(c)
	withStreamReader(c, "")

	_, err := c.GenerateMessage(
		context.Background(),
		sampleActionItem("a", "b"),
		sampleProject(),
		commitCatalog(),
		AuthBundle{},
	)
	if !errors.Is(err, ErrCommitMessageTooLong) {
		t.Fatalf("GenerateMessage err = %v; want errors.Is ErrCommitMessageTooLong", err)
	}
	if !strings.Contains(err.Error(), longMsg) {
		t.Errorf("GenerateMessage err = %v; want offending message wrapped in error text", err)
	}
}

// TestCommitAgentGenerateMessageMultilineMessageRejected asserts that a
// message containing a newline is rejected as overlong even when the
// first line is ≤ cap — the "Single-Line Commits" rule prohibits a
// body, not just a long subject.
func TestCommitAgentGenerateMessageMultilineMessageRejected(t *testing.T) {
	t.Parallel()

	multiline := "feat: subject line\n\nbody paragraph here"

	c := &CommitAgent{
		GitDiff:           &stubGitDiff{diff: []byte("diff")},
		BuildSpawnCommand: fakeSpawnBuilder(t.TempDir(), nil),
		Monitor:           fakeMonitor(TerminalReport{Reason: multiline}, nil),
	}
	withMockAdapter(c)
	withNopRunCmd(c)
	withStreamReader(c, "")

	_, err := c.GenerateMessage(
		context.Background(),
		sampleActionItem("a", "b"),
		sampleProject(),
		commitCatalog(),
		AuthBundle{},
	)
	if !errors.Is(err, ErrCommitMessageTooLong) {
		t.Errorf("GenerateMessage err = %v; want errors.Is ErrCommitMessageTooLong", err)
	}
}

// TestCommitAgentGenerateMessageSpawnBuildFails asserts that a
// BuildSpawnCommand failure surfaces as a wrapped error to the caller.
// The spawn-build error itself is captured via errors.Is for the Wave-2
// caller's outcome routing.
func TestCommitAgentGenerateMessageSpawnBuildFails(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("synthetic build failure")
	c := &CommitAgent{
		GitDiff:           &stubGitDiff{diff: []byte("diff")},
		BuildSpawnCommand: fakeSpawnBuilder(t.TempDir(), wantErr),
		Monitor:           fakeMonitor(TerminalReport{}, nil),
	}

	_, err := c.GenerateMessage(
		context.Background(),
		sampleActionItem("a", "b"),
		sampleProject(),
		commitCatalog(),
		AuthBundle{},
	)
	if err == nil {
		t.Fatalf("GenerateMessage err = nil; want non-nil")
	}
	if !errors.Is(err, wantErr) {
		t.Errorf("GenerateMessage err = %v; want errors.Is wantErr", err)
	}
}

// TestCommitAgentGenerateMessageMonitorFails asserts that a Monitor
// failure (e.g. malformed stream, ctx cancellation, reader I/O error)
// surfaces as a wrapped error to the caller.
func TestCommitAgentGenerateMessageMonitorFails(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("synthetic monitor failure")
	c := &CommitAgent{
		GitDiff:           &stubGitDiff{diff: []byte("diff")},
		BuildSpawnCommand: fakeSpawnBuilder(t.TempDir(), nil),
		Monitor:           fakeMonitor(TerminalReport{}, wantErr),
	}
	withMockAdapter(c)
	withNopRunCmd(c)
	withStreamReader(c, "")

	_, err := c.GenerateMessage(
		context.Background(),
		sampleActionItem("a", "b"),
		sampleProject(),
		commitCatalog(),
		AuthBundle{},
	)
	if err == nil {
		t.Fatalf("GenerateMessage err = nil; want non-nil")
	}
	if !errors.Is(err, wantErr) {
		t.Errorf("GenerateMessage err = %v; want errors.Is wantErr", err)
	}
}

// TestCommitAgentGenerateMessageAssistantTextWins asserts that the
// last non-terminal assistant-text event captured via the sink channel
// takes priority over TerminalReport.Reason. This pins the production
// claude path — where commit-message-agent's actual response lands in
// "assistant" events and the terminal "result" event carries only
// telemetry (terminal_reason="completed").
func TestCommitAgentGenerateMessageAssistantTextWins(t *testing.T) {
	t.Parallel()

	wantMsg := "feat(dispatcher): wire commit agent"
	events := []StreamEvent{
		{Type: "assistant", Text: "thinking about the diff…"},
		{Type: "assistant", Text: wantMsg},
	}

	c := &CommitAgent{
		GitDiff:           &stubGitDiff{diff: []byte("diff")},
		BuildSpawnCommand: fakeSpawnBuilder(t.TempDir(), nil),
		Monitor:           fakeMonitorWithSink(TerminalReport{Reason: "completed"}, events, nil),
	}
	withMockAdapter(c)
	withNopRunCmd(c)
	withStreamReader(c, "")

	got, err := c.GenerateMessage(
		context.Background(),
		sampleActionItem("a", "b"),
		sampleProject(),
		commitCatalog(),
		AuthBundle{},
	)
	if err != nil {
		t.Fatalf("GenerateMessage err = %v; want nil", err)
	}
	if got != wantMsg {
		t.Errorf("GenerateMessage = %q; want %q (assistant-text path should win over terminal Reason)", got, wantMsg)
	}
}

// TestCommitAgentGenerateMessageNoTerminalText asserts that a Monitor
// returning a TerminalReport with empty Reason and empty Errors yields
// ErrCommitSpawnNoTerminal — the spawn ran cleanly but produced no
// usable message.
func TestCommitAgentGenerateMessageNoTerminalText(t *testing.T) {
	t.Parallel()

	c := &CommitAgent{
		GitDiff:           &stubGitDiff{diff: []byte("diff")},
		BuildSpawnCommand: fakeSpawnBuilder(t.TempDir(), nil),
		Monitor:           fakeMonitor(TerminalReport{}, nil),
	}
	withMockAdapter(c)
	withNopRunCmd(c)
	withStreamReader(c, "")

	_, err := c.GenerateMessage(
		context.Background(),
		sampleActionItem("a", "b"),
		sampleProject(),
		commitCatalog(),
		AuthBundle{},
	)
	if !errors.Is(err, ErrCommitSpawnNoTerminal) {
		t.Errorf("GenerateMessage err = %v; want errors.Is ErrCommitSpawnNoTerminal", err)
	}
}

// TestCommitAgentGenerateMessageNilReceiver asserts that calling
// GenerateMessage on a nil *CommitAgent yields
// ErrCommitAgentMisconfigured rather than panicking.
func TestCommitAgentGenerateMessageNilReceiver(t *testing.T) {
	t.Parallel()

	var c *CommitAgent
	_, err := c.GenerateMessage(
		context.Background(),
		sampleActionItem("a", "b"),
		sampleProject(),
		commitCatalog(),
		AuthBundle{},
	)
	if !errors.Is(err, ErrCommitAgentMisconfigured) {
		t.Errorf("GenerateMessage on nil receiver err = %v; want errors.Is ErrCommitAgentMisconfigured", err)
	}
}

// TestCommitAgentGenerateMessageNilDeps asserts that each of the three
// required dependency fields produces ErrCommitAgentMisconfigured when
// nil, naming the missing field in the error message.
func TestCommitAgentGenerateMessageNilDeps(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		mutate    func(*CommitAgent)
		wantField string
	}{
		{
			name:      "GitDiff nil",
			mutate:    func(c *CommitAgent) { c.GitDiff = nil },
			wantField: "GitDiff",
		},
		{
			name:      "BuildSpawnCommand nil",
			mutate:    func(c *CommitAgent) { c.BuildSpawnCommand = nil },
			wantField: "BuildSpawnCommand",
		},
		{
			name:      "Monitor nil",
			mutate:    func(c *CommitAgent) { c.Monitor = nil },
			wantField: "Monitor",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			c := &CommitAgent{
				GitDiff:           &stubGitDiff{},
				BuildSpawnCommand: fakeSpawnBuilder(t.TempDir(), nil),
				Monitor:           fakeMonitor(TerminalReport{}, nil),
			}
			tc.mutate(c)

			_, err := c.GenerateMessage(
				context.Background(),
				sampleActionItem("a", "b"),
				sampleProject(),
				commitCatalog(),
				AuthBundle{},
			)
			if !errors.Is(err, ErrCommitAgentMisconfigured) {
				t.Errorf("GenerateMessage err = %v; want errors.Is ErrCommitAgentMisconfigured", err)
			}
			if !strings.Contains(err.Error(), tc.wantField) {
				t.Errorf("GenerateMessage err = %v; want error message to contain %q", err, tc.wantField)
			}
		})
	}
}

// TestCommitAgentGenerateMessageGitDiffFails asserts that a GitDiff
// implementation returning a non-nil error surfaces as a wrapped error
// to the caller (distinct from "empty diff" — that's nil bytes + nil
// error).
func TestCommitAgentGenerateMessageGitDiffFails(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("synthetic git failure")
	c := &CommitAgent{
		GitDiff:           &stubGitDiff{err: wantErr},
		BuildSpawnCommand: fakeSpawnBuilder(t.TempDir(), nil),
		Monitor:           fakeMonitor(TerminalReport{}, nil),
	}

	_, err := c.GenerateMessage(
		context.Background(),
		sampleActionItem("a", "b"),
		sampleProject(),
		commitCatalog(),
		AuthBundle{},
	)
	if !errors.Is(err, wantErr) {
		t.Errorf("GenerateMessage err = %v; want errors.Is wantErr", err)
	}
}

// TestCommitAgentGenerateMessageMissingMCPConfigPath asserts that a
// SpawnDescriptor with empty MCPConfigPath surfaces a clear error rather
// than silently writing the diff under the wrong directory.
func TestCommitAgentGenerateMessageMissingMCPConfigPath(t *testing.T) {
	t.Parallel()

	c := &CommitAgent{
		GitDiff: &stubGitDiff{diff: []byte("diff")},
		BuildSpawnCommand: func(_ domain.ActionItem, _ domain.Project, _ templates.KindCatalog, _ AuthBundle) (*exec.Cmd, SpawnDescriptor, error) {
			return exec.Command("/bin/true"), SpawnDescriptor{
				AgentName:     "commit-message-agent",
				MCPConfigPath: "", // sentinel — empty means descriptor wiring broke
			}, nil
		},
		Monitor: fakeMonitor(TerminalReport{}, nil),
	}

	_, err := c.GenerateMessage(
		context.Background(),
		sampleActionItem("a", "b"),
		sampleProject(),
		commitCatalog(),
		AuthBundle{},
	)
	if err == nil {
		t.Fatalf("GenerateMessage err = nil; want non-nil")
	}
	if !strings.Contains(err.Error(), "MCPConfigPath") {
		t.Errorf("GenerateMessage err = %v; want message to mention MCPConfigPath", err)
	}
}

// TestCommitAgentGenerateMessageNoBindingForKind asserts that a catalog
// missing a binding for KindCommit surfaces a clear error rather than
// nil-derefing the resolved binding downstream.
func TestCommitAgentGenerateMessageNoBindingForKind(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	c := &CommitAgent{
		GitDiff:           &stubGitDiff{diff: []byte("diff")},
		BuildSpawnCommand: fakeSpawnBuilder(tmp, nil),
		Monitor:           fakeMonitor(TerminalReport{Reason: "feat: x"}, nil),
	}
	withMockAdapter(c)
	withNopRunCmd(c)
	withStreamReader(c, "")

	emptyCatalog := templates.KindCatalog{
		AgentBindings: map[domain.Kind]templates.AgentBinding{},
	}
	_, err := c.GenerateMessage(
		context.Background(),
		sampleActionItem("a", "b"),
		sampleProject(),
		emptyCatalog,
		AuthBundle{},
	)
	if err == nil {
		t.Fatalf("GenerateMessage err = nil; want non-nil")
	}
	if !strings.Contains(err.Error(), "no binding") {
		t.Errorf("GenerateMessage err = %v; want message to mention 'no binding'", err)
	}
}

// readFileForTest is a tiny os.ReadFile wrapper kept as a helper so the
// happy-path / empty-diff tests have a single line to assert disk
// contents. The indirection is also a futureproofing seam: a follow-up
// droplet that injects a writableFS for sandbox-friendly testing can swap
// this without touching the assertion sites.
func readFileForTest(path string) ([]byte, error) {
	return os.ReadFile(path)
}
