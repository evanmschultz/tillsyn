package common

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/hylla/tillsyn/internal/app"
	"github.com/hylla/tillsyn/internal/domain"
)

// GetBootstrapGuide returns summary-first onboarding guidance for empty-instance,
// pre-approval, and first-use coordination flows.
func (a *AppServiceAdapter) GetBootstrapGuide(_ context.Context) (BootstrapGuide, error) {
	if a == nil || a.service == nil {
		return BootstrapGuide{}, fmt.Errorf("app service adapter is not configured: %w", ErrInvalidCaptureStateRequest)
	}
	return BootstrapGuide{
		Mode:          "bootstrap_required",
		Summary:       "No project context exists yet. If you already have an approved global agent session, create a project; otherwise open an auth request, wait for approval, and claim the continuation with the requester-owned resume_token returned by till.auth_request(operation=create) before continuing. Once work exists, use comments for shared discussion, mentions for routed comment inbox items, and handoffs for explicit next-action coordination that are action-required only for the addressed viewer.",
		WhatTillsynIs: "Tillsyn is a strict task/state planner with level-scoped work (project|branch|phase|task|subtask), guardrailed mutations, shared comment and handoff coordination, routed mention inbox attention, pre-session auth requests, summary-first recovery context, waitable stdio coordination watchers, and SQLite-backed template libraries for generated workflow contracts.",
		Capabilities: []string{
			"Level-scoped capture_state for summary-first recovery",
			"Task graph operations across branch/phase/task/subtask scopes",
			"Markdown comments as the shared discussion thread for human-agent and agent-agent coordination",
			"Role-routed @mentions that materialize viewer-scoped comment inbox rows",
			"Durable handoffs for explicit next-action routing and action-required notifications",
			"Attention/blocker signaling with user-action visibility across comments, handoffs, and other coordination rows",
			"Kind catalog plus template-library-driven generated follow-up work and node-contract snapshots",
			"Pre-session auth requests, approval, and continuation claims",
			"Capability lease issuance and guardrailed non-user mutations",
			"Instruction/bootstrap guidance for README plus optional external agent-policy and skill alignment",
		},
		NextSteps: []string{
			"If this session is already approved for global work, create a project with till.project(operation=create)",
			"If it is not approved yet, create an auth request with till.auth_request(operation=create); if you omit continuation_json it will auto-generate and return a requester-owned resume_token for later claim/cancel",
			"After approval, claim the request with till.auth_request(operation=claim), then create the project with till.project(operation=create)",
			"After the project exists, claim or reuse a project-scoped approved session before guarded in-project mutations such as till.plan_item(operation=create)",
			"Never reuse another actor's session or auth_context_id; each actor should claim or validate its own scoped auth and clean up stale child sessions, leases, and pending requests truthfully after the run",
			"If the project should use workflow contracts, inspect approved template libraries with till.template(operation=list) and bind one with till.project(operation=bind_template) before creating level-scoped work",
			"Use till.comment(operation=create) for shared discussion and status updates inside Tillsyn; role mentions such as @human, @builder, @qa, @orchestrator, and @research route comment inbox rows",
			"Use till.handoff for explicit next-action routing; open handoffs should be interpreted as Action Required only for the addressed viewer and as oversight warnings for everyone else",
			"Prefer till.get_instructions(mode=explain, focus=topic, topic=bootstrap) for the canonical richer bootstrap explanation; till.get_bootstrap_guide remains the lightweight compatibility wrapper on the frozen surface",
			"For active coordination watchers, keep till.attention_item/till.comment/till.handoff operation=list calls open with wait_timeout so they wait for the next change after current baseline state instead of polling",
			"After a client restart on an existing instance, recover with till.capture_state first, then till.attention_item(operation=list), till.handoff(operation=list), and till.comment(operation=list) for the threads you need to resume",
		},
		Recommended: []string{
			"till.get_instructions",
			"till.auth_request",
			"till.project",
			"till.template",
			"till.plan_item",
			"till.comment",
			"till.handoff",
			"till.capture_state",
		},
		RoadmapNotice: "Import/export transport-closure and advanced conflict tooling remain roadmap-only for this wave.",
	}, nil
}

// ListProjects returns project rows from app-level APIs.
func (a *AppServiceAdapter) ListProjects(ctx context.Context, includeArchived bool) ([]domain.Project, error) {
	if a == nil || a.service == nil {
		return nil, fmt.Errorf("app service adapter is not configured: %w", ErrInvalidCaptureStateRequest)
	}
	projects, err := a.service.ListProjects(ctx, includeArchived)
	if err != nil {
		return nil, mapAppError("list projects", err)
	}
	return projects, nil
}

// CreateAuthRequest creates one persisted pre-session auth request through app-level APIs.
func (a *AppServiceAdapter) CreateAuthRequest(ctx context.Context, in CreateAuthRequestRequest) (AuthRequestRecord, error) {
	if a == nil || a.service == nil {
		return AuthRequestRecord{}, fmt.Errorf("app service adapter is not configured: %w", ErrInvalidCaptureStateRequest)
	}
	requestedTTL, err := parseOptionalDurationString(in.RequestedTTL, "requested_ttl")
	if err != nil {
		return AuthRequestRecord{}, err
	}
	timeout, err := parseOptionalDurationString(in.Timeout, "timeout")
	if err != nil {
		return AuthRequestRecord{}, err
	}
	continuation, err := normalizeCreateAuthRequestContinuation(in.ContinuationJSON)
	if err != nil {
		return AuthRequestRecord{}, err
	}
	requestedByActor, requestedByType, requesterClientID, err := a.resolveCreateAuthRequestRequester(ctx, in)
	if err != nil {
		return AuthRequestRecord{}, err
	}
	request, err := a.service.CreateAuthRequest(ctx, app.CreateAuthRequestInput{
		Path:                strings.TrimSpace(in.Path),
		PrincipalID:         strings.TrimSpace(in.PrincipalID),
		PrincipalType:       strings.TrimSpace(in.PrincipalType),
		PrincipalRole:       strings.TrimSpace(in.PrincipalRole),
		PrincipalName:       strings.TrimSpace(in.PrincipalName),
		ClientID:            strings.TrimSpace(in.ClientID),
		ClientType:          strings.TrimSpace(in.ClientType),
		ClientName:          strings.TrimSpace(in.ClientName),
		RequesterClientID:   requesterClientID,
		RequestedSessionTTL: requestedTTL,
		Reason:              strings.TrimSpace(in.Reason),
		Continuation:        continuation,
		RequestedBy:         requestedByActor,
		RequestedType:       requestedByType,
		Timeout:             timeout,
	})
	if err != nil {
		return AuthRequestRecord{}, mapAppError("create auth request", err)
	}
	return mapAuthRequestRecord(request), nil
}

// resolveCreateAuthRequestRequester derives requester ownership for direct and delegated auth-request creation.
func (a *AppServiceAdapter) resolveCreateAuthRequestRequester(ctx context.Context, in CreateAuthRequestRequest) (string, domain.ActorType, string, error) {
	actingSessionID := strings.TrimSpace(in.ActingSessionID)
	actingSessionSecret := strings.TrimSpace(in.ActingSessionSecret)
	if actingSessionID == "" && actingSessionSecret == "" {
		return firstNonEmptyRequestedBy(in.RequestedByActor, in.PrincipalID), requestedActorType(in.RequestedByType, in.PrincipalType), requesterClientID(in), nil
	}
	if actingSessionID == "" || actingSessionSecret == "" {
		return "", "", "", fmt.Errorf("acting_session_id and acting_session_secret are both required for delegated auth request creation: %w", ErrInvalidCaptureStateRequest)
	}
	actingSession, actingPath, err := a.authorizeAuthSessionGovernance(ctx, actingSessionID, actingSessionSecret)
	if err != nil {
		return "", "", "", err
	}
	requestedPath, err := domain.ParseAuthRequestPath(strings.TrimSpace(in.Path))
	if err != nil {
		return "", "", "", err
	}
	if !authRequestPathWithin(actingPath, requestedPath) {
		return "", "", "", ErrAuthorizationDenied
	}
	if err := validateDelegatedAuthRequestRequester(in, actingSession); err != nil {
		return "", "", "", err
	}
	if delegatedAuthRequestIdentityDiffers(in, actingSession) && !authSessionRoleMayGovernOthers(actingSession) {
		return "", "", "", ErrAuthorizationDenied
	}
	return strings.TrimSpace(actingSession.PrincipalID), domain.ActorType(strings.TrimSpace(actingSession.PrincipalType)), strings.TrimSpace(actingSession.ClientID), nil
}

// ListAuthRequests returns auth-request inventory rows from app-level APIs.
func (a *AppServiceAdapter) ListAuthRequests(ctx context.Context, in ListAuthRequestsRequest) ([]AuthRequestRecord, error) {
	if a == nil || a.service == nil {
		return nil, fmt.Errorf("app service adapter is not configured: %w", ErrInvalidCaptureStateRequest)
	}
	requests, err := a.service.ListAuthRequests(ctx, domain.AuthRequestListFilter{
		ProjectID: strings.TrimSpace(in.ProjectID),
		State:     domain.AuthRequestState(strings.TrimSpace(in.State)),
		Limit:     in.Limit,
	})
	if err != nil {
		return nil, mapAppError("list auth requests", err)
	}
	out := make([]AuthRequestRecord, 0, len(requests))
	for _, request := range requests {
		out = append(out, mapAuthRequestRecord(request))
	}
	return out, nil
}

// GetAuthRequest returns one auth request by id through app-level APIs.
func (a *AppServiceAdapter) GetAuthRequest(ctx context.Context, requestID string) (AuthRequestRecord, error) {
	if a == nil || a.service == nil {
		return AuthRequestRecord{}, fmt.Errorf("app service adapter is not configured: %w", ErrInvalidCaptureStateRequest)
	}
	requestID = strings.TrimSpace(requestID)
	if requestID == "" {
		return AuthRequestRecord{}, fmt.Errorf("request_id is required: %w", ErrInvalidCaptureStateRequest)
	}
	request, err := a.service.GetAuthRequest(ctx, requestID)
	if err != nil {
		return AuthRequestRecord{}, mapAppError("get auth request", err)
	}
	return mapAuthRequestRecord(request), nil
}

// ClaimAuthRequest returns one requester-visible auth request state and approved session secret through continuation proof.
func (a *AppServiceAdapter) ClaimAuthRequest(ctx context.Context, in ClaimAuthRequestRequest) (AuthRequestClaimResult, error) {
	if a == nil || a.service == nil {
		return AuthRequestClaimResult{}, fmt.Errorf("app service adapter is not configured: %w", ErrInvalidCaptureStateRequest)
	}
	requestID := strings.TrimSpace(in.RequestID)
	if requestID == "" {
		return AuthRequestClaimResult{}, fmt.Errorf("request_id is required: %w", ErrInvalidCaptureStateRequest)
	}
	waitTimeout, err := parseOptionalDurationString(in.WaitTimeout, "wait_timeout")
	if err != nil {
		return AuthRequestClaimResult{}, err
	}
	result, err := a.service.ClaimAuthRequest(ctx, app.ClaimAuthRequestInput{
		RequestID:   requestID,
		ResumeToken: strings.TrimSpace(in.ResumeToken),
		PrincipalID: strings.TrimSpace(in.PrincipalID),
		ClientID:    strings.TrimSpace(in.ClientID),
		WaitTimeout: waitTimeout,
	})
	if err != nil {
		return AuthRequestClaimResult{}, mapAppError("claim auth request", err)
	}
	return AuthRequestClaimResult{
		Request:       mapAuthRequestRecord(result.Request),
		SessionSecret: result.SessionSecret,
		Waiting:       result.Waiting,
	}, nil
}

