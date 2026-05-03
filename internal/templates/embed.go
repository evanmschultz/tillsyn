package templates

import "embed"

// DefaultTemplateFS embeds the builtin default cascade template TOML
// (builtin/default.toml) into the binary. Per droplet 3.14 fix L4 the
// embed path lives under internal/templates/builtin/ — NOT a repo-root
// templates/ directory — and the //go:embed directive uses a relative
// "builtin/default.toml" with no parent traversal.
//
// Canonical spec: workflow/drop_3/PLAN.md droplet 3.14 + main/PLAN.md § 19.3.
//
//go:embed builtin/default.toml
var DefaultTemplateFS embed.FS

// LoadDefaultTemplate parses and validates the builtin default template
// embedded at builtin/default.toml. Equivalent to calling Load with a
// reader over the embedded TOML bytes. The returned Template carries the
// closed 12-kind catalog, the four reverse-hierarchy prohibitions
// (encoded as KindRule.AllowedChildKinds enumerations per finding
// 5.B.16 N3), the auto-create [[child_rules]] for build / plan / drop
// structural_type, and agent bindings for every kind.
//
// Returns (Template{}, err) on:
//   - embed.FS open failure (programmer error — the file is compiled in).
//   - any error returned by Load — schema-version mismatch, unknown key,
//     unknown kind reference, child-rule cycle, etc. See load.go's
//     sentinel errors for the closed routing set.
//
// Pure function: no I/O beyond the embed.FS open + TOML parse, no clock
// or random dependency. Safe to call from any goroutine; the embed.FS
// is read-only.
func LoadDefaultTemplate() (Template, error) {
	f, err := DefaultTemplateFS.Open("builtin/default.toml")
	if err != nil {
		return Template{}, err
	}
	defer f.Close()
	return Load(f)
}
