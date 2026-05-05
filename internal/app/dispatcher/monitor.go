package dispatcher

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/evanmschultz/tillsyn/internal/domain"
)

// Wave 2.8 process-monitor contract overview.
//
// processMonitor consumes the *exec.Cmd produced by BuildSpawnCommand (4a.19)
// and is responsible for two things:
//
//  1. Starting the subprocess and tracking its lifetime, so callers can wait
//     on a single Handle to receive the final TerminationOutcome.
//  2. Detecting agent crashes (non-zero exit OR signal termination) and
//     transitioning the action item to StateFailed via Service.MoveActionItem,
//     while populating outcome metadata via Service.UpdateActionItem.
//
// Clean-exit semantics: if the agent exits 0 the monitor takes NO action on
// the action item — the agent is responsible for moving its own state to
// StateComplete from inside its run. The monitor only intervenes on crash.
//
// State-conflict guard (acceptance §5): before applying the failed
// transition the monitor refetches the action item via Service.GetActionItem
// and inspects its current LifecycleState. If the item is already in
// StateComplete (the agent succeeded and updated its own state before the
// process exit was observed), the monitor logs the conflict and skips the
// move + update calls. The downgrade is rejected by the service guard at
// internal/app/service.go:1003 anyway (transitions FROM terminal states are
// blocked); the in-monitor check is a clean signal for the test suite and a
// no-side-effect short-circuit so the wave does not depend on
// ErrTransitionBlocked surfacing for routine race resolution.
//
// Concurrency contract (acceptance §6): a single processMonitor instance
// services concurrent Track calls from multiple goroutines. Per-Handle state
// (the *exec.Cmd, the Wait result, the Close signal) is owned by that Handle
// alone; the monitor's own mu guards only the tracked map of action-item ID
// → Handle pointer used by Drop 4b's continuous-mode dashboard. Each Handle
// runs exactly one cmd.Wait() goroutine; Wait() and Close() are both
// idempotent and goroutine-leak-free — once the cmd.Wait() goroutine exits,
// no further goroutine survives the Handle.
//
// Test-helper carve-out (acceptance §7): the test suite compiles a
// throwaway agent binary from testdata/fakeagent.go via exec.Command("go",
// "build", ...) so the monitor can exercise real process semantics
// (exit-codes, signal kills, durations) without depending on the
// claude binary being on PATH. This is the one documented exception to the
// project's "never raw `go`" rule: see WAVE_2_PLAN.md §2.8 Q5 and
// monitor_test.go's package doc-comment for the rationale. Production code
// in this file does NOT shell out to `go`.

// monitorService is the narrow consumer-side view the process monitor uses
// to refetch action-item state and apply crash transitions. *app.Service
// satisfies this interface; the test suite injects a deterministic stub so
// monitor scenarios run without standing up a full *app.Service graph.
//
// Method names mirror Service exactly so the production binding is a trivial
// assignment in the dispatcher constructor wired in 4a.23.
type monitorService interface {
	GetActionItem(ctx context.Context, actionItemID string) (domain.ActionItem, error)
	ListColumns(ctx context.Context, projectID string, includeArchived bool) ([]domain.Column, error)
	MoveActionItem(ctx context.Context, actionItemID, toColumnID string, position int) (domain.ActionItem, error)
	UpdateActionItem(ctx context.Context, in updateActionItemInput) (domain.ActionItem, error)
}

// updateActionItemInput is the local alias for app.UpdateActionItemInput.
// The dispatcher package must avoid an import cycle with internal/app once
// 4a.23 wires *app.Service into the dispatcher constructor, so the narrow
// monitor interface uses a forward-declared shape that the *app.Service
// adapter (a one-line binding in 4a.23) supplies. The fields here are the
// exact subset the monitor populates; see internal/app/service.go for the
// full UpdateActionItemInput.
type updateActionItemInput struct {
	ActionItemID string
	Metadata     *domain.ActionItemMetadata
}

