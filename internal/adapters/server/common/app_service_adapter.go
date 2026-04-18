package common

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/evanmschultz/tillsyn/internal/adapters/auth/autentauth"
	"github.com/evanmschultz/tillsyn/internal/app"
	"github.com/evanmschultz/tillsyn/internal/domain"
)

// AppServiceAdapter maps transport contracts onto app.Service capture_state and attention APIs.
type AppServiceAdapter struct {
	service *app.Service
	auth    *autentauth.Service
}

// NewAppServiceAdapter builds one common adapter over an app.Service instance.
func NewAppServiceAdapter(service *app.Service, auth *autentauth.Service) *AppServiceAdapter {
	return &AppServiceAdapter{
		service: service,
		auth:    auth,
	}
}

// AuthorizeMutation resolves one authenticated caller for a mutating MCP request.
func (a *AppServiceAdapter) AuthorizeMutation(ctx context.Context, in MutationAuthorizationRequest) (domain.AuthenticatedCaller, error) {
	if a == nil || a.auth == nil {
		return domain.AuthenticatedCaller{}, fmt.Errorf("mutation auth is not configured")
	}
	req, err := a.normalizeMutationAuthorizationRequest(ctx, in)
	if err != nil {
		return domain.AuthenticatedCaller{}, err
	}
	result, err := a.auth.Authorize(ctx, autentauth.AuthorizationRequest{
		SessionID:     req.SessionID,
		SessionSecret: req.SessionSecret,
		Action:        req.Action,
		Namespace:     req.Namespace,
		ResourceType:  req.ResourceType,
		ResourceID:    req.ResourceID,
		Context:       req.Context,
	})
	if err != nil {
		return domain.AuthenticatedCaller{}, err
	}
	switch result.DecisionCode {
	case "allow":
		return domain.NormalizeAuthenticatedCaller(result.Caller), nil
	case "session_required":
		return domain.AuthenticatedCaller{}, fmt.Errorf("session is required: %w", ErrSessionRequired)
	case "session_expired":
		return domain.AuthenticatedCaller{}, fmt.Errorf("session expired: %w", ErrSessionExpired)
	case "grant_required":
		if strings.TrimSpace(result.GrantID) != "" {
			return domain.AuthenticatedCaller{}, fmt.Errorf("grant %q is required: %w", result.GrantID, ErrGrantRequired)
		}
		return domain.AuthenticatedCaller{}, fmt.Errorf("grant approval is required: %w", ErrGrantRequired)
	case "deny":
		return domain.AuthenticatedCaller{}, fmt.Errorf("auth denied: %w", ErrAuthorizationDenied)
	case "invalid":
		return domain.AuthenticatedCaller{}, fmt.Errorf("invalid session or secret: %w", ErrInvalidAuthentication)
	default:
		return domain.AuthenticatedCaller{}, fmt.Errorf("unsupported auth decision %q", result.DecisionCode)
	}
}

// CaptureState resolves one summary-first capture_state snapshot through app-level APIs.
func (a *AppServiceAdapter) CaptureState(ctx context.Context, in CaptureStateRequest) (CaptureState, error) {
	if a == nil || a.service == nil {
		return CaptureState{}, fmt.Errorf("app service adapter is not configured: %w", ErrInvalidCaptureStateRequest)
	}

	req, err := normalizeCaptureStateRequest(in)
	if err != nil {
		return CaptureState{}, err
	}

	summary, err := a.service.CaptureState(ctx, app.CaptureStateInput{
		Level: domain.LevelTupleInput{
			ProjectID: req.ProjectID,
			ScopeType: domain.ScopeLevel(req.ScopeType),
			ScopeID:   req.ScopeID,
		},
		View: app.CaptureStateView(req.View),
	})
	if err != nil {
		if errors.Is(err, app.ErrNotFound) {
			projects, listErr := a.service.ListProjects(ctx, true)
			if listErr == nil && len(projects) == 0 {
				return CaptureState{}, fmt.Errorf("capture state: %w", errors.Join(ErrBootstrapRequired, err))
			}
		}
		return CaptureState{}, mapAppError("capture state", err)
	}

	project, err := a.lookupProject(ctx, req.ProjectID)
	if err != nil {
		return CaptureState{}, err
	}
	commentOverview, err := a.buildCommentOverview(ctx, summary.Level)
	if err != nil {
		return CaptureState{}, err
	}

	out, err := convertCaptureStateSummary(summary, req, project, commentOverview)
	if err != nil {
		return CaptureState{}, err
	}
	return out, nil
}

