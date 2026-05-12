# Drop 4c.6 — Builder QA Proof

Per-droplet build-QA-proof rounds append below. Each round entry stamps droplet
ID, round number, findings, missing evidence, summary verdict, and Hylla
feedback.

---

## Droplet 4c.6.W3.D5 — Round 1

**Reviewer:** go-qa-proof-agent (subagent).
**Date:** 2026-05-11.
**Droplet:** `4c.6.W3.D5 — Post-render validator wired at Render's exit + sentinel test`.
**Verdict:** **PASS.**

### Check matrix (A–J per orchestrator spawn prompt)

| # | Subject | Evidence | Result |
| --- | --- | --- | --- |
| A | `ErrInvalidAgentBody` declared at package scope adjacent to existing sentinels | `render.go:137` — `var ErrInvalidAgentBody = errors.New("render: invalid agent body in rendered bundle")`, sitting between `ErrAgentBodyNotFound` (L108) and `ErrInvalidAgentTemplatePath` (L170). Doc-comment block L110-136 documents the 3-signal AND check. | PASS |
| B | `validateBundle` runs at `Render`'s exit BEFORE final return, AFTER `renderSettings` | `render.go:283-297` — `renderSettings(...)` at L283; new `if err := validateBundle(bundle, binding); err != nil { rollback.run(); return "", fmt.Errorf("render: validate bundle: %w", err) }` at L294-297; `return promptBody, nil` at L299. Wiring is load-bearing per HF8 — exit-path call, not a dangling helper. | PASS |
| C | `validateAgentBodyShape` is a pure-function helper | `render.go:354` — `func validateAgentBodyShape(body string) error`. Signature takes a string + returns error; body L355-392 performs string operations only (`strings.HasPrefix`, `strings.Index`, `strings.Contains`, `fmt.Errorf`). Zero filesystem access; zero stateful side-effects. | PASS |
| D | 3 signals B → A → C with deterministic order | `render.go:358-389`. Order: (1) L358-359 leading `---\n` prefix [B], (2) L362-365 closing `---\n` index [B], (3) L369-371 `name:` in frontmatter [B], (4) L373-376 `len(postFrontmatter) <= 200` [A], (5) L384-389 marker disjunction `# PLACEHOLDER` OR `# Section 0` OR `## Role` [C]. Three markers exactly as W3-FF6 LOCKED. | PASS |
| E | Each signal wraps via `fmt.Errorf("%w: <specifics>", ErrInvalidAgentBody)` | `render.go:359, 364, 370, 374, 390` — every failure returns `fmt.Errorf("%w: ...", ErrInvalidAgentBody, ...)`. `errors.Is(err, ErrInvalidAgentBody)` succeeds for every signal. | PASS |
| F | 5 new tests with the exact names specified | `render_test.go:1458` `TestRenderValidatorFailsOnTooShortBody` (Signal A); L1495 `_FailsOnMissingFrontmatter` (Signal B); L1522 `_FailsOnMissingMarker` (Signal C); L1554 `_PassesOnSubstantiveBody` (positive control); L1592 `_AcceptsAllEmbeddedPlaceholders` (W1.D1 floor). All five tests use `errors.Is(err, render.ErrInvalidAgentBody)` for the failure-path assertions and prove HF8 wiring by asserting on `Render()`'s return value, NOT on `validateBundle()` standalone. | PASS |
| G | `_AcceptsAllEmbeddedPlaceholders` walks `templates.DefaultTemplateFS` under `builtin/agents/` — 27/27 placeholders pass | `render_test.go:1598-1612` walks `agentsRoot = "builtin/agents"` via `fs.WalkDir(templates.DefaultTemplateFS, ...)`; L1620-1650 sub-tests each placeholder by injecting the body into the project tier and running `Render`. Filesystem enumeration confirms 27 .md files: `till-gdd/` = 7, `till-gen/` = 8, `till-go/` = 12 (totals confirmed via `ls -1` of each group). L1657-1660 enforces `minPlaceholders = 27` floor. Builder worklog (BUILDER_WORKLOG.md:36-44) reports 32 GREEN sub-tests (5 top-level + 27 WalkDir). | PASS |
| H | 5 pre-existing tests adjusted to use the shared validator-conforming helper | `render_test.go:809-813` declares `validatorConformingBodySuffix()` returning `"# PLACEHOLDER — ..." + 5 × repeated filler` (~300 chars post-frontmatter, clears Signal A floor at 200). Direct fixture mutation: L902 `_UserOverride` body line uses `validatorConformingBodySuffix()`; L935 + L942 `_ProjectOverride` both use it. Indirect fixture mutation via `d3UserTierFrontmatter()` (L1066-1070 — now appends `validatorConformingBodySuffix()` + body-bytes-preserve-marker sentinel) consumed by `_FrontmatterStripModelOnAgentsTOMLSet` (L1081), `_FrontmatterStripToolsOnAgentsTOMLSet` (L1123), `_FrontmatterPreservedWhenAgentsTOMLAbsent` (L1384). Five sites total; sentinel substrings (`SENTINEL_USER_TIER`, `SENTINEL_PROJECT_TIER`, `body-bytes-preserve-marker`) remain assertable because they appear AFTER the suffix. | PASS |
| I | `mage test-pkg ./internal/app/dispatcher/cli_claude/render` GREEN (70/70) | Builder-reported only — BUILDER_WORKLOG.md:45 states `"70 passed, 0 failed"`. Drop-orch will re-run `mage ci` at Phase 6 / drop-end per the per-droplet constraint (`mage ci NOT run by this builder per droplet constraint`). Mage gate is GREEN per builder verdict; CI re-validation is drop-orch's responsibility, not this QA round's. | PASS (per builder; drop-orch re-runs at Phase 6) |
| J | PLAN.md W3.D5 state = `done` | `workflow/drop_4c_6/DROP_4c.6.W3_BUNDLE_AND_ISOLATION/PLAN.md:224` reads `**State:** done`. `git diff --stat` shows the PLAN.md file in this drop's W3 sub-plan at +2/−2 (matches a single state-line flip). L1 `workflow/drop_4c_6/PLAN.md` is untouched by this droplet (L1 row is the W3 sub-plan container row, not a per-droplet row). | PASS |

### Cross-cutting verifications

- **HF8 wiring contract satisfied** — Every failure-path test (`_FailsOnTooShortBody`, `_FailsOnMissingFrontmatter`, `_FailsOnMissingMarker`) calls `render.Render(...)` end-to-end and asserts on the returned error + the rollback wipe (`<bundle>/system-prompt.md` + `<bundle>/plugin` both gone via `os.Stat` returning `os.ErrNotExist`). No test calls `validateBundle` standalone. This is the exact pattern HF8 mandates: shipped-but-not-wired anti-pattern impossible because no test would pass if the wiring were dangling.
- **Rollback invariant preserved** — `validateBundle` failure inside `Render` invokes `rollback.run()` at `render.go:295` BEFORE returning the error, mirroring the renderSystemPrompt / renderPluginManifest / renderAgentFile / renderMCPConfig / renderSettings failure path conventions (lines 258, 264, 271, 277, 284). The rollback removes `<bundle>/system-prompt.md` + `<bundle>/plugin/` blanket per `render.go:418-419`.
- **Signal C 3-marker disjunction works on every shipped placeholder** — All 27 W1.D1 placeholders use `# PLACEHOLDER — substantive content lands in Drop 4c.8 W4` as their H1 heading per the W1.D1 ship contract. Signal C's `strings.Contains(postFrontmatter, "# PLACEHOLDER")` matches every one. Confirmed via the WalkDir test's 27/27 pass count.
- **Pure-function split serves dual purpose** — `validateBundle` reads from disk (catches future regression where `os.WriteFile` silently truncates); `validateAgentBodyShape` is pure (signal logic unit-testable without filesystem). The two-layer split lets `Render`'s integration tests prove HF8 wiring while still keeping the signal logic structurally inspectable.
- **W3-FF13 NIT — Signal C unanchored substring match** — Builder accepted-by-design per round-3 W3-FF13 NIT disposition (W3 sub-plan PLAN.md L307). A stub deliberately quoting `` `# PLACEHOLDER` `` inside backticks would pass Signal C falsely, but Signals A + B catch the common-case stub shapes regardless. The accepted refinement candidate is line-anchored matching; out of scope here.
- **W3-FF14 NIT — `## Role` marker is speculative-future** — The third marker has no current consumer; round-3 W3-FF14 NIT accepted-by-design (W3 sub-plan PLAN.md L308). Forward-flex hedge with negligible runtime cost (one substring scan); harmless.
- **Signal A floor is `<=` not `<`** — `render.go:373` reads `if n := len(postFrontmatter); n <= minBodyLength` where `minBodyLength = 200`. PLAN.md L232 spec says "body length > 200" (strict-greater); the implementation interprets "exceeds 200" as `> 200` ⇔ `<= 200` fails. Boundary case: body of exactly 200 chars FAILS Signal A. Consistent with the W4-floor-as-forward-dep spec (W3-FF8) — every legitimate body MUST exceed the floor.

### Findings

None blocking.

- **W3-D5-PF1 (informational, accepted-by-design):** Signal C uses `strings.Contains` (substring match, not line-anchored). W3 sub-plan PLAN.md round-3 W3-FF13 NIT acknowledges this trade-off and accepts it as a refinement candidate. The 3-signal AND structure means a marker-quoted-in-prose stub still needs to clear Signal A (length > 200) AND Signal B (frontmatter intact) — both of which the contrived case would also need to satisfy. No action required from this builder; routed to refinement backlog per W3 sub-plan PLAN.md L307.
- **W3-D5-PF2 (informational, scope-correct):** Dirty peer files in `git status` (`cmd/till/init_cmd.go`, `internal/templates/builtin/agents.example.toml`, `internal/templates/embed.go`, etc.) belong to in-flight W2 + W5 + W6 droplet builders. Per the orchestrator spawn prompt's review scope and the dirty-files-belong-to-peers convention, these are NOT in W3.D5's review surface. Builder confined edits to `render.go` + `render_test.go` + the W3 sub-plan `PLAN.md` per declared paths.
- **W3-D5-PF3 (informational, forward-dep documented in code):** The 200-char Signal A floor is a forward-only honor-system constraint on Drop 4c.8 W4 prompt authoring per W3-FF8. `render.go:121` doc-comment references this as "W3-FF8 W4-floor-as-forward-dep." No W3-shipped enforcement seam exists for it beyond today's placeholder set (which all 27 clear); future W4 prompt authoring must verify body length > 200 at authoring time. W3 sub-plan PLAN.md L251 routes this propagation target to W4 sub-planner. Documented; not a defect.

### Missing evidence

None. All ten A-J check categories closed with direct file Reads, embedded-FS enumeration (`ls -1` of three group directories returning 27 .md totals), and `git status` / `git diff --stat` for delta verification. The builder-reported 70/70 mage gate is the standard per-droplet contract; drop-orch re-runs at Phase 6 per the no-`mage ci`-per-droplet constraint.

### Summary

**PASS.**

Builder shipped a clean W3.D5 droplet: `ErrInvalidAgentBody` sentinel declared at package scope adjacent to existing sentinels, `validateBundle` wired at `Render`'s exit between `renderSettings` and the final return (HF8 wiring contract satisfied), `validateAgentBodyShape` pure-function helper applying the 3-signal AND check with deterministic B → A → C order, every signal wrapped via `fmt.Errorf("%w: ...", ErrInvalidAgentBody, ...)`. 5 new tests at the specified names assert on `Render()` end-to-end (proving HF8 wiring, not dangling-helper). `TestRenderValidatorAcceptsAllEmbeddedPlaceholders` walks the embed.FS under `builtin/agents/` and exercises all 27 W1.D1 placeholders — 27/27 pass per builder verdict; embedded FS enumeration independently confirms 7 + 8 + 12 = 27 placeholder files across till-gdd/till-gen/till-go. 5 pre-existing W3.D2/D3 fixture-using tests adjusted via the shared `validatorConformingBodySuffix()` helper + updated `d3UserTierFrontmatter()` — no production-behavior change to D2/D3 — preserving sentinel-substring assertions because the suffix prepends to (not replaces) the sentinel content. State flipped `todo → in_progress → done` on the W3 sub-plan PLAN.md. Mage gate green (70/70 builder-reported; drop-orch re-runs `mage ci` at Phase 6). Three informational findings (W3-D5-PF1 substring-match NIT, W3-D5-PF2 dirty peer files out-of-scope, W3-D5-PF3 W4 forward-dep documented) are accepted-by-design or routed to refinement backlog; none block the droplet's completion.

### Hylla Feedback

None — Hylla was not queried during this QA review. The droplet's review surface is entirely freshly-uncommitted code (W3.D2 + W3.D3 + W3.D5 production code landed within this drop and is uncommitted at review time per the dirty `git status`; the W3 sub-plan PLAN.md is similarly uncommitted). Hylla's last `github.com/evanmschultz/tillsyn@main` ingest predates the W3.D2 commit `d671b91` mentioned in the worklog, so a Hylla round-trip would return stale symbol shapes for `assembleAgentFileBody` / `validateAgentBasename` / `ErrAgentBodyNotFound` and miss `ErrInvalidAgentBody` / `validateBundle` / `validateAgentBodyShape` entirely. `Read` on the four target files (`render.go`, `render_test.go`, the W3 sub-plan `PLAN.md`, `BUILDER_WORKLOG.md`) plus `ls -1` against the three embedded agent directories covered the proof surface exhaustively with zero ambiguity. The fallback was tool-fit-driven, not a Hylla miss; no Hylla query was attempted, so there's no miss to record. Ergonomic ask: a Hylla query mode "newest-ingest-of-uncommitted-file" returning a clear "freshly-uncommitted; query git diff instead" signal would short-circuit the same-drop-staleness deliberation pattern that recurs every drop. Not blocking.

---

## Droplet 4c.6.W6.D3 — Round 1

**Reviewer:** go-qa-proof-agent (subagent, build-QA-proof axis).
**Date:** 2026-05-09.
**Droplet:** `4c.6.W6.D3 — GDD_METHODOLOGY.md placeholder`.
**Parent kind:** `build`.
**Artifact under review:** `GDD_METHODOLOGY.md` at repo root (commit `cd6aa68`).
**Spec sources:** `workflow/drop_4c_6/PLAN.md` lines 292-309 (W6.D3 row);
`workflow/drop_4c_6/SKETCH.md` § 14.2 + § 26.W6 + § 14.2.1 reference;
`workflow/drop_4c_6/BUILDER_WORKLOG.md` § Droplet 4c.6.W6.D3 — Round 1.

### Findings

(none — see Summary)

### Missing Evidence

(none — every acceptance component maps to a concrete artifact location; see
Trace below.)

#### Acceptance trace

L1 acceptance bullet (PLAN.md line 299) decomposes into four components:

1. **Title.** `GDD_METHODOLOGY.md:1` — `# GDD Methodology — Graph-Driven
   Development`. H1 is unambiguous on first read; matches builder design note
   (worklog line 57-59).
2. **1-paragraph description.** `GDD_METHODOLOGY.md:5-15` — paragraph framing
   GDD as Cascade companion, citing `project_methodology_docs_tracker.md` and
   `SKETCH.md` § 14.2, deferring substantive content to "post-Hylla-rev /
   post-dogfood." Wording matches the L1 acceptance quoted text ("Graph-Driven
   Development methodology — populated post-Hylla-rev / post-dogfood per
   `project_methodology_docs_tracker.md` and `SKETCH.md` § 14.2") in spirit;
   the builder paraphrased rather than verbatim-quoted, but every required
   semantic element is present (GDD identity, post-Hylla-rev gate,
   post-dogfood gate, both citations).
3. **`<!-- TODO populate post-dogfood -->` marker.** `GDD_METHODOLOGY.md:3`
   exact-match of the required HTML comment, with a paired `<!-- END TODO -->`
   at line 62 bookending the placeholder body.
4. **Prior-art research note placeholder per `SKETCH.md` § 14.2.1.**
   `GDD_METHODOLOGY.md:45-53` — `## Prior-art research note (per SKETCH.md §
   14.2.1)` section explicitly cites §14.2.1, names survey targets
   (code-knowledge graphs, graph-RAG, semantic search, structural code
   search), and intentionally leaves conclusions empty pre-dogfood. Matches
   `SKETCH.md:513` "§14.2.1 prior-art research note still applies"
   carry-forward intent.

#### Constraint preservation trace

RiskNote (PLAN.md line 305): "Placeholder MUST clearly mark itself as
'populate post-dogfood' so adopters don't expect substantive content here."

Mitigations layered four-deep:

- `GDD_METHODOLOGY.md:3` — top-of-file `<!-- TODO populate post-dogfood -->`
  marker.
- `GDD_METHODOLOGY.md:5` — first sentence `This document is a **placeholder**.`
- `GDD_METHODOLOGY.md:10` — "Substantive content lands **post-Hylla-rev /
  post-dogfood**" in the lead paragraph.
- `GDD_METHODOLOGY.md:17-24` — dedicated `## Status` block stamping
  `**State:** placeholder. Do not treat the contents below as normative.` plus
  `**Populate after:** Hylla revision …` plus MVP-release-blocker call-out.
- `GDD_METHODOLOGY.md:55-60` — `## Non-goals (explicit)` block disclaiming
  "Not a Hylla user manual" and "Not a replacement for Cascade."

Adopters reading the file land on the placeholder framing immediately and
repeatedly. Constraint clear.

#### Diff-vs-spec audit

Commit `cd6aa68` ("docs(methodology): add GDD methodology placeholder")
touches exactly three paths:

- `GDD_METHODOLOGY.md` (NEW, +62 lines).
- `workflow/drop_4c_6/BUILDER_WORKLOG.md` (NEW, +72 lines).
- `workflow/drop_4c_6/PLAN.md` (-2/+2; W6.D3 `**State:**` flip `todo → done`
  AND a sibling W6.D2 `todo → in_progress` flip — W6.D2 is a concurrent
  build droplet not owned by this review).

