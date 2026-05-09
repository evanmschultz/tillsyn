# Plan-QA-Falsification Round 2 — Drop 4c.6

**Reviewer:** plan-QA-falsification subagent (Round 2)
**Authored:** 2026-05-09
**Plan under attack:** `workflow/drop_4c_6/PLAN.md` REVISED Round 2 (commit `95ebe58`).
**Sketch source-of-truth (locked, NOT under attack):** `workflow/drop_4c_6/SKETCH.md` v2.8.4 POST-QA FINAL.
**Round 1 verdict:** FAIL — 7 CONFIRMED counterexamples (C2.1-C2.7) + 1 low borderline (A6).
**Working dir:** `/Users/evanschultz/Documents/Code/hylla/tillsyn/main`.
**HEAD at review:** `95ebe58`.

## 1. Round-1 Counterexamples Verification

| # | Round-1 finding | Round-2 status | Evidence |
| --- | --- | --- | --- |
| C2.1 | W1.D1 ↔ W5.D1 race on `embed.go` + `embed_test.go` | **FIXED** | `_BLOCKERS.toml` lines 42-45 (`4c.6.W5.D1 blocked_by ["4c.6.W1.D1"]`); PLAN.md line 178 inline `Blocked by: 4c.6.W1.D1` for W5.D1 with HF1 rationale. |
| C2.2 | W1.D1 ↔ W5.D2 transitive race | **FIXED** | `_BLOCKERS.toml` lines 47-50 (`4c.6.W5.D2 blocked_by ["4c.6.W5.D1"]`); transitive chain to W1.D1 via W5.D1 explicit. PLAN.md line 209. |
| C2.3 | `internal/templates` package serialization gap (W0.5 ↔ W1.D1 ↔ W5.{D1,D2,D3}) | **FIXED** | New chain W0.5 → W1.D1 → W5.D1 → W5.D2 → W5.D3 enforced in `_BLOCKERS.toml` rows + PLAN.md inline blockers (lines 101, 178, 209, 233). Graph summary at PLAN.md lines 361-378 documents the chain. |
| C2.4 | W3 vs W6.D4 SPAWN_PIPELINE.md:24-31 ownership overlap | **FIXED** | PLAN.md line 148 explicit `**SPAWN_PIPELINE.md:24-31` rewrite is OUT OF SCOPE for W3** — sole owner is W6.D4`. PLAN.md line 153 (L2 sub-planner directive) confirms the strip from D6 scope. `_BLOCKERS.toml` lines 62-65 reason-string updated to "W6.D4 is SOLE OWNER of SPAWN_PIPELINE.md:24-31". |
| C2.5 | W5.D1 acceptance bullet 4 vs bullet 5 contradiction (`git grep "default-go.toml"` zero hits vs retained historical comments) | **FIXED** | PLAN.md line 167 reworded to "zero hits in **non-doc-comment locations** (string literals, embed directives, switch cases, `BuiltinTemplateNames` literal entries)". W5.D2 mirror at line 207. Phrasing is now grep-able + contradiction-free. |
| C2.6 | W5.D1 caller-audit drift (over-claimed `template_service.go`; omitted `mcp_surface.go`, `auto_generate_steward_test.go`, `service_test.go`) | **PARTIAL — see C2.6.R2 below** | PLAN.md L1 header (line 5) and W5.D1 line 168-176 caller-audit list ARE corrected (5 sites verified against `git grep "default-go.toml" cmd/ internal/` against HEAD `95ebe58`: `service.go:383`, `service_test.go:7-hit cluster including :6534`, `auto_generate_steward_test.go:18`, `mcp_surface.go:906`, `extended_tools.go:1867`). HOWEVER the W5.D1 inline `**Paths:**` field at line 161 STILL names `internal/app/auto_generate_steward.go` (non-test, zero refs) and `internal/app/template_service.go` (zero refs) — both files were verified by line 176 to be over-claimed and removed. The fix landed in the audit-detail bullet list (line 168-176) but did NOT propagate to the path summary at line 161. See fresh counterexample 2.1 below. |
| C2.7 | W3 path list omits `cli_adapter.go` where `BindingResolved` is defined | **FIXED** | PLAN.md L1 header (line 5) now includes `internal/app/dispatcher/cli_adapter.go (BindingResolved struct definition site — ROUND-2 HF4 verified)` and `internal/app/dispatcher/cli_adapter_test.go`. W3 Scope (line 141) carries the HF4 callout. L2 sub-planner directive (line 153) includes the cli_adapter.go path explicitly under D1. Verified via `git grep -n "type BindingResolved" internal/` returning ONLY `internal/app/dispatcher/cli_adapter.go:102` against HEAD. |
| A6 | W1.D1 ~24-file footprint | **KEPT AS-AUTHORED (per dev disposition)** | PLAN.md W1.D1 file count unchanged. Mechanical-bundle defensibility argument retained. NOT actioned per Round-1's low-severity classification + dev decision. |

**Net Round-1 fix status:** 6 of 7 counterexamples FULLY fixed; 1 (C2.6) PARTIALLY fixed — the audit-detail list is correct but the W5.D1 inline `**Paths:**` summary at line 161 retained the stale over-claimed entries. Captured below as fresh counterexample 2.1.

## 2. Fresh Counterexamples

- 2.1 [Family: hidden-coupling] [severity: medium] **W5.D1 inline `**Paths:**` field at PLAN.md line 161 contradicts the corrected caller-audit list at lines 168-176 + the HF6 removal note at line 176.** Line 161 reads: *"plus caller-audit edits (string literal updates only) at `internal/app/auto_generate_steward.go`, `internal/app/service.go`, `internal/app/template_service.go`, `internal/adapters/server/mcpapi/extended_tools.go`"* — naming `auto_generate_steward.go` (non-test, ZERO refs verified via `git grep -c "default-go.toml" internal/app/auto_generate_steward.go`) and `template_service.go` (ZERO refs verified). Line 176 explicitly states: *"Removed from caller-audit list (ROUND-2 HF6): `internal/app/auto_generate_steward.go` (zero `default-go.toml` refs verified via `git grep`), `internal/app/template_service.go` (zero refs verified). Both were over-claimed in Round-1 PLAN.md."* Two competing path declarations within the same droplet contract. → repro: a builder spawned against W5.D1 reads line 161 (the canonical `**Paths:**` summary, the field the post-Drop-1 paths-discipline gate inspects) and either (a) dutifully edits the two files that have nothing to edit (no-op-but-confusing churn), or (b) recognizes the contradiction with line 176 and asks the orchestrator which is canonical (planning-loop overhead). The fix that landed at L1 header (line 5) was applied correctly — line 5 lists ONLY the post-fix five caller-audit sites and explicitly notes "replaces over-claimed `template_service.go`." The W5.D1 inline `**Paths:**` summary (line 161) was simply not regenerated to match. → fix_hint: edit PLAN.md line 161 to read *"plus caller-audit edits (string literal updates only) at `internal/app/service.go`, `internal/app/service_test.go`, `internal/app/auto_generate_steward_test.go`, `internal/adapters/server/common/mcp_surface.go`, `internal/adapters/server/mcpapi/extended_tools.go`"* (the same five sites named in lines 168-174 + L1 header line 5). The "12 source files, ~30 string edits" parenthetical is a citation to `RESEARCH/CASCADE_ENFORCEMENT_AND_CONTEXT_PRELOAD.md §C` and may stay as historical reference, but the explicit file list naming the over-claimed pair must go. CONFIRMED — concrete repro, single-spot fix, doesn't need re-respawn of any sub-planner.

- 2.2 [Family: contract-mismatch] [severity: low] **W5.D1 acceptance bullet for `internal/templates/embed.go` lists "5 hits including the `//go:embed` directive at :34, switch case at :138, plus historical doc-comments at :17, :62, :106"** (PLAN.md line 174). Verified against HEAD `95ebe58` via `git grep -n "default-go.toml" internal/templates/embed.go`: actual hits are at lines 17, 34, 62, 106, 138 (5 total — matches the count). However, line 174's parenthetical claims "5 hits" but PLAN.md L1 header (line 5) line and earlier audit material independently states the rename touches `embed.go` directly (the `//go:embed` directive is the W5.D1-defining edit, not a caller-audit). The 3 historical doc-comments at :17, :62, :106 are within `internal/templates/embed.go` itself — already in W5.D1's scope by virtue of being IN `internal/templates`. The W5.D1 acceptance bullet bullet's classification "doc-comments classified as historical (Drop 4c.5 F.2.1 rebadge history) are RETAINED verbatim" gives the right rule, but pairing this with bullet 4's "zero hits in non-doc-comment locations" creates the question: is line 17 (`internal/templates/embed.go:17:// `default-go.toml` so sibling builtins...`) a forward-looking doc-comment (should be UPDATED to `till-go.toml`) or a historical rename-history doc-comment (RETAINED)? `embed.go:17` reads in context: "_renamed from `default.toml` to `default-go.toml` ... so sibling builtins (and, post-Q1 resolution, future ...)_" — this is DUAL-history (records past rename + uses the name forward-looking). The plan's RiskNotes line 183 acknowledges this: *"Drop 4c.5 F.2.1 doc-comment in `embed.go` LINES 16-23 references 'rebadged from `default.toml` to `default-go.toml`' — update those to record the second rebadge 'to `till-go.toml`' per dual-history note."* Good — the plan does pin the dual-history disposition. NOT a CONFIRMED counterexample because the plan does carry the disposition explicitly in RiskNotes; just observing that the acceptance bullet 4 + bullet 5 + RiskNotes pin this together by triangulation rather than at a single load-bearing spot. REFUTED but flagged as fragile.

