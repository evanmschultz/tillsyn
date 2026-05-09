// Package config — agents.toml schema + decode wiring.
//
// agents.go ships the runtime-config foundation for Drop 4c.6 W0: the
// `Preset` defaults block, per-kind `Override` partial-shape struct, the
// merged-result `AgentRuntime`, and the loaded `AgentsRegistry`. Subsequent
// W0 droplets (D2 Resolve, D3 MergeLocal, D5 ConfigError envelope) layer
// inheritance, local-file deep-merge, and position-aware error wrapping
// atop the types defined here. Frontmatter strip helper lives in the sibling
// `frontmatter.go` (D4).
//
// Schema source of truth: SKETCH.md § 4.1 (defaults) + § 4.2 (per-kind
// overrides) + § 4.2.1-4.2.3 (inheritance semantics — applied in D2). The
// pointer-based Override discriminates "absent" (nil) from "explicit zero
// value" — load-bearing for D2's resolver.
package config

import (
	"bytes"
	"fmt"
	"os"

	toml "github.com/pelletier/go-toml/v2"

	"github.com/evanmschultz/tillsyn/internal/domain"
)

// Preset captures the [agents] defaults block in agents.toml. Every field is
// a concrete value (not a pointer) — Preset is the floor that per-kind
// Override pointers fall through to in D2's Resolve. Field naming follows
// PascalCase Go convention with snake_case TOML keys.
//
// SKETCH.md § 4.1 schema ordering preserved: client/model/effort first
// (identity), then caps (max_tries / max_budget_usd / max_turns / blocked_*),
// then auto_push, then env maps, then list-typed knobs.
type Preset struct {
	Client               string            `toml:"client"`
	Model                string            `toml:"model"`
	Effort               string            `toml:"effort"`
	MaxTries             int               `toml:"max_tries"`
	MaxBudgetUSD         float64           `toml:"max_budget_usd"`
	MaxTurns             int               `toml:"max_turns"`
	BlockedRetries       int               `toml:"blocked_retries"`
	BlockedRetryCooldown string            `toml:"blocked_retry_cooldown"`
	AutoPush             bool              `toml:"auto_push"`
	EnvSet               map[string]string `toml:"env_set"`
	EnvFromShell         map[string]string `toml:"env_from_shell"`
	CliArgs              []string          `toml:"cli_args"`
	ToolsAllow           []string          `toml:"tools_allow"`
	ToolsDeny            []string          `toml:"tools_deny"`
	ClaudeMDAddons       []string          `toml:"claude_md_addons"`
}

// Override is a per-kind partial-shape mirror of Preset. Every field is a
// pointer so callers can distinguish "absent — inherit from Preset" (nil)
// from "explicit zero value override" (non-nil pointer to zero). D2's
// Resolve walks this 1-1 correspondence to merge.
//
// Map fields (EnvSet, EnvFromShell) keep `*map` rather than just `map`
// because nil-map vs empty-map carries semantic weight: nil = "inherit",
// non-nil empty = "explicitly drop all defaults" (the latter only meaningful
// once D2 lands and is documented there).
type Override struct {
	Client               *string            `toml:"client"`
	Model                *string            `toml:"model"`
	Effort               *string            `toml:"effort"`
	MaxTries             *int               `toml:"max_tries"`
	MaxBudgetUSD         *float64           `toml:"max_budget_usd"`
	MaxTurns             *int               `toml:"max_turns"`
	BlockedRetries       *int               `toml:"blocked_retries"`
	BlockedRetryCooldown *string            `toml:"blocked_retry_cooldown"`
	AutoPush             *bool              `toml:"auto_push"`
	EnvSet               *map[string]string `toml:"env_set"`
	EnvFromShell         *map[string]string `toml:"env_from_shell"`
	CliArgs              *[]string          `toml:"cli_args"`
	ToolsAllow           *[]string          `toml:"tools_allow"`
	ToolsDeny            *[]string          `toml:"tools_deny"`
	ClaudeMDAddons       *[]string          `toml:"claude_md_addons"`
}

// AgentRuntime is the effective per-kind config produced by D2's Resolve.
// Same field set as Preset because every Override field falls through to a
// Preset default at resolution time. Adapters (e.g. dispatcher CLI builder)
// consume AgentRuntime, never the raw registry.
type AgentRuntime struct {
	Client               string
	Model                string
	Effort               string
	MaxTries             int
	MaxBudgetUSD         float64
	MaxTurns             int
	BlockedRetries       int
	BlockedRetryCooldown string
	AutoPush             bool
	EnvSet               map[string]string
	EnvFromShell         map[string]string
	CliArgs              []string
	ToolsAllow           []string
	ToolsDeny            []string
	ClaudeMDAddons       []string
}

// AgentsRegistry is the loaded agents.toml document: the [agents] defaults
// block plus the map of per-kind override blocks keyed by domain.Kind. The
// map is always non-nil after a successful LoadRegistry — absent per-kind
// blocks simply do not appear as keys.
type AgentsRegistry struct {
	Preset    Preset
	Overrides map[domain.Kind]Override
}

