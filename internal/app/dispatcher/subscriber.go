package dispatcher

import (
	"context"
	"fmt"
	"strings"
)

// Drop 4b.7 — continuous-mode subscriber loop.
//
// Start enumerates the project list once via projectsLister.ListProjects and
// spins one goroutine per project. Each goroutine subscribes to
// LiveWaitEventActionItemChanged via subscribeBroker (broker_sub.go) and on
// every received event walks the project tree via walker.EligibleForPromotion
// + invokes RunOnce on every eligible item. Errors are intentionally
// swallowed inside the loop because the next state-change event gives the
// dispatcher another chance to promote any straggler item.
//
// Stop cancels the parent ctx all subscriber goroutines share and waits for
// every goroutine to drain via the internal sync.WaitGroup.
//
// Spawn invocation today goes through the existing 4a.19 spawn stub
// (BuildSpawnCommand). Drop 4c F.7 lands the real Claude-Code spawn pipeline;
// the subscriber loop survives that swap unchanged because it only consumes
// RunOnce's DispatchOutcome contract.
//
// New projects added during runtime are NOT picked up — Start enumerates
// projects once at startup. This is acceptable for the Drop 4b MVP scope per
// WAVE_C_PLAN.md §4.5 Option B; Drop 4c / Drop 5 dogfood evaluates whether
// dynamic project subscription is needed.

// Start begins the continuous-mode dispatcher loop.
//
// Idempotency:
//   - A second Start call before Stop returns ErrAlreadyStarted without
//     spawning duplicate goroutines.
//   - A Start call after Stop is also rejected with ErrAlreadyStarted —
//     re-start of a stopped dispatcher is not supported. Construct a fresh
//     *dispatcher instead.
//
// ctx is the parent context for Start's one-shot ListProjects call. The
// subscriber goroutines run under a cancellable child context derived from
// context.Background() (NOT from ctx) so a short-lived ctx does not tear
// down the long-lived subscribers when Start returns. Stop owns the
// child-ctx cancellation.
//
// Returns nil on success; ErrAlreadyStarted on duplicate Start;
// ErrInvalidDispatcherConfig on a missing dependency (broker / walker /
// projectsLister); a wrapped error on ListProjects failure.
func (d *dispatcher) Start(ctx context.Context) error {
	if d == nil {
		return fmt.Errorf("%w: dispatcher is nil", ErrInvalidDispatcherConfig)
	}
	if d.broker == nil {
		return fmt.Errorf("%w: broker is nil", ErrInvalidDispatcherConfig)
	}
	if d.walker == nil {
		return fmt.Errorf("%w: walker is nil", ErrInvalidDispatcherConfig)
	}
	if d.projectsLister == nil {
		return fmt.Errorf("%w: projects lister is nil", ErrInvalidDispatcherConfig)
	}

	d.subMu.Lock()
	if d.started || d.stopped {
		d.subMu.Unlock()
		return ErrAlreadyStarted
	}

	projects, err := d.projectsLister.ListProjects(ctx, false)
	if err != nil {
		d.subMu.Unlock()
		return fmt.Errorf("dispatcher start: list projects: %w", err)
	}

	subCtx, cancel := context.WithCancel(context.Background())
	d.subCancel = cancel
	d.started = true

	for _, project := range projects {
		projectID := strings.TrimSpace(project.ID)
		if projectID == "" {
			continue
		}
		d.subWG.Add(1)
		go d.runSubscriberLoop(subCtx, projectID)
	}

	d.subMu.Unlock()
	return nil
}

// Stop tears down the continuous-mode dispatcher loop. Stop cancels the
// parent context all subscriber goroutines share and waits for every
// goroutine to drain via the internal sync.WaitGroup. Stop returns nil on
// clean drain, ctx.Err() if the supplied context expires before every
// goroutine exits.
//
// Stop is idempotent: calling Stop without a prior Start, or calling Stop
// twice, returns nil. Concurrent Start + Stop are safe — Start holds the
// internal mutex across project enumeration + goroutine spawning so Stop
// never observes a half-spawned dispatcher.
func (d *dispatcher) Stop(ctx context.Context) error {
	if d == nil {
		return nil
	}

	d.subMu.Lock()
	if !d.started || d.stopped {
		d.subMu.Unlock()
		return nil
	}
	d.stopped = true
	cancel := d.subCancel
	d.subMu.Unlock()

	if cancel != nil {
		cancel()
	}

	// Wait for all subscriber goroutines to drain. Honor ctx deadline so a
	// hung subscriber does not block process shutdown indefinitely.
	done := make(chan struct{})
	go func() {
		d.subWG.Wait()
		close(done)
	}()

	if ctx == nil {
		<-done
		return nil
	}
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// runSubscriberLoop is the per-project subscriber goroutine body. Subscribes
// to LiveWaitEventActionItemChanged for projectID, then on every received
// event walks the project tree via walker.EligibleForPromotion and invokes
// RunOnce on every eligible item.
//
// The loop exits cleanly when subscribeBroker's channel closes (which happens
// on parent-ctx cancellation per broker_sub.go's contract) or when the
// parent ctx expires.
func (d *dispatcher) runSubscriberLoop(ctx context.Context, projectID string) {
	defer d.subWG.Done()

	events := d.subscribeBroker(ctx, projectID)
	for {
		select {
		case <-ctx.Done():
			return
		case _, ok := <-events:
			if !ok {
				return
			}
			d.handleSubscriberEvent(ctx, projectID)
		}
	}
}

// handleSubscriberEvent walks the project tree once and invokes RunOnce on
// every eligible item. Split out from runSubscriberLoop so the test suite
// can drive the per-event handler without standing up the broker plumbing.
//
// Errors are swallowed: a single eligibility-walk or spawn failure must not
// kill the subscriber loop because the next event gives the dispatcher
// another chance to promote the item. Pre-cascade-dogfood there is no
// logger wired into the dispatcher package; surface logging arrives with
// Drop 4c F.7's spawn pipeline rewrite.
func (d *dispatcher) handleSubscriberEvent(ctx context.Context, projectID string) {
	if d == nil || d.walker == nil {
		return
	}
	eligible, err := d.walker.EligibleForPromotion(ctx, projectID)
	if err != nil {
		return
	}
	for _, item := range eligible {
		if err := ctx.Err(); err != nil {
			return
		}
		// Empty projectIDOverride: RunOnce resolves the project from the
		// item itself. The override exists for the manual-trigger CLI path
		// (4a.23 §2.2) where the dev passes --project; the subscriber has
		// already trusted the broker's project-scoped fan-out.
		_, _ = d.RunOnce(ctx, item.ID, "")
	}
}
