package templates

import (
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	toml "github.com/pelletier/go-toml/v2"

	"github.com/evanmschultz/tillsyn/internal/domain"
)

// fullyPopulatedAgentBinding returns a fresh AgentBinding with every one of
// the populated fields set to a non-zero, validation-passing value. Used as
// the shared baseline for the round-trip test and as the start state for each
// Validate table-row's targeted mutation.
//
// Drop 4c F.7.17.1 extension: Env + CLIKind are populated so the round-trip
// assertion exercises the new TOML tags symmetrically with the original
// 11 fields. CLIKind is set to the closed-enum literal "claude" (REV-1
// closed enum: claude | codex; codex lands in Drop 4d).
//
// Drop 4c F.7.18.1 extension: Context is populated with every ContextRules
// field set so the round-trip exercises the new sub-struct's TOML tags
// symmetrically. The kind-walk slices (SiblingsByKind / AncestorsByKind /
// DescendantsByKind) carry at least one valid closed-enum kind so the
// validator does not surface ErrUnknownKindReference at Load time, and so
// reflect.DeepEqual does not trip on the nil-vs-empty-slice asymmetry that
// pelletier/go-toml/v2 introduces for empty arrays.
//
// Drop 4c F.7.2 extension: ToolsAllowed / ToolsDisallowed /
// SystemPromptTemplatePath / Sandbox are populated with non-zero,
// validator-passing values so the round-trip exercises the new TOML tags
// symmetrically. Sandbox carries one entry per slice (one allow_write, one
// deny_read, one allowed_domains, one denied_domains) for the same nil-vs-
// empty asymmetry rationale documented above.
func fullyPopulatedAgentBinding() AgentBinding {
	return AgentBinding{
		AgentName:            "go-builder-agent",
		Model:                "opus",
		Effort:               "high",
		Tools:                []string{"Edit", "Write", "Bash", "Read"},
		MaxTries:             3,
		MaxBudgetUSD:         2.5,
		MaxTurns:             50,
		AutoPush:             true,
		CommitAgent:          "commit-agent",
		BlockedRetries:       2,
		BlockedRetryCooldown: Duration(30 * time.Second),
		Env:                  []string{"ANTHROPIC_API_KEY", "https_proxy", "HTTP_PROXY"},
		CLIKind:              "claude",
		Context: ContextRules{
			Parent:            true,
			ParentGitDiff:     true,
			SiblingsByKind:    []domain.Kind{domain.KindBuildQAProof, domain.KindBuildQAFalsification},
			AncestorsByKind:   []domain.Kind{domain.KindPlan},
			DescendantsByKind: []domain.Kind{domain.KindBuild},
			Delivery:          ContextDeliveryFile,
			MaxChars:          50000,
			MaxRuleDuration:   Duration(500 * time.Millisecond),
		},
		ToolsAllowed:             []string{"Read", "Edit", "Bash(mage *)"},
		ToolsDisallowed:          []string{"WebFetch", "Bash(curl *)"},
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
	}
}

// TestAgentBindingTOMLRoundTrip marshals a fully-populated AgentBinding via
// pelletier/go-toml/v2 and unmarshals the resulting bytes back into a struct,
// asserting reflect.DeepEqual equivalence. Every one of the 13 fields is
// populated with a non-zero value (Tools and Env each have multiple elements,
// BlockedRetryCooldown is non-zero, CLIKind = "claude") so the assertion does
// not pass trivially for an unmarshaller that silently drops fields.
func TestAgentBindingTOMLRoundTrip(t *testing.T) {
	original := fullyPopulatedAgentBinding()

	encoded, err := toml.Marshal(original)
	if err != nil {
		t.Fatalf("toml.Marshal: %v", err)
	}

	var decoded AgentBinding
	if err := toml.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("toml.Unmarshal: %v\nencoded TOML:\n%s", err, encoded)
	}

	if !reflect.DeepEqual(original, decoded) {
		t.Fatalf("round-trip mismatch\noriginal: %#v\ndecoded:  %#v\nencoded TOML:\n%s", original, decoded, encoded)
	}
}

