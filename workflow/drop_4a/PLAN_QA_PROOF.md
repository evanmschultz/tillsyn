# DROP_4A — Plan QA Proof, Round 1

**Target plan:** `workflow/drop_4a/PLAN.md` (32 droplets across 5 waves; cross-references `WAVE_0_PLAN.md` through `WAVE_4_PLAN.md` and `REVISION_BRIEF.md`)
**Verdict:** PASS-WITH-NIT
**Date:** 2026-05-03
**Reviewer:** go-qa-proof-agent (filesystem-MD mode)
**Mode:** read-only review; no Tillsyn calls; no code edits

This report verifies proof-completeness of the unified Drop 4a plan. Both QA-proof and QA-falsification must return PASS (or PASS-WITH-NIT) before any builder fires. The findings below are NIT-level — they are recorded for builder + orchestrator awareness, none rise to BLOCKING.

---

## 1. Findings

### 1.1 [NIT] §8 cross-wave edge `4a.18 → 4a.6` is correct but understates direct dependencies

**Severity:** nit — transitively safe; flagged for clarity only.

**Evidence:**
`PLAN.md:412` lists `4a.6 → 4a.18` ("Walker reads `actionItem.Paths` AND `actionItem.Packages`"). Walker reads BOTH `Paths` (introduced in 4a.5) and `Packages` (introduced in 4a.6).

`WAVE_2_PLAN.md:174` confirms: *"Cross-wave: Wave 1 `paths`/`packages` on `ActionItem` (walker reads these to call lock managers in the next droplet — but pure walker logic here doesn't yet acquire locks), Wave 1 `state` MCP-create+move (walker uses lifecycle state names)."*

The §8 table edge `4a.6 → 4a.18` is transitively sufficient because 4a.6's row at `PLAN.md:171` says `Blocked by: 4a.5`, so walker effectively waits on both. But the table reader scanning §8 alone may not realize that `paths` (4a.5) is also a dependency; only the wave plan reveals it.

**Suggested fix (optional):** add a parenthetical to the §8 row: `4a.6 → 4a.18 (transitively 4a.5 — paths)`. Pure docs improvement; no plan-shape change.

### 1.2 [NIT] §8 cross-wave edges `4a.20` does not list `4a.5` (paths)

**Severity:** nit — same transitive-safety as 1.1.

**Evidence:**
`PLAN.md:413` says `4a.6 → 4a.20` ("Conflict detector reads `Paths` + `Packages` for sibling-overlap detection"). But the conflict detector reads BOTH fields. As with 1.1, transitively covered via 4a.6 → 4a.5.

**Suggested fix:** same as 1.1 — optional clarifying parenthetical.

### 1.3 [NIT] §8 omits `4a.11 → 4a.20` even though conflict detector relies on always-on parent-block invariant indirectly

**Severity:** nit — actually safe to omit; flagged for transparency.

**Evidence:**
Conflict detector (4a.20) takes `siblings []domain.ActionItem` as input (`WAVE_2_PLAN.md:211`) and detects path/package overlap. It does NOT itself depend on `RequireChildrenComplete` removal (4a.11). Walker (4a.18) is the only Wave 2 droplet that needs 4a.11's invariant for eligibility checks. Since 4a.20 is `blocked_by 4a.18` (per `PLAN.md:289`), it transitively waits on 4a.11 anyway. Confirmed safe.

**No fix needed** — current §8 wiring is correct.

### 1.4 [NIT] §9 topological-order ASCII diagram does not explicitly draw the 4a.4 fan-out edge for 4a.14 and 4a.24

**Severity:** nit — diagram is summary-style, not exhaustive; the §8 matrix is canonical.

**Evidence:**
`PLAN.md:444-454` says "Wave 2 ... `4a.14 → 4a.15 → 4a.18` ..." without redrawing the upstream `4a.4 → 4a.14` edge that `PLAN.md:410` declares in §8. Same for Wave 3: line 453 says `4a.24 → 4a.25 → ...` without showing `4a.4 → 4a.24`.

A reader using §9 alone could misread Wave 2 and Wave 3 as having no Wave-0 dependency. Lines `PLAN.md:464-466` correct this informally ("After 4a.4 closes, 4a.5 + 4a.12 + 4a.13 + 4a.14 + 4a.24 all start in parallel"), so the information is present in §9, just split between the diagram and the prose.

**Suggested fix:** none required; the prose is fine. If revising, add `4a.4 → ` prefix to each wave's first node.

