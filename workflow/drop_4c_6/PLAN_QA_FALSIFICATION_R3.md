# Plan-QA-Falsification Round 3 — Drop 4c.6

**Reviewer:** plan-QA-falsification subagent (Round 3)
**Authored:** 2026-05-09
**Plan under attack:** `workflow/drop_4c_6/PLAN.md` REVISED Round 3 (commit `774df9f`).
**Sketch source-of-truth (locked, NOT under attack):** `workflow/drop_4c_6/SKETCH.md` v2.8.4 POST-QA FINAL.
**Round-1 verdict:** FAIL — 7 CONFIRMED counterexamples + 1 borderline.
**Round-2 verdict:** FAIL — 1 CONFIRMED counterexample (FF2.1, paths-summary residue at PLAN.md:161).
**Working dir:** `/Users/evanschultz/Documents/Code/hylla/tillsyn/main`.
**HEAD at review:** `774df9f`.

## 1. Round-2 Counterexample Verification

| # | Round-2 finding | Round-3 status | Evidence |
| --- | --- | --- | --- |
| FF2.1 | W5.D1 inline `**Paths:**` summary at PLAN.md line 161 named two over-claimed files (`internal/app/auto_generate_steward.go`, `internal/app/template_service.go`) — contradicting line 176's HF6 removal note + the L1 header at line 5. | **FIXED** | `git diff 95ebe58..774df9f -- workflow/drop_4c_6/PLAN.md` shows the single-line edit replacing the four over-claimed-pair entries with the correct five caller-audit sites (`service.go`, `service_test.go`, `auto_generate_steward_test.go`, `mcp_surface.go`, `extended_tools.go`). PLAN.md line 161 (post-Round-3) now matches L1 header line 5 + W5.D1 audit-detail block lines 168-176 + HF6 removal note line 176. Independent verification via `git grep -c "default-go.toml" <each-file>` against HEAD `774df9f`: `service.go=1`, `service_test.go=7`, `auto_generate_steward_test.go=1`, `mcp_surface.go=1`, `extended_tools.go=1`; `auto_generate_steward.go` and `template_service.go` return zero hits (correctly removed). The Round-3 commit also added a verification trail string ("caller list verified independently in ROUND-3 via `git grep ...` against HEAD `f32b9d8`; over-claimed `...auto_generate_steward.go` and `...template_service.go` REMOVED — both have zero `default-go.toml` refs at HEAD") which captures the round-3 fix discipline in-prose. |

**Net Round-2 fix status:** 1/1 fully fixed. Single-spot edit landed cleanly; no collateral damage to surrounding bullets, audit-detail block, blockers, or graph summary.

## 2. Fresh Counterexamples

Fresh 7-family attack pass on the Round-3 plan state.

### 2.1 Dev-flagged "minor pre-existing prose drift" at line 174 — INVESTIGATED, REFUTED

The orchestrator spawn prompt asked me to decide whether the following is a CONFIRMED counterexample:

> "PLAN.md line 174's W5.D1 RiskNotes prose says `internal/templates/embed.go` has '5 hits including … historical doc-comments at :17, :62, :106' of `default-go.toml`; at HEAD, `git grep -c "default-go.toml" internal/templates/embed.go` returns 6, not 5."

**Verification at HEAD `774df9f`:**

```
$ git grep -c "default-go.toml" internal/templates/embed.go
internal/templates/embed.go:5

$ git grep -n "default-go.toml" internal/templates/embed.go
internal/templates/embed.go:17:// `default-go.toml` so sibling builtins (and, post-Q1 resolution, future
internal/templates/embed.go:34://go:embed builtin/default-go.toml builtin/default-generic.toml
internal/templates/embed.go:62:// `default-go.toml` directly, so every caller received the Go-flavored
internal/templates/embed.go:106://   - `"go"`   → loads `builtin/default-go.toml` (the Go-flavored full
internal/templates/embed.go:138:		path = "builtin/default-go.toml"
```

