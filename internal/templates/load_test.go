package templates

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/evanmschultz/tillsyn/internal/domain"
)

// TestLoadValidTemplate verifies the happy-path: a well-formed TOML stream
// with all four top-level tables decodes into a Template whose fields are
// populated, and no validator fires.
func TestLoadValidTemplate(t *testing.T) {
	src := `
schema_version = "v1"

[kinds.build]
owner = "STEWARD"
allowed_parent_kinds = ["plan"]
allowed_child_kinds = ["build-qa-proof", "build-qa-falsification"]
structural_type = "droplet"

[[child_rules]]
when_parent_kind = "build"
create_child_kind = "build-qa-proof"
title = "BUILD-QA-PROOF"
blocked_by_parent = true

[[child_rules]]
when_parent_kind = "build"
create_child_kind = "build-qa-falsification"
title = "BUILD-QA-FALSIFICATION"
blocked_by_parent = true

[agent_bindings.build]
agent_name = "go-builder-agent"
model = "opus"
`
	tpl, err := Load(strings.NewReader(src))
	if err != nil {
		t.Fatalf("Load: unexpected error: %v", err)
	}
	if tpl.SchemaVersion != SchemaVersionV1 {
		t.Fatalf("SchemaVersion = %q; want %q", tpl.SchemaVersion, SchemaVersionV1)
	}
	if _, ok := tpl.Kinds[domain.KindBuild]; !ok {
		t.Fatalf("Kinds[%q] missing", domain.KindBuild)
	}
	if got, want := len(tpl.ChildRules), 2; got != want {
		t.Fatalf("len(ChildRules) = %d; want %d", got, want)
	}
	if _, ok := tpl.AgentBindings[domain.KindBuild]; !ok {
		t.Fatalf("AgentBindings[%q] missing", domain.KindBuild)
	}
}

// TestLoadGateRulesReservedTable verifies the L6 forward-compat hatch: a
// document with a populated [gate_rules] table decodes cleanly and the
// content lands on Template.GateRulesRaw without triggering
// ErrUnknownTemplateKey.
func TestLoadGateRulesReservedTable(t *testing.T) {
	src := `
schema_version = "v1"

[gate_rules.mage_ci]
mage_target = "ci"
required = true
`
	tpl, err := Load(strings.NewReader(src))
	if err != nil {
		t.Fatalf("Load: unexpected error: %v", err)
	}
	if tpl.GateRulesRaw == nil {
		t.Fatalf("GateRulesRaw nil; want populated map")
	}
	mageCI, ok := tpl.GateRulesRaw["mage_ci"]
	if !ok {
		t.Fatalf("GateRulesRaw[%q] missing", "mage_ci")
	}
	if mageCI == nil {
		t.Fatalf("GateRulesRaw[%q] = nil; want non-nil map value", "mage_ci")
	}
}

