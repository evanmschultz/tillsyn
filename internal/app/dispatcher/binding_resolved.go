package dispatcher

import (
	"time"

	"github.com/evanmschultz/tillsyn/internal/templates"
)

// binding_resolved.go ships the priority-cascade resolver for Drop 4c
// droplet F.7.17.8. It is a PURE FUNCTION over a raw templates.AgentBinding
// and an ordered list of optional override layers, returning the
// BindingResolved that adapters consume via CLIAdapter.BuildCommand.
//
// Cross-references:
//
//   - Type definitions: internal/app/dispatcher/cli_adapter.go (F.7.17.2).
//   - Master plan priority cascade: workflow/drop_4c/F7_17_CLI_ADAPTER_PLAN.md
//     (REVISIONS POST-AUTHORING) — locked decision L9 / L16.
//   - Default-to-claude rule: F.7.17 locked decision L15.

// BindingOverrides carries the upper-priority overrides that should win
// against the binding's template-declared defaults. Each field is a pointer
// so absent ("no override at this layer") is distinguishable from explicit
// zero — falls through to the next layer in the priority cascade per F.7.17
// locked decision L9.
//
// Priority cascade (highest → lowest, master PLAN L9 / L16):
//
//  1. CLI flag overrides (highest)
//  2. MCP arg overrides
//  3. TUI overrides
//  4. Template TOML defaults (rawBinding fields on templates.AgentBinding)
//  5. Absent (zero-value pointer / empty slice / empty string)
//
// Each upper layer is represented as one *BindingOverrides instance;
// ResolveBinding merges them in order, picking the first non-nil value at
// each pointer field. Callers construct the highest-priority overlays
// (CLI/MCP/TUI) and pass them to ResolveBinding alongside the raw binding.
//
// Tools, ToolsAllowed, ToolsDisallowed, Env, AgentName, CLIKind, and
// CommitAgent are NOT yet plumbed for override (only template-level
// declaration). When CLI/MCP/TUI surfaces grow knobs for those, extend the
// struct.
type BindingOverrides struct {
	// Model is the LLM model identifier override (e.g. "opus", "sonnet",
	// "haiku"). Nil means "no override at this layer."
	Model *string

	// Effort is the model effort tier override (e.g. "low", "medium",
	// "high"). Nil means "no override at this layer."
	Effort *string

	// MaxTries overrides the dispatch-attempt cap. Nil means "no override
	// at this layer."
	MaxTries *int

	// MaxBudgetUSD overrides the per-spawn dollar cap. Nil means "no
	// override at this layer." Explicit zero is meaningful ("no spend
	// allowed") and is preserved.
	MaxBudgetUSD *float64

	// MaxTurns overrides the conversation-turn cap. Nil means "no override
	// at this layer." Explicit zero is preserved.
	MaxTurns *int

	// AutoPush overrides the post-build auto-push flag. Nil means "no
	// override at this layer." Explicit false ("never auto-push for this
	// spawn") is preserved.
	AutoPush *bool

	// BlockedRetries overrides the blocked-retry cap. Nil means "no
	// override at this layer." Explicit zero ("never retry") is preserved.
	BlockedRetries *int

	// BlockedRetryCooldown overrides the blocked-retry wall-clock delay.
	// Nil means "no override at this layer." Explicit zero ("retry
	// immediately") is preserved.
	BlockedRetryCooldown *time.Duration
}

