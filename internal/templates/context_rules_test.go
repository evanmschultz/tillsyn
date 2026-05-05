package templates

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/evanmschultz/tillsyn/internal/domain"
)

// TestLoadAgentBindingContextHappyPath verifies a build binding with a
// fully-populated `[context]` block decodes cleanly and every field lands on
// the AgentBinding.Context sub-struct verbatim. Pins the master PLAN.md L13
// "bounded mode" shape: parent + parent_git_diff + ancestors_by_kind=["plan"]
// + delivery="file" + max_chars=50000 + max_rule_duration="500ms".
func TestLoadAgentBindingContextHappyPath(t *testing.T) {
	src := `
schema_version = "v1"

[agent_bindings.build]
agent_name = "go-builder-agent"
model = "opus"

[agent_bindings.build.context]
parent = true
parent_git_diff = true
ancestors_by_kind = ["plan"]
delivery = "file"
max_chars = 50000
max_rule_duration = "500ms"
`
	tpl, err := Load(strings.NewReader(src))
	if err != nil {
		t.Fatalf("Load: unexpected error: %v", err)
	}
	binding, ok := tpl.AgentBindings[domain.KindBuild]
	if !ok {
		t.Fatalf("AgentBindings[%q] missing", domain.KindBuild)
	}
	ctx := binding.Context
	if !ctx.Parent {
		t.Fatalf("Context.Parent = false; want true")
	}
	if !ctx.ParentGitDiff {
		t.Fatalf("Context.ParentGitDiff = false; want true")
	}
	if len(ctx.AncestorsByKind) != 1 || ctx.AncestorsByKind[0] != domain.KindPlan {
		t.Fatalf("Context.AncestorsByKind = %v; want [%q]", ctx.AncestorsByKind, domain.KindPlan)
	}
	if ctx.Delivery != ContextDeliveryFile {
		t.Fatalf("Context.Delivery = %q; want %q", ctx.Delivery, ContextDeliveryFile)
	}
	if ctx.MaxChars != 50000 {
		t.Fatalf("Context.MaxChars = %d; want 50000", ctx.MaxChars)
	}
	if got := time.Duration(ctx.MaxRuleDuration); got != 500*time.Millisecond {
		t.Fatalf("Context.MaxRuleDuration = %s; want 500ms", got)
	}
}

// TestLoadAgentBindingContextEmptyTablePresent verifies a binding that
// declares the `[context]` table heading but omits every nested key loads
// cleanly — the AgentBinding.Context sub-struct decodes to its zero value
// and the validator's "if non-zero must be positive" rules pass trivially.
//
// Distinct from the omitted-table case below: pelletier/go-toml/v2 will
// emit the `[context]` table marker into the AST regardless of nested-key
// presence; this test pins the contract that an empty-but-present table is
// equivalent to absence at the validation layer.
func TestLoadAgentBindingContextEmptyTablePresent(t *testing.T) {
	src := `
schema_version = "v1"

[agent_bindings.build]
agent_name = "go-builder-agent"
model = "opus"

[agent_bindings.build.context]
`
	tpl, err := Load(strings.NewReader(src))
	if err != nil {
		t.Fatalf("Load: unexpected error: %v", err)
	}
	binding, ok := tpl.AgentBindings[domain.KindBuild]
	if !ok {
		t.Fatalf("AgentBindings[%q] missing", domain.KindBuild)
	}
	assertContextRulesZero(t, binding.Context)
}

// TestLoadAgentBindingContextOmittedAltogether verifies a binding without
// any `[context]` table loads cleanly and the AgentBinding.Context
// sub-struct is the zero value — the master PLAN.md L13 "fully-agentic
// mode" path. This is the expected default for adopters who do not opt
// into bounded pre-staging.
func TestLoadAgentBindingContextOmittedAltogether(t *testing.T) {
	src := `
schema_version = "v1"

[agent_bindings.build]
agent_name = "go-builder-agent"
model = "opus"
`
	tpl, err := Load(strings.NewReader(src))
	if err != nil {
		t.Fatalf("Load: unexpected error: %v", err)
	}
	binding, ok := tpl.AgentBindings[domain.KindBuild]
	if !ok {
		t.Fatalf("AgentBindings[%q] missing", domain.KindBuild)
	}
	assertContextRulesZero(t, binding.Context)
}

