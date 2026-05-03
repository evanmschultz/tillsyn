package domain

import (
	"strings"
	"time"
)

// NewActionItemForTest is a TEST-ONLY fixture constructor. It wraps
// NewActionItem so cross-package test fixtures that pre-date droplet 3.2
// (which made StructuralType a required field on ActionItemInput) continue
// to construct valid rows without forcing every fixture to spell the
// structural type explicitly.
//
// Production code MUST use NewActionItem directly so the
// ErrInvalidStructuralType enforcement fires; this helper is reserved for
// `*_test.go` files that build ActionItem fixtures across the six packages
// (`internal/domain`, `internal/app`, `internal/adapters/storage/sqlite`,
// `internal/tui`, `cmd/till`) touched by the droplet 3.2 sweep.
//
// Defaulting behaviour (mirrors the Drop 1.75 Kind-default precedent at
// `internal/tui/model_test.go` `newActionItemForTest`):
//   - If `in.StructuralType` (after NormalizeStructuralType) is empty,
//     default it to StructuralTypeDroplet — the dominant cascade-leaf shape.
//   - Other fields are passed through unchanged. Role's existing permissive
//     empty semantics are preserved.
//
// Future hardening could move this helper to a dedicated `domaintest`
// sub-package so it is unimportable from non-test files; doing so requires
// renaming every call site, so it is left as a refinement.
func NewActionItemForTest(in ActionItemInput, now time.Time) (ActionItem, error) {
	if strings.TrimSpace(string(in.StructuralType)) == "" {
		in.StructuralType = StructuralTypeDroplet
	}
	return NewActionItem(in, now)
}
