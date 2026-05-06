package templates

import (
	"embed"
	"errors"
	"fmt"
)

// DefaultTemplateFS embeds the builtin default cascade template TOML files
// into the binary. Per droplet 3.14 fix L4 the embed path lives under
// internal/templates/builtin/ — NOT a repo-root templates/ directory — and
// the //go:embed directive uses relative paths with no parent traversal.
//
// Drop 4c.5 droplet F.2.1 rebadged the original `default.toml` to
// `default-go.toml` so sibling builtins (and, post-Q1 resolution, future
// language-flavored variants) can ship alongside without stepping on the
// Go-flavored content. Drop 4c.5 droplet F.2.2 added the language-agnostic
// `default-generic.toml` sibling. The directive uses an EXPLICIT FILE LIST
// rather than a glob (`builtin/*.toml`) per F.2.1 falsification mitigation
// #2 (carried forward to F.2.2): an explicit list cannot accidentally pick
// up unrelated .toml fixtures or leftover files in builtin/.
//
// Drop 4c.5 droplet F.1.3 added the language-aware resolver
// `LoadDefaultTemplateForLanguage` and reduced `LoadDefaultTemplate` to a
// thin wrapper that selects the language-AGNOSTIC (generic) file. See the
// SEMANTIC SHIFT note on `LoadDefaultTemplate` for the implications for
// existing callers.
//
// Canonical specs: workflow/drop_3/PLAN.md droplet 3.14 + main/PLAN.md § 19.3
// + workflow/drop_4c_5/THEME_F_PLAN.md droplets F.2.1 + F.2.2 + F.1.3.
//
//go:embed builtin/default-go.toml builtin/default-generic.toml
var DefaultTemplateFS embed.FS

// ErrLanguageNotSupported is the closed sentinel returned by
// `LoadDefaultTemplateForLanguage` when the caller-supplied language axis
// is recognized as a not-yet-shipped value (currently `"fe"` per the Q1
// resolution in workflow/drop_4c_5/THEME_F_PLAN.md §3 Note 5) OR is
// outside the closed `domain.Project.Language` enum entirely (e.g. a
// hand-rolled `"rust"`).
//
// Routing contract: callers programmatically distinguish "no template for
// this language" from a TOML parse error or schema-version mismatch via
// `errors.Is(err, ErrLanguageNotSupported)`. The wrapped error always
// includes the offending language string verbatim so dev surfaces (CLI,
// MCP error envelopes) can name the input that failed.
//
// Closed-enum drift guard: when a future drop extends
// `domain.Project.Language` (e.g. landing FE adopter support), the new
// language MUST also be wired into `LoadDefaultTemplateForLanguage`'s
// switch AND ship a matching `builtin/default-<lang>.toml` file in the
// embed.FS. The sentinel exists precisely so the resolver fails LOUD on
// the gap rather than silently returning the Go default.
var ErrLanguageNotSupported = errors.New("template language not supported")

// LoadDefaultTemplate parses and validates the language-AGNOSTIC builtin
// embedded at `builtin/default-generic.toml`.
//
// SEMANTIC SHIFT (Drop 4c.5 droplet F.1.3): pre-F.1.3 this function read
// `default-go.toml` directly, so every caller received the Go-flavored
// catalog (12 kinds + 4 child_rules + 6 STEWARD seeds + the full
// agent-bindings + gates + context tables). Post-F.1.3 this function is
// a thin wrapper around `LoadDefaultTemplateForLanguage("")`, which
// resolves to `default-generic.toml`. The generic template ships the same
// 12 kinds + 4 child_rules + 6 STEWARD seeds BUT INTENTIONALLY OMITS
// `[agent_bindings]` entirely (per F.2.2 acceptance criterion #2).
// Adopters declare bindings in their project-local
// `<project_root>/.tillsyn/template.toml`.
//
// Existing pre-F.1.3 callers (`seedStewardAnchors` at
// `internal/app/auto_generate_steward.go:44` and the Drop-3.14 stub
// `loadProjectTemplate` in `internal/app/service.go`) WILL inherit this
// shift. Drop 4c.5 droplet F.2.4 (later in Theme F's chain) audits each
// caller and redirects to the language-explicit form
// `LoadDefaultTemplateForLanguage(project.Language)` so language-aware
// behavior lands at the correct seam. Until F.2.4 lands, callers using
// the unsuffixed `LoadDefaultTemplate()` get the GENERIC catalog — which
// for `seedStewardAnchors` happens to materialize the same 6 STEWARD
// seeds (the seed set is identical across both builtins per F.2.2
// criterion #5), but for a future drop that depends on agent_bindings
// the change would matter.
//
// Pure function: no I/O beyond the embed.FS open + TOML parse, no clock
// or random dependency. Safe to call from any goroutine; the embed.FS
// is read-only.
//
// Returns (Template{}, err) on:
//   - embed.FS open failure (programmer error — the file is compiled in).
//   - any error returned by Load — schema-version mismatch, unknown key,
//     unknown kind reference, child-rule cycle, etc. See load.go's
//     sentinel errors for the closed routing set.
func LoadDefaultTemplate() (Template, error) {
	return LoadDefaultTemplateForLanguage("")
}

