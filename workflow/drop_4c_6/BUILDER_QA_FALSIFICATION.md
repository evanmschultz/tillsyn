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

---

## Droplet 4c.6.W6.D5 — Round 1

**Reviewer:** go-qa-falsification-agent (subagent, doc-only mode).
**Date:** 2026-05-09.
**Droplet:** `4c.6.W6.D5 — README.md pointer additions to new docs`.
**Artifact under attack:** `README.md` lines 27-30 (one prose lead-in + three bullets pointing at `AGENTS_CONFIG.md`, `CASCADE_METHODOLOGY.md`, `GDD_METHODOLOGY.md`), inserted between the existing repo-doc cross-reference block (lines 22-25) and the "Local dogfood repo layout note" block (now line 32).

### Counterexamples

(none — empty list)

### Findings (non-CONFIRMED, recorded for audit)

- 1.1 [Family: B5 spec-compliance] [severity: low] **Acceptance bullet 1 says "three short bullets (or a 'Methodology Docs' section)" — builder shipped a prose lead-in + three bullets, NOT an `## H2 Methodology Docs` section.** PLAN.md line 338 frames the choice as either a flat 3-bullet pointer block OR a dedicated section heading. The actual diff at README.md:27 is `Methodology docs (top-level, read these to understand how Tillsyn is built and used):` followed by three bullets — a prose-paragraph header rather than a bullets-only block or an `## H2 Methodology Docs` section. Reading "or a 'Methodology Docs' section" inclusively, a one-line prose label that titles the bullet block satisfies the spirit of the bullet. The acceptance bullet does NOT mandate either a literal `## Methodology Docs` H2 or a header-less bullet list — the parenthetical is permissive. Repro: read `README.md:27` (prose lead-in) vs `README.md:28-30` (bullets). Fix hint: leave as-shipped — the prose lead-in clearly labels the block AND mirrors the surrounding pre-existing pointer style at lines 22-25 (which is also prose, not a heading), so the no-restructuring constraint actually argues for a prose lead-in over an `## H2`. *Verdict: REFUTED — prose lead-in + 3 bullets satisfies the spec-bullet's permissive "(or a 'Methodology Docs' section)" clause AND honors the no-restructuring acceptance bullet by mirroring the surrounding pointer-prose style.*

- 1.2 [Family: B4 yagni] [severity: low] **Lead-in characterizes Tillsyn product purpose ("how Tillsyn is built and used") but README:8-9 already supplies a richer product description.** The new line 27 reads `Methodology docs (top-level, read these to understand how Tillsyn is built and used):`. The "how Tillsyn is built and used" framing partially duplicates the product-purpose framing already in README:8-9 (`A core product purpose is maintaining one DB-backed source of truth for coordination and execution state...`). However, the new lead-in is a *pointer-block label*, not a product-purpose paragraph — its job is to tell the reader what the bullets are for, not to introduce Tillsyn. The duplication is minimal (16 words) and serves a distinct rhetorical purpose. *Verdict: REFUTED — wording overlap is incidental; the lead-in's role is to introduce the bullet list, not to re-describe Tillsyn. Not a counterexample.*

### Attack family table

| Family                  | Result    | Notes                                                                 |
| ----------------------- | --------- | --------------------------------------------------------------------- |
| B1 test-coverage        | N/A       | Doc-only droplet — no test surface.                                   |
| B2 contract-preservation| REFUTED   | README pre-existing structure preserved: lines 22-25 (CONTRIBUTING/AGENTS/Integration Framing/sync-rule prose pointers) unchanged; "Local dogfood repo layout note" block unchanged in content (now starts at line 32 vs the pre-edit line 27 — purely a 5-line shift from the inserted block, no content edit). The new pointer block sits in the natural neighborhood of the existing CONTRIBUTING.md / AGENTS.md / CLAUDE.md cross-references per the builder's "placement choice" rationale (worklog lines 437-441). No content elsewhere in the README touched. |
| B3 hidden-coupling      | REFUTED   | All three pointer targets verified to exist at repo root via `wc -l`: `AGENTS_CONFIG.md` (396 lines), `CASCADE_METHODOLOGY.md` (200 lines), `GDD_METHODOLOGY.md` (62 lines). No anchor fragments used (`#anchor`-style links absent), so no broken-anchor surface. The GDD bullet explicitly flags "(placeholder; populated post-dogfood)" so a reader who clicks through and finds a stub isn't surprised — actively MITIGATES the placeholder-shape coupling concern. The AGENTS_CONFIG bullet describes "schema and authoring guide for `agents.toml` / `agents.local.toml`" which matches the doc's actual H1 (`# `agents.toml` Configuration Reference` at AGENTS_CONFIG.md:1) and §1 framing ("adopter-facing reference for `agents.toml` and its per-machine companion `agents.local.toml`" at AGENTS_CONFIG.md:3). The CASCADE_METHODOLOGY bullet ("plan-down / build-up, droplet sizing, planner-calls-planner recursion, QA discipline") matches the doc's stated scope (CASCADE_METHODOLOGY.md:3 "how work decomposes top-down through a recursive plan/build/QA tree...") and the methodology spine memory `feedback_plan_down_build_up.md`. |
| B4 yagni                | REFUTED   | 4 lines added (1 prose lead-in + 3 bullets). Minimal pointer block — no over-engineering, no new heading level (mirrors surrounding pointer-prose style), no inline duplication of the target docs' content. Builder explicitly chose `Edit` over `Write` (worklog line 451) and Grep'd for prior occurrences before adding (worklog line 449). Soft framing-overlap with README:8-9 product purpose recorded as Finding 1.2; not scope creep. |
| B5 spec-compliance      | REFUTED   | All three L1 acceptance bullets verified line-by-line: **(AC1)** Three pointers added — README.md:28 → AGENTS_CONFIG.md ✓; README.md:29 → CASCADE_METHODOLOGY.md ✓; README.md:30 → GDD_METHODOLOGY.md ✓. No restructuring of existing README content (B2 verified above). **(AC2)** Each bullet mentions purpose in 1 line + path: AGENTS_CONFIG bullet "schema and authoring guide for `agents.toml` / `agents.local.toml`, the runtime's per-kind agent + model bindings" (1 line, file backticked); CASCADE_METHODOLOGY bullet "the cascade methodology spine (plan-down / build-up, droplet sizing, planner-calls-planner recursion, QA discipline) that drives Tillsyn's coordination model" (1 line, file backticked); GDD_METHODOLOGY bullet "Goal-Driven Development methodology (placeholder; populated post-dogfood)" (1 line, file backticked). Prose-lead-in vs `## H2` choice recorded as Finding 1.1; REFUTED with mitigation. **(Idempotency RiskNote)** Builder pre-edit Grep across README.md for `AGENTS_CONFIG\|CASCADE_METHODOLOGY\|GDD_METHODOLOGY\|Methodology Docs\|methodology docs` returned `NO_MATCHES` per worklog line 449; `Edit` (not `Write`) used per worklog line 451. Re-running the droplet would Grep-detect the existing block and either no-op or fail-loud. |
| B6 shipped-but-not-wired| REFUTED   | All three pointer targets are real, populated files at the cited paths (sizes confirmed above). The pointers wire to existing docs that were either freshly shipped earlier in this drop (W6.D1 / W6.D2 / W6.D3) or are a deliberate placeholder (GDD, flagged as such in the bullet text). No dangling references. |
| B7 prompt-injection     | EXHAUSTED | DORMANT pre-team-feature per agent definition.                        |

### Summary

**Verdict: pass.** Counterexample count: 0. All applicable attack families either REFUTED (B2, B3, B4, B5, B6), N/A (B1), or EXHAUSTED (B7). Findings 1.1 (prose lead-in vs `## H2 Methodology Docs` heading — permissive spec clause satisfied) and 1.2 (incidental framing overlap with README:8-9 product description — distinct rhetorical role) recorded for audit; both REFUTED with mitigation rationale. README pointer block satisfies all hard acceptance criteria: three pointers to the three target docs (each verified to exist at repo root with non-trivial content), 1-line purpose per bullet, no restructuring of existing README content, idempotency RiskNote honored via pre-edit Grep + `Edit` (not `Write`).

### Hylla Feedback

N/A — task touched non-Go files only (`README.md` + the three target methodology docs at repo root). Hylla today indexes Go files only; verification of file existence + cited content used `Read` + `wc -l` directly per the project's "Non-Go files use Read/Grep/Glob/Bash" rule.

---

## Droplet 4c.6.W5.D1 — Round 1

**Reviewer:** go-qa-falsification-agent (subagent, doc-only mode).
**Date:** 2026-05-09.
**Droplet:** `4c.6.W5.D1 — Rename default-go.toml → till-go.toml (file move + embed.go + caller audit)`.
**Artifact under attack:** the rename `internal/templates/builtin/default-go.toml` → `till-go.toml` plus 7 caller-audit-edited Go files (`embed.go`, `embed_test.go`, `service.go`, `service_test.go`, `auto_generate_steward_test.go`, `mcp_surface.go`, `extended_tools.go`) per PLAN.md W5.D1 line 161 declared paths.

### Counterexamples

(none — empty list)

### Findings (non-CONFIRMED, recorded for audit)

- 1.1 [Family: B3 hidden-coupling] [severity: low] **`internal/adapters/server/common/mcp_surface.go:922` retains the stale forward-looking literal `(today: ["default-generic", "default-go"])` in a doc-comment 16 lines below the L906 BakeSource comment the builder did update — and `mcp_surface.go` IS in W5.D1's declared paths.** Production `BuiltinTemplateNames()` at `internal/templates/embed.go:247` returns `["default-generic", "till-go"]` post-W5.D1. The doc-comment on `ListBuiltinTemplatesResult` at `mcp_surface.go:922` says `// names embedded in the binary (today: ["default-generic", "default-go"])` — the parenthetical "(today: …)" framing is forward-looking, not historical (no rebadge note, no "pre-W5.D1" qualifier). The acceptance bullet at PLAN.md:167 is filename-scoped to `default-go.toml` (with `.toml`), so the literal `"default-go"` short-name reference at :922 does NOT trigger the AC4 grep. Repro: `git grep '"default-go"' -- internal/adapters/server/common/mcp_surface.go` returns the single hit at L922. Fix hint: extend the L906 BakeSource comment block's "rebadged from `default-go.toml` in Drop 4c.6 W5.D1" pattern to L922 — `(today: ["default-generic", "till-go"])` — coherent with the dual-history note pattern the builder applied elsewhere (`till-go.toml` header, `auto_generate_steward_test.go:18`, `service.go:385`, `extended_tools.go:1867`). *Verdict: REFUTED at counterexample level — the AC4 acceptance bullet is filename-scoped (`default-go.toml`) so the short-name `"default-go"` reference at :922 does not breach the literal acceptance criteria; production code returns the correct value; tests pass; no functional impact. Routed to orchestrator as a low-severity audit-trail miss inside a declared-path file. Single missed line in a 7-file edit set is a clean batting average; the cleanest fix is a one-line follow-up at the same time the W5.D2 caller-audit lands its `default-generic` flips, since the comment at :922 will require simultaneous update from `default-generic` → `till-gen` then anyway.*

- 1.2 [Family: B4 yagni / scope-creep] [severity: low] **HF6 audit scope was filename-only (`default-go.toml`), not short-name-inclusive (`default-go`).** The Round-2 HF6 regenerated audit at PLAN.md:161 + :168 verified callers via `git grep "default-go.toml" cmd/ internal/` — i.e. with `.toml` suffix. Short-name `"default-go"` literal references (no `.toml`) were NOT in the HF6 audit's regex. Hits left unaudited: `internal/adapters/server/common/mcp_surface.go:922` (Finding 1.1), `internal/app/template_service.go:114`, plus the four `extended_tools_test.go` hits already routed as builder Unknown #1. The audit-scope narrowing is defensible because (a) the file rename is the load-bearing change and the `.toml` suffix uniquely identifies callers that REFERENCE THE FILE, (b) a wider regex would also match the `embedded-default-go` BakeSource wire string — which the builder explicitly + correctly preserves as a stable wire identifier separate from on-disk file naming (`mcp_surface.go:907-909`), so wider grep would force per-hit triage between "file ref → update" vs "wire string → retain". Repro: rerun `git grep -nE '"default-go"' -- 'cmd/' 'internal/' '*.go'` (short-name regex, file-only); compare against PLAN.md HF6 audit regex `git grep "default-go.toml"`. Fix hint: future rename droplets ship two regex variants in their HF-audit subroutine — narrow `"<name>.toml"` for filename callers + wide `"<name>"` for short-name doc-comment refs, with explicit per-hit triage column distinguishing "wire string preserved intentionally" from "doc-comment forward-looking → update" — analogous to the historical-vs-forward-looking dual-classification the builder already applies. *Verdict: REFUTED — HF6 narrow-scope filename audit is defensible policy choice (avoids false positives on wire-string preservation); short-name leakage at :922 + :114 is the cost of that choice and is correctly classified as low-severity audit-trail drift, not scope-creep. The audit policy correctly prioritized "file ref" over "name ref" because the rename's load-bearing surface is the file itself.*

