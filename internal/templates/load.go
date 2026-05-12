package templates

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	toml "github.com/pelletier/go-toml/v2"

	"github.com/evanmschultz/tillsyn/internal/domain"
)

// LoadOptions carries optional parameters to LoadWithOptions. The zero value
// is legal and matches Load's behavior: no warning logger (warnings are
// silently dropped) and the production filesystem stat function (os.Stat) for
// agent-binding-file existence checks.
//
// Drop 4c.5 F.5.1 introduced this struct so adopters can plumb a warn-logger
// through Load without breaking the existing Load(io.Reader) signature.
// Adopters who want strict-fail behavior on missing agent-binding files
// (rather than the default warn-only) wrap WarnLogger to escalate at their
// call site.
type LoadOptions struct {
	// WarnLogger receives one line per non-fatal validation finding. Today
	// only validateAgentBindingFiles (F.5.1) emits warnings. Nil is legal
	// and means "drop warnings on the floor" — preserves Load's pre-F.5.1
	// silence-by-default contract.
	WarnLogger func(string)

	// StatFn is the existence-check function used by validateAgentBindingFiles
	// for `~/.claude/agents/<name>.md`. Nil resolves to a default that calls
	// os.Stat and returns true on success. Tests inject in-memory stubs so
	// the validator's warning behavior is deterministic regardless of the
	// dev's machine state.
	StatFn func(path string) bool

	// AgentLookupFn is the existence-check used by
	// validateAgentBindingNames (Drop 4c.6 W0.5.D2) for the EMBEDDED tier
	// of the 3-tier agent resolver (per SKETCH.md §3.4). Returns true when
	// `<name>` resolves to a real `*.md` file in any of the embedded
	// `builtin/agents/{till-gen,till-go,till-gdd}/<name>.md` paths.
	//
	// Nil resolves to a default that walks DefaultAgentLibraryFS
	// unconditionally — no project-tier or user-tier lookup is performed
	// at template load time (those are spawn-time concerns handled by the
	// existing warn-only validateAgentBindingFiles). Tests inject
	// synthetic lookup tables so the validator's behaviour is
	// deterministic regardless of which embedded files have shipped.
	//
	// Pre-W1.D1 (embedded agent .md files not yet shipped) the default
	// walker returns false for every name — exercising the default in a
	// unit test without an explicit injection deliberately fails-loud per
	// W0.5 round-2 FF2 disclosure: tests inject; production callers
	// (LoadDefaultTemplate*) inherit the post-W1.D1 reality where the
	// real placeholder files satisfy the floor.
	AgentLookupFn func(name string) bool

	// BlockedByGraphFn is the test-only injection point used by
	// validateBlockedByAcyclicity (Drop 4c.6 W0.5.D5) to substitute a
	// synthetic kind-level blocked_by graph for the production walker's
	// output. Nil resolves to the production walker that today builds the
	// graph from every ChildRule whose BlockedByParent is true (one edge
	// per such rule, from CreateChildKind to WhenParentKind).
	//
	// Today's production graph is degenerate (forest of child→parent
	// edges, trivially acyclic). The injection point exists so the
	// validator can be exercised by a real RED→GREEN test against a
	// hypothetical kind-level cycle that today's `BlockedByParent bool`
	// schema cannot otherwise construct. Per W0.5 plan FF1 disclosure:
	// the validator's value is forward-looking — when a future schema
	// addition gives ChildRule a richer blocked_by axis (e.g. a
	// `BlockedByKinds []domain.Kind` field), the DFS already covers
	// cycles in that graph and the injection point becomes vestigial.
	//
	// Production callers (LoadDefaultTemplate*) leave this nil and
	// inherit the production walker. Test callers MAY inject a synthetic
	// graph fn that returns whatever shape exercises the validator.
	BlockedByGraphFn func(rules []ChildRule) map[domain.Kind][]domain.Kind

	// ClaimedConsumersFn is the test-only injection point used by
	// validateClaimVsImplCoherence (Drop 4c.6 W0.5.D6) to substitute a
	// synthetic claimed-consumer list for the production walker's output.
	// Nil resolves to the production walker that returns an empty slice
	// for every template — today's schema has no
	// `[[child_rules]] consumer = "..."` field, so no template can author
	// a real claim against the known-wired-consumer set without test-only
	// injection.
	//
	// For Drop 4c.6 the validator's `knownWiredConsumers` map is empty
	// (Drop 4c.7 W7 + W8 add `child_rules_for` and `context_resolve` when
	// those waves wire the first real consumers). Pre-4c.7 the validator's
	// only meaningful exercised path is the test-seam injection — tests
	// supply a synthetic claim list and assert either rejection (when the
	// claim is unknown) or acceptance (when the claim is in a temporarily-
	// registered known-wired entry).
	//
	// Forward-looking: when Drop 4c.7 W7 + W8 wire the first real
	// consumers AND a future schema addition gives ChildRule a
	// `consumer = "..."` field, the production walker extracts the
	// claimed consumers from the parsed Template and the injection point
	// becomes vestigial.
	//
	// Production callers (LoadDefaultTemplate*) leave this nil and
	// inherit the production walker. Test callers MAY inject a synthetic
	// claim list that returns whatever shape exercises the validator.
	ClaimedConsumersFn func(tpl Template) []string
}

// Load parses a Tillsyn template TOML stream and validates it.
//
// Load is preserved as a thin wrapper around LoadWithOptions(r, LoadOptions{})
// so the pre-Drop-4c.5 single-arg shape continues to compile for every
// existing caller. Adopters who want to plumb a warn-logger or inject a stub
// stat function call LoadWithOptions directly.
//
// Decoding order is fixed by Drop 3 finding 5.B.10 (CE5 schema-version
// pre-pass mitigation):
//
//  1. Tolerant pre-pass — read the entire stream, then decode ONLY the
//     schema_version key with a separate Decoder that does NOT reject
//     unknown fields. This isolates the version check from any other
//     vocabulary churn.
//  2. Reject if the declared schema_version is not SchemaVersionV1.
//  3. Strict decode — re-decode the buffered bytes into a Template using a
//     Decoder configured with DisallowUnknownFields so any unknown FIELD
//     inside a known table (e.g. a misspelled key inside an existing
//     [kinds.build] row) becomes ErrUnknownTemplateKey. The forward-compat
//     [gate_rules] table reserved on Template.GateRulesRaw is exempt.
//     Strict decode does NOT validate map KEYS themselves: TOML treats
//     [kinds.bulid] and [agent_bindings.totally-bogus] as legitimate map
//     entries with arbitrary keys, so a transposed-letter or otherwise
//     unknown kind survives this pass and is caught by validateMapKeys
//     below.
//  4. Load-time validators in this order:
//     a. validateMapKeys — assert every map key in Template.Kinds,
//     Template.AgentBindings, and Template.Gates is a member of the
//     closed 12-value Kind enum. Catches typos like [kinds.bulid] that
//     strict decode cannot.
//     a'. validateAgentMapKeys — same closed-enum check applied to
//     Template.Agents (the runtime-config map W0 will wire). Drop 4c.6
//     W0.5.D1 hook; mirrors validateMapKeys' canonicalization shape.
//     b. validateChildRuleKinds — assert every Kind referenced in
//     [child_rules] is a member of the closed enum.
//     c. validateChildRuleCycles — DFS the unified [child_rules] kind
//     graph for directed cycles. Walks BOTH the parent → child auto-create
//     graph AND the blocked_by-induced graph (every BlockedByParent=true
//     rule contributes a child → parent edge). The wrapped cycle-path
//     message names the offending edge type ("[parent->child]" or
//     "[blocked_by]"). Drop 4c.6 W0.5.D3 extended the pre-existing
//     parent→child detector to cover the unified graph; root iteration is
//     sorted-key for reproducible cycle-path renderings.
//     c'. validateChildRuleRecursionDepth — DAG longest-path DFS over the
//     parent → child graph; reject when any reachable depth exceeds
//     childRuleRecursionDepthMax (5 by default per SKETCH.md § 26.W0.5).
//     Runs immediately after the cycle detector so cyclic graphs are
//     rejected with the better diagnostic (cycle path) before the depth
//     DFS could be invoked. Drop 4c.6 W0.5.D4 hook.
//     c”. validateBlockedByAcyclicity — colored-DFS over the kind-level
//     blocked_by graph (the [blocked_by] edge subgraph independent of
//     the parent→child auto-create graph); reject directed cycles with
//     ErrTemplateBlockedByCycle. Today's production graph is degenerate
//     (child→parent forest, trivially acyclic) — the validator's value
//     is forward-looking against future schema additions like a
//     `BlockedByKinds []domain.Kind` field. Coupled cycles (today's
//     BlockedByParent=true rules contributing one edge to BOTH the
//     parent→child and the blocked_by graphs) are caught by D3 with
//     ErrTemplateCycle FIRST so the diagnostic stays consistent with
//     pre-D5 behaviour. Drop 4c.6 W0.5.D5 hook.
//     c”'. validateClaimVsImplCoherence — assert every claimed
//     `[[child_rules]]` output kind / template feature is a member of
//     the closed Go-internal `knownWiredConsumers` map. The map is
//     empty for Drop 4c.6 (Drop 4c.7 W7 adds `child_rules_for`; Drop
//     4c.7 W8 adds `context_resolve`); production walker returns an
//     empty claim list for every template today, so the validator
//     vacuously passes on every embedded default template. The
//     scaffolding + sentinel + tests ship now so the
//     shipped-but-not-wired anti-pattern (Drop 3 droplet 3.20) cannot
//     recur: every future schema feature claiming a runtime consumer
//     fails Load until the consumer's identifier is added to
//     `knownWiredConsumers`. Drop 4c.6 W0.5.D6 hook.
//     d. validateRequiredChildRules — assert that every present
//     `kind=plan` row has both QA twin child_rules
//     (`plan-qa-proof` + `plan-qa-falsification`) and every present
//     `kind=build` row has both QA twin child_rules
//     (`build-qa-proof` + `build-qa-falsification`). Conditional on
//     the parent kind being declared in [kinds]; absent kinds skip
//     the check. Drop 4c.5 F.5.1 hook.
//     e. validateChildRuleReachability — assert every member of the
//     closed 12-value domain.Kind enum (except the 6 standalone kinds
//     `closeout`/`commit`/`refinement`/`discussion`/`human-verify`/
//     `research`) is referenced by at least one [[child_rules]] entry,
//     either as WhenParentKind or as CreateChildKind. Vacuously true on
//     the embedded default templates; catches typo-stripped adopter
//     templates. Drop 4c.5 F.5.2 hook (replaced the prior no-op stub).
//     f. validateKindStructuralCoherence — assert every [kinds.X] row
//     whose structural_type == "drop" has at least one [[child_rules]]
//     entry with when_parent_kind == X. The thin cross-axis wedge
//     between structural_type and child_rules; full coherence is
//     post-MVP. Drop 4c.5 F.5.2 hook.
//     g. validateGateKinds — assert every gate-kind string in
//     Template.Gates value slices is a member of the closed
//     GateKind enum (4b.1 hook).
//     h. validateAgentBindingEnvNames — assert every entry in each
//     AgentBinding.Env slice matches the closed env-var name regex
//     (`^[A-Za-z][A-Za-z0-9_]*$`), is non-empty, contains no `=`, and
//     is unique within its binding (Drop 4c F.7.17.1 hook).
//     i. validateAgentBindingContext — assert every AgentBinding.Context
//     sub-struct satisfies the closed delivery enum, non-negative MaxChars
//     and MaxRuleDuration, and that every kind referenced by the kind-walk
//     fields (SiblingsByKind / AncestorsByKind / DescendantsByKind) is a
//     member of the closed 12-value Kind enum. Drop 4c F.7.18.1 hook.
//     j. validateAgentBindingToolGating — assert every AgentBinding's
//     ToolsAllowed / ToolsDisallowed entries are non-empty + unique
//     within-binding; SystemPromptTemplatePath is project-relative,
//     traversal-free, and shell-metachar-free; Sandbox.Filesystem
//     AllowWrite / DenyRead entries are clean absolute paths;
//     Sandbox.Network AllowedDomains / DeniedDomains entries are
//     non-empty and carry no URL scheme. Drop 4c F.7.2 hook.
//     k. validateAgentBindingFiles — for every AgentBinding emit a warning
//     (NOT an error) when the resolved `~/.claude/agents/<name>.md` file
//     does not exist. Warn-only per Drop 4c.5 Q2 resolution: dev-machine
//     state is not template-correctness; adopters wanting strict-fail
//     wrap WarnLogger at the call site. Drop 4c.5 F.5.1 hook.
//     k'. validateAgentBindingNames — for every AgentBinding HARD-FAIL
//     when AgentName does not resolve at the embedded tier of the 3-tier
//     resolver (`internal/templates/builtin/agents/{till-gen,till-go,
//     till-gdd}/<name>.md`). Distinct from validateAgentBindingFiles:
//     warn-only is for dev-machine state; this hard-fail is for template
//     correctness — a dangling agent_name reference catches typos like
//     "buidler-agent" at Load rather than at spawn. Drop 4c.6 W0.5.D2
//     hook.
//     l. validateTillsyn — assert the top-level [tillsyn] globals satisfy
//     the closed contract: non-negative MaxContextBundleChars and
//     MaxAggregatorDuration (zero is legal — engine-time default
//     substitution; negative values are rejected) AND SpawnTempRoot is a
//     member of the closed {"", "os_tmp", "project"} enum. Drop 4c
//     F.7.18.2 hook (MaxContextBundleChars/MaxAggregatorDuration) +
//     F.7-CORE F.7.1 hook (SpawnTempRoot).
//
// Sentinel errors at package scope wrap the underlying failure so callers
// can use errors.Is for routing without reaching into pelletier/go-toml/v2
// internals.
func Load(r io.Reader) (Template, error) {
	return LoadWithOptions(r, LoadOptions{})
}

