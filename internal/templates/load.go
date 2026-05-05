package templates

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"

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