// CancelAuthRequest cancels one pending auth request through app-level APIs.
func (a *AppServiceAdapter) CancelAuthRequest(ctx context.Context, in CancelAuthRequestRequest) (AuthRequestRecord, error) {
	if a == nil || a.service == nil {
		return AuthRequestRecord{}, fmt.Errorf("app service adapter is not configured: %w", ErrInvalidCaptureStateRequest)
	}
	requestID := strings.TrimSpace(in.RequestID)
	if requestID == "" {
		return AuthRequestRecord{}, fmt.Errorf("request_id is required: %w", ErrInvalidCaptureStateRequest)
	}
	request, err := a.GetAuthRequest(ctx, requestID)
	if err != nil {
		return AuthRequestRecord{}, mapAppError("cancel auth request", err)
	}
	if err := authRequestCancelOwnershipMatches(request, in.PrincipalID, in.ClientID, in.ResumeToken); err != nil {
		return AuthRequestRecord{}, err
	}
	resolvedBy, resolvedType, err := resolvedAuthRequestActor(
		strings.TrimSpace(in.PrincipalID),
		string(request.RequestedByType),
	)
	if err != nil {
		return AuthRequestRecord{}, err
	}
	canceled, err := a.service.CancelAuthRequest(ctx, app.CancelAuthRequestInput{
		RequestID:      requestID,
		ResolvedBy:     resolvedBy,
		ResolvedType:   resolvedType,
		ResolutionNote: strings.TrimSpace(in.ResolutionNote),
	})
	if err != nil {
		return AuthRequestRecord{}, mapAppError("cancel auth request", err)
	}
	return mapAuthRequestRecord(canceled), nil
}

// ListAuthSessions returns caller-safe auth-session inventory through app-level APIs.
func (a *AppServiceAdapter) ListAuthSessions(ctx context.Context, in ListAuthSessionsRequest) ([]AuthSessionRecord, error) {
	if a == nil || a.service == nil {
		return nil, fmt.Errorf("app service adapter is not configured: %w", ErrInvalidCaptureStateRequest)
	}
	actingSession, actingPath, err := a.authorizeAuthSessionGovernance(ctx, in.ActingSessionID, in.ActingSessionSecret)
	if err != nil {
		return nil, err
	}
	projectID := strings.TrimSpace(in.ProjectID)
	if projectID != "" && !actingPath.MatchesProject(projectID) {
		return nil, ErrAuthorizationDenied
	}
	sessionID := strings.TrimSpace(in.SessionID)
	if !authSessionRoleMayGovernOthers(actingSession) {
		if sessionID == "" || sessionID != strings.TrimSpace(actingSession.SessionID) {
			return nil, ErrAuthorizationDenied
		}
	}
	if projectID == "" && actingPath.Kind == domain.AuthRequestPathKindProject {
		projectID = actingPath.ProjectID
	}
	sessions, err := a.service.ListAuthSessions(ctx, app.AuthSessionFilter{
		SessionID:   sessionID,
		ProjectID:   projectID,
		PrincipalID: strings.TrimSpace(in.PrincipalID),
		ClientID:    strings.TrimSpace(in.ClientID),
		State:       strings.TrimSpace(in.State),
		Limit:       0,
	})
	if err != nil {
		return nil, mapAppError("list auth sessions", err)
	}
	out := make([]AuthSessionRecord, 0, len(sessions))
	for _, session := range sessions {
		if !authSessionWithinApprovedPath(session, actingPath) {
			continue
		}
		out = append(out, mapAuthSessionRecord(session, time.Now().UTC()))
		if in.Limit > 0 && len(out) >= in.Limit {
			break
		}
	}
	return out, nil
}

// ValidateAuthSession validates one session/secret pair through app-level APIs.
func (a *AppServiceAdapter) ValidateAuthSession(ctx context.Context, in ValidateAuthSessionRequest) (AuthSessionRecord, error) {
	if a == nil || a.service == nil {
		return AuthSessionRecord{}, fmt.Errorf("app service adapter is not configured: %w", ErrInvalidCaptureStateRequest)
	}
	sessionID := strings.TrimSpace(in.SessionID)
	if sessionID == "" {
		return AuthSessionRecord{}, fmt.Errorf("session_id is required: %w", ErrInvalidCaptureStateRequest)
	}
	sessionSecret := strings.TrimSpace(in.SessionSecret)
	if sessionSecret == "" {
		return AuthSessionRecord{}, fmt.Errorf("session_secret is required: %w", ErrInvalidCaptureStateRequest)
	}
	validated, err := a.service.ValidateAuthSession(ctx, sessionID, sessionSecret)
	if err != nil {
		return AuthSessionRecord{}, mapAppError("validate auth session", err)
	}
	return mapAuthSessionRecord(validated.Session, time.Now().UTC()), nil
}

// CheckAuthSessionGovernance returns one non-destructive auth-session governance decision.
func (a *AppServiceAdapter) CheckAuthSessionGovernance(ctx context.Context, in CheckAuthSessionGovernanceRequest) (AuthSessionGovernanceCheckResult, error) {
	if a == nil || a.service == nil {
		return AuthSessionGovernanceCheckResult{}, fmt.Errorf("app service adapter is not configured: %w", ErrInvalidCaptureStateRequest)
	}
	targetSession, err := a.lookupAuthSessionForGovernance(ctx, in.SessionID)
	if err != nil {
		return AuthSessionGovernanceCheckResult{}, err
	}
	decision, err := a.resolveAuthSessionGovernance(ctx, in.ActingSessionID, in.ActingSessionSecret, targetSession)
	if err != nil {
		return AuthSessionGovernanceCheckResult{}, err
	}
	return decision, nil
}

// RevokeAuthSession revokes one auth session through app-level APIs.
func (a *AppServiceAdapter) RevokeAuthSession(ctx context.Context, in RevokeAuthSessionRequest) (AuthSessionRecord, error) {
	if a == nil || a.service == nil {
		return AuthSessionRecord{}, fmt.Errorf("app service adapter is not configured: %w", ErrInvalidCaptureStateRequest)
	}
	targetSession, err := a.lookupAuthSessionForGovernance(ctx, in.SessionID)
	if err != nil {
		return AuthSessionRecord{}, err
	}
	decision, err := a.resolveAuthSessionGovernance(ctx, in.ActingSessionID, in.ActingSessionSecret, targetSession)
	if err != nil {
		return AuthSessionRecord{}, err
	}
	if !decision.Authorized {
		return AuthSessionRecord{}, ErrAuthorizationDenied
	}
	revoked, err := a.service.RevokeAuthSession(ctx, strings.TrimSpace(in.SessionID), strings.TrimSpace(in.Reason))
	if err != nil {
		return AuthSessionRecord{}, mapAppError("revoke auth session", err)
	}
	return mapAuthSessionRecord(revoked, time.Now().UTC()), nil
}

// authorizeAuthSessionGovernance validates one acting auth session and returns its effective approved path.
func (a *AppServiceAdapter) authorizeAuthSessionGovernance(ctx context.Context, actingSessionID, actingSessionSecret string) (app.AuthSession, domain.AuthRequestPath, error) {
	if a == nil || a.service == nil {
		return app.AuthSession{}, domain.AuthRequestPath{}, fmt.Errorf("auth session governance is not configured: %w", ErrInvalidCaptureStateRequest)
	}
	if a.auth == nil {
		return app.AuthSession{}, domain.AuthRequestPath{}, fmt.Errorf("auth session governance is not configured: %w", ErrInvalidCaptureStateRequest)
	}
	actingSessionID = strings.TrimSpace(actingSessionID)
	actingSessionSecret = strings.TrimSpace(actingSessionSecret)
	if actingSessionID == "" || actingSessionSecret == "" {
		return app.AuthSession{}, domain.AuthRequestPath{}, fmt.Errorf("acting_session_id and acting_session_secret are required: %w", ErrInvalidCaptureStateRequest)
	}
	validated, err := a.service.ValidateAuthSession(ctx, actingSessionID, actingSessionSecret)
	if err != nil {
		return app.AuthSession{}, domain.AuthRequestPath{}, mapAppError("authorize auth session governance", err)
	}
	actingPath, err := authSessionApprovedPath(validated.Session)
	if err != nil {
		return app.AuthSession{}, domain.AuthRequestPath{}, err
	}
	return validated.Session, actingPath, nil
}

// lookupAuthSessionForGovernance resolves one governed auth session row for check/revoke flows.
func (a *AppServiceAdapter) lookupAuthSessionForGovernance(ctx context.Context, sessionID string) (app.AuthSession, error) {
	if a == nil || a.service == nil {
		return app.AuthSession{}, fmt.Errorf("auth session governance is not configured: %w", ErrInvalidCaptureStateRequest)
	}
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return app.AuthSession{}, fmt.Errorf("session_id is required: %w", ErrInvalidCaptureStateRequest)
	}
	targetSessions, err := a.service.ListAuthSessions(ctx, app.AuthSessionFilter{SessionID: sessionID, Limit: 1})
	if err != nil {
		return app.AuthSession{}, mapAppError("lookup auth session", err)
	}
	if len(targetSessions) == 0 {
		return app.AuthSession{}, fmt.Errorf("lookup auth session: %w", ErrNotFound)
	}
	return targetSessions[0], nil
}

// resolveAuthSessionGovernance evaluates whether one acting session may govern one target session.
func (a *AppServiceAdapter) resolveAuthSessionGovernance(ctx context.Context, actingSessionID, actingSessionSecret string, targetSession app.AuthSession) (AuthSessionGovernanceCheckResult, error) {
	actingSession, actingPath, err := a.authorizeAuthSessionGovernance(ctx, actingSessionID, actingSessionSecret)
	if err != nil {
		return AuthSessionGovernanceCheckResult{}, err
	}
	decision := AuthSessionGovernanceCheckResult{
		Authorized:          false,
		ActingSessionID:     strings.TrimSpace(actingSession.SessionID),
		ActingPrincipalID:   strings.TrimSpace(actingSession.PrincipalID),
		ActingPrincipalRole: strings.TrimSpace(actingSession.PrincipalRole),
		ActingApprovedPath:  strings.TrimSpace(actingSession.ApprovedPath),
		TargetSession:       mapAuthSessionRecord(targetSession, time.Now().UTC()),
	}
	if strings.TrimSpace(actingSession.SessionID) == strings.TrimSpace(targetSession.SessionID) {
		decision.Authorized = true
		decision.DecisionReason = "self"
		return decision, nil
	}
	if !authSessionRoleMayGovernOthers(actingSession) {
		decision.DecisionReason = "role_denied"
		return decision, nil
	}
	if !authSessionWithinApprovedPath(targetSession, actingPath) {
		decision.DecisionReason = "out_of_scope"
		return decision, nil
	}
	decision.Authorized = true
	decision.DecisionReason = "within_scope"
	return decision, nil
}

// authSessionRoleMayGovernOthers reports whether one acting session may govern other sessions in-scope.
func authSessionRoleMayGovernOthers(session app.AuthSession) bool {
	return strings.TrimSpace(session.PrincipalRole) == string(domain.AuthRequestRoleOrchestrator)
}

// delegatedAuthRequestIdentityDiffers reports whether one delegated auth request targets a different principal/client pair.
func delegatedAuthRequestIdentityDiffers(in CreateAuthRequestRequest, actingSession app.AuthSession) bool {
	return strings.TrimSpace(in.PrincipalID) != strings.TrimSpace(actingSession.PrincipalID) || strings.TrimSpace(in.ClientID) != strings.TrimSpace(actingSession.ClientID)
}

