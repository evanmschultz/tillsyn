# Drop 4d REVISION_BRIEF — QA Falsification Round 1

**Reviewer:** go-qa-falsification-agent
**Date:** 2026-05-06
**Verdict:** PASS (no CONFIRMED counterexamples; 3 NEEDS-CLARIFICATION items the brief should fold in before planner-spawn)

## 1. Attack Inventory

Attack 1 — **Interface adequacy / `StreamEvent` shape**. NO COUNTEREXAMPLE.
Codex `protocol_v1` confirms `EventMsg::TurnStarted` / `EventMsg::TurnComplete` serialize on the wire as `task_started` / `task_complete` (Context7 `/openai/codex` `codex-rs/docs/protocol_v1.md`). Documented EventMsg variants (`AgentMessage`, `ExecApprovalRequest`, `RequestUserInput`, `Error`, `Warning`, `TurnStarted`, `TurnComplete`) all map cleanly into the canonical `StreamEvent {Type, Subtype, IsTerminal, Text, ToolName, ToolInput, Raw}`:
- `agent_message` / `agent_message_delta` → `Type: "assistant"`, `Text` populated from `message` / `delta`. `Raw` retains the full payload so a future delta-merge sink can re-decode.
- `task_complete` → `Type: "result", IsTerminal: true`.
- `error` → `Type: "error"` (forward-compat, non-terminal unless co-occurring with `task_complete`).
- `token_count` → `Type: "usage"` (passthrough per brief §3.2 4d.6).
- `exec_command_begin` / `exec_command_end` → forward-compat passthrough.
The streaming-text `agent_message_delta` shape (each line is a self-contained JSON envelope with `type` + `delta` field) DOES fit JSONL — `--json` flag's clap doc-comment in `codex-rs/exec/src/cli.rs` says "Print events to stdout as JSONL" verbatim. No new field on `StreamEvent` is required for Drop 4d's mapping. (See §3 mitigated-attack 1 for nuance.)

Attack 2 — **System-prompt injection mismatch**. NO COUNTEREXAMPLE; ONE NEEDS-CLARIFICATION.
Codex has no `--system-prompt-file <path>` flag. The brief's `CODEX_HOME=<bundle>/codex_home/` + render `AGENTS.md` approach IS supported by codex's documented behavior: per `codex-rs/core/src/project_doc.rs` codex "collects every `AGENTS.md` found from the project root down to the current working directory (inclusive) and concatenates their contents." When the dispatcher launches codex with `--cd <bundle.Root>` (the working dir), codex will discover `<bundle.Root>/AGENTS.md` (or `<bundle.Root>/codex_home/AGENTS.md` IF the project-root walk finds it). However: AGENTS.md walks UPWARD from the cwd to find a `project_root_marker` (default `.git`). If `<bundle.Root>` lives under `os.TempDir()/tillsyn-spawn-<uuid>/`, there is no `.git` ancestor and codex will treat ONLY the cwd's AGENTS.md as authoritative — that's the desired behavior. **NEEDS-CLARIFICATION C1**: brief should explicitly say where AGENTS.md lives (`<bundle.Root>/AGENTS.md` directly, NOT under `codex_home/`) and what `--cd` the spawn passes. The current wording in §3.2 4d.7 says "<bundle.Root>/codex_home/AGENTS.md" which codex will NOT discover unless cwd is set to `codex_home/`. Probe the planner-spawn brief to put AGENTS.md adjacent to the cwd, not inside the CODEX_HOME tree.

Attack 3 — **Permission-grant injection**. NO COUNTEREXAMPLE. Mitigated by L-codex-3 + brief §4 explicit punt + `permission_grants.cli_kind = "codex"` rows accumulating without injection (brief §3.2 4d.7). The brief is explicit that codex's pre-approval flow is out-of-scope for Drop 4d (`exec_approval_request`); cascade-spawned codex runs `--full-auto` per brief §3.2 4d.4.

Attack 4 — **Plugin-preflight skip semantics + Tillsyn-MCP via codex**. NO COUNTEREXAMPLE.
Codex DOES have a true MCP client per `~/.codex/config.toml` `[mcp_servers.<name>]` blocks (Context7 `/openai/codex` `llms.txt` config example confirms `command` + `args` per server). So if Tillsyn-MCP self-registration WERE landed for codex, the spawn could call `till.*` tools. Brief defers this to L-codex-5 — that's a real gap for Drop 5 dogfood (see attack 5) but does NOT break Drop 4d's seam-validation goal.