// TestAgentBindingValidate exhausts each validation rule encoded by
// AgentBinding.Validate per main/PLAN.md § 19.3 lines 1653-1656. Every row
// starts from a fully-populated valid binding and applies a single targeted
// mutation so the failure mode under test is unambiguous.
func TestAgentBindingValidate(t *testing.T) {
	tests := []struct {
		name      string
		mutate    func(*AgentBinding)
		wantValid bool
	}{
		{
			name:      "valid fully populated binding passes",
			mutate:    func(b *AgentBinding) {},
			wantValid: true,
		},
		{
			name:      "missing agent_name rejected",
			mutate:    func(b *AgentBinding) { b.AgentName = "" },
			wantValid: false,
		},
		{
			name:      "whitespace-only agent_name rejected",
			mutate:    func(b *AgentBinding) { b.AgentName = "   \t\n" },
			wantValid: false,
		},
		{
			name:      "missing model rejected",
			mutate:    func(b *AgentBinding) { b.Model = "" },
			wantValid: false,
		},
		{
			name:      "whitespace-only model rejected",
			mutate:    func(b *AgentBinding) { b.Model = "  " },
			wantValid: false,
		},
		{
			name:      "max_tries zero rejected",
			mutate:    func(b *AgentBinding) { b.MaxTries = 0 },
			wantValid: false,
		},
		{
			name:      "max_tries negative rejected",
			mutate:    func(b *AgentBinding) { b.MaxTries = -1 },
			wantValid: false,
		},
		{
			name:      "max_turns zero rejected",
			mutate:    func(b *AgentBinding) { b.MaxTurns = 0 },
			wantValid: false,
		},
		{
			name:      "max_turns negative rejected",
			mutate:    func(b *AgentBinding) { b.MaxTurns = -10 },
			wantValid: false,
		},
		{
			name:      "max_budget_usd negative rejected",
			mutate:    func(b *AgentBinding) { b.MaxBudgetUSD = -1.0 },
			wantValid: false,
		},
		{
			name:      "max_budget_usd zero allowed (means unlimited)",
			mutate:    func(b *AgentBinding) { b.MaxBudgetUSD = 0 },
			wantValid: true,
		},
		{
			name:      "blocked_retries negative rejected",
			mutate:    func(b *AgentBinding) { b.BlockedRetries = -1 },
			wantValid: false,
		},
		{
			name:      "blocked_retries zero allowed",
			mutate:    func(b *AgentBinding) { b.BlockedRetries = 0 },
			wantValid: true,
		},
		{
			name:      "blocked_retry_cooldown negative rejected",
			mutate:    func(b *AgentBinding) { b.BlockedRetryCooldown = Duration(-time.Second) },
			wantValid: false,
		},
		{
			name:      "blocked_retry_cooldown zero allowed",
			mutate:    func(b *AgentBinding) { b.BlockedRetryCooldown = 0 },
			wantValid: true,
		},
		{
			name:      "empty Tools slice allowed (Drop 4 deferral)",
			mutate:    func(b *AgentBinding) { b.Tools = nil },
			wantValid: true,
		},
		{
			name:      "empty Effort allowed (free-form pass-through)",
			mutate:    func(b *AgentBinding) { b.Effort = "" },
			wantValid: true,
		},
		{
			name:      "empty CommitAgent allowed (free-form pass-through)",
			mutate:    func(b *AgentBinding) { b.CommitAgent = "" },
			wantValid: true,
		},
		{
			name:      "AutoPush false allowed (free-form pass-through)",
			mutate:    func(b *AgentBinding) { b.AutoPush = false },
			wantValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			binding := fullyPopulatedAgentBinding()
			tt.mutate(&binding)
			err := binding.Validate()
			if tt.wantValid {
				if err != nil {
					t.Fatalf("Validate() = %v; want nil", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("Validate() = nil; want error")
			}
			if !errors.Is(err, ErrInvalidAgentBinding) {
				t.Fatalf("Validate() = %v; want errors.Is(_, ErrInvalidAgentBinding) true", err)
			}
		})
	}
}

// TestAgentBindingDurationStringWireForm exercises the TOML string-form wire
// path PLAN.md § 19.3 line 333 promises: blocked_retry_cooldown declared as a
// duration string ("30s", "5m") must decode into AgentBinding.BlockedRetryCooldown
// and re-marshal back to the same canonical string. The check is asymmetric on
// purpose — pelletier/go-toml/v2 won't decode a string into a bare
// time.Duration, so this test is the regression bar for the templates.Duration
// wrapper's TextMarshaler / TextUnmarshaler pair.
func TestAgentBindingDurationStringWireForm(t *testing.T) {
	const wireDoc = `agent_name = "go-builder-agent"
model = "opus"
max_tries = 3
max_turns = 50
blocked_retry_cooldown = "30s"
`
	var decoded AgentBinding
	if err := toml.Unmarshal([]byte(wireDoc), &decoded); err != nil {
		t.Fatalf("toml.Unmarshal of duration-string wire form: %v\ndoc:\n%s", err, wireDoc)
	}
	want := Duration(30 * time.Second)
	if decoded.BlockedRetryCooldown != want {
		t.Fatalf("decoded BlockedRetryCooldown = %v; want %v", time.Duration(decoded.BlockedRetryCooldown), time.Duration(want))
	}

	encoded, err := toml.Marshal(decoded)
	if err != nil {
		t.Fatalf("toml.Marshal after decode: %v", err)
	}
	// pelletier/go-toml/v2 emits TOML strings using either basic ("...") or
	// literal ('...') form depending on content; both round-trip identically
	// per the TOML spec. Accept either quote style for the canonical "30s"
	// payload so the test asserts wire-form correctness without coupling to
	// the encoder's quote-style heuristic.
	encodedStr := string(encoded)
	if !strings.Contains(encodedStr, `blocked_retry_cooldown = "30s"`) &&
		!strings.Contains(encodedStr, `blocked_retry_cooldown = '30s'`) {
		t.Fatalf("re-marshaled TOML missing canonical duration string \"30s\" or '30s'\nencoded:\n%s", encoded)
	}

	// Round-trip the re-encoded bytes one more time to confirm the canonical
	// string form decodes back into the same wrapper value the original wire
	// document produced — closes the loop end-to-end.
	var roundTripped AgentBinding
	if err := toml.Unmarshal(encoded, &roundTripped); err != nil {
		t.Fatalf("toml.Unmarshal of re-marshaled doc: %v\nencoded:\n%s", err, encoded)
	}
	if roundTripped.BlockedRetryCooldown != want {
		t.Fatalf("round-tripped BlockedRetryCooldown = %v; want %v", time.Duration(roundTripped.BlockedRetryCooldown), time.Duration(want))
	}
}

// TestAgentBindingValidateZeroValueRejected confirms the zero-value
// AgentBinding fails Validate. The zero value has empty AgentName + Model +
// MaxTries=0 + MaxTurns=0; the first rule to fire (AgentName) wraps
// ErrInvalidAgentBinding, but the assertion only requires that some rule
// fires, not which one.
func TestAgentBindingValidateZeroValueRejected(t *testing.T) {
	var b AgentBinding
	err := b.Validate()
	if err == nil {
		t.Fatalf("Validate() on zero AgentBinding = nil; want error")
	}
	if !errors.Is(err, ErrInvalidAgentBinding) {
		t.Fatalf("Validate() on zero AgentBinding = %v; want errors.Is(_, ErrInvalidAgentBinding) true", err)
	}
}
