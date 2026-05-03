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
			},
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
