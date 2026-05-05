// Package dispatcher commit_agent.go ships the F.7-CORE F.7.12 commit-agent
// integration: a CLI-agnostic shim that spawns the commit-message-agent (haiku
// model) via the F.7.17.x spawn pipeline and surfaces a single-line
// conventional-commit message back to the caller.
//
// F.7.12 ships ONLY the agent invocation surface. The post-build commit gate
// itself (F.7.13) is a separate droplet — it consumes CommitAgent.GenerateMessage
// to obtain the message string, then runs `git add` + `git commit` against the
// project worktree under the dispatcher's commit-cadence rules.
//
// Diff context: the caller MUST populate the input action item's StartCommit
// and EndCommit (Drop 4a Wave 1 first-class fields). GenerateMessage shells
// `git diff <start>..<end>` via the injected GitDiffReader and writes the
// patch into the per-spawn bundle's context/git_diff.patch so the downstream
// commit-message-agent prompt template (owned by F.7.13 + render) can read it.
//
// The agent kind for the commit step is domain.KindCommit. The catalog's
// AgentBinding for KindCommit SHOULD set Model="haiku" (cost efficiency) and
// declare a narrow tools allow-list (Read + Bash for `git diff` inspection,
// nothing else). Tool-gating policy is the binding author's concern; F.7.12
// only plumbs the spawn.
//
// Length validation: the agent's terminal-event Text is asserted ≤72 chars per
// the project CLAUDE.md "Single-Line Commits" rule. A longer message returns
// ErrCommitMessageTooLong with the offending text wrapped for diagnosis.
package dispatcher

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/evanmschultz/tillsyn/internal/domain"
	"github.com/evanmschultz/tillsyn/internal/templates"
)

// CommitMessageMaxLen is the per-project single-line commit ceiling (matches
// CLAUDE.md "Single-Line Commits"). The spec was 72 chars verbatim — the
// constant is exported so callers (F.7.13 and beyond) reuse the same number.
const CommitMessageMaxLen = 72

// commitDiffPatchFilename is the filename written under the spawn bundle's
// context/ directory holding `git diff <start>..<end>` output. The
// commit-message-agent's prompt template (F.7.13 / render concern) reads from
// this canonical location. Hardcoding here couples F.7.12 to the prompt
// template's read path; that coupling is intentional — both surfaces are
// owned inside the dispatcher and the filename is part of the contract.
const commitDiffPatchFilename = "git_diff.patch"

// ErrNoCommitDiff is returned by GenerateMessage when the input action item
// has an empty StartCommit or EndCommit. Diff anchors are mandatory inputs
// to the commit agent — the agent has no other way to know what changed.
//
// Callers detect via errors.Is and route the missing-diff condition to a
// "no commit message produced" failure path rather than treating it as a
// transient retryable error.
var ErrNoCommitDiff = errors.New("dispatcher: action item has no start_commit/end_commit for diff context")

// ErrCommitMessageTooLong is returned by GenerateMessage when the
// commit-message-agent emits a terminal-event Text longer than
// CommitMessageMaxLen characters. The offending message is wrapped after
// the sentinel via fmt.Errorf("%w: <message>") so callers can extract the
// text for logging without re-running the spawn.
//
// Callers detect via errors.Is and surface the agent failure to the
// dispatcher's outcome pipeline (Drop 4b post-build pipeline owns the
// retry / fail-the-build decision).
var ErrCommitMessageTooLong = errors.New("dispatcher: commit-agent produced message exceeding length cap")

// ErrCommitAgentMisconfigured is returned by GenerateMessage when CommitAgent
// is missing one of its required dependency fields (GitDiff, BuildSpawnCommand,
// Monitor). The error message names the missing field so production wiring
// loud-fails at first dispatch rather than silently producing nil-pointer
// panics from inside the spawn pipeline.
var ErrCommitAgentMisconfigured = errors.New("dispatcher: commit agent dependency not wired")

