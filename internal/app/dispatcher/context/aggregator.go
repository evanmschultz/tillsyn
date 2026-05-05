// Package context implements the dispatcher's pre-spawn context aggregator
// engine. Given an [templates.AgentBinding] context-rules block plus the
// triggering [domain.ActionItem] and a pair of injected ports
// ([ActionItemReader] + [GitDiffReader]), the engine returns a [Bundle] that
// the spawn-pipeline render layer (F.7-CORE F.7.3b) writes into the agent's
// spawn directory.
//
// The engine is a pure function: no global state, no logging, no filesystem
// or git side effects beyond the injected ports. Two-axis caps are enforced
// via [stdcontext.WithTimeout] (per-rule + per-bundle) and a greedy-fit
// character-count walk per master PLAN.md L14/L15.
//
// Empty-binding short-circuit (master PLAN.md L13): when the supplied
// [templates.ContextRules] declares no rules — every boolean false, every
// slice empty, every cap zero — Resolve returns an empty Bundle without doing
// any reads. Adopters who want fully-agentic spawns simply omit `[context]`
// from their template TOML.
//
// Package-name note: this package is `context`, sharing the name with the
// standard library `context` package. The stdlib package is imported as
// `stdcontext` to avoid a naming collision. Callers from outside this package
// import it as `aggcontext "github.com/evanmschultz/tillsyn/internal/app/dispatcher/context"`
// or similar.
package context

import (
	stdcontext "context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/evanmschultz/tillsyn/internal/domain"
	"github.com/evanmschultz/tillsyn/internal/templates"
)

// Default-substitution constants per master PLAN.md L13/L14/L15. The schema
// layer (templates.ContextRules / templates.Tillsyn) accepts zero-valued caps
// + timeouts; the engine substitutes these defaults at Resolve-time so the
// schema stays "field present means non-default; field absent means default."
const (
	// defaultBundleCharCap is the bundle-global character cap engine-time
	// default substituted when [ResolveArgs.BundleCharCap] is zero.
	defaultBundleCharCap = 200_000

	// defaultRuleCharCap is the per-rule character cap engine-time default
	// substituted when [templates.ContextRules.MaxChars] is zero.
	defaultRuleCharCap = 50_000

	// defaultBundleDuration is the per-bundle wall-clock budget engine-time
	// default substituted when [ResolveArgs.BundleDuration] is zero.
	defaultBundleDuration = 2 * time.Second

	// defaultRuleDuration is the per-rule wall-clock budget engine-time
	// default substituted when [templates.ContextRules.MaxRuleDuration] is
	// zero.
	defaultRuleDuration = 500 * time.Millisecond
)

// Closed-enum rule names. The slice [allRuleNames] below pins iteration order
// to the [templates.ContextRules] struct-declaration order, matching the
// "TOML declaration order" rule from the F.7.18 spec. Adding a new rule
// requires a constant here, an entry in [allRuleNames], a switch arm in
// [Resolve], and a resolver in rules.go — the ordered slice is the single
// source of truth for ordering, NOT a switch on the constant string.
const (
	ruleParent            = "parent"
	ruleParentGitDiff     = "parent_git_diff"
	ruleSiblingsByKind    = "siblings_by_kind"
	ruleAncestorsByKind   = "ancestors_by_kind"
	ruleDescendantsByKind = "descendants_by_kind"
)

// allRuleNames lists every rule the engine evaluates, in the canonical
// struct-declaration order. The order is significant: greedy-fit cap
// enforcement walks rules in this order, so the first-fit semantics are
// stable across runs.
var allRuleNames = []string{
	ruleParent,
	ruleParentGitDiff,
	ruleSiblingsByKind,
	ruleAncestorsByKind,
	ruleDescendantsByKind,
}

// ActionItemReader is the read-only port the aggregator needs for tree walks.
// The dispatcher's adapter layer (F.7-CORE F.7.3b consumer) supplies a
// concrete implementation backed by the SQLite repository. Tests inject a
// table-driven mock so the engine stays pure.
type ActionItemReader interface {
	// GetActionItem returns the action item with the given ID. Returns a
	// non-nil error wrapping a domain-layer sentinel when the ID is unknown.
	GetActionItem(ctx stdcontext.Context, id string) (domain.ActionItem, error)

	// ListChildren returns every direct child of the given parent. Order is
	// adapter-defined; the aggregator does NOT depend on any particular
	// ordering. Empty slice + nil error is the "no children" signal.
	ListChildren(ctx stdcontext.Context, parentID string) ([]domain.ActionItem, error)

	// ListSiblings returns every direct child of the given parent. Identical
	// signature to ListChildren but documents intent: callers querying
	// siblings of an action item pass that item's ParentID. Empty slice +
	// nil error is the "no siblings" signal (single child or no parent).
	ListSiblings(ctx stdcontext.Context, parentID string) ([]domain.ActionItem, error)
}