- 1.3 [Family: B3 hidden-coupling] [severity: low] **`internal/app/template_service.go:114` doc-comment `// `["default-generic", "default-go"]` post-F.2.` is stale post-W5.D1 — but this file is OUTSIDE W5.D1's declared paths (HF6 explicitly removed it at PLAN.md:176).** Production at L117 calls `templates.BuiltinTemplateNames()`, which now returns `["default-generic", "till-go"]`, so the doc-comment at L114 contradicts the runtime value the function actually produces. Builder's strict declared-paths discipline correctly avoided editing this file. The line will require update either at W5.D2 (when the comment also needs `default-generic` → `till-gen` flipped, making it a single-line co-edit) or at a follow-up refinement drop. Repro: `git grep '"default-go"' -- internal/app/template_service.go` returns hit at L114. Fix hint: defer to W5.D2's natural touch-set; the dual-flip is cleaner than two separate edits. *Verdict: REFUTED — out-of-scope file per declared-paths discipline; defer to W5.D2 where the line co-edits naturally with the second flip. Noted as Routed Unknown #2 confirmation: the deferral target is correct.*

- 1.4 [Family: B1 test-coverage] [severity: low] **`internal/adapters/server/mcpapi/extended_tools_test.go:883,3815` stub fixture drift is the routed Unknown #1 — confirmed correctly deferred.** Stub at L883 returns `[]string{"default-generic", "default-go"}`; assertion at L3815 asserts `want := []string{"default-generic", "default-go"}`. Both are the test stub's own internal contract — `TestTillTemplate_ListBuiltin` exercises the MCP wire surface against `stubExpandedService`, NOT against production `BuiltinTemplateNames()`. The test passes today (verified: `mage test-pkg ./internal/adapters/server/mcpapi` → 226/226 GREEN). The stub's purpose, per its own doc-comment at :874-876, is to "[mirror] templates.BuiltinTemplateNames so tests assert against the same wire vocabulary the production resolver exposes" — but post-W5.D1 the stub no longer mirrors production, it mirrors the pre-rename value. This is a hidden test-coverage regression: a future bug where the wire resolver flipped to a wrong list would not be caught by `TestTillTemplate_ListBuiltin` because the stub fixture has frozen at the wrong value. **Why deferred is correct**: (a) `extended_tools_test.go` is OUTSIDE W5.D1's declared paths (declared `extended_tools.go`, not `_test.go`); (b) the W5.D1 KindPayload `shape_hint` reads "string literal updates only" — flipping a stub fixture is more than a string-literal update, it touches a test contract; (c) the W5.D2 droplet's caller-audit naturally re-touches this surface when the second rename lands (`extended_tools_test.go` will need flips for `default-generic` → `till-gen` simultaneously); (d) the failing test would surface immediately if the stub drift caused real production drift, because the assertion compares against the stub's own return — they are coupled within the test file. Repro: `mage test-pkg ./internal/adapters/server/mcpapi` GREEN today; review extended_tools_test.go:874-883 + :3771-3819 stub-vs-assertion contract. Fix hint: W5.D2 should add `extended_tools_test.go` to its declared paths and update L883 + L3815 (and the related doc-comments at :874, :3673, :3704, :3705, :3773) to `["default-generic", "till-gen"]` — capturing both rebadges in one test-file edit. *Verdict: REFUTED — stub fixture drift is real but correctly deferred per declared-paths discipline + KindPayload `shape_hint` "string literal updates only" + the natural co-edit window in W5.D2. The deferral target W5.D2 (NOT W5.D3 as the worklog speculated at line 663) is the cleanest landing zone because W5.D2 already touches `extended_tools_test.go`-adjacent surfaces and the dual-rebadge co-edits in one pass. Recommend the orchestrator add an explicit `extended_tools_test.go` audit bullet to W5.D2's PLAN.md when it spawns. Confirmed correctly deferred.*

- 1.5 [Family: B2 contract-preservation] [severity: high — preserved correctly] **`embedded-default-go` BakeSource wire string explicit retention is correct.** Two production sites pin the wire string: `internal/app/template_service.go:44` (`templateBakeSourceEmbeddedGo = "embedded-default-go"`) and `internal/adapters/server/mcpapi/extended_tools.go:1921` (MCP description enumerates the closed BakeSource vocabulary `<bare-root>|<primary-worktree>|embedded-default-go|embedded-default-generic`). Builder's L906-909 doc-comment in mcp_surface.go explicitly notes: "the BakeSource string value `embedded-default-go` is intentionally retained as a stable wire identifier separate from the on-disk file name." Renaming the wire string would break MCP wire compatibility for any external `till.template get` consumer. Repro: `git grep -E 'embedded-default-go' -- 'cmd/' 'internal/' '*.go'` returns 11 hits, all coherent (production constant + MCP description + tests + doc-comments). *Verdict: REFUTED — contract preservation is exemplary. The on-disk-name vs wire-string separation is correctly enforced + documented. Not a counterexample.*

### Verdict on routed Unknowns

- **Routed Unknown #1** (`extended_tools_test.go:883,3815` stub fixture drift): **CORRECTLY DEFERRED to W5.D2** (Finding 1.4). Per-droplet declared-paths discipline + KindPayload `shape_hint` "string literal updates only" justify deferral. Recommended landing target is W5.D2 (NOT W5.D3 as the builder's worklog tentatively proposed) because W5.D2's natural caller-audit re-touch of test files makes the dual-rebadge a one-pass co-edit. Tests are green today; the drift surfaces immediately if production changes.
- **Routed Unknown #2** (forward-looking refs in out-of-scope files: `internal/templates/load.go:255,592,735,388,1240,1383,2096`, `internal/templates/load_test.go:1709,1927,2004,2222`, `internal/templates/builtin/agents/till-{gen,go}/*.md`, `internal/templates/builtin/default-generic.toml:3,7,35,40,253,261,273,312`, `.tillsyn/template.toml`): **CORRECTLY DEFERRED to W5.D2/W5.D3** per declared-paths discipline. None of these files are in W5.D1's declared paths; all are doc-comment / Markdown-frontmatter / TOML-comment refs (zero load-bearing strings). The `internal/templates/load.go` 4-hit set + `load_test.go` 4-hit set co-edit naturally with W5.D2's second rename. The agent placeholder .md files (5 hits across `till-go/go-*.md`) co-edit naturally with W5.D3's name-strip. The `default-generic.toml` 8-hit set co-edits with W5.D2's file rename itself. *Verdict: routing is correct.*

### Per-family attack-result table

| Family                   | Attack                                                                                          | Result    | Notes                                                                                                                                                |
| ------------------------ | ----------------------------------------------------------------------------------------------- | --------- | ---------------------------------------------------------------------------------------------------------------------------------------------------- |
| B1 test-coverage         | Stub fixture drift in `extended_tools_test.go:883,3815`                                         | REFUTED   | Confirmed Routed Unknown #1; correctly deferred to W5.D2. Tests 226/226 GREEN today (Finding 1.4)                                                    |
| B2 contract-preservation | `embedded-default-go` BakeSource wire string preservation; `BuiltinTemplateNames()` return shape| REFUTED   | Wire-string explicitly retained at production const + MCP description; production return value `["default-generic", "till-go"]` correct (Finding 1.5)|
| B3 hidden-coupling       | Short-name `"default-go"` literal in declared-path files                                        | REFUTED   | One miss at `mcp_surface.go:922` doc-comment (Finding 1.1) — literal acceptance bullet AC4 satisfied (filename-scoped); routed to orch as low-sev    |
| B4 YAGNI / scope-creep   | HF6 audit narrow-scope (filename-only, not short-name-inclusive)                                | REFUTED   | Narrow-scope is defensible policy (avoids wire-string false positives); short-name leakage is the cost of that choice (Finding 1.2)                  |
| B5 spec-compliance       | Each L1 acceptance bullet line-by-line                                                          | REFUTED   | All 7 ACs verified: rename ✓, `//go:embed` ✓, switch case ✓, `BuiltinTemplateNames()` ✓, `git grep "default-go.toml"` zero non-doc-comment hits ✓     |
| B6 shipped-but-not-wired | N/A                                                                                             | N/A       | Pure rename + caller audit; no new shipped surface to wire                                                                                            |
| B7 prompt-injection      | DORMANT pre-team-feature                                                                        | EXHAUSTED | Per agent rules + `feedback_prompt_injection_team.md`                                                                                                  |

### Gates

