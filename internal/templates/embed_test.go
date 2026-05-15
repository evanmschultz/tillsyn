package templates

import (
	"errors"
	"io"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/evanmschultz/tillsyn/internal/domain"
)

// loadDefaultOrFatal is the shared helper for embed tests asserting on the
// GO-flavored embedded template. Centralizing the load invocation keeps each
// test focused on its assertion and gives a single failure point if the
// embed pipeline regresses (e.g. the //go:embed directive falls out of
// sync with the on-disk file path).
//
// Drop 4c.5 droplet F.1.3 SEMANTIC SHIFT: prior to F.1.3 this helper called
// `LoadDefaultTemplate()` which read default-go.toml directly. Post-F.1.3
// `LoadDefaultTemplate()` is a thin wrapper around
// `LoadDefaultTemplateForLanguage("")` and returns the language-AGNOSTIC
// generic template (zero agent_bindings, no gates, no context blocks).
// The catalog-shape assertions in this file (agent bindings cover 12
// kinds, gates carry mage_ci/commit/push, context blocks for plan/build/
// QA kinds, etc.) all target the GO template specifically, so the helper
// now invokes `LoadDefaultTemplateForLanguage("go")` explicitly. Tests
// asserting on the generic template use `loadGenericOrFatal`.
func loadDefaultOrFatal(t *testing.T) Template {
	t.Helper()
	tpl, err := LoadDefaultTemplateForLanguage("go")
	if err != nil {
		t.Fatalf("LoadDefaultTemplateForLanguage(\"go\"): unexpected error: %v", err)
	}
	return tpl
}

// TestDefaultTemplateGoLoadsCleanly verifies the embedded
// builtin/till-go.toml parses + validates without error. Any sentinel
// from load.go (unknown key, schema-version mismatch, unknown kind
// reference, child-rule cycle) would surface here, so this is the canary
// for the whole embed pipeline. Renamed from `TestDefaultTemplateLoadsCleanly`
// in Drop 4c.5 droplet F.2.1 alongside the `default.toml` → `default-go.toml`
// file rebadge; rewired in Drop 4c.5 droplet F.1.3 to call
// `LoadDefaultTemplateForLanguage("go")` directly because the
// `LoadDefaultTemplate()` wrapper now resolves to the generic template.
// Drop 4c.6 W5.D1 rebadged the file a second time, `default-go.toml` →
// `till-go.toml`, to align with the `till-` prefix family; the test
// function name is intentionally retained to keep the caller-audit
// footprint of W5.D1 minimal.
func TestDefaultTemplateGoLoadsCleanly(t *testing.T) {
	t.Parallel()

	tpl, err := LoadDefaultTemplateForLanguage("go")
	if err != nil {
		t.Fatalf("LoadDefaultTemplateForLanguage(\"go\"): unexpected error: %v", err)
	}
	if tpl.SchemaVersion != SchemaVersionV1 {
		t.Fatalf("SchemaVersion = %q; want %q", tpl.SchemaVersion, SchemaVersionV1)
	}
}

