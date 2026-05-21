// Package templates — tests for the Drop 4d_5 D0 parallel-peer conflict
// detector between agents.toml Preset.Client and template.toml
// AgentBinding.CLIKind. Co-located with the production validator in
// load.go per CLAUDE.md § "Tests" discipline.
package templates

import (
	"errors"
	"strings"
	"testing"

	"github.com/evanmschultz/tillsyn/internal/config"
	"github.com/evanmschultz/tillsyn/internal/domain"
)

// makeRegistry builds an AgentsRegistry whose single group named `group`
// carries Preset.Client = `defaultClient` and (when overrideKind is non-empty)
// a per-kind override resolving Client = `overrideClient`. The helper keeps
// the test rows compact — every scenario boils down to "agents resolves
// kind=X to client=Y" against a template binding with CLIKind=Z.
func makeRegistry(group, defaultClient, overrideKind, overrideClient string) config.AgentsRegistry {
	gc := config.GroupConfig{
		Default: config.Preset{Client: defaultClient},
		Kinds:   make(map[domain.Kind]config.Override),
	}
	if overrideKind != "" {
		c := overrideClient
		gc.Kinds[domain.Kind(overrideKind)] = config.Override{Client: &c}
	}
	return config.AgentsRegistry{group: gc}
}

// makeCatalog builds a KindCatalog whose single AgentBinding for `kind`
// carries CLIKind = `cliKind`. The helper mirrors the agents-side
// makeRegistry shape so the test scenarios pair one-for-one.
func makeCatalog(kind domain.Kind, cliKind string) KindCatalog {
	return KindCatalog{
		AgentBindings: map[domain.Kind]AgentBinding{
			kind: {AgentName: "builder-agent", Model: "sonnet", CLIKind: cliKind},
		},
	}
}

// TestValidateAgentsTemplateClientConflict_AgreementCases covers the
// happy-path scenarios where the validator returns nil. Two distinct
// agreement shapes:
//
//  1. Normalised-agreement — both sides set "claude" verbatim. The
//     pre-normalised forms collapse to the same canonical form.
//  2. Case-difference-agreement — agents="Codex", template="codex". Per
//     round-2 falsification HIGH-3 the symmetric normalizeClient helper
//     case-folds both sides BEFORE comparison so this MUST NOT raise a
//     false conflict.
//
// Per-scenario row drives ValidateAgentsTemplateClientConflict against
// the corresponding registry + catalog pair and asserts err == nil.
func TestValidateAgentsTemplateClientConflict_AgreementCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name              string
		agentsClient      string
		templateClient    string
		caseFoldRationale string
	}{
		{
			name:              "normalised-agreement-claude",
			agentsClient:      "claude",
			templateClient:    "claude",
			caseFoldRationale: "both sides canonical; no fold required",
		},
		{
			name:              "normalised-agreement-codex",
			agentsClient:      "codex",
			templateClient:    "codex",
			caseFoldRationale: "both sides canonical; no fold required",
		},
		{
			name:              "case-difference-agreement-codex-vs-Codex",
			agentsClient:      "codex",
			templateClient:    "Codex",
			caseFoldRationale: "round-2 HIGH-3: case-fold both sides via normalizeClient before equality",
		},
		{
			name:              "case-difference-agreement-Codex-vs-codex",
			agentsClient:      "Codex",
			templateClient:    "codex",
			caseFoldRationale: "round-2 HIGH-3: symmetric normalisation — directionality must not matter",
		},
		{
			name:              "whitespace-agreement",
			agentsClient:      "  claude  ",
			templateClient:    "claude",
			caseFoldRationale: "normalizeClient trims surrounding whitespace symmetrically",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			registry := makeRegistry("go", tc.agentsClient, "", "")
			catalog := makeCatalog(domain.KindBuild, tc.templateClient)
			err := ValidateAgentsTemplateClientConflict(&registry, &catalog)
			if err != nil {
				t.Fatalf("expected nil error (%s); got %v", tc.caseFoldRationale, err)
			}
		})
	}
}

