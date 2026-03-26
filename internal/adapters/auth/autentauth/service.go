// Package autentauth adapts autent into tillsyn's local shared-DB runtime.
package autentauth

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	autent "github.com/evanmschultz/autent"
	autentdomain "github.com/evanmschultz/autent/domain"
	autentsqlite "github.com/evanmschultz/autent/sqlite"
	autenttoken "github.com/evanmschultz/autent/token"
	"github.com/hylla/tillsyn/internal/app"
	"github.com/hylla/tillsyn/internal/domain"
	sqlite3 "github.com/ncruces/go-sqlite3"
)

const (
	// DefaultTablePrefix scopes autent tables inside the shared tillsyn database.
	DefaultTablePrefix = "autent_"
	// DogfoodPolicyRuleID identifies the default permissive local-first dogfood rule.
	DogfoodPolicyRuleID = "tillsyn-dogfood-allow-all"
	// authRequestsTableName stores local pre-session auth-request state used by tillsyn.
	authRequestsTableName      = "auth_requests"
	authRequestWriteRetryLimit = 5
	authRequestWriteRetryBase  = 25 * time.Millisecond
)

// Config configures one shared-database autent integration.
type Config struct {
	DB          *sql.DB
	TablePrefix string
	IDGenerator func() string
	Clock       func() time.Time
}

// Service wraps one autent service configured for the tillsyn runtime.
type Service struct {
	service *autent.Service
	store   *autentsqlite.Store
	db      *sql.DB
	idGen   func() string
	clock   func() time.Time
}

// AuthorizationRequest describes one MCP mutation auth check.
type AuthorizationRequest struct {
	SessionID     string
	SessionSecret string
	Action        string
	Namespace     string
	ResourceType  string
	ResourceID    string
	Context       map[string]string
}

// AuthorizationResult stores one auth decision plus the authenticated caller when allowed.
type AuthorizationResult struct {
	Caller         domain.AuthenticatedCaller
	DecisionCode   string
	DecisionReason string
	GrantID        string
}

// IssueSessionInput describes one local dogfood session issuance request.
type IssueSessionInput struct {
	PrincipalID   string
	PrincipalType string
	PrincipalName string
	ClientID      string
	ClientType    string
	ClientName    string
	TTL           time.Duration
	Metadata      map[string]string
}

// SessionListFilter narrows session inventory queries.
type SessionListFilter struct {
	SessionID   string
	ProjectID   string
	PrincipalID string
	ClientID    string
	State       string
	Limit       int
}

// NewSharedDB configures autent against the caller-owned SQLite handle.
func NewSharedDB(cfg Config) (*Service, error) {
	if cfg.DB == nil {
		return nil, fmt.Errorf("shared autent db is required: %w", autentdomain.ErrInvalidConfig)
	}
	tablePrefix := strings.TrimSpace(cfg.TablePrefix)
	if tablePrefix == "" {
		tablePrefix = DefaultTablePrefix
	}
	store, err := autentsqlite.OpenDB(cfg.DB, autentsqlite.Options{TablePrefix: tablePrefix})
	if err != nil {
		return nil, fmt.Errorf("open shared autent sqlite store: %w", err)
	}
	authService, err := autent.NewService(autent.Config{
		Repository:  store,
		Secrets:     autenttoken.OpaqueSecretManager{},
		IDGenerator: cfg.IDGenerator,
		Clock:       cfg.Clock,
	})
	if err != nil {
		return nil, fmt.Errorf("construct autent service: %w", err)
	}
	idGen := cfg.IDGenerator
	if idGen == nil {
		idGen = func() string {
			return fmt.Sprintf("authreq-%d", time.Now().UnixNano())
		}
	}
	clock := cfg.Clock
	if clock == nil {
		clock = time.Now
	}
	if err := ensureAuthRequestSchema(cfg.DB); err != nil {
		return nil, fmt.Errorf("ensure auth request schema: %w", err)
	}
	return &Service{
		service: authService,
		store:   store,
		db:      cfg.DB,
		idGen:   idGen,
		clock:   clock,
	}, nil
}

// EnsureDogfoodPolicy installs the default local dogfood allow rule when no auth policy exists yet.
func (s *Service) EnsureDogfoodPolicy(ctx context.Context) error {
	if s == nil || s.service == nil {
		return fmt.Errorf("autent service is not configured: %w", autentdomain.ErrInvalidConfig)
	}
	rules, err := s.service.ListRules(ctx)
	if err != nil {
		return fmt.Errorf("list autent rules: %w", err)
	}
	if len(rules) > 0 {
		return nil
	}
	rule, err := autentdomain.ValidateAndNormalizeRule(autentdomain.Rule{
		ID:     DogfoodPolicyRuleID,
		Effect: autentdomain.EffectAllow,
		Actions: []autentdomain.StringPattern{
			{Operator: autentdomain.MatchAny, Value: "*"},
		},
		Resources: []autentdomain.ResourcePattern{
			{
				Namespace: autentdomain.StringPattern{Operator: autentdomain.MatchAny, Value: "*"},
				Type:      autentdomain.StringPattern{Operator: autentdomain.MatchAny, Value: "*"},
				ID:        autentdomain.StringPattern{Operator: autentdomain.MatchAny, Value: "*"},
			},
		},
		Priority: 100,
	})
	if err != nil {
		return fmt.Errorf("build dogfood autent rule: %w", err)
	}
	if err := s.service.ReplaceRules(ctx, []autentdomain.Rule{rule}); err != nil {
		return fmt.Errorf("replace dogfood autent rules: %w", err)
	}
	return nil
}

// ReplaceRules validates and replaces the persisted autent policy set.
func (s *Service) ReplaceRules(ctx context.Context, rules []autentdomain.Rule) error {
	if s == nil || s.service == nil {
		return fmt.Errorf("autent service is not configured: %w", autentdomain.ErrInvalidConfig)
	}
	if err := s.service.ReplaceRules(ctx, rules); err != nil {
		return fmt.Errorf("replace autent rules: %w", err)
	}
	return nil
}

