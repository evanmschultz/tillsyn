package app

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/hylla/tillsyn/internal/domain"
)

const (
	defaultAuthRequestTimeout    = 30 * time.Minute
	defaultRequestedSessionTTL   = 8 * time.Hour
	authRequestNotificationLabel = "auth request"
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
	PrincipalID string
	ClientID    string
	State       string
	Limit       int
}

// AuthSession stores one caller-safe auth-session record.
type AuthSession struct {
	SessionID        string
	PrincipalID      string
	PrincipalType    string
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
	PrincipalName       string
	ClientID            string
	ClientType          string
	ClientName          string
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

// AuthRequestGateway defines the auth-request lifecycle needed by the app service.
type AuthRequestGateway interface {
	CreateAuthRequest(context.Context, domain.AuthRequest) (domain.AuthRequest, error)
	GetAuthRequest(context.Context, string) (domain.AuthRequest, error)
	ListAuthRequests(context.Context, domain.AuthRequestListFilter) ([]domain.AuthRequest, error)
	ApproveAuthRequest(context.Context, ApproveAuthRequestGatewayInput) (ApprovedAuthRequestResult, error)
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
		PrincipalName:       strings.TrimSpace(in.PrincipalName),
		ClientID:            strings.TrimSpace(in.ClientID),
		ClientType:          strings.TrimSpace(in.ClientType),
		ClientName:          strings.TrimSpace(in.ClientName),
		RequestedSessionTTL: sessionTTL,
		Reason:              strings.TrimSpace(in.Reason),
		Continuation:        cloneAuthRequestContinuation(in.Continuation),
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
	attention, err := authRequestAttentionItem(req, s.clock())
	if err != nil {
		return domain.AuthRequest{}, err
	}
	if err := s.repo.CreateAttentionItem(ctx, attention); err != nil {
		return domain.AuthRequest{}, err
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
	requests, err := s.authRequests.ListAuthRequests(ctx, filter)
	if err != nil {
		return nil, err
	}
	for _, request := range requests {
		if err := s.syncExpiredAuthRequestAttention(ctx, request); err != nil {
			return nil, err
		}
	}
	return requests, nil
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
	if _, err := s.repo.ResolveAttentionItem(ctx, out.Request.ID, resolvedBy, resolvedType, s.clock().UTC()); err != nil {
		return ApprovedAuthRequestResult{}, err
	}
	return out, nil
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
	if _, err := s.repo.ResolveAttentionItem(ctx, req.ID, resolvedBy, resolvedType, s.clock().UTC()); err != nil {
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
	if _, err := s.repo.ResolveAttentionItem(ctx, req.ID, resolvedBy, resolvedType, s.clock().UTC()); err != nil {
		return domain.AuthRequest{}, err
	}
	return req, nil
}

// ListAuthSessions returns caller-safe auth-session inventory through the configured backend.
func (s *Service) ListAuthSessions(ctx context.Context, filter AuthSessionFilter) ([]AuthSession, error) {
	if s.authBackend == nil {
		return nil, fmt.Errorf("auth backend is not configured")
	}
	return s.authBackend.ListAuthSessions(ctx, filter)
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

// authRequestAttentionItem builds the mirrored user-action notification for one pending auth request.
func authRequestAttentionItem(req domain.AuthRequest, now time.Time) (domain.AttentionItem, error) {
	item, err := domain.NewAttentionItem(domain.AttentionItemInput{
		ID:                 req.ID,
		ProjectID:          req.ProjectID,
		BranchID:           req.BranchID,
		ScopeType:          req.ScopeType,
		ScopeID:            req.ScopeID,
		Kind:               domain.AttentionKindApprovalRequired,
		Summary:            fmt.Sprintf("%s: %s wants %s", authRequestNotificationLabel, firstNonEmptyTrimmed(req.PrincipalName, req.PrincipalID), req.Path),
		BodyMarkdown:       authRequestThreadBody(req),
		RequiresUserAction: true,
		CreatedByActor:     req.RequestedByActor,
		CreatedByType:      req.RequestedByType,
	}, now.UTC())
	if err != nil {
		return domain.AttentionItem{}, err
	}
	return item, nil
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
	if _, err := s.repo.ResolveAttentionItem(ctx, req.ID, "tillsyn-system", domain.ActorTypeSystem, s.clock().UTC()); err != nil {
		return err
	}
	return nil
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
