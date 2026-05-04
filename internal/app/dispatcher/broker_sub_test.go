package dispatcher

import (
	"context"
	"testing"
	"time"

	"github.com/evanmschultz/tillsyn/internal/app"
)

// TestDispatcherSubscribesToActionItemChanges asserts that publishing one
// LiveWaitEventActionItemChanged event via the broker delivers it to the
// subscriber channel within 100 ms.
//
// Synchronization note: the in-process broker stamps every Publish into its
// latest map BEFORE waking waiters; the subscriber goroutine seeds its
// cursor from Latest at start. To avoid a race on whether the publish lands
// before or after the goroutine registers, the test publishes AFTER giving
// the subscriber a brief moment to seed its cursor and register a waiter.
// 100 ms is the contract bound — actual delivery latency is microseconds.
func TestDispatcherSubscribesToActionItemChanges(t *testing.T) {
	t.Parallel()

	broker := app.NewInProcessLiveWaitBroker()
	d := &dispatcher{
		svc:    &stubActionItemReader{},
		broker: broker,
		clock:  time.Now,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch := d.subscribeBroker(ctx, "proj-1")

	// Publish in a goroutine after a small settle delay so the subscriber
	// has registered a Wait before the publish lands. If the publish wins
	// the race, the subscriber's first Latest seed (Sequence=N) advances
	// its cursor past the published event and the subsequent Wait blocks
	// forever — failure mode the 100 ms timeout catches.
	go func() {
		// 5 ms is generous: the subscriber goroutine yields once via the
		// scheduler before reaching Wait, so this is comfortably in the
		// "after register" window without being a flaky long sleep.
		time.Sleep(5 * time.Millisecond)
		broker.Publish(app.LiveWaitEvent{
			Type:  app.LiveWaitEventActionItemChanged,
			Key:   "proj-1",
			Value: "proj-1",
		})
	}()

	select {
	case event, ok := <-ch:
		if !ok {
			t.Fatalf("subscriber channel closed before delivering event")
		}
		if event.Type != app.LiveWaitEventActionItemChanged {
			t.Fatalf("event Type = %q, want %q", event.Type, app.LiveWaitEventActionItemChanged)
		}
		if event.Key != "proj-1" {
			t.Fatalf("event Key = %q, want %q", event.Key, "proj-1")
		}
		if event.Sequence <= 0 {
			t.Fatalf("event Sequence = %d, want > 0", event.Sequence)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatalf("timed out waiting for subscriber delivery within 100 ms")
	}
}

// TestDispatcherStopsOnContextCancel asserts that cancelling the subscription
// context terminates the goroutine within 100 ms (no leak) and closes the
// returned channel.
func TestDispatcherStopsOnContextCancel(t *testing.T) {
	t.Parallel()

	broker := app.NewInProcessLiveWaitBroker()
	d := &dispatcher{
		svc:    &stubActionItemReader{},
		broker: broker,
		clock:  time.Now,
	}

	ctx, cancel := context.WithCancel(context.Background())
	ch := d.subscribeBroker(ctx, "proj-cancel")

	// Yield briefly so the goroutine reaches Wait before we cancel; this
	// exercises the in-Wait cancellation path. The pre-Wait short-circuit
	// is exercised by TestSubscribeBrokerCancelledContextExitsImmediately
	// below.
	time.Sleep(5 * time.Millisecond)
	cancel()

	select {
	case _, ok := <-ch:
		if ok {
			t.Fatalf("subscriber channel delivered an event after cancel; expected closed channel")
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatalf("subscriber goroutine did not close channel within 100 ms of cancel")
	}
}

// TestSubscribeBrokerEmptyProjectIDReturnsClosedChannel asserts that a
// whitespace-only projectID returns an immediately-closed channel without
// leaking a goroutine.
func TestSubscribeBrokerEmptyProjectIDReturnsClosedChannel(t *testing.T) {
	t.Parallel()

	broker := app.NewInProcessLiveWaitBroker()
	d := &dispatcher{
		svc:    &stubActionItemReader{},
		broker: broker,
		clock:  time.Now,
	}

	ch := d.subscribeBroker(context.Background(), "   ")
	select {
	case _, ok := <-ch:
		if ok {
			t.Fatalf("subscriber channel delivered an event for empty projectID")
		}
	case <-time.After(50 * time.Millisecond):
		t.Fatalf("subscriber channel for empty projectID was not closed within 50 ms")
	}
}

// TestSubscribeBrokerNilDispatcherReturnsClosedChannel asserts a nil-receiver
// call surfaces as a closed channel rather than panicking.
func TestSubscribeBrokerNilDispatcherReturnsClosedChannel(t *testing.T) {
	t.Parallel()

	var d *dispatcher
	ch := d.subscribeBroker(context.Background(), "proj-nil")
	select {
	case _, ok := <-ch:
		if ok {
			t.Fatalf("nil-dispatcher subscriber channel delivered an event")
		}
	case <-time.After(50 * time.Millisecond):
		t.Fatalf("nil-dispatcher subscriber channel was not closed within 50 ms")
	}
}

// TestSubscribeBrokerForwardsMultipleEventsInOrder asserts the re-subscribe
// loop advances its cursor correctly so successive publishes are each
// observed exactly once in monotonic order.
func TestSubscribeBrokerForwardsMultipleEventsInOrder(t *testing.T) {
	t.Parallel()

	broker := app.NewInProcessLiveWaitBroker()
	d := &dispatcher{
		svc:    &stubActionItemReader{},
		broker: broker,
		clock:  time.Now,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch := d.subscribeBroker(ctx, "proj-multi")

	const total = 3
	go func() {
		for i := 0; i < total; i++ {
			// Spread publishes so each has time to be observed +
			// re-registered before the next; this verifies the cursor
			// advances rather than dropping the second/third event.
			time.Sleep(5 * time.Millisecond)
			broker.Publish(app.LiveWaitEvent{
				Type:  app.LiveWaitEventActionItemChanged,
				Key:   "proj-multi",
				Value: "proj-multi",
			})
		}
	}()

	deadline := time.After(300 * time.Millisecond)
	var seenSeqs []int64
	for len(seenSeqs) < total {
		select {
		case event, ok := <-ch:
			if !ok {
				t.Fatalf("subscriber channel closed before receiving %d events (got %d)", total, len(seenSeqs))
			}
			seenSeqs = append(seenSeqs, event.Sequence)
		case <-deadline:
			t.Fatalf("timed out collecting %d events; got %d", total, len(seenSeqs))
		}
	}
	for i := 1; i < len(seenSeqs); i++ {
		if seenSeqs[i] <= seenSeqs[i-1] {
			t.Fatalf("subscriber observed non-monotonic sequences: %v", seenSeqs)
		}
	}
}

// TestSubscribeBrokerCancelledContextExitsImmediately asserts that a
// pre-cancelled context produces a closed channel without entering Wait.
// This pins the loop's leading ctx.Err() check so a future refactor that
// drops it surfaces here rather than as a goroutine leak.
func TestSubscribeBrokerCancelledContextExitsImmediately(t *testing.T) {
	t.Parallel()

	broker := app.NewInProcessLiveWaitBroker()
	d := &dispatcher{
		svc:    &stubActionItemReader{},
		broker: broker,
		clock:  time.Now,
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	ch := d.subscribeBroker(ctx, "proj-precancel")
	select {
	case _, ok := <-ch:
		if ok {
			t.Fatalf("subscriber delivered an event with pre-cancelled context")
		}
	case <-time.After(50 * time.Millisecond):
		t.Fatalf("subscriber goroutine did not exit promptly with pre-cancelled context")
	}
}