// TestLoadRejectionTable covers every rejection path documented in droplet
// 3.9's acceptance list: unknown top-level keys, cycle detection, unknown
// kind references, missing schema_version, wrong schema_version with the
// pre-pass UX path, and malformed TOML.
func TestLoadRejectionTable(t *testing.T) {
	tests := []struct {
		name         string
		src          string
		wantSentinel error
		wantSubstr   string // substring search in err.Error() for UX assertion; empty = skip
	}{
		{
			name: "unknown top-level key rejected",
			src: `
schema_version = "v1"

[bogus_table]
foo = "bar"
`,
			wantSentinel: ErrUnknownTemplateKey,
		},
		{
			name: "cycle build->plan->build rejected",
			src: `
schema_version = "v1"

[[child_rules]]
when_parent_kind = "build"
create_child_kind = "plan"
title = "BOGUS-1"

[[child_rules]]
when_parent_kind = "plan"
create_child_kind = "build"
title = "BOGUS-2"
`,
			wantSentinel: ErrTemplateCycle,
			wantSubstr:   "build",
		},
		{
			name: "unreachable when_parent_kind surfaces as unknown-kind",
			src: `
schema_version = "v1"

[[child_rules]]
when_parent_kind = "no-such-kind"
create_child_kind = "build"
title = "BOGUS"
`,
			wantSentinel: ErrUnknownKindReference,
			wantSubstr:   "when_parent_kind",
		},
		{
			name: "unknown create_child_kind rejected",
			src: `
schema_version = "v1"

[[child_rules]]
when_parent_kind = "build"
create_child_kind = "bogus"
title = "BOGUS"
`,
			wantSentinel: ErrUnknownKindReference,
			wantSubstr:   "create_child_kind",
		},
		{
			name: "missing schema_version rejected",
			src: `
[kinds.build]
owner = "STEWARD"
`,
			wantSentinel: ErrUnsupportedSchemaVersion,
			wantSubstr:   `""`,
		},
		{
			name: "wrong schema_version v2 carries actual value in UX",
			src: `
schema_version = "v2"

[kinds.build]
owner = "STEWARD"
`,
			wantSentinel: ErrUnsupportedSchemaVersion,
			wantSubstr:   `"v2"`,
		},
		{
			name: "schema_version pre-pass beats unknown-key pass for v2 + bogus table",
			src: `
schema_version = "v2"

[bogus_top_level]
junk = 1
`,
			wantSentinel: ErrUnsupportedSchemaVersion,
			wantSubstr:   `"v2"`,
		},
		{
			name:         "malformed toml unbalanced bracket rejected",
			src:          "schema_version = \"v1\"\n[unbalanced\n",
			wantSentinel: nil, // pre-pass returns underlying parser error; assert non-sentinel branch
			wantSubstr:   "templates: parse:",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Load(strings.NewReader(tc.src))
			if err == nil {
				t.Fatalf("Load: expected error; got nil")
			}
			if tc.wantSentinel != nil && !errors.Is(err, tc.wantSentinel) {
				t.Fatalf("Load: errors.Is(_, %v) = false; err = %v", tc.wantSentinel, err)
			}
			if tc.wantSubstr != "" && !strings.Contains(err.Error(), tc.wantSubstr) {
				t.Fatalf("Load: err = %q; want substring %q", err.Error(), tc.wantSubstr)
			}
		})
	}
}

// TestLoadNilReader verifies the nil-reader guard returns a non-panicking
// error rather than letting io.ReadAll panic on a nil io.Reader.
func TestLoadNilReader(t *testing.T) {
	_, err := Load(nil)
	if err == nil {
		t.Fatalf("Load(nil): expected error; got nil")
	}
}

// TestLoadRejectsBogusKindsMapKey verifies validateMapKeys rejects a
// [kinds.<bogus>] table whose map key is not a member of the closed 12-value
// domain.Kind enum. Strict decode cannot catch this: TOML treats arbitrary
// keys under [kinds.*] as legitimate map entries, so a typo like
// [kinds.totally-bogus] survives strict decode and must be caught by the
// dedicated map-key validator at load time.
func TestLoadRejectsBogusKindsMapKey(t *testing.T) {
	src := `
schema_version = "v1"

[kinds.totally-bogus]
owner = "STEWARD"
allowed_parent_kinds = ["plan"]
allowed_child_kinds = []
structural_type = "droplet"
`
	_, err := Load(strings.NewReader(src))
	if err == nil {
		t.Fatalf("Load: expected ErrUnknownKindReference; got nil")
	}
	if !errors.Is(err, ErrUnknownKindReference) {
		t.Fatalf("Load: errors.Is(_, ErrUnknownKindReference) = false; err = %v", err)
	}
	if !strings.Contains(err.Error(), "kinds map key") {
		t.Fatalf("Load: err = %q; want substring %q", err.Error(), "kinds map key")
	}
	if !strings.Contains(err.Error(), "totally-bogus") {
		t.Fatalf("Load: err = %q; want offending key %q in message", err.Error(), "totally-bogus")
	}
}

