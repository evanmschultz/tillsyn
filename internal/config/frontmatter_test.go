// Tests for the StripFrontmatterKeys helper.
//
// Per workflow/drop_4c_6/DROP_4c.6.W0_AGENTS_TOML_SCHEMA/PLAN.md § Droplet
// 4c.6.W0.D4 — six required cases plus a top-level-only nested-key case
// derived from the PLAN's "constraint (high)" ContextBlock at line 173:
//
//   - StripModel:               removes the model: top-level key.
//   - StripTools:                removes tools:, allowedTools:, disallowedTools:.
//   - BothFalse:                 returns input string verbatim (no parse cycle).
//   - PreservesOtherFields:      name:, description:, etc., survive.
//   - InvalidYAML:               returns an error with a parse-position message.
//   - Idempotent:                strip-then-strip equals strip.
//   - TopLevelOnly:              nested keys named model: / tools: survive.
//
// The both-false short-circuit must NOT round-trip through the YAML encoder;
// gopkg.in/yaml.v3's Marshal explicitly does not preserve original textual
// representation (per `go doc gopkg.in/yaml.v3 Node` — "the content when
// re-encoded will not have its original textual representation preserved").
// TestStripFrontmatterKeys_BothFalse asserts byte-for-byte identity to lock
// this in.
package config

import (
	"errors"
	"strings"
	"testing"
)

// TestStripFrontmatterKeys_StripModel asserts model: removal when stripModel
// is true and stripTools is false. Surrounding fields must survive.
func TestStripFrontmatterKeys_StripModel(t *testing.T) {
	in := "name: foo\ndescription: bar\nmodel: claude-sonnet-4-6\n"
	out, err := StripFrontmatterKeys(in, true, false)
	if err != nil {
		t.Fatalf("StripFrontmatterKeys returned error: %v", err)
	}
	if strings.Contains(out, "model:") {
		t.Errorf("expected model: stripped; got:\n%s", out)
	}
	if !strings.Contains(out, "name: foo") {
		t.Errorf("expected name: foo preserved; got:\n%s", out)
	}
	if !strings.Contains(out, "description: bar") {
		t.Errorf("expected description: bar preserved; got:\n%s", out)
	}
}

// TestStripFrontmatterKeys_StripTools asserts that tools:, allowedTools:, and
// disallowedTools: are all removed together when stripTools is true. Per
// SKETCH.md § 15 the three keys are governed by the same setting because the
// runtime narrows the agent surface to {name, description} once the runtime
// owns the kind→tools mapping.
func TestStripFrontmatterKeys_StripTools(t *testing.T) {
	in := "name: foo\ntools:\n  - Read\n  - Edit\nallowedTools:\n  - Bash\ndisallowedTools:\n  - Write\ndescription: bar\n"
	out, err := StripFrontmatterKeys(in, false, true)
	if err != nil {
		t.Fatalf("StripFrontmatterKeys returned error: %v", err)
	}
	for _, k := range []string{"tools:", "allowedTools:", "disallowedTools:"} {
		if strings.Contains(out, k) {
			t.Errorf("expected %s stripped; got:\n%s", k, out)
		}
	}
	if !strings.Contains(out, "name: foo") || !strings.Contains(out, "description: bar") {
		t.Errorf("expected name + description preserved; got:\n%s", out)
	}
}

// TestStripFrontmatterKeys_BothFalse asserts byte-for-byte identity when
// stripModel and stripTools are both false. The function MUST short-circuit
// before parsing — yaml.v3's Marshal does not preserve original whitespace,
// comments, or quoting style, so any parse-and-reemit cycle would corrupt
// frontmatter that the caller intended to leave alone.
func TestStripFrontmatterKeys_BothFalse(t *testing.T) {
	in := "# leading comment\nname:    foo\ndescription:   bar  # trailing\nmodel: claude\ntools:\n  - Read\n"
	out, err := StripFrontmatterKeys(in, false, false)
	if err != nil {
		t.Fatalf("StripFrontmatterKeys returned error: %v", err)
	}
	if out != in {
		t.Errorf("expected byte-for-byte identity; got:\n%q\nwant:\n%q", out, in)
	}
}

