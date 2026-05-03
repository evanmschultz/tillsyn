# DROP_3 — UNIT D — PLAN QA PROOF — ROUND 1

**Reviewer:** `go-qa-proof-agent` (subagent)
**Target:** `workflow/drop_3/UNIT_D_PLAN.md` (Unit D — Adopter Bootstrap + Cascade-Vocabulary Doc Sweep)
**Round:** 1
**Date:** 2026-05-02
**Verdict:** PASS with nits.

---

## 1. Scope of Review

Verify the planner's 6-droplet decomposition against:

- 1.1 PLAN.md § 19.3 bullets at lines 1650–1651 (adopter bootstrap + cascade-vocabulary doc sweep).
- 1.2 Filesystem reality: `~/.claude/agents/*.md` (10 files), `~/.claude/skills/{go,fe}-project-bootstrap/`, in-repo doc paths enumerated by the plan.
- 1.3 Cross-unit dependencies on Units A / B / C.
- 1.4 Pre-MVP rules in effect (memory rules: opus builders, no closeout MD, never-`git-rm` workflow files, self-QA MD updates, never `mage install`, native tools not shell parsers).

## 2. Required Proof Checks — Per-Item Verdicts

### 2.1 Droplet ID prefix `5.D.N` instead of `3.D.N` — clerical or semantic?

**Verdict: clerical, no semantic impact. PASS.**

- 2.1.1 Sibling unit plans use the correct prefix: `UNIT_A_PLAN.md` ships droplets `3.A.1` through `3.A.N`; `UNIT_B_PLAN.md` ships `3.B.N`; `UNIT_C_PLAN.md` ships `3.C.N`. Unit D's `5.D.N` is the only outlier — almost certainly a copy-paste artifact (possibly from a Drop 5 reference somewhere or a Phase-5 mental model).
- 2.1.2 Internal references inside `UNIT_D_PLAN.md` are self-consistent: every `blocked_by` references `5.D.*` to other `5.D.*` siblings (e.g., `5.D.5 blocked_by: 5.D.4`, `5.D.6 blocked_by: 5.D.5`). The Cross-Unit Dependency Map table also uses `5.D.*` consistently. So the renumber is purely mechanical: orch does `s/5\.D\./3.D./g` at synthesis time and every reference resolves correctly.
- 2.1.3 No external doc currently references Unit D droplet IDs by `5.D.N` (this is the first doc that introduces those IDs), so there is no external blast radius.
- 2.1.4 **Recommendation to orch:** rename all 6 droplets to `3.D.1` … `3.D.6` at synthesis. Update the Cross-Unit Dependency Map. Acceptance criteria are unaffected because they reference paths/files, not droplet IDs.

### 2.2 All 10 agent files named — completeness check

**Verdict: PASS.**

- 2.2.1 Filesystem ground truth (`ls /Users/evanschultz/.claude/agents/`):
  - Go variants (5): `go-builder-agent.md`, `go-planning-agent.md`, `go-qa-proof-agent.md`, `go-qa-falsification-agent.md`, `go-research-agent.md`.
  - FE variants (5): `fe-builder-agent.md`, `fe-planning-agent.md`, `fe-qa-proof-agent.md`, `fe-qa-falsification-agent.md`, `fe-research-agent.md`.
- 2.2.2 Droplet 5.D.1's `Paths` block enumerates exactly these 10 files. No misses, no phantoms.
- 2.2.3 The plan correctly notes all 10 are NOT git-tracked (confirmed via `git ls-files | grep .claude/` returning empty — `.claude/` is wholly outside the repo tree).

### 2.3 `go-project-bootstrap` + `fe-project-bootstrap` skills exist — path + structure

**Verdict: PASS.**

- 2.3.1 Both skill directories exist:
  - `/Users/evanschultz/.claude/skills/go-project-bootstrap/SKILL.md` (3.5k) + `references/template.md` (2.3k).
  - `/Users/evanschultz/.claude/skills/fe-project-bootstrap/SKILL.md` (3.8k) + `references/template.md` (2.4k).
