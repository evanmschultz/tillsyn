package dispatcher

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/evanmschultz/tillsyn/internal/domain"
)

// handshake.go ships the TUI permission handshake for spawn-time tool
// denials.
//
// Drop 4c F.7-CORE F.7.5b. Per master PLAN.md L10 the permission-denied
// handshake fires AT THE TERMINAL EVENT (not real-time mid-stream): when a
// CLI spawn finishes and its terminal event carries a non-empty
// permission_denials[] slice, the dispatcher posts one Tillsyn attention item
// per denial so the dev can approve "allow once" / "allow always" / "deny"
// via the TUI. Approved-always entries persist to the permission_grants
// table substrate landed in F.7.17.7 — that wiring is a separate later
// droplet (F.7.5c) and is NOT this droplet's concern.
//
// PLAN reference: workflow/drop_4c/PLAN.md §6.1 / F.7-CORE F.7.5 split
// policy records the F.7.5 split: F.7.5a (permission_grants table) absorbed
// into F.7.17.7 already-merged; F.7.5b (this file) ships the TUI handshake
// type; F.7.5c (settings.json grant injection) is a separate later droplet.
//
// Wiring deferral: PostDenials is the consumer-facing primitive. Calling it
// from the dispatcher monitor's terminal-event hook (so denials surfaced
// from adapter.ExtractTerminalReport flow into attention items) is a
// follow-up for the dispatcher orchestration layer and lives outside this
// droplet to keep cross-droplet ordering concerns isolated.

// AttentionItemStore is the dispatcher-local port for posting attention items
// during the permission handshake. Production wiring binds this to the
// existing internal/app attention service; tests inject a mock.
//
// The port is intentionally minimal: handshake-time attention posting only
// needs Create. Listing / acknowledging / resolving lives in the broader
// attention service and is irrelevant here.
type AttentionItemStore interface {
	// Create persists one attention item and returns the stored row. The
	// returned AttentionItem MAY differ from the input (e.g. server-stamped
	// CreatedAt). Implementations MUST treat input ID as authoritative.
	Create(ctx context.Context, item domain.AttentionItem) (domain.AttentionItem, error)
}

// PermissionGrantsStore is the dispatcher-local port for reading and writing
// previously approved permission grants. Production wiring binds this to the
// permission_grants substrate landed in F.7.17.7; the handshake itself does
// not write grants — the TUI does after dev approval — but the field lives
// on PermissionHandshake so future droplets (F.7.5c grant-injection wiring,
// re-prompt suppression) can compose against the same struct without churn.
//
// Today this port has no methods on PermissionHandshake's behalf. The empty
// interface is a deliberate placeholder; future droplets fill it in. Tests
// and production code may pass nil where the field is unused.
type PermissionGrantsStore interface{}

// PermissionHandshake processes a spawn's terminal-report permission_denials
// and posts one attention item per denial so the dev can approve / deny via
// the TUI.
//
// AttentionStore is required for PostDenials to function; passing nil yields
// a nil-deref panic on first denial. GrantsStore is reserved for downstream
// droplets and may be nil today.
type PermissionHandshake struct {
	// AttentionStore posts attention items. Required.
	AttentionStore AttentionItemStore

	// GrantsStore reads/writes permission grants. Reserved for F.7.5c
	// grant-injection wiring; safe to leave nil in this droplet's scope.
	GrantsStore PermissionGrantsStore

	// Now is the clock used for attention-item CreatedAt stamping. Tests
	// inject a fixed time.Time; production leaves it nil, which falls
	// through to time.Now().UTC().
	Now func() time.Time
}

// now returns the handshake's clock value (h.Now if set, else time.Now), in
// UTC. domain.NewAttentionItem normalizes to UTC again internally — this
// helper just keeps the call site terse.
func (h *PermissionHandshake) now() time.Time {
	if h.Now != nil {
		return h.Now()
	}
	return time.Now()
}

