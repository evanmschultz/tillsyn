package dispatcher

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/evanmschultz/tillsyn/internal/domain"
	"github.com/evanmschultz/tillsyn/internal/templates"
)

// AuthBundle is the placeholder seam for the dispatcher's auth/lease plumbing.
//
// Wave 3: Wave 3 of Drop 4a (auth flow) replaces this stub with the populated
// form — session/lease IDs, MCP config materialization, capability scope. The
// Wave-2.6 spawner accepted a zero-value AuthBundle; Drop 4c F.7.17.5 keeps
// the same seam unchanged so callers do not break across the adapter wiring
// landing. F.7-CORE F.7.1 (bundle-materializer) and Wave 3 (auth materializer)
// will populate the bundle paths through dependencies of BuildSpawnCommand,
// not through fields on AuthBundle.
//
// The struct is intentionally empty today: any populated form Wave 3 chooses
// (struct vs interface vs functional option) is a non-breaking change because
// no caller is reading fields off the bundle. Tests in this droplet pass the
// zero value verbatim.
type AuthBundle struct {
	// _ keeps positional struct-literal usage compile-broken so Wave 3 can add
	// fields without silently changing existing call sites.
	_ struct{}
}

// SpawnDescriptor captures the inputs and resolved outputs of one
// BuildSpawnCommand call. The struct is the logging + monitor handoff —
// 4a.21's process monitor consumes it to populate trace fields, the
// continuous-mode loop in Drop 4b surfaces it on dispatcher dashboards, and
// Drop 4c F.7.17.5 carries the bundle root path through MCPConfigPath so
// `till dispatcher run --dry-run` JSON output continues to reflect the spawn
// intent across the adapter wiring landing.
//
// Every field is set by BuildSpawnCommand on success. Callers MUST NOT mutate
// the returned descriptor; the dispatcher treats the value as immutable for
// the lifetime of the spawn.
type SpawnDescriptor struct {
	// AgentName is the resolved agent variant from
	// templates.KindCatalog.LookupAgentBinding (e.g. "go-builder-agent"). Per
	// the Wave 2.6 division of labor the binding is taken verbatim — this
	// droplet does NOT synthesize {lang}-{role} variants from project.Language.
	AgentName string
	// Model is the LLM model identifier propagated to the agent CLI flags
	// (e.g. "opus", "sonnet", "haiku"). Sourced from binding.Model.
	Model string
	// MaxBudgetUSD is the per-spawn dollar cap propagated as the
	// `--max-budget-usd` flag value.
	MaxBudgetUSD float64
	// MaxTurns is the conversation-turn cap propagated as the `--max-turns`
	// flag value.
	MaxTurns int
	// MCPConfigPath is the absolute filesystem path passed to `--mcp-config`.
	// Drop 4c F.7.17.5 wires this from the per-spawn bundle root: claude's
	// adapter materializes its plugin tree under <bundleRoot>/plugin/ and the
	// MCP config lives at <bundleRoot>/plugin/.mcp.json. F.7-CORE F.7.1 will
	// own the bundle lifecycle (manifest.json, deferred cleanup, project-mode
	// root); the path shape stays compatible.
	MCPConfigPath string
	// Prompt is the assembled spawn prompt body. The body is opaque to the
	// dispatcher; the F.7.17.5 promptAssembler only guarantees that
	// structural fields (task_id, project_dir, move-state directive) are
	// present so downstream agents can self-locate. The prompt is written to
	// disk at BundlePaths.SystemPromptPath and surfaced to claude via
	// `--system-prompt-file`; the descriptor field carries the same body for
	// dry-run JSON / monitor logging.
	Prompt string
	// WorkingDir is the project worktree the agent runs in (cmd.Dir).
	WorkingDir string
}

// ErrNoAgentBinding is returned by BuildSpawnCommand when the project's baked
// KindCatalog has no AgentBinding registered for item.Kind. Callers detect
// this via errors.Is and route the action item to a "no agent configured"
// failure rather than treating it as a transient error.
var ErrNoAgentBinding = errors.New("dispatcher: no agent binding for kind")

// ErrInvalidSpawnInput is returned by BuildSpawnCommand when a required input
// field is empty or malformed (empty action-item ID, empty
// project.RepoPrimaryWorktree, etc.). Callers detect this via errors.Is.
var ErrInvalidSpawnInput = errors.New("dispatcher: invalid spawn input")