### 1.5 [NIT] §6 wave-to-global mapping table headers use ASCII box-drawing inconsistently

**Severity:** cosmetic.

**Evidence:**
`PLAN.md:77-110` uses Markdown table syntax. Two cells — line 102 (`AUTH ROLE ENUM WIDENING + ORCH SELF-APPROVAL GATE + STEWARD EXCEPTION`) and the per-droplet rows in §7 — use a slightly inconsistent title casing in §6 vs. §7. §6:102 reads `STEWARD EXCEPTION`; §7 droplet 4a.24 header at `PLAN.md:322` reads `STEWARD CROSS-SUBTREE EXCEPTION`. The latter is the canonical longer form.

**Suggested fix:** harmonize §6 row 4a.24 to `STEWARD CROSS-SUBTREE EXCEPTION` for consistency. Not load-bearing.

### 1.6 [NIT] L8 (single drop, single PR) has no per-droplet binding

**Severity:** nit — architectural decision rather than per-droplet acceptance.

**Evidence:**
L8 at `PLAN.md:30` says *"Single drop, single PR, single closeout, single Hylla reingest. Not split into sub-drops."* This is a drop-level invariant enforced by Wave 4's closeout sweep (4a.29-4a.32), not by any single droplet's acceptance. Other locked decisions (L1-L7) all map to specific droplets.

This is by design — L8 governs the macro shape — but a verifier sweeping "every L bound to droplets" may flag the gap.

**Suggested fix:** add a one-liner under L8 noting "Reflected in Wave 4's unified-closeout droplet bundle (4a.29-4a.32)" so the cross-reference is explicit.

### 1.7 [NIT] Q9 (Wave 4 description-symbol verification, plan-number drift) needs orchestrator-confirmation BEFORE 4a.29 builder fires

**Severity:** nit but actionable — ambiguity that should be resolved at synthesis time, not in-flight.

**Evidence:**
`PLAN.md:483` (Q9): *"Wave 4 4a.29 description-symbol verification. `## Action-Item Lifecycle` references 'Drop 1 of the cascade plan' — verify against actual code state which Wave landed `failed` (Wave 1 of Drop 4a vs Drop 1 of original plan). Plan numbers shifted; orch confirms before edit."*

This question affects what historical-vs-canonical phrasing 4a.29 should use. The 4a.29 acceptance text at `PLAN.md:371` already says "flips Drop-1 references to past tense" — but the plan-number drift means "Drop-1 of original plan" vs "Wave 1 of Drop 4a" is not yet disambiguated.

**Suggested fix:** the orchestrator should resolve Q9 (which Wave/Drop landed `failed` as a real terminal state) BEFORE spawning the 4a.29 builder. Resolution lands in 4a.29's spawn-prompt as a pre-decided convention; otherwise the builder must resolve in-flight which risks inconsistent vocabulary across W4.1 and W4.4.

### 1.8 [NIT] Q11 (Wave 1.7 `failed` blocking parent close) is partially resolved in WAVE_1_PLAN.md but PLAN.md row 4a.11 doesn't surface the resolution

**Severity:** nit — substantive resolution exists; just not propagated.

**Evidence:**
`PLAN.md:485` (Q11): *"Wave 1.7's stuck-parent test for `failed` children (4a.11). Brief-default is `failed` blocks parent close. Plan-QA falsification attack: is this the right semantics, or does `failed` get a separate 'stuck' treatment?"*

`WAVE_1_PLAN.md:195` resolves it: *"Decision: `failed` children also block parent close (matches Drop 1's always-on rule semantics — `failed` is a terminal-non-success state, not a closeable state)."* And `WAVE_1_PLAN.md:215` explicitly documents the bypass-via-supersede rationale.

The 4a.11 row at `PLAN.md:212` notes: *"Stuck-parent failure mode (no supersede CLI in 4a) is an explicit pre-MVP cost — dev fresh-DBs is the escape valve."* So the resolution IS present.

The gap: Q11 in §10 reads as if the question is open. It is in fact resolved (by both the wave plan and the 4a.11 row); §10 hasn't been updated to reflect the resolution.

**Suggested fix:** mark Q11 as `[RESOLVED in WAVE_1_PLAN.md §1.7]` so falsification doesn't re-litigate it. Same disposition for Q12 (which is also resolved by 4a.10's "specify exactly one" rule at `PLAN.md:202`).

