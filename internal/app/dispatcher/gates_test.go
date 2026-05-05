package dispatcher

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/evanmschultz/tillsyn/internal/domain"
	"github.com/evanmschultz/tillsyn/internal/templates"
)

// gateFixtureItem returns an action item whose Kind is KindBuild — the
// canonical post-build gate target the Wave A 4b.2 runner exercises.
func gateFixtureItem() domain.ActionItem {
	return domain.ActionItem{
		ID:   "ai-build-gate-1",
		Kind: domain.KindBuild,
	}
}

// gateFixtureProject returns a minimal project value for gate tests. The
// runner does not read project fields directly; gateFuncs receive it as
// pass-through, and the fixture exists so signature wiring matches the
// production call site.
func gateFixtureProject() domain.Project {
	return domain.Project{
		ID: "proj-gate-1",
	}
}

// gateFixtureTemplate returns a *templates.Template whose Gates map binds
// KindBuild to the supplied gate sequence. Used by every Run() test.
func gateFixtureTemplate(sequence []templates.GateKind) *templates.Template {
	return &templates.Template{
		SchemaVersion: templates.SchemaVersionV1,
		Gates: map[domain.Kind][]templates.GateKind{
			domain.KindBuild: sequence,
		},
	}
}

// passingGate returns a gateFunc that records its invocation count via the
// supplied counter and returns GateStatusPassed under the supplied name.
func passingGate(name templates.GateKind, counter *int) gateFunc {
	return func(ctx context.Context, _ domain.ActionItem, _ domain.Project) GateResult {
		*counter++
		return GateResult{
			GateName: name,
			Status:   GateStatusPassed,
			Duration: time.Millisecond,
		}
	}
}

// failingGate returns a gateFunc that records its invocation count and
// returns GateStatusFailed under the supplied name with the supplied error
// embedded in Err.
func failingGate(name templates.GateKind, counter *int, err error) gateFunc {
	return func(ctx context.Context, _ domain.ActionItem, _ domain.Project) GateResult {
		*counter++
		return GateResult{
			GateName: name,
			Status:   GateStatusFailed,
			Output:   "stderr line\n",
			Duration: time.Millisecond,
			Err:      err,
		}
	}
}

// TestGateRunnerRegisterRejectsDuplicate asserts Register() returns
// ErrGateAlreadyRegistered when the same gate name is registered twice and
// the original gateFunc remains bound (i.e. silent overwrite is rejected).
func TestGateRunnerRegisterRejectsDuplicate(t *testing.T) {
	t.Parallel()

	runner := newGateRunner()
	originalCalls := 0
	original := passingGate(templates.GateKindMageCI, &originalCalls)
	if err := runner.Register(templates.GateKindMageCI, original); err != nil {
		t.Fatalf("first Register error = %v, want nil", err)
	}

	overwriteCalls := 0
	overwrite := passingGate(templates.GateKindMageCI, &overwriteCalls)
	err := runner.Register(templates.GateKindMageCI, overwrite)
	if err == nil {
		t.Fatalf("second Register error = nil, want non-nil")
	}
	if !errors.Is(err, ErrGateAlreadyRegistered) {
		t.Fatalf("second Register error = %v, want errors.Is ErrGateAlreadyRegistered", err)
	}
	if !strings.Contains(err.Error(), string(templates.GateKindMageCI)) {
		t.Fatalf("Register error = %v, want gate name in message", err)
	}

	// Confirm the original gateFunc is still bound — the second Register
	// must not silently overwrite. Run one gate sequence and verify the
	// original counter ticks, not the overwrite counter.
	tpl := gateFixtureTemplate([]templates.GateKind{templates.GateKindMageCI})
	runner.Run(context.Background(), gateFixtureItem(), gateFixtureProject(), tpl)
	if originalCalls != 1 {
		t.Fatalf("originalCalls = %d, want 1 (original gateFunc still bound)", originalCalls)
	}
	if overwriteCalls != 0 {
		t.Fatalf("overwriteCalls = %d, want 0 (overwrite must NOT have replaced original)", overwriteCalls)
	}
}

