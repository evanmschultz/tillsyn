// Package templates defines the closed TOML schema for cascade-template
// definitions. The schema is the authoritative wire format for binding
// action-item kinds to agents, encoding parent/child kind constraints, and
// describing the cascade's auto-create rules.
//
// This file ships pure type definitions only. The TOML loader, schema-version
// gate, structural validators, and KindCatalog bake-out land in subsequent
// droplets (3.9 parser; 3.10 validation; 3.11 ChildRulesFor; 3.12 catalog).
//
// Canonical sources:
//   - main/PLAN.md § 19.3 — closed schema specification.
//   - ta-docs/cascade-methodology.md §11 — structural-type axis the schema
//     binds against.
package templates

import (
	"fmt"
	"strings"
	"time"

	"github.com/evanmschultz/tillsyn/internal/domain"
)

// SchemaVersionV1 is the locked initial schema-version string. Templates that
// declare any other schema_version are rejected at load time (3.9). Forward-
// compat bumps land as new constants ("v2", ...) alongside their migration
// gate.
const SchemaVersionV1 = "v1"

// Duration is a time.Duration that round-trips as a TOML / JSON string
// via encoding.TextMarshaler / encoding.TextUnmarshaler. Strings parse
// per time.ParseDuration ("30s", "5m", "1h30m", etc.).
type Duration time.Duration

// MarshalText implements encoding.TextMarshaler.
func (d Duration) MarshalText() ([]byte, error) {
	return []byte(time.Duration(d).String()), nil
}

// UnmarshalText implements encoding.TextUnmarshaler.
func (d *Duration) UnmarshalText(text []byte) error {
	parsed, err := time.ParseDuration(string(text))
	if err != nil {
		return err
	}
	*d = Duration(parsed)
	return nil
}

// Template is the closed-schema root for a cascade-template definition. It
// pairs a schema_version pin with the closed kind, child-rule, and
// agent-binding tables.
//
// The struct intentionally has no GateRules field: per Drop 3 fix L6 and
// finding 5.B.11, the [gate_rules] TOML table is reserved for forward-compat
// and consumed by Drop 4's dispatcher. The type for that table lands in the
// dispatcher droplet, not here.
//
// Canonical spec: main/PLAN.md § 19.3, ta-docs/cascade-methodology.md §11.
type Template struct {
	// SchemaVersion pins the template to a specific closed-schema revision.
	// The value must match SchemaVersionV1 today; Drop 3.9's loader rejects
	// any other value.
	SchemaVersion string `toml:"schema_version"`

	// Kinds maps each domain.Kind referenced by the template to its rule
	// row. Closed-enum coverage is enforced at validation time (3.10), not
	// at struct-tag level.
	Kinds map[domain.Kind]KindRule `toml:"kinds"`

	// ChildRules is the ordered list of auto-create directives consumed by
	// Template.ChildRulesFor (3.11) when a parent action-item is created.
	ChildRules []ChildRule `toml:"child_rules"`

	// AgentBindings maps each domain.Kind to the agent-spawn parameters the
	// dispatcher uses when the kind transitions to in_progress.
	AgentBindings map[domain.Kind]AgentBinding `toml:"agent_bindings"`

	// GateRulesRaw is the strict-decode escape hatch for the [gate_rules] TOML
	// table reserved per Drop 3 fix L6 (finding 5.B.11). The Go struct for the
	// gate-rule schema lands in Drop 4's dispatcher; until then the loader
	// preserves whatever the document declares as a free-form map so strict
	// decode does not reject the reserved table. The field is excluded from
	// any structural validation in Drop 3 and exists purely for forward-compat.
	GateRulesRaw map[string]any `toml:"gate_rules"`

	// StewardSeeds is the ordered list of long-lived coordination anchor
	// nodes the auto-generator (droplet 3.20) materializes once at project-
	// creation time. Per fix L3 the canonical six are DISCUSSIONS,
	// HYLLA_FINDINGS, LEDGER, WIKI_CHANGELOG, REFINEMENTS, HYLLA_REFINEMENTS;
	// the seed list is open-ended at the schema level so projects with their
	// own template TOML can declare additional anchors.
	//
	// Per droplet 3.14's deferred comment the steward-seed encoding chose
	// option (a): a separate [[steward_seeds]] TOML table with its own
	// loader path. The reasoning kept the closed-enum ChildRule.WhenParentKind
	// invariant intact (every ChildRule still triggers off a real Kind),
	// while making "fires at project creation" semantics explicit on a
	// distinct top-level table rather than overloading ChildRule with a
	// sentinel parent kind or a new at-project-creation bool.
	//
	// Each StewardSeed materializes as a level_1 ActionItem under the
	// project root with Owner = "STEWARD", Persistent = true, DevGated = false,
	// Kind = "discussion" (the closest cascade fit for cross-cutting anchor
	// nodes), and StructuralType = "droplet" (single non-decomposable anchor;
	// per PLAN.md § 19.3 line 1637 these are not cascade work themselves but
	// domain.NewActionItem rejects an empty StructuralType, so the closest
	// approximation is droplet — a terminal node that does not decompose).
	StewardSeeds []StewardSeed `toml:"steward_seeds"`
}

