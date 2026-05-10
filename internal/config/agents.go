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
	"errors"
	"fmt"
	"os"

	toml "github.com/pelletier/go-toml/v2"

	"github.com/evanmschultz/tillsyn/internal/domain"
)

// ErrToolsDenyNotOverridable is the closed sentinel returned when an
// agents.local.toml registry sets `tools_deny` (in either the [agents]
// defaults block or any per-kind block). Per SKETCH.md § 4.3.1, tools_deny
// is the safety floor: users CANNOT relax denials via .local.toml. D5's
// *ConfigError envelope wraps this sentinel with file/line/block position
// information; D3 raises only the bare sentinel.
//
// Callers inspect the rejection contract via errors.Is(err, ErrToolsDenyNotOverridable).
var ErrToolsDenyNotOverridable = errors.New("tools_deny is not user-overridable; remove the field")

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

// MergeLocal deep-merges the local AgentsRegistry over the project
// AgentsRegistry, returning a fresh registry whose contents reflect both
// inputs per SKETCH.md § 4.3 + § 5. The merge runs at registry-level BEFORE
// Resolve runs at kind-level — order is load-bearing: per-kind blocks in
// local must field-merge into project's per-kind blocks; running Resolve
// first would collapse each side to a flat AgentRuntime and lose the
// pointer-vs-zero discrimination Override carries.
//
// Field-merge semantics:
//
//   - Top-level Preset (concrete fields, no pointer discrimination): zero
//     values in local are treated as "absent" (project survives); non-zero
//     values in local win. Map fields (EnvSet, EnvFromShell) merge per-key
//     with local keys winning on collision. List fields (CliArgs,
//     ToolsAllow, ClaudeMDAddons) full-replace if local sets a non-empty
//     list; empty/nil local list preserves project. Concrete-field semantics
//     necessarily collapse "absent" and "explicit zero" — users who need
//     explicit-zero must use a per-kind override block, which carries
//     pointer-based discrimination.
//   - Per-kind Override blocks (pointer-shaped): local's non-nil pointers
//     win field-by-field over project's pointers; nil pointers preserve
//     project's pointers. Pointer-to-slice / pointer-to-map preserve the
//     explicit-empty-vs-absent distinction inherited from D1.
//
// tools_deny rejection: any non-empty tools_deny in local — whether in the
// [agents] defaults block (Preset.ToolsDeny) or in a per-kind Override
// (Override.ToolsDeny non-nil and non-empty) — returns the bare sentinel
// ErrToolsDenyNotOverridable. D5's envelope wraps this with file/line/block
// position info; D3 surfaces only the sentinel.
//
// MergeLocal(project, nil) returns a deep-cloned copy of project — local
// .toml is optional. MergeLocal(nil, _) returns an error: project agents.toml
// is required per SKETCH § 3.3.
//
// Usage:
//
//	merged, err := MergeLocal(project, local)
//	if err != nil { return err }
//	runtime, err := Resolve(merged, kind)
func MergeLocal(project, local *AgentsRegistry) (*AgentsRegistry, error) {
	if project == nil {
		return nil, fmt.Errorf("MergeLocal: project registry is nil; agents.toml is required")
	}

	// Reject local tools_deny BEFORE merging — fail loud per SKETCH § 4.3.1.
	if local != nil {
		if len(local.Preset.ToolsDeny) > 0 {
			return nil, ErrToolsDenyNotOverridable
		}
		for _, ov := range local.Overrides {
			if ov.ToolsDeny != nil && len(*ov.ToolsDeny) > 0 {
				return nil, ErrToolsDenyNotOverridable
			}
		}
	}

	out := &AgentsRegistry{
		Preset:    project.Preset,
		Overrides: make(map[domain.Kind]Override, len(project.Overrides)),
	}
	// Deep-clone project's Preset map fields so output never aliases inputs.
	out.Preset.EnvSet = copyMap(project.Preset.EnvSet)
	out.Preset.EnvFromShell = copyMap(project.Preset.EnvFromShell)
	out.Preset.CliArgs = copySlice(project.Preset.CliArgs)
	out.Preset.ToolsAllow = copySlice(project.Preset.ToolsAllow)
	out.Preset.ToolsDeny = copySlice(project.Preset.ToolsDeny)
	out.Preset.ClaudeMDAddons = copySlice(project.Preset.ClaudeMDAddons)
	for kind, ov := range project.Overrides {
		out.Overrides[kind] = cloneOverride(ov)
	}

	if local == nil {
		return out, nil
	}

	// Preset field-merge: local non-zero wins; zero treated as absent.
	mergePreset(&out.Preset, local.Preset)

	// Per-kind Override merge: local pointers win field-by-field.
	for kind, lov := range local.Overrides {
		existing, ok := out.Overrides[kind]
		if !ok {
			out.Overrides[kind] = cloneOverride(lov)
			continue
		}
		out.Overrides[kind] = mergeOverride(existing, lov)
	}

	return out, nil
}