// ListAttentionItems lists scoped attention items through app-level APIs.
func (a *AppServiceAdapter) ListAttentionItems(ctx context.Context, in ListAttentionItemsRequest) ([]AttentionItem, error) {
	if a == nil || a.service == nil {
		return nil, fmt.Errorf("app service adapter is not configured: %w", ErrAttentionUnavailable)
	}

	req, err := normalizeAttentionListRequest(in)
	if err != nil {
		return nil, err
	}

	listInput := app.ListAttentionItemsInput{
		Level: domain.LevelTupleInput{
			ProjectID: req.ProjectID,
			ScopeType: domain.ScopeLevel(req.ScopeType),
			ScopeID:   req.ScopeID,
		},
		AllScopes:  req.AllScopes,
		TargetRole: req.TargetRole,
	}
	waitTimeout, err := parseOptionalDurationString(req.WaitTimeout, "wait_timeout")
	if err != nil {
		return nil, err
	}
	listInput.WaitTimeout = waitTimeout
	if req.State != "" {
		listInput.States = []domain.AttentionState{domain.AttentionState(req.State)}
	}

	items, err := a.service.ListAttentionItems(ctx, listInput)
	if err != nil {
		return nil, mapAppError("list attention items", err)
	}
	return mapDomainAttentionItems(items), nil
}

// RaiseAttentionItem creates one scoped attention item through app-level APIs.
func (a *AppServiceAdapter) RaiseAttentionItem(ctx context.Context, in RaiseAttentionItemRequest) (AttentionItem, error) {
	if a == nil || a.service == nil {
		return AttentionItem{}, fmt.Errorf("app service adapter is not configured: %w", ErrAttentionUnavailable)
	}

	req, err := normalizeRaiseAttentionItemRequest(in)
	if err != nil {
		return AttentionItem{}, err
	}
	ctx, actorType, err := withMutationGuardContext(ctx, req.Actor)
	if err != nil {
		return AttentionItem{}, err
	}
	actorID, _ := deriveMutationActorIdentity(req.Actor)

	item, err := a.service.RaiseAttentionItem(ctx, app.RaiseAttentionItemInput{
		Level: domain.LevelTupleInput{
			ProjectID: req.ProjectID,
			ScopeType: domain.ScopeLevel(req.ScopeType),
			ScopeID:   req.ScopeID,
		},
		Kind:               domain.AttentionKind(req.Kind),
		Summary:            req.Summary,
		BodyMarkdown:       req.BodyMarkdown,
		TargetRole:         req.TargetRole,
		RequiresUserAction: req.RequiresUserAction,
		CreatedBy:          actorID,
		CreatedType:        actorType,
	})
	if err != nil {
		return AttentionItem{}, mapAppError("raise attention item", err)
	}
	return mapDomainAttentionItem(item), nil
}

// ResolveAttentionItem resolves one attention item through app-level APIs.
func (a *AppServiceAdapter) ResolveAttentionItem(ctx context.Context, in ResolveAttentionItemRequest) (AttentionItem, error) {
	if a == nil || a.service == nil {
		return AttentionItem{}, fmt.Errorf("app service adapter is not configured: %w", ErrAttentionUnavailable)
	}

	req, err := normalizeResolveAttentionItemRequest(in)
	if err != nil {
		return AttentionItem{}, err
	}
	ctx, actorType, err := withMutationGuardContext(ctx, req.Actor)
	if err != nil {
		return AttentionItem{}, err
	}
	actorID, _ := deriveMutationActorIdentity(req.Actor)
	item, err := a.service.ResolveAttentionItem(ctx, app.ResolveAttentionItemInput{
		AttentionID:  req.ID,
		ResolvedBy:   actorID,
		ResolvedType: actorType,
	})
	if err != nil {
		return AttentionItem{}, mapAppError("resolve attention item", err)
	}
	return mapDomainAttentionItem(item), nil
}

