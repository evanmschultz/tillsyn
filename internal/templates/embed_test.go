package templates

import (
	"slices"
	"testing"

	"github.com/evanmschultz/tillsyn/internal/domain"
)

// loadDefaultOrFatal is the shared helper for embed tests. Centralizing the
// LoadDefaultTemplate invocation keeps each test focused on its assertion and
// gives a single failure point if the embed pipeline regresses (e.g. the
// //go:embed directive falls out of sync with the on-disk file path).
func loadDefaultOrFatal(t *testing.T) Template {
	t.Helper()
	tpl, err := LoadDefaultTemplate()
	if err != nil {
		t.Fatalf("LoadDefaultTemplate(): unexpected error: %v", err)
	}
	return tpl
}

// TestDefaultTemplateLoadsCleanly verifies the embedded builtin/default.toml
// parses + validates without error. Any sentinel from load.go (unknown key,
// schema-version mismatch, unknown kind reference, child-rule cycle) would
// surface here, so this is the canary for the whole embed pipeline.
func TestDefaultTemplateLoadsCleanly(t *testing.T) {
	t.Parallel()

	tpl, err := LoadDefaultTemplate()
	if err != nil {
		t.Fatalf("LoadDefaultTemplate(): unexpected error: %v", err)
	}
	if tpl.SchemaVersion != SchemaVersionV1 {
		t.Fatalf("SchemaVersion = %q; want %q", tpl.SchemaVersion, SchemaVersionV1)
	}
}

// TestDefaultTemplateCoversAllTwelveKinds asserts every member of the closed
// 12-value Kind enum has a [kinds.<kind>] section. Mirrors the assertion in
// the soon-to-be-deleted internal/adapters/storage/sqlite/repo_test.go
// TestRepositoryFreshOpenKindCatalog (finding 5.B.8 CE3): the equivalent
// "all 12 kinds present" guarantee migrates here so deleting the legacy
// boot-seed test in 3.15 leaves no coverage gap.
func TestDefaultTemplateCoversAllTwelveKinds(t *testing.T) {
	t.Parallel()

	tpl := loadDefaultOrFatal(t)
	if got, want := len(tpl.Kinds), len(allKinds); got != want {
		t.Fatalf("len(Kinds) = %d; want %d (all 12 kinds covered)", got, want)
	}
	for _, kind := range allKinds {
		if _, ok := tpl.Kinds[kind]; !ok {
			t.Fatalf("Kinds[%q] missing — every closed-12-kind must have a [kinds.<kind>] section", kind)
		}
	}
}

// TestDefaultTemplateRejectsReverseHierarchyProhibitions asserts the four
// PLAN.md § 19.3 reverse-hierarchy prohibitions are EXPLICITLY rejected by
// the loaded template's AllowsNesting. Per finding 5.B.16 (N3 explicit-deny)
// these are NOT implicit-by-absence — adding a 13th kind in a future drop
// must require explicit opt-in via the existing allow-list.
func TestDefaultTemplateRejectsReverseHierarchyProhibitions(t *testing.T) {
	t.Parallel()

	tpl := loadDefaultOrFatal(t)

	prohibitions := []struct {
		name   string
		parent domain.Kind
		child  domain.Kind
	}{
		{"closeout-no-closeout-parent", domain.KindCloseout, domain.KindCloseout},
		{"commit-no-plan-child", domain.KindCommit, domain.KindPlan},
		{"human-verify-no-build-child", domain.KindHumanVerify, domain.KindBuild},
		{"build-qa-proof-no-plan-child", domain.KindBuildQAProof, domain.KindPlan},
		{"build-qa-falsification-no-plan-child", domain.KindBuildQAFalsification, domain.KindPlan},
	}

	for _, p := range prohibitions {
		t.Run(p.name, func(t *testing.T) {
			t.Parallel()
			allowed, reason := tpl.AllowsNesting(p.parent, p.child)
			if allowed {
				t.Fatalf("AllowsNesting(%q, %q) = (true, _); want (false, _) — reverse-hierarchy prohibition must reject", p.parent, p.child)
			}
			if reason == "" {
				t.Fatalf("AllowsNesting(%q, %q) returned empty reason; want non-empty rejection reason", p.parent, p.child)
			}
		})
	}
}

