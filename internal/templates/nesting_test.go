package templates

import (
	"fmt"
	"testing"

	"github.com/evanmschultz/tillsyn/internal/domain"
)

// allKinds is the canonical declaration order of the closed 12-value Kind
// enum. It mirrors internal/domain/kind.go's validKinds slice and is used by
// the 144-row cartesian-product test below.
var allKinds = []domain.Kind{
	domain.KindPlan,
	domain.KindResearch,
	domain.KindBuild,
	domain.KindPlanQAProof,
	domain.KindPlanQAFalsification,
	domain.KindBuildQAProof,
	domain.KindBuildQAFalsification,
	domain.KindCloseout,
	domain.KindCommit,
	domain.KindRefinement,
	domain.KindDiscussion,
	domain.KindHumanVerify,
}

// allKindsExcept returns a copy of allKinds with the given kind removed. The
// helper exists to build AllowedChildKinds slices that enumerate every kind
// EXCEPT the one prohibited under a given parent — the canonical encoding for
// the four reverse-hierarchy prohibitions documented in main/PLAN.md § 19.3
// (closeout-no-closeout-parent, commit-no-plan-child, human-verify-no-build-
// child, build-qa-*-no-plan-child).
func allKindsExcept(exclude domain.Kind) []domain.Kind {
	out := make([]domain.Kind, 0, len(allKinds)-1)
	for _, k := range allKinds {
		if k == exclude {
			continue
		}
		out = append(out, k)
	}
	return out
}

// fixtureTemplate hand-codes the four reverse-hierarchy prohibitions on top
// of a permissive default. Per droplet 3.10 finding 5.B.12 (CE7), this test
// MUST NOT load default.toml — 3.14's embed_test.go independently asserts the
// loaded TOML round-trips against the same hand-coded fixture, giving the
// system two distinct assertion paths against one source of truth.
//
// Encoding strategy: each prohibition-source parent carries an
// AllowedChildKinds slice that enumerates every kind EXCEPT the prohibited
// one. Non-prohibition kinds are absent from the map, so AllowsNesting's
// step 1.1 universal-allow fallback applies to them.
func fixtureTemplate() Template {
	return Template{
		SchemaVersion: SchemaVersionV1,
		Kinds: map[domain.Kind]KindRule{
			// closeout-no-closeout-parent: closeout cannot parent another
			// closeout. Every other kind is allowed under closeout.
			domain.KindCloseout: {
				Owner:             "STEWARD",
				AllowedChildKinds: allKindsExcept(domain.KindCloseout),
				StructuralType:    domain.StructuralTypeDroplet,
			},
			// commit-no-plan-child: commit cannot parent a plan. Every other
			// kind is allowed under commit.
			domain.KindCommit: {
				AllowedChildKinds: allKindsExcept(domain.KindPlan),
				StructuralType:    domain.StructuralTypeDroplet,
			},
			// human-verify-no-build-child: human-verify cannot parent a
			// build. Every other kind is allowed under human-verify.
			domain.KindHumanVerify: {
				AllowedChildKinds: allKindsExcept(domain.KindBuild),
				StructuralType:    domain.StructuralTypeDroplet,
			},
			// build-qa-proof-no-plan-child: build-qa-proof cannot parent a
			// plan. Every other kind is allowed.
			domain.KindBuildQAProof: {
				AllowedChildKinds: allKindsExcept(domain.KindPlan),
				StructuralType:    domain.StructuralTypeDroplet,
			},
			// build-qa-falsification-no-plan-child: same prohibition as
			// build-qa-proof.
			domain.KindBuildQAFalsification: {
				AllowedChildKinds: allKindsExcept(domain.KindPlan),
				StructuralType:    domain.StructuralTypeDroplet,
			},
		},
	}
}

// expectedReject returns the prohibited child kind under the given parent
// kind in the fixtureTemplate, or empty if the parent has no prohibition.
// Mirrors the four reverse-hierarchy prohibitions encoded by
// fixtureTemplate.
func expectedReject(parent domain.Kind) (child domain.Kind, ok bool) {
	switch parent {
	case domain.KindCloseout:
		return domain.KindCloseout, true
	case domain.KindCommit:
		return domain.KindPlan, true
	case domain.KindHumanVerify:
		return domain.KindBuild, true
	case domain.KindBuildQAProof, domain.KindBuildQAFalsification:
		return domain.KindPlan, true
	}
	return "", false
}