// LoadWithOptions is the all-fields-explicit variant of Load. See Load's
// godoc for the validation chain; see LoadOptions for the per-call knobs.
//
// Drop 4c.5 F.5.1: opts.WarnLogger receives one line per missing
// `~/.claude/agents/<name>.md` referenced by an AgentBinding. opts.StatFn
// (when non-nil) overrides os.Stat for the existence check — tests inject a
// deterministic stub. Both fields are zero-value-safe; nil WarnLogger drops
// warnings, nil StatFn falls back to os.Stat.
func LoadWithOptions(r io.Reader, opts LoadOptions) (Template, error) {
	if r == nil {
		return Template{}, errors.New("templates: nil reader")
	}

	raw, err := io.ReadAll(r)
	if err != nil {
		return Template{}, fmt.Errorf("templates: read: %w", err)
	}

	// Step 1+2 — tolerant version pre-pass. A separate Decoder without
	// DisallowUnknownFields decodes only the schema_version key. Any
	// unknown vocabulary in the document is ignored here so the version
	// check fires BEFORE strict-key validation.
	var versionProbe struct {
		SchemaVersion string `toml:"schema_version"`
	}
	if err := toml.NewDecoder(bytes.NewReader(raw)).Decode(&versionProbe); err != nil {
		// Pre-pass parse failures (malformed TOML, type mismatch on
		// schema_version) surface to the caller as the underlying parse
		// error so the user sees the position-aware DecodeError text.
		return Template{}, fmt.Errorf("templates: parse: %w", err)
	}
	if versionProbe.SchemaVersion != SchemaVersionV1 {
		return Template{}, fmt.Errorf("schema_version %q: %w", versionProbe.SchemaVersion, ErrUnsupportedSchemaVersion)
	}

	// Step 3 — strict decode of the full template. Unknown FIELDS inside a
	// known table (e.g. an unrecognized key inside a [kinds.build] row)
	// become StrictMissingError, which we wrap with ErrUnknownTemplateKey so
	// callers can route on the sentinel. Note this does NOT validate map
	// KEYS: TOML accepts arbitrary keys in [kinds.*] and [agent_bindings.*]
	// because those tables decode into maps. Bogus map keys are caught by
	// validateMapKeys in Step 4 below.
	var tpl Template
	strictDecoder := toml.NewDecoder(bytes.NewReader(raw))
	strictDecoder.DisallowUnknownFields()
	if err := strictDecoder.Decode(&tpl); err != nil {
		if strictErr, ok := errors.AsType[*toml.StrictMissingError](err); ok {
			return Template{}, fmt.Errorf("%w: %s", ErrUnknownTemplateKey, strictErr.String())
		}
		return Template{}, fmt.Errorf("templates: parse: %w", err)
	}

	// Step 4 — load-time validators. Order matters: map-key membership and
	// child-rule kind membership run first so cycle detection never
	// traverses a corrupt vocabulary.
	if err := validateMapKeys(&tpl); err != nil {
		return Template{}, err
	}
	if err := validateAgentMapKeys(&tpl); err != nil {
		return Template{}, err
	}
	if err := validateChildRuleKinds(tpl.ChildRules); err != nil {
		return Template{}, err
	}
	if err := validateChildRuleCycles(tpl.ChildRules); err != nil {
		return Template{}, err
	}
	if err := validateChildRuleRecursionDepth(tpl.ChildRules); err != nil {
		return Template{}, err
	}
	if err := validateBlockedByAcyclicity(tpl.ChildRules, opts.BlockedByGraphFn); err != nil {
		return Template{}, err
	}
	if err := validateClaimVsImplCoherence(tpl, opts.ClaimedConsumersFn); err != nil {
		return Template{}, err
	}
	if err := validateRequiredChildRules(tpl); err != nil {
		return Template{}, err
	}
	if err := validateChildRuleReachability(tpl); err != nil {
		return Template{}, err
	}
	if err := validateKindStructuralCoherence(tpl); err != nil {
		return Template{}, err
	}
	if err := validateGateKinds(tpl); err != nil {
		return Template{}, err
	}
	if err := validateAgentBindingEnvNames(tpl); err != nil {
		return Template{}, err
	}
	if err := validateAgentBindingContext(tpl); err != nil {
		return Template{}, err
	}
	if err := validateAgentBindingToolGating(tpl); err != nil {
		return Template{}, err
	}
	validateAgentBindingFiles(tpl, opts.WarnLogger, opts.StatFn)
	if err := validateAgentBindingNames(tpl, opts.AgentLookupFn); err != nil {
		return Template{}, err
	}
	if err := validateTillsyn(tpl); err != nil {
		return Template{}, err
	}

	return tpl, nil
}

// Sentinel errors returned by Load. Callers use errors.Is to route on the
// sentinel; the wrapped message preserves position-aware context from
// pelletier/go-toml/v2 or names the offending kind for UX.
var (
	// ErrUnknownTemplateKey is returned when strict decoding rejects a
	// top-level (or nested) key that has no matching struct field. The
	// reserved forward-compat [gate_rules] table is excluded — it lands on
	// Template.GateRulesRaw without triggering this error.
	ErrUnknownTemplateKey = errors.New("unknown template key")

	// ErrUnsupportedSchemaVersion is returned when the tolerant pre-pass
	// observes a schema_version that is not SchemaVersionV1, including the
	// empty-string case produced by a missing schema_version key.
	ErrUnsupportedSchemaVersion = errors.New("unsupported schema version")

	// ErrTemplateCycle is returned when the [child_rules] parent → child
	// kind graph contains a directed cycle. The wrapped message names the
	// participating kinds in path order.
	ErrTemplateCycle = errors.New("template child_rules contain a cycle")

	// ErrUnreachableChildRule is returned by validateChildRuleReachability
	// when a member of the closed 12-value domain.Kind enum is neither in
	// the closed reachabilityStandaloneKinds set nor referenced (as a
	// WhenParentKind or a CreateChildKind) by any [[child_rules]] entry.
	//
	// The validator is vacuously true for the embedded default templates
	// (till-go.toml + till-gen.toml ← default-go.toml + default-generic.toml,
	// rebadged in Drop 4c.6 W5.D1 + W5.D2) because their 4 standard
	// child_rules + 6 standalone-kinds classification cover every member of
	// the closed enum. The sentinel's real value is for ADOPTER templates
	// that strip [[child_rules]] entries — typo protection at template Load
	// time. Drop 4c.5 F.5.2 lit this sentinel by replacing the no-op
	// reachability stub with a real set-membership check.
	//
	// The wrapped message names the offending kind verbatim so adopters see
	// the exact rule they need to add (or the kind they need to remove from
	// [kinds] if their template legitimately drops a vocabulary entry).
	ErrUnreachableChildRule = errors.New("template child_rules contain an unreachable rule")

	// ErrIncoherentStructuralType is returned by validateKindStructuralCoherence
	// when a [kinds.X] row declares `structural_type = "drop"` AND no
	// [[child_rules]] entry has `when_parent_kind = X`. A drop structural type
	// names a kind that decomposes into cascade work; a drop kind with no
	// auto-create children is structurally orphaned.
	//
	// The check is restricted to structural_type=drop today; other structural
	// types (droplet / segment / confluence) do not gate on child_rules
	// presence in Drop 4c.5. Full structural_type ↔ kind ↔ role coherence
	// validation is post-MVP.
	//
	// The wrapped message names the offending kind, the structural_type
	// value, and the missing-rule shape so adopters see the exact line they
	// need to add (a [[child_rules]] entry with when_parent_kind set to the
	// offending kind).
	ErrIncoherentStructuralType = errors.New("template kind has incoherent structural_type")

	// ErrUnknownKindReference is returned when a [child_rules] entry
	// references a kind that is not a member of the closed 12-value Kind
	// enum, or when a [kinds.*] / [agent_bindings.*] map key does likewise.
	ErrUnknownKindReference = errors.New("template references an unknown kind")

	// ErrInvalidAgentBinding is returned by AgentBinding.Validate when one of
	// its fields fails the rules in main/PLAN.md § 19.3 lines 1653-1656
	// (empty agent_name/model, non-positive max_tries/max_turns, negative
	// max_budget_usd/blocked_retries/blocked_retry_cooldown). The wrapped
	// message names the offending field and the offending value for UX.
	ErrInvalidAgentBinding = errors.New("invalid agent binding")

	// ErrUnknownGateKind is returned by validateGateKinds when a value-slice
	// element under Template.Gates is not a member of the closed GateKind
	// enum (templates.IsValidGateKind). The wrapped message names the parent
	// kind and the offending gate-kind string for UX.
	ErrUnknownGateKind = errors.New("template references an unknown gate kind")

	// ErrInvalidAgentBindingEnv is returned by validateAgentBindingEnvNames
	// when an AgentBinding.Env slice contains an entry that fails the closed
	// env-var name contract (Drop 4c F.7.17.1 locked decision L5 + REV-2
	// expanded baseline). Each entry MUST match `^[A-Za-z][A-Za-z0-9_]*$`,
	// be non-empty, contain no `=`, and be unique within its binding. The
	// wrapped message names the offending kind, the offending entry, and
	// the failure reason for UX. The error wraps ErrInvalidAgentBinding so
	// callers using `errors.Is(err, ErrInvalidAgentBinding)` continue to
	// work.
	ErrInvalidAgentBindingEnv = fmt.Errorf("%w: env", ErrInvalidAgentBinding)

	// ErrInvalidContextRules is returned by validateAgentBindingContext when an
	// AgentBinding.Context sub-struct contains a field that fails the closed
	// rule contract (Drop 4c F.7.18.1 acceptance criteria):
	//
	//   - Delivery is set to a value outside the closed enum
	//     {"", "inline", "file"}.
	//   - MaxChars is negative (zero is legal — engine-time default applies).
	//   - MaxRuleDuration is negative (zero is legal — engine-time default
	//     applies).
	//
	// Kind references inside SiblingsByKind / AncestorsByKind /
	// DescendantsByKind are validated against the closed 12-value
	// domain.Kind enum and surface as ErrUnknownKindReference (consistent
	// with the existing kinds-map / child-rules / agent-bindings-map
	// vocabulary checks).
	//
	// The error wraps ErrInvalidAgentBinding so callers using
	// `errors.Is(err, ErrInvalidAgentBinding)` continue to route correctly
	// without reaching for the context-specific sentinel.
	ErrInvalidContextRules = fmt.Errorf("%w: context", ErrInvalidAgentBinding)

	// ErrInvalidAgentBindingToolGating is returned by
	// validateAgentBindingToolGating when an AgentBinding's tool-gating /
	// system-prompt-template / sandbox fields fail the Drop 4c F.7.2 closed
	// contract:
	//
	//   - ToolsAllowed / ToolsDisallowed entries are empty strings or
	//     within-binding duplicates.
	//   - SystemPromptTemplatePath is absolute (begins with `/`), contains
	//     `..` traversal segments, or contains shell metacharacters
	//     `;` `|` `&` backtick `$`.
	//   - Sandbox.Filesystem.AllowWrite / DenyRead entries are non-absolute,
	//     contain `..`, or contain double-slashes (path is not clean).
	//   - Sandbox.Network.AllowedDomains / DeniedDomains entries are empty
	//     strings or carry a URL scheme prefix (`http://`, `https://`).
	//
	// The error wraps ErrInvalidAgentBinding so callers using
	// `errors.Is(err, ErrInvalidAgentBinding)` continue to route correctly
	// without reaching for the tool-gating-specific sentinel. The wrapped
	// message names the offending kind, the offending field, and the failure
	// reason for UX.
	ErrInvalidAgentBindingToolGating = fmt.Errorf("%w: tool_gating", ErrInvalidAgentBinding)

	// ErrMissingRequiredChildRule is returned by validateRequiredChildRules
	// when a parent kind that is declared in [kinds] is missing one of its
	// REQUIRED auto-create child rules. Drop 4c.5 F.5.1 fixes the closed
	// required-set:
	//
	//   - kind=plan  → MUST have [[child_rules]] entries creating
	//                  `plan-qa-proof` AND `plan-qa-falsification`.
	//   - kind=build → MUST have [[child_rules]] entries creating
	//                  `build-qa-proof` AND `build-qa-falsification`.
	//
	// The check is conditional on the parent kind being declared in [kinds];
	// adopter templates that strip `kind=plan` or `kind=build` entirely (a
	// pre-MVP-rare but spec-permitted shape) do not trigger this validator.
	// Rationale: required-rules-for-undeclared-parents would over-fire on
	// language-agnostic templates that delegate kind declarations to a
	// project-local override.
	//
	// The wrapped message names the parent kind and the missing child kind
	// verbatim so adopters see the exact rule they need to add.
	ErrMissingRequiredChildRule = errors.New("template missing required child rule")

	// ErrInvalidTillsynGlobals is returned by validateTillsyn when the
	// top-level [tillsyn] table contains a field that fails the closed
	// rule contract (Drop 4c F.7.18.2 + F.7-CORE F.7.1 acceptance):
	//
	//   - MaxContextBundleChars is negative (zero is legal — engine-time
	//     default substitution applies per master PLAN L14).
	//   - MaxAggregatorDuration is negative (zero is legal — engine-time
	//     default substitution applies per master PLAN L15).
	//   - SpawnTempRoot is set to a value outside the closed enum
	//     {"", "os_tmp", "project"} (empty is legal — F.7.1 NewBundle
	//     resolves the empty string to "os_tmp" at spawn time).
	//
	// The wrapped message names the offending field and the offending
	// value for UX. The sentinel is a top-level Load error rather than a
	// nested wrap of ErrInvalidAgentBinding because the [tillsyn] table is
	// distinct from [agent_bindings] — failures here are global, not
	// per-binding.
	ErrInvalidTillsynGlobals = errors.New("invalid tillsyn globals")

	// ErrUnknownAgentName is returned by validateAgentBindingNames when an
	// AgentBinding.AgentName does not resolve at the EMBEDDED tier of the
	// 3-tier agent resolver (per SKETCH.md §3.4: project →
	// `<projectRoot>/.tillsyn/agents/<name>.md` → user
	// `~/.tillsyn/agents/<group>/<name>.md` → embedded
	// `internal/templates/builtin/agents/{till-gen,till-go,till-gdd}/<name>.md`).
	// Resolution succeeds if the agent .md file exists in ANY of the three
	// embedded groups — that is the floor every binding's name must clear.
	//
	// The check is HARD-FAIL (distinct from validateAgentBindingFiles, which
	// is warn-only against `~/.claude/agents/<name>.md` for dev-machine
	// state). A dangling agent_name like "buidler-agent" (transposed
	// letters) silently survives Load today and surfaces only when a
	// dispatcher attempts to spawn the kind — this sentinel catches the
	// typo at Load time so adopters see the failure at template-author
	// time, not at first-spawn time.
	//
	// Empty AgentName is also rejected here (the empty string cannot
	// resolve to any `<group>/.md` file path). The existing
	// AgentBinding.Validate sentinel ErrInvalidAgentBinding already covers
	// empty AgentName for downstream consumers of the AgentBinding type;
	// this validator covers it at Load so Load is a closed contract on its
	// own.
	//
	// The wrapped message names the binding's parent kind and the
	// offending agent_name verbatim so adopters see the exact line they
	// need to fix. Per W0.5 plan FF1 disclosure: pelletier/go-toml/v2's
	// post-decode validators do not carry source-line numbers, so
	// adopters grep their TOML for the field path.
	//
	// Drop 4c.6 W0.5.D2 hook.
	ErrUnknownAgentName = errors.New("template references an unknown agent name")

	// ErrChildRuleRecursionTooDeep is returned by validateChildRuleRecursionDepth
	// when the parent→child kind graph induced by [[child_rules]] contains a
	// reachable depth greater than childRuleRecursionDepthMax (5 by default
	// per SKETCH.md § 26.W0.5: "default 5; configurable post-MVP via
	// template").
	//
	// "Depth" is measured in EDGES from any root: a chain
	// k0 → k1 → k2 → k3 → k4 → k5 has depth 5 (5 edges, 6 nodes) and PASSES
	// the bound. Adding one more edge — k0 → ... → k6 — pushes the depth to
	// 6 and trips this sentinel.
	//
	// The wrapped message names the offending kind, the observed depth, the
	// bound, and the path from a root that achieved the depth (rendered with
	// formatCyclePath's " -> " separator so the diagnostic UX is visually
	// consistent with cycle errors). The graph is a DAG by the time this
	// validator runs because validateChildRuleCycles (the chain step
	// immediately preceding D4) rejects every cycle with ErrTemplateCycle —
	// cyclic graphs have unbounded depth and the cycle is the better
	// diagnostic, so D4 never has to handle them.
	//
	// Per W0.5 plan FF1 disclosure: pelletier/go-toml/v2's post-decode
	// validators do not carry source-line numbers, so adopters grep their
	// TOML for the participating [[child_rules]] chain rather than jumping
	// to a `line=N` pointer.
	//
	// The bound is a Go-internal constant for Drop 4c.6 — adopter templates
	// cannot raise or lower it. Post-MVP refinement: a `[tillsyn]
	// recursion_depth_max = N` field gives adopters template-level control.
	//
	// Drop 4c.6 W0.5.D4 hook.
	ErrChildRuleRecursionTooDeep = errors.New("template child_rules exceed recursion depth bound")

	// ErrTemplateBlockedByCycle is returned by validateBlockedByAcyclicity
	// when the kind-level blocked_by graph contains a directed cycle. The
	// validator is the load-time mirror of Drop 4a Wave 1.7's runtime
	// BlockedBy acyclicity check on action-item UUIDs (see
	// internal/domain/action_item.go) — same DFS shape, same back-edge
	// rejection, but operating on KINDS at template Load rather than on
	// action-item UUIDs at create time.
	//
	// Distinct from ErrTemplateCycle — D3's unified-graph cycle detector
	// (validateChildRuleCycles) walks the parent→child auto-create graph
	// AND the BlockedByParent-induced graph in one pass and reports
	// whichever edge set produces the cycle first. ErrTemplateCycle is the
	// sentinel for cycles found by that detector. ErrTemplateBlockedByCycle
	// is the sentinel for cycles in the standalone blocked_by graph that
	// validateBlockedByAcyclicity walks independently — useful when the
	// schema gains richer kind-level blocked_by edges (e.g. a
	// `BlockedByKinds []domain.Kind` field on ChildRule) whose graph
	// diverges from the parent→child auto-create graph.
	//
	// Today's coupled cycles (every BlockedByParent=true rule contributing
	// one edge to BOTH graphs) are caught by D3 with ErrTemplateCycle
	// BEFORE D5 runs — the chain order is pinned by
	// TestLoadValidatesBlockedByAcyclicityRunsAfterChildRuleCycles. D5's
	// production effect on today's schema is therefore vacuously satisfied;
	// the sentinel + validator + tests ship now so future schema
	// expansions inherit acyclicity for free.
	//
	// The wrapped message names the participating kinds in path order with
	// the [blocked_by] edge label appended (mirroring D3's formatCyclePath
	// shape with the [parent->child] vs [blocked_by] edge-type label).
	//
	// Drop 4c.6 W0.5.D5 hook.
	ErrTemplateBlockedByCycle = errors.New("template blocked_by edges form a cycle")

	// ErrClaimVsImplUnknownConsumer is returned by validateClaimVsImplCoherence
	// when a template claims a feature whose consumer identifier is not a
	// member of the closed Go-internal `knownWiredConsumers` map. The
	// validator exists to prevent the "shipped-but-not-wired" anti-pattern
	// (Drop 3 droplet 3.20): a schema feature ships without a runtime
	// consumer, adopters author against it, and Load silently accepts the
	// claim with no diagnostic until the dispatcher reaches the unwired
	// path at runtime.
	//
	// For Drop 4c.6 the `knownWiredConsumers` map is INTENTIONALLY EMPTY
	// per L1 W0.5 sub-plan container Acceptance bullet 4 + Open Question #1
	// resolution. Drop 4c.7 W7 adds `child_rules_for` and Drop 4c.7 W8 adds
	// `context_resolve` when those waves wire the first real runtime
	// consumers. Pre-4c.7 the validator's only meaningful exercised path is
	// the test-seam injection (LoadOptions.ClaimedConsumersFn); production
	// callers leave the field nil and inherit the empty production walker,
	// so the validator vacuously passes on every embedded default template.
	//
	// LOUD WARNING TO FUTURE DROPS: adding a runtime consumer for a
	// template-claimed feature requires adding the consumer's identifier to
	// `knownWiredConsumers` in this file. Failing to do so will cause every
	// template that claims the new feature to fail Load with this sentinel.
	// Conversely, adding an entry to `knownWiredConsumers` WITHOUT also
	// wiring the runtime consumer recreates the anti-pattern this validator
	// exists to prevent — TestLoadValidatesClaimVsImplCoherenceEmptyKnownWired-
	// SetGuard pins the Drop 4c.6 invariant; that guard's expected length
	// advances when Drop 4c.7 W7 + W8 land.
	//
	// The wrapped message names the offending consumer identifier so
	// adopters can grep their TOML for whatever schema field claims that
	// consumer once the schema gains a `[[child_rules]] consumer = "..."`
	// axis. Per W0.5 plan FF1 disclosure: pelletier/go-toml/v2's
	// post-decode validators do not carry source-line numbers, so the
	// message names the field-path rather than `line=N`.
	//
	// The validator does NOT parse `CLAUDE.md` at runtime. The closed
	// `knownWiredConsumers` Go map is the source-of-truth; CLAUDE.md §
	// Cascade Tree Structure is the authoring reference for adopters but
	// is not consulted at Load.
	//
	// Drop 4c.6 W0.5.D6 hook.
	ErrClaimVsImplUnknownConsumer = errors.New("template claims a feature with no wired consumer")
)

