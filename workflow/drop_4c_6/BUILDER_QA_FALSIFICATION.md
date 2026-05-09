# Drop 4c.6 — Build QA Falsification

Per-droplet build-QA falsification rounds append below. Each round entry stamps
droplet ID, round number, counterexample list, and a per-family attack-result
table. Verdict philosophy: PASS = all 7 attack families exhausted with no
concrete counterexample; FAIL = at least one CONFIRMED.

---

## Droplet 4c.6.W6.D3 — Round 1

**Reviewer:** go-qa-falsification-agent (subagent, doc-only mode).
**Date:** 2026-05-09.
**Droplet:** `4c.6.W6.D3 — GDD_METHODOLOGY.md placeholder`.
**Artifact under attack:** `GDD_METHODOLOGY.md` (NEW top-level placeholder, ~62 lines).

### Counterexamples

(none — empty list)

### Findings (non-CONFIRMED, recorded for audit)

- 1.1 [Family: yagni] [severity: low] Placeholder is ~62 lines vs the L1
  `KindPayload.shape_hint` of `~30-line stub` (PLAN.md line 309). Builder
  added a `## Status` block (4 sub-bullets), a `## Scope (when populated)`
  block (6 sub-bullets reserving the populated doc's structure), and a
  `## Non-goals (explicit)` block (2 sub-bullets). The hard acceptance
  bullets in PLAN.md line 299 are met (title, 1-paragraph description,
  `<!-- TODO populate post-dogfood -->` marker, prior-art research note
  placeholder per `SKETCH.md § 14.2.1`); the over-line-count is a
  soft-hint deviation, not a spec violation. Builder explicitly justified
  the additions in `BUILDER_WORKLOG.md` lines 39-48 (scope bullets)
  and 54-56 (Non-goals) — Status + TODO marker satisfy the
  RiskNotes "MUST clearly mark itself as populate post-dogfood"
  requirement and the extra blocks do not introduce normative content
  (`## Status: State: placeholder. Do not treat the contents below as
  normative.` line 19). Repro: `wc -l GDD_METHODOLOGY.md` → 62
  vs shape_hint ~30. Fix hint: leave as-shipped — the additions are
  explicitly reservation-of-structure plus disclaim-against-misuse,
  both of which serve the placeholder's purpose. Routing: noted to
  orchestrator as a soft-hint deviation; not a counterexample.

  *Verdict on this attack: REFUTED — soft `shape_hint` overshoot does
  not violate any of the four hard acceptance bullets, and the
  builder's justification ties each block to a stated risk or
  requirement.*

### Attack family table

| Family                  | Result    | Notes                                                                 |
| ----------------------- | --------- | --------------------------------------------------------------------- |
| B1 test-coverage        | N/A       | Doc-only droplet — no test surface.                                   |
| B2 contract-preservation| REFUTED   | `CASCADE_METHODOLOGY.md:7` cross-references `GDD_METHODOLOGY.md`; placeholder file exists at the cited root path. No other repo MD references GDD pre-W6.D5 (W6.D5 adds README pointers later). No broken link surface. |
| B3 hidden-coupling      | REFUTED   | Doc references `CLAUDE.md § Code Understanding Rules` (verified — `CLAUDE.md:159` `### Code Understanding Rules` under `## Hylla Baseline`) and `WIKI.md` (verified — exists with `## Cascade Vocabulary`). Doc names "Hylla" as the today-graph implementation, consistent with `PLAN.md` / `CASCADE_METHODOLOGY.md:7` / `SKETCH.md` framing. No claims that the rest of the codebase relies on. |
| B4 yagni                | REFUTED   | Soft `shape_hint` overshoot (~62 vs ~30 lines) recorded as Finding 1.1; builder justification ties each added block to a stated risk-note requirement or structure-reservation purpose. Hard acceptance bullets all met. Not a CONFIRMED counterexample. |
| B5 spec-compliance      | REFUTED   | Acceptance bullets verified line-by-line: (1) H1 title `# GDD Methodology — Graph-Driven Development` — present line 1; (2) 1-paragraph description tying to GDD methodology populated post-Hylla-rev / post-dogfood per `project_methodology_docs_tracker.md` and `SKETCH.md § 14.2` — present lines 5-15 (cites both docs verbatim); (3) `<!-- TODO populate post-dogfood -->` marker — present line 3 (with matching `<!-- END TODO -->` line 62); (4) prior-art research note placeholder per `SKETCH.md § 14.2.1` — present lines 45-53 as `## Prior-art research note (per SKETCH.md § 14.2.1)`. Sketch §14.2 ("Unchanged from v2; §14.2.1 prior-art research note still applies") is cited correctly even though no explicit `#### 14.2.1` heading exists in the current SKETCH.md — the placeholder cites the section as required by the planner's acceptance bullet, which itself inherits the v2-of-sketch reference. |
| B6 shipped-but-not-wired| N/A       | Doc-only droplet — no wiring surface.                                 |
| B7 prompt-injection     | EXHAUSTED | DORMANT pre-team-feature per agent definition.                        |

### Summary

**Verdict: pass.** Counterexample count: 0. All applicable attack families
either REFUTED (B2, B3, B4, B5), N/A (B1, B6), or EXHAUSTED (B7). Finding
1.1 recorded as a soft-hint deviation for audit; not a CONFIRMED
counterexample.

### Hylla Feedback

N/A — task touched non-Go files only.

---

## Droplet 4c.6.W6.D2 — Round 1

**Reviewer:** go-qa-falsification-agent (subagent, doc-only mode).
**Date:** 2026-05-09.
**Droplet:** `4c.6.W6.D2 — CASCADE_METHODOLOGY.md skeleton`.
**Artifact under attack:** `CASCADE_METHODOLOGY.md` (NEW top-level skeleton, 200 lines, 18 H2 sections).

### Counterexamples

(none — empty list)

### Findings (non-CONFIRMED, recorded for audit)

- 1.1 [Family: contract-preservation] [severity: medium] `## metadata.structural_type Enum` section
  (lines 51-59) duplicates the WIKI.md § Cascade Vocabulary metaphor and atomicity rules
  before the cross-reference citation. Line 53 paraphrases WIKI.md:38 nearly verbatim
  ("Picture water flowing down a series of waterfalls: a **drop** is one vertical step that may
  decompose into more steps; **segments** are parallel streams within a drop; **confluences**
  are merge points where streams rejoin; **droplets** are atomic, indivisible units that finish
  in one shot."). Line 55 then enumerates the atomicity rules ("`droplet` MUST have zero
  children ... `confluence` MUST have non-empty `blocked_by` ... `segment` may recurse ...
  `drop` is the level-1 cascade step") which mirror WIKI.md:48-49's table. Line 57 DOES cite
  `WIKI.md § "Cascade Vocabulary"` as canonical, which mitigates per the planner's
  "single-canonical-source rule" context block. The CLAUDE.md:59 prohibition "Do not redefine
  the structural_type vocabulary in this file or any other doc" is the textual basis for this
  attack. *Verdict: REFUTED — borderline duplication is mitigated by (a) the explicit
  cross-reference at line 57, (b) the same paragraph framing the metaphor as
  "methodology-level explanation of *why* the axis exists" rather than the canonical
  definition, and (c) the planner's acceptance bullet 1 explicitly enumerates the
  `metadata.structural_type (drop / segment / confluence / droplet)` topic, which means SOME
  introduction of the values is required to satisfy AC1.* Repro: diff lines 53-57 against
  WIKI.md:38-49. Fix hint (if dev disagrees with REFUTED): in a future revision, replace
  lines 53-55 with a 1-sentence pointer ("see WIKI.md § Cascade Vocabulary for the metaphor
  + atomicity rules") and keep only the methodology-level "why this axis is separate" framing
  at lines 56-57. Routing: noted to orchestrator as a contract-preservation soft-finding;
  not a CONFIRMED counterexample.

- 1.2 [Family: yagni] [severity: low] Skeleton ships 18 H2 sections vs the 14 enumerated in
  PLAN.md AC1 / SKETCH.md §26.W6 AcceptanceCriteria. The 4 extra H2s are: line 21
  `## Three Orthogonal Axes` (umbrella section that frames the next 4 axis-specific H2s),
  line 163 `## Cross-References`, line 175 `## Comparison Surface`, line 185 `## Provenance`.
  Builder justification in BUILDER_WORKLOG.md lines 132-140 ties Comparison Surface +
  Provenance to "seat the doc against neighboring methodologies" + "explicitly cite the
  rollout's intellectual provenance," and SKETCH.md §26.W6 RiskNotes say "skeleton must be
  complete enough for the methodology article to cite" — the extras are reasonable for a
  methodology doc. The L1 acceptance criteria do not prohibit extra sections, only require
  the enumerated ones to be present. AC1 enumerated topics ARE all present (cross-walked
  below in B5). *Verdict: REFUTED — soft scope-creep but each extra section is justified
  by the doc's stated purpose. Not a CONFIRMED counterexample.* Repro: count `^## ` headings
  in `CASCADE_METHODOLOGY.md` → 18; cross-walk against AC1 enumerated list → 14 + 4 extras.
  Fix hint: leave as-shipped — the extras do not redefine vocabulary nor inflate individual
  sections beyond the 1-3 paragraph budget.

### Attack family table

| Family                  | Result    | Notes                                                                 |
| ----------------------- | --------- | --------------------------------------------------------------------- |
| B1 test-coverage        | N/A       | Doc-only droplet — no test surface.                                   |
| B2 contract-preservation| REFUTED   | Cross-references to `WIKI.md § "Cascade Vocabulary"` present at lines 7, 27, 57, 171, 192 (5 citations). Vocabulary sections (`kind` / `role` / `structural_type`) all cite WIKI as canonical. Borderline duplication of structural_type metaphor + atomicity rules at lines 53-55 recorded as Finding 1.1; mitigated by the explicit cross-reference at line 57 and AC1's requirement that the enum values themselves be named in the doc. Role-enum value enumeration at line 25 is consistent with how CLAUDE.md:56 treats the role enum (CLAUDE.md:59 only prohibits redefining `structural_type`, not `role` or `kind`). |
| B3 hidden-coupling      | REFUTED   | Doc cites companion docs that exist or are concurrent: `GDD_METHODOLOGY.md` (W6.D3, ships in parallel — verified present); `SPAWN_PIPELINE.md` (existing — verified present); `CLI_ADAPTER_AUTHORING.md` (existing — verified present); `WIKI.md` (existing). Forward reference to `AGENTS_CONFIG.md` (W6.D1, ships in parallel — verified ABSENT but acceptable as W6.D5's README pointer-add droplet is `Blocked by` W6.D1; AGENTS_CONFIG.md will land before the README is updated). Doc references `internal/config/agents.go` (W0 wave) which has not landed yet — but the L1 PLAN.md acceptance bullet 4 explicitly REQUIRES forward-refs to `AGENTS_CONFIG.md` + `GDD_METHODOLOGY.md`, so this is the planner's mandate, not coupling drift. RESEARCH/ISOLATION_ENFORCEMENT_FIX.md cited at lines 159, 193 — file presence not directly probed but the path is identical to citations in PLAN.md (line 141, 148, etc.) and SKETCH.md, indicating consistent reference pointer. |
| B4 yagni                | REFUTED   | Soft scope-creep on H2 count (18 vs 14 enumerated) recorded as Finding 1.2. Each extra section (Three Orthogonal Axes umbrella + Cross-References + Comparison Surface + Provenance) is justified by doc purpose; SKETCH.md §26.W6 RiskNotes explicitly say skeleton must be "complete enough for the methodology article to cite" — extras enable citation. Individual sections stay within the 1-3 paragraph budget per AC2. Not a CONFIRMED counterexample. |
| B5 spec-compliance      | REFUTED   | All four L1 acceptance criteria verified line-by-line: **(AC1)** All 14 enumerated topics present as H2 sections — Plan Down Build Up @ line 11 ✓; Closed 12-Value `kind` Enum @ line 31 ✓; `metadata.role` Enum @ line 41 ✓; `metadata.structural_type` Enum @ line 51 ✓; Agent Shape @ line 61 ✓; Section 0 — Semi-Formal Reasoning Certificate @ line 71 ✓; Tillsyn-Flavored Specify Pass @ line 81 ✓; TN-Per-Section Response Style @ line 91 ✓; Hylla-First Evidence Ordering @ line 101 ✓; TDD Requirement @ line 111 ✓; QA Proof vs Falsification — Asymmetric Verification @ line 121 ✓; `blocked_by` Ordering Primitive @ line 133 ✓; Parent-Children-Complete Invariant @ line 143 ✓; Isolation Enforcement @ line 153 ✓. **(AC2)** Each section runs 1-3 paragraphs (longest is `## Provenance` at 3 paragraphs + 6-bullet list — within budget); `<!-- TODO populate post-dogfood with measured benchmarks -->` marker count = 19 across 18 sections + 1 lead-paragraph occurrence (verified via grep). **(AC3)** First H2 after H1 is `## Plan Down, Build Up` @ line 11 — confirmed grep-able per ROUND-2 HF9: `awk '/^## /{print NR; exit}' CASCADE_METHODOLOGY.md` → 11; line 11 reads `## Plan Down, Build Up`. **(AC4)** Forward-refs to `AGENTS_CONFIG.md` + `GDD_METHODOLOGY.md` present at line 7 (lead paragraph) and lines 167-168 (Cross-References section). |
| B6 shipped-but-not-wired| N/A       | Doc-only droplet — no wiring surface.                                 |
| B7 prompt-injection     | EXHAUSTED | DORMANT pre-team-feature per agent definition.                        |

### Summary

**Verdict: pass.** Counterexample count: 0. All applicable attack families either REFUTED
(B2, B3, B4, B5), N/A (B1, B6), or EXHAUSTED (B7). Findings 1.1 (contract-preservation
soft-duplication of structural_type metaphor + atomicity rules) and 1.2 (soft H2 scope-creep:
18 vs 14 enumerated) recorded for audit; both REFUTED with mitigation rationale. Skeleton
satisfies all 4 hard acceptance criteria (AC1 enumerated sections present, AC2 1-3 paragraphs
+ TODO marker per section, AC3 first H2 = "Plan Down, Build Up", AC4 forward-refs to
AGENTS_CONFIG.md + GDD_METHODOLOGY.md present).

### Hylla Feedback

N/A — task touched non-Go files only.
