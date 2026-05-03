# UNIT D — PLAN QA FALSIFICATION (Round 1)

**Reviewer:** go-qa-falsification-agent (subagent)
**Working dir:** `/Users/evanschultz/Documents/Code/hylla/tillsyn/main/`
**Target:** `workflow/drop_3/UNIT_D_PLAN.md` (6-droplet decomposition)
**Round:** 1
**Verdict:** **FAIL — multiple CONFIRMED counterexamples; refactor required before builder fires**

---

## 1. Scope of Attack

Adversarial review against the 8 attack vectors prescribed in the spawn prompt: (1) same-file conflict with 3.A.7; (2) skill frontmatter convention; (3) per-drop wrap-up timing; (4) agent file frontmatter insertion edge case; (5) doc sweep completeness; (6) worklog-vs-commit boundary on `~/.claude/`; (7) adopter bootstrap propagation correctness; (8) coverage of PLAN.md § 19.3 lines 1650–1651.

---

## 2. CONFIRMED Counterexamples

### 2.1 Three-way write conflict on `~/.claude/agents/go-qa-falsification-agent.md` (severity: HIGH — most damaging)

**Premise.** Plan-QA must guarantee that two siblings sharing a path have an explicit `blocked_by`, AND that the planner is aware of every site editing that path so the orchestrator can wire ordering correctly.

**Evidence.**
- `workflow/drop_3/UNIT_A_PLAN.md` line 144, droplet 3.A.7, **Paths:** `~/.claude/agents/go-qa-falsification-agent.md` — adds the 5-vector cascade-vocabulary attack block.
- `workflow/drop_3/UNIT_A_PLAN.md` line 171 explicitly flags the conflict and prescribes the wiring: *"Orchestrator must wire either `Unit_D_agent_file_droplet blocked_by 3.A.7` (preferred) … OR merge both edits into a single droplet at synthesis time."*
- `workflow/drop_3/UNIT_D_PLAN.md` droplet 5.D.1 (line 65–91) — also writes `~/.claude/agents/go-qa-falsification-agent.md` (the frontmatter pointer line). **Zero mention of 3.A.7 anywhere in UNIT_D_PLAN.md** (`rg -n "3\.A\.7" UNIT_D_PLAN.md` returns zero results).
- `workflow/drop_3/UNIT_D_PLAN.md` droplet 5.D.5 (line 163) **also** writes `~/.claude/agents/*.md` for the legacy-vocab sweep — *"`~/.claude/agents/*.md` — full pass after 5.D.1's frontmatter reminder is added"*. That is a THIRD edit to the same file from the same unit alone, on top of 3.A.7 from Unit A.

**Trace.** A single physical file (`~/.claude/agents/go-qa-falsification-agent.md`) is touched by at minimum three droplets across two units: 3.A.7 (Unit A — adds attack-vector block), 5.D.1 (Unit D — adds glossary-pointer line above identity), 5.D.5 (Unit D — sweeps legacy `slice` / `qa-check` / `build-task` / `plan-task` strings; the file currently contains 6 instances of `Slice-N` framing per `rg -c "Slice-[0-9]" go-qa-falsification-agent.md` = 6, plus references to `build-task`, `plan-task`, `qa-check`).

**Conclusion.** **CONFIRMED.** Unit D's planner failed to acknowledge the conflict Unit A explicitly flagged, and worse, introduced a second internal conflict (5.D.1 vs 5.D.5 on the same agent files) without `blocked_by` wiring between them. The intra-unit blocker section (lines 56–59) declares 5.D.5 `blocked_by: 5.D.4` and 5.D.6 `blocked_by: 5.D.5`, but says nothing about 5.D.5 sharing `~/.claude/agents/*.md` with 5.D.1.

**Required fix.** Unit D's `UNIT_D_PLAN.md` must:
- Add explicit acknowledgement of the 3.A.7 conflict + the proposed wiring (5.D.1 `blocked_by: 3.A.7`).
- Add `5.D.5 blocked_by: 5.D.1` (or merge the agent-file portion of 5.D.5 into 5.D.1).
- Update the dependency map table on lines 220–227 to reflect both edges.

**Unknowns.** Whether the orch-synthesis step Unit A points to is documented sufficiently for orch to act. Route to orch.

---

### 2.2 Doc sweep coverage gaps — historical drop dirs `drop_1_5/` + `drop_1_75/` not excluded (severity: HIGH)

**Premise.** The planner correctly excludes `workflow/drop_0/`, `workflow/drop_1/`, `workflow/drop_2/` as historical audit trail. The same audit-trail rationale must apply to every closed historical drop dir.

