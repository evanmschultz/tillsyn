package common

import (
	"context"
	"errors"

	"github.com/evanmschultz/tillsyn/internal/domain"
)

// ErrSessionRequired reports that one mutating request did not present a required auth session.
var ErrSessionRequired = errors.New("session required")

// ErrInvalidAuthentication reports that one presented auth session or secret is invalid.
var ErrInvalidAuthentication = errors.New("invalid authentication")

// ErrSessionExpired reports that one presented auth session is expired.
var ErrSessionExpired = errors.New("session expired")

// ErrAuthorizationDenied reports that a valid caller was denied by auth
// policy. Drop 4a Wave 3 W3.1 promoted the canonical sentinel into
// `internal/domain/errors.go` so the app layer's orch-self-approval gate
// can return it without crossing the adapter import boundary; this alias
// preserves source compatibility for existing
// `errors.Is(err, common.ErrAuthorizationDenied)` callers.
var ErrAuthorizationDenied = domain.ErrAuthorizationDenied

// ErrGrantRequired reports that a valid caller requires explicit grant approval before proceeding.
var ErrGrantRequired = errors.New("grant required")

// MutationAuthorizationRequest describes one auth check for a mutating MCP request.
type MutationAuthorizationRequest struct {
	SessionID     string
	SessionSecret string
	Action        string
	Namespace     string
	ResourceType  string
	ResourceID    string
	Context       map[string]string
}

// MutationAuthorizer resolves one authenticated caller for a mutating request.
type MutationAuthorizer interface {
	AuthorizeMutation(context.Context, MutationAuthorizationRequest) (domain.AuthenticatedCaller, error)
}
