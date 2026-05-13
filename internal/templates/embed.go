package templates

import (
	"embed"
	"errors"
	"fmt"

	toml "github.com/pelletier/go-toml/v2"
)

// DefaultTemplateFS embeds the builtin default cascade template TOML files
// AND the placeholder agent .md scaffolding + agents.example.toml runtime-
// config example into the binary. Per droplet 3.14 fix L4 the embed path
// lives under internal/templates/builtin/ — NOT a repo-root templates/
// directory — and the //go:embed directive uses relative paths with no
// parent traversal.
//
// Drop 4c.5 droplet F.2.1 rebadged the original `default.toml` to
// `default-go.toml` so sibling builtins (and, post-Q1 resolution, future
// language-flavored variants) can ship alongside without stepping on the
// Go-flavored content. Drop 4c.5 droplet F.2.2 added the language-agnostic
// `default-generic.toml` sibling. Drop 4c.6 W5.D1 rebadged the Go-flavored
// builtin a second time, from `default-go.toml` to `till-go.toml`, to
// align with the `till-` prefix family the cascade-methodology trilogy
// adopts (per `SKETCH.md` § 3.5.1 / § 21.6 — communicates "shipped from
// Tillsyn binary"). Drop 4c.6 W5.D2 rebadged the language-agnostic
// builtin in lockstep, from `default-generic.toml` to `till-gen.toml`,
// completing the `till-` prefix family. The dual-history records
// (default.toml → default-go.toml → till-go.toml AND
// default-generic.toml → till-gen.toml) preserve every rebadge event so
// future readers can trace each file's lineage. The directive uses an
// EXPLICIT FILE LIST rather than a glob (`builtin/*.toml`) per F.2.1
// falsification mitigation #2 (carried forward to F.2.2 + W5.D1 + W5.D2):
// an explicit list cannot accidentally pick up unrelated .toml fixtures
// or leftover files in builtin/.
//
// Drop 4c.5 droplet F.1.3 added the language-aware resolver
// `LoadDefaultTemplateForLanguage` and reduced `LoadDefaultTemplate` to a
// thin wrapper that selects the language-AGNOSTIC (generic) file. See the
// SEMANTIC SHIFT note on `LoadDefaultTemplate` for the implications for
// existing callers.
//
// Drop 4c.6 W1.D1 extended the directive with placeholder agent .md files
// and the agents.example.toml runtime-config fixture. Per the W1.D1
// acceptance bullet + ContextBlocks `constraint` (high), the directive uses
// an EXPLICIT PER-FILE LIST — never `**/*.md` or `builtin/agents/*` glob —
// carrying forward Drop 4c.5 F.2.1's falsification mitigation #2. The
// agent .md bodies are PLACEHOLDER scaffolding only; substantive prompt
// content lands in Drop 4c.8 W4. The till-gdd group ships placeholder
// shape only at this drop per `SKETCH.md` § 14.2 / § 21.6.
//
// Drop 4c.6 W1.D1 cross-droplet handoff with W0.5: the W0.5 validator
// `validateAgentBindingNames` flips from fail-permissive to fail-strict
// the moment any `builtin/agents/<group>/*.md` file ships into the
// embed.FS (probed at package init via `embeddedAgentLibraryShipped`).
// Drop 4c.6 W5.D3 dropped the `go-` prefix from `till-go.toml`'s
// agent_name values, so the file now references `builder-agent`,
// `planning-agent`, `research-agent`, `qa-proof-agent`,
// `qa-falsification-agent`, `commit-message-agent`, and
// `orchestrator-managed`.
//
// Drop 4c.6.1 W4.D1 restructured the embedded agent group subdirs:
//   - `builtin/agents/till-go/` renamed to `builtin/agents/go/` (canonical
//     group name, no `till-` prefix) via `git mv` (history preserved).
//   - `builtin/agents/till-gen/` renamed to `builtin/agents/gen/` likewise.
//   - 5 legacy `go-*-agent.md` orphans under `go/` removed via `git rm`
//     (they were transitional residue never referenced by till-go.toml).
//   - Monolithic `qa-proof-agent.md` and `qa-falsification-agent.md` in
//     both `go/` and `gen/` split into 4 dedicated files each:
//     `plan-qa-proof-agent.md`, `build-qa-proof-agent.md`,
//     `plan-qa-falsification-agent.md`, `build-qa-falsification-agent.md`.
//   - `orchestrator-managed.md` added to `go/` (was only in `gen/`).
//   - NEW `fe/` group added with 10 placeholder files (same 10 standard
//     names as `go/` and `gen/`).
//   - `till-gdd/` is NOT renamed — it is a template-family identifier,
//     not a group name; its 7 files are unchanged.
//
// All paths below use the post-rename canonical names. The directive
// remains an explicit per-file list per the F.2.1 mitigation pattern.
//
// Canonical specs: workflow/drop_3/PLAN.md droplet 3.14 + main/PLAN.md § 19.3
// + workflow/drop_4c_5/THEME_F_PLAN.md droplets F.2.1 + F.2.2 + F.1.3 +
// workflow/drop_4c_6/PLAN.md droplet 4c.6.W1.D1 +
// workflow/drop_4c_6/SKETCH.md § 4.1 + § 4.2 + § 11.1 + § 14.2 + § 21.6 +
// workflow/drop_4c_6_1/PLAN.md droplet 4c.6.1.W4.D1.
//
//go:embed builtin/till-go.toml builtin/till-gen.toml
//go:embed builtin/agents.example.toml
//go:embed builtin/agents/gen/planning-agent.md
//go:embed builtin/agents/gen/builder-agent.md
//go:embed builtin/agents/gen/plan-qa-proof-agent.md
//go:embed builtin/agents/gen/build-qa-proof-agent.md
//go:embed builtin/agents/gen/plan-qa-falsification-agent.md
//go:embed builtin/agents/gen/build-qa-falsification-agent.md
//go:embed builtin/agents/gen/research-agent.md
//go:embed builtin/agents/gen/closeout-agent.md
//go:embed builtin/agents/gen/commit-message-agent.md
//go:embed builtin/agents/gen/orchestrator-managed.md
//go:embed builtin/agents/go/planning-agent.md
//go:embed builtin/agents/go/builder-agent.md
//go:embed builtin/agents/go/plan-qa-proof-agent.md
//go:embed builtin/agents/go/build-qa-proof-agent.md
//go:embed builtin/agents/go/plan-qa-falsification-agent.md
//go:embed builtin/agents/go/build-qa-falsification-agent.md
//go:embed builtin/agents/go/research-agent.md
//go:embed builtin/agents/go/closeout-agent.md
//go:embed builtin/agents/go/commit-message-agent.md
//go:embed builtin/agents/go/orchestrator-managed.md
//go:embed builtin/agents/fe/planning-agent.md
//go:embed builtin/agents/fe/builder-agent.md
//go:embed builtin/agents/fe/plan-qa-proof-agent.md
//go:embed builtin/agents/fe/build-qa-proof-agent.md
//go:embed builtin/agents/fe/plan-qa-falsification-agent.md
//go:embed builtin/agents/fe/build-qa-falsification-agent.md
//go:embed builtin/agents/fe/research-agent.md
//go:embed builtin/agents/fe/closeout-agent.md
//go:embed builtin/agents/fe/commit-message-agent.md
//go:embed builtin/agents/fe/orchestrator-managed.md
//go:embed builtin/agents/till-gdd/planning-agent.md
//go:embed builtin/agents/till-gdd/builder-agent.md
//go:embed builtin/agents/till-gdd/qa-proof-agent.md
//go:embed builtin/agents/till-gdd/qa-falsification-agent.md
//go:embed builtin/agents/till-gdd/research-agent.md
//go:embed builtin/agents/till-gdd/closeout-agent.md
//go:embed builtin/agents/till-gdd/commit-message-agent.md
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
// embedded at `builtin/till-gen.toml` (rebadged from `default-generic.toml`
// in Drop 4c.6 W5.D2 alongside the W5.D1 + F.2.1 dual-history).
//
// SEMANTIC SHIFT (Drop 4c.5 droplet F.1.3): pre-F.1.3 this function read
// `default-go.toml` directly, so every caller received the Go-flavored
// catalog (12 kinds + 4 child_rules + 6 STEWARD seeds + the full
// agent-bindings + gates + context tables). Post-F.1.3 this function is
// a thin wrapper around `LoadDefaultTemplateForLanguage("")`, which
// resolves to `till-gen.toml`. The generic template ships the same
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
//   - `""`     → loads `builtin/till-gen.toml` (the
//     language-agnostic showcase shipped by F.2.2 as
//     `default-generic.toml` and rebadged by Drop 4c.6 W5.D2 to the
//     `till-` prefix family — 12 kinds + 4 child rules + 6 STEWARD
//     seeds, ZERO `[agent_bindings]`).
//   - `"go"`   → loads `builtin/till-go.toml` (the Go-flavored full
//     catalog rebadged by F.2.1 from `default.toml` and again by Drop
//     4c.6 W5.D1 to the `till-` prefix family — 12 kinds + child rules +
//     STEWARD seeds + agent bindings + gates + context).
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
		path = "builtin/till-gen.toml"
	case "go":
		path = "builtin/till-go.toml"
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