### 1.9 [NIT] §8 lacks an explicit row for `4a.4 → 4a.24` even though it's enumerated

**Severity:** nit — trivially present in the table, but worth confirming I read it right.

**Evidence:**
`PLAN.md:411`: `| 4a.4 | 4a.24 (Wave 3.1) | Wave 0 close — auth code can start parallel with Wave 1+2.` ✓ Confirmed present. Initial scan missed this; recording the verification so the matrix is fully audited.

**No fix needed.**

### 1.10 [NIT] 4a.13 acceptance has Branch A / Branch B split — both are valid completion outcomes, document explicitly in PLAN row

**Severity:** nit — wave plan documents this clearly; PLAN.md row could surface it more.

**Evidence:**
`PLAN.md:226`: *"**Branch A** (no drift): worklog records evidence, no code change. **Branch B** (drift found): flip stray `\"Done\"` → `\"Complete\"` in seeding code."* OK, surfaced.

`WAVE_1_PLAN.md:268-269` resolves the expected branch ("primary expected outcome is Branch A — confirmation note").

**No fix needed** — the row is already explicit. Recording confirmation.

---

## 2. Missing Evidence

None found. Every concrete claim in the unified PLAN is backed by either:
- a wave-plan citation (cross-referenced in §5),
- a code-symbol reference verified against the wave plans,
- or an explicit "verify before edit" instruction (see Q9, 4a.13, 4a.29).

---

## 3. Per-Section Verification

### 3.1 §1 Goal — `PLAN.md:11-16`

Goal text matches `REVISION_BRIEF.md:8-12` (manual-trigger dispatcher; git/commit/push/Hylla stay manual; 4a + 4b = MVP-feature-complete cascade). ✓ Consistent.

### 3.2 §2 Locked Architectural Decisions L1-L8 — `PLAN.md:19-30`

Each of L1-L8 maps directly to `REVISION_BRIEF.md:80-88`. Matches verbatim. ✓ All 8 decisions reflected.

L1-L8 droplet bindings (verified against §7 droplet rows):

| Locked | Droplet(s) | Evidence |
| --- | --- | --- |
| L1 (state MCP) | 4a.10 | `PLAN.md:202` "L1. column_id stays in DB" |
| L2 (always-on parent-blocks) | 4a.11 | `PLAN.md:212` "L2." |
| L3 (paths/packages/files/start_commit/end_commit first-class) | 4a.5, 4a.6, 4a.7, 4a.8, 4a.9 | `PLAN.md:158-196` per-droplet |
| L4 (project first-class fields) | 4a.12 | `PLAN.md:220` "L4." |
| L5 (Drop 1.6 absorbed) | 4a.24, 4a.25, 4a.26, 4a.27, 4a.28 | `PLAN.md:54` "Drop 1.6 abs." in wave header |
| L6 (Wave 0 first) | 4a.1-4a.4 + cross-wave 4a.4→all | `PLAN.md:51` Wave 0 sequence "First" |
| L7 (manual-trigger) | 4a.23 | `PLAN.md:314` "L7 manual-trigger milestone deliverable" |
| L8 (single drop, single PR) | implicit (Wave 4 closeout bundle 4a.29-4a.32) | see finding 1.6 |

All L1-L7 have explicit droplet bindings. L8 is implicitly enforced by the unified Wave 4 closeout (see finding 1.6).

### 3.3 §3 Pre-MVP Rules — `PLAN.md:34-43`

All seven pre-MVP rules match memory:
- No migration logic in Go ✓ (`PLAN.md:36`, `feedback_no_migration_logic_pre_mvp.md`)
- No closeout MD rollups ✓ (`PLAN.md:37`, `feedback_no_closeout_md_pre_dogfood.md`)
- Opus builders ✓ (`PLAN.md:38`, `feedback_opus_builders_pre_mvp.md`)
- Filesystem-MD mode ✓ (`PLAN.md:39`)
- Section 0 + tillsyn-flow ✓ (`PLAN.md:40`)
- Single-line commits ≤72 chars ✓ (`PLAN.md:41`, `feedback_commit_style.md`)
- Never raw `go test`/`go build`/`go vet`/`mage install` ✓ (`PLAN.md:42`)
- Hylla Go-only ✓ (`PLAN.md:43`, `feedback_hylla_go_only_today.md`)

Test-helper fakeagent compilation carve-out documented at `PLAN.md:42` and `WAVE_2_PLAN.md:246` — explicit and bounded. ✓