// assertContextRulesZero fails the test when ctx is not equivalent to the
// ContextRules zero value. Direct struct equality (`ctx == ContextRules{}`)
// does not compile because ContextRules carries slices; this helper does a
// field-by-field comparison instead and treats nil-vs-empty-slice asymmetry
// (which pelletier/go-toml/v2 can introduce) as equivalent for the
// "zero-or-effectively-zero" contract this droplet tests.
func assertContextRulesZero(t *testing.T, ctx ContextRules) {
	t.Helper()
	if ctx.Parent {
		t.Fatalf("Context.Parent = true; want false")
	}
	if ctx.ParentGitDiff {
		t.Fatalf("Context.ParentGitDiff = true; want false")
	}
	if len(ctx.SiblingsByKind) != 0 {
		t.Fatalf("Context.SiblingsByKind = %v; want empty", ctx.SiblingsByKind)
	}
	if len(ctx.AncestorsByKind) != 0 {
		t.Fatalf("Context.AncestorsByKind = %v; want empty", ctx.AncestorsByKind)
	}
	if len(ctx.DescendantsByKind) != 0 {
		t.Fatalf("Context.DescendantsByKind = %v; want empty", ctx.DescendantsByKind)
	}
	if ctx.Delivery != "" {
		t.Fatalf("Context.Delivery = %q; want empty", ctx.Delivery)
	}
	if ctx.MaxChars != 0 {
		t.Fatalf("Context.MaxChars = %d; want 0", ctx.MaxChars)
	}
	if time.Duration(ctx.MaxRuleDuration) != 0 {
		t.Fatalf("Context.MaxRuleDuration = %s; want 0", time.Duration(ctx.MaxRuleDuration))
	}
}

// TestLoadAgentBindingContextRejectsInvalidDelivery verifies the closed-enum
// delivery vocabulary {"", "inline", "file"}: any other value (e.g. "stream",
// "Inline", " file ") MUST surface as ErrInvalidContextRules at Load time.
// Mirror's the IsValidGateKind exact-match rationale — silent case-fold
// matching would mask "Inline" / "FILE" typos at load time, well before the
// dispatcher's aggregator engine fires.
func TestLoadAgentBindingContextRejectsInvalidDelivery(t *testing.T) {
	src := `
schema_version = "v1"

[agent_bindings.build]
agent_name = "go-builder-agent"
model = "opus"

[agent_bindings.build.context]
delivery = "stream"
`
	_, err := Load(strings.NewReader(src))
	if err == nil {
		t.Fatalf("Load: expected ErrInvalidContextRules; got nil")
	}
	if !errors.Is(err, ErrInvalidContextRules) {
		t.Fatalf("Load: errors.Is(_, ErrInvalidContextRules) = false; err = %v", err)
	}
	// Sentinel chain: ErrInvalidContextRules wraps ErrInvalidAgentBinding so
	// callers using the umbrella sentinel route correctly.
	if !errors.Is(err, ErrInvalidAgentBinding) {
		t.Fatalf("Load: errors.Is(_, ErrInvalidAgentBinding) = false; err = %v", err)
	}
	if !strings.Contains(err.Error(), `"stream"`) {
		t.Fatalf("Load: err = %q; want offending value %q in message", err.Error(), "stream")
	}
}

// TestLoadAgentBindingContextRejectsNegativeMaxChars verifies the validator
// rejects a negative max_chars value with ErrInvalidContextRules. Zero is
// legal at the schema layer (engine-time default-substitution); only strictly
// negative values fail.
func TestLoadAgentBindingContextRejectsNegativeMaxChars(t *testing.T) {
	src := `
schema_version = "v1"

[agent_bindings.build]
agent_name = "go-builder-agent"
model = "opus"

[agent_bindings.build.context]
max_chars = -1
`
	_, err := Load(strings.NewReader(src))
	if err == nil {
		t.Fatalf("Load: expected ErrInvalidContextRules; got nil")
	}
	if !errors.Is(err, ErrInvalidContextRules) {
		t.Fatalf("Load: errors.Is(_, ErrInvalidContextRules) = false; err = %v", err)
	}
	if !strings.Contains(err.Error(), "max_chars") {
		t.Fatalf("Load: err = %q; want substring %q", err.Error(), "max_chars")
	}
}

// TestLoadAgentBindingContextRejectsNegativeMaxRuleDuration verifies the
// validator rejects a negative max_rule_duration with ErrInvalidContextRules.
// Zero is legal (engine-time default-substitution to 500ms in F.7.18.4); only
// strictly negative durations fail.
//
// Implementation note: the spawn-prompt's test scenario list includes
// `max_rule_duration = "0s"` as a "MUST fail load" case, but that conflicts
// with both (a) the spawn-prompt's validators rule "if non-zero, must be
// positive" and (b) the empty-`[context]` happy-path test which decodes to a
// zero-valued MaxRuleDuration. Resolved per F.7.18 plan body L96 + the
// non-contradictory reading: zero is legal and means "use bundle-global
// default at engine-time"; only `"-1s"` (and any other negative duration)
// is rejected. Documented in the F.7.18.1 builder worklog.
func TestLoadAgentBindingContextRejectsNegativeMaxRuleDuration(t *testing.T) {
	src := `
schema_version = "v1"

[agent_bindings.build]
agent_name = "go-builder-agent"
model = "opus"

[agent_bindings.build.context]
max_rule_duration = "-1s"
`
	_, err := Load(strings.NewReader(src))
	if err == nil {
		t.Fatalf("Load: expected ErrInvalidContextRules; got nil")
	}
	if !errors.Is(err, ErrInvalidContextRules) {
		t.Fatalf("Load: errors.Is(_, ErrInvalidContextRules) = false; err = %v", err)
	}
	if !strings.Contains(err.Error(), "max_rule_duration") {
		t.Fatalf("Load: err = %q; want substring %q", err.Error(), "max_rule_duration")
	}
}

