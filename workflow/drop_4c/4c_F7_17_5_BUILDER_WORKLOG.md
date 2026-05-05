# 4c F.7.17.5 — Dispatcher wiring (Builder Worklog)

## 1. Goal

Replace the Drop-4a 4a.19 stub `BuildSpawnCommand` with multi-adapter wiring keyed by `CLIKind`:

- `BindingResolved` resolved via `ResolveBinding(rawBinding)`.
- Adapter looked up from a registry keyed by `CLIKind` (default-to-claude per F.7.17 L15; unknown kinds → `ErrUnsupportedCLIKind`).
- Provisional per-spawn `BundlePaths` materialized via `os.MkdirTemp` (TODO marker — F.7-CORE F.7.1 will replace).
- `system-prompt.md` rendered with action-item structural fields (task_id, project_id, project_dir, kind, title, paths, packages, move-state directive) — NO `hylla_artifact_ref` per F.7.10.
- `adapter.BuildCommand(ctx, resolved, bundlePaths)` invoked; `cmd.Dir` set to the project worktree post-call.
- Existing 4-arg signature preserved (`BuildSpawnCommand(item, project, catalog, AuthBundle)`) so all four production callers (`dispatcher.go` L545, L748; `cmd/till/dispatcher_cli_test.go`) compile unchanged.

## 2. Changes

### 2.1 `internal/app/dispatcher/spawn.go` (rewritten)

- Added `ErrUnsupportedCLIKind` sentinel.
- Added concurrency-safe registry: `adaptersMu sync.RWMutex` + `adaptersMap map[CLIKind]CLIAdapter` + public `RegisterAdapter(kind, adapter)` + private `lookupAdapter(kind)`.
- Replaced inline argv assembly with: `ResolveBinding(rawBinding)` → adapter lookup → bundle creation → prompt write → `adapter.BuildCommand(ctx, resolved, bundlePaths)` → `cmd.Dir = project.RepoPrimaryWorktree`.
- Bundle creation uses `os.MkdirTemp("", "tillsyn-spawn-")` with a clear `TODO(F.7.1)` marker noting that F.7-CORE F.7.1 owns full bundle lifecycle (manifest.json, deferred cleanup, project-mode root, materialized plugin tree).
- System prompt is written to `<bundleRoot>/system-prompt.md` with `0o600` perms.
- `assemblePrompt` extended to surface `paths` and `packages` when set on the action item; existing `task_id` / `project_id` / `project_dir` / `kind` / `title` / move-state directive shape preserved; `hylla_artifact_ref` deliberately absent.
- `SpawnDescriptor.MCPConfigPath` set to `<bundleRoot>/plugin/.mcp.json` so `till dispatcher run --dry-run` JSON output continues to point at the same path the claude adapter wires into argv.
- Pointer-typed BindingResolved fields (`Model`, `MaxBudgetUSD`, `MaxTurns`) deref-helpered into the descriptor's value fields.
- Uses `context.Background()` internally with `TODO(F.7-CORE)` to plumb the outer dispatcher ctx; preserves the 4-arg signature so 4a.19 callers don't need patching.

### 2.2 `internal/app/dispatcher/spawn_test.go` (rewritten)

- Converted from `package dispatcher` (internal) to `package dispatcher_test` (external) so the test file can blank-import `cli_claude` for side-effect adapter registration without forming a cycle.
- Side-effect import `_ "github.com/evanmschultz/tillsyn/internal/app/dispatcher/cli_claude"` registers the real claude adapter at test-binary init.
- Existing tests rewritten against the new argv shape (claude adapter long form: `--bare`, `--plugin-dir`, `--agent`, `--system-prompt-file`, ...). Tests don't pin the entire argv (the claude adapter owns that contract; pinned in `cli_claude/adapter_test.go`); they use `argFlagValue(argv, flag)` + `argvContains(argv, s)` helpers to assert the load-bearing markers.
- New scenarios:
  - `TestBuildSpawnCommandUsesClaudeAdapterByDefault` — empty `rawBinding.CLIKind` defaults to claude.
  - `TestBuildSpawnCommandHonorsExplicitClaudeCLIKind` — explicit `"claude"` produces same result.
  - `TestBuildSpawnCommandRejectsUnknownCLIKind` — `"bogus"` trips `ErrUnsupportedCLIKind`.
  - `TestBuildSpawnCommandWritesSystemPromptFile` — reads the file at `--system-prompt-file`, asserts task_id / project_id / paths / packages / move-state directive present, asserts `hylla_artifact_ref` absent.
  - `TestBuildSpawnCommandPropagatesBundlePaths` — descriptor's `MCPConfigPath` ends with `plugin/.mcp.json` and lives under the same bundle root as `--plugin-dir` and `--system-prompt-file`.
  - `TestRegisterAdapterRoutesCustomCLIKind` — registers a `fakeAdapter` under a custom `CLIKind` and confirms `BuildCommand` is routed there (Drop-4d codex seam).