### 3.4 §4 Wave Structure — `PLAN.md:47-57`

Total count matches: 4+9+10+5+4 = 32. Verified per-wave below.

### 3.5 §5 Wave-Internal-Plan Cross-References — `PLAN.md:61-71`

Five wave plans referenced; all five exist on disk (verified via `Bash` ls of `workflow/drop_4a/`). ✓

### 3.6 §6 Wave-to-Global ID Mapping — `PLAN.md:75-110`

Counted 32 rows: 4 (W0) + 9 (W1) + 10 (W2) + 5 (W3) + 4 (W4) = 32. Each row has a wave-internal ID, a global ID, and a title. Titles are FULL UPPERCASE per memory `feedback_tillsyn_titles.md`. ✓

Minor cosmetic gap noted in finding 1.5 (title-text consistency 4a.24 across §6 vs §7).

### 3.7 §7 Per-Droplet Rows — `PLAN.md:114-397`

Each of the 32 droplet rows carries:
- Title (FULL UPPERCASE, in `## #### 4a.X — TITLE` format).
- Paths summary.
- Packages summary.
- Acceptance summary (with WAVE_N_PLAN.md cross-reference).
- Blocked by (with global IDs).
- Notes (rationale, irreducibility flag, DB-action callout).

Spot-checks:
- 4a.5 (`PLAN.md:158-164`): 12-file Paths list ✓; 5 packages ✓; "domain field + CreateActionItemInput + UpdateActionItemInput + SQL paths_json TEXT NOT NULL DEFAULT '[]' + MCP wire + snapshot. Domain validation: trim + dedup; reject whitespace-only / backslash-paths with `ErrInvalidPaths`. See WAVE_1_PLAN.md §1.1." ✓ Concrete, measurable.
- 4a.18 (`PLAN.md:268-274`): paths `walker.go` + test ✓; package `internal/app/dispatcher` ✓; acceptance enumerates `EligibleForPromotion`, `Promote`, `ErrPromotionBlocked` typed error ✓; blocked_by `4a.14, 4a.15, 4a.6, 4a.10, 4a.11` ✓.
- 4a.27 (`PLAN.md:346-351`): 5 test cases enumerated explicitly with expected pass/reject for each ✓.
- 4a.29 (`PLAN.md:367-374`): section-by-section sweep instructions with line numbers from `main/CLAUDE.md` ✓; blocked_by spans Wave 1 (4a.11, 4a.12, 4a.13), Wave 2 (4a.23), Wave 3 (4a.28) ✓.

Acceptance criteria are concrete and measurable — every droplet either delegates to its wave plan for full detail (which I verified contains testable conditions) or self-contains acceptance bullets. No "module works correctly" placeholders found.

### 3.8 §8 Cross-Wave Blocker Wiring (DAG) — `PLAN.md:401-428`

Walked the matrix as a directed graph. Nodes: 32 droplets. Edges: 19 cross-wave + intra-wave from per-droplet rows.

**DAG acyclicity**: every edge points from lower-numbered droplet to higher-numbered droplet (verified by inspection of every blocked-by relation). Since the partial order respects the global ID ordering, the graph is acyclic by construction. ✓

**No dangling references**: every blocker mentioned (4a.4, 4a.5, 4a.6, 4a.10, 4a.11, 4a.12, 4a.13, 4a.14, 4a.15, 4a.16, 4a.17, 4a.18, 4a.19, 4a.20, 4a.21, 4a.22, 4a.23, 4a.24, 4a.25, 4a.26, 4a.27, 4a.28, 4a.29) exists in §6's mapping table. ✓

