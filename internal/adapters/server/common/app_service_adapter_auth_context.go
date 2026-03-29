package common

import (
	"context"
	"strings"

	"github.com/hylla/tillsyn/internal/app"
	"github.com/hylla/tillsyn/internal/domain"
)

// normalizeMutationAuthorizationRequest canonicalizes auth input and enriches project-rooted scope context.
func (a *AppServiceAdapter) normalizeMutationAuthorizationRequest(ctx context.Context, in MutationAuthorizationRequest) (MutationAuthorizationRequest, error) {
	in.SessionID = strings.TrimSpace(in.SessionID)
	in.SessionSecret = strings.TrimSpace(in.SessionSecret)
	in.Action = strings.TrimSpace(in.Action)
	in.Namespace = strings.TrimSpace(in.Namespace)
	in.ResourceType = strings.TrimSpace(in.ResourceType)
	in.ResourceID = strings.TrimSpace(in.ResourceID)
	in.Context = cloneStringMap(in.Context)
	if in.Context == nil {
		in.Context = map[string]string{}
	}
	if in.Namespace != "" {
		in.Context["namespace"] = in.Namespace
	}
	if a == nil || a.service == nil || a.auth == nil {
		return in, nil
	}
	if _, err := a.auth.ValidateSession(ctx, in.SessionID, in.SessionSecret); err != nil {
		return in, nil
	}
	if err := a.enrichMutationAuthorizationContext(ctx, &in); err != nil {
		return MutationAuthorizationRequest{}, mapAppError("authorize mutation", err)
	}
	return in, nil
}

// enrichMutationAuthorizationContext derives the narrowest project-rooted auth path for one mutation.
func (a *AppServiceAdapter) enrichMutationAuthorizationContext(ctx context.Context, in *MutationAuthorizationRequest) error {
	if in == nil {
		return nil
	}
	contextValues := in.Context
	switch in.Action {
	case "create_task":
		return a.populateCreateTaskAuthContext(ctx, contextValues)
	case "create_comment":
		return a.populateCommentAuthContext(ctx, contextValues, in.ResourceID)
	case "create_handoff":
		return a.populateLevelAuthContext(ctx, contextValues,
			contextValues["project_id"],
			contextValues["scope_type"],
			firstNonEmptyTrimmed(contextValues["scope_id"], in.ResourceID),
		)
	case "raise_attention_item":
		return a.populateLevelAuthContext(ctx, contextValues,
			contextValues["project_id"],
			contextValues["scope_type"],
			firstNonEmptyTrimmed(contextValues["scope_id"], in.ResourceID),
		)
	case "issue_capability_lease", "revoke_all_capability_leases":
		return a.populateCapabilityScopeAuthContext(ctx, contextValues,
			contextValues["project_id"],
			contextValues["scope_type"],
			firstNonEmptyTrimmed(contextValues["scope_id"], in.ResourceID),
		)
	}

	switch in.ResourceType {
	case "project":
		populateProjectAuthContext(contextValues, firstNonEmptyTrimmed(contextValues["project_id"], in.ResourceID))
		return nil
	case "task":
		return a.populateTaskAuthContext(ctx, contextValues, firstNonEmptyTrimmed(contextValues["task_id"], in.ResourceID))
	case "handoff":
		return a.populateHandoffAuthContext(ctx, contextValues, firstNonEmptyTrimmed(contextValues["handoff_id"], in.ResourceID))
	case "attention_item":
		return a.populateAttentionAuthContext(ctx, contextValues, firstNonEmptyTrimmed(contextValues["attention_id"], in.ResourceID))
	case "capability_lease":
		return a.populateLeaseAuthContext(ctx, contextValues, firstNonEmptyTrimmed(contextValues["agent_instance_id"], in.ResourceID))
	default:
		if projectID := firstNonEmptyTrimmed(contextValues["project_id"], projectIDFromNamespace(in.Namespace)); projectID != "" {
			populateProjectAuthContext(contextValues, projectID)
		}
		return nil
	}
}

// populateCreateTaskAuthContext derives the auth path for create-task mutations.
func (a *AppServiceAdapter) populateCreateTaskAuthContext(ctx context.Context, contextValues map[string]string) error {
	parentID := strings.TrimSpace(contextValues["parent_id"])
	if parentID == "" {
		populateProjectAuthContext(contextValues, contextValues["project_id"])
		return nil
	}
	return a.populateTaskAuthContext(ctx, contextValues, parentID)
}

// populateCommentAuthContext derives the auth path for comment creation.
func (a *AppServiceAdapter) populateCommentAuthContext(ctx context.Context, contextValues map[string]string, targetID string) error {
	projectID := strings.TrimSpace(contextValues["project_id"])
	switch strings.ToLower(strings.TrimSpace(contextValues["target_type"])) {
	case "", "project":
		populateProjectAuthContext(contextValues, projectID)
		return nil
	default:
		return a.populateTaskAuthContext(ctx, contextValues, targetID)
	}
}

// populateTaskAuthContext derives auth scope from one existing task row.
func (a *AppServiceAdapter) populateTaskAuthContext(ctx context.Context, contextValues map[string]string, taskID string) error {
	task, err := a.service.GetTask(ctx, taskID)
	if err != nil {
		return err
	}
	scopeType := domain.ScopeLevelFromKindAppliesTo(task.Scope)
	if scopeType == "" {
		return domain.ErrInvalidScopeType
	}
	return a.populateLevelAuthContext(ctx, contextValues, task.ProjectID, string(scopeType), task.ID)
}

