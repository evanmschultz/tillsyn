package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/evanmschultz/tillsyn/internal/domain"
)

const (
	defaultAuthRequestTimeout    = 30 * time.Minute
	defaultRequestedSessionTTL   = 8 * time.Hour
	claimAuthRequestPollInterval = 100 * time.Millisecond
	authRequestNotificationLabel = "auth request"
	// AuthRequestContinuationRequesterClientIDKey stores one requester-bound claim client identifier inside private continuation metadata.
	AuthRequestContinuationRequesterClientIDKey = "_tillsyn_requester_client_id"
)

// AuthBackend defines caller-safe session inventory and lifecycle operations.
type AuthBackend interface {
	IssueAuthSession(context.Context, AuthSessionIssueInput) (IssuedAuthSession, error)
	ListAuthSessions(context.Context, AuthSessionFilter) ([]AuthSession, error)
	ValidateAuthSession(context.Context, string, string) (ValidatedAuthSession, error)
	RevokeAuthSession(context.Context, string, string) (AuthSession, error)
}

// AuthSessionIssueInput captures fields for issuing one direct auth session.
type AuthSessionIssueInput struct {
	PrincipalID   string
	PrincipalType string
	PrincipalName string
	ClientID      string
	ClientType    string
	ClientName    string
	TTL           time.Duration
}

// AuthSessionFilter captures deterministic auth-session list filters.
type AuthSessionFilter struct {
	SessionID   string
	ProjectID   string
	PrincipalID string
	ClientID    string
	State       string
	Limit       int
}

// AuthSession stores one caller-safe auth-session record.
//
// Two-axis principal-type design (Drop 4a Wave 3 W3.1 fix to droplet 4a.24):
//
//   - PrincipalType carries the autent-axis value (closed enum
//     {user, agent, service}). For tillsyn-side "steward" principals the
//     autent IssueSession boundary collapses "steward" → "agent" so the
//     value stored here is "agent", not "steward".
//   - AuthRequestPrincipalType carries the tillsyn-axis value preserved
//     across the autent collapse via session metadata key
//     "auth_request_principal_type". This is the field role gates (e.g.
//     STEWARD cross-subtree exception in checkOrchSelfApprovalGate) MUST
//     read when distinguishing steward from non-steward orch sessions —
//     PrincipalType cannot make that distinction. Mirrors the
//     domain.AuthenticatedCaller.AuthRequestPrincipalType precedent landed
//     in Drop 3 droplet 3.19.
type AuthSession struct {
	SessionID                string
	ProjectID                string
	AuthRequestID            string
	ApprovedPath             string
	PrincipalID              string
	PrincipalType            string
	AuthRequestPrincipalType string
	PrincipalRole            string
	PrincipalName            string
	ClientID                 string
	ClientType               string
	ClientName               string
	IssuedAt                 time.Time
	ExpiresAt                time.Time
	LastValidatedAt          *time.Time
	RevokedAt                *time.Time
	RevocationReason         string
}

// IssuedAuthSession stores one newly issued auth-session bundle plus its secret.
type IssuedAuthSession struct {
	Session AuthSession
	Secret  string
}

// ValidatedAuthSession stores one validated auth-session record.
type ValidatedAuthSession struct {
	Session AuthSession
}

// ApprovedAuthRequestResult carries one resolved auth request plus the issued secret material.
type ApprovedAuthRequestResult struct {
	Request       domain.AuthRequest
	SessionSecret string
}

// ClaimedAuthRequestResult carries one requester-visible auth request plus continuation secret material when available.
type ClaimedAuthRequestResult struct {
	Request       domain.AuthRequest
	SessionSecret string
	Waiting       bool
}

// ApproveAuthRequestInput captures fields for approving one auth request.
//
// Approver-identity fields (Drop 4a Wave 3 W3.1):
//
//   - ApproverPrincipalID, ApproverAgentInstanceID, ApproverLeaseToken,
//     ApproverSessionID together identify the orchestrator approving on
//     behalf of the dev (the orch-self-approval cascade path). When ANY of
//     the four is non-empty, ALL FOUR must be non-empty — partial input is
//     rejected with domain.ErrInvalidID-wrapped error. ApproveAuthRequest
//     loads the approver session via the configured AuthBackend and runs
//     the role / self-approval / scope gate before delegating to the
//     gateway.
//   - All four empty = legacy dev-TUI / system path. Approver-identity gate
//     is bypassed; ApprovedBy + ResolvedType identify the human approver.
//     This is the path used by the till TUI today and by autentauth's own
//     fixture tests; preserving it keeps the W3.1 change additive.
//
// Audit-trail surfacing of the populated approver-identity fields lands in
// W3.3 (column adds + scanner + record fields). W3.1 plumbs the input shape
// only.
type ApproveAuthRequestInput struct {
	RequestID               string
	Path                    string
	SessionTTL              time.Duration
	ResolvedBy              string
	ResolvedType            domain.ActorType
	ResolutionNote          string
	ApproverPrincipalID     string
	ApproverAgentInstanceID string
	ApproverLeaseToken      string
	ApproverSessionID       string
}

// CreateAuthRequestInput captures fields for creating one pre-session auth request.
type CreateAuthRequestInput struct {
	Path                string
	PrincipalID         string
	PrincipalType       string
	PrincipalRole       string
	PrincipalName       string
	ClientID            string
	ClientType          string
	ClientName          string
	RequesterClientID   string
	RequestedSessionTTL time.Duration
	Reason              string
	Continuation        map[string]any
	RequestedBy         string
	RequestedType       domain.ActorType
	Timeout             time.Duration
}

// DenyAuthRequestInput captures fields for denying one auth request.
type DenyAuthRequestInput struct {
	RequestID      string
	ResolvedBy     string
	ResolvedType   domain.ActorType
	ResolutionNote string
}