// IssueSession ensures the requested principal/client exist and returns one issued session bundle.
func (s *Service) IssueSession(ctx context.Context, in IssueSessionInput) (autent.IssuedSession, error) {
	if s == nil || s.service == nil {
		return autent.IssuedSession{}, fmt.Errorf("autent service is not configured: %w", autentdomain.ErrInvalidConfig)
	}
	principalType := normalizePrincipalType(in.PrincipalType)
	if principalType == "" {
		principalType = autentdomain.PrincipalTypeUser
	}
	if _, err := s.ensurePrincipal(ctx, strings.TrimSpace(in.PrincipalID), principalType, strings.TrimSpace(in.PrincipalName)); err != nil {
		return autent.IssuedSession{}, err
	}
	if _, err := s.ensureClient(ctx, strings.TrimSpace(in.ClientID), strings.TrimSpace(in.ClientType), strings.TrimSpace(in.ClientName)); err != nil {
		return autent.IssuedSession{}, err
	}
	issued, err := s.service.IssueSession(ctx, autent.IssueSessionInput{
		PrincipalID: strings.TrimSpace(in.PrincipalID),
		ClientID:    strings.TrimSpace(in.ClientID),
		TTL:         in.TTL,
		Metadata:    cloneContext(in.Metadata),
	})
	if err != nil {
		return autent.IssuedSession{}, fmt.Errorf("issue autent session: %w", err)
	}
	return issued, nil
}

// RevokeSession revokes one existing auth session.
func (s *Service) RevokeSession(ctx context.Context, sessionID, reason string) (autent.SessionView, error) {
	if s == nil || s.service == nil {
		return autent.SessionView{}, fmt.Errorf("autent service is not configured: %w", autentdomain.ErrInvalidConfig)
	}
	view, err := s.service.RevokeSession(ctx, strings.TrimSpace(sessionID), strings.TrimSpace(reason))
	if err != nil {
		return autent.SessionView{}, fmt.Errorf("revoke autent session: %w", err)
	}
	return view, nil
}

// ListSessions returns caller-safe auth session inventory.
func (s *Service) ListSessions(ctx context.Context, filter SessionListFilter) ([]autent.SessionView, error) {
	if s == nil || s.service == nil {
		return nil, fmt.Errorf("autent service is not configured: %w", autentdomain.ErrInvalidConfig)
	}
	projectID := strings.TrimSpace(filter.ProjectID)
	state, err := normalizeSessionStateFilter(filter.State)
	if err != nil {
		return nil, err
	}
	sessionFilter := autent.SessionFilter{
		SessionID:   strings.TrimSpace(filter.SessionID),
		PrincipalID: strings.TrimSpace(filter.PrincipalID),
		ClientID:    strings.TrimSpace(filter.ClientID),
		State:       state,
		Limit:       filter.Limit,
	}
	if projectID != "" {
		sessionFilter.Limit = 0
	}
	sessions, err := s.service.ListSessions(ctx, sessionFilter)
	if err != nil {
		return nil, fmt.Errorf("list autent sessions: %w", err)
	}
	if projectID == "" {
		return sessions, nil
	}
	filtered := make([]autent.SessionView, 0, len(sessions))
	for _, session := range sessions {
		if sessionProjectID(session) != projectID {
			continue
		}
		filtered = append(filtered, session)
		if filter.Limit > 0 && len(filtered) >= filter.Limit {
			break
		}
	}
	return filtered, nil
}

// ValidateSession returns validated principal/client/session details for one presented session.
func (s *Service) ValidateSession(ctx context.Context, sessionID, secret string) (autent.ValidatedSession, error) {
	if s == nil || s.service == nil {
		return autent.ValidatedSession{}, fmt.Errorf("autent service is not configured: %w", autentdomain.ErrInvalidConfig)
	}
	validated, err := s.service.ValidateSession(ctx, strings.TrimSpace(sessionID), strings.TrimSpace(secret))
	if err != nil {
		return autent.ValidatedSession{}, fmt.Errorf("validate autent session: %w", err)
	}
	return validated, nil
}

// CreateAuthRequest persists one new pre-session auth request.
func (s *Service) CreateAuthRequest(ctx context.Context, req domain.AuthRequest) (domain.AuthRequest, error) {
	if s == nil || s.db == nil {
		return domain.AuthRequest{}, fmt.Errorf("autent service is not configured: %w", autentdomain.ErrInvalidConfig)
	}
	req = cloneAuthRequest(req)
	if strings.TrimSpace(req.ID) == "" {
		req.ID = s.idGen()
	}
	phaseIDsJSON, err := json.Marshal(req.PhaseIDs)
	if err != nil {
		return domain.AuthRequest{}, fmt.Errorf("encode auth request phase ids: %w", err)
	}
	continuationJSON, err := json.Marshal(req.Continuation)
	if err != nil {
		return domain.AuthRequest{}, fmt.Errorf("encode auth request continuation: %w", err)
	}
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO auth_requests(
			id, project_id, branch_id, phase_ids_json, path, scope_type, scope_id,
			principal_id, principal_type, principal_role, principal_name,
			client_id, client_type, client_name,
			requested_session_ttl_seconds, approved_path, approved_session_ttl_seconds, reason, continuation_json,
			state, requested_by_actor, requested_by_type, created_at, expires_at,
			resolved_by_actor, resolved_by_type, resolved_at, resolution_note,
			issued_session_id, issued_session_secret, issued_session_expires_at
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		req.ID,
		req.ProjectID,
		req.BranchID,
		string(phaseIDsJSON),
		req.Path,
		string(req.ScopeType),
		req.ScopeID,
		req.PrincipalID,
		req.PrincipalType,
		req.PrincipalRole,
		req.PrincipalName,
		req.ClientID,
		req.ClientType,
		req.ClientName,
		int64(req.RequestedSessionTTL/time.Second),
		strings.TrimSpace(req.ApprovedPath),
		int64(req.ApprovedSessionTTL/time.Second),
		req.Reason,
		string(continuationJSON),
		string(req.State),
		req.RequestedByActor,
		string(req.RequestedByType),
		req.CreatedAt.UTC(),
		req.ExpiresAt.UTC(),
		req.ResolvedByActor,
		string(req.ResolvedByType),
		nullableTime(req.ResolvedAt),
		req.ResolutionNote,
		req.IssuedSessionID,
		req.IssuedSessionSecret,
		nullableTime(req.IssuedSessionExpiresAt),
	)
	if err != nil {
		return domain.AuthRequest{}, fmt.Errorf("insert auth request: %w", err)
	}
	return req, nil
}

// GetAuthRequest loads one auth request by id and lazily marks it expired when necessary.
func (s *Service) GetAuthRequest(ctx context.Context, requestID string) (domain.AuthRequest, error) {
	if s == nil || s.db == nil {
		return domain.AuthRequest{}, fmt.Errorf("autent service is not configured: %w", autentdomain.ErrInvalidConfig)
	}
	req, err := s.getAuthRequest(ctx, strings.TrimSpace(requestID))
	if err != nil {
		return domain.AuthRequest{}, err
	}
	return s.expireAuthRequestIfNeeded(ctx, req)
}