// ErrCommitSpawnNoTerminal is returned by GenerateMessage when the spawn
// completed cleanly but the stream-jsonl Monitor never observed a terminal
// event carrying assistant text. This is distinct from a spawn-time failure:
// the process exited but produced nothing the dispatcher can use as a commit
// message. Callers route this to the same "agent produced no message" outcome
// as a crashed spawn.
var ErrCommitSpawnNoTerminal = errors.New("dispatcher: commit-agent spawn produced no terminal text")

// GitDiffReader is the consumer-side seam CommitAgent uses to obtain the
// `git diff <fromCommit>..<toCommit>` patch for the action item under
// commit. The dispatcher's adapter layer supplies a concrete implementation
// that shells out via `git diff` in the project worktree; tests inject a
// deterministic mock.
//
// Note: a parallel GitDiffReader interface lives in
// internal/app/dispatcher/context for the F.7.18 context aggregator. The
// shapes are intentionally identical so a single concrete adapter satisfies
// both interfaces; declaring a local copy here keeps the dispatcher package
// from importing its own subpackage (which would form a cycle since context
// imports dispatcher today).
type GitDiffReader interface {
	// Diff returns the byte content of `git diff <fromCommit>..<toCommit>`.
	// Empty fromCommit OR empty toCommit MUST yield (nil, nil) — the "no
	// commit anchors" case is signaled by empty bytes, not via an error,
	// so consumers can distinguish missing-anchors from real git failures.
	Diff(ctx context.Context, fromCommit, toCommit string) ([]byte, error)
}

// SpawnBuilder is the production signature of BuildSpawnCommand exposed as a
// field on CommitAgent so tests can inject mocks without standing up the
// full spawn pipeline. Production wiring assigns
// dispatcher.BuildSpawnCommand directly.
type SpawnBuilder func(
	item domain.ActionItem,
	project domain.Project,
	catalog templates.KindCatalog,
	auth AuthBundle,
) (*exec.Cmd, SpawnDescriptor, error)

// MonitorRunner is the production signature of the F.7.4 stream-jsonl
// monitor (a thin wrapper over (*Monitor).Run) exposed as a field on
// CommitAgent so tests can inject canned TerminalReports.
//
// Production wiring constructs `NewMonitor(adapter, reader, sink, logger)`
// and binds .Run as the field value. The signature here mirrors the spec
// at F.7.12 acceptance — (ctx, adapter, reader, sink) → (TerminalReport,
// error).
type MonitorRunner func(
	ctx context.Context,
	adapter CLIAdapter,
	reader io.Reader,
	sink chan<- StreamEvent,
) (TerminalReport, error)

// CommitAgent orchestrates spawning the commit-message-agent (haiku) via the
// F.7.17.x spawn pipeline. Reads `git diff <start>..<end>` from the action
// item's first-class commit fields, invokes the agent through the registered
// CLIAdapter, and returns the agent's single-line conventional-commit
// message after enforcing the CommitMessageMaxLen ceiling.
//
// Production wiring assigns:
//
//	commitAgent := &CommitAgent{
//	    GitDiff:           gitDiffAdapter,
//	    BuildSpawnCommand: dispatcher.BuildSpawnCommand,
//	    Monitor: func(ctx context.Context, adapter CLIAdapter, reader io.Reader, sink chan<- StreamEvent) (TerminalReport, error) {
//	        return dispatcher.NewMonitor(adapter, reader, sink, logger).Run(ctx)
//	    },
//	}
//
// All three fields MUST be non-nil; GenerateMessage returns
// ErrCommitAgentMisconfigured otherwise.
//
// Concurrency: a single CommitAgent value services concurrent GenerateMessage
// calls — the struct holds no mutable state, every spawn carries its own
// per-call inputs through the SpawnBuilder + MonitorRunner closures.
type CommitAgent struct {
	// GitDiff resolves `git diff <start>..<end>` for the action item under
	// commit. MUST be non-nil; ErrCommitAgentMisconfigured otherwise.
	GitDiff GitDiffReader

	// BuildSpawnCommand assembles the *exec.Cmd that runs the
	// commit-message-agent. Production wires dispatcher.BuildSpawnCommand
	// directly. MUST be non-nil; ErrCommitAgentMisconfigured otherwise.
	BuildSpawnCommand SpawnBuilder

	// Monitor consumes the spawn's stream.jsonl and returns the agent's
	// TerminalReport. Production wires (*Monitor).Run. MUST be non-nil;
	// ErrCommitAgentMisconfigured otherwise.
	Monitor MonitorRunner

	// runCmd executes the assembled *exec.Cmd. Defaults to (*exec.Cmd).Run
	// when nil. Tests inject to skip actual exec; production wiring leaves
	// this nil so the default is used. The unexported lowercase keeps the
	// struct's public surface limited to the three spec'd fields.
	runCmd func(*exec.Cmd) error

	// openStream opens the spawn's stream.jsonl for the Monitor to read.
	// Defaults to a thin os.Open wrapper when nil. Tests inject to feed a
	// pre-populated buffer through the Monitor without writing to disk.
	openStream func(path string) (io.ReadCloser, error)

	// lookupAdapterFn defaults to the in-package lookupAdapter when nil.
	// Tests inject to substitute a MockAdapter without touching the
	// process-wide adapter registry.
	lookupAdapterFn func(CLIKind) (CLIAdapter, bool)
}