// TerminationOutcome captures the per-process result the monitor surfaces
// from a single Handle.Wait. Field semantics:
//
//   - ExitCode is the process exit code on a clean exit (0..255). On a
//     signal-driven termination ExitCode is -1, mirroring
//     os.ProcessState.ExitCode.
//   - Signal is the human-readable signal name on a signal-driven
//     termination (e.g. "killed", "terminated"); empty otherwise. The string
//     is sourced from syscall.WaitStatus.Signal().String() on Unix; on
//     platforms where the Sys() cast does not yield WaitStatus, Signal
//     falls back to a parsed prefix of ProcessState.String() so the test
//     suite stays portable.
//   - Crashed is true iff the process did NOT exit cleanly (any non-zero
//     exit OR a signal). Equivalent to !ProcessState.Success().
//   - Duration is the wall-clock time between Track starting the process
//     and Wait observing its termination, sampled from the monitor's clock
//     (time.Now in production; injectable in tests).
type TerminationOutcome struct {
	ExitCode int
	Signal   string
	Crashed  bool
	Duration time.Duration
}

// ErrMonitorNotStarted is returned by Track when the supplied *exec.Cmd has
// already been started or has nil Process — both indicate misuse since the
// monitor is the sole owner of cmd.Start lifecycle.
var ErrMonitorNotStarted = errors.New("dispatcher: monitor failed to start process")

// ErrMonitorInvalidInput is returned by Track when actionItemID is empty or
// cmd is nil. Callers detect via errors.Is.
var ErrMonitorInvalidInput = errors.New("dispatcher: invalid monitor input")

// Handle is the per-process tracking record returned by Track. The owning
// goroutine runs cmd.Wait inside the monitor and signals completion via
// done. Wait blocks on done and returns the cached outcome; Close requests
// the process die (best-effort cmd.Process.Kill) and returns once the
// goroutine has reaped.
//
// Wait/Close are both safe to call from multiple goroutines; sync.Once
// guarantees the cmd.Wait result is computed exactly once. Close after Wait
// is a no-op; Wait after Close returns the post-kill outcome.
type Handle struct {
	actionItemID string
	cmd          *exec.Cmd
	startedAt    time.Time

	// done is closed by the monitor goroutine after the cmd.Wait result has
	// been cached into outcome+waitErr. Wait() blocks on this channel.
	done chan struct{}

	// closeOnce guards the kill-on-Close path so concurrent Close calls do
	// not double-kill the process or panic on Process == nil.
	closeOnce sync.Once

	// resultMu guards outcome and waitErr so Wait observers see a coherent
	// snapshot regardless of whether they read before or after the goroutine
	// closes done.
	resultMu sync.Mutex
	outcome  TerminationOutcome
	waitErr  error
}

// Wait blocks until the underlying process terminates (or until Close
// observes the kill propagating) and returns the cached TerminationOutcome.
// Subsequent calls return the same value; the call is goroutine-leak-free
// because exactly one cmd.Wait goroutine ever runs per Handle.
//
// The returned error is the wrapped Service mutation error if the
// monitor's crash-handling pipeline (state-refetch → MoveActionItem →
// UpdateActionItem) failed. Process-level outcomes (non-zero exit, signal
// kill) are NOT returned as errors — they appear in the TerminationOutcome
// fields. The error return is reserved for service-side failures so callers
// can distinguish "agent crashed (expected, action-item updated)" from
// "agent crashed AND we could not record the failure".
func (h *Handle) Wait() (TerminationOutcome, error) {
	if h == nil {
		return TerminationOutcome{}, fmt.Errorf("%w: nil handle", ErrMonitorInvalidInput)
	}
	<-h.done
	h.resultMu.Lock()
	defer h.resultMu.Unlock()
	return h.outcome, h.waitErr
}

