package templates

import (
	"reflect"
	"testing"

	"github.com/evanmschultz/tillsyn/internal/domain"
)

// childRulesFixture hand-codes a Template encoding the three auto-create
// rule families documented in main/PLAN.md § 19.3 line 1635:
//
//   - build → 2 children: build-qa-proof + build-qa-falsification.
//   - plan → 2 children: plan-qa-proof + plan-qa-falsification.
//   - structural_type=drop (parent kind=plan) → 3 droplet children: a
//     planner droplet (kind=plan), a qa-proof droplet (kind=plan-qa-proof),
//     and a qa-falsification droplet (kind=plan-qa-falsification). The
//     planner droplet is encoded as the existing plan-QA pair PLUS one
//     extra plan-kind child to mirror the line-1635 "planner droplet"
//     wording without inventing a new kind.
//
// The fixture also populates t.Kinds for every CreateChildKind so
// ChildRulesFor can resolve StructuralType + Owner from the child's row.
//
// Per droplet 3.11's contract this fixture intentionally does NOT load
// default.toml — the unit test isolates the resolution logic from the
// embedded-template round-trip surface (3.14).
func childRulesFixture() Template {
	return Template{
		SchemaVersion: SchemaVersionV1,
		Kinds: map[domain.Kind]KindRule{
			domain.KindPlan: {
				Owner:          "STEWARD",
				StructuralType: domain.StructuralTypeDroplet,
			},
			domain.KindBuild: {
				Owner:          "STEWARD",
				StructuralType: domain.StructuralTypeDroplet,
			},
			domain.KindBuildQAProof: {
				Owner:          "STEWARD",
				StructuralType: domain.StructuralTypeDroplet,
			},
			domain.KindBuildQAFalsification: {
				Owner:          "STEWARD",
				StructuralType: domain.StructuralTypeDroplet,
			},
			domain.KindPlanQAProof: {
				Owner:          "STEWARD",
				StructuralType: domain.StructuralTypeDroplet,
			},
			domain.KindPlanQAFalsification: {
				Owner:          "STEWARD",
				StructuralType: domain.StructuralTypeDroplet,
			},
		},
		ChildRules: []ChildRule{
			// build → build-qa-proof + build-qa-falsification.
			{
				WhenParentKind:  domain.KindBuild,
				CreateChildKind: domain.KindBuildQAProof,
				Title:           "BUILD-QA-PROOF",
				BlockedByParent: true,
			},
			{
				WhenParentKind:  domain.KindBuild,
				CreateChildKind: domain.KindBuildQAFalsification,
				Title:           "BUILD-QA-FALSIFICATION",
				BlockedByParent: true,
			},
			// plan → plan-qa-proof + plan-qa-falsification (no
			// structural-type filter — fires for every plan parent).
			{
				WhenParentKind:  domain.KindPlan,
				CreateChildKind: domain.KindPlanQAProof,
				Title:           "PLAN-QA-PROOF",
				BlockedByParent: true,
			},
			{
				WhenParentKind:  domain.KindPlan,
				CreateChildKind: domain.KindPlanQAFalsification,
				Title:           "PLAN-QA-FALSIFICATION",
				BlockedByParent: true,
			},
			// structural_type=drop on a plan parent adds one extra plan
			// (the "planner droplet" of PLAN.md line 1635). This rule
			// fires only when the parent has structural_type=drop.
			{
				WhenParentKind:           domain.KindPlan,
				CreateChildKind:          domain.KindPlan,
				Title:                    "DROP-PLANNER-DROPLET",
				BlockedByParent:          false,
				WhenParentStructuralType: domain.StructuralTypeDrop,
			},
		},
	}
}