**Cycle check (REQUIRED Phase-2-step-1 attack):** walked every `blocked_by` edge in the new total order:
- W0.5 → ∅ (Wave A head)
- W0 → ∅ (Wave A head)
- W6.D2 → ∅ (Wave A head)
- W6.D3 → ∅ (Wave A head)
- W1.D1 → W0.5
- W2 → W1.D1
- W3 → W1.D1, W0
- W6.D1 → W0
- W5.D1 → W1.D1
- W5.D2 → W5.D1
- W5.D3 → W5.D1, W5.D2, W1.D1
- W6.D4 → W3
- W6.D5 → W6.D1, W6.D2, W6.D3

Topological sort terminates: {W0, W0.5, W6.D2, W6.D3} → {W1.D1, W6.D1} → {W2, W3, W5.D1} → {W5.D2, W6.D4} → {W5.D3, W6.D5}. No cycle. The longest path (W0.5 → W1.D1 → W5.D1 → W5.D2 → W5.D3) is 5 hops — matches PLAN.md line 378's claimed wall-clock bottleneck. REFUTED.

**`_BLOCKERS.toml` / PLAN.md drift (REQUIRED Phase-2-step-1 attack):** every `[[blockers]]` row matches an inline `Blocked by:` bullet in PLAN.md and vice versa:
- `_BLOCKERS.toml` Wave-A heads (W0, W0.5, W6.D2, W6.D3) match PLAN.md lines 65, 83, 280, 300.
- `_BLOCKERS.toml` lines 27-30 (W1.D1 → W0.5) match PLAN.md line 101.
- `_BLOCKERS.toml` lines 32-35 (W2 → W1.D1) match PLAN.md line 130.
- `_BLOCKERS.toml` lines 37-40 (W3 → W1.D1, W0) match PLAN.md line 151.
- `_BLOCKERS.toml` lines 42-45 (W5.D1 → W1.D1) match PLAN.md line 178.
- `_BLOCKERS.toml` lines 47-50 (W5.D2 → W5.D1) match PLAN.md line 209.
- `_BLOCKERS.toml` lines 52-55 (W5.D3 → W5.D1, W5.D2, W1.D1) match PLAN.md line 233.
- `_BLOCKERS.toml` lines 57-60 (W6.D1 → W0) match PLAN.md line 258.
- `_BLOCKERS.toml` lines 62-65 (W6.D4 → W3) match PLAN.md line 320.
- `_BLOCKERS.toml` lines 67-70 (W6.D5 → W6.D1, W6.D2, W6.D3) match PLAN.md line 340.

