package dispatcher

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/evanmschultz/tillsyn/internal/domain"
)

// fakeAttentionStore is a programmable AttentionItemStore mock used by the
// PermissionHandshake tests. Each Create call records the item it received
// and returns the configured (item, error) pair for that call index. If
// callResults is shorter than the number of Create calls, additional calls
// return (item, nil) — a permissive default that keeps "all-success" tests
// terse.
type fakeAttentionStore struct {
	mu          sync.Mutex
	received    []domain.AttentionItem
	callResults []fakeAttentionCallResult
}

// fakeAttentionCallResult tells fakeAttentionStore what to return on a given
// Create call. When err is non-nil the stored item is still recorded — this
// lets test assertions inspect what the handshake attempted to persist even
// on failure paths.
type fakeAttentionCallResult struct {
	err error
}

// Create implements AttentionItemStore. Records the input, returns the
// configured (item, err) pair.
func (f *fakeAttentionStore) Create(ctx context.Context, item domain.AttentionItem) (domain.AttentionItem, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	idx := len(f.received)
	f.received = append(f.received, item)
	if idx < len(f.callResults) {
		return item, f.callResults[idx].err
	}
	return item, nil
}

// snapshot returns a copy of the recorded items so tests can read the slice
// without racing with future Create calls (no live calls in these tests, but
// the discipline avoids surprises if the suite adds parallelism).
func (f *fakeAttentionStore) snapshot() []domain.AttentionItem {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]domain.AttentionItem, len(f.received))
	copy(out, f.received)
	return out
}

// fixedClock returns a function suitable for PermissionHandshake.Now that
// always reports the supplied time. UTC-stable timestamps make
// CreatedAt-equality assertions deterministic.
func fixedClock(t time.Time) func() time.Time {
	return func() time.Time { return t }
}

// TestPermissionHandshakePostDenialsEmpty verifies that an empty denials
// slice is a no-op: no Create calls, no error, no IDs returned. This is the
// hot path for clean spawns and must not error.
func TestPermissionHandshakePostDenialsEmpty(t *testing.T) {
	t.Parallel()

	store := &fakeAttentionStore{}
	h := &PermissionHandshake{
		AttentionStore: store,
		Now:            fixedClock(time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)),
	}

	ids, err := h.PostDenials(context.Background(), uuid.New(), domain.KindBuild, "claude", uuid.New(), nil)
	if err != nil {
		t.Fatalf("expected nil error on empty denials, got %v", err)
	}
	if ids != nil {
		t.Fatalf("expected nil IDs on empty denials, got %v", ids)
	}
	if got := store.snapshot(); len(got) != 0 {
		t.Fatalf("expected zero Create calls, got %d", len(got))
	}

	// Empty-slice (not nil) variant — same behavior expected.
	ids, err = h.PostDenials(context.Background(), uuid.New(), domain.KindBuild, "claude", uuid.New(), []ToolDenial{})
	if err != nil {
		t.Fatalf("expected nil error on empty-slice denials, got %v", err)
	}
	if ids != nil {
		t.Fatalf("expected nil IDs on empty-slice denials, got %v", ids)
	}
}