// TestGateRunnerRunHaltsOnFirstFailure asserts the runner stops invoking
// subsequent gates as soon as one returns GateStatusFailed. The second gate's
// invocation counter MUST remain zero.
func TestGateRunnerRunHaltsOnFirstFailure(t *testing.T) {
	t.Parallel()

	runner := newGateRunner()
	firstCalls := 0
	secondCalls := 0
	gateErr := errors.New("synthetic gate failure")
	if err := runner.Register(templates.GateKindMageCI, failingGate(templates.GateKindMageCI, &firstCalls, gateErr)); err != nil {
		t.Fatalf("Register mage_ci error = %v, want nil", err)
	}
	if err := runner.Register(templates.GateKindMageTestPkg, passingGate(templates.GateKindMageTestPkg, &secondCalls)); err != nil {
		t.Fatalf("Register mage_test_pkg error = %v, want nil", err)
	}

	tpl := gateFixtureTemplate([]templates.GateKind{templates.GateKindMageCI, templates.GateKindMageTestPkg})
	results := runner.Run(context.Background(), gateFixtureItem(), gateFixtureProject(), tpl)

	if firstCalls != 1 {
		t.Fatalf("firstCalls = %d, want 1", firstCalls)
	}
	if secondCalls != 0 {
		t.Fatalf("secondCalls = %d, want 0 (halt-on-first-failure violated)", secondCalls)
	}
	if len(results) != 1 {
		t.Fatalf("len(results) = %d, want 1 (only failing gate's row appended)", len(results))
	}
	if results[0].Status != GateStatusFailed {
		t.Fatalf("results[0].Status = %q, want %q", results[0].Status, GateStatusFailed)
	}
	if !errors.Is(results[0].Err, gateErr) {
		t.Fatalf("results[0].Err = %v, want errors.Is gateErr", results[0].Err)
	}
}

// TestGateRunnerRunReturnsResultsInOrder asserts every gate's result lands in
// the returned slice in template-declared order when all gates pass.
func TestGateRunnerRunReturnsResultsInOrder(t *testing.T) {
	t.Parallel()

	runner := newGateRunner()
	firstCalls := 0
	secondCalls := 0
	if err := runner.Register(templates.GateKindMageCI, passingGate(templates.GateKindMageCI, &firstCalls)); err != nil {
		t.Fatalf("Register mage_ci error = %v, want nil", err)
	}
	if err := runner.Register(templates.GateKindMageTestPkg, passingGate(templates.GateKindMageTestPkg, &secondCalls)); err != nil {
		t.Fatalf("Register mage_test_pkg error = %v, want nil", err)
	}

	tpl := gateFixtureTemplate([]templates.GateKind{templates.GateKindMageCI, templates.GateKindMageTestPkg})
	results := runner.Run(context.Background(), gateFixtureItem(), gateFixtureProject(), tpl)

	if firstCalls != 1 {
		t.Fatalf("firstCalls = %d, want 1", firstCalls)
	}
	if secondCalls != 1 {
		t.Fatalf("secondCalls = %d, want 1", secondCalls)
	}
	if len(results) != 2 {
		t.Fatalf("len(results) = %d, want 2", len(results))
	}
	if results[0].GateName != templates.GateKindMageCI {
		t.Fatalf("results[0].GateName = %q, want %q", results[0].GateName, templates.GateKindMageCI)
	}
	if results[1].GateName != templates.GateKindMageTestPkg {
		t.Fatalf("results[1].GateName = %q, want %q", results[1].GateName, templates.GateKindMageTestPkg)
	}
	for i, r := range results {
		if r.Status != GateStatusPassed {
			t.Fatalf("results[%d].Status = %q, want %q", i, r.Status, GateStatusPassed)
		}
		if r.Err != nil {
			t.Fatalf("results[%d].Err = %v, want nil", i, r.Err)
		}
	}
}