**Evidence.**
- `ls workflow/` shows `drop_1_5`, `drop_1_75`, `drop_2`, `drop_3`, `example`, `README.md` present. (`drop_0` and `drop_1` were merged at unspecified points; only the listed dirs survive.)
- `UNIT_D_PLAN.md` exclusion list (lines 168–175) names ONLY `workflow/drop_0/**`, `workflow/drop_1/**`, `workflow/drop_2/**` — `drop_1_5/` and `drop_1_75/` are absent.
- `workflow/drop_1_5/P4_T4_BUILD_DIFF_INPUT_FROM_RESOURCEREFS/` and `workflow/drop_1_75/` contain BUILDER_WORKLOG.md / BUILDER_QA_*.md files — historical worklogs identical in nature to the explicitly-excluded `drop_2/` files.
- Per memory rule "Never Remove Workflow Drop Files": *"Audit trail is load-bearing."*

**Trace.** A builder reading the in-scope set in 5.D.5 / 5.D.6 sees `drop_1_5/` and `drop_1_75/` as not-explicitly-excluded → falls under the implicit "everything not excluded is in scope" reading → sweeps historical worklog audit trail → contradicts memory rule.

**Conclusion.** **CONFIRMED.** Exclusion list is incomplete.

**Required fix.** Add to 5.D.5's "Paths (out of scope — explicitly excluded)" block:
- `main/workflow/drop_1_5/**`
- `main/workflow/drop_1_75/**`

**Unknowns.** None.

---

### 2.3 Doc sweep missing `~/.claude/CLAUDE.md` retired-vocab edits (severity: MEDIUM)

**Premise.** 5.D.5 explicitly carves out `~/.claude/CLAUDE.md` as *"review only, no edits unless a rule has actually been retired"* (line 166). For the carve-out to be safe, the plan must verify the file contains no retired rules. The plan does not do that verification; it punts.

**Evidence.**
- `rg -c "qa-check|build-task|plan-task" ~/.claude/CLAUDE.md` = 3 hits.
- Line 10: *"a `build-task` generating required QA subtasks owned by `qa`"* — `build-task` is retired post-Drop-1.75 (now `build` per the closed 12-kind enum, see project CLAUDE.md "Cascade Tree Structure").
- Line 121: *"Template-generated QA subtasks (via `child_rules` on a `build-task` …)"* — same retirement.
- Line 147: *"Before any build-task can be marked done"* — same.
- Line 9: *"slice-by-slice / release-by-release"* — `slice` is the retired pre-cascade-vocab term per dev direction 2026-04-25 (PLAN.md § 19.3 line 1625, "waterfall metaphor that aligns the branding").

**Trace.** A builder running 5.D.5 sees the *"review only, no edits unless a rule has actually been retired"* clause → reviews the file → these rules ARE retired → so the file SHOULD be edited → but the planner framed the touch as exceptional ("no edits unless"), creating ambiguity about how aggressive the rewrite should be (single-line surgical fix vs. full sweep) and who gets to decide (builder self-discretion vs. orch confirmation).

**Conclusion.** **CONFIRMED.** The carve-out language is wrong; this file is squarely in-scope at the legacy-vocab pass.

**Required fix.** Promote `~/.claude/CLAUDE.md` from "review only" to first-class in-scope edit target with explicit known hits enumerated:
- Line 9 `slice-by-slice` → drop or rephrase per cascade vocabulary.
- Lines 10, 121, 147 `build-task` → `build`.

**Unknowns.** Whether `~/.claude/CLAUDE.md` carries dev-personal rules outside Tillsyn's vocabulary scope. Route to orch (this file is not under repo control; dev decision required before edit).

---

### 2.4 Adopter bootstrap propagation — bootstrap skills don't currently own WIKI.md (severity: MEDIUM)

**Premise.** 5.D.2 / 5.D.3 acceptance prescribe extending an existing "Required WIKI scaffolding" subsection. That requires the subsection to exist or to be a coherent extension of existing coverage.

**Evidence.**
- `rg -n "WIKI" ~/.claude/skills/go-project-bootstrap/` returns zero hits — the skill mentions WIKI nowhere.
- `rg -n "WIKI" ~/.claude/skills/fe-project-bootstrap/` returns zero hits — same for FE.
- Skill `Workflow` step 3 ("Add Go-specific guidance" / "Add FE-specific guidance") concerns CLAUDE.md content only; there is no current step that touches WIKI.md.
- Skill `Resources` section names only `references/template.md` — WIKI scaffolding is brand-new conceptual territory.