// TestLoadDefaultGenericTemplate is the canary for the language-agnostic
// builtin shipped in Drop 4c.5 droplet F.2.2 and rebadged in Drop 4c.6
// W5.D2 from `default-generic.toml` to `till-gen.toml`. It verifies that
// builtin/till-gen.toml:
//
//  1. Opens cleanly from the embed.FS (the //go:embed directive on
//     DefaultTemplateFS extends to both files in F.2.2).
//  2. Parses + validates through the full templates.Load() chain (every
//     load.go sentinel — unknown key, schema-version mismatch, unknown kind
//     reference, child-rule cycle, agent-binding-tool-gating — would
//     surface here).
//  3. Carries the closed 12-kind catalog (same vocabulary as till-go).
//  4. Carries exactly four standard child_rules: build→build-qa-proof,
//     build→build-qa-falsification, plan→plan-qa-proof,
//     plan→plan-qa-falsification. The two drop-narrowed entries
//     (DROP-PLAN-QA-PROOF, DROP-PLAN-QA-FALSIFICATION) that till-go.toml
//     ships are INTENTIONALLY OMITTED — drop-level cascade is
//     Tillsyn-runtime-specific scaffolding, not language-agnostic shape.
//     Per F.2.2 acceptance criterion #4 + the corresponding test scenario.
//  5. Carries the same six STEWARD persistent-parent seeds as till-go
//     (DISCUSSIONS, HYLLA_FINDINGS, LEDGER, WIKI_CHANGELOG, REFINEMENTS,
//     HYLLA_REFINEMENTS) — STEWARD coordination scaffolding is
//     language-agnostic.
//  6. Has ZERO agent_bindings — `len(tpl.AgentBindings) == 0`. Per F.2.2
//     acceptance criterion #2 + falsification mitigations F1+F2+F3, the
//     generic template intentionally OMITS [agent_bindings] entirely.
//     Adopters declare bindings in their project-local
//     <project_root>/.tillsyn/template.toml.
//
// Drop 4c.5 droplet F.1.3 (later in Theme F's chain) will land
// `LoadDefaultTemplateForLanguage("")` which selects this file via the
// resolver. Until then this test exercises the file via a direct embed.FS
// open + Load() pass — proving the file ships and parses cleanly without
// pre-shipping the F.1.3 entry point.
func TestLoadDefaultGenericTemplate(t *testing.T) {
	t.Parallel()

	f, err := DefaultTemplateFS.Open("builtin/till-gen.toml")
	if err != nil {
		t.Fatalf("DefaultTemplateFS.Open(till-gen.toml): unexpected error: %v", err)
	}
	defer f.Close()

	tpl, err := Load(f)
	if err != nil {
		t.Fatalf("Load(till-gen.toml): unexpected error: %v", err)
	}

	if tpl.SchemaVersion != SchemaVersionV1 {
		t.Fatalf("SchemaVersion = %q; want %q", tpl.SchemaVersion, SchemaVersionV1)
	}

	// Closed 12-kind catalog — same vocabulary as till-go.
	if got, want := len(tpl.Kinds), len(allKinds); got != want {
		t.Fatalf("len(Kinds) = %d; want %d (closed 12-kind catalog)", got, want)
	}
	for _, kind := range allKinds {
		if _, ok := tpl.Kinds[kind]; !ok {
			t.Fatalf("Kinds[%q] missing — every closed-12-kind must have a [kinds.<kind>] section", kind)
		}
	}

	// Exactly four standard child_rules — drop-narrowed entries omitted.
	if got, want := len(tpl.ChildRules), 4; got != want {
		t.Fatalf("len(ChildRules) = %d; want %d (four standard rules; drop-narrowed entries intentionally omitted)", got, want)
	}
	wantChildRuleEdges := map[string]bool{
		"build->build-qa-proof":         false,
		"build->build-qa-falsification": false,
		"plan->plan-qa-proof":           false,
		"plan->plan-qa-falsification":   false,
	}
	for _, rule := range tpl.ChildRules {
		// Drop-narrowed entries (when_parent_structural_type set) are
		// explicitly forbidden in the generic file.
		if rule.WhenParentStructuralType != "" {
			t.Fatalf("ChildRules carries drop-narrowed entry (when_parent_structural_type=%q); generic must omit drop-narrowed scaffolding", rule.WhenParentStructuralType)
		}
		edge := string(rule.WhenParentKind) + "->" + string(rule.CreateChildKind)
		if _, expected := wantChildRuleEdges[edge]; !expected {
			t.Fatalf("ChildRules carries unexpected edge %q; generic ships only the four standard rules", edge)
		}
		wantChildRuleEdges[edge] = true
	}
	for edge, seen := range wantChildRuleEdges {
		if !seen {
			t.Fatalf("ChildRules missing expected edge %q", edge)
		}
	}

	// Six STEWARD seeds — same coordination scaffold as till-go.
	if got, want := len(tpl.StewardSeeds), 6; got != want {
		t.Fatalf("len(StewardSeeds) = %d; want %d (DISCUSSIONS / HYLLA_FINDINGS / LEDGER / WIKI_CHANGELOG / REFINEMENTS / HYLLA_REFINEMENTS)", got, want)
	}
	wantSeedTitles := map[string]bool{
		"DISCUSSIONS":       false,
		"HYLLA_FINDINGS":    false,
		"LEDGER":            false,
		"WIKI_CHANGELOG":    false,
		"REFINEMENTS":       false,
		"HYLLA_REFINEMENTS": false,
	}
	for _, seed := range tpl.StewardSeeds {
		if _, expected := wantSeedTitles[seed.Title]; !expected {
			t.Fatalf("StewardSeeds carries unexpected title %q", seed.Title)
		}
		wantSeedTitles[seed.Title] = true
	}
	for title, seen := range wantSeedTitles {
		if !seen {
			t.Fatalf("StewardSeeds missing expected title %q", title)
		}
	}

	// Zero agent_bindings — the load-bearing showcase contract.
	if got := len(tpl.AgentBindings); got != 0 {
		t.Fatalf("len(AgentBindings) = %d; want 0 (generic template intentionally omits [agent_bindings] table)", got)
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
// cycle validator rejects (see comment in till-go.toml). The drop-orch
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
// till-go.toml against the hand-coded fixtureTemplate() in nesting_test.go
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
				t.Fatalf("loaded till-go.toml AllowsNesting(%q, %q) = true; fixture rejects — prohibition set drifted", parent, child)
			}
			if loadedReason == "" {
				t.Fatalf("loaded till-go.toml AllowsNesting(%q, %q) reason empty; fixture rejects with non-empty reason", parent, child)
			}
		}
	}
}

// TestDefaultTemplateStewardOwnedKinds verifies the kinds that PLAN.md
// § 15.7 (§ 19.3 bullet 9) names as STEWARD-owned have owner = "STEWARD"
// in their KindRule. The auth gate at internal/adapters/mcp_common/
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

// TestDefaultTemplateLoadsWithGates asserts the embedded till-go.toml
// decodes the [gates] section with the Drop 4c F.7.16 shape:
// [gates.build] = ["mage_ci", "commit", "push"]. Drop 4b Wave A 4b.1 originally
// shipped only ["mage_ci"]; Drop 4c F.7.16 expanded the sequence per master
// PLAN.md L20 — commit + push gates ship in the LIST but are INDEPENDENTLY
// GATED via ProjectMetadata.DispatcherCommitEnabled / DispatcherPushEnabled,
// which both default OFF (nil/false). Slice ORDER is load-bearing because the
// gate runner halts on first failure: mage_ci must run before commit (a green
// build is a precondition for committing the work) and commit must run before
// push (push without a fresh local commit on the working ref is a no-op or a
// stale-state push).
//
// Other kinds (plan-qa-proof, build-qa-proof, closeout, etc.) are ABSENT from
// [gates.*]; the gate runner treats absence as "no gates" not "all gates" per
// the 4b.2 doc-comment.
func TestDefaultTemplateLoadsWithGates(t *testing.T) {
	t.Parallel()

	tpl := loadDefaultOrFatal(t)

	gateSeq, ok := tpl.Gates[domain.KindBuild]
	if !ok {
		t.Fatalf("Gates[%q] missing — Drop 4c F.7.16 ships [gates.build] = [\"mage_ci\", \"commit\", \"push\"]", domain.KindBuild)
	}
	want := []GateKind{GateKindMageCI, GateKindCommit, GateKindPush}
	if !slices.Equal(gateSeq, want) {
		t.Fatalf("Gates[%q] = %v; want %v (Drop 4c F.7.16 — order is load-bearing: mage_ci then commit then push)", domain.KindBuild, gateSeq, want)
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
			t.Fatalf("Gates[%q] should be absent — only build carries a gate sequence in the default template", kind)
		}
	}
}

