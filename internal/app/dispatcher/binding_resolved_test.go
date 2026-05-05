package dispatcher

import (
	"testing"
	"time"

	"github.com/evanmschultz/tillsyn/internal/templates"
)

// binding_resolved_test.go covers the priority-cascade resolver shipped in
// binding_resolved.go for Drop 4c droplet F.7.17.8. The eight test scenarios
// map 1:1 to the spawn prompt's acceptance criteria.

// strPtr / intPtr / float64Ptr / boolPtr / durPtr build short-lived pointer
// literals for table-driven test inputs.
func strPtr(v string) *string               { return &v }
func intPtr(v int) *int                     { return &v }
func float64Ptr(v float64) *float64         { return &v }
func boolPtr(v bool) *bool                  { return &v }
func durPtr(v time.Duration) *time.Duration { return &v }

// rawBindingFixture returns a populated templates.AgentBinding the resolver
// tests use as the bottom-priority layer. Every scalar is non-zero so the
// "no override" fallback path is observable.
func rawBindingFixture() templates.AgentBinding {
	return templates.AgentBinding{
		AgentName:            "go-builder-agent",
		Model:                "opus",
		Effort:               "high",
		Tools:                []string{"Read", "Edit"},
		MaxTries:             3,
		MaxBudgetUSD:         5.0,
		MaxTurns:             50,
		AutoPush:             false,
		CommitAgent:          "commit-agent",
		BlockedRetries:       2,
		BlockedRetryCooldown: templates.Duration(30 * time.Second),
		Env:                  []string{"PATH", "HOME"},
		CLIKind:              "claude",
		ToolsAllowed:         []string{"Read"},
		ToolsDisallowed:      []string{"Bash"},
	}
}

// TestResolveBindingNoOverrides — scenario 1: rawBinding values pass through
// directly when overrides is empty. CLIKind populated explicitly on the raw
// stays as-is (no default substitution).
func TestResolveBindingNoOverrides(t *testing.T) {
	t.Parallel()

	raw := rawBindingFixture()
	got := ResolveBinding(raw)

	if got.AgentName != "go-builder-agent" {
		t.Errorf("AgentName = %q; want %q", got.AgentName, "go-builder-agent")
	}
	if got.CLIKind != CLIKindClaude {
		t.Errorf("CLIKind = %q; want %q", got.CLIKind, CLIKindClaude)
	}
	if got.Model == nil || *got.Model != "opus" {
		t.Errorf("Model = %v; want pointer to %q", got.Model, "opus")
	}
	if got.Effort == nil || *got.Effort != "high" {
		t.Errorf("Effort = %v; want pointer to %q", got.Effort, "high")
	}
	if got.MaxTries == nil || *got.MaxTries != 3 {
		t.Errorf("MaxTries = %v; want pointer to %d", got.MaxTries, 3)
	}
	if got.MaxBudgetUSD == nil || *got.MaxBudgetUSD != 5.0 {
		t.Errorf("MaxBudgetUSD = %v; want pointer to %f", got.MaxBudgetUSD, 5.0)
	}
	if got.MaxTurns == nil || *got.MaxTurns != 50 {
		t.Errorf("MaxTurns = %v; want pointer to %d", got.MaxTurns, 50)
	}
	if got.AutoPush == nil || *got.AutoPush != false {
		t.Errorf("AutoPush = %v; want pointer to false", got.AutoPush)
	}
	if got.BlockedRetries == nil || *got.BlockedRetries != 2 {
		t.Errorf("BlockedRetries = %v; want pointer to %d", got.BlockedRetries, 2)
	}
	if got.BlockedRetryCooldown == nil || *got.BlockedRetryCooldown != 30*time.Second {
		t.Errorf("BlockedRetryCooldown = %v; want pointer to %s", got.BlockedRetryCooldown, 30*time.Second)
	}
	if got.CommitAgent == nil || *got.CommitAgent != "commit-agent" {
		t.Errorf("CommitAgent = %v; want pointer to %q", got.CommitAgent, "commit-agent")
	}
	if len(got.Tools) != 2 || got.Tools[0] != "Read" || got.Tools[1] != "Edit" {
		t.Errorf("Tools = %v; want [Read Edit]", got.Tools)
	}
	if len(got.Env) != 2 || got.Env[0] != "PATH" || got.Env[1] != "HOME" {
		t.Errorf("Env = %v; want [PATH HOME]", got.Env)
	}
	if len(got.ToolsAllowed) != 1 || got.ToolsAllowed[0] != "Read" {
		t.Errorf("ToolsAllowed = %v; want [Read]", got.ToolsAllowed)
	}
	if len(got.ToolsDisallowed) != 1 || got.ToolsDisallowed[0] != "Bash" {
		t.Errorf("ToolsDisallowed = %v; want [Bash]", got.ToolsDisallowed)
	}
}

