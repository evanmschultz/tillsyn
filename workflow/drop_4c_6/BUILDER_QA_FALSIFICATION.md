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

---

## Droplet 4c.6.W6.D1 — Round 1

**Reviewer:** go-qa-falsification-agent (subagent, doc-only mode).
**Date:** 2026-05-09.
**Droplet:** `4c.6.W6.D1 — AGENTS_CONFIG.md (new top-level doc)`.
**Artifact under attack:** `AGENTS_CONFIG.md` (NEW top-level adopter-facing reference, 396 lines, 12 numbered H2 sections + ToC).

### Counterexamples

(none — empty list)

### Findings (non-CONFIRMED, recorded for audit)

- 1.1 [Family: contract-preservation] [severity: low] **Dangling `L20` reference at line 62.**
  The `[agents]` schema example renders `auto_push = false` with the trailing comment
  `# post-build commit-and-push gate (off by default per L20)`. `L20` is not defined or
  referenced anywhere else in `AGENTS_CONFIG.md`, in `workflow/drop_4c_6/SKETCH.md`
  (verified via `git grep -nE "L20" workflow/drop_4c_6/SKETCH.md` → 0 hits), or in any
  other doc at repo root. SKETCH.md actually addresses `auto_push` at lines 189
  (`auto_push = false` example) and 632 (`21.2 auto_push location — AGREED: agents.toml
  defaults block`) — neither carries an `L20` label. The reference appears to be a stale
  pointer to a sketch sub-numbering scheme that was reorganized. Repro:
  `git grep -nE "L20" AGENTS_CONFIG.md workflow/drop_4c_6/SKETCH.md` → only the line-62
  hit in AGENTS_CONFIG.md, no source for the label. Fix hint: replace `per L20` with
  either `per SKETCH.md § 21.2` (the AGREED locator) or simply drop the parenthetical —
  the `# off by default` comment already conveys the default.
  *Verdict: REFUTED — dangling reference is a soft cosmetic blemish in an inline TOML
  comment, not a B2 contract drift on a Go symbol or schema field. The accompanying
  `auto_push = false` schema is correct and matches `Preset.AutoPush` in the shipped
  `internal/config/agents.go:171` and the `agents.example.toml:35` default. Routing:
  noted to orchestrator as a low-severity audit-trail item; not a CONFIRMED counterexample.*

- 1.2 [Family: contract-preservation] [severity: medium] **`agents/` directory layout
  drift at line 33.** §1 line 33 says `agents.toml` lives "at the project root (the same
  directory as `.tillsyn/`, the project's `agents/` directory, and the `.gitignore`...)."
  This phrasing implies a sibling `agents/` directory adjacent to `.tillsyn/` at the
  project root. The actual file-layout contract per `SKETCH.md` lines 101, 117, 140, 147
  is `<project>/.tillsyn/agents/<name>.md` — agent files live in `agents/` **inside**
  `.tillsyn/`, not next to it. The W2 PLAN.md scope (line 121) confirms `till init`
  copies embedded agent files to `<project>/.tillsyn/agents/*.md` FLAT — there is no
  top-level `<project>/agents/` directory. Repro:
  `git grep -nE "\.tillsyn/agents/|<project>/agents/" workflow/drop_4c_6/SKETCH.md` →
  every hit shows `.tillsyn/agents/`, no hits show a sibling `agents/`. Fix hint:
  rewrite line 33 to "at the project root (the same directory as `.tillsyn/` (which
  contains the `agents/` subdirectory) and the `.gitignore`...)" or simpler: drop the
  `agents/` reference from §1 line 33 entirely — §1 is about `agents.toml` location,
  not the agent-files directory layout, and adopters reading §1 don't need that detail
  yet.
  *Verdict: REFUTED — phrasing ambiguity rather than a hard structural drift.
  The doc never explicitly claims a top-level `<project>/agents/` directory exists or
  is consumed; the prose "the project's `agents/` directory" is parseable as either
  "next to `.tillsyn/`" or "the `agents/` subdirectory the project owns." The latter
  reading is consistent with SKETCH. Falsification did NOT find a downstream consumer
  in the doc that depends on the wrong reading. Routing: noted to orchestrator as a
  medium-severity wording finding worth tightening on next pass; not a CONFIRMED
  counterexample because no doc claim downstream depends on the misreading.*

