// Package render owns the per-spawn bundle render for the claude adapter.
// Drop 4c F.7-CORE F.7.3b: writes the cross-CLI system-prompt at
// <bundle.Root>/system-prompt.md PLUS the claude-specific plugin subtree
// under <bundle.Root>/plugin/ per spawn architecture memory §2.
//
// The function Render is the only exported surface; sub-renderers are
// package-private so the contract stays narrow. Render is called by the
// dispatcher's BuildSpawnCommand AFTER F.7.1's WriteManifest and BEFORE
// the claude adapter's BuildCommand argv assembly. It REPLACES the
// provisional minimal prompt block landed in F.7.17.5 (spawn.go's
// assemblePrompt + os.WriteFile pair).
//
// Phasing per planner-review §6.1: system-prompt.md is cross-CLI and lives
// at the bundle root; the plugin/ subtree is claude-specific. A future
// codex adapter (Drop 4d) renders its own subtree (codex_home/...) at the
// same level — this package covers ONLY the claude phasing.
//
// Permissions JSON shape per memory §4 (verbatim from claude docs + probes):
// settings.json carries `permissions: {allow, ask, deny}`. Render emits all
// three keys with empty arrays when the binding has no entries — explicit
// empty is more debuggable than absent + relies on claude's documented
// default-deny semantic for unmatched calls.
package render

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/evanmschultz/tillsyn/internal/app/dispatcher"
	"github.com/evanmschultz/tillsyn/internal/domain"
)

// PermissionGrantsLister is the narrower read-only view of
// app.PermissionGrantsStore the render package consumes. Drop 4c F.7.5c
// adds it so previously approved tool-permission grants from F.7.5b's TUI
// handshake (persisted via F.7.17.7's storage adapter) merge into the
// rendered settings.json without re-prompting the dev.
//
// Structural typing means callers wire the full app.PermissionGrantsStore
// and Go satisfies this narrower interface implicitly — render does not
// import the app package, breaking the otherwise-unavoidable cycle
// (app would need dispatcher → render → app).
//
// A nil lister is the documented graceful-skip path: Render proceeds with
// binding.ToolsAllowed only and emits no error. That keeps the spawn
// pipeline functional during the deferred plumbing window between F.7.5c
// (this droplet) and the follow-up that wires the production
// app.PermissionGrantsStore handle through BuildSpawnCommand.
type PermissionGrantsLister interface {
	// ListGrantsForKind returns every grant matching the supplied
	// (projectID, kind, cliKind) triple. Matches the signature on
	// app.PermissionGrantsStore.ListGrantsForKind verbatim so the
	// production storage adapter satisfies this interface without an
	// adapter shim. cliKind is matched case-insensitively per the
	// storage adapter's lowercase normalization.
	ListGrantsForKind(ctx context.Context, projectID string, kind domain.Kind, cliKind string) ([]domain.PermissionGrant, error)
}

// pluginSubdir is the conventional path under bundle.Root that the claude
// adapter materializes for its plugin tree. Mirrors cli_claude.pluginSubdir
// (kept duplicated here to avoid importing the parent package and creating
// a dependency cycle render → cli_claude → dispatcher → render).
const pluginSubdir = "plugin"

// claudePluginManifestSubdir is the canonical location of plugin.json
// inside a Claude plugin per `claude plugin validate` requirements.
const claudePluginManifestSubdir = ".claude-plugin"

// agentsSubdir is the canonical location for agent files inside a Claude
// plugin tree.
const agentsSubdir = "agents"

// ErrInvalidRenderInput is returned by Render when one of the supplied
// values fails an obvious shape check (empty bundle root, empty agent
// name, etc.) before any disk I/O. Callers detect via errors.Is and route
// to ErrInvalidSpawnInput-equivalent failure.
var ErrInvalidRenderInput = errors.New("render: invalid render input")

// ErrInvalidGrantsLister is returned by the dispatcher-seam adapter
// (init.go's adaptRender) when the `any`-typed grantsLister supplied by
// BuildSpawnCommand is non-nil but does NOT satisfy
// PermissionGrantsLister. The error surfaces a configuration mistake at
// the spawn-pipeline boundary rather than panicking on the type
// assertion.
var ErrInvalidGrantsLister = errors.New("render: grants lister does not implement PermissionGrantsLister")