// Close requests the underlying process exit and waits for the monitor
// goroutine to reap. Close is safe to call concurrently with Wait, before
// Wait, after Wait, or multiple times — sync.Once + the done channel
// linearize the teardown.
//
// The kill is best-effort: if cmd.Process is nil (Start never succeeded),
// or if the process has already exited by the time Close is invoked, Kill
// returns an error which is intentionally swallowed — the Handle is
// already on its way to terminated.
func (h *Handle) Close() {
	if h == nil {
		return
	}
	h.closeOnce.Do(func() {
		if h.cmd != nil && h.cmd.Process != nil {
			// Best-effort kill; ignore the error because the process may
			// already have exited. The monitor goroutine still observes the
			// termination and reports it via outcome.
			_ = h.cmd.Process.Kill()
		}
	})
	<-h.done
}

// processMonitor is the in-process subprocess tracker described above. It
// holds a small mutex-guarded map of in-flight Handles purely so Drop 4b's
// continuous-mode dashboard can introspect live agents; today the map is
// produce-only (Track inserts, the goroutine deletes on termination).
type processMonitor struct {
	svc   monitorService
	clock func() time.Time

	mu      sync.Mutex
	tracked map[string]*Handle
}

// newProcessMonitor constructs a processMonitor bound to svc. svc MUST be
// non-nil; callers wire the production *app.Service via the dispatcher
// constructor (deferred to 4a.23). The test suite passes a stub
// monitorService directly through this constructor.
//
// clock defaults to time.Now when nil — tests inject a deterministic clock
// to assert Duration math.
func newProcessMonitor(svc monitorService, clock func() time.Time) *processMonitor {
	if clock == nil {
		clock = time.Now
	}
	return &processMonitor{
		svc:     svc,
		clock:   clock,
		tracked: make(map[string]*Handle),
	}
}

// Track starts cmd and returns a Handle that callers Wait on. The monitor
// owns the cmd's lifecycle from this point forward — callers MUST NOT call
// cmd.Start, cmd.Wait, or cmd.Process.Kill themselves. Use Handle.Close to
// terminate.
//
// Track returns ErrMonitorInvalidInput on empty actionItemID or nil cmd.
// Returns ErrMonitorNotStarted wrapped with the underlying os/exec error
// when cmd.Start fails.
func (m *processMonitor) Track(ctx context.Context, actionItemID string, cmd *exec.Cmd) (*Handle, error) {
	if m == nil || m.svc == nil {
		return nil, fmt.Errorf("%w: process monitor service is nil", ErrInvalidDispatcherConfig)
	}
	trimmed := strings.TrimSpace(actionItemID)
	if trimmed == "" {
		return nil, fmt.Errorf("%w: action item ID is empty", ErrMonitorInvalidInput)
	}
	if cmd == nil {
		return nil, fmt.Errorf("%w: cmd is nil", ErrMonitorInvalidInput)
	}

	startedAt := m.clock()
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrMonitorNotStarted, err)
	}

	h := &Handle{
		actionItemID: trimmed,
		cmd:          cmd,
		startedAt:    startedAt,
		done:         make(chan struct{}),
	}

	m.mu.Lock()
	m.tracked[trimmed] = h
	m.mu.Unlock()

	go m.runHandle(ctx, h)
	return h, nil
}

// Unsubscribe removes the supplied actionItemID from the tracked-PID map. It
// is the production-side seam consumed by cleanupHook (cleanup.go) when an
// action item enters a terminal lifecycle state — by the time cleanup runs
// the per-Handle goroutine has already deleted its own entry on exit, so
// this method is a defensive scrub: an idempotent no-op when the entry is
// already absent. The signature mirrors the cleanup.monitorUnsubscriber
// interface and is fire-and-forget — the tracked map is a dashboard-facing
// convenience, not a correctness invariant. Wired in droplet 4a.23 as part
// of the dispatcher constructor's cleanup-hook plumbing.
func (m *processMonitor) Unsubscribe(actionItemID string) {
	if m == nil {
		return
	}
	trimmed := strings.TrimSpace(actionItemID)
	if trimmed == "" {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.tracked, trimmed)
}

