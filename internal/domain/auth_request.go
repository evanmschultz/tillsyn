package domain

import (
	"fmt"
	"slices"
	"strings"
	"time"
)

// AuthRequestState identifies one persisted auth-request lifecycle state.
type AuthRequestState string

// AuthRequestState values.
const (
	AuthRequestStatePending  AuthRequestState = "pending"
	AuthRequestStateApproved AuthRequestState = "approved"
	AuthRequestStateDenied   AuthRequestState = "denied"
	AuthRequestStateCanceled AuthRequestState = "canceled"
	AuthRequestStateExpired  AuthRequestState = "expired"
)

// validAuthRequestStates stores supported auth-request states.
var validAuthRequestStates = []AuthRequestState{
	AuthRequestStatePending,
	AuthRequestStateApproved,
	AuthRequestStateDenied,
	AuthRequestStateCanceled,
	AuthRequestStateExpired,
}

// AuthRequestPath stores one canonical auth-request scope path.
type AuthRequestPath struct {
	Kind       AuthRequestPathKind
	ProjectID  string
	ProjectIDs []string
	BranchID   string
	PhaseIDs   []string
	ScopeType  ScopeLevel
	ScopeID    string
}

// AuthRequestPathKind identifies the canonical auth-scope shape.
type AuthRequestPathKind string

// Auth request path kinds.
const (
	AuthRequestPathKindProject  AuthRequestPathKind = "project"
	AuthRequestPathKindProjects AuthRequestPathKind = "projects"
	AuthRequestPathKindGlobal   AuthRequestPathKind = "global"
)

// AuthRequestGlobalProjectID is the internal sentinel project id used for global auth-request routing.
const AuthRequestGlobalProjectID = "__global__"

// AuthRequestRole identifies one auth-request agent role for gatekeeping policy.
//
// This is a closed enum distinct from two other role-like axes in the
// codebase:
//
//   - domain.Role — the action-item role enum (builder | qa-proof | qa-falsification | …)
//     attached to ActionItem.Metadata.Role for cascade dispatch lookup.
//   - action_items.kind — the closed 12-kind enum (plan | build | research | …)
//     describing the work the action item carries.
//
// AuthRequestRole names the agent class for the auth-session a caller wants
// issued; it is consumed by the orch-self-approval gate (Drop 4a Wave 3) to
// decide whether an in-orch cascade approval may issue a session for a
// requesting subagent. Three orthogonal axes — the same shape as Drop 3 L7
// where steward principal_type became a tillsyn axis distinct from autent's
// closed enum.
//
// Drop 4a Wave 3 (W3.1) widened the closed set from 4 values
// (orchestrator | builder | qa | research) to 7 (orchestrator | planner |
// qa-proof | qa-falsification | builder | research | commit). The bare "qa"
// constant survives as a deprecated alias that NormalizeAuthRequestRole
// REJECTS — callers must pick qa-proof or qa-falsification explicitly.
type AuthRequestRole string

// Auth request role values.
const (
	AuthRequestRoleOrchestrator    AuthRequestRole = "orchestrator"
	AuthRequestRolePlanner         AuthRequestRole = "planner"
	AuthRequestRoleQAProof         AuthRequestRole = "qa-proof"
	AuthRequestRoleQAFalsification AuthRequestRole = "qa-falsification"
	AuthRequestRoleBuilder         AuthRequestRole = "builder"
	AuthRequestRoleResearch        AuthRequestRole = "research"
	AuthRequestRoleCommit          AuthRequestRole = "commit"

	// AuthRequestRoleQA is a DEPRECATED alias kept for source compatibility.
	// NormalizeAuthRequestRole rejects bare "qa" — callers must choose
	// qa-proof or qa-falsification explicitly per Drop 4a Wave 3 W3.1.
	AuthRequestRoleQA AuthRequestRole = "qa"

	// AuthRequestRoleSubagent preserves the legacy subagent token as an alias for builder.
	AuthRequestRoleSubagent AuthRequestRole = AuthRequestRoleBuilder
)