// GenerateMessage spawns the commit-message-agent and returns the produced
// commit message string. Algorithm:
//
//  1. Validate item.StartCommit + item.EndCommit are both non-empty;
//     ErrNoCommitDiff otherwise.
//  2. Validate dependency fields are wired; ErrCommitAgentMisconfigured
//     otherwise.
//  3. Resolve `git diff <start>..<end>` via GitDiff. An empty diff is NOT
//     an error — the agent decides what to do with it (the cmd-message
//     might still be valid, e.g. "chore: empty commit for tag").
//  4. Build a synthetic action item carrying the parent's commit anchors
//     plus Kind=KindCommit so BuildSpawnCommand resolves the
//     commit-message-agent binding.
//  5. Invoke BuildSpawnCommand → (*exec.Cmd, SpawnDescriptor).
//  6. Resolve the bundle root from the descriptor's MCPConfigPath
//     (canonical layout: <root>/plugin/.mcp.json → <root> = dirname×2).
//  7. Materialize the diff at <root>/context/git_diff.patch so the
//     downstream prompt template can reference a fixed location.
//  8. Resolve the CLIAdapter for the binding's CLIKind.
//  9. Run the spawn (cmd.Run / runCmd injection point).
//  10. Open the stream.jsonl and feed it through Monitor.
//  11. Validate the terminal-event Text is non-empty and ≤
//     CommitMessageMaxLen chars; ErrCommitMessageTooLong otherwise.
//  12. Return the trimmed message.
//
// Returns:
//
//   - (msg, nil) on the happy path.
//   - (zero, ErrNoCommitDiff) when commit anchors are missing.
//   - (zero, ErrCommitAgentMisconfigured) when a CommitAgent dep is nil.
//   - (zero, wrapped err) when GitDiff / BuildSpawnCommand / runCmd /
//     openStream / Monitor surface an error.
//   - (zero, ErrCommitSpawnNoTerminal) when the spawn ran cleanly but
//     produced no terminal text.
//   - (zero, ErrCommitMessageTooLong) when the agent's text exceeds the
//     length cap.
func (c *CommitAgent) GenerateMessage(
	ctx context.Context,
	item domain.ActionItem,
	project domain.Project,
	catalog templates.KindCatalog,
	auth AuthBundle,
) (string, error) {
	if c == nil {
		return "", fmt.Errorf("%w: nil CommitAgent receiver", ErrCommitAgentMisconfigured)
	}
	if strings.TrimSpace(item.StartCommit) == "" || strings.TrimSpace(item.EndCommit) == "" {
		return "", fmt.Errorf(
			"%w (start=%q end=%q action_item=%q)",
			ErrNoCommitDiff, item.StartCommit, item.EndCommit, item.ID,
		)
	}
	if c.GitDiff == nil {
		return "", fmt.Errorf("%w: GitDiff", ErrCommitAgentMisconfigured)
	}
	if c.BuildSpawnCommand == nil {
		return "", fmt.Errorf("%w: BuildSpawnCommand", ErrCommitAgentMisconfigured)
	}
	if c.Monitor == nil {
		return "", fmt.Errorf("%w: Monitor", ErrCommitAgentMisconfigured)
	}

	// Step 3: resolve diff. Empty bytes is NOT an error — pass through and
	// let the agent decide. A real git failure (worktree missing, bad
	// commit hash) propagates as a wrapped error.
	diff, err := c.GitDiff.Diff(ctx, item.StartCommit, item.EndCommit)
	if err != nil {
		return "", fmt.Errorf("dispatcher: commit-agent git diff: %w", err)
	}

	// Step 4: synthesize a commit-kind action item carrying the parent's
	// commit anchors. Title preserves the parent ID for log forensics.
	syntheticItem := domain.ActionItem{
		ID:          item.ID + "-commit-msg",
		ProjectID:   item.ProjectID,
		ParentID:    item.ID,
		Kind:        domain.KindCommit,
		StartCommit: item.StartCommit,
		EndCommit:   item.EndCommit,
		Title:       fmt.Sprintf("commit-message for %s", item.ID),
		// Paths/Packages intentionally empty: the commit-message-agent does
		// not edit code, so it has no write scope.
	}

	cmd, descriptor, err := c.BuildSpawnCommand(syntheticItem, project, catalog, auth)
	if err != nil {
		return "", fmt.Errorf("dispatcher: commit-agent build spawn: %w", err)
	}
	if cmd == nil {
		return "", fmt.Errorf("dispatcher: commit-agent build spawn returned nil cmd")
	}

	// Step 6: locate bundle root from descriptor.MCPConfigPath (canonical
	// shape: <root>/plugin/.mcp.json). The walk-up couples F.7.12 to the
	// claude bundle layout — when codex / future adapters publish their
	// own MCPConfigPath layout, the bundle-root recovery moves onto the
	// SpawnDescriptor itself (a Bundle field is the obvious shape).
	if strings.TrimSpace(descriptor.MCPConfigPath) == "" {
		return "", fmt.Errorf("dispatcher: commit-agent spawn descriptor missing MCPConfigPath")
	}
	bundleRoot := filepath.Dir(filepath.Dir(descriptor.MCPConfigPath))
	if strings.TrimSpace(bundleRoot) == "" {
		return "", fmt.Errorf("dispatcher: commit-agent could not derive bundle root from MCPConfigPath %q", descriptor.MCPConfigPath)
	}

	// Step 7: write the diff to <bundleRoot>/context/git_diff.patch. The
	// context/ subdir is materialized by F.7.1's NewBundle (it is the
	// canonical staging directory for per-spawn context attachments).
	contextDir := filepath.Join(bundleRoot, "context")
	if err := os.MkdirAll(contextDir, 0o700); err != nil {
		return "", fmt.Errorf("dispatcher: commit-agent ensure context dir: %w", err)
	}
	diffPath := filepath.Join(contextDir, commitDiffPatchFilename)
	if err := os.WriteFile(diffPath, diff, 0o600); err != nil {
		return "", fmt.Errorf("dispatcher: commit-agent write git diff: %w", err)
	}

	// Step 8: resolve adapter for monitor parsing. We re-resolve the
	// binding here rather than threading the adapter through the
	// SpawnBuilder return because BuildSpawnCommand's contract today
	// returns *exec.Cmd + SpawnDescriptor, not the adapter handle. The
	// re-resolution mirrors the in-spawn lookup path verbatim.
	rawBinding, ok := catalog.LookupAgentBinding(syntheticItem.Kind)
	if !ok {
		return "", fmt.Errorf("dispatcher: commit-agent no binding for kind %q", syntheticItem.Kind)
	}
	resolved := ResolveBinding(rawBinding)
	lookupFn := c.lookupAdapterFn
	if lookupFn == nil {
		lookupFn = lookupAdapter
	}
	adapter, ok := lookupFn(resolved.CLIKind)
	if !ok {
		return "", fmt.Errorf("%w: %q", ErrUnsupportedCLIKind, resolved.CLIKind)
	}

	// Step 9: run the spawn. Default to cmd.Run; tests inject a no-op or
	// failure injector through runCmd.
	runner := c.runCmd
	if runner == nil {
		runner = func(c *exec.Cmd) error { return c.Run() }
	}
	if err := runner(cmd); err != nil {
		return "", fmt.Errorf("dispatcher: commit-agent run spawn: %w", err)
	}

	// Step 10: open the stream.jsonl. Canonical location:
	// <bundleRoot>/stream.jsonl per F.7.1 NewBundle.
	streamPath := filepath.Join(bundleRoot, "stream.jsonl")
	open := c.openStream
	if open == nil {
		open = func(p string) (io.ReadCloser, error) { return os.Open(p) }
	}
	streamReader, err := open(streamPath)
	if err != nil {
		return "", fmt.Errorf("dispatcher: commit-agent open stream %q: %w", streamPath, err)
	}
	defer func() { _ = streamReader.Close() }()

	// The sink channel captures every parsed StreamEvent so we can extract
	// the LAST assistant-text event before IsTerminal fires. The claude
	// adapter today routes commit-message-agent's actual response through
	// non-terminal "assistant" events; the terminal "result" event carries
	// only telemetry (cost, denials, terminal_reason). Without the sink
	// the commit message would be lost. The buffer size (256) is large
	// enough to absorb a typical commit-agent run (handful of assistant
	// events + a few tool_use steps) without backpressure tipping the
	// Monitor into its dropped-events fallback.
	sink := make(chan StreamEvent, 256)
	type sinkResult struct {
		lastAssistant string
	}
	sinkDone := make(chan sinkResult, 1)
	go func() {
		var last string
		for ev := range sink {
			// Capture every non-terminal assistant text. Last write wins —
			// the agent's final assistant message before terminal IS the
			// message we want. Trim leading/trailing whitespace so a
			// stylistically formatted message round-trips clean.
			if ev.Type == "assistant" && ev.Text != "" {
				last = strings.TrimSpace(ev.Text)
			}
		}
		sinkDone <- sinkResult{lastAssistant: last}
	}()

	report, err := c.Monitor(ctx, adapter, streamReader, sink)
	close(sink)
	result := <-sinkDone
	if err != nil {
		return "", fmt.Errorf("dispatcher: commit-agent monitor stream: %w", err)
	}

	// Step 11: extract message. Priority order:
	//
	//  1. The last non-terminal assistant text (production claude path —
	//     the commit-message-agent's actual response lands here).
	//  2. report.Reason (TestMockAdapter / unit-test path — the fixture
	//     routes the message through the terminal report's Reason field).
	//  3. report.Errors[0] (defensive fallback for adapters that route
	//     short completions through the errors channel).
	//
	// When all three are empty, surface ErrCommitSpawnNoTerminal — the
	// spawn ran cleanly but produced nothing usable.
	message := result.lastAssistant
	if message == "" {
		message = strings.TrimSpace(report.Reason)
	}
	if message == "" && len(report.Errors) > 0 {
		message = strings.TrimSpace(report.Errors[0])
	}
	if message == "" {
		return "", fmt.Errorf("%w (action_item=%q)", ErrCommitSpawnNoTerminal, item.ID)
	}

	// Step 11 (cont'd): enforce the length cap. Multi-line input is
	// rejected as overlong even when the first line is ≤ cap — the
	// "Single-Line Commits" rule prohibits a body, not just a long
	// subject. Strip a single trailing newline before the length check
	// so an agent that helpfully appended "\n" at the end does not trip
	// the cap by one character.
	if strings.ContainsRune(message, '\n') || len(message) > CommitMessageMaxLen {
		return "", fmt.Errorf("%w: %q (len=%d, cap=%d)", ErrCommitMessageTooLong, message, len(message), CommitMessageMaxLen)
	}

	return message, nil
}