// StewardSeed is one persistent-anchor specification consumed by droplet
// 3.20's auto-generator at project-creation time. Per fix L13 the type is a
// domain primitive — STEWARD is one consumer of Owner = "STEWARD", but seed
// rows could in principle be authored for non-STEWARD anchor patterns by
// future templates.
//
// Both fields are required: an empty Title would yield a duplicate-or-nil
// row under the project root, and an empty Description would leave the
// anchor with no descriptive text for downstream readers. The auto-generator
// enforces non-empty Title; Description is recommended but not strictly
// rejected at the schema layer.
type StewardSeed struct {
	// Title is the literal, ALL-CAPS title applied to the seeded anchor
	// (e.g. "DISCUSSIONS"). Per repo memory project_tillsyn_titles all
	// Tillsyn plan-item titles are FULL UPPERCASE.
	Title string `toml:"title"`

	// Description is the seeded anchor's description prose, written into
	// ActionItem.Description verbatim at materialization time.
	Description string `toml:"description"`
}

// KindRule encodes one closed-enum kind's structural constraints inside a
// Template. Each row pairs a kind with its owner principal, parent/child
// allow-lists, and structural-type axis.
type KindRule struct {
	// Owner is the principal identifier responsible for materializing
	// children of this kind. Per Drop 3 fix L7 the value "STEWARD" marks
	// the kind as STEWARD-owned; the auto-generator (3.20) reads this when
	// deciding which principal queues the create call. Other principal
	// identifiers are accepted verbatim.
	Owner string `toml:"owner"`

	// AllowedParentKinds is the closed-enum list of kinds that may parent
	// this kind. An empty slice means universal-allow, matching the Drop
	// 2.8 semantics where no parent-allow restriction is recorded.
	AllowedParentKinds []domain.Kind `toml:"allowed_parent_kinds"`

	// AllowedChildKinds is the closed-enum list of kinds that may be
	// children of this kind. An empty slice means universal-allow.
	AllowedChildKinds []domain.Kind `toml:"allowed_child_kinds"`

	// StructuralType binds the kind to one of the four cascade structural
	// types declared by domain.StructuralType (drop / segment / confluence
	// / droplet). Per droplet 3.1 this axis is independent of Kind.
	StructuralType domain.StructuralType `toml:"structural_type"`
}

// ChildRule is one auto-create directive evaluated when a parent action-item
// is created. Drop 3.11's Template.ChildRulesFor scans Template.ChildRules
// and returns the entries whose WhenParentKind (and optional
// WhenParentStructuralType) match the parent.
type ChildRule struct {
	// WhenParentKind is the parent action-item kind that triggers this
	// rule. Closed-enum.
	WhenParentKind domain.Kind `toml:"when_parent_kind"`

	// CreateChildKind is the kind of the child auto-created by this rule.
	// Closed-enum.
	CreateChildKind domain.Kind `toml:"create_child_kind"`

	// Title is the literal title applied to the auto-created child.
	Title string `toml:"title"`

	// BlockedByParent, when true, wires the auto-created child with a
	// blocked_by edge to the parent so the child cannot start until the
	// parent reaches its terminal completion state.
	BlockedByParent bool `toml:"blocked_by_parent"`

	// WhenParentStructuralType narrows the rule to parents whose
	// structural_type matches the supplied value. Empty means match any
	// structural type. Per main/PLAN.md line 1635 the rule binds on the
	// structural_type axis as well as the kind axis.
	WhenParentStructuralType domain.StructuralType `toml:"when_parent_structural_type"`
}