// lookupProject resolves one project by id for response decoration.
func (a *AppServiceAdapter) lookupProject(ctx context.Context, projectID string) (domain.Project, error) {
	projects, err := a.service.ListProjects(ctx, true)
	if err != nil {
		return domain.Project{}, mapAppError("list projects", err)
	}
	project, ok := findProjectByID(projects, projectID)
	if !ok {
		return domain.Project{}, fmt.Errorf("project %q: %w", projectID, ErrNotFound)
	}
	return project, nil
}

// normalizeAttentionListRequest validates and canonicalizes list_attention_items input.
func normalizeAttentionListRequest(in ListAttentionItemsRequest) (ListAttentionItemsRequest, error) {
	state, err := normalizeAttentionStateFilter(in.State)
	if err != nil {
		return ListAttentionItemsRequest{}, err
	}
	projectID := strings.TrimSpace(in.ProjectID)
	if projectID == "" {
		return ListAttentionItemsRequest{}, fmt.Errorf("project_id is required: %w", ErrInvalidCaptureStateRequest)
	}
	scopeType := strings.ToLower(strings.TrimSpace(in.ScopeType))
	scopeID := strings.TrimSpace(in.ScopeID)
	if !in.AllScopes {
		projectID, scopeType, scopeID, err = normalizeScopeTuple(projectID, scopeType, scopeID)
		if err != nil {
			return ListAttentionItemsRequest{}, err
		}
	} else if scopeType != "" || scopeID != "" {
		return ListAttentionItemsRequest{}, fmt.Errorf("scope_type and scope_id are unsupported when all_scopes is true: %w", ErrUnsupportedScope)
	}
	return ListAttentionItemsRequest{
		ProjectID:   projectID,
		ScopeType:   scopeType,
		ScopeID:     scopeID,
		State:       state,
		AllScopes:   in.AllScopes,
		TargetRole:  strings.TrimSpace(strings.ToLower(in.TargetRole)),
		WaitTimeout: strings.TrimSpace(in.WaitTimeout),
	}, nil
}

// normalizeRaiseAttentionItemRequest validates and canonicalizes raise_attention_item input.
func normalizeRaiseAttentionItemRequest(in RaiseAttentionItemRequest) (RaiseAttentionItemRequest, error) {
	projectID := strings.TrimSpace(in.ProjectID)
	if projectID == "" {
		return RaiseAttentionItemRequest{}, fmt.Errorf("project_id is required: %w", ErrInvalidCaptureStateRequest)
	}
	if strings.TrimSpace(in.ScopeType) == "" {
		return RaiseAttentionItemRequest{}, fmt.Errorf("scope_type is required: %w", ErrUnsupportedScope)
	}
	if strings.TrimSpace(in.ScopeID) == "" {
		return RaiseAttentionItemRequest{}, fmt.Errorf("scope_id is required: %w", ErrUnsupportedScope)
	}
	_, scopeType, scopeID, err := normalizeScopeTuple(projectID, in.ScopeType, in.ScopeID)
	if err != nil {
		return RaiseAttentionItemRequest{}, err
	}

	kind := string(domain.NormalizeAttentionKind(domain.AttentionKind(in.Kind)))
	if kind == "" {
		return RaiseAttentionItemRequest{}, fmt.Errorf("kind is required: %w", ErrInvalidCaptureStateRequest)
	}
	summary := strings.TrimSpace(in.Summary)
	if summary == "" {
		return RaiseAttentionItemRequest{}, fmt.Errorf("summary is required: %w", ErrInvalidCaptureStateRequest)
	}

	return RaiseAttentionItemRequest{
		ProjectID:          projectID,
		ScopeType:          scopeType,
		ScopeID:            scopeID,
		Kind:               kind,
		Summary:            summary,
		BodyMarkdown:       strings.TrimSpace(in.BodyMarkdown),
		TargetRole:         strings.TrimSpace(strings.ToLower(in.TargetRole)),
		RequiresUserAction: in.RequiresUserAction,
		Actor:              in.Actor,
	}, nil
}