// TestGateRunnerRunReturnsErrGateNotRegistered asserts that when a template
// references a gate name with no Register() binding, the runner returns one
// GateStatusFailed result wrapping ErrGateNotRegistered and halts subsequent
// gates per the same halt-on-first-failure contract.
func TestGateRunnerRunReturnsErrGateNotRegistered(t *testing.T) {
	t.Parallel()

	runner := newGateRunner()
	subsequentCalls := 0
	// Only register mage_test_pkg. The template references mage_ci first
	// (unregistered) followed by mage_test_pkg (registered) — the runner
	// must fail on mage_ci and never invoke mage_test_pkg.
	if err := runner.Register(templates.GateKindMageTestPkg, passingGate(templates.GateKindMageTestPkg, &subsequentCalls)); err != nil {
		t.Fatalf("Register mage_test_pkg error = %v, want nil", err)
	}

	tpl := gateFixtureTemplate([]templates.GateKind{templates.GateKindMageCI, templates.GateKindMageTestPkg})
	results := runner.Run(context.Background(), gateFixtureItem(), gateFixtureProject(), tpl)

	if subsequentCalls != 0 {
		t.Fatalf("subsequentCalls = %d, want 0 (halt after unregistered-gate failure)", subsequentCalls)
	}
	if len(results) != 1 {
		t.Fatalf("len(results) = %d, want 1", len(results))
	}
	if results[0].GateName != templates.GateKindMageCI {
		t.Fatalf("results[0].GateName = %q, want %q", results[0].GateName, templates.GateKindMageCI)
	}
	if results[0].Status != GateStatusFailed {
		t.Fatalf("results[0].Status = %q, want %q", results[0].Status, GateStatusFailed)
	}
	if !errors.Is(results[0].Err, ErrGateNotRegistered) {
		t.Fatalf("results[0].Err = %v, want errors.Is ErrGateNotRegistered", results[0].Err)
	}
}

// TestGateRunnerRunEmptyGatesForKind asserts that a template whose Gates map
// has no entry for the action item's kind (or an empty slice) yields a nil
// results slice with no gateFunc invocations.
func TestGateRunnerRunEmptyGatesForKind(t *testing.T) {
	t.Parallel()

	runner := newGateRunner()
	calls := 0
	if err := runner.Register(templates.GateKindMageCI, passingGate(templates.GateKindMageCI, &calls)); err != nil {
		t.Fatalf("Register mage_ci error = %v, want nil", err)
	}

	// Case 1: KindBuild bound to an empty slice.
	tplEmpty := gateFixtureTemplate([]templates.GateKind{})
	results := runner.Run(context.Background(), gateFixtureItem(), gateFixtureProject(), tplEmpty)
	if len(results) != 0 {
		t.Fatalf("len(results) for empty slice = %d, want 0", len(results))
	}
	if calls != 0 {
		t.Fatalf("calls for empty slice = %d, want 0", calls)
	}

	// Case 2: Gates map has no entry for KindBuild at all.
	tplMissing := &templates.Template{
		SchemaVersion: templates.SchemaVersionV1,
		Gates: map[domain.Kind][]templates.GateKind{
			domain.KindPlan: {templates.GateKindMageCI},
		},
	}
	results = runner.Run(context.Background(), gateFixtureItem(), gateFixtureProject(), tplMissing)
	if len(results) != 0 {
		t.Fatalf("len(results) for missing kind = %d, want 0", len(results))
	}
	if calls != 0 {
		t.Fatalf("calls for missing kind = %d, want 0", calls)
	}
}

// TestGateRunnerRunNilTemplate asserts that a nil template input is handled
// defensively (no panic, empty result slice).
func TestGateRunnerRunNilTemplate(t *testing.T) {
	t.Parallel()

	runner := newGateRunner()
	calls := 0
	if err := runner.Register(templates.GateKindMageCI, passingGate(templates.GateKindMageCI, &calls)); err != nil {
		t.Fatalf("Register mage_ci error = %v, want nil", err)
	}

	results := runner.Run(context.Background(), gateFixtureItem(), gateFixtureProject(), nil)
	if len(results) != 0 {
		t.Fatalf("len(results) = %d, want 0 for nil template", len(results))
	}
	if calls != 0 {
		t.Fatalf("calls = %d, want 0 for nil template", calls)
	}
}

