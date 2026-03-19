# Collaborative MCP STDIO + Autent Execution Plan

Created: 2026-03-17  
Owner: orchestrator (Codex)  
Status: updated with runtime/transport consensus; follow-up implementation pending  
Primary parent log: `PLAN.md`  
Primary validation baseline: `COLLABORATIVE_POST_FIX_VALIDATION_WORKSHEET.md`  
Primary collaborative task worksheet: `COLLAB_VECTOR_MCP_E2E_WORKSHEET.md`

## 1) Objective

Ship the next dogfood-ready `tillsyn` foundation wave by:

1. making local MCP invocation work reliably in this environment again,
2. promoting STDIO MCP to the default/primary transport surface,
3. replacing the current ad hoc mutation-auth foundation with an `autent`-aligned authenticated-caller seam that `tillsyn` can build on next,
4. preserving `tillsyn` as a self-contained product that does not depend on `blick`,
5. fixing remaining known identity/attribution/runtime issues that block collaborative dogfooding,
6. producing an updated collaborative worksheet for final user rerun.

## 2) Locked Product Direction

1. `blick` is out of scope for implementation in this wave.
2. `tillsyn` must remain usable without `blick`.
3. `autent` is the target reusable auth/session/grant foundation beneath `tillsyn`.
4. STDIO is the default/primary MCP transport for local use.
5. HTTP serve remains optional/secondary for explicit long-running server scenarios.
6. `tillsyn` should expose MCP without requiring `./till serve`.
7. Current user-visible attribution must prefer readable names over raw ids or UUID-like values.
8. This wave should not add backward-compatibility shims or preserve stale `Kan` naming; rename in place to `tillsyn`/`till` where appropriate.

## 3) Known Problems To Close

1. MCP lease issuance originally failed in this environment with `sqlite3: attempt to write a readonly database`.
2. Current stdio MCP runtime contract is surprising for dogfooding because it defaults to an isolated repo-local DB instead of the same runtime as `./till`.
3. Local builds still default to `dev_mode=true`, so `./till`, `./till mcp`, and `./till serve` silently open dev paths unless the user passes `--dev=false`.
4. Mutation auth/identity handling is still based on local lease tuple conventions and fallback labels such as `tillsyn-user`.
5. Actor attribution is improved but still inconsistent across all read/write paths.
6. Stale fallback labels and old-name copy still exist across domain/storage/runtime defaults.
7. `Ctrl-C` on `./till mcp` currently bubbles out as `context canceled` error logging instead of clean shutdown.
8. Collaborative docs still assume the older HTTP-first MCP shape and do not yet reflect the updated dogfood-runtime consensus.

## 4) Documentation + Research Basis

### 4.1 Context7 consulted before edits

1. `/mark3labs/mcp-go`
   - basis used:
     - `server.ServeStdio(...)` for STDIO server mode
     - `server.NewStreamableHTTPServer(...)` for HTTP mode

### 4.2 Fallback source recorded

Context7 does not provide `autent` project-specific docs. Fallback source for this wave:

1. local clone README:
   - `.artifacts/external/autent/README.md`

This fallback is acceptable for this wave because `autent` is a local first-party codebase under active inspection.

## 5) Target End State

### 5.1 Runtime + transport

1. `till mcp` remains the raw STDIO MCP server entrypoint.
2. STDIO MCP defaults to the same runtime/config contract as `./till` for dogfooding.
3. Dev isolation becomes explicit (`--dev` and/or a future dedicated isolation flag), not the silent default.
4. `serve` remains available for HTTP API + streamable HTTP MCP, but is secondary and optional.
5. MCP tests and dogfood docs default to STDIO first against the real runtime unless the user explicitly opts into dev isolation.
6. `Ctrl-C` on `till mcp` is treated as clean shutdown, not failure.
7. A future `till mcp-inspect` command is documented as a visible developer MCP inspector/debug client, separate from the raw stdio server.

### 5.2 Auth foundation

1. This wave lands the internal authenticated-caller seam and readable actor persistence needed for later `autent` integration.
2. `tillsyn` still owns its user-facing task/comment/agent workflows.
3. Actor identity is derived from authenticated/session context where possible, not caller-supplied display fields alone.
4. Multi-user and agent attribution remain visible and human-readable.

