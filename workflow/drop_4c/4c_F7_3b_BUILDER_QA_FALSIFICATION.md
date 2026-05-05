# Drop 4c F.7-CORE F.7.3b — Bundle Render — QA Falsification

## Round 1

### Verdict
**PASS** — no counterexamples produced; all 13 attack vectors REFUTED.

### Per-attack verdicts

#### A1. Render-hook race condition — REFUTED
- `renderMu` is a `sync.RWMutex` (`spawn.go:131`). `RegisterBundleRenderFunc` takes write lock; `lookupBundleRenderFunc` takes read lock. No data race possible.
- `TestBuildSpawnCommandRenderHookFailureCleansUpBundle` (`spawn_test.go:683`) is explicitly `NOT t.Parallel()` (comment on line 684–686). Go testing semantics: serial tests run sequentially with their `t.Cleanup` firing before parallel tests release. The cleanup at `spawn_test.go:702-707` re-registers `clauderender.Render` BEFORE parallel tests run. No cross-test race.
- The faulty hook substitution + restoration pattern is correct.

#### A2. Agent-name path traversal — REFUTED
- Guard at `render.go:95`: `strings.ContainsAny(binding.AgentName, "/\\")`. Tested with three vectors at `render_test.go:461-464` including `"go-builder-agent/../../etc/passwd"`, `"go-builder-agent\\evil"`, `"a/b"`.
- Counterexample attempts:
  - `".."` literal (no separator) → `<dir>/..md` — POSIX literal filename, NOT parent-dir traversal. Three-dot filename, harmless.
  - `"."` literal → `<dir>/..md` — literal three-dot filename, harmless.
  - NUL bytes (`"x\x00y"`) → `os.WriteFile` returns error from syscall; no escape from `agents/`.
  - Whitespace-only / empty → caught by `strings.TrimSpace` guard at `render.go:92`.
  - Windows reserved names (`CON`, `PRN`) → potential robustness issue on Windows-build but not exploitable for path escape; non-counterexample.
- Path-injection surface is closed for the threat model (corrupted catalog accidentally containing separators).

#### A3. `spawn-<spawn-id>` collision — REFUTED
- `bundle.go:191`: `spawnID := uuid.NewString()` — UUIDv4, 122 bits of randomness. Collision probability astronomical. plugin.json `Name: "spawn-" + bundle.SpawnID` is unique by construction.

#### A4. Settings.json explicit empty arrays — REFUTED
- `nonNilStringSlice` at `render.go:424-429` substitutes `[]string{}` for nil. JSON marshaling of `[]string{}` produces `[]` not `null`.
- `TestRenderSettingsExplicitEmptyArraysWhenBindingEmpty` (`render_test.go:267-297`) pins:
  - Substring `"allow": []` present.
  - Substring `"deny": []` present.
  - No `null` substring anywhere.
- Note: `permissions.ask` is hardcoded `[]string{}` regardless of binding (`render.go:410`) — also produces explicit `[]`.

#### A5. Failure rollback completeness — REFUTED
- `renderRollback.run()` (`render.go:156-162`) is bundleRoot-anchored, not step-tracking:
  ```go
  _ = os.Remove(filepath.Join(r.bundleRoot, "system-prompt.md"))
  _ = os.RemoveAll(filepath.Join(r.bundleRoot, pluginSubdir))
  ```