var validAuthRequestRoles = []AuthRequestRole{
	AuthRequestRoleOrchestrator,
	AuthRequestRolePlanner,
	AuthRequestRoleQAProof,
	AuthRequestRoleQAFalsification,
	AuthRequestRoleBuilder,
	AuthRequestRoleResearch,
	AuthRequestRoleCommit,
}

// AuthRequest stores one persisted auth request and its approval outcome.
type AuthRequest struct {
	ID                     string
	ProjectID              string
	BranchID               string
	PhaseIDs               []string
	Path                   string
	ScopeType              ScopeLevel
	ScopeID                string
	PrincipalID            string
	PrincipalType          string
	PrincipalRole          string
	PrincipalName          string
	ClientID               string
	ClientType             string
	ClientName             string
	RequestedSessionTTL    time.Duration
	ApprovedPath           string
	ApprovedSessionTTL     time.Duration
	Reason                 string
	Continuation           map[string]any
	State                  AuthRequestState
	RequestedByActor       string
	RequestedByType        ActorType
	CreatedAt              time.Time
	ExpiresAt              time.Time
	ResolvedByActor        string
	ResolvedByType         ActorType
	ResolvedAt             *time.Time
	ResolutionNote         string
	IssuedSessionID        string
	IssuedSessionSecret    string
	IssuedSessionExpiresAt *time.Time
}

// AuthRequestInput holds write-time values for creating one auth request.
type AuthRequestInput struct {
	ID                  string
	Path                AuthRequestPath
	PrincipalID         string
	PrincipalType       string
	PrincipalRole       string
	PrincipalName       string
	ClientID            string
	ClientType          string
	ClientName          string
	RequestedSessionTTL time.Duration
	Reason              string
	Continuation        map[string]any
	RequestedByActor    string
	RequestedByType     ActorType
	Timeout             time.Duration
}

// AuthRequestListFilter stores deterministic query fields for auth-request listings.
type AuthRequestListFilter struct {
	ProjectID string
	State     AuthRequestState
	Limit     int
}

// ParseAuthRequestPath validates and canonicalizes one auth-request scope path.
func ParseAuthRequestPath(raw string) (AuthRequestPath, error) {
	raw = strings.TrimSpace(raw)
	raw = strings.Trim(raw, "/")
	if raw == "" {
		return AuthRequestPath{}, ErrInvalidAuthRequestPath
	}
	if strings.EqualFold(raw, string(AuthRequestPathKindGlobal)) {
		return AuthRequestPath{Kind: AuthRequestPathKindGlobal}.Normalize()
	}
	if rest, ok := strings.CutPrefix(raw, "projects/"); ok {
		parts := strings.Split(rest, ",")
		return AuthRequestPath{
			Kind:       AuthRequestPathKindProjects,
			ProjectIDs: normalizeAuthRequestIDs(parts),
		}.Normalize()
	}
	parts := strings.Split(raw, "/")
	if len(parts) < 2 || len(parts)%2 != 0 {
		return AuthRequestPath{}, ErrInvalidAuthRequestPath
	}
	if parts[0] != "project" || strings.TrimSpace(parts[1]) == "" {
		return AuthRequestPath{}, ErrInvalidAuthRequestPath
	}
	path := AuthRequestPath{Kind: AuthRequestPathKindProject, ProjectID: strings.TrimSpace(parts[1])}
	seenBranch := false
	for idx := 2; idx < len(parts); idx += 2 {
		segment := strings.TrimSpace(strings.ToLower(parts[idx]))
		value := strings.TrimSpace(parts[idx+1])
		if value == "" {
			return AuthRequestPath{}, ErrInvalidAuthRequestPath
		}
		switch segment {
		case "branch":
			if seenBranch || len(path.PhaseIDs) > 0 {
				return AuthRequestPath{}, ErrInvalidAuthRequestPath
			}
			path.BranchID = value
			seenBranch = true
		case "phase":
			if !seenBranch {
				return AuthRequestPath{}, ErrInvalidAuthRequestPath
			}
			path.PhaseIDs = append(path.PhaseIDs, value)
		default:
			return AuthRequestPath{}, ErrInvalidAuthRequestPath
		}
	}
	return path.Normalize()
}