// TestResolveBindingSingleLayerOverride — scenario 2: a single
// *BindingOverrides with Model = "haiku" wins over the rawBinding's "opus".
// Untouched fields fall through to rawBinding.
func TestResolveBindingSingleLayerOverride(t *testing.T) {
	t.Parallel()

	raw := rawBindingFixture()
	override := &BindingOverrides{Model: strPtr("haiku")}
	got := ResolveBinding(raw, override)

	if got.Model == nil || *got.Model != "haiku" {
		t.Fatalf("Model = %v; want pointer to %q (override should win)", got.Model, "haiku")
	}
	// Untouched fields fall through to rawBinding.
	if got.Effort == nil || *got.Effort != "high" {
		t.Errorf("Effort = %v; want pointer to %q (untouched fall-through)", got.Effort, "high")
	}
	if got.MaxTries == nil || *got.MaxTries != 3 {
		t.Errorf("MaxTries = %v; want pointer to %d (untouched fall-through)", got.MaxTries, 3)
	}
}

// TestResolveBindingMultiLayerPriority — scenario 3: with three layers each
// setting Model differently, the highest-priority (first) layer wins.
func TestResolveBindingMultiLayerPriority(t *testing.T) {
	t.Parallel()

	raw := rawBindingFixture()
	highest := &BindingOverrides{Model: strPtr("haiku")}
	mid := &BindingOverrides{Model: strPtr("sonnet")}
	low := &BindingOverrides{Model: strPtr("opus-mid-tier")}

	got := ResolveBinding(raw, highest, mid, low)

	if got.Model == nil || *got.Model != "haiku" {
		t.Fatalf("Model = %v; want pointer to %q (highest layer wins)", got.Model, "haiku")
	}
}

// TestResolveBindingMixedFieldOverrides — scenario 4: highest sets Model
// only, lowest sets MaxTurns only. Both land in the resolved struct;
// untouched fields fall through to rawBinding.
func TestResolveBindingMixedFieldOverrides(t *testing.T) {
	t.Parallel()

	raw := rawBindingFixture()
	highest := &BindingOverrides{Model: strPtr("haiku")}
	low := &BindingOverrides{MaxTurns: intPtr(99)}

	got := ResolveBinding(raw, highest, low)

	if got.Model == nil || *got.Model != "haiku" {
		t.Errorf("Model = %v; want pointer to %q (highest layer)", got.Model, "haiku")
	}
	if got.MaxTurns == nil || *got.MaxTurns != 99 {
		t.Errorf("MaxTurns = %v; want pointer to %d (lowest layer)", got.MaxTurns, 99)
	}
	// Untouched: Effort still falls back to raw.
	if got.Effort == nil || *got.Effort != "high" {
		t.Errorf("Effort = %v; want pointer to %q (untouched fall-through)", got.Effort, "high")
	}
	if got.MaxTries == nil || *got.MaxTries != 3 {
		t.Errorf("MaxTries = %v; want pointer to %d (untouched fall-through)", got.MaxTries, 3)
	}
}

// TestResolveBindingNilLayerSkipped — scenario 5: nil entries in the
// overrides slice are skipped without panic; non-nil layers contribute their
// values normally.
func TestResolveBindingNilLayerSkipped(t *testing.T) {
	t.Parallel()

	raw := rawBindingFixture()
	override := &BindingOverrides{Model: strPtr("haiku"), MaxTurns: intPtr(77)}

	// [nil, override, nil] — nil entries must NOT panic, and override's
	// values must land.
	got := ResolveBinding(raw, nil, override, nil)

	if got.Model == nil || *got.Model != "haiku" {
		t.Errorf("Model = %v; want pointer to %q", got.Model, "haiku")
	}
	if got.MaxTurns == nil || *got.MaxTurns != 77 {
		t.Errorf("MaxTurns = %v; want pointer to %d", got.MaxTurns, 77)
	}
}

// TestResolveBindingEmptyOverridesSlice — scenario 6: passing no overrides
// at all yields rawBinding values unchanged. Equivalent to scenario 1 but
// asserts the variadic-empty case explicitly (different call shape).
func TestResolveBindingEmptyOverridesSlice(t *testing.T) {
	t.Parallel()

	raw := rawBindingFixture()
	emptyOverrides := []*BindingOverrides{}
	got := ResolveBinding(raw, emptyOverrides...)

	if got.Model == nil || *got.Model != "opus" {
		t.Errorf("Model = %v; want pointer to %q", got.Model, "opus")
	}
	if got.MaxTurns == nil || *got.MaxTurns != 50 {
		t.Errorf("MaxTurns = %v; want pointer to %d", got.MaxTurns, 50)
	}
	if got.AutoPush == nil || *got.AutoPush != false {
		t.Errorf("AutoPush = %v; want pointer to false", got.AutoPush)
	}
}