// runHandle is the per-Handle goroutine that waits on cmd, builds the
// TerminationOutcome, and (on crash) drives the action-item state
// transition. Exactly one goroutine ever runs per Handle; it removes the
// Handle from the tracked map on exit and closes h.done.
func (m *processMonitor) runHandle(ctx context.Context, h *Handle) {
	defer func() {
		m.mu.Lock()
		delete(m.tracked, h.actionItemID)
		m.mu.Unlock()
		close(h.done)
	}()

	waitErr := h.cmd.Wait()
	endedAt := m.clock()

	outcome := buildOutcome(h.cmd, waitErr, endedAt.Sub(h.startedAt))

	h.resultMu.Lock()
	h.outcome = outcome
	h.resultMu.Unlock()

	if !outcome.Crashed {
		// Clean exit: the agent owns its own terminal-state transition.
		// The monitor takes no action — its job is crash detection, not
		// success recording.
		return
	}

	if err := m.applyCrashTransition(ctx, h.actionItemID, outcome); err != nil {
		h.resultMu.Lock()
		h.waitErr = err
		h.resultMu.Unlock()
	}
}

// applyCrashTransition is the crash-handling pipeline: refetch the action
// item, short-circuit if it is already complete (state-conflict guard),
// otherwise resolve the failed-column ID, move the action item to failed,
// and update its metadata with outcome + reason.
func (m *processMonitor) applyCrashTransition(ctx context.Context, actionItemID string, outcome TerminationOutcome) error {
	current, err := m.svc.GetActionItem(ctx, actionItemID)
	if err != nil {
		return fmt.Errorf("monitor: refetch action item %q: %w", actionItemID, err)
	}
	// State-conflict guard: a complete action item must NOT be downgraded.
	// The agent self-updated its terminal state before the process exit
	// was observed; the monitor's transition would either be a silent
	// downgrade (semantically wrong) or a service-rejected transition
	// (line noise). Skip both.
	if current.LifecycleState == domain.StateComplete {
		return nil
	}

	columns, err := m.svc.ListColumns(ctx, current.ProjectID, true)
	if err != nil {
		return fmt.Errorf("monitor: list columns for project %q: %w", current.ProjectID, err)
	}
	failedColumnID := columnIDForLifecycleState(columns, domain.StateFailed)
	if failedColumnID == "" {
		return fmt.Errorf("monitor: project %q has no failed column", current.ProjectID)
	}

	if _, err := m.svc.MoveActionItem(ctx, current.ID, failedColumnID, current.Position); err != nil {
		return fmt.Errorf("monitor: move action item %q to failed: %w", current.ID, err)
	}

	// Populate outcome metadata. ActionItemMetadata has no FailureReason
	// field today — Drop 4b refactors the failure shape into a structured
	// type per PLAN.md §17.3.Q5; until then we use the existing free-form
	// BlockedReason slot as the carrier (the only free-form failure-context
	// string on the closed metadata struct). The "agent process crashed:"
	// prefix is load-bearing for the test suite so future regressions on
	// the carrier-field choice fail loudly.
	updated := current.Metadata
	updated.Outcome = "failure"
	updated.BlockedReason = formatFailureReason(outcome)

	if _, err := m.svc.UpdateActionItem(ctx, updateActionItemInput{
		ActionItemID: current.ID,
		Metadata:     &updated,
	}); err != nil {
		return fmt.Errorf("monitor: update action item %q metadata: %w", current.ID, err)
	}
	return nil
}

