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

	got, err := broker.Wait(context.Background(), LiveWaitEventAuthRequestResolved, "req-1")
	if err != nil {
		t.Fatalf("Wait() error = %v", err)
	}
	if got != want {
		t.Fatalf("Wait() = %#v, want %#v", got, want)
	}
}
