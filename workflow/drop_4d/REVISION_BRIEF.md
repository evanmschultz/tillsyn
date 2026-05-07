# Drop 4d — Codex Adapter (Revision Brief)

**Status:** revision-brief authoring 2026-05-06.
**Author:** orchestrator (post-Drop-4c-merge / Drop-4c.5-in-flight).
**Drop scope (LOCKED):** add `codex` as the second `CLIAdapter` exercising the Drop 4c F.7.17 seam. Setup work for Drop 5 multi-CLI dogfood from day one.
**Out of scope:** F.4 marketplace CLI (Drop 4d-prime, post-Drop-5); Drop 5 dogfood validation; non-JSONL adapters (hard-cut interface rewrite is post-MVP); pre-execution permission interception (codex's `exec_approval_request` flow); cost synthesis from codex `token_count` events.

## 1. Hard Prerequisites

- Drop 4a + 4b + 4c on `main`. HEAD `5021174` at brief authoring time (Drop 4c.5 in flight in `workflow/drop_4c_5/`).
- **Drop 4c.5 closed and merged before Drop 4d builders fire.** Themes A (silent-data-loss), B (escape hatches), and Theme E residue affect dispatcher correctness — Drop 4d's planner does NOT need them, but Drop 5's multi-CLI dogfood does. The cleanest cut is "4c.5 lands → Drop 4d planner spawns → Drop 4d ships → Drop 5 dogfoods both."
- F.7 spawn pipeline shipped — `internal/app/dispatcher/cli_claude/`, `cli_adapter.go`, `bundle.go`, `monitor.go`, `handshake.go`, `commit_agent.go`, `gate_*.go`, `orphan_scan.go`, `binding_resolved.go`, `permission_grants*` storage, `cli_claude/render/`. The codex adapter slots into the same registry (`dispatcher.RegisterAdapter` at `cli_adapter.go:33-49`, `spawn.go:240` example).
- **Codex CLI binary available on the dev machine.** `codex --version` resolvable via `$PATH`. Verified-once-by-dev assumption — the dispatcher's `LookPath` is via `cmd.Env` PATH at exec time, not at template-load time.
- **HARD PRECONDITION (Q1 routing):** dev captures `testdata/codex_stream_minimal.jsonl` from a real `codex exec --json` run BEFORE Drop 4d builders start. Without the fixture, droplet 4d.3 (stream.go) cannot pin the event taxonomy. See §9 Q1.

## 2. Goal

Add `codex` as the second `CLIAdapter` so the F.7.17 seam stops being a one-adapter abstraction (cheating against the abstraction's stated purpose). Sets up Drop 5 to validate cascade-on-itself with multi-CLI dogfood from day one — a single-CLI Drop 5 would relitigate sequencing later, with a second forced re-spin to add codex.

The codex adapter is also the falsification of the Drop 4c `CLIAdapter` interface: if codex's surface forces interface changes, those changes must land in Drop 4d, NOT Drop 5. Drop 4d is the seam-validation drop; Drop 5 is the cascade-validation drop. Conflating them was the failure mode the swap-sequence fixes.

## 3. Scope (~7-9 droplets)

Three logical phases — the planner may collapse / split based on file-overlap.

### 3.1 Phase 1 — Type / enum / preflight extension (~2 droplets)

- **4d.1 — `CLIKindCodex` constant + `IsValidCLIKind` extension.**
  - Files: `internal/app/dispatcher/cli_adapter.go`.
  - Adds `const CLIKindCodex CLIKind = "codex"` and a second case in the `IsValidCLIKind` switch at `cli_adapter.go:43-48`.
  - Updates `BindingResolved.CLIKind` doc-comment at `cli_adapter.go:108-111` to remove "Drop 4c only" qualifier.
  - Acceptance: `mage test-pkg internal/app/dispatcher` green; `IsValidCLIKind(CLIKindCodex)` returns true; `IsValidCLIKind("")` still returns false (L15 default semantic preserved).
  - Blocked_by: none.

- **4d.2 — Plugin-preflight per-adapter routing.**
  - Files: `internal/app/dispatcher/plugin_preflight.go`, possibly `cli_adapter.go` (interface extension or sentinel).
  - Today `tillsyn.requires_plugins` calls `claude plugin list --json` unconditionally. Codex has no plugin model. Routing: skip preflight when binding's resolved `CLIKind == CLIKindCodex`. Two implementation options — planner picks:
    - **Option A (locked-cleanness)**: hardcode the skip in `plugin_preflight.go` with `if cliKind == CLIKindCodex { return nil }`. Simple; couples preflight to the closed enum.
    - **Option B (per-adapter contract)**: add a fourth method `Preflight(ctx, requiresPlugins []string) error` to `CLIAdapter` with claude calling its real impl and codex returning nil. Cleaner; widens the interface.
  - Acceptance: codex-only spawn with `requires_plugins = ["something"]` declared on the project skips preflight without erroring. Claude-only spawn behavior unchanged.
  - Blocked_by: 4d.1 (needs `CLIKindCodex` constant).

### 3.2 Phase 2 — Codex package implementation (~4-5 droplets)

Mirrors `internal/app/dispatcher/cli_claude/` package layout per `CLI_ADAPTER_AUTHORING.md` §1.

- **4d.3 — `cli_codex` package skeleton + `adapter.go`.**
  - Paths: NEW `internal/app/dispatcher/cli_codex/adapter.go`, NEW `internal/app/dispatcher/cli_codex/init.go`.
  - `codexAdapter` struct + `New() dispatcher.CLIAdapter` constructor + `var _ dispatcher.CLIAdapter = (*codexAdapter)(nil)` compile-time assertion. `init()` calls `dispatcher.RegisterAdapter(dispatcher.CLIKindCodex, New())`. Constant `codexBinaryName = "codex"`. Per L1 / L2 / REV-1 NO `command` override.
  - Acceptance: `mage test-pkg internal/app/dispatcher/cli_codex` green; package compiles; blank import in test file confirms registry wiring.
  - Blocked_by: 4d.1.

- **4d.4 — `cli_codex/argv.go` (assembleArgv).**
  - Files: NEW `internal/app/dispatcher/cli_codex/argv.go`.
  - Codex argv shape (per Context7 `/openai/codex` `codex-rs/exec/src/cli.rs` evidence):
    ```
    codex exec --json --color never \
      --cd <bundle.Root or working dir> \
      --skip-git-repo-check \
      --output-last-message <bundle.Root>/codex_last_message.txt \
      [--ephemeral] \
      [--full-auto] \
      [--add-dir <path>...] \
      [--output-schema <path>] \
      -  # read prompt from stdin
    ```
    NO `--bare` / `--plugin-dir` / `--system-prompt-file` / `--mcp-config` / `--settings` / `--permission-mode` / `--strict-mcp-config` / `--no-session-persistence` (those are claude-only).
  - Open subdecisions for planner: working-dir source (bundle.Root vs project worktree), `--full-auto` vs `--ephemeral` vs `--dangerously-bypass-approvals-and-sandbox` for cascade-managed spawns. **Recommendation**: `--full-auto` (sandboxed workspace-write) + `--ephemeral` (no persisted session files) for cascade-spawned codex; never `--dangerously-bypass`.
  - Conditional pointer-typed flags from `BindingResolved`: `MaxTurns`, `Effort` (codex calls it reasoning effort, set via `--config model_reasoning_effort=...`), `Model` (`--model`). `MaxBudgetUSD` does NOT translate — codex has no budget cap flag; document the silent skip in argv.go's doc-comment.
  - Stable argv order for test snapshotting (mirror cli_claude/argv.go pattern).
  - Acceptance: argv unit tests pin shape against a recorded snapshot; conditional emission on/off semantics verified per pointer-nil.
  - Blocked_by: 4d.3.

- **4d.5 — `cli_codex/env.go` (assembleEnv) + `CODEX_HOME` per-spawn override.**
  - Files: NEW `internal/app/dispatcher/cli_codex/env.go`.
  - Same closed POSIX baseline as cli_claude (18 vars per L4) — direct lift from `cli_claude/env.go:38-58`. Same `binding.Env` allow-list rules. Same `os.LookupEnv` failure-loud semantic per F.7.17 P5.
  - **Codex-specific addition**: `CODEX_HOME=<bundle.Root>/codex_home/`. This is the bundle-isolation knob — codex reads `~/.codex/config.toml` by default; setting `CODEX_HOME` forces it to read a per-spawn config.toml the dispatcher renders. Without this knob, every codex spawn shares the dev's `~/.codex/` and per-spawn isolation breaks.
  - Acceptance: env unit tests confirm `CODEX_HOME` present in returned slice with bundle-relative path; orchestrator-only env vars don't leak through.
  - Blocked_by: 4d.3.

- **4d.6 — `cli_codex/stream.go` (parseStreamEvent + extractTerminalReport).**
  - Files: NEW `internal/app/dispatcher/cli_codex/stream.go`.
  - **DEPENDS ON Q1 RESOLUTION** — the recorded `codex_stream_minimal.jsonl` fixture pins the event-type strings. Per Context7 partial evidence the codex `EventMsg` enum (Rust, public source) includes `session_configured`, `task_started`, `agent_message`, `agent_message_delta`, `exec_command_begin`, `exec_command_end`, `task_complete`, `error`, `token_count`. Mapping into canonical `StreamEvent`:
    - `session_configured` → `Type: "system_init"` (cross-CLI canonical from `SPAWN_PIPELINE.md` §"Stream-JSON Event Taxonomy").
    - `agent_message` / `agent_message_delta` → `Type: "assistant"` with `Text` populated from `message` / `delta` field.
    - `task_complete` → `Type: "result", IsTerminal: true`.
    - `error` → `Type: "error"` (forward-compat passthrough; not terminal unless co-occurring with `task_complete`).
    - `token_count` → `Type: "usage"` (passthrough; not surfaced to monitor today).
    - Unknown types → forward-compat passthrough per cli_claude pattern.
  - `extractTerminalReport`: `Cost = nil` for codex spawns (no `total_cost_usd` on `task_complete`). `Reason` from `task_complete.reason` if present. `Denials = nil` (codex's denial path is `error` events or pre-approval, not terminal-bundled).
  - Acceptance: 3-line fixture round-trips through `ParseStreamEvent` + `ExtractTerminalReport`; cross-CLI contract test (TestCLIAdapterContractTableDriven in `mock_adapter_test.go:555`) passes for codex row.
  - Blocked_by: 4d.3, **Q1 fixture available**.

- **4d.7 — `cli_codex/render/` per-spawn bundle render (`config.toml` + `AGENTS.md`).**
  - Files: NEW `internal/app/dispatcher/cli_codex/render/render.go`, NEW `internal/app/dispatcher/cli_codex/render/init.go`.
  - Mirrors `cli_claude/render/` shape — `Render(ctx, ...) error` writes per-spawn artifacts under `<bundle.Root>/codex_home/`:
    - `<bundle.Root>/codex_home/config.toml` — minimal codex config (model, profile selection, `[mcp_servers]` if Tillsyn-MCP self-registration is in scope; **planner decides**: lean SKIP MCP for Drop 4d, surface as drop-extending refinement).
    - `<bundle.Root>/codex_home/AGENTS.md` — codex's system-instruction equivalent. Render cross-CLI `system-prompt.md` (already at `<bundle.Root>/system-prompt.md`) into AGENTS.md form OR symlink. **Recommendation**: render via simple write (no symlink — Windows refinement risk + macOS sandbox edge cases).
  - Permission-grants injection: codex doesn't have `permissions.allow|ask|deny` in `settings.json` form. Codex grants land in `[shell_environment_policy]` / `[sandbox]` blocks of config.toml. **Drop 4d punts on grants persistence for codex** — adapter writes empty grants section. Permission_grants table's `cli_kind = "codex"` rows accumulate but don't get injected. Document the gap; surface as a Drop 5 readiness item if needed.
  - Acceptance: `codex_home/config.toml` renders with valid TOML; `codex_home/AGENTS.md` non-empty; bundle-render test passes.
  - Blocked_by: 4d.3.

### 3.3 Phase 3 — Wiring + tests + sample template (~1-2 droplets)

- **4d.8 — `cmd/till/main.go` blank import + multi-adapter contract test row.**
  - Files: `cmd/till/main.go`, `internal/app/dispatcher/mock_adapter_test.go` (extend `TestCLIAdapterContractTableDriven` table), possibly `internal/app/dispatcher/spawn_test.go`.
  - Adds `_ "github.com/evanmschultz/tillsyn/internal/app/dispatcher/cli_codex"` blank import next to existing `cli_claude` import at `cmd/till/main.go:31`.
  - Extends the contract-test table at `mock_adapter_test.go:568` with a `codexAdapter` row. The row's fixture lines come from `cli_codex/testdata/codex_stream_minimal.jsonl` (Q1 fixture relocated under cli_codex testdata).
  - Acceptance: `mage ci` green; contract test table includes 3 rows (Mock + claude + codex); registry lookup of `CLIKindCodex` returns the registered codex adapter.
  - Blocked_by: 4d.4, 4d.5, 4d.6, 4d.7.

- **4d.9 — Sample codex-flavored project template + dogfood-readiness handoff.**
  - Files: NEW `internal/templates/builtin/default-codex.toml` (or extend `default-go.toml` with explicit `cli_kind = "codex"` on a sample agent_binding for the dogfood hook), `workflow/drop_4d/DOGFOOD_HANDOFF.md`.
  - The sample template proves Drop 5 has at least one binding declaring `cli_kind = "codex"` for the multi-CLI dogfood. **Planner decides scope**: full new TOML vs minimal binding-row addition.
  - `DOGFOOD_HANDOFF.md` is the Drop 5 cold-start counterpart to `project_drop_4c_5_handoff.md`.
  - Acceptance: TOML loads cleanly through `templates.Load`; binding round-trips with `CLIKind = "codex"`; sample agent_binding validates against schema.
  - Blocked_by: 4d.8.

### 3.4 Optional ride-along droplet (Q2 routing)

- **4d.0 (or 4d.10) — Fix R-A.4-1 metadata-before-move ordering in dispatcher.**
  - Files: `internal/app/dispatcher/monitor.go` (`applyCrashTransition` at line 344-369), `internal/app/dispatcher/dispatcher.go` (`transitionToFailed` at line 631-639).
  - 4c.5 raised: both paths violate the new Theme A "non-empty `metadata.outcome` required on `failed` transitions" guard. Pre-cascade today the orchestrator drives transitions manually; once Drop 5 dogfood fires the dispatcher for real, this hits A.4's guard immediately.
  - Routing decision (Q2): land here as 4d.0 (pre-Phase-1) OR as a separate housekeeping drop. **Lean: land here as 4d.0** — small, dispatcher-internal, prevents Drop 5 fire-drill.
  - Acceptance: both transition paths set `metadata.outcome: "failure"` BEFORE move-to-failed; existing dispatcher tests still green; new test pins ordering against A.4 guard.
  - Blocked_by: none (independent of codex work).

## 4. Out of Scope

- F.4 marketplace CLI (Drop 4d-prime — separate post-Drop-5 drop, ~5 droplets).
- F.7 spawn pipeline (shipped Drop 4c).
- Drop 4c.5 themes (in flight, prerequisite — not re-litigated here).
- Codex pre-execution permission interception (`exec_approval_request` mid-stream flow). Cascade-spawned codex runs with `--full-auto`; pre-approval gating is a future-drop concern.
- Codex cost synthesis from `token_count` events. `TerminalReport.Cost` stays nil for codex spawns (the F.7.17 L11 `*float64` was designed for exactly this case).
- Tillsyn-MCP self-registration into codex's `~/.codex/config.toml` `[mcp_servers]`. Cascade-on-itself MCP loop for codex is a Drop 5 readiness concern; default Drop 4d codex spawns run without Tillsyn-MCP. Surface as Drop 5 prep refinement.
- Permission grants persistence for codex spawns. The `permission_grants.cli_kind` column already accepts `"codex"` rows but Drop 4d's render does not inject them into config.toml. Drop 5 dogfood may require this; route as a refinement if surfaced.
- Codex's `--output-schema` JSON-schema-output flag. Useful for structured-output cascade tasks but not Drop 4d scope.
- Drop 5 dogfood validation (codex spawn-on-itself end-to-end testing).
- Drop 4.5 TUI overhaul (concurrent FE/TUI track).

## 5. Locked Architectural Decisions Inherited from Drop 4c

These are non-negotiable carried forward; do NOT relitigate without dev signoff:

- **L1** Tillsyn never holds secrets. `Env []string` is name-only allow-list. Codex's API key (`OPENAI_API_KEY`) is a name in the list, never a value.
- **L2** No Docker awareness, no OAuth registry, no `command` override knob. Codex adapter calls `codex` directly. Adopters who want process isolation install OS-level wrappers.
- **L3** POSIX-only. `CODEX_HOME` works on Linux + macOS; Windows deferred per L3.
- **L4** Closed POSIX env baseline (18 vars). Codex env baseline is the SAME closed list — no codex-specific additions to the baseline. The per-spawn `CODEX_HOME=<bundle.Root>/codex_home/` is added by codex's `assembleEnv`, NOT by changing the baseline.
- **L11** Dispatcher monitor stays CLI-agnostic via the `StreamEvent.Type` + `IsTerminal` fields. Codex's adapter MAY introduce new canonical type strings (`"usage"`, `"error"`) but MUST NOT change interface signatures.
- **L13** `BundlePaths` carries cross-CLI paths only. Codex's `codex_home/` subdir is computed by codex adapter under `BundlePaths.Root` — NOT added to `BundlePaths`.
- **L20** Commit + push gates default OFF via `dispatcher_commit_enabled` / `dispatcher_push_enabled` toggles. Drop 4d does NOT change defaults.

Codex-specific locked decisions (NEW for Drop 4d):

- **L-codex-1** Per-spawn `CODEX_HOME=<bundle.Root>/codex_home/` is the bundle-isolation knob. Without it, codex shares dev's `~/.codex/` and isolation breaks. Adopter cannot disable this — it's required for cascade-on-itself.
- **L-codex-2** `TerminalReport.Cost = nil` for codex spawns. The `*float64` field's design intent. Drop 5's cost-tracking metadata.actual_cost_usd is nil for codex spawns; downstream tooling MUST respect nil.
- **L-codex-3** `TerminalReport.Denials = nil` for codex spawns. Codex's pre-approval flow is out-of-scope for Drop 4d.
- **L-codex-4** Plugin-preflight is skipped for codex via per-adapter routing (Option A simple skip OR Option B interface extension — planner picks). `tillsyn.requires_plugins` declared on a project does not error a codex spawn.
- **L-codex-5** Drop 4d codex spawns run without Tillsyn-MCP self-registration. Cascade-on-itself MCP loop for codex is Drop 5 readiness work.

## 6. Pre-MVP Rules In Force

Same as Drop 4c.5:

- No migration logic in Go. No CLI-side migration. Schema additions ship inline; dev fresh-DBs.
- No closeout MD rollups (LEDGER / WIKI_CHANGELOG / REFINEMENTS / HYLLA_FEEDBACK) — pre-dogfood.
- Opus builders. Every builder spawn carries `model: opus`.
- Filesystem-MD mode. No Tillsyn-runtime per-droplet plan items.
- Tillsyn-flow output style + Section 0 SEMI-FORMAL REASONING in every subagent response.
- Single-line conventional commits ≤72 chars.
- NEVER raw `go test` / `go build` / `go vet` / `mage install`. Always `mage <target>`. Verification: `mage check` + `mage ci`.
- Hylla is Go-only today AND stale post-Drop-4c-merge AND the active Drop 4c.5 may not yet have ingested. **Drop 4d builders use `Read` / `Grep` / `LSP` / `git diff` directly** — no Hylla calls.
- **Builder spawn prompts MUST include "do NOT commit" directive (per F.7-CORE REV-13).** Orchestrator drives commits AFTER QA pair returns green.
- **Each builder reads REVISIONS POST-AUTHORING section first** if the per-droplet sub-plan has one.
- **External research via Context7 + WebSearch + WebFetch** for codex CLI semantics. `~/.codex/config.toml` schema, `codex exec --json` event taxonomy, sandbox modes — verify against Context7 `/openai/codex` first; fall back to GitHub source code reads.
- Sandbox-test-hang gotcha (`mage testPkg internal/app/dispatcher` may hang in builder sessions on `monitor_test.go`'s `exec.Command("go", "build", ...)`). Orchestrator-shell `mage ci` is authoritative.

## 7. Wave Structure (Tentative)

Tighter than 4c.5 — single-package work for Phase 2, file-scope mostly disjoint.

- **Wave 1 (~2 droplets, parallel-friendly):** 4d.0 (R-A.4-1 ride-along, IF Q2 routes here) + 4d.1 (`CLIKindCodex` constant). Disjoint files; can run parallel.
- **Wave 2 (~1 droplet, gates Phase 2):** 4d.2 (plugin-preflight routing).
- **Wave 3 (~3-4 droplets, parallel-friendly):** 4d.3 (skeleton) → then 4d.4, 4d.5, 4d.7 in parallel (argv / env / render are file-disjoint within `cli_codex/`). 4d.6 (stream.go) parallel-eligible IF Q1 fixture in hand; otherwise gates on Q1.
- **Wave 4 (~1 droplet):** 4d.8 (cmd/till wiring + contract-test row). Gates on Wave 3 completion.
- **Wave 5 (~1 droplet):** 4d.9 (sample template + DOGFOOD_HANDOFF). Cleanup + dogfood-readiness handoff.

Total: **7-9 droplets**. Plan-QA may absorb / split / re-sequence.

## 8. Concrete Planner Spawn Contract

Single planner OR parallel theme planners — see Q6. Each planner:

- Reads this `REVISION_BRIEF` + `project_drop_4c_shipped.md` + `project_drop_4c_5_handoff.md` (if 4c.5 closed) + `CLI_ADAPTER_AUTHORING.md` + `SPAWN_PIPELINE.md`.
- Reads code surfaces directly (`internal/app/dispatcher/cli_adapter.go`, `cli_claude/*.go` as reference, `binding_resolved.go`, `spawn.go`, `mock_adapter_test.go`).
- External research: Context7 `/openai/codex` for codex CLI semantics; WebFetch `https://github.com/openai/codex/blob/main/codex-rs/exec/src/cli.rs` for argv flag canonicality; reads dev-supplied `testdata/codex_stream_minimal.jsonl` fixture for event taxonomy.
- Authors a per-droplet PLAN.md (or per-phase PLAN MDs) under `workflow/drop_4d/` with per-droplet acceptance criteria, test scenarios, falsification mitigations, verification gates (mage targets — never raw `go test`).
- NO Hylla calls (Hylla stale + Go-only). Section 0 SEMI-FORMAL REASONING required. Tillsyn-flow output style.

Each builder spawn prompt carries:

- "You are NOT permitted to run `git commit`." (F.7-CORE REV-13.)
- Hylla artifact ref `github.com/evanmschultz/tillsyn@main` for orientation, but no Hylla queries.
- The droplet's `paths []string` + `packages []string` from this brief.
- The acceptance criteria from §3.
- Verification gate: `mage check` + `mage test-pkg <pkg>` (NOT `mage ci` per-droplet — orchestrator runs `mage ci` between droplets).

## 9. Open Questions for Plan-QA Review

- **Q1 — Codex `exec --json` event taxonomy fixture (HARD PRECONDITION).**
  Context7 evidence covers codex's argv (`codex-rs/exec/src/cli.rs`) and app-server JSON-RPC flow but does NOT pin the exact `codex exec --json` per-event field shape. Possibilities for resolving:
  - **Path A (recommended)**: Dev runs `codex exec --json --skip-git-repo-check -C /tmp "say hi"` once on their machine, captures stdout, commits as `internal/app/dispatcher/cli_codex/testdata/codex_stream_minimal.jsonl` BEFORE Drop 4d builders fire.
  - **Path B**: Brief surfaces ambiguity to planner; planner runs the probe themselves (planner has `Bash` access; builders may not).
  - **Path C**: Drop 4d.6 (stream.go) lands as a stub with a TODO + JSONL-fixture-pending comment; Drop 5's first dogfood iteration replaces the stub.
  **Recommendation**: Path A. Cleanest. ~30 seconds of dev time to record. Drop 4d planner-spawn gates on the fixture.

- **Q2 — R-A.4-1 ordering bug — Drop 4d ride-along or separate housekeeping drop?**
  4c.5 raised: `monitor.go applyCrashTransition` and `dispatcher.go transitionToFailed` violate "metadata-before-move" ordering vs Theme A.4's new outcome guard. Drop 5 dogfood will hit this immediately.
  **Recommendation**: ride-along as 4d.0. Small (~50 LOC + tests), dispatcher-internal, no codex coupling. Avoids fire-drill in Drop 5. Re-route as separate drop only if planner finds it touches files codex work also touches (low risk — codex work is in `cli_codex/`; ordering fix is in `monitor.go` + `dispatcher.go`).

- **Q3 — F.4 marketplace CLI (Drop 4d-prime) — fold into Drop 4d or stay separate?**
  4d-prime is the marketplace registry surface (`till marketplace install <name>`). Pre-build memory deferred it post-Drop-5.
  **Recommendation**: stay separate post-Drop-5. Marketplace is orthogonal to adapter seam validation; folding it pulls Drop 4d to ~12+ droplets and conflates two unrelated concerns. Keep 4d-prime at ~5 droplets after Drop 5 dogfood validates the multi-CLI loop end-to-end.

- **Q4 — Refinements transcription timing (Drop 4c.5 raised 6: R-A.4-1, R-A.2-3, D.2-R1, D.2-R2, F.5.2 NIT, C.3 NIT).**
  Should the 4c.5 refinements transcribe to `project_drop_4c_5_refinements_raised.md` memory NOW (before Drop 4d planning) or fold into Drop 4d's prep work?
  **Recommendation**: transcribe NOW, in parallel with Drop 4d planner-spawn. The memory file is the durable record; Drop 4d's brief already covers R-A.4-1 in Q2. Other refinements don't gate Drop 4d. Dev runs the transcription as a 5-minute side-task before / during Drop 4d planning.

- **Q5 — Drop 5 readiness gate.**
  What does multi-CLI dogfood require that wasn't already gated by 4c.5 Themes A+B?
  - 4c.5 Themes A+B (silent-data-loss + escape hatches): ALREADY SUFFICIENT for single-CLI dogfood per `project_drop_4c_5_handoff.md` Q5.
  - For multi-CLI (codex+claude): ALSO NEEDS Drop 4d (this drop) shipped. Optional: codex Tillsyn-MCP self-registration (L-codex-5 deferred — Drop 5 may surface need).
  **Recommendation**: Drop 5 readiness = 4c.5 Themes A+B merged + Drop 4d merged. Codex MCP self-registration stays deferred unless Drop 5's first iteration explicitly fails without it (route as in-flight refinement to Drop 4d-bis or Drop 5 itself).

- **Q6 — Drop 4d planner-spawn shape (single planner vs parallel theme planners).**
  Drop 4c.5 used 4 parallel theme planners. Drop 4d is 7-9 droplets, single package surface.
  - **Single planner**: simpler, no synthesis step, full coherence. ~30-45 min planning session.
  - **Parallel theme planners**: faster wall-clock if 2-3 planners run concurrently; orchestrator synthesis step adds ~10-15 min.
  **Recommendation**: SINGLE PLANNER. Drop 4d's surface is too small to amortize parallel-planner overhead. Two reasonable planner sub-scopes (codex package implementation, dispatcher-side wiring) overlap heavily on `cli_adapter.go` and `mock_adapter_test.go` — synthesis pain outweighs parallelism win.

## 10. Approximate Size

- **Droplet count**: 7-9 (Phase 1: 2; Phase 2: 4-5; Phase 3: 1-2; optional ride-along 4d.0).
- **Estimated session time**: 1-2 days at Drop 4a/4b/4c's pace. Smaller than Drop 4c.5 (~27-34 droplets); single-package work means less file-overlap coordination; codex package is ~5-6 new files mirroring cli_claude shape.
- **Risk-adjusted**: 2-3 days if Q1 fixture runs late or codex `exec --json` event shape requires interface adjustments (Phase 2 redo).

## 11. References

- `internal/app/dispatcher/cli_adapter.go` — `CLIAdapter` interface, `CLIKind` enum, `BindingResolved`, `BundlePaths`, `StreamEvent`, `TerminalReport`, `ToolDenial`. Drop 4d adds `CLIKindCodex` constant.
- `internal/app/dispatcher/cli_claude/` — reference implementation. Drop 4d mirrors layout in `cli_codex/`.
- `internal/app/dispatcher/cli_claude/render/` — claude-side bundle render (settings.json + plugin tree). Drop 4d's analog is `cli_codex/render/` writing `codex_home/config.toml` + `AGENTS.md`.
- `internal/app/dispatcher/mock_adapter_test.go` — `TestCLIAdapterContractTableDriven` at line 555. Drop 4d extends the table with codex row.
- `internal/app/dispatcher/spawn.go` — `BuildSpawnCommand` + adapter registry seam. Lines 240+ document the codex registration pattern.
- `internal/app/dispatcher/binding_resolved.go:127-130` — L15 default-to-claude resolver semantic. Codex adapter does NOT disturb this; codex spawns require explicit `cli_kind = "codex"` on the binding.
- `internal/app/dispatcher/plugin_preflight.go` — claude-specific `claude plugin list --json` invocation. Drop 4d.2 routes codex around it.
- `internal/templates/schema.go:475-488` — `AgentBinding.CLIKind` is a free string at template-load time; validated at adapter-lookup. Drop 4d does NOT change schema.
- `cmd/till/main.go:31` — single blank-import line. Drop 4d.8 adds parallel codex import.
- `internal/adapters/storage/sqlite/permission_grants_repo_test.go:57` — `cli_kind` column already accepts `"codex"` rows. Drop 4d does NOT change schema.
- `CLI_ADAPTER_AUTHORING.md` (top-level) — canonical guide for adding a new adapter; Drop 4d follows it verbatim.
- `SPAWN_PIPELINE.md` (top-level) — pipeline architecture overview.
- `~/.claude/projects/-Users-evanschultz-Documents-Code-hylla-tillsyn/memory/project_drop_4c_shipped.md` — what Drop 4c shipped + load-bearing dev decisions.
- `~/.claude/projects/-Users-evanschultz-Documents-Code-hylla-tillsyn/memory/project_drop_4c_5_handoff.md` — Drop 4c.5 sibling-drop chart (NEEDS UPDATE post-swap; see §12).
- Context7 `/openai/codex` `codex-rs/exec/src/cli.rs` — codex `exec` mode CLI flag canonical reference.
- Context7 `/openai/codex` `codex-rs/app-server/README.md` — codex JSON-RPC event vocabulary (related but distinct from `exec --json` line shape).

## 12. Swap Mechanics Checklist

The original sequencing was `4c.5 → 5 → 4d → 4d-prime → 5.5/6`. Dev decided `4c.5 → 4d → 5 → 4d-prime` with Drop 5 absorbing the multi-CLI dogfood that was 5.5/6. Doc-level changes:

- [ ] **`PLAN.md`** (project root) — reorder the row sequence: 4c.5 → 4d → 5 (multi-CLI dogfood) → 4d-prime. Update Drop 5's scope text to reflect "multi-CLI dogfood from day one" instead of "claude-only dogfood." Remove or re-purpose Drops 5.5 / 6 (absorbed into Drop 5).
- [ ] **`project_drop_4c_5_handoff.md`** memory — update §"Sibling Drop Sequencing" chart to show new order (lines 88-103 of the memory file).
- [ ] **`project_drop_4c_shipped.md`** memory — NO CHANGES. The shipped state of Drop 4c is unaffected by downstream drop reordering.
- [ ] **NEW `project_drop_4d_handoff.md`** memory — cold-start parity with `project_drop_4c_5_handoff.md`. Authored AFTER Drop 4d brief is approved by dev (this brief is the predecessor — handoff memory is post-approval). Carries the full §1-§13 of this brief in compressed form plus pre-flight checklist for the orchestrator who picks up Drop 4d cold.
- [ ] **`workflow/drop_4d/REVISION_BRIEF.md`** — THIS FILE.
- [ ] **`workflow/drop_4d/DOGFOOD_HANDOFF.md`** — Drop 5 cold-start handoff, authored as droplet 4d.9 deliverable.

## 13. Risks of the Swap vs Original Sequencing

- **R1 — Codex event taxonomy diverges in ways that force `CLIAdapter` interface changes.** If codex `exec --json` emits SSE-like fragments (delta events without per-event JSON envelope) instead of strict JSONL, the existing `ParseStreamEvent(line []byte) (StreamEvent, error)` interface breaks. **Mitigation**: Q1 fixture probe BEFORE Drop 4d planner-spawn. If the fixture confirms strict JSONL (likely per `--json` flag's `Print events to stdout as JSONL` doc-comment in codex-rs/exec/src/cli.rs), no interface change needed. If the fixture shows a mismatch, the interface rewrite per `CLI_ADAPTER_AUTHORING.md` "Non-JSONL Extensibility" lands in Drop 4d FIRST (~2-3 droplet upfront cost), then the adapter on top.
- **R2 — External research can't resolve Q1, dev can't capture fixture in time.** Drop 4d planner-spawn would either fabricate event-type strings (planner-fabrication risk per "Drop 0 Orchestrator Owns Description Accuracy" memory) OR stall. **Mitigation**: brief explicitly routes Q1 as HARD PRECONDITION; dev gates planner-spawn on fixture availability. Worst-case: Drop 4d planner spawns with Path C (stub stream.go), Drop 5 first iteration replaces stub.
- **R3 — Drop 5 dogfood scope explosion.** Multi-CLI dogfood from day one means Drop 5 must validate BOTH claude AND codex spawn-on-itself. Doubles probe surface; doubles failure modes; doubles fix-builder cost. **Mitigation**: Drop 5 sequences validation as Wave 1 (claude-only, retains current Drop-5 scope) → Wave 2 (codex-only, adds the multi-CLI angle). Failures in Wave 2 don't roll back Wave 1 progress. Brief recommends this split in §9 Q5.
- **R4 — Codex's permission / approval / sandbox model breaks Tillsyn's permission_grants abstraction.** Tillsyn's TUI handshake assumes terminal-event-bundled `permission_denials[]`; codex doesn't emit those. **Mitigation**: L-codex-3 explicitly punts. permission_grants table accumulates codex rows but adapter doesn't inject them. Drop 5 may surface need; route as in-flight refinement.
- **R5 — `CODEX_HOME` per-spawn override has unexpected side effects.** Setting `CODEX_HOME` to a per-spawn temp directory means codex won't read dev's `~/.codex/auth.json` or `~/.codex/config.toml` profiles. Authentication may break. **Mitigation**: render `auth.json` into `<bundle.Root>/codex_home/` from dev's `~/.codex/auth.json` at spawn time (acceptable per L1 — auth.json contents are NOT secret env vars; they're user-managed credentials living in a user-controlled directory). Alternative: forward `OPENAI_API_KEY` env var via `binding.Env` allow-list and let codex pick that up. Surface as planner subdecision in 4d.5.
- **R6 — Drop 4d landing means TWO dispatcher commits going green simultaneously is harder to test.** Process monitor + spawn registry + lock manager all need to handle codex spawns AND claude spawns concurrently. Orphan scan's cmdline-match (F.7.8) needs to recognize both binary names. **Mitigation**: orphan scan already reads `cli_kind` from manifest (per `project_drop_4c_shipped.md` §"Production Surfaces Shipped" and `monitor.go` references); test coverage for codex orphan recovery should be planner-tracked as part of 4d.6 acceptance.

## 14. Recommendations

Lean answers for Q1-Q6 with rationale (so dev can confirm/override in one read):

- **Q1**: Path A — dev captures `codex_stream_minimal.jsonl` fixture before planner-spawn. ~30 seconds of dev time prevents fabrication risk in 4d.6. Hard precondition.
- **Q2**: Land R-A.4-1 ordering fix as 4d.0 ride-along. Small, dispatcher-internal, prevents Drop 5 fire-drill.
- **Q3**: Keep 4d-prime separate post-Drop-5. Marketplace work is orthogonal to adapter-seam validation. Folding pulls Drop 4d to ~12+ droplets and conflates concerns.
- **Q4**: Transcribe 4c.5 refinements to memory NOW (5-min dev side-task, parallel with Drop 4d planning). R-A.4-1 already covered in Drop 4d Q2; remaining refinements don't gate Drop 4d.
- **Q5**: Drop 5 readiness = 4c.5 A+B merged + Drop 4d merged. Codex Tillsyn-MCP self-registration deferred unless Drop 5 first-iteration explicitly fails without it.
- **Q6**: Single planner. Drop 4d's 7-9 droplets in a single package surface don't justify parallel-planner orchestration overhead.

If dev confirms the lean, planner-spawn proceeds. If dev overrides any of Q1-Q6, brief is revised before planner-spawn.
