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
//go:embed builtin/till-go.toml builtin/till-gen.toml builtin/till-fe.toml
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

var ErrLanguageNotSupported = errors.New("template language not supported")

// ErrBuiltinNotFound is the closed sentinel returned by LoadBuiltinTemplate
// when the caller-supplied name is outside the closed builtin-name list
// returned by BuiltinTemplateNames (currently "till-fe", "till-gen",
// "till-go"). Callers programmatically distinguish "no builtin by that name"
// from a TOML parse error or embed.FS open failure via
// `errors.Is(err, ErrBuiltinNotFound)`. The wrapped error always includes
// the offending name verbatim so CLI / MCP surfaces can name the input.
//
// Closed-list drift guard: when a future drop ships a new builtin (e.g.
// `till-rust`), the new name MUST be wired into LoadBuiltinTemplate's switch
// AND shipped as a matching `builtin/till-<lang>.toml` in the embed.FS AND
// returned by BuiltinTemplateNames. The sentinel exists so unknown names
// fail loud rather than silently falling through.
var ErrBuiltinNotFound = errors.New("builtin template not found")

// BuiltinTemplateNames returns the closed list of builtin template names that
// LoadBuiltinTemplate can resolve to an embedded TOML file. The list is kept
// in stable lexical order so MCP / CLI surfaces enumerate the builtins
// deterministically across processes.
//
// Drop 4c.5 droplet F.3.1: `till.template list_builtin` consumes this list to
// answer the wire surface without walking DefaultTemplateFS. Per F.3.1
// falsification mitigation #3 the values are hard-coded at package scope (NOT
// derived from a runtime fs.WalkDir on DefaultTemplateFS) so future fixture
// files dropped into builtin/ cannot accidentally appear in the wire result.
//
// The function returns a fresh slice on every call so callers cannot mutate
// the package-level source of truth. Drop 4c.6.1 W4.D2 adds "till-fe" to the
// list alongside "till-gen" + "till-go": the Q1 deferral of the FE template
// (workflow/drop_4c_5/THEME_F_PLAN.md §3 Note 5) is resolved now that the FE
// agent scaffold + till-fe.toml ship together. Stable lexical order preserved
// (`till-fe` < `till-gen` < `till-go`).
func BuiltinTemplateNames() []string {
	return []string{"till-fe", "till-gen", "till-go"}
}

// LoadBuiltinTemplate resolves a builtin template by its canonical name
// (the name-axis identifier, not the language-axis value) and returns the
// parsed + validated Template. The accepted closed list mirrors
// BuiltinTemplateNames() exactly:
//
//   - "till-go"  → loads builtin/till-go.toml (Go-flavored full catalog).
//   - "till-gen" → loads builtin/till-gen.toml (language-agnostic generic).
//   - "till-fe"  → loads builtin/till-fe.toml (FE / Wails catalog, shipped
//     in Drop 4c.6.1 W4.D2).
//
// For any other name, returns (Template{}, err) wrapping ErrBuiltinNotFound
// with the offending name in the message, e.g.:
//
//	LoadBuiltinTemplate("rust"): builtin template not found
//
// ErrBuiltinNotFound is the sentinel for unknown names; callers distinguish
// "no builtin by that name" from a TOML parse error via errors.Is.
//
// Closed-list drift guard: when BuiltinTemplateNames() is extended (e.g. by
// a future "till-rust" entry), the switch below and the //go:embed directive
// on DefaultTemplateFS must extend in the same drop. The sentinel exists so
// unknown names fail loud rather than silently returning a default.
//
// Pure function: no I/O beyond the embed.FS open + TOML parse, no global
// mutation. Safe to call from any goroutine.
func LoadBuiltinTemplate(name string) (Template, error) {
	var path string
	switch name {
	case "till-go":
		path = "builtin/till-go.toml"
	case "till-gen":
		path = "builtin/till-gen.toml"
	case "till-fe":
		path = "builtin/till-fe.toml"
	default:
		return Template{}, fmt.Errorf("LoadBuiltinTemplate(%q): %w", name, ErrBuiltinNotFound)
	}

	f, err := DefaultTemplateFS.Open(path)
	if err != nil {
		return Template{}, fmt.Errorf("open embedded %q: %w", path, err)
	}
	defer f.Close()
	return Load(f)
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