// childRuleRecursionDepthMax bounds the maximum reachable depth (counted in
// edges from any root) of the parent→child kind graph induced by
// [[child_rules]]. Default 5 per SKETCH.md § 26.W0.5; the constant is
// package-internal because adopter-template control of the bound lands
// post-MVP via a `[tillsyn] recursion_depth_max = N` schema field.
//
// LOUD WARNING TO FUTURE DROPS: lowering this constant is a soft-breaking
// change against any adopter template whose chain depth is ≤ the old bound
// but > the new bound. Raise it freely; lower it only via a deprecation
// cycle that surfaces the new bound through a non-fatal warning before
// flipping to hard-fail.
const childRuleRecursionDepthMax = 5

// validateMapKeys asserts every key in Template.Kinds,
// Template.AgentBindings, and Template.Gates is a member of the closed
// 12-value domain.Kind enum AND canonicalizes those keys to their lowercase
// form so consumer-side lookups by domain.KindBuild succeed even when the
// authoring template wrote [kinds.BUILD] / [gates.Build] / etc. Catches typos
// like [kinds.bulid] (transposed letters), [agent_bindings.totally-bogus], or
// [gates.bogus-kind] at load time rather than letting them silently coexist
// with the real entries — strict decode validates fields inside a row but not
// the map keys themselves, because pelletier/go-toml/v2 treats arbitrary keys
// as legitimate map entries when the destination type is a map.
//
// Drop 4c.5 E.6 fix-path decision: post-decode canonicalization (NOT
// exact-match rejection). Rationale: domain.IsValidKind already case-folds
// (kind.go:50-52 trims + lowers before slice-contains), so the validation
// surface ALREADY tolerates uppercase. Forcing exact-match here would diverge
// the value-validation contract from the key-validation contract for no
// adopter-visible win. Canonicalization keeps the load surface tolerant of
// authoring case-drift while ensuring downstream consumers can index by the
// canonical lowercase domain.Kind constants without first re-folding.
//
// The signature takes *Template (not Template by value) so the canonicalized
// rebuild is visible to the caller. Each map is rebuilt only if at least one
// key actually canonicalized (cheap pre-scan), to avoid touching the map's
// underlying allocation on the all-lowercase happy path that the embedded
// default templates exercise.
//
// Collision: if two distinct TOML keys (e.g. [gates.BUILD] AND [gates.build])
// canonicalize to the same domain.Kind, the rebuild detects the collision and
// returns ErrUnknownKindReference wrapping a message that names the offending
// kind. The TOML decoder accepts the two as legitimate sibling tables (probed
// 2026-05-05 against pelletier/go-toml/v2 — case-sensitive at the TOML layer);
// the collision surfaces only after canonicalization.
func validateMapKeys(tpl *Template) error {
	if rebuilt, err := canonicalizeMapKeys(tpl.Kinds, "kinds"); err != nil {
		return err
	} else if rebuilt != nil {
		tpl.Kinds = rebuilt
	}
	if rebuilt, err := canonicalizeMapKeys(tpl.AgentBindings, "agent_bindings"); err != nil {
		return err
	} else if rebuilt != nil {
		tpl.AgentBindings = rebuilt
	}
	if rebuilt, err := canonicalizeMapKeys(tpl.Gates, "gates"); err != nil {
		return err
	} else if rebuilt != nil {
		tpl.Gates = rebuilt
	}
	return nil
}

// validateAgentMapKeys asserts every key in Template.Agents is a member of
// the closed 12-value domain.Kind enum and canonicalises those keys to their
// lowercase form, mirroring validateMapKeys' contract for the existing
// Kinds / AgentBindings / Gates maps. Drop 4c.6 W0.5.D1 introduced this
// validator alongside the Template.Agents stub field so the closed-enum
// invariant gates the new `[agents.<kind>]` TOML table at load time —
// catching typos like `[agents.totally-bogus]` before any runtime consumer
// (W0 wires those) silently misses the binding.
//
// Reuses canonicalizeMapKeys verbatim. Adding a separate validator (rather
// than extending validateMapKeys to fold Agents into its body) keeps the
// W0.5 load-chain insertion explicit per PLAN.md § "Cross-Cutting Decisions
// / Tradeoffs" → "Validator chain insertion point" — the W0.5 plan inserts
// validateAgentMapKeys after validateMapKeys at the LoadWithOptions chain
// site so adopters who diff the chain order see a separate D1 step.
//
// TOML-line pointers in the wrapped error: pelletier/go-toml/v2's
// post-decode validators do NOT receive original-source line numbers —
// canonicalizeMapKeys names the offending field path ("agents") and the
// offending key verbatim so adopters can grep their TOML for the bad
// entry. The W0.5 plan accepts this as a stable mitigation pending any
// upstream go-toml/v2 API extension that exposes per-key positions.
func validateAgentMapKeys(tpl *Template) error {
	if rebuilt, err := canonicalizeMapKeys(tpl.Agents, "agents"); err != nil {
		return err
	} else if rebuilt != nil {
		tpl.Agents = rebuilt
	}
	return nil
}

// canonicalizeMapKeys validates and canonicalizes every key in m. Returns
// (rebuilt, nil) when canonicalization actually changed at least one key
// (caller should swap the map), (nil, nil) when every key was already
// canonical (caller leaves the map alone), or (nil, err) when validation
// fails — either an unknown kind or a post-canonicalization collision.
//
// The fieldName argument names the TOML field ("kinds" / "agent_bindings" /
// "gates") so the error UX points adopters at the exact line they need to
// fix.
//
// Generic over the value type so all three Template maps share the same
// validation + canonicalization path. The constraint is `any` because Go
// generics do not let us express "any value type V such that map[Kind]V is
// the destination" more precisely; the function is invariant in V.
func canonicalizeMapKeys[V any](m map[domain.Kind]V, fieldName string) (map[domain.Kind]V, error) {
	if len(m) == 0 {
		return nil, nil
	}
	// Pre-scan: validate every key + detect whether any key needs
	// canonicalization. The all-lowercase happy path returns early without
	// allocating a new map.
	needsRebuild := false
	for k := range m {
		if !domain.IsValidKind(k) {
			return nil, fmt.Errorf("%w: %s map key %q", ErrUnknownKindReference, fieldName, k)
		}
		if domain.Kind(strings.ToLower(strings.TrimSpace(string(k)))) != k {
			needsRebuild = true
		}
	}
	if !needsRebuild {
		return nil, nil
	}
	// Rebuild with canonicalized keys. A post-canonicalization collision
	// (two distinct authoring keys folding to the same domain.Kind) surfaces
	// here as ErrUnknownKindReference wrapping a message that names the
	// duplicated canonical form.
	rebuilt := make(map[domain.Kind]V, len(m))
	for k, v := range m {
		canon := domain.Kind(strings.ToLower(strings.TrimSpace(string(k))))
		if _, dup := rebuilt[canon]; dup {
			return nil, fmt.Errorf("%w: %s map has duplicate key %q after case-fold canonicalization", ErrUnknownKindReference, fieldName, canon)
		}
		rebuilt[canon] = v
	}
	return rebuilt, nil
}

