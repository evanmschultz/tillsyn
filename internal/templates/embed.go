package templates

import "embed"

// DefaultTemplateFS embeds the builtin default cascade template TOML files
// into the binary. Per droplet 3.14 fix L4 the embed path lives under
// internal/templates/builtin/ — NOT a repo-root templates/ directory — and
// the //go:embed directive uses relative paths with no parent traversal.
//
// Drop 4c.5 droplet F.2.1 rebadged the original `default.toml` to
// `default-go.toml` so a sibling `default-generic.toml` (and, post-Q1
// resolution, future language-flavored variants) can ship alongside without
// stepping on the Go-flavored content. The directive uses an EXPLICIT FILE
// LIST rather than a glob (`builtin/*.toml`) per F.2.1 falsification
// mitigation #2: an explicit list cannot accidentally pick up unrelated
// .toml fixtures or leftover files in builtin/.
//
// Subsequent Theme F droplets extend the list:
//   - F.2.2 adds `builtin/default-generic.toml`.
//   - F.1.3 adds the language-aware resolver `LoadDefaultTemplateForLanguage`
//     and reduces `LoadDefaultTemplate` to a thin wrapper.
//
// Canonical specs: workflow/drop_3/PLAN.md droplet 3.14 + main/PLAN.md § 19.3
// + workflow/drop_4c_5/THEME_F_PLAN.md droplet F.2.1.
//
//go:embed builtin/default-go.toml
var DefaultTemplateFS embed.FS

// LoadDefaultTemplate parses and validates the builtin Go-flavored default
// template embedded at builtin/default-go.toml. Equivalent to calling Load
// with a reader over the embedded TOML bytes. The returned Template carries
// the closed 12-kind catalog, the four reverse-hierarchy prohibitions
// (encoded as KindRule.AllowedChildKinds enumerations per finding 5.B.16 N3),
// the auto-create [[child_rules]] for build / plan / drop structural_type,
// and agent bindings for every kind.
//
// Pre-F.1.3 contract: this function reads `default-go.toml` directly. Drop
// 4c.5 droplet F.1.3 generalizes this to a language-aware resolver
// (`LoadDefaultTemplateForLanguage`) and reduces `LoadDefaultTemplate` to a
// thin wrapper that selects the generic template. Until F.1.3 lands, every
// existing caller of `LoadDefaultTemplate` (the embedded fallback path in
// `internal/app/service.go`'s `loadProjectTemplate` + `seedStewardAnchors`
// at `internal/app/auto_generate_steward.go:44`) continues to receive the
// Go-flavored template — the prior behavior is preserved byte-for-byte.
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
	f, err := DefaultTemplateFS.Open("builtin/default-go.toml")
	if err != nil {
		return Template{}, err
	}
	defer f.Close()
	return Load(f)
}
