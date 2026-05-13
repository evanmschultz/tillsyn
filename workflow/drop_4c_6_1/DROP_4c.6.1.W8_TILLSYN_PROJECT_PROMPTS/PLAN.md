# DROP_4c.6.1.W8 — Tillsyn-Project-Local Agent Prompts + Bindings + Gitignore

**State:** todo (Wave A; no L1 blockers; prompt droplets D0-D20 unblock D21 after W1 also lands)
**Kind:** plan (L2 sub-drop container)
**Wave:** A (D0-D20), C (D21 — blocked by D0-D20 + 4c.6.1.W1)
**Directory:** `workflow/drop_4c_6_1/DROP_4c.6.1.W8_TILLSYN_PROJECT_PROMPTS/`
**Source-of-truth scope:** REVISION_BRIEF §2.18 (prompts), §2.19 (bindings.json), §2.20 (.gitignore); SKETCH §10 Tillsyn-project-local row; L1 PLAN.md lines 706-846.
**PLAN-QA-DISCIPLINE-R2:** This plan declares exactly 22 droplets (D0 + D1-D20 + D21). Count: 1 + 20 + 1 = 22. Enumerated D-list below; count must match.

## Round 2 Changes (round-2 L2 planner absorption — 2026-05-12)

Round-1 plan-QA returned PASS-WITH-NITS (proof: 2 FFs + 4 NITs) + PASS-WITH-ABSORB (fals: 2 FFs + 6 NITs). R10-D4 (bare-alias model frontmatter) also applies. Absorbed in this round:

- **Proof FF1.1 / Fals FF2 ABSORB (model values)**: Updated model assignments table + all per-droplet AcceptanceCriteria `model:` bullets to bare aliases per R10-D4: `model: sonnet` (D2/D12 builder), `model: opus` (D1/D3/D4/D5/D6/D7/D11/D13/D14/D15/D16/D17 planning/qa-*/research), `model: haiku` (D9/D19 commit-message), `model: orchestrator-managed` (D8/D10/D18/D20 closeout/orchestrator-managed kinds). Matches live `~/.claude/agents/go-*-agent.md` system frontmatter.
- **Proof FF1.2 ABSORB (Hylla omitted from tools list)**: Added `Hylla` to the tools column for `planning-agent`, `plan-qa-proof-agent`, `plan-qa-falsification-agent`, `build-qa-proof-agent`, `build-qa-falsification-agent`, and `research-agent` rows in the model/tools table. Updated all per-droplet AcceptanceCriteria `tools:` bullets accordingly (D1, D3, D4, D5, D6, D7, D11, D13, D14, D15, D16, D17). Note: Hylla-OFF applies to the current orchestration cycle only; authored prompts govern future dogfood when Hylla is operational — per L1 PLAN.md line 833 directive.
- **Proof NIT1 ABSORB (D21 `binding.AgentName` unspecified)**: Added explicit AcceptanceCriteria bullet to D21: "Test sets `binding.AgentName = \"builder-agent\"` so rendered file path at `<bundle.Root>/plugin/agents/builder-agent.md` matches the asserted location."
- **Proof NIT2 / Fals FF1 ABSORB (naming-convention drift `W8.D*` vs bare `D*`)**: Canonicalized on bare `D*` form throughout. PLAN.md mirror already used bare `D*`; `_BLOCKERS.toml` on-disk updated to bare `D*` form (matching sibling L2 plans + the mirror). Cross-wave reference `4c.6.1.W1` in D21 blockers retains its fully-qualified form.
- **Proof NIT3 ABSORB (migration-marker ASCII-apostrophe)**: Added CommonBuilderConstraint bullet forbidding U+2019 smart-quote conversion of `Tillsyn's` apostrophe.
- **Proof NIT4 ABSORB (`extends_path` CWD invariant)**: Added ContextBlock `reference` to D0 declaring the loader CWD invariant so W5/W6 build droplets honor it.
- **Fals NIT1 ABSORB (D21 `~/tmp/...` typo)**: Fixed D21 RiskNote `~/tmp/tillsyn/main` → `/tmp/tillsyn/main`.
- **Fals NIT2 ABSORB (`_BLOCKERS.toml` missing D0 head-node row)**: Added explicit `[[blockers]] node = "D0" blocked_by = []` entry to both PLAN.md mirror and `_BLOCKERS.toml` on-disk.
- **Fals NIT3 DEFERRED**: "visibly DIFFERENT" qualitative wording — precise diff IS in next paragraph; adding a quantitative metric risks over-rigidifying the prompt-authoring task.
- **Fals NIT4 DEFERRED**: D8/D18 WORKFLOW.md §"Phase 7 — Closeout" reference — builder verifies section header at Read-time; low-risk.
- **Fals NIT5 DEFERRED**: `extends_path` loader robustness — covered by NIT4 (proof) ContextBlock above; full hardening needs W5/W6 coordination.
- **Fals NIT6 DEFERRED**: PLAN-QA-DISCIPLINE-R2 count sync methodology — promote to separate refinement row at drop closeout.

Droplet count after round-2: 22 (unchanged). Wave graph, blocked_by topology, atomic sizing all unchanged.

---

## Objective

Author substantive Tillsyn-aware agent prompt files for Tillsyn's own project work at `.tillsyn/agents/go/` and `.tillsyn/agents/fe/`, configure the project-local vim bindings extension at `.tillsyn/bindings.json`, and update `.gitignore` to track these files. These 20 prompt files are Tillsyn's tier-1 (project-local) overrides of the 3-tier resolver — they encode mage discipline, Section 0 reasoning, MD-only workflow mode, plan-down/build-up, atomic-droplet sizing, Hylla usage, CONSUMER-TIE test contract, and QA discipline specific to Tillsyn. A dedicated smoke-test droplet (D21) verifies that `assembleAgentFileBody` (accessed via `render.Render`) correctly resolves project-tier prompts over embedded defaults when the W1 subdir-per-group path shape is in place.

---

## Wave Structure

```
Wave A (all parallel, no cross-blockers within wave):
  D0   — .gitignore update + .tillsyn/bindings.json (MUST commit before D1-D20)
  D1   — .tillsyn/agents/go/planning-agent.md          (blocked_by D0)
  D2   — .tillsyn/agents/go/builder-agent.md            (blocked_by D0)
  D3   — .tillsyn/agents/go/plan-qa-proof-agent.md      (blocked_by D0)
  D4   — .tillsyn/agents/go/plan-qa-falsification-agent.md (blocked_by D0)
  D5   — .tillsyn/agents/go/build-qa-proof-agent.md     (blocked_by D0)
  D6   — .tillsyn/agents/go/build-qa-falsification-agent.md (blocked_by D0)
  D7   — .tillsyn/agents/go/research-agent.md           (blocked_by D0)
  D8   — .tillsyn/agents/go/closeout-agent.md           (blocked_by D0; FROM SCRATCH)
  D9   — .tillsyn/agents/go/commit-message-agent.md     (blocked_by D0; FROM SCRATCH)
  D10  — .tillsyn/agents/go/orchestrator-managed.md     (blocked_by D0; FROM SCRATCH)
  D11  — .tillsyn/agents/fe/planning-agent.md           (blocked_by D0)
  D12  — .tillsyn/agents/fe/builder-agent.md            (blocked_by D0)
  D13  — .tillsyn/agents/fe/plan-qa-proof-agent.md      (blocked_by D0)
  D14  — .tillsyn/agents/fe/plan-qa-falsification-agent.md (blocked_by D0)
  D15  — .tillsyn/agents/fe/build-qa-proof-agent.md     (blocked_by D0)
  D16  — .tillsyn/agents/fe/build-qa-falsification-agent.md (blocked_by D0)
  D17  — .tillsyn/agents/fe/research-agent.md           (blocked_by D0)
  D18  — .tillsyn/agents/fe/closeout-agent.md           (blocked_by D0; FROM SCRATCH)
  D19  — .tillsyn/agents/fe/commit-message-agent.md     (blocked_by D0; FROM SCRATCH)
  D20  — .tillsyn/agents/fe/orchestrator-managed.md     (blocked_by D0; FROM SCRATCH)

Wave C (after Wave A D0-D20 complete + 4c.6.1.W1 completes):
  D21  — render_test.go smoke test  (blocked_by D0,D1,...,D20,4c.6.1.W1)
```

---

## Common Builder Constraints (ALL Droplets)

Every builder for every droplet in this wave MUST:

1. **Never run `mage install`.** Build verification uses `mage test-pkg` or `mage ci` only.
2. **Never run raw `go test`, `go build`, `go vet`.** Always `mage <target>`.
3. **Single-line conventional commits ≤72 chars; no body.** Example: `feat(prompts): add .tillsyn/agents/go/builder-agent.md`.
4. **No Section 0 inside any prompt body.** Prompts INSTRUCT subagents to render Section 0 in THEIR responses — the instruction text goes in the prompt body — but the prompt file itself is NOT a Section 0 block.
5. **Migration marker on every prompt file.** Every `.md` file under `.tillsyn/agents/` carries this comment near the top of the body (after frontmatter): `<!-- Tillsyn-project-local; lifted from ~/.claude/agents/ and adapted for Tillsyn's workflow. Future projects use embedded defaults shipped in Drop 4c.8. -->`.
6. **Source material discipline.** For D1-D7, D11-D17: PRIMARY source is `~/.claude/agents/<group>-<role>-agent.md`. Copy and adapt; do NOT write from scratch. For D8, D9, D10, D18, D19, D20: NO source file exists; author FROM SCRATCH citing CLAUDE.md + WORKFLOW.md + WIKI.md.
7. **ASCII apostrophe in migration marker.** The migration marker string uses an ASCII single-quote (U+0027) in `Tillsyn's` — do NOT let any editor or autocorrect convert it to a U+2019 right single quotation mark (curly/smart quote). Build-QA greps for the exact verbatim string; a U+2019 substitution silently breaks the grep check.