// TestGateRunnerContextCancel asserts that a pre-cancelled context halts Run
// before any gateFunc fires (pre-loop ctx.Err() returns nil results) AND that
// a context cancelled BETWEEN gates appends one Skipped result naming the
// next-to-run gate then halts. Skipped — not Failed — because cancellation
// is external (caller decision: timeout, shutdown) rather than a verdict
// from the gate itself. WAVE_A_PLAN.md §4b.2 spec test.
func TestGateRunnerContextCancel(t *testing.T) {
	t.Parallel()

	// Case 1: pre-cancelled ctx → no gate fires, nil result slice.
	t.Run("PreCancelReturnsNil", func(t *testing.T) {
		t.Parallel()
		runner := newGateRunner()
		calls := 0
		if err := runner.Register(templates.GateKindMageCI, passingGate(templates.GateKindMageCI, &calls)); err != nil {
			t.Fatalf("Register error = %v, want nil", err)
		}
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // cancel before Run dispatches.
		tpl := gateFixtureTemplate([]templates.GateKind{templates.GateKindMageCI, templates.GateKindMageTestPkg})
		results := runner.Run(ctx, gateFixtureItem(), gateFixtureProject(), tpl)
		if len(results) != 0 {
			t.Fatalf("len(results) for pre-cancel = %d, want 0", len(results))
		}
		if calls != 0 {
			t.Fatalf("calls for pre-cancel = %d, want 0 (no gate must fire)", calls)
		}
	})

	// Case 2: ctx cancelled BETWEEN gates → first gate runs, second is
	// recorded as Skipped with ctx.Err() in Err, no further gates fire.
	t.Run("InterGateCancelEmitsSkipped", func(t *testing.T) {
		t.Parallel()
		runner := newGateRunner()
		ctx, cancel := context.WithCancel(context.Background())
		firstCalls := 0
		secondCalls := 0
		// First gate cancels ctx as a side effect, simulating an external
		// teardown (parent ctx aborted) firing concurrent with the gate.
		// The runner's inter-gate ctx.Err() check then catches the cancel
		// before the second gate dispatches.
		first := func(_ context.Context, _ domain.ActionItem, _ domain.Project) GateResult {
			firstCalls++
			cancel()
			return GateResult{
				GateName: templates.GateKindMageCI,
				Status:   GateStatusPassed,
				Duration: time.Millisecond,
			}
		}
		if err := runner.Register(templates.GateKindMageCI, first); err != nil {
			t.Fatalf("Register first error = %v, want nil", err)
		}
		if err := runner.Register(templates.GateKindMageTestPkg, passingGate(templates.GateKindMageTestPkg, &secondCalls)); err != nil {
			t.Fatalf("Register second error = %v, want nil", err)
		}

		tpl := gateFixtureTemplate([]templates.GateKind{templates.GateKindMageCI, templates.GateKindMageTestPkg})
		results := runner.Run(ctx, gateFixtureItem(), gateFixtureProject(), tpl)

		if firstCalls != 1 {
			t.Fatalf("firstCalls = %d, want 1", firstCalls)
		}
		if secondCalls != 0 {
			t.Fatalf("secondCalls = %d, want 0 (inter-gate cancel must halt before second gate)", secondCalls)
		}
		if len(results) != 2 {
			t.Fatalf("len(results) = %d, want 2 (first Passed + second Skipped)", len(results))
		}
		if results[0].Status != GateStatusPassed {
			t.Fatalf("results[0].Status = %q, want %q", results[0].Status, GateStatusPassed)
		}
		if results[1].Status != GateStatusSkipped {
			t.Fatalf("results[1].Status = %q, want %q", results[1].Status, GateStatusSkipped)
		}
		if results[1].GateName != templates.GateKindMageTestPkg {
			t.Fatalf("results[1].GateName = %q, want %q", results[1].GateName, templates.GateKindMageTestPkg)
		}
		if !errors.Is(results[1].Err, context.Canceled) {
			t.Fatalf("results[1].Err = %v, want errors.Is context.Canceled", results[1].Err)
		}
	})
}