// NormalizeAuthRequestRole canonicalizes one auth-request role value.
//
// Closed mapping (Drop 4a Wave 3 W3.1):
//
//   - orchestrator → orchestrator
//   - planner → planner
//   - qa-proof → qa-proof
//   - qa-falsification → qa-falsification
//   - builder | subagent | worker → builder (legacy aliases preserved)
//   - research → research
//   - commit → commit
//   - qa (bare) → "" (REJECTED — caller must pick qa-proof or qa-falsification)
//
// Empty-string return for bare "qa" forces IsValidAuthRequestRole to fail,
// surfacing the migration as a hard rejection at NewAuthRequest time.
func NormalizeAuthRequestRole(role AuthRequestRole) AuthRequestRole {
	switch strings.TrimSpace(strings.ToLower(string(role))) {
	case string(AuthRequestRoleOrchestrator):
		return AuthRequestRoleOrchestrator
	case string(AuthRequestRolePlanner):
		return AuthRequestRolePlanner
	case string(AuthRequestRoleQAProof):
		return AuthRequestRoleQAProof
	case string(AuthRequestRoleQAFalsification):
		return AuthRequestRoleQAFalsification
	case string(AuthRequestRoleBuilder), "subagent", "worker":
		return AuthRequestRoleBuilder
	case string(AuthRequestRoleResearch):
		return AuthRequestRoleResearch
	case string(AuthRequestRoleCommit):
		return AuthRequestRoleCommit
	case string(AuthRequestRoleQA):
		// Drop 4a Wave 3 W3.1: bare "qa" is no longer a valid auth-request
		// role. Callers must pick qa-proof or qa-falsification explicitly.
		// Return empty so IsValidAuthRequestRole rejects the value at
		// NewAuthRequest validation time.
		return ""
	default:
		return AuthRequestRole(strings.TrimSpace(strings.ToLower(string(role))))
	}
}

// IsValidAuthRequestRole reports whether one auth-request role is supported.
func IsValidAuthRequestRole(role AuthRequestRole) bool {
	return slices.Contains(validAuthRequestRoles, NormalizeAuthRequestRole(role))
}

// Normalize validates and canonicalizes one auth-request path value.
func (p AuthRequestPath) Normalize() (AuthRequestPath, error) {
	p.Kind = AuthRequestPathKind(strings.TrimSpace(strings.ToLower(string(p.Kind))))
	p.ProjectID = strings.TrimSpace(p.ProjectID)
	p.ProjectIDs = normalizeAuthRequestIDs(p.ProjectIDs)
	p.BranchID = strings.TrimSpace(p.BranchID)
	p.PhaseIDs = normalizeAuthRequestIDs(p.PhaseIDs)
	if p.Kind == "" {
		switch {
		case len(p.ProjectIDs) > 0:
			p.Kind = AuthRequestPathKindProjects
		case p.ProjectID != "":
			p.Kind = AuthRequestPathKindProject
		default:
			p.Kind = AuthRequestPathKindGlobal
		}
	}
	switch p.Kind {
	case AuthRequestPathKindGlobal:
		if p.ProjectID != "" || len(p.ProjectIDs) > 0 || p.BranchID != "" || len(p.PhaseIDs) > 0 {
			return AuthRequestPath{}, ErrInvalidAuthRequestPath
		}
		p.ScopeType = ScopeLevelProject
		p.ScopeID = AuthRequestGlobalProjectID
		return p, nil
	case AuthRequestPathKindProjects:
		if p.ProjectID != "" || p.BranchID != "" || len(p.PhaseIDs) > 0 {
			return AuthRequestPath{}, ErrInvalidAuthRequestPath
		}
		if len(p.ProjectIDs) == 0 {
			return AuthRequestPath{}, ErrInvalidAuthRequestPath
		}
		if len(p.ProjectIDs) == 1 {
			return AuthRequestPath{
				Kind:      AuthRequestPathKindProject,
				ProjectID: p.ProjectIDs[0],
			}.Normalize()
		}
		p.ScopeType = ScopeLevelProject
		p.ScopeID = p.ProjectIDs[0]
		return p, nil
	case AuthRequestPathKindProject:
	default:
		return AuthRequestPath{}, ErrInvalidAuthRequestPath
	}
	if p.ProjectID == "" {
		return AuthRequestPath{}, ErrInvalidAuthRequestPath
	}
	for _, phaseID := range p.PhaseIDs {
		if phaseID == "" {
			return AuthRequestPath{}, ErrInvalidAuthRequestPath
		}
	}
	switch {
	case len(p.PhaseIDs) > 0:
		p.ScopeType = ScopeLevelPhase
		p.ScopeID = p.PhaseIDs[len(p.PhaseIDs)-1]
	case p.BranchID != "":
		p.ScopeType = ScopeLevelBranch
		p.ScopeID = p.BranchID
	default:
		p.ScopeType = ScopeLevelProject
		p.ScopeID = p.ProjectID
	}
	return p, nil
}