// TestLoadRejectsBogusAgentBindingsMapKey verifies validateMapKeys rejects an
// [agent_bindings.<bogus>] table whose map key is not a member of the closed
// 12-value domain.Kind enum. Same rationale as TestLoadRejectsBogusKindsMapKey
// but for the agent_bindings map.
func TestLoadRejectsBogusAgentBindingsMapKey(t *testing.T) {
	src := `
schema_version = "v1"

[agent_bindings.totally-bogus]
agent_name = "go-builder-agent"
model = "opus"
`
	_, err := Load(strings.NewReader(src))
	if err == nil {
		t.Fatalf("Load: expected ErrUnknownKindReference; got nil")
	}
	if !errors.Is(err, ErrUnknownKindReference) {
		t.Fatalf("Load: errors.Is(_, ErrUnknownKindReference) = false; err = %v", err)
	}
	if !strings.Contains(err.Error(), "agent_bindings map key") {
		t.Fatalf("Load: err = %q; want substring %q", err.Error(), "agent_bindings map key")
	}
	if !strings.Contains(err.Error(), "totally-bogus") {
		t.Fatalf("Load: err = %q; want offending key %q in message", err.Error(), "totally-bogus")
	}
}

// TestTemplateGatesAndGateRulesCoexist verifies the [gates] and [gate_rules]
// TOML keys decode independently. The Drop 4b Wave A `gates` field is
// distinct from the Drop 3 reserved-but-untyped `gate_rules` map; a template
// with both populated must load cleanly with each landing on its own field.
//
// Mitigates falsification attack A2 from WAVE_A_PLAN.md 4b.1: "The reserved
// [gate_rules] table from Drop 3 conflicts with this [gates] table." It does
// not — the TOML keys are different and the strict decoder treats them as
// separate fields.
func TestTemplateGatesAndGateRulesCoexist(t *testing.T) {
	src := `
schema_version = "v1"

[gates]
build = ["mage_ci"]

[gate_rules.mage_ci]
mage_target = "ci"
required = true
`
	tpl, err := Load(strings.NewReader(src))
	if err != nil {
		t.Fatalf("Load: unexpected error: %v", err)
	}

	gateSeq, ok := tpl.Gates[domain.KindBuild]
	if !ok {
		t.Fatalf("Gates[%q] missing", domain.KindBuild)
	}
	if len(gateSeq) != 1 || gateSeq[0] != GateKindMageCI {
		t.Fatalf("Gates[%q] = %v; want [%q]", domain.KindBuild, gateSeq, GateKindMageCI)
	}

	if tpl.GateRulesRaw == nil {
		t.Fatalf("GateRulesRaw nil; want populated map (forward-compat seam preserved)")
	}
	if _, ok := tpl.GateRulesRaw["mage_ci"]; !ok {
		t.Fatalf("GateRulesRaw[%q] missing — forward-compat seam should still decode", "mage_ci")
	}
}

// TestValidateGateKindsRejectsUnknownKind verifies validateGateKinds rejects
// a template whose [gates.<kind>] value-slice carries a gate-kind string that
// is not a member of the closed GateKind enum. The error wraps
// ErrUnknownGateKind and names the offending parent kind plus the offending
// gate value for UX.
func TestValidateGateKindsRejectsUnknownKind(t *testing.T) {
	src := `
schema_version = "v1"

[gates]
build = ["bogus"]
`
	_, err := Load(strings.NewReader(src))
	if err == nil {
		t.Fatalf("Load: expected ErrUnknownGateKind; got nil")
	}
	if !errors.Is(err, ErrUnknownGateKind) {
		t.Fatalf("Load: errors.Is(_, ErrUnknownGateKind) = false; err = %v", err)
	}
	if !strings.Contains(err.Error(), `"build"`) {
		t.Fatalf("Load: err = %q; want offending parent kind %q in message", err.Error(), "build")
	}
	if !strings.Contains(err.Error(), `"bogus"`) {
		t.Fatalf("Load: err = %q; want offending gate value %q in message", err.Error(), "bogus")
	}
}