**Trace.** The droplet treats *"gains a 'Required WIKI scaffolding' subsection"* as a small additive change. It is actually a scope expansion: the bootstrap skill currently produces a single CLAUDE.md; after Unit D it must also produce / seed a WIKI.md. That implies new responsibilities (detect WIKI absence, write a new file, place the seeded content correctly, decide what other WIKI sections the bootstrap must / must not seed). None of that is acknowledged in 5.D.2 / 5.D.3 acceptance.

**Conclusion.** **CONFIRMED.** Plan understates 5.D.2 / 5.D.3's scope.

**Required fix.** Either (a) widen acceptance to acknowledge new bootstrap responsibilities (write/check WIKI.md, decide section ordering, decide handling when adopter project's WIKI already has the section), or (b) narrow scope to "CLAUDE.md pointer line only" and defer WIKI scaffolding to a follow-up ledger refinement.

**Unknowns.** Adopter use case specificity. Route to orch + dev — the bootstrap-skill-now-also-owns-WIKI question is a real new contract that needs sign-off.

---

### 2.5 Worklog vs commit boundary — accountability gap on `~/.claude/` edits (severity: MEDIUM)

**Premise.** All edits inside `~/.claude/` happen outside the repo tree. They land neither in the Drop 3 PR nor in any git history. The plan's mitigation is *"diff is recorded in `UNIT_D_BUILDER_WORKLOG.md`"* (5.D.1, 5.D.2, 5.D.3, partial of 5.D.5 + 5.D.6).

**Evidence.**
- 5.D.1 paths: 10 files in `~/.claude/agents/` — none git-tracked.
- 5.D.2/5.D.3 paths: 4 files in `~/.claude/skills/{go,fe}-project-bootstrap/` — none git-tracked.
- 5.D.5 includes `~/.claude/skills/*/SKILL.md` + `~/.claude/skills/*/references/*.md` plus `~/.claude/CLAUDE.md` — none git-tracked.
- The diff capture mechanism prescribed is "record in `UNIT_D_BUILDER_WORKLOG.md`." The worklog is a markdown file; it captures whatever the builder writes into it, NOT a structurally-validated diff. There is no equivalent of `git show` to verify the recorded diff matches the actual filesystem state at any later date.

**Trace.** Six months later, a maintainer reading `UNIT_D_BUILDER_WORKLOG.md` sees a diff transcript. The actual files have since been edited again (other drops, or dev-personal changes). The worklog cannot reconstruct ground truth, and there is no independent record of which edit landed when.

**Conclusion.** **CONFIRMED gap.** The plan's "diff in worklog" mitigation is weaker than git-tracked equivalents and the plan does not say so.

**Required fix.** One of:
- (a) Capture per-file SHA256 / md5 + character-level diff in the worklog so future maintainers can verify against current filesystem state.
- (b) Add a step at builder completion: copy each touched `~/.claude/` file into `workflow/drop_3/snapshots/` (read-only audit copies) and commit those into the Drop 3 PR. The repo retains a permanent copy of what the user-level files looked like at Drop 3.
- (c) Explicitly accept the audit gap in `UNIT_D_PLAN.md` Notes section so the limitation is documented.

**Unknowns.** Dev preference between (a)/(b)/(c). Route to orch + dev.

---

### 2.6 5.D.4 wording conflates two separate sections in `workflow/example/CLAUDE.md` (severity: LOW)

**Premise.** The plan's prescribed insertion site must reference an actual section / paragraph in the target file.

**Evidence.**
- `UNIT_D_PLAN.md` 5.D.4 line 137: *"in the existing top-level matter (around the 'This file lives in the **primary work checkout**' paragraph), add an explicit pointer line … The line lands near the existing reading-order bullet ('Read `WIKI.md` + `PLAN.md` + ...')."*
- `workflow/example/CLAUDE.md` line 14 IS the "This file lives in the **primary work checkout**" paragraph — it sits between the generic-template blockquote (lines 3–12) and the "## Coordination Model — At a Glance" header (line 16).
- The "Read `WIKI.md` + `PLAN.md` + …" bullet IS line 26 — inside `## Coordination Model — At a Glance` (lines 16+), NOT in the top-level matter.
- The plan's "around X **and near** Y" language conflates two structurally distinct sites. A builder reading this is forced to guess.

**Conclusion.** **CONFIRMED nit.** Acceptance criteria ambiguous on insertion site.

**Required fix.** Pick one site (recommend: as a new bullet inside `## Coordination Model — At a Glance` immediately after the line-26 reading-order bullet, since cascade vocabulary is coordination-model concern, not template-preamble concern) and make the acceptance prose unambiguous.

**Unknowns.** None — purely an editorial fix.

---

### 2.7 Per-drop wrap-up timing locks (a) without surfacing the strongest counter-argument (severity: LOW)

**Premise.** The plan must surface the trade-off honestly so the orch + dev decide deliberately.

**Evidence.** `UNIT_D_PLAN.md` line 234 locks option (a) — sweep ships inside Drop 3 PR. The architectural-question line surfaces (b) — sweep lands on `main` post-merge — but in one sentence and with no attempt at attacking either side.

**Counter-argument the plan misses.** Option (a) (in-PR) means the Drop 3 PR is the LAST chance to drift the docs before the new vocabulary is "live." That makes it the right time. Option (b) is a strict accountability gap — any post-merge sweep on `main` directly bypasses PR review entirely; STEWARD or a follow-up commit cannot get the same review density. The current plan picks (a) — correct — but doesn't articulate WHY (a) wins on the review-density axis. The dev cannot confirm a recommendation whose reasoning is hidden.

**Conclusion.** **CONFIRMED nit.** Decision is locked correctly but the rationale is implicit.

**Required fix.** Extend the Notes "Architectural Questions Returned to Orchestrator" entry #2 to spell out: *"Option (a) preserves PR-review density; option (b) bypasses PR review and accumulates documentation drift on `main`. Lock (a) for review-density preservation."*

---

### 2.8 Skill frontmatter convention — YAML doesn't have a free-form-notes slot (severity: LOW — planner picked correctly but rationale is incomplete)

**Premise.** Verify the planner's choice of `references/template.md` for the pointer is correct given Claude Code's skill-frontmatter conventions.

**Evidence.**
- `~/.claude/skills/go-project-bootstrap/SKILL.md` lines 1–4: YAML frontmatter has `name` + `description` only. No `references` key, no `tools` key (unlike agent files), no free-form annotation slot.
- `~/.claude/skills/fe-project-bootstrap/SKILL.md`: identical shape.
- Other skills (`qa-falsification-checker`, `qa-proof-checker`, `semi-formal-reasoning`, `plan-from-hylla`, `tui-golden-review`, `tui-vhs-review`): all use only `name` + `description`.
- Claude Code's documented skill format treats unknown YAML keys as silently dropped (matching the agent-file behavior).

**Counterexample search.** Could the YAML `description` field absorb the cascade-glossary pointer? It's free-form prose. But (a) the description's stated purpose is to help the model decide whether to invoke the skill — semantic load shouldn't be diluted with cross-references; (b) the description is single-line; multi-line glossary references would need escaping; (c) downstream tools (Claude Code's skill autoloader) parse description for relevance scoring — adding meta-references degrades signal.

