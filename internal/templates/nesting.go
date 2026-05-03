package templates

import (
	"fmt"
	"slices"

	"github.com/evanmschultz/tillsyn/internal/domain"
)

// AllowsNesting reports whether a child kind may nest under a parent kind per
// the template's KindRule allow-lists. It returns (true, "") when the pairing
// is allowed and (false, reason) when the pairing is rejected. The reason
// string is stable enough for callers to assert against in tests and matches
// the format `kind %q cannot nest under parent kind %q (rule: %s)`.
//
// Resolution order, mirroring droplet 3.10's acceptance criteria:
//
//  1. If the parent kind has no rule recorded (literal Go zero-value
//     KindRule, i.e. either absent from t.Kinds or explicitly zero), nesting
//     is universally allowed — matches Drop 2.8's empty-AllowedParentScopes
//     semantics where no constraint means no rejection.
//  2. If the parent's AllowedChildKinds explicitly lists the child, the pair
//     is allowed regardless of any constraint on the child's side.
//  3. If the parent's AllowedChildKinds is non-empty and does NOT list the
//     child, the pair is rejected with the not-in-allowed-child-kinds reason.
//  4. If the child's AllowedParentKinds explicitly lists the parent, the pair
//     is allowed.
//  5. If the child's AllowedParentKinds is non-empty and does NOT list the
//     parent, the pair is rejected with the not-in-allowed-parent-kinds
//     reason.
//  6. If neither side carries a constraint that excludes the other, the
//     universal-allow fallback returns (true, "").
//
// Canonical spec: main/PLAN.md § 19.3 droplet 3.10 + finding 5.B.12.
func (t Template) AllowsNesting(parent, child domain.Kind) (allowed bool, reason string) {
	parentRule := t.Kinds[parent]
	if isZeroKindRule(parentRule) {
		return true, ""
	}

	if len(parentRule.AllowedChildKinds) > 0 {
		if slices.Contains(parentRule.AllowedChildKinds, child) {
			return true, ""
		}
		return false, fmt.Sprintf("kind %q cannot nest under parent kind %q (rule: %s)", child, parent, "not in allowed_child_kinds")
	}

	childRule := t.Kinds[child]
	if isZeroKindRule(childRule) {
		return true, ""
	}
	if len(childRule.AllowedParentKinds) > 0 {
		if slices.Contains(childRule.AllowedParentKinds, parent) {
			return true, ""
		}
		return false, fmt.Sprintf("kind %q cannot nest under parent kind %q (rule: %s)", child, parent, "not in allowed_parent_kinds")
	}

	return true, ""
}

// isZeroKindRule reports whether a KindRule carries no information at all
// (literal Go zero-value). Per droplet 3.10 step 1.1, a missing-from-map or
// explicitly-zero KindRule short-circuits AllowsNesting to universal-allow;
// any rule with at least one populated field continues to the allow-list
// resolution.
func isZeroKindRule(r KindRule) bool {
	return r.Owner == "" &&
		len(r.AllowedParentKinds) == 0 &&
		len(r.AllowedChildKinds) == 0 &&
		r.StructuralType == ""
}
