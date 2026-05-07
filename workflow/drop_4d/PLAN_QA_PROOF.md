# Drop 4d REVISION_BRIEF — QA Proof Round 1

**Reviewer:** go-qa-proof-agent
**Date:** 2026-05-06
**Verdict:** PASS

## 1. Trace Coverage

Each check below maps to a brief claim with file/line + external-source citations.

### 1.1 Adapter seam claim accuracy (cli_adapter.go:33,42) — COVERED

Brief claims `cli_adapter.go:33,42` is the exact F.7.17 seam (`CLIKindClaude` constant + `IsValidCLIKind` switch).

- Line 33 (verbatim): `const CLIKindClaude CLIKind = "claude"`. ✅
- Line 42 (verbatim): `func IsValidCLIKind(k CLIKind) bool {`. ✅
- Switch body lines 43-48: `case CLIKindClaude: return true; default: return false`. ✅
- BindingResolved.CLIKind doc-comment lines 108-111 contain the exact "default-to-claude per F.7.17 L15" prose the brief edits in 4d.1. ✅

### 1.2 Reference implementation file count — COVERED (with minor footnote)

Brief claims `cli_claude/{adapter,argv,env,stream,init}.go` plus `render/render.go` plus `testdata/claude_stream_minimal.jsonl`.

- Verified directory listing of `internal/app/dispatcher/cli_claude/`: `adapter.go`, `adapter_test.go`, `argv.go`, `env.go`, `init.go`, `stream.go` (6 files in package root).
- Brief enumerates 5 production files (omitting the test file `adapter_test.go`); reasonable since CLI_ADAPTER_AUTHORING.md §1 lists `adapter_test.go` separately on line 42 of that doc. Not a defect.
- `render/` subdir: `render.go`, `init.go`, `render_test.go`. Brief said `render.go`; consistent.
- `testdata/`: `claude_stream_minimal.jsonl`. ✅

### 1.3 Codex CLI identity (OpenAI's openai/codex) — COVERED

Brief claims OpenAI's `codex` (Rust, openai/codex). Confirmed via Context7 `/openai/codex` evidence:

- `codex-rs/exec/src/cli.rs` source returned by Context7 query confirms Rust origin + `openai/codex` repo path. ✅
- `--json` flag exists with `pub json: bool` and doc-comment "Print events to stdout as JSONL." ✅
- Codex DOES emit JSONL via `codex exec --json` per Context7 source verbatim.

### 1.4 Argv divergence claims — COVERED

Brief claims codex has NO `--bare`, `--plugin-dir`, `--system-prompt-file`, `--mcp-config`, `--settings`, `--permission-mode`, `--strict-mcp-config`, `--no-session-persistence`.

Verified via Context7 dump of `codex-rs/exec/src/cli.rs`: codex's exec-mode flag set is `--full-auto`, `--dangerously-bypass-approvals-and-sandbox` (alias `--yolo`), `--cd` / `-C`, `--skip-git-repo-check`, `--add-dir`, `--ephemeral`, `--output-schema`, `--config` (config_overrides), `--color`, `--json`, `--output-last-message` / `-o`, positional `PROMPT` (or `-` for stdin). ✅ NO claude-specific flags appear in this enumeration. Cross-checked against `cli_claude/argv.go:25-50` doc-comment which lists the claude argv recipe — disjoint sets. ✅

### 1.5 CODEX_HOME env-override claim — COVERED

Brief claims per-spawn isolation requires `CODEX_HOME=<bundle>/codex_home/`.

Verified via Context7 evidence: `export CODEX_HOME="${CODEX_HOME:-$HOME/.codex}"` is the canonical pattern (codex skill assets sample). Default resolution is `~/.codex/`. Setting `CODEX_HOME` redirects codex's config root, which is the exact knob brief identifies for per-spawn isolation. ✅ (Note: codex source-code doc on `CODEX_HOME` resolution semantics is partial in Context7 — the brief acknowledges this in §13 R5 with a planner subdecision around `auth.json` rendering. Acceptable per-brief phrasing.)

### 1.6 Stream taxonomy partial-evidence claim — COVERED (with documented uncertainty)

Brief explicitly says codex's `EventMsg` includes `session_configured`, `agent_message`, `agent_message_delta`, `task_complete`, `error`, `token_count`, but per-event field shape NOT pinned and Q1 fixture is hard precondition.

Context7 evidence (Rust source `codex-rs/docs/protocol_v1.md`) returns `EventMsg::AgentMessage`, `EventMsg::ExecApprovalRequest`, `EventMsg::RequestUserInput`, `EventMsg::TurnStarted`, `EventMsg::TurnComplete`, `EventMsg::Error`, `EventMsg::Warning`. Note: brief uses snake_case (`task_complete`, `agent_message`) — the wire-form for `--json` JSONL output — while protocol_v1 PascalCase (`TurnComplete`, `AgentMessage`) is the Rust enum. Brief's mapping captures the correct semantic family with appropriate uncertainty hedging in Q1 / R1 / R2.

