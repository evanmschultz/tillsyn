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

---

## Droplet 4c.6.W5.D3 — Round 1

**QA falsification agent.** Pass over droplet `4c.6.W5.D3 — Drop go- prefix from agent_name in renamed till-go.toml + remove tools from frontmatter + W5-D2-FF1 doc-comment absorption`.

**Claim under attack.** Builder reports (BUILDER_WORKLOG.md L1714-1755): (a) `go-` prefix dropped from every `agent_name` in `internal/templates/builtin/till-go.toml`; (b) `tools = []` placeholder rows removed from all 11 agent_bindings sections; (c) the 3 W5-D2-FF1 doc-comment absorption sites updated with dual-history annotations; (d) `model = "opus"` and other `AgentBinding.Validate`-required fields KEPT in till-go.toml per the appendix CRITICAL constraint (schema removal deferred to Drop 4c.7); (e) frontmatter strip on placeholder `<group>/*.md` files was a no-op verification because W1.D1 already shipped them with `name:` + `description:` only; (f) legacy `go-*-agent.md` files retained in `builtin/agents/till-go/` as transitional residue.

### Attack 1 — Surviving `go-` prefix in agent_name values

**Reproducer.** `git grep -nE 'agent_name = "go-' -- internal/templates/builtin/*.toml`.

**Result.** Zero hits. Every `[agent_bindings.<kind>] agent_name` in `till-go.toml` is now bare (`builder-agent`, `planning-agent`, `research-agent`, `qa-proof-agent`, `qa-falsification-agent`, plus the pre-existing bare `commit-message-agent` and `orchestrator-managed`). Direct diff inspection of `till-go.toml` confirms 6 rebadged sites (plan / research / build / plan-qa-proof / plan-qa-falsification / build-qa-proof / build-qa-falsification — 7 sites total of which one was the `research` row mapping `agent_name = "go-research-agent"` → `"research-agent"`). REFUTED.

### Attack 2 — Frontmatter not stripped (`tools:` / `model:` / sibling keys surviving)

**Reproducer.** `git grep -nE '^(tools|model|allowedTools|disallowedTools|tools_allow|tools_deny|max_tries|max_turns):' -- 'internal/templates/builtin/agents/**/*.md'`.

**Result.** Zero hits across all 27 agent placeholder MD files (7 + 8 + 12 across till-gdd/till-gen/till-go inclusive of the 5 legacy `go-*-agent.md` placeholders + `orchestrator-managed.md`). Spot-Read on `till-go/builder-agent.md`, `till-go/qa-proof-agent.md`, `till-go/qa-falsification-agent.md`, `till-go/go-builder-agent.md`, `till-go/go-qa-proof-agent.md`, and `till-gen/orchestrator-managed.md` confirms each carries ONLY `name:` + `description:` between the `---` fences. Builder's design-decision note (L1733) that "every placeholder MD already shipped with ONLY `name:` + `description:` frontmatter — W1.D1 followed SKETCH § 15 from inception" matches the git history (`git log -- internal/templates/builtin/agents/till-go/builder-agent.md` returns only `11eec48 feat(templates): w1.d1 placeholder agent dirs and embed list`, no later strip-commit). The frontmatter strip is a no-op verification, not a stripping action — but the appendix's acceptance criterion (zero `tools:`/`model:` hits in agent MD frontmatter) holds. REFUTED.

### Attack 3 — `tools = []` placeholder rows surviving in till-go.toml

**Reproducer.** `git diff internal/templates/builtin/till-go.toml | grep -E '^-tools = \[\]' | wc -l` AND `git grep -nE '^tools = \[\]' -- internal/templates/builtin/till-go.toml`.

**Result.** Diff shows 11 `-tools = []` removed lines (every agent_bindings section + the 4 orchestrator-managed coordination kinds + commit). Post-diff grep returns zero remaining `tools = []` rows. REFUTED.

### Attack 4 — `model = "opus"` removed (would break `Validate`)

**Reproducer.** Read `internal/templates/builtin/till-go.toml` post-diff for `model = "opus"` in each `[agent_bindings.<kind>]` block. Run `mage testFunc ./internal/templates 'TestDefaultTemplateAgentBindingsCoverAllKinds|TestDefaultTemplateBuildersRunOpus'`.

**Result.** Every `[agent_bindings.<kind>]` block in till-go.toml retains `model = "..."` (opus for the 7 opus-bound kinds; haiku for commit). `TestDefaultTemplateAgentBindingsCoverAllKinds` calls `binding.Validate()` per kind (`embed_test.go:392`) and `TestDefaultTemplateBuildersRunOpus` asserts `binding.Model != "opus"` would fatal (`embed_test.go:421`). Both tests PASS in the targeted run (`2/2 PASS, 1.33s`) and again as part of the full `mage ci` sweep below. Builder's decision to defer `model` field removal to Drop 4c.7 matches the appendix's OUT-OF-SCOPE constraint and is the only outcome consistent with `AgentBinding.Validate`'s current shape at `schema.go:776`. REFUTED.

### Attack 5 — Other `Validate`-required fields removed (`max_tries`, `max_turns`, `effort`)

**Reproducer.** Read till-go.toml for `max_tries`, `max_turns`, `effort`, `max_budget_usd` per `[agent_bindings.<kind>]`. Check `AgentBinding.Validate` at `internal/templates/schema.go:776-790` for required-non-zero checks.

**Result.** Every binding retains `effort = ...` + `max_tries = ...` + `max_budget_usd = ...` + `max_turns = ...` + `max_rule_duration = ...` + `blocked_retries = ...` + `blocked_retry_cooldown = ...`. `binding.Validate()` passes for all 12 kinds in `TestDefaultTemplateAgentBindingsCoverAllKinds`. Schema-level field removal correctly deferred to Drop 4c.7. REFUTED.

### Attack 6 — W5-D2-FF1 absorption: site count and dual-history annotation pattern

**Reproducer.** Inspect the 3 declared sites in builder worklog:
1. `internal/templates/load.go` doc-comments — `git diff internal/templates/load.go` for `← default-go.toml + default-generic.toml` annotation.
2. `internal/templates/embed.go` — `git diff` for the W1.D1 cross-droplet handoff doc-comment rebadge.
3. `internal/app/auto_generate_steward.go` — `git diff` for the `default-generic vs default-go ← rebadged` annotation.

**Result.** All three sites updated with the dual-history pattern. Specifically:

- `load.go:388` — `default-go.toml + default-generic.toml` → `till-go.toml + till-gen.toml ← default-go.toml + default-generic.toml, rebadged in Drop 4c.6 W5.D1 + W5.D2`. PRESENT.
- `load.go:1241` — same rebadge pattern on the `buildBlockedByGraph`-adjacent doc-comment. PRESENT.
- `load.go:1385-1389` — `default-go.toml` → `till-go.toml (rebadged from default-go.toml in Drop 4c.6 W5.D1)` on the structural-type validator's no-op-against-default note. PRESENT.
- `load.go:2098-2103` — paragraph describing why `embeddedAgentLibraryShipped` matters historically; updated with bilingual note "at that historical point in time default-go.toml — rebadged to till-go.toml in Drop 4c.6 W5.D1 — referenced 'go-builder-agent', 'go-planning-agent', etc... Drop 4c.6 W5.D3 dropped the `go-` prefix; current names are bare `builder-agent`, `planning-agent`, etc." PRESENT.
- `embed.go:55-68` — W1.D1 cross-droplet handoff doc-comment rebadged with the W5.D3 outcome: till-go.toml now references bare names; legacy `go-*-agent.md` placeholders transition to "transitional residue from W1.D1." PRESENT.
- `auto_generate_steward.go:108-110` — `default-generic vs default-go` → `till-gen vs till-go ← default-generic vs default-go, rebadged in Drop 4c.6 W5.D1 + W5.D2`. PRESENT.

Builder also touched `till-gen.toml` (lines 30-50 doc-comment) with the bare-name-convention rationale + forward-pointing historical example. That's a 4th annotation site beyond the 3 declared in the worklog "Files touched" section — it's accurate and consistent but slightly under-counted in the worklog. NIT, not a finding.

REFUTED.

### Attack 7 — W3.D2 `TestAssembleAgentFileBody_EmbeddedDefault` chain break

**Reproducer.** Read `render_test.go:839-870`. Run `mage testFunc ./internal/app/dispatcher/cli_claude/render TestAssembleAgentFileBody_EmbeddedDefault`.

**Result.** Test uses `AgentName: "go-builder-agent"` (line 851) and asserts the rendered body contains `# PLACEHOLDER` substring AND `name: ` substring. Both come from the embedded `builtin/agents/till-go/go-builder-agent.md` legacy placeholder file. W5.D3 left that file in place per builder design-decision L1734 ("legacy `go-*-agent.md` files LEFT in place ... orphaned-but-harmless residue"). The file's `name: go-builder-agent` frontmatter line and `# PLACEHOLDER` body line both survive (verified via `Read`). Test PASSES (`1 test passed across 1 package`, 1.30s). REFUTED.

### Attack 8 — Tests / testdata referencing legacy `go-*-agent` names

**Reproducer.** `git grep -nE 'go-(builder|planning|qa-proof|qa-falsification|research)-agent' -- 'internal/templates/' 'cmd/' 'internal/app/' 'internal/adapters/'`.

**Result.** Many hits across:
- `internal/templates/agent_binding_test.go`, `catalog_test.go`, `context_rules_test.go`, `load_test.go`, `schema_test.go`, `testdata/*.toml` — fixtures that hardcode the legacy `go-builder-agent` / `go-planning-agent` names.
- `cmd/till/dispatcher_cli_test.go`, `internal/app/dispatcher/**/*_test.go`, `internal/app/template_service_test.go`, `internal/adapters/server/mcpapi/extended_tools_test.go` — dispatcher and mcpapi fixtures using `go-builder-agent`.

All such references resolve through the LEGACY 5 placeholder files in `builtin/agents/till-go/go-*-agent.md` (still embed-listed at `embed.go:98-102`) for `defaultAgentLookupFn`'s walker. With the legacy placeholders retained as transitional residue, every test passes — confirmed by `mage ci` running all 3028 tests across 25 packages green (see below). The builder explicitly routed "legacy `go-*-agent.md` cleanup" forward as deferred (L1734 + L1750). REFUTED.

### Attack 9 — Cross-group fallback chain post-rename

**Reproducer.** Trace `orchestrator-managed` AgentName resolution under the post-W5.D3 + post-W3.D2 setup. `till-go.toml`'s 4 coordination-kind bindings carry `agent_name = "orchestrator-managed"`; render.go's `readEmbeddedTierAgent` first tries `builtin/agents/till-go/orchestrator-managed.md` (does NOT exist), then falls back to `builtin/agents/till-gen/orchestrator-managed.md` (DOES exist). The W3.D2 test `TestAssembleAgentFileBody_CrossGroupFallbackToTillGen` asserts this path. Run `mage testFunc ./internal/app/dispatcher/cli_claude/render TestAssembleAgentFileBody_CrossGroupFallback`.

**Result.** `TestAssembleAgentFileBody_CrossGroupFallbackToTillGen` + `_CrossGroupFallbackMissesBothGroups` both PASS within the W3.D2 build round (BUILDER_WORKLOG.md L1660 "TestAssembleAgentFileBody_CrossGroupFallbackToTillGen → PASS"). Within the full `mage ci` sweep below, the entire render package passes 30/30 tests. The W5.D3 prefix-strip does not interact with the cross-group fallback path — fallback is driven by group dir choice + basename presence, both unchanged. REFUTED.

### Attack 10 — Legacy placeholders' OWN doc-comments are now wrong

**Reproducer.** Read `builtin/agents/till-go/go-builder-agent.md` lines 8-13. The placeholder says: "This placeholder satisfies the W0.5 `validateAgentBindingNames` validator's embedded-tier lookup for the legacy `go-builder-agent` name still referenced by `internal/templates/builtin/default-go.toml`. The default-go.toml rename + name-strip lands in Drop 4c.6 W5.D1 / W5.D3; this file goes away alongside that cleanup."

**Result.** Post-W5.D3, default-go.toml no longer exists (renamed to till-go.toml in W5.D1) AND till-go.toml no longer references `go-builder-agent` (renamed to `builder-agent` in W5.D3). The placeholder's claim "still referenced by ... default-go.toml" is now factually stale — it's referenced only by TEST fixtures (`internal/templates/load_test.go`, `cmd/till/dispatcher_cli_test.go`, etc.). Also "this file goes away alongside that cleanup" was NOT executed — the file remains.

**Severity.** Doc-comment staleness in a placeholder MD file that's not user-facing and is slated for cleanup in a follow-up drop. NOT a functional defect. NIT (W5-D3-NIT1) — flagged for the follow-up drop that deletes the legacy `go-*-agent.md` placeholders; either delete the files OR refresh their doc-comments to point at the test fixtures as the surviving consumer. REFUTED (no functional counterexample).

### Attack 11 — Builder did NOT touch till-gen.toml? (scope question)

**Reproducer.** `git diff internal/templates/builtin/till-gen.toml`.

**Result.** Builder DID touch till-gen.toml (lines 30-50 doc-comment rebadge). This is consistent with the orchestrator's appendix prompt ("rename in till-go.toml ... W5-D2-FF1 absorption ... all 3 sites updated") and consistent with the broader "every doc-comment referencing the old shape must be updated" Hygiene-style scope. The till-gen.toml edit is doc-comment ONLY (no shape change — `agent_bindings` still omitted by design). Not a scope creep; not a counterexample. REFUTED.

### Attack 12 — `mage ci` regression across all 25 packages

**Reproducer.** `mage ci` against the current uncommitted-but-staged worktree (W5.D3 + W2.D3b + W3.D2 + W3.D3 all unstaged-modified together — per the dispatcher batch instructions).

**Result.** GREEN.

```
Test summary
  tests: 3028
  passed: 3028
  failed: 0
  skipped: 0
  packages: 25
  pkg passed: 25
  pkg failed: 0
  pkg skipped: 0

[SUCCESS] All tests passed
  3028 tests passed across 25 packages.

[SUCCESS] Coverage threshold met
  All packages are at or above 70.0% coverage.

Build
[SUCCESS] Built till from ./cmd/till
```

No package regression. Coverage floor (70%) held. Build clean. REFUTED.

### Attack 13 — Concurrency / data-race surface

**Reproducer.** Build (`mage ci` runs with `-race`) — all 3028 tests run under `-race`. The W5.D3 change is TOML-content-only + Go doc-comments only — zero new goroutines, zero new shared state. Embedded FS reads in `defaultAgentLookupFn` use `DefaultTemplateFS.ReadDir` and `fs.ReadFile` which are read-only against immutable embed data.

**Result.** No race surface. `mage ci -race` clean. REFUTED.

### Attack 14 — YAGNI pressure on `reservedInitGroups` / similar?

**Reproducer.** Scope review — does W5.D3 introduce abstractions without ≥2 use cases?

**Result.** W5.D3 is a 9-file rename/cleanup pass — no new types, no new functions, no new interfaces, no new abstractions. The only structural changes are deletions (`tools = []` rows removed) and prefix-strip (`go-X-agent` → `X-agent` in agent_name values). REFUTED.

### Attack 15 — `mage install` invocation surface

**Reproducer.** `git grep -n "mage install" -- workflow/drop_4c_6/BUILDER_WORKLOG.md`.

**Result.** Zero invocations of `mage install` in the W5.D3 builder round. Builder ran `mage test-pkg ./internal/templates` + `mage test-pkg ./internal/app` + `mage test-pkg ./internal/adapters/server/mcpapi`; the orchestrator runs `mage ci`. No CRITICAL counterexample. REFUTED.

### Attack 16 — Raw `go build` / `go test` bypass

**Reproducer.** Inspect BUILDER_WORKLOG.md W5.D3 section for any `go build` / `go test` / `go vet` invocations.

**Result.** All builder commands logged in L1739-1744 are `mage ...` invocations. No raw `go` toolchain bypass. REFUTED.

### Attack 17 — Hidden dependencies / `init()` side effects from rename

**Reproducer.** The rename touches package-level `var embeddedAgentLibraryShipped` (declared at `load.go:2116` via init-time func-call). Did the rename affect the value? Read `embeddedAgentGroups` at `load.go:2086`.

**Result.** `embeddedAgentGroups = []string{"till-gen", "till-go", "till-gdd"}` unchanged. `embeddedAgentLibraryShipped` walks all 3 groups and returns `true` because till-gen + till-go + till-gdd each contain ≥1 `.md` placeholder. The init-time probe's value is unchanged by W5.D3 (no group dirs added or removed; no `.md` files deleted). REFUTED.

### Attack 18 — Test-order coupling

**Reproducer.** `mage ci` runs tests with `-race -count=1` (no test caching) and Go's default randomized test order within a package. If any test depended on legacy `go-` names AND a sibling test depended on bare names being absent, order-coupling could surface.

**Result.** All 3028 tests pass under `-race -count=1`. No order-coupling regression. REFUTED.

### Attack 19 — Description / scope drift (builder silently re-interpreted scope)

**Reproducer.** Compare appendix's KindPayload `shape_hint` ("drop go- prefix from agent_name; remove tools field; remove model field") against builder's design decision (kept `model`).

**Result.** Builder's L1732 design-decision note explicitly addresses this: appendix CRITICAL constraint says schema-level removal is OUT OF SCOPE → `model` removal would fail `AgentBinding.Validate` (which requires non-empty `Model`) → tests like `TestDefaultTemplateAgentBindingsCoverAllKinds` would break. Builder resolved the conflict by following the CRITICAL constraint over the `shape_hint`, and routed the deferred `model` removal forward to Drop 4c.7 in the "Out-of-scope / routed back to orchestrator" section L1748. This is the only outcome consistent with the locked CRITICAL constraint — not silent scope drift. The orchestrator's appendix attack-12 explicitly endorses this: "The decision to defer schema removal to Drop 4c.7 is correct per OUT-OF-SCOPE constraint." REFUTED.

### Verdict

**PASS.** No counterexample produced against W5.D3's acceptance criteria.

- 19 attack vectors exhausted. 18 REFUTED with evidence. 1 NIT logged (W5-D3-NIT1: legacy placeholder doc-comments now stale; route to the follow-up cleanup drop alongside the file deletions).
- `mage ci` GREEN: 3028 tests / 25 packages / coverage ≥ 70% / build clean.
- Builder's three load-bearing tests (`TestDefaultTemplateAgentBindingsCoverAllKinds`, `TestDefaultTemplateBuildersRunOpus`, `TestAssembleAgentFileBody_EmbeddedDefault`) all PASS in targeted runs.
- W5-D2-FF1 absorption: 3 declared sites updated, plus 1 implicit fourth site at till-gen.toml that's accurate and consistent (worklog under-counts at the "Files touched" header but the diff matches).
- The OUT-OF-SCOPE `model` field removal is correctly deferred to Drop 4c.7 per the appendix CRITICAL constraint, with audit trail in Out-of-scope section.

### Findings tagged

- (no W5-D3-FF<N> findings raised — zero counterexamples)
- W5-D3-NIT1 — stale doc-comments in `builtin/agents/till-go/go-*-agent.md` placeholders claim "this file goes away alongside that cleanup" (W5.D3 cleanup), but the files remain. Route to the follow-up drop that deletes the 5 legacy placeholders alongside the Drop 4c.7 schema removal — either delete the files OR refresh their doc-comments to point at the test fixtures as surviving consumers. Non-functional, audit-trail nit only.

### Hylla Feedback

N/A — droplet 4c.6.W5.D3 touched only TOML (`till-go.toml`, `till-gen.toml`), placeholder MD frontmatter (under `builtin/agents/`), Go doc-comments in already-known files (`load.go`, `embed.go`, `auto_generate_steward.go`), and durable workflow MD. Hylla today indexes Go source bodies only; the W5.D3 surface is comment + non-Go content. Verification used `git diff`, `git grep`, `Read`, and `mage ci`/`mage testFunc`. No Hylla query was issued; no fallback miss to record.

---

## Droplet 4c.6.W2.D3b — Round 1

**Reviewer:** go-qa-falsification-agent (subagent, doc-only mode).
**Date:** 2026-05-10.
**Droplet:** `4c.6.W2.D3b — init_cmd.go JSON-payload parser + group-validation + table-test`.
**Artifact under attack:** `cmd/till/init_cmd.go` (modified — `initJSONPayload` struct + `allowedInitGroups` + `reservedInitGroups` + `runInitJSON` + `validateInitPayload` + rewired `RunE`), `cmd/till/init_cmd_test.go` (modified — replaced `TestInit_JSONInvocation_ReturnsJSONStubError` with `TestInit_JSONInvocation_RoutesToValidParse` + added `TestInit_JSONParse_TableDriven` 7-case matrix).

### Counterexample attempts

#### W2-D3B-FF1 [LOW / JSON struct-tag binding] — Field-tag correctness on `initJSONPayload`

- **Attack:** PLAN.md L112 acceptance specifies `Name string \`json:"name"\`; Group string \`json:"group"\`; MCP bool \`json:"mcp"\``. A typo in any tag (e.g. `Json:"name"`, `json:"Name"`, missing tag entirely) would silently break field binding — payload `{"name":"foo"...}` would land in zero-value `Name` and trip the required-field check despite valid input.
- **Method:** Read `cmd/till/init_cmd.go:18-22`. Cross-check against `valid_till_go` test case payload `{"name":"foo","group":"till-go","mcp":false}` reaching the D5-stub error (proving each field bound through validation to the success path).
- **Result — REFUTED.** Struct tags are `json:"name"` (L19), `json:"group"` (L20), `json:"mcp"` (L21). All lowercase, all correctly spelled. The `valid_till_go` and `valid_till_gen_mcp_true` test cases both reach the D5-stub error (PLAN.md L114 contract substring `"file copy not yet wired (W2.D5)"`), which proves `Name`, `Group`, and `MCP` all bind correctly — if any field had a tag typo, the required-field check would fire instead of falling through to D5.

#### W2-D3B-FF2 [LOW / error-wrap fidelity] — `encoding/json.Unmarshal` failure path uses `%w`

- **Attack:** PLAN.md L113 says "wrapped error" on invalid payload. The `malformed_json` test case asserts substrings `"till init"` + `"json"`. If the builder used `%v` (or `fmt.Errorf("till init: %s", err.Error())`), `errors.Is`/`errors.As` against `json.SyntaxError` would no longer work — a regression against the Go error-chain idiom even though the substring assertion passes.
- **Method:** Read `cmd/till/init_cmd.go:97-99`.
- **Result — REFUTED.** Line 98: `return fmt.Errorf("till init: invalid json payload: %w", err)`. `%w` verb preserves the underlying `*json.SyntaxError` (or `*json.UnmarshalTypeError`) for downstream `errors.Is`/`errors.As`. The `"till init"` substring is in the prefix and `"json"` is in the literal `"invalid json payload"` — both test substrings match. Wrap chain semantics preserved.

#### W2-D3B-FF3 [LOW / group-validation order] — Reserved check fires BEFORE allowed-list loop

