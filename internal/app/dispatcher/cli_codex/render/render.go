// Package render owns the per-spawn bundle render for the codex adapter.
// Drop 4d D4 ships the codex-side mirror of cli_claude/render: writes the
// cross-CLI system-prompt at <bundle.Root>/system-prompt.md PLUS the
// codex-specific <bundle.Root>/codex-config.toml carrying model + effort
// + MCP-server block + tool allow/deny entries.
//
// The function Render is the only exported surface; sub-renderers are
// package-private so the contract stays narrow. Render is called by the
// dispatcher's BuildSpawnCommand AFTER F.7.1's WriteManifest and BEFORE
// the codex adapter's BuildCommand argv assembly.
//
// Import-cycle phasing mirrors cli_claude/render: this package imports
// dispatcher (for Bundle / BindingResolved / domain types) and registers
// itself via dispatcher.RegisterBundleRenderFunc at init() time. The
// dispatcher cannot import render directly (cycle); init.go bridges the
// any-typed grants-lister parameter at the seam.
//
// CLIKind defense (AC#4): Render returns ErrUnsupportedBinding for any
// binding whose CLIKind != CLIKindCodex. The single-global registration
// seam (RegisterBundleRenderFunc) is last-writer-wins today; the defense
// check ensures a misrouted spawn fails loud at render time rather than
// emitting a codex-shaped bundle for a claude binding.
//
// TODO(post-D4): the single-global RegisterBundleRenderFunc seam will need
// to become CLIKind-keyed once cmd/till side-effect-imports both
// cli_claude/render and cli_codex/render at process start. Today only
// claude's render is wired in production; D4's package ships the codex
// implementation + its registration but the wiring at cmd/till is the
// next drop's concern (so production stays single-registrant).
package render

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/evanmschultz/tillsyn/internal/app/dispatcher"
	"github.com/evanmschultz/tillsyn/internal/domain"
)

// codexConfigFilename is the conventional basename of the codex-side
// per-spawn config TOML the render layer writes alongside system-prompt.md.
// Codex consumes per-spawn config via `-c key=value` argv flags (D2's
// concern); this file is the human-readable record of what the spawn
// receives plus the source-of-truth the per-spawn -c argv injector reads
// from in a future droplet that wires file-driven argv expansion.
const codexConfigFilename = "codex-config.toml"

// tillsynMCPServerName is the registered MCP server name codex must use
// in its `[mcp_servers.<name>]` config block. Matches the convention
// documented in the D2 action-item description AND the project's
// `.mcp.json` for parity with claude's MCP wiring.
const tillsynMCPServerName = "tillsyn-dev"

// ErrInvalidRenderInput is returned by Render when one of the supplied
// values fails an obvious shape check before any disk I/O. Callers
// detect via errors.Is and route to ErrInvalidSpawnInput-equivalent
// failure.
var ErrInvalidRenderInput = errors.New("render: invalid render input")

// ErrInvalidGrantsLister is returned by the dispatcher-seam adapter
// (init.go's adaptRender) when the `any`-typed grantsLister supplied by
// BuildSpawnCommand is non-nil but does NOT satisfy the codex render's
// PermissionGrantsLister interface. Mirrors cli_claude/render's
// equivalent sentinel so callers see a clean failure at the seam rather
// than a downstream panic.
var ErrInvalidGrantsLister = errors.New("render: grants lister does not implement PermissionGrantsLister")

// ErrUnsupportedBinding is returned by Render when binding.CLIKind is
// not dispatcher.CLIKindCodex. The single-global RegisterBundleRenderFunc
// seam routes every spawn to one registered hook (last-writer-wins); this
// defense check ensures a misrouted spawn fails loud at render time
// rather than emitting a codex-shaped bundle for a claude binding.
//
// Callers detect via errors.Is(err, ErrUnsupportedBinding) and route to
// the standard spawn-failure disposition.
var ErrUnsupportedBinding = errors.New("render: binding CLIKind is not codex")

// PermissionGrantsLister is the narrower read-only view of
// app.PermissionGrantsStore the codex render package consumes. Mirrors
// the cli_claude/render interface verbatim so the production storage
// adapter satisfies both without an adapter shim.
//
// Structural typing means callers wire the full app.PermissionGrantsStore
// and Go satisfies this narrower interface implicitly — render does not
// import the app package, breaking the otherwise-unavoidable cycle.
//
// A nil lister is the documented graceful-skip path: Render proceeds with
// binding.ToolsAllowed only and emits no error. Today the codex render
// layer does not actively merge grants into the rendered config (D2's
// per-tool approval_mode block is the gating injection); the lister
// argument is plumbed for future symmetry with cli_claude/render and
// for the seam's any-typed contract.
type PermissionGrantsLister interface {
	// ListGrantsForKind returns every grant matching the supplied
	// (projectID, kind, cliKind) triple. Matches the signature on
	// app.PermissionGrantsStore.ListGrantsForKind verbatim.
	ListGrantsForKind(ctx context.Context, projectID string, kind domain.Kind, cliKind string) ([]domain.PermissionGrant, error)
}

