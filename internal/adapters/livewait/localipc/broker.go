// Package localipc provides a local cross-process live-wait broker backed by SQLite and loopback delivery.
package localipc

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/hylla/tillsyn/internal/app"
)

// Package-level constants define the durable storage and loopback delivery defaults for the broker.
const (
	liveWaitSubscriptionsTable = "live_wait_subscriptions"
	liveWaitEventsTable        = "live_wait_events"
	defaultWaitTTL             = 24 * time.Hour
	localListenAddr            = "127.0.0.1:0"
	dialTimeout                = 250 * time.Millisecond
)

// errBrokerClosed reports that the broker has been closed.
var errBrokerClosed = errors.New("live wait broker is closed")

// Config configures one local cross-process live-wait broker.
type Config struct {
	// Clock returns the current wall clock time.
	Clock func() time.Time
	// Secret authenticates wake packets for broker instances in the same runtime.
	Secret string
}

// Broker implements app.LiveWaitBroker with durable SQLite registration and local wake delivery.
type Broker struct {
	db      *sql.DB
	clock   func() time.Time
	secret  string
	addr    string
	ln      net.Listener
	closeCh chan struct{}

	mu      sync.Mutex
	nextID  uint64
	waiters map[subscriptionKey]map[uint64]*localWaiter
	closed  bool
}

// localWaiter stores one active local wait registration and its wake channels.
type localWaiter struct {
	eventCh        chan app.LiveWaitEvent
	doneCh         chan struct{}
	subscriptionID string
}

// subscriptionKey identifies one waiter bucket in both durable and in-memory state.
type subscriptionKey struct {
	eventType app.LiveWaitEventType
	key       string
}

// subscriptionRow stores one durable subscription row paired with one callback address.
type subscriptionRow struct {
	id          string
	callbackURL string
}

// wakePacket carries one authenticated wake delivery across local processes.
type wakePacket struct {
	Secret string            `json:"secret"`
	Event  app.LiveWaitEvent `json:"event"`
}

// NewBroker constructs one local cross-process live-wait broker.
func NewBroker(db *sql.DB, cfg Config) (*Broker, error) {
	if db == nil {
		return nil, fmt.Errorf("live wait sqlite db is required")
	}
	secret := strings.TrimSpace(cfg.Secret)
	if secret == "" {
		return nil, fmt.Errorf("live wait broker secret is required")
	}
	clock := cfg.Clock
	if clock == nil {
		clock = time.Now
	}
	if err := ensureSchema(context.Background(), db); err != nil {
		return nil, err
	}
	ln, err := net.Listen("tcp", localListenAddr)
	if err != nil {
		return nil, fmt.Errorf("listen for live wait delivery: %w", err)
	}
	b := &Broker{
		db:      db,
		clock:   clock,
		secret:  secret,
		addr:    ln.Addr().String(),
		ln:      ln,
		closeCh: make(chan struct{}),
		waiters: map[subscriptionKey]map[uint64]*localWaiter{},
	}
	go b.acceptLoop(ln)
	return b, nil
}

// Close stops the broker listener, releases local waiters, and deletes durable subscriptions for this broker.
func (b *Broker) Close() error {
	if b == nil {
		return nil
	}
	waiters, callbackURL, alreadyClosed := b.closeWaiters()
	if alreadyClosed {
		return nil
	}
	for _, waiter := range waiters {
		close(waiter.doneCh)
	}
	_ = b.deleteSubscriptionsByCallbackURL(context.Background(), callbackURL)
	ln := b.stopListener()
	if ln != nil {
		return ln.Close()
	}
	return nil
}