- 2.3.2 Both `SKILL.md` files use YAML frontmatter with `name` + `description` keys only — confirming the planner's claim in droplet 5.D.2 notes that the YAML schema is "two keys" with no free-form notes slot.
- 2.3.3 Both `SKILL.md` files contain a `## Workflow` section with numbered steps. Droplets 5.D.2 + 5.D.3 propose inserting a new bullet under "3. Add Go-specific guidance" / "3. Add FE-specific guidance" — those headings exist literally in the source files (lines 27 of each `SKILL.md`).
- 2.3.4 Both `SKILL.md` files contain a `## Completion Bar` section that the droplets propose extending — sections exist as named.
- 2.3.5 Both `references/template.md` files contain a "Start from these rules:" block — droplets propose appending a new top-level rule. The block exists at line 3 of both files.

### 2.4 Cross-unit dependency on Unit A's WIKI section — hard blocker correctly surfaced

**Verdict: PASS.**

- 2.4.1 Unit A's `## Scope` paragraph (line 5) declares: *"Add the `## Cascade Vocabulary` glossary section to `main/WIKI.md` as the single canonical source for waterfall semantics."* So the WIKI section is genuinely Unit A's deliverable.
- 2.4.2 Unit D's Cross-Unit Dependency Map names "Unit A: WIKI § Cascade Vocabulary authored" as a dep on **all 6 droplets** (5.D.1 through 5.D.6). Correct — every droplet in Unit D either:
  - Inserts a pointer to that WIKI section (5.D.1, 5.D.2, 5.D.3, 5.D.4 — the pointer must resolve), or
  - Sweeps for legacy vocabulary that gets replaced with the new WIKI-glossary phrasing (5.D.5, 5.D.6 — the rewrite target text comes from the WIKI section).
- 2.4.3 Unit D explicitly disclaims authorship of the WIKI section (Out-of-scope bullet 1: *"Authoring the canonical WIKI `## Cascade Vocabulary` section content — that is **Unit A's** deliverable. Unit D consumes it."*). Clean separation.
- 2.4.4 The plan also correctly surfaces a finer dep: 5.D.1 specifically depends on Unit A's `structural_type` enum 4-value list being final (so the agent frontmatter reminder names exactly `drop | segment | confluence | droplet`). This is more granular than just "WIKI authored" — good.

### 2.5 Cross-unit dependency on B + C closing for final wrap-up — 5.D.6 blocker

**Verdict: PASS.**

- 2.5.1 5.D.6's `Blocked by` line: *"5.D.5 (shares paths). Inter-unit: Units A + B + C all closed (`mage ci` green at unit boundary)."* Both intra-unit (5.D.5) and inter-unit (A/B/C closure) blockers are explicit.
- 2.5.2 5.D.6's acceptance criteria correctly enumerate the new vocabulary from each unit:
  - Unit A: `structural_type`, the 4 enum values, atomicity rules, plan-QA-falsification new attack vectors.
  - Unit B: TOML template-system terms (`Template.AllowsNesting`, `[child_rules]`, `KindCatalog`, `templates/builtin/default.toml`, agent-binding fields).
  - Unit C: `principal_type: steward`, `metadata.owner = STEWARD`, auth-level state-lock, template auto-generation.
- 2.5.3 Cross-checked against Unit B's scope (line 3 of `UNIT_B_PLAN.md`) and Unit C's scope (line 13 of `UNIT_C_PLAN.md`) — the new symbols Unit D names match what B and C are landing.
- 2.5.4 The Cross-Unit Dependency Map row for 5.D.6 names the same closure dep. Consistent.

### 2.6 In-repo doc sweep scope — historical worklogs excluded

**Verdict: PASS with one exclusion-list nit (see Findings).**

