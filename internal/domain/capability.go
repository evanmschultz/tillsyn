package domain

import (
	"slices"
	"strings"
	"time"
)

// CapabilityAction identifies one explicit policy action class for agent roles.
type CapabilityAction string

// Capability action values.
const (
	CapabilityActionRead                    CapabilityAction = "read"
	CapabilityActionComment                 CapabilityAction = "comment"
	CapabilityActionCreateChild             CapabilityAction = "create-child"
	CapabilityActionEditNode                CapabilityAction = "edit-node"
	CapabilityActionRequestAuth             CapabilityAction = "request-auth"
	CapabilityActionApproveAuthWithinBounds CapabilityAction = "approve-auth-within-bounds"
	CapabilityActionMarkInProgress          CapabilityAction = "mark-in-progress"
	CapabilityActionMarkComplete            CapabilityAction = "mark-complete"
	CapabilityActionMarkFailed              CapabilityAction = "mark-failed"
	CapabilityActionReopen                  CapabilityAction = "reopen"
	CapabilityActionAttachEvidence          CapabilityAction = "attach-evidence"
	CapabilityActionSignoff                 CapabilityAction = "signoff"
	CapabilityActionResolveAttention        CapabilityAction = "resolve-attention"
	CapabilityActionArchiveOrCleanup        CapabilityAction = "archive-or-cleanup"
)

// validCapabilityActions stores supported policy-action values.
var validCapabilityActions = []CapabilityAction{
	CapabilityActionRead,
	CapabilityActionComment,
	CapabilityActionCreateChild,
	CapabilityActionEditNode,
	CapabilityActionRequestAuth,
	CapabilityActionApproveAuthWithinBounds,
	CapabilityActionMarkInProgress,
	CapabilityActionMarkComplete,
	CapabilityActionMarkFailed,
	CapabilityActionReopen,
	CapabilityActionAttachEvidence,
	CapabilityActionSignoff,
	CapabilityActionResolveAttention,
	CapabilityActionArchiveOrCleanup,
}

// CapabilityRole identifies the role of a capability lease owner.
type CapabilityRole string

// Capability role values.
const (
	CapabilityRoleOrchestrator CapabilityRole = "orchestrator"
	CapabilityRoleBuilder      CapabilityRole = "builder"
	CapabilityRoleQA           CapabilityRole = "qa"
	CapabilityRoleResearch     CapabilityRole = "research"
	CapabilityRoleSystem       CapabilityRole = "system"

	// CapabilityRoleWorker preserves the legacy worker token as an alias for builder.
	CapabilityRoleWorker CapabilityRole = CapabilityRoleBuilder
)

// CapabilityScopeType identifies the scope a capability lease is bound to.
type CapabilityScopeType string

// Capability scope values.
const (
	CapabilityScopeProject    CapabilityScopeType = "project"
	CapabilityScopeBranch     CapabilityScopeType = "branch"
	CapabilityScopePhase      CapabilityScopeType = "phase"
	CapabilityScopeActionItem CapabilityScopeType = "actionItem"
	CapabilityScopeSubtask    CapabilityScopeType = "subtask"
)

// validCapabilityRoles stores supported capability roles.
var validCapabilityRoles = []CapabilityRole{
	CapabilityRoleOrchestrator,
	CapabilityRoleBuilder,
	CapabilityRoleQA,
	CapabilityRoleResearch,
	CapabilityRoleSystem,
}

// validCapabilityScopes stores supported capability scope values.
var validCapabilityScopes = []CapabilityScopeType{
	CapabilityScopeProject,
	CapabilityScopeBranch,
	CapabilityScopePhase,
	CapabilityScopeActionItem,
	CapabilityScopeSubtask,
}