// ResolveBinding merges a raw template binding with prioritized overrides
// (highest first) and returns a fully-resolved BindingResolved that adapters
// consume via BuildCommand. Pure function: no I/O, no global state, no side
// effects.
//
// `overrides` is variadic and ordered highest-priority-first. Pass nil for
// layers without overrides at this spawn — nil entries are skipped without
// panic. An empty `overrides` slice (or all-nil) yields the rawBinding values
// unchanged, with the F.7.17 L15 default-to-claude substitution applied to
// CLIKind when rawBinding.CLIKind is empty.
//
// Field handling:
//
//   - Pointer-typed fields on BindingResolved (Model, Effort, MaxTries,
//     MaxBudgetUSD, MaxTurns, AutoPush, BlockedRetries,
//     BlockedRetryCooldown): walk overrides highest→lowest; first non-nil
//     pointer wins. If no override sets the field, use the rawBinding scalar
//     promoted to a pointer (a copy — adapters MUST NOT mutate through the
//     pointer).
//
//   - Slice-typed fields (Tools, Env, ToolsAllowed, ToolsDisallowed): copy
//     verbatim from rawBinding (override plumbing not yet wired).
//
//   - String-typed fields (AgentName): copy verbatim from rawBinding
//     (template-controlled).
//
//   - CommitAgent (*string): promoted from rawBinding.CommitAgent (string).
//     Empty string → nil; non-empty → pointer to a copy.
//
//   - CLIKind: copy from rawBinding; if empty, substitute CLIKindClaude per
//     F.7.17 locked decision L15.
//
//   - BlockedRetryCooldown override: stored as a *time.Duration; the
//     rawBinding's templates.Duration is promoted to *time.Duration via the
//     templates.Duration → time.Duration conversion.
func ResolveBinding(rawBinding templates.AgentBinding, overrides ...*BindingOverrides) BindingResolved {
	resolved := BindingResolved{
		AgentName:       rawBinding.AgentName,
		CLIKind:         CLIKind(rawBinding.CLIKind),
		Env:             cloneStringSlice(rawBinding.Env),
		Tools:           cloneStringSlice(rawBinding.Tools),
		ToolsAllowed:    cloneStringSlice(rawBinding.ToolsAllowed),
		ToolsDisallowed: cloneStringSlice(rawBinding.ToolsDisallowed),
	}

	// F.7.17 locked decision L15: default-to-claude when rawBinding.CLIKind
	// is empty. Override plumbing for CLIKind is not yet wired (would belong
	// on BindingOverrides if/when needed).
	if resolved.CLIKind == "" {
		resolved.CLIKind = CLIKindClaude
	}

	// CommitAgent: promote string → *string. Empty stays nil so adapters
	// can distinguish "no commit agent configured" from "explicit empty".
	if rawBinding.CommitAgent != "" {
		v := rawBinding.CommitAgent
		resolved.CommitAgent = &v
	}

	// Pointer-typed cascade: walk overrides highest→lowest; first non-nil
	// pointer wins. On miss, fall back to the rawBinding scalar promoted to
	// a pointer.
	resolved.Model = resolveStringPtr(rawBinding.Model, overrides, func(o *BindingOverrides) *string { return o.Model })
	resolved.Effort = resolveStringPtr(rawBinding.Effort, overrides, func(o *BindingOverrides) *string { return o.Effort })
	resolved.MaxTries = resolveIntPtr(rawBinding.MaxTries, overrides, func(o *BindingOverrides) *int { return o.MaxTries })
	resolved.MaxBudgetUSD = resolveFloat64Ptr(rawBinding.MaxBudgetUSD, overrides, func(o *BindingOverrides) *float64 { return o.MaxBudgetUSD })
	resolved.MaxTurns = resolveIntPtr(rawBinding.MaxTurns, overrides, func(o *BindingOverrides) *int { return o.MaxTurns })
	resolved.AutoPush = resolveBoolPtr(rawBinding.AutoPush, overrides, func(o *BindingOverrides) *bool { return o.AutoPush })
	resolved.BlockedRetries = resolveIntPtr(rawBinding.BlockedRetries, overrides, func(o *BindingOverrides) *int { return o.BlockedRetries })
	resolved.BlockedRetryCooldown = resolveDurationPtr(time.Duration(rawBinding.BlockedRetryCooldown), overrides, func(o *BindingOverrides) *time.Duration { return o.BlockedRetryCooldown })

	return resolved
}

// resolveStringPtr returns the highest-priority non-nil override pointer
// from `overrides` (ordered highest→lowest), or a pointer to a copy of
// `rawValue` if no layer overrides. The accessor `pick` extracts the
// pointer-field of interest from each override layer. nil entries in
// `overrides` are skipped.
func resolveStringPtr(rawValue string, overrides []*BindingOverrides, pick func(*BindingOverrides) *string) *string {
	for _, o := range overrides {
		if o == nil {
			continue
		}
		if v := pick(o); v != nil {
			vCopy := *v
			return &vCopy
		}
	}
	v := rawValue
	return &v
}

// resolveIntPtr is the int analogue of resolveStringPtr.
func resolveIntPtr(rawValue int, overrides []*BindingOverrides, pick func(*BindingOverrides) *int) *int {
	for _, o := range overrides {
		if o == nil {
			continue
		}
		if v := pick(o); v != nil {
			vCopy := *v
			return &vCopy
		}
	}
	v := rawValue
	return &v
}

// resolveFloat64Ptr is the float64 analogue of resolveStringPtr.
func resolveFloat64Ptr(rawValue float64, overrides []*BindingOverrides, pick func(*BindingOverrides) *float64) *float64 {
	for _, o := range overrides {
		if o == nil {
			continue
		}
		if v := pick(o); v != nil {
			vCopy := *v
			return &vCopy
		}
	}
	v := rawValue
	return &v
}

// resolveBoolPtr is the bool analogue of resolveStringPtr.
func resolveBoolPtr(rawValue bool, overrides []*BindingOverrides, pick func(*BindingOverrides) *bool) *bool {
	for _, o := range overrides {
		if o == nil {
			continue
		}
		if v := pick(o); v != nil {
			vCopy := *v
			return &vCopy
		}
	}
	v := rawValue
	return &v
}

// resolveDurationPtr is the time.Duration analogue of resolveStringPtr.
func resolveDurationPtr(rawValue time.Duration, overrides []*BindingOverrides, pick func(*BindingOverrides) *time.Duration) *time.Duration {
	for _, o := range overrides {
		if o == nil {
			continue
		}
		if v := pick(o); v != nil {
			vCopy := *v
			return &vCopy
		}
	}
	v := rawValue
	return &v
}

// cloneStringSlice returns a defensive copy of `in` so callers cannot mutate
// the rawBinding's slice through the resolved BindingResolved. nil input
// yields nil output (preserves the "no override" identity).
func cloneStringSlice(in []string) []string {
	if in == nil {
		return nil
	}
	out := make([]string, len(in))
	copy(out, in)
	return out
}