// TestDefaultTemplateAllowsLegitimateNestings spot-checks that the four
// reverse-hierarchy prohibitions did not over-constrain — common legitimate
// nestings still pass AllowsNesting.
func TestDefaultTemplateAllowsLegitimateNestings(t *testing.T) {
	t.Parallel()

	tpl := loadDefaultOrFatal(t)

	allowed := []struct {
		name   string
		parent domain.Kind
		child  domain.Kind
	}{
		{"plan_under_plan", domain.KindPlan, domain.KindPlan},
		{"plan_qa_proof_under_plan", domain.KindPlan, domain.KindPlanQAProof},
		{"plan_qa_falsification_under_plan", domain.KindPlan, domain.KindPlanQAFalsification},
		{"build_under_plan", domain.KindPlan, domain.KindBuild},
		{"build_qa_proof_under_build", domain.KindBuild, domain.KindBuildQAProof},
		{"build_qa_falsification_under_build", domain.KindBuild, domain.KindBuildQAFalsification},
		{"research_under_plan", domain.KindPlan, domain.KindResearch},
		{"discussion_under_plan", domain.KindPlan, domain.KindDiscussion},
	}

	for _, a := range allowed {
		t.Run(a.name, func(t *testing.T) {
			t.Parallel()
			ok, reason := tpl.AllowsNesting(a.parent, a.child)
			if !ok {
				t.Fatalf("AllowsNesting(%q, %q) = (false, %q); want (true, \"\")", a.parent, a.child, reason)
			}
			if reason != "" {
				t.Fatalf("AllowsNesting(%q, %q) reason = %q; want empty", a.parent, a.child, reason)
			}
		})
	}
}

// TestDefaultTemplateChildRulesForBuild verifies the auto-create rules for a
// build parent: build → build-qa-proof + build-qa-falsification per PLAN.md
// § 19.3 line 1635. Both children must carry blocked_by_parent=true so they
// cannot start until the parent reaches its terminal state.
func TestDefaultTemplateChildRulesForBuild(t *testing.T) {
	t.Parallel()

	tpl := loadDefaultOrFatal(t)
	resolutions := tpl.ChildRulesFor(domain.KindBuild, domain.StructuralTypeDroplet)
	if got, want := len(resolutions), 2; got != want {
		t.Fatalf("ChildRulesFor(build, droplet) returned %d resolutions; want %d", got, want)
	}

	wantKinds := map[domain.Kind]bool{
		domain.KindBuildQAProof:         false,
		domain.KindBuildQAFalsification: false,
	}
	for _, res := range resolutions {
		if _, expected := wantKinds[res.Kind]; !expected {
			t.Fatalf("ChildRulesFor(build, droplet) returned unexpected kind %q", res.Kind)
		}
		if !res.BlockedByParent {
			t.Fatalf("resolution kind %q BlockedByParent = false; want true", res.Kind)
		}
		wantKinds[res.Kind] = true
	}
	for kind, seen := range wantKinds {
		if !seen {
			t.Fatalf("ChildRulesFor(build, droplet) missing expected child kind %q", kind)
		}
	}
}

