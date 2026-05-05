package dispatcher

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/evanmschultz/tillsyn/internal/domain"
	"github.com/evanmschultz/tillsyn/internal/templates"
)

// GateStatus is the closed-enum verdict produced by one gate execution. The
// type is unexported as a package-internal closed set: gate consumers (the
// subscriber wiring landing in 4b.7, the gate-failure routing landing in
// Drop 4c) read GateResult.Status directly and never need to import a public
// closed-enum helper.
//
// Drop 4b Wave A 4b.2 ships exactly three values — there is no
// GateStatusUnknown / GateStatusPending sentinel. A gate either ran and
// passed, ran and failed, or was deliberately skipped. GateStatusSkipped is
// emitted by the runner when ctx is cancelled between gates (the next-to-run
// gate gets one Skipped row); it is also reserved for future runner-level
// filters (per-kind disable flags, dev-skip CLI knobs) that want to record
// the skip decision rather than omit the gate entirely from the result slice.
type GateStatus string

// Closed-enum GateStatus values consumed by the gate runner and downstream
// gate-failure routing (Drop 4c). The string literals are the canonical wire
// form persisted in any future gate-result audit log.
const (
	// GateStatusPassed indicates the gate's underlying gateFunc returned a
	// nil error. The result's Output and Err fields are empty/nil.
	GateStatusPassed GateStatus = "passed"

	// GateStatusFailed indicates the gate's underlying gateFunc returned a
	// non-nil error OR the gate's name was not registered in the runner. The
	// result's Output carries the captured tail of the gate's stdout/stderr
	// (per Q7 last-100-lines-or-8KB-shorter rule) and Err carries the error
	// (wrapping ErrGateNotRegistered when applicable).
	GateStatusFailed GateStatus = "failed"

	// GateStatusSkipped indicates the gate was deliberately bypassed. The
	// 4b.2 runner emits this value when ctx is cancelled BETWEEN gates
	// (inter-gate ctx.Err() check inside Run) — the next-to-run gate gets
	// one Skipped row so the result slice records "this gate did not fire
	// because ctx was cancelled," distinguishing caller-driven teardown
	// from a Failed verdict that the gate itself produced. The enum value
	// is also reserved for future runner-level filters (per-kind disable
	// flags, dev-skip CLI knobs) that want to record the skip decision
	// rather than omit the gate entirely from the result slice.
	GateStatusSkipped GateStatus = "skipped"
)

// gateFunc is the synchronous executor signature registered against a
// GateKind. The function MUST NOT spawn goroutines or rely on the runner to
// orchestrate concurrency: the runner executes every gateFunc serially in
// template-declared order so halt-on-first-failure semantics are deterministic.
//
// Implementations return a fully-populated GateResult — including GateName
// matching the registered kind, Status, Duration, and Output/Err on failure.
// The runner does NOT post-process the result beyond appending it to the
// returned slice.
type gateFunc func(ctx context.Context, item domain.ActionItem, project domain.Project) GateResult

// GateResult captures one gate's outcome. The struct is the gate-runner's
// hand-off shape consumed by the Drop 4b.7 subscriber and the Drop 4c
// gate-failure routing. Every field is populated by the gateFunc that
// produced the result; the runner only appends results to a slice.
//
// Output is bounded per REVISION_BRIEF Q7: the last 100 lines OR last 8KB,
// whichever is shorter, after defensive UTF-8 sanitization (null bytes
// dropped, invalid UTF-8 sequences replaced with U+FFFD). The bound applies
// only to failed gates — a passing gate has no output to capture.
type GateResult struct {
	// GateName identifies which gate produced this result. The value MUST
	// match the GateKind under which the gateFunc was registered; the runner
	// does not enforce this invariant but downstream consumers rely on it.
	GateName templates.GateKind

	// Status is the closed-enum verdict. GateStatusPassed means the underlying
	// gateFunc returned nil; GateStatusFailed means it returned a non-nil error
	// OR the gate name was not registered in the runner; GateStatusSkipped is
	// reserved for runner-level filters not yet wired in 4b.2.
	Status GateStatus

	// Output is the bounded tail of the gate's captured stdout/stderr per the
	// last-100-lines-or-8KB-shorter rule. Empty on GateStatusPassed (the gate
	// passed; nothing to capture) and GateStatusSkipped (the gate never ran).
	// Sanitized for UTF-8 validity.
	Output string

	// Duration is the wall-clock execution time of the underlying gateFunc.
	// Set even on GateStatusFailed so dashboards and the dev's failure-loop
	// instinct (slow gate vs cold cache vs flaky network) have a number to
	// reach for.
	Duration time.Duration

	// Err carries the non-nil error returned by the gateFunc on failure or
	// the ErrGateNotRegistered sentinel when the runner encountered a gate
	// name it could not resolve. On GateStatusSkipped emitted by the inter-
	// gate ctx-cancel check, Err carries the ctx.Err() (context.Canceled or
	// context.DeadlineExceeded) so the caller can distinguish skip-causes.
	// Nil on GateStatusPassed.
	Err error
}