**Blocker-completeness verification (by walking each droplet)**:
- 4a.1: no blockers ✓ (root of Wave 0)
- 4a.2: 4a.1 ✓
- 4a.3: 4a.2 ✓
- 4a.4: 4a.3 ✓
- 4a.5: 4a.4 (Wave 0 close) ✓
- 4a.6: 4a.5 (chain) ✓
- 4a.7: 4a.6 ✓
- 4a.8: 4a.7 ✓
- 4a.9: 4a.8 ✓
- 4a.10: 4a.9 ✓
- 4a.11: 4a.10 ✓
- 4a.12: 4a.4 (parallel from Wave 0) ✓
- 4a.13: 4a.4 (parallel from Wave 0) ✓
- 4a.14: 4a.4 ✓
- 4a.15: 4a.14 ✓
- 4a.16: 4a.14 ✓
- 4a.17: 4a.14, 4a.16 ✓
- 4a.18: 4a.14, 4a.15, 4a.6, 4a.10, 4a.11 ✓
- 4a.19: 4a.14, 4a.12 ✓
- 4a.20: 4a.14, 4a.18, 4a.6 ✓
- 4a.21: 4a.14, 4a.19 ✓
- 4a.22: 4a.14, 4a.16, 4a.17, 4a.19 ✓
- 4a.23: 4a.14, 4a.18, 4a.19, 4a.20, 4a.21, 4a.22, 4a.12 ✓
- 4a.24: 4a.4 (parallel with Wave 2) ✓
- 4a.25: 4a.24 ✓
- 4a.26: 4a.25 ✓
- 4a.27: 4a.26 ✓
- 4a.28: 4a.27 ✓
- 4a.29: 4a.11, 4a.12, 4a.13, 4a.23, 4a.28 ✓
- 4a.30: 4a.11, 4a.12, 4a.13, 4a.23, 4a.28 ✓
- 4a.31: 4a.28 ✓
- 4a.32: 4a.28, 4a.29 ✓

All 32 droplets accounted for. Every blocker reference resolves. DAG is acyclic.

### 3.9 §9 Topological Order — `PLAN.md:432-461`

Spot-check every node appears at least once in §9's ASCII diagram:

Wave 0: `4a.1 → 4a.2 → 4a.3 → 4a.4` — 4 nodes ✓
Wave 1: `4a.5 → 4a.6 → 4a.7 → 4a.8 → 4a.9 → 4a.10 → 4a.11`, `4a.12`, `4a.13` — 9 nodes ✓
Wave 2: `4a.14 → 4a.15 → 4a.18`, `4a.14 → 4a.16 → 4a.17`, `4a.14 → 4a.19 → 4a.21`, `4a.18 + 4a.6 → 4a.20`, `4a.16 + 4a.17 + 4a.19 → 4a.22`, `4a.18 + 4a.19 + 4a.20 + 4a.21 + 4a.22 + 4a.12 → 4a.23` — all 10 nodes (4a.14, 4a.15, 4a.16, 4a.17, 4a.18, 4a.19, 4a.20, 4a.21, 4a.22, 4a.23) ✓
Wave 3: `4a.24 → 4a.25 → 4a.26 → 4a.27 → 4a.28` — 5 nodes ✓
Wave 4: `4a.29`, `4a.30`, `4a.31`, `4a.32` — 4 nodes ✓

Total: 4+9+10+5+4 = 32 ✓.

Topological consistency: each "→" in §9 corresponds to a real edge in §8's matrix (or to an intra-wave per-droplet `Blocked by` row). Spot-verified the Wave 2 reconvergence at 4a.23 against `PLAN.md:313`. ✓

§9 lines 463-467 (parallelism notes) match §8 fan-out from 4a.4. ✓

### 3.10 §10 Open Questions Q1-Q12 — `PLAN.md:471-486`

Categorization for build-dispatch readiness:

| Q | Subject | Status | Resolvable in-flight? |
| --- | --- | --- | --- |
| Q1 | 4a.11 stuck-parent failure mode | RESOLVED in WAVE_1_PLAN.md §1.7 — accept pre-MVP cost, dev fresh-DBs is escape valve. | n/a |
| Q2 | Wave 2 sub-package locks | RESOLVED in WAVE_2_PLAN.md §"File layout decision" — keep flat. | n/a |
| Q3 | spawn-vs-execution split (4a.19 vs 4a.21) | OPEN-but-acceptable: planner stance documented; QA Falsification may push back, but builder can proceed. | yes |
| Q4 | auth-bundle stub seam (4a.19) | OPEN-but-acceptable: stub interface documented; Wave 3 plugs in. | yes |
| Q5 | metadata.failure_reason shape (4a.21) | RESOLVED: free-form for 4a; 4b refactors. | n/a |
| Q6 | test-helper `go build` carve-out (4a.21) | RESOLVED in PLAN.md §3 ("Test-helper fakeagent compilation in Wave 2.8 is a documented carve-out"). | n/a |
| Q7 | CLI bootstrap symmetry with `till serve` | RESOLVED: minimal CLI today; 4b refactors. | n/a |
| Q8 | 4a.24 scope size | OPEN-but-acceptable: builder discipline (package-by-package `mage test-pkg` early) documented. | yes |
| Q9 | 4a.29 plan-number drift | **NEEDS resolution before 4a.29 builder fires** (see finding 1.7). | should be pre-resolved |
| Q10 | 4a.32 outside-repo audit-gap | RESOLVED: drop_3/3.27 finding 5.D.10 acceptance reaffirmed. | n/a |
| Q11 | failed children blocking parent close | RESOLVED in WAVE_1_PLAN.md §1.7 (see finding 1.8). | n/a |
| Q12 | column_id back-compat (4a.10) | RESOLVED: dual-acceptance + "specify exactly one" rejection. Load-bearing for TUI. | n/a |