- `removeBundle` cleanup hook reads `--plugin-dir` from argv to derive the bundle root and schedules `os.RemoveAll` via `t.Cleanup` so each test's bundle is reaped automatically.
- All previous validation guards retained: empty action-item ID / Kind / RepoPrimaryWorktree, corrupted AgentBinding, unbound kind, empty catalog, fractional budget formatting, SpawnDescriptor zero-value invariants.

### 2.3 `internal/app/dispatcher/cli_claude/init.go` (NEW — scope expansion, see §4)

- `func init()` calls `dispatcher.RegisterAdapter(dispatcher.CLIKindClaude, New())`.
- Inverts the dependency direction: `cli_claude` already imports `dispatcher` for type definitions; the init() reuses that one-way import to populate the registry. `dispatcher/spawn.go` does NOT import `cli_claude`, so no cycle.

### 2.4 `cmd/till/main.go` (1-line change — scope expansion, see §4)

- Added blank import `_ "github.com/evanmschultz/tillsyn/internal/app/dispatcher/cli_claude"` so the production binary triggers `cli_claude.init()` at startup, populating the registry before the dispatcher dispatches anything.

## 3. Acceptance criteria

- [x] `adaptersMap` registers `CLIKindClaude` → `cli_claude.New()` via `cli_claude.init()`.
- [x] `ErrUnsupportedCLIKind` sentinel exported.
- [x] `BuildSpawnCommand` looks up adapter, errors loudly on unknown CLIKind.
- [x] `BindingResolved` populated via `ResolveBinding`.
- [x] Provisional `BundlePaths` with `TODO(F.7.1)` marker.
- [x] `system-prompt.md` written with action-item fields, NO `hylla_artifact_ref`.
- [x] All test scenarios pass (2496 tests across 23 packages; 192 dispatcher tests; 229 cmd/till tests).
- [x] `mage check` + `mage ci` green; coverage 74.9% on `internal/app/dispatcher` (≥70% threshold), 95.7% on `internal/app/dispatcher/cli_claude`.
- [x] Worklog written.
- [x] **NO commit by builder** (per F.7-CORE REV-13).

## 4. Scope deviations

The droplet spec listed three files for edit/create: `spawn.go`, `spawn_test.go`, the worklog. Two scope expansions were unavoidable to satisfy the spec's intent without forming an import cycle:

### 4.1 Why the spec's literal Go is unbuildable

The spec's pseudo-code:

```go
var adapters = map[CLIKind]CLIAdapter{
    CLIKindClaude: cli_claude.New(),
}
```

would require `dispatcher` (where `spawn.go` lives) to import `internal/app/dispatcher/cli_claude`. But `cli_claude` already imports `dispatcher` (for `BindingResolved`, `BundlePaths`, `CLIAdapter`, `StreamEvent`, `TerminalReport`, etc. — see `cli_claude/adapter.go:32`, `argv.go:8`, `env.go:9`, `stream.go:8`). That is a hard import cycle Go refuses to compile.

### 4.2 Resolution

Inverted the dependency: `dispatcher` exposes a public `RegisterAdapter(kind, adapter)` setter; `cli_claude.init()` calls it. The registration trigger requires SOMETHING to import `cli_claude`:

- **`internal/app/dispatcher/cli_claude/init.go`** (NEW): one-file package addition holding the `init()` that registers the claude adapter. No production logic — just the registration.
- **`cmd/till/main.go`** (1-line change): blank import `_ "..../cli_claude"` so the production binary's startup pulls cli_claude.init().

Test wiring uses the same blank import inside `spawn_test.go`'s external package, so no production-only registration path can drift undetected.

### 4.3 Why this is the minimum-viable expansion

Alternatives considered and rejected:

- **Move the adapters map into a wiring sub-package** (`internal/app/dispatcher/cli_register/`): adds a new package directory + new file; cmd/till still needs the side-effect import. More files than the chosen route.
- **Pass adapter map as a parameter to `BuildSpawnCommand`**: changes the function signature, breaks the 4 production callers. Spec explicitly forbids signature change.
- **Keep map empty and expect tests to register inline**: doesn't satisfy the acceptance criterion "`adapters` map registers `CLIKindClaude` → `cli_claude.New()`" — production binaries would have an empty registry.
- **Have `dispatcher` import `cli_claude` and break the cycle by moving types out of `dispatcher`**: gigantic refactor of types-package layout, far outside scope.

The chosen path adds 2 new files (`cli_claude/init.go`, the worklog) and modifies 3 (`spawn.go`, `spawn_test.go`, `cmd/till/main.go`). The orchestrator should review whether this scope-expansion was correct or whether the spec's literal pseudo-code was intended to be a different shape entirely.

## 5. Verification

```
mage check  # SUCCESS — 2496 tests pass, coverage 74.9% dispatcher / 95.7% cli_claude / 75.5% cmd/till
mage ci     # SUCCESS — same numbers + lint + format + build
```

## 6. Conventional commit message (≤72 chars, single line)

```
feat(dispatcher): wire CLIAdapter registry through BuildSpawnCommand
```
