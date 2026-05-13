# BUILDER WORKLOG — DROP 4c.6.1 W2 TILL INIT

---

## Round W2.D4 — TUI MCP CONFIRM STEP

**Date:** 2026-05-13
**Droplet:** W2.D4
**Paths:** `cmd/till/init_cmd.go`, `cmd/till/init_cmd_test.go`

### Changes made

**`cmd/till/init_cmd.go`**
- Added `initTUIStepMCP` constant in `initTUIStep` enum between `initTUIStepGroup` and `initTUIStepDone`. Updated doc comment on the type to reflect three-step flow.
- Added `mcpConfirm components.ConfirmModel` field to `initTUIModel` struct.
- `newInitTUIModel`: initialized `mcpConfirm = components.NewConfirm("Register MCP server in .mcp.json?", true)` (defaultYes=true per REVISION_BRIEF §2.6).
- `Update` — group step: removed the D3→D4 interim hardwire (`mcpFalse := false; m.finalPayload.MCP = &mcpFalse; m.step = initTUIStepDone; return m, tea.Quit`). Replaced with `m.step = initTUIStepMCP; return m, nil`.
- `Update` — new `case initTUIStepMCP:`: intercepts Esc directly (distinct from 'n' which is a valid NO answer — the confirm component merges both into `Cancelled()`, so Esc must be separated at the outer dispatch). For all other key messages, forwards to `m.mcpConfirm.Update(msg)`; when `Done()`, reads `Confirmed()` to set `mcpYes bool`, stores `m.finalPayload.MCP = &mcpYes`, advances to `initTUIStepDone` + `tea.Quit`.
- `View`: added `case initTUIStepMCP:` rendering `m.mcpConfirm.View() + "\n"`.

**`cmd/till/init_cmd_test.go`**
- Added `TestInitTUIModel_MCPStep` with three sub-tests (enter_yes / n_no / esc_cancel) using direct model Update calls (no teatest program). RED→GREEN verified per-function.
- Added `TestRunInit_JSONMode_MCPPaths` CONSUMER-TIE supplement with three sub-tests (mcp_true / mcp_false / no_mcp_key). GREEN verified per-function.
- Updated `TestRunInitTUI_AcceptsDefaultNameAndSelectsTillGo`: added WaitFor on MCP prompt + third Enter; flipped MCPRegistration assertion to `want true` (default YES after D4).
- Updated `TestRunInitTUI_SelectsFeRow`: added WaitFor on MCP prompt + Enter to accept YES after group confirm.
- Updated `TestInitTUIModel_GroupMultiSelect` (all three sub-tests): each now expects `initTUIStepMCP` after group Enter, adds a second Enter for MCP confirm, then asserts `initTUIStepDone`.

### Design decision

Esc-vs-n disambiguation: `confirm.go` maps both Esc and 'n' to `Cancelled()=true`. The outer MCP step case intercepts Esc before forwarding to the confirm component, so Esc cancels the walk and 'n' advances to `initTUIStepDone` with `MCP = false`. This preserves the spec requirement that n/N is a valid NO answer (not a walk cancel).

### Test results

- `mage test-func ./cmd/till TestInitTUIModel_MCPStep`: 4/4 GREEN
- `mage test-func ./cmd/till TestRunInit_JSONMode_MCPPaths`: 4/4 GREEN
- `mage test-pkg ./cmd/till`: 321/321 GREEN
- `mage ci`: ALL GREEN (coverage 76.3% on cmd/till; all packages >= 70%)

---

## Round W2.D7 — createProjectDBRecord UPGRADE TO CreateProjectWithMetadata

**Date:** 2026-05-13
**Droplet:** W2.D7
**Paths:** `cmd/till/init_cmd.go`, `cmd/till/init_cmd_test.go`

### Changes made