### 5.3 Identity + UX correctness

1. Local user display surfaces show `Evan`, not raw ids or `tillsyn-user`.
2. Agent/orchestrator/subagent actions persist and render readable actor names and actor types.
3. Legacy raw-id fallback display is removed from normal task/info/thread/activity surfaces.

## 6) Lane Plan

## 6.1 Lane A: MCP Transport + Runtime

Objective:
- make MCP work locally again and promote STDIO as the primary transport.

Lock scope:
- `cmd/till/**`
- `internal/adapters/server/**`
- related tests in those areas

Out of scope:
- `internal/tui/**`
- `internal/domain/**`
- `internal/app/**` except transport-facing wiring if strictly required
- `PLAN.md`

Acceptance:
1. a STDIO MCP command exists and is tested,
2. existing HTTP serve transport still works,
3. STDIO MCP shares the main runtime/config contract by default instead of silently diverging to `.tillsyn/mcp/...`,
4. dev isolation remains available only by explicit opt-in,
5. MCP transport docs/help output reflect the new default and the `serve`/`mcp` transport split.

Expected checks:
1. `just test-pkg ./cmd/till`
2. `just test-pkg ./internal/adapters/server`
3. `just test-pkg ./internal/adapters/server/mcpapi`

## 6.2 Lane B: Autent Foundation + Identity/Auth Refactor

Objective:
- replace the current ad hoc mutation-auth identity/lease model with an `autent`-aligned authenticated-caller seam while preserving `tillsyn` semantics.

Lock scope:
- `internal/domain/**`
- `internal/app/**`
- `internal/adapters/storage/sqlite/**`
- `internal/adapters/server/common/**`
- related tests in those areas

Out of scope:
- `internal/tui/**`
- `cmd/till/**` except interface/wiring follow-up requested by integrator
- `PLAN.md`

Acceptance:
1. current lease/auth behavior is reimplemented on top of the new auth foundation or cleanly superseded,
2. actor id/name/type are persisted correctly for user, orchestrator, and subagent flows,
3. stale fallback identity usage is removed from normal product behavior rather than preserved behind compatibility shims,
4. tests cover session/identity/grant/actor attribution behavior.

Expected checks:
1. `just test-pkg ./internal/domain`
2. `just test-pkg ./internal/app`
3. `just test-pkg ./internal/adapters/storage/sqlite`
4. `just test-pkg ./internal/adapters/server/common`

## 6.3 Lane C: TUI + Attribution Surface Closeout

Objective:
- align all remaining visible attribution/readability surfaces and update dogfood/collab docs.

Lock scope:
- `internal/tui/**`
- collaborative markdown under repo root except `PLAN.md`
- test fixtures/goldens under `internal/tui/testdata/**`

Out of scope:
- `cmd/till/**`
- `internal/app/**`
- `internal/adapters/storage/sqlite/**`
- `PLAN.md`

Acceptance:
1. task info/system/activity/thread/comment surfaces show readable actor names,
2. any new STDIO-first MCP/user guidance is reflected in the worksheets,
3. a new collaborative rerun worksheet exists for the post-fix user pass.

Expected checks:
1. `just test-pkg ./internal/tui`
2. `just test-golden`

## 7) Shared Integration Rules

1. The orchestrator is the only writer to `PLAN.md`.
2. Worker lanes must not edit outside their lock scope.
3. Worker lanes must use Context7 before code edits and again after any failed test/runtime error.
4. If Context7 does not cover `autent`, the worker must record fallback use of `.artifacts/external/autent/README.md`.
5. Worker lanes run package-scoped `just test-pkg` checks only.
6. Integrator runs:
   - `just fmt`
   - `just test-golden`
   - `just check`
   - `just ci`
7. Two independent QA passes are required before handoff.

## 8) QA Reference Checklist

QA must confirm:

1. STDIO MCP is the default documented invocation path.
2. `serve` remains optional and does not regress.
3. MCP mutation path works from the same default runtime as the TUI unless the user explicitly opts into isolation/dev mode.
4. Actor attribution persists and renders correctly for:
   - local user
   - orchestrator agent
   - subagent/worker agent
   - system actor where applicable