**Builder-blocking open questions:** Q9 (see finding 1.7).
**In-flight resolvable:** Q3, Q4, Q8.
**Already resolved (can be marked as such):** Q1, Q2, Q5, Q6, Q7, Q10, Q11, Q12.

§10 represents legitimate falsification surface; only Q9 needs proactive resolution. All other open questions are either documented decisions or builder-discipline items.

### 3.11 §11 Drop 4b Kickoff — `PLAN.md:490-504`

Out-of-scope items deferred to 4b match `REVISION_BRIEF.md:73-78` ("Drop 4b scope: gate runner reading template `[gates]`, commit-agent (haiku) integration, `git commit` + `git push` automation, Hylla reingest hook on `closeout`, auth auto-revoke on terminal state, git-status-pre-check on action-item creation"). ✓ Consistent.

---

## 4. DAG Verification (Explicit Walk)

Treating the 32 droplets as nodes and §8 + per-droplet `Blocked by` rows as edges, I constructed the directed graph in memory and verified:

- **Acyclicity**: every edge runs from a lower-numbered droplet to a higher-numbered droplet. There exists no edge `4a.X → 4a.Y` where X > Y. Therefore no back-edge exists; the graph is a DAG.
- **Topological consistency with §9**: Kahn's algorithm starting from in-degree-0 nodes gives `[4a.1]` → `[4a.2]` → `[4a.3]` → `[4a.4]` → `[4a.5, 4a.12, 4a.13, 4a.14, 4a.24]` → ... matching §9's prose "After 4a.4 closes, 4a.5 + 4a.12 + 4a.13 + 4a.14 + 4a.24 all start in parallel" (`PLAN.md:465`). ✓
- **Same-file lock chains**:
  - `internal/domain/action_item.go` chain (Wave 1.1-1.5 + 1.7): 4a.5 → 4a.6 → 4a.7 → 4a.8 → 4a.9 → ... with 4a.10's adapter touch and 4a.11's domain touch enforcing same-file lock through to chain end. Linear chain. ✓
  - `internal/app/dispatcher` package chain (Wave 2): every Wave 2 droplet `blocked_by 4a.14`. ✓
  - `app_service_adapter_mcp.go` Wave 3 chain (4a.24 → 4a.25 → 4a.26): linear, same-file lock ✓.
  - Wave 4 (4a.29-4a.32): all four `Irreducible: true`; W4.4 blocks on W4.1 for cross-doc consistency ✓.

No cycles. No missing nodes. No stranded edges.

---

## 5. Locked-Decision Verification

L1-L8 mapped to droplets at §3.2 above. Re-stating concisely:

- **L1 (state MCP)** → 4a.10. Eight test cases (4 create + 4 move) cover dual-input, both-empty rejection, both-non-empty rejection.
- **L2 (always-on parent-blocks)** → 4a.11. Four new test cases (complete-child / in_progress-child / failed-child / archived-child).
- **L3 (5 first-class action-item fields)** → 4a.5, 4a.6, 4a.7, 4a.8, 4a.9. Each has its own surgical pattern: domain struct + Input + SQL + MCP + snapshot.
- **L4 (6 first-class project fields)** → 4a.12 (bundled per Drop-3-3.21 precedent; rationale documented).
- **L5 (Drop 1.6 absorbed in Wave 3)** → 4a.24, 4a.25, 4a.26, 4a.27, 4a.28.
- **L6 (Wave 0 first)** → 4a.1-4a.4 + every Wave 1/2/3/4 droplet's blocker chain transits 4a.4.
- **L7 (manual-trigger)** → 4a.23 (CLI manual-trigger milestone).
- **L8 (single drop, single PR)** → architectural; reflected in Wave 4's unified-closeout bundle (see finding 1.6).

All eight locked decisions are reflected. ✓

---