The actual count at HEAD `774df9f` is **5 hits at lines 17, 34, 62, 106, 138**. PLAN.md line 174 reads exactly: *"5 hits including the `//go:embed` directive at :34, switch case at :138, plus historical doc-comments at :17, :62, :106"* — that's 5 hits, lines 17/34/62/106/138 — which matches `git grep` output line-for-line. The orchestrator's spawn-prompt assertion that `git grep -c` returns "6, not 5" is itself incorrect at HEAD `774df9f`. NOT a CONFIRMED counterexample. **REFUTED.** (Pre-existing prose IS accurate against current HEAD; the perceived drift was a false alarm.)

Note: I do not modify PLAN.md or post a fix-it suggestion because there is nothing to fix. The dev-flagged drift does not exist.

### 2.2 Cycle check (REQUIRED Phase-2-step-1 attack) — REFUTED

Walked every `blocked_by` edge in PLAN.md + `_BLOCKERS.toml` post-Round-3 (Round-3 only edited line 161 — graph + blockers untouched):

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

Topological sort terminates: {W0, W0.5, W6.D2, W6.D3} → {W1.D1, W6.D1} → {W2, W3, W5.D1} → {W5.D2, W6.D4} → {W5.D3, W6.D5}. No cycle introduced. Identical to Round-2 result (graph unchanged). REFUTED.

### 2.3 `_BLOCKERS.toml` / PLAN.md drift (REQUIRED Phase-2-step-1 attack) — REFUTED

`_BLOCKERS.toml` was NOT modified in Round-3 (commit `774df9f` touches PLAN.md only — verified via `git diff 95ebe58..774df9f` shows only `workflow/drop_4c_6/PLAN.md` changed). The blockers ledger correspondence verified in Round-2 still holds: every `[[blockers]]` row matches an inline `Blocked by:` bullet in PLAN.md and vice versa. Re-walked the 9 mappings (Wave-A heads + 8 explicit edges) — identical to Round-2's clean result. Zero drift. REFUTED.

### 2.4 Sibling overlap without `blocked_by` (REQUIRED Phase-2-step-1 attack) — REFUTED

Re-walked every (paths, packages) intersection across the 13 immediate L1 children. The Round-3 edit at PLAN.md line 161 changed the prose of W5.D1's `**Paths:**` field but did NOT add or remove any path entry from W5.D1's effective scope (the audit-detail block at lines 168-176 was already the truth; line 161 merely caught up). All `internal/templates`-touching siblings still serialize via the chain W0.5 → W1.D1 → W5.D1 → W5.D2 → W5.D3. No new file collisions introduced. REFUTED.

### 2.5 A1 Concurrency / blocked_by attacks — REFUTED

Round-3 edit doesn't touch any blocker. The W0.5 → W1.D1 → W5.D1 → W5.D2 → W5.D3 `internal/templates` chain remains intact. The W3 / W2 / W6 chains untouched. REFUTED.

### 2.6 A2 Contract-mismatch — EXHAUSTED, no concrete counterexample

Re-read W5.D1's full droplet block (lines 157-189) post-Round-3:
- Line 161 paths summary now matches lines 168-176 audit-detail block — internal consistency restored (this is the Round-2 FF2.1 fix).
- Line 161 paths summary names the same 5 caller-audit sites that L1 header line 5 names — cross-block consistency restored.
- Line 161's "12 source files, ~30 string edits" parenthetical from Round-2 is **gone** (Round-3 commit replaced the entire parenthetical with the new "ROUND-2 HF6 regenerated audit + ROUND-3 verification" annotation). The historical parenthetical's removal is appropriate because the figure was a citation to `RESEARCH/CASCADE_ENFORCEMENT_AND_CONTEXT_PRELOAD.md §C` which is research-doc and not an acceptance contract; its absence does not weaken any acceptance bullet.
- Acceptance bullet at line 167 (post-rename grep should return zero hits in non-doc-comment locations) coheres with bullet at lines 168-176 (which classifies historical doc-comments as RETAINED) — the contradiction Round-1 raised was already cured in Round-2; Round-3 didn't re-introduce it.
- KindPayload at line 189 still references `<4 caller audit sites>` literal in the placeholder — that's a hand-wavy stub and was so before Round-3 too; with the L1 paths line now naming 5 sites, this is a minor RiskNote-class observation rather than a contract mismatch (KindPayload is dispatcher-bound metadata for downstream automation; an L2 builder reads the explicit Acceptance + paths line, not the KindPayload's `<4 caller audit sites>` placeholder text). Concrete repro requires an automation that consumes KindPayload directly — none exists pre-cascade. NOT a CONFIRMED counterexample under Counterexample-vs-Noise discipline.