// TestValidateAgentsTemplateClientConflict_DisagreementRaisesConflict
// covers the canonical conflict scenario: agents resolves kind=build to
// "claude" but template's build binding declares cli_kind="codex". The
// validator MUST return *ConflictingCLIKindError wrapping
// ErrConflictingCLIKind and the envelope's Group / Kind / AgentsValue /
// TemplateValue fields MUST match the offending pair.
func TestValidateAgentsTemplateClientConflict_DisagreementRaisesConflict(t *testing.T) {
	t.Parallel()

	registry := makeRegistry("go", "claude", "", "")
	catalog := makeCatalog(domain.KindBuild, "codex")

	err := ValidateAgentsTemplateClientConflict(&registry, &catalog)
	if err == nil {
		t.Fatal("expected conflict error; got nil")
	}
	if !errors.Is(err, ErrConflictingCLIKind) {
		t.Fatalf("err does not wrap ErrConflictingCLIKind: %v", err)
	}

	var typedErr *ConflictingCLIKindError
	if !errors.As(err, &typedErr) {
		t.Fatalf("err does not unwrap to *ConflictingCLIKindError: %v", err)
	}
	if typedErr.Group != "go" {
		t.Errorf("Group = %q; want %q", typedErr.Group, "go")
	}
	if typedErr.Kind != domain.KindBuild {
		t.Errorf("Kind = %q; want %q", typedErr.Kind, domain.KindBuild)
	}
	if typedErr.AgentsValue != "claude" {
		t.Errorf("AgentsValue = %q; want %q", typedErr.AgentsValue, "claude")
	}
	if typedErr.TemplateValue != "codex" {
		t.Errorf("TemplateValue = %q; want %q", typedErr.TemplateValue, "codex")
	}

	// UX assertion: the rendered message includes both values so the dev
	// sees the offending knobs at a glance.
	rendered := err.Error()
	for _, want := range []string{"go", "claude", "codex", "parallel-peer"} {
		if !strings.Contains(rendered, want) {
			t.Errorf("rendered message %q missing %q", rendered, want)
		}
	}
}

// TestValidateAgentsTemplateClientConflict_SingleSideSpecification verifies
// the "open" side wins implicitly when only one of (agents, template)
// declares a non-empty CLIKind. Per the validator contract single-side
// specification is NOT a conflict — the open side defers to the populated
// side at adapter-lookup time per F.7.17 L15.
//
// Two distinct scenarios:
//
//  1. Only-agents — agents resolves to "codex"; template's binding has
//     empty CLIKind. No conflict.
//  2. Only-template — agents resolves to "" (empty Preset.Client);
//     template's binding declares CLIKind="codex". No conflict.
func TestValidateAgentsTemplateClientConflict_SingleSideSpecification(t *testing.T) {
	t.Parallel()

	t.Run("only-agents-set", func(t *testing.T) {
		t.Parallel()
		registry := makeRegistry("go", "codex", "", "")
		catalog := makeCatalog(domain.KindBuild, "")
		if err := ValidateAgentsTemplateClientConflict(&registry, &catalog); err != nil {
			t.Fatalf("only-agents-set raised unexpected conflict: %v", err)
		}
	})

	t.Run("only-template-set", func(t *testing.T) {
		t.Parallel()
		registry := makeRegistry("go", "", "", "")
		catalog := makeCatalog(domain.KindBuild, "codex")
		if err := ValidateAgentsTemplateClientConflict(&registry, &catalog); err != nil {
			t.Fatalf("only-template-set raised unexpected conflict: %v", err)
		}
	})

	t.Run("both-empty", func(t *testing.T) {
		t.Parallel()
		// Both sides "open" — F.7.17 L15 default-to-claude resolves at
		// adapter time. No conflict at validation time.
		registry := makeRegistry("go", "", "", "")
		catalog := makeCatalog(domain.KindBuild, "")
		if err := ValidateAgentsTemplateClientConflict(&registry, &catalog); err != nil {
			t.Fatalf("both-empty raised unexpected conflict: %v", err)
		}
	})

	t.Run("only-template-set-whitespace-agents", func(t *testing.T) {
		t.Parallel()
		// Whitespace-only on the agents side normalises to "" — same as
		// absent. Template wins implicitly.
		registry := makeRegistry("go", "   ", "", "")
		catalog := makeCatalog(domain.KindBuild, "codex")
		if err := ValidateAgentsTemplateClientConflict(&registry, &catalog); err != nil {
			t.Fatalf("whitespace-only agents side raised conflict: %v", err)
		}
	})
}

// TestValidateAgentsTemplateClientConflict_NilRegistryGuard exercises the
// nil-registry guard per the validator's docstring contract: a nil
// registry pointer is treated as "no agents-side preset for any kind" so
// every kind passes vacuously. Same shape for nil catalog. The validator
// must not panic on any nil input combination.
//
// This guard is load-bearing per round-2 Open Q 6.3: adopters on a fresh
// machine boot with no agents.toml. nil-registry must boot, not crash.
func TestValidateAgentsTemplateClientConflict_NilRegistryGuard(t *testing.T) {
	t.Parallel()

	catalog := makeCatalog(domain.KindBuild, "codex")
	registry := makeRegistry("go", "codex", "", "")
	emptyRegistry := config.AgentsRegistry{}
	emptyCatalog := KindCatalog{}

	tests := []struct {
		name     string
		registry *config.AgentsRegistry
		catalog  *KindCatalog
	}{
		{name: "nil-registry", registry: nil, catalog: &catalog},
		{name: "nil-catalog", registry: &registry, catalog: nil},
		{name: "both-nil", registry: nil, catalog: nil},
		{name: "empty-registry-map", registry: &emptyRegistry, catalog: &catalog},
		{name: "empty-catalog-bindings", registry: &registry, catalog: &emptyCatalog},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("validator panicked on %s: %v", tc.name, r)
				}
			}()
			if err := ValidateAgentsTemplateClientConflict(tc.registry, tc.catalog); err != nil {
				t.Fatalf("expected nil error on %s; got %v", tc.name, err)
			}
		})
	}
}

