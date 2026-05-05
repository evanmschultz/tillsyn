package app

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/evanmschultz/tillsyn/internal/domain"
)

// stubAuthBackend is the deterministic in-package fake for AuthBackend used
// to exercise RevokeSessionForActionItem without standing up the real
// autentauth + sqlite stack. Every method records the call and returns
// canned responses so tests can pin both the request payloads flowing
// through Service AND the error-aggregation path.
type stubAuthBackend struct {
	mu sync.Mutex

	// listSessions is the canned response for ListAuthSessions.
	listSessions []AuthSession
	// listErr, when non-nil, is returned as the ListAuthSessions error.
	listErr error

	// revokeOverrides maps session IDs to canned errors for RevokeAuthSession.
	// A session ID not present in the map returns nil (success).
	revokeOverrides map[string]error
	// revokeCalls records each (sessionID, reason) tuple in call order so
	// tests can assert iteration order + reason payload.
	revokeCalls []stubRevokeCall
}

// stubRevokeCall captures one RevokeAuthSession invocation.
type stubRevokeCall struct {
	SessionID string
	Reason    string
}

// IssueAuthSession is unimplemented in the stub — RevokeSessionForActionItem
// does not exercise issue.
func (s *stubAuthBackend) IssueAuthSession(_ context.Context, _ AuthSessionIssueInput) (IssuedAuthSession, error) {
	return IssuedAuthSession{}, errors.New("stubAuthBackend: IssueAuthSession not implemented")
}

// ListAuthSessions returns the canned slice + canned error.
func (s *stubAuthBackend) ListAuthSessions(_ context.Context, _ AuthSessionFilter) ([]AuthSession, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.listErr != nil {
		return nil, s.listErr
	}
	out := make([]AuthSession, len(s.listSessions))
	copy(out, s.listSessions)
	return out, nil
}

// ValidateAuthSession is unimplemented in the stub — RevokeSessionForActionItem
// does not exercise validate.
func (s *stubAuthBackend) ValidateAuthSession(_ context.Context, _, _ string) (ValidatedAuthSession, error) {
	return ValidatedAuthSession{}, errors.New("stubAuthBackend: ValidateAuthSession not implemented")
}

// RevokeAuthSession records the call and returns the canned error (if any).
// On success it also flips the matching listSessions entry to mark it
// revoked so subsequent ListAuthSessions reflects the revoked state.
func (s *stubAuthBackend) RevokeAuthSession(_ context.Context, sessionID, reason string) (AuthSession, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.revokeCalls = append(s.revokeCalls, stubRevokeCall{SessionID: sessionID, Reason: reason})
	if err, ok := s.revokeOverrides[sessionID]; ok && err != nil {
		return AuthSession{}, err
	}
	for idx := range s.listSessions {
		if s.listSessions[idx].SessionID != sessionID {
			continue
		}
		now := time.Now().UTC()
		s.listSessions[idx].RevokedAt = &now
		s.listSessions[idx].RevocationReason = reason
		return s.listSessions[idx], nil
	}
	return AuthSession{}, nil
}

// gotRevokeCalls returns a copy of the recorded revoke calls.
func (s *stubAuthBackend) gotRevokeCalls() []stubRevokeCall {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]stubRevokeCall, len(s.revokeCalls))
	copy(out, s.revokeCalls)
	return out
}

// makeBranchScopedSession returns one session whose ApprovedPath resolves
// to a (project, branch) tuple keyed by the action item's ID. Mirrors the
// pre-Drop-2 auth-path branch quirk recorded in the orchestrator's memory:
// drop-scoped auth uses /branch/<id> even though level_1 drops are
// kind=task, scope=task. RevokeSessionForActionItem must match against
// path.ScopeID (the BranchID after Normalize), not against any stored
// project_id field.
func makeBranchScopedSession(sessionID, projectID, actionItemID string) AuthSession {
	return AuthSession{
		SessionID:    sessionID,
		ProjectID:    projectID,
		ApprovedPath: "project/" + projectID + "/branch/" + actionItemID,
	}
}

// makeProjectScopedSession returns one project-only session; its ScopeID
// after Normalize is the project_id, NOT an action_item ID. The method
// must skip these sessions when revoking for an action item.
func makeProjectScopedSession(sessionID, projectID string) AuthSession {
	return AuthSession{
		SessionID:    sessionID,
		ProjectID:    projectID,
		ApprovedPath: "project/" + projectID,
	}
}

