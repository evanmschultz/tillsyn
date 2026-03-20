// Package autentauth adapts autent into tillsyn's local shared-DB runtime.
package autentauth

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	autent "github.com/evanmschultz/autent"
	autentdomain "github.com/evanmschultz/autent/domain"
	autentsqlite "github.com/evanmschultz/autent/sqlite"
	autenttoken "github.com/evanmschultz/autent/token"
	"github.com/hylla/tillsyn/internal/domain"
)

const (
	// DefaultTablePrefix scopes autent tables inside the shared tillsyn database.
	DefaultTablePrefix = "autent_"
	// DogfoodPolicyRuleID identifies the default permissive local-first dogfood rule.
	DogfoodPolicyRuleID = "tillsyn-dogfood-allow-all"
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
	Caller        domain.AuthenticatedCaller
	DecisionCode  string
	DecisionReason string
	GrantID       string
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
	return &Service{
		service: authService,
		store:   store,
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