// CancelAuthRequestInput captures fields for canceling one auth request.
type CancelAuthRequestInput struct {
	RequestID      string
	ResolvedBy     string
	ResolvedType   domain.ActorType
	ResolutionNote string
}

// ClaimAuthRequestInput captures fields for resuming one auth request through continuation metadata.
type ClaimAuthRequestInput struct {
	RequestID   string
	ResumeToken string
	PrincipalID string
	ClientID    string
	WaitTimeout time.Duration
}

// AuthRequestGateway defines the auth-request lifecycle needed by the app service.
type AuthRequestGateway interface {
	CreateAuthRequest(context.Context, domain.AuthRequest) (domain.AuthRequest, error)
	GetAuthRequest(context.Context, string) (domain.AuthRequest, error)
	ListAuthRequests(context.Context, domain.AuthRequestListFilter) ([]domain.AuthRequest, error)
	ApproveAuthRequest(context.Context, ApproveAuthRequestGatewayInput) (ApprovedAuthRequestResult, error)
	ClaimAuthRequest(context.Context, ClaimAuthRequestInput) (ClaimedAuthRequestResult, error)
	DenyAuthRequest(context.Context, string, string, domain.ActorType, string) (domain.AuthRequest, error)
	CancelAuthRequest(context.Context, string, string, domain.ActorType, string) (domain.AuthRequest, error)
}

// ApproveAuthRequestGatewayInput defines the gateway payload for approving one auth request.
//
// The four ApproverPrincipalID / ApproverAgentInstanceID / ApproverLeaseToken
// / ApproverSessionID fields land in W3.1 to plumb orch-self-approval
// approver identity through to the gateway. Audit-trail persistence lands
// in W3.3 (column adds, scanner extension, record-mapping); W3.1 simply
// forwards the four fields to the gateway unchanged. The gateway today
// (autentauth.Service.ApproveAuthRequest) ignores them — that's intentional
// and W3.3 will surface them on the auth_requests row.
type ApproveAuthRequestGatewayInput struct {
	RequestID               string
	ResolvedBy              string
	ResolvedType            domain.ActorType
	ResolutionNote          string
	PathOverride            *domain.AuthRequestPath
	TTLOverride             time.Duration
	ApproverPrincipalID     string
	ApproverAgentInstanceID string
	ApproverLeaseToken      string
	ApproverSessionID       string
}

// CreateAuthRequest creates one pending auth request and mirrors it into the attention surface.
func (s *Service) CreateAuthRequest(ctx context.Context, in CreateAuthRequestInput) (domain.AuthRequest, error) {
	if s.authRequests == nil {
		return domain.AuthRequest{}, fmt.Errorf("auth requests are not configured")
	}
	path, err := domain.ParseAuthRequestPath(in.Path)
	if err != nil {
		return domain.AuthRequest{}, err
	}
	ctx, _, _ = withResolvedMutationActor(ctx, in.RequestedBy, "", in.RequestedType)
	requestedBy, requestedType := resolvedAuthRequestActor(ctx, in.RequestedBy, in.RequestedType)
	sessionTTL := in.RequestedSessionTTL
	if sessionTTL <= 0 {
		sessionTTL = defaultRequestedSessionTTL
	}
	timeout := in.Timeout
	if timeout <= 0 {
		timeout = defaultAuthRequestTimeout
	}
	req, err := domain.NewAuthRequest(domain.AuthRequestInput{
		ID:                  s.idGen(),
		Path:                path,
		PrincipalID:         strings.TrimSpace(in.PrincipalID),
		PrincipalType:       strings.TrimSpace(in.PrincipalType),
		PrincipalRole:       strings.TrimSpace(in.PrincipalRole),
		PrincipalName:       strings.TrimSpace(in.PrincipalName),
		ClientID:            strings.TrimSpace(in.ClientID),
		ClientType:          strings.TrimSpace(in.ClientType),
		ClientName:          strings.TrimSpace(in.ClientName),
		RequestedSessionTTL: sessionTTL,
		Reason:              strings.TrimSpace(in.Reason),
		Continuation:        authRequestContinuationForCreate(in.Continuation, in.RequesterClientID),
		RequestedByActor:    requestedBy,
		RequestedByType:     requestedType,
		Timeout:             timeout,
	}, s.clock())
	if err != nil {
		return domain.AuthRequest{}, err
	}
	req, err = s.authRequests.CreateAuthRequest(ctx, req)
	if err != nil {
		return domain.AuthRequest{}, err
	}
	attentionItems, err := authRequestAttentionItems(req, s.clock())
	if err != nil {
		return domain.AuthRequest{}, err
	}
	for _, attention := range attentionItems {
		if err := s.repo.CreateAttentionItem(ctx, attention); err != nil {
			return domain.AuthRequest{}, err
		}
	}
	return req, nil
}

// GetAuthRequest returns one auth request by id.
func (s *Service) GetAuthRequest(ctx context.Context, requestID string) (domain.AuthRequest, error) {
	if s.authRequests == nil {
		return domain.AuthRequest{}, fmt.Errorf("auth requests are not configured")
	}
	req, err := s.authRequests.GetAuthRequest(ctx, strings.TrimSpace(requestID))
	if err != nil {
		return domain.AuthRequest{}, err
	}
	if err := s.syncExpiredAuthRequestAttention(ctx, req); err != nil {
		return domain.AuthRequest{}, err
	}
	return req, nil
}