// newRevokeServiceFixture wires *Service against the in-memory fakeRepo +
// stub auth backend. The service uses a frozen clock (the lease-revoke
// path requires a deterministic now-value) and a no-op id generator.
func newRevokeServiceFixture(t *testing.T, sessions []AuthSession, revokeOverrides map[string]error) (*Service, *fakeRepo, *stubAuthBackend) {
	t.Helper()
	repo := newFakeRepo()
	auth := &stubAuthBackend{
		listSessions:    sessions,
		revokeOverrides: revokeOverrides,
	}
	svc := NewService(repo, func() string { return "" }, func() time.Time {
		return time.Date(2026, 5, 4, 12, 0, 0, 0, time.UTC)
	}, ServiceConfig{
		AuthBackend: auth,
	})
	return svc, repo, auth
}

// TestRevokeSessionForActionItemRevokesAllSessions verifies the multi-session
// iteration contract from WC-A2: when retries / fix-builder cycles leave
// multiple sessions tied to the same action item, RevokeSessionForActionItem
// MUST iterate over every match and revoke each. Returning early on the
// first match would leak the rest, which would still validate against the
// ApprovedPath until their TTL expired.
func TestRevokeSessionForActionItemRevokesAllSessions(t *testing.T) {
	t.Parallel()

	const actionItemID = "ai-multi"
	const projectID = "proj-multi"

	sessions := []AuthSession{
		makeBranchScopedSession("sess-1", projectID, actionItemID),
		makeBranchScopedSession("sess-2", projectID, actionItemID),
		makeBranchScopedSession("sess-3", projectID, actionItemID),
		// A noise session for a sibling action item should NOT be revoked.
		makeBranchScopedSession("sess-other", projectID, "ai-other"),
	}

	svc, _, auth := newRevokeServiceFixture(t, sessions, nil)

	if err := svc.RevokeSessionForActionItem(context.Background(), actionItemID); err != nil {
		t.Fatalf("RevokeSessionForActionItem: %v", err)
	}

	got := auth.gotRevokeCalls()
	if len(got) != 3 {
		t.Fatalf("expected 3 revoke calls (one per matching session), got %d: %v", len(got), got)
	}
	wantSessions := map[string]bool{"sess-1": true, "sess-2": true, "sess-3": true}
	for _, call := range got {
		if !wantSessions[call.SessionID] {
			t.Fatalf("unexpected revoke call for session %q (want one of sess-1/sess-2/sess-3)", call.SessionID)
		}
		if call.Reason != terminalStateCleanupRevokeReason {
			t.Fatalf("revoke call reason for session %q = %q, want %q", call.SessionID, call.Reason, terminalStateCleanupRevokeReason)
		}
	}
	for _, call := range got {
		if call.SessionID == "sess-other" {
			t.Fatalf("sibling session sess-other was revoked but should not have been: %v", got)
		}
	}
}