// mergePreset overlays local's non-zero Preset fields onto out. Top-level
// Preset uses concrete (non-pointer) fields, so "zero value" is the only
// signal for "absent" available at this layer. Users who need explicit-zero
// override semantics must use per-kind Override blocks.
func mergePreset(out *Preset, local Preset) {
	if local.Client != "" {
		out.Client = local.Client
	}
	if local.Model != "" {
		out.Model = local.Model
	}
	if local.Effort != "" {
		out.Effort = local.Effort
	}
	if local.MaxTries != 0 {
		out.MaxTries = local.MaxTries
	}
	if local.MaxBudgetUSD != 0 {
		out.MaxBudgetUSD = local.MaxBudgetUSD
	}
	if local.MaxTurns != 0 {
		out.MaxTurns = local.MaxTurns
	}
	if local.BlockedRetries != 0 {
		out.BlockedRetries = local.BlockedRetries
	}
	if local.BlockedRetryCooldown != "" {
		out.BlockedRetryCooldown = local.BlockedRetryCooldown
	}
	if local.AutoPush {
		// AutoPush=false in local cannot disable a project-true; documented
		// limitation of concrete-field merge. Per-kind Override carries the
		// pointer discrimination if explicit-false override is needed.
		out.AutoPush = local.AutoPush
	}
	// Maps: merge per-key, local wins on collision.
	if len(local.EnvSet) > 0 {
		if out.EnvSet == nil {
			out.EnvSet = make(map[string]string, len(local.EnvSet))
		}
		for k, v := range local.EnvSet {
			out.EnvSet[k] = v
		}
	}
	if len(local.EnvFromShell) > 0 {
		if out.EnvFromShell == nil {
			out.EnvFromShell = make(map[string]string, len(local.EnvFromShell))
		}
		for k, v := range local.EnvFromShell {
			out.EnvFromShell[k] = v
		}
	}
	// Lists: full-replace if local sets a non-empty list. Empty/nil local
	// preserves project's list (concrete-field layer cannot express
	// "explicit empty replaces non-empty"; per-kind Override carries that).
	if len(local.CliArgs) > 0 {
		out.CliArgs = copySlice(local.CliArgs)
	}
	if len(local.ToolsAllow) > 0 {
		out.ToolsAllow = copySlice(local.ToolsAllow)
	}
	// ToolsDeny: rejected up-front via ErrToolsDenyNotOverridable; never
	// reaches this branch under valid local registries.
	if len(local.ClaudeMDAddons) > 0 {
		out.ClaudeMDAddons = copySlice(local.ClaudeMDAddons)
	}
}

