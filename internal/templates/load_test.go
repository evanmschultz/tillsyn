package templates

import (
	"errors"
	"strings"
	"testing"

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