// TestRevokeSessionForActionItemRevokesLeaseAndSession verifies the lease
// cascade contract from WC-A1: each matching session's revoke is paired
// with a repo.RevokeCapabilityLeasesByScope call scoped to the same
// (project_id, scope_type, scope_id) tuple. RevokeAuthSession does NOT
// cascade to capability leases, so the explicit cascade is load-bearing.
func TestRevokeSessionForActionItemRevokesLeaseAndSession(t *testing.T) {
	t.Parallel()

	const actionItemID = "ai-paired"
	const projectID = "proj-paired"

	sessions := []AuthSession{
		makeBranchScopedSession("sess-paired", projectID, actionItemID),
	}
	svc, repo, auth := newRevokeServiceFixture(t, sessions, nil)

	// Seed a capability lease scoped to the same (project, branch, scope_id)
	// tuple. The fakeRepo's RevokeCapabilityLeasesByScope flips the lease's
	// Revoked* state in-place so we can pin the cascade by checking the
	// post-call state. We construct the struct directly rather than going
	// through domain.NewCapabilityLease because the validation surface
	// (Role / AgentName / token-non-empty) is orthogonal to what this test
	// exercises — the lease only needs to match the (project, scope_type,
	// scope_id) tuple the cascade uses to revoke.
	lease := domain.CapabilityLease{
		InstanceID:  "agent-1",
		LeaseToken:  "token-1",
		AgentName:   "agent-1-name",
		ProjectID:   projectID,
		ScopeType:   domain.CapabilityScopeBranch,
		ScopeID:     actionItemID,
		Role:        domain.CapabilityRoleOrchestrator,
		IssuedAt:    time.Date(2026, 5, 4, 11, 0, 0, 0, time.UTC),
		ExpiresAt:   time.Date(2026, 5, 4, 14, 0, 0, 0, time.UTC),
		HeartbeatAt: time.Date(2026, 5, 4, 11, 0, 0, 0, time.UTC),
	}
	repo.capabilityLeases[lease.InstanceID] = lease

	if err := svc.RevokeSessionForActionItem(context.Background(), actionItemID); err != nil {
		t.Fatalf("RevokeSessionForActionItem: %v", err)
	}

	// Session must have been revoked.
	gotSession := auth.gotRevokeCalls()
	if len(gotSession) != 1 || gotSession[0].SessionID != "sess-paired" {
		t.Fatalf("expected one session revoke for sess-paired, got %v", gotSession)
	}

	// Lease must have been revoked.
	revokedLease, ok := repo.capabilityLeases[lease.InstanceID]
	if !ok {
		t.Fatalf("expected lease %q to remain in repo after revoke", lease.InstanceID)
	}
	if revokedLease.RevokedAt == nil {
		t.Fatalf("expected lease.RevokedAt to be non-nil after RevokeSessionForActionItem")
	}
	if revokedLease.RevokedReason != terminalStateCleanupRevokeReason {
		t.Fatalf("lease.RevokedReason = %q, want %q", revokedLease.RevokedReason, terminalStateCleanupRevokeReason)
	}
}

// TestRevokeSessionForActionItemNoMatchingSessions verifies the idempotent
// no-op path: when zero sessions match the action item ID, the method
// returns nil (no error) so the cleanup hook can fire-and-forget for items
// that never claimed auth (orchestrator-driven creation, persistent /
// human-verify items).
func TestRevokeSessionForActionItemNoMatchingSessions(t *testing.T) {
	t.Parallel()

	sessions := []AuthSession{
		// Project-scoped session — its ScopeID is the project_id, not an
		// action_item ID, so the action-item match must skip it.
		makeProjectScopedSession("sess-project", "proj-x"),
		// A branch-scoped session for a DIFFERENT action item.
		makeBranchScopedSession("sess-other", "proj-x", "ai-other"),
	}
	svc, _, auth := newRevokeServiceFixture(t, sessions, nil)

	if err := svc.RevokeSessionForActionItem(context.Background(), "ai-target"); err != nil {
		t.Fatalf("RevokeSessionForActionItem on no-match: %v", err)
	}

	got := auth.gotRevokeCalls()
	if len(got) != 0 {
		t.Fatalf("expected zero revoke calls for no-match, got %v", got)
	}
}

// TestRevokeSessionForActionItemIteratesPastSessionRevokeError verifies the
// errors.Join aggregation contract: when one session's revoke errors, the
// method MUST still attempt every other matching session and aggregate
// every per-session failure into the return value via errors.Join. Tests
// the load-bearing safety property that a single backend hiccup cannot
// strand the rest of the cascade.
func TestRevokeSessionForActionItemIteratesPastSessionRevokeError(t *testing.T) {
	t.Parallel()

	const actionItemID = "ai-err"
	const projectID = "proj-err"

	sessions := []AuthSession{
		makeBranchScopedSession("sess-fail", projectID, actionItemID),
		makeBranchScopedSession("sess-ok", projectID, actionItemID),
	}
	revokeErr := errors.New("backend exploded")
	revokeOverrides := map[string]error{
		"sess-fail": revokeErr,
	}
	svc, _, auth := newRevokeServiceFixture(t, sessions, revokeOverrides)

	err := svc.RevokeSessionForActionItem(context.Background(), actionItemID)
	if err == nil {
		t.Fatalf("expected aggregated error, got nil")
	}
	if !errors.Is(err, revokeErr) {
		t.Fatalf("expected errors.Is(err, revokeErr), got err=%v", err)
	}

	// The second session must still have been attempted.
	got := auth.gotRevokeCalls()
	if len(got) != 2 {
		t.Fatalf("expected 2 revoke calls (errored + retried sibling), got %d: %v", len(got), got)
	}
	calledSessions := map[string]bool{}
	for _, call := range got {
		calledSessions[call.SessionID] = true
	}
	if !calledSessions["sess-fail"] {
		t.Fatalf("expected sess-fail to have been attempted")
	}
	if !calledSessions["sess-ok"] {
		t.Fatalf("expected sess-ok to have been attempted despite sess-fail error")
	}
}