// Wait blocks until the requested event is published, the context ends, or the broker closes.
func (b *Broker) Wait(ctx context.Context, eventType app.LiveWaitEventType, key string) (app.LiveWaitEvent, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if b == nil {
		return app.LiveWaitEvent{}, fmt.Errorf("live wait broker is not configured")
	}
	if b.isClosed() {
		return app.LiveWaitEvent{}, errBrokerClosed
	}
	if err := ensureSchema(ctx, b.db); err != nil {
		return app.LiveWaitEvent{}, err
	}
	subKey := subscriptionKey{eventType: eventType, key: key}
	if event, ok, err := b.latestEvent(ctx, subKey); err != nil {
		return app.LiveWaitEvent{}, err
	} else if ok {
		return event, nil
	}

	waiterID, waiter := b.registerLocal(subKey)
	subID, err := b.insertSubscription(ctx, subKey)
	if err != nil {
		b.unregisterLocal(subKey, waiterID)
		return app.LiveWaitEvent{}, err
	}
	waiter.subscriptionID = subID
	defer func() {
		b.unregisterLocal(subKey, waiterID)
		_ = b.deleteSubscription(context.Background(), subID)
	}()

	if b.isClosed() {
		return app.LiveWaitEvent{}, errBrokerClosed
	}
	if event, ok, err := b.latestEvent(ctx, subKey); err != nil {
		return app.LiveWaitEvent{}, err
	} else if ok {
		return event, nil
	}

	select {
	case <-b.closeCh:
		return app.LiveWaitEvent{}, errBrokerClosed
	case <-ctx.Done():
		return app.LiveWaitEvent{}, ctx.Err()
	case <-waiter.doneCh:
		return app.LiveWaitEvent{}, errBrokerClosed
	case event := <-waiter.eventCh:
		return event, nil
	}
}

// Publish stores one durable event and wakes matching local and remote waiters.
func (b *Broker) Publish(event app.LiveWaitEvent) {
	if b == nil {
		return
	}
	if b.isClosed() {
		return
	}
	ctx := context.Background()
	if err := ensureSchema(ctx, b.db); err != nil {
		return
	}
	if err := b.upsertLatestEvent(ctx, event); err != nil {
		return
	}

	b.publishLocal(event)
	rows, err := b.listSubscriptions(ctx, event)
	if err != nil {
		return
	}
	grouped := map[string][]string{}
	for _, row := range rows {
		grouped[row.callbackURL] = append(grouped[row.callbackURL], row.id)
	}
	for callbackURL, ids := range grouped {
		if err := sendWake(callbackURL, b.secret, event); err != nil {
			for _, id := range ids {
				_ = b.deleteSubscription(ctx, id)
			}
		}
	}
}

// acceptLoop receives authenticated wake packets and forwards them to local waiters.
func (b *Broker) acceptLoop(ln net.Listener) {
	for {
		conn, err := ln.Accept()
		if err != nil {
			select {
			case <-b.closeCh:
				return
			default:
			}
			continue
		}
		go b.handleConn(conn)
	}
}

// handleConn decodes one wake packet and rejects packets with the wrong broker secret.
func (b *Broker) handleConn(conn net.Conn) {
	defer conn.Close()
	var packet wakePacket
	if err := json.NewDecoder(conn).Decode(&packet); err != nil {
		return
	}
	if packet.Secret != b.secret {
		return
	}
	b.publishLocal(packet.Event)
}

// registerLocal stores one local waiter for the given event key.
func (b *Broker) registerLocal(key subscriptionKey) (uint64, *localWaiter) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.nextID++
	waiterID := b.nextID
	waiter := &localWaiter{
		eventCh: make(chan app.LiveWaitEvent, 1),
		doneCh:  make(chan struct{}),
	}
	if b.waiters[key] == nil {
		b.waiters[key] = map[uint64]*localWaiter{}
	}
	b.waiters[key][waiterID] = waiter
	return waiterID, waiter
}

