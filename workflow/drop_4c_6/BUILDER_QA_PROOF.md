# Drop 4c.6 — Builder QA Proof

Per-droplet build-QA-proof rounds append below. Each round entry stamps droplet
ID, round number, findings, missing evidence, summary verdict, and Hylla
feedback.

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
