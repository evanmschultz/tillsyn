package app

import (
	"context"
	"testing"
)

// TestInProcessLiveWaitBrokerReplaysLatestEvent verifies a waiter registering after publication still receives the resolved event.
func TestInProcessLiveWaitBrokerReplaysLatestEvent(t *testing.T) {
	broker := NewInProcessLiveWaitBroker()
	want := LiveWaitEvent{
		Type:  LiveWaitEventAuthRequestResolved,
		Key:   "req-1",
		Value: "approved",
	}
	broker.Publish(want)

	got, err := broker.Wait(context.Background(), LiveWaitEventAuthRequestResolved, "req-1", 0)
	if err != nil {
		t.Fatalf("Wait() error = %v", err)
	}
	if got.Type != want.Type || got.Key != want.Key || got.Value != want.Value || got.Sequence != 1 {
		t.Fatalf("Wait() = %#v, want type/key/value preserved with sequence 1", got)
	}
}

// TestInProcessLiveWaitBrokerDoesNotReplayStaleSequence verifies reused keys only wake for newer events.
func TestInProcessLiveWaitBrokerDoesNotReplayStaleSequence(t *testing.T) {
	broker := NewInProcessLiveWaitBroker()
	broker.Publish(LiveWaitEvent{
		Type:  LiveWaitEventCommentChanged,
		Key:   "project-1|project|project-1",
		Value: "project-1|project|project-1",
	})
	latest, ok, err := broker.Latest(context.Background(), LiveWaitEventCommentChanged, "project-1|project|project-1")
	if err != nil {
		t.Fatalf("Latest() error = %v", err)
	}
	if !ok || latest.Sequence != 1 {
		t.Fatalf("Latest() = %#v, %v, want sequence 1", latest, ok)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	errCh := make(chan error, 1)
	eventCh := make(chan LiveWaitEvent, 1)
	go func() {
		got, err := broker.Wait(ctx, LiveWaitEventCommentChanged, "project-1|project|project-1", latest.Sequence)
		if err != nil {
			errCh <- err
			return
		}
		eventCh <- got
	}()

	select {
	case got := <-eventCh:
		t.Fatalf("Wait() returned stale event %#v without a newer publish", got)
	case err := <-errCh:
		t.Fatalf("Wait() error = %v", err)
	default:
	}

	broker.Publish(LiveWaitEvent{
		Type:  LiveWaitEventCommentChanged,
		Key:   "project-1|project|project-1",
		Value: "project-1|project|project-1",
	})

	select {
	case err := <-errCh:
		t.Fatalf("Wait() error = %v", err)
	case got := <-eventCh:
		if got.Sequence != 2 {
			t.Fatalf("Wait() sequence = %d, want 2", got.Sequence)
		}
	}
}