**`cmd/till/init_cmd.go`**
- Added `domain` import.
- Added `detectBareRoot(ctx context.Context, cwd string) string` helper: `exec.LookPath("git")` first-guard; then `exec.CommandContext(ctx, "git", "rev-parse", "--git-common-dir")`; trims + resolves absolute via `filepath.Abs(filepath.Join(cwd, trimmed))`; returns `""` on any error (non-fatal).
- Added `mapGroupsToLanguage(groups []string) string` helper: maps `groups[0]` through the closed language enum (go→"go", fe→"fe", anything else including "gen"→""). Selection-order wins; empty slice returns "" defensively.
- Changed `createProjectDBRecord` signature: `projectName string` → `payload initJSONPayload`. Updated call site in `runInitPipeline` from `createProjectDBRecord(context.Background(), opts, payload.Name)` to `createProjectDBRecord(context.Background(), opts, payload)`.
- `createProjectDBRecord` now calls `svc.CreateProjectWithMetadata(ctx, app.CreateProjectInput{...})` with `Name`, `RepoPrimaryWorktree`, `RepoBareRoot`, `Language`, `Metadata.Groups` all populated.
- Added `templateGroupMarkerPrefix` constant and `templateGroupMarkerPresent` helper for partial-state group detection.
- Updated `writeTemplateTOML`: removed `[<group>]` TOML section header prefix (which was nesting `schema_version` inside a sub-table, breaking `templates.Load`). Now writes `# till-init-groups: <group>` marker comment followed by the embedded template content for single-group projects. For multi-group projects, skips writing entirely (returns `0, 0, nil`) to avoid duplicate TOML key errors from naively concatenated templates.
- Updated `writeTemplateTOML` partial-state check: now looks for the marker comment (`templateGroupMarkerPresent`) OR the legacy `[<group>]`/`[<group>.]` patterns.

**`cmd/till/init_cmd_test.go`**
- Added `os/exec`, `domain`, `templates` imports.
- Added `TestCreateProjectDBRecord_GitRepoCase`: CONSUMER-TIE test (a) — `git init` in temp dir, runs `till init`, verifies `RepoPrimaryWorktree` non-empty + absolute, `RepoBareRoot` non-empty, `Language="go"`, `Metadata.Groups=["go"]`. Skips if git binary absent.
- Added `TestCreateProjectDBRecord_NonGitDirCase`: CONSUMER-TIE test (b) — temp dir without git init, verifies `RepoBareRoot=""`, `RepoPrimaryWorktree` non-empty, `Language="fe"`, `Metadata.Groups=["fe"]`.
- Added `TestCreateProjectDBRecord_IdempotentRerun`: CONSUMER-TIE test (c) — two runs with same project name, second run returns nil error with "already exists" in output.
- Added `projectNamesFromSlice` helper for diagnostic messages.
- Updated `TestWriteTemplateTOML_HOMETierPresent`: HOME-tier template now uses the embedded `till-go.toml` content with a `# home-tier-sentinel` comment (valid `templates.Template`); checks for `# till-init-groups: go` marker and `# home-tier-sentinel` (not `[go]` section header).
- Updated `TestWriteTemplateTOML_HOMETierAbsent`: checks for `# till-init-groups: go` marker (not `[go]` section header).
- Updated `TestWriteTemplateTOML_PartialStateWarning`: seeded file uses embedded `till-gen.toml` content with `# till-init-groups: gen` marker (valid `templates.Template`).

### Design decisions

1. **`writeTemplateTOML` section-header removal (D7 fix)**: The `[<group>]` TOML section header was nesting the embedded template's `schema_version = "v1"` inside a sub-table, making the top-level `schema_version` empty. `templates.Load` rejected with "unsupported schema version". This collision only became visible in D7 when `RepoPrimaryWorktree = cwd` was set — before D7, `RepoPrimaryWorktree` was empty so `bakeProjectKindCatalog` never tried to read the on-disk file.

2. **Multi-group `template.toml` skip**: For multi-group projects, writing multiple groups' templates requires a semantic merge (not naive concatenation). Both `till-go.toml` and `till-fe.toml` declare `[kinds.plan]`, `[kinds.build]`, etc. Naive concatenation → `templates.Load` error "table plan already exists". Proper merge requires the unexported `mergeTemplates` function from `internal/app` (out of D7's declared paths). Solution: skip writing `template.toml` for multi-group. `bakeProjectKindCatalog` resolves each group via HOME tier and embedded defaults (correct behavior). PLATFORM-TEMPLATES-R1 refinement tracks future proper multi-group aggregation.

3. **`Language` mapping policy**: `groups[0]` determines Language per selection-order policy (NIT5 absorption). "gen"-first → Language="" (no language bias). Documented in `mapGroupsToLanguage` doc comment.