// TestPermissionHandshakePostDenialsAllSucceed verifies the happy path with
// 3 denials: all 3 attention items created, 3 IDs returned, each item
// carries the denial's tool_name + tool_input + project_id + kind in
// BodyMarkdown, and the IDs are unique.
func TestPermissionHandshakePostDenialsAllSucceed(t *testing.T) {
	t.Parallel()

	store := &fakeAttentionStore{}
	clock := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	h := &PermissionHandshake{
		AttentionStore: store,
		Now:            fixedClock(clock),
	}

	projectID := uuid.New()
	actionItemID := uuid.New()
	denials := []ToolDenial{
		{ToolName: "Bash", ToolInput: json.RawMessage(`{"cmd":"rm -rf /"}`)},
		{ToolName: "Edit", ToolInput: json.RawMessage(`{"path":"/etc/hosts"}`)},
		{ToolName: "WebFetch", ToolInput: json.RawMessage(`{"url":"http://x"}`)},
	}

	ids, err := h.PostDenials(context.Background(), projectID, domain.KindBuild, "claude", actionItemID, denials)
	if err != nil {
		t.Fatalf("expected nil error on all-success path, got %v", err)
	}
	if len(ids) != 3 {
		t.Fatalf("expected 3 IDs, got %d", len(ids))
	}

	// IDs must be unique — uuid.New collisions are astronomically unlikely
	// but the assertion guards against accidental ID reuse in postOne.
	seen := map[uuid.UUID]struct{}{}
	for _, id := range ids {
		if _, dup := seen[id]; dup {
			t.Fatalf("duplicate ID %s in returned slice", id)
		}
		seen[id] = struct{}{}
	}

	created := store.snapshot()
	if len(created) != 3 {
		t.Fatalf("expected 3 Create calls, got %d", len(created))
	}

	for i, item := range created {
		// The returned ID at index i must match the persisted item's ID
		// (the handshake hands the same uuid to both the domain ctor and
		// the returned slice).
		if item.ID != ids[i].String() {
			t.Errorf("item[%d] ID %q does not match returned ID %s", i, item.ID, ids[i])
		}
		if item.ProjectID != projectID.String() {
			t.Errorf("item[%d] ProjectID = %q, want %q", i, item.ProjectID, projectID.String())
		}
		if item.Kind != domain.AttentionKindApprovalRequired {
			t.Errorf("item[%d] Kind = %q, want %q", i, item.Kind, domain.AttentionKindApprovalRequired)
		}
		if item.State != domain.AttentionStateOpen {
			t.Errorf("item[%d] State = %q, want %q", i, item.State, domain.AttentionStateOpen)
		}
		if !item.RequiresUserAction {
			t.Errorf("item[%d] RequiresUserAction = false, want true", i)
		}
		// Decode the BodyMarkdown JSON payload and verify all four fields.
		var payload permissionDenialPayload
		if err := json.Unmarshal([]byte(item.BodyMarkdown), &payload); err != nil {
			t.Fatalf("item[%d] BodyMarkdown is not valid JSON: %v (body=%q)", i, err, item.BodyMarkdown)
		}
		if payload.ToolName != denials[i].ToolName {
			t.Errorf("item[%d] payload.ToolName = %q, want %q", i, payload.ToolName, denials[i].ToolName)
		}
		if string(payload.ToolInput) != string(denials[i].ToolInput) {
			t.Errorf("item[%d] payload.ToolInput = %q, want %q", i, payload.ToolInput, denials[i].ToolInput)
		}
		if payload.ProjectID != projectID.String() {
			t.Errorf("item[%d] payload.ProjectID = %q, want %q", i, payload.ProjectID, projectID.String())
		}
		if payload.Kind != domain.KindBuild {
			t.Errorf("item[%d] payload.Kind = %q, want %q", i, payload.Kind, domain.KindBuild)
		}
		if payload.CLIKind != "claude" {
			t.Errorf("item[%d] payload.CLIKind = %q, want %q", i, payload.CLIKind, "claude")
		}
		if payload.ActionItemID != actionItemID {
			t.Errorf("item[%d] payload.ActionItemID = %s, want %s", i, payload.ActionItemID, actionItemID)
		}
		// Summary surfaces the tool name so the dev sees what's being
		// asked for at the inbox glance level.
		wantSummary := fmt.Sprintf("Tool permission denied: %s", denials[i].ToolName)
		if item.Summary != wantSummary {
			t.Errorf("item[%d] Summary = %q, want %q", i, item.Summary, wantSummary)
		}
	}
}