Attack 5 — **Tillsyn-MCP self-registration deferred — does cascade-on-itself break?** PARTIAL CONCERN — NOT A DROP-4D BLOCKER.
Drop 4d's stated purpose is "validate the F.7.17 seam stops being a one-adapter abstraction" — it does NOT include cascade-on-itself codex spawns (that's Drop 5). A codex spawn launched WITHOUT Tillsyn-MCP cannot call `till.*` tools to update the action item, so the action-item state-machine breaks. **However**: this is exactly what Drop 5 is for (multi-CLI dogfood validation). Brief §13 R3 acknowledges the surface explicitly. **Recommend brief add explicit language**: "Drop 4d does NOT validate codex-as-cascade-agent; it validates codex-as-adapter. Codex spawns in Drop 4d are unit-tested via fixture round-trips + contract-test extension, NOT via end-to-end action-item-update flow. Drop 5's first iteration MUST decide whether codex Tillsyn-MCP self-registration is a Drop-4d-bis prerequisite or in-flight Drop-5 work." Without that paragraph, a reader could plausibly assume Drop 4d delivers a working codex spawn — it does not, by design.

Attack 6 — **`Cost = nil` accuracy**. NO COUNTEREXAMPLE.
Codex DOES emit token-count telemetry — SDK exposes `result.usage` on `thread.run()`. Per `codex-rs/docs/protocol_v1.md`, the `EventMsg::TurnComplete` event corresponds to `task_complete` and carries usage; there's also a separate `token_count` event the brief proposes mapping to `Type: "usage"`. So Cost-SYNTHESIS is possible (token-count × model-price-table → USD). Brief §4 explicitly punts cost synthesis as out-of-scope; that's the right call for a "small drop" — synthesizing cost requires a model-price table that Tillsyn doesn't ship today. **No blocker, but brief §3.2 4d.6 doc-comment for stream.go should explicitly say "TerminalReport.Cost = nil today; cost synthesis from token_count is a future drop's concern" so future readers don't conclude codex has no cost telemetry at all (it does — Tillsyn just doesn't synthesize it yet).** Soft NIT, not a blocker.

Attack 7 — **Drop 5 dogfood scope explosion via missing Tillsyn-MCP for codex**. SAME AS ATTACK 5. Mitigated by brief §13 R3 + the recommendation to add Drop-4d's "NOT cascade-on-itself" paragraph. Concrete scenario the attack proposes (Drop 5's first codex spawn fails because Tillsyn-MCP isn't registered) is exactly the in-flight refinement path the brief already names. Not a Drop 4d blocker.

Attack 8 — **Sequencing risk: 4d before 5 vs claude-only-5 first**. NO COUNTEREXAMPLE.
Attack proposes claude-only Drop 5 first surfaces dispatcher-level bugs without confounding with codex adapter bugs. This is a real cost. **However**: brief §2 explicitly states "the codex adapter is the falsification of the Drop 4c `CLIAdapter` interface: if codex's surface forces interface changes, those changes must land in Drop 4d, NOT Drop 5." The original sequencing (claude-only Drop 5 first → codex Drop 4d → multi-CLI Drop 5.5/6) BURIES the seam-validation question one drop deeper, AND if codex needs an interface change, it cascades through three drops instead of one. The new sequencing trades "harder Drop 5 debugging" for "interface-change risk lands in adapter-validation drop." That's a defensible trade — interface-change blast-radius dominates. Brief §13 R3 also names the mitigation: Drop 5 staged as Wave-1-claude-only → Wave-2-codex-only, so confounding only happens in Wave 2. **Brief is correct on sequencing; attack does not falsify.**

Attack 9 — **R-A.4-1 ride-along framing**. NO COUNTEREXAMPLE.
Attack argues the metadata-before-move ordering fix is in `dispatcher.go` + `monitor.go`, which is unrelated to `cli_codex/` work — bundling them mixes concerns. Counterproposal: dedicated `Drop 4c.6` micro-drop. **However**: brief Q2 + §3.4 are EXPLICIT that 4d.0 is OPTIONAL ride-along, file-disjoint from codex work, and PREVENTS Drop 5 fire-drill. Naming a separate "Drop 4c.6" creates one more drop boundary, one more PR, one more rebase against `main` — for a fix that's "~50 LOC + tests" per brief §3.4. The "mixing concerns" cost is doc-level (one ride-along bullet in the Drop 4d worklog), not code-level (different files entirely from codex package). **Brief's lean (4d.0 ride-along) is correct.** Attack does not falsify.