// agentsTOMLRoot is the on-disk shape pelletier/go-toml/v2 decodes into.
// The [agents] block decodes into Agents; nested [agents.<kind>] subtables
// decode into the per-kind pointer fields below.
type agentsTOMLRoot struct {
	Agents agentsTOMLBlock `toml:"agents"`
}

// agentsTOMLBlock embeds Preset so the [agents] block's scalar/map/list
// fields decode at this level, while [agents.<kind>] subtables decode into
// the per-kind pointer fields. Each kind gets its own typed field rather
// than a `map[string]Override` so DisallowUnknownFields() rejects typos in
// kind names at decode time — silent drop on unknown kind names would be
// a serious user-experience regression.
//
// Adding a new kind requires updating the closed enum in
// internal/domain/kind.go AND adding the matching field here.
type agentsTOMLBlock struct {
	Preset
	Plan                 *Override `toml:"plan"`
	Research             *Override `toml:"research"`
	Build                *Override `toml:"build"`
	PlanQAProof          *Override `toml:"plan-qa-proof"`
	PlanQAFalsification  *Override `toml:"plan-qa-falsification"`
	BuildQAProof         *Override `toml:"build-qa-proof"`
	BuildQAFalsification *Override `toml:"build-qa-falsification"`
	Closeout             *Override `toml:"closeout"`
	Commit               *Override `toml:"commit"`
	Refinement           *Override `toml:"refinement"`
	Discussion           *Override `toml:"discussion"`
	HumanVerify          *Override `toml:"human-verify"`
}

// LoadRegistry reads and decodes an agents.toml file at path. The decoder is
// strict: unknown top-level fields are rejected so user-typos in field names
// fail loud rather than silently drop. Returns a position-aware
// *toml.DecodeError (wrapped via fmt.Errorf with %w) on malformed input —
// callers can recover row/column via errors.As. D5 wraps DecodeError into
// the unified ConfigError envelope; pre-D5 callers see the raw DecodeError
// in the chain.
//
// A nil registry is never returned alongside a nil error. On any error, the
// returned registry is nil.
func LoadRegistry(path string) (*AgentsRegistry, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read agents.toml at %q: %w", path, err)
	}

	var root agentsTOMLRoot
	dec := toml.NewDecoder(bytes.NewReader(content))
	dec = dec.DisallowUnknownFields()
	if err := dec.Decode(&root); err != nil {
		return nil, fmt.Errorf("decode agents.toml at %q: %w", path, err)
	}

	overrides := make(map[domain.Kind]Override, 12)
	addOverride(overrides, domain.KindPlan, root.Agents.Plan)
	addOverride(overrides, domain.KindResearch, root.Agents.Research)
	addOverride(overrides, domain.KindBuild, root.Agents.Build)
	addOverride(overrides, domain.KindPlanQAProof, root.Agents.PlanQAProof)
	addOverride(overrides, domain.KindPlanQAFalsification, root.Agents.PlanQAFalsification)
	addOverride(overrides, domain.KindBuildQAProof, root.Agents.BuildQAProof)
	addOverride(overrides, domain.KindBuildQAFalsification, root.Agents.BuildQAFalsification)
	addOverride(overrides, domain.KindCloseout, root.Agents.Closeout)
	addOverride(overrides, domain.KindCommit, root.Agents.Commit)
	addOverride(overrides, domain.KindRefinement, root.Agents.Refinement)
	addOverride(overrides, domain.KindDiscussion, root.Agents.Discussion)
	addOverride(overrides, domain.KindHumanVerify, root.Agents.HumanVerify)

	return &AgentsRegistry{
		Preset:    root.Agents.Preset,
		Overrides: overrides,
	}, nil
}

// addOverride records the override in the per-kind map only when the
// pointer is non-nil, i.e. the user actually provided a [agents.<kind>]
// block. Absent blocks remain absent in the map — D2's Resolve treats a
// missing key as "no override, inherit Preset wholesale."
func addOverride(out map[domain.Kind]Override, kind domain.Kind, ov *Override) {
	if ov == nil {
		return
	}
	out[kind] = *ov
}

