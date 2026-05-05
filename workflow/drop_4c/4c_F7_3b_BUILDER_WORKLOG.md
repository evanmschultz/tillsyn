# Drop 4c F.7-CORE F.7.3b Builder Worklog

## Round 1 — Bundle render landed

**Scope.** Drop 4c F.7-CORE F.7.3b — bundle render. Replaces F.7.17.5's
provisional minimal-prompt-only block in `BuildSpawnCommand` with a full
five-artifact render under the per-spawn bundle directory:

1. `<bundle.Root>/system-prompt.md` (cross-CLI per memory §2)
2. `<bundle.Root>/plugin/.claude-plugin/plugin.json`
3. `<bundle.Root>/plugin/agents/<binding.AgentName>.md`
4. `<bundle.Root>/plugin/.mcp.json`
5. `<bundle.Root>/plugin/settings.json`

Per REVISIONS POST-AUTHORING:

- **REV-2** — F.7.3 split into F.7.3a + F.7.3b; this droplet ships ONLY F.7.3b's
  bundle render piece.
- **REV-15** — F.7.3a was absorbed into F.7.17.3 (`assembleArgv` already ships
  the headless argv recipe). This droplet does NOT touch argv.

## Files

### New
- `/Users/evanschultz/Documents/Code/hylla/tillsyn/main/internal/app/dispatcher/cli_claude/render/render.go`
  — `Render` function + 5 sub-renderers (`renderSystemPrompt`,
  `renderPluginManifest`, `renderAgentFile`, `renderMCPConfig`,
  `renderSettings`) + rollback helper. Returns
  `(systemPromptBody string, err error)` so the caller (BuildSpawnCommand)
  can populate `SpawnDescriptor.Prompt` without a second disk read.
- `/Users/evanschultz/Documents/Code/hylla/tillsyn/main/internal/app/dispatcher/cli_claude/render/render_test.go`
  — 11 unit tests covering happy path + rollback + path-injection guards
  + empty-binding edge cases.
- `/Users/evanschultz/Documents/Code/hylla/tillsyn/main/internal/app/dispatcher/cli_claude/render/init.go`
  — registers `Render` with the dispatcher's bundle-render hook seam at
  package import time.

### Edited
- `/Users/evanschultz/Documents/Code/hylla/tillsyn/main/internal/app/dispatcher/spawn.go`
  — added `BundleRenderFunc` registration seam (`RegisterBundleRenderFunc` /
  `lookupBundleRenderFunc` + `ErrNoBundleRenderFunc`); replaced the
  F.7.17.5 `assemblePrompt` + `os.WriteFile` block with a
  `lookupBundleRenderFunc()` invocation; removed `assemblePrompt`; dropped
  the `os` import.
- `/Users/evanschultz/Documents/Code/hylla/tillsyn/main/internal/app/dispatcher/cli_claude/init.go`
  — added blank-import of `cli_claude/render` so `cli_claude` side-effect
  imports automatically wire BOTH the adapter AND the render hook.
- `/Users/evanschultz/Documents/Code/hylla/tillsyn/main/internal/app/dispatcher/spawn_test.go`
  — added `TestBuildSpawnCommandRendersFullBundleSubtree` (asserts every
  bundle artifact exists post-BuildSpawnCommand); added
  `TestBuildSpawnCommandRenderHookFailureCleansUpBundle` (fault-injection
  test substituting a faulty render hook + asserting bundle cleanup);
  added named import `clauderender` for hook restoration.

## Architectural Notes

### Import-cycle resolution (registration seam)

`render` package imports `dispatcher` (for `Bundle`, `BindingResolved`,
`BundlePaths` types + the `domain.ActionItem` / `domain.Project` types
flow through it). `dispatcher.BuildSpawnCommand` calls into render. Direct
import would form a cycle.

Mirroring the existing `RegisterAdapter` pattern: dispatcher declares a
`BundleRenderFunc` type + `RegisterBundleRenderFunc(fn)` + private
`lookupBundleRenderFunc()`. `render/init.go` calls the register at package
import time; `BuildSpawnCommand` looks up the hook on every spawn.

`cli_claude/init.go` blank-imports `cli_claude/render`, so anyone
side-effect-importing `cli_claude` (cmd/till, dispatcher tests) gets BOTH
the adapter and the render hook wired in. No new top-level wiring needed
elsewhere.

### Permissions JSON shape

Per memory §4 (verbatim from claude docs + probes), settings.json carries:

```json
{
  "permissions": {
    "allow": [...],
    "ask":   [...],
    "deny":  [...]
  }
}
```

Render emits all three keys with explicit empty arrays when the binding has
no entries. `permissions.allow` mirrors `binding.ToolsAllowed`;
`permissions.deny` mirrors `binding.ToolsDisallowed`. `permissions.ask` is
unused today — F.7.5b (TUI handshake) populates it with stored grants in
a future droplet. nil-slice → `[]string{}` substitution ensures explicit
JSON `[]` rather than `null`, more debuggable when a dev opens the file.

### Sandbox emission deferred