// TestStripFrontmatterKeys_PreservesOtherFields asserts that all non-stripped
// top-level keys survive when both flags are true.
func TestStripFrontmatterKeys_PreservesOtherFields(t *testing.T) {
	in := "name: foo\ndescription: bar\nmodel: claude\ntools:\n  - Read\nmaxBudgetUSD: 5\n"
	out, err := StripFrontmatterKeys(in, true, true)
	if err != nil {
		t.Fatalf("StripFrontmatterKeys returned error: %v", err)
	}
	if !strings.Contains(out, "name: foo") {
		t.Errorf("expected name: foo preserved; got:\n%s", out)
	}
	if !strings.Contains(out, "description: bar") {
		t.Errorf("expected description: bar preserved; got:\n%s", out)
	}
	if !strings.Contains(out, "maxBudgetUSD: 5") {
		t.Errorf("expected maxBudgetUSD: 5 preserved; got:\n%s", out)
	}
	if strings.Contains(out, "model:") || strings.Contains(out, "tools:") {
		t.Errorf("expected model: and tools: stripped; got:\n%s", out)
	}
}

// TestStripFrontmatterKeys_InvalidYAML asserts that malformed YAML surfaces
// a non-nil error with a parse-position-bearing message. The underlying
// yaml.v3 parser returns a *yaml.TypeError or a yaml.errors-package error
// whose Error() string includes "line N"; we assert the caller can see the
// line marker without depending on the exact unexported error type.
func TestStripFrontmatterKeys_InvalidYAML(t *testing.T) {
	// Unbalanced quoting on line 2 — yaml.v3 surfaces a line-bearing error.
	in := "name: foo\ndescription: \"unterminated\nmodel: claude\n"
	_, err := StripFrontmatterKeys(in, true, false)
	if err == nil {
		t.Fatal("expected non-nil error for malformed YAML")
	}
	if !strings.Contains(err.Error(), "line") {
		t.Errorf("expected error message to include parse-position 'line N'; got: %v", err)
	}
}

// TestStripFrontmatterKeys_Idempotent asserts that repeated invocation with
// the same flags returns the same string. Stripping a key the second time is
// a no-op; the function MUST tolerate the absent-key case without erroring.
func TestStripFrontmatterKeys_Idempotent(t *testing.T) {
	in := "name: foo\ndescription: bar\nmodel: claude\ntools:\n  - Read\n"
	once, err := StripFrontmatterKeys(in, true, true)
	if err != nil {
		t.Fatalf("first strip failed: %v", err)
	}
	twice, err := StripFrontmatterKeys(once, true, true)
	if err != nil {
		t.Fatalf("second strip failed: %v", err)
	}
	if once != twice {
		t.Errorf("expected idempotency; first:\n%s\nsecond:\n%s", once, twice)
	}
}

// TestStripFrontmatterKeys_TopLevelOnly asserts that nested keys named
// model: or tools: survive — only the top-level YAML mapping keys are
// candidates for removal. Per PLAN.md `constraint (high)` at line 173:
// "top-level YAML keys only; nested model: / tools: keys survive."
func TestStripFrontmatterKeys_TopLevelOnly(t *testing.T) {
	// metadata.model and metadata.tools are nested one level under the root
	// mapping — stripping must not reach them.
	in := "name: foo\nmetadata:\n  model: nested-keep-me\n  tools:\n    - nested-keep\nmodel: top-strip\ntools:\n  - top-strip\n"
	out, err := StripFrontmatterKeys(in, true, true)
	if err != nil {
		t.Fatalf("StripFrontmatterKeys returned error: %v", err)
	}
	if !strings.Contains(out, "nested-keep-me") {
		t.Errorf("expected nested model value preserved; got:\n%s", out)
	}
	if !strings.Contains(out, "nested-keep") {
		t.Errorf("expected nested tools value preserved; got:\n%s", out)
	}
	if strings.Contains(out, "top-strip") {
		t.Errorf("expected top-level model:/tools: stripped; got:\n%s", out)
	}
}