// TestLoadAgentBindingContextRejectsInvalidKindReference verifies a
// kind-walk slice (siblings_by_kind / ancestors_by_kind / descendants_by_kind)
// containing an entry that is not a member of the closed 12-value
// domain.Kind enum surfaces as ErrUnknownKindReference. Consistent with the
// existing kinds-map / child-rules / agent-bindings-map vocabulary checks —
// the context sub-struct's kind references use the same sentinel.
func TestLoadAgentBindingContextRejectsInvalidKindReference(t *testing.T) {
	src := `
schema_version = "v1"

[agent_bindings.build]
agent_name = "go-builder-agent"
model = "opus"

[agent_bindings.build.context]
siblings_by_kind = ["bogus_kind"]
`
	_, err := Load(strings.NewReader(src))
	if err == nil {
		t.Fatalf("Load: expected ErrUnknownKindReference; got nil")
	}
	if !errors.Is(err, ErrUnknownKindReference) {
		t.Fatalf("Load: errors.Is(_, ErrUnknownKindReference) = false; err = %v", err)
	}
	if !strings.Contains(err.Error(), "siblings_by_kind") {
		t.Fatalf("Load: err = %q; want substring %q", err.Error(), "siblings_by_kind")
	}
	if !strings.Contains(err.Error(), "bogus_kind") {
		t.Fatalf("Load: err = %q; want offending kind %q in message", err.Error(), "bogus_kind")
	}
}

// TestLoadAgentBindingContextAllowsDescendantsOnPlanKind verifies the
// "template authors trusted" flexibility framing from master PLAN.md L13:
// a `kind=plan` binding declaring `descendants_by_kind = ["build"]` MUST
// load cleanly. Round-history fix-planners + tree-pruners legitimately walk
// down the cascade subtree from a plan parent; the schema does NOT reject
// this on principle. Drop 4c F.7.18.1 acceptance: an explicit allow-test
// exists so a future tightening of the validator (e.g. "planners walk up,
// not down") cannot land silently.
func TestLoadAgentBindingContextAllowsDescendantsOnPlanKind(t *testing.T) {
	src := `
schema_version = "v1"

[agent_bindings.plan]
agent_name = "go-planning-agent"
model = "opus"

[agent_bindings.plan.context]
parent = true
descendants_by_kind = ["build", "plan"]
delivery = "file"
`
	tpl, err := Load(strings.NewReader(src))
	if err != nil {
		t.Fatalf("Load: unexpected error: %v (descendants on kind=plan must be allowed per master PLAN L13 flexibility)", err)
	}
	binding, ok := tpl.AgentBindings[domain.KindPlan]
	if !ok {
		t.Fatalf("AgentBindings[%q] missing", domain.KindPlan)
	}
	want := []domain.Kind{domain.KindBuild, domain.KindPlan}
	if len(binding.Context.DescendantsByKind) != len(want) {
		t.Fatalf("Context.DescendantsByKind = %v; want %v", binding.Context.DescendantsByKind, want)
	}
	for i, k := range want {
		if binding.Context.DescendantsByKind[i] != k {
			t.Fatalf("Context.DescendantsByKind[%d] = %q; want %q", i, binding.Context.DescendantsByKind[i], k)
		}
	}
}

// TestLoadAgentBindingContextStrictDecodeRejectsUnknownKey verifies the
// strict-decode chain (load.go step 3) rejects unknown keys nested under
// `[agent_bindings.<kind>.context]` as ErrUnknownTemplateKey at Load time.
// Proves that closed-struct unknown-key rejection — which depends on every
// ContextRules field carrying an explicit TOML tag — actually fires for the
// new sub-struct. Without this regression bar, a future refactor that drops
// a TOML tag would silently relax strict decode.
func TestLoadAgentBindingContextStrictDecodeRejectsUnknownKey(t *testing.T) {
	src := `
schema_version = "v1"

[agent_bindings.build]
agent_name = "go-builder-agent"
model = "opus"

[agent_bindings.build.context]
bogus_field = true
`
	_, err := Load(strings.NewReader(src))
	if err == nil {
		t.Fatalf("Load: expected ErrUnknownTemplateKey; got nil")
	}
	if !errors.Is(err, ErrUnknownTemplateKey) {
		t.Fatalf("Load: errors.Is(_, ErrUnknownTemplateKey) = false; err = %v", err)
	}
	if !strings.Contains(err.Error(), "bogus_field") {
		t.Fatalf("Load: err = %q; want offending field %q in message", err.Error(), "bogus_field")
	}
}