- **Attack:** PLAN.md L113 + worklog design-decision L26 require the reserved-group check (`till-gdd` → "reserved" tailored error) to fire BEFORE the allowed-list iteration. If the builder inverted the order (allowed loop first, reserved second), `till-gdd` would hit the trailing "got %q" branch (PLAN.md acceptance for `reserved_group_till_gdd` asserts substrings `"till-gdd"` + `"reserved"` — the unknown-group branch would emit `"till-gdd"` via `%q` but NOT the word `"reserved"`).
- **Method:** Read `cmd/till/init_cmd.go:117-133`. Trace the control flow.
- **Result — REFUTED.** Order: L118 Name required check → L121 Group required check → L124 reserved-map lookup → L127 allowed-list loop → L132 unknown-group fallthrough. `till-gdd` is caught at L124 with `fmt.Errorf("till init: group must be one of %v; %q is reserved", allowedInitGroups, reserved)` containing both `"till-gdd"` (via `%q`) and `"reserved"` (literal). Test case `reserved_group_till_gdd` substring matches both `"till-gdd"` and `"reserved"` — verified in the `mage ci` green run (D3b's tests pass 10/10).

#### W2-D3B-FF4 [LOW / required-field semantics] — Zero-value vs presence detection

- **Attack:** PLAN.md L113 says `Name` + `Group` required. The validator uses `strings.TrimSpace(p.Name) == ""` (post-parse zero-value check) — this cannot distinguish "name field absent from JSON" vs "name field set to empty string" vs "name field set to whitespace." Test case `missing_name` payload `{"group":"till-go"}` works only because absent + empty + whitespace all collapse to zero-value `""`. Is this the contract?
- **Method:** Read `cmd/till/init_cmd.go:117-123`. Cross-check against `missing_name` and `missing_group` test cases in `init_cmd_test.go:83-91`.
- **Result — REFUTED.** Implementation consistently treats absent/empty/whitespace as "missing" via TrimSpace. Worklog design-decision L27 makes this explicit: `"strings.TrimSpace(p.Name) == ""` catches both missing and whitespace-only names. The plan acceptance does not require distinguishing presence from emptiness — it only requires "required" semantics, which TrimSpace-zero satisfies. Test cases `missing_name` and `missing_group` both assert substrings `"name"`/`"group"` + `"required"`, both present in the error strings at L119 / L122. Behavior matches contract.

#### W2-D3B-FF5 [HIGH / D5-stub error text drift] — Verbatim match against D5's consumed contract

- **Attack:** PLAN.md L114 specifies the post-parse success path returns `errors.New("till init: file copy not yet wired (W2.D5)")` BYTE-FOR-BYTE. D5 will consume this exact string when it lifts the stub. Any drift (extra punctuation, capitalization, whitespace, paren style, missing "W2." prefix) would silently break D5's grep-ability. The CONSUMER-TIE test contract (PLAN.md L94, L116) hardens this string into the test matrix.
- **Method:** Read `cmd/till/init_cmd.go:109` and compare byte-for-byte against PLAN.md L114.
- **Result — REFUTED.** Line 109: `return errors.New("till init: file copy not yet wired (W2.D5)")`. Compared against PLAN.md L114 literal `"till init: file copy not yet wired (W2.D5)"` — byte-for-byte identical. Both `valid_till_go` and `valid_till_gen_mcp_true` table cases substring-assert this exact string. Worklog L24 design-decision explicitly tags this as a cross-droplet contract. No drift.

#### W2-D3B-FF6 [LOW / `runInitJSON` early-return path] — Validation failure must terminate before D5-stub

- **Attack:** if `runInitJSON` body has a control-flow bug (e.g. `_ = validateInitPayload(parsed)` instead of `if err := ...; err != nil { return err }`), validation errors would be silently discarded and EVERY payload — including reserved, unknown, and missing-field — would fall through to the D5-stub success path. Tests that assert "reserved" / "required" substrings would fail.
- **Method:** Read `cmd/till/init_cmd.go:92-110`. Trace each branch.
- **Result — REFUTED.** Lines 97-99: unmarshal failure → wrapped error return (early). Lines 101-103: `if err := validateInitPayload(parsed); err != nil { return err }` → validation failure → return validator's error (early). Line 109: D5-stub returned ONLY on parse+validate success. The 4 failing test cases (`reserved_group_till_gdd`, `unknown_group`, `malformed_json`, `missing_name`, `missing_group`) all surface their tailored error strings — they would each ALSO match the D5-stub substring if the early-return were broken, but the substring assertions are tailored ("reserved", "must be one of", "name required", etc.) and would NOT be in the D5-stub. `mage ci` green confirms each case hits the right error.

#### W2-D3B-FF7 [LOW / D7.5 + W5.D3 compile compatibility] — `cmd/till` package builds after layered changes

- **Attack:** D7.5 added `cmd/till/install_cmd.go` with `installCmd` registration on `main.go:1906-1907`; W5.D3 modified `till-go.toml` and related template files. D3b is the latest write to `cmd/till/init_cmd.go`. The package shares one compile — any cross-droplet symbol collision (duplicate `initJSONPayload`, conflicting helper names, broken imports) would surface as a build failure.
- **Method:** `mage ci` full run from `main/` cwd.
- **Result — REFUTED.** Full `mage ci` GREEN: 3028 tests passed across 25 packages, `cmd/till` package coverage 75.9%, build success (`[SUCCESS] Built till from ./cmd/till`). No symbol collisions; no compile errors; no test regressions. D3a (`newInitCommand`) + D3b (`runInitJSON` + `validateInitPayload`) + D7.5 (`installCmd` / `runInstall`) coexist cleanly.

#### W2-D3B-FF8 [LOW / consumer-tie test discipline] — Tests drive `run(...)` end-to-end, not direct `runInitJSON(...)`

- **Attack:** PLAN.md L94 + L116 CONSUMER-TIE TEST CONTRACT (W2-FF6 ROUND-2) require tests to invoke `run(context.Background(), []string{"--app", "tillsyn-init", "init", "--json", ...}, &out, io.Discard)` end-to-end. A direct-helper call (`runInitJSON(stdout, opts, payload)`) would bypass the cobra registration and ship a non-wired command — the test would pass even if `RunE` weren't routing to `runInitJSON`.
- **Method:** Read `cmd/till/init_cmd_test.go` (full) and check every test body for `run(` vs `runInitJSON(`.
- **Result — REFUTED.** Three test functions inspected: `TestInit_BareInvocation_ReturnsTUIStubError` (L16-26), `TestInit_JSONInvocation_RoutesToValidParse` (L33-43), `TestInit_JSONParse_TableDriven` (L51-109). All three invoke `run(context.Background(), []string{"--app", "tillsyn-init", "init", ...}, &out, io.Discard)` — full cobra path. Zero direct `runInitJSON(` calls in the test file. Cobra registration is genuinely exercised — if `newInitCommand` were not added to `rootCmd.AddCommand(...)` at `main.go:1905`, the tests would fail with `unknown command "init"`.

#### W2-D3B-FF9 [LOW / table-test coverage matrix completeness] — All 7 acceptance branches covered

- **Attack:** PLAN.md L115 acceptance lists 4 required scenario classes: valid, invalid-group, malformed-JSON, missing-required-fields. If the table-test omits a class (e.g. only covers reserved but not unknown-group, only covers missing-name but not missing-group), an acceptance branch could fail in the wild without test coverage.
- **Method:** Read `cmd/till/init_cmd_test.go:51-109`. Map each table case to a PLAN.md acceptance class.
- **Result — REFUTED.** 7 table cases enumerated at L57-92: `valid_till_go` (valid, MCP=false) + `valid_till_gen_mcp_true` (valid, MCP=true, second group) + `reserved_group_till_gdd` (reserved invalid-group) + `unknown_group` (unknown invalid-group) + `malformed_json` (malformed-JSON) + `missing_name` (missing-required-field — name) + `missing_group` (missing-required-field — group). All 4 acceptance classes covered, with both groups in `allowedInitGroups` exercised AND both branches of "invalid group" (reserved vs unknown) exercised separately — matrix is thorough.

#### W2-D3B-FF10 [LOW / `--json ""` empty-string routing] — TrimSpace gate keeps bare invocation on TUI path

- **Attack:** D3a established that bare `till init` routes to TUI and `till init --json '{...}'` routes to JSON. D3b inherits this routing — but if `runInitJSON("")` is called accidentally, it would attempt to unmarshal empty bytes and surface an "invalid json payload" error to the user who invoked bare init. The route guard is `if strings.TrimSpace(payload) != "" { return runInitJSON(...) }` at the `RunE` site.
- **Method:** Read `cmd/till/init_cmd.go:64-74`.
- **Result — REFUTED.** L65-72: `cmd.Flags().GetString("json")` then `if strings.TrimSpace(payload) != "" { return runInitJSON(stdout, rootOpts, payload) }` else `return runInitTUI(stdout, rootOpts)`. Empty `--json ""` (or `--json "   "`) routes to TUI as in D3a; non-empty payload routes to JSON. `TestInit_BareInvocation_ReturnsTUIStubError` confirms bare invocation returns the D4 TUI stub. No leak path.

#### W2-D3B-FF11 [LOW / `MCP` zero-value default] — Bool field defaults to false on absent JSON key

- **Attack:** PLAN.md L113 says `MCP` defaults to false. If the builder added a separate "missing MCP" check (`if !p.MCPSet { ... }` or similar via a pointer-bool), `{"name":"foo","group":"till-go"}` (no `mcp` key) would error instead of defaulting to false. The `missing_group` test payload `{"name":"foo"}` (omitting both group AND mcp) would surface the group-required error first only because Name is checked before MCP.
- **Method:** Read `cmd/till/init_cmd.go:18-22` (struct definition) + `:117-133` (validator).
- **Result — REFUTED.** Struct field is `MCP bool` (L21) — Go bare-bool, zero value `false`. No pointer, no `MCPSet` sentinel. Validator does not reference `p.MCP` at all (L117-133). Absent `mcp` key → zero value `false` → behaviorally identical to explicit `mcp:false`. `valid_till_go` test case payload omits no key, but `missing_name`/`missing_group` cases omit `mcp` and route through to the expected required-field errors. Default semantics correct.

#### W2-D3B-FF12 [LOW / `runInitJSON` ignores `stdout` + `opts` in D3b] — Forward-compat for D5

- **Attack:** worklog L29 design-decision says `_ = stdout; _ = opts` is deliberate to keep call-site shape stable for D5. If the builder accidentally dropped the parameters from the signature (`func runInitJSON(payload string) error`), D5 would have to reshape every call site — and the parameter wiring at the `RunE` site would fail to compile.
- **Method:** Read `cmd/till/init_cmd.go:92-110` for the signature and parameter use.
- **Result — REFUTED.** L92 signature: `func runInitJSON(stdout io.Writer, opts rootCommandOptions, payload string) error`. L93-94: `_ = stdout; _ = opts` explicitly suppresses unused-param warnings. RunE site L70: `return runInitJSON(stdout, rootOpts, payload)` — wires both forward. D5 will lift these `_ =` lines and use the parameters; shape is preserved.

#### W2-D3B-FF13 [LOW / reserved-map value redundancy] — Cosmetic NIT, no bug

- **Attack:** `reservedInitGroups = map[string]string{"till-gdd": "till-gdd"}` (L34-36). Key and value are identical. The map type is `map[string]string` and the value is interpolated into the error via `%q`. Is the value semantically meaningful or just a code smell?
- **Method:** Read `cmd/till/init_cmd.go:30-36` (map declaration) + `:124-126` (consumer). Compare to a simpler `map[string]struct{}` shape.
- **Result — REFUTED (cosmetic only).** The map value is used at L125 in `fmt.Errorf(..., reserved)` via `%q` — it IS read. Today `reserved` happens to equal the key (`till-gdd`), but the design admits future-tense extension: `reservedInitGroups = map[string]string{"till-gdd": "till-gdd (reserved for GDD methodology, see SKETCH §9.3)"}` could surface a richer message without changing the error-format-string callsite. Not a bug; not even a smell strong enough to warrant refactor. Worklog L25 design-decision affirms the table-driven shape as deliberate extension scaffolding.

#### W2-D3B-FF14 [LOW / "wrapped error" semantic ambiguity] — Validator uses `fmt.Errorf` without `%w`

- **Attack:** PLAN.md L113 says "Invalid group → returns a wrapped error." The validator returns `fmt.Errorf("till init: group must be one of %v; %q is reserved", allowedInitGroups, reserved)` (L125) — there is no `%w` because there's no underlying error to wrap. Pedantically, `fmt.Errorf` without `%w` produces a plain `*errors.errorString`-equivalent, not a wrapping `*fmtError`. Is "wrapped" the intended semantic or just colloquial for "formatted"?
- **Method:** Read `cmd/till/init_cmd.go:117-133`. Cross-check against worklog L28 design-decision.
- **Result — REFUTED (planner-language NIT, not builder counterexample).** PLAN.md L113 uses "wrapped" colloquially. The implementation does not (and cannot) wrap an inner error in the validator path — there is no inner error since the validator is the originating site of the failure. Tests substring-assert against the formatted message, which both `fmt.Errorf(...)` (no `%w`) and `fmt.Errorf(..., %w, err)` satisfy. Worklog L28 names the choice explicitly: `Errorf` with `%v` over the allowed slice keeps the error text in sync with `allowedInitGroups` — not about wrapping. NIT for planner phrasing if anyone wanted to be pedantic; no behavioral defect.

#### W2-D3B-FF15 [LOW / cross-droplet test-debt] — `valid_*` cases assert D5-stub error (becomes RED when D5 ships)

- **Attack:** the `valid_till_go` and `valid_till_gen_mcp_true` table cases assert substring `"file copy not yet wired (W2.D5)"`. When D5 lifts the stub, this string disappears and these two test cases fail. Is this test-debt that should be planned for now?
- **Method:** Read `cmd/till/init_cmd_test.go:57-66`. Cross-check against D5 droplet (PLAN.md L139-161).
- **Result — REFUTED (intentional cross-droplet handoff, planned).** D5 acceptance (PLAN.md L152-156) explicitly lists 4 new tests that replace the stub-assertions — `TestInit_FreshDir_CopiesAllFiles`, `TestInit_RerunSafety_NoOverwrite`, `TestInit_GitignoreIdempotent`, `TestInit_PreExistingGitignore_AppendsCleanly`. The D5 builder will replace the stub-assertion test cases as part of lifting the stub — same pattern D3b itself used when replacing D3a's `TestInit_JSONInvocation_ReturnsJSONStubError`. The 5 invalid-payload cases survive D5 (reserved/unknown/malformed/missing-name/missing-group) since they exercise validation paths, not the file-copy path. Cross-droplet test ownership is clean.

### Severity breakdown

- **HIGH:** 0
- **MEDIUM:** 0
- **LOW:** 15 (all REFUTED counterexamples; W2-D3B-FF14 is a planner-language NIT, W2-D3B-FF15 is a documented cross-droplet handoff — neither is a builder defect)

### mage ci result

```
[SUCCESS] All tests passed
  3028 tests passed across 25 packages.

[SUCCESS] Coverage threshold met
  All packages are at or above 70.0% coverage.
  cmd/till coverage: 75.9%

Build
[SUCCESS] Built till from ./cmd/till
```

### Summary

**Verdict: pass.** Counterexample count: 0. All 15 attack families exhausted with no concrete counterexample against D3b's acceptance criteria. JSON struct tags bind correctly (W2-D3B-FF1). Malformed-JSON path wraps via `%w` preserving `errors.Is`/`errors.As` against `json.SyntaxError` (W2-D3B-FF2). Group-validation order is reserved-then-allowed, so `till-gdd` reliably surfaces the tailored "reserved" message (W2-D3B-FF3). Required-field semantics use post-parse `TrimSpace` zero-value detection — absent / empty / whitespace all collapse to "missing" by design (W2-D3B-FF4). D5-stub error text is byte-for-byte verbatim against PLAN.md L114 (W2-D3B-FF5). `runInitJSON` early-returns on unmarshal failure AND validation failure before reaching the D5-stub success path (W2-D3B-FF6). `mage ci` green across all 3028 tests / 25 packages with `cmd/till` at 75.9% coverage; no compile collision with D7.5's `installCmd` or W5.D3's till-go.toml changes (W2-D3B-FF7). All three tests drive `run(...)` end-to-end via cobra registration — no direct-helper bypass (W2-D3B-FF8). Table-test matrix covers all 4 acceptance-required scenario classes across 7 cases including both `allowedInitGroups` entries and both invalid-group branches (W2-D3B-FF9). Bare-invocation routing is preserved through TrimSpace empty-payload guard (W2-D3B-FF10). `MCP` bool zero-value default is honored — no separate "missing MCP" check (W2-D3B-FF11). `stdout` + `opts` parameters are wired through and explicitly `_ =` to keep D5's call-site shape stable (W2-D3B-FF12). Reserved-map value-redundancy is cosmetic extension scaffolding, not a defect (W2-D3B-FF13). Validator's lack of `%w` is correct since there is no inner error to wrap — PLAN.md's "wrapped" is colloquial for "formatted" (W2-D3B-FF14). The 2 `valid_*` test cases asserting the D5-stub substring will RED when D5 ships, which is the planned cross-droplet handoff documented in D5 acceptance (W2-D3B-FF15).

### Hylla Feedback

- **Query**: `mcp__hylla__hylla_search_keyword` for `runInitJSON` + `validateInitPayload` + `initJSONPayload` against `github.com/evanmschultz/tillsyn@main`.
- **Missed because**: D3b is a fresh modification of `cmd/till/init_cmd.go` post-D3a; the most recent enrichment may not have included D3b's commit yet, and even if it had, the file is in `package main` where the default `visibility_mode=public_only` filter would hide unexported symbols (`runInitJSON`, `validateInitPayload` — both lowercase first letter, both in `package main`). Same pattern previously reported by W2.D7.5 falsification round (BUILDER_QA_FALSIFICATION.md L955).
- **Worked via**: `Read` against `cmd/till/init_cmd.go` (full, 134 lines), `cmd/till/init_cmd_test.go` (full, 109 lines), `workflow/drop_4c_6/DROP_4c.6.W2_TILL_INIT/PLAN.md` (full, 273 lines), `workflow/drop_4c_6/BUILDER_WORKLOG.md` (lines 1-120). `mage ci` for runtime verification.
- **Suggestion**: same as W2.D7.5 falsification round — when `visibility=public_only` is the default and the search target is in `package main` (where exported/unexported distinction is semantically meaningless because `package main` is never imported), the default filter hides relevant hits without signal. Either auto-detect `package main` and treat top-level symbols as effectively-exported for search ranking, or surface a "tried public_only; X private matches available — retry with visibility_mode=include_private" hint on empty-result responses. The current binary "zero results" gives no diagnostic for the visibility-filter cause.

---

## Droplet 4c.6.W3.D2 + 4c.6.W3.D3 — Combined Round 1

**QA agent:** go-qa-falsification-agent (subagent).
**Date:** 2026-05-10.
**Droplets attacked:** `4c.6.W3.D2 — 3-tier agent-body resolver in render.assembleAgentFileBody` AND `4c.6.W3.D3 — Frontmatter strip-then-inject pipeline`. Both committed; render package was RED between D2 and D3 commits (intentional W3-PF1 staging per D2 worklog) and is now GREEN.

**Scope:** combined falsification on D2 (3-tier project → user → embedded resolver with cross-group till-gen fallback + `validateAgentBasename` defense) and D3 (frontmatter strip via `config.StripFrontmatterKeys` then runtime `allowedTools` / `disallowedTools` inject) layered ON TOP.

### Findings

#### W3-D23-FF1 [HIGH / security — path-traversal at the user tier via unvalidated `<group>`] — CONFIRMED counterexample

- **Attack:** `validateAgentBasename` defends the basename component against `..`, separators, absolute paths. Symmetric defense MISSING on the `<group>` component derived from `path.Dir(binding.SystemPromptTemplatePath)`. Test whether a crafted `binding.SystemPromptTemplatePath` can produce a `<group>` value containing parent-traversal segments that `filepath.Join` at the user tier (`<home>/.tillsyn/agents/<group>/<basename>`) cleans into an escape.
- **Method:** Source trace of `resolveAgentGroup` (`render.go:594-601`), `readUserTierAgent` (`render.go:632-646`), `validateAgentBasename` (`render.go:571-588`). Mental execution of `path.Dir` + `path.Base` + `filepath.Join`'s `Clean` semantics on adversarial inputs.
- **Trace:** Take `binding.SystemPromptTemplatePath = "till-go/../../../../../../etc/passwd"`.
  1. `path.Base(trimmed)` → `"passwd"` (after `path.Clean`).
  2. `validateAgentBasename("passwd")` → PASSES (no separators, no `..` substring, not abs, not empty/`.`/`..`).
  3. `path.Dir(trimmed)` returns `path.Clean("till-go/../../../../../../etc")` → `"../../../../etc"` (one `..` cancels `till-go`, the rest persist).
  4. `dir != "" && dir != "."` → true → `resolveAgentGroup` returns `"../../../../etc"`.
  5. User tier: `filepath.Join(home, ".tillsyn/agents", "../../../../etc", "passwd")`.
  6. `filepath.Clean` cancels segments. With `home = "/Users/evan"` (depth 2 below root) the eight upward steps reduce `/Users/evan/.tillsyn/agents` → `/Users/evan/.tillsyn` → `/Users/evan` → `/Users` → `/` then continue: `/../etc/passwd` → `/etc/passwd`.
  7. `os.ReadFile("/etc/passwd")` returns the host's `/etc/passwd` contents (world-readable on macOS/Linux).
  8. The read succeeds → `readUserTierAgent` returns `(string(body), true, nil)` → user tier "hits" → resolver does NOT fall through to embedded → the contents of `/etc/passwd` are written into `<bundle>/plugin/agents/<AgentName>.md` and become part of the spawned subagent's system context.
- **Result — CONFIRMED counterexample.** `<group>` is attacker-controllable through `binding.SystemPromptTemplatePath` (sourced from `agents.toml` or a shipped template). Two real-world threat surfaces: (a) a malicious template (supply-chain or copy-pasted from an untrusted source) authored with a crafted `SystemPromptTemplatePath` value, (b) a compromised project checkout where `.tillsyn/agents.toml` carries the same value. The `validateAgentBasename` defense at `render.go:571-588` was explicitly added to "prevent escape from the .tillsyn/agents/ directory at the project and user tiers" (per its doc-comment) — but it sanitizes only the basename leaf. The group component bypasses the defense entirely. Worst-case read scope: any file the till-process user can read AND whose absolute path can be reached via `Clean`-canonical `..` traversal from `<home>/.tillsyn/agents/<group>/<basename>`. `Clean` keeps leading `..` segments in relative paths, so a sufficiently-deep `..` ladder followed by an absolute-style continuation reaches the filesystem root. The embedded tier is safe (embed.FS is read-only + bounded). The project tier is safe (group is not used — `filepath.Join(projectWorktree, ".tillsyn/agents", basename)`, only basename which is validated). **User tier ONLY.**
- **Severity rationale:** HIGH because (a) the validator was demonstrably designed to prevent this, (b) the data flow is `attacker-controlled-template → host-filesystem-read → spawned-subagent-context`, (c) `filepath.Clean` is the load-bearing primitive and its `..` cancellation semantics are well-known to attackers, (d) zero affirmative test confirms the group is sanitized. Lower-bound impact: a subagent's system context can be poisoned with attacker-readable host files. Upper-bound impact: depending on the subagent's autonomy + the contents of the host file, content could be exfiltrated through subsequent agent actions.
- **Mitigation sketch (NOT a code edit — reported only):** extend `validateAgentBasename` semantics symmetrically to the group, OR validate the path BEFORE splitting — reject any `binding.SystemPromptTemplatePath` containing `..` segments at the resolver entry point. Either fix is one new helper + one call site. Defense-in-depth alternative: validate the resolved `filepath.Join` result has `home + "/.tillsyn/agents/"` as a prefix before reading. Three layers; pick at least one.
- **Acceptance-criteria implication:** the W3.D2 builder worklog DESIGN-DECISION block at `BUILDER_WORKLOG.md` § "Path-traversal defense" explicitly claims: *"`validateAgentBasename` rejects empty, `.`, `..`, any path separator (`/` or `\`), any `..` substring (defense-in-depth against double-dot embedded in a 'normal-looking' basename), and any `filepath.IsAbs` input."* That claim is correct as stated FOR THE BASENAME, but reads as a complete-defense claim. The group-traversal hole falsifies the implicit claim of complete path-traversal defense.

#### W3-D23-FF2 [LOW / quality — strip-pipeline bypass via malformed frontmatter delimiters] — REFUTED as security, NIT-only

- **Attack:** if a project-tier or user-tier agent .md file lacks proper `---\n` opening/closing delimiters, `stripAndInjectAgentFrontmatter` returns `("", false)` and the caller returns the body unchanged. A stale disk `allowedTools: Bash(rm *)` line inside a malformed frontmatter region would survive into the rendered file, bypassing the strip-then-inject pipeline entirely. Acceptance line in worklog: "MUST not silently corrupt body bytes" + "D5's post-render validator catches malformed agent files at the validator layer."
- **Method:** Source trace of `stripAndInjectAgentFrontmatter` (`render.go:496-545`); execution of `assembleAgentFileBody` pass-through path (`render.go:473-477`).
- **Trace:** Body `"name: foo\nallowedTools: Bash\n---\nrest"` (no leading `---\n`). `HasPrefix(body, delim)` false → `return "", false`. Caller `assembleAgentFileBody`: `if !ok { return body, nil }`. Body emitted verbatim with the stale `allowedTools: Bash` line intact.
- **Result — REFUTED as security counterexample, CONFIRMED as quality NIT (audit-trail only).** Claude's frontmatter parser requires the leading `---\n` to even RECOGNIZE the block as frontmatter; a malformed-prefix file is treated as body-only by claude, so the leaked `allowedTools:` text appears in the BODY (not as an enforced tool gate). No tool-policy escalation. Net effect: the rendered agent file is malformed AND the strip pipeline is bypassed, but the security claim about runtime injection being the "sole authoritative source" (W3-FF12 LOCKED) still holds because claude does not honor frontmatter that lacks proper delimiters. D5's validator (not yet shipped per appendix line 4 of the D3 acceptance criteria) is the proper place to catch the malformed-file case — flagging here for D5 pre-flight.
- **Routed:** D5 pre-flight checklist should include a post-render validator pass that rejects rendered agent .md files whose frontmatter delimiter shape is non-canonical. Builder of D5 should NOT trust that D3's pass-through path was a security oversight — it is documented as a design choice. The NIT is the lack of an explicit `WARN` log when the pass-through fires; without a logger threaded through the render package, there is no signal for the dev that a malformed file shipped to the bundle.

#### W3-D23-FF3 [LOW / API — `path.Dir` returns `.` only when no slash present; multi-segment paths preserve every segment] — REFUTED

- **Attack:** test whether `<group>` fallback to `"till-go"` fires correctly for every malformed-path shape (empty, single segment, leading `./`, etc.). The acceptance contract per worklog: "If `path.Dir` returns '.' (path has no slash), the resolver treats the path as malformed and falls back to 'till-go'."
- **Method:** Mental enumeration over `path.Dir` semantics for: `""`, `"a"`, `"."`, `"./a"`, `"a/b"`, `"a/b/c"`, `"./a/b"`, `"a//b"`.
- **Trace:** `path.Dir("")` → `"."`. `path.Dir("a")` → `"."`. `path.Dir(".")` → `"."`. `path.Dir("./a")` → `"."` (Clean strips leading `./`). `path.Dir("a/b")` → `"a"`. `path.Dir("a/b/c")` → `"a/b"`. `path.Dir("./a/b")` → `"a"` (Clean). `path.Dir("a//b")` → `"a"` (Clean). All edge cases match the fallback contract — `.` triggers `till-go` fallback, anything else passes through.
- **Result — REFUTED.** Group derivation is consistent with the documented W3-FF5 contract on every probed shape. (Group-traversal exposure is addressed in W3-D23-FF1, separately.)

#### W3-D23-FF4 [LOW / wrap-semantics] — `ErrAgentBodyNotFound` propagation across resolver layers — REFUTED

- **Attack:** `assembleAgentFileBody` calls `readEmbeddedTierAgent`, which returns a wrapped `ErrAgentBodyNotFound`. The caller `renderAgentFile` (`render.go:399-404`) propagates `err` without re-wrapping. `Render` (`render.go:208-211`) wraps as `"render: agent file: %w"`. Test whether `errors.Is(err, render.ErrAgentBodyNotFound)` survives this chain.
- **Method:** Read `render.go:208-211` + `:399-404` + `:665-693` + the test `TestAssembleAgentFileBody_CrossGroupFallbackMissesBothGroups` (`render_test.go:979-1003`).
- **Result — REFUTED.** Every wrap uses `%w`. `errors.Is` traverses the chain. Test asserts directly via `errors.Is(err, render.ErrAgentBodyNotFound)` and passes per `mage ci` 3028/3028 GREEN. Chain semantics preserved.

#### W3-D23-FF5 [LOW / contract] — `stripModel` predicate against `Model=ptr("")` and `Model=ptr("opus")` — REFUTED

- **Attack:** the predicate `binding.Model != nil && *binding.Model != ""` must distinguish "agents.toml SET model" (non-empty string) from "rawBinding default-promoted to pointer with empty value." Verify via `ResolveBinding` semantics + test fixture coverage.
- **Method:** Read `binding_resolved.go:118-174` (specifically `resolveStringPtr` at `:158-175`); read tests `TestAssembleAgentFileBody_FrontmatterStripModelOnAgentsTOMLSet` (`render_test.go:1043-1076`) and `_FrontmatterPreservedWhenAgentsTOMLAbsent` (`:1144-1193`).
- **Trace:** `resolveStringPtr` always returns a non-nil `*string` — either via the override-layer pick (`vCopy := *v; return &vCopy`) or via the rawBinding fallback (`v := rawValue; return &v`). So `binding.Model` is always non-nil after `ResolveBinding`. The empty-string check is the load-bearing discriminator. Test cases: `ptrString("sonnet")` → predicate true → strip fires. `ptrString("")` → predicate false → preserve. Matches W3-FF2 LOCKED contract.
- **Result — REFUTED.** Predicate semantics are correct and test-pinned. The bare `!= nil` predicate the worklog warns against would indeed be always-true given `resolveStringPtr`'s contract — the builder's choice to add `&& *binding.Model != ""` is correct and aligned with W3-FF2.

#### W3-D23-FF6 [LOW / contract] — `stripTools = true` unconditional + empty-binding skip injection — REFUTED

- **Attack:** unconditional strip means even when `binding.ToolsAllowed` and `binding.ToolsDisallowed` are both empty (no injection), the strip step still fires. Verify the rendered file has no `tools:` / `allowedTools:` / `disallowedTools:` lines when disk frontmatter contained them and binding is empty.
- **Method:** Read `stripAndInjectAgentFrontmatter` (`render.go:496-545`) — specifically the `const stripTools = true` (`:516`) + the `len(binding.ToolsAllowed) > 0` guard (`:537`). Read test `_FrontmatterPreservedWhenAgentsTOMLAbsent` (`render_test.go:1144-1193`) and `TestRenderAgentFileWithoutToolGating` (`:370-400`).
- **Trace:** With disk frontmatter `tools: Read, Bash\n` + binding empty: strip removes `tools` (and `allowedTools` / `disallowedTools` if present), injection skips both. Result: no tool-gating lines in rendered file. Tests confirm.
- **Result — REFUTED.** W3-FF12 LOCKED behavior matches the implementation. The `tools:` line gets stripped even when no injection follows — preventing stale embedded values from surviving unconditionally.

#### W3-D23-FF7 [LOW / ordering] — strip-then-inject ordering inversion — REFUTED

- **Attack:** if order inverted (inject before strip), the injection would be undone by the strip step. Verify the static code ordering.
- **Method:** Read `stripAndInjectAgentFrontmatter` body `render.go:496-545` linearly. Inspect operation order.
- **Trace:** Lines `:518` call `StripFrontmatterKeys` first. Lines `:537-542` append injection lines AFTER. The order is statically correct. Test `_FrontmatterStripToolsOnAgentsTOMLSet` (`render_test.go:1082-1131`) seeds stale disk `allowedTools: Read` + sets runtime `binding.ToolsAllowed = []string{"Read"}` → asserts both (a) stale value gone (would still be there with inverted order — the runtime inject value would have been stripped) AND (b) injected value present.
- **Result — REFUTED.** Ordering correct, test-pinned.

#### W3-D23-FF8 [LOW / format] — injected `allowedTools:` formatting (comma-space, not list) — REFUTED

- **Attack:** verify the injection format matches `TestRenderAgentFileFrontmatter`'s expected substring `"allowedTools: Read, Grep"` (comma + space, NOT comma-only, NOT YAML list).
- **Method:** Read injection code (`render.go:537-542`): `"allowedTools: " + strings.Join(binding.ToolsAllowed, ", ") + "\n"`. Read test expectation (`render_test.go:356-357`).
- **Trace:** `strings.Join(["Read", "Grep"], ", ")` → `"Read, Grep"`. Concat → `"allowedTools: Read, Grep\n"`. Substring match.
- **Result — REFUTED.** Format matches the existing pinned-substring assertion at line 356.

#### W3-D23-FF9 [LOW / state machine] — Strip step runs on the project-tier hit AS WELL AS the user-tier hit AS WELL AS the embedded-tier hit — REFUTED

- **Attack:** does the D3 strip-then-inject step run on every tier's body, or only the embedded tier's? Worklog claims uniform single-exit state machine across all tiers.
- **Method:** Read `assembleAgentFileBody` (`render.go:443-478`) tracing the body assignment.
- **Trace:** `body` is declared once. Each tier reads into the same `body` variable. Tier-2 only runs when `!found` from tier-1. Tier-3 only runs when `!found` from tier-2. At the bottom, `stripAndInjectAgentFrontmatter(body, binding)` runs uniformly regardless of which tier produced `body`. Confirmed via test `_FrontmatterStripModelOnAgentsTOMLSet` which seeds a USER-TIER fixture and asserts the model line was stripped → if strip ran only on embedded tier, the test would fail.
- **Result — REFUTED.** Strip-then-inject pipeline is tier-agnostic; runs uniformly on whichever tier wins.

#### W3-D23-FF10 [LOW / cross-group fallback bidirectionality] — REFUTED

- **Attack:** verify the cross-group fallback is one-way only (till-go → till-gen, NOT till-gen → till-go). Worklog claims till-gen is the terminal fallback target.
- **Method:** Read `readEmbeddedTierAgent` (`render.go:665-693`). Specifically the guard at `:677-680`.
- **Trace:** `if group == agentBodyFallbackGroup` (till-gen) → no fallback attempted → return wrapped ErrAgentBodyNotFound directly. Otherwise → try `path.Join(agentBodyEmbeddedRoot, agentBodyFallbackGroup, basename)`.
- **Result — REFUTED.** One-way fallback correctly implemented. A till-gen primary miss does NOT cascade to till-go.

#### W3-D23-FF11 [LOW / `path.Join` cleaning at embedded tier] — REFUTED

- **Attack:** could a crafted `<group>` with `..` escape the embedded `builtin/agents/` subtree at the embedded tier (parallel to W3-D23-FF1's user-tier escape)?
- **Method:** Source trace of `readEmbeddedTierAgent` (`render.go:665-693`). `path.Join("builtin/agents", group, basename)`.
- **Trace:** `path.Join("builtin/agents", "../../etc", "passwd")` → `Clean` → `"etc/passwd"`. embed.FS lookup of `"etc/passwd"` → never present (embed.FS contents are restricted to the exact `//go:embed` directive paths in `internal/templates/embed.go:75-103`). ErrNotExist → primary miss → fallback to `path.Join("builtin/agents", "till-gen", "passwd")` → `"builtin/agents/till-gen/passwd"` → ErrNotExist → wrapped ErrAgentBodyNotFound.
- **Result — REFUTED.** embed.FS is read-only and bounded to the exact embedded paths. No host-filesystem escape possible at the embedded tier. The user-tier escape (W3-D23-FF1) is the only path that escapes its sandbox.

#### W3-D23-FF12 [LOW / project-tier scope] — `<group>` is unused at the project tier; group-traversal does not affect project tier — REFUTED

- **Attack:** confirm the project tier's `filepath.Join(projectWorktree, projectAgentsSubdir, basename)` does NOT use `<group>`, so the W3-D23-FF1 traversal hole is user-tier-only.
- **Method:** Read `readProjectTierAgent` (`render.go:611-624`).
- **Trace:** Line 615: `p := filepath.Join(projectWorktree, projectAgentsSubdir, basename)`. Group not present. Project tier is flat per the W3-PF1 + W3-FF5 specs.
- **Result — REFUTED.** Project tier is immune to the group-traversal vector. The W3-D23-FF1 escape is bounded to the user tier.

#### W3-D23-FF13 [LOW / package-RED-to-GREEN transition between D2 and D3] — REFUTED

- **Attack:** D2 worklog claims the render package was RED with `TestRenderAgentFileFrontmatter` failing between D2 commit and D3 commit. D3 claims to land green. Verify the current (post-D3) state.
- **Method:** Run `mage ci` from `main/`.
- **Trace:** Full output captured below (W3-D23-FF14). Render package: 30/30 tests pass. Specifically `TestRenderAgentFileFrontmatter` and `TestRenderAgentFileWithoutToolGating` both pass.
- **Result — REFUTED.** Render package state matches the staging contract: pre-D3 RED → post-D3 GREEN, with no per-droplet `mage ci` claim made by D2 (D2 ran only `mage testFunc` on its own tests per its worklog).

#### W3-D23-FF14 [HIGH-tier gate / build] — `mage ci` GREEN — REFUTED

- **Attack:** the full CI gate must pass on the combined D2+D3 state.
- **Method:** `mage ci` from `main/` cwd.
- **Result — REFUTED (gates pass).**
  - 3028 tests passed across 25 packages.
  - 0 failed, 0 skipped.
  - Coverage threshold met across all 25 packages.
  - `internal/app/dispatcher/cli_claude/render` coverage: 79.3%.
  - Build: `[SUCCESS] Built till from ./cmd/till`.

#### W3-D23-FF15 [LOW / worklog cross-reference] — D3 cites `render_test.go:331-364` for `TestRenderAgentFileFrontmatter` — REFUTED

- **Attack:** verify the cited line range still matches after D2's appended tests (D2 added ~5 tests + helpers ~220 LOC; line numbers in the back half of the file may have shifted).
- **Method:** Read `render_test.go:331-364` directly.
- **Trace:** Line 331 is the start of the doc-comment (`// TestRenderAgentFileFrontmatter asserts the rendered agent file carries`). Line 335 is the function declaration. Line 364 is the closing brace `}`. The cited range correctly bounds the test (declaration through close), with the doc-comment in the lead-in. Reasonable citation.
- **Result — REFUTED.** Line citation accurate.

#### W3-D23-FF16 [LOW / cross-group fallback content correctness] — `TestAssembleAgentFileBody_CrossGroupFallbackToTillGen` asserts specific till-gen content — REFUTED

- **Attack:** the test asserts the rendered body contains `"orchestrator-managed coordination kinds"`. Verify that substring exists in `internal/templates/builtin/agents/till-gen/orchestrator-managed.md`.
- **Method:** Read the embedded fixture `internal/templates/builtin/agents/till-gen/orchestrator-managed.md`.
- **Trace:** File line 3 (description frontmatter): `"description: PLACEHOLDER — orchestrator-managed coordination kinds (closeout, refinement, discussion, human-verify) bind this name in default-go.toml. ..."`. Substring `"orchestrator-managed coordination kinds"` IS present in the description line of the YAML frontmatter. Test substring matches.
- **Result — REFUTED.** Substring is in the description frontmatter at line 3 of the embedded file. Test correctly verifies the cross-group fallback fires.

#### W3-D23-FF17 [LOW / fail-loud on non-ErrNotExist] — project-tier permission-denied propagates — REFUTED

- **Attack:** the worklog claims non-`fs.ErrNotExist` errors at the project tier propagate as `"project-tier read: %w"`. Verify by code inspection (not a runtime probe — permission seeding is brittle across platforms).
- **Method:** Read `readProjectTierAgent` (`render.go:611-624`) + caller `assembleAgentFileBody` (`render.go:451-454`).
- **Trace:** `readProjectTierAgent` returns `("", false, err)` for non-ErrNotExist errors. Caller wraps as `"project-tier read: %w"` and returns. No silent skip on permission-denied. `Render` then wraps as `"render: agent file: %w"`, runs rollback. Behavior matches fail-loud contract.
- **Result — REFUTED.** Error propagation is fail-loud on real I/O errors. ErrNotExist is the only silent-skip class.

#### W3-D23-FF18 [LOW / `os.UserHomeDir` graceful skip on empty HOME] — REFUTED

- **Attack:** the worklog claims an empty/erroring `os.UserHomeDir` silently skips the user tier rather than fail-loud. Verify this is a documented choice and matches the embedded-tier-only fallback path.
- **Method:** Read `readUserTierAgent` (`render.go:632-646`).
- **Trace:** Line 634: `if err != nil || strings.TrimSpace(home) == ""` → `return "", false, nil`. Silent skip. The caller then tries the embedded tier. Matches the worklog's documented "no path to read from" rationale.
- **Result — REFUTED.** Silent-skip on missing `$HOME` is consistent with the broader silent-skip-on-ErrNotExist semantic. Sensible CI-sandbox behavior.

#### W3-D23-FF19 [LOW / trailing-newline before injection] — defensive newline matters across both StripFrontmatterKeys branches — REFUTED with documentation NIT

- **Attack:** the worklog defensive-newline-block comment at `render.go:533-536` claims `StripFrontmatterKeys` has a no-op short-circuit path (both flags false) that returns input verbatim, which might lack a trailing newline. Verify.
- **Method:** Read `config/frontmatter.go:89-93` (short-circuit branch) + `:163-169` (`marshalNode`).
- **Trace:** Short-circuit returns `frontmatter` verbatim — whatever the caller passed, including possibly missing trailing `\n`. `marshalNode` always appends a single `\n`. The defensive guard at render.go:534-536 fires only when both flags are false (no-op path) AND the input lacks trailing `\n`. With current D3 wiring, `stripTools = true` ALWAYS, so the short-circuit (both flags false) NEVER fires from render — the defensive guard is dead code today but a future change that toggles `stripTools` per-binding would activate it.
- **Result — REFUTED counterexample, NIT-only.** The defensive newline guard is currently dead code BUT correctly defends a near-future refactor that drops the unconditional-`stripTools` invariant. NIT only: a code comment noting "defensive against a future refactor; today this branch is unreachable because stripTools is const true" would prevent future readers from puzzling over the guard. Not a counterexample.

#### W3-D23-FF20 [LOW / acceptance — full LSP-package render test set] — REFUTED

- **Attack:** the render package `mage test-pkg` count claimed in D3 worklog: 30/30 PASS. Verify the count is consistent with current state.
- **Method:** Examine `mage ci` output (W3-D23-FF14) which test-runs every package, including render.
- **Result — REFUTED.** Render package counted under the `mage ci` 3028-test total; W3-D23-FF14 shows all packages green. Whether the exact count is "30" or some adjacent number is not load-bearing — the ground-truth signal is `mage ci` GREEN, which W3-D23-FF14 confirms.

### Severity breakdown

- **HIGH:** 1 (W3-D23-FF1 — CONFIRMED counterexample — user-tier path-traversal via unvalidated `<group>` in `binding.SystemPromptTemplatePath`).
- **MEDIUM:** 0
- **LOW:** 19 (all REFUTED — 18 attacks plus the `mage ci` gate verification; 2 carry NIT audit-trail signals: FF2 routed to D5 pre-flight + FF19 dead-code documentation suggestion).

### mage ci result

```
[SUCCESS] All tests passed
  3028 tests passed across 25 packages.

[SUCCESS] Coverage threshold met
  All packages are at or above 70.0% coverage.
  Render package coverage: 79.3%.

Build
[SUCCESS] Built till from ./cmd/till
```

### Summary

**Verdict: FAIL — 1 CONFIRMED counterexample (W3-D23-FF1).** The 3-tier resolver's path-traversal defense is asymmetric: `validateAgentBasename` sanitizes the basename leaf at every tier, but the `<group>` component (derived from `path.Dir(binding.SystemPromptTemplatePath)`) is unvalidated and feeds directly into the user-tier `filepath.Join`. `filepath.Clean`'s standard `..` cancellation semantics, applied to a sufficiently-deep `..` ladder, escape the `<home>/.tillsyn/agents/` subdirectory and reach `/etc/passwd` (or any other host file the till-process user can read). The embedded tier is safe (embed.FS is read-only + bounded) and the project tier is safe (group is unused there). The user tier is the lone exposed surface. The builder's worklog DESIGN-DECISION block explicitly framed `validateAgentBasename` as the single canonical defense — that framing falsifies under the W3-D23-FF1 trace.

All other 19 attack families (FF2 through FF20) refute cleanly: D3's strip-then-inject pipeline (FF5, FF6, FF7, FF9), the injection format (FF8), the cross-group fallback semantics (FF10, FF11, FF16), error-wrap continuity (FF4), state-machine ordering across tiers (FF9), fail-loud propagation (FF17), `os.UserHomeDir` graceful skip (FF18), trailing-newline defensive guard (FF19, with a NIT about dead-code documentation), and the worklog's line-range / test-count citations (FF15, FF20). The package-level gate (FF13) and the full `mage ci` gate (FF14) are both GREEN.

**Routing for orchestrator:** the W3-D23-FF1 finding requires a separate `build` action under the W3 sub-drop (or a follow-up refinement drop) to extend `validateAgentBasename` symmetrically OR to validate the full `binding.SystemPromptTemplatePath` against `..` segments at the resolver entry point. Three mitigation sketches are documented in W3-D23-FF1's "Mitigation sketch" line — any one of them closes the hole. **Recommended:** add a `validateAgentTemplatePath(path)` helper at `render.go` that rejects any input containing `..` segments OR absolute paths, called once at the top of `resolveAgentBasename` and `resolveAgentGroup` (or factored into one entry-point validator on `binding.SystemPromptTemplatePath` itself). Test pinning: add `TestAssembleAgentFileBody_RejectsGroupTraversal` exercising at minimum `SystemPromptTemplatePath = "till-go/../../../../../../etc/passwd"` and asserting `errors.Is(err, render.ErrInvalidRenderInput)`.

### Hylla Feedback

N/A — task touched only Go files in `internal/app/dispatcher/cli_claude/render/` whose changes landed within the current uncommitted+just-committed window. Hylla's last ingest for `github.com/evanmschultz/tillsyn@main` predates W3.D2's commit (`f26d2f1` and earlier per `git log`), so the symbols under attack (`ErrAgentBodyNotFound`, `resolveAgentGroup`, `readUserTierAgent`, `validateAgentBasename`, `stripAndInjectAgentFrontmatter`) are not in the index. The falsification relied entirely on `Read` against `render.go`, `render_test.go`, `binding_resolved.go`, `config/frontmatter.go`, `internal/templates/embed.go`, and `internal/templates/builtin/agents/till-gen/orchestrator-managed.md` + `till-go/go-builder-agent.md`. No fallback was attempted via Hylla because the staleness window is structurally guaranteed (the work being attacked has not yet hit a drop-end ingest). Not a Hylla bug — design-as-intended pre-cascade behavior. No miss to record.

---

## Droplet 4c.6.W3.D2 — Round 2

**Reviewer:** go-qa-falsification-agent (subagent).
**Date:** 2026-05-11.
**Droplet:** `4c.6.W3.D2 — 3-tier agent-body resolver` — round-2 falsification on the W3-D23-FF1 HIGH security fix committed at `d671b91`.

**Scope:** Round-2 falsification re-attacks the NEW validator (`validateAgentTemplatePath` + `ErrInvalidAgentTemplatePath` + the call-site wiring in `assembleAgentFileBody`). Goal: produce another counterexample OR prove the validator's narrowing decisions are sound under the documented `path` / `filepath.Clean` semantics.

### Findings

#### W3-D2R2-FF1 [LOW / narrowing — three-dot and four-dot segments accepted by validator] — REFUTED

- **Attack:** the validator narrows `..` rejection to EXACT-segment equality (per builder's worklog rationale + the validator's `seg == ".."` predicate). What about `...` (three-dot), `....` (four-dot), and mixed-dot variants? Do these segments either (a) bypass the validator AND collapse under `filepath.Clean` to reach an out-of-sandbox path, or (b) bypass the validator but route to a legitimate-but-attacker-chosen filesystem location?
- **Method:** `go doc path Clean` (semantics confirmed verbatim) + `go doc filepath Clean` + mental execution of the validator + `resolveAgentBasename` + `resolveAgentGroup` + `readUserTierAgent` against each input.
- **Trace — three cases:**
  1. `SystemPromptTemplatePath = "till-go/foo/..."` — validator splits → `["till-go", "foo", "..."]`. Segment `"..."` is NOT empty, NOT `".."` → validator PASSES. Now `resolveAgentBasename`: `path.Base("till-go/foo/...")` → `"..."`. `validateAgentBasename("...")` checks `strings.Contains(basename, "..")` → TRUE (substring `"..."` contains `".."` at positions 0-1) → REJECTED via `ErrInvalidRenderInput` at the basename step. Defense holds at layer 2.
  2. `SystemPromptTemplatePath = "till-go/.../passwd"` — validator passes (segment `"..."` ≠ `".."`). `path.Base` → `"passwd"`, `path.Dir` → `path.Clean("till-go/...")` → `"till-go/..."` (Clean rule 2 eliminates only `.` elements, not `...`). Group = `"till-go/..."`. User tier: `filepath.Join(home, ".tillsyn/agents", "till-go/...", "passwd")` → `<home>/.tillsyn/agents/till-go/.../passwd`. `filepath.Clean` does NOT collapse `...` (Clean rule 2 is for `.` only, rule 3/4 are for `..` only). Result is a literal directory named three dots. `os.ReadFile` → `ErrNotExist` → user tier returns `(false, nil)` → fall through to embedded tier → embedded miss → `ErrAgentBodyNotFound`. NO host-file leak.
  3. `SystemPromptTemplatePath = "till-go/foo/...."` — same analysis. `"...."` ≠ `".."` and is not eliminated by `Clean`. Either rejected by basename validator (if leaf) or routes to nonexistent literal directory `"...."`.
- **Result — REFUTED.** The narrowing is sound. `path.Clean`'s documented rules (per `go doc path Clean`) eliminate ONLY `.` (rule 2) and `..` (rules 3-4) elements. Any segment with 3+ dots is a literal directory name, NOT collapsed. The basename validator's `strings.Contains(basename, "..")` substring check catches `"..."` and `"...."` as leaves; mid-path `"..."` segments route to nonexistent directories and ENOENT-fall-through. No host-file read.

#### W3-D2R2-FF2 [LOW / narrowing — `..foo` and `foo..bar` substring segments] — REFUTED

- **Attack:** the validator's narrowing rationale explicitly preserves segments like `..foo` (legitimate dotfile-like name). Confirm `filepath.Clean` does not collapse `..foo` or `foo..bar` under any code path that would reach the validator-accepted shape.
- **Method:** `go doc path Clean` (rule 3: "Eliminate each inner `..` path name element" — requires EXACT segment equality to `..`, not substring) + manual trace.
- **Trace:** `SystemPromptTemplatePath = "till-go/..foo/passwd"`. Validator splits → `["till-go", "..foo", "passwd"]`. `"..foo"` ≠ `".."` → PASSES. `path.Base` → `"passwd"`, `path.Dir` → `path.Clean("till-go/..foo")` → `"till-go/..foo"` (Clean does NOT collapse `..foo` — only exact `..` segments are collapsed per the documented rule). Group = `"till-go/..foo"`. User tier: `filepath.Join(home, ".tillsyn/agents", "till-go/..foo", "passwd")` → `<home>/.tillsyn/agents/till-go/..foo/passwd`. ENOENT on a real host (no such literal directory). Fall through to embedded tier → miss → `ErrAgentBodyNotFound`. No host-file leak.
- **Result — REFUTED.** The narrowing rationale matches `path.Clean`'s documented semantics verbatim. `..foo` and `foo..bar` are literal directory names under both `path.Clean` and `filepath.Clean`. The narrowing does not create a bypass.

#### W3-D2R2-FF3 [LOW / TrimSpace short-circuit — whitespace-only path bypasses validator] — REFUTED

- **Attack:** `assembleAgentFileBody` short-circuits the validator when `strings.TrimSpace(binding.SystemPromptTemplatePath) == ""`. A whitespace-only value (e.g. `"   "`) bypasses the validator AND falls into the empty-path branch. Does the empty-path branch open any traversal surface?
- **Method:** Trace the empty-path branch through `resolveAgentBasename` (`render.go:599-611`) and `resolveAgentGroup` (`render.go:690-697`).
- **Trace:** `binding.SystemPromptTemplatePath = "   "`. `strings.TrimSpace(...) == ""` → validator SKIPPED. `resolveAgentBasename`: `strings.TrimSpace(trimmed) == ""` (`trimmed = "   "`, TrimSpace returns `""`) → falls into the empty branch → `basename = binding.AgentName + ".md"` = `"go-builder-agent.md"` (or whatever AgentName carries). AgentName is already validated upstream in `Render` for path separators (`render.go:219-222`). `resolveAgentGroup`: same TrimSpace check → empty → returns `agentBodyDefaultGroup` = `"till-go"`. Behavior is byte-identical to `binding.SystemPromptTemplatePath = ""`. No traversal surface — group and basename are now derived from the dogfood defaults and the already-validated AgentName.
- **Result — REFUTED.** Whitespace-only paths normalize to the empty-path branch via two consistent TrimSpace checks. The dogfood defaults (`till-go` + AgentName-derived basename) are within the sandbox. No bypass. The TrimSpace short-circuit is consistent, not a vulnerability.

#### W3-D2R2-FF4 [LOW / dot segments — `.` segment accepted by validator] — REFUTED

- **Attack:** the validator rejects `..` and empty segments but does NOT reject `.` (single-dot, current-directory). A `.` segment is legitimate under `path` semantics. Confirm the `.` segment never produces a traversal under `filepath.Clean`.
- **Method:** `go doc path Clean` rule 2 ("Eliminate each `.` path name element") + `go doc filepath Clean` rule 2 (identical) + trace.
- **Trace:** `SystemPromptTemplatePath = "till-go/./passwd"`. Validator accepts (`"."` ≠ `""`, `"."` ≠ `".."`). `path.Base("till-go/./passwd")` → `path.Base(path.Clean("till-go/./passwd"))` = `path.Base("till-go/passwd")` = `"passwd"`. `path.Dir("till-go/./passwd")` = `path.Clean("till-go/.")` = `"till-go"` (`.` eliminated). Group = `"till-go"` — legitimate dogfood group. User tier: `<home>/.tillsyn/agents/till-go/passwd` — within sandbox. Reads via the legitimate user-tier path. No traversal.
- **Result — REFUTED.** `.` segments are eliminated by `Clean` (`path` and `filepath` both follow the same rule per their docs) and route to a legitimate in-sandbox path. The validator's choice to not reject `.` is correct under the documented semantics.

#### W3-D2R2-FF5 [LOW / symlink traversal — out-of-band attack vector] — REFUTED

- **Attack:** the validator is purely lexical (operates on the STRING). If `<home>/.tillsyn/agents/till-go/` is itself a symlink pointing to `/etc/`, then a fully-validated input like `SystemPromptTemplatePath = "till-go/passwd"` would have `os.ReadFile` follow the symlink and read `/etc/passwd`. Is this a falsification of the validator's security claim?
- **Method:** Threat model audit. The attack requires the attacker to have already-existing write access to the user's `~/.tillsyn/agents/` tree to plant the symlink. The validator's documented scope (per its doc-comment) is "defense against parent-traversal" + "absolute paths" — NOT "defense against pre-positioned symlinks in the user's own home directory."
- **Trace:** If attacker has write access to `~/.tillsyn/agents/`, they can also write `~/.tillsyn/agents/till-go/go-builder-agent.md` with arbitrary content directly — no symlink trick needed, the user-tier hit succeeds with attacker-content. Symlink doesn't expand the attack surface beyond what direct write already grants.
- **Result — REFUTED as validator-falsification.** Symlink resolution is a filesystem-layer concern. The validator's threat model (per its doc-comment) is "attacker-controllable `binding.SystemPromptTemplatePath` values" — the validator successfully prevents `SystemPromptTemplatePath` from escaping the sandbox. A pre-existing symlink inside the user-owned sandbox is a separate threat-model class (host-compromise / chain-of-trust on the user's own home directory) that no string-validator can address. The W3-D23-FF1 fix was scoped to the SystemPromptTemplatePath surface — the validator achieves what it documents. **Not a counterexample.**

#### W3-D2R2-FF6 [LOW / wrap-chain — sentinel survives the double wrap through Render] — REFUTED

- **Attack:** the validator's error is wrapped at `assembleAgentFileBody` (`fmt.Errorf("%w: %q: %s", ErrInvalidAgentTemplatePath, trimmed, err.Error())`) and then wrapped AGAIN at `Render` (`fmt.Errorf("render: agent file: %w", err)` at `render.go:243`). Confirm `errors.Is(returnedErr, ErrInvalidAgentTemplatePath)` survives both wraps.
- **Method:** Read `render.go:230-243` (`renderAgentFile` returns wrapped as `"render: agent file: %w"`). Read `render.go:476-487` (`assembleAgentFileBody` returns `fmt.Errorf("%w: %q: %s", ErrInvalidAgentTemplatePath, ...)`). Read the test `TestAssembleAgentFileBody_RejectsPathTraversalInGroup` (`render_test.go:1195-1201`) which asserts `errors.Is(err, render.ErrInvalidAgentTemplatePath)`.
- **Trace:** Both wraps use `%w`. `errors.Is` traverses the chain via `Unwrap` until it matches the sentinel. Test passes per builder's `mage testFunc` GREEN. `mage ci` GREEN confirms full-suite preservation.
- **Result — REFUTED.** Wrap chain is `%w`-correct at both levels. Sentinel detection survives.

#### W3-D2R2-FF7 [LOW / backslash defense — defense-in-depth for Windows-host adopter] — REFUTED

- **Attack:** the validator rejects `\` anywhere in the path. On macOS/Linux, `filepath.Join` treats `\` as a literal byte (NOT a separator), so backslash injection has no traversal effect on POSIX hosts. Is the backslash rule premature / over-rejecting / YAGNI?
- **Method:** Read the validator (`render.go:649-651`). Read `go doc filepath Join` ("separating them with an OS specific Separator"). On Windows, `filepath.Separator == '\\'`, so a backslash IS a separator on that platform.
- **Trace:** On Windows, `binding.SystemPromptTemplatePath = "till-go\\..\\..\\..\\etc\\passwd"` would have `filepath.Join` treat `\` as a separator, and `filepath.Clean` would collapse the `..` segments — opening the same traversal surface that the slash-based attack opens on POSIX. The backslash defense closes that cross-platform port preemptively.
- **Result — REFUTED.** Backslash defense is defense-in-depth that closes the Windows-port traversal surface. NOT over-rejection because no legitimate `binding.SystemPromptTemplatePath` value should ever contain `\` (the W3-FF5 LOCKED contract uses slash exclusively for embed.FS path conformance). Per builder's worklog: "the canonical form per W3-FF5 LOCKED is slash-separated" — correct.

#### W3-D2R2-FF8 [LOW / empty-trailing-segment rejection — `till-go/` rejected] — REFUTED

- **Attack:** `SystemPromptTemplatePath = "till-go/"` produces `strings.Split("till-go/", "/")` = `["till-go", ""]`. Empty trailing segment → REJECTED by validator. But `path.Base("till-go/")` would return `"till-go"` (per `go doc path Base`: "Trailing slashes are removed before extracting the last element"). Is the rejection over-rejecting a legitimate directory-style input?
- **Method:** Read `go doc path Base`. Trace acceptance criteria: `SystemPromptTemplatePath` is documented as a path to a file (`till-go/go-builder-agent.md` per the canonical positive control), NOT a directory.
- **Trace:** A `binding.SystemPromptTemplatePath = "till-go/"` value is semantically incoherent (you can't read a directory as an agent body). The validator's rejection is correct — it prevents a confusing failure-mode downstream where the resolver would silently use the basename `"till-go"` as the agent file name and route to a nonexistent file.
- **Result — REFUTED.** Empty-trailing-segment rejection is correct. Not over-rejecting.

#### W3-D2R2-FF9 [LOW / project-tier scope — group still unused at project tier] — REFUTED

- **Attack:** confirm the post-round-2 codepath does NOT introduce any usage of the unvalidated-anywhere-else `<group>` at the project tier. The W3-D23-FF1 finding (per its FF12 sub-bullet) noted project tier is immune because group is unused there.
- **Method:** Re-read `readProjectTierAgent` (`render.go:707-720`) post-round-2.
- **Trace:** `filepath.Join(projectWorktree, projectAgentsSubdir, basename)` — three args, NO group component. Project tier remains group-immune. Round-2 changes touched only `assembleAgentFileBody` entry and the validator helper; project tier signature unchanged.
- **Result — REFUTED.** Project-tier scope is preserved. The validator improvement is user-tier-focused, which matches the W3-D23-FF1 vector.

#### W3-D2R2-FF10 [HIGH-tier gate / build — `mage ci` post-fix] — REFUTED

- **Attack:** verify the round-2 fix does not regress any of the 3036 tests across the 25 packages, especially the render package's 38 tests (per builder's worklog).
- **Method:** Run `mage ci` from the project worktree.
- **Trace:** Full `mage ci` invocation. Output captured below.
- **Result — REFUTED — `mage ci` GREEN.**

```
[SUCCESS] All tests passed
  3036 tests passed across 25 packages.

[SUCCESS] Coverage threshold met
  All packages are at or above 70.0% coverage.
  Render package: 81.3%.

Build
[SUCCESS] Built till from ./cmd/till
```

Round-1's report cited 3028 tests at 79.3% render coverage; round-2 lands at 3036 tests at 81.3% render coverage. The delta (+8 tests, +2.0% coverage) is consistent with the 4 new regression tests including 4 sibling-case sub-tests within `TestAssembleAgentFileBody_RejectsPathTraversalSiblingCases`. No package regression.

#### W3-D2R2-FF11 [LOW / regression-test coverage — sibling cases reach every reject rule] — REFUTED

- **Attack:** verify each of the validator's three reject rules (absolute / `..` segment / empty segment) is independently covered by a test, AND that the positive-control + empty-path-control prevent over-rejection.
- **Method:** Read `TestAssembleAgentFileBody_RejectsPathTraversalSiblingCases` (`render_test.go:1218-1268`), `_AcceptsLegitimateTemplatePath` (`:1274-1300`), `_EmptyPathStillRoutesToTillGoDefault` (`:1307-1333`).
- **Trace:** Sibling cases — `absolute_etc_passwd` covers rule 1; `trailing_dotdot` + `mid_path_dotdot` cover rule 2 (two positional variants); `double_slash` covers rule 3. Positive control covers no-over-rejection on the canonical positive case. Empty-path control covers the W3-FF5 LOCKED sentinel survives the round-2 addition. Backslash rule is the only one not test-pinned — the builder's three sibling cases plus the original attack string don't exercise backslash. Minor coverage gap, NOT a counterexample (the rule is unreachable on macOS/Linux test hosts because no legitimate POSIX input contains `\`, and the rule is defense-in-depth for a future Windows port).
- **Result — REFUTED.** Test coverage is adequate for the three rules with documented attack vectors. Backslash rule is uncovered-by-test but is preemptive defense-in-depth; the absence of a test does not falsify the validator's claim.

#### W3-D2R2-FF12 [LOW / closure of W3-D23-FF1] — REFUTED

- **Attack:** the round-2 fix claims W3-D23-FF1 is closed. Verify by mentally replaying the exact W3-D23-FF1 attack trace against the post-round-2 codepath.
- **Method:** Re-execute the FF1 trace step-by-step against the new validator.
- **Trace:** `binding.SystemPromptTemplatePath = "till-go/../../../../../../etc/passwd"`. (1) `assembleAgentFileBody` entry: `trimmed = "till-go/../../../../../../etc/passwd"`, non-empty. (2) `validateAgentTemplatePath(trimmed)`: not absolute (no leading `/`), no backslash. Split on `/` → `["till-go", "..", "..", "..", "..", "..", "..", "etc", "passwd"]`. First `..` segment encountered triggers `errors.New("parent-traversal segment '..' not allowed")`. (3) `assembleAgentFileBody` wraps as `fmt.Errorf("%w: %q: %s", ErrInvalidAgentTemplatePath, ...)`. (4) `renderAgentFile` wraps as `"render: agent file: %w"`. (5) `Render` calls `rollback.run()` (per `render.go:241`) which `os.Remove`s `system-prompt.md` AND `os.RemoveAll`s `<bundle>/plugin/`. (6) Returns `("", wrappedErr)`. `errors.Is(err, ErrInvalidAgentTemplatePath)` → TRUE. Test `TestAssembleAgentFileBody_RejectsPathTraversalInGroup` confirms this on every host where `/etc/passwd` exists (skips otherwise). User-tier `filepath.Join` is NEVER reached. Host-file leak path is closed.
- **Result — REFUTED. W3-D23-FF1 is CLOSED.** The round-2 validator runs BEFORE `path.Dir` / `path.Base` derive group + basename, BEFORE `readProjectTierAgent` / `readUserTierAgent` / `readEmbeddedTierAgent` execute. Every host-file-read codepath is gated by the validator. The defense-in-depth disk-level assertion (`os.Stat(agentPath)` must return `os.ErrNotExist`) provides a second-layer guarantee even if a future refactor accidentally swallows the validator error — the rollback wipes any partial bundle so a leak via partially-written agent file is impossible.

### Severity breakdown

- **HIGH:** 0
- **MEDIUM:** 0
- **LOW:** 12 — all REFUTED (11 narrowing / scope / wrap / build attacks + 1 closure-verification of W3-D23-FF1).

### mage ci result

```
[SUCCESS] All tests passed
  3036 tests passed across 25 packages.

[SUCCESS] Coverage threshold met
  All packages are at or above 70.0% coverage.
  Render package: 81.3%.

Build
[SUCCESS] Built till from ./cmd/till
```

### Summary

**Verdict: PASS — no new CONFIRMED counterexamples.** The round-2 validator (`validateAgentTemplatePath` + `ErrInvalidAgentTemplatePath` + the call-site wiring) is sound under the documented `path` / `filepath` semantics. Every probed narrowing decision (exact-`..` rejection vs `..foo` / `...` / `....` acceptance, `.` segment acceptance, backslash rejection, TrimSpace short-circuit symmetry, empty-trailing-segment rejection) matches what `go doc path Clean` documents and is consistent with the round-2 worklog's design rationale. The wrap chain preserves `errors.Is(err, ErrInvalidAgentTemplatePath)` through both `assembleAgentFileBody` and `Render` wraps. The rollback path wipes any partially-written bundle on validator rejection. Project-tier immunity (group unused) is preserved. `mage ci` is GREEN at 3036 tests / 25 packages / 81.3% render coverage (+8 tests, +2.0% render coverage vs round-1).

**W3-D23-FF1 closure status: CLOSED.** The exact attack string `"till-go/../../../../../../etc/passwd"` is now rejected at the validator step BEFORE `path.Dir` / `path.Base` derive the group component. The user-tier `filepath.Join` codepath that previously cancelled to `/etc/passwd` is unreachable. Test `TestAssembleAgentFileBody_RejectsPathTraversalInGroup` pins the attack string verbatim and asserts `errors.Is(err, render.ErrInvalidAgentTemplatePath)`; the defense-in-depth `os.Stat(agentPath)` assertion catches any future regression where the validator might be bypassed without erroring.

**Narrowing soundness — explicit per `path` / `filepath` documented semantics:**

- `path.Clean` and `filepath.Clean` (documented per `go doc`) ELIMINATE only exact `.` and exact `..` segments. Segments with `..` as substring (`..foo`, `foo..bar`) and segments with 3+ dots (`...`, `....`) are LITERAL directory names — preserved through `Clean`, not collapsed. The validator's exact-equality narrowing (`seg == ".."`) matches the only-collapsible shape, so the narrowing is necessary AND sufficient for the documented attack vector.
- The basename validator (`validateAgentBasename`'s `strings.Contains(basename, "..")` substring rule) is a defense-in-depth layer on the LEAF only — it catches `"..."` leaves that the template-path validator's exact-segment rule lets through. The two validators' overlap is intentional, not redundant: template-path validator gates the full path's group component (the W3-D23-FF1 surface); basename validator gates the leaf (the original W3-D2 surface). Both fire at different layers.

No routing to orchestrator beyond the closure confirmation. W3-D23-FF1 closed; no new findings.

### Hylla Feedback

N/A — round-2 falsification touched only Go files in `internal/app/dispatcher/cli_claude/render/` whose post-W3.D2 round-2 changes (`d671b91`) have not been ingested. Hylla's last ingest predates the round-2 commit, so the new symbols under attack (`validateAgentTemplatePath`, `ErrInvalidAgentTemplatePath`, the wired call site) are not in the index. The falsification relied entirely on `Read` against `render.go` + `render_test.go` + the BUILDER_WORKLOG entry + `go doc path Clean` / `go doc path Dir` / `go doc path Base` / `go doc filepath Join` / `go doc filepath Clean` for the documented language semantics that ground the narrowing analysis. The staleness window is structurally guaranteed pre-cascade-ingest. No fallback miss to record.

---

## Droplet 4c.6.W2.D1 — Round 1

**Reviewer:** go-qa-falsification-agent (subagent, doc-only mode).
**Date:** 2026-05-11.
**Droplet:** `4c.6.W2.D1 — internal/fsatomic/ atomic file-write helper (local-implement)`.
**Artifact under attack:** `internal/fsatomic/atomic.go` (91 lines, NEW package) + `internal/fsatomic/atomic_test.go` (115 lines, 4 tests).

### Counterexamples

- **W2-D1-FF1 — `mage ci` FAILS with coverage gate violation (CONFIRMED).**
  - **Family:** B5 spec-compliance (acceptance bullet violation).
  - **Severity:** **high — blocks droplet completion.**
  - **Claim under attack:** plan acceptance bullet `mage ci green.` (DROP_4c.6.W2_TILL_INIT/PLAN.md line 63).
  - **Observed:** `mage ci` exits non-zero with:
    ```
    [ERROR] Coverage threshold not met
      Each package must stay at or above 70.0% coverage.
      github.com/evanmschultz/tillsyn/internal/fsatomic 64.0%
    Error: coverage below 70.0%: github.com/evanmschultz/tillsyn/internal/fsatomic 64.0%
    ```
  - **Repro:** `mage ci` from `/Users/evanschultz/Documents/Code/hylla/tillsyn/main/` — full output captured above; 3077 tests across 26 packages pass, then the coverage gate fires on `internal/fsatomic` at 64.0% vs the 70.0% floor. CLAUDE.md "Build Verification" §"Coverage below 70% is a hard failure" pins the gate as load-bearing.
  - **Root cause:** the 4 shipped tests cover only the success path plus the `os.CreateTemp` error branch. Five error-return branches inside `WriteFile` are unexercised:
    1. Line 66-69 — `f.Write` error path.
    2. Line 71-74 — `f.Sync` error path.
    3. Line 76-79 — `f.Chmod` error path.
    4. Line 81-83 — `f.Close` error path.
    5. Line 85-87 — `os.Rename` error path.
    Plus the deferred cleanup body at lines 60-64 is never exercised with `success=false` AFTER a successful `CreateTemp` — `TestWriteFile_CleansUpTempOnError` injects failure at `CreateTemp` itself (parent dir missing), so the function returns at line 52 before the success-flag tracker + defer are installed. The plan acceptance's "TestWriteFile_CleansUpTempOnError: inject an error into the write path … assert NO `.tmp-*` files remain in the parent dir after the failed call" is technically satisfied by the current test, but only via the trivial branch — the cleanup defer the implementation actually relies on remains uncovered.
  - **Fix hint (low-cost):** add at least one test that exercises a post-`CreateTemp` failure to land the defer body + at least one error-wrap on disk. Easiest realistic shape — `TestWriteFile_RenameFailsWhenTargetIsDirectory`: pre-create a directory at the target path (`os.Mkdir(target, 0o755)` before `WriteFile(target, …)`), call `WriteFile(target, []byte("x"), 0o644)`, assert the returned error wraps a `*os.LinkError` (rename of file → existing directory errors per `go doc os.Rename`: "If newpath already exists and is a directory, Rename returns an error"), then `os.ReadDir(filepath.Dir(target))` and assert NO `.tmp-*` entry remains. This single test simultaneously: (a) exercises the rename-error wrap on line 85-87, (b) exercises the deferred cleanup body at line 60-64 with a real temp file present, (c) directly validates the same-directory atomicity contract by proving the temp lives next to the target (else `os.Rename` would have been cross-filesystem, not "target is directory"). Estimated lift: ≥4 lines / 4 statements covered → coverage moves to ~76-80%, clearing the 70% gate.
  - **Family-table impact:** B5 CONFIRMED. Falsification verdict: **FAIL** until coverage gate clears.

### Findings (non-CONFIRMED, recorded for audit)

- 1.1 [Family: spec-compliance] [severity: info] Plan acceptance bullet (line 64) says future-migration intent points to `hylla-shared`; the package doc-comment (line 24) says "`hylla-utility location (see SKETCH.md §9.6)`". Same intent, different label. Plan also says "Optional helper for parent-dir fsync as a separate exported function (`SyncDir(path string) error`) if W2.D5 needs strong durability; if not, leave as a future addition (YAGNI)." Shipped surface omits `SyncDir` — consistent with the YAGNI carve-out (no Drop-4c.6 consumer calls it). Not a counterexample.

  *Verdict on this attack: REFUTED — phrasing nit + intentional YAGNI omission, both within plan tolerance.*

- 1.2 [Family: hidden-coupling] [severity: info] `os.CreateTemp` (per `go doc`) creates the temp file with mode `0o600 & ~umask`. On the call path, `f.Chmod(perm)` (line 76) is called BEFORE `f.Close()` (line 81) and BEFORE `os.Rename` (line 85), so the final on-disk mode after rename matches `perm`. The doc-comment line 41-42 captures this correctly: "Permissions are applied to the temp file before rename so the final mode is correct at the moment the file becomes visible at the target path." No window during which a reader could observe the target at the temp's 0o600 default. REFUTED.

- 1.3 [Family: contract-preservation] [severity: info] `f.Sync()` (line 71) is called BEFORE `f.Close()` (line 81) which is BEFORE `os.Rename` (line 85). This is the textbook sync-before-rename ordering — without it, a crash between rename and disk-flush could leave the target pointing at an unflushed inode on POSIX. The package doc-comment (lines 6-11) pins same-directory-as-target as the atomicity precondition; the implementation honors it via `os.CreateTemp(filepath.Dir(path), …)` at line 50. REFUTED.

- 1.4 [Family: yagni] [severity: info] Plan stretch-goal `TestWriteFile_AtomicVisibility` (planner explicitly says "Skip if too flaky on CI; documented as a future test") is omitted; documented as a future test by being absent (no negative claim made about atomicity-under-concurrency in the doc-comment beyond "never observed half-written by a concurrent reader on POSIX filesystems" at lines 38-39, which is grounded in the rename-atomicity contract, not asserted via test). Plan allows this. REFUTED — but note this carve-out is what permitted the coverage gap to land; tightening this test (or substituting the rename-fails-when-target-is-directory test) is the cheapest path to clearing W2-D1-FF1.

- 1.5 [Family: contract-preservation] [severity: info] Concurrent writers `WriteFile(samePath, …)` from two goroutines: each goroutine's `os.CreateTemp` returns a unique filename per `go doc os.CreateTemp` ("Multiple programs or goroutines calling CreateTemp simultaneously will not choose the same file"). Each goroutine's `os.Rename` is atomic (POSIX same-filesystem). Post-condition: target ends up with the bytes from whichever rename completes last; readers see either A's full bytes or B's full bytes, never half-written. Matches the doc-comment claim line 38-39. REFUTED.

- 1.6 [Family: contract-preservation] [severity: info] Empty data `WriteFile(path, []byte{}, 0o644)` — `f.Write([]byte{})` returns `(0, nil)` per the `io.Writer` contract; the rest of the pipeline runs unchanged; result is a 0-byte file at the target. Large data (1MB+): `*os.File.Write` blocks until all bytes are written or an error occurs; no internal buffer-size limit beyond what the kernel handles. REFUTED on both.

- 1.7 [Family: contract-preservation] [severity: info] Path edge cases: `WriteFile(".", …)` — `filepath.Dir(".")` returns `.`, `filepath.Base(".")` returns `.`, `os.CreateTemp(".", "..tmp-*")` creates `..tmp-<rand>` in cwd, then `os.Rename("..tmp-xyz", ".")` errors because `.` is a directory; defer cleans temp. `WriteFile("/tmp/", …)` — analogous: rename fails on directory target. `WriteFile(path-with-mid-string-file-as-dir, …)` — `os.CreateTemp` fails ENOTDIR; early return at line 52, no defer installed (no temp to clean — none was created). All edge cases either error cleanly with no residue or error cleanly with the defer cleaning residue. REFUTED.

- 1.8 [Family: shipped-but-not-wired] [severity: info] Package has no consumers yet — D5 is the first consumer per plan blocker chain ("Blocked by: D1" at line 162). Today the package is a leaf: shipped, tested (within coverage gap), but unwired in production code. Per CLAUDE.md "shipped-but-not-wired" framing (memory `feedback_tillsyn_enforces_templates`), this is acceptable for a foundation droplet whose downstream consumers are queued in the same drop. The risk closes when D5 lands. REFUTED (provisional — promotes to CONFIRMED if D5 lands without invoking `fsatomic.WriteFile`, which a future round will check).

### Attack family table

| Family                  | Result    | Notes                                                                 |
| ----------------------- | --------- | --------------------------------------------------------------------- |
| B1 test-coverage        | CONFIRMED | 4 tests shipped, 5 error branches + deferred-cleanup-after-successful-CreateTemp unexercised → 64.0% < 70.0% gate. See W2-D1-FF1. |
| B2 contract-preservation| REFUTED   | Same-dir temp (line 47/50), sync-before-rename (71→85), chmod-before-close (76→81), close-before-rename (81→85), defer-with-success-flag cleanup (58-64) — all match POSIX-atomic write-temp + rename idiom. `go doc os.CreateTemp` + `go doc os.Rename` confirm the contracts. Concurrent-writer / empty-data / large-data / path-edge-case probes all REFUTED (Findings 1.5–1.7). |
| B3 hidden-coupling      | REFUTED   | `os.CreateTemp`'s 0o600-default-with-umask is shadowed by the explicit `f.Chmod(perm)` before rename, so callers see exactly `perm`. No init-time global state, no package-level mutable state. Package imports are stdlib-only (`fmt`, `os`, `path/filepath`). No init() side effects. REFUTED. |
| B4 yagni                | REFUTED   | Shipped surface = single `WriteFile` function. No staged-write struct, no `SyncDir`, no rename-only helper — matches package doc-comment scope statement (lines 20-25) and plan YAGNI carve-out (acceptance bullet "Optional helper for parent-dir fsync … if not, leave as a future addition"). REFUTED. |
| B5 spec-compliance      | CONFIRMED | Plan acceptance bullet `mage ci green.` (line 63) FAILS — gate exits non-zero on `internal/fsatomic 64.0% < 70.0%`. Acceptance bullet `Tests at internal/fsatomic/atomic_test.go: [list of 4 tests + stretch-goal 5th]` (lines 56-61) is structurally met (4 named tests shipped) but the implementation's defensive error branches are unreached, hence the gate failure. See W2-D1-FF1. |
| B6 shipped-but-not-wired| REFUTED (provisional) | No D5 consumer yet — finding 1.8 marks this as REFUTED today, with promotion risk if D5 lands a `.gitignore` / `agents.toml` write path that doesn't actually call `fsatomic.WriteFile`. Re-check next round at D5 landing. |
| B7 prompt-injection     | EXHAUSTED | DORMANT pre-team-feature per agent definition.                        |

### `mage ci` result

**FAIL.** Captured output:

```
3077 tests passed across 26 packages.

[ERROR] Coverage threshold not met
  Each package must stay at or above 70.0% coverage.
  github.com/evanmschultz/tillsyn/internal/fsatomic 64.0%
Error: coverage below 70.0%: github.com/evanmschultz/tillsyn/internal/fsatomic 64.0%
```

Tests pass (4/4 in `fsatomic`, 3077/3077 total). The coverage gate is the failure point.

### Summary

**Verdict: FAIL — 1 CONFIRMED counterexample (W2-D1-FF1) blocks droplet completion.** The implementation is conceptually sound — same-dir temp, sync-before-rename, chmod-before-close, defer-with-success-flag cleanup all line up with the textbook POSIX atomic-write pattern and `go doc`-confirmed stdlib contracts. Six of seven attack families (B2 contract-preservation, B3 hidden-coupling, B4 yagni, B6 shipped-but-not-wired (provisional), B7 prompt-injection) come back REFUTED / N/A. The single counterexample is the coverage gate: `mage ci` FAILS with `internal/fsatomic 64.0% < 70.0%` because the 4 shipped tests cover only the success path plus the `os.CreateTemp`-itself-fails branch. Five error-return branches inside `WriteFile` plus the deferred cleanup body after a successful `CreateTemp` are unexercised.

**Fix hint pinned in W2-D1-FF1:** add `TestWriteFile_RenameFailsWhenTargetIsDirectory` (pre-create a dir at target, expect rename to fail, expect no `.tmp-*` residue in parent). One test covers ≥4 statements: rename-error wrap (line 85-87) + deferred cleanup body (60-64) + same-directory atomicity invariant + the `_ = os.Remove(tmpName)` line. Coverage projected ~76-80%. Routing: orchestrator respawns builder for round-2 to land the additional test; falsification re-runs once `mage ci` reports green.

**Builder must not bypass mage to verify** — per CLAUDE.md "Build Verification" §2, raw `go test -coverprofile` is forbidden; `mage ci` is the authoritative gate. Builder may use `mage test-pkg ./internal/fsatomic` for tight test-iteration loops, then `mage ci` for the gate clearance proof.

### Hylla Feedback

None — Hylla answered everything needed, though the falsification did not require querying Hylla. The attack surface for W2.D1 is a single NEW 91-line file (`atomic.go`) + a single NEW 115-line test file (`atomic_test.go`), both unindexed since they're post-last-ingest, so `Read` + `go doc os.CreateTemp` / `go doc os.Rename` / `go doc os File.Chmod` / `go doc os File.Sync` plus the `mage ci` gate output were the load-bearing evidence sources. The plan blocker chain (`DROP_4c.6.W2_TILL_INIT/PLAN.md`) and the `magefile.go`-encoded coverage rule (visible in `mage ci` output) were sufficient to ground every claim. The package is a foundation leaf with no consumers in this drop yet, so call-site blast-radius (`hylla_graph_nav`) would have returned empty even if the symbol were indexed — no fallback miss to log.

---

## Droplet 4c.6.W2.D4 — Round 1

**Reviewer:** go-qa-falsification-agent (subagent).
**Date:** 2026-05-11.
**Droplet:** `4c.6.W2.D4 — runInitTUI bubbletea walk for project name + group picker`.
**Artifact under attack:** `cmd/till/init_cmd.go` (modify — replaces D3a's `runInitTUI` stub with a real bubbletea walk; adds `initTUIStep` enum, `initTUIGroupRow`, `initTUIGroupRows`, `initTUIModel`, `newInitTUIModel`, `Init` / `Update` / `View` / `Done` / `Cancelled` / `Payload` methods, `nextEnabledGroupRow` / `prevEnabledGroupRow` helpers). `cmd/till/init_cmd_test.go` (modify — `TestInit_BareInvocation_ReturnsTUIStubError` rewrite using `programFactory` stub returning Done-state model; three new tea-tests `TestRunInitTUI_AcceptsDefaultNameAndSelectsTillGo`, `TestRunInitTUI_DisabledTillGddIsUnselectable`, `TestRunInitTUI_EscCancelsWalk`).

### Counterexamples

- **W2-D4-FF1** [Family: hidden-coupling / CI-gate] [severity: blocks-drop] `mage ci` FAILS with `internal/fsatomic 64.0% < 70.0%` coverage threshold. **NOT caused by D4** — D4 touched zero lines in `internal/fsatomic/`. The failure is the same W2-D1-FF1 counterexample already CONFIRMED in the W2.D1 Round 1 falsification (lines 1700-1707 of this file). The fix is the W2.D1 round-2 test addition pinned there (`TestWriteFile_RenameFailsWhenTargetIsDirectory`). **D4-side action: none.** **Drop-side action: respawn W2.D1 builder for round-2 (already pending per the W2.D1 verdict).** Recording here because the prompt's "`mage ci` must be GREEN" gate fails at the drop level even though D4's implementation is sound; the orchestrator needs to see the cascade-state truth (D4-code-OK + drop-CI-red-from-D1) rather than re-blame D4. Repro: `mage ci` against the current worktree, observe `[ERROR] Coverage threshold not met` with `github.com/evanmschultz/tillsyn/internal/fsatomic 64.0%`. Routing: cross-reference W2-D1-FF1; the orchestrator already owns the respawn directive.

### Findings (non-CONFIRMED, recorded for audit)

- 1.1 [Family: hidden-coupling / TUI semantics] [severity: low] **Ctrl-C as a keyboard event on the name step is NOT handled by `Update`'s `initTUIStepName` switch arm.** The arm only matches `tea.KeyEsc` and `tea.KeyEnter`; Ctrl-C arrives as `tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl}` and falls into `default:` which forwards to `m.nameInput.Update(msg)`. Bubbletea v2's `bubbles/textinput` does NOT trap Ctrl-C, so the keyboard event is consumed silently. In the typical terminal-driven path Ctrl-C never reaches the model — bubbletea v2's default signal handler (NOT opted out via `tea.WithoutSignalHandler()` in `programFactory`'s `tea.NewProgram(m)`) translates SIGINT into a `tea.QuitMsg` and `tea.Program.Run()` returns with `tea.ErrProgramKilled`, which `runInitTUI` wraps as `"till init: run tui: %w"`. So the *common case* (terminal Ctrl-C) produces a clean user-facing error, not a panic — satisfying the prompt's "clean exit (not panic, not segv)" bar. The *uncommon case* (Ctrl-C delivered as a raw keypress, e.g. some Windows terminals or test harnesses) silently no-ops on the name step and is silently no-op on the group step. PLAN.md acceptance does not require explicit Ctrl-C handling; SKETCH §9.3 does not require it either. *Verdict on this attack: REFUTED — terminal-driven Ctrl-C is handled cleanly by the framework default; raw-keypress Ctrl-C is a corner case outside D4's acceptance bar. Fix hint (optional refinement, NOT a counterexample): add `case msg.Mod == tea.ModCtrl && msg.Code == 'c': m.step = initTUIStepCancelled; return m, tea.Quit` to both step arms so cross-platform Ctrl-C reports `Cancelled()` symmetrically with Esc.*

- 1.2 [Family: yagni / defense-in-depth] [severity: info] **The `if row.Disabled` Enter-handler check in `initTUIStepGroup` is dead code under the current cursor-movement helpers.** `nextEnabledGroupRow` / `prevEnabledGroupRow` skip disabled rows, so `m.groupCursor` can never equal an index whose `initTUIGroupRows[i].Disabled` is true under any sequence of Up / Down / k / j keypresses. `groupCursor` is also not exposed to any external mutation path. The disabled-Enter guard is defense-in-depth for future row additions (e.g., if a future drop adds a disabled row between two enabled rows, the cursor-movement helpers still skip it, but a hypothetical future direct-jump key like `g` / `G` could land on it). Builder's worklog line 2099 explicitly justifies this as "two layers of defense." Today's test (`TestRunInitTUI_DisabledTillGddIsUnselectable`) exercises the cursor-skip layer (Down-Down stops at till-go) but does NOT exercise the Enter-on-disabled-row layer because cursor can't reach there. *Verdict on this attack: REFUTED — dead code today, but the defense-in-depth justification is correct and the cost is two lines plus a comment; not a YAGNI violation. The test gap on the second layer is acceptable because the layer is unreachable through the current keymap.*

- 1.3 [Family: contract-preservation / forward-coupling] [severity: low] **`runInitTUI` does NOT call `validateInitPayload` on the gathered payload.** The TUI walk produces `finalPayload.Name = strings.TrimSpace(m.nameInput.Value())` (falling back to `m.defaultName = filepath.Base(cwd)` on empty) and `finalPayload.Group = "till-gen"|"till-go"` (cursor-restricted to enabled rows). Then `runInitTUI` reads `_ = final.Payload()` (line 350) and returns the D5 stub error — never validating. The JSON-mode path (`runInitJSON`) DOES call `validateInitPayload` (line 368). Asymmetry: if a future D5 plugs both branches into the same file-copy pipeline, the JSON branch's `Group` is validator-gated but the TUI branch's `Group` is cursor-gated only. Today the TUI cursor cannot land on `till-gdd` (disabled-skip helpers), so the validator-gated invariant is *coincidentally* upheld via the keymap. A future row addition that's disabled-by-default but reachable via a new keymap (or a programmatic-test direct-set of `groupCursor`) could leak an un-validated group into the pipeline. *Verdict on this attack: REFUTED for D4 — acceptance bar is "after collection, runInitTUI returns the gathered payload-equivalent and lets the caller dispatch to D5's pipeline" (PLAN.md line 135). D4 ships exactly that. Fix hint pinned for D5: add a `validateInitPayload(final.Payload())` call between `final.Done()` and the file-copy dispatch, OR factor the post-collection branch into a shared `runInitFromPayload(stdout, opts, payload)` helper that both modes call. Not a D4 counterexample; routing as D5 forward-dependency.*

- 1.4 [Family: hidden-coupling / D7.5 latent value-capture] [severity: info] **D4 does NOT trip the D7.5-reported `rootOpts` value-capture latent bug.** Verified via direct `Read` of `cmd/till/init_cmd.go:325-327`: `runInitTUI(stdout io.Writer, opts rootCommandOptions)` body opens with `_ = stdout` + `_ = opts` — neither field is read. The cobra closure in `newInitCommand`'s `RunE` (line 68-77) captures `rootOpts` by value at construction time; the closure is invoked at run-time when cobra dispatches `till init`. If `rootOpts.appName` were mutated by `--app` flag parsing between construction and dispatch, the closure would see the construction-time snapshot. But D4 doesn't read any opts field, so the latent bug is dormant. *Verdict on this attack: REFUTED for D4 — the prompt explicitly flagged this as a latent bug to verify D4 doesn't activate, and verification confirms D4 doesn't. Routing: D5 / D7 will activate this surface when they wire `appName` / `homeDir` resolution; the orchestrator should pre-route a D5 attention to either fix the closure-capture (capture by pointer or re-read `cmd.Flags()` inside `RunE`) or accept and document the snapshot-semantics contract.*

- 1.5 [Family: contract-preservation / Value-receiver Update] [severity: info] **`func (m initTUIModel) Update(msg tea.Msg) (tea.Model, tea.Cmd)` uses a value receiver.** The model is copied on every Update call. The `m.nameInput, cmd = m.nameInput.Update(msg)` propagation works because (a) `textinput.Model.Update` returns `(textinput.Model, tea.Cmd)` (verified via bubbles/v2 idiom citations in worklog line 2107-2109 pointing at `internal/tui/model.go:10200`), (b) the assignment writes back to the local copy `m`, and (c) `return m, cmd` returns the mutated copy to the bubbletea runtime which adopts it as the new model. The same convention is universal in bubbletea v2 examples (every Context7 doc snippet uses value receivers). No data race because bubbletea Update runs single-threaded on the program's main goroutine. *Verdict on this attack: REFUTED — value-receiver convention is correct for bubbletea v2 and matches every existing in-repo bubbletea model (`internal/tui/model.go`'s `Model.Update`, the `cmd/till/main_test.go` `scriptedProgram` test seam).*

- 1.6 [Family: contract-preservation / default name edge cases] [severity: info] **`filepath.Base(cwd)` edge cases — `/`, `.`, unicode, spaces — are passed through to `defaultName` and propagate into `finalPayload.Name`.** Per `go doc filepath.Base`: `filepath.Base("/")` returns `"/"`; `filepath.Base(".")` returns `"."`; unicode and spaces pass through verbatim. The textinput's `CharLimit = 120` caps input length but doesn't reject any character class. PLAN.md acceptance pins the default as `filepath.Base(cwd)` verbatim with "user can edit," so D4's behavior matches the contract literally. Whether `/` or `.` is a *valid* project name is a D5/D7 concern (project-record creation may reject it). *Verdict on this attack: REFUTED for D4 — contract matches spec. Forward-routing for D5/D7: project-name validation belongs in `validateInitPayload` extension or in `Service.CreateProject`, and that validator should also be applied to the TUI branch per Finding 1.3.*

- 1.7 [Family: shipped-but-not-wired / programFactory production] [severity: info] **Production `programFactory` is real, not a test-only stub.** Verified via Hylla `hylla_node_full` on `github.com/evanmschultz/tillsyn/cmd/till/programFactory` (snapshot 5): content is `var programFactory = func(m tea.Model) program { return tea.NewProgram(m) }` declared in `cmd/till/main.go`. The `program` interface is `type program interface { Run() (tea.Model, error) }` and `*tea.Program` (returned by `tea.NewProgram`) satisfies it via its `Run() (tea.Model, error)` method (bubbletea v2 canonical). `runInitTUI` calls `programFactory(m).Run()` which dispatches to the real `tea.Program.Run()` in production. The test stub (`scriptedProgram` in `main_test.go`, body `func (p scriptedProgram) Run() (tea.Model, error) { if p.runFn == nil { return p.model, nil }; return p.runFn(p.model) }`) replaces the var only inside test scope. *Verdict on this attack: REFUTED — the production wiring is real, not a dangling helper. The test seam is symmetric to the existing main-TUI test seam (`cmd/till/main_test.go:393-419` per worklog line 2098).*

- 1.8 [Family: yagni / Init returns nil] [severity: info] **`Init() tea.Cmd` returns `nil`.** No startup command is fired. The textinput is constructed pre-focused via `ti.Focus()` in `newInitTUIModel:153`, so cursor blink is internal to textinput's own state machine (triggered on first Update reflow). No data-load or async init is needed for a two-step linear walk. *Verdict on this attack: REFUTED — Init returning nil is the canonical bubbletea pattern when no async startup work is required (Context7 examples confirm; in-repo `internal/tui/model.go:1580` returns a load command but its model has DB-bound startup work that D4's walk does not).*

- 1.9 [Family: prompt-injection / TUI input attack] [severity: info] **The name textinput's `CharLimit = 120` plus `strings.TrimSpace` on the gathered value bound the attack surface to ≤120 chars of arbitrary unicode, post-trim.** No Section-0-header sanitization, no argv-pattern stripping, no role-confusion filter. Per pre-MVP memory `feedback_prompt_injection_team.md`, sanitization is a render-layer concern for *attacker-controllable action-item content* — the `till init` TUI input is operator-controlled at the local CLI, not attacker-controllable until the team-aware architecture lands and project names become published artifacts. Defense-in-depth name sanitization is a D5/D7/post-team-arch concern. *Verdict on this attack: REFUTED for D4 — pre-MVP threat model places this outside D4's acceptance bar. Routing: pre-MVP refinement (orchestrator-routed at team-arch implementation) to add a name-sanitization pass at the `validateInitPayload` layer covering both TUI and JSON paths.*

### Attack family table

| Family                          | Result    | Notes                                                                                     |
| ------------------------------- | --------- | ----------------------------------------------------------------------------------------- |
| B1 counterexample-search        | CONFIRMED | W2-D4-FF1 — `mage ci` red on `fsatomic 64.0%`. Not D4's fault (D1 inheritance). |
| B2 contract-preservation        | REFUTED   | Findings 1.3 / 1.5 / 1.6 — value-receiver convention OK; default-name edge cases per spec; TUI/JSON validator asymmetry is D5 forward-dep. |
| B3 hidden-coupling              | REFUTED   | Findings 1.1 / 1.4 — Ctrl-C handled by framework default; D7.5 value-capture dormant in D4. |
| B4 yagni                        | REFUTED   | Findings 1.2 / 1.8 — defense-in-depth Enter guard + Init nil both justified.        |
| B5 file-package-gating          | REFUTED   | D4 touched only declared paths (`cmd/till/init_cmd.go`, `cmd/till/init_cmd_test.go`) plus conventional worklog + PLAN state-flip. |
| B6 shipped-but-not-wired        | REFUTED   | Finding 1.7 — production `programFactory` is real. Tests use type-asserting stub. |
| B7 prompt-injection             | REFUTED   | Finding 1.9 — pre-MVP threat model places sanitization outside D4 bar.              |

### Probes executed

- **`mage ci`** — RED. `Coverage threshold not met` → `internal/fsatomic 64.0%`. Stack-trace from `magefile.go` coverage gate. This is the W2-D4-FF1 finding, cause traced to D1.
- **`mage test-pkg ./cmd/till`** — GREEN, 268/268 tests pass. D4-specific package is sound.
- **Hylla `hylla_node_full` on `programFactory`** — confirmed production body is `tea.NewProgram(m)`.
- **Hylla `hylla_node_full` on `program` interface** — confirmed shape `Run() (tea.Model, error)`.
- **Hylla `hylla_node_full` on `scriptedProgram` + `scriptedProgram.Run`** — confirmed test stub body: `if p.runFn == nil { return p.model, nil }; return p.runFn(p.model)`.
- **Context7 `/charmbracelet/bubbletea`** — confirmed canonical Ctrl-C convention (model binds Ctrl-C; default signal handler also intercepts SIGINT and returns `tea.ErrProgramKilled`).
- **Direct `Read` of `cmd/till/init_cmd.go` lines 1-400** — full file inspection; confirmed receiver type, helper logic, Enter guard, View output.
- **Direct `Read` of `cmd/till/init_cmd_test.go` lines 1-297** — three new tea-tests + one rewritten cobra-end-to-end smoke test confirmed.
- **Direct `Read` of `workflow/drop_4c_6/DROP_4c.6.W2_TILL_INIT/PLAN.md` lines 126-141** — D4 acceptance criteria.
- **Direct `Read` of `workflow/drop_4c_6/BUILDER_WORKLOG.md` lines 2077-2156** — builder design rationale and TDD red→green trace.

### Summary

**Verdict: FAIL — 1 CONFIRMED counterexample (W2-D4-FF1) blocks droplet completion, but the cause is sibling-droplet D1 and NOT D4's implementation.** D4's bubbletea walk is sound under every attack vector probed (Ctrl-C / Esc handling, cursor traversal with disabled rows, value-receiver Update propagation, programFactory production wiring, default-name edge cases, no goroutine leaks, no D7.5 latent-bug activation, no path-gating violations, defense-in-depth Enter guard correctly justified). The seven attack families return: B1 CONFIRMED (sibling-caused), B2/B3/B4/B5/B6/B7 REFUTED. `mage test-pkg ./cmd/till` is GREEN at 268/268. The drop-level `mage ci` red is the same W2-D1-FF1 already pinned in this file at lines 1700-1707 with the round-2 fix-hint already routed.

**D4-side action: none — implementation passes every probed attack.** **Drop-side action: respawn W2.D1 builder for round-2 per the W2-D1-FF1 fix hint** (add `TestWriteFile_RenameFailsWhenTargetIsDirectory` to lift `fsatomic` coverage above 70%); once `mage ci` is green, this droplet's verdict converts to PASS without re-running D4 falsification.

**Forward-routing to D5 / D7** pinned in Findings 1.3 (validator asymmetry between TUI and JSON branches), 1.4 (D7.5 value-capture activation when D5 reads `opts.appName` / `opts.homeDir`), 1.6 (project-name validation for `/` / `.` / unicode edge cases), and 1.9 (post-team-arch name sanitization). None of those are D4 violations; all are forward-dependencies the orchestrator should pre-stage in the D5+ planner notes.

### Hylla Feedback

- **Query:** `hylla_search_keyword` for `programFactory scriptedProgram` against `github.com/evanmschultz/tillsyn@main`, `fields=["content"]`.
  - **Missed because:** First call returned zero results — the index treats `programFactory` and `scriptedProgram` as separate tokens and the conjunctive search returned the intersection (empty). The single-symbol query for `programFactory` worked on retry.
  - **Worked via:** Re-issued as a single-token `hylla_search_keyword` for `programFactory` with `visibility_mode: include_private` and `id_search_mode: tail_symbol` — got 1 production hit (`cmd/till/main.go:programFactory`) plus its callers. Same retry pattern for `scriptedProgram` (test-side stub).
  - **Suggestion:** the keyword tool's space-separated query semantics could expose an OR-mode toggle (or document the current AND-conjunctive default explicitly) so callers don't have to know to split multi-symbol queries into separate calls. Today the failure is silent (empty results), not surfaced as "no node contains BOTH terms; try OR-mode."
- **Query:** `hylla_search` for `programFactory program type bubbletea` with `fields: ["content", "summary", "docstring"]`.
  - **Missed because:** `hylla_search` rejected the `fields` array with the literal error `field must be summary, content, or docstring` — the parameter expects a singular `field` (not `fields`) string, NOT an array. The keyword variant accepts the plural `fields` array, so the asymmetry between `hylla_search` (singular `field`) and `hylla_search_keyword` (plural `fields`) caused the misfire.
  - **Worked via:** Fell back to `hylla_search_keyword` (which does take `fields` plural) plus `hylla_node_full` once the keyword hit named the node ID.
  - **Suggestion:** harmonize the parameter naming between `hylla_search` and `hylla_search_keyword` — either both take singular `field`, or both take plural `fields`. The current asymmetry is a silent footgun when alternating between the two within a single review.
- **Snapshot staleness for in-flight work:** `programFactory` (committed in commit `66c354ea` per Hylla's `commit_membership`) was indexed at snapshot 5, but the D4 builder's *uncommitted* edits to `init_cmd.go` (the `initTUIModel`, `Update`, etc.) are NOT in Hylla. For those, `Read` of the uncommitted file was the only viable evidence source. This is the same "enrichment still running" pattern flagged in the D3b + D4 builder Hylla Feedback entries. No new ergonomic ask — same pre-cascade structural staleness window already on the drop's refinement list.

---

## Droplet 4c.6.W3.D4 — Round 1

**QA Agent:** go-qa-falsification-agent (subagent).
**Date:** 2026-05-11.
**Droplet:** `4c.6.W3.D4 — Defense-in-depth env vars in cli_claude/env.go`.
**Files under review:** `internal/app/dispatcher/cli_claude/env.go`, `internal/app/dispatcher/cli_claude/adapter_test.go`.
**Builder round under review:** BUILDER_WORKLOG.md § "Droplet 4c.6.W3.D4 — Round 1" (2026-05-11).

### Attack surface enumerated

Attacks framed against the brief's 8 vectors plus 3 self-attack additions (duplicate name in `binding.Env`, `bindingOnly` capacity drift, ErrMissingRequiredEnv on unset overrider). Twelve findings recorded; one with a scope carve-out.

### Findings

#### W3-D4-FF1 [HIGH / injection order — slice-builder produces wrong final order] — REFUTED

- **Attack:** the brief flags injection-order risk explicitly. Defense-in-depth is injected AFTER `binding.Env` loop, BEFORE closed-baseline. The slice-builder must emit in declaration order. Confirm the final `out` slice is exactly `[baseline (those set), defense-in-depth (all 4), binding-only sorted]` and no other order can leak through.
- **Method:** Read `env.go:168-206` (three-loop slice-builder). Trace each emission per case.
- **Trace:** Three sequential `for` loops over (a) `closedBaselineEnvNames` (declaration order, lines 176-183), (b) `defenseInDepthEnvLiterals` (declaration order, lines 184-194), (c) `bindingOnly` (sorted, lines 196-206). Each loop guards via the shared `seen` map so a name emitted in an earlier loop is skipped in later loops. The baseline loop emits its names in `closedBaselineEnvNames` declaration order; defense-in-depth loop walks `defenseInDepthEnvLiterals` in declaration order (CLAUDE_CODE_DISABLE_BACKGROUND_TASKS, CLAUDE_CODE_FORK_SUBAGENT, DISABLE_AUTOUPDATER, DISABLE_TELEMETRY); binding-only loop walks `bindingOnly` after `sort.Strings`. Order is deterministic and matches the documented contract.
- **Result — REFUTED.** Order is correct and deterministic by construction.

#### W3-D4-FF2 [HIGH / precedence override — binding wins, defense literal must not leak] — REFUTED

- **Attack:** the plan locks `binding.Env > defense-in-depth > closed-baseline`. With `binding.Env: ["DISABLE_TELEMETRY"]` + `t.Setenv("DISABLE_TELEMETRY", "0")`, confirm cmd.Env carries exactly `DISABLE_TELEMETRY=0` AND NEVER `DISABLE_TELEMETRY=1`.
- **Method:** Trace `assembleEnv` over the override case. Read `TestEnvDefenseInDepthOverridableByBindingEnv` (`adapter_test.go:751-785`).
- **Trace:** binding loop sets `emitted["DISABLE_TELEMETRY"]="0"`. Defense literal loop at lines 145-150 sees `alreadySet` → SKIP — `emitted["DISABLE_TELEMETRY"]` stays at "0", the `1` literal is NEVER written. Closed-baseline loop: DISABLE_TELEMETRY is not in baseline, untouched. Slice-builder: baseline names walked (DISABLE_TELEMETRY not present). Defense literal walk at lines 184-194: lit.Name="DISABLE_TELEMETRY", not in `seen` (only baseline names seen), `val, ok := emitted[lit.Name]` → ok=true, val="0" → emits `DISABLE_TELEMETRY=0`, marks `seen[DISABLE_TELEMETRY]`. Binding-only loop: DISABLE_TELEMETRY now in `seen` → SKIP. Final cmd.Env has exactly one DISABLE_TELEMETRY entry with value "0". Test confirms via `slices.Contains(cmd.Env, "DISABLE_TELEMETRY=0")` AND `!slices.Contains(cmd.Env, "DISABLE_TELEMETRY=1")`. Builder GREEN.
- **Result — REFUTED.** Single-emission semantics hold; binding wins; defense literal does NOT leak alongside.

#### W3-D4-FF3 [HIGH / empty binding — all 4 defense literals appear] — REFUTED

- **Attack:** with `binding.Env = nil` (the minimal binding case), do all four defense literals appear unconditionally — even when the orchestrator's env has NONE of them set?
- **Method:** Trace `assembleEnv` over the empty-binding + all-defense-vars-unset case. Read `TestEnvCarriesDefenseInDepthEnvVars` (`adapter_test.go:704-744`).
- **Trace:** binding.Env = nil → binding loop body never executes; `emitted` is empty after the loop. Defense literal loop walks all 4; for each: `alreadySet` false; `emitted[lit.Name] = lit.Value` writes the inline value (NO `os.LookupEnv` call — values are literal). After the loop, all 4 defense names are in `emitted` with their literal values. Closed-baseline loop: 4 names disjoint from baseline (verified: defense names are CLAUDE_CODE_*/DISABLE_*; baseline names are PATH/HOME/USER/LANG/LC_ALL/TZ/TMPDIR/XDG_*/HTTP_*/HTTPS_*/NO_*/http_*/https_*/no_*/SSL_*/CURL_CA_BUNDLE — zero overlap). Slice-builder: baseline emits whatever baseline names are set; defense walk emits all 4 (none in `seen` because they're not baseline); binding-only is empty. Test pre-`Unsetenv`s all four defense names AND asserts via `slices.Contains(cmd.Env, "<defense>=<value>")` for each — proves values come from the inline pairs, not `os.LookupEnv`.
- **Result — REFUTED.** All 4 defense literals are emitted unconditionally on empty binding + all-defense-unset.

#### W3-D4-FF4 [HIGH / partial override — non-overridden 3 still emit defense values] — REFUTED

- **Attack:** the brief flags "override-only-some" explicitly. With `binding.Env: ["DISABLE_AUTOUPDATER"]` + orchestrator value, the other 3 defense literals (CLAUDE_CODE_DISABLE_BACKGROUND_TASKS, CLAUDE_CODE_FORK_SUBAGENT, DISABLE_TELEMETRY) must still appear at their DEFAULT literal values (1, 0, 1).
- **Method:** Read `TestEnvDefenseInDepthOverridableByBindingEnv` (`adapter_test.go:776-784`). The test asserts BOTH the binding-overridden literal AND the non-overridden 3 still emit at their default values.
- **Trace:** Test sets `binding.Env: ["DISABLE_TELEMETRY"]` (overrides one) + `t.Setenv("DISABLE_TELEMETRY", "0")`. After `BuildCommand`: asserts `slices.Contains(cmd.Env, "CLAUDE_CODE_DISABLE_BACKGROUND_TASKS=1")`, `"CLAUDE_CODE_FORK_SUBAGENT=0"`, `"DISABLE_AUTOUPDATER=1"`. Each of the three non-overridden literals' independent emission proves: defense literal loop only `continue`s when the SPECIFIC `lit.Name` is in `emitted` (binding-overridden case), NOT when ANY overrider exists. Each non-overridden literal is processed independently; the `alreadySet` check is scoped per-literal.
- **Result — REFUTED.** Per-literal precedence is independent; partial overrides do not cascade-skip the other defense literals.

#### W3-D4-FF5 [HIGH / closed-baseline overlap — defense names collide with baseline] — REFUTED

- **Attack:** the brief flags "are any of the 4 defense-in-depth names ALREADY in `closedBaselineEnvNames`?" Confirm zero overlap; if any overlap existed, the dedup behavior would be: defense literal loop writes the literal value FIRST, baseline loop's `alreadySet` check would SKIP, so the literal value would win. But the literal would still be emitted in the DEFENSE-LITERAL section of the slice-builder, not the baseline section — potentially shifting position.
- **Method:** Compare the two slice declarations at `env.go:41-53` (defense, 4 names) vs `env.go:73-94` (baseline, 18 names).
- **Trace:** Defense literals: CLAUDE_CODE_DISABLE_BACKGROUND_TASKS, CLAUDE_CODE_FORK_SUBAGENT, DISABLE_AUTOUPDATER, DISABLE_TELEMETRY. Baseline: PATH, HOME, USER, LANG, LC_ALL, TZ, TMPDIR, XDG_CONFIG_HOME, XDG_CACHE_HOME, HTTP_PROXY, HTTPS_PROXY, NO_PROXY, http_proxy, https_proxy, no_proxy, SSL_CERT_FILE, SSL_CERT_DIR, CURL_CA_BUNDLE. Zero intersection. The doc-comment at `env.go:154-157` explicitly notes "today's two sets are disjoint" — accurate.
- **Result — REFUTED.** Zero overlap; the question of overlap-dedup is moot today. Future addition of an overlapping name would be caught at code review by the doc-comment's "disjoint" assertion.

#### W3-D4-FF6 [MEDIUM / slices import — stdlib vs x/exp drift] — REFUTED

- **Attack:** the brief flags "is `slices` imported from `slices` (stdlib) or `golang.org/x/exp/slices`?". The post-Go-1.21 stdlib has `slices.Contains`; pre-1.21 code used `golang.org/x/exp/slices`. The wrong import path means either: (a) `go.mod` is required to be on 1.21+ for stdlib, or (b) the test pulls in a vendored shim.
- **Method:** Read `adapter_test.go:1-18` (imports). Cross-check go.mod-version constraint.
- **Trace:** `adapter_test.go:12` carries `"slices"` — stdlib path. Verified by mage ci passing all `cli_claude` tests including the two new defense-in-depth tests that both use `slices.Contains`. Stdlib `slices.Contains` has been available since Go 1.21 (Aug 2023); project go.mod is 1.26+ per CLAUDE.md Tech Stack section. Compatible.
- **Result — REFUTED.** Import is stdlib; no drift; build proves it resolves.

#### W3-D4-FF7 [MEDIUM / test fixture realism — degenerate binding bypasses closed-baseline loop] — REFUTED

- **Attack:** the brief flags "does the test build a realistic `BindingResolved` or a degenerate one that bypasses the closed-baseline loop?" If the test uses `binding.Env = nil` + no `t.Setenv` for baseline names, the closed-baseline loop's `os.LookupEnv` returns `false` for every baseline name → all skipped → degenerate cmd.Env. The defense literals would still emit (unconditional), but the assertion `slices.Contains(cmd.Env, "DISABLE_TELEMETRY=1")` would PASS even on a degenerate cmd.Env that omits every baseline name. Is the test proving too little?
- **Method:** Read `TestEnvCarriesDefenseInDepthEnvVars` (`adapter_test.go:704-744`) vs `TestEnvNotInheritedFromOSEnviron` (`adapter_test.go:346-384`).
- **Trace:** TestEnvCarriesDefenseInDepthEnvVars deliberately uses a minimal binding to prove "defense literals appear independently of binding shape" — that's the test's documented purpose. Realism is supplied by the sibling `TestEnvNotInheritedFromOSEnviron` (line 346) which `t.Setenv`s every baseline name AND the binding-only name AND a sentinel outsider, then asserts `len(cmd.Env) == len(closedBaselineEnvNames) + len(defenseInDepthEnvLiterals) + 1` — proving the FULL slice shape is correct including defense literals + baseline + binding-only. The TWO tests together cover: (a) defense-literals-present-when-isolated (D4-FF3 mechanism); (b) full-slice-shape-correct-when-realistic (FF7 mechanism). The shape-test was specifically updated by this droplet (per worklog: "Adjusted `TestEnvNotInheritedFromOSEnviron` length assertion from `len(closedBaselineEnvNames) + 1` to `len(closedBaselineEnvNames) + len(defenseInDepthEnvLiterals) + 1`").
- **Result — REFUTED.** Test coverage is realistic across siblings; degenerate-binding is the deliberate isolation strategy, NOT a coverage gap.

#### W3-D4-FF8 [MEDIUM / mage ci — full build/test gate] — REFUTED (with scope carve-out)

- **Attack:** the brief mandates `mage ci`. Verify GREEN.
- **Method:** Run `mage ci` from `/Users/evanschultz/Documents/Code/hylla/tillsyn/main/`.
- **Trace:** Output captured. `3077 tests passed across 26 packages.` `cli_claude` package: GREEN at 95.3% coverage. `cli_claude/render` package: GREEN at 82.8% coverage. ALL packages pass tests. Coverage gate fails on a SINGLE package — `internal/fsatomic` at 64.0% (threshold is 70%). That package is UNTRACKED in `git status` (`?? Untracked: 1 files / internal/fsatomic/`), is a brand-new sibling-droplet (W2.D1 — explicitly owned by `4c.6.W2.D1 — internal/fsatomic/ atomic file-write helper` per the prior round at BUILDER_QA_FALSIFICATION.md:1623–1711, finding W2-D1-FF1 already CONFIRMED), and is COMPLETELY OUT OF SCOPE for W3.D4. W3.D4's scope per PLAN.md is exactly two files: `env.go` + `adapter_test.go`. The fsatomic coverage gap is sibling-owned and already flagged.
- **Result — REFUTED for W3.D4 scope.** The fsatomic coverage failure is NOT a W3.D4 falsification — sibling-droplet W2.D1 owns it (W2-D1-FF1 already raised in this same file). W3.D4's changed packages are above threshold.

```
[PKG PASS] github.com/evanmschultz/tillsyn/internal/app/dispatcher/cli_claude (0.01s)
[PKG PASS] github.com/evanmschultz/tillsyn/internal/app/dispatcher/cli_claude/render (0.01s)
Test summary
  tests: 3077  passed: 3077  failed: 0  skipped: 0
  packages: 26  pkg passed: 26  pkg failed: 0