// TestRevokeSessionForActionItemEmptyIDIsNoop verifies the documented
// degenerate-input contract: empty actionItemID returns nil without
// calling the backend. Mirrors the cleanup hook's empty-ID short-circuit
// so the production wiring composes cleanly.
func TestRevokeSessionForActionItemEmptyIDIsNoop(t *testing.T) {
	t.Parallel()

	svc, _, auth := newRevokeServiceFixture(t, nil, nil)
	if err := svc.RevokeSessionForActionItem(context.Background(), ""); err != nil {
		t.Fatalf("RevokeSessionForActionItem on empty ID: %v", err)
	}
	if err := svc.RevokeSessionForActionItem(context.Background(), "   "); err != nil {
		t.Fatalf("RevokeSessionForActionItem on whitespace ID: %v", err)
	}
	if got := auth.gotRevokeCalls(); len(got) != 0 {
		t.Fatalf("expected zero revoke calls on empty ID, got %v", got)
	}
}

// TestRevokeSessionForActionItemNilAuthBackendIsNoop verifies the documented
// auth-backend-absent contract: when authBackend is nil, the method returns
// nil without erroring. Test fixtures that construct *Service without an
// auth backend (early Drop 4a tests) must not break the cleanup pipeline.
func TestRevokeSessionForActionItemNilAuthBackendIsNoop(t *testing.T) {
	t.Parallel()

	repo := newFakeRepo()
	svc := NewService(repo, func() string { return "" }, time.Now, ServiceConfig{})
	if err := svc.RevokeSessionForActionItem(context.Background(), "any-id"); err != nil {
		t.Fatalf("RevokeSessionForActionItem with nil authBackend: %v", err)
	}
}

// TestRevokeSessionForActionItemSkipsRevokedSessions verifies that already-
// revoked sessions are skipped: the method must NOT re-revoke a session
// whose RevokedAt is non-nil (idempotency under restart / re-fire).
func TestRevokeSessionForActionItemSkipsRevokedSessions(t *testing.T) {
	t.Parallel()

	const actionItemID = "ai-skip"
	const projectID = "proj-skip"

	now := time.Date(2026, 5, 4, 11, 0, 0, 0, time.UTC)
	revoked := makeBranchScopedSession("sess-revoked", projectID, actionItemID)
	revoked.RevokedAt = &now
	active := makeBranchScopedSession("sess-active", projectID, actionItemID)

	svc, _, auth := newRevokeServiceFixture(t, []AuthSession{revoked, active}, nil)

	if err := svc.RevokeSessionForActionItem(context.Background(), actionItemID); err != nil {
		t.Fatalf("RevokeSessionForActionItem: %v", err)
	}

	got := auth.gotRevokeCalls()
	if len(got) != 1 {
		t.Fatalf("expected exactly 1 revoke call (skip already-revoked), got %d: %v", len(got), got)
	}
	if got[0].SessionID != "sess-active" {
		t.Fatalf("expected sess-active to be revoked, got %q", got[0].SessionID)
	}
}

// TestRevokeSessionForActionItemSkipsMalformedApprovedPath verifies that
// a session whose ApprovedPath fails to parse is skipped silently rather
// than aborting the whole cleanup. Malformed paths are an upstream-bug
// surface that is not this method's concern.
func TestRevokeSessionForActionItemSkipsMalformedApprovedPath(t *testing.T) {
	t.Parallel()

	const actionItemID = "ai-malformed-target"

	good := makeBranchScopedSession("sess-good", "proj-x", actionItemID)
	bad := AuthSession{
		SessionID:    "sess-bad",
		ApprovedPath: "this is not a valid auth path",
	}

	svc, _, auth := newRevokeServiceFixture(t, []AuthSession{bad, good}, nil)

	if err := svc.RevokeSessionForActionItem(context.Background(), actionItemID); err != nil {
		t.Fatalf("RevokeSessionForActionItem: %v", err)
	}

	got := auth.gotRevokeCalls()
	if len(got) != 1 || got[0].SessionID != "sess-good" {
		t.Fatalf("expected only sess-good to be revoked (sess-bad has malformed path), got %v", got)
	}
}
