// frontmatter.go — render-time YAML frontmatter key strip helper.
//
// Drop 4c.6 W3 (per SKETCH.md § 4.4 + § 15) renders agent .md files at runtime
// by composing the embedded source frontmatter with effective per-kind config
// from agents.toml. When agents.toml sets `model:` or `tools:` (and the
// related `allowedTools:` / `disallowedTools:` keys per § 15), the runtime
// owns the corresponding agent surface and the embedded frontmatter MUST NOT
// re-declare those keys — otherwise the rendered file fights the runtime.
//
// StripFrontmatterKeys is the single entry point W3 calls into. Lives in this
// sibling file (not agents.go) per W0.D4 PLAN.md so the schema layer (D1-D3,
// D5) and the render-helper layer (D4) stay file-isolated within the same
// package — neither layer's changes ripple into the other's surface.
//
// Implementation choices (verified by Context7 + `go doc` + dep survey before
// authoring; documented for future maintainers):
//
//   - YAML lib: gopkg.in/yaml.v3. Already in go.mod (line 75) as an indirect
//     dep of the existing dep graph; promoting to direct adds zero new deps.
//     v3's *yaml.Node API is the only path to order-preserving key removal —
//     the higher-level map[string]interface{} decode loses YAML key order.
//
//   - Both-false short-circuit returns the input string verbatim WITHOUT
//     parsing. yaml.v3's docstring is explicit: "the content when re-encoded
//     will not have its original textual representation preserved." A
//     parse-then-re-emit cycle would corrupt whitespace, comments, quoting
//     style, and key order even when no keys are stripped. Tests assert
//     byte-for-byte identity for this path.
//
//   - Top-level keys only. Per PLAN.md "constraint (high)" — a nested mapping
//     value with a `model:` or `tools:` child must NOT be stripped. The
//     implementation walks only the root MappingNode.Content slice.
//
//   - tools strip removes `tools:` AND `allowedTools:` AND `disallowedTools:`
//     as a unit (per SKETCH.md § 15: the runtime narrows the agent surface
//     to {name, description}; whichever of the three legacy keys appears in
//     embedded frontmatter must go).
package config

import (
	"bytes"
	"fmt"

	"gopkg.in/yaml.v3"
)

// frontmatterToolsKeys lists the three top-level YAML keys governed by the
// stripTools flag. Per SKETCH.md § 15, when agents.toml owns the per-kind
// tools_allow / tools_deny shape, the embedded agent file's tools surface is
// fully delegated to the runtime — every alias for that surface must go.
var frontmatterToolsKeys = []string{"tools", "allowedTools", "disallowedTools"}

// frontmatterModelKey is the single top-level YAML key governed by the
// stripModel flag. Listed as a slice for symmetry with frontmatterToolsKeys
// and to keep the strip loop uniform.
var frontmatterModelKey = "model"

// StripFrontmatterKeys returns the input frontmatter string with the
// `model:` and/or `tools:` (plus `allowedTools:` / `disallowedTools:`)
// top-level YAML keys removed according to stripModel and stripTools.
//
// frontmatter is the YAML body between an agent .md file's leading and
// trailing `---` delimiters, NOT including the delimiters themselves.
// W3's render layer handles delimiter stripping before calling in.
//
// When BOTH stripModel and stripTools are false, the input is returned
// verbatim — no YAML parse cycle runs, so whitespace, comments, and quoting
// style are preserved byte-for-byte. This is the documented "no-strip"
// short-circuit path.
//
// When EITHER flag is true, the helper:
//   1. Parses frontmatter into a *yaml.Node tree.
//   2. Walks the root MappingNode's Content slice (alternating key/value
//      pairs) and drops the targeted top-level keys. Nested mappings whose
//      keys happen to be named `model` / `tools` are NOT touched — only the
//      root level is in scope per PLAN.md "constraint (high)".
//   3. Re-encodes the modified tree via yaml.Marshal.
//
// Returns the modified frontmatter string and a nil error on success. On
// malformed YAML, returns ("", error) where the error message includes the
// parse position (yaml.v3 surfaces "line N: …" in its error strings).
//
// The function is pure: no I/O, no global state mutation. Safe for
// concurrent invocation from multiple goroutines.
//
// Callers (W3) MUST handle the empty-input case as a valid frontmatter (an
// agent file with empty frontmatter declares neither model nor tools and
// produces no error here regardless of flags).
func StripFrontmatterKeys(frontmatter string, stripModel bool, stripTools bool) (string, error) {
	// No-op short-circuit: return verbatim to preserve exact bytes.
	if !stripModel && !stripTools {
		return frontmatter, nil
	}

	// Empty input is a valid frontmatter (no keys to strip); short-circuit
	// to avoid yaml.v3 returning an empty-document warning or similar.
	if len(frontmatter) == 0 {
		return "", nil
	}

	// Parse into a *yaml.Node tree. yaml.Unmarshal returns a wrapped error
	// with a "line N: …" prefix on parse failures — we surface that to the
	// caller so PLAN.md's "error message includes parse-position info"
	// acceptance is met without re-parsing.
	var doc yaml.Node
	if err := yaml.Unmarshal([]byte(frontmatter), &doc); err != nil {
		return "", fmt.Errorf("frontmatter parse failed: %w", err)
	}

	// Empty document (e.g., frontmatter is whitespace-only): nothing to
	// strip. yaml.Unmarshal of "" or all-whitespace produces a zero-Kind
	// Node; treat as empty result.
	if doc.Kind == 0 || len(doc.Content) == 0 {
		return "", nil
	}

	// Root is a DocumentNode wrapping exactly one child node. The child is
	// the actual top-level YAML value — typically a MappingNode for our
	// frontmatter shape.
	root := doc.Content[0]
	if root.Kind != yaml.MappingNode {
		// Non-mapping top-level (e.g., a list or scalar) has no keys to
		// strip; re-emit verbatim through Marshal so the caller still gets
		// canonicalized output for the strip path.
		return marshalNode(&doc)
	}

	// Build the strip set based on flags. Use a small map for O(1) lookup;
	// the universe is at most four keys so allocation cost is trivial.
	stripKeys := make(map[string]struct{}, 4)
	if stripModel {
		stripKeys[frontmatterModelKey] = struct{}{}
	}
	if stripTools {
		for _, k := range frontmatterToolsKeys {
			stripKeys[k] = struct{}{}
		}
	}

	// Walk root.Content as alternating key/value pairs. Filter pairs whose
	// key is a ScalarNode AND its Value matches one of the strip targets.
	// Preserve all other pairs in original order.
	filtered := make([]*yaml.Node, 0, len(root.Content))
	for i := 0; i+1 < len(root.Content); i += 2 {
		key := root.Content[i]
		val := root.Content[i+1]
		if key.Kind == yaml.ScalarNode {
			if _, drop := stripKeys[key.Value]; drop {
				continue
			}
		}
		filtered = append(filtered, key, val)
	}
	root.Content = filtered

	return marshalNode(&doc)
}

// marshalNode renders a parsed *yaml.Node back to its serialized YAML form.
// Returns the rendered string with no trailing whitespace beyond what
// yaml.Marshal naturally appends (yaml.v3 always emits a trailing newline
// for a non-empty document; the caller can rely on that).
func marshalNode(node *yaml.Node) (string, error) {
	buf, err := yaml.Marshal(node)
	if err != nil {
		return "", fmt.Errorf("frontmatter re-encode failed: %w", err)
	}
	return string(bytes.TrimRight(buf, "\n")) + "\n", nil
}