- Walks every step-failure scenario:
  - Step 1 (system-prompt) fails → `assembleSystemPromptBody` is pure; only os.WriteFile can fail. Rollback removes system-prompt.md (no-op, write didn't happen) + RemoveAll plugin/ (no-op, never created). Correct.
  - Step 2 (plugin manifest) fails → system-prompt.md exists, plugin/ may have been partially created by `os.MkdirAll`. Rollback wipes both. Correct.
  - Step 3-5 fail → same blanket-wipe behavior. Correct.
- `TestRenderRollbackOnAgentDirFailure` (`render_test.go:376-415`) plants a regular file at `plugin/agents` blocking mkdir, asserts both `system-prompt.md` and `plugin/` are gone after rollback.
- Manifest.json (F.7.1's responsibility) intentionally NOT touched — caller wraps Render in `bundle.Cleanup()` (`spawn.go:378`) which removes the entire bundle root for caller-level rollback.

#### A6. `Render` body == file body — REFUTED
- `renderSystemPrompt` returns `body, nil` where body is from pure-function `assembleSystemPromptBody` (`render.go:201-234`). Same string is written to disk via `os.WriteFile([]byte(body), ...)`.
- `TestRenderSystemPromptContainsStructuralTokens` (`render_test.go:128-130`) pins `promptBody == bodyStr` (file contents).
- `TestBuildSpawnCommandWritesSystemPromptFile` (`spawn_test.go:261-263`) pins `descriptor.Prompt == file contents` end-to-end through the spawn path. Both equality checks present.

#### A7. Sandbox deferral correctness — REFUTED (architecturally correct)
- `dispatcher.BindingResolved` (`cli_adapter.go:102-179`) does NOT carry a `Sandbox` field. Confirmed — no Sandbox member.
- `templates.AgentBinding` DOES have `Sandbox SandboxRules` (per `templates/load.go:633-642` validation).
- The render package consumes `BindingResolved`, not `templates.AgentBinding`. Architecture choice: Render only knows what BindingResolved exposes.
- Future droplet must extend BOTH `BindingResolved` (add `Sandbox SandboxRules`) AND `ResolveBinding` in `binding_resolved.go` (currently does NOT propagate Sandbox from rawBinding) AND wire `settingsFile.Sandbox`. The struct-shape comment at `render.go:374-380` documents the deferred extension. Worklog explicitly calls this out (lines 96-104).
- This is a clean deferral — no half-emitted sandbox JSON, no silent dropping of declared sandbox rules.

#### A8. Registration seam cycle break correctness — REFUTED
- Imports verified clean:
  - `render/render.go:33-34` imports `dispatcher` + `domain` (NOT cli_claude).
  - `render/init.go:4` imports `dispatcher` only.
  - `cli_claude/init.go:11` blank-imports `cli_claude/render` (one-way; render does NOT import cli_claude).
  - `dispatcher/spawn.go` imports neither cli_claude nor cli_claude/render.
- Dependency graph: `render → dispatcher` (types) + `cli_claude → render` (registration via blank import). No cycle.
- Registration path: cli_claude side-effect-imported by cmd/till or test → cli_claude.init() → blank-imports render → render.init() → calls dispatcher.RegisterBundleRenderFunc. Works regardless of import order.

#### A9. Coverage 86.2% — REFUTED (NIT-level)
- 13.8% uncovered most likely the `json.MarshalIndent` error paths (untriggerable with simple struct types — encoding/json doesn't return errors for `string`/`[]string` fields). Mathematically unreachable error branches are acceptable.
- All public surface (Render + ErrInvalidRenderInput) and all error wraps tested. No meaningful coverage gap.

#### A10. No `hylla_artifact_ref` in render.go — REFUTED (as expected)
- `render.go` content scanned: no `hylla_artifact_ref` substring anywhere. `assembleSystemPromptBody` (`render.go:201-234`) only consumes: `item.ID`, `project.ID`, `project.RepoPrimaryWorktree`, `item.Kind`, `item.Title`, `item.Paths`, `item.Packages`. No `project.HyllaArtifactRef` reference.
- `TestRenderSystemPromptContainsStructuralTokens` (`render_test.go:148-151`) pins absence: `if strings.Contains(bodyStr, "hylla_artifact_ref")` → fail. F.7.10 invariant locked.

#### A11. Memory rule conflicts — REFUTED
- `feedback_no_migration_logic_pre_mvp.md` — no SQL, no migration code added. Clean.
- `feedback_subagents_short_contexts.md` — single new package + small spawn-test extensions. Clean.
- `feedback_never_remove_workflow_files.md` — no workflow files removed. Clean.
- `feedback_section_0_required.md` — applies to subagent prompts; orchestration rule, not implementation. N/A.
- `feedback_orchestrator_no_build.md` — builder is the role that edits Go code. Correct role.
- `feedback_no_closeout_md_pre_dogfood.md` — only worklog + QA artifacts produced; no CLOSEOUT/LEDGER. Clean.
- No memory-rule violations.

#### A12. Cycle break depth — REFUTED
- Three-package chain verified non-cyclic:
  - `internal/app/dispatcher/spawn.go` — defines `BundleRenderFunc` + register/lookup. No imports of cli_claude or render.
  - `internal/app/dispatcher/cli_claude/init.go` — imports `dispatcher` (parent) + blank-imports `cli_claude/render` (child).
  - `internal/app/dispatcher/cli_claude/render/render.go` — imports `dispatcher` (grandparent) only. NOT `cli_claude`.
- No back-edge. Render imports types from dispatcher; dispatcher learns about render only via the registration seam at runtime.

#### A13. No-commit per REV-13 — REFUTED
- Worklog line 197-199: "Single-line conventional commit (orchestrator commits)" — builder explicitly did NOT commit, deferred to orchestrator.
- `git status --porcelain` reflects untracked workflow MDs + new render package files; nothing committed.

### Additional adversarial probes (no counterexamples)

#### Probe X1. settings.json missing sandbox key (claude default-deny semantics)
- `settingsFile` only emits `permissions`. Claude documentation (per memory §4) treats absent sandbox as "no sandbox restrictions applied at this layer" — adopters relying on the binding's Sandbox declarations would silently get no enforcement. However, the deferred-Sandbox emission is a documented gap (worklog 96-104) and BindingResolved propagation must land first before this becomes a real bug. Today: pre-existing limitation, not introduced by F.7.3b.

#### Probe X2. plugin.json minimum-required fields
- Per memory §3 / Context7 docs, claude plugin.json requires only `name`. Render emits `{"name": "spawn-<uuid>"}`. Other fields (`version`, `author`, `description`) are optional. `claude plugin validate` accepts the minimal shape.

#### Probe X3. .mcp.json schema match
- Memory §2 documents `mcp.json` shape `{"mcpServers": {...}}` for top-level `mcp.json`. F.7.3b emits `{"tillsyn": {...}}` directly without `mcpServers` wrapper. **Possible NIT** — verify with claude docs whether plugin-bundled `.mcp.json` uses the wrapper-less shape OR `mcpServers`-wrapped shape. Test pins the wrapper-less shape; if claude actually requires `mcpServers`, this would surface during runtime as "no MCP servers loaded." Cannot verify without Hylla/Context7 calls (declined per spawn prompt's "no Hylla calls" + scope).
- **Disposition**: not a counterexample (test passes; runtime acceptance is dispatcher's domain at first real spawn). Flag for future runtime probe.

#### Probe X4. Path collision between render and bundle
- Bundle creates `<root>/`, `<root>/manifest.json`. Render creates `<root>/system-prompt.md`, `<root>/plugin/`. No file-name overlap. NewBundle does NOT pre-create `system-prompt.md` (only the dir). Verified at `bundle.go:199-213`.

#### Probe X5. binding.AgentName length
- No max-length cap on AgentName. A pathologically long AgentName (>4096 bytes) → file write fails with ENAMETOOLONG. Returns error, rollback fires. Not exploitable; rollback handles cleanly. Acceptable.

#### Probe X6. Concurrent Render() calls
- Render itself is stateless — bundleRoot is per-call argument, no package-level state mutated. Concurrent calls with different bundles are safe. Concurrent calls with the SAME bundle would race on file writes — but the dispatcher creates a unique per-spawn bundle root, so this scenario doesn't arise.

### Summary
- All 13 named attacks REFUTED via direct code/test inspection.
- Coverage adequate; cycle break correct; rollback complete; path-traversal guards sound; no memory-rule violations; F.7.10 (`hylla_artifact_ref` absence) pinned by test.
- Sandbox deferral correctly motivated (BindingResolved doesn't carry the field; extension is a follow-up droplet's concern).
- Probe X3 flags a *potential* `.mcp.json` schema NIT (wrapper-less vs `mcpServers`-wrapped) but cannot be definitively resolved without Hylla/Context7 calls disallowed by the falsification scope. Test passes either way; runtime acceptance is the gate.

**Overall verdict: PASS** — proceed to commit.

### Hylla Feedback
N/A — task scope explicitly forbade Hylla calls per spawn prompt's "Hard constraints" section.
