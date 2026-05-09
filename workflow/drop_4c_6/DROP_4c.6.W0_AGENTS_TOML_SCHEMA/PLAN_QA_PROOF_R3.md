# Plan-QA-Proof Round 3 — DROP_4c.6.W0

**Reviewed:** `workflow/drop_4c_6/DROP_4c.6.W0_AGENTS_TOML_SCHEMA/PLAN.md` (round-3 revised) + `_BLOCKERS.toml`.
**Round-2 inputs (context):** `PLAN_QA_PROOF_R2.md` (R2 verdict: FAIL, 1 finding — W0-FF2-R2: D3 KindPayload at PLAN.md:137 retained line-number language) + `PLAN_QA_FALSIFICATION_R2.md` (R2 verdict: FAIL, 1 CONFIRMED — same family, surgical KindPayload drift).
**Round-3 fix applied:** Option α (continued) — 2 verbatim edits to D3 KindPayload `shape_hint` strings at PLAN.md:137:
- Production `shape_hint` for `MergeLocal, ErrToolsDenyNotOverridable`: `"; rejects local tools_deny with sentinel error citing TOML line; reuses D2's mergeMaps helper"` → `"; rejects local tools_deny by returning bare ErrToolsDenyNotOverridable sentinel (D5 wraps with file/line/block); reuses D2's mergeMaps helper"`.
- Test `shape_hint` for `TestMergeLocal_*` test list: `"+ line-number presence in message"` → `"; line-number wrapping asserted separately by D5's TestMergeLocal_ToolsDenyPositionWrapped"`.
**Reviewer scope:** Half A — verify W0-FF2-R2 fix landed at the named site + sweep D3 for any residual D3-owned line-number language. Half B — fresh 5-axis proof pass per `~/.claude/agents/go-qa-proof-agent.md`.
**Round:** 3.
**Verdict mechanic:** PASS = zero findings across all 5 axes AND W0-FF2-R2 fully fixed. FAIL = ≥1 finding (round-2-residual OR fresh).

## 1. Round-2 Findings Verification

**W0-FF2-R2 status: FIXED**

The R2 falsification's two verbatim replacement strings both landed at PLAN.md:137:

- **Sub-clause (a) — production-file `shape_hint`** — FIXED. Round-3 reads (PLAN.md:137, first `changes[]` entry):
  > `"MergeLocal(project, local) deep-merges local registry over project; rejects local tools_deny by returning bare ErrToolsDenyNotOverridable sentinel (D5 wraps with file/line/block); reuses D2's mergeMaps helper"`
  The R2-flagged phrase `"; rejects local tools_deny with sentinel error citing TOML line"` is gone. The replacement matches the falsification's exact suggested text. The clause `"(D5 wraps with file/line/block)"` makes the layer attribution explicit at the structured-contract surface — symmetric with AC b2 (PLAN.md:111) and AC b3 (PLAN.md:112). ✓

- **Sub-clause (b) — test-file `shape_hint`** — FIXED. Round-3 reads (PLAN.md:137, second `changes[]` entry):
  > `"table-driven; tools_deny test asserts errors.Is(err, ErrToolsDenyNotOverridable); line-number wrapping asserted separately by D5's TestMergeLocal_ToolsDenyPositionWrapped"`
  The R2-flagged phrase `"+ line-number presence in message"` is gone. The replacement names D5's specific test (`TestMergeLocal_ToolsDenyPositionWrapped`) verbatim — symmetric with AC b8 (PLAN.md:117) which made the same handoff in prose. ✓

**Residual-drift sweep (Half A continuation):** I re-read all of D3 (PLAN.md:103-137) plus the cross-cutting D3 references inside D5 (PLAN.md:188-194) and confirmed no surviving D3-owned line/file/position claim. The clean sites:

| D3 surface | Line | Status | Position-attribution layer |
| --- | --- | --- | --- |
| AC b2 (sentinel rejection contract) | 111 | clean | "wrapping is added by D5's envelope" |
| AC b3 (sentinel message shape) | 112 | clean | "no file/line/block prefix at the D3 boundary" + D5 wrap format named |
| AC b8 (`TestMergeLocal_ToolsDenyRejected` expectation) | 117 | clean | "Position-wrapping at the envelope layer is asserted separately by D5's `TestMergeLocal_ToolsDenyPositionWrapped`" |
| RiskNote on `tools_deny` | 128 | clean | "D3 raises the bare sentinel; D5's envelope adds the file/block/line wrapping" |
| ContextBlock `constraint` critical | 132 | clean | "(D3) wrapped into a position-aware `*ConfigError` envelope (D5)" |
| KindPayload production `shape_hint` (sub-clause a) | 137 | clean (this round) | "(D5 wraps with file/line/block)" |
| KindPayload test `shape_hint` (sub-clause b) | 137 | clean (this round) | "asserted separately by D5's TestMergeLocal_ToolsDenyPositionWrapped" |
| KindPayload `local_tools_deny_rejected.toml` fixture `shape_hint` | 137 | clean | `"local sets [agents.build].tools_deny — must reject"` (no position claim) |
| D3 Objective / AcceptanceCriteria / ValidationPlan | 123-125 | clean | no D3-owned position assertion |
| ContextBlocks 2-5 (ordering, field-merge, sketch ref, sentinel decision) | 133-136 | clean | no position claim |
| D5 references back to D3 (RiskNote on `MergeLocal` not parsing TOML) | 203 | clean | correctly observes line numbers must come from D1's load step |