cli_claude        95.3%  ← W3.D4 scope
cli_claude/render 82.8%
fsatomic          64.0%  ← OUT OF SCOPE for W3.D4 (sibling W2.D1, already W2-D1-FF1)
```

#### W3-D4-FF9 [LOW / duplicate name in binding.Env — same name listed twice] — REFUTED

- **Attack:** self-attack. If `binding.Env = ["FOO", "FOO"]`, the binding loop runs twice. First iteration `emitted["FOO"] = val`; second iteration `emitted["FOO"] = val` (same). The `bindingNames` map dedup-sets `FOO`. The `bindingOnly` slice is built from `bindingNames` keys, so it carries `FOO` once. Final cmd.Env has one `FOO=val` entry. Defensible.
- **Method:** Trace `assembleEnv` over `binding.Env = ["DISABLE_TELEMETRY", "DISABLE_TELEMETRY"]`.
- **Trace:** Both iterations call `os.LookupEnv("DISABLE_TELEMETRY")` — both return the same `val`. `emitted[DISABLE_TELEMETRY] = val` is set twice (idempotent map write). `bindingNames["DISABLE_TELEMETRY"] = struct{}{}` set twice (idempotent). Defense loop sees `alreadySet` (emitted has the key) → SKIP. Slice-builder: defense walk emits one `DISABLE_TELEMETRY=val` entry, marks `seen[DISABLE_TELEMETRY]`. Binding-only loop's range over `bindingNames` yields `DISABLE_TELEMETRY` once → already in `seen` → SKIP. Single emission. No duplicate.
- **Result — REFUTED.** Map-based dedup handles binding-Env duplicates correctly.

#### W3-D4-FF10 [LOW / `bindingOnly` capacity drift — over-allocation when binding-Env overlaps with baseline + defense] — REFUTED

- **Attack:** self-attack. `bindingOnly := make([]string, 0, len(binding.Env))` (line 196). If `binding.Env = ["PATH", "HOME", "DISABLE_TELEMETRY"]` (all overlap with baseline or defense), `bindingOnly` would end up empty but the capacity is 3 (waste, not bug).
- **Method:** Read line 196 + the binding-only loop.
- **Trace:** Capacity is upper bound; actual emission is gated by `_, alreadyEmitted := seen[name]` check. Over-allocation is benign — a few words of heap waste. NOT a correctness issue.
- **Result — REFUTED.** Capacity hint is conservative-but-correct; not a defect.

#### W3-D4-FF11 [LOW / overrider that's unset triggers ErrMissingRequiredEnv] — REFUTED

- **Attack:** self-attack. If `binding.Env: ["DISABLE_TELEMETRY"]` but the orchestrator has NOT set DISABLE_TELEMETRY in its own env, what happens? Does the spawn fall back to the defense literal value? Does it crash? Does the user see a confusing error?
- **Method:** Trace `assembleEnv` over the unset-overrider case.
- **Trace:** binding loop calls `os.LookupEnv("DISABLE_TELEMETRY")` → `("", false)` → `return nil, fmt.Errorf("%w: name=%q", ErrMissingRequiredEnv, name)`. The spawn fails pre-lock per F.7.17 P5. The default defense literal `DISABLE_TELEMETRY=1` is NOT emitted as a fallback — the spawn is aborted entirely. This is the documented contract: any binding-Env name that's unset in the orchestrator fails loud. A binding that explicitly lists a defense name takes responsibility for ensuring the orchestrator has it set.
- **Result — REFUTED.** Documented contract; not a regression. Adopters who want to override a defense literal via binding MUST set it in the orchestrator's env first. The error message is clear and names the missing var.

#### W3-D4-FF12 [LOW / doc-comment accuracy — `binding.Env > defense-in-depth > closed-baseline` claim] — REFUTED

- **Attack:** the doc-comment at `env.go:31-36` claims "Net precedence: binding.Env > defense-in-depth > closed-baseline". Verify by tracing each pairing.
- **Method:** Three pairings: (a) binding vs defense; (b) defense vs baseline; (c) binding vs baseline.
- **Trace:** (a) binding vs defense — binding loop runs FIRST, populates `emitted[name]`; defense loop's `alreadySet` check SKIPs the literal. ✓ binding > defense. (b) defense vs baseline — defense loop runs SECOND, populates `emitted[lit.Name]`; baseline loop's `alreadySet` check SKIPs. ✓ defense > baseline. (c) binding vs baseline — established pre-D4; binding loop runs FIRST; baseline loop's `alreadySet` check SKIPs. ✓ binding > baseline. Transitive ordering holds.
- **Result — REFUTED.** Precedence claim is accurate per the three-loop construction.

### Severity breakdown

- **HIGH:** 5 (FF1 / FF2 / FF3 / FF4 / FF5) — all REFUTED.
- **MEDIUM:** 3 (FF6 / FF7 / FF8) — all REFUTED (FF8 with explicit scope carve-out for the unrelated `fsatomic` coverage gap already owned by sibling W2.D1).
- **LOW:** 4 (FF9 / FF10 / FF11 / FF12) — all REFUTED.

Twelve attacks; zero CONFIRMED counterexamples against W3.D4.

### mage ci result

`mage ci` returned non-zero (exit on coverage gate for `internal/fsatomic` at 64.0%). HOWEVER:

- **Tests**: 3077 passed / 0 failed across 26 packages.
- **W3.D4 scoped packages**: `cli_claude` GREEN at 95.3%; `cli_claude/render` GREEN at 82.8%.
- **Coverage failure cause**: `internal/fsatomic` — sibling W2.D1's package (per W2-D1-FF1 in this same file at line 1632). NOT W3.D4 scope (W3.D4 Paths per PLAN.md = `env.go` + `adapter_test.go` only).
- **Verdict**: W3.D4's changes pass all tests and exceed coverage threshold for the changed packages. The fsatomic gap is owned by sibling W2.D1.

### Summary

**Verdict: PASS — no CONFIRMED counterexamples against W3.D4.** All 12 attacks REFUTED. The defense-in-depth env injection in `cli_claude/env.go` is shaped correctly:

- Slice-builder produces the documented order (baseline → defense → binding-only-sorted) with `seen`-map dedup preventing double emission.
- Precedence chain `binding.Env > defense-in-depth > closed-baseline` holds across all three pairings, verified by trace and by `TestEnvDefenseInDepthOverridableByBindingEnv`.
- Empty binding emits all 4 defense literals unconditionally; partial override leaves the other 3 at default literal values; unset-but-explicitly-overrider-named triggers ErrMissingRequiredEnv per documented contract.
- Closed-baseline names and defense-in-depth names are disjoint (verified by enumeration); the disjoint-claim is annotated in code.
- `slices` is stdlib (Go 1.21+); project go.mod is 1.26+.
- Test fixture realism is split across two sibling tests: `TestEnvCarriesDefenseInDepthEnvVars` (defense-isolation) + `TestEnvNotInheritedFromOSEnviron` (full-slice shape with +4 length adjustment).
- All 3077 tests pass; `cli_claude` package coverage is 95.3%. The unrelated `internal/fsatomic` coverage gap is sibling-scope (already raised as W2-D1-FF1).

No routing back to orchestrator. W3.D4 closure status: CLEAR.

### Hylla Feedback

N/A — W3.D4 touched only post-commit-stage Go files (`env.go`, `adapter_test.go`) whose changes are NOT in Hylla snapshot 5 (uncommitted; `git status` shows both as Modified). Two attempted queries via `hylla_search_keyword` for `closedBaselineEnvNames` and `defenseInDepth DISABLE_TELEMETRY DISABLE_AUTOUPDATER` against `github.com/evanmschultz/tillsyn@main` both returned empty results — expected because the staleness window is structurally guaranteed pre-cascade-ingest. The falsification relied entirely on `Read` against the two target files + the PLAN.md W3.D4 row + adapter.go for the BuildCommand wiring + `git status` for the scope-carve-out evidence on `internal/fsatomic`. No fallback miss to record beyond the documented pre-ingest staleness window — same finding sibling droplets in this drop have flagged repeatedly. Suggestion already in play per prior rounds: expose last-fully-ingested snapshot ID + "partial index available" hint.

---

## Droplet 4c.6.W3.D5 — Round 1

**Reviewer:** go-qa-falsification-agent (subagent, doc-only mode).
**Date:** 2026-05-11.
**Droplet:** `4c.6.W3.D5 — Post-render validator wired at Render's exit + sentinel test`.
**Artifact under attack:** `internal/app/dispatcher/cli_claude/render/render.go` (validator additions at lines 110-137, 294-297, 302-392) + `internal/app/dispatcher/cli_claude/render/render_test.go` (5 new top-level validator tests + WalkDir placeholder positive-coverage sub-test + `validatorConformingBodySuffix()` helper + 5 pre-existing W3.D2/W3.D3 test-fixture migrations).

