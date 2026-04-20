// Package httpapi provides the REST HTTP adapter for the server surfaces.
package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/charmbracelet/log"
	"github.com/evanmschultz/tillsyn/internal/adapters/server/common"
	"github.com/evanmschultz/tillsyn/internal/domain"
)

// maxRequestBodyBytes limits decoded JSON payload size for fail-closed request handling.
const maxRequestBodyBytes int64 = 1 << 20

// Handler serves the versioned API subrouter mounted under `/api/v1`.
type Handler struct {
	captureState common.CaptureStateReader
	attention    common.AttentionService
	auth         common.MutationAuthorizer
}

// APIError represents one structured API failure response.
type APIError struct {
	Code    string         `json:"code"`
	Message string         `json:"message"`
	Hint    string         `json:"hint,omitempty"`
	Context map[string]any `json:"context,omitempty"`
}

// ErrorEnvelope wraps one structured API error.
type ErrorEnvelope struct {
	Error APIError `json:"error"`
}

// raiseAttentionItemPayload captures HTTP attention-create inputs plus session auth.
type raiseAttentionItemPayload struct {
	ProjectID          string `json:"project_id"`
	ScopeType          string `json:"scope_type"`
	ScopeID            string `json:"scope_id"`
	Kind               string `json:"kind"`
	Summary            string `json:"summary"`
	BodyMarkdown       string `json:"body_markdown,omitempty"`
	RequiresUserAction bool   `json:"requires_user_action"`
	SessionID          string `json:"session_id"`
	SessionSecret      string `json:"session_secret"`
	AgentInstanceID    string `json:"agent_instance_id,omitempty"`
	LeaseToken         string `json:"lease_token,omitempty"`
	OverrideToken      string `json:"override_token,omitempty"`
}

// resolveAttentionItemPayload captures HTTP attention-resolve inputs plus session auth.
type resolveAttentionItemPayload struct {
	Reason          string `json:"reason,omitempty"`
	SessionID       string `json:"session_id"`
	SessionSecret   string `json:"session_secret"`
	AgentInstanceID string `json:"agent_instance_id,omitempty"`
	LeaseToken      string `json:"lease_token,omitempty"`
	OverrideToken   string `json:"override_token,omitempty"`
}

// httpMutationGuardArgs stores the local lease tuple used after session auth succeeds.
type httpMutationGuardArgs struct {
	AgentInstanceID string
	LeaseToken      string
	OverrideToken   string
}

// NewHandler constructs one HTTP API adapter from capture and optional attention services.
func NewHandler(captureState common.CaptureStateReader, attention common.AttentionService) *Handler {
	var auth common.MutationAuthorizer
	if authorizer, ok := attention.(common.MutationAuthorizer); ok {
		auth = authorizer
	}
	return &Handler{
		captureState: captureState,
		attention:    attention,
		auth:         auth,
	}
}

// ServeHTTP routes one versioned API request to the matching handler.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := normalizePath(r.URL.Path)
	switch {
	case path == "capture_state":
		if r.Method != http.MethodGet {
			writeMethodNotAllowed(w, http.MethodGet)
			return
		}
		h.handleCaptureState(w, r)
		return
	case path == "attention/items":
		switch r.Method {
		case http.MethodGet:
			h.handleListAttentionItems(w, r)
		case http.MethodPost:
			h.handleRaiseAttentionItem(w, r)
		default:
			writeMethodNotAllowed(w, http.MethodGet, http.MethodPost)
		}
		return
	default:
		itemID, ok := resolveAttentionItemID(path)
		if !ok {
			writeJSONError(w, http.StatusNotFound, APIError{
				Code:    "not_found",
				Message: "endpoint not found",
			})
			return
		}
		if r.Method != http.MethodPost {
			writeMethodNotAllowed(w, http.MethodPost)
			return
		}
		h.handleResolveAttentionItem(w, r, itemID)
	}
}