// CapabilityLease stores one scoped, revocable capability token lease.
type CapabilityLease struct {
	InstanceID                string
	LeaseToken                string
	AgentName                 string
	ProjectID                 string
	ScopeType                 CapabilityScopeType
	ScopeID                   string
	Role                      CapabilityRole
	ParentInstanceID          string
	AllowEqualScopeDelegation bool
	IssuedAt                  time.Time
	ExpiresAt                 time.Time
	HeartbeatAt               time.Time
	RevokedAt                 *time.Time
	RevokedReason             string
}

// CapabilityLeaseInput holds values used to issue a new lease.
type CapabilityLeaseInput struct {
	InstanceID                string
	LeaseToken                string
	AgentName                 string
	ProjectID                 string
	ScopeType                 CapabilityScopeType
	ScopeID                   string
	Role                      CapabilityRole
	ParentInstanceID          string
	AllowEqualScopeDelegation bool
	ExpiresAt                 time.Time
}

// NewCapabilityLease normalizes and validates one lease issuance request.
func NewCapabilityLease(in CapabilityLeaseInput, now time.Time) (CapabilityLease, error) {
	in.InstanceID = strings.TrimSpace(in.InstanceID)
	in.LeaseToken = strings.TrimSpace(in.LeaseToken)
	in.AgentName = strings.TrimSpace(in.AgentName)
	in.ProjectID = strings.TrimSpace(in.ProjectID)
	in.ScopeType = NormalizeCapabilityScopeType(in.ScopeType)
	in.ScopeID = strings.TrimSpace(in.ScopeID)
	in.Role = NormalizeCapabilityRole(in.Role)
	in.ParentInstanceID = strings.TrimSpace(in.ParentInstanceID)

	if in.InstanceID == "" {
		return CapabilityLease{}, ErrInvalidID
	}
	if in.LeaseToken == "" {
		return CapabilityLease{}, ErrInvalidCapabilityToken
	}
	if in.AgentName == "" {
		return CapabilityLease{}, ErrInvalidName
	}
	if in.ProjectID == "" {
		return CapabilityLease{}, ErrInvalidID
	}
	if !IsValidCapabilityScopeType(in.ScopeType) {
		return CapabilityLease{}, ErrInvalidCapabilityScope
	}
	if in.ScopeType == CapabilityScopeProject && in.ScopeID == "" {
		in.ScopeID = in.ProjectID
	}
	if in.ScopeType != CapabilityScopeProject && in.ScopeID == "" {
		return CapabilityLease{}, ErrInvalidCapabilityScope
	}
	if !IsValidCapabilityRole(in.Role) {
		return CapabilityLease{}, ErrInvalidCapabilityRole
	}
	if in.ExpiresAt.IsZero() || !in.ExpiresAt.After(now.UTC()) {
		return CapabilityLease{}, ErrInvalidCapabilityExpiry
	}

	ts := now.UTC()
	return CapabilityLease{
		InstanceID:                in.InstanceID,
		LeaseToken:                in.LeaseToken,
		AgentName:                 in.AgentName,
		ProjectID:                 in.ProjectID,
		ScopeType:                 in.ScopeType,
		ScopeID:                   in.ScopeID,
		Role:                      in.Role,
		ParentInstanceID:          in.ParentInstanceID,
		AllowEqualScopeDelegation: in.AllowEqualScopeDelegation,
		IssuedAt:                  ts,
		ExpiresAt:                 in.ExpiresAt.UTC(),
		HeartbeatAt:               ts,
	}, nil
}

// NormalizeCapabilityRole canonicalizes role values.
func NormalizeCapabilityRole(role CapabilityRole) CapabilityRole {
	switch strings.TrimSpace(strings.ToLower(string(role))) {
	case string(CapabilityRoleOrchestrator):
		return CapabilityRoleOrchestrator
	case string(CapabilityRoleBuilder), "worker", "subagent":
		return CapabilityRoleBuilder
	case string(CapabilityRoleQA):
		return CapabilityRoleQA
	case string(CapabilityRoleResearch):
		return CapabilityRoleResearch
	case string(CapabilityRoleSystem):
		return CapabilityRoleSystem
	default:
		return CapabilityRole(strings.TrimSpace(strings.ToLower(string(role))))
	}
}