// ListAuthRequests lists persisted auth requests in deterministic order.
func (s *Service) ListAuthRequests(ctx context.Context, filter domain.AuthRequestListFilter) ([]domain.AuthRequest, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("autent service is not configured: %w", autentdomain.ErrInvalidConfig)
	}
	filter, err := domain.NormalizeAuthRequestListFilter(filter)
	if err != nil {
		return nil, err
	}
	query := `
		SELECT
			id, project_id, branch_id, phase_ids_json, path, scope_type, scope_id,
			principal_id, principal_type, principal_role, principal_name,
			client_id, client_type, client_name,
			requested_session_ttl_seconds, approved_path, approved_session_ttl_seconds, reason, continuation_json,
			state, requested_by_actor, requested_by_type, created_at, expires_at,
			resolved_by_actor, resolved_by_type, resolved_at, resolution_note,
			issued_session_id, issued_session_secret, issued_session_expires_at
		FROM auth_requests
		WHERE 1 = 1
	`
	args := make([]any, 0, 3)
	if filter.ProjectID != "" {
		query += ` AND project_id = ?`
		args = append(args, filter.ProjectID)
	}
	if filter.State != "" {
		query += ` AND state = ?`
		args = append(args, string(filter.State))
	}
	query += ` ORDER BY created_at DESC, id DESC`
	if filter.Limit > 0 {
		query += ` LIMIT ?`
		args = append(args, filter.Limit)
	}
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query auth requests: %w", err)
	}
	raw := make([]domain.AuthRequest, 0)
	for rows.Next() {
		req, scanErr := scanAuthRequest(rows)
		if scanErr != nil {
			_ = rows.Close()
			return nil, scanErr
		}
		raw = append(raw, req)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return nil, fmt.Errorf("iterate auth requests: %w", err)
	}
	if err := rows.Close(); err != nil {
		return nil, fmt.Errorf("close auth request rows: %w", err)
	}
	out := make([]domain.AuthRequest, 0, len(raw))
	for _, req := range raw {
		resolved, scanErr := s.expireAuthRequestIfNeeded(ctx, req)
		if scanErr != nil {
			return nil, scanErr
		}
		if filter.State != "" && domain.NormalizeAuthRequestState(resolved.State) != filter.State {
			continue
		}
		out = append(out, resolved)
	}
	return out, nil
}

// ApproveAuthRequest approves one pending request and issues the corresponding session.
func (s *Service) ApproveAuthRequest(ctx context.Context, in app.ApproveAuthRequestGatewayInput) (app.ApprovedAuthRequestResult, error) {
	if s == nil || s.db == nil {
		return app.ApprovedAuthRequestResult{}, fmt.Errorf("autent service is not configured: %w", autentdomain.ErrInvalidConfig)
	}
	req, err := s.GetAuthRequest(ctx, in.RequestID)
	if err != nil {
		return app.ApprovedAuthRequestResult{}, err
	}
	approvedPath, approvedTTL, err := approvedAuthRequestValues(req, in.PathOverride, in.TTLOverride)
	if err != nil {
		return app.ApprovedAuthRequestResult{}, err
	}
	sessionMetadata := map[string]string{
		"auth_request_id":        req.ID,
		"approved_path":          approvedPath.String(),
		"project_id":             approvedPath.ProjectID,
		"branch_id":              approvedPath.BranchID,
		"scope_type":             string(approvedPath.ScopeType),
		"scope_id":               approvedPath.ScopeID,
		"requested_principal_id": req.PrincipalID,
		"requested_client_id":    req.ClientID,
	}
	if principalRole := strings.TrimSpace(req.PrincipalRole); principalRole != "" {
		sessionMetadata["principal_role"] = principalRole
	}
	issued, err := s.IssueSession(ctx, IssueSessionInput{
		PrincipalID:   req.PrincipalID,
		PrincipalType: req.PrincipalType,
		PrincipalName: req.PrincipalName,
		ClientID:      req.ClientID,
		ClientType:    req.ClientType,
		ClientName:    req.ClientName,
		TTL:           approvedTTL,
		Metadata:      sessionMetadata,
	})
	if err != nil {
		return app.ApprovedAuthRequestResult{}, err
	}
	req.ApprovedPath = approvedPath.String()
	req.ApprovedSessionTTL = approvedTTL
	if err := req.Approve(strings.TrimSpace(in.ResolvedBy), in.ResolvedType, in.ResolutionNote, issued.Session.ID, issued.Secret, issued.Session.ExpiresAt, s.clock()); err != nil {
		return app.ApprovedAuthRequestResult{}, err
	}
	if err := s.retryAuthRequestWrite(ctx, func() error { return s.updateAuthRequest(ctx, req) }); err != nil {
		return app.ApprovedAuthRequestResult{}, err
	}
	return app.ApprovedAuthRequestResult{
		Request:       req,
		SessionSecret: issued.Secret,
	}, nil
}

// ClaimAuthRequest returns requester-visible request state and approved session secret when the continuation token matches.
func (s *Service) ClaimAuthRequest(ctx context.Context, in app.ClaimAuthRequestInput) (app.ClaimedAuthRequestResult, error) {
	if s == nil || s.db == nil {
		return app.ClaimedAuthRequestResult{}, fmt.Errorf("autent service is not configured: %w", autentdomain.ErrInvalidConfig)
	}
	requestID := strings.TrimSpace(in.RequestID)
	if requestID == "" {
		return app.ClaimedAuthRequestResult{}, fmt.Errorf("auth request id is required: %w", autentdomain.ErrInvalidID)
	}
	req, err := s.GetAuthRequest(ctx, requestID)
	if err != nil {
		return app.ClaimedAuthRequestResult{}, err
	}
	if err := authRequestClaimIdentityMatches(req, strings.TrimSpace(in.PrincipalID), strings.TrimSpace(in.ClientID)); err != nil {
		return app.ClaimedAuthRequestResult{}, err
	}
	if !authRequestResumeTokenMatches(req.Continuation, strings.TrimSpace(in.ResumeToken)) {
		return app.ClaimedAuthRequestResult{}, domain.ErrInvalidAuthContinuation
	}
	result := app.ClaimedAuthRequestResult{Request: req}
	if domain.NormalizeAuthRequestState(req.State) == domain.AuthRequestStateApproved {
		result.SessionSecret = req.IssuedSessionSecret
	}
	return result, nil
}