## 6. Cross-Wave Edge Sample Verification (per spawn-prompt axis 6)

Sampled 5 cross-wave edges from §8 and verified each against the wave plans:

### 6.1 4a.18 → 4a.6 (walker reads paths + packages)

`PLAN.md:412` claim: *"Walker reads `actionItem.Paths` AND `actionItem.Packages`."*
`WAVE_2_PLAN.md:174` confirms: *"Cross-wave: Wave 1 `paths`/`packages` on `ActionItem`."*
Walker eligibility code path uses these for lock acquisition coordination (`WAVE_2_PLAN.md:160-178`). ✓

### 6.2 4a.18 → 4a.10 (walker uses state MCP)

`PLAN.md:414` claim: *"Walker uses `state` MCP API for promotion."*
`WAVE_2_PLAN.md:174`: *"Wave 1 `state` MCP-create+move (walker uses lifecycle state names)."*
`WAVE_2_PLAN.md:167`: *"`Promote` ... resolves the `in_progress` column ID via the existing `lifecycleStateForColumnID` helper, then calls `Service.MoveActionItem`."* The `state`-resolution path is what 4a.10 introduces. ✓

### 6.3 4a.18 → 4a.11 (walker eligibility relies on always-on parent-block invariant)

`PLAN.md:415` claim: *"Eligibility predicate relies on always-on parent-block invariant."*
`WAVE_2_PLAN.md:162` (eligibility predicate item 3): *"Children-complete is a precondition for promotion *to* `complete`, enforced by `ensureActionItemCompletionBlockersClear` already."*

That's the precondition-reading side. 4a.11 makes it unconditional (`PLAN.md:212` — removes `RequireChildrenComplete` policy bit). The walker's eligibility check simplifies because there's no conditional branch on the policy bit. ✓

### 6.4 4a.19 → 4a.12 (spawn reads project fields)

`PLAN.md:416` claim: *"Spawn reads `RepoPrimaryWorktree`, `Language`, `HyllaArtifactRef`, `DevMcpServerName`."*
`WAVE_2_PLAN.md:188` (spawn `Dir = project.RepoPrimaryWorktree`).
`WAVE_2_PLAN.md:198`: *"Cross-wave: Wave 1 project-node fields (`RepoPrimaryWorktree`, `Language`, `HyllaArtifactRef`, `DevMcpServerName`)."* ✓

### 6.5 4a.23 → 4a.12 (CLI bootstraps Service against project with project fields populated)

`PLAN.md:417` claim: *"CLI bootstrap constructs Service against project with project fields populated."*
`WAVE_2_PLAN.md:289`: *"Cross-wave: Wave 1 project-node fields (CLI must construct `Service` against a project that has `RepoPrimaryWorktree` populated)."* ✓

### 6.6 4a.27 → 4a.26 (audit fields exercised in case (a) + (d))

(Bonus — Wave 3 internal edge.) `PLAN.md:351` claim: *"4a.26 (audit fields exercised in case (a) + (d) assertions)."*
`WAVE_3_PLAN.md:135`: case (a) asserts *"`auth_request` row's `approving_principal_id` matches the orch's principal_id"* — that's 4a.26's audit-trail field. ✓

All sampled cross-wave edges have wave-plan-verified backing. No fictional dependencies.

---

## 7. Pre-MVP Discipline Verification

Checked each pre-MVP rule against per-droplet acceptance:

- **No migration logic**: every schema-touching droplet (4a.5, 4a.6, 4a.7, 4a.8, 4a.9, 4a.11 vestigial JSON, 4a.12, 4a.26) carries explicit "DB action: dev fresh-DBs `~/.tillsyn/tillsyn.db` BEFORE `mage ci`." ✓
- **No closeout MD rollups**: Wave 4 (`PLAN.md:367-397`) explicitly lists CLAUDE.md / WIKI.md / STEWARD_ORCH_PROMPT.md / outside-repo files only; no LEDGER.md / WIKI_CHANGELOG.md / REFINEMENTS.md / HYLLA_FEEDBACK.md sweeps. `WAVE_4_PLAN.md:204` confirms. ✓
- **Opus builders**: PLAN.md §3 line 38, reaffirmed in `WAVE_4_PLAN.md:240`. ✓
- **Filesystem-MD mode**: PLAN.md:6 + §3 line 39 + WAVE_4_PLAN.md:211 ("No subagent QA twins per droplet"). ✓
- **Tillsyn-flow + Section 0**: PLAN.md §3 line 40. (Spawn-prompt directive lives in CLAUDE.md / SEMI-FORMAL-REASONING.md; not duplicated here, correct.)
- **Single-line commits**: PLAN.md §3 line 41; `WAVE_4_PLAN.md:241`. ✓
- **NEVER `mage install` / raw `go test`**: PLAN.md §3 line 42; `WAVE_2_PLAN.md:70`. The fakeagent `go build` test-helper carve-out at 4a.21 (`PLAN.md:298`) is explicit and bounded — single use, documented precedent. ✓
- **Hylla Go-only**: `PLAN.md:43`; `WAVE_4_PLAN.md:210`. ✓