// ListAuthRequests lists auth requests using deterministic filters.
func (s *Service) ListAuthRequests(ctx context.Context, filter domain.AuthRequestListFilter) ([]domain.AuthRequest, error) {
	if s.authRequests == nil {
		return nil, fmt.Errorf("auth requests are not configured")
	}
	projectID := strings.TrimSpace(filter.ProjectID)
	repoFilter := filter
	if projectID != "" {
		repoFilter.ProjectID = ""
	}
	requests, err := s.authRequests.ListAuthRequests(ctx, repoFilter)
	if err != nil {
		return nil, err
	}
	out := make([]domain.AuthRequest, 0, len(requests))
	for _, request := range requests {
		if err := s.syncExpiredAuthRequestAttention(ctx, request); err != nil {
			return nil, err
		}
		if projectID != "" && !authRequestMatchesProject(request, projectID) {
			continue
		}
		out = append(out, request)
	}
	return out, nil
}

// ApproveAuthRequest approves one pending request and resolves its notification row.
//
// Approval paths (Drop 4a Wave 3 W3.1):
//
//   - Dev-TUI / system path: all four ApproverPrincipalID /
//     ApproverAgentInstanceID / ApproverLeaseToken / ApproverSessionID
//     fields empty. The orch-self-approval gate is bypassed; the dev's
//     approval comes through the human-driven TUI flow with ResolvedBy +
//     ResolvedType identifying the approver. This is the legacy path
//     preserved across the W3.1 change.
//
//   - Orch-self-approval cascade path: all four approver-identity fields
//     non-empty. The gate runs four checks in order:
//     1. Approver session loads cleanly via the configured AuthBackend
//     and the supplied ApproverSessionID maps to a session whose
//     PrincipalRole == "orchestrator".
//     2. Approver principal_id != request principal_id (no self-issue).
//     3. Request principal_role != "orchestrator" (orch-on-orch stays
//     dev-only).
//     4. Approver's effective approved-path encompasses the request's
//     path (path-within check). Cross-orch hardening: if the request
//     was created by a different orchestrator, additionally require
//     approver session principal_type == "steward" AND the request's
//     path roots under an action_item ancestor with
//     Metadata.Persistent=true && Metadata.Owner=="STEWARD".
//
// Required-non-empty rule: when ANY of the four approver-identity fields is
// non-empty, ALL FOUR must be non-empty. Partial input is rejected with a
// domain.ErrInvalidID-wrapped error before the gate even runs.
//
// Concurrent approve race: when two approvers race on the same request_id,
// the second call hits ErrAuthRequestNotPending because the row already
// transitioned pending → approved. No new race is introduced; this is the
// same race the dev-TUI path has always exposed.
func (s *Service) ApproveAuthRequest(ctx context.Context, in ApproveAuthRequestInput) (ApprovedAuthRequestResult, error) {
	if s.authRequests == nil {
		return ApprovedAuthRequestResult{}, fmt.Errorf("auth requests are not configured")
	}
	ctx, _, _ = withResolvedMutationActor(ctx, in.ResolvedBy, "", in.ResolvedType)
	resolvedBy, resolvedType := resolvedAuthRequestActor(ctx, in.ResolvedBy, in.ResolvedType)
	var pathOverride *domain.AuthRequestPath
	if strings.TrimSpace(in.Path) != "" {
		path, err := domain.ParseAuthRequestPath(in.Path)
		if err != nil {
			return ApprovedAuthRequestResult{}, err
		}
		pathOverride = &path
	}

	approverPrincipalID := strings.TrimSpace(in.ApproverPrincipalID)
	approverAgentInstanceID := strings.TrimSpace(in.ApproverAgentInstanceID)
	approverLeaseToken := strings.TrimSpace(in.ApproverLeaseToken)
	approverSessionID := strings.TrimSpace(in.ApproverSessionID)
	hasApproverIdentity := approverPrincipalID != "" || approverAgentInstanceID != "" || approverLeaseToken != "" || approverSessionID != ""
	if hasApproverIdentity {
		if approverPrincipalID == "" || approverAgentInstanceID == "" || approverLeaseToken == "" || approverSessionID == "" {
			return ApprovedAuthRequestResult{}, fmt.Errorf("approver identity fields must be all non-empty or all empty: %w", domain.ErrInvalidID)
		}
		if err := s.checkOrchSelfApprovalGate(ctx, strings.TrimSpace(in.RequestID), approverPrincipalID, approverSessionID); err != nil {
			return ApprovedAuthRequestResult{}, err
		}
	}

	out, err := s.authRequests.ApproveAuthRequest(ctx, ApproveAuthRequestGatewayInput{
		RequestID:               strings.TrimSpace(in.RequestID),
		ResolvedBy:              resolvedBy,
		ResolvedType:            resolvedType,
		ResolutionNote:          strings.TrimSpace(in.ResolutionNote),
		PathOverride:            pathOverride,
		TTLOverride:             in.SessionTTL,
		ApproverPrincipalID:     approverPrincipalID,
		ApproverAgentInstanceID: approverAgentInstanceID,
		ApproverLeaseToken:      approverLeaseToken,
		ApproverSessionID:       approverSessionID,
	})
	if err != nil {
		return ApprovedAuthRequestResult{}, err
	}
	s.publishAuthRequestResolved(out.Request)
	if err := s.resolveAuthRequestAttention(ctx, out.Request, resolvedBy, resolvedType); err != nil {
		return ApprovedAuthRequestResult{}, err
	}
	return out, nil
}

