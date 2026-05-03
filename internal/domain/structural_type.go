package domain

import (
	"regexp"
	"slices"
	"strings"
)

// StructuralType represents the closed 4-value enum of cascade structural
// types per `ta-docs/cascade-methodology.md` §11.3. The four values describe
// where a node sits in the cascade tree's shape vocabulary, independent of
// its `Kind` (which describes the work-type axis).
type StructuralType string

// Built-in structural-type values. String values are lowercase ASCII letters
// only; none of the four values contain hyphens or digits.
const (
	StructuralTypeDrop       StructuralType = "drop"
	StructuralTypeSegment    StructuralType = "segment"
	StructuralTypeConfluence StructuralType = "confluence"
	StructuralTypeDroplet    StructuralType = "droplet"
)

// validStructuralTypes stores every member of the closed 4-value
// StructuralType enum in declaration order.
var validStructuralTypes = []StructuralType{
	StructuralTypeDrop,
	StructuralTypeSegment,
	StructuralTypeConfluence,
	StructuralTypeDroplet,
}

// structuralTypeDescriptionRegex matches a line of the form
// `StructuralType: <value>` where `<value>` is composed of lowercase ASCII
// letters only. The `(?m)` flag enables multiline mode so `^` and `$` anchor
// to line boundaries inside a multi-line description rather than just the
// string boundaries. The pattern is intentionally case-sensitive — a
// capitalized variant such as `StructuralType: Drop` produces no match,
// matching the acceptance contract that only the canonical lowercase form is
// recognized.
//
// The character class is tightened to `[a-z]+` rather than the Drop 2.2 Role
// precedent's `[a-z-]+` because none of the four StructuralType values
// (`drop`, `segment`, `confluence`, `droplet`) contain hyphens or digits.
// Per droplet 3.1 finding 5.A.10 this tightening is the recommended choice
// when no enum value needs the extra characters; it narrows the surface
// where a typo silently advances to the enum-membership check.
var structuralTypeDescriptionRegex = regexp.MustCompile(`(?m)^StructuralType:\s*([a-z]+)\s*$`)

// IsValidStructuralType reports whether s is a member of the closed
// StructuralType enum. The empty string is considered invalid; callers that
// want to permit an optional / unset structural type should short-circuit on
// emptiness before calling IsValidStructuralType. Surrounding whitespace and
// uppercase characters are normalized away before the membership check, so
// `"  Drop  "` is treated the same as `"drop"`.
func IsValidStructuralType(s StructuralType) bool {
	return slices.Contains(validStructuralTypes, StructuralType(strings.TrimSpace(strings.ToLower(string(s)))))
}

// NormalizeStructuralType canonicalizes a StructuralType value by trimming
// surrounding whitespace and lowercasing the input. Empty input — including
// whitespace-only input that collapses to empty after trimming — returns the
// empty string.
func NormalizeStructuralType(s StructuralType) StructuralType {
	trimmed := strings.TrimSpace(string(s))
	if trimmed == "" {
		return ""
	}
	return StructuralType(strings.ToLower(trimmed))
}

// ParseStructuralTypeFromDescription extracts a StructuralType from a
// free-form action-item description by scanning for the first line of the
// form `StructuralType: <value>`. The regex anchors to line boundaries via
// `(?m)`, so mid-paragraph occurrences of `StructuralType:` are ignored.
//
// Return contract:
//   - No `StructuralType:` line found → ("", nil).
//   - First matching line carries a value in the closed StructuralType enum
//     → (StructuralType, nil) where StructuralType is the typed constant.
//   - First matching line carries a value that does not appear in the closed
//     enum → ("", ErrInvalidStructuralType).
//
// Only the first match is consulted; subsequent `StructuralType:` lines are
// not inspected, so the description's earliest declaration wins.
func ParseStructuralTypeFromDescription(desc string) (StructuralType, error) {
	match := structuralTypeDescriptionRegex.FindStringSubmatch(desc)
	if match == nil {
		return "", nil
	}
	candidate := StructuralType(match[1])
	if !IsValidStructuralType(candidate) {
		return "", ErrInvalidStructuralType
	}
	return candidate, nil
}