// Render writes the per-spawn bundle artifacts the claude adapter needs:
//
//  1. <bundle.Root>/system-prompt.md (cross-CLI; full action-item context)
//  2. <bundle.Root>/plugin/.claude-plugin/plugin.json (claude plugin manifest)
//  3. <bundle.Root>/plugin/agents/<binding.AgentName>.md (agent stub)
//  4. <bundle.Root>/plugin/.mcp.json (Tillsyn MCP self-registration)
//  5. <bundle.Root>/plugin/settings.json (permissions + sandbox shell)
//
// Render is invoked by dispatcher.BuildSpawnCommand AFTER F.7.1's
// WriteManifest succeeds. The caller is responsible for cleaning up the
// bundle root on Render failure — Render itself only rolls back what it
// created (system-prompt.md + the plugin/ subtree) so a failed render
// leaves manifest.json from F.7.1 intact for orphan-scan correlation.
//
// Returns the rendered system-prompt body alongside the error so callers
// can populate dispatcher.SpawnDescriptor.Prompt without re-reading from
// disk. On error the returned body is the empty string.
//
// Files are written with 0o600 perms to match F.7.1's manifest write —
// the bundle directory is per-spawn and tooling-private and may carry
// action-item structural data the dev does not want broadcast.
//
// Validation is intentionally minimal: bundle.Paths.Root must be non-empty
// and binding.AgentName must be non-empty + free of path separators
// (defensive against accidental path-injection through a corrupted catalog
// — production AgentName values like "go-builder-agent" are safe).
//
// ctx is forwarded to grantsLister.ListGrantsForKind so the lister's
// underlying storage call can honor cancellation. grantsLister MAY be nil
// — render skips the grants-merge step and renders the binding's
// ToolsAllowed only. This is the deferred-plumbing path used by Drop 4c
// F.7.5c until the production app.PermissionGrantsStore handle reaches
// BuildSpawnCommand.
func Render(
	ctx context.Context,
	bundle dispatcher.Bundle,
	item domain.ActionItem,
	project domain.Project,
	binding dispatcher.BindingResolved,
	grantsLister PermissionGrantsLister,
) (string, error) {
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

	rollback := newRenderRollback(bundle.Paths.Root)

	// 1. system-prompt.md at bundle root (cross-CLI per memory §2).
	promptBody, err := renderSystemPrompt(bundle, item, project)
	if err != nil {
		rollback.run()
		return "", fmt.Errorf("render: system prompt: %w", err)
	}

	// 2. plugin/.claude-plugin/plugin.json
	if err := renderPluginManifest(bundle); err != nil {
		rollback.run()
		return "", fmt.Errorf("render: plugin manifest: %w", err)
	}

	// 3. plugin/agents/<name>.md
	if err := renderAgentFile(bundle, binding); err != nil {
		rollback.run()
		return "", fmt.Errorf("render: agent file: %w", err)
	}

	// 4. plugin/.mcp.json
	if err := renderMCPConfig(bundle); err != nil {
		rollback.run()
		return "", fmt.Errorf("render: mcp config: %w", err)
	}

	// 5. plugin/settings.json — F.7.5c merges previously stored
	// permission grants into permissions.allow when grantsLister is non-nil.
	if err := renderSettings(ctx, bundle, item, project, binding, grantsLister); err != nil {
		rollback.run()
		return "", fmt.Errorf("render: settings: %w", err)
	}

	return promptBody, nil
}

// renderRollback captures the paths created by Render so a partial render
// can clean itself up on failure. The struct deliberately tracks the
// bundle root rather than each individual file write — Render is the sole
// writer under <Root>/system-prompt.md and <Root>/plugin/, so a failed
// render can blanket-remove those two paths without touching F.7.1's
// manifest.json (which lives at the bundle root and is the caller's
// concern).
type renderRollback struct {
	bundleRoot string
}

// newRenderRollback returns a rollback handle bound to bundleRoot.
func newRenderRollback(bundleRoot string) renderRollback {
	return renderRollback{bundleRoot: bundleRoot}
}

// run removes every path Render writes (system-prompt.md + the plugin/
// subtree). Best-effort: cleanup errors are swallowed because the caller
// is already returning a non-nil error and the F.7.8 orphan scan will
// reap any straggler files via the bundle's manifest.
func (r renderRollback) run() {
	if r.bundleRoot == "" {
		return
	}
	_ = os.Remove(filepath.Join(r.bundleRoot, "system-prompt.md"))
	_ = os.RemoveAll(filepath.Join(r.bundleRoot, pluginSubdir))
}

