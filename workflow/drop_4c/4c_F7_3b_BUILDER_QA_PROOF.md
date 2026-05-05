# Drop 4c F.7-CORE F.7.3b — Builder QA Proof Review

## Round 1

### Verdict: PROOF GREEN

All 12 prompted checks pass. No GAPs. One advisory NIT logged on out-of-scope `SKETCH.md` edit (not blocking).

---

### Evidence by check

#### Check 1 — `Render` exists with 5 sub-renderers

`internal/app/dispatcher/cli_claude/render/render.go:83-134` declares `Render(bundle, item, project, binding) (string, error)` with the exact signature specified. The five sub-renderers are present:

- `renderSystemPrompt` (`render.go:175-181`)
- `renderPluginManifest` (`render.go:249-260`)
- `renderAgentFile` (`render.go:281-288`)
- `renderMCPConfig` (`render.go:346-362`)
- `renderSettings` (`render.go:402-419`)

Each is invoked in order from `Render` (`render.go:103-131`) with rollback-on-error semantics. PASS.

#### Check 2 — `BundleRenderFunc` registration seam in `spawn.go`

Mirrors `RegisterAdapter` pattern verbatim:

- Type: `BundleRenderFunc` (`spawn.go:121-126`).
- Mutex: `renderMu sync.RWMutex` (`spawn.go:131`) + module-level `bundleRenderFunc BundleRenderFunc` (`spawn.go:136`).
- Setter: `RegisterBundleRenderFunc(fn)` (`spawn.go:148-152`).
- Getter: `lookupBundleRenderFunc()` (`spawn.go:156-163`).
- Sentinel: `ErrNoBundleRenderFunc` (`spawn.go:171`).

Concurrency model matches `adaptersMu` / `RegisterAdapter` / `lookupAdapter` (`spawn.go:176-218`): RWMutex around plain map/var, last-writer-wins on register. PASS.

#### Check 3 — Hook ordering: WriteManifest → render → BuildCommand

In `BuildSpawnCommand` (`spawn.go:260-410`):

- Bundle creation + manifest write: `spawn.go:346-360` (`NewBundle` → `bundle.WriteManifest`).
- Render hook lookup + invocation: `spawn.go:371-380` (`lookupBundleRenderFunc()` → `render(bundle, item, project, resolved)`).
- Adapter `BuildCommand`: `spawn.go:384` (`adapter.BuildCommand(context.Background(), resolved, bundlePaths)`).

Ordering verified. Failure cleanup: render-hook miss (`spawn.go:372-375`) and render error (`spawn.go:377-380`) both call `bundle.Cleanup()` then return — no leakage. PASS.

#### Check 4 — System-prompt content + no `hylla_artifact_ref`

`assembleSystemPromptBody` (`render.go:201-234`) emits:

- `task_id` (item.ID, `render.go:203-205`)
- `project_id` (`render.go:206-208`)
- `project_dir` (project.RepoPrimaryWorktree, `render.go:209-211`)
- `kind` (`render.go:212-214`)
- Optional `title` (`render.go:215-219`)
- Optional `paths` (`render.go:220-224`)
- Optional `packages` (`render.go:225-229`)
- `move-state directive:` block (`render.go:230-232`)

Function does NOT reference `project.HyllaArtifactRef` or any `Hylla*` field. Confirmed by full read of render.go — only doc-comment mentions of "Hylla" appear (`render.go:172-174`), none in code paths.

Test `TestRenderSystemPromptContainsStructuralTokens` (`render_test.go:112-152`) pins all 8 expected tokens AND asserts `!strings.Contains(bodyStr, "hylla_artifact_ref")` (`render_test.go:149-151`). PASS.

#### Check 5 — `settings.json` permissions + explicit `[]string{}`

`settingsFile` struct (`render.go:374-380`) has single `Permissions permissionsBlock` field. `permissionsBlock` (`render.go:386-390`) declares `Allow / Ask / Deny []string` — all three keys with no `omitempty` tags, so all three serialize unconditionally.

`renderSettings` (`render.go:402-419`) populates:

- `Allow: nonNilStringSlice(binding.ToolsAllowed)` (`render.go:409`)
- `Ask: []string{}` (`render.go:410` — explicit literal)
- `Deny: nonNilStringSlice(binding.ToolsDisallowed)` (`render.go:411`)

`nonNilStringSlice` (`render.go:424-429`): nil → `[]string{}`, non-nil → passthrough. Test `TestRenderSettingsExplicitEmptyArraysWhenBindingEmpty` (`render_test.go:267-297`) parses raw JSON and asserts `"allow": []` + `"deny": []` + absence of `null`. PASS.

#### Check 6 — `plugin.json` exact shape