// ErrUnsupportedCLIKind is returned by BuildSpawnCommand when the resolved
// binding's CLIKind has no registered adapter in the `adapters` map. Drop 4c
// ships CLIKindClaude only; Drop 4d adds CLIKindCodex by registering an
// additional entry. Callers detect via errors.Is and route the action item
// to a "no agent configured" failure (same disposition as ErrNoAgentBinding).
var ErrUnsupportedCLIKind = errors.New("dispatcher: unsupported CLIKind")

// BundleRenderFunc is the signature of the per-spawn bundle-render hook
// the claude adapter's render package registers via init(). Drop 4c
// F.7-CORE F.7.3b ships the seam:
//
//   - render package imports dispatcher (for Bundle / BindingResolved /
//     domain types).
//   - dispatcher CANNOT import render directly (would form a cycle).
//   - render package's init() calls RegisterBundleRenderFunc to inject
//     itself; BuildSpawnCommand looks the hook up at spawn time.
//
// The seam mirrors the CLIAdapter registry pattern (RegisterAdapter +
// lookupAdapter) — same import-cycle resolution, same concurrency
// primitives, same test-substitution affordance.
//
// The hook returns the rendered system-prompt body alongside any error
// so BuildSpawnCommand can mirror the body into SpawnDescriptor.Prompt
// without re-reading from disk. On error the body is the empty string.
//
// When no render function is registered (e.g. the dispatcher boots
// without the cli_claude/render side-effect import) BuildSpawnCommand
// returns ErrNoBundleRenderFunc so callers see a clean failure rather
// than a missing system-prompt.md file.
type BundleRenderFunc func(
	bundle Bundle,
	item domain.ActionItem,
	project domain.Project,
	binding BindingResolved,
) (string, error)

// renderMu guards bundleRenderFunc. RegisterBundleRenderFunc is rare
// (init-time once); BuildSpawnCommand reads the hook on every spawn,
// which is hot — RWMutex matches the read-heavy access pattern.
var renderMu sync.RWMutex

// bundleRenderFunc holds the registered per-spawn render hook. nil
// when no render package has been imported for side-effects;
// BuildSpawnCommand surfaces that as ErrNoBundleRenderFunc.
var bundleRenderFunc BundleRenderFunc

// RegisterBundleRenderFunc wires fn into the spawn pipeline as the
// per-spawn bundle-render hook. The cli_claude/render package calls
// this from init() so production binaries that side-effect-import the
// package see the hook populated before the dispatcher dispatches
// anything.
//
// Repeat registrations under the same call overwrite — last writer
// wins. Tests use this to substitute their own render hook for fault
// injection (e.g. simulating a render failure in BuildSpawnCommand
// integration tests).
func RegisterBundleRenderFunc(fn BundleRenderFunc) {
	renderMu.Lock()
	defer renderMu.Unlock()
	bundleRenderFunc = fn
}

// lookupBundleRenderFunc returns the registered render hook, or nil +
// false when no render package has been imported.
func lookupBundleRenderFunc() (BundleRenderFunc, bool) {
	renderMu.RLock()
	defer renderMu.RUnlock()
	if bundleRenderFunc == nil {
		return nil, false
	}
	return bundleRenderFunc, true
}

// ErrNoBundleRenderFunc is returned by BuildSpawnCommand when no
// render hook has been registered via RegisterBundleRenderFunc.
// Production wiring side-effect-imports cli_claude/render at
// process start; if a build path skips that import, this error
// surfaces at the first spawn rather than a confusing
// "missing system-prompt.md" downstream failure.
var ErrNoBundleRenderFunc = errors.New("dispatcher: no bundle render function registered")

// adaptersMu guards adaptersMap. RegisterAdapter / lookupAdapter are
// concurrency-safe so wiring code can populate the registry from package
// init() in any order, and tests can register/swap adapters without races.
var adaptersMu sync.RWMutex

// adaptersMap maps CLIKind to its adapter implementation. Drop 4c F.7.17.5
// ships CLIKindClaude only; Drop 4d adds CLIKindCodex via:
//
//	dispatcher.RegisterAdapter(dispatcher.CLIKindCodex, cli_codex.New())
//
// The map is populated at process start by a wiring package that imports
// both dispatcher and the per-CLI adapter packages — this indirection breaks
// the import cycle that would otherwise form (cli_claude already imports
// dispatcher for the BindingResolved / BundlePaths / CLIAdapter types).
//
// Production wiring lives at internal/app/dispatcher/cli_register; cmd/till
// imports it for side-effects so the registry is populated before the
// dispatcher dispatches anything. Test code that drives BuildSpawnCommand
// directly does the same import.
var adaptersMap = map[CLIKind]CLIAdapter{}

