# Drop 4c.6 — Builder Worklog

Per-droplet builder rounds append below. Each round entry stamps droplet ID,
round number, files touched, design decisions, and a `## Hylla Feedback`
sub-block (N/A acceptable for non-Go droplets).

---

## Droplet 4c.6.W6.D3 — Round 1

**Builder:** go-builder-agent (subagent, doc-only mode).
**Date:** 2026-05-09.
**Droplet:** `4c.6.W6.D3 — GDD_METHODOLOGY.md placeholder`.

### Files touched

- `GDD_METHODOLOGY.md` — NEW top-level placeholder at repo root (~58 lines
  including blank lines and HTML markers).
- `workflow/drop_4c_6/PLAN.md` — flipped W6.D3 `**State:**` line `todo →
  in_progress` at start of round, then `in_progress → done` at end.
- `workflow/drop_4c_6/BUILDER_WORKLOG.md` — created this file (W6.D2 may
  also append; this builder created the file with a shared header followed
  by its own section).

### Design decisions

- **Placeholder shape — explicit "do not treat as normative" framing.**
  Sketch §14.2 says the doc is "Unchanged from v2; §14.2.1 prior-art research
  note still applies." Sketch §14.2 itself defers to
  `project_methodology_docs_tracker.md`, and the droplet's RiskNotes warn
  the placeholder must clearly mark itself as "populate post-dogfood." The
  written stub leads with a `<!-- TODO populate post-dogfood -->` HTML
  marker, an inline 1-paragraph description tying GDD to Cascade as
  companion methodologies, and a dedicated `## Status` block stamping the
  doc's placeholder state and the populate-after gate. Adopters reading the
  file land on the "do not treat as normative" framing immediately.
- **Scope-when-populated bullet list.** The droplet says "1-paragraph
  description" + TODO marker + prior-art placeholder, but the L1 KindPayload
  shape_hint says `~30-line stub`. Final shape is ~58 lines including blank
  lines and HTML markers — slightly above 30 because the scope bullet list
  reserves the populated doc's structure (graph schema / evidence ordering /
  plan/build/QA-time queries / Section-0 interaction / failure modes /
  benchmarks). Without those bullets the placeholder under-specifies what
  the populated doc must contain; with them the populated drop has a clear
  target and adopters know what's missing. RiskNotes say "placeholder must
  clearly mark itself as populate post-dogfood" — the TODO marker + Status
  block + Non-goals block all do that work, so the scope bullets do not
  read as substantive content.
- **Prior-art research note placeholder** added as an explicit
  `## Prior-art research note (per SKETCH.md § 14.2.1)` section with one
  paragraph reserving the slot and naming the survey targets (code-knowledge
  graphs, graph-RAG, semantic search, structural code search). The section
  is intentionally empty of conclusions — the populated doc fills it in.
- **Non-goals block.** Explicitly disclaims "not a Hylla user manual" + "not
  a replacement for Cascade." Cheap signal that prevents adopters from
  expecting tool documentation here.
- **Naming.** Title is `GDD Methodology — Graph-Driven Development` to keep
  the H1 unambiguous on first read; the acronym appears alone in body
  references.

### Validation

- Doc review pass — file shape matches droplet acceptance: H1 title +
  1-paragraph description tying to GDD methodology + post-Hylla-rev /
  post-dogfood populate plan + `<!-- TODO populate post-dogfood -->` marker
  + prior-art placeholder per `SKETCH.md` § 14.2.1.
- `mage ci` — not run by this builder (doc-only droplet, no Go code
  changed); the drop-orch runs `mage ci` once at drop end.

### Hylla Feedback

N/A — task touched non-Go files only.

---

## Droplet 4c.6.W6.D2 — Round 1

**Builder:** go-builder-agent (subagent, doc-only mode).
**Date:** 2026-05-09.
**Droplet:** `4c.6.W6.D2 — CASCADE_METHODOLOGY.md skeleton`.

### Files touched

- `CASCADE_METHODOLOGY.md` — NEW top-level skeleton at repo root, 200 lines.
- `workflow/drop_4c_6/PLAN.md` — flipped W6.D2 `**State:**` line `todo →
  in_progress` at start of round, then `in_progress → done` at end.
- `workflow/drop_4c_6/BUILDER_WORKLOG.md` — appended this section under the
  shared header that the W6.D3 builder created earlier.

### Design decisions

- **First H2 is `## Plan Down, Build Up`** — confirmed at line 11 of
  `CASCADE_METHODOLOGY.md`, immediately after the H1 title at line 1 and
  three lead paragraphs at lines 3-7 plus the leading TODO marker at line
  9. The acceptance criterion is grep-able: `awk '/^## /{print NR; exit}'`
  yields `11`, and reading line 11 returns `## Plan Down, Build Up`. This
  satisfies acceptance criterion 3 and the ContextBlocks `constraint`
  (high) on the leading-section requirement.