- 2.6.1 Filesystem reality: `workflow/` contains `drop_1_5/`, `drop_1_75/`, `drop_2/`, `drop_3/`, `example/`, `README.md`. `drop_0/` and `drop_1/` do NOT exist (those drops happened before the per-drop workflow dir was introduced).
- 2.6.2 Droplet 5.D.5's "Paths (out of scope — explicitly excluded)" list names: `workflow/drop_0/**`, `workflow/drop_1/**`, `workflow/drop_2/**`, plus `workflow/drop_3/**` (except this very file). **Missing from the explicit-exclusion list: `workflow/drop_1_5/**` and `workflow/drop_1_75/**`.** These directories exist on disk and contain historical audit-trail content.
- 2.6.3 The intent is clear (audit trail = excluded), but the explicit list is incomplete. Risk: a builder reading 5.D.5's path list literally could interpret `drop_1_5/` and `drop_1_75/` as in-scope. See Finding F1.
- 2.6.4 Other audit-trail exclusions are correctly enumerated: `LEDGER.md`, `WIKI_CHANGELOG.md`, `HYLLA_FEEDBACK.md`, `REFINEMENTS.md`, `HYLLA_REFINEMENTS.md`. All these files exist on disk and are correctly classified as audit trail.

### 2.7 `~/.claude/` files NOT git-tracked — handling

**Verdict: PASS.**

- 2.7.1 `git ls-files` confirms zero `.claude/` entries — those paths are wholly outside the repo tree.
- 2.7.2 Droplets 5.D.1, 5.D.2, 5.D.3, and the relevant subset of 5.D.5 each annotate paths with `(NOT git-tracked)` and prescribe: builder records diffs in `UNIT_D_BUILDER_WORKLOG.md`, nothing lands in the Drop 3 PR for those files.
- 2.7.3 The plan also correctly flags that builder spawn prompts MUST explicitly authorize edits to those paths since they sit outside the repo working dir (notes on each affected droplet).
- 2.7.4 Consistent with memory rule "Bare-Root CLAUDE.md Is Not Git-Tracked" — same handling pattern.
- 2.7.5 In-repo files that DO land in the PR are correctly distinguished: 5.D.4's two paths (`main/CLAUDE.md`, `main/workflow/example/CLAUDE.md`) are explicitly noted as "git-tracked" and "land in the Drop 3 PR." The split is clean.

### 2.8 6 droplets cover full spec — PLAN.md § 19.3 lines 1650–1651

**Verdict: PASS.**

- 2.8.1 PLAN.md line 1650 (adopter bootstrap) decomposes into:
  - "go-project-bootstrap + fe-project-bootstrap skills" → 5.D.2 + 5.D.3.
  - "every CLAUDE.md template" → 5.D.4 (`main/CLAUDE.md` + `workflow/example/CLAUDE.md`); 5.D.5 sweep also touches `~/.claude/CLAUDE.md` (the global one) for legacy-vocab pass.
  - "Every agent file under `.claude/agents/` … gets a one-line reminder in its frontmatter body" → 5.D.1.
- 2.8.2 PLAN.md line 1651 (per-drop wrap-up) decomposes into:
  - "sweep every lingering action_item / action-item / action item / ActionItem string across docs, agent prompts, slash-command files, skill files, and memory files" → 5.D.5 (pre-A/B/C legacy sweep) + 5.D.6 (post-A/B/C new-vocab sweep).
  - "Update metadata.role vs metadata.structural_type crosswalk wherever docs previously conflated role with kind" → 5.D.5 sweep pass 4 + 5.D.6 acceptance criteria.
  - "Commit the sweep as a final docs-only droplet under drop 3" → 5.D.6 explicitly tagged as "the **final docs-only droplet under Drop 3**" with the orchestrator-question parking the timing decision (commit before vs after PR merge).