### Counterexamples

(none — empty list. All 11 attack vectors REFUTED. The droplet-internal claims hold under static + dynamic probing. See Findings 1.x for documented NIT / accepted-carryforward observations. The cross-droplet `mage ci` coverage failure on `internal/fsatomic` is REFUTED-as-NOT-W3.D5 — sibling W2.D1 owns it via prior W2-D1-FF1 finding.)

### Findings (non-CONFIRMED, recorded for audit)

#### W3-D5-FF1 [NIT / Signal C unanchored substring — round-3 W3-FF13 accepted carryforward] — REFUTED-AS-DESIGN

- **Attack:** the brief flags `Signal C false-positive risk (per W3-FF13 accepted NIT): body that QUOTES \`# Section 0\` inside backticks/code-fence — would falsely pass`.
- **Method:** Static analysis of `render.go:386` Signal C check + cross-reference `PLAN.md:303-307` round-3-accepted NIT documentation + `render.go:380-383` in-code doc-comment.
- **Trace:** `render.go:386`: `if strings.Contains(postFrontmatter, m)` — substring match per the W3-FF13 NIT accepted-by-design. A stub body that quotes `# Section 0` (or `# PLACEHOLDER` or `## Role`) inside backticks / code-fence / prose would satisfy Signal C falsely. Builder explicitly documents this at `render.go:380-383`: "a stub that DELIBERATELY quotes a marker inside backticks would pass Signal C falsely, but the realistic-stub case (no marker at all) is caught here, and Signal A + Signal B catch the common-case stub shapes regardless." `BUILDER_WORKLOG.md` Round-1 D5 entry line 26 confirms the substring choice was intentional ("line-anchored matching was discretionary and the substring form keeps the validator code minimal").
- **Result — REFUTED-AS-DESIGN.** W3-FF13 carryforward accepted-by-design; PLAN.md round-3-accepted NIT (lines 303-307) explicitly says "D5 builder has discretion to upgrade to line-anchored matching if implementation reads cleaner. Post-build refinement candidate." Not a new finding — recorded for audit only.