// String returns the canonical slash-delimited auth-request path.
func (p AuthRequestPath) String() string {
	p, err := p.Normalize()
	if err != nil {
		return ""
	}
	switch p.Kind {
	case AuthRequestPathKindGlobal:
		return string(AuthRequestPathKindGlobal)
	case AuthRequestPathKindProjects:
		return "projects/" + strings.Join(p.ProjectIDs, ",")
	}
	parts := []string{"project", p.ProjectID}
	if p.BranchID != "" {
		parts = append(parts, "branch", p.BranchID)
	}
	for _, phaseID := range p.PhaseIDs {
		parts = append(parts, "phase", phaseID)
	}
	return strings.Join(parts, "/")
}

// LevelTuple converts one auth-request path into a canonical level tuple.
func (p AuthRequestPath) LevelTuple() (LevelTuple, error) {
	p, err := p.Normalize()
	if err != nil {
		return LevelTuple{}, err
	}
	if p.Kind != AuthRequestPathKindProject {
		return LevelTuple{}, ErrInvalidAuthRequestPath
	}
	switch {
	case len(p.PhaseIDs) > 0:
		return NewLevelTuple(LevelTupleInput{
			ProjectID: p.ProjectID,
			BranchID:  p.BranchID,
			ScopeType: ScopeLevelPhase,
			ScopeID:   p.PhaseIDs[len(p.PhaseIDs)-1],
		})
	case p.BranchID != "":
		return NewLevelTuple(LevelTupleInput{
			ProjectID: p.ProjectID,
			BranchID:  p.BranchID,
			ScopeType: ScopeLevelBranch,
			ScopeID:   p.BranchID,
		})
	default:
		return NewLevelTuple(LevelTupleInput{
			ProjectID: p.ProjectID,
			ScopeType: ScopeLevelProject,
			ScopeID:   p.ProjectID,
		})
	}
}

// PrimaryProjectID returns the primary project identifier used for routing and indexing.
func (p AuthRequestPath) PrimaryProjectID() string {
	p, err := p.Normalize()
	if err != nil {
		return ""
	}
	switch p.Kind {
	case AuthRequestPathKindProject:
		return p.ProjectID
	case AuthRequestPathKindProjects:
		if len(p.ProjectIDs) > 0 {
			return p.ProjectIDs[0]
		}
	case AuthRequestPathKindGlobal:
		return AuthRequestGlobalProjectID
	}
	return ""
}

// MatchesProject reports whether the canonical auth scope applies to one project id.
func (p AuthRequestPath) MatchesProject(projectID string) bool {
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		return false
	}
	p, err := p.Normalize()
	if err != nil {
		return false
	}
	switch p.Kind {
	case AuthRequestPathKindGlobal:
		return true
	case AuthRequestPathKindProjects:
		return slices.Contains(p.ProjectIDs, projectID)
	default:
		return p.ProjectID == projectID
	}
}