// mergeOverride produces a fresh Override that combines existing (project)
// with local, where local's non-nil pointers win field-by-field. Pointer-vs-
// nil discrimination preserves the absent-vs-explicit-zero semantics from D1.
func mergeOverride(existing, local Override) Override {
	out := cloneOverride(existing)
	if local.Client != nil {
		v := *local.Client
		out.Client = &v
	}
	if local.Model != nil {
		v := *local.Model
		out.Model = &v
	}
	if local.Effort != nil {
		v := *local.Effort
		out.Effort = &v
	}
	if local.MaxTries != nil {
		v := *local.MaxTries
		out.MaxTries = &v
	}
	if local.MaxBudgetUSD != nil {
		v := *local.MaxBudgetUSD
		out.MaxBudgetUSD = &v
	}
	if local.MaxTurns != nil {
		v := *local.MaxTurns
		out.MaxTurns = &v
	}
	if local.BlockedRetries != nil {
		v := *local.BlockedRetries
		out.BlockedRetries = &v
	}
	if local.BlockedRetryCooldown != nil {
		v := *local.BlockedRetryCooldown
		out.BlockedRetryCooldown = &v
	}
	if local.AutoPush != nil {
		v := *local.AutoPush
		out.AutoPush = &v
	}
	if local.EnvSet != nil {
		// Per-key merge into existing map (or fresh map).
		merged := make(map[string]string)
		if out.EnvSet != nil {
			for k, v := range *out.EnvSet {
				merged[k] = v
			}
		}
		for k, v := range *local.EnvSet {
			merged[k] = v
		}
		out.EnvSet = &merged
	}
	if local.EnvFromShell != nil {
		merged := make(map[string]string)
		if out.EnvFromShell != nil {
			for k, v := range *out.EnvFromShell {
				merged[k] = v
			}
		}
		for k, v := range *local.EnvFromShell {
			merged[k] = v
		}
		out.EnvFromShell = &merged
	}
	if local.CliArgs != nil {
		v := copySlice(*local.CliArgs)
		out.CliArgs = &v
	}
	if local.ToolsAllow != nil {
		v := copySlice(*local.ToolsAllow)
		out.ToolsAllow = &v
	}
	// ToolsDeny rejected up-front; never reaches this branch.
	if local.ClaudeMDAddons != nil {
		v := copySlice(*local.ClaudeMDAddons)
		out.ClaudeMDAddons = &v
	}
	return out
}

// cloneOverride returns a deep-clone of ov so the merged AgentsRegistry never
// aliases input pointers. Pointer-shape preserved: a nil pointer in the input
// stays nil in the output, a non-nil pointer is duplicated (fresh underlying
// value, fresh pointer).
func cloneOverride(ov Override) Override {
	out := Override{}
	if ov.Client != nil {
		v := *ov.Client
		out.Client = &v
	}
	if ov.Model != nil {
		v := *ov.Model
		out.Model = &v
	}
	if ov.Effort != nil {
		v := *ov.Effort
		out.Effort = &v
	}
	if ov.MaxTries != nil {
		v := *ov.MaxTries
		out.MaxTries = &v
	}
	if ov.MaxBudgetUSD != nil {
		v := *ov.MaxBudgetUSD
		out.MaxBudgetUSD = &v
	}
	if ov.MaxTurns != nil {
		v := *ov.MaxTurns
		out.MaxTurns = &v
	}
	if ov.BlockedRetries != nil {
		v := *ov.BlockedRetries
		out.BlockedRetries = &v
	}
	if ov.BlockedRetryCooldown != nil {
		v := *ov.BlockedRetryCooldown
		out.BlockedRetryCooldown = &v
	}
	if ov.AutoPush != nil {
		v := *ov.AutoPush
		out.AutoPush = &v
	}
	if ov.EnvSet != nil {
		v := copyMap(*ov.EnvSet)
		out.EnvSet = &v
	}
	if ov.EnvFromShell != nil {
		v := copyMap(*ov.EnvFromShell)
		out.EnvFromShell = &v
	}
	if ov.CliArgs != nil {
		v := copySlice(*ov.CliArgs)
		out.CliArgs = &v
	}
	if ov.ToolsAllow != nil {
		v := copySlice(*ov.ToolsAllow)
		out.ToolsAllow = &v
	}
	if ov.ToolsDeny != nil {
		v := copySlice(*ov.ToolsDeny)
		out.ToolsDeny = &v
	}
	if ov.ClaudeMDAddons != nil {
		v := copySlice(*ov.ClaudeMDAddons)
		out.ClaudeMDAddons = &v
	}
	return out
}

// copySlice returns a fresh slice with the same elements as s, preserving
// nil-vs-empty: nil input returns nil; empty non-nil input returns a fresh
// empty slice (so callers cannot aliase into the input's backing array).
func copySlice(s []string) []string {
	if s == nil {
		return nil
	}
	out := make([]string, len(s))
	copy(out, s)
	return out
}
