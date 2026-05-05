package templates

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"

	toml "github.com/pelletier/go-toml/v2"

	"github.com/evanmschultz/tillsyn/internal/domain"
)

// Load parses a Tillsyn template TOML stream and validates it.
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
//     b. validateChildRuleKinds — assert every Kind referenced in
//     [child_rules] is a member of the closed enum.
//     c. validateChildRuleCycles — DFS the parent → child kind graph for
//     directed cycles.
//     d. validateChildRuleReachability — reserved extension point;
//     currently a no-op.
//     e. validateGateKinds — assert every gate-kind string in
//     Template.Gates value slices is a member of the closed
//     GateKind enum (4b.1 hook).
//     f. validateAgentBindingEnvNames — assert every entry in each
//     AgentBinding.Env slice matches the closed env-var name regex
//     (`^[A-Za-z][A-Za-z0-9_]*$`), is non-empty, contains no `=`, and
//     is unique within its binding (Drop 4c F.7.17.1 hook).
//     g. validateAgentBindingContext — assert every AgentBinding.Context
//     sub-struct satisfies the closed delivery enum, non-negative MaxChars
//     and MaxRuleDuration, and that every kind referenced by the kind-walk
//     fields (SiblingsByKind / AncestorsByKind / DescendantsByKind) is a
//     member of the closed 12-value Kind enum. Drop 4c F.7.18.1 hook.
//     h. validateTillsyn — assert the top-level [tillsyn] globals satisfy
//     the closed contract: non-negative MaxContextBundleChars and
//     MaxAggregatorDuration. Zero is legal (engine-time default
//     substitution); negative values are rejected. Drop 4c F.7.18.2 hook.
//
// Sentinel errors at package scope wrap the underlying failure so callers
// can use errors.Is for routing without reaching into pelletier/go-toml/v2
// internals.
func Load(r io.Reader) (Template, error) {
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
	if err := validateMapKeys(tpl); err != nil {
		return Template{}, err
	}
	if err := validateChildRuleKinds(tpl.ChildRules); err != nil {
		return Template{}, err
	}
	if err := validateChildRuleCycles(tpl.ChildRules); err != nil {
		return Template{}, err
	}
	if err := validateChildRuleReachability(tpl.ChildRules); err != nil {
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

	// ErrUnreachableChildRule is reserved for future expansion of the
	// reachability validator. Drop 3's reachability pass collapses to a
	// no-op because every member of the closed 12-value Kind enum is
	// reachable from project-creation; later drops that introduce
	// reachability semantics beyond closed-enum membership will surface
	// this sentinel.
	ErrUnreachableChildRule = errors.New("template child_rules contain an unreachable rule")

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

	// ErrInvalidTillsynGlobals is returned by validateTillsyn when the
	// top-level [tillsyn] table contains a field that fails the closed
	// rule contract (Drop 4c F.7.18.2 acceptance criteria):
	//
	//   - MaxContextBundleChars is negative (zero is legal — engine-time
	//     default substitution applies per master PLAN L14).
	//   - MaxAggregatorDuration is negative (zero is legal — engine-time
	//     default substitution applies per master PLAN L15).
	//
	// The wrapped message names the offending field and the offending
	// value for UX. The sentinel is a top-level Load error rather than a
	// nested wrap of ErrInvalidAgentBinding because the [tillsyn] table is
	// distinct from [agent_bindings] — failures here are global, not
	// per-binding.
	ErrInvalidTillsynGlobals = errors.New("invalid tillsyn globals")
)

// validateMapKeys asserts every key in Template.Kinds,
// Template.AgentBindings, and Template.Gates is a member of the closed
// 12-value domain.Kind enum. Catches typos like [kinds.bulid] (transposed
// letters), [agent_bindings.totally-bogus], or [gates.bogus-kind] at load
// time rather than letting them silently coexist with the real entries —
// strict decode validates fields inside a row but not the map keys themselves,
// because pelletier/go-toml/v2 treats arbitrary keys as legitimate map
// entries when the destination type is a map.
func validateMapKeys(tpl Template) error {
	for k := range tpl.Kinds {
		if !domain.IsValidKind(k) {
			return fmt.Errorf("%w: kinds map key %q", ErrUnknownKindReference, k)
		}
	}
	for k := range tpl.AgentBindings {
		if !domain.IsValidKind(k) {
			return fmt.Errorf("%w: agent_bindings map key %q", ErrUnknownKindReference, k)
		}
	}
	for k := range tpl.Gates {
		if !domain.IsValidKind(k) {
			return fmt.Errorf("%w: gates map key %q", ErrUnknownKindReference, k)
		}
	}
	return nil
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

// validateChildRuleCycles runs DFS over the parent → child kind graph
// derived from [child_rules] to detect directed cycles. The choice of
// visited-set vs colored-DFS is left to the implementer per Drop 3 finding
// 5.B.4 — this implementation uses a single recursion-stack set so a node
// is reported as part of a cycle precisely when a successor's traversal
// re-enters it.
func validateChildRuleCycles(rules []ChildRule) error {
	if len(rules) == 0 {
		return nil
	}

	graph := make(map[domain.Kind][]domain.Kind, len(rules))
	for _, rule := range rules {
		graph[rule.WhenParentKind] = append(graph[rule.WhenParentKind], rule.CreateChildKind)
	}

	const (
		colorWhite = 0 // unseen
		colorGray  = 1 // on current DFS path
		colorBlack = 2 // fully explored
	)
	color := make(map[domain.Kind]int, len(graph))

	var dfs func(node domain.Kind, stack []domain.Kind) error
	dfs = func(node domain.Kind, stack []domain.Kind) error {
		color[node] = colorGray
		stack = append(stack, node)
		for _, next := range graph[node] {
			switch color[next] {
			case colorGray:
				cycle := append(append([]domain.Kind{}, stack...), next)
				return fmt.Errorf("%w: %s", ErrTemplateCycle, formatCyclePath(cycle, next))
			case colorWhite:
				if err := dfs(next, stack); err != nil {
					return err
				}
			}
		}
		color[node] = colorBlack
		return nil
	}

	for node := range graph {
		if color[node] == colorWhite {
			if err := dfs(node, nil); err != nil {
				return err
			}
		}
	}
	return nil
}

// formatCyclePath renders a cycle's traversal as a "kindA -> kindB -> kindA"
// string for the wrapped error message. The closure point is the kind where
// the back edge re-entered the recursion stack.
func formatCyclePath(stack []domain.Kind, closure domain.Kind) string {
	startIdx := 0
	for idx, k := range stack {
		if k == closure {
			startIdx = idx
			break
		}
	}
	parts := make([]string, 0, len(stack)-startIdx+1)
	for _, k := range stack[startIdx:] {
		parts = append(parts, string(k))
	}
	parts = append(parts, string(closure))
	return strings.Join(parts, " -> ")
}

// validateChildRuleReachability collapses to a no-op for Drop 3 by design.
// Per droplet 3.9's contract, a [child_rules] WhenParentKind is "reachable"
// if it appears as a CreateChildKind of another rule OR if it is a member
// of the closed 12-value Kind enum (project-creation can spawn any kind
// directly). validateChildRuleKinds already enforces enum membership for
// every WhenParentKind, so reachability is automatically satisfied.
//
// The function exists as a named extension point so later drops that
// introduce reachability semantics beyond closed-enum membership (for
// example, project-level allowed_kinds restrictions) can return
// ErrUnreachableChildRule without reshuffling the public API.
func validateChildRuleReachability(rules []ChildRule) error {
	_ = rules
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

// validateTillsyn asserts the top-level [tillsyn] table satisfies the Drop 4c
// F.7.18.2 closed contract:
//
//   - MaxContextBundleChars is non-negative. Zero is legal and means "use
//     bundle-global default at engine-time" (F.7.18.4 substitutes 200000 per
//     master PLAN L14).
//   - MaxAggregatorDuration is non-negative. Zero is legal and means "use
//     bundle-global default at engine-time" (F.7.18.4 substitutes 2s per
//     master PLAN L15).
//
// All non-nil returns wrap ErrInvalidTillsynGlobals so callers can route on
// the sentinel via errors.Is. The validator runs after
// validateAgentBindingContext so per-binding failures surface with their
// original sentinel rather than being masked by a global rule.
//
// Per REV-3 the Tillsyn struct ships with exactly two fields in F.7.18.2;
// F.7-CORE F.7.1 + F.7.6 extend it with SpawnTempRoot + RequiresPlugins
// later. Strict-decode unknown-key rejection on the Tillsyn struct is
// inherited automatically from load.go step 3 (DisallowUnknownFields), so
// future extenders do not need to reshape this validator — they add their
// own field-level checks alongside the existing two.
func validateTillsyn(tpl Template) error {
	if tpl.Tillsyn.MaxContextBundleChars < 0 {
		return fmt.Errorf("%w: max_context_bundle_chars must be >= 0 (got %d)",
			ErrInvalidTillsynGlobals, tpl.Tillsyn.MaxContextBundleChars)
	}
	if time.Duration(tpl.Tillsyn.MaxAggregatorDuration) < 0 {
		return fmt.Errorf("%w: max_aggregator_duration must be >= 0 (got %s)",
			ErrInvalidTillsynGlobals, time.Duration(tpl.Tillsyn.MaxAggregatorDuration))
	}
	return nil
}