// ErrGateNotRegistered is the typed sentinel returned via GateResult.Err when
// the runner encounters a gate name in tpl.Gates[kind] that has no matching
// Register() call. Detect with errors.Is. The runner halts subsequent gates
// after appending one GateStatusFailed result with Err: ErrGateNotRegistered
// — same halt-on-first-failure contract as a gate that ran and returned an
// error.
var ErrGateNotRegistered = errors.New("dispatcher: gate not registered")

// ErrGateAlreadyRegistered is the typed sentinel returned by Register when
// the supplied gate name is already mapped to a gateFunc. Detect with
// errors.Is. Re-registration is rejected rather than silently overwritten so
// dispatcher wiring (4b.7 subscriber) cannot accidentally double-bind a gate
// after a hot-reload or test-fixture reuse.
var ErrGateAlreadyRegistered = errors.New("dispatcher: gate already registered")

// Output-capture bounds per REVISION_BRIEF Q7. Exposed as exported constants
// so test fixtures and downstream consumers (Drop 4c gate-failure attention
// items) can reference the same numbers without re-deriving them.
const (
	// MaxGateOutputLines caps the tail line count per Q7 (last 100 lines).
	MaxGateOutputLines = 100

	// MaxGateOutputBytes caps the tail byte count per Q7 (last 8KB).
	MaxGateOutputBytes = 8 * 1024
)

// gateRunner orchestrates serial execution of the gate sequence declared by a
// template's Template.Gates[kind] entry against one action item. The runner
// is the dispatcher's single chokepoint for post-build verification: every
// gate fires through Run, every result lands in the returned slice, and the
// caller (Drop 4b.7 subscriber) is solely responsible for converting the
// slice into action-item state transitions.
//
// Halt-on-first-failure semantics: Run iterates tpl.Gates[item.Kind] in
// template-declared order; the first GateStatusFailed result halts iteration
// and the failed result is appended last. Subsequent gates in the slice are
// NOT invoked. This matches REVISION_BRIEF locked decisions L1 (closed-enum
// gates) and L2 (sequential execution, fail-fast).
//
// Rollback policy: Wave A gates (mage_ci, mage_test_pkg) have NO side effects
// requiring rollback — failed gates leave no state to undo. Drop 4c's commit
// + push gates DO produce side effects (a created commit, a pushed ref); the
// rollback design for those gates is explicitly out of scope here and lands
// alongside the Drop 4c gate definitions.
//
// Concurrency: gateRunner IS safe for concurrent Register and Run calls under
// a sync.RWMutex. Register takes the write lock; Run takes the read lock for
// the duration of each gate-name lookup. The expected lifecycle is still
// construct → register every gate during dispatcher startup → call Run from
// the subscriber's serial event loop, but the lock ensures concurrent Runs
// across goroutines (e.g. parallel-dispatch experiments, or test fixtures
// invoking Run from multiple goroutines under -race) do not data-race on the
// registry map. Per WAVE_A_PLAN.md §4b.2 line 143.
type gateRunner struct {
	mu    sync.RWMutex
	gates map[templates.GateKind]gateFunc
}

