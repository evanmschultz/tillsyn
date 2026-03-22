package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/hylla/tillsyn/internal/domain"
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
type AuthSession struct {
	SessionID        string
	ProjectID        string
	AuthRequestID    string
	ApprovedPath     string
	PrincipalID      string
	PrincipalType    string
	PrincipalRole    string
	PrincipalName    string
	ClientID         string
	ClientType       string
	ClientName       string
	IssuedAt         time.Time
	ExpiresAt        time.Time
	LastValidatedAt  *time.Time
	RevokedAt        *time.Time
	RevocationReason string
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
type ApproveAuthRequestInput struct {
	RequestID      string
	Path           string
	SessionTTL     time.Duration
	ResolvedBy     string
	ResolvedType   domain.ActorType
	ResolutionNote string
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
type ApproveAuthRequestGatewayInput struct {
	RequestID      string
	ResolvedBy     string
	ResolvedType   domain.ActorType
	ResolutionNote string
	PathOverride   *domain.AuthRequestPath
	TTLOverride    time.Duration
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
	out, err := s.authRequests.ApproveAuthRequest(ctx, ApproveAuthRequestGatewayInput{
		RequestID:      strings.TrimSpace(in.RequestID),
		ResolvedBy:     resolvedBy,
		ResolvedType:   resolvedType,
		ResolutionNote: strings.TrimSpace(in.ResolutionNote),
		PathOverride:   pathOverride,
		TTLOverride:    in.SessionTTL,
	})
	if err != nil {
		return ApprovedAuthRequestResult{}, err
	}
	if err := s.resolveAuthRequestAttention(ctx, out.Request, resolvedBy, resolvedType); err != nil {
		return ApprovedAuthRequestResult{}, err
	}
	return out, nil
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
		result, err := s.authRequests.ClaimAuthRequest(ctx, ClaimAuthRequestInput{
			RequestID:   requestID,
			ResumeToken: strings.TrimSpace(in.ResumeToken),
			PrincipalID: strings.TrimSpace(in.PrincipalID),
			ClientID:    strings.TrimSpace(in.ClientID),
		})
		if err != nil {
			return ClaimedAuthRequestResult{}, err
		}
		if err := s.syncExpiredAuthRequestAttention(ctx, result.Request); err != nil {
			return ClaimedAuthRequestResult{}, err
		}
		return result, nil
	}
	deadline := time.Now().UTC().Add(waitTimeout)
	for {
		result, err := s.authRequests.ClaimAuthRequest(ctx, ClaimAuthRequestInput{
			RequestID:   requestID,
			ResumeToken: strings.TrimSpace(in.ResumeToken),
			PrincipalID: strings.TrimSpace(in.PrincipalID),
			ClientID:    strings.TrimSpace(in.ClientID),
		})
		if err != nil {
			return ClaimedAuthRequestResult{}, err
		}
		if domain.NormalizeAuthRequestState(result.Request.State) != domain.AuthRequestStatePending {
			if err := s.syncExpiredAuthRequestAttention(ctx, result.Request); err != nil {
				return ClaimedAuthRequestResult{}, err
			}
			return result, nil
		}
		if !time.Now().UTC().Before(deadline) {
			result.Waiting = true
			if err := s.syncExpiredAuthRequestAttention(ctx, result.Request); err != nil {
				return ClaimedAuthRequestResult{}, err
			}
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
	if err := s.resolveAuthRequestAttention(ctx, req, resolvedBy, resolvedType); err != nil {
		return domain.AuthRequest{}, err
	}
	return req, nil
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