// TestDefaultTemplateGatesAllValidGateKinds asserts every entry in the
// loaded [gates.build] sequence is a member of the closed GateKind enum
// per IsValidGateKind. Drop 4c F.7.16 acceptance bullet #2: "Default loads
// + validates clean (closed-enum gate kinds all valid post-F.7.13/14)."
//
// Regression guard against two distinct failure modes:
//  1. Someone adds a new string to [gates.build] in till-go.toml without
//     also extending the closed GateKind enum + validGateKinds in schema.go.
//  2. Someone removes a GateKind constant in schema.go without checking
//     that no template TOML still references it.
//
// Both modes would silently let an unknown gate kind survive load-time
// validation if the template were authored before Drop 4b's validateGateKinds
// chain was wired up. This test pins the post-F.7.16 invariant: every gate
// the default template names is reachable by the gate runner's lookup.
func TestDefaultTemplateGatesAllValidGateKinds(t *testing.T) {
	t.Parallel()

	tpl := loadDefaultOrFatal(t)
	for kind, seq := range tpl.Gates {
		for i, gk := range seq {
			if !IsValidGateKind(gk) {
				t.Fatalf("Gates[%q][%d] = %q; IsValidGateKind returned false (closed-enum violation)", kind, i, gk)
			}
		}
	}
}

