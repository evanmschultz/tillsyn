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
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/evanmschultz/tillsyn/internal/app/dispatcher"
	"github.com/evanmschultz/tillsyn/internal/config"
	"github.com/evanmschultz/tillsyn/internal/domain"
	"github.com/evanmschultz/tillsyn/internal/templates"
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

// ErrAgentBodyNotFound is returned by assembleAgentFileBody when the
// 3-tier resolver exhausts every tier — project tier, user tier, and the
// embedded tier's cross-group fallback to till-gen — without finding a
// matching agent file.
//
// Per the W3-FF7 LOCKED contract the embedded tier's lookup ladder is
// primary `builtin/agents/<group>/<basename>` → fallback
// `builtin/agents/till-gen/<basename>` on fs.ErrNotExist. If till-gen
// also misses the resolver returns ErrAgentBodyNotFound wrapped with the
// failing AgentName + group + basename context. Callers detect via
// errors.Is(err, ErrAgentBodyNotFound) and route to the standard
// Render-rollback path.
var ErrAgentBodyNotFound = errors.New("render: agent body not found in project, user, or embedded tier")

// ErrInvalidAgentTemplatePath is returned by assembleAgentFileBody when
// binding.SystemPromptTemplatePath fails the full-path traversal-defense
// check (round-2 fix for W3-D23-FF1).
//
// Reject rules (applied AFTER the empty-path-is-OK short-circuit per
// W3-FF5 LOCKED — empty SystemPromptTemplatePath still routes to the
// till-go embedded default):
//
//  1. Absolute paths (starts with "/").
//  2. Any path segment equal to ".." (full-path defense; the existing
//     validateAgentBasename only catches leaf "..").
//  3. Empty intermediate segments (catches "till-go//passwd" shapes that
//     would otherwise split into an empty segment under strings.Split).
//
// Without this validator a malicious template could set
// SystemPromptTemplatePath = "till-go/../../../../etc/passwd" — path.Base
// returns "passwd" (passes the basename validator), path.Dir returns
// "../../../../etc" (UNVALIDATED), and the user tier's
// filepath.Join(home, ".tillsyn/agents", group, basename) cancels under
// filepath.Clean to /etc/passwd, leaking host-file content into the
// rendered agent body when os.ReadFile succeeds.
//
// Threat model: today bounded (SystemPromptTemplatePath comes from
// repo-owned till-*.toml templates), but becomes attacker-controllable
// as team-aware architecture matures (per
// feedback_prompt_injection_team.md +
// project_team_aware_architecture.md). Adding the validator now is
// defense-in-depth ahead of the team-feature landing.
//
// Callers detect via errors.Is(err, ErrInvalidAgentTemplatePath) and
// route to the standard Render-rollback path.
var ErrInvalidAgentTemplatePath = errors.New("render: invalid agent template path (traversal-defense reject)")

// agentBodyEmbeddedRoot is the embed.FS-relative root under
// internal/templates.DefaultTemplateFS where placeholder agent .md
// scaffolding lives. Per the W3-FF5 + W3-FF7 LOCKED contracts the
// resolver walks <agentBodyEmbeddedRoot>/<group>/<basename> first, then
// falls back to <agentBodyEmbeddedRoot>/till-gen/<basename> on miss.
const agentBodyEmbeddedRoot = "builtin/agents"

// agentBodyDefaultGroup is the dogfood default group selected when
// binding.SystemPromptTemplatePath is empty (W3-FF5 LOCKED). Adopters
// targeting till-gen / till-gdd MUST set SystemPromptTemplatePath
// explicitly in their template; the empty-path fallback is documented
// dogfood-only behavior.
const agentBodyDefaultGroup = "till-go"

// agentBodyFallbackGroup is the canonical shared-agents group the
// embedded-tier cross-group fallback descends into (W3-FF7 LOCKED).
// One-way fallback only — till-gen does NOT fall back to other groups.
const agentBodyFallbackGroup = "till-gen"

// projectAgentsSubdir is the per-project override directory the project
// tier reads from (`<project.RepoPrimaryWorktree>/.tillsyn/agents/`).
const projectAgentsSubdir = ".tillsyn/agents"