// buildOutcome consumes a finished *exec.Cmd and constructs the
// TerminationOutcome. waitErr is the error returned by cmd.Wait (typically
// nil for clean exit, *exec.ExitError for non-zero exit, signal-killed, or
// other os/exec failure).
func buildOutcome(cmd *exec.Cmd, waitErr error, duration time.Duration) TerminationOutcome {
	out := TerminationOutcome{
		Duration: duration,
	}
	state := cmd.ProcessState
	if state == nil {
		// cmd.Wait returned an error before producing a ProcessState (rare
		// — typically only when the process never started, but we can hit
		// it on file-descriptor exhaustion). Treat as a crash with -1
		// exit; surface waitErr context via Signal so the test/operator
		// has a hint.
		out.ExitCode = -1
		out.Crashed = true
		if waitErr != nil {
			out.Signal = "wait_error: " + waitErr.Error()
		} else {
			out.Signal = "wait_error: process state unavailable"
		}
		return out
	}
	out.ExitCode = state.ExitCode()
	out.Crashed = !state.Success()
	if out.Crashed {
		out.Signal = signalNameFromState(state)
	}
	return out
}

// signalNameFromState extracts a human-readable signal name from a
// finished os.ProcessState. On Unix the Sys() value is a syscall.WaitStatus
// from which Signal() yields the os.Signal; we stringify that. On
// non-Unix platforms (or when the cast fails for any reason) the function
// falls back to parsing ProcessState.String() so the value is still useful
// to the test suite and the dev reading the action-item metadata.
//
// Returns the empty string when ExitCode() is non-negative (clean exit or
// non-zero exit without a signal); callers always check Crashed before
// reading Signal.
func signalNameFromState(state interface {
	ExitCode() int
	String() string
	Sys() any
},
) string {
	if state == nil || state.ExitCode() >= 0 {
		return ""
	}
	if ws, ok := state.Sys().(syscall.WaitStatus); ok {
		if ws.Signaled() {
			return ws.Signal().String()
		}
	}
	// Fallback: parse "signal: killed" / "signal: terminated" out of the
	// platform's ProcessState.String() shape. Strip the leading "signal:"
	// prefix when present so the metadata field stores just the name.
	s := state.String()
	if rest, ok := strings.CutPrefix(s, "signal: "); ok {
		return strings.TrimSpace(rest)
	}
	return s
}

// formatFailureReason renders the BlockedReason value the monitor writes
// into action-item metadata on a crash transition. The shape is the
// load-bearing prefix "agent process crashed:" followed by the most
// specific signal-or-exit-code descriptor available — tests pin both the
// prefix and the suffix.
func formatFailureReason(outcome TerminationOutcome) string {
	if outcome.Signal != "" {
		return "agent process crashed: signal: " + outcome.Signal
	}
	return fmt.Sprintf("agent process crashed: exit code %d", outcome.ExitCode)
}

// =============================================================================
// Stream-JSON Monitor (F.7-CORE F.7.4)
// =============================================================================
//
// The Monitor type below is the CLI-agnostic stream-JSON consumer the
// dispatcher uses to read a spawn's <bundle>/stream.jsonl line-by-line and
// surface terminal reports + tool-denial events to upstream consumers (the
// F.7.5b TUI handshake, the spawn descriptor's actual_cost_usd update, and the
// post-spawn outcome pipeline).
//
// CLI-AGNOSTIC RULE (F.7-CORE F.7.4 + master PLAN.md L11): the Monitor source
// MUST NOT contain CLI-specific wire-format event-type literals — no claude
// init-shortcut strings, no terminal-event string-match against the wire
// format, no MockAdapter terminal-token literals baked into the routing.
// Every event-family decision is made by the injected adapter via
// StreamEvent.Type + IsTerminal; the Monitor itself only knows the canonical
// shape declared in cli_adapter.go.
//
// Two-axis decoupling (REV-7 supersession of F.7.17.9):
//
//  1. CLIAdapter abstracts the wire format. The Monitor does not know which
//     adapter-private terminal-event token its CLI emits; only the canonical
//     IsTerminal flag matters.
//  2. The sink channel + logger callback abstract the destination. The
//     Monitor does not know whether the consumer is the TUI permission-denial
//     dialog, the dispatcher's metadata.actual_cost_usd writer, or a forensic
//     tee.
//
// Algorithm (per droplet 4c.F.7.4 acceptance):
//
//   - Wrap reader in bufio.Scanner with a 1 MiB max-token to absorb large
//     assistant-event blobs (claude can emit single events past the default
//     64 KiB limit).
//   - For each scanned line: skip empty / whitespace-only lines silently;
//     decode via adapter.ParseStreamEvent. Decode errors log a warning and
//     continue — claude streams may emit interleaved progress lines the
//     canonical taxonomy doesn't yet cover, and forward-compat trumps
//     halt-on-malformed.
//   - Forward every successfully-parsed event to the optional sink channel
//     via a non-blocking send (select default) so a slow consumer cannot
//     deadlock the reader. Dropped events are counted via a debug log line.
//   - On IsTerminal == true: extract the TerminalReport via the adapter and
//     remember the most recent one. Continue reading until EOF — the
//     terminal event SHOULD be the last line in a well-formed stream, but
//     the Monitor is defensive against trailing noise (multiple terminal
//     events, post-result heartbeats) and returns the LAST seen report.
//   - On ctx.Done(): return (zero TerminalReport, ctx.Err()).
//   - On EOF: return (last seen TerminalReport, nil). When no terminal
//     event was seen, the returned TerminalReport is the zero value and the
//     error is nil — the caller distinguishes "spawn ended without a
//     terminal report" from "spawn was cut off by ctx" via the error.