// validateDelegatedAuthRequestRequester rejects caller-supplied requester metadata that disagrees with the acting session.
func validateDelegatedAuthRequestRequester(in CreateAuthRequestRequest, actingSession app.AuthSession) error {
	if requestedByActor := strings.TrimSpace(in.RequestedByActor); requestedByActor != "" && requestedByActor != strings.TrimSpace(actingSession.PrincipalID) {
		return fmt.Errorf("requested_by_actor must match acting session principal for delegated auth request creation: %w", ErrInvalidCaptureStateRequest)
	}
	if requestedByType := strings.TrimSpace(in.RequestedByType); requestedByType != "" && requestedByType != strings.TrimSpace(actingSession.PrincipalType) {
		return fmt.Errorf("requested_by_type must match acting session principal type for delegated auth request creation: %w", ErrInvalidCaptureStateRequest)
	}
	if requesterClientID := strings.TrimSpace(in.RequesterClientID); requesterClientID != "" && requesterClientID != strings.TrimSpace(actingSession.ClientID) {
		return fmt.Errorf("requester_client_id must match acting session client_id for delegated auth request creation: %w", ErrInvalidCaptureStateRequest)
	}
	return nil
}

// approvedPathFromSessionMetadata parses one approved path from session metadata.
func approvedPathFromSessionMetadata(sessionMetadata map[string]string) (domain.AuthRequestPath, error) {
	if sessionMetadata == nil {
		return domain.AuthRequestPath{}, fmt.Errorf("auth session governance is not configured: %w", ErrInvalidCaptureStateRequest)
	}
	approvedPath := strings.TrimSpace(sessionMetadata["approved_path"])
	if approvedPath == "" {
		return domain.AuthRequestPath{}, ErrAuthorizationDenied
	}
	path, err := domain.ParseAuthRequestPath(approvedPath)
	if err != nil {
		return domain.AuthRequestPath{}, fmt.Errorf("approved_path %q is invalid: %w", approvedPath, ErrAuthorizationDenied)
	}
	return path, nil
}

// authSessionWithinApprovedPath reports whether one governed session row stays within the acting approved path.
func authSessionWithinApprovedPath(session app.AuthSession, actingPath domain.AuthRequestPath) bool {
	targetPath, err := authSessionApprovedPath(session)
	if err != nil {
		return false
	}
	return authRequestPathWithin(actingPath, targetPath)
}

// authSessionApprovedPath resolves one auth-session row back to its effective approved path.
func authSessionApprovedPath(session app.AuthSession) (domain.AuthRequestPath, error) {
	if approvedPath := strings.TrimSpace(session.ApprovedPath); approvedPath != "" {
		return domain.ParseAuthRequestPath(approvedPath)
	}
	if projectID := strings.TrimSpace(session.ProjectID); projectID != "" {
		return domain.AuthRequestPath{ProjectID: projectID}.Normalize()
	}
	return domain.AuthRequestPath{}, ErrAuthorizationDenied
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

// CreateProject creates one project with optional kind and metadata.
func (a *AppServiceAdapter) CreateProject(ctx context.Context, in CreateProjectRequest) (domain.Project, error) {
	if a == nil || a.service == nil {
		return domain.Project{}, fmt.Errorf("app service adapter is not configured: %w", ErrInvalidCaptureStateRequest)
	}
	ctx, actorType, err := withMutationGuardContextAllowUnguardedAgent(ctx, in.Actor, true)
	if err != nil {
		return domain.Project{}, err
	}
	actorID, actorName := deriveMutationActorIdentity(in.Actor)
	project, err := a.service.CreateProjectWithMetadata(ctx, app.CreateProjectInput{
		Name:              strings.TrimSpace(in.Name),
		Description:       strings.TrimSpace(in.Description),
		Kind:              domain.KindID(strings.TrimSpace(in.Kind)),
		TemplateLibraryID: strings.TrimSpace(in.TemplateLibraryID),
		Metadata:          in.Metadata,
		UpdatedBy:         actorID,
		UpdatedByName:     actorName,
		UpdatedType:       actorType,
	})
	if err != nil {
		return domain.Project{}, mapAppError("create project", err)
	}
	return project, nil
}

// UpdateProject updates one project.
func (a *AppServiceAdapter) UpdateProject(ctx context.Context, in UpdateProjectRequest) (domain.Project, error) {
	if a == nil || a.service == nil {
		return domain.Project{}, fmt.Errorf("app service adapter is not configured: %w", ErrInvalidCaptureStateRequest)
	}
	ctx, actorType, err := withMutationGuardContext(ctx, in.Actor)
	if err != nil {
		return domain.Project{}, err
	}
	actorID, actorName := deriveMutationActorIdentity(in.Actor)
	project, err := a.service.UpdateProject(ctx, app.UpdateProjectInput{
		ProjectID:     strings.TrimSpace(in.ProjectID),
		Name:          strings.TrimSpace(in.Name),
		Description:   strings.TrimSpace(in.Description),
		Kind:          domain.KindID(strings.TrimSpace(in.Kind)),
		Metadata:      in.Metadata,
		UpdatedBy:     actorID,
		UpdatedByName: actorName,
		UpdatedType:   actorType,
	})
	if err != nil {
		return domain.Project{}, mapAppError("update project", err)
	}
	return project, nil
}

// ListTasks returns tasks for one project with deterministic ordering from app-level APIs.
func (a *AppServiceAdapter) ListTasks(ctx context.Context, projectID string, includeArchived bool) ([]domain.Task, error) {
	if a == nil || a.service == nil {
		return nil, fmt.Errorf("app service adapter is not configured: %w", ErrInvalidCaptureStateRequest)
	}
	tasks, err := a.service.ListTasks(ctx, strings.TrimSpace(projectID), includeArchived)
	if err != nil {
		return nil, mapAppError("list tasks", err)
	}
	return tasks, nil
}

// GetTask returns one task/work-item row by id.
func (a *AppServiceAdapter) GetTask(ctx context.Context, taskID string) (domain.Task, error) {
	if a == nil || a.service == nil {
		return domain.Task{}, fmt.Errorf("app service adapter is not configured: %w", ErrInvalidCaptureStateRequest)
	}
	task, err := a.service.GetTask(ctx, strings.TrimSpace(taskID))
	if err != nil {
		return domain.Task{}, mapAppError("get task", err)
	}
	return task, nil
}

// CreateTask creates one level-scoped task/work item.
func (a *AppServiceAdapter) CreateTask(ctx context.Context, in CreateTaskRequest) (domain.Task, error) {
	if a == nil || a.service == nil {
		return domain.Task{}, fmt.Errorf("app service adapter is not configured: %w", ErrInvalidCaptureStateRequest)
	}
	dueAt, err := parseOptionalRFC3339(in.DueAt)
	if err != nil {
		return domain.Task{}, err
	}
	ctx, actorType, err := withMutationGuardContext(ctx, in.Actor)
	if err != nil {
		return domain.Task{}, err
	}
	actorID, actorName := deriveMutationActorIdentity(in.Actor)
	task, err := a.service.CreateTask(ctx, app.CreateTaskInput{
		ProjectID:      strings.TrimSpace(in.ProjectID),
		ParentID:       strings.TrimSpace(in.ParentID),
		Kind:           domain.WorkKind(strings.TrimSpace(in.Kind)),
		Scope:          domain.KindAppliesTo(strings.TrimSpace(in.Scope)),
		ColumnID:       strings.TrimSpace(in.ColumnID),
		Title:          strings.TrimSpace(in.Title),
		Description:    strings.TrimSpace(in.Description),
		Priority:       domain.Priority(strings.TrimSpace(strings.ToLower(in.Priority))),
		DueAt:          dueAt,
		Labels:         append([]string(nil), in.Labels...),
		Metadata:       in.Metadata,
		CreatedByActor: actorID,
		CreatedByName:  actorName,
		UpdatedByActor: actorID,
		UpdatedByName:  actorName,
		UpdatedByType:  actorType,
	})
	if err != nil {
		return domain.Task{}, mapAppError("create task", err)
	}
	return task, nil
}

// UpdateTask updates one task/work-item row.
func (a *AppServiceAdapter) UpdateTask(ctx context.Context, in UpdateTaskRequest) (domain.Task, error) {
	if a == nil || a.service == nil {
		return domain.Task{}, fmt.Errorf("app service adapter is not configured: %w", ErrInvalidCaptureStateRequest)
	}
	dueAt, err := parseOptionalRFC3339(in.DueAt)
	if err != nil {
		return domain.Task{}, err
	}
	ctx, actorType, err := withMutationGuardContext(ctx, in.Actor)
	if err != nil {
		return domain.Task{}, err
	}
	actorID, actorName := deriveMutationActorIdentity(in.Actor)
	task, err := a.service.UpdateTask(ctx, app.UpdateTaskInput{
		TaskID:        strings.TrimSpace(in.TaskID),
		Title:         strings.TrimSpace(in.Title),
		Description:   strings.TrimSpace(in.Description),
		Priority:      domain.Priority(strings.TrimSpace(strings.ToLower(in.Priority))),
		DueAt:         dueAt,
		Labels:        append([]string(nil), in.Labels...),
		Metadata:      in.Metadata,
		UpdatedBy:     actorID,
		UpdatedByName: actorName,
		UpdatedType:   actorType,
	})
	if err != nil {
		return domain.Task{}, mapAppError("update task", err)
	}
	return task, nil
}

// MoveTask moves one task to a target column/position.
func (a *AppServiceAdapter) MoveTask(ctx context.Context, in MoveTaskRequest) (domain.Task, error) {
	if a == nil || a.service == nil {
		return domain.Task{}, fmt.Errorf("app service adapter is not configured: %w", ErrInvalidCaptureStateRequest)
	}
	ctx, _, err := withMutationGuardContext(ctx, in.Actor)
	if err != nil {
		return domain.Task{}, err
	}
	task, err := a.service.MoveTask(ctx, strings.TrimSpace(in.TaskID), strings.TrimSpace(in.ToColumnID), in.Position)
	if err != nil {
		return domain.Task{}, mapAppError("move task", err)
	}
	return task, nil
}

// MoveTaskState moves one task/work-item to the column that represents the requested lifecycle state.
func (a *AppServiceAdapter) MoveTaskState(ctx context.Context, in MoveTaskStateRequest) (domain.Task, error) {
	if a == nil || a.service == nil {
		return domain.Task{}, fmt.Errorf("app service adapter is not configured: %w", ErrInvalidCaptureStateRequest)
	}
	ctx, _, err := withMutationGuardContext(ctx, in.Actor)
	if err != nil {
		return domain.Task{}, err
	}
	taskID := strings.TrimSpace(in.TaskID)
	if taskID == "" {
		return domain.Task{}, fmt.Errorf("task_id is required: %w", ErrInvalidCaptureStateRequest)
	}
	state, err := normalizeTaskStateInput(in.State)
	if err != nil {
		return domain.Task{}, err
	}
	task, err := a.service.GetTask(ctx, taskID)
	if err != nil {
		return domain.Task{}, mapAppError("move task state", err)
	}
	targetColumnID, err := a.resolveTaskColumnIDForState(ctx, task.ProjectID, state)
	if err != nil {
		return domain.Task{}, err
	}
	if strings.TrimSpace(task.ColumnID) == targetColumnID && task.LifecycleState == state {
		return task, nil
	}
	moved, err := a.service.MoveTask(ctx, task.ID, targetColumnID, task.Position)
	if err != nil {
		return domain.Task{}, mapAppError("move task state", err)
	}
	return moved, nil
}

// DeleteTask applies archive/hard delete behavior for one task.
func (a *AppServiceAdapter) DeleteTask(ctx context.Context, in DeleteTaskRequest) error {
	if a == nil || a.service == nil {
		return fmt.Errorf("app service adapter is not configured: %w", ErrInvalidCaptureStateRequest)
	}
	ctx, _, err := withMutationGuardContext(ctx, in.Actor)
	if err != nil {
		return err
	}
	if err := a.service.DeleteTask(ctx, strings.TrimSpace(in.TaskID), app.DeleteMode(strings.TrimSpace(in.Mode))); err != nil {
		return mapAppError("delete task", err)
	}
	return nil
}

// RestoreTask restores one archived task.
func (a *AppServiceAdapter) RestoreTask(ctx context.Context, in RestoreTaskRequest) (domain.Task, error) {
	if a == nil || a.service == nil {
		return domain.Task{}, fmt.Errorf("app service adapter is not configured: %w", ErrInvalidCaptureStateRequest)
	}
	ctx, _, err := withMutationGuardContext(ctx, in.Actor)
	if err != nil {
		return domain.Task{}, err
	}
	task, err := a.service.RestoreTask(ctx, strings.TrimSpace(in.TaskID))
	if err != nil {
		return domain.Task{}, mapAppError("restore task", err)
	}
	return task, nil
}

// ReparentTask changes the parent relationship for one task.
func (a *AppServiceAdapter) ReparentTask(ctx context.Context, in ReparentTaskRequest) (domain.Task, error) {
	if a == nil || a.service == nil {
		return domain.Task{}, fmt.Errorf("app service adapter is not configured: %w", ErrInvalidCaptureStateRequest)
	}
	ctx, _, err := withMutationGuardContext(ctx, in.Actor)
	if err != nil {
		return domain.Task{}, err
	}
	task, err := a.service.ReparentTask(ctx, strings.TrimSpace(in.TaskID), strings.TrimSpace(in.ParentID))
	if err != nil {
		return domain.Task{}, mapAppError("reparent task", err)
	}
	return task, nil
}

// ListChildTasks lists children for one parent task.
func (a *AppServiceAdapter) ListChildTasks(ctx context.Context, projectID, parentID string, includeArchived bool) ([]domain.Task, error) {
	if a == nil || a.service == nil {
		return nil, fmt.Errorf("app service adapter is not configured: %w", ErrInvalidCaptureStateRequest)
	}
	tasks, err := a.service.ListChildTasks(ctx, strings.TrimSpace(projectID), strings.TrimSpace(parentID), includeArchived)
	if err != nil {
		return nil, mapAppError("list child tasks", err)
	}
	return tasks, nil
}

func (a *AppServiceAdapter) resolveTaskColumnIDForState(ctx context.Context, projectID string, state domain.LifecycleState) (string, error) {
	columns, err := a.service.ListColumns(ctx, strings.TrimSpace(projectID), true)
	if err != nil {
		return "", mapAppError("resolve task state column", err)
	}
	for _, column := range columns {
		if taskLifecycleStateForColumnName(column.Name) == state {
			return strings.TrimSpace(column.ID), nil
		}
	}
	return "", fmt.Errorf("state %q has no mapped column in project %q: %w", state, strings.TrimSpace(projectID), ErrInvalidCaptureStateRequest)
}

func normalizeTaskStateInput(raw string) (domain.LifecycleState, error) {
	switch taskLifecycleStateForColumnName(raw) {
	case domain.StateTodo, domain.StateProgress, domain.StateDone:
		return taskLifecycleStateForColumnName(raw), nil
	case domain.StateArchived:
		return "", fmt.Errorf("state %q is unsupported for move_state; use delete/restore for archive flows: %w", strings.TrimSpace(raw), ErrInvalidCaptureStateRequest)
	default:
		return "", fmt.Errorf("state %q is unsupported: %w", strings.TrimSpace(raw), ErrInvalidCaptureStateRequest)
	}
}

func taskLifecycleStateForColumnName(name string) domain.LifecycleState {
	switch normalizeStateLikeID(name) {
	case "todo":
		return domain.StateTodo
	case "progress":
		return domain.StateProgress
	case "done":
		return domain.StateDone
	case "archived":
		return domain.StateArchived
	default:
		return ""
	}
}

func normalizeStateLikeID(name string) string {
	name = strings.TrimSpace(strings.ToLower(name))
	if name == "" {
		return ""
	}
	var b strings.Builder
	lastDash := false
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
			lastDash = false
		case r >= '0' && r <= '9':
			b.WriteRune(r)
			lastDash = false
		default:
			if !lastDash {
				b.WriteByte('-')
				lastDash = true
			}
		}
	}
	normalized := strings.Trim(b.String(), "-")
	switch normalized {
	case "to-do", "todo":
		return "todo"
	case "in-progress", "progress", "doing":
		return "progress"
	case "done", "complete", "completed":
		return "done"
	case "archived", "archive":
		return "archived"
	default:
		return normalized
	}
}

