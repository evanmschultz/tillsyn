package domain

import "strings"

// AuthenticatedCaller captures the normalized caller identity resolved before app mutations run.
// Future autent-backed session validation should populate this shape so transport code does not
// need to push attribution through ad hoc request-specific fields.
type AuthenticatedCaller struct {
	PrincipalID   string
	PrincipalName string
	// PrincipalType carries the actor-class axis (user|agent|system) used by
	// the existing mutation guard layer. autent's closed principal-type enum
	// {user, agent, service} maps onto this via principalTypeToActorType in
	// the autentauth adapter.
	PrincipalType ActorType
	// AuthRequestPrincipalType carries the auth-request principal-class axis
	// (user|agent|service|steward) sourced from the originating auth_request
	// row + persisted on the issued AuthSession. Distinct from PrincipalType
	// (the actor-class axis) so the STEWARD owner-state-lock can key on the
	// "steward" value without collapsing it into "agent". Drop 3 droplet 3.19
	// added this field to enforce the gate that drop-orchs cannot move
	// STEWARD-owned action items through state.
	AuthRequestPrincipalType string
	SessionID                string
}

// NormalizeAuthenticatedCaller trims and canonicalizes one authenticated caller value.
func NormalizeAuthenticatedCaller(caller AuthenticatedCaller) AuthenticatedCaller {
	caller.PrincipalID = strings.TrimSpace(caller.PrincipalID)
	caller.PrincipalName = strings.TrimSpace(caller.PrincipalName)
	caller.SessionID = strings.TrimSpace(caller.SessionID)
	caller.AuthRequestPrincipalType = strings.TrimSpace(strings.ToLower(caller.AuthRequestPrincipalType))
	caller.PrincipalType = normalizeActorTypeValue(caller.PrincipalType)
	switch caller.PrincipalType {
	case ActorTypeUser, ActorTypeAgent, ActorTypeSystem:
	case "":
		if caller.PrincipalID != "" || caller.PrincipalName != "" || caller.SessionID != "" {
			caller.PrincipalType = ActorTypeUser
		}
	default:
		caller.PrincipalType = ActorTypeUser
	}
	if caller.PrincipalName == "" && caller.PrincipalID != "" {
		caller.PrincipalName = caller.PrincipalID
	}
	return caller
}

// IsZero reports whether the authenticated caller carries no caller/session identity.
func (caller AuthenticatedCaller) IsZero() bool {
	caller = NormalizeAuthenticatedCaller(caller)
	return caller.PrincipalID == "" && caller.PrincipalName == "" && caller.SessionID == ""
}