// userAgentsSubdir is the per-user override directory the user tier
// reads from (`<user-home>/.tillsyn/agents/<group>/`). The user tier is
// group-scoped; the project tier is not (the project owns its agents
// directly).
const userAgentsSubdir = ".tillsyn/agents"

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

	// 3. plugin/agents/<name>.md — sources body via the W3.D2 3-tier
	// resolver (project → user → embedded with till-gen cross-group fallback).
	if err := renderAgentFile(bundle, project, binding); err != nil {
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

// renderAgentFile writes <plugin>/agents/<name>.md with the resolved
// agent body from the W3.D2 3-tier resolver. The body is emitted verbatim
// to disk — frontmatter splitting and per-spawn tool-gating frontmatter
// injection are D3's concern (the strip-then-inject pipeline layers ON
// TOP of D2's output).
//
// The `project` parameter feeds the project-tier branch of the resolver
// (`<project.RepoPrimaryWorktree>/.tillsyn/agents/<basename>`). Per the
// W3-FF5 + W3-FF7 LOCKED contracts the resolver walks the tiers in this
// priority order:
//
//  1. project tier — `<project.RepoPrimaryWorktree>/.tillsyn/agents/<basename>`
//  2. user tier    — `<user-home>/.tillsyn/agents/<group>/<basename>`
//  3. embedded tier — `templates.DefaultTemplateFS` via
//     `builtin/agents/<group>/<basename>` with cross-group fallback to
//     `builtin/agents/till-gen/<basename>` on fs.ErrNotExist.
//
// `<group>` derivation (W3-FF5 LOCKED):
//
//   - `<group> = path.Dir(binding.SystemPromptTemplatePath)` when non-empty
//     (slash-aware `path.Dir`, NOT OS-aware `filepath.Dir`, because
//     embed.FS paths are always slash-separated).
//   - `<group> = "till-go"` (dogfood default) when empty.
//   - If `path.Dir` returns "." (path has no slash), the resolver treats
//     the path as malformed and falls back to "till-go".
//
// `<basename>` derivation:
//
//   - `<basename> = path.Base(binding.SystemPromptTemplatePath)` when
//     non-empty.
//   - `<basename> = binding.AgentName + ".md"` when empty.
//
// On any tier returning an error other than fs.ErrNotExist (e.g.
// permission denied on the project tier), the resolver propagates the
// error wrapped with the failing tier's identity — fail-loud rather than
// silently skipping to the next tier.
//
// On a 3-tier exhaustion the resolver returns ErrAgentBodyNotFound wrapped
// with the failing AgentName + group + basename context.
func renderAgentFile(bundle dispatcher.Bundle, project domain.Project, binding dispatcher.BindingResolved) error {
	dir := filepath.Join(bundle.Paths.Root, pluginSubdir, agentsSubdir)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("mkdir agents dir: %w", err)
	}
	body, err := assembleAgentFileBody(project, binding)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, binding.AgentName+".md"), []byte(body), 0o600)
}

