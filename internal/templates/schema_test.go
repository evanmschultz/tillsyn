package templates

import (
	"reflect"
	"testing"
	"time"

	toml "github.com/pelletier/go-toml/v2"

	"github.com/evanmschultz/tillsyn/internal/domain"
)

// TestSchemaVersionV1Constant verifies the locked initial schema-version
// string is the literal "v1". Subsequent schema revisions land as new
// constants alongside their migration gate; this test guards against an
// accidental re-spelling.
func TestSchemaVersionV1Constant(t *testing.T) {
	if SchemaVersionV1 != "v1" {
		t.Fatalf("SchemaVersionV1 = %q; want %q", SchemaVersionV1, "v1")
	}
}

// TestZeroValuesConstructible verifies every top-level type in the schema is
// constructible at its zero value. The assertion is a smoke check that the
// types compile and that the package does not require any mandatory
// constructor argument (parsing/validation lives in 3.9+).
func TestZeroValuesConstructible(t *testing.T) {
	var (
		tpl     Template
		kr      KindRule
		cr      ChildRule
		binding AgentBinding
	)
	// Touch each value so the compiler does not strip the declarations.
	if tpl.SchemaVersion != "" || kr.Owner != "" || cr.Title != "" || binding.AgentName != "" {
		t.Fatalf("zero-value Template/KindRule/ChildRule/AgentBinding fields should be empty")
	}
}

// TestTemplateTOMLRoundTrip verifies struct-tag correctness end-to-end by
// marshalling a populated Template via pelletier/go-toml/v2 and unmarshalling
// the resulting TOML back into an equivalent value. Slices and maps are
// populated with at least one element so the assertion does not trip on the
// nil-vs-empty-slice asymmetry that TOML round-trips can introduce.
//
// This test does not exercise the schema-version gate, kind-membership
// validator, or any other behavior reserved for later droplets — it only
// confirms that every TOML tag declared in schema.go round-trips cleanly.
//
// Drop 4b Wave A 4b.1 extension: the populated Gates field round-trips so
// the [gates] TOML key encoding survives marshal+unmarshal symmetrically with
// the closed-enum GateKind value-slice axis.
func TestTemplateTOMLRoundTrip(t *testing.T) {
	original := Template{
		SchemaVersion: SchemaVersionV1,
		Kinds: map[domain.Kind]KindRule{
			domain.KindBuild: {
				Owner:              "STEWARD",
				AllowedParentKinds: []domain.Kind{domain.KindPlan},
				AllowedChildKinds:  []domain.Kind{domain.KindBuildQAProof, domain.KindBuildQAFalsification},
				StructuralType:     domain.StructuralTypeDroplet,
			},
		},
		ChildRules: []ChildRule{
			{
				WhenParentKind:           domain.KindBuild,
				CreateChildKind:          domain.KindBuildQAProof,
				Title:                    "BUILD-QA-PROOF",
				BlockedByParent:          true,
				WhenParentStructuralType: domain.StructuralTypeDroplet,
			},
		},
		AgentBindings: map[domain.Kind]AgentBinding{
			domain.KindBuild: {
				AgentName:            "go-builder-agent",
				Model:                "opus",
				Effort:               "high",
				Tools:                []string{"Edit", "Write", "Bash"},
				MaxTries:             3,
				MaxBudgetUSD:         2.5,
				MaxTurns:             50,
				AutoPush:             true,
				CommitAgent:          "commit-agent",
				BlockedRetries:       2,
				BlockedRetryCooldown: Duration(30 * time.Second),
				Env:                  []string{"ANTHROPIC_API_KEY", "https_proxy"},
				CLIKind:              "claude",
				// Drop 4c F.7.18.1: every ContextRules field populated so
				// the round-trip exercises the new sub-struct's TOML tags
				// symmetrically with the existing AgentBinding fields.
				Context: ContextRules{
					Parent:            true,
					ParentGitDiff:     true,
					SiblingsByKind:    []domain.Kind{domain.KindBuildQAProof},
					AncestorsByKind:   []domain.Kind{domain.KindPlan},
					DescendantsByKind: []domain.Kind{domain.KindBuild},
					Delivery:          ContextDeliveryFile,
					MaxChars:          50000,
					MaxRuleDuration:   Duration(500 * time.Millisecond),
				},
				// Drop 4c F.7.2: tool-gating + system-prompt-template +
				// sandbox fields populated so the round-trip exercises the
				// new TOML tags symmetrically with the existing fields.
				ToolsAllowed:             []string{"Read", "Edit", "Bash(mage *)"},
				ToolsDisallowed:          []string{"WebFetch"},
				SystemPromptTemplatePath: "prompts/build.md",
				Sandbox: SandboxRules{
					Filesystem: SandboxFilesystem{
						AllowWrite: []string{"/Users/me/repo"},
						DenyRead:   []string{"/etc/secrets"},
					},
					Network: SandboxNetwork{
						AllowedDomains: []string{"github.com", "*.npmjs.org"},
						DeniedDomains:  []string{"badactor.example"},
					},
				},
			},
		},
		Gates: map[domain.Kind][]GateKind{
			domain.KindBuild: {GateKindMageCI, GateKindMageTestPkg},
		},
		// Drop 4c F.7.18.2: every Tillsyn field populated so the round-trip
		// exercises the new top-level struct's TOML tags symmetrically with
		// the existing Template fields.
		Tillsyn: Tillsyn{
			MaxContextBundleChars: 200000,
			MaxAggregatorDuration: Duration(2 * time.Second),
		},
		StewardSeeds: []StewardSeed{
			{Title: "DISCUSSIONS", Description: "Cross-cutting discussion topics."},
			{Title: "REFINEMENTS", Description: "Tillsyn product refinements."},
		},
	}

	encoded, err := toml.Marshal(original)
	if err != nil {
		t.Fatalf("toml.Marshal: %v", err)
	}

	var decoded Template
	if err := toml.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("toml.Unmarshal: %v\nencoded TOML:\n%s", err, encoded)
	}

	if !reflect.DeepEqual(original, decoded) {
		t.Fatalf("round-trip mismatch\noriginal: %#v\ndecoded:  %#v\nencoded TOML:\n%s", original, decoded, encoded)
	}
}