// NewAuthRequest validates and constructs one persisted auth request.
func NewAuthRequest(in AuthRequestInput, now time.Time) (AuthRequest, error) {
	in.ID = strings.TrimSpace(in.ID)
	if in.ID == "" {
		return AuthRequest{}, ErrInvalidID
	}
	path, err := in.Path.Normalize()
	if err != nil {
		return AuthRequest{}, err
	}
	if in.RequestedSessionTTL <= 0 || in.Timeout <= 0 {
		return AuthRequest{}, ErrInvalidAuthRequestTTL
	}
	principalID := strings.TrimSpace(in.PrincipalID)
	clientID := strings.TrimSpace(in.ClientID)
	if principalID == "" || clientID == "" {
		return AuthRequest{}, ErrInvalidID
	}
	principalType, err := normalizeAuthRequestPrincipalType(in.PrincipalType)
	if err != nil {
		return AuthRequest{}, err
	}
	principalRole := strings.TrimSpace(string(NormalizeAuthRequestRole(AuthRequestRole(in.PrincipalRole))))
	switch principalType {
	case "agent":
		if principalRole == "" {
			principalRole = string(AuthRequestRoleBuilder)
		}
		if !IsValidAuthRequestRole(AuthRequestRole(principalRole)) {
			return AuthRequest{}, ErrInvalidAuthRequestRole
		}
		// Only orchestrator-role agent requests may carry global or
		// projects/<list>... path shapes. Every non-orchestrator role
		// (builder, planner, qa-proof, qa-falsification, research, commit
		// post-Drop-4a-W3.1) MUST stay rooted under a single
		// project/<id>[/branch/...]/[/phase/...] path. Drop 4a Wave 3 W3.1
		// extended this rule to cover the four new values without changing
		// its shape.
		if path.Kind != AuthRequestPathKindProject && principalRole != string(AuthRequestRoleOrchestrator) {
			return AuthRequest{}, ErrInvalidAuthRequestRole
		}
	case "steward":
		// Drop 3 droplet 3.19: steward principal-type only ever pairs with the
		// orchestrator role. STEWARD itself is a persistent orchestrator that
		// owns post-merge MD collation + worktree cleanup; no other role makes
		// sense for it. Reject every non-orchestrator role with the same
		// sentinel agent role mismatches use.
		if principalRole == "" {
			principalRole = string(AuthRequestRoleOrchestrator)
		}
		if principalRole != string(AuthRequestRoleOrchestrator) {
			return AuthRequest{}, ErrInvalidAuthRequestRole
		}
	default:
		if principalRole != "" {
			return AuthRequest{}, ErrInvalidAuthRequestRole
		}
	}
	scopeType := path.ScopeType
	scopeID := path.ScopeID
	if path.Kind == AuthRequestPathKindGlobal {
		scopeType = ScopeLevelProject
	}
	requestedByActor := strings.TrimSpace(in.RequestedByActor)
	if requestedByActor == "" {
		requestedByActor = "tillsyn-user"
	}
	requestedByType := normalizeActorTypeValue(in.RequestedByType)
	if requestedByType == "" {
		requestedByType = ActorTypeUser
	}
	if !isValidActorType(requestedByType) {
		return AuthRequest{}, ErrInvalidActorType
	}
	ts := now.UTC()
	return AuthRequest{
		ID:                  in.ID,
		ProjectID:           path.PrimaryProjectID(),
		BranchID:            path.BranchID,
		PhaseIDs:            append([]string(nil), path.PhaseIDs...),
		Path:                path.String(),
		ScopeType:           scopeType,
		ScopeID:             scopeID,
		PrincipalID:         principalID,
		PrincipalType:       principalType,
		PrincipalRole:       principalRole,
		PrincipalName:       strings.TrimSpace(in.PrincipalName),
		ClientID:            clientID,
		ClientType:          strings.TrimSpace(in.ClientType),
		ClientName:          strings.TrimSpace(in.ClientName),
		RequestedSessionTTL: in.RequestedSessionTTL,
		Reason:              strings.TrimSpace(in.Reason),
		Continuation:        cloneAuthRequestObjectMap(in.Continuation),
		State:               AuthRequestStatePending,
		RequestedByActor:    requestedByActor,
		RequestedByType:     requestedByType,
		CreatedAt:           ts,
		ExpiresAt:           ts.Add(in.Timeout),
	}, nil
}