`pluginManifest` struct (`render.go:240-246`) has single `Name string` field with `json:"name"` tag. `renderPluginManifest` (`render.go:249-260`) builds `pluginManifest{Name: "spawn-" + bundle.SpawnID}` and writes it to `<root>/plugin/.claude-plugin/plugin.json`.

Test `TestRenderPluginManifestExactShape` (`render_test.go:157-182`) round-trips through `json.Unmarshal` into `map[string]any` and asserts exactly 1 key with name = `"spawn-" + bundle.SpawnID`. PASS.

#### Check 7 — `.mcp.json` exact shape

`mcpConfig` (`render.go:325-332`) declares `Tillsyn mcpServerEntry json:"tillsyn"`. `mcpServerEntry` (`render.go:336-343`) declares `Command string` + `Args []string`. `renderMCPConfig` (`render.go:346-362`) builds `mcpConfig{Tillsyn: mcpServerEntry{Command: "till", Args: []string{"serve-mcp"}}}`.

Test `TestRenderMCPConfigExactShape` (`render_test.go:186-216`) parses and asserts `Command == "till"` and `Args == ["serve-mcp"]`. PASS.

#### Check 8 — Agent file rendered + path-traversal guard

`renderAgentFile` (`render.go:281-288`) writes `<root>/plugin/agents/<binding.AgentName>.md`. Body assembled by `assembleAgentFileBody` (`render.go:295-319`) carries `name`, `description`, optional `allowedTools`, optional `disallowedTools` frontmatter + non-empty body line.