// validateChildRuleKinds asserts every Kind referenced in [child_rules] is a
// member of the closed 12-value enum. The check runs before cycle detection
// so the graph traversal never encounters a corrupt vocabulary.
func validateChildRuleKinds(rules []ChildRule) error {
	for _, rule := range rules {
		if !domain.IsValidKind(rule.WhenParentKind) {
			return fmt.Errorf("%w: when_parent_kind %q", ErrUnknownKindReference, rule.WhenParentKind)
		}
		if !domain.IsValidKind(rule.CreateChildKind) {
			return fmt.Errorf("%w: create_child_kind %q", ErrUnknownKindReference, rule.CreateChildKind)
		}
	}
	return nil
}

// validateChildRuleCycles runs DFS over the kind graph derived from
// [child_rules] to detect directed cycles. The validator walks BOTH edge
// sets in a unified-graph DFS pass:
//
//  1. The parent→child auto-create graph (every rule contributes an edge
//     from rule.WhenParentKind to rule.CreateChildKind).
//  2. The blocked_by-induced graph (every rule whose BlockedByParent is
//     true contributes an edge from rule.CreateChildKind back to
//     rule.WhenParentKind, encoding "child cannot start until parent
//     terminal-completes" at the kind level).
//
// The two graphs are checked for cycles independently — a cycle in either
// edge set surfaces as ErrTemplateCycle wrapping a cycle-path string with
// the edge-type label appended ("[parent->child]" or "[blocked_by]") so
// adopters know which rule wiring is at fault. Drop 4c.6 W0.5.D3 introduced
// the unified-graph extension; the parent→child detection mirrors the
// pre-W0.5.D3 behaviour for back-compat with existing tests.
//
// The DFS root iteration uses sorted-key order via dfsDetectCycle so the
// wrapped cycle-path message is reproducible across runs / OSes / Go's
// map-iteration randomness. The previous implementation used `for node :=
// range graph` (non-deterministic); W0.5.D3 round-2 FF3 fixed that as part
// of the helper extraction.
//
// Today's schema couples the two edge sets — every rule with
// BlockedByParent=true contributes one edge to each graph, so today every
// blocked_by cycle is also a parent→child cycle. The unified DFS still
// reports the first cycle it finds (parent→child first by validator chain
// order); the second-edge-set pass is forward-looking for future schema
// additions (e.g. a hypothetical BlockedByKinds []domain.Kind field whose
// edges are decoupled from the parent→child auto-create graph).
func validateChildRuleCycles(rules []ChildRule) error {
	if len(rules) == 0 {
		return nil
	}

	parentChildGraph := make(map[domain.Kind][]domain.Kind, len(rules))
	blockedByGraph := make(map[domain.Kind][]domain.Kind)
	for _, rule := range rules {
		parentChildGraph[rule.WhenParentKind] = append(parentChildGraph[rule.WhenParentKind], rule.CreateChildKind)
		if rule.BlockedByParent {
			blockedByGraph[rule.CreateChildKind] = append(blockedByGraph[rule.CreateChildKind], rule.WhenParentKind)
		}
	}

	if cycle, found := dfsDetectCycle(parentChildGraph); found {
		return fmt.Errorf("%w: %s [parent->child]", ErrTemplateCycle, formatCyclePath(cycle))
	}
	if cycle, found := dfsDetectCycle(blockedByGraph); found {
		return fmt.Errorf("%w: %s [blocked_by]", ErrTemplateCycle, formatCyclePath(cycle))
	}
	return nil
}

// dfsDetectCycle runs a colored-DFS cycle detector over the supplied
// directed graph and returns the cycle path (closure node included as the
// final element) plus a found flag. Roots are iterated in sorted order over
// the graph's keys so the returned cycle path is reproducible across runs,
// OSes, and Go's map-iteration randomness. The colored-DFS pattern (white /
// gray / black) is preserved from the pre-extraction validateChildRuleCycles
// implementation per Drop 3 finding 5.B.4.
//
// The generic constraint is `~string` rather than the broader `comparable`
// because every caller in this package keys its graphs by domain.Kind (a
// string-typed enum), and ~string lets the helper sort roots without
// requiring callers to project keys to []string + back. Drop 4c.6 W0.5.D4
// and W0.5.D5 reuse this helper for the recursion-depth bound and the
// blocked_by-acyclicity check respectively, so the constraint must cover
// every cascade kind-keyed graph.
//
// The returned cyclePath starts with the closure-recursion-stack-entry
// kind, walks the stack to the back-edge target, and ends with the closure
// kind appended one more time so the rendered string makes the cycle's
// closure visually obvious (e.g. "kindA -> kindB -> kindA"). On a cycle-
// free graph the helper returns (nil, false).
func dfsDetectCycle[K ~string](graph map[K][]K) (cyclePath []K, found bool) {
	if len(graph) == 0 {
		return nil, false
	}

	const (
		colorWhite = 0 // unseen
		colorGray  = 1 // on current DFS path
		colorBlack = 2 // fully explored
	)
	color := make(map[K]int, len(graph))

	var resultPath []K
	var dfs func(node K, stack []K) bool
	dfs = func(node K, stack []K) bool {
		color[node] = colorGray
		stack = append(stack, node)
		for _, next := range graph[node] {
			switch color[next] {
			case colorGray:
				resultPath = append(append([]K{}, stack...), next)
				return true
			case colorWhite:
				if dfs(next, stack) {
					return true
				}
			}
		}
		color[node] = colorBlack
		return false
	}

	roots := make([]string, 0, len(graph))
	for k := range graph {
		roots = append(roots, string(k))
	}
	sort.Strings(roots)

	for _, root := range roots {
		node := K(root)
		if color[node] == colorWhite {
			if dfs(node, nil) {
				return resultPath, true
			}
		}
	}
	return nil, false
}

// formatCyclePath renders a cycle's traversal as a "kindA -> kindB -> kindA"
// string for the wrapped error message. The cyclePath input is what
// dfsDetectCycle returns: the recursion stack from the closure node to the
// back-edge target with the closure node appended one more time. The first
// occurrence of the closure-recursion-stack-entry kind is treated as the
// cycle's start so prefix nodes that led TO the cycle but are not part of
// the cycle itself are stripped from the rendering.
//
// Drop 4c.6 W0.5.D3 generalised the helper from `domain.Kind`-only to a
// type-parameterised over `~string` constraint so D4 (recursion-depth path
// rendering) and D5 (blocked_by-acyclicity path rendering) can reuse the
// same renderer.
func formatCyclePath[K ~string](cyclePath []K) string {
	if len(cyclePath) == 0 {
		return ""
	}
	closure := cyclePath[len(cyclePath)-1]
	startIdx := 0
	for idx, k := range cyclePath {
		if k == closure {
			startIdx = idx
			break
		}
	}
	parts := make([]string, 0, len(cyclePath)-startIdx)
	for _, k := range cyclePath[startIdx:] {
		parts = append(parts, string(k))
	}
	return strings.Join(parts, " -> ")
}

// validateChildRuleRecursionDepth walks the parent→child kind graph induced
// by [[child_rules]] and rejects any reachable depth that exceeds
// childRuleRecursionDepthMax. "Depth" is counted in edges from any root: a
// chain k0 → k1 → k2 → k3 → k4 → k5 has depth 5 (5 edges, 6 nodes) and
// PASSES the bound; one more edge trips ErrChildRuleRecursionTooDeep.
//
// Algorithm — DAG longest-path with memoised DFS:
//
//  1. Build the parent→child graph (one edge per [[child_rules]] entry,
//     from rule.WhenParentKind to rule.CreateChildKind). The graph is a
//     DAG by the time this validator runs because validateChildRuleCycles
//     (the chain step immediately preceding D4 in LoadWithOptions)
//     rejected every cycle. Cyclic input would either infinite-loop the
//     recursive depth DFS or surface as ErrChildRuleRecursionTooDeep with
//     a misleading path — neither is reachable here.
//  2. For each kind in the graph, compute the longest path that begins at
//     that kind via memoised recursion: depth[k] = 1 + max(depth[child])
//     over k's out-edges; leaves have depth 0. Memoisation makes the walk
//     linear in the number of edges; without it a diamond shape A→B,
//     A→C, B→D, C→D would visit D twice.
//  3. Iterate the graph's roots in sorted-key order (mirrors
//     dfsDetectCycle's reproducibility contract). The first kind whose
//     depth exceeds childRuleRecursionDepthMax wins the diagnostic — its
//     longest-path rendering is appended to the wrapped error message via
//     formatCyclePath's " -> " separator.
//
// Multi-root handling: when several kinds qualify as roots (no incoming
// edges), the validator iterates them in sort.Strings order so the
// diagnostic is reproducible across runs / OSes / Go map-iteration
// randomness. Disjoint subgraphs are walked independently; the deepest
// chain anywhere in the graph wins the diagnostic.
//
// Empty [[child_rules]] is the trivial pass case — len(graph) == 0 returns
// nil immediately. Single-edge graphs (depth 1) and any chain up to depth
// 5 also pass.
//
// Drop 4c.6 W0.5.D4 hook. The validator is wired into LoadWithOptions
// immediately after validateChildRuleCycles per the L2 PLAN insertion-point
// directive — that ordering is load-bearing because cyclic graphs would
// either infinite-loop the depth DFS or surface as a misleading
// recursion-too-deep error rather than the actionable cycle path. The
// chain ordering is pinned by TestLoadValidatesChildRuleRecursionDepthRunsAfterCycleDetection.
func validateChildRuleRecursionDepth(rules []ChildRule) error {
	if len(rules) == 0 {
		return nil
	}

	graph := make(map[domain.Kind][]domain.Kind, len(rules))
	for _, rule := range rules {
		graph[rule.WhenParentKind] = append(graph[rule.WhenParentKind], rule.CreateChildKind)
	}

	// Memoised longest-path-from-here. depthFrom[k] = max edges in any
	// path that begins at k. Built via DFS from every kind appearing as a
	// graph key; leaf kinds (no out-edges in the graph) implicitly have
	// depth 0 because depthFrom[leaf] is read as the zero value of int.
	// successorOnLongest[k] picks the out-edge that achieved the longest
	// path so the final rendering can walk the chain back from the
	// offending root.
	depthFrom := make(map[domain.Kind]int, len(graph))
	successorOnLongest := make(map[domain.Kind]domain.Kind, len(graph))
	visited := make(map[domain.Kind]bool, len(graph))

	var compute func(node domain.Kind) int
	compute = func(node domain.Kind) int {
		if d, ok := depthFrom[node]; ok {
			return d
		}
		if visited[node] {
			// Defense-in-depth: validateChildRuleCycles already rejected
			// every cycle, so this branch is unreachable in production.
			// If it ever fires, treat the node as a leaf (depth 0)
			// rather than infinite-looping. The depth-bound check still
			// fires correctly for the longest acyclic prefix.
			return 0
		}
		visited[node] = true

		bestChild := domain.Kind("")
		bestDepth := -1
		for _, child := range graph[node] {
			childDepth := compute(child)
			if childDepth > bestDepth {
				bestDepth = childDepth
				bestChild = child
			}
		}
		if bestDepth < 0 {
			// Leaf in the graph (kind has no out-edges). Depth 0.
			depthFrom[node] = 0
			return 0
		}
		depthFrom[node] = bestDepth + 1
		successorOnLongest[node] = bestChild
		return depthFrom[node]
	}

	roots := make([]string, 0, len(graph))
	for k := range graph {
		roots = append(roots, string(k))
	}
	sort.Strings(roots)

	for _, rootStr := range roots {
		root := domain.Kind(rootStr)
		depth := compute(root)
		if depth <= childRuleRecursionDepthMax {
			continue
		}
		// Walk the longest-path successor chain from root to render the
		// diagnostic. The chain length is depth + 1 (nodes vs edges).
		path := make([]domain.Kind, 0, depth+1)
		node := root
		for {
			path = append(path, node)
			next, ok := successorOnLongest[node]
			if !ok {
				break
			}
			node = next
		}
		return fmt.Errorf(
			"%w: kind %q reaches depth %d (max %d): %s",
			ErrChildRuleRecursionTooDeep,
			root,
			depth,
			childRuleRecursionDepthMax,
			formatChainPath(path),
		)
	}
	return nil
}

// formatChainPath renders an acyclic kind chain as a "kindA -> kindB -> kindC"
// string for the depth-bound diagnostic. Distinct from formatCyclePath
// because depth paths have NO closure node — the chain ends at a leaf, and
// every node in the slice is a distinct member of the rendered path. Reusing
// formatCyclePath here would mis-handle the chain because that helper treats
// the last element as the cycle's closure node and strips every prefix node
// that led TO it.
//
// Drop 4c.6 W0.5.D4 hook. The renderer is type-parameterised over `~string`
// to mirror formatCyclePath's signature so future depth-style validators can
// reuse it without forcing a domain.Kind-specific projection.
func formatChainPath[K ~string](chain []K) string {
	if len(chain) == 0 {
		return ""
	}
	parts := make([]string, 0, len(chain))
	for _, k := range chain {
		parts = append(parts, string(k))
	}
	return strings.Join(parts, " -> ")
}