// checkOrchSelfApprovalGate runs the Drop 4a Wave 3 W3.1 self-approval gate:
// validates the approver session, rejects orch-self-approval and orch-on-orch,
// and enforces path-encompasses + STEWARD cross-subtree exception.
func (s *Service) checkOrchSelfApprovalGate(ctx context.Context, requestID, approverPrincipalID, approverSessionID string) error {
	if s.authBackend == nil {
		return fmt.Errorf("auth backend is not configured for orch-self-approval gate: %w", domain.ErrInvalidID)
	}
	req, err := s.authRequests.GetAuthRequest(ctx, requestID)
	if err != nil {
		return err
	}

	// Reject 1: request principal_role IS orchestrator → stays dev-only.
	if domain.NormalizeAuthRequestRole(domain.AuthRequestRole(req.PrincipalRole)) == domain.AuthRequestRoleOrchestrator {
		return fmt.Errorf("orch cannot approve another orchestrator's auth request: %w", domain.ErrAuthorizationDenied)
	}

	// Reject 2: orch-self-approval (approver and requester share principal_id).
	if strings.TrimSpace(req.PrincipalID) == approverPrincipalID {
		return fmt.Errorf("orch cannot approve its own auth request (principal_id %q): %w", approverPrincipalID, domain.ErrAuthorizationDenied)
	}

	// Validate approver session via the configured backend. The transport
	// layer is responsible for supplying acting_session_secret matching
	// approverSessionID — see common.AppServiceAdapter.ApproveAuthRequest.
	approverSession, err := s.authBackend.ListAuthSessions(ctx, AuthSessionFilter{SessionID: approverSessionID, Limit: 1})
	if err != nil {
		return fmt.Errorf("load approver session: %w", err)
	}
	if len(approverSession) == 0 {
		return fmt.Errorf("approver session %q not found: %w", approverSessionID, domain.ErrAuthorizationDenied)
	}
	session := approverSession[0]

	// Reject 3: approver session is not orchestrator-roled.
	if strings.TrimSpace(session.PrincipalRole) != string(domain.AuthRequestRoleOrchestrator) {
		return fmt.Errorf("approver session role %q is not orchestrator: %w", session.PrincipalRole, domain.ErrAuthorizationDenied)
	}

	// Reject 4: approver's session principal_id does not match the supplied
	// ApproverPrincipalID (defense-in-depth — caller cannot lie about whose
	// identity they're acting under).
	if strings.TrimSpace(session.PrincipalID) != approverPrincipalID {
		return fmt.Errorf("approver session principal_id %q does not match supplied ApproverPrincipalID %q: %w", session.PrincipalID, approverPrincipalID, domain.ErrAuthorizationDenied)
	}

	// Reject 5: approver path does not encompass request path.
	approverPath, err := domain.ParseAuthRequestPath(strings.TrimSpace(session.ApprovedPath))
	if err != nil {
		// Fallback: project-only path derived from session.ProjectID.
		if pid := strings.TrimSpace(session.ProjectID); pid != "" {
			approverPath, err = domain.AuthRequestPath{ProjectID: pid}.Normalize()
		}
		if err != nil {
			return fmt.Errorf("approver session has no usable approved path: %w", domain.ErrAuthorizationDenied)
		}
	}
	requestPath, err := domain.ParseAuthRequestPath(req.Path)
	if err != nil {
		return fmt.Errorf("request path %q is invalid: %w", req.Path, err)
	}
	if !authRequestPathWithin(approverPath, requestPath) {
		return fmt.Errorf("approver path %q does not encompass request path %q: %w", approverPath.String(), requestPath.String(), domain.ErrAuthorizationDenied)
	}

	// Cross-orch hardening (Drop 4a Wave 3 W3.1, falsification mitigations
	// 1+5): path-encompasses is necessary but not sufficient — STEWARD's
	// project-scoped lease encompasses every drop subtree, so a non-STEWARD
	// orch with the right project lease could otherwise approve another
	// orch's subagent. If the requesting actor is a different orchestrator,
	// require the approver to be a STEWARD session AND require the request
	// path to root under an action_item with Metadata.Persistent=true &&
	// Metadata.Owner=="STEWARD".
	requestedBy := strings.TrimSpace(req.RequestedByActor)
	if requestedBy != "" && requestedBy != approverPrincipalID {
		// Different orch created the request. STEWARD-only cross-subtree
		// exception applies.
		//
		// Drop 4a droplet 4a.24 fix: read AuthRequestPrincipalType (the
		// tillsyn-axis value preserved across the autent boundary's
		// steward → agent collapse), NOT PrincipalType (always "agent" for
		// steward principals because autent's closed enum lacks "steward").
		// Reading PrincipalType made this branch ALWAYS reject — the
		// STEWARD cross-subtree exception was non-functional.
		if strings.TrimSpace(session.AuthRequestPrincipalType) != "steward" {
			return fmt.Errorf("cross-orch approval requires steward approver (request requested_by %q, approver %q): %w", requestedBy, approverPrincipalID, domain.ErrAuthorizationDenied)
		}
		if err := s.requireStewardPersistentAncestor(ctx, requestPath); err != nil {
			return err
		}
	}
	return nil
}

// requireStewardPersistentAncestor walks the request's leaf action-item
// ancestry looking for any node with Metadata.Persistent=true &&
// Metadata.Owner=="STEWARD". Returns nil when found; ErrAuthorizationDenied
// otherwise. The check is metadata-driven, not name-driven (Wave 3
// falsification attack 5 mitigation — no hardcoded persistent-parent IDs).
func (s *Service) requireStewardPersistentAncestor(ctx context.Context, path domain.AuthRequestPath) error {
	leafID := strings.TrimSpace(path.ScopeID)
	if leafID == "" || path.Kind != domain.AuthRequestPathKindProject {
		// Project-scope-only path with no action_item resolution. Without an
		// action item to walk, the cross-subtree exception cannot fire —
		// the gate already verified approver is steward and path
		// encompassed; reject here so a STEWARD with a project-wide lease
		// cannot blanket-approve every subagent in the project without the
		// request rooting under one of STEWARD's persistent parents.
		if path.BranchID == "" && len(path.PhaseIDs) == 0 {
			return fmt.Errorf("steward cross-subtree approval requires the request path to root under a persistent STEWARD-owned ancestor; project-scope-only paths do not qualify: %w", domain.ErrAuthorizationDenied)
		}
	}
	cursorID := leafID
	visited := make(map[string]struct{}, 8)
	for cursorID != "" {
		if _, seen := visited[cursorID]; seen {
			return fmt.Errorf("action_item ancestry walk hit a cycle at %q: %w", cursorID, domain.ErrInvalidID)
		}
		visited[cursorID] = struct{}{}
		ai, err := s.repo.GetActionItem(ctx, cursorID)
		if err != nil {
			if errors.Is(err, ErrNotFound) {
				break
			}
			return fmt.Errorf("load action_item %q for steward ancestry walk: %w", cursorID, err)
		}
		if ai.Persistent && strings.EqualFold(strings.TrimSpace(ai.Owner), "STEWARD") {
			return nil
		}
		cursorID = strings.TrimSpace(ai.ParentID)
	}
	return fmt.Errorf("request path does not root under a STEWARD-owned persistent ancestor: %w", domain.ErrAuthorizationDenied)
}