- 2.8.3 The split of line 1651 across 5.D.5 + 5.D.6 is a defensible interpretation: 5.D.5 targets pre-existing legacy vocabulary that exists today (`slice`, `build-task`, `action_item` confusions) and can be swept BEFORE A/B/C land; 5.D.6 targets new vocabulary introduced by A/B/C and can only be swept AFTER they land. PLAN.md line 1651 says singular "a final docs-only droplet" but the substantive work has two distinct timing windows, so splitting is reasonable. Note in Finding F2.

## 3. Findings

### 3.1 Blocking findings

**None.** No finding rises to blocking — the plan can proceed with the orch making the renumber + 2 nit fixes at synthesis.

### 3.2 Nits

- **F1 — Exclusion list missing `workflow/drop_1_5/**` and `workflow/drop_1_75/**` (5.D.5).** Both directories exist on disk (verified via `ls workflow/`) and contain historical audit-trail content. The planner's rationale ("historical worklogs are audit trail") clearly intends to exclude them, but the explicit list only names `drop_0`, `drop_1`, `drop_2`. **Fix:** orch extends the 5.D.5 "Paths (out of scope — explicitly excluded)" list to `workflow/drop_0/**`, `workflow/drop_1/**`, `workflow/drop_1_5/**`, `workflow/drop_1_75/**`, `workflow/drop_2/**`, `workflow/drop_3/**` (except `UNIT_D_*.md`). Same fix applies to 5.D.6's path list (currently inherits from 5.D.5).

- **F2 — Singular "final docs-only droplet" in PLAN.md line 1651 vs Unit D's split into 5.D.5 + 5.D.6.** Defensible split, but worth flagging at synthesis for dev awareness. Two options: (a) keep the split — 5.D.5 commits as `docs(drop-3): legacy vocabulary sweep`, 5.D.6 commits as `docs(drop-3): cascade vocabulary final sweep`. (b) merge 5.D.5 + 5.D.6 into a single droplet that runs after A/B/C close — eliminates the timing-window argument by waiting for everything. The plan locks (a). Orch should confirm with dev pre-Phase-1.

- **F3 — File-level race risk: 5.D.5 vs Unit A on `~/.claude/agents/go-qa-falsification-agent.md`.** 5.D.5's path list includes `~/.claude/agents/*.md` (full pass after 5.D.1's frontmatter reminder is added). Unit A's `## Scope` (line 5 of `UNIT_A_PLAN.md`) declares: *"Teach `~/.claude/agents/go-qa-falsification-agent.md` the five new attack vectors."* Without explicit `blocked_by` between 5.D.5 and Unit A's qa-falsification-edit droplet, the two writes race on the same file. **Fix:** orch wires `blocked_by: <Unit A's go-qa-falsification-agent edit droplet>` on droplet 5.D.5 at synthesis. The Cross-Unit Dependency Map correctly notes "Unit A: WIKI § Cascade Vocabulary authored" but is silent on Unit A's agent-file edit dep.

- **F4 — Memory rule "No subagents for MD work" tension on 5.D.5.** Memory rule literal text: *"No subagents for MD work. After every MD edit, self-QA for consistency/cross-refs/drift. Present QA findings to dev and wait for approval before applying."* Unit D dispatches a builder subagent for the sweep (5.D.5 notes line: *"this rule applies to in-session orch-driven MD updates; for Unit D, the builder subagent IS the work-doer"*). The planner's reading is defensible (the memory rule was authored for in-orch quick MD updates, not for systematic doc sweeps), but the literal text doesn't carve that out. **Recommendation:** orch surfaces this to dev pre-Phase-1 — either (a) confirm the carve-out applies, or (b) orch handles 5.D.5 + 5.D.6 directly (no subagent) and folds the work into the orch session. Option (b) is feasible because Unit D is doc-only and the orch is allowed to edit MD.

