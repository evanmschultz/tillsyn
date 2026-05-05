package domain

import (
	"strings"
	"time"
)

// PermissionGrant records one approved tool-permission rule for a project +
// kind + CLI. Drop 4c F.7.17.7 introduces the substrate; the dispatcher
// reads grants when assembling per-spawn permission sets so previously
// approved rules do not re-prompt the dev.
//
// Identity model: PermissionGrant.ID is a string (mirroring every other
// domain identifier in this codebase — Project.ID, ActionItem.ID, etc.) and
// the storage layer persists it as TEXT. The spawn-prompt's `uuid.UUID`
// signature was a vocabulary suggestion; codebase consistency wins. Callers
// generate the ID (typically a UUID-shaped string) before calling
// NewPermissionGrant.
//
// Composite uniqueness key: (ProjectID, Kind, Rule, CLIKind). The storage
// adapter enforces UNIQUE on those four columns; re-inserting an identical
// quadruple is a noop and leaves the original row's GrantedAt + GrantedBy
// untouched.
type PermissionGrant struct {
	ID        string
	ProjectID string
	Kind      Kind
	Rule      string
	CLIKind   string
	GrantedBy string
	GrantedAt time.Time
}

// PermissionGrantInput holds write-time values for creating one grant.
//
// CLIKind is intentionally accepted as a free-form string at this boundary;
// closed-enum membership for the CLI vocabulary lives in the
// `internal/app/dispatcher` package and the templates layer. Domain-level
// validation only requires the value be non-empty so the storage adapter
// never sees a "" CLIKind in its UNIQUE composite. ID is supplied by the
// caller — the constructor validates non-empty rather than minting one
// internally to match the rest of the domain (Comment, Handoff, ...).
type PermissionGrantInput struct {
	ID        string
	ProjectID string
	Kind      Kind
	Rule      string
	CLIKind   string
	GrantedBy string
}

// NewPermissionGrant validates and normalizes one PermissionGrant.
//
// Validation:
//   - ID non-empty (after trim).
//   - ProjectID non-empty (after trim).
//   - Kind valid via IsValidKind. Empty / unknown values are rejected.
//   - Rule non-empty (after trim). No further pattern validation here —
//     callers are responsible for shape (e.g. "Bash(npm run *)").
//   - CLIKind non-empty (after trim + lower). The closed enum check runs at
//     the templates / dispatcher layer; the domain just refuses "".
//   - GrantedBy non-empty (after trim).
//
// Normalization:
//   - ID, ProjectID, Rule, GrantedBy: TrimSpace.
//   - Kind: pass-through (IsValidKind handles its own normalization).
//   - CLIKind: TrimSpace + ToLower so "Claude" / " claude " hash to one
//     storage form and the UNIQUE composite stays stable.
//   - GrantedAt: now.UTC().
func NewPermissionGrant(in PermissionGrantInput, now time.Time) (PermissionGrant, error) {
	in.ID = strings.TrimSpace(in.ID)
	if in.ID == "" {
		return PermissionGrant{}, ErrInvalidID
	}

	in.ProjectID = strings.TrimSpace(in.ProjectID)
	if in.ProjectID == "" {
		return PermissionGrant{}, ErrInvalidID
	}

	if !IsValidKind(in.Kind) {
		return PermissionGrant{}, ErrInvalidKind
	}

	in.Rule = strings.TrimSpace(in.Rule)
	if in.Rule == "" {
		return PermissionGrant{}, ErrInvalidPermissionGrantRule
	}

	in.CLIKind = strings.TrimSpace(strings.ToLower(in.CLIKind))
	if in.CLIKind == "" {
		return PermissionGrant{}, ErrInvalidPermissionGrantCLIKind
	}

	in.GrantedBy = strings.TrimSpace(in.GrantedBy)
	if in.GrantedBy == "" {
		return PermissionGrant{}, ErrInvalidPermissionGrantGrantedBy
	}

	return PermissionGrant{
		ID:        in.ID,
		ProjectID: in.ProjectID,
		Kind:      in.Kind,
		Rule:      in.Rule,
		CLIKind:   in.CLIKind,
		GrantedBy: in.GrantedBy,
		GrantedAt: now.UTC(),
	}, nil
}
