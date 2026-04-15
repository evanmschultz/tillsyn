package localipc

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net"
	"path/filepath"
	"testing"
	"time"

	"github.com/evanmschultz/tillsyn/internal/adapters/storage/sqlite"
	"github.com/evanmschultz/tillsyn/internal/app"
)

// testBrokerSecret authenticates local test wake packets across broker instances.
const testBrokerSecret = "shared-secret"

// newBrokerDB opens one isolated SQLite database file for broker tests.
func newBrokerDB(t *testing.T, name string) *sqlite.Repository {
	t.Helper()
	repo, err := sqlite.Open(filepath.Join(t.TempDir(), name))
	if err != nil {
		t.Fatalf("sqlite.Open() error = %v", err)
	}
	t.Cleanup(func() {
		_ = repo.Close()
	})
	return repo
}

// newTestBroker constructs one broker with the shared test secret.
func newTestBroker(t *testing.T, db *sql.DB) *Broker {
	t.Helper()
	broker, err := NewBroker(db, Config{Secret: testBrokerSecret})
	if err != nil {
		t.Fatalf("NewBroker() error = %v", err)
	}
	t.Cleanup(func() { _ = broker.Close() })
	return broker
}

// TestBrokerReplaysLatestEvent verifies a later waiter sees the durable latest event immediately.
func TestBrokerReplaysLatestEvent(t *testing.T) {
	repo := newBrokerDB(t, "replay.db")
	broker := newTestBroker(t, repo.DB())

	want := app.LiveWaitEvent{
		Type:  app.LiveWaitEventAuthRequestResolved,
		Key:   "req-1",
		Value: "approved",
	}
	broker.Publish(want)

	got, err := broker.Wait(context.Background(), app.LiveWaitEventAuthRequestResolved, "req-1", 0)
	if err != nil {
		t.Fatalf("Wait() error = %v", err)
	}
	if got.Type != want.Type || got.Key != want.Key || got.Value != want.Value || got.Sequence != 1 {
		t.Fatalf("Wait() event = %#v, want %#v", got, want)
	}
}