// GitDiffReader is the read-only port the aggregator needs for the
// parent_git_diff rule. The dispatcher's adapter layer supplies a concrete
// implementation that shells out to `git diff <from>..<to>` in the project
// worktree; tests inject a deterministic mock.
type GitDiffReader interface {
	// Diff returns the byte content of `git diff <fromCommit>..<toCommit>`.
	// Empty fromCommit OR empty toCommit MUST yield (nil, nil) — the "no
	// commit anchors" case is signaled to the engine via empty bytes, not via
	// an error, so the engine can emit a marker rather than fail the whole
	// bundle.
	Diff(ctx stdcontext.Context, fromCommit, toCommit string) ([]byte, error)
}

// ResolveArgs are the inputs to [Resolve]. Every field is required except the
// caps + duration knobs, which substitute engine-time defaults when zero.
type ResolveArgs struct {
	// Binding is the agent binding for the spawning kind. The engine reads
	// only the Context sub-struct — every other field is opaque to the
	// aggregator.
	Binding templates.AgentBinding

	// Item is the action item the dispatcher is about to spawn an agent for.
	// The engine reads ParentID, ID, Kind, and (transitively) the parent's
	// fields via the injected reader.
	Item domain.ActionItem

	// ProjectID scopes any future reader calls that need it. Today the
	// reader interface does not require it (lookups are by action-item ID),
	// but the field is kept on ResolveArgs so callers can populate it
	// without API churn when the reader gains project-scoped methods.
	ProjectID string

	// BundleCharCap is the resolved Tillsyn.MaxContextBundleChars. Zero
	// substitutes [defaultBundleCharCap] (200_000) at Resolve-time.
	BundleCharCap int

	// BundleDuration is the resolved Tillsyn.MaxAggregatorDuration. Zero
	// substitutes [defaultBundleDuration] (2 * time.Second) at Resolve-time.
	BundleDuration time.Duration

	// Reader is the injected action-item reader. Must be non-nil whenever
	// Binding.Context declares any rule that walks the action-item tree.
	// When the binding is empty (every rule disabled), the reader is never
	// called and may be nil.
	Reader ActionItemReader

	// DiffReader is the injected git-diff reader. Must be non-nil whenever
	// Binding.Context.ParentGitDiff is true. Otherwise may be nil.
	DiffReader GitDiffReader
}

// Bundle is the aggregator's output, consumed by the F.7-CORE spawn-pipeline
// render layer (F.7.3b). Three independent surfaces:
//
//   - RenderedInline carries content destined for the spawn's
//     `system-append.md` file. When [templates.ContextRules.Delivery] is
//     "inline", this field holds the rendered rule content concatenated with
//     truncation/timeout markers. When Delivery is "file", this field holds
//     ONLY markers (the rule content lives in Files).
//
//   - Files carries content destined for `<bundle>/context/<filename>`. The
//     map key is the filename relative to the context directory (e.g.
//     "parent.md", "parent_git_diff.full") and the value is the raw bytes.
//     The render layer is responsible for writing each entry.
//
//   - Markers carries the ordered list of skip / truncation / timeout
//     markers emitted while resolving rules. Each marker is a single line
//     of human-readable text. The markers slice is the audit trail; for
//     "file" delivery it is the only content that lands in
//     `system-append.md` so the agent knows which rules were skipped.
type Bundle struct {
	// RenderedInline is the inline-mode payload (system-append.md content).
	// Empty string when no rules render inline.
	RenderedInline string

	// Files maps filename → content for file-mode rules. Empty map (or nil)
	// when no rules render to file. The render layer writes each entry to
	// `<bundle>/context/<filename>`.
	Files map[string][]byte

	// Markers is the ordered list of skip / truncation / timeout markers
	// emitted while resolving rules. Always non-nil but may be empty.
	Markers []string
}

// ErrNilReader is returned by [Resolve] when the binding declares a rule
// that requires the action-item reader but [ResolveArgs.Reader] is nil. The
// error wraps a precise message naming the missing reader. Callers route on
// errors.Is.
var ErrNilReader = errors.New("aggregator: action-item reader is nil")