- **F5 — Open architectural questions parked for orch (not findings, but worth re-flagging).** The plan correctly parks 4 open questions for orch confirmation pre-builder-fire:
  1. Skill-file frontmatter convention (insert pointer in YAML vs `references/template.md`).
  2. Per-drop final wrap-up timing (5.D.6 before vs after PR merge).
  3. Boilerplate `## Cascade Vocabulary` content owner (where the bootstrap seeds it from).
  4. Agent frontmatter insertion location (PLAN.md line 1650 says "frontmatter body" — ambiguous).
  
  Each is correctly disclaimed as "open architectural question for orch (not for the builder)" so builders won't proceed until orch confirms. Good discipline.

### 3.3 Most important finding

**F1 (exclusion list completeness for 5.D.5/5.D.6).** Trivial to fix — orch extends the path list at synthesis. But unfixed, a literal-reading builder could rewrite historical drop_1_5/drop_1_75 audit trail, violating memory rule "Never Remove Workflow Drop Files" by the same impulse that protects them.

## 4. Unknowns

- 4.1 The 4 open architectural questions (F5) need dev confirmation pre-Phase-1. Not blocking the plan's PASS, but builders cannot fire until they're answered.
- 4.2 Whether 5.D.5 (subagent for MD sweep) is acceptable under memory rule "No subagents for MD work." Routed to orch.
- 4.3 Whether to merge 5.D.5 + 5.D.6 into a single post-A/B/C droplet (F2 option b). Routed to orch + dev.

## 5. Verdict

**PASS with nits.**

- Plan correctly enumerates all 10 agent files, both bootstrap skills, all in-repo doc paths.
- Cross-unit dependencies on Units A / B / C are surfaced (with one gap on the qa-falsification-agent edit race — F3).
- ID prefix `5.D.N` is clerical, no semantic impact, mechanically renumberable at synthesis.
- 6 droplets cover PLAN.md § 19.3 lines 1650–1651 fully, with one defensible split of line 1651 across 5.D.5 + 5.D.6.
- Pre-MVP rules (no closeout MD, opus builders, never `mage install`, never `git rm` workflow files, native tools) all reflected in plan notes.
- Out-of-scope confirmations are explicit (no Go code, no historical drop dir edits, no audit-trail MD edits).

Orch can proceed to Phase 1 after applying the renumber (`5.D.* → 3.D.*`), extending the exclusion list (F1), wiring the qa-falsification-file `blocked_by` (F3), and getting dev signoff on F4 + F5.

## 6. Hylla Feedback

`N/A — task touched non-Go files only.` Unit D and this review are pure markdown. Hylla today indexes Go only (per memory rule "Hylla Indexes Only Go Files Today"). All evidence gathered via `Read`, `Bash` (`ls`, `git ls-files`, `awk`), `Glob` on doc + agent + skill files. No Hylla queries attempted, no fallback misses to log.

## TL;DR

- T1 Reviewed Unit D's 6-droplet decomposition against PLAN.md § 19.3 lines 1650–1651, filesystem reality, sibling unit plans, and memory rules.
- T2 All 8 required proof checks PASS: ID prefix is clerical-only, 10 agents + 2 skills exist as named, cross-unit deps surfaced, 6 droplets cover the spec, audit-trail exclusions handled, `~/.claude/` correctly classified as not-tracked.
- T3 Five non-blocking findings: F1 (exclusion list missing `drop_1_5`/`drop_1_75` — most important), F2 (line-1651 split defensible but flag for dev), F3 (qa-falsification-file race risk vs Unit A), F4 (subagent-for-MD memory-rule tension), F5 (4 open architectural questions correctly parked).
- T4 Unknowns routed: open architectural questions need dev signoff pre-Phase-1; subagent-for-MD-sweep needs dev confirmation; 5.D.5/5.D.6 merge-vs-split needs dev call.
- T5 Verdict: PASS with nits. Orch applies renumber + F1 + F3 fixes at synthesis, gets dev signoff on F4 + F5, then proceeds to Phase 1.
- T6 Hylla feedback: N/A — non-Go review surface.
