package domain

import "strings"

// AuthenticatedCaller captures the normalized caller identity resolved before app mutations run.
// Future autent-backed session validation should populate this shape so transport code does not
// need to push attribution through ad hoc request-specific fields.
type AuthenticatedCaller struct {
	PrincipalID   string
	PrincipalName string
	PrincipalType ActorType
	SessionID     string
}

// NormalizeAuthenticatedCaller trims and canonicalizes one authenticated caller value.
func NormalizeAuthenticatedCaller(caller AuthenticatedCaller) AuthenticatedCaller {
	caller.PrincipalID = strings.TrimSpace(caller.PrincipalID)
	caller.PrincipalName = strings.TrimSpace(caller.PrincipalName)
	caller.SessionID = strings.TrimSpace(caller.SessionID)
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