// TestPermissionHandshakePostDenialsContinuesAfterFailure verifies the
// load-bearing partial-failure contract: when the 2nd Create call fails,
// PostDenials still attempts the 3rd, returns 2 successful IDs (1st + 3rd),
// and aggregates the 2nd-call error via errors.Join. Acceptance criterion:
// "Mock store returns error on 2nd call → PostDenials returns aggregated
// error containing both successes (1+3) AND the 2nd-call error."
func TestPermissionHandshakePostDenialsContinuesAfterFailure(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("simulated persist failure")
	store := &fakeAttentionStore{
		callResults: []fakeAttentionCallResult{
			{err: nil},     // 1st: success
			{err: wantErr}, // 2nd: failure
			{err: nil},     // 3rd: success
		},
	}
	h := &PermissionHandshake{
		AttentionStore: store,
		Now:            fixedClock(time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)),
	}

	denials := []ToolDenial{
		{ToolName: "Bash", ToolInput: json.RawMessage(`{"cmd":"ls"}`)},
		{ToolName: "Edit", ToolInput: json.RawMessage(`{"path":"x"}`)},
		{ToolName: "WebFetch", ToolInput: json.RawMessage(`{"url":"y"}`)},
	}

	ids, err := h.PostDenials(context.Background(), uuid.New(), domain.KindBuild, "claude", uuid.New(), denials)

	// Aggregated error must surface the 2nd-call failure.
	if err == nil {
		t.Fatalf("expected aggregated error, got nil")
	}
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected aggregated error to wrap %v, got %v", wantErr, err)
	}

	// The successful 1st + 3rd IDs must come back. The failure is at
	// index 1, so the survivors are 0 and 2 — two IDs total.
	if len(ids) != 2 {
		t.Fatalf("expected 2 surviving IDs (indexes 0 and 2), got %d", len(ids))
	}

	// All three Create calls must have been attempted (the loop must not
	// short-circuit on failure).
	created := store.snapshot()
	if len(created) != 3 {
		t.Fatalf("expected 3 Create attempts, got %d", len(created))
	}

	// The aggregated error message must include the per-denial context
	// (tool name) so multi-failure reports stay attributable.
	if msg := err.Error(); !strings.Contains(msg, `tool="Edit"`) {
		t.Errorf("aggregated error %q does not mention failing tool name", msg)
	}
}

// TestPermissionHandshakePostDenialsAggregatesMultipleFailures verifies
// errors.Join correctly composes when more than one denial fails. Required
// for falsification cover: a single-failure test alone leaves
// aggregation unverified.
func TestPermissionHandshakePostDenialsAggregatesMultipleFailures(t *testing.T) {
	t.Parallel()

	err1 := errors.New("first failure")
	err2 := errors.New("second failure")
	store := &fakeAttentionStore{
		callResults: []fakeAttentionCallResult{
			{err: err1},
			{err: err2},
		},
	}
	h := &PermissionHandshake{
		AttentionStore: store,
		Now:            fixedClock(time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)),
	}

	denials := []ToolDenial{
		{ToolName: "Bash", ToolInput: json.RawMessage(`{}`)},
		{ToolName: "Edit", ToolInput: json.RawMessage(`{}`)},
	}

	ids, err := h.PostDenials(context.Background(), uuid.New(), domain.KindBuild, "claude", uuid.New(), denials)

	if err == nil {
		t.Fatalf("expected aggregated error, got nil")
	}
	if !errors.Is(err, err1) {
		t.Errorf("aggregated error does not wrap err1: %v", err)
	}
	if !errors.Is(err, err2) {
		t.Errorf("aggregated error does not wrap err2: %v", err)
	}
	if len(ids) != 0 {
		t.Errorf("expected 0 surviving IDs (both failed), got %d", len(ids))
	}
}

// TestPermissionHandshakeNowDefault verifies the clock-default fallback:
// when PermissionHandshake.Now is nil, postOne uses time.Now (not a panic
// on nil-call). This guards the production path where callers leave the
// clock unset.
func TestPermissionHandshakeNowDefault(t *testing.T) {
	t.Parallel()

	store := &fakeAttentionStore{}
	h := &PermissionHandshake{
		AttentionStore: store,
		// Now intentionally nil.
	}

	denials := []ToolDenial{
		{ToolName: "Bash", ToolInput: json.RawMessage(`{}`)},
	}

	before := time.Now().Add(-time.Second)
	_, err := h.PostDenials(context.Background(), uuid.New(), domain.KindBuild, "claude", uuid.New(), denials)
	after := time.Now().Add(time.Second)

	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	created := store.snapshot()
	if len(created) != 1 {
		t.Fatalf("expected 1 Create call, got %d", len(created))
	}
	got := created[0].CreatedAt
	if got.Before(before) || got.After(after) {
		t.Errorf("CreatedAt %v outside expected window [%v, %v]", got, before, after)
	}
}