// DenyAuthRequest denies one pending auth request.
func (s *Service) DenyAuthRequest(ctx context.Context, requestID, resolvedBy string, resolvedType domain.ActorType, note string) (domain.AuthRequest, error) {
	req, err := s.GetAuthRequest(ctx, requestID)
	if err != nil {
		return domain.AuthRequest{}, err
	}
	if err := req.Deny(strings.TrimSpace(resolvedBy), resolvedType, note, s.clock()); err != nil {
		return domain.AuthRequest{}, err
	}
	if err := s.retryAuthRequestWrite(ctx, func() error { return s.updateAuthRequest(ctx, req) }); err != nil {
		return domain.AuthRequest{}, err
	}
	return req, nil
}

// CancelAuthRequest cancels one pending auth request.
func (s *Service) CancelAuthRequest(ctx context.Context, requestID, resolvedBy string, resolvedType domain.ActorType, note string) (domain.AuthRequest, error) {
	req, err := s.GetAuthRequest(ctx, requestID)
	if err != nil {
		return domain.AuthRequest{}, err
	}
	if err := req.Cancel(strings.TrimSpace(resolvedBy), resolvedType, note, s.clock()); err != nil {
		return domain.AuthRequest{}, err
	}
	if err := s.retryAuthRequestWrite(ctx, func() error { return s.updateAuthRequest(ctx, req) }); err != nil {
		return domain.AuthRequest{}, err
	}
	return req, nil
}

// retryAuthRequestWrite retries one transiently locked auth-request write with bounded backoff.
func (s *Service) retryAuthRequestWrite(ctx context.Context, fn func() error) error {
	if fn == nil {
		return nil
	}
	var lastErr error
	for attempt := 0; attempt < authRequestWriteRetryLimit; attempt++ {
		if err := fn(); err != nil {
			lastErr = err
			if !isRetryableSQLiteLock(err) {
				return err
			}
			if attempt == authRequestWriteRetryLimit-1 {
				break
			}
			timer := time.NewTimer(time.Duration(attempt+1) * authRequestWriteRetryBase)
			select {
			case <-ctx.Done():
				if !timer.Stop() {
					<-timer.C
				}
				return ctx.Err()
			case <-timer.C:
			}
			continue
		}
		return nil
	}
	return lastErr
}

// isRetryableSQLiteLock reports whether the error is one transient SQLite BUSY or LOCKED failure.
func isRetryableSQLiteLock(err error) bool {
	var sqliteErr *sqlite3.Error
	if !errors.As(err, &sqliteErr) {
		return false
	}
	code := sqliteErr.Code()
	return code == sqlite3.BUSY || code == sqlite3.LOCKED
}

// IssueAuthSession issues one app-facing auth session bundle through autent.
func (s *Service) IssueAuthSession(ctx context.Context, in app.AuthSessionIssueInput) (app.IssuedAuthSession, error) {
	issued, err := s.IssueSession(ctx, IssueSessionInput{
		PrincipalID:   strings.TrimSpace(in.PrincipalID),
		PrincipalType: strings.TrimSpace(in.PrincipalType),
		PrincipalName: strings.TrimSpace(in.PrincipalName),
		ClientID:      strings.TrimSpace(in.ClientID),
		ClientType:    strings.TrimSpace(in.ClientType),
		ClientName:    strings.TrimSpace(in.ClientName),
		TTL:           in.TTL,
	})
	if err != nil {
		return app.IssuedAuthSession{}, err
	}
	return app.IssuedAuthSession{
		Session: mapSessionView(issued.Session, firstNonEmpty(in.PrincipalType, "user"), in.PrincipalName, in.ClientType, in.ClientName),
		Secret:  issued.Secret,
	}, nil
}

// ListAuthSessions lists app-facing auth sessions with principal and client decoration.
func (s *Service) ListAuthSessions(ctx context.Context, filter app.AuthSessionFilter) ([]app.AuthSession, error) {
	if s == nil || s.service == nil {
		return nil, fmt.Errorf("autent service is not configured: %w", autentdomain.ErrInvalidConfig)
	}
	state, err := normalizeSessionStateFilter(filter.State)
	if err != nil {
		return nil, err
	}
	sessionFilter := autent.SessionFilter{
		SessionID:   strings.TrimSpace(filter.SessionID),
		PrincipalID: strings.TrimSpace(filter.PrincipalID),
		ClientID:    strings.TrimSpace(filter.ClientID),
		State:       state,
		Limit:       filter.Limit,
	}
	if strings.TrimSpace(filter.ProjectID) != "" {
		sessionFilter.Limit = 0
	}
	rows, err := s.service.ListSessions(ctx, sessionFilter)
	if err != nil {
		return nil, fmt.Errorf("list autent sessions: %w", err)
	}
	principals, err := s.service.ListPrincipals(ctx)
	if err != nil {
		return nil, fmt.Errorf("list autent principals: %w", err)
	}
	clients, err := s.service.ListClients(ctx)
	if err != nil {
		return nil, fmt.Errorf("list autent clients: %w", err)
	}
	principalByID := make(map[string]autentdomain.Principal, len(principals))
	for _, principal := range principals {
		principalByID[principal.ID] = principal
	}
	clientByID := make(map[string]autentdomain.Client, len(clients))
	for _, client := range clients {
		clientByID[client.ID] = client
	}
	out := make([]app.AuthSession, 0, len(rows))
	for _, row := range rows {
		if projectID := strings.TrimSpace(filter.ProjectID); projectID != "" && !sessionMatchesProject(row, projectID) {
			continue
		}
		principal := principalByID[row.PrincipalID]
		client := clientByID[row.ClientID]
		out = append(out, mapSessionView(
			row,
			string(principal.Type),
			principal.DisplayName,
			client.Type,
			client.DisplayName,
		))
		if filter.ProjectID != "" && filter.Limit > 0 && len(out) >= filter.Limit {
			break
		}
	}
	return out, nil
}

// ValidateAuthSession validates one session secret pair and returns app-facing session details.
func (s *Service) ValidateAuthSession(ctx context.Context, sessionID, secret string) (app.ValidatedAuthSession, error) {
	if s == nil || s.service == nil {
		return app.ValidatedAuthSession{}, fmt.Errorf("autent service is not configured: %w", autentdomain.ErrInvalidConfig)
	}
	validated, err := s.service.ValidateSession(ctx, strings.TrimSpace(sessionID), strings.TrimSpace(secret))
	if err != nil {
		return app.ValidatedAuthSession{}, fmt.Errorf("validate autent session: %w", err)
	}
	return app.ValidatedAuthSession{
		Session: mapSessionView(
			validated.Session,
			string(validated.Principal.Type),
			validated.Principal.DisplayName,
			validated.Client.Type,
			validated.Client.DisplayName,
		),
	}, nil
}