// renderSystemPrompt writes <bundle.Root>/system-prompt.md with the full
// action-item context the spawned agent needs. The body shape mirrors
// the F.7.17.5 assemblePrompt body but is owned by render going forward;
// spawn.go's assemblePrompt is removed in this droplet.
//
// Returns the rendered body so the caller can mirror it into
// dispatcher.SpawnDescriptor.Prompt without a second disk read.
//
// Hylla awareness is deliberately omitted per F.7.10: Hylla is dev-local,
// not part of Tillsyn's shipped cascade. Adopters who want Hylla refs in
// their prompt body do so via SystemPromptTemplatePath (F.7.2).
func renderSystemPrompt(bundle dispatcher.Bundle, item domain.ActionItem, project domain.Project) (string, error) {
	body := assembleSystemPromptBody(item, project)
	if err := os.WriteFile(bundle.Paths.SystemPromptPath, []byte(body), 0o600); err != nil {
		return "", err
	}
	return body, nil
}

// assembleSystemPromptBody builds the system-prompt.md body. Pure
// function so tests can pin the exact text without touching the
// filesystem.
//
// Body fields:
//
//   - task_id, project_id, project_dir, kind, title — every spawn carries
//     these structural fields. The agent uses task_id to route Tillsyn
//     state moves; project_dir is the cd-target.
//   - paths, packages — emitted only when the action item declares them
//     (write scope + lock domain).
//   - move-state directive — every spawn instructs the agent to take
//     ownership of its lifecycle transitions.
//
// Auth credentials (session_id) intentionally NOT emitted today — Wave 3
// of Drop 4a (orch self-approval) is the seam where session_id gets
// folded into the prompt. F.7.3b ships the structural body; auth fold-in
// is the next-droplet concern.
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

// pluginManifest is the JSON shape of <plugin>/.claude-plugin/plugin.json
// per the verbatim CLI probe in spawn architecture memory §2. The
// minimum required field is `name`; additional fields (version, author,
// etc.) are accepted by claude but not emitted today.
type pluginManifest struct {
	// Name is the unique plugin identifier. Tillsyn uses "spawn-<spawn-id>"
	// so each per-spawn bundle has a distinct plugin name; this avoids
	// any cache collision claude might apply if two bundles shared a
	// plugin name in the same session.
	Name string `json:"name"`
}

// renderPluginManifest writes <plugin>/.claude-plugin/plugin.json.
func renderPluginManifest(bundle dispatcher.Bundle) error {
	dir := filepath.Join(bundle.Paths.Root, pluginSubdir, claudePluginManifestSubdir)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("mkdir plugin manifest dir: %w", err)
	}
	manifest := pluginManifest{Name: "spawn-" + bundle.SpawnID}
	payload, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal plugin manifest: %w", err)
	}
	return os.WriteFile(filepath.Join(dir, "plugin.json"), payload, 0o600)
}

// renderAgentFile writes <plugin>/agents/<name>.md with the canonical
// Tillsyn agent template per kind. F.7.3b ships a MINIMAL stub: the
// frontmatter carries `name` + `description`; the body carries a single
// pointer line directing the agent to consult its cascade-binding for
// behavior. The full canonical templates at ~/.claude/agents/<name>.md
// remain the source of truth for behavior — they are loaded by claude
// from the system-installed plugin path (Path B per memory §1), not from
// the per-spawn plugin (Path A).
//
// Two-layer tool-gating per memory §5: the frontmatter `disallowedTools`
// field mirrors binding.ToolsDisallowed for human readability. Layer B
// (settings.json permissions) remains the authoritative gate; this is
// the safety-net + readability layer.
//
// Future evolution (deferred — out of scope for F.7.3b):
//
//   - System-prompt template path read from binding.SystemPromptTemplatePath.
//     F.7.2 landed the field; F.7.3b's MINIMAL stub does not consult it.
//     A follow-up droplet adds template-rendering against the path.
func renderAgentFile(bundle dispatcher.Bundle, binding dispatcher.BindingResolved) error {
	dir := filepath.Join(bundle.Paths.Root, pluginSubdir, agentsSubdir)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("mkdir agents dir: %w", err)
	}
	body := assembleAgentFileBody(binding)
	return os.WriteFile(filepath.Join(dir, binding.AgentName+".md"), []byte(body), 0o600)
}