**Conclusion.** **REFUTED.** Planner's choice of `references/template.md` is correct. But the plan's stated rationale is thin.

**Suggested improvement.** Strengthen the architectural-question note (line 233) with the concrete reason (description field is consumed by autoloader for relevance scoring; pollution degrades skill discovery).

**Unknowns.** None.

---

### 2.9 PLAN.md § 19.3 lines 1650–1651 coverage check (severity: LOW — REFUTED with one trivial gap)

**Premise.** Every concrete deliverable in PLAN.md § 19.3 lines 1650–1651 must map to at least one Unit-D droplet.

**Evidence.** Line-by-line walk of lines 1650–1651:

| Line 1650 deliverable | Unit D droplet | Coverage |
|---|---|---|
| `go-project-bootstrap` skill update | 5.D.2 | ✓ |
| `fe-project-bootstrap` skill update | 5.D.3 | ✓ |
| `CLAUDE.md` template pointer | 5.D.4 | ✓ |
| WIKI scaffolding pre-fills `## Cascade Vocabulary` | 5.D.2 + 5.D.3 acceptance | ✓ (but see §2.4) |
| CLAUDE.md pointer line *"Cascade vocabulary canonical: WIKI.md § Cascade Vocabulary"* | 5.D.2 + 5.D.3 + 5.D.4 | ✓ |
| Agent files get one-line frontmatter-body reminder *"Structural classifications (drop \| segment \| confluence \| droplet) live in WIKI glossary — never redefine"* | 5.D.1 | ✓ |

| Line 1651 deliverable | Unit D droplet | Coverage |
|---|---|---|
| Sweep `action_item` / `action-item` / `action item` / `ActionItem` across docs / agent prompts / slash-command files / skill files / memory files | 5.D.5 | partial |
| Update `metadata.role` vs `metadata.structural_type` crosswalk where docs conflated role with kind | 5.D.5 (pass 4) + 5.D.6 | ✓ |
| Commit the sweep as a final docs-only droplet under drop 3 | 5.D.6 | ✓ |