Path-traversal guard: `Render` (`render.go:95-98`) rejects `binding.AgentName` containing `/` or `\` via `strings.ContainsAny(binding.AgentName, "/\\")`. Tests:

- `TestRenderAgentFileFrontmatter` (`render_test.go:303-332`) — happy path with allowed/disallowed tools.
- `TestRenderAgentFileWithoutToolGating` (`render_test.go:338-368`) — empty-binding branch.
- `TestRenderRejectsAgentNameWithPathSeparator` (`render_test.go:458-481`) — exercises `../../etc/passwd`, backslash, simple `a/b` cases; all return `errors.Is(err, render.ErrInvalidRenderInput)`.
- `TestRenderRejectsEmptyAgentName` (`render_test.go:438-452`).

PASS.

#### Check 9 — Failure rollback

`renderRollback` (`render.go:143-162`): tracks bundle root, on `run()` calls `os.Remove(<root>/system-prompt.md)` + `os.RemoveAll(<root>/plugin)` — best-effort cleanup. Each sub-renderer error-branch in `Render` (`render.go:103-131`) calls `rollback.run()` before returning the wrapped error.

`renderRollback.run` does NOT touch `manifest.json` (correct — that is the dispatcher's responsibility per `render.go:155-156` doc comment). Dispatcher's `bundle.Cleanup()` invocation in `spawn.go:378` covers the manifest.json removal on render-error.

Test `TestRenderRollbackOnAgentDirFailure` (`render_test.go:376-415`) plants a regular file at `<root>/plugin/agents` (blocking the `os.MkdirAll` in `renderAgentFile`), runs Render, asserts non-nil error AND that both `<root>/system-prompt.md` and `<root>/plugin/` are gone afterwards. PASS.

#### Check 10 — Scope: 7 files, no F.7.4 attribution

`git status --short` working-tree shows F.7.3b files:

- NEW `internal/app/dispatcher/cli_claude/render/` (3 files: render.go, render_test.go, init.go)
- MODIFIED `internal/app/dispatcher/cli_claude/init.go` (1-line blank-import add — scope-expansion #2)
- MODIFIED `internal/app/dispatcher/spawn.go`
- MODIFIED `internal/app/dispatcher/spawn_test.go`
- NEW `workflow/drop_4c/4c_F7_3b_BUILDER_WORKLOG.md`

Total = 7 files matching the prompt's scope envelope.

F.7.4 attribution check: `internal/app/dispatcher/monitor.go` and `monitor_test.go` are untracked-or-modified (227 + 528 inserted lines per `git diff --stat`). These are F.7.4 sibling work — confirmed by the existence of `workflow/drop_4c/4c_F7_4_BUILDER_WORKLOG.md` as a separate artifact. NOT attributed to F.7.3b.

ADVISORY NIT (non-blocking): `git status` also shows `workflow/drop_4c/SKETCH.md` modified. Not in F.7.3b's declared scope envelope; likely orchestrator-side or sibling-droplet edit. The change does not affect any F.7.3b verification check. Recommend orchestrator note this in closeout if it was meant to be touched here.

PASS (with advisory NIT).

#### Check 11 — `mage ci` green

Independently re-ran `mage ci` from the worktree root. Output:

- `mage test-pkg ./internal/app/dispatcher/cli_claude/render`: 16 tests passed (the 11 declared `Test*` functions including the 3-case `TestRenderRejectsAgentNameWithPathSeparator` subtest expansion = 16 t.Run-rooted leaves).
- `mage test-pkg ./internal/app/dispatcher`: 265 tests passed.
- `mage ci`: full sweep green. 1 skip (pre-existing `TestStewardIntegrationDropOrchSupersedeRejected` in `mcpapi` — unrelated). 24 packages compiled, format clean, lint clean, build artifact produced.

Coverage observed:

- `internal/app/dispatcher/cli_claude/render`: **86.2%** (matches worklog claim).
- `internal/app/dispatcher`: **74.1%** (matches worklog claim).

Both above the 70% gate. PASS.

#### Check 12 — No commit by builder

`git log --oneline -3 -- internal/app/dispatcher/cli_claude/render/` returns empty — no commit exists touching the render package. Most-recent HEAD commit is `ad040b9 feat(dispatcher): per-spawn bundle lifecycle and plugin preflight` from prior droplet F.7.1.

REV-13 honored: builder left work uncommitted for orchestrator commit.

PASS.

---

### Falsification attacks (all mitigated)

1. **"Render leaks partial files on mid-write failure."** Mitigated by `renderRollback.run()` (`render.go:156-162`) called on every sub-renderer error branch; pinned by `TestRenderRollbackOnAgentDirFailure`.
2. **"`hylla_artifact_ref` leaks in some sub-renderer other than system-prompt."** Mitigated by full read of render.go — no `Hylla*` field reference in any code path; `project.HyllaArtifactRef` is read only in the test fixture (to set up the negative assertion) and never propagates to written output.
3. **"Path-traversal guard too narrow."** Guard rejects forward + back slash; tests cover `..`-via-slash, backslash-injection, simple `a/b`. Null-byte / absolute-tilde injection NOT tested but cannot escape the agents dir without a separator. Acceptable scope.
4. **"settings.json nil-slice claim is wrong (json.Marshal turns nil into null)."** Mitigated by `nonNilStringSlice` (`render.go:424-429`) substitution + `Ask: []string{}` literal; pinned by `TestRenderSettingsExplicitEmptyArraysWhenBindingEmpty` which string-searches raw JSON for absence of `null`.
5. **"F.7.4 monitor.go work leaks into F.7.3b scope."** Mitigated by file inventory: render package contains only 3 files, no monitor*.go present. F.7.4 has its own builder worklog as a separate artifact.
6. **"7-file scope mismatch."** Mitigated by git status enumeration above — exactly 7 files in F.7.3b scope. SKETCH.md edit is advisory NIT.
7. **"Registration seam race condition."** Mitigated by `renderMu sync.RWMutex` (`spawn.go:131`) wrapping read + write paths.
8. **"86.2% coverage misses critical paths."** Direct test inventory: happy-path × 1, system-prompt content × 1, plugin.json × 1, .mcp.json × 1, settings.json × 2 (populated + nil), agent file × 2 (with + without tools), rollback × 1, input-validation × 3 (empty root, empty name, path-separator with 3 sub-cases), prompt-omission × 1. Plus 2 integration tests at the dispatcher boundary. Critical paths covered.

### Sandbox emission deferral — verified justified

Builder claims `BindingResolved` doesn't carry `Sandbox` so emission is deferred. Verified by `rg Sandbox` against `internal/app/dispatcher/cli_adapter.go` (where `BindingResolved` is defined) — empty result. The deferral is structurally enforced today; cannot be implemented without first extending `BindingResolved` + `ResolveBinding`. Acceptable.

---

## Hylla Feedback

N/A — Hylla calls were prohibited per the spawn-prompt hard constraint. All evidence gathered via Read / Bash (mage) / file inspection.

---

## TL;DR

- **T1**: PROOF GREEN — all 12 prompted checks PASS with file:line evidence. 1 advisory NIT on out-of-scope `SKETCH.md` edit (non-blocking).
- **T2**: `Render` + 5 sub-renderers shipped at `render.go:83-419`; registration seam mirrors `RegisterAdapter` exactly at `spawn.go:121-171`; hook order WriteManifest → render → BuildCommand verified at `spawn.go:346-388`.
- **T3**: System-prompt body carries all required structural tokens, NO `hylla_artifact_ref` (test pin at `render_test.go:149-151`); settings.json explicit `[]` substitution verified by `TestRenderSettingsExplicitEmptyArraysWhenBindingEmpty`; path-traversal guard rejects `/` + `\`.
- **T4**: `mage ci` independently green — render coverage 86.2%, dispatcher 74.1%, both above gate; no commit by builder per REV-13.
- **T5**: All 8 falsification attacks mitigated; sandbox-emission deferral verified justified via empty `rg Sandbox` against `cli_adapter.go`.