Zero drift. REFUTED.

**Sibling overlap without `blocked_by` (REQUIRED Phase-2-step-1 attack):** revisited every (paths, packages) pair across the 13 immediate L1 children:
- All `internal/templates`-touching siblings now serialize via the chain W0.5 → W1.D1 → W5.D1 → W5.D2 → W5.D3 (C2.1+C2.2+C2.3 fixes).
- W0 (`internal/config`) does NOT touch `internal/templates` — `internal/config` is NEW package per L1 paths line 6. No overlap with W0.5 or other `internal/templates` work. REFUTED.
- W3 (`internal/app/dispatcher/cli_claude/render`, `internal/app/dispatcher`, `internal/app/dispatcher/cli_claude`) does NOT share files with W2 (`cmd/till`, `internal/vendor/...`) or W6.{D1,D2,D3,D4,D5} (markdown-only). REFUTED.
- W6.D1 (`AGENTS_CONFIG.md` + `README.md` pointer added by W6.D5) — W6.D1 modifies AGENTS_CONFIG.md only; W6.D5 modifies README.md. No file overlap with W6.D1. The W6.D1 path "single pointer link added — touches a different droplet, see W6.D5" (line 252) is hand-off-style — pointer addition is W6.D5 alone. REFUTED.
- W6.D5 → W6.D1 + W6.D2 + W6.D3 chain is correct: W6.D5 cites docs that must exist (line 340).
- W2 (`cmd/till/main.go`, `cmd/till/help.go`, `cmd/till/init_cmd.go` NEW, `cmd/till/main_test.go`, `cmd/till/init_cmd_test.go` NEW) does NOT share `cmd/till` files with any other droplet. REFUTED.