// permissionDenialPayload is the structured metadata embedded in each
// attention item's BodyMarkdown so the TUI can route the approve/deny flow
// without re-decoding the original terminal event. Per workflow/drop_4c/PLAN.md
// §6.1 F.7.5 acceptance criteria the payload carries tool_name + tool_input
// + kind + cli_kind + action_item_id; project_id rides for scope context.
//
// Field roles:
//   - tool_name / tool_input: what was denied (TUI surfaces these to the dev).
//   - project_id: scope context (mirrors the AttentionItem's ScopeID).
//   - kind: action_item kind that triggered the spawn.
//   - cli_kind: which CLI's settings.json the F.7.5c grant injector targets
//     when the dev approves "always". Required so cross-CLI grant misuse
//     (e.g. a Codex-scoped grant landing in Claude's settings) is not
//     possible.
//   - action_item_id: lets the deny-handler (and future approve handler)
//     locate the action item driving the spawn so it can be moved to
//     `failed` with `metadata.failure_reason = "permission_denied"`.
//
// Adding fields later is non-breaking because the consumer is in-tree.
type permissionDenialPayload struct {
	ToolName     string          `json:"tool_name"`
	ToolInput    json.RawMessage `json:"tool_input"`
	ProjectID    string          `json:"project_id"`
	Kind         domain.Kind     `json:"kind"`
	CLIKind      string          `json:"cli_kind"`
	ActionItemID uuid.UUID       `json:"action_item_id"`
}

// PostDenials creates one attention item per ToolDenial and returns the
// generated attention-item IDs. Empty denials is a no-op and returns
// (nil, nil) — handshake on a clean spawn must never error.
//
// Failure mode: if any single denial fails to post (JSON marshal error,
// AttentionStore.Create error, domain validation error), PostDenials
// continues processing the remaining denials and aggregates every error via
// errors.Join. The successful IDs slice and the aggregated error are both
// returned so the caller has full visibility into partial success — this is
// load-bearing for the test "1+3 succeed, 2nd fails" case where the caller
// must see both the two successful IDs and the second-call error.
//
// Per acceptance "Failure on one denial doesn't halt remaining" — the loop
// never short-circuits.
//
// Per the F.7.17.7 PermissionGrant ID convention, attention-item IDs are
// stored in the domain layer as strings. PostDenials generates fresh
// uuid.UUID values, persists them via the string conversion the domain
// layer expects, and returns the typed uuid.UUID slice the spawn-prompt
// signature requires.
func (h *PermissionHandshake) PostDenials(
	ctx context.Context,
	projectID uuid.UUID,
	kind domain.Kind,
	cliKind string,
	actionItemID uuid.UUID,
	denials []ToolDenial,
) ([]uuid.UUID, error) {
	if len(denials) == 0 {
		return nil, nil
	}

	projectIDStr := projectID.String()
	createdIDs := make([]uuid.UUID, 0, len(denials))
	var errs []error

	for i, denial := range denials {
		id, err := h.postOne(ctx, projectIDStr, kind, cliKind, actionItemID, denial)
		if err != nil {
			errs = append(errs, fmt.Errorf("denial %d (tool=%q): %w", i, denial.ToolName, err))
			continue
		}
		createdIDs = append(createdIDs, id)
	}

	return createdIDs, errors.Join(errs...)
}

// postOne handles one denial: marshal payload, build the AttentionItem,
// persist via AttentionStore. Returns the generated uuid.UUID on success.
//
// Split out so the per-denial loop in PostDenials reads as a flat
// success/error accumulator rather than a deeply nested expression.
func (h *PermissionHandshake) postOne(
	ctx context.Context,
	projectIDStr string,
	kind domain.Kind,
	cliKind string,
	actionItemID uuid.UUID,
	denial ToolDenial,
) (uuid.UUID, error) {
	payload := permissionDenialPayload{
		ToolName:     denial.ToolName,
		ToolInput:    denial.ToolInput,
		ProjectID:    projectIDStr,
		Kind:         kind,
		CLIKind:      cliKind,
		ActionItemID: actionItemID,
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return uuid.Nil, fmt.Errorf("marshal denial payload: %w", err)
	}

	id := uuid.New()
	summary := fmt.Sprintf("Tool permission denied: %s", denial.ToolName)

	item, err := domain.NewAttentionItem(domain.AttentionItemInput{
		ID:                 id.String(),
		ProjectID:          projectIDStr,
		ScopeType:          domain.ScopeLevelProject,
		ScopeID:            projectIDStr,
		State:              domain.AttentionStateOpen,
		Kind:               domain.AttentionKindApprovalRequired,
		Summary:            summary,
		BodyMarkdown:       string(payloadBytes),
		RequiresUserAction: true,
		CreatedByActor:     "tillsyn-dispatcher",
		CreatedByType:      domain.ActorTypeAgent,
	}, h.now())
	if err != nil {
		return uuid.Nil, fmt.Errorf("build attention item: %w", err)
	}

	if _, err := h.AttentionStore.Create(ctx, item); err != nil {
		return uuid.Nil, fmt.Errorf("persist attention item: %w", err)
	}
	return id, nil
}