EXHAUSTED, no concrete repro found.

### 2.7 A3 Hidden-coupling — EXHAUSTED, no concrete counterexample

The Round-3 line 161 edit eliminates the only Round-2 hidden-coupling counterexample (FF2.1 is the test case — now fixed). Re-grepped `default-go.toml` against `cmd/ internal/` at HEAD `774df9f`:
- 5 caller-audit sites named in PLAN.md line 161 each have ≥ 1 hit. ✓
- 2 over-claimed sites named in line 176 each have 0 hits. ✓
- `internal/templates/embed.go` 5 hits matches PLAN.md line 174's "5 hits" claim. ✓
- `internal/templates/load.go` 3 hits at 255, 592, 735 matches PLAN.md line 174's "3 hits in doc-comments at line 255, 592, 735" claim exactly. ✓
- `internal/templates/load_test.go` 2 hits at 1709, 1927 matches PLAN.md line 174's "2 hits at 1709, 1927" claim exactly. ✓
- `internal/templates/builtin/default-generic.toml` 8 hits at lines 3, 7, 35, 40, 253, 261, 273, 312 — PLAN.md line 175 enumerates the same 8 lines exactly. ✓
- `internal/app/service_test.go` 7 hits matches the "7 hits" claim at line 170; line 6534 is the load-bearing `filepath.Join(...)` literal as the plan claims. ✓

Every concrete number + line citation in the plan's W5.D1 caller-audit + acceptance prose matches `git grep` reality at HEAD `774df9f`. EXHAUSTED, no concrete repro found.

### 2.8 A4 YAGNI / scope-creep — EXHAUSTED, no concrete counterexample

Round-3 edit replaced one prose sentence with a more accurate prose sentence — net delta is the same files in scope, more accurate text describing them. No new abstraction, no new field, no new validator, no new helper. The verification-trail string ("caller list verified independently in ROUND-3 via `git grep ...`") is provenance prose, not a new acceptance contract. EXHAUSTED.

### 2.9 A5 Shipped-but-not-wired — EXHAUSTED, no concrete counterexample

Round-2 HF8 wiring contract for W3's post-render validator (PLAN.md line 147 + L2 sub-planner directive line 153) is unchanged in Round-3. W0.5 known-wired-set deferral (Open Questions disposition) is unchanged. EXHAUSTED.

### 2.10 A6 Atomicity — EXHAUSTED-low, no fresh decomp change

Round-3 edit doesn't change W5.D1's atomicity classification (still atomic droplet) or footprint (the path list at L1 header line 5 already named the corrected 5 sites in Round-2; the W5.D1 inline summary at line 161 simply caught up). W1.D1's ~24-file footprint borderline argument (Round-1 1.8 / Round-2 A6) is unchanged — no Round-3 movement on this dimension. KEPT-AS-AUTHORED per dev disposition.

### 2.11 A7 Prompt-injection — EXHAUSTED (DORMANT pre-team-feature)

Per `feedback_prompt_injection_team.md`, prompt-injection axis is dormant pre-team-feature. No team functionality lands in this drop. EXHAUSTED.

## 3. Summary

**Verdict: pass**