// newGateRunner constructs an empty gateRunner. Callers populate it via
// Register before invoking Run. The zero gateRunner is NOT a usable runner —
// the gates map must be allocated, hence the constructor.
func newGateRunner() *gateRunner {
	return &gateRunner{
		gates: make(map[templates.GateKind]gateFunc),
	}
}

// Register binds the supplied gateFunc to the closed-enum gate name. Returns
// ErrGateAlreadyRegistered (wrapped with the duplicate name for diagnostics)
// when name is already mapped. The runner does NOT validate name against
// templates.IsValidGateKind here — the validation lives at template-load time
// (Drop 3.10) and re-running it on every Register would couple the runner to
// the template package's enum more tightly than necessary. A caller that
// registers an out-of-enum name has a bug; the runner detects it indirectly
// when Run() iterates a template that references the same out-of-enum name
// (and returns ErrGateNotRegistered for the lookup miss, which is the right
// surface for the failure mode).
//
// The fn argument is permitted to be nil — a nil gateFunc registered against
// a name will panic when Run invokes it, which is the dev's bug to fix
// rather than the runner's to absorb. Nil-checking would mask a wiring bug
// that should be caught at startup smoke-test.
func (r *gateRunner) Register(name templates.GateKind, fn gateFunc) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.gates[name]; exists {
		return fmt.Errorf("%w: %q", ErrGateAlreadyRegistered, name)
	}
	r.gates[name] = fn
	return nil
}

// Run executes the gate sequence declared in tpl.Gates[item.Kind] against the
// supplied action item and project. Returns one GateResult per gate executed
// (including the failing one when execution halts early). The caller is
// responsible for converting the slice into action-item state transitions —
// the runner does NOT mutate item.
//
// Empty-template handling: tpl == nil returns nil, nil — defensive against a
// caller that has not yet bound a template to the project. Likewise an empty
// tpl.Gates[item.Kind] (no gates declared for the kind) returns nil, nil.
// Both cases are documented in WAVE_A_PLAN.md §4b.2 acceptance: "absence of
// gates means no gates fire."
//
// Halt-on-first-failure: as soon as a gateFunc returns Status != Passed (or
// the lookup misses with ErrGateNotRegistered), the runner appends the result
// and returns without invoking the remaining gates in the sequence.
//
// Inter-gate context cancellation: ctx.Err() is checked BEFORE each gateFunc
// invocation. On a non-nil ctx.Err() the runner appends one
// GateStatusSkipped result naming the next-to-run gate (with Err set to the
// cancellation cause) and halts iteration. Skipped — not Failed — because the
// cancellation is external (caller decision: timeout, shutdown, parent ctx
// teardown) rather than a verdict from the gate itself. Skipping a gate that
// never ran is the honest report; failing it would conflate caller-driven
// teardown with a gate-driven rejection. A pre-loop ctx.Err() (cancelled
// before any gate dispatches) returns nil — there is no gate to attribute
// the skip to, and the empty result slice already encodes "no work happened".
//
// Concurrent Run calls across goroutines are safe: the registry-map read
// path takes r.mu.RLock(), and per-Run state (the results slice, gateFunc
// invocations) is local. Each goroutine's Run returns its own results slice.
func (r *gateRunner) Run(ctx context.Context, item domain.ActionItem, project domain.Project, tpl *templates.Template) []GateResult {
	if tpl == nil {
		return nil
	}
	sequence := tpl.Gates[item.Kind]
	if len(sequence) == 0 {
		return nil
	}
	// Pre-loop ctx-cancel check: caller cancelled before any gate dispatched.
	// Return nil (no gate to attribute a skip to) — the empty slice already
	// encodes "no work happened". See doc-comment for the inter-gate variant.
	if err := ctx.Err(); err != nil {
		return nil
	}

	results := make([]GateResult, 0, len(sequence))
	for _, name := range sequence {
		// Inter-gate ctx-cancel check: caller cancelled between gates.
		// Append one Skipped row naming the next-to-run gate so the result
		// slice carries an honest record of "this gate did not fire because
		// ctx was cancelled," then halt. Skipped — not Failed — because the
		// cancellation is external rather than a verdict from the gate.
		if err := ctx.Err(); err != nil {
			results = append(results, GateResult{
				GateName: name,
				Status:   GateStatusSkipped,
				Err:      err,
			})
			return results
		}
		r.mu.RLock()
		fn, ok := r.gates[name]
		r.mu.RUnlock()
		if !ok {
			results = append(results, GateResult{
				GateName: name,
				Status:   GateStatusFailed,
				Err:      fmt.Errorf("%w: %q", ErrGateNotRegistered, name),
			})
			return results
		}
		result := fn(ctx, item, project)
		results = append(results, result)
		if result.Status != GateStatusPassed {
			return results
		}
	}
	return results
}

