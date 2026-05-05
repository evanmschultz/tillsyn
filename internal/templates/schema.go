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

// GateKind names one closed-enum gate identifier consumed by the cascade
// dispatcher's gate runner (Drop 4b Wave A 4b.2). The closed set is
// deliberately small: each constant binds to a deterministic, no-LLM gate
// implementation in internal/app/dispatcher (mage_ci → 4b.3 gate_mage_ci.go,
// mage_test_pkg → 4b.4 gate_mage_test_pkg.go, hylla_reingest → 4b.7).
//
// Drop 4c expands the enum with "commit" and "push" — two additional closed
// values that bind to the commit-message-agent + git-push gates respectively.
// Until Drop 4c lands, IsValidGateKind rejects those literals so a template
// authored against a future cascade vocabulary fails at load time rather than
// silently no-op'ing at run time. The closed-enum + load-time validation
// pattern mirrors domain.Kind / domain.StructuralType per Drop 4b REVISION_BRIEF
// locked decisions L1 (closed-enum gates) and L6 (default ships only mage_ci).
type GateKind string

// Closed-enum GateKind constants. Adding a new value requires both a constant
// here AND an entry in validGateKinds below — IsValidGateKind reads the
// validGateKinds slice rather than a switch so the membership set stays in
// one place.
const (
	// GateKindMageCI runs `mage ci` in the project's primary worktree. The
	// canonical post-build verification gate; ships in the default template
	// per Drop 4b L6.
	GateKindMageCI GateKind = "mage_ci"

	// GateKindMageTestPkg runs `mage testPkg <pkg>` for each package in the
	// triggering action item's domain.ActionItem.Packages slice. Used as a
	// scoped pre-flight when the full mage_ci gate is too coarse.
	GateKindMageTestPkg GateKind = "mage_test_pkg"

	// GateKindHyllaReingest invokes the Hylla MCP `hylla_ingest` tool against
	// the project's GitHub remote. Runs at drop-end only per Drop 4b Wave C
	// 4b.7 wiring; never per-build.
	GateKindHyllaReingest GateKind = "hylla_reingest"
)

// validGateKinds stores every member of the closed GateKind enum. Drop 4c
// extends this slice with "commit" and "push"; until then the closed set is
// exactly the three constants above.
var validGateKinds = []GateKind{
	GateKindMageCI,
	GateKindMageTestPkg,
	GateKindHyllaReingest,
}

// IsValidGateKind reports whether g is a member of the closed GateKind enum.
// The check is exact-match against validGateKinds — no whitespace trimming or
// case folding. Template authors are responsible for canonical spelling.
//
// Unlike domain.IsValidKind which normalizes via strings.TrimSpace +
// strings.ToLower (case-insensitive, whitespace-tolerant), IsValidGateKind
// does exact-match. Gate kinds are stricter than action-item kinds because
// templates author them explicitly and silent fold-matching would mask typos
// (e.g. " Mage_CI " quietly resolving to mage_ci) at load time, well before
// the dispatcher runs.
func IsValidGateKind(g GateKind) bool {
	for _, candidate := range validGateKinds {
		if candidate == g {
			return true
		}
	}
	return false
}