// assembleAgentFileBody resolves the agent file body via the 3-tier
// ladder documented on renderAgentFile, then layers the W3.D3
// strip-then-inject pipeline on top of the resolved body.
//
// Pure read-only function (no disk mutation): performs disk + embed.FS
// reads and string transformations only.
//
// D3 strip-then-inject pipeline (post W3-PF1 / W3-FF2 / W3-FF12 LOCKED):
//
//  1. Split the resolved body at the leading + trailing `---\n` delimiters
//     to extract (frontmatter, postFrontmatter). If either delimiter is
//     absent the body is returned unchanged — D5's post-render validator
//     catches malformed agent files at the validator layer.
//  2. Strip template-time frontmatter keys via
//     config.StripFrontmatterKeys(frontmatter, stripModel, stripTools):
//     - stripModel = binding.Model != nil && *binding.Model != "" (W3-FF2;
//     ResolveBinding always populates *Model via resolveStringPtr → a
//     bare !=nil predicate is always-true and would mis-strip every
//     render).
//     - stripTools = true ALWAYS (W3-FF12; tool-gating keys are
//     template-time only — runtime per-spawn injection is the sole
//     authoritative source, so unconditional strip prevents stale disk
//     tool-gating frontmatter from surviving when binding has empty
//     tool gates).
//  3. Inject runtime per-spawn fields:
//     - allowedTools: <comma-joined binding.ToolsAllowed> when non-empty.
//     - disallowedTools: <comma-joined binding.ToolsDisallowed> when
//     non-empty.
//     Empty binding tool-gate slices skip injection (no line emitted).
//  4. Re-concatenate `---\n` + stripped+injected frontmatter + `---\n` +
//     postFrontmatter.
//
// Ordering is mandatory: strip first removes any stale disk tool-gating
// frontmatter regardless of binding state; injection then adds runtime
// values back when the binding has them. Inverting the order would
// either (a) merge stale embedded values with runtime values OR (b) strip
// the freshly-injected runtime values — both wrong.
func assembleAgentFileBody(project domain.Project, binding dispatcher.BindingResolved) (string, error) {
	// Round-2 W3-D23-FF1 fix: validate the full template path BEFORE
	// any path.Dir / path.Base derivation. The empty-path branch is
	// the W3-FF5 LOCKED "use embedded till-go default" sentinel and
	// MUST short-circuit before the validator runs — the validator
	// only inspects non-empty paths.
	if trimmed := strings.TrimSpace(binding.SystemPromptTemplatePath); trimmed != "" {
		if err := validateAgentTemplatePath(trimmed); err != nil {
			return "", fmt.Errorf("%w: %q: %s",
				ErrInvalidAgentTemplatePath, trimmed, err.Error())
		}
	}

	basename, err := resolveAgentBasename(binding)
	if err != nil {
		return "", err
	}
	group := resolveAgentGroup(binding)

	// Tier 1 — project tier.
	body, found, err := readProjectTierAgent(project.RepoPrimaryWorktree, basename)
	if err != nil {
		return "", fmt.Errorf("project-tier read: %w", err)
	}
	if !found {
		// Tier 2 — user tier.
		body, found, err = readUserTierAgent(group, basename)
		if err != nil {
			return "", fmt.Errorf("user-tier read: %w", err)
		}
	}
	if !found {
		// Tier 3 — embedded tier with cross-group fallback to till-gen.
		body, err = readEmbeddedTierAgent(group, basename)
		if err != nil {
			return "", err
		}
	}

	// D3 strip-then-inject pipeline. Operates only when both `---\n`
	// delimiters are present — malformed bodies pass through unchanged
	// for D5's validator to catch.
	transformed, ok := stripAndInjectAgentFrontmatter(body, binding)
	if !ok {
		return body, nil
	}
	return transformed, nil
}

// stripAndInjectAgentFrontmatter applies the W3.D3 strip-then-inject
// pipeline to body. Returns the transformed body string + true on success
// or ("", false) if body lacks the leading or trailing `---\n` delimiter
// (caller passes the body through unchanged in that case).
//
// Strip predicates (LOCKED):
//   - stripModel = binding.Model != nil && *binding.Model != "" (W3-FF2).
//   - stripTools = true ALWAYS (W3-FF12).
//
// Injection (post-strip):
//   - non-empty binding.ToolsAllowed → append `allowedTools: <comma-joined>`.
//   - non-empty binding.ToolsDisallowed → append `disallowedTools: <comma-joined>`.
//
// Both lines are appended as plain YAML scalars (NOT YAML lists). The
// claude frontmatter convention from memory §5 expects comma-joined
// string form, mirroring the F.7.3b stub's renderAgentFile output.
func stripAndInjectAgentFrontmatter(body string, binding dispatcher.BindingResolved) (string, bool) {
	const delim = "---\n"

	// Leading delimiter must be at position 0 OR after a BOM/leading
	// whitespace block. Accept the canonical shape only — D5 catches
	// any non-canonical malformed shape downstream.
	if !strings.HasPrefix(body, delim) {
		return "", false
	}
	afterOpen := body[len(delim):]

	// Find the closing `---\n` delimiter.
	closeIdx := strings.Index(afterOpen, delim)
	if closeIdx < 0 {
		return "", false
	}
	frontmatter := afterOpen[:closeIdx]
	postFrontmatter := afterOpen[closeIdx+len(delim):]

	stripModel := binding.Model != nil && *binding.Model != ""
	const stripTools = true // W3-FF12: always-strip; runtime inject is sole source.

	stripped, err := config.StripFrontmatterKeys(frontmatter, stripModel, stripTools)
	if err != nil {
		// Parse failure on the frontmatter — surface as malformed-body
		// path so D5 catches it. Returning (body, false) preserves the
		// original bytes for the caller's pass-through path.
		return "", false
	}

	// Ensure trailing newline before injecting so the appended lines
	// don't accidentally merge with the last line of the stripped
	// frontmatter. config.StripFrontmatterKeys' marshalNode helper
	// already guarantees a single trailing `\n` on non-empty output,
	// but the no-op short-circuit path (both flags false) returns the
	// input verbatim — defensive trailing-newline ensures correctness
	// across both paths.
	injected := stripped
	if injected != "" && !strings.HasSuffix(injected, "\n") {
		injected += "\n"
	}
	if len(binding.ToolsAllowed) > 0 {
		injected += "allowedTools: " + strings.Join(binding.ToolsAllowed, ", ") + "\n"
	}
	if len(binding.ToolsDisallowed) > 0 {
		injected += "disallowedTools: " + strings.Join(binding.ToolsDisallowed, ", ") + "\n"
	}

	return delim + injected + delim + postFrontmatter, true
}