// ClaimAuthRequest returns one requester-visible auth request state and approved session secret when the continuation token matches.
func (s *Service) ClaimAuthRequest(ctx context.Context, in ClaimAuthRequestInput) (ClaimedAuthRequestResult, error) {
	if s.authRequests == nil {
		return ClaimedAuthRequestResult{}, fmt.Errorf("auth requests are not configured")
	}
	requestID := strings.TrimSpace(in.RequestID)
	if requestID == "" {
		return ClaimedAuthRequestResult{}, fmt.Errorf("auth request id is required")
	}
	waitTimeout := in.WaitTimeout
	if waitTimeout < 0 {
		return ClaimedAuthRequestResult{}, fmt.Errorf("wait timeout must be >= 0")
	}
	if waitTimeout <= 0 {
		return s.claimAuthRequestOnce(ctx, requestID, in)
	}
	if s.liveWait != nil {
		return s.claimAuthRequestLive(ctx, requestID, in, waitTimeout)
	}
	return s.claimAuthRequestPolling(ctx, requestID, in, waitTimeout)
}

// claimAuthRequestOnce loads one requester-visible auth request state without waiting.
func (s *Service) claimAuthRequestOnce(ctx context.Context, requestID string, in ClaimAuthRequestInput) (ClaimedAuthRequestResult, error) {
	result, err := s.claimAuthRequestRecord(ctx, requestID, in)
	if err != nil {
		return ClaimedAuthRequestResult{}, err
	}
	if err := s.syncExpiredAuthRequestAttention(ctx, result.Request); err != nil {
		return ClaimedAuthRequestResult{}, err
	}
	return result, nil
}

// claimAuthRequestRecord loads one auth request and validates the approved child claimant identity against it.
func (s *Service) claimAuthRequestRecord(ctx context.Context, requestID string, in ClaimAuthRequestInput) (ClaimedAuthRequestResult, error) {
	req, err := s.authRequests.GetAuthRequest(ctx, requestID)
	if err != nil {
		return ClaimedAuthRequestResult{}, err
	}
	if err := authRequestClaimIdentityMatches(req, strings.TrimSpace(in.PrincipalID), strings.TrimSpace(in.ClientID)); err != nil {
		return ClaimedAuthRequestResult{}, err
	}
	if !authRequestResumeTokenMatches(req.Continuation, strings.TrimSpace(in.ResumeToken)) {
		return ClaimedAuthRequestResult{}, domain.ErrInvalidAuthContinuation
	}
	result := ClaimedAuthRequestResult{Request: req}
	if domain.NormalizeAuthRequestState(req.State) == domain.AuthRequestStateApproved {
		result.SessionSecret = req.IssuedSessionSecret
	}
	return result, nil
}

// claimAuthRequestLive waits on one in-process resolution event instead of polling storage.
func (s *Service) claimAuthRequestLive(ctx context.Context, requestID string, in ClaimAuthRequestInput, waitTimeout time.Duration) (ClaimedAuthRequestResult, error) {
	baselineSequence, err := s.liveWaitBaselineSequence(ctx, LiveWaitEventAuthRequestResolved, requestID)
	if err != nil {
		return ClaimedAuthRequestResult{}, err
	}
	result, err := s.claimAuthRequestOnce(ctx, requestID, in)
	if err != nil {
		return ClaimedAuthRequestResult{}, err
	}
	if domain.NormalizeAuthRequestState(result.Request.State) != domain.AuthRequestStatePending {
		return result, nil
	}
	waitCtx, cancel := context.WithDeadline(ctx, authRequestWaitDeadline(s.clock().UTC(), result.Request.ExpiresAt, waitTimeout))
	defer cancel()
	if _, err := s.liveWait.Wait(waitCtx, LiveWaitEventAuthRequestResolved, requestID, baselineSequence); err != nil {
		if !errors.Is(err, context.DeadlineExceeded) {
			return ClaimedAuthRequestResult{}, err
		}
	}
	result, err = s.claimAuthRequestOnce(ctx, requestID, in)
	if err != nil {
		return ClaimedAuthRequestResult{}, err
	}
	if domain.NormalizeAuthRequestState(result.Request.State) == domain.AuthRequestStatePending {
		result.Waiting = true
	}
	return result, nil
}