5. No raw UUID-like ids leak into normal task/thread/info surfaces when a readable name exists.
6. Updated collaborative worksheet matches implemented behavior and transport flow.
7. Raw `till mcp` remains protocol-clean, while any future human debugging surface is separate from the server command.

## 9) Updated Collaborative Validation Requirements

The post-fix worksheet must cover:

1. STDIO MCP startup and tool discovery without `./till serve`
2. MCP mutation flow under the new auth foundation
3. user attribution display
4. agent attribution display
5. thread/task/activity/system attribution consistency
6. HTTP serve sanity only as a secondary regression check
7. default runtime behavior no longer silently forces dev mode for normal dogfooding

## 10) Execution Log

### Checkpoint 000

1. User confirmed:
   - ignore `blick`,
   - make STDIO MCP the default,
   - move `autent` earlier as the auth foundation,
   - keep `tillsyn` self-contained,
   - fix known identity/runtime issues,
   - create a new collaborative rerun worksheet after implementation.
2. Context7 consulted for `mcp-go` STDIO/HTTP transport surface.
3. `autent` fallback source recorded from the local clone README.

Next step:
1. log this wave in `PLAN.md`,
2. spawn worker lanes with the lock scopes above.

### Checkpoint 001

1. Inspected current shared branch state and found the wave was already partially implemented in code:
   - `till mcp` command wiring already existed,
   - stdio server adapter wiring already existed,
   - authenticated-caller seam and task actor-name fields already existed,
   - TUI already had readable actor-label helpers.
2. Ran focused package gates:
   - `just test-pkg ./cmd/till` -> FAIL (`scanTask` missing `CreatedByName` / `UpdatedByName` destinations during export/import round-trip)
   - `just test-pkg ./internal/tui` -> FAIL (three stale tests: add-task save trigger, thread fallback entry path, schema coverage map)
   - `just test-pkg ./internal/app` -> PASS
   - `just test-pkg ./internal/adapters/server/mcpapi` -> PASS
3. Context7 was re-consulted after the failing test loop:
   - Bubble Tea key handling reference for focused component input behavior,
   - SQLite prepared insert/value-count reference for SQL placeholder alignment.
4. Integrated the first remediation slice:
   - fixed SQLite `scanTask` to read actor-name columns,
   - updated stale TUI tests to match the current save/thread/schema contract.

### Checkpoint 002

1. Worker-lane findings were reviewed before final integration:
   - Lane A: stdio MCP transport/runtime was present but under-tested in `cmd/till`.
   - Lane B: task row writes still did not persist `created_by_name` / `updated_by_name`; change-event fallbacks still ignored task-row readable names.
   - Lane C: the remaining TUI failures were stale tests, not live product regressions.
2. Integrated the second remediation slice:
   - `work_items` create/update writes now persist actor-name columns,
   - task change-event attribution now prefers readable task-row names when context does not provide one,
   - storage tests now assert persisted human-readable task row names,
   - MCP adapter attribution test now checks `UpdatedByName`,
   - `cmd/till` tests now cover stdio MCP help, repo-local runtime fallback, and the `mcp` command path.
3. Revalidation sequence:
   - `just fmt` -> PASS
   - `just test-pkg ./cmd/till` -> PASS
   - `just test-pkg ./internal/tui` -> PASS
   - `just test-pkg ./internal/app` -> PASS
   - `just test-pkg ./internal/adapters/storage/sqlite` -> PASS
   - `just test-pkg ./internal/adapters/server/common` -> PASS (`[no test files]`)
   - `just test-golden` -> PASS
   - `just check` -> PASS
   - `just ci` -> FAIL once on `cmd/till` coverage (69.8%)
4. After the `just ci` failure, Context7 was re-consulted for Cobra test patterns before the next edit.
5. Added focused `cmd/till` tests for stdio MCP and reran:
   - `just fmt` -> PASS
   - `just test-pkg ./cmd/till` -> FAIL once because the test assumed config seeding for `mcp`
   - Context7 re-consulted again (Cobra contract + startup config flow inspection)
   - adjusted the test to assert the real contract: repo-local runtime directory + DB, but no auto-seeded config for non-TUI commands
   - `just fmt` -> PASS
   - `just test-pkg ./cmd/till` -> PASS
   - `just ci` -> PASS (`cmd/till` coverage 74.8%)
   - final `just check` -> PASS