REFUTED — no missing sibling blocker.

## 3. Summary

**Verdict: fail** (1 CONFIRMED counterexample — single-spot fix needed before plan acceptance).

**Counterexample count:** 1 medium (2.1) + 1 low-flagged-but-REFUTED (2.2 — fragile but covered).

**Round-1 fix net:** 6/7 fully fixed; 1 partially fixed (C2.6, leaving 2.1 above as the residue).

**Rationale:** Round 2 of the plan resolves the substantive Round-1 findings — the `internal/templates` package serialization chain is correctly enforced (C2.1, C2.2, C2.3), the SPAWN_PIPELINE.md ownership cut is unambiguous (C2.4), the W5.D1 acceptance grep-bullet is now grep-able + non-contradictory (C2.5), the HF6 caller-audit list at the L1 header (line 5) and the W5.D1 audit-detail bullet block (lines 168-176) match `git grep` reality at HEAD `95ebe58`, and the W3 path list now correctly includes `cli_adapter.go` (C2.7). The cycle check, `_BLOCKERS.toml` drift check, and sibling-overlap-without-blocked_by check are all clean. The single residual counterexample (2.1) is a localized inconsistency in the W5.D1 inline `**Paths:**` summary at line 161 — the path list there names two files that line 176 explicitly removes as over-claimed — and it's a one-line edit to fix. Recommend: route 2.1 to a Round-2 planner touch-up edit (no full re-spawn needed); rerun plan-QA after the edit.

| Family | Result |
| --- | --- |
| A1 concurrency-blocked_by | **REFUTED** — Round-1 high-severity findings (1.1/1.2/1.3/1.4) all FIXED; new total order verified acyclic; package-overlap chain correct. |
| A2 contract-mismatch | **REFUTED** — HF5 grep-bullet phrasing fixed; HF3 strip removed dangling W3 references; W5.D1 acceptance bullets cohere via RiskNotes triangulation (2.2 fragile but not CONFIRMED). |
| A3 hidden-coupling | **CONFIRMED** (2.1 medium — W5.D1 line 161 inline `**Paths:**` field still names two over-claimed files contradicting line 176's removal note); HF4 cli_adapter.go addition complete. |
| A4 yagni-scope-creep | **EXHAUSTED, no counterexample found** — Round-2 edits added clarifying text only; no new scope crept into W1.D1 / W3 / W5 / W6 droplets vs sketch §25.1 wave breakdown. |
| A5 shipped-but-not-wired | **EXHAUSTED, no counterexample found** — HF8 wiring contract for W3 post-render validator now reads "MUST be wired into render.Render's exit path... NOT shipped as an unwired exported helper" (PLAN.md line 147 + L2 sub-planner directive line 153). W0.5 known-wired-set deferral correctly Open-Questioned (line 420 — RESOLVED 2026-05-09 empty-set-as-authored). |
| A6 atomicity | **EXHAUSTED-low** — W1.D1 24-file count KEPT as-authored per dev disposition; mechanical-bundle defensibility argument is documented in PLAN.md line 41. No new over- or under-decomposition introduced by Round-2 edits. |
| A7 prompt-injection | **EXHAUSTED, no counterexample found** — dormant pre-team-feature per `feedback_prompt_injection_team.md`. |

## 4. Hylla Feedback

N/A — review touched non-Go artifacts only (PLAN.md, `_BLOCKERS.toml`, SKETCH.md, WORKFLOW.md, PLAN_QA_PROOF.md, PLAN_QA_FALSIFICATION.md). The Go-source verifications (`git grep` against `default-go.toml`, `default-generic.toml`, `type BindingResolved`, the `render.go` symbol citations) used `git grep` on the active worktree at HEAD `95ebe58` rather than Hylla. Hylla was not consulted for this round because all attacks targeted PLAN.md/`_BLOCKERS.toml` (markdown + TOML) and the Go-side verifications were single-grep-then-line-number-check — within `git grep`'s sweet spot. No misses; no fallbacks; no ergonomic gripes.