// assembleAgentFileBody builds the agent stub body. Pure function so
// tests can pin the exact text. The frontmatter shape mirrors the
// canonical Tillsyn agent files at ~/.claude/agents/<name>.md — `name`
// + `description` are the load-bearing fields claude consumes; tools /
// allowedTools / disallowedTools carry the per-spawn gating layer A.
func assembleAgentFileBody(binding dispatcher.BindingResolved) string {
	var b strings.Builder
	b.WriteString("---\n")
	b.WriteString("name: ")
	b.WriteString(binding.AgentName)
	b.WriteString("\n")
	b.WriteString("description: Tillsyn-spawned ")
	b.WriteString(binding.AgentName)
	b.WriteString(" subagent.\n")
	if len(binding.ToolsAllowed) > 0 {
		b.WriteString("allowedTools: ")
		b.WriteString(strings.Join(binding.ToolsAllowed, ", "))
		b.WriteString("\n")
	}
	if len(binding.ToolsDisallowed) > 0 {
		b.WriteString("disallowedTools: ")
		b.WriteString(strings.Join(binding.ToolsDisallowed, ", "))
		b.WriteString("\n")
	}
	b.WriteString("---\n\n")
	b.WriteString("Tillsyn-spawned subagent stub. Behavior loaded from the canonical ")
	b.WriteString(binding.AgentName)
	b.WriteString(" template at the system-installed plugin path.\n")
	return b.String()
}

// mcpConfig is the JSON shape of <plugin>/.mcp.json per spawn architecture
// memory §2. Tillsyn registers itself as a stdio MCP child of claude so
// the spawned agent can call till.* MCP methods (auth, action_item, etc.)
// against the same orchestrator-managed Tillsyn process.
type mcpConfig struct {
	// Tillsyn registers the dev's till binary as a stdio MCP server named
	// "tillsyn". Future adopters may shadow this name; see CLAUDE.md
	// CONTRIBUTING §"Dev MCP Server Setup" for the per-worktree naming
	// scheme. F.7.3b ships the production-canonical "tillsyn" name —
	// per-worktree shadow names land in F.7.17.4 / cli_register wiring.
	Tillsyn mcpServerEntry `json:"tillsyn"`
}

// mcpServerEntry is one entry under mcpConfig.Tillsyn matching claude's
// MCP-config schema for stdio servers.
type mcpServerEntry struct {
	// Command is the executable claude spawns; "till" relies on PATH
	// resolution at exec time.
	Command string `json:"command"`
	// Args are the CLI arguments passed to Command. "serve-mcp" is
	// Tillsyn's MCP-stdio entrypoint.
	Args []string `json:"args"`
}

// renderMCPConfig writes <plugin>/.mcp.json.
func renderMCPConfig(bundle dispatcher.Bundle) error {
	dir := filepath.Join(bundle.Paths.Root, pluginSubdir)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("mkdir plugin dir: %w", err)
	}
	cfg := mcpConfig{
		Tillsyn: mcpServerEntry{
			Command: "till",
			Args:    []string{"serve-mcp"},
		},
	}
	payload, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal mcp config: %w", err)
	}
	return os.WriteFile(filepath.Join(dir, ".mcp.json"), payload, 0o600)
}

// settingsFile is the JSON shape of <plugin>/settings.json per memory §4
// (verbatim from claude docs + probes). Two top-level keys ship today:
// `permissions` (Layer B tool-gating per memory §5) and `sandbox` (per
// memory §4 — filesystem + network from F.7.2 [agent.sandbox] TOML).
//
// REV-11 worktree-escape check: F.7.2 deferred AllowWrite-escape-from-
// worktree validation to spawn-time. Render today does NOT enforce that
// check — it ships in a follow-up droplet that has the project worktree
// path threaded through (the binding.* path doesn't carry it; project
// does). Keeping this an explicit deferral is consistent with REV-11.
type settingsFile struct {
	Permissions permissionsBlock `json:"permissions"`
	// Sandbox is omitted (zero value) when the binding carries no sandbox
	// directives. Today BindingResolved does not yet carry the Sandbox
	// sub-struct from templates.AgentBinding — extension is a future
	// droplet; render emits permissions only.
}

// permissionsBlock is the verbatim claude permissions schema from
// memory §4. All three keys are emitted with explicit empty arrays when
// the binding has no entries — explicit empty is more debuggable than
// absent.
type permissionsBlock struct {
	Allow []string `json:"allow"`
	Ask   []string `json:"ask"`
	Deny  []string `json:"deny"`
}