// SearchTasks runs a scoped or cross-project search query.
func (a *AppServiceAdapter) SearchTasks(ctx context.Context, in SearchTasksRequest) (SearchTasksResult, error) {
	if a == nil || a.service == nil {
		return SearchTasksResult{}, fmt.Errorf("app service adapter is not configured: %w", ErrInvalidCaptureStateRequest)
	}
	result, err := a.service.SearchTasks(ctx, app.SearchTasksFilter{
		ProjectID:       strings.TrimSpace(in.ProjectID),
		Query:           strings.TrimSpace(in.Query),
		CrossProject:    in.CrossProject,
		IncludeArchived: in.IncludeArchived,
		States:          append([]string(nil), in.States...),
		Levels:          append([]string(nil), in.Levels...),
		Kinds:           append([]string(nil), in.Kinds...),
		LabelsAny:       append([]string(nil), in.LabelsAny...),
		LabelsAll:       append([]string(nil), in.LabelsAll...),
		Mode:            app.SearchMode(strings.TrimSpace(in.Mode)),
		Sort:            app.SearchSort(strings.TrimSpace(in.Sort)),
		Limit:           in.Limit,
		Offset:          in.Offset,
	})
	if err != nil {
		return SearchTasksResult{}, mapAppError("search task matches", err)
	}
	out := make([]SearchTaskMatch, 0, len(result.Matches))
	for _, match := range result.Matches {
		out = append(out, SearchTaskMatch{
			Project:                   match.Project,
			Task:                      match.Task,
			StateID:                   match.StateID,
			EmbeddingSubjectType:      string(match.EmbeddingSubjectType),
			EmbeddingSubjectID:        match.EmbeddingSubjectID,
			EmbeddingStatus:           string(match.EmbeddingStatus),
			EmbeddingUpdatedAt:        match.EmbeddingUpdatedAt,
			EmbeddingStaleReason:      match.EmbeddingStaleReason,
			EmbeddingLastErrorSummary: match.EmbeddingLastErrorSummary,
			SemanticScore:             match.SemanticScore,
			UsedSemantic:              match.UsedSemantic,
		})
	}
	return SearchTasksResult{
		Matches:                out,
		RequestedMode:          string(result.RequestedMode),
		EffectiveMode:          string(result.EffectiveMode),
		FallbackReason:         result.FallbackReason,
		SemanticAvailable:      result.SemanticAvailable,
		SemanticCandidateCount: result.SemanticCandidateCount,
		EmbeddingSummary:       mapEmbeddingSummary(result.EmbeddingSummary),
	}, nil
}

// GetEmbeddingsStatus returns lifecycle inventory rows and summary counts for the requested scope.
func (a *AppServiceAdapter) GetEmbeddingsStatus(ctx context.Context, in EmbeddingsStatusRequest) (EmbeddingsStatusResult, error) {
	if a == nil || a.service == nil {
		return EmbeddingsStatusResult{}, fmt.Errorf("app service adapter is not configured: %w", ErrInvalidCaptureStateRequest)
	}
	projectIDs, err := appEmbeddingProjectScope(ctx, a.service, strings.TrimSpace(in.ProjectID), in.CrossProject, in.IncludeArchived)
	if err != nil {
		return EmbeddingsStatusResult{}, mapAppError("get embeddings status", err)
	}
	statuses, err := parseEmbeddingLifecycleStatuses(in.Statuses)
	if err != nil {
		return EmbeddingsStatusResult{}, err
	}
	rows, err := a.service.ListEmbeddingStates(ctx, app.EmbeddingListFilter{
		ProjectIDs: projectIDs,
		Statuses:   statuses,
		Limit:      in.Limit,
	})
	if err != nil {
		return EmbeddingsStatusResult{}, mapAppError("get embeddings status", err)
	}
	summary, err := a.service.SummarizeEmbeddingStates(ctx, app.EmbeddingListFilter{
		ProjectIDs: projectIDs,
	})
	if err != nil {
		return EmbeddingsStatusResult{}, mapAppError("get embeddings status", err)
	}
	out := make([]EmbeddingStatusRow, 0, len(rows))
	for _, row := range rows {
		out = append(out, mapEmbeddingStatusRow(row))
	}
	return EmbeddingsStatusResult{
		ProjectIDs:         append([]string(nil), projectIDs...),
		RuntimeOperational: a.service.EmbeddingsOperational(),
		Summary:            mapEmbeddingSummary(summary),
		Rows:               out,
	}, nil
}

// ReindexEmbeddings triggers one explicit backfill/reindex request for the requested scope.
func (a *AppServiceAdapter) ReindexEmbeddings(ctx context.Context, in ReindexEmbeddingsRequest) (ReindexEmbeddingsResult, error) {
	if a == nil || a.service == nil {
		return ReindexEmbeddingsResult{}, fmt.Errorf("app service adapter is not configured: %w", ErrInvalidCaptureStateRequest)
	}
	result, err := a.service.ReindexEmbeddings(ctx, app.ReindexEmbeddingsInput{
		ProjectID:        strings.TrimSpace(in.ProjectID),
		CrossProject:     in.CrossProject,
		IncludeArchived:  in.IncludeArchived,
		Force:            in.Force,
		Wait:             in.Wait,
		WaitTimeout:      in.WaitTimeout,
		WaitPollInterval: in.WaitPollInterval,
	})
	if err != nil {
		return ReindexEmbeddingsResult{}, mapAppError("reindex embeddings", err)
	}
	return ReindexEmbeddingsResult{
		TargetProjects: append([]string(nil), result.TargetProjects...),
		ScannedCount:   result.ScannedCount,
		QueuedCount:    result.QueuedCount,
		ReadyCount:     result.ReadyCount,
		FailedCount:    result.FailedCount,
		StaleCount:     result.StaleCount,
		RunningCount:   result.RunningCount,
		PendingCount:   result.PendingCount,
		Completed:      result.Completed,
		TimedOut:       result.TimedOut,
	}, nil
}

// appEmbeddingProjectScope resolves one embeddings inventory scope to concrete project ids.
func appEmbeddingProjectScope(ctx context.Context, service *app.Service, projectID string, crossProject, includeArchived bool) ([]string, error) {
	if service == nil {
		return nil, fmt.Errorf("app service is not configured")
	}
	if crossProject {
		projects, err := service.ListProjects(ctx, includeArchived)
		if err != nil {
			return nil, err
		}
		out := make([]string, 0, len(projects))
		for _, project := range projects {
			out = append(out, project.ID)
		}
		return out, nil
	}
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		return nil, domain.ErrInvalidID
	}
	projects, err := service.ListProjects(ctx, includeArchived)
	if err != nil {
		return nil, err
	}
	for _, project := range projects {
		if project.ID == projectID {
			return []string{project.ID}, nil
		}
	}
	if !includeArchived {
		allProjects, err := service.ListProjects(ctx, true)
		if err != nil {
			return nil, err
		}
		for _, project := range allProjects {
			if project.ID == projectID {
				return nil, fmt.Errorf("project %q is archived; rerun with include_archived=true", projectID)
			}
		}
	}
	return nil, fmt.Errorf("project %q not found", projectID)
}

// parseEmbeddingLifecycleStatuses maps transport-facing lifecycle filters to app statuses.
func parseEmbeddingLifecycleStatuses(values []string) ([]app.EmbeddingLifecycleStatus, error) {
	out := make([]app.EmbeddingLifecycleStatus, 0, len(values))
	for _, value := range values {
		switch strings.TrimSpace(strings.ToLower(value)) {
		case "":
		case string(app.EmbeddingLifecyclePending):
			out = append(out, app.EmbeddingLifecyclePending)
		case string(app.EmbeddingLifecycleRunning):
			out = append(out, app.EmbeddingLifecycleRunning)
		case string(app.EmbeddingLifecycleReady):
			out = append(out, app.EmbeddingLifecycleReady)
		case string(app.EmbeddingLifecycleFailed):
			out = append(out, app.EmbeddingLifecycleFailed)
		case string(app.EmbeddingLifecycleStale):
			out = append(out, app.EmbeddingLifecycleStale)
		default:
			return nil, fmt.Errorf("unsupported embeddings status %q; allowed values: pending, running, ready, failed, stale", value)
		}
	}
	return out, nil
}

