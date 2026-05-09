# Plan-QA-Proof Round 2 — DROP_4c.6.W0

**Reviewed:** `workflow/drop_4c_6/DROP_4c.6.W0_AGENTS_TOML_SCHEMA/PLAN.md` (round-2 revised) + `_BLOCKERS.toml`.
**Round-1 inputs (context):** `PLAN_QA_PROOF.md` (R1 verdict: PASS, 0 findings) + `PLAN_QA_FALSIFICATION.md` (R1 verdict: FAIL, 1 CONFIRMED — FF1: D3↔D5 line-number / separation-of-concerns contradiction).
**Round-2 fix applied:** Option α — reword 5 drift sites so D3 raises a bare sentinel (`ErrToolsDenyNotOverridable`) and D5's `*ConfigError` envelope wraps with file/block/line position. Sites named in the spawn brief: AC bullet 2, AC bullet 3, AC bullet 8, RiskNote on `tools_deny`, ContextBlock `constraint`.
**Reviewer scope:** Half A — verify FF1 fix landed at the 5 named sites + sweep for residual drift inside D3. Half B — fresh 5-axis proof pass per `~/.claude/agents/go-qa-proof-agent.md`.
**Round:** 2.
**Verdict mechanic:** PASS = zero findings across all 5 axes AND FF1 fully fixed. FAIL = ≥1 finding (round-1-residual OR fresh).

## 1. Round-1 Findings Verification

**FF1 status: PARTIAL**

The 5 named reword sites all landed correctly:

- **AC bullet 2 (PLAN.md:111)** — FIXED. Reads "Per § 4.3.1 + § 5: '`tools_deny` is the safety floor; setting it in `.local.toml` fails loud.' TOML-position wrapping is added by D5's envelope; D3 raises the bare sentinel." Explicit D3/D5 separation. ✓
- **AC bullet 3 (PLAN.md:112)** — FIXED. Reads "The bare sentinel's message reads `\"tools_deny is not user-overridable; remove the field\"` (no file/line/block prefix at the D3 boundary). D5 wraps the sentinel into `*ConfigError` so the user-facing message renders as `\"agents.local.toml [agents.<kind>]:<line>: tools_deny is not user-overridable; remove the field\"` (or `[agents]:<line>` for the defaults block)." Bare-sentinel string named explicitly; D5's wrap-format named explicitly. ✓
- **AC bullet 8 (PLAN.md:117)** — FIXED. Was the round-1 FF1 site itself. Reads "Test `TestMergeLocal_ToolsDenyRejected` loads `local_tools_deny_rejected.toml`; asserts `errors.Is(err, ErrToolsDenyNotOverridable)` succeeds against the sentinel returned by `MergeLocal`. (Position-wrapping at the envelope layer is asserted separately by D5's `TestMergeLocal_ToolsDenyPositionWrapped`; this bullet covers only the sentinel-rejection contract D3 owns.)" Line-number clause dropped at D3 boundary; explicit handoff to D5's test for position assertion. ✓
- **RiskNote on `tools_deny` (PLAN.md:128)** — FIXED. Reads "`tools_deny` rejection MUST surface a position-aware error to the user — without TOML-line context, the user gets a hostile 'your file is broken somewhere' message. D3 raises the bare sentinel; D5's envelope adds the file/block/line wrapping. D3's tests assert sentinel-rejection only; D5's tests assert the wrapping." Layer ownership explicit. ✓
- **ContextBlock `constraint` critical (PLAN.md:132)** — FIXED. Reads "`tools_deny` is NEVER user-overridable; setting it in `.local.toml` fails loud via the closed sentinel `ErrToolsDenyNotOverridable` (D3) wrapped into a position-aware `*ConfigError` envelope (D5). Safety floor per `SKETCH.md` § 4.3.1." D3/D5 layer split explicit. ✓

However, the FF1 fix-list missed one residual drift site INSIDE D3's KindPayload — see finding 1.1 below. Because the residual is a contract-internal contradiction with the freshly-reworded AC b3, FF1 is graded PARTIAL, not FIXED.