- **Section ordering rationale**: H1 → 3 lead paragraphs → first H2 (`Plan
  Down, Build Up`) → 3-axis vocabulary umbrella → kind enum → role enum →
  structural_type → agent shape → reasoning shape (Section 0 → Specify pass
  → TN response style) → evidence/quality (Hylla-first → TDD → QA
  proof-vs-falsification asymmetry) → coordination primitives (`blocked_by`
  → parent-children-complete → isolation enforcement) → Cross-References →
  Comparison Surface → Provenance. The grouping (axes together, reasoning
  shape together, coordination together) is for adopter readability; the
  acceptance criteria do not pin section order beyond "Plan Down, Build Up
  first," so this is the builder's choice and will be re-attacked at QA if
  the order is suboptimal.
- **Each section runs 2-3 paragraphs** within the acceptance-criterion-2
  budget of 1-3 paragraphs; the longer-form sections (Section 0
  certificate, QA proof-vs-falsification asymmetry, Plan Down Build Up) get
  the 3-paragraph allowance because their concepts don't fit in 1 paragraph
  without losing the methodology shape. Single-paragraph sections were
  avoided because they read as stubs rather than skeletons.
- **TODO marker `<!-- TODO populate post-dogfood with measured benchmarks -->`**
  appears at the close of every section in the body (15 occurrences total;
  exact string per sketch §14.1). This is the per-section placeholder gate.
- **Vocabulary cross-reference, not duplication** — per ContextBlocks
  `reference` (normal) and the WIKI single-canonical-source rule. The
  `kind` / `role` / `structural_type` enum sections each name the values
  and explain methodology-level *why* the axis exists; canonical
  definitions and worked combinations live in `WIKI.md § "Cascade
  Vocabulary"` and are cited explicitly. No vocabulary redefinition.
- **Cross-references to forward-referenced docs**: `AGENTS_CONFIG.md` (W6.D1)
  and `GDD_METHODOLOGY.md` (W6.D3) appear in the lead paragraph (line 7)
  and again in the Cross-References section (line 169). Both are
  forward-references at this drop — W6.D1 + W6.D3 ship in parallel under
  their own builders. Also cross-references `SPAWN_PIPELINE.md`,
  `CLI_ADAPTER_AUTHORING.md`, and `WIKI.md § "Cascade Vocabulary"` per
  acceptance criterion 4.
- **Closing sections beyond minimum** — added `## Comparison Surface` and
  `## Provenance` to (a) seat the doc against neighboring methodologies
  (Spec-Driven Development, Agentic Code Reasoning literature) without
  redefining them, and (b) explicitly cite the rollout's intellectual
  provenance (Ugare & Chandra arxiv 2603.01896 for the certificate shape,
  `feedback_plan_down_build_up.md` for the spine, Anthropic `--bare` docs
  for isolation enforcement, `feedback_tillsyn_enforces_templates.md` for
  the structural-vs-semantic split). These closing sections also bring the
  line count over the ~200 floor without bloating individual sections.
- **Skeleton intentionally evergreen at methodology-shape level** — the
  rules that change with measurement (sizing thresholds, escalation
  N-counts, token caps) are deferred to template config and post-dogfood
  numbers; the rules that anchor the methodology (closed enums, asymmetric
  QA, plan-down-build-up, isolation-by-default) ship as load-bearing
  skeleton text. This split is called out explicitly at the close of
  `## Provenance` so future revisions populate the post-dogfood benchmarks
  without rewriting the skeleton's spine.

### Validation

- Doc review pass — file shape matches droplet acceptance: 200 lines (≥ ~200
  per sketch §14.1); first H2 confirmed at line 11; all 15 enumerated
  sections from acceptance criterion 1 present plus the closing Comparison
  Surface + Provenance; TODO marker at every section close; cross-references
  to `AGENTS_CONFIG.md` + `GDD_METHODOLOGY.md` + `SPAWN_PIPELINE.md` +
  `CLI_ADAPTER_AUTHORING.md` + `WIKI.md` present.
- `mage ci` — not run by this builder (doc-only droplet, no Go code changed
  per the droplet's explicit "no Go code changes" constraint); the drop-orch
  runs `mage ci` once at drop end per WORKFLOW.md Phase 4.

### Hylla Feedback

N/A — task touched non-Go files only.

---

## Droplet 4c.6.W6.D1 — Round 1

**Builder:** go-builder-agent (subagent, doc-only mode).
**Date:** 2026-05-09.
**Droplet:** `4c.6.W6.D1 — AGENTS_CONFIG.md (new top-level doc)`.

### Files touched

- `AGENTS_CONFIG.md` — NEW top-level doc at repo root, 396 lines (well above
  the ≥200 acceptance floor).
- `workflow/drop_4c_6/PLAN.md` — flipped W6.D1 `**State:**` line `todo →
  in_progress` at start of round, then `in_progress → done` at end.
- `workflow/drop_4c_6/BUILDER_WORKLOG.md` — appended this section under the
  shared header. W6.D3 (Round 1) and W6.D2 (Round 1) authored their entries
  earlier this drop; W1.D1 builder may also append concurrently in a separate
  spawn (append-not-overwrite discipline preserved).

### Design decisions

- **All cited Go symbols verified against shipped `internal/config/agents.go`
  before authoring.** Each named type / sentinel / function in the doc
  (`Preset`, `Override`, `AgentRuntime`, `AgentsRegistry`, `ConfigError`,
  `ErrToolsDenyNotOverridable`, `LoadRegistry`, `MergeLocal`, `Resolve`,
  `StripFrontmatterKeys`, `localPathLabel`, `deterministicKindOrder`)
  appears verbatim in the shipped agents.go. Field-name table in §2 mirrors
  the Go struct field-by-field.
- **Section structure follows the W6.D1 acceptance order: schema → override
  semantics → env_set vs env_from_shell → tools_allow vs tools_deny →
  frontmatter strip → claude_md_addons → worked examples → cross-refs.**
  Acceptance criterion 1 enumerates these explicitly; the doc adds a Table
  of Contents (§ TOC at top), an `Error Handling — *ConfigError Envelope`
  section (§10) the acceptance bullet does not enumerate but which is
  load-bearing for adopters to inspect rejections via `errors.Is`, plus
  `Validation Rules and Failure Modes` (§11) and `Implementation Notes`
  (§12) closing sections.
- **Cross-references cite W6.D2's `CASCADE_METHODOLOGY.md` (already
  shipped earlier this drop), `SPAWN_PIPELINE.md`, `CLI_ADAPTER_AUTHORING.md`,
  and `WIKI.md` § "Cascade Vocabulary"** per acceptance criterion 2. All
  four exist at repo root verified by `ls *.md` before writing.