// TestDefaultTemplateNoProjectMetadataOverrides asserts the act of loading
// the default template does NOT alter the project-metadata dispatcher-toggle
// defaults — IsDispatcherCommitEnabled() and IsDispatcherPushEnabled() both
// remain false on a zero-value ProjectMetadata. This pins master PLAN.md L20:
// commit and push gates are LISTED in [gates.build] but each is GATED OFF by
// default via project-metadata toggles. Adopter flips the toggle per project;
// no template re-bake required.
//
// The test exists as a structural invariant — the Template type carries no
// project-metadata-shaped fields, so loading it cannot produce overrides. A
// future drop that adds template-side toggle defaults (e.g.
// `[project_metadata]` sub-table) would have to break this test before
// shipping, forcing the toggle-default contract to be re-confirmed.
func TestDefaultTemplateNoProjectMetadataOverrides(t *testing.T) {
	t.Parallel()

	_ = loadDefaultOrFatal(t)

	// A fresh ProjectMetadata{} (zero-valued) must report both dispatcher
	// toggles as disabled. Loading the default template above is the
	// regression hook — if a future change adds template-driven defaults
	// that mutate project-metadata zero values, this assertion would need
	// to be re-derived from the loaded template state.
	var meta domain.ProjectMetadata
	if meta.IsDispatcherCommitEnabled() {
		t.Fatalf("IsDispatcherCommitEnabled() = true on zero ProjectMetadata; want false (master PLAN L20 default-OFF)")
	}
	if meta.IsDispatcherPushEnabled() {
		t.Fatalf("IsDispatcherPushEnabled() = true on zero ProjectMetadata; want false (master PLAN L20 default-OFF)")
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

// contextSeededKinds names the six kinds the F.7.18.5 default-template seed
// populates with an `[agent_bindings.<kind>.context]` block. The remaining
// six kinds (research, closeout, commit, refinement, discussion,
// human-verify) intentionally have a zero-value Context per the F.7.18.5
// plan + master PLAN L13 FLEXIBLE-not-REQUIRED framing. The test
// TestDefaultTemplateNonContextSeededKindsHaveZeroContext below pins that
// half of the contract.
var contextSeededKinds = []domain.Kind{
	domain.KindPlan,
	domain.KindBuild,
	domain.KindPlanQAProof,
	domain.KindPlanQAFalsification,
	domain.KindBuildQAProof,
	domain.KindBuildQAFalsification,
}

// TestDefaultTemplateBuildContextSeedsParentGitDiff asserts the default-template
// build binding declares `parent_git_diff = true` so the dispatcher's
// aggregator engine pre-stages the parent's diff for the builder agent.
//
// REV-4 contract: ONLY the build binding gets parent_git_diff in the default
// seed. The four QA bindings have NO parent_git_diff rule — see the negative
// assertions below. F.7.18.5 acceptance: builder lens reduces redundant tool
// calls during implementation.
func TestDefaultTemplateBuildContextSeedsParentGitDiff(t *testing.T) {
	t.Parallel()

	tpl := loadDefaultOrFatal(t)
	binding, ok := tpl.AgentBindings[domain.KindBuild]
	if !ok {
		t.Fatalf("AgentBindings[%q] missing", domain.KindBuild)
	}
	if !binding.Context.ParentGitDiff {
		t.Fatalf("AgentBindings[build].Context.ParentGitDiff = false; want true (REV-4 builder lens)")
	}
}

// TestDefaultTemplateQABindingsRejectParentGitDiff is the REV-4 regression
// guard. Per F.7.18 REV-4 the four QA bindings (build-qa-proof,
// build-qa-falsification, plan-qa-proof, plan-qa-falsification) MUST NOT
// pre-stage `parent_git_diff` — independent verification is load-bearing for
// cascade-on-itself trustworthiness.
//
// The test runs as a subtest per binding so a regression on any one binding
// surfaces with a precise failure name. ContextRules.ParentGitDiff is a Go
// bool; the zero value is false so omitting the field in TOML and explicitly
// setting `parent_git_diff = false` are equivalent at this assertion.
func TestDefaultTemplateQABindingsRejectParentGitDiff(t *testing.T) {
	t.Parallel()

	tpl := loadDefaultOrFatal(t)
	qaKinds := []domain.Kind{
		domain.KindBuildQAProof,
		domain.KindBuildQAFalsification,
		domain.KindPlanQAProof,
		domain.KindPlanQAFalsification,
	}
	for _, kind := range qaKinds {
		t.Run(string(kind), func(t *testing.T) {
			t.Parallel()
			binding, ok := tpl.AgentBindings[kind]
			if !ok {
				t.Fatalf("AgentBindings[%q] missing", kind)
			}
			if binding.Context.ParentGitDiff {
				t.Fatalf("AgentBindings[%q].Context.ParentGitDiff = true; want false (REV-4 — QA must verify independently)", kind)
			}
		})
	}
}

// TestDefaultTemplateContextSeedsAncestorsByKind asserts every context-seeded
// binding declares `ancestors_by_kind = ["plan"]`. The walk lets the spawned
// agent see its enclosing plan ancestor regardless of how deeply nested the
// action item sits in the cascade subtree.
func TestDefaultTemplateContextSeedsAncestorsByKind(t *testing.T) {
	t.Parallel()

	tpl := loadDefaultOrFatal(t)
	for _, kind := range contextSeededKinds {
		t.Run(string(kind), func(t *testing.T) {
			t.Parallel()
			binding, ok := tpl.AgentBindings[kind]
			if !ok {
				t.Fatalf("AgentBindings[%q] missing", kind)
			}
			got := binding.Context.AncestorsByKind
			if len(got) != 1 || got[0] != domain.KindPlan {
				t.Fatalf("AgentBindings[%q].Context.AncestorsByKind = %v; want [%q]", kind, got, domain.KindPlan)
			}
		})
	}
}

// TestDefaultTemplateContextSeedsDelivery asserts every context-seeded binding
// declares `delivery = "file"`. The default seed renders pre-staged context
// into `<bundle>/context/<rule>.md` files the agent loads on demand via the
// Read tool — distinct from `inline` which appends to system-append.md.
func TestDefaultTemplateContextSeedsDelivery(t *testing.T) {
	t.Parallel()

	tpl := loadDefaultOrFatal(t)
	for _, kind := range contextSeededKinds {
		t.Run(string(kind), func(t *testing.T) {
			t.Parallel()
			binding, ok := tpl.AgentBindings[kind]
			if !ok {
				t.Fatalf("AgentBindings[%q] missing", kind)
			}
			if binding.Context.Delivery != ContextDeliveryFile {
				t.Fatalf("AgentBindings[%q].Context.Delivery = %q; want %q", kind, binding.Context.Delivery, ContextDeliveryFile)
			}
		})
	}
}

// TestDefaultTemplateContextSeedsCaps asserts every context-seeded binding
// declares `max_chars = 50000` and `max_rule_duration = "500ms"`. The
// per-rule caps localize truncation + timeouts to a single rule before the
// bundle-global caps under [tillsyn] consider skipping.
func TestDefaultTemplateContextSeedsCaps(t *testing.T) {
	t.Parallel()

	tpl := loadDefaultOrFatal(t)
	const wantMaxChars = 50000
	const wantMaxRuleDuration = 500 * time.Millisecond
	for _, kind := range contextSeededKinds {
		t.Run(string(kind), func(t *testing.T) {
			t.Parallel()
			binding, ok := tpl.AgentBindings[kind]
			if !ok {
				t.Fatalf("AgentBindings[%q] missing", kind)
			}
			if binding.Context.MaxChars != wantMaxChars {
				t.Fatalf("AgentBindings[%q].Context.MaxChars = %d; want %d", kind, binding.Context.MaxChars, wantMaxChars)
			}
			got := time.Duration(binding.Context.MaxRuleDuration)
			if got != wantMaxRuleDuration {
				t.Fatalf("AgentBindings[%q].Context.MaxRuleDuration = %s; want %s", kind, got, wantMaxRuleDuration)
			}
		})
	}
}

// TestDefaultTemplateContextSeedsParentTrue asserts every context-seeded
// binding sets `parent = true`. The aggregator's `parent` rule renders the
// parent action-item's identity + description into the spawn bundle so the
// agent has the immediate cascade context without needing a separate MCP
// call. Companion to the AncestorsByKind / Delivery / Caps tests above.
func TestDefaultTemplateContextSeedsParentTrue(t *testing.T) {
	t.Parallel()

	tpl := loadDefaultOrFatal(t)
	for _, kind := range contextSeededKinds {
		t.Run(string(kind), func(t *testing.T) {
			t.Parallel()
			binding, ok := tpl.AgentBindings[kind]
			if !ok {
				t.Fatalf("AgentBindings[%q] missing", kind)
			}
			if !binding.Context.Parent {
				t.Fatalf("AgentBindings[%q].Context.Parent = false; want true", kind)
			}
		})
	}
}

// TestDefaultTemplateNonContextSeededKindsHaveZeroContext asserts the six
// kinds NOT in contextSeededKinds (research, closeout, commit, refinement,
// discussion, human-verify) carry a zero-value Context — the master PLAN L13
// "fully-agentic mode" path. F.7.18.5 acceptance: scope creep guard — only
// the six SKETCH-named bindings get a default seed; adopters override per
// project for the rest.
func TestDefaultTemplateNonContextSeededKindsHaveZeroContext(t *testing.T) {
	t.Parallel()

	tpl := loadDefaultOrFatal(t)
	seeded := make(map[domain.Kind]bool, len(contextSeededKinds))
	for _, kind := range contextSeededKinds {
		seeded[kind] = true
	}
	for _, kind := range allKinds {
		if seeded[kind] {
			continue
		}
		t.Run(string(kind), func(t *testing.T) {
			t.Parallel()
			binding, ok := tpl.AgentBindings[kind]
			if !ok {
				t.Fatalf("AgentBindings[%q] missing", kind)
			}
			ctx := binding.Context
			if ctx.Parent {
				t.Fatalf("AgentBindings[%q].Context.Parent = true; want false (no [context] block in default for non-seeded kind)", kind)
			}
			if ctx.ParentGitDiff {
				t.Fatalf("AgentBindings[%q].Context.ParentGitDiff = true; want false", kind)
			}
			if len(ctx.AncestorsByKind) != 0 {
				t.Fatalf("AgentBindings[%q].Context.AncestorsByKind = %v; want empty", kind, ctx.AncestorsByKind)
			}
			if len(ctx.SiblingsByKind) != 0 {
				t.Fatalf("AgentBindings[%q].Context.SiblingsByKind = %v; want empty", kind, ctx.SiblingsByKind)
			}
			if len(ctx.DescendantsByKind) != 0 {
				t.Fatalf("AgentBindings[%q].Context.DescendantsByKind = %v; want empty", kind, ctx.DescendantsByKind)
			}
			if ctx.Delivery != "" {
				t.Fatalf("AgentBindings[%q].Context.Delivery = %q; want empty", kind, ctx.Delivery)
			}
			if ctx.MaxChars != 0 {
				t.Fatalf("AgentBindings[%q].Context.MaxChars = %d; want 0", kind, ctx.MaxChars)
			}
			if time.Duration(ctx.MaxRuleDuration) != 0 {
				t.Fatalf("AgentBindings[%q].Context.MaxRuleDuration = %s; want 0", kind, time.Duration(ctx.MaxRuleDuration))
			}
		})
	}
}

// TestDefaultTemplatePlanContextHasNoDescendants asserts the default-seed
// plan binding does NOT declare `descendants_by_kind` — default planners
// walk UP only. The schema (per F.7.18.1) accepts the field; the default
// just doesn't seed it. F.7.18.5 acceptance: planner-flexibility cross-check
// (master PLAN L13 A-λ) — adopters who want fix-planner / tree-pruner
// behavior add the field themselves in their project template.
func TestDefaultTemplatePlanContextHasNoDescendants(t *testing.T) {
	t.Parallel()

	tpl := loadDefaultOrFatal(t)
	binding, ok := tpl.AgentBindings[domain.KindPlan]
	if !ok {
		t.Fatalf("AgentBindings[%q] missing", domain.KindPlan)
	}
	if len(binding.Context.DescendantsByKind) != 0 {
		t.Fatalf("AgentBindings[plan].Context.DescendantsByKind = %v; want empty (default planners walk UP only — adopter opt-in)", binding.Context.DescendantsByKind)
	}
}

// TestLoadDefaultTemplateForLanguage_Generic asserts that the empty-string
// language axis (the closed-enum zero value for `domain.Project.Language`)
// resolves to `builtin/till-gen.toml` (rebadged from `default-generic.toml`
// in Drop 4c.6 W5.D2) and parses cleanly through the full validation chain.
//
// Drop 4c.5 droplet F.1.3 acceptance criterion #2 + #8. Mirrors the
// generic-template content asserts in TestLoadDefaultGenericTemplate (F.2.2)
// but exercises the resolver entry point rather than a direct embed.FS open.
// The two together pin the resolver-to-content path: F.2.2's test asserts
// the file content; this test asserts the resolver routes lang="" to that
// file.
func TestLoadDefaultTemplateForLanguage_Generic(t *testing.T) {
	t.Parallel()

	tpl, err := LoadDefaultTemplateForLanguage("")
	if err != nil {
		t.Fatalf("LoadDefaultTemplateForLanguage(\"\"): unexpected error: %v", err)
	}
	if tpl.SchemaVersion != SchemaVersionV1 {
		t.Fatalf("SchemaVersion = %q; want %q", tpl.SchemaVersion, SchemaVersionV1)
	}

	// Generic template's load-bearing distinguishing feature vs till-go:
	// zero agent_bindings (per F.2.2 acceptance criterion #2). If the
	// resolver mistakenly routed lang="" to till-go.toml this assertion
	// would fail because till-go ships 12 agent bindings.
	if got := len(tpl.AgentBindings); got != 0 {
		t.Fatalf("len(AgentBindings) = %d; want 0 (lang=\"\" must route to till-gen.toml; till-go ships 12 bindings)", got)
	}
}

// TestLoadDefaultTemplateForLanguage_Go asserts that the `"go"` language
// axis (the only currently-shipping non-empty closed-enum value per the
// Q1 deferral of FE) resolves to `builtin/till-go.toml` and parses
// cleanly through the full validation chain.
//
// Drop 4c.5 droplet F.1.3 acceptance criterion #3 + #8. The
// content-shape canary across the resolver entry point: the Go template
// is the catalog the dispatcher binds during pre-MVP dogfooding, so any
// regression in the resolver-to-Go-file routing immediately surfaces.
//
// The Go-distinguishing assertion is the 12 agent bindings — the
// generic file ships zero, till-go ships 12. The bindings count is
// thus the cleanest discriminator without baking content drift into the
// test.
func TestLoadDefaultTemplateForLanguage_Go(t *testing.T) {
	t.Parallel()

	tpl, err := LoadDefaultTemplateForLanguage("go")
	if err != nil {
		t.Fatalf("LoadDefaultTemplateForLanguage(\"go\"): unexpected error: %v", err)
	}
	if tpl.SchemaVersion != SchemaVersionV1 {
		t.Fatalf("SchemaVersion = %q; want %q", tpl.SchemaVersion, SchemaVersionV1)
	}

	// Go template's load-bearing distinguishing feature vs generic: 12
	// agent bindings (one per closed-enum kind). If the resolver
	// mistakenly routed lang="go" to till-gen.toml this
	// assertion would fail.
	if got, want := len(tpl.AgentBindings), len(allKinds); got != want {
		t.Fatalf("len(AgentBindings) = %d; want %d (lang=\"go\" must route to till-go.toml; generic ships 0 bindings)", got, want)
	}
}

// TestLoadDefaultTemplateForLanguage_FESupported asserts the `"fe"` axis
// successfully loads `builtin/till-fe.toml` and returns a valid Template.
//
// Drop 4c.6.1 W4.D2 resolves the Q1 deferral from
// workflow/drop_4c_5/THEME_F_PLAN.md §3 Note 5: the FE group now ships
// `builtin/till-fe.toml` alongside the W4.D1 agent scaffold. This test
// replaces the prior `TestLoadDefaultTemplateForLanguage_FERejected` from
// Drop 4c.5 F.1.3. The FE-distinguishing assertion is the 12 agent bindings
// (same count as till-go.toml) — generic till-gen.toml ships zero.
func TestLoadDefaultTemplateForLanguage_FESupported(t *testing.T) {
	t.Parallel()

	tpl, err := LoadDefaultTemplateForLanguage("fe")
	if err != nil {
		t.Fatalf("LoadDefaultTemplateForLanguage(\"fe\"): unexpected error: %v", err)
	}
	if tpl.SchemaVersion != SchemaVersionV1 {
		t.Fatalf("SchemaVersion = %q; want %q", tpl.SchemaVersion, SchemaVersionV1)
	}

	// FE template ships 12 agent bindings (one per closed-enum kind) — same
	// count as till-go.toml. If the resolver mistakenly routed lang="fe" to
	// till-gen.toml this assertion would fail because till-gen ships 0 bindings.
	if got, want := len(tpl.AgentBindings), len(allKinds); got != want {
		t.Fatalf("len(AgentBindings) = %d; want %d (lang=\"fe\" must route to till-fe.toml; generic ships 0 bindings)", got, want)
	}
}

// TestLoadDefaultTemplateFEResolves is the canary for the FE builtin shipped
// in Drop 4c.6.1 W4.D2. It verifies that builtin/till-fe.toml:
//
//  1. Opens cleanly from the embed.FS (the //go:embed directive extends
//     to include till-fe.toml in W4.D2).
//  2. Parses + validates through the full templates.Load() chain.
//  3. Carries the closed 12-kind catalog (same vocabulary as till-go).
//  4. Carries exactly six standard + drop-narrowed child_rules: the four
//     standard rules (build->build-qa-{proof,falsification},
//     plan->plan-qa-{proof,falsification}) plus the two drop-narrowed
//     entries (DROP-PLAN-QA-PROOF, DROP-PLAN-QA-FALSIFICATION). Mirrors
//     till-go.toml's shape.
//  5. Carries the same six STEWARD persistent-parent seeds as till-go.
//  6. Has 12 agent_bindings — one per closed-enum kind.
func TestLoadDefaultTemplateFEResolves(t *testing.T) {
	t.Parallel()

	f, err := DefaultTemplateFS.Open("builtin/till-fe.toml")
	if err != nil {
		t.Fatalf("DefaultTemplateFS.Open(till-fe.toml): unexpected error: %v", err)
	}
	defer f.Close()

	tpl, err := Load(f)
	if err != nil {
		t.Fatalf("Load(till-fe.toml): unexpected error: %v", err)
	}

	if tpl.SchemaVersion != SchemaVersionV1 {
		t.Fatalf("SchemaVersion = %q; want %q", tpl.SchemaVersion, SchemaVersionV1)
	}

	// Closed 12-kind catalog — same vocabulary as till-go.
	if got, want := len(tpl.Kinds), len(allKinds); got != want {
		t.Fatalf("len(Kinds) = %d; want %d (closed 12-kind catalog)", got, want)
	}
	for _, kind := range allKinds {
		if _, ok := tpl.Kinds[kind]; !ok {
			t.Fatalf("Kinds[%q] missing — every closed-12-kind must have a [kinds.<kind>] section", kind)
		}
	}

	// Six child_rules: four standard + two drop-narrowed. Mirrors till-go.toml.
	if got, want := len(tpl.ChildRules), 6; got != want {
		t.Fatalf("len(ChildRules) = %d; want %d (four standard + two drop-narrowed rules, mirrors till-go.toml)", got, want)
	}

	// Six STEWARD seeds — same coordination scaffold as till-go.
	if got, want := len(tpl.StewardSeeds), 6; got != want {
		t.Fatalf("len(StewardSeeds) = %d; want %d (DISCUSSIONS / HYLLA_FINDINGS / LEDGER / WIKI_CHANGELOG / REFINEMENTS / HYLLA_REFINEMENTS)", got, want)
	}

	// 12 agent_bindings — one per kind.
	if got, want := len(tpl.AgentBindings), len(allKinds); got != want {
		t.Fatalf("len(AgentBindings) = %d; want %d (FE template ships one binding per closed-enum kind)", got, want)
	}
}

// TestLoadDefaultTemplateForLanguage_UnknownRejected asserts an axis value
// outside the closed `domain.Project.Language` enum (the test uses
// `"rust"` as the canonical "obviously not yet supported" value) returns
// an error wrapping `ErrLanguageNotSupported` with the offending value
// in the message.
//
// Drop 4c.5 droplet F.1.3 acceptance criterion #5 + #8. The closed-enum
// drift guard: a hand-rolled DB or a future drop that adds a new
// `domain.Project.Language` value WITHOUT extending the resolver must
// fail loud rather than silently returning the Go default. The sentinel
// is the routing contract; the message carries the offending value.
func TestLoadDefaultTemplateForLanguage_UnknownRejected(t *testing.T) {
	t.Parallel()

	tpl, err := LoadDefaultTemplateForLanguage("rust")
	if err == nil {
		t.Fatalf("LoadDefaultTemplateForLanguage(\"rust\"): err = nil; want wrapped ErrLanguageNotSupported")
	}
	if !errors.Is(err, ErrLanguageNotSupported) {
		t.Fatalf("LoadDefaultTemplateForLanguage(\"rust\"): err %v not errors.Is(ErrLanguageNotSupported); closed-enum drift guard broken", err)
	}
	if got := err.Error(); !strings.Contains(got, `"rust"`) {
		t.Fatalf("LoadDefaultTemplateForLanguage(\"rust\"): error message = %q; want to contain offending lang value `\"rust\"`", got)
	}
	if tpl.SchemaVersion != "" || len(tpl.Kinds) != 0 {
		t.Fatalf("LoadDefaultTemplateForLanguage(\"rust\"): returned non-zero Template = %+v; want zero value on rejection", tpl)
	}
}

// TestLoadBuiltinTemplate_TillGo asserts that LoadBuiltinTemplate("till-go")
// returns a Template deep-equal to LoadDefaultTemplateForLanguage("go").
// This pins the name-axis → language-axis equivalence for the Go builtin:
// the two entry points must resolve to the same embedded TOML, so any
// divergence in path wiring surfaces here immediately.
func TestLoadBuiltinTemplate_TillGo(t *testing.T) {
	t.Parallel()

	got, err := LoadBuiltinTemplate("till-go")
	if err != nil {
		t.Fatalf("LoadBuiltinTemplate(\"till-go\"): unexpected error: %v", err)
	}

	want, err := LoadDefaultTemplateForLanguage("go")
	if err != nil {
		t.Fatalf("LoadDefaultTemplateForLanguage(\"go\"): unexpected error: %v", err)
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("LoadBuiltinTemplate(\"till-go\") != LoadDefaultTemplateForLanguage(\"go\"); name→language path wiring diverged")
	}
}

// TestLoadBuiltinTemplate_TillGen asserts that LoadBuiltinTemplate("till-gen")
// returns a Template deep-equal to LoadDefaultTemplateForLanguage("") (the
// language-agnostic generic template). Pins the name-axis → language-axis
// equivalence for the generic builtin.
func TestLoadBuiltinTemplate_TillGen(t *testing.T) {
	t.Parallel()

	got, err := LoadBuiltinTemplate("till-gen")
	if err != nil {
		t.Fatalf("LoadBuiltinTemplate(\"till-gen\"): unexpected error: %v", err)
	}

	want, err := LoadDefaultTemplateForLanguage("")
	if err != nil {
		t.Fatalf("LoadDefaultTemplateForLanguage(\"\"): unexpected error: %v", err)
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("LoadBuiltinTemplate(\"till-gen\") != LoadDefaultTemplateForLanguage(\"\"); name→language path wiring diverged")
	}
}

// TestLoadBuiltinTemplate_TillFE asserts that LoadBuiltinTemplate("till-fe")
// returns a Template deep-equal to LoadDefaultTemplateForLanguage("fe").
// Pins the name-axis → language-axis equivalence for the FE builtin (shipped
// in Drop 4c.6.1 W4.D2).
func TestLoadBuiltinTemplate_TillFE(t *testing.T) {
	t.Parallel()

	got, err := LoadBuiltinTemplate("till-fe")
	if err != nil {
		t.Fatalf("LoadBuiltinTemplate(\"till-fe\"): unexpected error: %v", err)
	}

	want, err := LoadDefaultTemplateForLanguage("fe")
	if err != nil {
		t.Fatalf("LoadDefaultTemplateForLanguage(\"fe\"): unexpected error: %v", err)
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("LoadBuiltinTemplate(\"till-fe\") != LoadDefaultTemplateForLanguage(\"fe\"); name→language path wiring diverged")
	}
}

// TestLoadBuiltinTemplate_UnknownRejected asserts that a name outside the
// closed builtin-name list (e.g. "rust") returns an error wrapping
// ErrBuiltinNotFound, includes the offending name in the error message, and
// returns a zero-value Template. Mirrors the closed-enum drift guard of
// TestLoadDefaultTemplateForLanguage_UnknownRejected for the name-axis API.
func TestLoadBuiltinTemplate_UnknownRejected(t *testing.T) {
	t.Parallel()

	tpl, err := LoadBuiltinTemplate("rust")
	if err == nil {
		t.Fatalf("LoadBuiltinTemplate(\"rust\"): err = nil; want wrapped ErrBuiltinNotFound")
	}
	if !errors.Is(err, ErrBuiltinNotFound) {
		t.Fatalf("LoadBuiltinTemplate(\"rust\"): err %v not errors.Is(ErrBuiltinNotFound); closed-name drift guard broken", err)
	}
	if got := err.Error(); !strings.Contains(got, `"rust"`) {
		t.Fatalf("LoadBuiltinTemplate(\"rust\"): error message = %q; want to contain offending name value `\"rust\"`", got)
	}
	if tpl.SchemaVersion != "" || len(tpl.Kinds) != 0 {
		t.Fatalf("LoadBuiltinTemplate(\"rust\"): returned non-zero Template = %+v; want zero value on rejection", tpl)
	}
}

// TestLoadDefaultTemplate_WrapsLanguageEmpty asserts the thin-wrapper
// contract: `LoadDefaultTemplate()` returns the SAME Template (deep-equal)
// as `LoadDefaultTemplateForLanguage("")`. Drop 4c.5 droplet F.1.3
// acceptance criterion #6 — the cross-test that pins the wrapper
// semantic. Re-affirmed by Drop 4c.5 droplet F.2.4 acceptance criterion #3
// + table-driven scenario "LoadDefaultTemplate() returns same as
// LoadDefaultTemplateForLanguage(\"\")": F.2.4's caller-audit redirected
// every PRODUCTION call to `LoadDefaultTemplateForLanguage(project.Language)`,
// but the thin wrapper is preserved for callers that intentionally want
// the language-AGNOSTIC generic template (none in production today; tests
// may still reach for it). This deep-equal assertion is the contract gate
// that lets future drops trust the equivalence.
//
// SEMANTIC SHIFT regression net: pre-F.1.3 `LoadDefaultTemplate()` read
// default-go.toml directly. Post-F.1.3 it routes to till-gen.toml
// (rebadged from default-generic.toml in Drop 4c.6 W5.D2) via
// `LoadDefaultTemplateForLanguage("")`. Future drops that touch the
// wrapper or the resolver must keep these two call paths in sync;
// reflect.DeepEqual is the strict invariant.
func TestLoadDefaultTemplate_WrapsLanguageEmpty(t *testing.T) {
	t.Parallel()

	wrapped, err := LoadDefaultTemplate()
	if err != nil {
		t.Fatalf("LoadDefaultTemplate(): unexpected error: %v", err)
	}
	direct, err := LoadDefaultTemplateForLanguage("")
	if err != nil {
		t.Fatalf("LoadDefaultTemplateForLanguage(\"\"): unexpected error: %v", err)
	}
	if !reflect.DeepEqual(wrapped, direct) {
		t.Fatalf("LoadDefaultTemplate() != LoadDefaultTemplateForLanguage(\"\"); wrapper-equality contract broken\nwrapped = %+v\ndirect  = %+v", wrapped, direct)
	}
}

// w4d1StandardAgentNames is the closed list of ten standard agent file names
// shipped under each canonical group directory (`go/`, `gen/`, `fe/`) by
// Drop 4c.6.1 W4.D1. The 10-file set supersedes the Drop 4c.6 W1.D1
// 7-file set: the monolithic `qa-proof-agent.md` and `qa-falsification-agent.md`
// are split into `plan-qa-proof-agent.md`, `build-qa-proof-agent.md`,
// `plan-qa-falsification-agent.md`, `build-qa-falsification-agent.md`, and
// `orchestrator-managed.md` is added. Drop 4c.8 W4 lands substantive content;
// these files are PLACEHOLDER scaffolding only until then.
var w4d1StandardAgentNames = []string{
	"planning-agent.md",
	"builder-agent.md",
	"plan-qa-proof-agent.md",
	"build-qa-proof-agent.md",
	"plan-qa-falsification-agent.md",
	"build-qa-falsification-agent.md",
	"research-agent.md",
	"closeout-agent.md",
	"commit-message-agent.md",
	"orchestrator-managed.md",
}

// w4d1CanonicalGroups is the closed list of three canonical group directories
// shipped by Drop 4c.6.1 W4.D1 under `internal/templates/builtin/agents/`.
// Each group ships the same ten standard agent names (w4d1StandardAgentNames).
// `gen` is the language-agnostic generic group (renamed from `till-gen`);
// `go` is Go+mage tuning (renamed from `till-go`); `fe` is the new FE group
// added in W4.D1. `till-gdd` is NOT in this list — it is a template-family
// identifier (not a group) and retains its 7-file shape unchanged.
var w4d1CanonicalGroups = []string{"gen", "go", "fe"}

// w4d1TillGDDAgentNames is the closed list of seven agent file names in the
// `till-gdd` template-family directory. `till-gdd` is NOT a group (the
// canonical groups are gen/go/fe); it ships the original 7-file shape from
// Drop 4c.6 W1.D1 and is NOT expanded to 10 files in W4.D1.
var w4d1TillGDDAgentNames = []string{
	"planning-agent.md",
	"builder-agent.md",
	"qa-proof-agent.md",
	"qa-falsification-agent.md",
	"research-agent.md",
	"closeout-agent.md",
	"commit-message-agent.md",
}

// TestDefaultTemplateFSEmbedsPlaceholderAgentFiles asserts every Drop 4c.6.1
// W4.D1 canonical agent file path resolves via `DefaultTemplateFS.Open` AND
// every agent .md body contains the literal string "PLACEHOLDER" so a builder
// mistakenly committing a stub prompt cannot pass embedded-FS introspection
// silently. Mirrors the F.2.1 falsification mitigation #2 pattern (explicit
// per-file list, never glob).
//
// Updated from the Drop 4c.6 W1.D1 version (21 paths = 3 groups × 7 names)
// to the W4.D1 version (37 paths = 3 canonical groups × 10 names + till-gdd × 7):
//   - Canonical groups (`gen`, `go`, `fe`): 10 files each = 30 files.
//   - `till-gdd` template-family: 7 files (unchanged from W1.D1).
//   - Total agent files: 37. Plus `agents.example.toml` = 38 distinct paths.
//
// Substantive prompt content for the agent files lands in Drop 4c.8 W4; the
// only contract this test enforces is (a) the embed.FS opens the file and
// (b) the body carries the PLACEHOLDER marker so accidental drift surfaces.
func TestDefaultTemplateFSEmbedsPlaceholderAgentFiles(t *testing.T) {
	t.Parallel()

	// 10-file canonical groups: gen/, go/, fe/.
	for _, group := range w4d1CanonicalGroups {
		for _, name := range w4d1StandardAgentNames {
			path := "builtin/agents/" + group + "/" + name
			t.Run(path, func(t *testing.T) {
				t.Parallel()
				f, err := DefaultTemplateFS.Open(path)
				if err != nil {
					t.Fatalf("DefaultTemplateFS.Open(%q): unexpected error: %v", path, err)
				}
				defer f.Close()
				body, err := io.ReadAll(f)
				if err != nil {
					t.Fatalf("io.ReadAll(%q): unexpected error: %v", path, err)
				}
				if !strings.Contains(string(body), "PLACEHOLDER") {
					t.Fatalf("agent file %q body missing required \"PLACEHOLDER\" marker; W4.D1 placeholder discipline (substantive content lands Drop 4c.8 W4)", path)
				}
			})
		}
	}

	// till-gdd template-family: 7-file shape unchanged from Drop 4c.6 W1.D1.
	// till-gdd is a template-family identifier (not a group); it is NOT
	// expanded to the 10-file standard in W4.D1.
	for _, name := range w4d1TillGDDAgentNames {
		path := "builtin/agents/till-gdd/" + name
		t.Run(path, func(t *testing.T) {
			t.Parallel()
			f, err := DefaultTemplateFS.Open(path)
			if err != nil {
				t.Fatalf("DefaultTemplateFS.Open(%q): unexpected error: %v", path, err)
			}
			defer f.Close()
			body, err := io.ReadAll(f)
			if err != nil {
				t.Fatalf("io.ReadAll(%q): unexpected error: %v", path, err)
			}
			if !strings.Contains(string(body), "PLACEHOLDER") {
				t.Fatalf("agent file %q body missing required \"PLACEHOLDER\" marker; W1.D1 placeholder discipline (substantive content lands Drop 4c.8 W4)", path)
			}
		})
	}

	// agents.example.toml is the runtime-config example shipped at
	// internal/templates/builtin/agents.example.toml per W1.D1 acceptance
	// bullet #2 + SKETCH.md § 4.2 (sane Anthropic-direct defaults). This
	// test only asserts the file resolves via embed.FS; semantic
	// correctness (parses cleanly through the W0 loader) is verified by
	// W0's loader tests once W0 lands. W1.D1 deliberately ships the
	// fixture without exercising the loader to avoid the chicken/egg.
	t.Run("agents.example.toml", func(t *testing.T) {
		t.Parallel()
		f, err := DefaultTemplateFS.Open("builtin/agents.example.toml")
		if err != nil {
			t.Fatalf("DefaultTemplateFS.Open(\"builtin/agents.example.toml\"): unexpected error: %v", err)
		}
		defer f.Close()
		body, err := io.ReadAll(f)
		if err != nil {
			t.Fatalf("io.ReadAll(\"builtin/agents.example.toml\"): unexpected error: %v", err)
		}
		if len(body) == 0 {
			t.Fatalf("agents.example.toml body empty; W1.D1 must ship a non-empty fixture")
		}
	})
}
