package app

import (
	"context"
	"fmt"
	"sync"
)

// LiveWaitEventType identifies one live in-process wakeup channel.
type LiveWaitEventType string

// Live-wait event types define the first reusable coordination wakeup channels.
const (
	LiveWaitEventAuthRequestResolved LiveWaitEventType = "auth_request_resolved"
)

// LiveWaitEvent carries one in-process wakeup payload for waiting callers.
type LiveWaitEvent struct {
	Type  LiveWaitEventType
	Key   string
	Value any
}

// LiveWaitBroker provides one reusable in-process wait/publish surface for live coordination.
type LiveWaitBroker interface {
	// Wait blocks until a matching event is published or the context ends.
	Wait(ctx context.Context, eventType LiveWaitEventType, key string) (LiveWaitEvent, error)
	// Publish wakes all callers waiting on the matching event type and key.
	Publish(event LiveWaitEvent)
}

// liveWaitSubscriptionKey identifies one waiter bucket.
type liveWaitSubscriptionKey struct {
	eventType LiveWaitEventType
	key       string
}

// inProcessLiveWaitBroker stores in-memory waiters for one process-local runtime.
type inProcessLiveWaitBroker struct {
	mu      sync.Mutex
	nextID  uint64
	waiters map[liveWaitSubscriptionKey]map[uint64]chan LiveWaitEvent
	latest  map[liveWaitSubscriptionKey]LiveWaitEvent
}

// NewInProcessLiveWaitBroker constructs one reusable in-process wait/publish broker.
func NewInProcessLiveWaitBroker() LiveWaitBroker {
	return &inProcessLiveWaitBroker{
		waiters: map[liveWaitSubscriptionKey]map[uint64]chan LiveWaitEvent{},
		latest:  map[liveWaitSubscriptionKey]LiveWaitEvent{},
	}
}

// Wait blocks until one matching event is published or the context ends.
func (b *inProcessLiveWaitBroker) Wait(ctx context.Context, eventType LiveWaitEventType, key string) (LiveWaitEvent, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if b == nil {
		return LiveWaitEvent{}, fmt.Errorf("live wait broker is not configured")
	}
	subscriptionKey := liveWaitSubscriptionKey{
		eventType: eventType,
		key:       key,
	}
	waiterID, ch, event, resolved := b.register(subscriptionKey)
	if resolved {
		return event, nil
	}
	defer b.unregister(subscriptionKey, waiterID)

	select {
	case <-ctx.Done():
		return LiveWaitEvent{}, ctx.Err()
	case event := <-ch:
		return event, nil
	}
}

// Publish wakes every waiter registered for the same type/key pair.
func (b *inProcessLiveWaitBroker) Publish(event LiveWaitEvent) {
	if b == nil {
		return
	}
	subscriptionKey := liveWaitSubscriptionKey{
		eventType: event.Type,
		key:       event.Key,
	}
	waiters := b.take(subscriptionKey, event)
	for _, ch := range waiters {
		select {
		case ch <- event:
		default:
		}
	}
}

// register stores one waiter channel and returns its stable id.
func (b *inProcessLiveWaitBroker) register(key liveWaitSubscriptionKey) (uint64, chan LiveWaitEvent, LiveWaitEvent, bool) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if event, ok := b.latest[key]; ok {
		return 0, nil, event, true
	}
	b.nextID++
	waiterID := b.nextID
	ch := make(chan LiveWaitEvent, 1)
	if b.waiters[key] == nil {
		b.waiters[key] = map[uint64]chan LiveWaitEvent{}
	}
	b.waiters[key][waiterID] = ch
	return waiterID, ch, LiveWaitEvent{}, false
}

// unregister removes one waiter if it is still present.
func (b *inProcessLiveWaitBroker) unregister(key liveWaitSubscriptionKey, waiterID uint64) {
	b.mu.Lock()
	defer b.mu.Unlock()

	waiters := b.waiters[key]
	if waiters == nil {
		return
	}
	delete(waiters, waiterID)
	if len(waiters) == 0 {
		delete(b.waiters, key)
	}
}

// take removes and returns all waiters for one subscription key.
func (b *inProcessLiveWaitBroker) take(key liveWaitSubscriptionKey, event LiveWaitEvent) []chan LiveWaitEvent {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.latest[key] = event
	waiters := b.waiters[key]
	if len(waiters) == 0 {
		return nil
	}
	delete(b.waiters, key)
	out := make([]chan LiveWaitEvent, 0, len(waiters))
	for _, ch := range waiters {
		out = append(out, ch)
	}
	return out
}