**Single gap on line 1651.** PLAN.md § 19.3 line 1651 names `slash-command files` as in-scope. `UNIT_D_PLAN.md` 5.D.5 path list does NOT explicitly include `~/.claude/commands/` (slash-command files live under `~/.claude/commands/*.md` per Claude Code's skill+command structure). The plan covers `~/.claude/skills/`, `~/.claude/agents/`, `~/.claude/CLAUDE.md`, and the project memory file — but not commands.

**Conclusion.** **CONFIRMED minor gap on slash-command files.**

**Required fix.** Add `~/.claude/commands/*.md` to 5.D.5's path list (with the same NOT-git-tracked / worklog-recording carve-out). Verify any slash-command files exist before committing acceptance text — if there are none, document the absence.

**Unknowns.** Whether the dev maintains slash commands at `~/.claude/commands/`. Route to orch.

---

## 3. Refuted Attacks (Honest Attempts)

### 3.1 5.D.2 + 5.D.3 parallel safety
Different physical directories (`~/.claude/skills/go-project-bootstrap/` vs `~/.claude/skills/fe-project-bootstrap/`). No shared file. **REFUTED.**

### 3.2 5.D.4 + 5.D.5 ordering
5.D.5 declared `blocked_by: 5.D.4`. Both share `main/CLAUDE.md` + `main/workflow/example/CLAUDE.md`. Wiring is correct. **REFUTED.**

### 3.3 Mage CI as smoke test
Unit D edits no Go code. `mage ci` exit status would still cover any tooling/CI changes. Acceptance correctly notes *"no Go code touched, but `mage ci` runs the markdown-aware checks."* **REFUTED.**

### 3.4 5.D.6 covers everything 5.D.5 missed
5.D.6 acceptance asserts coverage of ALL new vocabulary from Units A/B/C. The methodology mirror is sound. **REFUTED.**

---

## 4. Unknowns Routed to Orchestrator

- **U1.** Whether `~/.claude/CLAUDE.md` carries dev-personal rules outside Drop-3 vocabulary scope (§2.3).
- **U2.** Dev preference for `~/.claude/` audit-gap mitigation: SHA256 in worklog, repo-snapshot copies, or explicit accept (§2.5).
- **U3.** Bootstrap skill scope expansion to own WIKI.md authoring, vs. narrow to CLAUDE.md pointer only with WIKI deferred (§2.4).
- **U4.** Whether `~/.claude/commands/` slash-command files exist locally and need sweep coverage (§2.9).
- **U5.** Whether 3.A.7's recommended wiring (`5.D.1 blocked_by: 3.A.7`) is the orch's preferred resolution, or merge-into-single-droplet (§2.1).

---

## 5. Convergence Statement

(a) QA Falsification produced **multiple unmitigated counterexamples** above (§2.1, §2.2, §2.3, §2.4, §2.5; plus three lower-severity in §2.6, §2.7, §2.9).

(b) Evidence completeness: every claim is backed by `rg`/`Read` against the actual files, with line numbers cited.

(c) Unknowns are routed (U1–U5).

**(a) failed**, so this round does NOT converge to PASS. **Verdict: FAIL — refactor required before Phase 1 builders fire.**

---

## 6. Required Refactor Checklist (For Round 2)

1. Acknowledge the 3.A.7 conflict + propose wiring (5.D.1 `blocked_by: 3.A.7`) in Unit D's blocker section.
2. Add `5.D.5 blocked_by: 5.D.1` for the same-file `~/.claude/agents/*.md` overlap.
3. Add `workflow/drop_1_5/**` and `workflow/drop_1_75/**` to 5.D.5's exclusion list.
4. Promote `~/.claude/CLAUDE.md` from "review only" to first-class in-scope edit target with the three known retired-vocab hits enumerated.
5. Resolve 5.D.2 / 5.D.3 scope question (WIKI.md ownership expansion vs. defer).
6. Pick a `~/.claude/` audit-gap mitigation (SHA, repo-snapshot, or explicit accept).
7. Disambiguate 5.D.4's `workflow/example/CLAUDE.md` insertion site to a single named structural location.
8. Add `~/.claude/commands/*.md` to 5.D.5's path list (or document its absence).
9. Update the dependency map table on lines 220–227 to reflect the 5.D.1 ↔ 3.A.7 and 5.D.1 ↔ 5.D.5 edges.

---

## 7. Hylla Feedback

**N/A — task touched non-Go files only.** Hylla today indexes Go only (per memory rule "Hylla Indexes Only Go Files Today"). All evidence gathered via `Read`, `rg`, file listings against doc files. No Hylla queries attempted.