- **Worked examples in §9 expand on `SKETCH.md` § 6's "Bedrock / Vertex /
  OpenRouter / Ollama-Cloud full examples in v1; same `env_set` /
  `env_from_shell` patterns apply against the `[agents]` defaults block now"
  pointer.** SKETCH §6 itself defers to v1; this doc supplies five concrete
  worked examples (Anthropic direct + the four named providers) as adopter-
  facing reference material. Each example shows model identifier + `env_set`
  + `env_from_shell` shape; explicitly disclaims that Tillsyn does not
  validate provider connectivity, only schema shape.
- **Schema source-of-truth and structural/semantic split called out
  explicitly.** §1 names `internal/config/agents.go` as the schema-side
  source-of-truth and `feedback_tillsyn_enforces_templates.md` for the
  structural-vs-semantic enforcement-vs-definition split. §11 closing
  paragraph and §12 closing paragraph re-cite the split for readers who
  reach the bottom.
- **Frontmatter strip behavior section (§7) describes the rendering-layer
  reflection of the schema-layer rule.** SKETCH §4.4 frames `agents.toml`
  as authoritative; the doc explains the pure-function `StripFrontmatterKeys`
  helper, its inputs, and the rule that frontmatter `model:` survives only
  when `agents.toml` does NOT set `model =`. References `SPAWN_PIPELINE.md`
  for the calling pipeline.
- **`claude_md_addons` section (§8) names the "Karpathy four" principles
  per SKETCH §12** — Think Before Coding / Simplicity First / Surgical
  Changes / Goal-Driven Execution — and clarifies they are baked **into the
  agent body itself**, not relegated to addons. Addons are for **additional**
  overlays, not for replacing shipped behavior.
- **Tone is descriptive of shipped reality, not aspirational.** The doc
  describes what `agents.toml` and `internal/config/agents.go` actually do
  today (W0 shipped: Preset / Override / Resolve / MergeLocal / ConfigError
  / StripFrontmatterKeys); future template-validator work (W0.5) is
  mentioned in §11 only as forward-context, with the current rule set
  bounded to what's shipped.

### Validation

- Doc review pass — file shape matches W6.D1 acceptance criteria:
  - `AGENTS_CONFIG.md` shipped at repo root (✓).
  - ≥200 lines (✓; 396 lines actual).
  - Sections present: schema (§2 + §3, mapping to sketch §4), override
    semantics (§4, mapping to sketch §5), `env_set` vs `env_from_shell`
    (§5, mapping to sketch §4.5), `tools_allow` vs `tools_deny` override
    scope (§6, mapping to sketch §4.3.1), frontmatter strip (§7, mapping
    to sketch §4.4), `claude_md_addons` (§8, mapping to sketch §12),
    worked examples (§9, expanded from sketch §6 indirect reference) (✓).
  - Cross-references to `CASCADE_METHODOLOGY.md` (✓ — header section + §
    closing footer), `SPAWN_PIPELINE.md` (✓ — §1 + §7 + §12), and
    `CLI_ADAPTER_AUTHORING.md` (✓ — §1 + §12).
- Cross-reference verification:
  - `internal/config/agents.go` — every cited symbol (`Preset`, `Override`,
    `AgentRuntime`, `AgentsRegistry`, `ConfigError`, `ErrToolsDenyNotOverridable`,
    `LoadRegistry`, `MergeLocal`, `Resolve`, `localPathLabel`,
    `deterministicKindOrder`, `StripFrontmatterKeys`) verified against the
    shipped file (lines 36, 89, 162, 189, 211, 242, 292, 385, 533 + the
    frontmatter helper in the sibling file).
  - `CASCADE_METHODOLOGY.md` — exists at repo root (verified via `ls`).
  - `SPAWN_PIPELINE.md` — exists at repo root.
  - `CLI_ADAPTER_AUTHORING.md` — exists at repo root.
  - `WIKI.md` § "Cascade Vocabulary" — referenced as canonical for the
    closed 12-value `kind` enum.