// normalizeResolveAttentionItemRequest validates and canonicalizes resolve_attention_item input.
func normalizeResolveAttentionItemRequest(in ResolveAttentionItemRequest) (ResolveAttentionItemRequest, error) {
	itemID := strings.TrimSpace(in.ID)
	if itemID == "" {
		return ResolveAttentionItemRequest{}, fmt.Errorf("id is required: %w", ErrInvalidCaptureStateRequest)
	}
	return ResolveAttentionItemRequest{
		ID:     itemID,
		Reason: strings.TrimSpace(in.Reason),
		Actor:  in.Actor,
	}, nil
}

// cloneStringMap deep-copies string maps used by auth and request normalization helpers.
func cloneStringMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

// normalizeScopeTuple validates and canonicalizes one project/scope tuple.
func normalizeScopeTuple(projectID, scopeType, scopeID string) (string, string, string, error) {
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		return "", "", "", fmt.Errorf("project_id is required: %w", ErrInvalidCaptureStateRequest)
	}

	scopeType = strings.ToLower(strings.TrimSpace(scopeType))
	if scopeType == "" {
		scopeType = ScopeTypeProject
	}
	if !slices.Contains(supportedScopeTypes, scopeType) {
		return "", "", "", fmt.Errorf("scope_type %q is unsupported: %w", scopeType, ErrUnsupportedScope)
	}

	scopeID = strings.TrimSpace(scopeID)
	switch scopeType {
	case ScopeTypeProject:
		if scopeID == "" {
			scopeID = projectID
		}
		if scopeID != projectID {
			return "", "", "", fmt.Errorf("scope_id %q must equal project_id %q for project scope: %w", scopeID, projectID, ErrUnsupportedScope)
		}
	default:
		if scopeID == "" {
			return "", "", "", fmt.Errorf("scope_id is required for scope_type %q: %w", scopeType, ErrUnsupportedScope)
		}
	}
	return projectID, scopeType, scopeID, nil
}

// normalizeAttentionStateFilter validates and canonicalizes one optional attention-state filter.
func normalizeAttentionStateFilter(raw string) (string, error) {
	state := strings.ToLower(strings.TrimSpace(raw))
	switch state {
	case "", AttentionStateOpen, AttentionStateAcknowledged, AttentionStateResolved:
		return state, nil
	default:
		return "", fmt.Errorf("state %q is unsupported: %w", state, ErrInvalidCaptureStateRequest)
	}
}