// TestChildRulesFor_BuildSpawnsBuildQA covers the canonical build →
// build-qa-{proof,falsification} cascade. Two resolutions, deterministic
// (StructuralType, Kind) order with both children at structural_type=
// droplet — the alphabetic Kind tiebreaker places "build-qa-falsification"
// before "build-qa-proof".
func TestChildRulesFor_BuildSpawnsBuildQA(t *testing.T) {
	t.Parallel()

	tpl := childRulesFixture()
	got := tpl.ChildRulesFor(domain.KindBuild, domain.StructuralTypeDroplet)

	want := []ChildRuleResolution{
		{
			Kind:            domain.KindBuildQAFalsification,
			Title:           "BUILD-QA-FALSIFICATION",
			BlockedByParent: true,
			StructuralType:  domain.StructuralTypeDroplet,
			Persistent:      false,
			DevGated:        false,
			Owner:           "STEWARD",
		},
		{
			Kind:            domain.KindBuildQAProof,
			Title:           "BUILD-QA-PROOF",
			BlockedByParent: true,
			StructuralType:  domain.StructuralTypeDroplet,
			Persistent:      false,
			DevGated:        false,
			Owner:           "STEWARD",
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ChildRulesFor(build, droplet) =\n%#v\nwant\n%#v", got, want)
	}
}

// TestChildRulesFor_PlanSpawnsPlanQA covers the canonical plan →
// plan-qa-{proof,falsification} cascade. Two resolutions at
// structural_type=droplet — the structural-type filter on the planner
// droplet rule keeps it OUT of this result because parentType is droplet,
// not drop.
func TestChildRulesFor_PlanSpawnsPlanQA(t *testing.T) {
	t.Parallel()

	tpl := childRulesFixture()
	got := tpl.ChildRulesFor(domain.KindPlan, domain.StructuralTypeDroplet)

	want := []ChildRuleResolution{
		{
			Kind:            domain.KindPlanQAFalsification,
			Title:           "PLAN-QA-FALSIFICATION",
			BlockedByParent: true,
			StructuralType:  domain.StructuralTypeDroplet,
			Persistent:      false,
			DevGated:        false,
			Owner:           "STEWARD",
		},
		{
			Kind:            domain.KindPlanQAProof,
			Title:           "PLAN-QA-PROOF",
			BlockedByParent: true,
			StructuralType:  domain.StructuralTypeDroplet,
			Persistent:      false,
			DevGated:        false,
			Owner:           "STEWARD",
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ChildRulesFor(plan, droplet) =\n%#v\nwant\n%#v", got, want)
	}
}

// TestChildRulesFor_DropStructuralTypeSpawnsThreeChildren covers the
// PLAN.md line 1635 case: a plan parent with structural_type=drop fires
// THREE rules — the universal plan-QA-proof + plan-QA-falsification
// pair PLUS the structural-type-gated planner droplet. Sorted by
// (StructuralType, Kind) ascending, all three sit at droplet so the
// secondary Kind tiebreaker orders them as plan, plan-qa-falsification,
// plan-qa-proof.
func TestChildRulesFor_DropStructuralTypeSpawnsThreeChildren(t *testing.T) {
	t.Parallel()

	tpl := childRulesFixture()
	got := tpl.ChildRulesFor(domain.KindPlan, domain.StructuralTypeDrop)

	want := []ChildRuleResolution{
		{
			Kind:            domain.KindPlan,
			Title:           "DROP-PLANNER-DROPLET",
			BlockedByParent: false,
			StructuralType:  domain.StructuralTypeDroplet,
			Persistent:      false,
			DevGated:        false,
			Owner:           "STEWARD",
		},
		{
			Kind:            domain.KindPlanQAFalsification,
			Title:           "PLAN-QA-FALSIFICATION",
			BlockedByParent: true,
			StructuralType:  domain.StructuralTypeDroplet,
			Persistent:      false,
			DevGated:        false,
			Owner:           "STEWARD",
		},
		{
			Kind:            domain.KindPlanQAProof,
			Title:           "PLAN-QA-PROOF",
			BlockedByParent: true,
			StructuralType:  domain.StructuralTypeDroplet,
			Persistent:      false,
			DevGated:        false,
			Owner:           "STEWARD",
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ChildRulesFor(plan, drop) =\n%#v\nwant\n%#v", got, want)
	}
}

// TestChildRulesFor_NoMatchingRulesReturnsEmpty covers the universal-no-
// match branch: a kind with no matching ChildRules entries returns an
// empty (length-zero, non-nil) slice. Research has no [child_rules] entry
// in the fixture, so every iteration step skips on WhenParentKind.
func TestChildRulesFor_NoMatchingRulesReturnsEmpty(t *testing.T) {
	t.Parallel()

	tpl := childRulesFixture()
	got := tpl.ChildRulesFor(domain.KindResearch, "")

	if got == nil {
		t.Fatalf("ChildRulesFor(research, \"\") = nil; want non-nil empty slice")
	}
	if len(got) != 0 {
		t.Fatalf("ChildRulesFor(research, \"\") = %#v; want empty", got)
	}
}

// TestChildRulesFor_StructuralTypeFilterRejectsNonMatching directly
// covers acceptance bullet 2.2.5: a rule with a non-empty
// WhenParentStructuralType MUST NOT fire when called with a different
// parentType. The fixture's structural-type-gated rule expects
// parentType=drop; calling with parentType=droplet must yield only the
// two unconditional plan-QA rules — the third "planner droplet" rule
// stays out.
func TestChildRulesFor_StructuralTypeFilterRejectsNonMatching(t *testing.T) {
	t.Parallel()

	tpl := childRulesFixture()
	got := tpl.ChildRulesFor(domain.KindPlan, domain.StructuralTypeDroplet)

	if len(got) != 2 {
		t.Fatalf("ChildRulesFor(plan, droplet) length = %d; want 2 (structural-type filter must exclude the drop-only planner-droplet rule)", len(got))
	}
	for _, r := range got {
		if r.Kind == domain.KindPlan {
			t.Fatalf("ChildRulesFor(plan, droplet) included a kind=plan resolution (%#v); the rule guarded by WhenParentStructuralType=drop must not fire under parentType=droplet", r)
		}
	}
}

// TestChildRulesFor_DeterministicOrder asserts that two identical calls
// against the same template produce identical slices. Map iteration in
// Go is intentionally nondeterministic, so any implementation that walks
// t.Kinds (a map) without a stable sort would flake here. Two calls is
// enough to catch the obvious reordering bug in CI; the rule still holds
// across arbitrary call counts because sort.SliceStable is deterministic.
func TestChildRulesFor_DeterministicOrder(t *testing.T) {
	t.Parallel()

	tpl := childRulesFixture()
	first := tpl.ChildRulesFor(domain.KindPlan, domain.StructuralTypeDrop)
	second := tpl.ChildRulesFor(domain.KindPlan, domain.StructuralTypeDrop)
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("ChildRulesFor not deterministic across calls\nfirst  = %#v\nsecond = %#v", first, second)
	}
}

// TestChildRulesFor_EmptyTemplateReturnsEmpty covers the zero-value
// Template path — no Kinds, no ChildRules — so every parent kind / type
// combination must yield an empty slice. The branch matters because
// dispatcher startup may legitimately consult a Template that has not
// been populated yet (e.g. a project still in setup).
func TestChildRulesFor_EmptyTemplateReturnsEmpty(t *testing.T) {
	t.Parallel()

	var empty Template
	for _, parent := range allKinds {
		for _, st := range []domain.StructuralType{
			"",
			domain.StructuralTypeDrop,
			domain.StructuralTypeSegment,
			domain.StructuralTypeConfluence,
			domain.StructuralTypeDroplet,
		} {
			got := empty.ChildRulesFor(parent, st)
			if len(got) != 0 {
				t.Fatalf("Template{}.ChildRulesFor(%q, %q) = %#v; want empty", parent, st, got)
			}
		}
	}
}