// NormalizeCapabilityScopeType canonicalizes scope values. Inputs are matched
// case-insensitively against the supported set and returned in their
// canonical camelCase form (e.g. "actionItem"); unknown values are returned
// lowercased so callers can still detect invalid inputs.
func NormalizeCapabilityScopeType(scope CapabilityScopeType) CapabilityScopeType {
	lowered := strings.TrimSpace(strings.ToLower(string(scope)))
	if lowered == "" {
		return ""
	}
	for _, candidate := range validCapabilityScopes {
		if strings.ToLower(string(candidate)) == lowered {
			return candidate
		}
	}
	return CapabilityScopeType(lowered)
}

// IsValidCapabilityRole reports whether a role value is supported.
func IsValidCapabilityRole(role CapabilityRole) bool {
	role = NormalizeCapabilityRole(role)
	return slices.Contains(validCapabilityRoles, role)
}

// NormalizeCapabilityAction canonicalizes one policy action value.
func NormalizeCapabilityAction(action CapabilityAction) CapabilityAction {
	return CapabilityAction(strings.TrimSpace(strings.ToLower(string(action))))
}

// IsValidCapabilityAction reports whether a policy action value is supported.
func IsValidCapabilityAction(action CapabilityAction) bool {
	return slices.Contains(validCapabilityActions, NormalizeCapabilityAction(action))
}

// DefaultCapabilityActions returns the default policy-action set for one role.
func DefaultCapabilityActions(role CapabilityRole) []CapabilityAction {
	switch NormalizeCapabilityRole(role) {
	case CapabilityRoleOrchestrator:
		return []CapabilityAction{
			CapabilityActionRead,
			CapabilityActionComment,
			CapabilityActionCreateChild,
			CapabilityActionEditNode,
			CapabilityActionRequestAuth,
			CapabilityActionApproveAuthWithinBounds,
			CapabilityActionMarkInProgress,
			CapabilityActionMarkComplete,
			CapabilityActionMarkFailed,
			CapabilityActionResolveAttention,
			CapabilityActionArchiveOrCleanup,
		}
	case CapabilityRoleBuilder:
		return []CapabilityAction{
			CapabilityActionRead,
			CapabilityActionComment,
			CapabilityActionCreateChild,
			CapabilityActionEditNode,
			CapabilityActionRequestAuth,
			CapabilityActionMarkInProgress,
			CapabilityActionMarkComplete,
			CapabilityActionMarkFailed,
			CapabilityActionAttachEvidence,
		}
	case CapabilityRoleQA:
		return []CapabilityAction{
			CapabilityActionRead,
			CapabilityActionComment,
			CapabilityActionEditNode,
			CapabilityActionRequestAuth,
			CapabilityActionMarkInProgress,
			CapabilityActionMarkComplete,
			CapabilityActionMarkFailed,
			CapabilityActionReopen,
			CapabilityActionAttachEvidence,
			CapabilityActionSignoff,
			CapabilityActionResolveAttention,
		}
	case CapabilityRoleResearch:
		return []CapabilityAction{
			CapabilityActionRead,
			CapabilityActionComment,
			CapabilityActionCreateChild,
			CapabilityActionEditNode,
			CapabilityActionRequestAuth,
			CapabilityActionMarkInProgress,
			CapabilityActionMarkComplete,
			CapabilityActionMarkFailed,
			CapabilityActionAttachEvidence,
		}
	case CapabilityRoleSystem:
		return append([]CapabilityAction(nil), validCapabilityActions...)
	default:
		return nil
	}
}

// CanPerform reports whether one role includes the requested default policy action.
func (r CapabilityRole) CanPerform(action CapabilityAction) bool {
	action = NormalizeCapabilityAction(action)
	if !IsValidCapabilityAction(action) {
		return false
	}
	return slices.Contains(DefaultCapabilityActions(r), action)
}

