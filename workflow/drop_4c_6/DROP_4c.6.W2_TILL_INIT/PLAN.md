# DROP_4c.6.W2 — TILL_INIT

**State:** planning
**Blocked by:** 4c.6.W1.D1 (W2 copies the agent .md files shipped by W1; without W1's embedded scaffolding there's nothing to copy)
**Paths (expected):** `internal/vendor/fsatomic/**`, `internal/vendor/configmerge/**`, `internal/vendor/VENDOR_SOURCE.md`, `cmd/till/init_cmd.go`, `cmd/till/init_cmd_test.go`, `cmd/till/install_cmd.go`, `cmd/till/install_cmd_test.go`, `cmd/till/main.go`, `cmd/till/main_test.go`, `cmd/till/help.go`
**Packages (expected):** `github.com/evanmschultz/tillsyn/internal/vendor/fsatomic`, `github.com/evanmschultz/tillsyn/internal/vendor/configmerge`, `github.com/evanmschultz/tillsyn/cmd/till`
**PLAN.md ref:** `workflow/drop_4c_6/PLAN.md` → `4c.6.W2` row (lines 117-133)
**Workflow:** `workflow/example/drops/WORKFLOW.md`
**Cascade concept:** `AGENT_CASCADE_DESIGN.md`
**Started:** 2026-05-09
**Closed:** —

## Scope