The W6.D2 state flip in the same commit is mildly cross-droplet (W6.D2 is
W6.D3's parallel sibling under `Wave A` per PLAN.md line 372) but it does NOT
mutate W6.D2's content, planning, paths, or acceptance — only its lifecycle
state. The state flip mirrors what the W6.D2 builder did in its own
in_progress claim. Per WORKFLOW.md § "File Lifecycle" (PLAN.md row 41:
`durable — refined across plan-QA rounds; final at close`), state-only
mutations on sibling droplets are acceptable; the falsification sibling can
attack this further if it considers the bundling material.

Pre-existing dirty files in the orchestrator env header (`internal/app/...`,
`internal/templates/embed*.go`, `template_service.go`, etc.) all predate
W6.D3 and are untouched by `cd6aa68` — not W6.D3's diff, not findings.

#### Shipped-but-not-wired audit

Doc-only droplet. No Go code, no consumer to wire. Axis N/A.

#### Completion-checklist audit

- Builder set state in PLAN.md → `done`. ✓
- Builder appended `## Droplet 4c.6.W6.D3 — Round 1` to BUILDER_WORKLOG.md. ✓
- Builder reported Hylla feedback (N/A — non-Go droplet). ✓
- Doc-only droplet does NOT run `mage ci` (deferred to drop-end). ✓ (per
  WORKFLOW.md Phase 6 — drop-end CI; per droplet ValidationPlan "doc review
  pass; `mage ci`" with `mage ci` at drop close).

### Summary

**Verdict:** pass.
**Finding count:** 0.

Every L1 acceptance component (title + 1-paragraph description +
`<!-- TODO populate post-dogfood -->` marker + prior-art research note
placeholder per SKETCH §14.2.1) maps to a concrete byte range in
`GDD_METHODOLOGY.md`. Constraint preservation (clear "populate post-dogfood"
framing) is layered four-deep (TODO marker + lead-paragraph framing + Status
block + Non-goals block). Diff scope matches WORKFLOW.md File Lifecycle —
exactly three allowed paths in commit `cd6aa68`. Shape-hint deviation
(`~30-line stub` → ~58 LOC) is justified in the builder worklog by reserving
populated-doc structure; L1 acceptance has no hard line cap and `shape_hint`
is advisory. No drive-by edits. No shipped-but-not-wired risk (doc-only).

### Hylla Feedback

N/A — droplet touched non-Go files only; Hylla index not consulted.

---

## Droplet 4c.6.W6.D2 — Round 1

### Findings

- 1.1 [Axis: acceptance-criteria-coverage] [severity: low] AC1 (skeleton structure) covers all 14 enumerated topics from `PLAN.md:276` — `## Plan Down, Build Up` at L11; `## Three Orthogonal Axes` umbrella at L21; `## Closed 12-Value kind Enum` at L31; `## metadata.role Enum` at L41; `## metadata.structural_type Enum` at L51; `## Agent Shape` at L61; `## Section 0 — Semi-Formal Reasoning Certificate` at L71; `## Tillsyn-Flavored Specify Pass` at L81; `## TN-Per-Section Response Style` at L91; `## Hylla-First Evidence Ordering` at L101; `## TDD Requirement` at L111; `## QA Proof vs Falsification — Asymmetric Verification` at L121; `## blocked_by Ordering Primitive` at L133; `## Parent-Children-Complete Invariant` at L143; `## Isolation Enforcement` at L153 → `git grep -n "^## " CASCADE_METHODOLOGY.md` → no fix needed.
- 1.2 [Axis: acceptance-criteria-coverage] [severity: low] AC2 (1-3 paragraphs per section + TODO marker per section close) holds — sample sections inspected (L21-L29 Three Orthogonal Axes = 3 paragraphs + TODO marker; L11-L19 Plan Down Build Up = 3 paragraphs + TODO marker; L41-L49 metadata.role Enum = 3 paragraphs + TODO marker). Total `<!-- TODO populate post-dogfood with measured benchmarks -->` count = 19 (`git grep -c` confirmed) covering 17 H2 sections + lead-paragraph + leading marker, all within budget → no fix needed.
- 1.3 [Axis: acceptance-criteria-coverage] [severity: low] AC3 (HF9 fix — first H2 after H1 is `## Plan Down, Build Up`) holds — `CASCADE_METHODOLOGY.md:11` first `## ` heading reads `## Plan Down, Build Up`, immediately preceded by H1 + 3 lead paragraphs + leading TODO marker; testable via `git grep -n "^## " CASCADE_METHODOLOGY.md | head -1` → returns `CASCADE_METHODOLOGY.md:11:## Plan Down, Build Up` → no fix needed.
- 1.4 [Axis: acceptance-criteria-coverage] [severity: low] AC4 (cross-refs to AGENTS_CONFIG.md + GDD_METHODOLOGY.md) holds — both cited in lead paragraph at `CASCADE_METHODOLOGY.md:7` and again in dedicated `## Cross-References` section at L167-L168; bonus cross-refs to `SPAWN_PIPELINE.md` (L169), `CLI_ADAPTER_AUTHORING.md` (L170), and `WIKI.md § "Cascade Vocabulary"` (L171) supplied → no fix needed.
- 1.5 [Axis: spec-conformance] [severity: low] Constraint preservation per `SKETCH.md §26.W6 ContextBlocks`: "Plan Down Build Up" leads the doc (L11 first H2 — confirmed); cascade vocabulary cross-references `WIKI.md § Cascade Vocabulary` rather than redefining (L7 + L27 + L37 + L57 + L171 all explicitly defer to WIKI as canonical source for the closed enum values, worked-combinations table, and atomicity rules) → no fix needed.
- 1.6 [Axis: spec-conformance] [severity: low] Diff scope clean — `git status --porcelain` + `git show --stat 841ebc4` shows the W6.D2 commit touched exactly `CASCADE_METHODOLOGY.md` (200 lines, NEW); the working-tree edits are `workflow/drop_4c_6/PLAN.md` (state flip `in_progress → done` at L271 only) + `workflow/drop_4c_6/BUILDER_WORKLOG.md` (W6.D2 round-1 append L71-L161). No drive-by edits, no sibling-droplet contamination → no fix needed.
- 1.7 [Axis: completion-checklist-audit] [severity: low] Worklog "Files touched" + "Design decisions" + "Validation" sections complete and accurately describe the artifact; `mage ci` correctly skipped per drop-orch-runs-it-at-drop-end discipline (doc-only droplet, no Go code) → no fix needed.
- 1.8 [Axis: shipped-but-not-wired] N/A — doc-only droplet; no implementation that could be unwired.

### Missing Evidence

- None. All four acceptance criteria + leading-section constraint + WIKI-cross-ref constraint verified by direct file inspection at cited line numbers.

### Summary

Verdict: **pass**. 7 informational findings, 0 high/medium severity, 0 blockers. The W6.D2 builder shipped a 200-line `CASCADE_METHODOLOGY.md` skeleton that satisfies every acceptance criterion: 14 required-topic sections present at confirmed line numbers; `## Plan Down, Build Up` at L11 first H2 (HF9-grep-able assertion holds); 1-3 paragraphs per section with `<!-- TODO populate post-dogfood with measured benchmarks -->` markers (19 total); cross-references to `AGENTS_CONFIG.md` + `GDD_METHODOLOGY.md` (W6.D1 + W6.D3 forward-refs) plus bonus `SPAWN_PIPELINE.md` / `CLI_ADAPTER_AUTHORING.md` / `WIKI.md § Cascade Vocabulary` cites. Cascade vocabulary cross-refs WIKI as single canonical source rather than redefining (confirmed at L7, L27, L37, L57, L171). Diff scope limited to the 3 spec'd files (commit `841ebc4` for the artifact + working-tree PLAN.md state flip + BUILDER_WORKLOG.md append). Skeleton is intentionally evergreen at methodology-shape level with post-dogfood benchmark slots reserved per `project_methodology_docs_tracker.md`.

### Hylla Feedback

N/A — droplet touched non-Go files only; Hylla index not consulted.

---

## Droplet 4c.6.W6.D1 — Round 1

**Reviewer:** go-qa-proof-agent (subagent, build-QA-proof axis).
**Date:** 2026-05-09.
**Droplet:** `4c.6.W6.D1 — AGENTS_CONFIG.md (new top-level doc)`.
**Parent kind:** `build`.
**Artifact under review:** `AGENTS_CONFIG.md` at repo root (396 lines).
**Spec sources:** `workflow/drop_4c_6/PLAN.md` lines 248-267 (W6.D1 row);
`workflow/drop_4c_6/SKETCH.md` § 4 + § 4.4 + § 4.5 + § 5 + § 6 + § 12 + § 26.W6;
`workflow/drop_4c_6/BUILDER_WORKLOG.md` § Droplet 4c.6.W6.D1 — Round 1.

### Findings

- 1.1 [Axis: acceptance-criteria-coverage] [severity: low] AC1 (≥200 lines) holds — `wc -l AGENTS_CONFIG.md` returns 396, well above the floor. No fix needed.
- 1.2 [Axis: acceptance-criteria-coverage] [severity: low] AC1 (sections enumerated in PLAN.md) all present — schema → §2 (`[agents]` Defaults Block) + §3 (`[agents.<kind>]` Per-Kind Override Blocks); override semantics → §4 (Project + Local Two-Layer Merge); `env_set` vs `env_from_shell` → §5; `tools_allow` vs `tools_deny` override scope → §6; frontmatter strip behavior → §7; `claude_md_addons` → §8; worked examples (Bedrock / Vertex / OpenRouter / Ollama Cloud) → §9 (5 sub-sections 9.1–9.5 covering Anthropic + the four named providers). Section sequence matches PLAN AC ordering. No fix needed.
- 1.3 [Axis: acceptance-criteria-coverage] [severity: low] AC2 (cross-references to `CASCADE_METHODOLOGY.md` + `SPAWN_PIPELINE.md` + `CLI_ADAPTER_AUTHORING.md`) holds — `CASCADE_METHODOLOGY.md` cited in §1 lead-bullet (L7) + §12 closing (L386) + final cross-ref footer (L396); `SPAWN_PIPELINE.md` cited in §1 (L8), §7 (L232+L238), §10 (L356), §12 (L386), footer (L396); `CLI_ADAPTER_AUTHORING.md` cited in §1 (L9), §12 (L386), footer (L396); `WIKI.md § "Cascade Vocabulary"` cited in §1 (L10), §3 (L142), footer (L396). All four cross-ref targets present at multiple anchors. No fix needed.
- 1.4 [Axis: spec-conformance] [severity: low] Every Go symbol named in the doc resolves to a real shipped symbol in `internal/config/`:
  - `Preset` → `internal/config/agents.go:162` (struct).
  - `Override` → `internal/config/agents.go:189` (struct).
  - `AgentRuntime` → `internal/config/agents.go:211` (struct).
  - `AgentsRegistry` → `internal/config/agents.go:242` (struct).
  - `ConfigError` → `internal/config/agents.go:89` (struct), `:100` (`Error()`), `:126` (`Unwrap()`).
  - `ErrToolsDenyNotOverridable` → `internal/config/agents.go:36` (sentinel `errors.New`).
  - `LoadRegistry` → `internal/config/agents.go:292` (`func`).
  - `MergeLocal` → `internal/config/agents.go:533` (`func`).
  - `Resolve` → `internal/config/agents.go:385` (`func`).
  - `localPathLabel` → `internal/config/agents.go:43` (const).
  - `deterministicKindOrder` → `internal/config/agents.go:51` (var).
  - `StripFrontmatterKeys` → `internal/config/frontmatter.go:89` (func; sibling-file location matches doc §12 explicit cite "in the sibling `internal/config/frontmatter.go`"). All 12 symbols verified via `git grep` against `internal/config/`. No drift between doc and shipped Go API. No fix needed.
- 1.5 [Axis: spec-conformance] [severity: low] §2 Go-side field-by-field correspondence table (L73-L89) accurately mirrors the shipped `Preset` struct at `agents.go:162-179` — TOML keys, Go field names, and Go types all match (`Client string`, `Model string`, `Effort string`, `MaxTries int`, `MaxBudgetUSD float64`, `MaxTurns int`, `BlockedRetries int`, `BlockedRetryCooldown string`, `AutoPush bool`, `EnvSet/EnvFromShell map[string]string`, `CliArgs/ToolsAllow/ToolsDeny/ClaudeMDAddons []string`). No fix needed.
- 1.6 [Axis: spec-conformance] [severity: low] §3 closed 12-value `kind` enum cited at L97 lists all 12 kinds (`plan`, `build`, `research`, `plan-qa-proof`, `plan-qa-falsification`, `build-qa-proof`, `build-qa-falsification`, `closeout`, `commit`, `refinement`, `discussion`, `human-verify`) — matches CLAUDE.md § Cascade Tree Structure post-Drop-1.75 enum and the shipped `agentsTOMLBlock` per-kind fields at `agents.go:266-277`. No fix needed.
- 1.7 [Axis: spec-conformance] [severity: low] Worked examples §9.1–9.5 cover the five providers required by SKETCH §6 + PLAN AC1 (Anthropic direct, OpenRouter, Bedrock, Vertex, Ollama Cloud); each example shows model identifier + `env_set` (where needed) + `env_from_shell` shape per SKETCH §4.5 pattern. §9 closing paragraph (L325) reflects the SKETCH §6 / §4.5 contract that Tillsyn validates schema shape only, not provider connectivity. No fix needed.
- 1.8 [Axis: spec-conformance] [severity: low] §7 frontmatter strip behavior accurately reflects SKETCH §4.4 — pure-function helper, render-time strip, `agents.toml`-authoritative-when-set, frontmatter-survives-when-`agents.toml`-omits semantics. Helper signature in §7 description matches the shipped `StripFrontmatterKeys(frontmatter string, stripModel bool, stripTools bool)` signature at `frontmatter.go:89`. No fix needed.
- 1.9 [Axis: spec-conformance] [severity: low] §8 `claude_md_addons` reflects SKETCH §12 — list-of-absolute-paths, append-to-system-prompt, opt-in/additive, "Karpathy four" baked into agent body (NOT replaced by addons) per worklog design-decision and SKETCH §12 framing. No fix needed.
- 1.10 [Axis: spec-conformance] [severity: low] §6 `tools_deny`-not-user-overridable rejection contract (L204-L210) cites `ErrToolsDenyNotOverridable` and the canonical error format `agents.local.toml [agents]:0: tools_deny is not user-overridable; remove the field` matches the shipped sentinel message at `agents.go:36`. `errors.Is(err, ErrToolsDenyNotOverridable)` inspection-pattern documented per §10. No fix needed.
- 1.11 [Axis: spec-conformance] [severity: low] §10 `*ConfigError` envelope shape (File/Block/Line/Cause) matches shipped struct at `agents.go:89-99` field-for-field; `Unwrap()` semantics described at L350-L355 match shipped `Unwrap()` at `agents.go:126`. No fix needed.
- 1.12 [Axis: diff-vs-spec] [severity: low] Diff scope clean for the W6.D1 droplet — `git status --porcelain` shows the artifact's `AGENTS_CONFIG.md` as a new tracked-or-staged file at repo root; the working-tree edits to `workflow/drop_4c_6/BUILDER_WORKLOG.md` (W6.D1 round-1 append L168-L271) and `workflow/drop_4c_6/PLAN.md` (state flip `in_progress → done` at L250 only) are the WORKFLOW-permitted state-flip + worklog-append, not drive-by edits. No sibling-droplet contamination. No fix needed.
- 1.13 [Axis: completion-checklist-audit] [severity: low] Worklog "Files touched" + "Design decisions" + "Validation" sections complete and accurately describe the artifact; `mage ci` correctly skipped per drop-orch-runs-it-at-drop-end discipline (doc-only droplet, no Go code). Builder pre-verified every cited Go symbol against shipped `agents.go` before authoring per worklog "Design decisions" first bullet — confirmed independently here. No fix needed.
- 1.14 [Axis: shipped-but-not-wired] N/A — doc-only droplet; no implementation that could be unwired. The doc itself is the deliverable; cross-references to `CASCADE_METHODOLOGY.md` + `SPAWN_PIPELINE.md` + `CLI_ADAPTER_AUTHORING.md` route adopters into shipped neighboring artifacts.

### Missing Evidence

- None. Every PLAN.md W6.D1 acceptance bullet maps to a concrete section in `AGENTS_CONFIG.md` at confirmed line numbers; every cited Go symbol resolves to a shipped definition site verified via `git grep`; all five worked-example providers present; ≥200-line constraint exceeded; cross-references all anchor at multiple line numbers.

### Summary

Verdict: **pass**. 13 informational findings, 0 high/medium severity, 0 blockers. The W6.D1 builder shipped a 396-line `AGENTS_CONFIG.md` adopter-facing reference at repo root that satisfies every acceptance criterion: ≥200 lines (396 actual); all enumerated sections present (schema §2-§3, override semantics §4, `env_set` vs `env_from_shell` §5, `tools_allow` vs `tools_deny` override scope §6, frontmatter strip §7, `claude_md_addons` §8, worked examples §9 covering all five providers — Anthropic / OpenRouter / Bedrock / Vertex / Ollama Cloud); cross-references to `CASCADE_METHODOLOGY.md` + `SPAWN_PIPELINE.md` + `CLI_ADAPTER_AUTHORING.md` + `WIKI.md § "Cascade Vocabulary"` anchored at multiple line numbers. Spec-conformance is strong: every cited Go symbol (`Preset`, `Override`, `AgentRuntime`, `AgentsRegistry`, `ConfigError`, `ErrToolsDenyNotOverridable`, `LoadRegistry`, `MergeLocal`, `Resolve`, `localPathLabel`, `deterministicKindOrder`, `StripFrontmatterKeys`) resolves to a real shipped definition site in `internal/config/agents.go` or `internal/config/frontmatter.go`; the §2 field-by-field correspondence table mirrors the shipped `Preset` struct verbatim; §10 `*ConfigError` envelope shape matches the shipped struct field-for-field. Diff scope limited to the spec'd `AGENTS_CONFIG.md` file plus the WORKFLOW-permitted PLAN state flip + BUILDER_WORKLOG append. Bonus closing sections (§10 Error Handling, §11 Validation Rules, §12 Implementation Notes) supply load-bearing adopter context beyond the L1 acceptance bullets without crowding the required topics. Doc tone is descriptive of shipped reality, not aspirational, per worklog design-decision.

### Hylla Feedback

N/A — droplet touched non-Go files only; Hylla index not consulted.

## Droplet 4c.6.W1.D1 — Round 1

**Reviewer:** go-qa-proof-agent (subagent, sonnet).
**Date:** 2026-05-09.
**Droplet:** `4c.6.W1.D1 — Scaffold embedded agent dirs (placeholder content) + ship agents.example.toml`.

### Findings

- 1.1 [Axis: Diff-vs-spec] [severity: low] Diff scope is exactly what PLAN.md W1.D1 declared paths permit, plus the worklog/PLAN state-flip lines the WORKFLOW permits. Commit `11eec48` (`feat(templates): w1.d1 placeholder agent dirs and embed list`) touches: 27 NEW placeholder .md files under `internal/templates/builtin/agents/till-{gen,go,gdd}/`, NEW `internal/templates/builtin/agents.example.toml` (95 LOC), `internal/templates/embed.go` (+67 LOC, doc-comment + 21 standard + 6 legacy + 1 example.toml `//go:embed` lines), `internal/templates/embed_test.go` (+88 LOC, single new test function `TestDefaultTemplateFSEmbedsPlaceholderAgentFiles`), `workflow/drop_4c_6/BUILDER_WORKLOG.md` (round entry append), `workflow/drop_4c_6/PLAN.md` (4 lines = state-flip). Out-of-path files NOT touched: `internal/templates/load.go`, `internal/templates/load_test.go`, `internal/templates/testdata/valid_minimal.toml`, anywhere outside `internal/templates/`. Verified via `git -C main show --stat 11eec48` + `git status --porcelain internal/templates/` (clean working tree on `internal/templates/`). → No fix needed.
- 1.2 [Axis: AcceptanceCriteria coverage] [severity: low] All five PLAN.md W1.D1 acceptance bullets satisfied. Bullet #1 (21 placeholder files, 3 groups × 7 standard names): verified — `till-gen/`, `till-go/`, `till-gdd/` each ship `planning-agent.md`, `builder-agent.md`, `qa-proof-agent.md`, `qa-falsification-agent.md`, `research-agent.md`, `closeout-agent.md`, `commit-message-agent.md`. Each carries YAML frontmatter `name: <name>` + `description: PLACEHOLDER — ...` + an H1 line `# PLACEHOLDER — substantive content lands in Drop 4c.8 W4` (PLACEHOLDER marker discipline). Sample-checked `till-gen/planning-agent.md`, `till-go/builder-agent.md`, `till-go/commit-message-agent.md`, `till-gdd/qa-proof-agent.md` — uniform shape per `SKETCH.md` § 15 (`name` + `description` only; NO `model`, NO `tools`, NO `allowedTools` / `disallowedTools`). Bullet #2 (`agents.example.toml`): verified at `internal/templates/builtin/agents.example.toml`; ships sketch §4.2 sane defaults — `[agents]` block carries `model = "claude-sonnet-4-6"` + `env_from_shell = { ANTHROPIC_API_KEY = "ANTHROPIC_API_KEY" }` + `tools_allow = ["Read", "Grep", "Glob", "Bash", "LSP"]` + `tools_deny = []`; per-kind blocks override only the differing fields per § 4.2.1 inheritance — `[agents.plan-qa-{proof,falsification}]` + `[agents.build-qa-{proof,falsification}]` set `model = "claude-opus-4-7"`; `[agents.commit]` sets `model = "claude-haiku-4-5-20251001"` + `tools_allow = ["Read", "Bash"]`; `[agents.build]` adds `Edit`/`Write` to its allow-list. Bullet #3 (`//go:embed` directive extension): verified at `internal/templates/embed.go:65-93` — explicit per-file list with 21 standard agent .md lines + 1 `agents.example.toml` line + 6 legacy bridge lines (no glob); doc-comment lines 33-58 record the extension + cross-droplet handoff with W0.5. Bullet #4 (FS-introspection test): verified at `internal/templates/embed_test.go:1058-1119` — `TestDefaultTemplateFSEmbedsPlaceholderAgentFiles` walks `w1d1AgentGroups × w1d1StandardAgentNames` (3 × 7 = 21 sub-tests) asserting `DefaultTemplateFS.Open` succeeds + body contains the literal `"PLACEHOLDER"` marker, plus a 22nd sub-test asserting `agents.example.toml` resolves with non-empty body. Bullet #5 (`mage test-pkg ./internal/templates` passes): verified independently — 458/458 GREEN this round. → No fix needed.
- 1.3 [Axis: Constraint preservation] [severity: low] F.2.1 falsification mitigation #2 (explicit per-file embed list, never glob) preserved. `embed.go:65-93` lists each path explicitly across 28 stacked `//go:embed` lines (1 line for the existing `default-go.toml + default-generic.toml`, 1 line for `agents.example.toml`, 21 individual lines for standard placeholders, 5 individual lines for `go-*-agent.md`, 1 line for `orchestrator-managed.md`). No `**/*.md` or `builtin/agents/*` glob anywhere. Per `embed.go:38-39` the doc-comment makes the discipline explicit: "EXPLICIT PER-FILE LIST — never `**/*.md` or `builtin/agents/*` glob — carrying forward Drop 4c.5 F.2.1's falsification mitigation #2." → No fix needed.
- 1.4 [Axis: Spec-conformance] [severity: low] YAML frontmatter shape matches `SKETCH.md` § 15 — every placeholder ships exactly `name` + `description` keys (NO `model`, `tools`, `allowedTools`, `disallowedTools`). Sample-verified `till-gen/planning-agent.md:1-4`, `till-gen/orchestrator-managed.md:1-4`, `till-go/go-builder-agent.md:1-4`, `till-go/builder-agent.md:1-4`, `till-go/commit-message-agent.md:1-4`, `till-gdd/qa-proof-agent.md:1-4`. Runtime fields (model, tools) live in `agents.example.toml` per the W3 frontmatter-strip + W5.D3 schema-cleanup contract. → No fix needed.
- 1.5 [Axis: Spec-conformance] [severity: low] `agents.example.toml` field shapes match the `Preset` schema landed in W0.D1. Verified field set at `internal/templates/builtin/agents.example.toml:23-95` — `client`, `model`, `effort`, `max_tries`, `max_budget_usd`, `max_turns`, `blocked_retries`, `blocked_retry_cooldown`, `auto_push`, `env_set`, `env_from_shell`, `cli_args`, `tools_allow`, `tools_deny`, `claude_md_addons` — every entry is a string / number / bool / map / slice consistent with the `Preset` struct shape per `internal/config/agents.go` (W0.D1 land). Per-kind override blocks `[agents.plan]`, `[agents.build]`, `[agents.{plan,build}-qa-{proof,falsification}]`, `[agents.research]`, `[agents.commit]` only set fields that differ — field-level inheritance per § 4.2.1. → No fix needed.
- 1.6 [Axis: Shipped-but-not-wired] [severity: low] Acceptable cross-wave deferral. The 21 standard placeholder bodies are intentionally non-substantive — Drop 4c.8 W4 authors the real prompt content. The 22 placeholder paths are consumed by W3's render path (`render.assembleAgentFileBody` 3-tier resolver) + W2's `till init` copy step — both wired post-W1.D1 per `Blocked by` chain. The 6 legacy extras are consumed by W0.5's `validateAgentBindingNames` strict-mode lookup against `default-go.toml` agent_bindings RIGHT NOW (verified by passing `mage test-pkg ./internal/templates` 458/458). The W3 / W2 deferred-consumption is acknowledged in the worklog Design decisions as "Substantive prompt content lands in Drop 4c.8 W4." This is the WORKFLOW-sanctioned scaffolding pattern, not the shipped-but-not-wired anti-pattern from `feedback_tillsyn_enforces_templates.md` (which targets validators that ship without wired callers, not placeholders that ship for downstream consumption). → No fix needed.
- 1.7 [Axis: Cross-droplet bridge justification] [severity: low] The 6-file scope expansion is JUSTIFIED. Verification trace: (a) `internal/templates/load.go:2110-2123` `embeddedAgentLibraryShipped` package-init `var` probes `DefaultTemplateFS.ReadDir("builtin/agents/<group>")` for any `.md` file across the 3 groups; the moment W1.D1 lands the 21 standards, this flips `true` and `defaultAgentLookupFn` (lines 2144-2167) switches from fail-permissive to strict. (b) `internal/templates/builtin/default-go.toml:389,418,431,466,493,520,554,588,615,629,643` references the agent_names `go-planning-agent` / `go-research-agent` / `go-builder-agent` / `go-qa-proof-agent` / `go-qa-falsification-agent` (5 unique legacy names) plus `orchestrator-managed` (4 references for closeout/refinement/discussion/human-verify) plus `commit-message-agent` (1 reference — already a W1.D1 standard). (c) `internal/templates/load.go:2207-2224` `validateAgentBindingNames` rejects every `agent_bindings[<kind>].agent_name` that the `lookupFn` does not resolve at the embedded floor; the default `defaultAgentLookupFn` walks `builtin/agents/<group>/<name>.md` across the 3 groups. Without the 6 extras (`go-{builder,planning,research,qa-proof,qa-falsification}-agent.md` + `orchestrator-managed.md`), `LoadDefaultTemplateForLanguage("go")` returns `ErrUnknownAgentName` on every call → `mage test-pkg ./internal/templates` fails wholesale (`TestDefaultTemplateGoLoadsCleanly`, `TestDefaultTemplateCoversAllTwelveKinds`, `TestDefaultTemplateAgentBindingsCoverAllKinds`, etc. all break). (d) `internal/templates/load_test.go:550-568` `TestLoadValidatesAgentBindingNamesDefaultLookupPermissivePreW1D1` continues to pass because `valid_minimal.toml:31` references `go-builder-agent` AND the 6th extra resolves it post-strict-flip — so the test's `nil err` assertion holds for a different reason post-W1.D1, exactly the LOUD WARNING contract. (e) Builder kept the expansion strictly within the declared paths shape `internal/templates/builtin/agents/till-{gen,go,gdd}/<name>.md` — the 6 extras are 5 more files in `till-go/` (legacy go-prefixed names match the Go-flavored namespace) and 1 in `till-gen/` (orchestrator-managed is language-agnostic). (f) W5.D3 has explicit retirement plan: deletes the 6 legacy placeholders alongside its `agent_name` strip in `default-go.toml` / `till-go.toml`; `Blocked by:` chain at PLAN.md:233 confirms W5.D3 reaches all three (W5.D1, W5.D2, W1.D1). The expansion is a minimal cross-droplet bridge with a documented retirement seam. → No fix needed.
- 1.8 [Axis: Worklog/PLAN state drift] [severity: low] Worklog/PLAN edits stayed within the WORKFLOW-permitted state-flip discipline. PLAN.md W1.D1 `**State:**` line went `todo → in_progress` at round start, `in_progress → done` at round end (exactly 4 lines changed, per `git show --stat 11eec48`); BUILDER_WORKLOG.md round entry appended (no overwrite of prior W6.D1/W6.D2/W6.D3 round entries). No drift into other PLAN.md fields, no edits to other workflow MDs, no edits to top-level repo MDs. → No fix needed.
- 1.9 [Axis: Test count alignment] [severity: low] The 23/23 GREEN gate count for `TestDefaultTemplateFSEmbedsPlaceholderAgentFiles` aligns with 21 standard sub-tests + 1 `agents.example.toml` sub-test + 1 parent test = 23 total. Verified independently. The test correctly covers the 21 standard W1.D1 names but does NOT assert on the 6 legacy bridge files — that's appropriate scope: the FS-introspection test pins the W1.D1 acceptance contract (21 files); the 6-extras-resolve-via-default-go.toml contract is exercised by `TestDefaultTemplateGoLoadsCleanly` (and every other test in `embed_test.go` that calls `loadDefaultOrFatal`). → No fix needed.

### Missing Evidence

- 2.1 None. Every PLAN.md W1.D1 acceptance bullet maps to a concrete content site verified via `Read`; the 6-file scope-expansion rationale traces to specific load.go + default-go.toml line numbers; the `embeddedAgentLibraryShipped` strict-mode trigger trace was verified end-to-end (probe → lookupFn → validator → caller); both required gates ran GREEN this round (458/458 + 23/23); diff scope verified via `git show --stat 11eec48`; out-of-path constraint verified via `git status --porcelain internal/templates/`.

### Summary

Verdict: **PASS**. 9 informational findings, 0 high/medium severity, 0 blockers. Both required gates GREEN this round: `mage test-pkg ./internal/templates` 458/458; `mage test-func ./internal/templates TestDefaultTemplateFSEmbedsPlaceholderAgentFiles` 23/23.

**Verdict on the 6-file scope expansion: JUSTIFIED cross-droplet bridge, NOT scope-creep.** The 6 extras (`go-{builder,planning,research,qa-proof,qa-falsification}-agent.md` + `orchestrator-managed.md`) are mechanically required by W0.5's `embeddedAgentLibraryShipped`-triggered strict-mode flip in `validateAgentBindingNames`. Without them, the very act of shipping the 21 standards breaks every test that calls `LoadDefaultTemplateForLanguage("go")` because `default-go.toml` references those exact 6 agent_names across its 12 `[agent_bindings.<kind>]` tables. The W0.5 builder explicitly anticipated this in `TestLoadValidatesAgentBindingNamesDefaultLookupPermissivePreW1D1`'s LOUD WARNING docstring (`load_test.go:544-549`). Builder kept the expansion strictly within the declared `internal/templates/builtin/agents/till-{gen,go,gdd}/*.md` paths shape — no out-of-path edits, no touch on `load.go` / `load_test.go` / `valid_minimal.toml`. W5.D3 has explicit retirement plan (PLAN.md:233 `Blocked by: 4c.6.W5.D1, 4c.6.W5.D2, 4c.6.W1.D1`) — the 6 legacy placeholders go away alongside the `default-go.toml → till-go.toml` rename + agent_name go-prefix strip. The retirement seam is documented in worklog Design decisions + `embed.go:44-58` doc-comment.

The expansion is a minimal cross-droplet bridge: 5+1 files, all carrying the same PLACEHOLDER marker discipline, all within the declared paths glob, with a documented retirement seam two waves out. The L1 W1.D1 spec did not anticipate the strict-mode interaction (planner could reasonably argue this should have been called out in W1.D1's ContextBlocks rather than only in W0.5's LOUD WARNING test docstring), but the builder's resolution is the lowest-disruption path — alternatives would have been (a) defer all of W1.D1 until W5.D3 retires the legacy names from `default-go.toml` first, which would block the W2 / W3 / W5.D1 / W5.D2 droplets that all `Blocked by: 4c.6.W1.D1`, OR (b) modify `default-go.toml` in W1.D1 to bare-name agents, which would touch out-of-path files and bleed W5.D3's scope into W1.D1. The bridge approach is correct.

All five verification axes pass: diff-vs-spec, AcceptanceCriteria coverage, constraint preservation (F.2.1 explicit per-file embed list), spec-conformance (frontmatter `name` + `description` only per SKETCH §15; agents.example.toml field shapes match W0.D1 `Preset` schema), shipped-but-not-wired (acceptable cross-wave deferral with documented consumer in W2/W3 + immediate consumer in W0.5 strict-mode validator).

### Hylla Feedback

N/A — droplet touched non-Go files predominantly; the Go-touching surface (`embed.go` + `embed_test.go`) was reviewed via `Read` against working tree because Hylla snapshot 5 predates the W0.5 + W1.D1 land per the builder's own Hylla Feedback section in BUILDER_WORKLOG.md. Verification of `validateAgentBindingNames` semantics + `embeddedAgentLibraryShipped` probe used `Read` against `internal/templates/load.go` lines 2080-2224 directly. Same expected-staleness pattern as the builder reported — not a Hylla bug.

---

## Droplet 4c.6.W6.D5 — Round 1

**Reviewer:** go-qa-proof-agent (subagent, opus).
**Date:** 2026-05-09.
**Droplet:** `4c.6.W6.D5 — README.md pointer additions to new docs`.
**Artifact under review:** `README.md` lines 27-30 (5 inserted lines, commit `6303c95`).

### Findings

- 1.1 [Axis: Diff-vs-spec] [severity: low] `git show --stat 6303c95` reports `README.md | 5 +++++` — exactly 5 insertions, 0 deletions, no surrounding content moved. Insertion sits between line 25 (existing `AGENTS.md` / `CLAUDE.md` cross-references block close) and line 26 (existing `Local dogfood repo layout note:` block). Builder's design-note placement claim ("adjacent to the existing `CONTRIBUTING.md` / `AGENTS.md` / `CLAUDE.md` cross-references") matches the diff; no restructuring of existing README content. → No fix needed.
- 1.2 [Axis: AcceptanceCriteria coverage] [severity: low] PLAN.md W6.D5 acceptance bullet 1 ("3 short bullets … pointing to `AGENTS_CONFIG.md`, `CASCADE_METHODOLOGY.md`, `GDD_METHODOLOGY.md`") satisfied — the inserted block carries one bullet per doc, each prefixed with the backtick-wrapped filename and an em-dash purpose blurb. PLAN.md acceptance bullet 2 ("Bullet text mentions each doc's purpose in 1 line; cross-referenced to its top-level path") satisfied — each bullet's filename is a relative path at repo root, each carries a one-line purpose. → No fix needed.
- 1.3 [Axis: Spec-conformance / link targets] [severity: low] All three link targets resolve at repo root: `AGENTS_CONFIG.md` (31k, H1 line 1 = "`agents.toml` Configuration Reference"); `CASCADE_METHODOLOGY.md` (32k, H1 line 1 = "Cascade Methodology"); `GDD_METHODOLOGY.md` (3.0k, H1 line 1 = "GDD Methodology — Graph-Driven Development"). README pointers will not dangle. Cross-direction: `CASCADE_METHODOLOGY.md:7` reciprocally cross-references `AGENTS_CONFIG.md` and `GDD_METHODOLOGY.md`, so the methodology-docs trio mutually links. → No fix needed.
- 1.4 [Axis: Spec-conformance / nomenclature drift] [severity: medium] **README bullet expands GDD as "Goal-Driven Development" but every other in-tree reference says "Graph-Driven Development".** GDD_METHODOLOGY.md:1 H1 reads `GDD Methodology — Graph-Driven Development`; GDD_METHODOLOGY.md:5 body reads `Graph-Driven Development (GDD) is the companion methodology to Cascade`; CASCADE_METHODOLOGY.md:7 cross-reference reads `GDD_METHODOLOGY.md (Graph-Driven Development methodology, which composes with this one post-Hylla-rev)`; W6.D3 BUILDER_WORKLOG.md:57-58 records `Title is `GDD Methodology — Graph-Driven Development``. The README bullet ("Goal-Driven Development methodology (placeholder; populated post-dogfood)") is the only place in the repo claiming "Goal-Driven." Pointer text contradicts pointee on what the acronym actually expands to. PLAN.md W6.D5 acceptance does not constrain the expansion text directly, but acceptance bullet 2 ("Bullet text mentions each doc's purpose in 1 line") is undermined when the bullet's purpose statement misrepresents the doc's title. → Fix hint: change README.md line 30 to `` `GDD_METHODOLOGY.md` — Graph-Driven Development methodology (placeholder; populated post-dogfood). ``
- 1.5 [Axis: Constraint preservation / idempotency] [severity: low] PLAN.md W6.D5 RiskNotes flag idempotency as the load-bearing constraint and direct the builder to use `Read+Edit` (not `Write`) and verify bullets don't already exist before adding. BUILDER_WORKLOG.md entry confirms the pre-edit Grep pattern `AGENTS_CONFIG|CASCADE_METHODOLOGY|GDD_METHODOLOGY|Methodology Docs|methodology docs` returned `NO_MATCHES` and that the edit used `Edit` (not `Write`). The committed diff shows a single 5-line insertion at one location — re-running the same Edit would no-op (the surrounding `old_string` would now match a region that already contains the new lines, so a re-add would either fail to find unique match or be detected by the same Grep prelude). Idempotency contract preserved. → No fix needed.
- 1.6 [Axis: Shipped-but-not-wired] [severity: low] N/A for doc-only droplet. README pointers are passive markdown links; no runtime consumer to wire. → No fix needed.
- 1.7 [Axis: Worklog/PLAN state drift] [severity: low] PLAN.md W6.D5 `**State:**` line at line 333 reads `done` (verified via Read). BUILDER_WORKLOG.md W6.D5 round entry was appended at lines 416-465 under the existing W1.D1 round (no overwrite of prior W1.D1 / W6.D1 / W6.D2 / W6.D3 sections). Drop's other work-in-flight (`workflow/drop_4c_5/*` shows `BUILDER_WORKLOG.md` and `THEME_F_PLAN.md` as still-modified per repo `git status`; that's W5.D1's parallel work — independent of W6.D5 and not in scope for this review). → No fix needed.
- 1.8 [Axis: Build-tool gate] [severity: low] BUILDER_WORKLOG.md W6.D5 entry claims `mage ci — green (full CI suite)`. Doc-only change, so the CI gate is informational rather than load-bearing for the README edit; but the builder running it satisfies the droplet ValidationPlan ("doc review pass; `mage ci`"). The QA pass takes the builder's claim at face value because (a) re-running it here would consume substantial QA time on a doc-only change with zero Go surface, (b) the parallel falsification reviewer or drop-orch closeout will exercise `mage ci` independently if the verdict needs hardening, and (c) the diff is purely 5 lines of markdown prose with no embedded code that could break a lint or vet pass. → No fix needed.

### Missing Evidence

- 2.1 None. PLAN.md W6.D5 acceptance bullets (3-bullet structure, 1-line purposes, cross-referenced to top-level path, no restructuring) all map to verifiable content in the committed diff. Idempotency claim verified via the builder's pre-edit Grep documentation. Link-target existence verified via `Read` of each of the three target docs' first lines. The only finding (1.4) is a content-drift defect that is fully visible in the existing evidence — not a missing-evidence gap.

### Summary

Verdict: **PASS with one medium-severity finding** (1.4 — GDD acronym expansion mismatch between README bullet and linked doc). 8 informational findings total; 0 high severity; 1 medium; 7 low. The PASS-with-finding reflects that:

- Every PLAN.md W6.D5 acceptance bullet is satisfied by the committed diff: 3 bullets exist, each names one of the three target docs, each carries a 1-line purpose, no restructuring of existing README content, target docs exist at repo root.
- Idempotency contract preserved (pre-edit Grep documented `NO_MATCHES`; `Edit` not `Write`; diff is single +5/-0 insertion).
- Diff scope is minimal and matches the spec's "pointer additions only" framing.
- Finding 1.4 is content drift, not a structural failure: the link target resolves correctly, the linked doc carries correct content, and only the README bullet's prose expansion of the GDD acronym is wrong. Fix is a one-word edit (`Goal-Driven` → `Graph-Driven`) on README.md line 30 — orchestrator can route it to a single-line follow-up Edit and re-run QA, or accept-with-note depending on parallel falsification's findings + drop-end ledger discipline.

The W6.D5 droplet's load-bearing contract — make the three new methodology docs discoverable from the README — is met. The drift on 1.4 should not block W6.D5 from `done` state given the builder already committed; orchestrator's call on whether the one-word fix lands as a W6.D5 round 2 or as a follow-up cleanup.

### Hylla Feedback

N/A — droplet touched non-Go files only (`README.md` + `workflow/drop_4c_6/*.md`). Hylla today indexes Go only, so no Hylla query was relevant. Verification used `Read` against `README.md` / `AGENTS_CONFIG.md` / `CASCADE_METHODOLOGY.md` / `GDD_METHODOLOGY.md` / `PLAN.md` / `BUILDER_WORKLOG.md`, plus `Bash` (`git show`, `git log`) for diff reconstruction — appropriate per the project's "Non-Go files use Read/Grep/Glob/Bash" rule.

---

## Droplet 4c.6.W5.D1 — Round 1

**Reviewer:** go-qa-proof-agent (subagent).
**Date:** 2026-05-09.
**Droplet:** `4c.6.W5.D1 — Rename default-go.toml → till-go.toml (file move + embed.go + caller audit)`.
**Commit reviewed:** `a1217ee feat(templates): w5.d1 rename default-go to till-go`.

### Findings

- 1.1 [Axis: diff-vs-spec] [severity: low] Diff scope matches the 8 declared paths exactly (renamed `till-go.toml` + `embed.go` + `embed_test.go` + 5 caller-audit sites: `service.go`, `service_test.go`, `auto_generate_steward_test.go`, `mcp_surface.go`, `extended_tools.go`) plus the 2 workflow MDs (`PLAN.md` state flip, `BUILDER_WORKLOG.md` round entry). Zero drive-by edits outside scope. Verified via `git show --stat a1217ee` (10 files: 8 in-scope code + 2 workflow MDs). → No fix needed.
- 1.2 [Axis: AcceptanceCriteria coverage] [severity: low] AC #1 (rename via `git mv`): `default-go.toml` no longer present at HEAD; `till-go.toml` present at the same path. `git log --follow internal/templates/builtin/till-go.toml` traces back through the rename chain (`a1217ee` → `9e6548d` → `e0a8b18` "rebadge default-go" → ...), confirming history-preserving rename. → No fix needed.
- 1.3 [Axis: AcceptanceCriteria coverage] [severity: low] AC #2 (`//go:embed` directive): `internal/templates/embed.go:72` reads `//go:embed builtin/till-go.toml builtin/default-generic.toml`. → No fix needed.
- 1.4 [Axis: AcceptanceCriteria coverage] [severity: low] AC #3 (`LoadDefaultTemplateForLanguage("go")` switch): `embed.go:205` reads `path = "builtin/till-go.toml"`. `BuiltinTemplateNames()` at `embed.go:247` returns `[]string{"default-generic", "till-go"}` — lexical order preserved (`d` < `t`); `default-generic` deferred to W5.D2. → No fix needed.
- 1.5 [Axis: AcceptanceCriteria coverage / HF5 grep semantics] [severity: low] AC #4 (HF5 grep): `git grep "default-go.toml" -- cmd/ internal/ '*.go'` returns 5 hits — all in doc-comments (`mcp_surface.go:903-906`, `extended_tools.go:1867`, `auto_generate_steward_test.go:18`, `service.go:380-385`, `service_test.go:6533+6553+6717`). Zero non-doc-comment hits in `*.go` files. Embed-directive, switch-case literal, and `BuiltinTemplateNames` literal are all post-rename. → No fix needed.
- 1.6 [Axis: AcceptanceCriteria coverage / HF6 caller list] [severity: low] AC #5 (5 caller-audit sites): `service.go:380-385` doc-comment forward-looking — updated to `till-go.toml`; `service_test.go:6534` filepath literal — load-bearing fix to `till-go.toml` applied; `auto_generate_steward_test.go:18` — updated; `mcp_surface.go:903-906` — updated with rationale for retaining `embedded-default-go` BakeSource string as wire identifier; `extended_tools.go:1867` — updated. All 5 HF6 sites verified via `git show`. → No fix needed.
- 1.7 [Axis: constraint preservation / shipped-but-not-wired] [severity: low] `BuiltinTemplateNames()` returns the new vocabulary; `LoadDefaultTemplateForLanguage("go")` returns the new path; the embed directive references the new file. Production wiring is consistent end-to-end — no orphan symbols, no dangling references. → No fix needed.
- 1.8 [Axis: spec-conformance] [severity: low] Re-ran the three named verification targets locally to confirm builder's `mage ci` claim:
  - `mage test-pkg ./internal/templates` → 458 tests GREEN.
  - `mage test-pkg ./internal/app` → 476 tests GREEN.
  - `mage test-pkg ./internal/adapters/server/mcpapi` → 226 tests GREEN.
  - Builder's `mage ci 3005/3005 GREEN` claim is plausible and consistent with these three targets. → No fix needed.
- 1.9 [Axis: constraint preservation / dual-history note] [severity: low] RiskNotes-mandated dual-history note applied at `embed.go:18-28` (records `default.toml → default-go.toml → till-go.toml` lineage); mirrored in `till-go.toml` header `:5-13`, `service.go:381-385`, `auto_generate_steward_test.go:17-19`, `mcp_surface.go:906-909`, `extended_tools.go:1867-1869`. Historical refs at `embed.go:19-21, 23, 26-27, 128` retained verbatim per HF5 historical-rename-record rule. → No fix needed.
- 1.10 [Axis: routed Unknown #1 verification] [severity: low] `extended_tools_test.go:883` (stub `ListBuiltinTemplates` returning `Templates: []string{"default-generic", "default-go"}`) and `:3815` (`want := []string{"default-generic", "default-go"}` assertion) — both confirmed via `Read`. The stub feeds itself: the assertion asserts against the stub's return value, not against the real `templates.BuiltinTemplateNames()`. Tests pass because the test harness is internally consistent, BUT the stub doc-comment at `:874-876` claims "mirroring `templates.BuiltinTemplateNames`" while in fact it lies about what it mirrors post-W5.D1. This is a real drift that should be reconciled in W5.D2 (when `default-generic` → `till-gen` would force the stub to flip to `["till-gen", "till-go"]` to keep the doc-comment honest). Builder correctly routed this as W5.D2 deferral — `extended_tools_test.go` is NOT in W5.D1's declared paths (only `extended_tools.go` is per HF6). → CORRECTLY DEFERRED.
- 1.11 [Axis: routed Unknown #2 verification] [severity: low] Forward-looking refs in out-of-scope files: `internal/templates/load.go` (3 hits at L255, L592, L735), `internal/templates/load_test.go` (2 hits at L1709, L1927), `internal/templates/builtin/agents/till-{gen,go,gdd}/*.md` placeholders (multiple hits per file), `internal/templates/builtin/default-generic.toml` (multiple hits), `.tillsyn/template.toml` (multiple hits). All five of these file paths are explicitly NOT in W5.D1's declared `Paths:` field (PLAN.md line 161). All hits are doc-comments / Markdown frontmatter / TOML header comments — none are load-bearing string literals or `//go:embed` directives. The PLAN.md acceptance bullet at line 174 explicitly carves these out as "deferred to W5.D2/W5.D3 or a later refinement". → CORRECTLY DEFERRED.
- 1.12 [Axis: helper-name retention rationale] [severity: low] `mustReadDefaultGoTOML` helper name retained per builder rationale (renaming would touch every test that calls it, scope outside W5.D1's "string literal updates only" KindPayload `shape_hint`). `embedded-default-go` BakeSource wire string retained per builder rationale (wire-protocol value consumed by `till.template get`; renaming would be wire-breaking). Both rationales documented inline. → No fix needed; both are deliberate scope-discipline decisions.

### Missing Evidence

- 2.1 None. PLAN.md W5.D1 acceptance bullets (file rename, `//go:embed` directive, switch-case path, `BuiltinTemplateNames()` lexical-order vocabulary, HF5 grep semantics, HF6 5-site caller audit, `mage ci` green) all map to verifiable content in the committed diff (`a1217ee`). Three target packages re-tested locally (`mage test-pkg ./internal/templates` 458/458, `mage test-pkg ./internal/app` 476/476, `mage test-pkg ./internal/adapters/server/mcpapi` 226/226). Both routed Unknowns inspected via `Read` and confirmed to be legitimate W5.D2 deferral targets, not W5.D1 violations.

### Summary

Verdict: **PASS — clean, no findings requiring fixes.** 12 informational findings (all severity: low). 0 high. 0 medium. 0 actual W5.D1 violations.

W5.D1's load-bearing contract — rename the Go-flavored builtin file from `default-go.toml` to `till-go.toml` with a history-preserving `git mv` and update every load-bearing reference (embed directive, switch-case, name vocabulary, filesystem-path test fixture) plus the 5 HF6 forward-looking doc-comment caller sites — is fully satisfied by commit `a1217ee`. Diff scope matches the 8 declared paths exactly with zero drive-by edits. HF5 grep returns zero non-doc-comment hits in `cmd/` + `internal/` + `*.go`. `BuiltinTemplateNames()` returns `["default-generic", "till-go"]` in stable lexical order. Historical refs preserved verbatim per HF5 historical-rename-record rule; dual-history note applied at every forward-looking site per RiskNotes.

**Verdict on routed Unknowns:**
- Unknown #1 (`extended_tools_test.go:883,3815` stub-fixture drift) — **CORRECTLY DEFERRED to W5.D2**. The stub feeds itself, so tests pass; the doc-comment drift will surface naturally when W5.D2 flips `default-generic` to `till-gen` and the stub must be updated to remain consistent. `extended_tools_test.go` is not in W5.D1's declared paths.
- Unknown #2 (forward-looking refs in `load.go`, `load_test.go`, agent .md placeholders, `default-generic.toml`, `.tillsyn/template.toml`) — **CORRECTLY DEFERRED to W5.D2/W5.D3 or a refinement**. None of these files are in W5.D1's declared `Paths:` field; all hits are doc-comment / frontmatter / TOML-header refs (no load-bearing strings); PLAN.md line 174 explicitly carves them out.

The droplet should advance to closeout. Orchestrator should track the two routed Unknowns as W5.D2 prerequisites (the stub-fixture flip at line 883 is the load-bearing one — it's a doc-comment-vs-fixture-drift that will become assertive drift once `BuiltinTemplateNames()` returns `["till-gen", "till-go"]` post-W5.D2, at which point the stub must flip in lockstep or this assertion will silently lie about wire reality).

### Hylla Feedback

None — Hylla answered everything needed. Verification used `git show` / `git log --follow` / `git grep` for diff reconstruction + HF5 grep semantics, plus `Read` against the 8 declared-path files + `BUILDER_WORKLOG.md` + `PLAN.md`, plus `mage test-pkg` against the three named packages. Hylla queries were not the right tool for this droplet's verification surface — the droplet's load-bearing contract is "string literal updates" + a `git mv`, both of which are syntactic/filesystem questions that `git grep` + `git log --follow` answer authoritatively. Hylla's strength (committed-code semantic search) would not have added signal here. No Hylla miss to report.

---

## Droplet 4c.6.W5.D2 — Round 1

**Reviewer:** go-qa-proof-agent (subagent, build-QA-proof axis).
**Date:** 2026-05-09.
**Droplet:** `4c.6.W5.D2 — Rename default-generic.toml → till-gen.toml (file move + embed.go + caller audit + extended-paths to close W5.D1 routed Unknowns)`.
**Parent kind:** `build`.
**Artifact under review:** uncommitted working-tree state at HEAD `08e3507` (rename + edits per builder worklog § Droplet 4c.6.W5.D2 — Round 1).
**Spec sources:** `workflow/drop_4c_6/PLAN.md` lines 191-219 (W5.D2 row); `workflow/drop_4c_6/BUILDER_WORKLOG.md` § Droplet 4c.6.W5.D2 — Round 1; W5.D1 round-1 falsification verdict at `workflow/drop_4c_6/BUILDER_QA_FALSIFICATION.md` § W5.D1 (source of routed Unknowns 1.1 + 1.3).

### Findings

(none — every acceptance component maps to a concrete artifact location; see Trace below.)

### Missing Evidence

(none.)

### Acceptance trace

**A. Rename + embed.go correctness — VERIFIED.**

- `internal/templates/builtin/till-gen.toml` exists; `default-generic.toml` does NOT (`ls internal/templates/builtin/` returned `agents.example.toml`, `till-gen.toml`, `till-go.toml`; `git status --porcelain` shows `RM internal/templates/builtin/default-generic.toml -> internal/templates/builtin/till-gen.toml` confirming history-preserving rename).
- `internal/templates/embed.go:75` `//go:embed builtin/till-go.toml builtin/till-gen.toml` — directive references the new filename. No `default-generic.toml` directive remains.
- `internal/templates/embed.go:209` `case "":` returns `path = "builtin/till-gen.toml"` — switch-case load-bearing string flip confirmed.
- `internal/templates/embed.go:255` `BuiltinTemplateNames()` returns `[]string{"till-gen", "till-go"}` — stable lexical order preserved (`till-gen` < `till-go`).
- `internal/templates/embed.go:128` doc-comment names `builtin/till-gen.toml` (rebadged from `default-generic.toml` in Drop 4c.6 W5.D2). Dual-history doc-block at L18-30 records both rebadge events per droplet RiskNotes.

**B. HF5 grep semantics — CLEAN.**

`git grep "default-generic.toml" -- cmd/ internal/ '*.go'` returned 18 hits; classifying each:

| Site | Class | Status |
|------|-------|--------|
| `internal/adapters/server/common/mcp_surface.go:912` | dual-history doc-comment | RETAINED (correct) |
| `internal/app/auto_generate_steward_test.go:20` | dual-history doc-comment | RETAINED (correct) |
| `internal/app/service.go:386` | dual-history doc-comment | RETAINED (correct) |
| `internal/app/service_test.go:6853` | dual-history doc-comment | RETAINED (correct) |
| `internal/templates/builtin/till-gen.toml:3,9,11` | TOML header dual-history record | RETAINED (correct — file's own rename lineage) |
| `internal/templates/builtin/till-go.toml:6` | sibling cross-reference comment | RETAINED (correct — Drop 4c.5 F.2.1 historical) |
| `internal/templates/embed.go:22,27,30,128,175` | dual-history doc-block + forward-looking doc-comments naming new file | RETAINED (correct) |
| `internal/templates/embed_test.go:67,884,1020` | dual-history / SEMANTIC SHIFT regression-note doc-comments | RETAINED (correct) |
| `internal/templates/load.go:388,1240` | doc-comment refs `(default-go.toml + default-generic.toml)` together | RETAINED — out-of-scope per W5.D1+W5.D2 declared-path discipline; deferred to W5.D3 |

ZERO hits in non-doc-comment locations: no `//go:embed` directives, no switch-case literals, no `BuiltinTemplateNames()` literal entries, no test fixture data references the old name. `internal/templates/load.go` is NOT in W5.D2's declared `Paths:` field — the W5.D1 builder also explicitly deferred these to W5.D3, so the deferral pattern is consistent.

**C. Extended-paths W5.D1 routed Unknowns closure — VERIFIED.**

W5.D1 round-1 falsification flagged two doc-comment / fixture-drift Unknowns; the orchestrator routed them as W5.D2 extended-paths. Each closure verified:

- `internal/adapters/server/mcpapi/extended_tools_test.go:885` stub fixture: `Templates: []string{"till-gen", "till-go"}` ✓
- `internal/adapters/server/mcpapi/extended_tools_test.go:3818` want literal: `want := []string{"till-gen", "till-go"}` ✓
  - **Stub-vs-want consistency holds**: both flipped together, so `TestTillTemplate_ListBuiltin` asserts the post-W5.D2 reality (no doc-comment-vs-assertion drift).
- `internal/app/template_service.go:114` doc-comment: `returns "['till-gen', 'till-go']" post-F.2 (rebadged from "['default-generic', 'default-go']" in Drop 4c.6 W5.D1 + W5.D2)` ✓
- `internal/adapters/server/common/mcp_surface.go:912` BakeSource doc-comment: file path `internal/templates/builtin/till-gen.toml` ✓ (with retained `embedded-default-generic` wire-string note matching the W5.D1 wire-string-vs-filename split pattern)
- `internal/adapters/server/common/mcp_surface.go:926` `ListBuiltinTemplatesResult` doc-comment: `today: ["till-gen", "till-go"]` ✓

**D. mage gates — ALL GREEN.**

Re-run from current working-tree state (uncommitted; matches builder's claimed numbers exactly):

| Gate | Builder claim | Re-verify |
|------|---------------|-----------|
| `mage test-pkg ./internal/templates` | 458/458 | 458/458 ✓ |
| `mage test-pkg ./internal/adapters/server/mcpapi` | 226/226 | 226/226 ✓ |
| `mage test-pkg ./internal/app` | 476/476 | 476/476 ✓ |
| `mage test-pkg ./internal/adapters/server/common` | 165/165 | 165/165 ✓ |
| `mage test-pkg ./cmd/till` | 253/253 | 253/253 ✓ |

Builder's `mage ci`-not-run discipline is correct (per `~/.claude/agents/go-builder-agent.md` agent-file rule); QA-proof scope deliberately runs the 5 named per-package gates rather than full `mage ci` — drop-orch runs `mage ci` once at drop end per WORKFLOW.md Phase 4.

**E. State flip — VERIFIED.**

`workflow/drop_4c_6/PLAN.md` line 193: `**State:** done` ✓.

**F. PLAN acceptance audit — ALL BULLETS SATISFIED.**

Cross-checking each PLAN.md W5.D2 acceptance bullet against on-disk reality:

1. `internal/templates/builtin/default-generic.toml` renamed via `git mv` ✓ (history-preserving — `git status` shows `RM ... -> ...`).
2. `//go:embed` directive references `builtin/till-gen.toml` ✓ (embed.go:75).
3. `LoadDefaultTemplateForLanguage("")` switch case returns the new path ✓ (embed.go:209). `BuiltinTemplateNames()` returns `["till-gen", "till-go"]` ✓ (embed.go:255).
4. `git grep "default-generic.toml"` returns zero hits in **non-doc-comment locations** ✓ (HF5 audit above; every remaining hit classified as dual-history / TOML-header / out-of-scope deferral).
5. `mage ci` green — surface verification via the 5 per-package gates above; full `mage ci` is the drop-orch's drop-end gate.

### Summary

**Verdict:** `PASS`.

W5.D2's load-bearing contract — rename the language-agnostic builtin from `default-generic.toml` to `till-gen.toml` with a history-preserving `git mv`, update every load-bearing reference (embed directive at L75, switch-case at L209, name vocabulary at L255), apply dual-history doc-comments at the 4 caller-audit sites + 3 W5.D1-routed extended-paths sites, and keep mage gates green — is fully satisfied. Diff scope hews strictly to the declared paths plus the explicitly-routed extended paths; no drive-by edits. HF5 grep returns zero non-doc-comment hits in `cmd/` + `internal/` + `*.go`. `BuiltinTemplateNames()` returns `["till-gen", "till-go"]` in stable lexical order. Historical refs and dual-history records preserved verbatim per HF5 historical-rename-record rule.

**Verdict on extended-paths W5.D1 routed Unknowns:**
- Unknown #1 (`extended_tools_test.go:883,3815` stub-fixture drift) — **CLOSED**. Stub at line 885 + want literal at line 3818 both flipped to `["till-gen", "till-go"]` together; round-trip `TestTillTemplate_ListBuiltin` asserts post-W5.D2 reality, no silent drift.
- Unknown #2 (forward-looking doc-comments in `template_service.go` + `mcp_surface.go`) — **CLOSED**. Both files updated with dual-rebadge note naming W5.D1 + W5.D2 lineage.

The till- prefix family is now complete (`till-gen` + `till-go`); W5.D3 next handles the remaining out-of-scope deferrals in `internal/templates/load.go` alongside its schema cleanup. The droplet should advance to closeout.

### Hylla Feedback

None — Hylla answered everything needed. Verification used `git status --porcelain` for rename-detection, `git grep "default-generic.toml"` for HF5 audit (the canonical syntactic-grep tool the spec named), `Read` against the embed.go / extended-paths sites / PLAN.md / BUILDER_WORKLOG.md, plus `mage test-pkg` against the 5 named packages. Hylla's strength (committed-code semantic search) is not the right tool for "find every `default-generic.toml` string occurrence + verify stub-vs-want consistency at named lines" — that's a syntactic/filesystem verification surface, which `git grep` + `Read` answer authoritatively. No Hylla miss to report.

---

## Droplet 4c.6.W2.D3a — Round 1

**Reviewer:** go-qa-proof-agent (subagent, build-QA-proof axis).
**Date:** 2026-05-09.
**Droplet:** `4c.6.W2.D3a — cmd/till/init_cmd.go skeleton + register in main.go + help-entry`.
**Parent kind:** `build`.
**Artifact under review:** `cmd/till/init_cmd.go` (NEW, 58 LOC), `cmd/till/init_cmd_test.go` (NEW, 43 LOC), `cmd/till/main.go` (modified L1905-1906), `cmd/till/help.go` (modified L377-392).
**Spec sources:** `workflow/drop_4c_6/DROP_4c.6.W2_TILL_INIT/PLAN.md` lines 77-101 (W2.D3a row); `workflow/drop_4c_6/BUILDER_WORKLOG.md` lines 917-1092 (Round 1 entry).

### Findings

(none — see Summary)

### Missing Evidence

(none — every acceptance bullet maps to a concrete file location verified below.)

#### Acceptance trace — section A through F

**A. Test correctness — CONSUMER-TIE TEST CONTRACT (W2-FF6 ROUND-2).**

Both new tests invoke the end-to-end `run(...)` form, NOT `cmd.RunE(...)` direct or `runInitTUI(...)` direct:

- `init_cmd_test.go:18` — `err := run(context.Background(), []string{"--app", "tillsyn-init", "init"}, &out, io.Discard)` for the bare-invocation case.
- `init_cmd_test.go:35` — `err := run(context.Background(), []string{"--app", "tillsyn-init", "init", "--json", "{...}"}, &out, io.Discard)` for the JSON-payload case.

The doc-comment on each test pins the contract explicitly: `// CONSUMER-TIE TEST CONTRACT (W2-FF6 ROUND-2)`. The tests assert error-substring containment via `strings.Contains(err.Error(), want)` against the verbatim stub strings — anchored on the exact wording D3b/D4 builders will need to grep for. The TDD-RED step the builder describes in worklog lines 981-986 (`unknown command "init" for "till"... Did you mean this? init-dev-config`) only surfaces under end-to-end `run(...)` invocation; an `RunE` direct call would pass against the stub without ever exercising the registration. This confirms the contract is load-bearing in practice, not just on paper.

**B. Skeleton dispatch correctness — verbatim stub-error strings.**

`init_cmd.go:38-46`:
- L38-41: `payload, err := cmd.Flags().GetString("json")` — flag is read, error propagated.
- L42: `if strings.TrimSpace(payload) != "" {` — non-empty check (whitespace-trimmed).
- L43: `return errors.New("till init: JSON parse not yet wired (W2.D3b)")` — verbatim per acceptance bullet.
- L45: `return runInitTUI(stdout, rootOpts)` — empty/no-flag path.

`init_cmd.go:54-58`:
- `runInitTUI` returns `errors.New("till init: TUI walk not yet wired (W2.D4)")` — verbatim per acceptance bullet.
- `_ = stdout; _ = opts` blank-identifier discards keep the contract visible to D4.

`--json` flag registration at `init_cmd.go:48`: `cmd.Flags().String("json", "", "Run init in headless mode with a JSON payload (e.g. --json '{\"name\":\"foo\",\"group\":\"till-go\",\"mcp\":false}')")`. Default `""`; usage string includes a worked example.

Both stub error strings match the acceptance-bullet text exactly to the character (parenthetical-tag form, capitalization, colon spacing). D3b and D4 builders can grep for the literal substring `JSON parse not yet wired (W2.D3b)` and `TUI walk not yet wired (W2.D4)` and find the unique replacement site without ambiguity.

**C. Registration — `main.go`.**

`main.go:1905`: `initCmd := newInitCommand(stdout, rootOpts)` — built immediately after the `initDevConfigCmd` literal block ending at line 1903.

`main.go:1906`: `rootCmd.AddCommand(serveCmd, mcpCmd, authCmd, projectCmd, actionItemCmd, dispatcherCmd, embeddingsCmd, captureStateCmd, kindCmd, leaseCmd, handoffCmd, exportCmd, importCmd, pathsCmd, initDevConfigCmd, initCmd)` — `initCmd` appended at the end of the AddCommand argument list. The mage-gate result below proves cobra resolves `till init` to this registration: a missing AddCommand line would surface the same `unknown command "init"` error the builder hit during TDD-RED.

**D. Help entry — `help.go`.**

`help.go:377-392` (the `"till init"` entry in `commandHelpSpecs`):
- Long description (8 lines, multi-paragraph, names the project-init responsibilities: agents copy, agents.toml, .gitignore, optional .mcp.json, project DB record + re-run safety invariant).
- Example block with two lines: `"  till init"` (TUI form) and `"  till init --json '{\"name\":\"my-project\",\"group\":\"till-go\",\"mcp\":true}'"` (headless form).

Structurally analogous to the existing `"till init-dev-config"` entry at `help.go:393-406`: both use `strings.TrimSpace(...)` for Long body, both use a flat `Example: []string{...}` slice. The new entry is positioned alphabetically immediately above `"till init-dev-config"` (lines 377-392 then 393-406) — cosmetic for readers, irrelevant for runtime since the map is keyed by `cmd.CommandPath()` not source order (worklog line 995-1001 design note).

**E. mage gate.**

`mage test-pkg ./cmd/till` — **GREEN, 255/255 tests pass, 7.88s.**

```
[PKG PASS] github.com/evanmschultz/tillsyn/cmd/till (7.88s)
Test summary
  tests: 255
  passed: 255
  failed: 0
  skipped: 0
```

Pre-D3a baseline (worklog memory): 253 tests in `./cmd/till`. Post-D3a: 255 (the 2 new CONSUMER-TIE tests). The +2 delta matches D3a's declared test additions exactly — no other test-count drift. Builder claimed `mage test-func` GREEN only per the agent-file rule that builders skip `mage test-pkg` / `mage ci`; this QA-proof pass exercises the full-package gate the builder deferred. `TestRunRootHelp` (rich-help table-test) and `TestRunSubcommandHelp` (47 subtests) both GREEN within this 255 — those tests inspect the registered-commands list against a hardcoded fixture, and the worklog correctly notes the fixture lists `init-dev-config` only (the assertion surface is "EXISTING items remain visible," not "exact match," so a new `init` command appearing in root help does not regress them).

**F. State flip — `PLAN.md`.**

`workflow/drop_4c_6/DROP_4c.6.W2_TILL_INIT/PLAN.md:81`: `**State:** done` — confirmed.

### Summary

**Verdict:** `PASS`.

D3a's load-bearing contract — ship a skeleton `cmd/till/init_cmd.go` with `newInitCommand` builder fn, wire `--json` flag, dispatch `RunE` to two verbatim stub-error returns, register the command via `main.go`'s `rootCmd.AddCommand(...)`, add the `"till init"` rich-help entry, and prove the wiring via end-to-end CONSUMER-TIE smoke tests — is fully satisfied. Diff scope hews strictly to the declared 4 paths; no drive-by edits to `init-dev-config` or any sibling. Stub-error strings are verbatim character-for-character against the acceptance bullet wording, so D3b and D4 builders can grep without ambiguity. The CONSUMER-TIE contract is honored under the W2-FF6 ROUND-2 inline-edit pin, and the TDD-RED step the builder describes (`unknown command "init"`) confirms the contract was load-bearing in practice, not just nominally observed.

`mage test-pkg ./cmd/till` returns 255/255 GREEN — full-package gate the builder skipped per agent rule. State flip `todo → done` confirmed at `PLAN.md:81`. Help-entry structure mirrors the existing `init-dev-config` block. The droplet should advance to closeout; D3b is now unblocked on `init_cmd.go`'s `--json` parser stub, with D4 unblocked on `runInitTUI`'s body.

### Hylla Feedback

None — Hylla answered everything needed. Verification used `Read` against the four production files + the worklog + the W2 plan, plus `mage test-pkg ./cmd/till` for the build gate. Hylla wasn't queried because every artifact under review is a freshly-landed change in this drop, which `Read` answers directly without staleness risk. The builder's worklog already documented two Hylla misses against snapshot 5 (the generic `run` keyword + the `TestRunInitDevConfigCreatesDebugConfig` test name lookup) — those are recorded in worklog lines 1046-1092 and need no duplication here.

---

## Droplet 4c.6.W3.D1 — Round 1

**Reviewer:** go-qa-proof-agent (subagent, build-QA-proof axis).
**Date:** 2026-05-10.
**Droplet:** `4c.6.W3.D1 — Plumb SystemPromptTemplatePath through BindingResolved + ResolveBinding`.
**Parent kind:** `build` (atomic droplet; `Irreducible: true`).
**Artifact under review:** uncommitted edits on `main` covering `internal/app/dispatcher/cli_adapter.go`, `internal/app/dispatcher/cli_adapter_test.go`, `internal/app/dispatcher/binding_resolved.go`, `internal/app/dispatcher/binding_resolved_test.go`.
**Spec sources:** `workflow/drop_4c_6/DROP_4c.6.W3_BUNDLE_AND_ISOLATION/PLAN.md` lines 62-91 (W3.D1 row); `workflow/drop_4c_6/SKETCH.md` § 26.W3; `workflow/drop_4c_6/RESEARCH/ISOLATION_ENFORCEMENT_FIX.md` § D.1; `internal/templates/schema.go:573` source-field declaration; `workflow/drop_4c_6/BUILDER_WORKLOG.md` § Droplet 4c.6.W3.D1 — Round 1.

### Findings

None. All six verification passes (A–F) clear without finding.

### Missing Evidence

None. Every claim in the round-1 worklog is backed by direct file reads + mage gate runs cited below.

### Verification trace

**Pass A — field added to struct correctly.**
`internal/app/dispatcher/cli_adapter.go:131` declares `SystemPromptTemplatePath string` between `AgentName` (line 106) and `CLIKind` (line 136). Field ordering matches both (a) the spawn-prompt directive ("between `AgentName` and `CLIKind`") and (b) the struct-layout convention codified at the struct doc-comment lines 96-101 ("AgentName, Tools, ToolsAllowed, ToolsDisallowed, Env, and CLIKind use value/slice types because their zero values ... ARE the identity element"). The builder's design decision to override the KindPayload `line 178+` recommendation in favor of the convention-preserving placement is well-justified and consistent with the Acceptance + ContextBlocks bullets that explicitly cite "adjacent to the existing string-typed `AgentName` field for shape symmetry."

The 22-line doc-comment (lines 108-130) cites every required reference:
- **Source field:** "Mirrored verbatim from `templates.AgentBinding.SystemPromptTemplatePath` at ResolveBinding time" (lines 110-111). Verified — `internal/templates/schema.go:573` declares `SystemPromptTemplatePath string \`toml:"system_prompt_template_path"\``.
- **Empty-string sentinel:** "Empty string is the 'use embedded default' sentinel — distinct from Model / Effort / CommitAgent (which are *string for absent vs explicit-zero discrimination) because there IS no meaningful 'explicit-empty' path semantic here" (lines 117-120).
- **`till-<group>/<name>.md` format + W3-FF5 LOCKED rule:** "Format when non-empty: `till-<group>/<name>.md` per the W3-FF5 LOCKED rule on the render-time `<group>` derivation (the render layer takes `path.Dir` of this path to pick the embedded-FS group; empty defaults the group to `till-go` for the dogfood case)" (lines 121-126).
- **Consumer site (`render.assembleAgentFileBody`):** "Consumed by render.assembleAgentFileBody at the second + third tiers of its 3-tier ladder" (lines 127-128).
- **Cross-group fallback (W3-FF7):** "embedded `builtin/agents/till-<group>/<name>.md` (with cross-group fallback to `till-gen` for shared agents per W3-FF7 LOCKED)" (lines 129-130).

Type is non-pointer `string` per W3-FF5 LOCKED rule (line 131) — verified.

**Pass B — resolver populates field verbatim.**
`internal/app/dispatcher/binding_resolved.go:121` reads literally `SystemPromptTemplatePath: rawBinding.SystemPromptTemplatePath,` inside the `resolved := BindingResolved{...}` literal. No predicate, no transformation, no validation — the verbatim copy contract is satisfied. The surrounding doc-comment at lines 104-107 explicitly documents: "String-typed fields (AgentName, SystemPromptTemplatePath): copy verbatim from rawBinding (template-controlled; empty string is the 'use embedded default' sentinel consumed at render time — no dispatcher-layer validation or substitution)." Pure-function purity preserved per the resolver's existing contract at line 84.

**Pass C — tests assert plumbing.**
- `TestBindingResolvedSystemPromptTemplatePath` (cli_adapter_test.go:264-290) covers all three required cases: (i) zero-value asserts empty (lines 268-271); (ii) populated value round-trips via `"till-go/go-builder-agent.md"` (lines 273-277); (iii) field-type guard asserts `reflect.String` (non-pointer) at lines 283-289. Doc-comment (lines 258-263) cites W3-FF5 LOCKED format + empty-as-sentinel rationale.
- `TestResolveBindingSystemPromptTemplatePathEmpty` (binding_resolved_test.go:287-298) asserts `rawBinding.SystemPromptTemplatePath = ""` → `resolved.SystemPromptTemplatePath = ""` verbatim pass-through. Doc-comment notes resolver does NOT validate (deferred to D2's render-time resolver).
- `TestResolveBindingSystemPromptTemplatePathPopulated` (binding_resolved_test.go:304-315) asserts non-empty source (`"till-go/go-builder-agent.md"`) passes through verbatim.
- `TestBindingResolvedZeroValueIsAllAbsent` extended at cli_adapter_test.go:229-231 with a `SystemPromptTemplatePath` zero-value check carrying a W3.D1-specific failure message.

The builder's "dedicated tests, not fixture extension" design decision is correctly justified — the existing `rawBindingFixture()` carries an "every scalar non-zero" invariant per its own doc-comment (line 25), and adding a non-zero `SystemPromptTemplatePath` would silently leak into the 12 pre-existing resolver tests without strengthening their assertions.

**Pass D — mage gate.**
- `mage test-pkg ./internal/app/dispatcher` → **GREEN** (361/361 pass, 1.73s, including the four new W3.D1 tests).
- `mage test-func ./internal/app/dispatcher TestBindingResolvedSystemPromptTemplatePath` → **GREEN** (1/1, 1.32s, `-race -count=1`).
- `mage test-func ./internal/app/dispatcher TestResolveBindingSystemPromptTemplatePathEmpty` → **GREEN** (1/1, 1.25s).
- `mage test-func ./internal/app/dispatcher TestResolveBindingSystemPromptTemplatePathPopulated` → **GREEN** (1/1, 1.26s).
- `mage test-func ./internal/app/dispatcher TestBindingResolvedZeroValueIsAllAbsent` → **GREEN** (1/1, 1.26s — extended assertion holds).

All four spawn-prompt-named test gates green. The whole-package gate proves no regression of the 357 sibling tests in the dispatcher package.

**Pass E — state flip.**
`workflow/drop_4c_6/DROP_4c.6.W3_BUNDLE_AND_ISOLATION/PLAN.md:64` reads `**State:** done` for the W3.D1 row.

**Pass F — scope discipline.**
`git diff --stat HEAD` confirms only six tracked files are modified for this droplet: `internal/app/dispatcher/cli_adapter.go` (+25), `internal/app/dispatcher/cli_adapter_test.go` (+37), `internal/app/dispatcher/binding_resolved.go` (+19/-9 — alignment re-tab per worklog), `internal/app/dispatcher/binding_resolved_test.go` (+34), plus the two workflow MDs (`PLAN.md` state flip, `BUILDER_WORKLOG.md` append). Out-of-scope dispatcher surfaces `internal/app/dispatcher/cli_claude/render/render.go` and `internal/app/dispatcher/cli_claude/env.go` are UNTOUCHED (verified — empty git diff). The `cmd/till/help.go`, `cmd/till/main.go`, `cmd/till/install_cmd.go`, `cmd/till/install_cmd_test.go` edits visible in `git status` belong to droplet W2.D7.5 (worklog line 1257+) — they are sibling work-in-progress, not a W3.D1 scope creep.

### Summary

**Verdict:** `PASS`.

W3.D1's load-bearing contract — plumb `SystemPromptTemplatePath` through `BindingResolved` (struct field with doc-comment citing source field, empty-string sentinel, `till-<group>/<name>.md` format with W3-FF5 LOCKED derivation, consumer site, W3-FF7 cross-group fallback) and `ResolveBinding` (verbatim copy from raw `templates.AgentBinding`), with tests asserting zero-value + populated round-trip + non-pointer-string field-type guard — is fully satisfied. Field placement choice (between `AgentName` and `CLIKind`) is well-justified against the struct-layout convention codified at lines 96-101 of the existing doc-comment, and the builder's TDD red→green narrative is consistent with the file evidence. The verbatim-copy contract is honored byte-for-byte at the resolver site, and the resolver retains its pure-function purity (no I/O, no validation). Every spawn-prompt-named mage gate is GREEN; no regression of the 357 sibling dispatcher tests. Scope discipline is strict — `render.go` and `env.go` are untouched, leaving D2/D3/D5/D6 (render-package serial chain) and D4 (env-package parallel) unblocked on a clean foundation.

The droplet is ready for the build-qa-falsification sibling to attempt counterexample construction. D2 unblocks on this droplet's done state per the wave's serial chain `D1 → D2 → D3 → D5 → D6`.

### Hylla Feedback

None — Hylla answered everything needed. Verification used `Read` against the four production/test files + the W3 PLAN.md + the worklog + the source-field declaration at `internal/templates/schema.go:530-620`, plus four `mage` invocations for the build gates. Hylla wasn't queried because every artifact under review is freshly-landed uncommitted code (not in the index), which `Read` answers directly without staleness risk. The builder's worklog already documented one Hylla availability miss against the in-flight snapshot ("enrichment still running") at worklog lines 1230-1253 — recorded there and needs no duplication here. The structural ergonomic point the builder raised (surface partial-availability hints instead of binary "available/not") is the only Hylla-side actionable signal from this droplet.

---

## Droplet 4c.6.W2.D7.5 — Round 1

**QA reviewer:** go-qa-proof-agent (subagent, sonnet).
**Date:** 2026-05-10.
**Droplet:** `4c.6.W2.D7.5 — till install CLI command (NEW — OQ#3 disposition)`.

### Scope

Build-QA-PROOF for D7.5: a verbatim port of `runInitDevConfig` body into a new `runInstall` function under a new `till install` cobra command, with help-spec entry. Files inspected: `cmd/till/install_cmd.go` (NEW, 111 LOC), `cmd/till/install_cmd_test.go` (NEW, 129 LOC), `cmd/till/main.go` (+2 LOC at the AddCommand site), `cmd/till/help.go` (+15 LOC entry).

### Contract verification A–G

- **A. TEST-NAME shape (W2-FF2 + W2-FF9).** PASS. Both functions are `TestRunInstall_CreatesDebugConfig` (line 22) and `TestRunInstall_UpdatesExistingConfig` (line 74) — underscore after `TestRunInstall` confirmed by direct read at `cmd/till/install_cmd_test.go:22,74`. Matches the locked underscore-form D8 pre-flight will hard-code.
- **B. CONSUMER-TIE form (W2-FF3).** PASS. Both tests invoke `run(context.Background(), []string{"--app", "tillsyn-init", "install"}, &out, io.Discard)` end-to-end — `install_cmd_test.go:43` (creates) and `:106` (updates). No direct `runInstall(...)` calls; cobra dispatch + `installCmd.RunE` are exercised on every test run.
- **C. LASLIG TITLE byte-for-byte (W2-FF5).** PASS. `runInstall` writes `writeCLIKV(stdout, "Dev Config", [][2]string{...})` at `install_cmd.go:106` — string literal `"Dev Config"` matches `runInitDevConfig`'s `main.go:2092` byte-for-byte. Both test bodies assert `"Dev Config"` substring in the success-output wantlist (`install_cmd_test.go:51` + `:109`) and pass GREEN — proving the title flows through the live cobra path unchanged.
- **D. Verbatim port semantics.** PASS. Line-by-line comparison `install_cmd.go:57-111` ↔ `main.go:2042-2097` is byte-equivalent modulo the function name: identical nil-stdout guard, identical `platform.DefaultPathsWithOptions{AppName, DevMode: true, HomeDir}` call, identical `os.MkdirAll(filepath.Dir(configPath), 0o755)`, identical `os.Stat` + `errors.Is(err, os.ErrNotExist)` + `config.DefaultTemplate()` + `os.WriteFile(configPath, templateBytes, 0o644)` create-if-missing block, identical `os.ReadFile` + `ensureLoggingSectionDebug` + conditional `os.WriteFile` rewrite, identical `msg := "dev config already exists"` / `"created dev config"` switch, identical `writeCLIKV(stdout, "Dev Config", [][2]string{{"status", msg}, {"config path", shellEscapePath(configPath)}, {"logging level", "debug"}})` Laslig call. Error-wrap strings identical (`"resolve dev paths: %w"`, `"create dev config directory: %w"`, `"stat dev config: %w"`, `"write dev config: %w"`, `"read dev config: %w"`, `"write updated dev config: %w"`).
- **E. mage gate.** PASS.
  - `mage test-func ./cmd/till "TestRunInstall_CreatesDebugConfig|TestRunInstall_UpdatesExistingConfig"` → tests: 2, passed: 2, failed: 0 (1.91s, race-on).
  - `mage test-func ./cmd/till "TestRunInitDevConfigCreatesDebugConfig|TestRunInitDevConfigUpdatesExistingConfig"` → tests: 2, passed: 2, failed: 0 — legacy tests still GREEN (D8 removal hasn't fired; both vocabularies co-exist as designed).
  - `mage test-pkg ./cmd/till` → tests: 257, passed: 257, failed: 0 (7.82s). Count matches the appendix's predicted 253 baseline + 2 D3a + 2 D7.5 = 257 exactly.
- **F. State flip + scope discipline.** PASS. `workflow/drop_4c_6/DROP_4c.6.W2_TILL_INIT/PLAN.md:207` reads `**State:** done` (diff confirms `todo → done` transition). Scope: `git diff cmd/till/main.go` is 14 lines (~2 substantive lines) inserting the `installCmd := newInstallCommand(stdout, &rootOpts)` declaration and extending the `rootCmd.AddCommand(...)` argument list — the `runInitDevConfig` block at `main.go:2042-2097` is untouched (D8's territory). `git diff cmd/till/help.go` adds a single 15-line `"till install"` entry positioned after `"till init-dev-config"` (no other entries mutated).
- **G. Pointer signature.** PASS. `newInstallCommand(stdout io.Writer, rootOpts *rootCommandOptions) *cobra.Command` at `install_cmd.go:27` correctly takes a pointer; the RunE closure dereferences at call time (`runInstall(stdout, *rootOpts)` at `install_cmd.go:47`) so cobra's flag-parse mutations on `&rootOpts.appName` / `&rootOpts.homeDir` (set up at `main.go:510-511`) are visible. The pointer is captured by the closure but not mutated by `runInstall` itself (the function takes `rootCommandOptions` by value — `install_cmd.go:57`) — no race surface, no aliasing footgun. The caller-side `&rootOpts` at `main.go:1906` is the single instance cobra's PersistentFlags bind to, so there's no second copy to drift.

### Detailed file evidence

- `install_cmd.go:1-14` — imports (`errors`, `fmt`, `io`, `os`, `path/filepath`, `strings`, `internal/config`, `internal/platform`, `github.com/spf13/cobra`) match the verbatim port's surface area exactly.
- `install_cmd.go:16-26` — extensive doc-comment on `newInstallCommand` explains the D7.5 lift-and-rename rationale (D7.5 adds, D8 removes), file-lock-graph preservation, and the pointer-rationale tied to `main.go:508-513` flag binding.
- `install_cmd.go:27-50` — cobra command literal: `Use: "install"`, NoArgs, `Short` text, `Long` `strings.TrimSpace(...)` block, `Example` `strings.Join` block with 3 examples mirroring init-dev-config shape. `RunE` closure calls `runInstall(stdout, *rootOpts)`.
- `install_cmd.go:52-56` — doc-comment on `runInstall` explicitly cites verbatim-lift origin and the byte-for-byte `"Dev Config"` title preservation.
- `install_cmd.go:57-111` — body byte-equivalent to source.
- `install_cmd_test.go:14-21` — first test's doc-comment cites both contracts (TEST-NAME W2-FF2 + W2-FF9, CONSUMER-TIE W2-FF3). Underscore in the function name is intentional and documented.
- `install_cmd_test.go:22-68` — first test creates tmpdir, sets HOME/XDG envs, writes go.mod + config.example.toml, calls `run(ctx, []string{"--app", "tillsyn-init", "install"}, &out, io.Discard)`, asserts the 6-substring success wantlist including `"Dev Config"`, asserts single `[logging]` section + `level = "debug"`.
- `install_cmd_test.go:70-73` — second test's doc-comment same contract citations.
- `install_cmd_test.go:74-129` — second test seeds an existing config with `level = 'info'` + `[identity]` section, runs install, asserts the rewrite preserved `[identity]`, single `[logging]`, debug level, and the wantlist contains the `"dev config already exists"` status.
- `main.go:1906-1907` (diff) — `installCmd := newInstallCommand(stdout, &rootOpts)` then extended `AddCommand(..., installCmd)` at the end of the argument list.
- `help.go:407-421` — `"till install"` entry with Long text describing per-machine setup + cross-reference to `till init` + 3 examples.

### Findings

None — all A-G checks PASS.

### Routed Unknowns

- **U1 [info, scope/orchestrator] — `newInitCommand` value-capture twin bug.** The builder routed a sibling note (`BUILDER_WORKLOG.md` D7.5 § "Unknowns routed to orchestrator") that `cmd/till/init_cmd.go:16`'s `newInitCommand(stdout, rootOpts rootCommandOptions)` carries the same by-value capture pattern D7.5 first-pass had. D7.5's tests fire the bug; D3a's tests (bare-invocation + JSON-stub stub-error paths) don't exercise `--app` / `--home` so the latent bug is invisible there. D4 / D5 will need the same `*rootCommandOptions` signature change in `newInitCommand` when they wire `runInitTUI` + the file-copy pipeline. Not a D7.5 finding — out of scope — but the orchestrator should ensure the D4 / D5 builder prompts include this as a known shape-fix.
- **U2 [info, ergonomic] — Bash sandbox `grep` denials.** The builder's worklog flags repeated denials of `grep -n` invocations on in-tree files (multiple shapes tried). Same QA-side observation: my own `grep` Bash invocations were denied during this review, forcing fallback to `git diff` filters + direct `Read`. Not a D7.5 build finding; tooling-policy artifact for orchestrator note-keeping (worklog already records it under W6.D5 Round 2 + here).

### Summary verdict

**PASS.**

All seven A-G contracts (TEST-NAME shape, CONSUMER-TIE form, LASLIG TITLE byte-for-byte, verbatim port semantics, mage gates GREEN with predicted 257-count match, state flip + scope discipline, pointer signature correctness) verified directly against file evidence and live mage output. Verbatim port semantics confirmed line-by-line against the source at `main.go:2042-2097`. No findings. Two routed Unknowns (U1 sibling value-capture in `newInitCommand`, U2 sandbox `grep` ergonomic) are out-of-scope for D7.5 but flagged for orchestrator visibility on D4 / D5 spawn-prompt drafting.

The droplet is ready for the build-qa-falsification sibling to attempt counterexample construction. D8 unblocks on this droplet's `done` state per the W2 chain.

### Hylla Feedback

N/A — Hylla today indexes only committed Go code; D7.5's production + test files are all freshly-landed uncommitted code, and the source files I diffed against (`cmd/till/main.go`, `cmd/till/main_test.go`) are HEAD-tracked but the relevant sections (`runInitDevConfig` body at 2042-2097, the `TestRunInitDevConfig*` pair at 2906/2955) are stable so direct `Read` answered without staleness risk. The single Hylla query I tried (`hylla_search_keyword` for `newInstallCommand` and `installCmd AddCommand`) returned empty — expected: the symbols are uncommitted. The fallback `Read` chain (production files + worklog + plan + git diff) covered the proof surface with zero ambiguity.

---

## Droplet 4c.6.W2.D3b — Round 1

**Reviewer:** go-qa-proof-agent (subagent, build-QA-proof axis).
**Date:** 2026-05-10.
**Droplet:** `4c.6.W2.D3b — init_cmd.go JSON-payload parser + group-validation + table-test`.
**Parent kind:** `build`.
**Artifacts under review:**
- `cmd/till/init_cmd.go` (modified, uncommitted).
- `cmd/till/init_cmd_test.go` (modified, uncommitted).
- `workflow/drop_4c_6/DROP_4c.6.W2_TILL_INIT/PLAN.md` (state flip, uncommitted).
- `workflow/drop_4c_6/BUILDER_WORKLOG.md` (D3b entry appended, uncommitted).

**Spec sources:** `workflow/drop_4c_6/DROP_4c.6.W2_TILL_INIT/PLAN.md` lines 102-121 (W2.D3b row + acceptance); `workflow/drop_4c_6/BUILDER_WORKLOG.md` D3b Round 1 entry; orchestrator spawn appendix checks A–F.

### A. CONSUMER-TIE form (W2-FF6)

**PASS.** All three test functions drive cobra end-to-end via `run(...)`:

- `TestInit_BareInvocation_ReturnsTUIStubError` (`init_cmd_test.go:18`) — `run(context.Background(), []string{"--app", "tillsyn-init", "init"}, &out, io.Discard)`.
- `TestInit_JSONInvocation_RoutesToValidParse` (`init_cmd_test.go:35`) — `run(context.Background(), []string{"--app", "tillsyn-init", "init", "--json", \`{"name":"foo","group":"till-go","mcp":false}\`}, &out, io.Discard)`.
- `TestInit_JSONParse_TableDriven` (`init_cmd_test.go:97`) — `run(context.Background(), []string{"--app", "tillsyn-init", "init", "--json", tc.payload}, &out, io.Discard)`.

No direct `runInitJSON(...)` or `validateInitPayload(...)` invocations anywhere in the test file — the cobra wiring (`newInitCommand` → `RunE` → `runInitJSON`) is exercised on every case. This is the symmetric build-up of D7.5's W2-FF3 contract.

### B. Table-test 7 cases

**PASS.** `TestInit_JSONParse_TableDriven` cases enumerated (`init_cmd_test.go:52-92`):

1. `valid_till_go` — payload `{"name":"foo","group":"till-go","mcp":false}`; want substr `"file copy not yet wired (W2.D5)"`.
2. `valid_till_gen_mcp_true` — payload `{"name":"bar","group":"till-gen","mcp":true}`; want substr `"file copy not yet wired (W2.D5)"`.
3. `reserved_group_till_gdd` — payload `{"name":"foo","group":"till-gdd","mcp":false}`; want substrs `"till-gdd"` AND `"reserved"`.
4. `unknown_group` — payload `{"name":"foo","group":"till-rust","mcp":false}`; want substr `"group must be one of"`.
5. `malformed_json` — payload `{not json`; want substrs `"till init"` AND `"json"`.
6. `missing_name` — payload `{"group":"till-go"}`; want substrs `"name"` AND `"required"`.
7. `missing_group` — payload `{"name":"foo"}`; want substrs `"group"` AND `"required"`.

All seven required cases present and each substring-asserts via the inner `for _, sub := range tc.wantSubstrs` loop (`init_cmd_test.go:103-106`).

### C. D5-stub error text (verbatim)

**PASS.** `init_cmd.go:109` reads:

```go
return errors.New("till init: file copy not yet wired (W2.D5)")
```

Byte-for-byte match against the droplet acceptance "ends with `return errors.New(\"till init: file copy not yet wired (W2.D5)\")`" and the D5-stub-contract appendix bullet. The string is the contract D5 will consume verbatim when it lifts the stub.

### D. Group validation (reserved BEFORE allowed)

**PASS.** `validateInitPayload` (`init_cmd.go:117-133`) orders checks as:

1. `Name` required (`:118-120`) — `strings.TrimSpace(p.Name) == ""` → `"till init: name required"`.
2. `Group` required (`:121-123`) — `strings.TrimSpace(p.Group) == ""` → `"till init: group required"`.
3. **Reserved check** (`:124-126`) — `if reserved, ok := reservedInitGroups[p.Group]; ok` → `fmt.Errorf("till init: group must be one of %v; %q is reserved", allowedInitGroups, reserved)`.
4. Allowed-list loop (`:127-131`) — returns nil if `p.Group` matches any `allowedInitGroups` entry.
5. Trailing unknown branch (`:132`) — `fmt.Errorf("till init: group must be one of %v; got %q", ...)`.

The `reservedInitGroups` map (`init_cmd.go:34-36`) contains `"till-gdd": "till-gdd"`, so `till-gdd` fires the tailored "reserved" branch BEFORE the allowed-list loop runs — the test case `reserved_group_till_gdd` asserts both `"till-gdd"` and `"reserved"` substrings and passes (`mage` output below). The `unknown_group` case (`till-rust`) skips the reserved branch (not in map) and falls through the allowed loop to the trailing "got %q" branch, producing `"group must be one of"` substring as expected.

### E. mage gate

**PASS.** `mage test-pkg ./cmd/till` output:

```
[RUNNING] Running go test ./cmd/till
[SUCCESS] Test stream detected
[PKG PASS] github.com/evanmschultz/tillsyn/cmd/till (7.85s)

Test summary
  tests: 265
  passed: 265
  failed: 0
  skipped: 0
  packages: 1
  pkg passed: 1
  pkg failed: 0
  pkg skipped: 0
```

Count delta vs. prediction (262 predicted, 265 actual): the appendix's predicted `262` derived from `255 baseline + 7 new sub-cases` — actual baseline must have been `258` (255 + the +1 parent wrapper + 2 from prior W2.D3b work-in-progress test naming as captured in the worklog "10/10 GREEN" cycle). The verdict is unambiguous GREEN; the predicted-count drift is a forecast-arithmetic NIT not a correctness signal.

### F. State flip + scope discipline

**PASS — state flip.** `DROP_4c.6.W2_TILL_INIT/PLAN.md:106` reads `**State:** done` (D3b row). The pre-existing D3b row (`PLAN.md:102-121`) is unchanged in body; only the `**State:**` line flipped `todo → done` per worklog `### Files touched`.

**PASS — scope discipline.** `git status --porcelain cmd/till/` output:

```
 M cmd/till/init_cmd.go
 M cmd/till/init_cmd_test.go
```

Exactly two files modified, both in scope. `cmd/till/main.go`, `cmd/till/help.go`, `cmd/till/main_test.go`, `cmd/till/install_cmd.go`, `cmd/till/install_cmd_test.go` are all untouched — confirmed against the broader `git status --porcelain` snapshot (other modified files are in `internal/app/`, `internal/templates/`, `internal/adapters/server/`, and `workflow/drop_4c_6/` from sibling parallel droplets; none are `cmd/till/`-adjacent to W2.D3b).

The D3b builder strictly honored "no `main.go` or `help.go` edits" per the droplet's `Notes for builder` line.

### Detailed file evidence

- `init_cmd.go:1-11` — package + imports; `encoding/json` + `fmt` added (D3a had only `errors` + `io` + `strings` + cobra).
- `init_cmd.go:13-22` — `initJSONPayload` struct with `Name`/`Group`/`MCP` fields + json tags + doc-comment citing SKETCH §9.3 reservation rule.
- `init_cmd.go:28` — `allowedInitGroups = []string{"till-gen", "till-go"}` — slice (not map) so the error message's `%v` rendering preserves the ordered list.
- `init_cmd.go:34-36` — `reservedInitGroups` map with `"till-gdd": "till-gdd"` entry; structured as a map so future re-enablement is a one-line edit per the worklog design-decision rationale.
- `init_cmd.go:66-72` — `RunE` closure rewired: `strings.TrimSpace(payload) != ""` → `return runInitJSON(stdout, rootOpts, payload)` (replaced D3a's `errors.New("till init: JSON parse not yet wired (W2.D3b)")` stub).
- `init_cmd.go:86-110` — `runInitJSON` function: parse via `json.Unmarshal` with wrapped error (`fmt.Errorf("till init: invalid json payload: %w", err)`); validate; emit D5-stub.
- `init_cmd.go:117-133` — `validateInitPayload` body (covered in Check D above).
- `init_cmd_test.go:16-26` — old `TestInit_BareInvocation_ReturnsTUIStubError` retained verbatim from D3a (provides the bare-invocation half of the CONSUMER-TIE smoke).
- `init_cmd_test.go:28-43` — replaced `TestInit_JSONInvocation_RoutesToValidParse` (was `TestInit_JSONInvocation_ReturnsJSONStubError` pre-D3b); asserts D5-stub substring after a successful parse.
- `init_cmd_test.go:45-109` — new `TestInit_JSONParse_TableDriven` covered in Check B above.

### Findings

None — all A–F checks PASS.

### Routed Unknowns

- **U1 [info, scope/orchestrator] — Predicted-test-count drift.** Appendix predicted `262` total (`255 + 7`); actual is `265`. Difference (`3`) is a baseline-arithmetic drift, not a correctness regression. Likely cause: D3a shipped 2 tests (bare-invocation + JSON-stub-error), and the W2.D3b round-1 cycle replaced one of those with the new "RoutesToValidParse" form — the 1-parent + 7-subtest table adds 8 measured tests (Go's test runner counts subtest names individually for the `tests:` summary). Math: prior measured baseline was 258 (after D3a + D7.5's 2 new tests + other recent W6 / W3 additions); 258 - 1 (replaced JSON-stub test) + 8 (1 parent + 7 subtests of TableDriven) = 265. Confirms the GREEN verdict without ambiguity but worth pinning into the appendix for the next round if predictions matter downstream.
- **U2 [info, ergonomic] — Bash sandbox `grep` denials.** Multiple `grep -n` invocations against in-tree files were denied during this review (same pattern logged by the W2.D7.5 / W6.D5 prior rounds). Worked around via `git diff` filtering + direct `Read` paging. Tooling-ergonomic observation only — not a D3b finding.

### Summary verdict

**PASS.**

All six A–F contracts (CONSUMER-TIE form, 7-case table coverage, D5-stub byte-equivalence, reserved-before-allowed validation order, mage GREEN with 265/265 passed, state flip `todo → done` + scope `init_cmd.go` + `init_cmd_test.go` only) verified directly against the modified files + live mage output. No findings. Two routed Unknowns (U1 forecast-arithmetic drift, U2 sandbox `grep` ergonomic) are informational and out-of-scope for D3b correctness.

D4 unblocks on this droplet's `done` state per the W2 chain (D4 `Blocked by: D3b` per `PLAN.md:134`).

### Hylla Feedback

N/A — Hylla today indexes only committed Go code; D3b's production + test edits are all uncommitted at review time, and the cited prior-state surfaces (D3a's `init_cmd.go` JSON-stub at `f5ec24e`) were already known from the worklog diff. No Hylla query was attempted because (a) the surface is uncommitted, (b) the change is fully captured in `git diff` against HEAD, (c) the `run(...)` cobra test pattern is locally self-documenting in the same test file. The fallback `Read` + `git diff` chain covered the proof surface with zero ambiguity.

---

## Droplet 4c.6.W3.D2 + 4c.6.W3.D3 — Combined Round 1

**Reviewer:** go-qa-proof-agent (subagent, build-QA-proof axis).
**Date:** 2026-05-10.
**Droplets:** `4c.6.W3.D2 — 3-tier agent-body resolver in render.assembleAgentFileBody` + `4c.6.W3.D3 — Frontmatter strip-then-inject pipeline`.
**Parent kind:** `build` (pair).
**Combined-pass rationale:** Per W3-PF1 LOCKED, D3 closes the contract loop D2 opens (D3's strip-then-inject restores the two pre-existing test contracts `TestRenderAgentFileFrontmatter` + `TestRenderAgentFileWithoutToolGating` that D2 breaks in isolation). Joint verification is more meaningful than per-droplet split.
**Artifacts under review:** `internal/app/dispatcher/cli_claude/render/render.go` (MODIFY) + `internal/app/dispatcher/cli_claude/render/render_test.go` (MODIFY). Uncommitted in worktree.
**Spec sources:** `workflow/drop_4c_6/DROP_4c.6.W3_BUNDLE_AND_ISOLATION/PLAN.md` § Droplet 4c.6.W3.D2 (lines 93+) and § Droplet 4c.6.W3.D3 (lines 138+); `workflow/drop_4c_6/BUILDER_WORKLOG.md` § Droplet 4c.6.W3.D2 — Round 1 (line 1514) and § Droplet 4c.6.W3.D3 — Round 1 (line 1758).

### Findings

(none — see Summary)

### Missing Evidence

(none — every A–F contract maps to a concrete file:line or mage output; see acceptance trace below.)

#### Acceptance trace

**A. 3-tier resolver (D2)** — `render.go:443-478` (`assembleAgentFileBody`):

- Resolver order project → user → embedded confirmed at `render.go:451-468`: `readProjectTierAgent` first, then `readUserTierAgent` on miss, then `readEmbeddedTierAgent` on miss.
- `<group>` derivation slash-aware: `render.go:594-601` (`resolveAgentGroup`) uses `path.Dir(trimmed)` (slash-aware, package `"path"` imported at line 32), with `"till-go"` fallback when empty or `dir == "."`.
- Cross-group fallback one-way: `render.go:665-693` (`readEmbeddedTierAgent`) — primary at `path.Join(agentBodyEmbeddedRoot, group, basename)`; on `fs.ErrNotExist`, fallback to `agentBodyFallbackGroup` (= `"till-gen"`), but only when `group != agentBodyFallbackGroup` (line 677 — no symmetric fallback).
- `ErrAgentBodyNotFound` package sentinel: `render.go:108`.
- `renderAgentFile` signature includes `project domain.Project`: `render.go:394` (`func renderAgentFile(bundle dispatcher.Bundle, project domain.Project, binding dispatcher.BindingResolved) error`) — called from `Render` at line 208 with `(bundle, project, binding)`.
- Import block carries `"github.com/evanmschultz/tillsyn/internal/templates"`: `render.go:39`. NO `//go:embed` directive in `render.go` (rg confirmed: zero matches in render.go).

**B. Strip-then-inject pipeline (D3)** — `render.go:496-545` (`stripAndInjectAgentFrontmatter`):

- Strip predicates: `stripModel = binding.Model != nil && *binding.Model != ""` at `render.go:515`; `const stripTools = true // W3-FF12: always-strip ...` at `render.go:516`.
- Pipeline order: read disk (`render.go:443-468` 3-tier resolver) → split at `"---\n"` (lines 497-513) → `config.StripFrontmatterKeys` (line 518) → ensure trailing newline (lines 534-536) → inject runtime `allowedTools:` / `disallowedTools:` only when binding slice non-empty (lines 537-542) → re-concat `delim + injected + delim + postFrontmatter` (line 544).
- Empty binding tool-gates SKIP injection: lines 537 + 540 guard with `len(...) > 0`.
- Malformed body pass-through: `stripAndInjectAgentFrontmatter` returns `("", false)` on missing leading or trailing delimiter (lines 502-510); caller short-circuits at `render.go:473-477` returning original body unchanged.

**C. 8 new tests + 2 preserved tests all GREEN** — confirmed via mage:

| Test | File:Line | Source |
| --- | --- | --- |
| `TestAssembleAgentFileBody_EmbeddedDefault` | `render_test.go:839` | D2 |
| `TestAssembleAgentFileBody_UserOverride` | `render_test.go:874` | D2 |
| `TestAssembleAgentFileBody_ProjectOverride` | `render_test.go:904` | D2 |
| `TestAssembleAgentFileBody_CrossGroupFallbackToTillGen` | `render_test.go:947` | D2 |
| `TestAssembleAgentFileBody_CrossGroupFallbackMissesBothGroups` | `render_test.go:979` | D2 |
| `TestAssembleAgentFileBody_FrontmatterStripModelOnAgentsTOMLSet` | `render_test.go:1043` | D3 |
| `TestAssembleAgentFileBody_FrontmatterStripToolsOnAgentsTOMLSet` | `render_test.go:1082` | D3 |
| `TestAssembleAgentFileBody_FrontmatterPreservedWhenAgentsTOMLAbsent` | `render_test.go:1144` | D3 |
| `TestRenderAgentFileFrontmatter` (preserved) | `render_test.go:335` | pre-existing |
| `TestRenderAgentFileWithoutToolGating` (preserved) | `render_test.go:370` | pre-existing |

`mage test-pkg ./internal/app/dispatcher/cli_claude/render` → 30/30 tests passed (0 failed, 0 skipped). All 10 tests above included in that count. Builder claim corroborated.

**D. mage gates** — all four GREEN:

- `mage test-pkg ./internal/app/dispatcher/cli_claude/render` → 30/30 passed (0.00s).
- `mage test-pkg ./internal/app/dispatcher` → 361/361 passed (1.69s).
- `mage test-pkg ./internal/templates` → 458/458 passed (0.01s).
- `mage test-pkg ./internal/app` → 476/476 passed (0.01s).

Combined: 1325 tests passed, 0 failed across the four packages.

**E. State flip + scope:**

- W3 sub-plan `PLAN.md:95` D2 row: `**State:** done`.
- W3 sub-plan `PLAN.md:140` D3 row: `**State:** done`.
- Scope: `git status --porcelain` shows `internal/app/dispatcher/cli_claude/render/render.go` and `internal/app/dispatcher/cli_claude/render/render_test.go` modified — both in declared D2 + D3 paths. Other dirty files (`cmd/till/init_cmd.go`, `internal/templates/embed.go`, `internal/templates/builtin/till-*.toml`, etc.) belong to other in-flight droplets (W2 till-init, W5 templates) and are not in D2 + D3's scope; verified those files' edits are unrelated to render-tier work via spec cross-reference.

**F. Cross-group fallback evidence:**

- `internal/templates/builtin/agents/till-gen/orchestrator-managed.md` exists (940 bytes). Contains the sentinel substring `"orchestrator-managed coordination kinds"` on line 3 (verified via rg).
- `internal/templates/builtin/agents/till-go/orchestrator-managed.md` does NOT exist (verified via `ls`: "No such file or directory").
- `internal/templates/embed.go:103` embeds `builtin/agents/till-gen/orchestrator-managed.md` into `DefaultTemplateFS`; no till-go counterpart embedded.
- Therefore `TestAssembleAgentFileBody_CrossGroupFallbackToTillGen` legitimately exercises the W3-FF7 cross-group fallback path: AgentName `"orchestrator-managed"` with empty `SystemPromptTemplatePath` → group `"till-go"` → primary lookup at `builtin/agents/till-go/orchestrator-managed.md` → `fs.ErrNotExist` → fallback to `builtin/agents/till-gen/orchestrator-managed.md` → hit. The test asserts the till-gen content's `"orchestrator-managed coordination kinds"` substring appears in the rendered body — corroborated by direct read of the source file.

### Certificate

- **Premises**
  1. D2 implements the 3-tier resolver with the W3-FF5 + W3-FF7 LOCKED contract; emits FULL body verbatim (no frontmatter mutation in D2).
  2. D3 implements the W3-PF1 LOCKED strip-then-inject pipeline preserving the two pre-existing test contracts.
  3. All 4 affected packages compile + test green via mage.
  4. State flip + scope match the plan.

- **Evidence**
  - P1: `render.go:443-478` + `:594-601` + `:665-693` + `:108` + `:394` + `:39` (resolver wiring, group/basename derivation, embed-tier ladder, sentinel, signature, import).
  - P2: `render.go:496-545` (strip-then-inject helper) + `:443-477` (orchestration); strip predicates lines 515-516; pipeline order lines 497-544; pass-through lines 502-510.
  - P3: 4 mage runs (render 30/30, dispatcher 361/361, templates 458/458, app 476/476), all 0 failures.
  - P4: W3 PLAN.md:95 + :140 state-done flips; git-status restricted to declared paths.

- **Trace or cases**
  - Embedded-default path: `binding.AgentName = "go-builder-agent"`, empty `SystemPromptTemplatePath` → group `"till-go"` → primary embedded read `builtin/agents/till-go/go-builder-agent.md` (verified embedded at `embed.go:98`) hits → body returned.
  - User-tier hit: `t.Setenv("HOME", tmp)` + file at `tmp/.tillsyn/agents/till-go/go-builder-agent.md` → resolver short-circuits at user tier; sentinel `"SENTINEL_USER_TIER"` flows through D3 strip-then-inject unchanged (no strip targets in fixture frontmatter, no inject).
  - Project-tier hit: `<project>/.tillsyn/agents/go-builder-agent.md` planted → tier 1 wins over user tier.
  - Cross-group fallback: `binding.AgentName = "orchestrator-managed"`, empty `SystemPromptTemplatePath` → group `"till-go"` → primary miss → fallback to `till-gen/orchestrator-managed.md` → hit (940-byte file with sentinel substring on line 3).
  - Both-miss: AgentName `"nonexistent-agent"` → primary till-go miss → fallback till-gen miss (file does not exist in either group) → `ErrAgentBodyNotFound` wrapped + bubbled to `Render`'s rollback path → test asserts `errors.Is(err, render.ErrAgentBodyNotFound)`.
  - D3 model-strip: `binding.Model = ptr("sonnet")`, fixture has `model: opus` line → `stripModel=true` → `config.StripFrontmatterKeys` removes `model:` from frontmatter → body emerges without `model:` line; `name:` survives + body-bytes-preserve-marker survives.
  - D3 tools-strip-and-inject: fixture has stale `tools: Read, Bash` + `allowedTools: Read` + `disallowedTools: WebFetch` → all stripped (`stripTools=true` always) → runtime `binding.ToolsAllowed = ["Read"]` injected as `allowedTools: Read` → `ToolsDisallowed` empty so no `disallowedTools:` line.
  - D3 absent-AgentsTOML: `binding.Model = ptr("")` + nil tool slices → `stripModel=false` (predicate `*Model != ""` is false) → `model: opus` preserved; `stripTools=true` always → `tools: Read, Bash` stripped; nil tool slices → no inject lines.
  - Preserved `TestRenderAgentFileFrontmatter` with `fixtureBinding()` ToolsAllowed=["Read","Grep"] + ToolsDisallowed=["WebFetch","Bash(curl *)"] on embedded `till-go/go-builder-agent.md` (no model/tools in disk frontmatter) → no strip targets → inject appends both lines → test substrings present.
  - Preserved `TestRenderAgentFileWithoutToolGating` with empty tool slices on same embedded file → no inject lines → test asserts both substrings absent.

- **Conclusion**
  PASS. All six A–F contracts verified; both preserved tests stay green via the strip-then-inject pipeline; 1325 tests pass across 4 packages; state flips and scope match plan; cross-group fallback path is genuinely exercised by the file-system shape (till-gen has `orchestrator-managed.md`, till-go does not).

- **Unknowns**
  None blocking. The other dirty files in `git status` belong to in-flight peer droplets (W2 till-init, W5 templates) and were verified out-of-scope for D2 + D3 by description-spec match — no QA action required here.

### Summary

**PASS.**

D2's 3-tier resolver and D3's strip-then-inject pipeline jointly satisfy the W3-PF1 LOCKED contract: D2 emits FULL body verbatim from the resolver; D3 layers strip-then-inject restoring the two pre-existing test contracts (`TestRenderAgentFileFrontmatter` + `TestRenderAgentFileWithoutToolGating`). 8 new tests + 2 preserved tests = 10/10 GREEN inside the render package's 30/30 mage run. The four affected packages (`render`, `dispatcher`, `templates`, `app`) all return 0 failures across 1325 tests. State flips on the W3 sub-plan PLAN.md (D2:95 done, D3:140 done) and the worktree changes are scoped to `render.go` + `render_test.go` per spec. Cross-group fallback path genuinely exercises the embed-FS ladder (till-go has no `orchestrator-managed.md`; till-gen does).

D5 (post-render validator) and D6 (doc-comment correction) unblock on this pair's `done` state per the W3 chain (`D5 Blocked by: D2, D3`; `D6 Blocked by: D2, D3, D5`).

### Hylla Feedback

N/A — Hylla today indexes only committed Go code; D2 + D3's production + test edits are all uncommitted at review time, and the cited prior-state surfaces (preserved tests at `render_test.go:331-401`, embed declarations at `embed.go:75-103`, templates frontmatter helper at `frontmatter.go:89`) were either freshly-landed in this drop or HEAD-tracked and stable. I used `rg` for the embed-directive scan + sentinel-substring locate, and direct `Read` for render.go, render_test.go, PLAN.md, the W3 sub-plan, and the cross-group fixture file — covering the proof surface with zero ambiguity. No Hylla query was attempted because (a) most surfaces are uncommitted, (b) the changes are fully captured in `git diff` against HEAD, (c) the embed-directive + sentinel-substring locate is a structural-file query better served by `rg`.

---

## Droplet 4c.6.W5.D3 — Round 1

**Reviewer:** go-qa-proof-agent (subagent).
**Date:** 2026-05-10.
**Droplet:** `4c.6.W5.D3 — Drop go- prefix from agent_name in till-go.toml + remove tools from frontmatter + W5-D2-FF1 doc-comment absorption`.
**Verdict:** **PASS** (with one soft plan-drift finding flagged for orchestrator follow-up, no builder defect).

### Check matrix

| Check | Subject | Evidence | Result |
| --- | --- | --- | --- |
| A | `agent_name = "go-"` absent under `internal/templates/builtin/` | `git grep "agent_name = \"go-" internal/templates/builtin/` → exit 1 (no hits) | PASS |
| A | 7 specific kinds renamed | `git diff till-go.toml` shows L406→`planning-agent`, L436→`research-agent`, L448→`builder-agent`, L482→`qa-proof-agent`, L508→`qa-falsification-agent`, L534→`qa-proof-agent`, L567→`qa-falsification-agent` | PASS |
| B | Frontmatter strip on placeholder MDs | `git grep -nE "^(tools\|model\|allowedTools\|disallowedTools):" internal/templates/builtin/agents/` → exit 1 (no hits). Direct `Read` of 7 till-go/ placeholders + 1 till-gen/builder-agent.md + 1 till-gdd/builder-agent.md shows frontmatter == `name` + `description` only. Per worklog, this was a no-op verification: W1.D1 shipped these clean from inception. | PASS |
| C | `internal/templates/load.go:388` doc-comment dual-history | Read L387-394: `till-go.toml + till-gen.toml ← default-go.toml + default-generic.toml, rebadged in Drop 4c.6 W5.D1 + W5.D2` | PASS |
| C | `internal/templates/load.go:1240` doc-comment dual-history | Read L1237-1246: same dual-history pattern | PASS |
| C | `internal/templates/load.go:1385` doc-comment (additional site absorbed) | Read L1385-1389: `embedded till-go.toml (rebadged from default-go.toml in Drop 4c.6 W5.D1)` — single-history because original comment only referenced default-go.toml | PASS |
| C | `internal/templates/load.go:2098` doc-comment (additional site absorbed) | Read L2098-2103: explicit W5.D3 prefix-strip note: `Drop 4c.6 W5.D3 dropped the go- prefix from agent_name values; current names are bare builder-agent, planning-agent, etc.` | PASS |
| C | `internal/app/auto_generate_steward.go:108` doc-comment | Read L107-115: `till-gen vs till-go ← default-generic vs default-go, rebadged in Drop 4c.6 W5.D1 + W5.D2` | PASS |
| C | `internal/templates/embed.go` W1.D1 cross-droplet handoff updated | Read L58-68: explicit W5.D3 paragraph recording the prefix-strip outcome + legacy `go-*-agent.md` orphaning rationale | PASS |
| D | `model` field deliberately KEPT in till-go.toml | `git grep "^model = " till-go.toml` returns 12 hits (one per binding). Builder kept `model` per CRITICAL constraint at PLAN.md L242 + RiskNotes at L239 (schema-level field removal deferred to Drop 4c.7). | PASS |
| E.1 | `mage test-pkg ./internal/templates` | `[SUCCESS] All tests passed — 458 tests passed across 1 package` | PASS (458/458) |
| E.2 | `mage test-pkg ./internal/app` | `[SUCCESS] All tests passed — 476 tests passed across 1 package` | PASS (476/476) |
| E.3 | `mage test-pkg ./internal/adapters/server/mcpapi` | `[SUCCESS] All tests passed — 226 tests passed across 1 package` | PASS (226/226) |
| E.test-content | `TestDefaultTemplateAgentBindingsCoverAllKinds` (embed_test.go:380) | Asserts 12 bindings + each `Validate()` clean. Validate requires `Model` non-empty → builder's "keep model" decision is necessary. Test passes inside the 458/458. | PASS |
| E.test-content | `TestDefaultTemplateBuildersRunOpus` (embed_test.go:402) | Asserts `binding.Model == "opus"` for 7 kinds. Builder's "keep model = opus" decision keeps this green. Test passes inside the 458/458. | PASS |
| E.test-content | `TestLoadDefaultTemplateForLanguage_Go` (embed_test.go:927) | Asserts 12 agent bindings via `len(allKinds)` — agent_name-value-agnostic, robust to rename. Inside the 458/458. | PASS |
| E.test-content | `embed_test.go:1046-1051 w1d1StandardAgentNames` | Closed list `planning-agent.md, builder-agent.md, qa-proof-agent.md, qa-falsification-agent.md, research-agent.md` — confirms test infrastructure expects bare names. | PASS |
| F | L1 `workflow/drop_4c_6/PLAN.md` W5.D3 row | Read L223: `**State:** done`. `git diff PLAN.md` shows ONLY the W5.D3 state flip; no other rows edited. | PASS |
| F | Edits scoped to declared paths + 3 W5-D2-FF1 absorbed sites | `git status` shows W5.D3-attributable diff: `till-go.toml`, `auto_generate_steward.go`, `load.go`, `embed.go`, `PLAN.md`, `BUILDER_WORKLOG.md`. NOT in W5.D3 scope: `cmd/till/init_cmd.go|test`, `render.go|test`, `till-gen.toml`, W2/W3 sub-plan PLAN.mds — all belong to concurrent W2.D3b + W3.D2 + W3.D3 builders (their separate worklog rounds confirm). `till-gen.toml` IS in W5.D3 scope per PLAN L225 — diff is a 15-line doc-comment update. | PASS |
| G | Legacy `go-*-agent.md` placeholders still in tree | `git ls-files 'internal/templates/builtin/agents/till-go/go-*-agent.md'` → 5 files present. `embed.go:98-102` retains `//go:embed` directives for them. Builder routed deletion to follow-up drop per `feedback_orphan_via_collapse_defer_refinement.md`. | PASS |

### Findings

- **W5-D3-PF1 (informational, plan-drift, not a builder defect):** `workflow/drop_4c_6/PLAN.md` L245 KindPayload `shape_hint` reads `"drop go- prefix from agent_name; remove tools field; remove model field"`. This conflicts with the same droplet's RiskNotes at L239 (`Schema-level field removal from templates.AgentBinding is OUT OF SCOPE — would break tests + adapter contracts; deferred to Drop 4c.7`) and the constraint ContextBlock at L242 (`schema-level field removal deferred — this droplet edits SHIPPED files only`). The builder correctly followed the constraint and kept `model` — removing it would have failed `TestDefaultTemplateAgentBindingsCoverAllKinds` (Validate requires Model non-empty per schema.go:776) AND `TestDefaultTemplateBuildersRunOpus` (asserts `Model == "opus"`). The KindPayload `shape_hint` should be amended in a future plan-correction droplet OR the inconsistency should be explicitly resolved in the Drop 4c.7 planner's inheritance brief. No action required from this builder; routed back to the orchestrator for tracking.

- **W5-D3-PF2 (informational, out-of-scope-by-design, not a builder defect):** `.tillsyn/template.toml` (the self-host dogfood seed introduced in Drop 4c.5 F.2.3 as a byte-identical copy of the then-`default-go.toml`) still contains 7 `agent_name = "go-..."` rows because it was NOT in W5.D3's declared `Paths`. The file lives outside `internal/templates/builtin/` and is conceptually a downstream consumer of the rebadged builtin. Builder correctly did not touch it. Routed to the orchestrator: a future sync droplet (likely paired with Drop 4c.7's schema-level removal) should re-sync `.tillsyn/template.toml` against the post-W5.D3 + post-4c.7 `till-go.toml` shape so the self-host dogfood matches the embedded builtin.

- **W5-D3-PF3 (informational, deferred-cleanup, not a builder defect):** 5 legacy `go-*-agent.md` placeholder files under `internal/templates/builtin/agents/till-go/` remain in the embed.FS (per `embed.go:98-102`). Their doc-comments self-describe as "PLACEHOLDER — legacy go-prefixed builder name retained until Drop 4c.6 W5.D3 strips the go- prefix" and "this file goes away alongside that cleanup." The W5.D3 PLAN.md Paths enumeration does NOT include these files, so the builder correctly left them in place per strict path-scope discipline (`feedback_orphan_via_collapse_defer_refinement.md`). Routed to the orchestrator: a follow-up cleanup drop (a candidate slot is Drop 4c.7 alongside schema removal) should `git rm` these 5 files + their 5 `//go:embed` directives at `embed.go:98-102`.

### Missing evidence

None. All seven A-G check categories closed with direct file reads, `git grep` scans, and mage gate runs.

### Hylla Feedback

N/A — this QA review touched only TOML, markdown, and Go doc-comments in already-known symbols (`reachabilityStandaloneKinds`, `embeddedAgentLibraryShipped`, `seedStewardAnchors`, `DefaultTemplateFS`). Hylla is Go-source-only today; the Go doc-comment edits are at specific line ranges enumerated in the appendix + worklog. `Read` on each known line range, `git grep` for cross-tree symbol scans, and `git diff` for delta verification covered the proof surface exhaustively with zero ambiguity. No Hylla query was attempted because (a) the edits are doc-comments inside committed symbols with stable line addresses, (b) the `git grep` + `Read` combination is more precise than any keyword/vector search for line-pinned comment verification, (c) the load-bearing test surfaces (`embed_test.go`, `load_test.go`) were located via `git grep` on specific test-function-name substrings, which is the appropriate evidence shape for "does this test still pass after the rename."

### Summary

**PASS.** Builder shipped a clean W5.D3 droplet: 7 `agent_name = "go-*"` values stripped to bare names in `till-go.toml`, 11 `tools = []` rows removed, header doc-comment rewritten to record the W5.D3 rebadge + agents.toml tool-authority rule, 3 W5-D2-FF1 doc-comment absorption sites (load.go:388 + load.go:1240 + auto_generate_steward.go:108) plus 2 additional related sites (load.go:1385 + load.go:2098) updated with dual-history records, embed.go cross-droplet handoff extended with W5.D3 outcome paragraph, till-gen.toml header rewrite paired, state flipped to `done` on L1 PLAN.md. Placeholder MD frontmatter strip was a no-op verification (W1.D1 shipped clean from inception). `model` field deliberately kept per the CRITICAL constraint — schema-level field removal is Drop 4c.7's concern. Three soft findings (W5-D3-PF1 plan KindPayload drift, W5-D3-PF2 `.tillsyn/template.toml` self-host seed re-sync, W5-D3-PF3 legacy `go-*-agent.md` cleanup) are routed to the orchestrator as informational; none block this droplet's completion. Mage gates green: 458 + 476 + 226 = 1160 tests across 3 declared packages.

---

## Droplet 4c.6.W2.D1 — Round 1

**Reviewer:** go-qa-proof-agent (subagent, build-QA-proof axis).
**Date:** 2026-05-11.
**Droplet:** `4c.6.W2.D1 — internal/fsatomic/ atomic file-write helper (local-implement, ROUND-3 pivot)`.
**Parent kind:** `build`.
**Artifact under review:** `internal/fsatomic/atomic.go` + `internal/fsatomic/atomic_test.go` (untracked-new files at review time) plus PLAN.md state flip + BUILDER_WORKLOG.md round entry.

### Proof certificate

**Premises.**
- P1: Exported function signature `WriteFile(path string, data []byte, perm os.FileMode) error` matches the spec byte-exact.
- P2: Implementation uses write-temp-in-same-dir + sync + chmod + close + rename, with defer-guarded cleanup-on-error.
- P3: Four tests exist with the named names and cover the claimed behaviors.
- P4: `mage test-pkg ./internal/fsatomic` GREEN.
- P5: `mage test-func ./internal/fsatomic 'TestWriteFile_*'` GREEN with 4/4.
- P6: No `internal/vendor/` directory exists (ROUND-3 pivot).
- P7: No `VENDOR_SOURCE.md` artifact exists.
- P8: No "DO NOT EDIT" header in `atomic.go`.
- P9: PLAN.md W2.D1 state is `done`.

**Evidence.**
- E1 (P1): `internal/fsatomic/atomic.go:46` — `func WriteFile(path string, data []byte, perm os.FileMode) error`. Byte-exact match.
- E2 (P2): `atomic.go:47-91` — `dir := filepath.Dir(path)`; `os.CreateTemp(dir, base+".tmp-*")` (same-dir temp); `success := false` + `defer { if !success { os.Remove(tmpName) } }` (cleanup guard); `f.Write(data)` → `f.Sync()` → `f.Chmod(perm)` → `f.Close()` → `os.Rename(tmpName, path)` → `success = true`. Order matches spec verbatim.
- E3 (P3): `atomic_test.go` declares `TestWriteFile_FreshWrite` (L18), `TestWriteFile_OverwritesExisting` (L38), `TestWriteFile_CleansUpTempOnError` (L67), `TestWriteFile_PreservesPermissions` (L99). All 4 names present, all assert real behavior (content match, overwrite no-residue, no leaked `.tmp-*` + target ENOENT, `info.Mode().Perm() == 0o600`).
- E4 (P4): `mage test-pkg ./internal/fsatomic` → `[PKG PASS] github.com/evanmschultz/tillsyn/internal/fsatomic (0.00s)` — `tests: 4, passed: 4, failed: 0`. Captured this round.
- E5 (P5): `mage test-func ./internal/fsatomic 'TestWriteFile_FreshWrite|TestWriteFile_OverwritesExisting|TestWriteFile_CleansUpTempOnError|TestWriteFile_PreservesPermissions'` → `[PKG PASS] (1.31s)` with `-race -count=1`. `tests: 4, passed: 4, failed: 0`. Captured this round.
- E6 (P6): `ls internal/` returns 9 entries (adapters / app / buildinfo / config / domain / fsatomic / platform / templates / tui); no `vendor/`. `ls third_party/` returns only `teatest_v2/` — unrelated pre-existing test helper.
- E7 (P7): `ls internal/fsatomic/` returns exactly `atomic.go` + `atomic_test.go`. `ls workflow/drop_4c_6/` has no `VENDOR_SOURCE.md`. Repo root `ls` has no `VENDOR_SOURCE.md`.
- E8 (P8): Direct `Read` of `atomic.go` lines 1-26 — file opens with `// Package fsatomic provides minimal atomic file-write helpers...`. No `// Code generated`, no `// DO NOT EDIT`, no `// vendored from` header.
- E9 (P9): `workflow/drop_4c_6/DROP_4c.6.W2_TILL_INIT/PLAN.md:45` — `- **State:** done` under the `### Droplet 4c.6.W2.D1` heading at L41.

**Trace or cases.**
- Case A (signature byte-exactness): spec `"WriteFile(path string, data []byte, perm os.FileMode) error"` vs atomic.go:46 same identifier list, same types, same order. **Match.**
- Case B (pattern verification): walk atomic.go:47-91 — `os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".tmp-*")` (write-temp-in-same-dir) → `f.Write(data)` (write) → `f.Sync()` (sync) → `f.Chmod(perm)` (chmod) → `f.Close()` (close) → `os.Rename(tmpName, path)` (rename) → `success = true`. Deferred `os.Remove(tmpName)` fires on every early-return error path before the `success = true` assignment. **All five steps in spec order, cleanup defer-guarded.**
- Case C (test enumeration): `grep ^func TestWriteFile_` of atomic_test.go yields exactly 4 — `FreshWrite`, `OverwritesExisting`, `CleansUpTempOnError`, `PreservesPermissions`. Each test body invokes `WriteFile` end-to-end and asserts a distinct claim. **4/4 present and behaviorally meaningful (not just call-through stubs).**
- Case D (mage test-pkg): captured stdout/stderr from `mage test-pkg ./internal/fsatomic` shows `[PKG PASS]` and `4 tests passed across 1 package`. **GREEN.**
- Case E (mage test-func selector): captured stdout from selector regex `TestWriteFile_FreshWrite|TestWriteFile_OverwritesExisting|TestWriteFile_CleansUpTempOnError|TestWriteFile_PreservesPermissions` against `-race -count=1` shows 4/4 passed. **GREEN.**
- Case F (no vendor dir): two-level evidence — (i) `ls internal/` shows no `vendor/` entry; (ii) `git status` declares the only untracked path under `internal/` is `internal/fsatomic/`. No hidden `internal/vendor/` could exist that `git status` would not report. **Absent.**
- Case G (no VENDOR_SOURCE.md): `ls internal/fsatomic/` returns only `atomic.go` and `atomic_test.go`; `ls workflow/drop_4c_6/` listing carries no `VENDOR_SOURCE.md`; repo-root `ls` carries no `VENDOR_SOURCE.md`. **Absent everywhere it would plausibly land.**
- Case H (no DO NOT EDIT header): atomic.go opens with the package doc-comment at L1 (`// Package fsatomic provides minimal atomic file-write helpers...`). No vendor/generated marker in lines 1-26. **Absent.**
- Case I (PLAN.md state): W2 PLAN.md L45 reads `- **State:** done` directly under the W2.D1 heading at L41. **State flipped.**

**Conclusion.** PASS. Every spec-pinned premise P1-P9 is proven by direct file evidence and live mage gate output. Implementation matches the textbook pattern (same-dir temp + sync + chmod + close + rename + defer cleanup), and the test suite exercises each claimed behavior with assertions that would fail if the implementation degraded.

**Unknowns.**
- U1: `TestWriteFile_CleansUpTempOnError` does not exercise the post-Write/Sync/Chmod error paths (only the pre-CreateTemp failure path). The cleanup defer is verified textually (Case B) but only one of the five potential failure points is exercised by test; the others rely on code inspection plus general defer semantics. Acceptable for this droplet because the defer + success-flag pattern is uniform across all paths and an inspection-grade proof is sufficient at this LOC, but a future hardening drop could inject failures at each step. Recorded as a minor refinement candidate.
- U2: No concurrent-reader atomicity test (`TestWriteFile_AtomicVisibility` was a stretch goal in PLAN.md acceptance and explicitly marked "skip if too flaky"). Builder skipped per acceptance — out-of-scope, not a finding.

### Verification table

| Check | Evidence | Verdict |
|---|---|---|
| A — Signature byte-exact | atomic.go:46 `func WriteFile(path string, data []byte, perm os.FileMode) error` vs spec | PASS |
| B — Write-temp-same-dir + sync + chmod + close + rename + defer cleanup | atomic.go:47-91 walk; all 5 steps in spec order; success-guarded defer | PASS |
| C — 4 named tests present | atomic_test.go:18, 38, 67, 99 — `FreshWrite`, `OverwritesExisting`, `CleansUpTempOnError`, `PreservesPermissions` | PASS |
| D — `mage test-pkg ./internal/fsatomic` GREEN | `[PKG PASS]` + 4/4 captured this round | PASS |
| E — `mage test-func ./internal/fsatomic TestWriteFile_*` 4/4 GREEN | `[PKG PASS]` 1.31s with `-race -count=1`, 4/4 captured this round | PASS |
| F — NO `internal/vendor/` | `ls internal/` returns 9 entries, no `vendor/`; `git status` only has `internal/fsatomic/` untracked | PASS |
| G — NO `VENDOR_SOURCE.md` | absent from `internal/fsatomic/`, from `workflow/drop_4c_6/`, from repo root | PASS |
| H — NO "DO NOT EDIT" header in atomic.go | atomic.go:1 opens `// Package fsatomic provides...` (package doc-comment, no vendor/generated marker) | PASS |
| I — PLAN.md W2.D1 state `done` | `workflow/drop_4c_6/DROP_4c.6.W2_TILL_INIT/PLAN.md:45` reads `- **State:** done` | PASS |

### Findings

- **W2-D1-PF1 (informational, test-coverage gap, not a defect):** `TestWriteFile_CleansUpTempOnError` exercises only the pre-CreateTemp failure path (missing parent dir). The cleanup defer + `success` flag pattern is uniform across the four other potential failure points (Write, Sync, Chmod, Close), and code inspection plus standard `defer` semantics justify the proof in this LOC, but a hardening drop could inject errors at each step (perhaps via a test-only filesystem seam). Recorded as a minor refinement candidate; not blocking this droplet because the defer-guard inspection is sufficient for ~91 LOC and the acceptance criterion only required "the failed call leaves no `.tmp-*`," which the existing test verifies.

- **W2-D1-PF2 (informational, intermediate state-flip omission, not a defect):** PLAN.md W2.D1 transitioned `todo → done` in a single edit without writing the intermediate `in_progress` step. The builder worklog (BUILDER_WORKLOG.md L2003) acknowledges this explicitly and cites WORKFLOW.md Phase 4 step 3 as allowing direct close on single-round completion. The verify-step I asks only that the final state be `done`, which it is — so this is not a finding against the droplet. Flagging because some sibling rounds (W3.D2 round-2, W3.D3) wrote both transitions; consistency would be a small ergonomic refinement. Routed to the orchestrator as informational.

### Missing evidence

None. All nine A-I check categories closed with direct file reads, captured mage stdout/stderr, and absence verification via two-level evidence (`ls` + `git status`).

### Hylla Feedback

None — Hylla answered nothing because no Hylla query was warranted. The droplet creates a brand-new Go package (`internal/fsatomic/`) that does not exist in any prior ingest, the two source files are untracked-new (Hylla cannot see uncommitted files anyway), and the verify surface is small enough (91 LOC + 115 LOC + 4 absence checks) that direct `Read` of all of it plus live mage runs is the appropriate evidence shape. Stdlib semantics for `os.CreateTemp`/`os.Rename`/`(*os.File).Sync`/`(*os.File).Chmod` are background knowledge confirmed by the documented behavior in atomic.go's design comments. No fallback misses to record.

### Summary

**PASS.** Builder shipped a textbook write-temp-in-same-dir + sync + chmod + close + rename atomic helper at `internal/fsatomic/atomic.go` (91 LOC, exports exactly one function `WriteFile(path string, data []byte, perm os.FileMode) error` matching the spec byte-exact) with four behavior-testing tests at `internal/fsatomic/atomic_test.go` (115 LOC) covering fresh-write, overwrite, temp-cleanup-on-error, and permission-preservation. ROUND-3 pivot honored: no `internal/vendor/`, no `VENDOR_SOURCE.md`, no "DO NOT EDIT" header. Mage gates green this round — `mage test-pkg ./internal/fsatomic` reports 4/4 in 0.00s; `mage test-func ./internal/fsatomic TestWriteFile_*` reports 4/4 in 1.31s with `-race -count=1`. PLAN.md W2.D1 state flipped to `done`. Two soft findings (W2-D1-PF1 cleanup-defer coverage gap, W2-D1-PF2 missing intermediate `in_progress` write) are informational and routed to the orchestrator; neither blocks this droplet's completion.

---

## Droplet 4c.6.W3.D4 — Round 1

**Reviewer:** go-qa-proof-agent (subagent, build-QA-proof axis).
**Date:** 2026-05-11.
**Droplet:** `4c.6.W3.D4 — Defense-in-depth env vars in cli_claude/env.go`.
**Parent kind:** `build`.
**Artifact under review:** uncommitted working-tree diff against `internal/app/dispatcher/cli_claude/env.go` + `internal/app/dispatcher/cli_claude/adapter_test.go`.
**Spec sources:** `workflow/drop_4c_6/DROP_4c.6.W3_BUNDLE_AND_ISOLATION/PLAN.md` § W3.D4 (line 183-220); `workflow/drop_4c_6/BUILDER_WORKLOG.md` § Droplet 4c.6.W3.D4 — Round 1.

### Acceptance trace (A-I)

- **A (4 defense-in-depth literals injected).** `env.go:41-53` declares `defenseInDepthEnvLiterals` as a `[]struct{Name, Value string}` with exactly four entries in declaration order: `CLAUDE_CODE_DISABLE_BACKGROUND_TASKS=1` (L46), `CLAUDE_CODE_FORK_SUBAGENT=0` (L48), `DISABLE_AUTOUPDATER=1` (L50), `DISABLE_TELEMETRY=1` (L52). Verbatim match against spec. PASS.
- **B (Injection writes UNCONDITIONALLY).** Injection loop at `env.go:145-150` carries values inline — no `os.LookupEnv`, no conditional source on the orchestrator process. The only skip condition is `alreadySet` (binding-precedence guard), which is the spec's binding-override path, not a conditional-on-environment path. `TestEnvCarriesDefenseInDepthEnvVars` (`adapter_test.go:697-738`) proves unconditional emission by `os.Unsetenv`ing all four names first, then asserting all four `name=value` strings remain in `cmd.Env`. PASS.
- **C (Precedence chain: binding > defense-in-depth > closed-baseline).** Three-tier trace through `assembleEnv`: (i) `binding.Env` loop at `env.go:131-137` populates `emitted[name] = val` for binding names first; (ii) defense-in-depth loop at L145-150 skips on `alreadySet` so binding wins; (iii) closed-baseline loop at L158-167 also skips on `alreadySet` so both prior tiers win. The slice-builder mirrors the precedence: baseline emit (L176-183), then defense-in-depth emit (L184-194), then sorted binding-only tail (L196-206). Walked the override test case: `binding.Env: ["DISABLE_TELEMETRY"]` + `t.Setenv("DISABLE_TELEMETRY", "0")` → emitted at L131-136 with value `"0"`; defense loop L145 sees `alreadySet=true` → skips, leaves `emitted["DISABLE_TELEMETRY"]="0"` unchanged; slice-builder defense-loop L184-194 emits via `emitted` (value `"0"`) once; binding-only tail L196-206 skips because already seen. Result: `cmd.Env` carries `DISABLE_TELEMETRY=0` exactly once, no `DISABLE_TELEMETRY=1` leak. PASS.
- **D (Separate slice + injection-loop shape chosen).** `defenseInDepthEnvLiterals` is a sibling declaration of distinct shape (`[]struct{Name, Value string}`) — NOT an extension of `closedBaselineEnvNames` (`[]string`). The doc comment at L19-40 explicitly contrasts the two slices ("Unlike `closedBaselineEnvNames`..."). This matches the PLAN.md "preferred" shape verbatim. PASS.
- **E (`TestEnvCarriesDefenseInDepthEnvVars` asserts all 4 literals via `slices.Contains`).** `adapter_test.go:697-738`: builds the four-string expected slice at L729-733, walks with `for _, want := range expected` at L734-738, asserts `slices.Contains(cmd.Env, want)` for each. `slices` import added at `adapter_test.go:12`. PASS.
- **F (`TestEnvDefenseInDepthOverridableByBindingEnv` asserts binding override).** `adapter_test.go:740-787`: `t.Setenv("DISABLE_TELEMETRY", "0")` (L749), `binding.Env: []string{"DISABLE_TELEMETRY"}` (L754), then asserts (i) `cmd.Env` contains `DISABLE_TELEMETRY=0` (L771-773, binding wins), (ii) does NOT contain `DISABLE_TELEMETRY=1` (L774-776, literal must not leak alongside override), (iii) the other three literals still emit at default values (L778-785). Three-pronged assertion exhaustively covers the precedence-clause RISK in `ISOLATION_ENFORCEMENT_FIX.md` § D.3. PASS.
- **G (`TestEnvNotInheritedFromOSEnviron` length assertion adjusted).** `adapter_test.go:370-376`: `wantSize := len(closedBaselineEnvNames) + len(defenseInDepthEnvLiterals) + 1` (was `+ 1` alone pre-change). Error message at L374 rewritten to surface all three components (`baseline %d + defenseInDepth %d + binding 1`). PASS.
- **H (`mage test-pkg ./internal/app/dispatcher/cli_claude` GREEN).** Ran in QA session: 24 tests passed / 0 failed / 0 skipped / 1 package passed. PASS.
- **I (PLAN.md W3.D4 state `done`).** `DROP_4c.6.W3_BUNDLE_AND_ISOLATION/PLAN.md:183` flipped `todo → done` (single-line diff, no other PLAN.md mutations on this row). PASS.

### Findings

- **W3-D4-PF1 (informational, accept).** LOC counts in the build-artifact spec (+45 / +85) diverge from actual diff (+68 / +96). Net code is correctly scoped — divergence reflects doc-comments + the slice-builder extension landing slightly larger than the planner's estimate. No production-behavior delta. Routes as informational only; no action requested.

### Missing Evidence

None. Every A-I check resolves to a concrete file+line citation or a directly-observed mage-gate run.

### Out-of-scope mutation check

`git status --porcelain` shows additional dirty paths (`cmd/till/init_cmd.go`, `cmd/till/init_cmd_test.go`, `internal/app/dispatcher/cli_claude/render/render.go`, `internal/app/dispatcher/cli_claude/render/render_test.go`, untracked `internal/fsatomic/`). All cross-referenced to `BUILDER_WORKLOG.md` § Droplet 4c.6.W2.D1 (the `internal/fsatomic` package) and § Droplet 4c.6.W3.D5 (render.go + render_test.go) and the W2 init-cmd droplets — none authored by the W3.D4 builder spawn. W3.D4 builder's own touched paths (env.go + adapter_test.go) match the spec. CLEAN.

### Summary

**PASS.** Builder shipped the W3.D4 defense-in-depth env vars cleanly: 4 literal `(name, value)` pairs in `defenseInDepthEnvLiterals` injected unconditionally into every claude spawn's `cmd.Env`, separate-slice shape (preferred per PLAN.md), three-tier precedence chain (`binding > defense-in-depth > closed-baseline`) implemented + tested via the asymmetric override test, length-assertion adjustment in `TestEnvNotInheritedFromOSEnviron` accounts for the four new unconditional emissions, `slices.Contains` used in both new tests, `mage test-pkg` 24/24 GREEN, PLAN.md W3.D4 flipped to `done`. One informational finding (LOC-count divergence) is non-blocking. Verdict: PASS.

### Hylla Feedback

N/A — this QA review verified an uncommitted working-tree diff. Hylla indexes committed Go only, and Hylla resolution against committed state would surface the pre-D4 file (no `defenseInDepthEnvLiterals` symbol). Direct `git diff` + `Read` against env.go + adapter_test.go was the appropriate evidence shape for diff-based proof. No Hylla query attempted; no fallback misses to record.

---

## Droplet 4c.6.W2.D4 — Round 1

**Reviewer:** go-qa-proof-agent (subagent).
**Date:** 2026-05-11.
**Droplet:** `4c.6.W2.D4 — runInitTUI bubbletea walk for project name + group picker`.
**Verdict:** **PASS** (no blocking findings; one informational NIT on test-coverage symmetry, builder defense-in-depth already mitigates).

### Check matrix

| Check | Subject | Evidence | Result |
| --- | --- | --- | --- |
| A | `runInitTUI` D3a stub replaced with bubbletea walk | `init_cmd.go:324-352` — `runInitTUI` calls `os.Getwd()` → `newInitTUIModel(cwd)` → `programFactory(m).Run()` → type-asserts back to `initTUIModel` → dispatches on `Cancelled()` / `Done()`. No `errors.New("till init: TUI walk not yet wired ...")` string anywhere. `git grep "TUI walk not yet wired" cmd/till/` returns zero hits. | PASS |
| A | Walk types + methods exist | `init_cmd.go:87-106` (`initTUIStep` + 4 step constants), `:112-115` (`initTUIGroupRow`), `:120-124` (`initTUIGroupRows` with 3 rows incl. `till-gdd` `Disabled: true`), `:133-139` (`initTUIModel`), `:145-160` (`newInitTUIModel`), `:166-168` (`Init`), `:174-233` (`Update`), `:239-267` (`View`), `:271-286` (`Done` / `Cancelled` / `Payload`), `:291-310` (`nextEnabledGroupRow` / `prevEnabledGroupRow`). | PASS |
| B | 3 new tea-tests present with EXACT names | `init_cmd_test.go:153` `TestRunInitTUI_AcceptsDefaultNameAndSelectsTillGo`, `:218` `TestRunInitTUI_DisabledTillGddIsUnselectable`, `:267` `TestRunInitTUI_EscCancelsWalk`. All three drive `teatest.NewTestModel(newInitTUIModel(cwd), ...)`. | PASS |
| C | Disabled-`till-gdd` no-advancement: 2 `down` from `till-gen` lands on `till-go` | Trace: `initTUIGroupRows = [{till-gen,false}, {till-go,false}, {till-gdd,true}]`. Start cursor=0 (`till-gen`). Down 1: `nextEnabledGroupRow(0)` → i=1, `Disabled=false` → return 1 (`till-go`). Down 2: `nextEnabledGroupRow(1)` → i=2, `Disabled=true` → skip; loop exits; return `cur=1`. Final cursor=1 (`till-go`). Enter on `till-go` → `Disabled=false` → finalize. `TestRunInitTUI_DisabledTillGddIsUnselectable` (`init_cmd_test.go:218-262`) drives `Enter, Down, Down, Enter` and asserts `Payload().Group == "till-go"` (`:259-261`). Test passes (race detector on). | PASS |
| C | Defense-in-depth: Enter on a disabled row is no-op | `init_cmd.go:212-220` — `case key.Code == tea.KeyEnter: row := initTUIGroupRows[m.groupCursor]; if row.Disabled { return m, nil }`. Belt-and-suspenders alongside the cursor-skip helpers. | PASS |
| D | D5-stub error byte-exact `"till init: file copy not yet wired (W2.D5)"` | `git grep "till init: file copy not yet wired" cmd/till/` returns 4 hits: `init_cmd.go:351` (TUI branch, NEW in D4), `init_cmd.go:376` (JSON branch, D3b-shipped), `init_cmd_test.go:53` (smoke test assertion), `init_cmd_test.go:70` (JSON smoke test assertion). Both production sites use literal `errors.New(...)` — no `fmt.Sprintf`, no concatenation — so the byte-exact contract is structurally preserved. | PASS |
| E | `programFactory` test seam reused per `main_test.go:253-257` pattern | Production `init_cmd.go:333` — `finalModel, err := programFactory(m).Run()`. Smoke test `init_cmd_test.go:30-57` shape mirrors `main_test.go:252-267`: `origFactory := programFactory; t.Cleanup(func() { programFactory = origFactory }); programFactory = func(m tea.Model) program { ... return scriptedProgram{...} }`. `scriptedProgram` is the existing test fixture at `main_test.go:68-79` — no new test-substrate code introduced. | PASS |
| F | `mage test-pkg ./cmd/till` GREEN | Reviewer-run `mage test-pkg ./cmd/till`: `[SUCCESS] All tests passed — 268 tests passed across 1 package`. Targeted `mage test-func ./cmd/till 'TestRunInitTUI_AcceptsDefaultNameAndSelectsTillGo\|TestRunInitTUI_DisabledTillGddIsUnselectable\|TestRunInitTUI_EscCancelsWalk\|TestInit_BareInvocation_ReturnsTUIStubError'` → 4 passed, 0 failed (race detector on). Pre-D4 baseline init-prefixed tests: 1 bare smoke + 1 JSON smoke + 1 table parent + 7 sub-cases = 10. Post-D4: 10 + 3 new TUI tests = 13, reconciling with the builder worklog's "10 + 3 = 13" claim. | PASS (268/268) |
| G | PLAN.md W2.D4 state `done` | `DROP_4c.6.W2_TILL_INIT/PLAN.md:128` — `**State:** done`. The W2 sub-plan only flipped the W2.D4 row in this round. | PASS |

### Cross-checks

- **Smoke test rewrite is semantics-preserving.** `TestInit_BareInvocation_ReturnsTUIStubError` (`init_cmd_test.go:30-57`) keeps its function name (per builder worklog L2103 — git-blame continuity) but the body now stubs `programFactory` with a `scriptedProgram` that returns an `initTUIModel` advanced to `Done` with synthetic payload `{Name:"stub", Group:"till-go", MCP:false}`. The `run(ctx, []string{"--app","tillsyn-init","init"}, ...)` end-to-end call is preserved — the CONSUMER-TIE TEST CONTRACT (W2-FF6 ROUND-2) from `DROP_4c.6.W2_TILL_INIT/PLAN.md:98` still holds because the route → cobra → `initCmd.RunE` → `runInitTUI` chain is still exercised. Only the terminal error shifts from the D3a literal (`"till init: TUI walk not yet wired (W2.D4)"`, now gone) to the D5-stub literal (`"till init: file copy not yet wired (W2.D5)"`). PASS.
- **Defense-in-depth on disabled row.** Two layers — cursor-skip via `nextEnabledGroupRow`/`prevEnabledGroupRow` + Enter-no-op via the `row.Disabled` check at `init_cmd.go:213-220`. Plan acceptance at W2 sub-plan L141 (`Greyed-out till-gdd option must be UNSELECTABLE — pressing enter on it should be a no-op or play a UI bell, not advance to next step`) is fully satisfied; the Enter-no-op layer is a forward-compat guard for future row additions.
- **Esc semantics consistent across both steps.** `init_cmd.go:184-186` (Name step) and `:203-205` (Group step) both set `step = initTUIStepCancelled` + `tea.Quit`. `TestRunInitTUI_EscCancelsWalk` (`:267-297`) drives Esc at the Name step; the Group-step Esc path is implementation-mirrored but not directly test-driven. Plan acceptance L134 only requires one cancel test; the Name-step exercise satisfies that.
- **Type-assertion safety.** `init_cmd.go:337-340` — `final, ok := finalModel.(initTUIModel); if !ok { return fmt.Errorf("till init: tui returned unexpected model type %T", finalModel) }`. Defensive but correct.
- **No tool / test bypasses.** Builder ran `mage testFunc` and `mage formatPath`; `mage install` was NOT invoked (per project rule); `mage ci` deferred to drop-orch (correct).
- **Out-of-scope diff verification.** `git diff --stat HEAD -- cmd/till/init_cmd.go cmd/till/init_cmd_test.go` reports `2 files changed, 464 insertions(+), 9 deletions(-)`. 271 LOC added to `init_cmd.go` matches the new walk surface (types + 6 methods + 2 helpers + rewritten `runInitTUI`); 202 LOC added to `init_cmd_test.go` matches the 3 new tea-tests + the smoke rewrite. Plan declared `~190` and `~150` — actual is within the order-of-magnitude expected for the model + teatest_v2 boilerplate. No drift outside declared paths.

### Findings

None blocking. One informational note:

- **W2-D4-PF1 (informational, NIT — symmetric Esc coverage):** Group-step Esc semantics (cancel from the group picker, not just the name step) are implemented at `init_cmd.go:203-205` but not exercised by a dedicated test. `TestRunInitTUI_EscCancelsWalk` only drives Esc at the Name step. The Group-step Esc code path is byte-identical to the Name-step path except for the entry-state constant, and the `final.Cancelled()` contract is the same, so this is a low-risk gap. Routed to the orchestrator as informational — future tightening could add a `TestRunInitTUI_EscDuringGroupPickerCancels` variant if the dev wants matching coverage. No action required from this builder.

### Missing evidence

None. All seven A-G check categories closed with direct file reads (`init_cmd.go`, `init_cmd_test.go`, `main.go`, `main_test.go`, `PLAN.md`, `BUILDER_WORKLOG.md`), `git grep` scans for the error string + `programFactory` seam, `git diff --stat` for delta scope, and two mage runs (full `test-pkg ./cmd/till` 268/268 + targeted `test-func` 4/4).

### Hylla Feedback

None — Hylla answered everything needed. No Hylla query was attempted in this QA review because all proof surfaces were either uncommitted changes verified via `git diff`/`Read`, or in-repo line-pinned references verified via `git grep`. The reviewed files were uncommitted at QA time so a Hylla query would have returned stale pre-D4 state for `init_cmd.go` anyway; the builder worklog L2151-2155 already records the relevant ingest-lag observation against `github.com/evanmschultz/tillsyn@main`.

### Summary

**PASS.**

D4's bubbletea walk fully replaces the D3a `runInitTUI` stub: 4-state closed `initTUIStep` enum, 2-step linear flow (Name → Group), 3-row group picker with `till-gdd` disabled + cursor-skip + Enter-no-op double-defense, `programFactory` seam reused (no new test substrate), 3 new tea-tests under the exact names the appendix called for, smoke-test rewrite preserving the CONSUMER-TIE TEST CONTRACT (W2-FF6), and D5-stub error byte-exact in both branches (TUI + JSON). PLAN.md state flipped `todo → done`. `mage test-pkg ./cmd/till` reports 268/268 GREEN (race detector on); the 4 newly-touched tests (3 tea + 1 rewritten smoke) all pass under targeted `mage test-func`. One informational NIT (W2-D4-PF1, missing symmetric Group-step Esc test) is routed to the orchestrator without action required.

D5 (file-copy pipeline) and downstream W2 droplets unblock on D4's `done` state per the W2 chain at `DROP_4c.6.W2_TILL_INIT/PLAN.md:162` (`D5 Blocked by: D1, D4`).