// TestBrokerWakesAcrossInstances verifies one broker instance can wake a waiter in another broker instance using the shared DB/runtime.
func TestBrokerWakesAcrossInstances(t *testing.T) {
	repoPath := filepath.Join(t.TempDir(), "cross.db")
	waitRepo, err := sqlite.Open(repoPath)
	if err != nil {
		t.Fatalf("sqlite.Open(wait) error = %v", err)
	}
	t.Cleanup(func() { _ = waitRepo.Close() })

	publishRepo, err := sqlite.Open(repoPath)
	if err != nil {
		t.Fatalf("sqlite.Open(publish) error = %v", err)
	}
	t.Cleanup(func() { _ = publishRepo.Close() })

	waitBroker := newTestBroker(t, waitRepo.DB())
	publishBroker := newTestBroker(t, publishRepo.DB())

	waitDone := make(chan app.LiveWaitEvent, 1)
	waitErr := make(chan error, 1)
	go func() {
		got, err := waitBroker.Wait(context.Background(), app.LiveWaitEventAuthRequestResolved, "req-2", 0)
		if err != nil {
			waitErr <- err
			return
		}
		waitDone <- got
	}()

	waitForSubscription(t, waitRepo.DB(), string(app.LiveWaitEventAuthRequestResolved), "req-2")
	publishBroker.Publish(app.LiveWaitEvent{
		Type:  app.LiveWaitEventAuthRequestResolved,
		Key:   "req-2",
		Value: "denied",
	})

	select {
	case err := <-waitErr:
		t.Fatalf("Wait() error = %v", err)
	case got := <-waitDone:
		if got.Key != "req-2" || got.Value != "denied" {
			t.Fatalf("Wait() event = %#v, want req-2 denied", got)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Wait() did not wake across instances")
	}
}

// TestBrokerCancelsAndCleansSubscription verifies canceled waits remove their durable registration.
func TestBrokerCancelsAndCleansSubscription(t *testing.T) {
	repo := newBrokerDB(t, "cancel.db")
	broker := newTestBroker(t, repo.DB())

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	if _, err := broker.Wait(ctx, app.LiveWaitEventAuthRequestResolved, "req-3", 0); err == nil {
		t.Fatal("Wait() error = nil, want context deadline exceeded")
	}

	var count int
	if err := repo.DB().QueryRowContext(context.Background(), `SELECT count(*) FROM live_wait_subscriptions WHERE event_type = ? AND key = ?`, string(app.LiveWaitEventAuthRequestResolved), "req-3").Scan(&count); err != nil {
		t.Fatalf("query subscription count error = %v", err)
	}
	if count != 0 {
		t.Fatalf("subscription count = %d, want 0 after cancel cleanup", count)
	}
}

// TestBrokerCloseReleasesActiveWaiters verifies Close wakes active local waiters and clears their durable rows.
func TestBrokerCloseReleasesActiveWaiters(t *testing.T) {
	repo := newBrokerDB(t, "close.db")
	broker := newTestBroker(t, repo.DB())

	waitDone := make(chan error, 1)
	go func() {
		_, err := broker.Wait(context.Background(), app.LiveWaitEventAuthRequestResolved, "req-4", 0)
		waitDone <- err
	}()

	waitForSubscription(t, repo.DB(), string(app.LiveWaitEventAuthRequestResolved), "req-4")
	if err := broker.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	select {
	case err := <-waitDone:
		if !errors.Is(err, errBrokerClosed) {
			t.Fatalf("Wait() error = %v, want errBrokerClosed", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Wait() did not release after Close()")
	}

	var count int
	if err := repo.DB().QueryRowContext(context.Background(), `SELECT count(*) FROM live_wait_subscriptions WHERE callback_url = ?`, broker.addr).Scan(&count); err != nil {
		t.Fatalf("query closed subscription count error = %v", err)
	}
	if count != 0 {
		t.Fatalf("subscription count = %d, want 0 after Close()", count)
	}
}

// TestBrokerRejectsWaitAfterClose verifies a closed broker fails new waits immediately.
func TestBrokerRejectsWaitAfterClose(t *testing.T) {
	repo := newBrokerDB(t, "after-close.db")
	broker := newTestBroker(t, repo.DB())
	if err := broker.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if _, err := broker.Wait(context.Background(), app.LiveWaitEventAuthRequestResolved, "req-5", 0); !errors.Is(err, errBrokerClosed) {
		t.Fatalf("Wait() error = %v, want errBrokerClosed", err)
	}
}

// TestBrokerRejectsSpoofedWakePacket verifies an unauthenticated local packet does not release a waiting caller.
func TestBrokerRejectsSpoofedWakePacket(t *testing.T) {
	repo := newBrokerDB(t, "spoof.db")
	broker := newTestBroker(t, repo.DB())

	waitDone := make(chan app.LiveWaitEvent, 1)
	waitErr := make(chan error, 1)
	go func() {
		got, err := broker.Wait(context.Background(), app.LiveWaitEventAuthRequestResolved, "req-6", 0)
		if err != nil {
			waitErr <- err
			return
		}
		waitDone <- got
	}()

	waitForSubscription(t, repo.DB(), string(app.LiveWaitEventAuthRequestResolved), "req-6")
	spoofConn, err := net.DialTimeout("tcp", broker.addr, dialTimeout)
	if err != nil {
		t.Fatalf("net.DialTimeout() error = %v", err)
	}
	if err := json.NewEncoder(spoofConn).Encode(wakePacket{
		Secret: "wrong-secret",
		Event: app.LiveWaitEvent{
			Type:  app.LiveWaitEventAuthRequestResolved,
			Key:   "req-6",
			Value: "spoofed",
		},
	}); err != nil {
		_ = spoofConn.Close()
		t.Fatalf("encode spoof packet error = %v", err)
	}
	_ = spoofConn.Close()

	select {
	case got := <-waitDone:
		t.Fatalf("Wait() woke on spoofed packet: %#v", got)
	case err := <-waitErr:
		t.Fatalf("Wait() error = %v", err)
	case <-time.After(100 * time.Millisecond):
	}

	broker.Publish(app.LiveWaitEvent{
		Type:  app.LiveWaitEventAuthRequestResolved,
		Key:   "req-6",
		Value: "approved",
	})

	select {
	case err := <-waitErr:
		t.Fatalf("Wait() error = %v", err)
	case got := <-waitDone:
		if got.Key != "req-6" || got.Value != "approved" {
			t.Fatalf("Wait() event = %#v, want req-6 approved", got)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Wait() did not wake after authenticated publish")
	}
}

// TestBrokerWaitAfterSequenceIgnoresStaleEvent verifies reused keys only wake on newer events.
func TestBrokerWaitAfterSequenceIgnoresStaleEvent(t *testing.T) {
	repo := newBrokerDB(t, "after-sequence.db")
	broker := newTestBroker(t, repo.DB())

	broker.Publish(app.LiveWaitEvent{
		Type:  app.LiveWaitEventCommentChanged,
		Key:   "project-1|project|project-1",
		Value: "project-1|project|project-1",
	})
	latest, ok, err := broker.Latest(context.Background(), app.LiveWaitEventCommentChanged, "project-1|project|project-1")
	if err != nil {
		t.Fatalf("Latest() error = %v", err)
	}
	if !ok || latest.Sequence != 1 {
		t.Fatalf("Latest() = %#v, %v, want sequence 1", latest, ok)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 75*time.Millisecond)
	defer cancel()
	if _, err := broker.Wait(ctx, app.LiveWaitEventCommentChanged, "project-1|project|project-1", latest.Sequence); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Wait() error = %v, want DeadlineExceeded while no newer event exists", err)
	}

	broker.Publish(app.LiveWaitEvent{
		Type:  app.LiveWaitEventCommentChanged,
		Key:   "project-1|project|project-1",
		Value: "project-1|project|project-1",
	})
	got, err := broker.Wait(context.Background(), app.LiveWaitEventCommentChanged, "project-1|project|project-1", latest.Sequence)
	if err != nil {
		t.Fatalf("Wait() second error = %v", err)
	}
	if got.Sequence != 2 {
		t.Fatalf("Wait() sequence = %d, want 2", got.Sequence)
	}
}

// TestBrokerRemovesDuplicateStaleRows verifies failed delivery removes every row tied to one dead callback address.
func TestBrokerRemovesDuplicateStaleRows(t *testing.T) {
	repo := newBrokerDB(t, "stale.db")
	broker := newTestBroker(t, repo.DB())

	staleAddr := closedLoopbackAddr(t)
	for i, subscriptionID := range []string{"stale-row-1", "stale-row-2"} {
		if _, err := repo.DB().ExecContext(context.Background(), `
			INSERT INTO live_wait_subscriptions(subscription_id, callback_url, event_type, key, created_at, expires_at)
			VALUES(?, ?, ?, ?, ?, ?)
		`, subscriptionID, staleAddr, string(app.LiveWaitEventAuthRequestResolved), "req-7", time.Now().UTC().Format(time.RFC3339Nano), time.Now().UTC().Add(time.Hour).Format(time.RFC3339Nano)); err != nil {
			t.Fatalf("insert stale subscription %d error = %v", i, err)
		}
	}

	broker.Publish(app.LiveWaitEvent{
		Type:  app.LiveWaitEventAuthRequestResolved,
		Key:   "req-7",
		Value: "approved",
	})

	var count int
	if err := repo.DB().QueryRowContext(context.Background(), `SELECT count(*) FROM live_wait_subscriptions WHERE callback_url = ?`, staleAddr).Scan(&count); err != nil {
		t.Fatalf("query stale subscription count error = %v", err)
	}
	if count != 0 {
		t.Fatalf("stale subscription count = %d, want 0 after unreachable cleanup", count)
	}
}

// TestNewIDAtRemainsUniqueWithinSameTick verifies the counter suffix keeps ids unique even when the clock does not advance.
func TestNewIDAtRemainsUniqueWithinSameTick(t *testing.T) {
	liveWaitIDCounter.Store(0)
	sameTick := time.Date(2026, 3, 26, 2, 30, 0, 123456789, time.UTC)

	first := newIDAt(sameTick)
	second := newIDAt(sameTick)
	if first == second {
		t.Fatalf("newIDAt() returned duplicate ids for the same timestamp: %q", first)
	}
}

// closedLoopbackAddr returns one loopback address that is no longer listening.
func closedLoopbackAddr(t *testing.T) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen loopback error = %v", err)
	}
	addr := ln.Addr().String()
	if err := ln.Close(); err != nil {
		t.Fatalf("close loopback listener error = %v", err)
	}
	return addr
}

// waitForSubscription blocks until the expected durable wait registration appears.
func waitForSubscription(t *testing.T, db *sql.DB, eventType, key string) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		var count int
		if err := db.QueryRowContext(context.Background(), `SELECT count(*) FROM live_wait_subscriptions WHERE event_type = ? AND key = ?`, eventType, key).Scan(&count); err != nil {
			t.Fatalf("query live wait subscription count error = %v", err)
		}
		if count > 0 {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("subscription for %s/%s never appeared", eventType, key)
}