// convertCaptureStateSummary maps app.CaptureStateSummary into transport-facing CaptureState.
func convertCaptureStateSummary(summary app.CaptureStateSummary, req CaptureStateRequest, project domain.Project, commentOverview CommentOverview) (CaptureState, error) {
	stateHash, err := computeCaptureSummaryHash(summary)
	if err != nil {
		return CaptureState{}, fmt.Errorf("compute capture summary hash: %w", err)
	}

	scopePath := buildScopePathFromLevel(summary.Level, project.Name)
	attentionOverview := AttentionOverview{
		Available:          true,
		OpenCount:          summary.AttentionOverview.UnresolvedCount,
		RequiresUserAction: summary.AttentionOverview.RequiresUserActionCount,
		Items:              make([]AttentionItem, 0, len(summary.AttentionOverview.Items)),
	}
	for _, item := range summary.AttentionOverview.Items {
		attentionOverview.Items = append(attentionOverview.Items, AttentionItem{
			ID:                 item.ID,
			ProjectID:          summary.Level.ProjectID,
			ScopeType:          string(summary.Level.ScopeType),
			ScopeID:            summary.Level.ScopeID,
			State:              string(item.State),
			Kind:               string(item.Kind),
			Summary:            item.Summary,
			RequiresUserAction: item.RequiresUserAction,
			CreatedAt:          item.CreatedAt.UTC(),
		})
	}

	todoTasks := summary.WorkOverview.ActiveItems - summary.WorkOverview.InProgressItems - summary.WorkOverview.DoneItems - summary.WorkOverview.FailedItems
	if todoTasks < 0 {
		todoTasks = 0
	}
	archivedTasks := summary.WorkOverview.TotalItems - summary.WorkOverview.ActiveItems
	if archivedTasks < 0 {
		archivedTasks = 0
	}
	workOverview := WorkOverview{
		TotalTasks:                   summary.WorkOverview.TotalItems,
		TodoTasks:                    todoTasks,
		InProgressTasks:              summary.WorkOverview.InProgressItems,
		DoneTasks:                    summary.WorkOverview.DoneItems,
		FailedTasks:                  summary.WorkOverview.FailedItems,
		ArchivedTasks:                archivedTasks,
		TasksWithOpenBlockers:        summary.WorkOverview.BlockedItems,
		IncompleteCompletionCriteria: summary.WorkOverview.OpenChildItems,
	}

	return CaptureState{
		CapturedAt:         summary.CapturedAt.UTC(),
		ScopePath:          scopePath,
		StateHash:          stateHash,
		GoalOverview:       GoalOverview{ProjectID: project.ID, ProjectName: project.Name, ProjectDescription: project.Description},
		AttentionOverview:  attentionOverview,
		WorkOverview:       workOverview,
		CommentOverview:    commentOverview,
		WarningsOverview:   buildWarningsOverview(workOverview, attentionOverview),
		ResumeHints:        buildResumeHintsFromFollowUps(summary.FollowUpPointers),
		RequestedView:      req.View,
		RequestedScopeType: req.ScopeType,
	}, nil
}

// buildCommentOverview resolves capture comment counters for one level tuple.
func (a *AppServiceAdapter) buildCommentOverview(ctx context.Context, level domain.LevelTuple) (CommentOverview, error) {
	targetType, ok := commentTargetTypeFromScope(string(level.ScopeType))
	if !ok {
		return CommentOverview{}, nil
	}
	comments, err := a.service.ListCommentsByTarget(ctx, app.ListCommentsByTargetInput{
		ProjectID:  level.ProjectID,
		TargetType: targetType,
		TargetID:   level.ScopeID,
	})
	if err != nil {
		return CommentOverview{}, mapAppError("list comments by target", err)
	}
	return summarizeCommentOverview(comments), nil
}

// summarizeCommentOverview computes comment counters from one deterministic comment set.
func summarizeCommentOverview(comments []domain.Comment) CommentOverview {
	overview := CommentOverview{
		RecentCount: len(comments),
	}
	for _, comment := range comments {
		if isImportantCommentMarkdown(comment.BodyMarkdown) {
			overview.ImportantCount++
		}
	}
	return overview
}

// isImportantCommentMarkdown reports whether markdown text carries high-priority signals.
func isImportantCommentMarkdown(markdown string) bool {
	markdown = strings.ToLower(strings.TrimSpace(markdown))
	if markdown == "" {
		return false
	}
	for _, signal := range []string{"important", "urgent", "blocker", "decision", "requires user action"} {
		if strings.Contains(markdown, signal) {
			return true
		}
	}
	return false
}

// buildScopePathFromLevel maps one app-level tuple into a transport scope path.
func buildScopePathFromLevel(level domain.LevelTuple, projectName string) []ScopeNode {
	scopePath := []ScopeNode{
		{
			ScopeType: ScopeTypeProject,
			ScopeID:   level.ProjectID,
			Name:      strings.TrimSpace(projectName),
		},
	}
	if strings.TrimSpace(scopePath[0].Name) == "" {
		scopePath[0].Name = level.ProjectID
	}
	if string(level.ScopeType) == ScopeTypeProject {
		return scopePath
	}
	scopePath = append(scopePath, ScopeNode{
		ScopeType: string(level.ScopeType),
		ScopeID:   level.ScopeID,
		Name:      level.ScopeID,
	})
	return scopePath
}