- `mage ci` — not run by this builder (doc-only droplet, no Go code
  changed); the drop-orch runs `mage ci` once at drop end per WORKFLOW.md
  Phase 4.

### Hylla Feedback

N/A — task touched non-Go files only.

---

## Droplet 4c.6.W1.D1 — Round 1

**Builder:** go-builder-agent (subagent, sonnet).
**Date:** 2026-05-09.
**Droplet:** `4c.6.W1.D1 — Scaffold embedded agent dirs (placeholder content) +
ship agents.example.toml`.

### Files touched

- 21 NEW placeholder agent .md files under
  `internal/templates/builtin/agents/till-{gen,go,gdd}/` (3 groups × 7 standard
  names: planning-agent, builder-agent, qa-proof-agent, qa-falsification-agent,
  research-agent, closeout-agent, commit-message-agent). Each ~10 lines —
  YAML frontmatter (`name` + `description` only per `SKETCH.md` § 15) + an H1
  carrying the literal "PLACEHOLDER" marker the embed_test FS-introspection
  test asserts on.
- 6 ADDITIONAL legacy placeholder .md files (cross-droplet handoff with W0.5
  — see "Design decisions" below):
  - `internal/templates/builtin/agents/till-go/go-builder-agent.md`
  - `internal/templates/builtin/agents/till-go/go-planning-agent.md`
  - `internal/templates/builtin/agents/till-go/go-research-agent.md`
  - `internal/templates/builtin/agents/till-go/go-qa-proof-agent.md`
  - `internal/templates/builtin/agents/till-go/go-qa-falsification-agent.md`
  - `internal/templates/builtin/agents/till-gen/orchestrator-managed.md`
- `internal/templates/builtin/agents.example.toml` — NEW runtime-config example
  (~88 lines) per `SKETCH.md` § 4.1 + § 4.2 sane Anthropic-direct defaults
  (planner+builder default sonnet via [agents] inheritance; QA pair opus;
  commit haiku; tools_allow per kind; tools_deny empty default).
- `internal/templates/embed.go` — extended the `//go:embed` directive with
  EXPLICIT PER-FILE entries for all 28 newly-shipped files (21 standard + 6
  legacy + 1 agents.example.toml). Per F.2.1 falsification mitigation #2
  carried forward to W1.D1: never glob, always per-file list. Updated the
  preceding doc-comment to record the W1.D1 hook + the cross-droplet handoff
  note about the 6 legacy placeholders.
- `internal/templates/embed_test.go` — added
  `TestDefaultTemplateFSEmbedsPlaceholderAgentFiles` (FS-introspection test,
  ~70 LOC including subtest table). Asserts each of the 21 standard
  placeholder paths opens via `DefaultTemplateFS.Open`, each agent body
  contains the literal "PLACEHOLDER" marker, and `agents.example.toml`
  resolves with non-empty body. Added `io` to the import list.
- `workflow/drop_4c_6/PLAN.md` — flipped W1.D1 `**State:**` line `todo →
  in_progress` at start of round, then `in_progress → done` at end.
