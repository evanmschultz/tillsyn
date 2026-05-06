package dispatcher

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/evanmschultz/tillsyn/internal/domain"
)

// orphan_scan.go ships the dispatcher startup orphan-scan API per Drop 4c
// F.7-CORE F.7.8. On Tillsyn process restart any action item that was sitting
// in StateInProgress when the prior dispatcher died is potentially orphaned —
// the spawn's monitoring goroutine died with the process, and no observer is
// driving the action item to its terminal state. The orphan scan walks every
// in_progress action item, reads <bundle>/manifest.json (per F.7.1's
// ManifestMetadata), and uses the recorded ClaudePID to distinguish three
// states per the spawn architecture memory §8 ("Crash Recovery Model"):
//
//  1. PID == 0 — spawn was created but never started; the dispatcher's normal
//     monitoring goroutine will pick it up on the next promotion pass. Skip.
//  2. PID alive — a process with the recorded PID is currently running. The
//     spawn is healthy under a different (newly-restarted) dispatcher cycle
//     OR was never orphaned. Skip; the regular monitor handles it.
//  3. PID dead — the recorded process is gone. The action item is orphaned
//     and the OnOrphanFound callback fires so the caller can move the item
//     to StateFailed and clean up the bundle.
//
// REV-5 (F7_CORE_PLAN.md): F.7.8 was BLOCKED on F.7.17.6 landing
// ManifestMetadata.CLIKind. F.7.17.6 shipped at commit cc2f3ee, which is the
// trigger for this droplet. The CLIKind value is plumbed through from the
// manifest into the OnOrphanFound callback context indirectly via the loaded
// ManifestMetadata; today every recorded manifest is "claude" so the scanner
// uses a uniform PID-liveness check, but the field is preserved on the
// ManifestMetadata payload so future codex bundles route adapter-specific
// liveness logic without a scanner-API change.
//
// SCOPE: this droplet ships the OrphanScanner API + the production
// ProcessChecker. It does NOT modify cmd/till/main.go or wire the scan into
// the dispatcher startup hook — that is deferred follow-up work owned by a
// later F.7-CORE droplet (or by Drop 5 service init). The scanner is fully
// testable today via the ProcessChecker injection seam.
//
// Round-2 cmdline guard (Attack 1 from F.7.8 QA-Falsification, accepted):
// signal-0 alone is insufficient to defeat PID reuse — between the prior
// dispatcher's death and the current scan the OS may reuse our recorded PID
// for an unrelated process (vim, ssh, a fresh shell). The plain signal-0
// probe would mistakenly report "still alive," leaving a real orphan
// unreaped indefinitely. ProcessChecker.IsAlive therefore takes a second
// argument — an expected-cmdline substring — and DefaultProcessChecker
// shells out to `ps -p <pid> -o comm=` to verify the live PID is actually a
// "claude" / "codex" binary (per spawn architecture memory §8 "Crash
// Recovery Model"). Scan derives the substring from manifest.CLIKind so the
// guard tightens automatically as new adapters land.
//
// Known limitations:
//
//   - Process-start-time tightening: a strict mitigation would compare a
//     manifest-recorded process-start-time against /proc/<pid>/stat (Linux)
//     or libproc proc_pidinfo (macOS). The cmdline-match guard catches the
//     overwhelmingly common case (PID reused for an unrelated binary
//     entirely) while a pathological start-up storm could still reuse a PID
//     for a fresh `claude` process within milliseconds. Acceptable for
//     today; recorded as a future refinement.
//   - Windows portability: the production ProcessChecker uses
//     proc.Signal(syscall.Signal(0)) AND `ps -p <pid> -o comm=`, both
//     POSIX-only. Pre-MVP Tillsyn does not target Windows; if cross-platform
//     support lands, the production checker grows a build-tagged Windows
//     variant (likely `tasklist /FI "PID eq <n>"` for the cmdline match).

// ErrInvalidOrphanScannerConfig is returned by OrphanScanner.Scan when the
// scanner was constructed with a nil ActionItems reader or a nil
// ProcessChecker. Callers detect via errors.Is.
var ErrInvalidOrphanScannerConfig = errors.New("dispatcher: invalid orphan scanner config")