// validateBlockedByAcyclicity walks the kind-level blocked_by graph induced
// by [[child_rules]] and rejects directed cycles with
// ErrTemplateBlockedByCycle. The validator is the load-time mirror of Drop
// 4a Wave 1.7's runtime BlockedBy acyclicity check on action-item UUIDs (see
// internal/domain/action_item.go's blocked_by-acyclicity validator) — same
// colored-DFS shape, same back-edge rejection, but operating on KINDS at
// template Load rather than on action-item UUIDs at create time.
//
// graphFn parameter — test-only injection point:
//
//   - When graphFn is non-nil, the validator walks the supplied graph
//     verbatim. Tests use this to exercise a synthetic kind-level cycle
//     against a schema (today's `BlockedByParent bool`) whose production
//     graph cannot otherwise express one. The injection is the floor of
//     forward-looking coverage: when a future schema expansion gives
//     ChildRule a richer blocked_by axis (e.g. a `BlockedByKinds
//     []domain.Kind` field), production code will produce real cyclic
//     graphs and the injection point becomes vestigial.
//
//   - When graphFn is nil, the production walker builds the graph from
//     every rule whose BlockedByParent is true. Each such rule contributes
//     one edge from rule.CreateChildKind to rule.WhenParentKind (child→
//     parent: child cannot start until parent terminal-completes). Today
//     the resulting graph is a forest — every edge is a child→parent edge
//     and parents have no incoming blocked_by edges, so no cycle can
//     exist. The validator returns nil for every embedded default
//     template.
//
// D3 vs D5 — chain-order contract:
//
//   - D3's validateChildRuleCycles walks a UNIFIED graph (parent→child
//     auto-create AND BlockedByParent-induced edges in one pass) and
//     reports whichever edge set fires first. Today every
//     BlockedByParent=true rule contributes one edge to BOTH the
//     parent→child graph AND the blocked_by graph, so coupled cycles
//     surface as ErrTemplateCycle in D3 BEFORE D5 ever runs.
//
//   - D5's validateBlockedByAcyclicity walks ONLY the blocked_by subgraph
//     and reports its cycles as ErrTemplateBlockedByCycle. The validator
//     exists so future schema additions (kind-level blocked_by axes that
//     diverge from the parent→child auto-create graph) inherit acyclicity
//     for free without requiring D3's body to be re-thought.
//
// Per W0.5 plan FF1 disclosure: pelletier/go-toml/v2's post-decode
// validators do not carry source-line numbers, so the wrapped sentinel
// message names the participating kinds in path order (mirroring D3's
// formatCyclePath shape with the [blocked_by] edge label) rather than
// `line=N`. Adopters grep their TOML for the participating kind names.
//
// Drop 4c.6 W0.5.D5 hook.
func validateBlockedByAcyclicity(rules []ChildRule, graphFn func(rules []ChildRule) map[domain.Kind][]domain.Kind) error {
	var graph map[domain.Kind][]domain.Kind
	if graphFn != nil {
		graph = graphFn(rules)
	} else {
		graph = buildBlockedByGraph(rules)
	}
	if len(graph) == 0 {
		return nil
	}
	if cycle, found := dfsDetectCycle(graph); found {
		return fmt.Errorf("%w: %s [blocked_by]", ErrTemplateBlockedByCycle, formatCyclePath(cycle))
	}
	return nil
}

// buildBlockedByGraph produces the production kind-level blocked_by graph
// from a [[child_rules]] slice. Each rule whose BlockedByParent is true
// contributes one edge from rule.CreateChildKind to rule.WhenParentKind
// (child→parent: child cannot start until parent terminal-completes).
//
// Today's schema produces a forest — every edge is a child→parent edge and
// parents have no incoming blocked_by edges, so no cycle can exist. The
// helper is its own function rather than inlined into
// validateBlockedByAcyclicity so future schema expansions (e.g. a
// `BlockedByKinds []domain.Kind` field on ChildRule) extend the helper in
// one place rather than threading new edge-construction logic through the
// validator body.
//
// Drop 4c.6 W0.5.D5 hook.
func buildBlockedByGraph(rules []ChildRule) map[domain.Kind][]domain.Kind {
	graph := make(map[domain.Kind][]domain.Kind)
	for _, rule := range rules {
		if rule.BlockedByParent {
			graph[rule.CreateChildKind] = append(graph[rule.CreateChildKind], rule.WhenParentKind)
		}
	}
	return graph
}

// reachabilityStandaloneKinds is the closed set of kinds that are exempt from
// the validateChildRuleReachability check. These kinds are spawn-by-orchestrator-
// or-template (NOT auto-create-from-plan), so failing the reachability scan for
// any of them is legitimate — adopters who never wire e.g. `commit` into their
// cascade should not have their template rejected for omitting it.
//
// LOUD WARNING TO FUTURE DROPS THAT ADD NEW KINDS: every new kind added to the
// closed 12-value `domain.Kind` enum MUST be classified at addition time —
// either it belongs in `reachabilityStandaloneKinds` (standalone, exempt from
// reachability) OR it MUST appear in some `[[child_rules]]` row of the embedded
// default template (till-go.toml + till-gen.toml ← default-go.toml +
// default-generic.toml, rebadged in Drop 4c.6 W5.D1 + W5.D2) so the
// reachability scan finds it. Failing to do either will surface as a
// load-time ErrUnreachableChildRule against the embedded default and break
// every project that loads it.
//
// Today's standalone set per Drop 4c.5 F.5.2 spec:
//
//   - closeout       — drop-end aggregation, orchestrator-managed.
//   - commit         — template-triggered under `plan` at level >= 2 (Drop 2
//     cadence rule); not auto-created from plan today.
//   - refinement     — perpetual rollup, orchestrator-managed.
//   - discussion     — converged-shape parking, orchestrator-managed.
//   - human-verify   — dev sign-off hold point, orchestrator-managed.
//   - research       — read-only investigation, manually spawned by planners.
var reachabilityStandaloneKinds = []domain.Kind{
	domain.KindCloseout,
	domain.KindCommit,
	domain.KindRefinement,
	domain.KindDiscussion,
	domain.KindHumanVerify,
	domain.KindResearch,
}

// isReachabilityStandaloneKind reports whether k is a member of the closed
// reachabilityStandaloneKinds set. Used by validateChildRuleReachability to
// short-circuit the missing-from-graph check for kinds that are legitimately
// outside the auto-create graph.
func isReachabilityStandaloneKind(k domain.Kind) bool {
	for _, candidate := range reachabilityStandaloneKinds {
		if candidate == k {
			return true
		}
	}
	return false
}