// NormalizeAuthRequestState canonicalizes one auth-request state value.
func NormalizeAuthRequestState(state AuthRequestState) AuthRequestState {
	return AuthRequestState(strings.TrimSpace(strings.ToLower(string(state))))
}

// IsValidAuthRequestState reports whether an auth-request state is supported.
func IsValidAuthRequestState(state AuthRequestState) bool {
	return slices.Contains(validAuthRequestStates, NormalizeAuthRequestState(state))
}

// NormalizeAuthRequestListFilter validates and canonicalizes one auth-request listing filter.
func NormalizeAuthRequestListFilter(filter AuthRequestListFilter) (AuthRequestListFilter, error) {
	filter.ProjectID = strings.TrimSpace(filter.ProjectID)
	filter.State = NormalizeAuthRequestState(filter.State)
	if filter.State != "" && !IsValidAuthRequestState(filter.State) {
		return AuthRequestListFilter{}, ErrInvalidAuthRequestState
	}
	if filter.Limit < 0 {
		return AuthRequestListFilter{}, ErrInvalidPosition
	}
	return filter, nil
}

// IsTerminal reports whether an auth request is in a final state.
func (r AuthRequest) IsTerminal() bool {
	switch NormalizeAuthRequestState(r.State) {
	case AuthRequestStateApproved, AuthRequestStateDenied, AuthRequestStateCanceled, AuthRequestStateExpired:
		return true
	default:
		return false
	}
}

// IsExpired reports whether one pending auth request has timed out.
func (r AuthRequest) IsExpired(now time.Time) bool {
	return NormalizeAuthRequestState(r.State) == AuthRequestStatePending && !r.ExpiresAt.IsZero() && !now.UTC().Before(r.ExpiresAt.UTC())
}

// Approve transitions one pending auth request into the approved state.
func (r *AuthRequest) Approve(resolvedBy string, resolvedByType ActorType, note, sessionID, sessionSecret string, sessionExpiresAt time.Time, now time.Time) error {
	if r == nil {
		return ErrInvalidID
	}
	if err := r.ensurePending(now); err != nil {
		return err
	}
	sessionID = strings.TrimSpace(sessionID)
	sessionSecret = strings.TrimSpace(sessionSecret)
	if sessionID == "" || sessionSecret == "" {
		return ErrInvalidID
	}
	resolvedByType = normalizeActorTypeValue(resolvedByType)
	if !isValidActorType(resolvedByType) {
		return ErrInvalidActorType
	}
	ts := now.UTC()
	r.State = AuthRequestStateApproved
	r.ResolvedByActor = strings.TrimSpace(resolvedBy)
	r.ResolvedByType = resolvedByType
	r.ResolvedAt = &ts
	r.ResolutionNote = strings.TrimSpace(note)
	r.IssuedSessionID = sessionID
	r.IssuedSessionSecret = sessionSecret
	exp := sessionExpiresAt.UTC()
	r.IssuedSessionExpiresAt = &exp
	return nil
}

// Deny transitions one pending auth request into the denied state.
func (r *AuthRequest) Deny(resolvedBy string, resolvedByType ActorType, note string, now time.Time) error {
	if r == nil {
		return ErrInvalidID
	}
	if err := r.ensurePending(now); err != nil {
		return err
	}
	resolvedByType = normalizeActorTypeValue(resolvedByType)
	if !isValidActorType(resolvedByType) {
		return ErrInvalidActorType
	}
	ts := now.UTC()
	r.State = AuthRequestStateDenied
	r.ResolvedByActor = strings.TrimSpace(resolvedBy)
	r.ResolvedByType = resolvedByType
	r.ResolvedAt = &ts
	r.ResolutionNote = strings.TrimSpace(note)
	r.IssuedSessionID = ""
	r.IssuedSessionSecret = ""
	r.IssuedSessionExpiresAt = nil
	return nil
}