// LoadDefaultTemplateForLanguage parses + validates the embedded builtin
// template that matches the supplied project-language axis. The accepted
// closed enum mirrors `domain.Project.Language` (see
// `internal/domain/project.go` `isValidProjectLanguage`):
//
//   - `""`     → loads `builtin/default-generic.toml` (the
//     language-agnostic showcase shipped by F.2.2 — 12 kinds + 4 child
//     rules + 6 STEWARD seeds, ZERO `[agent_bindings]`).
//   - `"go"`   → loads `builtin/default-go.toml` (the Go-flavored full
//     catalog rebadged by F.2.1 — 12 kinds + child rules + STEWARD
//     seeds + agent bindings + gates + context).
//   - `"fe"`   → returns an error wrapping `ErrLanguageNotSupported`
//     per Q1 resolution (workflow/drop_4c_5/THEME_F_PLAN.md §3 Note 5).
//     FE template ships post-MVP via the F.4 marketplace CLI once an
//     FE adopter materializes; until then dev-FE-projects must author
//     `<project_root>/.tillsyn/template.toml` directly.
//   - Anything else → returns an error wrapping
//     `ErrLanguageNotSupported` with the offending value verbatim.
//
// Closed-enum drift contract: `domain.Project.Language` and this
// resolver must extend together. A future drop that adds (e.g.) `"rust"`
// to the domain validator MUST also wire the new value into the switch
// below AND ship `builtin/default-rust.toml` (or be willing to surface
// `ErrLanguageNotSupported` for the new lang until a builtin lands).
//
// Pure function: no I/O beyond the embed.FS open + TOML parse, no
// global mutation. Safe to call from any goroutine.
//
// Returns (Template{}, err) on:
//   - `lang == "fe"` (deferred per Q1).
//   - `lang` outside the closed enum.
//   - embed.FS open failure (programmer error — files compiled in).
//   - any error returned by Load — schema-version mismatch, unknown
//     key, unknown kind reference, child-rule cycle, etc.
func LoadDefaultTemplateForLanguage(lang string) (Template, error) {
	var path string
	switch lang {
	case "":
		path = "builtin/default-generic.toml"
	case "go":
		path = "builtin/default-go.toml"
	case "fe":
		// Deferred per workflow/drop_4c_5/THEME_F_PLAN.md §3 Note 5
		// (Q1 resolution). Surface a clear, lang-tagged error so
		// project-create boundaries can route the dev to author a
		// project-local template.
		return Template{}, fmt.Errorf("language %q: fe template unavailable; defer until FE adopter materializes: %w", lang, ErrLanguageNotSupported)
	default:
		return Template{}, fmt.Errorf("language %q: outside closed Project.Language enum: %w", lang, ErrLanguageNotSupported)
	}

	f, err := DefaultTemplateFS.Open(path)
	if err != nil {
		// embed.FS open failure is a programmer-error path — the file
		// is compiled into the binary by the //go:embed directive
		// above. Surfaced rather than panicked so callers in
		// release-mode builds can route via toolResultFromError.
		return Template{}, fmt.Errorf("open embedded %q: %w", path, err)
	}
	defer f.Close()
	return Load(f)
}