// resolveAgentBasename returns the file-portion of the agent body lookup
// derived from binding.SystemPromptTemplatePath (when non-empty) or from
// `binding.AgentName + ".md"` (when empty). The returned basename is
// sanitized — no path separators, no parent-traversal segments, no
// absolute paths — to prevent escape from the .tillsyn/agents/ directory
// at the project and user tiers and from the builtin/agents/ subtree at
// the embedded tier.
func resolveAgentBasename(binding dispatcher.BindingResolved) (string, error) {
	var basename string
	if trimmed := strings.TrimSpace(binding.SystemPromptTemplatePath); trimmed != "" {
		basename = path.Base(trimmed)
	} else {
		basename = binding.AgentName + ".md"
	}
	if err := validateAgentBasename(basename); err != nil {
		return "", fmt.Errorf("%w: agent basename %q invalid: %s",
			ErrInvalidRenderInput, basename, err.Error())
	}
	return basename, nil
}

// validateAgentTemplatePath enforces the full-path traversal-defense
// applied at the top of assembleAgentFileBody, BEFORE path.Dir /
// path.Base derive the group + basename from the input. Round-2 fix for
// W3-D23-FF1: validateAgentBasename only inspects the leaf returned by
// path.Base; the group derived from path.Dir was previously unvalidated
// and could carry ".." segments that filepath.Join + filepath.Clean
// collapsed into a host-file traversal at the user tier.
//
// Reject rules (per ErrInvalidAgentTemplatePath doc-comment):
//
//  1. Absolute paths.
//  2. Any path segment equal to ".." (anywhere in the path).
//  3. Empty intermediate segments (consecutive separators).
//
// Note on narrowing: a segment containing ".." as a substring (e.g.
// "..foo") is NOT rejected. filepath.Clean does not collapse
// "..foo" — it is a literal directory name, not a traversal. The W3-D23-FF1
// counterexample exploited segments equal to "..", not substrings. Keeping
// the rule narrow avoids spuriously rejecting legitimate dotfile-like
// segments while still closing the documented attack.
//
// Empty path is NOT inspected here — the W3-FF5 LOCKED empty-string
// sentinel routes to the till-go embedded default and short-circuits
// before this validator runs. The caller's TrimSpace guard handles that.
func validateAgentTemplatePath(p string) error {
	// Absolute-path rule. path.IsAbs would work on slash-separated
	// strings (which is the convention for binding.SystemPromptTemplatePath
	// per W3-FF5: embed.FS paths are always slash-separated), but
	// inspecting the leading byte is simpler and equivalent here.
	if strings.HasPrefix(p, "/") {
		return errors.New("absolute path not allowed")
	}
	// Backslash defense: even though slash is the canonical separator,
	// reject backslashes anywhere in the path so a Windows-host
	// adopter can't accidentally craft a path that filepath.Join
	// would treat as separator on its platform.
	if strings.Contains(p, `\`) {
		return errors.New("backslash separator not allowed")
	}
	// Split on slash and inspect each segment.
	for _, seg := range strings.Split(p, "/") {
		if seg == "" {
			return errors.New("empty path segment (consecutive separators)")
		}
		if seg == ".." {
			return errors.New("parent-traversal segment `..` not allowed")
		}
	}
	return nil
}

// validateAgentBasename enforces the path-traversal defense applied at
// the project, user, and embedded tier lookups. Mirrors the existing
// AgentName path-separator check (render.go input validation in Render).
func validateAgentBasename(basename string) error {
	if basename == "" {
		return errors.New("basename is empty")
	}
	if basename == "." || basename == ".." {
		return errors.New("basename is a traversal segment")
	}
	if strings.ContainsAny(basename, `/\`) {
		return errors.New("basename contains path separator")
	}
	if strings.Contains(basename, "..") {
		return errors.New("basename contains parent-traversal sequence")
	}
	if filepath.IsAbs(basename) {
		return errors.New("basename is an absolute path")
	}
	return nil
}

// resolveAgentGroup applies the W3-FF5 LOCKED <group> derivation. Slash-
// aware path.Dir on binding.SystemPromptTemplatePath when non-empty;
// agentBodyDefaultGroup (till-go) when empty OR when path.Dir returns "."
// (malformed path).
func resolveAgentGroup(binding dispatcher.BindingResolved) string {
	if trimmed := strings.TrimSpace(binding.SystemPromptTemplatePath); trimmed != "" {
		if dir := path.Dir(trimmed); dir != "" && dir != "." {
			return dir
		}
	}
	return agentBodyDefaultGroup
}

// readProjectTierAgent attempts to read the project-tier agent file at
// `<projectWorktree>/.tillsyn/agents/<basename>`. Returns (body, true,
// nil) on hit, ("", false, nil) on fs.ErrNotExist or when the worktree
// path is empty, and ("", false, err) on any other I/O error.
//
// An empty projectWorktree (project not yet bootstrapped) skips this
// tier silently — there is no path to read from and no I/O error to
// surface.
func readProjectTierAgent(projectWorktree, basename string) (string, bool, error) {
	if strings.TrimSpace(projectWorktree) == "" {
		return "", false, nil
	}
	p := filepath.Join(projectWorktree, projectAgentsSubdir, basename)
	body, err := os.ReadFile(p)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return "", false, nil
		}
		return "", false, err
	}
	return string(body), true, nil
}

// readUserTierAgent attempts to read the user-tier agent file at
// `$HOME/.tillsyn/agents/<group>/<basename>` using os.UserHomeDir to
// resolve `$HOME`. Returns (body, true, nil) on hit, ("", false, nil) on
// fs.ErrNotExist OR when os.UserHomeDir fails (no $HOME available — a
// rare-but-real case in CI sandboxes; skip the tier rather than fail-loud),
// and ("", false, err) on other I/O errors.
func readUserTierAgent(group, basename string) (string, bool, error) {
	home, err := os.UserHomeDir()
	if err != nil || strings.TrimSpace(home) == "" {
		return "", false, nil
	}
	p := filepath.Join(home, userAgentsSubdir, group, basename)
	body, err := os.ReadFile(p)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return "", false, nil
		}
		return "", false, err
	}
	return string(body), true, nil
}

// readEmbeddedTierAgent reads from templates.DefaultTemplateFS via the
// W3-FF7 LOCKED cross-group fallback ladder:
//
//  1. primary  — `builtin/agents/<group>/<basename>`
//  2. fallback — `builtin/agents/till-gen/<basename>` on fs.ErrNotExist
//
// If the primary group is already till-gen, the fallback is skipped (no
// symmetric fallback — till-gen is the only fallback target).
//
// On primary hit returns the primary content. On primary miss + fallback
// hit returns the fallback content. On both miss returns a wrapped
// ErrAgentBodyNotFound. On any non-ErrNotExist read error the error is
// propagated verbatim.
//
// embed.FS read errors that are NOT ErrNotExist (e.g. malformed embed,
// although embed.FS is read-only and compiled-in so this is improbable)
// propagate up fail-loud.
func readEmbeddedTierAgent(group, basename string) (string, error) {
	primary := path.Join(agentBodyEmbeddedRoot, group, basename)
	body, err := fs.ReadFile(templates.DefaultTemplateFS, primary)
	if err == nil {
		return string(body), nil
	}
	if !errors.Is(err, fs.ErrNotExist) {
		return "", fmt.Errorf("embedded primary read %q: %w", primary, err)
	}

	// Primary miss — fall back to till-gen unless group already is
	// till-gen (no symmetric fallback).
	if group == agentBodyFallbackGroup {
		return "", fmt.Errorf("%w: agent_name lookup miss (group %q, basename %q)",
			ErrAgentBodyNotFound, group, basename)
	}

	fallback := path.Join(agentBodyEmbeddedRoot, agentBodyFallbackGroup, basename)
	body, err = fs.ReadFile(templates.DefaultTemplateFS, fallback)
	if err == nil {
		return string(body), nil
	}
	if !errors.Is(err, fs.ErrNotExist) {
		return "", fmt.Errorf("embedded fallback read %q: %w", fallback, err)
	}

	return "", fmt.Errorf("%w: agent_name lookup miss (group %q, basename %q, fallback %q)",
		ErrAgentBodyNotFound, group, basename, fallback)
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