// renderSettings writes <plugin>/settings.json. Permissions.allow is
// sourced from binding.ToolsAllowed PLUS any previously-stored
// permission grants the lister returns for (project.ID, item.Kind,
// binding.CLIKind); permissions.deny mirrors binding.ToolsDisallowed.
// permissions.ask stays an explicit empty array — F.7.5b's TUI handshake
// owns the in-flight ask vocabulary; persisted grants land in allow.
//
// Drop 4c F.7.5c grants-merge contract:
//
//   - grantsLister == nil → graceful skip; allow = binding.ToolsAllowed only.
//   - binding.CLIKind == "" → grants lookup is skipped (the storage layer's
//     UNIQUE composite requires a non-empty CLIKind, so the lookup would
//     never match anyway).
//   - lister returns an error → wrapped and returned; render's rollback
//     cleans up the partially-written bundle.
//   - Order: binding.ToolsAllowed entries first (preserved verbatim),
//     then grants in the lister's storage order (granted_at-ASC per
//     PermissionGrantsStore.ListGrantsForKind contract). Within each
//     group, dedup is preserve-first-seen.
//
// Empty / nil slices on the binding render as `[]` (explicit empty)
// rather than omitted JSON keys — claude's evaluation order is
// deny → ask → allow with first-match-wins, so an explicit empty-allow
// list is functionally identical to a missing-allow but more
// debuggable when the dev opens the file.
func renderSettings(
	ctx context.Context,
	bundle dispatcher.Bundle,
	item domain.ActionItem,
	project domain.Project,
	binding dispatcher.BindingResolved,
	grantsLister PermissionGrantsLister,
) error {
	dir := filepath.Join(bundle.Paths.Root, pluginSubdir)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("mkdir plugin dir: %w", err)
	}

	allow, err := mergeAllowList(ctx, item, project, binding, grantsLister)
	if err != nil {
		return fmt.Errorf("merge allow list: %w", err)
	}

	settings := settingsFile{
		Permissions: permissionsBlock{
			Allow: allow,
			Ask:   []string{},
			Deny:  nonNilStringSlice(binding.ToolsDisallowed),
		},
	}
	payload, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal settings: %w", err)
	}
	return os.WriteFile(filepath.Join(dir, "settings.json"), payload, 0o600)
}

// mergeAllowList combines binding.ToolsAllowed with stored grants from
// grantsLister, preserving binding-first-then-grants order and
// deduplicating against the running set. Returned slice is always
// non-nil (json.Marshal emits `[]` instead of `null`).
//
// Skip-conditions for the grants lookup:
//
//   - grantsLister == nil → deferred plumbing path; binding only.
//   - binding.CLIKind == "" → the storage UNIQUE composite requires
//     non-empty cli_kind so a lookup with "" would never match; we
//     short-circuit rather than emit a no-op DB query.
//
// Errors from grantsLister.ListGrantsForKind propagate — render's
// rollback cleans up the partially-written bundle.
func mergeAllowList(
	ctx context.Context,
	item domain.ActionItem,
	project domain.Project,
	binding dispatcher.BindingResolved,
	grantsLister PermissionGrantsLister,
) ([]string, error) {
	merged := make([]string, 0, len(binding.ToolsAllowed))
	seen := make(map[string]struct{}, len(binding.ToolsAllowed))
	for _, rule := range binding.ToolsAllowed {
		if _, ok := seen[rule]; ok {
			continue
		}
		seen[rule] = struct{}{}
		merged = append(merged, rule)
	}

	if grantsLister == nil || strings.TrimSpace(string(binding.CLIKind)) == "" {
		return merged, nil
	}

	grants, err := grantsLister.ListGrantsForKind(ctx, project.ID, item.Kind, string(binding.CLIKind))
	if err != nil {
		return nil, fmt.Errorf("list grants for kind %q cli %q: %w", item.Kind, binding.CLIKind, err)
	}
	for _, g := range grants {
		if _, ok := seen[g.Rule]; ok {
			continue
		}
		seen[g.Rule] = struct{}{}
		merged = append(merged, g.Rule)
	}
	return merged, nil
}

// nonNilStringSlice returns []string{} when in is nil so json.Marshal
// emits `[]` rather than `null`. Explicit empty is more debuggable than
// `null` for an allow/deny list a dev might inspect manually.
func nonNilStringSlice(in []string) []string {
	if in == nil {
		return []string{}
	}
	return in
}