// TestAllowsNestingCartesianProduct exhaustively asserts AllowsNesting's
// (allowed, reason) verdict for every (parent, child) pair in the closed
// 12-value Kind enum — 144 rows. The fixtureTemplate encodes the four
// reverse-hierarchy prohibitions documented in main/PLAN.md § 19.3, all
// other pairings universally allow.
func TestAllowsNestingCartesianProduct(t *testing.T) {
	t.Parallel()

	tpl := fixtureTemplate()

	type row struct {
		parent     domain.Kind
		child      domain.Kind
		wantAllow  bool
		wantReason string
	}

	rows := make([]row, 0, len(allKinds)*len(allKinds))
	for _, parent := range allKinds {
		for _, child := range allKinds {
			r := row{parent: parent, child: child, wantAllow: true, wantReason: ""}
			if rejectChild, has := expectedReject(parent); has && child == rejectChild {
				r.wantAllow = false
				r.wantReason = fmt.Sprintf("kind %q cannot nest under parent kind %q (rule: %s)", child, parent, "not in allowed_child_kinds")
			}
			rows = append(rows, r)
		}
	}

	if got, want := len(rows), 144; got != want {
		t.Fatalf("cartesian-product row count = %d; want %d", got, want)
	}

	for _, r := range rows {
		name := fmt.Sprintf("%s_under_%s", r.child, r.parent)
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			gotAllow, gotReason := tpl.AllowsNesting(r.parent, r.child)
			if gotAllow != r.wantAllow {
				t.Fatalf("AllowsNesting(%q, %q) allowed = %v; want %v (reason = %q)", r.parent, r.child, gotAllow, r.wantAllow, gotReason)
			}
			if gotReason != r.wantReason {
				t.Fatalf("AllowsNesting(%q, %q) reason = %q; want %q", r.parent, r.child, gotReason, r.wantReason)
			}
		})
	}
}

// TestAllowsNestingEmptyTemplateUniversalAllow verifies droplet 3.10 step
// 1.1: a Template with no Kinds map always returns (true, ""). Mirrors the
// Drop 2.8 universal-allow semantics for empty parent-scope rules.
func TestAllowsNestingEmptyTemplateUniversalAllow(t *testing.T) {
	t.Parallel()

	var empty Template
	gotAllow, gotReason := empty.AllowsNesting(domain.KindBuild, domain.KindBuildQAProof)
	if !gotAllow {
		t.Fatalf("Template{}.AllowsNesting(build, build-qa-proof) allowed = false; want true (reason = %q)", gotReason)
	}
	if gotReason != "" {
		t.Fatalf("Template{}.AllowsNesting(build, build-qa-proof) reason = %q; want empty", gotReason)
	}

	for _, parent := range allKinds {
		for _, child := range allKinds {
			allow, reason := empty.AllowsNesting(parent, child)
			if !allow || reason != "" {
				t.Fatalf("Template{}.AllowsNesting(%q, %q) = (%v, %q); want (true, \"\")", parent, child, allow, reason)
			}
		}
	}
}

// TestAllowsNestingChildSideAllowList covers AllowsNesting's child-side
// branch (steps 1.4 + 1.5) by constructing a Template whose parent rule has
// no AllowedChildKinds constraint but whose child rule restricts which
// parents may nest it. The branch is unreachable from fixtureTemplate alone
// because the four prohibitions all encode the constraint on the parent
// side.
func TestAllowsNestingChildSideAllowList(t *testing.T) {
	t.Parallel()

	tpl := Template{
		SchemaVersion: SchemaVersionV1,
		Kinds: map[domain.Kind]KindRule{
			// Parent rule exists (so step 1.1 does not short-circuit) but
			// records no nesting constraints — only Owner is set.
			domain.KindBuild: {
				Owner: "STEWARD",
			},
			// Child rule restricts allowed parents to {plan}; build is NOT
			// in the list, so build->build-qa-proof must reject.
			domain.KindBuildQAProof: {
				AllowedParentKinds: []domain.Kind{domain.KindPlan},
			},
			// Same kind with a plan-allow used to confirm the allow path.
			domain.KindBuildQAFalsification: {
				AllowedParentKinds: []domain.Kind{domain.KindPlan, domain.KindBuild},
			},
			// Constraint-free child rule (Owner set, no allow-lists) used to
			// cover the final universal-allow fallthrough — the branch where
			// both parent and child rules exist but neither carries an
			// allow-list.
			domain.KindResearch: {
				Owner: "STEWARD",
			},
		},
	}

	tests := []struct {
		name       string
		parent     domain.Kind
		child      domain.Kind
		wantAllow  bool
		wantReason string
	}{
		{
			name:       "child allow-list excludes parent rejects",
			parent:     domain.KindBuild,
			child:      domain.KindBuildQAProof,
			wantAllow:  false,
			wantReason: fmt.Sprintf("kind %q cannot nest under parent kind %q (rule: %s)", domain.KindBuildQAProof, domain.KindBuild, "not in allowed_parent_kinds"),
		},
		{
			name:      "child allow-list includes parent allows",
			parent:    domain.KindBuild,
			child:     domain.KindBuildQAFalsification,
			wantAllow: true,
		},
		{
			name:      "absent child rule falls through to universal-allow",
			parent:    domain.KindBuild,
			child:     domain.KindCommit,
			wantAllow: true,
		},
		{
			name:      "non-zero child rule with empty allow-list falls through to universal-allow",
			parent:    domain.KindBuild,
			child:     domain.KindResearch,
			wantAllow: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			gotAllow, gotReason := tpl.AllowsNesting(tc.parent, tc.child)
			if gotAllow != tc.wantAllow {
				t.Fatalf("AllowsNesting(%q, %q) allowed = %v; want %v (reason = %q)", tc.parent, tc.child, gotAllow, tc.wantAllow, gotReason)
			}
			if gotReason != tc.wantReason {
				t.Fatalf("AllowsNesting(%q, %q) reason = %q; want %q", tc.parent, tc.child, gotReason, tc.wantReason)
			}
		})
	}
}