// RevokeAuthSession revokes one session and returns app-facing session details.
func (s *Service) RevokeAuthSession(ctx context.Context, sessionID, reason string) (app.AuthSession, error) {
	view, err := s.RevokeSession(ctx, sessionID, reason)
	if err != nil {
		return app.AuthSession{}, err
	}
	return mapSessionView(view, "", "", "", ""), nil
}

// Authorize evaluates one mutation request and returns the decision plus caller identity when allowed.
func (s *Service) Authorize(ctx context.Context, in AuthorizationRequest) (AuthorizationResult, error) {
	if s == nil || s.service == nil {
		return AuthorizationResult{}, fmt.Errorf("autent service is not configured: %w", autentdomain.ErrInvalidConfig)
	}
	resource := autentdomain.ResourceRef{
		Namespace: strings.TrimSpace(in.Namespace),
		Type:      strings.TrimSpace(in.ResourceType),
		ID:        strings.TrimSpace(in.ResourceID),
	}
	decision, err := s.service.Authorize(ctx, autent.AuthorizeInput{
		SessionID:     strings.TrimSpace(in.SessionID),
		SessionSecret: strings.TrimSpace(in.SessionSecret),
		Action:        autentdomain.Action(strings.TrimSpace(in.Action)),
		Resource:      resource,
		Context:       cloneContext(in.Context),
	})
	if err != nil {
		return AuthorizationResult{}, fmt.Errorf("authorize autent mutation: %w", err)
	}
	result := AuthorizationResult{
		DecisionCode:   string(decision.Code),
		DecisionReason: decision.Reason,
		GrantID:        strings.TrimSpace(decision.GrantID),
	}
	if decision.Code != autentdomain.DecisionAllow {
		return result, nil
	}
	validated, err := s.service.ValidateSession(ctx, strings.TrimSpace(in.SessionID), strings.TrimSpace(in.SessionSecret))
	if err != nil {
		return AuthorizationResult{}, fmt.Errorf("validate allowed autent session: %w", err)
	}
	if err := authorizeApprovedPath(validated.Session.Metadata, in.Context); err != nil {
		return AuthorizationResult{
			DecisionCode:   string(autentdomain.DecisionDeny),
			DecisionReason: "approved_path_denied",
		}, nil
	}
	result.Caller = domain.NormalizeAuthenticatedCaller(domain.AuthenticatedCaller{
		PrincipalID:   validated.Principal.ID,
		PrincipalName: validated.Principal.DisplayName,
		PrincipalType: principalTypeToActorType(validated.Principal.Type),
		SessionID:     validated.Session.ID,
	})
	return result, nil
}

// ensurePrincipal creates the requested principal when it does not already exist.
func (s *Service) ensurePrincipal(ctx context.Context, principalID string, principalType autentdomain.PrincipalType, displayName string) (autentdomain.Principal, error) {
	principalID = strings.TrimSpace(principalID)
	displayName = strings.TrimSpace(displayName)
	if principalID == "" {
		return autentdomain.Principal{}, fmt.Errorf("principal id is required: %w", autentdomain.ErrInvalidID)
	}
	if displayName == "" {
		displayName = principalID
	}
	principal, err := s.service.RegisterPrincipal(ctx, autentdomain.PrincipalInput{
		ID:          principalID,
		Type:        principalType,
		DisplayName: displayName,
	})
	if err == nil {
		return principal, nil
	}
	if !errors.Is(err, autentdomain.ErrAlreadyExists) {
		return autentdomain.Principal{}, fmt.Errorf("register autent principal %q: %w", principalID, err)
	}
	principals, listErr := s.service.ListPrincipals(ctx)
	if listErr != nil {
		return autentdomain.Principal{}, fmt.Errorf("list autent principals: %w", listErr)
	}
	for _, principal := range principals {
		if principal.ID == principalID {
			return principal, nil
		}
	}
	return autentdomain.Principal{}, fmt.Errorf("principal %q missing after already-exists result: %w", principalID, autentdomain.ErrPrincipalNotFound)
}

// ensureClient creates the requested client when it does not already exist.
func (s *Service) ensureClient(ctx context.Context, clientID, clientType, displayName string) (autentdomain.Client, error) {
	clientID = strings.TrimSpace(clientID)
	clientType = strings.TrimSpace(strings.ToLower(clientType))
	displayName = strings.TrimSpace(displayName)
	if clientID == "" {
		return autentdomain.Client{}, fmt.Errorf("client id is required: %w", autentdomain.ErrInvalidID)
	}
	if clientType == "" {
		return autentdomain.Client{}, fmt.Errorf("client type is required: %w", autentdomain.ErrInvalidClientType)
	}
	if displayName == "" {
		displayName = clientID
	}
	client, err := s.service.RegisterClient(ctx, autentdomain.ClientInput{
		ID:          clientID,
		Type:        clientType,
		DisplayName: displayName,
	})
	if err == nil {
		return client, nil
	}
	if !errors.Is(err, autentdomain.ErrAlreadyExists) {
		return autentdomain.Client{}, fmt.Errorf("register autent client %q: %w", clientID, err)
	}
	clients, listErr := s.service.ListClients(ctx)
	if listErr != nil {
		return autentdomain.Client{}, fmt.Errorf("list autent clients: %w", listErr)
	}
	for _, client := range clients {
		if client.ID == clientID {
			return client, nil
		}
	}
	return autentdomain.Client{}, fmt.Errorf("client %q missing after already-exists result: %w", clientID, autentdomain.ErrClientNotFound)
}

// normalizePrincipalType canonicalizes one user-facing principal type value.
func normalizePrincipalType(raw string) autentdomain.PrincipalType {
	return autentdomain.NormalizePrincipalType(autentdomain.PrincipalType(raw))
}

// principalTypeToActorType maps autent principal types into tillsyn mutation actor types.
func principalTypeToActorType(principalType autentdomain.PrincipalType) domain.ActorType {
	switch autentdomain.NormalizePrincipalType(principalType) {
	case autentdomain.PrincipalTypeAgent:
		return domain.ActorTypeAgent
	case autentdomain.PrincipalTypeService:
		return domain.ActorTypeAgent
	default:
		return domain.ActorTypeUser
	}
}