// tailOutput returns the bounded tail of b per the REVISION_BRIEF Q7 rule:
// the last maxLines lines OR the last maxBytes bytes, whichever produces the
// SHORTER output. Defensive UTF-8 sanitization runs after truncation: every
// null byte is dropped and every invalid UTF-8 sequence is replaced with
// U+FFFD via strings.ToValidUTF8.
//
// The "shorter of" semantics mean a single very-long line (e.g. a Go test
// failure dump on one line) gets byte-bounded even when its line count is
// well under maxLines, and a high-volume verbose log gets line-bounded when
// the last 100 lines fit under 8KB. Defensive against either runaway output
// shape.
//
// Edge cases:
//   - len(b) == 0 returns "".
//   - maxLines <= 0 falls back to byte-only bounding.
//   - maxBytes <= 0 falls back to line-only bounding.
//   - Both <= 0 returns the sanitized full input (degenerate; callers are
//     expected to pass positive bounds, but the function does not panic).
func tailOutput(b []byte, maxLines int, maxBytes int) string {
	if len(b) == 0 {
		return ""
	}

	byLines := tailByLines(b, maxLines)
	byBytes := tailByBytes(b, maxBytes)

	var chosen []byte
	switch {
	case maxLines <= 0 && maxBytes <= 0:
		chosen = b
	case maxLines <= 0:
		chosen = byBytes
	case maxBytes <= 0:
		chosen = byLines
	default:
		// Shorter byte length wins — matches the Q7 "whichever is shorter" rule.
		if len(byLines) <= len(byBytes) {
			chosen = byLines
		} else {
			chosen = byBytes
		}
	}

	return sanitizeUTF8(chosen)
}

// tailByLines returns a sub-slice of b containing the last maxLines lines.
// A "line" is everything terminated by '\n' (the trailing newline is kept).
// If b has fewer than maxLines newlines the full input is returned. Returns
// b unchanged when maxLines <= 0.
func tailByLines(b []byte, maxLines int) []byte {
	if maxLines <= 0 || len(b) == 0 {
		return b
	}
	count := 0
	// Walk backwards counting '\n'. We want the slice that starts AFTER the
	// (maxLines)th-to-last '\n' so the last maxLines lines remain.
	for i := len(b) - 1; i >= 0; i-- {
		if b[i] == '\n' {
			count++
			if count > maxLines {
				return b[i+1:]
			}
		}
	}
	return b
}

// tailByBytes returns a sub-slice of b containing the last maxBytes bytes.
// If len(b) <= maxBytes the input is returned unchanged. Returns b unchanged
// when maxBytes <= 0.
func tailByBytes(b []byte, maxBytes int) []byte {
	if maxBytes <= 0 || len(b) <= maxBytes {
		return b
	}
	return b[len(b)-maxBytes:]
}

// sanitizeUTF8 drops null bytes from b and replaces every invalid UTF-8
// sequence with U+FFFD via strings.ToValidUTF8. Defensive against gate
// outputs that include binary blobs (e.g. a panic dump that captured a
// non-UTF-8 raw byte) which would otherwise crash JSON-encoding consumers
// downstream (action-item metadata persistence, attention-item bodies).
func sanitizeUTF8(b []byte) string {
	withoutNulls := bytes.ReplaceAll(b, []byte{0}, nil)
	return strings.ToValidUTF8(string(withoutNulls), "�")
}