- 1.3 [Family: hidden-coupling] [severity: medium] **Forward-looking present-tense
  claims about W3-not-yet-shipped wiring.** §7 (Frontmatter Strip Behavior) line 232
  says "the **frontmatter strip helper** `StripFrontmatterKeys` ... removes these keys
  from the frontmatter that lands in the bundle's `<bundle>/plugin/agents/<name>.md`,"
  and line 238 says "The render layer in the spawn pipeline calls it once per spawn
  during bundle assembly." Both are present-tense claims about wiring that has NOT
  shipped — `StripFrontmatterKeys` is implemented in `internal/config/frontmatter.go`
  but is not yet called from `internal/app/dispatcher/cli_claude/render/render.go`
  (`assembleAgentFileBody` at line 340 is the pre-W3 stub per PLAN.md W3 Scope, line
  141, and the PLAN.md W3.D5 acceptance bullet explicitly says the wiring lands in W3
  not W6.D1). Verified via `git grep -nE "StripFrontmatterKeys" internal/app/`
  → zero hits in the dispatcher path. Similarly §8 (claude_md_addons) line 246 says
  "Tillsyn loads at spawn time and **concatenates onto the agent's system prompt**" —
  but no consumer of `AgentRuntime.ClaudeMDAddons` is wired anywhere outside the
  schema + Resolve / MergeLocal layer (verified via
  `git grep -nE "ClaudeMDAddons" internal/` → only schema + test references in
  `internal/config/`, no render-layer or dispatcher consumer). Both sections describe
  forward-shipped W3 behavior in present tense, which would mislead an adopter who
  reads `AGENTS_CONFIG.md` against the W6.D1-shipped HEAD and tries `claude_md_addons`
  expecting it to flow through.
  *Verdict: REFUTED — W6.D1's `Blocked by: 4c.6.W0` (PLAN.md line 258) only requires
  W0's schema-level types to land before this doc; PLAN.md W6.D1 acceptance criterion 1
  (line 255) lists "frontmatter strip behavior (§4.4)" as a required topic. The doc
  was authored by spec to describe the **end-state** behavior — frontmatter strip and
  `claude_md_addons` consumption ARE part of the schema's stated semantics that adopters
  configure their `agents.toml` against, even before W3 wires the consumers. The doc
  does NOT promise W3 has shipped (no "as of HEAD `<sha>`" claim); it describes the
  intended runtime contract. Acceptable per the planner's mandate that this doc be the
  "single source for the question 'how do I configure my agents per-machine.'" Routing:
  noted to orchestrator with a fix-hint suggestion that §7 line 238 + §8 line 246
  could add a one-sentence pre-W3 caveat ("wired in Drop 4c.6 W3"); not required for
  PASS, since W6.D5's README pointer-add is `Blocked by` W6.D1 + W6.D2 + W6.D3 only,
  and the cascade methodology articulates that adopters track HEAD by drop, not by
  intermediate droplet.*