The `task_complete` vs `TurnComplete` case-form discrepancy is a real evidence gap, but brief explicitly routes Q1 (Path A: dev captures fixture) as HARD PRECONDITION before 4d.6 builders fire, and §13 R1 covers the worst-case interface-rewrite scenario. The hedge is appropriate. ✅

### 1.7 Cost = nil / Denials = nil design — COVERED

Brief says F.7.17 L11 designed `*float64 Cost` for adapters lacking cost telemetry.

- Verified `cli_adapter.go:282-291`: `TerminalReport` doc-comment says "Per F.7.17 locked decision L11 Cost is *float64 so adapters whose CLI does not emit cost telemetry can return (TerminalReport{Cost: nil, ...}, true) without the caller mistaking absent-cost for zero-cost." ✅ Direct quote matches brief's claim.
- L-codex-2 (`Cost = nil for codex`) and L-codex-3 (`Denials = nil for codex`) are textbook applications of L11's design intent.

### 1.8 Plugin-preflight claude-specificity — COVERED

Brief says `plugin_preflight.go` calls `claude plugin list --json`.

- Verified `plugin_preflight.go:80-82`: "ClaudePluginLister is the package-private test seam between CheckRequiredPlugins and the underlying `claude plugin list --json` invocation." ✅
- Lines 100, 121: error sentinels `ErrClaudeBinaryMissing`, `ErrPluginListUnparseable` — claude-specific by name + semantic. ✅
- Lines 232-233: production lister "shells out to `claude plugin list --json` via exec.CommandContext." ✅
- Hardcoded coupling to `claude` binary; routing for codex (skip) is a real seam-extension need.

### 1.9 Permission-grants table accepts cli_kind=codex (line 57) — COVERED

Brief cites `permission_grants_repo_test.go:57`.

- Verified line 57: `for _, want := range []string{"id", "project_id", "kind", "rule", "cli_kind", "granted_by", "granted_at"} {` — the schema-existence check confirms the `cli_kind` column is in the table. ✅
- Brief said the table "already accepts `cli_kind=codex` rows" — strictly the schema accepts any string in `cli_kind`; no enum constraint at the SQLite level. Test confirms column existence; insertion of `"codex"` literal is implicit (column is text). ✅

### 1.10 R-A.4-1 line citations — COVERED

Brief cites `monitor.go:344` (`applyCrashTransition`) and `dispatcher.go:639` (`transitionToFailed`).

- `monitor.go:344-348`: `// applyCrashTransition is the crash-handling pipeline...` `func (m *processMonitor) applyCrashTransition(ctx context.Context, actionItemID string, outcome TerminationOutcome) error {`. Function name and start line match exactly. ✅
- `monitor.go:371-391`: confirmed move-to-failed (`MoveActionItem` line 371) BEFORE outcome metadata write (`updated.Outcome = "failure"` line 383, then `UpdateActionItem` line 386). Order: MOVE → THEN METADATA. Violates Theme A "metadata-before-move." ✅
- `dispatcher.go:631-664`: `// transitionToFailed moves item to its project's failed column...` `func (d *dispatcher) transitionToFailed(ctx context.Context, item domain.ActionItem, reason string) error {` at line 639. ✅
- Lines 651-661: confirmed same MOVE-then-METADATA pattern (`MoveActionItem` 651 before `updated.Outcome = "failure"` 655, `UpdateActionItem` 657). ✅

Both citations are EXACT and the ordering violation is real. R-A.4-1 routing as 4d.0 ride-along is grounded.

### 1.11 Q1-Q6 lean justifications — COVERED

Spot-checks:

- **Q1 (Path A — dev captures fixture)**: Justification "Cleanest. ~30 seconds of dev time. Drop 4d planner-spawn gates on the fixture." Concrete, defensible. The risk-cost tradeoff is correct: planner / builder fabrication of event-string strings creates bigger downstream pain than 30 sec of dev time. Memory `feedback_drop0_orchestrator_description_drift.md` directly applies (concrete-symbol fabrication risk). ✅
- **Q2 (R-A.4-1 ride-along as 4d.0)**: "Small (~50 LOC + tests), dispatcher-internal, prevents Drop 5 fire-drill." Verified via §1.10: ordering violation is real and dispatcher-side. Estimated ~50 LOC is plausible (two functions, ~10-15 lines of code change each + tests for ordering). Brief acknowledges file-overlap risk (low — codex work is in `cli_codex/`; ordering fix is in `monitor.go` + `dispatcher.go`, disjoint). Justification stands. ✅
- **Q5 (Drop 5 readiness gate)**: "4c.5 Themes A+B merged + Drop 4d merged. Codex Tillsyn-MCP self-registration deferred." Cross-checked against `project_drop_4c_5_handoff.md` Q5 — already states 4c.5 A+B sufficient for single-CLI dogfood. Brief's incremental claim (multi-CLI also needs Drop 4d) is the minimum sufficient gate. MCP self-registration deferral is consistent with the YAGNI pattern and L-codex-5. ✅