// ActionItemReader is the narrow consumer-side view OrphanScanner needs to
// enumerate currently-in-progress action items. Implementations return every
// action item across every project whose LifecycleState is StateInProgress.
//
// The production binding is *app.Service (via a thin adapter that calls
// ListActionItems per project and filters in-process); test suites inject a
// stub directly to drive specific scenarios. The interface is defined HERE in
// the dispatcher package — not in internal/app — so the dispatcher does not
// depend on a higher-level service contract for this leaf concern.
type ActionItemReader interface {
	// ListInProgress returns every action item across every project whose
	// LifecycleState is StateInProgress at the time of the call. Order is
	// implementation-defined and not load-bearing for the scanner. The
	// returned slice is owned by the caller (callers may append / filter
	// without affecting the producer).
	ListInProgress(ctx context.Context) ([]domain.ActionItem, error)
}

// ProcessChecker reports whether a given OS process ID corresponds to a
// currently-running process whose binary identity matches an expected
// substring. The interface exists so OrphanScanner.Scan can be tested
// deterministically without spawning real processes; production callers
// wire DefaultProcessChecker.
type ProcessChecker interface {
	// IsAlive returns true if the supplied PID corresponds to a running
	// process AND the running process's binary name (cmdline / comm)
	// contains expectedCmdlineSubstring. The cmdline match defeats PID
	// reuse — a recycled PID held by an unrelated binary (vim, ssh, etc.)
	// must report not-alive so the orphan is reaped.
	//
	// Implementations MUST treat pid <= 0 as "not alive" (the scanner
	// short-circuits PID == 0 before calling IsAlive, but the contract is
	// defensive against future callers). When expectedCmdlineSubstring is
	// the empty string, implementations MUST fall back to plain liveness
	// (signal-0 / equivalent) without performing any cmdline comparison —
	// this preserves a callable seam for callers that legitimately do not
	// care which binary owns the PID (today: none; reserved for forensic
	// tooling).
	IsAlive(pid int, expectedCmdlineSubstring string) bool
}

// DefaultProcessChecker is the production ProcessChecker implementation. It
// uses os.FindProcess + proc.Signal(syscall.Signal(0)) for liveness AND a
// command-name lookup (default: `ps -p <pid> -o comm=`) for the
// cmdline-match guard, both POSIX-only. Per the file-level commentary,
// Windows portability is a future concern.
//
// Liveness probe (signal 0): on Unix os.FindProcess always succeeds (it does
// not actually verify the PID); the real liveness check is the signal-0
// send. signal-0 returns nil when the process exists and the caller has
// permission to signal it, syscall.ESRCH ("no such process") when the PID
// is dead, and syscall.EPERM when the process exists but the caller lacks
// permission. EPERM still means "alive" — DefaultProcessChecker treats it
// as alive to avoid mis-reaping a running spawn whose effective UID
// changed.
//
// Cmdline-match probe (default `ps -p <pid> -o comm=`): when
// expectedCmdlineSubstring is non-empty, IsAlive invokes CommLookup (or the
// package-level psCommLookup default) to fetch the process's command name
// (basename of argv[0] on macOS, equivalent to /proc/<pid>/comm on Linux)
// and reports alive only when the trimmed command name contains
// expectedCmdlineSubstring. The CommLookup seam exists so unit tests can
// replace the real `ps` invocation with a deterministic stub — under the
// race detector the actual `ps` shell-out has produced sandboxing-related
// hangs in CI, and an injection seam keeps the unit test path strictly
// in-process. Production callers leave CommLookup nil and the package
// default runs.
//
// Cmdline-match error semantics: the lookup function returns ("", err) when
// the PID has exited between signal-0 and the lookup (or any other error).
// IsAlive treats both error and empty-comm cases as "not alive" — reaping
// a since-exited process is a no-op.
//
// PATH note: ps is required on every supported POSIX system at /bin/ps and
// /usr/bin/ps; Go's exec.LookPath ("ps") consults the inherited PATH.
// Tillsyn's dispatcher inherits the dev's shell PATH so this is not a
// portability risk in practice.
type DefaultProcessChecker struct {
	// CommLookup, when non-nil, replaces the package-default `ps`-shellout
	// path used to fetch the live process's command name. Tests inject a
	// stub for deterministic behaviour; production callers leave this nil.
	// The function MUST return a non-empty trimmed command name on success
	// and ("", err) when the PID has exited or is otherwise unobservable.
	CommLookup func(pid int) (string, error)
}