6. Final QA follow-up found one remaining mixed-override runtime gap and one documentation clarity gap:
   - `till mcp --config ...` still reused the platform DB path,
   - docs overstated this wave as full `autent` replacement instead of the authenticated-caller seam now in code.
7. Integrated the final follow-up:
   - stdio runtime fallback is now per-path, not all-or-nothing,
   - mixed override coverage was added in `cmd/till/main_test.go`,
   - the validation worksheet was regenerated with concrete section ids and exact actions,
   - execution docs now describe the wave as `autent`-aligned groundwork rather than full integration.
8. Final rerun after the QA follow-up:
   - `just fmt` -> PASS
   - `just test-pkg ./cmd/till` -> PASS
   - `just check` -> PASS
   - `just ci` -> PASS (`cmd/till` coverage 75.2%)

### Checkpoint 003

Integrated outcomes:
1. `till mcp` is wired and test-covered as the stdio-first local MCP entrypoint.
2. previous stdio MCP runtime-path isolation under `.tillsyn/mcp/<app>/...` was validated as intentional current behavior, but is now superseded by the new dogfood-runtime consensus to share the main runtime by default.
3. authenticated-caller groundwork is in place for future `autent` session integration without requiring `blick`.
4. task rows persist `created_by_name` and `updated_by_name` alongside ids/types.
5. task/thread/system/activity rendering paths have readable-name coverage instead of raw-id leakage when names exist.
6. the collaborative worksheet is now actionable instead of placeholder-only.

Commands and outcomes:
1. `just fmt` -> PASS
2. `just test-pkg ./cmd/till` -> PASS
3. `just test-pkg ./internal/adapters/server/mcpapi` -> PASS
4. `just test-pkg ./internal/app` -> PASS
5. `just test-pkg ./internal/tui` -> PASS
6. `just test-pkg ./internal/adapters/storage/sqlite` -> PASS
7. `just test-golden` -> PASS
8. `just check` -> PASS
9. `just ci` -> PASS

QA state:
1. Prior QA on the transport groundwork passed.
2. A new implementation/QA follow-up is required for the updated default-runtime consensus and clean stdio shutdown contract.

Remaining step:
1. implement the updated dogfood-runtime contract,
2. rerun transport/runtime QA,
3. then continue the worksheet section-by-section with the user from the updated `S2` contract onward.

## 11) Lane Status

| Lane | Owner | Scope | Outcome | Notes |
|---|---|---|---|---|
| A | Hubble | `cmd/till/**`, `internal/adapters/server/**` | Integrated | Correctly identified stdio MCP as under-tested rather than under-implemented. |
| B | Cicero | `internal/domain/**`, `internal/app/**`, `internal/adapters/storage/sqlite/**`, `internal/adapters/server/common/**` | Integrated | Identified the remaining task-row actor-name write gap and change-event fallback gap. |
| C | Euclid | `internal/tui/**`, collab markdown except `PLAN.md` | Integrated | Confirmed the remaining TUI failures were stale tests tied to the updated form/thread/schema contracts. |

## 12) Final Validation Snapshot

1. `just test-pkg ./cmd/till` -> PASS
2. `just test-pkg ./internal/adapters/server/mcpapi` -> PASS
3. `just test-pkg ./internal/app` -> PASS
4. `just test-pkg ./internal/adapters/storage/sqlite` -> PASS
5. `just test-pkg ./internal/adapters/server/common` -> PASS (`[no test files]`)
6. `just test-pkg ./internal/tui` -> PASS
7. `just test-golden` -> PASS
8. `just check` -> PASS
9. `just ci` -> PASS

## 13) Ready-For-User Scope

The collaborative rerun now needs to verify:

1. stdio MCP works without `./till serve`,
2. stdio MCP and `./till` share the same default runtime when no overrides are provided,
3. readable actor names appear on task system/thread/activity surfaces,
4. agent/user attribution persists sanely through task mutations,
5. `Ctrl-C` stops raw stdio MCP cleanly without error-like shutdown noise.