// AgentBinding describes the agent spawn parameters the cascade dispatcher
// uses when an action-item of the bound kind transitions to in_progress.
//
// This struct is skeletal in droplet 3.8 per finding 5.B.17 (N4): top-level
// fields are declared with TOML tags but no field-level validation is
// applied. Drop 3.13 fills the validator + a populated round-trip test.
//
// Canonical spec: main/PLAN.md § 19.3 lines 1653-1656.
type AgentBinding struct {
	// AgentName is the canonical agent identifier the dispatcher resolves
	// to a concrete subagent specification (e.g. "go-builder-agent").
	AgentName string `toml:"agent_name"`

	// Model is the LLM model identifier used for the spawn (e.g. "opus",
	// "sonnet", "haiku"). Closed validation deferred to Drop 3.13.
	Model string `toml:"model"`

	// Effort is the model effort tier (e.g. "low", "medium", "high"). The
	// concrete vocabulary is established by the agent layer; Drop 3.13
	// will lock the closed set if one is required.
	Effort string `toml:"effort"`

	// Tools is the allow-list of tool names the spawned agent may call.
	// Validation against the actual MCP/Claude tool catalog is deferred to
	// Drop 4 per finding 5.B.5.
	Tools []string `toml:"tools"`

	// MaxTries caps the number of dispatch attempts before the dispatcher
	// marks the action-item failed.
	MaxTries int `toml:"max_tries"`

	// MaxBudgetUSD caps the per-spawn dollar budget enforced by the
	// dispatcher. A float64 to permit fractional budgets.
	MaxBudgetUSD float64 `toml:"max_budget_usd"`

	// MaxTurns caps the conversation turn count for a single spawn.
	MaxTurns int `toml:"max_turns"`

	// AutoPush, when true, instructs the dispatcher to invoke `git push`
	// after a successful build action-item completes its post-build gates.
	AutoPush bool `toml:"auto_push"`

	// CommitAgent identifies the agent name (typically "commit-agent")
	// used to author commit messages on behalf of build action-items.
	CommitAgent string `toml:"commit_agent"`

	// BlockedRetries caps how many times the dispatcher retries a spawn
	// that returned a "blocked" outcome before escalating.
	BlockedRetries int `toml:"blocked_retries"`

	// BlockedRetryCooldown is the wall-clock delay between blocked-retry
	// attempts. Parsed from TOML duration strings ("30s", "5m", "1h30m") via
	// templates.Duration's TextUnmarshaler. Round-trips back to the canonical
	// time.Duration.String() form.
	BlockedRetryCooldown Duration `toml:"blocked_retry_cooldown"`
}

// Validate reports field-level errors on an AgentBinding. Returns nil if all
// fields are within acceptable bounds.
//
// Validation rules per main/PLAN.md § 19.3 lines 1653-1656:
//   - AgentName: trimmed non-empty.
//   - Model: trimmed non-empty.
//   - MaxTries: > 0 (must allow at least one attempt).
//   - MaxBudgetUSD: >= 0 (zero permitted; means unlimited at dispatcher's choice).
//   - MaxTurns: > 0 (must allow at least one turn).
//   - BlockedRetries: >= 0.
//   - BlockedRetryCooldown: >= 0.
//
// Fields without validation rules (Effort, Tools, AutoPush, CommitAgent) are
// free-form pass-through to the dispatcher; their interpretation is Drop 4's
// concern. Tools content validation against the actual MCP/Claude tool catalog
// is deferred to Drop 4 per finding 5.B.5.
//
// All non-nil returns wrap ErrInvalidAgentBinding so callers can route on the
// sentinel via errors.Is.
func (b AgentBinding) Validate() error {
	if strings.TrimSpace(b.AgentName) == "" {
		return fmt.Errorf("%w: agent_name must be non-empty", ErrInvalidAgentBinding)
	}
	if strings.TrimSpace(b.Model) == "" {
		return fmt.Errorf("%w: model must be non-empty", ErrInvalidAgentBinding)
	}
	if b.MaxTries <= 0 {
		return fmt.Errorf("%w: max_tries must be > 0 (got %d)", ErrInvalidAgentBinding, b.MaxTries)
	}
	if b.MaxTurns <= 0 {
		return fmt.Errorf("%w: max_turns must be > 0 (got %d)", ErrInvalidAgentBinding, b.MaxTurns)
	}
	if b.MaxBudgetUSD < 0 {
		return fmt.Errorf("%w: max_budget_usd must be >= 0 (got %v)", ErrInvalidAgentBinding, b.MaxBudgetUSD)
	}
	if b.BlockedRetries < 0 {
		return fmt.Errorf("%w: blocked_retries must be >= 0 (got %d)", ErrInvalidAgentBinding, b.BlockedRetries)
	}
	if time.Duration(b.BlockedRetryCooldown) < 0 {
		return fmt.Errorf("%w: blocked_retry_cooldown must be >= 0 (got %s)", ErrInvalidAgentBinding, time.Duration(b.BlockedRetryCooldown))
	}
	return nil
}