// TestDefaultTemplateChildRulesForPlan verifies the auto-create rules for a
// plain (non-drop) plan parent: plan → plan-qa-proof + plan-qa-falsification.
// Drop-structural plans have a different rule set (covered by
// TestDefaultTemplateChildRulesForDropPlan).
func TestDefaultTemplateChildRulesForPlan(t *testing.T) {
	t.Parallel()

	tpl := loadDefaultOrFatal(t)
	resolutions := tpl.ChildRulesFor(domain.KindPlan, domain.StructuralTypeDroplet)
	if got, want := len(resolutions), 2; got != want {
		t.Fatalf("ChildRulesFor(plan, droplet) returned %d resolutions; want %d", got, want)
	}

	wantKinds := map[domain.Kind]bool{
		domain.KindPlanQAProof:         false,
		domain.KindPlanQAFalsification: false,
	}
	for _, res := range resolutions {
		if _, expected := wantKinds[res.Kind]; !expected {
			t.Fatalf("ChildRulesFor(plan, droplet) returned unexpected kind %q", res.Kind)
		}
		if !res.BlockedByParent {
			t.Fatalf("resolution kind %q BlockedByParent = false; want true", res.Kind)
		}
		wantKinds[res.Kind] = true
	}
	for kind, seen := range wantKinds {
		if !seen {
			t.Fatalf("ChildRulesFor(plan, droplet) missing expected child kind %q", kind)
		}
	}
}

// TestDefaultTemplateChildRulesForDropPlan verifies the drop-level rule set:
// when the parent is a plan with structural_type=drop, ChildRulesFor returns
// the two universal-plan rules PLUS the two drop-specific QA-twin rules.
//
// The drop-planner droplet rule named by PLAN.md § 19.3 line 1635 is
// DEFERRED because it produces a plan->plan self-loop the load-time
// cycle validator rejects (see comment in default.toml). The drop-orch
// creates the drop-planner manually pre-cascade.
func TestDefaultTemplateChildRulesForDropPlan(t *testing.T) {
	t.Parallel()

	tpl := loadDefaultOrFatal(t)
	resolutions := tpl.ChildRulesFor(domain.KindPlan, domain.StructuralTypeDrop)

	// Two universal-plan rules + two drop-specific QA-twin rules = four total.
	if got, want := len(resolutions), 4; got != want {
		t.Fatalf("ChildRulesFor(plan, drop) returned %d resolutions; want %d", got, want)
	}

	gotKinds := make(map[domain.Kind]int, len(resolutions))
	for _, res := range resolutions {
		gotKinds[res.Kind]++
	}
	if gotKinds[domain.KindPlanQAProof] != 2 {
		t.Fatalf("ChildRulesFor(plan, drop) plan-qa-proof count = %d; want 2 (universal + drop-specific)", gotKinds[domain.KindPlanQAProof])
	}
	if gotKinds[domain.KindPlanQAFalsification] != 2 {
		t.Fatalf("ChildRulesFor(plan, drop) plan-qa-falsification count = %d; want 2 (universal + drop-specific)", gotKinds[domain.KindPlanQAFalsification])
	}
}

// TestDefaultTemplateAgentBindingsCoverAllKinds asserts every closed-12-kind
// has a populated [agent_bindings.<kind>] section AND the binding passes
// AgentBinding.Validate. Mirrors the spirit of the deleted
// TestRepositoryFreshOpenKindCatalogUniversalParentAllow assertion (finding
// 5.B.8 CE3) — every kind has the configuration the dispatcher needs.
func TestDefaultTemplateAgentBindingsCoverAllKinds(t *testing.T) {
	t.Parallel()

	tpl := loadDefaultOrFatal(t)
	if got, want := len(tpl.AgentBindings), len(allKinds); got != want {
		t.Fatalf("len(AgentBindings) = %d; want %d", got, want)
	}
	for _, kind := range allKinds {
		binding, ok := tpl.AgentBindings[kind]
		if !ok {
			t.Fatalf("AgentBindings[%q] missing", kind)
		}
		if err := binding.Validate(); err != nil {
			t.Fatalf("AgentBindings[%q].Validate(): %v (binding = %#v)", kind, err, binding)
		}
	}
}

