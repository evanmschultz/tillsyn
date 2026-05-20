# DROP 4D CODEX — Executor Session Prompt

**Copy the entire content below the `---` line into the first message of a fresh Claude Code session launched from `/Users/evanschultz/Documents/Code/hylla/tillsyn/main`. This session executes the codex adapter build + cascade through drop_4d_codex to merge + starts the multi-backend dogfood arc.**

The dev's PRIMARY session (the one this file came from) stays with the dev to work through MD migration + cleanup. THIS session is the executor — focused on shipping drop_4d_codex.

---

You are a Claude Code orchestrator session in the Tillsyn project (`/Users/evanschultz/Documents/Code/hylla/tillsyn/main`). Your specific job is to finish executing **drop_4d_codex** — the codex CLIAdapter implementation that unlocks multi-backend dogfood and ~60-70% Anthropic spend reduction. After it merges, start the dogfood arc by routing future drops' planning + QA-falsification roles to codex.

A parallel session (the dev's primary working session) is working on MD-to-Tillsyn migration in this same repo. Coordinate via Tillsyn handoffs + comments if your work touches their files. Do NOT touch files outside drop_4d_codex's scope (paths listed below) unless explicitly asked.

## Read These In Order Before Doing Anything

1. **`~/.claude/projects/-Users-evanschultz-Documents-Code-hylla-tillsyn/memory/project_2026_05_20_session_handoff.md`** — full session-end state from the work that produced this prompt. Read it cold.
2. **`~/.claude/projects/-Users-evanschultz-Documents-Code-hylla-tillsyn/memory/project_multi_backend_dogfood_direction.md`** — LOAD-BEARING permanent architectural reference. Hylla-verified picture of what tillsyn already ships + the routing thesis + the security stance against bash bridges.
3. **`main/CLAUDE.md`** — auto-loaded by Claude Code; pay special attention to `## Hard Rules (Inviolable)` (no time estimates, Tillsyn-only tracking, mage-only Go gates, no bash dispatcher bridges, no arbitrary-argv knobs, atomicity in planner prompt, multi-backend dogfood, **ALWAYS parallelize FE and core**).
4. **`main/WIKI.md`** — current best-practice snapshot for cascade vocabulary + Tillsyn-native coordination.
5. **Tillsyn drop state**: call `mcp__tillsyn__till_action_item(operation=get, action_item_id="ee5f16f8-931e-4730-bc7f-a03b1d506804")` once you've claimed auth. That's the drop_4d_codex root.

## Project Anchor

- **Project ID**: `5d9b530c-b568-4830-9e16-058c957cfc05`
- **Project slug**: `tillsyn`
- **Project name**: `TILLSYN`
- **Hylla artifact**: `github.com/evanmschultz/tillsyn@main`
- **Working dir**: `/Users/evanschultz/Documents/Code/hylla/tillsyn/main`
- **Git state at handoff**: 6 commits ahead of `origin/main`, working tree clean.

## Auth Setup (Do First)

The previous session's auth + lease expire `2026-05-21T02:15:54Z`. If you're starting after that timestamp, request fresh:

```
mcp__tillsyn__till_auth_request(
  operation=create,
  path=project/5d9b530c-b568-4830-9e16-058c957cfc05,
  principal_id="DROP_4D_CODEX_EXECUTOR_ORCH",
  principal_name="DROP_4D_CODEX_EXECUTOR_ORCH",
  principal_type=agent,
  principal_role=orchestrator,
  client_id="claude-code-cli",
  client_name="Claude Code CLI",
  reason="Execute drop_4d_codex D2/D3/D5 cascade + start multi-backend dogfood arc.",
  requested_ttl="8h",
  timeout="30m",
  wait_timeout="30m",
)
```

After dev approves in TUI, claim with the `request_id` + `resume_token` from the create response. Then issue an orchestrator-role capability lease via `mcp__tillsyn__till_capability_lease(operation=issue, role=orchestrator, requested_ttl_seconds=28800)`. Pass `session_id` + `session_secret` + `auth_context_id` + `agent_instance_id` + `lease_token` on every Tillsyn mutation.

**IMPORTANT lesson learned**: do NOT use `timeout=8h` on auth_request create — the request expires before dev can react. Use `timeout="30m"` + `wait_timeout="30m"`.

## drop_4d_codex Status At Handoff

**Root**: `ee5f16f8-931e-4730-bc7f-a03b1d506804` (state=`in_progress`, kind=`plan`, structural_type=`drop`).

**Children (8 + auto-QA twins)**:

| ID | Title | Kind | State | Notes |
|---|---|---|---|---|
| `6a63a608-e902-4447-b6f0-3944477b565d` | D1 ADD CLIKINDCODEX CONST AND ISVALIDCLIKIND ROW | build | **complete** | Shipped at `fab5fef`. QA twins complete (retroactive). |
| `ef328c37-cfdc-4611-95da-c482d57a53f5` | D2 CLI_CODEX ADAPTER PACKAGE ADAPTER ARGV ENV STREAM | build | **todo** | **Blocked on OQ1 + OQ2 from dev.** |
| `77e1e6b2-cecc-4c4b-95e0-e85c9546ed88` | D3 CLI_CODEX REGISTER PACKAGE INIT WIRING | build | **todo** | `blocked_by: D2`. |
| (attention_item `8232baba-...`) | D4 codex fixture capture (dev action) | — | open | Dev provides JSONL fixture from `codex exec --json`. |
| `98ce4c90-47b5-43ee-9735-f565a2e7b926` | D5 CLI_CODEX TABLE-DRIVEN TESTS AGAINST FIXTURE | build | **todo** | `blocked_by: D2 + D4`. |
| `91ad4120-ed46-4280-8642-1eb21f7667db` | D6 SUBSTANTIVE EMBEDDED AGENT PROMPTS FOR GEN GROUP | build | **complete** | Shipped at `580794e`. |
| `1f90b33c-8170-404b-8d47-9211f7745a95` | D7 MULTI-BACKEND ROUTING EXAMPLE IN AGENTS.EXAMPLE.TOML | build | **complete** | Shipped at `d2cf520`. Contains 4× `MODEL_PLACEHOLDER` awaiting OQ2. |
| `e67c3238-e7fc-4fb9-94c6-a997e1a1a101` | D8 CLAUDE.MD AND WIKI.MD COORDINATION-MODEL SWEEP | refinement | **complete** | Shipped at `ee8d0df`. |

**Plan-QA twins on the root (BOTH todo)**: `58e7e9c2-...` (plan-qa-falsification), `8d35985c-...` (plan-qa-proof). Move to complete with retroactive-verdict comments after the cascade finishes OR dispatch real plan-QA agents — orch's call based on time/cost budget.

## Hard Prereqs From Dev (Block The Build)

These three items live as `till.attention_item` `8232baba-8272-48ad-85b3-69ee99f6d2e7`. Until dev provides them, D2 / D5 cannot ship cleanly:

1. **OQ1 — codex argv shape**: paste of `codex exec --help` (or `codex --help`). Need: how prompt is passed (`-i <file>` / positional / stdin), whether `--ephemeral` / `--skip-git-repo-check` are real flags, exact `model_reasoning_effort` flag syntax. Without this, D2 `argv.go` is speculation.
2. **OQ2 — codex model identifier**: actual model string for dev's ChatGPT-tier auth (e.g. `gpt-4o` / `o3` / `o4-mini` / `gpt-5-codex` — unknown until dev tells you). Replaces 4× `MODEL_PLACEHOLDER` in `internal/templates/builtin/agents.example.toml`.
3. **D4 fixture**: dev runs `codex exec --json -m <model> <<< "Reply with just: ok"` and saves stdout to `main/internal/app/dispatcher/cli_codex/testdata/codex_stream_minimal.jsonl` (or pastes for you to save). Needs init + assistant/result + terminal events.

**Ask the dev for these first thing.** Until they land, you can read code + Hylla in parallel to confirm D2's design, but can't ship the actual adapter.

## What You Build (Once Dev Unblocks)

### D2 — `cli_codex` adapter package

Files to create:
- `internal/app/dispatcher/cli_codex/adapter.go` — unexported `codexAdapter{}` struct + compile-time `var _ dispatcher.CLIAdapter = (*codexAdapter)(nil)` assertion + `New() dispatcher.CLIAdapter` constructor. `BuildCommand` calls `assembleArgv` + `assembleEnv`, returns `exec.CommandContext(ctx, "codex", argv[1:]...)` with explicit `cmd.Env`.
- `internal/app/dispatcher/cli_codex/argv.go` — `assembleArgv(binding dispatcher.BindingResolved, paths dispatcher.BundlePaths) []string`. Real flag shape per OQ1.
- `internal/app/dispatcher/cli_codex/env.go` — `assembleEnv(binding dispatcher.BindingResolved) ([]string, error)`. Mirror `cli_claude/env.go` closed POSIX baseline. Codex uses `OPENAI_API_KEY` not `ANTHROPIC_API_KEY`.
- `internal/app/dispatcher/cli_codex/stream.go` — `ParseStreamEvent` + `ExtractTerminalReport`. Codex JSONL event family → canonical `dispatcher.StreamEvent`. Preserve raw JSON.

Reference: read `internal/app/dispatcher/cli_claude/` package end-to-end before starting D2 — it's the reference implementation. Use Hylla `hylla_node_full` on each cli_claude symbol.

### D3 — register/init wiring

- `internal/app/dispatcher/cli_codex/register/register.go` — `init()` → `dispatcher.RegisterAdapter(dispatcher.CLIKindCodex, cli_codex.New())`.
- Update `cmd/till/main.go` (or wherever `cli_claude` is blank-imported) to add `_ "github.com/evanmschultz/tillsyn/internal/app/dispatcher/cli_codex/register"`.

Find the existing cli_claude blank-import location FIRST via Hylla (`hylla_refs_find` on `cli_claude/register` or grep for `_ "github.com/evanmschultz/tillsyn/internal/app/dispatcher/cli_claude"`).

### D5 — table-driven tests

- `internal/app/dispatcher/cli_codex/adapter_test.go` (or split into `argv_test.go`, `stream_test.go`, etc. — mirror `cli_claude/` test layout).
- Required tests: `TestBuildCommandArgvShapeMinimal`, `TestBuildCommandArgvShapeWithEffort`, `TestBuildCommandHardcodedBinary`, `TestEnvNotInheritedFromOSEnviron`, `TestParseStreamEventFromFixture`, `TestExtractTerminalReportFromFixture`.
- Tests read `testdata/codex_stream_minimal.jsonl` (the D4 fixture from dev).

### Plan-QA twins resolution (optional but cascade-honest)

`58e7e9c2-...` + `8d35985c-...` sit `todo` on the root. After D2/D3/D5 complete + commit, either:
- **Honest path**: dispatch real plan-qa-proof-agent + plan-qa-falsification-agent against the planner's full output. Costs Anthropic tokens; ensures cascade discipline.
- **Pragmatic path**: orch posts verdict comments on each ("plan implemented as decomposed; no falsification surfaced during build"), moves to complete. Documents the shortcut.

Dev's call — ask if uncertain.

## Drop Closeout (After D2 + D3 + D5 Land)

1. `mage ci` from `main/` worktree root. Must pass green. Coverage ≥70% on touched packages.
2. `git push origin main` (the dev approved 6 commits ahead at handoff; you'll add 3-4 more).
3. `gh run watch --exit-status` until CI green.
4. Hylla reingest from remote: `mcp__hylla__hylla_ingest(source_url="github.com/evanmschultz/tillsyn", ref="<HEAD-SHA>", enrichment_mode="full_enrichment")`. **NEVER ingest before push + CI green.** **NEVER ingest from local working copy.**
5. Update `project_multi_backend_dogfood_direction.md` memory with "drop_4d_codex SHIPPED at HEAD `<sha>`" + the codex routing thesis is now LIVE.
6. Move root `ee5f16f8-...` to `complete` after plan-QA twins close.

## Multi-Backend Dogfood Arc (After Drop Merges)

Once `drop_4d_codex` is on `origin/main` + Hylla reingested, every subsequent drop benefits from codex routing IF the agents.toml is wired and the dispatcher reads it. Two paths:

### Path A — Verify dispatcher reads agents.toml on next drop

The dispatcher's `ResolveBinding` (at `internal/app/dispatcher/binding_resolved.go`) merges template bindings with overrides. The `agents.example.toml` shipped in D7 has `client = "codex"` for `plan` + `*-qa-falsification` + `research`. When a real drop with those kinds dispatches, the dispatcher should route to codex IF the agents.toml is loaded.

**Verify by smoke test**: create a small `kind=plan` action_item, move to `in_progress`, watch what gets spawned. Should be codex if routing works.

### Path B — drop_4e_ollama (separate future drop)

Per `project_multi_backend_dogfood_direction.md`, ollama support is DEFERRED until a verification spike confirms `claude -p --bare` actually talks to Ollama's localhost endpoint (Context7 surfaced Ollama exposes OpenAI-compat, not Anthropic-compat — needs spike).

**DO NOT** start drop_4e_ollama from this session. That's a separate drop with its own scope decision.

## Hard Rules You MUST Follow

These are in `main/CLAUDE.md § Hard Rules (Inviolable)` — repeated here so they can't be missed:

1. **No human time estimates** — use cascade-shape work estimates (droplets / plans / drops). NEVER "1-2 days" / "a few hours" / "a week."
2. **Tillsyn-only for work tracking** — no Claude Code built-in `TaskCreate` / `TaskUpdate` / `TaskList` / `TaskGet` / `TaskStop` / `TaskOutput`. Finer granularity goes in child Tillsyn action_items.
3. **Mage targets only for Go gates** — never `go test` / `go build` / `go vet` / `gofmt`. Always `mage <target>`. If a target is missing, ADD it.
4. **No bash-dispatcher bridges in this repo** — Tillsyn's adapter framework IS the dispatch surface. Sandbox is declarative; process isolation is OS-level.
5. **No arbitrary-argv knobs on `BindingResolved`** — REV-1 supersession killed `Command []string` + `ArgsPrefix []string`. Templates declare `cli_kind`; adapters encapsulate argv.
6. **Atomicity is a planner-prompt concern, not dispatcher Go code** — builders' droplet sizing is enforced via planner prompt ("≤4 small blocks per build droplet") + structural file/package locks.
7. **Multi-backend dogfood is the cost-relief mechanism** — after drop_4d_codex merges, plan + QA-falsif route to codex; QA-proof stays on opus; build + commit stay on haiku.
8. **ALWAYS parallelize FE and core work** — NEVER serialize without a real `blocked_by` dependency naming a specific cross-lane symbol. Dispatch fe-builder-agent + go-builder-agent concurrently against unblocked droplets every cascade tick.

## Coordination Rules (Tillsyn-Native)

- **No new MD files for workflow/planning.** `workflow/drop_*/` MDs are HISTORICAL AUDIT per `feedback_never_remove_workflow_files.md`. Don't write `BUILDER_WORKLOG.md` / `PLAN_QA_*.md` / `CLOSEOUT.md` etc. — builder outputs go in `till.comment` on the build action_item; QA verdicts go in comments on QA twin action_items.
- **Builder spawns**: use Agent tool with `subagent_type=go-builder-agent` (or `fe-builder-agent` for FE work), `run_in_background=true` by default. Pass Tillsyn auth credentials + action_item_id + Section 0 directive in the spawn prompt.
- **Subagents do NOT inherit CLAUDE.md or the Section 0 output style.** Every subagent spawn prompt MUST carry the Section 0 directive verbatim.
- **Parallel QA pairs** (qa-proof + qa-falsification) MUST go background — they're independent. Same for any naturally parallel agent set.

## Section 0 Reasoning (Required On Every Substantive Response)

Render `# Section 0 — SEMI-FORMAL REASONING` with 5 passes (Planner / Builder / QA Proof / QA Falsification / Convergence) before the numbered body. Each pass uses the 5-field certificate (Premises / Evidence / Trace or cases / Conclusion / Unknowns). Section 0 stays in the orchestrator-facing response ONLY — never in Tillsyn descriptions, comments, or completion notes. Trivial one-line answers skip Section 0.

## What You Must NOT Do

- DO NOT touch the other parallel session's MD migration work. The dev is migrating PLAN.md / REFINEMENTS.md / HYLLA_REFINEMENTS / E2E_FIXES / Claude TOS / drop_fe_2 decomposition. That's their lane (action_item `44187373-9b32-4078-b756-20f698f9a34a` "DROP MD MIGRATION FULL SWEEP TO TILLSYN NATIVE FOR MVP FEATURE COMPLETE").
- DO NOT speculate on codex argv if dev hasn't provided OQ1. Ask, wait, then build.
- DO NOT ingest Hylla before push + CI green.
- DO NOT use `--no-verify` / `--amend` / `--no-edit` on commits.
- DO NOT skip Section 0 on substantive turns. (The previous session got called out hard for this.)
- DO NOT use a separate `kind=plan` action_item structure for the codex adapter — the existing drop_4d_codex tree is the structure. You're an executor, not a re-planner.

## First-Turn Checklist

When this session boots, your first turn should:

1. Read the 5 files listed in "Read These In Order Before Doing Anything" above.
2. Claim auth (request → claim → lease per the auth setup section).
3. `mcp__tillsyn__till_capture_state(project_id="5d9b530c-...", view="summary")` to anchor.
4. `mcp__tillsyn__till_attention_item(operation=list, project_id="5d9b530c-...", all_scopes=true, state=open)` — find attention_item `8232baba-...` (dev's codex info ask).
5. `mcp__tillsyn__till_action_item(operation=get, action_item_id="ee5f16f8-...")` — drop_4d_codex root.
6. Check if dev has provided OQ1 / OQ2 / D4 fixture (look in chat history of the new session + check for the fixture file at `main/internal/app/dispatcher/cli_codex/testdata/codex_stream_minimal.jsonl`).
7. If unblocked: start D2 builder dispatch (background, go-builder-agent, with the spawn prompt containing Section 0 directive + action_item_id `ef328c37-...` + auth credentials).
8. If still blocked: chat with dev to get OQ1 + OQ2 + D4 — be specific about what each unblocks.

## When Drop_4d_codex Is Fully Done

Final response to dev should include:
- Final commit SHA on `origin/main`.
- Hylla ingest task ID + status.
- Updated `project_multi_backend_dogfood_direction.md` memory with "SHIPPED" marker.
- Confirmation that next drop will route plan + QA-falsif → codex automatically via the dispatcher reading agents.toml.
- A new handoff memory if the dogfood-arc validation needs another session.

Then you can wind down. The dev's primary session continues with MD migration in parallel; coordinate via Tillsyn comments if anything cross-touches.

---

End of executor prompt. Dev: copy from the `---` line at the top through this `---` line into the new session's first message. The new session will boot with focused context and execute drop_4d_codex through to merge + dogfood-arc kickoff.