// claimAuthRequestPolling preserves the legacy polling-based wait behavior when no live broker is configured.
func (s *Service) claimAuthRequestPolling(ctx context.Context, requestID string, in ClaimAuthRequestInput, waitTimeout time.Duration) (ClaimedAuthRequestResult, error) {
	deadline := time.Now().UTC().Add(waitTimeout)
	for {
		result, err := s.claimAuthRequestOnce(ctx, requestID, in)
		if err != nil {
			return ClaimedAuthRequestResult{}, err
		}
		if domain.NormalizeAuthRequestState(result.Request.State) != domain.AuthRequestStatePending {
			return result, nil
		}
		if !time.Now().UTC().Before(deadline) {
			result.Waiting = true
			return result, nil
		}
		timer := time.NewTimer(claimAuthRequestPollInterval)
		select {
		case <-ctx.Done():
			if !timer.Stop() {
				<-timer.C
			}
			return ClaimedAuthRequestResult{}, ctx.Err()
		case <-timer.C:
		}
	}
}

// DenyAuthRequest denies one pending request and resolves its notification row.
func (s *Service) DenyAuthRequest(ctx context.Context, in DenyAuthRequestInput) (domain.AuthRequest, error) {
	if s.authRequests == nil {
		return domain.AuthRequest{}, fmt.Errorf("auth requests are not configured")
	}
	ctx, _, _ = withResolvedMutationActor(ctx, in.ResolvedBy, "", in.ResolvedType)
	resolvedBy, resolvedType := resolvedAuthRequestActor(ctx, in.ResolvedBy, in.ResolvedType)
	req, err := s.authRequests.DenyAuthRequest(ctx, strings.TrimSpace(in.RequestID), resolvedBy, resolvedType, strings.TrimSpace(in.ResolutionNote))
	if err != nil {
		return domain.AuthRequest{}, err
	}
	s.publishAuthRequestResolved(req)
	if err := s.resolveAuthRequestAttention(ctx, req, resolvedBy, resolvedType); err != nil {
		return domain.AuthRequest{}, err
	}
	return req, nil
}

// CancelAuthRequest cancels one pending request and resolves its notification row.
func (s *Service) CancelAuthRequest(ctx context.Context, in CancelAuthRequestInput) (domain.AuthRequest, error) {
	if s.authRequests == nil {
		return domain.AuthRequest{}, fmt.Errorf("auth requests are not configured")
	}
	ctx, _, _ = withResolvedMutationActor(ctx, in.ResolvedBy, "", in.ResolvedType)
	resolvedBy, resolvedType := resolvedAuthRequestActor(ctx, in.ResolvedBy, in.ResolvedType)
	req, err := s.authRequests.CancelAuthRequest(ctx, strings.TrimSpace(in.RequestID), resolvedBy, resolvedType, strings.TrimSpace(in.ResolutionNote))
	if err != nil {
		return domain.AuthRequest{}, err
	}
	s.publishAuthRequestResolved(req)
	if err := s.resolveAuthRequestAttention(ctx, req, resolvedBy, resolvedType); err != nil {
		return domain.AuthRequest{}, err
	}
	return req, nil
}

// publishAuthRequestResolved wakes any live waiters for one terminal auth request.
func (s *Service) publishAuthRequestResolved(req domain.AuthRequest) {
	if s == nil || s.liveWait == nil {
		return
	}
	s.liveWait.Publish(LiveWaitEvent{
		Type:  LiveWaitEventAuthRequestResolved,
		Key:   strings.TrimSpace(req.ID),
		Value: domain.NormalizeAuthRequestState(req.State),
	})
}

// authRequestWaitDeadline bounds one live wait by both user timeout and auth-request expiry.
func authRequestWaitDeadline(now time.Time, expiresAt time.Time, waitTimeout time.Duration) time.Time {
	deadline := now.Add(waitTimeout)
	if !expiresAt.IsZero() && expiresAt.Before(deadline) {
		return expiresAt
	}
	return deadline
}

// authRequestResumeTokenMatches reports whether one continuation payload carries the expected shared resume token.
func authRequestResumeTokenMatches(continuation map[string]any, want string) bool {
	want = strings.TrimSpace(want)
	if want == "" {
		return false
	}
	got, _ := continuation["resume_token"].(string)
	return strings.TrimSpace(got) == want
}

// authRequestPathWithin reports whether candidate is equal to or narrower
// than requested. Mirrors the helper used by the autent gateway and the MCP
// adapter (`internal/adapters/auth/autentauth/service.go:1135` and
// `internal/adapters/server/common/app_service_adapter_mcp.go:477`); kept
// app-private so the orch-self-approval gate (Drop 4a Wave 3 W3.1) can run
// without crossing the import boundary into either adapter. Future
// refinement: lift this to `domain.AuthRequestPath.Encompasses(other)` once
// the three call sites are unified.
func authRequestPathWithin(requested, candidate domain.AuthRequestPath) bool {
	requested, err := requested.Normalize()
	if err != nil {
		return false
	}
	candidate, err = candidate.Normalize()
	if err != nil {
		return false
	}
	switch requested.Kind {
	case domain.AuthRequestPathKindGlobal:
		return true
	case domain.AuthRequestPathKindProjects:
		switch candidate.Kind {
		case domain.AuthRequestPathKindGlobal:
			return false
		case domain.AuthRequestPathKindProjects:
			for _, projectID := range candidate.ProjectIDs {
				if !requested.MatchesProject(projectID) {
					return false
				}
			}
			return len(candidate.ProjectIDs) > 0
		default:
			return requested.MatchesProject(candidate.ProjectID)
		}
	case domain.AuthRequestPathKindProject:
	default:
		return false
	}
	if requested.ProjectID != candidate.ProjectID {
		return false
	}
	if requested.BranchID == "" {
		return true
	}
	if requested.BranchID != candidate.BranchID {
		return false
	}
	if len(requested.PhaseIDs) == 0 {
		return true
	}
	if len(candidate.PhaseIDs) < len(requested.PhaseIDs) {
		return false
	}
	for idx, phaseID := range requested.PhaseIDs {
		if candidate.PhaseIDs[idx] != phaseID {
			return false
		}
	}
	return true
}

