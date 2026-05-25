# Agent Dispatch Model — tillsyn/main

> tillsyn is an **FE+Go** repo using the **tillsyn** coordination substrate
> (`mcp__tillsyn__till_*`; `action_item` + `kind=plan/build/human-verify` + `structural_type`
> + `blocked_by`). The **canonical dispatch model** lives in `sand/main/AGENT_DISPATCH.md` —
> read it for the full dual-path + hermetic-codex + MCP-injection spec. This file explains how
> to make the Playwright MCP + hermetic headless-codex setup work for tillsyn's OWN agents, and
> records the audit of tillsyn's agents vs the ta-family. **This is a doc only — do not change
> tillsyn's agent personas to match; the substrate difference below is intentional.**

## How to make Playwright MCP + hermetic codex work for tillsyn's codex agents

Apply the keystone §4 hermetic model, substituting the **tillsyn** substrate for `ta`:

- `codex exec --ignore-user-config -c project_doc_max_bytes=0 --ignore-rules` + hermetic
  `CODEX_HOME` (throwaway dir, only `auth.json`/`version.json`/`installation_id`/`models_cache.json`
  symlinked) + `-c web_search="live"`.
- Role-conditional inline `-c "mcp_servers.<name>={…}"` with per-tool `approval_mode="approve"`:
  - **`tillsyn`** — always (stdio: `till mcp`) — this REPLACES the `ta` injection used by the
    ta-family. (Under hermetic `CODEX_HOME`, the HOME `mcp_servers.tillsyn` is ignored, so it
    must be injected inline.)
  - `hylla` — planning + plan-qa only, read-only; skip for `*build-qa*`.
  - `context7` — always (HTTP, `CONTEXT7_API_KEY` env header).
  - `gopls` — `*-go-*` roles.
  - `playwright` — `*-fe-*` roles (`playwright-mcp --headless --isolated`; `@playwright/mcp`
    global npm install; browsers in `~/Library/Caches/ms-playwright`).
- **Capabilities come from MCP injection, NOT codex skills** (keystone §5). The global
  `~/.codex/skills/playwright` SKILL.md is correctly ignored under hermetic `CODEX_HOME`; the
  injected `@playwright/mcp` provides the browser tools. Nothing goes in `.agents/skills`/`.codex/`.
  (Caveat: codex BUNDLED skills remain visible but inert; codex #14316.)
- **Endpoints**: tillsyn Wails AssetServer `http://localhost:34115`; bare Astro `http://localhost:51428`
  (binding-less, never target). Personas are generic; the orchestrator passes the live-backend
  URL into FE spawn prompts (tillsyn's CLAUDE.md is the source of truth + carries that rule).
- **Dispatch channels** (keystone §3): OAuth claude → built-in Agent tool only (never `claude -p`);
  codex (planning/qa-falsification) → hermetic `codex exec`; non-OAuth claude (ollama/API-key)
  → `claude -p --bare` + injected `--mcp-config`.

## Audit: tillsyn agents vs the ta-family (2026-05-24)

The ta-family personas (hylla/ta/valv/sand + `~/.ta`) were DERIVED from tillsyn's split
personas. Audit of tillsyn's `.claude/agents/` (13: closeout + 6 fe + 6 go) vs the ta-family
confirms **the ONLY drift is substrate**, as intended:

- **Coordination model**: tillsyn = `mcp__tillsyn__till_*` (+ `tillsyn-dev` mirror), `action_item`,
  `kind=plan/build/human-verify`, `structural_type=droplet/drop/segment`, `metadata.blocked_by`,
  `till.comment`/`till.handoff`/`till.attention_item`. ta-family = `mcp__ta__*`,
  `cascade.planner`/`cascade.droplet`, `blockers`, `attention_needed`, verdict via `mcp__ta__update`.
- **Endpoints**: tillsyn hardcodes `34115`/`51428`; ta-family personas are generic (orch-provided).
- **Packages**: tillsyn `github.com/evanmschultz/tillsyn/ui`; each ta-family project its own.
- **NO structural/behavioral drift**: the QA proof/falsification split, the plan/build axis split,
  the recursive-planning + atomicity rules ("multi-level decomposition is the norm"; 3-block
  droplet = anti-pattern → emit a `kind=plan`/`cascade.planner` child), the FE CSS-first/zero-JS/
  a11y rules, the Playwright-at-3-breakpoints mandate, the evidence order, and the Section 0 +
  response formats all MATCH.

No non-substrate conflict was found. If tillsyn later diverges structurally, record it here.

## Translation-logic handoff (with sand)

The skills/hooks → codex translation logic (keystone §7) is shared work between sand and
tillsyn. **Whoever builds + confirms it first hands it off to the other.** The target end-state
is project-local skills/hooks/MCP so hermetic headless codex ignores all global state and still
has what it needs. See `sand/main/AGENT_DISPATCH.md` §7.