// Render writes the per-spawn bundle artifacts the codex adapter needs:
//
//  1. <bundle.Root>/system-prompt.md (cross-CLI; full action-item context)
//  2. <bundle.Root>/codex-config.toml (codex-specific: model, effort, MCP
//     servers, tool allow/deny)
//
// Render is invoked by dispatcher.BuildSpawnCommand AFTER F.7.1's
// WriteManifest succeeds. The caller is responsible for cleaning up the
// bundle root on Render failure — Render itself only rolls back what it
// created (system-prompt.md + codex-config.toml) so a failed render
// leaves manifest.json intact for orphan-scan correlation.
//
// Returns the rendered system-prompt body alongside the error so callers
// can populate dispatcher.SpawnDescriptor.Prompt without re-reading from
// disk. On error the returned body is the empty string.
//
// Files are written with 0o600 perms to match the cli_claude/render
// convention — the bundle directory is per-spawn and tooling-private and
// may carry action-item structural data the dev does not want broadcast.
//
// Validation is intentionally minimal: bundle.Paths.Root must be non-empty,
// binding.AgentName must be non-empty + free of path separators, and
// binding.CLIKind must be CLIKindCodex. The CLIKind check is the
// belt-and-suspenders defense documented on ErrUnsupportedBinding.
//
// ctx is forwarded to grantsLister.ListGrantsForKind for future symmetry
// with cli_claude/render's grants-merge path; today the codex render layer
// does not invoke the lister (D2 owns the per-tool approval_mode injection
// the lister would feed into).
func Render(
	ctx context.Context,
	bundle dispatcher.Bundle,
	item domain.ActionItem,
	project domain.Project,
	binding dispatcher.BindingResolved,
	grantsLister PermissionGrantsLister,
) (string, error) {
	_ = ctx
	_ = grantsLister

	if strings.TrimSpace(bundle.Paths.Root) == "" {
		return "", fmt.Errorf("%w: bundle.Paths.Root is empty", ErrInvalidRenderInput)
	}
	if strings.TrimSpace(binding.AgentName) == "" {
		return "", fmt.Errorf("%w: binding.AgentName is empty", ErrInvalidRenderInput)
	}
	if strings.ContainsAny(binding.AgentName, `/\`) {
		return "", fmt.Errorf("%w: binding.AgentName %q contains path separator",
			ErrInvalidRenderInput, binding.AgentName)
	}
	if binding.CLIKind != dispatcher.CLIKindCodex {
		return "", fmt.Errorf("%w: got %q, want %q",
			ErrUnsupportedBinding, string(binding.CLIKind), string(dispatcher.CLIKindCodex))
	}

	rollback := newRenderRollback(bundle.Paths.Root)

	// 1. system-prompt.md at bundle root (cross-CLI; same body shape as
	// cli_claude/render so cross-CLI dispatch produces identical prompts).
	promptBody, err := renderSystemPrompt(bundle, item, project)
	if err != nil {
		rollback.run()
		return "", fmt.Errorf("render: system prompt: %w", err)
	}

	// 2. codex-config.toml at bundle root (codex-specific).
	if err := renderCodexConfig(bundle, binding); err != nil {
		rollback.run()
		return "", fmt.Errorf("render: codex config: %w", err)
	}

	return promptBody, nil
}

// renderRollback captures the paths created by Render so a partial render
// can clean itself up on failure. The struct deliberately tracks the
// bundle root + the two known artifact filenames rather than every
// individual write — Render is the sole writer of system-prompt.md and
// codex-config.toml under <Root>, so a failed render can blanket-remove
// those two paths without touching F.7.1's manifest.json.
type renderRollback struct {
	bundleRoot string
}

// newRenderRollback returns a rollback handle bound to bundleRoot.
func newRenderRollback(bundleRoot string) renderRollback {
	return renderRollback{bundleRoot: bundleRoot}
}

// run removes every path Render writes. Best-effort: cleanup errors are
// swallowed because the caller is already returning a non-nil error and
// the F.7.8 orphan scan will reap any straggler files via the bundle's
// manifest.
func (r renderRollback) run() {
	if r.bundleRoot == "" {
		return
	}
	_ = os.Remove(filepath.Join(r.bundleRoot, "system-prompt.md"))
	_ = os.Remove(filepath.Join(r.bundleRoot, codexConfigFilename))
}

// renderSystemPrompt writes <bundle.Root>/system-prompt.md with the full
// action-item context the spawned agent needs. The body shape mirrors
// cli_claude/render's assembleSystemPromptBody verbatim so cross-CLI
// dispatch produces identical prompts — the spawned subagent's lifecycle
// directives, structural fields, and move-state contract are CLI-agnostic
// by design.
//
// Returns the rendered body so the caller can mirror it into
// dispatcher.SpawnDescriptor.Prompt without a second disk read.
func renderSystemPrompt(bundle dispatcher.Bundle, item domain.ActionItem, project domain.Project) (string, error) {
	body := assembleSystemPromptBody(item, project)
	if err := os.WriteFile(bundle.Paths.SystemPromptPath, []byte(body), 0o600); err != nil {
		return "", err
	}
	return body, nil
}

// assembleSystemPromptBody builds the system-prompt.md body. Pure
// function so tests can pin the exact text without touching the
// filesystem. Mirrors cli_claude/render.assembleSystemPromptBody —
// keep the two in sync if either changes.
//
// Body fields:
//
//   - task_id, project_id, project_dir, kind, title — every spawn carries
//     these structural fields. The agent uses task_id to route Tillsyn
//     state moves; project_dir is the cd-target.
//   - paths, packages — emitted only when the action item declares them.
//   - move-state directive — every spawn instructs the agent to take
//     ownership of its lifecycle transitions.
func assembleSystemPromptBody(item domain.ActionItem, project domain.Project) string {
	var b strings.Builder
	b.WriteString("task_id: ")
	b.WriteString(item.ID)
	b.WriteString("\n")
	b.WriteString("project_id: ")
	b.WriteString(project.ID)
	b.WriteString("\n")
	b.WriteString("project_dir: ")
	b.WriteString(project.RepoPrimaryWorktree)
	b.WriteString("\n")
	b.WriteString("kind: ")
	b.WriteString(string(item.Kind))
	b.WriteString("\n")
	if item.Title != "" {
		b.WriteString("title: ")
		b.WriteString(item.Title)
		b.WriteString("\n")
	}
	if len(item.Paths) > 0 {
		b.WriteString("paths: ")
		b.WriteString(strings.Join(item.Paths, ", "))
		b.WriteString("\n")
	}
	if len(item.Packages) > 0 {
		b.WriteString("packages: ")
		b.WriteString(strings.Join(item.Packages, ", "))
		b.WriteString("\n")
	}
	b.WriteString("move-state directive: Move the action item to in_progress on start. ")
	b.WriteString("On success set metadata.outcome=\"success\" and move to complete. ")
	b.WriteString("On blocking findings record them in metadata + a closing comment and return.\n")
	return b.String()
}

// renderCodexConfig writes <bundle.Root>/codex-config.toml with the
// codex-specific per-spawn settings: model, effort, MCP server block,
// and tool allow/deny lists.
//
// Today this file is the human-readable record of what the spawn
// receives; D2's argv builder injects the same values via per-spawn `-c
// key=value` flags. A future droplet may swap to file-driven `-c
// @<path>` expansion (if codex supports it) so this file becomes the
// source-of-truth instead of a parallel record. Either way, the TOML
// shape here matches what D2 emits at the argv layer for parity.
//
// Tool-name translation (D2.5) is NOT yet shipped. Render writes
// binding.ToolsAllowed / ToolsDisallowed verbatim; once D2.5 lands its
// TranslateToolName helper, this function will route names through it
// before emit. The TODO is intentional and called out at the call site.
func renderCodexConfig(bundle dispatcher.Bundle, binding dispatcher.BindingResolved) error {
	body := assembleCodexConfigBody(binding)
	configPath := filepath.Join(bundle.Paths.Root, codexConfigFilename)
	return os.WriteFile(configPath, []byte(body), 0o600)
}

// assembleCodexConfigBody renders the codex-config.toml body. Pure
// function so tests can pin the exact text without touching the
// filesystem.
//
// TOML shape (matches D2's argv-injection convention):
//
//	# model + reasoning effort
//	model = "<binding.Model>"          # omitted when nil
//	model_reasoning_effort = "<binding.Effort>"  # omitted when nil
//
//	# MCP server registration
//	[mcp_servers.tillsyn-dev]
//	command = "till"
//	args = ["mcp"]
//
//	# Tool allow/deny (D2.5 will translate names to codex-canonical form)
//	allow_tools = ["Read", "Grep", ...]
//	deny_tools = ["WebFetch", ...]
//
// Empty/nil slices render the array key with an empty array literal
// (`[]`) so the file is self-documenting — a dev opening the file sees
// the key even when no values are bound. This mirrors cli_claude/render's
// explicit-empty convention on settings.json's permissions block.
func assembleCodexConfigBody(binding dispatcher.BindingResolved) string {
	var b strings.Builder

	b.WriteString("# tillsyn-rendered codex per-spawn config\n")
	b.WriteString("# agent: ")
	b.WriteString(binding.AgentName)
	b.WriteString("\n\n")

	// Model + effort. Pointer-typed fields per F.7.17 L9: emit-only-on-non-nil.
	if binding.Model != nil && *binding.Model != "" {
		b.WriteString("model = ")
		b.WriteString(tomlString(*binding.Model))
		b.WriteString("\n")
	}
	if binding.Effort != nil && *binding.Effort != "" {
		b.WriteString("model_reasoning_effort = ")
		b.WriteString(tomlString(*binding.Effort))
		b.WriteString("\n")
	}
	b.WriteString("\n")

	// MCP server block. Tillsyn registers itself as a stdio MCP child of
	// codex so the spawned agent can call till.* MCP methods (auth,
	// action_item, etc.) against the same orchestrator-managed Tillsyn
	// process. Server name matches the project's `.mcp.json` for parity
	// with claude's wiring.
	b.WriteString("[mcp_servers.")
	b.WriteString(tillsynMCPServerName)
	b.WriteString("]\n")
	b.WriteString(`command = "till"`)
	b.WriteString("\n")
	b.WriteString(`args = ["mcp"]`)
	b.WriteString("\n\n")

	// Tool allow/deny. TODO(D2.5): route names through TranslateToolName
	// once that helper ships; today we emit binding values verbatim.
	b.WriteString("allow_tools = ")
	b.WriteString(tomlStringArray(binding.ToolsAllowed))
	b.WriteString("\n")
	b.WriteString("deny_tools = ")
	b.WriteString(tomlStringArray(binding.ToolsDisallowed))
	b.WriteString("\n")

	return b.String()
}

// tomlString returns a TOML basic-string literal for s. Quotes are
// escaped per the TOML spec; backslashes are escaped first so a value
// containing `\"` round-trips correctly. The output includes the
// surrounding double quotes.
//
// This intentionally does NOT pull in github.com/pelletier/go-toml/v2
// for one-shot scalar emission — the render package's TOML output is a
// small fixed shape and a 6-line helper keeps the dependency surface
// flat. If the shape grows (nested tables, inline arrays of tables,
// etc.) the right move is to refactor onto go-toml's marshaller.
func tomlString(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	return `"` + s + `"`
}

// tomlStringArray returns a TOML inline-array literal of basic-string
// values. Nil and empty slices render as `[]` (explicit empty) so the
// emitted key is unambiguously present-with-no-values rather than
// missing. Each element passes through tomlString for escape correctness.
func tomlStringArray(in []string) string {
	if len(in) == 0 {
		return "[]"
	}
	parts := make([]string, 0, len(in))
	for _, v := range in {
		parts = append(parts, tomlString(v))
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

// init registers adaptRender with the dispatcher's bundle-render hook
// seam at package import time. The dispatcher cannot import this package
// directly because render imports dispatcher (for Bundle / BindingResolved
// / domain types); a registration init() inverts the dependency direction
// so the spawn-side seam stays cycle-free.
//
// init is colocated in render.go (rather than a separate init.go) to
// stay within the action item's declared paths scope. The cli_claude/render
// equivalent lives in its own init.go file; D4's path-scope discipline
// pulled the equivalent function inline here.
//
// TODO(post-D4): the single-global RegisterBundleRenderFunc seam is
// last-writer-wins today. Wiring BOTH cli_claude/render and cli_codex/render
// at cmd/till boot will require a CLIKind-keyed registry — that refactor
// is a follow-up drop. Today only one render package is side-effect-
// imported by production; D4 ships the codex implementation and its
// registration so the codex test suite can exercise the full hook.
func init() {
	dispatcher.RegisterBundleRenderFunc(adaptRender)
}

// adaptRender bridges dispatcher.BundleRenderFunc's `any` lister
// parameter to the concrete render.PermissionGrantsLister Render
// expects. nil → nil; anything else must satisfy the interface or
// the adapter returns ErrInvalidGrantsLister so callers see a clean
// failure rather than a downstream panic.
//
// The signature MUST stay byte-for-byte compatible with
// dispatcher.BundleRenderFunc; changing one without the other would
// break the registration line above at compile time.
func adaptRender(
	ctx context.Context,
	bundle dispatcher.Bundle,
	item domain.ActionItem,
	project domain.Project,
	binding dispatcher.BindingResolved,
	grantsLister any,
) (string, error) {
	var lister PermissionGrantsLister
	if grantsLister != nil {
		typed, ok := grantsLister.(PermissionGrantsLister)
		if !ok {
			return "", ErrInvalidGrantsLister
		}
		lister = typed
	}
	return Render(ctx, bundle, item, project, binding, lister)
}
