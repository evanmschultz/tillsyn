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
// Drop 4c expands the enum with "commit" (F.7.13, landed) and "push" (a
// follow-up droplet) — two additional closed values that bind to the
// commit-message-agent + git-push gates respectively. The closed-enum +
// load-time validation pattern mirrors domain.Kind / domain.StructuralType
// per Drop 4b REVISION_BRIEF locked decisions L1 (closed-enum gates) and L6
// (default ships only mage_ci); IsValidGateKind rejects "push" until that
// droplet lands so templates authored against a future cascade vocabulary
// fail at load time rather than silently no-op'ing at run time.
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

	// GateKindCommit invokes the F.7.13 commit gate: shells the F.7.12
	// CommitAgent for a single-line conventional-commit message, then runs
	// path-scoped `git add` + `git commit` against the project worktree and
	// records the resulting HEAD hash on the action item's EndCommit field.
	// Drop 4c F.7.13 ships the gate implementation; Drop 4c F.7.16 expands
	// the default template's [gates.build] sequence to include this kind.
	// Off-by-default per project metadata DispatcherCommitEnabled toggle
	// (F.7.15) — the gate is a no-op when the toggle is unset / false.
	GateKindCommit GateKind = "commit"
)

// validGateKinds stores every member of the closed GateKind enum. Drop 4c
// F.7.13 added "commit"; "push" lands in a follow-up droplet alongside its
// gate implementation.
var validGateKinds = []GateKind{
	GateKindMageCI,
	GateKindMageTestPkg,
	GateKindHyllaReingest,
	GateKindCommit,
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

	// Tillsyn carries top-level dispatcher / aggregator globals declared
	// under the [tillsyn] table. F.7.18.2 ships the initial declaration with
	// two fields (MaxContextBundleChars, MaxAggregatorDuration); subsequent
	// F.7-CORE droplets extend it with SpawnTempRoot (F.7.1) and
	// RequiresPlugins (F.7.6) per master PLAN.md §5 "Tillsyn struct extension
	// policy". Without this field, strict-decode would reject any [tillsyn]
	// table at load time per load.go step 3.
	Tillsyn Tillsyn `toml:"tillsyn"`

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

// Tillsyn carries top-level dispatcher / aggregator globals declared under
// the [tillsyn] table in template TOML. F.7.18.2 shipped the initial
// declaration with two fields; subsequent F.7-CORE droplets extend it with
// SpawnTempRoot (F.7.1) and RequiresPlugins (F.7.6) per master PLAN.md §5
// "Tillsyn struct extension policy".
//
// Closed-struct unknown-key rejection: every field carries an explicit TOML
// tag so templates.Load's strict-decode chain (load.go step 3) rejects
// unknown keys nested under [tillsyn] as ErrUnknownTemplateKey at load time.
// The strict-decode contract is exercised by
// TestLoadTillsynStrictDecodeUnknownFieldRejected.
//
// Default-substitution semantics: a zero-valued field is LEGAL at the schema
// layer; the aggregator engine substitutes the bundle-global default at
// runtime (F.7.18.4 territory). Negative values are rejected at load time
// by validateTillsyn.
type Tillsyn struct {
	// MaxContextBundleChars caps the total bytes the F.7.18 aggregator
	// emits per spawn bundle. Engine-time default substituted when zero
	// per master PLAN L14 (greedy-fit algorithm). Negative values are
	// rejected at load time by validateTillsyn.
	MaxContextBundleChars int `toml:"max_context_bundle_chars"`

	// MaxAggregatorDuration caps total wall-clock time the F.7.18
	// aggregator spends building one spawn bundle. Engine-time default
	// substituted when zero per master PLAN L15 (two-axis cap; per-rule
	// cap lives on AgentBinding.Context.MaxRuleDuration). Negative values
	// are rejected at load time by validateTillsyn.
	MaxAggregatorDuration Duration `toml:"max_aggregator_duration"`

	// SpawnTempRoot selects where the dispatcher's per-spawn ephemeral
	// bundle directory is materialized. Closed-enum string values:
	//
	//   - "" (omitted): consumer-time default = "os_tmp" (the dispatcher's
	//     bundle materializer resolves the empty string to OS temp dir at
	//     spawn time per Drop 4c F.7.1 NewBundle).
	//   - "os_tmp": bundle root = os.MkdirTemp under os.TempDir() with the
	//     `tillsyn-spawn-` prefix. Bundles are reaped on terminal-state by
	//     the dispatcher's cleanup hook (F.7.8 owns orphan-scan).
	//   - "project": bundle root = <projectRoot>/.tillsyn/spawns/<spawn-id>/.
	//     Used when the dev wants forensics under the worktree (e.g. for
	//     post-mortem inspection); F.7.7 lands the gitignore auto-add when
	//     this mode is selected.
	//
	// validateTillsyn rejects any other value at load time. Per Drop 4c
	// F.7-CORE F.7.1 REV-7 the field belongs on this struct (declared in
	// F.7.18.2) rather than a separate top-level table — the [tillsyn]
	// table is the single carrier for dispatcher-global knobs.
	SpawnTempRoot string `toml:"spawn_temp_root"`

	// RequiresPlugins is the closed list of Claude Code plugin identifiers
	// the project's spawn pipeline requires the dev's local Claude install
	// to carry. The dispatcher's pre-flight check (Drop 4c F.7.6
	// CheckRequiredPlugins) shells out to `claude plugin list --json`,
	// parses the installed-plugin set, and fails hard with a
	// `claude plugin install <name>` instruction whenever any entry in this
	// slice is missing.
	//
	// Format: each entry MUST be one of two shapes —
	//
	//   - `<name>` — bare plugin identifier (marketplace-implicit). The
	//     pre-flight matcher accepts any installed entry whose `id` matches
	//     the supplied name, regardless of marketplace.
	//   - `<name>@<marketplace>` — plugin identifier scoped to a specific
	//     marketplace source. The pre-flight matcher accepts only installed
	//     entries whose `id` AND `marketplace` both match.
	//
	// validateTillsyn rejects empty entries, entries containing whitespace,
	// entries containing more than one `@`, and within-list duplicates at
	// load time. NO closed-enum vocabulary check on plugin names — adopters
	// supply real Claude plugin identifiers and the dispatcher's runtime
	// shell-out is the authoritative gate.
	//
	// Empty slice (or omitted field) means "no required plugins" — the
	// pre-flight check returns nil immediately without invoking
	// `claude plugin list --json` so adopters who do not depend on any
	// plugin pay no exec cost per spawn.
	//
	// Per Drop 4c REV-7 the field belongs on this struct (declared in
	// F.7.18.2; SpawnTempRoot added by F.7.1) rather than a separate
	// top-level table — the [tillsyn] table is the single carrier for
	// dispatcher-global knobs.
	RequiresPlugins []string `toml:"requires_plugins,omitempty"`
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

	// Context declares optional pre-staged context the dispatcher's aggregator
	// renders into the spawn bundle before the CLI fires. Adopters may omit
	// the [context] table entirely; the spawn then runs in fully agentic mode
	// (the agent uses MCP for whatever context it needs). Both modes are
	// equally first-class — neither is the recommended default.
	//
	// Per Drop 4c F.7.18 (master PLAN.md L13): the schema is FLEXIBLE not
	// REQUIRED. Validation in templates.Load enforces field-shape +
	// closed-enum-membership only; default-substitution for zero-valued
	// caps + timeouts happens at engine-time in F.7.18.4 (the aggregator's
	// greedy-fit + two-axis wall-clock wrapper).
	//
	// REV-3 (Drop 4c F.7.18 REVISIONS POST-AUTHORING): F.7.18.1 lands ONLY
	// the Context sub-struct on AgentBinding. The companion top-level
	// `Tillsyn` struct that supplies bundle-global caps + the aggregator
	// engine itself land in F.7.18.2 + F.7.18.3 respectively.
	Context ContextRules `toml:"context"`

	// ToolsAllowed names the tools an agent's settings.json `permissions.allow`
	// list will include at spawn-render time (F.7.3b). Empty means no explicit
	// allow rules — the agent inherits whatever defaults the rendered
	// settings.json carries. Per memory §5 / SKETCH §F.7.2 the two-layer
	// tool-gating strategy uses this field as Layer A: the agent's frontmatter
	// `allowedTools` mirrors the entries here for human readability while the
	// settings.json `permissions.allow` rendered from the same slice is the
	// authoritative gate at runtime.
	//
	// Validated at template Load time by validateAgentBindingToolGating: each
	// entry must be a non-empty string, and within-binding duplicates are
	// rejected. NO closed-enum membership check — tool names are open-ended
	// (Read / Edit / Bash(mage *) / WebFetch / etc.) and template authors are
	// trusted to supply real Claude / MCP tool identifiers.
	ToolsAllowed []string `toml:"tools_allowed"`

	// ToolsDisallowed names the tools the settings.json `permissions.deny`
	// list will include at spawn-render time. Per memory §5 / SKETCH §F.7.2
	// this is the AUTHORITATIVE tool-gating layer — the probe-grounded finding
	// is that agents route around `--allowed-tools` / `--disallowed-tools` CLI
	// flag removal via Bash, so only deny patterns inside settings.json catch
	// the workaround. F.7.3b will additionally auto-mirror these entries into
	// `--allowed-tools` / `--disallowed-tools` flag emission AND auto-add the
	// closed set of Bash-workaround patterns when `WebFetch` is denied.
	//
	// Validated by validateAgentBindingToolGating with the same rules as
	// ToolsAllowed (non-empty entries, within-binding-unique).
	ToolsDisallowed []string `toml:"tools_disallowed"`

	// SystemPromptTemplatePath optionally overrides the per-kind built-in
	// system-prompt template the dispatcher's render layer (F.7.3b) uses when
	// assembling `<bundle>/system-prompt.md`. When empty the render layer
	// falls back to the canonical built-in template for the binding's kind.
	//
	// Format contract: a project-relative path under `.tillsyn/`. Validation
	// at template Load time (validateAgentBindingToolGating) rejects:
	//
	//   - Absolute paths (path begins with `/`).
	//   - Paths containing `..` traversal segments.
	//   - Paths containing shell metacharacters `;`, `|`, `&`, backtick, or `$`
	//     (defense-in-depth — the dispatcher's render layer never invokes a
	//     shell against this path, but rejecting at the schema layer keeps the
	//     resolved-path safe in the face of future refactors).
	//
	// The actual file is NOT opened or stat'd at template Load time — the
	// path may legitimately reference a resource that doesn't exist until the
	// template is consumed. Resolution + read errors surface at spawn-render
	// time inside F.7.3b.
	SystemPromptTemplatePath string `toml:"system_prompt_template_path"`

	// Sandbox declares per-spawn sandbox configuration the dispatcher's render
	// layer (F.7.3b) renders into the rendered settings.json. Per memory §4
	// the sandbox semantics rely on Claude Code's settings.json
	// `permissions.{allow,deny}` for filesystem AND on out-of-process network
	// gating; this field is the schema seam, NOT the enforcement layer.
	//
	// Closed sub-struct: every field carries an explicit TOML tag so
	// templates.Load's strict-decode chain (load.go step 3) rejects unknown
	// keys nested under `[agent_bindings.<kind>.sandbox]` as
	// ErrUnknownTemplateKey at load time.
	Sandbox SandboxRules `toml:"sandbox"`
}

// SandboxRules is the closed sub-struct on AgentBinding declaring per-spawn
// sandbox directives consumed by the F.7.3b render layer when rendering
// settings.json. Both nested fields are optional; the zero-value struct
// (no `[sandbox]` table at all) means "no sandbox rules" — the spawn inherits
// whatever permissions the rendered settings.json grants by default.
//
// Closed-struct unknown-key rejection: every field carries an explicit TOML
// tag so templates.Load's strict-decode chain (load.go step 3) rejects
// unknown keys nested under `[agent_bindings.<kind>.sandbox]` as
// ErrUnknownTemplateKey at load time.
//
// Per Drop 4c F.7.2 plan acceptance criteria.
type SandboxRules struct {
	// Filesystem declares filesystem permissions for the spawn's sandbox.
	Filesystem SandboxFilesystem `toml:"filesystem"`

	// Network declares network permissions for the spawn's sandbox.
	Network SandboxNetwork `toml:"network"`
}

// SandboxFilesystem encodes filesystem permissions for the spawn's sandbox.
// Both slices are optional. Each entry must be a clean absolute path (begins
// with `/`, no `..` segments, no double-slashes); validation at template Load
// time is performed by validateAgentBindingToolGating.
type SandboxFilesystem struct {
	// AllowWrite is the set of absolute paths the spawn may write to. Each
	// entry must be a clean absolute path.
	AllowWrite []string `toml:"allow_write"`

	// DenyRead is the set of absolute paths the spawn must NOT read. Each
	// entry must be a clean absolute path.
	DenyRead []string `toml:"deny_read"`
}

// SandboxNetwork encodes network permissions for the spawn's sandbox. Both
// slices are optional. Each entry must be a non-empty string with no URL
// scheme prefix (no `http://`, `https://`); a leading `*` glob is permitted
// (e.g. `*.npmjs.org`).
type SandboxNetwork struct {
	// AllowedDomains is the set of network destinations (e.g. "github.com",
	// "*.npmjs.org") the spawn may reach.
	AllowedDomains []string `toml:"allowed_domains"`

	// DeniedDomains is the set of network destinations the spawn must NOT
	// reach.
	DeniedDomains []string `toml:"denied_domains"`
}

// ContextRules is the closed sub-struct on AgentBinding declaring the
// dispatcher aggregator's per-rule pre-staging directives. Every field is
// optional; the zero-value struct (no `[context]` table at all) selects
// fully-agentic mode where the spawned agent receives no pre-staged context
// and uses MCP for whatever it needs (per master PLAN.md L13 — both modes
// first-class).
//
// Closed-struct unknown-key rejection: every field carries an explicit TOML
// tag so templates.Load's strict-decode chain (load.go step 3) rejects
// unknown keys nested under `[agent_bindings.<kind>.context]` as
// ErrUnknownTemplateKey at load time.
//
// Default-substitution semantics: a zero-valued MaxChars or MaxRuleDuration
// is LEGAL at the schema layer; the aggregator engine substitutes the
// bundle-global default at runtime (F.7.18.4 territory). Negative values are
// rejected at load time by validateAgentBindingContext.
//
// Per Drop 4c F.7.18 plan acceptance criteria + REV-3.
type ContextRules struct {
	// Parent, when true, instructs the aggregator to render the parent
	// action-item's identity + description into the pre-staged context.
	// Plan-walks (`AncestorsByKind`/`DescendantsByKind`) start from the
	// parent the spawned action-item is nested under.
	Parent bool `toml:"parent"`

	// ParentGitDiff, when true, instructs the aggregator to capture the
	// `git diff <parent.start_commit>..<parent.end_commit>` payload when the
	// parent action-item carries non-empty start_commit + end_commit fields
	// (Drop 4a Wave 1 first-class fields per PLAN.md). When the parent has
	// no commit anchors the rule renders a marker instead of failing the
	// spawn — the field's purpose is "if this is observable, give it to me."
	ParentGitDiff bool `toml:"parent_git_diff"`

	// SiblingsByKind selects sibling action-items (same parent) by kind for
	// pre-staging. The aggregator emits the LATEST round only — superseded
	// predecessors are skipped — so the spawn sees the most recent sibling
	// artifact for each entry in the closed-12-kind enum.
	SiblingsByKind []domain.Kind `toml:"siblings_by_kind"`

	// AncestorsByKind walks UP the parent chain and captures the FIRST
	// ancestor whose Kind matches an entry in this slice. The walk respects
	// declaration order: `["plan", "build"]` returns the nearest plan ancestor
	// when one exists, falling back to the nearest build ancestor otherwise.
	AncestorsByKind []domain.Kind `toml:"ancestors_by_kind"`

	// DescendantsByKind walks DOWN the cascade subtree and captures every
	// direct + transitive descendant whose Kind matches an entry in this
	// slice. Usually empty in default-template seeds — adopters writing
	// fix-planners or tree-pruners explicitly opt in.
	//
	// Per master PLAN.md F.7.18 + plan acceptance: NO schema rule against
	// `descendants_by_kind` on `kind=plan`. Template authors trusted; if
	// the use case is illegitimate, the planner simply does the right thing.
	DescendantsByKind []domain.Kind `toml:"descendants_by_kind"`

	// Delivery selects how the aggregator surfaces rendered context to the
	// spawned agent. Closed-enum string values:
	//   - "" (omitted): consumer-time default = "file" (F.7.18.3 engine
	//     resolves the empty string to file-mode at Resolve-time).
	//   - "inline": rendered context is appended to the spawn's
	//     system-append.md.
	//   - "file": rendered context is written into <bundle>/context/<rule>.md
	//     and the agent uses Read to load on demand.
	// validateAgentBindingContext rejects any other value at load time.
	Delivery string `toml:"delivery"`

	// MaxChars caps the rendered byte count for THIS binding's per-rule
	// renderers. Engine-time default = 50000 when zero (F.7.18.3 territory).
	// Negative values are rejected at load time. The bundle-global cap
	// (max_context_bundle_chars under [tillsyn]) is layered on top in
	// F.7.18.4's greedy-fit cap algorithm; this per-rule cap localizes
	// truncation to a single rule before the bundle-cap considers skipping.
	MaxChars int `toml:"max_chars"`

	// MaxRuleDuration caps the per-rule wall-clock budget. Engine-time
	// default = 500ms when zero (F.7.18.4 wires context.WithTimeout per
	// rule). Negative values are rejected at load time. The per-bundle cap
	// (max_aggregator_duration under [tillsyn]) wraps the entire rule
	// iteration; this per-rule cap ensures one slow rule cannot starve the
	// remaining rules.
	MaxRuleDuration Duration `toml:"max_rule_duration"`
}

// Closed-enum delivery vocabulary for ContextRules.Delivery. The empty string
// is permitted at the schema layer and resolves to ContextDeliveryFile at
// engine-time (F.7.18.3) per master PLAN.md L13's "consumer-time default"
// framing.
const (
	// ContextDeliveryInline marks rendered context for inline append into
	// the spawn's system-append.md.
	ContextDeliveryInline = "inline"

	// ContextDeliveryFile marks rendered context for file-mode write into
	// <bundle>/context/<rule>.md. The aggregator's resolver substitutes this
	// for an omitted Delivery value at Resolve-time.
	ContextDeliveryFile = "file"
)

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