// TestValidateGateKindsRejectsUnknownParentKind verifies validateMapKeys
// rejects a [gates.<bogus-kind>] table whose map key is not a member of the
// closed 12-value domain.Kind enum. Mirrors the existing kinds/agent_bindings
// map-key checks — strict decode treats arbitrary keys under [gates.*] as
// legitimate map entries, so a typo like [gates.bogus_kind] survives strict
// decode and must be caught by the dedicated map-key validator at load time.
func TestValidateGateKindsRejectsUnknownParentKind(t *testing.T) {
	src := `
schema_version = "v1"

[gates]
bogus_kind = ["mage_ci"]
`
	_, err := Load(strings.NewReader(src))
	if err == nil {
		t.Fatalf("Load: expected ErrUnknownKindReference; got nil")
	}
	if !errors.Is(err, ErrUnknownKindReference) {
		t.Fatalf("Load: errors.Is(_, ErrUnknownKindReference) = false; err = %v", err)
	}
	if !strings.Contains(err.Error(), "gates map key") {
		t.Fatalf("Load: err = %q; want substring %q", err.Error(), "gates map key")
	}
	if !strings.Contains(err.Error(), "bogus_kind") {
		t.Fatalf("Load: err = %q; want offending key %q in message", err.Error(), "bogus_kind")
	}
}

// TestTemplateGatesEmptyMapDecodes verifies that a template TOML document
// without a [gates] table loads cleanly and Template.Gates decodes to its
// zero value (nil map). The cascade gate runner treats a nil Gates map — and
// any absent per-kind entry — as "no gates," so the nil-vs-empty distinction
// is not load-bearing for runtime semantics; this test pins the
// zero-value-on-absence contract so future schema edits cannot silently
// change it (e.g. by initialising Gates to a non-nil empty map at load
// time, which would defeat the explicit-presence test in
// TestDefaultTemplateLoadsWithGates).
func TestTemplateGatesEmptyMapDecodes(t *testing.T) {
	src := `
schema_version = "v1"

[kinds.build]
owner = "STEWARD"
allowed_parent_kinds = ["plan"]
allowed_child_kinds = []
structural_type = "droplet"
`
	tpl, err := Load(strings.NewReader(src))
	if err != nil {
		t.Fatalf("Load: unexpected error: %v", err)
	}
	if tpl.Gates != nil {
		t.Fatalf("Gates = %v; want nil (zero-value on absent [gates] table)", tpl.Gates)
	}
}

// TestLoadSelfCycleSingleRule verifies a self-loop child_rule (A -> A) is
// detected as ErrTemplateCycle. A self-loop is the smallest-possible cycle
// and exercises the gray-color branch of the DFS without relying on
// multi-rule choreography.
func TestLoadSelfCycleSingleRule(t *testing.T) {
	src := `
schema_version = "v1"

[[child_rules]]
when_parent_kind = "build"
create_child_kind = "build"
title = "SELF-LOOP"
`
	_, err := Load(strings.NewReader(src))
	if err == nil {
		t.Fatalf("Load: expected ErrTemplateCycle; got nil")
	}
	if !errors.Is(err, ErrTemplateCycle) {
		t.Fatalf("Load: errors.Is(_, ErrTemplateCycle) = false; err = %v", err)
	}
	if !strings.Contains(err.Error(), "build -> build") {
		t.Fatalf("Load: err = %q; want cycle path %q", err.Error(), "build -> build")
	}
}