// TestGateRunnerNoDeduplication asserts that when a template lists the same
// gate name twice in sequence (["mage_ci", "mage_ci"]), the runner fires the
// gate twice. Per REVISION_BRIEF Q2 + locked decision L2: the gateRunner
// performs no implicit deduplication — sequence content is the template
// author's responsibility, and template-load validation (4b.1's
// validateGateKinds) is the layer that polices duplicates if the project
// chooses to forbid them. WAVE_A_PLAN.md §4b.2 spec test.
func TestGateRunnerNoDeduplication(t *testing.T) {
	t.Parallel()

	runner := newGateRunner()
	calls := 0
	if err := runner.Register(templates.GateKindMageCI, passingGate(templates.GateKindMageCI, &calls)); err != nil {
		t.Fatalf("Register mage_ci error = %v, want nil", err)
	}

	tpl := gateFixtureTemplate([]templates.GateKind{templates.GateKindMageCI, templates.GateKindMageCI})
	results := runner.Run(context.Background(), gateFixtureItem(), gateFixtureProject(), tpl)

	if calls != 2 {
		t.Fatalf("calls = %d, want 2 (no implicit dedup)", calls)
	}
	if len(results) != 2 {
		t.Fatalf("len(results) = %d, want 2", len(results))
	}
	for i, r := range results {
		if r.Status != GateStatusPassed {
			t.Fatalf("results[%d].Status = %q, want %q", i, r.Status, GateStatusPassed)
		}
		if r.GateName != templates.GateKindMageCI {
			t.Fatalf("results[%d].GateName = %q, want %q", i, r.GateName, templates.GateKindMageCI)
		}
	}
}

// TestGateRunnerConcurrentRuns asserts that multiple goroutines invoking Run
// concurrently against the same gateRunner instance do not race (verified by
// -race on `mage test-pkg ./internal/app/dispatcher`) and that each Run
// returns its own results slice with the correct outcomes. WAVE_A_PLAN.md
// §4b.2 spec test — the lock added in Round 2 is what makes this safe.
func TestGateRunnerConcurrentRuns(t *testing.T) {
	t.Parallel()

	runner := newGateRunner()
	// Use atomic-safe counter via mutex so the gate's invocation count is
	// race-free under concurrent dispatch (the test is also exercising the
	// runner's own lock; the counter mutex isolates the test fixture from
	// noise the runner is responsible for).
	var counterMu sync.Mutex
	totalCalls := 0
	gate := func(_ context.Context, _ domain.ActionItem, _ domain.Project) GateResult {
		counterMu.Lock()
		totalCalls++
		counterMu.Unlock()
		return GateResult{
			GateName: templates.GateKindMageCI,
			Status:   GateStatusPassed,
			Duration: time.Millisecond,
		}
	}
	if err := runner.Register(templates.GateKindMageCI, gate); err != nil {
		t.Fatalf("Register error = %v, want nil", err)
	}

	const goroutines = 16
	tpl := gateFixtureTemplate([]templates.GateKind{templates.GateKindMageCI})

	var wg sync.WaitGroup
	resultSlices := make([][]GateResult, goroutines)
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func(idx int) {
			defer wg.Done()
			resultSlices[idx] = runner.Run(context.Background(), gateFixtureItem(), gateFixtureProject(), tpl)
		}(i)
	}
	wg.Wait()

	if totalCalls != goroutines {
		t.Fatalf("totalCalls = %d, want %d (one gate dispatch per goroutine)", totalCalls, goroutines)
	}
	// Each goroutine got its own results slice with one Passed row.
	for i, results := range resultSlices {
		if len(results) != 1 {
			t.Fatalf("resultSlices[%d] len = %d, want 1", i, len(results))
		}
		if results[0].Status != GateStatusPassed {
			t.Fatalf("resultSlices[%d][0].Status = %q, want %q", i, results[0].Status, GateStatusPassed)
		}
	}
	// Distinct backing arrays per Run — pin against any accidental sharing.
	if goroutines >= 2 && len(resultSlices[0]) > 0 && len(resultSlices[1]) > 0 {
		// Mutate one slice's element; the other must remain unchanged. Use a
		// status-string write rather than swapping a backing array because
		// equal-content slices on different backing arrays compare equal by
		// value.
		resultSlices[0][0].Status = "mutated"
		if resultSlices[1][0].Status == "mutated" {
			t.Fatalf("resultSlices[0] and resultSlices[1] share backing array — Run must return distinct slices per goroutine")
		}
	}
}