- `git grep -nE '"default-go"' -- 'cmd/' 'internal/' '*.go'` → 6 hits total: 1 in declared-path file (`mcp_surface.go:922` — Finding 1.1, low-sev doc-comment); 4 in out-of-scope file `extended_tools_test.go` (Routed Unknown #1 confirmed); 1 in out-of-scope `template_service.go:114` (Finding 1.3, defer to W5.D2).
- `git grep -nE 'default-go|default_go' -- 'cmd/' 'internal/' '*.go'` → 65 total hits; manual triage confirms every hit falls into one of: (a) declared-path doc-comment with rebadge note (correctly updated), (b) historical-rename-record retained per HF5 historical-rename rule, (c) `embedded-default-go` BakeSource wire string (intentionally retained), (d) out-of-scope file deferred to W5.D2/W5.D3 per declared-paths discipline.
- `mage test-pkg ./internal/adapters/server/mcpapi` → 226 tests / 226 PASS / 0 FAIL / 0 SKIP (0.00s). Confirms stub-fixture drift does not break tests today (the test asserts against the stub's own return, so stub-vs-assertion are co-pinned).
- File presence: `ls internal/templates/builtin/` shows `agents/`, `agents.example.toml`, `default-generic.toml`, `till-go.toml` — `default-go.toml` GONE; rename complete.
- `internal/templates/embed.go:72` `//go:embed builtin/till-go.toml builtin/default-generic.toml` directive verified.
- `internal/templates/embed.go:205` switch-case path literal `builtin/till-go.toml` verified.
- `internal/templates/embed.go:247` `BuiltinTemplateNames()` returns `[]string{"default-generic", "till-go"}` verified.

### Summary

**Verdict: pass.** Counterexample count: 0. All 7 attack families either REFUTED (B1, B2, B3, B4, B5), N/A (B6), or EXHAUSTED (B7). Five low-severity findings recorded for audit:

- Finding 1.1 (`mcp_surface.go:922` short-name `"default-go"` literal in declared-path file's doc-comment, missed by HF5 narrow-grep) — **routed to orchestrator** as a one-line follow-up (cleanest landing target: W5.D2's natural touch-set on the same comment for the `default-generic` → `till-gen` flip).
- Finding 1.2 (HF6 audit narrow-scope filename-only) — defensible policy; cost is the Finding 1.1 miss.
- Finding 1.3 (`template_service.go:114` doc-comment, OUT-of-scope per declared-paths) — defer to W5.D2.
- Finding 1.4 (`extended_tools_test.go:883,3815` stub fixture drift = Routed Unknown #1) — **CORRECTLY DEFERRED to W5.D2**; recommended W5.D2 add explicit declared-paths bullet for the test file.
- Finding 1.5 (BakeSource wire-string preservation) — exemplary contract preservation; not a counterexample.

**Verdict on routed Unknowns:** Routed Unknown #1 = correctly deferred to W5.D2 (NOT W5.D3 as worklog tentatively proposed); Routed Unknown #2 = correctly deferred to W5.D2/W5.D3 per declared-paths discipline. **Both deferrals are sound — neither was a W5.D1 violation.** The W5.D1 droplet hewed to declared paths discipline correctly; the only audit-trail miss inside declared paths (mcp_surface.go:922, Finding 1.1) is a single-line low-severity doc-comment that does not breach the filename-scoped AC4 acceptance bullet and has no functional impact (tests green; production return value correct). All hard acceptance criteria from PLAN.md:163-177 are satisfied.

### Hylla Feedback

None — Hylla answered everything needed. The droplet's surface is rename-driven (string-literal flips + file-rename), which is intrinsically a `git grep` job per the AC4 acceptance bullet's explicit `git grep "default-go.toml"` verification phrase. Hylla's strength (committed-code semantic search) is not the right tool for "find every `default-go` short-name occurrence" — that's a syntactic regex job which `git grep` handles directly. `Read` against the named declared-path files (8 files per PLAN.md:161) handled file-by-file inspection; `mage test-pkg` exercised the runtime contract for the mcpapi package. No Hylla query was attempted because the verification path is fully syntactic (regex + read + test-runner). Mirrors the builder's own Hylla-Feedback rationale at BUILDER_WORKLOG.md:669-679.

---

## Droplet 4c.6.W5.D2 — Round 1

**Reviewer:** go-qa-falsification-agent (subagent, sonnet).
**Date:** 2026-05-09.
**Droplet:** `4c.6.W5.D2 — Rename default-generic.toml → till-gen.toml (file move + embed.go + caller audit + extended-paths W5.D1 routed Unknowns)`.
**Artifact under attack:** the rename `internal/templates/builtin/default-generic.toml` → `till-gen.toml` plus 7 caller-audit-edited Go files (`embed.go`, `embed_test.go`, `service.go`, `service_test.go`, `auto_generate_steward_test.go`, `mcp_surface.go`, `template_service.go`, `extended_tools_test.go` — last 3 are extended-paths absorbing W5.D1 routed Unknowns 1.1 / 1.3 / 1.4).

### Counterexamples

(none — empty list)

### Findings (non-CONFIRMED, recorded for audit)

- **W5-D2-FF1** [Family: B3 hidden-coupling] [severity: medium] **W5.D2 builder routed three deferred sites (`internal/templates/load.go:388`, `:1240`, `internal/app/auto_generate_steward.go:108`) to W5.D3, but W5.D3's `Paths:` field at PLAN.md:225 does NOT include either file.** Worklog at lines 879-890 says "Did NOT touch `internal/templates/load.go` (lines 388 + 1240 reference `default-generic.toml` + `default-go.toml` together as historical doc-comments — outside W5.D2 declared paths; cleanest fix lives in W5.D3 alongside schema cleanup)." But W5.D3's PLAN row (lines 221-244) declares `Paths:` as `internal/templates/builtin/till-go.toml` + `internal/templates/builtin/till-gen.toml` + `internal/templates/builtin/agents/<group>/*.md` — explicitly NOT `internal/templates/load.go` and NOT `internal/app/auto_generate_steward.go`. Repro: read PLAN.md:225 (`Paths:` field of W5.D3) and compare to the worklog's deferral target. The deferral is to a sibling that never agreed to absorb the work. *Verdict: REFUTED at counterexample level — the three deferred sites are doc-comments only (verified via `Read` of `load.go:383-397`, `:1235-1249`, `auto_generate_steward.go:103-114` — all are `//` comments naming `default-go.toml + default-generic.toml` together as the embedded default's reachability/seam framing), zero functional impact, all tests green (`mage ci` 3005/3005 GREEN, `mage test-pkg ./internal/templates` 458/458 GREEN). The acceptance bullet "zero hits in non-doc-comment locations" is satisfied — the deferred sites ARE doc-comments. Routed to orchestrator as a planner-level audit-trail finding: either W5.D3's `Paths:` field needs amending to include `internal/templates/load.go` + `internal/app/auto_generate_steward.go` BEFORE W5.D3 spawns, OR a follow-up refinement droplet picks up the three sites alongside the broader Drop 4c.7 schema cleanup the W5.D3 RiskNote already names. Severity MEDIUM (vs LOW) because the W5.D1 → W5.D2 routing pattern was clean (W5.D1's routed Unknowns named explicit line numbers + W5.D2 absorbed all 3 with extended paths); the W5.D2 → W5.D3 routing breaks that pattern by deferring to a droplet that doesn't list the target paths.*

- **W5-D2-FF2** [Family: B3 hidden-coupling] [severity: low] **Test function name `TestLoadDefaultGenericTemplate` retained verbatim (5 hits across `embed_test.go:99`, `embed_test.go:65`, `embed_test.go:888`, `till-gen.toml:275`, `till-gen.toml:346`).** Mirrors W5.D1's `mustReadDefaultGoTOML` retention rationale (worklog W5.D1 lines 685-689: "Helper retained with a doc-comment update naming the rebadge; cleaner unification can land in W5.D2/W5.D3 or a later refinement drop"). The test body opens `builtin/till-gen.toml` correctly (verified at `embed_test.go:102`); the function NAME contains the legacy short-name. Renaming the function would touch every test that references it (5 sites) and is outside W5.D2's "string literal updates only" KindPayload `shape_hint`. *Verdict: REFUTED — name retention is consistent with W5.D1's documented pattern; test body assertions correct; zero functional impact. Refinement candidate for a post-MVP cleanup drop alongside `mustReadDefaultGoTOML` rename.*

- **W5-D2-FF3** [Family: B5 spec-compliance] [severity: low] **W5.D2 PLAN row (line 191-219) does NOT enumerate the 3 extended-paths sites (`extended_tools_test.go`, `template_service.go`, `mcp_surface.go:922`) in the `Paths:` declaration at line 195.** PLAN.md:195-201 lists 4 caller-audit sites + the renamed file + `embed.go` + `embed_test.go`. The 3 extended-paths sites that absorb W5.D1's routed Unknowns 1.1 / 1.3 / 1.4 are NOT in the PLAN's `Paths:` field — they appear only in the spawn prompt (per the W5.D2 worklog Round 1 design-decisions section line 879). Repro: read PLAN.md:195 vs worklog line 879. Builder correctly absorbed them per spawn-prompt directive; the asymmetry is at the planner level. *Verdict: REFUTED — same shape as W5.D1's Finding 1.1 (PLAN `Paths:` line vs worklog `Files Touched:` line asymmetry, reported low-severity); spawn-prompt extension was the orchestrator's chosen channel for routing the Unknowns. Routed to orchestrator as a planner-spec accuracy refinement candidate; the PLAN-vs-spawn-prompt audit-trail asymmetry is a recurring pattern this drop, not a build-level counterexample.*

- **W5-D2-FF4** [Family: B2 contract-preservation] [severity: high — preserved correctly] **`embedded-default-generic` BakeSource wire-string retention is correct (mirrors W5.D1's `embedded-default-go` pattern).** Six production sites pin the wire string: `internal/app/template_service.go:36` (comment naming `""` → `"embedded-default-generic"` map), `:43` (production constant `templateBakeSourceEmbeddedGeneric`), `internal/adapters/server/common/mcp_surface.go:910-915` (explicit doc-comment justifying wire-string retention), `internal/adapters/server/mcpapi/extended_tools.go:1921` (MCP description enumerates closed BakeSource vocabulary `<bare-root>|<primary-worktree>|embedded-default-go|embedded-default-generic`), `internal/adapters/server/mcpapi/extended_tools_test.go:837` (stub doc-comment), `:866` (stub return value). Builder's `mcp_surface.go:910-915` doc-comment explicitly notes: "the BakeSource string value `embedded-default-generic` is intentionally retained as a stable wire identifier separate from the on-disk file name, mirroring the W5.D1 wire-string-vs-filename split." Renaming would break MCP wire compatibility for any external `till.template get` consumer. Repro: `git grep -nE 'embedded-default-generic' -- 'cmd/' 'internal/' '*.go'` returns 6 hits, all coherent. *Verdict: REFUTED — contract preservation is exemplary. The on-disk-name vs wire-string separation is correctly enforced + documented. Not a counterexample.*

- **W5-D2-FF5** [Family: B1 test-coverage] [severity: low] **`extended_tools_test.go:885` stub return + L3818 want literal both flipped to `["till-gen", "till-go"]`; W5.D1's Routed Unknown #1 is closed.** Stub at L885 returns `[]string{"till-gen", "till-go"}` (matches production `BuiltinTemplateNames()` at `embed.go:255`); want literal at L3818 asserts the same. Pre-W5.D2 the stub was frozen at `["default-generic", "default-go"]` per W5.D1 Finding 1.4 — the drift would have hidden a future production-resolver wire bug. Post-W5.D2 the stub mirrors production correctly. `mage test-pkg ./internal/adapters/server/mcpapi` → 226/226 GREEN (verified via the `mage ci` run). *Verdict: REFUTED — the routed Unknown is correctly closed; stub-vs-production drift eliminated; tests green.*

- **W5-D2-FF6** [Family: B5 spec-compliance] [severity: low] **`BuiltinTemplateNames()` literal sort order verified.** Production at `embed.go:255` returns `[]string{"till-gen", "till-go"}`. Lexical sort: `t-i-l-l-`-`-g-e-n` (`till-gen`) vs `t-i-l-l-`-`-g-o` (`till-go`) — the comparison turns on position 8 where `e` (101) < `o` (111), so `till-gen` < `till-go`. Order is preserved. Returned by VALUE (literal slice constructed each call) — no shared backing array, mutation isolated to caller's copy. Doc-comment at `embed.go:245-253` explicitly affirms "fresh slice on every call so callers cannot mutate the package-level source of truth." *Verdict: REFUTED — sort order correct, immutability invariant correct.*

### Closed-list verification table

| Site | Pre-W5.D2 | Post-W5.D2 | Status |
| --- | --- | --- | --- |
| `embed.go:75` `//go:embed` directive | `builtin/till-go.toml builtin/default-generic.toml` | `builtin/till-go.toml builtin/till-gen.toml` | ✓ updated |
| `embed.go:209` switch case `""` | `builtin/default-generic.toml` | `builtin/till-gen.toml` | ✓ updated |
| `embed.go:255` `BuiltinTemplateNames()` literal | `["default-generic", "till-go"]` | `["till-gen", "till-go"]` | ✓ updated |
| `embed_test.go:102` test body open path | `builtin/default-generic.toml` | `builtin/till-gen.toml` | ✓ updated |
| `template_service.go:115` doc-comment `today: …` | `["default-generic", "default-go"]` | (rebadge note) | ✓ doc-comment updated; W5.D1 Routed Unknown 1.3 closed |
| `mcp_surface.go:926` doc-comment `today: …` | `["default-generic", "default-go"]` | `["till-gen", "till-go"]` | ✓ updated; W5.D1 Routed Unknown 1.1 closed |
| `extended_tools_test.go:885` stub return | `["default-generic", "default-go"]` | `["till-gen", "till-go"]` | ✓ updated; W5.D1 Routed Unknown 1.4 closed |
| `extended_tools_test.go:3818` want literal | `["default-generic", "default-go"]` | `["till-gen", "till-go"]` | ✓ updated; W5.D1 Routed Unknown 1.4 closed |
| `embedded-default-generic` BakeSource wire string (6 sites) | retained | retained | ✓ explicit retention with doc-comment justification (W5-D2-FF4) |
| `internal/templates/load.go:388` + `:1240` doc-comments | retained | retained | ⚠ deferred to W5.D3, but W5.D3's `Paths:` does NOT cover (W5-D2-FF1) |
| `internal/app/auto_generate_steward.go:108` doc-comment | retained | retained | ⚠ deferred to W5.D3, but W5.D3's `Paths:` does NOT cover (W5-D2-FF1) |
| Test function name `TestLoadDefaultGenericTemplate` (5 sites) | retained | retained | ✓ stylistic retention; matches W5.D1 `mustReadDefaultGoTOML` pattern (W5-D2-FF2) |

### Per-family attack-result table

| Family                   | Attack                                                                                          | Result    | Notes                                                                                                                                                |
| ------------------------ | ----------------------------------------------------------------------------------------------- | --------- | ---------------------------------------------------------------------------------------------------------------------------------------------------- |
| B1 test-coverage         | Stub fixture flip (`extended_tools_test.go:885` + `:3818`); `BuiltinTemplateNames` literal vs sort order | REFUTED   | Both stub fixture and want literal flipped to `["till-gen", "till-go"]`; production return matches; sort order correct; tests 226/226 + 458/458 GREEN; Routed Unknown #1 closed (W5-D2-FF5/FF6) |
| B2 contract-preservation | `embedded-default-generic` BakeSource wire string preservation; `embedded-default-go` parallel  | REFUTED   | Wire-string explicitly retained at production const + MCP description + 3 stub sites; doc-comment at `mcp_surface.go:910-915` justifies retention with explicit "intentionally retained as a stable wire identifier separate from the on-disk file name" framing (W5-D2-FF4) |
| B3 hidden-coupling       | Three deferred sites (`load.go:388`, `:1240`, `auto_generate_steward.go:108`) routed to W5.D3 whose `Paths:` does NOT include them | REFUTED at counterexample level | Sites are doc-comments only, zero functional impact, all tests green; medium-severity audit-trail finding routed to orch (W5-D2-FF1)              |
| B4 YAGNI / scope-creep   | Extended-paths absorption (3 sites beyond PLAN `Paths:`)                                        | REFUTED   | Spawn-prompt directive routed W5.D1 Unknowns 1.1/1.3/1.4 to W5.D2 explicitly; absorption is justified operationalization of the routing chain, not scope creep; PLAN-vs-spawn-prompt asymmetry is a recurring pattern this drop, not a counterexample (W5-D2-FF3) |
| B5 spec-compliance       | Each L1 acceptance bullet line-by-line                                                          | REFUTED   | (AC1) `git mv` rename verified ✓; (AC2) `//go:embed` directive references `builtin/till-gen.toml` ✓; (AC3) switch case + `BuiltinTemplateNames()` returns `["till-gen", "till-go"]` ✓; (AC4) `git grep "default-generic.toml"` zero non-doc-comment hits in cmd/ + internal/ ✓; (AC5) `mage ci` GREEN (3005/3005) ✓ |
| B6 shipped-but-not-wired | N/A                                                                                             | N/A       | Pure rename + caller audit; no new shipped surface to wire                                                                                            |
| B7 prompt-injection      | DORMANT pre-team-feature                                                                        | EXHAUSTED | Per agent rules + `feedback_prompt_injection_team.md`                                                                                                  |

### Gates

- `mage ci` → 3005/3005 PASS across 25 packages, all packages ≥ 70% coverage (`internal/templates` at 94.5%, `internal/app` at 71.6%); build of `./cmd/till` SUCCESS. Verified locally as falsification gate (W5.D2 worklog explicitly defers `mage ci` to QA per `~/.claude/agents/go-builder-agent.md` rule).
- `mage test-pkg ./internal/templates` → 458/458 PASS (matches worklog claim).
- `git status` confirms RM-rename `internal/templates/builtin/default-generic.toml -> internal/templates/builtin/till-gen.toml` (Git history-preserving rename detected).
- `git grep -nE '"default-generic"' -- 'cmd/' 'internal/' '*.go'` → 3 hits, ALL doc-comments (rebadge-history records at `extended_tools_test.go:875`, `:3776`, `template_service.go:115`).
- `git grep -nE 'default-generic\.toml' -- 'cmd/' 'internal/' '*.go'` → 18 hits, ALL doc-comments / TOML header-comments / rebadge-history. No production load-bearing string.
- `git grep -nE 'embedded-default-generic' -- 'cmd/' 'internal/' '*.go'` → 6 hits, all coherent (BakeSource constant + MCP description + stub + doc-comment justification).
- File presence: `ls internal/templates/builtin/` shows `agents/`, `agents.example.toml`, `till-gen.toml`, `till-go.toml` — `default-generic.toml` GONE; rename complete.
- `internal/templates/embed.go:75` `//go:embed builtin/till-go.toml builtin/till-gen.toml` directive verified.
- `internal/templates/embed.go:209` switch-case path literal `builtin/till-gen.toml` verified.
- `internal/templates/embed.go:255` `BuiltinTemplateNames()` returns `[]string{"till-gen", "till-go"}` verified (lexical order: `till-gen` < `till-go`).

### Severity breakdown

- **HIGH (preserved-correctly):** 1 — W5-D2-FF4 (BakeSource wire-string retention, exemplary).
- **MEDIUM:** 1 — W5-D2-FF1 (sibling-deferral routing breaks W5.D1's clean pattern; W5.D3 `Paths:` does not cover the 3 deferred sites).
- **LOW:** 4 — W5-D2-FF2 (test function name retention, consistent with W5.D1 pattern), W5-D2-FF3 (PLAN-vs-spawn-prompt `Paths:` asymmetry, recurring), W5-D2-FF5 (Routed Unknown #1 closed), W5-D2-FF6 (sort order + immutability).

### Summary

**Verdict: pass.** Counterexample count: 0. All 7 attack families either REFUTED (B1, B2, B3, B4, B5), N/A (B6), or EXHAUSTED (B7). The W5.D2 droplet correctly absorbed all three W5.D1 routed Unknowns (1.1 → `mcp_surface.go:926`, 1.3 → `template_service.go:115`, 1.4 → `extended_tools_test.go:885 + :3818`); zero stub-fixture-vs-production drift remains. The rename is structurally complete (`git status` confirms RM-rename, `embed.go` directive + switch + names literal all updated, file gone from disk). Production return value `["till-gen", "till-go"]` verified at the source literal in `embed.go:255` with lexical-order audit. Wire-string retention pattern (W5-D2-FF4) is exemplary contract preservation. `mage ci` 3005/3005 GREEN, `mage test-pkg ./internal/templates` 458/458 GREEN.

**One MEDIUM-severity finding (W5-D2-FF1) routed to orchestrator:** the three deferred sites (`load.go:388`, `:1240`, `auto_generate_steward.go:108`) live in files NOT named in W5.D3's `Paths:` declaration at PLAN.md:225 — the W5.D2-to-W5.D3 hand-off breaks the clean routing pattern W5.D1 established. Recommended remediation: either amend W5.D3's `Paths:` to include `internal/templates/load.go` + `internal/app/auto_generate_steward.go` BEFORE W5.D3 spawns, OR file a follow-up refinement droplet alongside the Drop 4c.7 schema cleanup the W5.D3 RiskNote already names. None of the deferred sites have functional impact (all doc-comments, all tests green); the finding is audit-trail integrity, not behavior.

### Hylla Feedback

None — Hylla answered everything needed. The droplet's surface is rename-driven (string-literal flips + file-rename + extended-paths absorption), which is intrinsically a `git grep` job per the AC4 acceptance bullet's explicit `git grep "default-generic.toml"` verification phrase. Hylla's strength (committed-code semantic search) is not the right tool for "find every `default-generic` short-name occurrence" — that's a syntactic regex job which `git grep` handles directly. `Read` against the named declared-path files (8 files per PLAN.md:195 + 3 extended-paths sites) handled file-by-file inspection; `mage ci` exercised the full runtime contract across 25 packages. No Hylla query was attempted because the verification path is fully syntactic (regex + read + test-runner). Mirrors the builder's own Hylla-Feedback rationale at BUILDER_WORKLOG.md:962-970 and the W5.D1 falsification's same rationale at this file's line 467.

---

## Droplet 4c.6.W2.D3a — Round 1

**Reviewer:** go-qa-falsification-agent (subagent, sonnet).
**Date:** 2026-05-09.
**Droplet:** `4c.6.W2.D3a — cmd/till/init_cmd.go skeleton + register in main.go + help-entry`.
**Artifact under attack:** `cmd/till/init_cmd.go` (NEW, 58 lines), `cmd/till/init_cmd_test.go` (NEW, 44 lines), `cmd/till/main.go` (modified: build `initCmd` + add to `rootCmd.AddCommand` line 1906), `cmd/till/help.go` (modified: added `"till init"` entry to `commandHelpSpecs` map at lines 377-392).

### Counterexamples

(none — empty list)

### Findings (non-CONFIRMED, recorded for audit)

- 1.1 [Family: cobra-wiring] [severity: low] **REFUTED — `cobra.NoArgs` matches local convention.** Builder uses `Args: cobra.NoArgs` at `init_cmd.go:36`. The sibling `init-dev-config` command at `main.go:1899` uses `Args: cobra.NoArgs` (verified by reading lines 1884-1903). Other call sites in `main.go` use `cobra.MaximumNArgs(1)` (project create at 657, project show at 694) for commands that DO accept positional args. The local convention for argless subcommands is `cobra.NoArgs`, which D3a follows. No drift.

- 1.2 [Family: cobra-wiring] [severity: low] **REFUTED — no Aliases collision.** Hylla keyword search `query="Aliases initDevConfigCmd cobra command"` returned only the magefile.go `Aliases` variable (mage tool's hyphenated-target alias map at `magefile.go`); no cobra `Aliases:` field is declared anywhere in `cmd/till/`. The new `Use: "init"` cannot collide with a hidden alias because no cobra command in the tree declares one. Refuted.

- 1.3 [Family: cobra-wiring] [severity: low] **REFUTED — flag retrieval is idiomatic.** `init_cmd.go:38` reads `payload, err := cmd.Flags().GetString("json")` — the idiomatic typed-getter form (NOT the non-idiomatic `cmd.Flag("json").Value.String()`). The error from `GetString` is propagated. Cobra-API discipline preserved.

- 1.4 [Family: stub-error-text] [severity: low] **REFUTED — both stub error strings match the contract verbatim.** Acceptance bullet at PLAN.md line 90 prescribes `"till init: JSON parse not yet wired (W2.D3b)"` and `"till init: TUI walk not yet wired (W2.D4)"`. `init_cmd.go:43` produces `errors.New("till init: JSON parse not yet wired (W2.D3b)")`; `init_cmd.go:57` produces `errors.New("till init: TUI walk not yet wired (W2.D4)")`. Both use `errors.New` (NOT wrapped via `fmt.Errorf` with `%w`) — appropriate for sentinel-style stub errors that downstream tests substring-match on, since wrapping would risk format-string drift. The downstream D3b/D4 contract is preserved byte-for-byte. The smoke tests at `init_cmd_test.go:22-25` and `:39-42` use `strings.Contains(err.Error(), want)` substring matching, so even if a future drop wraps these errors, the consumer-tie tests stay green.

- 1.5 [Family: consumer-tie] [severity: low] **REFUTED — W2-FF6 invocation form is correct.** The smoke tests at `init_cmd_test.go:18` and `:35` invoke `run(context.Background(), []string{"--app", "tillsyn-init", "init", ...}, &out, io.Discard)`. The `--app` flag is a real persistent root flag at `main.go:511` (`rootCmd.PersistentFlags().StringVar(&rootOpts.appName, "app", rootOpts.appName, "Application name for config/data path resolution")`); appName=`tillsyn-init` only affects path-resolution, not the subcommand name. The `init` token after the persistent flags is the subcommand. The pre-existing `TestRunInitDevConfigCreatesDebugConfig` at `main_test.go:2928` uses the symmetric form `[]string{"--app", "tillsyn-init", "init-dev-config"}` — D3a's invocation pattern is the same shape, validated against the same path resolver. Verified the run-tree exercises cobra registration: `mage test-func ./cmd/till TestInit_BareInvocation_ReturnsTUIStubError` GREEN (1.89s).

- 1.6 [Family: help-entry-key] [severity: low] **REFUTED — `cmd.CommandPath()` resolves to `"till init"` exactly.** `applyCommandHelpSpecs` at `help.go:419-432` walks the cobra command tree and keys by `cmd.CommandPath()` (line 421). For a child of `rootCmd` (`Use: "till"` at `main.go:480`) with `Use: "init"` (at `init_cmd.go:18`), `CommandPath()` returns the parent's `Use` joined with the child's `Use` separated by a single space → `"till init"`. The `commandHelpSpecs` map key at `help.go:377` is `"till init"` (exact match). No whitespace / case-sensitivity / separator mismatch. Defensive note: if the key DID mismatch, `applyCommandHelpSpecs` silently `return`s without error (line 422-424) — the inline `Long` set in `init_cmd.go:20-31` would still apply, so the user-facing failure mode is "help text reverts to the inline default" rather than "help blows up." Belt-and-suspenders, not a bug.

- 1.7 [Family: help-entry-key] [severity: low] **REFUTED — alphabetical placement is cosmetic-only.** Builder at BUILDER_WORKLOG.md:995-1001 acknowledges Go map iteration is randomized, and the placement of `"till init"` immediately above `"till init-dev-config"` in `help.go` source order is for human readability — `applyCommandHelpSpecs` keys by `cmd.CommandPath()` at runtime, not by source position. The comment is accurate; the behavior is correct.

- 1.8 [Family: rich-help-content] [severity: low] **REFUTED — help-entry content is sane.** `help.go:377-392` `"till init"` block has: `Long` (lines 378-387) describing project-init responsibilities (agents-dir copy, agents.toml, .gitignore, optional .mcp.json, project DB record) and re-run-safety invariant; `Example` (lines 388-391) covering bare-TUI invocation and `--json` headless invocation. No leftover placeholder strings, no broken markdown, no copy-paste artifacts from `init-dev-config`. The content matches the SKETCH §9 init-vs-install separation (the long-form names cwd-local seeding behavior, not home-local dev-bootstrap behavior — those words are reserved for D7.5's `till install` entry).

- 1.9 [Family: register-call] [severity: low] **REFUTED — `initCmd` registered at the right level.** `main.go:1905` builds `initCmd := newInitCommand(stdout, rootOpts)` immediately after the `initDevConfigCmd` literal block; `main.go:1906` adds `initCmd` as the trailing arg of the `rootCmd.AddCommand(serveCmd, mcpCmd, ..., initDevConfigCmd, initCmd)` call. `initCmd` is registered as a sibling of `initDevConfigCmd` (both children of `rootCmd`), NOT nested under `initDevConfigCmd`. The order of args to `AddCommand` does not impact help output (cobra sorts subcommands alphabetically in help text). No regression.

- 1.10 [Family: adjacent-regressions] [severity: low] **REFUTED — `TestRunRootHelp` does NOT regress.** `main_test.go:476` asserts a hard-coded list of substrings present in root help (`"serve", "mcp", "auth", "project", "embeddings", "capture-state", "kind", "lease", "handoff", "export", "import", "paths", "init-dev-config"`). The list does NOT include `"init"`, but the assertion uses `!strings.Contains(output, want)` — i.e. it requires EXISTING items to remain visible, not that the list be exhaustive. New commands appearing in root help do NOT trigger this assertion. `mage ci` confirms `TestRunRootHelp` GREEN. No regression. (Audit-trail observation: if a future drop wants to extend the assertion to include `"init"`, that's a small follow-on edit; not load-bearing for D3a.)

- 1.11 [Family: adjacent-regressions] [severity: low] **REFUTED — `TestRunSubcommandHelp` does NOT regress, but coverage gap noted.** `main_test.go:498-736` is a hard-coded `cases` table iterated by name (line 740). The table does NOT include a row for `"init"`. The test assertion logic only iterates the named cases — it does NOT dynamically discover all registered subcommands. Therefore: (a) the test still passes (no row to fail) — `mage ci` GREEN confirms; (b) **the new `till init` rich-help block is NOT exercised by any case in the hardcoded table.** A future regression where the `"till init"` map key drifts (e.g. typo, case mismatch, whitespace) would not be caught by `TestRunSubcommandHelp`. Routing this as an **Unknown** rather than a counterexample because: (i) the inline `Long` in `init_cmd.go:20-31` provides a fallback (per Finding 1.6 defensive note), so the user-visible failure mode is graceful degradation not breakage; (ii) the planner's D3a acceptance does NOT name a `TestRunSubcommandHelp` row for `init` — extending the table is out of scope for D3a; (iii) D3b/D4/D5/D6/D7 will continue building out `init`'s body and a natural test extension can land alongside one of those droplets. Recommended follow-up: **add a `"init"` row to `TestRunSubcommandHelp`'s `cases` table** in a future droplet (likely D7 when the success message + rich help fully stabilize). Audit-trail finding, not a counterexample.

- 1.12 [Family: yagni] [severity: low] **REFUTED — stub error text uses `errors.New`, not wrapped.** Both stubs use `errors.New(...)` (lines 43, 57). No premature wrapping with `fmt.Errorf("%w", ...)` for stub-stage messages. The downstream consumer-tie tests substring-match the literal text — `errors.New` is the right call here. When D3b/D4 replace these with real error paths, wrapping is welcome (they'll be wrapping real underlying errors); for stubs, plain is correct.

- 1.13 [Family: file-gating] [severity: low] **REFUTED — edits stay within declared `paths`.** PLAN.md:82-86 declares D3a's paths: `cmd/till/init_cmd.go` (NEW), `cmd/till/init_cmd_test.go` (NEW), `cmd/till/main.go` (modify), `cmd/till/help.go` (modify). Builder edited exactly these four files (BUILDER_WORKLOG.md:923-957). No `main_test.go` edits (those are D8's responsibility per the planner's gating); no out-of-package edits. Clean gating discipline.

- 1.14 [Family: yagni] [severity: low] **REFUTED — skeleton stays minimal.** `init_cmd.go` is 58 lines: package + imports + `newInitCommand` (35 lines including help text) + `runInitTUI` stub (8 lines). No premature abstractions, no helper functions that D3b–D7 don't need. The `runInitTUI` signature `(stdout io.Writer, opts rootCommandOptions) error` is the minimum surface D4 needs to fill. The `_ = stdout; _ = opts` blank-identifier pattern at lines 55-56 prevents unused-parameter lints without introducing dead code. Skeleton-grade only.

- 1.15 [Family: hidden-deps] [severity: low] **REFUTED — no `init()` side effects, no package-level state added.** `init_cmd.go` declares only the `newInitCommand` and `runInitTUI` functions. No `init()` block, no package-level vars or const, no import-side-effect imports. Test file similarly clean. No global state introduced by D3a.

- 1.16 [Family: error-handling] [severity: low] **REFUTED — flag retrieval error is propagated.** `init_cmd.go:39-41` returns the error from `cmd.Flags().GetString("json")` directly, no swallowing. `errors.New(...)` at lines 43 and 57 are sentinel-creating, not swallowing. No `_ = err` patterns, no logged-but-not-returned errors.

- 1.17 [Family: concurrency] [severity: n/a] No concurrency added. Cobra `RunE` is a serial dispatch; the `runInitTUI` stub is synchronous. No goroutines, no channels, no shared state, no context-cancellation paths to exercise yet (D4's bubbletea walk will introduce a `tea.Program`, but D3a is pre-walk).

- 1.18 [Family: raw-go-or-mage-install] [severity: low] **REFUTED — no raw `go` / `mage install` violations.** `BUILDER_WORKLOG.md:1027-1044` records mage targets only: `mage test-func`, `mage format`. No `go test`, `go build`, `go vet`, no `mage install`. Builder explicitly defers `mage ci` to the QA pair per agent-file rule. Disciplined.

### Per-family attack-result table

| Family                      | Verdict        | Notes                                                                 |
| --------------------------- | -------------- | --------------------------------------------------------------------- |
| cobra-wiring (1.1–1.3)      | REFUTED        | NoArgs matches convention, no Aliases, GetString idiomatic.           |
| stub-error-text (1.4)       | REFUTED        | Both stubs verbatim per contract; `errors.New` (no wrapping).         |
| consumer-tie (1.5)          | REFUTED        | `--app tillsyn-init` form mirrors existing init-dev-config tests.     |
| help-entry-key (1.6, 1.7)   | REFUTED        | `cmd.CommandPath()` resolves exactly; alphabetical placement cosmetic. |
| rich-help-content (1.8)     | REFUTED        | Long + Example sane; SKETCH §9 init-vs-install separation honored.    |
| register-call (1.9)         | REFUTED        | Sibling of `initDevConfigCmd`, not nested.                            |
| adjacent-regressions (1.10) | REFUTED        | `TestRunRootHelp` not regressing; assertion non-exhaustive.           |
| adjacent-regressions (1.11) | REFUTED + UNK  | `TestRunSubcommandHelp` non-regressing but D3a's help-entry NOT exercised by table — coverage gap routed as Unknown for D7-era follow-up. |
| yagni (1.12, 1.14)          | REFUTED        | Skeleton minimal; no premature wrapping/abstractions.                 |
| file-gating (1.13)          | REFUTED        | Edits stay within declared paths.                                     |
| hidden-deps (1.15)          | REFUTED        | No init(), no package-level state.                                    |
| error-handling (1.16)       | REFUTED        | Flag-retrieval error propagated; no swallowing.                       |
| concurrency (1.17)          | N/A            | No concurrency added in skeleton stage.                               |
| raw-go-or-mage-install (1.18) | REFUTED      | Mage-only test invocations; no `mage install`.                        |

### Mage gate

- `mage ci` — **GREEN**, 3007 tests pass across 25 packages (this reviewer ran end-to-end). All package coverage thresholds met: `cmd/till` at 75.8%, every package ≥ 70.0%. Build of `./cmd/till` SUCCESS.
- `mage test-pkg ./cmd/till` — **GREEN**, 255 tests pass.
- `mage test-func ./cmd/till TestInit_BareInvocation_ReturnsTUIStubError` — **GREEN**, 1 test pass (1.89s).

### Severity breakdown

- **HIGH:** 0
- **MEDIUM:** 0
- **LOW:** 17 (audit-trail / verification-of-claim findings only; all REFUTED counterexamples or confirming-evidence)
- **N/A:** 1 (concurrency family, nothing to attack at skeleton stage)

### Summary

**Verdict: pass.** Counterexample count: 0. All 14 attack families either REFUTED (1.1–1.10, 1.12–1.16, 1.18) or N/A (1.17). One Unknown routed: Finding 1.11 — `TestRunSubcommandHelp`'s hardcoded `cases` table at `main_test.go:498-736` does NOT include a row for `"init"`, so D3a's new `"till init"` help-entry rich text is not exercised by any test. Recommended follow-up: extend the table with an `"init"` row in a future droplet (D7 candidate when the success-message rich text fully stabilizes). The inline `Long` in `init_cmd.go:20-31` provides a graceful-degradation fallback if the help-spec map key ever drifts, so the user-visible risk of the gap is "help reverts to inline default," not "help breaks." Audit-trail signal, not a counterexample.

D3a's surface (skeleton + register + help-entry, ~58 LOC of new production code + 44 LOC of new test) is correctly scoped to what the plan declares: cobra command exists, `--json` flag wired (parser body STUB, owned by D3b), `RunE` dispatch routes to TUI-stub or JSON-stub with verbatim error text, registration in `rootCmd.AddCommand` lands at `main.go:1906`, help-entry lands at `help.go:377-392` with proper `cmd.CommandPath()` keying. Both consumer-tie smoke tests (`TestInit_BareInvocation_ReturnsTUIStubError`, `TestInit_JSONInvocation_ReturnsJSONStubError`) invoke the run-tree end-to-end via `run(context.Background(), []string{"--app", "tillsyn-init", "init", ...}, ...)` per the W2-FF6 ROUND-2 contract. `mage ci` GREEN. No counterexamples found across 14 attack families.

### Hylla Feedback

- **Query**: `mcp__hylla__hylla_search_keyword query="func run context Args Stdout cmd/till"` and `query="appFlag tillsyn-init root command --app"` (both `node_type=block`, `fields=[content]`).
- **Missed because**: the relevant code (`run` at `cmd/till/main.go:394`, `--app` flag at `:511`, the new `init_cmd.go`) was either too high-noise to surface (the `run` symbol shares the `func run` shape with many domain-package helpers, all of which dominated keyword scoring) OR not yet ingested (the new `init_cmd.go` shipped in this drop and Hylla's snapshot 5 predates it). Both are expected staleness / noise patterns, not bugs.
- **Worked via**: `Read` against `cmd/till/main.go` lines 1-120 + 340-540 + 650-855 + 1860-1916 (multiple ranges); `Read` against `cmd/till/init_cmd.go` (full file, 58 lines); `Read` against `cmd/till/init_cmd_test.go` (full file, 44 lines); `Read` against `cmd/till/help.go` lines 1-100 + 370-447; `Read` against `cmd/till/main_test.go` lines 460-487 + 720-790 + 2890-2960; `Read` against `workflow/drop_4c_6/BUILDER_WORKLOG.md` lines 750-1050. `mage test-func` and `mage test-pkg` and `mage ci` for runtime verification.
- **Suggestion**: Hylla's keyword search ranks by tail-symbol frequency, which buries the `cmd/till/main.go:run` function under a flood of domain-helper hits with the same `run` prefix. A search-mode that filters by `parent_id` prefix (e.g. `parent_id=github.com/hylla/tillsyn/cmd/till`) at query-input time would have surfaced the right `run` function in one query rather than requiring `Read`-based fallback navigation. Today the `parent_id` field is a response attribute, not a query filter.

---

## Droplet 4c.6.W3.D1 — Round 1

**Reviewer:** go-qa-falsification-agent (subagent, plumbing-only mode).
**Date:** 2026-05-10.
**Droplet:** `4c.6.W3.D1 — Plumb SystemPromptTemplatePath through BindingResolved + ResolveBinding`.
**Artifact under attack:** `internal/app/dispatcher/cli_adapter.go` (`BindingResolved` struct +24 LOC), `internal/app/dispatcher/binding_resolved.go` (`ResolveBinding` populator +1 LOC + doc-comment), `internal/app/dispatcher/cli_adapter_test.go` (+~35 LOC), `internal/app/dispatcher/binding_resolved_test.go` (+~36 LOC).

### Findings (W3-D1-FF<N>)

#### W3-D1-FF1 [INFO — confirmed REFUTED] — type-choice attack

**Attack:** Builder chose `string` for `BindingResolved.SystemPromptTemplatePath` (not `*string`). F.7.17 L9 (`cli_adapter.go:96-101`) names POINTER-typed fields for absent-vs-explicit-zero distinction. Probe whether the round-trip from `templates.AgentBinding.SystemPromptTemplatePath` is lossless.

**Evidence:**
- Source field (`internal/templates/schema.go:573`): `SystemPromptTemplatePath string` (verbatim, non-pointer).
- Source doc-comment (`schema.go:554-572`): "When empty the render layer falls back to the canonical built-in template for the binding's kind." Empty-string IS the documented sentinel.
- Target field (`cli_adapter.go:131`): `SystemPromptTemplatePath string` — same type.
- F.7.17 L9 doc-comment (`cli_adapter.go:96-101`) explicitly enumerates the exceptions: "AgentName, Tools, ToolsAllowed, ToolsDisallowed, Env, and CLIKind use value/slice types because their zero values (empty string, nil slice) ARE the identity element — no absent vs explicit distinction is meaningful." The new field's doc-comment (`cli_adapter.go:116-119`) extends the same rationale: "there IS no meaningful 'explicit-empty' path semantic here: an empty source value always means 'fall back to the embedded default for this binding's group.'"

**Trace:** source `string` → target `string` → no information loss at the type boundary. The semantic intent ("empty = use embedded default") is documented at both the source schema and the target struct, consistent with the existing precedent for `AgentName` / `CLIKind`. The pointer-typed neighbors (`Model` / `Effort` / `CommitAgent`) carry their own justification ("explicit zero is meaningful 'no spend allowed' / 'no commit agent configured'") that does NOT apply to a relative file path.

**Conclusion:** REFUTED. Type choice matches source schema, doc-comment justification is grounded in the existing struct convention, no round-trip information loss.

---

#### W3-D1-FF2 [INFO — confirmed REFUTED] — field-placement attack

**Attack:** Plan KindPayload says "append at end of struct (line 178+)"; Acceptance + ContextBlocks say "adjacent to AgentName for shape symmetry" with "consistent with existing field placement order (string-typed first, then pointer-typed)." The two are contradictory. Probe whether the builder's choice (between `AgentName` and `CLIKind`) violates the convention codified in the struct's own doc-comment.

**Evidence:**
- Struct doc-comment (`cli_adapter.go:96-101`): "Pointer-typed fields distinguish 'absent' ... per F.7.17 locked decision L9. AgentName, Tools, ToolsAllowed, ToolsDisallowed, Env, and CLIKind use value/slice types because their zero values ... ARE the identity element." Convention is codified — non-pointer first, pointer last.
- Struct body inspection (`cli_adapter.go:102-204`): non-pointer block is `AgentName`, then post-D1 `SystemPromptTemplatePath`, then `CLIKind`, then `Env`, then `Tools`, `ToolsAllowed`, `ToolsDisallowed`. Pointer block starts at `Model *string` (line 149) and continues through `BlockedRetryCooldown *time.Duration`.
- Appending at "line 178+" (literal end-of-struct) would put a non-pointer `string` AFTER nine pointer fields — that breaks the convention the doc-comment codifies.
- The plan's Acceptance (line 69) explicitly cites the convention rationale; the KindPayload (line 91) is a shape-hint default that the Acceptance language overrides.

**Trace:** builder chose Acceptance over KindPayload; the choice is consistent with the struct doc-comment; the alternative would have introduced a convention violation visible on every Read of the struct. Worklog § "Design decisions" (lines 1146-1165) documents the trade-off explicitly.

**Conclusion:** REFUTED. Field placement is correct; the apparent plan contradiction resolves in favor of the convention the struct doc-comment codifies. The Acceptance bullet's "adjacent to AgentName" wording lands exactly where the field sits.

---

#### W3-D1-FF3 [INFO — confirmed REFUTED] — populator fallback-path attack

**Attack:** `ResolveBinding` literal now includes `SystemPromptTemplatePath: rawBinding.SystemPromptTemplatePath`. Probe whether there is any fallback path (e.g. "no binding configured for this kind") where the resolver constructs `BindingResolved` without the new field — leaving stale-but-not-default state.

**Evidence:**
- Resolver entrypoint (`binding_resolved.go:118-156`): single `BindingResolved{...}` literal at lines 119-127; field is populated at line 121. No alternate construction site exists in the file.
- `rg -l "BindingResolved"` (executed against the repo) returns construction sites only in test files (`cli_adapter_test.go:274`, `binding_resolved_test.go:*`, `mock_adapter_test.go`, etc.) plus the resolver. Test sites use Go zero-value for omitted fields (default `""`) — correct sentinel-mapping behavior.
- No "no binding configured" fallback path exists in `ResolveBinding` — the function is a pure cascade-merge over the inputs it receives. The "no agent binding for this kind" decision lives upstream in the dispatcher's lookup path (outside D1's scope).
- nil-typed-rawBinding probe: `templates.AgentBinding` is a value type (not a pointer), so `rawBinding` cannot be nil-typed. `rawBinding.SystemPromptTemplatePath` always evaluates; no panic risk.

**Trace:** the populator is one assignment in one literal; the field's zero value is the documented sentinel; no alternate construction path exists in production code that bypasses the new assignment. Test-only construction sites correctly rely on Go's zero-value default.

**Conclusion:** REFUTED. Populator is complete; no fallback-path leak.

---

#### W3-D1-FF4 [INFO — confirmed REFUTED] — test-coverage attack

**Attack:** Probe whether the four new test sites actually cover the W3.D1 acceptance criteria: zero-value, populated round-trip, resolver empty pass-through, resolver populated pass-through. Probe specifically whether any malformed-input case is missing.

**Evidence:**
- `TestBindingResolvedZeroValueIsAllAbsent` (`cli_adapter_test.go:229-230`): adds `if br.SystemPromptTemplatePath != ""` assertion — covers zero-value sentinel.
- `TestBindingResolvedSystemPromptTemplatePath` (`cli_adapter_test.go:264-290`): covers (a) zero-value via `var zero BindingResolved`, (b) populated round-trip via `BindingResolved{SystemPromptTemplatePath: "till-go/go-builder-agent.md"}`, (c) type guard via `reflect.TypeOf(...).FieldByName(...).Type.Kind() != reflect.String`. Three sub-assertions — matches Acceptance bullet 3.
- `TestResolveBindingSystemPromptTemplatePathEmpty` (`binding_resolved_test.go:287-298`): empty source → empty resolved. Matches Acceptance bullet 4(a).
- `TestResolveBindingSystemPromptTemplatePathPopulated` (`binding_resolved_test.go:304-315`): non-empty source `"till-go/go-builder-agent.md"` → equal resolved. Matches Acceptance bullet 4(b).
- Malformed-input check (path traversal): plan RiskNotes (line 84) explicitly defers validation to D2 ("validating here would couple D1 to filesystem state"). The plan's Acceptance bullet 2 (line 70) reinforces: "Whatever the source value is — empty string or non-empty path — passes through verbatim; resolver does NOT validate path existence here." A malformed-input verbatim-pass-through test would be a useful defensive assertion but is NOT required by the plan's acceptance criteria. Builder's choice to omit is consistent with the plan's "D1 is plumbing-only" scope-lock.

**Trace:** every plan-mandated test case is implemented. The only candidate missing test (malformed input verbatim pass-through) is explicitly out-of-scope per the plan's RiskNotes.

**Conclusion:** REFUTED. Test coverage matches plan acceptance criteria. Recommendation surfaced as W3-D1-FF7 below (advisory, not blocking).

---

#### W3-D1-FF5 [INFO — confirmed REFUTED] — downstream-consumer attack

**Attack:** D2's `render.assembleAgentFileBody` is the next consumer; D2 isn't built yet but its spawn brief will assume a stable field shape. Probe whether any OTHER consumer of `BindingResolved` (besides D2's future site) reads the new field today.

**Evidence:**
- `rg -l "BindingResolved" internal cmd` returns 15 files. All consumer sites are: (a) the dispatcher core (`spawn.go`, `cli_adapter.go`), (b) the claude adapter (`cli_claude/{adapter,argv,env,init}.go`), (c) the render package (`cli_claude/render/{render,init}.go`), plus tests and mocks.
- `rg -n "SystemPromptTemplatePath" internal cmd` returns zero production consumers reading `BindingResolved.SystemPromptTemplatePath` outside D1's own files. The closest cross-references are:
  - `spawn.go:533` — doc-comment hint about the future F.7.2 plumbing.
  - `render.go:219, :323` — doc-comment hints; the existing `assembleAgentFileBody` stub still uses the F.7.3b template-installed-path path.
- No production consumer SWITCHES on `BindingResolved`'s struct shape (no reflection-keyed dispatch, no struct-equality comparison) that would have a stale-zero-value trap with the new field.

**Trace:** D1 is genuinely plumbing-only; the field exists for D2 to consume; no other production consumer reads it today. Adding a non-pointer string with `""` default is additive — every existing consumer that constructs `BindingResolved{...}` via field-named literals gets the zero-value default automatically.

**Conclusion:** REFUTED. No downstream consumer breakage; D2's spawn brief lands on a stable field type/name/zero-value contract.

---

#### W3-D1-FF6 [INFO — confirmed REFUTED] — mage-gate attack

**Attack:** Worklog claims `mage ci` was NOT run by the builder (per the appendix constraint "Do NOT run `mage install` or `mage ci`"). QA must run `mage ci` to confirm no regression introduced by the plumbing.

**Evidence (this session, executed against working directory `/Users/evanschultz/Documents/Code/hylla/tillsyn/main/`):**

`mage test-pkg ./internal/app/dispatcher`:
```
[PKG PASS] github.com/evanmschultz/tillsyn/internal/app/dispatcher (0.01s)
tests: 361, passed: 361, failed: 0
```

`mage ci`:
```
[SUCCESS] All tests passed
  3012 tests passed across 25 packages.
Minimum package coverage: 70.0%.
[SUCCESS] Coverage threshold met
[SUCCESS] Built till from ./cmd/till
```

Dispatcher-specific coverage: `internal/app/dispatcher` at 76.1% (above the 70% floor).

**Trace:** the plumbing-only addition is benign at the build layer; 3012 tests pass across 25 packages; coverage threshold met; binary builds cleanly. No regression.

**Conclusion:** REFUTED. `mage ci` green.

---

#### W3-D1-FF7 [NIT — advisory only] — defensive malformed-input test absent

**Attack:** Plan acceptance (line 70) says "Whatever the source value is — empty string or non-empty path — passes through verbatim." A defensive test that asserts the resolver passes a path-traversal input (e.g. `"../etc/passwd"` or `";rm -rf /"`) through verbatim would prove the "no dispatcher-layer validation" contract is honored AND would catch a future regression where a builder accidentally adds a length/format check at the resolver layer.

**Why advisory only:** the plan's RiskNote (line 84) explicitly defers all validation to D2 / template-Load-time; the existing test coverage IS sufficient to satisfy plan acceptance; adding a malformed-input test is a defensive extension, not a plan-mandated gap. The W3.D1 droplet is plumbing-only; over-testing is its own scope leak.

**Recommendation:** if a future round-2 of W3.D1 lands (unlikely — droplet is in `done` state with green tests + green mage ci), consider adding `TestResolveBindingSystemPromptTemplatePathMalformedPassesThrough` asserting `raw.SystemPromptTemplatePath = "../etc/passwd"` → resolved equals source verbatim. Otherwise, accept as an advisory note for D2's defensive scope.

**Conclusion:** NIT only. Not a counterexample to the W3.D1 plumbing claim; surfaced as an advisory for D2's planner / sub-planner to mirror at the consumer layer.

---

### Severity breakdown

| Severity                | Count |
| ----------------------- | ----- |
| CONFIRMED counterexample (HIGH) | 0 |
| CONFIRMED counterexample (NORMAL) | 0 |
| REFUTED (INFO) attack | 6 |
| NIT / advisory | 1 |

### Per-family attack-result table

| Attack family                     | Result   | Finding(s)       |
| --------------------------------- | -------- | ---------------- |
| Type choice (string vs *string)   | REFUTED  | W3-D1-FF1        |
| Field placement                   | REFUTED  | W3-D1-FF2        |
| Populator fallback paths          | REFUTED  | W3-D1-FF3        |
| Test coverage gaps                | REFUTED  | W3-D1-FF4        |
| Downstream consumer breakage      | REFUTED  | W3-D1-FF5        |
| mage ci regression                | REFUTED  | W3-D1-FF6        |
| Worklog accuracy                  | REFUTED  | (W3-D1-FF6 incl) |
| Defensive malformed-input test    | NIT      | W3-D1-FF7        |

### mage ci result

`mage ci`: GREEN. 3012 tests passed across 25 packages. Coverage threshold (70%) met for all packages. `till` binary built successfully from `./cmd/till`. `internal/app/dispatcher` coverage at 76.1%.

### Verdict

**PASS.** Zero CONFIRMED counterexamples across seven attack families. The W3.D1 plumbing-only droplet is sound: type choice matches source schema, field placement honors the struct's doc-comment convention, populator is complete, tests cover all plan-mandated cases, no downstream consumer breakage, mage ci green. One NIT (W3-D1-FF7 — optional defensive malformed-input test) surfaced as advisory only and explicitly out-of-scope per the plan's RiskNotes.

### Hylla Feedback

- **Query**: `mcp__hylla__hylla_search_keyword query="SystemPromptTemplatePath" artifact_ref="github.com/evanmschultz/tillsyn@main" limit=30`.
- **Missed because**: enrichment still running on the snapshot — response `enrichment still running for github.com/evanmschultz/tillsyn@main`. Same transient-availability miss the builder reported in BUILDER_WORKLOG.md § "Hylla Feedback" for this droplet. Not a Hylla schema gap; an ingestion-pipeline timing miss.
- **Worked via**: `Read` against `internal/templates/schema.go:530-660` confirming the source field at `schema.go:573` with `toml:"system_prompt_template_path"` tag; `Read` against the four production + test files in `internal/app/dispatcher/`; `rg -l "BindingResolved"` and `rg -n "SystemPromptTemplatePath"` across `internal/` + `cmd/` for consumer mapping; `mage test-pkg ./internal/app/dispatcher` + `mage ci` for runtime verification.
- **Suggestion**: when Hylla returns `enrichment still running` on a known-fresh symbol, surface a hint about which secondary indices may already be queryable today (e.g. structural keyword search on top-level field declarations may complete earlier than semantic-summary indexing). The current binary response forces a 100% fallback to `Read` + `rg` even when partial-index data may exist. Same suggestion as the builder's bullet — re-recording here so the orchestrator's drop-end Hylla-feedback rollup sees both sides of the same miss.

---

## Droplet 4c.6.W2.D7.5 — Round 1

**Reviewer:** go-qa-falsification-agent (subagent, doc-only mode).
**Date:** 2026-05-10.
**Droplet:** `4c.6.W2.D7.5 — till install CLI command (NEW — OQ#3 disposition)`.
**Artifact under attack:** `cmd/till/install_cmd.go` (NEW, 112 lines), `cmd/till/install_cmd_test.go` (NEW, 130 lines), `cmd/till/main.go` (modified — `installCmd` registration), `cmd/till/help.go` (modified — `"till install"` rich-help entry).

### Counterexample attempts

#### W2-D75-FF1 [LOW / verbatim-port fidelity] — Body diff `runInitDevConfig` → `runInstall`

- **Attack:** byte-equivalence between source `runInitDevConfig` body (`cmd/till/main.go:2042-2097`) and ported `runInstall` body (`cmd/till/install_cmd.go:57-111`). Builder claimed "verbatim port preserves" (a)–(d) in PLAN.md L216.
- **Method:** line-by-line read of both functions.
- **Result — REFUTED.** Both functions are 55-line bodies including signature. Source signature `func runInitDevConfig(stdout io.Writer, opts rootCommandOptions) error` vs ported `func runInstall(stdout io.Writer, opts rootCommandOptions) error` — only the name differs. All inner statements byte-identical: `paths, err := platform.DefaultPathsWithOptions(platform.Options{AppName: opts.appName, DevMode: true, HomeDir: opts.homeDir})`, error wrap strings (`"resolve dev paths: %w"`, `"create dev config directory: %w"`, `"stat dev config: %w"`, `"write dev config: %w"`, `"read dev config: %w"`, `"write updated dev config: %w"`), `errors.Is(err, os.ErrNotExist)` check, `config.DefaultTemplate()` call, `os.WriteFile(configPath, templateBytes, 0o644)`, `ensureLoggingSectionDebug(string(content))`, `os.WriteFile(configPath, []byte(updated), 0o644)`, `msg := "dev config already exists"` + `if created { msg = "created dev config" }`, and the closing `writeCLIKV(stdout, "Dev Config", [][2]string{{"status", msg}, {"config path", shellEscapePath(configPath)}, {"logging level", "debug"}})`. No off-by-one, no swapped argument order, no string drift. The `"Dev Config"` Laslig title is preserved byte-for-byte per W2-FF5 ROUND-2 LASLIG TITLE CONTRACT (PLAN.md L223).

#### W2-D75-FF2 [LOW / test-port fidelity] — Test body diff `TestRunInitDevConfig*` → `TestRunInstall_*`

- **Attack:** ported test bodies (`install_cmd_test.go:22-68` + `:74-129`) versus source (`main_test.go:2906-2953` + `:2955-3011`). Builder claimed "Same test body, just `[]string{\"install\"}` instead of `[]string{\"init-dev-config\"}`" (PLAN.md L220). Check for accidentally-copied `"init-dev-config"` substring assertions that would yield false positives.
- **Method:** field-by-field comparison across `t.TempDir()`, env setup, `t.Chdir`, `go.mod` write, example/existing const, `run(...)` args slice, error-format strings, assertion-loop strings, `ReadFile + Count` checks.
- **Result — REFUTED.** Both ported test bodies match source byte-for-byte modulo the args-slice rename (`"init-dev-config"` → `"install"`) AND the proportional error-format-string rename (`"run(init-dev-config) error = %v"` → `"run(install) error = %v"`, `"...in init-dev-config output, got..."` → `"...in install output, got..."`). No stray `"init-dev-config"` substring asserts in the ported tests — verified via full file read of `install_cmd_test.go`. Both `for _, want := range []string{"Dev Config", "status", "created dev config", ...}` loops in the ports match source loops in identical order. The assertion contents do NOT mention `"init-dev-config"` or `"init dev config"` anywhere — they assert on the Laslig output (`"Dev Config"` table title + kv keys), which is unchanged across rename. The test-name underscore introduction (`TestRunInstall_CreatesDebugConfig` vs source `TestRunInitDevConfigCreatesDebugConfig`) is honored per W2-FF2/W2-FF9 ROUND-2 contracts.

#### W2-D75-FF3 [LOW / consumer-tie discipline] — End-to-end vs direct-helper invocation

- **Attack:** verify both ported tests exercise the cobra registration via `run(ctx, args, ...)` end-to-end (not direct `runInstall(...)` calls). Per W2-FF3 ROUND-2 CONSUMER-TIE TEST CONTRACT (PLAN.md L222), direct-helper calls would bypass the `rootCmd.AddCommand(installCmd)` wiring and ship a non-wired command.
- **Method:** grep both test bodies for direct calls to `runInstall(`.
- **Result — REFUTED.** Both tests use `run(context.Background(), []string{"--app", "tillsyn-init", "install"}, &out, io.Discard)` (`install_cmd_test.go:43` + `:106`). Zero direct `runInstall(` invocations in test files. Cobra round-trip is exercised; if `installCmd` were not added to `rootCmd.AddCommand(...)` at `main.go:1907`, the test would fail with `unknown command "install"` — exactly the RED state the builder reported pre-impl (BUILDER_WORKLOG.md L1314-1316).

#### W2-D75-FF4 [LOW / pointer-signature refactor] — Closure capture of `rootOpts` for cobra flag-parse-mutation

- **Attack:** verify the builder's pointer-signature refactor (`newInstallCommand(stdout, rootOpts *rootCommandOptions)`) is the correct fix and not over-engineering. Compare against the legacy sibling `initDevConfigCmd := &cobra.Command{...}` block (`main.go:1884-1903`) which uses value semantics in `return runInitDevConfig(stdout, rootOpts)` (L1901) and still works.
- **Method:** Read `main.go:508-513` (PersistentFlags binding), L1884-1907 (legacy registrations), `install_cmd.go:27-50` (new factory). Trace closure-variable binding for the two patterns.
- **Result — REFUTED.** The legacy `initDevConfigCmd` block is constructed INLINE inside the enclosing `run(ctx, args, stdout, stderr)` function, so the `func(_ *cobra.Command, _ []string) error { return runInitDevConfig(stdout, rootOpts) }` closure captures the OUTER-scope `rootOpts` variable BY REFERENCE per Go closure semantics. Cobra's `PersistentFlags().StringVar(&rootOpts.appName, "app", ...)` (L511) writes through the pointer to the SAME outer-scope variable. Both legacy and pointer-refactor paths therefore read the live mutated struct at RunE invocation time. However, the refactored `newInstallCommand(stdout, rootOpts rootCommandOptions)` would have received a value-COPY at function entry, and the inner closure would have captured that copy's address — NOT the outer-scope variable cobra writes to. The builder's pointer fix (`*rootCommandOptions` param + `*rootOpts` deref at RunE-time) is the correct minimal patch: it routes cobra's mutation target back to the same variable the closure reads. NOT over-engineering — value receiver would have produced the pre-parse-default bug the builder caught in TDD red round 2 (BUILDER_WORKLOG.md L1323-1332).

#### W2-D75-FF5 [LOW / concurrency] — Data-race on pointer-aliased `rootOpts`

- **Attack:** since `&rootOpts` is now shared between cobra flag-parse and the RunE closure, check whether parse and RunE could fire on different goroutines (data race).
- **Method:** Context7 on cobra concerning RunE invocation goroutine ordering + read of `fang.Execute` call at `main.go:1909` and surrounding `run` function structure.
- **Result — REFUTED.** Cobra's `cmd.Execute()` is synchronous — it parses flags, walks the subcommand tree, and invokes the matched `RunE` callback all on the calling goroutine. `fang.Execute(ctx, rootCmd)` follows the same synchronous contract (it adds CTRL-C signal handling but does not fan out RunE to a separate goroutine). The lifecycle is strict: parse-mutates-`rootOpts`, then-invokes-RunE-which-reads-`rootOpts`. No concurrent access; no race. The pointer-aliased shared state is read-after-write in a single-goroutine ordering, exactly the idiom cobra programs are written to.

#### W2-D75-FF6 [LOW / regression — legacy `init-dev-config`] — `TestRunInitDevConfigCreatesDebugConfig` + `TestRunInitDevConfigUpdatesExistingConfig` survive

- **Attack:** D7.5 explicitly leaves the legacy `init-dev-config` registration + tests in place (D8 removes). Verify neither legacy test regresses.
- **Method:** Read source-test bodies + run `mage test-pkg ./cmd/till`.
- **Result — REFUTED.** Source tests at `main_test.go:2906-2953` + `:2955-3011` are untouched (verified via Read). `mage test-pkg ./cmd/till` GREEN: 257/257 tests passed (full output captured in W2-D75-FF12 below). No regression in the legacy `init-dev-config` path.

#### W2-D75-FF7 [LOW / registered-commands assert] — `main_test.go:476` hardcoded command list

- **Attack:** `main_test.go:476` hardcodes a string slice of command tokens (`"serve", "mcp", "auth", "project", "embeddings", "capture-state", "kind", "lease", "handoff", "export", "import", "paths", "init-dev-config"`). D3a added `init` (yet absent from this list — already flagged as Finding 1.11 in the prior round). D7.5 adds `install` — also absent. Does this break or weaken the test?
- **Method:** Read `main_test.go:460-487` (TestRunRootHelp), trace the assertion semantics.
- **Result — REFUTED as breakage, CONFIRMED as coverage gap (audit-trail only).** The loop only checks PRESENCE — `if !strings.Contains(output, want) { t.Fatalf(...) }`. Missing `"install"` from the list means the test does NOT assert install appears in root help, but it ALSO does not fail. `mage test-pkg ./cmd/till` GREEN confirms. The coverage-gap signal is identical to D3a's Finding 1.11 (Round 1 falsification listed the same gap for `"init"`). Recommended follow-up: a single droplet-grouped sweep adding `{"init", "install"}` to the hardcoded list (and the rich-help table at `:498-736`) when the help text stabilizes. Not a counterexample against D7.5's PASS verdict — D7.5's acceptance does not require expanding the registered-commands assertion list.

#### W2-D75-FF8 [LOW / rich-help table coverage] — `main_test.go:498-736` table missing `install` row

- **Attack:** the rich-help table-test at `:498-736` contains an `init-dev-config` row (`:732-734`) but no `install` row. Per the parallel-test-coverage assumption (D8 removes the `init-dev-config` row in tandem with the legacy code), this could leave a gap where neither row covers the dev-config help-output.
- **Method:** Read `main_test.go:498-736`; check whether D7.5 acceptance explicitly required a new row.
- **Result — REFUTED as breakage, CONFIRMED as coverage gap (audit-trail only).** PLAN.md L225-226 specifies "mage test-pkg ./cmd/till passes — both old init-dev-config tests AND new install tests are green. (Old tests stay until D8.)" The plan does NOT require D7.5 to add an `install` row to the rich-help table. `mage test-pkg ./cmd/till` GREEN. D8 must add the `install` row in tandem with removing the `init-dev-config` row — flagged here for the D8 pre-flight check. D7.5 verdict unaffected.

#### W2-D75-FF9 [LOW / help-text alphabetical ordering] — `commandHelpSpecs` map key order

- **Attack:** builder placed `"till install"` entry at `help.go:407-421` immediately after `"till init-dev-config"` (L393-406). Verify this is correct alphabetically AND that the placement is functionally significant.
- **Method:** Read `help.go:1-50` (map declaration) + `:425-461` (consumer).
- **Result — REFUTED (placement is cosmetic).** `commandHelpSpecs` is `map[string]commandHelpSpec` (L16). Go map iteration order is undefined. `applyCommandHelpSpecs` (L434-447) walks the COMMAND TREE via `walkCommands(root, ...)` and looks up each command's `cmd.CommandPath()` in the map. Source-code positioning of map literal entries is cosmetic — has zero effect on runtime help-output ordering. Alphabetical positioning (`init` < `init-dev-config` < `install` since `-` 0x2D < `t` 0x74 < `…`) is achieved anyway: `init` at L377, `init-dev-config` at L393, `install` at L407. Source readability preserved; no functional risk.

#### W2-D75-FF10 [LOW / help cross-reference] — Long-text distinction between `install` and `init`

- **Attack:** OQ#3 (PLAN.md L33-36) requires that `till install` (per-machine setup) and `till init` (per-project setup) are distinct user-visible commands with non-overlapping scopes. Verify the help Long text makes this distinction clear, not just internally documented.
- **Method:** Read `install_cmd.go:31-39` (cobra Long) + `help.go:407-421` (rich-help Long).
- **Result — REFUTED.** Both copies of the Long text include the explicit sentence "This is a per-machine setup command — see till init for per-project setup." (`install_cmd.go:38-39` cobra Long + `help.go:413-414` rich-help Long). The cross-reference points the user at the sibling command. The Short text differs deliberately: `install` Short is "Bootstrap the local Tillsyn dev environment (creates the dev config, enforces [logging] level = \"debug\")"; `init` Short is "Seed a Tillsyn project (agents directory, agents.toml, .gitignore, optional .mcp.json)" — non-overlapping scopes.

#### W2-D75-FF11 [LOW / routed Unknown verification] — `newInitCommand` same-shape value-capture bug

- **Attack:** builder routed an Unknown (BUILDER_WORKLOG.md L1417-1430) claiming D3a's `newInitCommand(stdout io.Writer, rootOpts rootCommandOptions) *cobra.Command` carries the same value-capture latent bug that bit D7.5. Verify the claim is correct AND that the bug is truly latent (not currently firing).
- **Method:** Read `init_cmd.go:1-58` (full file) + `init_cmd_test.go` per builder's worklog references.
- **Result — REFUTED (Unknown is VERIFIED LATENT).** `init_cmd.go:16` signature is `func newInitCommand(stdout io.Writer, rootOpts rootCommandOptions) *cobra.Command` — value receiver. The `RunE` closure (L37-46) calls `runInitTUI(stdout, rootOpts)` reading the local-copy `rootOpts`. Today this would surface the same `--app`/`--home` freeze bug — BUT `runInitTUI` is itself a stub (L54-58): `func runInitTUI(stdout io.Writer, opts rootCommandOptions) error { _ = stdout; _ = opts; return errors.New("till init: TUI walk not yet wired (W2.D4)") }`. The function consumes neither `stdout` nor `opts`; it returns the stub error before reading any field. Same for the JSON-stub branch at L42-44 — returns `errors.New(...)` before touching path resolution. So the bug is genuinely latent today (no test failure produced), but WILL surface when D4 wires the bubbletea walk (D4 needs `appName`/`homeDir` for path resolution per PLAN.md L130-131) and when D5 wires the file-copy pipeline (uses `<dev-paths>` via `appName`). Builder's routed Unknown is CORRECT; the fix in D4 / D5 will be a sibling change of signature `newInitCommand(stdout io.Writer, rootOpts *rootCommandOptions) *cobra.Command` + RunE deref `*rootOpts`. The Unknown is properly flagged for orchestrator visibility; D7.5's verdict is unaffected because D7.5's surface (`install_cmd.go`) is independent of `init_cmd.go`.

#### W2-D75-FF12 [HIGH-tier gate / build] — `mage test-pkg ./cmd/till` + `mage ci`

- **Attack:** the package-level gate + the full CI target must pass on the staged D7.5 changes. Per WORKFLOW.md Phase 5 step 1, the package gate is a precondition for LLM QA verdict; per CLAUDE.md "NEVER run raw go test...always mage", QA must use mage.
- **Method:** `mage test-pkg ./cmd/till` then `mage ci` from `main/` cwd.
- **Result — REFUTED (gates pass).**
  - `mage test-pkg ./cmd/till`: `[PKG PASS] github.com/evanmschultz/tillsyn/cmd/till (0.00s)`. Test summary: 257 passed / 0 failed / 0 skipped / 1 package.
  - `mage ci`: `[SUCCESS] All tests passed — 3012 tests passed across 25 packages.` `[SUCCESS] Coverage threshold met — All packages are at or above 70.0% coverage.` `cmd/till` package coverage: 75.7%. Build: `[SUCCESS] Built till from ./cmd/till`.

#### W2-D75-FF13 [LOW / file-lock graph] — D3a → D7.5 → D8 chain on `cmd/till/main.go`

- **Attack:** D7.5's `Blocked by: D3a` is on the `cmd/till/main.go` file lock. Verify the actual `main.go` change is additive (no overlap with D3a's earlier change beyond the single `rootCmd.AddCommand` line + the new `installCmd := newInstallCommand(...)` insertion).
- **Method:** `git diff cmd/till/main.go`.
- **Result — REFUTED.** Diff against committed D3a state shows exactly 2 line changes: (a) insertion of `installCmd := newInstallCommand(stdout, &rootOpts)` at L1906, (b) extension of the `rootCmd.AddCommand(...)` call at L1907 to append `, installCmd` at the end of the argument list. No mutation of D3a's `initCmd := newInitCommand(stdout, rootOpts)` (L1905). Additive-only edit; lock graph respected.

#### W2-D75-FF14 [LOW / error-wrap continuity] — `%w` chain across rename

- **Attack:** the source `runInitDevConfig` wraps errors via `fmt.Errorf("...: %w", err)`. The ported `runInstall` must preserve `%w` (not switch to `%v`) so callers can still `errors.Is` / `errors.As`.
- **Method:** Read `install_cmd.go:62-99` for every error-return.
- **Result — REFUTED.** Every error-return uses `%w`: L68 `"resolve dev paths: %w"`, L73 `"create dev config directory: %w"`, L79 `"stat dev config: %w"`, L86 `"write dev config: %w"`, L93 `"read dev config: %w"`, L98 `"write updated dev config: %w"`. Identical to source. `errors.Is(os.ErrNotExist)` at L78 also preserved (same line as source L2064). Error chain semantics preserved.

#### W2-D75-FF15 [LOW / hidden coupling — same-package helpers] — `shellEscapePath` + `ensureLoggingSectionDebug` + `writeCLIKV`

- **Attack:** `runInstall` references helpers (`shellEscapePath` L108, `ensureLoggingSectionDebug` L95, `writeCLIKV` L106). These are unexported and live in `main.go`. Verify cross-file same-package resolution works AND that no shadowed/duplicate names exist in `install_cmd.go`.
- **Method:** Read `install_cmd.go` (full); check imports + identifier declarations.
- **Result — REFUTED.** `install_cmd.go` declares package `main` at L1 (same package as `main.go`). No `func shellEscapePath`, `func ensureLoggingSectionDebug`, or `func writeCLIKV` declared in `install_cmd.go`. All three resolve via package linkage to the definitions in `main.go` (`shellEscapePath` at `main.go:2100`, others nearby). `mage ci` build green confirms no link-time collision.

### Severity breakdown

- **HIGH:** 0
- **MEDIUM:** 0
- **LOW:** 15 (all REFUTED counterexamples; 2 have CONFIRMED coverage-gap audit-trail signals — W2-D75-FF7, W2-D75-FF8 — routed to D8 pre-flight)

### mage ci result

```
[SUCCESS] All tests passed
  3012 tests passed across 25 packages.

[SUCCESS] Coverage threshold met
  All packages are at or above 70.0% coverage.
  cmd/till coverage: 75.7%

Build
[SUCCESS] Built till from ./cmd/till
```

### Summary

**Verdict: pass.** Counterexample count: 0. All 15 attack families exhausted with no concrete counterexample against D7.5's acceptance criteria. The verbatim port of `runInitDevConfig` → `runInstall` is byte-equivalent modulo function name (W2-D75-FF1). The two ported tests are byte-equivalent modulo args-slice rename and proportional error-format-string rename (W2-D75-FF2). Both tests exercise the cobra registration end-to-end via `run(...)` (W2-D75-FF3). The pointer-signature refactor (`newInstallCommand(stdout, rootOpts *rootCommandOptions)`) is the correct minimal patch — value-receiver would have frozen pre-parse `--app`/`--home` defaults, which the builder caught in TDD red round 2 and fixed in red round 3 → green (W2-D75-FF4). No data race — cobra `Execute` is synchronous and `fang.Execute` preserves that contract (W2-D75-FF5). Legacy `init-dev-config` registration + tests survive intact (W2-D75-FF6). `mage test-pkg ./cmd/till` and `mage ci` both green (W2-D75-FF12). Error-wrap chain preserved (W2-D75-FF14). Cross-file same-package helper resolution clean (W2-D75-FF15). Builder's routed Unknown about `newInitCommand`'s latent value-capture bug is VERIFIED LATENT (W2-D75-FF11) and properly routed forward to D4/D5.

Two audit-trail coverage gaps surfaced (W2-D75-FF7 + W2-D75-FF8): the hardcoded registered-commands list at `main_test.go:476` and the rich-help table-test at `:498-736` do not include `"install"` (they also still don't include `"init"` per D3a's Finding 1.11). Neither is required by D7.5's acceptance; both should be addressed by D8 in tandem with removing the `init-dev-config` row. Flagged for D8 pre-flight. PASS verdict for D7.5 unaffected.

### Hylla Feedback

- **Query**: `mcp__hylla__hylla_search_keyword query="newInstallCommand"` and `query="init-dev-config"` (both `node_type=block`, `fields=[content]`, `artifact_ref=github.com/evanmschultz/tillsyn@main`).
- **Missed because**: `newInstallCommand` returns zero results because the file `cmd/till/install_cmd.go` is a NEW Go file in this drop, postdating Hylla's snapshot 5. `init-dev-config` keyword search returned only `internal/config/*` symbol matches (Config / DefaultTemplate / etc.) — the `runInitDevConfig` Go function in `cmd/till/main.go` did not surface despite being indexed in the same snapshot, suggesting either keyword-rank suppression (function names without an exported tail-symbol form may rank below struct-method results) or the search-mode filtering against `visibility=public_only` may have hidden the unexported `runInitDevConfig`. Same staleness + ranking pattern the builder reported in BUILDER_WORKLOG.md L1432-1454.
- **Worked via**: `Read` against `cmd/till/install_cmd.go` (full), `cmd/till/install_cmd_test.go` (full), `cmd/till/main.go` ranges (`490-575`, `1860-1908`, `2040-2110`), `cmd/till/init_cmd.go` (full), `cmd/till/help.go` ranges (`1-50`, `370-461`), `cmd/till/main_test.go` ranges (`460-487`, `700-800`, `2900-3015`), `cmd/till/help_alias.go` (full), `git diff cmd/till/main.go` + `git diff cmd/till/help.go` for the staged delta. Used `mage test-pkg ./cmd/till` and `mage ci` for runtime verification.
- **Suggestion**: when `visibility=public_only` is the default search mode but the target is an UNexported function in a `main` package (where the exported/unexported distinction has no semantic meaning since `package main` is never imported), the default mode actively hides relevant matches. Two options: (a) auto-detect `package main` and treat all top-level symbols as effectively-exported for search ranking, or (b) surface a hint in the response when zero results return for what looks like a well-known symbol, suggesting the user retry with `visibility_mode=include_private`. Today the binary "zero results" message gives no signal that the visibility filter may be the cause.
