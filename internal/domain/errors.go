package domain

import "errors"

// ErrInvalidID and related errors describe validation and runtime failures.
var (
	ErrInvalidID                   = errors.New("invalid id")
	ErrInvalidName                 = errors.New("invalid name")
	ErrInvalidTitle                = errors.New("invalid title")
	ErrInvalidSummary              = errors.New("invalid summary")
	ErrInvalidBodyMarkdown         = errors.New("invalid body markdown")
	ErrInvalidPriority             = errors.New("invalid priority")
	ErrInvalidPosition             = errors.New("invalid position")
	ErrInvalidColumnID             = errors.New("invalid column id")
	ErrInvalidParentID             = errors.New("invalid parent id")
	ErrInvalidScopeType            = errors.New("invalid scope type")
	ErrInvalidScopeID              = errors.New("invalid scope id")
	ErrInvalidTargetID             = errors.New("invalid target id")
	ErrInvalidTargetType           = errors.New("invalid target type")
	ErrInvalidKind                 = errors.New("invalid kind")
	ErrInvalidKindID               = errors.New("invalid kind id")
	ErrInvalidKindAppliesTo        = errors.New("invalid kind applies_to")
	ErrKindNotAllowed              = errors.New("kind is not allowed for this project")
	ErrKindNotFound                = errors.New("kind definition not found")
	ErrInvalidKindPayload          = errors.New("invalid kind payload")
	ErrInvalidKindPayloadSchema    = errors.New("invalid kind payload schema")
	ErrInvalidRole                 = errors.New("invalid role")
	ErrInvalidStructuralType       = errors.New("invalid structural type")
	ErrInvalidDropNumber           = errors.New("invalid drop number")
	ErrInvalidPaths                = errors.New("invalid paths")
	ErrInvalidPackages             = errors.New("invalid packages")
	ErrInvalidFiles                = errors.New("invalid files")
	ErrInvalidLanguage             = errors.New("invalid language")
	ErrInvalidRepoPath             = errors.New("invalid repo path")
	ErrInvalidLifecycleState       = errors.New("invalid lifecycle state")
	ErrInvalidActorType            = errors.New("invalid actor type")
	ErrInvalidAttentionState       = errors.New("invalid attention state")
	ErrInvalidAttentionKind        = errors.New("invalid attention kind")
	ErrInvalidHandoffStatus        = errors.New("invalid handoff status")
	ErrInvalidHandoffTransition    = errors.New("invalid handoff transition")
	ErrInvalidAuthRequestPath      = errors.New("invalid auth request path")
	ErrInvalidAuthRequestState     = errors.New("invalid auth request state")
	ErrInvalidAuthRequestTTL       = errors.New("invalid auth request ttl")
	ErrInvalidAuthRequestRole      = errors.New("invalid auth request role")
	ErrInvalidAuthContinuation     = errors.New("invalid auth request continuation")
	ErrAuthRequestClaimMismatch    = errors.New("auth request claim mismatch")
	ErrInvalidCapabilityRole       = errors.New("invalid capability role")
	ErrInvalidCapabilityAction     = errors.New("invalid capability action")
	ErrInvalidCapabilityDelegation = errors.New("invalid capability delegation")
	ErrInvalidCapabilityScope      = errors.New("invalid capability scope")
	ErrInvalidCapabilityToken      = errors.New("invalid capability token")
	ErrInvalidCapabilityExpiry     = errors.New("invalid capability expiry")
	ErrMutationLeaseRequired       = errors.New("mutation lease is required")
	ErrMutationLeaseInvalid        = errors.New("mutation lease is invalid")
	ErrMutationLeaseExpired        = errors.New("mutation lease is expired")
	ErrMutationLeaseRevoked        = errors.New("mutation lease is revoked")
	ErrOrchestratorOverlap         = errors.New("overlapping orchestrator lease blocked")
	ErrOverrideTokenRequired       = errors.New("override token is required for overlapping orchestrator lease")
	ErrOverrideTokenInvalid        = errors.New("override token is invalid")
	ErrTransitionBlocked           = errors.New("transition blocked by completion contract")
	ErrAuthRequestNotPending       = errors.New("auth request is not pending")
	ErrAuthRequestExpired          = errors.New("auth request is expired")
	// ErrAuthorizationDenied reports that a valid caller was denied by auth
	// policy. Drop 4a Wave 3 W3.1 lifted this from the
	// `internal/adapters/server/common` package into `domain` so the app
	// layer's orch-self-approval gate can return it without crossing into
	// the adapter import boundary. The `common.ErrAuthorizationDenied`
	// alias is preserved for source compatibility — both values are equal,
	// so existing `errors.Is(err, common.ErrAuthorizationDenied)` checks
	// still work.
	ErrAuthorizationDenied = errors.New("authorization denied")
	// ErrOrchSelfApprovalDisabled reports that the request's project has
	// opted out of the orch-self-approval cascade via the project-metadata
	// toggle Metadata.OrchSelfApprovalEnabled = *false (Drop 4a Wave 3 W3.2).
	// The check fires BEFORE the role / path / cross-orch gate so the
	// rejection is total — including the STEWARD cross-subtree exception.
	// Distinct from ErrAuthorizationDenied to keep observability sharp:
	// callers branching via errors.Is can surface the toggle status without
	// confusing it with the role-gate's denials.
	ErrOrchSelfApprovalDisabled = errors.New("orch self-approval disabled by project metadata")
	// ErrInvalidPermissionGrantRule reports a missing or empty permission
	// rule on PermissionGrant creation (Drop 4c F.7.17.7). Domain layer
	// rejects the empty string only; rule shape ("Bash(npm run *)" etc.)
	// is the caller's responsibility.
	ErrInvalidPermissionGrantRule = errors.New("invalid permission grant rule")
	// ErrInvalidPermissionGrantCLIKind reports a missing CLI kind on
	// PermissionGrant creation (Drop 4c F.7.17.7). Closed-enum membership
	// for the CLI vocabulary is enforced at the templates / dispatcher
	// layer; the domain only refuses the empty string so the UNIQUE
	// composite never stores a blank cli_kind.
	ErrInvalidPermissionGrantCLIKind = errors.New("invalid permission grant cli kind")
	// ErrInvalidPermissionGrantGrantedBy reports a missing principal on
	// PermissionGrant creation (Drop 4c F.7.17.7).
	ErrInvalidPermissionGrantGrantedBy = errors.New("invalid permission grant granted_by")
)