// authRequestClaimIdentityMatches reports whether one claim request matches the requested child principal/client pair.
func authRequestClaimIdentityMatches(req domain.AuthRequest, principalID, clientID string) error {
	principalID = strings.TrimSpace(principalID)
	clientID = strings.TrimSpace(clientID)
	if principalID == "" || clientID == "" {
		return domain.ErrAuthRequestClaimMismatch
	}
	if authRequestChildClaimIdentityMatches(req, principalID, clientID) {
		return nil
	}
	return domain.ErrAuthRequestClaimMismatch
}

// authRequestChildClaimIdentityMatches reports whether one claim request matches the requested child principal/client pair.
func authRequestChildClaimIdentityMatches(req domain.AuthRequest, principalID, clientID string) bool {
	return strings.TrimSpace(req.PrincipalID) == principalID && strings.TrimSpace(req.ClientID) == clientID
}

// ListAuthSessions returns caller-safe auth-session inventory through the configured backend.
func (s *Service) ListAuthSessions(ctx context.Context, filter AuthSessionFilter) ([]AuthSession, error) {
	if s.authBackend == nil {
		return nil, fmt.Errorf("auth backend is not configured")
	}
	projectID := strings.TrimSpace(filter.ProjectID)
	backendFilter := filter
	if projectID != "" {
		backendFilter.ProjectID = ""
	}
	sessions, err := s.authBackend.ListAuthSessions(ctx, backendFilter)
	if err != nil {
		return nil, err
	}
	if projectID == "" {
		return sessions, nil
	}
	out := make([]AuthSession, 0, len(sessions))
	for _, session := range sessions {
		if authSessionMatchesProject(session, projectID) {
			out = append(out, session)
		}
	}
	return out, nil
}

// ValidateAuthSession validates one session/secret pair through the configured backend.
func (s *Service) ValidateAuthSession(ctx context.Context, sessionID, sessionSecret string) (ValidatedAuthSession, error) {
	if s.authBackend == nil {
		return ValidatedAuthSession{}, fmt.Errorf("auth backend is not configured")
	}
	return s.authBackend.ValidateAuthSession(ctx, strings.TrimSpace(sessionID), strings.TrimSpace(sessionSecret))
}

// RevokeAuthSession revokes one auth session through the configured backend.
func (s *Service) RevokeAuthSession(ctx context.Context, sessionID, reason string) (AuthSession, error) {
	if s.authBackend == nil {
		return AuthSession{}, fmt.Errorf("auth backend is not configured")
	}
	return s.authBackend.RevokeAuthSession(ctx, strings.TrimSpace(sessionID), strings.TrimSpace(reason))
}