// Resolve produces the effective per-kind AgentRuntime by merging
// registry.Overrides[kind] over registry.Preset per the inheritance contract
// in SKETCH.md § 4.2.1-4.2.3:
//
//   - Scalar fields (string / int / float / bool): if the Override pointer
//     is nil the Preset value is used; otherwise the dereferenced override
//     value wins (even if it is the zero value of the type — pointer-vs-
//     dereference carries the absent-vs-zero discrimination).
//   - Map fields (EnvSet, EnvFromShell): per-key merge. The Preset map is
//     copied first; then each key in the override map is written into the
//     copy, overwriting Preset entries on collision. Override-nil leaves
//     the Preset map intact; override-empty contributes zero keys (so the
//     resulting map equals the Preset map). Output is always a fresh map
//     so callers cannot mutate Preset's storage through the AgentRuntime.
//   - List fields (CliArgs, ToolsAllow, ToolsDeny, ClaudeMDAddons): full
//     replace if the override pointer is non-nil; inherit Preset otherwise.
//     A non-nil empty slice (e.g. &[]string{}) replaces a non-empty Preset
//     list with an empty slice — load-bearing for users who need to
//     explicitly drop a default. Returned slice is the override's slice
//     directly (no defensive copy); mutation by the caller is out of
//     scope today, but D5's envelope or a future hardening pass may copy.
//
// A registry whose Overrides map has no entry for kind (or has the zero
// Override) returns the Preset values verbatim — pure inheritance.
//
// Resolve currently never returns a non-nil error; the (AgentRuntime, error)
// signature is reserved for D5's ConfigError envelope and future per-field
// validators (e.g. unknown model name on a per-kind block). Callers that
// strictly need an error-free resolution can use the result and ignore err
// today, but should still wire errors.Is checks for forward-compat.
func Resolve(registry *AgentsRegistry, kind domain.Kind) (AgentRuntime, error) {
	if registry == nil {
		return AgentRuntime{}, fmt.Errorf("Resolve: registry is nil")
	}

	// Start from Preset values — every field is the floor.
	out := AgentRuntime{
		Client:               registry.Preset.Client,
		Model:                registry.Preset.Model,
		Effort:               registry.Preset.Effort,
		MaxTries:             registry.Preset.MaxTries,
		MaxBudgetUSD:         registry.Preset.MaxBudgetUSD,
		MaxTurns:             registry.Preset.MaxTurns,
		BlockedRetries:       registry.Preset.BlockedRetries,
		BlockedRetryCooldown: registry.Preset.BlockedRetryCooldown,
		AutoPush:             registry.Preset.AutoPush,
		EnvSet:               copyMap(registry.Preset.EnvSet),
		EnvFromShell:         copyMap(registry.Preset.EnvFromShell),
		CliArgs:              registry.Preset.CliArgs,
		ToolsAllow:           registry.Preset.ToolsAllow,
		ToolsDeny:            registry.Preset.ToolsDeny,
		ClaudeMDAddons:       registry.Preset.ClaudeMDAddons,
	}

	ov, ok := registry.Overrides[kind]
	if !ok {
		// No per-kind block: pure inheritance.
		return out, nil
	}

	// Scalars: nil pointer = inherit, non-nil = override.
	if ov.Client != nil {
		out.Client = *ov.Client
	}
	if ov.Model != nil {
		out.Model = *ov.Model
	}
	if ov.Effort != nil {
		out.Effort = *ov.Effort
	}
	if ov.MaxTries != nil {
		out.MaxTries = *ov.MaxTries
	}
	if ov.MaxBudgetUSD != nil {
		out.MaxBudgetUSD = *ov.MaxBudgetUSD
	}
	if ov.MaxTurns != nil {
		out.MaxTurns = *ov.MaxTurns
	}
	if ov.BlockedRetries != nil {
		out.BlockedRetries = *ov.BlockedRetries
	}
	if ov.BlockedRetryCooldown != nil {
		out.BlockedRetryCooldown = *ov.BlockedRetryCooldown
	}
	if ov.AutoPush != nil {
		out.AutoPush = *ov.AutoPush
	}

	// Maps: per-key merge. Preset already copied above; layer override keys.
	if ov.EnvSet != nil {
		if out.EnvSet == nil {
			out.EnvSet = make(map[string]string, len(*ov.EnvSet))
		}
		for k, v := range *ov.EnvSet {
			out.EnvSet[k] = v
		}
	}
	if ov.EnvFromShell != nil {
		if out.EnvFromShell == nil {
			out.EnvFromShell = make(map[string]string, len(*ov.EnvFromShell))
		}
		for k, v := range *ov.EnvFromShell {
			out.EnvFromShell[k] = v
		}
	}

	// Lists: full replace if non-nil (including non-nil empty).
	if ov.CliArgs != nil {
		out.CliArgs = *ov.CliArgs
	}
	if ov.ToolsAllow != nil {
		out.ToolsAllow = *ov.ToolsAllow
	}
	if ov.ToolsDeny != nil {
		out.ToolsDeny = *ov.ToolsDeny
	}
	if ov.ClaudeMDAddons != nil {
		out.ClaudeMDAddons = *ov.ClaudeMDAddons
	}

	return out, nil
}

// copyMap returns a shallow copy of m. Used by Resolve to give the caller a
// fresh map they can mutate without aliasing into Preset's storage. Returns
// nil for nil input — preserves the absent-vs-empty distinction at the
// AgentRuntime boundary.
func copyMap(m map[string]string) map[string]string {
	if m == nil {
		return nil
	}
	out := make(map[string]string, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}