// TestValidateAgentsTemplateClientConflict_PerKindOverrideWinsOverDefault
// verifies the validator walks config.Resolve to obtain the EFFECTIVE
// agents-side Client (Preset → per-kind Override). The default Client is
// "claude" but the per-kind override sets "codex"; template declares
// "codex". After Resolve runs, agents and template AGREE — no conflict.
//
// This locks the contract: the validator must NOT compare against the raw
// Preset.Client when an Override.Client pointer is set, otherwise an
// adopter who overrides one kind to "codex" while keeping the group
// default "claude" sees a false conflict.
func TestValidateAgentsTemplateClientConflict_PerKindOverrideWinsOverDefault(t *testing.T) {
	t.Parallel()

	// Default = claude, override build = codex. Template build = codex.
	// Expected: no conflict (Resolve returns codex for kind=build).
	registry := makeRegistry("go", "claude", string(domain.KindBuild), "codex")
	catalog := makeCatalog(domain.KindBuild, "codex")

	if err := ValidateAgentsTemplateClientConflict(&registry, &catalog); err != nil {
		t.Fatalf("expected per-kind override to win over default; got conflict: %v", err)
	}

	// Inverse: default = codex, override build = claude. Template build =
	// codex. Resolve returns claude for kind=build. Expected: conflict.
	inverted := makeRegistry("go", "codex", string(domain.KindBuild), "claude")
	if err := ValidateAgentsTemplateClientConflict(&inverted, &catalog); err == nil {
		t.Fatal("expected per-kind override (claude) to conflict with template (codex); got nil")
	} else if !errors.Is(err, ErrConflictingCLIKind) {
		t.Fatalf("expected ErrConflictingCLIKind; got %v", err)
	}
}

// TestValidateAgentsTemplateClientConflict_MultiGroupDeterministicOrder
// pins the validator's first-conflict diagnostic against the sorted group
// iteration order. Two groups disagree with the same template binding; the
// validator returns the conflict for the lexically-first group (`fe` <
// `go`). Reproducibility matters for dev grep + CI logs.
func TestValidateAgentsTemplateClientConflict_MultiGroupDeterministicOrder(t *testing.T) {
	t.Parallel()

	registry := config.AgentsRegistry{
		"fe": {Default: config.Preset{Client: "claude"}},
		"go": {Default: config.Preset{Client: "codex"}},
	}
	catalog := makeCatalog(domain.KindBuild, "codex")
	// fe resolves to claude (conflict with template=codex);
	// go resolves to codex (no conflict).
	// Sorted iteration visits fe first, so the conflict surfaces with
	// Group="fe".

	err := ValidateAgentsTemplateClientConflict(&registry, &catalog)
	if err == nil {
		t.Fatal("expected conflict; got nil")
	}
	var typedErr *ConflictingCLIKindError
	if !errors.As(err, &typedErr) {
		t.Fatalf("err does not unwrap to *ConflictingCLIKindError: %v", err)
	}
	if typedErr.Group != "fe" {
		t.Errorf("Group = %q; want %q (sorted iteration must visit fe before go)", typedErr.Group, "fe")
	}
}

// TestNormalizeClient verifies the closed normalisation contract: trim
// surrounding ASCII whitespace, fold to lowercase. The helper is the
// symmetric mitigation against round-2 falsification HIGH-3 so D0's
// conflict-compare and D2's BindingOverridesFromPreset (when it lands)
// route through the SAME function.
func TestNormalizeClient(t *testing.T) {
	t.Parallel()

	tests := []struct {
		in   string
		want string
	}{
		{in: "", want: ""},
		{in: "claude", want: "claude"},
		{in: "Claude", want: "claude"},
		{in: "CLAUDE", want: "claude"},
		{in: "  codex  ", want: "codex"},
		{in: "\tCodex\n", want: "codex"},
		{in: "   ", want: ""},
		{in: "Mixed Case With Spaces", want: "mixed case with spaces"},
	}

	for _, tc := range tests {
		got := normalizeClient(tc.in)
		if got != tc.want {
			t.Errorf("normalizeClient(%q) = %q; want %q", tc.in, got, tc.want)
		}
	}
}