// ErrNilDiffReader is returned by [Resolve] when the binding declares
// ParentGitDiff but [ResolveArgs.DiffReader] is nil. Callers route on
// errors.Is.
var ErrNilDiffReader = errors.New("aggregator: git-diff reader is nil")

// Resolve runs the per-rule pipeline against args and returns a [Bundle].
//
// Algorithm (master PLAN.md L13–L15):
//
//  1. If args.Binding.Context is the zero value (no rules declared), return
//     an empty Bundle. This is the FAST PATH for fully-agentic spawns —
//     no reader calls, no allocations beyond the empty Bundle.
//
//  2. Substitute engine-time defaults for zero-valued caps + timeouts.
//
//  3. Wrap the supplied context with the bundle-global wall-clock cap.
//
//  4. Iterate rules in struct-declaration order
//     ([allRuleNames]). For each rule:
//
//     a. Skip the rule if the binding has it disabled.
//
//     b. Wrap the bundle context with the per-rule wall-clock cap and call
//     the rule's resolver (rules.go).
//
//     c. On per-rule timeout: emit a marker, continue to the next rule.
//
//     d. On per-rule error other than timeout: return the error wrapped
//     with the rule name.
//
//     e. Truncate per-rule output to MaxChars; emit a truncation marker
//     and stash the full content under "<rule>.full" in Files.
//
//     f. Greedy-fit cap: if cumulative + ruleSize > BundleCharCap, SKIP
//     the rule (emit a skip marker), continue to the next rule. Earlier
//     rules that fit stay in the bundle; subsequent rules that fit still
//     land. Greedy-fit, NOT serial-drop.
//
//  5. Render the surviving rule content to either RenderedInline (Delivery=
//     "inline") or Files (Delivery="file" or empty). Markers always land in
//     RenderedInline so the agent's system-append.md surfaces them.
//
//  6. On bundle timeout: emit an outer-timeout marker listing pending
//     rules, return the partial bundle with a nil error (timeout is a
//     soft-degradation, not a hard failure).
func Resolve(ctx stdcontext.Context, args ResolveArgs) (Bundle, error) {
	rules := args.Binding.Context

	// Fast path: empty binding => agentic mode. No reader calls, no work.
	if isEmptyContextRules(rules) {
		return Bundle{}, nil
	}

	// Default substitution. Zero-valued knobs at the schema layer pick up
	// engine-time defaults here. Negative values are caught at template Load
	// time by templates.validateAgentBindingContext / validateTillsyn.
	bundleCharCap := args.BundleCharCap
	if bundleCharCap == 0 {
		bundleCharCap = defaultBundleCharCap
	}
	bundleDuration := args.BundleDuration
	if bundleDuration == 0 {
		bundleDuration = defaultBundleDuration
	}
	ruleCharCap := rules.MaxChars
	if ruleCharCap == 0 {
		ruleCharCap = defaultRuleCharCap
	}
	ruleDuration := time.Duration(rules.MaxRuleDuration)
	if ruleDuration == 0 {
		ruleDuration = defaultRuleDuration
	}

	delivery := rules.Delivery
	if delivery == "" {
		delivery = templates.ContextDeliveryFile
	}

	// Outer hard ceiling — every rule's resolver runs under this context.
	outerCtx, cancelOuter := stdcontext.WithTimeout(ctx, bundleDuration)
	defer cancelOuter()

	bundle := Bundle{
		Files:   map[string][]byte{},
		Markers: []string{},
	}

	// per-rule rendered payloads (filename → content) accumulated in struct-
	// declaration order. We render to a stable sequence first, then post-
	// process for delivery + greedy-fit so a single decision point governs
	// what survives into RenderedInline vs Files.
	type ruleOutput struct {
		name    string
		content []byte
		// truncatedFull is the pre-truncation content when truncation
		// occurred (else nil). Always written to Files["<rule>.full"].
		truncatedFull []byte
	}
	var renderedRules []ruleOutput

	// inlineBuilder + cumulativeChars track greedy-fit at render time. We
	// count cumulative against the bundleCharCap as each rule completes so
	// later rules see accurate state.
	cumulativeChars := 0

	pendingOnTimeout := func(remainingFrom int) []string {
		out := []string{}
		for j := remainingFrom; j < len(allRuleNames); j++ {
			name := allRuleNames[j]
			if isRuleEnabled(rules, name) {
				out = append(out, name)
			}
		}
		return out
	}

	for i, name := range allRuleNames {
		// Outer timeout check before each rule. If outer is dead, emit the
		// aggregator-timeout marker (with pending list) and break.
		if err := outerCtx.Err(); err != nil {
			pending := pendingOnTimeout(i)
			bundle.Markers = append(bundle.Markers, fmt.Sprintf(
				"[aggregator timed out after %s; rules pending: %s]",
				bundleDuration, strings.Join(pending, ","),
			))
			break
		}

		if !isRuleEnabled(rules, name) {
			continue
		}

		// Per-rule timeout wrapper.
		ruleCtx, cancelRule := stdcontext.WithTimeout(outerCtx, ruleDuration)
		content, err := evaluateRule(ruleCtx, name, args, rules)
		cancelRule()

		if err != nil {
			// Distinguish per-rule timeout from outer timeout. If the
			// outer is also dead, we're in aggregator-timeout territory —
			// emit that marker on the next loop iteration (or below).
			if errors.Is(err, stdcontext.DeadlineExceeded) || errors.Is(err, stdcontext.Canceled) {
				if outerCtx.Err() != nil {
					// Outer dead too. Emit aggregator-timeout marker with
					// pending rules from this point and break.
					pending := pendingOnTimeout(i)
					bundle.Markers = append(bundle.Markers, fmt.Sprintf(
						"[aggregator timed out after %s; rules pending: %s]",
						bundleDuration, strings.Join(pending, ","),
					))
					break
				}
				// Per-rule timeout only — outer still alive. Emit per-rule
				// timeout marker, continue to next rule.
				bundle.Markers = append(bundle.Markers, fmt.Sprintf(
					"[rule %s timed out after %s; partial output discarded]",
					name, ruleDuration,
				))
				continue
			}
			// Non-timeout error: bubble up wrapped with the rule name.
			return Bundle{}, fmt.Errorf("aggregator: rule %s: %w", name, err)
		}

		// Empty content is a legitimate "nothing to render" result (e.g.
		// parent_git_diff with no commit anchors, ancestor walk that
		// reached root without a match). Skip the rule entirely — no
		// file written, no marker emitted, no cap charge.
		if len(content) == 0 {
			continue
		}

		// Per-rule cap (truncation): truncate the rule's content to
		// ruleCharCap. The full pre-truncation content lands in Files
		// under "<rule>.full" so the agent can read it on demand.
		var truncatedFull []byte
		if len(content) > ruleCharCap {
			truncatedFull = content
			content = []byte(string(content[:ruleCharCap]))
			bundle.Markers = append(bundle.Markers, fmt.Sprintf(
				"[truncated to %d chars; full content at <bundle>/context/%s.full]",
				ruleCharCap, name,
			))
		}

		// Bundle-global greedy-fit cap. Counted in cumulativeChars; if this
		// rule would push past the cap, SKIP and continue to the next rule
		// (earlier fits stay; later rules that fit still land).
		ruleSize := len(content)
		remaining := bundleCharCap - cumulativeChars
		if ruleSize > remaining {
			bundle.Markers = append(bundle.Markers, fmt.Sprintf(
				"[skipped: %s (would have added %d; remaining %d)]",
				name, ruleSize, remaining,
			))
			// We still preserve the full content in Files["<rule>.full"]
			// when truncation happened above so the agent can read it on
			// demand even if the rule was skipped from the bundle. The
			// truncated content itself is dropped.
			if truncatedFull != nil {
				bundle.Files[name+".full"] = truncatedFull
			}
			continue
		}
		cumulativeChars += ruleSize

		renderedRules = append(renderedRules, ruleOutput{
			name:          name,
			content:       content,
			truncatedFull: truncatedFull,
		})
	}

	// Stitch the surviving rules into the chosen delivery surface.
	var inline strings.Builder
	for _, r := range renderedRules {
		// Always preserve the truncation full-content side channel.
		if r.truncatedFull != nil {
			bundle.Files[r.name+".full"] = r.truncatedFull
		}

		switch delivery {
		case templates.ContextDeliveryInline:
			if inline.Len() > 0 {
				inline.WriteString("\n\n")
			}
			inline.WriteString("## ")
			inline.WriteString(r.name)
			inline.WriteString("\n\n")
			inline.Write(r.content)
		default:
			// File mode (templates.ContextDeliveryFile or empty). Each
			// rule's content lands in Files["<rule>.md"]; only markers
			// surface in RenderedInline (small, system-append.md).
			bundle.Files[r.name+".md"] = r.content
		}
	}

	// Markers always land in RenderedInline so the agent's system-append.md
	// surfaces them. For inline delivery, markers append after the rule
	// content. For file delivery, markers ARE the inline payload.
	if len(bundle.Markers) > 0 {
		if inline.Len() > 0 {
			inline.WriteString("\n\n")
		}
		for i, marker := range bundle.Markers {
			if i > 0 {
				inline.WriteString("\n")
			}
			inline.WriteString(marker)
		}
	}
	bundle.RenderedInline = inline.String()

	return bundle, nil
}