// TestDefaultTemplateBuildersRunOpus asserts the pre-MVP "no optimization
// before measurement" rule encoded in memory feedback_opus_builders_pre_mvp.md:
// builders + QA agents + planning + research run opus until cascade
// dogfooding lands. The commit kind binds haiku per CLAUDE.md.
func TestDefaultTemplateBuildersRunOpus(t *testing.T) {
	t.Parallel()

	tpl := loadDefaultOrFatal(t)

	opusKinds := []domain.Kind{
		domain.KindPlan,
		domain.KindResearch,
		domain.KindBuild,
		domain.KindPlanQAProof,
		domain.KindPlanQAFalsification,
		domain.KindBuildQAProof,
		domain.KindBuildQAFalsification,
	}
	for _, kind := range opusKinds {
		binding, ok := tpl.AgentBindings[kind]
		if !ok {
			t.Fatalf("AgentBindings[%q] missing", kind)
		}
		if binding.Model != "opus" {
			t.Fatalf("AgentBindings[%q].Model = %q; want %q (pre-MVP rule)", kind, binding.Model, "opus")
		}
	}

	commitBinding, ok := tpl.AgentBindings[domain.KindCommit]
	if !ok {
		t.Fatalf("AgentBindings[commit] missing")
	}
	if commitBinding.Model != "haiku" {
		t.Fatalf("AgentBindings[commit].Model = %q; want %q (CLAUDE.md commit-message-agent)", commitBinding.Model, "haiku")
	}
}

// TestDefaultTemplateMatchesNestingFixture cross-validates the loaded
// default.toml against the hand-coded fixtureTemplate() in nesting_test.go
// per finding 5.B.12 (CE7). The two assertion paths share one source of
// truth: the four reverse-hierarchy prohibitions. We assert that for every
// (parent, child) pair the hand-coded fixture rejects, the loaded template
// also rejects. We do NOT assert literal Template equality because the
// loaded TOML carries agent_bindings + child_rules + extra kind rows the
// hand-coded fixture deliberately omits — only the prohibition set is the
// shared ground truth.
func TestDefaultTemplateMatchesNestingFixture(t *testing.T) {
	t.Parallel()

	loaded := loadDefaultOrFatal(t)
	fixture := fixtureTemplate()

	for _, parent := range allKinds {
		for _, child := range allKinds {
			fixtureAllow, _ := fixture.AllowsNesting(parent, child)
			if fixtureAllow {
				continue
			}
			loadedAllow, loadedReason := loaded.AllowsNesting(parent, child)
			if loadedAllow {
				t.Fatalf("loaded default.toml AllowsNesting(%q, %q) = true; fixture rejects — prohibition set drifted", parent, child)
			}
			if loadedReason == "" {
				t.Fatalf("loaded default.toml AllowsNesting(%q, %q) reason empty; fixture rejects with non-empty reason", parent, child)
			}
		}
	}
}

// TestDefaultTemplateStewardOwnedKinds verifies the kinds that PLAN.md
// § 15.7 (§ 19.3 bullet 9) names as STEWARD-owned have owner = "STEWARD"
// in their KindRule. The auth gate at internal/adapters/server/common/
// app_service_adapter_mcp.go reads this field via 3.20's auto-generator;
// regression here would silently let drop-orchs move STEWARD-owned items.
func TestDefaultTemplateStewardOwnedKinds(t *testing.T) {
	t.Parallel()

	tpl := loadDefaultOrFatal(t)
	stewardKinds := []domain.Kind{
		domain.KindCloseout,
		domain.KindRefinement,
	}
	for _, kind := range stewardKinds {
		rule, ok := tpl.Kinds[kind]
		if !ok {
			t.Fatalf("Kinds[%q] missing — STEWARD-owned kinds must have a [kinds.<kind>] section", kind)
		}
		if rule.Owner != "STEWARD" {
			t.Fatalf("Kinds[%q].Owner = %q; want %q (STEWARD-owned per PLAN.md § 15.7)", kind, rule.Owner, "STEWARD")
		}
	}
}