- 1.4 [Family: yagni] [severity: low] **396 lines vs 200-line acceptance floor — not
  scope creep.** L1 acceptance bullet 1 (PLAN.md line 255) requires "≥ 200 lines" and
  enumerates 7 topical sections (schema, override semantics, env_set vs env_from_shell,
  tools_allow vs tools_deny, frontmatter strip, claude_md_addons, worked examples).
  The doc ships 12 numbered H2 sections — 7 enumerated + 5 closing (§ 1 File Locations,
  § 4 Override Semantics two-layer merge, § 10 Error Handling, § 11 Validation Rules,
  § 12 Implementation Notes). The 5 closing sections are NOT explicitly enumerated in
  the L1 acceptance bullet but cover load-bearing topics for adopters: §1 file locations
  + resolution order is prerequisite reading for §2-§9 to make sense; §4 override
  semantics is split out from §3 schema for two-layer-merge clarity; §10 ConfigError
  envelope is the inspection contract for `errors.Is(err, ErrToolsDenyNotOverridable)`
  which §6 references (without §10 the reader has no `errors.Is` recipe); §11
  validation rules consolidate the fail-loud-at-load-time semantics scattered across
  §1-§9; §12 implementation notes name the shipped Go API surface. None of the 5 extra
  sections introduce schema fields beyond `Preset` / `Override` / `AgentRuntime` / the
  three sentinel/helper symbols. Repro: `awk '/^## /{print NR": "$0}' AGENTS_CONFIG.md`
  → 12 H2 sections. Cross-walk: AC1 7 enumerated topics ALL present (§2+§3 schema,
  §4 override semantics, §5 env_set vs env_from_shell, §6 tools_allow vs tools_deny,
  §7 frontmatter strip, §8 claude_md_addons, §9 worked examples). Fix hint: leave
  as-shipped — the 5 extras serve adopter pedagogy, not bloat. Builder explicitly
  justified §10 / §11 / §12 in BUILDER_WORKLOG.md lines 198-202 ("`Error Handling
  — *ConfigError Envelope` section the acceptance bullet does not enumerate but which
  is load-bearing for adopters to inspect rejections").
  *Verdict: REFUTED — line count is well above the 200-line floor by deliberate
  pedagogy choices, not scope creep. Each extra section ties to a load-bearing reader
  contract.*

### Attack family table

| Family                  | Result    | Notes                                                                 |
| ----------------------- | --------- | --------------------------------------------------------------------- |
| B1 test-coverage        | N/A       | Doc-only droplet — no test surface.                                   |
| B2 contract-preservation| REFUTED   | Every cited Go symbol verified verbatim against shipped `internal/config/`: `Preset` (`agents.go:162`), `Override` (`agents.go:189`), `AgentRuntime` (`agents.go:211`), `AgentsRegistry` (`agents.go:242`), `ConfigError` (`agents.go:89`), `ErrToolsDenyNotOverridable` (`agents.go:36`), `LoadRegistry` (`agents.go:292`), `MergeLocal` (`agents.go:533`), `Resolve` (`agents.go:385`), `StripFrontmatterKeys` (`frontmatter.go:89`), `localPathLabel` (`agents.go:43`), `deterministicKindOrder` (`agents.go:51`). Field-by-field correspondence table (§2) matches `Preset`'s 15 fields exactly. `pelletier/go-toml/v2` import + `DisallowUnknownFields()` claim verified at `agents.go:23` + `agents.go:300`. The closed 12-value `kind` enum cited at §3 line 96 enumerates all 12 kinds; order differs from `validKinds` in `internal/domain/kind.go:34-47` (`plan, research, build, ...` shipped vs `plan, build, research, ...` doc) but order is not contract — `domain.IsValidKind` walks the slice via `slices.Contains`. Cross-references to `CASCADE_METHODOLOGY.md`, `SPAWN_PIPELINE.md`, `CLI_ADAPTER_AUTHORING.md`, `WIKI.md § "Cascade Vocabulary"` all verified present at repo root and the WIKI section verified at `WIKI.md:36`. Two soft findings (1.1 dangling `L20` ref + 1.2 `agents/` directory phrasing) recorded for audit; both REFUTED with mitigation. |
| B3 hidden-coupling      | REFUTED   | Worked examples (§9) cite `Preset` schema fields verbatim — `model`, `env_set`, `env_from_shell` — all match shipped struct. Provider env-var names (`ANTHROPIC_BASE_URL`, `ANTHROPIC_BEDROCK_BASE_URL`, `ANTHROPIC_VERTEX_PROJECT_ID`, `CLOUD_ML_REGION`, `GOOGLE_APPLICATION_CREDENTIALS`, etc.) are CLI-side contracts the doc explicitly disclaims Tillsyn validates ("Tillsyn never validates the model name, endpoint URL, or API-key value — only the schema shape" — §5 line 192, §9 line 325). The doc's structure-vs-semantic split is the right contract: schema shape gated, value semantics deferred to provider. Forward-looking present-tense claims about W3-not-yet-shipped wiring (§7 + §8) recorded as Finding 1.3; REFUTED because PLAN.md line 255 explicitly requires both topics, the doc describes runtime contract not HEAD state, and W6.D1 is `Blocked by` only W0 not W3. |
| B4 yagni                | REFUTED   | 396 lines vs 200-line acceptance floor recorded as Finding 1.4. 12 H2 sections vs 7 enumerated topics: every enumerated topic present + 5 closing sections (file locations, override semantics, ConfigError envelope, validation rules, implementation notes) each tied to load-bearing adopter pedagogy. No abstractions invented; doc is descriptive of shipped W0 + intended W3 reality. Three closing sections (§10 / §11 / §12) sequentially: §10 documents the inspection-contract (`errors.Is`) §6 references; §11 consolidates fail-loud-at-load-time invariants; §12 lists the Go API surface for adapter authors. Each justified in BUILDER_WORKLOG.md design-decision section. Not scope creep. |
| B5 spec-compliance      | REFUTED   | All four L1 acceptance bullets verified line-by-line: **(AC1)** ≥ 200 lines: 396 actual ✓; sections present: schema (§2 + §3 — `[agents]` + `[agents.<kind>]` ✓), override semantics (§4 — two-layer merge ✓), `env_set` vs `env_from_shell` (§5 ✓), `tools_allow` vs `tools_deny` override scope (§6 ✓), frontmatter strip behavior (§7 ✓), `claude_md_addons` (§8 ✓), worked examples for Bedrock / Vertex / OpenRouter / Ollama Cloud (§9 — five examples including Anthropic-direct + the four named providers ✓). **(AC2)** Cross-references to `CASCADE_METHODOLOGY.md` (lines 7, 396), `SPAWN_PIPELINE.md` (lines 8, 42, 232, 386, 396), `CLI_ADAPTER_AUTHORING.md` (lines 9, 386, 396), `WIKI.md § "Cascade Vocabulary"` (lines 10, 142, 396) — all four references present and target paths verified. **(AC3)** `mage ci` — not run by builder per W6.D1 doc-only convention; runs at drop end per WORKFLOW.md Phase 4. |
| B6 shipped-but-not-wired| N/A       | Doc-only droplet — no shipped wiring surface (the `StripFrontmatterKeys` + `claude_md_addons` consumer-not-wired observations at Finding 1.3 belong to W3 wave, not W6.D1's surface). |
| B7 prompt-injection     | EXHAUSTED | DORMANT pre-team-feature per agent definition.                        |

### Summary

**Verdict: pass.** Counterexample count: 0. All applicable attack families either REFUTED
(B2, B3, B4, B5), N/A (B1, B6), or EXHAUSTED (B7). Findings 1.1 (dangling `L20` reference,
low), 1.2 (`agents/` directory phrasing ambiguity, medium), 1.3 (forward-looking present-tense
W3-wiring claims in §7 + §8, medium), and 1.4 (line count + H2 count above acceptance floor,
low) recorded for audit; all REFUTED with mitigation rationale. Doc satisfies all hard L1
acceptance bullets: every cited Go symbol resolves verbatim in `internal/config/`, every
required topic from the 7 enumerated AC1 sections appears, and every cross-reference target
exists at repo root.

### Hylla Feedback

N/A — task touched non-Go files only.

---

## Droplet 4c.6.W1.D1 — Round 1

**Reviewer:** go-qa-falsification-agent (subagent, sonnet).
**Date:** 2026-05-09.
**Droplet:** `4c.6.W1.D1 — Scaffold embedded agent dirs (placeholder content) + ship agents.example.toml`.
**Artifacts under attack:** 28 placeholder agent .md files under `internal/templates/builtin/agents/till-{gen,go,gdd}/`, `internal/templates/builtin/agents.example.toml`, `//go:embed` directive expansion in `internal/templates/embed.go`, `TestDefaultTemplateFSEmbedsPlaceholderAgentFiles` in `internal/templates/embed_test.go`. Primary attack vector: 6-FILE SCOPE EXPANSION beyond L1 W1.D1's 21-placeholder acceptance bullet (5 legacy `go-*` names in `till-go/` + `orchestrator-managed.md` in `till-gen/`).

### Counterexamples

(none — empty list)

### Findings (non-CONFIRMED, recorded for audit)

- 1.1 [Family: B3 hidden-coupling] [severity: low] **Cross-droplet bridge has correct shape but is asymmetric in the audit trail.** PLAN.md W1.D1 acceptance bullet 1 (line 96) explicitly says "21 placeholder agent .md files shipped (3 groups × 7 standard names)" — it does NOT enumerate the 6 EXTRAS. The bridge to W0.5 (`embeddedAgentLibraryShipped` flips strict-mode at package init the moment any agent .md ships per `internal/templates/load.go:2110-2123`) and the bridge to W5.D3 (will delete the 6 extras alongside the `agent_name` strip in `default-go.toml`/`till-go.toml` per PLAN.md line 230) are documented in BUILDER_WORKLOG.md lines 350-373 + `internal/templates/embed.go:44-58` doc-comment, but PLAN.md W1.D1's `Paths:` line (PLAN.md line 93) does NOT enumerate the 6 extras either. Reproduction: read PLAN.md line 93's `Paths:` field, then list `internal/templates/builtin/agents/till-go/` — `Paths` lists `same 7 names under till-go/` (+ till-gdd, till-gen) = 21, but disk shows 12 files in till-go/ (7 standard + 5 legacy). Asymmetry: builder correctly chose the least-disruptive resolution (ship the 6 to keep `LoadDefaultTemplateForLanguage("go")` loading) and explicitly documented it in WORKLOG, but PLAN.md was never amended to enumerate the 6 extras in `Paths:`. Fix hint: this is a planner-level finding not a build-level finding — escalate to ancestor re-QA per WORKFLOW.md Phase 5 step 7. Severity LOW because (a) the `Paths` declaration is informational pre-Drop-1 (Tillsyn's `paths` field hard-locks post-Drop-1), (b) the WORKLOG audit trail is complete, (c) the embed.go doc-comment carries the cross-droplet bridge note, (d) `mage test-pkg ./internal/templates` is 458/458 GREEN. REFUTED as a build-QA counterexample; routed to orchestrator as a planner-spec accuracy refinement candidate for round-2 PLAN.md if the orchestrator chooses to amend.
- 1.2 [Family: B4 YAGNI] [severity: low] **Alternative orderings considered.** (a) "W1.D1 ships 21, breaks tests" — W0.5 already shipped strict-mode validator with `embeddedAgentLibraryShipped` probe at `load.go:2110`; shipping only 21 would have left 5 legacy names + `orchestrator-managed` unresolvable, breaking every `LoadDefaultTemplateForLanguage("go")` test (worklog lines 387-391: mid-build the package returned 347/406 — exactly that breakage). (b) "W1.D1 ships 28 (chosen)" — preserves test green, defers cleanup to W5.D3, no scope creep into other droplets' `paths`. (c) "W5.D3 reorders before W1.D1" — would require renaming `default-go.toml` → `till-go.toml` (W5.D1 territory) AND stripping `go-` from agent_name AND deleting `tools` frontmatter all before any agent .md ships into the embed.FS; the planner-side blocker chain orders W5.D3 → {W5.D1, W5.D2, W1.D1} (PLAN.md line 233) — reversing that chain orphans W5.D3's "edits the placeholder agent .md files from W1" precondition. (d) "W1.D1 updates default-go.toml itself" — touches PLAN.md W5.D1's `Paths:` (default-go.toml is W5.D1's file), violates the file-blocking rule (sibling droplets sharing a path need explicit `blocked_by`). Builder picked (b) — the only option that doesn't crash a sibling. Verdict on alternatives: chosen path is correct. REFUTED.
- 1.3 [Family: B5 spec-compliance] [severity: low] **Acceptance-bullet enumeration is precise on 21+1 but loose on 6 extras.** PLAN.md W1.D1 acceptance bullet 4 (line 99): "embed_test.go adds an FS-introspection test asserting all 21 placeholder paths + agents.example.toml resolve via DefaultTemplateFS.Open." Reading `embed_test.go:1058-1119` confirms the test enumerates exactly 21 standard paths (3 × 7 from `w1d1AgentGroups` × `w1d1StandardAgentNames`) + 1 agents.example.toml = 22 distinct files. Test framework reports 23 because Go test counts the parent + 22 sub-tests (`mage test-func ./internal/templates TestDefaultTemplateFSEmbedsPlaceholderAgentFiles` → 23/23 GREEN, confirmed). The 6 extras are NOT covered by `TestDefaultTemplateFSEmbedsPlaceholderAgentFiles`; their existence is asserted only indirectly via the `defaultAgentLookupFn` walk during `LoadDefaultTemplateForLanguage("go")` (which `mage test-pkg ./internal/templates` exercises to 458/458 GREEN). Fix hint: when W5.D3 deletes the 6 extras, the `mage test-pkg` integration tests will catch any deletion-vs-rename mismatch — no test gap to plug here. REFUTED.
- 1.4 [Family: B1 test-coverage] [severity: low] **Edge-case rigor is light but adequate.** `TestDefaultTemplateFSEmbedsPlaceholderAgentFiles` does NOT cover (a) "non-existent name" (e.g. `DefaultTemplateFS.Open("builtin/agents/till-go/typo-agent.md")` — would return embed.FS error), (b) "wrong group" (e.g. `till-rust/`), (c) "empty file" (the PLACEHOLDER marker check catches a fully empty file via `strings.Contains(body, "PLACEHOLDER")` returning false, but does not separately exercise zero-byte file shape). These are all programmer-error / future-drift cases rather than acceptance-criteria gaps. The W5.D3 deletion path is the relevant adversarial case for the 6 extras and is not in scope here. REFUTED — edge-case coverage is adequate for the specific shipped contract; logged as a refinement candidate for any future TDD pass that adds defense-in-depth assertions.
- 1.5 [Family: B2 contract-preservation] [severity: low] **PLACEHOLDER marker discipline + frontmatter shape both hold.** Spot-checked 4 files via `Read`: `till-gen/builder-agent.md` (10 lines, frontmatter `name`+`description`, body `# PLACEHOLDER — substantive content lands in Drop 4c.8 W4`), `till-go/go-builder-agent.md` (13 lines, same pattern with legacy-bridge note in description), `till-gen/orchestrator-managed.md` (15 lines, coordination-kind explanatory note + PLACEHOLDER body), `till-gdd/closeout-agent.md` (file size 463 bytes — same shape as till-gen analogue). Every spot-checked file has the literal "PLACEHOLDER" string. `agents.example.toml` ships with the SKETCH §4.2 `[agents]` defaults block (sonnet, env_from_shell, tools_allow), per-kind blocks for plan/build/qa-pair/research/commit (verified via `Read` of file lines 23-90). Schema match against `Preset` (in `internal/config/agents.go`, not yet loaded by W0) is vacuously true at this drop — chicken/egg correctly avoided per PLAN.md RiskNotes (line 106). REFUTED.

### Bridge-shape verdict (PRIMARY ATTACK)

**Cross-droplet bridge L1 W1.D1 ↔ W0.5 ↔ W5.D3 is correctly shaped.**

- **W0.5 → W1.D1 direction**: W0.5 ships the strict-mode validator + `embeddedAgentLibraryShipped` probe BEFORE W1.D1 lights any agent .md into the embed.FS (load.go:2110-2123). W0.5's "LOUD WARNING TO W1.D1 BUILDER" docstring on `TestLoadValidatesAgentBindingNamesDefaultLookupPermissivePreW1D1` (per worklog line 363) explicitly anticipates the cross-droplet handoff. Pre-W1.D1 the validator fails-permissive (returns true for every name) so existing tests pass; post-W1.D1 the same code-path flips to strict via the package-init probe seeing real files. Zero W0.5 code change required at W1.D1 land — exactly the sketch §FF2 disclosure pattern.
- **W1.D1 → W5.D3 direction**: W5.D3's PLAN.md acceptance bullet 1 (line 228) "every `[agent_bindings.<kind>] agent_name = "go-<name>"` becomes `agent_name = "<name>"`" + bullet 3 (line 230) "every internal/templates/builtin/agents/<group>/*.md placeholder file shipped by W1, frontmatter is name + description ONLY" + the explicit `Blocked by: 4c.6.W5.D1, 4c.6.W5.D2, 4c.6.W1.D1` chain (line 233) ensure W5.D3 picks up the 6 extras at deletion time without re-touching W1.D1's surface. Pre-Drop-1 `paths`-locking, this is a documented coordination contract; post-Drop-1 the orchestrator-cascade enforces via `paths` overlap detection.
- **Ordering correctness**: W5.D3 cannot ship before W1.D1 (would have nothing to delete); W1.D1 cannot ship before W0.5 (per PLAN.md `Blocked by: 4c.6.W0.5` at line 101 — same `internal/templates` package compile/test unit, plus W0.5's known-wired-set type lands first). Reordering would either crash sibling tests or violate package-lock chains. The chosen order is the only one that holds.
- **Asymmetry**: PLAN.md W1.D1's `Paths:` line and acceptance bullet 1 do not enumerate the 6 extras (Finding 1.1). This is a planner-level audit-trail asymmetry, not a build-level counterexample. The WORKLOG cross-droplet-handoff section + the `embed.go` doc-comment carry the explicit rationale; PLAN.md is the silent party. Routed to orchestrator as a refinement candidate for ancestor re-QA per WORKFLOW.md Phase 5 step 7 if orchestrator chooses to amend PLAN.md retroactively.

### Scope-expansion verdict (PRIMARY ATTACK)

**Justified operationalization of W0.5's now-shipped strict-mode contract — NOT scope-creep.**

The 6-file expansion is the load-bearing minimum for `LoadDefaultTemplateForLanguage("go")` to keep loading at HEAD post-W0.5-strict-flip + post-W1.D1-FS-population. Mid-build evidence: builder's worklog line 388 — "mid-build it returned 347/406 — exactly the W0.5 strict-mode-on-embed-shipped behaviour the W0.5 builder anticipated via the LOUD WARNING; landed the 6 legacy placeholders to resolve it." Counterfactual: had W1.D1 shipped only 21 files, ~59 tests would have broken pre-commit, the cascade-package-blocking-rule would have shipped a planner-level violation (default-go.toml editing belongs to W5.D1's `paths`, not W1.D1's), and the only YAGNI-respecting alternative would have been a planner-round-2 PLAN.md amendment to enumerate the 6 extras BEFORE the build. Builder chose to ship the operational fix + log the asymmetry to orchestrator certificate (worklog line 374-381) rather than block on a planner-round-2 — defensible given the evidence trail is complete and W5.D3 already owns the cleanup. Verdict: justified. The PLAN.md `Paths` + acceptance-bullet imprecision is a planner-spec audit-trail nit (Finding 1.1), not a build-level counterexample.

### Per-family attack-result table

| Family | Attack | Result | Notes |
| --- | --- | --- | --- |
| B1 | Test-coverage edge cases | REFUTED | 23/23 sub-tests GREEN; non-existent-name + empty-file + wrong-group not exercised but adequate for shipped contract (Finding 1.4) |
| B2 | Contract-preservation (PLACEHOLDER + frontmatter + agents.example.toml schema) | REFUTED | Spot-checked 4 files; marker held; schema chicken/egg correctly deferred to W0 (Finding 1.5) |
| B3 | Hidden-coupling (cross-droplet bridge) | REFUTED | Bridge L1 W1.D1 ↔ W0.5 ↔ W5.D3 correctly shaped; asymmetry in PLAN.md `Paths` enumeration logged as planner-level refinement (Finding 1.1) |
| B4 | YAGNI / scope-creep | REFUTED | All 4 alternative orderings (a)/(b)/(c)/(d) considered; chosen (b) is the only one without sibling-droplet collision (Finding 1.2) |
| B5 | Spec-compliance (21 vs 28 enumeration) | REFUTED | Test enumerates exactly 21 standard + 1 agents.example.toml; 6 extras covered by transitive `mage test-pkg` integration; 458/458 GREEN (Finding 1.3) |
| B6 | Shipped-but-not-wired | N/A | Placeholders consumed by W3 (resolver) + W2 (init copy) + W0.5 (validator floor) — all legitimate cross-wave wiring per PLAN.md `Blocked by` graph |
| B7 | Prompt-injection | EXHAUSTED | DORMANT pre-team-feature per agent rules + `feedback_prompt_injection_team.md`; no team-feature surface yet |

### Gates

- `mage test-func ./internal/templates TestDefaultTemplateFSEmbedsPlaceholderAgentFiles` → 23/23 PASS (1.23s).
- `mage test-pkg ./internal/templates` → 458/458 PASS (0.01s).
- `git grep` for legacy agent_name values in `default-go.toml` → confirms 5 unique `go-*` legacy names + `orchestrator-managed` (matches the 6 EXTRAS); `commit-message-agent` covered by the 21 standard set.

### Summary

**Verdict: pass.** Counterexample count: 0. All 7 attack families either REFUTED (B1, B2, B3, B4, B5), N/A (B6), or EXHAUSTED (B7). The 6-file scope expansion is JUSTIFIED operationalization of W0.5's strict-mode contract — NOT scope-creep — because (a) ordering W5.D3 before W1.D1 violates package-lock chains, (b) shipping only 21 files breaks `LoadDefaultTemplateForLanguage("go")` and ~59 tests, (c) editing `default-go.toml` from W1.D1 violates W5.D1's `paths` ownership, and (d) the WORKLOG + `embed.go` doc-comment carry a complete audit trail of the cross-droplet bridge. The L1 W1.D1 ↔ W0.5 ↔ W5.D3 cross-droplet bridge is CORRECTLY SHAPED. Finding 1.1 (PLAN.md `Paths` line + acceptance bullet 1 do not enumerate the 6 extras) is a planner-level audit-trail asymmetry routed to the orchestrator as a refinement candidate; not a build-level counterexample. Build is GREEN at 458/458 tests; targeted test 23/23 GREEN.

### Hylla Feedback

N/A — task touched primarily non-Go files (placeholder .md files + agents.example.toml). The Go-touching surface (`embed.go` //go:embed directive + comment, `embed_test.go` test addition, `load.go` validator semantics confirmation) was inspected via `Read` against local files; W0.5 was just-shipped pre-blocker so Hylla snapshot likely predates it. Builder's worklog already flagged the same staleness (worklog line 408-409). No additional feedback to record.