// reachabilityCheckKinds is the closed iteration order over the 12-value
// domain.Kind enum used by validateChildRuleReachability. Hard-coded slice
// (mirrors domain/kind.go's validKinds) so the validator's error UX is
// byte-identical across runs — Go map iteration order is non-deterministic and
// would otherwise produce flapping error messages.
//
// LOUD WARNING TO FUTURE DROPS THAT ADD NEW KINDS: extend this slice with the
// new kind constant in the same drop that introduces it; otherwise the new
// kind will silently bypass the reachability scan and an adopter's typo-stripped
// template will not be caught.
var reachabilityCheckKinds = []domain.Kind{
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

// validateChildRuleReachability asserts that every kind DECLARED in [kinds]
// (and absent from reachabilityStandaloneKinds) appears as either a
// WhenParentKind or a CreateChildKind in at least one [[child_rules]] row.
//
// Algorithm: build the set of kinds "touched" by any [[child_rules]] entry
// (union of every WhenParentKind and every CreateChildKind across all rules).
// Iterate the closed 12-kind enum in declaration order; for each kind that is
// (a) declared in tpl.Kinds, (b) NOT in reachabilityStandaloneKinds, and (c)
// not touched by the rule graph, return ErrUnreachableChildRule wrapping the
// offending kind name. The first offending kind wins — the error surface is
// bounded.
//
// Conditional-on-declaration rationale (F.5.1 mitigation F2 carry-over):
// adopter templates that strip a kind from [kinds] (e.g. a language-agnostic
// template that delegates `kind=plan` to a project-local override) should not
// be rejected here. The validator only enforces the contract for kinds the
// template actually uses. validateRequiredChildRules uses the same
// conditional-on-declaration rule for the QA-twin invariant.
//
// Spec equivalence to "DFS from kind=plan": when every kind that appears as a
// WhenParentKind is treated as a synthetic root (project-creation can directly
// spawn any kind, plus a planner can spawn into any of its declared
// allowed_child_kinds), the reachable set is exactly the union of parent and
// child kinds across all rules. The set-membership form below computes this
// directly without recursion.
//
// Vacuously true on the embedded default per Drop 4c.5 F.5.2 spec Note 1: the
// closed 12-kind enum + the 4 standard child_rules + the 6 standalone-kinds
// classification together cover every member of the enum. The validator's
// real value is for ADOPTER templates that declare a kind in [kinds] but
// forget the corresponding child_rule — typo protection at template Load
// time.
//
// Conditional-on-membership rationale: validateChildRuleKinds (run before
// this validator in the chain) asserts every WhenParentKind / CreateChildKind
// is a member of the closed enum, so reachabilityCheckKinds membership and
// rule-graph membership share the same vocabulary.
func validateChildRuleReachability(tpl Template) error {
	touched := make(map[domain.Kind]struct{}, len(tpl.ChildRules)*2)
	for _, rule := range tpl.ChildRules {
		touched[rule.WhenParentKind] = struct{}{}
		touched[rule.CreateChildKind] = struct{}{}
	}
	for _, kind := range reachabilityCheckKinds {
		if isReachabilityStandaloneKind(kind) {
			continue
		}
		// Skip kinds that are not declared in [kinds] — adopter templates
		// may legitimately strip vocabulary entries (per F.5.1 mitigation
		// F2). validateChildRuleKinds (run before this validator) asserts
		// every WhenParentKind / CreateChildKind reference is a member of
		// the closed enum, so a [[child_rules]] entry for a stripped kind
		// is structurally legal but vacuous.
		if _, declared := tpl.Kinds[kind]; !declared {
			continue
		}
		if _, ok := touched[kind]; !ok {
			return fmt.Errorf("%w: kind %q is declared in [kinds] but neither standalone nor referenced by any [[child_rules]] entry", ErrUnreachableChildRule, kind)
		}
	}
	return nil
}

// validateKindStructuralCoherence asserts a thin cross-axis invariant between
// the [kinds.X] structural_type axis and the [[child_rules]] auto-create axis:
// any kind declared with `structural_type = "drop"` MUST have at least one
// [[child_rules]] entry where `when_parent_kind == X`. A "drop" structural type
// names a kind that decomposes into cascade work; a drop kind with no
// child-rule entries pointing at it is structurally orphaned and represents a
// template-author mistake.
//
// The check is conditional on the kind's structural_type being EXACTLY "drop"
// (the closed StructuralType enum has four values: drop / segment / confluence
// / droplet — see WIKI.md §"Cascade Vocabulary"). Other structural types do
// NOT trigger this check:
//
//   - droplet: terminal node, no decomposition expected.
//   - segment: may recurse but coherence shape is handled by the kind's own
//     auto-create chain in a future drop.
//   - confluence: defined by `blocked_by` non-empty, not by child_rules.
//
// Drop 4c.5 F.5.2 ships only the drop-coherence wedge; full structural-type ↔
// kind ↔ role coherence is post-MVP. The embedded till-go.toml (rebadged
// from default-go.toml in Drop 4c.6 W5.D1) uses `structural_type = "droplet"`
// for every kind today (per Drop 3 Note 1 in THEME_F_PLAN.md), so this
// validator is a no-op against the default. It only fires on adopter
// templates that opt into structural_type=drop.
//
// The validator returns on the FIRST offending kind to keep the error surface
// bounded; outer-map iteration order over tpl.Kinds is non-deterministic in
// Go but the closed-enum check below is order-independent because exactly
// zero or one kind will fail on any given template.
//
// All non-nil returns wrap ErrIncoherentStructuralType so callers using
// `errors.Is(err, ErrIncoherentStructuralType)` route correctly.
func validateKindStructuralCoherence(tpl Template) error {
	// Index existing [[child_rules]] entries by parent kind for O(1) lookup
	// of the "any rule with when_parent_kind == X" presence test.
	parentsWithRules := make(map[domain.Kind]struct{}, len(tpl.ChildRules))
	for _, rule := range tpl.ChildRules {
		parentsWithRules[rule.WhenParentKind] = struct{}{}
	}
	for kind, row := range tpl.Kinds {
		if row.StructuralType != domain.StructuralTypeDrop {
			continue
		}
		if _, ok := parentsWithRules[kind]; !ok {
			return fmt.Errorf("%w: kind %q has structural_type=%q but no [[child_rules]] entry has when_parent_kind=%q (drop kinds must decompose)",
				ErrIncoherentStructuralType, kind, domain.StructuralTypeDrop, kind)
		}
	}
	return nil
}

// validateGateKinds asserts every gate-kind string in Template.Gates value
// slices is a member of the closed GateKind enum (templates.IsValidGateKind).
// The map-key axis (parent kind) is validated by validateMapKeys above —
// validateGateKinds focuses exclusively on the value-slice axis.
//
// Drop 4b Wave A 4b.1 hook: invoked from Load after
// validateChildRuleReachability so the gate vocabulary check fires on a
// template whose kind/child-rule axes already passed. Drop 4c will extend
// IsValidGateKind to accept "commit" and "push"; this validator's body is
// agnostic to the closed-enum size — it delegates entirely to
// IsValidGateKind.
func validateGateKinds(tpl Template) error {
	for parentKind, gateSeq := range tpl.Gates {
		for _, g := range gateSeq {
			if !IsValidGateKind(g) {
				return fmt.Errorf("%w: gates[%q] entry %q", ErrUnknownGateKind, parentKind, g)
			}
		}
	}
	return nil
}

// envVarNameRegex pins the closed env-var name pattern per Drop 4c F.7.17
// locked decision L5 (formerly L5 in the F.7.17 sub-plan). The pattern is the
// literal `^[A-Za-z][A-Za-z0-9_]*$` per A1.c — explicit anchors inside the
// pattern AND the call uses MatchString against a compiled regex so the
// "regex without anchors" footgun documented in falsification round 2 is
// statically impossible.
//
// Both uppercase and lowercase leading letters are allowed: corporate-network
// adopters routinely set `https_proxy` (lowercase, the conventional cURL
// spelling) AND `HTTPS_PROXY` (uppercase, the conventional Go net/http
// spelling) on the same machine, and Tillsyn's per-binding env declarations
// must be able to forward either form without the dev rewriting their shell.
var envVarNameRegex = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_]*$`)

// validateAgentBindingEnvNames asserts every entry in every AgentBinding.Env
// slice satisfies the closed env-var-name contract before the dispatcher's
// CLI adapter resolves it via os.Getenv at spawn time.
//
// Per Drop 4c F.7.17.1 acceptance contract each entry MUST:
//
//   - Be non-empty.
//   - Contain no `=` character (a TOML editor writing `KEY=value` instead of
//     just `KEY` is the most common authoring footgun and merits a precise,
//     ahead-of-regex error message naming the offending entry verbatim).
//   - Match `^[A-Za-z][A-Za-z0-9_]*$` — leading letter, trailing
//     alphanumerics + underscore. Whitespace, hyphens, dots, and leading
//     digits are rejected.
//   - Be unique within its binding's env list (case-sensitive comparison;
//     `FOO` and `foo` are distinct entries because POSIX env-var names are
//     case-sensitive).
//
// The validator iterates `tpl.AgentBindings` deterministically by stable
// field order inside each binding's slice, but the outer map iteration order
// is not deterministic — the function returns on the FIRST offending entry
// to keep the error surface bounded. Future drops that want exhaustive
// reporting can switch to error aggregation; the closed-enum + load-time-
// reject pattern doesn't need it today.
//
// All non-nil returns wrap ErrInvalidAgentBindingEnv (which itself wraps
// ErrInvalidAgentBinding) so callers using `errors.Is(err, ErrInvalidAgentBinding)`
// route correctly without reaching for the env-specific sentinel.
func validateAgentBindingEnvNames(tpl Template) error {
	for kind, binding := range tpl.AgentBindings {
		seen := make(map[string]struct{}, len(binding.Env))
		for _, entry := range binding.Env {
			if entry == "" {
				return fmt.Errorf("%w: agent_bindings[%q].env entry is empty", ErrInvalidAgentBindingEnv, kind)
			}
			if strings.Contains(entry, "=") {
				return fmt.Errorf("%w: agent_bindings[%q].env entry %q contains '='; declare names only (values resolve via os.Getenv)", ErrInvalidAgentBindingEnv, kind, entry)
			}
			if !envVarNameRegex.MatchString(entry) {
				return fmt.Errorf("%w: agent_bindings[%q].env entry %q does not match %s", ErrInvalidAgentBindingEnv, kind, entry, envVarNameRegex.String())
			}
			if _, dup := seen[entry]; dup {
				return fmt.Errorf("%w: agent_bindings[%q].env entry %q is duplicated", ErrInvalidAgentBindingEnv, kind, entry)
			}
			seen[entry] = struct{}{}
		}
	}
	return nil
}

// validContextDeliveryValues lists the closed-enum values accepted by
// ContextRules.Delivery. The empty string is included because omission is
// legal at the schema layer — the F.7.18.3 aggregator engine substitutes
// ContextDeliveryFile at Resolve-time per master PLAN.md L13's "consumer-time
// default" framing.
var validContextDeliveryValues = []string{
	"",
	ContextDeliveryInline,
	ContextDeliveryFile,
}

// validateAgentBindingContext asserts every AgentBinding.Context sub-struct
// satisfies the Drop 4c F.7.18.1 closed contract:
//
//   - Delivery is one of {"", "inline", "file"} — closed enum. Empty string
//     resolves to "file" at engine-time (NOT at validation-time).
//   - MaxChars is non-negative. Zero is legal and means "use bundle-global
//     default at engine-time" (F.7.18.3 substitutes 50000).
//   - MaxRuleDuration is non-negative. Zero is legal and means "use
//     bundle-global default at engine-time" (F.7.18.4 substitutes 500ms).
//   - Every kind in SiblingsByKind / AncestorsByKind / DescendantsByKind is a
//     member of the closed 12-value domain.Kind enum. Empty slices are
//     legal.
//
// NO schema rule against `descendants_by_kind` on `kind=plan`. Per master
// PLAN.md L13's flexibility framing the schema trusts template authors;
// fix-planners + tree-pruners are legitimate uses for a planner that walks
// down. This is enforced by an explicit allow-test in load_test.go.
//
// All non-nil returns wrap ErrInvalidContextRules (which itself wraps
// ErrInvalidAgentBinding) so callers using
// `errors.Is(err, ErrInvalidAgentBinding)` route correctly without reaching
// for the context-specific sentinel. Kind-reference failures wrap
// ErrUnknownKindReference for consistency with the existing kinds-map /
// child-rules / agent-bindings-map vocabulary checks.
//
// The validator returns on the FIRST offending field to keep the error
// surface bounded; outer-map iteration order is non-deterministic but the
// inner per-binding checks run in a stable field order
// (Delivery → MaxChars → MaxRuleDuration → SiblingsByKind → AncestorsByKind
// → DescendantsByKind).
func validateAgentBindingContext(tpl Template) error {
	for kind, binding := range tpl.AgentBindings {
		ctx := binding.Context
		if !isValidContextDelivery(ctx.Delivery) {
			return fmt.Errorf("%w: agent_bindings[%q].context.delivery %q not in {%q, %q, %q}",
				ErrInvalidContextRules, kind, ctx.Delivery, "", ContextDeliveryInline, ContextDeliveryFile)
		}
		if ctx.MaxChars < 0 {
			return fmt.Errorf("%w: agent_bindings[%q].context.max_chars must be >= 0 (got %d)",
				ErrInvalidContextRules, kind, ctx.MaxChars)
		}
		if time.Duration(ctx.MaxRuleDuration) < 0 {
			return fmt.Errorf("%w: agent_bindings[%q].context.max_rule_duration must be >= 0 (got %s)",
				ErrInvalidContextRules, kind, time.Duration(ctx.MaxRuleDuration))
		}
		if err := validateContextKindList(kind, "siblings_by_kind", ctx.SiblingsByKind); err != nil {
			return err
		}
		if err := validateContextKindList(kind, "ancestors_by_kind", ctx.AncestorsByKind); err != nil {
			return err
		}
		if err := validateContextKindList(kind, "descendants_by_kind", ctx.DescendantsByKind); err != nil {
			return err
		}
	}
	return nil
}

// isValidContextDelivery reports whether v is a member of the closed
// {"", "inline", "file"} delivery-vocabulary. Exact-match — no whitespace
// trimming or case folding, mirroring the IsValidGateKind rationale (silent
// case-fold matching would mask "Inline" / "FILE" typos at load time).
func isValidContextDelivery(v string) bool {
	for _, candidate := range validContextDeliveryValues {
		if v == candidate {
			return true
		}
	}
	return false
}

// validateContextKindList asserts every entry in a kind-walk slice is a
// member of the closed 12-value domain.Kind enum. fieldName is the TOML key
// name (e.g. "siblings_by_kind") used in the error UX so adopters see the
// exact line they need to fix.
func validateContextKindList(kind domain.Kind, fieldName string, kinds []domain.Kind) error {
	for _, k := range kinds {
		if !domain.IsValidKind(k) {
			return fmt.Errorf("%w: agent_bindings[%q].context.%s entry %q",
				ErrUnknownKindReference, kind, fieldName, k)
		}
	}
	return nil
}

// systemPromptShellMetacharRunes pins the closed set of shell metacharacters
// rejected inside SystemPromptTemplatePath at template Load time. The set is
// deliberately conservative — defense-in-depth against future render-layer
// refactors — and matches the most-likely command-injection vectors a
// malicious template author could embed in a path string. The dispatcher
// render layer (F.7.3b) never invokes a shell against this path, but
// rejecting at the schema layer keeps the resolved-path safe.
var systemPromptShellMetacharRunes = []rune{';', '|', '&', '`', '$'}

// validateAgentBindingToolGating asserts every AgentBinding's tool-gating /
// system-prompt-template / sandbox fields satisfy the Drop 4c F.7.2 closed
// contract. Validates four logical groups in stable order so the error
// surface for any binding is deterministic:
//
//  1. ToolsAllowed / ToolsDisallowed — entries non-empty + within-binding
//     unique. Tool-name vocabulary is open-ended (Read / Edit / Bash(mage *) /
//     WebFetch / etc.); no closed-enum check.
//  2. SystemPromptTemplatePath — when non-empty, a project-relative path
//     under `.tillsyn/`. Absolute paths, `..` traversal, and shell
//     metacharacters `;` `|` `&` backtick `$` are rejected.
//  3. Sandbox.Filesystem.AllowWrite / DenyRead — each entry must be a clean
//     absolute path (begins with `/`, no `..` segments, no double-slashes).
//  4. Sandbox.Network.AllowedDomains / DeniedDomains — each entry must be a
//     non-empty string with no URL scheme prefix (`http://`, `https://`).
//     A leading `*` glob is permitted (e.g. `*.npmjs.org`).
//
// All non-nil returns wrap ErrInvalidAgentBindingToolGating (which itself
// wraps ErrInvalidAgentBinding) so callers using
// `errors.Is(err, ErrInvalidAgentBinding)` route correctly without reaching
// for the tool-gating-specific sentinel.
//
// Outer-map iteration order is non-deterministic; the validator returns on
// the FIRST offending field per binding to keep the error surface bounded.
// Future drops that want exhaustive reporting can switch to error
// aggregation; the closed-enum + load-time-reject pattern doesn't need it
// today.
func validateAgentBindingToolGating(tpl Template) error {
	for kind, binding := range tpl.AgentBindings {
		if err := validateToolNameList(kind, "tools_allowed", binding.ToolsAllowed); err != nil {
			return err
		}
		if err := validateToolNameList(kind, "tools_disallowed", binding.ToolsDisallowed); err != nil {
			return err
		}
		if err := validateSystemPromptTemplatePath(kind, binding.SystemPromptTemplatePath); err != nil {
			return err
		}
		if err := validateSandboxAbsolutePathList(kind, "sandbox.filesystem.allow_write", binding.Sandbox.Filesystem.AllowWrite); err != nil {
			return err
		}
		if err := validateSandboxAbsolutePathList(kind, "sandbox.filesystem.deny_read", binding.Sandbox.Filesystem.DenyRead); err != nil {
			return err
		}
		if err := validateSandboxDomainList(kind, "sandbox.network.allowed_domains", binding.Sandbox.Network.AllowedDomains); err != nil {
			return err
		}
		if err := validateSandboxDomainList(kind, "sandbox.network.denied_domains", binding.Sandbox.Network.DeniedDomains); err != nil {
			return err
		}
	}
	return nil
}

// validateToolNameList enforces the entries-non-empty + within-list-unique
// contract on a tool-gating slice (ToolsAllowed or ToolsDisallowed).
// fieldName is the TOML key name (e.g. "tools_allowed") used in the error
// UX so adopters see the exact line they need to fix. Tool-name vocabulary
// is open-ended — no closed-enum check is applied.
func validateToolNameList(kind domain.Kind, fieldName string, entries []string) error {
	seen := make(map[string]struct{}, len(entries))
	for _, entry := range entries {
		if entry == "" {
			return fmt.Errorf("%w: agent_bindings[%q].%s entry is empty",
				ErrInvalidAgentBindingToolGating, kind, fieldName)
		}
		if _, dup := seen[entry]; dup {
			return fmt.Errorf("%w: agent_bindings[%q].%s entry %q is duplicated",
				ErrInvalidAgentBindingToolGating, kind, fieldName, entry)
		}
		seen[entry] = struct{}{}
	}
	return nil
}

// validateSystemPromptTemplatePath asserts a non-empty path is project-
// relative, traversal-free, and shell-metachar-free. Empty paths are legal —
// the render layer (F.7.3b) substitutes the per-kind built-in template. The
// path is NOT opened or stat'd here: validation is purely syntactic so a
// template referencing a not-yet-materialized resource still loads cleanly.
func validateSystemPromptTemplatePath(kind domain.Kind, path string) error {
	if path == "" {
		return nil
	}
	if strings.HasPrefix(path, "/") {
		return fmt.Errorf("%w: agent_bindings[%q].system_prompt_template_path %q is absolute; must be relative to .tillsyn/",
			ErrInvalidAgentBindingToolGating, kind, path)
	}
	if pathContainsTraversal(path) {
		return fmt.Errorf("%w: agent_bindings[%q].system_prompt_template_path %q contains '..' traversal segment",
			ErrInvalidAgentBindingToolGating, kind, path)
	}
	for _, r := range systemPromptShellMetacharRunes {
		if strings.ContainsRune(path, r) {
			return fmt.Errorf("%w: agent_bindings[%q].system_prompt_template_path %q contains shell metacharacter %q",
				ErrInvalidAgentBindingToolGating, kind, path, string(r))
		}
	}
	return nil
}

// validateSandboxAbsolutePathList enforces the non-empty + clean-absolute-
// path contract on a sandbox filesystem slice (AllowWrite or DenyRead).
// "Clean" means: starts with `/`, contains no `..` segment, contains no
// double-slashes (`//`). The check is syntactic — the path is not opened
// or stat'd, mirroring SystemPromptTemplatePath's syntactic validator.
func validateSandboxAbsolutePathList(kind domain.Kind, fieldName string, entries []string) error {
	for _, entry := range entries {
		if entry == "" {
			return fmt.Errorf("%w: agent_bindings[%q].%s entry is empty",
				ErrInvalidAgentBindingToolGating, kind, fieldName)
		}
		if !strings.HasPrefix(entry, "/") {
			return fmt.Errorf("%w: agent_bindings[%q].%s entry %q must be an absolute path (starts with '/')",
				ErrInvalidAgentBindingToolGating, kind, fieldName, entry)
		}
		if pathContainsTraversal(entry) {
			return fmt.Errorf("%w: agent_bindings[%q].%s entry %q contains '..' traversal segment",
				ErrInvalidAgentBindingToolGating, kind, fieldName, entry)
		}
		if strings.Contains(entry, "//") {
			return fmt.Errorf("%w: agent_bindings[%q].%s entry %q contains '//' (path is not clean)",
				ErrInvalidAgentBindingToolGating, kind, fieldName, entry)
		}
	}
	return nil
}

