package templates

import (
	"github.com/evanmschultz/tillsyn/internal/domain"
)

// KindCatalog is the per-project baked snapshot of kind definitions and
// agent bindings derived from a Template at project-creation time. Lookups
// are O(1) map access; the catalog is treated as immutable after Bake — no
// mutator methods are exposed.
//
// Per Drop 3 fix L5 (CE4 import-cycle resolution) the catalog is owned by
// internal/templates and is persisted on a Project via the
// KindCatalogJSON json.RawMessage envelope on Project. Decoding from that
// envelope happens in internal/app or internal/templates — never on the
// Project type itself, which would re-introduce the
// internal/domain → internal/templates dependency.
//
// Per Drop 3 finding 5.B.14 (runtime mutability): edits to a project's
// <project_root>/.tillsyn/template.toml AFTER project creation are ignored
// until the dev fresh-DBs ~/.tillsyn/tillsyn.db. The catalog is baked once
// and frozen for the project's lifetime.
//
// Canonical spec: main/PLAN.md § 19.3 droplet 3.12 + fix L5 + finding 5.B.14.
type KindCatalog struct {
	// SchemaVersion mirrors the Template.SchemaVersion the catalog was baked
	// from, so loaders can route on schema version after a JSON round-trip
	// without re-decoding the source TOML.
	SchemaVersion string `json:"schema_version"`

	// Kinds is the per-kind rule snapshot keyed by domain.Kind. Empty Kinds
	// (zero-value catalog) means "no template was bound at create time" —
	// callers must fall back to the legacy repo path per droplet 3.12
	// acceptance criterion (preserves boot compatibility with Drop 2.8
	// universal-nesting default).
	Kinds map[domain.Kind]KindRule `json:"kinds,omitempty"`

	// AgentBindings is the per-kind agent-spawn snapshot keyed by
	// domain.Kind. Empty bindings mean the dispatcher has no agent
	// configuration for this catalog — Drop 4 routes that case explicitly.
	AgentBindings map[domain.Kind]AgentBinding `json:"agent_bindings,omitempty"`
}

// Bake constructs a KindCatalog from a Template. The function is pure: no
// I/O, no clock dependency, no shared state. It deep-copies map entries so
// later mutations to t.Kinds or t.AgentBindings cannot bleed into the
// returned catalog.
//
// Bake is idempotent: Bake(t) deep-equals Bake(t) for any t. Tests assert
// this property explicitly (catalog_test.go TestKindCatalogBakeIsIdempotent).
//
// Canonical spec: main/PLAN.md § 19.3 droplet 3.12.
func Bake(t Template) KindCatalog {
	out := KindCatalog{
		SchemaVersion: t.SchemaVersion,
	}
	if len(t.Kinds) > 0 {
		out.Kinds = make(map[domain.Kind]KindRule, len(t.Kinds))
		for k, v := range t.Kinds {
			out.Kinds[k] = cloneKindRule(v)
		}
	}
	if len(t.AgentBindings) > 0 {
		out.AgentBindings = make(map[domain.Kind]AgentBinding, len(t.AgentBindings))
		for k, v := range t.AgentBindings {
			out.AgentBindings[k] = cloneAgentBinding(v)
		}
	}
	return out
}

// Lookup returns the KindRule recorded for the given kind plus a bool
// reporting whether the kind was present in the catalog. A zero-value
// KindCatalog (no Kinds map allocated) returns (zero, false) for any input
// — callers rely on this to fall back to the legacy repo path per droplet
// 3.12 acceptance criterion.
//
// Lookup performs no normalization: callers are expected to pass a Kind
// constant from internal/domain/kind.go. This matches Template.Kinds map
// usage everywhere else in the package.
func (c KindCatalog) Lookup(kind domain.Kind) (KindRule, bool) {
	if c.Kinds == nil {
		return KindRule{}, false
	}
	rule, ok := c.Kinds[kind]
	return rule, ok
}

// LookupAgentBinding returns the AgentBinding recorded for the given kind
// plus a bool reporting presence. Mirrors Lookup's contract: a zero-value
// catalog returns (zero, false) for any input.
func (c KindCatalog) LookupAgentBinding(kind domain.Kind) (AgentBinding, bool) {
	if c.AgentBindings == nil {
		return AgentBinding{}, false
	}
	binding, ok := c.AgentBindings[kind]
	return binding, ok
}

// AllowsNesting reports whether a child kind may nest under a parent kind per
// the catalog's KindRule allow-lists. It mirrors Template.AllowsNesting's
// contract by promoting the catalog's Kinds map into a Template envelope and
// delegating to the canonical resolver. Per droplet 3.15 this is the catalog-
// side entry point used by internal/app/kind_capability.go's parent-scope
// gate, replacing the legacy KindDefinition.AllowsParentScope check.
func (c KindCatalog) AllowsNesting(parent, child domain.Kind) (allowed bool, reason string) {
	return Template{Kinds: c.Kinds}.AllowsNesting(parent, child)
}

// cloneKindRule returns a deep copy of one KindRule so the catalog cannot
// observe later mutations to the source Template's slice fields.
func cloneKindRule(r KindRule) KindRule {
	out := KindRule{
		Owner:          r.Owner,
		StructuralType: r.StructuralType,
	}
	if len(r.AllowedParentKinds) > 0 {
		out.AllowedParentKinds = make([]domain.Kind, len(r.AllowedParentKinds))
		copy(out.AllowedParentKinds, r.AllowedParentKinds)
	}
	if len(r.AllowedChildKinds) > 0 {
		out.AllowedChildKinds = make([]domain.Kind, len(r.AllowedChildKinds))
		copy(out.AllowedChildKinds, r.AllowedChildKinds)
	}
	return out
}

// cloneAgentBinding returns a deep copy of one AgentBinding so the catalog
// cannot observe later mutations to the source Template's Tools slice.
func cloneAgentBinding(b AgentBinding) AgentBinding {
	out := b
	if len(b.Tools) > 0 {
		out.Tools = make([]string, len(b.Tools))
		copy(out.Tools, b.Tools)
	}
	return out
}