// Template is the closed-schema root for a cascade-template definition. It
// pairs a schema_version pin with the closed kind, child-rule, and
// agent-binding tables.
//
// Per Drop 4b Wave A 4b.1 the Gates field encodes the per-kind gate sequence
// the dispatcher's gate runner executes when an action item of that kind
// transitions to its provisional terminal state. Gates is distinct from the
// reserved-but-untyped GateRulesRaw map below: the two TOML keys (`gates` vs
// `gate_rules`) decode independently. Wave A consumes Gates; GateRulesRaw
// remains the forward-compat seam for richer gate config (timeouts, retry
// policy) authored under a different TOML key.
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

	// Gates is the per-kind gate sequence consumed by the dispatcher's gate
	// runner (Drop 4b Wave A 4b.2). Each map entry pairs a parent action-item
	// kind with the ordered list of GateKind values the runner executes
	// post-build, halting on the first failure. Absence of an entry for a
	// given kind means "no gates" (gate runner returns Success: true
	// immediately) — NOT "all gates" (resolves Drop 4b WAVE_A_PLAN.md 4b.1
	// acceptance bullet).
	//
	// Both axes are validated at load time: validateMapKeys asserts every
	// map key is a member of the closed domain.Kind enum, and
	// validateGateKinds asserts every value-slice element is a member of
	// the closed GateKind enum. Slice order is preserved by go-toml/v2's
	// array decoder so [gates.build] = ["mage_ci", "mage_test_pkg"] runs
	// mage_ci first, then mage_test_pkg.
	Gates map[domain.Kind][]GateKind `toml:"gates"`

	// GateRulesRaw is the strict-decode escape hatch for the [gate_rules] TOML
	// table reserved per Drop 3 fix L6 (finding 5.B.11). The Go struct for the
	// gate-rule schema lands in Drop 4's dispatcher; until then the loader
	// preserves whatever the document declares as a free-form map so strict
	// decode does not reject the reserved table. The field is excluded from
	// any structural validation in Drop 3 and exists purely for forward-compat.
	//
	// Distinct from the Gates field above — the two consume different TOML
	// keys (`gates` vs `gate_rules`) and decode independently. Drop 4b's
	// `gates` table lands the closed-enum gate sequence; a future drop may
	// type GateRulesRaw into a richer per-gate config struct (timeouts,
	// retry policy, env-var pinning) without colliding with `gates`.
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

	// Env is the closed allow-list of environment-variable NAMES the
	// dispatcher's CLI adapter forwards from the orchestrator's process to
	// the spawned agent process. Per Drop 4c F.7.17 locked decision L4 the
	// adapter resolves each name via os.Getenv at spawn time; missing values
	// fail loud per-action-item (the action item moves to failed with a
	// metadata.failure_reason naming the offending var). Per L8 the adapter
	// constructs cmd.Env explicitly — os.Environ() is NOT inherited — so
	// only the closed POSIX baseline (L6) plus the names listed here ever
	// reach the spawn.
	//
	// Each entry must match `^[A-Za-z][A-Za-z0-9_]*$` (uppercase OR
	// lowercase leading letter; trailing alphanumerics + underscore). The
	// regex permits both `HTTP_PROXY` and the conventional cURL `http_proxy`
	// spelling per L5 + Drop 4c F.7.17 falsification round 2 A2.d. Empty
	// strings, entries containing `=` (KEY=value form), whitespace,
	// hyphens, dots, leading digits, and within-binding duplicates are
	// rejected at template Load time by validateAgentBindingEnvNames.
	//
	// REV-1 (Drop 4c F.7.17.1 REVISIONS POST-AUTHORING): the originally
	// scoped `Command []string` and `ArgsPrefix []string` fields were
	// dropped from the design — adapters invoke their CLI binary directly
	// (`claude` / `codex`) and process-isolation is an OS-level concern.
	// `Env` and `CLIKind` are the only F.7.17.1 additions.
	Env []string `toml:"env"`

	// CLIKind selects which CLI adapter the dispatcher routes the spawn to.
	// Closed enum: "claude" (Drop 4c) and "codex" (Drop 4d). The empty
	// string is permitted at the schema level and resolves to "claude" at
	// adapter-lookup time per Drop 4c F.7.17 locked decision L15
	// (back-compat default). Validation against the closed set is
	// performed at adapter-lookup time, NOT at template Load time, so a
	// template authored against a future Tillsyn release that adds new
	// CLIKind values still loads cleanly under an older binary — the spawn
	// fails at dispatch time with a precise "no adapter for cli_kind X"
	// error instead.
	//
	// REV-1: companion to Env. The wrapper-interop knob (`Command` /
	// `ArgsPrefix`) is GONE; adapters hardcode their CLI binary internally.
	CLIKind string `toml:"cli_kind"`
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
