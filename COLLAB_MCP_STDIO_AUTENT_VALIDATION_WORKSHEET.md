# Collaborative MCP STDIO + Autent Validation Worksheet

Created: 2026-03-17  
Status: ready for collaborative rerun  
Owner: User + Codex (single-writer updates by Codex)

## 1) Purpose

Validate the stdio-first MCP + authenticated-caller foundation wave together, with:

1. user-driven stdio MCP verification,
2. agent-driven attribution/auth verification,
3. shared pass/fail decisions with evidence,
4. strict fail-stop remediation before any next section.

## 2) Locked Collaboration Protocol

For every failing step, use this exact sequence before any next step:

1. Stop progression immediately on first fail.
2. Record the user's exact intent and observed behavior in this worksheet.
3. Spawn subagents to inspect code and local runtime context.
4. Re-consult Context7 (and fallback docs if needed) before edits.
5. Discuss options with user and reach explicit consensus.
6. Implement the smallest fix scope that closes the failing section.
7. Run required package tests and full gates.
8. Run two independent QA passes.
9. Re-run the same collaborative step and record fresh evidence.
10. Only then proceed to the next section.

## 3) Preconditions

1. Current branch validation is green:
   - `just test-pkg ./cmd/till`
   - `just test-pkg ./internal/adapters/server/mcpapi`
   - `just test-pkg ./internal/app`
   - `just test-pkg ./internal/adapters/storage/sqlite`
   - `just test-pkg ./internal/tui`
   - `just test-golden`
   - `just check`
   - `just ci`
2. Local MCP should be started with `./till mcp`, not `./till serve`.
3. Consensus target for the next implementation wave: default `./till` and default `./till mcp` should both dogfood the real runtime, not silently force dev/isolation mode.
4. Consensus target for the next implementation wave: remove stale `Kan` branding in current product/runtime surfaces rather than preserving old names for compatibility.

## 4) Evidence Destinations

Primary evidence files:

1. `COLLAB_MCP_STDIO_AUTENT_VALIDATION_WORKSHEET.md` (this file)
2. `COLLAB_MCP_STDIO_AUTENT_EXECUTION_PLAN.md`
3. `PLAN.md`

Secondary corroboration files:

1. `COLLABORATIVE_POST_FIX_VALIDATION_WORKSHEET.md`
2. `MCP_FULL_TESTER_AGENT_RUNBOOK.md`

## 5) Collaborative Queue (Run In Order)

### Section S1: STDIO MCP Startup

| ID | Action | Expected | Status | User Detailed Findings | Evidence | Notes |
|---|---|---|---|---|---|---|
| S1-01 | Start stdio MCP with `./till mcp` from repo root | Process starts without requiring `./till serve` | PASS | User started `./till mcp`; process booted cleanly, opened the repo-local stdio DB, and logged `command=mcp transport=stdio`. | User shell transcript 2026-03-17; local `./till mcp` startup logs | Stdio server is a subprocess transport, not HTTP serve mode. |
| S1-02 | Inspect the client-visible tool list from the MCP host | Capture-state + attention tools are present | PASS | `/mcp` in local Codex showed `tillsyn` as enabled with command `/Users/evanschultz/Documents/Code/hylla/tillsyn/till mcp` and the expected `till_*` tool list. | User Codex `/mcp` output 2026-03-17 | Confirms stdio MCP registration is healthy after removing the conflicting repo-local Codex HTTP config. |
| S1-03 | Stop stdio MCP and restart it once | Restart is clean and deterministic | PENDING_USER | | | |

### Section S2: Local Runtime Path Contract

| ID | Action | Expected | Status | User Detailed Findings | Evidence | Notes |
|---|---|---|---|---|---|---|
| S2-01 | Run stdio MCP with no `--db` or `--config` overrides | Repo-local MCP runtime is used automatically | PASS | User expected the existing `User Project` to appear, but `till.list_projects` returned empty. Investigation confirmed stdio MCP intentionally used the isolated repo-local runtime under `.tillsyn/mcp/tillsyn-dev/...`, not the app-support dev DB used by the TUI. | `./till mcp` logs showed config/db under `.tillsyn/mcp/tillsyn-dev/`; `till.list_projects` returned `[]`; `./till paths` showed the normal TUI dev DB under `~/Library/Application Support/tillsyn-dev/...` | Working as currently designed, but this contract is surprising and needs clearer docs/help. |
| S2-02 | Run stdio MCP with only `--config <path>` overridden | Config override is honored and the DB still stays on the local stdio runtime path | PENDING_USER | | | This closes the mixed-override regression fixed in this wave. |
| S2-03 | Confirm stdio MCP mutations no longer depend on the app-support DB path | No readonly-db failure from the old app-support path | PENDING_USER | | | |

### Section S3: User Attribution

| ID | Action | Expected | Status | User Detailed Findings | Evidence | Notes |
|---|---|---|---|---|---|---|
| S3-01 | Create or update a task/comment as the local user | New activity/thread/task ownership uses `Evan` | PENDING_USER | | | |
| S3-02 | Open task info `system:` section | `created_by` / `updated_by` show readable names, not UUID-like ids | PENDING_USER | | | |
| S3-03 | Open thread/comments for the same item | Local-user entries show `Evan`, not `tillsyn-user` | PENDING_USER | | | |

### Section S4: Agent Attribution

