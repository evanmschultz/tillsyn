package dispatcher

import (
	"context"
	"errors"
	"strings"

	"github.com/evanmschultz/tillsyn/internal/app"
)

// subscribeBroker wraps the single-shot LiveWaitBroker.Wait contract in a
// re-subscribe loop and emits every project-scoped action-item-changed event
// on the returned channel.
//
// Cursor handling (afterSequence): the loop seeds its cursor from
// LiveWaitBroker.Latest so it does not replay history seen before the
// dispatcher started. After each event the cursor advances to the event's
// Sequence so the next Wait request only wakes on strictly newer changes.
// Wave 2.5's tree walker reads the same event stream and relies on this
// monotonic ordering. Each wake corresponds to *at least one* repo write
// since the previous wake. Under burst conditions the broker coalesces
// consecutive publishes into the latest event observed; the walker
// re-reads the project tree on each wake so coalescing is invisible to the
// dispatch contract.
//
// Cancellation contract: callers MUST cancel ctx to terminate the
// subscription. The goroutine exits within one Wait round-trip of
// cancellation; the returned channel is closed before the goroutine returns
// so consumers iterating with `for ev := range ch` see a clean termination.
//
// Broker-close contract (mitigates the "spin on Sequence=0 forever" attack
// raised by plan-QA falsification): the in-process LiveWaitBroker today has
// no Close method — the broker lives for the process lifetime and the only
// supported termination signal for a subscriber is ctx cancellation. Any
// future broker implementation that closes its underlying channel MUST
// surface a non-nil, non-context error from Wait when the broker is closed
// so this loop exits cleanly. The loop treats every non-context error as a
// fatal subscription error and stops; this prevents a buggy broker from
// returning (zero-event, nil-error) in a tight loop. Both context.Canceled
// and context.DeadlineExceeded are treated as termination signals and exit
// the loop. subscribeBroker does not wrap Wait in its own timeout; if a
// caller passes a deadline-bearing ctx, the deadline IS the intended
// subscription lifetime.
//
// projectID is trimmed; an empty projectID returns an immediately-closed
// channel so callers that pass a misconfigured input do not leak a goroutine.
func (d *dispatcher) subscribeBroker(ctx context.Context, projectID string) <-chan app.LiveWaitEvent {
	out := make(chan app.LiveWaitEvent, 1)
	projectID = strings.TrimSpace(projectID)
	if d == nil || d.broker == nil || projectID == "" {
		close(out)
		return out
	}

	go d.runBrokerSubscriber(ctx, projectID, out)
	return out
}

// runBrokerSubscriber is the goroutine body for subscribeBroker. Split out so
// the test suite can read the subscription contract without re-implementing
// the channel + cancellation handling.
func (d *dispatcher) runBrokerSubscriber(ctx context.Context, projectID string, out chan<- app.LiveWaitEvent) {
	defer close(out)

	cursor := int64(0)
	if latest, ok, err := d.broker.Latest(ctx, app.LiveWaitEventActionItemChanged, projectID); err == nil && ok {
		cursor = latest.Sequence
	}

	for {
		if err := ctx.Err(); err != nil {
			return
		}
		event, err := d.broker.Wait(ctx, app.LiveWaitEventActionItemChanged, projectID, cursor)
		if err != nil {
			// ctx cancelled or timed out: exit the loop. Wave 2.2 does not
			// distinguish DeadlineExceeded from Canceled because callers
			// own the context lifetime.
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return
			}
			// Any other error indicates a broker-level fault (closed
			// channel, configuration error). Exit so we do not spin.
			return
		}
		// Advance the cursor before delivery so a slow consumer does not
		// stall the broker's monotonic ordering.
		if event.Sequence > cursor {
			cursor = event.Sequence
		}
		select {
		case <-ctx.Done():
			return
		case out <- event:
		}
	}
}