// buildResumeHintsFromFollowUps maps app follow-up pointers into transport resume hints.
func buildResumeHintsFromFollowUps(in app.CaptureStateFollowUpPointers) []ResumeHint {
	hints := make([]ResumeHint, 0, 3)
	if pointer := strings.TrimSpace(in.ListAttentionItems); pointer != "" {
		hints = append(hints, ResumeHint{
			Rel:  "till.attention_item",
			Note: pointer,
		})
	}
	if pointer := strings.TrimSpace(in.ListProjectChangeEvents); pointer != "" {
		hints = append(hints, ResumeHint{
			Rel:  "till.project",
			Note: pointer,
		})
	}
	if pointer := strings.TrimSpace(in.ListChildTasks); pointer != "" {
		hints = append(hints, ResumeHint{
			Rel:  "till.action_item",
			Note: pointer,
		})
	}
	if len(hints) == 0 {
		hints = append(hints, ResumeHint{
			Rel:  "till.capture_state",
			Note: "request view=full for expanded, summary-safe context",
		})
	}
	return hints
}

// mapDomainAttentionItems maps domain attention rows into transport DTO rows.
func mapDomainAttentionItems(items []domain.AttentionItem) []AttentionItem {
	out := make([]AttentionItem, 0, len(items))
	for _, item := range items {
		out = append(out, mapDomainAttentionItem(item))
	}
	return out
}

// mapDomainAttentionItem maps one domain attention row into one transport DTO row.
func mapDomainAttentionItem(item domain.AttentionItem) AttentionItem {
	return AttentionItem{
		ID:                 item.ID,
		ProjectID:          item.ProjectID,
		ScopeType:          string(item.ScopeType),
		ScopeID:            item.ScopeID,
		State:              string(domain.NormalizeAttentionState(item.State)),
		Kind:               string(domain.NormalizeAttentionKind(item.Kind)),
		Summary:            item.Summary,
		BodyMarkdown:       item.BodyMarkdown,
		TargetRole:         item.TargetRole,
		RequiresUserAction: item.RequiresUserAction,
		CreatedAt:          item.CreatedAt.UTC(),
		ResolvedAt:         item.ResolvedAt,
	}
}

// computeCaptureSummaryHash computes a deterministic hash from app capture summary data.
func computeCaptureSummaryHash(summary app.CaptureStateSummary) (string, error) {
	attentionOverview := summary.AttentionOverview
	attentionOverview.Items = append([]app.CaptureStateAttentionItem(nil), attentionOverview.Items...)
	slices.SortFunc(attentionOverview.Items, func(a, b app.CaptureStateAttentionItem) int {
		if !a.CreatedAt.Equal(b.CreatedAt) {
			if a.CreatedAt.Before(b.CreatedAt) {
				return -1
			}
			return 1
		}
		return strings.Compare(a.ID, b.ID)
	})
	payload := struct {
		Level             domain.LevelTuple                 `json:"level"`
		GoalOverview      string                            `json:"goal_overview"`
		AttentionOverview app.CaptureStateAttentionOverview `json:"attention_overview"`
		WorkOverview      app.CaptureStateWorkOverview      `json:"work_overview"`
		FollowUpPointers  app.CaptureStateFollowUpPointers  `json:"follow_up_pointers"`
	}{
		Level:             summary.Level,
		GoalOverview:      summary.GoalOverview,
		AttentionOverview: attentionOverview,
		WorkOverview:      summary.WorkOverview,
		FollowUpPointers:  summary.FollowUpPointers,
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal capture summary: %w", err)
	}
	sum := sha256.Sum256(encoded)
	return hex.EncodeToString(sum[:]), nil
}