| ID | Action | Expected | Status | User Detailed Findings | Evidence | Notes |
|---|---|---|---|---|---|---|
| S4-01 | Let the agent create or mutate one task/comment through MCP | Ownership stores and displays a readable agent name | PENDING_AGENT | | | |
| S4-02 | Inspect task info/activity/thread for that item | Agent entry shows readable actor name + type, not a UUID/instance token | PENDING_USER | | | |

### Section S5: Surface Consistency

| ID | Action | Expected | Status | User Detailed Findings | Evidence | Notes |
|---|---|---|---|---|---|---|
| S5-01 | Compare project notifications, task activity, task `system:`, and thread comments for the same actor | Same readable name appears everywhere a readable name exists | PENDING_USER | | | |
| S5-02 | Compare system actor output if one system-originated event is available | System actor is rendered with a readable system label, not a fallback raw id | PENDING_AGENT | | | |

### Section S6: HTTP Regression Sanity

| ID | Action | Expected | Status | User Detailed Findings | Evidence | Notes |
|---|---|---|---|---|---|---|
| S6-01 | Run `./till serve --help` | HTTP serve remains available as a secondary path | PENDING_AGENT | | | No HTTP mutation testing needed in this phase. |

## 6) Findings + Remediation Ledger

| Finding ID | Section/Step | Severity | User Detailed Findings | Agent Rephrase | Decision | Status | Evidence |
|---|---|---|---|---|---|---|---|
| FR-MCP-001 | S1-02 | medium | Codex still threw `url is not supported for stdio in mcp_servers.tillsyn` even after the home config was changed to stdio. | Trusted-project Codex config layering loaded the repo-local `.codex/config.toml`, which still defined `mcp_servers.tillsyn` with the old HTTP `url`, creating a mixed stdio+url conflict for the same server id. | Fix repo-local Codex config to remove the stale HTTP `tillsyn` entry or align it to stdio. | CLOSED | Local inspection of `~/.codex/config.toml`; local inspection of [`.codex/config.toml`](/Users/evanschultz/Documents/Code/hylla/tillsyn/.codex/config.toml); successful `codex mcp list`; successful `/mcp` tool listing in local Codex. |
| FR-MCP-002 | S2-01 | medium | The user expected the existing `User Project` to appear from stdio MCP, but `till.list_projects` returned no projects. | Stdio MCP currently defaults to an isolated repo-local DB under `.tillsyn/mcp/...`, while the TUI and `./till paths` default to the app-support dev DB under `~/Library/Application Support/tillsyn-dev/...`; this mismatch is intentional today but not obvious. | Consensus: change the default contract so `./till` and `./till mcp` share the same real runtime by default, with dev/isolation moved behind explicit opt-in. | OPEN | `./till mcp` startup logs; `till.list_projects -> []`; `./till paths`; local shell discussion 2026-03-17 and 2026-03-19. |
| FR-MCP-003 | S2-01 | low | Bootstrap output still says `Kan` instead of `tillsyn`. | Bootstrap guide product copy is stale and no longer matches the product name. | Consensus: remove stale `Kan` branding from current product/runtime surfaces in place; do not keep old-name compatibility shims. | OPEN | `till.get_bootstrap_guide` output captured 2026-03-17. |
| FR-MCP-004 | S2 discussion | high | Running `./till` without `--dev` still opens the dev-mode app paths, which is not desired for dogfooding. | Root CLI default currently derives `devMode` from `version == "dev"`. Locally built binaries keep `version = "dev"`, so `./till` defaults to dev mode unless the user passes `--dev=false` or sets `TILL_DEV_MODE`. | Consensus: stop defaulting local dogfood binaries to dev mode; `--dev` remains explicit opt-in. Apply the same default-runtime change consistently to `./till`, `./till mcp`, and `./till serve`. | OPEN | `./till paths`; `./till --dev=false paths`; local code inspection in `cmd/till/main.go`; discussion 2026-03-19. |
| FR-MCP-005 | S1 discussion | medium | Pressing `Ctrl-C` on `./till mcp` reports `ERRO ... context canceled`, which looks like a failure instead of normal shutdown. | Raw stdio server shutdown appears to work, but interrupt-driven exit is surfaced as an error path rather than a clean, user-expected cancellation. | Consensus: treat normal interrupt cancellation as clean shutdown in the next implementation wave. | OPEN | User terminal transcript 2026-03-19 showing `^C ... err=\"context canceled\"`; local code inspection in `cmd/till/main.go`. |
| FR-MCP-006 | S2 discussion | low | The user wants a human-usable MCP debugging surface, but not by overloading raw `till mcp`. | Best practice from MCP docs and `mcp-go` is to keep the stdio server command protocol-clean and provide a separate debug client/inspector if needed. | Consensus: keep `till mcp` as the raw server command; add a future visible `till mcp-inspect` developer MCP inspector/debug client instead of overloading `mcp`. | OPEN | Context7 `/mark3labs/mcp-go`; MCP debugging and inspector docs; discussion 2026-03-19. |

## 7) Final Sign-Off

- STDIO MCP startup complete: `PENDING`
- Local runtime path contract complete: `PENDING`
- User attribution complete: `PENDING`
- Agent attribution complete: `PENDING`
- Surface consistency complete: `PENDING`
- HTTP sanity complete: `PENDING`
- Final collaborative verdict: `PENDING`