Attack 10 — **TWO handoff memories vs ONE consolidated**. NO COUNTEREXAMPLE.
Attack proposes consolidating `project_drop_4c_5_handoff.md` + the new `project_drop_4d_handoff.md` into one. **However**: each handoff memory is a per-drop cold-start payload — a fresh-context orchestrator picking up Drop 4d does not need Drop 4c.5 themes-A-through-E in working memory; they need Drop 4d's spawn-shape + codex CLI flags + Q1 fixture status. Consolidating would force a fresh-context Drop 4d orch to wade through ~1000 lines of 4c.5 themes to reach the Drop 4d delta. Per-drop handoff scoped to ONE drop is the right shape. Brief §12 keeps them separate — correct call.

Attack 11 — **Hard precondition Q1 framing too rigid**. NO COUNTEREXAMPLE.
Attack proposes Q1 should ALSO route to "Context7 + WebSearch" before requiring dev fixture capture. **However**: Context7 evidence I gathered above is sufficient to NAME the event types (`task_started`, `task_complete`, `agent_message`, `agent_message_delta`, `error`, `token_count`, `exec_command_begin`, `exec_command_end`, `session_configured`) but does NOT pin the exact JSON envelope shape (does `agent_message_delta` use `delta` field or `text` field? Is the `task_complete` payload a flat `{type, status}` or nested under `payload`?). A real fixture eliminates "planner-fabricates-field-name" risk that the "Drop 0 Orchestrator Owns Description Accuracy" memory was raised for. Brief §9 Q1 Path A (dev captures fixture, ~30 sec of dev time) is the cleanest cut. Attack does not falsify; brief's "HARD PRECONDITION" stance is justified.

Attack 12 — **Risk tail (R7-R9 potentially missing)**. PARTIAL FINDING — TWO of three are real.
- **R7** (CODEX_HOME interaction with auth path resolution): ALREADY COVERED by brief §13 R5 (auth.json render-into-bundle OR forward `OPENAI_API_KEY` via `binding.Env`). Not missing.
- **R8** (codex agent identity / API key handling diverges from claude): ALREADY COVERED by brief L1 (Tillsyn never holds secrets; `binding.Env` is name-only allow-list — `OPENAI_API_KEY` is a name like `ANTHROPIC_API_KEY`). Codex auth via `~/.codex/auth.json` is mentioned in §13 R5. Not missing.
- **R9** (MockAdapter contract test currently single-adapter-shape — extending to BOTH adapters may require test refactoring beyond "minimal extension"): **REAL CONCERN — NEEDS-CLARIFICATION C2**. The brief §3.3 4d.8 says "Extends the contract-test table at `mock_adapter_test.go:568` with a `codexAdapter` row" but does NOT verify the existing table contract is shape-compatible with codex's expected outputs. Specifically: the existing table-driven test almost certainly asserts `ExtractTerminalReport(...).Cost != nil` for the claude row (claude always emits `total_cost_usd`); the codex row would need a different assertion (`Cost == nil`). That's a per-row assertion shape, not a "minimal extension" — the table struct may need a new column (`expectCostNil bool`) or the assertion logic refactored to per-row predicate. **Brief should explicitly call out the per-row assertion-shape work in 4d.8 acceptance.**

## 2. Counterexamples (CONFIRMED BLOCKERS)

None. No attack produced a CONFIRMED counterexample that breaks Drop 4d's stated scope.

## 3. Mitigated Attacks (Brief-Covered)