**Round-2 fix net:** 1/1 fully fixed (FF2.1 paths-summary residue → fixed at PLAN.md:161 in commit `774df9f`).

**Fresh counterexamples:** 0.

**Dev-flagged line-174 prose drift:** REFUTED — `git grep -c "default-go.toml" internal/templates/embed.go` returns 5 at HEAD `774df9f`, matching the plan's claim. The dev-flagged drift does not exist; the plan's prose IS accurate.

**Rationale:** Round 3 of the plan applies the single-line fix Round-2 falsification flagged. Verification:
- The W5.D1 inline `**Paths:**` summary at line 161 (post-fix) names the same 5 caller-audit sites as L1 header line 5 + W5.D1 audit-detail block lines 168-176 + HF6 removal note line 176 — internal consistency restored.
- The 5 caller-audit sites are independently verified at HEAD `774df9f` via `git grep -c` (counts 1, 7, 1, 1, 1 respectively) and the over-claimed pair (`auto_generate_steward.go`, `template_service.go`) verified to have zero hits.
- Round-3 edit is scoped to PLAN.md only — `_BLOCKERS.toml` untouched, graph summary untouched, no collateral damage.
- The dev-flagged "minor pre-existing prose drift" at line 174 turns out to be a false alarm — the actual `git grep -c` count at HEAD is 5, matching the plan exactly.
- Re-ran cycle check, blockers-drift check, sibling-overlap-without-blocker check — all clean.
- Re-ran 7-family attack pass with concrete-repro discipline — every family either has its Round-1+Round-2 findings already-fixed (A1, A3) or no concrete counterexample to construct (A2, A4, A5, A6, A7).

The plan is ready for build-phase entry pending parallel proof-side Round-3 verdict.

| Family | Result |
| --- | --- |
| A1 concurrency-blocked_by | **REFUTED** — Round-1+Round-2 high-severity findings all fixed; `internal/templates` chain intact; cycle-check clean; `_BLOCKERS.toml`/PLAN.md zero-drift verified. |
| A2 contract-mismatch | **EXHAUSTED, no counterexample** — line 161 ↔ line 5 ↔ lines 168-176 ↔ line 176 cross-block consistency restored by Round-3 fix; KindPayload `<4 caller audit sites>` placeholder is provenance prose, not a contract violation. |
| A3 hidden-coupling | **EXHAUSTED, no counterexample** — Round-2 FF2.1 fixed; every concrete file:line citation in W5.D1's prose verified independently against `git grep` at HEAD `774df9f`. |
| A4 yagni-scope-creep | **EXHAUSTED, no counterexample** — Round-3 edit is a single-line prose replacement; no new abstraction or field. |
| A5 shipped-but-not-wired | **EXHAUSTED, no counterexample** — Round-2 HF8 wiring contract unchanged; W0.5 known-wired-set deferral unchanged. |
| A6 atomicity | **EXHAUSTED-low** — W1.D1 24-file count KEPT as-authored per dev disposition; Round-3 doesn't touch decomp shape. |
| A7 prompt-injection | **EXHAUSTED, DORMANT** — pre-team-feature per `feedback_prompt_injection_team.md`. |

## 4. Hylla Feedback

N/A — review touched non-Go artifacts only (PLAN.md, `_BLOCKERS.toml`, SKETCH.md, Round-1 + Round-2 verdict files). Hylla is Go-only today; no Hylla queries were made or required. The Go-source verifications used `git grep` against HEAD `774df9f` for: `default-go.toml` per-file counts (matched plan claims at all 5 caller-audit sites + zero on the 2 over-claimed sites + 5 in embed.go matching the prose claim + 3 in load.go matching the prose claim + 2 in load_test.go matching the prose claim + 8 in default-generic.toml matching the line-list claim), `type BindingResolved` (single result confirming HF4 location at `cli_adapter.go:102` still holds). Round-3's Go-side verifications were single-grep-then-line-number-check — within `git grep`'s sweet spot. No Hylla misses to report; no fallbacks; no ergonomic gripes.