// cloneContext deep-copies auth request context values.
func cloneContext(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

// ensureAuthRequestSchema creates the local auth-request table used by tillsyn's pre-session approvals.
func ensureAuthRequestSchema(db *sql.DB) error {
	_, err := db.Exec(`
	CREATE TABLE IF NOT EXISTS auth_requests (
		id TEXT PRIMARY KEY,
		project_id TEXT NOT NULL,
		branch_id TEXT NOT NULL DEFAULT '',
		phase_ids_json TEXT NOT NULL DEFAULT '[]',
		path TEXT NOT NULL,
		scope_type TEXT NOT NULL,
		scope_id TEXT NOT NULL,
		principal_id TEXT NOT NULL,
		principal_type TEXT NOT NULL,
		principal_role TEXT NOT NULL DEFAULT '',
		principal_name TEXT NOT NULL DEFAULT '',
		client_id TEXT NOT NULL,
		client_type TEXT NOT NULL,
		client_name TEXT NOT NULL DEFAULT '',
		requested_session_ttl_seconds INTEGER NOT NULL,
		approved_path TEXT NOT NULL DEFAULT '',
		approved_session_ttl_seconds INTEGER NOT NULL DEFAULT 0,
		reason TEXT NOT NULL DEFAULT '',
		continuation_json TEXT NOT NULL DEFAULT '{}',
		state TEXT NOT NULL,
			requested_by_actor TEXT NOT NULL DEFAULT '',
			requested_by_type TEXT NOT NULL DEFAULT '',
			created_at TIMESTAMP NOT NULL,
			expires_at TIMESTAMP NOT NULL,
			resolved_by_actor TEXT NOT NULL DEFAULT '',
			resolved_by_type TEXT NOT NULL DEFAULT '',
			resolved_at TIMESTAMP NULL,
			resolution_note TEXT NOT NULL DEFAULT '',
			issued_session_id TEXT NOT NULL DEFAULT '',
			issued_session_secret TEXT NOT NULL DEFAULT '',
			issued_session_expires_at TIMESTAMP NULL
		);
		CREATE INDEX IF NOT EXISTS idx_auth_requests_project_state_created_at ON auth_requests(project_id, state, created_at DESC, id DESC);
	`)
	if err != nil {
		return err
	}
	alterStatements := []string{
		`ALTER TABLE auth_requests ADD COLUMN principal_role TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE auth_requests ADD COLUMN approved_path TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE auth_requests ADD COLUMN approved_session_ttl_seconds INTEGER NOT NULL DEFAULT 0`,
	}
	for _, stmt := range alterStatements {
		if _, err := db.Exec(stmt); err != nil && !isDuplicateColumnErr(err) {
			return err
		}
	}
	return nil
}

// isDuplicateColumnErr reports whether SQLite rejected an ALTER TABLE because the column already exists.
func isDuplicateColumnErr(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(strings.ToLower(err.Error()), "duplicate column name")
}

// getAuthRequest loads one raw auth request row without lazy expiration.
func (s *Service) getAuthRequest(ctx context.Context, requestID string) (domain.AuthRequest, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT
			id, project_id, branch_id, phase_ids_json, path, scope_type, scope_id,
			principal_id, principal_type, principal_role, principal_name,
			client_id, client_type, client_name,
			requested_session_ttl_seconds, approved_path, approved_session_ttl_seconds, reason, continuation_json,
			state, requested_by_actor, requested_by_type, created_at, expires_at,
			resolved_by_actor, resolved_by_type, resolved_at, resolution_note,
			issued_session_id, issued_session_secret, issued_session_expires_at
		FROM auth_requests
		WHERE id = ?
	`, strings.TrimSpace(requestID))
	req, err := scanAuthRequest(row)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.AuthRequest{}, fmt.Errorf("auth request %q not found: %w", requestID, domain.ErrInvalidID)
	}
	if err != nil {
		return domain.AuthRequest{}, fmt.Errorf("load auth request %q: %w", requestID, err)
	}
	return req, nil
}

// expireAuthRequestIfNeeded lazily persists expired pending requests.
func (s *Service) expireAuthRequestIfNeeded(ctx context.Context, req domain.AuthRequest) (domain.AuthRequest, error) {
	if !req.IsExpired(s.clock()) {
		return req, nil
	}
	if err := req.Expire(s.clock()); err != nil {
		return domain.AuthRequest{}, err
	}
	if err := s.updateAuthRequest(ctx, req); err != nil {
		return domain.AuthRequest{}, err
	}
	return req, nil
}

// updateAuthRequest persists one auth-request state transition.
func (s *Service) updateAuthRequest(ctx context.Context, req domain.AuthRequest) error {
	phaseIDsJSON, err := json.Marshal(req.PhaseIDs)
	if err != nil {
		return fmt.Errorf("encode auth request phase ids: %w", err)
	}
	continuationJSON, err := json.Marshal(req.Continuation)
	if err != nil {
		return fmt.Errorf("encode auth request continuation: %w", err)
	}
	res, err := s.db.ExecContext(ctx, `
		UPDATE auth_requests
		SET project_id = ?, branch_id = ?, phase_ids_json = ?, path = ?, scope_type = ?, scope_id = ?,
			principal_id = ?, principal_type = ?, principal_role = ?, principal_name = ?,
			client_id = ?, client_type = ?, client_name = ?,
			requested_session_ttl_seconds = ?, approved_path = ?, approved_session_ttl_seconds = ?, reason = ?, continuation_json = ?,
			state = ?, requested_by_actor = ?, requested_by_type = ?, created_at = ?, expires_at = ?,
			resolved_by_actor = ?, resolved_by_type = ?, resolved_at = ?, resolution_note = ?,
			issued_session_id = ?, issued_session_secret = ?, issued_session_expires_at = ?
		WHERE id = ?
	`,
		req.ProjectID,
		req.BranchID,
		string(phaseIDsJSON),
		req.Path,
		string(req.ScopeType),
		req.ScopeID,
		req.PrincipalID,
		req.PrincipalType,
		req.PrincipalRole,
		req.PrincipalName,
		req.ClientID,
		req.ClientType,
		req.ClientName,
		int64(req.RequestedSessionTTL/time.Second),
		strings.TrimSpace(req.ApprovedPath),
		int64(req.ApprovedSessionTTL/time.Second),
		req.Reason,
		string(continuationJSON),
		string(req.State),
		req.RequestedByActor,
		string(req.RequestedByType),
		req.CreatedAt.UTC(),
		req.ExpiresAt.UTC(),
		req.ResolvedByActor,
		string(req.ResolvedByType),
		nullableTime(req.ResolvedAt),
		req.ResolutionNote,
		req.IssuedSessionID,
		req.IssuedSessionSecret,
		nullableTime(req.IssuedSessionExpiresAt),
		req.ID,
	)
	if err != nil {
		return fmt.Errorf("update auth request: %w", err)
	}
	rows, err := res.RowsAffected()
	if err == nil && rows == 0 {
		return fmt.Errorf("auth request %q not found: %w", req.ID, domain.ErrInvalidID)
	}
	return nil
}

// scanAuthRequest decodes one auth-request row.
func scanAuthRequest(scanner interface{ Scan(...any) error }) (domain.AuthRequest, error) {
	var (
		req                  domain.AuthRequest
		phaseIDsJSON         string
		continuationJSON     string
		requestedTTLSeconds  int64
		approvedTTLSeconds   int64
		resolvedAt           sql.NullTime
		issuedSessionExpires sql.NullTime
	)
	if err := scanner.Scan(
		&req.ID,
		&req.ProjectID,
		&req.BranchID,
		&phaseIDsJSON,
		&req.Path,
		&req.ScopeType,
		&req.ScopeID,
		&req.PrincipalID,
		&req.PrincipalType,
		&req.PrincipalRole,
		&req.PrincipalName,
		&req.ClientID,
		&req.ClientType,
		&req.ClientName,
		&requestedTTLSeconds,
		&req.ApprovedPath,
		&approvedTTLSeconds,
		&req.Reason,
		&continuationJSON,
		&req.State,
		&req.RequestedByActor,
		&req.RequestedByType,
		&req.CreatedAt,
		&req.ExpiresAt,
		&req.ResolvedByActor,
		&req.ResolvedByType,
		&resolvedAt,
		&req.ResolutionNote,
		&req.IssuedSessionID,
		&req.IssuedSessionSecret,
		&issuedSessionExpires,
	); err != nil {
		return domain.AuthRequest{}, err
	}
	req.RequestedSessionTTL = time.Duration(requestedTTLSeconds) * time.Second
	req.ApprovedSessionTTL = time.Duration(approvedTTLSeconds) * time.Second
	if phaseIDsJSON != "" {
		if err := json.Unmarshal([]byte(phaseIDsJSON), &req.PhaseIDs); err != nil {
			return domain.AuthRequest{}, fmt.Errorf("decode auth request phase ids: %w", err)
		}
	}
	if continuationJSON != "" {
		if err := json.Unmarshal([]byte(continuationJSON), &req.Continuation); err != nil {
			return domain.AuthRequest{}, fmt.Errorf("decode auth request continuation: %w", err)
		}
	}
	if resolvedAt.Valid {
		ts := resolvedAt.Time.UTC()
		req.ResolvedAt = &ts
	}
	if issuedSessionExpires.Valid {
		ts := issuedSessionExpires.Time.UTC()
		req.IssuedSessionExpiresAt = &ts
	}
	if strings.TrimSpace(req.ApprovedPath) == "" && domain.NormalizeAuthRequestState(req.State) == domain.AuthRequestStateApproved {
		req.ApprovedPath = req.Path
	}
	if req.ApprovedSessionTTL <= 0 && domain.NormalizeAuthRequestState(req.State) == domain.AuthRequestStateApproved {
		req.ApprovedSessionTTL = req.RequestedSessionTTL
	}
	return cloneAuthRequest(req), nil
}

// approvedAuthRequestValues resolves one approval's effective path and TTL.
func approvedAuthRequestValues(req domain.AuthRequest, override *domain.AuthRequestPath, ttlOverride time.Duration) (domain.AuthRequestPath, time.Duration, error) {
	path, err := domain.ParseAuthRequestPath(req.Path)
	if err != nil {
		return domain.AuthRequestPath{}, 0, err
	}
	if override != nil {
		normalized, normErr := override.Normalize()
		if normErr != nil {
			return domain.AuthRequestPath{}, 0, normErr
		}
		if !authRequestPathWithin(path, normalized) {
			return domain.AuthRequestPath{}, 0, domain.ErrInvalidAuthRequestPath
		}
		path = normalized
	}
	ttl := req.RequestedSessionTTL
	if ttlOverride > 0 {
		if ttlOverride > req.RequestedSessionTTL {
			return domain.AuthRequestPath{}, 0, domain.ErrInvalidAuthRequestTTL
		}
		ttl = ttlOverride
	}
	return path, ttl, nil
}

// authRequestPathWithin reports whether candidate is equal to or narrower than requested.
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

// authRequestResumeTokenMatches reports whether one continuation payload carries the expected requester token.
func authRequestResumeTokenMatches(continuation map[string]any, want string) bool {
	want = strings.TrimSpace(want)
	if want == "" {
		return false
	}
	got, _ := continuation["resume_token"].(string)
	return strings.TrimSpace(got) == want
}

// authRequestClaimIdentityMatches reports whether one claim request matches the original requester identity.
func authRequestClaimIdentityMatches(req domain.AuthRequest, principalID, clientID string) error {
	principalID = strings.TrimSpace(principalID)
	clientID = strings.TrimSpace(clientID)
	if principalID == "" || clientID == "" {
		return domain.ErrAuthRequestClaimMismatch
	}
	if req.RequestedByActor != principalID || app.AuthRequestClaimClientIDFromContinuation(req.Continuation, req.ClientID) != clientID {
		return domain.ErrAuthRequestClaimMismatch
	}
	return nil
}

// sessionProjectID returns the project id that one session metadata payload is scoped to.
func sessionProjectID(view autent.SessionView) string {
	if projectID := strings.TrimSpace(view.Metadata["project_id"]); projectID != "" {
		if projectID == domain.AuthRequestGlobalProjectID {
			return ""
		}
		return projectID
	}
	approvedPath := strings.TrimSpace(view.Metadata["approved_path"])
	if approvedPath == "" {
		return ""
	}
	path, err := domain.ParseAuthRequestPath(approvedPath)
	if err != nil {
		return ""
	}
	return path.ProjectID
}

// sessionMatchesProject reports whether one session metadata payload applies to the requested project id.
func sessionMatchesProject(view autent.SessionView, projectID string) bool {
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		return true
	}
	approvedPath := strings.TrimSpace(view.Metadata["approved_path"])
	if approvedPath == "" {
		return sessionProjectID(view) == projectID
	}
	path, err := domain.ParseAuthRequestPath(approvedPath)
	if err != nil {
		return sessionProjectID(view) == projectID
	}
	return path.MatchesProject(projectID)
}

// authorizeApprovedPath enforces approved request path limits carried on session metadata.
func authorizeApprovedPath(sessionMetadata, contextValues map[string]string) error {
	approvedPath := strings.TrimSpace(sessionMetadata["approved_path"])
	if approvedPath == "" {
		return nil
	}
	path, err := domain.ParseAuthRequestPath(approvedPath)
	if err != nil {
		return err
	}
	contextPath, err := authContextPath(contextValues)
	if err != nil {
		return err
	}
	if !authRequestPathWithin(path, contextPath) {
		return domain.ErrInvalidScopeID
	}
	return nil
}

// authContextPath derives the narrowest project-rooted path the current mutation context proves.
func authContextPath(contextValues map[string]string) (domain.AuthRequestPath, error) {
	projectID := strings.TrimSpace(contextValues["project_id"])
	if projectID == "" {
		namespace := strings.TrimSpace(contextValues["namespace"])
		if namespaceProject := strings.TrimPrefix(namespace, "project:"); namespaceProject != namespace {
			projectID = namespaceProject
		}
	}
	if projectID == "" {
		return domain.AuthRequestPath{}, domain.ErrInvalidScopeID
	}
	scopeType := domain.ScopeLevel(strings.TrimSpace(contextValues["scope_type"]))
	scopeID := strings.TrimSpace(contextValues["scope_id"])
	branchID := strings.TrimSpace(contextValues["branch_id"])
	phaseIDs := authContextPhaseIDs(contextValues)
	switch scopeType {
	case "", domain.ScopeLevelProject:
		return domain.AuthRequestPath{ProjectID: projectID}.Normalize()
	case domain.ScopeLevelBranch:
		if scopeID == "" {
			scopeID = branchID
		}
		return domain.AuthRequestPath{ProjectID: projectID, BranchID: scopeID}.Normalize()
	case domain.ScopeLevelPhase:
		if scopeID != "" && len(phaseIDs) == 0 {
			phaseIDs = []string{scopeID}
		}
		if branchID == "" || len(phaseIDs) == 0 {
			return domain.AuthRequestPath{}, domain.ErrInvalidScopeID
		}
		return domain.AuthRequestPath{ProjectID: projectID, BranchID: branchID, PhaseIDs: phaseIDs}.Normalize()
	case domain.ScopeLevelTask, domain.ScopeLevelSubtask:
		if branchID == "" {
			return domain.AuthRequestPath{}, domain.ErrInvalidScopeID
		}
		return domain.AuthRequestPath{ProjectID: projectID, BranchID: branchID, PhaseIDs: phaseIDs}.Normalize()
	default:
		return domain.AuthRequestPath{}, domain.ErrInvalidScopeID
	}
}

// authContextPhaseIDs extracts any explicit phase lineage the mutation context proves.
func authContextPhaseIDs(contextValues map[string]string) []string {
	if raw := strings.TrimSpace(contextValues["phase_path"]); raw != "" {
		parts := strings.Split(raw, "/")
		out := make([]string, 0, len(parts))
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			out = append(out, part)
		}
		if len(out) > 0 {
			return out
		}
	}
	if phaseID := strings.TrimSpace(contextValues["phase_id"]); phaseID != "" {
		return []string{phaseID}
	}
	return nil
}

// nullableTime converts optional timestamps into SQL values.
func nullableTime(ts *time.Time) any {
	if ts == nil {
		return nil
	}
	return ts.UTC()
}

// normalizeSessionStateFilter canonicalizes user-facing session state filters.
func normalizeSessionStateFilter(raw string) (autent.SessionState, error) {
	switch strings.TrimSpace(strings.ToLower(raw)) {
	case "", "any":
		return autent.SessionStateAny, nil
	case "active":
		return autent.SessionStateActive, nil
	case "revoked":
		return autent.SessionStateRevoked, nil
	case "expired":
		return autent.SessionStateExpired, nil
	default:
		return autent.SessionState(""), fmt.Errorf("session state %q is unsupported: %w", raw, domain.ErrInvalidAuthRequestState)
	}
}

// cloneAuthRequest deep-copies nested auth-request fields before callers mutate them.
func cloneAuthRequest(req domain.AuthRequest) domain.AuthRequest {
	req.PhaseIDs = append([]string(nil), req.PhaseIDs...)
	req.Continuation = cloneJSONMap(req.Continuation)
	return req
}

// cloneJSONMap deep-copies one JSON-compatible auth-request metadata object.
func cloneJSONMap(in map[string]any) map[string]any {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]any, len(in))
	for key, value := range in {
		out[key] = cloneJSONValue(value)
	}
	return out
}

// cloneJSONValue deep-copies one JSON-compatible nested auth-request metadata value.
func cloneJSONValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		return cloneJSONMap(typed)
	case []any:
		out := make([]any, 0, len(typed))
		for _, item := range typed {
			out = append(out, cloneJSONValue(item))
		}
		return out
	default:
		return typed
	}
}

// mapSessionView converts one autent session view into the app-facing session shape.
func mapSessionView(view autent.SessionView, principalType, principalName, clientType, clientName string) app.AuthSession {
	lastSeen := view.LastSeenAt.UTC()
	session := app.AuthSession{
		SessionID:        strings.TrimSpace(view.ID),
		ProjectID:        sessionProjectID(view),
		AuthRequestID:    strings.TrimSpace(view.Metadata["auth_request_id"]),
		ApprovedPath:     strings.TrimSpace(view.Metadata["approved_path"]),
		PrincipalID:      strings.TrimSpace(view.PrincipalID),
		PrincipalType:    strings.TrimSpace(principalType),
		PrincipalRole:    strings.TrimSpace(view.Metadata["principal_role"]),
		PrincipalName:    strings.TrimSpace(principalName),
		ClientID:         strings.TrimSpace(view.ClientID),
		ClientType:       strings.TrimSpace(clientType),
		ClientName:       strings.TrimSpace(clientName),
		IssuedAt:         view.IssuedAt.UTC(),
		ExpiresAt:        view.ExpiresAt.UTC(),
		LastValidatedAt:  nil,
		RevokedAt:        view.RevokedAt,
		RevocationReason: strings.TrimSpace(view.RevocationReason),
	}
	if !lastSeen.IsZero() {
		session.LastValidatedAt = &lastSeen
	}
	return session
}

// firstNonEmpty returns the first non-empty trimmed value in order.
func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}