// handleCaptureState serves GET `/capture_state`.
func (h *Handler) handleCaptureState(w http.ResponseWriter, r *http.Request) {
	if h.captureState == nil {
		writeJSONError(w, http.StatusServiceUnavailable, APIError{
			Code:    "service_unavailable",
			Message: "capture_state service is not configured",
		})
		return
	}
	req := common.CaptureStateRequest{
		ProjectID: r.URL.Query().Get("project_id"),
		ScopeType: r.URL.Query().Get("scope_type"),
		ScopeID:   r.URL.Query().Get("scope_id"),
		View:      r.URL.Query().Get("view"),
	}
	captureState, err := h.captureState.CaptureState(r.Context(), req)
	if err != nil {
		writeErrorFrom(w, err)
		return
	}
	writeJSON(w, http.StatusOK, captureState)
}

// handleListAttentionItems serves GET `/attention/items`.
func (h *Handler) handleListAttentionItems(w http.ResponseWriter, r *http.Request) {
	if h.attention == nil {
		writeJSONError(w, http.StatusNotImplemented, APIError{
			Code:    "not_implemented",
			Message: "attention APIs are not available",
		})
		return
	}
	req := common.ListAttentionItemsRequest{
		ProjectID: strings.TrimSpace(r.URL.Query().Get("project_id")),
		ScopeType: strings.TrimSpace(r.URL.Query().Get("scope_type")),
		ScopeID:   strings.TrimSpace(r.URL.Query().Get("scope_id")),
		State:     strings.TrimSpace(r.URL.Query().Get("state")),
	}
	if req.ProjectID == "" {
		writeJSONError(w, http.StatusBadRequest, APIError{
			Code:    "invalid_request",
			Message: "project_id is required",
		})
		return
	}
	items, err := h.attention.ListAttentionItems(r.Context(), req)
	if err != nil {
		writeErrorFrom(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"items": items,
	})
}