// validateSandboxDomainList enforces the non-empty + no-URL-scheme contract
// on a sandbox network slice (AllowedDomains or DeniedDomains). A leading
// `*` glob is permitted (e.g. `*.npmjs.org`). Schemes other than `http://`
// and `https://` are not enumerated — the canonical command-injection
// surface is HTTP / HTTPS, and template authors writing custom schemes
// trigger the same generic "looks like a URL" guard via the `://`
// substring check.
func validateSandboxDomainList(kind domain.Kind, fieldName string, entries []string) error {
	for _, entry := range entries {
		if entry == "" {
			return fmt.Errorf("%w: agent_bindings[%q].%s entry is empty",
				ErrInvalidAgentBindingToolGating, kind, fieldName)
		}
		if strings.Contains(entry, "://") {
			return fmt.Errorf("%w: agent_bindings[%q].%s entry %q contains URL scheme '://'; declare host only (e.g. 'github.com', '*.npmjs.org')",
				ErrInvalidAgentBindingToolGating, kind, fieldName, entry)
		}
	}
	return nil
}

// pathContainsTraversal reports whether path has any `..` path segment.
// Splits on `/` so substrings like `foo..bar` (a literal filename containing
// two dots) do not trip the check — only a true traversal segment qualifies.
func pathContainsTraversal(path string) bool {
	for _, segment := range strings.Split(path, "/") {
		if segment == ".." {
			return true
		}
	}
	return false
}

// validTillsynSpawnTempRootValues lists the closed-enum values accepted by
// Tillsyn.SpawnTempRoot. The empty string is included because omission is
// legal at the schema layer — the F.7.1 NewBundle materializer substitutes
// "os_tmp" at spawn time per the consumer-time-default convention used
// elsewhere in the schema (e.g. ContextRules.Delivery).
var validTillsynSpawnTempRootValues = []string{
	"",
	"os_tmp",
	"project",
}

// isValidTillsynSpawnTempRoot reports whether v is a member of the closed
// {"", "os_tmp", "project"} spawn-temp-root vocabulary. Exact match — no
// whitespace trimming or case folding. Mirrors the IsValidGateKind /
// isValidContextDelivery rationale (silent case-fold matching would mask
// "OS_TMP" / "Project" typos at load time).
func isValidTillsynSpawnTempRoot(v string) bool {
	for _, candidate := range validTillsynSpawnTempRootValues {
		if v == candidate {
			return true
		}
	}
	return false
}

// validateTillsyn asserts the top-level [tillsyn] table satisfies the Drop 4c
// F.7.18.2 + F.7-CORE F.7.1 + F.7-CORE F.7.6 closed contract:
//
//   - MaxContextBundleChars is non-negative. Zero is legal and means "use
//     bundle-global default at engine-time" (F.7.18.4 substitutes 200000 per
//     master PLAN L14).
//   - MaxAggregatorDuration is non-negative. Zero is legal and means "use
//     bundle-global default at engine-time" (F.7.18.4 substitutes 2s per
//     master PLAN L15).
//   - SpawnTempRoot is one of {"", "os_tmp", "project"} — closed enum. Empty
//     string resolves to "os_tmp" at engine-time (NOT at validation-time)
//     per Drop 4c F.7.1 NewBundle materializer.
//   - RequiresPlugins entries each match `<name>` OR `<name>@<marketplace>`
//     where each segment is non-empty, contains no whitespace, and the
//     entry contains at most one `@`. Within-list duplicates are rejected.
//     Empty slice is legal and means "no required plugins" — the pre-flight
//     check (Drop 4c F.7.6 CheckRequiredPlugins) returns nil immediately.
//
// All non-nil returns wrap ErrInvalidTillsynGlobals so callers can route on
// the sentinel via errors.Is. The validator runs after
// validateAgentBindingContext so per-binding failures surface with their
// original sentinel rather than being masked by a global rule.
//
// Per REV-3 the Tillsyn struct ships with two fields in F.7.18.2; F.7-CORE
// F.7.1 extends with SpawnTempRoot; F.7-CORE F.7.6 extends with
// RequiresPlugins. Strict-decode unknown-key rejection on the Tillsyn struct
// is inherited automatically from load.go step 3 (DisallowUnknownFields),
// so future extenders do not need to reshape this validator — they add
// their own field-level checks alongside the existing ones.
func validateTillsyn(tpl Template) error {
	if tpl.Tillsyn.MaxContextBundleChars < 0 {
		return fmt.Errorf("%w: max_context_bundle_chars must be >= 0 (got %d)",
			ErrInvalidTillsynGlobals, tpl.Tillsyn.MaxContextBundleChars)
	}
	if time.Duration(tpl.Tillsyn.MaxAggregatorDuration) < 0 {
		return fmt.Errorf("%w: max_aggregator_duration must be >= 0 (got %s)",
			ErrInvalidTillsynGlobals, time.Duration(tpl.Tillsyn.MaxAggregatorDuration))
	}
	if !isValidTillsynSpawnTempRoot(tpl.Tillsyn.SpawnTempRoot) {
		return fmt.Errorf("%w: spawn_temp_root %q not in {%q, %q, %q}",
			ErrInvalidTillsynGlobals, tpl.Tillsyn.SpawnTempRoot, "", "os_tmp", "project")
	}
	if err := validateTillsynRequiresPlugins(tpl.Tillsyn.RequiresPlugins); err != nil {
		return err
	}
	return nil
}

// validateTillsynRequiresPlugins enforces the Drop 4c F.7-CORE F.7.6 entry
// contract on each RequiresPlugins slice entry:
//
//   - Non-empty.
//   - Contains no ASCII whitespace (space, tab, CR, LF). Plugin identifiers
//     are single-token names; whitespace inside an entry is always a
//     template-author error.
//   - Contains at most one `@` separator. Two valid shapes are accepted:
//     `<name>` (bare, marketplace-implicit) and `<name>@<marketplace>`
//     (marketplace-scoped). A second `@` would yield ambiguous parsing in
//     the pre-flight matcher.
//   - When `<name>@<marketplace>` shape is used, BOTH segments must be
//     non-empty (`@marketplace` and `name@` are both rejected).
//   - Within-list duplicates are rejected. Case-sensitive comparison —
//     plugin identifiers in Claude's plugin catalog are case-sensitive and
//     silent fold-matching would mask "Context7" / "context7" typos at load
//     time.
//
// Returns the first offending entry; the error wraps ErrInvalidTillsynGlobals
// so callers using `errors.Is(err, ErrInvalidTillsynGlobals)` route correctly
// without reaching for a separate sentinel.
func validateTillsynRequiresPlugins(entries []string) error {
	seen := make(map[string]struct{}, len(entries))
	for _, entry := range entries {
		if entry == "" {
			return fmt.Errorf("%w: requires_plugins entry is empty",
				ErrInvalidTillsynGlobals)
		}
		if strings.ContainsAny(entry, " \t\r\n") {
			return fmt.Errorf("%w: requires_plugins entry %q contains whitespace",
				ErrInvalidTillsynGlobals, entry)
		}
		if strings.Count(entry, "@") > 1 {
			return fmt.Errorf("%w: requires_plugins entry %q contains more than one '@'; expected `<name>` or `<name>@<marketplace>`",
				ErrInvalidTillsynGlobals, entry)
		}
		if at := strings.IndexByte(entry, '@'); at >= 0 {
			name, marketplace := entry[:at], entry[at+1:]
			if name == "" {
				return fmt.Errorf("%w: requires_plugins entry %q has empty name before '@'",
					ErrInvalidTillsynGlobals, entry)
			}
			if marketplace == "" {
				return fmt.Errorf("%w: requires_plugins entry %q has empty marketplace after '@'",
					ErrInvalidTillsynGlobals, entry)
			}
		}
		if _, dup := seen[entry]; dup {
			return fmt.Errorf("%w: requires_plugins entry %q is duplicated",
				ErrInvalidTillsynGlobals, entry)
		}
		seen[entry] = struct{}{}
	}
	return nil
}

// requiredChildRulesByParent encodes the closed REQUIRED-CHILD-RULES set
// validated by validateRequiredChildRules. Drop 4c.5 F.5.1 ships this as a
// hard-coded map (NOT a template-level config) because the QA-twins-on-plan
// and QA-twins-on-build invariant is part of the cascade contract itself —
// adopters who skip these twins do not have a working cascade.
//
// Future cascade extensions (e.g. design-twins-on-plan) extend this map; the
// validator is parametric over the map's contents so adding a new required
// pair is a one-line change here. The set is keyed by parent kind for O(1)
// presence-check during validation.
var requiredChildRulesByParent = map[domain.Kind][]domain.Kind{
	domain.KindPlan:  {domain.KindPlanQAProof, domain.KindPlanQAFalsification},
	domain.KindBuild: {domain.KindBuildQAProof, domain.KindBuildQAFalsification},
}

// validateRequiredChildRules asserts that every parent kind in the closed
// REQUIRED-CHILD-RULES set (requiredChildRulesByParent) — when DECLARED in
// [kinds] — has a [[child_rules]] entry materializing each of its required
// QA-twin children.
//
// Conditional-on-presence rationale (Drop 4c.5 F.5.1 falsification mitigation
// F2): adopter templates that strip `kind=plan` or `kind=build` entirely
// (e.g. an extreme language-agnostic template that delegates all kind
// declarations to a project-local override) should not be rejected here. The
// validator only enforces the contract for parents that are declared.
//
// The check iterates parent kinds in stable map order (returns on the first
// missing rule per parent + scans parents in the requiredChildRulesByParent
// declaration order via a deterministic key list) so the error UX is
// reproducible across runs.
//
// Returns ErrMissingRequiredChildRule wrapping a message that names the
// parent kind and the missing child kind verbatim so adopters see the exact
// rule they need to add.
func validateRequiredChildRules(tpl Template) error {
	// Index existing [[child_rules]] entries by parent kind for O(1) lookup
	// of the (parent, child) presence test below.
	rulesByParent := make(map[domain.Kind]map[domain.Kind]struct{}, len(tpl.ChildRules))
	for _, rule := range tpl.ChildRules {
		set, ok := rulesByParent[rule.WhenParentKind]
		if !ok {
			set = make(map[domain.Kind]struct{})
			rulesByParent[rule.WhenParentKind] = set
		}
		set[rule.CreateChildKind] = struct{}{}
	}

	// Stable parent-kind iteration order. Hard-coded slice rather than
	// sorting requiredChildRulesByParent's keys at runtime so the error UX
	// is byte-identical across Go map iteration shuffling.
	for _, parent := range []domain.Kind{domain.KindPlan, domain.KindBuild} {
		required, ok := requiredChildRulesByParent[parent]
		if !ok {
			continue
		}
		// Skip parents that are not declared in [kinds] — adopter
		// templates may legitimately strip plan or build (per F.5.1
		// falsification mitigation F2). validateChildRuleKinds already
		// asserts every WhenParentKind / CreateChildKind reference is a
		// member of the closed enum, so a [[child_rules]] entry for a
		// stripped parent is structurally legal but vacuous.
		if _, declared := tpl.Kinds[parent]; !declared {
			continue
		}
		existing := rulesByParent[parent] // nil-safe; map access on zero-value returns zero-value of V
		for _, child := range required {
			if _, ok := existing[child]; !ok {
				return fmt.Errorf("%w: parent %q must have a [[child_rules]] entry creating %q",
					ErrMissingRequiredChildRule, parent, child)
			}
		}
	}
	return nil
}

// claudeAgentsDirEnvVar names an optional environment variable that overrides
// the default `~/.claude/agents/` lookup directory used by
// validateAgentBindingFiles. The override exists for adopters whose Claude
// Code install lives outside the conventional home-directory layout — e.g.
// containerized CI runners that mount agents at /workspace/agents. When the
// var is unset (the default) the validator resolves
// `${HOME}/.claude/agents/<name>.md`.
const claudeAgentsDirEnvVar = "TILLSYN_CLAUDE_AGENTS_DIR"

// resolveClaudeAgentsDir returns the absolute path to the directory containing
// per-agent markdown files referenced by AgentBinding.AgentName. When
// claudeAgentsDirEnvVar is set, that value wins verbatim. Otherwise the
// function joins the OS-reported home directory with `.claude/agents`.
//
// Returns ("", err) when neither the env var nor os.UserHomeDir resolves —
// validateAgentBindingFiles treats this as "cannot warn deterministically;
// drop the warning silently" rather than escalating, matching F.5.1's
// warn-only contract.
func resolveClaudeAgentsDir() (string, error) {
	if dir := os.Getenv(claudeAgentsDirEnvVar); dir != "" {
		return dir, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home directory: %w", err)
	}
	return filepath.Join(home, ".claude", "agents"), nil
}