// mapEmbeddingSummary converts one app-layer embedding summary into the transport shape.
func mapEmbeddingSummary(in app.EmbeddingSummary) EmbeddingSummary {
	return EmbeddingSummary{
		SubjectType:  string(in.SubjectType),
		ProjectIDs:   append([]string(nil), in.ProjectIDs...),
		PendingCount: in.PendingCount,
		RunningCount: in.RunningCount,
		ReadyCount:   in.ReadyCount,
		FailedCount:  in.FailedCount,
		StaleCount:   in.StaleCount,
	}
}

// mapEmbeddingStatusRow converts one app-layer lifecycle record into the transport shape.
func mapEmbeddingStatusRow(in app.EmbeddingRecord) EmbeddingStatusRow {
	return EmbeddingStatusRow{
		SubjectType:      string(in.SubjectType),
		SubjectID:        in.SubjectID,
		ProjectID:        in.ProjectID,
		Status:           string(in.Status),
		ModelSignature:   in.ModelSignature,
		NextAttemptAt:    in.NextAttemptAt,
		LastStartedAt:    in.LastStartedAt,
		LastSucceededAt:  in.LastSucceededAt,
		LastFailedAt:     in.LastFailedAt,
		LastErrorSummary: in.LastErrorSummary,
		StaleReason:      in.StaleReason,
		UpdatedAt:        in.UpdatedAt,
	}
}

// ListProjectChangeEvents returns recent change events for one project.
func (a *AppServiceAdapter) ListProjectChangeEvents(ctx context.Context, projectID string, limit int) ([]domain.ChangeEvent, error) {
	if a == nil || a.service == nil {
		return nil, fmt.Errorf("app service adapter is not configured: %w", ErrInvalidCaptureStateRequest)
	}
	events, err := a.service.ListProjectChangeEvents(ctx, strings.TrimSpace(projectID), limit)
	if err != nil {
		return nil, mapAppError("list project change events", err)
	}
	return events, nil
}

// GetProjectDependencyRollup returns dependency counts for one project.
func (a *AppServiceAdapter) GetProjectDependencyRollup(ctx context.Context, projectID string) (domain.DependencyRollup, error) {
	if a == nil || a.service == nil {
		return domain.DependencyRollup{}, fmt.Errorf("app service adapter is not configured: %w", ErrInvalidCaptureStateRequest)
	}
	rollup, err := a.service.GetProjectDependencyRollup(ctx, strings.TrimSpace(projectID))
	if err != nil {
		return domain.DependencyRollup{}, mapAppError("get project dependency rollup", err)
	}
	return rollup, nil
}

// ListKindDefinitions lists kind catalog entries.
func (a *AppServiceAdapter) ListKindDefinitions(ctx context.Context, includeArchived bool) ([]domain.KindDefinition, error) {
	if a == nil || a.service == nil {
		return nil, fmt.Errorf("app service adapter is not configured: %w", ErrInvalidCaptureStateRequest)
	}
	kinds, err := a.service.ListKindDefinitions(ctx, includeArchived)
	if err != nil {
		return nil, mapAppError("list kind definitions", err)
	}
	return kinds, nil
}

// UpsertKindDefinition creates or updates one kind catalog entry.
func (a *AppServiceAdapter) UpsertKindDefinition(ctx context.Context, in UpsertKindDefinitionRequest) (domain.KindDefinition, error) {
	if a == nil || a.service == nil {
		return domain.KindDefinition{}, fmt.Errorf("app service adapter is not configured: %w", ErrInvalidCaptureStateRequest)
	}
	kind, err := a.service.UpsertKindDefinition(ctx, app.CreateKindDefinitionInput{
		ID:                  domain.KindID(strings.TrimSpace(in.ID)),
		DisplayName:         strings.TrimSpace(in.DisplayName),
		DescriptionMarkdown: strings.TrimSpace(in.DescriptionMarkdown),
		AppliesTo:           toKindAppliesToList(in.AppliesTo),
		AllowedParentScopes: toKindAppliesToList(in.AllowedParentScopes),
		PayloadSchemaJSON:   strings.TrimSpace(in.PayloadSchemaJSON),
		Template:            in.Template,
	})
	if err != nil {
		return domain.KindDefinition{}, mapAppError("upsert kind definition", err)
	}
	return kind, nil
}

// SetProjectAllowedKinds updates a project's kind allowlist.
func (a *AppServiceAdapter) SetProjectAllowedKinds(ctx context.Context, in SetProjectAllowedKindsRequest) error {
	if a == nil || a.service == nil {
		return fmt.Errorf("app service adapter is not configured: %w", ErrInvalidCaptureStateRequest)
	}
	if err := a.service.SetProjectAllowedKinds(ctx, app.SetProjectAllowedKindsInput{
		ProjectID: strings.TrimSpace(in.ProjectID),
		KindIDs:   toKindIDList(in.KindIDs),
	}); err != nil {
		return mapAppError("set project allowed kinds", err)
	}
	return nil
}

// ListTemplateLibraries lists template libraries with optional filters.
func (a *AppServiceAdapter) ListTemplateLibraries(ctx context.Context, in ListTemplateLibrariesRequest) ([]domain.TemplateLibrary, error) {
	if a == nil || a.service == nil {
		return nil, fmt.Errorf("app service adapter is not configured: %w", ErrInvalidCaptureStateRequest)
	}
	libraries, err := a.service.ListTemplateLibraries(ctx, app.ListTemplateLibrariesInput{
		Scope:     in.Scope,
		ProjectID: strings.TrimSpace(in.ProjectID),
		Status:    in.Status,
	})
	if err != nil {
		return nil, mapAppError("list template libraries", err)
	}
	return libraries, nil
}

// GetTemplateLibrary loads one template library by id.
func (a *AppServiceAdapter) GetTemplateLibrary(ctx context.Context, libraryID string) (domain.TemplateLibrary, error) {
	if a == nil || a.service == nil {
		return domain.TemplateLibrary{}, fmt.Errorf("app service adapter is not configured: %w", ErrInvalidCaptureStateRequest)
	}
	library, err := a.service.GetTemplateLibrary(ctx, strings.TrimSpace(libraryID))
	if err != nil {
		return domain.TemplateLibrary{}, mapAppError("get template library", err)
	}
	return library, nil
}

// GetBuiltinTemplateLibraryStatus loads one builtin template library lifecycle status view.
func (a *AppServiceAdapter) GetBuiltinTemplateLibraryStatus(ctx context.Context, libraryID string) (domain.BuiltinTemplateLibraryStatus, error) {
	if a == nil || a.service == nil {
		return domain.BuiltinTemplateLibraryStatus{}, fmt.Errorf("app service adapter is not configured: %w", ErrInvalidCaptureStateRequest)
	}
	status, err := a.service.GetBuiltinTemplateLibraryStatus(ctx, strings.TrimSpace(libraryID))
	if err != nil {
		return domain.BuiltinTemplateLibraryStatus{}, mapAppError("get builtin template library status", err)
	}
	return status, nil
}

// EnsureBuiltinTemplateLibrary installs or refreshes one supported builtin template library explicitly.
func (a *AppServiceAdapter) EnsureBuiltinTemplateLibrary(ctx context.Context, in EnsureBuiltinTemplateLibraryRequest) (domain.BuiltinTemplateLibraryEnsureResult, error) {
	if a == nil || a.service == nil {
		return domain.BuiltinTemplateLibraryEnsureResult{}, fmt.Errorf("app service adapter is not configured: %w", ErrInvalidCaptureStateRequest)
	}
	caller, _ := app.AuthenticatedCallerFromContext(ctx)
	result, err := a.service.EnsureBuiltinTemplateLibrary(ctx, app.EnsureBuiltinTemplateLibraryInput{
		LibraryID: strings.TrimSpace(in.LibraryID),
		ActorID:   caller.PrincipalID,
		ActorName: caller.PrincipalName,
		ActorType: caller.PrincipalType,
	})
	if err != nil {
		return domain.BuiltinTemplateLibraryEnsureResult{}, mapAppError("ensure builtin template library", err)
	}
	return result, nil
}

// UpsertTemplateLibrary creates or updates one template library.
func (a *AppServiceAdapter) UpsertTemplateLibrary(ctx context.Context, in UpsertTemplateLibraryRequest) (domain.TemplateLibrary, error) {
	if a == nil || a.service == nil {
		return domain.TemplateLibrary{}, fmt.Errorf("app service adapter is not configured: %w", ErrInvalidCaptureStateRequest)
	}
	nodeTemplates := make([]app.UpsertNodeTemplateInput, 0, len(in.NodeTemplates))
	for _, nodeTemplate := range in.NodeTemplates {
		childRules := make([]app.UpsertTemplateChildRuleInput, 0, len(nodeTemplate.ChildRules))
		for _, childRule := range nodeTemplate.ChildRules {
			childRules = append(childRules, app.UpsertTemplateChildRuleInput{
				ID:                        strings.TrimSpace(childRule.ID),
				Position:                  childRule.Position,
				ChildScopeLevel:           childRule.ChildScopeLevel,
				ChildKindID:               childRule.ChildKindID,
				TitleTemplate:             strings.TrimSpace(childRule.TitleTemplate),
				DescriptionTemplate:       strings.TrimSpace(childRule.DescriptionTemplate),
				ResponsibleActorKind:      childRule.ResponsibleActorKind,
				EditableByActorKinds:      append([]domain.TemplateActorKind(nil), childRule.EditableByActorKinds...),
				CompletableByActorKinds:   append([]domain.TemplateActorKind(nil), childRule.CompletableByActorKinds...),
				OrchestratorMayComplete:   childRule.OrchestratorMayComplete,
				RequiredForParentDone:     childRule.RequiredForParentDone,
				RequiredForContainingDone: childRule.RequiredForContainingDone,
			})
		}
		nodeTemplates = append(nodeTemplates, app.UpsertNodeTemplateInput{
			ID:                      strings.TrimSpace(nodeTemplate.ID),
			ScopeLevel:              nodeTemplate.ScopeLevel,
			NodeKindID:              nodeTemplate.NodeKindID,
			DisplayName:             strings.TrimSpace(nodeTemplate.DisplayName),
			DescriptionMarkdown:     strings.TrimSpace(nodeTemplate.DescriptionMarkdown),
			ProjectMetadataDefaults: nodeTemplate.ProjectMetadataDefaults,
			TaskMetadataDefaults:    nodeTemplate.TaskMetadataDefaults,
			ChildRules:              childRules,
		})
	}
	library, err := a.service.UpsertTemplateLibrary(ctx, app.UpsertTemplateLibraryInput{
		ID:              strings.TrimSpace(in.ID),
		Scope:           in.Scope,
		ProjectID:       strings.TrimSpace(in.ProjectID),
		Name:            strings.TrimSpace(in.Name),
		Description:     strings.TrimSpace(in.Description),
		Status:          in.Status,
		SourceLibraryID: strings.TrimSpace(in.SourceLibraryID),
		BuiltinManaged:  in.BuiltinManaged,
		BuiltinSource:   strings.TrimSpace(in.BuiltinSource),
		BuiltinVersion:  strings.TrimSpace(in.BuiltinVersion),
		NodeTemplates:   nodeTemplates,
	})
	if err != nil {
		return domain.TemplateLibrary{}, mapAppError("upsert template library", err)
	}
	return library, nil
}