// TestResolveBindingCLIKindExplicit — scenario 7: rawBinding.CLIKind =
// "claude" passes through verbatim — no default substitution kicks in
// (substitution is only triggered by empty-string).
func TestResolveBindingCLIKindExplicit(t *testing.T) {
	t.Parallel()

	raw := rawBindingFixture()
	raw.CLIKind = "claude"

	got := ResolveBinding(raw)

	if got.CLIKind != CLIKindClaude {
		t.Errorf("CLIKind = %q; want %q (explicit value preserved)", got.CLIKind, CLIKindClaude)
	}
}

// TestResolveBindingCLIKindDefaultsToClaude — scenario 8 (corrected per
// spawn prompt): rawBinding.CLIKind = "" and no overrides → resolved.CLIKind
// = "claude" via the F.7.17 locked decision L15 default substitution.
//
// The spawn prompt's earlier "CLIKind override" scenario was withdrawn
// (BindingOverrides intentionally has no CLIKind field today — see struct
// doc-comment).
func TestResolveBindingCLIKindDefaultsToClaude(t *testing.T) {
	t.Parallel()

	raw := rawBindingFixture()
	raw.CLIKind = ""

	got := ResolveBinding(raw)

	if got.CLIKind != CLIKindClaude {
		t.Errorf("CLIKind = %q; want %q (empty raw → default-to-claude per L15)", got.CLIKind, CLIKindClaude)
	}
}

// TestResolveBindingCommitAgentEmptyToNil — defensive: rawBinding.CommitAgent
// = "" (string zero value) promotes to *string == nil on BindingResolved so
// adapters can distinguish "no commit agent configured" from "explicit
// empty". Non-empty CommitAgent already covered by the scenario-1 fixture.
func TestResolveBindingCommitAgentEmptyToNil(t *testing.T) {
	t.Parallel()

	raw := rawBindingFixture()
	raw.CommitAgent = ""

	got := ResolveBinding(raw)

	if got.CommitAgent != nil {
		t.Errorf("CommitAgent = %v; want nil (empty raw should map to nil pointer)", got.CommitAgent)
	}
}

// TestResolveBindingPointerOverridesPreservedAsCopy — pointer fields on the
// resolved struct must be COPIES of the override values, not aliases. A
// caller mutating the override after ResolveBinding returns must not leak
// into the resolved struct.
func TestResolveBindingPointerOverridesPreservedAsCopy(t *testing.T) {
	t.Parallel()

	raw := rawBindingFixture()
	model := "haiku"
	override := &BindingOverrides{Model: &model}

	got := ResolveBinding(raw, override)

	// Mutate the original override-side value.
	model = "MUTATED"

	if got.Model == nil || *got.Model != "haiku" {
		t.Errorf("Model = %v; want pointer to %q (resolved should be insulated from caller mutation)", got.Model, "haiku")
	}
}

// TestResolveBindingDurationOverride — exercises the time.Duration pointer
// path explicitly. rawBinding has 30s; override sets 5m → resolved is 5m.
func TestResolveBindingDurationOverride(t *testing.T) {
	t.Parallel()

	raw := rawBindingFixture()
	override := &BindingOverrides{BlockedRetryCooldown: durPtr(5 * time.Minute)}

	got := ResolveBinding(raw, override)

	if got.BlockedRetryCooldown == nil || *got.BlockedRetryCooldown != 5*time.Minute {
		t.Errorf("BlockedRetryCooldown = %v; want pointer to %s", got.BlockedRetryCooldown, 5*time.Minute)
	}
}

// TestResolveBindingPureFunctionDoesNotMutateRaw — the resolver MUST NOT
// mutate the rawBinding it is given. Slice fields on rawBinding are passed
// by header value but share underlying storage; the resolver clones them
// defensively so the caller's rawBinding stays intact even if a future
// codepath writes through the resolved struct's slice.
func TestResolveBindingPureFunctionDoesNotMutateRaw(t *testing.T) {
	t.Parallel()

	raw := rawBindingFixture()
	gotResolved := ResolveBinding(raw)

	// Mutate the resolved struct's slice — raw must not see it.
	if len(gotResolved.Tools) > 0 {
		gotResolved.Tools[0] = "MUTATED"
	}
	if raw.Tools[0] != "Read" {
		t.Errorf("rawBinding.Tools[0] = %q; want %q (resolver must clone slice headers)", raw.Tools[0], "Read")
	}
}