// CanTargetBroaderThanProject reports whether a role may target broader-than-project paths.
func (r CapabilityRole) CanTargetBroaderThanProject() bool {
	switch NormalizeCapabilityRole(r) {
	case CapabilityRoleOrchestrator, CapabilityRoleSystem:
		return true
	default:
		return false
	}
}

// CanDelegateTo reports whether one role may mint a child role.
func (r CapabilityRole) CanDelegateTo(child CapabilityRole) bool {
	switch NormalizeCapabilityRole(r) {
	case CapabilityRoleOrchestrator:
		switch NormalizeCapabilityRole(child) {
		case CapabilityRoleBuilder, CapabilityRoleQA, CapabilityRoleResearch:
			return true
		default:
			return false
		}
	case CapabilityRoleSystem:
		return IsValidCapabilityRole(child)
	default:
		return false
	}
}

// IsInternalOnly reports whether a role is reserved for internal-only flows.
func (r CapabilityRole) IsInternalOnly() bool {
	return NormalizeCapabilityRole(r) == CapabilityRoleSystem
}

// IsValidCapabilityScopeType reports whether a scope value is supported.
func IsValidCapabilityScopeType(scope CapabilityScopeType) bool {
	scope = NormalizeCapabilityScopeType(scope)
	return slices.Contains(validCapabilityScopes, scope)
}

// IsExpired reports whether the lease expired at the provided time.
func (l CapabilityLease) IsExpired(now time.Time) bool {
	return !now.UTC().Before(l.ExpiresAt.UTC())
}

// IsRevoked reports whether the lease was revoked.
func (l CapabilityLease) IsRevoked() bool {
	return l.RevokedAt != nil
}

// IsActive reports whether a lease is currently valid for mutation use.
func (l CapabilityLease) IsActive(now time.Time) bool {
	if l.IsRevoked() {
		return false
	}
	return !l.IsExpired(now)
}

// MatchesScope reports whether the lease can operate on a requested scope.
func (l CapabilityLease) MatchesScope(scopeType CapabilityScopeType, scopeID string) bool {
	scopeType = NormalizeCapabilityScopeType(scopeType)
	scopeID = strings.TrimSpace(scopeID)
	if l.ScopeType == CapabilityScopeProject {
		return true
	}
	if l.ScopeType != scopeType {
		return false
	}
	if strings.TrimSpace(l.ScopeID) == "" {
		return true
	}
	return l.ScopeID == scopeID
}

// MatchesIdentity reports whether the lease matches a request identity tuple.
func (l CapabilityLease) MatchesIdentity(agentName, leaseToken string) bool {
	return strings.TrimSpace(l.AgentName) == strings.TrimSpace(agentName) &&
		strings.TrimSpace(l.LeaseToken) == strings.TrimSpace(leaseToken)
}

// Heartbeat updates the lease heartbeat timestamp when active.
func (l *CapabilityLease) Heartbeat(now time.Time) {
	if l == nil {
		return
	}
	l.HeartbeatAt = now.UTC()
}

// Renew extends the lease expiry after validation.
func (l *CapabilityLease) Renew(expiresAt, now time.Time) error {
	if l == nil {
		return ErrInvalidID
	}
	if l.IsRevoked() {
		return ErrMutationLeaseRevoked
	}
	expiresAt = expiresAt.UTC()
	if !expiresAt.After(now.UTC()) {
		return ErrInvalidCapabilityExpiry
	}
	l.ExpiresAt = expiresAt
	l.HeartbeatAt = now.UTC()
	return nil
}

// Revoke marks a lease as revoked and captures the revocation reason.
func (l *CapabilityLease) Revoke(reason string, now time.Time) {
	if l == nil {
		return
	}
	reason = strings.TrimSpace(reason)
	if reason == "" {
		reason = "revoked"
	}
	ts := now.UTC()
	l.RevokedAt = &ts
	l.RevokedReason = reason
}