// BindProjectTemplateLibrary binds one project to one approved template library.
func (a *AppServiceAdapter) BindProjectTemplateLibrary(ctx context.Context, in BindProjectTemplateLibraryRequest) (domain.ProjectTemplateBinding, error) {
	if a == nil || a.service == nil {
		return domain.ProjectTemplateBinding{}, fmt.Errorf("app service adapter is not configured: %w", ErrInvalidCaptureStateRequest)
	}
	caller, _ := app.AuthenticatedCallerFromContext(ctx)
	binding, err := a.service.BindProjectTemplateLibrary(ctx, app.BindProjectTemplateLibraryInput{
		ProjectID:        strings.TrimSpace(in.ProjectID),
		LibraryID:        strings.TrimSpace(in.LibraryID),
		BoundByActorID:   caller.PrincipalID,
		BoundByActorName: caller.PrincipalName,
		BoundByActorType: caller.PrincipalType,
	})
	if err != nil {
		return domain.ProjectTemplateBinding{}, mapAppError("bind project template library", err)
	}
	return binding, nil
}

// GetProjectTemplateBinding loads one project's active template binding.
func (a *AppServiceAdapter) GetProjectTemplateBinding(ctx context.Context, projectID string) (domain.ProjectTemplateBinding, error) {
	if a == nil || a.service == nil {
		return domain.ProjectTemplateBinding{}, fmt.Errorf("app service adapter is not configured: %w", ErrInvalidCaptureStateRequest)
	}
	binding, err := a.service.GetProjectTemplateBinding(ctx, strings.TrimSpace(projectID))
	if err != nil {
		return domain.ProjectTemplateBinding{}, mapAppError("get project template binding", err)
	}
	return binding, nil
}

// GetProjectTemplateReapplyPreview loads one project's explicit template reapply preview.
func (a *AppServiceAdapter) GetProjectTemplateReapplyPreview(ctx context.Context, projectID string) (domain.ProjectTemplateReapplyPreview, error) {
	if a == nil || a.service == nil {
		return domain.ProjectTemplateReapplyPreview{}, fmt.Errorf("app service adapter is not configured: %w", ErrInvalidCaptureStateRequest)
	}
	preview, err := a.service.GetProjectTemplateReapplyPreview(ctx, strings.TrimSpace(projectID))
	if err != nil {
		return domain.ProjectTemplateReapplyPreview{}, mapAppError("get project template reapply preview", err)
	}
	return preview, nil
}

// ApproveProjectTemplateMigrations applies the latest approved child-rule contract to selected eligible generated nodes.
func (a *AppServiceAdapter) ApproveProjectTemplateMigrations(ctx context.Context, in ApproveProjectTemplateMigrationsRequest) (domain.ProjectTemplateMigrationApprovalResult, error) {
	if a == nil || a.service == nil {
		return domain.ProjectTemplateMigrationApprovalResult{}, fmt.Errorf("app service adapter is not configured: %w", ErrInvalidCaptureStateRequest)
	}
	ctx, actorType, err := withMutationGuardContext(ctx, in.Actor)
	if err != nil {
		return domain.ProjectTemplateMigrationApprovalResult{}, err
	}
	actorID, actorName := deriveMutationActorIdentity(in.Actor)
	result, err := a.service.ApproveProjectTemplateMigrations(ctx, app.ApproveProjectTemplateMigrationsInput{
		ProjectID:      strings.TrimSpace(in.ProjectID),
		TaskIDs:        append([]string(nil), in.TaskIDs...),
		ApproveAll:     in.ApproveAll,
		ApprovedBy:     actorID,
		ApprovedByName: actorName,
		ApprovedByType: actorType,
	})
	if err != nil {
		return domain.ProjectTemplateMigrationApprovalResult{}, mapAppError("approve project template migrations", err)
	}
	return result, nil
}

// GetNodeContractSnapshot loads one generated-node contract snapshot.
func (a *AppServiceAdapter) GetNodeContractSnapshot(ctx context.Context, nodeID string) (domain.NodeContractSnapshot, error) {
	if a == nil || a.service == nil {
		return domain.NodeContractSnapshot{}, fmt.Errorf("app service adapter is not configured: %w", ErrInvalidCaptureStateRequest)
	}
	snapshot, err := a.service.GetNodeContractSnapshot(ctx, strings.TrimSpace(nodeID))
	if err != nil {
		return domain.NodeContractSnapshot{}, mapAppError("get node contract snapshot", err)
	}
	return snapshot, nil
}

// parseOptionalDurationString parses one optional Go duration string used by transport auth request inputs.
func parseOptionalDurationString(raw, field string) (time.Duration, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, nil
	}
	value, err := time.ParseDuration(raw)
	if err != nil {
		return 0, fmt.Errorf("%s %q is invalid: %w", field, raw, ErrInvalidCaptureStateRequest)
	}
	if value < 0 {
		return 0, fmt.Errorf("%s %q is invalid: %w", field, raw, ErrInvalidCaptureStateRequest)
	}
	return value, nil
}

// parseContinuationJSON decodes one optional continuation metadata object encoded as JSON.
func parseContinuationJSON(raw string) (map[string]any, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	var continuation map[string]any
	if err := json.Unmarshal([]byte(raw), &continuation); err != nil {
		return nil, fmt.Errorf("continuation_json is invalid: %w", ErrInvalidCaptureStateRequest)
	}
	return cloneJSONObject(continuation), nil
}

// normalizeCreateAuthRequestContinuation returns one claimable continuation payload for MCP auth-request create flows.
func normalizeCreateAuthRequestContinuation(raw string) (map[string]any, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return map[string]any{"resume_token": generateAuthRequestResumeToken()}, nil
	}
	continuation, err := parseContinuationJSON(raw)
	if err != nil {
		return nil, err
	}
	if authRequestResumeToken(continuation) == "" {
		return nil, fmt.Errorf("continuation_json.resume_token is required when continuation_json is provided: %w", ErrInvalidCaptureStateRequest)
	}
	return continuation, nil
}

// requestedActorType resolves explicit requester attribution and falls back to requested principal type.
func requestedActorType(requestedByType, principalType string) domain.ActorType {
	switch strings.TrimSpace(strings.ToLower(requestedByType)) {
	case string(domain.ActorTypeAgent):
		return domain.ActorTypeAgent
	case string(domain.ActorTypeSystem):
		return domain.ActorTypeSystem
	case string(domain.ActorTypeUser):
		return domain.ActorTypeUser
	}
	switch strings.TrimSpace(strings.ToLower(principalType)) {
	case "agent", "service", "system":
		return domain.ActorTypeAgent
	default:
		return domain.ActorTypeUser
	}
}

// firstNonEmptyRequestedBy returns the first non-empty trimmed requester identifier.
func firstNonEmptyRequestedBy(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

// requesterClientID resolves one requester-bound claim client identifier with fallback to the requested client.
func requesterClientID(in CreateAuthRequestRequest) string {
	if trimmed := strings.TrimSpace(in.RequesterClientID); trimmed != "" {
		return trimmed
	}
	return strings.TrimSpace(in.ClientID)
}

// resolvedAuthRequestActor normalizes auth-request resolver identity for transport-facing lifecycle actions.
func resolvedAuthRequestActor(resolvedByActor, resolvedByType string) (string, domain.ActorType, error) {
	resolvedBy := strings.TrimSpace(resolvedByActor)
	if resolvedBy == "" {
		resolvedBy = "tillsyn-user"
	}
	resolvedType, err := normalizeResolvedActorType(resolvedByType)
	if err != nil {
		return "", "", err
	}
	return resolvedBy, resolvedType, nil
}

// mapAuthRequestRecord converts one domain auth request into one transport-facing record.
func mapAuthRequestRecord(request domain.AuthRequest) AuthRequestRecord {
	approvedSessionTTL := ""
	if request.ApprovedSessionTTL > 0 {
		approvedSessionTTL = request.ApprovedSessionTTL.String()
	}
	return AuthRequestRecord{
		ID:                     request.ID,
		State:                  string(request.State),
		Path:                   request.Path,
		ApprovedPath:           request.ApprovedPath,
		ProjectID:              request.ProjectID,
		BranchID:               request.BranchID,
		PhaseIDs:               append([]string(nil), request.PhaseIDs...),
		ScopeType:              string(request.ScopeType),
		ScopeID:                request.ScopeID,
		PrincipalID:            request.PrincipalID,
		PrincipalType:          request.PrincipalType,
		PrincipalRole:          request.PrincipalRole,
		PrincipalName:          request.PrincipalName,
		ClientID:               request.ClientID,
		ClientType:             request.ClientType,
		ClientName:             request.ClientName,
		RequestedSessionTTL:    request.RequestedSessionTTL.String(),
		ApprovedSessionTTL:     approvedSessionTTL,
		Reason:                 request.Reason,
		HasContinuation:        len(request.Continuation) > 0,
		Continuation:           cloneJSONObject(request.Continuation),
		RequestedByActor:       request.RequestedByActor,
		RequestedByType:        string(request.RequestedByType),
		CreatedAt:              request.CreatedAt.UTC(),
		ExpiresAt:              request.ExpiresAt.UTC(),
		ResolvedByActor:        request.ResolvedByActor,
		ResolvedByType:         string(request.ResolvedByType),
		ResolvedAt:             request.ResolvedAt,
		ResolutionNote:         request.ResolutionNote,
		IssuedSessionID:        request.IssuedSessionID,
		IssuedSessionExpiresAt: request.IssuedSessionExpiresAt,
	}
}

// mapAuthSessionRecord converts one app-facing auth session into one transport-facing record.
func mapAuthSessionRecord(session app.AuthSession, now time.Time) AuthSessionRecord {
	return AuthSessionRecord{
		SessionID:        strings.TrimSpace(session.SessionID),
		State:            authSessionState(session, now),
		ProjectID:        strings.TrimSpace(session.ProjectID),
		AuthRequestID:    strings.TrimSpace(session.AuthRequestID),
		ApprovedPath:     strings.TrimSpace(session.ApprovedPath),
		PrincipalID:      strings.TrimSpace(session.PrincipalID),
		PrincipalType:    strings.TrimSpace(session.PrincipalType),
		PrincipalRole:    strings.TrimSpace(session.PrincipalRole),
		PrincipalName:    strings.TrimSpace(session.PrincipalName),
		ClientID:         strings.TrimSpace(session.ClientID),
		ClientType:       strings.TrimSpace(session.ClientType),
		ClientName:       strings.TrimSpace(session.ClientName),
		IssuedAt:         session.IssuedAt.UTC(),
		ExpiresAt:        session.ExpiresAt.UTC(),
		LastValidatedAt:  session.LastValidatedAt,
		RevokedAt:        session.RevokedAt,
		RevocationReason: strings.TrimSpace(session.RevocationReason),
	}
}

// authSessionState normalizes one auth session into the transport-facing lifecycle label.
func authSessionState(session app.AuthSession, now time.Time) string {
	if session.RevokedAt != nil {
		return "revoked"
	}
	if !session.ExpiresAt.IsZero() && !now.Before(session.ExpiresAt.UTC()) {
		return "expired"
	}
	return "active"
}

// authRequestCancelOwnershipMatches verifies one cancel request uses requester-owned continuation proof.
func authRequestCancelOwnershipMatches(request AuthRequestRecord, principalID, clientID, resumeToken string) error {
	principalID = strings.TrimSpace(principalID)
	clientID = strings.TrimSpace(clientID)
	resumeToken = strings.TrimSpace(resumeToken)
	if principalID == "" || clientID == "" || resumeToken == "" {
		return fmt.Errorf("cancel auth request proof is required: %w", ErrInvalidCaptureStateRequest)
	}
	if strings.TrimSpace(request.RequestedByActor) != principalID {
		return fmt.Errorf("cancel auth request requester mismatch: %w", ErrInvalidCaptureStateRequest)
	}
	if requesterClientID := app.AuthRequestClaimClientIDFromContinuation(request.Continuation, request.ClientID); requesterClientID != clientID {
		return fmt.Errorf("cancel auth request client mismatch: %w", ErrInvalidCaptureStateRequest)
	}
	if !authRequestResumeTokenMatches(request.Continuation, resumeToken) {
		return fmt.Errorf("cancel auth request continuation mismatch: %w", ErrInvalidCaptureStateRequest)
	}
	return nil
}

// authRequestResumeTokenMatches reports whether one continuation payload carries the expected resume token.
func authRequestResumeTokenMatches(continuation map[string]any, want string) bool {
	want = strings.TrimSpace(want)
	if want == "" {
		return false
	}
	return authRequestResumeToken(continuation) == want
}

// authRequestResumeToken returns one normalized requester-owned resume token from continuation metadata.
func authRequestResumeToken(continuation map[string]any) string {
	got, _ := continuation["resume_token"].(string)
	return strings.TrimSpace(got)
}

// generateAuthRequestResumeToken returns one opaque continuation token for MCP claim/cancel ownership proof.
func generateAuthRequestResumeToken() string {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("resume-%d", time.Now().UTC().UnixNano())
	}
	return "resume-" + hex.EncodeToString(buf)
}