// MonitorLogger is the narrow logging seam the stream-JSON Monitor uses to
// surface malformed-line warnings and slow-sink drops. Production callers
// inject a charmbracelet/log adapter at the dispatcher boundary; the Monitor
// itself depends only on the Printf-shaped interface so the dispatcher's
// logger choice stays at the call site, not in this file.
//
// A nil MonitorLogger is treated as "discard" — see Monitor.log for the
// guard. Tests pass a slice-capturing logger so log-line assertions stay
// deterministic.
type MonitorLogger interface {
	Printf(format string, args ...any)
}

// Monitor reads a spawn's stream.jsonl and dispatches every parsed event
// through the supplied CLIAdapter, forwarding canonical StreamEvent values to
// an optional sink channel and remembering the last seen terminal report for
// the Run return value. The Monitor is single-shot — call Run exactly once
// per Monitor; subsequent Run calls on a Monitor whose reader has already hit
// EOF return the zero TerminalReport + nil immediately.
//
// CLI-agnostic by construction: every event-family decision goes through the
// adapter. The Monitor source contains no CLI-specific event-type literals.
type Monitor struct {
	adapter CLIAdapter
	reader  io.Reader
	sink    chan<- StreamEvent
	logger  MonitorLogger
}

// NewMonitor constructs a Monitor bound to adapter + reader. sink is optional
// — pass nil to suppress event forwarding (the Run terminal-report return
// value still works). logger is optional — pass nil to discard the warning
// stream.
//
// adapter and reader MUST be non-nil; Run returns ErrInvalidMonitorConfig
// when either is nil so misuse fails loudly.
func NewMonitor(adapter CLIAdapter, reader io.Reader, sink chan<- StreamEvent, logger MonitorLogger) *Monitor {
	return &Monitor{
		adapter: adapter,
		reader:  reader,
		sink:    sink,
		logger:  logger,
	}
}

// ErrInvalidMonitorConfig is returned by Run when the Monitor was constructed
// with a nil adapter or nil reader. Callers detect via errors.Is.
var ErrInvalidMonitorConfig = errors.New("dispatcher: invalid stream monitor config")

// monitorScannerMaxBytes caps the bufio.Scanner per-line buffer at 1 MiB.
// claude's --output-format stream-json can emit single assistant events
// larger than the default 64 KiB scanner buffer (long thinking blocks, large
// tool inputs, multi-paragraph text). The 1 MiB ceiling is generous enough
// for every recorded fixture in repo and small enough that a runaway stream
// cannot exhaust process memory before the adapter's parser surfaces the
// problem.
const monitorScannerMaxBytes = 1 << 20