// TestGateKindClosedEnum verifies the three Drop 4b Wave A GateKind constants
// are members of the closed enum (IsValidGateKind returns true) and that
// Drop-4c-future values ("commit", "push") plus arbitrary garbage and the
// empty string are rejected. Adding "commit" / "push" in Drop 4c flips the
// commit/push assertions; this test pins the Wave A vocabulary explicitly.
func TestGateKindClosedEnum(t *testing.T) {
	t.Parallel()

	validCases := []GateKind{
		GateKindMageCI,
		GateKindMageTestPkg,
		GateKindHyllaReingest,
	}
	for _, g := range validCases {
		t.Run("valid_"+string(g), func(t *testing.T) {
			t.Parallel()
			if !IsValidGateKind(g) {
				t.Fatalf("IsValidGateKind(%q) = false; want true", g)
			}
		})
	}

	invalidCases := []GateKind{
		GateKind("commit"),    // Drop 4c will accept this; Wave A rejects.
		GateKind("push"),      // Drop 4c will accept this; Wave A rejects.
		GateKind(""),          // Empty string is never valid.
		GateKind("garbage"),   // Arbitrary unknown value.
		GateKind("MAGE_CI"),   // Case mismatch — exact match enforced.
		GateKind(" mage_ci "), // Whitespace padding — exact match enforced.
		GateKind("mage-ci"),   // Hyphen vs underscore — exact match enforced.
	}
	for _, g := range invalidCases {
		t.Run("invalid_"+string(g), func(t *testing.T) {
			t.Parallel()
			if IsValidGateKind(g) {
				t.Fatalf("IsValidGateKind(%q) = true; want false", g)
			}
		})
	}
}