Land `till init` per `SKETCH.md` §9 + §26.W2 — TUI walk (project name + group picker), copy embedded `internal/templates/builtin/agents/<group>/*.md` → `<project>/.tillsyn/agents/*.md` FLAT, copy `agents.example.toml` → `<project>/agents.toml`, ensure `agents.local.toml` in `.gitignore`, optional `.mcp.json` registration, project-DB record creation, Laslig success message, JSON mode (`--json '{...}'`) with identical behavior, re-run safety (never overwrites). Plus: vendor `fsatomic` (52 LOC, zero deps) + `configmerge` (~12kB + tests) from `ta` to `internal/vendor/` with `VENDOR_SOURCE.md` provenance per `SKETCH.md` §9.6 — every vendored Go file MUST carry a 2-3 line block-comment header `// DO NOT EDIT — re-vendor from upstream` plus pointer to `internal/vendor/VENDOR_SOURCE.md` (ROUND-2 OQ#2). Plus: ADD `till install` CLI command (NEW — see OQ#3 verification below) that takes over the dev-config-creation behavior currently in `cmd/till/main.go:2039 runInitDevConfig` BEFORE D8 finalizes the removal of `init-dev-config`. JSON-mode + TUI behaviors must be IDENTICAL apart from input source.

## OQ#3 Verification — `till install` Coverage Status

**Verification performed (2026-05-09):**

- `cmd/till/main.go:1885-1903` registers a `init-dev-config` Cobra command whose `RunE` calls `runInitDevConfig` (`cmd/till/main.go:2039-2094`).
- `runInitDevConfig` semantically does TWO things: (a) creates `<dev-paths>/till.toml` from `config.DefaultTemplate()` if it doesn't exist (lines 2059-2072), and (b) rewrites the `[logging]` section to `level = "debug"` (lines 2074-2083). Closes with a `writeCLIKV` Laslig table.
- `git grep -n "runInstall\|installCmd" cmd/till/ internal/` returns NO matches.
- `git grep -n init-dev-config cmd/till/` shows references in `main.go` (the registration), `help.go:377-390` (rich-help spec), and `main_test.go` (lines 476, 732-734, 2906, 2928-2938, 2955, 2988-2993, 3105 — five test functions plus the registered-commands list and the rich-help table-test).
- `magefile.go:140-145` defines a `mage install` build target (`mage install` writes the binary to `~/.local/bin/till`). This is the ONLY thing called "install" in the codebase. **It is a build-tool target, not a `till` CLI subcommand.**

**Verdict: `till install` does NOT exist as a CLI command today.** The L1 directive's premise that "`till install` covers (or is extended to cover) the dev-config-creation behavior" is FALSE. SKETCH §9.1 names `till install` as the destination ("install-time setup (DB creation, default config) folds into `till install`"), so the SKETCH explicitly intends a NEW `till install` CLI command — not a fold-into existing logic.

**Disposition: EXPAND W2 with a NEW droplet `D7.5` that creates `cmd/till/install_cmd.go` (the `till install` Cobra command) and ports the dev-config-creation behavior from `runInitDevConfig` into it.** D7.5 is `blocked_by` D3 (registers commands in `main.go`'s `rootCmd.AddCommand` call) and is a **hard precondition for D8** (D8's removal of `init-dev-config` is a behavior regression unless D7.5 lands first). D8's `Blocked by:` accordingly includes D7.5.

**Important:** the `init` and `install` commands are SEPARATE — they do different things:
- `till init` (D3-D7): seeds a project (cwd-local). Copies agent `.md` files, writes `agents.toml`, updates `.gitignore`, optionally writes `.mcp.json`, creates the project DB record. Per-project setup.
- `till install` (D7.5): bootstraps the local Tillsyn dev environment (home-local). Creates `<dev-paths>/till.toml` with `[logging] level = "debug"`. Per-machine setup.

The two are wired via `main.go`'s `rootCmd.AddCommand` line; both share the `cmd/till/main.go` file lock so D8 ordering chain remains valid.

## Planner

### Droplet 4c.6.W2.D1 — Vendor `fsatomic` package from `ta`

- **State:** todo
- **Paths:**
  - `internal/vendor/fsatomic/atomic.go` (NEW)
  - `internal/vendor/fsatomic/atomic_test.go` (NEW)
  - `internal/vendor/VENDOR_SOURCE.md` (NEW — created by D1; D2 appends to the same file)
- **Packages:** `github.com/evanmschultz/tillsyn/internal/vendor/fsatomic` (NEW package)
- **Acceptance:**
  - Package `fsatomic` exists at `internal/vendor/fsatomic/`. Source matches the `ta` upstream commit cited in `VENDOR_SOURCE.md` byte-for-byte (run `diff` to confirm during build).
  - **Every Go file under `internal/vendor/fsatomic/` carries a 2-3 line block-comment header at the very top of the file (above `package fsatomic`):**
    ```
    // DO NOT EDIT — re-vendor from upstream.
    // Source + provenance: see ../VENDOR_SOURCE.md.
    ```
  - `internal/vendor/VENDOR_SOURCE.md` lists `fsatomic` with: upstream repo URL (the `ta` repo path), upstream commit hash (full 40-char SHA the dev pins), date vendored (2026-05-09), LOC count (~52), zero-deps confirmation, future-migration plan ("when `ta` reaches MVP, extract to `hylla-shared` per SKETCH.md §9.6"). Markdown table or `### fsatomic` + bullets — pick whichever scales to multiple vendored packages cleanly.
  - `mage test-pkg ./internal/vendor/fsatomic` passes (the upstream test file vendored alongside).
  - `mage ci` green — no lint failures on the vendored block-comment header.
  - Builder confirms `ta` upstream license permits vendoring; LICENSE file (or equivalent attribution) included alongside the source if upstream's license requires it.
- **Blocked by:** —
- **Notes for builder:**
  - `fsatomic` per SKETCH §9.6: "52 LOC, zero deps." Spawn-prompt MUST include the exact upstream commit hash (dev provides). If the dev hasn't pinned a commit, escalate to orchestrator before vendoring (do NOT pick HEAD silently — provenance is load-bearing).
  - The vendor directory does NOT exist today (`internal/vendor/`: No such file or directory verified 2026-05-09). D1 creates it.
  - Builder runs `mage format` after vendoring; vendored upstream code may not match `gofumpt` and that's acceptable per "DO NOT EDIT" — but the BLOCK-COMMENT HEADER prefix must come BEFORE the original `package` declaration so `gofumpt` doesn't re-order. If `gofumpt` rewrites the file, file a refinement (do NOT keep the rewrite).
  - The package is a NEW Go package — no file-collision with `cmd/till` or anything else, so D1 runs fully parallel with D3-D8 until D5 needs it.

### Droplet 4c.6.W2.D2 — Vendor `configmerge` package from `ta`

- **State:** todo
- **Paths:**
  - `internal/vendor/configmerge/*.go` (NEW — multiple files; ~12kB total per SKETCH §9.6)
  - `internal/vendor/configmerge/*_test.go` (NEW — tests vendored alongside)
  - `internal/vendor/VENDOR_SOURCE.md` (APPEND — D1 created it; D2 adds the `configmerge` section)
- **Packages:** `github.com/evanmschultz/tillsyn/internal/vendor/configmerge` (NEW package)
- **Acceptance:**
  - Package `configmerge` exists at `internal/vendor/configmerge/`. Sources match the `ta` upstream commit cited in `VENDOR_SOURCE.md` byte-for-byte.
  - **Every Go file under `internal/vendor/configmerge/` carries the 2-3 line block-comment header** (same shape as D1).
  - `internal/vendor/VENDOR_SOURCE.md` updated with `configmerge` section: upstream commit hash, date, LOC, dependency note ("one dep already in Tillsyn — verify in builder spawn which one and confirm it's in `go.mod` already"), future-migration plan.
  - The "one dep already in Tillsyn" claim from SKETCH §9.6 is verified by the builder against `go.mod` BEFORE vendoring. If the dep is NOT already in `go.mod`, escalate to orchestrator (do NOT silently `go get` a new dependency — that's a separate decision and a separate droplet).
  - `mage test-pkg ./internal/vendor/configmerge` passes.
  - `mage ci` green.
- **Blocked by:** —
- **Notes for builder:**
  - File-level: D2 appends to `internal/vendor/VENDOR_SOURCE.md`, which D1 also writes. **D2 is `Blocked by: D1`** for the `VENDOR_SOURCE.md` shared-file lock. (Listed below in the explicit `Blocked by` field.)
  - **Correction:** D2 IS blocked by D1 on the `VENDOR_SOURCE.md` file. Updating the `Blocked by` line above to `D1`.
- **Blocked by (final):** D1 (shared file `internal/vendor/VENDOR_SOURCE.md`).

### Droplet 4c.6.W2.D3 — `cmd/till/init_cmd.go` skeleton + flag wiring + JSON mode parser + register in `main.go`

- **State:** todo
- **Paths:**
  - `cmd/till/init_cmd.go` (NEW)
  - `cmd/till/init_cmd_test.go` (NEW — tests for `--json` flag parsing + skeleton invocation)
  - `cmd/till/main.go` (modify: add `initCmd := newInitCommand(...)` build + add `initCmd` to `rootCmd.AddCommand(...)` at line 1904)
  - `cmd/till/help.go` (modify: add a new entry to the rich-help table at the analogous position to the existing `"till init-dev-config"` block — section is the `commandHelpSpecs` map ending at line 391)
- **Packages:** `github.com/evanmschultz/tillsyn/cmd/till`
- **Acceptance:**
  - `cmd/till/init_cmd.go` exports `newInitCommand(stdout io.Writer, rootOpts rootCommandOptions) *cobra.Command` (or whichever signature matches the existing pattern in `main.go`'s sibling builders — verify by reading `main.go` for the `initDevConfigCmd := &cobra.Command{...}` shape and matching). Returns a `*cobra.Command` with `Use: "init"`, short + long help, `Example` block, `cobra.NoArgs` (or `cobra.MaximumNArgs(0)` to match local convention).
  - The `RunE` function dispatches on whether `--json <payload>` is set: if set, calls `runInitJSON(stdout, opts, payload)`; otherwise calls `runInitTUI(stdout, opts)`. Both functions are STUBS in D3 — they return `errors.New("till init: TUI walk not yet wired (W2.D4)")` and `errors.New("till init: JSON parse OK; file copy not yet wired (W2.D5)")` respectively. The stubs let D3 land in isolation; D4-D7 fill them in.
  - `--json` flag is wired with a `String` flag default `""`. JSON payload struct is defined in `init_cmd.go`: `type initJSONPayload struct { Name string \`json:"name"\`; Group string \`json:"group"\`; MCP bool \`json:"mcp"\` }`. `runInitJSON` parses the payload via `encoding/json.Unmarshal` and validates `Group` is one of `{"till-gen", "till-go"}` (NOT `till-gdd` per SKETCH §9.3 — greyed-out until post-Hylla-rev). Invalid group → returns a wrapped error.
  - `cmd/till/main.go:1904` `rootCmd.AddCommand(...)` line includes `initCmd` (`initCmd` built earlier in the same function via `newInitCommand(...)`).
  - `cmd/till/help.go` rich-help table includes a `"till init"` entry analogous to the existing `"till init-dev-config"` entry at lines 377-390.
  - `cmd/till/init_cmd_test.go`: table-driven test for JSON-payload parsing (valid, invalid-group, malformed-JSON, missing-required-fields). At least one test confirms `--json` + bare TUI invocation route to the right `RunE` branch (using either a fake stdin or a flag-set assertion — pick whichever matches `cmd/till/main_test.go`'s style).
  - `mage test-pkg ./cmd/till` passes.
  - `mage ci` green.
- **Blocked by:** —
- **Notes for builder:**
  - **Sizing watch:** if D3's combined LOC (skeleton + JSON-payload struct + dispatch + tests + main.go register + help.go entry) exceeds ~120 LOC across production files, escalate to orchestrator. D3 may need to split into D3a (skeleton + register, no JSON) and D3b (JSON parser); for now keep one droplet.
  - File-locks: D3 writes `cmd/till/main.go` (registers init). D8 also writes `cmd/till/main.go` (removes init-dev-config). D3 lands first per L1 directive.

### Droplet 4c.6.W2.D4 — `runInitTUI` — bubbletea walk for project name + group picker

- **State:** todo
- **Paths:**
  - `cmd/till/init_cmd.go` (modify: replace `runInitTUI` stub with real implementation)
  - `cmd/till/init_cmd_test.go` (modify: add tea-test for the walk — using `teatest_v2` since that's the in-repo test substrate per `go.mod:11` replace directive)
- **Packages:** `github.com/evanmschultz/tillsyn/cmd/till`
- **Acceptance:**
  - `runInitTUI` collects two inputs interactively: project `name` (default = `filepath.Base(cwd)`; user can edit), `group` (selectable list: `till-gen` (default highlighted) + `till-go`; `till-gdd` shown but disabled per SKETCH §9.3).
  - After collection, `runInitTUI` invokes the same downstream pipeline that `runInitJSON` invokes (D5 wires the pipeline). In D4, `runInitTUI` returns the gathered `initJSONPayload`-equivalent struct and lets the caller dispatch to D5's pipeline. In D4, the caller still returns a stub error from the file-copy step — D5 fills that in.
  - Tea-test in `init_cmd_test.go` simulates user pressing `enter` on the default name + selecting `till-go`, asserts the resulting struct.
  - `mage test-pkg ./cmd/till` passes.
- **Blocked by:** D3 (D4 modifies `cmd/till/init_cmd.go` which D3 created — same-file lock).
- **Notes for builder:**
  - Use existing bubbletea infrastructure per SKETCH §26.W2 — `internal/tui/` packages have form/picker patterns; cite the file you're cloning the pattern from in `BUILDER_WORKLOG.md`.
  - Greyed-out `till-gdd` option must be UNSELECTABLE — pressing enter on it should be a no-op or play a UI bell, not advance to next step.

### Droplet 4c.6.W2.D5 — File-copy pipeline + `.gitignore` ensure (uses fsatomic + configmerge)

- **State:** todo
- **Paths:**
  - `cmd/till/init_cmd.go` (modify: implement the file-copy pipeline that both `runInitTUI` and `runInitJSON` call)
  - `cmd/till/init_cmd_test.go` (modify: add file-copy + re-run-safety + `.gitignore`-idempotent tests against `t.TempDir()`)
- **Packages:** `github.com/evanmschultz/tillsyn/cmd/till`
- **Acceptance:**
  - `copyAgentFiles(destDir, group string)` reads embedded `internal/templates/builtin/agents/<group>/*.md` (FS exposed by W1.D1's embed.go) and writes to `<destDir>/.tillsyn/agents/*.md` FLAT (no group prefix). Uses `fsatomic` for atomic writes (write-temp + rename pattern).
  - `copyAgentsTOML(destDir)` copies embedded `internal/templates/builtin/agents.example.toml` → `<destDir>/agents.toml` atomically.
  - `ensureGitignore(destDir)` adds `agents.local.toml` to `<destDir>/.gitignore` (creates the file if absent; idempotent — re-run does NOT duplicate the line). Uses `configmerge` if the merge logic fits; otherwise hand-written line-presence check is acceptable (justify in `BUILDER_WORKLOG.md`).
  - **Re-run safety:** every write goes through a `_, err := os.Stat(target); errors.Is(err, fs.ErrNotExist)` pre-check. Existing files are skipped, NOT overwritten. The function returns counts: `added int, skippedExisting int`.
  - JSON-mode and TUI-mode both call the same `copyAgentFiles` + `copyAgentsTOML` + `ensureGitignore` sequence — behavior IDENTICAL apart from input source.
  - Tests in `init_cmd_test.go`:
    - `TestInit_FreshDir_CopiesAllFiles`: empty `t.TempDir()`, `till init --json '{"name":"foo","group":"till-go","mcp":false}'`, asserts 7 agent .md files appear under `.tillsyn/agents/` + `agents.toml` exists + `.gitignore` contains `agents.local.toml`.
    - `TestInit_RerunSafety_NoOverwrite`: run init twice; second run reports `added=0, skipped=N` and does NOT mutate any existing file (compare mtimes or hashes).
    - `TestInit_GitignoreIdempotent`: `.gitignore` already contains `agents.local.toml`; re-run does NOT add a duplicate line.
    - `TestInit_PreExistingGitignore_AppendsCleanly`: `.gitignore` exists with unrelated entries; re-run appends `agents.local.toml` once with proper newline handling.
  - `mage test-pkg ./cmd/till` passes.
- **Blocked by:** D1 (uses `fsatomic`), D2 (uses `configmerge`), D4 (D5 modifies `cmd/till/init_cmd.go` which D4 last touched — same-file lock).
- **Notes for builder:**
  - Embedded FS access: read `internal/templates/builtin/embed.go` (extended by W1.D1) to confirm the embed-FS API for accessing per-group `agents.<group>` subdirs. If W1.D1 hasn't shipped yet at builder time, the shared `Blocked by: 4c.6.W1.D1` at the W2 container level catches this.
  - The 7 standard agent names (planning, builder, qa-proof, qa-falsification, research, closeout, commit-message) per SKETCH §11.1 + the `_BLOCKERS.toml` reference in W1.D1's plan; confirm count matches when D5 runs.

### Droplet 4c.6.W2.D6 — `.mcp.json` optional registration

- **State:** todo
- **Paths:**
  - `cmd/till/init_cmd.go` (modify: add `registerMCPJSON(destDir, includeMCP bool)` function)
  - `cmd/till/init_cmd_test.go` (modify: add tests for the `--mcp` / `mcp:true` JSON branch)
- **Packages:** `github.com/evanmschultz/tillsyn/cmd/till`
- **Acceptance:**
  - **Schema verification (builder responsibility BEFORE writing code):** the builder MUST verify Claude Code's `.mcp.json` shape via Context7 (`mcp__plugin_context7_context7__resolve-library-id` + `mcp__plugin_context7_context7__query-docs` for "claude-code mcp config" / "Claude Code .mcp.json schema"). If Context7 doesn't surface a usable shape, the builder reads an authoritative live `.mcp.json` from a known-good install (the dev's own machine — escalate to orchestrator for the path) and matches that. Hardcoded guesses are NOT acceptable — schema mismatches break MCP registration silently.
  - `registerMCPJSON` reads existing `<destDir>/.mcp.json` (if present), parses it, adds a `tillsyn` server entry pointing at the local `till` binary path (`exec.LookPath("till")` or `~/.local/bin/till` per `magefile.go:144`), writes back atomically via `fsatomic`. If `.mcp.json` is absent, creates a minimal one with just the `tillsyn` server.
  - Re-run safety: if the `tillsyn` entry already exists, skip (do NOT duplicate or overwrite). Report `added=0, skipped=1` style.
  - TUI mode: confirms with the user before mutating (yes/no prompt). JSON mode: respects the `mcp` boolean.
  - Tests:
    - `TestInit_MCPJSON_FreshFile`: creates `.mcp.json` with `tillsyn` entry.
    - `TestInit_MCPJSON_AppendsToExisting`: existing `.mcp.json` with another server entry; re-run adds `tillsyn` without removing the other.
    - `TestInit_MCPJSON_Idempotent`: existing `tillsyn` entry; re-run is a no-op.
    - `TestInit_MCPJSON_OptOut`: `mcp:false`; no `.mcp.json` written.
  - `mage test-pkg ./cmd/till` passes.
- **Blocked by:** D5 (D6 modifies `cmd/till/init_cmd.go` which D5 last touched — same-file lock).
- **Notes for builder:**
  - Document Context7 query result (or the authoritative-source path) in `BUILDER_WORKLOG.md` — schema-divergence risk is real per SKETCH §26.W2 RiskNotes.

### Droplet 4c.6.W2.D7 — Project-DB record creation + Laslig success message

- **State:** todo
- **Paths:**
  - `cmd/till/init_cmd.go` (modify: add `createProjectDBRecord(...)` + Laslig success print)
  - `cmd/till/init_cmd_test.go` (modify: add DB-record-existence assertion + success-message-format snapshot)
- **Packages:** `github.com/evanmschultz/tillsyn/cmd/till`
- **Acceptance:**
  - After file copy + `.mcp.json` registration succeed, `runInitJSON` and `runInitTUI` both call into `internal/app/service.go`'s `Service.CreateProject` (or `CreateProjectWithMetadata`) to create a project record so the project shows up in the TUI per SKETCH §9.3. Use the existing service constructor — do NOT bypass into the SQLite adapter directly.
  - The project name comes from the user input (TUI) or the JSON `name` field. Other project fields (HyllaArtifactRef, RepoBareRoot, etc.) are populated from sensible cwd-derived defaults; consult `internal/app/service.go:CreateProject*` signature when wiring.
  - **Idempotency:** if a project with the same name already exists (which can happen on re-run from the same cwd), the function reports it but does NOT error or duplicate. Decide between "project already exists; skipping" and "project found; updated metadata" based on what `Service.CreateProject*` already supports — confirm via Read on service.go before deciding.
  - Closing message uses `writeCLIKV` (existing helper in `cmd/till/main.go:1934`) to print a Laslig key/value table summarizing: project name, group, agents-dir path, agents.toml path, .gitignore status, .mcp.json status, project-DB status, added/skipped counts.
  - Tests:
    - `TestInit_CreatesProjectRecord`: after `till init --json '{...}'`, the project appears in the underlying DB via the service layer's list method.
    - `TestInit_SuccessMessage_Format`: assert key strings appear in stdout (`"project name"`, `"group"`, `"agents copied"`, `"added"`, `"skipped"`).
  - `mage test-pkg ./cmd/till` passes.
- **Blocked by:** D6 (D7 modifies `cmd/till/init_cmd.go` which D6 last touched — same-file lock).
- **Notes for builder:**
  - Spinning up the `Service` in `runInitJSON` / `runInitTUI` requires the same wiring `cmd/till/main.go` already uses — likely going through `resolveRuntimePaths` + `ensureRuntimePathParents` + a helper that constructs the SQLite adapter. Read `cmd/till/main.go` for an existing command (e.g. `projectCmd`) that already does this and clone the pattern.

### Droplet 4c.6.W2.D7.5 — `till install` CLI command (NEW — OQ#3 disposition)

- **State:** todo
- **Paths:**
  - `cmd/till/install_cmd.go` (NEW)
  - `cmd/till/install_cmd_test.go` (NEW)
  - `cmd/till/main.go` (modify: build `installCmd := newInstallCommand(...)` + add to `rootCmd.AddCommand(...)` line 1904)
  - `cmd/till/help.go` (modify: add `"till install"` entry to the `commandHelpSpecs` rich-help table)
- **Packages:** `github.com/evanmschultz/tillsyn/cmd/till`
- **Acceptance:**
  - `cmd/till/install_cmd.go` exports `newInstallCommand(stdout io.Writer, rootOpts rootCommandOptions) *cobra.Command` returning a `*cobra.Command` with `Use: "install"`, descriptive help/long/example, `cobra.NoArgs`. The `RunE` invokes `runInstall(stdout, rootOpts)`.
  - `runInstall` performs the SAME dev-config-creation behavior currently in `cmd/till/main.go:2039 runInitDevConfig` — verbatim port preserves: (a) `platform.DefaultPathsWithOptions(platform.Options{AppName: opts.appName, DevMode: true, HomeDir: opts.homeDir})` resolution, (b) `os.MkdirAll` + create-if-missing of `<dev-paths>/till.toml` from `config.DefaultTemplate()`, (c) `ensureLoggingSectionDebug` rewrite of the logging section to `debug`, (d) `writeCLIKV` Laslig success message with status / config path / logging level keys.
  - `runInstall` is implemented in `install_cmd.go` (NOT `main.go`) — D7.5 lifts the body of `runInitDevConfig` from `main.go` into `install_cmd.go:runInstall` byte-equivalent (but renamed). D8 then removes the now-orphaned `runInitDevConfig` from `main.go`.
  - `cmd/till/main.go:1904` `rootCmd.AddCommand(...)` line includes `installCmd`.
  - `cmd/till/help.go` rich-help table includes a `"till install"` entry.
  - `cmd/till/install_cmd_test.go` ports the existing `TestRunInitDevConfigCreatesDebugConfig` (`main_test.go:2906`) and `TestRunInitDevConfigUpdatesExistingConfig` (`main_test.go:2955`) into `TestRunInstall_CreatesDebugConfig` and `TestRunInstall_UpdatesExistingConfig`. Same body, just `[]string{"install"}` instead of `[]string{"init-dev-config"}`. The originals stay in `main_test.go` until D8 removes them.
  - `TestShellEscapePath` (`main_test.go:3105`) is NOT moved — `shellEscapePath` itself stays in `main.go` as a shared helper used by `runInstall` (which now lives in `install_cmd.go` and imports `shellEscapePath` from the same package).
  - `mage test-pkg ./cmd/till` passes — both old `init-dev-config` tests AND new `install` tests are green. (Old tests stay until D8.)
  - `mage ci` green.
- **Blocked by:** D3 (shares `cmd/till/main.go` file-lock with D3's `rootCmd.AddCommand` modification — D3 lands first; D7.5 amends the same line).
- **Notes for builder:**
  - **Critical:** D7.5 LIFTS-and-RENAMES the `runInitDevConfig` body into `runInstall` in a new file. It does NOT delete the original — that's D8's job. The duplication is intentional and short-lived: D7.5 leaves D8 in charge of removing the old. This keeps the file-lock graph clean (D7.5 adds; D8 removes).
  - **Why not a same-droplet add+remove?** Because D7.5 + D8 have different scopes: D7.5 ADDS a CLI command (low risk, additive); D8 REMOVES a CLI command (higher risk — touches `help.go`, `main_test.go` with multiple test functions, the `rootCmd.AddCommand` line, the `commandHelpSpecs` map). Splitting lets D7.5 ship + verify-green BEFORE D8's surgery starts.

### Droplet 4c.6.W2.D8 — Remove `init-dev-config` from `main.go` + `help.go` + `main_test.go`

- **State:** todo
- **Paths:**
  - `cmd/till/main.go` (modify: REMOVE `initDevConfigCmd := &cobra.Command{...}` block at lines 1884-1903; REMOVE `initDevConfigCmd` from the `rootCmd.AddCommand(...)` call at line 1904; REMOVE the `runInitDevConfig` function definition at lines 2039-2094)
  - `cmd/till/help.go` (modify: REMOVE the `"till init-dev-config"` entry at lines 377-390 from the `commandHelpSpecs` map)
  - `cmd/till/main_test.go` (modify: at line 476 remove `"init-dev-config"` from the registered-commands assertion list; at lines 732-734 remove the `init-dev-config` rich-help table-test row; REMOVE the test functions `TestRunInitDevConfigCreatesDebugConfig` (line 2906), `TestRunInitDevConfigUpdatesExistingConfig` (line 2955), and update the `TestShellEscapePath` doc-comment at line 3105 to drop the "init-dev-config" mention since the equivalent path-output behavior now lives under `till install` and is covered by D7.5's `TestRunInstall_*` tests)
- **Packages:** `github.com/evanmschultz/tillsyn/cmd/till`
- **Acceptance:**
  - `git grep -n init-dev-config cmd/till/` returns ZERO matches after D8 commits.
  - `git grep -n runInitDevConfig cmd/till/ internal/` returns ZERO matches after D8 commits.
  - `cmd/till/main.go` no longer contains the `initDevConfigCmd` block or the `runInitDevConfig` function.
  - `cmd/till/help.go` no longer contains the `"till init-dev-config"` entry in `commandHelpSpecs`.
  - `cmd/till/main_test.go` no longer references `init-dev-config` in any test name, table-test row, or assertion list.
  - `mage test-pkg ./cmd/till` passes — D7.5's `TestRunInstall_CreatesDebugConfig` + `TestRunInstall_UpdatesExistingConfig` cover the equivalent behavior; coverage does NOT regress.
  - `mage ci` green.
- **Blocked by:** D3 (shares `cmd/till/main.go` file-lock — D3's add lands first per L1 directive: "D3 first since D8 only removes; safer than reverse"); D7.5 (D8's behavior-removal is a regression unless D7.5 has shipped the replacement `till install` first — OQ#3 verification finding).
- **Notes for builder:**
  - **Pre-flight check before deletion:** D8 builder MUST run `mage test-pkg ./cmd/till` against the current state and confirm `TestRunInstall_CreatesDebugConfig` + `TestRunInstall_UpdatesExistingConfig` are present and passing. If they are NOT (i.e. D7.5 didn't ship those tests under those exact names), STOP and escalate to orchestrator — D8's removal premise depends on D7.5 having landed equivalent coverage under the new names.
  - The `TestShellEscapePath` test at `main_test.go:3105` does NOT need to be removed — `shellEscapePath` is still in `main.go` and still used by `runInstall` (in `install_cmd.go`). D8 only updates the doc-comment to drop the "init-dev-config" mention.
  - Be mindful when removing the rich-help table-test row (`main_test.go:732-734`) that other rows have similar shape — use line-anchored deletion, not pattern-match.

## Notes

- **OQ#3 verification result is a load-bearing finding.** The L1 directive's premise that "`till install` covers (or is extended to cover) the dev-config-creation behavior" is FALSE — `till install` does NOT exist in the CLI today (only `mage install` exists, which is a build target, not a CLI subcommand). SKETCH §9.1 names `till install` as the destination, so the disposition is to add `till install` as a NEW CLI command (D7.5) before D8 finalizes the removal of `init-dev-config`. This is documented inline in the "OQ#3 Verification" section above and reflected in the droplet decomposition (D7.5 is new; D8 is `Blocked by` D7.5 in addition to D3).

- **OQ#2 verification result.** Both D1 and D2 acceptance criteria explicitly require the 2-3 line `// DO NOT EDIT — re-vendor from upstream.` block-comment header on every vendored Go file. Provenance also lives in `internal/vendor/VENDOR_SOURCE.md` (created by D1, appended by D2).

- **Vendor packages are NEW (verified 2026-05-09).** `internal/vendor/` does not exist; `fsatomic` and `configmerge` are not in `go.mod` (only nominal substring matches in unrelated dependency names). D1+D2 are clean greenfield package additions — they do not collide with `cmd/till` and run fully parallel until D5 needs them.

- **Same-package serialization (`cmd/till`).** D3, D4, D5, D6, D7, D7.5, D8 all live in `cmd/till` and most touch `cmd/till/init_cmd.go` directly. The serial chain D3 → D4 → D5 → D6 → D7 reflects the same-file lock on `init_cmd.go`. D7.5 introduces a NEW file (`install_cmd.go`) but shares `cmd/till/main.go` with D3, so D7.5 is `Blocked by: D3`. D8 modifies BOTH `main.go` and `init_cmd.go`-adjacent surfaces (`help.go`, `main_test.go`) and is `Blocked by: D3, D7.5`.

- **D5 cross-package blockers.** D5 is `Blocked by: D1, D2, D4` because D5 imports `internal/vendor/fsatomic` and `internal/vendor/configmerge` (D1+D2 are NEW packages that must exist before D5 compiles), and D5 modifies `cmd/till/init_cmd.go` which D4 last edited (same-file lock).

- **Sizing watch for D3.** D3 combines skeleton + flag wiring + JSON parser + `main.go` register + `help.go` entry. Builder is asked to escalate if combined LOC exceeds ~120 — possible split is D3a (skeleton + register, no JSON) and D3b (JSON parser + validation). Not pre-decided here; flagged for the builder to surface.

- **Re-run safety is a hard invariant.** D5's idempotency tests are mandatory — every file write must check existence first and skip-not-overwrite. Re-running `till init` in an already-initialized project is the most common dev workflow and must be safe.

- **TUI vs JSON mode equivalence.** D5+D6+D7 enforce the SKETCH §26.W2 RiskNote that TUI and JSON modes call the same downstream pipeline; only the input source differs. Builder includes at least one test that runs both modes against the same `t.TempDir()` and asserts the resulting filesystem state is identical.

- **Hylla applicability to this plan.** Planning relied on `git grep` + `Read` for verification because (a) some surfaces are uncommitted work (`runInitDevConfig` is in committed `main.go`, but `init_cmd.go` is brand-new), and (b) `go.mod` / non-Go file shape checks aren't Hylla territory per project rule "Hylla today understands Go only." No Hylla miss occurred — see `## Hylla Feedback` in the planner closing response.