// BuiltinTemplateNames returns the closed list of language-axis names that
// LoadDefaultTemplateForLanguage can resolve to an embedded TOML file. The
// list is kept in stable lexical order so MCP / CLI surfaces enumerate the
// builtins deterministically across processes.
//
// Drop 4c.5 droplet F.3.1: `till.template list_builtin` consumes this list to
// answer the wire surface without walking DefaultTemplateFS. Per F.3.1
// falsification mitigation #3 the values are hard-coded at package scope (NOT
// derived from a runtime fs.WalkDir on DefaultTemplateFS) so future fixture
// files dropped into builtin/ cannot accidentally appear in the wire result.
//
// The function returns a fresh slice on every call so callers cannot mutate
// the package-level source of truth. Pre-MVP the list contains
// "till-gen" + "till-go" only (Drop 4c.6 W5.D1 rebadged the Go-flavored
// builtin from `default-go` to `till-go`; W5.D2 rebadged the
// language-agnostic builtin from `default-generic` to `till-gen`,
// completing the `till-` prefix family). Stable lexical order preserved
// (`till-gen` < `till-go`). The FE template ships post-MVP via the F.4
// marketplace CLI per the Q1 resolution in
// workflow/drop_4c_5/THEME_F_PLAN.md §3 Note 5.
func BuiltinTemplateNames() []string {
	return []string{"till-gen", "till-go"}
}

// MarshalTOML serializes a Template back to canonical TOML bytes via
// pelletier/go-toml/v2's Marshal entry point. The function is the inverse
// of Load — feeding the output bytes back through Load returns an
// equivalent Template (modulo TOML key-order, which the marshaller does
// not promise to preserve across versions).
//
// Drop 4c.5 droplet F.3.1: `till.template get` consumes this helper to
// wire the active per-project Template back to the MCP client as TOML-OUT
// (rather than a JSON envelope of the decoded struct). The Pelletier
// marshaller honors the existing `toml:"…"` struct tags on Template and
// every nested type, so re-marshalling does not require new tags. Pure
// function: no I/O, no globals.
//
// Returns the canonical underlying error from toml.Marshal (e.g. when a
// future Template field grows a non-marshalable type) wrapped with a
// stable prefix so callers can route on `errors.Is` against the
// pelletier sentinel without losing context.
func MarshalTOML(tpl Template) ([]byte, error) {
	encoded, err := toml.Marshal(tpl)
	if err != nil {
		return nil, fmt.Errorf("templates: marshal toml: %w", err)
	}
	return encoded, nil
}