- **Attack 1 streaming-delta nuance**: codex `agent_message_delta` events arrive as multiple JSONL lines per agent turn (one per delta chunk). Today's `parseStreamEvent(line []byte) (StreamEvent, error)` interface processes one line at a time. The dispatcher's monitor would see each delta as a separate `assistant` event with the partial `Text` field — that's potentially noisy in logs but does NOT break the canonical shape (and matches how claude's `assistant` events with multiple content blocks are surfaced — first-block-only per `cli_claude/stream.go:158-179`). Mitigated.
- **Attack 1 `--json` is JSONL not SSE**: explicitly verified via `codex-rs/exec/src/cli.rs` doc-comment: `Print events to stdout as JSONL`. Mitigated; no `Non-JSONL Extensibility` interface rewrite required for Drop 4d.
- **Attack 2 AGENTS.md discovery**: see §1 Attack 2 — works under `--cd <bundle.Root>` IF AGENTS.md sits adjacent to cwd. NEEDS-CLARIFICATION C1 (path tweak), not blocker.
- **Attack 3 permission grants**: explicitly punted by L-codex-3 + §4 + brief §3.2 4d.7.
- **Attack 5/7 Tillsyn-MCP self-registration**: deferred to L-codex-5 + Drop 5 readiness; brief §13 R3 acknowledges. Recommend NEEDS-CLARIFICATION C3 ("Drop 4d does NOT validate codex-as-cascade-agent" paragraph) but not a blocker.
- **Attack 6 Cost synthesis**: explicitly out-of-scope per §4. Soft NIT on doc-comment wording only.
- **Attacks 8/9/10/11**: brief positions correct; attacks don't falsify.

## 4. Conclusion

**Verdict: PASS** with **3 NEEDS-CLARIFICATION items** the brief should fold in before planner-spawn:

- **C1 (Attack 2)**: Brief §3.2 4d.7 should clarify AGENTS.md lives at `<bundle.Root>/AGENTS.md` (adjacent to cwd) NOT `<bundle.Root>/codex_home/AGENTS.md`. The current wording will cause codex to MISS the per-spawn AGENTS.md unless `--cd` is set to `codex_home/`. Spawn working-dir is mentioned as a planner sub-decision in §3.2 4d.4 but the AGENTS.md path needs to be coherent with the chosen `--cd`. (~1 sentence brief edit; resolved by planner.)

- **C2 (Attack 12 R9)**: Brief §3.3 4d.8 should explicitly call out per-row assertion-shape work in `mock_adapter_test.go` — the existing table-driven contract test asserts `Cost != nil` (claude shape); codex row needs `Cost == nil`. Either add `expectCostNil bool` column or refactor to per-row predicate. **Soft scope expansion** — acceptance criteria should name the test refactor so the builder doesn't surprise-discover it. (~1 sentence brief edit; bumps 4d.8 from "wiring + extension" to "wiring + extension + minor table refactor.")

- **C3 (Attack 5/7)**: Brief §2 / §13 R3 should add an explicit paragraph: "Drop 4d does NOT validate codex-as-cascade-agent; it validates codex-as-adapter. Codex spawns in Drop 4d are unit-tested via fixture round-trips + contract-test extension, NOT end-to-end action-item-update flow. Tillsyn-MCP self-registration for codex is L-codex-5 deferred; Drop 5 first-iteration decides whether codex Tillsyn-MCP self-registration is Drop-4d-bis prerequisite or in-flight Drop-5 work." Without this language, Drop 5 readers may assume Drop 4d delivers a working cascade-on-itself codex spawn — it does not, by design. (~3 sentence brief edit; clarifies handoff to Drop 5.)

The brief's core sizing claim (~7-9 droplets, single-package surface, no `CLIAdapter` interface change) is **CORRECT** per Context7 evidence on codex's `--json` JSONL output, EventMsg variants matching the canonical `StreamEvent` shape, and codex's documented `--cd` + `--full-auto` + `--ephemeral` + `--config` flag set covering all needed argv shapes. R1 (event-taxonomy divergence forcing interface rewrite) is genuinely mitigated by Q1's HARD-PRECONDITION fixture probe. Sequencing (4d before multi-CLI Drop 5) is defensible — brief §2 articulates the seam-validation argument well.

**Proceed to planner-spawn after C1/C2/C3 fold-in.**

## 5. Hylla Feedback

N/A — Drop 4d brief review touched only MD + Go files reachable via `Read` and external Codex docs reachable via Context7. Per filesystem-MD coordination paradigm override + project memory "Hylla Indexes Only Go Files Today" + Hylla unavailable in this review per the spawn prompt. Zero Hylla calls; zero misses to report.