// handleRaiseAttentionItem serves POST `/attention/items`.
func (h *Handler) handleRaiseAttentionItem(w http.ResponseWriter, r *http.Request) {
	if h.attention == nil {
		writeJSONError(w, http.StatusNotImplemented, APIError{
			Code:    "not_implemented",
			Message: "attention APIs are not available",
		})
		return
	}

	var payload raiseAttentionItemPayload
	if err := decodeJSONBody(r.Context(), w, r, &payload); err != nil {
		writeErrorFrom(w, err)
		return
	}
	caller, err := authorizeHTTPMutation(
		r.Context(),
		h.auth,
		payload.SessionID,
		payload.SessionSecret,
		"raise_attention_item",
		"project:"+strings.TrimSpace(payload.ProjectID),
		"attention_item",
		"new",
		map[string]string{
			"project_id": strings.TrimSpace(payload.ProjectID),
			"scope_type": strings.TrimSpace(payload.ScopeType),
			"scope_id":   strings.TrimSpace(payload.ScopeID),
			"kind":       strings.TrimSpace(payload.Kind),
		},
	)
	if err != nil {
		writeErrorFrom(w, err)
		return
	}
	actor, err := buildAuthenticatedHTTPActor(caller, httpMutationGuardArgs{
		AgentInstanceID: payload.AgentInstanceID,
		LeaseToken:      payload.LeaseToken,
		OverrideToken:   payload.OverrideToken,
	})
	if err != nil {
		writeErrorFrom(w, err)
		return
	}
	req := common.RaiseAttentionItemRequest{
		ProjectID:          strings.TrimSpace(payload.ProjectID),
		ScopeType:          strings.TrimSpace(payload.ScopeType),
		ScopeID:            strings.TrimSpace(payload.ScopeID),
		Kind:               strings.TrimSpace(payload.Kind),
		Summary:            strings.TrimSpace(payload.Summary),
		BodyMarkdown:       strings.TrimSpace(payload.BodyMarkdown),
		RequiresUserAction: payload.RequiresUserAction,
		Actor:              actor,
	}
	item, err := h.attention.RaiseAttentionItem(r.Context(), req)
	if err != nil {
		writeErrorFrom(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

// handleResolveAttentionItem serves POST `/attention/items/{id}/resolve`.
func (h *Handler) handleResolveAttentionItem(w http.ResponseWriter, r *http.Request, itemID string) {
	if h.attention == nil {
		writeJSONError(w, http.StatusNotImplemented, APIError{
			Code:    "not_implemented",
			Message: "attention APIs are not available",
		})
		return
	}

	req := common.ResolveAttentionItemRequest{
		ID: itemID,
	}
	var payload resolveAttentionItemPayload
	if err := decodeOptionalJSONBody(r.Context(), w, r, &payload); err != nil {
		writeErrorFrom(w, err)
		return
	}
	caller, err := authorizeHTTPMutation(
		r.Context(),
		h.auth,
		payload.SessionID,
		payload.SessionSecret,
		"resolve_attention_item",
		"attention",
		"attention_item",
		strings.TrimSpace(itemID),
		nil,
	)
	if err != nil {
		writeErrorFrom(w, err)
		return
	}
	actor, err := buildAuthenticatedHTTPActor(caller, httpMutationGuardArgs{
		AgentInstanceID: payload.AgentInstanceID,
		LeaseToken:      payload.LeaseToken,
		OverrideToken:   payload.OverrideToken,
	})
	if err != nil {
		writeErrorFrom(w, err)
		return
	}
	if trimmed := strings.TrimSpace(payload.Reason); trimmed != "" {
		req.Reason = trimmed
	}
	req.Actor = actor

	item, err := h.attention.ResolveAttentionItem(r.Context(), req)
	if err != nil {
		writeErrorFrom(w, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

// authorizeHTTPMutation validates one authenticated session for HTTP write routes.
func authorizeHTTPMutation(
	ctx context.Context,
	authorizer common.MutationAuthorizer,
	sessionID string,
	sessionSecret string,
	action string,
	namespace string,
	resourceType string,
	resourceID string,
	authContext map[string]string,
) (domain.AuthenticatedCaller, error) {
	if authorizer == nil {
		return domain.AuthenticatedCaller{}, fmt.Errorf("mutation authorizer is unavailable")
	}
	return authorizer.AuthorizeMutation(ctx, common.MutationAuthorizationRequest{
		SessionID:     strings.TrimSpace(sessionID),
		SessionSecret: strings.TrimSpace(sessionSecret),
		Action:        strings.TrimSpace(action),
		Namespace:     strings.TrimSpace(namespace),
		ResourceType:  strings.TrimSpace(resourceType),
		ResourceID:    strings.TrimSpace(resourceID),
		Context:       authContext,
	})
}

// buildAuthenticatedHTTPActor derives the app-level actor tuple from one authenticated caller.
func buildAuthenticatedHTTPActor(caller domain.AuthenticatedCaller, guard httpMutationGuardArgs) (common.ActorLeaseTuple, error) {
	caller = domain.NormalizeAuthenticatedCaller(caller)
	if caller.IsZero() {
		return common.ActorLeaseTuple{}, fmt.Errorf("authenticated caller is required: %w", common.ErrInvalidCaptureStateRequest)
	}
	actor := common.ActorLeaseTuple{
		ActorID:   caller.PrincipalID,
		ActorName: caller.PrincipalName,
		ActorType: string(caller.PrincipalType),
	}
	guard.AgentInstanceID = strings.TrimSpace(guard.AgentInstanceID)
	guard.LeaseToken = strings.TrimSpace(guard.LeaseToken)
	guard.OverrideToken = strings.TrimSpace(guard.OverrideToken)
	hasGuardTuple := guard.AgentInstanceID != "" || guard.LeaseToken != "" || guard.OverrideToken != ""
	if caller.PrincipalType != domain.ActorTypeAgent {
		if hasGuardTuple {
			return common.ActorLeaseTuple{}, fmt.Errorf("guarded mutation tuple requires an authenticated agent session; remove agent_instance_id/lease_token to act as a human or claim/validate an approved agent session first: %w", common.ErrInvalidCaptureStateRequest)
		}
		return actor, nil
	}
	if guard.AgentInstanceID == "" || guard.LeaseToken == "" {
		return common.ActorLeaseTuple{}, fmt.Errorf("agent_instance_id and lease_token are required for authenticated agent mutations: %w", common.ErrInvalidCaptureStateRequest)
	}
	actor.AgentName = firstNonEmptyString(caller.PrincipalName, caller.PrincipalID)
	actor.AgentInstanceID = guard.AgentInstanceID
	actor.LeaseToken = guard.LeaseToken
	actor.OverrideToken = guard.OverrideToken
	return actor, nil
}

// resolveAttentionItemID parses `/attention/items/{id}/resolve` and returns `{id}`.
func resolveAttentionItemID(path string) (string, bool) {
	const (
		prefix = "attention/items/"
		suffix = "/resolve"
	)
	if !strings.HasPrefix(path, prefix) || !strings.HasSuffix(path, suffix) {
		return "", false
	}
	id := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(path, prefix), suffix))
	if id == "" || strings.Contains(id, "/") {
		return "", false
	}
	return id, true
}

// normalizePath canonicalizes one request path for route matching.
func normalizePath(path string) string {
	path = strings.TrimSpace(path)
	path = strings.Trim(path, "/")
	return path
}

// writeErrorFrom maps adapter errors into structured HTTP responses.
func writeErrorFrom(w http.ResponseWriter, err error) {
	mapped := mapHTTPError(err)
	log.Error(
		"http api error mapped",
		"transport",
		"http",
		"error_class",
		mapped.Class,
		"error_code",
		mapped.APIError.Code,
		"status_code",
		mapped.StatusCode,
		"err",
		err,
	)
	writeJSONError(w, mapped.StatusCode, mapped.APIError)
}

// httpErrorMapping captures one mapped HTTP error classification and response payload.
type httpErrorMapping struct {
	Class      string
	StatusCode int
	APIError   APIError
}

// mapHTTPError converts one adapter error into deterministic HTTP API error metadata.
func mapHTTPError(err error) httpErrorMapping {
	switch {
	case err == nil:
		return httpErrorMapping{
			Class:      "internal",
			StatusCode: http.StatusInternalServerError,
			APIError: APIError{
				Code:    "internal_error",
				Message: "unknown error",
			},
		}
	case errors.Is(err, common.ErrBootstrapRequired):
		return httpErrorMapping{
			Class:      "bootstrap",
			StatusCode: http.StatusConflict,
			APIError: APIError{
				Code:    "bootstrap_required",
				Message: err.Error(),
				Hint:    "Create the first project before calling capture_state.",
			},
		}
	case errors.Is(err, common.ErrGuardrailViolation):
		return httpErrorMapping{
			Class:      "guardrail",
			StatusCode: http.StatusConflict,
			APIError: APIError{
				Code:    "guardrail_failed",
				Message: err.Error(),
			},
		}
	case errors.Is(err, common.ErrSessionRequired):
		return httpErrorMapping{
			Class:      "auth",
			StatusCode: http.StatusUnauthorized,
			APIError: APIError{
				Code:    "session_required",
				Message: err.Error(),
			},
		}
	case errors.Is(err, common.ErrInvalidAuthentication):
		return httpErrorMapping{
			Class:      "auth",
			StatusCode: http.StatusUnauthorized,
			APIError: APIError{
				Code:    "invalid_auth",
				Message: err.Error(),
			},
		}
	case errors.Is(err, common.ErrSessionExpired):
		return httpErrorMapping{
			Class:      "auth",
			StatusCode: http.StatusUnauthorized,
			APIError: APIError{
				Code:    "session_expired",
				Message: err.Error(),
			},
		}
	case errors.Is(err, common.ErrAuthorizationDenied):
		return httpErrorMapping{
			Class:      "auth",
			StatusCode: http.StatusForbidden,
			APIError: APIError{
				Code:    "auth_denied",
				Message: err.Error(),
			},
		}
	case errors.Is(err, common.ErrGrantRequired):
		return httpErrorMapping{
			Class:      "auth",
			StatusCode: http.StatusForbidden,
			APIError: APIError{
				Code:    "grant_required",
				Message: err.Error(),
			},
		}
	case errors.Is(err, common.ErrNotFound):
		return httpErrorMapping{
			Class:      "not_found",
			StatusCode: http.StatusNotFound,
			APIError: APIError{
				Code:    "not_found",
				Message: err.Error(),
			},
		}
	case errors.Is(err, common.ErrInvalidCaptureStateRequest), errors.Is(err, common.ErrUnsupportedScope):
		return httpErrorMapping{
			Class:      "invalid",
			StatusCode: http.StatusBadRequest,
			APIError: APIError{
				Code:    "invalid_request",
				Message: err.Error(),
			},
		}
	case errors.Is(err, common.ErrAttentionUnavailable):
		return httpErrorMapping{
			Class:      "not_implemented",
			StatusCode: http.StatusNotImplemented,
			APIError: APIError{
				Code:    "not_implemented",
				Message: err.Error(),
			},
		}
	default:
		return httpErrorMapping{
			Class:      "internal",
			StatusCode: http.StatusInternalServerError,
			APIError: APIError{
				Code:    "internal_error",
				Message: err.Error(),
			},
		}
	}
}

// firstNonEmptyString returns the first trimmed non-empty string in order.
func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

// writeMethodNotAllowed writes a structured 405 response with `Allow` headers.
func writeMethodNotAllowed(w http.ResponseWriter, methods ...string) {
	if len(methods) > 0 {
		w.Header().Set("Allow", strings.Join(methods, ", "))
	}
	writeJSONError(w, http.StatusMethodNotAllowed, APIError{
		Code:    "method_not_allowed",
		Message: "method not allowed",
	})
}

// writeJSONError writes one structured error envelope.
func writeJSONError(w http.ResponseWriter, statusCode int, apiErr APIError) {
	writeJSON(w, statusCode, ErrorEnvelope{Error: apiErr})
}

// writeJSON writes one JSON response envelope.
func writeJSON(w http.ResponseWriter, statusCode int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		http.Error(w, fmt.Sprintf(`{"error":{"code":"encode_error","message":"%s"}}`, err.Error()), http.StatusInternalServerError)
	}
}

// decodeJSONBody decodes one required JSON request body with strict shape checks.
func decodeJSONBody(ctx context.Context, w http.ResponseWriter, r *http.Request, out any) error {
	reader := http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)
	defer reader.Close()

	decoder := json.NewDecoder(reader)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(out); err != nil {
		return fmt.Errorf("decode request body: %w", errors.Join(common.ErrInvalidCaptureStateRequest, err))
	}
	// Reject trailing payloads so malformed JSON bodies fail closed.
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return fmt.Errorf("decode request body: trailing content: %w", common.ErrInvalidCaptureStateRequest)
	}
	select {
	case <-ctx.Done():
		return fmt.Errorf("request canceled: %w", ctx.Err())
	default:
		return nil
	}
}

// decodeOptionalJSONBody decodes one optional JSON body and ignores empty payloads.
func decodeOptionalJSONBody(ctx context.Context, w http.ResponseWriter, r *http.Request, out any) error {
	reader := http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)
	defer reader.Close()

	decoder := json.NewDecoder(reader)
	decoder.DisallowUnknownFields()
	err := decoder.Decode(out)
	if err == nil {
		select {
		case <-ctx.Done():
			return fmt.Errorf("request canceled: %w", ctx.Err())
		default:
			return nil
		}
	}
	if errors.Is(err, io.EOF) {
		return nil
	}
	return fmt.Errorf("decode request body: %w", errors.Join(common.ErrInvalidCaptureStateRequest, err))
}