// TestLoadAgentBindingEnvAndCLIKindHappyPath verifies a Template TOML stream
// declaring an [agent_bindings.<kind>] row with the new Drop 4c F.7.17.1
// fields (`env` and `cli_kind`) decodes cleanly: every entry in the env
// allow-list is preserved verbatim, the cli_kind value lands on the field,
// and validateAgentBindingEnvNames does not fire. Mixed-case env names
// (uppercase HTTP_PROXY + lowercase https_proxy) ride the same binding so
// the test pins the L5 + REV-2 lowercase-allowed contract.
func TestLoadAgentBindingEnvAndCLIKindHappyPath(t *testing.T) {
	src := `
schema_version = "v1"

[agent_bindings.build]
agent_name = "go-builder-agent"
model = "opus"
env = ["ANTHROPIC_API_KEY", "https_proxy", "HTTP_PROXY"]
cli_kind = "claude"
`
	tpl, err := Load(strings.NewReader(src))
	if err != nil {
		t.Fatalf("Load: unexpected error: %v", err)
	}
	binding, ok := tpl.AgentBindings[domain.KindBuild]
	if !ok {
		t.Fatalf("AgentBindings[%q] missing", domain.KindBuild)
	}
	wantEnv := []string{"ANTHROPIC_API_KEY", "https_proxy", "HTTP_PROXY"}
	if len(binding.Env) != len(wantEnv) {
		t.Fatalf("binding.Env = %v; want %v", binding.Env, wantEnv)
	}
	for i, want := range wantEnv {
		if binding.Env[i] != want {
			t.Fatalf("binding.Env[%d] = %q; want %q", i, binding.Env[i], want)
		}
	}
	if binding.CLIKind != "claude" {
		t.Fatalf("binding.CLIKind = %q; want %q", binding.CLIKind, "claude")
	}
}

// TestLoadAgentBindingCLIKindOmittedDefaultsToEmpty verifies that omitting
// `cli_kind` from a TOML binding leaves AgentBinding.CLIKind at the empty
// string. Per Drop 4c F.7.17 locked decision L15 the empty string is the
// "back-compat default → claude" sentinel handled at adapter-lookup time
// (NOT at template Load time), so this droplet just verifies the field is
// settable AND zero-valued on absence.
func TestLoadAgentBindingCLIKindOmittedDefaultsToEmpty(t *testing.T) {
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
	if binding.CLIKind != "" {
		t.Fatalf("binding.CLIKind = %q; want empty string (omitted cli_kind sentinel)", binding.CLIKind)
	}
	if binding.Env != nil {
		t.Fatalf("binding.Env = %v; want nil (omitted env)", binding.Env)
	}
}