// TestDefaultTemplateLoadsWithGates asserts the embedded default.toml decodes
// the Drop 4b Wave A 4b.1 [gates] section: [gates.build] = ["mage_ci"]. Per
// REVISION_BRIEF locked decision L6 the build sequence stays minimal in
// Drop 4b — Drop 4c expands to ["mage_ci", "commit", "push"]. Other kinds
// (plan-qa-proof, build-qa-proof, closeout, etc.) are ABSENT from [gates.*];
// the gate runner treats absence as "no gates" not "all gates" per the
// 4b.2 doc-comment.
func TestDefaultTemplateLoadsWithGates(t *testing.T) {
	t.Parallel()

	tpl := loadDefaultOrFatal(t)

	gateSeq, ok := tpl.Gates[domain.KindBuild]
	if !ok {
		t.Fatalf("Gates[%q] missing — Drop 4b Wave A 4b.1 ships [gates.build] = [\"mage_ci\"]", domain.KindBuild)
	}
	if len(gateSeq) != 1 {
		t.Fatalf("Gates[%q] len = %d; want 1 (Drop 4b L6: only mage_ci ships in default; Drop 4c expands)", domain.KindBuild, len(gateSeq))
	}
	if gateSeq[0] != GateKindMageCI {
		t.Fatalf("Gates[%q][0] = %q; want %q", domain.KindBuild, gateSeq[0], GateKindMageCI)
	}

	// Sibling kinds carry no gate sequence — the gate runner treats absence
	// as "no gates" (returns Success: true immediately).
	absentKinds := []domain.Kind{
		domain.KindPlan,
		domain.KindResearch,
		domain.KindBuildQAProof,
		domain.KindBuildQAFalsification,
		domain.KindPlanQAProof,
		domain.KindPlanQAFalsification,
		domain.KindCloseout,
		domain.KindCommit,
		domain.KindRefinement,
		domain.KindDiscussion,
		domain.KindHumanVerify,
	}
	for _, kind := range absentKinds {
		if _, present := tpl.Gates[kind]; present {
			t.Fatalf("Gates[%q] should be absent in Drop 4b default — only build carries a gate sequence", kind)
		}
	}
}

// TestDefaultTemplateProhibitionsAreExplicit asserts the four reverse-
// hierarchy prohibitions are encoded via NON-EMPTY allowed_child_kinds
// allow-lists, not via implicit absence. Per finding 5.B.16 (N3) adding a
// 13th kind must be an explicit opt-in. We enforce this by checking that
// the prohibition-source parent kinds carry an allowed_child_kinds slice
// of length 11 (the closed enum minus the prohibited child).
func TestDefaultTemplateProhibitionsAreExplicit(t *testing.T) {
	t.Parallel()

	tpl := loadDefaultOrFatal(t)

	cases := []struct {
		name             string
		parent           domain.Kind
		prohibitedChild  domain.Kind
		wantAllowListLen int
	}{
		{"closeout", domain.KindCloseout, domain.KindCloseout, len(allKinds) - 1},
		{"commit", domain.KindCommit, domain.KindPlan, len(allKinds) - 1},
		{"human-verify", domain.KindHumanVerify, domain.KindBuild, len(allKinds) - 1},
		{"build-qa-proof", domain.KindBuildQAProof, domain.KindPlan, len(allKinds) - 1},
		{"build-qa-falsification", domain.KindBuildQAFalsification, domain.KindPlan, len(allKinds) - 1},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			rule, ok := tpl.Kinds[tc.parent]
			if !ok {
				t.Fatalf("Kinds[%q] missing", tc.parent)
			}
			if got := len(rule.AllowedChildKinds); got != tc.wantAllowListLen {
				t.Fatalf("Kinds[%q].AllowedChildKinds len = %d; want %d (explicit allow-list of all kinds except %q)", tc.parent, got, tc.wantAllowListLen, tc.prohibitedChild)
			}
			if slices.Contains(rule.AllowedChildKinds, tc.prohibitedChild) {
				t.Fatalf("Kinds[%q].AllowedChildKinds contains %q; want exclusion (reverse-hierarchy prohibition)", tc.parent, tc.prohibitedChild)
			}
		})
	}
}
