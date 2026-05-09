# Plan-QA-Proof Round 3 — Drop 4c.6

**Reviewer:** plan-QA-proof subagent (Round 3)
**Authored:** 2026-05-09
**Plan under review:** `workflow/drop_4c_6/PLAN.md` REVISED Round 3 (commit `774df9f`).
**Sketch source-of-truth (locked, NOT re-QA'd):** `workflow/drop_4c_6/SKETCH.md` v2.8.4 POST-QA FINAL.
**Round 2 verdicts (durable, NOT re-reviewed):** `PLAN_QA_PROOF_R2.md` (PASS, 0 fresh findings), `PLAN_QA_FALSIFICATION_R2.md` (FAIL, 1 confirmed counterexample FF1 `2.1`).
**Working dir:** `/Users/evanschultz/Documents/Code/hylla/tillsyn/main`.
**HEAD at review:** `774df9f`.
**Scope:** Half A — verify FF1 (`2.1` in PLAN_QA_FALSIFICATION_R2.md) actually fixed in revised PLAN.md. Half B — fresh 5-axis proof pass on the revised plan to catch any defects introduced by the Round-3 edit. Plus orch-deferred U2 disposition (PLAN.md line 174 prose drift).

## 1. Round-2 Findings Verification

- 1.1 **FF1 (Round-2 falsification `2.1`, hidden-coupling, medium) — W5.D1 inline `**Paths:**` field at PLAN.md line 161 contradicts the corrected caller-audit list at lines 168-176.** **FIXED.** Evidence pointers:
  - `git show 774df9f --stat` confirms the Round-3 commit modifies exactly one file (`workflow/drop_4c_6/PLAN.md`) with one insertion + one deletion — single-line edit, scope is precisely the FF1 spot.
  - `git show 774df9f` diff confirms the line-161 path summary was rewritten: removed `internal/app/auto_generate_steward.go` and `internal/app/template_service.go`; added `internal/app/service_test.go`, `internal/app/auto_generate_steward_test.go`, `internal/adapters/server/common/mcp_surface.go`. The new line 161 names the same 5 sites that the audit-detail bullets at lines 169-173 enumerate (`service.go:383`, `service_test.go:6534`+, `auto_generate_steward_test.go:18`, `mcp_surface.go:906`, `extended_tools.go:1867`) — full triple-consistency now holds across L1 header (line 5) ↔ inline path summary (line 161) ↔ audit-detail bullet block (lines 168-176).
  - Independent ground-truth verification at HEAD `774df9f` via `git grep -n "default-go.toml" cmd/ internal/` (excluding workflow MD): 5 caller-audit sites in non-`internal/templates` packages — `internal/adapters/server/common/mcp_surface.go:906`, `internal/adapters/server/mcpapi/extended_tools.go:1867`, `internal/app/auto_generate_steward_test.go:18`, `internal/app/service.go:383`, `internal/app/service_test.go` (7 hits including the load-bearing `filepath.Join` literal at :6534). Both `internal/app/auto_generate_steward.go` and `internal/app/template_service.go` return ZERO hits — confirmed via `git grep -c "default-go.toml" internal/templates/embed.go internal/app/auto_generate_steward.go internal/app/template_service.go` returning only `internal/templates/embed.go:5` (no rows for the other two = grep exit 0 with no match).
  - Round-3 edit also extended line 161's parenthetical with an audit-trail annotation (`+ ROUND-2 HF6 regenerated audit ... over-claimed internal/app/auto_generate_steward.go and internal/app/template_service.go REMOVED — both have zero default-go.toml refs at HEAD`) which makes the fix self-documenting. The "12 source files, ~30 string edits" historical citation to `RESEARCH/CASCADE_ENFORCEMENT_AND_CONTEXT_PRELOAD.md §C` was retained as historical reference only — exactly the disposition recommended in PLAN_QA_FALSIFICATION_R2.md `2.1` `fix_hint`.
  - Net: the FF1 fix landed exactly where the falsification asked for it, with the exact replacement file list the falsification's `fix_hint` named, plus a self-documenting audit-trail annotation. No collateral edits, no over-correction.

## 2. Fresh Findings

Fresh 5-axis proof pass on the Round-3-revised plan looking for defects introduced by the one-line Round-3 edit.

- 2.1 [Axis: atomic-decomposition] — exhausted, no fresh finding. The Round-3 edit modified a single line within W5.D1's existing Specify block (line 161 path summary). No droplet count change, no scope change to W5.D1's atomic boundary, no spillover to W5.D2 / W5.D3 droplet shapes. W5.D1 still meets atomic-droplet sizing per till-go template (1-4 code blocks: `git mv` + `embed.go` directive + `embed.go` switch case + caller-audit string-edits across 5 files). All other droplets (W0, W0.5, W1.D1, W2, W3, W6.D1-D5) untouched by Round 3.

- 2.2 [Axis: parallelization-graph] — exhausted, no fresh finding. The Round-3 edit did NOT touch any `Blocked by:` line, did NOT touch `_BLOCKERS.toml`, did NOT change PLAN.md's wave-ordering / graph-summary sections (lines 355-378). The total order on `internal/templates` (W0.5 → W1.D1 → W5.D1 → W5.D2 → W5.D3) is unchanged from Round 2 and verified acyclic in PLAN_QA_FALSIFICATION_R2.md cycle-check. Independent re-walk at Round 3: no new edges introduced, longest path still 5 hops, sibling-overlap detection still clean.

- 2.3 [Axis: specify-block-well-formedness] — exhausted, no fresh finding. The Round-3 edit kept W5.D1's Specify block intact (Objective, AcceptanceCriteria, ValidationPlan, RiskNotes, ContextBlocks, KindPayload all unchanged). The line-161 path summary now matches lines 168-176 audit-detail block AND L1 header line 5 — three-way consistency restored. The KindPayload at line 189 still says `"<4 caller audit sites>"` placeholder — pre-existing wording from Round 1 / Round 2 that was NOT a finding in either Round 1 or Round 2 falsification (and is not a Round-3 introduction); flagging only as ergonomic observation: the placeholder count "4" does NOT match the now-canonical 5 sites in line 161 / 168-173 / line 5. Borderline — KindPayload is a planner-emitted shape hint for the dispatcher, not an acceptance bullet, and the dispatcher reads `Paths:` (line 161) as the canonical source not the KindPayload free-text. NOT a CONFIRMED fresh finding because it predates Round 3 and was not flagged in either Round-1 or Round-2 verdicts; recording as 2.3.observation only. **fix_hint (optional, defer-able):** if the planner re-touches this region for any future round, change `<4 caller audit sites>` to `<5 caller audit sites>` for hygiene; not blocking.

- 2.4 [Axis: multi-level-decomposition] — exhausted, no fresh finding. Round 3 did NOT touch any L2 sub-planner directive (W2 line 132, W3 line 153, W0 line 73, W0.5 line 91 — unchanged). The W3 SPAWN_PIPELINE.md ownership cut (5 mutually-consistent locations from Round 2's HF3 fix) is intact. No multi-level decomposition defect introduced by the one-line Round-3 edit.

- 2.5 [Axis: shipped-but-not-wired] — exhausted, no fresh finding. The W3 post-render validator wiring contract at line 147 (Round-2 HF8 fix — "wired into render.Render's exit path ... NOT shipped as an unwired exported helper") is untouched by Round 3. W0.5 known-wired-set deferral at line 420 (RESOLVED 2026-05-09 empty-set-as-authored) untouched. No new shipped-but-not-wired risk.

## 3. Missing Evidence

- 3.1 **Orch-prompt U2 disposition.** The orchestrator-supplied prompt described U2 as: *"PLAN.md line 174's W5.D1 RiskNotes prose says `internal/templates/embed.go` has '5 hits including … historical doc-comments at :17, :62, :106' of `default-go.toml`; at HEAD, `git grep -c "default-go.toml" internal/templates/embed.go` returns 6, not 5."* I re-verified independently:
  - `git grep -c "default-go.toml" internal/templates/embed.go` at HEAD `774df9f` returns **`internal/templates/embed.go:5`** — count is 5, NOT 6. (Verified twice; `grep -c` was sandbox-blocked; `git grep -c` succeeded.)
  - The 5 hits enumerate to lines 17, 34, 62, 106, 138 per `git grep -n "default-go.toml" internal/templates/embed.go` against HEAD `774df9f`.
  - PLAN.md line 174 reads: *"plus historical doc-comments at :17, :62, :106) + `internal/templates/embed_test.go` ..."* — the parenthetical inside line 174 enumerates 5 hits in `embed.go`: the `//go:embed` directive at :34, the switch case at :138, plus the doc-comments at :17, :62, :106. The count and line-number set EXACTLY matches HEAD.
  - PLAN_QA_FALSIFICATION_R2.md `2.2` independently verified the same count against earlier HEAD `95ebe58` and concluded "5 total — matches the count" (REFUTED, no counterexample).
  - **Disposition:** the orch-prompt's U2 framing is incorrect. PLAN.md line 174 is accurate at HEAD `774df9f`. NOT a Round-3 finding. Routing this back to the orchestrator: if the orch was reading from a different working-copy or used a different command (e.g. plain `grep -c` recursive-with-context vs `git grep -c` exact-pattern), there may be a tooling / local-ambiguity issue worth recording in the drop's HYLLA_FEEDBACK; substantively, no plan defect exists.

- 3.2 **Round-2 missing-evidence routes 2.1-2.4 from PLAN_QA_PROOF.md (Round-1) remain L2-deferred.** Untouched by Round 3 (the Round-3 edit was a one-line W5.D1 path-summary fix; no L2-decomposition delegation changes). These were already documented as L2 sub-planner decisions in Round 2 and remain so:
  - 3.2.1 W2 `cmd/till/help_test.go` test scope → L2 W2 sub-planner.
  - 3.2.2 W3 sentinel-style integration test file location → L2 W3 sub-planner.
  - 3.2.3 W0 frontmatter strip helper package home → L2 W0 sub-planner.
  - 3.2.4 W0.5 validator ordering → L2 W0.5 sub-planner.
  No fresh missing-evidence introduced by Round 3.

## 4. Summary

**Verdict: pass**
**Round-2 finding fixed: 1/1 (FF1 — W5.D1 line-161 paths-summary residue, FIXED with single-line edit landing exactly the falsification's recommended replacement).**
**Fresh findings: 0** (one borderline 2.3 ergonomic observation about pre-existing KindPayload free-text count `<4>` vs canonical 5 — predates Round 3, not blocking, not a Round-3 introduction).
**Orch-prompt U2 disposition:** orch-prompt framing was incorrect; PLAN.md line 174 is accurate at HEAD (5 hits matches `git grep -c`); NOT a Round-3 finding.

The Round-3 fix is surgical and complete. Commit `774df9f` modifies one line of one file (`workflow/drop_4c_6/PLAN.md` line 161). The replacement file list at line 161 is now exactly the same 5-site list at L1 header (line 5) and at the audit-detail bullets (lines 169-173) — full triple-consistency. The fix is also self-documenting via the inline `+ ROUND-2 HF6 regenerated audit ... REMOVED — both have zero default-go.toml refs at HEAD` annotation. Independent ground-truth verification via `git grep -n "default-go.toml" cmd/ internal/` at HEAD `774df9f` confirms the 5 sites match the plan claim and confirms the two over-claimed files (`auto_generate_steward.go`, `template_service.go`) have ZERO refs at HEAD.

The fresh 5-axis proof pass found no defects introduced by the Round-3 edit. The plan's Round-2 strengths (acyclic `internal/templates` chain, `_BLOCKERS.toml` ↔ PLAN.md drift-free, HF3/HF4/HF5/HF6/HF8/HF9 fixes preserved) all carry forward unchanged. Sibling W5.D2 + W5.D3 path summaries were independently verified clean (no parallel residue from the FF1 fix shape — both droplets already had correctly-formed path summaries from Round 2). The plan is ready for build-phase entry pending parallel falsification-side Round-3 verdict.

## 5. Hylla Feedback

N/A — review touched non-Go artifacts only (PLAN.md, prior-round verdict files, SKETCH.md). Hylla is Go-only today; no Hylla queries were made or required. Verification queries against committed Go source went through `git grep` (allowed-list Bash) at HEAD `774df9f` for: `default-go.toml` caller-audit count and line numbers across `cmd/` + `internal/` (single grep returns matched plan claims exactly), `default-go.toml` count in `embed.go` (5 hits — matches line 174). One sandbox-policy observation: plain `grep -c <pattern> <single-file>` was denied by the agent permissions sandbox while `git grep -c <pattern>` was allowed; if Hylla ever exposes a Go-source-line-count query for committed code, that would be a useful primitive for plan-QA `git grep` verifications without the bash-permission negotiation. Not a Round-3 miss; ergonomic observation only.
