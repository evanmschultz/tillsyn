package mcpapi

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"

	"github.com/hylla/tillsyn/internal/adapters/server/common"
	"github.com/mark3labs/mcp-go/mcp"
)

type mcpAuthContextStore struct {
	enabled     bool
	mu          sync.RWMutex
	byID        map[string]storedMCPAuthContext
	bySessionID map[string]string
}

type storedMCPAuthContext struct {
	sessionID     string
	sessionSecret string
}

type (
	mcpAuthContextStoreKey    struct{}
	mcpAuthContextIDKey       struct{}
	mcpActingAuthContextIDKey struct{}
)

func newMCPAuthContextStore(enabled bool) *mcpAuthContextStore {
	return &mcpAuthContextStore{
		enabled:     enabled,
		byID:        make(map[string]storedMCPAuthContext),
		bySessionID: make(map[string]string),
	}
}

func (s *mcpAuthContextStore) Bind(sessionID, sessionSecret string) (string, error) {
	if s == nil || !s.enabled {
		return "", nil
	}
	sessionID = strings.TrimSpace(sessionID)
	sessionSecret = strings.TrimSpace(sessionSecret)
	if sessionID == "" || sessionSecret == "" {
		return "", fmt.Errorf("session_id and session_secret are required to bind one auth context")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if authContextID := strings.TrimSpace(s.bySessionID[sessionID]); authContextID != "" {
		s.byID[authContextID] = storedMCPAuthContext{sessionID: sessionID, sessionSecret: sessionSecret}
		return authContextID, nil
	}

	for {
		authContextID, err := newMCPAuthContextID()
		if err != nil {
			return "", err
		}
		if _, exists := s.byID[authContextID]; exists {
			continue
		}
		s.byID[authContextID] = storedMCPAuthContext{sessionID: sessionID, sessionSecret: sessionSecret}
		s.bySessionID[sessionID] = authContextID
		return authContextID, nil
	}
}

func (s *mcpAuthContextStore) Resolve(authContextID, expectedSessionID string) (string, string, error) {
	if s == nil || !s.enabled {
		return "", "", fmt.Errorf("%w: auth contexts are unavailable on this MCP transport", common.ErrInvalidAuthentication)
	}
	authContextID = strings.TrimSpace(authContextID)
	expectedSessionID = strings.TrimSpace(expectedSessionID)
	if authContextID == "" {
		return "", "", fmt.Errorf("%w: auth_context_id is required", common.ErrInvalidAuthentication)
	}

	s.mu.RLock()
	stored, ok := s.byID[authContextID]
	s.mu.RUnlock()
	if !ok {
		return "", "", fmt.Errorf("%w: auth_context_id %q was not found", common.ErrInvalidAuthentication, authContextID)
	}
	if expectedSessionID != "" && stored.sessionID != expectedSessionID {
		return "", "", fmt.Errorf("%w: auth_context_id %q is bound to session %q, not %q", common.ErrInvalidAuthentication, authContextID, stored.sessionID, expectedSessionID)
	}
	return stored.sessionID, stored.sessionSecret, nil
}

func newMCPAuthContextID() (string, error) {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", fmt.Errorf("generate auth context id: %w", err)
	}
	return "authctx-" + hex.EncodeToString(raw[:]), nil
}

func withMCPToolAuthRuntime(ctx context.Context, authContexts *mcpAuthContextStore, req mcp.CallToolRequest) context.Context {
	if authContexts != nil {
		ctx = context.WithValue(ctx, mcpAuthContextStoreKey{}, authContexts)
	}
	if authContextID := strings.TrimSpace(req.GetString("auth_context_id", "")); authContextID != "" {
		ctx = context.WithValue(ctx, mcpAuthContextIDKey{}, authContextID)
	}
	if authContextID := strings.TrimSpace(req.GetString("acting_auth_context_id", "")); authContextID != "" {
		ctx = context.WithValue(ctx, mcpActingAuthContextIDKey{}, authContextID)
	}
	return ctx
}

func bindMCPAuthContext(ctx context.Context, sessionID, sessionSecret string) (string, error) {
	store, _ := ctx.Value(mcpAuthContextStoreKey{}).(*mcpAuthContextStore)
	if store == nil {
		return "", nil
	}
	return store.Bind(sessionID, sessionSecret)
}

func resolveMCPMutationAuth(ctx context.Context, auth mcpSessionAuthArgs) (mcpSessionAuthArgs, error) {
	auth.SessionID = strings.TrimSpace(auth.SessionID)
	auth.SessionSecret = strings.TrimSpace(auth.SessionSecret)
	if auth.SessionSecret != "" {
		return auth, nil
	}
	authContextID, _ := ctx.Value(mcpAuthContextIDKey{}).(string)
	authContextID = strings.TrimSpace(authContextID)
	if authContextID == "" {
		return auth, nil
	}
	store, _ := ctx.Value(mcpAuthContextStoreKey{}).(*mcpAuthContextStore)
	if store == nil {
		return mcpSessionAuthArgs{}, fmt.Errorf("%w: auth_context_id is unavailable on this MCP transport", common.ErrInvalidAuthentication)
	}
	sessionID, sessionSecret, err := store.Resolve(authContextID, auth.SessionID)
	if err != nil {
		return mcpSessionAuthArgs{}, err
	}
	auth.SessionID = sessionID
	auth.SessionSecret = sessionSecret
	return auth, nil
}

func resolveMCPActingSessionAuth(ctx context.Context, sessionID, sessionSecret string) (string, string, error) {
	sessionID = strings.TrimSpace(sessionID)
	sessionSecret = strings.TrimSpace(sessionSecret)
	if sessionSecret != "" {
		return sessionID, sessionSecret, nil
	}
	authContextID, _ := ctx.Value(mcpActingAuthContextIDKey{}).(string)
	authContextID = strings.TrimSpace(authContextID)
	if authContextID == "" {
		return sessionID, sessionSecret, nil
	}
	store, _ := ctx.Value(mcpAuthContextStoreKey{}).(*mcpAuthContextStore)
	if store == nil {
		return "", "", fmt.Errorf("%w: acting_auth_context_id is unavailable on this MCP transport", common.ErrInvalidAuthentication)
	}
	return store.Resolve(authContextID, sessionID)
}