W0-FF2-R2 graded FIXED, not PARTIAL. Both sub-clauses propagated; no residual D3-owned position claim survived the sweep.

## 2. Fresh Findings

No fresh findings.

### Per-axis confirmations (round 3)

- **Atomic-decomposition** — five droplets, sizing unchanged from round 2. Each droplet still maps to one production file edit + co-located tests + one or more golden fixtures, ~80-120 LOC each per `~/.claude/agents/go-planning-agent.md` § "Atomic droplet sizing." ✓
- **Parallelization-graph** — serial chain D1→D2→D3→D4→D5 unchanged. PLAN.md `Blocked by:` bullets at :48 (D1=—), :85 (D2=D1), :121 (D3=D2), :160 (D4=D3), :196 (D5=D4) all consistent. `_BLOCKERS.toml` mirror unchanged from R2 (mirror was PASS in R2). Package-shared serialization on `internal/config` correctly enforced per planning-agent HARD RULES. ✓
- **Specify-block well-formedness** — D1, D2, D4, D5 unchanged from R2 PASS. D3 round-2-residual W0-FF2-R2 closed by round-3 KindPayload edits. All 12 D3 acceptance bullets, both RiskNotes blocks, all 5 ContextBlocks, and the KindPayload JSON internally consistent at this round. ✓
- **Multi-level-decomposition** — single L2 layer, no L3 sub-drops; consistent with the parent L1 W0 row at `workflow/drop_4c_6/PLAN.md:51-67`. ✓
- **Shipped-but-not-wired** — symbol-consumer wiring unchanged: D1's `Preset`/`Override`/`AgentRuntime`/`AgentsRegistry`/`Kind` consumed by D2 (`Resolve`), D3 (`MergeLocal`), D5 (`ConfigError`). D2's `Resolve`/`mergeMaps` consumed by D3 + W3. D3's `MergeLocal`/`ErrToolsDenyNotOverridable` consumed by D5 + W2/W3. D4's `StripFrontmatterKeys` consumed by W3 (PLAN.md:170, :230). D5's `ConfigError`/`WrapWithPosition` consumed by W3 + W11 (PLAN.md:231). Out-of-scope § at PLAN.md:226-231 names every downstream wave consumer. ✓

## 3. Missing Evidence

None. Every claim in this round-3 review traces to a concrete line-number citation in `PLAN.md` (round-3 revised) or to round-2's `PLAN_QA_PROOF_R2.md` / `PLAN_QA_FALSIFICATION_R2.md`. The FIXED grading on W0-FF2-R2 is backed by a verbatim quote of both round-3 `shape_hint` strings against the falsification's exact suggested replacement text. The residual-drift sweep enumerates 11 D3 surfaces (lines 111-137) plus 1 cross-cutting D5 surface (line 203) — every one cited and verified clean.

No future-lookup hedges introduced beyond the existing D1 / D5 `pelletier/go-toml/v2` API-shape RiskNotes, which round 3 did not touch and which remain correctly captured (builder verifies via `go doc` before authoring per `CLAUDE.md` § Tool Discipline — those RiskNotes are the right shape for "lib-API discovery deferred to build time").

## 4. Summary

**Verdict: pass**

Finding count: 0.

Rationale: Round 3 applied the falsification's two verbatim replacement strings to D3's KindPayload at PLAN.md:137; both sub-clauses landed exactly as suggested. The R2-flagged phrases (`"sentinel error citing TOML line"`, `"+ line-number presence in message"`) are both removed; the replacement clauses (`"(D5 wraps with file/line/block)"`, `"asserted separately by D5's TestMergeLocal_ToolsDenyPositionWrapped"`) make D3↔D5 layer attribution explicit at the structured-contract surface, symmetric with the round-2 prose reword at AC b2/b3/b8 + RiskNote + ContextBlock. The residual-drift sweep confirms no other D3 surface retains a line/file/position claim D3 has no infrastructure to back. The four other axes (atomic-decomposition, parallelization-graph, multi-level-decomposition, shipped-but-not-wired) remain PASS unchanged from R1/R2.

W0-FF2-R2 is the only finding round 2 left open; it converged to FIXED here. The plan is now ready for orchestrator-side gate-clear and W0 build dispatch (D1 first, serial chain through D5).

## 5. Hylla Feedback

N/A — this round-3 review touched only Markdown / TOML inputs (`PLAN.md`, `_BLOCKERS.toml`, `PLAN_QA_PROOF_R2.md`, `PLAN_QA_FALSIFICATION_R2.md`). Hylla indexes Go files only per project memory (`feedback_hylla_go_only_today.md`); no Go-symbol queries were warranted at the L2 plan-QA-proof scope. No fallbacks logged.