// mapAppError maps app/domain errors into transport-layer error sentinels.
func mapAppError(operation string, err error) error {
	if err == nil {
		return nil
	}

	switch {
	case errors.Is(err, ErrBootstrapRequired):
		return fmt.Errorf("%s: %w", operation, errors.Join(ErrBootstrapRequired, err))
	case errors.Is(err, domain.ErrBuiltinTemplateBootstrapRequired):
		return fmt.Errorf("%s: %w", operation, errors.Join(ErrBuiltinTemplateBootstrapRequired, err))
	case errors.Is(err, app.ErrNotFound):
		return fmt.Errorf("%s: %w", operation, errors.Join(ErrNotFound, err))
	case errors.Is(err, domain.ErrMutationLeaseRequired),
		errors.Is(err, domain.ErrMutationLeaseInvalid),
		errors.Is(err, domain.ErrMutationLeaseExpired),
		errors.Is(err, domain.ErrMutationLeaseRevoked),
		errors.Is(err, domain.ErrOrchestratorOverlap),
		errors.Is(err, domain.ErrOverrideTokenRequired),
		errors.Is(err, domain.ErrOverrideTokenInvalid),
		errors.Is(err, domain.ErrTransitionBlocked):
		return fmt.Errorf("%s: %w", operation, errors.Join(ErrGuardrailViolation, err))
	case errors.Is(err, domain.ErrInvalidID),
		errors.Is(err, domain.ErrInvalidScopeType),
		errors.Is(err, domain.ErrInvalidScopeID),
		errors.Is(err, domain.ErrInvalidAuthRequestPath),
		errors.Is(err, domain.ErrInvalidAuthRequestRole),
		errors.Is(err, domain.ErrInvalidAuthRequestState),
		errors.Is(err, domain.ErrInvalidAuthRequestTTL),
		errors.Is(err, domain.ErrInvalidAuthContinuation),
		errors.Is(err, domain.ErrAuthRequestClaimMismatch),
		errors.Is(err, domain.ErrAuthRequestNotPending),
		errors.Is(err, domain.ErrAuthRequestExpired),
		errors.Is(err, domain.ErrInvalidSummary),
		errors.Is(err, domain.ErrInvalidBodyMarkdown),
		errors.Is(err, domain.ErrInvalidAttentionState),
		errors.Is(err, domain.ErrInvalidAttentionKind),
		errors.Is(err, domain.ErrInvalidActorType),
		errors.Is(err, domain.ErrInvalidName),
		errors.Is(err, domain.ErrInvalidTitle),
		errors.Is(err, domain.ErrInvalidParentID),
		errors.Is(err, domain.ErrInvalidColumnID),
		errors.Is(err, domain.ErrInvalidPriority),
		errors.Is(err, domain.ErrInvalidLifecycleState),
		errors.Is(err, domain.ErrInvalidKind),
		errors.Is(err, domain.ErrInvalidKindID),
		errors.Is(err, domain.ErrInvalidKindAppliesTo),
		errors.Is(err, domain.ErrInvalidKindTemplate),
		errors.Is(err, domain.ErrInvalidTemplateLibrary),
		errors.Is(err, domain.ErrInvalidTemplateLibraryScope),
		errors.Is(err, domain.ErrInvalidTemplateStatus),
		errors.Is(err, domain.ErrInvalidTemplateActorKind),
		errors.Is(err, domain.ErrInvalidTemplateBinding),
		errors.Is(err, domain.ErrInvalidKindPayload),
		errors.Is(err, domain.ErrInvalidKindPayloadSchema),
		errors.Is(err, domain.ErrKindNotAllowed),
		errors.Is(err, app.ErrInvalidDeleteMode):
		return fmt.Errorf("%s: %w", operation, errors.Join(ErrInvalidCaptureStateRequest, err))
	case errors.Is(err, domain.ErrKindNotFound),
		errors.Is(err, domain.ErrTemplateLibraryNotFound):
		return fmt.Errorf("%s: %w", operation, errors.Join(ErrNotFound, err))
	default:
		return fmt.Errorf("%s: %w", operation, err)
	}
}
