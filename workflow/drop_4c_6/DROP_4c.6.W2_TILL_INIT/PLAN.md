# DROP_4c.6.W2 — TILL_INIT

**State:** planning
**Blocked by:** 4c.6.W1.D1 (W2 copies the agent .md files shipped by W1; without W1's embedded scaffolding there's nothing to copy)
**Paths (expected):** `internal/fsatomic/**`, `cmd/till/init_cmd.go`, `cmd/till/init_cmd_test.go`, `cmd/till/install_cmd.go`, `cmd/till/install_cmd_test.go`, `cmd/till/main.go`, `cmd/till/main_test.go`, `cmd/till/help.go`
**Packages (expected):** `github.com/evanmschultz/tillsyn/internal/fsatomic`, `github.com/evanmschultz/tillsyn/cmd/till`
**PLAN.md ref:** `workflow/drop_4c_6/PLAN.md` → `4c.6.W2` row (lines 117-133)
**Workflow:** `workflow/example/drops/WORKFLOW.md`
**Cascade concept:** `AGENT_CASCADE_DESIGN.md`
**Started:** 2026-05-09
**Closed:** —

## Scope

Land `till init` per `SKETCH.md` §9 + §26.W2 — TUI walk (project name + group picker), copy embedded `internal/templates/builtin/agents/<group>/*.md` → `<project>/.tillsyn/agents/*.md` FLAT, copy `agents.example.toml` → `<project>/agents.toml`, ensure `agents.local.toml` in `.gitignore`, optional `.mcp.json` registration, project-DB record creation, Laslig success message, JSON mode (`--json '{...}'`) with identical behavior, re-run safety (never overwrites). Plus: ship a local `internal/fsatomic/` atomic file-write helper (~30–50 LOC, zero deps) per `SKETCH.md` §9.6 — write-temp + rename pattern with cleanup-on-error and optional parent-dir fsync; ROUND-3 W2.D1 pivot from "vendor from `ta`" to "implement locally in Tillsyn" because `ta` is not yet at MVP and the pattern is small enough to own here (future migration to `hylla-shared` post-MVP is unaffected). Plus: ADD `till install` CLI command (NEW — see OQ#3 verification below) that takes over the dev-config-creation behavior currently in `cmd/till/main.go:2039 runInitDevConfig` BEFORE D8 finalizes the removal of `init-dev-config`. JSON-mode + TUI behaviors must be IDENTICAL apart from input source.

**ROUND-2 update (W2-FF4 — configmerge removed):** earlier rounds vendored `configmerge` from `ta` alongside `fsatomic`. Round-2 audit (W2-FF4) confirmed `configmerge` was never load-bearing in this wave — `agents.example.toml` is COPIED (not merged) per D5, and `.gitignore` is a single-line idempotent append. Removing the `configmerge` vendor saves a droplet and reduces vendored surface. If a future drop needs section-merging TOML behavior, vendor it then.

## OQ#3 Verification — `till install` Coverage Status

**Verification performed (2026-05-09):**

- `cmd/till/main.go:1885-1903` registers a `init-dev-config` Cobra command whose `RunE` calls `runInitDevConfig` (`cmd/till/main.go:2039-2094`).
- `runInitDevConfig` semantically does TWO things: (a) creates `<dev-paths>/till.toml` from `config.DefaultTemplate()` if it doesn't exist (lines 2059-2072), and (b) rewrites the `[logging]` section to `level = "debug"` (lines 2074-2083). Closes with a `writeCLIKV` Laslig table.
- `git grep -n "runInstall\|installCmd" cmd/till/ internal/` returns NO matches.
- `git grep -n init-dev-config cmd/till/` shows references in `main.go` (the registration), `help.go:377-390` (rich-help spec), and `main_test.go` (lines 476, 732-734, 2906, 2928-2938, 2955, 2988-2993, 3105 — five test functions plus the registered-commands list and the rich-help table-test).
- `magefile.go:140-145` defines a `mage install` build target (`mage install` writes the binary to `~/.local/bin/till`). This is the ONLY thing called "install" in the codebase. **It is a build-tool target, not a `till` CLI subcommand.**

**Verdict: `till install` does NOT exist as a CLI command today.** The L1 directive's premise that "`till install` covers (or is extended to cover) the dev-config-creation behavior" is FALSE. SKETCH §9.1 names `till install` as the destination ("install-time setup (DB creation, default config) folds into `till install`"), so the SKETCH explicitly intends a NEW `till install` CLI command — not a fold-into existing logic.

**Disposition: EXPAND W2 with a NEW droplet `D7.5` that creates `cmd/till/install_cmd.go` (the `till install` Cobra command) and ports the dev-config-creation behavior from `runInitDevConfig` into it.** D7.5 is `blocked_by` D3a (registers commands in `main.go`'s `rootCmd.AddCommand` call — D3a owns the additive register edit; D3b is `init_cmd.go`-only) and is a **hard precondition for D8** (D8's removal of `init-dev-config` is a behavior regression unless D7.5 lands first). D8's `Blocked by:` accordingly includes D7.5.

**Important:** the `init` and `install` commands are SEPARATE — they do different things:
- `till init` (D3a–D7): seeds a project (cwd-local). Copies agent `.md` files, writes `agents.toml`, updates `.gitignore`, optionally writes `.mcp.json`, creates the project DB record. Per-project setup.
- `till install` (D7.5): bootstraps the local Tillsyn dev environment (home-local). Creates `<dev-paths>/till.toml` with `[logging] level = "debug"`. Per-machine setup.

The two are wired via `main.go`'s `rootCmd.AddCommand` line; both share the `cmd/till/main.go` file lock so D8 ordering chain remains valid.

## Planner

### Droplet 4c.6.W2.D1 — `internal/fsatomic/` atomic file-write helper (local-implement)

**ROUND-3 PIVOT (W2.D1 local-implement, dev-approved 2026-05-11):** round-2 plan vendored `fsatomic` from `ta` upstream. Dev call: `ta` is not yet at MVP and the atomic-write pattern is small enough (~30–50 LOC) to own locally in Tillsyn. Pivot from "vendor from `ta`" → "implement locally at `internal/fsatomic/`." No upstream provenance, no `VENDOR_SOURCE.md`, no `internal/vendor/` directory, no "DO NOT EDIT — re-vendor from upstream" header rule. Future migration to `hylla-shared` post-MVP is unaffected by where the package originates.

- **State:** todo
- **Paths:**
  - `internal/fsatomic/atomic.go` (NEW)
  - `internal/fsatomic/atomic_test.go` (NEW)
- **Packages:** `github.com/evanmschultz/tillsyn/internal/fsatomic` (NEW package)
- **Acceptance:**
  - Package `fsatomic` exists at `internal/fsatomic/`. Exports the API surface W2.D5 needs at minimum: `WriteFile(path string, data []byte, perm os.FileMode) error` (atomic version of `os.WriteFile`) — write-to-temp-in-same-dir + sync + rename pattern.
  - Implementation uses `os.CreateTemp(filepath.Dir(target), filepath.Base(target)+".tmp-*")` to land the temp in the same directory as the target (required for `os.Rename` to be atomic on POSIX; cross-filesystem renames are NOT atomic).
  - Error paths cleanup the temp file via `defer` + a guard (don't double-remove on success). Partial writes never leave a temp file behind.
  - Sync semantics: `f.Sync()` on the temp file BEFORE close + rename. Optional helper for parent-dir fsync as a separate exported function (`SyncDir(path string) error`) if W2.D5 needs strong durability; if not, leave as a future addition (YAGNI).
  - Idempotency on existing target: `WriteFile` overwrites by default (matches `os.WriteFile`'s contract). Re-run safety is the CALLER's responsibility (D5's `os.Stat`-then-skip dance lives in `init_cmd.go`, not in fsatomic).
  - Tests at `internal/fsatomic/atomic_test.go`:
    - `TestWriteFile_FreshWrite`: write to a `t.TempDir()` path; assert file exists with the expected content + permissions.
    - `TestWriteFile_OverwritesExisting`: write to a pre-existing file; assert new content overwrites cleanly.
    - `TestWriteFile_CleansUpTempOnError`: inject an error into the write path (e.g., zero perms on the parent dir, or write to a path that can't exist); assert NO `.tmp-*` files remain in the parent dir after the failed call.
    - `TestWriteFile_PreservesPermissions`: write with `0o600`; assert resulting file mode matches.
    - `TestWriteFile_AtomicVisibility`: stretch goal — concurrent reader sees either the OLD content OR the NEW content, never a half-written file. Skip if too flaky on CI; documented as a future test.
  - `mage test-pkg ./internal/fsatomic` passes.
  - `mage ci` green.
  - Doc-comments: package-level doc-comment names the design (write-temp + rename), pins the same-directory-temp requirement for POSIX atomicity, and notes future-migration intent to `hylla-shared` post-MVP per SKETCH §9.6.
- **Blocked by:** —
- **Notes for builder:**
  - Per SKETCH §9.6: "52 LOC, zero deps." Implement minimally — no exotic API, no struct-based staged writes unless W2.D5's consumer actually needs them.
  - The package is a NEW Go package — no file-collision with `cmd/till` or anything else, so D1 runs fully parallel with D3a–D8 until D5 needs it.
  - **No upstream vendoring** — this droplet writes original code in the Tillsyn repo. No `DO NOT EDIT` header. No `VENDOR_SOURCE.md`. Future ports to `hylla-shared` can pull the API from this location directly.

### Droplet 4c.6.W2.D2 — REMOVED (W2-FF4: `configmerge` vestigial)

**ROUND-2 removal (W2-FF4):** round-1 D2 vendored `configmerge` from `ta` for use in D5's `.gitignore` ensure step. Round-2 audit confirmed `configmerge` is NOT load-bearing in this wave:

- D5 `copyAgentsTOML` is a COPY operation, not a merge (PLAN.md D5 acceptance line 135: "copies embedded `agents.example.toml` → `<destDir>/agents.toml` atomically").
- D5 `ensureGitignore` is a single-line idempotent append. Plan-QA-proof finding 2.3 + plan-QA-falsification finding 1.8 both flagged `configmerge` as the fallback ("hand-written line-presence check is acceptable"); the hand-written path is in fact the right path for the actual D5 use cases. No section-merging TOML behavior is needed in W2.
- Removing D2 saves a droplet, removes ~12kB of vendored Go + tests, drops one cross-package blocker on D5, and avoids the "one dep already in Tillsyn — verify in builder spawn" escalation hazard.

If a future drop needs section-merging TOML behavior, vendor `configmerge` then. D5 acceptance is updated below to use a hand-written `os.ReadFile` + `bytes.Contains` + `os.WriteFile` (atomically via `fsatomic`) sequence for `.gitignore` line-presence — no vendored merge dep.

### Droplet 4c.6.W2.D3a — `cmd/till/init_cmd.go` skeleton + register in `main.go` + help-entry

**ROUND-2 split (W2-FF1):** the round-1 D3 droplet touched 3 production files (`init_cmd.go` + `main.go` + `help.go`) at the under-decomposed smell threshold. Round 2 splits it into D3a (this droplet — skeleton + register + help-entry; no JSON parser) and D3b (JSON parser + table-test, `init_cmd.go`-only). D3b is `Blocked by: D3a`.

- **State:** done
- **Paths:**
  - `cmd/till/init_cmd.go` (NEW — skeleton only, no JSON parser body)
  - `cmd/till/init_cmd_test.go` (NEW — minimal smoke test that confirms `--json ""` (empty) and bare invocation both route through `RunE` and return the expected D3a-stage stub errors)
  - `cmd/till/main.go` (modify: add `initCmd := newInitCommand(...)` build + add `initCmd` to `rootCmd.AddCommand(...)` at line 1904)
  - `cmd/till/help.go` (modify: add a new entry to the rich-help table at the analogous position to the existing `"till init-dev-config"` block — section is the `commandHelpSpecs` map ending at line 391)
- **Packages:** `github.com/evanmschultz/tillsyn/cmd/till`
- **Acceptance:**
  - `cmd/till/init_cmd.go` exports `newInitCommand(stdout io.Writer, rootOpts rootCommandOptions) *cobra.Command` (or whichever signature matches the existing pattern in `main.go`'s sibling builders — verify by reading `main.go` for the `initDevConfigCmd := &cobra.Command{...}` shape and matching). Returns a `*cobra.Command` with `Use: "init"`, short + long help, `Example` block, `cobra.NoArgs` (or `cobra.MaximumNArgs(0)` to match local convention).
  - `--json` flag is wired with a `String` flag default `""` — but the parser body is a STUB in D3a; the flag is registered + readable but the value is not parsed. D3a's stub `RunE` dispatches: if `--json <payload>` is set (non-empty), it returns `errors.New("till init: JSON parse not yet wired (W2.D3b)")`; otherwise calls `runInitTUI(stdout, opts)` which itself returns `errors.New("till init: TUI walk not yet wired (W2.D4)")`. This keeps D3a's surface area to skeleton + register; D3b fills the JSON parser.
  - `cmd/till/main.go:1904` `rootCmd.AddCommand(...)` line includes `initCmd` (`initCmd` built earlier in the same function via `newInitCommand(...)`).
  - `cmd/till/help.go` rich-help table includes a `"till init"` entry analogous to the existing `"till init-dev-config"` entry at lines 377-390.
  - `cmd/till/init_cmd_test.go`: smoke test that `till init` (bare) returns the TUI-stub error AND `till init --json '{...}'` returns the JSON-stub error. No JSON-payload parsing assertions in D3a — those move to D3b.
  - **CONSUMER-TIE TEST CONTRACT (W2-FF6 ROUND-2 — symmetric to D7.5's W2-FF3 contract):** the D3a smoke tests MUST invoke `run(context.Background(), []string{"--app", "tillsyn-init", "init"}, &out, io.Discard)` (or equivalent end-to-end form) and exercise the route → `cobra` → `initCmd.RunE` chain. Do NOT call `cmd.RunE(...)` directly or invoke unexported helpers — that would ship a non-wired `init` command (the cobra registration in `main.go` would not be exercised). Same discipline as D7.5; pinned here so the smoke tests prove `init` is genuinely wired.
  - `mage test-pkg ./cmd/till` passes.
  - `mage ci` green.
- **Blocked by:** —
- **Notes for builder:**
  - File-locks: D3a writes `cmd/till/main.go` (registers init). D7.5 also writes `cmd/till/main.go` (registers install). D8 also writes `cmd/till/main.go` (removes init-dev-config). D3a lands first per L1 directive (additive before subtractive).
  - D3a is the FIRST `cmd/till/init_cmd.go` write — D3b, D4, D5, D6, D7 all serialize behind it on the same file lock.

### Droplet 4c.6.W2.D3b — `init_cmd.go` JSON-payload parser + group-validation + table-test

**ROUND-2 NEW (W2-FF1):** split out from round-1 D3 to keep D3a's surface to skeleton + register. D3b lives in `cmd/till/init_cmd.go` only — no `main.go` or `help.go` edits.

- **State:** done
- **Paths:**
  - `cmd/till/init_cmd.go` (modify: replace the D3a JSON-stub error in `RunE` with a real `runInitJSON` function that parses + validates the payload; the stub's downstream file-copy step still returns a D5-stub error in D3b — D5 fills that in)
  - `cmd/till/init_cmd_test.go` (modify: add table-driven JSON-payload parsing test cases — valid, invalid-group, malformed-JSON, missing-required-fields)
- **Packages:** `github.com/evanmschultz/tillsyn/cmd/till`
- **Acceptance:**
  - JSON payload struct defined in `init_cmd.go`: `type initJSONPayload struct { Name string \`json:"name"\`; Group string \`json:"group"\`; MCP bool \`json:"mcp"\` }`.
  - `runInitJSON` parses the payload via `encoding/json.Unmarshal` and validates `Group` is one of `{"till-gen", "till-go"}` (NOT `till-gdd` per SKETCH §9.3 — greyed-out until post-Hylla-rev). Invalid group → returns a wrapped error.
  - On valid parse, `runInitJSON` proceeds to call the file-copy pipeline; in D3b that pipeline is still a stub (D5 wires it). So D3b's `runInitJSON` ends with `return errors.New("till init: file copy not yet wired (W2.D5)")` after a successful parse + validation.
  - `cmd/till/init_cmd_test.go` table-driven test for JSON-payload parsing: valid (asserts the D5-stub error fires AFTER a successful parse — proving parse + validate ran), invalid-group, malformed-JSON, missing-required-fields. At least one test confirms `--json` and bare TUI invocations route to the right `RunE` branch.
  - `mage test-pkg ./cmd/till` passes.
  - `mage ci` green.
- **Blocked by:** D3a (same-file lock on `cmd/till/init_cmd.go`).
- **Notes for builder:**
  - D3b is `init_cmd.go`-only — DO NOT modify `main.go` or `help.go`. If you find yourself wanting to, escalate; the surface allocation between D3a and D3b is deliberate.

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
- **Blocked by:** D3b (D4 modifies `cmd/till/init_cmd.go` which D3b last touched — same-file lock; D3b is itself blocked on D3a, so D4 transitively waits for both).
- **Notes for builder:**
  - Use existing bubbletea infrastructure per SKETCH §26.W2 — `internal/tui/` packages have form/picker patterns; cite the file you're cloning the pattern from in `BUILDER_WORKLOG.md`.
  - Greyed-out `till-gdd` option must be UNSELECTABLE — pressing enter on it should be a no-op or play a UI bell, not advance to next step.

### Droplet 4c.6.W2.D5 — File-copy pipeline + `.gitignore` ensure (uses fsatomic)

- **State:** todo
- **Paths:**
  - `cmd/till/init_cmd.go` (modify: implement the file-copy pipeline that both `runInitTUI` and `runInitJSON` call)
  - `cmd/till/init_cmd_test.go` (modify: add file-copy + re-run-safety + `.gitignore`-idempotent tests against `t.TempDir()`)
- **Packages:** `github.com/evanmschultz/tillsyn/cmd/till`
- **Acceptance:**
  - `copyAgentFiles(destDir, group string)` reads embedded `internal/templates/builtin/agents/<group>/*.md` (FS exposed by W1.D1's embed.go) and writes to `<destDir>/.tillsyn/agents/*.md` FLAT (no group prefix). Uses `fsatomic` for atomic writes (write-temp + rename pattern).
  - `copyAgentsTOML(destDir)` copies embedded `internal/templates/builtin/agents.example.toml` → `<destDir>/agents.toml` atomically.
  - `ensureGitignore(destDir)` adds `agents.local.toml` to `<destDir>/.gitignore` (creates the file if absent; idempotent — re-run does NOT duplicate the line). **Implementation (W2-FF4 round-2 decision; W2-FF10 round-2 line-iteration fix):** hand-written `os.ReadFile` + LINE-ITERATION via `strings.Split(string(data), "\n")` (or `bufio.Scanner` against the file content) checking each trimmed line against the literal `"agents.local.toml"` for presence + `os.WriteFile` atomically via `fsatomic`. **Why line-iteration NOT raw `bytes.Contains([]byte("\nagents.local.toml\n"))`:** the raw-`bytes.Contains` form requires a leading `\n` and misses the first-line-only case (file consists solely of `agents.local.toml\n` from a prior run with no preceding entries). Line-iteration handles that case correctly. Handle the trailing-newline corner case (file ends with no `\n` → append `\n` + `agents.local.toml\n`; file ends with `\n` → append `agents.local.toml\n`). NO `configmerge` dependency — round-1 considered it; round-2 removed D2 because section-merging TOML behavior is not needed here.
  - **Re-run safety:** every write goes through a `_, err := os.Stat(target); errors.Is(err, fs.ErrNotExist)` pre-check. Existing files are skipped, NOT overwritten. The function returns counts: `added int, skippedExisting int`.
  - JSON-mode and TUI-mode both call the same `copyAgentFiles` + `copyAgentsTOML` + `ensureGitignore` sequence — behavior IDENTICAL apart from input source.
  - Tests in `init_cmd_test.go`:
    - `TestInit_FreshDir_CopiesAllFiles`: empty `t.TempDir()`, `till init --json '{"name":"foo","group":"till-go","mcp":false}'`, asserts 7 agent .md files appear under `.tillsyn/agents/` + `agents.toml` exists + `.gitignore` contains `agents.local.toml`.
    - `TestInit_RerunSafety_NoOverwrite`: run init twice; second run reports `added=0, skipped=N` and does NOT mutate any existing file (compare mtimes or hashes).
    - `TestInit_GitignoreIdempotent`: `.gitignore` already contains `agents.local.toml`; re-run does NOT add a duplicate line.
    - `TestInit_PreExistingGitignore_AppendsCleanly`: `.gitignore` exists with unrelated entries; re-run appends `agents.local.toml` once with proper newline handling.
  - `mage test-pkg ./cmd/till` passes.
- **Blocked by:** D1 (uses `fsatomic`), D4 (D5 modifies `cmd/till/init_cmd.go` which D4 last touched — same-file lock). **W2-FF4 round-2:** D2 dropped from the blocker list — `configmerge` no longer vendored.
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

- **State:** done
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
  - `cmd/till/install_cmd_test.go` ports the existing `TestRunInitDevConfigCreatesDebugConfig` (`main_test.go:2906`, no-underscore-camelCase shape) and `TestRunInitDevConfigUpdatesExistingConfig` (`main_test.go:2955`, no-underscore-camelCase shape) into `TestRunInstall_CreatesDebugConfig` and `TestRunInstall_UpdatesExistingConfig` (**WITH underscore between `TestRunInstall` and the rest** — deliberate shape change). Same test body, just `[]string{"install"}` instead of `[]string{"init-dev-config"}` for the args slice. **TEST-NAME SHAPE DISAMBIGUATION (W2-FF9 ROUND-2):** the verbatim-port framing ("Same body...") refers to the test's BODY only, NOT the function name. The new test function names introduce an underscore that the no-underscore originals did not have — `TestRunInstallCreatesDebugConfig` (no underscore) is INCORRECT and would fail D8's pre-flight check. The exact correct names are `TestRunInstall_CreatesDebugConfig` and `TestRunInstall_UpdatesExistingConfig`. The originals stay in `main_test.go` until D8 removes them.
  - **TEST-NAME CONTRACT (W2-FF2 ROUND-2):** the test names `TestRunInstall_CreatesDebugConfig` and `TestRunInstall_UpdatesExistingConfig` are a HARD CONTRACT between D7.5 and D8 — D8's pre-flight check (this droplet's twin in the chain) hard-codes these exact names when verifying D7.5 has shipped equivalent coverage before deleting the originals. **If you rename either test in D7.5, you MUST update D8's pre-flight bullet at the same time.** Renaming silently breaks D8's "coverage does NOT regress" gate.
  - **CONSUMER-TIE TEST CONTRACT (W2-FF3 ROUND-2):** each new test MUST invoke `run(context.Background(), []string{"--app", "tillsyn-init", "install"}, &out, io.Discard)` end-to-end (NOT call `runInstall(...)` directly). This exercises the route → `cobra` → `installCmd.RunE` → `runInstall` chain and proves the command is genuinely wired. Calling `runInstall` directly would ship a non-wired install command (the cobra registration in `main.go` would not be exercised). Verify wiring via `mage test-func ./cmd/till TestRunInstall_CreatesDebugConfig`.
  - **LASLIG TITLE CONTRACT (W2-FF5 ROUND-2):** `runInstall`'s `writeCLIKV` first arg (the table title) is `"Dev Config"` — preserved BYTE-FOR-BYTE from the existing `runInitDevConfig` body at `cmd/till/main.go:2089`. The ported test bodies at `main_test.go:2936` and `main_test.go:2991` assert `"Dev Config"` substring in the output; preserving the title keeps the verbatim port mechanical and decision-free. **DO NOT rename the title to `"Install"` or any other string.** If a future drop wants the title to say `"Install"` instead, that is a separate user-visible-label-rename droplet — out of scope for D7.5.
  - `TestShellEscapePath` (`main_test.go:3105`) is NOT moved — `shellEscapePath` itself stays in `main.go` as a shared helper used by `runInstall` (which now lives in `install_cmd.go` and imports `shellEscapePath` from the same package).
  - `mage test-pkg ./cmd/till` passes — both old `init-dev-config` tests AND new `install` tests are green. (Old tests stay until D8.)
  - `mage ci` green.
- **Blocked by:** D3a (shares `cmd/till/main.go` file-lock with D3a's `rootCmd.AddCommand` modification — D3a lands first; D7.5 amends the same line). **W2-FF1 round-2 update:** was `D3` round-1; D3a is the part of the round-1 D3 split that owns `main.go` + `help.go`, so the `main.go` file-lock binds D7.5 to D3a (not D3b, which is `init_cmd.go`-only).
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
- **Blocked by:** D3a (shares `cmd/till/main.go` file-lock — D3a's add lands first per L1 directive: "D3 first since D8 only removes; safer than reverse"; round-2 W2-FF1 split renamed D3 → D3a for the `main.go`-touching half); D7.5 (D8's behavior-removal is a regression unless D7.5 has shipped the replacement `till install` first — OQ#3 verification finding).
- **Notes for builder:**
  - **Pre-flight check before deletion (W2-FF2 ROUND-2 contract):** D8 builder MUST run `mage test-pkg ./cmd/till` against the current state and confirm `TestRunInstall_CreatesDebugConfig` + `TestRunInstall_UpdatesExistingConfig` are present and passing. **These exact names are a hard contract pinned in D7.5's acceptance** — if they are NOT present (i.e. D7.5 didn't ship those tests under those exact names), STOP and escalate to orchestrator. The names are not internal builder choice; renaming requires updating BOTH D7.5 AND this pre-flight bullet at the same time. Cross-reference: D7.5 acceptance "TEST-NAME CONTRACT (W2-FF2 ROUND-2)".
  - The `TestShellEscapePath` test at `main_test.go:3105` does NOT need to be removed — `shellEscapePath` is still in `main.go` and still used by `runInstall` (in `install_cmd.go`). D8 only updates the doc-comment to drop the "init-dev-config" mention.
  - **D8 doc-comment phrasing (W2-PF1 ROUND-2 carryforward, pinned at orchestrator dispatch time):** the exact replacement string for the `init-dev-config` mention in `TestShellEscapePath`'s doc-comment is set by the orchestrator in the D8 spawn prompt — NOT a builder-discretion choice. Builder reads the spawn-prompt directive and applies it verbatim. (Round-1 NIT 1.6 + round-2 W2-PF1 left the exact string unspecified at plan-level; orchestrator pins it inline at D8 dispatch time so the plan stays decision-free.)
  - Be mindful when removing the rich-help table-test row (`main_test.go:732-734`) that other rows have similar shape — use line-anchored deletion, not pattern-match.

## Notes

- **OQ#3 verification result is a load-bearing finding.** The L1 directive's premise that "`till install` covers (or is extended to cover) the dev-config-creation behavior" is FALSE — `till install` does NOT exist in the CLI today (only `mage install` exists, which is a build target, not a CLI subcommand). SKETCH §9.1 names `till install` as the destination, so the disposition is to add `till install` as a NEW CLI command (D7.5) before D8 finalizes the removal of `init-dev-config`. This is documented inline in the "OQ#3 Verification" section above and reflected in the droplet decomposition (D7.5 is new; D8 is `Blocked by` D7.5 in addition to D3a — round-2 W2-FF1 split renamed D3 → D3a for the `main.go`-touching half).

- **OQ#2 verification result (SUPERSEDED by ROUND-3 W2.D1 pivot).** Round-2 D1 required vendor scaffolding — `DO NOT EDIT` headers + `VENDOR_SOURCE.md`. ROUND-3 pivots D1 to local-implement at `internal/fsatomic/` per dev call 2026-05-11 (`ta` not yet at MVP; ~50 LOC is small enough to own here). No vendor scaffolding, no provenance file, no header rule. Future migration to `hylla-shared` post-MVP pulls the API from `internal/fsatomic/` directly.

- **`internal/fsatomic/` is NEW (verified 2026-05-09; ROUND-3 path pivoted from `internal/vendor/fsatomic/`).** `internal/fsatomic/` does not exist; `fsatomic` is not in `go.mod` (only nominal substring matches in unrelated dependency names). D1 is a clean greenfield package addition — it does not collide with `cmd/till` and runs fully parallel until D5 needs it. _(W2-FF4 round-2: `configmerge` was originally going to be a second vendor droplet; round-2 removed it as vestigial — see Scope §ROUND-2 update.)_

- **Same-package serialization (`cmd/till`).** D3a, D3b, D4, D5, D6, D7, D7.5, D8 all live in `cmd/till` and most touch `cmd/till/init_cmd.go` directly. The serial chain D3a → D3b → D4 → D5 → D6 → D7 reflects the same-file lock on `init_cmd.go` (D3a creates it; D3b–D7 each modify it in turn). D7.5 introduces a NEW file (`install_cmd.go`) but shares `cmd/till/main.go` with D3a, so D7.5 is `Blocked by: D3a`. D8 modifies BOTH `main.go` and `init_cmd.go`-adjacent surfaces (`help.go`, `main_test.go`) and is `Blocked by: D3a, D7.5`.

- **D5 cross-package blocker.** D5 is `Blocked by: D1, D4` because D5 imports `internal/fsatomic` (D1 is a NEW package that must exist before D5 compiles; ROUND-3 pivot from `internal/vendor/fsatomic` to local-implement), and D5 modifies `cmd/till/init_cmd.go` which D4 last edited (same-file lock). _(W2-FF4 round-2: D2 removed; `configmerge` not used.)_

- **D3 split (W2-FF1 round-2).** Round-1 D3 combined skeleton + flag wiring + JSON parser + `main.go` register + `help.go` entry across 3 production files at the under-decomposed smell threshold. Round-2 split it into D3a (skeleton + register + help-entry; touches `init_cmd.go` + `main.go` + `help.go`) and D3b (JSON parser + validation + table-test; touches `init_cmd.go` only). D3b `Blocked by: D3a`. Downstream chain rewired: D4 `Blocked by: D3b` (was `D3`); D7.5 `Blocked by: D3a` (was `D3`); D8 `Blocked by: D3a, D7.5` (was `D3, D7.5`).

- **Re-run safety is a hard invariant.** D5's idempotency tests are mandatory — every file write must check existence first and skip-not-overwrite. Re-running `till init` in an already-initialized project is the most common dev workflow and must be safe.

- **TUI vs JSON mode equivalence.** D5+D6+D7 enforce the SKETCH §26.W2 RiskNote that TUI and JSON modes call the same downstream pipeline; only the input source differs. Builder includes at least one test that runs both modes against the same `t.TempDir()` and asserts the resulting filesystem state is identical.

- **Hylla applicability to this plan.** Planning relied on `git grep` + `Read` for verification because (a) some surfaces are uncommitted work (`runInitDevConfig` is in committed `main.go`, but `init_cmd.go` is brand-new), and (b) `go.mod` / non-Go file shape checks aren't Hylla territory per project rule "Hylla today understands Go only." No Hylla miss occurred — see `## Hylla Feedback` in the planner closing response.