- 1.1 [Axis: specify-block-well-formedness] [severity: medium] **D3 KindPayload drift residue contradicts freshly-reworded AC b3.** PLAN.md:137 D3 KindPayload contains TWO drift instances inside one structured-contract bullet:
  - First instance — production-file `shape_hint` reads: `"MergeLocal(project, local) deep-merges local registry over project; rejects local tools_deny with sentinel error citing TOML line; reuses D2's mergeMaps helper"` — the phrase "citing TOML line" is exactly the round-1-FF1 contradiction the rewording was meant to remove.
  - Second instance — test-file `shape_hint` reads: `"table-driven; tools_deny test asserts errors.Is(err, ErrToolsDenyNotOverridable) + line-number presence in message"` — the phrase "+ line-number presence in message" assigns the position assertion to D3's test, which directly contradicts AC b8 (which moved that assertion to D5's `TestMergeLocal_ToolsDenyPositionWrapped`).
  - Effect on the builder: KindPayload is the structured-contract sub-field a builder agent reads to scope its edit. A builder reading D3's Specify block sees AC b3 saying "no file/line/block prefix at the D3 boundary" AND KindPayload `shape_hint` saying "citing TOML line" — an internal contradiction at the same level of the contract. The builder either picks one and violates the other, or escalates back. Both outcomes regress the FF1 fix.
  → **Evidence pointer:** PLAN.md:137 D3 KindPayload (sole bullet under D3's `**KindPayload:**` heading); compare to AC b3 PLAN.md:112 + AC b8 PLAN.md:117 + RiskNote PLAN.md:128 + ContextBlock PLAN.md:132 (all 4 freshly-reworded to put the line/position assertion on D5).
  → **Fix hint:** sweep D3's KindPayload one more time. (a) Remove ` citing TOML line` from the production-file `shape_hint` (or rephrase as " + position-wrapping deferred to D5's `*ConfigError` envelope"). (b) Remove `+ line-number presence in message` from the test-file `shape_hint` (or rephrase as " — position assertion lives in D5's `TestMergeLocal_ToolsDenyPositionWrapped`"). Both edits are surgical; FF1's Option α applied to one more site closes the gap. No re-validation of D1/D2/D4/D5 needed.

## 2. Fresh Findings

No fresh findings on the four axes that round 1 cleared (atomic-decomposition, parallelization-graph, multi-level-decomposition, shipped-but-not-wired). The serial chain D1→D2→D3→D4→D5 is unchanged; `_BLOCKERS.toml` mirror unchanged; per-droplet sizing unchanged; consumer-routing unchanged.

The single finding above (1.1) sits on the specify-block-well-formedness axis and is round-1-residual, not fresh. No fresh axis-violations are introduced by the round-2 rewording.

### Per-axis confirmations (round 2)

- **Atomic-decomposition** — five droplets, sizing unchanged from round 1. ✓
- **Parallelization-graph** — serial chain unchanged; `_BLOCKERS.toml` mirror unchanged at lines 15-33. PLAN.md `Blocked by:` bullets at :48 (D1=—), :85 (D2=D1), :121 (D3=D2), :160 (D4=D3), :196 (D5=D4) all match. ✓
- **Specify-block well-formedness** — D1, D2, D4, D5 unchanged from round 1 (PASS). D3 has finding 1.1 (KindPayload drift residue contradicts AC b3 + AC b8). FAIL on this axis pending finding-1.1 resolution.
- **Multi-level-decomposition** — single L2 layer, no L3 sub-drops, unchanged. ✓
- **Shipped-but-not-wired** — symbol-consumer wiring unchanged: D1's types → D2/D3/D5; D2's `Resolve`/`mergeMaps` → D3 + W3; D3's `MergeLocal`/`ErrToolsDenyNotOverridable` → D5 + W2/W3; D4's `StripFrontmatterKeys` → W3; D5's `ConfigError`/`WrapWithPosition` → D1+D3 mods + W3/W11. Out-of-scope § (PLAN.md:226-231) intact. ✓

## 3. Missing Evidence

None. Every claim in this round-2 review traces to a concrete line-number citation in `PLAN.md` (round-2 revised) or to round-1's `PLAN_QA_PROOF.md` / `PLAN_QA_FALSIFICATION.md`. The FF1 site-by-site verification names the 5 reword sites explicitly and quotes the surviving text. The new finding (1.1) names the exact `shape_hint` substrings to remove or rephrase. No future-lookup hedges introduced beyond round 1's existing `pelletier/go-toml/v2` API-shape RiskNotes (which round 2 did not touch and which remain correctly captured).

## 4. Summary

**Verdict: fail**

Finding count: 1 (round-1-residual FF1 partial-fix; medium severity).

Rationale: Round 2 applied Option α to the 5 named drift sites (AC b2, AC b3, AC b8, RiskNote on `tools_deny`, ContextBlock constraint critical) — all 5 landed correctly with explicit D3↔D5 layer separation. However, the fix-list missed D3's KindPayload (PLAN.md:137), which carries TWO surviving drift instances ("citing TOML line" in the production-file `shape_hint` + "line-number presence in message" in the test-file `shape_hint`) that directly contradict the freshly-reworded AC b3 ("no file/line/block prefix at the D3 boundary") and AC b8 (which moved the line-number assertion to D5's `TestMergeLocal_ToolsDenyPositionWrapped`). Because KindPayload is the structured-contract sub-field a builder agent reads, the contradiction is contract-internal, not cosmetic — a builder reading D3's Specify block would see two layers of the same contract giving opposite instructions about whether D3's rejection emits a line number.

The fix is surgical: sweep D3's KindPayload one more time, applying the same Option α rewording (defer line/position language to D5). No round-3 of the broader plan needed; just an editorial pass on one bullet. Once that lands, FF1 converges fully and Round 3 of plan-QA-proof should clear at zero findings.

The four other plan-QA axes (atomic-decomposition, parallelization-graph, multi-level-decomposition, shipped-but-not-wired) all remain PASS as in round 1; the round-2 rewording neither introduced nor exposed any axis-2/4/5 regressions.

## 5. Hylla Feedback

N/A — this round-2 review touched only Markdown / TOML inputs (`PLAN.md`, `_BLOCKERS.toml`, `PLAN_QA_PROOF.md`, `PLAN_QA_FALSIFICATION.md`). Hylla indexes Go files only per project memory (`feedback_hylla_go_only_today.md`); no Go-symbol queries were warranted at the L2 plan-QA-proof scope. No fallbacks logged.