---

## Common Frontmatter Shape

Every prompt must have YAML frontmatter matching this shape:

```yaml
---
name: <agent-name>
description: <one-sentence description of this agent's role in Tillsyn>
model: <per cascade-model-policy — see table below>
tools: <comma-separated tools list — see table below>
---
```

**Model and tools per role (cascade-model-policy — R10-D4 bare aliases):**

| Role | Model | Tools |
|---|---|---|
| planning-agent | opus | Read, Grep, Glob, Hylla |
| builder-agent | sonnet | Read, Edit, Write, Grep, Glob |
| plan-qa-proof-agent | opus | Read, Grep, Glob, Hylla |
| plan-qa-falsification-agent | opus | Read, Grep, Glob, Hylla |
| build-qa-proof-agent | opus | Read, Grep, Glob, Hylla |
| build-qa-falsification-agent | opus | Read, Grep, Glob, Hylla |
| research-agent | opus | Read, Grep, Glob, Hylla |
| closeout-agent | orchestrator-managed | (orchestrator-managed — same as builder scope) |
| commit-message-agent | haiku | Read |
| orchestrator-managed | orchestrator-managed | Read, Edit, Write, Grep, Glob |

Note: `closeout` and `orchestrator-managed` kinds are handled by the orchestrator, not a dedicated model. Their prompt files document the orchestrator's role and constraints. `model: orchestrator-managed` is a string value indicating orchestrator-managed scope — matches Tillsyn's orchestrator-managed-role convention (R10-D4).

---

## Common Validator Requirements (Signal A + B + C)

Every prompt file must satisfy the `validateAgentBodyShape` 3-signal AND check (from `render.go`):

- **Signal B** — leading `---\n` + closing `---\n` frontmatter delimiters present; inner block has `name:` field.
- **Signal A** — post-frontmatter body length > 200 bytes (actual requirement is > 200; aim for >= 1000 chars total body per L1 acceptance).
- **Signal C** — body contains at least one of: `# PLACEHOLDER`, `# Section 0`, `## Role`. Use `## Role` as the primary role-section header in all substantive prompts.

---

## Planner

### D0 — .gitignore Update + .tillsyn/bindings.json Authoring

- **State:** todo
- **Kind:** build
- **Irreducible:** true
- **Paths:**
  - `.gitignore` (MODIFY)
  - `.tillsyn/bindings.json` (NEW)
- **Packages:** (none — non-Go files)
- **Blocked by:** (none — Wave A leader)
- **Commit must precede D1-D20:** D0 makes `.tillsyn/agents/**/*.md` trackable by git. Commit D0 before any prompt-file droplets so `git ls-files .tillsyn/agents/` confirms tracking after each prompt.

#### Specify (D0)

**Objective:** Update `.gitignore` to re-include the project-local agent prompts and bindings file alongside the existing `template.toml` re-include. Author `.tillsyn/bindings.json` with the 5 Tillsyn-local additions (ID-based deep merge with stil baseline).

**AcceptanceCriteria:**
- `.gitignore` contains all four re-include lines in this order, adjacent to the existing `!.tillsyn/template.toml`:
  ```
  !.tillsyn/template.toml
  !.tillsyn/agents/
  !.tillsyn/agents/**/*.md
  !.tillsyn/bindings.json
  ```
  Note: `!.tillsyn/agents/` must be present — without it the directory remains excluded and `/**/*.md` has no effect.
- Runtime state files (`.tillsyn/config.toml`, `.tillsyn/tillsyn.db*`, `.tillsyn/logs/`, `.tillsyn/livewait.secret`) remain ignored (verify by inspection that no new broad-inclusion rules were added).
- `.tillsyn/bindings.json` exists with the exact JSON structure below:
  ```json
  {
    "schema_version": 1,
    "name": "tillsyn-bindings",
    "description": "Tillsyn project-local vim bindings extension. ID-merges with stil baseline; local wins on collision.",
    "extends": "stil-baseline",
    "extends_path": "../../../stil/main/src/bindings/baseline.json",
    "product_extensions": {
      "tillsyn": {
        "description": "Tillsyn-specific commands ADDED to baseline (ID-based deep merge; local wins on collision).",
        "commands": [
          { "id": "dispatch", "command": "dispatch", "description": "Trigger dispatcher on the focused action item — accepts optional flags." },
          { "id": "plan",     "command": "plan",     "description": "Open the planner for the focused project or sub-plan." },
          { "id": "archive",  "command": "archive",  "description": "Archive the focused project or action item." },
          { "id": "settings", "command": "settings", "description": "Open the settings panel (agents.toml + template.toml + groups)." },
          { "id": "help",     "command": "help",     "description": "Open the help panel — keybinding reference + tips." }
        ]
      }
    }
  }
  ```