// TestLoadAgentBindingEnvRejectionTable exhausts the closed reject contract
// for AgentBinding.Env enforced by validateAgentBindingEnvNames. Every row
// declares a single offending entry (or duplicate pair) so the failure mode
// under test is unambiguous. Each rejection wraps ErrInvalidAgentBinding via
// ErrInvalidAgentBindingEnv; both sentinel routings are asserted.
//
// Acceptance per Drop 4c F.7.17.1 spec:
//
//   - Reject `=` in entry: most common authoring footgun (TOML editor writes
//     KEY=value instead of just KEY).
//   - Reject empty entry.
//   - Reject duplicate within a single binding's env list.
//   - Reject malformed names: whitespace, hyphen, dot, leading digit.
//   - Allow lowercase: `https_proxy`, `foo_bar` MUST pass.
func TestLoadAgentBindingEnvRejectionTable(t *testing.T) {
	tests := []struct {
		name         string
		env          string // raw TOML literal for the env array, e.g. `["KEY=value"]`
		wantValid    bool   // when true, Load returns nil error and the entry survives the validator
		wantSubstr   string // substring required in err.Error() when wantValid=false; empty = skip
		wantSentinel error
	}{
		{
			name:         "reject equals in entry KEY=value",
			env:          `["KEY=value"]`,
			wantSentinel: ErrInvalidAgentBindingEnv,
			wantSubstr:   "KEY=value",
		},
		{
			name:         "reject empty entry",
			env:          `[""]`,
			wantSentinel: ErrInvalidAgentBindingEnv,
			wantSubstr:   "is empty",
		},
		{
			name:         "reject duplicate entry within same binding",
			env:          `["FOO", "FOO"]`,
			wantSentinel: ErrInvalidAgentBindingEnv,
			wantSubstr:   "duplicated",
		},
		{
			name:         "reject whitespace-containing name FOO BAR",
			env:          `["FOO BAR"]`,
			wantSentinel: ErrInvalidAgentBindingEnv,
			wantSubstr:   "FOO BAR",
		},
		{
			name:         "reject hyphen-containing name FOO-BAR",
			env:          `["FOO-BAR"]`,
			wantSentinel: ErrInvalidAgentBindingEnv,
			wantSubstr:   "FOO-BAR",
		},
		{
			name:         "reject dot-containing name FOO.BAR",
			env:          `["FOO.BAR"]`,
			wantSentinel: ErrInvalidAgentBindingEnv,
			wantSubstr:   "FOO.BAR",
		},
		{
			name:         "reject leading-digit name 1FOO",
			env:          `["1FOO"]`,
			wantSentinel: ErrInvalidAgentBindingEnv,
			wantSubstr:   "1FOO",
		},
		{
			name:      "allow lowercase https_proxy",
			env:       `["https_proxy"]`,
			wantValid: true,
		},
		{
			name:      "allow lowercase foo_bar",
			env:       `["foo_bar"]`,
			wantValid: true,
		},
		{
			name:      "allow uppercase HTTP_PROXY",
			env:       `["HTTP_PROXY"]`,
			wantValid: true,
		},
		{
			name:      "allow trailing digits FOO123",
			env:       `["FOO123"]`,
			wantValid: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			src := `
schema_version = "v1"

[agent_bindings.build]
agent_name = "go-builder-agent"
model = "opus"
env = ` + tc.env + `
`
			_, err := Load(strings.NewReader(src))
			if tc.wantValid {
				if err != nil {
					t.Fatalf("Load: unexpected error %v for env=%s", err, tc.env)
				}
				return
			}
			if err == nil {
				t.Fatalf("Load: expected error for env=%s; got nil", tc.env)
			}
			if tc.wantSentinel != nil && !errors.Is(err, tc.wantSentinel) {
				t.Fatalf("Load: errors.Is(_, %v) = false; err = %v", tc.wantSentinel, err)
			}
			// Every env-rejection error MUST also satisfy errors.Is(_, ErrInvalidAgentBinding)
			// since ErrInvalidAgentBindingEnv wraps the umbrella sentinel.
			if !errors.Is(err, ErrInvalidAgentBinding) {
				t.Fatalf("Load: errors.Is(_, ErrInvalidAgentBinding) = false; err = %v", err)
			}
			if tc.wantSubstr != "" && !strings.Contains(err.Error(), tc.wantSubstr) {
				t.Fatalf("Load: err = %q; want substring %q", err.Error(), tc.wantSubstr)
			}
		})
	}
}

// TestLoadAgentBindingDuplicateEnvNamesEntry verifies the duplicate-detection
// state is per-binding, NOT cross-binding. Two distinct binding rows may each
// declare `FOO` without colliding because env values resolve at spawn time
// per their owning binding, not globally. The test seeds two bindings with
// identical env names and asserts Load succeeds.
func TestLoadAgentBindingDuplicateEnvNamesAcrossBindingsAllowed(t *testing.T) {
	src := `
schema_version = "v1"

[agent_bindings.build]
agent_name = "go-builder-agent"
model = "opus"
env = ["ANTHROPIC_API_KEY"]

[agent_bindings.plan]
agent_name = "go-planning-agent"
model = "opus"
env = ["ANTHROPIC_API_KEY"]
`
	if _, err := Load(strings.NewReader(src)); err != nil {
		t.Fatalf("Load: cross-binding duplicate env names should be allowed; err = %v", err)
	}
}

