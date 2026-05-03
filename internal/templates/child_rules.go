package templates

import (
	"sort"

	"github.com/evanmschultz/tillsyn/internal/domain"
)

// ChildRuleResolution is one materialized child specification produced by
// Template.ChildRulesFor. The auto-generator (Drop 3 droplet 3.20) creates
// one ActionItem per resolution.
//
// The struct intentionally mirrors only what the auto-generator needs at
// create time: the child kind + title + blocked_by wiring + structural-type
// classification + STEWARD-ownership flag (Drop 3 fix L7) + the
// methodology-§11.2 persistence / dev-gating axes (Drop 3 fixes L9 + L10).
// Persistent and DevGated remain hardcoded false in this droplet because
// the [child_rules] TOML schema does not yet expose those bits — the
// methodology calls them out as forward-compat axes the auto-generator
// reads from KindRule once Drop 3.13's expanded validator lands.
type ChildRuleResolution struct {
	// Kind is the closed-enum Kind of the child to auto-create.
	Kind domain.Kind

	// Title is the literal title applied to the auto-created child,
	// copied verbatim from ChildRule.Title.
	Title string

	// BlockedByParent, when true, instructs the auto-generator to wire a
	// blocked_by edge from the child to the parent so the child cannot
	// start until the parent reaches its terminal completion state.
	BlockedByParent bool

	// StructuralType carries the cascade-shape classification the child
	// inherits from its KindRule. The auto-generator writes this into
	// metadata.structural_type at creation time so plan-QA-falsification
	// rules (PLAN.md § 19.3 line 1633) have a value to attack against.
	StructuralType domain.StructuralType

	// Persistent is the methodology §11.2 persistence flag indicating a
	// long-lived coordination anchor (PLAN.md § 19.3 line 1637 — STEWARD's
	// six persistent level_1 parents). Hardcoded false in droplet 3.11 —
	// the [child_rules] schema does not yet expose this axis.
	Persistent bool

	// DevGated is the methodology §11.2 dev-sign-off flag indicating the
	// child cannot auto-progress past a checkpoint without explicit dev
	// approval. Hardcoded false in droplet 3.11 — the [child_rules]
	// schema does not yet expose this axis.
	DevGated bool

	// Owner is the principal identifier responsible for materializing the
	// child (Drop 3 fix L7). The auto-generator routes the create call to
	// the named principal — "STEWARD" for STEWARD-owned children, other
	// values are accepted verbatim. Sourced from the child kind's
	// KindRule.Owner.
	Owner string
}

// ChildRulesFor returns the child specifications to auto-create for a
// newly-created action-item with the given kind and structural type.
//
// Resolution order, mirroring droplet 3.11's acceptance criteria:
//
//  1. Iterate t.ChildRules in declaration order.
//  2. Skip a rule whose WhenParentKind != parent.
//  3. Skip a rule whose WhenParentStructuralType is non-empty AND does not
//     equal parentType. An empty WhenParentStructuralType matches every
//     parent type (universal-allow on the structural-type axis).
//  4. Resolve the child's KindRule from t.Kinds[rule.CreateChildKind] for
//     the StructuralType + Owner defaults; the child's KindRule is the
//     authoritative source for those axes per droplet 3.10's universal-
//     allow rules. Absent KindRule rows yield zero values, which is
//     correct for the universal-allow case.
//  5. Build a ChildRuleResolution capturing the rule's own fields plus the
//     child KindRule's StructuralType + Owner.
//
// The returned slice is sorted in stable, deterministic order by
// (StructuralType, Kind) ascending so callers can assert exact sequences
// in tests without depending on map iteration nondeterminism. Stable sort
// preserves declaration order for any pair of resolutions that compare
// equal under the sort key, which keeps round-trip behavior predictable
// for templates that declare two rules with the same (StructuralType,
// Kind) under different parents — though the closed schema and the
// load-time cycle validator make that combination unusual.
//
// One level only: the returned slice describes the children of the
// supplied parent, not its grandchildren. The dispatcher (Drop 4)
// recursively expands the cascade by calling ChildRulesFor again on each
// auto-created child as it transitions to in_progress.
//
// Pure function: no I/O, no DB access, no side effects. Safe to call from
// any goroutine; the receiver Template is read but not mutated.
//
// Canonical spec: main/PLAN.md § 19.3 droplet 3.11 + line 1635.
func (t Template) ChildRulesFor(parent domain.Kind, parentType domain.StructuralType) []ChildRuleResolution {
	out := make([]ChildRuleResolution, 0, len(t.ChildRules))
	for _, rule := range t.ChildRules {
		if rule.WhenParentKind != parent {
			continue
		}
		if rule.WhenParentStructuralType != "" && rule.WhenParentStructuralType != parentType {
			continue
		}
		childRule := t.Kinds[rule.CreateChildKind]
		out = append(out, ChildRuleResolution{
			Kind:            rule.CreateChildKind,
			Title:           rule.Title,
			BlockedByParent: rule.BlockedByParent,
			StructuralType:  childRule.StructuralType,
			Persistent:      false,
			DevGated:        false,
			Owner:           childRule.Owner,
		})
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].StructuralType != out[j].StructuralType {
			return out[i].StructuralType < out[j].StructuralType
		}
		return out[i].Kind < out[j].Kind
	})
	return out
}