// TestTailOutputBoundedByLines asserts tailOutput returns the last maxLines
// lines when that tail is shorter than the maxBytes tail.
func TestTailOutputBoundedByLines(t *testing.T) {
	t.Parallel()

	// Build 10 short lines. Tailing the last 3 lines should yield exactly
	// "L08\nL09\nL10\n" (12 bytes) — well under maxBytes = 1024.
	var buf strings.Builder
	for i := 1; i <= 10; i++ {
		buf.WriteString("L")
		if i < 10 {
			buf.WriteByte('0')
		}
		// tiny zero-pad
		if i < 10 {
			buf.WriteString(itoa(i))
		} else {
			buf.WriteString(itoa(i))
		}
		buf.WriteByte('\n')
	}
	got := tailOutput([]byte(buf.String()), 3, 1024)
	want := "L08\nL09\nL10\n"
	if got != want {
		t.Fatalf("tailOutput line-bounded = %q, want %q", got, want)
	}

	// Negative-bytes edge: maxBytes <= 0 falls back to line-only bounding.
	got = tailOutput([]byte(buf.String()), 2, 0)
	want = "L09\nL10\n"
	if got != want {
		t.Fatalf("tailOutput line-bounded (maxBytes=0) = %q, want %q", got, want)
	}
}

// TestTailOutputBoundedByBytes asserts tailOutput returns the last maxBytes
// bytes when that tail is shorter than the maxLines tail.
func TestTailOutputBoundedByBytes(t *testing.T) {
	t.Parallel()

	// Build a single very-long line of 'A's followed by a newline. Line count
	// is 1, byte count is 5001. With maxLines=100 the line tail is the full
	// 5001 bytes; with maxBytes=64 the byte tail is exactly 64 bytes — the
	// runner picks the shorter (64).
	long := strings.Repeat("A", 5000) + "\n"
	got := tailOutput([]byte(long), 100, 64)
	if len(got) != 64 {
		t.Fatalf("len(tailOutput byte-bounded) = %d, want 64", len(got))
	}
	// All bytes should be 'A' except possibly the last newline (which falls
	// at byte 5000; the last 64 bytes are bytes 4937..5000 — all 'A's plus
	// the trailing '\n'). Confirm last char is '\n'.
	if got[len(got)-1] != '\n' {
		t.Fatalf("tailOutput byte-bounded final byte = %q, want '\\n'", got[len(got)-1])
	}

	// maxLines <= 0 falls back to byte-only bounding.
	got = tailOutput([]byte(long), 0, 16)
	if len(got) != 16 {
		t.Fatalf("len(tailOutput byte-only) = %d, want 16", len(got))
	}
}

// TestTailOutputUTF8Sanitization asserts tailOutput drops null bytes and
// replaces invalid UTF-8 sequences with the U+FFFD replacement character.
func TestTailOutputUTF8Sanitization(t *testing.T) {
	t.Parallel()

	// Input mixes valid ASCII, a null byte, and an invalid UTF-8 sequence
	// (lone continuation byte 0x80 — never valid in any UTF-8 stream).
	input := []byte{'h', 'e', 'l', 'l', 'o', 0x00, 0x80, 'w', 'o', 'r', 'l', 'd', '\n'}
	got := tailOutput(input, 100, 1024)

	if strings.ContainsRune(got, 0) {
		t.Fatalf("tailOutput retained null byte: %q", got)
	}
	if !strings.HasPrefix(got, "hello") {
		t.Fatalf("tailOutput = %q, want prefix \"hello\"", got)
	}
	if !strings.HasSuffix(got, "world\n") {
		t.Fatalf("tailOutput = %q, want suffix \"world\\n\"", got)
	}
	// U+FFFD should appear in place of the lone 0x80 continuation byte.
	if !strings.ContainsRune(got, '�') {
		t.Fatalf("tailOutput = %q, want U+FFFD replacement for invalid UTF-8", got)
	}

	// Empty input edge: returns "" with no panic.
	if got := tailOutput(nil, 100, 1024); got != "" {
		t.Fatalf("tailOutput(nil) = %q, want \"\"", got)
	}
	if got := tailOutput([]byte{}, 100, 1024); got != "" {
		t.Fatalf("tailOutput([]) = %q, want \"\"", got)
	}
}

// itoa is a tiny zero-alloc int formatter used by the line-bounded test
// fixture so the test does not import strconv twice (style match with
// existing dispatcher tests that prefer hand-rolled helpers for fixtures).
func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	var digits [20]byte
	pos := len(digits)
	for i > 0 {
		pos--
		digits[pos] = byte('0' + i%10)
		i /= 10
	}
	return string(digits[pos:])
}