// defaultAgentBindingStatFn is the production existence-check used by
// validateAgentBindingFiles when LoadOptions.StatFn is nil. Returns true when
// os.Stat reports the file present, false otherwise (including not-found,
// permission errors, and any other os.Stat failure mode). The function
// deliberately collapses every failure mode to false because the warning is
// purely informational — distinguishing "ENOENT" from "EACCES" inside a
// validator that drops the result on the floor would be UX noise.
func defaultAgentBindingStatFn(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// validateAgentBindingFiles emits one warning per AgentBinding.AgentName whose
// corresponding `~/.claude/agents/<name>.md` file is absent at validation time.
// Drop 4c.5 F.5.1 hook.
//
// Warn-only per Drop 4c.5 Q2 resolution (workflow/drop_4c_5/THEME_F_PLAN.md
// §3 Note 6): dev-machine state is NOT template-correctness. A template that
// references `go-builder-agent` is no less valid because the dev's machine
// has not yet materialized the agent file — and forcing strict failure here
// would block CI builds on machines without the developer agent layout.
// Adopters who want strict-fail wrap their LoadOptions.WarnLogger with an
// escalation closure at the call site.
//
// Behavior contract:
//
//   - logger == nil → warnings drop silently (preserves Load(io.Reader)
//     pre-F.5.1 silence-by-default).
//   - statFn == nil → defaults to os.Stat via defaultAgentBindingStatFn.
//   - resolveClaudeAgentsDir() failure → warnings drop silently (cannot
//     issue a deterministic path-shaped warning without a base dir).
//   - For each binding whose AgentName resolves to a missing file, emit one
//     line shaped: `agent_bindings[<kind>]: agent_name="<name>" referenced
//     by template but ~/.claude/agents/<name>.md not found at <abs-path>`.
//
// The function never returns an error. Outer-map iteration order is
// non-deterministic (Go map iteration); tests that assert warning lists
// must sort the captured slice before comparing.
func validateAgentBindingFiles(tpl Template, logger func(string), statFn func(string) bool) {
	if logger == nil {
		return
	}
	if statFn == nil {
		statFn = defaultAgentBindingStatFn
	}
	dir, err := resolveClaudeAgentsDir()
	if err != nil {
		// Cannot resolve a base dir — drop the warning rather than emit a
		// half-shaped message. Matches F.5.1's warn-only floor: never
		// surface filesystem-shaped problems at template-load time.
		return
	}
	for kind, binding := range tpl.AgentBindings {
		name := binding.AgentName
		if name == "" {
			// Empty agent_name is rejected upstream by AgentBinding.Validate
			// (ErrInvalidAgentBinding); reaching here means a programmer
			// invariant broke. Skip rather than emit a malformed warning.
			continue
		}
		path := filepath.Join(dir, name+".md")
		if statFn(path) {
			continue
		}
		logger(fmt.Sprintf("agent_bindings[%q]: agent_name=%q referenced by template but %s not found",
			kind, name, path))
	}
}

// embeddedAgentGroups names the closed set of embedded agent-library groups
// the default AgentLookupFn walks. Per SKETCH.md §3.4 the 3-tier resolver's
// embedded floor unions across these three groups; an agent_name resolves at
// the floor if its `<name>.md` file exists in ANY group's directory.
//
// LOUD WARNING TO FUTURE DROPS THAT ADD NEW EMBEDDED GROUPS: extend this
// slice in the same drop that ships the new group's `builtin/agents/<group>/`
// directory. Failing to do so will silently bypass the new group from the
// resolver floor — adopters' agent_name references that resolve only against
// the new group will appear as ErrUnknownAgentName at Load.
//
// Pre-W1.D1 the `internal/templates/builtin/agents/` directory does not yet
// exist. The default walker handles the missing-directory case gracefully via
// embed.FS.Open returning an error, which the walker translates to "not
// found" without panicking.
var embeddedAgentGroups = []string{"till-gen", "till-go", "till-gdd"}

// embeddedAgentLibraryShipped reports whether DefaultTemplateFS contains at
// least one agent .md file under any of the embedded groups
// (`builtin/agents/{till-gen,till-go,till-gdd}/`). Probed once at package
// init via fs.ReadDir against the embed.FS so the default walker can
// distinguish "library has shipped, walk strictly" from "library has not
// yet shipped (pre-W1.D1), walk permissively."
//
// Per W0.5 plan FF2: D2's validator code ships before W1.D1 lights up the
// embedded library. Pre-W1.D1 the FS contains no agent .md files at the
// embedded paths AND the //go:embed directive in embed.go does not even
// list the `builtin/agents/` subtree — every default walker call would
// return false, which would break every existing test that loads the
// embedded default templates (at that historical point in time
// default-go.toml — rebadged to till-go.toml in Drop 4c.6 W5.D1 — referenced
// "go-builder-agent", "go-planning-agent", etc. — none of which resolved
// pre-W1.D1. Drop 4c.6 W5.D3 dropped the `go-` prefix from agent_name
// values; current names are bare `builder-agent`, `planning-agent`, etc.).
//
// The reconciliation: the default walker fails-permissive when the
// embedded library has not yet shipped (no agent .md files probed), and
// fails-strict once W1.D1 lands the placeholder files. Production callers
// transition from "validator is structurally wired but vacuously passes"
// to "validator hard-fails on dangling names" with no D2 code change —
// the trigger is the embed.FS contents, exactly as the W0.5 plan FF2
// disclosure pins.
//
// Tests that want strict behaviour pre-W1.D1 inject an explicit
// LoadOptions.AgentLookupFn — this bypasses the default and exercises the
// hard-fail path the validator's body implements.
var embeddedAgentLibraryShipped = func() bool {
	for _, group := range embeddedAgentGroups {
		entries, err := DefaultTemplateFS.ReadDir("builtin/agents/" + group)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".md") {
				return true
			}
		}
	}
	return false
}()

// defaultAgentLookupFn is the production existence-check used by
// validateAgentBindingNames when LoadOptions.AgentLookupFn is nil. Walks the
// embedded `builtin/agents/{till-gen,till-go,till-gdd}/<name>.md` paths in
// DefaultTemplateFS and returns true on the first hit, false otherwise.
//
// Per W0.5 plan FF2 disclosure: this default walker exercises embed.FS
// UNCONDITIONALLY — it does not depend on any post-W0.5.D2 rewire. Pre-W1.D1
// the FS contains no agent .md files at the embedded paths AND the embed
// directive does not even cover `builtin/agents/` yet, so the function
// fails-permissive (returns true for every name) per the
// `embeddedAgentLibraryShipped` probe at package init. Tests inject
// synthetic LoadOptions.AgentLookupFn values to exercise the validator's
// hard-fail path pre-W1.D1. Post-W1.D1 the same default walker switches to
// strict mode automatically because the probe sees real placeholder files;
// the validator code is final on D2 land.
//
// Path layout: `builtin/agents/<group>/<name>.md`. The walker strips no
// extension or prefix — `<name>` is the literal AgentBinding.AgentName as
// authored.
func defaultAgentLookupFn(name string) bool {
	if name == "" {
		return false
	}
	if !embeddedAgentLibraryShipped {
		// Pre-W1.D1 fail-permissive mode: the embedded agent library
		// has not yet shipped, so the floor is structurally vacuous.
		// Production callers (LoadDefaultTemplate*) get a no-op pass on
		// the validator until W1.D1 lights the library. Tests that need
		// to exercise the hard-fail path inject an explicit
		// LoadOptions.AgentLookupFn.
		return true
	}
	for _, group := range embeddedAgentGroups {
		path := "builtin/agents/" + group + "/" + name + ".md"
		f, err := DefaultTemplateFS.Open(path)
		if err != nil {
			continue
		}
		_ = f.Close()
		return true
	}
	return false
}

// validateAgentBindingNames asserts every AgentBinding.AgentName resolves at
// the EMBEDDED tier of the 3-tier agent resolver per SKETCH.md §3.4. The
// embedded tier is the union of `internal/templates/builtin/agents/{till-gen,
// till-go,till-gdd}/<name>.md`; resolution succeeds on the first hit across
// the three groups.
//
// Hard-fail (NOT warn-only — distinct from validateAgentBindingFiles, which
// checks `~/.claude/agents/<name>.md` warn-only for dev-machine state). A
// dangling agent_name reference is a template-correctness failure: typos
// like "buidler-agent" silently survive Load today and surface only at
// dispatch time when the dispatcher cannot resolve the binding.
//
// Project-tier (`<projectRoot>/.tillsyn/agents/<name>.md`) and user-tier
// (`~/.tillsyn/agents/<group>/<name>.md`) checks are NOT performed here —
// those are spawn-time concerns. The embedded floor is the load-time
// invariant every binding must clear; project + user tiers can legitimately
// add (override) but cannot subtract.
//
// lookupFn nil resolves to defaultAgentLookupFn (walks DefaultTemplateFS
// across embeddedAgentGroups). Tests inject a synthetic lookupFn so the
// validator's behaviour is deterministic regardless of which embedded files
// have shipped.
//
// Empty AgentName is rejected with a distinct error message ("agent_name is
// empty") before the lookup is attempted — the empty string cannot resolve
// to any `<group>/.md` filename, and the explicit message gives adopters a
// clearer diagnostic than a generic "not found at embedded floor."
//
// Outer-map iteration order is non-deterministic (Go map iteration); the
// validator returns on the FIRST offending binding to keep the error surface
// bounded. Future drops that want exhaustive reporting can switch to error
// aggregation; the closed-enum + load-time-reject pattern doesn't need it
// today.
//
// All non-nil returns wrap ErrUnknownAgentName so callers using
// `errors.Is(err, ErrUnknownAgentName)` route correctly.
//
// Drop 4c.6 W0.5.D2 hook.
func validateAgentBindingNames(tpl Template, lookupFn func(string) bool) error {
	if lookupFn == nil {
		lookupFn = defaultAgentLookupFn
	}
	for kind, binding := range tpl.AgentBindings {
		name := binding.AgentName
		if name == "" {
			return fmt.Errorf("%w: agent_bindings[%q].agent_name is empty",
				ErrUnknownAgentName, kind)
		}
		if lookupFn(name) {
			continue
		}
		return fmt.Errorf("%w: agent_bindings[%q].agent_name %q does not resolve at the embedded floor (looked under builtin/agents/{%s}/%s.md)",
			ErrUnknownAgentName, kind, name, strings.Join(embeddedAgentGroups, ","), name)
	}
	return nil
}

// knownWiredConsumers is the closed Go-internal set of consumer identifiers
// representing template features that have at least one wired runtime consumer.
// validateClaimVsImplCoherence (Drop 4c.6 W0.5.D6) checks every claimed
// `[[child_rules]]` output kind / template feature against this set and rejects
// any claim whose consumer identifier is not a member.
//
// For Drop 4c.6 the map is INTENTIONALLY EMPTY per L1 W0.5 sub-plan container
// Acceptance bullet 4 + Open Question #1 resolution. The validator + sentinel
// + sentinel-test ship now against the empty set; Drop 4c.7 W7 will add
// `child_rules_for` (the dispatcher-side consumer for `[[child_rules]]`
// auto-create) and Drop 4c.7 W8 will add `context_resolve` (the
// context-block resolver) when those waves wire the first real runtime
// consumers.
//
// LOUD WARNING TO FUTURE DROPS: adding a runtime consumer for a
// template-claimed feature requires adding the consumer's identifier to this
// map. Failing to do so will cause every template that claims the new feature
// to fail Load with ErrClaimVsImplUnknownConsumer. Conversely, adding an
// entry to this map WITHOUT also wiring the runtime consumer recreates the
// shipped-but-not-wired anti-pattern (Drop 3 droplet 3.20) this validator
// exists to prevent — TestLoadValidatesClaimVsImplCoherenceEmptyKnownWired-
// SetGuard pins the Drop 4c.6 invariant of length 0; that guard's expected
// length advances when Drop 4c.7 W7 + W8 land.
//
// The map's value type is `struct{}` (zero-byte sentinel) — set membership
// is the only relation tested. Mutations from tests are guarded by t.Cleanup
// so the production set is restored after each test row.
//
// Drop 4c.6 W0.5.D6 hook.
var knownWiredConsumers = map[string]struct{}{}

// defaultClaimedConsumersFn is the production walker that extracts the
// claimed-consumer list from a parsed Template. Today's schema has no
// `[[child_rules]] consumer = "..."` field — no claim can be authored against
// the known-wired-consumer set without test-only injection — so the walker
// returns an empty slice for every template. The function exists as its own
// callable rather than inlined into validateClaimVsImplCoherence so a future
// schema addition (e.g. a `consumer = "..."` field on ChildRule) extends the
// production walker in one place rather than threading new extraction logic
// through the validator body.
//
// Drop 4c.6 W0.5.D6 hook.
func defaultClaimedConsumersFn(_ Template) []string {
	return nil
}

// validateClaimVsImplCoherence checks every claimed template-feature consumer
// identifier against the closed Go-internal `knownWiredConsumers` map and
// rejects any claim whose identifier is not a member with
// ErrClaimVsImplUnknownConsumer.
//
// claimsFn parameter — test-only injection point:
//
//   - When claimsFn is non-nil, the validator walks the supplied claim list
//     verbatim. Tests use this to exercise both the rejection path (claim
//     not in the empty `knownWiredConsumers` set) and the success path
//     (claim temporarily registered in `knownWiredConsumers` via
//     t.Cleanup-restored mutation).
//
//   - When claimsFn is nil, the production walker (defaultClaimedConsumersFn)
//     returns an empty slice for every template. Today's schema has no
//     `[[child_rules]] consumer = "..."` field, so production callers
//     pass through the validator vacuously. The sentinel + tests ship now
//     so Drop 4c.7 W7 (`child_rules_for`) and Drop 4c.7 W8 (`context_resolve`)
//     inherit the validator without further work.
//
// `knownWiredConsumers` is the closed Go-internal source-of-truth — the
// validator does NOT parse `CLAUDE.md` at runtime. CLAUDE.md § Cascade Tree
// Structure is the authoring reference for adopters but is not consulted at
// Load.
//
// Per W0.5 plan FF1 disclosure: pelletier/go-toml/v2's post-decode validators
// do not carry source-line numbers, so the wrapped sentinel message names the
// offending consumer identifier rather than `line=N`. Adopters grep their TOML
// for whatever schema field claims that consumer once the schema gains a
// `[[child_rules]] consumer = "..."` axis.
//
// Drop 4c.6 W0.5.D6 hook.
func validateClaimVsImplCoherence(tpl Template, claimsFn func(tpl Template) []string) error {
	if claimsFn == nil {
		claimsFn = defaultClaimedConsumersFn
	}
	for _, claimed := range claimsFn(tpl) {
		if _, ok := knownWiredConsumers[claimed]; !ok {
			return fmt.Errorf("%w: claimed consumer %q has no wired runtime implementation (knownWiredConsumers is closed; see internal/templates/load.go and CLAUDE.md § Cascade Tree Structure)",
				ErrClaimVsImplUnknownConsumer, claimed)
		}
	}
	return nil
}