// Cancel transitions one pending auth request into the canceled state.
func (r *AuthRequest) Cancel(resolvedBy string, resolvedByType ActorType, note string, now time.Time) error {
	if r == nil {
		return ErrInvalidID
	}
	if err := r.ensurePending(now); err != nil {
		return err
	}
	resolvedByType = normalizeActorTypeValue(resolvedByType)
	if !isValidActorType(resolvedByType) {
		return ErrInvalidActorType
	}
	ts := now.UTC()
	r.State = AuthRequestStateCanceled
	r.ResolvedByActor = strings.TrimSpace(resolvedBy)
	r.ResolvedByType = resolvedByType
	r.ResolvedAt = &ts
	r.ResolutionNote = strings.TrimSpace(note)
	r.IssuedSessionID = ""
	r.IssuedSessionSecret = ""
	r.IssuedSessionExpiresAt = nil
	return nil
}

// Expire transitions one pending auth request into the expired state.
func (r *AuthRequest) Expire(now time.Time) error {
	if r == nil {
		return ErrInvalidID
	}
	if NormalizeAuthRequestState(r.State) != AuthRequestStatePending {
		return ErrAuthRequestNotPending
	}
	ts := now.UTC()
	r.State = AuthRequestStateExpired
	r.ResolvedAt = &ts
	r.ResolutionNote = "timed_out"
	r.ResolvedByActor = ""
	r.ResolvedByType = ""
	r.IssuedSessionID = ""
	r.IssuedSessionSecret = ""
	r.IssuedSessionExpiresAt = nil
	return nil
}

// ensurePending verifies that one auth request is still eligible for mutation.
func (r *AuthRequest) ensurePending(now time.Time) error {
	if NormalizeAuthRequestState(r.State) != AuthRequestStatePending {
		return ErrAuthRequestNotPending
	}
	if r.IsExpired(now) {
		return ErrAuthRequestExpired
	}
	return nil
}

// normalizeAuthRequestPrincipalType canonicalizes caller principal types for auth requests.
//
// Accepted values (closed set, post-Drop-3 droplet 3.19): user, agent, service, steward.
// "steward" is a tillsyn-internal axis distinct from autent's closed
// {user, agent, service} principal-type enum. The autentauth adapter
// boundary-maps steward → autentdomain.PrincipalTypeAgent (per Drop 3 L2);
// tillsyn preserves the steward value in its own auth_requests table and on
// AuthenticatedCaller.AuthRequestPrincipalType for the STEWARD owner-state
// gate.
func normalizeAuthRequestPrincipalType(raw string) (string, error) {
	switch strings.TrimSpace(strings.ToLower(raw)) {
	case "", "user":
		return "user", nil
	case "agent":
		return "agent", nil
	case "service", "system":
		return "service", nil
	case "steward":
		return "steward", nil
	default:
		return "", fmt.Errorf("%w: unsupported principal type %q", ErrInvalidActorType, raw)
	}
}

// cloneAuthRequestObjectMap deep-copies one auth-request metadata map.
func cloneAuthRequestObjectMap(in map[string]any) map[string]any {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]any, len(in))
	for key, value := range in {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		out[key] = cloneAuthRequestObjectValue(value)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// cloneAuthRequestObjectValue deep-copies one JSON-compatible auth-request continuation value.
func cloneAuthRequestObjectValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		return cloneAuthRequestObjectMap(typed)
	case []any:
		out := make([]any, 0, len(typed))
		for _, item := range typed {
			out = append(out, cloneAuthRequestObjectValue(item))
		}
		return out
	default:
		return typed
	}
}

// normalizeAuthRequestIDs trims empty identifiers while preserving stable order.
func normalizeAuthRequestIDs(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		out = append(out, value)
	}
	return out
}