// populateHandoffAuthContext derives auth scope from one existing handoff row.
func (a *AppServiceAdapter) populateHandoffAuthContext(ctx context.Context, contextValues map[string]string, handoffID string) error {
	handoff, err := a.service.GetHandoff(ctx, handoffID)
	if err != nil {
		return err
	}
	return a.populateLevelAuthContext(ctx, contextValues, handoff.ProjectID, string(handoff.ScopeType), handoff.ScopeID)
}

// populateAttentionAuthContext derives auth scope from one existing attention item.
func (a *AppServiceAdapter) populateAttentionAuthContext(ctx context.Context, contextValues map[string]string, attentionID string) error {
	item, err := a.service.GetAttentionItem(ctx, attentionID)
	if err != nil {
		return err
	}
	return a.populateLevelAuthContext(ctx, contextValues, item.ProjectID, string(item.ScopeType), item.ScopeID)
}

// populateLeaseAuthContext derives auth scope from one existing capability lease.
func (a *AppServiceAdapter) populateLeaseAuthContext(ctx context.Context, contextValues map[string]string, instanceID string) error {
	lease, err := a.service.GetCapabilityLease(ctx, instanceID)
	if err != nil {
		return err
	}
	return a.populateCapabilityScopeAuthContext(ctx, contextValues, lease.ProjectID, string(lease.ScopeType), lease.ScopeID)
}

// populateCapabilityScopeAuthContext converts capability scope values into project-rooted auth context.
func (a *AppServiceAdapter) populateCapabilityScopeAuthContext(ctx context.Context, contextValues map[string]string, projectID, capabilityScopeType, scopeID string) error {
	scopeLevel := domain.ScopeLevelFromCapabilityScopeType(domain.CapabilityScopeType(capabilityScopeType))
	if scopeLevel == "" {
		scopeLevel = domain.ScopeLevelProject
	}
	return a.populateLevelAuthContext(ctx, contextValues, projectID, string(scopeLevel), scopeID)
}

// populateLevelAuthContext resolves one level tuple into the auth-path context expected by autent.
func (a *AppServiceAdapter) populateLevelAuthContext(ctx context.Context, contextValues map[string]string, projectID, scopeType, scopeID string) error {
	projectHint := firstNonEmptyTrimmed(contextValues["project_id"], projectIDFromNamespace(contextValues["namespace"]))
	projectID = firstNonEmptyTrimmed(projectID, projectHint)
	scopeType = strings.TrimSpace(scopeType)
	scopeID = strings.TrimSpace(scopeID)
	if projectID == "" {
		return nil
	}
	if projectHint != "" && projectHint != projectID {
		return app.ErrNotFound
	}
	if scopeType == "" {
		scopeType = ScopeTypeProject
	}
	if scopeType == ScopeTypeProject && scopeID == "" {
		scopeID = projectID
	}
	resolved, err := a.service.ResolveAuthScopeContext(ctx, domain.LevelTupleInput{
		ProjectID: projectID,
		ScopeType: domain.ScopeLevel(scopeType),
		ScopeID:   scopeID,
	})
	if err != nil {
		return err
	}
	applyResolvedAuthScopeContext(contextValues, resolved)
	return nil
}

// applyResolvedAuthScopeContext writes one resolved auth scope into an auth context map.
func applyResolvedAuthScopeContext(contextValues map[string]string, resolved app.AuthScopeContext) {
	if contextValues == nil {
		return
	}
	populateProjectAuthContext(contextValues, resolved.ProjectID)
	contextValues["scope_type"] = string(resolved.ScopeType)
	contextValues["scope_id"] = strings.TrimSpace(resolved.ScopeID)
	if strings.TrimSpace(resolved.BranchID) != "" {
		contextValues["branch_id"] = strings.TrimSpace(resolved.BranchID)
	} else {
		delete(contextValues, "branch_id")
	}
	if len(resolved.PhaseIDs) > 0 {
		contextValues["phase_path"] = strings.Join(resolved.PhaseIDs, "/")
		contextValues["phase_id"] = resolved.PhaseIDs[len(resolved.PhaseIDs)-1]
	} else {
		delete(contextValues, "phase_path")
		delete(contextValues, "phase_id")
	}
}

// populateProjectAuthContext writes project-rooted auth context when only project scope is proven.
func populateProjectAuthContext(contextValues map[string]string, projectID string) {
	if contextValues == nil {
		return
	}
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		return
	}
	contextValues["project_id"] = projectID
	contextValues["scope_type"] = ScopeTypeProject
	contextValues["scope_id"] = projectID
	delete(contextValues, "branch_id")
	delete(contextValues, "phase_id")
	delete(contextValues, "phase_path")
}

// projectIDFromNamespace extracts a project id from one `project:<id>` namespace value.
func projectIDFromNamespace(namespace string) string {
	namespace = strings.TrimSpace(namespace)
	if namespace == "" {
		return ""
	}
	if strings.HasPrefix(namespace, "project:") {
		return strings.TrimSpace(strings.TrimPrefix(namespace, "project:"))
	}
	return ""
}

// firstNonEmptyTrimmed returns the first non-empty trimmed string from left to right.
func firstNonEmptyTrimmed(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