// unregisterLocal removes one local waiter from the in-memory map.
func (b *Broker) unregisterLocal(key subscriptionKey, waiterID uint64) {
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

// publishLocal wakes every local waiter registered for one subscription key.
func (b *Broker) publishLocal(event app.LiveWaitEvent) {
	key := subscriptionKey{eventType: event.Type, key: event.Key}
	b.mu.Lock()
	waiters := b.waiters[key]
	if len(waiters) == 0 {
		b.mu.Unlock()
		return
	}
	delete(b.waiters, key)
	out := make([]*localWaiter, 0, len(waiters))
	for _, waiter := range waiters {
		out = append(out, waiter)
	}
	b.mu.Unlock()
	for _, waiter := range out {
		select {
		case waiter.eventCh <- event:
		default:
		}
	}
}

// latestEvent loads one durable latest-event snapshot if it exists.
func (b *Broker) latestEvent(ctx context.Context, key subscriptionKey) (app.LiveWaitEvent, bool, error) {
	var payload string
	err := b.db.QueryRowContext(ctx, `SELECT payload_json FROM live_wait_events WHERE event_type = ? AND key = ?`, string(key.eventType), key.key).Scan(&payload)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return app.LiveWaitEvent{}, false, nil
		}
		return app.LiveWaitEvent{}, false, fmt.Errorf("query latest live wait event: %w", err)
	}
	event, err := decodeEvent(payload)
	if err != nil {
		return app.LiveWaitEvent{}, false, err
	}
	return event, true, nil
}

// upsertLatestEvent persists one durable latest-event snapshot for replay.
func (b *Broker) upsertLatestEvent(ctx context.Context, event app.LiveWaitEvent) error {
	payload, err := encodeEvent(event)
	if err != nil {
		return err
	}
	_, err = b.db.ExecContext(ctx, `
		INSERT INTO live_wait_events(event_type, key, payload_json, created_at)
		VALUES(?, ?, ?, ?)
		ON CONFLICT(event_type, key) DO UPDATE SET
			payload_json = excluded.payload_json,
			created_at = excluded.created_at
	`, string(event.Type), event.Key, payload, b.clock().UTC().Format(time.RFC3339Nano))
	if err != nil {
		return fmt.Errorf("upsert latest live wait event: %w", err)
	}
	return nil
}

// insertSubscription records one durable wait registration for the active broker address.
func (b *Broker) insertSubscription(ctx context.Context, key subscriptionKey) (string, error) {
	subID := newID()
	deadline := b.clock().Add(defaultWaitTTL)
	if dl, ok := ctx.Deadline(); ok {
		deadline = dl
	}
	_, err := b.db.ExecContext(ctx, `
		INSERT INTO live_wait_subscriptions(subscription_id, callback_url, event_type, key, created_at, expires_at)
		VALUES(?, ?, ?, ?, ?, ?)
	`, subID, b.addr, string(key.eventType), key.key, b.clock().UTC().Format(time.RFC3339Nano), deadline.UTC().Format(time.RFC3339Nano))
	if err != nil {
		return "", fmt.Errorf("insert live wait subscription: %w", err)
	}
	return subID, nil
}

// deleteSubscription removes one durable wait registration by id.
func (b *Broker) deleteSubscription(ctx context.Context, subscriptionID string) error {
	if subscriptionID == "" {
		return nil
	}
	if _, err := b.db.ExecContext(ctx, `DELETE FROM live_wait_subscriptions WHERE subscription_id = ?`, subscriptionID); err != nil {
		return fmt.Errorf("delete live wait subscription: %w", err)
	}
	return nil
}

// deleteSubscriptionsByCallbackURL removes every durable registration tied to one broker address.
func (b *Broker) deleteSubscriptionsByCallbackURL(ctx context.Context, callbackURL string) error {
	if strings.TrimSpace(callbackURL) == "" {
		return nil
	}
	if _, err := b.db.ExecContext(ctx, `DELETE FROM live_wait_subscriptions WHERE callback_url = ?`, callbackURL); err != nil {
		return fmt.Errorf("delete live wait subscriptions by callback url: %w", err)
	}
	return nil
}