All eight pre-MVP rules carried into per-droplet acceptance. ✓

---

## 8. Count Verification

§4 wave structure declares 4+9+10+5+4 = 32. §6 mapping table contains exactly 32 rows (manually counted: lines 80-110 of PLAN.md, one droplet per row). §7 per-droplet section contains 32 droplet headers (counted by `####` h4 markers). §9 topological-order block enumerates all 32 IDs. ✓

---

## 9. Verdict Summary

**PASS-WITH-NIT.**

The unified Drop 4a plan is proof-complete:
- 32 droplets total, each carrying global ID, FULL UPPERCASE title, paths, packages, acceptance summary (with wave-plan cross-reference for full detail), and `blocked_by` chain.
- DAG is acyclic, has no dangling references, has no missing nodes.
- §9 topological order is consistent with §8 blocker matrix.
- All 8 locked architectural decisions (L1-L8) are reflected in droplet bindings (L8 implicitly via Wave 4 unified-closeout — see finding 1.6).
- Cross-wave edges sampled verified against wave plans (6/6 confirmed).
- Pre-MVP rules honored across all droplets.
- Acceptance criteria are concrete and measurable (bullet-level test scenarios; specific symbol names; specific file paths; specific mage targets).
- Same-file-lock chains correctly serialize touching droplets.

Ten NIT-level findings recorded. None block builder dispatch. **Recommended action items before builder fires:**

- **Pre-resolve Q9 (finding 1.7)** — orchestrator confirms which Wave/Drop landed `failed` as a real terminal state so 4a.29's vocabulary lands consistently across the closeout sweep.

All other findings are documentation-clarity adjustments that the orchestrator may choose to apply or defer; none alter plan correctness.

The plan is cleared for builder dispatch pending QA-falsification's parallel result.

---

## Hylla Feedback

N/A — task touched non-Go files only. This was a pure plan-doc review across `workflow/drop_4a/PLAN.md`, `WAVE_0_PLAN.md`, `WAVE_1_PLAN.md`, `WAVE_2_PLAN.md`, `WAVE_3_PLAN.md`, `WAVE_4_PLAN.md`, and `REVISION_BRIEF.md`. No Go-symbol queries were issued because the plan's symbol references either point at code that is yet-to-be-written (Wave 2's new dispatcher package, Wave 3's audit columns) or at existing code that the wave plans already cite with line numbers. Hylla today indexes only Go and is not the right tool for cross-Markdown plan review.

---

## TL;DR

- **T1**: Ten NIT-level findings recorded; none blocking. Most are documentation-clarity items (cross-wave edge transitivity, §10 question-status updates, L8 droplet-binding gap).
- **T2**: No missing evidence — every concrete claim has a wave-plan or memory citation.
- **T3**: Per-section verification: §1-§11 all cleared. L1-L8 each map to specific droplets (L8 implicitly to Wave 4 closeout bundle).
- **T4**: DAG explicitly walked — 32 nodes, all blocker references resolve, no cycles, topological order in §9 consistent with §8 matrix.
- **T5**: All 8 locked decisions L1-L8 verified against droplet bindings.
- **T6**: 6 cross-wave edges sampled (5 spawn-prompt-required + 1 bonus); all six confirmed against wave plans.
- **T7**: All 8 pre-MVP rules carry into per-droplet acceptance — no migration logic, no closeout rollups, opus builders, filesystem-MD, single-line commits, never mage install / raw go, Hylla Go-only.
- **T8**: Count verified: 4+9+10+5+4 = 32 in §4 / §6 / §7 / §9.
- **T9**: Verdict PASS-WITH-NIT. Cleared for builder dispatch pending falsification sibling. Pre-resolve Q9 (4a.29 plan-number drift) before Wave 4 builder fires.