// Run reads from the configured reader until EOF or context cancellation.
// Returns the last-seen TerminalReport on clean EOF (zero value if no
// terminal event was observed) plus a nil error. On context cancellation
// returns (zero TerminalReport, ctx.Err()). On reader error returns (zero
// TerminalReport, wrapped error).
//
// Concurrency: Run is intended to be called from a single goroutine per
// Monitor. The optional sink channel is sent to via a non-blocking select
// (with a debug-log fallback) so a slow consumer does not deadlock the
// reader.
func (m *Monitor) Run(ctx context.Context) (TerminalReport, error) {
	if m == nil {
		return TerminalReport{}, fmt.Errorf("%w: nil monitor", ErrInvalidMonitorConfig)
	}
	if m.adapter == nil {
		return TerminalReport{}, fmt.Errorf("%w: nil adapter", ErrInvalidMonitorConfig)
	}
	if m.reader == nil {
		return TerminalReport{}, fmt.Errorf("%w: nil reader", ErrInvalidMonitorConfig)
	}

	scanner := bufio.NewScanner(m.reader)
	// Replace the default 64 KiB buffer with one capped at 1 MiB so claude's
	// largest documented event payloads round-trip without truncation.
	scanner.Buffer(make([]byte, 0, 64*1024), monitorScannerMaxBytes)

	var (
		lastReport    TerminalReport
		seenTerminal  bool
		droppedEvents int
	)

	for scanner.Scan() {
		// Honor cancellation between iterations — Scan itself does not
		// observe the context, but checking before each parse keeps Run
		// responsive at line granularity.
		select {
		case <-ctx.Done():
			return TerminalReport{}, ctx.Err()
		default:
		}

		line := scanner.Bytes()
		// Skip empty / whitespace-only lines silently — many stream
		// producers emit a trailing newline after the terminal event and
		// some CLIs blank-pad between events.
		if len(strings.TrimSpace(string(line))) == 0 {
			continue
		}

		// Copy line so the StreamEvent.Raw retained inside the adapter
		// detaches from the scanner's reusable buffer. Adapters that
		// already copy (claude, mock) absorb the extra alloc; adapters
		// that don't are protected from a buffer-aliasing bug.
		buf := make([]byte, len(line))
		copy(buf, line)

		event, parseErr := m.adapter.ParseStreamEvent(buf)
		if parseErr != nil {
			m.log("monitor: skip malformed stream line: %v", parseErr)
			continue
		}

		// Forward to sink (non-blocking) BEFORE extracting terminal
		// report — downstream consumers may want to observe terminal
		// events too (for the TUI's permission-denial handshake the
		// terminal IS the point).
		if m.sink != nil {
			select {
			case m.sink <- event:
			default:
				droppedEvents++
				m.log("monitor: sink full, dropped event type=%q (total dropped=%d)", event.Type, droppedEvents)
			}
		}

		if event.IsTerminal {
			report, ok := m.adapter.ExtractTerminalReport(event)
			if ok {
				lastReport = report
				seenTerminal = true
			}
		}
	}

	if err := scanner.Err(); err != nil {
		// Distinguish reader error from clean EOF. ErrInvalidMonitorConfig
		// is reserved for setup errors; reader errors get a fresh wrap so
		// callers can errors.Is the underlying io error.
		return TerminalReport{}, fmt.Errorf("monitor: read stream: %w", err)
	}

	// Final cancellation check — if ctx was cancelled exactly as the
	// reader hit EOF, prefer the cancellation signal so the caller sees a
	// non-nil error.
	if err := ctx.Err(); err != nil {
		return TerminalReport{}, err
	}

	if !seenTerminal {
		return TerminalReport{}, nil
	}
	return lastReport, nil
}

// log writes a formatted line to the configured logger when one is set.
// A nil logger silently discards — production callers always inject one,
// tests inject a capturing fake when log assertions matter.
func (m *Monitor) log(format string, args ...any) {
	if m == nil || m.logger == nil {
		return
	}
	m.logger.Printf(format, args...)
}