// TestLoadAgentBindingStrictDecodeUnknownFieldStillRejects verifies the
// strict-decode chain (load.go step 3) STILL rejects unknown nested fields
// inside [agent_bindings.<kind>] after Drop 4c F.7.17.1 widens the closed
// AgentBinding struct. This is regression coverage, not new functionality:
// adding `Env` and `CLIKind` to the struct doesn't relax strict decode for
// any other key. A bogus key like `bogus_field` MUST still surface as
// ErrUnknownTemplateKey at Load time.
func TestLoadAgentBindingStrictDecodeUnknownFieldStillRejects(t *testing.T) {
	src := `
schema_version = "v1"

[agent_bindings.build]
agent_name = "go-builder-agent"
model = "opus"
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

// TestLoadTillsynHappyPath verifies a Template TOML stream declaring the new
// Drop 4c F.7.18.2 [tillsyn] table with both fields populated decodes cleanly:
// MaxContextBundleChars and MaxAggregatorDuration land on tpl.Tillsyn with
// their declared values and validateTillsyn does not fire.
func TestLoadTillsynHappyPath(t *testing.T) {
	src := `
schema_version = "v1"

[tillsyn]
max_context_bundle_chars = 200000
max_aggregator_duration = "2s"
`
	tpl, err := Load(strings.NewReader(src))
	if err != nil {
		t.Fatalf("Load: unexpected error: %v", err)
	}
	if got, want := tpl.Tillsyn.MaxContextBundleChars, 200000; got != want {
		t.Fatalf("tpl.Tillsyn.MaxContextBundleChars = %d; want %d", got, want)
	}
	if got, want := time.Duration(tpl.Tillsyn.MaxAggregatorDuration), 2*time.Second; got != want {
		t.Fatalf("tpl.Tillsyn.MaxAggregatorDuration = %s; want %s", got, want)
	}
}

// TestLoadTillsynEmptyTableDecodes verifies a [tillsyn] table present in TOML
// but with all fields omitted loads cleanly with the zero-value Tillsyn
// struct. This pins the empty-table contract — the table itself is OK; only
// the field-level negative checks fire.
func TestLoadTillsynEmptyTableDecodes(t *testing.T) {
	src := `
schema_version = "v1"

[tillsyn]
`
	tpl, err := Load(strings.NewReader(src))
	if err != nil {
		t.Fatalf("Load: unexpected error: %v", err)
	}
	if tpl.Tillsyn.MaxContextBundleChars != 0 {
		t.Fatalf("tpl.Tillsyn.MaxContextBundleChars = %d; want 0 (omitted-field zero value)",
			tpl.Tillsyn.MaxContextBundleChars)
	}
	if time.Duration(tpl.Tillsyn.MaxAggregatorDuration) != 0 {
		t.Fatalf("tpl.Tillsyn.MaxAggregatorDuration = %s; want 0 (omitted-field zero value)",
			time.Duration(tpl.Tillsyn.MaxAggregatorDuration))
	}
}

// TestLoadTillsynOmittedTableZeroValue verifies a Template TOML stream WITHOUT
// any [tillsyn] table loads cleanly with the zero-value Tillsyn struct. The
// engine-time default-substitution layer (F.7.18.4) reads the zero value and
// substitutes its own bundle-global default, so omission is the canonical
// "use defaults" sentinel at the schema layer.
func TestLoadTillsynOmittedTableZeroValue(t *testing.T) {
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
	if tpl.Tillsyn.MaxContextBundleChars != 0 {
		t.Fatalf("tpl.Tillsyn.MaxContextBundleChars = %d; want 0 (absent-table zero value)",
			tpl.Tillsyn.MaxContextBundleChars)
	}
	if time.Duration(tpl.Tillsyn.MaxAggregatorDuration) != 0 {
		t.Fatalf("tpl.Tillsyn.MaxAggregatorDuration = %s; want 0 (absent-table zero value)",
			time.Duration(tpl.Tillsyn.MaxAggregatorDuration))
	}
}

// TestLoadTillsynZeroValuesAllowed verifies that explicitly setting both
// fields to zero in TOML loads cleanly — zero is the engine-time-default
// sentinel per master PLAN L14/L15, not a rejected value. This pins the
// "zero is legal" contract against accidental drift to a `> 0` validator.
func TestLoadTillsynZeroValuesAllowed(t *testing.T) {
	src := `
schema_version = "v1"

[tillsyn]
max_context_bundle_chars = 0
max_aggregator_duration = "0s"
`
	tpl, err := Load(strings.NewReader(src))
	if err != nil {
		t.Fatalf("Load: explicit zero values must load cleanly; err = %v", err)
	}
	if tpl.Tillsyn.MaxContextBundleChars != 0 {
		t.Fatalf("tpl.Tillsyn.MaxContextBundleChars = %d; want 0", tpl.Tillsyn.MaxContextBundleChars)
	}
	if time.Duration(tpl.Tillsyn.MaxAggregatorDuration) != 0 {
		t.Fatalf("tpl.Tillsyn.MaxAggregatorDuration = %s; want 0",
			time.Duration(tpl.Tillsyn.MaxAggregatorDuration))
	}
}

// TestLoadTillsynRejectsNegativeMaxContextBundleChars verifies validateTillsyn
// rejects a negative MaxContextBundleChars value with ErrInvalidTillsynGlobals.
// The error message names the offending value verbatim for UX.
func TestLoadTillsynRejectsNegativeMaxContextBundleChars(t *testing.T) {
	src := `
schema_version = "v1"

[tillsyn]
max_context_bundle_chars = -1
`
	_, err := Load(strings.NewReader(src))
	if err == nil {
		t.Fatalf("Load: expected ErrInvalidTillsynGlobals; got nil")
	}
	if !errors.Is(err, ErrInvalidTillsynGlobals) {
		t.Fatalf("Load: errors.Is(_, ErrInvalidTillsynGlobals) = false; err = %v", err)
	}
	if !strings.Contains(err.Error(), "max_context_bundle_chars") {
		t.Fatalf("Load: err = %q; want substring %q", err.Error(), "max_context_bundle_chars")
	}
	if !strings.Contains(err.Error(), "-1") {
		t.Fatalf("Load: err = %q; want offending value %q in message", err.Error(), "-1")
	}
}

// TestLoadTillsynRejectsNegativeMaxAggregatorDuration verifies validateTillsyn
// rejects a negative MaxAggregatorDuration value with ErrInvalidTillsynGlobals.
// The error message names the offending duration string verbatim for UX.
func TestLoadTillsynRejectsNegativeMaxAggregatorDuration(t *testing.T) {
	src := `
schema_version = "v1"

[tillsyn]
max_aggregator_duration = "-1s"
`
	_, err := Load(strings.NewReader(src))
	if err == nil {
		t.Fatalf("Load: expected ErrInvalidTillsynGlobals; got nil")
	}
	if !errors.Is(err, ErrInvalidTillsynGlobals) {
		t.Fatalf("Load: errors.Is(_, ErrInvalidTillsynGlobals) = false; err = %v", err)
	}
	if !strings.Contains(err.Error(), "max_aggregator_duration") {
		t.Fatalf("Load: err = %q; want substring %q", err.Error(), "max_aggregator_duration")
	}
	if !strings.Contains(err.Error(), "-1s") {
		t.Fatalf("Load: err = %q; want offending value %q in message", err.Error(), "-1s")
	}
}

// TestLoadTillsynStrictDecodeUnknownFieldRejected is the REV-3 contract test:
// a [tillsyn] table with an unknown key MUST fail load with
// ErrUnknownTemplateKey. This proves the closed-struct unknown-key rejection
// from load.go step 3 (DisallowUnknownFields) actually fires for the new
// top-level table, so the F.7-CORE F.7.1 + F.7.6 extenders inherit the
// rejection automatically per pelletier/go-toml v2 semantics.
func TestLoadTillsynStrictDecodeUnknownFieldRejected(t *testing.T) {
	src := `
schema_version = "v1"

[tillsyn]
max_context_bundle_chars = 200000
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