// RegisterAdapter wires `adapter` into the dispatcher's CLIKind→adapter
// registry under `kind`. Wiring packages (cli_register today; per-CLI
// register packages in future drops) call this at init() time. Repeat
// registrations under the same kind overwrite — last writer wins. The
// concurrency primitives are conservative: registration is rare and lookup
// is hot, so a sync.RWMutex around a plain map is the right shape.
//
// Adapters MUST be safe for concurrent calls — the dispatcher reuses one
// instance per CLIKind across all spawns. The adapter implementations
// shipped today (cli_claude) hold no state, so this is automatically true.
func RegisterAdapter(kind CLIKind, adapter CLIAdapter) {
	adaptersMu.Lock()
	defer adaptersMu.Unlock()
	adaptersMap[kind] = adapter
}

// lookupAdapter returns the registered adapter for `kind`, or nil + false
// when no adapter is registered. BuildSpawnCommand wraps the false case as
// ErrUnsupportedCLIKind.
func lookupAdapter(kind CLIKind) (CLIAdapter, bool) {
	adaptersMu.RLock()
	defer adaptersMu.RUnlock()
	a, ok := adaptersMap[kind]
	return a, ok
}

// BuildSpawnCommand assembles one *exec.Cmd that — when later executed by the
// process monitor in 4a.21 — launches a subagent for the supplied action
// item. This function ONLY constructs the Cmd; it does not Start, Run, or
// Wait.
//
// Drop 4c F.7.17.5 multi-adapter wiring:
//
//  1. Validate caller-supplied inputs (preserved from the 4a.19 stub).
//  2. Look up the AgentBinding in catalog. ErrNoAgentBinding on miss.
//  3. ResolveBinding(rawBinding) → BindingResolved (with the F.7.17 L15
//     default-to-claude rule applied for empty CLIKind).
//  4. Look up the adapter from `adapters` keyed by resolved.CLIKind.
//     ErrUnsupportedCLIKind on miss.
//  5. Materialize the per-spawn bundle via F.7-CORE F.7.1 NewBundle, write
//     the cross-CLI manifest.json (spawn_id / action_item_id / kind /
//     started_at / paths). Today the spawn_temp_root is hardcoded to "" so
//     NewBundle resolves to "os_tmp" mode; a follow-up droplet plumbs
//     catalog.Tillsyn.SpawnTempRoot through so adopters can flip to
//     project-mode bundles via template TOML without code changes.
//  6. Render the system-prompt body and write it to
//     BundlePaths.SystemPromptPath. The body carries action-item structural
//     fields (task_id, project_id, project_dir, kind, title, paths,
//     packages, move-state directive) but NOT hylla_artifact_ref — Hylla is
//     a dev-local tool, not part of Tillsyn's shipped cascade per F.7.10.
//  7. Call adapter.BuildCommand(ctx, resolved, bundlePaths) to assemble the
//     CLI-specific argv + cmd.Env, and set cmd.Dir to the project worktree.
//  8. Build SpawnDescriptor mirroring the resolved fields and bundle-derived
//     MCP config path (the path claude actually wires under the bundle).
//
// Returns ErrNoAgentBinding wrapped with the offending kind when the catalog
// has no entry for item.Kind. Returns ErrInvalidSpawnInput wrapped with a
// reason for empty / malformed inputs. Returns templates.ErrInvalidAgentBinding
// (via Validate) when a corrupted binding survived template-load. Returns
// ErrUnsupportedCLIKind wrapped with the offending kind when the resolved
// CLIKind has no registered adapter.
//
// Context handling: the 4a.19 signature predates F.7.17's context-aware
// adapter contract, so this function uses context.Background() internally.
// TODO(F.7-CORE): plumb a real ctx parameter through BuildSpawnCommand once
// dispatcher.Dispatch can pass its outer ctx through stage 6.
func BuildSpawnCommand(
	item domain.ActionItem,
	project domain.Project,
	catalog templates.KindCatalog,
	authBundle AuthBundle,
) (*exec.Cmd, SpawnDescriptor, error) {
	if strings.TrimSpace(item.ID) == "" {
		return nil, SpawnDescriptor{}, fmt.Errorf("%w: action item ID is empty", ErrInvalidSpawnInput)
	}
	if item.Kind == "" {
		return nil, SpawnDescriptor{}, fmt.Errorf("%w: action item kind is empty", ErrInvalidSpawnInput)
	}
	if strings.TrimSpace(project.RepoPrimaryWorktree) == "" {
		return nil, SpawnDescriptor{}, fmt.Errorf(
			"%w: project.RepoPrimaryWorktree is empty (project=%s)",
			ErrInvalidSpawnInput, project.ID,
		)
	}

	rawBinding, ok := catalog.LookupAgentBinding(item.Kind)
	if !ok {
		return nil, SpawnDescriptor{}, fmt.Errorf("%w: kind %q", ErrNoAgentBinding, item.Kind)
	}
	// Defensive re-validate. Validate already runs at template-load (per
	// schema.go) so a corrupted binding only reaches here via direct
	// in-memory mutation (tests use this to assert empty-AgentName trips a
	// loud failure rather than silently emitting an empty `--agent` value).
	if err := rawBinding.Validate(); err != nil {
		return nil, SpawnDescriptor{}, fmt.Errorf("dispatcher: invalid agent binding for kind %q: %w", item.Kind, err)
	}

	// Drop 4c F.7-CORE F.7.6 pre-dispatch plugin pre-flight. Resolve the
	// project's required-plugin list via the package-level injection hook
	// (RequiredPluginsForProject), then verify every required entry is
	// installed locally. A nil hook OR an empty list short-circuits before
	// invoking the lister so adopters with no required plugins pay no exec
	// cost per spawn. ErrMissingRequiredPlugins (and lister-side failures
	// like ErrClaudeBinaryMissing) propagate through BuildSpawnCommand;
	// callers in dispatcher.RunOnce wrap the failure the same way they wrap
	// any other spawn-construction error.
	//
	// The hook seam exists because KindCatalog (the per-project baked
	// snapshot fed to BuildSpawnCommand) does NOT carry the [tillsyn]
	// globals today — only Kinds + AgentBindings. A future droplet that
	// extends KindCatalog with a `RequiresPlugins []string` field will
	// populate the hook from catalog.RequiresPlugins; until then the hook
	// remains a nil seam adopters can populate at process boot via direct
	// assignment to RequiredPluginsForProject.
	if hook := RequiredPluginsForProject; hook != nil {
		required := hook(project)
		if err := CheckRequiredPlugins(context.Background(), defaultClaudePluginLister, required); err != nil {
			return nil, SpawnDescriptor{}, fmt.Errorf("dispatcher: plugin pre-flight for kind %q: %w", item.Kind, err)
		}
	}

	// Resolve the binding through the priority cascade. F.7.17.5 does NOT
	// plumb CLI/MCP/TUI overrides — those layers grow knobs in later
	// droplets. Today only the rawBinding fields contribute, with the
	// F.7.17 L15 default-to-claude substitution applied to CLIKind.
	resolved := ResolveBinding(rawBinding)

	adapter, ok := lookupAdapter(resolved.CLIKind)
	if !ok {
		return nil, SpawnDescriptor{}, fmt.Errorf("%w: %q", ErrUnsupportedCLIKind, resolved.CLIKind)
	}

	// Per-spawn bundle materialization via F.7-CORE F.7.1's NewBundle. The
	// dispatcher passes the empty string for spawnTempRoot today because
	// templates.KindCatalog does not yet carry the Tillsyn block — a
	// follow-up droplet plumbs catalog.Tillsyn.SpawnTempRoot through here
	// so adopters can flip to project-mode bundles via template TOML
	// without further code changes. Empty string resolves to
	// SpawnTempRootOSTmp inside NewBundle, preserving the 4a.19 / F.7.17.5
	// behavior byte-for-byte.
	//
	// project.RepoPrimaryWorktree is passed unconditionally so the
	// project-mode codepath has the worktree available the moment the
	// catalog plumbing lands; in os_tmp mode NewBundle ignores it.
	//
	// Bundle.WriteManifest writes the cross-CLI manifest (spawn_id,
	// action_item_id, kind, started_at, paths) so F.7.8's orphan scan can
	// correlate bundles back to action items even if the spawn crashes
	// before the monitor's terminal-state observer fires. Failure to write
	// the manifest is non-fatal here only because the F.7.8 orphan scan
	// will treat a manifest-absent bundle as orphaned and reap it; we
	// surface the error to the caller for diagnostic visibility.
	bundle, err := NewBundle(item, "", project.RepoPrimaryWorktree)
	if err != nil {
		return nil, SpawnDescriptor{}, fmt.Errorf("dispatcher: create spawn bundle: %w", err)
	}
	if err := bundle.WriteManifest(ManifestMetadata{
		SpawnID:      bundle.SpawnID,
		ActionItemID: item.ID,
		Kind:         item.Kind,
		StartedAt:    bundle.StartedAt,
		Paths:        item.Paths,
	}); err != nil {
		// Cleanup the half-materialized bundle so we don't leak.
		_ = bundle.Cleanup()
		return nil, SpawnDescriptor{}, fmt.Errorf("dispatcher: write spawn manifest: %w", err)
	}
	bundlePaths := bundle.Paths

	// Render the per-spawn bundle subtree (system-prompt.md cross-CLI +
	// claude-specific plugin/ subtree per spawn architecture memory §2)
	// via the registered hook. The cli_claude/render package wires itself
	// in at init time; production binaries side-effect-import the package
	// alongside cli_claude itself. F.7.17.5's provisional minimal prompt
	// block is REPLACED by this hook — Render owns system-prompt.md from
	// here forward and additionally writes plugin.json / agents/<name>.md
	// / .mcp.json / settings.json under <bundle.Root>/plugin/.
	render, ok := lookupBundleRenderFunc()
	if !ok {
		_ = bundle.Cleanup()
		return nil, SpawnDescriptor{}, fmt.Errorf("%w (kind=%q)", ErrNoBundleRenderFunc, item.Kind)
	}
	prompt, err := render(bundle, item, project, resolved)
	if err != nil {
		_ = bundle.Cleanup()
		return nil, SpawnDescriptor{}, fmt.Errorf("dispatcher: render spawn bundle: %w", err)
	}

	// TODO(F.7-CORE): replace context.Background() with the outer dispatcher
	// ctx so cancellation propagates through the spawned process tree.
	cmd, err := adapter.BuildCommand(context.Background(), resolved, bundlePaths)
	if err != nil {
		return nil, SpawnDescriptor{}, fmt.Errorf("dispatcher: adapter build command: %w", err)
	}
	cmd.Dir = project.RepoPrimaryWorktree

	// MCPConfigPath mirrors what claude's argv builder wires:
	// <bundleRoot>/plugin/.mcp.json. Hardcoding the subpath here couples
	// spawn.go to claude's bundle layout, which is acceptable in Drop 4c
	// because claude is the only registered adapter and the descriptor's
	// MCPConfigPath field IS surfaced via `till dispatcher run --dry-run`'s
	// JSON output. F.7-CORE F.7.1 will lift this onto the bundle materializer
	// so future adapters (codex) can publish their own MCP config path.
	mcpConfigPath := filepath.Join(bundlePaths.Root, "plugin", ".mcp.json")

	descriptor := SpawnDescriptor{
		AgentName:     resolved.AgentName,
		Model:         derefString(resolved.Model),
		MaxBudgetUSD:  derefFloat64(resolved.MaxBudgetUSD),
		MaxTurns:      derefInt(resolved.MaxTurns),
		MCPConfigPath: mcpConfigPath,
		Prompt:        prompt,
		WorkingDir:    project.RepoPrimaryWorktree,
	}

	return cmd, descriptor, nil
}

// derefString returns *p when p is non-nil, else "". ResolveBinding ALWAYS
// populates pointer-typed fields (it promotes the rawBinding scalar to a
// pointer copy on no-override), so derefString never returns "" via the nil
// branch in production — the helper exists for defensive symmetry with
// derefInt / derefFloat64.
func derefString(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

// derefInt returns *p when p is non-nil, else 0.
func derefInt(p *int) int {
	if p == nil {
		return 0
	}
	return *p
}

// derefFloat64 returns *p when p is non-nil, else 0.
func derefFloat64(p *float64) float64 {
	if p == nil {
		return 0
	}
	return *p
}

// assemblePrompt is removed in Drop 4c F.7-CORE F.7.3b. The per-spawn
// system-prompt body now ships from cli_claude/render's Render function
// — registered into the dispatcher via RegisterBundleRenderFunc and
// consumed by BuildSpawnCommand's render-hook lookup. Adopters who want
// custom per-kind prompt templates use the binding's
// SystemPromptTemplatePath field (F.7.2) which will plumb through
// render in a follow-up droplet.
