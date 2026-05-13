// Package config — agents.toml schema + decode wiring.
//
// agents.go ships the runtime-config foundation for multi-group agents.toml
// support (Drop 4c.6.1 W0). The schema shifts from the old single-group
// [agents] / [agents.<kind>] layout to a multi-group [<group>] /
// [<group>.<kind>] layout where each group has its own defaults block and
// per-kind overrides.
//
// Key types:
//
//   - [GroupConfig] — per-group config block: default Preset + per-kind Override map.
//   - [AgentsRegistry] — the loaded agents.toml document: map[group]GroupConfig.
//   - [LoadMultiGroupRegistry] — decodes a multi-group agents.toml file.
//   - [Resolve] — returns the effective Preset for a group + kind combination.
//   - [Merge] — deep-merges two AgentsRegistry values; local wins.
//
// TOML schema (multi-group):
//
//	[go]
//	model = "sonnet"
//	tools_allow = ["Read", "Edit"]
//
//	[go.build]
//	model = "sonnet"
//	tools_allow = ["Read", "Edit", "Write", "Bash"]
//
//	[fe]
//	model = "sonnet"
//
//	[fe.build-qa-proof]
//	model = "opus"
//
// Inheritance semantics inside a group (SKETCH.md § 4.2.1-4.2.3):
//   - Scalars: per-kind Override pointer nil → inherit Default value.
//   - Maps (EnvSet, EnvFromShell): per-key merge; Default first, then Override keys.
//   - Lists (CliArgs, ToolsAllow, ToolsDeny, ClaudeMDAddons): full-replace if
//     Override pointer is non-nil (including non-nil empty slice).
//
// Resolver fallback for missing group: returns empty Preset (no panic).
//
// Frontmatter strip helper lives in the sibling frontmatter.go.
// Config envelope errors: [ConfigError] wraps decode + merge failures.
package config

import (
	"bytes"
	"errors"
	"fmt"
	"os"

	toml "github.com/pelletier/go-toml/v2"

	"github.com/evanmschultz/tillsyn/internal/domain"
)

// ErrToolsDenyNotOverridable is the closed sentinel returned when a local
// AgentsRegistry (the first argument to Merge) sets tools_deny in any group
// default block or per-kind block. Per SKETCH.md § 4.3.1, tools_deny is the
// safety floor: users CANNOT relax denials via a local registry.
//
// Callers inspect via errors.Is(err, ErrToolsDenyNotOverridable).
var ErrToolsDenyNotOverridable = errors.New("tools_deny is not user-overridable; remove the field")

// localPathLabel is the canonical user-facing file label for errors raised
// from Merge — independent of the actual on-disk path.
const localPathLabel = "agents.local.toml"

// deterministicKindOrder mirrors the closed 12-value Kind enum sequence in
// internal/domain/kind.go. Used to iterate the per-kind override map in a
// stable order so error messages naming the offending block are reproducible
// across runs — Go's map iteration is intentionally randomized.
var deterministicKindOrder = []domain.Kind{
	domain.KindPlan,
	domain.KindResearch,
	domain.KindBuild,
	domain.KindPlanQAProof,
	domain.KindPlanQAFalsification,
	domain.KindBuildQAProof,
	domain.KindBuildQAFalsification,
	domain.KindCloseout,
	domain.KindCommit,
	domain.KindRefinement,
	domain.KindDiscussion,
	domain.KindHumanVerify,
}

// ConfigError is the unified envelope wrapping every error returned from
// LoadMultiGroupRegistry and Merge. It carries file/block/line position
// context alongside the underlying cause so downstream consumers get a single
// typed error to inspect via errors.As and a stable user-facing format.
//
// Format produced by Error():
//
//	"<file> <block>:<line>: <cause>"
//
// When Block is "" the envelope renders "<file>:<line>: <cause>"; when Line
// is 0 the envelope renders "<file> <block>: <cause>" (or "<file>: <cause>"
// if Block is also empty). The canonical case (every field set) reads e.g.
// "agents.local.toml [go.build]:42: tools_deny is not user-overridable;
// remove the field".
//
// Unwrap returns Cause so errors.Is and errors.As walk transitively.
type ConfigError struct {
	File  string // user-facing file label (e.g. "agents.toml" or "agents.local.toml")
	Block string // TOML table path in bracket form (e.g. "[go.build]"); "" if no block context
	Line  int    // 1-based source line; 0 if unavailable
	Cause error  // wrapped underlying error
}