4. **Test DB isolation**: The three CONSUMER-TIE tests (`TestCreateProjectDBRecord_*`) use a real SQLite database rooted in `t.TempDir()` (not a mock or in-memory stub). Isolation is achieved via `t.Setenv("HOME", tmp)` redirecting `~/.tillsyn` to `tmp/.tillsyn-init/` + `--app tillsyn-init` flag. No contact with the dev's real `~/.tillsyn/tillsyn.db`. This matches the existing `init_cmd_test.go` pattern throughout.

### Test results

- `mage test-func ./cmd/till TestCreateProjectDBRecord_NonGitDirCase`: 1/1 GREEN (RED→GREEN confirmed)
- `mage test-func ./cmd/till TestCreateProjectDBRecord_IdempotentRerun`: 1/1 GREEN
- `mage test-func ./cmd/till TestCreateProjectDBRecord_GitRepoCase`: 1/1 GREEN
- `mage test-pkg ./cmd/till`: 336/336 GREEN
- `mage ci`: ALL GREEN (coverage 76.8% on cmd/till; all packages >= 70%)

---

## Round W2.D7-ABS — W2.D7 QA NIT ABSORPTIONS

**Date:** 2026-05-13
**Paths:** `cmd/till/init_cmd.go`, `cmd/till/init_cmd_test.go`, `workflow/drop_4c_6_1/DROP_4c.6.1.W2_TILL_INIT/BUILDER_WORKLOG.md`

### Changes made

**A — `detectBareRoot` defense-in-depth (Proof NIT.1 + Falsification NIT-2)**
- Set `cmd.Dir = cwd` on the `exec.CommandContext` call so the git subprocess always runs in the intended directory regardless of process CWD at call time.
- Added `filepath.IsAbs(trimmed)` branch: absolute git output (linked-worktree case) is used directly without `filepath.Join(cwd, trimmed)` concatenation. Relative output (`.git`) still goes through `filepath.Join(cwd, trimmed)` then `filepath.Abs`. Fixes the linked-worktree `filepath.Join("/cwd", "/abs/path")` garbage-path bug.
- Updated doc comment to accurately describe the path resolution logic.

**B — Multi-group Laslig status string (Proof NIT.3 / Falsification NIT-1)**
- Changed `writeTemplateTOML` signature from `(int, int, error)` to `(int, int, string, error)`. The returned string is the Laslig status for the `template.toml` row.
- Three distinct status values: `"added"` (file written, single group), `"skipped (already exists)"` (file existed), `"skipped (multi-group — uses per-group HOME/embedded resolution)"` (multi-group, no file written).
- Updated `runInitPipeline` call site to receive 4-value return and use the status string directly for the Laslig row. Removed the `if templateAdded > 0` conditional that was producing the misleading `"skipped (already exists)"` for the multi-group fresh-install case.

**C — `mapGroupsToLanguage` test table (Proof NIT.2)**
- Added `TestMapGroupsToLanguage` with 5 table-driven cases: `gen→""`, `gen+go→""` (selection-order), `go→"go"`, `fe→"fe"`, `[]→""` (empty-slice no-panic).

**D — Multi-group CONSUMER-TIE test (Proof NIT.4)**
- Added `TestCreateProjectDBRecord_MultiGroup`: `run()` end-to-end with `groups:["go","fe"]`. Asserts: exit zero, `Metadata.Groups=["go","fe"]`, `Language="go"` (first-group-wins), `template.toml` absent, per-group agent subdirs `agents/go/` and `agents/fe/` exist.

**E — Worklog phrasing clarification (Falsification NIT-3)**
- Added design-decision 4 to the D7 entry clarifying that tests use real SQLite rooted in `t.TempDir()` (not a mock), with isolation via `t.Setenv("HOME", tmp)` + `--app tillsyn-init`.

### Test results

- `mage test-func ./cmd/till TestWriteTemplateTOML_HOMETierPresent`: 1/1 GREEN (signature change does not break existing test)
- `mage test-func ./cmd/till TestMapGroupsToLanguage`: 6/6 GREEN (5 table cases)
- `mage test-func ./cmd/till TestCreateProjectDBRecord_MultiGroup`: 1/1 GREEN