### 1.12 Droplet count integrity — COVERED

Brief claims 7-9 droplets across 3 phases. Verification:

- Phase 1: 4d.1 (CLIKindCodex const) + 4d.2 (preflight routing) = 2 droplets. ✅
- Phase 2: 4d.3 (skeleton) + 4d.4 (argv) + 4d.5 (env) + 4d.6 (stream) + 4d.7 (render) = 5 droplets. ✅
- Phase 3: 4d.8 (wiring + contract test) + 4d.9 (sample template + DOGFOOD_HANDOFF) = 2 droplets. ✅
- Optional 4d.0 ride-along.
- TOTAL: 9 (with 4d.0) or 8 (without). Falls within "7-9" claim.

Each droplet has explicit acceptance criteria, blocked_by wiring, and file paths — single concern, atomic semantics. Grain is appropriate (~50-300 LOC per droplet based on cli_claude reference impl sizes: argv.go ~5.5k bytes, env.go ~5.2k bytes, stream.go ~9.1k bytes, adapter.go ~4.4k bytes, init.go ~1.6k bytes, render.go ~23k bytes — render largest at ~23k bytes ≈ ~500-600 LOC).

## 2. Findings

No BLOCKER findings. Two NITs:

- **N1 (NIT, NON-BLOCKER): Q1 evidence-strength caveat undersold for protocol_v1 case-form mismatch.** The brief cites Context7 EventMsg evidence (line 83) using snake_case names (`session_configured`, `agent_message`, `task_complete`, etc.), while Context7's actual `protocol_v1.md` returns PascalCase (`TurnStarted`, `TurnComplete`, `AgentMessage`, `Error`, `Warning`). The brief says "per Context7 partial evidence" — accurate but slightly understates the case-form gap. The wire form (`--json` JSONL) MAY use snake_case per Rust's typical serde rename conventions, but this is unverified. **Recommended fix:** in §3.2 droplet 4d.6 description, add a one-line clarification: "Event-type strings are snake_case in the JSONL wire form per recorded fixture (Q1 Path A); the Rust `EventMsg` enum is PascalCase. Wire form is authoritative for stream.go." This is a documentation NIT, not a structural concern — Q1 fixture is already routed as HARD PRECONDITION which surfaces the truth empirically.

- **N2 (NIT, NON-BLOCKER): cli_claude file count brief shorthand.** Brief §11 References list mentions "`cli_claude/{adapter,argv,env,stream,init}.go`" — this is 5 files, but the package root has 6 (adding `adapter_test.go`). Brief is using production-file shorthand consistent with CLI_ADAPTER_AUTHORING.md §1 which separately lists the test file. **Recommended fix:** either (a) leave as-is (matches CLI_ADAPTER_AUTHORING.md convention) or (b) add `adapter_test.go` to the parenthetical for completeness. Either is acceptable. This is purely a stylistic NIT.

## 3. Conclusion

**PASS.** Every concrete claim in the REVISION_BRIEF is backed by direct file/line evidence in the committed tree (Drop 4c shipped state) OR by Context7 / WebSearch evidence for codex CLI semantics. The two cases of evidentiary weakness — codex `exec --json` per-event field shape (§1.6) and `CODEX_HOME` resolution semantics (§1.5) — are explicitly acknowledged in the brief's §9 Q1 / §13 R1 / §13 R5 with appropriate routing (HARD PRECONDITION fixture + planner subdecisions). Brief's risk hedging is grounded.

The droplet decomposition is granular (7-9 droplets), each droplet has acceptance criteria + blocked_by wiring + file paths. Phase 1/2/3 sequencing is sound; Wave structure (§7) preserves parallel-friendly opportunities while respecting blocked_by constraints. The R-A.4-1 ride-along (4d.0) routing is justified by direct file/line verification of the ordering bug at `monitor.go:371→383` and `dispatcher.go:651→655`.

Q1 is correctly identified as the load-bearing precondition. Without the dev fixture, droplet 4d.6 (stream.go) cannot pin event-type strings without fabrication risk. Brief gates planner-spawn on fixture availability, which is the correct hedge.

Two NITs (N1, N2) are documentation polish; neither blocks planner spawn. Brief is ready to proceed to planner spawn after Q1 fixture is captured.

## Hylla Feedback

N/A — review touched only Go files for evidence verification, but per project rule (§"Code Understanding Rules" item 1) Hylla was not invoked because (a) the active filesystem-MD coordination paradigm explicitly bypasses Hylla, (b) Drop 4c.5 is in flight and Hylla may not have re-ingested post-Drop-4c merge, and (c) all needed evidence was directly accessible via `Read` against committed code at known paths cited by the brief. No Hylla misses to report.