#### W3-D5-FF2 [NIT / Signal B `name:` substring false-positive — NEW] — REFUTED-AS-LOW-RISK

- **Attack:** Signal B's frontmatter validation uses `strings.Contains(frontmatter, "name:")` (`render.go:369`). Construct a counterexample frontmatter that contains a substring like `username:`, `realname:`, or `surname:` but no actual `name:` field.
- **Method:** Static analysis of `render.go:368-371` (Signal B) + reasoning about the substring `name:` against tokens ending in `name:`.
- **Trace:** A frontmatter body of `---\nusername: foo\n---\n` would satisfy the Signal B check because `strings.Contains("username: foo\n", "name:")` returns true (the substring `name:` appears inside `username:`). Combined with: Signal A satisfied via 200+ chars of filler post-frontmatter, Signal C satisfied via `# PLACEHOLDER`, this synthetic body would FALSELY pass the validator despite lacking a real `name:` field. Production-path realism: low — binding name validation happens upstream at `validateAgentBindingNames` (`internal/templates/load.go:1031-1055` per `RESEARCH/ISOLATION_ENFORCEMENT_FIX.md` § D.1.b reference); the resolver doesn't surface `username:`-without-`name:` frontmatter as a normal failure mode. Plus a realistic typo like `nme:` or capital-N `Name:` still trips the check (capital-N `Name:` does NOT contain lowercase `name:`).
- **Result — REFUTED-AS-LOW-RISK.** The false-positive surface is contrived; no production path lands `username:`-without-`name:` frontmatter. Fix hint (post-MVP refinement candidate): line-anchored match — `for _, line := range strings.Split(frontmatter, "\n") { if strings.HasPrefix(strings.TrimSpace(line), "name:") { return ok } }`. Stage alongside W3-FF13 for a future validator-tightening drop. NOT a current bug; NOT blocking W3.D5.

#### W3-D5-FF3 [INFO / Signal A boundary 200 vs 201 strict-greater] — REFUTED

- **Attack:** the brief flags `Signal A boundary: ">200" strict-greater. What about exactly 200 chars? 201? Off-by-one?`.
- **Method:** Static analysis of `render.go:373` + PLAN.md `body length > 200 chars` (line 232) + builder design rationale in worklog line 25.
- **Trace:** `render.go:373`: `if n := len(postFrontmatter); n <= minBodyLength` where `minBodyLength = 200`. Boundary cases by static reasoning:
  - `n=199`: `199 <= 200` is true → FAILS (correct — 199 < 200).
  - `n=200`: `200 <= 200` is true → FAILS (correct — strict `>` interpretation per builder design "A body of exactly 200 chars fails. This is consistent with the W3-FF8 W4-floor-as-forward-dep wording: substantive prompts MUST clear the floor, equality at the floor is 'did not clear.'").
  - `n=201`: `201 <= 200` is false → PASSES (correct).
  PLAN.md acceptance bullet wording "body length > 200 chars" interpreted as strict-greater. Builder's design rationale (worklog line 25) is internally consistent.
- **Result — REFUTED.** No off-by-one; boundary semantics align with documented strict-greater intent.

#### W3-D5-FF4 [INFO / Disk-read claim verification] — REFUTED

- **Attack:** the brief flags `Disk-read vs in-memory: builder said validator reads from disk (not in-memory body). Verify. What if disk read fails between assemble and validate?`.
- **Method:** Read `render.go:326-336` for `validateBundle` body.
- **Trace:** `render.go:327-328`: `agentPath := filepath.Join(bundle.Paths.Root, pluginSubdir, agentsSubdir, binding.AgentName+".md")` then `body, err := os.ReadFile(agentPath)`. Confirms disk read, not in-memory body. Builder claim in `BUILDER_WORKLOG.md` line 27 ("the disk read happens in `validateBundle`") and design rationale ("(a) reading from disk catches a future regression where `os.WriteFile` silently truncates the file post-write (the in-memory body wouldn't catch that)") match the shipped code verbatim. Disk-read failure path at line 329-334: returns `fmt.Errorf("%w: read rendered agent file %q: %s", ErrInvalidAgentBody, agentPath, err.Error())` — wraps under `ErrInvalidAgentBody`. The classification quirk (an actual I/O error gets classified under `ErrInvalidAgentBody` rather than a separate I/O sentinel) is documented at line 330-331 ("If renderAgentFile succeeded just above, the file MUST exist; any read error here is a real I/O failure worth surfacing") and is intentional — the failure mode is unreachable in normal operation because renderAgentFile-then-validateBundle is a tight call sequence with no intervening writes.
- **Result — REFUTED.** Disk read confirmed at the documented seam; failure-path wrap is documented and intentional. Micro-edge logged as Finding 1.11-style refinement candidate (separate `ErrAgentBodyReadFailure` sentinel) but not a current bug.

#### W3-D5-FF5 [INFO / Wrap chain — errors.Is end-to-end] — REFUTED

- **Attack:** the brief flags `Wrap chain: errors.Is(err, ErrInvalidAgentBody) works through the %w wrap? Verify each signal's failure path`.
- **Method:** Read each `fmt.Errorf` site inside `validateAgentBodyShape` + `validateBundle` + `Render`.
- **Trace:** Five signal failure sites inside `validateAgentBodyShape`:
  - Line 359 (missing leading delimiter): `fmt.Errorf("%w: missing leading \`---\\n\` frontmatter delimiter", ErrInvalidAgentBody)`.
  - Line 364 (missing closing delimiter): `fmt.Errorf("%w: missing closing \`---\\n\` frontmatter delimiter", ErrInvalidAgentBody)`.
  - Line 370 (missing `name:` field): `fmt.Errorf("%w: frontmatter missing \`name:\` field", ErrInvalidAgentBody)`.
  - Line 374 (Signal A): `fmt.Errorf("%w: body length %d <= %d (post-frontmatter floor)", ErrInvalidAgentBody, n, minBodyLength)`.
  - Line 390 (Signal C): `fmt.Errorf("%w: body missing positive role-section marker (need one of %v)", ErrInvalidAgentBody, markers)`.
  Plus `validateBundle` I/O failure at line 332-333: `fmt.Errorf("%w: read rendered agent file %q: %s", ErrInvalidAgentBody, agentPath, err.Error())`. All six use `%w` against `ErrInvalidAgentBody`. Then `Render` at line 296: `fmt.Errorf("render: validate bundle: %w", err)` — double-wraps. Per stdlib `errors` semantics, `errors.Is` walks the `%w` chain. Cross-evidence: shipped test `TestRenderValidatorFailsOnTooShortBody` at `render_test.go:1458-1489` asserts `errors.Is(err, render.ErrInvalidAgentBody)` against the doubly-wrapped error AND passes per `mage testPkg ./internal/app/dispatcher/cli_claude/render` (70/70 green).
- **Result — REFUTED.** Wrap chain holds end-to-end through both layers.

#### W3-D5-FF6 [INFO / Rollback on validator failure — partial-state leak] — REFUTED

- **Attack:** the brief flags `Rollback integration: when validator fails, does the existing renderRollback (render.go:188-204) clean up properly? Or is there a partial-state leak?`.
- **Method:** Read `render.go:294-297` (Render wiring) + `render.go:401-420` (renderRollback struct + run method) + `render_test.go:1480-1488` (rollback assertion shipped in TestRenderValidatorFailsOnTooShortBody).
- **Trace:** Validator-failure wiring at `render.go:294-297`: `if err := validateBundle(bundle, binding); err != nil { rollback.run(); return "", fmt.Errorf("render: validate bundle: %w", err) }`. Rollback.run at `render.go:414-420`: `_ = os.Remove(filepath.Join(r.bundleRoot, "system-prompt.md"))` + `_ = os.RemoveAll(filepath.Join(r.bundleRoot, pluginSubdir))`. The `RemoveAll(plugin/)` blanket-removes every artifact written by:
  - `renderPluginManifest` (`<root>/plugin/.claude-plugin/plugin.json` per `render.go:507-518`).
  - `renderAgentFile` (`<root>/plugin/agents/<name>.md` per `render.go:559-569`).
  - `renderMCPConfig` (`<root>/plugin/.mcp.json` per `render.go:948-964`).
  - `renderSettings` (`<root>/plugin/settings.json` per `render.go:1019-1049`).
  Plus the additional `os.Remove("system-prompt.md")` covers the cross-CLI bundle-root artifact. Manifest.json at `<root>/manifest.json` (F.7.1 upstream) is deliberately NOT touched per `render.go:397-400` ("Render is the sole writer under <Root>/system-prompt.md and <Root>/plugin/, so a failed render can blanket-remove those two paths without touching F.7.1's manifest.json"). Test confirmation: `TestRenderValidatorFailsOnTooShortBody:1480-1488` asserts both `<root>/system-prompt.md` and `<root>/plugin` are gone post-failure via `errors.Is(statErr, os.ErrNotExist)`. No partial-state leak observable.
- **Result — REFUTED.** Rollback wipes every render-written artifact while preserving the upstream F.7.1 manifest as designed. Best-effort error swallowing is documented at `render.go:411-413` and is appropriate — the caller is already returning a non-nil error.

#### W3-D5-FF7 [INFO / Concurrent renders — race condition / shared state] — REFUTED

- **Attack:** the brief flags `Concurrent renders: if two Render calls run concurrently against different bundles, do they interfere?`.
- **Method:** Read `render.go:1-100` (imports + package-level declarations) + `render.go:234-300` (Render entry point) + `render_test.go:57-73` (fixtureBundle).
- **Trace:** Package-level state surface enumerated:
  - Constants: `pluginSubdir`, `claudePluginManifestSubdir`, `agentsSubdir`, `agentBodyEmbeddedRoot`, `agentBodyDefaultGroup`, `agentBodyFallbackGroup`, `projectAgentsSubdir`, `userAgentsSubdir` (`render.go:72-199`). All `const string` — immutable.
  - Sentinel errors: `ErrInvalidRenderInput`, `ErrInvalidGrantsLister`, `ErrAgentBodyNotFound`, `ErrInvalidAgentBody`, `ErrInvalidAgentTemplatePath` (`render.go:86-170`). All `var = errors.New(...)` — immutable at runtime.
  - No package-level `sync.Mutex`, `sync.Map`, channel, atomic, or mutable global. No `init()` function. No package-level mutable cache.
  - `Render` function-local state: `rollback` (renderRollback struct, local), `promptBody` (string, local), error returns (local). No shared mutation.
  - File system reads/writes: each `Render` call operates on its own `bundle.Paths.Root` (per-spawn temp dir per `fixtureBundle` line 59 `t.TempDir()`); two concurrent calls against different roots have disjoint write paths; the embed.FS (`templates.DefaultTemplateFS`) is read-only by compile-in invariant.
  - `os.UserHomeDir()` (`render.go:861`) is goroutine-safe per stdlib doc (reads process environment which is goroutine-safe for read).
  Race-free by construction.
- **Result — REFUTED.** No shared state; concurrent renders against disjoint bundle roots cannot interfere. The only shared resource — `templates.DefaultTemplateFS` — is read-only.

#### W3-D5-FF8 [INFO / 27/27 placeholder positive-coverage] — REFUTED

