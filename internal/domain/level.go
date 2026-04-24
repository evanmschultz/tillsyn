package domain

import (
	"slices"
	"strings"
)

// ScopeLevel identifies one canonical hierarchy level.
type ScopeLevel string

// ScopeLevel values.
const (
	ScopeLevelProject    ScopeLevel = "project"
	ScopeLevelBranch     ScopeLevel = "branch"
	ScopeLevelPhase      ScopeLevel = "phase"
	ScopeLevelActionItem ScopeLevel = "actionItem"
	ScopeLevelSubtask    ScopeLevel = "subtask"
)

// validScopeLevels stores all supported level values.
var validScopeLevels = []ScopeLevel{
	ScopeLevelProject,
	ScopeLevelBranch,
	ScopeLevelPhase,
	ScopeLevelActionItem,
	ScopeLevelSubtask,
}

// LevelTuple stores one canonical scope tuple for level-scoped operations.
type LevelTuple struct {
	ProjectID string     `json:"project_id"`
	BranchID  string     `json:"branch_id,omitempty"`
	ScopeType ScopeLevel `json:"scope_type"`
	ScopeID   string     `json:"scope_id"`
}

// LevelTupleInput holds write-time values for LevelTuple normalization.
type LevelTupleInput struct {
	ProjectID string
	BranchID  string
	ScopeType ScopeLevel
	ScopeID   string
}

// NewLevelTuple validates and normalizes one level tuple.
func NewLevelTuple(in LevelTupleInput) (LevelTuple, error) {
	in.ProjectID = strings.TrimSpace(in.ProjectID)
	in.BranchID = strings.TrimSpace(in.BranchID)
	in.ScopeType = NormalizeScopeLevel(in.ScopeType)
	in.ScopeID = strings.TrimSpace(in.ScopeID)

	if in.ProjectID == "" {
		return LevelTuple{}, ErrInvalidID
	}
	if in.ScopeType == "" {
		in.ScopeType = ScopeLevelProject
	}
	if !IsValidScopeLevel(in.ScopeType) {
		return LevelTuple{}, ErrInvalidScopeType
	}
	if in.ScopeType == ScopeLevelProject && in.ScopeID == "" {
		in.ScopeID = in.ProjectID
	}
	if in.ScopeType != ScopeLevelProject && in.ScopeID == "" {
		return LevelTuple{}, ErrInvalidScopeID
	}
	if in.ScopeType == ScopeLevelBranch && in.BranchID == "" {
		in.BranchID = in.ScopeID
	}

	return LevelTuple{
		ProjectID: in.ProjectID,
		BranchID:  in.BranchID,
		ScopeType: in.ScopeType,
		ScopeID:   in.ScopeID,
	}, nil
}

// NormalizeScopeLevel canonicalizes one scope-level value. Inputs are
// matched case-insensitively against the supported set and returned in
// their canonical camelCase form (e.g. "actionItem"); unknown values are
// returned lowercased so callers can still detect invalid inputs.
func NormalizeScopeLevel(level ScopeLevel) ScopeLevel {
	lowered := strings.TrimSpace(strings.ToLower(string(level)))
	if lowered == "" {
		return ""
	}
	for _, candidate := range validScopeLevels {
		if strings.ToLower(string(candidate)) == lowered {
			return candidate
		}
	}
	return ScopeLevel(lowered)
}

// IsValidScopeLevel reports whether a scope-level value is supported.
func IsValidScopeLevel(level ScopeLevel) bool {
	level = NormalizeScopeLevel(level)
	return slices.Contains(validScopeLevels, level)
}

// ScopeLevelFromCapabilityScopeType converts a capability scope into a scope level.
func ScopeLevelFromCapabilityScopeType(scope CapabilityScopeType) ScopeLevel {
	switch NormalizeCapabilityScopeType(scope) {
	case CapabilityScopeProject:
		return ScopeLevelProject
	case CapabilityScopeBranch:
		return ScopeLevelBranch
	case CapabilityScopePhase:
		return ScopeLevelPhase
	case CapabilityScopeSubtask:
		return ScopeLevelSubtask
	case CapabilityScopeActionItem:
		return ScopeLevelActionItem
	default:
		return ""
	}
}

// ToCapabilityScopeType maps one level value into a capability scope value.
func (level ScopeLevel) ToCapabilityScopeType() CapabilityScopeType {
	switch NormalizeScopeLevel(level) {
	case ScopeLevelProject:
		return CapabilityScopeProject
	case ScopeLevelBranch:
		return CapabilityScopeBranch
	case ScopeLevelPhase:
		return CapabilityScopePhase
	case ScopeLevelSubtask:
		return CapabilityScopeSubtask
	case ScopeLevelActionItem:
		return CapabilityScopeActionItem
	default:
		return ""
	}
}