The droplet acceptance-criteria mentioned "Include sandbox.filesystem/network
if `binding.Sandbox.*` populated." However, `dispatcher.BindingResolved`
does NOT today carry the `Sandbox` sub-struct from `templates.AgentBinding`
— extending `BindingResolved` is out of scope for F.7.3b (would need a
companion change to `ResolveBinding` in `binding_resolved.go`). Render
currently emits `permissions` only; a follow-up droplet wires the sandbox
fields onto `BindingResolved` and into `settingsFile`. The struct shape in
`render.go` is documented to make the deferred extension obvious.

REV-11 worktree-escape check — F.7.2 deferred this to spawn-time. Render
today does NOT enforce the AllowWrite-escape-from-worktree check; that
lands when the sandbox emission lands and Render gets the project worktree
path threaded through.

### Agent file shape

`agents/<name>.md` ships the canonical Tillsyn agent frontmatter shape:

- `name: <binding.AgentName>` (load-bearing for claude)
- `description: <stub>` (load-bearing for claude)
- `allowedTools: <ToolsAllowed>` (Layer A per memory §5; mirror of
  `permissions.allow` for human readability)
- `disallowedTools: <ToolsDisallowed>` (Layer A; mirror of
  `permissions.deny`)
- Body: pointer to canonical agent template at the system-installed
  plugin path (Path B per memory §1).

The full canonical agent behavior at `~/.claude/agents/<name>.md` remains
the source of truth — claude loads it from the system-installed plugin
path. The per-spawn bundle ships a minimal stub that names the agent +
emits the per-spawn tool-gating layer A.

### Rollback behavior

Render owns ONLY `system-prompt.md` + the `plugin/` subtree under the
bundle root. F.7.1's `manifest.json` is the caller's concern. On any
sub-renderer failure, Render's rollback handle wipes both via
`os.Remove(<root>/system-prompt.md)` + `os.RemoveAll(<root>/plugin)` —
best-effort, errors swallowed because the caller is already returning
non-nil. Manifest stays put for orphan-scan correlation.

Tested via `TestRenderRollbackOnAgentDirFailure` (POSIX-only; planted file
at `<root>/plugin/agents` blocks the agents-dir mkdir, asserting cleanup
wipes both pre-existing render output and the planted blocker).

## Test scenarios (8 acceptance + 6 additional unit)

Acceptance criteria scenarios (all green):

1. **Happy path** — `Render` writes all 5 files; returns `nil`. Pinned in
   `TestRenderHappyPathWritesAllFiveFiles`.
2. **System-prompt content** — body carries action_item_id, project_id,
   project_dir, kind, title, paths, packages; F.7.10 negative on
   `hylla_artifact_ref`. Pinned in
   `TestRenderSystemPromptContainsStructuralTokens`.
3. **plugin.json shape** — exactly `{"name": "spawn-<id>"}`. Pinned in
   `TestRenderPluginManifestExactShape`.
4. **.mcp.json shape** — exactly `{"tillsyn": {"command": "till", "args":
   ["serve-mcp"]}}`. Pinned in `TestRenderMCPConfigExactShape`.
5. **settings.json permissions** — `permissions.allow` mirrors
   `binding.ToolsAllowed`; `permissions.deny` mirrors
   `binding.ToolsDisallowed`. Pinned in `TestRenderSettingsPermissions`.
6. **Agent file rendered** — `agents/<name>.md` non-empty with frontmatter.
   Pinned in `TestRenderAgentFileFrontmatter`.
7. **Failure rollback** — faulty pre-existing file blocks render; no
   partial residue. Pinned in `TestRenderRollbackOnAgentDirFailure`.
8. **Spawn integration** — `BuildSpawnCommand` produces full subtree. Pinned
   in `TestBuildSpawnCommandRendersFullBundleSubtree`.

Additional unit coverage:

9. `TestRenderSettingsExplicitEmptyArraysWhenBindingEmpty` — nil-slice →
   `[]` JSON substitution.
10. `TestRenderAgentFileWithoutToolGating` — frontmatter omits
    allowedTools/disallowedTools when binding has no entries.
11. `TestRenderRejectsEmptyBundleRoot`,
    `TestRenderRejectsEmptyAgentName`,
    `TestRenderRejectsAgentNameWithPathSeparator` — input-validation
    guards.
12. `TestRenderOmitsOptionalSystemPromptFields` — title/paths/packages
    lines omitted when not declared.
13. `TestBuildSpawnCommandRenderHookFailureCleansUpBundle` — dispatcher
    failure-path: faulty render → bundle wholesale cleanup.

## Verification

`mage check` — green. 2608 tests passed across 24 packages, 1 skipped
(pre-existing, unrelated). Dispatcher package coverage 74.1%, new render
package coverage 86.2%, both above 70% threshold.

`mage ci` — green. Same 2608 / 24 / 1 figures, plus build artifact
produced and format check + lint clean.

## Hylla Feedback

N/A — task touched non-Go and Go files but I did not invoke Hylla
(per the spawn prompt's "NO Hylla calls" hard constraint).

## Single-line conventional commit (orchestrator commits)

```
feat(dispatcher): land per-spawn bundle render for claude adapter
```