- **Exactly 5 commands** in the local file (the Tillsyn-local ADDITIONS only: `dispatch`, `plan`, `archive`, `settings`, `help`). The original `close` from any earlier draft is NOT present — it was redundant with stil's canonical `complete-drop`.
- IDs `dispatch`, `plan`, `archive`, `settings`, `help` are disjoint from stil baseline's 4 IDs (`new-drop`, `complete-drop`, `handoff`, `comment`) — no collision.
- After commit: `git ls-files .tillsyn/bindings.json` returns the file as tracked.
- `git ls-files .tillsyn/agents/` returns empty (agents not yet authored — that's correct at this stage).

**ValidationPlan:**
- `git diff .gitignore` — verify the 3 new re-include lines are added in the right place.
- `cat .tillsyn/bindings.json` — verify JSON is valid and has exactly 5 commands.
- `git ls-files .tillsyn/bindings.json` — confirms tracking.

**RiskNotes:**
- Git re-inclusion order matters: `!.tillsyn/agents/` must come before `!.tillsyn/agents/**/*.md` or the directory-level exclude prevents the glob.
- Do NOT add `!.tillsyn/agents/**` (without `.md`) — would re-include runtime-adjacent files that end up in agent subdirs.
- The `extends_path` value is relative to the consuming loader's working directory at runtime. Keep verbatim as specified.

**ContextBlocks:**
- `constraint` (high): Do NOT use `!.tillsyn/agents/**` (too broad). Use `!.tillsyn/agents/` (dir) + `!.tillsyn/agents/**/*.md` (files) as separate rules.
- `reference`: Current `.gitignore` lines 12-19: `.tillsyn/*` + `!.tillsyn/template.toml`. New lines go AFTER `!.tillsyn/template.toml`.
- `reference`: Stil baseline has IDs `new-drop`, `complete-drop`, `handoff`, `comment` in `product_extensions.tillsyn`. These are NOT in the local file — local file has only ADDITIONS.
- `decision`: `extends_path` is `"../../../stil/main/src/bindings/baseline.json"` (relative path from `tillsyn/main/.tillsyn/bindings.json` to the stil source). Loaders resolve this at runtime.
- `reference` (W5/W6 loader CWD invariant): `extends_path` resolves correctly ONLY when the loader's CWD is `tillsyn/main/.tillsyn/` (three `..` steps reach the directory containing both `tillsyn/` and `stil/`). W5 (Go TUI keybinding loader) and W6 (FE JS loader) build droplets MUST honor this CWD invariant when reading this file. If a loader runs from a different CWD, the path will silently miss. W5/W6 builders: verify your loader's CWD or resolve the path relative to the file location, not the process CWD.

**KindPayload:**
```json
{"changes": [
  {"file": ".gitignore", "symbol": "tillsyn runtime state comment block", "action": "modify", "shape_hint": "Add 3 lines after !.tillsyn/template.toml: !.tillsyn/agents/ + !.tillsyn/agents/**/*.md + !.tillsyn/bindings.json"},
  {"file": ".tillsyn/bindings.json", "symbol": "(new file)", "action": "add", "shape_hint": "JSON with schema_version:1, product_extensions.tillsyn.commands array of 5 entries"}
]}
```

**Mage target:** `mage ci` (no Go changed; runs format + vet only but confirms no regressions).

---

### D1 — .tillsyn/agents/go/planning-agent.md

- **State:** todo
- **Kind:** build
- **Irreducible:** true
- **Paths:** `.tillsyn/agents/go/planning-agent.md` (NEW)
- **Packages:** (none)
- **Blocked by:** D0

#### Specify (D1)

**Objective:** Author the Tillsyn-project-local go planning agent prompt. PRIMARY source: `~/.claude/agents/go-planning-agent.md`. Copy and adapt for Tillsyn's project-specific discipline. The prompt encodes: mage-first build gates, Section 0 reasoning requirement (as directive to the planning agent), plan-down/build-up methodology, atomic-droplet sizing (1-4 code blocks, 80-120 LOC + tests per till-go template), Hylla evidence order (Hylla first then fallback), CONSUMER-TIE test contract awareness, Tillsyn-specific PLAN.md droplet shape (paths/packages/acceptance/blocked_by), single-line conventional commits.

**AcceptanceCriteria:**
- File exists at `.tillsyn/agents/go/planning-agent.md`.
- Frontmatter: `name: planning-agent`, `description: <one-sentence>`, `model: opus`, `tools: Read, Grep, Glob, Hylla`.
- Migration marker present in body after frontmatter: `<!-- Tillsyn-project-local; lifted from ~/.claude/agents/ and adapted for Tillsyn's workflow. Future projects use embedded defaults shipped in Drop 4c.8. -->`.
- Body >= 1000 chars (post-frontmatter).
- Body contains `## Role` header (satisfies Signal C of post-render validator).
- Body explicitly encodes: mage targets (never raw `go test`/`go build`/`mage install`), Section 0 5-pass certificate requirement, plan-down/build-up, atomic-droplet sizing per till-go template, Hylla evidence-first order, `blocked_by` graph wiring rules, paths/packages declarations.
- No Section 0 block rendered in the file itself — the instruction to render Section 0 appears as directive text, not as a rendered block.
- Passes `validateAgentBodyShape` (Signal A + B + C all green): `mage ci` or `mage test-pkg ./internal/app/dispatcher/cli_claude/render` green after file exists (D21 will exercise this directly post-W1).

**ValidationPlan:**
- Word-count check: `wc -c .tillsyn/agents/go/planning-agent.md` — confirm > 1000 chars total.
- Manual inspection: frontmatter keys present, migration marker present, `## Role` present.
- `mage ci` green (no Go changed by this droplet).

**RiskNotes:**
- Primary source `~/.claude/agents/go-planning-agent.md` is the CURRENT production-grade prompt. Builders have filesystem access to read it. Do NOT write from scratch — copy and adapt.
- The prompt is a Tillsyn-specific SPECIALIZATION of the global agent. Keep all the general planning discipline from the source; ADD Tillsyn-specific rules (mage targets, Hylla order, MD-only workflow mode, CONSUMER-TIE).
- Per SKETCH §3.1: planning-agent is distinct from plan-qa-proof-agent. Planning decomposes work; plan-qa-proof verifies the plan's decomposition. Do not conflate roles.

**ContextBlocks:**
- `reference`: Source file `~/.claude/agents/go-planning-agent.md` — PRIMARY starting point.
- `reference`: CLAUDE.md §"Go Development Rules" — mage discipline, TDD, error wrapping.
- `reference`: CLAUDE.md §"Build Verification" — never `go test`, never `mage install`.
- `reference`: CLAUDE.md §"Cascade Tree Structure" — agent bindings, kind hierarchy.
- `reference`: Memory `feedback_plan_down_build_up.md` — NO cap on children per planning pass; only atomic-droplet sizing caps recursion.
- `reference`: Memory `feedback_section_0_required.md` — 5-pass certificate required for substantive responses.
- `reference`: Memory `feedback_hylla_go_code.md` — Hylla first, git diff for changed-since-ingest.
- `decision`: Planning agent uses opus model (cascade-model-policy).
- `constraint` (high): No `mage install` in any verification step the builder agent is instructed to run.

**KindPayload:**
```json
{"changes": [
  {"file": ".tillsyn/agents/go/planning-agent.md", "symbol": "(new file)", "action": "add", "shape_hint": "frontmatter + migration marker + ## Role section + Tillsyn-specific planning discipline"}
]}
```

**Mage target:** `mage ci` (no Go changed).

---

### D2 — .tillsyn/agents/go/builder-agent.md

- **State:** todo
- **Kind:** build
- **Irreducible:** true
- **Paths:** `.tillsyn/agents/go/builder-agent.md` (NEW)
- **Packages:** (none)
- **Blocked by:** D0

#### Specify (D2)

**Objective:** Author the Tillsyn-project-local go builder agent prompt. PRIMARY source: `~/.claude/agents/go-builder-agent.md`. Copy and adapt. The prompt encodes: builder is THE ONLY role that edits Go source; mage-first build verification; never `mage install`; TDD-first; table-driven tests; error wrapping with `%w`; `charmbracelet/log` for logging; mage targets for verification (`mage test-pkg`, `mage ci`); hexagonal architecture + interface-first; CONSUMER-TIE test contract (existing `run(ctx, args, &out, io.Discard)` end-to-end pattern for CLI tests); single-line conventional commits.

**AcceptanceCriteria:**
- File exists at `.tillsyn/agents/go/builder-agent.md`.
- Frontmatter: `name: builder-agent`, `description: <one-sentence>`, `model: sonnet`, `tools: Read, Edit, Write, Grep, Glob`.
- Migration marker present.
- Body >= 1000 chars.
- Body contains `## Role` header.
- Body explicitly encodes: builder is sole code-editing role; mage targets; never raw `go test`/`go build`/`go vet`/`mage install`; TDD-first; CONSUMER-TIE test pattern; error wrapping; logger discipline; single-line commits.
- No Section 0 block in the file itself (only the directive instruction to generate it).
- `mage ci` green.

**RiskNotes:**
- This is the only role that writes source code. The prompt must be explicit that orchestrators NEVER call `Edit`/`Write` on Go files — only this agent does.
- CONSUMER-TIE is the existing CLI test pattern (`run(ctx, args, &out, io.Discard)` end-to-end). Builder must preserve this contract.

**ContextBlocks:**
- `reference`: Source `~/.claude/agents/go-builder-agent.md`.
- `reference`: CLAUDE.md §"Go Development Rules" — full stack of Go rules.
- `reference`: Memory `feedback_orchestrator_no_build.md` — orchestrator never writes code; builder subagent is sole source.
- `reference`: Memory `feedback_nits_are_first_class.md` — address every NIT.
- `constraint` (critical): Builder prompt must include explicit instruction: "NEVER run `mage install` — this promotes a binary to `$HOME/.local/bin/till` and is dev-only."

**KindPayload:**
```json
{"changes": [{"file": ".tillsyn/agents/go/builder-agent.md", "symbol": "(new file)", "action": "add", "shape_hint": "frontmatter + migration marker + ## Role + Go-build discipline + mage targets + CONSUMER-TIE + commit style"}]}
```

**Mage target:** `mage ci`.

---

### D3 — .tillsyn/agents/go/plan-qa-proof-agent.md

- **State:** todo
- **Kind:** build
- **Irreducible:** true
- **Paths:** `.tillsyn/agents/go/plan-qa-proof-agent.md` (NEW)
- **Packages:** (none)
- **Blocked by:** D0

#### Specify (D3)

**Objective:** Author the Tillsyn-project-local go plan-QA-proof agent prompt. PRIMARY source to START FROM: `~/.claude/agents/go-qa-proof-agent.md`. BUT: this file must be DIFFERENTIATED from D5 (build-qa-proof). Per SKETCH §3.1:

- **plan-qa-proof** reviews PLAN.MD DECOMPOSITION: blocked_by graph correctness, paths/packages declarations, acceptance bullets, surface boundaries, structural_type classifications. Evidence sources: PLAN.md + REVISION_BRIEF.md + SKETCH.md (not Go source).
- This is NOT the same as build-qa-proof which reviews Go source changes.

**AcceptanceCriteria:**
- File exists at `.tillsyn/agents/go/plan-qa-proof-agent.md`.
- Frontmatter: `name: plan-qa-proof-agent`, `model: opus`, `tools: Read, Grep, Glob, Hylla`.
- Migration marker present.
- Body >= 1000 chars, contains `## Role`.
- **Evidence Sources section** explicitly lists: PLAN.md, REVISION_BRIEF.md, SKETCH.md — NOT Go source files, NOT test output.
- **What To Check section** covers: blocked_by graph for missing blockers between droplets sharing paths/packages; paths/packages declared for every build droplet; acceptance criteria are yes/no testable; structural_type consistency (droplet = leaf = no children; confluence = non-empty blocked_by); no scope creep beyond planner boundary.
- Body is visibly DIFFERENT from D5 (build-qa-proof) in its Evidence Sources and What To Check sections — NOT a near-identical copy.
- `mage ci` green.

**RiskNotes:**
- The global `go-qa-proof-agent.md` is a SINGLE file that seeds BOTH plan-qa-proof and build-qa-proof. This must NOT produce near-identical copies. The differentiation is mandatory per QA-SPLIT-R1 tracked for Drop 4c.8; this drop must at minimum have visibly different Evidence Sources and What To Check.
- Plan-QA-proof is adversarial toward the PLAN; build-QA-proof is adversarial toward the BUILD. Different attack surfaces.

**ContextBlocks:**
- `reference`: Source `~/.claude/agents/go-qa-proof-agent.md` (starting point, then differentiate).
- `reference`: SKETCH §3.1 — plan-qa-proof vs build-qa-proof work distinction.
- `reference`: WORKFLOW.md §"Phase 2 — Plan QA" — plan-QA checks blocked_by coverage, paths/packages, acceptance.
- `constraint` (high): Must NOT be a near-identical copy of D5. Builder must actively differentiate the Evidence Sources and What To Check sections.
- `decision`: QA-SPLIT-R1 tracks full split in Drop 4c.8; this drop's requirement is minimum visible differentiation.

**KindPayload:**
```json
{"changes": [{"file": ".tillsyn/agents/go/plan-qa-proof-agent.md", "symbol": "(new file)", "action": "add", "shape_hint": "frontmatter + ## Role + Evidence Sources: PLAN.md/REVISION_BRIEF/SKETCH + What To Check: blocked_by graph/paths/packages/acceptance/structural_type"}]}
```

**Mage target:** `mage ci`.

---

### D4 — .tillsyn/agents/go/plan-qa-falsification-agent.md

- **State:** todo
- **Kind:** build
- **Irreducible:** true
- **Paths:** `.tillsyn/agents/go/plan-qa-falsification-agent.md` (NEW)
- **Packages:** (none)
- **Blocked by:** D0

#### Specify (D4)

**Objective:** Author the Tillsyn-project-local go plan-QA-falsification agent prompt. PRIMARY source: `~/.claude/agents/go-qa-falsification-agent.md`. Differentiate from D6 (build-qa-falsification). Plan-QA-falsification ATTACKS THE PLAN — finds missing blockers, hidden dependencies, scope creep, blocker cycles, paths/packages overlaps. Evidence: PLAN.md + REVISION_BRIEF.md + SKETCH.md.

**AcceptanceCriteria:**
- File exists at `.tillsyn/agents/go/plan-qa-falsification-agent.md`.
- Frontmatter: `name: plan-qa-falsification-agent`, `model: opus`, `tools: Read, Grep, Glob, Hylla`.
- Migration marker present.
- Body >= 1000 chars, contains `## Role`.
- **Evidence Sources section**: PLAN.md, REVISION_BRIEF.md, SKETCH.md.
- **Attack Vectors section** covers: missing `blocked_by` between droplets sharing paths/packages; cycles in blocked_by graph; `_BLOCKERS.toml` vs PLAN.md inline drift; structural_type violations (droplet with children, confluence with empty blocked_by); acceptance criteria that aren't yes/no testable; planner scope creep.
- Visibly DIFFERENT from D6 (build-qa-falsification) — plan attacks vs build attacks are different.
- `mage ci` green.

**ContextBlocks:**
- `reference`: Source `~/.claude/agents/go-qa-falsification-agent.md`.
- `reference`: SKETCH §3.1.
- `constraint` (high): Must NOT be near-identical to D6.

**KindPayload:**
```json
{"changes": [{"file": ".tillsyn/agents/go/plan-qa-falsification-agent.md", "symbol": "(new file)", "action": "add", "shape_hint": "frontmatter + ## Role + Evidence: PLAN.md/REVISION_BRIEF/SKETCH + Attack Vectors: missing blockers/cycles/drift/structural violations"}]}
```

**Mage target:** `mage ci`.

---

### D5 — .tillsyn/agents/go/build-qa-proof-agent.md

- **State:** todo
- **Kind:** build
- **Irreducible:** true
- **Paths:** `.tillsyn/agents/go/build-qa-proof-agent.md` (NEW)
- **Packages:** (none)
- **Blocked by:** D0

#### Specify (D5)

**Objective:** Author the Tillsyn-project-local go build-QA-proof agent prompt. PRIMARY source: `~/.claude/agents/go-qa-proof-agent.md`. Differentiate from D3 (plan-qa-proof). Build-QA-proof verifies ACTUAL CODE CHANGES against the plan: test pass rates, no scope creep beyond declared paths, evidence for each acceptance bullet in the build droplet. Evidence: Go source + test output + `git diff`.

**AcceptanceCriteria:**
- File exists at `.tillsyn/agents/go/build-qa-proof-agent.md`.
- Frontmatter: `name: build-qa-proof-agent`, `model: opus`, `tools: Read, Grep, Glob, Hylla`.
- Migration marker present.
- Body >= 1000 chars, contains `## Role`.
- **Evidence Sources section**: Go source files (declared paths), `git diff`, `mage test-pkg` output, PLAN.md (for the acceptance criteria being verified against).
- **What To Check section**: test pass rates, coverage not below 70%, no files modified outside declared paths, each acceptance criterion bullet verifiable from code, no TODO/FIXME/stub left in production code, `mage ci` green evidence.
- Visibly DIFFERENT from D3 (plan-qa-proof).
- `mage ci` green.

**ContextBlocks:**
- `reference`: Source `~/.claude/agents/go-qa-proof-agent.md`.
- `constraint` (high): Must NOT be near-identical to D3.

**KindPayload:**
```json
{"changes": [{"file": ".tillsyn/agents/go/build-qa-proof-agent.md", "symbol": "(new file)", "action": "add", "shape_hint": "frontmatter + ## Role + Evidence: Go source/git diff/test output/PLAN.md + What To Check: test pass/coverage/paths/acceptance/mage ci"}]}
```

**Mage target:** `mage ci`.

---

### D6 — .tillsyn/agents/go/build-qa-falsification-agent.md

- **State:** todo
- **Kind:** build
- **Irreducible:** true
- **Paths:** `.tillsyn/agents/go/build-qa-falsification-agent.md` (NEW)
- **Packages:** (none)
- **Blocked by:** D0

#### Specify (D6)

**Objective:** Author the Tillsyn-project-local go build-QA-falsification agent prompt. PRIMARY source: `~/.claude/agents/go-qa-falsification-agent.md`. Differentiate from D4 (plan-qa-falsification). Build-QA-falsification ATTACKS THE BUILD — counterexamples to test claims, race conditions, edge cases tests miss, false-passes, test residue, security gaps. Evidence: Go source + test output + `git diff`.

**AcceptanceCriteria:**
- File exists at `.tillsyn/agents/go/build-qa-falsification-agent.md`.
- Frontmatter: `name: build-qa-falsification-agent`, `model: opus`, `tools: Read, Grep, Glob, Hylla`.
- Migration marker present.
- Body >= 1000 chars, contains `## Role`.
- **Evidence Sources section**: Go source, `git diff`, test output.
- **Attack Vectors section**: counterexamples to test table claims, missing race-safety checks (no `-race` coverage), edge cases not covered, test that passes with wrong implementation (false-positive test), test residue (skipped tests, `t.Skip`, commented-out assertions), security gap (prompt injection surface, error leakage).
- Visibly DIFFERENT from D4 (plan-qa-falsification).
- `mage ci` green.

**ContextBlocks:**
- `reference`: Source `~/.claude/agents/go-qa-falsification-agent.md`.
- `constraint` (high): Must NOT be near-identical to D4.

**KindPayload:**
```json
{"changes": [{"file": ".tillsyn/agents/go/build-qa-falsification-agent.md", "symbol": "(new file)", "action": "add", "shape_hint": "frontmatter + ## Role + Evidence: Go source/git diff/test output + Attack Vectors: counterexamples/race/edge cases/false-positive tests/security"}]}
```

**Mage target:** `mage ci`.

---

### D7 — .tillsyn/agents/go/research-agent.md

- **State:** todo
- **Kind:** build
- **Irreducible:** true
- **Paths:** `.tillsyn/agents/go/research-agent.md` (NEW)
- **Packages:** (none)
- **Blocked by:** D0

#### Specify (D7)

**Objective:** Author the Tillsyn-project-local go research agent prompt. PRIMARY source: `~/.claude/agents/go-research-agent.md`. Copy and adapt. Research agent is READ-ONLY — compiles findings, posts findings, dies. Encodes: Hylla evidence order (Hylla → git diff → Context7/go doc/LSP), never edits code, findings returned as structured response for orchestrator to route.

**AcceptanceCriteria:**
- File exists at `.tillsyn/agents/go/research-agent.md`.
- Frontmatter: `name: research-agent`, `model: opus`, `tools: Read, Grep, Glob, Hylla`.
- Migration marker present.
- Body >= 1000 chars, contains `## Role`.
- Body encodes: read-only role (no Edit/Write), Hylla evidence order, findings returned to orchestrator, never creates action items.
- `mage ci` green.

**ContextBlocks:**
- `reference`: Source `~/.claude/agents/go-research-agent.md`.
- `constraint` (high): Research agent NEVER edits code. Must be explicit in prompt.

**KindPayload:**
```json
{"changes": [{"file": ".tillsyn/agents/go/research-agent.md", "symbol": "(new file)", "action": "add", "shape_hint": "frontmatter + ## Role + read-only discipline + Hylla evidence order + findings-to-orchestrator pattern"}]}
```

**Mage target:** `mage ci`.

---

### D8 — .tillsyn/agents/go/closeout-agent.md

- **State:** todo
- **Kind:** build
- **Irreducible:** true
- **Paths:** `.tillsyn/agents/go/closeout-agent.md` (NEW)
- **Packages:** (none)
- **Blocked by:** D0

#### Specify (D8)

**Objective:** Author the Tillsyn-project-local go closeout agent prompt. NO SOURCE FILE EXISTS at `~/.claude/agents/closeout-agent.md` — author FROM SCRATCH. Primary references: CLAUDE.md §"Cascade Tree Structure" (closeout kind semantics), WORKFLOW.md §"Phase 7 — Closeout" (what closeout does), WIKI.md (cascade vocabulary), CLAUDE.md §"Cascade Ledger + Hylla Feedback". The closeout kind is orchestrator-managed in Drop 4c (see ORCH-MANAGED-R1 — Drop 4c.8 will split into dedicated closeout-agent, refinement-agent, etc.; for now closeout is handled by the orchestrator).

Closeout scope per WORKFLOW.md Phase 7: aggregate Hylla feedback from BUILDER_WORKLOG.md, aggregate refinements, write CLOSEOUT.md, flip drop state to done. Does NOT trigger Hylla reingest (orchestrator does that drop-end only).

**AcceptanceCriteria:**
- File exists at `.tillsyn/agents/go/closeout-agent.md`.
- Frontmatter: `name: closeout-agent`, `description: <one-sentence>`, `model: orchestrator-managed`, `tools: Read, Edit, Write, Grep, Glob`.
- Migration marker present.
- Body >= 1000 chars, contains `## Role`.
- Body encodes: closeout is orchestrator-managed; references WORKFLOW.md Phase 7 steps; Hylla reingest is drop-end only (orchestrator runs it, not this agent); aggregation of Hylla feedback from BUILDER_WORKLOG.md; CLOSEOUT.md authoring; drop state flip; no per-droplet reingest.
- Note ORCH-MANAGED-R1: this prompt serves as a placeholder until Drop 4c.8 splits the orchestrator-managed kinds into dedicated agents.
- Body authored FROM SCRATCH (no `~/.claude/agents/closeout-agent.md` source exists).
- `mage ci` green.

**RiskNotes:**
- Builder must NOT look for `~/.claude/agents/closeout-agent.md` — it does not exist. Author from scratch.
- The closeout kind is handled by the orchestrator in the current cascade. The prompt documents what the orchestrator does during closeout, not a separate subagent.
- Per `feedback_no_closeout_md_pre_dogfood.md`: skip CLOSEOUT/LEDGER/WIKI_CHANGELOG/REFINEMENTS/HYLLA_FEEDBACK rollups while not dogfooding. This memory entry applies to the orchestrator behavior, not to this prompt's content — include the full Phase 7 discipline anyway for post-dogfood use.

**ContextBlocks:**
- `reference`: WORKFLOW.md §"Phase 7 — Closeout" — authoritative scope for this agent.
- `reference`: CLAUDE.md §"Cascade Ledger + Hylla Feedback".
- `reference`: CLAUDE.md §"Drop Closeout" — Hylla ingest invariants.
- `decision`: ORCH-MANAGED-R1 — Drop 4c.8 will split into dedicated agents. This file is a placeholder with full closeout discipline documented.
- `constraint` (high): Hylla reingest is NEVER triggered by this agent — orchestrator only, drop-end only, after CI green.

**KindPayload:**
```json
{"changes": [{"file": ".tillsyn/agents/go/closeout-agent.md", "symbol": "(new file)", "action": "add", "shape_hint": "frontmatter + ## Role + closeout-phase discipline + Hylla-ingest invariants + ORCH-MANAGED-R1 note"}]}
```

**Mage target:** `mage ci`.

---

### D9 — .tillsyn/agents/go/commit-message-agent.md

- **State:** todo
- **Kind:** build
- **Irreducible:** true
- **Paths:** `.tillsyn/agents/go/commit-message-agent.md` (NEW)
- **Packages:** (none)
- **Blocked by:** D0

#### Specify (D9)

**Objective:** Author the Tillsyn-project-local go commit-message agent prompt. NO SOURCE FILE at `~/.claude/agents/commit-message-agent.md` — author FROM SCRATCH. References: CLAUDE.md §"Git Commit Format", CLAUDE.md §"Build-QA-Commit Discipline" (QA before commit), memory `feedback_commit_style.md` (single-line, no body). Model: haiku (cheapest, commit authoring is mechanical).

Commit discipline: conventional-commit `type(scope): message`, single line ≤72 chars, no body, no bullet lists, no period at end, lowercase except proper nouns/acronyms, subject only (diff records the what; subject carries the human summary).

**AcceptanceCriteria:**
- File exists at `.tillsyn/agents/go/commit-message-agent.md`.
- Frontmatter: `name: commit-message-agent`, `model: haiku`, `tools: Read`.
- Migration marker present.
- Body >= 1000 chars, contains `## Role`.
- Body encodes: conventional-commit format, single-line ≤72 chars, no body paragraphs, types (`feat`/`fix`/`refactor`/`chore`/`docs`/`test`/`ci`/`style`/`perf`), examples of good commit messages from the project's history, QA-before-commit rule (never commit without both QA passes).
- Authored FROM SCRATCH.
- `mage ci` green.

**ContextBlocks:**
- `reference`: CLAUDE.md §"Git Commit Format".
- `reference`: Memory `feedback_commit_style.md` — single-line commits only.
- `reference`: Memory `feedback_qa_before_commit.md` — QA before commit is mandatory.
- `decision`: Haiku model — commit authoring is mechanical; expensive models not needed.

**KindPayload:**
```json
{"changes": [{"file": ".tillsyn/agents/go/commit-message-agent.md", "symbol": "(new file)", "action": "add", "shape_hint": "frontmatter + ## Role + conventional-commit format + single-line discipline + QA-before-commit rule"}]}
```

**Mage target:** `mage ci`.

---

### D10 — .tillsyn/agents/go/orchestrator-managed.md

- **State:** todo
- **Kind:** build
- **Irreducible:** true
- **Paths:** `.tillsyn/agents/go/orchestrator-managed.md` (NEW)
- **Packages:** (none)
- **Blocked by:** D0

#### Specify (D10)

**Objective:** Author the Tillsyn-project-local go orchestrator-managed agent prompt. NO SOURCE FILE — author FROM SCRATCH. This is the 10th special file per group (FF3 disposition — ORCH-MANAGED-R1). It covers the orchestrator-managed kinds: `closeout`, `refinement`, `discussion`, `human-verify`. References: CLAUDE.md §"Orchestrator-as-Hub Architecture", CLAUDE.md §"Cascade Tree Structure" (kind semantics for these 4 kinds), WORKFLOW.md, WIKI.md cascade vocabulary.

The orchestrator-managed prompt documents: what the orchestrator does for each of these 4 kinds, the full-toolset scope (orchestrator has Read/Edit/Write/Grep/Glob for MD docs), when to spawn subagents vs handle inline, never-edit-Go-source rule (applies even in orchestrator role), attention_item for human sign-off on human-verify kind.

**AcceptanceCriteria:**
- File exists at `.tillsyn/agents/go/orchestrator-managed.md`.
- Frontmatter: `name: orchestrator-managed`, `model: orchestrator-managed`, `tools: Read, Edit, Write, Grep, Glob`.
- Migration marker present.
- Body >= 1000 chars, contains `## Role`.
- Body covers all 4 orchestrator-managed kinds: `closeout`, `refinement`, `discussion`, `human-verify`.
- Body encodes: orchestrator never edits Go source; MD-doc ownership split (drop-orch owns drop branch MDs; STEWARD owns post-merge collation); never use TodoWrite/TaskCreate (use Tillsyn or drop-dir MDs); ORCH-MANAGED-R1 note that Drop 4c.8 will split into dedicated agents.
- Authored FROM SCRATCH.
- `mage ci` green.

**ContextBlocks:**
- `reference`: CLAUDE.md §"Orchestrator-as-Hub Architecture".
- `reference`: CLAUDE.md §"Coordination Model".
- `reference`: WIKI.md §"Closed 12-Value `kind` Enum" — semantics of closeout/refinement/discussion/human-verify.
- `reference`: Memory `feedback_orchestrator_no_build.md`.
- `decision`: ORCH-MANAGED-R1 — Drop 4c.8 splits into dedicated agents; this file is the interim.

**KindPayload:**
```json
{"changes": [{"file": ".tillsyn/agents/go/orchestrator-managed.md", "symbol": "(new file)", "action": "add", "shape_hint": "frontmatter + ## Role + 4-kind coverage (closeout/refinement/discussion/human-verify) + orchestrator discipline + ORCH-MANAGED-R1 note"}]}
```

**Mage target:** `mage ci`.

---

### D11 — .tillsyn/agents/fe/planning-agent.md

- **State:** todo
- **Kind:** build
- **Irreducible:** true
- **Paths:** `.tillsyn/agents/fe/planning-agent.md` (NEW)
- **Packages:** (none)
- **Blocked by:** D0

#### Specify (D11)

**Objective:** Author the Tillsyn-project-local FE planning agent prompt. PRIMARY source: `~/.claude/agents/fe-planning-agent.md`. Copy and adapt for Tillsyn's FE-specific project discipline. The FE planning agent encodes: Astro/SolidJS/Wails v2 awareness, stil tokens consumption from `src/styles/tokens.css` (not `dist/`), size-adaptive CSS (container queries), component-boundary planning, Playwright via MCP for FE tests, Vitest for unit tests, migration marker targets (`// MIGRATION TARGET: @hylla/stil-solid`, `// MIGRATION TARGET: github.com/hylla-org/ro-vim`), Section 0 reasoning directive, plan-down/build-up.

**AcceptanceCriteria:**
- File at `.tillsyn/agents/fe/planning-agent.md`.
- Frontmatter: `name: planning-agent`, `model: opus`, `tools: Read, Grep, Glob, Hylla`.
- Migration marker.
- Body >= 1000 chars, `## Role`.
- Body encodes FE-specific planning: component architecture, a11y planning, responsive CSS, Wails IPC awareness, vim keybinding engine (fe/frontend/src/lib/vim/), stil tokens from src/ path.
- `mage ci` green.

**ContextBlocks:**
- `reference`: Source `~/.claude/agents/fe-planning-agent.md`.
- `reference`: REVISION_BRIEF §2.15 — FE scaffold scope (Wails v2 + Astro + Solid + stil).
- `reference`: SKETCH §5 — FE architecture decisions.
- `decision`: Stil tokens path is `src/styles/tokens.css` NOT `dist/tokens.css` (per R3-NIT7 decision in SKETCH §10).

**KindPayload:**
```json
{"changes": [{"file": ".tillsyn/agents/fe/planning-agent.md", "symbol": "(new file)", "action": "add", "shape_hint": "frontmatter + ## Role + FE planning discipline: Astro/Solid/Wails + stil tokens + a11y + responsive + vim engine awareness"}]}
```

**Mage target:** `mage ci`.

---

### D12 — .tillsyn/agents/fe/builder-agent.md

- **State:** todo
- **Kind:** build
- **Irreducible:** true
- **Paths:** `.tillsyn/agents/fe/builder-agent.md` (NEW)
- **Packages:** (none)
- **Blocked by:** D0

#### Specify (D12)

**Objective:** Author the Tillsyn-project-local FE builder agent prompt. PRIMARY source: `~/.claude/agents/fe-builder-agent.md`. Adapt for Tillsyn's FE stack: Wails v2 + Astro + SolidJS + stil tokens, vim engine at `fe/frontend/src/lib/vim/`, wails-keys.ts filter, Playwright via MCP, Vitest, TS strict, ESLint. The builder is the ONLY role that edits FE source (`fe/`). Migration markers on every component file (`// MIGRATION TARGET: @hylla/stil-solid`, etc.).

**AcceptanceCriteria:**
- File at `.tillsyn/agents/fe/builder-agent.md`.
- Frontmatter: `name: builder-agent`, `model: sonnet`, `tools: Read, Edit, Write, Grep, Glob`.
- Migration marker.
- Body >= 1000 chars, `## Role`.
- Body: Wails v2 IPC awareness (no till-serve dep), Astro/Solid component authoring, stil tokens from `src/` path, vim engine integration, Playwright via MCP for visual tests (browser_snapshot + browser_take_screenshot), Vitest for unit tests, TS strict + ESLint, migration marker discipline.
- `mage ci` green.

**ContextBlocks:**
- `reference`: Source `~/.claude/agents/fe-builder-agent.md`.
- `reference`: REVISION_BRIEF §2.15.
- `reference`: SKETCH §5.4 — Wails layout.
- `constraint` (high): Stil tokens path is `src/styles/tokens.css` (not `dist/`).

**KindPayload:**
```json
{"changes": [{"file": ".tillsyn/agents/fe/builder-agent.md", "symbol": "(new file)", "action": "add", "shape_hint": "frontmatter + ## Role + FE builder: Wails IPC/Astro/Solid/stil + vim engine + Playwright MCP + Vitest + migration markers"}]}
```

**Mage target:** `mage ci`.

---

### D13 — .tillsyn/agents/fe/plan-qa-proof-agent.md

- **State:** todo
- **Kind:** build
- **Irreducible:** true
- **Paths:** `.tillsyn/agents/fe/plan-qa-proof-agent.md` (NEW)
- **Packages:** (none)
- **Blocked by:** D0

#### Specify (D13)

**Objective:** Author the Tillsyn-project-local FE plan-QA-proof agent prompt. PRIMARY source: `~/.claude/agents/fe-qa-proof-agent.md`. Differentiate from D15 (build-qa-proof). FE plan-qa-proof reviews PLAN.MD DECOMPOSITION for FE droplets: component boundary correctness, a11y coverage planning, responsive coverage planning, surface boundary isolation, blocked_by graph for file/package overlaps. Evidence: PLAN.md, REVISION_BRIEF.md, SKETCH.md.

**AcceptanceCriteria:**
- File at `.tillsyn/agents/fe/plan-qa-proof-agent.md`.
- Frontmatter: `name: plan-qa-proof-agent`, `model: opus`, `tools: Read, Grep, Glob, Hylla`.
- Migration marker, body >= 1000 chars, `## Role`.
- Evidence Sources: PLAN.md, REVISION_BRIEF.md, SKETCH.md.
- What To Check: blocked_by graph, component boundary isolation, a11y coverage in plan, responsive coverage in plan.
- Visibly DIFFERENT from D15.
- `mage ci` green.

**ContextBlocks:**
- `reference`: Source `~/.claude/agents/fe-qa-proof-agent.md`.
- `constraint` (high): Must NOT be near-identical to D15.

**KindPayload:**
```json
{"changes": [{"file": ".tillsyn/agents/fe/plan-qa-proof-agent.md", "symbol": "(new file)", "action": "add", "shape_hint": "frontmatter + ## Role + Evidence: PLAN.md/REVISION_BRIEF/SKETCH + FE-specific checks: component boundaries/a11y/responsive"}]}
```

**Mage target:** `mage ci`.

---

### D14 — .tillsyn/agents/fe/plan-qa-falsification-agent.md

- **State:** todo
- **Kind:** build
- **Irreducible:** true
- **Paths:** `.tillsyn/agents/fe/plan-qa-falsification-agent.md` (NEW)
- **Packages:** (none)
- **Blocked by:** D0

#### Specify (D14)

**Objective:** Author the Tillsyn-project-local FE plan-QA-falsification agent prompt. PRIMARY source: `~/.claude/agents/fe-qa-falsification-agent.md`. Differentiate from D16. FE plan-qa-falsification ATTACKS THE FE PLAN — finds missing a11y coverage, missing responsive coverage, hidden Wails IPC dependencies, cross-component state coupling not declared in blocked_by, stil tokens dependency chain issues.

**AcceptanceCriteria:**
- File at `.tillsyn/agents/fe/plan-qa-falsification-agent.md`.
- Frontmatter: `name: plan-qa-falsification-agent`, `model: opus`, `tools: Read, Grep, Glob, Hylla`.
- Migration marker, body >= 1000 chars, `## Role`.
- Evidence Sources: PLAN.md, REVISION_BRIEF.md, SKETCH.md.
- Attack Vectors: missing a11y plan coverage, missing responsive coverage, missing blocked_by between FE droplets sharing TS modules/CSS files, Wails IPC dependency not declared.
- Visibly DIFFERENT from D16.
- `mage ci` green.

**ContextBlocks:**
- `reference`: Source `~/.claude/agents/fe-qa-falsification-agent.md`.
- `constraint` (high): Must NOT be near-identical to D16.

**KindPayload:**
```json
{"changes": [{"file": ".tillsyn/agents/fe/plan-qa-falsification-agent.md", "symbol": "(new file)", "action": "add", "shape_hint": "frontmatter + ## Role + Evidence: PLAN.md + FE Attack Vectors: a11y/responsive/Wails IPC/cross-component coupling"}]}
```

**Mage target:** `mage ci`.

---

### D15 — .tillsyn/agents/fe/build-qa-proof-agent.md

- **State:** todo
- **Kind:** build
- **Irreducible:** true
- **Paths:** `.tillsyn/agents/fe/build-qa-proof-agent.md` (NEW)
- **Packages:** (none)
- **Blocked by:** D0

#### Specify (D15)

**Objective:** Author the Tillsyn-project-local FE build-QA-proof agent prompt. PRIMARY source: `~/.claude/agents/fe-qa-proof-agent.md`. Differentiate from D13. FE build-qa-proof verifies ACTUAL FE CODE CHANGES: Playwright pass rate, a11y violation count, type errors, test coverage, browser_snapshot semantic check. Evidence: FE source + Playwright output + `git diff` + `browser_snapshot` results.

**AcceptanceCriteria:**
- File at `.tillsyn/agents/fe/build-qa-proof-agent.md`.
- Frontmatter: `name: build-qa-proof-agent`, `model: opus`, `tools: Read, Grep, Glob, Hylla`.
- Migration marker, body >= 1000 chars, `## Role`.
- Evidence Sources: FE source files, `git diff`, Playwright MCP output (`browser_snapshot`, `browser_take_screenshot`), Vitest results.
- What To Check: Playwright pass rate, a11y no new violations, TS strict no type errors, ESLint clean, no files modified outside declared paths, migration markers present on new components.
- Visibly DIFFERENT from D13.
- `mage ci` green.

**ContextBlocks:**
- `reference`: Source `~/.claude/agents/fe-qa-proof-agent.md`.
- `constraint` (high): Must NOT be near-identical to D13.

**KindPayload:**
```json
{"changes": [{"file": ".tillsyn/agents/fe/build-qa-proof-agent.md", "symbol": "(new file)", "action": "add", "shape_hint": "frontmatter + ## Role + Evidence: FE source/Playwright/Vitest/git diff + What To Check: pass rate/a11y/types/ESLint/paths"}]}
```

**Mage target:** `mage ci`.

---

### D16 — .tillsyn/agents/fe/build-qa-falsification-agent.md

- **State:** todo
- **Kind:** build
- **Irreducible:** true
- **Paths:** `.tillsyn/agents/fe/build-qa-falsification-agent.md` (NEW)
- **Packages:** (none)
- **Blocked by:** D0

#### Specify (D16)

**Objective:** Author the Tillsyn-project-local FE build-QA-falsification agent prompt. PRIMARY source: `~/.claude/agents/fe-qa-falsification-agent.md`. Differentiate from D14. FE build-qa-falsification ATTACKS THE FE BUILD — visual regressions the Playwright test misses, a11y violations browser_snapshot catches, Wails IPC race conditions, false-positive visual tests (screenshots don't match design), stil token drift.

**AcceptanceCriteria:**
- File at `.tillsyn/agents/fe/build-qa-falsification-agent.md`.
- Frontmatter: `name: build-qa-falsification-agent`, `model: opus`, `tools: Read, Grep, Glob, Hylla`.
- Migration marker, body >= 1000 chars, `## Role`.
- Evidence Sources: FE source, Playwright output, `git diff`.
- Attack Vectors: visual regression not caught by text-based assertions, a11y violation in browser_snapshot, Wails IPC error path not tested, stil token not applied (hardcoded color instead), missing migration marker on new component.
- Visibly DIFFERENT from D14.
- `mage ci` green.

**ContextBlocks:**
- `reference`: Source `~/.claude/agents/fe-qa-falsification-agent.md`.
- `constraint` (high): Must NOT be near-identical to D14.

**KindPayload:**
```json
{"changes": [{"file": ".tillsyn/agents/fe/build-qa-falsification-agent.md", "symbol": "(new file)", "action": "add", "shape_hint": "frontmatter + ## Role + Evidence: FE source/Playwright + Attack Vectors: visual regression/a11y/Wails IPC race/stil drift/false-positive tests"}]}
```

**Mage target:** `mage ci`.

---

### D17 — .tillsyn/agents/fe/research-agent.md

- **State:** todo
- **Kind:** build
- **Irreducible:** true
- **Paths:** `.tillsyn/agents/fe/research-agent.md` (NEW)
- **Packages:** (none)
- **Blocked by:** D0

#### Specify (D17)

**Objective:** Author the Tillsyn-project-local FE research agent prompt. PRIMARY source: `~/.claude/agents/fe-research-agent.md`. Read-only. FE research compiles findings about FE state, component decisions, Playwright behavior, Wails IPC patterns. Evidence: FE source + Playwright MCP (browser_snapshot for semantic inspection) + MDN/CanIUse for CSS/HTML semantics.

**AcceptanceCriteria:**
- File at `.tillsyn/agents/fe/research-agent.md`.
- Frontmatter: `name: research-agent`, `model: opus`, `tools: Read, Grep, Glob, Hylla`.
- Migration marker, body >= 1000 chars, `## Role`.
- Body: read-only; FE-specific evidence sources (MDN/CanIUse, Playwright MCP, stil token files, Hylla for Go cross-reference); never edits code; findings returned to orchestrator.
- `mage ci` green.

**ContextBlocks:**
- `reference`: Source `~/.claude/agents/fe-research-agent.md`.
- `constraint` (high): Read-only — no Edit/Write.

**KindPayload:**
```json
{"changes": [{"file": ".tillsyn/agents/fe/research-agent.md", "symbol": "(new file)", "action": "add", "shape_hint": "frontmatter + ## Role + read-only + FE evidence sources: MDN/CanIUse/Playwright MCP/stil tokens"}]}
```

**Mage target:** `mage ci`.

---

### D18 — .tillsyn/agents/fe/closeout-agent.md

- **State:** todo
- **Kind:** build
- **Irreducible:** true
- **Paths:** `.tillsyn/agents/fe/closeout-agent.md` (NEW)
- **Packages:** (none)
- **Blocked by:** D0

#### Specify (D18)

**Objective:** Author the Tillsyn-project-local FE closeout agent prompt. NO SOURCE FILE — author FROM SCRATCH. Same scope as D8 (go/closeout-agent.md) but adapted for FE drops: FE closeout also aggregates Playwright findings, visual regression notes, a11y coverage notes. References: WORKFLOW.md Phase 7, CLAUDE.md §"Cascade Ledger + Hylla Feedback", ORCH-MANAGED-R1.

**AcceptanceCriteria:**
- File at `.tillsyn/agents/fe/closeout-agent.md`.
- Frontmatter: `name: closeout-agent`, `model: orchestrator-managed`, `tools: Read, Edit, Write, Grep, Glob`.
- Migration marker, body >= 1000 chars, `## Role`.
- Body: closeout phase steps for FE drops; FE-specific: aggregate Playwright coverage notes, visual regression audit in CLOSEOUT.md; Hylla reingest not triggered by this agent; ORCH-MANAGED-R1 note.
- Authored FROM SCRATCH.
- `mage ci` green.

**ContextBlocks:**
- `constraint` (high): No source file exists. Do NOT look for `~/.claude/agents/fe-closeout-agent.md`.
- `reference`: WORKFLOW.md Phase 7.
- `reference`: D8 can be used as a structural template (adapt go closeout for FE context).

**KindPayload:**
```json
{"changes": [{"file": ".tillsyn/agents/fe/closeout-agent.md", "symbol": "(new file)", "action": "add", "shape_hint": "frontmatter + ## Role + FE closeout: Playwright findings aggregation + visual regression notes + ORCH-MANAGED-R1"}]}
```

**Mage target:** `mage ci`.

---

### D19 — .tillsyn/agents/fe/commit-message-agent.md

- **State:** todo
- **Kind:** build
- **Irreducible:** true
- **Paths:** `.tillsyn/agents/fe/commit-message-agent.md` (NEW)
- **Packages:** (none)
- **Blocked by:** D0

#### Specify (D19)

**Objective:** Author the Tillsyn-project-local FE commit-message agent prompt. NO SOURCE FILE — author FROM SCRATCH. Same discipline as D9 (go/commit-message-agent.md): conventional-commit, single-line ≤72 chars, no body, haiku model. FE-specific scope notes: FE commits use `feat(fe)`, `fix(fe)`, etc.

**AcceptanceCriteria:**
- File at `.tillsyn/agents/fe/commit-message-agent.md`.
- Frontmatter: `name: commit-message-agent`, `model: haiku`, `tools: Read`.
- Migration marker, body >= 1000 chars, `## Role`.
- Body: conventional-commit format, single-line, FE scopes (`feat(fe)`, `fix(fe)`, `style(fe)`, `test(fe)`), QA-before-commit rule.
- Authored FROM SCRATCH.
- `mage ci` green.

**ContextBlocks:**
- `constraint` (high): No source file exists.
- `reference`: D9 (go/commit-message-agent.md) structural template.
- `reference`: CLAUDE.md §"Git Commit Format".

**KindPayload:**
```json
{"changes": [{"file": ".tillsyn/agents/fe/commit-message-agent.md", "symbol": "(new file)", "action": "add", "shape_hint": "frontmatter + ## Role + FE conventional-commit format + single-line + FE-specific scope tokens"}]}
```

**Mage target:** `mage ci`.

---

### D20 — .tillsyn/agents/fe/orchestrator-managed.md

- **State:** todo
- **Kind:** build
- **Irreducible:** true
- **Paths:** `.tillsyn/agents/fe/orchestrator-managed.md` (NEW)
- **Packages:** (none)
- **Blocked by:** D0

#### Specify (D20)

**Objective:** Author the Tillsyn-project-local FE orchestrator-managed agent prompt. NO SOURCE FILE — author FROM SCRATCH. Same scope as D10 (go/orchestrator-managed.md) but adapted for FE drops: orchestrator-managed kinds (`closeout`, `refinement`, `discussion`, `human-verify`) for FE work include FE-specific close-out notes (Playwright coverage summary, visual audit, stil token consistency). References: CLAUDE.md §"Orchestrator-as-Hub Architecture", WORKFLOW.md, WIKI.md, ORCH-MANAGED-R1.

**AcceptanceCriteria:**
- File at `.tillsyn/agents/fe/orchestrator-managed.md`.
- Frontmatter: `name: orchestrator-managed`, `model: orchestrator-managed`, `tools: Read, Edit, Write, Grep, Glob`.
- Migration marker, body >= 1000 chars, `## Role`.
- Body: 4-kind coverage (closeout/refinement/discussion/human-verify) with FE-specific notes; orchestrator-never-edits-FE-source rule; ORCH-MANAGED-R1 note.
- Authored FROM SCRATCH.
- `mage ci` green.

**ContextBlocks:**
- `constraint` (high): No source file. Do NOT look for `~/.claude/agents/fe-orchestrator-managed.md`.
- `reference`: D10 (go/orchestrator-managed.md) structural template.
- `reference`: CLAUDE.md §"Orchestrator-as-Hub Architecture".

**KindPayload:**
```json
{"changes": [{"file": ".tillsyn/agents/fe/orchestrator-managed.md", "symbol": "(new file)", "action": "add", "shape_hint": "frontmatter + ## Role + FE orchestrator-managed: 4-kind coverage with FE-specific notes + ORCH-MANAGED-R1"}]}
```

**Mage target:** `mage ci`.

---

### D21 — Smoke Test: Project-Tier Prompt Resolution (Wave C)

- **State:** todo
- **Kind:** build
- **Irreducible:** true
- **Paths:**
  - `internal/app/dispatcher/cli_claude/render/render_test.go` (MODIFY — add test function)
- **Packages:** `internal/app/dispatcher/cli_claude/render`
- **Blocked by:** D0, D1, D2, D3, D4, D5, D6, D7, D8, D9, D10, D11, D12, D13, D14, D15, D16, D17, D18, D19, D20, 4c.6.1.W1
- **Wave:** C — fires after all 20 prompt droplets complete AND after W1 (subdir-per-group resolver) lands

#### Specify (D21)

**Objective:** Add a unit test to `render_test.go` that verifies `render.Render()` resolves a project-tier agent file over the embedded default when the W1 subdir-per-group path shape is in place. The test creates a real `.tillsyn/agents/go/builder-agent.md` file (using the W8-authored content from D2), calls `render.Render()` with `project.RepoPrimaryWorktree` pointing at the temp dir, and asserts the rendered output body contains the W8 file's post-frontmatter content — NOT the embedded till-go default.

This verifies the 3-tier resolver's project-tier priority after W1 changes `readProjectTierAgent` from flat (`/.tillsyn/agents/<basename>`) to subdir-per-group (`/.tillsyn/agents/<group>/<basename>`).

**Test name (new, not yet in tree):** `TestRenderProjectTierOverridesEmbeddedDefault`

**Critical constraint on path shape:** The test must create the project-tier file at `<tmpdir>/.tillsyn/agents/go/builder-agent.md` (with `go/` subdir) — the W1 subdir-per-group shape. If the test places the file at `<tmpdir>/.tillsyn/agents/builder-agent.md` (flat), it will fail after W1 lands (W1 changes the resolver to read the subdir path). This droplet is blocked by W1 to ensure the new resolver shape is in place before this test is written.

**Strip-then-inject note:** `assembleAgentFileBody` strips and re-injects frontmatter keys (`model`, `allowedTools`, `disallowedTools`). The test should assert on the POST-FRONTMATTER body section (the text after the closing `---\n` delimiter) rather than the full raw file bytes, OR use a binding with `Model: nil` and empty `ToolsAllowed`/`ToolsDisallowed` to minimize strip-inject transformations. Either approach is valid; document the choice.

**AcceptanceCriteria:**
- `TestRenderProjectTierOverridesEmbeddedDefault` exists in `render_test.go`.
- Test is table-driven (at minimum one case: project-tier file present → project body resolved; bonus: second case where project-tier file absent → falls through to embedded default).
- Test creates a temp dir via `t.TempDir()`, places a file at `<tmpdir>/.tillsyn/agents/go/builder-agent.md` containing a valid agent body (substantive body, frontmatter with `name: builder-agent`, body length > 200, contains `## Role`).
- Test calls `render.Render()` with `project.RepoPrimaryWorktree = tmpdir` and `binding.SystemPromptTemplatePath = "go/builder-agent.md"` (the subdir-per-group path that W1 introduces).
- Test sets `binding.AgentName = "builder-agent"` so the rendered file path resolves to `<bundle.Root>/plugin/agents/builder-agent.md` — the filename is driven by `binding.AgentName` (per `render.go:327`), NOT by `SystemPromptTemplatePath`. Omitting this field causes the rendered file to land at a different path than asserted.
- Test asserts rendered agent file at `<bundle.Root>/plugin/agents/builder-agent.md` contains the test fixture's post-frontmatter body content (NOT the embedded default body).
- Test uses `t.Parallel()`.
- `mage test-pkg ./internal/app/dispatcher/cli_claude/render` passes with the new test.
- `mage ci` green.

**W1 dependency rationale:** W1 changes `readProjectTierAgent` to use subdir-per-group. This test exercises that exact change. Writing the test before W1 lands would require guessing the new resolver path. Blocking on W1 ensures the test is written against the actual post-W1 resolver implementation.

**ValidationPlan:**
- `mage test-func ./internal/app/dispatcher/cli_claude/render TestRenderProjectTierOverridesEmbeddedDefault` — passes.
- `mage test-pkg ./internal/app/dispatcher/cli_claude/render` — full package green.
- `mage ci` — full CI green.

**RiskNotes:**
- Do NOT use `/tmp/tillsyn/main` as the RepoPrimaryWorktree value — use `t.TempDir()` for proper test isolation.
- The `SystemPromptTemplatePath` in the binding determines BOTH the group (`path.Dir("go/builder-agent.md")` = `"go"`) AND the basename (`path.Base("go/builder-agent.md")` = `"builder-agent.md"`). This is the key linkage: `binding.SystemPromptTemplatePath = "go/builder-agent.md"` makes the resolver look at `<worktree>/.tillsyn/agents/go/builder-agent.md` in the project tier (after W1's change to `readProjectTierAgent`).
- Current pre-W1 resolver at `readProjectTierAgent` uses: `filepath.Join(projectWorktree, ".tillsyn/agents", basename)` — flat path. W1 will change this to `filepath.Join(projectWorktree, ".tillsyn/agents", group, basename)`. The test is correct for the post-W1 world only.
- Coverage gate: existing package coverage is > 70%; adding one test should not drop it.

**ContextBlocks:**
- `reference`: `internal/app/dispatcher/cli_claude/render/render.go:readProjectTierAgent` — the function W1 modifies (project tier path).
- `reference`: `internal/app/dispatcher/cli_claude/render/render.go:assembleAgentFileBody` — 3-tier resolver logic.
- `reference`: `internal/app/dispatcher/cli_claude/render/render.go:resolveAgentGroup` — how group is derived from `binding.SystemPromptTemplatePath`.
- `reference`: `internal/app/dispatcher/cli_claude/render/render_test.go` — existing test patterns: `fixtureBundle(t)`, `fixtureItem()`, `fixtureProject()`, `t.Parallel()`, `t.TempDir()`.
- `decision`: W8-SMOKE-R1 deferred — full end-to-end dispatcher flow (spawn descriptor → subagent invocation) deferred to Drop 4c.7. This test is unit-test only: `render.Render()` with a stubbed project worktree.
- `warning` (high): Do NOT write `render_test.go` changes before W1 completes. The test assumes the subdir-per-group path shape that W1 introduces.

**KindPayload:**
```json
{"changes": [
  {"file": "internal/app/dispatcher/cli_claude/render/render_test.go", "symbol": "TestRenderProjectTierOverridesEmbeddedDefault", "action": "add", "shape_hint": "table-driven test, t.TempDir() worktree, .tillsyn/agents/go/builder-agent.md fixture, Render() call with SystemPromptTemplatePath=go/builder-agent.md, assert post-frontmatter body matches fixture"}
]}
```

**Mage target:** `mage test-pkg ./internal/app/dispatcher/cli_claude/render` then `mage ci`.

---

## _BLOCKERS.toml Mirror

Note: PLAN.md mirror uses bare `D*` IDs (consistent with sibling L2 plans). The on-disk `_BLOCKERS.toml` uses the same bare `D*` convention (updated in round-2 to match). Cross-wave reference `4c.6.1.W1` in D21 retains its fully-qualified form.

```toml
# _BLOCKERS.toml — workflow/drop_4c_6_1/DROP_4c.6.1.W8_TILLSYN_PROJECT_PROMPTS/
# Immediate-children sibling blocker ledger. PLAN.md is truth.

[[blockers]]
node = "D0"
blocked_by = []
reason = "Wave A head — no upstream blockers"

[[blockers]]
node = "D1"
blocked_by = ["D0"]
reason = "D0 must commit .gitignore re-includes so D1's .md file is tracked by git"

[[blockers]]
node = "D2"
blocked_by = ["D0"]
reason = "same as D1"

[[blockers]]
node = "D3"
blocked_by = ["D0"]
reason = "same as D1"

[[blockers]]
node = "D4"
blocked_by = ["D0"]
reason = "same as D1"

[[blockers]]
node = "D5"
blocked_by = ["D0"]
reason = "same as D1"

[[blockers]]
node = "D6"
blocked_by = ["D0"]
reason = "same as D1"

[[blockers]]
node = "D7"
blocked_by = ["D0"]
reason = "same as D1"

[[blockers]]
node = "D8"
blocked_by = ["D0"]
reason = "same as D1"

[[blockers]]
node = "D9"
blocked_by = ["D0"]
reason = "same as D1"

[[blockers]]
node = "D10"
blocked_by = ["D0"]
reason = "same as D1"

[[blockers]]
node = "D11"
blocked_by = ["D0"]
reason = "same as D1"

[[blockers]]
node = "D12"
blocked_by = ["D0"]
reason = "same as D1"

[[blockers]]
node = "D13"
blocked_by = ["D0"]
reason = "same as D1"

[[blockers]]
node = "D14"
blocked_by = ["D0"]
reason = "same as D1"

[[blockers]]
node = "D15"
blocked_by = ["D0"]
reason = "same as D1"

[[blockers]]
node = "D16"
blocked_by = ["D0"]
reason = "same as D1"

[[blockers]]
node = "D17"
blocked_by = ["D0"]
reason = "same as D1"

[[blockers]]
node = "D18"
blocked_by = ["D0"]
reason = "same as D1"

[[blockers]]
node = "D19"
blocked_by = ["D0"]
reason = "same as D1"

[[blockers]]
node = "D20"
blocked_by = ["D0"]
reason = "same as D1"

[[blockers]]
node = "D21"
blocked_by = ["D0", "D1", "D2", "D3", "D4", "D5", "D6", "D7", "D8", "D9", "D10", "D11", "D12", "D13", "D14", "D15", "D16", "D17", "D18", "D19", "D20", "4c.6.1.W1"]
reason = "D21 smoke test needs all prompt files written (D0-D20) AND needs W1's subdir-per-group resolver change in place"
```

---

## Droplet Count Verification (PLAN-QA-DISCIPLINE-R2)

Enumerated D-list: D0, D1, D2, D3, D4, D5, D6, D7, D8, D9, D10, D11, D12, D13, D14, D15, D16, D17, D18, D19, D20, D21.

Count: 22. Matches L1 PLAN.md directive ("TOTAL droplet count: 22").

---

## Unknowns

- **U1 (explicit, accepted):** Builder agents cannot read `~/.claude/agents/<group>-<role>-agent.md` from this planner's tool session. Builders WILL have filesystem access to their home directory. Each prompt-authoring droplet (D1-D7, D11-D17) declares the source file path explicitly. If a builder finds the source file differs from expectations (e.g. content has changed since this plan was authored), they should adapt accordingly and note it in the build worklog.
- **U2 (explicit, accepted):** W1's exact API for `readProjectTierAgent` (the subdir-per-group change) is not yet landed. D21's smoke test is written post-W1. If W1 changes the API in unexpected ways, the D21 builder must adapt the test to match the actual post-W1 signature. This is why D21 blocks on W1.
- **U3 (explicit, routed):** W8-SMOKE-R1 — full end-to-end dispatcher flow test deferred to Drop 4c.7 per L1 plan. D21 is unit-test only.