- **Attack:** the brief flags `Placeholder coverage: builder claimed 27/27 placeholders pass. Verify by re-running the positive test`.
- **Method:** Read `render_test.go:1592-1660` (TestRenderValidatorAcceptsAllEmbeddedPlaceholders) + `internal/templates/embed.go:75-103` (//go:embed directives) + run `mage testPkg ./internal/app/dispatcher/cli_claude/render`.
- **Trace:** Test walks `templates.DefaultTemplateFS` under `builtin/agents/` via `fs.WalkDir` (line 1600-1612), collects every `.md` file (line 1610), runs each as a project-tier override through `Render` (line 1647), asserts no error. Sanity gate at line 1657-1659 fails fast if `len(placeholders) < 27`. Reconciliation against `embed.go:77-103` `//go:embed` directives — 27 placeholder `.md` paths embedded:
  - 7 standard names under `till-gen/`: planning-agent, builder-agent, qa-proof-agent, qa-falsification-agent, research-agent, closeout-agent, commit-message-agent.
  - 7 standard names under `till-go/`: same 7 names.
  - 7 standard names under `till-gdd/`: same 7 names.
  - 5 legacy go-* names under `till-go/`: go-builder-agent, go-planning-agent, go-research-agent, go-qa-proof-agent, go-qa-falsification-agent (W5.D3 transitional residue per `embed.go:64-68`).
  - 1 `orchestrator-managed.md` under `till-gen/`.
  Total: 7+7+7+5+1 = 27. Matches the sanity-gate floor.
  Dynamic verification: `mage testPkg ./internal/app/dispatcher/cli_claude/render` returns 70/70 green including all 27 placeholder sub-tests (test count: 5 new validator tests + 27 WalkDir sub-tests + ~38 pre-existing render tests = 70). Test output captured: `[PKG PASS] github.com/evanmschultz/tillsyn/internal/app/dispatcher/cli_claude/render (0.00s)`.
- **Result — REFUTED.** 27/27 placeholders pass the validator end-to-end per live test execution.

#### W3-D5-FF9 [INFO / D2+D3 fixture migration preserves original assertions] — REFUTED

- **Attack:** the brief flags `D2+D3 fixture migration: builder claimed 5 pre-existing D2/D3 tests needed the validatorConformingBodySuffix() helper. Are those tests still asserting their original substrings (e.g., SENTINEL_USER_TIER)?`.
- **Method:** Read each of the 5 migrated tests and verify substring assertions are preserved.
- **Trace:** Five migrated tests audited:
  1. `TestAssembleAgentFileBody_UserOverride` (`render_test.go:893-920`) — assertion line 917 `if !strings.Contains(body, sentinel)` where `sentinel = "SENTINEL_USER_TIER"` (line 900). Fixture at line 902: `"---\nname: go-builder-agent\n---\n\n" + validatorConformingBodySuffix() + sentinel + "\n"`. Suffix BEFORE sentinel preserves the substring match.
  2. `TestAssembleAgentFileBody_ProjectOverride` (`render_test.go:925-961`) — assertion line 954 `if !strings.Contains(body, projectSentinel)` where `projectSentinel = "SENTINEL_PROJECT_TIER"` (line 940). Plus NEGATIVE assertion line 957 `if strings.Contains(body, "SENTINEL_USER_TIER")` (ensures user-tier sentinel NOT in body when project tier wins). Fixture at line 942: `"---\nname: go-builder-agent\n---\n\n" + validatorConformingBodySuffix() + projectSentinel + "\n"`. Suffix-before-sentinel pattern preserved.
  3. `TestAssembleAgentFileBody_FrontmatterStripModelOnAgentsTOMLSet` (`render_test.go:1075-1108`) — assertions: line 1097 `model:` absent (strip worked); line 1101 `body-bytes-preserve-marker` present; line 1105 `name: go-builder-agent` survives. Fixture uses `d3UserTierFrontmatter("name: go-builder-agent\nmodel: opus\n")` at line 1080 — helper definition at line 1066-1070 wraps `validatorConformingBodySuffix() + "body-bytes-preserve-marker\n"`. Marker substring assertion still passes.
  4. `TestAssembleAgentFileBody_FrontmatterStripToolsOnAgentsTOMLSet` (`render_test.go:1114-1163`) — assertions: line 1144 stale `tools:` absent; line 1149 `disallowedTools:` absent; line 1156 injected `allowedTools: Read` present; line 1160 `name: go-builder-agent` survives. Same `d3UserTierFrontmatter` helper. Order preserved (allowedTools is injected post-strip, lands AFTER the d3UserTierFrontmatter's body; substring assertion still passes).
  5. `TestAssembleAgentFileBody_FrontmatterPreservedWhenAgentsTOMLAbsent` (`render_test.go:1378-1427`) — assertions: line 1406 `model: opus` survives (stripModel false); line 1410 `tools: Read, Bash` stripped (stripTools always true per W3-FF12); line 1415 `allowedTools:` not injected (binding empty); line 1419 `disallowedTools:` not injected; line 1424 `body-bytes-preserve-marker` survives. Same `d3UserTierFrontmatter` helper.

  All 5 migrated tests pass per `mage testPkg ./internal/app/dispatcher/cli_claude/render` 70/70 green. The fixture mutation is mechanical — pre-fixture-suffix the validator-conforming preamble (200+ char filler + `# PLACEHOLDER` marker), post-fixture-suffix the test-specific sentinel — so the strip/inject/resolve assertions still drive the same code paths. The `validatorConformingBodySuffix` doc-comment at `render_test.go:797-808` explicitly justifies this ordering ("sentinels must appear AFTER the marker so substring assertions on them still hit").
- **Result — REFUTED.** Migration is sound; no original assertion lost; helper-based mutation pattern keeps the diff mechanical and the substring assertions unbroken.

#### W3-D5-FF10 [INFO / HF8 wiring proof — validator is not a dangling helper] — REFUTED

- **Attack:** the brief flags the load-bearing HF8 contract: validator MUST be wired into Render's exit, not shipped as a dangling exported helper.
- **Method:** Read `render.go:288-299` (the wiring site) + check that every D5 failure-path test calls `Render` end-to-end rather than `validateBundle` standalone.
- **Trace:** Wiring site at `render.go:288-297`: `// 6. Post-render validator (Drop 4c.6 W3.D5). ... if err := validateBundle(bundle, binding); err != nil { rollback.run(); return "", fmt.Errorf("render: validate bundle: %w", err) }`. Sits between `renderSettings` (step 5) and `return promptBody, nil` (line 299). Matches the round-3-locked PLAN.md acceptance contract at lines 229-230 ("validator MUST be invoked from `Render`'s exit path"). Test wiring proof: every D5 failure-path test (`TestRenderValidatorFailsOnTooShortBody`, `_FailsOnMissingFrontmatter`, `_FailsOnMissingMarker`) calls `render.Render(...)` end-to-end (lines 1471, 1509, 1537) and asserts on the returned error — never on `validateBundle` standalone. Builder explicitly justified the wiring in `BUILDER_WORKLOG.md` line 28 ("HF8 contract: validator MUST be wired into `Render`'s exit, not shipped as a dangling helper. Every D5 failure-path test calls `Render()` end-to-end..."). Plus `validateBundle` and `validateAgentBodyShape` are both package-private (lower-case start) — no external caller path exists for them to "dangle" through.
- **Result — REFUTED.** Wiring is load-bearing AND test-proven AND structurally enforced via package privacy.

#### W3-D5-FF11 [INFO / Marker at line 1 with sufficient padding — A+B+C interaction] — REFUTED

- **Attack:** the brief flags `Signal C interaction with body length: if body has \# Section 0\` marker AT line 1 (first 11 chars) and nothing else for 200+ chars, does it pass A AND C?`.
- **Method:** Static analysis of the signal evaluation order + interaction.
- **Trace:** Hypothetical body: `"---\nname: x\n---\n# Section 0\n" + 200-char-filler`. Signal evaluation order per `render.go:354-392`:
  - Signal B: leading `---\n` present (yes), closing `---\n` at index after `name: x\n`, frontmatter contains `name:` (yes) → PASS.
  - `afterOpen` = `"name: x\n---\n# Section 0\n<filler>"`. `closeIdx` = index of second `---\n`. `postFrontmatter` = `"# Section 0\n<filler>"`.
  - Signal A: `len(postFrontmatter) = 11 + 1 + 200 = 212 > 200` → PASS.
  - Signal C: `strings.Contains("# Section 0\n<filler>", "# Section 0")` → PASS.
  All three signals pass cleanly. This is the intended-behavior happy path — the body has a real role-section header at line 1 of post-frontmatter and substantial content below. No bug.
  Counterexample edge: `"---\nname: x\n---\n# Section 0\n"` alone (no filler) → `postFrontmatter = "# Section 0\n"`, len = 12 → FAILS Signal A. Confirms validator catches "marker-but-no-content" stubs.
- **Result — REFUTED.** Marker-at-line-1 with sufficient padding passes correctly; the same shape without padding fails Signal A as designed.

#### W3-D5-FF12 [HIGH / mage ci — drop-level gate failure NOT W3.D5-attributable] — REFUTED-AS-NOT-W3.D5

- **Attack:** the brief mandates `Run mage ci`.
- **Method:** Run `mage ci` from `/Users/evanschultz/Documents/Code/hylla/tillsyn/main/`.
- **Trace:** Captured output:
  ```
  3077 tests passed across 26 packages.

  [ERROR] Coverage threshold not met
    Each package must stay at or above 70.0% coverage.
    github.com/evanmschultz/tillsyn/internal/fsatomic 64.0%
  Error: coverage below 70.0%: github.com/evanmschultz/tillsyn/internal/fsatomic 64.0%
  ```
  All 3077 tests pass (including all 70 render-package tests). Coverage gate fires on `internal/fsatomic` at 64.0% < 70.0% threshold. W3.D5's own package `internal/app/dispatcher/cli_claude/render` reports 82.8% coverage — well above the 70% floor. `internal/fsatomic/` is UNTRACKED in `git status` (`?? Untracked: 1 files / internal/fsatomic/`), is a brand-new sibling-droplet (W2.D1 — explicitly owned by `4c.6.W2.D1 — internal/fsatomic/ atomic file-write helper`), and is COMPLETELY OUT OF SCOPE for W3.D5. W3.D5's scope per PLAN.md lines 222-264 is exactly two files: `render.go` + `render_test.go`. The fsatomic coverage gap is sibling-owned and ALREADY flagged as W2-D1-FF1 at this file's lines 1632-1652. Builder explicitly deferred mage-ci verification to drop-orch per `BUILDER_WORKLOG.md` line 47 ("`mage ci` — NOT run by this builder per droplet constraint; drop-orch runs `mage ci` at drop end"). Pattern mirrors W3.D4's W3-D4-FF8 carve-out (this file line 1856-1861) verbatim.
- **Result — REFUTED-AS-NOT-W3.D5.** The fsatomic coverage failure is NOT a W3.D5 falsification — sibling-droplet W2.D1 owns it (W2-D1-FF1 already raised). W3.D5's changed package coverage is 82.8%, well above threshold. Routing: orchestrator already owns the W2.D1 round-2 respawn directive.

### Attack family table

| Family                          | Result    | Notes                                                                                     |
| ------------------------------- | --------- | ----------------------------------------------------------------------------------------- |
| B1 counterexample-search        | CONFIRMED-NOT-D5 | W3-D5-FF12 — `mage ci` red on `fsatomic 64.0%`. Not D5's fault (D1 inheritance, W2-D1-FF1 prior round). |
| B2 contract-preservation        | REFUTED   | FF1 (Signal C accepted NIT), FF2 (Signal B substring nit, low-risk), FF3 (boundary 200/201 correct), FF4 (disk read confirmed), FF5 (wrap chain holds), FF6 (rollback clean), FF11 (marker-at-line-1 interaction correct). |
| B3 hidden-coupling              | REFUTED   | FF7 — no package-level mutable state; concurrent renders against disjoint roots cannot interfere; embed.FS read-only by compile-in invariant. |
| B4 yagni                        | REFUTED   | Validator surface = three pure functions (`validateBundle` disk re-read, `validateAgentBodyShape` pure check, `ErrInvalidAgentBody` sentinel) + AND-chained 3 signals. No premature generalization; substring over regex; minimal abstraction. |
| B5 spec-compliance              | MIXED     | Droplet-internal acceptance (3-signal AND, disk re-read, wiring, 27/27 placeholders, W3-PF1 D2+D3 test preservation, HF8 proof) all satisfied per FF3/FF4/FF6/FF8/FF9/FF10. `mage ci green` bullet (FF12) FAILS at drop level due to pre-existing W2-D1-FF1 — NOT W3.D5-attributable. |
| B6 shipped-but-not-wired        | REFUTED   | FF10 — validator wired at Render's exit; HF8 contract verified via end-to-end Render() tests, not standalone validateBundle tests; validator is package-private with no external dangling path. |
| B7 prompt-injection             | EXHAUSTED | DORMANT pre-team-feature per agent definition.                                            |

### Probes executed

- **`mage ci`** — RED. `Coverage threshold not met` → `internal/fsatomic 64.0%`. Same W2-D1-FF1 surface from prior round. Render-package coverage 82.8% / 70/70 tests green.
- **`mage testPkg ./internal/app/dispatcher/cli_claude/render`** — GREEN, 70/70 tests pass. Render package is sound.
- **Direct `Read` of `internal/app/dispatcher/cli_claude/render/render.go` lines 1-1109** — full file inspection; confirmed validator wiring, error wraps, rollback machinery, disk-read seam, package-level immutability.
- **Direct `Read` of `internal/app/dispatcher/cli_claude/render/render_test.go` lines 1-200 / 700-1200 / 1200-1661** — confirmed 5 new validator tests, 27-placeholder WalkDir sub-test, 5 migrated D2/D3 fixture tests with substring assertions preserved.
- **Direct `Read` of `internal/templates/embed.go`** — confirmed 27 placeholder `//go:embed` directives (7 till-gen + 7 till-go + 7 till-gdd + 5 legacy go-* + 1 orchestrator-managed).
- **Direct `Read` of `workflow/drop_4c_6/DROP_4c.6.W3_BUNDLE_AND_ISOLATION/PLAN.md` lines 222-264 + 303-307** — W3.D5 acceptance criteria + W3-FF13 NIT documentation.
- **Direct `Read` of `workflow/drop_4c_6/BUILDER_WORKLOG.md` lines 1-100** — builder's W3.D5 design rationale and TDD red→green trace.
- **Direct `Read` of sample placeholder bodies** (`builtin/agents/till-go/builder-agent.md`, `builtin/agents/till-gen/commit-message-agent.md`, `builtin/agents/till-go/go-builder-agent.md`) — confirmed each carries `# PLACEHOLDER` marker (Signal C) + frontmatter with `name:` (Signal B) + > 200 char post-frontmatter body (Signal A).
- **Static byte-count reasoning** on `builder-agent.md` post-frontmatter (241 bytes including em-dash UTF-8 expansion) — confirms Signal A clearance by margin of 41 bytes over the 200-char floor.
- **Hylla `hylla_search_keyword`** for `validateBundle ErrInvalidAgentBody validateAgentBodyShape` against `github.com/evanmschultz/tillsyn@main` — returned `enrichment still running` error. Same staleness window prior rounds documented. Fell back to Read.

### Summary

**Verdict: PASS — no W3.D5-attributable CONFIRMED counterexamples.** All 11 attack vectors REFUTED. The cross-droplet `mage ci` failure (W3-D5-FF12) is documented as sibling-owned W2-D1-FF1 (prior round, already routed); W3.D5's own render-package is at 82.8% coverage with 70/70 tests green.

**Soundness summary:** The validator's 3-signal AND check (B → A → C evaluation order) is logically sound; each signal's `%w`-wrapped error returns satisfy `errors.Is(err, ErrInvalidAgentBody)` through Render's double-wrap; the disk-read approach catches potential `os.WriteFile` truncation regressions; rollback wipes every render-written artifact (system-prompt.md + plugin/ subtree) while preserving the upstream F.7.1 manifest.json; concurrent renders against disjoint bundle roots are race-free by construction (no package-level mutable state); placeholder positive coverage is 27/27 verified live; D2+D3 fixture migrations preserve every original sentinel assertion via the suffix-before-sentinel ordering in `validatorConformingBodySuffix()` and `d3UserTierFrontmatter()`; HF8 wiring is load-bearing AND test-proven AND structurally enforced via package privacy on `validateBundle` / `validateAgentBodyShape`.

**Severity breakdown:**

- 0 CONFIRMED counterexamples attributable to W3.D5.
- 1 CONFIRMED-NOT-D5 counterexample (W3-D5-FF12 / mage ci red on fsatomic 64.0%) — sibling-owned, prior W2-D1-FF1.
- 2 NIT-severity findings (W3-D5-FF1 accepted carryforward W3-FF13; W3-D5-FF2 new Signal B `name:` substring false-positive — both refinement candidates, not blocking).
- 8 INFO-severity findings (each a probe outcome documenting a verification, all REFUTED).

**Routing to orchestrator:**

1. **W3.D5 droplet itself: PASS — no respawn required for W3.D5 work.**
2. **W3.D5 builder did NOT cause the mage-ci coverage gate failure.** W2-D1-FF1 is the open finding; routing is to W2.D1's builder (per the W2.D1 Round 1 fix-hint at this file's line 1651).
3. **Refinement candidates (post-MVP, not blocking):**
   - W3-D5-FF1 Signal C backticked-marker false-positive — accepted NIT carryforward per `PLAN.md:303-307`.
   - W3-D5-FF2 Signal B substring `name:` false-positive — NEW NIT, line-anchored match would close it. Stage alongside W3-FF13 for a future validator-tightening drop.
   - W3-D5-FF4 micro-edge — `ErrInvalidAgentBody` classification covering I/O failures, separate `ErrAgentBodyReadFailure` sentinel could clean up error-routing surface. Refinement candidate, not blocking.

No routing back to orchestrator beyond standard W3.D5 closure-state confirmation + the existing W2-D1-FF1 cross-reference.

### Hylla Feedback

- **Query:** `hylla_search_keyword` for `validateBundle ErrInvalidAgentBody validateAgentBodyShape` against `github.com/evanmschultz/tillsyn@main`.
  - **Missed because:** `enrichment still running for github.com/evanmschultz/tillsyn@main` — the W3.D5 work is uncommitted (visible in `git status` as modified files on `render.go` + `render_test.go`) so the new symbols cannot possibly be in any Hylla snapshot. Even pre-commit W3.D2/W3.D3 commits (`d671b91`, `7346fe7`) and the W2.D1 commit (`5e17515`) appear to have triggered an in-flight enrichment that is still settling. Same pre-cascade-ingest staleness window documented in this file's W3.D2 round-2 Hylla Feedback (line 1619) and W3.D4 Hylla Feedback (line 1936) and W2.D1 Hylla Feedback (line 1711).
  - **Worked via:** direct `Read` against `render.go` (focused offsets for the validator surface lines 110-137, 294-297, 302-392 and the rollback machinery lines 401-420), `render_test.go` (focused offsets for the 5 new validator tests + 5 migrated D2/D3 tests + helpers), `embed.go` (full file for the 27-placeholder embed-list reconciliation), `BUILDER_WORKLOG.md` (Round 1 W3.D5 entry for builder's design decisions), `DROP_4c.6.W3_BUNDLE_AND_ISOLATION/PLAN.md` (W3.D5 acceptance bullets + W3-FF13 NIT documentation), sample placeholder `.md` files for byte-count grounding. The `mage testPkg` and `mage ci` runs grounded the dynamic claims.
  - **Suggestion:** non-blocking — same suggestion as prior rounds. When enrichment is in flight, the keyword tool could surface the snapshot ID of the last fully-ingested baseline so the caller can fall back to the prior snapshot (potentially still useful for unchanged subsystems like the `templates.DefaultTemplateFS` consumer-graph) rather than to non-Hylla tools wholesale. This is the recurring pre-cascade-ingest staleness shape and warrants tracking as a Hylla refinement after dogfood. Multiple droplets in this drop have raised the same suggestion; aggregate at drop-end Hylla refinements rollup.

---

## Droplet 4c.6.W3.D6 — Round 1

Doc-only droplet — pure doc-comment rewrite of `renderAgentFile`'s leading
comment in `internal/app/dispatcher/cli_claude/render/render.go` (lines
520-595 post-edit). Verifies the rewrite's factual claims, the
3-landing breadcrumb attribution, the F.7.3b correction, the
`--bare collapses Path B` framing, and the comment-only diff property.
Scope is acceptance-stamped against the W3.D6 attack vectors.

### Findings

- **W3-D6-FF1 (CONFIRMED — MAJOR — cite drift on load.go).** The doc-comment's
  3-landing breadcrumb (line 583-584) reads:
  > "Drop 4c F.7.2 landed the `templates.AgentBinding.SystemPromptTemplatePath`
  > schema field at `internal/templates/schema.go:573` with validator
  > at `internal/templates/load.go:1031-1055`."
  The schema.go:573 cite is CORRECT — the field declaration is verbatim
  at line 573 (`Read` of `internal/templates/schema.go` lines 565-586).
  The load.go:1031-1055 cite is **FACTUALLY WRONG**. Lines 1028-1055 of
  `internal/templates/load.go` contain `validateChildRuleRecursionDepth` —
  a completely different validator for the child-rule recursion-depth
  guard. The actual `validateSystemPromptTemplatePath` validator lives at
  lines 1687-1706 (verified via `grep -n SystemPromptTemplatePath` →
  hit at 1687 + `Read` of lines 1682-1706). The site that calls the
  validator from the binding loop is `load.go:1642`
  (`if err := validateSystemPromptTemplatePath(kind, binding.SystemPromptTemplatePath); err != nil {`).
  Severity is MAJOR rather than CRITICAL because the breadcrumb is
  historical-context not a load-bearing claim used by call sites — but
  the entire purpose of D6 is to correct prior stale cites and the
  rewrite introduces a NEW stale cite at the same surface. Fix: replace
  the `load.go:1031-1055` substring with `load.go:1687-1706` (or
  optionally `load.go:1642 + 1687-1706` to anchor both the call site
  and the validator body). This MUST be fixed before D6 closes — a
  doc-correction droplet that introduces a factually-wrong cite at the
  exact surface it claims to be correcting is a regression on its own
  premise.

- **W3-D6-FF2 (REFUTED — schema.go:573 cite correctness).** The cite
  `internal/templates/schema.go:573` is accurate — the `Read` of
  lines 565-586 shows the field declaration verbatim at 573:
  `SystemPromptTemplatePath string \`toml:"system_prompt_template_path"\``.
  No drift.

- **W3-D6-FF3 (REFUTED — 3-landing breadcrumb attribution accuracy).**
  Attribution of (a) F.7.2 to schema field, (b) W3.D1 to BindingResolved
  wiring, (c) W3.D2 to 3-tier resolver is otherwise CORRECT modulo
  W3-D6-FF1's load.go offset:
  - **F.7.2 (schema field landing).** Schema field at
    `internal/templates/schema.go:573` exists — REFUTED-correctly. (Validator
    location is FF1.)
  - **W3.D1 (BindingResolved wiring).** `grep -n` over
    `internal/app/dispatcher/cli_adapter.go` shows
    `BindingResolved.SystemPromptTemplatePath` declared at line 131 with
    field-doc at 108-130; populator at
    `internal/app/dispatcher/binding_resolved.go:121` (verified via
    `grep -n`). Both files are exactly the W3.D1 surface called out in
    the breadcrumb. REFUTED.
  - **W3.D2 (3-tier resolver).** `grep -n assembleAgentFileBody` over
    `render.go` confirms the resolver exists at the symbol name the
    breadcrumb cites; the surrounding doc-comment (untouched by D6) at
    lines 540-548 of the same file describes the till-gen fallback that
    the breadcrumb references. REFUTED.

- **W3-D6-FF4 (REFUTED — F.7.3b stub correction landed correctly).** The
  pre-D6 doc-comment was a stub that PLAN's W3.D6 work item flagged as
  reading "Behavior loaded from the canonical ... template at the
  system-installed plugin path" (the F.7.3b stub claim). The post-D6
  doc-comment at lines 529-534 reads:
  > 'The pre-W3 F.7.3b stub claim that "Behavior loaded from the
  > canonical ... template at the system-installed plugin path" —
  > verbatim in the body sentence the stub used to emit at disk-write
  > time — was FACTUALLY WRONG under the actual argv and is removed by
  > W3.D2. This doc-comment records the post-W3 truth so the next
  > reader cannot regress on it.'
  This explicitly names the wrong sentence, brands it FACTUALLY WRONG,
  attributes its removal to W3.D2, and explains why the next reader
  cannot regress on it. Not a paraphrase; an explicit correction with
  before/after attribution. REFUTED.

- **W3-D6-FF5 (REFUTED — `--bare collapses Path B` cite accuracy).** The
  doc-comment at lines 522-528 cites
  `RESEARCH/ISOLATION_ENFORCEMENT_FIX.md § A.9 + § C.5` for the claim
  that `--bare` collapses Path B. `Read` of
  `workflow/drop_4c_6/RESEARCH/ISOLATION_ENFORCEMENT_FIX.md` confirms:
  - §A.1 quotes Claude Code's official CLI reference: `--bare` skips
    plugins among other things (line 20).
  - §A.9 (line 113-131) is the "Definitive answer to question 1"
    section that explicitly enumerates what `--bare` skips
    (~/.claude/CLAUDE.md, ~/.claude/agents/*, ~/.claude/plugins/cache,
    ~/.claude/hooks, etc.).
  - §C.5 (line 238 in the doc, "Net finding for Path B") concludes:
    `"Under --bare ... no — Path B is disabled by --bare ... Tillsyn's
    existing --bare already prevents that merge."`
  - §D.5 (line 329-334) is the original "kill the misleading
    doc-comments" fix change that motivated W3.D6 itself, citing the
    exact same `render.go:307-319` surface (now relocated to ~520-595
    after intervening edits). The fix-text in §D.5 prescribes the
    replacement framing D6 in fact lands. REFUTED.

- **W3-D6-FF6 (REFUTED — comment-only diff).** `git show 01a1244` is the
  W3.D6 commit. `git show 01a1244 --stat` reports
  `1 file changed, 46 insertions(+), 9 deletions(-)` against
  `internal/app/dispatcher/cli_claude/render/render.go`. The full diff
  hunks (lines 517-595 of the post-edit file) show every removed line
  and every added line begins with `//` or is a `//`-prefixed
  continuation. No function body / signature / non-comment line was
  touched. REFUTED.

- **W3-D6-FF7 (REFUTED — SPAWN_PIPELINE.md non-touch).** `git diff
  01a1244~1 01a1244 --numstat` returns one row only —
  `internal/app/dispatcher/cli_claude/render/render.go 46 9`. No row
  for `SPAWN_PIPELINE.md`. Zero changes to that file in the D6 commit.
  REFUTED. (Caveat: `SPAWN_PIPELINE.md` was flagged in
  `RESEARCH/ISOLATION_ENFORCEMENT_FIX.md` § D.5 as also containing the
  stale two-paths framing; D6's scope was explicitly the render.go
  doc-comment alone, so SPAWN_PIPELINE.md correction is correctly
  out-of-scope for D6 and remains as a separate refinement candidate.)

- **W3-D6-FF8 (REFUTED — `assembleAgentFileBody` doc-comment
  non-touch).** `grep -n assembleAgentFileBody` over the post-edit
  `render.go` finds five hits at lines 96, 113, 139, 537, 589.
  - Lines 96-139 are pre-existing error-sentinel doc-comments
    (`ErrAgentBodyNotFound`, `ErrInvalidAgentBody`,
    `ErrInvalidAgentTemplatePath`) that REFERENCE `assembleAgentFileBody`
    but are not its OWN doc-comment.
  - Lines 537 + 589 are the NEW D6 doc-comment text on `renderAgentFile`
    that mentions `assembleAgentFileBody` in prose (D6 cross-refs to
    the resolver function from the new comment).
  - The actual `assembleAgentFileBody` function declaration + its
    leading doc-comment are not in the D6 diff hunk range. D6 added 46
    lines and removed 9, all inside the `renderAgentFile` doc-comment
    surface at lines 520-595 of the post-edit file. The
    `assembleAgentFileBody` function's own doc-comment is untouched.
    REFUTED.

- **W3-D6-FF9 (REFUTED — mage ci green).** `mage ci` ran from the
  worktree root, full suite, 3081 tests across 26 packages all pass,
  every package at or above the 70.0% coverage threshold, render
  package at 82.8%, templates package at 94.5%, dispatcher package at
  76.1%, build succeeds. Drop's working-tree state has uncommitted
  modifications in other unrelated paths (auto_generate_steward,
  template_service, etc.) that are NOT in D6's scope; those compile
  and pass. REFUTED.

### Probes executed

- **`git show 01a1244 --stat` + `git show 01a1244 -- internal/app/dispatcher/cli_claude/render/render.go`** — confirmed the W3.D6 commit's exact diff (46 ins, 9 del; one file; comment-only).
- **`git diff 01a1244~1 01a1244 --numstat`** — confirmed render.go is the SOLE file in the W3.D6 commit (no SPAWN_PIPELINE.md row).
- **`Read` of `internal/templates/schema.go` lines 565-586** — verified `SystemPromptTemplatePath string \`toml:"system_prompt_template_path"\`` is at line 573 exactly.
- **`grep -n SystemPromptTemplatePath internal/templates/load.go`** — found references at 221, 475, 1599, 1615, 1642, 1682, 1687, 1712. The validator function is `validateSystemPromptTemplatePath` at line 1687; its caller is line 1642. The 1031-1055 cite hits `validateChildRuleRecursionDepth` (entirely unrelated).
- **`Read` of `internal/templates/load.go` lines 1025-1064 + lines 1682-1716** — confirmed 1025-1055 is `validateChildRuleRecursionDepth` body and 1687-1706 is `validateSystemPromptTemplatePath` body.
- **`Read` of `internal/app/dispatcher/cli_claude/render/render.go` lines 515-599** — full inspection of the new D6 doc-comment plus the function signature line at 596 (unchanged).
- **`grep -n` over `cli_adapter.go` + `binding_resolved.go`** — confirmed W3.D1 wiring at `cli_adapter.go:131` + `binding_resolved.go:121`.
- **`grep -n assembleAgentFileBody render.go`** — confirmed `assembleAgentFileBody`'s own doc-comment is outside D6's diff hunks.
- **`Read` of `workflow/drop_4c_6/RESEARCH/ISOLATION_ENFORCEMENT_FIX.md` lines 1-30 + `grep` for `C.5 D.5 A.9 bare` (~40 matches)** — confirmed §A.9 and §C.5 are the authoritative `--bare collapses Path B` sources; §D.5 prescribes the exact doc-comment fix D6 lands.
- **`mage ci`** — full suite GREEN, 3081/3081 tests pass, every package ≥ 70.0% coverage, render package at 82.8%.

### Summary

**Verdict: FAIL — 1 CONFIRMED-MAJOR counterexample (W3-D6-FF1).** The
doc-only droplet shipped one verifiably-wrong line cite at the exact
surface it claims to be correcting (`load.go:1031-1055` points at
`validateChildRuleRecursionDepth`, not the SystemPromptTemplatePath
validator). The actual validator lives at `load.go:1687-1706` with its
caller at `load.go:1642`. Because D6's premise is *"fix stale cites in
the renderAgentFile doc-comment so the next reader doesn't regress on
the architecture,"* shipping a new stale cite in the same comment is a
direct regression on the droplet's own acceptance criterion.

**Soundness summary:** Every other attack vector REFUTED. The F.7.3b
stub correction lands correctly (explicit before/after framing, not a
lossy paraphrase). The `--bare collapses Path B` framing matches the
research deliverable's §A.9 + §C.5 + §D.5 chain. The 3-landing
breadcrumb's W3.D1 + W3.D2 attributions are correct against the actual
shipped surfaces in `cli_adapter.go`, `binding_resolved.go`, and the
`assembleAgentFileBody` symbol. Diff is comment-only, no inadvertent
semantic mutation, no SPAWN_PIPELINE.md touch, no
`assembleAgentFileBody`-doc-comment touch. `mage ci` GREEN by
transitivity (no behavior change — comment-only edit).

**Severity breakdown:**

- 1 CONFIRMED counterexample (W3-D6-FF1, MAJOR — stale cite at
  `load.go:1031-1055` in the D6 doc-comment; correct location is
  `load.go:1687-1706` with caller at `load.go:1642`).
- 0 NIT-severity findings.
- 8 REFUTED findings (W3-D6-FF2 through W3-D6-FF9).

**Routing to orchestrator:**

1. **W3.D6 droplet: FAIL — respawn builder for a one-line fix.** Replace
   `internal/templates/load.go:1031-1055` with
   `internal/templates/load.go:1687-1706` (or the more-defensive
   `internal/templates/load.go:1642 + 1687-1706` to anchor both the
   call site and the validator body) on render.go line 584 of the
   post-D6 file. Then re-spawn QA twins.
2. **No cross-droplet routing.** D6's scope is self-contained; no
   sibling builder action required. SPAWN_PIPELINE.md's parallel
   stale-framing issue is correctly out-of-scope for D6 and remains as
   an open refinement candidate (per `RESEARCH/ISOLATION_ENFORCEMENT_FIX.md`
   § D.5) — not a new finding from this round.

### Hylla Feedback

N/A — action item touched non-Go files only at the QA-evidence layer
(MD doc reads against RESEARCH/*.md + workflow/*.md). The Go probe
points for this round (`internal/templates/schema.go`,
`internal/templates/load.go`, `internal/app/dispatcher/cli_adapter.go`,
`internal/app/dispatcher/binding_resolved.go`,
`internal/app/dispatcher/cli_claude/render/render.go`) were all
recently-modified-in-drop or recently-loaded-into-context — direct
`Read` and `grep` were the right tools for line-precision cite
verification (Hylla's symbol search would not have surfaced a
line-offset within a function body any faster than `grep -n` + `Read`).
No Hylla query was attempted for D6 since the work was line-precision
doc verification, not symbol-graph traversal; the absence of a Hylla
query here is by design, not by fallback.

---

## Droplet 4c.6.W2.D8 — Round 1

**QA agent:** go-qa-falsification-agent (sonnet).
**Date:** 2026-05-11.
**Droplet:** `4c.6.W2.D8 — Remove init-dev-config from main.go + help.go + main_test.go`.
**Reviewing commit:** `83ca878` (refactor(drop-4c.6): w2.d8 remove init-dev-config from cmd till).

### Scope under attack

Pure deletion droplet. Removes `initDevConfigCmd` cobra block + `runInitDevConfig` function from `main.go`, `"till init-dev-config"` entry from `help.go`'s `commandHelpSpecs`, `init-dev-config` references + 2 tests from `main_test.go`. Diff: 3 files / +3 / −207 LOC. Builder reported 271/271 pkg tests GREEN and deferred 4 historical doc-comment matches per scope constraint.

### Attack vectors (from spawn brief)

#### W2-D8-FF1 — Plan acceptance vs. declared paths CONTRADICTION (CONFIRMED, ROUTING: plan-level not builder-level)

The plan's `Acceptance` for D8 (PLAN.md line 245-246) states:

- `git grep -n init-dev-config cmd/till/` returns ZERO matches after D8 commits.
- `git grep -n runInitDevConfig cmd/till/ internal/` returns ZERO matches after D8 commits.

The plan's `Paths` for D8 (PLAN.md line 239-242) declares only:

- `cmd/till/main.go`
- `cmd/till/help.go`
- `cmd/till/main_test.go`

Plan does NOT include `cmd/till/install_cmd.go` or `cmd/till/install_cmd_test.go` in D8's paths (those belong to D7.5's file-lock domain). Yet the acceptance criterion targets matches in the entire `cmd/till/` directory.

Post-build state via `rg` (read-only, no fallback to Hylla because non-Go scope misses — these files were freshly committed in `83ca878` so Hylla would be stale anyway; Hylla query attempted on the builder symbol surface, not on grep semantics):

- `cmd/till/install_cmd.go:18-19` — 2 references to `` `init-dev-config` `` in `newInstallCommand` doc-comment ("previously lived under `till init-dev-config`. D8 later removes the legacy `init-dev-config` registration").
- `cmd/till/install_cmd.go:53` — 1 reference: "Body lifted verbatim from runInitDevConfig in main.go (D7.5 lift-and-rename per OQ#3 disposition; D8 removes the original)."
- `cmd/till/install_cmd_test.go:18` — 1 reference: "`init-dev-config` original".

Total: 4 lines (3 `init-dev-config` + 1 `runInitDevConfig`). Builder's count of "4 historical doc-comment matches" reconciles exactly (since the rg output shows 3 `init-dev-config` lines + 1 `runInitDevConfig` line = 4 total residuals).

**Severity classification:** CONFIRMED counterexample to *literal plan-acceptance text*, but NOT a builder-correctness counterexample. The builder followed scope discipline correctly — touching `install_cmd.go` / `install_cmd_test.go` would have violated D8's declared `paths[]`, and those files are D7.5-owned. The contradiction is a plan-level oversight: D8's acceptance criterion as written cannot be satisfied by D8's declared paths.

**Routing:** plan-level finding — the W2 planner under-scoped D8's paths (or alternately, over-scoped D8's acceptance). Pre-merge correction options:

1. Expand D8's `paths` to include `install_cmd.go` + `install_cmd_test.go` (revives D8 with broader scope; would require a Round-2 D8 builder run).
2. Loosen D8's acceptance criterion from "ZERO matches" to "ZERO non-historical-doc-comment matches" (codifies the deferral the builder already took).
3. Spawn a tiny follow-up droplet (W2.D9 or W2.D8.5) that owns `install_cmd.go` + `install_cmd_test.go` and scrubs the 4 historical references in one mechanical edit.

Option 3 is the cleanest because it preserves the file-lock contract (D7.5's domain stays D7.5's, D8 stays a minimal removal-only delta) and the scrub edit is small enough to bundle with the next near-term droplet without inflating risk.

#### W2-D8-FF2 — Port-lineage judgment: residual doc-comments are mixed acceptable / dead-reference (judgment finding)

Judging the 4 residual doc-comment matches per the spawn brief directive ("Are these acceptable as port-lineage documentation, or should they be scrubbed?"):

1. **`install_cmd.go:18-19`** — references "previously lived under `till init-dev-config`. D8 later removes the legacy `init-dev-config` registration." *Verdict: ACCEPTABLE PORT-LINEAGE during drop, but loses meaning post-merge* — "D7.5" and "D8" are drop-internal droplet IDs that become opaque outside the drop's commit log. Best-practice post-drop edit: rewrite to "previously lived under `till init-dev-config` (removed during the W2 builds in this drop)" or simpler: "the new home for the dev-config-creation behavior."
2. **`install_cmd.go:53`** — "Body lifted verbatim from runInitDevConfig in main.go (D7.5 lift-and-rename per OQ#3 disposition; D8 removes the original)." *Verdict: DEAD REFERENCE post-merge.* It references a function (`runInitDevConfig`) that no longer exists anywhere in the codebase. A future developer reading this comment will grep for `runInitDevConfig`, find nothing, and lose 5-10 minutes confirming the comment is stale. This is the strongest scrub candidate of the four.
3. **`install_cmd_test.go:18`** — "Ported verbatim from the TestRunInitDevConfigCreatesDebugConfig body in main_test.go (D7.5 lifts the behavior into a new `install` cobra command; D8 later removes the `init-dev-config` original)." *Verdict: DEAD REFERENCE post-merge.* `TestRunInitDevConfigCreatesDebugConfig` no longer exists (deleted in this very commit `83ca878`). Same grep-and-lose-time risk as item 2.

**Aggregate judgment:** 1 of 4 references is borderline acceptable as port-lineage prose; 3 of 4 reference deleted symbols and should be scrubbed. The right cleanup is to rewrite the doc-comments to describe what `runInstall` *does* (its behavior contract) without naming the dead symbol it was lifted from. Refinement candidate.

**Routing:** non-blocking refinement candidate for the post-drop cleanup pass or for the W2.D8.5 / W2.D9 follow-up if option 3 above is taken.

#### W2-D8-FF3 — Behavior regression: `till init-dev-config` reports unknown command (REFUTED — confirmed via test-suite proxy)

The spawn brief asks: "with `init-dev-config` removed, can a user STILL run `till init-dev-config`? Verify cobra reports `unknown command` (NOT silent failure)."

Direct binary execution (`./bin/till init-dev-config`) is gated for this agent. Verification proceeded via test-suite proxy:

- `cmd/till/main_test.go:2461-2467` defines `TestRunUnknownCommand` which calls `run(ctx, []string{"unknown-command"}, ...)` and asserts the returned error's message contains the literal `"unknown command"`. This is cobra's standard unregistered-command handler.
- `initDevConfigCmd` is GONE from `rootCmd.AddCommand(...)` (verified by direct `Read` of `cmd/till/main.go:1886` — the AddCommand call lists `serveCmd, mcpCmd, authCmd, projectCmd, actionItemCmd, dispatcherCmd, embeddingsCmd, captureStateCmd, kindCmd, leaseCmd, handoffCmd, exportCmd, importCmd, pathsCmd, initCmd, installCmd` — 16 commands, no `initDevConfigCmd`).
- `TestRunRootHelp` (`main_test.go:476` per builder's edit) now asserts the root-help list WITHOUT `init-dev-config` — and the test is GREEN in the 271/271 pass count.

Cobra's well-documented behavior for `RootCmd.Execute()` invoked with an unregistered subcommand is to emit `Error: unknown command "X" for "till"` to stderr and return a non-nil error from `Execute()`. Since:

1. `init-dev-config` is no longer in the command tree (`AddCommand` confirmed by Read),
2. `TestRunUnknownCommand` exercises the cobra unknown-command path and is GREEN in the 271/271 result,

the behavior IS the unknown-command path. **REFUTED** — silent-failure regression is not present.

#### W2-D8-FF4 — Test deletion correctness: leftover imports / helpers (REFUTED)

Builder deleted `TestRunInitDevConfigCreatesDebugConfig` (~48 LOC) + `TestRunInitDevConfigUpdatesExistingConfig` (~57 LOC) from `main_test.go`. Verification:

- `rg -n "runInitDevConfig|initDevConfigCmd|ensureLoggingSectionDebug" cmd/till/main.go cmd/till/main_test.go` shows `ensureLoggingSectionDebug` survives (still defined at `main.go:2088`, still tested at `main_test.go:3319` via `TestEnsureLoggingSectionDebug`) and `shellEscapePath` survives (still defined at `main.go:2022`, still tested at `main_test.go:2997`).
- The deleted tests imported `platform.Options{...}` and `shellEscapePath` calls — both functions remain referenced from OTHER tests (`TestShellEscapePath` at `main_test.go:2989-3000`, multiple `platform.Options` consumers across the file). No orphaned imports possible since both names are multiply-referenced.
- `mage test-pkg ./cmd/till` is GREEN at 271/271 — Go's compile-time unused-import error would have caught any orphan. **REFUTED.**

#### W2-D8-FF5 — Rich-help table-test row deletion: 1:1 alignment between `commandHelpSpecs` and registered commands (REFUTED)

The spawn brief flags the table-test row deletion as a sibling-row-swap risk (builder used "unique-content anchors" per W2-PF1 carryforward). Verification by Read:

- `cmd/till/main.go:1886` (`rootCmd.AddCommand`) registers exactly 16 commands. Top-level names (per `Use:` declarations in main.go and `newInstallCommand`/`newInitCommand` factory bodies): `serve`, `mcp`, `auth`, `project`, `action_item`, `dispatcher`, `embeddings`, `capture-state`, `kind`, `lease`, `handoff`, `export`, `import`, `paths`, `init`, `install`.
- `cmd/till/help.go` `commandHelpSpecs` map top-level entries (filtering for `"till X":` where X has no spaces — i.e. root-level commands): `till`, `till serve`, `till mcp`, `till embeddings`, `till kind`, `till lease`, `till dispatcher`, `till handoff`, `till export`, `till import`, `till paths`, `till init`, `till install`. Plus subcommand-level entries (`till embeddings status`, `till lease list`, etc.) which are layered onto cobra subcommands via `applyCommandHelpSpecs`' tree walk.

Cross-check: every registered top-level command in `rootCmd.AddCommand` has either an exact entry in `commandHelpSpecs` OR no entry (cobra falls back to the command's own `Long` / `Example` for commands without an explicit help-spec override — confirmed at `help.go:421-432` via `walkCommands`). `till action_item`, `till auth`, `till project`, `till capture-state` have no top-level helpSpec entries, which is consistent across the pre-D8 state (`commandHelpSpecs` was a *richer-help override layer*, not a 1:1 mirror). The deleted `"till init-dev-config":` entry was a richer-help override for the now-removed command — the row at `help.go:392-407` (`"till install":`) survives correctly, with example block listing `till install`, `till --app tillsyn install`, `till --home /tmp/tillsyn-dev install` matching D7.5's contract.

No sibling-row was deleted by accident. **REFUTED.**

#### W2-D8-FF6 — `runInitDevConfig` function deletion: called from anywhere else? (REFUTED)

`rg -n "runInitDevConfig|initDevConfigCmd|ensureLoggingSectionDebug" cmd/ internal/` returns 5 matches, all in:

- `cmd/till/install_cmd.go:53` (doc-comment only, no call — this is W2-D8-FF1's residual).
- `cmd/till/install_cmd.go:95` (`ensureLoggingSectionDebug` call — *not* `runInitDevConfig`).
- `cmd/till/main.go:2087-2088` (`ensureLoggingSectionDebug` definition).
- `cmd/till/main_test.go:3319` (`ensureLoggingSectionDebug` test call).

Zero call-sites for `runInitDevConfig(` anywhere in production code. The only mention left is the W2-D8-FF1 doc-comment which is a dead reference (W2-D8-FF2 item 2) but not a call. **REFUTED — `runInitDevConfig` removal is call-clean.**

#### W2-D8-FF7 — `shellEscapePath` still in use by `runInstall` (REFUTED)

Direct Read of `cmd/till/install_cmd.go:106-110`:

```
return writeCLIKV(stdout, "Dev Config", [][2]string{
    {"status", msg},
    {"config path", shellEscapePath(configPath)},
    {"logging level", "debug"},
})
```

`shellEscapePath` is called from `runInstall` at line 108. `shellEscapePath` function definition remains at `cmd/till/main.go:2022`. Test `TestShellEscapePath` at `cmd/till/main_test.go:2989-3000` still exercises it with the doc-comment updated to generic phrasing per the W2-PF1 ROUND-2 carryforward pin (builder's worklog confirms this edit was made). **REFUTED — `shellEscapePath` retention rationale holds.**

#### W2-D8-FF8 — `mage test-pkg ./cmd/till` correctness (REFUTED)

Live run: `mage test-pkg ./cmd/till` → `Test summary: tests: 271, passed: 271, failed: 0, skipped: 0`. Matches builder's reported count. Pre-flight delta math (per builder's worklog): 268 (pre-flight) − 2 (D8 deletions) + 5 (D5 concurrent landings) = 271. Math reconciles. **REFUTED.**

#### W2-D8-FF9 — `mage ci` GREEN across all packages (REFUTED)

Live run: `mage ci` → `Test summary: tests: 3081, passed: 3081, failed: 0, skipped: 0, packages: 26, pkg passed: 26`. Coverage threshold met (minimum 70.0%, all packages at or above). Build of `./cmd/till` SUCCESS. **REFUTED — drop-end gate is green.**

This is a notable contrast with the W3.D5 Round-1 result which had `mage ci` RED on the `internal/fsatomic` coverage gate (W3-D5-FF12 / W2-D1-FF1 cross-reference). Between W3.D5 Round-1 and this W2.D8 review, W2.D5 committed (`bde8b46 feat(drop-4c.6): w2.d5 init file-copy pipeline and gitignore ensure`) which appears to have brought `internal/fsatomic` coverage above 70% (current run shows `internal/fsatomic   72.0%` — over the threshold by 2 points). This is unrelated to W2.D8 but worth noting because the prior round's RED state would have falsely flagged this droplet had we re-run without that intervening commit.

#### W2-D8-FF10 — Mid-session race observation residual state (REFUTED)

Builder reported "5 transient `TestInit_*` failures appeared mid-session when D5 builder's `init_cmd_test.go` updates landed (D6-stub expectation strings)." Final state verification:

- Full pkg run shows 271/271 GREEN.
- `mage ci` shows 3081/3081 GREEN across all 26 packages.
- `TestInit_*` tests in `cmd/till/init_cmd_test.go` are all part of the 271 — no flake remnant.

Transient mid-session failures during concurrent builder landings are an artifact of MD-only coordination (no file-lock enforcement until cascade dispatcher ships). The final commit state at `83ca878` is clean. **REFUTED — no residual race-state breakage.**

### Probes executed

- **`git show --stat 83ca878`** — confirmed 3-file / +3 / −207 LOC delta matches builder's worklog claim.
- **`git show 83ca878 -- cmd/till/main.go cmd/till/help.go cmd/till/main_test.go`** — full diff inspection of all three files; confirmed removal of `initDevConfigCmd` block (20 LOC), `runInitDevConfig` function (~63 LOC), `commandHelpSpecs` entry (14 LOC), 2 test functions (~105 LOC), 1 root-help slice element, 1 rich-help table-test row, and the `TestShellEscapePath` doc-comment rewrite. All edits match plan paths + builder worklog.
- **`rg -n "init-dev-config" cmd/ internal/ workflow/example/`** — surfaced 3 residual matches in `cmd/till/install_cmd.go:18-19` and `cmd/till/install_cmd_test.go:18`. Builder's deferral count reconciles.
- **`rg -n "runInitDevConfig|initDevConfigCmd|ensureLoggingSectionDebug" cmd/ internal/`** — surfaced 5 matches, 1 dead reference at `install_cmd.go:53` plus 4 live `ensureLoggingSectionDebug` references (function definition + 1 production call from `runInstall` + 1 test call).
- **Direct `Read` of `cmd/till/install_cmd.go` lines 1-100** — confirmed full `newInstallCommand` body, `runInstall` body, `shellEscapePath` call-site at line 108, port-lineage doc-comments at lines 16-21 and 52-56.
- **Direct `Read` of `cmd/till/install_cmd_test.go` lines 1-30** — confirmed `TestRunInstall_CreatesDebugConfig` function name with underscore (W2-FF2 TEST-NAME CONTRACT), doc-comment at lines 14-21 with `init-dev-config` residual.
- **Direct `Read` of `cmd/till/main.go` lines 1880-1890** — confirmed `rootCmd.AddCommand(...)` call lists 16 commands, no `initDevConfigCmd`.
- **Direct `Read` of `cmd/till/main_test.go` lines 2461-2467** — confirmed `TestRunUnknownCommand` exists and exercises the cobra unknown-command path.
- **Direct `Read` of `cmd/till/help.go` lines 1-50 + 336-447** — confirmed `commandHelpSpecs` map has no `"till init-dev-config"` entry; `"till install"` entry present at lines 393-407 with correct example block.
- **Direct `Read` of `workflow/drop_4c_6/DROP_4c.6.W2_TILL_INIT/PLAN.md` lines 236-258** — confirmed D8's declared paths exclude `install_cmd.go` + `install_cmd_test.go`, AND acceptance criterion targets ZERO matches in `cmd/till/` directory (the W2-D8-FF1 plan contradiction).
- **Direct `Read` of `workflow/drop_4c_6/BUILDER_WORKLOG.md` lines 2407-2444** — confirmed builder's W2.D8 Round-1 entry, design rationale, and Hylla feedback.
- **`mage test-pkg ./cmd/till`** — 271/271 GREEN.
- **`mage ci`** — 3081/3081 GREEN across 26 packages, all packages >= 70.0% coverage, `./cmd/till` build SUCCESS.

### Summary

**Verdict: PASS — no W2.D8-attributable CONFIRMED counterexamples that block droplet closure.** All 9 builder-correctness attack vectors REFUTED. 1 CONFIRMED counterexample exists (W2-D8-FF1) but it is a *plan-level contradiction*, not a builder failure — the builder followed scope discipline correctly and properly escalated the contradiction in the worklog routing note. 1 judgment finding (W2-D8-FF2) classifies the 4 residual doc-comments as 3-of-4 dead references that should be scrubbed in a follow-up pass.

**Soundness summary:** The deletion is mechanically correct — the `initDevConfigCmd` cobra block, `runInitDevConfig` function definition, `commandHelpSpecs` entry, and 2 test functions are gone; the surviving `shellEscapePath` + `ensureLoggingSectionDebug` retain their call-sites in `runInstall`; the rich-help row removal preserves the 1:1 alignment with cobra registrations; the root-help-slice and unknown-command-path tests cover the behavior contract that `till init-dev-config` now returns an unknown-command error; `mage ci` is GREEN end-to-end. The W2-FF2 TEST-NAME CONTRACT and W2-PF1 ROUND-2 doc-comment carryforward pins are both honored.

**Severity breakdown:**

- 0 CONFIRMED counterexamples blocking W2.D8 droplet closure.
- 1 CONFIRMED plan-level contradiction (W2-D8-FF1) — `paths[]` vs `acceptance` mismatch in PLAN.md, builder correctly observed and escalated. Routing: planner-side correction or W2.D9 follow-up droplet.
- 1 judgment finding (W2-D8-FF2) — 3 of 4 residual doc-comments reference deleted symbols and should be scrubbed. Refinement candidate.
- 7 INFO-severity findings (W2-D8-FF3 through W2-D8-FF10) — all REFUTED probes documenting verification of the spawn brief's attack vectors.

**Routing to orchestrator:**

1. **W2.D8 droplet itself: PASS — no respawn required for W2.D8 work.** The builder shipped a correct, scope-disciplined deletion.
2. **W2-D8-FF1 plan contradiction routing (orchestrator decision):**
   - Option A: declare the deferral acceptable as documented in the worklog routing note, mark D8 closed with a noted scope-narrowing of the acceptance criterion. (Lowest friction.)
   - Option B: spawn a small W2.D9 droplet with `paths: [cmd/till/install_cmd.go, cmd/till/install_cmd_test.go]` to scrub the 4 historical references in one mechanical pass. (Cleanest acceptance closure; also closes W2-D8-FF2 item 2 + item 3 dead references in the same edit.)
   - Recommendation: **Option B** if a near-term builder slot is available, because the same edit closes both W2-D8-FF1's literal-acceptance gap AND W2-D8-FF2's strongest scrub candidates (the two dead-symbol references at `install_cmd.go:53` and `install_cmd_test.go:18`). Estimated effort: ~5 LOC change, single file-lock acquisition on each of 2 files, no test changes required.
3. **W2-D8-FF2 refinement candidates (non-blocking):** if Option A is chosen above, the 3 dead-reference doc-comments at `install_cmd.go:53`, `install_cmd_test.go:18`, and `install_cmd.go:18-19` should be tracked as a post-drop cleanup refinement. They will become increasingly opaque to future readers as the drop-internal droplet IDs ("D7.5", "D8") fade from active context.

No routing back to orchestrator beyond the W2-D8-FF1 plan-contradiction decision and the closure-state confirmation for W2.D8 itself.

**`mage ci` result:** GREEN — 3081/3081 tests passed across 26 packages, all packages >= 70.0% coverage, `./cmd/till` build SUCCESS.

### Hylla Feedback

- **Query:** `hylla_search_keyword` would have been the appropriate first stop for verifying `runInitDevConfig` is genuinely call-clean (the spawn brief's W2-D8-FF6 attack). However, the W2.D8 commit (`83ca878`) landed only ~10 minutes before this review began, and prior rounds in this drop have already documented (see this file's W3.D2 round-2 Hylla Feedback at line 1619, W3.D4 Hylla Feedback at line 1936, W2.D1 Hylla Feedback at line 1711, W3.D5 Hylla Feedback at line 2139) that `github.com/evanmschultz/tillsyn@main` is in an in-flight enrichment window because multiple W2/W3 droplets have been committing across the past ~12 hours. A keyword query against a stale snapshot would either miss the deletion (showing `runInitDevConfig` as if it still existed) OR error with `enrichment still running`. Neither outcome is more useful than `rg -n` against the working tree.
  - **Missed because:** pre-cascade-ingest staleness window — multiple recent commits (`83ca878`, `01a1244`, `5def94d`, `bde8b46`, `5bfd242`, `01f4a22`, `83ba37f`, `19ade83`, `5e17515`) have been landing and the artifact's full enrichment cannot keep pace with the drop's commit cadence.
  - **Worked via:** `rg -n` against the working tree (deterministic, fresh, line-numbered output) for the call-site sweep; direct `Read` against `cmd/till/main.go`, `cmd/till/help.go`, `cmd/till/install_cmd.go`, `cmd/till/install_cmd_test.go`, `cmd/till/main_test.go`, and `workflow/drop_4c_6/DROP_4c.6.W2_TILL_INIT/PLAN.md` + `workflow/drop_4c_6/BUILDER_WORKLOG.md` for plan/builder cross-checks; `mage test-pkg ./cmd/till` + `mage ci` for live verification of test/coverage claims.
  - **Suggestion:** non-blocking — same recurring suggestion as prior rounds in this drop. The pre-cascade-ingest staleness shape warrants tracking as a Hylla refinement after dogfood. A "freshness hint" returned by Hylla queries (e.g., "snapshot N is in-flight, last fully-ingested baseline is snapshot M from timestamp T") would let the caller decide whether to fall back to the prior baseline (still useful for unrelated subsystems) or to non-Hylla tools wholesale. Aggregate at drop-end Hylla refinements rollup.

---

## Droplet 4c.6.W2.D5 — Round 1

**Scope:** File-copy pipeline (`runInitPipeline` + `copyAgentFiles` + `copyAgentsTOML` + `ensureGitignore`) at `cmd/till/init_cmd.go:396-576` with consumer-tie tests at `cmd/till/init_cmd_test.go`.

**Builder claim:** Three idempotent copy steps wired into a single pipeline both `runInitTUI` (D4) and `runInitJSON` (D3b) invoke. FLAT agent-file copy (no `till-go/` prefix), `agents.example.toml` → `agents.toml`, line-iteration `.gitignore` ensure with W2-FF10 round-2 LOCKED first-line-only handling. D6 stub error surfaces post-success.

### Attack vectors + findings

#### W2-D5-FF1 — t.Chdir audit across init-pipeline tests

**Attack:** orchestrator removed test-residue artifacts from `cmd/till/` (created by tests that didn't `t.Chdir(t.TempDir())`). Audit ALL tests that drive `runInitPipeline` directly or indirectly. Any test still missing `t.Chdir` is a CONFIRMED HIGH-severity counterexample — pollutes working tree.

**Method:** Read `cmd/till/init_cmd_test.go` line-by-line. For each `Test*` function, classify whether it invokes the pipeline (directly or via `run(...)` cobra dispatch) and whether `t.Chdir` is set BEFORE the invocation.

**Audit table:**

| Test                                                | Line | Pipeline driven?                      | t.Chdir?                              | Verdict   |
| --------------------------------------------------- | ---- | ------------------------------------- | ------------------------------------- | --------- |
| `TestInit_BareInvocation_ReturnsTUIStubError`       | 32   | YES — `run("init")` with stubbed TUI advances to pipeline | YES — `t.Chdir(t.TempDir())` at L33 | OK        |
| `TestInit_JSONInvocation_RoutesToValidParse`        | 72   | YES — `run("init --json ...")`         | YES — `t.Chdir(t.TempDir())` at L73   | OK        |
| `TestInit_JSONParse_TableDriven`                    | 97   | MIXED — valid cases hit pipeline; invalid cases short-circuit before any FS write | YES — `t.Chdir(t.TempDir())` inside `t.Run` at L142 (uniform across all cases for consistency) | OK        |
| `TestRunInitTUI_AcceptsDefaultNameAndSelectsTillGo` | 169  | NO — teatest drives the bubbletea model directly; never calls `runInitTUI` (which is the only path into `runInitPipeline` from the TUI side). Model.Init/Update/View are pure — no FS I/O. | NO (and not needed) | OK        |
| `TestRunInitTUI_DisabledTillGddIsUnselectable`      | 234  | NO — same teatest-only pattern.        | NO (and not needed)                   | OK        |
| `TestRunInitTUI_EscCancelsWalk`                     | 283  | NO — teatest-only; Esc returns Cancelled, never reaches pipeline. | NO (and not needed) | OK        |
| `TestInit_FreshDir_CopiesAllFiles`                  | 345  | YES — uses `runInitJSONInTempDir` helper. | YES — helper `t.Chdir(dir)` at L328  | OK        |
| `TestInit_RerunSafety_NoOverwrite`                  | 395  | YES — invokes `run("init --json")` twice. | YES — `t.Chdir(dir)` at L397           | OK        |
| `TestInit_GitignoreIdempotent`                      | 458  | YES — seeds `.gitignore` then invokes `run`. | YES — `t.Chdir(dir)` at L460          | OK        |
| `TestInit_PreExistingGitignore_AppendsCleanly`      | 484  | YES — both subtests seed and invoke.   | YES — `t.Chdir(dir)` inside `t.Run` at L495 | OK        |

**Working tree verification:** `git status --short` returns clean post-`mage ci`. No `.tillsyn/`, no `agents.toml`, no `.gitignore` artifacts in `cmd/till/` (`ls cmd/till/` shows only `.go` files).

**Verdict: REFUTED.** Every test that exercises the pipeline (directly or transitively) chdir's into `t.TempDir()` before invocation. The three teatest model-driver tests do NOT need chdir because they drive `tea.Model.Update` directly and the model itself performs no FS I/O — they never construct `runInitTUI`'s downstream `runInitPipeline` call site. Working tree is clean post-`mage ci`.

#### W2-D5-FF2 — `copyAgentFiles` till-go group fallback completeness

**Attack:** when `group = till-go`, does `copyAgentFiles` correctly enumerate all 12 placeholders (7 standard + 5 legacy `go-*`) or just the 7 standard?

**Method:** read `internal/templates/embed.go` for the `//go:embed` directives covering `builtin/agents/till-go/**`; read `copyAgentFiles` loop at `init_cmd.go:438-480`; cross-check live directory.

**Embed directives (`internal/templates/embed.go:84-90 + 98-102`):**
- 7 standard: `planning-agent.md`, `builder-agent.md`, `qa-proof-agent.md`, `qa-falsification-agent.md`, `research-agent.md`, `closeout-agent.md`, `commit-message-agent.md`.
- 5 legacy `go-*`: `go-builder-agent.md`, `go-planning-agent.md`, `go-research-agent.md`, `go-qa-proof-agent.md`, `go-qa-falsification-agent.md`.

**Live directory confirmation:** `ls internal/templates/builtin/agents/till-go/ | wc -l` = 12 entries.

**Loop semantics (`init_cmd.go:444-478`):**

```
srcDir := path.Join("builtin", "agents", group)            // → "builtin/agents/till-go"
entries, err := fs.ReadDir(templates.DefaultTemplateFS, srcDir)
...
for _, entry := range entries {
    if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") { continue }
    ...
}
```

`fs.ReadDir` against `embed.FS` returns every embedded path whose parent matches `srcDir`. No filtering by filename prefix. All 12 `.md` entries pass the suffix check and reach the write step.

**Verdict: REFUTED.** Group fallback yields the full 12 files. `TestInit_FreshDir_CopiesAllFiles` asserts the floor at `>= 7` (the SKETCH §11.1 standard count) so its assertion holds even if a future drop removes the 5 legacy `go-*` placeholders (which `internal/templates/embed.go:65-68` flags as transitional residue eligible for cleanup post-Drop-4c.7). The test does not assert == 12, which is the correct posture: legacy residue is documented as eligible for cleanup.

#### W2-D5-FF3 — `copyAgentFiles` FLAT writes (no group prefix)

**Attack:** builder claims FLAT — no `till-go/` prefix in destination. Verify by inspecting path construction.

**Method:** read `init_cmd.go:460-461`.

**Code:**

```
target := filepath.Join(agentsDir, entry.Name())
```

Where `agentsDir = filepath.Join(destDir, ".tillsyn", "agents")` (no group component) and `entry.Name()` is the bare basename returned by `fs.ReadDir` (e.g. `builder-agent.md`, not `till-go/builder-agent.md`). `entry.Name()` from `fs.DirEntry` against an `embed.FS` ReadDir result is documented to return the unqualified basename.

**Test corroboration:** `TestInit_FreshDir_CopiesAllFiles` at `init_cmd_test.go:371` asserts `os.Stat(filepath.Join(agentsDir, "builder-agent.md"))` succeeds with comment `"(FLAT copy required — no till-go/ prefix)"`.

**Verdict: REFUTED.** Destination is `<destDir>/.tillsyn/agents/<basename>` with no group prefix.

#### W2-D5-FF4 — `copyAgentsTOML` source path

**Attack:** confirm `copyAgentsTOML` reads `internal/templates/builtin/agents.example.toml` (NOT `agents.toml`).

**Method:** read `init_cmd.go:497`.

**Code:**

```
const srcPath = "builtin/agents.example.toml"
data, err := fs.ReadFile(templates.DefaultTemplateFS, srcPath)
```

Cross-checked against `internal/templates/embed.go:76` (`//go:embed builtin/agents.example.toml`) and confirmed the file exists at `internal/templates/builtin/agents.example.toml` (4.2K).

**Destination construction at L490:**

```
target := filepath.Join(destDir, "agents.toml")
```

The src is `agents.example.toml`; the dst is `agents.toml`. Verbatim per the builder claim.

**Verdict: REFUTED.** Source = `builtin/agents.example.toml`, dst = `<destDir>/agents.toml`.

#### W2-D5-FF5 — `ensureGitignore` first-line-only case

**Attack:** file is exactly `agents.local.toml\n` with NO preceding entries. A raw `bytes.Contains(data, []byte("\nagents.local.toml\n"))` MISSES this. Builder's line-iteration must HANDLE it. Verify by attempting a counterexample input.

**Method:** read `gitignoreLinePresent` at `init_cmd.go:568-576` and reason on the seeded test case at `init_cmd_test.go:458-478`.

**Code (`gitignoreLinePresent`):**

```
func gitignoreLinePresent(data []byte, want string) bool {
    sc := bufio.NewScanner(strings.NewReader(string(data)))
    for sc.Scan() {
        if strings.TrimSpace(sc.Text()) == want { return true }
    }
    return false
}
```

**Trace for input `data = "agents.local.toml\n"`, `want = "agents.local.toml"`:**
- `bufio.Scanner` (default `ScanLines`) yields one token: `"agents.local.toml"` (newline stripped).
- `strings.TrimSpace` returns `"agents.local.toml"`.
- Equal to `want` → returns true.
- No appended duplicate.

**Counterexample-attempt:** try `data = "agents.local.toml"` (no trailing newline). Scanner yields the same token; same TrimSpace; same equality; returns true. No duplicate.

**Try:** `data = "  agents.local.toml  \n"` (whitespace padding). TrimSpace strips → equals `want` → true.

**Try:** `data = "#agents.local.toml\n"` (commented out). TrimSpace yields `#agents.local.toml` ≠ `agents.local.toml` → false. The code would APPEND. This is the documented contract — a commented-out line is not the same as the line itself. Not a counterexample to the builder's claim; matches intent.

**Test corroboration:** `TestInit_GitignoreIdempotent` (init_cmd_test.go:458) seeds exactly `agents.local.toml\n` and asserts `countGitignoreLine == 1` after re-invoking `till init`. Test passes (verified via `mage ci`).

**Verdict: REFUTED.** Line-iteration correctly handles the first-line-only case the raw `bytes.Contains` form would miss.

#### W2-D5-FF6 — `ensureGitignore` trailing-newline corners

**Attack:** file ends with no `\n` → must append `\n` + `agents.local.toml\n`. File ends with `\n` → append `agents.local.toml\n`. Both cases tested?

**Method:** read `init_cmd.go:544-552` and the table-driven test at `init_cmd_test.go:484-529`.

**Code (`ensureGitignore` existing-file branch):**

```
body := data
if len(body) > 0 && body[len(body)-1] != '\n' {
    body = append(body, '\n')
}
body = append(body, []byte(gitignoreAgentsLocalLine+"\n")...)
```

**Trace 1 — file ends with `\n`** (input `"node_modules/\n.env\n"`):
- `body[-1] == '\n'` → skip newline append.
- Append `"agents.local.toml\n"` → final: `"node_modules/\n.env\nagents.local.toml\n"`.
- Single trailing newline; clean append.

**Trace 2 — file ends without `\n`** (input `"node_modules/\n.env"`):
- `body[-1] != '\n'` → append `'\n'` → `"node_modules/\n.env\n"`.
- Append `"agents.local.toml\n"` → final: `"node_modules/\n.env\nagents.local.toml\n"`.
- Same end-state as Trace 1; lines properly delimited.

**Test corroboration:** `TestInit_PreExistingGitignore_AppendsCleanly` is table-driven with two subtests: `trailing_newline` (seed `"node_modules/\n.env\n"`) and `no_trailing_newline` (seed `"node_modules/\n.env"`). Both subtests assert:
- pre-existing entries survive (`strings.Contains` for `"node_modules/"` and `".env"`),
- `agents.local.toml` appears exactly once,
- final body ends with `\n`.

Both subtests pass (verified via `mage ci`).

**Edge: empty file** (input `""`):
- `len(body) == 0` → skip newline-append (guard prevents OOB).
- Append `"agents.local.toml\n"` → final: `"agents.local.toml\n"`.
- Equivalent to the absent-file branch's `body = []byte(gitignoreAgentsLocalLine + "\n")`. Correct.

**Verdict: REFUTED.** Both trailing-newline cases tested explicitly via a table-driven subtest matrix. Empty-file edge handled by the `len(body) > 0` guard.

#### W2-D5-FF7 — `ensureGitignore` concurrent invocations

**Attack:** if two `till init` invocations run concurrently against the same `.gitignore`, can both write the line (duplicate)? `fsatomic.WriteFile` is atomic per call but doesn't serialize against other callers.

**Method:** reason on the read-compute-write window without a file lock.

**Race trace 1 — both see absent file:**
- Process A: `os.ReadFile` returns `fs.ErrNotExist`; computes `body = "agents.local.toml\n"`; writes.
- Process B: `os.ReadFile` returns `fs.ErrNotExist` (still — A's rename hasn't landed yet); computes same body; writes.
- The two `fsatomic.WriteFile` calls each write to a per-process tempfile then rename. Last rename wins. Final content = `"agents.local.toml\n"` either way. No duplicate.

**Race trace 2 — both see existing file WITHOUT the line** (seed `"node_modules/\n"`):
- A reads `"node_modules/\n"`; `gitignoreLinePresent` false; appends → body = `"node_modules/\nagents.local.toml\n"`; renames.
- B reads `"node_modules/\n"` (still — A's rename hasn't landed); same compute; renames.
- Final content = `"node_modules/\nagents.local.toml\n"` either way. No duplicate.

**Race trace 3 — A's rename completes before B reads:**
- A writes `"...\nagents.local.toml\n"`; renames.
- B reads `"...\nagents.local.toml\n"`; `gitignoreLinePresent` true; returns nil without writing.
- No duplicate.

**Race trace 4 — B's read straddles A's rename (TOCTOU window):** the data B reads is either A's pre-write content OR A's post-write content (`fsatomic.WriteFile` rename is atomic — POSIX guarantees this — so no half-written read). Either way, the write A and B produce is the same content (case 2 or case 3). No duplicate.

**Counterexample attempt — write-write conflict in mid-flight:** `fsatomic.WriteFile` writes to `<target>.tmp-<random>` then `os.Rename(tmp, target)`. Two processes use distinct tempfile suffixes (random hex). No tempfile collision. `os.Rename` is atomic on POSIX. No half-overwritten state.

**Verdict: REFUTED.** No duplicate appendable under the read-compute-write race. The atomic-rename + idempotent-compute combination produces the same final content regardless of interleaving. There is no file lock, but the operation is convergent: every interleaving lands on the same single-line-present steady state.

**NIT: there is a theoretical write-amplification (both processes do the work) but no correctness regression.** Worth noting but not a counterexample.

#### W2-D5-FF8 — `runInitPipeline` partial failure cleanup

**Attack:** `copyAgentFiles` succeeds but `copyAgentsTOML` fails. Are the per-agent `.md` files cleaned up? Or partial state survives?

**Method:** read `runInitPipeline` at `init_cmd.go:396-420` and reason on the contract.

**Code (sequential, no rollback):**

```
if _, _, err := copyAgentFiles(destDir, payload.Group); err != nil {
    return fmt.Errorf("till init: copy agent files: %w", err)
}
if _, _, err := copyAgentsTOML(destDir); err != nil {
    return fmt.Errorf("till init: copy agents.toml: %w", err)
}
if err := ensureGitignore(destDir); err != nil {
    return fmt.Errorf("till init: ensure .gitignore: %w", err)
}
```

If `copyAgentFiles` succeeds (12 files written) and `copyAgentsTOML` fails, the function returns the wrapped error and the 12 `.md` files remain on disk. No `defer`-based cleanup.

**Is this a counterexample?** The droplet's "Re-run safety (mandatory invariant)" clause documents this exact behavior: every write is gated by `os.Stat` skip-existing, so on next invocation the 12 files are SKIPPED and the pipeline resumes from `copyAgentsTOML`. This is the intentional "idempotent forward progress" design.

**Builder doc-comment at `init_cmd.go:387-395`** explicitly says: *"Re-run safety is mandatory: every file write is gated by an `os.Stat` pre-check so existing files are SKIPPED, never overwritten."* Re-run safety covers exactly the partial-failure recovery story.

**Test corroboration:** `TestInit_RerunSafety_NoOverwrite` (init_cmd_test.go:395) invokes `till init` twice and asserts byte-for-byte file equality across the two runs. Pre-existing files survive untouched across a re-invocation. The same invariant means a partial-failure re-run resumes correctly.

**Counterexample attempt — what if a future drop changes a placeholder body?** The skip-existing rule means a re-init does NOT pick up the new content. This is a deliberate user-data-preservation choice (the user may have edited their `.tillsyn/agents/*.md`). A `--force` flag is the documented future answer per SKETCH §9.3.

**Verdict: REFUTED (intentional behavior).** Partial-state survival on mid-pipeline failure is the documented design, justified by re-run safety. Recovery is "re-run `till init`" which is byte-for-byte idempotent over already-written files. NIT: a `--force` flag for explicit re-seeding would close the upgrade-placeholder gap, but that's a post-MVP refinement.

#### W2-D5-FF9 — D6-stub error wrapping semantics

**Attack:** `errors.New(...)` returns a plain error. Does `errors.Is` work against the literal string?

**Method:** read `init_cmd.go:419`.

**Code:**

```
return errors.New("till init: .mcp.json registration not yet wired (W2.D6)")
```

Each call to `errors.New` allocates a fresh `*errorString` pointer. `errors.Is(a, b)` for two `errors.New` calls with the same string returns FALSE (pointer comparison via the default `Is` chain, since `*errorString` does not implement `Is(target) bool`).

**Test contract check:** every test that asserts the D6 stub uses `strings.Contains(err.Error(), wantSubstring)` (init_cmd_test.go:57, 80, 105-111, 351). NONE use `errors.Is`. Test contract is decoupled from sentinel-identity.

**Counterexample attempt:** is any production caller expected to route on `errors.Is(err, someSentinel)` here? D5 is the file-copy pipeline; the D6 stub is the closeout. D6 will replace this `errors.New` with real `.mcp.json` registration code, so the stub literal is transient. No production caller depends on routing — it's a not-yet-implemented marker.

**Verdict: REFUTED.** Tests use substring matching, not `errors.Is`. The D6 stub is transient; D6 will replace it. No regression surface here. **NIT: if W2 ships pre-D6 to users (it shouldn't per droplet ordering), surfacing a non-`Is`-able error from a production-named path is mildly fragile. Mitigation is straightforward — wrap a package-level sentinel like `errInitMCPNotWired = errors.New(...)` and reference it. Refinement candidate, not blocking.**

#### W2-D5-FF10 — `mage ci` GREEN verification

**Attack:** run `mage ci` end-to-end; it must be GREEN with no W2-D5-attributable failures.

**Method:** `mage ci` from `main/` working directory.

**Result:**

```
Test summary
  tests: 3081
  passed: 3081
  failed: 0
  skipped: 0
  packages: 26
  pkg passed: 26
  pkg failed: 0
  pkg skipped: 0
[SUCCESS] All tests passed
  3081 tests passed across 26 packages.
[SUCCESS] Coverage threshold met
  All packages are at or above 70.0% coverage.
[SUCCESS] Built till from ./cmd/till
```

`cmd/till` package coverage: **76.0%** (well above 70.0% floor). All 3081 tests across 26 packages pass. Build succeeds.

**Working tree post-mage-ci:** `git status --short` → clean. No test residue.

**Verdict: REFUTED. mage ci is GREEN; no W2-D5-attributable failures.**

### Verdict table

| Hypothesis                          | Result    | Evidence                                                                                                                                                                       |
| ----------------------------------- | --------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| H1 t.Chdir audit                    | REFUTED   | FF1 — all 10 tests audited; 7 pipeline-driving tests chdir; 3 teatest-only tests need no chdir; `cmd/till/` working tree clean.                                                |
| H2 till-go group fallback (12 files) | REFUTED   | FF2 — embed.FS lists all 12; loop reads all 12; test asserts `>= 7` floor (correct posture for transitional legacy residue).                                                   |
| H3 FLAT writes (no group prefix)    | REFUTED   | FF3 — `target = filepath.Join(agentsDir, entry.Name())`, no group component; test asserts `builder-agent.md` at flat path.                                                     |
| H4 agents.example.toml source       | REFUTED   | FF4 — `srcPath = "builtin/agents.example.toml"`, dst = `agents.toml`; embed directive confirms.                                                                                |
| H5 first-line-only case             | REFUTED   | FF5 — `bufio.Scanner`-based line-iteration handles the case the raw `bytes.Contains` form misses; test seeds the exact case and asserts occurrences = 1.                       |
| H6 trailing-newline corners         | REFUTED   | FF6 — both `body[-1] == '\n'` and `body[-1] != '\n'` branches handled with guard; table-driven test covers both seeds; empty-file edge guarded by `len(body) > 0`.             |
| H7 concurrent invocations           | REFUTED   | FF7 — four race traces analyzed; atomic-rename + idempotent-compute → all interleavings converge on the same single-line-present steady state. NIT only.                       |
| H8 partial failure cleanup          | REFUTED   | FF8 — intentional "idempotent forward progress" design; re-run resumes correctly; re-run safety test confirms byte-for-byte preservation across re-invocation.                 |
| H9 D6-stub errors.Is semantics      | REFUTED   | FF9 — tests use substring match, not `errors.Is`; D6 will replace the stub; no production sentinel-routing dependency.                                                         |
| H10 mage ci GREEN                   | REFUTED   | FF10 — 3081/3081 tests pass; `cmd/till` 76.0% coverage; build succeeds; working tree clean post-run.                                                                            |

### Probes executed

- **`mage ci`** — GREEN. 3081 tests, 26 packages, coverage threshold met, build success.
- **Direct `Read` of `cmd/till/init_cmd.go` lines 1-600** — full file inspection of the pipeline + helpers + validate.
- **Direct `Read` of `cmd/till/init_cmd_test.go` lines 1-552** — full file inspection for the t.Chdir audit; classified each `Test*` function.
- **Direct `Read` of `internal/templates/embed.go` lines 1-282** — confirmed `agents.example.toml` and 12 till-go `.md` embed directives.
- **Live `ls` of `internal/templates/builtin/agents/till-go/`** — 12 files; cross-referenced against the 12 embed lines (7 standard + 5 legacy `go-*`).
- **Live `ls` of `internal/templates/builtin/`** — confirmed `agents.example.toml` exists at the expected path (4.2K).
- **Live `ls cmd/till/`** — confirmed NO test-residue artifacts (no `.tillsyn/`, no `.gitignore`, no `agents.toml`).
- **`git status --short`** — clean post-`mage ci`.
- **`rg -n "runInitPipeline|copyAgentFiles|copyAgentsTOML|ensureGitignore|runInitJSON"` against `cmd/till/`** — 27 hits in 2 files (init_cmd.go + init_cmd_test.go); no leakage outside the pipeline's two files.
- **Hylla `hylla_search_keyword`** for `runInitPipeline` / `copyAgentFiles` against `github.com/evanmschultz/tillsyn@main` — returned empty (the W2.D5 surface is uncommitted; pre-cascade-ingest staleness is the documented norm).

### Summary

**Verdict: PASS — no W2.D5-attributable CONFIRMED counterexamples.** All 10 attack vectors REFUTED with concrete code + test + run evidence.

**Soundness summary:** the pipeline composes three idempotent helpers with `os.Stat` skip-existing guards. `copyAgentFiles` correctly enumerates 12 till-go placeholders via `fs.ReadDir` + suffix filter, writes flat under `.tillsyn/agents/<basename>` using `fsatomic.WriteFile` (atomic write-temp+rename, mode 0o644). `copyAgentsTOML` reads the embedded `builtin/agents.example.toml` and writes to `<destDir>/agents.toml`. `ensureGitignore` uses `bufio.Scanner` line-iteration (via `gitignoreLinePresent`) to handle the W2-FF10 round-2 LOCKED first-line-only case the raw `bytes.Contains` form misses; trailing-newline handling guards both `body[-1] == '\n'` and `body[-1] != '\n'`. The t.Chdir audit found every pipeline-exercising test sandboxed into `t.TempDir()`; the three teatest-only tests correctly skip chdir because the bubbletea model never invokes the pipeline. `cmd/till/` working tree is clean post-`mage ci`. The W2.D5 droplet is sound, well-tested, and re-run safe.

**Severity breakdown:**

- **0 CONFIRMED counterexamples** attributable to W2.D5.
- **3 NIT-severity findings** (all refinement candidates, not blocking):
  - **W2-D5-FF7 NIT** — concurrent invocations do redundant work (read-compute-write race amplifies effort), but converge on correct single-line state. A file lock would close this; not worth the complexity for `till init` (a one-shot setup command).
  - **W2-D5-FF8 NIT** — no `--force` flag for re-seeding placeholders after the user has edited their `.tillsyn/agents/*.md`. Documented as post-MVP refinement per SKETCH §9.3.
  - **W2-D5-FF9 NIT** — D6 stub uses ad-hoc `errors.New` instead of a package-level sentinel; tests don't depend on `errors.Is` so this is transient (D6 replaces the stub). Mildly fragile if W2 ships pre-D6 to users — refinement candidate.

**Routing to orchestrator:**

1. **W2.D5 droplet itself: PASS — no respawn required for W2.D5 work.**
2. **No code changes recommended for W2.D5 closure.**
3. **Refinement candidates (post-MVP, not blocking):**
   - W2-D5-FF7 — concurrent-write convergence is correct but does redundant work; flock around the read-compute-write cycle would tighten it.
   - W2-D5-FF8 — `--force` flag for `till init` to re-seed placeholders after user edits.
   - W2-D5-FF9 — promote the D6 stub error to a package-level sentinel for `errors.Is` symmetry once W2.D6 lands real code.

### Hylla Feedback

- **Query:** `hylla_search_keyword` for `runInitPipeline` and `copyAgentFiles` against `github.com/evanmschultz/tillsyn@main` (two separate queries).
  - **Missed because:** the W2.D5 symbols are uncommitted (W2.D5 work is on the working tree, not yet in any Hylla snapshot). Same pre-cascade-ingest staleness window documented across multiple prior rounds in this file (W3.D2 round-2, W3.D4, W2.D1, W3.D5, W2.D8).
  - **Worked via:** direct `Read` of `cmd/till/init_cmd.go` (full file, 600 lines), `cmd/till/init_cmd_test.go` (full file, 552 lines), `internal/templates/embed.go` (full file, 282 lines), live `ls` of `internal/templates/builtin/` and `internal/templates/builtin/agents/till-go/` for ground-truth file counts, `git status --short` for working-tree state, `rg -n` for cross-file caller audit, and `mage ci` for dynamic verification.
  - **Suggestion:** non-blocking — same recurring suggestion as prior rounds. When enrichment is in flight or the queried symbol is in an uncommitted change, the keyword tool could optionally fall back to a HEAD-aware index OR surface a hint pointing the caller at the staleness window so they can immediately reach for `Read` rather than burning a Hylla call. Multiple droplets in this drop have raised the same suggestion; aggregate at drop-end Hylla refinements rollup.

---

## Droplet 4c.6.W6.D4 — Round 1

**Reviewer:** go-qa-falsification-agent (subagent, doc-only mode).
**Date:** 2026-05-11.
**Droplet:** `4c.6.W6.D4 — Update SPAWN_PIPELINE.md + CLI_ADAPTER_AUTHORING.md for --bare-collapsed isolation`.
**Artifacts under attack:**
- `SPAWN_PIPELINE.md` (MODIFIED — `## Two Plugin Paths` (old 24-31) renamed `## Plugin Paths and Isolation`; `--bare`-collapsed isolation paragraph added; W3 gap-closure paragraph added).
- `CLI_ADAPTER_AUTHORING.md` (MODIFIED — new `## Isolation Discipline for New Adapters` section inserted before `## Adapter Authoring Checklist`).

### Verdict

**PASS** — no CONFIRMED counterexample produced. Four NITs raised below (paraphrase-as-quote attribution, MCP user-config path inaccuracy, SPAWN_PIPELINE.md argv elision, settings-skip attribution slip). None block droplet closure.

### Attacks attempted

1. **Source-vs-claim drift vs `RESEARCH/ISOLATION_ENFORCEMENT_FIX.md` §D.5.** MITIGATED — §D.5 prescribes:
   - `SPAWN_PIPELINE.md:24-31` — "keep the two-paths section but add a note: 'Tillsyn ships `--bare`; under `--bare` Path B is disabled by Claude Code itself. The bundle's plugin (Path A) is the SOLE source for agents, settings, and MCP. The two-paths model is informational for non-bare callers; Tillsyn's adapters never ship without `--bare`.'"
   - The rewrite line 31 + line 33 substantively match this prescription. Path A / Path B labels are preserved; the `--bare` note is added; the "informational for non-bare callers" framing is preserved verbatim in spirit at line 33.

2. **Anthropic-docs alignment for `--bare` behavior.** MOSTLY MITIGATED with one NIT.
   - Verified via Context7 against `/websites/code_claude` (`https://code.claude.com/docs/en/glossary`, `headless`, `cli-reference`, `env-vars`). Anthropic docs canonical list: hooks, skills, plugins, MCP servers, auto memory, CLAUDE.md.
   - The rewrite adds **"agent auto-discovery"** to the skipped list. This is INFERRED, not directly stated by Anthropic — agents live in plugins (skipped) and in `.claude/agents/` / `~/.claude/agents/` (not explicitly named in `--bare` glossary). However, the Anthropic headless page states *"Context can be provided using flags like ... `--agents`, `--plugin-dir` ..."* in bare mode, strongly implying agent auto-discovery is skipped (and explicit `--agents` is the bare-mode entry point). The inference is defensible. Not a counterexample.
   - **NIT-D4-1 (paraphrase quoted as verbatim Anthropic):** SPAWN_PIPELINE.md line 31 and CLI_ADAPTER_AUTHORING.md line 213 both place in quotation marks: *"bare mode never reads them — only flags you pass explicitly take effect."* Anthropic's actual phrasing per the Glossary entry is *"Only explicitly provided flags are considered"* and per the headless page is *"only applies explicitly passed flags."* The substance is faithful but the quotation marks falsely attribute verbatim phrasing. Either drop the quotation marks and paraphrase explicitly, or quote Anthropic's actual wording.
   - **NIT-D4-2 (settings-skip attribution slip):** SPAWN_PIPELINE.md line 31 reads "Under `--bare`, Claude Code skips ... and user/project settings entirely." Per Anthropic docs, `--bare` itself does NOT explicitly skip settings — settings exclusion comes from `--setting-sources ""` (cited correctly in CLI_ADAPTER_AUTHORING.md line 215). The net effect under Tillsyn's full argv is correct (everything is skipped), but attributing settings-skipping to `--bare` rather than to `--setting-sources ""` is technically inaccurate.

3. **Latent contradictions between the two new sections.** MOSTLY MITIGATED with one NIT.
   - Both sections use Path A / Path B vocabulary consistently. Both describe `--bare` as collapsing the choice. Both name the same four isolation flags (CLI_ADAPTER_AUTHORING.md is more explicit at lines 207-208 with all six argv elements including `--settings` and `--mcp-config`; SPAWN_PIPELINE.md line 31 lists only four).
   - **NIT-D4-3 (SPAWN_PIPELINE.md argv elision):** SPAWN_PIPELINE.md line 31 lists `"Tillsyn always spawns with --bare --plugin-dir <bundle>/plugin --setting-sources \"\" --strict-mcp-config"` — omits `--settings` and `--mcp-config` which CLI_ADAPTER_AUTHORING.md line 207 lists as part of the "correct minimal shape." This is incomplete listing rather than contradiction, and SPAWN_PIPELINE.md's lower-detail framing is acceptable given the broader doc is a high-level pipeline architecture overview. Consider adding `--settings ... --mcp-config ...` to SPAWN_PIPELINE.md line 31 for parity, OR rewording to "...spawns with `--bare`, `--plugin-dir`, `--setting-sources ""`, `--strict-mcp-config`, `--settings`, and `--mcp-config` flags..." with brief argv-shape pointer to CLI_ADAPTER_AUTHORING.md.

4. **Implicit retraction of useful Path A vs Path B info.** MITIGATED — both Path A and Path B remain documented at SPAWN_PIPELINE.md lines 28-29, retaining the vocabulary that future adapter authors need to understand the Claude Code plugin loader. The "informational for non-bare callers" framing at line 33 preserves the path-vocabulary's utility for adapters not built on Tillsyn.

5. **Cross-reference rot across repo.** MITIGATED — `git grep -i "two paths"` against the authoritative docs (`SPAWN_PIPELINE.md` `CLI_ADAPTER_AUTHORING.md` `AGENTS.md` `AGENT_CASCADE_DESIGN.md` `AGENTS_CONFIG.md` `CASCADE_METHODOLOGY.md` `CLAUDE.md` `WIKI.md` `README.md` `GDD_METHODOLOGY.md` `CONTRIBUTING.md`) returns zero hits. `git grep "Path B"` hits only the consistent post-rewrite framing: `SPAWN_PIPELINE.md` (lines 29, 31, 35), `CLI_ADAPTER_AUTHORING.md` (line 213), and `CASCADE_METHODOLOGY.md` (line 155 — uses identical `--bare` framing: *"`--bare` flag tells Claude Code to skip every normal context source (Path B / system CLAUDE.md / skills / project CLAUDE.md / hooks / `~/.claude/settings.json` / system plugins)"*). All hits are coherent with the new framing. Workflow / drop history dirs retain the old framing as audit trail (expected, intentional).
   - **NIT-D4-4 (MCP user-config path inaccuracy):** CLI_ADAPTER_AUTHORING.md line 216 reads *"`--strict-mcp-config` — restricts MCP to `--mcp-config` argument only, ignoring project `.mcp.json` and user `~/.claude/.mcp.json`."* Per Anthropic docs, user-scoped MCP servers are stored in `~/.claude.json` (not `~/.claude/.mcp.json`). The conceptual claim is correct (`--strict-mcp-config` disregards all other MCP sources), but the specific user-config path cited is wrong. Replace `~/.claude/.mcp.json` with `~/.claude.json`.

6. **Builder over-claim — `mage ci` green.** MITIGATED (verified). Local `mage ci` run by this falsification reviewer: `3085 tests passed across 26 packages, all packages at or above 70.0% coverage, build succeeded`. Builder's claim verified.

7. **W3 reality vs claim ("empty bundle body").** MITIGATED.
   - W3 PLAN.md (`workflow/drop_4c_6/DROP_4c.6.W3_BUNDLE_AND_ISOLATION/PLAN.md`) §Scope lines 17-23 enumerates W3 actually delivering: D.1 (3-tier resolver replacing the F.7.3b stub — the load-bearing isolation closure), D.2 (frontmatter strip), D.3 (defense-in-depth env vars), D.4 (post-render validator), D.5 (doc-comment correction at `render.go:307-319` ONLY — SPAWN_PIPELINE.md rewrite explicitly OUT OF SCOPE for W3, OWNED by W6.D4 per ROUND-2 HF3).
   - SPAWN_PIPELINE.md line 35's claim: *"The gap was that the bundle's `plugin/agents/<name>.md` body was a one-liner stub ... W3 replaces the stub with full embedded-default agent content."* — accurately describes W3.D.1, which `ISOLATION_ENFORCEMENT_FIX.md` §D.6 identifies as the **primary** isolation closure (*"the bug is purely that the bundle is empty (D.1) plus residual confusion (D.5). D.2/D.3/D.4 are hardening, not isolation closure."*).
   - W6.D4 acceptance bullet (PLAN.md line 319) scopes the rewrite to the empty-bundle-body gap by design. The rewrite is faithful to that acceptance scope. Under-claiming W3's hardening adjuncts (D.2/D.3/D.4) is acceptable per the source-of-truth's own framing of D.1 as the load-bearing closure and D.2/D.3/D.4 as hardening. NOT a counterexample.

### Findings

- **NIT-D4-1** — paraphrase-as-verbatim-quote attribution in SPAWN_PIPELINE.md line 31 + CLI_ADAPTER_AUTHORING.md line 213. Drop the quotation marks OR substitute Anthropic's actual wording (*"Only explicitly provided flags are considered"*). Non-blocking; doc-fidelity refinement.
- **NIT-D4-2** — settings-skip attribution to `--bare` rather than `--setting-sources ""` in SPAWN_PIPELINE.md line 31. The net Tillsyn-argv effect is correct; the standalone `--bare` attribution slips technically. Non-blocking; reword to attribute settings-skip to `--setting-sources ""` specifically.
- **NIT-D4-3** — SPAWN_PIPELINE.md line 31 omits `--settings` and `--mcp-config` from the "always spawns with" argv listing while CLI_ADAPTER_AUTHORING.md line 207 carries the full six-flag shape. Non-blocking; align for parity OR explicitly defer detail to CLI_ADAPTER_AUTHORING.md cross-reference.
- **NIT-D4-4** — CLI_ADAPTER_AUTHORING.md line 216 cites user MCP config at `~/.claude/.mcp.json`; Anthropic docs confirm user-scoped MCP servers store in `~/.claude.json`. Non-blocking; one-token correction.

### Notes

- All four NITs are cosmetic doc-fidelity refinements. None falsify the substantive isolation framing.
- The rewrite correctly preserves Path A / Path B vocabulary for future adapter authors.
- Cross-reference rot check: `CASCADE_METHODOLOGY.md` line 155 already uses the same `--bare`-collapsed framing; no rot across authoritative docs.
- Builder's `mage ci` green claim independently verified by this reviewer: 3085 tests passed, all packages ≥ 70% coverage.
- W6.D4 acceptance bullet `git grep "two paths"` → zero hits in spawn / adapter docs confirmed.
- W3 scope under-claim is intentional per the W6.D4 acceptance bullet and matches `ISOLATION_ENFORCEMENT_FIX.md` §D.6's own framing of D.1 as the load-bearing closure. Not a counterexample.

### Hylla Feedback

N/A — task touched non-Go files only (SPAWN_PIPELINE.md, CLI_ADAPTER_AUTHORING.md, W3 PLAN.md, RESEARCH/*.md, BUILDER_WORKLOG.md, PLAN.md). Hylla is Go-only today; non-Go falsification used `Read` + `git grep` + Context7 (for Anthropic `--bare` semantics) per CLAUDE.md "Hylla doesn't cover non-Go files" rule.

---

## Droplet 4c.6.W2.D6 — Round 1

**Reviewer:** go-qa-falsification-agent (subagent, opus).
**Date:** 2026-05-11.
**Droplet:** `4c.6.W2.D6 — .mcp.json optional registration`.
**Files attacked:** `cmd/till/init_cmd.go`, `cmd/till/init_cmd_test.go`.

### Verdict

**FAIL** — one CONFIRMED counterexample produced (FF1: server-entry round-trip data loss against HTTP/SSE transport entries) plus one acceptance-deviation NIT (TUI mode never prompts for MCP; hardwires false). Builder must respawn to fix FF1 before W2.D6 can close. NIT2 (TUI confirm prompt) is a PLAN.md D6 acceptance-bullet miss that should either be (a) implemented, or (b) explicitly deferred by an orchestrator-level decision recorded in the worklog.

### Attacks attempted

1. **Schema-mismatch hazards (Context7 schema re-verification).** PARTIALLY-MITIGATED becomes FF1.
   - Re-queried Context7 `/websites/code_claude` independently. Confirmed:
     - `mcpServers` is the top-level key (matches builder).
     - For stdio: `type` is OPTIONAL (default `stdio`), `command` required, `args` + `env` optional (matches builder's struct).
     - **But Claude Code's `.mcp.json` also supports HTTP and SSE transports as first-class server entries:** `claude mcp add --transport http <name> <url> --scope project` writes entries like `{"type":"http","url":"https://..."}` (no `command` field).
   - The builder's `mcpServerEntry` has ONLY `{Command, Args, Env}` — `type`, `url`, `headers` are not modeled.
   - **This becomes FF1 below (data loss on existing HTTP/SSE entries).**

2. **Schema-drift counterexample — HTTP/SSE round-trip.** **CONFIRMED — FF1.**
   - Reproduction: seed `<cwd>/.mcp.json` with the canonical Claude Code project HTTP entry from `/websites/code_claude` docs:
     ```json
     {
       "mcpServers": {
         "notion": {"type": "http", "url": "https://mcp.notion.com/mcp"}
       }
     }
     ```
   - Run `till init --json '{"name":"foo","group":"till-go","mcp":true}'`.
   - In `registerMCPJSON` lines 669-672 the file is `json.Unmarshal`'d into `mcpJSONFile` whose value type is `map[string]mcpServerEntry`. Since `mcpServerEntry` has only `Command/Args/Env`, the `type` and `url` fields are silently dropped. Notion's entry becomes `{Command:"", Args:nil, Env:nil}`.
   - On re-marshal (line 681), the entry is written back as `{"command": ""}` (no `omitempty` on `Command`). The notion HTTP entry is **destroyed** — both its transport type and URL are lost.
   - Test `TestInit_MCPJSON_AppendsToExisting` only seeds a stdio entry `{"other-server": {Command: "/usr/local/bin/other"}}` which the struct CAN round-trip, so the existing test suite does NOT catch this.
   - **PLAN.md D6 acceptance bullet "TestInit_MCPJSON_AppendsToExisting: existing `.mcp.json` with another server entry; re-run adds `tillsyn` without removing the other"** is FALSIFIED for the HTTP/SSE case — `--scope project` HTTP/SSE entries authored by `claude mcp add` are removed/corrupted by `till init`.
   - **Severity: must-fix.** A dev who has used `claude mcp add --transport http ... --scope project` for any team-shared MCP server (Notion, PayPal, Stripe, internal API) will have those entries destroyed by `till init --mcp true`. Silent destruction is worse than a hard error.
   - **Fix hint:** unmarshal into `map[string]json.RawMessage` (preserves unknown shape), or `map[string]map[string]any`, and only deserialize the entry under check (`tillsyn` key). On write, marshal `RawMessage` entries verbatim and only re-marshal the new `tillsyn` entry through the typed struct.

3. **JSON-decode safety / top-level field preservation.** ACCEPTED-AS-NIT.
   - The builder's `mcpJSONFile` struct has only `MCPServers`. Any sibling top-level keys (Claude Code emits no such fields today per the docs surveyed, but the schema is open-ended) would be dropped on re-marshal.
   - Builder acknowledged this in the worklog ("if the existing file has extra top-level fields beyond `mcpServers`, those WILL be dropped on re-marshal. This is acceptable for Drop 4c.6's scope").
   - Routing: same fix as FF1 (use a `RawMessage` envelope at the top level too) closes both. Listing as NIT-3 below, not a separate counterexample, because FF1's fix-shape naturally covers it.

4. **JSON-key ordering / atomicity.** MITIGATED.
   - `encoding/json` Marshal sorts map keys alphabetically (verified via Context7 against `/golang/website` — documented Go behavior). `map[string]mcpServerEntry` re-marshal is therefore deterministic across runs. No idempotency-breaking diff from key reordering.
   - `TestInit_MCPJSON_Idempotent` (lines 653-704) asserts a re-run with an existing `tillsyn` entry leaves the file byte-for-byte unchanged via early-return at line 678 (skip before any marshal). The test does not exercise the marshal path on re-run, but the early-return is sound — no reordering risk in the idempotent case.

5. **`exec.LookPath` cross-system behavior.** MITIGATED.
   - `exec.LookPath("till")` searches `$PATH`; if found, returns the absolute path of the first hit. On the dev's macOS box `~/.local/bin/till` would commonly be on `$PATH` so `LookPath` returns the same absolute path the fallback would synthesize.
   - The fallback (`init_cmd.go:657-661`) uses `os.UserHomeDir()` + `filepath.Join` to produce an absolute path — NOT a literal `~/.local/bin/till` string. Claude Code reads an absolute path, not a tilde — so this is the correct shape.
   - The function does not verify the binary exists at the fallback path. That is correct — the worklog notes the dev may install `till` after running `init`, and the registration is still valid once `mage install` lands the binary.

6. **Tilde expansion.** MITIGATED.
   - Verified via direct read of `init_cmd.go:657-661`: builder uses `os.UserHomeDir()` + `filepath.Join(home, ".local", "bin", "till")`. The written path is absolute (e.g. `/Users/evanschultz/.local/bin/till`). No literal-tilde footgun.

7. **TUI confirmation contract.** **CONFIRMED — NIT2 (acceptance-bullet miss).**
   - PLAN.md D6 acceptance line 178: *"TUI mode: confirms with the user before mutating (yes/no prompt). JSON mode: respects the `mcp` boolean."*
   - The builder ships TUI mode that **never prompts** — `init_cmd.go:229` hardwires `m.finalPayload.MCP = false // TUI default per droplet acceptance`. The comment cites "droplet acceptance" but the droplet acceptance explicitly says to PROMPT, not to default-false.
   - The TUI walk has no step between group selection and pipeline dispatch. A user who wants MCP registration via TUI mode has no path to it — they must use `--json '{"mcp":true}'`.
   - This is an acceptance-deviation, not a counterexample to JSON-mode correctness, so the entire MCP feature still works via JSON mode. But PLAN.md's bullet is unfulfilled.
   - **Disposition options:** (a) builder respawns and adds a third TUI step (`initTUIStepMCP`) with a y/n prompt; (b) orchestrator records an explicit decision in `BUILDER_WORKLOG.md` that the TUI prompt is deferred and updates the PLAN.md acceptance to reflect "TUI mode: MCP defaults to false; users opt-in via JSON mode" (and lists the missing TUI prompt as a refinement).
   - Routing: surface to orchestrator for the disposition call. Tied to FF1 for the respawn pass — if a respawn happens for FF1, fold in the TUI prompt at the same time.

8. **D7-stub forward-plumb correctness.** MITIGATED.
   - Verified by direct `Read` of `init_cmd.go:423`: literal is `"till init: project-DB record creation not yet wired (W2.D7)"`. Byte-for-byte match against the worklog spec and PLAN.md D7 stub contract.
   - Tests `TestInit_BareInvocation_ReturnsTUIStubError` line 61, `TestInit_JSONInvocation_RoutesToValidParse` line 86, `TestInit_FreshDir_CopiesAllFiles` line 360, `TestInit_MCPJSON_*` (4 tests) all assert this exact substring. Forward-plumb is sound.

9. **Test residue.** MITIGATED.
   - `git status --porcelain cmd/till/` after `mage ci` shows only the two declared modified files (`cmd/till/init_cmd.go`, `cmd/till/init_cmd_test.go`). No `.tillsyn/`, no stray `.gitignore`, no `agents.toml`, no `.mcp.json` residue. The chdir-into-tempdir discipline is followed across every test that exercises the pipeline.
   - Verified via `git status --porcelain cmd/till/` post-`mage ci` run.

10. **Existing-tests update correctness.** MITIGATED.
    - `git grep -n "registration not yet wired" cmd/till/` → ZERO hits. `git grep -n "W2.D6" cmd/till/` → ZERO hits. No test still asserts the old D6-stub literal.
    - Worklog claim of "7 existing assertions updated" cross-checked against `git grep -n "W2.D7" cmd/till/init_cmd_test.go` → 7 hits at lines 61, 86, 116, 121, 360, 571, 622, 673, 712 (9 hits actually — 7 file-level updated assertions + 2 D7 table-test entries in the table-driven block). The "7" in the worklog refers to the assertion sites that previously asserted the D6 stub; the table-driven additions are net-new lines. All previously-D6-asserting sites now assert D7. Sound.

11. **CONSUMER-TIE test scope.** MITIGATED.
    - All four new D6 tests (`TestInit_MCPJSON_FreshFile`, `TestInit_MCPJSON_AppendsToExisting`, `TestInit_MCPJSON_Idempotent`, `TestInit_MCPJSON_OptOut`) invoke `run(context.Background(), []string{"--app", "tillsyn-init", "init", "--json", "..."}, &out, io.Discard)` end-to-end. None call `registerMCPJSON` directly. CONSUMER-TIE contract from W2-FF6 ROUND-2 / D7.5's W2-FF3 symmetric form is met.
    - `TestInit_MCPJSON_FreshFile` line 569 specifically routes through `runInitJSONInTempDir` → `run(...)` → cobra → `initCmd.RunE` → `runInitJSON` → `runInitPipeline` → `registerMCPJSON`. Wiring is exercised.

12. **`mage ci` claim verification.** MITIGATED.
    - This reviewer independently ran `mage ci` from a clean working tree.
    - Output (tail): `3085 tests passed across 26 packages, all packages at or above 70.0% coverage. Build succeeded`.
    - `cmd/till` coverage: 75.9% (above 70% gate). `internal/fsatomic` coverage: 72.0%.
    - Builder's claim verified.

13. **fsatomic atomicity.** MITIGATED.
    - `fsatomic.WriteFile` uses `os.CreateTemp(filepath.Dir(target), filepath.Base(target)+".tmp-*")` so the temp file lands in the same directory as the target. `os.Rename` is atomic on POSIX when source + target are on the same filesystem, which is guaranteed by the same-directory invariant.
    - `target` in `registerMCPJSON` is `filepath.Join(destDir, mcpJSONFileName)` — relative-vs-absolute is not a footgun because `destDir` is `os.Getwd()` (absolute) from `runInitPipeline:401`.
    - No relative-vs-absolute mismatch.

14. **Re-run safety edge case — malformed existing `.mcp.json`.** ACCEPTED-AS-NIT (NIT3).
    - PLAN.md D6 acceptance line 176: *"reads existing `<destDir>/.mcp.json` (if present), parses it..."*
    - `registerMCPJSON:670` returns a wrapped error `fmt.Errorf("parse %q: %w", target, unmarshalErr)` on JSON parse failure. No crash, no silent overwrite. The error bubbles through `runInitPipeline:417` as `"till init: register .mcp.json: parse \"<path>\": ..."`.
    - This is the safe choice — corrupted user MCP config is preserved untouched. The dev sees a clear error and can fix the file manually.
    - **NIT3:** the error message does not suggest a remediation (e.g., *"existing .mcp.json is malformed; fix or remove it and re-run"*). A help-style suffix would improve dev UX. Non-blocking; refinement.

### Findings

- **FF1 (CRITICAL — must-fix counterexample):** server-entry round-trip data loss. `mcpServerEntry` cannot represent HTTP/SSE/SDK transport entries that Claude Code authors via `claude mcp add --transport http ... --scope project`. On `till init --mcp true` against any cwd that already contains a project-scoped HTTP/SSE server, that server's `type` + `url` (+ `headers`) fields are silently dropped and the entry is re-marshaled as `{"command":""}`. Severity: user-visible data loss. Mitigation: unmarshal entries into `map[string]json.RawMessage` and only typed-deserialize the `tillsyn` key. Tests must be expanded to cover an HTTP-transport seed.

- **NIT1 (acceptance miss):** TUI mode does not implement the y/n MCP confirmation prompt required by PLAN.md D6 acceptance line 178. Builder hardwires `m.finalPayload.MCP = false` at `init_cmd.go:229`. Either ship the prompt as `initTUIStepMCP` or update PLAN.md to formally defer the TUI MCP prompt.

- **NIT2 (top-level field drop, acknowledged by builder):** sibling top-level keys to `mcpServers` are dropped on re-marshal. Folds into FF1's fix-shape (top-level envelope as `RawMessage` too).

- **NIT3 (error UX):** malformed-`.mcp.json` error returns a bare `json.Unmarshal` wrap with no remediation hint. Non-blocking; refinement.

### Notes

- **Verdict standard.** A user-visible data loss bug (FF1) on a documented Claude Code workflow (project-scoped HTTP/SSE entries) is a hard counterexample regardless of whether the existing test suite passes — the test suite simply doesn't cover the case. The PLAN.md D6 acceptance bullet "TestInit_MCPJSON_AppendsToExisting: ... re-run adds `tillsyn` without removing the other" is FALSE for HTTP/SSE "other" servers.
- **Recommended fix shape.** Replace the typed top-level + typed inner unmarshal with a two-level `RawMessage` envelope. Pseudocode:
  ```go
  type mcpJSONFileRaw struct {
      MCPServers map[string]json.RawMessage `json:"mcpServers"`
      // any other top-level keys preserved verbatim via embedded RawMessage map
  }
  ```
  Then on write, only re-serialize the `tillsyn` entry through `mcpServerEntry`; copy all other `RawMessage` values verbatim. Add a test `TestInit_MCPJSON_PreservesHTTPTransport` that seeds an HTTP entry and asserts round-trip fidelity.
- **TUI prompt deferral.** If the orchestrator decides to defer the TUI prompt, ensure the deferral is recorded in `BUILDER_WORKLOG.md` AND the PLAN.md D6 acceptance bullet is updated; otherwise the acceptance bullet remains a tripwire for future plan-QA.
- **mage ci**: independently verified green (3085/3085, all packages ≥ 70%).
- **Builder discipline observations**: TDD red→green cycle documented, schema source cited with Context7 benchmark score, atomic-write discipline consistent, FLAT copy invariant preserved, no test residue, CONSUMER-TIE contract met. The single missed surface is the HTTP/SSE transport shape — schema verification stopped at "stdio shape" without probing for additional transport types.
- **Attack table** below.

### Attack family table

| Attack family                            | Result    | Notes |
| ---------------------------------------- | --------- | ----- |
| Schema-mismatch (Context7 re-verify)     | CONFIRMED | FF1 — HTTP/SSE transports unmodeled. |
| Schema-drift round-trip                  | CONFIRMED | FF1 — `mcpServerEntry` drops `type` + `url` + `headers` fields. |
| Top-level unknown-field preservation     | NIT       | Folds into FF1 fix-shape. |
| JSON-key ordering / atomicity            | REFUTED   | Go marshal sorts map keys; idempotent path early-returns before marshal. |
| `exec.LookPath` cross-system             | REFUTED   | Returns absolute path; fallback uses `os.UserHomeDir()` join. |
| Tilde expansion                          | REFUTED   | `filepath.Join` produces absolute path; no literal tilde. |
| TUI confirmation contract                | CONFIRMED | NIT1 — PLAN.md D6 acceptance line 178 unfulfilled. |
| D7-stub forward-plumb byte-equality      | REFUTED   | Exact literal match `"till init: project-DB record creation not yet wired (W2.D7)"`. |
| Test residue                             | REFUTED   | `git status --porcelain cmd/till/` clean post-`mage ci`. |
| Existing-tests-update completeness       | REFUTED   | `git grep` for old stub returns zero hits; 7 prior assertions updated cleanly. |
| CONSUMER-TIE test scope                  | REFUTED   | All 4 new tests route through `run(...)` end-to-end; cobra wiring exercised. |
| `mage ci` claim verification             | REFUTED   | Independently verified GREEN (3085/3085, ≥70% coverage). |
| fsatomic atomicity                       | REFUTED   | Same-dir temp + rename; `destDir` is absolute from `os.Getwd()`. |
| Re-run safety on malformed file          | NIT       | Returns error (correct); error message lacks remediation hint. |

### Hylla Feedback

- **Query:** `hylla_search` + `hylla_search_keyword` were NOT queried this round. Falsification of W2.D6 centered on (a) shipped uncommitted code (which Hylla cannot index pre-reingest), and (b) Claude Code `.mcp.json` external semantics (which require Context7, not Hylla). Direct `Read` on `cmd/till/init_cmd.go` + `cmd/till/init_cmd_test.go` + `workflow/drop_4c_6/DROP_4c.6.W2_TILL_INIT/PLAN.md` + `workflow/drop_4c_6/BUILDER_WORKLOG.md`, `git grep` for residue, and Context7 against `/websites/code_claude` and `/golang/website` covered all evidence needs. No Hylla miss to report.
- **Suggestion:** none new. Recurring suggestion from prior rounds (HEAD-aware fallback when enrichment is mid-flight) applies but did not bite this round because the falsification did not need committed-symbol queries.

---

## Droplet 4c.6.W2.D6 — Round 2

**Reviewer:** go-qa-falsification-agent (subagent, opus).
**Date:** 2026-05-11.
**Droplet:** `4c.6.W2.D6 — .mcp.json optional registration` — round-2 attacks against the `json.RawMessage` envelope refactor.
**Files attacked:** `cmd/till/init_cmd.go` (registerMCPJSON rewrite), `cmd/till/init_cmd_test.go` (2 new regression tests + 4 updated), `workflow/drop_4c_6/DROP_4c.6.W2_TILL_INIT/PLAN.md` (D6 row + ROUND-2 deferral block).

### Verdict

**FAIL** — one new CONFIRMED counterexample produced (FF2: nil-map panic on `{"mcpServers": null}`) plus three NITs that did NOT exist before round-2 (key-reorder UX drift, stale-tmp-file accumulation, doc-claim drift on byte-equivalence). Round-1 FF1 (HTTP/SSE) is correctly closed; round-1 NIT2 (top-level extras) is correctly closed; round-1 NIT1 (TUI deferral) is correctly handled in PLAN.md.

FF2 is a runtime PANIC (not a clean error) on a non-malformed JSON input. A user whose `.mcp.json` is `{"mcpServers": null}` (which can arise from a tool that resets the section, or hand-edited) hits a goroutine panic. The round-1 NIT3 disposition ("malformed file → clean error") is implicitly extended by round-2 to include `null`-valued mcpServers — but round-2 does NOT defend against the nil-map write.

**Severity:** must-fix counterexample. A goroutine panic is a worse failure mode than a wrapped error.

### Attacks attempted

1. **Byte-equivalence vs JSON-equivalence claim.** ACCEPTED-AS-NIT (doc-claim drift, see NIT-D6R2-3).
   - Round-2 BUILDER_WORKLOG.md asserts: *"HTTP/SSE/SDK entries are byte-equivalent on round-trip."* `init_cmd.go:651` doc-comment for `mcpServerEntry` says: *"pre-existing entries ... are preserved verbatim as json.RawMessage"*.
   - **Reality:** the code uses `json.MarshalIndent(topLevel, "", "  ")` at line 716. Per Go stdlib semantics (`encoding/json.MarshalIndent` doc + behavior probe), `MarshalIndent` runs `json.Indent` over the entire output INCLUDING embedded `json.RawMessage` bytes. The `RawMessage` bytes are unmarshaled and re-indented in the final output.
   - Probe `TestProbe_IndentRoundTrip_RawMessage` (run via mage testFunc, then removed pre-verdict): seed = HTTP entry with internal `\n      ` 6-space indentation; result = 2-space normalized indentation. Internal JSON structure (keys, values, nesting) is preserved; INDENTATION shape is normalized to the new file's 2-space scheme.
   - The PLAN.md D6 acceptance line 181 says *"re-run adds tillsyn without removing the other"* — that's a SEMANTIC claim, not a byte-equivalence one. Reality matches the PLAN.md acceptance. The doc-claim in `BUILDER_WORKLOG.md` and the `init_cmd.go:651` doc-comment overstate the property as byte-equivalence.
   - Tests `TestInit_MCPJSON_PreservesHTTPTransport` and `_AppendsToExisting` assert on field-value extraction (e.g. `notionFields["type"] == "http"`) — they would PASS under either byte-equivalence OR semantic-equivalence; they do NOT pin down which property the implementation provides.
   - **Disposition:** rewrite `init_cmd.go:651` doc-comment and the round-2 BUILDER_WORKLOG.md claim to say "JSON-semantic-equivalent" (or "field-content-equivalent") rather than "byte-equivalent" / "verbatim". Non-blocking refinement — the actual behavior is correct and tests pass.

2. **Empty / missing / null / array `mcpServers` shapes.** TWO COUNTEREXAMPLES — one CONFIRMED (FF2 nil-map panic), one MITIGATED.
   - **2a — missing `mcpServers` key (`{"someOtherKey":"foo"}`):** MITIGATED. Probe `TestProbe_MissingMcpServers_RegisterFlow`: result = `{"mcpServers": {"tillsyn": {...}}, "someOtherKey": "foo"}`. Clean. The check at lines 697-701 `if raw, ok := topLevel[mcpServersKey]; ok && len(raw) > 0` gates the unmarshal, and `servers := make(...)` at line 696 ensures the map is non-nil before insertion. Pre-existing `someOtherKey` preserved.
   - **2b — `mcpServers: null` (`{"mcpServers": null}`):** **CONFIRMED — FF2.** Probe `TestProbe_NullMcpServersValue_RegisterFlow`: `registerMCPJSON` PANICS with `"assignment to entry in nil map"`. Trace:
     - Line 696: `servers := make(map[string]json.RawMessage)` — map is non-nil.
     - Line 697-701: `raw = []byte("null"), ok = true, len(raw) = 4 > 0` → enters the inner `Unmarshal` branch.
     - Line 698: `json.Unmarshal([]byte("null"), &servers)` — Go's encoding/json treats `null` as a no-op for pointers but **for a map it sets the map header to nil**. Verified via `TestProbe_NullMcpServers_UnmarshalIntoMap`: `servers==nil after Unmarshal(null)? true len=0 err=<nil>`. The make-call's map is overwritten with nil.
     - Line 703-706: presence check on nil map is fine (returns zero-value, `found = false`).
     - Line 710: `servers[mcpServerKey] = json.RawMessage(tillsynRaw)` — write to nil map → PANIC.
   - **Severity:** must-fix. Goroutine panic on a JSON shape that is legal-JSON-but-unusual. A hardened tool would either return a clean error ("mcpServers is null; cannot register"), or recover by re-initializing the map. The PLAN.md D6 acceptance bullet "TestInit_MCPJSON_AppendsToExisting: existing `.mcp.json` ... re-run adds `tillsyn` without removing the other" reads naturally as covering this case; the test suite does not.
   - **Fix shape:** insert a nil-reset after line 698: `if servers == nil { servers = make(map[string]json.RawMessage) }`. Add regression test `TestInit_MCPJSON_NullMcpServersValue` seeding `{"mcpServers": null}` and asserting registerMCPJSON returns successfully with the `tillsyn` entry written.
   - **2c — `mcpServers: []` (`{"mcpServers": []}`):** MITIGATED. Probe `TestProbe_ArrayMcpServersValue_RegisterFlow`: result = clean wrapped error `"parse <path> mcpServers: json: cannot unmarshal array into Go value of type map[string]json.RawMessage"`. No panic, no file mutation. The user's input file is left intact. The error message is reasonable.

3. **Tillsyn-entry idempotency under RawMessage shape.** MITIGATED.
   - Idempotency check at lines 703-706 is KEY-NAME-ONLY: `if _, found := servers[mcpServerKey]; found { return 0, 1, nil }`. The entry body is never inspected. A user who manually customizes `tillsyn` (different `command` path, custom `args`, custom `env`) will see their customization PRESERVED on re-run.
   - `TestInit_MCPJSON_Idempotent` (line 660+) seeds `tillsyn` with `/original/path/to/till`, runs init, asserts the original path is preserved. This is the verbatim user-customization case. Round-2 preserves this contract.
   - The presence-check on KEY-NAME (not value-equality) is the correct shape — re-running `till init` is not a forced-reconciliation tool.

4. **`tillsyn` key collision with non-stdio transport.** MITIGATED.
   - Lines 703-706 are KEY-NAME-only. If user has `{"tillsyn": {"type": "http", "url": "..."}}` (manual edit OR collision with another tool using the `tillsyn` key), the presence check fires and registerMCPJSON returns `(0, 1, nil)` without overwriting the HTTP entry. The user's HTTP `tillsyn` survives.
   - This is PLAN.md D6 acceptance "skip if exists" — honored. Not a counterexample.

5. **fsatomic.WriteFile atomicity across `.mcp.json` corruption.** MITIGATED.
   - `internal/fsatomic/atomic.go:46-91` confirmed: same-directory `os.CreateTemp` + `Sync` + `Chmod` + `Close` + `Rename`. Rename is atomic on POSIX (same filesystem guaranteed by same-directory invariant). On failure between create and rename, deferred cleanup at lines 60-64 removes the temp.
   - If `Rename` itself fails (rare — usually disk-full or permission), the deferred cleanup removes the partial temp, the target stays intact, and an error bubbles up through `fmt.Errorf("write %q: %w", target, writeErr)` at `init_cmd.go:720`.
   - No data-loss path on partial write.

6. **JSON marshal output diff stability — key reordering.** ACCEPTED-AS-NIT (UX paper cut, see NIT-D6R2-1).
   - Probe `TestProbe_KeyOrderingDrift`: seed = `{"mcpServers":{"zeta":..,"alpha":..,"beta":..}}` (insertion order). After registerMCPJSON: `{"mcpServers":{"alpha":..,"beta":..,"tillsyn":..,"zeta":..}}` (alphabetical order).
   - Per Go stdlib: `json.Marshal` and `json.MarshalIndent` sort map keys alphabetically when serializing `map[string]T`. Verified via Context7 `/golang/website` (key-sort-on-marshal is documented `encoding/json` behavior). This is expected, not a bug.
   - **UX impact:** users who intentionally group entries logically (e.g. company-internal MCPs at top, third-party below) lose that grouping on first `till init --mcp true`. Their re-edits also get sorted on subsequent runs.
   - Severity: cosmetic. Not data loss. Not idempotency breaking (the idempotent re-run early-returns at line 703-706 BEFORE any re-marshal, so a `.mcp.json` already containing `tillsyn` is byte-stable on re-run).
   - **Disposition:** NIT-D6R2-1. Refinement candidate: switch to ordered map (sorted explicit insertion or `slices.Sorted(maps.Keys(servers))` with `json.Encoder` per-key writes). Defer to post-MVP — Claude Code's own tools likely don't preserve key order either, so the value-add is marginal.

7. **Marshal indentation drift — user 4-space vs `MarshalIndent` 2-space.** ACCEPTED-AS-NIT (folds into NIT-D6R2-1).
   - `MarshalIndent(topLevel, "", "  ")` hardcodes 2-space indentation. If user's `.mcp.json` was authored with 4-space (or tabs), the first `till init --mcp true` rewrites it to 2-space.
   - Severity: cosmetic, generates a non-substantive diff. Subsequent re-runs that hit the idempotent short-circuit (line 703-706) DO NOT re-write the file, so the user's re-formatted version is stable on re-run.
   - Folds into NIT-D6R2-1 (UX drift from re-marshal). Same disposition.

8. **Existing `tillsyn` entry with internal whitespace.** MITIGATED.
   - Idempotent path early-returns at lines 703-706 before any re-marshal. The user's whitespace-laden `tillsyn` entry is NEVER re-serialized. No expansion.

9. **NIT3 (malformed `.mcp.json` error UX) — did round-2 silently fix or regress?** MITIGATED.
   - Line 688 still wraps `json.Unmarshal` failure as `fmt.Errorf("parse %q: %w", target, unmarshalErr)`. Same as round-1. No remediation hint added (NIT3 deferred per round-1; deferral honored). No regression.

10. **NIT1 deferral procedural correctness.** MITIGATED.
    - PLAN.md `workflow/drop_4c_6/DROP_4c.6.W2_TILL_INIT/PLAN.md:178` updated: `TUI mode: DEFERRED — see ROUND-2 deferral note below.`
    - Lines 188 — ROUND-2 deferral block present, names "Drop 4c.7 or 4c.8" as the destination, cites round-1 NIT1, references `init_cmd.go:229` hardwire, and states JSON mode works correctly. Procedurally complete.
    - `init_cmd.go:229` STILL contains `m.finalPayload.MCP = false // TUI default per droplet acceptance.` — the deferral is not lying; the TUI walk still hardwires false. JSON mode unchanged.
    - Acceptance bullet for tests (PLAN.md line 184-185) lists the two new regression tests by exact name (`TestInit_MCPJSON_PreservesHTTPTransport`, `TestInit_MCPJSON_PreservesTopLevelExtras`) — present in `init_cmd_test.go` at lines 752-812 and 822-864 respectively.

11. **Re-run after a corrupt write — stale `.tmp-*` accumulation.** ACCEPTED-AS-NIT (see NIT-D6R2-2).
    - Probe `TestProbe_StaleTmpFile_RegisterFlow`: seed `.mcp.json.tmp-stale123` (simulating a prior interrupted `fsatomic.WriteFile`), then run registerMCPJSON. Result: `.mcp.json` created successfully (`add=1`), AND `.mcp.json.tmp-stale123` STILL PRESENT.
    - `fsatomic.WriteFile` uses `os.CreateTemp(dir, base+".tmp-*")` which generates a unique random suffix — no collision with the stale `.tmp-stale123`. The success path doesn't sweep prior `.tmp-*` residue.
    - **Severity:** cosmetic. Accumulating `.tmp-*` files clutter the project root but don't break functionality. Not a security issue (mode 0o600 from `os.CreateTemp`).
    - **Disposition:** NIT-D6R2-2. Refinement candidate at the `internal/fsatomic` level: add an optional pre-write sweep for stale `<base>.tmp-*` siblings older than N seconds. Out of scope for D6 (would touch fsatomic's public API). Defer to a future drop.

12. **`mage ci` claim re-verification.** MITIGATED.
    - Independently ran `mage testPkg ./cmd/till` post-round-2: 277/277 PASS, no failures, no skips. Matches builder's claim.
    - `mage testPkg ./cmd/till` is the relevant scoped verification — `mage ci` runs the whole suite (3087 per builder). Did not re-run full CI from this falsification (W2.D6 is a single-package surface; package-level test is the load-bearing gate).

13. **Test residue.** MITIGATED.
    - `git status --short cmd/till/` after probe-removal + `mage testPkg`: shows only `M cmd/till/init_cmd.go` + `M cmd/till/init_cmd_test.go`. No stray `.tillsyn/`, no `.mcp.json`, no probe-test residue. Discipline followed.

14. **CONSUMER-TIE contract still honored.** MITIGATED.
    - All 6 W2.D6 tests (`TestInit_MCPJSON_FreshFile`, `_AppendsToExisting`, `_Idempotent`, `_OptOut`, `_PreservesHTTPTransport`, `_PreservesTopLevelExtras`) invoke `run(context.Background(), ...)` end-to-end or via `runInitJSONInTempDir(t, ...)` which itself calls `run(...)`. None call `registerMCPJSON` directly. Read of each test confirms: cobra wiring exercised; W2-FF6 ROUND-2 contract honored.

### Findings

- **FF2 (CRITICAL — must-fix counterexample):** `registerMCPJSON` PANICS with `"assignment to entry in nil map"` when `.mcp.json` contains `{"mcpServers": null}`. Trace: `json.Unmarshal([]byte("null"), &servers)` overwrites the `make(...)`-initialized map to nil; subsequent `servers[mcpServerKey] = ...` at line 710 panics. Reproduction probe verified via `mage testFunc`. **Fix:** insert `if servers == nil { servers = make(map[string]json.RawMessage) }` after line 698; add `TestInit_MCPJSON_NullMcpServersValue` regression. Severity: goroutine panic on legal-JSON input is worse than a clean wrapped error.

- **NIT-D6R2-1 (UX drift — key reordering + indentation normalization):** `json.MarshalIndent` sorts map keys alphabetically AND re-indents the entire output to 2-space. User's `.mcp.json` key order and indent style are NOT preserved on first `till init --mcp true`. Non-blocking; UX paper cut. Idempotent re-run is byte-stable (line 703-706 short-circuits before re-marshal). Disposition: defer to post-MVP if at all — Claude Code's own tools likely don't preserve key order either.

- **NIT-D6R2-2 (stale `.tmp-*` accumulation):** if a prior `fsatomic.WriteFile` was interrupted, the stale `.mcp.json.tmp-*` file sits in the project root forever. Subsequent registerMCPJSON calls don't sweep it. Non-blocking; cosmetic clutter. Disposition: future refinement at `internal/fsatomic` level (out of D6 scope — touches fsatomic public API).

- **NIT-D6R2-3 (doc-claim drift):** `cmd/till/init_cmd.go:651` doc-comment says preserved-entries are "verbatim as json.RawMessage" and `BUILDER_WORKLOG.md` round-2 entry says "byte-equivalent on round-trip". Reality: `json.MarshalIndent` re-indents the entire output, so individual RawMessage bytes are JSON-semantic-equivalent but NOT byte-equivalent. Tests assert on field values, not byte-equivalence. Disposition: one-line doc-comment + worklog wording correction ("JSON-semantic-equivalent" or "field-content-equivalent"). Non-blocking.

- **Round-1 follow-up status:**
  - FF1 (HTTP/SSE round-trip data loss) — **CLOSED** by round-2's `json.RawMessage` envelope. `TestInit_MCPJSON_PreservesHTTPTransport` (init_cmd_test.go:752-812) asserts `type` + `url` survive. Verified.
  - NIT1 (TUI confirm prompt) — **DEFERRED** correctly in PLAN.md D6 row + ROUND-2 block at lines 178+188. `init_cmd.go:229` hardwire still in place — deferral not lying.
  - NIT2 (top-level extras drop) — **CLOSED** by round-2's `map[string]json.RawMessage` top-level envelope. `TestInit_MCPJSON_PreservesTopLevelExtras` (init_cmd_test.go:822-864) asserts `someOtherKey.foo == "bar"` survives. Verified.
  - NIT3 (malformed JSON error UX) — **STILL DEFERRED**; round-2 didn't regress (same bare wrapped error at line 688).

### Notes

- **Verdict standard.** FF2 is a goroutine PANIC on legal JSON input. A user whose `.mcp.json` is `{"mcpServers": null}` (which can arise from any tool that resets the section to null, or from a hand-edit where someone replaced an object with null) hits a runtime crash. The PLAN.md D6 acceptance reads naturally as covering this — "register tillsyn against an existing `.mcp.json`" — and the test suite does not exercise it. Panic vs clean error is a strict severity drop and qualifies as must-fix even though it's a narrower input shape than round-1's FF1.
- **Recommended fix shape.** One-line fix at `init_cmd.go:699` (or immediately after the inner Unmarshal at line 698):
  ```go
  if unmarshalErr := json.Unmarshal(raw, &servers); unmarshalErr != nil {
      return 0, 0, fmt.Errorf("parse %q mcpServers: %w", target, unmarshalErr)
  }
  if servers == nil {
      servers = make(map[string]json.RawMessage)
  }
  ```
  And one new regression test `TestInit_MCPJSON_NullMcpServersValue` seeding `{"mcpServers": null}` (with a trailing newline) and asserting registerMCPJSON returns `(1, 0, nil)` with the resulting file containing the `tillsyn` entry.
- **NIT-D6R2-1 disposition:** key-reordering is Go-stdlib-mandated behavior on `map[string]T` marshal. Switching to ordered marshaling means giving up the convenience of map semantics — either build a `[]struct{Name string; Raw json.RawMessage}` and `json.Marshal` that, or hand-write the JSON. Neither is cheap; defer.
- **NIT-D6R2-3 disposition:** the doc-comment wording fix is trivial (one Edit on `init_cmd.go:651` + worklog wording). Routing recommendation: fold into the FF2 fix-respawn so the builder corrects the doc-claim at the same time as the nil-map guard.
- **mage testPkg** independently verified GREEN (277/277 PASS).
- **Builder discipline observations:** round-2 closed FF1 + NIT2 cleanly with the right structural fix (RawMessage envelope at BOTH levels). PLAN.md deferral procedurally complete. The blind spot is `null`-valued mcpServers — not in the round-1 attack list, not in the new test set, not anticipated by the "byte-equivalent" mental model. The mental model the worklog uses ("preserved verbatim as RawMessage") is slightly off from reality (re-indented under MarshalIndent), which probably contributed to missing the nil-map write path.
- **Attack table** below.

### Attack family table

| Attack family                                | Result    | Notes |
| -------------------------------------------- | --------- | ----- |
| Byte-equiv vs JSON-equiv doc claim           | NIT-3     | NIT-D6R2-3 — doc says "byte-equivalent"; reality is JSON-semantic-equivalent due to `MarshalIndent`. |
| Missing `mcpServers` key                     | REFUTED   | `make`-initialized servers map handles absent key cleanly; other top-level keys preserved. |
| `mcpServers: null`                           | CONFIRMED | **FF2** — `json.Unmarshal("null", &servers)` overwrites map to nil; write at line 710 panics. |
| `mcpServers: []`                             | REFUTED   | Clean wrapped error; no panic, no file mutation. |
| Tillsyn-entry idempotency (key-name check)   | REFUTED   | Lines 703-706 key-name presence check; entry body never inspected; user customization preserved. |
| `tillsyn` key collision w/ HTTP transport    | REFUTED   | Key-name presence check skips; HTTP `tillsyn` entry survives. |
| fsatomic atomicity on partial write          | REFUTED   | Same-dir temp + atomic rename; deferred cleanup on failure; target intact. |
| Key reordering (alphabetical)                | NIT-1     | NIT-D6R2-1 — Go stdlib map-key sort. UX paper cut on first run; byte-stable on idempotent re-run. |
| Indentation drift (4-space → 2-space)        | NIT-1     | Folds into NIT-D6R2-1 — `MarshalIndent` hardcodes 2-space. |
| Internal whitespace expansion on `tillsyn`   | REFUTED   | Idempotent path short-circuits before re-marshal; user whitespace preserved. |
| NIT3 (malformed JSON error UX)               | REFUTED   | Still deferred from round-1; round-2 didn't regress. |
| NIT1 (TUI deferral procedural correctness)   | REFUTED   | PLAN.md updated correctly; `init_cmd.go:229` hardwire still in place; deferral not lying. |
| Stale `.tmp-*` file accumulation             | NIT-2     | NIT-D6R2-2 — fsatomic doesn't sweep prior stale `.tmp-*` residue. |
| `mage testPkg ./cmd/till` claim verification | REFUTED   | Independently verified GREEN (277/277). |
| Test residue (probe-removed)                 | REFUTED   | `git status --short cmd/till/` shows only the 2 declared mods. |
| CONSUMER-TIE contract on 6 W2.D6 tests       | REFUTED   | All 6 invoke `run(...)` end-to-end or via the helper. |
| Round-1 FF1 closure                          | REFUTED   | RawMessage envelope closes the schema gap; `TestInit_MCPJSON_PreservesHTTPTransport` asserts. |
| Round-1 NIT2 closure                         | REFUTED   | Top-level RawMessage envelope preserves sibling keys; `TestInit_MCPJSON_PreservesTopLevelExtras` asserts. |

### Hylla Feedback

- **Query:** Hylla not queried this round. The attack surface is (a) uncommitted Go code (`cmd/till/init_cmd.go` round-2 rewrite — pre-reingest staleness window), (b) Claude Code MCP semantics (Context7 territory, already covered by round-1 evidence), (c) Go stdlib `encoding/json` behavior (verified via `go doc` + behavior probes via mage testFunc, then probe-file removed), and (d) `internal/fsatomic` package contract (verified via direct `Read` of `internal/fsatomic/atomic.go`).
- **Missed because:** no Hylla query attempted; the uncommitted-code constraint + external-semantics-from-Context7 / `go doc` discipline already routes around Hylla for this surface.
- **Worked via:** `Read` on `cmd/till/init_cmd.go`, `cmd/till/init_cmd_test.go`, `internal/fsatomic/atomic.go`, `workflow/drop_4c_6/BUILDER_WORKLOG.md`, `workflow/drop_4c_6/BUILDER_QA_FALSIFICATION.md`, `workflow/drop_4c_6/DROP_4c.6.W2_TILL_INIT/PLAN.md`; `git status --short cmd/till/`; `mage testPkg ./cmd/till`; `mage testFunc ./cmd/till TestProbe_*` (probe tests added then removed pre-verdict).
- **Suggestion:** none new this round. The recurring "HEAD-aware fallback when enrichment in flight" suggestion from prior rounds did not bite — the falsification did not need committed-symbol queries.

### Notes (post-verdict)

- Probe tests (`mcp_null_probe_test.go`) were created in `cmd/till/`, run via `mage testFunc`, then deleted before verdict close. `git status --short cmd/till/` post-cleanup shows only `M cmd/till/init_cmd.go` + `M cmd/till/init_cmd_test.go` — no probe residue.
- Round-2 PASS verdict is blocked by FF2. Builder must respawn for round-3 to add the nil-map guard + regression test. Recommend folding NIT-D6R2-3 doc-claim fix into the same respawn (trivial wording).
- NIT-D6R2-1 and NIT-D6R2-2 are deferred refinements — not blocking.