// isEmptyContextRules reports whether the supplied rules struct declares no
// rules at all. The check is field-by-field rather than reflect.DeepEqual
// against the zero value because slice-vs-nil distinctions in TOML decode
// would otherwise force every empty-binding test to use a concrete nil slice
// literal. Empty slices and nil slices are equivalent here per master
// PLAN.md L13.
//
// A rules struct is considered empty when every boolean is false, every
// slice has zero length, every cap is zero, and Delivery is empty. The
// Delivery check matters: a binding declaring `[context] delivery = "file"`
// with every other field absent is EMPTY semantically (the agent gets no
// content), but adopters who set delivery explicitly likely intend to add
// rules later, so we treat that as a non-empty binding (returns false here)
// and let the per-rule loop produce a marker-only bundle.
func isEmptyContextRules(rules templates.ContextRules) bool {
	return !rules.Parent &&
		!rules.ParentGitDiff &&
		len(rules.SiblingsByKind) == 0 &&
		len(rules.AncestorsByKind) == 0 &&
		len(rules.DescendantsByKind) == 0 &&
		rules.MaxChars == 0 &&
		rules.MaxRuleDuration == 0 &&
		rules.Delivery == ""
}

// isRuleEnabled reports whether the supplied rule name is enabled by the
// rules struct. Boolean rules (parent, parent_git_diff) check the bool
// directly; kind-list rules are enabled iff the slice is non-empty.
func isRuleEnabled(rules templates.ContextRules, name string) bool {
	switch name {
	case ruleParent:
		return rules.Parent
	case ruleParentGitDiff:
		return rules.ParentGitDiff
	case ruleSiblingsByKind:
		return len(rules.SiblingsByKind) > 0
	case ruleAncestorsByKind:
		return len(rules.AncestorsByKind) > 0
	case ruleDescendantsByKind:
		return len(rules.DescendantsByKind) > 0
	default:
		return false
	}
}