// TestStripFrontmatterKeys_EmptyInput asserts that an empty input string
// passes through cleanly (no error) regardless of flag combination. Both-false
// preserves verbatim; either-true short-circuits via "nothing to strip" when
// no top-level keys exist.
func TestStripFrontmatterKeys_EmptyInput(t *testing.T) {
	cases := []struct {
		name              string
		stripModel, tools bool
	}{
		{"both false", false, false},
		{"strip model only", true, false},
		{"strip tools only", false, true},
		{"both true", true, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out, err := StripFrontmatterKeys("", tc.stripModel, tc.tools)
			if err != nil {
				t.Fatalf("expected nil error on empty input; got: %v", err)
			}
			if tc.stripModel || tc.tools {
				// Either-true path may parse and re-emit; an empty input
				// should still result in a stripped (likely empty) string.
				if strings.Contains(out, "model:") || strings.Contains(out, "tools:") {
					t.Errorf("expected no keys present; got:\n%q", out)
				}
				return
			}
			// Both-false short-circuits to verbatim identity.
			if out != "" {
				t.Errorf("expected verbatim empty; got: %q", out)
			}
		})
	}
}

// TestStripFrontmatterKeys_StripModelKeepsTools asserts that stripping model:
// alone leaves tools:, allowedTools:, disallowedTools: untouched.
func TestStripFrontmatterKeys_StripModelKeepsTools(t *testing.T) {
	in := "name: foo\nmodel: claude\ntools:\n  - Read\nallowedTools:\n  - Edit\n"
	out, err := StripFrontmatterKeys(in, true, false)
	if err != nil {
		t.Fatalf("StripFrontmatterKeys returned error: %v", err)
	}
	if strings.Contains(out, "model:") {
		t.Errorf("expected model: stripped; got:\n%s", out)
	}
	if !strings.Contains(out, "tools:") {
		t.Errorf("expected tools: preserved; got:\n%s", out)
	}
	if !strings.Contains(out, "allowedTools:") {
		t.Errorf("expected allowedTools: preserved; got:\n%s", out)
	}
}

// TestStripFrontmatterKeys_StripToolsKeepsModel asserts that stripping tools:
// alone leaves model: untouched.
func TestStripFrontmatterKeys_StripToolsKeepsModel(t *testing.T) {
	in := "name: foo\nmodel: claude\ntools:\n  - Read\n"
	out, err := StripFrontmatterKeys(in, false, true)
	if err != nil {
		t.Fatalf("StripFrontmatterKeys returned error: %v", err)
	}
	if !strings.Contains(out, "model:") {
		t.Errorf("expected model: preserved; got:\n%s", out)
	}
	if strings.Contains(out, "tools:") {
		t.Errorf("expected tools: stripped; got:\n%s", out)
	}
}

// TestStripFrontmatterKeys_InvalidYAMLReturnsNonNilErr is a defensive guard
// that yaml.v3's parse error stays in scope of the standard library's
// errors-package contract — i.e. the returned error is never nil under the
// malformed-input path. Belt-and-suspenders against future yaml.v3 upgrades
// that might silently lenient-parse.
//
// The input below uses a tab character at the start of the value position,
// which yaml.v3 rejects with a "found character that cannot start any token"
// error (tabs are not valid YAML indentation). Pure-space indent issues are
// often coerced by yaml.v3 into multi-line scalars and don't trigger errors.
func TestStripFrontmatterKeys_InvalidYAMLReturnsNonNilErr(t *testing.T) {
	_, err := StripFrontmatterKeys("name: foo\nlist:\n\t- bad-tab\n", true, true)
	if err == nil {
		t.Fatal("expected non-nil error on tab-indented YAML")
	}
	// errors.Unwrap chain is allowed to be empty (yaml.v3 returns a sentinel-
	// like top-level error without wrapping); we only require non-nil.
	if errors.Is(err, nil) {
		t.Fatal("errors.Is(err, nil) should never be true for a non-nil error")
	}
}