// ListProjectAllowedKinds lists canonical kind ids in one project's allowlist.
func (a *AppServiceAdapter) ListProjectAllowedKinds(ctx context.Context, projectID string) ([]string, error) {
	if a == nil || a.service == nil {
		return nil, fmt.Errorf("app service adapter is not configured: %w", ErrInvalidCaptureStateRequest)
	}
	kindIDs, err := a.service.ListProjectAllowedKinds(ctx, strings.TrimSpace(projectID))
	if err != nil {
		return nil, mapAppError("list project allowed kinds", err)
	}
	out := make([]string, 0, len(kindIDs))
	for _, kindID := range kindIDs {
		out = append(out, string(kindID))
	}
	return out, nil
}

// ListCapabilityLeases lists scoped capability leases.
func (a *AppServiceAdapter) ListCapabilityLeases(ctx context.Context, in ListCapabilityLeasesRequest) ([]domain.CapabilityLease, error) {
	if a == nil || a.service == nil {
		return nil, fmt.Errorf("app service adapter is not configured: %w", ErrInvalidCaptureStateRequest)
	}
	leases, err := a.service.ListCapabilityLeases(ctx, app.ListCapabilityLeasesInput{
		ProjectID:      strings.TrimSpace(in.ProjectID),
		ScopeType:      domain.CapabilityScopeType(strings.TrimSpace(in.ScopeType)),
		ScopeID:        strings.TrimSpace(in.ScopeID),
		IncludeRevoked: in.IncludeRevoked,
	})
	if err != nil {
		return nil, mapAppError("list capability leases", err)
	}
	return leases, nil
}

// IssueCapabilityLease issues one scope-bound capability lease.
func (a *AppServiceAdapter) IssueCapabilityLease(ctx context.Context, in IssueCapabilityLeaseRequest) (domain.CapabilityLease, error) {
	if a == nil || a.service == nil {
		return domain.CapabilityLease{}, fmt.Errorf("app service adapter is not configured: %w", ErrInvalidCaptureStateRequest)
	}
	lease, err := a.service.IssueCapabilityLease(ctx, app.IssueCapabilityLeaseInput{
		ProjectID:                 strings.TrimSpace(in.ProjectID),
		ScopeType:                 domain.CapabilityScopeType(strings.TrimSpace(in.ScopeType)),
		ScopeID:                   strings.TrimSpace(in.ScopeID),
		Role:                      domain.CapabilityRole(strings.TrimSpace(in.Role)),
		AgentName:                 strings.TrimSpace(in.AgentName),
		AgentInstanceID:           strings.TrimSpace(in.AgentInstanceID),
		ParentInstanceID:          strings.TrimSpace(in.ParentInstanceID),
		AllowEqualScopeDelegation: in.AllowEqualScopeDelegation,
		RequestedTTL:              durationFromSeconds(in.RequestedTTLSeconds),
		OverrideToken:             strings.TrimSpace(in.OverrideToken),
	})
	if err != nil {
		return domain.CapabilityLease{}, mapAppError("issue capability lease", err)
	}
	return lease, nil
}

// HeartbeatCapabilityLease records one heartbeat against an active lease.
func (a *AppServiceAdapter) HeartbeatCapabilityLease(ctx context.Context, in HeartbeatCapabilityLeaseRequest) (domain.CapabilityLease, error) {
	if a == nil || a.service == nil {
		return domain.CapabilityLease{}, fmt.Errorf("app service adapter is not configured: %w", ErrInvalidCaptureStateRequest)
	}
	lease, err := a.service.HeartbeatCapabilityLease(ctx, app.HeartbeatCapabilityLeaseInput{
		AgentInstanceID: strings.TrimSpace(in.AgentInstanceID),
		LeaseToken:      strings.TrimSpace(in.LeaseToken),
	})
	if err != nil {
		return domain.CapabilityLease{}, mapAppError("heartbeat capability lease", err)
	}
	return lease, nil
}

// RenewCapabilityLease extends one lease expiry.
func (a *AppServiceAdapter) RenewCapabilityLease(ctx context.Context, in RenewCapabilityLeaseRequest) (domain.CapabilityLease, error) {
	if a == nil || a.service == nil {
		return domain.CapabilityLease{}, fmt.Errorf("app service adapter is not configured: %w", ErrInvalidCaptureStateRequest)
	}
	lease, err := a.service.RenewCapabilityLease(ctx, app.RenewCapabilityLeaseInput{
		AgentInstanceID: strings.TrimSpace(in.AgentInstanceID),
		LeaseToken:      strings.TrimSpace(in.LeaseToken),
		TTL:             durationFromSeconds(in.TTLSeconds),
	})
	if err != nil {
		return domain.CapabilityLease{}, mapAppError("renew capability lease", err)
	}
	return lease, nil
}

// RevokeCapabilityLease revokes one lease by instance id.
func (a *AppServiceAdapter) RevokeCapabilityLease(ctx context.Context, in RevokeCapabilityLeaseRequest) (domain.CapabilityLease, error) {
	if a == nil || a.service == nil {
		return domain.CapabilityLease{}, fmt.Errorf("app service adapter is not configured: %w", ErrInvalidCaptureStateRequest)
	}
	lease, err := a.service.RevokeCapabilityLease(ctx, app.RevokeCapabilityLeaseInput{
		AgentInstanceID: strings.TrimSpace(in.AgentInstanceID),
		Reason:          strings.TrimSpace(in.Reason),
	})
	if err != nil {
		return domain.CapabilityLease{}, mapAppError("revoke capability lease", err)
	}
	return lease, nil
}

// RevokeAllCapabilityLeases revokes all matching leases for one scope tuple.
func (a *AppServiceAdapter) RevokeAllCapabilityLeases(ctx context.Context, in RevokeAllCapabilityLeasesRequest) error {
	if a == nil || a.service == nil {
		return fmt.Errorf("app service adapter is not configured: %w", ErrInvalidCaptureStateRequest)
	}
	if err := a.service.RevokeAllCapabilityLeases(ctx, app.RevokeAllCapabilityLeasesInput{
		ProjectID: strings.TrimSpace(in.ProjectID),
		ScopeType: domain.CapabilityScopeType(strings.TrimSpace(in.ScopeType)),
		ScopeID:   strings.TrimSpace(in.ScopeID),
		Reason:    strings.TrimSpace(in.Reason),
	}); err != nil {
		return mapAppError("revoke all capability leases", err)
	}
	return nil
}

// CreateComment creates one markdown-rich comment for a concrete target.
func (a *AppServiceAdapter) CreateComment(ctx context.Context, in CreateCommentRequest) (CommentRecord, error) {
	if a == nil || a.service == nil {
		return CommentRecord{}, fmt.Errorf("app service adapter is not configured: %w", ErrInvalidCaptureStateRequest)
	}
	summary := strings.TrimSpace(in.Summary)
	if summary == "" {
		return CommentRecord{}, fmt.Errorf("summary is required: %w", ErrInvalidCaptureStateRequest)
	}
	ctx, actorType, err := withMutationGuardContext(ctx, in.Actor)
	if err != nil {
		return CommentRecord{}, err
	}
	actorID, actorName := deriveMutationActorIdentity(in.Actor)
	comment, err := a.service.CreateComment(ctx, app.CreateCommentInput{
		ProjectID:    strings.TrimSpace(in.ProjectID),
		TargetType:   domain.CommentTargetType(strings.TrimSpace(in.TargetType)),
		TargetID:     strings.TrimSpace(in.TargetID),
		BodyMarkdown: buildCommentBodyMarkdown(summary, in.BodyMarkdown),
		ActorID:      actorID,
		ActorName:    actorName,
		ActorType:    actorType,
	})
	if err != nil {
		return CommentRecord{}, mapAppError("create comment", err)
	}
	record := mapDomainCommentRecord(comment)
	record.Summary = summary
	return record, nil
}

// ListCommentsByTarget lists comments for one concrete target.
func (a *AppServiceAdapter) ListCommentsByTarget(ctx context.Context, in ListCommentsByTargetRequest) ([]CommentRecord, error) {
	if a == nil || a.service == nil {
		return nil, fmt.Errorf("app service adapter is not configured: %w", ErrInvalidCaptureStateRequest)
	}
	waitTimeout, err := parseOptionalDurationString(in.WaitTimeout, "wait_timeout")
	if err != nil {
		return nil, err
	}
	comments, err := a.service.ListCommentsByTarget(ctx, app.ListCommentsByTargetInput{
		ProjectID:   strings.TrimSpace(in.ProjectID),
		TargetType:  domain.CommentTargetType(strings.TrimSpace(in.TargetType)),
		TargetID:    strings.TrimSpace(in.TargetID),
		WaitTimeout: waitTimeout,
	})
	if err != nil {
		return nil, mapAppError("list comments by target", err)
	}
	out := make([]CommentRecord, 0, len(comments))
	for _, comment := range comments {
		out = append(out, mapDomainCommentRecord(comment))
	}
	return out, nil
}

// mapDomainCommentRecord maps one domain comment into the transport comment contract.
func mapDomainCommentRecord(comment domain.Comment) CommentRecord {
	return CommentRecord{
		ID:           comment.ID,
		ProjectID:    comment.ProjectID,
		TargetType:   string(comment.TargetType),
		TargetID:     comment.TargetID,
		Summary:      commentSummaryFromMarkdown(comment.BodyMarkdown),
		BodyMarkdown: comment.BodyMarkdown,
		ActorID:      comment.ActorID,
		ActorName:    comment.ActorName,
		ActorType:    string(comment.ActorType),
		CreatedAt:    comment.CreatedAt.UTC(),
		UpdatedAt:    comment.UpdatedAt.UTC(),
	}
}