// listSubscriptions loads all still-active durable registrations for one event key.
func (b *Broker) listSubscriptions(ctx context.Context, event app.LiveWaitEvent) ([]subscriptionRow, error) {
	if _, err := b.db.ExecContext(ctx, `DELETE FROM live_wait_subscriptions WHERE expires_at <= ?`, b.clock().UTC().Format(time.RFC3339Nano)); err != nil {
		return nil, fmt.Errorf("cleanup expired live wait subscriptions: %w", err)
	}
	rows, err := b.db.QueryContext(ctx, `
		SELECT subscription_id, callback_url
		FROM live_wait_subscriptions
		WHERE event_type = ? AND key = ? AND expires_at > ?
	`, string(event.Type), event.Key, b.clock().UTC().Format(time.RFC3339Nano))
	if err != nil {
		return nil, fmt.Errorf("query live wait subscriptions: %w", err)
	}
	defer rows.Close()

	out := make([]subscriptionRow, 0)
	for rows.Next() {
		var row subscriptionRow
		if err := rows.Scan(&row.id, &row.callbackURL); err != nil {
			return nil, fmt.Errorf("scan live wait subscription: %w", err)
		}
		out = append(out, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate live wait subscriptions: %w", err)
	}
	return out, nil
}

// closeWaiters snapshots and clears all local waiters so Close can wake them.
func (b *Broker) closeWaiters() ([]*localWaiter, string, bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return nil, "", true
	}
	b.closed = true
	close(b.closeCh)
	callbackURL := b.addr
	out := make([]*localWaiter, 0)
	for _, bucket := range b.waiters {
		for _, waiter := range bucket {
			out = append(out, waiter)
		}
	}
	b.waiters = map[subscriptionKey]map[uint64]*localWaiter{}
	return out, callbackURL, false
}

// stopListener detaches the listener so Close can shut it down without holding the lock.
func (b *Broker) stopListener() net.Listener {
	b.mu.Lock()
	defer b.mu.Unlock()
	ln := b.ln
	b.ln = nil
	return ln
}

// isClosed reports whether Close has already been called.
func (b *Broker) isClosed() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.closed
}

// ensureSchema creates the live-wait tables if they are not already present.
func ensureSchema(ctx context.Context, db *sql.DB) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS live_wait_subscriptions (
			subscription_id TEXT PRIMARY KEY,
			callback_url TEXT NOT NULL,
			event_type TEXT NOT NULL,
			key TEXT NOT NULL,
			created_at TEXT NOT NULL,
			expires_at TEXT NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS live_wait_events (
			event_type TEXT NOT NULL,
			key TEXT NOT NULL,
			payload_json TEXT NOT NULL,
			created_at TEXT NOT NULL,
			PRIMARY KEY(event_type, key)
		);`,
		`CREATE INDEX IF NOT EXISTS idx_live_wait_subscriptions_event ON live_wait_subscriptions(event_type, key, expires_at, callback_url);`,
	}
	for _, stmt := range stmts {
		if _, err := db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("ensure live wait schema: %w", err)
		}
	}
	return nil
}

// encodeEvent serializes one live-wait event for durable storage or transport.
func encodeEvent(event app.LiveWaitEvent) (string, error) {
	raw, err := json.Marshal(event)
	if err != nil {
		return "", fmt.Errorf("encode live wait event: %w", err)
	}
	return string(raw), nil
}

// decodeEvent reconstructs one live-wait event from its durable payload.
func decodeEvent(payload string) (app.LiveWaitEvent, error) {
	var event app.LiveWaitEvent
	if err := json.Unmarshal([]byte(payload), &event); err != nil {
		return app.LiveWaitEvent{}, fmt.Errorf("decode live wait event: %w", err)
	}
	return event, nil
}

// sendWake delivers one authenticated wake packet to another broker process.
func sendWake(addr, secret string, event app.LiveWaitEvent) error {
	conn, err := net.DialTimeout("tcp", addr, dialTimeout)
	if err != nil {
		return err
	}
	defer conn.Close()
	if err := conn.SetWriteDeadline(time.Now().Add(dialTimeout)); err != nil {
		return err
	}
	return json.NewEncoder(conn).Encode(wakePacket{Secret: secret, Event: event})
}

// newID returns one stable-enough identifier for a durable subscription row.
func newID() string {
	return fmt.Sprintf("livewait-%d", time.Now().UTC().UnixNano())
}