// Error formats the envelope per the canonical "<file> <block>:<line>: <cause>"
// shape. Empty Block / zero Line gracefully degrade so the format is never
// misleading (e.g. ":0:" appearing in user output).
func (e *ConfigError) Error() string {
	if e == nil {
		return "<nil ConfigError>"
	}
	cause := "<nil cause>"
	if e.Cause != nil {
		cause = e.Cause.Error()
	}

	switch {
	case e.Block != "" && e.Line > 0:
		return fmt.Sprintf("%s %s:%d: %s", e.File, e.Block, e.Line, cause)
	case e.Block != "" && e.Line == 0:
		return fmt.Sprintf("%s %s: %s", e.File, e.Block, cause)
	case e.Block == "" && e.Line > 0:
		return fmt.Sprintf("%s:%d: %s", e.File, e.Line, cause)
	default:
		return fmt.Sprintf("%s: %s", e.File, cause)
	}
}

// Unwrap returns the wrapped cause so errors.Is / errors.As walk transitively.
func (e *ConfigError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

// keyToBlock formats a pelletier/go-toml/v2 Key (a []string of dotted path
// segments) into the canonical "[seg.seg.seg]" bracket form used by
// ConfigError.Block. Empty key returns "" so callers can omit the block
// component for top-level syntax errors that don't resolve to any key.
//
// Example: Key{"go", "build"} → "[go.build]";  Key{} → "".
func keyToBlock(key toml.Key) string {
	if len(key) == 0 {
		return ""
	}
	out := "["
	for i, seg := range key {
		if i > 0 {
			out += "."
		}
		out += seg
	}
	out += "]"
	return out
}

// Preset captures the group-level defaults block in agents.toml. Every field is
// a concrete value (not a pointer) — Preset is the floor that per-kind
// Override pointers fall through to in Resolve. Field naming follows
// PascalCase Go convention with snake_case TOML keys.
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
// from "explicit zero value override" (non-nil pointer to zero).
//
// Map fields (EnvSet, EnvFromShell) keep *map rather than just map because
// nil-map vs empty-map carries semantic weight: nil = "inherit", non-nil
// empty = "explicitly drop all defaults."
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

// GroupConfig is the per-group configuration: a Default Preset that applies to
// all kinds in the group, and a Kinds map of per-kind Override blocks that
// layer on top of Default in Resolve.
//
// Kinds is always non-nil after a successful LoadMultiGroupRegistry — absent
// per-kind blocks simply do not appear as keys. A GroupConfig constructed in
// code with nil Kinds is treated as if it has an empty Kinds map (Resolve
// handles nil safely).
type GroupConfig struct {
	// Default is the group-level defaults block (the [<group>] TOML section).
	Default Preset
	// Kinds is the per-kind override map (the [<group>.<kind>] TOML sections).
	// Keyed by domain.Kind.
	Kinds map[domain.Kind]Override
}

// AgentsRegistry is the loaded agents.toml document: a map from group name to
// GroupConfig. The map is always non-nil after a successful
// LoadMultiGroupRegistry — absent groups simply do not appear as keys.
//
// An AgentsRegistry constructed with a nil map is valid for reads (Go nil-map
// reads return zero values) but callers should prefer an initialized empty
// registry (use make(AgentsRegistry) or LoadMultiGroupRegistry).
type AgentsRegistry map[string]GroupConfig

// agentsTOMLGroupBlock embeds Preset so the [<group>] block's scalar/map/list
// fields decode at this level, while [<group>.<kind>] subtables decode into
// the per-kind pointer fields. Each kind gets its own typed field rather than
// a map[string]Override so DisallowUnknownFields() rejects typos in kind names
// at decode time — silent drop on unknown kind names would be a serious
// user-experience regression.
//
// Adding a new kind requires updating the closed enum in
// internal/domain/kind.go AND adding the matching field here.
type agentsTOMLGroupBlock struct {
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

// LoadMultiGroupRegistry reads and decodes a multi-group agents.toml file at
// path. The decoder is strict: unknown fields within each group block are
// rejected so user-typos in field names fail loud rather than silently drop.
// Group names (map keys) are user-defined and are not validated against a
// closed enum.
//
// Returns a *ConfigError envelope wrapping the underlying *toml.DecodeError on
// malformed input — callers can recover the inner DecodeError via
// errors.As(err, &*toml.DecodeError) and the envelope itself via
// errors.As(err, &*ConfigError) for File/Block/Line position info.
// File-read failures (path missing, permission denied) are returned via
// fmt.Errorf("%w") rather than the envelope because they have no source-
// position to report.
//
// A nil registry is never returned alongside a nil error. On any error, the
// returned registry is nil.
func LoadMultiGroupRegistry(path string) (AgentsRegistry, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read agents.toml at %q: %w", path, err)
	}

	var raw map[string]agentsTOMLGroupBlock
	dec := toml.NewDecoder(bytes.NewReader(content))
	dec = dec.DisallowUnknownFields()
	if err := dec.Decode(&raw); err != nil {
		var decodeErr *toml.DecodeError
		if errors.As(err, &decodeErr) {
			row, _ := decodeErr.Position()
			return nil, &ConfigError{
				File:  path,
				Block: keyToBlock(decodeErr.Key()),
				Line:  row,
				Cause: decodeErr,
			}
		}
		return nil, &ConfigError{
			File:  path,
			Cause: err,
		}
	}

	registry := make(AgentsRegistry, len(raw))
	for group, block := range raw {
		gc := GroupConfig{
			Default: block.Preset,
			Kinds:   make(map[domain.Kind]Override, 12),
		}
		addOverride(gc.Kinds, domain.KindPlan, block.Plan)
		addOverride(gc.Kinds, domain.KindResearch, block.Research)
		addOverride(gc.Kinds, domain.KindBuild, block.Build)
		addOverride(gc.Kinds, domain.KindPlanQAProof, block.PlanQAProof)
		addOverride(gc.Kinds, domain.KindPlanQAFalsification, block.PlanQAFalsification)
		addOverride(gc.Kinds, domain.KindBuildQAProof, block.BuildQAProof)
		addOverride(gc.Kinds, domain.KindBuildQAFalsification, block.BuildQAFalsification)
		addOverride(gc.Kinds, domain.KindCloseout, block.Closeout)
		addOverride(gc.Kinds, domain.KindCommit, block.Commit)
		addOverride(gc.Kinds, domain.KindRefinement, block.Refinement)
		addOverride(gc.Kinds, domain.KindDiscussion, block.Discussion)
		addOverride(gc.Kinds, domain.KindHumanVerify, block.HumanVerify)
		registry[group] = gc
	}
	return registry, nil
}

// addOverride records the override in the per-kind map only when the
// pointer is non-nil, i.e. the user actually provided a [<group>.<kind>]
// block. Absent blocks remain absent in the map — Resolve treats a missing
// key as "no override, inherit Default wholesale."
func addOverride(out map[domain.Kind]Override, kind domain.Kind, ov *Override) {
	if ov == nil {
		return
	}
	out[kind] = *ov
}

// Resolve produces the effective Preset for the given group + kind by merging
// registry[group].Kinds[kind] over registry[group].Default per the inheritance
// contract in SKETCH.md § 4.2.1-4.2.3:
//
//   - Scalar fields (string / int / float / bool): if the Override pointer is
//     nil the Default value is used; otherwise the dereferenced override value
//     wins (even if it is the zero value of the type).
//   - Map fields (EnvSet, EnvFromShell): per-key merge. The Default map is
//     copied first; then each key in the override map is written into the copy,
//     overwriting Default entries on collision. Override-nil leaves the Default
//     map intact. Output is always a fresh map.
//   - List fields (CliArgs, ToolsAllow, ToolsDeny, ClaudeMDAddons): full
//     replace if the override pointer is non-nil; inherit Default otherwise.
//     A non-nil empty slice replaces a non-empty Default list with an empty
//     slice — load-bearing for users who need to explicitly drop a default.
//
// Missing group: if registry does not contain the requested group, Resolve
// returns an empty Preset (no panic, no error). Callers requiring a group to
// exist should validate before calling Resolve.
//
// Missing kind override: returns the group's Default values verbatim (pure
// inheritance). This includes a registry with nil Kinds map.
//
// Resolve currently never returns a non-nil error; the (Preset, error)
// signature is reserved for future per-field validators. Callers should still
// wire errors.Is checks for forward-compat.
func Resolve(registry AgentsRegistry, group, kind string) (Preset, error) {
	gc, ok := registry[group]
	if !ok {
		// Unknown group — return empty Preset; no error.
		return Preset{}, nil
	}

	// Start from Default values — every field is the floor.
	out := Preset{
		Client:               gc.Default.Client,
		Model:                gc.Default.Model,
		Effort:               gc.Default.Effort,
		MaxTries:             gc.Default.MaxTries,
		MaxBudgetUSD:         gc.Default.MaxBudgetUSD,
		MaxTurns:             gc.Default.MaxTurns,
		BlockedRetries:       gc.Default.BlockedRetries,
		BlockedRetryCooldown: gc.Default.BlockedRetryCooldown,
		AutoPush:             gc.Default.AutoPush,
		EnvSet:               copyMap(gc.Default.EnvSet),
		EnvFromShell:         copyMap(gc.Default.EnvFromShell),
		CliArgs:              gc.Default.CliArgs,
		ToolsAllow:           gc.Default.ToolsAllow,
		ToolsDeny:            gc.Default.ToolsDeny,
		ClaudeMDAddons:       gc.Default.ClaudeMDAddons,
	}

	if gc.Kinds == nil {
		return out, nil
	}

	ov, ok := gc.Kinds[domain.Kind(kind)]
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

	// Maps: per-key merge. Default already copied above; layer override keys.
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
// fresh map they can mutate without aliasing into Default's storage. Returns
// nil for nil input — preserves the absent-vs-empty distinction at the
// Preset boundary.
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

// Merge deep-merges local on top of project, returning a fresh AgentsRegistry
// whose contents reflect both inputs. The merge runs at registry-level BEFORE
// Resolve runs at kind-level — order is load-bearing: per-kind blocks in local
// must field-merge into project's per-kind blocks; running Resolve first would
// collapse each side to a flat Preset and lose the pointer-vs-zero Override
// discrimination.
//
// Per-group merge semantics:
//
//   - Group exists in both local and project: merge Default fields (local
//     non-zero wins; zero treated as absent) + merge Kinds map field-by-field
//     (local Override pointers win over project pointers).
//   - Group only in local: the local GroupConfig is deep-cloned into the output.
//   - Group only in project: the project GroupConfig is deep-cloned into the output.
//
// Default field-merge semantics:
//
//   - Top-level Default uses concrete (non-pointer) Preset fields. Zero values
//     in local.Default are treated as "absent" (project survives); non-zero
//     values in local.Default win. Map fields (EnvSet, EnvFromShell) merge
//     per-key with local keys winning on collision. List fields (CliArgs,
//     ToolsAllow, ClaudeMDAddons) full-replace if local sets a non-empty list.
//
// Per-kind Override merge: local's non-nil pointers win field-by-field over
// project's pointers. Pointer-to-slice / pointer-to-map preserve the
// explicit-empty-vs-absent distinction.
//
// tools_deny rejection: any non-empty tools_deny in local — whether in a group
// Default (Preset.ToolsDeny) or in a per-kind Override (Override.ToolsDeny
// non-nil and non-empty) — returns the bare sentinel ErrToolsDenyNotOverridable
// wrapped in a *ConfigError envelope with the offending block context.
//
// Merge(nil, project) returns a deep-clone of project — local is optional.
// Merge(local, nil) returns a deep-clone of local — project is optional.
// Merge(nil, nil) returns an initialized empty AgentsRegistry (no error).
func Merge(local, project AgentsRegistry) (AgentsRegistry, error) {
	// Validate local for tools_deny BEFORE merging — fail loud per SKETCH § 4.3.1.
	if local != nil {
		if err := rejectLocalToolsDeny(local); err != nil {
			return nil, err
		}
	}

	out := make(AgentsRegistry)

	// Clone project into output first.
	for group, gc := range project {
		out[group] = cloneGroupConfig(gc)
	}

	// Merge local on top.
	for group, localGC := range local {
		existing, ok := out[group]
		if !ok {
			// Group only in local: clone into output.
			out[group] = cloneGroupConfig(localGC)
			continue
		}
		// Group in both: merge Default + Kinds.
		mergePreset(&existing.Default, localGC.Default)
		for kind, lov := range localGC.Kinds {
			existingOv, ok := existing.Kinds[kind]
			if !ok {
				existing.Kinds[kind] = cloneOverride(lov)
				continue
			}
			existing.Kinds[kind] = mergeOverride(existingOv, lov)
		}
		out[group] = existing
	}

	return out, nil
}

// rejectLocalToolsDeny checks all groups in the local registry for tools_deny
// entries and returns the first *ConfigError wrapping ErrToolsDenyNotOverridable
// it finds. Returns nil if no violations are found.
//
// Iterates groups in sorted order for deterministic error messages.
func rejectLocalToolsDeny(local AgentsRegistry) error {
	// Sort group names for deterministic iteration.
	groups := make([]string, 0, len(local))
	for g := range local {
		groups = append(groups, g)
	}
	sortStrings(groups)

	for _, group := range groups {
		gc := local[group]
		if len(gc.Default.ToolsDeny) > 0 {
			return &ConfigError{
				File:  localPathLabel,
				Block: "[" + group + "]",
				Cause: ErrToolsDenyNotOverridable,
			}
		}
		for _, kind := range deterministicKindOrder {
			ov, ok := gc.Kinds[kind]
			if !ok {
				continue
			}
			if ov.ToolsDeny != nil && len(*ov.ToolsDeny) > 0 {
				return &ConfigError{
					File:  localPathLabel,
					Block: "[" + group + "." + string(kind) + "]",
					Cause: ErrToolsDenyNotOverridable,
				}
			}
		}
	}
	return nil
}

// sortStrings sorts ss in-place using a simple insertion-style loop. Avoids
// importing sort for a small slice common in this package.
func sortStrings(ss []string) {
	for i := 1; i < len(ss); i++ {
		for j := i; j > 0 && ss[j] < ss[j-1]; j-- {
			ss[j], ss[j-1] = ss[j-1], ss[j]
		}
	}
}

// cloneGroupConfig returns a deep-clone of gc so the merged AgentsRegistry
// never aliases input maps.
func cloneGroupConfig(gc GroupConfig) GroupConfig {
	out := GroupConfig{
		Default: gc.Default,
		Kinds:   make(map[domain.Kind]Override, len(gc.Kinds)),
	}
	// Deep-clone Default map fields.
	out.Default.EnvSet = copyMap(gc.Default.EnvSet)
	out.Default.EnvFromShell = copyMap(gc.Default.EnvFromShell)
	out.Default.CliArgs = copySlice(gc.Default.CliArgs)
	out.Default.ToolsAllow = copySlice(gc.Default.ToolsAllow)
	out.Default.ToolsDeny = copySlice(gc.Default.ToolsDeny)
	out.Default.ClaudeMDAddons = copySlice(gc.Default.ClaudeMDAddons)
	// Deep-clone Kinds.
	for kind, ov := range gc.Kinds {
		out.Kinds[kind] = cloneOverride(ov)
	}
	return out
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
	// preserves project's list.
	if len(local.CliArgs) > 0 {
		out.CliArgs = copySlice(local.CliArgs)
	}
	if len(local.ToolsAllow) > 0 {
		out.ToolsAllow = copySlice(local.ToolsAllow)
	}
	// ToolsDeny: rejected up-front via ErrToolsDenyNotOverridable in
	// rejectLocalToolsDeny; never reaches this branch under valid local registries.
	if len(local.ClaudeMDAddons) > 0 {
		out.ClaudeMDAddons = copySlice(local.ClaudeMDAddons)
	}
}

// mergeOverride produces a fresh Override that combines existing (project)
// with local, where local's non-nil pointers win field-by-field.
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
// aliases input pointers.
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
// empty slice.
func copySlice(s []string) []string {
	if s == nil {
		return nil
	}
	out := make([]string, len(s))
	copy(out, s)
	return out
}