// CreateHandoff creates one durable handoff record.
func (a *AppServiceAdapter) CreateHandoff(ctx context.Context, in CreateHandoffRequest) (domain.Handoff, error) {
	if a == nil || a.service == nil {
		return domain.Handoff{}, fmt.Errorf("app service adapter is not configured: %w", ErrInvalidCaptureStateRequest)
	}
	summary := strings.TrimSpace(in.Summary)
	if summary == "" {
		return domain.Handoff{}, fmt.Errorf("summary is required: %w", ErrInvalidCaptureStateRequest)
	}
	ctx, actorType, err := withMutationGuardContext(ctx, in.Actor)
	if err != nil {
		return domain.Handoff{}, err
	}
	actorID, _ := deriveMutationActorIdentity(in.Actor)
	handoff, err := a.service.CreateHandoff(ctx, app.CreateHandoffInput{
		Level: domain.LevelTupleInput{
			ProjectID: strings.TrimSpace(in.ProjectID),
			BranchID:  strings.TrimSpace(in.BranchID),
			ScopeType: domain.ScopeLevel(strings.TrimSpace(in.ScopeType)),
			ScopeID:   strings.TrimSpace(in.ScopeID),
		},
		SourceRole:      strings.TrimSpace(in.SourceRole),
		TargetBranchID:  strings.TrimSpace(in.TargetBranchID),
		TargetScopeType: domain.ScopeLevel(strings.TrimSpace(in.TargetScopeType)),
		TargetScopeID:   strings.TrimSpace(in.TargetScopeID),
		TargetRole:      strings.TrimSpace(in.TargetRole),
		Status:          domain.HandoffStatus(strings.TrimSpace(in.Status)),
		Summary:         summary,
		NextAction:      strings.TrimSpace(in.NextAction),
		MissingEvidence: append([]string(nil), in.MissingEvidence...),
		RelatedRefs:     append([]string(nil), in.RelatedRefs...),
		CreatedBy:       actorID,
		CreatedType:     actorType,
	})
	if err != nil {
		return domain.Handoff{}, mapAppError("create handoff", err)
	}
	return handoff, nil
}

// GetHandoff returns one durable handoff by id.
func (a *AppServiceAdapter) GetHandoff(ctx context.Context, handoffID string) (domain.Handoff, error) {
	if a == nil || a.service == nil {
		return domain.Handoff{}, fmt.Errorf("app service adapter is not configured: %w", ErrInvalidCaptureStateRequest)
	}
	handoff, err := a.service.GetHandoff(ctx, strings.TrimSpace(handoffID))
	if err != nil {
		return domain.Handoff{}, mapAppError("get handoff", err)
	}
	return handoff, nil
}

// ListHandoffs lists durable handoffs for one scope tuple.
func (a *AppServiceAdapter) ListHandoffs(ctx context.Context, in ListHandoffsRequest) ([]domain.Handoff, error) {
	if a == nil || a.service == nil {
		return nil, fmt.Errorf("app service adapter is not configured: %w", ErrInvalidCaptureStateRequest)
	}
	waitTimeout, err := parseOptionalDurationString(in.WaitTimeout, "wait_timeout")
	if err != nil {
		return nil, err
	}
	handoffs, err := a.service.ListHandoffs(ctx, app.ListHandoffsInput{
		Level: domain.LevelTupleInput{
			ProjectID: strings.TrimSpace(in.ProjectID),
			BranchID:  strings.TrimSpace(in.BranchID),
			ScopeType: domain.ScopeLevel(strings.TrimSpace(in.ScopeType)),
			ScopeID:   strings.TrimSpace(in.ScopeID),
		},
		Statuses:    toHandoffStatusList(in.Statuses),
		Limit:       in.Limit,
		WaitTimeout: waitTimeout,
	})
	if err != nil {
		return nil, mapAppError("list handoffs", err)
	}
	return handoffs, nil
}

// UpdateHandoff updates one durable handoff.
func (a *AppServiceAdapter) UpdateHandoff(ctx context.Context, in UpdateHandoffRequest) (domain.Handoff, error) {
	if a == nil || a.service == nil {
		return domain.Handoff{}, fmt.Errorf("app service adapter is not configured: %w", ErrInvalidCaptureStateRequest)
	}
	ctx, actorType, err := withMutationGuardContext(ctx, in.Actor)
	if err != nil {
		return domain.Handoff{}, err
	}
	actorID, _ := deriveMutationActorIdentity(in.Actor)
	handoff, err := a.service.UpdateHandoff(ctx, app.UpdateHandoffInput{
		HandoffID:       strings.TrimSpace(in.HandoffID),
		Status:          domain.HandoffStatus(strings.TrimSpace(in.Status)),
		SourceRole:      strings.TrimSpace(in.SourceRole),
		TargetBranchID:  strings.TrimSpace(in.TargetBranchID),
		TargetScopeType: domain.ScopeLevel(strings.TrimSpace(in.TargetScopeType)),
		TargetScopeID:   strings.TrimSpace(in.TargetScopeID),
		TargetRole:      strings.TrimSpace(in.TargetRole),
		Summary:         strings.TrimSpace(in.Summary),
		NextAction:      strings.TrimSpace(in.NextAction),
		MissingEvidence: append([]string(nil), in.MissingEvidence...),
		RelatedRefs:     append([]string(nil), in.RelatedRefs...),
		UpdatedBy:       actorID,
		UpdatedType:     actorType,
		ResolvedBy:      actorID,
		ResolvedType:    actorType,
		ResolutionNote:  strings.TrimSpace(in.ResolutionNote),
	})
	if err != nil {
		return domain.Handoff{}, mapAppError("update handoff", err)
	}
	return handoff, nil
}

// commentSummaryFromMarkdown extracts one deterministic summary line from markdown text.
func commentSummaryFromMarkdown(markdown string) string {
	lines := strings.Split(strings.TrimSpace(markdown), "\n")
	for _, line := range lines {
		candidate := strings.TrimSpace(line)
		candidate = strings.TrimLeft(candidate, "#>*-` ")
		candidate = strings.TrimSpace(candidate)
		if candidate != "" {
			return candidate
		}
	}
	return ""
}

// toHandoffStatusList normalizes transport handoff status values.
func toHandoffStatusList(values []string) []domain.HandoffStatus {
	out := make([]domain.HandoffStatus, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		out = append(out, domain.HandoffStatus(trimmed))
	}
	return out
}

// buildCommentBodyMarkdown combines summary and optional markdown details into one comment body.
func buildCommentBodyMarkdown(summary, bodyMarkdown string) string {
	summary = strings.TrimSpace(summary)
	bodyMarkdown = strings.TrimSpace(bodyMarkdown)
	switch {
	case summary == "":
		return bodyMarkdown
	case bodyMarkdown == "":
		return summary
	case bodyMarkdown == summary:
		return summary
	default:
		return summary + "\n\n" + bodyMarkdown
	}
}

// parseOptionalRFC3339 parses one optional RFC3339 timestamp string.
func parseOptionalRFC3339(raw string) (*time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	ts, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return nil, fmt.Errorf("due_at must be RFC3339: %w", ErrInvalidCaptureStateRequest)
	}
	utc := ts.UTC()
	return &utc, nil
}

// withMutationGuardContext validates actor tuple semantics and optionally attaches lease guard context.
func withMutationGuardContext(ctx context.Context, actor ActorLeaseTuple) (context.Context, domain.ActorType, error) {
	return withMutationGuardContextAllowUnguardedAgent(ctx, actor, false)
}

// withMutationGuardContextAllowUnguardedAgent validates actor tuple semantics and
// optionally permits agent session identity without a lease guard tuple for
// global-admin mutations such as project creation.
func withMutationGuardContextAllowUnguardedAgent(ctx context.Context, actor ActorLeaseTuple, allowUnguardedAgent bool) (context.Context, domain.ActorType, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	actorType := normalizeActorType(actor.ActorType)
	if !isValidActorType(actorType) {
		return nil, "", fmt.Errorf("actor_type %q is unsupported: %w", actor.ActorType, ErrInvalidCaptureStateRequest)
	}

	agentName := strings.TrimSpace(actor.AgentName)
	agentInstanceID := strings.TrimSpace(actor.AgentInstanceID)
	leaseToken := strings.TrimSpace(actor.LeaseToken)
	overrideToken := strings.TrimSpace(actor.OverrideToken)
	hasGuardTuple := agentInstanceID != "" || leaseToken != "" || overrideToken != ""
	if hasGuardTuple && actorType == domain.ActorTypeUser {
		return nil, "", fmt.Errorf("actor_type=user cannot be used with guarded mutation tuple: %w", ErrInvalidCaptureStateRequest)
	}
	if actorType != domain.ActorTypeUser || hasGuardTuple {
		if allowUnguardedAgent && actorType == domain.ActorTypeAgent && !hasGuardTuple {
			goto attachIdentity
		}
		if agentName == "" || agentInstanceID == "" || leaseToken == "" {
			return nil, "", fmt.Errorf("agent_name, agent_instance_id, and lease_token are required for non-user or guarded mutations: %w", ErrInvalidCaptureStateRequest)
		}
		ctx = app.WithMutationGuard(ctx, app.MutationGuard{
			AgentName:       agentName,
			AgentInstanceID: agentInstanceID,
			LeaseToken:      leaseToken,
			OverrideToken:   overrideToken,
		})
	}
attachIdentity:
	hasIdentityInput := strings.TrimSpace(actor.ActorID) != "" ||
		strings.TrimSpace(actor.ActorName) != "" ||
		agentName != "" ||
		agentInstanceID != ""
	if hasIdentityInput {
		actorID, actorName := deriveMutationActorIdentity(actor)
		ctx = app.WithAuthenticatedCaller(ctx, domain.AuthenticatedCaller{
			PrincipalID:   actorID,
			PrincipalName: actorName,
			PrincipalType: actorType,
		})
	}
	return ctx, actorType, nil
}

// deriveMutationActorIdentity resolves deterministic actor tuple values for mutating requests.
// Transport adapters should populate this from authenticated session identity whenever available.
func deriveMutationActorIdentity(actor ActorLeaseTuple) (string, string) {
	actorID := strings.TrimSpace(actor.ActorID)
	if actorID == "" {
		actorID = strings.TrimSpace(actor.AgentInstanceID)
	}
	if actorID == "" {
		actorID = strings.TrimSpace(actor.AgentName)
	}
	if actorID == "" {
		actorID = "tillsyn-user"
	}
	actorName := strings.TrimSpace(actor.ActorName)
	if actorName == "" {
		actorName = strings.TrimSpace(actor.AgentName)
	}
	if actorName == "" {
		actorName = actorID
	}
	return actorID, actorName
}

// normalizeActorType canonicalizes actor type values and defaults to user.
func normalizeActorType(actorType string) domain.ActorType {
	normalized := domain.ActorType(strings.TrimSpace(strings.ToLower(actorType)))
	if normalized == "" {
		return domain.ActorTypeUser
	}
	return normalized
}

// normalizeResolvedActorType canonicalizes auth-request resolution actor types and defaults to user.
func normalizeResolvedActorType(actorType string) (domain.ActorType, error) {
	normalized := domain.ActorType(strings.TrimSpace(strings.ToLower(actorType)))
	if normalized == "" {
		return domain.ActorTypeUser, nil
	}
	switch normalized {
	case domain.ActorTypeUser, domain.ActorTypeAgent, domain.ActorTypeSystem:
		return normalized, nil
	default:
		return "", fmt.Errorf("resolved_by_type %q is unsupported: %w", actorType, ErrInvalidCaptureStateRequest)
	}
}

// isValidActorType reports whether actor type values are supported by app/domain rules.
func isValidActorType(actorType domain.ActorType) bool {
	switch actorType {
	case domain.ActorTypeUser, domain.ActorTypeAgent:
		return true
	default:
		return false
	}
}

// toKindAppliesToList maps string scope values into domain kind applies_to values.
func toKindAppliesToList(scopes []string) []domain.KindAppliesTo {
	out := make([]domain.KindAppliesTo, 0, len(scopes))
	for _, scope := range scopes {
		out = append(out, domain.KindAppliesTo(strings.TrimSpace(scope)))
	}
	return out
}

// toKindIDList maps string kind ids into domain kind ids.
func toKindIDList(kindIDs []string) []domain.KindID {
	out := make([]domain.KindID, 0, len(kindIDs))
	for _, kindID := range kindIDs {
		out = append(out, domain.KindID(strings.TrimSpace(kindID)))
	}
	return out
}

// cloneJSONObject deep-copies one JSON-compatible object map.
func cloneJSONObject(in map[string]any) map[string]any {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]any, len(in))
	for key, value := range in {
		out[key] = cloneJSONValue(value)
	}
	return out
}

// cloneJSONValue deep-copies one JSON-compatible nested value.
func cloneJSONValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		return cloneJSONObject(typed)
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