// authRequestAttentionItems builds mirrored user-action notifications for one pending auth request.
func authRequestAttentionItems(req domain.AuthRequest, now time.Time) ([]domain.AttentionItem, error) {
	attentionIDs, projectIDs := authRequestAttentionTargets(req)
	items := make([]domain.AttentionItem, 0, len(attentionIDs))
	for idx := range attentionIDs {
		item, err := domain.NewAttentionItem(domain.AttentionItemInput{
			ID:                 attentionIDs[idx],
			ProjectID:          projectIDs[idx],
			ScopeType:          domain.ScopeLevelProject,
			ScopeID:            projectIDs[idx],
			Kind:               domain.AttentionKindApprovalRequired,
			Summary:            fmt.Sprintf("%s: %s wants %s", authRequestNotificationLabel, firstNonEmptyTrimmed(req.PrincipalName, req.PrincipalID), req.Path),
			BodyMarkdown:       authRequestThreadBody(req),
			RequiresUserAction: true,
			CreatedByActor:     req.RequestedByActor,
			CreatedByType:      req.RequestedByType,
		}, now.UTC())
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

// authRequestThreadBody renders one markdown-rich review summary for auth-request notifications.
func authRequestThreadBody(req domain.AuthRequest) string {
	lines := []string{
		fmt.Sprintf("Requested principal: `%s`", firstNonEmptyTrimmed(req.PrincipalName, req.PrincipalID)),
		fmt.Sprintf("Principal type: `%s`", req.PrincipalType),
		fmt.Sprintf("Client: `%s` (`%s`)", firstNonEmptyTrimmed(req.ClientName, req.ClientID), req.ClientType),
		fmt.Sprintf("Path: `%s`", req.Path),
		fmt.Sprintf("Requested session TTL: `%s`", req.RequestedSessionTTL),
		fmt.Sprintf("Request timeout: `%s`", req.ExpiresAt.UTC().Format(time.RFC3339)),
	}
	if reason := strings.TrimSpace(req.Reason); reason != "" {
		lines = append(lines, "", "Reason:", reason)
	}
	if len(req.Continuation) > 0 {
		lines = append(lines, "", "Continuation metadata:")
		keys := make([]string, 0, len(req.Continuation))
		for key := range req.Continuation {
			keys = append(keys, key)
		}
		slices.Sort(keys)
		for _, key := range keys {
			lines = append(lines, fmt.Sprintf("- `%s`: `%s`", key, formatAuthRequestContinuationValue(req.Continuation[key])))
		}
	}
	return strings.Join(lines, "\n")
}

// resolvedAuthRequestActor resolves approval/create attribution from explicit values or context.
func resolvedAuthRequestActor(ctx context.Context, actorID string, actorType domain.ActorType) (string, domain.ActorType) {
	if actor, ok := MutationActorFromContext(ctx); ok {
		return actor.ActorID, actor.ActorType
	}
	actorID = strings.TrimSpace(actorID)
	if actorID == "" {
		actorID = "tillsyn-user"
	}
	actorType = domain.ActorType(strings.TrimSpace(strings.ToLower(string(actorType))))
	if actorType == "" {
		actorType = domain.ActorTypeUser
	}
	return actorID, actorType
}

// syncExpiredAuthRequestAttention resolves mirrored request notifications once a request has timed out.
func (s *Service) syncExpiredAuthRequestAttention(ctx context.Context, req domain.AuthRequest) error {
	if domain.NormalizeAuthRequestState(req.State) != domain.AuthRequestStateExpired {
		return nil
	}
	if err := s.resolveAuthRequestAttention(ctx, req, "tillsyn-system", domain.ActorTypeSystem); err != nil {
		return err
	}
	return nil
}

// resolveAuthRequestAttention resolves every mirrored attention row for one auth request.
func (s *Service) resolveAuthRequestAttention(ctx context.Context, req domain.AuthRequest, resolvedBy string, resolvedType domain.ActorType) error {
	for _, attentionID := range authRequestAttentionIDs(req) {
		if _, err := s.repo.ResolveAttentionItem(ctx, attentionID, resolvedBy, resolvedType, s.clock().UTC()); err != nil && !errors.Is(err, ErrNotFound) {
			return err
		}
	}
	return nil
}

// authRequestAttentionIDs returns every mirrored attention id for one auth request.
func authRequestAttentionIDs(req domain.AuthRequest) []string {
	ids, _ := authRequestAttentionTargets(req)
	return ids
}

// authRequestAttentionTargets returns mirrored attention ids plus the routed project id for each row.
func authRequestAttentionTargets(req domain.AuthRequest) ([]string, []string) {
	path, err := domain.ParseAuthRequestPath(req.Path)
	if err != nil {
		projectID := firstNonEmptyTrimmed(req.ProjectID, domain.AuthRequestGlobalProjectID)
		return []string{req.ID}, []string{projectID}
	}
	switch path.Kind {
	case domain.AuthRequestPathKindProjects:
		ids := make([]string, 0, len(path.ProjectIDs))
		projectIDs := make([]string, 0, len(path.ProjectIDs))
		for _, projectID := range path.ProjectIDs {
			ids = append(ids, authRequestProjectAttentionID(req.ID, projectID))
			projectIDs = append(projectIDs, projectID)
		}
		return ids, projectIDs
	case domain.AuthRequestPathKindGlobal:
		return []string{authRequestGlobalAttentionID(req.ID)}, []string{domain.AuthRequestGlobalProjectID}
	default:
		return []string{req.ID}, []string{path.ProjectID}
	}
}

// AuthRequestIDFromAttentionID resolves one mirrored attention id back to the stored auth request id.
func AuthRequestIDFromAttentionID(attentionID string) string {
	attentionID = strings.TrimSpace(attentionID)
	if base, _, ok := strings.Cut(attentionID, "::"); ok && strings.TrimSpace(base) != "" {
		return strings.TrimSpace(base)
	}
	return attentionID
}

func authRequestProjectAttentionID(requestID, projectID string) string {
	return strings.TrimSpace(requestID) + "::project::" + strings.TrimSpace(projectID)
}

func authRequestGlobalAttentionID(requestID string) string {
	return strings.TrimSpace(requestID) + "::global"
}

// authRequestMatchesProject reports whether one request applies to the requested project filter.
func authRequestMatchesProject(req domain.AuthRequest, projectID string) bool {
	path, err := domain.ParseAuthRequestPath(firstNonEmptyTrimmed(req.ApprovedPath, req.Path))
	if err != nil {
		return strings.TrimSpace(req.ProjectID) == strings.TrimSpace(projectID)
	}
	return path.MatchesProject(projectID)
}

// authSessionMatchesProject reports whether one session applies to the requested project filter.
func authSessionMatchesProject(session AuthSession, projectID string) bool {
	path, err := domain.ParseAuthRequestPath(strings.TrimSpace(session.ApprovedPath))
	if err != nil {
		return strings.TrimSpace(session.ProjectID) == strings.TrimSpace(projectID)
	}
	return path.MatchesProject(projectID)
}

// cloneAuthRequestContinuation deep-copies request continuation metadata.
func cloneAuthRequestContinuation(in map[string]any) map[string]any {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]any, len(in))
	for key, value := range in {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		out[key] = cloneAuthRequestContinuationValue(value)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// authRequestContinuationForCreate clones continuation metadata and injects one requester-bound claim client id when provided.
func authRequestContinuationForCreate(in map[string]any, requesterClientID string) map[string]any {
	out := cloneAuthRequestContinuation(in)
	if trimmed := strings.TrimSpace(requesterClientID); trimmed != "" {
		if out == nil {
			out = make(map[string]any, 1)
		}
		out[AuthRequestContinuationRequesterClientIDKey] = trimmed
	}
	return out
}

// AuthRequestClaimClientIDFromContinuation resolves the requester-bound claim client identifier from private continuation metadata.
func AuthRequestClaimClientIDFromContinuation(continuation map[string]any, fallback string) string {
	if continuation != nil {
		if raw, ok := continuation[AuthRequestContinuationRequesterClientIDKey].(string); ok {
			if trimmed := strings.TrimSpace(raw); trimmed != "" {
				return trimmed
			}
		}
	}
	return strings.TrimSpace(fallback)
}

// cloneAuthRequestContinuationValue deep-copies one JSON-compatible continuation value.
func cloneAuthRequestContinuationValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		return cloneAuthRequestContinuation(typed)
	case []any:
		out := make([]any, 0, len(typed))
		for _, item := range typed {
			out = append(out, cloneAuthRequestContinuationValue(item))
		}
		return out
	default:
		return typed
	}
}

// formatAuthRequestContinuationValue renders nested continuation metadata deterministically for notifications.
func formatAuthRequestContinuationValue(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	default:
		encoded, err := json.Marshal(typed)
		if err != nil {
			return fmt.Sprintf("%v", typed)
		}
		return string(encoded)
	}
}