// evaluateRule dispatches to the per-rule resolver implementation in
// rules.go. The dispatch is a closed switch on the rule-name constant; the
// allRuleNames slice keeps iteration order stable.
func evaluateRule(
	ctx stdcontext.Context,
	name string,
	args ResolveArgs,
	rules templates.ContextRules,
) ([]byte, error) {
	switch name {
	case ruleParent:
		if args.Reader == nil {
			return nil, ErrNilReader
		}
		return resolveParent(ctx, args.Item, args.Reader)
	case ruleParentGitDiff:
		if args.Reader == nil {
			return nil, ErrNilReader
		}
		if args.DiffReader == nil {
			return nil, ErrNilDiffReader
		}
		return resolveParentGitDiff(ctx, args.Item, args.Reader, args.DiffReader)
	case ruleSiblingsByKind:
		if args.Reader == nil {
			return nil, ErrNilReader
		}
		return resolveSiblingsByKind(ctx, args.Item, rules.SiblingsByKind, args.Reader)
	case ruleAncestorsByKind:
		if args.Reader == nil {
			return nil, ErrNilReader
		}
		return resolveAncestorsByKind(ctx, args.Item, rules.AncestorsByKind, args.Reader)
	case ruleDescendantsByKind:
		if args.Reader == nil {
			return nil, ErrNilReader
		}
		return resolveDescendantsByKind(ctx, args.Item, rules.DescendantsByKind, args.Reader)
	default:
		return nil, fmt.Errorf("aggregator: unknown rule %q", name)
	}
}

// kindMatches reports whether kind is in the list of accepted kinds. Closed
// linear scan — the kind lists are template-bounded (max ~12 entries) so a
// linear scan is faster than building a map per evaluation.
func kindMatches(kind domain.Kind, accepted []domain.Kind) bool {
	for _, a := range accepted {
		if a == kind {
			return true
		}
	}
	return false
}