// TestPermissionHandshakePostDenialsPayloadIncludesCLIKind verifies the
// CLIKind field is round-tripped through the BodyMarkdown JSON payload. This
// is load-bearing for the F.7.5c grant-injection flow: when the dev approves
// "always", the grant-injector reads cli_kind from the attention item and
// writes the grant into the matching CLI's settings.json. A wrong or missing
// cli_kind reopens the cross-CLI grant-misuse vector.
func TestPermissionHandshakePostDenialsPayloadIncludesCLIKind(t *testing.T) {
	t.Parallel()

	store := &fakeAttentionStore{}
	h := &PermissionHandshake{
		AttentionStore: store,
		Now:            fixedClock(time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)),
	}

	denials := []ToolDenial{
		{ToolName: "Bash", ToolInput: json.RawMessage(`{}`)},
	}

	const wantCLIKind = "codex"
	_, err := h.PostDenials(context.Background(), uuid.New(), domain.KindBuild, wantCLIKind, uuid.New(), denials)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	created := store.snapshot()
	if len(created) != 1 {
		t.Fatalf("expected 1 Create call, got %d", len(created))
	}

	var payload permissionDenialPayload
	if err := json.Unmarshal([]byte(created[0].BodyMarkdown), &payload); err != nil {
		t.Fatalf("BodyMarkdown is not valid JSON: %v (body=%q)", err, created[0].BodyMarkdown)
	}
	if payload.CLIKind != wantCLIKind {
		t.Errorf("payload.CLIKind = %q, want %q", payload.CLIKind, wantCLIKind)
	}

	// Verify the JSON tag is `cli_kind` (not `CLIKind` / `cliKind`). The
	// TUI consumer pattern-matches on the wire-format key; a tag drift
	// here breaks downstream silently.
	var raw map[string]any
	if err := json.Unmarshal([]byte(created[0].BodyMarkdown), &raw); err != nil {
		t.Fatalf("BodyMarkdown is not a JSON object: %v", err)
	}
	if _, ok := raw["cli_kind"]; !ok {
		t.Errorf("payload JSON missing key %q; got keys=%v", "cli_kind", keysOf(raw))
	}
}

// TestPermissionHandshakePostDenialsPayloadIncludesActionItemID verifies the
// ActionItemID field is round-tripped through the BodyMarkdown JSON payload.
// This is load-bearing for the deny-flow: when the dev clicks Deny, the
// dispatcher's deny-handler reads action_item_id from the attention item and
// moves the action item to `failed` with `metadata.failure_reason =
// "permission_denied"`. Missing this field strands the action item in
// in_progress on permission denial.
func TestPermissionHandshakePostDenialsPayloadIncludesActionItemID(t *testing.T) {
	t.Parallel()

	store := &fakeAttentionStore{}
	h := &PermissionHandshake{
		AttentionStore: store,
		Now:            fixedClock(time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)),
	}

	denials := []ToolDenial{
		{ToolName: "Bash", ToolInput: json.RawMessage(`{}`)},
	}

	wantActionItemID := uuid.New()
	_, err := h.PostDenials(context.Background(), uuid.New(), domain.KindBuild, "claude", wantActionItemID, denials)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	created := store.snapshot()
	if len(created) != 1 {
		t.Fatalf("expected 1 Create call, got %d", len(created))
	}

	var payload permissionDenialPayload
	if err := json.Unmarshal([]byte(created[0].BodyMarkdown), &payload); err != nil {
		t.Fatalf("BodyMarkdown is not valid JSON: %v (body=%q)", err, created[0].BodyMarkdown)
	}
	if payload.ActionItemID != wantActionItemID {
		t.Errorf("payload.ActionItemID = %s, want %s", payload.ActionItemID, wantActionItemID)
	}

	// Verify the JSON tag is `action_item_id` (not `ActionItemID` /
	// `actionItemID`). Wire-format key drift breaks downstream silently.
	var raw map[string]any
	if err := json.Unmarshal([]byte(created[0].BodyMarkdown), &raw); err != nil {
		t.Fatalf("BodyMarkdown is not a JSON object: %v", err)
	}
	if _, ok := raw["action_item_id"]; !ok {
		t.Errorf("payload JSON missing key %q; got keys=%v", "action_item_id", keysOf(raw))
	}
}

// keysOf returns the sorted keys of a map for use in test failure messages.
// Sorting keeps assertion output deterministic when the JSON unmarshal
// returns keys in indeterminate order.
func keysOf(m map[string]any) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