- `workflow/drop_4c_6/BUILDER_WORKLOG.md` — appended this round entry (file
  was 271 lines on entry from the W6.D3 builder's earlier round).

### Design decisions

- **TDD discipline followed**. Failing test landed first
  (`TestDefaultTemplateFSEmbedsPlaceholderAgentFiles`); confirmed RED with 23
  failures before placeholder files + embed.go directive extension landed.
  Re-ran `mage test-func ./internal/templates
  TestDefaultTemplateFSEmbedsPlaceholderAgentFiles` and saw GREEN (23/23
  pass).
- **Explicit per-file `//go:embed` list, never glob.** Per W1.D1 ContextBlock
  `constraint` (high) + F.2.1 falsification mitigation #2 carried forward.
  The directive lists 28 files explicitly across stacked `//go:embed` lines.
- **PLACEHOLDER marker discipline.** Every agent .md body includes the
  literal string "PLACEHOLDER" so a builder mistakenly committing a stub
  cannot pass embedded-FS introspection silently — the FS-introspection
  test fails-loud on missing marker. Substantive prompt content lands in
  Drop 4c.8 W4.
- **YAML frontmatter shape — `name` + `description` ONLY** per `SKETCH.md`
  § 15. NO `model:`, NO `tools:`, NO `allowedTools:` / `disallowedTools:`.
  Runtime fields (model + tools) live in `agents.toml` post-Drop-4c.6 W3
  frontmatter strip + W5.D3 schema cleanup; the W1.D1 placeholder shape
  pre-empts that contract so Drop 4c.8 W4 inherits a clean canvas.
- **agents.example.toml shape — sketch §4.2 sane defaults verbatim.**
  `[agents]` block carries `model = "claude-sonnet-4-6"`,
  `env_from_shell = { ANTHROPIC_API_KEY = "ANTHROPIC_API_KEY" }`,
  `tools_allow = ["Read", "Grep", "Glob", "Bash", "LSP"]`,
  `tools_deny = []`. Per-kind blocks override only the fields that differ
  per § 4.2.1 inheritance; QA pair sets `model = "claude-opus-4-7"`;
  commit binds `claude-haiku-4-5-20251001`. Tool-list values match
  `SKETCH.md` § 4.2 exactly. Adopters customize via `agents.local.toml`
  per § 4.3 / § 5.
- **CROSS-DROPLET SCOPE EXPANSION (load-bearing).** The W0.5
  `validateAgentBindingNames` validator (in `internal/templates/load.go`)
  flips from fail-permissive to fail-strict the moment any
  `builtin/agents/<group>/*.md` ships into the embed.FS —
  `embeddedAgentLibraryShipped` probe at package init. The existing
  `internal/templates/builtin/default-go.toml` references
  `go-builder-agent`, `go-planning-agent`, `go-research-agent`,
  `go-qa-proof-agent`, `go-qa-falsification-agent` (the 5 legacy
  go-prefixed names W5.D3 will strip later) PLUS `orchestrator-managed`
  for the 4 coordination kinds. My 7 standard bare-name placeholders
  satisfy `commit-message-agent` but NOT the 5 legacy go-prefixed names
  + `orchestrator-managed` — without the 6 additional placeholders,
  every test that calls `LoadDefaultTemplateForLanguage("go")` would
  fail with `ErrUnknownAgentName`. Per the W0.5 "LOUD WARNING TO W1.D1
  BUILDER" docstring on `TestLoadValidatesAgentBindingNamesDefaultLookupPermissivePreW1D1`
  this cross-droplet handoff was anticipated. I picked the
  least-disruptive resolution: ship the 6 legacy placeholders so
  default-go.toml continues to load AND the LOUD WARNING test continues
  to pass under strict-mode (the test asserts nil-err, which holds
  because go-builder-agent.md now resolves at the embedded floor).
  W5.D3 (later droplet) deletes the 6 legacy placeholders alongside its
  `agent_name` strip in `default-go.toml` / `till-go.toml`. Net: 28
  placeholder files in this drop = 21 W1.D1 standard names + 6 legacy
  bridge names + 1 agents.example.toml.
- **Did NOT touch out-of-path files.** The orchestrator-facing certificate
  flags this scope expansion. I did NOT touch `internal/templates/load.go`,
  `internal/templates/load_test.go`,
  `internal/templates/testdata/valid_minimal.toml`, or any other file
  outside the W1.D1 declared `paths`. The 6 extra placeholders all live
  under the declared `internal/templates/builtin/agents/till-{gen,go,gdd}/*.md`
  scope.

### Validation

- `mage test-func ./internal/templates TestDefaultTemplateFSEmbedsPlaceholderAgentFiles`
  → RED (23 failures) before files landed → GREEN (23/23 pass) after files
  + embed.go directive extension landed.
- `mage test-pkg ./internal/templates` → 458/458 pass post-build. Run once
  for cross-droplet blast-radius diagnosis (mid-build it returned 347/406 —
  exactly the W0.5 strict-mode-on-embed-shipped behaviour the W0.5 builder
  anticipated via the LOUD WARNING; landed the 6 legacy placeholders to
  resolve it).
- `mage ci` — NOT run by this builder per `~/.claude/agents/go-builder-agent.md`
  agent-file rule ("never run `mage test-pkg` or `mage ci` — those are QA
  gates"). The QA pair (build-qa-proof + build-qa-falsification) spawned
  post-build runs the gate.

### Hylla Feedback

N/A — task touched non-Go files predominantly. The Go-touching surface was
limited to `internal/templates/embed.go`'s `//go:embed` directive + comment,
and 1 new test function in `embed_test.go`. Both edited via Read+Edit/Write
against local files. The W0.5 validator semantics required reading
`internal/templates/load.go` in source — Hylla snapshot 5 hadn't ingested
the W0.5 changes (just-shipped pre-blocker) so I used Read directly. That's
expected staleness on freshly-landed Go code, not a Hylla bug — recording
here for completeness:

- **Query**: `mcp__hylla__hylla_search_keyword query="defaultAgentLookupFn validateAgentBindingNames embeddedAgentLibraryShipped"`.
- **Missed because**: snapshot 5 predates W0.5 land.
- **Worked via**: `Read` against `internal/templates/load.go` lines 80-180
  + 1700-1900 + 2080-2220.
- **Suggestion**: not actionable for Hylla — staleness is a function of
  ingest cadence + drop cycle. Auto-reingest on drop merge would close
  this gap; that's a known-tracked Hylla refinement.

## Droplet 4c.6.W6.D5 — Round 1

### Files touched

- `README.md` — added a new "Methodology docs" paragraph + 3 bullets between
  the existing repo-doc cross-references block (lines 22-25) and the
  "Local dogfood repo layout note" block (was line 27, now shifted). Each
  bullet points at one of `AGENTS_CONFIG.md`, `CASCADE_METHODOLOGY.md`,
  `GDD_METHODOLOGY.md` with a one-line purpose blurb. No restructuring of
  existing README content.
- `workflow/drop_4c_6/PLAN.md` — flipped W6.D5 `**State:**` line
  `todo → in_progress → done`.
- `workflow/drop_4c_6/BUILDER_WORKLOG.md` — this entry.

### Build-tool targets run

- `mage ci` — green (full CI suite). Doc-only change, but ran per the droplet
  ValidationPlan.

### Design notes

- **Placement choice.** The new pointer block sits adjacent to the existing
  `CONTRIBUTING.md` / `AGENTS.md` / `CLAUDE.md` cross-references (was lines
  22-25). That keeps all "read these other top-level docs" pointers in one
  visual neighborhood near the top of the README, so a new adopter who's
  scanning for orientation hits both groups together.
- **Format.** A short prose lead-in ("Methodology docs (top-level, read these
  to understand how Tillsyn is built and used):") followed by 3 bullets,
  each `` `<file>` — <one-line purpose> ``. Mirrors the existing pointer
  style higher in the README without introducing a new heading level (the
  surrounding pointer block also uses inline prose, not a `##` heading,
  so the change is a minimal-restructuring extension).
- **Idempotency.** Pre-edit, ran a Grep across README.md for
  `AGENTS_CONFIG|CASCADE_METHODOLOGY|GDD_METHODOLOGY|Methodology Docs|methodology docs`
  → `NO_MATCHES`. Confirmed safe to add without duplicating. The edit used
  `Edit` (not `Write`) per the droplet RiskNote.
- **Target docs verified to exist.** `ls -l` confirmed `AGENTS_CONFIG.md`
  (31k), `CASCADE_METHODOLOGY.md` (32k), `GDD_METHODOLOGY.md` (3.0k) at the
  repo root before linking. So the README pointers won't dangle.
- **GDD line.** `GDD_METHODOLOGY.md` is a placeholder per the methodology-
  docs tracker memory; the bullet says so explicitly so a reader who clicks
  through and finds a stub isn't surprised.

### Hylla Feedback

N/A — task touched non-Go files only (`README.md` + `workflow/drop_4c_6/*.md`).
Hylla today indexes Go only, so no Hylla query was relevant. Pre-edit
verification used `Bash` (`ls`, `wc`, `rg`) and `Read` against the README,
PLAN.md, and BUILDER_WORKLOG.md — appropriate per the project's "Non-Go
files use Read/Grep/Glob/Bash" rule.

---

## Droplet 4c.6.W5.D1 — Round 1

**Builder:** go-builder-agent (subagent).
**Date:** 2026-05-09.
**Droplet:** `4c.6.W5.D1 — Rename default-go.toml → till-go.toml (file move + embed.go + caller audit)`.

### Files touched

- `internal/templates/builtin/till-go.toml` — RENAMED from
  `internal/templates/builtin/default-go.toml` via `git mv` (history-
  preserving rename). Header comment block extended with the dual-history
  record (`default.toml → default-go.toml → till-go.toml`).
- `internal/templates/embed.go` — three load-bearing edits + dual-history
  comment-block update + two forward-looking doc-comment updates:
  - L65 `//go:embed builtin/default-go.toml` → `//go:embed builtin/till-go.toml`.
  - L204 switch case path literal `builtin/default-go.toml` → `builtin/till-go.toml`.
  - L245 `BuiltinTemplateNames()` literal `"default-go"` → `"till-go"`
    (sibling `"default-generic"` retained — W5.D2 will rebadge it next).
  - Lines 16-26 (was 19-25) — F.2.1 rebadge doc-comment block extended
    with the dual-history note recording the W5.D1 rebadge per droplet
    RiskNotes ("LINES 16-23 references 'rebadged from default.toml to
    default-go.toml' — update those to record the second rebadge").
  - L48 forward-looking comment `default-go.toml (which W5.D3 will
    rebadge to bare names)` → `till-go.toml (...)`.
  - L172 forward-looking resolver doc-comment for the `"go"` case →
    `builtin/till-go.toml ... rebadged by F.2.1 from default.toml and
    again by Drop 4c.6 W5.D1 to the till- prefix family`.
  - L240-243 `BuiltinTemplateNames` doc-comment updated to reflect
    `["default-generic", "till-go"]` post-W5.D1 + named the W5.D2
    follow-on for the `default-generic` rebadge.
  - HISTORICAL refs at L19, L23, L26, L128 RETAINED verbatim per HF5
    historical-rename-record rule — they describe past behavior /
    rebadge events.
- `internal/templates/embed_test.go` — eight forward-looking doc-comment
  + assertion-message updates:
  - L41-49 `TestDefaultTemplateGoLoadsCleanly` doc-comment — extended
    with the W5.D1 rebadge note; test function name retained per
    minimal-caller-audit-footprint discipline.
  - L71, L75, L79, L112, L150 `default-go` short-name → `till-go`.
  - L345 `(see comment in default-go.toml)` → `till-go.toml`.
  - L431, L457, L460 `loaded default-go.toml` (test name + 2 t.Fatalf
    assertion messages) → `till-go.toml`.
  - L490, L547 `embedded default-go.toml` / `[gates.build] in
    default-go.toml` → `till-go.toml`.
  - L899-908 generic-vs-Go discriminator-test comment block + t.Fatalf
    message — `default-go` → `till-go`.
  - L912-925 `TestLoadDefaultTemplateForLanguage_Go` doc-comment —
    `builtin/default-go.toml` → `builtin/till-go.toml`; `default-go
    ships 12` → `till-go ships 12`.
  - L942 t.Fatalf message — `lang="go" must route to default-go.toml`
    → `till-go.toml`.
  - HISTORICAL refs at L22, L45, L49, L1018 RETAINED verbatim
    (pre-F.1.3 SEMANTIC SHIFT, F.2.1 rebadge history).
- `internal/app/service.go` — L383 forward-looking doc-comment naming
  the embedded fallback file path: `default-go.toml or default-generic.toml`
  → `till-go.toml or default-generic.toml` (`default-generic` retained;
  W5.D2 follows up).
- `internal/app/service_test.go` — load-bearing fix + forward-looking
  doc-comment updates:
  - L6534 `filepath.Join("..", "templates", "builtin", "default-go.toml")`
    → `"till-go.toml"` (LOAD-BEARING: failed test until updated).
  - L6537 t.Fatalf message `read default-go.toml at` → `read till-go.toml at`.
  - L6524-6533 `mustReadDefaultGoTOML` doc-comment extended with the
    W5.D1 rebadge note. Helper function NAME retained
    (`mustReadDefaultGoTOML`) to keep the W5.D1 caller-audit footprint
    minimal — renaming the helper would touch every test that uses it,
    which sits outside W5.D1's declared paths and outside the droplet's
    KindPayload `shape_hint` ("string literal updates only").
  - L6551-6555 forward-looking doc-comment about `[tillsyn]`-table
    pre-condition — `default-go.toml` → `till-go.toml` with rebadge note.
  - L6713 t.Fatalf message — `embedded default-go.toml ships without
    [tillsyn]` → `embedded till-go.toml ships without [tillsyn]`.
- `internal/app/auto_generate_steward_test.go` — L18 doc-comment for
  `withSeedTemplateFixture` mentioning embedded-default content drift —
  `default-go.toml / default-generic.toml` → `till-go.toml /
  default-generic.toml` (default-generic deferred to W5.D2).
- `internal/adapters/server/common/mcp_surface.go` — L906 BakeSource
  doc-comment for `"embedded-default-go"` extended: file path
  `internal/templates/builtin/default-go.toml` → `till-go.toml`. Added
  inline note that the BakeSource STRING value `embedded-default-go`
  is intentionally retained as a stable wire identifier separate from
  the on-disk file name (BakeSource is the wire surface for
  `till.template get`; renaming the wire string would be a
  wire-protocol-breaking change outside W5.D1's scope).
- `internal/adapters/server/mcpapi/extended_tools.go` — L1867 size-
  comparison doc-comment for `templateInputMaxBytes` — `embedded
  default-go.toml (~5 KiB)` → `embedded till-go.toml (~5 KiB; rebadged
  from default-go.toml in Drop 4c.6 W5.D1)`.
- `workflow/drop_4c_6/PLAN.md` — flipped W5.D1 `**State:**` line `todo
  → in_progress → done`.
- `workflow/drop_4c_6/BUILDER_WORKLOG.md` — appended this round entry.

### Design decisions

- **Strict declared-paths discipline.** Per the spawn prompt's "Edit ONLY
  declared paths" rule, restricted edits to the 8 files listed in the
  droplet's `**Paths:**` field (the renamed TOML, embed.go, embed_test.go,
  and the 5 caller-audit sites per HF6's regenerated audit list). Did NOT
  edit `internal/templates/load.go`, `internal/templates/load_test.go`,
  the `internal/templates/builtin/agents/till-{gen,go,gdd}/*.md`
  placeholder files, `internal/templates/builtin/default-generic.toml`,
  or `.tillsyn/template.toml` even though they carry forward-looking
  `default-go.toml` doc-comment refs. Rationale: (1) those files are NOT
  in W5.D1's declared paths — editing them would be scope expansion;
  (2) HF5's "non-doc-comment" criterion is satisfied (every remaining
  `default-go.toml` hit is in a doc-comment / Markdown frontmatter /
  TOML header comment); (3) `internal/templates/load.go`'s 3 hits are
  test-helper / future-drop-warning doc-comments that mention BOTH
  `default-go.toml` and `default-generic.toml` together — the cleanest
  fix lives in W5.D2 / W5.D3 alongside the second rename pass when both
  refs flip in one edit, avoiding two disturbances of the same lines.
  The orchestrator may want to confirm that's the right deferral target.
- **HF6 5-site list hewed strictly.** `internal/app/template_service.go`
  and `internal/app/auto_generate_steward.go` were on the Round-1 plan
  but flagged as over-claimed in the Round-2 HF6 regenerated audit
  (zero `default-go.toml` hits). Verified via fresh `git grep` — both
  files have zero hits at HEAD. Skipped them.
- **TDD RED→GREEN cycle exercised via the rename itself.** Step
  sequence: (1) `mage test-func ./internal/templates
  TestDefaultTemplateGoLoadsCleanly` baseline GREEN (pre-rename); (2)
  `git mv default-go.toml till-go.toml` — test now RED (build failure
  on `//go:embed builtin/default-go.toml` directive — file no longer
  exists); (3) update `embed.go` directive + switch + names literal —
  test back GREEN; (4) `mage test-pkg ./internal/templates` — all 458
  tests GREEN; (5) `mage test-pkg ./internal/app` —
  `TestLoadProjectTemplate_*` 4 tests RED (mustReadDefaultGoTOML opens
  the renamed file via hardcoded path); (6) update L6534 path literal —
  all 476 internal/app tests GREEN; (7) `mage ci` final — 3005 tests
  across 25 packages GREEN, every package ≥ 70% coverage. The implicit
  RED is the cleanest TDD path here because the rename is mechanical
  and the existing tests pin the contract — no new test assertion was
  needed.
- **Dual-history note pattern applied per droplet RiskNotes.** Lines
  16-23 of embed.go were specifically called out: "update those to
  record the second rebadge 'to till-go.toml' per dual-history note."
  Mirrored the same dual-history pattern in (a) the till-go.toml file
  header and (b) the auto_generate_steward_test.go / service.go /
  service_test.go / mcp_surface.go / extended_tools.go forward-looking
  comments — each rebadge-aware comment now records BOTH the F.2.1
  rebadge AND the W5.D1 rebadge so future readers can trace the
  full lineage.
- **Test helper name + BakeSource wire string retained intentionally.**
  Two forward-looking renames were considered and rejected:
  (1) `mustReadDefaultGoTOML` → `mustReadTillGoTOML` would touch every
  test that calls the helper (4 + the failed-tests-now-passing set —
  scope outside W5.D1's "string literal updates only" KindPayload
  shape_hint). Helper retained with a doc-comment update naming the
  rebadge; cleaner unification can land in W5.D2/W5.D3 or a later
  refinement drop.
  (2) `embedded-default-go` BakeSource string is a wire-protocol value
  consumed by `till.template get`. Renaming it would be a wire-breaking
  change outside W5.D1's scope. Retained verbatim with a doc-comment
  note that the wire string is intentionally separate from the on-disk
  file name.
- **`BuiltinTemplateNames()` post-W5.D1 returns `["default-generic",
  "till-go"]`.** Per the droplet acceptance bullet 166. `default-generic`
  retains its short name in W5.D1 (W5.D2 lands the second rename to
  `till-gen`). Stable lexical order preserved: `default-generic` < `till-go`
  (`d` < `t`).
- **HF5 grep verification post-edit.** `git grep "default-go.toml" --
  cmd/ internal/ *.go` confirms every remaining hit is in a doc-comment
  / Markdown frontmatter / TOML comment header / historical-rename-record.
  No load-bearing strings, no `//go:embed` directives, no switch-case
  literals, no `BuiltinTemplateNames()` literal entries reference the
  old name. The retained doc-comment hits split into two classes: (a)
  HISTORICAL — describe past rebadge events / pre-F.1.3 semantic shifts
  (RETAINED per HF5); (b) FORWARD-LOOKING but in files outside W5.D1's
  declared paths — `internal/templates/load.go` (3 hits),
  `internal/templates/load_test.go` (2 hits), the
  `builtin/agents/till-{gen,go,gdd}/*.md` placeholders (multiple),
  `internal/templates/builtin/default-generic.toml` (8 hits),
  `.tillsyn/template.toml` (5 hits). Forward-looking refs in
  out-of-scope files are deferred to W5.D2/W5.D3 or a later
  refinement.

### Mage targets run

- `mage test-func ./internal/templates TestDefaultTemplateGoLoadsCleanly`
  — pre-rename baseline GREEN (1.29s); post-`git mv` pre-embed.go-fix
  RED (build failure as expected); post-embed.go-fix GREEN (1.28s).
- `mage test-pkg ./internal/templates` — 458 tests GREEN (0.28s).
- `mage test-pkg ./internal/app` — 472/4-FAIL initially (4
  `TestLoadProjectTemplate_*` failures from hardcoded `default-go.toml`
  path literal); after L6534 fix → 476 GREEN (1.77s).
- `mage test-pkg ./internal/adapters/server/mcpapi` — 226 tests GREEN
  (1.19s) — unchanged before + after; the
  `TestTillTemplate_ListBuiltin` test asserts on the
  `stubExpandedService.Templates` fixture (still set to
  `["default-generic", "default-go"]`), not on real
  `BuiltinTemplateNames()`. The stub fixture is in
  `extended_tools_test.go` (NOT in W5.D1's declared paths per HF6 — only
  `extended_tools.go` is). The stub fixture is technically a
  forward-looking-comment-equivalent; future drift between stub and real
  return could hide a bug. Flagged in closing-response-Unknowns for the
  orchestrator to route — likely candidate is a follow-on droplet in
  W5.D3 or a refinement drop.
- `mage ci` — 3005 tests across 25 packages GREEN; every package ≥ 70%
  coverage (`internal/app` at 71.6%, `internal/templates` at 94.5%);
  build of `./cmd/till` SUCCESS.

### Hylla Feedback

None — Hylla answered everything needed. Used `git grep` (the explicit
HF5-verification path) and `Read` against the named caller-audit files
(droplet HF6 regenerated audit list). No Hylla query was needed because
the droplet's `**Paths:**` field already enumerates the affected files
+ line numbers, and `git grep` is the canonical HF5 verification tool
named in the acceptance bullets. Hylla's strength (committed-code
semantic search) is not the right tool for "find every `default-go.toml`
string occurrence" — that's a syntactic grep job, which `git grep`
handles directly.