// psCommLookup is the production CommLookup implementation. It shells out
// to `ps -p <pid> -o comm=` and returns the trimmed first line. Used when
// DefaultProcessChecker.CommLookup is nil.
func psCommLookup(pid int) (string, error) {
	out, err := exec.Command("ps", "-p", strconv.Itoa(pid), "-o", "comm=").Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// IsAlive implements ProcessChecker via signal-0 (liveness) plus a
// CommLookup-driven cmdline match. See the type-level doc-comment for the
// full algorithm.
func (d DefaultProcessChecker) IsAlive(pid int, expectedCmdlineSubstring string) bool {
	if pid <= 0 {
		return false
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	err = proc.Signal(syscall.Signal(0))
	switch {
	case err == nil:
		// Process exists and we can signal it — fall through to cmdline
		// check below.
	case errors.Is(err, syscall.EPERM):
		// Process exists but we lack permission to signal it. Still alive
		// for orphan-scan purposes; fall through to cmdline check.
	default:
		// ESRCH or any other error → process is gone.
		return false
	}

	if expectedCmdlineSubstring == "" {
		// Caller opted out of the cmdline match — preserve plain
		// signal-0-only semantics for forensic tooling. No production
		// caller does this today; OrphanScanner.Scan always supplies a
		// non-empty substring derived from manifest.CLIKind.
		return true
	}

	lookup := d.CommLookup
	if lookup == nil {
		lookup = psCommLookup
	}
	comm, lookupErr := lookup(pid)
	if lookupErr != nil {
		// Most common cause: PID exited between signal-0 and the
		// lookup. Treat as dead — reaping a since-exited process is a
		// no-op.
		return false
	}
	if comm == "" {
		// Empty output should not happen for a valid live process;
		// defending against it keeps Contains("", "claude") from
		// returning a false-positive.
		return false
	}
	return strings.Contains(comm, expectedCmdlineSubstring)
}

// OrphanScanner enumerates in_progress action items at dispatcher startup
// and resolves orphans (spawns whose monitoring goroutine died alongside a
// prior dispatcher process). Per spawn architecture memory §8.
//
// Lifecycle: the scanner is single-use per dispatcher startup. Construct it
// with NewOrphanScanner (or by populating fields directly), call Scan once,
// then discard. Concurrent calls to Scan on the same scanner are not safe —
// the scanner has no internal locking — but callers do not need locking
// today because Scan runs exactly once per dispatcher startup.
type OrphanScanner struct {
	// ActionItems is the reader the scanner uses to enumerate currently-
	// in-progress action items. Required; Scan returns
	// ErrInvalidOrphanScannerConfig when nil.
	ActionItems ActionItemReader

	// ProcessChecker probes PID liveness. Required; Scan returns
	// ErrInvalidOrphanScannerConfig when nil. Production callers wire
	// DefaultProcessChecker; tests inject a stub.
	ProcessChecker ProcessChecker

	// Logger is the narrow Printf-shaped logger the scanner uses for
	// diagnostic output (skipped items, callback errors). Optional — a nil
	// Logger discards every log line. Mirrors MonitorLogger's contract so
	// production wiring can pass the same charmbracelet/log adapter.
	Logger MonitorLogger

	// OnOrphanFound is invoked once per orphaned action item the scanner
	// detects. Production wiring routes this to a service that moves the
	// item to StateFailed (with metadata.failure_reason =
	// "dispatcher_restart_orphan") and reaps the bundle. Optional — when
	// nil, the scanner still reports orphan IDs in the Scan return slice
	// but performs no remediation. Callback errors are logged and
	// aggregated via errors.Join in the Scan return error; one failing
	// callback does NOT halt the scan.
	OnOrphanFound func(ctx context.Context, item domain.ActionItem, manifestPath string) error
}

// Scan walks every in_progress action item, reads manifest.json from
// metadata.SpawnBundlePath, checks PID liveness, and invokes OnOrphanFound
// for each item whose recorded PID is dead. Returns the list of orphan
// action-item IDs detected and an aggregated error containing every
// callback failure (errors.Join'd) — when no callback errors fire the
// returned error is nil.
//
// Per-item branches (per spawn architecture memory §8):
//
//   - SpawnBundlePath empty / whitespace-only → skip with a debug log line
//     ("action item never dispatched / bundle path missing").
//   - manifest.json missing (errors.Is(err, os.ErrNotExist)) → skip with a
//     debug log line; the bundle was reaped externally and the action item
//     should not be re-reaped here.
//   - manifest.json malformed (any other ReadManifest error) → skip with a
//     warning log line; forensic recovery tooling can inspect the bundle
//     out-of-band.
//   - ClaudePID == 0 → skip ("spawn not yet started, leave alone" per memory
//     §8); the regular dispatcher monitor handles it.
//   - PID alive → skip; healthy spawn under a fresh dispatcher cycle.
//   - PID dead → orphan. OnOrphanFound fires (when non-nil) and the action
//     item ID is appended to the result slice.
//
// ctx cancellation is honored between items; mid-item progress is not
// preempted (manifest reads are short, callback execution is the caller's
// concern).
func (s *OrphanScanner) Scan(ctx context.Context) ([]string, error) {
	if s == nil || s.ActionItems == nil {
		return nil, fmt.Errorf("%w: ActionItems reader is nil", ErrInvalidOrphanScannerConfig)
	}
	if s.ProcessChecker == nil {
		return nil, fmt.Errorf("%w: ProcessChecker is nil", ErrInvalidOrphanScannerConfig)
	}

	items, err := s.ActionItems.ListInProgress(ctx)
	if err != nil {
		return nil, fmt.Errorf("orphan scan: list in_progress action items: %w", err)
	}

	var orphans []string
	var callbackErrs []error

	for _, item := range items {
		if err := ctx.Err(); err != nil {
			return orphans, err
		}

		bundlePath := strings.TrimSpace(item.Metadata.SpawnBundlePath)
		if bundlePath == "" {
			s.logf("orphan scan: action item %s has no spawn_bundle_path; skipping", item.ID)
			continue
		}

		manifest, err := ReadManifest(bundlePath)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				s.logf("orphan scan: action item %s manifest missing at %s; skipping", item.ID, bundlePath)
				continue
			}
			s.logf("orphan scan: action item %s manifest unreadable at %s: %v; skipping", item.ID, bundlePath, err)
			continue
		}

		if manifest.ClaudePID == 0 {
			// Memory §8: PID==0 means spawn never started; leave alone.
			s.logf("orphan scan: action item %s has zero PID (spawn not yet started); skipping", item.ID)
			continue
		}

		expectedCmdline := expectedCmdlineForCLIKind(manifest.CLIKind)
		if s.ProcessChecker.IsAlive(manifest.ClaudePID, expectedCmdline) {
			s.logf("orphan scan: action item %s PID %d (cli_kind=%q) still alive; skipping", item.ID, manifest.ClaudePID, manifest.CLIKind)
			continue
		}

		// Dead PID → orphan. Fire callback (if any) and append ID.
		manifestPath := filepath.Join(bundlePath, "manifest.json")
		s.logf("orphan scan: action item %s PID %d dead; reaping orphan", item.ID, manifest.ClaudePID)
		if s.OnOrphanFound != nil {
			if cbErr := s.OnOrphanFound(ctx, item, manifestPath); cbErr != nil {
				s.logf("orphan scan: OnOrphanFound for action item %s returned error: %v", item.ID, cbErr)
				callbackErrs = append(callbackErrs, fmt.Errorf("orphan scan: OnOrphanFound %s: %w", item.ID, cbErr))
			}
		}
		orphans = append(orphans, item.ID)
	}

	if len(callbackErrs) > 0 {
		return orphans, errors.Join(callbackErrs...)
	}
	return orphans, nil
}

// logf forwards a Printf-shaped log line to the configured Logger when
// non-nil. Mirrors Monitor.log's discard-on-nil contract.
func (s *OrphanScanner) logf(format string, args ...any) {
	if s == nil || s.Logger == nil {
		return
	}
	s.Logger.Printf(format, args...)
}

// expectedCmdlineForCLIKind maps a manifest.CLIKind string to the substring
// the ProcessChecker uses to verify a live PID is actually owned by the
// expected adapter binary (Attack 1 from F.7.8 QA-Falsification: PID-reuse
// guard via cmdline match).
//
// Today the adapter set is small — "claude" and (future) "codex". The
// switch returns the binary basename verbatim. Unknown CLIKind values
// (including the empty string from older manifests written before
// F.7.17.6) fall back to the empty string, which DefaultProcessChecker
// interprets as "skip cmdline check, signal-0 only" — preserving round-1
// behaviour for legacy bundles rather than mis-reaping a healthy spawn
// that happens to predate the guard. New adapters land here at the same
// time their CLIKind value lands in BuildSpawnCommand.
func expectedCmdlineForCLIKind(cliKind string) string {
	switch cliKind {
	case "claude":
		return "claude"
	case "codex":
		return "codex"
	default:
		return ""
	}
}
