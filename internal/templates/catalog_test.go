package templates

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"github.com/evanmschultz/tillsyn/internal/domain"
)

// catalogFixtureTemplate returns a hand-coded Template covering all four
// shape concerns the catalog exercises: a kind with both allow-list axes
// populated, a kind with only the parent axis populated, a stand-alone
// AgentBinding row, and a structural-type assignment. Per droplet 3.12 the
// fixture is hand-coded (not loaded from default.toml) for the same drift-
// mitigation reason droplet 3.10 hand-codes its own — two distinct
// assertion paths against the same source of truth.
func catalogFixtureTemplate() Template {
	return Template{
		SchemaVersion: SchemaVersionV1,
		Kinds: map[domain.Kind]KindRule{
			domain.KindBuild: {
				Owner:              "STEWARD",
				AllowedParentKinds: []domain.Kind{domain.KindPlan},
				AllowedChildKinds:  []domain.Kind{domain.KindBuildQAProof, domain.KindBuildQAFalsification},
				StructuralType:     domain.StructuralTypeDroplet,
			},
			domain.KindPlan: {
				AllowedChildKinds: []domain.Kind{domain.KindBuild, domain.KindPlanQAProof, domain.KindPlanQAFalsification},
				StructuralType:    domain.StructuralTypeSegment,
			},
		},
		AgentBindings: map[domain.Kind]AgentBinding{
			domain.KindBuild: {
				AgentName:            "go-builder-agent",
				Model:                "opus",
				Effort:               "high",
				Tools:                []string{"Read", "Edit", "Bash"},
				MaxTries:             3,
				MaxBudgetUSD:         5.0,
				MaxTurns:             50,
				AutoPush:             true,
				CommitAgent:          "commit-agent",
				BlockedRetries:       2,
				BlockedRetryCooldown: Duration(30 * time.Second),
			},
		},
	}
}

// TestKindCatalogBakeIsIdempotent verifies that Bake produces deeply equal
// catalogs across repeated calls on the same template. This is the load-
// time invariant a Drop 4 dispatcher relies on: a freshly re-baked catalog
// after a process restart must equal the persisted KindCatalogJSON.
func TestKindCatalogBakeIsIdempotent(t *testing.T) {
	tpl := catalogFixtureTemplate()
	first := Bake(tpl)
	second := Bake(tpl)
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("Bake() not idempotent: first=%#v second=%#v", first, second)
	}
}

// TestKindCatalogLookupHit verifies a kind populated in the template's
// KindRule map round-trips through Bake → Lookup with deep equality, and
// that the returned ok flag is true.
func TestKindCatalogLookupHit(t *testing.T) {
	tpl := catalogFixtureTemplate()
	cat := Bake(tpl)
	rule, ok := cat.Lookup(domain.KindBuild)
	if !ok {
		t.Fatal("Lookup(KindBuild) ok = false, want true")
	}
	want := tpl.Kinds[domain.KindBuild]
	if !reflect.DeepEqual(rule, want) {
		t.Fatalf("Lookup(KindBuild) = %#v, want %#v", rule, want)
	}
}

// TestKindCatalogLookupMiss verifies a kind absent from the template
// returns (zero, false). KindResearch is missing from catalogFixtureTemplate
// by construction.
func TestKindCatalogLookupMiss(t *testing.T) {
	cat := Bake(catalogFixtureTemplate())
	rule, ok := cat.Lookup(domain.KindResearch)
	if ok {
		t.Fatalf("Lookup(KindResearch) ok = true, want false (rule=%#v)", rule)
	}
	if !reflect.DeepEqual(rule, KindRule{}) {
		t.Fatalf("Lookup(KindResearch) rule = %#v, want zero KindRule", rule)
	}
}

// TestKindCatalogLookupZeroValue verifies that the zero-value catalog
// (no Kinds map allocated) returns (zero, false) for any input without
// panicking. The zero-value catalog is the legitimate "empty" case the
// app-layer fallback path keys on per droplet 3.12 acceptance criterion.
func TestKindCatalogLookupZeroValue(t *testing.T) {
	var cat KindCatalog
	rule, ok := cat.Lookup(domain.KindBuild)
	if ok {
		t.Fatalf("Lookup on zero-value catalog ok = true, want false (rule=%#v)", rule)
	}
	if !reflect.DeepEqual(rule, KindRule{}) {
		t.Fatalf("Lookup on zero-value catalog rule = %#v, want zero KindRule", rule)
	}
	binding, ok := cat.LookupAgentBinding(domain.KindBuild)
	if ok {
		t.Fatalf("LookupAgentBinding on zero-value catalog ok = true, want false (binding=%#v)", binding)
	}
	if !reflect.DeepEqual(binding, AgentBinding{}) {
		t.Fatalf("LookupAgentBinding on zero-value catalog binding = %#v, want zero AgentBinding", binding)
	}
}

// TestKindCatalogJSONRoundTrip verifies the catalog persists via the
// json.RawMessage envelope on Project. A baked catalog marshalled to JSON
// and unmarshalled back must equal the original — that property is what
// makes Project.KindCatalogJSON a faithful snapshot of the original
// Template's kind / agent-binding data.
func TestKindCatalogJSONRoundTrip(t *testing.T) {
	original := Bake(catalogFixtureTemplate())
	encoded, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("json.Marshal(catalog) error = %v", err)
	}
	var decoded KindCatalog
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("json.Unmarshal(catalog) error = %v", err)
	}
	if !reflect.DeepEqual(original, decoded) {
		t.Fatalf("JSON round-trip mismatch:\noriginal=%#v\ndecoded=%#v", original, decoded)
	}
}

// TestKindCatalogLookupAgentBindingHit verifies an AgentBinding round-trips
// through Bake → LookupAgentBinding with deep equality.
func TestKindCatalogLookupAgentBindingHit(t *testing.T) {
	tpl := catalogFixtureTemplate()
	cat := Bake(tpl)
	binding, ok := cat.LookupAgentBinding(domain.KindBuild)
	if !ok {
		t.Fatal("LookupAgentBinding(KindBuild) ok = false, want true")
	}
	want := tpl.AgentBindings[domain.KindBuild]
	if !reflect.DeepEqual(binding, want) {
		t.Fatalf("LookupAgentBinding(KindBuild) = %#v, want %#v", binding, want)
	}
}

// TestKindCatalogBakeDeepCopiesSlices verifies that mutating a slice on the
// source Template after Bake does not bleed into the catalog. This is the
// invariant that justifies treating the catalog as immutable downstream.
func TestKindCatalogBakeDeepCopiesSlices(t *testing.T) {
	tpl := catalogFixtureTemplate()
	cat := Bake(tpl)
	// Mutate the source: extend the AllowedChildKinds slice on KindBuild.
	srcRule := tpl.Kinds[domain.KindBuild]
	srcRule.AllowedChildKinds[0] = domain.KindResearch
	tpl.Kinds[domain.KindBuild] = srcRule
	// The catalog's AllowedChildKinds[0] must still be its original value.
	cataloged, ok := cat.Lookup(domain.KindBuild)
	if !ok {
		t.Fatal("Lookup(KindBuild) ok = false, want true")
	}
	if cataloged.AllowedChildKinds[0] != domain.KindBuildQAProof {
		t.Fatalf("Bake aliased slice: catalog rule[0] = %q, want %q", cataloged.AllowedChildKinds[0], domain.KindBuildQAProof)
	}
}
